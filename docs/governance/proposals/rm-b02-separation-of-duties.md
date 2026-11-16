# RM-B02 — Separation of duties (sensitive actions, person-level independence)

**PROPOSAL — NOT APPROVED. Documentation only.** Baseline `main` @ `24f8a5badcefdfb6506df344bb686d71e5b0d634`. Production **NO-GO**.

## 1. Rules that apply to every workflow below

These rules come from existing sources; this document does not create them.

| Rule | Source |
|---|---|
| One account holding several roles never counts as two approvers | AGENTS.md; R5 P-SEC §1 |
| Two accounts of one human never count as two people | AGENTS.md; R5 P-SEC §1; M2 §6 |
| Person-level independence cannot be enforced today: there is no verified person and no account-to-person link. Finance decision and refund contracts fail closed with `IDENTITY_PERSON_LINK_UNMODELLED` | R5 P-SEC §1 (EV-019, EV-020); R5-SEC-01 |
| Option C of R5-SEC-01 (declarative SoD) does **not** prove human independence; A or B can | R5 P-SEC §2 (engineering view, not a selection) |
| Requester ≠ reviewer ≠ final applier; Workforce Subject is not a reviewer | XR §5, §11 |
| Rule Author, Run Reviewer and Posting Actor are separable; whether they must be different people (and any emergency exception) is OPD-08 | G0 §10; R5 OPD-08 |
| Maker / checker is configurable, not imposed, until decided | G0 INV-025 |

**Consequence:** every workflow below that needs an *independent* approver stays **BLOCKED for real users** until R5-SEC-01 is decided with option A or B **and** implemented, regardless of which grant model D-42 picks (M2 §2-2). This is the single largest SoD blocker.

## 2. Workflow matrix

Legend for "Exists in target": **CMD** = command exists behind flags (granted to nobody); **READ** = only reads exist; **NONE** = no capability or route.

| # | Workflow | Exists in target | Initiator | Reviewer | Independent approver | Executor | Auditor (proposed) | Conflicting role combinations (same person must not hold both for the same item) | Person-level independence requirement | Unresolved dependencies |
|---|---|---|---|---|---|---|---|---|---|---|
| W01 | MIS source modification vs sales use | NONE (MIS import, assignment, repair have no capability) | MIS (ingest, source assignment, maintenance) | MIS source owner + affected Sales (impact) | Not defined; destructive maintenance needs impact + protected guard (MS §13; XR §12 `C impact + protected guard`) | MIS under policy | Product / MIS owner; read audit by Security | MIS operator who repairs or re-links a source row **and** is holder / credited seller of the affected case | Repairer ≠ beneficiary of the case; needs person link | OPD-01, F01, OPD-03, D-30, R5-SEC-01 |
| W02 | Lead distribution vs receipt and reassignment | READ (`custody.custody.read`); allocation / transfer / return BLOCKED | current custody owner (Deputy, Manager, Senior, Supervisor, MIS governed) | — | Not defined for ordinary allocation; skip-level / personal receiving conditional | custody service with apply-time recheck | Sales owner | Distributor allocating to **themselves** ("personal receiving grant from rank" is MUST NOT OWN for Senior, XR §2); distributor and recipient re-assigning in a loop to reset history | Allocator ≠ recipient when the allocator is above the recipient tier, unless an approved personal-receiving path exists | OPD-10 (SD-01…03), F01, ADR-01, OPD-03 |
| W03 | Sales conversion vs financial confirmation | Partial: contact outcome CMD; conversion / ready / issuance NONE; Finance decision NONE | Seller (conversion, evidence) | Finance reviewer | Finance (authorized stage decision) | finalizer under policy | Finance + Sales owners | Any Sales role on the case **and** Finance decision on the same invoice | Finance decider ≠ seller, issuer, submitter or assisting supervisor of that invoice | ADR-01, OPD-08, NO_DECISION_CAPABILITY (R4 gap), R5-SEC-01 |
| W04 | Invoice creation vs payment verification | issuance NONE; batch submit CMD; verification NONE (`finance.review.read` only) | Seller issues and submits evidence | Finance reviewer | Finance | finalizer | Finance owner | issuer / submitter / uploader **and** reviewer of the same stage; also `GATEWAY_AUTO` must never display as Finance-reviewed (D-25) | Reviewer ≠ issuer ≠ submitter, per person | ADR-01, D-32, OPD-08, D-43, D-25, R5-SEC-01 |
| W05 | Finance review vs finance execution / posting | NONE (decision, posting, run approval absent) | Rule Author / engine run | Run Reviewer | per OPD-08 thresholds | Posting Actor via approved engine | Finance owner; external audit | Rule Author + Run Reviewer + Posting Actor on the same run (G0 §10); F06 today lets finance-view perform sensitive writes in legacy | Distinct people if OPD-08 requires; emergency exception path is OPD-08 | OPD-08, D-41, OPD-06, OPD-09, F06, R5-SEC-01 |
| W06 | Refund request vs refund approval | NONE (R4 refund state machine: every transition BLOCKED) | Sales / after-sales (request / context) | Finance | Finance (authority per OPD-04) | Finance with bank / provider proof | Finance + Legal (compensation) | requester **and** approver; approver **and** executor where OPD-04 requires; recipient of funds **and** any role on the refund | Approver ≠ requester; person-level independence fail-closed in R4 | OPD-04, OPD-08, D-41, D-44, R5-SEC-01 |
| W07 | HR request vs final HR application | create + step review CMD; final apply NONE | Supervisor / Senior / Manager (Deputy conditional) | current reviewer chain (parent of requester) | HR (final authority) | HR final apply (no capability) | HR owner; Sales owner for handover | requester **and** step reviewer; subject **and** reviewer; requester **and** HR final applier; subject's beneficiary successor **and** applier | Reviewer ≠ requester ≠ subject per person; today checked only per account (snapshot equality) | OPD-05, SD-04, R5-SEC-01 |
| W08 | Credential administration vs account approval | NONE (no provisioning, credential or grant command) | HR or Security (per D-34 / D-42 option) | Security (D-42 option 2) | per D-42 | identity command | Security owner; periodic grant review (owner open, M2 Q5 / D-42 question) | person who requests an account **and** grants its capabilities **and** approves the grant; HR granting to itself (D-42 option 3) | Grant approver ≠ grant requester ≠ grantee | D-34, D-42, E-7, R5-SEC-01, ADR #18 |
| W09 | Commission calculation vs approval and settlement | READ (shadow runs, policies, own credits); approve / post / settle NONE | engine (system) with HR effective inputs | Finance Run Reviewer | Finance per OPD-08 | approved engine posting; settlement per proof | Finance; HR for input integrity | recipient of credit **and** run reviewer / poster; HR person editing own compensation input **and** being a recipient; rule author **and** reviewer | Reviewer / poster ≠ recipient; HR input editor ≠ recipient of the affected period | OPD-09, D-36, D-41, OPD-08, R5-SEC-01 |
| W10 | Biawin activation vs provider reconciliation | activation SHADOW only (read); provider reconcile CMD (any proven user) | activation decision (system, SHADOW) | — | not defined | provider dispatch (no production connector) | Product + Finance | person who decided / dispatched an operation **and** records its reconciliation evidence; reconciler **and** beneficiary of the external effect | Reconciler independent of the activation path; **no such check exists** in `integration.provider_operation.reconcile` | D-35a … D-35e, D-38, R5-SEC-01, GAP-06 (operations actor) |
| W11 | Legacy write-back vs verification | write-back via workers behind `_writeback` flags; diagnostics READ (`integration.writeback.read`) | system worker after a target command | operator verification of reconciliation | not defined | worker | Engineering + Human Owner | operator who enables a write-back flag **and** verifies its results; author of the target command **and** verifier of its write-back | Flag changer ≠ verifier; flag changes have no capability or audit today (E-7) | E-3, P1-8 (engineering gap), E-7, ADR-01 |

