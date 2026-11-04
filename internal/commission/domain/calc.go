package domain

import (
	"errors"
	"sort"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/money"
)

// FunctionID names one evidenced calculation function. The registry is a
// closed, typed set: there is no DSL, no formula text and no user-defined
// expression.
type FunctionID string

const (
	// FnPurposeMatrixStagePercentV1: amount = round(stage_amount * rate / 100, 2)
	// (class-sn-purpose-commission.php:593-606). Base = approved stage amount.
	FnPurposeMatrixStagePercentV1 FunctionID = "purpose_matrix_stage_percent_v1"
	// FnRuleInvoicePercentFixedV1: amount = round(max(0, fixed + invoice_total * rate / 100), 2)
	// for ONE rule (class-sn-plugin.php:32905-32912). Base = invoice total.
	FnRuleInvoicePercentFixedV1 FunctionID = "rule_invoice_percent_fixed_v1"
	// FnComponentProRata is NOT registered: D-36 component allocation is
	// unsupported. Named only so a request for it is recognisably BLOCKED.
	FnComponentProRata FunctionID = "component_pro_rata"
	// FnDailyAttendance is NOT registered: a non-invoice entitlement with no
	// approved key (OPD-09; class-sn-plugin.php:32835-32887).
	FnDailyAttendance FunctionID = "daily_attendance_plus_percent"
)

// BaseKind names the calculation base.
type BaseKind string

const (
	BaseApprovedStageAmount BaseKind = "APPROVED_STAGE_AMOUNT"
	BaseInvoiceTotal        BaseKind = "INVOICE_TOTAL"
	BaseComponentLine       BaseKind = "COMPONENT_LINE"
)

// Function is one registered calculation function.
type Function struct {
	ID       FunctionID
	Base     BaseKind
	Evidence string
	calc     func(Input) (Exact, []Block)
}

// Registry returns the closed set of evidenced functions.
func Registry() map[FunctionID]Function {
	return map[FunctionID]Function{
		FnPurposeMatrixStagePercentV1: {ID: FnPurposeMatrixStagePercentV1, Base: BaseApprovedStageAmount,
			Evidence: "plugin/current/includes/class-sn-purpose-commission.php:593-606", calc: percentOnly},
		FnRuleInvoicePercentFixedV1: {ID: FnRuleInvoicePercentFixedV1, Base: BaseInvoiceTotal,
			Evidence: "plugin/current/includes/class-sn-plugin.php:32905-32912", calc: percentPlusFixedFloorZero},
	}
}

