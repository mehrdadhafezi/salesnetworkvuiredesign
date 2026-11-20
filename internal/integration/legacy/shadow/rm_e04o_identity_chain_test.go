package shadow_test

// RM-E04O — Seller identity chain characterization (SYNTHETIC data, disposable PostgreSQL, TEST ONLY).
//
// Question: can the existing import → authentication → session pipeline produce a trustworthy equality between
// authorization.Subject.UserID and custody CurrentHolder.UserID for the same synthetic Seller?
//
// Real components exercised (no mocks): legacy snapshot staging (staging.Loader), shadow.IdentityCommand +
// identity/imports, shadow.OrganizationCommand + organization/imports, identity/query ResolveUsers,
// shadow.CasesCommand + cases/imports, shadow.CustodyCommand + custody/imports, custody/query CurrentHolder,
// identity/session Login / Authenticate / Logout / LogoutAll, the production HTTP stack (apiserver +
// identity/http, POST /api/v1/auth/login and GET /api/v1/session), authorization.SessionSubjects with the real
// identity ActiveGrants loader, and the existing Slice 11 row-scope comparator cases/command Eligible.
//
// Seams that are NOT production code and are labelled where used:
//   - CREDENTIAL PROVISIONING: there is no enrolment command (D-34 open; the application role has SELECT only on
//     identity.credential). A credential row is inserted by the test superuser for the account the real import
//     created, exactly as internal/identity/session tests do. This is the only way to obtain a session today.
//   - PRINCIPAL INJECTION: apiserver.sessionAuth is unexported. To reach SessionSubjects the test calls the same
//     websession.Authenticator (session.Service.Authenticate) the middleware calls and attaches the principal with
//     websession.WithPrincipal / WithState, which is the middleware's own two-line success branch. The HTTP leg is
//     verified separately through GET /api/v1/session.
//   - DISABLE / KEY CHANGE: there is no command that disables a credential or edits an account key; the test does
//     it with SQL (superuser, or the identity owner role inside a rolled-back transaction) and says so.
//
// Every assertion is CURRENT_BEHAVIOR_CHARACTERIZATION. A passing assertion is not an approved policy, does not
// prove Seller row-scope enforcement (no own-read exists) and opens nothing. Group B stays closed. Tokens are never
// logged.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases"
	casescommand "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/command"
	custodyapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/api"
	custodyquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	identityhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/password"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/session"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/legacytest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/shadow"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/websession"
)

// Synthetic legacy WordPress user keys (HRScenario accounts 1..13; profiles in legacytest/hr.go).
const (
	oSellerA  = "3"  // profile 103, legacy role sn_seller, employment active
	oSellerB  = "9"  // profile 109, legacy role sn_seller, employment active
	oInactive = "10" // profile 110, employment inactive
	oNoCred   = "11" // imported account, never given a credential
	oUnmapped = "77" // referenced by a lead, no wp_user row
	oPassword = "synthetic-RM-E04O-pass-1"
	oOrigin   = "https://crm.example.test"
	oLead     = shadow.SpecLead
	oItem     = shadow.SpecDistItem
	oGatePath = "../../../../docs/governance/rm-e04c-group-b-gate.json"
	oCorr     = "rm-e04o"
)

var oFast = password.Params{MemoryKiB: 64, Iterations: 1, Parallelism: 1, SaltBytes: 16, KeyBytes: 32}

type oClock struct {
	mu sync.Mutex
	at time.Time
}

func (c *oClock) Now() timeutil.Instant {
	c.mu.Lock()
	defer c.mu.Unlock()
	i, _ := timeutil.NewInstant(c.at)
	return i
}

func (c *oClock) add(d time.Duration) {
	c.mu.Lock()
	c.at = c.at.Add(d)
	c.mu.Unlock()
}

// oHolders is identical to cmd/crm-api/writes.go holderAdapter (composition root, package main, not importable).
type oHolders struct{ q *custodyquery.Service }

func (h oHolders) Holder(ctx context.Context, caseID string) (casescommand.Holder, error) {
	c, err := h.q.CurrentHolder(ctx, caseID)
	return casescommand.Holder{Known: c.Known, UserID: c.UserID}, err
}

type oEnv struct {
	*salesEnv
	clk      *oClock
	flag     atomic.Bool
	sess     *session.Service
	subjects authorization.SessionSubjects
	cmd      *casescommand.Service
	h        http.Handler
}

func oUsers(extra ...string) []legacytest.Rec {
	var out []legacytest.Rec
	for i := 1; i <= 13; i++ {
		out = append(out, legacytest.Rec{"ID": fmt.Sprint(i), "user_login": fmt.Sprintf("dev-hr-user-%03d", i),
			"display_name": fmt.Sprintf("Synthetic Account %03d", i), "user_registered": "2025-01-01 00:00:00"})
	}
	for _, k := range extra {
		out = append(out, legacytest.Rec{"ID": k, "user_login": "dev-rm-e04o-" + k, "display_name": "Synthetic RM-E04O " + k,
			"user_registered": "2025-01-01 00:00:00"})
	}
	return out
}

