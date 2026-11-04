# Slice 15 — Decision approval register

**Sources:**
- `docs/implementation/CRM-STANDALONE-TARGET-ARCHITECTURE-V1.md` §32 (D-23…D-45, ADR-01)
- `docs/source-of-truth/CRM-CROSS-ROLE-ARCHITECTURE-V1.md` §19 (OPD-01…10, SD-04, F01)
- `docs/implementation/E0-governance-baseline-freeze.md` §7

**This register approves nothing.** It opens no new business question: every entry already exists in
those sources. The only new items are engineering prerequisites, which appear in §3 and are marked as
such. Options are listed without selecting one.

**Status of every entry: OPEN.** The only approved records are D-01, D-02 and the Slice 0 money
contract (E0); D-27 is closed by reconciliation. D-03…D-22 are not recorded on `main` and carry no
authority (E0 rule 1).

**"Blocks" column key:** dev = development, merge, act = activation, prod = production cutover.

## 1. Register

| ID | Exact business question | Modules | Blocks | Safe current behaviour | Required evidence | Options (none selected) | Owner | Approval artefact |
|---|---|---|---|---|---|---|---|---|
| ADR-01 (candidate) | Accept the reservation protocol for INV-009 (invoice issue vs case return), or approve a specific multi-module transaction ADR? | cases, custody, billing | act, prod | No invoice-issue, claim or return writes | Race analysis, union-link proof (F01) | (a) reservation protocol; (b) multi-module transaction ADR | Human Owner + Engineering | Signed ADR on main |
| F01 | What is the all-link protected return/reset/race contract? | all inventory writers | act, prod | No return/reset/release | Union-link resolution + commit-conflict proof | per Cross-Role §5 | Sales + MIS + Finance + Engineering | Contract record |
| D-03…D-22 | Content not recorded on main | various | prod (any dependent phase) | Nothing dependent implemented | Commit each decision | — | Human Owner | Decision records on main |
| D-23 | Invoice Review visibility, PII and export policy | reporting | act (export), prod | No export; audited read | Privacy review | scoped export / no export | Product + Finance | Policy record |
| D-24 | Who owns the Invoice Review product/design? | reporting UI | prod (UI) | No target UI | — | — | Product | Owner record |
| D-25 | Gateway auto-approval policy | payments, finance | act, prod | `GATEWAY_AUTO` provenance, never Finance-reviewed | Finance policy | keep auto / require review | Finance | Policy record |
| D-26 | Customer auto-login after gateway callback | identity, payments | act, prod | Not replicated | Security review | replicate with controls / drop | Product + Security | Decision record |
| D-28 | Assessment invoice partial-payment policy | billing, cases | act, prod | 2.0.160 full-payment rule | — | full / partial | Product + Finance | Decision record |
| D-29 / D-31 | Seller duplicate-phone intent; phone as dedupe | sales, customers | act, prod | Discovery only; no dedupe | — | — | Product + Sales | Decision record |
| D-30 | Customer merge authority and evidence threshold | customers | act, prod | No merges | Resolution report on real data | — | Product + MIS + Sales | Decision record |
| D-32 | Hierarchy-based next-stage authority | billing, authorization | act, prod | Owning seller + admin (2.0.160) | billing lifecycle contract | — | Product + Sales + Finance | Decision record |
| D-33 | Operations upgrade-payment authority | payments, fulfillment | act, prod | Not granted | — | — | Product + Finance | Decision record |
| D-34 | HR initial-password policy | identity | act, prod | Random password + reset (proposed) | Auth design | — | Security + HR | Decision record |
| D-35a–e | Subscription+ model: cash-credit semantics, installment credit, expiry/redemption, Biawin as redemption system of record, mandatory conversion | catalog, biawin, finance | act, prod | SHADOW only; financial components BLOCKED | Business model definition | per Arch §14 | Product + Finance | Decision record per sub-item |
| D-36 | Pro-rata commission split across bundle components | commission | act, prod | No component allocation | Historical parity | pro-rata / per-component policy | Finance + Product | Decision record |
| D-37 | Installment credit: financing product or entitlement? | finance | act, prod | Entitlement account, provisional | — | — | Finance | Decision record |
| D-38 | Biawin wallet charge semantics and system of record | biawin, finance | act, prod | No target charge; legacy is master (DB CHECK) | Provider contract | — | Finance + Product | Decision record |
| D-39 | Catalog `all_except` semantics | catalog, authorization | act | `selected`/`all` only | — | — | Sales + Product | Decision record |
| D-40 | Payment reward wheel: keep or retire? | sales, finance | prod | Read-only import (no spec yet) | — | keep / retire | Product | Decision record |
| D-41 | Ledger basis: accrual vs cash | finance | act, prod | Shadow computes both, publishes neither | Real-data parity | accrual / cash | Finance | Decision record |
| D-42 | Can HR edit sales capabilities? | organization, authorization | act, prod | HR cannot grant; identity owns grants | — | — | Product + HR + Sales | Decision record |
| D-43 | Finance approval while receipts are not READY | payments, finance | act | Approval blocked unless READY | — | — | Finance | Decision record |
| D-44 | Evidence retention; unbound expiry; legal retention | payments | act, prod | Bound kept indefinitely; unbound 7 days | Legal review | — | Finance + Legal | Policy record |
| D-45 | Behaviour when a sensitive-read audit write fails | reporting, audit | prod | Fail closed for PII views | — | — | Product + Security | Decision record |
| OPD-01 | Ambiguous lineage, aliases, case coverage | all | prod | Unknown kept unknown | Real-data coverage | per §19 | per §19 | Decision record |
| OPD-02 | Sensitive ACK/SLA/escalation owner and routing | cross-domain | act | No ACK/SLA fields | — | — | Product | Decision record |
| OPD-03 | Cancelled release / reset / archive revival authority | sales, MIS, finance | act, prod | No release/reset/revival | — | — | Sales + MIS + Finance | Decision record |
| OPD-04 | Refund object, proof and authority | finance, sales, complaint | act, prod | No refund execution | Bank/source proof policy | — | Product + Finance | Decision record |
| OPD-05 | Termination/transfer handover and access timing | HR, sales, finance | act, prod (**P0 impact**) | No final apply | Open-work impact + named successor policy | — | Product + HR + Sales + Finance | Decision record |
| OPD-06 | Units, rounding, 0.5 boundary | financial, reports | act, prod | Exact integer, one unit, no conversion | Source-unit proof | — | Product + Finance | Money contract extension |
| OPD-07 | Cohort, window, rate and attribution for KPIs | sales, MIS, reports | prod | Metrics marked NOT FINAL | Cohort metadata | — | per §19 | Decision record |
| OPD-08 | Maker/checker thresholds and emergency exceptions | finance (MIS when applicable) | act, prod | Submitter ≠ reviewer; no approval executes | Org policy | — | Product + Finance | Policy record |
| OPD-09 | Entitlement engine routing, non-invoice keys, correction | finance, HR, sales | act, prod | No posting; shadow only | Active routes + key proof | — | Product + Finance | Decision record |
| OPD-10 | Skip-level, personal receiving, invoice assist/delegation | sales tiers, MIS, finance | act, prod | Current paths preserved; no new grants | Per-action guards | — | Product + Sales + Finance + HR | Decision record |
| SD-04 | HR cross-scope transfer destinations / no-op policy | HR, sales | act | Not implemented | — | — | Product + HR + Sales | Decision record |
| ADR #18 (E0) | Staging and anonymisation policy | integration | prod (real import) | Synthetic-only load | Privacy review | — | Product + Security | Policy record |

