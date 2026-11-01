// Package http adapts the finance SHADOW ledger read service to the
// generated OpenAPI strict server. Handlers guard (flag
// finance.shadow_ledger_read + evaluator + ORGANIZATION scope), call the read
// service and map results. GET only: no SQL, no derivation, no posting and
// no write of any kind (architecture lint). Every answer repeats the SHADOW
// notice and names the one run it came from.
package http

import (
	"context"
	"errors"
	"strconv"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/query"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
)

// Notice is the fixed SHADOW notice of every finance answer.
const Notice = oapi.SHADOWDerivedForArchitectureValidationAndReconciliationOnlyNotTheFinancialSystemOfRecordNotSettledBankConfirmedOrFinanceApprovedMoneyNotAnOfficialOrAccountingBalance

// Handlers serves the finance read operations.
type Handlers struct {
	guard *authorization.Guard
	q     *query.Service
}

// New returns the handlers.
func New(guard *authorization.Guard, q *query.Service) *Handlers {
	return &Handlers{guard: guard, q: q}
}

// FinanceEnabled reports the Slice 9 flag (session navigation hint).
func (h *Handlers) FinanceEnabled(ctx context.Context) bool {
	on, err := h.guard.Enabled(ctx)
	return err == nil && on
}

func corr(ctx context.Context) string { return httpx.CorrelationID(ctx) }

type gate[R any] struct {
	disabled func(o oapi.Outcome, r []oapi.ReasonCode) R
	denied   func(d oapi.DeniedJSONResponse) R
}

// guardOp checks flag + capability and the evaluator scope. Only
// ORGANIZATION is accepted; anything narrower is denied, never broadened
// (no finance row-scope resolver exists).
func guardOp[R any](ctx context.Context, h *Handlers, cp identityapi.Capability, g gate[R]) (authorization.Check, *R, error) {
	c, err := h.guard.Check(ctx, cp)
	if err != nil {
		return c, nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		res := g.disabled(o, r)
		return c, &res, nil
	case authorization.GateDenied:
		res := g.denied(authzhttp.Denied(ctx, c.Decision))
		return c, &res, nil
	}
	if c.Decision.Scope != authorization.ScopeOrganization {
		d := c.Decision
		code := reasoncodes.RowScopeDenied
		d.Allowed, d.Outcome, d.Reason = false, result.Unauthorized, &code
		res := g.denied(authzhttp.Denied(ctx, d))
		return c, &res, nil
	}
	return c, nil, nil
}

func deref[R any](p *R) R {
	var zero R
	if p == nil {
		return zero
	}
	return *p
}

// outcomeOf: the run's technical outcome; a STALE run is PARTIAL with
// STALE_STATE; no run is UNKNOWN (never an empty zero answer).
func outcomeOf(run *query.Run) (oapi.Outcome, []oapi.ReasonCode) {
	if run == nil {
		return authzhttp.Outcome(result.Unknown), authzhttp.Reasons(reasoncodes.SourceUnavailable)
	}
	status := result.Success
	var codes []reasoncodes.Code
	for _, r := range run.Reasons {
		if c, err := reasoncodes.ParseCode(r); err == nil {
			codes = append(codes, c)
		}
	}
	if run.Outcome == "PARTIAL" {
		status = result.Partial
	}
	if run.Freshness == "STALE" {
		status = result.Partial
		codes = append(codes, reasoncodes.StaleState)
	}
	codes, _ = reasoncodes.Normalize(codes)
	return authzhttp.Outcome(status), authzhttp.Reasons(codes...)
}

func page(next *string) *oapi.PageInfo { return &oapi.PageInfo{HasMore: next != nil, NextCursor: next} }

func limitOf(l *oapi.Limit) int {
	if l == nil {
		return 0
	}
	return int(*l)
}

func runIDOf(u *openapi_types.UUID) *string {
	if u == nil {
		return nil
	}
	s := u.String()
	return &s
}

func strOf[T ~string](v *T) *string {
	if v == nil {
		return nil
	}
	s := string(*v)
	return &s
}

