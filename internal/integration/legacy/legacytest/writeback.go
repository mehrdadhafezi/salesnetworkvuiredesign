package legacytest

import (
	"context"
	"errors"
	"strconv"
	"sync"

	lwb "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/writeback"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

// Fault is an injected legacy failure for the next call of one operation.
type Fault uint8

// Faults.
const (
	NoFault Fault = iota
	// TimeoutAfterCommit commits the effect, then reports a transport error.
	TimeoutAfterCommit
	// TimeoutBeforeCommit reports a transport error with nothing committed.
	TimeoutBeforeCommit
	// ObserveDown makes the next observe fail (legacy unreadable).
	ObserveDown
)

// ErrTransport is the synthetic transport failure.
var ErrTransport = errors.New("synthetic legacy: transport timeout")

// SyntheticLead is one synthetic legacy lead (SYNTHETIC data only).
type SyntheticLead struct {
	SellerLegacyUserID string
	Archived           bool
	StatusLabel        string
	NotPurchaseReason  string
	CallbackJalali     string
}

type flowEvent struct {
	id    int
	lead  string
	actor string
}

type hrEvent struct {
	action, key, actor string
}

// SyntheticHRRequest is one synthetic legacy HR change request.
type SyntheticHRRequest struct {
	ID                                                  int
	RequestType, Subject, Target, RequestedBy, Reviewer string
	Status, Reason                                      string
	Path                                                []hrEvent
}

// SyntheticLegacy is an isolated in-memory legacy master implementing the
// proven 2.0.160 semantics of the Slice 11 write paths (owner-only lead
// update, archive refusal, UNIQUE request_token, one open HR request per
// subject + type, reviewer-only step decisions). It never touches a real
// database or network. Calls counts every mutating call per operation.
type SyntheticLegacy struct {
	mu       sync.Mutex
	Leads    map[string]*SyntheticLead
	events   map[string]flowEvent
	nextFlow int
	Requests map[int]*SyntheticHRRequest
	nextHR   int
	faults   map[string]Fault
	Calls    map[string]int
	// Invoices are the synthetic legacy invoices for payments write-back.
	Invoices map[string]*SyntheticInvoice
	nextPay  int
}

var _ lwb.Legacy = (*SyntheticLegacy)(nil)

// NewSyntheticLegacy returns an empty synthetic legacy store.
func NewSyntheticLegacy() *SyntheticLegacy {
	return &SyntheticLegacy{Leads: map[string]*SyntheticLead{}, events: map[string]flowEvent{}, Requests: map[int]*SyntheticHRRequest{},
		faults: map[string]Fault{}, Calls: map[string]int{}, nextFlow: 500, nextHR: 9000, Invoices: map[string]*SyntheticInvoice{}, nextPay: 70000}
}

// Inject sets a fault for the next call of op ("lead", "lead_observe",
// "hr_insert", "hr_update", "hr_observe").
func (s *SyntheticLegacy) Inject(op string, f Fault) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.faults[op] = f
}

func (s *SyntheticLegacy) take(op string) Fault {
	f := s.faults[op]
	delete(s.faults, op)
	return f
}

// Lead returns a copy of a lead.
func (s *SyntheticLegacy) Lead(id string) SyntheticLead {
	s.mu.Lock()
	defer s.mu.Unlock()
	if l := s.Leads[id]; l != nil {
		return *l
	}
	return SyntheticLead{}
}

// FlowEvents returns the number of seller-flow events.
func (s *SyntheticLegacy) FlowEvents() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.events)
}

