//go:build e2e

package shadow_test

// R7-1 full-stack browser E2E (build tag e2e; run by .github/workflows/e2e.yml):
//
//	real Chromium (Playwright) → built SPA (web/dist) → real cmd/crm-api binary → disposable PostgreSQL
//
// The disposable database, its restricted per-module logins and the SYNTHETIC
// Slice 11 organisation / lead set come from the existing test environment
// (pgtest + setupWrites); both are dropped when the test ends. Nothing here
// adds an endpoint, capability, grant, migration or business rule:
//
//   - two synthetic credentials are inserted for synthetic sellers X (holds
//     lead 101) and Y (holds lead 102), exactly as the R1 auth tests do;
//   - flags are switched ON only inside this disposable database
//     (identity.session_auth, sales.read_shadow, cases.nonfinancial_write);
//   - crm-api runs with CRM_ENV=test (no dev stub), real R1 sessions and the
//     production subject resolver: a subject holds only ACTIVE role grants,
//     and none exist (D-42). The suite therefore proves what the shipped
//     product does today, including that a seller cannot read cases or
//     record a contact outcome.
//
// Every listener binds 127.0.0.1; every DSN must be loopback; the browser
// aborts and reports any non-local request.

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/password"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

// browserSeller is one synthetic login (password random per run, never logged).
type browserSeller struct {
	login, password, userID string
}

func TestBrowserFullStack(t *testing.T) {
	root := repoRoot(t)
	dist := filepath.Join(root, "web", "dist", "index.html")
	if _, err := os.Stat(dist); err != nil {
		t.Fatalf("web/dist is missing (run `npm run build` in web/ first): %v", err)
	}

	e := setupWrites(t) // disposable DB + synthetic writes scenario (dropped on cleanup)
	requireLoopbackDSN(t, e.DSN)

	x := e.syntheticCredential(t, "e2e-seller-x", e.users["6"]) // holds lead 101
	y := e.syntheticCredential(t, "e2e-seller-y", e.users["7"]) // holds lead 102

	// Flags ON only inside this disposable database.
	e.setFlags(t, map[string]bool{"identity.session_auth": true, "sales.read_shadow": true, "cases.nonfinancial_write": true})

	api := e.startCRMAPI(t, root)
	web := serveSPA(t, filepath.Join(root, "web", "dist"), api)

	env := map[string]string{
		"E2E_BASE_URL":          web,
		"E2E_SELLER_X_LOGIN":    x.login,
		"E2E_SELLER_X_PASSWORD": x.password,
		"E2E_SELLER_Y_LOGIN":    y.login,
		"E2E_SELLER_Y_PASSWORD": y.password,
		"E2E_CASE_HELD_BY_X":    e.caseIDs["101"],
		"E2E_CASE_HELD_BY_Y":    e.caseIDs["102"],
		"E2E_SELLER_X_USER_ID":  x.userID,
		"E2E_SELLER_Y_USER_ID":  y.userID,
	}
	runPlaywright(t, root, "write-flag-on", env, x, y)

	// Turn the write flag OFF inside the disposable database, then prove both
	// the UI and the API refuse the command.
	e.setFlags(t, map[string]bool{"cases.nonfinancial_write": false})
	runPlaywright(t, root, "write-flag-off", env, x, y)

	// No contact outcome, command record or duplicate was written by any test.
	for _, q := range []string{
		`SELECT count(*) FROM cases.contact_outcome_event`,
		`SELECT count(*) FROM cases.command_record`,
		`SELECT count(*) FROM cases.contact_outcome_current`,
	} {
		if n := e.q(t, q); n != "0" {
			t.Errorf("%s = %s, want 0 (no write may have been applied)", q, n)
		}
	}
}

// syntheticCredential inserts one ACTIVE password credential for a synthetic
// imported user, the way the R1 auth tests do. It grants nothing.
func (e *writesEnv) syntheticCredential(t *testing.T, login, userID string) browserSeller {
	t.Helper()
	if userID == "" {
		t.Fatalf("%s: synthetic user not imported", login)
	}
	var b [18]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}
	pw := "e2e-" + hex.EncodeToString(b[:])
	hash, err := password.Hash(pw, password.Default)
	if err != nil {
		t.Fatal(err)
	}
	cid, _ := ids.NewUUID()
	if _, err := e.admin.Exec(context.Background(), `INSERT INTO identity.credential
		(credential_id, user_id, login_name, kind, secret_hash, status, created_at, updated_at, version)
		VALUES ($1, $2, $3, 'PASSWORD_ARGON2ID', $4, 'ACTIVE', now(), now(), 1)`, cid, userID, login, hash); err != nil {
		t.Fatal(err)
	}
	return browserSeller{login: login, password: pw, userID: userID}
}

