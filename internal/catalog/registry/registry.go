// Package registry is the catalog's recorder (offline administrative job only)
// and its published read implementation. Every record carries an external
// Human Owner approval reference; nothing here decides policy.
package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/catalog"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/catalog/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
)

// ErrInvalid reports a malformed record.
var ErrInvalid = errors.New("catalog: invalid record")

func invalid(f string) error { return fmt.Errorf("%w: %s", ErrInvalid, f) }

var (
	keyPattern  = regexp.MustCompile(`^[a-z][a-z0-9_.-]{1,80}$`)
	uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

// Store is the catalog store over a catalog-owned runner.
type Store struct {
	runner *db.Runner
	clock  timeutil.Clock
}

// New returns the store.
func New(runner *db.Runner, clock timeutil.Clock) (*Store, error) {
	if runner == nil || runner.Module() != catalog.Module || clock == nil {
		return nil, errors.New("catalog: runner must be owned by catalog; clock required")
	}
	return &Store{runner: runner, clock: clock}, nil
}

// Recording is one administrative batch: every row shares the approval
// reference and the recorder identity.
type Recording struct {
	ApprovalRef string               `json:"approval_ref"`
	RecordedBy  string               `json:"recorded_by"`
	Services    []api.ServiceVersion `json:"services"`
	Components  []ComponentRecord    `json:"components"`
	Scopes      []ScopeRecord        `json:"scopes"`
}

// ComponentRecord is one product component.
type ComponentRecord struct {
	ProductKey     string `json:"product_key"`
	ProductVersion int    `json:"product_version"`
	api.Component
}

// ScopeRecord is one scope event.
type ScopeRecord struct {
	ServiceKey string `json:"service_key"`
	UserID     string `json:"user_id"`
	Kind       string `json:"scope_kind"`
	Action     string `json:"action"`
}

func validate(r Recording) error {
	if len(r.ApprovalRef) < 3 || len(r.ApprovalRef) > 200 {
		return invalid("approval_ref (an external Human Owner approval reference is required)")
	}
	if r.RecordedBy == "" || len(r.RecordedBy) > 128 {
		return invalid("recorded_by")
	}
	for _, s := range r.Services {
		if !keyPattern.MatchString(s.ServiceKey) || s.Version < 1 || s.Title == "" || s.WorkflowKey != api.WorkflowOperationalTaskV1 {
			return invalid("service " + s.ServiceKey)
		}
		if len(s.CompletionOutcomes) == 0 {
			return invalid("completion outcomes")
		}
		for _, o := range s.CompletionOutcomes {
			if o != api.OutcomeExecuted && o != api.OutcomeNotExecuted {
				return invalid("completion outcome " + o)
			}
		}
		for _, e := range s.Evidence {
			if !keyPattern.MatchString(e.Type) {
				return invalid("evidence type")
			}
		}
		switch s.FinancialBehaviour {
		case api.FinancialNone, api.FinancialConsumesCredit, api.FinancialChargesLedger:
		default:
			return invalid("financial behaviour")
		}
		switch s.IntegrationProvider {
		case api.ProviderNone, api.ProviderBiawin:
		default:
			return invalid("integration provider")
		}
		if s.Status != "ACTIVE" && s.Status != "RETIRED" {
			return invalid("status")
		}
	}
	for _, c := range r.Components {
		if !keyPattern.MatchString(c.ProductKey) || c.ProductVersion < 1 || c.Seq < 1 {
			return invalid("component")
		}
	}
	for _, s := range r.Scopes {
		if !keyPattern.MatchString(s.ServiceKey) || !uuidPattern.MatchString(s.UserID) || (s.Kind != api.ScopeSupervisor && s.Kind != api.ScopeExecutor) ||
			(s.Action != "GRANT" && s.Action != "REVOKE") {
			return invalid("scope")
		}
	}
	return nil
}

// Record writes one administrative batch in one catalog transaction.
// Existing versions are immutable: re-recording an identical version is a
// no-op, a different content under the same version is refused.
func (s *Store) Record(ctx context.Context, r Recording) error {
	if err := validate(r); err != nil {
		return err
	}
	now := s.clock.Now().Time().UTC()
	return s.runner.InTx(ctx, db.TxOptions{Isolation: db.Serializable}, func(ctx context.Context, tx db.Tx) error {
		for _, v := range r.Services {
			ev, _ := json.Marshal(v.Evidence)
			existing, ok, err := service(ctx, tx, v.ServiceKey, v.Version)
			if err != nil {
				return err
			}
			if ok {
				a, _ := json.Marshal(existing)
				v.ApprovalRef = existing.ApprovalRef
				b, _ := json.Marshal(v)
				if string(a) != string(b) {
					return invalid("service version " + v.ServiceKey + " already exists with different content (versions are immutable)")
				}
				continue
			}
			if _, err := tx.Exec(ctx, `INSERT INTO catalog.service_version (service_key, version, title, workflow_key, evidence_requirements,
				completion_outcomes, financial_behaviour, integration_provider, status, approval_ref, recorded_by, recorded_at)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, v.ServiceKey, v.Version, v.Title, v.WorkflowKey, ev, v.CompletionOutcomes,
				v.FinancialBehaviour, v.IntegrationProvider, v.Status, r.ApprovalRef, r.RecordedBy, now); err != nil {
				return err
			}
		}
		for _, c := range r.Components {
			if _, err := tx.Exec(ctx, `INSERT INTO catalog.subscription_component (product_key, product_version, seq, service_key, service_version,
				approval_ref, recorded_by, recorded_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT DO NOTHING`, c.ProductKey, c.ProductVersion, c.Seq,
				c.ServiceKey, c.ServiceVersion, r.ApprovalRef, r.RecordedBy, now); err != nil {
				return err
			}
		}
		for i, sc := range r.Scopes {
			id := ids.Stable("catalog.scope", r.ApprovalRef, sc.ServiceKey, sc.UserID, sc.Kind, sc.Action, fmt.Sprint(i))
			if _, err := tx.Exec(ctx, `INSERT INTO catalog.service_scope_event (event_id, service_key, user_id, scope_kind, action, approval_ref,
				recorded_by, recorded_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (event_id) DO NOTHING`, id, sc.ServiceKey, sc.UserID, sc.Kind,
				sc.Action, r.ApprovalRef, r.RecordedBy, now); err != nil {
				return err
			}
		}
		return nil
	})
}

func service(ctx context.Context, tx db.Tx, key string, version int) (api.ServiceVersion, bool, error) {
	var v api.ServiceVersion
	var ev []byte
	err := tx.QueryRow(ctx, `SELECT service_key, version, title, workflow_key, evidence_requirements, completion_outcomes, financial_behaviour,
		integration_provider, status, approval_ref FROM catalog.service_version WHERE service_key = $1 AND version = $2`, key, version).
		Scan(&v.ServiceKey, &v.Version, &v.Title, &v.WorkflowKey, &ev, &v.CompletionOutcomes, &v.FinancialBehaviour, &v.IntegrationProvider,
			&v.Status, &v.ApprovalRef)
	if errors.Is(err, pgx.ErrNoRows) {
		return v, false, nil
	}
	if err != nil {
		return v, false, err
	}
	if err := json.Unmarshal(ev, &v.Evidence); err != nil {
		return v, false, err
	}
	if v.Evidence == nil {
		v.Evidence = []api.EvidenceRequirement{}
	}
	return v, true, nil
}

func (s *Store) read(ctx context.Context, fn func(ctx context.Context, tx db.Tx) error) error {
	return s.runner.InTx(ctx, db.TxOptions{ReadOnly: true, Isolation: db.RepeatableRead}, fn)
}

var _ api.Reader = (*Store)(nil)

// Service implements api.Reader.
func (s *Store) Service(ctx context.Context, key string, version int) (api.ServiceVersion, bool, error) {
	var v api.ServiceVersion
	var ok bool
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		v, ok, err = service(ctx, tx, key, version)
		return err
	})
	return v, ok, err
}

// Components implements api.Reader.
func (s *Store) Components(ctx context.Context, productKey string, productVersion int) ([]api.Component, error) {
	out := []api.Component{}
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		rows, err := tx.Query(ctx, `SELECT seq, service_key, service_version FROM catalog.subscription_component
			WHERE product_key = $1 AND product_version = $2 ORDER BY seq`, productKey, productVersion)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var c api.Component
			if err := rows.Scan(&c.Seq, &c.ServiceKey, &c.ServiceVersion); err != nil {
				return err
			}
			out = append(out, c)
		}
		return rows.Err()
	})
	return out, err
}

// current returns the users currently holding kind on service (latest
// event per user is GRANT).
func current(ctx context.Context, tx db.Tx, serviceKey, kind string) ([]string, error) {
	rows, err := tx.Query(ctx, `SELECT DISTINCT ON (user_id) user_id::text, action FROM catalog.service_scope_event
		WHERE service_key = $1 AND scope_kind = $2 ORDER BY user_id, seq DESC`, serviceKey, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var u, a string
		if err := rows.Scan(&u, &a); err != nil {
			return nil, err
		}
		if a == "GRANT" {
			out = append(out, u)
		}
	}
	sort.Strings(out)
	return out, rows.Err()
}

// Supervisors implements api.Reader.
func (s *Store) Supervisors(ctx context.Context, serviceKey string) ([]string, error) {
	var out []string
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		out, err = current(ctx, tx, serviceKey, api.ScopeSupervisor)
		return err
	})
	return out, err
}

// InScope implements api.Reader.
func (s *Store) InScope(ctx context.Context, serviceKey, userID, kind string) (bool, error) {
	var in bool
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		us, err := current(ctx, tx, serviceKey, kind)
		for _, u := range us {
			in = in || u == userID
		}
		return err
	})
	return in, err
}

// ServicesInScope implements api.Reader.
func (s *Store) ServicesInScope(ctx context.Context, userID string) ([]string, error) {
	out := []string{}
	if !uuidPattern.MatchString(userID) {
		return out, nil
	}
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		rows, err := tx.Query(ctx, `SELECT service_key FROM (SELECT DISTINCT ON (service_key, scope_kind) service_key, action
			FROM catalog.service_scope_event WHERE user_id = $1 ORDER BY service_key, scope_kind, seq DESC) x WHERE action = 'GRANT'
			GROUP BY service_key ORDER BY service_key`, userID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var k string
			if err := rows.Scan(&k); err != nil {
				return err
			}
			out = append(out, k)
		}
		return rows.Err()
	})
	return out, err
}
