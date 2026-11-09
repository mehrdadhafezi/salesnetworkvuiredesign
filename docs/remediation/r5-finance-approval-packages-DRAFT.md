# R5 — Finance approval packages P-FIN-A / B / C / D (DRAFT — NOT APPROVED)

**Blocker:** P0-7 (PARTIAL / BLOCKED). No financial approval, posting, refund, wallet credit, commission
payment or settlement command is authorized by this document.

All decision records below are **DRAFT — NOT APPROVED**. The approval fields (approval date, named
approvers, selected option, conditions, evidence of approval, effective date) are intentionally empty:
no valid approval evidence exists in the repository (EV-040). Code behaviour, passing tests and merged
PRs are never treated as management authorization. A recommendation, where given, is an engineering
view and is conditional; the choice belongs to the named owners.

**Kept separate (independent approvals):** P-FIN-A payment approval (Finance + Legal), P-FIN-B ledger and
money (Finance), P-FIN-C entitlements (Finance + Product), P-FIN-D refunds (Finance + Product, independent
of P-FIN-A). Financial approval and refund approval stay separate.

**Identity, capability and approval authority are distinct** (see `r5-person-identity-sod-DRAFT.md`):
- *identity* — which verified human acts (today only the account is known);
- *capability* — what an account may technically do (no Finance decision capability exists);
- *approval authority* — which human may approve what, within which scope and threshold (OPD-08).
One account holding several roles never simulates two independent approvers; two accounts of one human
never count as two people.

## P-FIN-A — Payment approval

### OPD-08 — Maker / checker thresholds and emergency exceptions

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | OPD-08 |
| Title | Maker / checker thresholds and emergency exceptions |
| Business question | What are the Finance approval authorities, independence rules (maker ≠ checker by person), thresholds, bulk-approval boundaries and emergency exceptions? |
| Background | Legacy allows Finance/admin to submit and approve (maker/checker conflict). Target: submitter ≠ reviewer, fail closed. |
| Current behavior | Finance approve / reject EVALUATOR-ONLY; no decision capability; person-level independence unmodelled → BLOCKED. |
| Evidence references | EV-018, EV-019, EV-026 (see `docs/governance/r5-evidence-index.json`) |
| Source | Cross-Role V1 §19 (OPD-08; MIS-G08 part) |
| Authority status | OPEN |
| Missing evidence | Organisation policy; person-identity model (P-SEC); rejection vocabulary; billing rejection transition. |
| Alternatives | Single independent reviewer for all amounts.<br>Amount thresholds: above a Finance-defined threshold, two independent reviewers.<br>Emergency exception: named role may approve own team's submission only with post-hoc independent review within a defined window. |
| Trade-offs | Thresholds and windows are Finance numbers; engineering must not choose them. Bulk approval must keep per-item evidence and outcomes. |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | finance, payments, authorization |
| Business owner | Product + Finance (MIS policy owner for MIS plans) |
| Required approver roles | Finance owner; Product; Security for identity aspects |
| Dependencies | P-SEC (person identity, capability model), D-43, rejection vocabulary |
| Affected blocker | P0-7 |
| Readiness | READY FOR HUMAN REVIEW |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Decision capability; approve / reject commands over the R4 contract; rejection vocabulary; bulk with per-item results. |
| Activation gate | Finance approval of activation + real role-account tests. |

### D-43 — Finance approval while receipts are not READY

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | D-43 |
| Title | Finance approval while receipts are not READY |
| Business question | May Finance approve a payment while any receipt is not READY (quarantined, storage error, not recoverable)? |
| Background | READY means bytes verified and stored content-addressed. |
| Current behavior | Approval blocked unless every receipt is READY (contract enforces). |
| Evidence references | EV-018 (see `docs/governance/r5-evidence-index.json`) |
| Source | Architecture V1 §32 |
| Authority status | OPEN (safe default available) |
| Missing evidence | — |
| Alternatives | Never (keep safe default).<br>Allow with a recorded exception and a follow-up obligation.<br>Allow for gateway payments only. |
| Trade-offs | Approving without verified evidence weakens audit (INV-015 / evidence durability). |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | payments, finance |
| Business owner | Finance |
| Required approver roles | Finance owner |
| Dependencies | OPD-08 |
| Affected blocker | P0-7 |
| Readiness | READY FOR HUMAN REVIEW |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | — |
| Activation gate | With OPD-08. |