// oScenario: HRScenario (identity + organization) plus SYNTHETIC sales rows. Leads have no event log
// (TARGET_UNKNOWN); items 601/602 have one assignment event each (DERIVED / MATCH).
func oScenario(users []legacytest.Rec) []snapshot.EntityRows {
	ents := legacytest.With(legacytest.HRScenario(), shadow.SpecUser, users, snapshot.Complete)
	ents = legacytest.With(ents, oLead, []legacytest.Rec{
		legacytest.Lead("701", "assigned", oSellerA, ""), legacytest.Lead("702", "assigned", oSellerB, ""),
		legacytest.Lead("703", "assigned", oUnmapped, ""), legacytest.Lead("704", "assigned", oInactive, ""),
		legacytest.Lead("705", "assigned", oNoCred, ""),
	}, snapshot.Complete)
	ents = legacytest.With(ents, oItem, []legacytest.Rec{
		legacytest.DistItem("601", oSellerA, "seller", "", "delivered_to_seller"),
		legacytest.DistItem("602", oSellerB, "seller", "", "delivered_to_seller"),
	}, snapshot.Complete)
	return legacytest.With(ents, shadow.SpecDistItemLog, []legacytest.Rec{
		legacytest.DistLog("61", "601", "item_created_from_mis_pool", "", oSellerA, "seller", "2025-02-01 08:00:00"),
		legacytest.DistLog("62", "602", "item_created_from_mis_pool", "", oSellerB, "seller", "2025-02-01 09:00:00"),
	}, snapshot.Complete)
}

func setupRME04O(t *testing.T) *oEnv {
	t.Helper()
	e := &oEnv{salesEnv: setupSales(t), clk: &oClock{at: time.Date(2026, 11, 7, 9, 0, 0, 0, time.UTC)}}
	e.flag.Store(true)
	svc, err := session.New(e.Runner(t, identity.Module), e.clk, session.DefaultConfig, func(context.Context) (bool, error) { return e.flag.Load(), nil })
	if err != nil {
		t.Fatal(err)
	}
	e.sess = svc
	// Production composition: cmd/crm-api/auth.go sets SessionSubjects{Grants: identity ActiveGrants loader}.
	e.subjects = authorization.SessionSubjects{Grants: e.ident}
	if e.cmd, err = casescommand.New(e.Runner(t, cases.Module), oHolders{e.custodyQ}, e.clk); err != nil {
		t.Fatal(err)
	}
	auth, err := identityhttp.NewAuth(svc, e.clk)
	if err != nil {
		t.Fatal(err)
	}
	cfg := apiserver.DefaultConfig() // production configuration
	cfg.AllowedOrigins = []string{oOrigin}
	if e.h, err = apiserver.NewHandlerWithSurfaces(cfg, e.clk, slog.New(slog.NewTextHandler(io.Discard, nil)), apiserver.Surfaces{Auth: auth}); err != nil {
		t.Fatal(err)
	}
	b := e.stage(t, "syn-rm-e04o-001", oScenario(oUsers()))
	e.importAll(t, b)
	if rc, rs, _ := e.importSales(t, b); rc == result.Failed || rs == result.Failed {
		t.Fatalf("sales import: cases %s custody %s", rc, rs)
	}
	return e
}

func (e *oEnv) uid(t *testing.T, key string) string {
	t.Helper()
	m, err := e.ident.ResolveUsers(context.Background(), shadow.SourceSystem, []string{key})
	if err != nil {
		t.Fatal(err)
	}
	return m[key]
}

func (e *oEnv) caseID(t *testing.T, spec, key string) string {
	t.Helper()
	var id string
	if err := e.admin.QueryRow(context.Background(), `SELECT case_id::text FROM cases.case_source WHERE source_spec = $1 AND source_key = $2`,
		spec, key).Scan(&id); err != nil {
		t.Fatalf("case %s/%s: %v", spec, key, err)
	}
	return id
}

// provision is the CREDENTIAL PROVISIONING seam (see file comment): no production enrolment path exists.
func (e *oEnv) provision(t *testing.T, userID, login string) string {
	t.Helper()
	h, err := password.Hash(oPassword, oFast)
	if err != nil {
		t.Fatal(err)
	}
	cid, _ := ids.NewUUID()
	if _, err := e.admin.Exec(context.Background(), `INSERT INTO identity.credential (credential_id, user_id, login_name, kind, secret_hash, status,
		created_at, updated_at, version) VALUES ($1, $2, $3, 'PASSWORD_ARGON2ID', $4, 'ACTIVE', now(), now(), 1)`, cid, userID, login, h); err != nil {
		t.Fatalf("provision %s: %v", login, err)
	}
	return cid
}

type oResp struct {
	code   int
	json   map[string]any
	cookie string
}

func (e *oEnv) do(method, path, cookie string, body any) oResp {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	q := httptest.NewRequest(method, path, rd)
	q.Host = "crm.example.test"
	if body != nil {
		q.Header.Set("Content-Type", "application/json")
	}
	if method != http.MethodGet {
		q.Header.Set("Origin", oOrigin)
		q.Header.Set("Sec-Fetch-Site", "same-origin")
	}
	if cookie != "" {
		q.AddCookie(&http.Cookie{Name: websession.CookieName, Value: cookie})
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, q)
	out := oResp{code: rec.Code}
	_ = json.Unmarshal(rec.Body.Bytes(), &out.json)
	for _, c := range rec.Result().Cookies() {
		if c.Name == websession.CookieName && c.MaxAge >= 0 {
			out.cookie = c.Value
		}
	}
	return out
}

