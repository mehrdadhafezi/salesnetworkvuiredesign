package command

import (
	"context"
	"crypto/sha256"
	"errors"
	"regexp"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/inbox"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

// Gateway request states.
const (
	GatewayUnknown        = "UNKNOWN"
	GatewayRedirected     = "REDIRECT_ISSUED"
	GatewayVerified       = "VERIFIED"
	GatewayFailed         = "FAILED_AT_GATEWAY"
	GatewayCancelled      = "CANCELLED_AT_GATEWAY"
	GatewayReconciliation = "RECONCILIATION_REQUIRED"
)

// CallbackConsumer is the inbox consumer that deduplicates callbacks.
const CallbackConsumer = "payments.gateway_callback"

// GatewayActor is the technical actor of callback / reconciliation steps.
const GatewayActor = "service:payments.gateway"

var externalRefPattern = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,128}$`)

var terminalGateway = map[string]bool{GatewayVerified: true, GatewayFailed: true, GatewayCancelled: true, GatewayReconciliation: true}

// Gateway runs the synthetic-only gateway flow (Architecture V1 §15.7). It
// has no HTTP route and no composition-root wiring in Slice 12: the
// customer channel (§15.10) is not built and the internal pay-online path is
// an assist path (OPD-10). A VERIFIED request carries GATEWAY_AUTO
// provenance and is never a Finance review (D-25); a callback never issues
// a session (D-26) and never changes a billing stage, the ledger or any
// other module.
type Gateway struct {
	svc  *Service
	port api.GatewayPort
}

// NewGateway returns the gateway flow over port.
func (s *Service) NewGateway(port api.GatewayPort) (*Gateway, error) {
	if port == nil || !providerPattern.MatchString(port.Provider()) {
		return nil, errors.New("payments/command: gateway port with a valid provider is required")
	}
	return &Gateway{svc: s, port: port}, nil
}

var providerPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,31}$`)

// GatewayStart starts one gateway request for an invoice's current stage.
type GatewayStart struct {
	InvoiceID      string
	IdempotencyKey string
	CorrelationID  string
	Actor          Actor
}

// GatewayResult reports a gateway request's state after a step.
type GatewayResult struct {
	Status             result.Status      `json:"-"`
	StatusName         string             `json:"status"`
	Reasons            []reasoncodes.Code `json:"reasons"`
	BlockedBy          []string           `json:"blocked_by,omitempty"`
	RequestID          string             `json:"request_id,omitempty"`
	State              string             `json:"state,omitempty"`
	ExternalRef        string             `json:"external_ref,omitempty"`
	ApprovalProvenance string             `json:"approval_provenance,omitempty"`
	// FinanceReviewed and SessionIssued are always false (D-25, D-26).
	FinanceReviewed bool `json:"finance_reviewed"`
	SessionIssued   bool `json:"session_issued"`
	Duplicate       bool `json:"duplicate"`
}

func gwRefusal(st result.Status, code reasoncodes.Code, blocked ...string) GatewayResult {
	return GatewayResult{Status: st, StatusName: st.String(), Reasons: []reasoncodes.Code{code}, BlockedBy: blocked}
}

type gwRequest struct {
	id, state, provenance, unit, invoiceID string
	externalRef                            *string
	amount, version                        int64
	stageNo                                int
}

func (r gwRequest) result() GatewayResult {
	st := result.Success
	switch r.state {
	case GatewayUnknown:
		st = result.Unknown
	case GatewayFailed, GatewayCancelled:
		st = result.Failed
	case GatewayReconciliation:
		st = result.Conflict
	}
	out := GatewayResult{Status: st, StatusName: st.String(), Reasons: []reasoncodes.Code{}, RequestID: r.id, State: r.state, ApprovalProvenance: r.provenance}
	if st == result.Unknown {
		out.Reasons = []reasoncodes.Code{reasoncodes.OutcomeUnknown}
	}
	if st == result.Conflict {
		out.Reasons = []reasoncodes.Code{reasoncodes.LinkageConflict}
	}
	if r.externalRef != nil {
		out.ExternalRef = *r.externalRef
	}
	return out
}

