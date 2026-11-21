package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases"
	casesmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/store"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

// RM-P01: the customer-filtered list runs only in a cases-owned READ ONLY
// transaction; a writable one is refused before any SQL.
func TestListCasesRefusesWritableTx(t *testing.T) {
	e := pgtest.New(t)
	ctx := context.Background()
	if _, err := migrations.Up(ctx, e.DSN, casesmigrations.Source()); err != nil {
		t.Fatal(err)
	}
	id := "a0000000-0000-4000-8000-00000000000a"
	f := store.CaseFilter{CustomerID: &id}
	r := e.Runner(t, cases.Module)
	err := r.InTx(ctx, db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		_, err := store.ListCases(ctx, tx, f, nil, 10)
		return err
	})
	if !errors.Is(err, store.ErrNotReadOnly) {
		t.Fatalf("writable tx: %v", err)
	}
	err = r.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
		rows, err := store.ListCases(ctx, tx, f, nil, 10)
		if err == nil && len(rows) != 0 {
			t.Fatalf("empty model returned %d rows", len(rows))
		}
		return err
	})
	if err != nil {
		t.Fatalf("read-only tx: %v", err)
	}
}
