package shadow

// Slice 6 sales read connector. It maps staged legacy lead, lead status
// history, distribution item and distribution item log rows into the neutral
// cases and custody import commands. All legacy knowledge (table semantics,
// status vocabularies, action meanings) lives here; the cases and custody
// modules never see a legacy name.
//
// Mappings are explicit and evidence-backed (plugin/current, 2.0.160). An
// action or status outside them is UNKNOWN with the raw value preserved:
// nothing is defaulted, guessed or synthesised.

import (
	"context"
	"sort"
	"strings"

	casesapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/api"
	custodyapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

// Spec ids consumed by Slice 6.
const (
	SpecLead            = "legacy.lead.v1"
	SpecLeadHistory     = "legacy.lead_status_history.v1"
	SpecDistItem        = "legacy.distribution_item.v1"
	SpecDistItemLog     = "legacy.distribution_item_log.v1"
	leadStateNamespace  = "legacy_lead.status"
	itemStateNamespace  = "distribution_item.status"
	conversionLinkBasis = "distribution item records the legacy lead id written on conversion (class-sn-plugin.php:21209)"
)

// SalesSpecs lists every staged spec Slice 6 reads.
func SalesSpecs() []string { return []string{SpecLead, SpecLeadHistory, SpecDistItem, SpecDistItemLog} }

// leadStatuses are sn_leads.status values evidenced in 2.0.160:
// 'unassigned' (DDL default, class-sn-activator.php:244), 'assigned'
// (class-sn-plugin.php:30914), 'supervisor_pool' (:5647), 'invoiced' (:5705).
var leadStatuses = map[string]bool{"unassigned": true, "assigned": true, "supervisor_pool": true, "invoiced": true}

// itemStatuses are sn_distribution_items.status values evidenced in 2.0.160:
// the DDL default (class-sn-migration-service.php:988), the status label
// dictionary (class-sn-plugin.php:9429-9456) and direct writers
// (:6413 needs_payment_assignment, :6442 payment_completed, :28514
// payment_assigned, :21618 referral_cancelled).
var itemStatuses = map[string]bool{"ready_for_distribution": true, "assigned_to_role": true, "needs_repeat_action": true,
	"pending_supervisor_action": true, "converted_to_dot_case": true, "distributed_forward": true, "delivered_to_seller": true,
	"returned_to_owner": true, "returned_to_manager": true, "returned_to_mis": true, "ready_for_lead_conversion": true,
	"live_lead_created": true, "conversion_failed": true, "duplicate_lead_detected": true, "conversion_requested": true,
	"cancelled": true, "duplicate": true, "needs_payment_assignment": true, "payment_completed": true, "payment_assigned": true,
	"referral_cancelled": true}

// actionClass is the verified custody meaning of one legacy log action.
type actionClass struct{ kind, context string }

// logActions maps sn_distribution_item_logs.action values whose holder
// effect was verified at their writer in class-sn-plugin.php:
//   - HOLDER_ASSIGNED: the item's current_owner_user_id is set to the log's
//     to_user_id at the writer;
//   - HOLDER_RELEASED: the item's holder is cleared (returned to MIS);
//   - NO_HOLDER_CHANGE: the writer changes status/workflow only (the log's
//     to_user_id is NOT the new holder, e.g. conversion logs carry the
//     final seller).
//
// Anything else (including returned_to_owner / returned_to_manager, whose
// writers were not verified) is UNKNOWN.
var logActions = map[string]actionClass{
	"item_created_from_mis_pool":         {custodyapi.HolderAssigned, "MIS"},              // :20489
	"distributed_forward":                {custodyapi.HolderAssigned, "DISTRIBUTION"},     // :20837-20853
	"delivered_to_seller":                {custodyapi.HolderAssigned, "DISTRIBUTION"},     // :20837-20853
	"mis_direct_delivery_to_seller":      {custodyapi.HolderAssigned, "MIS"},              // :22752
	"mis_quick_assignment_to_role":       {custodyapi.HolderAssigned, "MIS"},              // :22752
	"invoice_payment_action_created":     {custodyapi.HolderAssigned, "PAYMENT_WORKFLOW"}, // :6408-6437
	"invoice_payment_sent_to_supervisor": {custodyapi.HolderAssigned, "PAYMENT_WORKFLOW"}, // :28682-28687
	"invoice_payment_assigned":           {custodyapi.HolderAssigned, "PAYMENT_WORKFLOW"}, // :28682-28694
	"paid_referral_routed":               {custodyapi.HolderAssigned, "REFERRAL"},         // :28692-28694
	"paid_invoice_referral_created":      {custodyapi.HolderAssigned, "REFERRAL"},         // :29105-29127
	"paid_invoice_repeat_action_created": {custodyapi.HolderAssigned, "REFERRAL"},         // :29105-29127
	"returned_to_mis":                    {custodyapi.HolderReleased, "MIS"},              // :22484-22500
	"seller_item_updated":                {custodyapi.NoHolderChange, "SELLER_UPDATE"},    // :14480
	"invoice_payment_stage_issued":       {custodyapi.NoHolderChange, "PAYMENT_WORKFLOW"}, // :28859
	"invoice_payment_archived":           {custodyapi.NoHolderChange, "PAYMENT_WORKFLOW"}, // :29229
	"payment_completed":                  {custodyapi.NoHolderChange, "PAYMENT_WORKFLOW"}, // :6442-6451 (status close)
	"duplicate_blocked_before_transfer":  {custodyapi.NoHolderChange, "DISTRIBUTION"},     // :21527
	"paid_referral_cancelled_by_seller":  {custodyapi.NoHolderChange, "REFERRAL"},         // :38048
	"paid_referral_restored":             {custodyapi.NoHolderChange, "REFERRAL"},         // :38064
	"live_lead_created":                  {custodyapi.NoHolderChange, "CONVERSION"},       // :21195-21218
	"conversion_failed":                  {custodyapi.NoHolderChange, "CONVERSION"},       // :21195-21218
	"duplicate_lead_detected":            {custodyapi.NoHolderChange, "CONVERSION"},       // :21195-21218
}

// ClassifyAction returns the verified kind/context of a legacy action, or
// UNKNOWN for anything not in the table.
func ClassifyAction(raw string) (kind, context string) {
	if c, ok := logActions[raw]; ok {
		return c.kind, c.context
	}
	return custodyapi.KindUnknown, "UNKNOWN"
}

func stateOf(raw *string, known map[string]bool) casesapi.StateMapping {
	if raw != nil && known[*raw] {
		return casesapi.StateKnown
	}
	return casesapi.StateUnknown
}

// Users resolves legacy user keys to identity user ids (published identity
// API). A nil map means no resolution: references stay legacy-only.
type Users map[string]string

func (u Users) casesRef(raw *string) *casesapi.UserRef {
	k, _ := key(raw)
	if k == nil {
		return nil
	}
	r := &casesapi.UserRef{LegacyKey: *k}
	if id, ok := u[*k]; ok {
		r.UserID = &id
	}
	return r
}

func (u Users) custodyRef(raw *string) *custodyapi.UserRef {
	k, _ := key(raw)
	if k == nil {
		return nil
	}
	r := &custodyapi.UserRef{LegacyKey: *k}
	if id, ok := u[*k]; ok {
		r.UserID = &id
	}
	return r
}

// UserKeys lists every legacy user key the Slice 6 rows reference.
func UserKeys(b *Batch) []string {
	set := map[string]bool{}
	add := func(raw *string) {
		if k, _ := key(raw); k != nil {
			set[*k] = true
		}
	}
	for _, r := range b.Entity(SpecLead).Rows {
		add(r.Get("seller_id"))
		add(r.Get("supervisor_id"))
	}
	for _, r := range b.Entity(SpecLeadHistory).Rows {
		add(r.Get("user_id"))
	}
	for _, r := range b.Entity(SpecDistItem).Rows {
		add(r.Get("current_owner_user_id"))
	}
	for _, r := range b.Entity(SpecDistItemLog).Rows {
		add(r.Get("actor_user_id"))
		add(r.Get("from_user_id"))
		add(r.Get("to_user_id"))
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func casesSrc(b *Batch, spec string, r Row) casesapi.SourceRef {
	return casesapi.SourceRef{System: SourceSystem, Spec: spec, Key: r.Key, BatchID: b.BatchID, RowSHA256: r.SHA}
}

// CasesCommand builds the cases import command. Attribution evidence for
// distribution items is taken from their FIRST holder-assigning log rows
// (original owner) and first delivery to a seller position (original
// seller); nothing reads the current organisation.
func CasesCommand(b *Batch, users Users) (casesapi.CaseImport, map[string]int64) {
	counts := map[string]int64{}
	leads, hist, items, logs := b.Entity(SpecLead), b.Entity(SpecLeadHistory), b.Entity(SpecDistItem), b.Entity(SpecDistItemLog)
	in := casesapi.CaseImport{BatchID: b.BatchID, SnapshotID: b.SnapshotID, SourceSystem: SourceSystem, Complete: map[string]bool{}}
	if leads.Usable() {
		in.Complete[SpecLead] = leads.Complete()
	}
	if items.Usable() {
		in.Complete[SpecDistItem] = items.Complete()
	}
	staged := map[string]bool{}
	for _, r := range leads.Rows {
		staged[r.Key] = true
		status := r.Get("status")
		ch, origin := r.Get("marketing_form_id"), r.Get("import_code")
		in.Cases = append(in.Cases, casesapi.CaseRecord{Workflow: casesapi.LegacyLead, Source: casesSrc(b, SpecLead, r),
			ChannelRaw: ch, OriginRefRaw: origin, StateNamespace: leadStateNamespace, StateRaw: status, StateMapping: stateOf(status, leadStatuses),
			Lead: &casesapi.LeadFacts{StatusRaw: status, LeadStatusRaw: r.Get("lead_status"), DestinationPanelRaw: r.Get("destination_panel"),
				ImportedAtRaw: r.Get("imported_at"), AssignedAtRaw: r.Get("assigned_at"), UpdatedAtRaw: r.Get("updated_at")}})
		// No lead assignment log exists in 2.0.160: the original owner and
		// seller of a legacy lead are not recoverable from the source.
		in.Attributions = append(in.Attributions,
			casesapi.AttributionRecord{Case: casesapi.RowRef{Spec: SpecLead, Key: r.Key}, Facet: casesapi.OriginalOwner, Resolution: casesapi.NotRecoverable,
				Basis: "legacy leads keep no assignment history; only the current holder columns exist"},
			casesapi.AttributionRecord{Case: casesapi.RowRef{Spec: SpecLead, Key: r.Key}, Facet: casesapi.OriginalSeller, Resolution: casesapi.NotRecoverable,
				Basis: "legacy leads keep no assignment history; the current seller is not the original seller"})
	}
	for _, r := range hist.Rows {
		lk, _ := key(r.Get("lead_id"))
		if lk == nil {
			counts["history_without_lead"]++
			continue
		}
		in.History = append(in.History, casesapi.HistoryRecord{Source: casesSrc(b, SpecLeadHistory, r), Case: casesapi.RowRef{Spec: SpecLead, Key: *lk},
			OldRaw: r.Get("old_status"), NewRaw: r.Get("new_status"), Actor: users.casesRef(r.Get("user_id")), OccurredRaw: r.Get("created_at")})
	}
	logsByItem := map[string][]Row{}
	for _, r := range logs.Rows {
		if ik, _ := key(r.Get("item_id")); ik != nil {
			logsByItem[*ik] = append(logsByItem[*ik], r)
		}
	}
	for _, r := range items.Rows {
		status := r.Get("status")
		in.Cases = append(in.Cases, casesapi.CaseRecord{Workflow: casesapi.DistributionItem, Source: casesSrc(b, SpecDistItem, r),
			ChannelRaw: r.Get("source_type"), OriginRefRaw: r.Get("source_id"), StateNamespace: itemStateNamespace, StateRaw: status,
			StateMapping: stateOf(status, itemStatuses)})
		if lk, bad := key(r.Get("live_lead_id")); lk != nil {
			in.Aliases = append(in.Aliases, casesapi.AliasRecord{From: casesapi.RowRef{Spec: SpecDistItem, Key: r.Key},
				Related: casesapi.RowRef{Spec: SpecLead, Key: *lk}, Relation: casesapi.ConversionLink, Basis: conversionLinkBasis})
			if !staged[*lk] {
				counts["conversion_link_to_unstaged_lead"]++
			}
		} else if bad {
			counts["conversion_link_unreadable"]++
		}
		in.Attributions = append(in.Attributions, itemAttribution(r.Key, logsByItem[r.Key], coverageOf(logs), users)...)
	}
	return in, counts
}

// logCoverage is how much of the distribution log source the batch holds.
type logCoverage int

const (
	logsNotStaged logCoverage = iota
	logsIncomplete
	logsComplete
)

func coverageOf(e Entity) logCoverage {
	switch {
	case e.Complete():
		return logsComplete
	case e.Usable():
		return logsIncomplete
	}
	return logsNotStaged
}

// itemAttribution freezes Original Owner / Original Seller from the item's
// own log evidence. A facet is RESOLVED only when the log source is
// COMPLETE and no unmapped action precedes the chosen row: otherwise an
// earlier (missing or unclassified) assignment could be the real origin,
// so the facet is UNKNOWN. NOT_RECOVERABLE only when complete logs prove
// there is no qualifying row. Credit Owner is never evaluated here.
func itemAttribution(item string, logs []Row, cov logCoverage, users Users) []casesapi.AttributionRecord {
	ref := casesapi.RowRef{Spec: SpecDistItem, Key: item}
	sort.SliceStable(logs, func(i, j int) bool { return lessKey(logs[i].Key, logs[j].Key) })
	var owner, seller *Row
	unmappedBefore := map[*Row]bool{}
	sawUnknown := false
	for i := range logs {
		r := &logs[i]
		kind, _ := ClassifyAction(textOr(r.Get("action")))
		if kind == custodyapi.KindUnknown {
			sawUnknown = true
			continue
		}
		if kind != custodyapi.HolderAssigned {
			continue
		}
		if to, _ := key(r.Get("to_user_id")); to == nil {
			continue
		}
		if owner == nil {
			owner, unmappedBefore[r] = r, sawUnknown
		}
		if seller == nil && strings.EqualFold(textOr(r.Get("to_position")), "seller") {
			seller, unmappedBefore[r] = r, sawUnknown
		}
	}
	snap := func(facet casesapi.Facet, r *Row, basis string) casesapi.AttributionRecord {
		switch {
		case cov == logsNotStaged:
			return casesapi.AttributionRecord{Case: ref, Facet: facet, Resolution: casesapi.Unknown, Basis: "distribution item logs are not staged"}
		case cov == logsIncomplete:
			return casesapi.AttributionRecord{Case: ref, Facet: facet, Resolution: casesapi.Unknown,
				Basis: "distribution item logs are not COMPLETE; an earlier assignment may be missing"}
		case r == nil && sawUnknown:
			return casesapi.AttributionRecord{Case: ref, Facet: facet, Resolution: casesapi.Unknown,
				Basis: "no mapped holder-assigning row; unmapped legacy actions may hide one"}
		case r == nil:
			return casesapi.AttributionRecord{Case: ref, Facet: facet, Resolution: casesapi.NotRecoverable,
				Basis: "complete logs hold no holder-assigning row for this item"}
		case unmappedBefore[r]:
			return casesapi.AttributionRecord{Case: ref, Facet: facet, Resolution: casesapi.Unknown,
				Basis: "an unmapped legacy action precedes the first mapped assignment"}
		}
		subject := users.casesRef(r.Get("to_user_id"))
		return casesapi.AttributionRecord{Case: ref, Facet: facet, Resolution: casesapi.Resolved, Subject: subject, PositionRaw: r.Get("to_position"),
			Basis: basis, SourceTimeRaw: r.Get("created_at"), Evidence: &casesapi.RowRef{Spec: SpecDistItemLog, Key: r.Key}}
	}
	return []casesapi.AttributionRecord{
		snap(casesapi.OriginalOwner, owner, "first holder-assigning custody log row of the item (complete logs)"),
		snap(casesapi.OriginalSeller, seller, "first holder-assigning custody log row to a seller position (complete logs)"),
		{Case: ref, Facet: casesapi.CreditOwner, Resolution: casesapi.Unknown, Basis: "credit ownership needs financial evidence; not evaluated in Slice 6"},
	}
}

func textOr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func lessKey(a, b string) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return a < b
}

// CustodyCommand builds the custody import command: one event per staged
// log row and one holder record per staged lead and item (source state).
func CustodyCommand(b *Batch, users Users) (custodyapi.CustodyImport, map[string]int64) {
	counts := map[string]int64{}
	leads, items, logs := b.Entity(SpecLead), b.Entity(SpecDistItem), b.Entity(SpecDistItemLog)
	in := custodyapi.CustodyImport{BatchID: b.BatchID, SnapshotID: b.SnapshotID, SourceSystem: SourceSystem, Complete: map[string]bool{}}
	if leads.Usable() {
		in.Complete[SpecLead] = leads.Complete()
	}
	if items.Usable() {
		in.Complete[SpecDistItem] = items.Complete()
	}
	for _, r := range leads.Rows {
		in.Holders = append(in.Holders, custodyapi.HolderRecord{Case: custodyapi.RowRef{Spec: SpecLead, Key: r.Key}, Holder: users.custodyRef(r.Get("seller_id"))})
	}
	for _, r := range items.Rows {
		in.Holders = append(in.Holders, custodyapi.HolderRecord{Case: custodyapi.RowRef{Spec: SpecDistItem, Key: r.Key},
			Holder: users.custodyRef(r.Get("current_owner_user_id")), PositionRaw: r.Get("current_owner_position")})
	}
	for _, r := range logs.Rows {
		ik, _ := key(r.Get("item_id"))
		action := textOr(r.Get("action"))
		if ik == nil || action == "" || len(action) > 100 {
			counts["log_rows_unusable"]++
			continue
		}
		kind, ctx := ClassifyAction(action)
		if kind == custodyapi.KindUnknown {
			counts["log_actions_unknown"]++
		}
		in.Events = append(in.Events, custodyapi.EventRecord{
			Source: custodyapi.SourceRef{System: SourceSystem, Spec: SpecDistItemLog, Key: r.Key, BatchID: b.BatchID, RowSHA256: r.SHA},
			Item:   custodyapi.RowRef{Spec: SpecDistItem, Key: *ik}, Action: action, Kind: kind, Context: ctx,
			Actor: users.custodyRef(r.Get("actor_user_id")), From: users.custodyRef(r.Get("from_user_id")), To: users.custodyRef(r.Get("to_user_id")),
			FromPositionRaw: r.Get("from_position"), ToPositionRaw: r.Get("to_position"), StepOrderRaw: r.Get("step_order"), OccurredRaw: r.Get("created_at")})
	}
	return in, counts
}

// SalesReport is the Slice 6 reconciliation report (JSON).
type SalesReport struct {
	Command       string             `json:"command"`
	SnapshotID    string             `json:"snapshot_id"`
	BatchID       string             `json:"batch_id"`
	Outcome       result.Status      `json:"outcome"`
	Reasons       []reasoncodes.Code `json:"reasons"`
	ConsumedSpecs []string           `json:"consumed_specs"`
	Staged        map[string]int64   `json:"staged"`
	Cases         casesapi.Coverage  `json:"cases"`
	Custody       custodyapi.Parity  `json:"custody"`
	Coverage      map[string]string  `json:"source_coverage"`
	Invariants    map[string]bool    `json:"invariants"`
	Notes         []string           `json:"notes"`
}

// ReconcileSales reports cases coverage and custody parity against the
// staged batch. It reads only; it never claims parity on partial evidence.
func ReconcileSales(ctx context.Context, b *Batch, cases casesapi.Reader, custody custodyapi.Reader) (SalesReport, error) {
	rep := SalesReport{Command: "reconcile-sales", SnapshotID: b.SnapshotID, BatchID: b.BatchID, Reasons: []reasoncodes.Code{},
		ConsumedSpecs: SalesSpecs(), Staged: map[string]int64{}, Coverage: map[string]string{}}
	for _, sp := range SalesSpecs() {
		e := b.Entity(sp)
		rep.Staged[sp] = int64(len(e.Rows))
		rep.Coverage[sp] = coverage(e)
	}
	rep.Coverage["legacy.seller_flow (not staged)"] = "UNKNOWN"
	rep.Coverage["legacy.dot_cases (not staged)"] = "UNKNOWN"
	rep.Coverage["legacy.mis_distribution (not staged)"] = "UNKNOWN"
	var err error
	if rep.Cases, err = cases.CoverageReport(ctx); err != nil {
		return rep, err
	}
	if rep.Custody, err = custody.ParityReport(ctx); err != nil {
		return rep, err
	}
	// Per-row provenance: every staged lead/item must be a present Case
	// last carried by THIS batch with the same checksum.
	matched, missing, other := int64(0), int64(0), int64(0)
	var stagedCases []string
	for _, sp := range []string{SpecLead, SpecDistItem} {
		e := b.Entity(sp)
		keys := make([]string, 0, len(e.Rows))
		for _, r := range e.Rows {
			keys = append(keys, r.Key)
		}
		m, err := cases.ResolveSources(ctx, SourceSystem, sp, keys)
		if err != nil {
			return rep, err
		}
		for _, r := range e.Rows {
			sc, ok := m[r.Key]
			switch {
			case !ok || !sc.Present:
				missing++
			case sc.BatchID != b.BatchID || string(sc.RowSHA256) != string(r.SHA):
				other++
			default:
				matched++
			}
			if ok {
				stagedCases = append(stagedCases, sc.CaseID)
			}
		}
	}
	// Custody coverage per staged Case: aggregate parity over existing
	// projections cannot prove that THIS batch's Cases have custody.
	covered, err := custody.CaseCoverage(ctx, stagedCases)
	if err != nil {
		return rep, err
	}
	custodyRun, err := custody.BatchImported(ctx, b.BatchID)
	if err != nil {
		return rep, err
	}
	noCustody, notMatch := int64(0), int64(0)
	for _, id := range stagedCases {
		switch p, ok := covered[id]; {
		case !ok:
			noCustody++
		case p != "MATCH":
			notMatch++
		}
	}
	rep.Staged["cases_without_custody_projection"] = noCustody
	rep.Staged["cases_custody_not_match"] = notMatch
	rep.Staged["cases_matched_to_batch"] = matched
	rep.Staged["cases_missing"] = missing
	rep.Staged["cases_from_other_batch_or_changed"] = other
	rep.Invariants = map[string]bool{
		"no_sourceless_case":            rep.Cases.SourcelessCases == 0,
		"no_resolved_alias":             rep.Cases.AliasesByResolution["RESOLVED"] == 0,
		"no_resolved_customer_link":     rep.Cases.CustomerLinks["RESOLVED"] == 0,
		"full_parity_only_if_all_match": !rep.Custody.FullParity || rep.Custody.ByParity["MATCH"] == rep.Custody.Assignments,
	}
	rep.Notes = []string{
		"one staged lead or distribution item = one Case; explicit conversion links are aliases, never merges (OPD-01 open)",
		"Legacy Leads Only (F03) is reported separately from Total Cases; F02 (Open Pre-invoices) is NOT_EVALUATED: no invoice state in Slice 6",
		"custody parity compares legacy holder columns with holders derived from append-only log rows; lead custody has no legacy log",
		"no phone, name or username is staged, stored or used as a key",
	}
	if !custodyRun {
		rep.Notes = append(rep.Notes, "no custody import run exists for this batch")
	}
	violated := false
	for _, ok := range rep.Invariants {
		violated = violated || !ok
	}
	switch {
	case violated:
		rep.Outcome = result.Conflict
		rep.Reasons = append(rep.Reasons, reasoncodes.LinkageConflict)
	case !b.Entity(SpecLead).Usable() || !b.Entity(SpecDistItem).Usable():
		rep.Outcome = result.Unknown
		rep.Reasons = append(rep.Reasons, reasoncodes.SourceUnavailable)
	case missing > 0 || other > 0 || noCustody > 0 || notMatch > 0 || !custodyRun || !rep.Custody.FullParity:
		// Partial evidence or imperfect parity is reported, never hidden.
		rep.Outcome = result.Partial
		rep.Reasons = append(rep.Reasons, reasoncodes.CoverageIncomplete)
	default:
		rep.Outcome = result.Success
	}
	return rep, nil
}
