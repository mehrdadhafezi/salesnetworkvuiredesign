package api

import "time"

// Customer-lifecycle branches (Cross-Role customer lifecycle). Only
// ADDITIONAL_SERVICES enters Biawin; a Complaint goes to After-Sales and an
// Archive is neither deletion nor a financial release.
const (
	RouteAdditionalServices = "ADDITIONAL_SERVICES"
	RouteComplaint          = "AFTER_SALES_COMPLAINT"
	RouteArchive            = "ARCHIVE"
	RouteUnknown            = "UNKNOWN"
)

// Activation decisions.
const (
	DecisionShadowActivated = "SHADOW_ACTIVATED"
	DecisionBlocked         = "BLOCKED"
	DecisionNotApplicable   = "NOT_APPLICABLE"
)

// Blocking causes (commission-style domain vocabulary, each mapped to a
// frozen reason code by the service).
const (
	BlockD35                = "D-35"
	BlockRouteUnknown       = "ROUTE_UNKNOWN"
	BlockIdentityUnknown    = "CUSTOMER_LINEAGE_UNKNOWN"
	BlockIdentityConflict   = "CUSTOMER_LINEAGE_CONFLICT"
	BlockComponentsMissing  = "CATALOG_COMPONENTS_MISSING"
	BlockServiceMissing     = "SERVICE_DEFINITION_MISSING"
	BlockServiceRetired     = "SERVICE_DEFINITION_RETIRED"
	BlockFinancialBehaviour = "FINANCIAL_ENTITLEMENT_UNRESOLVED"
)

// Customer is the customer-identity evidence of a paid invoice (customers
// published resolution; never a phone match).
type Customer struct {
	State    string  `json:"state"` // RESOLVED / UNKNOWN / CONFLICT
	ID       *string `json:"customer_id,omitempty"`
	Existing bool    `json:"existing_customer"`
	Basis    string  `json:"basis"`
}

// Route is the customer-lifecycle branch evidence of the invoice.
type Route struct {
	Branch string `json:"branch"`
	Basis  string `json:"basis"`
}

// ProductLine is one paid subscription product line.
type ProductLine struct {
	ProductKey     string `json:"product_key"`
	ProductVersion int    `json:"product_version"`
}

// InvoiceFullyPaid is the inbound activation trigger (billing / payments
// fact relayed by integration). Slice 14 has NO production producer: the
// legacy system stays master, so only synthetic and shadow messages exist.
type InvoiceFullyPaid struct {
	MessageID     string        `json:"message_id"`
	CorrelationID string        `json:"correlation_id"`
	InvoiceID     string        `json:"invoice_id"`
	InvoiceRef    *string       `json:"invoice_ref,omitempty"`
	Customer      Customer      `json:"customer"`
	Route         Route         `json:"route"`
	Lines         []ProductLine `json:"lines"`
	PaidAt        time.Time     `json:"paid_at"`
}

// Outbox event types.
const (
	EventSubscriptionActivated = "biawin.subscription_activated"
	EventItemActivated         = "biawin.subscription_item_activated"
)

// SubscriptionActivated is published once per SHADOW activation.
type SubscriptionActivated struct {
	ActivationID   string    `json:"activation_id"`
	BusinessKey    string    `json:"business_key"`
	SubscriptionID string    `json:"subscription_id"`
	CustomerID     string    `json:"customer_id"`
	InvoiceID      string    `json:"invoice_id"`
	ItemCount      int       `json:"item_count"`
	RolloutMode    string    `json:"rollout_mode"`
	CorrelationID  string    `json:"correlation_id"`
	CausationID    string    `json:"causation_id"`
	OccurredAt     time.Time `json:"occurred_at"`
}

// SubscriptionItemActivated is published once per item of a SHADOW
// activation; fulfillment creates one case per item from it.
type SubscriptionItemActivated struct {
	ActivationID   string    `json:"activation_id"`
	SubscriptionID string    `json:"subscription_id"`
	ItemID         string    `json:"item_id"`
	Seq            int       `json:"seq"`
	ServiceKey     string    `json:"service_key"`
	ServiceVersion int       `json:"service_version"`
	CustomerID     string    `json:"customer_id"`
	RolloutMode    string    `json:"rollout_mode"`
	CorrelationID  string    `json:"correlation_id"`
	CausationID    string    `json:"causation_id"`
	OccurredAt     time.Time `json:"occurred_at"`
}