func loadRequest(ctx context.Context, tx db.Tx, id string, lock bool) (gwRequest, bool, error) {
	sql := `SELECT request_id::text, state, approval_provenance, unit, invoice_id::text, external_ref, amount, version, stage_no
		FROM payments.gateway_request WHERE request_id = $1`
	if lock {
		sql += ` FOR UPDATE`
	}
	var r gwRequest
	err := tx.QueryRow(ctx, sql, id).Scan(&r.id, &r.state, &r.provenance, &r.unit, &r.invoiceID, &r.externalRef, &r.amount, &r.version, &r.stageNo)
	if isNoRows(err) {
		return r, false, nil
	}
	return r, err == nil, err
}

func (g *Gateway) event(ctx context.Context, tx db.Tx, requestID *string, kind string, outcome result.Status, ext *string, detail string) error {
	at := g.svc.clock.Now().Time().UTC()
	id, err := ids.NewUUID()
	if err != nil {
		return err
	}
	var d *string
	if detail != "" {
		t := truncate300(detail)
		d = &t
	}
	_, err = tx.Exec(ctx, `INSERT INTO payments.gateway_event (event_id, request_id, provider, kind, outcome, external_ref, detail, actor_principal,
		occurred_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`, id, requestID, g.port.Provider(), kind, outcome.String(), ext, d, GatewayActor, at)
	return err
}

func (g *Gateway) setState(ctx context.Context, tx db.Tx, r gwRequest, state string, ext *string, verifiedRef []byte) (gwRequest, error) {
	at := g.svc.clock.Now().Time().UTC()
	prov := "NOT_APPROVED"
	if state == GatewayVerified {
		prov = "GATEWAY_AUTO"
	}
	if ext == nil {
		ext = r.externalRef
	}
	err := tx.QueryRow(ctx, `UPDATE payments.gateway_request SET state = $3, external_ref = $4, verified_ref_sha256 = $5, approval_provenance = $6,
		updated_at = $7, version = version + 1 WHERE request_id = $1 AND version = $2 RETURNING version`,
		r.id, r.version, state, ext, verifiedRef, prov, at).Scan(&r.version)
	if err != nil {
		return r, err
	}
	r.state, r.externalRef, r.provenance = state, ext, prov
	return r, nil
}