func filters(kv map[string]*string) map[string]string {
	out := map[string]string{}
	for k, v := range kv {
		if v != nil {
			out[k] = *v
		}
	}
	return out
}

func money(amount *int64, unit string) *oapi.Money {
	if amount == nil || (unit != "IRT" && unit != "IRR") {
		return nil
	}
	return &oapi.Money{Amount: strconv.FormatInt(*amount, 10), Unit: oapi.Unit(unit)}
}

func shadowAmount(a query.Amount, unit string) oapi.ShadowAmount {
	if a.Value == nil {
		return oapi.ShadowAmount{State: oapi.AmountStateOVERFLOW}
	}
	return oapi.ShadowAmount{Money: money(a.Value, unit), State: oapi.AmountStateREPRESENTABLE}
}

func runOf(r *query.Run) oapi.ShadowRun {
	out := oapi.ShadowRun{RecordKind: oapi.SHADOW, RunId: authzhttp.UUID(r.RunID), RunSeq: r.Seq, DerivationVersion: r.DerivationVersion,
		SourceCutoff: r.SourceCutoff.UTC(), StartedAt: r.Started.UTC(), CompletedAt: r.Completed.UTC(), Outcome: oapi.ShadowRunOutcome(r.Outcome),
		Reasons: oapi.Reasons{}, FactCounts: r.Counts, FactsSha256: r.FactsSHA, Sources: []oapi.FinanceRunSource{},
		Freshness: oapi.FinanceFreshness(r.Freshness)}
	if out.FactCounts == nil {
		out.FactCounts = map[string]int64{}
	}
	for _, b := range r.Bases {
		out.Bases = append(out.Bases, oapi.ShadowBasis(b))
	}
	for _, c := range r.Reasons {
		if code, err := reasoncodes.ParseCode(c); err == nil {
			out.Reasons = append(out.Reasons, oapi.ReasonCode(code))
		}
	}
	for _, s := range r.Sources {
		src := oapi.FinanceRunSource{Source: oapi.FinanceRunSourceSource(s.Source), Status: oapi.FinanceCoverage(s.Status), RowsRead: s.Rows,
			SnapshotId: s.SnapshotID, ReadAt: s.ReadAt.UTC(), Reason: s.Reason, Freshness: oapi.FinanceFreshness(s.Freshness)}
		if s.Cutoff != nil {
			t := s.Cutoff.UTC()
			src.SourceCutoff = &t
		}
		out.Sources = append(out.Sources, src)
	}
	return out
}

func metaOf(r *query.Run, kv map[string]*string) *oapi.FinanceMeta {
	return &oapi.FinanceMeta{RecordKind: oapi.SHADOW, Notice: Notice, Run: runOf(r),
		Query: oapi.FinanceQuery{RunId: authzhttp.UUID(r.RunID), Filters: filters(kv)}}
}

func accountOf(a query.Account) oapi.LedgerAccount {
	out := oapi.LedgerAccount{AccountId: authzhttp.UUID(a.AccountID), AccountKey: a.Key, Category: oapi.LedgerAccountCategory(a.Category),
		NormalSide: oapi.LedgerAccountNormalSide(a.NormalSide), SubjectDomain: a.SubjectDomain, SubjectRef: a.SubjectRef,
		CustomerId: authzhttp.OptUUID(a.CustomerID), HolderKey: a.HolderKey, HolderUserId: authzhttp.OptUUID(a.HolderUserID),
		WalletTypeRaw: a.WalletTypeRaw, Provenance: a.Provenance, Route: a.Route}
	if a.CustomerLink != nil {
		l := oapi.LedgerAccountCustomerLink(*a.CustomerLink)
		out.CustomerLink = &l
	}
	return out
}

