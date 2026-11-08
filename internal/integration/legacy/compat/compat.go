// Package compat is the client side of the PROPOSED legacy write-back
// compatibility contract (R2, E-3). It implements the legacy/writeback
// Legacy port over HTTPS.
//
// No such endpoint exists in the canonical legacy baseline (plugin 2.0.160):
// its only REST route is the read-only campaign report, and every write path
// the write-back needs (lead outcome, HR change request create / review) is
// an interactive admin-ajax / admin-post handler behind a logged-in
// WordPress session and a nonce. A service cannot use those without reusing
// user sessions, which is forbidden. The contract this client speaks is
// therefore a PROPOSAL (docs/remediation/r2-writeback-contract.md); it must
// be implemented on the legacy side by a separately reviewed change and
// approved before this client can be constructed outside tests:
//
//   - New refuses unless the approval names approvedContract, which is
//     empty: no contract is approved, so no configuration, environment
//     variable or flag can activate a real legacy write-back.
//   - The deterministic synthetic store (legacytest) and an in-process fake
//     server (tests) are the only counterparts this code has ever talked to.
//
// Safety properties of the client (independent of the counterpart):
//
//   - only HTTPS endpoints; no redirects; bounded request and response size;
//   - every request carries the contract version, the operation, the
//     request token (= target command id; the legacy idempotency key and
//     reconciliation key), the correlation id and, where the legacy state is
//     known, the expected source state;
//   - requests are signed (Credentials, an abstraction; keys never leave
//     it and are never logged); responses must carry a valid server
//     signature over the exact body;
//   - a response is accepted only if it is HTTP 200, signed, strictly
//     schema-valid (unknown fields, trailing data, missing fields refused)
//     and semantically bound to the request (same contract, operation and
//     token). Anything else — timeout, transport error, other status, bad
//     signature, invalid schema, mismatch — is an UNKNOWN outcome (an
//     error), never a success and never a rejection;
//   - every exchange is recorded (digests and verdict only) before the
//     result is returned; a failed record makes the outcome UNKNOWN.
package compat

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"

	lwb "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/writeback"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
)

// ContractVersion is the PROPOSED contract this client speaks.
const ContractVersion = "sn-crm-writeback-compat/0.1-proposed"

// approvedContract is the approved contract identifier. It is empty: no
// legacy write-back contract is approved (E-3 BLOCKED). Setting it is a
// reviewed code change that must cite the approval record and the verified
// legacy endpoint evidence.
const approvedContract = ""

// Signature headers.
const (
	HeaderKeyID     = "X-SN-Key-Id"
	HeaderSignature = "X-SN-Signature"
)

// Bounds.
const (
	MaxBodyBytes   = 16 << 10
	MaxTimeout     = 30 * time.Second
	DefaultTimeout = 10 * time.Second
)

// Errors.
var (
	// ErrNotApproved: no approved contract; the client cannot be built.
	ErrNotApproved = errors.New("compat: no approved legacy write-back contract (E-3 BLOCKED)")
	// ErrConfig reports an unsafe or incomplete configuration.
	ErrConfig = errors.New("compat: invalid configuration")
	// ErrOutcomeUnknown wraps every exchange whose outcome is not proven.
	ErrOutcomeUnknown = errors.New("compat: legacy outcome unknown")
)

// Credentials is the authentication abstraction. Implementations hold the
// key material (e.g. from a secret manager); this package never sees it.
type Credentials interface {
	KeyID() string
	// Sign returns the request signature over body.
	Sign(body []byte) (string, error)
	// VerifyResponse reports whether signature is the counterpart's valid
	// signature over body.
	VerifyResponse(body []byte, signature string) bool
}

// Verdict classifies one exchange.
type Verdict string

// Verdicts (the integration.writeback_exchange vocabulary).
const (
	VerdictValid        Verdict = "VALID"
	VerdictTransport    Verdict = "TRANSPORT_ERROR"
	VerdictTimeout      Verdict = "TIMEOUT"
	VerdictHTTPStatus   Verdict = "HTTP_STATUS"
	VerdictBadSignature Verdict = "BAD_SIGNATURE"
	VerdictSchema       Verdict = "INVALID_SCHEMA"
	VerdictMismatch     Verdict = "MISMATCH"
)

// Exchange is the evidence of one request / response.
type Exchange struct {
	RequestToken, Operation, Contract, CorrelationID string
	RequestSHA256                                    [32]byte
	ResponseSHA256                                   *[32]byte
	HTTPStatus                                       int
	Verdict                                          Verdict
	Outcome                                          string // APPLIED / REJECTED / OBSERVED when Verdict is VALID
	StartedAt, FinishedAt                            time.Time
}

// Recorder persists exchanges (integration.writeback_exchange).
type Recorder interface {
	Record(ctx context.Context, e Exchange) error
}

