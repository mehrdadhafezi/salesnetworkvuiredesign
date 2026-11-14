package biawin_test

// Regression for the work-case intake race found in CI on PR #58
// (TestR74FulfillmentConcurrency E6, "duplicate key value violates unique
// constraint work_case_pkey"). case_id is CaseID(item_id), so one item
// conflicts on BOTH unique keys of fulfillment.work_case. With the arbiter
// limited to item_id, a concurrent redelivery could pass the item_id
// pre-check, then hit the non-arbiter primary key and fail with 23505
// instead of answering EXISTING_CASE. The race is timing-dependent, so this
// repeats the E6 shape many times; every intake must succeed and each item
// must get exactly one case, one CREATED event, one outbox event and one
// intake row per message.

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"

	biawinapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/work"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/outbox"
)

func TestWorkIntakeConcurrentRedeliveryNeverErrors(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	e.seed(t, "r7-intake", 1, map[string]string{"r7-intake.svc-01": uid(t)}, map[string]string{"r7-intake.svc-01": uid(t)}, nil)
	const rounds, n = 60, 8
	for r := 0; r < rounds; r++ {
		item := uid(t)
		payload, _ := json.Marshal(biawinapi.SubscriptionItemActivated{ActivationID: uid(t), SubscriptionID: uid(t), ItemID: item, Seq: 1,
			ServiceKey: "r7-intake.svc-01", ServiceVersion: 1, CustomerID: uid(t), RolloutMode: "SHADOW_COMPARE", CorrelationID: "corr-intake-race",
			CausationID: "r7-intake", OccurredAt: e.clock.Now().Time()})
		msgs := [2]string{uid(t), uid(t)} // one message redelivered + a second message for the same item
		outs := make([]work.IntakeOutcome, n)
		errs := make([]error, n)
		start := make(chan struct{})
		var wg sync.WaitGroup
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				outs[i], errs[i] = e.work.OnItemActivated(ctx, outbox.Record{Event: outbox.Event{ID: msgs[i%2], Type: biawinapi.EventItemActivated,
					Version: 1, Payload: payload, OccurredAt: e.clock.Now()}})
			}(i)
		}
		close(start)
		wg.Wait()
		created := 0
		for i, err := range errs {
			if err != nil {
				t.Fatalf("round %d intake %d: %v", r, i, err)
			}
			if outs[i].Decision == "CASE_CREATED" && !outs[i].Duplicate {
				created++
			}
		}
		if created != 1 ||
			e.count(t, `SELECT count(*) FROM fulfillment.work_case WHERE item_id = $1`, item) != 1 ||
			e.count(t, `SELECT count(*) FROM fulfillment.work_event WHERE case_id = $1 AND kind = 'CREATED'`, work.CaseID(item)) != 1 ||
			e.count(t, `SELECT count(*) FROM fulfillment.outbox_event WHERE event_type = $1 AND payload->>'item_id' = $2`, api.EventWorkCaseCreated, item) != 1 ||
			e.count(t, `SELECT count(*) FROM fulfillment.work_intake WHERE item_id = $1`, item) != 2 {
			t.Fatalf("round %d: created=%d (want exactly one case, event, outbox event and two intake rows)", r, created)
		}
	}
}

// intakeOf builds one SubscriptionItemActivated message for item.
func intakeOf(t *testing.T, e *env, item, service, msg string) outbox.Record {
	t.Helper()
	payload, _ := json.Marshal(biawinapi.SubscriptionItemActivated{ActivationID: uid(t), SubscriptionID: uid(t), ItemID: item, Seq: 1,
		ServiceKey: service, ServiceVersion: 1, CustomerID: uid(t), RolloutMode: "SHADOW_COMPARE", CorrelationID: "corr-intake-integrity",
		CausationID: "r7-intake", OccurredAt: e.clock.Now().Time()})
	return outbox.Record{Event: outbox.Event{ID: msg, Type: biawinapi.EventItemActivated, Version: 1, Payload: payload, OccurredAt: e.clock.Now()}}
}

