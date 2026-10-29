package reasoncodes

import (
	"errors"
	"fmt"
	"strings"
)

// The known-legacy registry is the frozen status catalog required by E0 §9C:
// every name/namespace row of docs/source-of-truth/STATUS-TRANSITIONS.fa.md,
// every value of GATE-0-PRODUCT-INVARIANTS.md §8, the legacy payment values
// approved/reconcile_error and the customer-portal values pending_payment,
// expired and failed. Values only have meaning inside their namespace.

var (
	// ErrUnknownNamespace reports an undeclared namespace.
	ErrUnknownNamespace = errors.New("reasoncodes: unknown namespace")
	// ErrUnknownKey reports a namespace/value pair that is not registered.
	ErrUnknownKey = errors.New("reasoncodes: unknown namespace/value")
	// ErrMalformedKey reports a key string that is not "namespace/value".
	ErrMalformedKey = errors.New("reasoncodes: malformed key")
	// ErrInvalidLegacyValue reports a LegacyValue without source or field.
	ErrInvalidLegacyValue = errors.New("reasoncodes: invalid legacy value")
)

// Namespace qualifies a legacy status value.
type Namespace string

// Declared namespaces. Catalog sources are noted per group.
const (
	// Gate-0 §8 "contact"; catalog "Seller Contact & Archive".
	NSContact        Namespace = "contact"
	NSContactArchive Namespace = "contact_archive"
	// Gate-0 §8 invoice / invoice_workflow / payment_stage; catalog "Invoice / Stage / Payment".
	NSInvoice         Namespace = "invoice"
	NSInvoiceWorkflow Namespace = "invoice_workflow"
	NSPaymentStage    Namespace = "payment_stage"
	NSTransaction     Namespace = "transaction"
	NSDotTransaction  Namespace = "dot_transaction"
	// E0 §9C legacy payment values and customer-portal values.
	NSPayment               Namespace = "payment"
	NSCustomerPortalPayment Namespace = "customer_portal_payment"
	// Catalog "Financial Return / Repeat".
	NSFinancialReturn Namespace = "financial_return"
	// Gate-0 §8 finance_review.
	NSFinanceReview Namespace = "finance_review"
	// Catalog "Custody / MIS"; Gate-0 §8 case/custody.
	NSMISRowValidity Namespace = "mis_row_validity"
	NSCustody        Namespace = "custody"
	// Gate-0 §8 conversion; catalog "Dot Case / Converter Contact".
	NSConversion        Namespace = "conversion"
	NSConversionContact Namespace = "conversion_contact"
	// Catalog "HR"; Gate-0 §8 hr_request.
	NSEmployment   Namespace = "employment"
	NSHRRequest    Namespace = "hr_request"
	NSCompensation Namespace = "compensation"
	// Catalog "Commission / Wallet / Report Job"; Gate-0 §8 commission_run/item,
	// wallet_transaction, report_job.
	NSCommissionRun              Namespace = "commission_run"
	NSCommissionRunItem          Namespace = "commission_run_item"
	NSCommissionPostResult       Namespace = "commission_post_result"
	NSCommissionPreview          Namespace = "commission_preview"
	NSWalletTransaction          Namespace = "wallet_transaction"
	NSWalletTransactionType      Namespace = "wallet_transaction_type"
	NSWalletTransactionDirection Namespace = "wallet_transaction_direction"
	NSReportJob                  Namespace = "report_job"
	NSReportExport               Namespace = "report_export"
)

// catalog is the frozen registry content, in catalog order.
var catalog = []struct {
	ns     Namespace
	values []string
}{
	{NSContact, []string{"empty", "unrecorded", "no_answer", "callback", "duplicate", "not_purchased", "pre_invoice"}},
	{NSContactArchive, []string{"archived_at_set"}},
	{NSInvoice, []string{"pre_invoice", "partial_paid", "receipt_uploaded", "pending_financial_approval", "approved", "paid", "rejected", "cancelled", "payment_archived", "recontact_requested"}},
	{NSInvoiceWorkflow, []string{"awaiting_payment", "awaiting_financial_approval", "awaiting_assignment", "completed", "archived"}},
	{NSPaymentStage, []string{"pending", "receipt_uploaded", "pending_financial_approval", "approved", "paid", "rejected", "cancelled"}},
	{NSTransaction, []string{"pending", "verified", "approved", "rejected"}},
	{NSDotTransaction, []string{"paid", "superseded", "pending_finance"}},
	{NSPayment, []string{"approved", "reconcile_error"}},
	{NSCustomerPortalPayment, []string{"pending_payment", "expired", "failed"}},
	{NSFinancialReturn, []string{"returned_to_seller", "resent_after_return", "cancelled_by_finance", "cancelled_by_finance_refund", "payment_submitted_after_recontact", "payment_completed"}},
	{NSFinanceReview, []string{"pending", "approved", "rejected"}},
	{NSMISRowValidity, []string{"valid", "invalid", "duplicate"}},
	{NSCustody, []string{"unassigned", "assigned_to_manager", "distributed_forward", "delivered_to_seller", "returned_to_manager", "returned_to_owner", "returned_to_mis", "converted_to_lead", "lead_created", "converted_to_dot_case", "archived", "cancelled", "referral_cancelled"}},
	{NSConversion, []string{"awaiting_access_sms", "awaiting_customer_selection", "ready_for_conversion", "assigned", "contacted", "customer_declined", "payment_link_sent", "payment_rejected", "deposit_paid", "completed", "archived_unpaid_subscription"}},
	{NSConversionContact, []string{"new", "no_answer", "contacted", "follow_up", "confirmed", "customer_declined", "payment_link"}},
	{NSEmployment, []string{"active", "inactive", "suspended", "resigned", "terminated", "probation"}},
	{NSHRRequest, []string{"pending_review", "pending_hr", "approved", "rejected", "failed"}},
	{NSCompensation, []string{"active", "locked_after_payroll"}},
	{NSCommissionRun, []string{"generated", "approved", "rejected"}},
	{NSCommissionRunItem, []string{"posted", "error"}},
	{NSCommissionPostResult, []string{"skipped", "existing"}},
	{NSCommissionPreview, []string{"ready", "no_rule"}},
	{NSWalletTransaction, []string{"approved"}},
	{NSWalletTransactionType, []string{"purpose_commission_reversed"}},
	{NSWalletTransactionDirection, []string{"credit", "debit"}},
	{NSReportJob, []string{"building", "ready", "cancelled"}},
	{NSReportExport, []string{"ready"}},
}

