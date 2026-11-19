# RM-E04H — R5-SEC-01 person-identity decision package (F1 unblocking)

**DRAFT — NOT APPROVED.** Human decision-support package for R5-SEC-01 ("Authoritative person identity and
account-to-person linkage"). It selects no option, records no decision, appoints no one, creates no identity or
account link and changes no status, gate, grant, schema or runtime. **Production: NO-GO. Group B: BLOCKED.
S4: BLOCKED.**

| | |
|---|---|
| Baseline | `main` @ `19ab171f2c2db9aa3e88dfc0ce4a13bb3d0f4657` (merge of PR #80, RM-E04G) |
| Branch | `docs/rm-e04h-person-identity-decision` (isolated worktree from `origin/main`) |
| Decision | R5-SEC-01 (proposed ADR): tracker `READY_FOR_REVIEW`, `approval_record` empty, verification `NOT_STARTED` — **unchanged** |
| Executive dashboard (Persian) and blank form | `docs/governance/proposals/rm-e04h-person-identity-decision-dashboard.fa.md` |
| Review handoff | `docs/implementation/rm-e04h-review-handoff.md` — REVIEW PENDING |
| Tests | `docs/governance/rm_e04h_test.go` |
| Reused, not repeated | RM-E04F architecture (`docs/governance/proposals/rm-e04f-authority-role-assignment-architecture.md`), RM-E04G F1 analysis (`docs/security/rm-e04g-f1-person-identity-independence.md`), RM-E04E authority packet |

Status vocabulary used below: **DOCUMENTED** (stated in a repository source), **PROPOSED** (this package's
engineering proposal), **VERIFIED** (checked against code or a test in this PR), **PENDING** (waiting for a named
human or reviewer), **BLOCKED** (cannot proceed until a named dependency is met).

---

## 1. Preflight

| Check | Result | Status |
|---|---|---|
| `origin/main` HEAD | `19ab171f2c2db9aa3e88dfc0ce4a13bb3d0f4657` = expected baseline | VERIFIED |
| PR #80 merged | `19ab171` is "Merge pull request #80 … fix/rm-e04f-person-identity-independence" | VERIFIED |
| Worktree | fresh worktree from `origin/main`, zero changes before this work | VERIFIED |
| Baseline `go test -count=1 ./docs/governance/` | PASS | VERIFIED |
| `docs/governance/approvals/` | absent — zero R6 records | VERIFIED |
| Group B gate / S4 manifest | both empty (closed) | VERIFIED |
| Nomination register | all `NOMINATED` / `UNASSIGNED`; nominee `person_ref` empty; T5 `UNASSIGNED` | VERIFIED |

---

## 2. Authoritative source audit and option fidelity

### 2.1 Where R5-SEC-01 is defined

| Source | Content | Weight |
|---|---|---|
| `docs/remediation/r5-person-identity-sod-DRAFT.md` §4, record R5-SEC-01, field **Alternatives** | Full option wording A–D (quoted in §2.2) | **Authoritative wording** (the R5 decision record) |
| same file, §2 comparison table | Short headings and trade-off grid; "Engineering view (conditional, not a selection)" | Engineering analysis, not authority |
| `docs/remediation/r5-decision-inventory.md` row R5-SEC-01 | Owner "Security owner + HR"; approvers "Security, HR, Legal (privacy), Product"; evidence EV-019, EV-020; dependencies D-34, D-42, ADR #18; blockers P0-7, P0-6 | DOCUMENTED |
| `docs/governance/r5-evidence-index.json` EV-019, EV-020 | EV-019: SoD proven only per account (`IDENTITY_PERSON_LINK_UNMODELLED`, R4 finance contract). EV-020: `identity.user_account` one row per legacy user; negative claim "no person entity" explicitly *not* proven by the anchor | DOCUMENTED (`TEXT_AT_COMMIT_VERIFIED`) |
| `docs/governance/r6-decision-tracker.json` | `R5-SEC-01 (proposed ADR)`: `READY_FOR_REVIEW`, meeting M2, prerequisites "D-34, D-42, ADR #18", outcome empty | DOCUMENTED |
| `docs/remediation/r6-m2-security-session-fa.md` | Persian M2 agenda; options as unticked boxes ☐ الف / ب / ج / د | DOCUMENTED (translation) |
| `docs/remediation/r6-m1-governance-data-session-fa.md` | Lead owner for R5-SEC-01: blank ("to be set by management") | DOCUMENTED — **UNASSIGNED** |
| `docs/governance/intake/m2/R5-SEC-01_proposed_ADR-v1.0-DECISION.BLANK.json` | `independent_approvals`: Security owner, HR owner, Legal owner, Product owner | DOCUMENTED |
| `docs/remediation/r6-approval-evidence-standard.md`, RM-E04E §1–§3 | Authority evidence, recorder rules, T5 checklist | DOCUMENTED |
| RM-E04F §6.1, §11, §13 | `resolve(user_id, at)` contract; R5-SEC-01 as hard dependency; C / D leave independence UNKNOWN | PROPOSED (earlier package) |
| RM-E04G §4–§7 | F1 reproduction, evaluator S1–S12, interim rule RM-E04G-I1, routes RT-1…RT-6 | PROPOSED + test-enforced |

### 2.2 Original options (verbatim from the R5 record)

| Id | Original wording (R5 record, field "Alternatives") | Original trade-off (R5 record) |
|---|---|---|
| A | A — HR-verified Person registry: HR records a Person per employee; accounts linked by a reviewed, reversible link with evidence. | A: fits existing HR data; risk of false merges (control: two-person reviewed links, reversible, append-only). |
| B | B — Security-managed identity provider (external IdP) as the person authority; accounts federated. | B: strongest verification; new infrastructure and legal review. |
| C | C — Declarative SoD only: accounts asserted distinct by an approver; no Person entity. | C: cheapest; does not prove independence. |
| D | D — Keep blocked: no person model; SoD-dependent actions stay BLOCKED. | D: blocks P0-7 decisions that need SoD. |

`TestRME04HOriginalOptionsQuotedVerbatim` fails if this table drifts from the source record.

### 2.3 Fidelity findings

| # | Finding | Classification |
|---|---|---|
| OF-1 | The §2 grid heading for B reads "External identity provider (IdP)"; the record reads "Security-managed identity provider (external IdP)"; the Persian M2 text reads "سامانهٔ هویت تحت مدیریت امنیت" (Security-managed, "external" omitted). Same option; the record wording governs | Minor wording variance — no meaning change |
| OF-2 | No other wording of A–D exists in the repository; no option was ever pre-selected (tracker `selected_option` empty; M2 boxes unticked) | VERIFIED |
| OF-3 | The R5 record's recommendation is *conditional*: "A or B satisfy the stated SoD requirement; C does not prove human independence. No option is selected." | DOCUMENTED (engineering view, not authority) |
| OF-4 | **Dependency cycle.** R5-SEC-01 lists D-34, D-42, ADR #18 as dependencies (tracker field `prerequisites`), while D-34 and D-42 each list R5-SEC-01 as their dependency. Read strictly as prerequisites this is circular. Engineering reading: R5-SEC-01 is upstream; D-34 / D-42 are downstream consumers; ADR #18 is a *co-requisite* only if option A needs real HR data in staging | **CONFLICT** — recorded, not resolved; question HQ-10 (§8.1) |
| OF-5 | EV-020's negative claim ("no person entity exists") is marked unproven by the evidence index itself. In fact `organization.person` exists (§3); it is a legacy HR profile row, not a verified human. The R5 statement is therefore *imprecise*, not false in effect: no **verified** person entity exists | Clarification (DOCUMENTED + VERIFIED against migrations) |

---

## 3. Current identity data model (audit, no personal data)

Classification key: VERIFIED_SOURCE · LEGACY_IMPORTED · UNVERIFIED · UNKNOWN · CONFLICT · NOT_IMPORTED.
Field-level classification refers to *what the value can prove about a human*, not to import correctness.

### 3.1 Source-to-target mapping

| Target field | Source | Classification | Can it establish person identity? |
|---|---|---|---|
| `identity.user_account.user_id` | generated uuid per legacy account (`source_system`, `source_key` unique) | LEGACY_IMPORTED | **No** — an account, not a human |
| `identity.user_account.display_name` | legacy profile display name | LEGACY_IMPORTED / UNVERIFIED | **No** (presentation only) |
| `identity.user_account.presence` | `PRESENT` / `ABSENT_IN_LATEST` | LEGACY_IMPORTED | No — import presence, not revocation |
| `identity.user_account.account_state`, `login_enabled` | constrained to `SHADOW_IMPORTED`, `false` | VERIFIED (DDL) | No |
| `organization.person.person_id` | generated uuid **per legacy HR profile** | LEGACY_IMPORTED | **No** — candidate anchor only; one human with two profiles yields two `person_id`s (undetectable) |
| `organization.person.display_name` | legacy HR profile | LEGACY_IMPORTED / UNVERIFIED | No |
| `organization.employment.person_id` | `UNIQUE` → exactly one employment per person row | LEGACY_IMPORTED | No; no employment history (transfers overwrite on refresh) |
| `organization.employment.user_id` | legacy profile → account pointer; **not a foreign key** | LEGACY_IMPORTED / UNVERIFIED | No — a legacy pointer, never reviewed |
| `organization.employment.user_link_state = LINKED` | profile points to one imported account used by no other profile | UNVERIFIED | Evidence only |
| `… = SHARED_USER` | account referenced by >1 profile (`imports.go`) | **CONFLICT** | Never; cannot sign |
| `… = USER_NOT_IMPORTED` / `USER_NOT_RECORDED` | pointer missing / account absent | **UNKNOWN** / NOT_IMPORTED | No |
| `organization.employment.employee_code` | legacy, nullable, **no uniqueness constraint** | UNVERIFIED | No |
| `employment_status`, `hire_date`, `termination_date` (+ `_raw`) | legacy HR values, mapped; `UNMAPPED` / `NOT_RECORDED` allowed | LEGACY_IMPORTED | No (status evidence only) |
| `organization.org_assignment(_history)` | reporting-line history with raw effective dates | LEGACY_IMPORTED | No — hierarchy ≠ identity |
| `identity.credential` (`login_name`, `status`) | none written (application role SELECT-only; D-34 open) | NOT_IMPORTED | No — a credential proves possession, not who |
| `identity.session`, `identity.auth_event` | runtime authentication history per account | VERIFIED (DDL) | No — account-level only |
| `authorization.Subject` (`PrincipalID`, `UserID`, `Grants`, `EvidenceConflict`, `Org`) | session + grants | VERIFIED (code) | No person field; `EvidenceConflict` fails closed on grant-evidence ambiguity |
| Phone, e-mail, GitHub login, PR reviewer | — | — | **Never** (AGENTS.md; RM-E04G T13) |
| National id / identity-document data | not present anywhere in the schema | NOT_IMPORTED | — (privacy: good; also means no proofing data exists) |

### 3.2 Scenario coverage today

| Scenario | Representable today? | Consequence |
|---|---|---|
| Multiple accounts per human | **No** — a profile points to at most one account; extra accounts have no profile and no link | Must resolve **UNKNOWN**, never "distinct person" |
| Shared accounts | Yes (`SHARED_USER`) | CONFLICT → cannot sign / record / verify |
| Identity collisions / duplicate people | No (two profiles of one human = two `person_id`s) | Undetectable; needs a reviewed merge workflow (§4.6) |
| Missing personnel links | Yes (`USER_NOT_*`) | UNKNOWN |
| Employment change / transfer | No history (single employment row, overwritten) | Cannot answer "as of signing time" |
| Disabled account | Credential `DISABLED`; account `presence` | Revocation of *identity evidence* not modelled |
| Revocation semantics | Session revoke reasons only | No link-revocation concept |
| Data access / privacy | Shadow import; sensitive-read audit fails closed (D-45 safe default) | No purpose limitation for identity lookups |

**Conclusion (VERIFIED against migrations and import code):** nothing in the repository can resolve an account to a
verified human. `user_id` must never be equated with `person_id`, and `organization.person.person_id` must not be
promoted to "verified person" without a reviewed link (KEEP + ADAPT: the legacy row and its id stay unchanged).

---

## 4. R5-SEC-01 options — decision matrix

### 4.1 Matrix (engineering interpretation of the original options)

| Criterion | A — HR-verified Person registry | B — Security-managed IdP (external) | C — Declarative SoD only | D — Keep blocked |
|---|---|---|---|---|
| Source of truth | HR-maintained Person register (new), legacy profiles as candidates | External IdP subject | Approver's assertion | None |
| Organizational owner | HR (records) + Security (link policy) | Security (+ IdP vendor) | Approver of each decision | — |
| Identity proofing | HR documents; method **to be approved** (HQ-2) | IdP proofing policy | None | None |
| Account→person resolver | Reviewed, effective-dated link records | Federation mapping (subject → account) | None | None (always UNKNOWN) |
| Privacy | Person key + evidence reference only; no document copies without Legal | Data held by IdP; processor contract | Minimal | Minimal |
| Integration complexity | Medium (new register, link workflow) | High (new infrastructure, SSO, provisioning) | Low | None |
| Migration complexity | Medium: shadow projection from legacy profiles, every link reviewed | High: enrol every human; map legacy accounts | Low | None |
| Audit | Link events + two-person review | IdP logs + federation history | Assertion record | — |
| Availability / failure modes | HR backlog → UNKNOWN (fail closed) | IdP outage → BLOCKED; vendor lock-in | Never fails, never proves | Permanently blocked |
| Operational cost | HR + Security reviewer time | Licence + operations | Low | Opportunity cost (P0-6, P0-7 stay blocked) |
| Security risks | False merge / false split; insider link forgery (control: 2-person review, reversible) | IdP compromise; mis-federation | **SoD bypass via second account or alias (F1 unresolved)** | None new |
| F1 independence checks | **Compatible** (verified `person_id`) | **Compatible** | **Incompatible** — independence stays UNKNOWN (RM-E04F §11.1) | Independence UNKNOWN; gates stay closed |
| Employee transfers | Compatible if links and employment are effective-dated (new) | Compatible (subject persists) | n/a | n/a |
| D-34 compatibility | Independent: provisioning still decided by D-34 | D-34 option "External IdP (see R5-SEC-01 B)" couples them | No effect | D-34 still open |
| D-42 compatibility | Enables person-level grant-change SoD | Same | Grant SoD only per account | D-42 approval stays name-level (RT-3) |
| Governance authority assignments (RM-E04F) | Supplies `person_id` for I1–I8 | Same | Cannot activate by person | Cannot activate by person |
| Reversibility | High (append-only unlink; decisions under a wrong link flagged) | Medium (IdP correction; exit cost) | High | Total |
| Prerequisites | HR owner appointed; proofing method; Legal privacy review; possibly ADR #18 (real data) | Security owner; vendor / contract; Legal; budget | None | None |

### 4.2 Separation

| Layer | Content | Status |
|---|---|---|
| 1. Original documented options | §2.2 (verbatim) | DOCUMENTED |
| 2. Engineering interpretation | §4.1 matrix | PROPOSED |
| 3. Recommendation (if supported) | §4.3 | PROPOSED — recommendation only |
| 4. Human decision still required | Option, owner, proofing method, privacy conditions, signers | PENDING |

### 4.3 Engineering recommendation (conditional; not a selection)

**Recommendation only.** On the evidence in the repository, **option A** is the better fit *if* HR can name an
accountable owner and an approved proofing method:

- the only existing personnel source is the HR profile import (§3), which A uses as candidates and B ignores;
- no IdP, federation code, vendor or contract exists in the repository (B would be wholly new);
- A satisfies the F1 contract (RM-E04G S1–S12) exactly as B does;
- A is the more reversible of the two.

Conditions that would change the recommendation:

| Condition | Effect |
|---|---|
| HR cannot commit an owner or a proofing method (HQ-1, HQ-2) | A is not viable → **D** is the safe outcome until it can; B only with a funded Security programme |
| Legal forbids HR from holding identity evidence (HQ-5) | Prefer B |
| Organisation already operates an IdP (unknown to the repository; HQ-9) | Re-evaluate B |
| C is chosen | Must be recorded as **not** satisfying F1: every person-independence gate (Group B interim rule, future RT-3 hardening) stays closed |

Never under any option: automatic merge; merge by phone, name or e-mail similarity; any grant or permission change
(R5 record, "Never (any option)").

---

## 5. Proposed person-identity contract (PROPOSED; not implemented)

### 5.1 Signature

```
resolvePerson(account_id, effective_at, purpose) →
    VERIFIED(person_id, evidence_reference, validity)
  | UNKNOWN(reason)
  | CONFLICT(reason)
  | BLOCKED(reason)
```

It refines RM-E04F §6.1 `resolve(user_id, at)` by adding `purpose` and `BLOCKED`; it does not replace RM-E04G's
evaluator, whose `rmE04GResolve` already maps every non-`VERIFIED` state (including a future `BLOCKED`) to
UNKNOWN or CONFLICT (`TestRME04HResolverContractFailsClosed`).

| Element | Rule |
|---|---|
| `account_id` | `identity.user_account.user_id` (or a future IdP subject under B). Never a name, phone, e-mail or login |
| `effective_at` | The instant the act happened (signing, recording, verifying), not "now" |
| `purpose` | Closed list of resolution contexts (e.g. R6 approval signature, R6 recording, T5 verification, delegation, person-level SoD check). Purpose limitation: an unlisted purpose is `BLOCKED` before any lookup |
| `person_id` | Immutable uuid of a **verified person**, distinct from `organization.person.person_id` (legacy profile) and from `user_id`. Never reused, never re-pointed |
| `evidence_reference` | Reference (not copy) to the link evidence and its review record |
| `validity` | `[valid_from, valid_to)` of the link and of the proofing evidence that cover `effective_at` |
| Precedence | `BLOCKED` (pre-lookup refusal; reveals nothing) → otherwise `CONFLICT` > `UNKNOWN` > `VERIFIED`. A conflict is never hidden |

### 5.2 Outcome rules

| Situation | Result |
|---|---|
| Exactly one active, reviewed link whose evidence covers `effective_at` | `VERIFIED` |
| No link; link pending review; `USER_NOT_*`; evidence expired at `effective_at` | `UNKNOWN(NO_VERIFIED_LINK / LINK_PENDING / EVIDENCE_EXPIRED)` |
| Two active links; `SHARED_USER`; competing HR records; open merge or split case | `CONFLICT(…)` |
| Account designated non-personal (service, automation, bot, shared mailbox) | `BLOCKED(NON_PERSONAL_ACCOUNT)` for signing, recording and T5 |
| Purpose not permitted for the caller; resolver not authorised (R5-SEC-01 not `APPROVED_VERIFIED`) | `BLOCKED(PURPOSE_NOT_PERMITTED / RESOLVER_NOT_AUTHORISED)` |
| Evidence revoked before `effective_at` | `UNKNOWN(EVIDENCE_REVOKED)`; acts *before* revocation keep their snapshot |

### 5.3 Required behaviours

| # | Topic | Proposed rule |
|---|---|---|
| P1 | Immutable `person_id` | Created once by a reviewed proofing act; never deleted (tombstone on merge) |
| P2 | Explicit verified links | `account ↔ person` only through a link record with evidence + two-person review (maker ≠ checker, by person) |
| P3 | Effective dating | Links, proofing evidence and employment are `[from, to)`; resolution is always as-of `effective_at` |
| P4 | Several accounts, one human | Allowed; all resolve to the same `person_id`; every rule counts persons |
| P5 | Shared accounts | Never linked to a person for signing; resolve `CONFLICT` / `BLOCKED` |
| P6 | Homonyms | Distinct `person_id`s; names never compared |
| P7 | Persian / Latin / changed names | Names are attributes with history, never keys; transliteration never used for matching |
| P8 | Employment change | Does not change `person_id`; a new employment period, not an overwrite |
| P9 | Account reassignment | Old link closed (`valid_to`), new link reviewed; past acts resolve to the old person |
| P10 | Revoked / expired evidence | Future resolution UNKNOWN; past snapshots kept and flagged for review, never rewritten |
| P11 | Competing HR records | CONFLICT until a human correction case resolves it |
| P12 | Duplicate people and merges | Merge = new event linking survivor and tombstone; decisions made under both ids are re-evaluated (flag only) |
| P13 | Human correction workflow | Case → evidence → independent reviewer → append-only correction event |
| P14 | Audit history | Append-only events for link, unlink, merge, split, proofing, revocation |
| P15 | Evidence provenance | Each link names its evidence type, reference, hash, reviewer person and date |
| P16 | Prohibited inputs | Name, phone, e-mail, GitHub account cannot *independently* establish identity; they may only help a human find candidates |
| P17 | Privacy | Store references, not document copies, unless Legal approves (HQ-5); lookups audited and purpose-limited |

---

## 6. F1 / R6 compatibility design (PROPOSED)

### 6.1 Future R6 identity extension (input to F-PR1)

Extends RM-E04F §11.2 (additive, optional on v1 → mandatory only where a reviewed decision says so, RV-04).

| Slot | Proposed field(s) |
|---|---|
| Each approver | `account_ref`, `person_ref`, `identity_evidence_ref`, `authority_assignment_ref`, `authority_evidence.sha256` |
| Decision-time snapshot | `identity_snapshot`: per participant `{account_ref, person_ref, resolution_state, evidence_ref, valid_from, valid_to, resolved_at}` |
| Recorder | `recorded_by.account_ref`, `recorded_by.person_ref` |
| T5 verifier | `human_verification.verifier_account_ref`, `verifier_person_ref` |
| Delegation | `delegation_ref`, `delegator_account_ref`, `delegator_person_ref` |

Until F-PR1 is approved and merged, **none of these fields may appear** in a committed record (RM-E04G F1-I guard,
`TestRME04GApprovalRecordsCarryOnlyR6V1Fields`). Decision-time identity evidence for R5-SEC-01 itself therefore
lives in the evidence referenced by `evidence_source` / `immutable_ref`, not in the record.

### 6.2 Separation-of-duties rules

| # | Rule |
|---|---|
| SD-R1 | One person = one approval per decision version, whatever the number of positions, authorities or accounts. One person holding Product, Sales and Security counts **once** |
| SD-R2 | Every required role needs a distinct verified person |
| SD-R3 | T5 verifier ∉ {approvers, delegating principals, recorder}; never automation (an agent is automation) |
| SD-R4 | Recorder may be an approver (R6 rule I7, unchanged); never the verifier |
| SD-R5 | Delegation reserves both delegate and delegator; never adds independence |
| SD-R6 | Any UNKNOWN / CONFLICT / BLOCKED participant ⇒ independence UNKNOWN / CONFLICT ⇒ not satisfiable |

These restate RM-E04F I1–I8 and RM-E04G S1–S12; no new rule is introduced.

### 6.3 Historical R6 v1 (name-only) records

Zero exist today. Any v1 record created before F-PR2 stays valid **for R6** (no rewrite, append-only) but evaluates
`PERSON_INDEPENDENCE_UNKNOWN` for every person-gated consumer. A later identity attestation, if approved (RV-05),
is a **new appended** verification artefact referencing the v1 record, never an edit of it.

### 6.4 Bootstrap risk — R5-SEC-01's own approval

R5-SEC-01 must be approved *before* any resolver exists, by the same name-level validator that F1 shows to be weak.
Its independence (four distinct humans: Security, HR, Legal, Product) therefore rests **only** on the T5 human
checklist (RM-E04E §3.5, "no person counted twice"). Proposed compensating controls (PENDING, HQ-8):

1. the T5 verifier confirms each signer's identity through an independent channel and states the method;
2. the decision evidence (`immutable_ref`) includes each signer's authority instrument;
3. after F-PR2, R5-SEC-01's own record is re-evaluated by person (flag only, no rewrite).

### 6.5 Replacing the RM-E04G interim guard

RM-E04G-I1 stays in force, unweakened. It may be relaxed only by a reviewed PR that:

1. replaces `rmE04GRepositoryResolver` with the F-PR2 resolver bound to an `APPROVED_VERIFIED` R5-SEC-01 (A or B);
2. replaces `rmE04GClaimsFromRecord` with an adapter for the F-PR1 schema;
3. keeps every RM-E04G test green and adds F-PR2 regression tests;
4. carries Codex + Security review evidence and an independent T5 of the R5-SEC-01 record.

Under C or D the guard is never replaced.

---

## 7. Other security paths that may treat `APPROVED_VERIFIED` as actionable

### 7.1 Route map (RM-E04G RT-1…RT-6, re-traced at `19ab171`, plus new findings)

| Route | Consumer | Person-independence today | Other gap found | Status |
|---|---|---|---|---|
| RT-1 Group B gate | `rmE04CGateState` + `rmE04GGroupBReadiness` | Required (interim rule) | — | BLOCKED (guarded) |
| RT-2 S4 readiness | `s4ReadinessProblems` (inside Group B) | Transitively required | — | BLOCKED (guarded) |
| RT-3 D-42 → `identity.role_grant` migrations | `grantConstraintProblems` (`rm_e04d_test.go`) | **Not required** | see §7.2 | **OPEN RISK** — decision PENDING (RV-09b) |
| RT-4 Follow-up eligibility | `followUpEligible` (documentation rule) | Not required | Documentation only | PENDING |
| RT-5 Tracker `*_VERIFIED` | `statusMatchesRecord` | Not required by design | — | Unchanged |
| RT-6 Nominee → approver | `TestRME04FNomineeIsNotAnApprover` | Name-keyed | Alias not caught | Residual |
| RT-7 D-34 provisioning | none in repository (no credential writer; app role SELECT-only) | n/a | Future provisioning must not use phone/e-mail as identity | BLOCKED (no path) |
| RT-8 Governance authority activation (RM-E04F) | none implemented; register forbids `ACTIVE` | n/a | Needs resolver (I1–I8) | BLOCKED (no path) |
| RT-9 Person-level SoD (Finance decision, refund) | R4 contracts return `IDENTITY_PERSON_LINK_UNMODELLED` | Fails closed | — | BLOCKED (fail-closed) |
| RT-10 R5-SEC-01 record itself | R6 validator + `TestM2TrackerUnchanged` (R5 signer floor via `m2Approvers`) | Name-level | Bootstrap risk (§6.4) | Signer floor VERIFIED; independence relies on T5 |

### 7.2 D-42 / RT-3 analysis

`grantConstraintProblems` returns no problem for any `identity.role_grant` migration once the D-42 tracker status
is `APPROVED_VERIFIED`. Traced weaknesses:

| # | Weakness | Evidence |
|---|---|---|
| G1 | No person independence: D-42 needs Product, HR, Sales, Security owners; the nominee N1 is nominated for three of those (Product, Sales, Security) and could pass the name-level validator under three spellings (F1-A) | intake `D-42-v1.0-DECISION.BLANK.json`; nominations register |
| G2 | *Checked, not a gap:* `validateApprovalRecord` alone checks only the roles a record declares, but `TestM2TrackerUnchanged` (`m2_test.go`, `m2Approvers`) refuses a D-42, D-34 or R5-SEC-01 status that is not backed by a record carrying every R5 approver role. The floor therefore holds; only person independence (G1) is missing | `m2_test.go`; `baselineOrVerifiedProblems` |
| G3 | D-42 depends on R5-SEC-01 and E-7, but the unlock checks neither | inventory row D-42; `grantConstraintProblems` |
| G4 | All-or-nothing: the unlock ignores the selected D-42 option (e.g. "Identity / Security only" vs "HR may grant within a capped set") | same |
| G5 | Scope: only `internal/identity/migrations/sql` is scanned. Today no other migration directory references `role_grant` (VERIFIED by search); a future one would be invisible | `identityMigrations` |

**Recommendation (PROPOSED, PENDING decision):** introduce one shared precondition for every security-sensitive
unlock route (RT-1, RT-2, RT-3, RT-8, and later D-34 provisioning): *decision `APPROVED_VERIFIED` **and**
R5-SEC-01 `APPROVED_VERIFIED` (A or B) **and** `PERSON_INDEPENDENCE_VERIFIED` **and** the R5 required-signer floor*.
Applying it to RT-3 changes an existing reviewed unlock rule and needs architectural approval (RM-E04G RV-09b; handoff H-RV-08). **No
consumer or gate is changed in this PR.**

### 7.3 Signer floor (re-checked, unchanged)

An early draft of this package assumed that no repository check enforced the R5 approver roles on the R5-SEC-01
record. That is wrong: `TestM2TrackerUnchanged` already applies `rmE04CDecisionProblems` with `m2Approvers`
(Security, HR, Legal, Product owner for R5-SEC-01; Product, HR, Sales, Security for D-42; Security, HR for D-34).
This PR adds no new floor; `TestRME04HSignerSlotsMatchSources` only pins that the intake blank, `m2Approvers` and
§8 of this package agree, so the "five distinct humans" statement cannot silently drift.

---

## 8. Authority and human approval (from the repository)

| Question | Answer | Source | Status |
|---|---|---|---|
| Decision owner | "Security owner + HR"; lead owner blank | R5 record; M1 lead table | DOCUMENTED; holders **UNASSIGNED** |
| Who must sign | Security owner, HR owner, Legal owner, Product owner — each independently | R5 record "Required approver roles"; intake blank `independent_approvals` | DOCUMENTED |
| Legal privacy review | **Mandatory** (Legal (privacy) is a required approver; R5 "Missing evidence: privacy review") | R5 record | DOCUMENTED |
| Who registers | R5-SEC-01 is already inventoried (no new id needed). The **record** is written by a Recorder (any role, may be an approver, never T5) | evidence standard §2; RM-E04E §3.4 | DOCUMENTED; holder UNASSIGNED |
| Who performs T5 | A person who is not an approver, not the recorder, not automation | evidence standard §3; RM-E04E §3.5 | DOCUMENTED; **UNASSIGNED** |
| Required evidence | Per approver: authority instrument + reference; decision evidence `immutable_ref`; T5 method + date | evidence standard §1–§3; RM-E04E §1.3 | DOCUMENTED |
| Appointing authority | CEO (project-owner statement) | nominations register `appointing_authority` | Design input; holder **UNASSIGNED**, not verified |
| مهرداد حافظی (N1) | Nominated (provisionally) for inventory owner, Product, Sales, Security, Legal, Finance, Recorder — **not** HR, **not** T5 | nominations register | **NOMINATED** — not appointed, not verified |

**Minimum distinct humans for a valid R5-SEC-01 decision: five** — four independent approvers (one each for
Security, HR, Legal, Product) plus one T5 verifier who is none of them. The recorder may be one of the approvers.
N1 can fill at most **one** approver slot (and the recorder role); at least three other approvers and a T5 verifier
are needed. HR owner has no nominee at all.

### 8.1 Unanswered business / HR / Security / Legal questions

| # | Question | Owner to answer |
|---|---|---|
| HQ-1 | Who is the accountable HR owner of person records? (No HR nominee exists) | M1 / CEO |
| HQ-2 | Which identity-proofing method is acceptable (e.g. HR in-person check against the employment file)? | HR + Security |
| HQ-3 | Is the legacy HR profile data complete enough to serve as *candidates* for A? (HR NOT LIVE VERIFIED; quality UNKNOWN) | HR |
| HQ-4 | Which accounts are non-personal or shared, and who designates them? | Security + HR |
| HQ-5 | What identity data may be stored, for how long and on what basis; may HR keep document copies? | Legal (privacy) |
| HQ-6 | Which positions may make and check an account-to-person link (two-person rule)? | Security + HR |
| HQ-7 | When does a link close on transfer, termination or reassignment (relation to OPD-05)? | HR + Security |
| HQ-8 | Are the bootstrap controls of §6.4 accepted for R5-SEC-01's own approval? | Security + T5 verifier |
| HQ-9 | Does the organisation already operate an identity provider / SSO? | Security |
| HQ-10 | Direction of the R5-SEC-01 ↔ D-34 / D-42 / ADR #18 dependency (OF-4) | Inventory owner + Codex |
| HQ-11 | If C is chosen, is it confirmed that C never satisfies a person-independence gate? | Security |
| HQ-12 | Who is the independent T5 verifier for R5-SEC-01? | M1 / CEO |

Not evidence of authority: this package, the project owner's session instructions, a GitHub account or approval, a
merge, CI results or an agent's recommendation.

---

## 9. Implementation roadmap (future; none authorised here)

All steps are BLOCKED until their precondition holds. No migration or runtime code is written in this phase.

| Step | Content | Precondition | Acceptance tests |
|---|---|---|---|
| H0 | This package reviewed | — | Handoff items answered by reviewers |
| H1 | Humans decide R5-SEC-01 (R6 record + independent T5) | M1 appoints signers with evidence; Legal privacy review | `TestM2TrackerUnchanged` passes on the real record (all four roles); R6 suite green |
| **F-PR1** | Additive R6 identity-evidence schema (§6.1): extend `rmE04GR6V1Fields` and the record struct; strict unknown-field rejection kept; optional fields validated for shape; no forged refs (refs must resolve in the registry or be absent) | H0; RV-04, RV-05, RV-11 answered | v1 records still validate unchanged; unknown keys still refused; a ref to a non-existent person / evidence refused; history scan still append-only; mutation: dropping strict validation is caught |
| **F-PR2** | Verified resolver (§5) replacing `rmE04GRepositoryResolver`; person-based independence in the R6 validator; adapter replacing `rmE04GClaimsFromRecord`; T5 / recorder / delegation checks by person; RM-E04G-I1 relaxed only as §6.5 | H1 = `APPROVED_VERIFIED` with A or B; F-PR1 merged | F1-A…F1-H all fail closed or are rejected by person; F1-E homonyms accepted by distinct `person_id`; BLOCKED / UNKNOWN / CONFLICT never satisfy a gate; display-name renames never change an outcome; T5 = approver under another account rejected; overlapping delegation rejected |
| F-PR3 | D-42 path hardening (shared precondition §7.2) | RV-09b decided; F-PR2 | G1–G5 each covered by a negative fixture; existing D-42 tests updated only as the decision says |
| F-PR4 | Governance authority registry (RM-E04F F-PR1) consuming `person_id` | F-PR2; RM-E04F AD1–AD3 | No `ACTIVE` without verified person + independent verifier (I8) |
| F-PR5 | Shadow-only identity projection (verified person, link, evidence tables) | F-PR2; Codex architecture + DBA review; ADR #18 if real data | Projection never authorises; parity with repository registry; append-only triggers; rollback = drop projection |
| F-PR6 | Optional admin interface (synthetic data first) | F-PR5; capability decision (permissions are not Claude's to invent) | RTL / a11y QA; no evidence leakage; purpose-limited lookups audited |
| — | Production activation | Separate authorisation | **NO-GO** |

---

## 10. Governance test evidence

### 10.1 Committed tests (`docs/governance/rm_e04h_test.go`)

| Test | Protects |
|---|---|
| `TestRME04HOriginalOptionsQuotedVerbatim` | §2.2 quotes A–D exactly as in the R5 record; the dashboard lists all four, none ticked |
| `TestRME04HRepositoryStateMatchesPackage` | R5-SEC-01, D-34, D-42 undecided (status, record, verification, empty outcome); no approvals directory; nominations provisional, nominee without identity claim; identity / organization migration set unchanged (no link table); repository resolver resolves nothing (names, aliases, phone, e-mail, login); Group B gate and S4 manifest closed |
| `TestRME04HSignerSlotsMatchSources` | §7.3 / §8 — intake blank, `m2Approvers` and this package name the same four approver roles; the minimum of five distinct humans is stated |
| `TestRME04HResolverContractFailsClosed` | A future `BLOCKED` (or any unknown state) never resolves to VERIFIED in the existing evaluator; independence then not VERIFIED |
| `TestRME04HDocumentsMarkedNotApproved` | DRAFT marker, NO-GO, Group B / S4 BLOCKED, no pre-ticked option, blank decision form, handoff REVIEW PENDING with every row PENDING, no unregistered decision id, no claimed sign-off |

These tests protect the *documents and repository state*. They do **not** enforce runtime identity; nothing does
until F-PR2.

### 10.2 Runs (actual results in this worktree)

| Run | Result |
|---|---|
| `go test -count=1 ./docs/governance/` (baseline, before changes) | PASS |
| `go test -count=1 ./docs/governance/` (with this package) | PASS |
| `R5_REQUIRE_PROVENANCE=1 go test -count=1 ./docs/governance/` (full clone) | PASS |
| `go build ./...`, `go vet ./docs/governance/`, `gofmt -l docs/governance/` | clean |
| `go test ./internal/identity/... ./internal/organization/... ./internal/authorization/...` | PASS (no source changed; packages without tests skipped) |

### 10.3 Mutation checks (throwaway copies in the session scratchpad; nothing committed; worktree verified clean)

| # | Mutation | Result |
|---|---|---|
| MH1 | Tick option A in the dashboard form | Killed (`…DocumentsMarkedNotApproved`, `…OriginalOptionsQuotedVerbatim`) |
| MH2 | Fill the form outcome with APPROVED | Killed (`…DocumentsMarkedNotApproved`) |
| MH3 | Fill a signer name, evidence, signature and date | Killed (same) |
| MH4 | Fill the T5 row with a VERIFIED result | Killed (same) |
| MH5 | Tick a T5 checklist item | Killed (same) |
| MH6 | Reword original option C in §2.2 | Killed (`…OriginalOptionsQuotedVerbatim`) |
| MH7 | Mark a handoff item ACCEPTED | Killed (`…DocumentsMarkedNotApproved`) |
| MH8 | Handoff status changed from REVIEW PENDING | Killed (same) |
| MH9 | DRAFT marker removed from the package | Killed (same) |
| MH10 | Unregistered decision id cited | Killed (same) |
| MH11 | "Security signed off" sentence added to the handoff | Killed (same). A first attempt survived only because the harness wrote a literal `\n`, merging the line with "Not claimed"; re-run with a real line break |
| MH12 | Tracker `selected_option` filled without a record | Killed (existing `TestR6NoStatusOrOutcomeWithoutApprovalRecord`) |
| MH13 | Repository resolver returns VERIFIED | Killed (`…RepositoryStateMatchesPackage` + RM-E04G tests) |
| MH14 | `rmE04GResolve` treats an unknown state (e.g. BLOCKED) with a person id as VERIFIED | Killed **only** by `…ResolverContractFailsClosed` |
| MH15 | Legal owner dropped from `m2Approvers` | Killed (`…SignerSlotsMatchSources`, existing `TestM2IntakeBlanksNeverValidate`) |
| MH16 | HR owner nominee added to the register | Killed **only** by `…SignerSlotsMatchSources` |
| MH17 | Nominee `person_ref` filled | Killed (`…RepositoryStateMatchesPackage`, existing `TestRME04FNominationsStayProvisional`) |
| MH18 | "five" distinct humans changed to "four" | Killed (`…SignerSlotsMatchSources`) |
| MH19 | R5-SEC-01 forged `APPROVED_VERIFIED` without a record | Killed (existing R6 / M2 / RM-E04C / RM-E04G tests + `…RepositoryStateMatchesPackage`) |
| MH20 | New organization migration adding a person-link table | Killed **only** by `…RepositoryStateMatchesPackage` |
| MH21 | Empty `docs/governance/approvals/` directory created | Killed **only** by `…RepositoryStateMatchesPackage` |
