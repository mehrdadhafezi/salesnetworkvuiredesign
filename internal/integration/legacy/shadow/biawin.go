package shadow

// Slice 10 Biawin / fulfillment READ connector. It maps staged legacy
// membership, item, event, invoice-link, action, execution, form-submission
// and shipping rows into the neutral biawin and fulfillment import records.
// All legacy knowledge (table / column names, status vocabularies, the
// messaging event keys, the provider reference fields) lives here.
//
// Evidence (plugin/current 2.0.160):
//   - Membership status: 'active' on insert (class-sn-projects.php:952,
//     :1093) and on reactivation of a 'suspended' row (:1059-1069, which
//     also OVERWRITES activated_at); 'suspended' on financial reversal
//     (:1241). Nothing else is written: any other raw value is UNKNOWN.
//     Active is never derived from payment, invoice or item presence.
//   - Item follow-up workflow_status: waiting / no_answer / follow_up /
//     cancelled / normal / upsell / purchased / regretted
//     (class-sn-projects.php:2351, labels :2143).
//   - Execution case status and labels: class-sn-operations-execution.php:595;
//     final_decision executed / not_executed (:545).
//   - Upgrade action status: awaiting_payment (default), paid,
//     remote_pending, remote_failed, fulfilled, retry_requested,
//     cancelled_continue_normal (class-sn-projects.php actions writers).
//   - Shipping fulfillment_status: class-sn-product-flow.php:234.
//   - Project event keys chat_message_sent / internal_note_added are
//     messaging (owned by cases): classified MESSAGING_ONLY, not imported.
//
// Customers are linked ONLY through the source invoice's Slice 5
// resolution (billing published API): never by phone, name or account id.

import (
	"crypto/sha256"
	"sort"

	biawinapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/api"
	billingapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/api"
	fulfillmentapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/api"
)

// Spec ids consumed by Slice 10.
const (
	SpecMembership       = "legacy.project_membership.v1"
	SpecMembershipItem   = "legacy.project_membership_item.v1"
	SpecProjectEvent     = "legacy.project_event.v1"
	SpecItemInvoiceLink  = "legacy.project_invoice_link.v1"
	SpecProjectAction    = "legacy.project_action.v1"
	SpecExecutionCase    = "legacy.operations_execution_case.v1"
	SpecExecutionEvent   = "legacy.operations_execution_event.v1"
	SpecFormSubmission   = "legacy.operations_form_submission.v1"
	SpecShippingOrder    = "legacy.shipping_order.v1"
	SpecShippingOrderLog = "legacy.shipping_order_log.v1"
)

// BiawinSpecs lists every staged spec Slice 10 reads.
func BiawinSpecs() []string {
	return []string{SpecMembership, SpecMembershipItem, SpecProjectEvent, SpecItemInvoiceLink, SpecProjectAction, SpecExecutionCase,
		SpecExecutionEvent, SpecFormSubmission, SpecShippingOrder, SpecShippingOrderLog}
}

