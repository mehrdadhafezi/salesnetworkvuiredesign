package work

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
)

// CaseView is one work case as read (READ ONLY transaction).
type CaseView struct {
	CaseID, ItemID, SubscriptionID, CustomerID, ServiceKey, WorkflowKey, Origin, State, Correlation string
	ServiceVersion                                                                                  int
	Assignee                                                                                        *string
	Version                                                                                         int64
	CreatedAt, UpdatedAt                                                                            time.Time
}

// EventView is one history event.
type EventView struct {
	EventID, Kind, Correlation string
	CommandID, Actor           *string
	Detail                     map[string]any
	Supervisors                []string
	Before                     *int64
	After                      int64
	OccurredAt                 time.Time
}

// EvidenceView is one evidence metadata record.
type EvidenceView struct {
	EvidenceID, Type, SHA, Label, SubmittedBy string
	OccurredAt                                time.Time
}

// Filter narrows the queue. Services is the caller's CURRENT scope (never
// empty-means-all: an empty scope reads nothing).
type Filter struct {
	Services       []string
	ServiceKey     *string
	State          *string
	SubscriptionID *string
	Cursor         *string
	Limit          int
}

const caseCols = `case_id::text, item_id::text, subscription_id::text, customer_id::text, service_key, service_version, workflow_key, origin_mode, state,
	assignee_user_id::text, version, correlation_id, created_at, updated_at`

func scanCase(r pgx.Row) (CaseView, error) {
	var c CaseView
	err := r.Scan(&c.CaseID, &c.ItemID, &c.SubscriptionID, &c.CustomerID, &c.ServiceKey, &c.ServiceVersion, &c.WorkflowKey, &c.Origin, &c.State,
		&c.Assignee, &c.Version, &c.Correlation, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (s *Service) read(ctx context.Context, fn func(ctx context.Context, tx db.Tx) error) error {
	return s.runner.InTx(ctx, db.TxOptions{ReadOnly: true, Isolation: db.RepeatableRead}, fn)
}

// Queue lists work cases of the caller's services only.
func (s *Service) Queue(ctx context.Context, f Filter) ([]CaseView, *string, error) {
	out := []CaseView{}
	if len(f.Services) == 0 {
		return out, nil, nil
	}
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	after := ""
	if f.Cursor != nil && *f.Cursor != "" {
		b, err := base64.RawURLEncoding.DecodeString(*f.Cursor)
		if err != nil || len(b) > 64 {
			return nil, nil, invalid("cursor")
		}
		after = string(b)
	}
	if f.SubscriptionID != nil && !uuidPattern.MatchString(*f.SubscriptionID) {
		return nil, nil, invalid("subscription_id")
	}
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		rows, err := tx.Query(ctx, `SELECT `+caseCols+` FROM fulfillment.work_case WHERE service_key = ANY($1) AND case_id::text > $2
			AND ($3::text IS NULL OR service_key = $3) AND ($4::text IS NULL OR state = $4) AND ($5::uuid IS NULL OR subscription_id = $5)
			ORDER BY case_id LIMIT $6`, f.Services, after, f.ServiceKey, f.State, f.SubscriptionID, limit+1)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			c, err := scanCase(rows)
			if err != nil {
				return err
			}
			out = append(out, c)
		}
		return rows.Err()
	})
	var next *string
	if len(out) > limit {
		out = out[:limit]
		n := base64.RawURLEncoding.EncodeToString([]byte(out[len(out)-1].CaseID))
		next = &n
	}
	return out, next, err
}

// Detail reads one case with its history and evidence metadata.
func (s *Service) Detail(ctx context.Context, caseID string) (CaseView, []EventView, []EvidenceView, error) {
	var c CaseView
	evs := []EventView{}
	evid := []EvidenceView{}
	if !uuidPattern.MatchString(caseID) {
		return c, nil, nil, invalid("case_id")
	}
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		c, err = scanCase(tx.QueryRow(ctx, `SELECT `+caseCols+` FROM fulfillment.work_case WHERE case_id = $1`, caseID))
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT event_id::text, kind, correlation_id, command_id::text, actor_user_id::text, detail, supervisors, before_version,
			after_version, occurred_at FROM fulfillment.work_event WHERE case_id = $1 ORDER BY occurred_at, after_version, event_id LIMIT 500`, caseID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var e EventView
			var d, sup []byte
			if err := rows.Scan(&e.EventID, &e.Kind, &e.Correlation, &e.CommandID, &e.Actor, &d, &sup, &e.Before, &e.After, &e.OccurredAt); err != nil {
				rows.Close()
				return err
			}
			_ = json.Unmarshal(d, &e.Detail)
			_ = json.Unmarshal(sup, &e.Supervisors)
			if e.Supervisors == nil {
				e.Supervisors = []string{}
			}
			evs = append(evs, e)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		rows, err = tx.Query(ctx, `SELECT evidence_id::text, evidence_type, content_sha256, label, submitted_by::text, occurred_at FROM fulfillment.work_evidence
			WHERE case_id = $1 ORDER BY occurred_at, evidence_id LIMIT 200`, caseID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var v EvidenceView
			var sha []byte
			if err := rows.Scan(&v.EvidenceID, &v.Type, &sha, &v.Label, &v.SubmittedBy, &v.OccurredAt); err != nil {
				return err
			}
			v.SHA = hex.EncodeToString(sha)
			evid = append(evid, v)
		}
		return rows.Err()
	})
	return c, evs, evid, err
}
