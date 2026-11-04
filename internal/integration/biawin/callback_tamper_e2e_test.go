package biawin_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"

	integrationbiawin "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/biawin"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/providerops"
)

// P2 regression: the signature covers the body only, so transport-supplied
// callback fields that disagree with the signed body must never confirm,
// reject, conflict, re-route or re-identify an operation. Body and signature
// stay byte-identical in every case; only the unsigned fields are altered.
func TestCallbackUnsignedFieldsCannotMutate(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	port := integrationbiawin.NewSynthetic()
	caseID := uid(t)
	acked := func(k string) providerops.Operation {
		t.Helper()
		op, _, err := e.ops.RequestOperation(ctx, providerops.OperationRequest{OperationKey: "case:" + k + "|op:SERVICE_ORDER", Provider: "BIAWIN",
			OpType: providerops.OpServiceOrder, SubjectCaseID: &caseID, CorrelationID: "corr-" + k, CausationID: "cmd-" + k, Payload: map[string]string{"service_key": "svc"}})
		if err != nil {
			t.Fatal(err)
		}
		done, err := e.ops.Dispatch(ctx, port, op.OperationID)
		if err != nil || done.State != providerops.StateAcknowledged || done.ExternalRef == nil {
			t.Fatalf("dispatch %s: %+v %v", k, done, err)
		}
		return done
	}
	a, b := acked("tamper-a"), acked("tamper-b")
	secret := []byte("synthetic-callback-secret-p2")
	// The canonical synthetic envelope (v1), built by hand so this test is
	// independent of the encoder under test.
	body := []byte(fmt.Sprintf(`{"v":1,"callback_id":"cb-signed","provider":"BIAWIN","reference":%q,"external_ref":%q,"status":"CONFIRMED"}`,
		a.Reference, *a.ExternalRef))
	m := hmac.New(sha256.New, secret)
	m.Write(body)
	sig := hex.EncodeToString(m.Sum(nil))
	signed := providerops.Callback{Body: body, Signature: sig, CallbackID: "cb-signed", Provider: "BIAWIN", Reference: a.Reference,
		ExternalRef: *a.ExternalRef, Status: "CONFIRMED"}

	fingerprint := func() string {
		var fp string
		if err := e.admin(t).QueryRow(ctx, `SELECT (SELECT count(*) FROM integration.provider_callback)::text || '/' ||
			(SELECT count(*) FROM integration.provider_operation_event)::text || '/' ||
			(SELECT string_agg(state || ':' || version || ':' || coalesce(external_ref, '-'), ',' ORDER BY reference) FROM integration.provider_operation)`).Scan(&fp); err != nil {
			t.Fatal(err)
		}
		return fp
	}
	before := fingerprint()
	for _, tc := range []struct {
		name   string
		mutate func(*providerops.Callback)
	}{
		{"status", func(c *providerops.Callback) { c.Status = "REJECTED" }},
		{"external reference", func(c *providerops.Callback) { c.ExternalRef = "syn-order-FORGED" }},
		{"operation reference", func(c *providerops.Callback) { c.Reference = b.Reference }},
		{"callback identity", func(c *providerops.Callback) { c.CallbackID = "cb-forged" }},
		{"provider", func(c *providerops.Callback) { c.Provider = "OTHER" }},
	} {
		forged := signed
		tc.mutate(&forged)
		if string(forged.Body) != string(body) || forged.Signature != sig {
			t.Fatalf("%s: body or signature changed", tc.name)
		}
		d, err := e.ops.HandleCallback(ctx, secret, forged)
		if err == nil || d != "" {
			t.Errorf("%s: an altered unsigned field was accepted (disposition %q)", tc.name, d)
		}
		if after := fingerprint(); after != before {
			t.Fatalf("%s: an altered unsigned field mutated the ledger:\n before %s\n after  %s", tc.name, before, after)
		}
	}
	// The untouched callback still applies exactly once, to the signed operation.
	if d, err := e.ops.HandleCallback(ctx, secret, signed); err != nil || d != "APPLIED" {
		t.Fatalf("valid callback: %q %v", d, err)
	}
	if got, _ := e.ops.Get(ctx, a.OperationID); got.State != providerops.StateConfirmed {
		t.Fatalf("signed operation: %s", got.State)
	}
	if got, _ := e.ops.Get(ctx, b.OperationID); got.State != providerops.StateAcknowledged {
		t.Fatalf("other operation touched: %s", got.State)
	}
	if d, _ := e.ops.HandleCallback(ctx, secret, signed); d != "DUPLICATE" {
		t.Fatalf("replay: %q", d)
	}
}

