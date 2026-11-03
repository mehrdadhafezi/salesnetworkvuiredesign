package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/money"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
)

// NormalizationVersion versions the shadow import + calculation derivation.
const NormalizationVersion = 1

// Engine names a legacy commission engine (routing is OPD-09: never chosen).
type Engine string

const (
	// EnginePurposeMatrix is E1 (class-sn-purpose-commission.php).
	EnginePurposeMatrix Engine = "PURPOSE_MATRIX"
	// EngineRulesAuto is E2, the rules auto-credit (class-sn-plugin.php:35854).
	EngineRulesAuto Engine = "RULES_AUTO"
	// EngineDryRunPosting is E3, the approved dry-run poster (:33602).
	EngineDryRunPosting Engine = "DRY_RUN_POSTING"
	// EngineProjectCard is E4, the Biawin project card (class-sn-projects.php:1181).
	EngineProjectCard Engine = "PROJECT_CARD"
	// EngineUnknown is a commission credit whose writer cannot be proven.
	EngineUnknown Engine = "ENGINE_ROUTING_UNKNOWN"
)

// Engines lists every engine value.
func Engines() []Engine {
	return []Engine{EnginePurposeMatrix, EngineRulesAuto, EngineDryRunPosting, EngineProjectCard, EngineUnknown}
}

// Attribution facets (INV-029; AGENTS.md "Ownership"): never collapsed.
type Facet string

const (
	FacetSourceOwner          Facet = "SOURCE_OWNER"
	FacetOriginalSeller       Facet = "ORIGINAL_SELLER"
	FacetCreditOwner          Facet = "CREDIT_OWNER"
	FacetCurrentCustody       Facet = "CURRENT_CUSTODY"
	FacetCurrentOrgAssignment Facet = "CURRENT_ORG_ASSIGNMENT"
	FacetEventActor           Facet = "EVENT_ACTOR"
)

// Facets lists the six distinct attribution facets.
func Facets() []Facet {
	return []Facet{FacetSourceOwner, FacetOriginalSeller, FacetCreditOwner, FacetCurrentCustody, FacetCurrentOrgAssignment, FacetEventActor}
}

// AttributionState is the state of one attribution facet.
type AttributionState string

const (
	// AttrSnapshot: the value was captured at event time by the writer.
	AttrSnapshot AttributionState = "SNAPSHOT"
	AttrUnknown  AttributionState = "UNKNOWN"
	AttrConflict AttributionState = "CONFLICT"
)

// FacetValue is one attribution facet of one entitlement. A value is kept
// only for SNAPSHOT; the current hierarchy is NEVER used to fill a gap.
type FacetValue struct {
	State   AttributionState `json:"state"`
	UserKey *string          `json:"user_key,omitempty"`
	Basis   string           `json:"basis"`
}

// Attribution is the immutable attribution snapshot of one entitlement.
type Attribution map[Facet]FacetValue

// Validate rejects a value without SNAPSHOT state and unknown facets.
func (a Attribution) Validate() error {
	known := map[Facet]bool{}
	for _, f := range Facets() {
		known[f] = true
	}
	for f, v := range a {
		if !known[f] {
			return fmt.Errorf("commission: unknown facet %q", f)
		}
		switch v.State {
		case AttrSnapshot:
			if v.UserKey == nil || *v.UserKey == "" {
				return fmt.Errorf("commission: facet %s SNAPSHOT needs a user key", f)
			}
		case AttrUnknown, AttrConflict:
			if v.UserKey != nil {
				return fmt.Errorf("commission: facet %s %s must not carry a user key", f, v.State)
			}
		default:
			return fmt.Errorf("commission: facet %s state %q", f, v.State)
		}
	}
	return nil
}

// Get returns the facet value, UNKNOWN when absent.
func (a Attribution) Get(f Facet) FacetValue {
	if v, ok := a[f]; ok {
		return v
	}
	return FacetValue{State: AttrUnknown, Basis: "not captured by the legacy writer"}
}

// EntitlementKind separates invoice and non-invoice entitlements.
type EntitlementKind string

const (
	KindInvoiceStage EntitlementKind = "INVOICE_STAGE"
	KindInvoice      EntitlementKind = "INVOICE"
	// KindNonInvoice (e.g. daily attendance, class-sn-plugin.php:32835) has
	// no approved identity key: OPD-09 blocks it.
	KindNonInvoice EntitlementKind = "NON_INVOICE"
	// KindComponent is a component (membership item / content line)
	// entitlement: D-36 component allocation is unsupported, no key.
	KindComponent EntitlementKind = "COMPONENT"
	// KindUnresolved is an entitlement whose engine cannot be proven
	// (OPD-09 routing): no key may be minted for it.
	KindUnresolved EntitlementKind = "UNRESOLVED"
)

