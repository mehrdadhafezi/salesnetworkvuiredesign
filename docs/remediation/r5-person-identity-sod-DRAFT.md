# R5 — P-SEC: person identity, authorization and separation of duties (DRAFT — NOT APPROVED)

**Critical R4 gap.** Every actor in the target is a **user account** (`identity.user_account`, one row per
legacy user; EV-020). An account id does **not** prove a distinct human: one person may hold several
accounts, and one account may carry several roles.

R4 therefore made every separation-of-duties check fail closed:
- the Finance decision contract;
- the refund review and authorization.

Both answer BLOCKED with `IDENTITY_PERSON_LINK_UNMODELLED` (EV-019, EV-023).

This package proposes an architecture decision. It approves nothing, and:
- no identity is merged or linked;
- no permission or grant is created.

All records below are **DRAFT — NOT APPROVED**; the approval fields are empty.

## 1. Concepts that must stay distinct

| Concept | Meaning | Today (target) | Gap |
|---|---|---|---|
| User account | A login / legacy user row | `identity.user_account` (shadow import) | None for accounts themselves |
| Verified real person | A human proven by an approved method | **Not modelled** | No person entity, no verification method |
| Multiple accounts of one person | Several accounts → one person | **Not detectable** | SoD can be bypassed with a second account |
| Employee identity | HR's record of an employee | Legacy HR profile (shadow) | Not linked to accounts authoritatively |
| Organizational role / position | Position in the hierarchy | organization module (shadow) | Hierarchy ≠ permission (AGENTS.md) |
| Capability | What an account may technically do | `identity.capability`, granted to no real role for writes | No Finance decision capability (R4 gap) |
| Permission scope | Route → module → action → row / field | Authorization evaluator (Slice 4) | Real role matrix untested (D-42) |
| Delegation | Acting for another person within limits | **Not modelled** (OPD-10) | Needs person identity |
| Effective date of authorization | When a grant or role starts / ends | Partly in HR requests; no grant history | Grant-change procedure (E-7) |
| Review independence | Maker ≠ checker by *human* | Account-level only | Needs person identity |
| Historical identity evidence | Who acted then, as recorded then | Event actor recorded per account | Corrections must not rewrite history |

**Rules carried over unchanged:**
- **Position ≠ role ≠ capability ≠ credential.**
- **Hierarchy ≠ permission.**
- **View ≠ write.**

Also by AGENTS.md:
- one account with several roles never simulates two approvers;
- two accounts of one human never count as two people.

## 2. Approaches compared (no selection)

| Criterion | A — HR-verified Person registry | B — External identity provider (IdP) | C — Declarative SoD (approver asserts distinctness) | D — Keep blocked |
|---|---|---|---|---|
| False merges (two people joined) | Possible; control: two-person reviewed link + reversible link records | Lower (IdP proofing); control: IdP policy | n/a (no merge) | n/a |
| Duplicate identities (one person, two persons) | Possible; control: periodic duplicate review | Lower | **Undetected** | n/a |
| Identity verification | HR documents; method to be approved | IdP proofing | None | None |
| Historical corrections | Append-only link history; past events keep their account actor | Federation history | None | n/a |
| Privacy and minimization | Store only person key + evidence reference; no national-ID copy without Legal approval | Data held by IdP; contract needed | Minimal | Minimal |
| SoD enforcement | Per person, at decision time (as-of link) | Per IdP subject | Account level only (**insufficient**) | Actions BLOCKED |
| Auditability | Link events + reviewer | IdP logs + federation | Assertion record only | — |
| Reversibility of mistaken mappings | Unlink event; decisions made under a wrong link flagged for review | IdP correction | — | — |
| Cost / dependencies | HR data quality; HR + Security owners | New infrastructure; Legal / contract | Low | None; P0-7 stays blocked |

**Engineering view (conditional, not a selection):** A or B can satisfy the stated SoD requirement. C cannot
prove human independence. D keeps every SoD-dependent action blocked.

The choice depends on HR data quality, the privacy review and Security ownership. Those are business and
security decisions.