func reconOf(r query.Reconciled) oapi.ReconciliationRow {
	out := oapi.ReconciliationRow{SubjectDomain: oapi.ReconciliationRowSubjectDomain(r.SubjectDomain), SubjectRef: r.SubjectRef,
		AccountId: authzhttp.OptUUID(r.AccountID), Basis: oapi.ShadowBasis(r.Basis), Unit: oapi.ReconciliationRowUnit(r.Unit),
		LegacyFigure: oapi.ReconciliationRowLegacyFigure(r.Figure), LegacyRaw: r.LegacyRaw, LegacyAmount: money(r.Legacy, r.Unit),
		LegacyRepr: oapi.ReconciliationRowLegacyRepr(r.LegacyRepr), ShadowAmount: money(r.Shadow, r.Unit), Difference: money(r.Difference, r.Unit),
		State: oapi.ReconciliationState(r.State), Reasons: r.Reasons, Coverage: oapi.FinanceCoverage(r.Coverage), SourceSnapshot: r.Snapshot}
	if out.Reasons == nil {
		out.Reasons = []string{}
	}
	return out
}

func balanceOf(b query.Balance) oapi.AccountBalance {
	out := oapi.AccountBalance{Account: accountOf(b.Account), Basis: oapi.ShadowBasis(b.Basis), Unit: oapi.Unit(b.Unit),
		Debits: shadowAmount(b.Debits, b.Unit), Credits: shadowAmount(b.Credits, b.Unit), Net: shadowAmount(b.Net, b.Unit), Lines: b.Lines}
	if b.Legacy != nil {
		r := reconOf(*b.Legacy)
		out.Legacy = &r
	}
	return out
}

func utc(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}

func entryOf(e query.Entry) oapi.LedgerEntry {
	out := oapi.LedgerEntry{RecordKind: oapi.SHADOW, EntryId: authzhttp.UUID(e.EntryID), EntrySeq: e.Seq, BusinessKey: e.BusinessKey,
		Basis: oapi.ShadowBasis(e.Basis), Unit: oapi.Unit(e.Unit), EffectKind: e.EffectKind, Provenance: e.Provenance,
		FinanceReview: oapi.NOTFINANCEREVIEWED, SourceRefs: []oapi.SourceRef{}, EffectiveAt: utc(e.EffectiveAt), PostedAt: utc(e.PostedAt),
		SourceTimeRaw: e.SourceTimeRaw, TimeEvidence: oapi.LedgerEntryTimeEvidence(e.TimeEvidence),
		ReversesEntryId: authzhttp.OptUUID(e.ReversesEntryID), ReversedByEntryId: authzhttp.OptUUID(e.ReversedBy),
		ReconciliationStatus: oapi.LedgerEntryReconciliationStatus(e.Status), Lines: []oapi.LedgerLine{}}
	for _, r := range e.SourceRefs {
		out.SourceRefs = append(out.SourceRefs, oapi.SourceRef{Domain: r.Domain, Ref: r.Ref, Role: r.Role})
	}
	for _, l := range e.Lines {
		amt := l.Amount
		out.Lines = append(out.Lines, oapi.LedgerLine{LineNo: l.No, AccountId: authzhttp.UUID(l.AccountID), AccountKey: l.AccountKey,
			Category: l.Category, Side: oapi.LedgerLineSide(l.Side), Amount: *money(&amt, l.Unit), SourceRef: l.SourceRef})
	}
	if m := money(e.Debits.Value, e.Unit); m != nil {
		out.Debits = *m
	}
	if m := money(e.Credits.Value, e.Unit); m != nil {
		out.Credits = *m
	}
	return out
}

func counts(cs []query.Count) []oapi.GroupCount {
	out := []oapi.GroupCount{}
	for _, c := range cs {
		g := oapi.GroupCount{Count: c.N}
		for _, kv := range []struct {
			v   string
			dst **string
		}{{c.Basis, &g.Basis}, {c.Unit, &g.Unit}, {c.State, &g.State}, {c.Reason, &g.Reason}} {
			if kv.v != "" {
				v := kv.v
				*kv.dst = &v
			}
		}
		out = append(out, g)
	}
	return out
}

func isInvalid(err error) bool  { return errors.Is(err, query.ErrInvalid) }
func isNoRun(err error) bool    { return errors.Is(err, query.ErrNoRun) }
func isNotFound(err error) bool { return errors.Is(err, query.ErrNotFound) }

// ---- shadow runs -------------------------------------------------------------