var (
	subscriptionStates = map[string]string{"active": biawinapi.StateActive, "suspended": biawinapi.StateSuspended}
	followupStates     = map[string]string{"waiting": fulfillmentapi.StatusNotStarted, "no_answer": fulfillmentapi.StatusInProgress,
		"follow_up": fulfillmentapi.StatusInProgress, "cancelled": fulfillmentapi.StatusCancelled, "normal": fulfillmentapi.StatusOutcomeRecorded,
		"upsell": fulfillmentapi.StatusOutcomeRecorded, "purchased": fulfillmentapi.StatusOutcomeRecorded, "regretted": fulfillmentapi.StatusOutcomeRecorded}
	executionStates = map[string]string{"manager_queue": fulfillmentapi.StatusNotStarted, "assigned": fulfillmentapi.StatusInProgress,
		"no_answer": fulfillmentapi.StatusInProgress, "follow_up": fulfillmentapi.StatusInProgress, "wallet_charged": fulfillmentapi.StatusProviderRecorded,
		"shipping_queue": fulfillmentapi.StatusInProgress, "shipping_processing": fulfillmentapi.StatusInProgress, "shipped": fulfillmentapi.StatusShipped,
		"delivered": fulfillmentapi.StatusDelivered, "shipping_not_sent": fulfillmentapi.StatusNotExecuted, "shipping_returned": fulfillmentapi.StatusReturned,
		"shipping_cancelled": fulfillmentapi.StatusCancelled, "form_completed": fulfillmentapi.StatusOutcomeRecorded, "completed": fulfillmentapi.StatusCompleted,
		"not_executed": fulfillmentapi.StatusNotExecuted, "cancelled": fulfillmentapi.StatusCancelled}
	actionStates = map[string]string{"awaiting_payment": fulfillmentapi.StatusAwaitingPayment, "paid": fulfillmentapi.StatusInProgress,
		"remote_pending": fulfillmentapi.StatusAwaitingProvider, "remote_failed": fulfillmentapi.StatusFailedAtProvider, "fulfilled": fulfillmentapi.StatusCompleted,
		"retry_requested": fulfillmentapi.StatusInProgress, "cancelled_continue_normal": fulfillmentapi.StatusCancelled}
	shippingStates = map[string]string{"awaiting_address": fulfillmentapi.StatusNotStarted, "awaiting_review": fulfillmentapi.StatusNotStarted,
		"preparing": fulfillmentapi.StatusInProgress, "shipped": fulfillmentapi.StatusShipped, "delivered": fulfillmentapi.StatusDelivered,
		"not_sent": fulfillmentapi.StatusNotExecuted, "returned": fulfillmentapi.StatusReturned, "cancelled": fulfillmentapi.StatusCancelled}
	outcomes        = map[string]string{"executed": fulfillmentapi.OutcomeExecuted, "not_executed": fulfillmentapi.OutcomeNotExecuted}
	messagingEvents = set("chat_message_sent", "internal_note_added")
)

func mapState(raw *string, m map[string]string, unknown, empty string) string {
	switch {
	case raw == nil || *raw == "":
		return empty
	case m[*raw] != "":
		return m[*raw]
	}
	return unknown
}

func refSHA(raw *string) []byte {
	if raw == nil || *raw == "" {
		return nil
	}
	h := sha256.Sum256([]byte(*raw))
	return h[:]
}

// BiawinUserKeys lists every legacy staff key the Slice 10 rows reference.
func BiawinUserKeys(b *Batch) []string {
	s := map[string]bool{}
	cols := map[string][]string{
		SpecMembership: {"origin_user_id"}, SpecMembershipItem: {"manager_user_id", "expert_user_id"}, SpecProjectEvent: {"actor_user_id"},
		SpecExecutionCase:  {"manager_user_id", "expert_user_id", "completed_by_user_id"},
		SpecExecutionEvent: {"actor_user_id"}, SpecFormSubmission: {"submitted_by_user_id", "decided_by_user_id"},
		SpecShippingOrder: {"handled_by"}, SpecShippingOrderLog: {"actor_user_id"},
	}
	for sp, cs := range cols {
		for _, r := range b.Entity(sp).Rows {
			for _, c := range cs {
				if k, _ := key(r.Get(c)); k != nil {
					s[*k] = true
				}
			}
		}
	}
	return sorted(s)
}

// BiawinInvoiceKeys lists every legacy invoice key the Slice 10 rows name.
func BiawinInvoiceKeys(b *Batch) []string {
	s := map[string]bool{}
	for sp, cs := range map[string][]string{SpecMembership: {"source_invoice_id", "first_payment_invoice_id"}, SpecItemInvoiceLink: {"invoice_id"},
		SpecProjectAction: {"invoice_id"}, SpecShippingOrder: {"invoice_id"}} {
		for _, r := range b.Entity(sp).Rows {
			for _, c := range cs {
				if k, _ := key(r.Get(c)); k != nil {
					s[*k] = true
				}
			}
		}
	}
	return sorted(s)
}

// Invoices indexes billing's published invoice facts by legacy source key.
type Invoices map[string]billingapi.InvoiceFact

func (inv Invoices) link(raw *string) biawinapi.Link {
	k, _ := key(raw)
	if k == nil {
		return biawinapi.Link{State: biawinapi.LinkUnknown, Basis: "no invoice reference"}
	}
	f, ok := inv[*k]
	if !ok || f.Presence != "PRESENT" {
		return biawinapi.Link{State: biawinapi.LinkUnknown, Basis: "legacy invoice not present in billing (published API)", LegacyKey: k}
	}
	id := f.InvoiceID
	return biawinapi.Link{State: biawinapi.LinkResolved, ID: &id, Basis: "legacy invoice resolved through the billing published API", LegacyKey: k}
}

