package providerops

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
)

func TestCallbackEnvelopeCanonical(t *testing.T) {
	e := CallbackEnvelope{Version: CallbackEnvelopeVersion, CallbackID: "cb-1", Provider: "BIAWIN", Reference: "crm-op-1", ExternalRef: "x-1", Status: "CONFIRMED"}
	body, err := EncodeCallback(e)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"v":1,"callback_id":"cb-1","provider":"BIAWIN","reference":"crm-op-1","external_ref":"x-1","status":"CONFIRMED"}`
	if string(body) != want {
		t.Fatalf("canonical body = %s", body)
	}
	got, err := DecodeCallback(body)
	if err != nil || got != e {
		t.Fatalf("round trip: %+v %v", got, err)
	}
	for name, b := range map[string]string{
		"whitespace":      `{"v":1, "callback_id":"cb-1","provider":"BIAWIN","reference":"crm-op-1","external_ref":"x-1","status":"CONFIRMED"}`,
		"reordered":       `{"callback_id":"cb-1","v":1,"provider":"BIAWIN","reference":"crm-op-1","external_ref":"x-1","status":"CONFIRMED"}`,
		"duplicate key":   `{"v":1,"callback_id":"cb-1","provider":"BIAWIN","reference":"crm-op-1","external_ref":"x-1","status":"REJECTED","status":"CONFIRMED"}`,
		"unknown field":   `{"v":1,"callback_id":"cb-1","provider":"BIAWIN","reference":"crm-op-1","external_ref":"x-1","status":"CONFIRMED","amount":"1"}`,
		"missing field":   `{"v":1,"callback_id":"cb-1","provider":"BIAWIN","reference":"crm-op-1","status":"CONFIRMED"}`,
		"trailing data":   want + `{}`,
		"wrong version":   `{"v":2,"callback_id":"cb-1","provider":"BIAWIN","reference":"crm-op-1","external_ref":"x-1","status":"CONFIRMED"}`,
		"not an object":   `["cb-1"]`,
		"empty":           ``,
		"escaped unicode": `{"v":1,"callback_id":"cb\u002d1","provider":"BIAWIN","reference":"crm-op-1","external_ref":"x-1","status":"CONFIRMED"}`,
	} {
		if _, err := DecodeCallback([]byte(b)); !errors.Is(err, ErrCallbackMalformed) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestAuthenticateBindsClaimsToSignedContent(t *testing.T) {
	secret := []byte("k")
	e := CallbackEnvelope{Version: 1, CallbackID: "cb-1", Provider: "BIAWIN", Reference: "crm-op-1", ExternalRef: "x-1", Status: "CONFIRMED"}
	body, _ := EncodeCallback(e)
	sign := func(b []byte) string {
		m := hmac.New(sha256.New, secret)
		m.Write(b)
		return hex.EncodeToString(m.Sum(nil))
	}
	ok := Callback{Body: body, Signature: sign(body)}
	if got, err := authenticate(secret, ok); err != nil || got != e {
		t.Fatalf("valid: %+v %v", got, err)
	}
	matching := ok
	matching.CallbackID, matching.Provider, matching.Reference, matching.ExternalRef, matching.Status = "cb-1", "BIAWIN", "crm-op-1", "x-1", "CONFIRMED"
	if _, err := authenticate(secret, matching); err != nil {
		t.Fatalf("matching claims: %v", err)
	}
	for name, mutate := range map[string]func(*Callback){
		"callback_id":  func(c *Callback) { c.CallbackID = "cb-2" },
		"provider":     func(c *Callback) { c.Provider = "OTHER" },
		"reference":    func(c *Callback) { c.Reference = "crm-op-2" },
		"external_ref": func(c *Callback) { c.ExternalRef = "x-2" },
		"status":       func(c *Callback) { c.Status = "REJECTED" },
	} {
		c := ok
		mutate(&c)
		if _, err := authenticate(secret, c); !errors.Is(err, ErrCallbackMismatch) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := authenticate(secret, Callback{Body: body, Signature: sign([]byte("other"))}); !errors.Is(err, ErrSignature) {
		t.Errorf("bad signature: %v", err)
	}
	if _, err := authenticate(nil, ok); !errors.Is(err, ErrSignature) {
		t.Errorf("empty secret: %v", err)
	}
	invalidEnv := e
	invalidEnv.Status, invalidEnv.ExternalRef = "CONFIRMED", ""
	b2, _ := EncodeCallback(invalidEnv)
	if _, err := authenticate(secret, Callback{Body: b2, Signature: sign(b2)}); !errors.Is(err, ErrInvalid) {
		t.Errorf("confirmed without external ref: %v", err)
	}
}
