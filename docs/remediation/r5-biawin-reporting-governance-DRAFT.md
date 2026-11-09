# R5 — P-BIAWIN, P-REPORT and cross-domain governance (DRAFT — NOT APPROVED)

All decision records below are **DRAFT — NOT APPROVED**. The approval fields (approval date, named
approvers, selected option, conditions, evidence of approval, effective date) are intentionally empty:
no valid approval evidence exists in the repository (EV-040). Code behaviour, passing tests and merged
PRs are never treated as management authorization. A recommendation, where given, is an engineering
view and is conditional; the choice belongs to the named owners.

**Accepted customer pathways (frozen; Cross-Role §7, EV-037) — unchanged by this package:**
1. ناراضی → خدمات پس از فروش → Complaint.
2. بدون خدمات → بایگانی.
3. خدمات مازاد → API / CRM بیاوین.

Biawin integration belongs to **خدمات مازاد**, never to the ناراضی branch. A Biawin subscription may
contain several independent service cards, each with its own operational supervisor. No new routing or
financial policy is approved here.

## P-BIAWIN

### D-35a — Sub-item (a): mapping UNRESOLVED (C-03)

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | D-35a |
| Title | Sub-item (a): mapping UNRESOLVED (C-03) |
| Business question | Two exact source mappings exist and neither is confirmed (C-03). Under Arch §14 / register / Slice 14 the question is: What is the cash-credit entitlement of a Subscription+ plan and how is it consumed? Under a positional reading of the unlettered Arch §32 list, sub-item (a) would be 'Subscription+ business model'. Product + Finance first confirm the mapping; this record presupposes neither. |
| Background | Arch §14 (560ca6c): '(a) cash credit = liability vs voucher vs merchant credit'. Arch §32 row D-35a–e (560ca6c), 1st of five unlettered topics: 'Subscription+ business model' (whether §32 is positional is not stated). Register §1 (7f1eb94) follows §14 order without letters; the Slice 14 matrix (1d873a7) assigns letters from §14. The implementation matrix is not treated as authority over the frozen architecture; R5 selects no mapping. |
| Current behavior | No cash-credit entitlement exists. |
| Evidence references | EV-028, EV-033, EV-046, EV-047, EV-048 (see `docs/governance/r5-evidence-index.json`) |
| Source | Architecture V1 §14 (explicit letter) and §32 (unlettered list); register §1; Slice 14 matrix §3. Mapping UNRESOLVED pending Product + Finance (C-03) |
| Authority status | OPEN |
| Missing evidence | Product + Finance confirmation of the D-35 sub-item mapping (C-03). Business model definition. |
| Alternatives | (Listed under the §14 labelling; void if a different mapping is confirmed.) Define as consumable credit.<br>Define as prepaid balance.<br>Not offered. |
| Trade-offs | Credit implies ledger accounts (P-FIN-B). |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | catalog, biawin, finance |
| Business owner | Product + Finance |
| Required approver roles | Product and Finance owners (mapping); Product and Finance owners (decision) |
| Dependencies | C-03 mapping confirmation, D-41, D-37 |
| Affected blocker | P1-2 |
| Readiness | NEEDS PRODUCT + FINANCE MAPPING CONFIRMATION (C-03) BEFORE A DECISION |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Entitlement model. |
| Activation gate | Separate. |

