package shadow_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/domain"
	commissionhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/http"
	commissionquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/query"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/featureflags"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

const commissionFlag = "commission.read_shadow"

func (e *commissionEnv) server(t *testing.T, subs authorization.SubjectResolver) http.Handler {
	t.Helper()
	flags := e.Runner(t, owner.FeatureFlags)
	guard, err := authorization.NewGuard(func(ctx context.Context) (bool, error) {
		var on bool
		err := flags.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
			var err error
			on, err = featureflags.Enabled(ctx, tx, commissionFlag)
			return err
		})
		return on, err
	}, subs)
	if err != nil {
		t.Fatal(err)
	}
	q, err := commissionquery.New(e.cr)
	if err != nil {
		t.Fatal(err)
	}
	at, _ := timeutil.ParseRFC3339("2026-11-03T10:00:00Z")
	cfg := apiserver.DefaultConfig()
	cfg.Environment, cfg.DevAuth = authstub.Development, true
	h, err := apiserver.NewHandlerWithSurfaces(cfg, timeutil.FixedClock{At: at}, slog.New(slog.NewTextHandler(io.Discard, nil)),
		apiserver.Surfaces{Commission: commissionhttp.New(guard, q)})
	if err != nil {
		t.Fatal(err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := r.Header.Get(actorHeader); a != "" {
			r = r.WithContext(context.WithValue(r.Context(), actorKey{}, a))
		}
		h.ServeHTTP(w, r)
	})
}

func (e *commissionEnv) setFlag(t *testing.T, on bool) {
	t.Helper()
	at, _ := timeutil.ParseRFC3339("2026-11-03T08:30:00Z")
	if err := e.Runner(t, owner.FeatureFlags).InTx(context.Background(), db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		return featureflags.Set(ctx, tx, featureflags.Flag{Key: commissionFlag, Enabled: on, UpdatedAt: at})
	}); err != nil {
		t.Fatal(err)
	}
}

func actorGet(t *testing.T, h http.Handler, target, actor string) resp {
	t.Helper()
	return actorDo(t, h, http.MethodGet, target, actor)
}