**Never (any option):**
- an automatic identity merge;
- a merge by phone, name or e-mail similarity;
- a new permission or grant;
- a change to Finance or HR authority without its own decision record.

## 3. Related security decisions

These are packaged here as P-SEC: D-34, D-42, D-45, D-26 and ADR #18. P-SEC is signed by the Security owner,
separately from Finance.

## 4. Decision records

### R5-SEC-01 (proposed ADR) — Authoritative person identity and account-to-person linkage

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | R5-SEC-01 (proposed ADR) |
| Title | Authoritative person identity and account-to-person linkage |
| Business question | Should the target introduce an authoritative Person (verified human) with account-to-person links, so separation of duties is enforced per human; and if so, by which approach? |
| Background | identity.user_account is one row per legacy user; one human may hold several accounts; account inequality does not prove two humans. |
| Current behavior | Finance decision and refund contracts BLOCKED by IDENTITY_PERSON_LINK_UNMODELLED; no merge or link exists. |
| Evidence references | EV-019, EV-020 (see `docs/governance/r5-evidence-index.json`) |
| Source | R4 gap IDENTITY_PERSON_LINK_UNMODELLED; this package |
| Authority status | PROPOSED (new ADR; not approved) |
| Missing evidence | HR source of employee identity; verification method; privacy review. |
| Alternatives | A — HR-verified Person registry: HR records a Person per employee; accounts linked by a reviewed, reversible link with evidence.<br>B — Security-managed identity provider (external IdP) as the person authority; accounts federated.<br>C — Declarative SoD only: accounts asserted distinct by an approver; no Person entity.<br>D — Keep blocked: no person model; SoD-dependent actions stay BLOCKED. |
| Trade-offs | A: fits existing HR data; risk of false merges (control: two-person reviewed links, reversible, append-only). B: strongest verification; new infrastructure and legal review. C: cheapest; does not prove independence. D: blocks P0-7 decisions that need SoD. |
| Recommended option, if defensible | Conditional engineering view: A or B satisfy the stated SoD requirement; C does not prove human independence. No option is selected. |
| Why the recommendation is conditional | Choice depends on HR data quality, privacy review and Security ownership — business and security decisions. |
| Risks and compensating controls | False merge (one link joins two people) → wrongful denial; false split → SoD bypass. Controls: reversible links, independent review of every link, audit, no automatic merge. |
| Dependent modules | identity, organization, authorization, finance, HR |
| Business owner | Security owner + HR |
| Required approver roles | Security, HR, Legal (privacy), Product |
| Dependencies | D-34, D-42, ADR #18 |
| Affected blocker | P0-7 (SoD), P0-6 (handover) |
| Readiness | READY FOR HUMAN REVIEW |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Person entity + link records + SoD evaluation; no grants. |
| Activation gate | Security activation + real account tests. |

### D-34 — Initial password / provisioning policy

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | D-34 |
| Title | Initial password / provisioning policy |
| Business question | How are accounts provisioned, how is the first secret set and delivered, and how are reset / recovery handled? |
| Background | Architecture §32 safe default 'random password + reset flow' (conflict C-04 with R1). |
| Current behavior | R1: authentication works; no code writes a credential (app role SELECT only); provisioning undecided. |
| Evidence references | EV-029, EV-035 (see `docs/governance/r5-evidence-index.json`) |
| Source | Architecture V1 §32; R1 |
| Authority status | OPEN |
| Missing evidence | Security design. |
| Alternatives | Admin-issued one-time password + forced change.<br>Invitation link with expiry.<br>External IdP (see R5-SEC-01 B). |
| Trade-offs | Recovery channel (SMS / e-mail / admin) has security implications. |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | identity |
| Business owner | Security + HR |
| Required approver roles | Security and HR owners |
| Dependencies | R5-SEC-01 |
| Affected blocker | P0-1 (provisioning part) |
| Readiness | READY FOR HUMAN REVIEW |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Provisioning command (credential write). |
| Activation gate | Security approval + real accounts. |

