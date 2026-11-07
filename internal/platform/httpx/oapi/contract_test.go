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

// slice11Commands are the only mutating operations: the evidenced, safe
// Slice 11 commands. No invoice-issue, return, release, reset, transfer,
// allocation, ready, HR final-apply or generic transition path may appear.
var slice11Commands = map[string]bool{
	"/api/v1/cases/{case_id}/contact-outcomes":   true,
	"/api/v1/hr/requests":                        true,
	"/api/v1/hr/requests/{request_id}/decisions": true,
	// Slice 12: receipt staging and the owning seller's manual payment batch
	// only. No review decision, posting, refund, settlement, wallet,
	// commission, gateway, next-stage, invoice-issue or return path.
	"/api/v1/payments/uploads":                       true,
	"/api/v1/payments/invoices/{invoice_id}/batches": true,
	// Slice 14: target-native SHADOW work-case commands and human provider
	// reconciliation only. No activation, wallet charge, cancel, reopen,
	// review, provider dispatch or legacy write-back path.
	"/api/v1/fulfillment/work-cases/{case_id}/assignment":               true,
	"/api/v1/fulfillment/work-cases/{case_id}/operational-actions":      true,
	"/api/v1/fulfillment/work-cases/{case_id}/evidence-records":         true,
	"/api/v1/fulfillment/work-cases/{case_id}/outcome":                  true,
	"/api/v1/integration/provider-operations/{operation_id}/resolution": true,
	// R1 authentication: server-side session lifecycle only. No enrolment,
	// password set / reset, recovery, MFA, OTP or customer login path.
	"/api/v1/auth/login":      true,
	"/api/v1/auth/logout":     true,
	"/api/v1/auth/logout-all": true,
}