### D-35b — Sub-item (b): mapping UNRESOLVED (C-03)

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | D-35b |
| Title | Sub-item (b): mapping UNRESOLVED (C-03) |
| Business question | Two exact source mappings exist and neither is confirmed (C-03). Under Arch §14 / register / Slice 14 the question is: How does installment credit apply to Subscription+ (D-37 decides the account type)? Under a positional reading of the unlettered Arch §32 list, sub-item (b) would be 'lifecycle'. Product + Finance first confirm the mapping; this record presupposes neither. |
| Background | Arch §14 (560ca6c): '(b) installment credit = financing product?'. Arch §32 row D-35a–e (560ca6c), 2nd of five unlettered topics: 'lifecycle' (whether §32 is positional is not stated). Register §1 (7f1eb94) follows §14 order without letters; the Slice 14 matrix (1d873a7) assigns letters from §14. The implementation matrix is not treated as authority over the frozen architecture; R5 selects no mapping. |
| Current behavior | No installment entitlement. |
| Evidence references | EV-028, EV-033, EV-046, EV-047, EV-048 (see `docs/governance/r5-evidence-index.json`) |
| Source | Architecture V1 §14 (explicit letter) and §32 (unlettered list); register §1; Slice 14 matrix §3. Mapping UNRESOLVED pending Product + Finance (C-03) |
| Authority status | OPEN |
| Missing evidence | Product + Finance confirmation of the D-35 sub-item mapping (C-03). D-37. |
| Alternatives | (Listed under the §14 labelling; void if a different mapping is confirmed.) Follow D-37.<br>Exclude from Subscription+. |
| Trade-offs | — |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | finance, biawin |
| Business owner | Finance |
| Required approver roles | Product and Finance owners (mapping); Finance owner (decision) |
| Dependencies | C-03 mapping confirmation, D-37 |
| Affected blocker | P1-2 |
| Readiness | NEEDS PRODUCT + FINANCE MAPPING CONFIRMATION (C-03) BEFORE A DECISION |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | — |
| Activation gate | — |

### D-35c — Sub-item (c): mapping UNRESOLVED (C-03)

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | D-35c |
| Title | Sub-item (c): mapping UNRESOLVED (C-03) |
| Business question | Two exact source mappings exist and neither is confirmed (C-03). Under Arch §14 / register / Slice 14 the question is: When do Subscription+ entitlements expire, how are they spent and redeemed, and may they be suspended or reactivated? Under a positional reading of the unlettered Arch §32 list, sub-item (c) would be 'cash-credit semantics'. Product + Finance first confirm the mapping; this record presupposes neither. |
| Background | Arch §14 (560ca6c): '(c) expiry/spend rules'. Arch §32 row D-35a–e (560ca6c), 3rd of five unlettered topics: 'cash-credit semantics' (whether §32 is positional is not stated). Register §1 (7f1eb94) follows §14 order without letters; the Slice 14 matrix (1d873a7) assigns letters from §14. The implementation matrix is not treated as authority over the frozen architecture; R5 selects no mapping. |
| Current behavior | No lifecycle transitions. |
| Evidence references | EV-028, EV-033, EV-046, EV-047, EV-048 (see `docs/governance/r5-evidence-index.json`) |
| Source | Architecture V1 §14 (explicit letter) and §32 (unlettered list); register §1; Slice 14 matrix §3. Mapping UNRESOLVED pending Product + Finance (C-03) |
| Authority status | OPEN |
| Missing evidence | Product + Finance confirmation of the D-35 sub-item mapping (C-03). Lifecycle definition. |
| Alternatives | (Listed under the §14 labelling; void if a different mapping is confirmed.) Fixed term from activation.<br>Term from first use.<br>No expiry. |
| Trade-offs | Expiry affects revenue recognition (D-41). |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | catalog, biawin |
| Business owner | Product + Finance |
| Required approver roles | Product and Finance owners (mapping); Product and Finance owners (decision) |
| Dependencies | C-03 mapping confirmation, D-41 |
| Affected blocker | P1-2 |
| Readiness | NEEDS PRODUCT + FINANCE MAPPING CONFIRMATION (C-03) BEFORE A DECISION |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Lifecycle commands. |
| Activation gate | Separate. |

