package review

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"sort"
	"unicode/utf8"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	paymentsapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
)

// R4: the Finance decision CONTRACT. It extends Evaluate (Slice 12) with
// every check a future approve / reject command must pass, without
// bypassing any Slice 12 guard: the Slice 12 gate is evaluated first and
// its reasons and decisions are always carried. Nothing here executes,
// records or routes a decision; OPD-08, D-43, D-41 and OPD-09 stay open, so
// the answer is never SUCCESS.

// More decision ids / engineering gaps named by the contract.
const (
	// RejectionVocabularyUndefined: no approved structured rejection code
	// vocabulary exists; any code is refused until one is approved.
	RejectionVocabularyUndefined = "REJECTION_REASON_VOCABULARY_UNDEFINED"
	// PersonLinkUnmodelled: identity has no person-level model linking
	// several accounts of one human; independence is proven only at the
	// account level until one exists.
	PersonLinkUnmodelled = "IDENTITY_PERSON_LINK_UNMODELLED"
)

// PersonLink is identity evidence about whether the reviewer's and the
// submitter's accounts belong to the same human.
type PersonLink string

const (
	// PersonLinkSameHuman: evidence links both accounts to one person.
	PersonLinkSameHuman PersonLink = "SAME_HUMAN"
	// PersonLinkDistinct: evidence proves two different people.
	PersonLinkDistinct PersonLink = "DISTINCT"
	// PersonLinkNotModelled: no person-level evidence exists (today).
	PersonLinkNotModelled PersonLink = "NOT_MODELLED"
)

// Reviewer is the acting reviewer as the authenticated request presents
// it. Every field is server-derived; a client never supplies authority.
type Reviewer struct {
	UserID string
	// Verified: the subject came from a verified session (not a stub, a
	// header or a client-supplied id).
	Verified bool
	// Capabilities are the reviewer's effective capabilities.
	Capabilities []string
	// InRecordScope: the attempt is inside the reviewer's record scope.
	InRecordScope bool
	PersonLink    PersonLink
}

// EvidenceRef is the evidence a decision cites: a receipt and the digest of
// the bytes the reviewer saw.
type EvidenceRef struct {
	ReceiptID     string
	ContentSHA256 string
}

// Rejection is a structured rejection reason.
type Rejection struct {
	Code   string
	Detail string
}

// MaxRejectionDetail bounds the free-text detail (runes).
const MaxRejectionDetail = 1000

// rejectionCodes is the approved structured rejection vocabulary. It is
// EMPTY: no vocabulary has been approved (OPD-08 / billing rejection
// transition). It is never invented here.
var rejectionCodes = map[string]bool{}

// DecisionRequest is one proposed Finance decision.
type DecisionRequest struct {
	Action   string
	Attempt  paymentsapi.ReviewAttempt
	Reviewer Reviewer
	// ExpectedVersion is the attempt version the reviewer saw.
	ExpectedVersion int64
	Evidence        []EvidenceRef
	// PriorDecision is a decision already recorded for this attempt.
	PriorDecision *string
	Rejection     *Rejection
}

// Decision is the contract's answer.
type Decision struct {
	Action           string
	Status           result.Status
	PreconditionsMet bool
	Reasons          []reasoncodes.Code
	BlockedBy        []string
	Findings         []string
	// EvaluationSHA256 is a digest of the canonical request and answer, for
	// an append-only audit record of the evaluation (not a decision).
	EvaluationSHA256 string
}

var hexSHA = regexp.MustCompile(`^[0-9a-f]{64}$`)

