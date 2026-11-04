// Package activation is biawin's Slice 14 activation contract:
// InvoiceFullyPaid → Biawin-owned activation decision → SubscriptionActivated
// → one SubscriptionItemActivated per service component. It runs in one
// biawin-owned transaction per message (inbox-deduplicated, business-key
// unique); fulfillment creates its cases later from the outbox, in its own
// transaction. No transaction spans two modules.
//
// D-35 is unresolved: rollout CONTROLLED (a real activation) is always
// refused; DISABLED records nothing; SHADOW records the decision and SHADOW
// subscriptions / items only. A Complaint or Archive branch never enters
// Biawin, an unknown branch or customer lineage is BLOCKED with its
// reconciliation context, and any component with a financial behaviour
// other than NONE (credit grants, ledger charges) is BLOCKED.
package activation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/api"
	catalogapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/catalog/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/inbox"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/outbox"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

// Consumer is the inbox consumer name.
const Consumer = "biawin.activation"

// ErrInvalid reports a malformed message.
var ErrInvalid = errors.New("biawin/activation: invalid message")

func invalid(f string) error { return fmt.Errorf("%w: %s", ErrInvalid, f) }

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Outcome is one line's decision.
type Outcome struct {
	BusinessKey    string             `json:"business_key"`
	ActivationID   string             `json:"activation_id"`
	Decision       string             `json:"decision"`
	Status         result.Status      `json:"-"`
	Reasons        []reasoncodes.Code `json:"reasons"`
	BlockedBy      []string           `json:"blocked_by"`
	SubscriptionID *string            `json:"subscription_id,omitempty"`
	ItemCount      int                `json:"item_count"`
	Existing       bool               `json:"existing"`
}

// Report is the outcome of one message.
type Report struct {
	Mode      rollout.Mode
	Refused   *Outcome // a refusal before anything is recorded
	Duplicate bool     // the message was already processed (inbox)
	Lines     []Outcome
}

// Service runs the activation decision.
type Service struct {
	runner  *db.Runner
	catalog catalogapi.Reader
	clock   timeutil.Clock
}

// New returns the service over a biawin-owned runner.
func New(runner *db.Runner, catalog catalogapi.Reader, clock timeutil.Clock) (*Service, error) {
	if runner == nil || runner.Module() != biawin.Module || catalog == nil || clock == nil {
		return nil, errors.New("biawin/activation: biawin runner, catalog reader and clock are required")
	}
	return &Service{runner: runner, catalog: catalog, clock: clock}, nil
}

func validate(m api.InvoiceFullyPaid) error {
	if !uuidPattern.MatchString(m.MessageID) || !uuidPattern.MatchString(m.InvoiceID) {
		return invalid("message_id / invoice_id")
	}
	if m.CorrelationID == "" || len(m.CorrelationID) > 128 {
		return invalid("correlation_id")
	}
	if len(m.Lines) == 0 || len(m.Lines) > 50 {
		return invalid("lines")
	}
	seen := map[string]bool{}
	for _, l := range m.Lines {
		k := fmt.Sprintf("%s#%d", l.ProductKey, l.ProductVersion)
		if l.ProductKey == "" || l.ProductVersion < 1 || seen[k] {
			return invalid("product line")
		}
		seen[k] = true
	}
	if m.Customer.ID != nil && !uuidPattern.MatchString(*m.Customer.ID) {
		return invalid("customer_id")
	}
	return nil
}

// BusinessKey is the stable activation key of one paid product line.
func BusinessKey(invoiceID, productKey string, productVersion int) string {
	return fmt.Sprintf("invoice:%s|product:%s#%d", invoiceID, productKey, productVersion)
}

// plan is one line's pre-transaction decision (catalog reads are immutable
// versions, so they do not need the biawin transaction).
type plan struct {
	line       api.ProductLine
	key        string
	decision   string
	blocks     []string
	components []catalogapi.Component
}