// Start creates the request write-ahead as UNKNOWN, then calls the
// provider outside any transaction and records the answer. A transport
// failure leaves it UNKNOWN: Reconcile must inquire before any retry.
func (g *Gateway) Start(ctx context.Context, mode rollout.Mode, c GatewayStart) (GatewayResult, error) {
	if !uuidPattern.MatchString(c.InvoiceID) || !keyPattern.MatchString(c.IdempotencyKey) || c.CorrelationID == "" || len(c.CorrelationID) > 128 {
		return GatewayResult{}, invalid("gateway start")
	}
	if st, code, blocked, refused := modeRefusal(mode); refused {
		return gwRefusal(st, code, blocked...), nil
	}
	if c.Actor.UserID == "" || c.Actor.Principal == "" {
		return gwRefusal(result.Unauthorized, reasoncodes.RowScopeDenied), nil
	}
	requestID := ids.Stable("payments.gateway_request", g.port.Provider(), c.Actor.UserID, c.IdempotencyKey)
	var (
		req      gwRequest
		existing bool
		refusal  *GatewayResult
	)
	err := g.svc.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		r, ok, err := loadRequest(ctx, tx, requestID, true)
		if err != nil {
			return err
		}
		if ok {
			req, existing = r, true
			return nil
		}
		e, found, err := g.svc.evaluate(ctx, tx, c.InvoiceID, c.Actor)
		if err != nil {
			return err
		}
		if !found {
			return ErrNotFound
		}
		if e.Status != result.Success {
			x := gwRefusal(e.Status, e.Reasons[0], e.BlockedBy...)
			refusal = &x
			return nil
		}
		// Serialise with manual submissions on the same collection account,
		// then re-evaluate under the lock.
		if _, err := lockAccount(ctx, tx, c.InvoiceID, e.StageNo, g.svc.clock.Now().Time().UTC()); err != nil {
			return err
		}
		if e, _, err = g.svc.evaluate(ctx, tx, c.InvoiceID, c.Actor); err != nil {
			return err
		}
		if e.Status != result.Success {
			x := gwRefusal(e.Status, e.Reasons[0], e.BlockedBy...)
			refusal = &x
			return nil
		}
		// 2.0.160 ajax_pay_online: no gateway payment while a registered
		// transfer is pending; the amount is the whole stage due.
		var open int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM payments.gateway_request WHERE invoice_id = $1 AND stage_no = $2
			AND state IN ('UNKNOWN', 'REDIRECT_ISSUED', 'RECONCILIATION_REQUIRED')`, c.InvoiceID, e.StageNo).Scan(&open); err != nil {
			return err
		}
		if e.Used > 0 || open > 0 {
			x := gwRefusal(result.Blocked, reasoncodes.ActionDenied)
			refusal = &x
			return nil
		}
		at := g.svc.clock.Now().Time().UTC()
		if _, err := tx.Exec(ctx, `INSERT INTO payments.gateway_request (request_id, idempotency_key, invoice_id, stage_no, amount, unit, provider,
			state, approval_provenance, requested_by_user_id, correlation_id, created_at, updated_at, version)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 'UNKNOWN', 'NOT_APPROVED', $8, $9, $10, $10, 1)`,
			requestID, c.IdempotencyKey, c.InvoiceID, e.StageNo, e.Due, e.Unit, g.port.Provider(), c.Actor.UserID, c.CorrelationID, at); err != nil {
			return err
		}
		req = gwRequest{id: requestID, state: GatewayUnknown, provenance: "NOT_APPROVED", unit: e.Unit, invoiceID: c.InvoiceID, amount: e.Due,
			version: 1, stageNo: e.StageNo}
		return g.event(ctx, tx, &requestID, "REQUEST", result.Unknown, nil, "write-ahead before provider call")
	})
	if err != nil {
		return GatewayResult{}, err
	}
	if refusal != nil {
		return *refusal, nil
	}
	if existing {
		if req.invoiceID != c.InvoiceID {
			// The key names another invoice's request: never a replay.
			return gwRefusal(result.Conflict, reasoncodes.ContractInvalid), nil
		}
		if req.state == GatewayUnknown {
			return g.Reconcile(ctx, requestID)
		}
		out := req.result()
		out.Duplicate = true
		return out, nil
	}
	return g.request(ctx, req)
}

// request calls the provider for a request that is UNKNOWN and proven not
// to exist at the provider (fresh, or reconciled as absent).
func (g *Gateway) request(ctx context.Context, req gwRequest) (GatewayResult, error) {
	ext, callErr := g.port.Request(ctx, api.GatewayRequest{RequestID: req.id, Amount: req.amount, Unit: req.unit})
	err := g.svc.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		r, ok, err := loadRequest(ctx, tx, req.id, true)
		if err != nil || !ok {
			return err
		}
		if r.state != GatewayUnknown {
			req = r // a callback or reconciliation got there first
			return nil
		}
		switch {
		case callErr == nil && externalRefPattern.MatchString(ext):
			if req, err = g.setState(ctx, tx, r, GatewayRedirected, &ext, nil); err != nil {
				return err
			}
			return g.event(ctx, tx, &r.id, "REQUEST", result.Success, &ext, "")
		case errors.Is(callErr, api.ErrGatewayRefused):
			if req, err = g.setState(ctx, tx, r, GatewayFailed, nil, nil); err != nil {
				return err
			}
			return g.event(ctx, tx, &r.id, "REQUEST", result.Failed, nil, "provider refused")
		default:
			req = r
			detail := "invalid provider reference"
			if callErr != nil {
				detail = callErr.Error()
			}
			return g.event(ctx, tx, &r.id, "REQUEST", result.Unknown, nil, detail)
		}
	})
	if err != nil {
		return GatewayResult{}, err
	}
	return req.result(), nil
}

// Reconcile inquires the provider before any retry (reconcile-before-retry):
// a request proven absent may be requested again under the same id; an
// unreadable provider leaves it UNKNOWN; a paid request is verified.
func (g *Gateway) Reconcile(ctx context.Context, requestID string) (GatewayResult, error) {
	var req gwRequest
	var ok bool
	if err := g.svc.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		req, ok, err = loadRequest(ctx, tx, requestID, false)
		return err
	}); err != nil {
		return GatewayResult{}, err
	}
	if !ok {
		return GatewayResult{}, ErrNotFound
	}
	if terminalGateway[req.state] {
		return req.result(), nil
	}
	inq, inqErr := g.port.Inquire(ctx, requestID)
	verify, retry := false, false
	err := g.svc.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		r, ok, err := loadRequest(ctx, tx, requestID, true)
		if err != nil || !ok {
			return err
		}
		req = r
		if terminalGateway[r.state] {
			return nil
		}
		if inqErr != nil {
			return g.event(ctx, tx, &r.id, "INQUIRY", result.Unknown, nil, inqErr.Error())
		}
		if !inq.Found {
			if err := g.event(ctx, tx, &r.id, "INQUIRY", result.Success, nil, "proven absent at provider"); err != nil {
				return err
			}
			retry = r.state == GatewayUnknown
			return nil
		}
		ext := inq.ExternalRef
		if !externalRefPattern.MatchString(ext) || (r.externalRef != nil && *r.externalRef != ext) {
			if req, err = g.setState(ctx, tx, r, GatewayReconciliation, nil, nil); err != nil {
				return err
			}
			return g.event(ctx, tx, &r.id, "INQUIRY", result.Conflict, nil, "provider reference conflicts with the recorded one")
		}
		if err := g.event(ctx, tx, &r.id, "INQUIRY", result.Success, &ext, "status "+inq.Status); err != nil {
			return err
		}
		switch inq.Status {
		case "PAID":
			if r.externalRef == nil {
				if req, err = g.setState(ctx, tx, r, GatewayRedirected, &ext, nil); err != nil {
					return err
				}
			}
			verify = true
		case "PENDING":
			if r.state == GatewayUnknown {
				req, err = g.setState(ctx, tx, r, GatewayRedirected, &ext, nil)
			}
		case "CANCELLED":
			req, err = g.setState(ctx, tx, r, GatewayCancelled, &ext, nil)
		case "FAILED":
			req, err = g.setState(ctx, tx, r, GatewayFailed, &ext, nil)
		default:
			req, err = g.setState(ctx, tx, r, GatewayReconciliation, &ext, nil)
		}
		return err
	})
	if err != nil {
		return GatewayResult{}, err
	}
	if retry {
		return g.request(ctx, req)
	}
	if verify {
		return g.verify(ctx, requestID)
	}
	return req.result(), nil
}

// verify asks the provider to verify server-side (a callback is never proof).
func (g *Gateway) verify(ctx context.Context, requestID string) (GatewayResult, error) {
	var req gwRequest
	if err := g.svc.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		req, _, err = loadRequest(ctx, tx, requestID, false)
		return err
	}); err != nil {
		return GatewayResult{}, err
	}
	if req.externalRef == nil || terminalGateway[req.state] {
		return req.result(), nil
	}
	v, verr := g.port.Verify(ctx, *req.externalRef, req.amount)
	err := g.svc.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		r, ok, err := loadRequest(ctx, tx, requestID, true)
		if err != nil || !ok {
			return err
		}
		req = r
		if terminalGateway[r.state] {
			return nil
		}
		switch {
		case verr != nil:
			if r.state != GatewayUnknown {
				if req, err = g.setState(ctx, tx, r, GatewayUnknown, nil, nil); err != nil {
					return err
				}
			}
			return g.event(ctx, tx, &r.id, "VERIFY", result.Unknown, r.externalRef, verr.Error())
		case v.Verified && v.Amount == r.amount:
			sum := sha256.Sum256([]byte(v.RefID))
			if req, err = g.setState(ctx, tx, r, GatewayVerified, nil, sum[:]); err != nil {
				return err
			}
			return g.event(ctx, tx, &r.id, "VERIFY", result.Success, r.externalRef, "verified; provenance GATEWAY_AUTO (not a Finance review)")
		case v.Verified:
			if req, err = g.setState(ctx, tx, r, GatewayReconciliation, nil, nil); err != nil {
				return err
			}
			return g.event(ctx, tx, &r.id, "VERIFY", result.Conflict, r.externalRef, "verified amount differs from the request")
		default:
			if req, err = g.setState(ctx, tx, r, GatewayFailed, nil, nil); err != nil {
				return err
			}
			return g.event(ctx, tx, &r.id, "VERIFY", result.Failed, r.externalRef, "provider did not verify")
		}
	})
	if err != nil {
		return GatewayResult{}, err
	}
	return req.result(), nil
}

// HandleCallback records an authenticated callback (inbox-deduplicated on
// provider, external reference and status). A duplicate is a no-op; a
// callback for an unknown request is recorded as an orphan; a callback
// after a terminal state is recorded as stale and changes nothing; an OK
// callback triggers server-side verification. It never creates a session.
func (g *Gateway) HandleCallback(ctx context.Context, cb api.GatewayCallback) (GatewayResult, error) {
	if cb.Provider != g.port.Provider() || !uuidPattern.MatchString(cb.RequestID) || !externalRefPattern.MatchString(cb.ExternalRef) ||
		(cb.Status != "OK" && cb.Status != "NOK") {
		return GatewayResult{}, invalid("gateway callback")
	}
	msgID := ids.Stable(CallbackConsumer, cb.Provider, cb.ExternalRef, cb.Status)
	var out GatewayResult
	verify := false
	err := g.svc.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		ext := cb.ExternalRef
		res, err := inbox.Consume(ctx, tx, CallbackConsumer, msgID, g.svc.clock.Now(), func(ctx context.Context) error {
			r, ok, err := loadRequest(ctx, tx, cb.RequestID, true)
			if err != nil {
				return err
			}
			if !ok {
				out = gwRefusal(result.Blocked, reasoncodes.DependencyUnknown)
				return g.event(ctx, tx, nil, "ORPHAN_CALLBACK", result.Blocked, &ext, "callback for an unknown request")
			}
			if terminalGateway[r.state] {
				out = r.result()
				return g.event(ctx, tx, &r.id, "STALE_CALLBACK", result.Success, &ext, "callback after a terminal state; nothing changed")
			}
			if r.externalRef != nil && *r.externalRef != ext {
				if r, err = g.setState(ctx, tx, r, GatewayReconciliation, nil, nil); err != nil {
					return err
				}
				out = r.result()
				return g.event(ctx, tx, &r.id, "CALLBACK", result.Conflict, &ext, "callback reference conflicts with the recorded one")
			}
			if cb.Status == "NOK" {
				if r, err = g.setState(ctx, tx, r, GatewayCancelled, &ext, nil); err != nil {
					return err
				}
				out = r.result()
				return g.event(ctx, tx, &r.id, "CALLBACK", result.Failed, &ext, "cancelled at gateway")
			}
			if r.externalRef == nil {
				// Out of order: the callback arrived before the request answer
				// was recorded.
				if r, err = g.setState(ctx, tx, r, GatewayRedirected, &ext, nil); err != nil {
					return err
				}
			}
			verify = true
			out = r.result()
			return g.event(ctx, tx, &r.id, "CALLBACK", result.Success, &ext, "callback received; verification required")
		})
		if err != nil {
			return err
		}
		if res == inbox.Duplicate {
			r, ok, err := loadRequest(ctx, tx, cb.RequestID, false)
			if err != nil {
				return err
			}
			if ok {
				out = r.result()
			} else {
				out = gwRefusal(result.Blocked, reasoncodes.DependencyUnknown)
			}
			out.Duplicate = true
		}
		return nil
	})
	if err != nil {
		return GatewayResult{}, err
	}
	if verify {
		return g.verify(ctx, cb.RequestID)
	}
	return out, nil
}
