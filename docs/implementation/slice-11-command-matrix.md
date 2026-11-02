# Slice 11 — Controlled non-financial writes: command matrix

Status: implementation record for Slice 11 (branch `feat/slice-11-controlled-nonfinancial-writes`,
baseline `main @ b639d4a`). This record resolves **no** owner decision. Every command below was proven
against the legacy 2.0.160 source (`plugin/current`, evidence only, unmodified) and the frozen contracts
(`CRM-STANDALONE-TARGET-ARCHITECTURE-V1.md`, `CRM-CROSS-ROLE-ARCHITECTURE-V1.md`,
`E0-governance-baseline-freeze.md`, `GATE-0-PRODUCT-INVARIANTS.md`). Where evidence or policy is
insufficient the command is **blocked**, never approximated.

## Recorded condition

Architecture V1 §34 states "ADR-01 must be accepted before Slice 11". ADR-01 is still a **candidate**.
The Human Owner's Slice 11 instruction directs implementation of the non-financial subset with ADR-01
unaccepted and its safe default enforced: **no target invoice-issue write and no target return write**.
This record keeps ADR-01 a candidate; nothing here accepts it.

## Legacy write paths inspected

| Legacy path (2.0.160) | Evidence | Used for |
|---|---|---|
| `ajax_update_lead_status` | `class-sn-plugin.php:28002-28107` | owner-only lead outcome update |
| `SN_Seller_Flow::statuses / validate_status_change / apply_activity` | `class-sn-seller-flow.php:168-471` | outcome vocabulary, reason/callback rules, archive refusal, `request_token` idempotency |
| `sn_log_status_history` | `class-sn-plugin.php:29949` | lead creation only (not an outcome path) |
| `ajax_assign_leads` (manual/count) | `class-sn-plugin.php:2505-2599` | supervisor pool → direct seller allocation |
| `ajax_assign_supervisor_leads` | `class-sn-plugin.php:2601-2667` | MIS/manager → supervisor pool (bulk by filter) |
| `ajax_supervisor_unassign_leads` | `class-sn-plugin.php:30042-30120` | return to pool (F01 class) |
| Dot conversion "ready" | `class-sn-dot-flow.php:2495-2678, 5248` | ready-for-conversion (Dot paid referral / customer option) |
| `handle_hr_change_request_create` | `class-sn-plugin.php:10443-10485` | HR change request creation |
| `sn_hr_change_request_can_review / handle_hr_change_request_review` | `class-sn-plugin.php:10488-10568` | step review, rejection, final apply |
| `sn_hr_change_request_apply` | `class-sn-plugin.php:10497-10527` | final apply: `assign_parent` / profile termination |

## Matrix

Legend: **SAFE** = active behind its flag (default OFF); **BLOCKED** = command unavailable, answered
`BLOCKED` (or absent route) with the decision ids named.

### Cases

| Field | `RecordContactOutcome` |
|---|---|
| owner module | cases |
| actor / capability | `cases.contact_outcome.record` (granted to no real role) |
| row scope | actor user id == current legacy lead holder (custody published read `legacy_holder_user_id`, case PRESENT); unknown holder → BLOCKED; any other actor (any rank) → UNAUTHORIZED. Legacy `manage_options` override is **not** ported. |
| field scope | `outcome` ∈ {`NO_ANSWER`, `CALLBACK`, `NOT_PURCHASED`}, `reason` (required for NOT_PURCHASED), `callback_at` (required for CALLBACK), `expected_version`; any other body field → 400 |
| blocked values | `DUPLICATE` → BLOCKED (D-29/D-31: phone duplicate intent unresolved); `PRE_INVOICE` → BLOCKED (ADR-01: only via invoice issue) |
| preconditions | case `LEGACY_LEAD` (distribution items: path not proven → BLOCKED); case PRESENT; `expected_version` == case version |
| transition | sales-workflow current outcome := new outcome; case version + 1; contact-outcome history appended |
| legacy source path | `ajax_update_lead_status` + `SN_Seller_Flow::apply_activity` |
| legacy side effects | `sn_leads.lead_status/note`, seller-flow state + event (`request_token`), campaign mark; stage-one statuses never route destination panel |
| cross-module reads | custody (holder), identity via integration (legacy actor key) |
| cross-module effects | none synchronous; outbox `cases.contact_outcome_recorded` → integration write-back |
| idempotency key | `Idempotency-Key` header, scope `cases.contact_outcome`; legacy `request_token` = command id |
| concurrency | `SELECT … FOR UPDATE` on the caller-owned case row + version check; stale → CONFLICT |
| flags | `cases.nonfinancial_write` (command) + `cases.nonfinancial_writeback` (CONTROLLED_WRITEBACK vs SHADOW_COMPARE) |
| write-back | lead id = case source key; label = legacy Persian status label; `callback_at` → Jalali Tehran `YYYY/MM/DD HH:MM`; legacy archive refusal → `REJECTED` + reconciliation |
| open decisions | D-29, D-31, ADR-01 (value-level blocks); OPD-01 (aliases not followed) |
| **safe?** | **SAFE (active)** |