### D-25 — Gateway auto-approval policy

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | D-25 |
| Title | Gateway auto-approval policy |
| Business question | Do gateway-confirmed payments stay auto-approved (legacy) or require Finance review? |
| Background | Legacy marks gateway payments approved without Finance review. |
| Current behavior | Synthetic gateway only; provenance GATEWAY_AUTO, never shown as Finance-reviewed. |
| Evidence references | EV-007, EV-015, EV-026 (see `docs/governance/r5-evidence-index.json`) |
| Source | Architecture V1 §32; E0 §6 #21 |
| Authority status | OPEN |
| Missing evidence | Real gateway adapter; callback verification design. |
| Alternatives | Keep auto-approval with distinct provenance.<br>Require Finance review for all gateway payments.<br>Auto-approve below a Finance threshold. |
| Trade-offs | Review adds latency; auto-approval relies on callback integrity. |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | payments, finance |
| Business owner | Finance |
| Required approver roles | Finance owner (+ Security for callback integrity) |
| Dependencies | D-26 (no auto-login), OPD-08 |
| Affected blocker | P0-7 (gateway path) |
| Readiness | READY FOR HUMAN REVIEW |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Real gateway adapter + callback route. |
| Activation gate | Separate. |

### D-44 — Payment evidence retention

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | D-44 |
| Title | Payment evidence retention |
| Business question | How long is bound payment evidence retained; when may unbound uploads be purged; what is the legal retention for NOT_RECOVERABLE records? |
| Background | Architecture §32 proposes 'bound indefinitely, unbound 7 days'. |
| Current behavior | Bound evidence kept; unbound expiry runs in REPORT mode only; purge refused outside development (R2). |
| Evidence references | EV-030, EV-034 (see `docs/governance/r5-evidence-index.json`) |
| Source | Architecture V1 §32 |
| Authority status | OPEN |
| Missing evidence | Legal review. |
| Alternatives | Retain all indefinitely.<br>Retain bound per legal minimum; purge unbound after N days (N set by Legal/Finance).<br>Retain per category with legal hold. |
| Trade-offs | Retention vs storage / privacy minimisation; legal obligations unknown to engineering. |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | payments, object store, backup |
| Business owner | Finance + Legal |
| Required approver roles | Finance and Legal owners |
| Dependencies | ADR #18 (privacy) |
| Affected blocker | P1-5 |
| Readiness | READY FOR HUMAN REVIEW |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Purge mode enablement; retention planner deletion (R3 dry-run only). |
| Activation gate | Separate operator authority. |

### REJECT-VOCAB (R4 gap) — Structured rejection reason vocabulary

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | REJECT-VOCAB (R4 gap) |
| Title | Structured rejection reason vocabulary |
| Business question | What structured reason codes may Finance use to reject a payment, and what billing transition follows a rejection? |
| Background | The contract requires a structured code but no vocabulary is approved; the rejection transition is undefined. |
| Current behavior | Any rejection → BLOCKED. |
| Evidence references | EV-018 (see `docs/governance/r5-evidence-index.json`) |
| Source | R4 finance review contract (REJECTION_REASON_VOCABULARY_UNDEFINED; BILLING_REJECTION_TRANSITION_UNDEFINED) |
| Authority status | OPEN (engineering gap surfaced by R4) |
| Missing evidence | Finance list of reasons; billing owner transition. |
| Alternatives | Finance-defined closed vocabulary.<br>Vocabulary + free-text detail (bounded).<br>Reuse legacy rejection labels after review. |
| Trade-offs | A closed vocabulary enables reporting; free text alone does not. |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | finance, billing |
| Business owner | Finance + billing owner |
| Required approver roles | Finance owner; Product |
| Dependencies | OPD-08 |
| Affected blocker | P0-7 |
| Readiness | READY FOR HUMAN REVIEW |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Vocabulary registry; transition. |
| Activation gate | With OPD-08. |

## P-FIN-B — Ledger and money

Legacy behaviour and R4 contracts: legacy posts wallet / commission effects on approval; the target
shadow ledger computes ACCRUAL_SHADOW and CASH_SHADOW and publishes neither; the R4 posting contract
needs an explicit basis, single unit, balanced int64 lines, effective vs recorded time and reversal as a
separate intent. **Never** is accrual / cash, conversion or rounding chosen here.