// Config configures the client.
type Config struct {
	Endpoint    string
	Timeout     time.Duration
	Credentials Credentials
	Recorder    Recorder
	Clock       timeutil.Clock
	// Transport is the HTTP round tripper (nil: a dedicated transport).
	Transport http.RoundTripper
}

// Approval identifies the approved contract and the verified endpoint.
type Approval struct {
	ContractID string
}

// Client implements lwb.Legacy over the compatibility contract.
type Client struct {
	endpoint *url.URL
	timeout  time.Duration
	creds    Credentials
	rec      Recorder
	clock    timeutil.Clock
	http     *http.Client
}

var _ lwb.Legacy = (*Client)(nil)

// New builds a client for an APPROVED contract. While approvedContract is
// empty it always returns ErrNotApproved.
func New(cfg Config, a Approval) (*Client, error) {
	if approvedContract == "" || a.ContractID != approvedContract {
		return nil, ErrNotApproved
	}
	return newClient(cfg)
}

func newClient(cfg Config) (*Client, error) {
	u, err := url.Parse(cfg.Endpoint)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("%w: endpoint must be an https URL without credentials, query or fragment", ErrConfig)
	}
	if cfg.Credentials == nil || cfg.Credentials.KeyID() == "" || cfg.Recorder == nil || cfg.Clock == nil {
		return nil, fmt.Errorf("%w: credentials, recorder and clock are required", ErrConfig)
	}
	t := cfg.Timeout
	if t == 0 {
		t = DefaultTimeout
	}
	if t < 0 || t > MaxTimeout {
		return nil, fmt.Errorf("%w: timeout", ErrConfig)
	}
	rt := cfg.Transport
	if rt == nil {
		rt = &http.Transport{Proxy: nil, MaxIdleConns: 4, IdleConnTimeout: 30 * time.Second, TLSHandshakeTimeout: 5 * time.Second,
			ResponseHeaderTimeout: t, ForceAttemptHTTP2: true}
	}
	hc := &http.Client{Transport: rt, Timeout: t, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return &Client{endpoint: u, timeout: t, creds: cfg.Credentials, rec: cfg.Recorder, clock: cfg.Clock, http: hc}, nil
}

// request is the canonical request envelope.
type request struct {
	Contract      string            `json:"contract"`
	Operation     string            `json:"operation"`
	RequestToken  string            `json:"request_token"`
	CorrelationID string            `json:"correlation_id"`
	Expected      map[string]string `json:"expected"`
	Payload       map[string]string `json:"payload"`
}

// response is the canonical response envelope.
type response struct {
	Contract     string            `json:"contract"`
	Operation    string            `json:"operation"`
	RequestToken string            `json:"request_token"`
	Outcome      string            `json:"outcome"`
	LegacyRef    string            `json:"legacy_ref"`
	Reason       string            `json:"reason"`
	Matches      int               `json:"matches"`
	State        map[string]string `json:"state"`
}

var (
	tokenPattern  = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)
	refPattern    = regexp.MustCompile(`^[0-9]{1,20}$`)
	reasonPattern = regexp.MustCompile(`^[A-Z][A-Z_]{2,47}$`)
	stateKey      = regexp.MustCompile(`^[a-z][a-z0-9_]{0,47}$`)
)

// rejection maps a contract rejection reason to a proven non-commit status.
var rejection = map[string]result.Status{
	"OWNER_MISMATCH": result.Failed, "ARCHIVED": result.Failed, "NOT_FOUND": result.Failed, "DUPLICATE_OPEN_REQUEST": result.Conflict,
	"STATE_CONFLICT": result.Conflict, "NOT_AUTHORIZED_REVIEWER": result.Unauthorized, "VALIDATION_FAILED": result.Failed,
}

type outcomeErr struct {
	v   Verdict
	msg string
}

func (e outcomeErr) Error() string { return fmt.Sprintf("%v: %s: %s", ErrOutcomeUnknown, e.v, e.msg) }
func (e outcomeErr) Unwrap() error { return ErrOutcomeUnknown }

