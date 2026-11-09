# R4 — Decision gate matrix

**No decision is approved by R4.** This matrix separates engineering completion from business-policy
approval.
- **"Further development safe"** means policy-independent engineering can continue without choosing the
  decision.
- **"Activation blocked"** is YES for every row.

Sources:
- `docs/cutover/decision-approval-register.md`;
- Cross-Role V1 §19;
- Architecture V1 §32.

| Decision | Exact unresolved question | Module | Existing safe behaviour | Approval owner | Implementation dependency | Tests available (R4 and earlier) | Missing runtime evidence | Further development safe? | Activation blocked? |
|---|---|---|---|---|---|---|---|---|---|
| **ADR-01** | Accept the reservation protocol for INV-009 (invoice issue vs case return), or approve a specific multi-module transaction ADR? | cases, custody, billing | No claim / issue / return command or route; gates answer BLOCKED (ADR-01) | Human Owner + Engineering | Claim / issue / return commands, owner-schema migrations, outbox / inbox, write-back | `claimgate`, `issuance`, `release` unit tests; `TestR4SalesRaceHarness` (15 scenarios + mutation check) | Legacy coexistence; real-data links; TTL and reconciliation-owner operation | YES (evaluators, link adapters, harness) | **YES** |
| **F01** | What is the all-link protected return / reset / race contract (union-link resolution + commit-conflict proof)? | all inventory writers | No return / reset / release; union classifier fails closed | Sales + MIS + Finance + Engineering | Link adapters for every §8 proof source from staged data | `ClassifyDependencies` tests; harness #2, #7–#10, #15 | Real-data union coverage per proof source | YES | **YES** |
| **OPD-03** | Who may release / reset / revive a cancelled or archived case, and under what conditions? | sales, MIS, finance | Cancelled → BLOCKED (never auto-released); reset / release gates name OPD-03 | Sales + MIS + Finance | Reset / release command semantics | Classifier CANCELLED; release gate; harness #10 | Policy record | YES (evaluation only) | **YES** |
| **OPD-04** | What is the refund object, proof and authority? | finance, sales, complaint | No refund execution or confirmation; state machine answers BLOCKED | Product + Finance | Refund commands, proof storage, posting (D-41) | `refund_test.go` (11 transitions, evidence rules, projection) | Bank / provider proof standard; real refunds | YES (contract) | **YES** |
| **OPD-06** | Units, rounding, the 0.5 boundary? | financial, reports | Exact int64 + explicit unit; no conversion or rounding; unrepresentable → BLOCKED | Product + Finance | Normalisation rules for posting / reports | Money kernel; posting validator (no float, overflow, cross-unit) | Source-unit proof on real data | YES | **YES** |
| **OPD-08** | Maker / checker thresholds and emergency exceptions? | finance (MIS when applicable) | Submitter ≠ reviewer (fail closed); no decision capability; contract BLOCKED | Product + Finance | Decision commands, capability, rejection vocabulary | `review_test.go`, `contract_test.go`, `TestR4FinanceDecisionContractOverReviewRead` | Org policy; person-level identity | YES (contract) | **YES** |
| **OPD-09** | Entitlement engine routing, non-invoice keys, correction? | finance, HR, sales | No posting; shadow only; non-invoice credits unkeyed | Product + Finance | Posting engine, key policy | `ReconcileEngines` tests; Slice 13 shadow tests | Active routes and key proof on real data | YES (reconciliation) | **YES** |
| **OPD-10** | Skip-level, personal receiving, invoice assist / delegation? | sales tiers, MIS, finance | Gates require actor = holder / claimer (no rank shortcut); no new grants | Product + Sales + Finance + HR | Per-action delegation guards | Gate UNAUTHORIZED tests; harness #3, #14 | Policy record | YES | **YES** |
| **D-25** | Gateway auto-approval policy? | payments, finance | `GATEWAY_AUTO` provenance, never Finance-reviewed; synthetic gateway only | Finance | Real gateway adapter + review policy | Slice 12 gateway tests | Real gateway | YES | **YES** |
| **D-28** | Assessment invoice partial-payment policy? | billing, cases | 2.0.160 full-payment rule kept; issuance gate names D-28 for a partial assessment | Product + Finance | Plan rules at issuance | `issuance_test.go` (D-28 named) | Policy record | YES | **YES** |
| **D-32** | Hierarchy-based next-stage authority? | billing, authorization | `IssueNextStage` BLOCKED (no stage state machine) | Product + Sales + Finance | Billing lifecycle contract | Slice 12 matrix | Lifecycle contract | YES | **YES** |
| **D-36** | Pro-rata commission split across bundle components? | commission | Component entitlements have no key (unkeyed) | Finance + Product | Component allocation policy | `ReconcileEngines` (unkeyed component) | Historical parity | YES | **YES** |
| **D-41** | Ledger basis: accrual vs cash? | finance | Shadow computes both and publishes neither; posting contract requires an explicit basis and BLOCKS both | Finance | Posting authority | `posting_test.go`, `posting_pg_test.go` | Real-data parity | YES (contract) | **YES** |
| **D-43** | Finance approval while receipts are not READY? | payments, finance | Approval blocked unless every receipt is READY | Finance | Decision commands | `review_test.go`, `contract_test.go` | Policy record | YES | **YES** |

## Engineering completion vs business approval

| Area | Engineering (R4) | Business approval |
|---|---|---|
| Inventory safety (P0-5) | Union classifier, three gates, race harness with mutation check | ADR-01, F01, OPD-03, OPD-10, D-28, D-32 **open** |
| Finance review (P0-7) | Decision contract over the real review read | OPD-08, D-43 **open**; capability and rejection vocabulary undefined |
| Ledger (P0-7) | Policy-independent posting contract, idempotency on PostgreSQL | D-41, OPD-06, OPD-08, OPD-09 **open** |
| Refunds (P0-7) | State machine contract and projection | OPD-04 **open** |
| Commission (P0-7) | Cross-engine reconciliation | OPD-09, D-36 **open** |

## Closure status

- **P0-5: PARTIAL.** Engineering foundations and race evidence exist. ADR-01 and F01 are unapproved,
  and no claim / issue / return command exists.
- **P0-7: PARTIAL / BLOCKED.**
  - Contracts and evaluators exist.
  - Every financial execution decision (OPD-08, OPD-09, D-41, OPD-04, OPD-06) is open.
  - No approve / post / refund / compensation command exists.