### D-35d — Sub-item (d): mapping UNRESOLVED (C-03)

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | D-35d |
| Title | Sub-item (d): mapping UNRESOLVED (C-03) |
| Business question | Two exact source mappings exist and neither is confirmed (C-03). Under Arch §14 / register / Slice 14 the question is: Is Biawin the system of record for redemption, or evidence only? Under a positional reading of the unlettered Arch §32 list, sub-item (d) would be 'Biawin fulfilment'. Product + Finance first confirm the mapping; this record presupposes neither. |
| Background | Arch §14 (560ca6c): '(d) Biawin as redemption system of record'. Arch §32 row D-35a–e (560ca6c), 4th of five unlettered topics: 'Biawin fulfilment' (whether §32 is positional is not stated). Register §1 (7f1eb94) follows §14 order without letters; the Slice 14 matrix (1d873a7) assigns letters from §14. The implementation matrix is not treated as authority over the frozen architecture; R5 selects no mapping. |
| Current behavior | Biawin never authoritative; synthetic provider only. |
| Evidence references | EV-028, EV-033, EV-046, EV-047, EV-048 (see `docs/governance/r5-evidence-index.json`) |
| Source | Architecture V1 §14 (explicit letter) and §32 (unlettered list); register §1; Slice 14 matrix §3. Mapping UNRESOLVED pending Product + Finance (C-03) |
| Authority status | OPEN |
| Missing evidence | Product + Finance confirmation of the D-35 sub-item mapping (C-03). Provider contract. |
| Alternatives | (Listed under the §14 labelling; void if a different mapping is confirmed.) CRM is SoR; Biawin reports.<br>Biawin is SoR; CRM mirrors.<br>Shared with reconciliation. |
| Trade-offs | SoR choice decides reconciliation direction. |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | biawin, integration |
| Business owner | Product + Finance |
| Required approver roles | Product and Finance owners (mapping); Product and Finance owners (decision) |
| Dependencies | C-03 mapping confirmation, D-38 |
| Affected blocker | P1-2 |
| Readiness | NEEDS PRODUCT + FINANCE MAPPING CONFIRMATION (C-03) BEFORE A DECISION |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Provider adapter. |
| Activation gate | Separate. |

### D-35e — Sub-item (e): mapping UNRESOLVED (C-03)

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | D-35e |
| Title | Sub-item (e): mapping UNRESOLVED (C-03) |
| Business question | Two exact source mappings exist and neither is confirmed (C-03). Under Arch §14 / register / Slice 14 the question is: Is conversion mandatory for the no-cash path? Under a positional reading of the unlettered Arch §32 list, sub-item (e) would be 'mandatory conversion'. Product + Finance first confirm the mapping; this record presupposes neither. |
| Background | Arch §14 (560ca6c): '(e) is conversion mandatory for the no-cash path'. Arch §32 row D-35a–e (560ca6c), 5th of five unlettered topics: 'mandatory conversion' (whether §32 is positional is not stated). Register §1 (7f1eb94) follows §14 order without letters; the Slice 14 matrix (1d873a7) assigns letters from §14. The implementation matrix is not treated as authority over the frozen architecture; R5 selects no mapping. |
| Current behavior | Routing explicit; never inferred. |
| Evidence references | EV-028, EV-033, EV-046, EV-047, EV-048 (see `docs/governance/r5-evidence-index.json`) |
| Source | Architecture V1 §14 (explicit letter) and §32 (unlettered list); register §1; Slice 14 matrix §3. Mapping UNRESOLVED pending Product + Finance (C-03) |
| Authority status | OPEN |
| Missing evidence | Product + Finance confirmation of the D-35 sub-item mapping (C-03). — |
| Alternatives | (Listed under the §14 labelling; void if a different mapping is confirmed.) Mandatory.<br>Optional.<br>Not applicable. |
| Trade-offs | — |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | biawin, sales |
| Business owner | Product (Finance for the C-03 mapping) |
| Required approver roles | Product and Finance owners (mapping); Product owner (decision) |
| Dependencies | C-03 mapping confirmation |
| Affected blocker | P1-2 |
| Readiness | NEEDS PRODUCT + FINANCE MAPPING CONFIRMATION (C-03) BEFORE A DECISION |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | — |
| Activation gate | — |

