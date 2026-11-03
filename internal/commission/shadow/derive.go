// Package shadow is the commission module's OFFLINE shadow job (a job
// command, never reachable from a read path, a GET or a page render). One
// staged snapshot under one normalization version and one rounding mode is
// one insert-only run: it records the legacy evidence verbatim, observes
// policy versions, recalculates every historical entitlement from its
// EVENT-TIME snapshot with the pure registry and records parity and
// reconciliation findings. It never approves, posts, credits, settles or
// reverses anything, never dispatches a posting request and never writes
// another module's schema.
package shadow

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/domain"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/money"
)

// PolicyObs is one observed policy content.
type PolicyObs struct {
	Policy domain.Policy
	SHA    []byte
}

// CalcRow is one derived calculation with its parity.
type CalcRow struct {
	SourceKey      string
	Engine         domain.Engine
	Identity       domain.EntitlementIdentity
	Attribution    domain.Attribution
	RecipientUser  *string
	PolicyKey      *string
	PolicySHA      []byte
	Input          domain.Input
	Output         domain.Output
	LegacyAmount   *domain.Exact
	Parity         domain.ParityRow
	FirstRecipient *string
	legacy         domain.Legacy
	dup            bool
}

// Finding is one reconciliation finding.
type Finding struct {
	Kind, Severity, SourceKey, Detail string
	EntitlementKey                    *string
}

// Derived is the pure output of one derivation.
type Derived struct {
	Policies []PolicyObs
	Calcs    []CalcRow
	Findings []Finding
	Summary  api.ParitySummary
}

const (
	purposeInvoiceRules = "invoice_rules"
	purposeProjectCard  = "project_card"
)

func exact(raw *string) *domain.Exact {
	if raw == nil {
		return nil
	}
	e, err := domain.ParseExact(*raw, money.IRT)
	if err != nil {
		return nil
	}
	return &e
}

func rate(raw *string) *domain.Rate {
	if raw == nil {
		return nil
	}
	r, err := domain.ParseRate(*raw)
	if err != nil {
		return nil
	}
	return &r
}

func date(raw *string) *time.Time {
	if raw == nil || *raw == "" || *raw == "0000-00-00" {
		return nil
	}
	t, err := time.Parse("2006-01-02", (*raw)[:min(10, len(*raw))])
	if err != nil {
		return nil
	}
	return &t
}

func snapshot(k *string, basis string) domain.FacetValue {
	if k == nil {
		return domain.FacetValue{State: domain.AttrUnknown, Basis: "not captured by the legacy writer"}
	}
	v := *k
	return domain.FacetValue{State: domain.AttrSnapshot, UserKey: &v, Basis: basis}
}

func matrixPolicyKey(purpose, role string) string { return "purpose_matrix:" + purpose + ":" + role }

func observe(p domain.Policy) PolicyObs {
	if p.Validate() != nil {
		p.Evidence, p.Rate, p.Fixed, p.EffectiveFrom, p.EffectiveTo = domain.EvidenceInvalid, nil, nil, nil, nil
	}
	return PolicyObs{Policy: p, SHA: p.ContentSHA()}
}

