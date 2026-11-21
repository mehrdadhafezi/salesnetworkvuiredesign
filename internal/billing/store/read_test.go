package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing"
	billingmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/store"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

// RM-P01: the customer/case-filtered invoice list runs only in a billing-owned READ ONLY
// transaction; a writable one is refused before any SQL.
func TestListInvoicesRefusesWritableTx(t *testing.T) {
	e := pgtest.New(t)
	ctx := context.Background()
	if _, err := migrations.Up(ctx, e.DSN, billingmigrations.Source()); err != nil {
		t.Fatal(err)
	}
	id := "a0000000-0000-4000-8000-00000000000a"
	c := "c0000000-0000-4000-8000-0000000000a1"
	f := store.InvoiceFilter{CustomerID: &id, CaseID: &c}
	r := e.Runner(t, billing.Module)
	err := r.InTx(ctx, db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		_, err := store.ListInvoices(ctx, tx, f, nil, 10)
		return err
	})
	if !errors.Is(err, store.ErrNotReadOnly) {
		t.Fatalf("writable tx: %v", err)
	}
	err = r.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
		rows, err := store.ListInvoices(ctx, tx, f, nil, 10)
		if err == nil && len(rows) != 0 {
			t.Fatalf("empty model returned %d rows", len(rows))
		}
		return err
	})
	if err != nil {
		t.Fatalf("read-only tx: %v", err)
	}
}