// httpLogin logs in through the production HTTP stack and returns the session token (never logged).
func (e *oEnv) httpLogin(t *testing.T, login string) string {
	t.Helper()
	r := e.do(http.MethodPost, "/api/v1/auth/login", "", map[string]string{"login": login, "password": oPassword})
	if r.code != http.StatusOK || r.cookie == "" {
		t.Fatalf("HTTP login %s: status %d, cookie issued %v", login, r.code, r.cookie != "")
	}
	return r.cookie
}

// httpPrincipal returns (authenticated, session.user_id) as GET /api/v1/session reports them.
func (e *oEnv) httpPrincipal(t *testing.T, token string) (bool, string) {
	t.Helper()
	r := e.do(http.MethodGet, "/api/v1/session", token, nil)
	data, _ := r.json["data"].(map[string]any)
	p, _ := data["session"].(map[string]any)
	id, _ := p["user_id"].(string)
	return data["authenticated"] == true, id
}

// subject is the PRINCIPAL INJECTION seam: Authenticate is the call apiserver.sessionAuth makes; on success the
// middleware attaches exactly WithPrincipal + WithState(StateActive). On failure no principal is attached.
func (e *oEnv) subject(t *testing.T, token string) (authorization.Subject, bool) {
	t.Helper()
	ctx := context.Background()
	p, ok, err := e.sess.Authenticate(ctx, token)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if ok {
		ctx = websession.WithState(websession.WithPrincipal(ctx, p), websession.StateActive)
	}
	return e.subjects.Subject(ctx), ok
}

func (e *oEnv) holder(t *testing.T, caseID string) custodyapi.CurrentHolder {
	t.Helper()
	h, err := e.custodyQ.CurrentHolder(context.Background(), caseID)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

// eligible runs the existing Slice 11 row-scope comparator with the actor built as internal/cases/http/write.go does.
func (e *oEnv) eligible(t *testing.T, caseID string, s authorization.Subject) (bool, reasoncodes.Code) {
	t.Helper()
	ok, code, err := e.cmd.Eligible(context.Background(), caseID, casescommand.Actor{Principal: s.PrincipalID, UserID: s.UserID})
	if err != nil {
		t.Fatalf("eligible: %v", err)
	}
	return ok, code
}

func pgCode(err error) string {
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		return pe.Code + "/" + pe.ConstraintName
	}
	return fmt.Sprint(err)
}

var errRollback = errors.New("rm-e04o: rollback")

// ownerTx runs fn as the identity owner (application) role and always rolls back.
func (e *oEnv) ownerTx(t *testing.T, fn func(ctx context.Context, tx db.Tx) error) error {
	t.Helper()
	err := e.Runner(t, identity.Module).InTx(context.Background(), db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		if err := fn(ctx, tx); err != nil {
			return err
		}
		return errRollback
	})
	if errors.Is(err, errRollback) {
		return nil
	}
	return err
}

// Test A — valid identity continuity over the longest real chain.
func TestRME04OIdentityContinuity(t *testing.T) {
	e := setupRME04O(t)
	ctx := context.Background()
	uidA := e.uid(t, oSellerA)
	if uidA == "" {
		t.Fatal("seller A not imported")
	}
	// 1. Import: legacy key → identity.user_account (shadow account, no login enablement).
	var sys, key, presence, state string
	var loginEnabled bool
	var loginEvidence *string
	if err := e.admin.QueryRow(ctx, `SELECT source_system, source_key, presence, account_state, login_enabled, login_evidence
		FROM identity.user_account WHERE user_id = $1`, uidA).Scan(&sys, &key, &presence, &state, &loginEnabled, &loginEvidence); err != nil {
		t.Fatal(err)
	}
	if sys != shadow.SourceSystem || key != oSellerA || presence != "PRESENT" || state != "SHADOW_IMPORTED" || loginEnabled ||
		loginEvidence == nil || *loginEvidence != "dev-hr-user-003" {
		t.Fatalf("account: %s/%s %s %s login_enabled=%v", sys, key, presence, state, loginEnabled)
	}
	// 2. Custody import: legacy holder key → identity user id (resolved through identity ResolveUsers).
	lead, item := e.caseID(t, oLead, "701"), e.caseID(t, oItem, "601")
	for _, c := range []struct{ id, parity string }{{lead, "TARGET_UNKNOWN"}, {item, "MATCH"}} {
		var hk, hu, parity string
		if err := e.admin.QueryRow(ctx, `SELECT legacy_holder_key, legacy_holder_user_id::text, parity FROM custody.assignment WHERE case_id = $1`,
			c.id).Scan(&hk, &hu, &parity); err != nil {
			t.Fatal(err)
		}
		if hk != oSellerA || hu != uidA || parity != c.parity {
			t.Fatalf("assignment %s: key %s user %s parity %s", c.id, hk, hu, parity)
		}
	}
	// 3. Authentication through the production HTTP stack (credential = PROVISIONING seam).
	cred := e.provision(t, uidA, "rm-e04o.seller.a")
	token := e.httpLogin(t, "rm-e04o.seller.a")
	digest, _ := websession.Digest(token)
	if e.count(t, `SELECT count(*) FROM identity.session WHERE token_sha256 = $1 AND user_id = $2 AND credential_id = $3 AND revoked_at IS NULL`,
		digest[:], uidA, cred) != 1 {
		t.Fatal("identity.session must bind the token digest to the imported account and its credential")
	}
	if ok, pid := e.httpPrincipal(t, token); !ok || pid != uidA {
		t.Fatalf("GET /api/v1/session: authenticated=%v principal matches=%v", ok, pid == uidA)
	}
	// 4. Session → SessionSubjects → Subject.UserID.
	s, ok := e.subject(t, token)
	if !ok || s.UserID != uidA || s.PrincipalID != uidA || len(s.Grants) != 0 || s.EvidenceConflict {
		t.Fatalf("subject: ok=%v user match=%v grants=%d conflict=%v", ok, s.UserID == uidA, len(s.Grants), s.EvidenceConflict)
	}
	// 5. CurrentHolder.UserID, and the existing comparator.
	for _, id := range []string{lead, item} {
		if h := e.holder(t, id); !h.Known || h.UserID != s.UserID || h.Basis != "LEGACY_HOLDER_COLUMNS" {
			t.Fatalf("holder %s: %+v", id, h)
		}
	}
	if ok, code := e.eligible(t, lead, s); !ok {
		t.Fatalf("Slice 11 comparator on own lead: %s", code)
	}
	if ok, code := e.eligible(t, item, s); ok || code != reasoncodes.PolicyUnresolved {
		t.Fatalf("item case is outside the Slice 11 command: %v %s", ok, code)
	}
	// 6. One join over authoritative records: session.user_id = user_account(legacy key) = assignment holder.
	if e.count(t, `SELECT count(*) FROM identity.session s JOIN identity.user_account u ON u.user_id = s.user_id
		JOIN custody.assignment a ON a.legacy_holder_user_id = u.user_id AND a.legacy_holder_key = u.source_key
		WHERE s.token_sha256 = $1 AND u.source_system = $2 AND a.case_id IN ($3, $4)`, digest[:], shadow.SourceSystem, lead, item) != 2 {
		t.Fatal("chain join must hold for both cases")
	}
}

