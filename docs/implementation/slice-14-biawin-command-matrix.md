# Slice 14 — Biawin writes and fulfillment workflows: command matrix, legacy audit, decisions, ownership and readiness

Status: **controlled implementation of the safe subset. SHADOW activation only, synthetic provider only, zero
financial effect, zero live delivery.**
Baseline: `origin/main` = `1e85fbb10ed8b9e7b14df219da9ac33eb69dc556` (Slice 13 merged; verified, no delta).
Legacy evidence: `plugin/current` (2.0.160), **read only and not modified** (CI `plugin/current unchanged`).

## 1. Recorded conditions

- No merge, no deploy, no Production API call, and no legacy data mutation. No real financial effect is activated.
- No open decision is treated as approved. D-35a–e and D-38 remain open. Every command that depends on them
  is **BLOCKED** in code (`work.BlockedCommands`), in the schema (DB CHECKs) and in the contract (no route).
- **Activation is SHADOW only.** `biawin.activation.rollout_mode` is pinned to `SHADOW_COMPARE` by a CHECK.
  `target_subscription` and `target_item` can only be `SHADOW_ACTIVE`. A CONTROLLED (real) activation request is
  recorded as `BLOCKED` with `D-35`.
- **No Biawin HTTP call from any domain.** The provider package `internal/integration/biawin` is imported only
  by tests. Production wiring has no port at all: crm-api reaches the operation ledger read and human
  reconciliation only. This is enforced by `TestAPIServerHasNoLegacyDependency`, `TestNoProviderInComposedSystem`
  and the CI `go list -deps` gates. The synthetic adapter is **not** a Biawin connector.
- **`charge_wallet` is not a target command (D-38).** A `WALLET_CHARGE` provider operation can only exist as
  `BLOCKED / D-38` (DB CHECK). The port is never called for it. The work-case command is listed as blocked.
- Activation, item creation and fulfillment-case creation are **not one transaction**. biawin commits its
  activation + items + outbox. A relay delivers `biawin.subscription_item_activated`. fulfillment consumes it in
  its own transaction through its inbox (`internal/platform/relay`, fault-injection test).
- Feature flags default **OFF**: `fulfillment.work_read`, `fulfillment.work_write` / `_writeback`, and
  `integration.provider_reconcile`. No migration seeds them (CI grep). The six new capabilities are
  **granted to no role** (CI DB invariant).
- A GET or a UI render never creates, assigns or completes a case. The HTTP e2e test fingerprints every Slice 14
  table before and after all GETs. The UI tests assert GET-only traffic until an explicit confirmation.
- Phone is never identity: no phone/mobile column exists in any Slice 14 table (CI). The legacy provider
  payload's `mobile` is documented as a connector gap, not ported.

## 2. Legacy Biawin / fulfillment write-path audit (2.0.160, file:line)

Columns follow the Phase 2 matrix. `P` = `includes/class-sn-projects.php`, `A` = `includes/class-sn-biavin-api.php`,
`O` = `includes/class-sn-operations-execution.php`.

