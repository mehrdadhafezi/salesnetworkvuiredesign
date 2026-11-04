// Package providerops is integration's external-operation ledger (Slice 14):
// one durable record per external provider operation, with write-ahead
// state, reconcile-before-retry and append-only evidence. It knows NO
// provider: the provider is a Port (integration/biawin implements the
// Biawin one). Nothing here touches billing, payments, finance, wallets or
// commission, and a wallet charge is representable only as BLOCKED (D-38).
//
// State machine:
//
//	PREPARED ──dispatch──▶ DISPATCHING (committed BEFORE the call)
//	DISPATCHING ──ack──▶ ACKNOWLEDGED ──callback / lookup──▶ CONFIRMED
//	DISPATCHING ──definitive rejection──▶ REJECTED
//	DISPATCHING ──timeout / ambiguous / crash──▶ UNKNOWN
//	UNKNOWN ──lookup found──▶ CONFIRMED
//	UNKNOWN ──lookup proves absence──▶ CONFIRMED_ABSENT ──dispatch──▶ DISPATCHING
//	UNKNOWN ──lookup unavailable──▶ RECONCILIATION_REQUIRED
//	RECONCILIATION_REQUIRED ──recorded human evidence──▶ CONFIRMED / CONFIRMED_ABSENT
//
// UNKNOWN and RECONCILIATION_REQUIRED are never dispatched again.
package providerops

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
)

// Operation types.
const (
	OpServiceOrder = "SERVICE_ORDER"
	OpWalletCharge = "WALLET_CHARGE"
)

// States.
const (
	StatePrepared       = "PREPARED"
	StateDispatching    = "DISPATCHING"
	StateAcknowledged   = "ACKNOWLEDGED"
	StateConfirmed      = "CONFIRMED"
	StateRejected       = "REJECTED"
	StateUnknown        = "UNKNOWN"
	StateReconciliation = "RECONCILIATION_REQUIRED"
	StateAbsent         = "CONFIRMED_ABSENT"
	StateBlocked        = "BLOCKED"
)

// Dispatch outcomes a port reports.
type Outcome string

const (
	OutcomeAcknowledged Outcome = "ACKNOWLEDGED"
	OutcomeRejected     Outcome = "REJECTED"
	OutcomeUnknown      Outcome = "UNKNOWN"
)

var (
	// ErrInvalid reports a malformed request.
	ErrInvalid = errors.New("providerops: invalid request")
	// ErrNotFound reports an unknown operation.
	ErrNotFound = errors.New("providerops: operation not found")
	// ErrState reports an action not allowed in the operation's state.
	ErrState = errors.New("providerops: not allowed in the current state")
	// ErrConflict reports the same key with a different request.
	ErrConflict = errors.New("providerops: operation key reused with a different request")
	// ErrPortDisabled reports a port that sends nothing (no connector).
	ErrPortDisabled = errors.New("providerops: provider port is disabled; nothing was sent")
	// ErrSignature reports a callback whose signature does not verify.
	ErrSignature = errors.New("providerops: callback signature invalid")
	// ErrCallbackMalformed reports a verified body that is not a canonical
	// callback envelope (unknown / missing / duplicate fields, trailing data,
	// non-canonical encoding or an unsupported version).
	ErrCallbackMalformed = errors.New("providerops: callback body is not a canonical envelope")
	// ErrCallbackMismatch reports a transport-supplied field that disagrees
	// with the authenticated envelope. Nothing is recorded or changed.
	ErrCallbackMismatch = errors.New("providerops: callback field does not match the authenticated envelope")
)

func invalid(f string) error { return fmt.Errorf("%w: %s", ErrInvalid, f) }

// Request is what a port sends.
type Request struct {
	OperationID   string
	OpType        string
	Reference     string // our deterministic reference (never assumed idempotent at the provider)
	CorrelationID string
	Payload       map[string]string // neutral, non-PII parameters
}

// Response is a port's dispatch answer.
type Response struct {
	Outcome     Outcome
	ExternalRef string
	Evidence    map[string]string
}

// Lookup is a port's reconciliation answer.
type Lookup struct {
	Supported   bool
	Found       bool
	ExternalRef string
	Evidence    map[string]string
}

