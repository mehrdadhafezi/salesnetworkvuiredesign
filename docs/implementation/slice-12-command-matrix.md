# Slice 12 — Controlled financial writes: command and decision matrix

Status: implementation record for Slice 12 (Architecture V1 §15, §16, §31 row 12, §32).
This record resolves **no** owner decision. Legacy (Sales Network 2.0.160, `plugin/current`)
remains the master. Every Slice 12 flag defaults OFF and every Slice 12 capability is granted
to no role.

## 1. Recorded conditions

- Architecture V1 §34: "ADR-01 must be accepted before Slice 11". ADR-01 is still a
  **CANDIDATE**. Slice 12 therefore implements **no** invoice issuance, claim or return:
  `ClaimCaseForInvoice`, `IssueInvoice`, `ReturnCase` have no route and no command.
- Cross-Role V1 §21 row "Posting / idempotency / unknown recovery": *IMPLEMENTATION BLOCKED
  execution readiness* (OPD-08/09). Slice 12 adds **no** ledger posting.
- HR and Finance runtime remain NOT LIVE VERIFIED (E0). No Finance role behaviour is inferred
  from administrator behaviour.

## 2. Decision readiness (Phase 2)

| Decision | Authoritative definition | Classification | Effect in Slice 12 |
|---|---|---|---|
| ADR-01 | Arch §6 (candidate reservation protocol), §32 | REQUIRES HUMAN OWNER DECISION | no target `ClaimCaseForInvoice` / `IssueInvoice` / `ReturnCase` |
| F01 | Cross-Role §5 (P0), §19 register, §21 | IMPLEMENTATION BLOCKED | no return / reset / release |
| D-25 | Arch §15.7, §32 | SAFE DEFAULT AVAILABLE | gateway success recorded as `GATEWAY_AUTO` provenance; never a Finance review; target never transitions the billing stage |
| D-26 | Arch §7 (Session), §15.7, §32 | SAFE DEFAULT AVAILABLE | a callback never creates a session or logs anyone in |
| D-28 | Arch §5 (Converter/Dot), §32 | SAFE DEFAULT AVAILABLE | 2.0.160 assessment full-payment rule is a plan rule enforced at invoice issuance; target issues no invoice and changes no plan, so the rule is preserved unchanged |
| D-32 | Arch §15.9, §32 | SAFE DEFAULT AVAILABLE (owning seller + evidenced admin) | target `IssueNextStage` is still BLOCKED: billing has no stage state machine (lifecycle `UNKNOWN`), so the §15.9 preconditions cannot be evaluated |
| D-33 | Arch §5 (Operations Sales), §32 | SAFE DEFAULT AVAILABLE (not granted) | no operations upgrade-payment authority |
| D-40 | Arch §5 #25, §32 | SAFE DEFAULT AVAILABLE (read-only import) | no reward / wheel write |
| D-41 | Arch §16, §32 | REQUIRES HUMAN OWNER DECISION (Finance) | no posting; shadow ledger keeps computing both bases and publishing neither |
| D-43 | Arch §15.6, §32 | SAFE DEFAULT AVAILABLE (approval blocked unless READY) | review gate refuses any decision while a receipt is not `READY`; info-only (receipt-less) submission is BLOCKED |
| D-44 | Arch §15.1, §32 | SAFE DEFAULT AVAILABLE (bound evidence kept indefinitely; unbound 7 days) | unbound uploads expire after 7 days; bound evidence is never deleted; no legal-retention policy is invented |
| OPD-04 | Cross-Role §19 | REQUIRES HUMAN OWNER DECISION | no refund execution / confirmation |
| OPD-06 | Cross-Role §19, Arch §16 | REQUIRES HUMAN OWNER DECISION | no rounding, tolerance or IRR↔IRT conversion: exact integer comparison in one unit; any other unit or unrepresentable amount → BLOCKED |
| OPD-08 | Cross-Role §19, Arch §17, INV-025 | REQUIRES HUMAN OWNER DECISION | no Finance approve / reject command; the evaluator still refuses a reviewer who is the submitter (fail closed) |
| OPD-09 | Cross-Role §19 | REQUIRES HUMAN OWNER DECISION | no entitlement / commission posting |
| OPD-10 | Cross-Role §19 | REQUIRES HUMAN OWNER DECISION | no supervisor / manager / Finance "assist" payment submission |

## 3. Legacy financial write paths inspected (2.0.160, read only)