| # | Command / trigger | Domain | Actor / auth | Preconditions | State changes | Schema / source | External API | Idempotency / concurrency | Financial effects | Unknown-result recovery | History / reversal |
|---|---|---|---|---|---|---|---|---|---|---|---|
| L1 | Activation on payment: `sn_invoice_payment_stage_approved` prio 20 (`P:43`, `P:971`), `sn_invoice_paid` (`P:44`, `P:991`) → `activate_from_invoice` (`P:1002`) / `activate_single_content_from_invoice` (`P:899`) | biawin + fulfillment (mixed) | system hook | invoice fully paid; product is a project/content product | membership + one membership_item **per content product**; suspended membership **reactivated** (`P:1059-1068`); manager routed via `manager_for_sales_branch` (`P:809`) | `sn_project_memberships`, `sn_project_membership_items`; source keys `dot:{case}:{sub}` (`P:1027`), `direct:{inv}:{sub}` (`P:1036`), `product_star:{inv}:{content}` (`P:918`) | none | source_key existence check; **one DB transaction** spanning membership + items (`P:939`/`P:1073` … `P:965`/`P:1109`) | E4 project commission via `credit_biavin_commission` (`P:1181`) when E1 is off | none (ROLLBACK on error) | `customer_phone` copied into the membership (`P:946`, `P:1083`), so phone is stored as data |
| L2 | Reversal: `sn_invoice_financial_rejected` / `_reopened` (`P:45-46`) → `on_invoice_reversed` (`P:1232`) | biawin + commission | system hook | paid ≤ 0.5 or status rejected (`P:1237`) | memberships **suspended** (`P:1241-1242`) | memberships | none | none | **reverses commission wallet debits** (`P:1247-1254`) | none | reversal rows; membership status overwritten |
| L3 | Assign expert `ajax_assign_expert` (`P:2281`), bulk (`P:2294`) | fulfillment | `project_manager` who is the item manager (`P:2284`) | membership active; expert must be a **current HR direct report** (`P:2285-2286`) and `project_expert` (`P:2287`) | `items.expert_user_id`, `assigned_at` (`P:2288`) | items | none | none (last write wins) | none | n/a | `expert_assigned` event |
| L4 | Status update `ajax_update_status` (`P:2350`) | fulfillment | `can_manage_item` (`P:2091`) | — | waiting / no_answer / follow_up / cancelled (reason required) / normal / upsell / purchased / regretted | items | none | none | none | n/a | event |
| L5 | Create action `ajax_create_action` (`P:2406`) | fulfillment + billing | `can_manage_item` | — | upsell / normal action, may create an action invoice | actions | none | none | **financial** (action invoice) | n/a | event |
| L6 | Purchase via API `ajax_api_purchase` (`P:2477`) | integration | `can_manage_item` | action paid | action → `remote_failed` on any error **including a timeout** (`P:2493`) | actions | `create_order` POST `/projects/{code}/orders` (`A:43-53`) | random `X-DCW-Request-ID` per call (`A:73`); **no retry key, no lookup** | none directly | **timeout = failure** (unknown result treated as failed) | event |
| L7 | Wallet charge via API `ajax_api_charge_wallet` (`P:2505`) | integration + finance | `can_manage_item` | — | `remote_failed` on error (`P:2523`) | actions | `charge_wallet` POST `/projects/{code}/wallet/charge` (`A:55-66`) | none | **external wallet charge** | timeout = failure | event |
| L8 | Fulfil action `ajax_fulfill_action` (`P:2559`) | fulfillment | `can_manage_item` | pre-check accepts `ready` **or** `paid` (`P:2564`) | update guarded by `status='paid'` **only** (`P:2565`), yet the event and `sn_project_action_fulfilled` fire anyway (`P:2566-2567`) | actions | none | none | downstream hooks | **bug: event without state change for `ready`** | event (may be false) |
| L9 | Operations assignment `assign_operations` (`O:477`) | fulfillment + commission | executive manager; expert must be a direct report (`O:478`, `direct_experts` `O:417`) | stage executive_manager_queue | case assigned | `sn_project_operations`, execution cases | none | none | **`SN_Purpose_Commission::reconcile_operation` (`O:481`): commission side effect on assignment** | n/a | event |
| L10 | Execution actions `handle_action` (`O:508`): no_answer (`O:515`), wallet_charge (`O:518-532`), wallet_refresh (`O:534-540`), physical_confirm (`O:543`), form_execute / form_skip (`O:545`) | fulfillment + integration | `can_work_case` (`O:500`): expert of the case, or admin | case not terminal (`O:512`) | status / wallet_status changes; success without a code → `sync_pending` (`O:528`) | execution cases | Biawin wallet charge / status query | wallet lock by status set (`O:521`) | **wallet charge** | `sync_pending` + manual refresh | events; **completion decided by the expert alone, no review** |
| L11 | Form submission `save_submission` (`O:586`) | fulfillment | case expert | — | submission row | submissions | none | none | none | n/a | append |

What the audit implies for the target:

- **Ported (adapted):** activation as an owner-local, event-driven decision; one item per catalog component; one
  fulfillment case per item; per-service assignment; operational actions (no answer / follow-up / note);
  evidence; completion with an approved outcome; an external operation ledger with UNKNOWN.
- **Not ported:** HR-hierarchy authority (L3, L9) is replaced by catalog per-service scopes. The single cross-module
  transaction (L1) is not ported. Membership reactivation and suspension (L1, L2) are not ported; they need an
  approved lifecycle (D-35b/c). Commission on activation or assignment (L1, L9) is not ported. Wallet charge
  (L7, L10) is not ported (D-38). Timeout-as-failure (L6, L7) is replaced by write-ahead UNKNOWN. The L8 bug is
  not reproduced: the target event is written in the same transaction as the versioned update. Phone storage
  (L1) is not ported.
