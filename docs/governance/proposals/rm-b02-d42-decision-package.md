# RM-B02 — D-42 decision package and dependent decisions

**DRAFT — NOT APPROVED. No option is selected. No status changes.** Baseline `main` @ `24f8a5badcefdfb6506df344bb686d71e5b0d634`. Production **NO-GO**.

This package does not create a decision ID, an approval authority or an approval record. Tracker statuses in `docs/governance/r6-decision-tracker.json` are quoted, not changed. Recording any outcome follows `docs/remediation/r6-approval-evidence-standard.md` and `r6-human-approval-workflow.md` (approval record under `docs/governance/approvals/` plus independent human verification). The blank intake forms for D-42 already exist in `docs/governance/intake/m2/`.

## 1. Scope boundary (read first)

The RM-B02 brief asks D-42 to cover the role-capability matrix and the multi-role policy. The governing record is narrower:

| | Text | Source |
|---|---|---|
| D-42 question (R5, authoritative English) | "Can HR edit sales capabilities?" — "May HR grant or edit sales capabilities (portal_access bulk / inline), or does identity own all grants?" | R5 inventory row D-42; M2 §3-1 |
| M2 position | D-42 decides **only the authority and procedure** to grant, edit and revoke. It does **not** decide which role is eligible for which capability, or read scope. "D-42 is not widened." | M2 D-42 seller analysis §2; M2 intake §3-1, §4 |

**Handling (no winner chosen):** §2 presents D-42 exactly as R5 defines it. Every other question the brief raised is presented in §3–§4 as a **separate proposed record without an ID** (M2 rule: IDs are assigned only by the authorized inventory owner, itself open question M1 Q2). Approving D-42 approves none of them. Recorded as conflict CF-01 in the matrix.

## 2. D-42 — Grant authority

| Field | Value |
|---|---|
| Decision ID | D-42 (unchanged) |
| Tracker status | `READY_FOR_REVIEW`; verification `NOT_STARTED`; priority P1; blocker P1-4; package P-SEC |
| Why needed | No ACTIVE grant can exist (`role_grant_shadow_only`); every positive business action of a real user is refused (R7 §5: D-42 "blocks every positive user action in production"). Before any grant command is built (RM-B03), someone must be authorized to grant, edit and revoke |
| Current safe behaviour | HR cannot grant; identity owns grants; no grant-change procedure exists (E-7) |
| Legacy evidence | In 2.0.160, HR edits portal access in bulk or inline (EV-007) |
| Options (R5, verbatim; none selected) | ☐ 1. Identity / Security only · ☐ 2. HR may request; Security approves · ☐ 3. HR may grant within a capped set with audit |
| R5 engineering recommendation | **None** — business choice. This package adds none |
| R5 trade-off note | "Separating request and grant supports SoD" |
| Accountable owner (R5 business owner) | Product + HR + Sales |
| Required approvers (R5) | Product owner, HR owner, Sales owner; Security — each separately |
| Prerequisites (R5) | R5-SEC-01, E-7 |
| Evidence missing (R5) | Grant-change audit procedure (E-7) |

**Consequences of each option** (from M2 §3-1; restated, not extended):

| Option | Business | Security | What RM-B02 adds |
|---|---|---|---|
| 1 | HR loses direct access editing (differs from 2.0.160); changes may be slower | One authority, simple audit; single point of compromise | Consistent with every matrix cell; no HR grant power |
| 2 | Needs a request workflow | Maker ≠ checker, but only per account until R5-SEC-01 | Matches SoD workflow W08; HR keeps only `identity.user.read` (A) |
| 3 | Closest to legacy, fastest | Over-granting inside the set; audit after the fact; the capped set must be defined separately | **The matrix is not a capped set.** Defining one needs the per-role eligibility records (§3) first. Option 3 would put grant power and HR's own workforce reads in one role: W08 requires that a person cannot grant to themselves |

**Questions to answer with D-42** (M2 Q5, Q6; still open): Does revocation (especially at exit or transfer) have the same authority as grant, or a separate emergency path? What exactly is the capped set in option 3, and who defines it? Who performs periodic grant review?

**Evidence needed for closure:** E-7 procedure design (grant / revoke / review with audit and append-only history); M1 answer on signing authority (Q1) and on the inventory owner (Q2); authority documents for each approver (`r6-approval-evidence-standard.md`); an agreed path through the circular dependency R5-SEC-01 ↔ D-34 / D-42 (M2 §2).