// customer follows the source invoice's Slice 5 resolution; an unresolved
// invoice or customer link stays explicit (never guessed, never by phone).
func (inv Invoices) customer(raw *string) biawinapi.Link {
	k, _ := key(raw)
	if k == nil {
		return biawinapi.Link{State: biawinapi.LinkUnknown, Basis: "no source invoice reference"}
	}
	f, ok := inv[*k]
	if !ok || f.Presence != "PRESENT" {
		return biawinapi.Link{State: biawinapi.LinkUnknown, Basis: "source invoice not present in billing; customer not resolvable"}
	}
	l := biawinapi.Link{State: f.CustomerLink, Basis: "customer link of the source invoice (Slice 5 issued-to subject via billing)"}
	if f.CustomerLink == biawinapi.LinkResolved && f.CustomerID != nil {
		id := *f.CustomerID
		l.ID = &id
	}
	if l.State == "" {
		l.State = biawinapi.LinkUnknown
	}
	return l
}

func (inv Invoices) fLink(raw *string) fulfillmentapi.Link {
	l := inv.link(raw)
	return fulfillmentapi.Link{State: l.State, ID: l.ID, LegacyKey: l.LegacyKey}
}

func (u Users) fRef(raw *string) *fulfillmentapi.UserRef {
	k, _ := key(raw)
	if k == nil {
		return nil
	}
	r := &fulfillmentapi.UserRef{LegacyKey: *k}
	if id, ok := u[*k]; ok {
		r.UserID = &id
	}
	return r
}

func entityCoverage(e Entity) string {
	switch {
	case e.Complete():
		return biawinapi.SourceAvailable
	case e.Usable():
		return biawinapi.SourcePartial
	}
	return biawinapi.SourceMissing
}

