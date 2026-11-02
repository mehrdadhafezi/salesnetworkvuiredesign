// Package hrrequest holds the organization module's Slice 11 controlled
// writes: CreateHRRequest and DecideHRRequest (step approval / rejection).
//
// Authority (after the capability, checked by the HTTP layer) is decided
// here from organization evidence only, inside the owner transaction
// (apply-time recheck):
//
//   - the actor holds exactly one LINKED, PRESENT, active employment whose
//     position is one of the legacy request creators (supervisor,
//     senior_supervisor, sales_manager, sales_deputy). Position alone never
//     grants: the capability is required first, and the subject must be a
//     seller DESCENDANT of the actor through current REPORTS_TO links;
//   - an HR creator, a final approval (HR actor or a pending_hr request) and
//     every apply effect are BLOCKED (OPD-05, SD-04; HR apply is
//     IMPLEMENTATION BLOCKED);
//   - a transfer destination must be an active supervisor inside the actor's
//     own scope; anything else is BLOCKED (SD-04), and a destination equal
//     to the subject's current parent is BLOCKED (SD-04 no-op);
//   - ambiguous or missing hierarchy evidence is BLOCKED, never guessed.
//
// Requester, subject parent, target and reviewer are snapshots taken at
// the transition. Nothing here writes identity, credentials, capabilities,
// hierarchy, employment, cases or custody.
package hrrequest

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/idempotency"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/outbox"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

// Idempotency scopes and command types.
const (
	ScopeCreate  = "organization.hr_request_create"
	ScopeDecide  = "organization.hr_request_decide"
	TypeCreate   = "organization.create_hr_request"
	TypeDecide   = "organization.decide_hr_request"
	maxHierDepth = 12
)

// Decision ids named by refusals.
const (
	DecisionOPD05     = "OPD-05"
	DecisionSD04      = "SD-04"
	DecisionHRRuntime = "HR_APPLY_IMPLEMENTATION_BLOCKED"
)

// Writeback states reported at commit time.
const (
	WritebackPending      = "LEGACY_WRITEBACK_PENDING"
	WritebackNotRequested = "NOT_REQUESTED"
)

var (
	// ErrInvalid reports a malformed command (400).
	ErrInvalid = errors.New("hrrequest: invalid command")
	// ErrNotFound reports an unknown request (404).
	ErrNotFound = errors.New("hrrequest: not found")
)

func invalid(f string) error { return fmt.Errorf("%w: %s", ErrInvalid, f) }

// creators are the legacy positions allowed to create a request
// (class-sn-plugin.php:10394-10399, without the HR and administrator paths).
var creators = map[string]bool{"supervisor": true, "senior_supervisor": true, "sales_manager": true, "sales_deputy": true}

const (
	positionSeller     = "seller"
	positionSupervisor = "supervisor"
	positionHR         = "hr"
)

// Actor is who issues a command.
type Actor struct{ Principal, UserID string }

// Result is a command outcome (also the idempotent replay).
type Result struct {
	Status         result.Status      `json:"-"`
	StatusName     string             `json:"status"`
	Reasons        []reasoncodes.Code `json:"reasons"`
	BlockedBy      []string           `json:"blocked_by,omitempty"`
	CommandID      string             `json:"command_id,omitempty"`
	RequestID      string             `json:"request_id,omitempty"`
	RequestStatus  string             `json:"request_status,omitempty"`
	Mode           rollout.Mode       `json:"rollout_mode"`
	Applied        bool               `json:"applied"`
	Replayed       bool               `json:"-"`
	Version        int64              `json:"version,omitempty"`
	CurrentVersion int64              `json:"-"`
	WritebackState string             `json:"writeback_state"`
}

func refuse(status result.Status, mode rollout.Mode, code reasoncodes.Code, blockedBy ...string) Result {
	return Result{Status: status, StatusName: status.String(), Reasons: []reasoncodes.Code{code}, BlockedBy: blockedBy, Mode: mode,
		WritebackState: WritebackNotRequested}
}

type errRefused struct{ r Result }