// Port is one external provider. Synthetic() is true only for test
// adapters; Enabled() is false for an adapter that sends nothing.
type Port interface {
	Provider() string
	Enabled() bool
	Synthetic() bool
	Submit(ctx context.Context, r Request) (Response, error)
	Lookup(ctx context.Context, reference string) (Lookup, error)
}

// Service is the ledger over an integration-owned runner.
type Service struct {
	runner *db.Runner
	clock  timeutil.Clock
}

// New returns the service.
func New(runner *db.Runner, clock timeutil.Clock) (*Service, error) {
	if runner == nil || runner.Module() != integration.Module || clock == nil {
		return nil, errors.New("providerops: integration runner and clock are required")
	}
	return &Service{runner: runner, clock: clock}, nil
}

var (
	keyPattern  = regexp.MustCompile(`^[A-Za-z0-9:|#._-]{8,191}$`)
	uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

// OperationRequest asks for one external operation.
type OperationRequest struct {
	OperationKey  string
	Provider      string
	OpType        string
	SubjectCaseID *string
	CorrelationID string
	CausationID   string
	Payload       map[string]string
}

// Operation is one stored operation.
type Operation struct {
	OperationID   string    `json:"operation_id"`
	OperationKey  string    `json:"operation_key"`
	Provider      string    `json:"provider"`
	OpType        string    `json:"op_type"`
	SubjectCaseID *string   `json:"subject_case_id"`
	Reference     string    `json:"reference"`
	State         string    `json:"state"`
	BlockedBy     *string   `json:"blocked_by"`
	Attempts      int       `json:"attempts"`
	ExternalRef   *string   `json:"external_ref"`
	Owner         string    `json:"reconciliation_owner"`
	Version       int64     `json:"version"`
	CorrelationID string    `json:"correlation_id"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func fingerprint(r OperationRequest) []byte {
	b, _ := json.Marshal(struct {
		K, P, T string
		S       *string
		Payload map[string]string
	}{r.OperationKey, r.Provider, r.OpType, r.SubjectCaseID, r.Payload})
	h := sha256.Sum256(b)
	return h[:]
}

// Reference is the deterministic provider reference of an operation key.
func Reference(operationKey string) string {
	h := sha256.Sum256([]byte("providerops.reference\x00" + operationKey))
	return "crm-op-" + hex.EncodeToString(h[:12])
}

func (s *Service) event(ctx context.Context, tx db.Tx, opID string, from *string, to, kind string, evidence map[string]string, actor string, at time.Time) error {
	if evidence == nil {
		evidence = map[string]string{}
	}
	ev, _ := json.Marshal(evidence)
	id, err := ids.NewUUID()
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO integration.provider_operation_event (event_id, operation_id, from_state, to_state, kind, evidence, actor, occurred_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, id, opID, from, to, kind, ev, actor, at)
	return err
}

// RequestOperation records one operation (PREPARED, or BLOCKED for a wallet
// charge). The same key with the same request is a replay (the existing
// operation is returned); with a different request it is ErrConflict.
func (s *Service) RequestOperation(ctx context.Context, r OperationRequest) (Operation, bool, error) {
	if !keyPattern.MatchString(r.OperationKey) || r.Provider != "BIAWIN" || r.CorrelationID == "" || len(r.CorrelationID) > 128 || r.CausationID == "" {
		return Operation{}, false, invalid("operation key / provider / correlation / causation")
	}
	if r.OpType != OpServiceOrder && r.OpType != OpWalletCharge {
		return Operation{}, false, invalid("op type")
	}
	if r.SubjectCaseID != nil && !uuidPattern.MatchString(*r.SubjectCaseID) {
		return Operation{}, false, invalid("subject case")
	}
	fp := fingerprint(r)
	var op Operation
	replay := false
	err := s.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		at := s.clock.Now().Time().UTC()
		existing, ok, err := load(ctx, tx, `operation_key = $1`, r.OperationKey, false)
		if err != nil {
			return err
		}
		if ok {
			var stored []byte
			if err := tx.QueryRow(ctx, `SELECT request_fingerprint FROM integration.provider_operation WHERE operation_id = $1`, existing.OperationID).Scan(&stored); err != nil {
				return err
			}
			if !hmac.Equal(stored, fp) {
				return ErrConflict
			}
			op, replay = existing, true
			return nil
		}
		state, blocked := StatePrepared, (*string)(nil)
		if r.OpType == OpWalletCharge {
			d := "D-38"
			state, blocked = StateBlocked, &d
		}
		id := ids.Stable("integration.provider_operation", r.OperationKey)
		if _, err := tx.Exec(ctx, `INSERT INTO integration.provider_operation (operation_id, operation_key, provider, op_type, subject_case_id, reference,
			request_fingerprint, correlation_id, causation_id, state, blocked_by, attempts, external_ref, reconciliation_owner, version, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,0,NULL,'integration.biawin',1,$12,$12)`, id, r.OperationKey, r.Provider, r.OpType, r.SubjectCaseID,
			Reference(r.OperationKey), fp, r.CorrelationID, r.CausationID, state, blocked, at); err != nil {
			return err
		}
		kind := "REQUESTED"
		if state == StateBlocked {
			kind = "BLOCKED"
		}
		if err := s.event(ctx, tx, id, nil, state, kind, map[string]string{"op_type": r.OpType}, "service:integration", at); err != nil {
			return err
		}
		op, _, err = load(ctx, tx, `operation_id = $1`, id, false)
		return err
	})
	return op, replay, err
}

const cols = `operation_id::text, operation_key, provider, op_type, subject_case_id::text, reference, state, blocked_by, attempts, external_ref,
	reconciliation_owner, version, correlation_id, updated_at`

func scan(row pgx.Row) (Operation, error) {
	var o Operation
	err := row.Scan(&o.OperationID, &o.OperationKey, &o.Provider, &o.OpType, &o.SubjectCaseID, &o.Reference, &o.State, &o.BlockedBy, &o.Attempts,
		&o.ExternalRef, &o.Owner, &o.Version, &o.CorrelationID, &o.UpdatedAt)
	return o, err
}

func load(ctx context.Context, tx db.Tx, where string, arg any, lock bool) (Operation, bool, error) {
	q := `SELECT ` + cols + ` FROM integration.provider_operation WHERE ` + where
	if lock {
		q += ` FOR UPDATE`
	}
	o, err := scan(tx.QueryRow(ctx, q, arg))
	if errors.Is(err, pgx.ErrNoRows) {
		return o, false, nil
	}
	return o, err == nil, err
}

func (s *Service) transition(ctx context.Context, tx db.Tx, o Operation, to, kind string, ext *string, evidence map[string]string, actor string, at time.Time, attempt bool) error {
	inc := 0
	if attempt {
		inc = 1
	}
	tag, err := tx.Exec(ctx, `UPDATE integration.provider_operation SET state = $3, external_ref = COALESCE($4, external_ref), attempts = attempts + $5,
		version = version + 1, updated_at = $6 WHERE operation_id = $1 AND version = $2`, o.OperationID, o.Version, to, ext, inc, at)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrState
	}
	from := o.State
	return s.event(ctx, tx, o.OperationID, &from, to, kind, evidence, actor, at)
}

// Dispatch sends one operation through port. The DISPATCHING state is
// committed before the call; the outcome is recorded after it. A disabled
// port sends nothing and changes nothing.
func (s *Service) Dispatch(ctx context.Context, port Port, operationID string) (Operation, error) {
	if port == nil {
		return Operation{}, invalid("port")
	}
	var o Operation
	if err := s.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		var ok bool
		var err error
		o, ok, err = load(ctx, tx, `operation_id = $1`, operationID, true)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		if o.Provider != port.Provider() {
			return invalid("port provider")
		}
		if o.State != StatePrepared && o.State != StateAbsent {
			// UNKNOWN / RECONCILIATION_REQUIRED are never retried blindly;
			// BLOCKED (D-38) and final states are never dispatched.
			return ErrState
		}
		if !port.Enabled() {
			return ErrPortDisabled
		}
		at := s.clock.Now().Time().UTC()
		if err := s.transition(ctx, tx, o, StateDispatching, "DISPATCH_STARTED", nil, map[string]string{"synthetic": fmt.Sprint(port.Synthetic())},
			"service:integration", at, true); err != nil {
			return err
		}
		o.State, o.Version = StateDispatching, o.Version+1
		return nil
	}); err != nil {
		return o, err
	}
	var payload map[string]string
	resp, callErr := port.Submit(ctx, Request{OperationID: o.OperationID, OpType: o.OpType, Reference: o.Reference, CorrelationID: o.CorrelationID,
		Payload: payload})
	err := s.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		cur, ok, err := load(ctx, tx, `operation_id = $1`, o.OperationID, true)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		at := s.clock.Now().Time().UTC()
		if cur.State != StateDispatching {
			// A callback already moved it (callback before acknowledgement):
			// keep the callback's evidence; record the late answer only.
			return s.event(ctx, tx, cur.OperationID, &cur.State, cur.State, "ACKNOWLEDGED", map[string]string{"late_dispatch_answer": string(resp.Outcome)},
				"service:integration", at)
		}
		switch {
		case callErr != nil || resp.Outcome == OutcomeUnknown || resp.Outcome == "":
			ev := map[string]string{"reason": "timeout or ambiguous provider answer"}
			if callErr != nil {
				ev["error"] = truncate(callErr.Error())
			}
			return s.transition(ctx, tx, cur, StateUnknown, "OUTCOME_UNKNOWN", nil, ev, "service:integration", at, false)
		case resp.Outcome == OutcomeAcknowledged && resp.ExternalRef != "":
			ref := resp.ExternalRef
			return s.transition(ctx, tx, cur, StateAcknowledged, "ACKNOWLEDGED", &ref, resp.Evidence, "service:integration", at, false)
		case resp.Outcome == OutcomeRejected:
			return s.transition(ctx, tx, cur, StateRejected, "REJECTED", nil, resp.Evidence, "service:integration", at, false)
		default:
			return s.transition(ctx, tx, cur, StateUnknown, "OUTCOME_UNKNOWN", nil, map[string]string{"reason": "acknowledgement without external reference"},
				"service:integration", at, false)
		}
	})
	if err != nil {
		return o, err
	}
	return s.Get(ctx, o.OperationID)
}

func truncate(s string) string {
	if len(s) > 300 {
		return s[:300]
	}
	return s
}

// Reconcile resolves an UNKNOWN operation through the provider lookup:
// found → CONFIRMED, proven absent → CONFIRMED_ABSENT (a retry is then
// permitted), lookup unsupported or failing → RECONCILIATION_REQUIRED.
func (s *Service) Reconcile(ctx context.Context, port Port, operationID string) (Operation, error) {
	var o Operation
	if err := s.runner.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
		var ok bool
		var err error
		o, ok, err = load(ctx, tx, `operation_id = $1`, operationID, false)
		if err == nil && !ok {
			err = ErrNotFound
		}
		return err
	}); err != nil {
		return o, err
	}
	if o.State != StateUnknown {
		return o, ErrState
	}
	var lk Lookup
	var lkErr error
	if port != nil && port.Enabled() {
		lk, lkErr = port.Lookup(ctx, o.Reference)
	}
	err := s.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		cur, ok, err := load(ctx, tx, `operation_id = $1`, operationID, true)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		if cur.State != StateUnknown {
			return ErrState
		}
		at := s.clock.Now().Time().UTC()
		switch {
		case port == nil || !port.Enabled() || lkErr != nil || !lk.Supported:
			ev := map[string]string{"reason": "provider lookup unavailable; human evidence required"}
			if lkErr != nil {
				ev["error"] = truncate(lkErr.Error())
			}
			return s.transition(ctx, tx, cur, StateReconciliation, "LOOKUP_UNAVAILABLE", nil, ev, "service:integration", at, false)
		case lk.Found && lk.ExternalRef != "":
			ref := lk.ExternalRef
			return s.transition(ctx, tx, cur, StateConfirmed, "LOOKUP_FOUND", &ref, lk.Evidence, "service:integration", at, false)
		case !lk.Found:
			return s.transition(ctx, tx, cur, StateAbsent, "LOOKUP_ABSENT", nil, lk.Evidence, "service:integration", at, false)
		default:
			return s.transition(ctx, tx, cur, StateReconciliation, "LOOKUP_UNAVAILABLE", nil, map[string]string{"reason": "found without external reference"},
				"service:integration", at, false)
		}
	})
	if err != nil {
		return o, err
	}
	return s.Get(ctx, operationID)
}

// Resolution is recorded human reconciliation evidence.
type Resolution struct {
	OperationID     string
	Present         bool
	ExternalRef     string
	EvidenceNote    string
	ExpectedVersion int64
	ActorUserID     string
}

// Resolve records human evidence for an UNKNOWN / RECONCILIATION_REQUIRED
// operation (an authorised integration operator; the HTTP layer checks the
// capability). Present needs the external reference observed at the
// provider; absent permits a later dispatch.
func (s *Service) Resolve(ctx context.Context, r Resolution) (Operation, error) {
	if !uuidPattern.MatchString(r.OperationID) || !uuidPattern.MatchString(r.ActorUserID) || len(r.EvidenceNote) < 3 || len(r.EvidenceNote) > 1000 ||
		r.ExpectedVersion < 1 || (r.Present && (r.ExternalRef == "" || len(r.ExternalRef) > 191)) || (!r.Present && r.ExternalRef != "") {
		return Operation{}, invalid("resolution")
	}
	err := s.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		cur, ok, err := load(ctx, tx, `operation_id = $1`, r.OperationID, true)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		if cur.State != StateUnknown && cur.State != StateReconciliation {
			return ErrState
		}
		if cur.Version != r.ExpectedVersion {
			return ErrState
		}
		at := s.clock.Now().Time().UTC()
		ev := map[string]string{"note": r.EvidenceNote}
		if r.Present {
			ref := r.ExternalRef
			return s.transition(ctx, tx, cur, StateConfirmed, "RESOLVED_PRESENT", &ref, ev, "user:"+r.ActorUserID, at, false)
		}
		return s.transition(ctx, tx, cur, StateAbsent, "RESOLVED_ABSENT", nil, ev, "user:"+r.ActorUserID, at, false)
	})
	if err != nil {
		return Operation{}, err
	}
	return s.Get(ctx, r.OperationID)
}

// CallbackEnvelopeVersion is the only supported envelope version.
//
// The envelope is a SYNTHETIC contract owned by this ledger. 2.0.160 has no
// Biawin callback at all (the legacy client only polls; see the Slice 14
// audit), so this is not a Biawin protocol: a real connector must map its
// authenticated provider payload onto this envelope inside the adapter, and
// none is wired (no callback route exists).
const CallbackEnvelopeVersion = 1

// CallbackEnvelope is the authenticated content of a callback. Every field
// that drives a transition, the dedupe identity or the recorded disposition
// comes from here, never from transport metadata.
type CallbackEnvelope struct {
	Version     int    `json:"v"`
	CallbackID  string `json:"callback_id"`
	Provider    string `json:"provider"`
	Reference   string `json:"reference"`
	ExternalRef string `json:"external_ref"`
	Status      string `json:"status"` // CONFIRMED / REJECTED
}

// EncodeCallback returns the canonical body for e: compact JSON with the
// fields in declaration order. The signature is HMAC-SHA256 over exactly
// these bytes.
func EncodeCallback(e CallbackEnvelope) ([]byte, error) {
	return json.Marshal(e)
}

// DecodeCallback parses an authenticated body strictly: one object, no
// unknown fields, no trailing data, the supported version, and the exact
// canonical encoding (so duplicate keys, reordered fields or whitespace
// cannot make two parsers disagree about what was signed).
func DecodeCallback(body []byte) (CallbackEnvelope, error) {
	var e CallbackEnvelope
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&e); err != nil {
		return CallbackEnvelope{}, fmt.Errorf("%w: %v", ErrCallbackMalformed, err)
	}
	if _, err := dec.Token(); err != io.EOF {
		return CallbackEnvelope{}, fmt.Errorf("%w: trailing data", ErrCallbackMalformed)
	}
	if e.Version != CallbackEnvelopeVersion {
		return CallbackEnvelope{}, fmt.Errorf("%w: version %d", ErrCallbackMalformed, e.Version)
	}
	canon, err := EncodeCallback(e)
	if err != nil || !bytes.Equal(canon, body) {
		return CallbackEnvelope{}, fmt.Errorf("%w: not canonical", ErrCallbackMalformed)
	}
	return e, nil
}

// Callback is one received callback: the signed body and its signature,
// plus optional transport claims (headers, query parameters an adapter may
// have parsed). Claims are never trusted: a non-empty claim must equal the
// authenticated envelope or the callback is refused with nothing recorded.
type Callback struct {
	Body      []byte // canonical CallbackEnvelope (EncodeCallback)
	Signature string // hex HMAC-SHA256(secret, Body)

	CallbackID  string // optional claim
	Provider    string // optional claim
	Reference   string // optional claim
	ExternalRef string // optional claim
	Status      string // optional claim
}

// Verify checks a callback signature in constant time.
func Verify(secret []byte, body []byte, signature string) bool {
	if len(secret) == 0 {
		return false
	}
	m := hmac.New(sha256.New, secret)
	m.Write(body)
	want := hex.EncodeToString(m.Sum(nil))
	return hmac.Equal([]byte(want), []byte(signature))
}

// authenticate verifies the signature, decodes the envelope and rejects any
// transport claim that disagrees with it.
func authenticate(secret []byte, cb Callback) (CallbackEnvelope, error) {
	if !Verify(secret, cb.Body, cb.Signature) {
		return CallbackEnvelope{}, ErrSignature
	}
	e, err := DecodeCallback(cb.Body)
	if err != nil {
		return CallbackEnvelope{}, err
	}
	for _, c := range []struct{ name, claim, signed string }{
		{"callback_id", cb.CallbackID, e.CallbackID}, {"provider", cb.Provider, e.Provider}, {"reference", cb.Reference, e.Reference},
		{"external_ref", cb.ExternalRef, e.ExternalRef}, {"status", cb.Status, e.Status},
	} {
		if c.claim != "" && c.claim != c.signed {
			return CallbackEnvelope{}, fmt.Errorf("%w: %s", ErrCallbackMismatch, c.name)
		}
	}
	if e.CallbackID == "" || len(e.CallbackID) > 128 || e.Provider != "BIAWIN" || (e.Status != "CONFIRMED" && e.Status != "REJECTED") ||
		e.Reference == "" || len(e.Reference) > 191 || len(e.ExternalRef) > 191 || (e.Status == "CONFIRMED" && e.ExternalRef == "") {
		return CallbackEnvelope{}, invalid("callback")
	}
	return e, nil
}

// HandleCallback verifies and records one callback. Only the authenticated
// envelope decides anything: the dedupe identity, the matched operation,
// the transition and the recorded disposition, all bound to the SHA-256 of
// the signed body. A replay of the same signed body is DUPLICATE; the same
// callback id with a different signed body is ID_CONFLICT and changes
// nothing; a callback arriving before the dispatch answer is applied
// (EARLY_APPLIED); a callback naming a different external reference than
// the one recorded moves the operation to RECONCILIATION_REQUIRED.
func (s *Service) HandleCallback(ctx context.Context, secret []byte, raw Callback) (string, error) {
	cb, err := authenticate(secret, raw)
	if err != nil {
		return "", err
	}
	body := sha256.Sum256(raw.Body)
	var disposition string
	err = s.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		at := s.clock.Now().Time().UTC()
		var seenSHA []byte
		switch err := tx.QueryRow(ctx, `SELECT body_sha256 FROM integration.provider_callback WHERE callback_id = $1`, cb.CallbackID).Scan(&seenSHA); {
		case err == nil:
			if bytes.Equal(seenSHA, body[:]) {
				disposition = "DUPLICATE"
			} else {
				disposition = "ID_CONFLICT" // the same id with different signed content: never applied
			}
			return nil
		case !errors.Is(err, pgx.ErrNoRows):
			return err
		}
		op, ok, err := load(ctx, tx, `reference = $1`, cb.Reference, true)
		if err != nil {
			return err
		}
		var opID any
		var ext any
		if cb.ExternalRef != "" {
			ext = cb.ExternalRef
		}
		evidence := map[string]string{"callback_id": cb.CallbackID, "body_sha256": hex.EncodeToString(body[:])}
		switch {
		case !ok:
			disposition = "UNMATCHED"
		default:
			opID = op.OperationID
			switch op.State {
			case StateConfirmed, StateRejected, StateBlocked:
				disposition = "ALREADY_FINAL"
				if op.State == StateConfirmed && op.ExternalRef != nil && cb.ExternalRef != "" && *op.ExternalRef != cb.ExternalRef {
					disposition = "CONFLICT"
				}
			default:
				if op.ExternalRef != nil && cb.ExternalRef != "" && *op.ExternalRef != cb.ExternalRef {
					disposition = "CONFLICT"
					if err := s.transition(ctx, tx, op, StateReconciliation, "REFERENCE_CONFLICT", nil,
						map[string]string{"recorded": *op.ExternalRef, "callback": cb.ExternalRef, "callback_id": cb.CallbackID, "body_sha256": evidence["body_sha256"]},
						"provider:callback", at, false); err != nil {
						return err
					}
					break
				}
				disposition = "APPLIED"
				if op.State == StateDispatching || op.State == StatePrepared {
					disposition = "EARLY_APPLIED"
				}
				if cb.Status == "CONFIRMED" {
					ref := cb.ExternalRef
					err = s.transition(ctx, tx, op, StateConfirmed, "CALLBACK_CONFIRMED", &ref, evidence, "provider:callback", at, false)
				} else {
					err = s.transition(ctx, tx, op, StateRejected, "CALLBACK_REJECTED", nil, evidence, "provider:callback", at, false)
				}
				if err != nil {
					return err
				}
			}
			if disposition == "CONFLICT" && op.State == StateConfirmed {
				if err := s.event(ctx, tx, op.OperationID, &op.State, op.State, "REFERENCE_CONFLICT",
					map[string]string{"callback": cb.ExternalRef, "callback_id": cb.CallbackID, "body_sha256": evidence["body_sha256"]}, "provider:callback", at); err != nil {
					return err
				}
			}
		}
		_, err = tx.Exec(ctx, `INSERT INTO integration.provider_callback (callback_id, provider, reference, operation_id, external_ref, status, disposition,
			body_sha256, received_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, cb.CallbackID, cb.Provider, cb.Reference, opID, ext, cb.Status, disposition, body[:], at)
		return err
	})
	return disposition, err
}