// TestNoBusinessOrMutatingEndpoints: Slice 2 exposes exactly the technical
// surface; no business domain path and no mutating method exists.
func TestNoBusinessOrMutatingEndpoints(t *testing.T) {
	doc := contract(t)
	var paths []string
	for path, item := range doc.Paths.Map() {
		paths = append(paths, path)
		for method := range item.Operations() {
			if method != "GET" && !(method == "POST" && slice11Commands[path]) {
				t.Fatalf("mutating operation %s %s is not allowed (only the Slice 11 / 12 / 14 evidenced commands and R1 session lifecycle mutate)", method, path)
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
	// Slice 2 technical surface + Slice 4 read-only identity/organization/authorization surface
	// + Slice 5 read-only customer identity surface (no merge/resolve/write path).
	want := []string{"/api/v1/authorization/effective-access",
		// Slice 10 Biawin read shadow: GET only; no activation / suspension /
		// order / wallet / provider path.
		"/api/v1/biawin/parity", "/api/v1/biawin/subscriptions", "/api/v1/biawin/subscriptions/{subscription_id}",
		"/api/v1/biawin/subscriptions/{subscription_id}/items",
		// Slice 7 billing + payments read shadow: GET only, no action path.
		"/api/v1/billing/reconciliation-report",
		// Slice 6 sales read shadow: GET only, no action path.
		"/api/v1/cases", "/api/v1/cases/{case_id}", "/api/v1/cases/{case_id}/attribution", "/api/v1/cases/{case_id}/contact-outcomes",
		"/api/v1/cases/{case_id}/history",
		"/api/v1/custody/assignments", "/api/v1/custody/cases/{case_id}",
		"/api/v1/customer-identity/merge-cases", "/api/v1/customer-identity/merge-cases/{merge_case_id}",
		"/api/v1/customer-identity/resolution-report", "/api/v1/customers", "/api/v1/customers/{customer_id}",
		"/api/v1/customers/{customer_id}/contact-points", "/api/v1/customers/{customer_id}/evidence",
		// Slice 9 finance SHADOW ledger read: GET only; no post / approve /
		// reverse / settle / refund / credit / debit path.
		"/api/v1/finance/accounts", "/api/v1/finance/accounts/{account_id}", "/api/v1/finance/balances", "/api/v1/finance/ledger",
		"/api/v1/finance/ledger/{entry_id}", "/api/v1/finance/reconciliation", "/api/v1/finance/shadow-runs", "/api/v1/finance/shadow-runs/{run_id}",
		// Slice 10 fulfillment read shadow: GET only; no assignment / shipping /
		// completion / evidence upload / upgrade path.
		"/api/v1/fulfillment/cases", "/api/v1/fulfillment/cases/{case_id}", "/api/v1/fulfillment/cases/{case_id}/history",
		// Slice 11 HR change requests: request / step decision only (no final apply path).
		"/api/v1/hr/requests", "/api/v1/hr/requests/{request_id}", "/api/v1/hr/requests/{request_id}/decisions",
		"/api/v1/identity/users", "/api/v1/identity/users/{user_id}",
		"/api/v1/invoice-review", "/api/v1/invoice-review/{invoice_id}",
		"/api/v1/invoices", "/api/v1/invoices/{invoice_id}", "/api/v1/invoices/{invoice_id}/items", "/api/v1/invoices/{invoice_id}/payments",
		"/api/v1/invoices/{invoice_id}/receipts", "/api/v1/invoices/{invoice_id}/stages", "/api/v1/invoices/{invoice_id}/totals",
		"/api/v1/mis/overview", "/api/v1/mis/reconciliation", "/api/v1/mis/source-quality",
		"/api/v1/organization/assignments", "/api/v1/organization/hierarchy-findings", "/api/v1/organization/levels",
		"/api/v1/organization/org-units", "/api/v1/organization/persons", "/api/v1/organization/persons/{person_id}",
		"/api/v1/organization/positions",
		"/api/v1/payments/attempts", "/api/v1/payments/attempts/{attempt_id}", "/api/v1/payments/attempts/{attempt_id}/history",
		"/api/v1/payments/receipts/{receipt_id}",
		"/api/v1/reporting/metrics", "/api/v1/reporting/metrics/{metric_id}", "/api/v1/reports/cases", "/api/v1/reports/cases/{case_id}",
		"/api/v1/reports/custody", "/api/v1/reports/custody/{case_id}", "/api/v1/reports/invoices", "/api/v1/reports/invoices/{invoice_id}",
		"/api/v1/reports/payments", "/api/v1/reports/payments/{attempt_id}", "/api/v1/sales/coverage-report", "/api/v1/sales/custody-parity",
		"/api/v1/session", "/api/v1/auth/login", "/api/v1/auth/logout", "/api/v1/auth/logout-all",
		// Slice 11 write-back diagnostics: read only, no fix / retry path.
		"/api/v1/writeback/commands", "/api/v1/writeback/commands/{command_id}", "/api/v1/writeback/summary",
		"/health/live", "/health/ready",
		// Slice 12 payments writes and the read-only Finance review queue
		// (no decision path).
		"/api/v1/payments/uploads", "/api/v1/payments/invoices/{invoice_id}/submission", "/api/v1/payments/invoices/{invoice_id}/batches",
		"/api/v1/payments/batches/{batch_id}", "/api/v1/payments/evidence", "/api/v1/finance/review/attempts",
		"/api/v1/finance/review/attempts/{attempt_id}",
		// Slice 13 commission controlled SHADOW read: GET only, no calculate,
		// approve, post, credit, settle or reverse path.
		"/api/v1/commission/overview", "/api/v1/commission/runs", "/api/v1/commission/calculations", "/api/v1/commission/findings",
		"/api/v1/commission/policies", "/api/v1/commission/historical-runs", "/api/v1/commission/me/earnings",
		"/api/v1/fulfillment/work-cases", "/api/v1/fulfillment/work-cases/{case_id}", "/api/v1/fulfillment/work-cases/{case_id}/assignment",
		"/api/v1/fulfillment/work-cases/{case_id}/operational-actions", "/api/v1/fulfillment/work-cases/{case_id}/evidence-records",
		"/api/v1/fulfillment/work-cases/{case_id}/outcome", "/api/v1/biawin/activation-decisions",
		"/api/v1/integration/provider-operations", "/api/v1/integration/provider-operations/{operation_id}/resolution"}
	sort.Strings(want)
	if strings.Join(paths, ",") != strings.Join(want, ",") {
		t.Fatalf("paths = %v", paths)
	}
}
