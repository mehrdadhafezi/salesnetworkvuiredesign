package featureflags_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/featureflags"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

func ctx(t *testing.T) context.Context {
	t.Helper()
	c, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return c
}

var (
	t1, _ = timeutil.ParseRFC3339("2026-10-29T10:00:00Z")
	t2, _ = timeutil.ParseRFC3339("2026-10-29T11:00:00Z")
)

func TestKeyValidation(t *testing.T) {
	for _, k := range []string{"syn.flag", "abc", "strangler.sales_read-v1"} {
		if err := featureflags.ValidateKey(k); err != nil {
			t.Fatalf("%q rejected: %v", k, err)
		}
	}
	for _, k := range []string{"", "ab", "Upper", "1abc", "has space", strings.Repeat("a", 65)} {
		if err := featureflags.ValidateKey(k); !errors.Is(err, featureflags.ErrInvalidFlag) {
			t.Fatalf("%q accepted: %v", k, err)
		}
	}
}

func TestPersistenceAndDefaultOff(t *testing.T) {
	env := pgtest.New(t)
	flags := env.Runner(t, owner.FeatureFlags)
	desc := "synthetic flag"
	err := flags.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		if on, err := featureflags.Enabled(c, tx, "syn.flag"); err != nil || on {
			t.Errorf("absent flag must be off: %v, %v", on, err)
		}
		return featureflags.Set(c, tx, featureflags.Flag{Key: "syn.flag", Enabled: true, Description: &desc, UpdatedAt: t1})
	})
	if err != nil {
		t.Fatal(err)
	}
	err = flags.InTx(ctx(t), db.TxOptions{ReadOnly: true}, func(c context.Context, tx db.Tx) error {
		f, ok, err := featureflags.Get(c, tx, "syn.flag")
		if err != nil || !ok || !f.Enabled || *f.Description != desc || !f.UpdatedAt.Equal(t1) {
			t.Errorf("Get = %+v, %v, %v", f, ok, err)
		}
		if err := featureflags.Set(c, tx, featureflags.Flag{Key: "syn.flag", UpdatedAt: t2}); !errors.Is(err, featureflags.ErrReadOnly) {
			t.Errorf("read-only Set err = %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := flags.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		return featureflags.Set(c, tx, featureflags.Flag{Key: "syn.flag", Enabled: false, UpdatedAt: t2})
	}); err != nil {
		t.Fatal(err)
	}
	err = flags.InTx(ctx(t), db.TxOptions{ReadOnly: true}, func(c context.Context, tx db.Tx) error {
		f, ok, err := featureflags.Get(c, tx, "syn.flag")
		if err != nil || !ok || f.Enabled || f.Description != nil || !f.UpdatedAt.Equal(t2) {
			t.Errorf("after update = %+v, %v, %v", f, ok, err)
		}
		on, err := featureflags.Enabled(c, tx, "syn.flag")
		if err != nil || on {
			t.Errorf("Enabled = %v, %v", on, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRulesAndOwnership(t *testing.T) {
	env := pgtest.New(t)
	flags := env.Runner(t, owner.FeatureFlags)
	empty, long := "", strings.Repeat("d", 501)
	err := flags.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		for i, f := range []featureflags.Flag{
			{Key: "Bad", UpdatedAt: t1},
			{Key: "syn.flag", Description: &empty, UpdatedAt: t1},
			{Key: "syn.flag", Description: &long, UpdatedAt: t1},
			{Key: "syn.flag"},
		} {
			if err := featureflags.Set(c, tx, f); !errors.Is(err, featureflags.ErrInvalidFlag) {
				t.Errorf("flag %d accepted: %v", i, err)
			}
		}
		if _, _, err := featureflags.Get(c, tx, "Bad"); !errors.Is(err, featureflags.ErrInvalidFlag) {
			t.Errorf("Get bad key err = %v", err)
		}
		if _, err := featureflags.Enabled(c, tx, "Bad"); !errors.Is(err, featureflags.ErrInvalidFlag) {
			t.Errorf("Enabled bad key err = %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	err = env.Runner(t, pgtest.Alpha).InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		if err := featureflags.Set(c, tx, featureflags.Flag{Key: "syn.flag", UpdatedAt: t1}); !errors.Is(err, featureflags.ErrWrongOwner) {
			t.Errorf("foreign Set err = %v", err)
		}
		if _, _, err := featureflags.Get(c, tx, "syn.flag"); !errors.Is(err, featureflags.ErrWrongOwner) {
			t.Errorf("foreign Get err = %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