// BiawinCommand builds the biawin import and the INDEPENDENT legacy active
// evidence of the same snapshot (raw status per stable source key).
func BiawinCommand(b *Batch, users Users, inv Invoices) (biawinapi.Import, biawinapi.LegacyEvidence) {
	in := biawinapi.Import{BatchID: b.BatchID, SnapshotID: b.SnapshotID, SourceSystem: SourceSystem, Cutoff: b.ExtractedAt.UTC(),
		Sources: map[string]string{}, SourceRows: map[string]int64{}, Skipped: map[string]int64{}}
	for _, sp := range []string{SpecMembership, SpecMembershipItem, SpecProjectEvent, SpecItemInvoiceLink} {
		in.Sources[sp] = entityCoverage(b.Entity(sp))
		in.SourceRows[sp] = int64(len(b.Entity(sp).Rows))
	}
	ev := biawinapi.LegacyEvidence{SnapshotID: b.SnapshotID, Cutoff: in.Cutoff, Status: in.Sources[SpecMembership], Classes: map[string]string{}}
	members := map[string]bool{}
	for _, r := range b.Entity(SpecMembership).Rows {
		members[r.Key] = true
		raw := r.Get("status")
		s := biawinapi.SubscriptionRecord{SourceKey: r.Key, SourceRefRaw: r.Get("source_key"), SourceTypeRaw: r.Get("source_type"),
			ProductKey: r.Get("subscription_product_id"), ProductTitleSnapshot: r.Get("subscription_name_snapshot"),
			Customer: inv.customer(r.Get("source_invoice_id")), CustomerAccountKey: r.Get("customer_wp_id"),
			Invoice: inv.link(r.Get("source_invoice_id")), FirstPaymentInvoice: inv.link(r.Get("first_payment_invoice_id")),
			StateNamespace: "legacy_project_membership.status", StateRaw: raw,
			StateMapped:        mapState(raw, subscriptionStates, biawinapi.StateUnknown, biawinapi.StateEmpty),
			ActivationEvidence: biawinapi.ActivationNone, ActivatedAtRaw: r.Get("activated_at"), CreatedAtRaw: r.Get("created_at"),
			UpdatedAtRaw: r.Get("updated_at")}
		if s.ActivatedAtRaw != nil && *s.ActivatedAtRaw != "" {
			s.ActivationEvidence = biawinapi.ActivationLegacyRecorded
		}
		if k, _ := key(r.Get("dot_case_id")); k != nil {
			s.Case = biawinapi.Link{State: biawinapi.LinkUnknown, LegacyKey: k,
				Basis: "Dot case reference kept raw; Dot cases are not a published Case source (no link manufactured)"}
		} else {
			s.Case = biawinapi.Link{State: biawinapi.LinkUnknown, Basis: "no case reference in the source"}
		}
		if k, _ := key(r.Get("origin_user_id")); k != nil {
			s.OriginUserKey = k
			if id, ok := users[*k]; ok {
				s.OriginUserID = &id
			}
		}
		in.Subscriptions = append(in.Subscriptions, s)
		switch {
		case raw != nil && *raw == "active":
			ev.Classes[r.Key] = biawinapi.LegacyActive
		case raw != nil && *raw == "suspended":
			ev.Classes[r.Key] = biawinapi.LegacyKnownOther
		default:
			ev.Classes[r.Key] = biawinapi.LegacyUnknown
		}
	}
	links := map[string][]biawinapi.ItemInvoiceLink{}
	for _, r := range b.Entity(SpecItemInvoiceLink).Rows {
		k, _ := key(r.Get("membership_item_id"))
		if k == nil {
			in.Skipped["invoice_link_without_item"]++
			continue
		}
		links[*k] = append(links[*k], biawinapi.ItemInvoiceLink{SourceKey: r.Key, Invoice: inv.link(r.Get("invoice_id")), StatusRaw: r.Get("status"),
			VerifiedAtRaw: r.Get("verified_at")})
	}
	items := map[string]string{}
	for _, r := range b.Entity(SpecMembershipItem).Rows {
		mk, _ := key(r.Get("membership_id"))
		if mk == nil || !members[*mk] {
			in.Skipped["item_without_staged_membership"]++
			continue
		}
		items[r.Key] = *mk
		in.Items = append(in.Items, biawinapi.ItemRecord{SourceKey: r.Key, SubscriptionKey: *mk, ServiceKey: r.Get("content_product_id"),
			ServiceTitleSnapshot: r.Get("content_name_snapshot"), InvoiceLinks: links[r.Key], CreatedAtRaw: r.Get("created_at")})
	}
	for _, r := range b.Entity(SpecProjectEvent).Rows {
		ek := r.Get("event_key")
		if ek == nil || *ek == "" {
			in.Skipped["event_without_key"]++
			continue
		}
		if messagingEvents[*ek] {
			in.Skipped["MESSAGING_ONLY"]++ // owned by cases; never merged into fulfillment
			continue
		}
		e := biawinapi.EventRecord{SourceKey: r.Key, EventKeyRaw: *ek, Class: biawinapi.EventSubscription, OccurredAtRaw: r.Get("created_at")}
		if k, _ := key(r.Get("membership_id")); k != nil && members[*k] {
			e.SubscriptionKey = k
		}
		if k, _ := key(r.Get("membership_item_id")); k != nil {
			if mk, ok := items[*k]; ok {
				e.ItemKey, e.Class = k, biawinapi.EventItem
				if e.SubscriptionKey == nil {
					m := mk
					e.SubscriptionKey = &m
				}
			}
		}
		if len(*ek) > len("project_remote_") && (*ek)[:len("project_remote_")] == "project_remote_" {
			e.Class = biawinapi.EventProvider // provider evidence only: nothing is replayed or called
		}
		if e.SubscriptionKey == nil {
			in.Skipped["event_without_staged_membership"]++
			continue
		}
		if k, _ := key(r.Get("actor_user_id")); k != nil {
			e.ActorKey = k
			if id, ok := users[*k]; ok {
				e.ActorUserID = &id
			}
		}
		in.Events = append(in.Events, e)
	}
	return in, ev
}

