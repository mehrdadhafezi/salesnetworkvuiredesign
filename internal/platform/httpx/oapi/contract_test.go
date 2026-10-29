package oapi_test

import (
	"context"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/refs"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
)

const bundled = "../../../../api/openapi/openapi.bundled.yaml"

func contract(t *testing.T) *openapi3.T {
	t.Helper()
	doc, err := openapi3.NewLoader().LoadFromFile(bundled)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Validate(context.Background()); err != nil {
		t.Fatalf("contract does not validate: %v", err)
	}
	return doc
}

func schema(t *testing.T, doc *openapi3.T, name string) *openapi3.Schema {
	t.Helper()
	ref, ok := doc.Components.Schemas[name]
	if !ok || ref.Value == nil {
		t.Fatalf("schema %s missing", name)
	}
	return ref.Value
}

func enum(s *openapi3.Schema) []string {
	var out []string
	for _, v := range s.Enum {
		out = append(out, v.(string))
	}
	return out
}

func TestOutcomeEnumIsTheFrozenSeven(t *testing.T) {
	doc := contract(t)
	got := enum(schema(t, doc, "Outcome"))
	var want []string
	for _, s := range result.Statuses() {
		want = append(want, s.String())
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("Outcome enum %v != kernel %v", got, want)
	}
	for _, forbidden := range []string{"OK", "ERROR", "TIMEOUT", "DENIED"} {
		if oapi.Outcome(forbidden).Valid() {
			t.Fatalf("%s must not be an outcome", forbidden)
		}
	}
}

func TestReasonCodeEnumIsFoundationV1(t *testing.T) {
	got := enum(schema(t, contract(t), "ReasonCode"))
	var want []string
	for _, c := range reasoncodes.Codes() {
		want = append(want, string(c))
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("ReasonCode enum %v != foundation-v1 %v", got, want)
	}
}

func TestMoneyIsStringAmountWithExplicitUnit(t *testing.T) {
	doc := contract(t)
	m := schema(t, doc, "Money")
	amount := m.Properties["amount"].Value
	if !amount.Type.Is("string") || amount.Format != "" || amount.Pattern == "" {
		t.Fatalf("Money.amount = %+v; must be a patterned string", amount)
	}
	for _, num := range []string{"number", "integer"} {
		if amount.Type.Is(num) {
			t.Fatalf("Money.amount must never be %s", num)
		}
	}
	sort.Strings(m.Required)
	if strings.Join(m.Required, ",") != "amount,unit" {
		t.Fatalf("Money.required = %v", m.Required)
	}
	if units := enum(schema(t, doc, "Unit")); strings.Join(units, ",") != "IRR,IRT" {
		t.Fatalf("Unit enum = %v", units)
	}
	// Generated Go: amount is a string, IRR and IRT are distinct enum values.
	if reflect.TypeOf(oapi.Money{}.Amount).Kind() != reflect.String {
		t.Fatal("generated Money.Amount must be a string")
	}
	if !oapi.Unit("IRR").Valid() || !oapi.Unit("IRT").Valid() || oapi.Unit("IRR") == oapi.Unit("IRT") || oapi.Unit("TOMAN").Valid() {
		t.Fatal("generated Unit enum")
	}
}

func TestTimestampIsRFC3339UTC(t *testing.T) {
	ts := schema(t, contract(t), "Timestamp")
	if !ts.Type.Is("string") || ts.Format != "date-time" || ts.Pattern != "Z$" {
		t.Fatalf("Timestamp = %+v", ts)
	}
}

func TestProblemAndFieldErrorShape(t *testing.T) {
	doc := contract(t)
	p := schema(t, doc, "Problem")
	sort.Strings(p.Required)
	if strings.Join(p.Required, ",") != "correlation_id,reason_code,status,title,type" {
		t.Fatalf("Problem.required = %v", p.Required)
	}
	for _, f := range []string{"type", "title", "status", "detail", "instance", "reason_code", "correlation_id", "errors"} {
		if _, ok := p.Properties[f]; !ok {
			t.Fatalf("Problem.%s missing", f)
		}
	}
	if p.Properties["errors"].Value.MaxItems == nil || *p.Properties["errors"].Value.MaxItems != 100 {
		t.Fatal("Problem.errors must be bounded")
	}
	fe := schema(t, doc, "FieldError")
	sort.Strings(fe.Required)
	if strings.Join(fe.Required, ",") != "path,reason_code" || fe.Properties["reason_code"].Ref != "#/components/schemas/ReasonCode" {
		t.Fatalf("FieldError = %+v", fe)
	}
	resp := doc.Components.Responses["Problem"].Value
	if resp.Content.Get("application/problem+json") == nil {
		t.Fatal("Problem response must be application/problem+json")
	}
}

