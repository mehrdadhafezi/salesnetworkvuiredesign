package compat

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	lwb "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/writeback"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
)

// All keys, tokens and data are SYNTHETIC. The counterpart is an
// in-process TLS server implementing the PROPOSED contract; nothing leaves
// the test process.

var (
	clientKey = []byte(strings.Repeat("c", 32))
	serverKey = []byte(strings.Repeat("s", 32))
	token     = "0f9c2a4e5b6d47e8a1b2c3d4e5f60718"
)

type memRecorder struct {
	mu   sync.Mutex
	got  []Exchange
	fail bool
}

func (m *memRecorder) Record(_ context.Context, e Exchange) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail {
		return errors.New("evidence store down")
	}
	m.got = append(m.got, e)
	return nil
}

func (m *memRecorder) last() Exchange {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.got[len(m.got)-1]
}

// fake is the synthetic counterpart; reply decides each answer.
type fake struct {
	mu      sync.Mutex
	reply   func(req request) (status int, body string, sign bool)
	applied map[string]string // request token → legacy ref (idempotent)
	calls   int
	lastReq request
	sigOK   bool
}

func (f *fake) handler(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var req request
		dec := json.NewDecoder(strings.NewReader(string(raw)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			t.Errorf("client sent a non-canonical request: %v", err)
		}
		f.mu.Lock()
		f.calls++
		f.lastReq = req
		f.sigOK = r.Header.Get(HeaderSignature) == mac(clientKey, raw) && r.Header.Get(HeaderKeyID) == "syn-key-01"
		reply := f.reply
		f.mu.Unlock()
		status, body, sign := reply(req)
		if sign {
			w.Header().Set(HeaderSignature, mac(serverKey, []byte(body)))
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	})
}