// FunctionIDs lists the registered ids in order.
func FunctionIDs() []FunctionID {
	out := []FunctionID{}
	for id := range Registry() {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Input is the immutable input snapshot of one calculation.
type Input struct {
	Identity    EntitlementIdentity
	FunctionID  FunctionID
	PolicyKey   string
	PolicyVer   int
	Base        *Exact
	BaseKind    BaseKind
	Rate        *Rate
	Fixed       *Exact
	Unit        money.Unit // the policy unit
	Rounding    Rounding
	Attribution Attribution
	// RecipientFacet is the facet whose SNAPSHOT names the recipient.
	RecipientFacet Facet
	EventRef       string
	Engine         Engine
}

// Eligibility of a calculation.
type Eligibility string

const (
	Eligible   Eligibility = "ELIGIBLE"
	Ineligible Eligibility = "INELIGIBLE"
	Blocked    Eligibility = "BLOCKED"
)

// Output is one calculation result. Amount is nil unless ELIGIBLE or
// INELIGIBLE-with-amount; it is never zero-filled for missing data.
type Output struct {
	EntitlementKey *string
	FunctionID     FunctionID
	PolicyKey      string
	PolicyVer      int
	EventRef       string
	InputSHA       string
	Recipient      FacetValue
	ShareFacet     string
	BaseKind       BaseKind
	Base           *Exact
	Rate           *Rate
	Amount         *Exact
	Eligibility    Eligibility
	Blocks         []Block
	Reasons        []string
	Coverage       string // FULL / PARTIAL / NONE
}

func percentOnly(in Input) (Exact, []Block) {
	if in.Rate == nil {
		return Exact{}, []Block{BlockRateMissing}
	}
	a, err := PercentOf(*in.Base, *in.Rate, in.Rounding)
	if err != nil {
		return Exact{}, []Block{BlockNotExact}
	}
	return a, nil
}

func percentPlusFixedFloorZero(in Input) (Exact, []Block) {
	if in.Rate == nil {
		return Exact{}, []Block{BlockRateMissing}
	}
	if in.Fixed == nil {
		return Exact{}, []Block{BlockRateMissing}
	}
	if in.Fixed.Unit != in.Base.Unit {
		return Exact{}, []Block{BlockUnitMismatch}
	}
	// round(max(0, fixed + amount*rate/100), 2): the fixed part is already at
	// 2 decimals, so rounding the percent part first is exact-equivalent.
	p, err := PercentOf(*in.Base, *in.Rate, in.Rounding)
	if err != nil {
		return Exact{}, []Block{BlockNotExact}
	}
	s, err := AddExact(p, *in.Fixed)
	if err != nil {
		return Exact{}, []Block{BlockNotExact}
	}
	if s.Hundredths < 0 {
		s.Hundredths = 0
	}
	return s, nil
}

// ErrInvalidInput reports a structurally invalid calculation input.
var ErrInvalidInput = errors.New("commission: invalid calculation input")

// Calculate is the PURE calculation: deterministic, no I/O, no clock, no
// float, no hierarchy read. Every missing / unresolved input BLOCKS the
// calculation with an explicit cause; nothing is assumed or zero-filled.
func Calculate(in Input) Output {
	out := Output{FunctionID: in.FunctionID, PolicyKey: in.PolicyKey, PolicyVer: in.PolicyVer, EventRef: in.EventRef,
		ShareFacet: in.Identity.ShareFacet, BaseKind: in.BaseKind, Base: in.Base, Rate: in.Rate, Coverage: "NONE"}
	out.InputSHA = inputSHA(in)
	var blocks []Block
	if k, err := in.Identity.Key(); err == nil {
		out.EntitlementKey = &k
	} else if errors.Is(err, ErrNoApprovedKey) {
		blocks = append(blocks, BlockNonInvoiceKey)
	} else if errors.Is(err, ErrComponentKey) {
		blocks = append(blocks, BlockComponentAllocation)
	} else if errors.Is(err, ErrKeyUnresolvable) {
		blocks = append(blocks, BlockEngineRoutingUnknown)
	} else {
		blocks = append(blocks, BlockPolicyInvalid)
	}
	out.Recipient = in.Attribution.Get(in.RecipientFacet)
	if err := in.Attribution.Validate(); err != nil {
		blocks = append(blocks, BlockPolicyInvalid)
	}
	switch out.Recipient.State {
	case AttrUnknown:
		blocks = append(blocks, BlockAttributionUnknown)
	case AttrConflict:
		blocks = append(blocks, BlockAttributionConflict)
	}
	if in.Engine == EngineUnknown || in.Engine == "" {
		blocks = append(blocks, BlockEngineRoutingUnknown)
	}
	fn, ok := Registry()[in.FunctionID]
	switch {
	case in.FunctionID == FnComponentProRata || in.BaseKind == BaseComponentLine:
		blocks = append(blocks, BlockComponentAllocation)
	case !ok:
		blocks = append(blocks, BlockFunctionUnregistered)
	case fn.Base != in.BaseKind:
		blocks = append(blocks, BlockPolicyInvalid)
	}
	if !in.Rounding.Known() {
		blocks = append(blocks, BlockRoundingMissing)
	}
	if in.Base == nil {
		blocks = append(blocks, BlockBaseMissing)
	} else if !in.Unit.Valid() || in.Base.Unit != in.Unit {
		blocks = append(blocks, BlockUnitMismatch)
	}
	if len(blocks) > 0 {
		out.Eligibility, out.Blocks = Blocked, SortBlocks(blocks)
		for _, b := range out.Blocks {
			out.Reasons = append(out.Reasons, string(b.Code()))
		}
		if out.EntitlementKey != nil && in.Base != nil {
			out.Coverage = "PARTIAL"
		}
		return out
	}
	amount, fb := fn.calc(in)
	if len(fb) > 0 {
		out.Eligibility, out.Blocks = Blocked, SortBlocks(fb)
		for _, b := range out.Blocks {
			out.Reasons = append(out.Reasons, string(b.Code()))
		}
		out.Coverage = "PARTIAL"
		return out
	}
	out.Amount, out.Coverage = &amount, "FULL"
	// Legacy writers skip a non-positive rate / amount (purpose matrix :603,
	// :606): the entitlement is INELIGIBLE, with its exact zero amount kept.
	if (in.Rate != nil && in.Rate.E4 <= 0 && in.Fixed == nil) || amount.Hundredths <= 0 {
		out.Eligibility = Ineligible
		out.Reasons = []string{"RECIPIENT_INELIGIBLE"}
		return out
	}
	out.Eligibility = Eligible
	out.Reasons = []string{}
	return out
}

func inputSHA(in Input) string {
	parts := []string{string(in.Identity.Kind), in.Identity.InvoiceKey, itoa(in.Identity.Stage), in.Identity.Purpose, in.Identity.ShareFacet,
		in.Identity.Context, string(in.FunctionID), in.PolicyKey, itoa(in.PolicyVer), string(in.BaseKind), in.Unit.String(), string(in.Rounding),
		string(in.RecipientFacet), in.EventRef, string(in.Engine)}
	if in.Base != nil {
		parts = append(parts, "base="+in.Base.String())
	} else {
		parts = append(parts, "base=-")
	}
	if in.Rate != nil {
		parts = append(parts, "rate="+in.Rate.String())
	} else {
		parts = append(parts, "rate=-")
	}
	if in.Fixed != nil {
		parts = append(parts, "fixed="+in.Fixed.String())
	} else {
		parts = append(parts, "fixed=-")
	}
	for _, f := range Facets() {
		v := in.Attribution.Get(f)
		k := "-"
		if v.UserKey != nil {
			k = *v.UserKey
		}
		parts = append(parts, string(f)+"="+string(v.State)+":"+k)
	}
	return SnapshotSHA(parts...)
}

func itoa(i int) string { return FormatDecimal(int64(i), 0) }
