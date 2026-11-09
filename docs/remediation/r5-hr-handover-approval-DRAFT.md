# R5 — P-HR employment and access handover (DRAFT — NOT APPROVED)

**Blocker:** P0-6 stays OPEN until OPD-05 / SD-04 are decided and implemented with real HR role
evidence (HR runtime is NOT LIVE VERIFIED, EV-044).

All decision records below are **DRAFT — NOT APPROVED**. The approval fields (approval date, named
approvers, selected option, conditions, evidence of approval, effective date) are intentionally empty:
no valid approval evidence exists in the repository (EV-040). Code behaviour, passing tests and merged
PRs are never treated as management authorization. A recommendation, where given, is an engineering
view and is conditional; the choice belongs to the named owners.

Required coverage of any option:
- termination and transfer effective times (business time vs recorded time);
- outstanding cases, invoices and pending Finance approvals (who owns them from the effective time);
- a named successor for each open responsibility (never "ownerless");
- access revocation timing (session revocation exists technically in R1);
- historical ownership preserved (original seller, credit owner, event actor unchanged);
- handover evidence (append-only, per item);
- concurrent HR changes (version checks; one final apply);
- independent approvals where required (HR final apply ≠ requester; person-level identity).

### OPD-05 — Termination / transfer handover and access timing

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | OPD-05 |
| Title | Termination / transfer handover and access timing |
| Business question | When an employee is terminated or transferred, what are the effective times, who is the named successor for open cases / invoices / approvals, when is access revoked, and what handover evidence is required? |
| Background | Ownerless work is a P0 risk; historical ownership must never be rewritten. |
| Current behavior | HR change requests exist; no final apply or handover. |
| Evidence references | EV-008, EV-013, EV-044 (see `docs/governance/r5-evidence-index.json`) |
| Source | Cross-Role V1 §19 (OPD-05; M-G07, HR-G02) — P0 impact |
| Authority status | OPEN |
| Missing evidence | Named-successor policy; open-work impact data; real HR role tests. |
| Alternatives | Final apply only after all open work has a named successor and pending approvals are reassigned.<br>Final apply at the effective time; open work moves to the manager queue with escalation.<br>Two-step: access suspended at effective time; ownership handover completed within a defined window. |
| Trade-offs | Access revocation timing vs continuity; concurrent HR changes need version checks; historical attribution stays immutable. |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | organization, identity, custody, billing, finance |
| Business owner | Product + HR + Sales + Finance |
| Required approver roles | Product, HR, Sales, Finance owners |
| Dependencies | SD-04, R5-SEC-01 (person identity), OPD-08 (pending approvals) |
| Affected blocker | P0-6 |
| Readiness | READY FOR HUMAN REVIEW |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Final-apply command with handover; access revocation. |
| Activation gate | HR + Security activation. |

### SD-04 — HR cross-scope transfer destinations / no-op policy

**DRAFT — NOT APPROVED**

| Field | Value |
|---|---|
| Decision ID | SD-04 |
| Title | HR cross-scope transfer destinations / no-op policy |
| Business question | Which destinations are valid for a cross-scope transfer, and when is a transfer a no-op? |
| Background | Normalised destination policy is new. |
| Current behavior | Not implemented. |
| Evidence references | EV-008 (see `docs/governance/r5-evidence-index.json`) |
| Source | Cross-Role V1 §19 (SD-04) |
| Authority status | OPEN |
| Missing evidence | Valid subject / target / parent / chain-scope evidence. |
| Alternatives | Same chain only.<br>Any chain with approval of both managers.<br>HR-only destinations. |
| Trade-offs | Wider transfers need dual approval. |
| Recommended option, if defensible | None — this is a business choice; engineering has no defensible preference. |
| Why the recommendation is conditional | — |
| Risks and compensating controls | See trade-offs; no control replaces the decision. |
| Dependent modules | organization, sales |
| Business owner | Product + HR + Sales |
| Required approver roles | Product, HR, Sales owners |
| Dependencies | OPD-05 |
| Affected blocker | P0-6 |
| Readiness | READY FOR HUMAN REVIEW |
| Approval date |  |
| Named approvers |  |
| Selected option |  |
| Conditions of approval |  |
| Evidence of approval |  |
| Effective date |  |
| Version and supersession | v0.1 DRAFT (R5). Supersedes nothing. Any approval creates v1.0 with a link to its evidence. |
| Implementation follow-up | Transfer command. |
| Activation gate | With OPD-05. |