// Get returns one operation.
func (s *Service) Get(ctx context.Context, operationID string) (Operation, error) {
	var o Operation
	err := s.runner.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
		var ok bool
		var err error
		o, ok, err = load(ctx, tx, `operation_id = $1`, operationID, false)
		if err == nil && !ok {
			err = ErrNotFound
		}
		return err
	})
	return o, err
}

// List returns operations, optionally of one state (newest first).
func (s *Service) List(ctx context.Context, state *string, limit int) ([]Operation, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	out := []Operation{}
	err := s.runner.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
		rows, err := tx.Query(ctx, `SELECT `+cols+` FROM integration.provider_operation WHERE ($1::text IS NULL OR state = $1)
			ORDER BY updated_at DESC, operation_id LIMIT $2`, state, limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			o, err := scan(rows)
			if err != nil {
				return err
			}
			out = append(out, o)
		}
		return rows.Err()
	})
	return out, err
}

// Confirmed reports whether any operation of a case is CONFIRMED (the
// read fulfillment's completion guard uses through its composition root).
func (s *Service) Confirmed(ctx context.Context, caseID string) (bool, error) {
	var n int
	err := s.runner.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
		return tx.QueryRow(ctx, `SELECT count(*) FROM integration.provider_operation WHERE subject_case_id = $1 AND state = 'CONFIRMED'`, caseID).Scan(&n)
	})
	return n > 0, err
}