### D-38 — Biawin wallet charge semantics and system of record

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | D-38 |
| Title | Biawin wallet charge semantics and system of record |
| Business question | What does a Biawin wallet charge mean financially, and which system is the record? |
| Background | Legacy charge_wallet. |
| Current behavior | No target charge; WALLET_CHARGE only BLOCKED (DB CHECK). |
| Evidence references | EV-028 (see `docs/governance/r5-evidence-index.json`) |
| Source | Architecture V1 §32 |
| Authority status | OPEN |
| Missing evidence | Provider contract. |
| Alternatives | Charge = prepaid credit purchase (ledger entry).<br>Charge = provider-side balance only (no CRM ledger).<br>Not supported. |
| Trade-offs | Ledger entries need D-41. |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | biawin, finance |
| Business owner | Finance + Product |
| Required approver roles | Finance and Product owners |
| Dependencies | D-41, D-35d (§14 labelling; C-03) |
| Affected blocker | P1-2 |
| Readiness | READY FOR HUMAN REVIEW |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Provider operation. |
| Activation gate | Separate. |

### D-33 — Operations upgrade-payment authority

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | D-33 |
| Title | Operations upgrade-payment authority |
| Business question | May Operations take an upgrade payment, and under which authority? |
| Background | — |
| Current behavior | Not granted. |
| Evidence references | EV-007 (see `docs/governance/r5-evidence-index.json`) |
| Source | Architecture V1 §32 |
| Authority status | OPEN |
| Missing evidence | — |
| Alternatives | Not granted.<br>Granted with Finance review.<br>Granted for named roles. |
| Trade-offs | Expands payment submitters (OPD-08). |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | payments, fulfillment |
| Business owner | Product + Finance |
| Required approver roles | Product and Finance owners |
| Dependencies | OPD-08 |
| Affected blocker | P1 |
| Readiness | READY FOR HUMAN REVIEW |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | — |
| Activation gate | — |

### D-39 — Catalog all_except semantics

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | D-39 |
| Title | Catalog all_except semantics |
| Business question | What does catalog scope all_except mean? |
| Background | — |
| Current behavior | selected / all only; all_except imported read-only. |
| Evidence references | EV-007 (see `docs/governance/r5-evidence-index.json`) |
| Source | Architecture V1 §32 |
| Authority status | OPEN |
| Missing evidence | — |
| Alternatives | All except listed.<br>Retire all_except.<br>Map to selected. |
| Trade-offs | — |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | catalog, authorization |
| Business owner | Sales + Product |
| Required approver roles | Sales and Product owners |
| Dependencies | — |
| Affected blocker | P1 |
| Readiness | READY FOR HUMAN REVIEW |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | — |
| Activation gate | — |

## P-REPORT

### OPD-07 — Cohort, window, rate and attribution for KPIs

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | OPD-07 |
| Title | Cohort, window, rate and attribution for KPIs |
| Business question | Which cohort, time window, rate formula and as-of attribution define each KPI? |
| Background | — |
| Current behavior | Metrics marked NOT FINAL. |
| Evidence references | EV-008 (see `docs/governance/r5-evidence-index.json`) |
| Source | Cross-Role V1 §19 (OPD-07; SD-05, SG-05) |
| Authority status | OPEN |
| Missing evidence | Same-cohort metadata; historical coverage. |
| Alternatives | Event-range cohorts.<br>Current-snapshot cohorts.<br>Both, labelled. |
| Trade-offs | — |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | Coverage evidence missing. |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | reporting, sales, MIS |
| Business owner | Product + Sales + MIS + Finance |
| Required approver roles | Product, Sales, MIS, Finance owners |
| Dependencies | OPD-01 |
| Affected blocker | P1-7 |
| Readiness | NEEDS FURTHER EVIDENCE BEFORE A DECISION |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Metric contracts. |
| Activation gate | — |

### D-23 — Invoice Review visibility, PII and export

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | D-23 |
| Title | Invoice Review visibility, PII and export |
| Business question | Who may see Invoice Review data, with which PII, and is export allowed? |
| Background | Legacy: organisation-wide uncapped CSV with name, phones, address. |
| Current behavior | No export; audited read behind a flag. |
| Evidence references | EV-007 (see `docs/governance/r5-evidence-index.json`) |
| Source | Architecture V1 §32; E0 §7B |
| Authority status | OPEN |
| Missing evidence | Privacy review. |
| Alternatives | No export.<br>Scoped export with audit and field masking.<br>Full export for named roles. |
| Trade-offs | — |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | reporting |
| Business owner | Product + Finance |
| Required approver roles | Product, Finance; Security / Legal (privacy) |
| Dependencies | ADR #18, D-45 |
| Affected blocker | P1-7 |
| Readiness | READY FOR HUMAN REVIEW |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | — |
| Activation gate | — |

