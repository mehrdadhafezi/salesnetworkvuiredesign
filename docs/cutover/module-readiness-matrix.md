# Slice 15 — Module readiness matrix

Baseline: `origin/main` = `b3bf31b5979edfa1ba9a5fb6a45521d21e7712c4` (PR #39 merge; tree identical to
`26f92b1`, whose three CI jobs passed on PR #39). Legacy behavioural baseline: `plugin/current` 2.0.160,
unchanged since `86859d7`.

**Evidence grades:**
- **VERIFIED**: observed in code, tests, CI or a committed artefact at the baseline.
- **REPORTED**: stated by an earlier implementation record but not reproduced here.
- **UNKNOWN**: no adequate evidence.

**Readiness classes** follow [go-no-go-report.md](go-no-go-report.md): GO, CONDITIONAL GO, NO-GO,
NOT VERIFIED, NOT APPLICABLE.

**Universal facts.** These apply to every module and are not repeated in each row. All are VERIFIED.

| ID | Fact | Evidence |
|---|---|---|
| U1 | No production authentication exists. Outside `development`/`test`, every request resolves to `authorization.Anonymous`, so every guarded read and command answers UNAUTHORIZED. The dev stub refuses to start in production. | `cmd/crm-api/writes.go:72-76`, `internal/authorization/guard.go:19-22`, `internal/platform/authstub/authstub.go:35,46`, `internal/platform/apiserver/config.go:79` |
| U2 | Every feature flag is absent, and therefore OFF, on a fresh database. No migration seeds one. An absent flag reads as disabled. | `internal/platform/featureflags/featureflags.go:108-115`; `scripts/cutover/readiness-evidence.sh static` |
| U3 | The only role→capability grants are six HR read grants. Every capability added in Slices 5–14 is granted to no role. | `internal/identity/migrations/sql/00001_identity.sql:137-143`; CI DB invariants per slice |
| U4 | No production data has been staged, imported or reconciled. The only staged data are synthetic fixtures. | `cmd/crm-legacy-import/main.go:10` ("stage a SYNTHETIC package (others are BLOCKED)") |
| U5 | No deployment, container, infrastructure, backup or monitoring definition exists in the repository. | repository tree; [infrastructure-operations-checklist.md](infrastructure-operations-checklist.md) |
| U6 | No offline command runs these background steps: receipt finaliser, unbound-upload expiry, outbox relay, legacy write-back worker, provider-operation dispatch/reconcile. | `internal/payments/command/finalise.go:43,321`, `internal/integration/writeback/worker.go:273`, `internal/platform/relay/relay.go:30`, `internal/integration/providerops/providerops.go:307,394`; `readiness-evidence.sh static` |
| U7 | No production legacy write-back connector exists. Only the `Legacy` interface and a synthetic fixture exist. | `internal/integration/legacy/writeback/connector.go:13-16` |
| U8 | Legacy 2.0.160 is the system of record for **every** module. No module has ever been switched. | architecture §30–31; no switch mechanism exists |

## Matrix

For readability, the 15 required fields per module are grouped into three blocks. "Legacy SoR" means the
current system of record.

### 1. Identity and authentication

| Field | Value |
|---|---|
| Legacy SoR | WordPress users/usermeta and legacy roles (`wp_user`, `wp_usermeta`). |
| Target implementation | `internal/identity`: shadow user accounts, role grants, capability catalogue. No credentials, no sessions (CI asserts there is no `credential`/`session` table). |
| Read paths | `GET /api/v1/identity/users[/{id}]`, `/authorization/effective-access`, `/session`. |
| Write paths | none. |
| Blocked commands | login, password set/reset (D-34), session issue, grant/revoke capability (D-42). |
| Flags (default) | `organization.hr_read_shadow` (OFF). |
| Permission assignments | U3. |
| External dependencies | WordPress auth remains the only working login. |
| Workers | none. |
| Historical coverage | `legacy.wp_user.v1` and `legacy.wp_usermeta.v1` specs exist. No real import. |
| Reconciliation | `reconcile-identity-org` exists (Slice 4). Run on synthetic data only. |
| Runtime evidence | none. |
| Rollback | not applicable (nothing live). |
| **Classification** | **NO-GO**: no production authentication (U1). |
| Blockers / closure evidence | An approved auth design (identity provider, session, cookie, CSRF, MFA), D-34, D-42, a real-account login acceptance test, and a security review. |

### 2. Organization, HR and authorization

| Field | Value |
|---|---|
| Legacy SoR | `sn_hr_*` tables (`hr_profiles`, `assignments`, `assignment_history`, positions, units…). |
| Target implementation | `internal/organization` (persons, employment, units, positions, assignments, hierarchy findings, HR change requests) and `internal/authorization` (route→module→action→row evaluator). |
| Read paths | 7 organization GETs and 2 HR request GETs. |
| Write paths | `POST /hr/requests`, `POST /hr/requests/{id}/decisions` (Slice 11, family `organization.hr_request`). Final apply is not implemented. |
| Blocked commands | HR final apply (`assign_parent`, termination), compensation, credentials, cross-scope transfer (SD-04), termination handover (OPD-05). |
| Flags (default) | `organization.hr_read_shadow`, `organization.hr_request_write`, `organization.hr_request_writeback` (all OFF). |
| Permission assignments | HR has 6 read capabilities (U3). `org.hr_request.*` is granted to no role. |
| External dependencies | Legacy write-back connector (U7). |
| Workers | Write-back worker (U6). |
| Historical coverage | 9 HR specs. `sn_hr_change_requests`, `sn_hr_compensation_*` and `sn_hr_profile_logs` have no spec. |
| Reconciliation | Capability diff and hierarchy conflicts on synthetic data only. |
| Runtime evidence | none. HR runtime is NOT LIVE VERIFIED (E0). |
| Rollback | Flag OFF stops new requests. Committed target requests remain (C). |
| **Classification** | **NO-GO**: U1, U6, U7, OPD-05 (P0 impact), HR runtime unverified. |
| Blockers / closure evidence | OPD-05 decision, SD-04, a production write-back connector, a real HR account acceptance test, and a real-snapshot reconciliation. |

### 3. Customers and identity lineage

| Field | Value |
|---|---|
| Legacy SoR | No customer table in legacy. Customers are implicit in invoice issued-to and lead phone/name. |
| Target implementation | `internal/customers`: customers, contact points, evidence, MergeCase candidates. RESOLVED/RELATED/CONFLICT only. |
| Read paths | 4 customers GETs and 3 customer-identity GETs. |
| Write paths | none. |
| Blocked commands | Merge (D-30), phone dedupe (D-29/D-31), create customer. |
| Flags (default) | `customers.identity_shadow_read` (OFF). |
| Permission assignments | none granted. |
| External dependencies | none. |
| Workers | none. |
| Historical coverage | `legacy.invoice_issued_to.v1` plus leads. Dot and campaign customers (`sn_dot_*`, `sn_campaign_*`) have no spec. |
| Reconciliation | Resolution report on synthetic data. Zero phone-only merges is asserted by tests. |
| Runtime evidence | none. |
| Rollback | not applicable. |
| **Classification** | **NO-GO** for authority (D-30, OPD-01, no real data). Shadow-read pilot: **NOT VERIFIED**. |
| Blockers / closure evidence | A real-snapshot resolution report with ambiguity counts, OPD-01, D-30. |

### 4. Sales and case management

| Field | Value |
|---|---|
| Legacy SoR | `sn_leads`, `sn_lead_status_history`, seller flow, Dot cases. |
| Target implementation | `internal/cases`: cases, attribution snapshots, contact outcomes. |
| Read paths | 5 cases GETs, sales coverage/parity reports, case/report GETs. |
| Write paths | `POST /cases/{id}/contact-outcomes` (family `cases.nonfinancial`). |
| Blocked commands | MarkReady, CreateCase, LinkAlias, ClaimCaseForInvoice, IssueInvoice (ADR-01); DUPLICATE and PRE_INVOICE outcome values. |
| Flags (default) | `sales.read_shadow`, `cases.nonfinancial_write`, `cases.nonfinancial_writeback` (OFF). |
| Permission assignments | none granted. |
| External dependencies | Legacy write-back (U7). |
| Workers | Write-back worker (U6). |
| Historical coverage | leads and status history. `sn_seller_flow_*`, `sn_dot_*`, `sn_lead_statuses` and `sn_activity_logs` have no spec. |
| Reconciliation | Custody parity on synthetic data. |
| Runtime evidence | none. |
| Rollback | Flag OFF. Committed target outcomes not yet written back stay UNKNOWN (D). |
| **Classification** | **NO-GO**: ADR-01/F01, U6/U7, Dot cases not covered. |
| Blockers / closure evidence | ADR-01 acceptance, an F01 union-link proof, a production connector, a real-snapshot custody parity. |

### 5. Custody and distribution

| Field | Value |
|---|---|
| Legacy SoR | `sn_distribution_items/logs`, `sn_leads.status`, distribution chains, MIS distribution. |
| Target implementation | `internal/custody`: append-only custody events. |
| Read paths | `GET /custody/assignments`, `/custody/cases/{id}`, custody reports. |
| Write paths | none. |
| Blocked commands | Allocate, Transfer, Return, Release, Reset (ADR-01, F01, OPD-03, OPD-10). |
| Flags (default) | `sales.read_shadow` (OFF). |
| Permission assignments | none granted. |
| External dependencies | none. |
| Workers | none. |
| Historical coverage | distribution items/logs. `sn_distribution_chains/steps`, `sn_mis_*` and `sn_extra_number_requests` have no spec. |
| Reconciliation | Synthetic only. |
| Runtime evidence | none. |
| Rollback | not applicable. |
| **Classification** | **NO-GO**: F01 (P0), ADR-01, OPD-03, OPD-10. |
| Blockers / closure evidence | Those decisions, plus MIS distribution source coverage. |

### 6. Catalog

| Field | Value |
|---|---|
| Legacy SoR | WooCommerce products, `sn_project_subscription_contents`, project manager/operations routes. |
| Target implementation | `internal/catalog`: immutable service versions, components, per-service scope events. Ships **empty**. |
| Read paths | Indirect, through work cases. |
| Write paths | `registry.Record` (Go API only; no route, no CLI). |
| Blocked commands | Any record without a Human Owner `approval_ref`. |
| Flags (default) | none (read through `fulfillment.work_read`). |
| Permission assignments | n/a. |
| External dependencies | WooCommerce (not integrated). |
| Workers | none. |
| Historical coverage | No spec for products or `sn_project_*` configuration tables. |
| Reconciliation | none. |
| Runtime evidence | none. |
| Rollback | not applicable. |
| **Classification** | **NOT VERIFIED** (no approved service definitions, no admin path). |
| Blockers / closure evidence | Owner-approved service definitions and scopes, and an administration procedure (CLI or UI) with an audit trail. |

### 7. Billing and invoices

| Field | Value |
|---|---|
| Legacy SoR | `sn_invoices`, `sn_invoice_items`, `sn_invoice_payment_stages`, WooCommerce orders. |
| Target implementation | `internal/billing` (read model; lifecycle UNKNOWN). |
| Read paths | 7 invoice GETs and the billing reconciliation report. |
| Write paths | none. |
| Blocked commands | IssueInvoice, IssueNextStage (D-32, lifecycle UNKNOWN), invoice rejection transition. |
| Flags (default) | `billing_payments.read_shadow` (OFF). |
| Permission assignments | none granted. |
| External dependencies | WooCommerce order coupling (not integrated). |
| Workers | none. |
| Historical coverage | invoice v2, items, stages, payments. `sn_invoice_wheel`, `sn_report_invoice_state_events` and Woo orders have no spec. |
| Reconciliation | `reconcile-billing` on synthetic data. |
| Runtime evidence | none. |
| Rollback | not applicable. |
| **Classification** | **NO-GO**: ADR-01, D-32, no stage state machine. |
| Blockers / closure evidence | Billing lifecycle contract, ADR-01, D-28, D-32, a real-snapshot per-invoice totals report. |

### 8. Payments and evidence

| Field | Value |
|---|---|
| Legacy SoR | `sn_payments`, receipt files, gateway (Zarinpal) callback. |
| Target implementation | `internal/payments`: upload staging, receipt-mode batch, finaliser, expiry, synthetic gateway flow (no route). |
| Read paths | 7 payments GETs. |
| Write paths | `POST /payments/uploads`, `POST /payments/invoices/{id}/batches` (family `payments.manual`; CONTROLLED only). |
| Blocked commands | Info-mode submit (D-43), assist submit (OPD-10/08), self-service (D-26), gateway routes (D-25), refunds (OPD-04). |
| Flags (default) | `payments.manual_write`, `payments.manual_writeback` (OFF). |
| Permission assignments | none granted. |
| External dependencies | Object store (`CRM_PAYMENTS_OBJECT_ROOT`, filesystem implementation), legacy write-back (U7), gateway (none). |
| Workers | Finaliser and expiry (U6). |
| Historical coverage | `legacy.payment.v1`. Receipt **files** are not extracted (links only). `sn_payment_reward_claims` has no spec. |
| Reconciliation | Synthetic only. |
| Runtime evidence | none. |
| Rollback | Flag OFF stops new uploads. Stored evidence and committed attempts remain (C). |
| **Classification** | **NO-GO**: U6 (receipts never finalise), U7, D-43/D-44/OPD-06, durable storage unverified. |
| Blockers / closure evidence | Worker process, production object store with verified durability, connector, decisions. |

### 9. Finance and ledger

| Field | Value |
|---|---|
| Legacy SoR | Legacy wallets (`sn_wallets`, `sn_wallet_transactions`); approvals in `sn_payments`. |
| Target implementation | `internal/finance`: SHADOW ledger derived offline (`crm-finance-shadow`), review queue (read), `review.Evaluate` gate. |
| Read paths | 10 finance GETs. |
| Write paths | none. |
| Blocked commands | Approve, reject, bulk approve, posting, reversal, settlement, refund (D-41, OPD-04/06/08/09, D-43). |
| Flags (default) | `finance.shadow_ledger_read`, `finance.review_read` (OFF). |
| Permission assignments | none granted. |
| External dependencies | none. |
| Workers | `crm-finance-shadow` (offline, manual). |
| Historical coverage | wallets and wallet transactions. |
| Reconciliation | Derived-vs-legacy balance diff on synthetic data. |
| Runtime evidence | none. Finance runtime is NOT LIVE VERIFIED. |
| Rollback | not applicable. |
| **Classification** | **NO-GO**: no execution path; critical decisions open. |
| Blockers / closure evidence | See [financial-readiness.md](financial-readiness.md). |

### 10. Commission

| Field | Value |
|---|---|
| Legacy SoR | Four legacy engines (purpose matrix, rules auto-credit, dry-run, project card). |
| Target implementation | `internal/commission`: SHADOW runs, parity, policy versions (never approved). |
| Read paths | 7 commission GETs. |
| Write paths | none. |
| Blocked commands | Approve, post, credit, settle, reverse, recalculate (OPD-08/09, D-36). |
| Flags (default) | `commission.read_shadow` (OFF). |
| Permission assignments | none granted. |
| External dependencies | none. |
| Workers | `crm-commission-shadow` (offline). |
| Historical coverage | 5 commission specs. `sn_project_commissions` and `sn_project_commission_rules` (engine E4) have no spec. |
| Reconciliation | Synthetic parity only. Engine routing for past windows is UNKNOWN. |
| Runtime evidence | none. |
| Rollback | not applicable. |
| **Classification** | **NO-GO** (OPD-09, OPD-08, D-36). |
| Blockers / closure evidence | Engine routing decision, historical-window parity on real data. |

### 11. Reporting and MIS

| Field | Value |
|---|---|
| Legacy SoR | Legacy report pages, Invoice Review, MIS data rows. |
| Target implementation | `internal/reporting`: projections (offline `crm-reporting`), metric contracts, audited Invoice Review query. |
| Read paths | 13 GETs (reports, MIS, reporting, invoice-review). |
| Write paths | none (export blocked, D-23). |
| Blocked commands | Export (D-23); Invoice Review UI (D-24). |
| Flags (default) | `reporting.read_shadow` (OFF). |
| Permission assignments | none granted. |
| External dependencies | Projection rebuild job. |
| Workers | `crm-reporting` (offline, manual). |
| Historical coverage | Depends on modules 4–9. `sn_mis_*` and campaign tables have no spec. |
| Reconciliation | INV-021 card=table=detail parity in tests (synthetic). |
| Runtime evidence | none. |
| Rollback | not applicable. |
| **Classification** | **NOT VERIFIED** for a shadow pilot. **NO-GO** for authority (OPD-07, D-23/D-24/D-45, sources missing). |
| Blockers / closure evidence | Real-data metric parity (INV-036) and the decisions above. |

### 12. Biawin subscriptions

| Field | Value |
|---|---|
| Legacy SoR | `sn_project_memberships`, membership items, Biawin API (legacy client). |
| Target implementation | `internal/biawin`: Slice 10 read shadow plus Slice 14 SHADOW activation decisions. |
| Read paths | 4 subscriptions/parity GETs and activation decisions. |
| Write paths | none (activation is an event consumer with no producer). |
| Blocked commands | Production activation (D-35), suspension/reactivation (D-35c), wallet charge (D-38). |
| Flags (default) | `biawin_fulfillment.read_shadow`, `fulfillment.work_read` (OFF). |
| Permission assignments | none granted. |
| External dependencies | Biawin (no connector). |
| Workers | Relay (U6). |
| Historical coverage | memberships, items, actions, events, links. `sn_project_subscriptions/contents` and API configs have no spec. |
| Reconciliation | Active-subscription parity on synthetic data. |
| Runtime evidence | none. |
| Rollback | not applicable. |
| **Classification** | **NO-GO** (D-35a–e, D-38, no connector). |
| Blockers / closure evidence | See [biawin-integration-readiness.md](biawin-integration-readiness.md). |

### 13. Fulfillment and operations

| Field | Value |
|---|---|
| Legacy SoR | `sn_operations_execution_*`, `sn_project_operations`, shipping orders. |
| Target implementation | `internal/fulfillment`: Slice 10 read shadow plus Slice 14 work cases (target-native SHADOW only). |
| Read paths | 3 legacy-case GETs and 2 work-case GETs. |
| Write paths | assignment, operational action, evidence, outcome (family `fulfillment.work`). |
| Blocked commands | cancel/reopen (OPD-04), review (OPD-08), reassign started (OPD-05), wallet charge (D-38), commands on legacy cases. |
| Flags (default) | `fulfillment.work_read`, `fulfillment.work_write`, `fulfillment.work_writeback` (OFF). |
| Permission assignments | none granted. Catalog scopes are empty. |
| External dependencies | Catalog, providerops. |
| Workers | Relay (U6). |
| Historical coverage | execution cases/events, form submissions, shipping. `sn_operations_form_definitions` and `wallet_codes` have no spec. |
| Reconciliation | Synthetic only. |
| Runtime evidence | none. |
| Rollback | Flag OFF. Target cases remain. |
| **Classification** | **NO-GO** (no activation producer, empty catalog, no relay). |
| Blockers / closure evidence | Catalog approvals, relay process, D-35. |

### 14. External integrations

| Field | Value |
|---|---|
| Legacy SoR | Legacy plugin talks to Zarinpal, Biawin, SMS. |
| Target implementation | Write-back worker and connector interface, synthetic gateway, providerops ledger, Biawin `DisabledPort`/`SyntheticPort`. |
| Read paths | writeback diagnostics, provider operations. |
| Write paths | provider resolution (human). |
| Blocked commands | Any live provider call. |
| Flags (default) | `integration.writeback_read`, `integration.provider_reconcile` (OFF). |
| Permission assignments | none granted. |
| External dependencies | All providers missing. |
| Workers | U6. |
| Historical coverage | n/a. |
| Reconciliation | n/a. |
| Runtime evidence | none. |
| Rollback | n/a. |
| **Classification** | **NO-GO** (U6, U7, no real adapters). |
| Blockers / closure evidence | Real adapters, provider contracts, secrets management. |

### 15. Customer portal

| Field | Value |
|---|---|
| Legacy SoR | Legacy customer portal (`class-sn-customer-portal.php`), public payment tokens. |
| Target implementation | **none** (architecture §15.10 not built). |
| Read paths | none. |
| Write paths | none. |
| Blocked commands | Self-service, auto-login (D-26). |
| Flags (default) | none. |
| Permission assignments | n/a. |
| External dependencies | n/a. |
| Workers | n/a. |
| Historical coverage | n/a. |
| Reconciliation | n/a. |
| Runtime evidence | n/a. |
| Rollback | n/a. |
| **Classification** | **NO-GO** (not implemented). |
| Blockers / closure evidence | Design, implementation, D-26. |

### 16. Background workers and outbox/inbox

| Field | Value |
|---|---|
| Legacy SoR | WordPress cron and synchronous hooks. |
| Target implementation | Per-module outbox/inbox/idempotency libraries, `relay.Drain`, worker steps (library functions). |
| Read paths | — |
| Write paths | — |
| Blocked commands | — |
| Flags (default) | — |
| Permission assignments | — |
| External dependencies | A process supervisor / scheduler (none). |
| Workers | **none deployable** (U6). |
| Historical coverage | n/a. |
| Reconciliation | Fault-injection tests (synthetic). |
| Runtime evidence | none. |
| Rollback | n/a. |
| **Classification** | **NO-GO**. |
| Blockers / closure evidence | Worker commands, scheduling, backlog metrics and alerting. |

### 17. Observability and audit history

| Field | Value |
|---|---|
| Legacy SoR | Legacy logs and event tables. |
| Target implementation | `slog` JSON logs with `X-Correlation-ID`, `/health/live` and `/health/ready`, append-only audit schema, sensitive-read audit for Invoice Review. |
| Read paths | health endpoints. |
| Write paths | audit rows (sensitive reads). |
| Blocked commands | — |
| Flags (default) | — |
| Permission assignments | — |
| External dependencies | Log shipping and metrics backend (none). |
| Workers | — |
| Historical coverage | Legacy event history is imported per module only where specs exist. |
| Reconciliation | — |
| Runtime evidence | none. |
| Rollback | — |
| **Classification** | **NOT VERIFIED**. No metrics, alerting or security log pipeline is defined. |
| Blockers / closure evidence | Operator evidence (checklist §7). |

**Result: no module is GO or CONDITIONAL GO for production authority.** Every module stays under
legacy 2.0.160 authority.
