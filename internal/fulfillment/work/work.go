// Package work is fulfillment's Slice 14 operational execution: one work
// case per activated SubscriptionItem (created by the inbox consumer from
// biawin's SubscriptionItemActivated event, in a fulfillment-owned
// transaction) and the evidenced, non-financial commands:
//
//	AssignCase      a CURRENT supervisor of the service assigns a CURRENT
//	                executor of the same service (legacy: item manager
//	                assigns an expert, class-sn-projects.php:2281-2292)
//	RecordAction    the assignee records NO_ANSWER / FOLLOW_UP / NOTE
//	                (class-sn-operations-execution.php:515-517,
//	                class-sn-projects.php:2571)
//	AttachEvidence  the assignee attaches evidence METADATA of a type the
//	                service version requires (form submission evidence,
//	                class-sn-operations-execution.php:586-602)
//	CompleteCase    the assignee records EXECUTED / NOT_EXECUTED (reason
//	                required) for ONE service (form_execute / form_skip,
//	                class-sn-operations-execution.php:527-529)
//
// Authority is per-service row scope from the catalog (never a role tier,
// never the HR hierarchy: the legacy "direct report" rule is not ported).
// Every check runs again inside the fulfillment transaction after the case
// row is locked. A refused command changes nothing and stores nothing.
// Blocked: cancel (OPD-04 refund implications), reopen / correction (no
// evidenced authority), submit-for-review (OPD-08), reassigning a case
// already in progress (OPD-05 handover), completion of a service whose
// financial behaviour is not NONE (D-35 / D-38) or that needs provider
// confirmation. No command touches billing, payments, finance, wallets or
// commission, and none calls a provider.
package work

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"

	biawinapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/api"
	catalogapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/catalog/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/idempotency"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/inbox"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/outbox"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/relay"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

// Consumer is the inbox consumer of item activation events.
const Consumer = "fulfillment.work"

// Command types.
const (
	TypeAssign   = "fulfillment.assign_case"
	TypeAction   = "fulfillment.record_action"
	TypeEvidence = "fulfillment.attach_evidence"
	TypeComplete = "fulfillment.complete_case"
)

var (
	// ErrInvalid reports a malformed command (400).
	ErrInvalid = errors.New("fulfillment/work: invalid command")
	// ErrNotFound reports an unknown case (404).
	ErrNotFound = errors.New("fulfillment/work: case not found")
)

func invalid(f string) error { return fmt.Errorf("%w: %s", ErrInvalid, f) }

