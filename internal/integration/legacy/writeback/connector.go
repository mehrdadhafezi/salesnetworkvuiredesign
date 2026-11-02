// Package writeback is the legacy (2.0.160) side of the Slice 11 write-back:
// it maps neutral write-back intents to the legacy operations proven in the
// source audit and back. It is the only place that knows those operations:
//
//   - contact outcome → ajax_update_lead_status + SN_Seller_Flow::apply_activity
//     (class-sn-plugin.php:28002-28107; class-sn-seller-flow.php:283-471):
//     lead id, Persian status label, not-purchase reason, Jalali Tehran
//     callback time, request_token = command id (legacy UNIQUE request_token
//     makes a repeated apply idempotent and is the reconciliation key);
//   - HR request create / step decision → handle_hr_change_request_create /
//     handle_hr_change_request_review step + reject branches
//     (class-sn-plugin.php:10443-10568); the approval_path event carries the
//     target command id as the reconciliation key.
//
// The Legacy interface is what a legacy store must implement. Slice 11 ships
// no production implementation (no production access): tests use the
// synthetic legacy fixture.
package writeback

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // Asia/Tehran display conversion without host tz data

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/shadow"
	wb "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/writeback"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
)

// Seller-flow labels (SN_Seller_Flow::statuses, class-sn-seller-flow.php:168-176).
var outcomeLabels = map[string]string{"NO_ANSWER": "جواب نداده", "CALLBACK": "تماس مجدد", "NOT_PURCHASED": "عدم خرید"}

// LeadOutcome is the legacy lead outcome update.
type LeadOutcome struct {
	LeadID, StatusLabel, NotPurchaseReason, CallbackJalali, RequestToken, ActorLegacyUserID string
}

// LeadObserved is the legacy state found for a lead outcome request token.
type LeadObserved struct {
	// Events is the number of seller-flow events carrying the request token.
	Events                                         int
	StatusLabel, NotPurchaseReason, CallbackJalali string
	EventRef                                       string
}

// HRCreate is the legacy HR change request insert.
type HRCreate struct {
	RequestType, SubjectLegacyUserID, TargetParentLegacyUserID, RequestedByLegacyUserID, ReviewerLegacyUserID string
	Status, Reason, ReconcileKey                                                                              string
}

// HRDecide is the legacy HR change request step / rejection update.
type HRDecide struct {
	LegacyRequestID, Decision, FromStatus, ToStatus, NextReviewerLegacyUserID, Notes, ActorLegacyUserID, ReconcileKey string
}

// HRObserved is the legacy state found for an HR reconciliation key.
type HRObserved struct {
	// Matches is the number of legacy approval-path events carrying the key.
	Matches                                                    int
	LegacyRequestID, Status, ReviewerLegacyUserID, RequestType string
	SubjectLegacyUserID, TargetParentLegacyUserID              string
}

// Result is a legacy store's answer. Outcome SUCCESS means the effect is
// committed; FAILED / CONFLICT / BLOCKED / UNAUTHORIZED mean proven
// non-commit; an error means the outcome is unknown.
type Result struct {
	Outcome result.Status
	Reason  string
	Ref     string
}

// Legacy is the legacy master store.
type Legacy interface {
	ApplyLeadOutcome(ctx context.Context, op LeadOutcome) (Result, error)
	ObserveLeadOutcome(ctx context.Context, leadID, requestToken string) (LeadObserved, error)
	InsertHRRequest(ctx context.Context, op HRCreate) (Result, error)
	ObserveHRCreate(ctx context.Context, reconcileKey string) (HRObserved, error)
	UpdateHRRequest(ctx context.Context, op HRDecide) (Result, error)
	ObserveHRDecide(ctx context.Context, legacyRequestID, reconcileKey string) (HRObserved, error)
}

// Users maps between target user ids and legacy user keys (identity
// published read API).
type Users interface {
	UserKeys(ctx context.Context, system string, ids []string) (map[string]string, error)
	ResolveUsers(ctx context.Context, system string, keys []string) (map[string]string, error)
}

