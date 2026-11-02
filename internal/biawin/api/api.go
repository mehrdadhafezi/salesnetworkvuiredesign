// Package api is the published biawin contract (Slice 10, READ / SHADOW).
// Integration normalizes legacy membership evidence into these neutral
// records (no legacy table, column or contact data reaches biawin); the
// offline shadow import consumes them. No command, provider call, wallet
// charge or activation exists.
package api

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// Subscription states. Only evidence-backed values are mapped: the legacy
// writers produce 'active' (insert / reactivation) and 'suspended'
// (financial reversal). Nothing is inferred from payment, invoice or item
// presence (D-35 unresolved): any other raw value is UNKNOWN with the raw
// value kept; an empty value is EMPTY.
const (
	StateActive    = "ACTIVE"
	StateSuspended = "SUSPENDED"
	StateUnknown   = "UNKNOWN"
	StateEmpty     = "EMPTY"
)

// ItemStateNotEvidenced: the source has no item-level lifecycle state; an
// item never inherits a generic status (its follow-up workflow belongs to
// fulfillment).
const ItemStateNotEvidenced = "NOT_EVIDENCED"

// Link states (Slices 5–7 vocabulary).
const (
	LinkResolved       = "RESOLVED"
	LinkRelated        = "RELATED"
	LinkUnknown        = "UNKNOWN"
	LinkConflict       = "CONFLICT"
	LinkNotRecoverable = "NOT_RECOVERABLE"
)

// Activation evidence of a subscription.
const (
	// ActivationLegacyRecorded: the legacy writer recorded the (last)
	// activation time with the active status. It is the legacy activation
	// fact only — never a target activation (no activation logic).
	ActivationLegacyRecorded = "LEGACY_ACTIVATION_RECORDED"
	ActivationNone           = "NOT_EVIDENCED"
)

// Source coverage.
const (
	SourceAvailable      = "AVAILABLE"
	SourcePartial        = "PARTIAL"
	SourceMissing        = "MISSING"
	SourceUnknown        = "UNKNOWN"
	SourceNotRecoverable = "NOT_RECOVERABLE"
	SourceStale          = "STALE"
)

// History classes.
const (
	EventSubscription = "SUBSCRIPTION_EVENT"
	EventItem         = "ITEM_EVENT"
	EventProvider     = "PROVIDER_EVENT_EVIDENCE"
)

// Link is one cross-module link with its resolution state; ID only when
// RESOLVED (never guessed, never by phone).
type Link struct {
	State string
	ID    *string
	Basis string
	// LegacyKey is the raw native reference the link was resolved from.
	LegacyKey *string
}

// SubscriptionRecord is one normalized subscription (legacy membership).
type SubscriptionRecord struct {
	SourceKey     string
	SourceRefRaw  *string
	SourceTypeRaw *string
	// Product is the historical product reference and title snapshot;
	// the product VERSION is never guessed from a title.
	ProductKey           *string
	ProductTitleSnapshot *string
	Customer             Link
	CustomerAccountKey   *string
	Invoice              Link
	FirstPaymentInvoice  Link
	Case                 Link
	OriginUserKey        *string
	OriginUserID         *string
	StateNamespace       string
	StateRaw             *string
	StateMapped          string
	ActivationEvidence   string
	ActivatedAtRaw       *string
	CreatedAtRaw         *string
	UpdatedAtRaw         *string
}

// ItemInvoiceLink is one invoice link of an item (billing ref only).
type ItemInvoiceLink struct {
	SourceKey     string
	Invoice       Link
	StatusRaw     *string
	VerifiedAtRaw *string
}

// ItemRecord is one subscription item (one included service instance).
type ItemRecord struct {
	SourceKey, SubscriptionKey string
	ServiceKey                 *string
	ServiceTitleSnapshot       *string
	InvoiceLinks               []ItemInvoiceLink
	CreatedAtRaw               *string
}

// EventRecord is one append-only legacy history event.
type EventRecord struct {
	SourceKey       string
	SubscriptionKey *string
	ItemKey         *string
	Class           string
	EventKeyRaw     string
	ActorKey        *string
	ActorUserID     *string
	OccurredAtRaw   *string
}

// LegacyClass classifies one legacy subscription for the active parity.
const (
	LegacyActive         = "ACTIVE"
	LegacyKnownOther     = "KNOWN_NOT_ACTIVE"
	LegacyUnknown        = "UNKNOWN"
	LegacyNotRecoverable = "NOT_RECOVERABLE"
)

// LegacyEvidence is the connector's INDEPENDENT legacy active classification
// of one staged snapshot (raw status, by stable source key).
type LegacyEvidence struct {
	SnapshotID string
	Cutoff     time.Time
	Status     string
	Classes    map[string]string // source key → LegacyClass
}

// Import is one normalized staged snapshot.
type Import struct {
	BatchID, SnapshotID, SourceSystem string
	Cutoff                            time.Time
	// Sources is the per-source coverage (spec id → status).
	Sources       map[string]string
	SourceRows    map[string]int64
	Subscriptions []SubscriptionRecord
	Items         []ItemRecord
	Events        []EventRecord
	// Skipped counts source rows deliberately not imported, by reason
	// (e.g. MESSAGING_ONLY events, unreadable keys).
	Skipped map[string]int64
}

func stableID(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	b := h.Sum(nil)
	b[6] = (b[6] & 0x0f) | 0x50
	b[8] = (b[8] & 0x3f) | 0x80
	x := hex.EncodeToString(b[:16])
	return x[0:8] + "-" + x[8:12] + "-" + x[12:16] + "-" + x[16:20] + "-" + x[20:32]
}

// SubscriptionID is the stable, deterministic id of a subscription: source
// system + entity kind + native key. Never random, never a phone.
func SubscriptionID(system, key string) string {
	return stableID("biawin", "subscription", system, key)
}

// ItemID is the stable id of a subscription item.
func ItemID(system, key string) string { return stableID("biawin", "subscription_item", system, key) }