func (h *Handlers) ListFinanceShadowRuns(ctx context.Context, req oapi.ListFinanceShadowRunsRequestObject) (oapi.ListFinanceShadowRunsResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapFinanceShadowRead, gate[oapi.ListFinanceShadowRunsResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.ListFinanceShadowRunsResponseObject {
			return oapi.ListFinanceShadowRuns200JSONResponse{Body: oapi.ShadowRunListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.ListFinanceShadowRunsResponseObject {
			return oapi.ListFinanceShadowRuns403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	runs, next, err := h.q.Runs(ctx, query.Page{Cursor: req.Params.Cursor, Limit: limitOf(req.Params.Limit)})
	if isInvalid(err) {
		return oapi.ListFinanceShadowRuns400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	}
	if err != nil {
		return nil, err
	}
	items := []oapi.ShadowRun{}
	for i := range runs {
		items = append(items, runOf(&runs[i]))
	}
	o, r := authzhttp.Outcome(result.Success), authzhttp.Reasons()
	if len(runs) == 0 {
		o, r = outcomeOf(nil)
	}
	return oapi.ListFinanceShadowRuns200JSONResponse{Body: oapi.ShadowRunListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice,
		Items: &items, Page: page(next), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

func (h *Handlers) GetFinanceShadowRun(ctx context.Context, req oapi.GetFinanceShadowRunRequestObject) (oapi.GetFinanceShadowRunResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapFinanceShadowRead, gate[oapi.GetFinanceShadowRunResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.GetFinanceShadowRunResponseObject {
			return oapi.GetFinanceShadowRun200JSONResponse{Body: oapi.ShadowRunEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.GetFinanceShadowRunResponseObject {
			return oapi.GetFinanceShadowRun403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	id := req.RunId.String()
	d, err := h.q.RunByID(ctx, &id)
	if isNoRun(err) {
		return oapi.GetFinanceShadowRun404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	}
	if isInvalid(err) {
		return oapi.GetFinanceShadowRun400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	}
	if err != nil {
		return nil, err
	}
	o, r := outcomeOf(&d.Run)
	run := runOf(&d.Run)
	rc, fc := counts(d.Reconciliation), counts(d.Findings)
	return oapi.GetFinanceShadowRun200JSONResponse{Body: oapi.ShadowRunEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice,
		Run: &run, ReconciliationSummary: &rc, FindingSummary: &fc, AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// ---- accounts ------------------------------------------------------------------

func (h *Handlers) ListFinanceAccounts(ctx context.Context, req oapi.ListFinanceAccountsRequestObject) (oapi.ListFinanceAccountsResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapFinanceShadowRead, gate[oapi.ListFinanceAccountsResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.ListFinanceAccountsResponseObject {
			return oapi.ListFinanceAccounts200JSONResponse{Body: oapi.AccountListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.ListFinanceAccountsResponseObject {
			return oapi.ListFinanceAccounts403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	p := req.Params
	cat := strOf(p.AccountCategory)
	run, accts, byCat, next, err := h.q.Accounts(ctx, query.AccountFilter{RunID: runIDOf(p.RunId), Category: cat},
		query.Page{Cursor: p.Cursor, Limit: limitOf(p.Limit)})
	switch {
	case isInvalid(err):
		return oapi.ListFinanceAccounts400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case isNoRun(err):
		o, r := outcomeOf(nil)
		return oapi.ListFinanceAccounts200JSONResponse{Body: oapi.AccountListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx)}}, nil
	case err != nil:
		return nil, err
	}
	items := []oapi.LedgerAccount{}
	for _, a := range accts {
		items = append(items, accountOf(a))
	}
	sum := []oapi.GroupCount{}
	for cat, n := range byCat {
		v := cat
		sum = append(sum, oapi.GroupCount{State: &v, Count: n})
	}
	o, r := outcomeOf(&run)
	return oapi.ListFinanceAccounts200JSONResponse{Body: oapi.AccountListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx),
		Meta: metaOf(&run, map[string]*string{"account_category": cat}), Summary: &sum, Items: &items, Page: page(next),
		AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

func (h *Handlers) GetFinanceAccount(ctx context.Context, req oapi.GetFinanceAccountRequestObject) (oapi.GetFinanceAccountResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapFinanceShadowRead, gate[oapi.GetFinanceAccountResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.GetFinanceAccountResponseObject {
			return oapi.GetFinanceAccount200JSONResponse{Body: oapi.AccountEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.GetFinanceAccountResponseObject {
			return oapi.GetFinanceAccount403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	run, d, err := h.q.AccountByID(ctx, runIDOf(req.Params.RunId), req.AccountId.String())
	switch {
	case isInvalid(err):
		return oapi.GetFinanceAccount400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case isNoRun(err), isNotFound(err):
		return oapi.GetFinanceAccount404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case err != nil:
		return nil, err
	}
	acct := accountOf(d.Account)
	bals := []oapi.AccountBalance{}
	for _, b := range d.Balances {
		bals = append(bals, balanceOf(b))
	}
	o, r := outcomeOf(&run)
	return oapi.GetFinanceAccount200JSONResponse{Body: oapi.AccountEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx),
		Meta: metaOf(&run, nil), Account: &acct, Balances: &bals, AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// ---- ledger --------------------------------------------------------------------

func (h *Handlers) ListFinanceLedger(ctx context.Context, req oapi.ListFinanceLedgerRequestObject) (oapi.ListFinanceLedgerResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapFinanceLedgerRead, gate[oapi.ListFinanceLedgerResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.ListFinanceLedgerResponseObject {
			return oapi.ListFinanceLedger200JSONResponse{Body: oapi.LedgerListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.ListFinanceLedgerResponseObject {
			return oapi.ListFinanceLedger403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	p := req.Params
	f := query.LedgerFilter{RunID: runIDOf(p.RunId), Basis: strOf(p.Basis), Unit: strOf(p.Unit), Status: strOf(p.State),
		SourceType: strOf(p.SourceType), AccountCategory: strOf(p.AccountCategory), EffectiveFrom: p.EffectiveFrom, EffectiveTo: p.EffectiveTo}
	run, es, sum, next, err := h.q.Ledger(ctx, f, query.Page{Cursor: p.Cursor, Limit: limitOf(p.Limit)})
	switch {
	case isInvalid(err):
		return oapi.ListFinanceLedger400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case isNoRun(err):
		o, r := outcomeOf(nil)
		return oapi.ListFinanceLedger200JSONResponse{Body: oapi.LedgerListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx)}}, nil
	case err != nil:
		return nil, err
	}
	items := []oapi.LedgerEntry{}
	for _, e := range es {
		items = append(items, entryOf(e))
	}
	groups := []oapi.LedgerGroup{}
	for _, g := range sum.Rows {
		groups = append(groups, oapi.LedgerGroup{Basis: oapi.ShadowBasis(g.Basis), Unit: oapi.Unit(g.Unit), Entries: g.Entries,
			Debits: shadowAmount(g.Debits, g.Unit), UnknownEffectiveAt: g.UnknownEffectiveAt})
	}
	kv := map[string]*string{"basis": f.Basis, "unit": f.Unit, "state": f.Status, "source_type": f.SourceType, "account_category": f.AccountCategory}
	if f.EffectiveFrom != nil {
		v := f.EffectiveFrom.UTC().Format(time.RFC3339)
		kv["effective_from"] = &v
	}
	if f.EffectiveTo != nil {
		v := f.EffectiveTo.UTC().Format(time.RFC3339)
		kv["effective_to"] = &v
	}
	o, r := outcomeOf(&run)
	return oapi.ListFinanceLedger200JSONResponse{Body: oapi.LedgerListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx),
		Meta: metaOf(&run, kv), Summary: &groups, Items: &items, Page: page(next), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

func (h *Handlers) GetFinanceLedgerEntry(ctx context.Context, req oapi.GetFinanceLedgerEntryRequestObject) (oapi.GetFinanceLedgerEntryResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapFinanceLedgerRead, gate[oapi.GetFinanceLedgerEntryResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.GetFinanceLedgerEntryResponseObject {
			return oapi.GetFinanceLedgerEntry200JSONResponse{Body: oapi.LedgerEntryEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.GetFinanceLedgerEntryResponseObject {
			return oapi.GetFinanceLedgerEntry403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	run, e, err := h.q.EntryByID(ctx, runIDOf(req.Params.RunId), req.EntryId.String())
	switch {
	case isInvalid(err):
		return oapi.GetFinanceLedgerEntry400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case isNoRun(err), isNotFound(err):
		return oapi.GetFinanceLedgerEntry404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case err != nil:
		return nil, err
	}
	entry := entryOf(e)
	o, r := outcomeOf(&run)
	return oapi.GetFinanceLedgerEntry200JSONResponse{Body: oapi.LedgerEntryEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx),
		Meta: metaOf(&run, nil), Entry: &entry, AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// ---- balances / reconciliation ---------------------------------------------------

func (h *Handlers) ListFinanceBalances(ctx context.Context, req oapi.ListFinanceBalancesRequestObject) (oapi.ListFinanceBalancesResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapFinanceShadowRead, gate[oapi.ListFinanceBalancesResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.ListFinanceBalancesResponseObject {
			return oapi.ListFinanceBalances200JSONResponse{Body: oapi.BalanceListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.ListFinanceBalancesResponseObject {
			return oapi.ListFinanceBalances403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	p := req.Params
	f := query.BalanceFilter{RunID: runIDOf(p.RunId), Basis: strOf(p.Basis), Unit: strOf(p.Unit), Category: strOf(p.AccountCategory), State: strOf(p.State)}
	run, bals, sum, next, err := h.q.Balances(ctx, f, query.Page{Cursor: p.Cursor, Limit: limitOf(p.Limit)})
	switch {
	case isInvalid(err):
		return oapi.ListFinanceBalances400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case isNoRun(err):
		o, r := outcomeOf(nil)
		return oapi.ListFinanceBalances200JSONResponse{Body: oapi.BalanceListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx)}}, nil
	case err != nil:
		return nil, err
	}
	items := []oapi.AccountBalance{}
	for _, b := range bals {
		items = append(items, balanceOf(b))
	}
	gc := counts(sum.Rows)
	o, r := outcomeOf(&run)
	return oapi.ListFinanceBalances200JSONResponse{Body: oapi.BalanceListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx),
		Meta:    metaOf(&run, map[string]*string{"basis": f.Basis, "unit": f.Unit, "account_category": f.Category, "state": f.State}),
		Summary: &gc, Items: &items, Page: page(next), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

func (h *Handlers) ListFinanceReconciliation(ctx context.Context, req oapi.ListFinanceReconciliationRequestObject) (oapi.ListFinanceReconciliationResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapFinanceReconciliationRead, gate[oapi.ListFinanceReconciliationResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.ListFinanceReconciliationResponseObject {
			return oapi.ListFinanceReconciliation200JSONResponse{Body: oapi.ReconciliationListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.ListFinanceReconciliationResponseObject {
			return oapi.ListFinanceReconciliation403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	p := req.Params
	f := query.ReconFilter{RunID: runIDOf(p.RunId), Basis: strOf(p.Basis), Unit: strOf(p.Unit), State: strOf(p.State), SubjectDomain: strOf(p.SubjectType)}
	run, rows, sum, next, err := h.q.Reconciliation(ctx, f, query.Page{Cursor: p.Cursor, Limit: limitOf(p.Limit)})
	switch {
	case isInvalid(err):
		return oapi.ListFinanceReconciliation400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case isNoRun(err):
		o, r := outcomeOf(nil)
		return oapi.ListFinanceReconciliation200JSONResponse{Body: oapi.ReconciliationListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx)}}, nil
	case err != nil:
		return nil, err
	}
	items := []oapi.ReconciliationRow{}
	for _, x := range rows {
		items = append(items, reconOf(x))
	}
	gc := counts(sum)
	o, r := outcomeOf(&run)
	return oapi.ListFinanceReconciliation200JSONResponse{Body: oapi.ReconciliationListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx),
		Meta:    metaOf(&run, map[string]*string{"basis": f.Basis, "unit": f.Unit, "state": f.State, "subject_type": f.SubjectDomain}),
		Summary: &gc, Items: &items, Page: page(next), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}
