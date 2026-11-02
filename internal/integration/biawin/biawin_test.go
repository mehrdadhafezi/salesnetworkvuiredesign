package biawin

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// The port has exactly one READ method: no order, charge, wallet,
// activation, suspension or termination method exists.
func TestPortIsReadOnly(t *testing.T) {
	ty := reflect.TypeOf((*Provider)(nil)).Elem()
	if ty.NumMethod() != 1 || ty.Method(0).Name != "SubscriptionStatus" {
		t.Fatalf("provider port must stay read-only: %d methods", ty.NumMethod())
	}
}

func TestDisabledAndRecorderSendNothing(t *testing.T) {
	ctx := context.Background()
	if s, err := (Disabled{}).SubscriptionStatus(ctx, "ref"); s != RemoteUnknown || !errors.Is(err, ErrDisabled) {
		t.Fatalf("disabled: %s %v", s, err)
	}
	r := &Recorder{}
	if r.Calls() != 0 {
		t.Fatal("fresh recorder")
	}
	if s, err := r.SubscriptionStatus(ctx, "ref"); s != RemoteUnknown || !errors.Is(err, ErrDisabled) || r.Calls() != 1 {
		t.Fatalf("recorder: %s %v %d", s, err, r.Calls())
	}
}