var (
	uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	keyPattern  = regexp.MustCompile(`^[A-Za-z0-9_-]{8,128}$`)
	typePattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{1,80}$`)
	shaPattern  = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// ProviderConfirmations answers whether an external provider operation
// for a case is confirmed (integration published read). A nil reader means
// no confirmation can ever be proven.
type ProviderConfirmations interface {
	Confirmed(ctx context.Context, caseID string) (bool, error)
}

// Service runs intake and the commands.
type Service struct {
	runner    *db.Runner
	catalog   catalogapi.Reader
	providers ProviderConfirmations
	clock     timeutil.Clock
}

// New returns the service over a fulfillment-owned runner.
func New(runner *db.Runner, catalog catalogapi.Reader, providers ProviderConfirmations, clock timeutil.Clock) (*Service, error) {
	if runner == nil || runner.Module() != fulfillment.Module || catalog == nil || clock == nil {
		return nil, errors.New("fulfillment/work: fulfillment runner, catalog reader and clock are required")
	}
	return &Service{runner: runner, catalog: catalog, providers: providers, clock: clock}, nil
}

// CaseID is the stable id of the work case of an item.
func CaseID(itemID string) string { return ids.Stable("fulfillment.work_case", itemID) }

// IntakeOutcome is the result of one item activation message.
type IntakeOutcome struct {
	Duplicate bool
	Decision  string
	CaseID    *string
	Reasons   []string
}

// OnItemActivated creates the work case of one activated item (inbox
// deduplicated; one case per item, ever). A missing or retired service
// definition blocks intake with its reason; nothing is guessed.
func (s *Service) OnItemActivated(ctx context.Context, rec outbox.Record) (IntakeOutcome, error) {
	var out IntakeOutcome
	if rec.Type != biawinapi.EventItemActivated {
		return out, invalid("event type " + rec.Type)
	}
	var ev biawinapi.SubscriptionItemActivated
	if err := json.Unmarshal(rec.Payload, &ev); err != nil {
		return out, invalid("payload")
	}
	if !uuidPattern.MatchString(ev.ItemID) || !uuidPattern.MatchString(ev.SubscriptionID) || !uuidPattern.MatchString(ev.CustomerID) {
		return out, invalid("item / subscription / customer id")
	}
	sv, ok, err := s.catalog.Service(ctx, ev.ServiceKey, ev.ServiceVersion)
	if err != nil {
		return out, err
	}
	reasons := []string{}
	switch {
	case !ok:
		reasons = append(reasons, api.BlockServiceMissing)
	case sv.Status != "ACTIVE" || sv.WorkflowKey != catalogapi.WorkflowOperationalTaskV1:
		reasons = append(reasons, api.BlockServiceMissing)
	}
	err = s.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		now := s.clock.Now()
		at := now.Time().UTC()
		res, err := inbox.Consume(ctx, tx, Consumer, rec.ID, now, func(ctx context.Context) error {
			rj, _ := json.Marshal(reasons)
			if len(reasons) > 0 {
				out.Decision, out.Reasons = "BLOCKED", reasons
				_, err := tx.Exec(ctx, `INSERT INTO fulfillment.work_intake (message_id, item_id, subscription_id, service_key, service_version, decision,
					reasons, case_id, correlation_id, causation_id, received_at) VALUES ($1,$2,$3,$4,$5,'BLOCKED',$6,NULL,$7,$8,$9)`,
					rec.ID, ev.ItemID, ev.SubscriptionID, ev.ServiceKey, ev.ServiceVersion, rj, ev.CorrelationID, ev.CausationID, at)
				return err
			}
			caseID := CaseID(ev.ItemID)
			out.CaseID = &caseID
			// case_id is CaseID(item_id), so one item conflicts on BOTH unique
			// keys (case_id, item_id); either conflict means "this item already
			// has its case". No arbiter: a concurrent redelivery that reaches the
			// primary key first must answer EXISTING_CASE, not fail with 23505.
			tag, err := tx.Exec(ctx, `INSERT INTO fulfillment.work_case (case_id, item_id, subscription_id, customer_id, service_key, service_version,
				workflow_key, origin_mode, state, assignee_user_id, version, source_message_id, correlation_id, created_at, updated_at)
				VALUES ($1,$2,$3,$4,$5,$6,$7,'SHADOW_COMPARE','QUEUED',NULL,1,$8,$9,$10,$10) ON CONFLICT DO NOTHING`,
				caseID, ev.ItemID, ev.SubscriptionID, ev.CustomerID, ev.ServiceKey, ev.ServiceVersion, sv.WorkflowKey, rec.ID, ev.CorrelationID, at)
			if err != nil {
				return err
			}
			decision := "CASE_CREATED"
			if tag.RowsAffected() == 0 {
				decision = "EXISTING_CASE"
			}
			out.Decision, out.Reasons = decision, []string{}
			if _, err := tx.Exec(ctx, `INSERT INTO fulfillment.work_intake (message_id, item_id, subscription_id, service_key, service_version, decision,
				reasons, case_id, correlation_id, causation_id, received_at) VALUES ($1,$2,$3,$4,$5,$6,'[]',$7,$8,$9,$10)`,
				rec.ID, ev.ItemID, ev.SubscriptionID, ev.ServiceKey, ev.ServiceVersion, decision, caseID, ev.CorrelationID, ev.CausationID, at); err != nil {
				return err
			}
			if decision != "CASE_CREATED" {
				return nil
			}
			sup, err := s.catalog.Supervisors(ctx, ev.ServiceKey)
			if err != nil {
				return err
			}
			supj, _ := json.Marshal(sup)
			detail, _ := json.Marshal(map[string]any{"item_id": ev.ItemID, "subscription_id": ev.SubscriptionID, "service_key": ev.ServiceKey,
				"service_version": ev.ServiceVersion, "causation_id": ev.CausationID})
			if _, err := tx.Exec(ctx, `INSERT INTO fulfillment.work_event (event_id, case_id, command_id, kind, detail, actor_user_id, supervisors,
				before_version, after_version, correlation_id, occurred_at) VALUES ($1,$2,NULL,'CREATED',$3,NULL,$4,NULL,1,$5,$6)`,
				ids.Stable("fulfillment.work_event", caseID, "created"), caseID, detail, supj, ev.CorrelationID, at); err != nil {
				return err
			}
			payload, _ := json.Marshal(map[string]any{"case_id": caseID, "item_id": ev.ItemID, "subscription_id": ev.SubscriptionID,
				"service_key": ev.ServiceKey, "rollout_mode": string(rollout.Shadow), "correlation_id": ev.CorrelationID, "causation_id": rec.ID})
			corr := ev.CorrelationID
			return outbox.Append(ctx, tx, outbox.Event{ID: ids.Stable("fulfillment.outbox.created", caseID), Type: api.EventWorkCaseCreated, Version: 1,
				Payload: payload, OccurredAt: now, CorrelationID: &corr}, now)
		})
		if err != nil {
			return err
		}
		out.Duplicate = res == inbox.Duplicate
		return nil
	})
	return out, err
}

// Actor is who issues a command.
type Actor struct {
	Principal string
	UserID    string
}

// Command is one fulfillment work command.
type Command struct {
	Type            string
	CaseID          string
	ExpectedVersion int64
	IdempotencyKey  string
	CorrelationID   string
	Actor           Actor
	// AssignCase
	AssigneeUserID string
	// RecordAction
	Action     string
	FollowUpAt *time.Time
	Note       *string
	// AttachEvidence
	EvidenceType string
	ContentSHA   string
	Label        string
	// CompleteCase
	Outcome string
	Reason  *string
}

// Result is the command outcome (also what an idempotent replay returns).
type Result struct {
	Status      result.Status      `json:"-"`
	StatusName  string             `json:"status"`
	Reasons     []reasoncodes.Code `json:"reasons"`
	BlockedBy   []string           `json:"blocked_by,omitempty"`
	CommandID   string             `json:"command_id,omitempty"`
	Mode        rollout.Mode       `json:"rollout_mode"`
	Applied     bool               `json:"applied"`
	Replayed    bool               `json:"-"`
	CaseVersion int64              `json:"case_version,omitempty"`
	State       string             `json:"state,omitempty"`
	Current     int64              `json:"-"`
}

func refuse(status result.Status, mode rollout.Mode, code reasoncodes.Code, blockedBy ...string) Result {
	return Result{Status: status, StatusName: status.String(), Reasons: []reasoncodes.Code{code}, BlockedBy: blockedBy, Mode: mode}
}

type errRefused struct{ r Result }

func (e errRefused) Error() string { return "refused: " + e.r.StatusName }

func validate(c Command) error {
	if !uuidPattern.MatchString(c.CaseID) {
		return invalid("case_id")
	}
	if !keyPattern.MatchString(c.IdempotencyKey) {
		return invalid("idempotency_key")
	}
	if c.CorrelationID == "" || len(c.CorrelationID) > 128 {
		return invalid("correlation_id")
	}
	if c.ExpectedVersion < 1 {
		return invalid("expected_version")
	}
	switch c.Type {
	case TypeAssign:
		if !uuidPattern.MatchString(c.AssigneeUserID) {
			return invalid("assignee_user_id")
		}
	case TypeAction:
		switch c.Action {
		case api.ActionNoAnswer:
		case api.ActionFollowUp:
			if c.FollowUpAt == nil {
				return invalid("follow_up_at (required for FOLLOW_UP)")
			}
		case api.ActionNote:
			if c.Note == nil {
				return invalid("note (required for NOTE)")
			}
		default:
			return invalid("action")
		}
		if c.Note != nil && (len(*c.Note) == 0 || len(*c.Note) > 2000) {
			return invalid("note")
		}
	case TypeEvidence:
		if !typePattern.MatchString(c.EvidenceType) || !shaPattern.MatchString(c.ContentSHA) || len(c.Label) == 0 || len(c.Label) > 200 {
			return invalid("evidence")
		}
	case TypeComplete:
		switch c.Outcome {
		case catalogapi.OutcomeExecuted:
			if c.Reason != nil {
				return invalid("reason (only for NOT_EXECUTED)")
			}
		case catalogapi.OutcomeNotExecuted:
			if c.Reason == nil || len(*c.Reason) == 0 || len(*c.Reason) > 1000 {
				return invalid("reason (required for NOT_EXECUTED)")
			}
		default:
			return invalid("outcome")
		}
	default:
		return invalid("command type")
	}
	return nil
}

func canonical(c Command) []byte {
	var fu *string
	if c.FollowUpAt != nil {
		v := c.FollowUpAt.UTC().Format(time.RFC3339)
		fu = &v
	}
	b, _ := json.Marshal(struct {
		T, Case, Assignee, Action, EvType, SHA, Label, Outcome, Actor string
		Expected                                                      int64
		FollowUp, Note, Reason                                        *string
	}{c.Type, c.CaseID, c.AssigneeUserID, c.Action, c.EvidenceType, c.ContentSHA, c.Label, c.Outcome, c.Actor.UserID, c.ExpectedVersion, fu, c.Note, c.Reason})
	return b
}

type caseRow struct {
	state, serviceKey, origin string
	serviceVersion            int
	assignee                  *string
	version                   int64
}

// Execute runs one command in mode.
func (s *Service) Execute(ctx context.Context, mode rollout.Mode, c Command) (Result, error) {
	if err := validate(c); err != nil {
		return Result{}, err
	}
	if mode == rollout.Disabled || !mode.Valid() {
		return refuse(result.Blocked, rollout.Disabled, reasoncodes.FeatureDisabled), nil
	}
	if c.Actor.UserID == "" || c.Actor.Principal == "" || !uuidPattern.MatchString(c.Actor.UserID) {
		return refuse(result.Unauthorized, mode, reasoncodes.RowScopeDenied), nil
	}
	sum := sha256.Sum256(canonical(c))
	req := idempotency.Request{Scope: c.Type, Key: c.IdempotencyKey, Hash: idempotency.Hash(sum)}
	commandID := ids.Stable(c.Type, c.IdempotencyKey)
	var out Result
	err := s.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		now := s.clock.Now()
		at := now.Time().UTC()
		acq, err := idempotency.Acquire(ctx, tx, req, now)
		if err != nil {
			return err
		}
		switch acq.State {
		case idempotency.Replay:
			if err := json.Unmarshal(acq.Result, &out); err != nil {
				return err
			}
			out.Replayed = true
			out.Status, _ = result.ParseStatus(out.StatusName)
			return nil
		case idempotency.Conflict:
			return errRefused{refuse(result.Conflict, mode, reasoncodes.ContractInvalid)}
		case idempotency.InProgress:
			return errRefused{refuse(result.Unknown, mode, reasoncodes.OutcomeUnknown)}
		}
		var cr caseRow
		err = tx.QueryRow(ctx, `SELECT state, service_key, service_version, origin_mode, assignee_user_id::text, version
			FROM fulfillment.work_case WHERE case_id = $1 FOR UPDATE`, c.CaseID).Scan(&cr.state, &cr.serviceKey, &cr.serviceVersion, &cr.origin, &cr.assignee, &cr.version)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		sv, ok, err := s.catalog.Service(ctx, cr.serviceKey, cr.serviceVersion)
		if err != nil {
			return err
		}
		if !ok {
			return errRefused{refuse(result.Blocked, mode, reasoncodes.PolicyUnresolved, api.BlockServiceMissing)}
		}
		next, detail, refusal, err := s.decide(ctx, tx, c, cr, sv)
		if err != nil {
			return err
		}
		if refusal != nil {
			refusal.Mode = mode
			return errRefused{*refusal}
		}
		if cr.version != c.ExpectedVersion {
			r := refuse(result.Conflict, mode, reasoncodes.StaleState)
			r.Current = cr.version
			return errRefused{r}
		}
		intent, _ := json.Marshal(detail)
		var after *int64
		if mode.Applies() {
			assignee := cr.assignee
			if c.Type == TypeAssign {
				assignee = &c.AssigneeUserID
			}
			tag, err := tx.Exec(ctx, `UPDATE fulfillment.work_case SET state = $3, assignee_user_id = $4, version = version + 1, updated_at = $5
				WHERE case_id = $1 AND version = $2`, c.CaseID, cr.version, next, assignee, at)
			if err != nil {
				return err
			}
			if tag.RowsAffected() != 1 {
				return errRefused{refuse(result.Conflict, mode, reasoncodes.StaleState)}
			}
			v := cr.version + 1
			after = &v
			if err := s.history(ctx, tx, c, cr, commandID, detail, v, at); err != nil {
				return err
			}
			if next == api.WorkCompleted || next == api.WorkNotExecuted {
				payload, _ := json.Marshal(map[string]any{"case_id": c.CaseID, "outcome": c.Outcome, "service_key": cr.serviceKey,
					"command_id": commandID, "rollout_mode": string(mode), "correlation_id": c.CorrelationID})
				corr := c.CorrelationID
				if err := outbox.Append(ctx, tx, outbox.Event{ID: ids.Stable("fulfillment.outbox.completed", commandID), Type: api.EventWorkCaseCompleted,
					Version: 1, Payload: payload, OccurredAt: now, CorrelationID: &corr}, now); err != nil {
					return err
				}
			}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO fulfillment.work_command_record (command_id, command_type, case_id, idempotency_key, request_sha256,
			rollout_mode, applied, intent, before_version, after_version, actor_principal, actor_user_id, correlation_id, occurred_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, commandID, c.Type, c.CaseID, c.IdempotencyKey, sum[:], string(mode), mode.Applies(),
			intent, cr.version, after, c.Actor.Principal, c.Actor.UserID, c.CorrelationID, at); err != nil {
			return err
		}
		out = Result{Status: result.Success, StatusName: result.Success.String(), Reasons: []reasoncodes.Code{}, CommandID: commandID, Mode: mode,
			Applied: mode.Applies(), CaseVersion: cr.version, State: cr.state}
		if after != nil {
			out.CaseVersion, out.State = *after, next
		}
		stored, err := json.Marshal(out)
		if err != nil {
			return err
		}
		return idempotency.Complete(ctx, tx, req, stored, now)
	})
	var refused errRefused
	if errors.As(err, &refused) {
		return refused.r, nil
	}
	if err != nil {
		return Result{}, err
	}
	return out, nil
}

// decide applies the authority, state and value rules (inside the tx,
// after the row lock) and returns the next state.
func (s *Service) decide(ctx context.Context, tx db.Tx, c Command, cr caseRow, sv catalogapi.ServiceVersion) (string, map[string]any, *Result, error) {
	deny := func(st result.Status, code reasoncodes.Code, by ...string) (string, map[string]any, *Result, error) {
		r := refuse(st, "", code, by...)
		return "", nil, &r, nil
	}
	if cr.origin != "SHADOW_COMPARE" {
		return deny(result.Blocked, reasoncodes.PolicyUnresolved, api.BlockUnprovenPath)
	}
	inScope := func(user, kind string) (bool, error) { return s.catalog.InScope(ctx, cr.serviceKey, user, kind) }
	isAssignee := cr.assignee != nil && *cr.assignee == c.Actor.UserID
	switch c.Type {
	case TypeAssign:
		ok, err := inScope(c.Actor.UserID, catalogapi.ScopeSupervisor)
		if err != nil {
			return "", nil, nil, err
		}
		if !ok {
			return deny(result.Unauthorized, reasoncodes.RowScopeDenied)
		}
		switch cr.state {
		case api.WorkQueued, api.WorkAssigned:
		case api.WorkInProgress:
			return deny(result.Blocked, reasoncodes.PolicyUnresolved, api.BlockOPD05)
		default:
			return deny(result.Conflict, reasoncodes.StaleState)
		}
		ok, err = inScope(c.AssigneeUserID, catalogapi.ScopeExecutor)
		if err != nil {
			return "", nil, nil, err
		}
		if !ok {
			return deny(result.Blocked, reasoncodes.RecipientIneligible)
		}
		return api.WorkAssigned, map[string]any{"assignee_user_id": c.AssigneeUserID, "previous_assignee": cr.assignee}, nil, nil
	}
	// Executor commands: the current assignee who still holds the executor
	// scope of THIS service. A supervisor role never substitutes.
	if !isAssignee {
		return deny(result.Unauthorized, reasoncodes.RowScopeDenied)
	}
	ok, err := inScope(c.Actor.UserID, catalogapi.ScopeExecutor)
	if err != nil {
		return "", nil, nil, err
	}
	if !ok {
		return deny(result.Unauthorized, reasoncodes.RowScopeDenied)
	}
	if cr.state != api.WorkAssigned && cr.state != api.WorkInProgress {
		return deny(result.Conflict, reasoncodes.StaleState)
	}
	switch c.Type {
	case TypeAction:
		d := map[string]any{"action": c.Action, "note": c.Note}
		if c.FollowUpAt != nil {
			d["follow_up_at"] = c.FollowUpAt.UTC().Format(time.RFC3339)
		}
		return api.WorkInProgress, d, nil, nil
	case TypeEvidence:
		allowed := false
		for _, e := range sv.Evidence {
			allowed = allowed || e.Type == c.EvidenceType
		}
		if !allowed {
			return deny(result.Blocked, reasoncodes.ContractInvalid)
		}
		sha, _ := hex.DecodeString(c.ContentSHA)
		var n int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM fulfillment.work_evidence WHERE case_id = $1 AND evidence_type = $2 AND content_sha256 = $3`,
			c.CaseID, c.EvidenceType, sha).Scan(&n); err != nil {
			return "", nil, nil, err
		}
		if n > 0 {
			return deny(result.Conflict, reasoncodes.ContractInvalid)
		}
		return api.WorkInProgress, map[string]any{"evidence_type": c.EvidenceType, "content_sha256": c.ContentSHA, "label": c.Label}, nil, nil
	case TypeComplete:
		if sv.FinancialBehaviour != catalogapi.FinancialNone {
			return deny(result.Blocked, reasoncodes.PolicyUnresolved, api.BlockD35, api.BlockD38)
		}
		approved := false
		for _, o := range sv.CompletionOutcomes {
			approved = approved || o == c.Outcome
		}
		if !approved {
			return deny(result.Blocked, reasoncodes.ContractInvalid)
		}
		if c.Outcome == catalogapi.OutcomeExecuted {
			for _, e := range sv.Evidence {
				if !e.Required {
					continue
				}
				var n int
				if err := tx.QueryRow(ctx, `SELECT count(*) FROM fulfillment.work_evidence WHERE case_id = $1 AND evidence_type = $2`, c.CaseID, e.Type).Scan(&n); err != nil {
					return "", nil, nil, err
				}
				if n == 0 {
					return deny(result.Blocked, reasoncodes.CoverageIncomplete, api.BlockEvidenceMissing)
				}
			}
			if sv.IntegrationProvider != catalogapi.ProviderNone {
				confirmed := false
				if s.providers != nil {
					var err error
					if confirmed, err = s.providers.Confirmed(ctx, c.CaseID); err != nil {
						return "", nil, nil, err
					}
				}
				if !confirmed {
					return deny(result.Blocked, reasoncodes.DependencyUnknown, api.BlockProviderConfirmation)
				}
			}
		}
		next := api.WorkCompleted
		if c.Outcome == catalogapi.OutcomeNotExecuted {
			next = api.WorkNotExecuted
		}
		return next, map[string]any{"outcome": c.Outcome, "reason": c.Reason}, nil, nil
	}
	return deny(result.Blocked, reasoncodes.ContractInvalid)
}