| Path | Location | Finding |
|---|---|---|
| Manual (card-to-card) payment | `ajax_submit_manual_payment` (`class-sn-plugin.php:30348`), `sn_save_invoice_manual_payment` (`:30189`) | one transaction per request; invoice + stage row lock; `submission_key` UNIQUE per invoice (idempotent replay); amount ≤ stage due − pending; invoice/stage terminal statuses refused; receipt mode (amount + files) or info mode (transfer details) |
| Receipt upload | `SN_Helpers::upload_receipt` (`class-sn-helpers.php:571`) | ≤ 10 files, ≤ 5 MB each, ≤ 20 MB total, jpg/png/webp/pdf by content check; **"a partial batch must never succeed"**; files removed at shutdown unless the DB transaction commits |
| Internal submit scope | `sn_can_perform_internal_invoice_payment_action` (`:29514`) | seller: own invoice, `manual_payment` only, conversion invoices only if issuer; supervisors/managers by hierarchy scope (OPD-10); Finance/admin may submit (maker/checker conflict, OPD-08) |
| Supervisor receipt | `ajax_supervisor_upload_receipt` (`:30455`) | hierarchy assist path (OPD-10) |
| Finance approve / bulk approve | `ajax_financial_approve_payment` (`:30614`), `ajax_financial_bulk_approve` (`:30629`) | capability only (`sn_approve_payment` or `manage_options`); no maker/checker; bulk per-item results; approval finalises stage and triggers wallet/commission effects |
| Finance reject | `ajax_financial_reject_payment` (`:30649`) | capability only; rejects every pending manual payment of the stage; writes invoice + stage + payments; cancels a reopened Woo order; fires `sn_invoice_financial_rejected` (listeners unverified) |
| Receipt reject / card confirm | `ajax_reject_receipt` (`:27623`), `ajax_confirm_card_payment` (`:27563`) | administrator review paths (OPD-08) |
| Gateway start | `ajax_pay_online` (`:6934`) | public token or internal; refuses while manual payments pend; amount = stage due |
| Gateway callback | `handle_zarinpal_callback` (`:7088`) | server-side verify; amount tolerance 0.5 (OPD-06); **auto-approves the stage** (D-25); **logs the customer in** (`wp_set_auth_cookie`, D-26); `reconcile_error` on finalise failure |
| Next stage | `ajax_converter_issue_payment_stage` (`:6916`) | converter or admin (D-32) |
| Invoice issue | `ajax_create_invoice` | ADR-01 |
| Payment batch (multi-attempt) | `nopriv_sn_submit_payment_batch` | **absent from 2.0.160** (later overlay; not authority) |

## 4. Command matrix

Legend: ACTIVE = implemented behind flags OFF and capabilities granted to nobody;
SERVICE-ONLY = implemented and tested without any route; BLOCKED = no route, no command.

