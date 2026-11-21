package query_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
	paymentsmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/store"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pagination"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

// RM-P03: an attempt's state history pages in append order (recorded_at,
// then history_id). history_id is a random UUID, so ordering by it alone
// returned rows in an arbitrary order; a timeline must never show that as a
// chronology. SYNTHETIC data, written through the existing import upsert.
func TestAttemptHistoryPagesInAppendOrder(t *testing.T) {
	e := pgtest.NewWith(t, paymentsmigrations.Source())
	ctx := context.Background()
	r := e.Runner(t, payments.Module)

	states := []string{"pending", "approved", "rejected", "approved", "cancelled", "pending"}
	base := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	var attemptID string
	for i, st := range states {
		sum := sha256.Sum256([]byte{byte(i)})
		raw, s := "600", st
		rec := api.AttemptRecord{
			Source:  api.SourceRef{System: "synthetic", Spec: "synthetic.rm_p03.payment.v1", Key: "1", BatchID: batchID(i), RowSHA256: sum[:]},
			Channel: api.ChannelGateway, Amount: api.RawMoney{Raw: &raw, Unit: "IRT"}, StateNamespace: "synthetic.payment.status", StateRaw: &s,
			ApprovalProvenance: api.ProvUnknown,
		}
		err := r.InTx(ctx, db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
			id, _, _, err := store.UpsertAttempt(ctx, tx, nil, rec, base.Add(time.Duration(i)*time.Hour))
			attemptID = id
			return err
		})
		if err != nil {
			t.Fatalf("synthetic upsert %d: %v", i, err)
		}
	}

	svc := query.New(r)
	var got []string
	var cursor *string
	one := 2
	for pages := 0; ; pages++ {
		if pages > 10 {
			t.Fatal("pagination did not terminate")
		}
		p, err := svc.History(ctx, attemptID, cursor, &one)
		if err != nil {
			t.Fatalf("history page %d: %v", pages, err)
		}
		for _, h := range p.Entries {
			got = append(got, *h.OldStateRaw+"→"+*h.NewRaw+"@"+h.RecordedAt.UTC().Format(time.RFC3339))
		}
		if p.NextCursor == nil {
			break
		}
		cursor = p.NextCursor
	}
	want := []string{
		"pending→approved@2026-11-01T10:00:00Z", "approved→rejected@2026-11-01T11:00:00Z", "rejected→approved@2026-11-01T12:00:00Z",
		"approved→cancelled@2026-11-01T13:00:00Z", "cancelled→pending@2026-11-01T14:00:00Z",
	}
	if len(got) != len(want) {
		t.Fatalf("history = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("history[%d] = %s, want %s (all: %v)", i, got[i], want[i], got)
		}
	}

	// A cursor issued for another keyset (the old id-only form) is refused, never misread.
	old, err := pagination.Encode(pagination.Cursor{Sort: "id", ID: "00000000-0000-4000-8000-000000000001"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.History(ctx, attemptID, &old, &one); !errors.Is(err, query.ErrInvalidQuery) {
		t.Fatalf("foreign cursor: %v", err)
	}
	bad, _ := pagination.Encode(pagination.Cursor{Sort: "recorded_at", Value: "yesterday", ID: "00000000-0000-4000-8000-000000000001"})
	if _, err := svc.History(ctx, attemptID, &bad, &one); !errors.Is(err, query.ErrInvalidQuery) {
		t.Fatalf("malformed cursor: %v", err)
	}

	// The read path is READ ONLY: a writable transaction is refused before any SQL.
	err = r.InTx(ctx, db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		_, err := store.History(ctx, tx, attemptID, nil, 10)
		return err
	})
	if !errors.Is(err, store.ErrNotReadOnly) {
		t.Fatalf("writable tx: %v", err)
	}
}

func batchID(i int) string {
	return "b0000000-0000-4000-8000-0000000000" + string(rune('a'+i)) + "0"
}