func (e errRefused) Error() string { return "refused: " + e.r.StatusName }

func stop(status result.Status, mode rollout.Mode, code reasoncodes.Code, blockedBy ...string) error {
	return errRefused{refuse(status, mode, code, blockedBy...)}
}

// Service runs the commands.
type Service struct {
	runner *db.Runner
	clock  timeutil.Clock
}

// New returns the service over an organization-owned runner.
func New(runner *db.Runner, clock timeutil.Clock) (*Service, error) {
	if runner == nil || runner.Module() != organization.Module || clock == nil {
		return nil, errors.New("hrrequest: runner must be owned by organization; clock required")
	}
	return &Service{runner: runner, clock: clock}, nil
}

var (
	uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	keyPattern  = regexp.MustCompile(`^[A-Za-z0-9_-]{8,128}$`)
)

func common(key, corr string, a Actor) error {
	if !keyPattern.MatchString(key) {
		return invalid("idempotency_key")
	}
	if corr == "" || len(corr) > 128 {
		return invalid("correlation_id")
	}
	return nil
}

// ---- organization evidence (own schema, read inside the command tx) --------

type employment struct {
	ID, UserID, Status, Position string
	Active                       *bool
}

// employmentOf returns the single PRESENT, LINKED employment of a user and
// its position slug. Zero, several, a shared user or an unresolved position
// is not evidence.
func employmentOf(ctx context.Context, tx db.Tx, userID string) (*employment, error) {
	rows, err := tx.Query(ctx, `SELECT e.employment_id::text, e.user_id::text, e.employment_status, e.is_active, p.slug, e.user_link_state, e.position_ref_state
		FROM organization.employment e LEFT JOIN organization.position p ON p.position_id = e.position_id
		WHERE e.user_id = $1 AND e.presence = 'PRESENT'`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []employment
	var bad bool
	for rows.Next() {
		var e employment
		var slug *string
		var link, posRef string
		if err := rows.Scan(&e.ID, &e.UserID, &e.Status, &e.Active, &slug, &link, &posRef); err != nil {
			return nil, err
		}
		if link != "LINKED" || posRef != "RESOLVED" || slug == nil {
			bad = true
			continue
		}
		e.Position = *slug
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if bad || len(out) != 1 {
		return nil, nil
	}
	return &out[0], nil
}

func knownStatus(s string) bool { return s != "UNMAPPED" && s != "NOT_RECORDED" }

func activeEmployment(e *employment) bool {
	return (e.Status == "ACTIVE" || e.Status == "PROBATION") && (e.Active == nil || *e.Active)
}

// parentOf returns the single current REPORTS_TO parent of an employment.
// none = no current link; ambiguous = several, an unresolved parent or an
// undecidable current flag.
func parentOf(ctx context.Context, tx db.Tx, employmentID string) (parent *employment, none, ambiguous bool, err error) {
	rows, err := tx.Query(ctx, `SELECT a.parent_ref_state, a.is_current, a.relationship_kind, pe.employment_id::text, pe.user_id::text, pe.user_link_state
		FROM organization.org_assignment a LEFT JOIN organization.employment pe ON pe.employment_id = a.parent_employment_id
		WHERE a.child_employment_id = $1 AND a.presence = 'PRESENT'`, employmentID)
	if err != nil {
		return nil, false, false, err
	}
	defer rows.Close()
	var current []employment
	for rows.Next() {
		var ref, kind string
		var cur *bool
		var pid, puser, plink *string
		if err := rows.Scan(&ref, &cur, &kind, &pid, &puser, &plink); err != nil {
			return nil, false, false, err
		}
		if cur == nil {
			ambiguous = true
			continue
		}
		if !*cur {
			continue
		}
		if kind != "REPORTS_TO" || ref != "RESOLVED" || pid == nil || puser == nil || plink == nil || *plink != "LINKED" {
			ambiguous = true
			continue
		}
		current = append(current, employment{ID: *pid, UserID: *puser})
	}
	if err := rows.Err(); err != nil {
		return nil, false, false, err
	}
	switch {
	case ambiguous || len(current) > 1:
		return nil, false, true, nil
	case len(current) == 0:
		return nil, true, false, nil
	}
	return &current[0], false, false, nil
}

// descendant reports whether child reaches ancestor through current
// REPORTS_TO links. Ambiguity anywhere on the path is not evidence.
func descendant(ctx context.Context, tx db.Tx, childEmployment, ancestorEmployment string) (yes, ambiguous bool, err error) {
	cur := childEmployment
	for i := 0; i < maxHierDepth; i++ {
		p, none, amb, err := parentOf(ctx, tx, cur)
		if err != nil || amb {
			return false, amb, err
		}
		if none {
			return false, false, nil
		}
		if p.ID == ancestorEmployment {
			return true, false, nil
		}
		cur = p.ID
	}
	return false, true, nil // depth exceeded or cycle: not evidence
}

// nextReviewer is the legacy next_reviewer_user_id rule: the actor's current
// parent; none → pending_hr.
func nextReviewer(ctx context.Context, tx db.Tx, actorEmployment string) (reviewer *string, status string, ambiguous bool, err error) {
	p, none, amb, err := parentOf(ctx, tx, actorEmployment)
	if err != nil || amb {
		return nil, "", amb, err
	}
	if none {
		return nil, api.HRPendingHR, false, nil
	}
	u := p.UserID
	return &u, api.HRPendingReview, false, nil
}

// actorEvidence resolves the acting employment. Not linked / ambiguous →
// BLOCKED; HR → BLOCKED (HR runtime); inactive or a non-creator position →
// UNAUTHORIZED.
func actorEvidence(ctx context.Context, tx db.Tx, mode rollout.Mode, a Actor) (*employment, error) {
	e, err := employmentOf(ctx, tx, a.UserID)
	if err != nil {
		return nil, err
	}
	if e == nil || !knownStatus(e.Status) {
		return nil, stop(result.Blocked, mode, reasoncodes.LinkageConflict)
	}
	if e.Position == positionHR {
		return nil, stop(result.Blocked, mode, reasoncodes.PolicyUnresolved, DecisionHRRuntime, DecisionOPD05)
	}
	if !activeEmployment(e) || !creators[e.Position] {
		return nil, stop(result.Unauthorized, mode, reasoncodes.ActionDenied)
	}
	return e, nil
}

// ---- CreateHRRequest -------------------------------------------------------

// Create is one CreateHRRequest command.
type Create struct {
	RequestType        string
	SubjectUserID      string
	TargetParentUserID *string
	Reason             string
	IdempotencyKey     string
	CorrelationID      string
	Actor              Actor
}

func canonicalCreate(c Create) []byte {
	b, _ := json.Marshal(struct {
		Type    string  `json:"request_type"`
		Subject string  `json:"subject_user_id"`
		Target  *string `json:"target_parent_user_id"`
		Reason  string  `json:"reason"`
		Actor   string  `json:"actor_user_id"`
	}{c.RequestType, c.SubjectUserID, c.TargetParentUserID, c.Reason, c.Actor.UserID})
	return b
}

// CreateRequest runs CreateHRRequest in mode.
func (s *Service) CreateRequest(ctx context.Context, mode rollout.Mode, c Create) (Result, error) {
	if err := common(c.IdempotencyKey, c.CorrelationID, c.Actor); err != nil {
		return Result{}, err
	}
	if c.RequestType != api.HRTransferSeller && c.RequestType != api.HRTerminateSeller {
		return Result{}, invalid("request_type")
	}
	if !uuidPattern.MatchString(c.SubjectUserID) {
		return Result{}, invalid("subject_user_id")
	}
	if (c.RequestType == api.HRTransferSeller) != (c.TargetParentUserID != nil) {
		return Result{}, invalid("target_parent_user_id (required exactly for transfer_seller)")
	}
	if c.TargetParentUserID != nil && !uuidPattern.MatchString(*c.TargetParentUserID) {
		return Result{}, invalid("target_parent_user_id")
	}
	if len(c.Reason) == 0 || len(c.Reason) > 1000 {
		return Result{}, invalid("reason")
	}
	if mode == rollout.Disabled || !mode.Valid() {
		return refuse(result.Blocked, rollout.Disabled, reasoncodes.FeatureDisabled), nil
	}
	if c.Actor.UserID == "" || c.Actor.Principal == "" {
		return refuse(result.Unauthorized, mode, reasoncodes.RowScopeDenied), nil
	}
	sum := sha256.Sum256(canonicalCreate(c))
	req := idempotency.Request{Scope: ScopeCreate, Key: c.IdempotencyKey, Hash: idempotency.Hash(sum)}
	commandID := ids.Stable(ScopeCreate, c.IdempotencyKey)
	requestID := ids.Stable("organization.hr_request", commandID)
	var out Result
	err := s.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		now := s.clock.Now()
		if done, err := acquire(ctx, tx, req, mode, now, &out); done || err != nil {
			return err
		}
		actor, err := actorEvidence(ctx, tx, mode, c.Actor)
		if err != nil {
			return err
		}
		if c.SubjectUserID == c.Actor.UserID {
			return stop(result.Unauthorized, mode, reasoncodes.RowScopeDenied)
		}
		subject, err := employmentOf(ctx, tx, c.SubjectUserID)
		if err != nil {
			return err
		}
		if subject == nil || !knownStatus(subject.Status) {
			return stop(result.Blocked, mode, reasoncodes.LinkageConflict)
		}
		if subject.Position != positionSeller {
			return stop(result.Unauthorized, mode, reasoncodes.RowScopeDenied)
		}
		inScope, amb, err := descendant(ctx, tx, subject.ID, actor.ID)
		if err != nil {
			return err
		}
		if amb {
			return stop(result.Blocked, mode, reasoncodes.LinkageConflict)
		}
		if !inScope {
			return stop(result.Unauthorized, mode, reasoncodes.RowScopeDenied)
		}
		subjectParent, none, amb, err := parentOf(ctx, tx, subject.ID)
		if err != nil {
			return err
		}
		if amb {
			return stop(result.Blocked, mode, reasoncodes.LinkageConflict)
		}
		var parentEmp, parentUser *string
		if !none {
			parentEmp, parentUser = &subjectParent.ID, &subjectParent.UserID
		}
		var targetEmp *string
		if c.RequestType == api.HRTransferSeller {
			target, err := employmentOf(ctx, tx, *c.TargetParentUserID)
			if err != nil {
				return err
			}
			if target == nil || !knownStatus(target.Status) {
				return stop(result.Blocked, mode, reasoncodes.LinkageConflict)
			}
			if target.Position != positionSupervisor || !activeEmployment(target) {
				return stop(result.Blocked, mode, reasoncodes.RecipientIneligible)
			}
			if parentEmp != nil && *parentEmp == target.ID {
				return stop(result.Blocked, mode, reasoncodes.PolicyUnresolved, DecisionSD04) // no-op transfer: not frozen
			}
			targetInScope, amb, err := descendant(ctx, tx, target.ID, actor.ID)
			if err != nil {
				return err
			}
			if amb || !targetInScope {
				return stop(result.Blocked, mode, reasoncodes.PolicyUnresolved, DecisionSD04) // cross-scope destination
			}
			targetEmp = &target.ID
		}
		reviewer, status, amb, err := nextReviewer(ctx, tx, actor.ID)
		if err != nil {
			return err
		}
		if amb {
			return stop(result.Blocked, mode, reasoncodes.LinkageConflict)
		}
		var open int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM organization.hr_request WHERE subject_user_id = $1 AND request_type = $2
			AND status IN ('pending_review', 'pending_hr')`, c.SubjectUserID, c.RequestType).Scan(&open); err != nil {
			return err
		}
		if open > 0 {
			return stop(result.Conflict, mode, reasoncodes.StaleState) // legacy duplicate guard
		}
		at := now.Time().UTC()
		if mode.Applies() {
			_, err := tx.Exec(ctx, `INSERT INTO organization.hr_request (request_id, request_type, subject_user_id, subject_employment_id,
				subject_parent_employment_id, subject_parent_user_id, target_parent_employment_id, target_parent_user_id, requested_by_user_id,
				requested_by_employment_id, current_reviewer_user_id, status, reason, created_command_id, version, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, 1, $15, $15)`,
				requestID, c.RequestType, c.SubjectUserID, subject.ID, parentEmp, parentUser, targetEmp, c.TargetParentUserID, c.Actor.UserID,
				actor.ID, reviewer, status, c.Reason, commandID, at)
			var pg *pgconn.PgError
			if errors.As(err, &pg) && pg.Code == "23505" {
				return stop(result.Conflict, mode, reasoncodes.StaleState) // a concurrent open request won
			}
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO organization.hr_request_event (event_id, request_id, command_id, action, from_status, to_status,
				next_reviewer_user_id, notes, before_version, after_version, actor_principal, actor_user_id, correlation_id, occurred_at)
				VALUES ($1, $2, $3, 'created', NULL, $4, $5, NULL, NULL, 1, $6, $7, $8, $9)`,
				ids.Stable("organization.hr_request_event", commandID), requestID, commandID, status, reviewer, c.Actor.Principal,
				c.Actor.UserID, c.CorrelationID, at); err != nil {
				return err
			}
		}
		intent, _ := json.Marshal(map[string]any{"request_type": c.RequestType, "subject_user_id": c.SubjectUserID,
			"target_parent_user_id": c.TargetParentUserID, "status": status, "current_reviewer_user_id": reviewer})
		if err := record(ctx, tx, commandID, TypeCreate, requestID, c.IdempotencyKey, sum[:], mode, intent, c.Actor, c.CorrelationID, at); err != nil {
			return err
		}
		payload, err := json.Marshal(api.HRRequestCreated{CommandID: commandID, IdempotencyKey: c.IdempotencyKey, CorrelationID: c.CorrelationID,
			RequestID: requestID, RequestType: c.RequestType, SubjectUserID: c.SubjectUserID, SubjectParentUserID: parentUser,
			TargetParentUserID: c.TargetParentUserID, RequestedByUserID: c.Actor.UserID, CurrentReviewerUser: reviewer, Status: status,
			Reason: c.Reason, RolloutMode: string(mode), Applied: mode.Applies(), Version: 1, OccurredAt: at})
		if err != nil {
			return err
		}
		if err := emit(ctx, tx, commandID, api.EventHRRequestCreated, payload, c.CorrelationID, now); err != nil {
			return err
		}
		out = Result{Status: result.Success, StatusName: result.Success.String(), Reasons: []reasoncodes.Code{}, CommandID: commandID,
			RequestID: requestID, RequestStatus: status, Mode: mode, Applied: mode.Applies(), WritebackState: WritebackNotRequested}
		if mode.Applies() {
			out.Version, out.WritebackState = 1, WritebackPending
		}
		return complete(ctx, tx, req, out, now)
	})
	return finish(out, err)
}