// Test B — two Sellers resolve to distinct accounts; no swapped pair matches.
func TestRME04OCrossSellerSeparation(t *testing.T) {
	e := setupRME04O(t)
	uidA, uidB := e.uid(t, oSellerA), e.uid(t, oSellerB)
	if uidA == "" || uidB == "" || uidA == uidB {
		t.Fatal("sellers A and B must resolve to two distinct accounts")
	}
	e.provision(t, uidA, "rm-e04o.seller.a")
	e.provision(t, uidB, "rm-e04o.seller.b")
	sA, _ := e.subject(t, e.httpLogin(t, "rm-e04o.seller.a"))
	sB, _ := e.subject(t, e.httpLogin(t, "rm-e04o.seller.b"))
	if sA.UserID != uidA || sB.UserID != uidB {
		t.Fatal("subjects must carry their own account ids")
	}
	owner := map[string]string{e.caseID(t, oLead, "701"): uidA, e.caseID(t, oItem, "601"): uidA,
		e.caseID(t, oLead, "702"): uidB, e.caseID(t, oItem, "602"): uidB}
	matches := 0
	for c, want := range owner {
		h := e.holder(t, c)
		if !h.Known || h.UserID != want {
			t.Fatalf("holder %s: %+v", c, h)
		}
		for _, s := range []authorization.Subject{sA, sB} {
			eq := h.UserID == s.UserID
			if eq != (s.UserID == want) {
				t.Fatalf("false identity match on case %s", c)
			}
			if eq {
				matches++
			}
		}
	}
	if matches != 4 {
		t.Fatalf("each subject must match exactly its own two cases, got %d matches", matches)
	}
	for _, x := range []struct {
		lead string
		s    authorization.Subject
		ok   bool
	}{{"701", sA, true}, {"702", sB, true}, {"701", sB, false}, {"702", sA, false}} {
		ok, code := e.eligible(t, e.caseID(t, oLead, x.lead), x.s)
		if ok != x.ok || (!ok && code != reasoncodes.RowScopeDenied) {
			t.Fatalf("comparator lead %s: %v %s", x.lead, ok, code)
		}
	}
}

// Test C — a holder key with no identity account.
func TestRME04OMissingIdentityMapping(t *testing.T) {
	e := setupRME04O(t)
	c703 := e.caseID(t, oLead, "703")
	if e.count(t, `SELECT count(*) FROM identity.user_account WHERE source_key = $1`, oUnmapped) != 0 {
		t.Fatal("no account may be invented for an unmapped key")
	}
	if e.count(t, `SELECT count(*) FROM custody.assignment WHERE case_id = $1 AND legacy_holder_key = $2 AND legacy_holder_user_id IS NULL
		AND presence = 'PRESENT'`, c703, oUnmapped) != 1 {
		t.Fatal("custody must keep the legacy key and leave the user id NULL (stored, not rejected)")
	}
	if h := e.holder(t, c703); h.Known || h.UserID != "" || h.Basis != "NO_RESOLVED_LEGACY_HOLDER" {
		t.Fatalf("holder: %+v", h)
	}
	uidA := e.uid(t, oSellerA)
	e.provision(t, uidA, "rm-e04o.seller.a")
	sA, _ := e.subject(t, e.httpLogin(t, "rm-e04o.seller.a"))
	if ok, code := e.eligible(t, c703, sA); ok || code != reasoncodes.DependencyUnknown {
		t.Fatalf("comparator: %v %s", ok, code)
	}
	// A later batch that imports the account resolves the orphan on the next sales import.
	b := e.stage(t, "syn-rm-e04o-002", oScenario(oUsers(oUnmapped)))
	e.importAll(t, b)
	e.importSales(t, b)
	u77 := e.uid(t, oUnmapped)
	if h := e.holder(t, c703); u77 == "" || !h.Known || h.UserID != u77 {
		t.Fatalf("after the account is imported: %+v", h)
	}
}