// Connector implements the integration write-back port over a legacy store.
type Connector struct {
	legacy Legacy
	users  Users
	tehran *time.Location
}

var _ wb.Connector = (*Connector)(nil)

// NewConnector returns the connector.
func NewConnector(legacy Legacy, users Users) (*Connector, error) {
	if legacy == nil || users == nil {
		return nil, errors.New("legacy/writeback: legacy store and users are required")
	}
	loc, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		return nil, err
	}
	return &Connector{legacy: legacy, users: users, tehran: loc}, nil
}

// RequestToken is the legacy request token of a command (sanitize_key-safe).
func RequestToken(commandID string) string { return strings.ReplaceAll(commandID, "-", "") }

func blocked(reason string) wb.Applied {
	return wb.Applied{Outcome: result.Blocked, Reasons: []string{reason}, Detail: reason}
}

// jalali formats an instant as the legacy Jalali Tehran "YYYY/MM/DD HH:MM".
func (c *Connector) jalali(rfc string) (string, error) {
	t, err := time.Parse(time.RFC3339, rfc)
	if err != nil {
		return "", err
	}
	local := t.In(c.tehran)
	d, err := timeutil.ToJalali(local)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%04d/%02d/%02d %02d:%02d", d.Year, d.Month, d.Day, local.Hour(), local.Minute()), nil
}

// fromJalali parses the legacy Jalali Tehran time back to RFC 3339 UTC.
func (c *Connector) fromJalali(s string) (string, error) {
	var y, mo, d, h, mi int
	if _, err := fmt.Sscanf(s, "%d/%d/%d %d:%d", &y, &mo, &d, &h, &mi); err != nil {
		return "", err
	}
	g, err := timeutil.FromJalali(timeutil.JalaliDate{Year: y, Month: mo, Day: d})
	if err != nil {
		return "", err
	}
	return time.Date(g.Year, g.Month, g.Day, h, mi, 0, 0, c.tehran).UTC().Format(time.RFC3339), nil
}

func (c *Connector) keys(ctx context.Context, ids ...string) (map[string]string, error) {
	var want []string
	for _, id := range ids {
		if id != "" {
			want = append(want, id)
		}
	}
	return c.users.UserKeys(ctx, shadow.SourceSystem, want)
}

// leadOp maps a contact outcome intent.
func (c *Connector) leadOp(ctx context.Context, in wb.Intent) (LeadOutcome, string, error) {
	f := in.Fields
	if f["source_system"] != shadow.SourceSystem || f["source_spec"] != shadow.SpecLead || f["source_key"] == "" {
		return LeadOutcome{}, "SUBJECT_NOT_A_LEGACY_LEAD", nil
	}
	label, ok := outcomeLabels[f["outcome"]]
	if !ok {
		return LeadOutcome{}, "OUTCOME_NOT_MAPPED", nil
	}
	op := LeadOutcome{LeadID: f["source_key"], StatusLabel: label, NotPurchaseReason: f["reason"], RequestToken: RequestToken(in.CommandID)}
	if cb := f["callback_at"]; cb != "" {
		j, err := c.jalali(cb)
		if err != nil {
			return LeadOutcome{}, "", err
		}
		op.CallbackJalali = j
	}
	keys, err := c.keys(ctx, f["actor_user_id"])
	if err != nil {
		return LeadOutcome{}, "", err
	}
	if op.ActorLegacyUserID = keys[f["actor_user_id"]]; op.ActorLegacyUserID == "" {
		// The legacy actor must be the real actor; never a service account.
		return LeadOutcome{}, "ACTOR_NOT_MAPPED", nil
	}
	return op, "", nil
}