// EvaluateDecision evaluates one proposed decision against the contract.
func EvaluateDecision(req DecisionRequest) Decision {
	d := Decision{Action: req.Action}
	reasons := map[reasoncodes.Code]bool{}
	var blocked []string
	unauthorized, conflict, unknown, fail := false, false, false, false
	finding := func(code reasoncodes.Code, f string) {
		reasons[code] = true
		d.Findings = append(d.Findings, f)
	}
	if req.Action != ActionApprove && req.Action != ActionReject {
		fail = true
		finding(reasoncodes.ContractInvalid, "unknown Finance action")
	}
	// 1. The Slice 12 gate, never bypassed.
	gate := Evaluate(req.Attempt, req.Reviewer.UserID)
	for _, b := range gate.Actions {
		if b.Action != req.Action {
			continue
		}
		for _, r := range b.Reasons {
			reasons[r] = true
		}
		blocked = append(blocked, b.BlockedBy...)
		if b.Status == result.Unauthorized {
			unauthorized = true
			d.Findings = append(d.Findings, "the submitter is never the reviewer (maker != checker)")
		}
	}
	if !gate.EvidenceReady {
		d.Findings = append(d.Findings, "evidence is not READY (D-43 safe default)")
	}
	// 2. Verified human identity and independence beyond account ids.
	r := req.Reviewer
	if !r.Verified || r.UserID == "" {
		unauthorized = true
		finding(reasoncodes.ActionDenied, "the reviewer is not a verified authenticated user")
	}
	switch r.PersonLink {
	case PersonLinkDistinct:
	case PersonLinkSameHuman:
		unauthorized = true
		finding(reasoncodes.ActionDenied, "the reviewer and the submitter are the same human (several accounts)")
	default:
		blocked = append(blocked, PersonLinkUnmodelled)
		finding(reasoncodes.CoverageIncomplete, "human independence is proven only at the account level")
	}
	// 3. Explicit capability and record scope.
	hasCap := false
	for _, c := range r.Capabilities {
		hasCap = hasCap || c == req.Action
	}
	if !hasCap {
		finding(reasoncodes.ActionDenied, "the reviewer holds no decision capability")
	}
	if !r.InRecordScope {
		unauthorized = true
		finding(reasoncodes.RowScopeDenied, "the attempt is outside the reviewer's record scope")
	}
	// 4. Reviewable state, stale version and duplicate decision.
	if req.Attempt.State != paymentsapi.AttemptPendingReview {
		conflict = true
		finding(reasoncodes.StaleState, "the attempt is not awaiting review ("+req.Attempt.State+")")
	}
	if req.ExpectedVersion != req.Attempt.Version {
		conflict = true
		finding(reasoncodes.StaleState, "the attempt changed since the reviewer saw it (stale version)")
	}
	if req.PriorDecision != nil {
		conflict = true
		finding(reasoncodes.StaleState, "a decision is already recorded for this attempt (duplicate review)")
	}
	// 5. Immutable evidence references: every READY receipt is cited by its
	// recorded digest, nothing else is cited.
	cited := map[string]string{}
	for _, e := range req.Evidence {
		cited[e.ReceiptID] = e.ContentSHA256
	}
	if len(req.Evidence) == 0 {
		fail = true
		finding(reasoncodes.ContractInvalid, "a decision must cite its evidence")
	}
	known := map[string]bool{}
	for _, rc := range req.Attempt.Receipts {
		known[rc.ReceiptID] = true
		sum, ok := cited[rc.ReceiptID]
		switch {
		case !ok:
			fail = true
			finding(reasoncodes.ContractInvalid, "receipt "+rc.ReceiptID+" is not cited")
		case !hexSHA.MatchString(rc.ContentSHA256) || sum != rc.ContentSHA256:
			conflict = true
			finding(reasoncodes.LinkageConflict, "cited digest of receipt "+rc.ReceiptID+" does not match the immutable evidence")
		}
	}
	for id := range cited {
		if !known[id] {
			conflict = true
			finding(reasoncodes.LinkageConflict, "cited receipt "+id+" does not belong to the attempt")
		}
	}
	// 6. Structured rejection reason.
	if req.Action == ActionReject {
		switch {
		case req.Rejection == nil || req.Rejection.Code == "":
			fail = true
			finding(reasoncodes.ContractInvalid, "a rejection needs a structured reason code")
		case !rejectionCodes[req.Rejection.Code]:
			blocked = append(blocked, RejectionVocabularyUndefined)
			finding(reasoncodes.PolicyUnresolved, "no approved rejection reason vocabulary")
		}
		if req.Rejection != nil && (req.Rejection.Detail == "" || utf8.RuneCountInString(req.Rejection.Detail) > MaxRejectionDetail) {
			fail = true
			finding(reasoncodes.ContractInvalid, "a rejection needs a bounded detail")
		}
	} else if req.Rejection != nil {
		fail = true
		finding(reasoncodes.ContractInvalid, "an approval carries no rejection reason")
	}
	// Policy: open decisions always block.
	reasons[reasoncodes.PolicyUnresolved] = true
	d.PreconditionsMet = !unauthorized && !conflict && !unknown && !fail && gate.EvidenceReady && hasCap &&
		r.PersonLink == PersonLinkDistinct
	switch {
	case unauthorized:
		d.Status = result.Unauthorized
	case conflict:
		d.Status = result.Conflict
	case unknown:
		d.Status = result.Unknown
	case fail:
		d.Status = result.Failed
	default:
		d.Status = result.Blocked
	}
	for _, c := range reasoncodes.Codes() {
		if reasons[c] {
			d.Reasons = append(d.Reasons, c)
		}
	}
	d.BlockedBy = dedupeS(blocked)
	d.EvaluationSHA256 = evaluationDigest(req, d)
	return d
}

// evaluationDigest is SHA-256 over a canonical JSON of the request and the
// answer (sorted evidence; capabilities sorted). Any change of an input or
// of the answer changes it.
func evaluationDigest(req DecisionRequest, d Decision) string {
	ev := append([]EvidenceRef(nil), req.Evidence...)
	sort.Slice(ev, func(i, j int) bool { return ev[i].ReceiptID < ev[j].ReceiptID })
	caps := append([]string(nil), req.Reviewer.Capabilities...)
	sort.Strings(caps)
	b, err := json.Marshal(struct {
		Action, Attempt, Reviewer, Submitter string
		Verified, InScope                    bool
		PersonLink                           PersonLink
		Capabilities                         []string
		Expected, Version                    int64
		Evidence                             []EvidenceRef
		Prior                                *string
		Rejection                            *Rejection
		Status                               string
		Reasons                              []reasoncodes.Code
		BlockedBy                            []string
	}{req.Action, req.Attempt.AttemptID, req.Reviewer.UserID, req.Attempt.SubmittedByUserID, req.Reviewer.Verified, req.Reviewer.InRecordScope,
		req.Reviewer.PersonLink, caps, req.ExpectedVersion, req.Attempt.Version, ev, req.PriorDecision, req.Rejection, d.Status.String(), d.Reasons,
		d.BlockedBy})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