// ApplyLeadOutcome implements lwb.Legacy.
func (s *SyntheticLegacy) ApplyLeadOutcome(_ context.Context, op lwb.LeadOutcome) (lwb.Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Calls["lead"]++
	fault := s.take("lead")
	if fault == TimeoutBeforeCommit {
		return lwb.Result{}, ErrTransport
	}
	if ev, ok := s.events[op.RequestToken]; ok { // UNIQUE request_token
		if ev.lead == op.LeadID {
			return lwb.Result{Outcome: result.Success, Ref: "flow_event:" + strconv.Itoa(ev.id)}, nil
		}
		return lwb.Result{Outcome: result.Conflict, Reason: "sn_seller_flow_token_conflict"}, nil
	}
	l := s.Leads[op.LeadID]
	switch {
	case l == nil:
		return lwb.Result{Outcome: result.Failed, Reason: "lead_not_found"}, nil
	case l.SellerLegacyUserID != op.ActorLegacyUserID:
		return lwb.Result{Outcome: result.Unauthorized, Reason: "not_lead_owner"}, nil
	case l.Archived:
		return lwb.Result{Outcome: result.Conflict, Reason: "sn_seller_flow_archived"}, nil
	case op.StatusLabel == "عدم خرید" && op.NotPurchaseReason == "":
		return lwb.Result{Outcome: result.Failed, Reason: "sn_not_purchase_reason_required"}, nil
	case op.StatusLabel == "تماس مجدد" && op.CallbackJalali == "":
		return lwb.Result{Outcome: result.Failed, Reason: "sn_callback_datetime_required"}, nil
	}
	l.StatusLabel, l.NotPurchaseReason, l.CallbackJalali = op.StatusLabel, op.NotPurchaseReason, op.CallbackJalali
	s.nextFlow++
	s.events[op.RequestToken] = flowEvent{id: s.nextFlow, lead: op.LeadID, actor: op.ActorLegacyUserID}
	if fault == TimeoutAfterCommit {
		return lwb.Result{}, ErrTransport
	}
	return lwb.Result{Outcome: result.Success, Ref: "flow_event:" + strconv.Itoa(s.nextFlow)}, nil
}

// ObserveLeadOutcome implements lwb.Legacy.
func (s *SyntheticLegacy) ObserveLeadOutcome(_ context.Context, leadID, token string) (lwb.LeadObserved, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.take("lead_observe") == ObserveDown {
		return lwb.LeadObserved{}, ErrTransport
	}
	ev, ok := s.events[token]
	if !ok || ev.lead != leadID {
		return lwb.LeadObserved{}, nil
	}
	l := s.Leads[leadID]
	return lwb.LeadObserved{Events: 1, StatusLabel: l.StatusLabel, NotPurchaseReason: l.NotPurchaseReason, CallbackJalali: l.CallbackJalali,
		EventRef: "flow_event:" + strconv.Itoa(ev.id)}, nil
}

func (s *SyntheticLegacy) byKey(key string) []*SyntheticHRRequest {
	var out []*SyntheticHRRequest
	for _, r := range s.Requests {
		for _, e := range r.Path {
			if e.key == key {
				out = append(out, r)
				break
			}
		}
	}
	return out
}

// InsertHRRequest implements lwb.Legacy.
func (s *SyntheticLegacy) InsertHRRequest(_ context.Context, op lwb.HRCreate) (lwb.Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Calls["hr_insert"]++
	fault := s.take("hr_insert")
	if fault == TimeoutBeforeCommit {
		return lwb.Result{}, ErrTransport
	}
	if found := s.byKey(op.ReconcileKey); len(found) == 1 {
		return lwb.Result{Outcome: result.Success, Ref: strconv.Itoa(found[0].ID)}, nil
	}
	for _, r := range s.Requests {
		if r.Subject == op.SubjectLegacyUserID && r.RequestType == op.RequestType && (r.Status == "pending_review" || r.Status == "pending_hr") {
			return lwb.Result{Outcome: result.Conflict, Reason: "duplicate"}, nil
		}
	}
	s.nextHR++
	s.Requests[s.nextHR] = &SyntheticHRRequest{ID: s.nextHR, RequestType: op.RequestType, Subject: op.SubjectLegacyUserID, Target: op.TargetParentLegacyUserID,
		RequestedBy: op.RequestedByLegacyUserID, Reviewer: op.ReviewerLegacyUserID, Status: op.Status, Reason: op.Reason,
		Path: []hrEvent{{action: "created", key: op.ReconcileKey, actor: op.RequestedByLegacyUserID}}}
	if fault == TimeoutAfterCommit {
		return lwb.Result{}, ErrTransport
	}
	return lwb.Result{Outcome: result.Success, Ref: strconv.Itoa(s.nextHR)}, nil
}