## 3. Combinations the matrix proposal deliberately avoids

The [matrix](rm-b02-role-capability-matrix.md) proposes no single role with both sides of any workflow above:

| Workflow | Side A proposed for | Side B proposed for | Overlap in one role? |
|---|---|---|---|
| W04 | Seller: `payments.upload.stage`, `payments.batch.submit` (C) | Finance: `finance.review.read` (A, read only; no decision capability) | No |
| W07 | Supervisor / Senior / Manager: create and step review (A) | HR: final apply (no capability; HR request cells DEFERRED) | No. **But** the same tier both creates and step-reviews **different** requests; the engine routes review to the requester's parent, so a person never reviews their own request at account level |
| W09 | Sales tiers: `commission.own.read` (A) | Finance: `commission.shadow.read` (A, read) | No write on either side exists |
| W10 | — | `integration.provider_operation.reconcile`: DEFERRED for every role | No role proposed |
| W08 | HR: `identity.user.read` (A, read) | grant administration: no capability | No; D-42 option 3 would create one (see decision package) |

**Multi-role risk.** If one person holds two roles (for example Finance + Sales tier, or HR + Sales tier), the union evaluator would give that person both sides. The matrix cannot prevent that; only a multi-role policy plus person-level SoD can (decision package §4).

## 4. SoD blockers (summary)

| Blocker | Blocks | Can any proposed cell proceed without it? |
|---|---|---|
| R5-SEC-01 undecided / unimplemented | W03–W09 independent approvals; any Finance, refund, HR-apply or grant-approval action | Yes for non-SoD actions only (reads; contact outcome; HR request create / step, which have no final effect) |
| OPD-08 undecided | W05, W06, W09 thresholds and emergency path | No Finance decision can be designed |
| E-7 absent | W08, W11 (grant and flag changes unaudited) | No grant can be issued (G1) |
| OPD-05 / SD-04 undecided | W07 final apply | Create / step review can run; nothing is applied |
| No capability for the independent side | W01, W02, W06, W08, W09 entirely; W03–W05 Finance side | — |
| No operations actor among the eight roles (GAP-06) | W10 | Provider reconciliation stays unassigned |

## 5. Audit expectations (proposal; derived from FS §16, XR §17, S11–S14)

For every sensitive decision, the record must keep: actor (human or system) and on-whose-behalf, role basis used, item and version, before / after, reason, evidence reference, correlation, result (applied / skipped / rejected / failed / unknown) and the independence check performed (which persons were compared). Corrections append and link; nothing is overwritten. Unknown outcomes are reconciled before any retry (G0 INV-034; AGENTS.md).