func (c *Connector) hrCreateOp(ctx context.Context, in wb.Intent) (HRCreate, string, error) {
	f := in.Fields
	keys, err := c.keys(ctx, f["subject_user_id"], f["target_parent_user_id"], f["requested_by_user_id"], f["current_reviewer_user_id"])
	if err != nil {
		return HRCreate{}, "", err
	}
	op := HRCreate{RequestType: f["request_type"], SubjectLegacyUserID: keys[f["subject_user_id"]], RequestedByLegacyUserID: keys[f["requested_by_user_id"]],
		Status: f["status"], Reason: f["reason"], ReconcileKey: in.CommandID}
	if op.SubjectLegacyUserID == "" || op.RequestedByLegacyUserID == "" {
		return HRCreate{}, "USER_NOT_MAPPED", nil
	}
	if t := f["target_parent_user_id"]; t != "" {
		if op.TargetParentLegacyUserID = keys[t]; op.TargetParentLegacyUserID == "" {
			return HRCreate{}, "USER_NOT_MAPPED", nil
		}
	}
	if r := f["current_reviewer_user_id"]; r != "" {
		if op.ReviewerLegacyUserID = keys[r]; op.ReviewerLegacyUserID == "" {
			return HRCreate{}, "USER_NOT_MAPPED", nil
		}
	}
	return op, "", nil
}

func (c *Connector) hrDecideOp(ctx context.Context, in wb.Intent) (HRDecide, string, error) {
	f := in.Fields
	if in.DependsOnLegacyRef == "" {
		return HRDecide{}, "REQUEST_NOT_IN_LEGACY", nil
	}
	keys, err := c.keys(ctx, f["actor_user_id"], f["next_reviewer_user_id"])
	if err != nil {
		return HRDecide{}, "", err
	}
	op := HRDecide{LegacyRequestID: in.DependsOnLegacyRef, Decision: f["decision"], FromStatus: f["from_status"], ToStatus: f["to_status"],
		Notes: f["notes"], ActorLegacyUserID: keys[f["actor_user_id"]], ReconcileKey: in.CommandID}
	if op.ActorLegacyUserID == "" {
		return HRDecide{}, "ACTOR_NOT_MAPPED", nil
	}
	if n := f["next_reviewer_user_id"]; n != "" {
		if op.NextReviewerLegacyUserID = keys[n]; op.NextReviewerLegacyUserID == "" {
			return HRDecide{}, "USER_NOT_MAPPED", nil
		}
	}
	return op, "", nil
}

func applied(r Result) wb.Applied {
	reasons := []string{}
	if r.Reason != "" {
		reasons = append(reasons, r.Reason)
	}
	return wb.Applied{Outcome: r.Outcome, Reasons: reasons, LegacyRef: r.Ref, Detail: r.Reason}
}

// Apply implements the port. A mapping refusal is BLOCKED (nothing was
// sent); a legacy store error is returned as an error (outcome unknown).
func (c *Connector) Apply(ctx context.Context, in wb.Intent) (wb.Applied, error) {
	switch in.CommandType {
	case wb.TypeContactOutcome:
		op, why, err := c.leadOp(ctx, in)
		if err != nil || why != "" {
			return blocked(why), err
		}
		r, err := c.legacy.ApplyLeadOutcome(ctx, op)
		if err != nil {
			return wb.Applied{}, err
		}
		return applied(r), nil
	case wb.TypeHRCreate:
		op, why, err := c.hrCreateOp(ctx, in)
		if err != nil || why != "" {
			return blocked(why), err
		}
		r, err := c.legacy.InsertHRRequest(ctx, op)
		if err != nil {
			return wb.Applied{}, err
		}
		return applied(r), nil
	case wb.TypeHRDecide:
		op, why, err := c.hrDecideOp(ctx, in)
		if err != nil || why != "" {
			return blocked(why), err
		}
		r, err := c.legacy.UpdateHRRequest(ctx, op)
		if err != nil {
			return wb.Applied{}, err
		}
		return applied(r), nil
	}
	return blocked("COMMAND_NOT_MAPPED"), nil
}