### D-41 — Ledger basis: accrual vs cash

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | D-41 |
| Title | Ledger basis: accrual vs cash |
| Business question | Is the target ledger accrual-basis or cash-basis (or both with a designated official basis)? |
| Background | The shadow ledger computes both and publishes neither. |
| Current behavior | Posting contract requires an explicit basis but chooses none; activation BLOCKED. |
| Evidence references | EV-021, EV-022 (see `docs/governance/r5-evidence-index.json`) |
| Source | Architecture V1 §32; Slice 9 |
| Authority status | OPEN |
| Missing evidence | Real-data parity of both bases against legacy wallet / commission facts. |
| Alternatives | Accrual (revenue at invoice issuance / stage approval).<br>Cash (revenue at verified collection).<br>Both, with one designated official basis and the other as management view. |
| Trade-offs | Accrual needs reliable issuance events (ADR-01); cash needs verified collection evidence (D-43, D-25). |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | Finance should see real-data parity of both bases before choosing. |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | finance |
| Business owner | Finance |
| Required approver roles | Finance owner (accounting authority) |
| Dependencies | OPD-06, ADR #18, E-4 |
| Affected blocker | P0-7 |
| Readiness | NEEDS FURTHER EVIDENCE BEFORE A DECISION |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Posting authority over the R4 contract; reconciliation. |
| Activation gate | Finance activation authority + parity. |

### OPD-06 — Units, rounding and the 0.5 boundary

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | OPD-06 |
| Title | Units, rounding and the 0.5 boundary |
| Business question | How are IRR and IRT handled (conversion, display, normalisation), and what rounding applies, including the 0.5 boundary? |
| Background | Money contract (approved): int64 + explicit unit; no implicit conversion; unknown ≠ zero. |
| Current behavior | Exact integer, one unit, no conversion; unrepresentable → BLOCKED; lint forbids ×10 / ÷10 in billing, payments, finance. |
| Evidence references | EV-001, EV-021 (see `docs/governance/r5-evidence-index.json`) |
| Source | Cross-Role V1 §19 (OPD-06); E0 §6 #13 |
| Authority status | OPEN |
| Missing evidence | Source-unit proof per legacy table on real data. |
| Alternatives | Keep units separate end to end; convert only in presentation with an approved ratio.<br>Normalise to one storage unit with an approved conversion and recorded source unit.<br>Per-domain unit policy. |
| Trade-offs | Any conversion needs an approved ratio and rounding rule; the existing money contract forbids implicit conversion. |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | Source-unit proof on real data is missing. |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | billing, payments, finance, reporting |
| Business owner | Product + Finance + Engineering (Cross-Role §19) |
| Required approver roles | Product and Finance owners; Engineering |
| Dependencies | ADR #18, E-4 |
| Affected blocker | P0-7 |
| Readiness | NEEDS FURTHER EVIDENCE BEFORE A DECISION |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Money contract extension. |
| Activation gate | With D-41. |

### D-37 — Installment credit: financing product or entitlement

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | D-37 |
| Title | Installment credit: financing product or entitlement |
| Business question | Is installment credit a financing product or an entitlement? |
| Background | Architecture §32 safe default: entitlement account flagged provisional. |
| Current behavior | No installment entitlement exists in the target (Slice 14). |
| Evidence references | EV-007, EV-028 (see `docs/governance/r5-evidence-index.json`) |
| Source | Architecture V1 §32; register (D-35b references D-37) |
| Authority status | OPEN |
| Missing evidence | Business model definition (D-35). |
| Alternatives | Financing product (receivable with schedule).<br>Entitlement (credit consumable for services).<br>Not offered in the target. |
| Trade-offs | Financing implies receivable accounting (D-41). |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | finance, catalog, biawin |
| Business owner | Finance |
| Required approver roles | Finance owner |
| Dependencies | D-35b, D-41 |
| Affected blocker | P1 |
| Readiness | READY FOR HUMAN REVIEW |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Account model. |
| Activation gate | Separate. |

## P-FIN-C — Entitlements and commissions

Engines (Slice 13): E1 purpose matrix (stage-approved hook; option default on), E2 rules auto-credit
(returns if E1 on), E3 dry-run poster (review / post actions), E4 Biawin project card (returns if E1 on).
The engine option has no single authority; `commission_engine` credits are written by both E2 and E3.
R4 reconciliation classifies MULTI_ENGINE_DUPLICATE, SAME_ENGINE_DUPLICATE, AMOUNT / UNIT / RECIPIENT
disagreement and SHADOW_ECONOMIC_EFFECT without choosing an engine. Shadow → posting requires OPD-09,
D-36, D-41, legacy engines disabled first, and per-item partial-processing outcomes with
reconcile-before-retry.

