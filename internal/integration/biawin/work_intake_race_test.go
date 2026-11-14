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
