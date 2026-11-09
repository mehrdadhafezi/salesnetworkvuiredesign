# Slice 15 — Financial readiness assessment

**No financial activation is authorized.** This document reports implemented execution boundaries at
`b3bf31b`. Legacy 2.0.160 is the financial system of record for every operation below.

> **R4 update (2026-11-09; historical text below unchanged).** R4 adds contracts and evaluators only:
> - the Finance decision contract (`review.EvaluateDecision`);
> - the ledger posting contract (`finance/posting`);
> - the refund state machine (`finance/refund`);
> - commission cross-engine reconciliation.
>
> Every capability row below keeps its state: approve / reject is EVALUATOR-ONLY; posting, refund,
> wallet, commission posting and settlement are BLOCKED. See `docs/remediation/r4-decision-gates.md`.
> **Overall financial readiness: still NO-GO.**

## 1. Execution boundary per capability

**Capability states used in the table:**
- **ACTIVE**: route exists behind flags OFF, with capabilities granted to no role.
- **SERVICE-ONLY**: implemented and tested, but with no route and no running process.
- **EVALUATOR-ONLY**: the gate computes a decision but executes nothing.
- **BLOCKED**: no command exists.

| Capability | Target state | Evidence | Legacy path (2.0.160) | Blocking decisions / gaps |
|---|---|---|---|---|
| Manual payment submission (receipt mode) | ACTIVE (flag `payments.manual_*`) | `cmd/crm-api/payments.go:36,63-71`; Slice 12 matrix §4 | `ajax_submit_manual_payment` (`class-sn-plugin.php:30348`) | U1 (no auth), U6 (no finaliser), U7 (no write-back), D-43, OPD-06, OPD-10 |
| Gateway payment | SERVICE-ONLY, synthetic gateway only | `internal/payments/command/gateway.go:38,152` (`GATEWAY_AUTO` provenance) | `ajax_pay_online`, `handle_zarinpal_callback` (`:6934`, `:7088`) | D-25, D-26, no real gateway adapter, no callback route |
| Receipt upload and durable storage | ACTIVE upload. Durability depends on the filesystem object store. | `internal/platform/objectstore/objectstore.go:1-7` (temp file, fsync, hash verify, rename, dir fsync) | `SN_Helpers::upload_receipt` | Production storage class, backup and replication NOT VERIFIED; D-44 retention |
| Evidence finalisation | SERVICE-ONLY (`Service.Finalise`) | `internal/payments/command/finalise.go:43` | shutdown cleanup | **No process runs it (U6).** Receipts would stay QUARANTINED. |
| Unbound upload expiry | SERVICE-ONLY (`ExpireUnbound`) | `finalise.go:321` | — | No process (U6); D-44 |
| Finance approve / reject | EVALUATOR-ONLY (`review.Evaluate`; maker ≠ checker fail-closed) | `internal/finance/review/review.go:13,26,71` | `ajax_financial_approve_payment` / `reject` (`:30614`, `:30649`) | OPD-08, D-41, D-43, OPD-09; rejection transition contract undefined |
| Bulk approve | BLOCKED | Slice 12 matrix row | `ajax_financial_bulk_approve` | OPD-08 |
| Ledger posting / reversal | BLOCKED. Only the shadow ledger is derived offline. | Slice 12 §1, §4; `internal/finance/domain/ledger.go` | wallet / commission writes on approval | D-41, OPD-06, OPD-08, OPD-09 |
| Refund execution | BLOCKED | Cross-Role §19 OPD-04 | Finance reject, Woo order cancel | OPD-04, INV-015 |
| Wallet credit / debit | BLOCKED | no wallet write anywhere in target (CI money-column invariants, Slice 14 test 28/29) | four commission engines, Biawin `charge_wallet` | OPD-09, D-38 |
| Commission posting | BLOCKED (`CommissionPostingRequested` never dispatched) | Slice 13 matrix §1, §10 | E1–E4 | OPD-08, OPD-09, D-36 |
| Settlement | BLOCKED | — | — | OPD-08/09 |
| Unknown provider outcomes | Write-ahead UNKNOWN, then reconcile before retry (gateway, write-back, providerops) | `gateway.go:20-25`; `writeback/worker.go`; `providerops.go:307,394` | timeouts treated as failure | No process runs reconcile (U6) |
| Reconciliation and recovery | Library functions plus human resolution (providerops) | `providerops.go:464` | none | No operator runbook in production; no worker |