func count(n int) wb.Observation {
	switch {
	case n == 0:
		return wb.Absent
	case n == 1:
		return wb.Committed
	}
	return wb.Ambiguous
}

// Observe implements the port: it reads legacy by the command's
// reconciliation key only (never by phone, name or time).
func (c *Connector) Observe(ctx context.Context, in wb.Intent) (wb.Observed, error) {
	switch in.CommandType {
	case wb.TypeContactOutcome:
		f := in.Fields
		if f["source_spec"] != shadow.SpecLead || f["source_key"] == "" {
			return wb.Observed{Observation: wb.Unseen, Detail: "subject is not a legacy lead"}, nil
		}
		o, err := c.legacy.ObserveLeadOutcome(ctx, f["source_key"], RequestToken(in.CommandID))
		if err != nil {
			return wb.Observed{}, err
		}
		obs := wb.Observed{Observation: count(o.Events), LegacyRef: o.EventRef, State: map[string]string{}}
		if obs.Observation != wb.Committed {
			return obs, nil
		}
		for k, v := range outcomeLabels {
			if v == o.StatusLabel {
				obs.State["outcome"] = k
			}
		}
		if obs.State["outcome"] == "" {
			obs.State["outcome"] = "UNMAPPED:" + o.StatusLabel
		}
		if o.NotPurchaseReason != "" {
			obs.State["reason"] = o.NotPurchaseReason
		}
		if o.CallbackJalali != "" {
			cb, err := c.fromJalali(o.CallbackJalali)
			if err != nil {
				cb = "UNPARSEABLE:" + o.CallbackJalali
			}
			obs.State["callback_at"] = cb
		}
		return obs, nil
	case wb.TypeHRCreate, wb.TypeHRDecide:
		var o HRObserved
		var err error
		if in.CommandType == wb.TypeHRCreate {
			o, err = c.legacy.ObserveHRCreate(ctx, in.CommandID)
		} else {
			if in.DependsOnLegacyRef == "" {
				return wb.Observed{Observation: wb.Absent, State: map[string]string{}}, nil
			}
			o, err = c.legacy.ObserveHRDecide(ctx, in.DependsOnLegacyRef, in.CommandID)
		}
		if err != nil {
			return wb.Observed{}, err
		}
		obs := wb.Observed{Observation: count(o.Matches), LegacyRef: o.LegacyRequestID, State: map[string]string{}}
		if obs.Observation != wb.Committed {
			return obs, nil
		}
		ids, err := c.users.ResolveUsers(ctx, shadow.SourceSystem, nonEmpty(o.SubjectLegacyUserID, o.TargetParentLegacyUserID, o.ReviewerLegacyUserID))
		if err != nil {
			return wb.Observed{}, err
		}
		target := func(k string) string {
			if k == "" {
				return ""
			}
			if id, ok := ids[k]; ok {
				return id
			}
			return "UNMAPPED:" + k
		}
		if in.CommandType == wb.TypeHRCreate {
			obs.State = map[string]string{"request_type": o.RequestType, "subject_user_id": target(o.SubjectLegacyUserID), "status": o.Status}
			if t := target(o.TargetParentLegacyUserID); t != "" {
				obs.State["target_parent_user_id"] = t
			}
			if r := target(o.ReviewerLegacyUserID); r != "" {
				obs.State["current_reviewer_user_id"] = r
			}
		} else {
			obs.State = map[string]string{"to_status": o.Status}
			if r := target(o.ReviewerLegacyUserID); r != "" {
				obs.State["next_reviewer_user_id"] = r
			}
		}
		return obs, nil
	}
	return wb.Observed{Observation: wb.Unseen, Detail: "command not mapped"}, nil
}

func nonEmpty(v ...string) []string {
	var out []string
	for _, s := range v {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// RefOf parses a numeric legacy reference.
func RefOf(ref string) (int64, bool) {
	n, err := strconv.ParseInt(ref, 10, 64)
	return n, err == nil && n > 0
}