// Derive is the pure derivation of one run. It never reads a clock, the
// current hierarchy or any live source: every value comes from ev.
func Derive(ev api.Evidence, rounding domain.Rounding) Derived {
	d := Derived{}
	policies := map[string]PolicyObs{}
	addPolicy := func(o PolicyObs) []byte {
		policies[o.Policy.PolicyKey+"\x1f"+string(o.SHA)] = o
		return o.SHA
	}
	observed := map[string]*domain.Rate{}
	for _, m := range ev.Matrix {
		if m.PurposeRaw == "" || m.RoleRaw == "" {
			continue
		}
		r := rate(m.RateRaw)
		observed[m.PurposeRaw+"\x1f"+m.RoleRaw] = r
		addPolicy(observe(domain.Policy{PolicyKey: matrixPolicyKey(m.PurposeRaw, m.RoleRaw), Purpose: m.PurposeRaw, ShareFacet: m.RoleRaw,
			EligibilityRef: "legacy.e1.rate_positive_recipient_resolved", FunctionID: domain.FnPurposeMatrixStagePercentV1, Rate: r,
			Base: domain.BaseApprovedStageAmount, Unit: money.IRT, AttributionFacet: domain.FacetCreditOwner, Scope: "invoice_stage",
			Provenance: "legacy.purpose_commission_rate.v1 (mutable; observed at snapshot)", Evidence: domain.EvidenceObservedUnversioned,
			ApprovalState: domain.PolicyNotApproved, SourceRef: m.SourceKey}))
	}
	rules := map[string]api.Rule{}
	rulePolicy := map[string]PolicyObs{}
	for _, r := range ev.Rules {
		rules[r.SourceKey] = r
		code := r.SourceKey
		if r.RuleCode != nil && *r.RuleCode != "" {
			code = *r.RuleCode
		}
		fn := domain.FnRuleInvoicePercentFixedV1
		if r.CalculationTypeRaw != nil && *r.CalculationTypeRaw != "invoice_percent_fixed" {
			continue // daily attendance: non-invoice (OPD-09); not a registered function
		}
		o := observe(domain.Policy{PolicyKey: "rule:" + code, Purpose: purposeInvoiceRules, ShareFacet: "seller",
			EligibilityRef: "legacy.e2e3.rule_match(position,level,employment,payment_method,invoice_status,amount_range,effective_interval)",
			FunctionID:     fn, Rate: rate(r.RateRaw), Fixed: exact(r.FixedRaw), Base: domain.BaseInvoiceTotal, Unit: money.IRT,
			EffectiveFrom: date(r.EffectiveFromRaw), EffectiveTo: date(r.EffectiveToRaw), AttributionFacet: domain.FacetOriginalSeller, Scope: "invoice",
			Provenance: "legacy.commission_rule.v1 (mutable; observed at snapshot)", Evidence: domain.EvidenceObservedUnversioned,
			ApprovalState: domain.PolicyNotApproved, SourceRef: r.SourceKey})
		addPolicy(o)
		rulePolicy[r.SourceKey] = o
	}

	// Live credits per entitlement (not rewritten in place, not reversed).
	reversed := map[string]bool{}
	byKey := map[string]api.LegacyCredit{}
	for _, c := range ev.Credits {
		byKey[c.TxKey] = c
		if c.Direction == api.DirDebit {
			if c.ReversesTxKey == nil {
				d.Findings = append(d.Findings, Finding{Kind: "REVERSAL_LINK_MISSING", Severity: "WARNING", SourceKey: c.TxKey,
					Detail: "reversal debit names no original credit (" + c.TypeRaw + ")"})
			} else {
				reversed[*c.ReversesTxKey] = true
			}
		}
		if c.RewrittenInPlace {
			d.Findings = append(d.Findings, Finding{Kind: "REWRITTEN_IN_PLACE", Severity: "INFO", SourceKey: c.TxKey,
				Detail: "legacy reversal rewrote this credit's type to " + c.TypeRaw + " and nulled its idempotency key"})
		}
	}
	for _, c := range ev.Credits {
		if c.Direction == api.DirDebit && c.ReversesTxKey != nil {
			if _, ok := byKey[*c.ReversesTxKey]; !ok {
				d.Findings = append(d.Findings, Finding{Kind: "REVERSAL_LINK_MISSING", Severity: "WARNING", SourceKey: c.TxKey,
					Detail: "reversal names transaction " + *c.ReversesTxKey + " which is not in the snapshot"})
			}
		}
	}
	live := func(c api.LegacyCredit) bool { return !c.RewrittenInPlace && !reversed[c.TxKey] }

	type group struct {
		identity domain.EntitlementIdentity
		credits  []api.LegacyCredit
	}
	groups := map[string]*group{}
	var order []string
	var keyless []api.LegacyCredit
	invoiceEngines := map[string]map[domain.Engine]bool{}
	for _, c := range ev.Credits {
		if c.Direction != api.DirCredit {
			continue
		}
		eng := domain.Engine(c.Engine)
		if c.InvoiceKey != nil && live(c) {
			if invoiceEngines[*c.InvoiceKey] == nil {
				invoiceEngines[*c.InvoiceKey] = map[domain.Engine]bool{}
			}
			invoiceEngines[*c.InvoiceKey][eng] = true
		}
		if eng == domain.EngineDryRunPosting {
			continue // represented by its dry-run item (and checked by posting linkage)
		}
		id, ok := identityOf(c)
		if !ok {
			keyless = append(keyless, c)
			continue
		}
		k, err := id.Key()
		if err != nil {
			keyless = append(keyless, c)
			continue
		}
		if groups[k] == nil {
			groups[k] = &group{identity: id}
			order = append(order, k)
		}
		groups[k].credits = append(groups[k].credits, c)
	}
	sort.Strings(order)
	invKeys := make([]string, 0, len(invoiceEngines))
	for k := range invoiceEngines {
		invKeys = append(invKeys, k)
	}
	sort.Strings(invKeys)
	for _, inv := range invKeys {
		if len(invoiceEngines[inv]) > 1 {
			names := []string{}
			for e := range invoiceEngines[inv] {
				names = append(names, string(e))
			}
			sort.Strings(names)
			d.Findings = append(d.Findings, Finding{Kind: "DUPLICATE_RISK", Severity: "BLOCKING", SourceKey: "invoice:" + inv,
				Detail: "live commission credits from more than one engine for the same invoice: " + strings.Join(names, ", ")})
		}
	}

	for _, k := range order {
		g := groups[k]
		var lives []api.LegacyCredit
		for _, c := range g.credits {
			if live(c) {
				lives = append(lives, c)
			}
		}
		chosen := g.credits[len(g.credits)-1]
		if len(lives) > 0 {
			chosen = lives[len(lives)-1]
		}
		dup := len(lives) > 1
		ek := k
		if dup {
			d.Findings = append(d.Findings, Finding{Kind: "DUPLICATE_RISK", Severity: "BLOCKING", SourceKey: chosen.TxKey, EntitlementKey: &ek,
				Detail: fmt.Sprintf("%d live credits for one entitlement", len(lives))})
		}
		recips := map[string]bool{}
		for _, c := range g.credits {
			if c.HolderKey != nil {
				recips[*c.HolderKey] = true
			}
		}
		if len(recips) > 1 {
			d.Findings = append(d.Findings, Finding{Kind: "RECIPIENT_MISMATCH", Severity: "WARNING", SourceKey: chosen.TxKey, EntitlementKey: &ek,
				Detail: "the same entitlement was credited to different recipients over time (a later credit used the then-current hierarchy); one entitlement kept"})
		}
		row := d.creditCalc(chosen, g.identity, rounding, observed, rules, rulePolicy, addPolicy, dup)
		if f := g.credits[0].HolderKey; f != nil {
			row.FirstRecipient = f
		}
		d.Calcs = append(d.Calcs, row)
	}
	for _, c := range keyless {
		d.Calcs = append(d.Calcs, d.creditCalc(c, identityFallback(c), rounding, observed, rules, rulePolicy, addPolicy, false))
	}

	// E3 dry-run items: calculations per item; credits linked to items.
	items := map[string]api.DryRunItem{}
	for _, it := range ev.DryRunItems {
		items[it.SourceKey] = it
		d.Calcs = append(d.Calcs, itemCalc(it, rounding, rules, rulePolicy))
		if it.NonInvoice {
			d.Findings = append(d.Findings, Finding{Kind: "NON_INVOICE_BLOCKED", Severity: "INFO", SourceKey: "item:" + it.SourceKey,
				Detail: "non-invoice dry-run entitlement (daily attendance): no approved key (OPD-09)"})
		}
	}
	for _, c := range ev.Credits {
		if c.Engine != string(domain.EngineDryRunPosting) || c.Direction != api.DirCredit {
			continue
		}
		it, ok := items[deref(c.DryRunItemKey)]
		switch {
		case !ok:
			d.Findings = append(d.Findings, Finding{Kind: "POSTING_LINK_MISMATCH", Severity: "WARNING", SourceKey: c.TxKey,
				Detail: "dry-run posting credit names an item that is not in the snapshot"})
		case it.WalletTxKey == nil || *it.WalletTxKey != c.TxKey:
			d.Findings = append(d.Findings, Finding{Kind: "POSTING_LINK_MISMATCH", Severity: "WARNING", SourceKey: c.TxKey,
				Detail: "dry-run item " + it.SourceKey + " does not link back to this credit"})
		case !sameExact(exact(it.CalculatedRaw), exact(c.AmountRaw)):
			d.Findings = append(d.Findings, Finding{Kind: "POSTING_LINK_MISMATCH", Severity: "WARNING", SourceKey: c.TxKey,
				Detail: "posted credit amount differs from the dry-run item's calculated amount"})
		}
	}
	for _, r := range ev.DryRuns {
		if r.ApprovalRaw != nil && *r.ApprovalRaw == "approved" && r.RejectedAtRaw != nil && *r.RejectedAtRaw != "" {
			d.Findings = append(d.Findings, Finding{Kind: "REJECTED_THEN_APPROVED", Severity: "INFO", SourceKey: "run:" + r.SourceKey,
				Detail: "legacy run was rejected and later approved (class-sn-plugin.php:33568 allows it); both are preserved"})
		}
	}

	keys := make([]string, 0, len(policies))
	for k := range policies {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		d.Policies = append(d.Policies, policies[k])
	}
	sort.SliceStable(d.Calcs, func(i, j int) bool { return d.Calcs[i].SourceKey < d.Calcs[j].SourceKey })
	sort.SliceStable(d.Findings, func(i, j int) bool {
		a, b := d.Findings[i], d.Findings[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.SourceKey < b.SourceKey
	})
	return d
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func sameExact(a, b *domain.Exact) bool {
	return a != nil && b != nil && a.Unit == b.Unit && a.Hundredths == b.Hundredths
}

// identityOf returns the entitlement identity of a credit whose engine has
// approved identity semantics (E1 stage, E2 invoice). E3 credits are
// represented by their dry-run items; E4 and unproven engines have none.
func identityOf(c api.LegacyCredit) (domain.EntitlementIdentity, bool) {
	if c.InvoiceKey == nil {
		return domain.EntitlementIdentity{}, false
	}
	switch domain.Engine(c.Engine) {
	case domain.EnginePurposeMatrix:
		if c.StageRaw == nil || c.PurposeRaw == nil || c.RoleRaw == nil {
			return domain.EntitlementIdentity{}, false
		}
		st, err := strconv.Atoi(*c.StageRaw)
		if err != nil || st < 1 {
			return domain.EntitlementIdentity{}, false
		}
		return domain.EntitlementIdentity{Kind: domain.KindInvoiceStage, InvoiceKey: *c.InvoiceKey, Stage: st, Purpose: *c.PurposeRaw,
			ShareFacet: *c.RoleRaw, Context: string(domain.EnginePurposeMatrix)}, true
	case domain.EngineRulesAuto:
		facet := "seller"
		if strings.HasPrefix(c.TypeRaw, "supervisor_commission") {
			facet = "supervisor"
		}
		return domain.EntitlementIdentity{Kind: domain.KindInvoice, InvoiceKey: *c.InvoiceKey, Purpose: purposeInvoiceRules, ShareFacet: facet,
			Context: string(domain.EngineRulesAuto)}, true
	}
	return domain.EntitlementIdentity{}, false
}

func identityFallback(c api.LegacyCredit) domain.EntitlementIdentity {
	switch domain.Engine(c.Engine) {
	case domain.EngineProjectCard:
		return domain.EntitlementIdentity{Kind: domain.KindComponent, InvoiceKey: deref(c.InvoiceKey), Purpose: purposeProjectCard, ShareFacet: "biavin",
			Context: string(domain.EngineProjectCard)}
	case domain.EngineDryRunPosting:
		return domain.EntitlementIdentity{Kind: domain.KindUnresolved, InvoiceKey: deref(c.InvoiceKey), Context: string(domain.EngineDryRunPosting)}
	}
	return domain.EntitlementIdentity{Kind: domain.KindUnresolved, InvoiceKey: deref(c.InvoiceKey), Context: c.Engine}
}

func (d *Derived) creditCalc(c api.LegacyCredit, id domain.EntitlementIdentity, rounding domain.Rounding, observed map[string]*domain.Rate,
	rules map[string]api.Rule, rulePolicy map[string]PolicyObs, addPolicy func(PolicyObs) []byte, dup bool) CalcRow {
	eng := domain.Engine(c.Engine)
	row := CalcRow{SourceKey: "tx:" + c.TxKey, Engine: eng, Identity: id, LegacyAmount: exact(c.AmountRaw)}
	in := domain.Input{Identity: id, Unit: money.IRT, Rounding: rounding, Engine: eng, RecipientFacet: domain.FacetCreditOwner,
		EventRef: "legacy.wallet_transaction.v1:" + c.TxKey}
	attr := domain.Attribution{}
	legacy := domain.Legacy{SourceKey: row.SourceKey, Engine: eng, Amount: row.LegacyAmount, AmountRaw: c.AmountRaw, RecipientKey: c.HolderKey,
		Comparable: true}
	switch eng {
	case domain.EnginePurposeMatrix:
		attr[domain.FacetCreditOwner] = snapshot(c.RecipientSnapKey, "meta.recipient_user_id written at credit time")
		in.FunctionID, in.BaseKind = domain.FnPurposeMatrixStagePercentV1, domain.BaseApprovedStageAmount
		in.Base, in.Rate = exact(c.BaseRaw), rate(c.RateRaw)
		legacy.SnapshotKey, legacy.Base, legacy.RateSnapshot = c.RecipientSnapKey, in.Base, in.Rate
		if c.PurposeRaw != nil && c.RoleRaw != nil {
			legacy.RateObserved = observed[*c.PurposeRaw+"\x1f"+*c.RoleRaw]
			if in.Rate != nil {
				o := observe(domain.Policy{PolicyKey: matrixPolicyKey(*c.PurposeRaw, *c.RoleRaw), Purpose: *c.PurposeRaw, ShareFacet: *c.RoleRaw,
					EligibilityRef: "legacy.e1.rate_positive_recipient_resolved", FunctionID: domain.FnPurposeMatrixStagePercentV1, Rate: in.Rate,
					Base: domain.BaseApprovedStageAmount, Unit: money.IRT, AttributionFacet: domain.FacetCreditOwner, Scope: "invoice_stage",
					Provenance: "legacy wallet transaction meta.rate_percent (event-time snapshot)", Evidence: domain.EvidenceEventSnapshot,
					ApprovalState: domain.PolicyNotApproved, SourceRef: c.TxKey})
				if o.Policy.Evidence != domain.EvidenceInvalid {
					pk := o.Policy.PolicyKey
					row.PolicyKey, row.PolicySHA = &pk, addPolicy(o)
				}
			}
		}
	case domain.EngineRulesAuto:
		attr[domain.FacetCreditOwner] = snapshot(c.HolderKey, "wallet transaction user_id written at credit time")
		legacy.SnapshotKey = c.HolderKey
		in.FunctionID, in.BaseKind = domain.FnRuleInvoicePercentFixedV1, domain.BaseInvoiceTotal
		in.Base, legacy.Base = exact(c.BaseRaw), exact(c.BaseRaw)
		switch {
		case strings.HasPrefix(c.TypeRaw, "supervisor_commission"):
			legacy.Comparable, legacy.NotComparableBy = false, "E2 supervisor share is option-based (supervisor commission settings): no staged rate evidence"
		case len(c.MatchedRuleKeys) != 1:
			legacy.Comparable, legacy.NotComparableBy = false, "E2 credit stacks several matched rules (cumulative): per-rule components not staged"
		default:
			o, ok := rulePolicy[c.MatchedRuleKeys[0]]
			if !ok || o.Policy.Evidence == domain.EvidenceInvalid {
				legacy.Comparable, legacy.NotComparableBy = false, "matched rule not staged or invalid"
				break
			}
			in.Rate, in.Fixed = o.Policy.Rate, o.Policy.Fixed
			legacy.RateSnapshot = o.Policy.Rate
			pk := o.Policy.PolicyKey
			row.PolicyKey, row.PolicySHA = &pk, o.SHA
		}
	case domain.EngineProjectCard:
		attr[domain.FacetCreditOwner] = snapshot(c.HolderKey, "wallet transaction user_id written at credit time")
		in.FunctionID, in.BaseKind = domain.FnComponentProRata, domain.BaseComponentLine
		legacy.Comparable, legacy.NotComparableBy = false, "D-36: component (membership item) allocation unsupported"
		d.Findings = append(d.Findings, Finding{Kind: "COMPONENT_ALLOCATION_BLOCKED", Severity: "INFO", SourceKey: c.TxKey,
			Detail: "Biawin project-card commission is component-based (D-36): not comparable"})
	default:
		attr[domain.FacetCreditOwner] = snapshot(c.HolderKey, "wallet transaction user_id written at credit time")
		d.Findings = append(d.Findings, Finding{Kind: "ENGINE_ROUTING_UNKNOWN", Severity: "BLOCKING", SourceKey: c.TxKey, Detail: c.EngineBasis})
	}
	in.Attribution = attr
	row.Attribution = attr
	if v := attr[domain.FacetCreditOwner]; v.UserKey != nil && c.HolderKey != nil && *v.UserKey == *c.HolderKey {
		row.RecipientUser = c.HolderUserID
	}
	in.PolicyKey = deref(row.PolicyKey)
	row.Input = in
	row.legacy = legacy
	row.dup = dup
	return row
}

func itemCalc(it api.DryRunItem, rounding domain.Rounding, rules map[string]api.Rule, rulePolicy map[string]PolicyObs) CalcRow {
	runKey := deref(it.RunKey)
	id := domain.EntitlementIdentity{Kind: domain.KindInvoice, InvoiceKey: deref(it.InvoiceKey), Purpose: purposeInvoiceRules, ShareFacet: "seller",
		Context: "DRY_RUN:" + runKey}
	if it.NonInvoice {
		id = domain.EntitlementIdentity{Kind: domain.KindNonInvoice, Purpose: "daily_attendance", ShareFacet: "seller", Context: "DRY_RUN:" + runKey}
	}
	attr := domain.Attribution{domain.FacetOriginalSeller: snapshot(it.SellerKey, "dry-run item seller_user_id at run time")}
	row := CalcRow{SourceKey: "item:" + it.SourceKey, Engine: domain.EngineDryRunPosting, Identity: id, Attribution: attr,
		LegacyAmount: exact(it.CalculatedRaw), RecipientUser: it.SellerUserID}
	in := domain.Input{Identity: id, FunctionID: domain.FnRuleInvoicePercentFixedV1, BaseKind: domain.BaseInvoiceTotal, Base: exact(it.InvoiceAmountRaw),
		Unit: money.IRT, Rounding: rounding, Attribution: attr, RecipientFacet: domain.FacetOriginalSeller, Engine: domain.EngineDryRunPosting,
		EventRef: "legacy.commission_dry_run_item.v1:" + it.SourceKey}
	legacy := domain.Legacy{SourceKey: row.SourceKey, Engine: domain.EngineDryRunPosting, Amount: row.LegacyAmount, AmountRaw: it.CalculatedRaw,
		RecipientKey: it.SellerKey, SnapshotKey: it.SellerKey, Base: in.Base, Comparable: true}
	switch {
	case it.NonInvoice:
		in.FunctionID = domain.FnDailyAttendance
		legacy.Comparable, legacy.NotComparableBy = false, "non-invoice entitlement (OPD-09)"
	case it.MatchedRuleKey == nil:
		legacy.Comparable, legacy.NotComparableBy = false, "legacy item matched no rule"
	default:
		o, ok := rulePolicy[*it.MatchedRuleKey]
		if !ok || o.Policy.Evidence == domain.EvidenceInvalid {
			legacy.Comparable, legacy.NotComparableBy = false, "matched rule not staged or invalid"
			break
		}
		in.Rate, in.Fixed = o.Policy.Rate, o.Policy.Fixed
		legacy.RateSnapshot = o.Policy.Rate
		pk := o.Policy.PolicyKey
		row.PolicyKey, row.PolicySHA = &pk, o.SHA
		in.PolicyKey = pk
	}
	row.Input = in
	row.legacy = legacy
	return row
}

// Finish computes every output and parity row given the resolved policy
// versions (policy key + content sha -> version).
func (d *Derived) Finish(versions map[string]int) {
	for i := range d.Calcs {
		c := &d.Calcs[i]
		if c.PolicyKey != nil {
			c.Input.PolicyVer = versions[*c.PolicyKey+"\x1f"+string(c.PolicySHA)]
		}
		c.Output = domain.Calculate(c.Input)
		c.legacy.EntitlementKey = c.Output.EntitlementKey
		c.Parity = domain.Compare(c.legacy, &c.Output, c.dup)
		if c.Engine == domain.EngineDryRunPosting && c.Parity.Class == domain.ParityAmountMismatch {
			c.Parity.Reasons = append(c.Parity.Reasons, "RULE_OBSERVED_AT_SNAPSHOT_NOT_EVENT_TIME_OR_STACKED")
		}
	}
	d.Summary = summarize(d.Calcs)
}

func summarize(calcs []CalcRow) api.ParitySummary {
	s := api.ParitySummary{Cohort: "legacy commission credits (E1 / E2 / E4 / unproven) and E3 dry-run items of one staged snapshot",
		Classes: map[string]int64{}, Engines: map[string]int64{}, Reasons: map[string]int64{}, LegacyTotals: map[string]string{}, TargetTotals: map[string]string{}}
	for _, c := range domain.ParityClasses() {
		s.Classes[string(c)] = 0
	}
	var lt, tt domain.Totals
	for _, c := range calcs {
		s.LegacyRows++
		s.Classes[string(c.Parity.Class)]++
		s.Engines[string(c.Engine)]++
		for _, r := range c.Parity.Reasons {
			s.Reasons[r]++
		}
		switch c.Parity.Class {
		case domain.ParityMatch, domain.ParityAmountMismatch, domain.ParityRecipientMismatch, domain.ParityPolicyMismatch, domain.ParityDuplicateRisk:
			s.ComparedRows++
		}
		_ = lt.Add(c.LegacyAmount)
		_ = tt.Add(c.Output.Amount)
	}
	s.LegacyTotals, s.TargetTotals, s.LegacyMissing, s.TargetMissing = lt.Decimal(), tt.Decimal(), lt.Missing, tt.Missing
	switch {
	case s.LegacyRows > 0 && s.ComparedRows == s.LegacyRows:
		s.Coverage = "FULL"
	case s.ComparedRows == 0:
		s.Coverage = "NONE"
	default:
		s.Coverage = "PARTIAL"
	}
	return s
}

// Window returns the raw min / max occurrence of the credits.
func Window(ev api.Evidence) (from, to *string) {
	for _, c := range ev.Credits {
		if c.CreatedAtRaw == nil || *c.CreatedAtRaw == "" {
			continue
		}
		v := *c.CreatedAtRaw
		if from == nil || v < *from {
			from = &v
		}
		if to == nil || v > *to {
			v2 := v
			to = &v2
		}
	}
	return from, to
}