- **Legacy-imported cases (Slice 10 read shadow) stay read-only.** No target command touches them. A command on
  them would need a proven write-back path (`UNPROVEN_LEGACY_PATH`).

## 3. Decision readiness (Phase 3)

| Decision | Current authoritative status | Source evidence | Safe default implemented | Commands allowed | Commands blocked | Human Owner approval required | Test / activation implication |
|---|---|---|---|---|---|---|---|
| D-35a cash credit semantics | OPEN | Target Arch §14 "ODR D-35 a–e"; register row D-35 | no cash-credit entitlement exists | none | any `CONSUMES_CREDIT` / `CHARGES_LEDGER` component activation or completion | Product + Finance | activation of a financial component → `BLOCKED / D-35` (test 11/30) |
| D-35b installment credit | OPEN (see D-37) | same | no installment entitlement | none | same | Finance | same |
| D-35c expiry / spend / redemption | OPEN | same | no expiry, no redemption, no suspension or reactivation | none | lifecycle transitions | Product + Finance | no lifecycle command exists |
| D-35d Biawin as redemption system of record | OPEN | same | Biawin is never authoritative; provider results are evidence only | provider ledger (synthetic) + human reconciliation | production activation (`biawin.activate_production`) | Product + Finance | CONTROLLED activation → `BLOCKED / D-35` |
| D-35e conversion mandatory (no-cash path) | OPEN | same | Dot conversion is never substituted for Biawin routing | — | — | Product | branch routing is explicit (`Route`), never inferred |
| D-38 wallet charge semantics / system of record | OPEN | register row D-38; `A:55-66`; `O:518-532` | legacy remains master; no target charge | none | `biawin.wallet_charge`; `WALLET_CHARGE` operation only `BLOCKED` | Finance + Product | DB CHECK + test 29 |
| D-33 operations upgrade-payment authority | OPEN | register row D-33 | not granted | none | upgrade payment by operations | Product + Finance | no route, no capability |
| D-36 component commission allocation | OPEN | register row D-36; Slice 13 | no commission from fulfillment | none | commission on activation / assignment / completion | Finance + Product | test 28/29: no commission write |
| D-41 ledger basis | OPEN | register row D-41 | no ledger posting | none | any posting | Finance | fulfillment writes only its schema |
| OPD-01 identity / lineage | OPEN | X§19 OPD-01 | customer link `UNKNOWN` / `CONFLICT` → activation `BLOCKED`; phone never used | RESOLVED lineage only | activation without proven lineage | Product + MIS | test 12 |
| OPD-02 handoff / ACK authority | OPEN | X§19 OPD-02 | no ACK / SLA / escalation fields; queue owners = catalog scope | assignment | ACK, escalation, SLA | Product | no column, no command |
| OPD-04 refund / proof of execution | OPEN | X§19 OPD-04 | no cancel or refund from fulfillment | — | `fulfillment.cancel_case` | Product + Finance | blocked list |
| OPD-05 transfer / handover | OPEN | X§19 OPD-05 | no reassignment of a started case; scope revocation never rewrites history | reassign while QUEUED / ASSIGNED | `fulfillment.reassign_started` | Product + HR + Sales + Finance | test 9 (history keeps the supervisor snapshot) |
| OPD-06 units / rounding | OPEN | X§19 OPD-06 | no amount in Slice 14 tables | — | — | Finance | CI: no numeric / money column |
| OPD-08 maker / checker | OPEN | X§19 OPD-08 | no review step invented; completion is one guarded executor command | completion with evidence | `fulfillment.submit_review` | Product + Finance | blocked list |
| OPD-09 engine routing | OPEN | X§19 OPD-09 | no commission engine is touched | — | — | Product + Finance | test 29 |
| OPD-10 distribution / assistance paths | OPEN | X§19 OPD-10 | no skip-level or assist grant; only catalog scopes | — | assistance paths | Product + Sales + Finance + HR | no hierarchy-derived permission |
| ADR-01 / F01 | CANDIDATE / P0 open | Target Arch §ADR-01; X F01 | Slice 14 never issues, returns, releases or deletes; archive is not deletion | — | — | Sales + MIS + Finance + Engineering | no invoice / case-return write |