func observed(r *SyntheticHRRequest, n int) lwb.HRObserved {
	return lwb.HRObserved{Matches: n, LegacyRequestID: strconv.Itoa(r.ID), Status: r.Status, ReviewerLegacyUserID: r.Reviewer,
		RequestType: r.RequestType, SubjectLegacyUserID: r.Subject, TargetParentLegacyUserID: r.Target}
}

// ObserveHRCreate implements lwb.Legacy.
func (s *SyntheticLegacy) ObserveHRCreate(_ context.Context, key string) (lwb.HRObserved, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.take("hr_observe") == ObserveDown {
		return lwb.HRObserved{}, ErrTransport
	}
	found := s.byKey(key)
	if len(found) == 0 {
		return lwb.HRObserved{}, nil
	}
	return observed(found[0], len(found)), nil
}

// UpdateHRRequest implements lwb.Legacy.
func (s *SyntheticLegacy) UpdateHRRequest(_ context.Context, op lwb.HRDecide) (lwb.Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Calls["hr_update"]++
	fault := s.take("hr_update")
	if fault == TimeoutBeforeCommit {
		return lwb.Result{}, ErrTransport
	}
	id, _ := strconv.Atoi(op.LegacyRequestID)
	r := s.Requests[id]
	if r == nil {
		return lwb.Result{Outcome: result.Failed, Reason: "request_missing"}, nil
	}
	for _, e := range r.Path {
		if e.key == op.ReconcileKey {
			return lwb.Result{Outcome: result.Success, Ref: op.LegacyRequestID}, nil
		}
	}
	if r.Status != op.FromStatus || r.Status != "pending_review" || r.Reviewer != op.ActorLegacyUserID {
		return lwb.Result{Outcome: result.Unauthorized, Reason: "review_permission"}, nil
	}
	action := "rejected"
	if op.Decision == "approve" {
		action = "approved_step"
	}
	r.Status, r.Reviewer = op.ToStatus, op.NextReviewerLegacyUserID
	r.Path = append(r.Path, hrEvent{action: action, key: op.ReconcileKey, actor: op.ActorLegacyUserID})
	if fault == TimeoutAfterCommit {
		return lwb.Result{}, ErrTransport
	}
	return lwb.Result{Outcome: result.Success, Ref: op.LegacyRequestID}, nil
}

// ObserveHRDecide implements lwb.Legacy.
func (s *SyntheticLegacy) ObserveHRDecide(_ context.Context, legacyID, key string) (lwb.HRObserved, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.take("hr_observe") == ObserveDown {
		return lwb.HRObserved{}, ErrTransport
	}
	id, _ := strconv.Atoi(legacyID)
	r := s.Requests[id]
	if r == nil {
		return lwb.HRObserved{}, nil
	}
	n := 0
	for _, e := range r.Path {
		if e.key == key {
			n++
		}
	}
	if n == 0 {
		return lwb.HRObserved{}, nil
	}
	return observed(r, n), nil
}

// Request returns a copy of a legacy HR request.
func (s *SyntheticLegacy) Request(id int) SyntheticHRRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r := s.Requests[id]; r != nil {
		return *r
	}
	return SyntheticHRRequest{}
}

// Duplicate copies the legacy event carrying key onto a second request, to
// simulate an ambiguous legacy result (two candidates for one intent).
func (s *SyntheticLegacy) Duplicate(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.byKey(key) {
		s.nextHR++
		cp := *r
		cp.ID = s.nextHR
		cp.Path = append([]hrEvent(nil), r.Path...)
		s.Requests[cp.ID] = &cp
		return
	}
}
