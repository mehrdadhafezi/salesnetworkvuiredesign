# RM-E04I — R5-SEC-01 Option A shadow implementation readiness

**DRAFT — NOT APPROVED.** Implementation-readiness contract for a *future* implementation of R5-SEC-01 option A
(HR-verified Person registry). Option A is the **project owner's preferred engineering direction**; that preference
is not a governance decision. This package selects no option, records no decision, appoints no one, creates no
person, link, grant, credential, schema, migration or runtime code, and changes no tracker, gate or manifest.
**Production: NO-GO. Group B: BLOCKED. S4: BLOCKED. Real grants: BLOCKED.**

| | |
|---|---|
| Baseline | `main` @ `739db621bdf834cd92d7b33ce9a31f224572cc22` (merge of PR #81, RM-E04H) |
| Branch | `docs/rm-e04i-option-a-shadow-readiness` (isolated worktree from `origin/main`) |
| Preference record (non-authoritative) | `docs/governance/proposals/rm-e04i-option-a-preference.NON-AUTHORITATIVE.json` |
| Dependency graph (machine-readable) | `docs/governance/proposals/rm-e04i-dependency-graph.json` |
| Review handoff | `docs/implementation/rm-e04i-review-handoff.md` — REVIEW PENDING |
| Tests | `docs/governance/rm_e04i_test.go` |
| Builds on, does not repeat | RM-E04H package (`docs/security/rm-e04h-person-identity-decision-package.md`: options, data audit §3, contract §5, F1 design §6, routes §7, authority §8); RM-E04G (`docs/security/rm-e04g-f1-person-identity-independence.md`); RM-E04F architecture; RM-E04E authority packet; R6 evidence standard |

Status vocabulary: **DOCUMENTED** (stated in a repository source), **PROPOSED** (engineering proposal in this
package), **VERIFIED** (checked against code or a test in this PR), **PENDING** (waiting for a named human or
reviewer), **BLOCKED** (cannot proceed until a named dependency is met), **UNRESOLVED** (a recorded conflict).

---

## 1. Preflight

| Check | Result | Status |
|---|---|---|
| `origin/main` HEAD | `739db621bdf834cd92d7b33ce9a31f224572cc22` = the SHA given in the task | VERIFIED |
| PR #81 merged | `739db62` is "Merge pull request #81 … docs/rm-e04h-person-identity-decision" | VERIFIED |
| Worktree | fresh worktree from `origin/main`; zero changes before this work | VERIFIED |
| Baseline `go test -count=1 ./docs/governance/...`, `go build ./...`, `go vet ./docs/governance/...` | PASS / clean | VERIFIED |
| R5-SEC-01, D-34, D-42 in tracker | `READY_FOR_REVIEW`, `approval_record` empty, verification `NOT_STARTED`, outcome empty | VERIFIED (unchanged) |
| `docs/governance/approvals/` | absent — zero R6 records | VERIFIED |
| Group B gate / S4 manifest | closed / empty | VERIFIED |

---

## 2. Project preference vs. authority (six separate layers)

| Layer | Value | Evidence | Status |
|---|---|---|---|
| Project preference | **A** | project owner instruction (RM-E04I task) | DOCUMENTED — `PREFERRED_OPTION = A` |
| Engineering recommendation | A, *conditional* on an HR owner and an approved proofing method; otherwise D | RM-E04H §4.3 | PROPOSED (unchanged) |
| Official governance decision | **none** | tracker `selected_option` empty | `DECISION_STATUS = NOT_APPROVED` |
| Human approval | **none** — Security, HR, Legal, Product signers not appointed | nominations register; RM-E04H §8 | PENDING |
| Independent T5 verification | **none** — verifier UNASSIGNED | nominations register | PENDING |
| Implementation authorization | **not granted** | this package | `IMPLEMENTATION_AUTHORIZATION = NOT_GRANTED` |

`AUTHORITY_STATUS = UNVERIFIED`. The preference is recorded **only** in the non-authoritative JSON record named
above, which no gate, evaluator, migration or provisioning workflow reads (`TestRME04IPreferenceIsNotConsumed`).
The original four alternatives stay intact and unselected (RM-E04H §2.2; the record repeats them verbatim):
A — HR-verified Person registry; B — Security-managed IdP; C — Declarative SoD only; D — Keep blocked.

Not changed by this package: R5 tracker, R5 decision inventory, approval records, authority nominations, R6 intake
blanks, Group B gate, S4 manifest.

---

## 3. Reuse inventory (no parallel identity model)

### 3.1 Reused as-is

| Existing contract / structure | Location | Reused for |
|---|---|---|
| `organization.person` (legacy HR profile row, uuid per profile, import-refreshed) | `internal/organization/migrations/sql/00001_organization.sql` | **Candidate source** only; referenced by evidence, never rewritten, never promoted (KEEP + ADAPT) |
| `organization.employment` (`person_id` unique, `user_id` non-FK identity reference, `user_link_state`) | same | Candidate account pointer (`LINKED` = hint; `SHARED_USER` = CONFLICT; `USER_NOT_*` = UNKNOWN); employment status as *evidence*, not identity |
| Non-FK cross-module reference pattern (`employment.user_id` = published identity API id) | same | `account_id` in the new link entity follows the same pattern |
| `identity.user_account.user_id` | `internal/identity/migrations/sql/00001_identity.sql` | The resolver's `account_id`; `display_name` never read |
| `audit.audit_event` (append-only by trigger; `subject_domain`, `action` dotted format, `metadata` ≤ 8 KiB) | `internal/platform/migrations/sql/00001_platform_audit.sql`; `internal/platform/audit` (`Append`) | Every transition event (§6) — no new audit table |
| Per-module `outbox_event`, `idempotency_record` | `organization` schema | Event publication and idempotent commands |
| Lint `foreign-schema-write`, `s11-hr-identity-write`, `phoneIdent` | `internal/architecture/lint.go` | Organization must not write identity grants / credentials; phone never in identity commands |
| Resolver contract `resolvePerson(account_id, effective_at, purpose)` + P1–P17 | RM-E04H §5 | Refined in §5 below, not replaced |
| F1 evaluator `rmE04GResolve`, `rmE04GIndependence`, interim rule RM-E04G-I1 | `docs/governance/rm_e04g_test.go` | Consumer of the resolver (§7); unchanged |
| RM-E04F I1–I8, `resolve(user_id, at)` | RM-E04F §6.1 | Authority activation consumes `verified_person_id` |
| R6 v1 schema + strict field list `rmE04GR6V1Fields`; evidence standard | `docs/governance/r6_test.go`, `rm_e04g_test.go`, `docs/remediation/r6-approval-evidence-standard.md` | Unchanged; F-PR1 extension in §7 |

### 3.2 Why one new anchor is unavoidable

`organization.person.person_id` cannot be the verified-person key: it is created per legacy HR **profile**, refreshed
by import (`version`, `presence`), and one human with two profiles yields two ids (RM-E04H §3.2). Re-purposing it
would rewrite history. Option A therefore needs exactly one new anchor, **VerifiedPerson**, plus the six records the
task names. Nothing else is added: no new audit table, no new account table, no new employment table.

---

## 4. Option A data contract (PROPOSED — no migration in this PR)

### 4.1 Ownership

All seven structures belong to the **organization** module (HR is the record owner of persons under A), in a new
`organization` sub-area "person registry". `account_id` is a **non-FK identity reference** (same pattern as
`organization.employment.user_id`), validated through the published identity API at write time. The identity module
never writes these tables and organization never writes `identity.*` (existing lint rules). Security owns the link
*policy* (who may make / check links, purposes) — a decision input (HQ-6), not a module.

Common rules for every structure:

- append-only: no `UPDATE` of business columns, no `DELETE`, no `TRUNCATE` (trigger, as `audit.audit_event`);
  state changes are new rows or new events; corrections supersede;
- `recorded_at` (system time) **and** `[valid_from, valid_to)` (effective time): bitemporal, so "what was known
  when" and "what was true when" are both answerable;
- every actor column stores `actor_account_id` **and** the actor's resolved `actor_verified_person_id` at action
  time (or `NULL` + state `UNKNOWN`, which blocks the SoD checks of §6);
- no name, phone, e-mail, login or GitHub handle is a key, a join column or a matching input;
- application role: `SELECT` only until PR-B is authorized; writes only through organization commands.

### 4.2 Entities

**E1 VerifiedPerson** (anchor — one human)

| Attribute | Contract |
|---|---|
| Owner module | organization (person registry) |
| Stable identifier | `verified_person_id uuid` — generated once by a completed IdentityVerification; never reused, never re-pointed |
| Data source | HR proofing act (E3); candidates from `organization.person` |
| Required fields | `verified_person_id`, `created_by_verification_id`, `lifecycle_state` (`ACTIVE`, `MERGED_INTO`, `SPLIT_SOURCE`, `RETIRED`), `recorded_at` |
| Immutable fields | `verified_person_id`, `created_by_verification_id`, `recorded_at` |
| Evidence references | creating verification; merge / split finding (E6) |
| Effective dates | `valid_from` = verification effective time; tombstone `valid_to` on merge |
| Audit events | `organization.person_registry.person_created`, `…person_merged`, `…person_split`, `…person_retired` |
| Retention | lifetime of the organisation's records + Legal minimum (HQ-5); tombstones never deleted |
| Privacy | INTERNAL-RESTRICTED: holds no name, document number or contact data |
| Failure states | creation without completed verification → rejected; merge target not `ACTIVE` → CONFLICT |
| Indexing / constraints | PK; `merged_into_id` FK to self; CHECK `merged_into_id IS NOT NULL` ⇔ `MERGED_INTO`; no unique on any attribute other than the id |

**E2 PersonIdentityEvidence** (what proves who someone is)

| Attribute | Contract |
|---|---|
| Owner module | organization |
| Stable identifier | `evidence_id uuid` |
| Data source | HR employment file, in-person check, signed appointment instrument (method set decided in HQ-2) |
| Required fields | `evidence_id`, `subject_kind` (`VERIFIED_PERSON` / `CANDIDATE_PROFILE`), `subject_id`, `evidence_type` (closed list from HQ-2), `reference` (location of the original, not a copy), `content_sha256` (32 bytes), `submitted_by_account_id`, `submitted_by_person_id`, `valid_from`, `valid_to`, `recorded_at` |
| Immutable fields | all; a correction is a new row with `supersedes_evidence_id` |
| Evidence references | external original (HR file id); optional legacy `organization.person.person_id` as candidate |
| Effective dates | `[valid_from, valid_to)`; `valid_to` mandatory where the method expires (HQ-2) |
| Audit events | `…evidence_submitted`, `…evidence_superseded`, `…evidence_expired` |
| Retention | reference + hash retained with the link it supports; document copies **not stored** unless Legal approves (HQ-5) |
| Privacy | CONFIDENTIAL-PERSONAL: purpose-limited reads, each read audited (D-45 safe default) |
| Failure states | missing hash / reference → rejected; hash mismatch at review → CONFLICT; expired → resolver `UNKNOWN(EVIDENCE_EXPIRED)` |
| Indexing / constraints | `(subject_kind, subject_id, valid_from)`; CHECK `octet_length(content_sha256) = 32`; CHECK `valid_to IS NULL OR valid_to > valid_from` |

**E3 IdentityVerification** (the human proofing act, maker side)

| Attribute | Contract |
|---|---|
| Owner module | organization |
| Stable identifier | `verification_id uuid` |
| Data source | HR verifier action over E2 evidence |
| Required fields | `verification_id`, `verified_person_id` (or `NULL` when creating), `evidence_ids[]`, `method`, `verifier_account_id`, `verifier_person_id`, `outcome` (`VERIFIED`, `REJECTED`, `INCONCLUSIVE`), `effective_from`, `recorded_at` |
| Immutable fields | all |
| Evidence references | E2 rows; method identifier |
| Effective dates | `effective_from`; expiry inherited from the evidence |
| Audit events | `…identity_verified`, `…identity_verification_rejected`, `…identity_verification_inconclusive` |
| Retention | as E1 |
| Privacy | INTERNAL-RESTRICTED |
| Failure states | verifier person unknown → rejected (bootstrap: §6.4); verifier = subject → rejected (no self-verification) |
| Indexing / constraints | `(verified_person_id, effective_from)`; CHECK `outcome` in list |

**E4 VerifiedPersonAccountLink** (account → person, effective-dated)

| Attribute | Contract |
|---|---|
| Owner module | organization |
| Stable identifier | `link_id uuid` |
| Data source | reviewed proposal (E5) — never an import, never an automatic match |
| Required fields | `link_id`, `account_id` (non-FK identity reference), `verified_person_id`, `account_kind` (`PERSONAL`; `NON_PERSONAL` accounts are never linked), `state` (`PROPOSED`, `UNDER_REVIEW`, `VERIFIED`, `EFFECTIVE`, `REVOKED`, `EXPIRED`, `REJECTED`, `SUPERSEDED`), `purposes[]` (subset of §5.2), `valid_from`, `valid_to`, `proposed_by_*`, `review_id`, `recorded_at` |
| Immutable fields | `link_id`, `account_id`, `verified_person_id`, `valid_from`, `proposed_by_*`, `recorded_at`; a state change is a new state row (state history table or event), never an edit |
| Evidence references | E2 evidence ids; E5 review; legacy `employment.user_link_state` snapshot as a *hint* |
| Effective dates | `[valid_from, valid_to)`; reassignment closes the old link before the new `valid_from` |
| Audit events | `…link_proposed`, `…link_reviewed`, `…link_effective`, `…link_revoked`, `…link_expired`, `…link_rejected`, `…link_superseded` |
| Retention | never deleted; past acts depend on it |
| Privacy | INTERNAL-RESTRICTED |
| Failure states | two effective links for one account overlapping → CONFLICT; link to a `MERGED_INTO` person → resolve through the survivor *and* flag |
| Indexing / constraints | exclusion constraint: no two `EFFECTIVE` rows with the same `account_id` and overlapping `tstzrange(valid_from, valid_to)`; index `(verified_person_id, valid_from)`; **no** uniqueness on `verified_person_id` (several accounts per person are allowed) |

**E5 PersonLinkReview** (the checker side of the two-person rule)

| Attribute | Contract |
|---|---|
| Owner module | organization |
| Stable identifier | `review_id uuid` |
| Data source | independent reviewer decision |
| Required fields | `review_id`, `link_id`, `reviewer_account_id`, `reviewer_person_id`, `decision` (`APPROVE`, `REJECT`, `REQUEST_CORRECTION`), `checked_evidence_ids[]`, `evidence_hashes_confirmed` (bool), `reason`, `recorded_at` |
| Immutable fields | all |
| Evidence references | E2 rows re-hashed at review |
| Effective dates | `recorded_at` only (the link carries effective time) |
| Audit events | `…link_review_approved`, `…link_review_rejected`, `…link_correction_requested` |
| Retention | as E4 |
| Privacy | INTERNAL-RESTRICTED |
| Failure states | reviewer person = proposer person, = subject person or UNKNOWN → review refused |
| Indexing / constraints | `(link_id, recorded_at)`; CHECK `decision` in list |

**E6 IdentityConflictFinding** (merge, split, shared, competing records)

| Attribute | Contract |
|---|---|
| Owner module | organization |
| Stable identifier | `finding_id uuid` |
| Data source | reviewer, import refresh (e.g. new `SHARED_USER`), resolver detection, reported correction |
| Required fields | `finding_id`, `kind` (`DUPLICATE_PERSON`, `SPLIT_PERSON`, `SHARED_ACCOUNT`, `COMPETING_HR_RECORD`, `ACCOUNT_REASSIGNED`, `EVIDENCE_HASH_MISMATCH`), `subject_refs[]`, `state` (`OPEN`, `RESOLVED_MERGE`, `RESOLVED_SPLIT`, `RESOLVED_NO_CHANGE`, `RESOLVED_REVOKE`), `opened_by_*`, `resolved_by_*`, `resolution_event_id` |
| Immutable fields | `finding_id`, `kind`, `subject_refs`, `opened_by_*`; resolution appended |
| Evidence references | E2, E4, E5 rows |
| Effective dates | `opened_at`, `resolved_at` |
| Audit events | `…conflict_opened`, `…conflict_resolved` |
| Retention | never deleted |
| Privacy | INTERNAL-RESTRICTED |
| Failure states | any `OPEN` finding touching an account or person → resolver `CONFLICT` for every purpose |
| Indexing / constraints | GIN on `subject_refs`; partial index `state = 'OPEN'` |

**E7 IdentityLinkRevocation**

| Attribute | Contract |
|---|---|
| Owner module | organization |
| Stable identifier | `revocation_id uuid` |
| Data source | HR / Security action, evidence revocation, termination (OPD-05 timing), conflict resolution |
| Required fields | `revocation_id`, `link_id`, `reason` (`TERMINATION`, `REASSIGNMENT`, `EVIDENCE_REVOKED`, `FRAUD_SUSPECTED`, `CONFLICT_RESOLUTION`, `CORRECTION`), `effective_at`, `revoked_by_account_id`, `revoked_by_person_id`, `recorded_at`, `retroactive` (bool = `effective_at < recorded_at`) |
| Immutable fields | all |
| Evidence references | termination record, conflict finding, evidence id |
| Effective dates | `effective_at`; retroactive revocations allowed but never rewrite past results (§5.5) |
| Audit events | `…link_revoked`, plus `…retroactive_revocation_flagged` when `retroactive` |
| Retention | never deleted |
| Privacy | INTERNAL-RESTRICTED |
| Failure states | revoking an already revoked link → idempotent no-op; revoker = link subject → refused |
| Indexing / constraints | `UNIQUE (link_id)` (one revocation per link) |

Audit events go to `audit.audit_event` with `subject_domain = 'organization'`, `actor_domain = 'identity'`,
`actor_id = <account_id>`, and `metadata` holding ids and hashes only (never names or documents).

---

## 5. Resolver contract (PROPOSED interface; no runtime resolver in this PR)

### 5.1 Interface

```
resolvePerson(account_id, effective_at, purpose) →
    VERIFIED(verified_person_id, evidence_refs[], link_id, validity, registry_revision)
  | UNKNOWN(reason, evidence_refs[])
  | CONFLICT(reason, evidence_refs[])
  | BLOCKED(reason)
```

Owner: organization query API (read-only). Consumers: R6 validator / F1 evaluator (via F-PR2), RM-E04F authority
activation, person-level SoD checks. It **returns identity only**: it never reads or writes grants, capabilities,
sessions or credentials, and never returns an `authorization.Subject` (requirement 10).

### 5.2 Purposes (closed list; anything else is `BLOCKED(PURPOSE_NOT_PERMITTED)` before any lookup)

| Purpose | Used by |
|---|---|
| `R6_APPROVAL_SIGNATURE` | approver independence |
| `R6_RECORDING` | recorder identity |
| `T5_VERIFICATION` | verifier independence |
| `DELEGATION_PRINCIPAL` | delegator / delegate |
| `AUTHORITY_ASSIGNMENT` | RM-E04F I1–I8 |
| `PERSON_SOD_CHECK` | Finance / refund person-level SoD (R4 `IDENTITY_PERSON_LINK_UNMODELLED`) |

A link carries `purposes[]`; a purpose not on the link resolves `BLOCKED(PURPOSE_NOT_ON_LINK)`.

### 5.3 Algorithm (deterministic; precedence BLOCKED → CONFLICT → UNKNOWN → VERIFIED)

1. `R5-SEC-01` not `APPROVED_VERIFIED` with option A → `BLOCKED(RESOLVER_NOT_AUTHORISED)`.
2. Purpose not in §5.2 → `BLOCKED(PURPOSE_NOT_PERMITTED)`.
3. `account_id` empty or not an identity account id → `UNKNOWN(NO_ACCOUNT)`. Names, phones, e-mails and logins are not
   accepted as input at all (requirement 2).
4. Account designated `NON_PERSONAL` (service, automation, bot, shared mailbox) → `BLOCKED(NON_PERSONAL_ACCOUNT)`.
5. Legacy `user_link_state = SHARED_USER` for the account at `effective_at`, or an `OPEN` E6 finding touching the
   account or its person → `CONFLICT(SHARED_ACCOUNT / OPEN_FINDING)` (requirement 4).
6. Select links of the account with state `EFFECTIVE` or later-`REVOKED` whose `[valid_from, valid_to)` contains
   `effective_at` and that were **recorded** before `effective_at` (no future knowledge). More than one → `CONFLICT(OVERLAPPING_LINKS)`.
   None → `UNKNOWN(NO_VERIFIED_LINK)`; a `PROPOSED` / `UNDER_REVIEW` link → `UNKNOWN(LINK_PENDING)` (requirement 5).
7. A revocation with `effective_at_revocation ≤ effective_at` → `UNKNOWN(LINK_REVOKED)`; evidence expired or
   superseded at `effective_at` → `UNKNOWN(EVIDENCE_EXPIRED)` (requirements 6, 7).
8. Purpose not on the link → `BLOCKED(PURPOSE_NOT_ON_LINK)` (requirement 8).
9. Person `MERGED_INTO` → `VERIFIED` with the survivor id **and** an E6 flag; `SPLIT_SOURCE` → `CONFLICT`.
10. Otherwise `VERIFIED(verified_person_id, evidence_refs, link_id, validity, registry_revision)` (requirements 1, 9).

Several accounts of one human resolve to the same `verified_person_id` (requirement 3). The resolver never compares
display names; homonyms resolve to distinct ids.

### 5.4 Requirement traceability

| # | Requirement | Rule | Spec test |
|---|---|---|---|
| 1 | Human person, not account | step 10 returns `verified_person_id` ≠ `account_id` | `TestRME04IResolverSpec` "two accounts one person" |
| 2 | Display name never evidence | step 3; no name parameter | "display name ignored" |
| 3 | Multiple accounts | step 6 + no unique on person | "two accounts one person" |
| 4 | SHARED_USER rejected | step 5 | "shared user" |
| 5 | Fail closed | steps 3, 6 | "no link", "pending link", "overlapping links" |
| 6 | Stale / revoked rejected | step 7 | "revoked", "expired evidence" |
| 7 | Historical effective time | steps 6, 7 use `effective_at` | "act before revocation stays verified", "act before link" |
| 8 | Purpose restrictions | steps 2, 8 | "unknown purpose", "purpose not on link" |
| 9 | Traceable evidence | step 10 output | "verified carries evidence refs" |
| 10 | Grants nothing | §5.1 | `TestRME04INoRuntimeResolverOrIdentityActivation` (no runtime symbol) |

The spec model in `rm_e04i_test.go` (`rmE04ISpecResolve`) is **test-only executable specification** over synthetic
data. It is not wired to any gate (the repository keeps `rmE04GRepositoryResolver`, which resolves nothing).

### 5.5 Concurrency, reassignment and revocation races

| # | Race | Rule |
|---|---|---|
| RC-1 | Two reviewers make overlapping effective links for one account concurrently | DB exclusion constraint (E4) + `SERIALIZABLE` command transaction; loser fails, no partial state |
| RC-2 | Account reassigned to another human | Old link `REVOKED(REASSIGNMENT)` with `effective_at = t`; new link `valid_from ≥ t`; acts before `t` resolve to the old person, after `t` to the new one; overlap → CONFLICT |
| RC-3 | Revocation recorded *after* an act it should have prevented (retroactive) | The act keeps its recorded snapshot (no rewrite). A `…retroactive_revocation_flagged` event lists every decision whose snapshot used the link; consumers re-evaluate (flag only); Security decides consequences |
| RC-4 | Resolve-then-sign TOCTOU | The signer's resolution is snapshotted into the R6 identity snapshot (F-PR1) with `registry_revision`; gate evaluation re-resolves at the act's `effective_at`; snapshot ≠ re-resolution → CONFLICT |
| RC-5 | Link created after the act but back-dated | Step 6 requires `recorded_at < effective_at`; a back-dated link cannot validate an earlier act |
| RC-6 | Evidence expiry during a multi-day decision | Each participant resolves at their own act time; the record shows each validity window |
| RC-7 | Merge while a decision is open | Resolution through the survivor + E6 flag; if two signers become one person, SD-R1 marks the decision `PERSON_INDEPENDENCE_VIOLATED` on re-evaluation (flag, no rewrite) |
| RC-8 | Caching | No cache for security purposes, or cache keyed by `(account_id, effective_at, purpose, registry_revision)`; any registry write increments the revision |
| RC-9 | Clock skew | `effective_at` comes from the act's recorded timestamp, never the client; future `effective_at` → `UNKNOWN(NOT_YET_EFFECTIVE)` |

---

## 6. HR verification workflow (PROPOSED; holders and authority are HR + Security decisions, HQ-1 / HQ-2 / HQ-6)

The workflow states below are **proposed design states for review**, not adopted business statuses (AGENTS.md:
Claude does not invent business statuses or role authority). Every "authority" cell names the decision that must
define it.

```
HR NOMINATION ─▶ EVIDENCE SUBMISSION ─▶ INDEPENDENT REVIEW ─▶ VERIFIED LINK ─▶ EFFECTIVE ─▶ REVOKED / EXPIRED
       │                 │                     │  │
       │                 └◀── CORRECTION ◀──────┘  └─▶ REJECTED
       └──────────────▶ CONFLICT HOLD (E6) ◀── any stage ──▶ resolved: MERGE / SPLIT / NO CHANGE / REVOKE
```

| # | Transition | Actor | Required authority | Required evidence | Separation of duties | Audit event | Rollback |
|---|---|---|---|---|---|---|---|
| W1 | — → NOMINATED (candidate person + account) | HR maker | HR link-maker position (HQ-6) | candidate `organization.person` ref, account id | maker ≠ subject (by person) | `…link_proposed` | withdraw → `REJECTED(WITHDRAWN)` |
| W2 | NOMINATED → EVIDENCE_SUBMITTED | HR maker | same | E2 rows with reference + hash | maker ≠ subject | `…evidence_submitted` | supersede evidence |
| W3 | EVIDENCE_SUBMITTED → UNDER_REVIEW | system on submit | — | completeness check | — | `…link_reviewed` (opened) | back to W2 |
| W4 | UNDER_REVIEW → VERIFIED | Independent checker | Security- or HR-checker position (HQ-6) | re-hashed evidence, method per HQ-2 | checker ≠ maker ≠ subject (persons); checker resolved VERIFIED; never automation | `…link_review_approved` | revoke (W8) |
| W5 | UNDER_REVIEW → REJECTED | checker | same | reason | as W4 | `…link_review_rejected` | new nomination (W1) |
| W6 | UNDER_REVIEW → CORRECTION_REQUESTED → EVIDENCE_SUBMITTED | checker → maker | as W2 / W4 | correction evidence | as W4 | `…link_correction_requested` | — |
| W7 | VERIFIED → EFFECTIVE | system at `valid_from` | — | no OPEN E6 finding; exclusion constraint | — | `…link_effective` | revoke (W8) |
| W8 | EFFECTIVE → REVOKED | HR or Security revoker | revoker position (HQ-6, HQ-7) | reason + evidence | revoker ≠ subject | `…link_revoked` | **none** — a new link needs W1–W7 |
| W9 | EFFECTIVE → EXPIRED | system at evidence / link `valid_to` | — | expiry | — | `…link_expired` | re-verification (W2–W7) |
| W10 | any → CONFLICT HOLD | any participant or detector | — | E6 finding | opener may not resolve | `…conflict_opened` | — |
| W11 | CONFLICT HOLD → resolved | two persons: resolver + independent checker | Security + HR (HQ-6) | merge / split evidence | neither is a subject; ≠ opener | `…conflict_resolved`, `…person_merged` / `…person_split` | inverse correction (split after merge), append-only |

Hard rules: no self-verification (W1, W4, W8, W11 refuse a subject actor); no automatic merging on matching names,
phone numbers or e-mails — matches may only *suggest* candidates to W1; shared and service accounts are designated
`NON_PERSONAL` (HQ-4) and never reach W4; a role holder with several positions still counts as one person in every
SoD check.

### 6.4 Bootstrap

The first checkers cannot be resolved by a registry that does not yet exist. RM-E04H §6.4 applies unchanged: the
initial verifier set is established by signed appointment instruments (CEO authority needs documentary proof) plus
an independent-channel identity check recorded by a T5 verifier; the first links are made for the checkers
themselves by *other* checkers, and none may verify their own link.

---

## 7. R6 / F1 integration contract (PROPOSED; the current R6 validator is not replaced here)

| Integration point | Future behaviour | Existing anchor |
|---|---|---|
| R6 approval records | F-PR1 adds optional `account_ref`, `person_ref`, `identity_evidence_ref`, `identity_snapshot` per participant (RM-E04H §6.1); strict unknown-field rejection kept | `rmE04GR6V1Fields`, `TestRME04GApprovalRecordsCarryOnlyR6V1Fields` |
| RM-E04G evaluator | F-PR2 replaces `rmE04GRepositoryResolver` with an adapter over `resolvePerson(…, purpose)` at each act's `effective_at`; `rmE04GClaimsFromRecord` replaced by an F-PR1 adapter | `rmE04GResolve`, `rmE04GIndependence` (unchanged rules S1–S12) |
| T5 verification | `verifier_person_ref` resolved with `T5_VERIFICATION`; must differ from every approver, delegator and the recorder | rule 5 in `rmE04GIndependence` |
| F-PR1 schema extension | additive, optional on v1; mandatory only where a reviewed decision says so | RM-E04H §6.1 |
| F-PR2 identity resolution | resolver bound to `APPROVED_VERIFIED` R5-SEC-01 (A) | RM-E04H §6.5 |
| Historical approval evidence | v1 records never rewritten; evaluate `PERSON_INDEPENDENCE_UNKNOWN`; later attestation is a new appended artefact | RM-E04H §6.3, `TestRME04FHistoricalApprovalNotRewritten` |
| Authority assignment | RM-E04F activation needs `VERIFIED` holder + independent verifier (I8) | RM-E04F §5 |
| Delegated approvals | both delegate and principal resolved with `DELEGATION_PRINCIPAL`; both reserved | rule 8 / I4 |

Required behaviours and how the spec tests demonstrate them through the **existing** evaluator fed by the spec
resolver (`TestRME04IOptionAFeedsExistingEvaluator`):

| Behaviour | Outcome |
|---|---|
| One verified person counts once per decision version | two roles signed by one person → `PERSON_INDEPENDENCE_VIOLATED` |
| Separate accounts of the same person give no extra signature | two accounts of one person → `VIOLATED` |
| Homonyms not falsely rejected | distinct verified persons sharing a display name → `VERIFIED` |
| Recorder cannot self-verify through another account | recorder and T5 on two accounts of one person → `VIOLATED` |
| T5 independent of all signers | T5 = approver person → `VIOLATED` |
| UNKNOWN / CONFLICT never satisfy a gate | unlinked or shared participant → `UNKNOWN` / `CONFLICT` |
| Historical v1 not rewritten | v1 claims (no account refs) → `UNKNOWN` |

---

## 8. D-42 security-dependency hardening specification (PROPOSED; consumer unchanged in this PR)

### 8.1 The current route (VERIFIED at `739db62`)

| Element | Location | Behaviour |
|---|---|---|
| DB guard | `internal/identity/migrations/sql/00001_identity.sql`, `CONSTRAINT role_grant_shadow_only CHECK (status = 'SHADOW')` | refuses every non-SHADOW grant |
| Runtime reader | `internal/identity/store/store.go`, query `… FROM identity.role_grant WHERE user_id = $1 AND status = 'ACTIVE' …` | would honour ACTIVE grants as soon as the CHECK is gone |
| Unlock rule | `docs/governance/rm_e04d_test.go`, `grantConstraintProblems(migrations, d42)` | if `d42.Status == "APPROVED_VERIFIED"` **any** migration may touch `role_grant` |
| Migration scan | same file, `identityMigrations(t)` | scans only `internal/identity/migrations/sql` |
| Test | `TestRME04DNoActiveGrantPathWithoutD42` | applies the rule to the repository |
| Signer floor | `docs/governance/m2_test.go`, `m2Approvers["D-42"]`, `TestM2TrackerUnchanged` | all four D-42 roles must sign, **by name** |

Gap: a D-42 record signed name-level (one human under several spellings, RM-E04G F1-A) plus a T5 that cannot
establish identities would reach `APPROVED_VERIFIED`, after which a migration dropping `role_grant_shadow_only` passes
every repository check (RM-E04H G1–G5). `TestRME04ID42GapIsDocumentedWhileOpen` keeps this section honest: while the
unlock still accepts a bare D-42 status, this section must say so.

### 8.2 Required preconditions for any future `role_grant` unlock

| # | Precondition | Check (future) |
|---|---|---|
| H-1 | R5-SEC-01 `APPROVED_VERIFIED` with option A (or B) | tracker + record via `rmE04CDecisionProblems` |
| H-2 | Verified person-account resolver in force (F-PR2 merged) | resolver is not `rmE04GRepositoryResolver` |
| H-3 | Person-level independence of the D-42 record | `rmE04GIndependence(claims, m2Approvers["D-42"], resolver) == PERSON_INDEPENDENCE_VERIFIED` |
| H-4 | D-42 `APPROVED_VERIFIED`, and the migration matches the **selected option** | selected option ↔ allowed grant writer (G4) |
| H-5 | E-7 grant-change procedure accepted with runtime evidence | evidence reference present and verified |
| H-6 | D-34 `APPROVED_VERIFIED` where the change provisions credentials | conditional on credential write |
| H-7 | Independent Group C authorization | separate record |
| H-8 | Separate production deployment authorization | separate record; never implied by H-1…H-7 |

### 8.3 Exact future change list (for an approved PR-D only)

| File | Function / object | Change |
|---|---|---|
| `docs/governance/rm_e04d_test.go` | `grantConstraintProblems` | replace the bare `d42.Status == "APPROVED_VERIFIED"` early return with `grantUnlockProblems(recs, load, claimsOf, resolve, evidence)` covering H-1…H-6 |
| same | `identityMigrations` | scan every `internal/*/migrations/sql` directory (G5) |
| same | `TestRME04DGrantConstraintRules` | add the negative fixtures below; keep every existing case |
| `docs/governance/rm_e04g_test.go` | none | evaluator reused unchanged |
| `docs/governance/m2_test.go` | none | signer floor kept |
| `internal/architecture/lint.go` | new rule (proposal) | forbid `status = 'ACTIVE'` grant writes outside the future identity grant command |
| `internal/identity/migrations/sql/` | none in PR-D | CHECK removal belongs to Group C, after H-7 |

### 8.4 Regression and negative tests (to be written in PR-D)

| # | Fixture | Expected |
|---|---|---|
| ND-1 | D-42 `APPROVED_VERIFIED`, R5-SEC-01 not approved | blocked (H-1) |
| ND-2 | Both approved, repository resolver still in place | blocked (H-2) |
| ND-3 | D-42 signed by one person under three spellings / accounts | blocked (`VIOLATED`, H-3) |
| ND-4 | D-42 T5 = an approver on another account | blocked (H-3) |
| ND-5 | Any D-42 participant UNKNOWN / CONFLICT | blocked (H-3) |
| ND-6 | Selected option "Identity / Security only" + HR grant-writer migration | blocked (H-4) |
| ND-7 | No E-7 evidence | blocked (H-5) |
| ND-8 | Credential write without D-34 | blocked (H-6) |
| ND-9 | `role_grant` touched from a non-identity migration directory | detected (G5) |
| ND-10 | All H-1…H-6 hold | the migration rule passes; deployment still needs H-7 / H-8 |
| ND-11 | Comment-only mention of `role_grant` | still ignored (existing behaviour kept) |

---

## 9. Dependency graph (human-readable; machine-readable file is authoritative for tests, not for governance)

| From (prerequisite) | To | Type | Source |
|---|---|---|---|
| D-34, D-42, ADR #18 | R5-SEC-01 | DECISION | DOCUMENTED (tracker) |
| R5-SEC-01 | D-34 | DECISION | DOCUMENTED (tracker) |
| R5-SEC-01 | D-42 | DECISION | DOCUMENTED (tracker) |
| E-7 | D-42 | TECHNICAL | DOCUMENTED (tracker) |
| EV-019, EV-020, Legal privacy review, T5 | R5-SEC-01 | EVIDENCE | DOCUMENTED |
| R5-SEC-01 | F-PR2 | DECISION | DOCUMENTED (RM-E04H §9) |
| F-PR1 | F-PR2 | TECHNICAL | DOCUMENTED |
| F-PR2 | RM-E04G (guard relaxation), RM-E04F (activation) | TECHNICAL | DOCUMENTED |
| RM-E04G | Group B | TECHNICAL | DOCUMENTED |
| Group B | S4 | TECHNICAL | DOCUMENTED |
| D-42 | Group C | DECISION | DOCUMENTED |
| D-34 | Group C | CONDITIONAL (provisioning) | DOCUMENTED |
| F-PR2 | D-42 hardened unlock | CONDITIONAL | PROPOSED |
| ADR #18 | R5-SEC-01 | CONDITIONAL (real HR data) | PROPOSED |
| Group B | Group C | TECHNICAL | PROPOSED |
| Group C, S4 | Production | DEPLOYMENT | DOCUMENTED / PROPOSED |

**Circular dependency — UNRESOLVED.** The tracker lists D-34 and D-42 as prerequisites of R5-SEC-01 *and* R5-SEC-01
as a prerequisite of D-34 and D-42 (RM-E04H OF-4; `unresolved_conflicts` OF-4a, OF-4b in the JSON).
`TestRME04IDependencyGraphRecordsCycle` recomputes the cycles from the documented edges and fails if one is not
listed as UNRESOLVED.

Recommended ordering (non-authoritative; tracker unchanged): evidence → **R5-SEC-01** (+ T5) → F-PR1 → F-PR2 →
guard relaxation / authority activation → E-7 → D-42 / D-34 → Group B → S4 → Group C → Production. Adopting it needs
an inventory-owner decision (HQ-10).

---

## 10. Future implementation PR sequence (plans only; none is authorized)

Mapping to RM-E04H §9: PR-A = F-PR1; PR-B = F-PR5 moved earlier as *schema + synthetic only*; PR-C = F-PR2;
PR-D = F-PR3; PR-E = new; PR-F = F-PR6. The move of PR-B before PR-C is a proposal (the resolver needs a model to
read); it activates nothing.

| | PR-A | PR-B | PR-C | PR-D | PR-E | PR-F |
|---|---|---|---|---|---|---|
| Title | Additive R6 identity-evidence schema | Shadow-only Option A link + verification model | Person resolver + identity-independent R6 validation | D-42 security-gate hardening | Shadow parity checks + audit evidence | Administrative identity-verification interface |
| Dependencies | RM-E04H H-RV-09 / H-RV-10 answered; R5-SEC-01 decision recorded (any option) | R5-SEC-01 `APPROVED_VERIFIED` = A; Legal privacy review; Codex + DBA architecture review | PR-A, PR-B merged | PR-C; H-RV-08 / I-RV-07 decided | PR-B, PR-C; ADR #18 if real data | PR-B; capability decision for the screens (not Claude's to invent) |
| Allowed scope | R6 record struct, `rmE04GR6V1Fields`, validator shape checks | organization migrations for E1–E7 (append-only triggers, exclusion constraint), commands behind a disabled flag, synthetic fixtures | read-only organization query API; F-PR1 adapter; replace `rmE04GRepositoryResolver` / `rmE04GClaimsFromRecord` | `rm_e04d_test.go` unlock rule + fixtures; lint rule | read-only parity job: legacy `employment.user_link_state` vs registry; audit report | UI over PR-B commands using the frozen design system; synthetic data first |
| Affected dirs | `docs/governance/` | `internal/organization/{migrations,domain,store,api}` | `internal/organization/query`, `docs/governance/` | `docs/governance/`, `internal/architecture/` | `internal/organization/`, `docs/evidence/` | `web/`, `api/openapi/` |
| Tests | v1 still valid; unknown keys refused; dangling refs refused | append-only triggers; overlap refused; no FK to identity; no name / phone column; app role SELECT-only | `TestRME04IResolverSpec` cases against the real implementation; F1-A…F1-H | ND-1…ND-11 | parity zero-diff on synthetic; mismatches reported, not fixed | RTL / a11y; purpose-limited lookups audited; no evidence leak |
| Acceptance | all existing governance tests green | migration up/down on empty DB; zero rows written by import | RM-E04G-I1 relaxed only per RM-E04H §6.5 | G1–G5 each killed by a fixture | report signed by HR + Security reviewers | Security + HR + design review |
| Data migration risk | none (additive, optional) | low (new empty tables); **never** touches `organization.person` | none (read-only) | none | reads legacy only | none until real data |
| Rollback | revert PR | down migration drops empty tables (forbidden once rows exist: then disable flag) | revert to repository resolver (fail closed) | revert (stricter rule removed → back to today) | delete report | disable flag |
| Independent reviews | Codex, Security | Codex, Security, HR, Legal, DBA | Codex, Security, T5 of R5-SEC-01 record | Codex, Security | HR, Security | Security, HR, Product, design |
| NO-GO if | any identity field mandatory for v1; strict validation weakened | R5-SEC-01 not approved for A; any link created from import or matching; any grant / credential write | resolver returns VERIFIED for name / phone / e-mail; any BLOCKED / UNKNOWN / CONFLICT satisfies a gate | any existing D-42 test weakened | auto-correction of links | screens invent permissions or show evidence content |

Production remains NO-GO after all six PRs; it needs its own deployment authorization (H-8).

---

## 11. Review and human decisions

Review packet: `docs/implementation/rm-e04i-review-handoff.md` (Codex, Security, HR, Legal, Product; every item
PENDING). Authority requirements re-checked against the repository (VERIFIED by `TestRME04ISignerFloorPreserved`):

- R5-SEC-01 needs independent Security, HR, Legal **and** Product approvals (`m2Approvers`, intake blank);
- T5 must be independent of every signer and the recorder (evidence standard; `rmE04GIndependence` rule 5);
- current nominations are provisional (`NOMINATED` / `UNASSIGNED`); HR owner and T5 verifier have no nominee;
- CEO appointment authority is a design input that needs documentary proof (register `holder_status = UNASSIGNED`);
- no candidate name substitutes for a signed appointment; a nominee's review is not an office holder's review.

Open human questions: RM-E04H HQ-1…HQ-12 unchanged, plus:

| # | Question | Owner |
|---|---|---|
| IQ-1 | Accept organization (HR) as owner module of the person registry, with Security owning link policy? | HR + Security + Codex |
| IQ-2 | Accept the proposed workflow states W1–W11 as business states? | HR + Security |
| IQ-3 | Accept the closed purpose list §5.2? | Security + Legal |
| IQ-4 | Accept PR-B before PR-C (schema before resolver)? | Codex |
| IQ-5 | Accept H-1…H-8 as the D-42 unlock preconditions? | Security + D-42 approvers |

---

## 12. Governance test evidence

### 12.1 Committed tests (`docs/governance/rm_e04i_test.go`)

| Test | Safety property |
|---|---|
| `TestRME04IPreferenceIsNotADecision` | preference record stays non-authoritative: A preferred, never APPROVED / VERIFIED / ACTIVE / GRANTED; no signature, no T5; four original options verbatim |
| `TestRME04IPreferenceIsNotConsumed` | no Go source other than this test, no gate / manifest / tracker / register / approval file references the record; a tracker option needs an approval record |
| `TestRME04INoRuntimeResolverOrIdentityActivation` | while R5-SEC-01 is not `APPROVED_VERIFIED`: no migration defines a person-registry table, no non-test Go defines a person resolver, repository resolver resolves nothing, nominees carry no person ref |
| `TestRME04IGatesStayClosedWithoutPersonIndependence` | Group B / S4 readiness false without a resolver; runtime seller bundle and contract consistent with person-gated readiness; no ACTIVE grant path without D-42 |
| `TestRME04ID42GapIsDocumentedWhileOpen` | while the unlock accepts a bare D-42 status, §8 says so and names the exact functions; the functions still exist |
| `TestRME04IResolverSpec` | executable contract §5 over synthetic data (fail-closed cases) |
| `TestRME04IOptionAFeedsExistingEvaluator` | §7 behaviours through the unchanged RM-E04G evaluator |
| `TestRME04IDependencyGraphRecordsCycle` | graph well-formed, tracker prerequisites represented, every documented cycle listed UNRESOLVED, graph non-authoritative |
| `TestRME04ISignerFloorPreserved` | four R5 approver roles, T5 independence, provisional nominations stated |
| `TestRME04IDocumentsMarkedNotApproved` | DRAFT marker, final status lines, no claimed sign-off, handoff PENDING |

All repository-state checks are **conditional** on the governing decision (e.g. "while R5-SEC-01 is not
`APPROVED_VERIFIED`"), so a future legitimate approval and its authorized PRs are not blocked by these tests.

### 12.2 Runs (actual results in this worktree)

| Run | Result |
|---|---|
| `go test -count=1 ./docs/governance/...` (baseline, before changes) | PASS |
| `go test -count=1 ./docs/governance/...` (with this package) | PASS (10 new tests) |
| `R5_REQUIRE_PROVENANCE=1 go test -count=1 ./docs/governance/` | first run FAIL on a **shallow clone** (4 provenance tests: "the clone is shallow") — environment, not this PR; after `git fetch --unshallow`: PASS |
| `go build ./...`, `go vet ./...`, `gofmt -l docs/governance/` | clean |
| `go test ./internal/architecture/... ./internal/authorization/... ./internal/identity/... ./internal/organization/...` | PASS (no source changed) |

### 12.3 Mutation checks (throwaway copy in the session scratchpad; nothing committed)

| # | Mutation | Killed by |
|---|---|---|
| MI1–MI5 | Preference record: decision `APPROVED_VERIFIED`; signature added; T5 added; official layer = A; consumer added | `…PreferenceIsNotADecision` |
| MI6 | Runtime Go file references the preference record | `…PreferenceIsNotConsumed` |
| MI7 | Tracker R5-SEC-01 `selected_option = A` without a record | `…PreferenceIsNotConsumed` + existing R6 / M2 tests |
| MI8 | Organization migration adding `verified_person_account_link` | `…NoRuntimeResolverOrIdentityActivation`, RM-E04H state test |
| MI9 | Runtime `ResolvePerson` added | `…NoRuntimeResolverOrIdentityActivation` (only) |
| MI10 | Nominee `person_ref` filled | RM-E04F / RM-E04H / RM-E04I |
| MI11 | Repository resolver returns VERIFIED | RM-E04G / RM-E04H / RM-E04I |
| MI12–MI16 | Spec resolver: SHARED_USER check, back-dating check, revocation, purpose restriction, authorisation check removed | `…ResolverSpec` (MI12, MI14, MI16 also `…FeedsExistingEvaluator`) |
| MI17 | Adapter treats BLOCKED as VERIFIED | `…FeedsExistingEvaluator` (only) |
| MI18–MI21 | Graph: cycle removed; cycle marked RESOLVED; tracker edge dropped; `authoritative: true` | `…DependencyGraphRecordsCycle` |
| MI22–MI25, MI31 | Final status promoted; handoff item ACCEPTED; handoff status changed; a fake Security sign-off claim added; DRAFT marker removed | `…DocumentsMarkedNotApproved` |
| MI26 | D-42 gap statement removed while the gap is open | `…D42GapIsDocumentedWhileOpen` (only) |
| MI27 | Legal owner dropped from `m2Approvers` | `…SignerFloorPreserved` + existing |
| MI28 | CEO authority `ASSIGNED` without evidence | `…SignerFloorPreserved`, RM-E04F |
| MI29 | Seller bundle gains `cases.own.read` | `…GatesStayClosedWithoutPersonIndependence`, RM-E04C |
| MI32 | Preference purpose drops "NOT implementation authorization" | `…PreferenceIsNotADecision` |

31 of 31 mutations killed; none survived. A positive fixture checks that the activation rule does **not** block
PR-B / PR-C once R5-SEC-01 is `APPROVED_VERIFIED`.

---

## 13. Final status

```
R5-SEC-01   = PREFERRED_A / NOT_APPROVED
F1          = INTERIM_GUARD_ONLY
Group B     = BLOCKED
S4          = BLOCKED
Real grants = BLOCKED
Production  = NO-GO
```