// A stored row that matches the new item on only ONE unique key is an
// integrity fault, never an existing case: the intake fails closed, writes
// nothing (no case, intake row, event or outbox event) and leaves the
// message unconsumed. A sequential redelivery of a processed message is a
// replay with no new row.
func TestWorkIntakeIdentityConflictFailsClosed(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	const svc = "r7-intake-integrity.svc-01"
	e.seed(t, "r7-intake-integrity", 1, map[string]string{svc: uid(t)}, map[string]string{svc: uid(t)}, nil)

	// A legitimate case to copy a valid row from, then its replay.
	base := uid(t)
	msg := uid(t)
	first, err := e.work.OnItemActivated(ctx, intakeOf(t, e, base, svc, msg))
	if err != nil || first.Decision != "CASE_CREATED" {
		t.Fatalf("base intake: %+v %v", first, err)
	}
	again, err := e.work.OnItemActivated(ctx, intakeOf(t, e, base, svc, msg))
	if err != nil || !again.Duplicate || e.count(t, `SELECT count(*) FROM fulfillment.work_intake WHERE item_id = $1`, base) != 1 {
		t.Fatalf("sequential redelivery must replay: %+v %v", again, err)
	}
	other := uid(t)
	rehome, err := e.work.OnItemActivated(ctx, intakeOf(t, e, base, svc, other))
	if err != nil || rehome.Decision != "EXISTING_CASE" || rehome.Duplicate || e.count(t, `SELECT count(*) FROM fulfillment.work_case WHERE item_id = $1`, base) != 1 {
		t.Fatalf("a second message for the same item answers EXISTING_CASE: %+v %v", rehome, err)
	}

	clone := `INSERT INTO fulfillment.work_case SELECT (jsonb_populate_record(NULL::fulfillment.work_case,
		to_jsonb(w) || jsonb_build_object('case_id', $1::text, 'item_id', $2::text))).* FROM fulfillment.work_case w WHERE w.item_id = $3`
	x, y := uid(t), uid(t)
	for _, c := range []struct {
		label, item, fixCase, fixItem string
	}{
		{"case_id taken by another item", x, work.CaseID(x), uid(t)},
		{"item_id stored under another case_id", y, uid(t), y},
	} {
		if _, err := e.admin(t).Exec(ctx, clone, c.fixCase, c.fixItem, base); err != nil {
			t.Fatalf("%s: fixture: %v", c.label, err)
		}
		before := e.intakeState(t, c.item)
		m := uid(t)
		out, err := e.work.OnItemActivated(ctx, intakeOf(t, e, c.item, svc, m))
		if !errors.Is(err, work.ErrCaseIdentityConflict) {
			t.Fatalf("%s: want ErrCaseIdentityConflict, got %+v %v", c.label, out, err)
		}
		if after := e.intakeState(t, c.item); !reflect.DeepEqual(before, after) {
			t.Fatalf("%s: a refused intake wrote something: %v -> %v", c.label, before, after)
		}
		if n := e.count(t, `SELECT count(*) FROM fulfillment.inbox_message WHERE message_id = $1 AND state = 'PROCESSED'`, m); n != 0 {
			t.Fatalf("%s: the message was consumed", c.label)
		}
	}
}

// intakeState counts every row an intake for item may write.
func (e *env) intakeState(t *testing.T, item string) []int {
	t.Helper()
	return []int{
		e.count(t, `SELECT count(*) FROM fulfillment.work_case`),
		e.count(t, `SELECT count(*) FROM fulfillment.work_intake WHERE item_id = $1`, item),
		e.count(t, `SELECT count(*) FROM fulfillment.work_event`),
		e.count(t, `SELECT count(*) FROM fulfillment.outbox_event`),
	}
}

// The bare ON CONFLICT is only safe while work_case has exactly two unique
// keys that both identify the item: case_id = CaseID(item_id) and item_id.
// A new unique or exclusion constraint must revisit the intake.
func TestWorkCaseUniqueKeysAreOnlyCaseAndItem(t *testing.T) {
	e := setup(t)
	rows, err := e.admin(t).Query(context.Background(), `SELECT pg_get_indexdef(i.indexrelid) FROM pg_index i
		WHERE i.indrelid = 'fulfillment.work_case'::regclass AND (i.indisunique OR i.indisexclusion) ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var defs []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			t.Fatal(err)
		}
		defs = append(defs, d)
	}
	want := []string{
		"CREATE UNIQUE INDEX work_case_item_id_key ON fulfillment.work_case USING btree (item_id)",
		"CREATE UNIQUE INDEX work_case_pkey ON fulfillment.work_case USING btree (case_id)",
	}
	if !reflect.DeepEqual(defs, want) {
		t.Fatalf("work_case unique / exclusion indexes changed; revisit ON CONFLICT in work.OnItemActivated: %v", defs)
	}
}