**No design candidate is converted into ratified policy.** The catalog stores what a Human Owner approved
(`approval_ref` is mandatory). It ships with **no** service definitions, so every activation fails closed until
an owner records one.

## 4. Customer lifecycle branches (Phase 4)

The activation consumer takes an explicit `Route` with its basis. Nothing is inferred from product names.

| Branch | Route | Activation decision | Biawin effect |
|---|---|---|---|
| Dissatisfied → After-Sales → Complaint | `AFTER_SALES_COMPLAINT` | `NOT_APPLICABLE` | none (test 14) |
| No additional services → Archive | `ARCHIVE` | `NOT_APPLICABLE` | none; archive is not deletion or release (test 15) |
| Additional services → API / Web Service → Biawin | `ADDITIONAL_SERVICES` | `SHADOW_ACTIVATED` only with `RESOLVED` lineage, approved components and no financial component | SHADOW subscription + one item per component |
| Unknown | `UNKNOWN` | `BLOCKED / ROUTE_UNKNOWN` | none, with reconciliation context recorded |

New and existing customers are both representable (`existing_customer`, test 13). Unknown or conflicting
lineage is recorded as `BLOCKED` with `CUSTOMER_LINEAGE_UNKNOWN` / `CONFLICT`, never guessed (test 12).

## 5. Workflow ownership specification (Phase 5)

| Owner | Owns (Slice 14 tables) | Never owns |
|---|---|---|
| **catalog** (new module, schema `catalog`) | `service_version` (ServiceDefinition + ServiceVersion: evidence requirements, `workflow_key` = `operational_task_v1`, approved completion outcomes, integration provider, financial behaviour), `subscription_component` (product version → ordered service versions), `service_scope_event` (per-service SUPERVISOR / EXECUTOR grants and revocations, each with `approval_ref`) | subscriptions, cases, money |
| **biawin** | `activation` (decision per invoice line), `target_subscription`, `target_item` (one per component), `activation_event` (append-only) | cases, assignment, evidence, ledger |
| **fulfillment** | `work_intake`, `work_case` (one per item; versioned current state), `work_event`, `work_assignment`, `work_evidence` (SHA-256 metadata only), `work_command_record` | subscriptions, ledger, wallet, commission |
| **integration** | `provider_operation` (write-ahead state machine), `provider_operation_event`, `provider_callback` | business state |

- A multi-service subscription is **never one inseparable case**. Each component becomes its own item and its own
  case. Cases are independent (test 6/7: 1, 12 and 31 services).
- No product-specific column on invoices. No generic workflow DSL: one fixed workflow key with a CHECK.
- **Per-service supervision:** a supervisor may assign only cases of services where they currently hold
  SUPERVISOR scope, and only to a user who currently holds EXECUTOR scope for that service. A supervisor role
  grants nothing by itself. No HR hierarchy is read, and no new human role or tier is introduced. A service with
  no supervisor can be assigned by nobody (test 11). Scope changes are events, so history keeps the supervisor
  snapshot at each event (test 9).

## 6. Event-driven activation contract (Phase 6)

```
billing (future producer) ──InvoiceFullyPaid──▶ biawin.activation  [biawin tx: inbox + activation + items + outbox]
                                                     │ outbox: biawin.subscription_activated
                                                     │         biawin.subscription_item_activated (one per item)
                                                     ▼
                                         relay.Drain (claim → deliver → mark; no shared tx)
                                                     ▼
                         fulfillment.work.OnItemActivated  [fulfillment tx: inbox + case + CREATED event + outbox]
```

- Business key `invoice:{invoice_id}|product:{product_key}#{version}` is UNIQUE. A replayed message returns
  the recorded decision (`DUPLICATE_MESSAGE`) and never mints a second subscription (test 4). A replayed item
  event is an inbox no-op (test 5/22).
- A failing fulfillment intake never rolls back biawin. The event stays in biawin's outbox marked failed, and
  the next relay delivers it exactly once (test 23).
- **No production `InvoiceFullyPaid` producer exists in this PR.** billing does not emit it yet, and the shape
  of the payment-complete fact for Subscription+ is part of D-35. The consumer is exercised by tests only.