### D-24 — Invoice Review product / design owner

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | D-24 |
| Title | Invoice Review product / design owner |
| Business question | Who owns the Invoice Review product and design (no frozen SN-207 design exists)? |
| Background | — |
| Current behavior | No target UI. |
| Evidence references | EV-007 (see `docs/governance/r5-evidence-index.json`) |
| Source | Architecture V1 §32; E0 §3 |
| Authority status | OPEN |
| Missing evidence | — |
| Alternatives | Product owns; design via SN process.<br>Finance owns. |
| Trade-offs | — |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | reporting UI |
| Business owner | Product |
| Required approver roles | Product owner |
| Dependencies | — |
| Affected blocker | P1-7 |
| Readiness | READY FOR HUMAN REVIEW |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | — |
| Activation gate | — |

### OPD-01 — Ambiguous lineage, aliases, case coverage

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | OPD-01 |
| Title | Ambiguous lineage, aliases, case coverage |
| Business question | How are ambiguous lineage, aliases and continuation vs distinct cases resolved? |
| Background | — |
| Current behavior | Unknown kept unknown. |
| Evidence references | EV-008 (see `docs/governance/r5-evidence-index.json`) |
| Source | Cross-Role V1 §19 (OPD-01; SG-06, MIS-G02) |
| Authority status | OPEN |
| Missing evidence | Stable source proof + coverage on real data. |
| Alternatives | Keep UNKNOWN until proof.<br>Accept MIS-reviewed aliases.<br>Rule-based continuation. |
| Trade-offs | — |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | Real-data coverage missing. |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | all |
| Business owner | Product + MIS + Sales + Engineering |
| Required approver roles | Product, MIS, Sales; Engineering |
| Dependencies | ADR #18 |
| Affected blocker | P0-3 / P1-7 |
| Readiness | NEEDS FURTHER EVIDENCE BEFORE A DECISION |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | — |
| Activation gate | — |

## Cross-domain

### OPD-02 — Sensitive ACK / SLA / escalation owner and routing

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | OPD-02 |
| Title | Sensitive ACK / SLA / escalation owner and routing |
| Business question | Who owns acknowledgement, SLA and escalation for sensitive handoffs, and how are they routed? |
| Background | — |
| Current behavior | No ACK / SLA fields. |
| Evidence references | EV-008 (see `docs/governance/r5-evidence-index.json`) |
| Source | Cross-Role V1 §19 (OPD-02) |
| Authority status | OPEN |
| Missing evidence | Existing owner / path evidence. |
| Alternatives | Named owner per handoff type.<br>Manager queue escalation.<br>Not in scope. |
| Trade-offs | — |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | cross-domain |
| Business owner | Product + Sales + MIS + HR |
| Required approver roles | Product, Sales, MIS, HR owners |
| Dependencies | — |
| Affected blocker | P2 |
| Readiness | READY FOR HUMAN REVIEW |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | — |
| Activation gate | — |

### D-40 — Payment reward wheel: keep or retire

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | D-40 |
| Title | Payment reward wheel: keep or retire |
| Business question | Is the payment reward wheel kept or retired in the target? |
| Background | — |
| Current behavior | Read-only import (no spec yet). |
| Evidence references | EV-007 (see `docs/governance/r5-evidence-index.json`) |
| Source | Architecture V1 §32 |
| Authority status | OPEN |
| Missing evidence | — |
| Alternatives | Keep.<br>Retire.<br>Keep read-only history. |
| Trade-offs | — |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | sales, finance |
| Business owner | Product |
| Required approver roles | Product owner |
| Dependencies | — |
| Affected blocker | P1 |
| Readiness | READY FOR HUMAN REVIEW |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | — |
| Activation gate | — |
