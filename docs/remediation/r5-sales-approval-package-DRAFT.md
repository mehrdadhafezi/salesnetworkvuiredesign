# R5 — P-SALES approval package (DRAFT — NOT APPROVED)

**Scope:** inventory and invoice authority: ADR-01, F01, OPD-03, OPD-10, D-32, D-28, D-29 / D-31, D-30.
**Blocker:** P0-5 (PARTIAL; see `r4-decision-gates.md`). It stays PARTIAL until human decisions,
implementation and runtime evidence exist.

All decision records below are **DRAFT — NOT APPROVED**. The approval fields (approval date, named
approvers, selected option, conditions, evidence of approval, effective date) are intentionally empty:
no valid approval evidence exists in the repository (EV-040). Code behaviour, passing tests and merged
PRs are never treated as management authorization. A recommendation, where given, is an engineering
view and is conditional; the choice belongs to the named owners.

**Invariants that no option may weaken** (frozen; Gate-0 / Cross-Role / AGENTS.md):
- Phone is discovery only, never canonical identity or a merge key (D-29 / D-31 / D-30).
- Original seller, credit owner, current custody, next actor and event actor are never collapsed or
  rewritten by any decision here.
- The union of all valid links decides financial dependency (Cross-Role §8); ambiguous lineage stays
  UNKNOWN.
- Cancelled unpaid / paid / refund-reported cases are never released automatically.

**The R4 race harness is technical evidence, not approval** (EV-016).

## ADR-01 option comparison