// startCRMAPI builds and starts the real cmd/crm-api binary bound to
// 127.0.0.1 with one restricted login per module owner, real R1 sessions and
// CRM_ENV=test (no dev stub). It returns the API base URL.
func (e *writesEnv) startCRMAPI(t *testing.T, root string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "crm-api")
	build := exec.Command("go", "build", "-o", bin, "./cmd/crm-api")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build crm-api: %v\n%s", err, out)
	}
	port := freePort(t)
	addr := "127.0.0.1:" + port
	login := func(m owner.Module) string {
		dsn := e.LoginDSN(t, m)
		requireLoopbackDSN(t, dsn)
		return dsn
	}
	flags := login(owner.FeatureFlags)
	envv := []string{
		"CRM_ENV=test",
		"CRM_API_ADDR=" + addr,
		"CRM_AUTH_ALLOWED_ORIGINS=" + spaOrigin.Load(t),
		"CRM_FEATUREFLAGS_DATABASE_URL=" + flags,
		"CRM_IDENTITY_AUTH_DATABASE_URL=" + login(identity.Module),
		"CRM_IDENTITY_READ_DATABASE_URL=" + login(identity.Module),
		"CRM_ORGANIZATION_READ_DATABASE_URL=" + login(organization.Module),
		"CRM_CASES_READ_DATABASE_URL=" + login(cases.Module),
		"CRM_CUSTODY_READ_DATABASE_URL=" + login(custody.Module),
		"CRM_CASES_WRITE_DATABASE_URL=" + login(cases.Module),
		"CRM_CUSTODY_HOLDER_DATABASE_URL=" + login(custody.Module),
		"CRM_ORGANIZATION_WRITE_DATABASE_URL=" + login(organization.Module),
		"CRM_WRITEBACK_READ_DATABASE_URL=" + login(integration.Module),
	}
	var logs lockedBuffer
	cmd := exec.Command(bin)
	cmd.Env = envv // explicit: nothing inherited from the runner environment
	cmd.Stdout, cmd.Stderr = &logs, &logs
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start crm-api: %v", err)
	}
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	t.Cleanup(func() {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			<-done
		}
		if t.Failed() {
			t.Logf("crm-api log (credentials redacted):\n%s", redact(logs.String(), envv))
		}
	})
	base := "http://" + addr
	deadline := time.Now().Add(60 * time.Second)
	for {
		select {
		case <-done:
			t.Fatalf("crm-api exited during startup:\n%s", redact(logs.String(), envv))
		default:
		}
		resp, err := http.Get(base + "/health/live")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return base
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("crm-api not healthy after 60s:\n%s", redact(logs.String(), envv))
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// originHolder keeps the SPA listener, opened before crm-api starts so the
// allowed origin is known.
type originHolder struct {
	mu sync.Mutex
	ln net.Listener
}

var spaOrigin originHolder

// Load opens (once) the loopback listener for the SPA and returns its origin.
func (o *originHolder) Load(t *testing.T) string {
	t.Helper()
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.ln == nil {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		o.ln = ln
		t.Cleanup(func() {
			o.mu.Lock()
			defer o.mu.Unlock()
			o.ln = nil
		})
	}
	return "http://" + o.ln.Addr().String()
}

// serveSPA serves web/dist (SPA fallback to index.html) on the loopback
// listener and reverse-proxies /api and /health to crm-api, so the browser
// sees one same-origin application. It returns the SPA origin.
func serveSPA(t *testing.T, dist, api string) string {
	t.Helper()
	origin := spaOrigin.Load(t)
	target, err := url.Parse(api)
	if err != nil {
		t.Fatal(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	files := http.FileServer(http.Dir(dist))
	mux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/health/") {
			proxy.ServeHTTP(w, r)
			return
		}
		if _, err := os.Stat(filepath.Join(dist, filepath.Clean("/"+r.URL.Path))); err != nil || r.URL.Path == "/" {
			http.ServeFile(w, r, filepath.Join(dist, "index.html"))
			return
		}
		files.ServeHTTP(w, r)
	})
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	spaOrigin.mu.Lock()
	ln := spaOrigin.ln
	spaOrigin.mu.Unlock()
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})
	return origin
}

// runPlaywright runs one Playwright project. Credentials reach the browser
// process only through its environment and are redacted from any output.
func runPlaywright(t *testing.T, root, project string, env map[string]string, secrets ...browserSeller) {
	t.Helper()
	cmd := exec.Command("npx", "playwright", "test", "--project="+project)
	cmd.Dir = filepath.Join(root, "web")
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME"), "CI=" + os.Getenv("CI")}
	if v := os.Getenv("PLAYWRIGHT_BROWSERS_PATH"); v != "" {
		cmd.Env = append(cmd.Env, "PLAYWRIGHT_BROWSERS_PATH="+v)
	}
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	var pws []string
	for _, s := range secrets {
		pws = append(pws, s.password)
	}
	t.Logf("playwright --project=%s:\n%s", project, redact(out.String(), pws))
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			t.Fatalf("playwright project %s failed (exit %d)", project, exit.ExitCode())
		}
		t.Fatalf("playwright project %s did not run: %v", project, err)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		t.Fatalf("repository root not found from %s", dir)
	}
	return dir
}

func freePort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	_, p, _ := net.SplitHostPort(ln.Addr().String())
	return p
}

// requireLoopbackDSN refuses any database that is not on this machine.
func requireLoopbackDSN(t *testing.T, dsn string) {
	t.Helper()
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("unparsable DSN")
	}
	switch h := u.Hostname(); h {
	case "127.0.0.1", "localhost", "::1":
	default:
		t.Fatalf("refusing non-loopback database host %q: the E2E suite runs only against a disposable local database", h)
	}
}

// redact removes every secret value (passwords, DSNs) from s.
func redact(s string, secrets []string) string {
	for _, v := range secrets {
		if i := strings.IndexByte(v, '='); i >= 0 && strings.Contains(v[:i], "DATABASE_URL") {
			v = v[i+1:]
		}
		if len(v) >= 8 {
			s = strings.ReplaceAll(s, v, "[redacted]")
		}
	}
	return s
}

type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return fmt.Sprint(l.b.String())
}