## 7. Safe fulfillment commands (Phase 7)

| Command | Route | Capability | Actor rule | State rule | Guards |
|---|---|---|---|---|---|
| `fulfillment.assign_case` | `POST /api/v1/fulfillment/work-cases/{case_id}/assignment` | `fulfillment.work.assign` | current SUPERVISOR of the service | QUEUED / ASSIGNED (IN_PROGRESS → `OPD-05`) | assignee must be a current EXECUTOR (`RECIPIENT_INELIGIBLE`) |
| `fulfillment.record_action` | `…/operational-actions` | `fulfillment.work.execute` | the assignee with current EXECUTOR scope | ASSIGNED / IN_PROGRESS | NO_ANSWER / FOLLOW_UP (instant) / NOTE |
| `fulfillment.attach_evidence` | `…/evidence-records` | `fulfillment.work.execute` | same | same | type must be in the definition; duplicate refused; metadata + SHA-256 only (no file storage) |
| `fulfillment.complete_case` | `…/outcome` | `fulfillment.work.execute` | same | same | financial behaviour `NONE` only (else `D-35` / `D-38`); approved outcome; required evidence present (`REQUIRED_EVIDENCE_MISSING`); `NOT_EXECUTED` needs a reason; provider ≠ NONE needs a **CONFIRMED** provider operation (`PROVIDER_CONFIRMATION_REQUIRED`) |
| `integration.resolve_provider_operation` | `POST /api/v1/integration/provider-operations/{operation_id}/resolution` | `integration.provider_operation.reconcile` | proven user id | UNKNOWN / RECONCILIATION_REQUIRED | human evidence note; expected version |

Every command needs an `Idempotency-Key` (same key + same request replays; a changed request is `CONFLICT`)
and `If-Match` with the expected version (stale → `CONFLICT`). The case row is locked `FOR UPDATE`, so
concurrent assignments produce exactly one winner (test 27). Rollout: write flag only → `SHADOW_COMPARE`
(evaluated and recorded, not applied). Write + writeback → `CONTROLLED`, which applies to the target-native SHADOW
case and **never** writes back to legacy.

## 8. BiawinPort and the synthetic adapter (Phase 9)

`providerops.Port` (`Provider / Enabled / Synthetic / Submit / Lookup`) is implemented by:

- `biawin.DisabledPort` — `Enabled() == false`. `Dispatch` returns `ErrPortDisabled` and changes nothing.
- `biawin.SyntheticPort` — deterministic test behaviours `ACK`, `REJECT`, `TIMEOUT_CREATED`, `TIMEOUT_ABSENT` and
  `TIMEOUT_NO_LOOKUP`. **Test-only; not a Biawin connector.**

Biawin native idempotency is **not assumed**. Legacy sends a fresh random `X-DCW-Request-ID` per call and has no
lookup or callback contract (`A:73`). The target therefore uses its own stable reference
(`Reference(business key)`) and a write-ahead `DISPATCHING` state. A timeout or transport error becomes **UNKNOWN**,
never failure and never success. UNKNOWN is never retried blindly (test 17/21).

## 9. External operation reconciliation model

```
PREPARED ─dispatch(write-ahead)─▶ DISPATCHING ─ack─▶ ACKNOWLEDGED ─callback/lookup─▶ CONFIRMED
                                       │ reject ─▶ REJECTED
                                       │ timeout/transport ─▶ UNKNOWN ─lookup present─▶ CONFIRMED
                                       │                            ─lookup absent──▶ CONFIRMED_ABSENT
                                       │                            ─no lookup─────▶ RECONCILIATION_REQUIRED ─human evidence─▶ CONFIRMED / CONFIRMED_ABSENT
WALLET_CHARGE ─▶ BLOCKED (D-38, DB CHECK; never dispatched)
```

Callbacks are HMAC-verified and recorded append-only with a disposition: `DUPLICATE`, `UNMATCHED`,
`ALREADY_FINAL`, `CONFLICT` (conflicting external reference, test 20), `APPLIED`, or `EARLY_APPLIED` (callback
before the dispatch answer, test 19). A late dispatch answer after a callback is recorded only. Every transition
is an append-only `provider_operation_event`.

## 10. Financial boundary (Phase 10)