### D-42 — Can HR edit sales capabilities?

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | D-42 |
| Title | Can HR edit sales capabilities? |
| Business question | May HR grant or edit sales capabilities (portal_access bulk / inline), or does identity own all grants? |
| Background | Legacy HR edits portal access in bulk / inline. |
| Current behavior | HR cannot grant; identity owns grants; no grant-change procedure (E-7). |
| Evidence references | EV-007 (see `docs/governance/r5-evidence-index.json`) |
| Source | Architecture V1 §32 |
| Authority status | OPEN |
| Missing evidence | Grant-change audit procedure (E-7). |
| Alternatives | Identity / Security only.<br>HR may request; Security approves.<br>HR may grant within a capped set with audit. |
| Trade-offs | Separating request and grant supports SoD. |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | organization, authorization, identity |
| Business owner | Product + HR + Sales |
| Required approver roles | Product, HR, Sales owners; Security |
| Dependencies | R5-SEC-01, E-7 |
| Affected blocker | P1-4 |
| Readiness | READY FOR HUMAN REVIEW |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Grant-change procedure. |
| Activation gate | Separate. |

### D-45 — Behaviour when a sensitive-read audit write fails

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | D-45 |
| Title | Behaviour when a sensitive-read audit write fails |
| Business question | When the audit record of a sensitive read cannot be written, must the read fail? |
| Background | Safe default: fail closed for PII-bearing views. |
| Current behavior | Fail closed for PII views (Slice 8). |
| Evidence references | EV-007 (see `docs/governance/r5-evidence-index.json`) |
| Source | Architecture V1 §32 |
| Authority status | OPEN (safe default available) |
| Missing evidence | — |
| Alternatives | Fail closed (safe default).<br>Serve and alert.<br>Serve masked data. |
| Trade-offs | Availability vs auditability. |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | reporting, audit |
| Business owner | Product + Security |
| Required approver roles | Product and Security owners |
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

### D-26 — Customer auto-login after gateway callback

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | D-26 |
| Title | Customer auto-login after gateway callback |
| Business question | Should a customer be logged in automatically after a gateway callback (legacy behaviour)? |
| Background | Legacy issues a session from a payment callback. |
| Current behavior | Not replicated; lint forbids session issuance in payments. |
| Evidence references | EV-007 (see `docs/governance/r5-evidence-index.json`) |
| Source | Architecture V1 §32 |
| Authority status | OPEN |
| Missing evidence | Security review of callback integrity. |
| Alternatives | Drop.<br>Replicate with signed, short-lived, single-use token.<br>Replace with explicit customer login. |
| Trade-offs | Convenience vs session fixation / account takeover risk. |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | identity, payments, portal |
| Business owner | Product + Security |
| Required approver roles | Product and Security owners |
| Dependencies | D-25 |
| Affected blocker | P1-3 |
| Readiness | READY FOR HUMAN REVIEW |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Portal auth. |
| Activation gate | Separate. |

### ADR #18 — Staging and anonymisation policy (real data)

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | ADR #18 |
| Title | Staging and anonymisation policy (real data) |
| Business question | Under what anonymisation, access and retention rules may real legacy data be loaded into staging for reconciliation? |
| Background | Real-data reconciliation (INV-036) has never run. |
| Current behavior | Synthetic-only load. |
| Evidence references | EV-015, EV-039 (see `docs/governance/r5-evidence-index.json`) |
| Source | E0 §6 #18 (DEFERRED); register |
| Authority status | DEFERRED / OPEN |
| Missing evidence | Privacy review. |
| Alternatives | No real data; synthetic only (status quo).<br>Anonymised extract (fields masked, keys preserved) in an isolated staging.<br>Full extract in a restricted environment with named access and expiry. |
| Trade-offs | Reconciliation (P0-3) is impossible without some real data; privacy obligations unknown to engineering. |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | integration, all |
| Business owner | Product + Security |
| Required approver roles | Product, Security and Legal owners |
| Dependencies | — |
| Affected blocker | P0-3 |
| Readiness | READY FOR HUMAN REVIEW |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | E-4 real snapshot. |
| Activation gate | Separate per environment. |
