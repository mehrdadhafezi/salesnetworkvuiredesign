package hrrequest

import (
	"context"
	"encoding/base64"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
)

// Request is the read view of one HR request.
type Request struct {
	RequestID, RequestType, Status, SubjectUserID, RequestedByUserID, Reason string
	SubjectParentUserID, TargetParentUserID, CurrentReviewerUserID           *string
	ReviewerNotes                                                            *string
	Version                                                                  int64
	CreatedAt, UpdatedAt                                                     time.Time
}

// Event is one HR request history event.
type Event struct {
	EventID, CommandID, Action, ToStatus, ActorUserID, CorrelationID string
	FromStatus, NextReviewerUserID, Notes                            *string
	AfterVersion                                                     int64
	OccurredAt                                                       time.Time
}

const requestCols = `request_id::text, request_type, status, subject_user_id::text, requested_by_user_id::text, reason,
	subject_parent_user_id::text, target_parent_user_id::text, current_reviewer_user_id::text, reviewer_notes, version, created_at, updated_at`

func scanRequest(r pgx.Row) (Request, error) {
	var q Request
	err := r.Scan(&q.RequestID, &q.RequestType, &q.Status, &q.SubjectUserID, &q.RequestedByUserID, &q.Reason, &q.SubjectParentUserID,
		&q.TargetParentUserID, &q.CurrentReviewerUserID, &q.ReviewerNotes, &q.Version, &q.CreatedAt, &q.UpdatedAt)
	return q, err
}

// visible is the row scope of an HR request read: its requester, its
// current reviewer, or an actor of its history. Rank never widens it.
const visible = `(requested_by_user_id = $1 OR current_reviewer_user_id = $1 OR EXISTS (SELECT 1 FROM organization.hr_request_event e
	WHERE e.request_id = r.request_id AND e.actor_user_id = $1))`

// List returns the requests visible to actor (newest first), cursor paged.
func (s *Service) List(ctx context.Context, actorUserID string, cursor *string, limit int) ([]Request, *string, error) {
	if !uuidPattern.MatchString(actorUserID) {
		return nil, nil, nil
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	var after *string
	if cursor != nil {
		b, err := base64.RawURLEncoding.DecodeString(*cursor)
		if err != nil || !uuidPattern.MatchString(string(b)) {
			return nil, nil, invalid("cursor")
		}
		v := string(b)
		after = &v
	}
	var out []Request
	err := s.runner.InTx(ctx, db.TxOptions{ReadOnly: true, Isolation: db.RepeatableRead}, func(ctx context.Context, tx db.Tx) error {
		rows, err := tx.Query(ctx, `SELECT `+requestCols+` FROM organization.hr_request r WHERE `+visible+`
			AND ($2::uuid IS NULL OR request_id > $2) ORDER BY request_id LIMIT $3`, actorUserID, after, limit+1)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			q, err := scanRequest(rows)
			if err != nil {
				return err
			}
			out = append(out, q)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, nil, err
	}
	var next *string
	if len(out) > limit {
		out = out[:limit]
		c := base64.RawURLEncoding.EncodeToString([]byte(out[limit-1].RequestID))
		next = &c
	}
	return out, next, nil
}

// Get returns one request visible to actor with its append-only history.
func (s *Service) Get(ctx context.Context, actorUserID, requestID string) (Request, []Event, error) {
	if !uuidPattern.MatchString(requestID) {
		return Request{}, nil, invalid("request_id")
	}
	if !uuidPattern.MatchString(actorUserID) {
		return Request{}, nil, ErrNotFound
	}
	var q Request
	var hist []Event
	err := s.runner.InTx(ctx, db.TxOptions{ReadOnly: true, Isolation: db.RepeatableRead}, func(ctx context.Context, tx db.Tx) error {
		var err error
		q, err = scanRequest(tx.QueryRow(ctx, `SELECT `+requestCols+` FROM organization.hr_request r WHERE request_id = $2 AND `+visible,
			actorUserID, requestID))
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound // invisible and absent look the same
		}
		if err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT event_id::text, command_id::text, action, to_status, actor_user_id::text, correlation_id, from_status,
			next_reviewer_user_id::text, notes, after_version, occurred_at FROM organization.hr_request_event WHERE request_id = $1
			ORDER BY after_version`, requestID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var e Event
			if err := rows.Scan(&e.EventID, &e.CommandID, &e.Action, &e.ToStatus, &e.ActorUserID, &e.CorrelationID, &e.FromStatus,
				&e.NextReviewerUserID, &e.Notes, &e.AfterVersion, &e.OccurredAt); err != nil {
				return err
			}
			hist = append(hist, e)
		}
		return rows.Err()
	})
	return q, hist, err
}

// Blocked lists the actions of a request that exist in legacy but are
// unavailable here, with the decisions blocking them.
func Blocked(q Request) map[string][]string {
	out := map[string][]string{}
	if q.Status == api.HRPendingHR || q.Status == api.HRPendingReview {
		out["apply_final"] = []string{DecisionHRRuntime, DecisionOPD05, DecisionSD04}
	}
	return out
}

// CanDecide reports whether actor may step-approve or reject q now (the
// same row-scope rule the command re-checks at commit).
func CanDecide(q Request, actorUserID string) bool {
	return q.Status == api.HRPendingReview && q.CurrentReviewerUserID != nil && *q.CurrentReviewerUserID == actorUserID
}