| Command | Owner | Actor / capability | Row / field scope | Preconditions (apply-time recheck) | State change | Legacy evidence | Decisions | Idempotency | Concurrency | Side effects | Rollback | Permitted |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| `payments.StageUpload` | payments | uploader / `payments.upload.stage` | caller's own uploads only | flag `payments.manual_write`; ≤ 5 MB; content-sniffed jpg/png/webp/pdf; server-computed SHA-256 | durable quarantine object, then `upload_object` UNBOUND (expires +7 days) | `upload_receipt` | D-44 | `(uploader, sha256)` returns the existing upload | unique `(uploader, sha256)` | none | flag off; unbound objects expire | **ACTIVE** |
| `payments.SubmitBatch` (receipt mode) | payments | owning seller / `payments.batch.submit` | `invoice.seller_user_id = actor` and `issued_by_user_id = actor`; fields: `item_key`, `amount`, `unit`, receipts `{upload_id, content_sha256}` | flag mode `CONTROLLED_WRITEBACK`; invoice PRESENT; stage coverage complete; current stage PRESENT, state PENDING / AWAITING_REVIEW / REJECTED; invoice main status known and not `paid/approved/cancelled/payment_archived`; due and remaining representable in one unit; due ≤ remaining; Σ items ≤ due − (pending + accepted, imported + target); every receipt UNBOUND, own, unexpired, hash match; ≤ 10 items, ≤ 10 receipts, ≤ 20 MB | batch + items + attempts (`PENDING_REVIEW`, provenance `NOT_REVIEWED`) + receipts `QUARANTINED`, uploads BOUND, history, command record, outbox — **one payments transaction, all-or-nothing** | `sn_save_invoice_manual_payment`, `upload_receipt` | D-43, D-28, OPD-06, OPD-10 | `Idempotency-Key` + request hash (incl. actor) → replay / CONFLICT; stable batch, item and attempt ids | `If-Match` collection version (0 before the first batch) + `collection_account FOR UPDATE` per (invoice, stage) | none outside payments | flag off; attempts stay as history; legacy decides | **ACTIVE** |
| `payments.SubmitBatch` (info mode, no receipt) | payments | — | — | — | — | info mode | D-43 | — | — | — | — | BLOCKED |
| multi-invoice batch | payments | — | — | — | — | absent in 2.0.160 | — | — | — | — | — | BLOCKED |
| supervisor / manager / Finance / converter submit | payments | — | — | — | — | `sn_can_perform_internal_invoice_payment_action` | OPD-10, OPD-08 | — | — | — | — | BLOCKED |
| customer self-service submit / pay online | payments | — | — | — | — | public token routes | §15.10 not built, D-26 | — | — | — | — | BLOCKED |
| `payments.FinaliseReceipt` (worker) | payments | service actor | receipt | receipt QUARANTINED / STORAGE_ERROR | write-ahead FINALIZING → content-addressed copy `evidence/<sha256>` → hash verify → READY; failure → STORAGE_ERROR; missing / mismatched bytes → NOT_RECOVERABLE + reconciliation case | `commit_receipt_uploads` | D-44 | re-run finds the object and only updates state | receipt version | when every receipt of an attempt is READY: outbox `payments.attempt_evidence_ready` | never rolls back the financial transaction | **SERVICE-ONLY** |
| `payments.ExpireUnbound` (worker) | payments | service actor | upload | UNBOUND and `expires_at` passed (DB check) | upload PURGED, quarantine bytes deleted | shutdown cleanup | D-44 | idempotent | conditional update on UNBOUND | none | — | **SERVICE-ONLY** |
| `payments.StartGatewayAttempt` | payments (+ integration adapter) | — (no route) | — | stage due, no pending manual attempt | write-ahead UNKNOWN → REDIRECT_ISSUED / FAILED_AT_GATEWAY | `ajax_pay_online` | D-25, D-26, §15.10 | stable request id from key | request version | none | — | **SERVICE-ONLY** (synthetic gateway; no route; production unsupported) |
| `payments.RecordGatewayCallback` / `ReconcileGatewayAttempt` | payments | service actor | request | verified callback (adapter), inbox `(provider, external_ref, status)` | VERIFIED (provenance `GATEWAY_AUTO`, not Finance-reviewed) / FAILED / CANCELLED / RECONCILIATION_REQUIRED | `handle_zarinpal_callback` | D-25, D-26 | inbox dedupe; replay = no-op | request version | no session, no stage change | — | **SERVICE-ONLY** |
| `finance.ApproveAttempt` | finance | — | — | — | — | `ajax_financial_approve_payment` | D-41, OPD-08, OPD-09, D-43 | — | — | — | — | BLOCKED (evaluator only) |
| `finance.RejectAttempt` | finance | — | — | — | — | `ajax_financial_reject_payment` | OPD-08; billing rejection transition contract not defined (Arch §16 defines only the approve path); legacy hook listeners unverified | — | — | — | — | BLOCKED (evaluator only) |
| bulk Finance approve | finance | — | — | — | — | `ajax_financial_bulk_approve` | OPD-08 | — | — | — | — | BLOCKED |
| ledger posting / reversal command | finance | — | — | — | — | wallet / commission posting | D-41, OPD-06, OPD-08, OPD-09 | — | — | — | — | BLOCKED |
| wallet charge, settlement, refund, commission | finance / biawin | — | — | — | — | various | D-38, OPD-04, OPD-09 | — | — | — | — | BLOCKED |
| `billing.IssueNextStage` | billing | — | — | — | — | `ajax_converter_issue_payment_stage` | D-32 (+ billing lifecycle UNKNOWN) | — | — | — | — | BLOCKED |
| operations upgrade payment | payments / fulfillment | — | — | — | — | `SN_Operations_Flow` | D-33 | — | — | — | — | BLOCKED |
| `billing.IssueInvoice`, `cases.ClaimCaseForInvoice`, `custody.ReturnCase` | billing / cases / custody | — | — | — | — | `ajax_create_invoice`, return paths | ADR-01, F01 | — | — | — | — | BLOCKED |
| Biawin activation / fulfillment completion | biawin / fulfillment | — | — | — | — | — | D-35, D-38 | — | — | — | — | BLOCKED |

## 5. Evidence lifecycle (Arch §15.1–15.5)

