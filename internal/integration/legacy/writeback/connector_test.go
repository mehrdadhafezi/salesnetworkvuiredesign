package writeback_test

import (
	"context"
	"errors"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/legacytest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/shadow"
	lwb "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/writeback"
	wb "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/writeback"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

// users is a synthetic identity map: target id ↔ legacy key.
type users map[string]string

func (u users) UserKeys(_ context.Context, _ string, ids []string) (map[string]string, error) {
	out := map[string]string{}
	for _, id := range ids {
		if k, ok := u[id]; ok {
			out[id] = k
		}
	}
	return out, nil
}

func (u users) ResolveUsers(_ context.Context, _ string, keys []string) (map[string]string, error) {
	out := map[string]string{}
	for id, k := range u {
		for _, want := range keys {
			if k == want {
				out[k] = id
			}
		}
	}
	return out, nil
}

const (
	seller   = "00000000-0000-4000-8000-000000000006"
	other    = "00000000-0000-4000-8000-000000000007"
	senior   = "00000000-0000-4000-8000-000000000003"
	manager  = "00000000-0000-4000-8000-000000000002"
	cmdID    = "00000000-0000-5000-8000-0000000000c1"
	cmdID2   = "00000000-0000-5000-8000-0000000000c2"
	hrCmd    = "00000000-0000-5000-8000-0000000000d1"
	hrDecide = "00000000-0000-5000-8000-0000000000d2"
)

var ids = users{seller: "6", other: "7", senior: "3", manager: "2"}

func lead(cmd, outcome string) wb.Intent {
	return wb.Intent{CommandID: cmd, CommandType: wb.TypeContactOutcome, SubjectKind: "case",
		Fields: map[string]string{"source_system": shadow.SourceSystem, "source_spec": shadow.SpecLead, "source_key": "101", "outcome": outcome,
			"actor_user_id": seller}}
}

func setup(t *testing.T) (*legacytest.SyntheticLegacy, *lwb.Connector) {
	t.Helper()
	l := legacytest.NewSyntheticLegacy()
	l.Leads["101"] = &legacytest.SyntheticLead{SellerLegacyUserID: "6"}
	c, err := lwb.NewConnector(l, ids)
	if err != nil {
		t.Fatal(err)
	}
	return l, c
}

func TestMappingAndStableToken(t *testing.T) {
	if lwb.RequestToken(cmdID) != "000000000000500080000000000000c1" || lwb.RequestToken(cmdID) != lwb.RequestToken(cmdID) {
		t.Fatal("the legacy request token is the command id (stable across retries)")
	}
	l, c := setup(t)
	ctx := context.Background()
	in := lead(cmdID, "CALLBACK")
	in.Fields["callback_at"] = "2026-11-03T10:30:00Z"
	a, err := c.Apply(ctx, in)
	if err != nil || a.Outcome != result.Success {
		t.Fatalf("apply: %+v %v", a, err)
	}
	if got := l.Lead("101"); got.StatusLabel != "تماس مجدد" || got.CallbackJalali != "1405/08/12 14:00" {
		t.Fatalf("legacy mapping: %+v", got)
	}
	o, err := c.Observe(ctx, in)
	if err != nil || o.Observation != wb.Committed || o.State["outcome"] != "CALLBACK" || o.State["callback_at"] != "2026-11-03T10:30:00Z" {
		t.Fatalf("observe maps back to the neutral expected keys: %+v %v", o, err)
	}
	if wb.ParityOf(wb.Expected(in), o) != "MATCH" {
		t.Fatal("parity")
	}
}

func TestKnownRejectionsAreNotUnknown(t *testing.T) {
	l, c := setup(t)
	ctx := context.Background()
	// Not the lead owner in legacy: UNAUTHORIZED, nothing committed.
	in := lead(cmdID, "NO_ANSWER")
	in.Fields["actor_user_id"] = other
	if a, err := c.Apply(ctx, in); err != nil || a.Outcome != result.Unauthorized || l.FlowEvents() != 0 {
		t.Fatalf("owner rule: %+v %v", a, err)
	}
	// Archived in legacy: CONFLICT.
	l.Leads["101"].Archived = true
	if a, _ := c.Apply(ctx, lead(cmdID2, "NO_ANSWER")); a.Outcome != result.Conflict {
		t.Fatalf("archived: %+v", a)
	}
	// Unmappable actor: BLOCKED before anything is sent.
	in = lead(cmdID, "NO_ANSWER")
	in.Fields["actor_user_id"] = "00000000-0000-4000-8000-0000000000ff"
	calls := l.Calls["lead"]
	if a, _ := c.Apply(ctx, in); a.Outcome != result.Blocked || l.Calls["lead"] != calls {
		t.Fatalf("unmapped actor is never written as a service user: %+v", a)
	}
	// Not a legacy lead: BLOCKED.
	in.Fields["source_spec"] = shadow.SpecDistItem
	if a, _ := c.Apply(ctx, in); a.Outcome != result.Blocked {
		t.Fatalf("non-lead subject: %+v", a)
	}
}

func TestTimeoutReconcileAndDuplicateProtection(t *testing.T) {
	l, c := setup(t)
	ctx := context.Background()
	// Timeout after commit: an error (outcome unknown), the effect exists.
	l.Inject("lead", legacytest.TimeoutAfterCommit)
	if _, err := c.Apply(ctx, lead(cmdID, "NO_ANSWER")); !errors.Is(err, legacytest.ErrTransport) {
		t.Fatalf("timeout: %v", err)
	}
	if o, _ := c.Observe(ctx, lead(cmdID, "NO_ANSWER")); o.Observation != wb.Committed {
		t.Fatalf("reconcile committed: %+v", o)
	}
	// Same intent applied again (e.g. a retry): the legacy request token
	// makes it idempotent — one effect.
	if a, _ := c.Apply(ctx, lead(cmdID, "NO_ANSWER")); a.Outcome != result.Success || l.FlowEvents() != 1 {
		t.Fatalf("duplicate protection: %+v events=%d", a, l.FlowEvents())
	}
	// Timeout before commit: observation proves absence.
	l.Inject("lead", legacytest.TimeoutBeforeCommit)
	if _, err := c.Apply(ctx, lead(cmdID2, "NOT_PURCHASED")); err == nil {
		t.Fatal("timeout before commit must be an error")
	}
	if o, _ := c.Observe(ctx, lead(cmdID2, "NOT_PURCHASED")); o.Observation != wb.Absent {
		t.Fatalf("reconcile absent: %+v", o)
	}
	// Unreadable legacy: an error, never ABSENT.
	l.Inject("lead_observe", legacytest.ObserveDown)
	if _, err := c.Observe(ctx, lead(cmdID2, "NO_ANSWER")); err == nil {
		t.Fatal("unreadable legacy must not look absent")
	}
}

func TestHRMappingAndAmbiguity(t *testing.T) {
	l, c := setup(t)
	ctx := context.Background()
	create := wb.Intent{CommandID: hrCmd, CommandType: wb.TypeHRCreate, SubjectKind: "hr_request",
		Fields: map[string]string{"request_type": "terminate_seller", "subject_user_id": other, "requested_by_user_id": senior,
			"current_reviewer_user_id": manager, "status": "pending_review", "reason": "synthetic"}}
	a, err := c.Apply(ctx, create)
	if err != nil || a.Outcome != result.Success || a.LegacyRef == "" {
		t.Fatalf("hr create: %+v %v", a, err)
	}
	if again, _ := c.Apply(ctx, create); again.LegacyRef != a.LegacyRef || len(l.Requests) != 1 {
		t.Fatalf("hr create is idempotent by its reconciliation key: %+v", again)
	}
	if o, _ := c.Observe(ctx, create); o.Observation != wb.Committed || wb.ParityOf(wb.Expected(create), o) != "MATCH" {
		t.Fatalf("hr observe: %+v", o)
	}
	decide := wb.Intent{CommandID: hrDecide, CommandType: wb.TypeHRDecide, SubjectKind: "hr_request", DependsOnLegacyRef: a.LegacyRef,
		Fields: map[string]string{"decision": "reject", "from_status": "pending_review", "to_status": "rejected", "actor_user_id": senior}}
	if r, _ := c.Apply(ctx, decide); r.Outcome != result.Unauthorized {
		t.Fatalf("legacy reviewer rule: only the current reviewer decides: %+v", r)
	}
	decide.Fields["actor_user_id"] = manager
	if r, _ := c.Apply(ctx, decide); r.Outcome != result.Success {
		t.Fatalf("decision: %+v", r)
	}
	// A duplicated legacy result for one intent is AMBIGUOUS, never success.
	l.Duplicate(hrCmd)
	if o, _ := c.Observe(ctx, create); o.Observation != wb.Ambiguous {
		t.Fatalf("ambiguous: %+v", o)
	}
	// A decision whose request never reached legacy is refused before any call.
	decide.DependsOnLegacyRef = ""
	if r, _ := c.Apply(ctx, decide); r.Outcome != result.Blocked {
		t.Fatalf("no legacy request: %+v", r)
	}
}