var registry = func() map[Namespace]map[string]bool {
	m := make(map[Namespace]map[string]bool, len(catalog))
	for _, group := range catalog {
		if _, dup := m[group.ns]; dup {
			panic("reasoncodes: duplicate namespace " + string(group.ns))
		}
		values := make(map[string]bool, len(group.values))
		for _, v := range group.values {
			if values[v] {
				panic("reasoncodes: duplicate value " + string(group.ns) + "/" + v)
			}
			values[v] = true
		}
		m[group.ns] = values
	}
	return m
}()

// Namespaces returns every declared namespace in catalog order.
func Namespaces() []Namespace {
	out := make([]Namespace, 0, len(catalog))
	for _, group := range catalog {
		out = append(out, group.ns)
	}
	return out
}

// Valid reports whether ns is declared.
func (ns Namespace) Valid() bool {
	_, ok := registry[ns]
	return ok
}

// Key is a namespace-qualified registered value, rendered "namespace/value".
type Key struct {
	Namespace Namespace
	Value     string
}

// String renders "namespace/value".
func (k Key) String() string { return string(k.Namespace) + "/" + k.Value }

// Known reports whether k is registered.
func (k Key) Known() bool { return registry[k.Namespace][k.Value] }

// Keys returns every registered key in catalog order.
func Keys() []Key {
	var out []Key
	for _, group := range catalog {
		for _, v := range group.values {
			out = append(out, Key{Namespace: group.ns, Value: v})
		}
	}
	return out
}

// Lookup returns the registered key for an exact namespace/value pair.
func Lookup(ns Namespace, value string) (Key, error) {
	if !ns.Valid() {
		return Key{}, fmt.Errorf("%w: %q", ErrUnknownNamespace, string(ns))
	}
	k := Key{Namespace: ns, Value: value}
	if !k.Known() {
		return Key{}, fmt.Errorf("%w: %s", ErrUnknownKey, k)
	}
	return k, nil
}

// ParseKey parses "namespace/value". A bare value without a namespace is
// rejected: identical values in different namespaces never resolve silently.
func ParseKey(s string) (Key, error) {
	ns, value, ok := strings.Cut(s, "/")
	if !ok || ns == "" || value == "" || strings.Contains(value, "/") {
		return Key{}, fmt.Errorf("%w: %q", ErrMalformedKey, s)
	}
	return Lookup(Namespace(ns), value)
}

// LegacyValue is a raw legacy value preserved verbatim with where it came from.
// Source and Field are opaque descriptors supplied by the legacy integration;
// the kernel holds no legacy table knowledge.
type LegacyValue struct {
	Source string
	Field  string
	Raw    string
}

// Validate requires a source and a field. Raw may be any byte sequence, including empty.
func (v LegacyValue) Validate() error {
	if v.Source == "" || v.Field == "" {
		return ErrInvalidLegacyValue
	}
	return nil
}

// Mapping states whether a legacy value matched the registry.
type Mapping uint8

const (
	mappingInvalid Mapping = iota
	// MappingKnown: the raw value is a registered value of the namespace.
	MappingKnown
	// MappingUnknown: no registered match; the raw value is preserved unchanged.
	MappingUnknown
)

// String returns "KNOWN", "UNKNOWN" or "INVALID".
func (m Mapping) String() string {
	switch m {
	case MappingKnown:
		return "KNOWN"
	case MappingUnknown:
		return "UNKNOWN"
	default:
		return "INVALID"
	}
}

// Classification is the result of classifying a LegacyValue in a namespace.
// Value is always the input, unchanged. Key is set only when Mapping is KNOWN.
type Classification struct {
	Value     LegacyValue
	Namespace Namespace
	Mapping   Mapping
	Key       *Key
}

// Classify matches v.Raw byte-for-byte against the namespace. No trimming, case
// folding or alias resolution is applied: anything else is UNKNOWN with the raw
// value preserved. An undeclared namespace is an error, never a guess.
func Classify(ns Namespace, v LegacyValue) (Classification, error) {
	if err := v.Validate(); err != nil {
		return Classification{}, err
	}
	if !ns.Valid() {
		return Classification{}, fmt.Errorf("%w: %q", ErrUnknownNamespace, string(ns))
	}
	if registry[ns][v.Raw] {
		k := Key{Namespace: ns, Value: v.Raw}
		return Classification{Value: v, Namespace: ns, Mapping: MappingKnown, Key: &k}, nil
	}
	return Classification{Value: v, Namespace: ns, Mapping: MappingUnknown}, nil
}