```
client bytes ─▶ StageUpload ─▶ quarantine/<upload_id> (fsync) ─▶ upload_object UNBOUND (expires +7d)
                                                                       │ SubmitBatch (one payments tx)
                                                                       ▼
                                         upload BOUND ─ receipt QUARANTINED ─▶ FINALIZING (write-ahead)
                                                                       │ copy-if-absent evidence/<sha256>, verify
                                         ┌─────────────────────────────┼────────────────────────────┐
                                         ▼                             ▼                            ▼
                                       READY                     STORAGE_ERROR ─ reconcile ─▶ NOT_RECOVERABLE
                              (all READY ⇒ outbox                (quarantine kept)           (+ reconciliation case,
                               attempt_evidence_ready)                                        Finance owner)
UNBOUND past expires_at ─▶ PURGED (row kept, bytes deleted; never for a bound upload)
```

- **Uploaded, unbound:** `upload_object.state = UNBOUND`; not evidence.
- **Quarantined:** receipt `QUARANTINED`: bound to exactly one attempt, batch, invoice and command; bytes durable in quarantine.
- **Undergoing validation:** receipt `FINALIZING` (write-ahead claim by the finaliser).
- **READY:** finalised, content-addressed, hash verified, bound.
- **Rejected:** content that fails validation is refused at upload (nothing stored); a Finance receipt rejection is a review command (BLOCKED, OPD-08).
- **Missing / unrecoverable:** `STORAGE_ERROR` (retry from quarantine) and `NOT_RECOVERABLE` (+ reconciliation case).
- **Replacement:** a new submission adds new receipts; earlier receipts, attempts and their history are never rebound, overwritten or deleted.
- Receipt state changes append to `receipt_state_history`; the content hash is immutable.
- A storage failure before commit leaves no attempt; a finalisation failure never rolls back the attempt.

## 6. Batch semantics

Authority: Arch §15.2 (validate the whole envelope first; one payments transaction) and
`upload_receipt` ("a partial batch must never succeed"). Partial success is **not** established
by any authority, so a batch is **all-or-nothing**: each item is validated and reported with its
own status and reasons; if any item is not acceptable, nothing is committed and the batch status
is that refusal. `PARTIAL` is never produced by Slice 12. A batch covers one invoice's current
stage (2.0.160 has no multi-invoice submission).

## 7. Rollout and write-back

- Family `payments.manual`: `payments.manual_write` + `payments.manual_writeback`. Only
  `CONTROLLED_WRITEBACK` accepts a submission; `SHADOW_COMPARE` is BLOCKED for payments (a
  target submission shares no identity with any legacy row, so there is nothing to compare).
- `finance.review_read`: Finance review queue (read only).
- Write-back reuses the Slice 11 integration pipeline: payments outbox
  `payments.attempt_evidence_ready` → integration inbox → write-ahead UNKNOWN → connector
  `Apply` (legacy `sn_save_invoice_manual_payment` with `submission_key` = attempt id) →
  observe-before-retry → parity. Ambiguous results go to `RECONCILIATION_REQUIRED`; retry only
  after proven absence. The only connector is the **synthetic** one: Production delivery is
  **unsupported**.

## 8. Rollback

Turn `payments.manual_write` off: new submissions answer BLOCKED. Committed attempts, receipts,
history and write-back records stay readable; nothing is deleted. Corrections are never made
by mutation: an attempt can only be superseded by legacy decisions recorded through
reconciliation. No ledger entry exists to reverse.

## 9. Readiness states (never conflated)

| State | Slice 12 |
|---|---|
| Code complete | the bounded safe subset above (upload, owning-seller batch, evidence finalisation / reconciliation / expiry, synthetic gateway flow, review gate, payments write-back mapping) |
| Tests passed | see the PR (PostgreSQL integration, race, fault injection, HTTP, lint, OpenAPI, web) |
| Decision blocked | ADR-01, F01, D-41, OPD-04, OPD-06, OPD-08, OPD-09, OPD-10; Finance approve / reject, posting, refund, settlement, next stage, invoice issue, return |
| Production connector unavailable | no legacy write-back connector, no gateway adapter, no worker binary (finaliser / expiry / write-back are service-level only); evidence storage is the filesystem store (durability class of the production volume not verified) |
| Merge ready | decided by review and CI only |
| Production ready | **NO**: no role holds any Slice 12 capability, production authentication does not exist, no worker runs, HR / Finance runtime NOT LIVE VERIFIED |

## 10. Known gaps

- Every imported legacy attempt must be linked to an invoice before any submission is accepted
  (an unlinked attempt blocks availability: COVERAGE_INCOMPLETE). Real data may need the Slice 7
  link reconciliation first.
- A conversion invoice cannot be told apart in the target: the seller must also be the issuer.
- Imported stage / remaining evidence is snapshot data; legacy re-checks at write-back and refuses
  stale submissions (REJECTED + reconciliation).
- No receipt file is served by the target (no download route).
- Legal retention for NOT_RECOVERABLE records (D-44) is not defined; rows are kept indefinitely.