| Command | Decision | Reason |
|---|---|---|
| `MarkReady` | BLOCKED | the only legacy "ready" is Dot `ready_for_conversion`, entangled with paid-referral migration and customer option selection; Dot cases are not target cases; authority (Seller → Sup ready assignment, "receiving ≠ assign") unproven for a target command |
| `CreateCase`, `LinkAlias`, `ClaimCaseForInvoice` | BLOCKED | out of Slice 11; ClaimCaseForInvoice is the ADR-01 candidate protocol (not approved) |
| billing `IssueInvoice` | BLOCKED (no route, no command) | ADR-01 candidate; Slice 12+ financial |

### Custody

| Command | Decision | Reason |
|---|---|---|
| `AllocateToSeller` (supervisor pool → direct seller) | BLOCKED | the legacy precondition (`sn_leads.status = 'supervisor_pool'` and `supervisor_id = actor`) is not represented in target custody: legacy leads carry only the seller holder column, so the pool owner and pool state cannot be proven from target evidence. An allocation that cannot prove its source is not enabled. |
| `Transfer` (holder → holder) | BLOCKED | no single legacy transfer command for leads (unassign + assign); distribution-chain forwards and skip-level paths are OPD-10 / SD-01 conditional |
| `ReturnCase` / `Release` / `Reset` | BLOCKED | ADR-01 (no target return write), F01 (union of protected dependencies unproven), OPD-03 (cancelled release / reset / revival) |

No custody write route exists. Current custody, original owner, original seller, credit owner and event
actor are untouched by every Slice 11 command.

### Organization (HR change requests)

| Field | `CreateHRRequest` | `DecideHRRequest` |
|---|---|---|
| owner module | organization | organization |
| capability | `org.hr_request.create` (no real role) | `org.hr_request.review` (no real role) |
| actor evidence | one LINKED, ACTIVE, PRESENT employment; position slug ∈ {`supervisor`, `senior_supervisor`, `sales_manager`, `sales_deputy`} (legacy guard); `hr` creator → BLOCKED (HR runtime NOT LIVE VERIFIED; all-sellers scope) | actor == request's current reviewer snapshot, status `pending_review` |
| row scope | subject is a seller **descendant** of the actor through current REPORTS_TO links (exactly one current parent per step; ambiguity → BLOCKED) | the request row (current reviewer) |
| request types | `transfer_seller`, `terminate_seller` (the only evidenced types) | `approve` (step), `reject` |
| SD-04 | `transfer_seller` target must be an ACTIVE supervisor **within the actor's own descendant scope**; a destination outside it → BLOCKED (SD-04); target == subject's current parent → BLOCKED (SD-04 no-op not frozen) | — |
| duplicate guard | open (`pending_review` / `pending_hr`) target request for the same subject + type → CONFLICT (legacy guard) | — |
| next reviewer | actor's current parent (legacy `next_reviewer_user_id`); none → `pending_hr`; ambiguous → BLOCKED | same rule on step approval |
| final apply | — | actor is HR or status `pending_hr` → BLOCKED: OPD-05 (handover / successor / access timing), SD-04 (destination), HR apply IMPLEMENTATION BLOCKED (cross-role §21). No hierarchy change, no termination, no credential or grant write. |
| history | append-only request events (actor, from/to state, next reviewer, notes, command, correlation) | same |
| snapshots | requester, subject current parent, target, current reviewer captured at creation; never re-derived later | — |
| legacy path | `handle_hr_change_request_create` | `handle_hr_change_request_review` (step + reject branches only) |
| write-back | insert legacy change request (reconcile by subject + type + requester) | update legacy request by the legacy id returned by the create write-back; waits while the create is not confirmed |
| flags | `organization.hr_request_write` + `organization.hr_request_writeback` | same |
| **safe?** | **SAFE (active)** | **SAFE (step/reject only); final approve BLOCKED** |

D-42: no HR command writes identity, role grants or capabilities. D-34: no credential is created; no
Slice 11 path reaches account creation.

## Rollout modes (per family)

| Mode | Flags | Behaviour |
|---|---|---|
| `TARGET_WRITE_DISABLED` | `<family>_write` OFF (default) | command answers BLOCKED / FEATURE_DISABLED; legacy only |
| `SHADOW_COMPARE` | `<family>_write` ON, `<family>_writeback` OFF | command fully evaluated (authority, scope, invariants, version); recorded as an evaluated intent; **no aggregate change, no legacy write**; integration observes legacy and records parity of the expected effect |
| `CONTROLLED_WRITEBACK` | both ON | target commits aggregate + history + outbox; integration writes back to legacy (legacy remains master), reconciles UNKNOWN, records parity |

Turning a family flag OFF stops new target commands; history and committed intents stay; already
committed intents are still delivered so legacy (the master) receives them.

## Write-back sequence

owner transaction (aggregate + history + command record + idempotency + outbox) → outbox dispatch →
integration consumer (inbox-deduplicated, own transaction) records the write-back command → attempt
(connector) → result recorded append-only → UNKNOWN: observe legacy first (reconcile), then mark
committed, retry safely, or open reconciliation → parity of expected vs observed.

Transport success is never business success; UNKNOWN never becomes FAILED automatically; a rejected
write-back never rewrites target history.
