package api

import (
	"context"
	"errors"
)

// ErrGatewayRefused reports a request the provider refused with proof of
// non-commit (a known refusal; never a transport failure).
var ErrGatewayRefused = errors.New("payments: gateway refused the request")

// GatewayRequest is one payment request to a provider. RequestID is the
// stable target request identity; providers must treat it as idempotent.
type GatewayRequest struct {
	RequestID string
	Amount    int64
	Unit      string
}

// GatewayInquiry is what a provider holds for a request id.
type GatewayInquiry struct {
	// Found is false only when the provider proves it holds nothing.
	Found       bool
	ExternalRef string
	// Status is PENDING, PAID, CANCELLED or FAILED.
	Status string
	Amount int64
}

// GatewayVerification is the provider's server-side verification.
type GatewayVerification struct {
	// Verified is true only for a provider-confirmed payment of Amount.
	Verified bool
	Amount   int64
	RefID    string
}

// GatewayCallback is a provider callback AFTER the integration adapter has
// authenticated it (signature / verification). It is a claim to verify, not
// proof of payment.
type GatewayCallback struct {
	Provider    string
	RequestID   string
	ExternalRef string
	// Status is OK or NOK as the provider reported it.
	Status string
}

// GatewayPort is the payments-defined port implemented behind the
// integration boundary (synthetic only in Slice 12). Errors other than
// ErrGatewayRefused mean the outcome is UNKNOWN.
type GatewayPort interface {
	Provider() string
	Request(ctx context.Context, r GatewayRequest) (externalRef string, err error)
	Inquire(ctx context.Context, requestID string) (GatewayInquiry, error)
	Verify(ctx context.Context, externalRef string, amount int64) (GatewayVerification, error)
}