func (s *Service) history(ctx context.Context, tx db.Tx, c Command, cr caseRow, commandID string, detail map[string]any, after int64, at time.Time) error {
	kind := map[string]string{TypeAssign: "ASSIGNED", TypeAction: "ACTION_RECORDED", TypeEvidence: "EVIDENCE_ATTACHED", TypeComplete: "COMPLETED"}[c.Type]
	sup, err := s.catalog.Supervisors(ctx, cr.serviceKey)
	if err != nil {
		return err
	}
	supj, _ := json.Marshal(sup)
	dj, _ := json.Marshal(detail)
	if _, err := tx.Exec(ctx, `INSERT INTO fulfillment.work_event (event_id, case_id, command_id, kind, detail, actor_user_id, supervisors,
		before_version, after_version, correlation_id, occurred_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		ids.Stable("fulfillment.work_event", commandID), c.CaseID, commandID, kind, dj, c.Actor.UserID, supj, cr.version, after, c.CorrelationID, at); err != nil {
		return err
	}
	switch c.Type {
	case TypeAssign:
		_, err = tx.Exec(ctx, `INSERT INTO fulfillment.work_assignment (assignment_id, case_id, command_id, assignee_user_id, previous_assignee,
			assigned_by_user_id, occurred_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, ids.Stable("fulfillment.work_assignment", commandID), c.CaseID, commandID,
			c.AssigneeUserID, cr.assignee, c.Actor.UserID, at)
	case TypeEvidence:
		sha, _ := hex.DecodeString(c.ContentSHA)
		_, err = tx.Exec(ctx, `INSERT INTO fulfillment.work_evidence (evidence_id, case_id, command_id, evidence_type, content_sha256, label,
			submitted_by, occurred_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, ids.Stable("fulfillment.work_evidence", commandID), c.CaseID, commandID,
			c.EvidenceType, sha, c.Label, c.Actor.UserID, at)
	}
	return err
}

// Blocked commands: each validates nothing and changes nothing; it names
// the unresolved decision so the UI and API can show why.
var blocked = map[string][]string{
	"fulfillment.cancel_case":      {api.BlockOPD04},
	"fulfillment.reopen_case":      {api.BlockUnprovenPath},
	"fulfillment.submit_review":    {api.BlockOPD08},
	"fulfillment.reassign_started": {api.BlockOPD05},
	"biawin.wallet_charge":         {api.BlockD38},
	"biawin.activate_production":   {api.BlockD35},
}

// BlockedCommand returns the fixed refusal of a decision-blocked command.
func BlockedCommand(name string) (Result, bool) {
	by, ok := blocked[name]
	if !ok {
		return Result{}, false
	}
	return refuse(result.Blocked, rollout.Disabled, reasoncodes.PolicyUnresolved, by...), true
}

// BlockedCommands lists the decision-blocked command names.
func BlockedCommands() map[string][]string {
	out := map[string][]string{}
	for k, v := range blocked {
		out[k] = append([]string(nil), v...)
	}
	return out
}

// IgnoredSubscriptionActivated documents why the biawin subscription-level
// event is dispatched without delivery: fulfillment intake acts per item.
const IgnoredSubscriptionActivated = "subscription-level event; fulfillment intake acts per item on subscription_item_activated (Slice 14)"

// IntakeRoute is the complete relay route of the biawin outbox (R2): the
// item-activated event creates the work case (inbox-deduplicated); the
// subscription-level event is declared non-actionable with its reason. Any
// other biawin event type fails visibly in the biawin outbox.
func (s *Service) IntakeRoute() relay.Route {
	return relay.Route{
		Name: Consumer,
		Handlers: map[string]relay.Handler{biawinapi.EventItemActivated: func(ctx context.Context, rec outbox.Record) error {
			_, err := s.OnItemActivated(ctx, rec)
			return err
		}},
		Ignore: map[string]string{biawinapi.EventSubscriptionActivated: IgnoredSubscriptionActivated},
	}
}