// Callback identity is bound to the signed body: replaying the same signed
// body is DUPLICATE; the same callback id carrying different signed content
// is ID_CONFLICT and changes nothing; a validly signed but non-canonical or
// unknown-field body is refused before any read or write.
func TestCallbackIdentityBoundToSignedBody(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	port := integrationbiawin.NewSynthetic()
	caseID := uid(t)
	op, _, err := e.ops.RequestOperation(ctx, providerops.OperationRequest{OperationKey: "case:idbind|op:SERVICE_ORDER", Provider: "BIAWIN",
		OpType: providerops.OpServiceOrder, SubjectCaseID: &caseID, CorrelationID: "corr-idbind", CausationID: "cmd-idbind", Payload: map[string]string{"service_key": "svc"}})
	if err != nil {
		t.Fatal(err)
	}
	if op, err = e.ops.Dispatch(ctx, port, op.OperationID); err != nil {
		t.Fatal(err)
	}
	secret := []byte("synthetic-callback-secret-id")
	count := func() int {
		return e.count(t, `SELECT (SELECT count(*) FROM integration.provider_callback) + (SELECT count(*) FROM integration.provider_operation_event)`)
	}
	// Validly signed but malformed bodies: nothing read or written.
	before := count()
	for name, body := range map[string]string{
		"extra field": fmt.Sprintf(`{"v":1,"callback_id":"cb-x","provider":"BIAWIN","reference":%q,"external_ref":"e","status":"CONFIRMED","note":"x"}`, op.Reference),
		"duplicate status": fmt.Sprintf(`{"v":1,"callback_id":"cb-x","provider":"BIAWIN","reference":%q,"external_ref":"e","status":"REJECTED","status":"CONFIRMED"}`,
			op.Reference),
		"legacy shape": `{"ref":"` + op.Reference + `"}`,
	} {
		if _, err := e.ops.HandleCallback(ctx, secret, providerops.Callback{Body: []byte(body), Signature: hmacHex(secret, []byte(body))}); !errors.Is(err, providerops.ErrCallbackMalformed) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if count() != before {
		t.Fatal("a malformed signed body wrote evidence")
	}
	// The first signed REJECTED callback is applied and bound to its body hash.
	first := signedCallback(t, secret, providerops.CallbackEnvelope{CallbackID: "cb-same-id", Reference: op.Reference, Status: "REJECTED"})
	if d, err := e.ops.HandleCallback(ctx, secret, first); err != nil || d != "APPLIED" {
		t.Fatalf("first: %q %v", d, err)
	}
	var stored []byte
	if err := e.admin(t).QueryRow(ctx, `SELECT body_sha256 FROM integration.provider_callback WHERE callback_id = 'cb-same-id'`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if want := sha256.Sum256(first.Body); hex.EncodeToString(stored) != hex.EncodeToString(want[:]) {
		t.Fatal("disposition is not bound to the signed body hash")
	}
	if n := e.count(t, `SELECT count(*) FROM integration.provider_operation_event WHERE operation_id = $1::uuid AND kind = 'CALLBACK_REJECTED'
		AND evidence->>'callback_id' = 'cb-same-id' AND evidence->>'body_sha256' = $2`, op.OperationID, hex.EncodeToString(stored)); n != 1 {
		t.Fatalf("transition evidence not bound to the signed body: %d", n)
	}
	mid := count()
	// Same id, different signed content (now claiming CONFIRMED): never applied.
	second := signedCallback(t, secret, providerops.CallbackEnvelope{CallbackID: "cb-same-id", Reference: op.Reference, ExternalRef: "syn-order-late", Status: "CONFIRMED"})
	if d, err := e.ops.HandleCallback(ctx, secret, second); err != nil || d != "ID_CONFLICT" {
		t.Fatalf("id reuse: %q %v", d, err)
	}
	if d, _ := e.ops.HandleCallback(ctx, secret, first); d != "DUPLICATE" {
		t.Fatalf("replay: %q", d)
	}
	if got, _ := e.ops.Get(ctx, op.OperationID); got.State != providerops.StateRejected || count() != mid {
		t.Fatalf("id reuse or replay changed the ledger: %s", got.State)
	}
}