func actorDo(t *testing.T, h http.Handler, method, target, actor string) resp {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	req.Header.Set(authstub.Header, "dev-user-001")
	if actor != "" {
		req.Header.Set(actorHeader, actor)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	r := resp{code: rec.Code, body: rec.Body.String()}
	_ = json.Unmarshal(rec.Body.Bytes(), &r.json)
	return r
}

var commissionGets = []string{"/api/v1/commission/overview", "/api/v1/commission/runs", "/api/v1/commission/calculations", "/api/v1/commission/findings",
	"/api/v1/commission/policies", "/api/v1/commission/historical-runs"}

// 20 / 21 / 23: GET-only proof over HTTP against the real owner: flag,
// capabilities, stored results only, own scope, no approve / post route,
// and no GET writes anything.
func TestCommissionHTTPReadOnly(t *testing.T) {
	e := setupCommission(t)
	ctx := context.Background()
	b := e.loaded(t, "syn-commission-http", commissionSource(snapshot.Complete))
	src := e.staged(t, b)
	all := []identityapi.Capability{identityapi.CapCommissionShadowRead, identityapi.CapCommissionPolicyRead, identityapi.CapCommissionOwnRead}
	subs := testSubjects{users: map[string]string{"5": src.Users["5"], "6": src.Users["6"], "viewer": src.Users["4"]},
		caps: map[string][]identityapi.Capability{"5": all, "6": {identityapi.CapCommissionOwnRead}, "viewer": nil}}
	if subs.users["5"] == "" || subs.users["6"] == "" {
		t.Fatal("synthetic users 5 / 6 must resolve")
	}
	h := e.server(t, subs)

	// Flag off (default): BLOCKED / FEATURE_DISABLED, no data.
	for _, p := range append(commissionGets, "/api/v1/commission/me/earnings") {
		if r := actorGet(t, h, p, "5"); r.code != 200 || r.json["outcome"] != "BLOCKED" || r.json["items"] != nil || r.json["overview"] != nil {
			t.Fatalf("%s with the flag off: %d %s", p, r.code, r.body)
		}
	}
	e.setFlag(t, true)
	// No run yet: UNKNOWN, never an empty zero answer.
	if r := actorGet(t, h, "/api/v1/commission/calculations", "5"); r.json["outcome"] != "UNKNOWN" || r.json["items"] != nil {
		t.Fatalf("no run: %s", r.body)
	}
	if _, err := e.cmd.Run(ctx, src, domain.RoundingLegacyHalfAwayTwoDP); err != nil {
		t.Fatal(err)
	}
	before := map[string]int64{}
	for _, tb := range commissionTables {
		before[tb] = e.count(t, `SELECT count(*) FROM `+tb)
	}
	for _, p := range commissionGets {
		r := actorGet(t, h, p, "5")
		if r.code != 200 || (r.json["outcome"] != "PARTIAL" && r.json["outcome"] != "SUCCESS") ||
			r.json["notice"] != "SHADOW only: no commission was calculated, approved, posted or credited by this read; legacy remains the system of record" {
			t.Fatalf("%s: %d %s", p, r.code, r.body[:min(500, len(r.body))])
		}
	}
	ov := actorGet(t, h, "/api/v1/commission/overview", "5")
	o, _ := ov.json["overview"].(map[string]any)
	if o["posting_state"] != "BLOCKED_NOT_AUTHORIZED" || o["parity"] == nil || !strings.Contains(ov.body, `"OPD-09"`) || !strings.Contains(ov.body, `"D-36"`) {
		t.Fatalf("overview: %s", ov.body)
	}
	acts, _ := ov.json["allowed_actions"].(map[string]any)
	for k, v := range acts {
		if a, _ := v.(map[string]any); a["allowed"] == true && !strings.HasSuffix(k, ".read") && !strings.Contains(k, "_read") {
			t.Fatalf("a non-read action is offered: %s", k)
		}
		if strings.Contains(k, "commission") && (strings.Contains(k, "approve") || strings.Contains(k, "post") || strings.Contains(k, "credit")) {
			t.Fatalf("commission command action exists: %s", k)
		}
	}
	calcs := actorGet(t, h, "/api/v1/commission/calculations?parity_class=MATCH", "5")
	for _, it := range items(calcs) {
		if it["parity_class"] != "MATCH" {
			t.Fatalf("filter: %v", it)
		}
	}
	if len(items(calcs)) != 4 {
		t.Fatalf("MATCH rows: %d %s", len(items(calcs)), calcs.body)
	}
	if r := actorGet(t, h, "/api/v1/commission/calculations?parity_class=GUESS", "5"); r.code != 400 {
		t.Fatalf("invalid filter: %d", r.code)
	}
	if r := actorGet(t, h, "/api/v1/commission/calculations?phone=0900", "5"); r.code != 400 {
		t.Fatalf("unknown parameter: %d", r.code)
	}

	// Own scope: user 5 sees only its own credits; user 6 only its own.
	mine := actorGet(t, h, "/api/v1/commission/me/earnings", "5")
	if mine.code != 200 || len(items(mine)) == 0 {
		t.Fatalf("own earnings of 5: %s", mine.body)
	}
	for _, it := range items(mine) {
		if it["source_key"] == "105" || it["source_key"] == "101" {
			t.Fatalf("another recipient's credit leaked: %v", it)
		}
	}
	six := actorGet(t, h, "/api/v1/commission/me/earnings", "6")
	if len(items(six)) != 1 || items(six)[0]["source_key"] != "105" {
		t.Fatalf("own earnings of 6: %s", six.body)
	}
	// Own read does not grant the organisation reads (View != broader view).
	if r := actorGet(t, h, "/api/v1/commission/calculations", "6"); r.code != 403 {
		t.Fatalf("own-only caller read calculations: %d", r.code)
	}
	if r := actorGet(t, h, "/api/v1/commission/me/earnings", "viewer"); r.code != 403 {
		t.Fatalf("no capability: %d", r.code)
	}
	if r := actorGet(t, h, "/api/v1/commission/me/earnings", ""); r.code != 403 {
		t.Fatalf("unproven user must be denied own scope: %d", r.code)
	}

	// 23: no approve / post / credit / recalculate route; non-GET is refused.
	for _, p := range []string{"/api/v1/commission/runs/approve", "/api/v1/commission/post", "/api/v1/commission/calculate", "/api/v1/commission/credit"} {
		if r := actorDo(t, h, http.MethodPost, p, "5"); r.code != 404 && r.code != 405 {
			t.Fatalf("POST %s: %d", p, r.code)
		}
	}
	for _, p := range commissionGets {
		if r := actorDo(t, h, http.MethodPost, p, "5"); r.code != 405 {
			t.Fatalf("POST %s: %d", p, r.code)
		}
	}
	// 21: no GET wrote anything (no calculation, import or credit on read).
	for _, tb := range commissionTables {
		if e.count(t, `SELECT count(*) FROM `+tb) != before[tb] {
			t.Fatalf("%s changed on GET", tb)
		}
	}
}