func (s *Service) plan(ctx context.Context, m api.InvoiceFullyPaid, mode rollout.Mode) ([]plan, error) {
	var out []plan
	for _, l := range m.Lines {
		p := plan{line: l, key: BusinessKey(m.InvoiceID, l.ProductKey, l.ProductVersion), decision: api.DecisionShadowActivated}
		switch m.Route.Branch {
		case api.RouteComplaint, api.RouteArchive:
			p.decision = api.DecisionNotApplicable
			out = append(out, p)
			continue
		case api.RouteAdditionalServices:
		default:
			p.blocks = append(p.blocks, api.BlockRouteUnknown)
		}
		switch m.Customer.State {
		case "RESOLVED":
			if m.Customer.ID == nil {
				p.blocks = append(p.blocks, api.BlockIdentityUnknown)
			}
		case "CONFLICT":
			p.blocks = append(p.blocks, api.BlockIdentityConflict)
		default:
			p.blocks = append(p.blocks, api.BlockIdentityUnknown)
		}
		comps, err := s.catalog.Components(ctx, l.ProductKey, l.ProductVersion)
		if err != nil {
			return nil, err
		}
		if len(comps) == 0 {
			p.blocks = append(p.blocks, api.BlockComponentsMissing)
		}
		for _, c := range comps {
			sv, ok, err := s.catalog.Service(ctx, c.ServiceKey, c.ServiceVersion)
			if err != nil {
				return nil, err
			}
			switch {
			case !ok:
				p.blocks = append(p.blocks, api.BlockServiceMissing)
			case sv.Status != "ACTIVE":
				p.blocks = append(p.blocks, api.BlockServiceRetired)
			case sv.FinancialBehaviour != catalogapi.FinancialNone:
				p.blocks = append(p.blocks, api.BlockFinancialBehaviour, api.BlockD35)
			}
		}
		p.components = comps
		if mode == rollout.Controlled {
			// A real activation is not authorised while D-35 is open.
			p.blocks = append(p.blocks, api.BlockD35)
		}
		if len(p.blocks) > 0 {
			p.decision = api.DecisionBlocked
			p.blocks = dedupe(p.blocks)
		}
		out = append(out, p)
	}
	return out, nil
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

func codeOf(block string) reasoncodes.Code {
	switch block {
	case api.BlockRouteUnknown, api.BlockIdentityUnknown:
		return reasoncodes.DependencyUnknown
	case api.BlockIdentityConflict:
		return reasoncodes.LinkageConflict
	default:
		return reasoncodes.PolicyUnresolved
	}
}

// HandleInvoiceFullyPaid decides activation for every paid product line of
// one message, in one biawin transaction (inbox-deduplicated).
func (s *Service) HandleInvoiceFullyPaid(ctx context.Context, mode rollout.Mode, m api.InvoiceFullyPaid) (Report, error) {
	rep := Report{Mode: mode}
	if err := validate(m); err != nil {
		return rep, err
	}
	if mode == rollout.Disabled || !mode.Valid() {
		rep.Refused = &Outcome{Decision: api.DecisionBlocked, Status: result.Blocked, Reasons: []reasoncodes.Code{reasoncodes.FeatureDisabled}, BlockedBy: []string{}}
		return rep, nil
	}
	plans, err := s.plan(ctx, m, mode)
	if err != nil {
		return rep, err
	}
	err = s.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		rep.Lines = nil
		now := s.clock.Now()
		out, err := inbox.Consume(ctx, tx, Consumer, m.MessageID, now, func(ctx context.Context) error {
			for _, p := range plans {
				o, err := s.record(ctx, tx, m, p, now)
				if err != nil {
					return err
				}
				rep.Lines = append(rep.Lines, o)
			}
			return nil
		})
		if err != nil {
			return err
		}
		if out == inbox.Duplicate {
			rep.Duplicate = true
		}
		return nil
	})
	return rep, err
}