// Test D — nonexistent, credential-less, absent and HR-inactive accounts.
func TestRME04ONonexistentAndInactiveAccounts(t *testing.T) {
	e := setupRME04O(t)
	ctx := context.Background()
	c705 := e.caseID(t, oLead, "705")
	uid11 := e.uid(t, oNoCred)

	// D1. Custody import accepts a well-formed uuid that is not an identity account, and a key/uuid pair that
	// contradicts identity (key 11 belongs to uid11). Only the import command contract allows this; the shadow
	// connector always resolves through identity.
	ghost, _ := ids.NewUUID()
	batch, _ := ids.NewUUID()
	rep, err := e.custIn.ImportCustody(ctx, custodyapi.CustodyImport{BatchID: batch, SnapshotID: "syn-rm-e04o-ghost", SourceSystem: shadow.SourceSystem,
		Complete: map[string]bool{}, Holders: []custodyapi.HolderRecord{{Case: custodyapi.RowRef{Spec: oLead, Key: "705"},
			Holder: &custodyapi.UserRef{LegacyKey: oNoCred, UserID: &ghost}}}})
	if err != nil || rep.Outcome != result.Success {
		t.Fatalf("ghost custody import: %s %v", rep.Outcome, err)
	}
	if h := e.holder(t, c705); !h.Known || h.UserID != ghost || h.UserID == uid11 {
		t.Fatalf("CURRENT_BEHAVIOR: custody admits an identity-unverified uuid: %+v", h)
	}
	if e.count(t, `SELECT count(*) FROM identity.user_account WHERE user_id = $1`, ghost) != 0 {
		t.Fatal("ghost must not exist in identity")
	}
	// ... but no session can ever carry that uuid: the credential foreign key refuses it.
	h, _ := password.Hash(oPassword, oFast)
	cid, _ := ids.NewUUID()
	_, err = e.admin.Exec(ctx, `INSERT INTO identity.credential (credential_id, user_id, login_name, kind, secret_hash, status, created_at, updated_at, version)
		VALUES ($1, $2, 'rm-e04o.ghost', 'PASSWORD_ARGON2ID', $3, 'ACTIVE', now(), now(), 1)`, cid, ghost, h)
	if got := pgCode(err); got != "23503/credential_user_id_fkey" {
		t.Fatalf("credential for a nonexistent account: %s", got)
	}

	// D2. Imported account without a credential: no login, whatever name is tried (the legacy user_login is
	// login_evidence only, never a login name).
	for _, name := range []string{"dev-hr-user-011", "rm-e04o.nocred"} {
		r, err := e.sess.Login(ctx, session.Login{Name: name, Password: oPassword, CorrelationID: oCorr})
		if err != nil || r.Status != session.Invalid || r.Issued != nil {
			t.Fatalf("login %s: %s", name, r.Status)
		}
	}
	if e.count(t, `SELECT count(*) FROM identity.session WHERE user_id = $1`, uid11) != 0 {
		t.Fatal("no session for an account without a credential")
	}

	// D3. HR employment INACTIVE does not affect authentication (session reads identity only).
	uid10 := e.uid(t, oInactive)
	var status string
	if err := e.admin.QueryRow(ctx, `SELECT employment_status FROM organization.employment WHERE user_id = $1`, uid10).Scan(&status); err != nil {
		t.Fatal(err)
	}
	e.provision(t, uid10, "rm-e04o.inactive")
	s10, ok := e.subject(t, e.httpLogin(t, "rm-e04o.inactive"))
	if status != "INACTIVE" || !ok || s10.UserID != uid10 {
		t.Fatalf("CURRENT_BEHAVIOR: employment %s, session subject resolved=%v", status, ok)
	}
	if h := e.holder(t, e.caseID(t, oLead, "704")); !h.Known || h.UserID != uid10 {
		t.Fatalf("holder for the HR-inactive seller: %+v", h)
	}

	// D4. Account ABSENT_IN_LATEST (a COMPLETE identity batch omits seller B): sessions stop resolving and login
	// is refused, but custody still names B, also after a fresh sales import (ResolveUsers ignores presence).
	uidB := e.uid(t, oSellerB)
	e.provision(t, uidB, "rm-e04o.seller.b")
	tokB := e.httpLogin(t, "rm-e04o.seller.b")
	var without []legacytest.Rec
	for _, u := range oUsers() {
		if u["ID"] != oSellerB {
			without = append(without, u)
		}
	}
	b := e.stage(t, "syn-rm-e04o-003", oScenario(without))
	e.importAll(t, b)
	if e.count(t, `SELECT count(*) FROM identity.user_account WHERE user_id = $1 AND presence = 'ABSENT_IN_LATEST'`, uidB) != 1 {
		t.Fatal("seller B must be ABSENT_IN_LATEST")
	}
	if s, ok := e.subject(t, tokB); ok || s.UserID != "" || s.PrincipalID != "" {
		t.Fatal("a session of an absent account must not produce a subject")
	}
	if r, _ := e.sess.Login(ctx, session.Login{Name: "rm-e04o.seller.b", Password: oPassword, CorrelationID: oCorr}); r.Status != session.Invalid {
		t.Fatalf("login of an absent account: %s", r.Status)
	}
	if e.count(t, `SELECT count(*) FROM identity.auth_event WHERE user_id = $1 AND reason = 'ACCOUNT_NOT_PRESENT'`, uidB) != 1 {
		t.Fatal("refusal reason must be recorded")
	}
	e.importSales(t, b)
	if h := e.holder(t, e.caseID(t, oLead, "702")); !h.Known || h.UserID != uidB {
		t.Fatalf("CURRENT_BEHAVIOR: custody keeps naming an identity-absent account: %+v", h)
	}
}