**Consequences of deferral:** no real user can perform any business action in the target; legacy 2.0.160 remains the only working system; RM-B03 stays blocked.

**Readiness assessment:** D-42 is **reviewable** (options, consequences and evidence are documented) but **not closable**: E-7 evidence is missing, the circular dependency is unresolved, M1 authority answers are missing, and option 3 cannot be evaluated until a capped set exists.

## 3. Separate questions the brief raised (proposed records, **no IDs**)

| Label (not an ID) | Question | Why needed | Evidence available | Options | Recommendation (only where evidence supports) | Proposed owner / approvers (to be confirmed under M1) | Evidence needed for closure |
|---|---|---|---|---|---|---|---|
| M2 proposed record 1 | Which capabilities is the `seller` role eligible for (first flow)? | D-42 decides who grants, not what | M2 §4-1; matrix SEL column | A: `cases.case.read` + `cases.contact_outcome.record`; B: read only; C: none yet | M2 engineering proposal (not a selection): A, **only if** record 2 picks a restricted scope enforced on the server first. RM-B02 matrix is consistent with that and adds C cells for invoices, attempts, uploads, batches, submissions and customer read, and A for `commission.own.read` | M2: Product + Sales; Security | registration by the inventory owner; record 2 decided |
| M2 proposed record 2 | Which cases does a seller see on every data-returning endpoint? | every read is organization-wide today | M2 §4-2 definitions | 1 current holder; 2 current legacy assignment; 3 current + historical; 4 organization; 5 decide later | M2 engineering proposal: 1 (aligned with the command's holder guard; fail closed on unknown). Choosing 1 is a deliberate change from 2.0.160, which shows current assignment | M2: Product + Sales + Security; open: Legal (privacy) | `legacy_holder_*` vs `sn_leads.seller_id` evidence (M2 Q10); endpoint inventory |
| GAP-01 | Capability eligibility for Supervisor, Senior, Manager, Deputy, MIS, HR, Finance and `invoice_reviewer` | M2 record 1 covers the seller only | this matrix (columns SUP…FIN; §8) | per-role bundles as in the matrix; or narrower | Use the matrix as the evidence base; decide per role, never by rank | Product + the role's business owner; Security (R5 pattern for role decisions) | per-role review of the matrix column; runtime tests with real role accounts for HR and Finance |
| GAP-02 | Read scope for Supervisor, Senior, Manager, Deputy (direct team, subtree, territory) | 46 matrix cells carry `RS` | SS §22, MG §15, DP §15, G0 §19, XR §2 | DIRECT_TEAM / AUTHORIZED_SUBTREE / GOVERNED_TERRITORY / organization | None beyond the sources' own wording ("no entire-org assumption", DP §15) | Product + Sales; Security; open: Legal | server-side scope design covering all endpoints and counts |
| GAP-03 | Finance row scope for billing / payment / evidence reads | registry cites D-25 / D-26 / D-28, which decide other questions (CF-04) | FS §15 `view_scoped_financial_data` ("data scope independent of view label") | organization-wide financial domain; linked-to-queue only; per-team | None | Finance + Product; Security | Finance confirmation; HR / Finance runtime NOT LIVE VERIFIED must be closed by real-account tests |
| GAP-04 | Who may read customer identity, with which fields | registry ties reads to D-30 (merge authority, CF-05) | DV per role; MS §7 | case-linked scoped read per role; MIS org-wide provenance; none | None | Product + MIS + Sales (D-30 owners); Legal (privacy) | field policy for phone / identifiers |
| GAP-05 | Multi-role and active-context policy (§4) | union evaluator; RM-E02 MULTIPLE state has no selection policy | RM-ACCESS-01 test J; RM-E02 §5 | §4 | §4 (partial) | Product + Security | §4 evidence list |
| GAP-06 | Which actors own fulfillment work, activation and provider reconciliation | 58 of the 84 DEFERRED matrix cells sit on the 9 Biawin, fulfillment and provider capabilities; operations roles UNMAPPED; XR §7 forbids inventing a ninth role | S14 §5 (per-service scope); mapping.go | map to an existing business role; keep as per-service scope only (no business role); defer | None | Product + Finance (owners of D-35a … D-35e) | product contract for operations (none exists) |
| GAP-07 | MIS row / source scope predicate | MIS cells carry `MISS` | MS §14 | organization-wide source data; governed subsets | None | Product + MIS | MIS confirmation |
| GAP-08 | Definition of GOVERNED_TERRITORY | DP §15 names it without a predicate | DP §15; DV Deputy | equal to authorized subtree; explicit territory assignment | None | Product + Sales | — |

## 4. Multi-role and hierarchy policy (GAP-05 and related)

Current facts: grants are evaluated as a **union**; every matched role is returned with no selection and no escalation (RM-ACCESS-01 test J); RM-E02 shows a MULTIPLE state and deliberately has **no role switcher**; delegation is not modelled (R5 P-SEC §1); grants have no interval or history yet (ARCH §18 proposes `RoleGrant (interval, granted_by, reason)`); person identity is not modelled.

| Topic | Options | Consequences | Recommendation (only if evidence supports) | Decided by |
|---|---|---|---|---|
| Users with more than one role | (a) union of all grants; (b) one explicit active role per session; (c) union for reads, explicit context for writes; (d) grant-time refusal of incompatible combinations | (a) simplest, but one person may hold both sides of W03–W09; (b) safer, needs a server-side context and UI; (c) mixed; (d) prevents SoD collisions at source | **Supported by evidence:** whatever model is chosen, it must not let one person satisfy both sides of an SoD workflow (AGENTS.md; R5 P-SEC). (d) plus decision-time person checks is the only listed option that enforces this; it needs R5-SEC-01. Between (a)/(b)/(c): no recommendation | GAP-05 (Product + Security) |
| Several organizational memberships | (a) keep "exactly one current parent / one LINKED ACTIVE employment", else BLOCKED; (b) multiple memberships each with its own scope | (a) today's command behaviour (S11), fails closed; (b) needs membership-qualified scopes | **Supported:** keep (a) until decided (S11 behaviour; AGENTS.md unknown → conflict) | GAP-05 with HR |
| Role selection / active context | server-decided context; user-selected context bound to session; none (union) | a client-side selection is never authority (RM-E02 §6) | If adopted, the server must decide or validate it; no client-held role | GAP-05 |
| Temporary delegation | none; time-bound capability-limited delegation | delegation without person identity cannot prove who acted | Any delegation must be time-bound, capability-limited and audited (ARCH §17 intent) and must never count as an independent approval | OPD-10 (+ R5-SEC-01) |
| Acting on behalf of another person | forbid; allow with on-behalf record | event actor must stay the real actor (XR §5) | Record both the real actor and the represented person; never rewrite the event actor | OPD-10 |
| Supervisor / manager inheritance | none; explicit per-capability scope using hierarchy as input | — | **Supported:** no blanket inheritance (XR §1, §2, §12; ARCH §17–18; lint `s11-rank-shortcut`) | already fixed by sources |
| Role expiration and revocation | same authority as grant; separate emergency revocation; automatic at termination | revocation slower than grant leaves access after exit | Grants should carry an interval and append-only grant / revoke history (M2 §6 items 1–2, engineering proposal). Who revokes: M2 Q5, open under D-42 | D-42 (authority), OPD-05 (timing at exit) |
| Organizational transfers | scope follows current hierarchy from the effective time; or holds until handover | open work could become ownerless (INV-030) | History is never rewritten (XR §5); access timing is OPD-05 / SD-04 | OPD-05, SD-04 |
| Emergency access | none (status quo); break-glass with post-hoc review; pre-authorized emergency role | break-glass without person identity is unattributable | No source authorizes emergency access outside OPD-08's Finance maker/checker exceptions; if introduced, it must never approve an SoD action | new record if needed; OPD-08 for Finance |
| Independent approval conflicts | account-level check; person-level check | account-level is insufficient (R5) | **Supported:** person-level (R5-SEC-01 A or B) for every independent approval | R5-SEC-01 |
| Historical audit preservation | record role basis at decision time; recompute later | recomputing from current grants rewrites history | **Supported:** record actor, role / grant basis and scope at decision time; later grant changes never rewrite past events (XR §5; R5 P-SEC §1 "Historical identity evidence") | engineering under E-7 |

Not done here: no role switcher, no evaluator change, no context field.

## 5. Dependent decisions (statuses quoted from the tracker; unchanged)

Columns "Owner" and "Approvers" are R5's columns, quoted. "Recommendation" is R5's engineering view where one exists; otherwise none.

| Decision | Tracker status | Why it matters to RM-B02 | Evidence available | Options (R5) | Recommendation | Owner (R5) | Approvers (R5) | Evidence for closure (R5 missing evidence) | Consequence of deferral |
|---|---|---|---|---|---|---|---|---|---|
| R5-SEC-01 (proposed ADR) | READY_FOR_REVIEW | person-level SoD for every independent approval (G4) | R5 P-SEC §1–2 | A HR registry / B external IdP / C declarative / D keep blocked | R5 conditional view: A or B satisfy SoD; C does not | Security + HR | Security, HR, Legal (privacy), Product | HR identity source; verification method; privacy review | W03–W09 stay blocked |
| E-7 | engineering prerequisite (not a tracker record), OPEN | grant / flag change with audit (G1) | R5 inventory | — | — | Engineering | Engineering + Security / Ops review | runtime evidence | no grant can be issued safely |
| D-34 | READY_FOR_REVIEW | real accounts (G3) | R1 | one-time password + forced change / invitation link / external IdP | none | Security + HR | Security, HR | security design (EV-029, EV-035) | no real user beyond R1 test accounts |
| ADR #18 | READY_FOR_REVIEW | real-data access; HR / Finance live verification | R5 | synthetic only / anonymised / restricted full extract | none | Product + Security | Product, Security, Legal | privacy review | LIV conditions cannot close |
| OPD-10 | READY_FOR_REVIEW | assist, skip-level, personal receiving, delegation (O10; MC-16, MC-21, MC-31) | XR §6 | preserve legacy paths / assisted issuance with record / disallow | none | Product + Sales + Finance + HR | same four | per-action legacy guard inventory | tier write cells stay DEFERRED |
| ADR-01 | READY_FOR_REVIEW | invoice issuance vs return (MC-14, MC-17) | R4 gates | A / B / C | none | Human Owner | Human Owner + Engineering lead | real-data link coverage | no issuance |
| F01 | EVIDENCE_MISSING | protected dependencies for return / reset | XR §8 | accept / accept with coverage / defer | none | Sales + MIS + Finance + Engineering | same | link adapters on real data | no return |
| OPD-03 | READY_FOR_REVIEW | cancelled release / archive revival (MC-17, MC-20) | R4 classifier | three options | none | Product + Sales + MIS + Finance | same | cancelled-case policy | no release / revival |
| D-32 | EVIDENCE_MISSING | next-stage authority (MC-15) | Slice 12 | keep 2.0.160 / hierarchy-based / Finance-only | none | Product + Sales + Finance | same | billing lifecycle contract | BLOCKED |
| D-29 / D-31 | READY_FOR_REVIEW | DUPLICATE outcome (MC-22) | S11 | three options | none | Product + Sales | same | duplicate workflow description | value stays BLOCKED |
| D-30 | EVIDENCE_MISSING | merge authority; cited by registry for customer reads (CF-05) | R5 | no merges / MIS with evidence / named role + independent review | none | Product + MIS + Sales | same | real-data resolution report | customer cells stay CONDITIONAL / DEFERRED |
| OPD-01 | EVIDENCE_MISSING | lineage for MIS / case reads | R5 | keep UNKNOWN / MIS aliases / rules | none | Product + MIS + Sales + Engineering | same | real-data proof | coverage conditional |
| OPD-07 | EVIDENCE_MISSING | metric definitions (O07) | XR §13 | event-range / snapshot / both | none | Product + Sales + MIS + Finance | same | cohort metadata | metrics NOT FINAL |
| OPD-08 | READY_FOR_REVIEW | maker / checker, emergency exceptions (W05, W06, W09) | R4 contract | single reviewer / thresholds / emergency | none | Product + Finance | Finance; Product; Security | org policy; P-SEC | no Finance decision |
| D-43 | READY_FOR_REVIEW | approval with non-READY receipts | contract | never / exception / gateway only | none | Finance | Finance | — | safe default stays |
| REJECT-VOCAB (R4 gap) | READY_FOR_REVIEW | rejection reasons | R4 | three options | none | Finance + billing owner | Finance; Product | Finance reason list | rejection BLOCKED |
| D-25 | READY_FOR_REVIEW | gateway auto-approval provenance (W04) | R4 | three options | none | Finance | Finance (+ Security) | real gateway adapter | synthetic only |
| D-41 | EVIDENCE_MISSING | ledger basis for posting (MC-10) | Slice 9 | accrual / cash / both | none | Finance | Finance | real-data parity | no posting |
| OPD-09 | EVIDENCE_MISSING | commission engine routing (W09, MC-11) | Slice 13 | three options | none | Product + Finance + HR + Engineering | same | active routes on real data | shadow only |
| D-36 | EVIDENCE_MISSING | bundle commission split | Slice 13 | three options | none | Finance + Product | same | historical parity | unkeyed |
| OPD-04 | READY_FOR_REVIEW | refund authority (W06) | R4 state machine | three options | none | Product + Finance | Product, Finance; Legal | proof standard | all refund transitions BLOCKED |
| D-44 | READY_FOR_REVIEW | evidence retention (D44) | R2 | three options | none | Finance + Legal | same | legal review | purge report-only |
| D-23 | READY_FOR_REVIEW | invoice review visibility / export (§8) | Slice 8 | no export / scoped / named roles | none | Product + Finance | Product, Finance; Security / Legal | privacy review | invoice review unassigned |
| D-24 | READY_FOR_REVIEW | invoice review owner | — | Product / Finance | none | Product | Product | — | no UI |
| D-45 | READY_FOR_REVIEW | sensitive-read audit failure | Slice 8 | fail closed / serve and alert / masked | safe default: fail closed (R5 "safe default available") | Product + Security | same | — | fail closed stays |
| OPD-05 | READY_FOR_REVIEW | HR final apply and access timing (W07, MC-04) | S11 | three options | none | Product + HR + Sales + Finance | same | successor policy; real HR tests | HR request cells DEFERRED |
| SD-04 | READY_FOR_REVIEW | transfer destinations | S11 | same chain / any chain with both managers / HR-only | none | Product + HR + Sales | same | scope evidence | out-of-scope destinations BLOCKED |
| D-35a … D-35e | DRAFT (C-03 mapping unresolved) | Biawin / fulfillment cells (GAP-06) | S14 | per sub-item | none | Product + Finance | Product, Finance | C-03 mapping confirmation | Biawin cells DEFERRED |
| D-38 | READY_FOR_REVIEW | wallet charge (MC-34) | S14 | three options | none | Finance + Product | same | provider contract | BLOCKED |
| D-33 | READY_FOR_REVIEW | operations upgrade payment (MC-36) | — | not granted / with Finance review / named roles | none | Product + Finance | same | — | not granted |
| OPD-02 | READY_FOR_REVIEW | ACK / SLA / escalation owner (MC-33) | XR §6 | three options | none | Product + Sales + MIS + HR | same | owner / path evidence | no escalation contract |

## 6. Decision readiness summary

| Item | Ready for human review? | Closable now? | Main reason |
|---|---|---|---|
| D-42 | Yes (tracker READY_FOR_REVIEW; this package adds the matrix as evidence) | **No** | E-7 evidence, circular dependency with R5-SEC-01 / D-34, M1 authority, option 3 capped set undefined |
| M2 proposed records 1 and 2 | Content yes | **No** | not registered; no ID (M1 Q2) |
| GAP-01 … GAP-08 | Content drafted here | **No** | not registered; several need product contracts first (GAP-06) |
| R5-SEC-01 | Yes | **No** | HR identity source, verification method, privacy review |

## 7. What can proceed safely without any decision

- RM-E03 (seller read screens) **can proceed with synthetic test data only**: destinations appear only with an ACTIVE grant plus the module flag (RM-ACCESS-01 §11); until D-42 and the eligibility / scope records are decided, screens can be exercised only with synthetic fixtures or the dev stub (refused in production). Screens must not assume organization-wide case read is approved, must not implement row scope in the client, and must render the server's refusal.
- Engineering design documents for E-7 (append-only grant history), server-side row scope (MC-30) and person identity, without enabling anything.
- Negative tests proving that today no ACTIVE grant can exist.

Not allowed before decisions: any grant, bundle change, ACTIVE grant basis migration, scope implementation claimed as approved, or activation.
