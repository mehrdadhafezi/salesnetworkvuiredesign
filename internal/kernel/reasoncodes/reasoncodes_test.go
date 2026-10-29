package reasoncodes

import (
	"errors"
	"sort"
	"testing"
)

// frozenCatalog is written out independently of registry.go so that a value
// silently dropped from the registry fails this test. Every entry is
// "namespace/value" from the frozen sources cited in registry.go.
var frozenCatalog = []string{
	// contact (Seller Contact & Archive; Gate-0 §8 contact)
	"contact/empty", "contact/unrecorded", "contact/no_answer", "contact/callback", "contact/duplicate",
	"contact/not_purchased", "contact/pre_invoice", "contact_archive/archived_at_set",
	// invoice
	"invoice/pre_invoice", "invoice/partial_paid", "invoice/receipt_uploaded", "invoice/pending_financial_approval",
	"invoice/approved", "invoice/paid", "invoice/rejected", "invoice/cancelled", "invoice/payment_archived",
	"invoice/recontact_requested",
	// invoice_workflow
	"invoice_workflow/awaiting_payment", "invoice_workflow/awaiting_financial_approval",
	"invoice_workflow/awaiting_assignment", "invoice_workflow/completed", "invoice_workflow/archived",
	// payment_stage
	"payment_stage/pending", "payment_stage/receipt_uploaded", "payment_stage/pending_financial_approval",
	"payment_stage/approved", "payment_stage/paid", "payment_stage/rejected", "payment_stage/cancelled",
	// transaction and Dot transaction
	"transaction/pending", "transaction/verified", "transaction/approved", "transaction/rejected",
	"dot_transaction/paid", "dot_transaction/superseded", "dot_transaction/pending_finance",
	// E0 §9C legacy payment and customer-portal values
	"payment/approved", "payment/reconcile_error",
	"customer_portal_payment/pending_payment", "customer_portal_payment/expired", "customer_portal_payment/failed",
	// financial return / repeat
	"financial_return/returned_to_seller", "financial_return/resent_after_return",
	"financial_return/cancelled_by_finance", "financial_return/cancelled_by_finance_refund",
	"financial_return/payment_submitted_after_recontact", "financial_return/payment_completed",
	// finance_review
	"finance_review/pending", "finance_review/approved", "finance_review/rejected",
	// MIS validity and custody
	"mis_row_validity/valid", "mis_row_validity/invalid", "mis_row_validity/duplicate",
	"custody/unassigned", "custody/assigned_to_manager", "custody/distributed_forward", "custody/delivered_to_seller",
	"custody/returned_to_manager", "custody/returned_to_owner", "custody/returned_to_mis", "custody/converted_to_lead",
	"custody/lead_created", "custody/converted_to_dot_case", "custody/archived", "custody/cancelled",
	"custody/referral_cancelled",
	// conversion (Dot case) and converter contact
	"conversion/awaiting_access_sms", "conversion/awaiting_customer_selection", "conversion/ready_for_conversion",
	"conversion/assigned", "conversion/contacted", "conversion/customer_declined", "conversion/payment_link_sent",
	"conversion/payment_rejected", "conversion/deposit_paid", "conversion/completed",
	"conversion/archived_unpaid_subscription",
	"conversion_contact/new", "conversion_contact/no_answer", "conversion_contact/contacted",
	"conversion_contact/follow_up", "conversion_contact/confirmed", "conversion_contact/customer_declined",
	"conversion_contact/payment_link",
	// HR
	"employment/active", "employment/inactive", "employment/suspended", "employment/resigned",
	"employment/terminated", "employment/probation",
	"hr_request/pending_review", "hr_request/pending_hr", "hr_request/approved", "hr_request/rejected", "hr_request/failed",
	"compensation/active", "compensation/locked_after_payroll",
	// commission / wallet / report job
	"commission_run/generated", "commission_run/approved", "commission_run/rejected",
	"commission_run_item/posted", "commission_run_item/error",
	"commission_post_result/skipped", "commission_post_result/existing",
	"commission_preview/ready", "commission_preview/no_rule",
	"wallet_transaction/approved", "wallet_transaction_type/purpose_commission_reversed",
	"wallet_transaction_direction/credit", "wallet_transaction_direction/debit",
	"report_job/building", "report_job/ready", "report_job/cancelled", "report_export/ready",
}