Biawin and fulfillment never write finance ledger, commission, wallet, billing invoice, payment approval or
customer credit tables. Enforcement:

- DB grants: each owner role holds privileges only on its own schema (CI).
- Lint: `foreign-schema-write`, `biawin-foreign-sql`, `append-only-write` with a single exact exception for
  `UPDATE fulfillment.work_case` from `internal/fulfillment/work`, and `module-boundary`.
- `go list -deps`: the domains reach no finance, billing, payments or commission package.
- Test 28/29: after full flows, the ledger, wallet and commission tables are untouched.

A SubscriptionItem activation or completion never generates a wallet credit or a commission posting.

## 11. Operations UI (Phase 11, Persian RTL)

`web/src/modules/operations`:

- Work queue (`/operations/work`) — scoped cases only.
- Case detail (`/operations/work/:caseId`) — definition, evidence requirements, guarded command forms with an
  explicit confirmation dialog (subject, current → new, effect, expected version), decision-blocked commands
  listed as locked with their decision IDs, evidence, and append-only history.
- Activation decisions (`/operations/activations`) — read only.
- Provider operations (`/operations/provider-operations`) — the connector is shown as `NOT_CONFIGURED`. UNKNOWN is
  shown as "not confirmed, do not resend". There is no dispatch or retry control. Human reconciliation appears
  only with the reconcile capability.

Navigation appears only when the session reports `fulfillment_work`. A SHADOW_COMPARE result is shown as "evaluated,
not applied". UNKNOWN is never shown as success and blocks further submission.

## 12. Explicit list of blocked commands

| Command | Blocked by | Where enforced |
|---|---|---|
| `biawin.activate_production` (real Subscription+ activation) | D-35 | domain (CONTROLLED → BLOCKED), DB CHECK `rollout_mode = 'SHADOW_COMPARE'`, no route |
| Activation of a component with `CONSUMES_CREDIT` / `CHARGES_LEDGER` | D-35 | domain plan, test 11/30 |
| Cash-credit / installment-credit grants | D-35a/b | no command, no table |
| Membership suspension / reactivation / expiry | D-35c | no command |
| `biawin.wallet_charge` (`charge_wallet`) | D-38 | blocked list, DB CHECK on `provider_operation`, no route, lint `biawin-provider-call` |
| `fulfillment.cancel_case` | OPD-04 | blocked list, no route |
| `fulfillment.reopen_case` | OPD-04 / OPD-08 | blocked list, no route |
| `fulfillment.submit_review` | OPD-08 | blocked list, no route |
| `fulfillment.reassign_started` | OPD-05 | domain (IN_PROGRESS → BLOCKED) |
| Upgrade payment by operations | D-33 | no route, no capability |
| Commission on activation / assignment / completion | D-36 / OPD-09 | no code path; lint + `go list -deps` |
| Live provider dispatch from the API | connector gap | no port reachable from crm-api |
| Any command on a legacy-imported (Slice 10) case | unproven write-back path | work cases are target-native only |

## 13. Migrations (Phase 13)

All migrations are additive and owner-scoped (crm-migrate applies 51):

- `catalog/00001_catalog.sql` (rendered infra + tables) and `00002_catalog_append_only_guards.sql`
- `biawin/00003_biawin_activation.sql`
- `fulfillment/00003_fulfillment_work.sql`
- `integration/00011_integration_provider_operations.sql`
- `identity/00011_identity_operations_capabilities.sql` (6 capabilities, no role grant)

No legacy row is imported, migrated or rewritten. No credential or wallet balance is migrated. Slice 10
historical identifiers stay untouched. Work-case provenance carries the item, subscription, activation message and
correlation IDs (test 32).

## 14. Parity and reconciliation evidence — separately labelled

### 14.1 Synthetic parity (this PR)

All figures come from deterministic synthetic fixtures in `internal/integration/biawin/flow_e2e_test.go` and
`ops_http_e2e_test.go` against real PostgreSQL:

