# Slice 15 — Go/No-Go report

| Item | Value |
|---|---|
| Baseline | `b3bf31b5979edfa1ba9a5fb6a45521d21e7712c4` (main after PR #39) |
| Assessment date | 2026-11-04 |
| Assessor | Claude (audit only; approves nothing) |
| **Overall recommendation** | **NO-GO for Production** |

**NO PRODUCTION CUTOVER WITHOUT EXPLICIT HUMAN GO/NO-GO APPROVAL.**

## 1. Approval model

| Grade | Meaning |
|---|---|
| **GO** | Evidence is sufficient for a precisely bounded module/step. |
| **CONDITIONAL GO** | Only for non-critical prerequisites with approved compensating controls. Never used when a critical financial or identity risk is unresolved. |
| **NO-GO** | A proven blocking defect or a missing mandatory decision. |
| **NOT VERIFIED** | Evidence is missing. Cannot be treated as GO. |
| **NOT APPLICABLE** | Justified exclusion. |

A Go decision must name:
- the module and step
- the environment
- the evidence links
- the approver(s) per [decision-approval-register.md](decision-approval-register.md) §2
- the rollback owner

Missing evidence is never converted into approval.

## 2. Dimension assessment (whole system)

| Dimension | Grade | Basis |
|---|---|---|
| Technical | **NO-GO** | No production authentication (U1); workers not deployable (U6); no legacy write-back connector (U7). Core revenue commands are not implemented (invoice issue/return, approve/reject, posting, refund). |
| Security | **NO-GO** | No auth, session or cookie model. Secrets handling, TLS and SIEM are NOT VERIFIED. D-26/D-34/D-42/D-45 open. |
| Data | **NO-GO** | Zero real-data reconciliation (U4). 55 of 89 legacy tables have no extract spec. The staging/anonymisation policy (ADR #18) is not decided. |
| Financial | **NO-GO** | ADR-01, D-25, D-28, D-33, D-36, D-41, D-43, OPD-04, OPD-06, OPD-08 and OPD-09 are open. No compensation path exists. |
| Integration | **NO-GO** | No real Biawin connector, gateway adapter or legacy write-back connector. D-35/D-38 open. |
| Operational | **NOT VERIFIED** | No deployment, backup, monitoring, DR or RPO/RTO artefacts. No on-call model. |
| Business decisions | **NO-GO** | Every D/OPD/ADR entry in the register is open; D-03…D-22 are unrecorded. |
| Rollback | **NOT VERIFIED** | Flag reversal exists for new writes only. No compensation commands, no rehearsed restore, no worker metrics. |

## 3. Per-module grade (production authority)

| Module | Grade |
|---|---|
| Identity / authentication | NO-GO |
| Organization / HR / authorization | NO-GO |
| Customers / lineage | NO-GO |
| Sales / cases | NO-GO |
| Custody / distribution | NO-GO |
| Catalog | NOT VERIFIED |
| Billing / invoices | NO-GO |
| Payments / evidence | NO-GO |
| Finance / ledger | NO-GO |
| Commission | NO-GO |
| Reporting / MIS | NO-GO (authority) · NOT VERIFIED (shadow pilot) |
| Biawin subscriptions | NO-GO |
| Fulfillment / operations | NO-GO |
| External integrations | NO-GO |
| Customer portal | NO-GO (not implemented) |
| Workers / outbox-inbox | NO-GO |
| Observability / audit | NOT VERIFIED |

**Lowest-risk next step.** A read-only shadow pilot (runbook P1) could become a GO candidate once
P0-1, P0-3 and P0-4 below close and pilot grants are approved. It is not a GO today.

## 4. Critical blockers

### P0 — must close before any production pilot or authority change

| # | Blocker | Evidence | Closure evidence |
|---|---|---|---|
| P0-1 | No production authentication or session: every request is anonymous | `cmd/crm-api/writes.go:72-76`; `guard.go:19-22` | Approved auth design + implementation + security review + real-account tests |
| P0-2 | Write-back connector and background workers are not deployable. CONTROLLED writes would diverge from legacy. | `writeback/connector.go:13-16`; `readiness-evidence.sh static` | Worker commands, production connector, rehearsal on a legacy staging copy |
| P0-3 | No real-data reconciliation (INV-036 unmet) | data register §0 | Snapshot-based register filled with zero unexplained deltas |
| P0-4 | No deployment, backup, restore or monitoring definition | infra checklist #1, #4, #13 | Signed checklist column B |
| P0-5 | ADR-01 / F01 inventory safety undecided; invoice issue, claim and return not implemented | Slice 11/12 matrices | Approved ADR + implementation + race tests |
| P0-6 | OPD-05 termination accountability (P0 impact); HR final apply absent | Cross-Role §19 | Decision + implementation |
| P0-7 | Financial execution decisions open (OPD-08, OPD-09, D-41, OPD-04, OPD-06) and no approve/post/refund/compensation commands | financial-readiness §1 | P-FIN packages decided + implementation + parity |

### P1 — must close before the affected module's switch

| # | Blocker |
|---|---|
| P1-1 | 55 legacy tables have no extract spec (complaints, Dot, MIS, campaigns, seller flow, HR change requests, project subscriptions/routes, engine E4) |
| P1-2 | Biawin: no connector; D-35a–e, D-38 open; catalog empty; no activation producer |
| P1-3 | Customer portal and gateway not implemented (D-25, D-26) |
| P1-4 | No audited flag-change or grant-change procedure (FF-3, FF-4); D-42 |
| P1-5 | Object store production durability; D-44 retention |
| P1-6 | D-03…D-22 unrecorded on main (E0 rule 1) |
| P1-7 | Reporting decisions OPD-07, D-23, D-24, D-45; MIS sources missing |
| P1-8 | No legacy-side mechanism to disable a legacy writer during a switch |

### P2 — hygiene

| # | Item |
|---|---|
| P2-1 | Compiled 16 MB binary tracked at the repo root (`crm-biawin-shadow`) |
| P2-2 | CI runs on pull requests only; no post-merge or release pipeline |
| P2-3 | Slice 9 and Slice 10 command matrices exist only in PR descriptions, not in `docs/` |
| P2-4 | CI warns that the Node.js 20 actions runtime is deprecated |

## 5. Decision

**NO-GO.** Production is not ready. No module may change its system of record. Legacy 2.0.160 remains
authoritative for all business operations. Additional implementation work is required: E-1…E-7 in the
decision register, plus the decision-dependent commands.