func TestEveryFrozenValueIsRegistered(t *testing.T) {
	want := map[string]bool{}
	for _, s := range frozenCatalog {
		if want[s] {
			t.Fatalf("test catalog lists %s twice", s)
		}
		want[s] = true
		k, err := ParseKey(s)
		if err != nil {
			t.Fatalf("frozen value %s not registered: %v", s, err)
		}
		if !k.Known() || k.String() != s {
			t.Fatalf("ParseKey(%s) = %v", s, k)
		}
		c, err := Classify(k.Namespace, LegacyValue{Source: "fixture", Field: "status", Raw: k.Value})
		if err != nil || c.Mapping != MappingKnown || c.Key == nil || *c.Key != k {
			t.Fatalf("Classify(%s) = %+v, %v (must be KNOWN, never UNKNOWN)", s, c, err)
		}
	}
	// The registry holds exactly the frozen catalog: nothing extra, nothing missing.
	got := Keys()
	if len(got) != len(want) {
		t.Fatalf("registry has %d keys, frozen catalog has %d", len(got), len(want))
	}
	for _, k := range got {
		if !want[k.String()] {
			t.Fatalf("registry contains unexpected key %s", k)
		}
	}
	if len(Namespaces()) != 27 {
		t.Fatalf("namespaces = %d, want 27", len(Namespaces()))
	}
	for _, ns := range Namespaces() {
		if !ns.Valid() {
			t.Fatalf("namespace %s invalid", ns)
		}
	}
}

// TestNamespacesAreSliceZeroNormalization pins the foundation-v1 machine
// namespaces chosen from the frozen catalog rows. They are a normalization, not
// legacy DB names: renaming one is a contract change, and legacy provenance is
// carried by LegacyValue.Source/Field rather than by the namespace.
func TestNamespacesAreSliceZeroNormalization(t *testing.T) {
	normalized := map[Namespace]string{
		NSContactArchive:        "catalog row \"archived_at set\" (Seller Contact & Archive)",
		NSCustomerPortalPayment: "E0 §9C customer-portal values",
		NSConversionContact:     "catalog \"Dot Case / Converter Contact\" contact states",
	}
	for ns, origin := range normalized {
		if !ns.Valid() {
			t.Fatalf("normalized namespace %q (%s) must be declared", ns, origin)
		}
	}
	v := LegacyValue{Source: "legacy-portal", Field: "payment_status", Raw: "expired"}
	c, err := Classify(NSCustomerPortalPayment, v)
	if err != nil || c.Mapping != MappingKnown {
		t.Fatalf("Classify = %+v, %v", c, err)
	}
	if c.Value.Source != v.Source || c.Value.Field != v.Field || string(c.Namespace) == v.Source {
		t.Fatal("legacy provenance must stay in LegacyValue, separate from the namespace")
	}
}

func TestRequiredE0PaymentValues(t *testing.T) {
	for _, s := range []string{"payment/approved", "payment/reconcile_error", "invoice/approved", "payment_stage/approved"} {
		if _, err := ParseKey(s); err != nil {
			t.Fatalf("E0 §9C requires %s: %v", s, err)
		}
	}
}

func TestNamespaceCollisionsNeverResolveSilently(t *testing.T) {
	// "approved" exists in several namespaces; each key is distinct.
	approved := map[string]bool{}
	for _, k := range Keys() {
		if k.Value == "approved" {
			approved[k.String()] = true
		}
	}
	if len(approved) < 5 {
		t.Fatalf("expected approved in many namespaces, got %v", approved)
	}
	// A bare value has no meaning without a namespace.
	for _, s := range []string{"approved", "/approved", "approved/", "", "invoice/", "invoice/approved/x", "invoice//approved"} {
		if _, err := ParseKey(s); !errors.Is(err, ErrMalformedKey) {
			t.Fatalf("ParseKey(%q) err = %v, want ErrMalformedKey", s, err)
		}
	}
	// A value registered in one namespace does not exist in another.
	if _, err := Lookup(NSInvoice, "reconcile_error"); !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("invoice/reconcile_error must be unknown, err = %v", err)
	}
	if _, err := Lookup(NSPayment, "pre_invoice"); !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("payment/pre_invoice must be unknown, err = %v", err)
	}
	if _, err := Lookup("Invoice", "approved"); !errors.Is(err, ErrUnknownNamespace) {
		t.Fatalf("namespace match must be exact, err = %v", err)
	}
	if _, err := ParseKey("status/approved"); !errors.Is(err, ErrUnknownNamespace) {
		t.Fatalf("undeclared namespace err = %v", err)
	}
	a, _ := Lookup(NSInvoice, "approved")
	b, _ := Lookup(NSPaymentStage, "approved")
	if a == b {
		t.Fatal("invoice/approved and payment_stage/approved must be different keys")
	}
}