// ---- DecideHRRequest -------------------------------------------------------

// Decide is one DecideHRRequest command.
type Decide struct {
	RequestID       string
	Decision        string
	Notes           *string
	ExpectedVersion int64
	IdempotencyKey  string
	CorrelationID   string
	Actor           Actor
}

func canonicalDecide(d Decide) []byte {
	b, _ := json.Marshal(struct {
		Request  string  `json:"request_id"`
		Decision string  `json:"decision"`
		Notes    *string `json:"notes"`
		Expected int64   `json:"expected_version"`
		Actor    string  `json:"actor_user_id"`
	}{d.RequestID, d.Decision, d.Notes, d.ExpectedVersion, d.Actor.UserID})
	return b
}

// DecideRequest runs DecideHRRequest in mode.
func (s *Service) DecideRequest(ctx context.Context, mode rollout.Mode, d Decide) (Result, error) {
	if err := common(d.IdempotencyKey, d.CorrelationID, d.Actor); err != nil {
		return Result{}, err
	}
	if !uuidPattern.MatchString(d.RequestID) {
		return Result{}, invalid("request_id")
	}
	if d.Decision != api.HRDecisionApprove && d.Decision != api.HRDecisionReject {
		return Result{}, invalid("decision")
	}
	if d.Notes != nil && (len(*d.Notes) == 0 || len(*d.Notes) > 1000) {
		return Result{}, invalid("notes")
	}
	if d.ExpectedVersion < 1 {
		return Result{}, invalid("expected_version")
	}
	if mode == rollout.Disabled || !mode.Valid() {
		return refuse(result.Blocked, rollout.Disabled, reasoncodes.FeatureDisabled), nil
	}
	if d.Actor.UserID == "" || d.Actor.Principal == "" {
		return refuse(result.Unauthorized, mode, reasoncodes.RowScopeDenied), nil
	}
	sum := sha256.Sum256(canonicalDecide(d))
	req := idempotency.Request{Scope: ScopeDecide, Key: d.IdempotencyKey, Hash: idempotency.Hash(sum)}
	commandID := ids.Stable(ScopeDecide, d.IdempotencyKey)
	var out Result
	err := s.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		now := s.clock.Now()
		if done, err := acquire(ctx, tx, req, mode, now, &out); done || err != nil {
			return err
		}
		var status string
		var reviewer *string
		var version int64
		err := tx.QueryRow(ctx, `SELECT status, current_reviewer_user_id::text, version FROM organization.hr_request WHERE request_id = $1 FOR UPDATE`,
			d.RequestID).Scan(&status, &reviewer, &version)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		e, err := employmentOf(ctx, tx, d.Actor.UserID)
		if err != nil {
			return err
		}
		if e != nil && e.Position == positionHR {
			return stop(result.Blocked, mode, reasoncodes.PolicyUnresolved, DecisionHRRuntime, DecisionOPD05, DecisionSD04)
		}
		if status == api.HRPendingHR {
			// Only HR finalizes; final apply is unavailable in Slice 11.
			return stop(result.Blocked, mode, reasoncodes.PolicyUnresolved, DecisionHRRuntime, DecisionOPD05, DecisionSD04)
		}
		if status != api.HRPendingReview {
			return stop(result.Conflict, mode, reasoncodes.StaleState)
		}
		if reviewer == nil || *reviewer != d.Actor.UserID {
			return stop(result.Unauthorized, mode, reasoncodes.RowScopeDenied)
		}
		if e == nil || !knownStatus(e.Status) {
			return stop(result.Blocked, mode, reasoncodes.LinkageConflict)
		}
		if !activeEmployment(e) {
			return stop(result.Unauthorized, mode, reasoncodes.ActionDenied)
		}
		if version != d.ExpectedVersion {
			r := refuse(result.Conflict, mode, reasoncodes.StaleState)
			r.CurrentVersion = version
			return errRefused{r}
		}
		to, action := api.HRRejected, "rejected"
		var next *string
		if d.Decision == api.HRDecisionApprove {
			var amb bool
			next, to, amb, err = nextReviewer(ctx, tx, e.ID)
			if err != nil {
				return err
			}
			if amb {
				return stop(result.Blocked, mode, reasoncodes.LinkageConflict)
			}
			action = "approved_step"
		}
		at := now.Time().UTC()
		var after *int64
		if mode.Applies() {
			tag, err := tx.Exec(ctx, `UPDATE organization.hr_request SET status = $2, current_reviewer_user_id = $3, reviewer_notes = $4,
				version = version + 1, updated_at = $5 WHERE request_id = $1 AND version = $6`, d.RequestID, to, next, d.Notes, at, version)
			if err != nil {
				return err
			}
			if tag.RowsAffected() != 1 {
				return stop(result.Conflict, mode, reasoncodes.StaleState)
			}
			v := version + 1
			after = &v
			if _, err := tx.Exec(ctx, `INSERT INTO organization.hr_request_event (event_id, request_id, command_id, action, from_status, to_status,
				next_reviewer_user_id, notes, before_version, after_version, actor_principal, actor_user_id, correlation_id, occurred_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
				ids.Stable("organization.hr_request_event", commandID), d.RequestID, commandID, action, status, to, next, d.Notes, version, v,
				d.Actor.Principal, d.Actor.UserID, d.CorrelationID, at); err != nil {
				return err
			}
		}
		intent, _ := json.Marshal(map[string]any{"decision": d.Decision, "from_status": status, "to_status": to, "next_reviewer_user_id": next})
		if err := record(ctx, tx, commandID, TypeDecide, d.RequestID, d.IdempotencyKey, sum[:], mode, intent, d.Actor, d.CorrelationID, at); err != nil {
			return err
		}
		payload, err := json.Marshal(api.HRRequestDecided{CommandID: commandID, IdempotencyKey: d.IdempotencyKey, CorrelationID: d.CorrelationID,
			RequestID: d.RequestID, Decision: d.Decision, FromStatus: status, ToStatus: to, NextReviewerUserID: next, Notes: d.Notes,
			ActorUserID: d.Actor.UserID, RolloutMode: string(mode), Applied: mode.Applies(), BeforeVersion: version, AfterVersion: after, OccurredAt: at})
		if err != nil {
			return err
		}
		if err := emit(ctx, tx, commandID, api.EventHRRequestDecided, payload, d.CorrelationID, now); err != nil {
			return err
		}
		out = Result{Status: result.Success, StatusName: result.Success.String(), Reasons: []reasoncodes.Code{}, CommandID: commandID,
			RequestID: d.RequestID, RequestStatus: to, Mode: mode, Applied: mode.Applies(), Version: version, WritebackState: WritebackNotRequested}
		if after != nil {
			out.Version, out.WritebackState = *after, WritebackPending
		} else {
			out.RequestStatus = status // shadow: nothing changed
		}
		return complete(ctx, tx, req, out, now)
	})
	return finish(out, err)
}

// ---- shared transaction steps ----------------------------------------------

func acquire(ctx context.Context, tx db.Tx, req idempotency.Request, mode rollout.Mode, now timeutil.Instant, out *Result) (bool, error) {
	acq, err := idempotency.Acquire(ctx, tx, req, now)
	if err != nil {
		return true, err
	}
	switch acq.State {
	case idempotency.Replay:
		if err := json.Unmarshal(acq.Result, out); err != nil {
			return true, err
		}
		out.Replayed = true
		out.Status, _ = result.ParseStatus(out.StatusName)
		return true, nil
	case idempotency.Conflict:
		return true, stop(result.Conflict, mode, reasoncodes.ContractInvalid)
	case idempotency.InProgress:
		return true, stop(result.Unknown, mode, reasoncodes.OutcomeUnknown)
	}
	return false, nil
}

func record(ctx context.Context, tx db.Tx, commandID, typ, subject, key string, sum []byte, mode rollout.Mode, intent []byte, a Actor, corr string, at any) error {
	_, err := tx.Exec(ctx, `INSERT INTO organization.command_record (command_id, command_type, subject_ref, idempotency_key, request_sha256,
		rollout_mode, applied, intent, actor_principal, actor_user_id, correlation_id, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		commandID, typ, subject, key, sum, string(mode), mode.Applies(), intent, a.Principal, a.UserID, corr, at)
	return err
}

func emit(ctx context.Context, tx db.Tx, commandID, typ string, payload []byte, corr string, now timeutil.Instant) error {
	return outbox.Append(ctx, tx, outbox.Event{ID: ids.Stable("organization.outbox", commandID), Type: typ, Version: 1, Payload: payload,
		OccurredAt: now, CorrelationID: &corr}, now)
}

func complete(ctx context.Context, tx db.Tx, req idempotency.Request, out Result, now timeutil.Instant) error {
	stored, err := json.Marshal(out)
	if err != nil {
		return err
	}
	return idempotency.Complete(ctx, tx, req, stored, now)
}

func finish(out Result, err error) (Result, error) {
	var refused errRefused
	if errors.As(err, &refused) {
		return refused.r, nil
	}
	if err != nil {
		return Result{}, err
	}
	return out, nil
}