func TestCorrelationPatternMatchesMiddleware(t *testing.T) {
	c := schema(t, contract(t), "CorrelationID")
	if c.Pattern != `^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$` {
		t.Fatalf("CorrelationID pattern %q drifted from httpx", c.Pattern)
	}
	for _, v := range []string{"client-req-0001", "00000000-0000-4000-8000-000000000000"} {
		if !httpx.ValidCorrelationID(v) {
			t.Fatalf("%q", v)
		}
	}
}

func TestPaginationAndAllowedActionsShape(t *testing.T) {
	doc := contract(t)
	page := schema(t, doc, "PageInfo")
	if _, ok := page.Properties["total"]; ok {
		t.Fatal("PageInfo must not carry a total")
	}
	if _, ok := page.Properties["next_cursor"]; !ok || page.Properties["has_more"].Value.Type.Is("boolean") == false {
		t.Fatalf("PageInfo = %+v", page)
	}
	for _, p := range []string{"Cursor", "Limit", "IdempotencyKey", "IfMatch", "CorrelationID"} {
		if _, ok := doc.Components.Parameters[p]; !ok {
			t.Fatalf("parameter %s missing", p)
		}
	}
	if !doc.Components.Parameters["IdempotencyKey"].Value.Required || doc.Components.Parameters["IdempotencyKey"].Value.Name != "Idempotency-Key" {
		t.Fatal("Idempotency-Key must be a required header component")
	}
	if doc.Components.Parameters["IfMatch"].Value.Name != "If-Match" {
		t.Fatal("If-Match component")
	}
	if _, ok := doc.Components.Headers["ETag"]; !ok {
		t.Fatal("ETag header component missing")
	}
	aa := schema(t, doc, "AllowedActions")
	if aa.AdditionalProperties.Schema == nil || aa.AdditionalProperties.Schema.Ref != "#/components/schemas/AllowedAction" {
		t.Fatalf("AllowedActions = %+v", aa.AdditionalProperties)
	}
	a := schema(t, doc, "AllowedAction")
	if strings.Join(a.Required, ",") != "allowed" || a.Properties["reason_code"].Ref != "#/components/schemas/ReasonCode" {
		t.Fatalf("AllowedAction = %+v", a)
	}
}

func TestNativeRefDomainsMatchKernel(t *testing.T) {
	got := enum(schema(t, contract(t), "NativeRef").Properties["domain"].Value)
	var want []string
	for _, d := range refs.Domains() {
		want = append(want, string(d))
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("NativeRef.domain %v != kernel %v", got, want)
	}
}

func TestResultEnvelopeShape(t *testing.T) {
	doc := contract(t)
	for _, name := range []string{"ResultEnvelope", "SessionEnvelope"} {
		e := schema(t, doc, name)
		for _, f := range []string{"outcome", "reasons", "correlation_id"} {
			found := false
			for _, r := range e.Required {
				found = found || r == f
			}
			if !found {
				t.Fatalf("%s must require %s", name, f)
			}
		}
		if e.Properties["outcome"].Ref != "#/components/schemas/Outcome" {
			t.Fatalf("%s.outcome must reference Outcome", name)
		}
	}
	re := schema(t, doc, "ResultEnvelope")
	for _, f := range []string{"items", "audit_id"} {
		if _, ok := re.Properties[f]; !ok {
			t.Fatalf("ResultEnvelope.%s missing", f)
		}
	}
}

// TestNoBusinessOrMutatingEndpoints: Slice 2 exposes exactly the technical
// surface; no business domain path and no mutating method exists.
func TestNoBusinessOrMutatingEndpoints(t *testing.T) {
	doc := contract(t)
	var paths []string
	for path, item := range doc.Paths.Map() {
		paths = append(paths, path)
		for method := range item.Operations() {
			if method != "GET" {
				t.Fatalf("mutating operation %s %s is not allowed in Slice 2", method, path)
			}
		}
		for _, op := range item.Operations() {
			for _, resp := range op.Responses.Map() {
				if resp.Value != nil && resp.Ref == "" {
					if _, ok := resp.Value.Headers["X-Correlation-ID"]; !ok {
						t.Fatalf("%s response lacks X-Correlation-ID header", path)
					}
				}
			}
		}
	}
	sort.Strings(paths)
	if strings.Join(paths, ",") != "/api/v1/session,/health/live,/health/ready" {
		t.Fatalf("paths = %v", paths)
	}
}
