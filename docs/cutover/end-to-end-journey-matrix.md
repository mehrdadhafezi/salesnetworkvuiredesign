# Slice 15 — End-to-end business journey matrix

**Real-account acceptance.** No real-account acceptance test exists for any journey. Production
authentication does not exist (U1), and HR and Finance runtime are NOT LIVE VERIFIED (E0). A green
automated test below means the target logic passed on synthetic data in local PostgreSQL. It is **not**
operational acceptance.

**Support levels:**

| Level | Meaning |
|---|---|
| E2E | Supported end to end in the target. |
| PARTIAL | Some steps are implemented. |
| READ | Shadow read only. |
| NONE | Not implemented. |

| # | Journey | Target support | Automated tests (synthetic) | Real-account acceptance | Unresolved dependencies | Financial / historical risk | Fail-closed behaviour |
|---|---|---|---|---|---|---|---|
| 1 | Seller login and authorization | **NONE** (authorization evaluator only) | evaluator tests; `api_e2e_test.go` with the dev stub | none | Auth design, D-34, D-42 | — | Anonymous → UNAUTHORIZED everywhere (VERIFIED) |
| 2 | Customer creation and identity resolution | READ (resolution from legacy evidence; no create) | `customers_e2e_test.go` | none | D-30, D-29/D-31, OPD-01 | Wrong merges | Phone never RESOLVED alone; RELATED/CONFLICT queue |
| 3 | Case assignment | READ (custody events). Allocate is BLOCKED. | `sales_e2e_test.go` | none | OPD-10, SD-01 | ownership history | No allocate route |
| 4 | Case transfer and return safety | NONE (Transfer/Return BLOCKED) | — | none | ADR-01, F01, OPD-03 | **P0: return/invoice race (INV-009)** | No route |
| 5 | Pre-invoice and invoice issuance | NONE (IssueInvoice BLOCKED; PRE_INVOICE outcome BLOCKED) | contact-outcome tests assert the PRE_INVOICE block | none | ADR-01 | invoice references | BLOCKED |
| 6 | Manual payment submission | PARTIAL (upload + receipt-mode batch, CONTROLLED only) | `payments_writes_e2e_test.go`, `payments_upload_lifecycle_e2e_test.go`, `payments_http_e2e_test.go` | none | U1, U6, U7, D-43, OPD-06, OPD-10 | **Payment divergence if enabled without write-back** | Flag OFF → BLOCKED; SHADOW → BLOCKED |
| 7 | Receipt finalisation | PARTIAL (service only; no process) | `payments_upload_lifecycle_e2e_test.go` | none | U6, storage, D-44 | Receipts stuck QUARANTINED | NOT_RECOVERABLE + reconciliation, never READY without bytes |
| 8 | Finance review | READ queue plus evaluator | `finance_http_e2e_test.go` | none | OPD-08, D-41, D-43, OPD-09 | — | No approve/reject route |
| 9 | Verified stage advancement | NONE (IssueNextStage BLOCKED; billing lifecycle UNKNOWN) | — | none | D-32, billing lifecycle | stage totals | BLOCKED |
| 10 | Commission eligibility and calculation | READ (shadow runs, parity) | `commission_e2e_test.go`, `commission_http_e2e_test.go` | none | OPD-09, D-36 | historical recipients | Missing input → BLOCKED, never zero |
| 11 | Commission posting | NONE | — | none | OPD-08, OPD-09 | double posting across engines | Posting request never dispatched |
| 12 | Customer complaint | **NONE** (no after-sales module; `sn_after_sales_*` has no spec) | — | none | OPD-04 | complaint history absent | Complaint route never enters Biawin (Slice 14 test 14) |
| 13 | Refund request and execution | NONE | — | none | OPD-04, INV-015 | **money** | No command |
| 14 | No services → archive | PARTIAL: archive is recognised as a branch (`NOT_APPLICABLE`); no archive command | `TestActivationBranchesAndBlocks` | none | OPD-03 (revival) | archive ≠ deletion | No deletion or release path |
| 15 | Additional services → Biawin | PARTIAL: SHADOW routing decision | `TestActivationFanOut`, `TestActivationBranchesAndBlocks` | none | D-35a–e, no producer | — | Unknown route/lineage → BLOCKED |
| 16 | Subscription activation | PARTIAL: SHADOW only; production BLOCKED | same | none | D-35 | entitlement creation | CONTROLLED → BLOCKED D-35 (DB CHECK) |
| 17 | Per-service fulfillment | PARTIAL: target-native SHADOW cases | `TestFulfillmentCommands`, `TestOperationsHTTP` | none | empty catalog, relay | — | Out-of-scope → UNAUTHORIZED / 404 |
| 18 | Operational service completion | PARTIAL: completion with evidence; provider-integrated completion needs CONFIRMED | `TestCompletionGuards`, `TestCallback*` | none | D-35/D-38, connector | false delivery | UNKNOWN never shown as delivered |
| 19 | HR transfer / termination affecting open work | PARTIAL: HR requests; no final apply or handover | `writes_e2e_test.go` (HR requests) | none | OPD-05, SD-04, INV-030 | **ownerless work (P0 impact)** | Final apply not implemented |
| 20 | MIS and financial reports | READ (projections, shadow ledger) | `reporting_e2e_test.go`, `finance_e2e_test.go` | none | OPD-07, D-23, D-24, D-45, D-41, sources missing | report trust | Missing data → UNKNOWN / PARTIAL, never zero |

**Result.** No journey is supported end to end in the target. The core revenue journeys (4, 5, 9, 11,
13) are not implemented, and the payment journeys (6, 7) cannot run safely without workers and
write-back.
