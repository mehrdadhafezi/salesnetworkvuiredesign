package biawin

// Slice 14 BiawinPort: the provider side of integration/providerops.
//
// Legacy evidence (plugin/current 2.0.160, class-sn-biavin-api.php):
//   - products   GET  /projects/{code}/products                       (read)
//   - create_order POST /projects/{code}/orders  {product_id, quantity,
//     customer{name, mobile, email}, reference, source}               (:43-53)
//   - charge_wallet POST /projects/{code}/wallet/charge                (:55-66)
//   - every request is HMAC-signed (api_key over method/path/timestamp/
//     payload) with a fresh random X-DCW-Request-ID; timeout 25 s; no
//     retry; no order lookup endpoint; no callback contract (:68-81).
//   - the customer is identified to the provider by name + mobile (PII):
//     the target never sends a phone as identity (AGENTS.md "Identity"),
//     so no production payload can be built from the legacy contract.
//   - a timeout is recorded by legacy as remote_failed although the order
//     may exist (class-sn-projects.php:2492-2496): the target records
//     UNKNOWN instead.
//
// Therefore: Disabled is the only adapter a production composition can
// use (sends nothing, Enabled() == false); Synthetic is a deterministic
// TEST adapter, never a production connector. charge_wallet has no port
// operation at all (D-38): providerops stores it only as BLOCKED.

import (
	"context"
	"errors"
	"sync"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/providerops"
)

// ProviderName is the provider code of the Biawin port.
const ProviderName = "BIAWIN"

// DisabledPort sends nothing.
type DisabledPort struct{}

var _ providerops.Port = DisabledPort{}

// Provider implements providerops.Port.
func (DisabledPort) Provider() string { return ProviderName }

// Enabled is always false: no production connector exists.
func (DisabledPort) Enabled() bool { return false }

// Synthetic is false.
func (DisabledPort) Synthetic() bool { return false }

// Submit never sends.
func (DisabledPort) Submit(context.Context, providerops.Request) (providerops.Response, error) {
	return providerops.Response{}, ErrDisabled
}

// Lookup is unsupported.
func (DisabledPort) Lookup(context.Context, string) (providerops.Lookup, error) {
	return providerops.Lookup{Supported: false}, ErrDisabled
}

// Behaviour scripts the synthetic adapter for one reference.
type Behaviour string

const (
	// BehaveAck acknowledges with an external reference.
	BehaveAck Behaviour = "ACK"
	// BehaveReject rejects definitively.
	BehaveReject Behaviour = "REJECT"
	// BehaveTimeoutCreated times out AFTER creating the order (the
	// ambiguous case): lookup finds it.
	BehaveTimeoutCreated Behaviour = "TIMEOUT_CREATED"
	// BehaveTimeoutAbsent times out WITHOUT creating anything: lookup
	// proves absence.
	BehaveTimeoutAbsent Behaviour = "TIMEOUT_ABSENT"
	// BehaveTimeoutNoLookup times out and offers no lookup.
	BehaveTimeoutNoLookup Behaviour = "TIMEOUT_NO_LOOKUP"
)

// SyntheticPort is a deterministic, in-memory TEST adapter. It never opens
// a network connection and holds no credential.
type SyntheticPort struct {
	mu        sync.Mutex
	behaviour map[string]Behaviour
	created   map[string]string // reference → external ref
	calls     map[string]int    // reference → dispatch calls
	Default   Behaviour
}

var _ providerops.Port = (*SyntheticPort)(nil)

// NewSynthetic returns a synthetic adapter acknowledging by default.
func NewSynthetic() *SyntheticPort {
	return &SyntheticPort{behaviour: map[string]Behaviour{}, created: map[string]string{}, calls: map[string]int{}, Default: BehaveAck}
}

// Script sets the behaviour of one reference.
func (s *SyntheticPort) Script(reference string, b Behaviour) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.behaviour[reference] = b
}

// Calls returns the dispatch calls made for a reference.
func (s *SyntheticPort) Calls(reference string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls[reference]
}

// Created reports how many distinct provider-side orders exist for a
// reference (0 or 1 when no duplicate was created).
func (s *SyntheticPort) Created(reference string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.created[reference]; ok {
		return 1
	}
	return 0
}

// Provider implements providerops.Port.
func (s *SyntheticPort) Provider() string { return ProviderName }

// Enabled is true (it is a test adapter).
func (s *SyntheticPort) Enabled() bool { return true }

// Synthetic is true.
func (s *SyntheticPort) Synthetic() bool { return true }

// ErrSyntheticTimeout is the synthetic timeout.
var ErrSyntheticTimeout = errors.New("integration/biawin: synthetic timeout")

// Submit implements providerops.Port.
func (s *SyntheticPort) Submit(_ context.Context, r providerops.Request) (providerops.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls[r.Reference]++
	b, ok := s.behaviour[r.Reference]
	if !ok {
		b = s.Default
	}
	if r.OpType != providerops.OpServiceOrder {
		return providerops.Response{Outcome: providerops.OutcomeRejected, Evidence: map[string]string{"reason": "unsupported op"}}, nil
	}
	switch b {
	case BehaveAck:
		ref := "syn-order-" + r.Reference
		s.created[r.Reference] = ref
		return providerops.Response{Outcome: providerops.OutcomeAcknowledged, ExternalRef: ref, Evidence: map[string]string{"synthetic": "true"}}, nil
	case BehaveReject:
		return providerops.Response{Outcome: providerops.OutcomeRejected, Evidence: map[string]string{"synthetic": "true"}}, nil
	case BehaveTimeoutCreated:
		s.created[r.Reference] = "syn-order-" + r.Reference
		return providerops.Response{}, ErrSyntheticTimeout
	default:
		return providerops.Response{}, ErrSyntheticTimeout
	}
}

// Lookup implements providerops.Port.
func (s *SyntheticPort) Lookup(_ context.Context, reference string) (providerops.Lookup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.behaviour[reference] == BehaveTimeoutNoLookup {
		return providerops.Lookup{Supported: false}, nil
	}
	if ref, ok := s.created[reference]; ok {
		return providerops.Lookup{Supported: true, Found: true, ExternalRef: ref, Evidence: map[string]string{"synthetic": "true"}}, nil
	}
	return providerops.Lookup{Supported: true, Found: false, Evidence: map[string]string{"synthetic": "true"}}, nil
}