// Test E — source-key repetition, cross-system keys, collisions and reassignment.
func TestRME04OSourceKeyCollisions(t *testing.T) {
	e := setupRME04O(t)
	ctx := context.Background()
	uidA := e.uid(t, oSellerA)

	// E1. Same key, re-imported consistently and with changed content: never a second account.
	b := e.stage(t, "syn-rm-e04o-004", oScenario(func() []legacytest.Rec {
		us := oUsers()
		us[2]["display_name"] = "Synthetic Account 003 renamed"
		return us
	}()))
	e.importAll(t, b)
	e.importSales(t, b)
	var version int64
	if err := e.admin.QueryRow(ctx, `SELECT version FROM identity.user_account WHERE user_id = $1`, uidA).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if e.uid(t, oSellerA) != uidA || version != 2 ||
		e.count(t, `SELECT count(*) FROM identity.user_account WHERE source_system = $1 AND source_key = $2`, shadow.SourceSystem, oSellerA) != 1 {
		t.Fatal("refresh must keep the account id and never fork")
	}

	// E2. Same key in another source system is a different account; resolution is per system.
	ob, _ := ids.NewUUID()
	sha := [32]byte{0xE0, 0x4F}
	rep, err := e.identIn.ImportIdentity(ctx, identityapi.IdentityImport{BatchID: ob, SnapshotID: "syn-rm-e04o-other", SourceSystem: "syn_other",
		Users: []identityapi.UserRecord{{Source: identityapi.SourceRef{System: "syn_other", Spec: "syn_other.user.v1", Key: oSellerA, BatchID: ob, RowSHA256: sha[:]}}}})
	if err != nil || rep.Outcome == result.Failed {
		t.Fatalf("other-system import: %s %v", rep.Outcome, err)
	}
	other, err := e.ident.ResolveUsers(ctx, "syn_other", []string{oSellerA})
	if err != nil || other[oSellerA] == "" || other[oSellerA] == uidA || e.uid(t, oSellerA) != uidA {
		t.Fatal("the same key in two systems must be two accounts, resolved per system")
	}
	if h := e.holder(t, e.caseID(t, oLead, "701")); h.UserID != uidA {
		t.Fatal("the legacy_wp custody path must keep resolving the legacy_wp account")
	}

	// E3. A duplicate key inside one batch is rejected before any write.
	dup, _ := ids.NewUUID()
	u := identityapi.UserRecord{Source: identityapi.SourceRef{System: shadow.SourceSystem, Spec: shadow.SpecUser, Key: oSellerA, BatchID: dup, RowSHA256: sha[:]}}
	if _, err := e.identIn.ImportIdentity(ctx, identityapi.IdentityImport{BatchID: dup, SnapshotID: "syn-rm-e04o-dup", SourceSystem: shadow.SourceSystem,
		Users: []identityapi.UserRecord{u, u}}); !errors.Is(err, identityapi.ErrInvalidImport) {
		t.Fatalf("duplicate key in batch: %v", err)
	}

	// E4. Database: a second account for an existing (system, key) is refused.
	sid, _ := ids.NewUUID()
	_, err = e.admin.Exec(ctx, `INSERT INTO identity.user_account (user_id, source_system, source_spec, source_key, first_batch_id, source_batch_id,
		source_row_sha256, imported_at, refreshed_at, presence, account_state, login_enabled, version)
		VALUES ($1, $2, $3, $4, $1, $1, $5, now(), now(), 'PRESENT', 'SHADOW_IMPORTED', false, 1)`, sid, shadow.SourceSystem, shadow.SpecUser, oSellerA, sha[:])
	if got := pgCode(err); got != "23505/user_account_source_unique" {
		t.Fatalf("collision: %s", got)
	}

	// E5. Reassignment by the identity owner (application) role, in rolled-back transactions. No code path does
	// this; the database permits some of it.
	e.provision(t, uidA, "rm-e04o.seller.a")
	err = e.ownerTx(t, func(ctx context.Context, tx db.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE identity.user_account SET source_key = $2 WHERE user_id = $1`, uidA, oSellerB)
		return err
	})
	if got := pgCode(err); got != "23505/user_account_source_unique" {
		t.Fatalf("key swap onto another account: %s", got)
	}
	err = e.ownerTx(t, func(ctx context.Context, tx db.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE identity.user_account SET source_key = '424242' WHERE user_id = $1`, uidA)
		return err
	})
	if err != nil {
		t.Fatalf("CURRENT_BEHAVIOR: the owner role may re-key an account (no immutability guard); got %v", err)
	}
	err = e.ownerTx(t, func(ctx context.Context, tx db.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE identity.user_account SET user_id = gen_random_uuid() WHERE user_id = $1`, uidA)
		return err
	})
	if got := pgCode(err); got != "23503/credential_user_id_fkey" && got != "23503/role_grant_user_id_fkey" {
		t.Fatalf("user id change with dependants: %s", got)
	}
	// Account 6 has no credential, grant or profile: its user id can be changed, and nothing in custody would
	// notice (custody holds user ids without a foreign key).
	uid6 := e.uid(t, "6")
	err = e.ownerTx(t, func(ctx context.Context, tx db.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE identity.user_account SET user_id = gen_random_uuid() WHERE user_id = $1`, uid6)
		return err
	})
	if uid6 == "" || err != nil {
		t.Fatalf("CURRENT_BEHAVIOR: user id change without dependants is permitted; got %v", err)
	}
	if e.uid(t, "6") != uid6 {
		t.Fatal("the probe must have been rolled back")
	}
}