// EntitlementIdentity is the business identity of an entitlement. The
// recipient is a snapshot ATTRIBUTE, never part of the key: a hierarchy or
// recipient change can never mint a second entitlement for the same
// invoice / stage / purpose / share facet / context.
type EntitlementIdentity struct {
	Kind       EntitlementKind
	InvoiceKey string
	Stage      int
	Purpose    string
	ShareFacet string
	// Context is the policy generation context (the engine family).
	Context string
}

// ErrNoApprovedKey reports an entitlement whose key is not approved (OPD-09).
var ErrNoApprovedKey = errors.New("commission: no approved entitlement key (OPD-09)")

// ErrComponentKey reports a component entitlement (D-36).
var ErrComponentKey = errors.New("commission: component allocation entitlement has no key (D-36)")

// ErrKeyUnresolvable reports an entitlement of an unproven engine.
var ErrKeyUnresolvable = errors.New("commission: entitlement engine unproven; no key (OPD-09)")

// Key returns the canonical entitlement key.
func (e EntitlementIdentity) Key() (string, error) {
	switch e.Kind {
	case KindNonInvoice:
		return "", ErrNoApprovedKey
	case KindComponent:
		return "", ErrComponentKey
	case KindUnresolved:
		return "", ErrKeyUnresolvable
	}
	if e.Kind != KindInvoiceStage && e.Kind != KindInvoice {
		return "", fmt.Errorf("commission: entitlement kind %q", e.Kind)
	}
	if e.InvoiceKey == "" || e.Purpose == "" || e.ShareFacet == "" || e.Context == "" {
		return "", errors.New("commission: invoice, purpose, share facet and context are required")
	}
	if e.Kind == KindInvoiceStage && e.Stage < 1 {
		return "", errors.New("commission: a stage entitlement needs stage >= 1")
	}
	for _, p := range []string{e.InvoiceKey, e.Purpose, e.ShareFacet, e.Context} {
		if strings.ContainsAny(p, "|\n") {
			return "", errors.New("commission: key parts must not contain separators")
		}
	}
	return fmt.Sprintf("inv:%s|stage:%d|purpose:%s|facet:%s|ctx:%s", e.InvoiceKey, e.Stage, e.Purpose, e.ShareFacet, e.Context), nil
}

// EvidenceState is how much a policy version is backed by evidence.
type EvidenceState string

const (
	// EvidenceObservedUnversioned: read from a mutable legacy table at
	// snapshot time; the value at any earlier date is UNKNOWN.
	EvidenceObservedUnversioned EvidenceState = "OBSERVED_UNVERSIONED"
	// EvidenceEventSnapshot: copied by the writer into the credit at event time.
	EvidenceEventSnapshot EvidenceState = "EVENT_SNAPSHOT"
	// EvidenceInvalid: the observed values are self-contradictory.
	EvidenceInvalid EvidenceState = "INVALID"
)

// ApprovalState of a policy: nothing is approved by Slice 13.
const PolicyNotApproved = "NOT_APPROVED"

// Policy is one commission policy version. Values are exact integers.
type Policy struct {
	PolicyKey        string // stable: source + purpose + share facet (+ rule code)
	Purpose          string
	ShareFacet       string
	EligibilityRef   string
	FunctionID       FunctionID
	Rate             *Rate
	Fixed            *Exact
	Base             BaseKind
	Unit             money.Unit
	EffectiveFrom    *time.Time // inclusive, date
	EffectiveTo      *time.Time // inclusive, date
	AttributionFacet Facet
	Scope            string
	Provenance       string
	Evidence         EvidenceState
	ApprovalState    string
	SourceRef        string
}

// ErrInvalidInterval reports an effective interval whose end precedes its start.
var ErrInvalidInterval = errors.New("commission: invalid effective interval")