| Evidence | Synthetic result |
|---|---|
| Subscription counts | one SHADOW subscription per ADDITIONAL_SERVICES line; 0 for complaint / archive / unknown / blocked |
| Item counts | items = catalog components (1, 12, 31) |
| Open fulfillment cases | cases = items; independent states |
| Service ownership | assignment only by current per-service supervisors to current executors |
| Activation state | SHADOW_ACTIVATED / BLOCKED / NOT_APPLICABLE with reasons; never production |
| Historical outcomes | append-only event history; supervisor snapshot preserved after scope changes |
| External provider references | stable reference per business key; conflicting references → CONFLICT |
| Duplicate operations | duplicate requests and callbacks → same operation / DUPLICATE |
| Unknown results | timeout → UNKNOWN; reconcile before any retry; human resolution with evidence |
| Source coverage gaps | missing definition / retired version / missing components → BLOCKED with reason |

### 14.2 Historical (production-data) parity

**NOT PERFORMED. No historical parity is claimed.** Slice 14 adds no legacy import, and no production snapshot
was available in this environment. Legacy membership / item / operation history remains the Slice 10 read
shadow, with that slice's parity report. Historical activation parity needs a reviewed snapshot and the D-35
lifecycle definition, so it remains open.

## 15. End-to-end coverage (Phase 12)

| # | Scenario | Test |
|---|---|---|
| 1–3 | 1, 12 and 31 services | `TestActivationFanOut` |
| 4 / 5 | activation / item event replayed | `TestActivationFanOut` |
| 6 / 7 | independent ownership; one completed, others open | `TestFulfillmentCommands` |
| 8 | unauthorized supervisor | `TestFulfillmentCommands`, `TestOperationsHTTP` |
| 9 | changed supervisor after assignment | `TestFulfillmentCommands` |
| 10 | missing definition / version | `TestActivationBranchesAndBlocks` |
| 11 | conflicting / absent service-owner policy | `TestFulfillmentCommands`, `TestActivationBranchesAndBlocks` |
| 12 / 13 | lineage unknown; existing customer | `TestActivationBranchesAndBlocks` |
| 14 / 15 | complaint / archive never trigger Biawin | `TestActivationBranchesAndBlocks` |
| 16–21 | duplicate request, timeout after creation, callback twice, early callback, reference conflict, reconcile-before-retry | `TestProviderOperations` |
| 22 / 23 | outbox / inbox replay; cross-module transaction boundary | `TestActivationFanOut`, `TestRelayFaultInjection` |
| 24–27 | unauthorized evidence, missing evidence, stale version, concurrent assignment | `TestFulfillmentCommands`, `TestOperationsHTTP` |
| 28 / 29 | no ledger mutation; no wallet charge or commission posting | `TestFulfillmentCommands`, `TestProviderOperations` |
| 30 | D-35 / D-38 blocked commands | `TestActivationBranchesAndBlocks`, `TestCompletionGuards` |
| 31 | no business write from GET or UI render | `TestOperationsHTTP` (table fingerprint), `operations.test.tsx` |
| 32 | historical source IDs and links preserved | `TestActivationFanOut` |
| 33 | Slices 0–13 regression | full `go test ./...`, web suite, CI |

The e2e suite lives in `internal/integration/biawin` because it is the only tree allowed to use the provider
adapter, and test files are composition roots.

## 16. Production readiness and connector gaps

**Production readiness: NOT READY. Correct by design: nothing is activated.**

| Gap | Status | Needed before activation |
|---|---|---|
| Real Biawin connector | **NOT AVAILABLE.** Only `DisabledPort` + test `SyntheticPort` exist | a provider contract with a stable client reference (idempotency), a lookup by reference, a signed callback contract, credentials via secret store (never migrated) |
| Customer payload | legacy sends name + **mobile** + email (`A:49`, `A:61`) | a privacy-approved minimal payload; phone is never a key |
| Provider idempotency | unproven (random request ID per call) | provider-side evidence; until then UNKNOWN + reconciliation |
| `InvoiceFullyPaid` producer | absent | billing event + D-35 definition of "fully paid" for Subscription+ |
| Relay / reconcile runtime job | absent (tests drive `relay.Drain` / `Reconcile`) | an offline job with ops runbook; no provider call without the connector |
| Catalog administration | recorder exists (`registry.Record`, approval ref required); no admin UI or CLI | Human Owner approvals for real services and scopes |
| Evidence files | metadata + SHA-256 only | storage, retention and PII policy |
| D-35a–e, D-38, D-33, OPD-04/05/08 | OPEN | Human Owner decisions (section 3) |
| HR / Finance runtime | NOT LIVE VERIFIED | real role accounts |