// Test F — session lifecycle: only an active session yields a subject.
func TestRME04OSessionLifecycle(t *testing.T) {
	e := setupRME04O(t)
	ctx := context.Background()
	uidA := e.uid(t, oSellerA)
	e.provision(t, uidA, "rm-e04o.seller.a")
	lead := e.caseID(t, oLead, "701")
	denied := func(name, token string) {
		t.Helper()
		s, ok := e.subject(t, token)
		if ok || !reflect.DeepEqual(s, authorization.Subject{}) {
			t.Fatalf("%s: a subject was produced", name)
		}
		if ok, code := e.eligible(t, lead, s); ok || code != reasoncodes.RowScopeDenied {
			t.Fatalf("%s: comparator %v %s", name, ok, code)
		}
		if d := authorization.Evaluate(authorization.Runtime, s, authorization.Request{Capability: identityapi.CapUserRead}); d.Allowed || d.Outcome != result.Unauthorized {
			t.Fatalf("%s: evaluator %+v", name, d)
		}
	}
	login := func() (string, websession.Principal) {
		t.Helper()
		r, err := e.sess.Login(ctx, session.Login{Name: "rm-e04o.seller.a", Password: oPassword, CorrelationID: oCorr})
		if err != nil || r.Status != session.OK {
			t.Fatalf("login: %s %v", r.Status, err)
		}
		return r.Issued.Token, r.Issued.Principal
	}

	tok, p := login()
	if s, ok := e.subject(t, tok); !ok || s.UserID != uidA {
		t.Fatal("valid session must yield the seller subject")
	}
	// Missing session: nothing on the context.
	if s := e.subjects.Subject(ctx); !reflect.DeepEqual(s, authorization.Subject{}) {
		t.Fatal("missing session")
	}
	// Invalid tokens: malformed and well-formed-but-unknown.
	random, _ := websession.NewToken()
	denied("malformed token", "not-a-session-token")
	denied("unknown token", random)
	if ok, _ := e.httpPrincipal(t, random); ok {
		t.Fatal("HTTP: unknown token authenticated")
	}
	// Logout.
	if n, err := e.sess.Logout(ctx, p, oCorr); err != nil || !n {
		t.Fatalf("logout: %v %v", n, err)
	}
	denied("revoked (logout)", tok)
	if ok, _ := e.httpPrincipal(t, tok); ok {
		t.Fatal("HTTP: revoked token authenticated")
	}
	// Logout-all.
	t1, p1 := login()
	t2, _ := login()
	if n, err := e.sess.LogoutAll(ctx, p1, oCorr); err != nil || n != 2 {
		t.Fatalf("logout-all: %d %v", n, err)
	}
	denied("revoked (logout-all) 1", t1)
	denied("revoked (logout-all) 2", t2)
	// Replaced at login (fixation defence).
	t3, p3 := login()
	if r, err := e.sess.Login(ctx, session.Login{Name: "rm-e04o.seller.a", Password: oPassword, ReplaceSessionID: p3.SessionID, CorrelationID: oCorr}); err != nil || r.Status != session.OK {
		t.Fatal("re-login")
	}
	denied("replaced at login", t3)
	// Idle expiry.
	t4, _ := login()
	e.clk.add(session.DefaultConfig.IdleTimeout + time.Second)
	denied("idle expired", t4)
	// Absolute expiry: kept alive below the idle limit until the absolute lifetime ends.
	t5, _ := login()
	for elapsed := time.Duration(0); elapsed < session.DefaultConfig.AbsoluteLifetime-25*time.Minute; elapsed += 25 * time.Minute {
		e.clk.add(25 * time.Minute)
		if _, ok := e.subject(t, t5); !ok {
			t.Fatalf("session dropped before absolute expiry at %s", elapsed)
		}
	}
	e.clk.add(26 * time.Minute)
	denied("absolute expired", t5)
	// Kill switch.
	t6, _ := login()
	e.flag.Store(false)
	denied("session_auth flag OFF", t6)
	if r, _ := e.sess.Login(ctx, session.Login{Name: "rm-e04o.seller.a", Password: oPassword, CorrelationID: oCorr}); r.Status != session.Disabled {
		t.Fatalf("flag OFF login: %s", r.Status)
	}
	e.flag.Store(true)
	// Credential disabled after the session was issued (no command exists; SQL as an operator would).
	t7, _ := login()
	if _, err := e.admin.Exec(ctx, `UPDATE identity.credential SET status = 'DISABLED', version = version + 1 WHERE user_id = $1`, uidA); err != nil {
		t.Fatal(err)
	}
	denied("credential disabled after issue", t7)
	d7, _ := websession.Digest(t7)
	if e.count(t, `SELECT count(*) FROM identity.session WHERE token_sha256 = $1 AND revoked_at IS NULL`, d7[:]) != 1 {
		t.Fatal("CURRENT_BEHAVIOR: disabling the credential does not revoke the row; resolution refuses it")
	}
	if r, _ := e.sess.Login(ctx, session.Login{Name: "rm-e04o.seller.a", Password: oPassword, CorrelationID: oCorr}); r.Status != session.Invalid {
		t.Fatalf("login with disabled credential: %s", r.Status)
	}
	// Custody is unaffected by any of this: the holder fact still names A.
	if h := e.holder(t, lead); !h.Known || h.UserID != uidA {
		t.Fatalf("holder: %+v", h)
	}
}