// exchange sends one request and returns the validated response.
func (c *Client) exchange(ctx context.Context, op, token string, expected, payload map[string]string, observe bool) (response, error) {
	if !tokenPattern.MatchString(token) {
		return response{}, fmt.Errorf("%w: request token", ErrConfig)
	}
	if expected == nil {
		expected = map[string]string{}
	}
	if payload == nil {
		payload = map[string]string{}
	}
	corr := lwb.CorrelationFrom(ctx)
	body, err := json.Marshal(request{Contract: ContractVersion, Operation: op, RequestToken: token, CorrelationID: corr, Expected: expected,
		Payload: payload})
	if err != nil || len(body) > MaxBodyBytes {
		return response{}, fmt.Errorf("%w: request body", ErrConfig)
	}
	ex := Exchange{RequestToken: token, Operation: op, Contract: ContractVersion, CorrelationID: corr, RequestSHA256: sha256.Sum256(body),
		StartedAt: c.clock.Now().Time()}
	resp, verdict, status, raw, err := c.send(ctx, body, op, token, observe)
	ex.Verdict, ex.HTTPStatus, ex.FinishedAt = verdict, status, c.clock.Now().Time()
	if ex.FinishedAt.Before(ex.StartedAt) {
		ex.FinishedAt = ex.StartedAt
	}
	if raw != nil {
		d := sha256.Sum256(raw)
		ex.ResponseSHA256 = &d
	}
	if verdict == VerdictValid {
		ex.Outcome = resp.Outcome
	}
	// The evidence is recorded before the answer is used; without it the
	// outcome is not provable and stays UNKNOWN.
	if rerr := c.rec.Record(context.WithoutCancel(ctx), ex); rerr != nil {
		return response{}, outcomeErr{v: verdict, msg: "exchange evidence not recorded: " + rerr.Error()}
	}
	if err != nil {
		return response{}, err
	}
	return resp, nil
}

func (c *Client) send(ctx context.Context, body []byte, op, token string, observe bool) (response, Verdict, int, []byte, error) {
	sig, err := c.creds.Sign(body)
	if err != nil {
		return response{}, VerdictTransport, 0, nil, outcomeErr{v: VerdictTransport, msg: "signing failed"}
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return response{}, VerdictTransport, 0, nil, outcomeErr{v: VerdictTransport, msg: "request"}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(HeaderKeyID, c.creds.KeyID())
	req.Header.Set(HeaderSignature, sig)
	hr, err := c.http.Do(req)
	if err != nil {
		v := VerdictTransport
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			v = VerdictTimeout
		}
		// The request may or may not have reached legacy: UNKNOWN.
		return response{}, v, 0, nil, outcomeErr{v: v, msg: "no response"}
	}
	defer hr.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(hr.Body, MaxBodyBytes+1))
	if err != nil {
		return response{}, VerdictTransport, hr.StatusCode, nil, outcomeErr{v: VerdictTransport, msg: "response read"}
	}
	if len(raw) > MaxBodyBytes {
		return response{}, VerdictSchema, hr.StatusCode, raw[:MaxBodyBytes], outcomeErr{v: VerdictSchema, msg: "response too large"}
	}
	if hr.StatusCode != http.StatusOK {
		// Only a signed, valid 200 envelope carries an outcome. Any other
		// status (including 4xx / 5xx bodies) proves nothing.
		return response{}, VerdictHTTPStatus, hr.StatusCode, raw, outcomeErr{v: VerdictHTTPStatus, msg: fmt.Sprintf("status %d", hr.StatusCode)}
	}
	if !c.creds.VerifyResponse(raw, hr.Header.Get(HeaderSignature)) {
		return response{}, VerdictBadSignature, hr.StatusCode, raw, outcomeErr{v: VerdictBadSignature, msg: "response signature"}
	}
	var r response
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&r); err != nil || dec.More() {
		return response{}, VerdictSchema, hr.StatusCode, raw, outcomeErr{v: VerdictSchema, msg: "response envelope"}
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return response{}, VerdictSchema, hr.StatusCode, raw, outcomeErr{v: VerdictSchema, msg: "trailing data"}
	}
	if r.Contract != ContractVersion || r.Operation != op || r.RequestToken != token {
		return response{}, VerdictMismatch, hr.StatusCode, raw, outcomeErr{v: VerdictMismatch, msg: "response not bound to the request"}
	}
	if err := validate(r, observe); err != nil {
		return response{}, VerdictSchema, hr.StatusCode, raw, outcomeErr{v: VerdictSchema, msg: err.Error()}
	}
	return r, VerdictValid, hr.StatusCode, raw, nil
}

func validate(r response, observe bool) error {
	for k, v := range r.State {
		if !stateKey.MatchString(k) || len(v) > 256 {
			return errors.New("state entry")
		}
	}
	switch {
	case observe:
		if r.Outcome != "OBSERVED" || r.Matches < 0 || r.Matches > 1000 || r.Reason != "" || r.LegacyRef != "" && !refPattern.MatchString(r.LegacyRef) {
			return errors.New("observation envelope")
		}
		if r.Matches == 1 && r.LegacyRef == "" {
			return errors.New("a single match needs its legacy reference")
		}
	case r.Outcome == "APPLIED":
		if !refPattern.MatchString(r.LegacyRef) || r.Reason != "" || r.Matches != 0 || len(r.State) != 0 {
			return errors.New("applied envelope")
		}
	case r.Outcome == "REJECTED":
		if _, ok := rejection[r.Reason]; !ok || !reasonPattern.MatchString(r.Reason) || r.LegacyRef != "" || r.Matches != 0 || len(r.State) != 0 {
			return errors.New("rejected envelope")
		}
	default:
		return errors.New("outcome")
	}
	return nil
}