func (s *Service) record(ctx context.Context, tx db.Tx, m api.InvoiceFullyPaid, p plan, now timeutil.Instant) (Outcome, error) {
	at := now.Time().UTC()
	o := Outcome{BusinessKey: p.key, Decision: p.decision, Reasons: []reasoncodes.Code{}, BlockedBy: append([]string{}, p.blocks...)}
	// A business key already decided (by another message) is never decided
	// twice: the existing decision stands and no second subscription exists.
	var existingID, existingDecision string
	var existingSub *string
	var existingCount int
	err := tx.QueryRow(ctx, `SELECT activation_id::text, decision, subscription_id::text, item_count FROM biawin.activation WHERE business_key = $1`, p.key).
		Scan(&existingID, &existingDecision, &existingSub, &existingCount)
	if err == nil {
		o.ActivationID, o.Decision, o.SubscriptionID, o.ItemCount, o.Existing = existingID, existingDecision, existingSub, existingCount, true
		o.Status = result.Success
		detail, _ := json.Marshal(map[string]any{"message_id": m.MessageID, "correlation_id": m.CorrelationID})
		if _, err := tx.Exec(ctx, `INSERT INTO biawin.activation_event (event_id, activation_id, kind, detail, occurred_at) VALUES ($1, $2, 'DUPLICATE_MESSAGE', $3, $4)`,
			ids.Stable("biawin.activation_event", existingID, m.MessageID), existingID, detail, at); err != nil {
			return o, err
		}
		return o, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return o, err
	}
	o.ActivationID = ids.Stable("biawin.activation", p.key)
	for _, b := range p.blocks {
		o.Reasons = append(o.Reasons, codeOf(b))
	}
	o.Reasons, _ = reasoncodes.Normalize(o.Reasons)
	if o.Reasons == nil {
		o.Reasons = []reasoncodes.Code{}
	}
	switch p.decision {
	case api.DecisionShadowActivated:
		o.Status = result.Success
	case api.DecisionNotApplicable:
		o.Status = result.Success
	default:
		o.Status = result.Blocked
	}
	var subID any
	if p.decision == api.DecisionShadowActivated {
		id := ids.Stable("biawin.target_subscription", p.key)
		subID, o.SubscriptionID, o.ItemCount = id, &id, len(p.components)
	}
	reasons, _ := json.Marshal(o.Reasons)
	blocked, _ := json.Marshal(o.BlockedBy)
	if _, err := tx.Exec(ctx, `INSERT INTO biawin.activation (activation_id, business_key, message_id, correlation_id, invoice_id, invoice_ref,
		customer_link_state, customer_id, existing_customer, customer_basis, route, route_basis, product_key, product_version, rollout_mode, decision,
		reasons, blocked_by, subscription_id, item_count, decided_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)`,
		o.ActivationID, p.key, m.MessageID, m.CorrelationID, m.InvoiceID, m.InvoiceRef, linkState(m.Customer), resolvedID(m.Customer),
		m.Customer.Existing, basis(m.Customer.Basis), route(m.Route.Branch), basis(m.Route.Basis), p.line.ProductKey, p.line.ProductVersion,
		string(rollout.Shadow), p.decision, reasons, blocked, subID, o.ItemCount, at); err != nil {
		return o, err
	}
	detail, _ := json.Marshal(map[string]any{"message_id": m.MessageID, "decision": p.decision, "blocked_by": o.BlockedBy})
	if _, err := tx.Exec(ctx, `INSERT INTO biawin.activation_event (event_id, activation_id, kind, detail, occurred_at) VALUES ($1, $2, 'DECIDED', $3, $4)`,
		ids.Stable("biawin.activation_event", o.ActivationID, "decided"), o.ActivationID, detail, at); err != nil {
		return o, err
	}
	if p.decision != api.DecisionShadowActivated {
		return o, nil
	}
	customerID := *m.Customer.ID
	if _, err := tx.Exec(ctx, `INSERT INTO biawin.target_subscription (subscription_id, activation_id, invoice_id, customer_id, product_key,
		product_version, state, origin_mode, created_at) VALUES ($1,$2,$3,$4,$5,$6,'SHADOW_ACTIVE','SHADOW_COMPARE',$7)`,
		*o.SubscriptionID, o.ActivationID, m.InvoiceID, customerID, p.line.ProductKey, p.line.ProductVersion, at); err != nil {
		return o, err
	}
	corr := m.CorrelationID
	sa, _ := json.Marshal(api.SubscriptionActivated{ActivationID: o.ActivationID, BusinessKey: p.key, SubscriptionID: *o.SubscriptionID,
		CustomerID: customerID, InvoiceID: m.InvoiceID, ItemCount: o.ItemCount, RolloutMode: string(rollout.Shadow), CorrelationID: corr,
		CausationID: m.MessageID, OccurredAt: at})
	if err := outbox.Append(ctx, tx, outbox.Event{ID: ids.Stable("biawin.outbox.subscription", o.ActivationID), Type: api.EventSubscriptionActivated,
		Version: 1, Payload: sa, OccurredAt: now, CorrelationID: &corr}, now); err != nil {
		return o, err
	}
	for _, c := range p.components {
		itemID := ids.Stable("biawin.target_item", *o.SubscriptionID, fmt.Sprint(c.Seq))
		if _, err := tx.Exec(ctx, `INSERT INTO biawin.target_item (item_id, subscription_id, seq, service_key, service_version, state, created_at)
			VALUES ($1,$2,$3,$4,$5,'SHADOW_ACTIVE',$6)`, itemID, *o.SubscriptionID, c.Seq, c.ServiceKey, c.ServiceVersion, at); err != nil {
			return o, err
		}
		ia, _ := json.Marshal(api.SubscriptionItemActivated{ActivationID: o.ActivationID, SubscriptionID: *o.SubscriptionID, ItemID: itemID,
			Seq: c.Seq, ServiceKey: c.ServiceKey, ServiceVersion: c.ServiceVersion, CustomerID: customerID, RolloutMode: string(rollout.Shadow),
			CorrelationID: corr, CausationID: o.ActivationID, OccurredAt: at})
		if err := outbox.Append(ctx, tx, outbox.Event{ID: ids.Stable("biawin.outbox.item", itemID), Type: api.EventItemActivated, Version: 1,
			Payload: ia, OccurredAt: now, CorrelationID: &corr}, now); err != nil {
			return o, err
		}
	}
	return o, nil
}

func linkState(c api.Customer) string {
	switch c.State {
	case "RESOLVED":
		if c.ID != nil {
			return "RESOLVED"
		}
		return "UNKNOWN"
	case "CONFLICT":
		return "CONFLICT"
	}
	return "UNKNOWN"
}

func resolvedID(c api.Customer) any {
	if linkState(c) == "RESOLVED" {
		return *c.ID
	}
	return nil
}

func route(b string) string {
	switch b {
	case api.RouteAdditionalServices, api.RouteComplaint, api.RouteArchive:
		return b
	}
	return api.RouteUnknown
}

func basis(b string) string {
	if b == "" {
		return "not supplied"
	}
	if len(b) > 300 {
		return b[:300]
	}
	return b
}
