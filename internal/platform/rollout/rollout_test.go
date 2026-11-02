package rollout

import (
	"context"
	"errors"
	"testing"
)

func TestModes(t *testing.T) {
	ctx := context.Background()
	for _, c := range []struct {
		write, writeback bool
		want             Mode
	}{{false, false, Disabled}, {false, true, Disabled}, {true, false, Shadow}, {true, true, Controlled}} {
		flags := map[string]bool{"cases.nonfinancial_write": c.write, "cases.nonfinancial_writeback": c.writeback}
		src, err := FromFlags("cases.nonfinancial", func(_ context.Context, k string) (bool, error) { return flags[k], nil })
		if err != nil {
			t.Fatal(err)
		}
		if m, err := src(ctx); err != nil || m != c.want {
			t.Fatalf("%+v: %s %v", c, m, err)
		}
	}
	boom := errors.New("boom")
	src, _ := FromFlags("cases.nonfinancial", func(context.Context, string) (bool, error) { return true, boom })
	if m, err := src(ctx); !errors.Is(err, boom) || m != Disabled {
		t.Fatalf("a flag error never enables: %s %v", m, err)
	}
	if _, err := FromFlags("Cases", func(context.Context, string) (bool, error) { return true, nil }); err == nil {
		t.Fatal("malformed family")
	}
	if Disabled.Applies() || Shadow.Applies() || !Controlled.Applies() || Mode("X").Valid() {
		t.Fatal("mode predicates")
	}
}