### OPD-09 — Entitlement engine routing, non-invoice keys, correction

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | OPD-09 |
| Title | Entitlement engine routing, non-invoice keys, correction |
| Business question | Which engine owns each entitlement type, what keys identify non-invoice entitlements, and how are corrections made? |
| Background | Four legacy engines E1–E4; E1 enabled disables E2 auto-credit, E3 posting and E4; the engine option has no single authority. |
| Current behavior | Shadow only; no posting; non-invoice credits unkeyed; cross-engine reconciliation classifies duplicates. |
| Evidence references | EV-024, EV-025 (see `docs/governance/r5-evidence-index.json`) |
| Source | Cross-Role V1 §19 (OPD-09) |
| Authority status | OPEN |
| Missing evidence | Active routes and flags on real data; key proof for non-invoice entitlements. |
| Alternatives | Single engine per entitlement type (named), others disabled first.<br>E1 as the sole engine for invoice entitlements; separate policy for non-invoice.<br>Keep legacy engines; target shadow only. |
| Trade-offs | Any routing must prevent double credit across engines (INV-032). |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | Real-data reconciliation across E1–E4 has not run. |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | commission, finance, HR, sales |
| Business owner | Product + Finance + HR + Engineering (Cross-Role §19) |
| Required approver roles | Product, Finance, HR owners; Engineering |
| Dependencies | D-36, D-41, ADR #18 |
| Affected blocker | P0-7 |
| Readiness | NEEDS FURTHER EVIDENCE BEFORE A DECISION |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Posting engine; legacy engines disabled first. |
| Activation gate | Separate Finance activation. |

### D-36 — Pro-rata commission split across bundle components

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | D-36 |
| Title | Pro-rata commission split across bundle components |
| Business question | How is commission allocated across components of a bundle (pro-rata, per component, none)? |
| Background | component_pro_rata not registered; E4 rows NOT_COMPARABLE. |
| Current behavior | Component entitlements have no key (unkeyed). |
| Evidence references | EV-024, EV-025 (see `docs/governance/r5-evidence-index.json`) |
| Source | Architecture V1 §32; Slice 13 |
| Authority status | OPEN |
| Missing evidence | Historical parity for bundles. |
| Alternatives | Pro-rata by list price.<br>Per-component fixed policy.<br>No component allocation (whole-bundle entitlement). |
| Trade-offs | Pro-rata needs a price basis per component. |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | Historical parity is missing. |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | commission |
| Business owner | Finance + Product |
| Required approver roles | Finance and Product owners |
| Dependencies | OPD-09 |
| Affected blocker | P0-7 |
| Readiness | NEEDS FURTHER EVIDENCE BEFORE A DECISION |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Allocation function. |
| Activation gate | With OPD-09. |

## P-FIN-D — Refunds

Preserved distinctions: requested ≠ reviewed ≠ authorized ≠ externally executed ≠ evidence received ≠
reconciled ≠ completed. Invoice cancellation is not refund execution. An HTTP 2xx is not proof of returned
money. Duplicate execution is prevented by idempotent keys and no UNKNOWN → AUTHORIZED path (R4).
Complaint outcomes may *request* a refund; they never execute one (Cross-Role §7, §9).

### OPD-04 — Refund object, proof and authority

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | OPD-04 |
| Title | Refund object, proof and authority |
| Business question | What is refunded (invoice, stage or payment), who requests / reviews / authorises (independent humans), what proof is required, how is it executed and reconciled, and how does it relate to Complaint outcomes? |
| Background | Request, report, proof and financial execution are distinct; the original invoice is preserved. Cancellation ≠ refund; HTTP 2xx ≠ money returned. |
| Current behavior | R4 refund state machine: every transition BLOCKED by OPD-04; money facts validated; history integrity; person-level independence fail-closed. |
| Evidence references | EV-023, EV-042 (see `docs/governance/r5-evidence-index.json`) |
| Source | Cross-Role V1 §19 (OPD-04), §9; Gate-0 INV-015 |
| Authority status | OPEN |
| Missing evidence | Bank / provider proof standard; execution channel; Complaint linkage; Legal input on compensation. |
| Alternatives | Refund object = payment (per verified payment), partial allowed up to remaining refundable.<br>Refund object = invoice stage.<br>Refund object = whole invoice only. |
| Trade-offs | Requester ≠ reviewer ≠ authoriser requires the P-SEC person model; partial refunds need exact remaining-refundable tracking (R4 enforces). Financial approval (OPD-08) and refund approval stay separate. |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | finance, payments, billing, complaint (after-sales) |
| Business owner | Product + Finance |
| Required approver roles | Product and Finance owners; Legal for compensation |
| Dependencies | P-SEC, OPD-08, D-41 (posting), D-44 (evidence) |
| Affected blocker | P0-7 |
| Readiness | READY FOR HUMAN REVIEW |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Refund commands over the R4 state machine; proof storage; reconciliation. |
| Activation gate | Separate refund activation authority. |
