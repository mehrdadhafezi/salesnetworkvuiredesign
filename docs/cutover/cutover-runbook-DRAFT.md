# Slice 15 — Cutover runbook (DRAFT, NOT APPROVED, NOT FOR EXECUTION)

**Status.** This draft is a proposal. **No step has been executed.** Executing any step needs an
explicit human Go decision recorded in [go-no-go-report.md](go-no-go-report.md) for that exact module and
step.

**Current position.** Every entry condition below is unmet: the overall recommendation is NO-GO.

## 0. Principles

1. **Staged strangler, never big-bang.** Authority moves one module (or one command family) at a time.
2. **One authoritative writer per business operation, always.** A switch is a hand-over:
   - legacy writes for that operation are stopped and verified stopped (including hooks, cron and
     callbacks);
   - in-flight work drains;
   - the target becomes the only writer.

   There is never an overlap window with two writers.
3. **Legacy cannot become read-only for a module until all of its writers are identified and
   controlled.** That includes `wp_ajax_*` handlers, `admin_post`, WP cron, Woo hooks, gateway callbacks,
   Biawin calls, SMS and portal routes. For every module, the writer inventory at 2.0.160 is
   incomplete: 55 tables have no spec, and hook listeners are partly unverified (Slice 12 notes
   `sn_invoice_financial_rejected` listeners as unverified).
4. **Reversibility is configuration only.** Turning a flag OFF stops new target writes. It never
   reverses committed business effects (see the rollback DRAFT).
5. **Reconcile before every step.** This is INV-036: old/new metrics with reasons, inside the agreed
   tolerance, recorded against a named snapshot.

## 1. Proposed sequence

| Phase | Scope | Entry conditions (all must be evidenced) | Authority before → after |
|---|---|---|---|
| **P0 Prerequisites** | E-1…E-7 (decision register §3), P-SEC, P-GOV | Engineering closure evidence; security sign-off | legacy → legacy |
| **P1 Read shadow pilot** | Slice 4–10, 13 reads for named pilot users | P0 complete; real snapshot reconciled (register filled) with discrepancies explained; capability grants approved per pilot user | legacy → legacy (target is read-only, labelled SHADOW) |
| **P2 Non-financial controlled writes** | `cases.nonfinancial`, `organization.hr_request` | P1 stable for an agreed period; write-back connector rehearsed on a legacy staging copy; UNKNOWN reconciliation runbook; D-29/31 values remain blocked | legacy → legacy (target commits and writes back; legacy stays master per Slice 11) |
| **P3 Payments controlled submission** | `payments.manual` (receipt mode) | P2 stable; finaliser and expiry running; object store durability verified; P-FIN-A decided; legacy payment write-back rehearsed; duplicate-payment detection report | legacy → legacy (legacy approves) |
| **P4 Finance authority** | Approve/reject, posting | P-FIN-A, B and C decided; implementation of approve/reject/posting; ledger parity on real data; maker/checker enforced | legacy → **target** for review and posting only after a full drain (see §3) |
| **P5 Sales inventory authority** | Invoice issue, claim, return, allocate, transfer | ADR-01, F01, OPD-03, OPD-10, D-32 decided and implemented; INV-009 race tests on real-shaped data | legacy → target per command family |
| **P6 Commission engine** | Shadow → posting | OPD-09, D-36; historical-window parity | legacy engines → target, with legacy engines disabled first |
| **P7 Biawin / fulfillment** | Activation, work, connector | D-35a–e, D-38; real connector; provider sandbox evidence | legacy → target per service |
| **P8 Legacy read-only per module** | Module by module | Writer inventory complete and controlled; 30-day (to be decided) clean reconciliation | legacy read-only for that module |
| **P9 Retirement** (Slice 16) | Archive | Owner approval; evidence archive verified | — |

## 2. Per-module template (to complete for each module before its switch)

| Field | Content to record |
|---|---|
| Entry conditions | Linked evidence for each gate (decisions, tests, reconciliation run id). |
| Source of record before / after | e.g. "legacy `sn_payments` → target `payments.attempt`". |
| Proposed toggle | e.g. `payments.manual_write=ON`, `payments.manual_writeback=ON`, executed by SQL on the flag store by the named operator; or a routing change in the proxy. |
| Preconditions / dependent modules | e.g. billing read parity, finaliser running, write-back connector healthy. |
| Data snapshot | Snapshot id + cutoff taken immediately before the switch. |
| Reconciliation checkpoint | Cohorts and tolerances; must pass before and after (T+1h, T+24h). |
| Authorized operator role | Named person + backup; never the same person who approves (OPD-08). |
| Monitoring | 5xx, latency, UNKNOWN counts, outbox age, write-back failures, reconciliation deltas. |
| Exit conditions | Metrics inside tolerance for the agreed period; zero unexplained discrepancies. |
| Rollback trigger | Any P0 failure mode in the rollback DRAFT; reconciliation delta beyond tolerance. |
| Rollback steps | Flags OFF (stop new writes) → drain → reconcile → compensations per rollback DRAFT class C/D. |
| Evidence retained | Flag change log, snapshots, reconciliation reports, incident log. |
| Final approval authority | Module owner + Finance (financial modules) + Security (auth) + Human Owner. |

## 3. Authority switch with in-flight work and cross-module transactions

1. **Freeze.** Announce a change window. Stop new legacy writes for the operation: disable the legacy UI
   action, and verify through legacy logs that no hooks or cron fire. A legacy-side switch mechanism
   **does not exist yet**; it needs a legacy change or proxy rule, and approval.
2. **Drain.**
   - Wait until every in-flight legacy request for the operation has committed or failed.
   - Every target outbox for the operation must be empty, and every write-back command terminal
     (APPLIED or REJECTED).
   - Any UNKNOWN must be resolved by reconcile or human resolution.
   - Gateway and provider operations must have no `DISPATCHING`/`UNKNOWN` rows.
3. **Snapshot and reconcile.** Take a snapshot at the cutoff and run the cohort reconciliation. Proceed only
   with zero unexplained deltas.
4. **Switch.** Flip the target flag(s) and the legacy disable together, under one change record.
5. **Verify.** Run a smoke test with real accounts, then reconcile at T+1h and T+24h.
6. **Cross-module operations.**
   - Activation → item → fulfillment case, and payment → finance review → posting, are chained through
     outboxes, never one transaction.
   - During a switch, the upstream module's outbox must be drained before the downstream authority moves.
     The reverse order is forbidden.
   - Partial execution is resolved forward by compensation, never by rewriting history.

## 4. What must never happen

- Two writers for the same operation at the same moment.
- Treating a flag OFF as a rollback of committed effects.
- A switch without a named snapshot and a passing reconciliation.
- A financial switch before P-FIN decisions are recorded on `main`.