// Validate checks the policy is internally consistent.
func (p Policy) Validate() error {
	if p.PolicyKey == "" || p.Purpose == "" || p.ShareFacet == "" || p.EligibilityRef == "" || p.Provenance == "" || p.SourceRef == "" {
		return errors.New("commission: policy key, purpose, share facet, eligibility, provenance and source are required")
	}
	if _, ok := Registry()[p.FunctionID]; !ok {
		return fmt.Errorf("commission: function %q is not registered", p.FunctionID)
	}
	if !p.Unit.Valid() {
		return money.ErrInvalidUnit
	}
	if p.EffectiveFrom != nil && p.EffectiveTo != nil && p.EffectiveTo.Before(*p.EffectiveFrom) {
		return ErrInvalidInterval
	}
	if p.Rate != nil && (p.Rate.E4 < 0 || p.Rate.E4 > 100*10000) {
		return errors.New("commission: rate must be within 0..100 percent")
	}
	if p.ApprovalState != PolicyNotApproved {
		return errors.New("commission: Slice 13 approves no policy")
	}
	switch p.Evidence {
	case EvidenceObservedUnversioned, EvidenceEventSnapshot, EvidenceInvalid:
	default:
		return fmt.Errorf("commission: evidence state %q", p.Evidence)
	}
	return nil
}

// ContentSHA is the content hash of a policy version (version identity).
func (p Policy) ContentSHA() []byte {
	parts := []string{p.PolicyKey, p.Purpose, p.ShareFacet, p.EligibilityRef, string(p.FunctionID), string(p.Base), p.Unit.String(),
		string(p.AttributionFacet), p.Scope, p.Provenance, string(p.Evidence)}
	if p.Rate != nil {
		parts = append(parts, "rate="+p.Rate.String())
	}
	if p.Fixed != nil {
		parts = append(parts, "fixed="+p.Fixed.String())
	}
	for _, t := range []*time.Time{p.EffectiveFrom, p.EffectiveTo} {
		if t != nil {
			parts = append(parts, t.UTC().Format("2006-01-02"))
		} else {
			parts = append(parts, "-")
		}
	}
	h := sha256.Sum256([]byte(strings.Join(parts, "\x1f")))
	return h[:]
}

// Covers reports whether date d is inside the effective interval. An absent
// bound is UNKNOWN, never "always": ok=false means coverage is not provable.
func (p Policy) Covers(d time.Time) (in bool, ok bool) {
	if p.EffectiveFrom == nil || p.EffectiveTo == nil {
		return false, false
	}
	day := d.UTC().Truncate(24 * time.Hour)
	return !day.Before(p.EffectiveFrom.UTC()) && !day.After(p.EffectiveTo.UTC()), true
}

// Block is a commission-domain cause for a BLOCKED / UNKNOWN calculation,
// mapped to a frozen foundation-v1 reason code.
type Block string

const (
	BlockRoundingMissing      Block = "ROUNDING_POLICY_MISSING"
	BlockFunctionUnregistered Block = "FUNCTION_NOT_REGISTERED"
	BlockComponentAllocation  Block = "COMPONENT_ALLOCATION_UNSUPPORTED"
	BlockNonInvoiceKey        Block = "NON_INVOICE_KEY_UNAPPROVED"
	BlockUnitMismatch         Block = "UNIT_MISMATCH"
	BlockBaseMissing          Block = "BASE_MISSING"
	BlockRateMissing          Block = "RATE_MISSING"
	BlockAttributionUnknown   Block = "ATTRIBUTION_UNKNOWN"
	BlockAttributionConflict  Block = "ATTRIBUTION_CONFLICT"
	BlockEngineRoutingUnknown Block = "ENGINE_ROUTING_UNKNOWN"
	BlockInvalidInterval      Block = "INVALID_EFFECTIVE_INTERVAL"
	BlockNotExact             Block = "AMOUNT_NOT_EXACT"
	BlockPolicyInvalid        Block = "POLICY_INVALID"
)

// Code maps a block to its frozen reason code.
func (b Block) Code() reasoncodes.Code {
	switch b {
	case BlockAttributionUnknown, BlockEngineRoutingUnknown:
		return reasoncodes.DependencyUnknown
	case BlockAttributionConflict, BlockUnitMismatch:
		return reasoncodes.LinkageConflict
	case BlockBaseMissing, BlockRateMissing:
		return reasoncodes.CoverageIncomplete
	case BlockNotExact, BlockInvalidInterval, BlockPolicyInvalid:
		return reasoncodes.ContractInvalid
	default:
		return reasoncodes.PolicyUnresolved
	}
}

// SortBlocks orders and de-duplicates blocks.
func SortBlocks(in []Block) []Block {
	seen := map[Block]bool{}
	out := []Block{}
	for _, b := range in {
		if !seen[b] {
			seen[b] = true
			out = append(out, b)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// SnapshotSHA hashes an ordered list of snapshot parts (input snapshot ref).
func SnapshotSHA(parts ...string) string {
	h := sha256.Sum256([]byte(strings.Join(parts, "\x1f")))
	return hex.EncodeToString(h[:])
}