func envelope(req request, outcome string, extra map[string]any) string {
	m := map[string]any{"contract": ContractVersion, "operation": req.Operation, "request_token": req.RequestToken, "outcome": outcome,
		"legacy_ref": "", "reason": "", "matches": 0, "state": map[string]string{}}
	for k, v := range extra {
		m[k] = v
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func setup(t *testing.T, reply func(request) (int, string, bool)) (*Client, *fake, *memRecorder) {
	t.Helper()
	rec := &memRecorder{}
	c, f := setupWith(t, reply, rec, 0)
	return c, f, rec
}

// setupWith builds a client over any recorder with a record deadline.
func setupWith(t *testing.T, reply func(request) (int, string, bool), rec Recorder, recordTO time.Duration) (*Client, *fake) {
	t.Helper()
	f := &fake{reply: reply, applied: map[string]string{}}
	srv := httptest.NewTLSServer(f.handler(t))
	t.Cleanup(srv.Close)
	creds, err := NewHMACCredentials("syn-key-01", clientKey, serverKey)
	if err != nil {
		t.Fatal(err)
	}
	c, err := newClient(Config{Endpoint: srv.URL + "/sn-crm/compat", Timeout: 300 * time.Millisecond, RecordTimeout: recordTO, Credentials: creds,
		Recorder: rec, Clock: timeutil.FixedClock{At: mustInstant()}, Transport: srv.Client().Transport})
	if err != nil {
		t.Fatal(err)
	}
	return c, f
}

func mustInstant() timeutil.Instant {
	i, _ := timeutil.ParseRFC3339("2026-11-08T09:00:00Z")
	return i
}

func lead() lwb.LeadOutcome {
	return lwb.LeadOutcome{LeadID: "4101", StatusLabel: "جواب نداده", RequestToken: token, ActorLegacyUserID: "6"}
}

func isUnknown(err error) bool { return errors.Is(err, ErrOutcomeUnknown) }

// No contract is approved: the client cannot be built outside tests,
// whatever the configuration.
func TestNotApprovedAndConfig(t *testing.T) {
	creds, _ := NewHMACCredentials("syn-key-01", clientKey, serverKey)
	if _, err := New(Config{Endpoint: "https://legacy.example.test/x", Credentials: creds, Recorder: &memRecorder{},
		Clock: timeutil.FixedClock{At: mustInstant()}}, Approval{ContractID: ContractVersion}); !errors.Is(err, ErrNotApproved) {
		t.Fatalf("an unapproved contract must never activate: %v", err)
	}
	for _, ep := range []string{"http://legacy.example.test/x", "https://user:pw@legacy.example.test/x", "https://legacy.example.test/x?a=1", "ftp://x", ""} {
		if _, err := newClient(Config{Endpoint: ep, Credentials: creds, Recorder: &memRecorder{}, Clock: timeutil.FixedClock{At: mustInstant()}}); !errors.Is(err, ErrConfig) {
			t.Fatalf("endpoint %q accepted", ep)
		}
	}
	// 21: missing or malformed credentials.
	for _, bad := range []func() (*HMACCredentials, error){
		func() (*HMACCredentials, error) { return NewHMACCredentials("", clientKey, serverKey) },
		func() (*HMACCredentials, error) { return NewHMACCredentials("syn-key-01", []byte("short"), serverKey) },
		func() (*HMACCredentials, error) { return NewHMACCredentials("syn-key-01", clientKey, clientKey) },
		func() (*HMACCredentials, error) { return NewHMACCredentials("bad key!", clientKey, serverKey) },
	} {
		if c, err := bad(); err == nil || c != nil {
			t.Fatal("malformed credentials accepted")
		}
	}
	if _, err := newClient(Config{Endpoint: "https://legacy.example.test/x", Recorder: &memRecorder{}, Clock: timeutil.FixedClock{At: mustInstant()}}); !errors.Is(err, ErrConfig) {
		t.Fatal("missing credentials accepted")
	}
	if s := creds.String(); strings.Contains(s, string(clientKey)) || strings.Contains(s, string(serverKey)) {
		t.Fatal("credentials must never print key material")
	}
}

// A valid signed acknowledgement is the only success; the request carries
// the contract, token, correlation id and expected state.
func TestAppliedAndRejected(t *testing.T) {
	c, f, rec := setup(t, func(req request) (int, string, bool) {
		if req.Operation == "hr_request.decide" {
			return 200, envelope(req, "REJECTED", map[string]any{"reason": "STATE_CONFLICT"}), true
		}
		return 200, envelope(req, "APPLIED", map[string]any{"legacy_ref": "77001"}), true
	})
	ctx := lwb.WithCorrelation(context.Background(), "corr-r2-0001")
	res, err := c.ApplyLeadOutcome(ctx, lead())
	if err != nil || res.Outcome != result.Success || res.Ref != "77001" {
		t.Fatalf("applied: %+v %v", res, err)
	}
	if !f.sigOK || f.lastReq.Contract != ContractVersion || f.lastReq.RequestToken != token || f.lastReq.CorrelationID != "corr-r2-0001" {
		t.Fatalf("request envelope: %+v sig=%v", f.lastReq, f.sigOK)
	}
	if e := rec.last(); e.Verdict != VerdictValid || e.Outcome != "APPLIED" || e.HTTPStatus != 200 || e.ResponseSHA256 == nil || e.CorrelationID != "corr-r2-0001" {
		t.Fatalf("evidence: %+v", e)
	}
	// 11: a business rejection carries a contract reason; expected state sent.
	res, err = c.UpdateHRRequest(ctx, lwb.HRDecide{LegacyRequestID: "9001", Decision: "APPROVE", FromStatus: "pending_manager", ToStatus: "pending_hr",
		ReconcileKey: "1d3b7e0c-8a0e-4f3b-9b51-6b8f8e6c2a11"})
	if err != nil || res.Outcome != result.Conflict || res.Reason != "STATE_CONFLICT" {
		t.Fatalf("rejected: %+v %v", res, err)
	}
	if f.lastReq.Expected["status"] != "pending_manager" || f.lastReq.Expected["legacy_request_id"] != "9001" {
		t.Fatalf("expected source state must be sent: %+v", f.lastReq.Expected)
	}
}

// 9 / 10 / HTTP 200 without proof: every unproven answer is UNKNOWN (an
// error), recorded with its verdict — never success, never rejection.
func TestUnprovenAnswersAreUnknown(t *testing.T) {
	cases := []struct {
		name    string
		verdict Verdict
		reply   func(request) (int, string, bool)
	}{
		{"timeout", VerdictTimeout, func(req request) (int, string, bool) {
			time.Sleep(600 * time.Millisecond)
			return 200, envelope(req, "APPLIED", map[string]any{"legacy_ref": "1"}), true
		}},
		{"http 500 with a body", VerdictHTTPStatus, func(req request) (int, string, bool) {
			return 500, envelope(req, "APPLIED", map[string]any{"legacy_ref": "1"}), true
		}},
		{"http 409 rejection body", VerdictHTTPStatus, func(req request) (int, string, bool) {
			return 409, envelope(req, "REJECTED", map[string]any{"reason": "STATE_CONFLICT"}), true
		}},
		{"unsigned 200", VerdictBadSignature, func(req request) (int, string, bool) {
			return 200, envelope(req, "APPLIED", map[string]any{"legacy_ref": "1"}), false
		}},
		{"plain OK", VerdictBadSignature, func(request) (int, string, bool) { return 200, "OK", false }},
		{"unknown field", VerdictSchema, func(req request) (int, string, bool) {
			return 200, strings.TrimSuffix(envelope(req, "APPLIED", map[string]any{"legacy_ref": "1"}), "}") + `,"debug":"x"}`, true
		}},
		{"trailing data", VerdictSchema, func(req request) (int, string, bool) {
			return 200, envelope(req, "APPLIED", map[string]any{"legacy_ref": "1"}) + `{}`, true
		}},
		{"applied without ref", VerdictSchema, func(req request) (int, string, bool) { return 200, envelope(req, "APPLIED", nil), true }},
		{"unknown outcome", VerdictSchema, func(req request) (int, string, bool) { return 200, envelope(req, "SUCCESS", nil), true }},
		{"unknown reason", VerdictSchema, func(req request) (int, string, bool) {
			return 200, envelope(req, "REJECTED", map[string]any{"reason": "SOMETHING"}), true
		}},
		{"other token", VerdictMismatch, func(req request) (int, string, bool) {
			req.RequestToken = "ffffffffffffffffffffffffffffffff"
			return 200, envelope(req, "APPLIED", map[string]any{"legacy_ref": "1"}), true
		}},
		{"other contract", VerdictMismatch, func(req request) (int, string, bool) {
			b := strings.Replace(envelope(req, "APPLIED", map[string]any{"legacy_ref": "1"}), ContractVersion, "sn-crm-writeback-compat/9", 1)
			return 200, b, true
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _, rec := setup(t, tc.reply)
			res, err := c.ApplyLeadOutcome(context.Background(), lead())
			if !isUnknown(err) || res.Outcome != result.Status(0) {
				t.Fatalf("must be UNKNOWN: %+v %v", res, err)
			}
			if e := rec.last(); e.Verdict != tc.verdict || e.Outcome != "" {
				t.Fatalf("evidence verdict: %+v", e)
			}
		})
	}
}

// 12: network recovery: a dropped exchange is UNKNOWN; the next exchange
// (the worker observes first) succeeds and is idempotent on the token.
func TestNetworkRecoveryAndIdempotentToken(t *testing.T) {
	var mu sync.Mutex
	drop := true
	var c *Client
	var f *fake
	c, f, rec := setup(t, func(req request) (int, string, bool) {
		mu.Lock()
		defer mu.Unlock()
		if req.Operation == "lead_outcome.observe" {
			ref, ok := f.applied[req.RequestToken]
			if !ok {
				return 200, envelope(req, "OBSERVED", map[string]any{"matches": 0}), true
			}
			return 200, envelope(req, "OBSERVED", map[string]any{"matches": 1, "legacy_ref": ref, "state": map[string]string{"status_label": "جواب نداده"}}), true
		}
		f.applied[req.RequestToken] = "88001" // legacy committed ...
		if drop {
			drop = false
			mu.Unlock()
			time.Sleep(600 * time.Millisecond) // ... but the answer is lost
			mu.Lock()
		}
		return 200, envelope(req, "APPLIED", map[string]any{"legacy_ref": f.applied[req.RequestToken]}), true
	})
	if _, err := c.ApplyLeadOutcome(context.Background(), lead()); !isUnknown(err) {
		t.Fatalf("lost answer must be UNKNOWN: %v", err)
	}
	obs, err := c.ObserveLeadOutcome(context.Background(), "4101", token)
	if err != nil || obs.Events != 1 || obs.EventRef != "88001" || obs.StatusLabel != "جواب نداده" {
		t.Fatalf("observation after recovery: %+v %v", obs, err)
	}
	if f.calls != 2 || len(rec.got) != 2 {
		t.Fatalf("one apply, one observe: calls=%d evidence=%d", f.calls, len(rec.got))
	}
}

// Evidence first: if the exchange cannot be recorded, the outcome is UNKNOWN
// even for a valid acknowledgement.
func TestEvidenceRequired(t *testing.T) {
	c, _, rec := setup(t, func(req request) (int, string, bool) {
		return 200, envelope(req, "APPLIED", map[string]any{"legacy_ref": "1"}), true
	})
	rec.fail = true
	if _, err := c.ApplyLeadOutcome(context.Background(), lead()); !isUnknown(err) {
		t.Fatalf("an unrecorded exchange must be UNKNOWN: %v", err)
	}
}