func applied(r response) lwb.Result {
	if r.Outcome == "APPLIED" {
		return lwb.Result{Outcome: result.Success, Ref: r.LegacyRef}
	}
	return lwb.Result{Outcome: rejection[r.Reason], Reason: r.Reason}
}

// ApplyLeadOutcome implements lwb.Legacy.
func (c *Client) ApplyLeadOutcome(ctx context.Context, op lwb.LeadOutcome) (lwb.Result, error) {
	r, err := c.exchange(ctx, "lead_outcome.apply", op.RequestToken, nil, map[string]string{"lead_id": op.LeadID, "status_label": op.StatusLabel,
		"not_purchase_reason": op.NotPurchaseReason, "callback_jalali": op.CallbackJalali, "actor_legacy_user_id": op.ActorLegacyUserID}, false)
	if err != nil {
		return lwb.Result{}, err
	}
	return applied(r), nil
}

// ObserveLeadOutcome implements lwb.Legacy.
func (c *Client) ObserveLeadOutcome(ctx context.Context, leadID, requestToken string) (lwb.LeadObserved, error) {
	r, err := c.exchange(ctx, "lead_outcome.observe", requestToken, nil, map[string]string{"lead_id": leadID}, true)
	if err != nil {
		return lwb.LeadObserved{}, err
	}
	return lwb.LeadObserved{Events: r.Matches, StatusLabel: r.State["status_label"], NotPurchaseReason: r.State["not_purchase_reason"],
		CallbackJalali: r.State["callback_jalali"], EventRef: r.LegacyRef}, nil
}

// InsertHRRequest implements lwb.Legacy.
func (c *Client) InsertHRRequest(ctx context.Context, op lwb.HRCreate) (lwb.Result, error) {
	r, err := c.exchange(ctx, "hr_request.create", op.ReconcileKey, nil, map[string]string{"request_type": op.RequestType,
		"subject_legacy_user_id": op.SubjectLegacyUserID, "target_parent_legacy_user_id": op.TargetParentLegacyUserID,
		"requested_by_legacy_user_id": op.RequestedByLegacyUserID, "reviewer_legacy_user_id": op.ReviewerLegacyUserID, "status": op.Status,
		"reason": op.Reason}, false)
	if err != nil {
		return lwb.Result{}, err
	}
	return applied(r), nil
}

func hrObserved(r response) lwb.HRObserved {
	return lwb.HRObserved{Matches: r.Matches, LegacyRequestID: r.LegacyRef, Status: r.State["status"], ReviewerLegacyUserID: r.State["reviewer_legacy_user_id"],
		RequestType: r.State["request_type"], SubjectLegacyUserID: r.State["subject_legacy_user_id"],
		TargetParentLegacyUserID: r.State["target_parent_legacy_user_id"]}
}

// ObserveHRCreate implements lwb.Legacy.
func (c *Client) ObserveHRCreate(ctx context.Context, reconcileKey string) (lwb.HRObserved, error) {
	r, err := c.exchange(ctx, "hr_request.create.observe", reconcileKey, nil, nil, true)
	if err != nil {
		return lwb.HRObserved{}, err
	}
	return hrObserved(r), nil
}

// UpdateHRRequest implements lwb.Legacy. The expected source state is the
// request's current legacy status: legacy must refuse (STATE_CONFLICT) a
// decision on a request that moved, never overwrite it.
func (c *Client) UpdateHRRequest(ctx context.Context, op lwb.HRDecide) (lwb.Result, error) {
	r, err := c.exchange(ctx, "hr_request.decide", op.ReconcileKey, map[string]string{"legacy_request_id": op.LegacyRequestID, "status": op.FromStatus},
		map[string]string{"decision": op.Decision, "to_status": op.ToStatus, "next_reviewer_legacy_user_id": op.NextReviewerLegacyUserID,
			"notes": op.Notes, "actor_legacy_user_id": op.ActorLegacyUserID}, false)
	if err != nil {
		return lwb.Result{}, err
	}
	return applied(r), nil
}

// ObserveHRDecide implements lwb.Legacy.
func (c *Client) ObserveHRDecide(ctx context.Context, legacyRequestID, reconcileKey string) (lwb.HRObserved, error) {
	r, err := c.exchange(ctx, "hr_request.decide.observe", reconcileKey, nil, map[string]string{"legacy_request_id": legacyRequestID}, true)
	if err != nil {
		return lwb.HRObserved{}, err
	}
	return hrObserved(r), nil
}
