package synthetic

import (
	"context"
	"errors"
	"testing"

	paymentsapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
)

func TestCallbackAuthentication(t *testing.T) {
	v, err := NewVerifier([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewVerifier([]byte("short")); err == nil {
		t.Fatal("short secret accepted")
	}
	c := RawCallback{Provider: Provider, RequestID: "r", ExternalRef: "syn-000001", Status: "OK"}
	c.Signature = v.Sign(c)
	if cb, err := v.Authenticate(c); err != nil || cb.Status != "OK" || cb.ExternalRef != "syn-000001" {
		t.Fatalf("valid callback: %+v %v", cb, err)
	}
	forged := c
	forged.Status = "NOK"
	if _, err := v.Authenticate(forged); !errors.Is(err, ErrSignature) {
		t.Fatal("a tampered callback must not authenticate")
	}
	forged = c
	forged.Signature = "zz"
	if _, err := v.Authenticate(forged); !errors.Is(err, ErrSignature) {
		t.Fatal("a malformed signature must not authenticate")
	}
}

func TestIdempotentRequestAndFaults(t *testing.T) {
	ctx := context.Background()
	g := New()
	r := paymentsapi.GatewayRequest{RequestID: "a", Amount: 10, Unit: "IRT"}
	ref1, err := g.Request(ctx, r)
	if err != nil {
		t.Fatal(err)
	}
	if ref2, _ := g.Request(ctx, r); ref2 != ref1 || g.Requests() != 1 {
		t.Fatal("request must be idempotent per request id")
	}
	g.Inject("request", RequestTimeoutAfterCommit)
	if _, err := g.Request(ctx, paymentsapi.GatewayRequest{RequestID: "b", Amount: 5}); !errors.Is(err, ErrTransport) {
		t.Fatal("timeout after commit")
	}
	if inq, _ := g.Inquire(ctx, "b"); !inq.Found {
		t.Fatal("committed request must be found")
	}
	g.Inject("request", RequestTimeoutBeforeCommit)
	_, _ = g.Request(ctx, paymentsapi.GatewayRequest{RequestID: "c", Amount: 5})
	if inq, _ := g.Inquire(ctx, "c"); inq.Found {
		t.Fatal("uncommitted request must be absent")
	}
}