## 2. Management approval packages

Sensitive financial and security decisions are kept in separate packages, as required.

| Package | Contents | Notes |
|---|---|---|
| **P-SEC: Security and identity** | Auth design (engineering prerequisite E-1), D-34, D-26, D-42, D-45, ADR #18 | Security owner signs separately. |
| **P-FIN-A: Payment approval** | OPD-08, D-43, D-25, D-44 | Finance and Legal. |
| **P-FIN-B: Ledger and money** | D-41, OPD-06, D-37 | Finance only. |
| **P-FIN-C: Entitlements and commission** | OPD-09, D-36 | Finance + Product. |
| **P-FIN-D: Refunds** | OPD-04 | Finance + Product. Kept independent. |
| **P-SALES: Inventory safety** | ADR-01, F01, OPD-03, OPD-10, D-32, D-28, D-29/31, D-30 | Sales + MIS + Finance + Engineering. |
| **P-HR: Workforce** | OPD-05, SD-04 | HR + Sales + Finance. |
| **P-BIAWIN: Services** | D-35a–e, D-38, D-33, D-39 | Product + Finance. D-38 also needs P-FIN-B. |
| **P-REPORT: Reporting** | OPD-07, D-23, D-24, OPD-01 | Product + Finance + MIS. |
| **P-GOV: Governance** | Record D-03…D-22 on main, or declare them void | Human Owner. |

## 3. Engineering prerequisites

These are not business decisions. They are required before any authority switch.

| ID | Prerequisite | Evidence of closure |
|---|---|---|
| E-1 | Production authentication and session (after P-SEC) | Security review + real-account tests |
| E-2 | Worker processes: finaliser, expiry, relay, write-back, provider reconcile | Commands + scheduler + backlog alerts |
| E-3 | Production legacy write-back connector | Connector against a legacy staging copy; UNKNOWN reconciliation rehearsal |
| E-4 | Real-snapshot extraction, load and reconciliation (after ADR #18) | Filled data-reconciliation register |
| E-5 | Extract specs for the 55 uncovered tables, or explicit owner exclusion | Spec golden files |
| E-6 | Deployment, backup, monitoring and DR definitions | Infrastructure checklist column B signed |
| E-7 | Flag-change and grant-change operator procedure with audit | Procedure + audit table or log |
