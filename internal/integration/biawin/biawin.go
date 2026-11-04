// Package biawin is the ONLY place allowed to know the Biawin provider: its
// endpoint shapes, field names, credentials and request / response details
// (Slice 10). Slice 10 performs NO external call of any kind: the shadow is
// built only from staged legacy snapshots, and nothing in the composed
// system (crm-api, crm-biawin-shadow) depends on this package
// (architecture dependency test).
//
// The Slice 10 Provider below is READ-only. Slice 14 adds the write-side
// BiawinPort (port.go) behind integration/providerops: a Disabled adapter
// (the only one a production composition may use; it sends nothing) and a
// deterministic Synthetic TEST adapter. There is still no wallet charge
// (D-38), no wallet balance mutation, no activation, suspension or
// termination call. The wallet balance is never modelled as finance truth.
//
// Two implementations exist: Disabled (the only one a production
// composition could ever use: every call is BLOCKED, nothing is sent) and
// Recorder (a test fake that records calls and sends nothing). Neither
// opens a network connection or holds a credential.
package biawin

import (
	"context"
	"errors"
	"sync"
)

// ErrDisabled reports that the provider is not enabled (Slice 10: always).
var ErrDisabled = errors.New("integration/biawin: provider disabled (no external call is allowed in Slice 10)")

// RemoteStatus is a provider-reported status. Unknown is the only value a
// disabled provider yields; it is never zero, never ACTIVE by default.
type RemoteStatus string

// RemoteUnknown is the status when the provider is not consulted.
const RemoteUnknown RemoteStatus = "UNKNOWN"

// Provider is the READ-only Biawin port. ref is a provider reference of the
// subscription's source (never a phone, name or email).
type Provider interface {
	SubscriptionStatus(ctx context.Context, ref string) (RemoteStatus, error)
}

// Disabled is the provider when external calls are not allowed.
type Disabled struct{}

var _ Provider = Disabled{}

// SubscriptionStatus is always BLOCKED: UNKNOWN with ErrDisabled.
func (Disabled) SubscriptionStatus(context.Context, string) (RemoteStatus, error) {
	return RemoteUnknown, ErrDisabled
}

// Recorder is a test fake: it records each call and sends nothing.
type Recorder struct {
	mu    sync.Mutex
	calls []string
}

var _ Provider = (*Recorder)(nil)

// SubscriptionStatus records the call and answers like Disabled.
func (r *Recorder) SubscriptionStatus(_ context.Context, ref string) (RemoteStatus, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, "SubscriptionStatus")
	return RemoteUnknown, ErrDisabled
}

// Calls returns the number of recorded calls.
func (r *Recorder) Calls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}