## 2. Control checks

| Control | Status | Evidence |
|---|---|---|
| Separation of duties | **PARTIAL (design).** The gate forbids submitter = reviewer. The thresholds and exceptions policy is open (OPD-08). No approval executes. | `review.go:13` |
| Idempotency | VERIFIED in code. `Idempotency-Key` on every command. Same key + same request replays; a changed request is CONFLICT. | `internal/platform/idempotency`; Slice 11/12/14 tests |
| Stable business keys | VERIFIED for target writes (submission key per invoice; activation business key; provider reference). Entitlement identity for commission (INV-032) exists in shadow only. | Slice 12 §6; Slice 13 §1 |
| Exact monetary units | VERIFIED. Integer minor units plus explicit unit; no float; no IRR↔IRT conversion; an unrepresentable amount is BLOCKED (OPD-06 safe default). | kernel money; CI money-column invariants |
| Evidence durability | Code VERIFIED (fsync + hash). Production storage NOT VERIFIED. | `objectstore.go` |
| Outbox / inbox delivery | Library VERIFIED by fault-injection tests. **No deployed relay** (U6). | `relay.go:30`; `TestRelayFaultInjection` |
| Historical balances | Shadow-derived on synthetic data only. Real-data parity NOT RUN. | [data-reconciliation-register.md](data-reconciliation-register.md) |
| Negative / partial payment | Amount ≤ stage due − pending (legacy rule ported in the batch preconditions). Partial assessment policy is D-28 (2.0.160 full-payment rule kept). | Slice 12 matrix |
| Concurrency / retry | `FOR UPDATE` plus version checks; UNKNOWN never blindly retried. | Slice 12/14 tests |
| Compensating actions | **None implemented.** No reversal, refund or correction command exists. | §1 |
| Customer / invoice attribution | Owning seller = invoice seller and issuer (row scope). Phone is never a key. | Slice 12 matrix `SubmitBatch` |

## 3. Named blockers

Status of each decision as of `b3bf31b`. **No decision is approved.**

| ID | Question | Effect on financial cutover |
|---|---|---|
| ADR-01 (candidate) | Invoice issuance vs case return mutual safety (INV-009) | Blocks invoice issue, claim and return. Billing and custody cannot leave legacy. |
| D-25 | Gateway auto-approval policy | Blocks gateway activation. Legacy auto-approves without Finance review. |
| D-28 | Assessment invoice partial-payment policy | Blocks stage/amount rules beyond 2.0.160 full payment. |
| D-33 | Operations upgrade-payment authority | Blocks the operations upgrade payment. |
| D-36 | Pro-rata commission split across components | Blocks bundle commission. |
| D-41 | Ledger basis (accrual vs cash) | Blocks any posting. |
| D-43 | Finance approval while receipts are not READY | Blocks info-mode submission. Approval stays gated to READY. |
| OPD-04 | Refund object, proof and authority | Blocks refund execution and confirmation. |
| OPD-06 | Units, rounding, 0.5 boundary | Blocks normalisation and borderline completion. Exact comparison only. |
| OPD-08 | Maker/checker thresholds and exceptions | Blocks approve, reject, bulk and posting. |
| OPD-09 | Entitlement engine routing | Blocks commission / entitlement posting. |

Also relevant:
- D-26: customer auto-login after gateway payment.
- D-37: installment credit.
- D-38: Biawin wallet charge.
- D-40: payment reward wheel.
- D-44: evidence retention.
- F01: return/reset race contract.

## 4. Assessment

| Dimension | Classification |
|---|---|
| Manual payment pilot (shadow) | **NO-GO**. SHADOW mode is BLOCKED for payments by design, CONTROLLED needs write-back plus finaliser (U6/U7), and no authenticated users exist (U1). |
| Finance review, posting, refund, settlement | **NO-GO**. Not implemented; decisions are open. |
| Gateway | **NO-GO**. Synthetic only; D-25/D-26. |
| Financial reconciliation | **NOT VERIFIED**. No real data. |

**Overall financial readiness: NO-GO.**
