// Package synthetic is the only payment gateway adapter in Slice 12: a
// deterministic in-memory provider for tests, behind the integration
// boundary. It never opens a network connection and holds no credential;
// Production gateway delivery is UNSUPPORTED. The callback verifier shows
// the authentication contract every real adapter must meet: a callback is
// accepted only with a valid HMAC-SHA256 signature over its canonical
// fields, compared in constant time.
package synthetic

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	paymentsapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
)

// Provider is the synthetic provider name.
const Provider = "synthetic"

// ErrTransport is an injected transport failure (outcome unknown).
var ErrTransport = errors.New("synthetic gateway: transport failure")

// ErrSignature reports a callback whose signature does not verify.
var ErrSignature = errors.New("synthetic gateway: callback signature invalid")

// refSeq makes provider references unique across synthetic instances (a
// real provider's references are unique per provider).
var refSeq atomic.Int64

// Fault is an injectable failure.
type Fault int

// Faults.
const (
	None Fault = iota
	// RequestTimeoutAfterCommit: the provider records the request, the
	// answer is lost.
	RequestTimeoutAfterCommit
	// RequestTimeoutBeforeCommit: the request never reaches the provider.
	RequestTimeoutBeforeCommit
	// RequestRefused: a known refusal (proven non-commit).
	RequestRefused
	InquireDown
	VerifyDown
)

type entry struct {
	ref, status string
	amount      int64
}

// Gateway is the synthetic provider.
type Gateway struct {
	mu       sync.Mutex
	requests map[string]*entry
	faults   map[string]Fault
	// VerifyAmountOverride, when set, is reported by Verify instead of the
	// requested amount (amount-mismatch tests).
	verifyOverride map[string]int64
	Calls          map[string]int
}

var _ paymentsapi.GatewayPort = (*Gateway)(nil)

// New returns an empty synthetic provider.
func New() *Gateway {
	return &Gateway{requests: map[string]*entry{}, faults: map[string]Fault{}, verifyOverride: map[string]int64{}, Calls: map[string]int{}}
}

// Inject arms one fault for the next call of op ("request", "inquire", "verify").
func (g *Gateway) Inject(op string, f Fault) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.faults[op] = f
}

func (g *Gateway) take(op string) Fault {
	f := g.faults[op]
	delete(g.faults, op)
	g.Calls[op]++
	return f
}

// Provider implements paymentsapi.GatewayPort.
func (g *Gateway) Provider() string { return Provider }

// Request implements paymentsapi.GatewayPort; it is idempotent per request id.
func (g *Gateway) Request(_ context.Context, r paymentsapi.GatewayRequest) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	switch g.take("request") {
	case RequestTimeoutBeforeCommit:
		return "", ErrTransport
	case RequestRefused:
		return "", paymentsapi.ErrGatewayRefused
	case RequestTimeoutAfterCommit:
		g.commit(r)
		return "", ErrTransport
	}
	return g.commit(r).ref, nil
}

func (g *Gateway) commit(r paymentsapi.GatewayRequest) *entry {
	if e, ok := g.requests[r.RequestID]; ok {
		return e
	}
	e := &entry{ref: fmt.Sprintf("syn-%06d", refSeq.Add(1)), status: "PENDING", amount: r.Amount}
	g.requests[r.RequestID] = e
	return e
}

// Pay marks a request as paid by the customer (test driver).
func (g *Gateway) Pay(requestID string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	e := g.requests[requestID]
	e.status = "PAID"
	return e.ref
}

// Cancel marks a request as cancelled at the gateway (test driver).
func (g *Gateway) Cancel(requestID string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	e := g.requests[requestID]
	e.status = "CANCELLED"
	return e.ref
}

// OverrideVerifiedAmount makes Verify report amount for ref.
func (g *Gateway) OverrideVerifiedAmount(ref string, amount int64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.verifyOverride[ref] = amount
}

// Requests reports how many distinct requests the provider holds.
func (g *Gateway) Requests() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.requests)
}

// Inquire implements paymentsapi.GatewayPort.
func (g *Gateway) Inquire(_ context.Context, requestID string) (paymentsapi.GatewayInquiry, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.take("inquire") == InquireDown {
		return paymentsapi.GatewayInquiry{}, ErrTransport
	}
	e, ok := g.requests[requestID]
	if !ok {
		return paymentsapi.GatewayInquiry{Found: false}, nil
	}
	return paymentsapi.GatewayInquiry{Found: true, ExternalRef: e.ref, Status: e.status, Amount: e.amount}, nil
}

// Verify implements paymentsapi.GatewayPort.
func (g *Gateway) Verify(_ context.Context, ref string, _ int64) (paymentsapi.GatewayVerification, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.take("verify") == VerifyDown {
		return paymentsapi.GatewayVerification{}, ErrTransport
	}
	for _, e := range g.requests {
		if e.ref != ref {
			continue
		}
		if e.status != "PAID" {
			return paymentsapi.GatewayVerification{Verified: false}, nil
		}
		amount := e.amount
		if v, ok := g.verifyOverride[ref]; ok {
			amount = v
		}
		return paymentsapi.GatewayVerification{Verified: true, Amount: amount, RefID: "ref-" + ref}, nil
	}
	return paymentsapi.GatewayVerification{Verified: false}, nil
}

// RawCallback is a callback as received, before authentication.
type RawCallback struct {
	Provider, RequestID, ExternalRef, Status, Signature string
}

// Verifier authenticates callbacks with a shared secret.
type Verifier struct{ secret []byte }

// NewVerifier returns a verifier; the secret must be at least 32 bytes.
func NewVerifier(secret []byte) (*Verifier, error) {
	if len(secret) < 32 {
		return nil, errors.New("synthetic gateway: secret too short")
	}
	return &Verifier{secret: append([]byte(nil), secret...)}, nil
}

func (v *Verifier) mac(c RawCallback) []byte {
	m := hmac.New(sha256.New, v.secret)
	fmt.Fprintf(m, "%s\n%s\n%s\n%s", c.Provider, c.RequestID, c.ExternalRef, c.Status)
	return m.Sum(nil)
}

// Sign returns the signature a provider would attach (test driver).
func (v *Verifier) Sign(c RawCallback) string { return hex.EncodeToString(v.mac(c)) }

// Authenticate verifies a raw callback and returns the neutral callback.
func (v *Verifier) Authenticate(c RawCallback) (paymentsapi.GatewayCallback, error) {
	got, err := hex.DecodeString(c.Signature)
	if err != nil || !hmac.Equal(got, v.mac(c)) || c.Provider != Provider {
		return paymentsapi.GatewayCallback{}, ErrSignature
	}
	return paymentsapi.GatewayCallback{Provider: c.Provider, RequestID: c.RequestID, ExternalRef: c.ExternalRef, Status: c.Status}, nil
}