// Test G — identity equality never infers privilege.
func TestRME04ONoPrivilegeInference(t *testing.T) {
	e := setupRME04O(t)
	ctx := context.Background()
	uidA := e.uid(t, oSellerA)
	// The imported legacy label is a SHADOW grant of the seller role.
	if e.count(t, `SELECT count(*) FROM identity.role_grant WHERE user_id = $1 AND status = 'SHADOW' AND role_key = 'seller'
		AND legacy_role_value = 'sn_seller'`, uidA) != 1 || e.count(t, `SELECT count(*) FROM identity.role_grant WHERE status <> 'SHADOW'`) != 0 {
		t.Fatal("seller label must exist only as a SHADOW grant")
	}
	e.provision(t, uidA, "rm-e04o.seller.a")
	s, _ := e.subject(t, e.httpLogin(t, "rm-e04o.seller.a"))
	before := s
	if h := e.holder(t, e.caseID(t, oLead, "701")); !h.Known || h.UserID != s.UserID {
		t.Fatal("precondition: identity equality holds")
	}
	if !reflect.DeepEqual(before, s) || len(s.Grants) != 0 {
		t.Fatal("a matching holder must not change the subject or add grants")
	}
	for c, d := range authorization.EvaluateAll(authorization.Runtime, s) {
		if d.Allowed {
			t.Fatalf("runtime capability %s allowed for an authenticated seller", c)
		}
	}
	// Narrower (unit-level, labelled): even counting the SHADOW grant, the Seller role bundle grants nothing.
	grants, err := e.ident.Grants(ctx)
	if err != nil {
		t.Fatal(err)
	}
	shadowSubject := authorization.Subject{PrincipalID: uidA, UserID: uidA}
	for _, g := range grants {
		if g.UserID == uidA {
			shadowSubject.Grants = append(shadowSubject.Grants, g)
		}
	}
	for c, d := range authorization.EvaluateAll(authorization.Shadow, shadowSubject) {
		if d.Allowed {
			t.Fatalf("shadow evaluation allowed %s for the seller bundle", c)
		}
	}
	if r, ok := identityapi.LookupRole(identityapi.RoleSeller); !ok || len(r.Capabilities) != 0 {
		t.Fatalf("Seller capability bundle changed: %v", r.Capabilities)
	}
	// RM-E04C candidate capabilities stay unregistered (commission.own.read is a pre-existing, separately
	// registered capability outside every role bundle; it is not a Group B candidate).
	for _, c := range []identityapi.Capability{"cases.own.read", "billing.own.read", "payments.own.read"} {
		if identityapi.KnownCapability(c) {
			t.Fatalf("%s registered", c)
		}
	}
	// No ACTIVE grant can be stored.
	gid, _ := ids.NewUUID()
	_, err = e.admin.Exec(ctx, `INSERT INTO identity.role_grant (grant_id, user_id, role_key, legacy_role_value, mapping_state, basis, status,
		evidence_spec, evidence_key, first_batch_id, source_batch_id, imported_at, refreshed_at, presence, version)
		SELECT $1, user_id, role_key, legacy_role_value, mapping_state, basis, 'ACTIVE', evidence_spec, '999999', first_batch_id, source_batch_id,
		imported_at, refreshed_at, presence, 1 FROM identity.role_grant WHERE user_id = $2 LIMIT 1`, gid, uidA)
	if got := pgCode(err); got != "23514/role_grant_shadow_only" {
		t.Fatalf("ACTIVE grant insert: %s", got)
	}
	// Group B gate unchanged and closed.
	raw, err := os.ReadFile(oGatePath)
	if err != nil {
		t.Fatal(err)
	}
	var gate struct {
		R1   string   `json:"m2_record_1_id"`
		R2   string   `json:"m2_record_2_id"`
		Caps []string `json:"approved_seller_capabilities"`
		Ops  []string `json:"approved_me_operations"`
	}
	if err := json.Unmarshal(raw, &gate); err != nil || gate.R1 != "" || gate.R2 != "" || len(gate.Caps) != 0 || len(gate.Ops) != 0 {
		t.Fatalf("Group B gate must stay closed: %+v %v", gate, err)
	}
}