// FulfillmentCommand builds the fulfillment import.
func FulfillmentCommand(b *Batch, users Users, inv Invoices) fulfillmentapi.Import {
	in := fulfillmentapi.Import{BatchID: b.BatchID, SnapshotID: b.SnapshotID, SourceSystem: SourceSystem, Cutoff: b.ExtractedAt.UTC(),
		Sources: map[string]string{}, SourceRows: map[string]int64{}, Skipped: map[string]int64{}}
	for _, sp := range []string{SpecMembershipItem, SpecProjectAction, SpecExecutionCase, SpecExecutionEvent, SpecFormSubmission, SpecShippingOrder,
		SpecShippingOrderLog} {
		in.Sources[sp] = entityCoverage(b.Entity(sp))
		in.SourceRows[sp] = int64(len(b.Entity(sp).Rows))
	}
	members := map[string]bool{}
	for _, r := range b.Entity(SpecMembership).Rows {
		members[r.Key] = true
	}
	itemSub := map[string]string{}
	for _, r := range b.Entity(SpecMembershipItem).Rows {
		if mk, _ := key(r.Get("membership_id")); mk != nil && members[*mk] {
			itemSub[r.Key] = *mk
		}
	}
	refs := func(c *fulfillmentapi.CaseRecord, itemRaw *string) {
		k, _ := key(itemRaw)
		if k == nil {
			return
		}
		if mk, ok := itemSub[*k]; ok {
			s, i := biawinapi.SubscriptionID(SourceSystem, mk), biawinapi.ItemID(SourceSystem, *k)
			c.SubscriptionID, c.ItemID = &s, &i
		}
	}
	assign := func(role string, raw, at *string) []fulfillmentapi.Assignment {
		if u := users.fRef(raw); u != nil {
			return []fulfillmentapi.Assignment{{Role: role, User: *u, AssignedAtRaw: at}}
		}
		return nil
	}
	noInvoice := fulfillmentapi.Link{State: biawinapi.LinkUnknown}

	for _, r := range b.Entity(SpecMembershipItem).Rows {
		raw := r.Get("workflow_status")
		c := fulfillmentapi.CaseRecord{Kind: fulfillmentapi.KindServiceFollowup, SourceSpec: SpecMembershipItem, SourceKey: r.Key, Invoice: noInvoice,
			StatusNamespace: "legacy_project_membership_item.workflow_status", StatusRaw: raw,
			StatusMapped:       mapState(raw, followupStates, fulfillmentapi.StatusUnknown, fulfillmentapi.StatusEmpty),
			StatusUpdatedAtRaw: r.Get("status_updated_at"), CreatedAtRaw: r.Get("created_at"), UpdatedAtRaw: r.Get("updated_at")}
		refs(&c, &r.Key)
		c.Assignments = append(assign(fulfillmentapi.RoleManager, r.Get("manager_user_id"), nil), assign(fulfillmentapi.RoleExpert, r.Get("expert_user_id"), r.Get("assigned_at"))...)
		in.Cases = append(in.Cases, c)
	}
	for _, r := range b.Entity(SpecProjectAction).Rows {
		raw := r.Get("status")
		c := fulfillmentapi.CaseRecord{Kind: fulfillmentapi.KindUpgradeRequest, SourceSpec: SpecProjectAction, SourceKey: r.Key, SubtypeRaw: r.Get("action_type"),
			Invoice: inv.fLink(r.Get("invoice_id")), StatusNamespace: "legacy_project_action.status", StatusRaw: raw,
			StatusMapped: mapState(raw, actionStates, fulfillmentapi.StatusUnknown, fulfillmentapi.StatusEmpty),
			CreatedAtRaw: r.Get("created_at"), UpdatedAtRaw: r.Get("updated_at")}
		refs(&c, r.Get("membership_item_id"))
		in.Cases = append(in.Cases, c)
	}
	execs := map[string]bool{}
	for _, r := range b.Entity(SpecExecutionCase).Rows {
		execs[r.Key] = true
		raw := r.Get("status")
		c := fulfillmentapi.CaseRecord{Kind: fulfillmentapi.KindExecution, SourceSpec: SpecExecutionCase, SourceKey: r.Key, SubtypeRaw: r.Get("fulfillment_type"),
			Invoice: noInvoice, StatusNamespace: "legacy_operations_execution_case.status", StatusRaw: raw,
			StatusMapped:      mapState(raw, executionStates, fulfillmentapi.StatusUnknown, fulfillmentapi.StatusEmpty),
			ShippingStatusRaw: r.Get("shipping_status"), ProviderStatusRaw: r.Get("wallet_status"), CompletedAtRaw: r.Get("completed_at"),
			CompletedBy: users.fRef(r.Get("completed_by_user_id")), CancelledAtRaw: r.Get("cancelled_at"), CarrierRaw: r.Get("shipping_carrier"),
			TrackingRefSHA: refSHA(r.Get("shipping_tracking_code")), ProviderRefSHA: refSHA(r.Get("wallet_external_id")),
			CreatedAtRaw: r.Get("created_at"), UpdatedAtRaw: r.Get("updated_at")}
		if c.ProviderRefSHA != nil {
			s := "biawin.wallet_external_id"
			c.ProviderRefScheme = &s
		}
		refs(&c, r.Get("membership_item_id"))
		c.Assignments = append(assign(fulfillmentapi.RoleManager, r.Get("manager_user_id"), nil), assign(fulfillmentapi.RoleExpert, r.Get("expert_user_id"), r.Get("assigned_at"))...)
		in.Cases = append(in.Cases, c)
	}
	ships := map[string]bool{}
	for _, r := range b.Entity(SpecShippingOrder).Rows {
		ships[r.Key] = true
		raw := r.Get("fulfillment_status")
		c := fulfillmentapi.CaseRecord{Kind: fulfillmentapi.KindShipping, SourceSpec: SpecShippingOrder, SourceKey: r.Key, SubtypeRaw: r.Get("shipping_method"),
			Invoice: inv.fLink(r.Get("invoice_id")), StatusNamespace: "legacy_shipping_order.fulfillment_status", StatusRaw: raw,
			StatusMapped:     mapState(raw, shippingStates, fulfillmentapi.StatusUnknown, fulfillmentapi.StatusEmpty),
			ContactStatusRaw: r.Get("contact_status"), ShippedAtRaw: r.Get("shipped_at"), DeliveredAtRaw: r.Get("delivered_at"),
			CarrierRaw: r.Get("carrier"), TrackingRefSHA: refSHA(r.Get("tracking_code")), CreatedAtRaw: r.Get("created_at"), UpdatedAtRaw: r.Get("updated_at"),
			Assignments: assign(fulfillmentapi.RoleHandler, r.Get("handled_by"), nil)}
		in.Cases = append(in.Cases, c)
	}
	for _, r := range b.Entity(SpecFormSubmission).Rows {
		ck, _ := key(r.Get("execution_case_id"))
		if ck == nil || !execs[*ck] {
			in.Skipped["evidence_without_staged_case"]++
			continue
		}
		raw := r.Get("final_decision")
		in.Evidence = append(in.Evidence, fulfillmentapi.EvidenceRecord{SourceKey: r.Key, CaseSourceSpec: SpecExecutionCase, CaseSourceKey: *ck,
			FormDefinitionKey: r.Get("form_definition_id"), SubmittedBy: users.fRef(r.Get("submitted_by_user_id")), SubmittedAtRaw: r.Get("submitted_at"),
			OutcomeRaw: raw, OutcomeMapped: mapState(raw, outcomes, fulfillmentapi.OutcomeUnknown, fulfillmentapi.OutcomeNone),
			DecidedBy: users.fRef(r.Get("decided_by_user_id")), DecidedAtRaw: r.Get("decided_at")})
	}
	for _, r := range b.Entity(SpecExecutionEvent).Rows {
		ck, _ := key(r.Get("execution_case_id"))
		ek := r.Get("event_key")
		if ck == nil || !execs[*ck] || ek == nil {
			in.Skipped["execution_event_without_staged_case"]++
			continue
		}
		in.Events = append(in.Events, fulfillmentapi.EventRecord{SourceSpec: SpecExecutionEvent, SourceKey: r.Key, CaseSourceSpec: SpecExecutionCase,
			CaseSourceKey: *ck, Actor: users.fRef(r.Get("actor_user_id")), EventKeyRaw: *ek, OccurredAtRaw: r.Get("created_at")})
	}
	for _, r := range b.Entity(SpecShippingOrderLog).Rows {
		ck, _ := key(r.Get("shipping_order_id"))
		ek := r.Get("event_key")
		if ck == nil || !ships[*ck] || ek == nil {
			in.Skipped["shipping_log_without_staged_order"]++
			continue
		}
		in.Events = append(in.Events, fulfillmentapi.EventRecord{SourceSpec: SpecShippingOrderLog, SourceKey: r.Key, CaseSourceSpec: SpecShippingOrder,
			CaseSourceKey: *ck, Actor: users.fRef(r.Get("actor_user_id")), EventKeyRaw: *ek, OldStatusRaw: r.Get("old_status"),
			NewStatusRaw: r.Get("new_status"), OccurredAtRaw: r.Get("created_at")})
	}
	sort.SliceStable(in.Cases, func(a, b int) bool { return in.Cases[a].SourceSpec < in.Cases[b].SourceSpec })
	return in
}