func TestUnknownLegacyRawIsPreserved(t *testing.T) {
	raws := []string{"Approved", " approved", "approved ", "APPROVED", "", "خرید کرده", "custom_status_9", "\x00\xff", "approved\n"}
	for _, raw := range raws {
		v := LegacyValue{Source: "fixture", Field: "status", Raw: raw}
		c, err := Classify(NSInvoice, v)
		if err != nil {
			t.Fatal(err)
		}
		if c.Mapping != MappingUnknown || c.Key != nil {
			t.Fatalf("raw %q mapped to %+v; must be UNKNOWN (no coercion)", raw, c)
		}
		if c.Value != v || c.Value.Raw != raw {
			t.Fatalf("raw %q not preserved byte-for-byte: %q", raw, c.Value.Raw)
		}
	}
	if _, err := Classify("nope", LegacyValue{Source: "s", Field: "f", Raw: "approved"}); !errors.Is(err, ErrUnknownNamespace) {
		t.Fatalf("undeclared namespace must error, got %v", err)
	}
	for _, v := range []LegacyValue{{Field: "f"}, {Source: "s"}} {
		if _, err := Classify(NSInvoice, v); !errors.Is(err, ErrInvalidLegacyValue) {
			t.Fatalf("invalid legacy value %+v err = %v", v, err)
		}
	}
	if MappingKnown.String() != "KNOWN" || MappingUnknown.String() != "UNKNOWN" || Mapping(0).String() != "INVALID" {
		t.Fatal("mapping names")
	}
}

func TestCRMCodes(t *testing.T) {
	codes := Codes()
	if len(codes) != 15 {
		t.Fatalf("foundation-v1 has 15 CRM codes, got %d", len(codes))
	}
	for _, c := range codes {
		parsed, err := ParseCode(string(c))
		if err != nil || parsed != c {
			t.Fatalf("ParseCode(%s) = %v, %v", c, parsed, err)
		}
		b, err := c.MarshalText()
		if err != nil || string(b) != string(c) {
			t.Fatalf("MarshalText(%s)", c)
		}
		var back Code
		if err := back.UnmarshalText(b); err != nil || back != c {
			t.Fatalf("UnmarshalText(%s)", c)
		}
	}
	for _, bad := range []string{"", "contract_invalid", "UNKNOWN", "SUCCESS", "invoice/approved"} {
		if _, err := ParseCode(bad); !errors.Is(err, ErrUnknownCode) {
			t.Fatalf("ParseCode(%q) err = %v", bad, err)
		}
	}
	if _, err := Code("X").MarshalText(); err == nil {
		t.Fatal("invalid code must not marshal")
	}
	var c Code
	if err := c.UnmarshalText([]byte("X")); err == nil {
		t.Fatal("invalid code must not unmarshal")
	}
	if Version != "foundation-v1" {
		t.Fatalf("Version = %s", Version)
	}
}

func TestNormalizeDeterministic(t *testing.T) {
	in := []Code{StaleState, ContractInvalid, StaleState, ActionDenied}
	got, err := Normalize(in)
	if err != nil {
		t.Fatal(err)
	}
	want := []Code{ActionDenied, ContractInvalid, StaleState}
	if len(got) != len(want) {
		t.Fatalf("Normalize = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Normalize = %v, want %v", got, want)
		}
	}
	if !sort.SliceIsSorted(got, func(i, j int) bool { return got[i] < got[j] }) {
		t.Fatal("not sorted")
	}
	reversed := []Code{ActionDenied, StaleState, ContractInvalid}
	again, _ := Normalize(reversed)
	for i := range again {
		if again[i] != got[i] {
			t.Fatal("Normalize must be order-independent")
		}
	}
	if empty, err := Normalize(nil); err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("Normalize(nil) = %v, %v", empty, err)
	}
	if _, err := Normalize([]Code{"BOGUS"}); !errors.Is(err, ErrUnknownCode) {
		t.Fatalf("Normalize(bogus) err = %v", err)
	}
}

func FuzzClassifyPreservesRaw(f *testing.F) {
	for _, s := range []string{"approved", "Approved", "", "reconcile_error", "x/y"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		c, err := Classify(NSPayment, LegacyValue{Source: "s", Field: "f", Raw: raw})
		if err != nil {
			t.Fatal(err)
		}
		if c.Value.Raw != raw {
			t.Fatalf("raw changed: %q -> %q", raw, c.Value.Raw)
		}
		known := raw == "approved" || raw == "reconcile_error"
		if known != (c.Mapping == MappingKnown) || (c.Key != nil) != known {
			t.Fatalf("raw %q classified %+v", raw, c)
		}
	})
}

func FuzzParseKey(f *testing.F) {
	for _, s := range []string{"invoice/approved", "approved", "payment/reconcile_error", "x/y/z", ""} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		k, err := ParseKey(s)
		if err != nil {
			return
		}
		if k.String() != s || !k.Known() {
			t.Fatalf("ParseKey(%q) accepted non-registered or non-canonical key %v", s, k)
		}
	})
}