| Aspect | A — Reservation / claim protocol | B — Explicit multi-module transaction | C — Keep blocked |
|---|---|---|---|
| Benefits | Keeps single-writer module ownership and owner-scoped logins; outbox events; R4 harness evidence | Simplest atomicity for one path | No new risk |
| Risks | Expired claims need a reconciliation owner; late confirmations | Breaks the owner-scoped transaction rule; wide locks; coupled failure | P0-5 stays open; billing / custody stay on legacy |
| Implementation impact | Claim, issue, confirm, return commands; claim table in cases; unique per claim in billing | New cross-schema transaction path; changes to `db.Runner` role model | None |
| Concurrency | Case row lock serialises claim vs return; unique(claim) prevents double issue (R4 #1–#3 with forced interleaving) | Single transaction; deadlock risk across schemas | n/a |
| Rollback / recovery limits | No automatic release; expired claim → reconciliation; issued invoice never deleted | Rolled back as a unit, but a later-found error in one module needs compensation | n/a |
| Missing test evidence | Real-data links, legacy coexistence, TTL tuning, runtime | No harness exists for B | — |
| Required approvers | Human Owner + Engineering (Sales, Finance consulted) | same | same |

**No option is selected.**

### ADR-01 — Invoice issuance vs case return mutual safety (INV-009)

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | ADR-01 |
| Title | Invoice issuance vs case return mutual safety (INV-009) |
| Business question | Accept the candidate reservation (claim) protocol for INV-009, or approve a specific multi-module transaction ADR, or keep issue / claim / return blocked until more evidence exists? |
| Background | Invoice issuance (billing) and ordinary return (custody) must never both commit from the same 'unused' assumption (Gate-0 INV-009; Cross-Role §8). The target forbids cross-module transactions unless an ADR approves one. |
| Current behavior | No ClaimCaseForInvoice / IssueInvoice / ReturnCase command or route exists (Slice 11/12). R4 gates answer BLOCKED (ADR-01). Legacy 2.0.160 remains the only writer. |
| Evidence references | EV-011, EV-016, EV-017, EV-027, EV-042 (see `docs/governance/r5-evidence-index.json`) |
| Source | Architecture V1 §6 (candidate), §32; register §1 |
| Authority status | OPEN (CANDIDATE, not accepted) |
| Missing evidence | Real-data link coverage; legacy-coexistence behaviour (legacy writers still run, P1-8); claim TTL and reconciliation-owner policy; runtime evidence. |
| Alternatives | A — Candidate reservation / claim protocol: cases claim (own tx, row lock + version) → billing issue against an active claim (own tx, unique per claim) → InvoiceIssued confirms the claim; expiry → reconciliation, never silent release.<br>B — Explicit multi-module transaction: one database transaction spanning cases, custody and billing writes (requires relaxing the single-writer / owner-scoped transaction rule for this path).<br>C — Keep blocked until additional evidence (real-data link coverage, legacy coexistence rehearsal) exists. |
| Trade-offs | A: keeps module ownership and the outbox model; needs TTL, reconciliation owner and late-confirmation handling; evidenced by the R4 synthetic harness (15 scenarios + mutation check). Concurrency: row lock on the case serialises claim vs return; unique(claim) prevents double issuance. Rollback: a claim that expires goes to reconciliation (no automatic release); an issued invoice is never deleted. Missing tests: real data, legacy coexistence, TTL tuning. B: simplest atomicity; breaks the owner-scoped transaction rule and the per-module role separation (db.Runner refuses shared logins); larger lock scope; harder recovery when one module's write is later found invalid; no harness evidence exists for B. C: zero new risk; P0-5 stays open; billing / custody cannot leave legacy. |
| Recommended option, if defensible | Conditional engineering view: A is the only option with existing technical evidence and it preserves the frozen module boundaries. This is NOT a selection. |
| Why the recommendation is conditional | The R4 harness is synthetic; it proves the candidate protocol's database mechanics only. Legacy coexistence, real-data link coverage and the TTL / reconciliation-owner policy are business inputs. The Human Owner and Engineering must choose. |
| Risks and compensating controls | A: an unreconciled expired claim blocks the case (compensating control: named reconciliation owner + alert). B: cross-module coupling, wider blast radius (control: none available without an ADR). C: business continues on legacy only. |
| Dependent modules | cases, custody, billing |
| Business owner | Human Owner |
| Required approver roles | Human Owner + Engineering lead; Sales + Finance consulted |
| Dependencies | F01 (union-link contract), OPD-03 (release), OPD-10 (who may claim), D-32 (next-stage authority), P-SEC person identity (actor verification) |
| Affected blocker | P0-5 |
| Readiness | READY FOR HUMAN REVIEW |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Claim / issue / return commands in owner transactions; owner-schema migrations; outbox / inbox wiring; legacy write-back; real-data parity. |
| Activation gate | Separate activation authority per command + legacy writer disabled for the same command (P1-8) + runtime evidence on real-shaped data. |

### F01 — All-link protected return / reset / race contract

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | F01 |
| Title | All-link protected return / reset / race contract |
| Business question | What is the all-link protected return / reset / race contract (union-link resolution + commit-conflict proof) that every inventory writer must obey? |
| Background | Cross-Role §8 defines the dependency table; F01 makes it the single shared obligation (SG-08, M-G01, SD-G01, MIS-G01, FIN-LINK). |
| Current behavior | No return / reset / release. R4 implements the union classifier (cases/api ClassifyDependencies) and the release gate; both fail closed. |
| Evidence references | EV-009, EV-010, EV-016, EV-017 (see `docs/governance/r5-evidence-index.json`) |
| Source | Cross-Role V1 §19 (F01), §8, §21 |
| Authority status | OPEN (contract READY; rollout BLOCKED; runtime validation REQUIRED) |
| Missing evidence | Link adapters that produce every §8 proof source from staged real data; per-source coverage statements; reconciliation-owner names. |
| Alternatives | Accept the §8 union contract as implemented by the R4 classifier (fail closed on any unresolved link).<br>Accept with additional per-source coverage thresholds (business sets which sources are mandatory).<br>Defer until link adapters produce real-data coverage reports. |
| Trade-offs | Option 1 is the frozen contract made executable; its weakness is coverage, not logic. Option 2 needs a business definition of mandatory sources. Option 3 delays P0-5. |
| Recommended option, if defensible | Conditional: the contract text already exists (frozen §8); what is missing is real-data coverage evidence, so a decision on mandatory sources should follow a coverage report. |
| Why the recommendation is conditional | Choosing mandatory sources without seeing coverage would be speculation. |
| Risks and compensating controls | False 'no dependency' on incomplete coverage (control: coverage incomplete → FAIL_CLOSED, already enforced). |
| Dependent modules | cases, custody, billing, payments, MIS |
| Business owner | Sales + MIS + Finance + Engineering |
| Required approver roles | Sales + MIS + Finance owners + Engineering |
| Dependencies | ADR #18 (real data), E-4 (real snapshot), OPD-01 (lineage) |
| Affected blocker | P0-5 |
| Readiness | NEEDS FURTHER EVIDENCE BEFORE A DECISION |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Link adapters per proof source; coverage report; reconciliation-owner routing. |
| Activation gate | Per inventory writer, after ADR-01 and OPD-03. |

### OPD-03 — Cancelled release / reset / archive revival authority

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | OPD-03 |
| Title | Cancelled release / reset / archive revival authority |
| Business question | Who may release, reset or revive a cancelled (unpaid / paid / refund-reported) or archived case, and under which dependency and history conditions? |
| Background | §8: cancelled unpaid / paid / refund-reported → automatic release forbidden; OPD-03 / OPD-04 + reconciliation required. |
| Current behavior | No release / reset / revival. R4 classifier returns BLOCKED_CANCELLED_NEEDS_POLICY; release gate names OPD-03. |
| Evidence references | EV-008, EV-009, EV-017 (see `docs/governance/r5-evidence-index.json`) |
| Source | Cross-Role V1 §19 (OPD-03; alias M-G09 part), §8 |
| Authority status | OPEN |
| Missing evidence | Reason-specific dependency and history policy; volume of cancelled cases in real data. |
| Alternatives | Release only by a named role after reconciliation evidence (per cancellation reason).<br>Never release cancelled-paid / refund-reported cases; release cancelled-unpaid after a cooling period with reconciliation.<br>Keep all cancelled cases locked (status quo). |
| Trade-offs | Each option trades inventory reuse against financial-claim safety; none can be selected by engineering. |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | cases, custody, billing, MIS |
| Business owner | Product + Sales + MIS + Finance (Cross-Role §19) |
| Required approver roles | Product, Sales, MIS and Finance owners |
| Dependencies | F01, OPD-04 (refund-reported), ADR-01 |
| Affected blocker | P0-5 |
| Readiness | READY FOR HUMAN REVIEW |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Release / reset / revival commands using the release gate; reason taxonomy. |
| Activation gate | After ADR-01 + F01 + OPD-03; per command. |

### OPD-10 — Skip-level, personal receiving, invoice assist / delegation

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | OPD-10 |
| Title | Skip-level, personal receiving, invoice assist / delegation |
| Business question | Which skip-level, personal-receiving and invoice-assist / delegation paths exist in the target, with which per-action guards and approvals? |
| Background | Current legacy paths are preserved; silent removal or expansion is forbidden. Assisted issuance and delegation would let a second account act for a seller. |
| Current behavior | Target gates require actor = holder / claimer; no rank shortcut (lint s11-rank-shortcut); no new grants. |
| Evidence references | EV-008, EV-017, EV-027 (see `docs/governance/r5-evidence-index.json`) |
| Source | Cross-Role V1 §19 (OPD-10; aliases SD-01, SD-02, SD-03, SG-03, SG-04, SG-07, M-G08, SD-G05) |
| Authority status | OPEN |
| Missing evidence | Per-action inventory of existing legacy guards; business approval per path. |
| Alternatives | Preserve today's legacy paths exactly, with explicit per-action guards and audit.<br>Allow assisted issuance only with a recorded delegation (delegator, delegate, scope, expiry) and attribution unchanged.<br>Disallow skip-level and assist in the target (legacy-only until decided). |
| Trade-offs | Delegation requires the P-SEC person-identity model to prevent one human acting as two; attribution must stay with the original seller. |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | sales tiers, MIS delivery, billing, authorization |
| Business owner | Product + Sales + Finance + HR (+ MIS source paths) |
| Required approver roles | Product, Sales, Finance, HR owners |
| Dependencies | P-SEC person identity, D-42 (capability grants), ADR-01 |
| Affected blocker | P0-5 |
| Readiness | READY FOR HUMAN REVIEW |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Delegation record + guards; attribution preservation tests. |
| Activation gate | Per path. |

### D-32 — Hierarchy-based next-stage authority

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | D-32 |
| Title | Hierarchy-based next-stage authority |
| Business question | Who may issue the next payment stage of an invoice (hierarchy-based authority vs the 2.0.160 rule: owning seller + evidenced admin)? |
| Background | 2.0.160: converter or admin (`ajax_converter_issue_payment_stage`). Billing has no stage state machine in the target (lifecycle UNKNOWN). |
| Current behavior | billing.IssueNextStage BLOCKED. |
| Evidence references | EV-007, EV-026 (see `docs/governance/r5-evidence-index.json`) |
| Source | Architecture V1 §32; Slice 12 matrix §2 |
| Authority status | OPEN (safe default available) |
| Missing evidence | Billing lifecycle contract (stage state machine). |
| Alternatives | Keep the 2.0.160 rule (owning seller + admin).<br>Add hierarchy-based authority (supervisor chain) with audit.<br>Finance-only next stage. |
| Trade-offs | Hierarchy authority expands write scope (D-42 / P-SEC); Finance-only changes the sales workflow. |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | Cannot be implemented until the billing lifecycle contract exists; the authority decision can still be reviewed. |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | billing, authorization |
| Business owner | Product + Sales + Finance |
| Required approver roles | Product, Sales and Finance owners |
| Dependencies | Billing lifecycle contract (engineering), D-42, P-SEC |
| Affected blocker | P0-5 (billing authority) |
| Readiness | NEEDS FURTHER EVIDENCE BEFORE A DECISION |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Stage state machine + command. |
| Activation gate | After lifecycle contract + decision. |

### D-28 — Assessment invoice partial-payment policy

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | D-28 |
| Title | Assessment invoice partial-payment policy |
| Business question | Do assessment invoices require full payment (2.0.160 rule) or may they be partially paid? |
| Background | Architecture §32 says 2.0.176 removed the full-payment rule; the repository holds only 2.0.160 (conflict C-05). |
| Current behavior | 2.0.160 full-payment rule preserved; R4 issuance gate names D-28 for a partial assessment. |
| Evidence references | EV-007, EV-032, EV-017 (see `docs/governance/r5-evidence-index.json`) |
| Source | Architecture V1 §32 (cites plugin 2.0.176 removing the rule); Slice 12 matrix §2 |
| Authority status | OPEN (safe default available) |
| Missing evidence | The 2.0.176 source / changelog cited by Architecture §32 (not in the repository). |
| Alternatives | Keep full payment (2.0.160).<br>Allow partial payment (as reported for 2.0.176).<br>Allow partial with a minimum first instalment. |
| Trade-offs | Partial payment changes stage and commission timing (OPD-09, D-41). |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | The 2.0.176 evidence must be provided before the decision so the business choice is made against the actual legacy behaviour. |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | billing, cases, commission |
| Business owner | Product + Finance |
| Required approver roles | Product and Finance owners |
| Dependencies | OPD-09, D-41 |
| Affected blocker | P0-5 / P0-7 |
| Readiness | NEEDS FURTHER EVIDENCE BEFORE A DECISION |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Plan rule at issuance. |
| Activation gate | With issuance. |

### D-29 / D-31 — Seller duplicate-phone intent; phone as dedupe

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | D-29 / D-31 |
| Title | Seller duplicate-phone intent; phone as dedupe |
| Business question | What is the seller's business intent when marking a duplicate (phone-based), and may phone serve as any kind of dedupe? |
| Background | AGENTS.md and Gate-0: phone is discovery only, never canonical identity or financial join proof. |
| Current behavior | Contact outcome DUPLICATE is BLOCKED (Slice 11); phone is discovery only; similarity links are never proof (R4). |
| Evidence references | EV-038, EV-027, EV-017 (see `docs/governance/r5-evidence-index.json`) |
| Source | Architecture V1 §32 (D-29, D-31 alias of D-29) |
| Authority status | OPEN |
| Missing evidence | Business description of the duplicate workflow; real-data duplicate volumes. |
| Alternatives | Duplicate = a discovery hint routed to MIS review (no merge, no ownership change).<br>Duplicate = close the case with reason DUPLICATE, history preserved, no customer merge.<br>Keep DUPLICATE blocked. |
| Trade-offs | No option may merge customers by phone (D-30 governs merges). |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | cases, customers, MIS |
| Business owner | Product + Sales |
| Required approver roles | Product and Sales owners |
| Dependencies | D-30 |
| Affected blocker | P1 (sales workflow) |
| Readiness | READY FOR HUMAN REVIEW |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Duplicate outcome command. |
| Activation gate | Per command. |

### D-30 — Customer merge authority and evidence threshold

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | D-30 |
| Title | Customer merge authority and evidence threshold |
| Business question | Who may merge customer records, and what evidence threshold is required (phone alone never suffices)? |
| Background | Slice 5 customers module is shadow; merges produce RELATED / CONFLICT only. |
| Current behavior | No merges (lint merge-path forbids merge identifiers). |
| Evidence references | EV-007, EV-038 (see `docs/governance/r5-evidence-index.json`) |
| Source | Architecture V1 §32 |
| Authority status | OPEN |
| Missing evidence | Resolution report on real data (false-merge rate). |
| Alternatives | No merges; RELATED links only.<br>Merge by MIS with documented multi-factor evidence and reversible links.<br>Merge by named role with independent review. |
| Trade-offs | Any merge needs reversibility and history preservation; real-data resolution evidence is missing. |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | A threshold chosen without the real-data resolution report would be guesswork. |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | customers, MIS, sales |
| Business owner | Product + MIS + Sales |
| Required approver roles | Product, MIS and Sales owners |
| Dependencies | ADR #18, E-4 |
| Affected blocker | P1 |
| Readiness | NEEDS FURTHER EVIDENCE BEFORE A DECISION |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Reversible merge records. |
| Activation gate | After real-data report. |
