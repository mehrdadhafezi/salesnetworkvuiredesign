# RM-E04N — CurrentHolder security characterization

**EVIDENCE ONLY — NOT APPROVED.** This document records how the code on `main` behaves today. It selects no ownership
policy, approves no predicate, opens no gate and authorizes no Seller read. M2 records 1 and 2 remain
`NOT_APPROVED_VERIFIED`; Group B stays **closed**; production **NO-GO**.

| | |
|---|---|
| Baseline | `main` @ `71847fcb55963072231b82a22658ab48bf7d4f7f` (PR #85, RM-E04L, merged) |
| Tests | [`internal/custody/query/current_holder_characterization_test.go`](../../internal/custody/query/current_holder_characterization_test.go) (7 tests, disposable PostgreSQL via `pgtest`, SYNTHETIC data) |
| Paths exercised | real `custody/imports` and `cases/imports` commands → real `custody/query.CurrentHolder`; real `cases/command.Eligible` and `RecordContactOutcome` (`SHADOW_COMPARE`) through an adapter identical to `cmd/crm-api/writes.go` `holderAdapter` |
| Runtime changes | none (no Go runtime, SQL migration, OpenAPI, grant, gate, tracker or approval change) |
| Input | RM-E04M assessment (owner-supplied); every claim below was re-checked against source |

## 1. Test classes

- **CURRENT_BEHAVIOR_CHARACTERIZATION** — all 7 tests. They pin what the code does now. A passing assertion is
  **not** an approved policy. Where behaviour is flagged UNRESOLVED or POTENTIALLY_UNSAFE it is recorded, not endorsed.
  An authorized future change to `CurrentHolder` is expected to break the matching assertion; the change must update
  the test and this document together.
- **SECURITY_REQUIREMENT_PENDING_IMPLEMENTATION** — §6 only, as non-gating specifications. None is encoded as a
  failing or skipped test.

## 2. Behaviour matrix

`Known` = `CurrentHolder` names exactly one user. All rows are produced by the real import unless marked *simulated*.

| Input fact (derivation / parity / presence) | Current result | Risk | Evidence | Proposed future disposition |
|---|---|---|---|---|
| DERIVED / MATCH / PRESENT, legacy user resolved | Known, legacy user, `LEGACY_HOLDER_COLUMNS` | — corroborated | `TestRME04NParityMatrix` 701 | human decision (admit set) |
| DERIVED / HISTORY_GAP | **Known** | UNRESOLVED — admitted although history has a gap | 702 | human decision |
| NONE / TARGET_UNKNOWN (no events, legacy holder) | **Known** | UNRESOLVED — legacy column only, no corroboration. **Every lead case is in this class** (lead custody has no event log) | 707; 801 in `TestRME04NEligibilityDistinguishesRefusals` | human decision |
| UNKNOWN / TARGET_UNKNOWN (only an UNKNOWN-kind event) | **Known** | UNRESOLVED — admitted although derivation is UNKNOWN | 708 | human decision |
| DERIVED / CONFLICT | not Known, `CONTRADICTING_HOLDER_EVIDENCE` | fail closed | 703 | keep |
| RELEASED / CONFLICT (legacy still names holder) | not Known, `CONTRADICTING_HOLDER_EVIDENCE` | fail closed | 709 | keep |
| MULTIPLE / MULTIPLE_CURRENT | not Known, `CONTRADICTING_HOLDER_EVIDENCE` | fail closed | 704 | keep |
| DERIVED / LEGACY_UNKNOWN (derived holder, no legacy holder) | not Known, `NO_RESOLVED_LEGACY_HOLDER` | fail closed; derived holder ignored | 706 | human decision (P1 vs P2) |
| NONE / NOT_RECOVERABLE | not Known, `NO_RESOLVED_LEGACY_HOLDER` | fail closed | 705 | keep |
| MATCH, legacy key without resolved user id | not Known, `NO_RESOLVED_LEGACY_HOLDER` | fail closed | 710 | keep |
| MATCH by legacy **key**, but holder column → user A, event → user B | **Known = user A** | POTENTIALLY_UNSAFE — parity compares keys, not user ids; divergence is invisible to the guard | 711 | engineering fix after authorization |
| Parity / derivation value outside the current CHECK lists (*simulated*: constraint dropped in the disposable DB only) | **Known** | POTENTIALLY_UNSAFE — deny-list is fail-open by construction; today's schema rejects such values (proved first) | `TestRME04NHypotheticalParityAndDerivationAdmitted` | engineering fix (allow-list) after authorization |
| ABSENT_IN_LATEST (complete batch omitted the case) | not Known, `CASE_ABSENT_IN_LATEST` | fail closed | `TestRME04NPresence` 730 | keep |
| Reassigned by a COMPLETE batch | Known = new holder on next read | as designed | 731 | keep |
| Case without custody record / nonexistent case id | not Known, `NO_CUSTODY_RECORD` (identical) | — | 732, random id | keep |
| Holder not in a later PARTIAL batch (old `refreshed_at`) | Known, old holder, still PRESENT | UNRESOLVED — stale indefinitely; no threshold | `TestRME04NFreshnessNotConsidered` 805 | human decision (window) + engineering |
| Re-confirmed unchanged in a later batch | Known; `refreshed_at` / `refreshed_batch_id` **not** updated | freshness signal cannot tell "re-confirmed" from "never re-seen" | 806 vs 805 | engineering (needs a last-seen fact; schema change) |
| Cases says ABSENT (work_case + case_source), custody still PRESENT | **Known** (read); `Eligible` → `STALE_STATE`; write → `BLOCKED/STALE_STATE` | read/write discrepancy | `TestRME04NCaseAbsentButCustodyPresent` | engineering (S1 read must check cases presence) |
| Holder user id not present in identity | Known, id returned as-is | identity existence/activity not verified by custody | `TestRME04NCustodyStoresUnverifiedUserIDs` | human + engineering |

## 3. Verified code facts

1. **Deny-list guard.** `CurrentHolder` (`internal/custody/query/query.go:216`) refuses only
   `derivation = MULTIPLE` or `parity ∈ {CONFLICT, MULTIPLE_CURRENT}`; it also requires `presence = PRESENT` and a
   non-NULL `legacy_holder_user_id`. Everything else is admitted. Record 2 §7.3 says rows whose ownership fact is
   UNKNOWN are "not shown and not counted"; the code admits `derivation = UNKNOWN` (708) and `NONE` (707). **The
   form text and the implementation disagree.**
2. **Freshness.** `RefreshedAt` is scanned (`store/read.go:23,36`) and never read by `CurrentHolder`.
   `UpsertAssignment` (`store/store.go:166-181`) updates `refreshed_at` and `refreshed_batch_id` only when the
   projection changes. `MarkAbsentAssignments` runs only for specs marked COMPLETE (`imports/imports.go:176-190`).
3. **No set-based held-by read.** `CurrentHolder` takes one case id. There is no published query "cases held by user
   X" and no index on `legacy_holder_user_id` (test asserts zero). A Seller list therefore cannot be built from the
   existing API without organization-wide enumeration.
4. **Cross-module presence.** `CurrentHolder` reads only `custody.assignment`. The custody import resolves cases via
   `ResolveSources`, which returns absent cases too, and does not check their presence. `Eligible`
   (`cases/command/read.go:90-122`) checks `work_case.presence`; `RecordContactOutcome`
   (`contact_outcome.go:246-276`) checks `work_case` **and** `case_source` presence. Through the cases import both
   presences move together (`cases/store/write.go:191-213`), so "work_case PRESENT, case_source ABSENT" was not
   reachable and is not tested.
5. **Adapter.** `cmd/crm-api/writes.go:53-58` drops `Basis`; the write path cannot tell why a holder is unknown.
6. **Eligibility signals differ** for nonexistent case (`ErrNotFound`), other holder (`ROW_SCOPE_DENIED`), no holder
   or no custody record (`DEPENDENCY_UNKNOWN`), absent (`STALE_STATE`), non-lead (`POLICY_UNRESOLVED`) — observed.
   Reusing these unchanged behind an own-read would break 404 parity.
7. **Identity space (static reading, not an end-to-end test).** Custody `legacy_holder_user_id` is set from
   `HolderRecord.Holder.UserID` (`imports.go:132`), which the legacy shadow integration fills from
   `identity.user_account.user_id` by `(source_system, source_key)` (`identity/store` `UserIDsByKey`, unique
   constraint). The session principal's `UserID` is `identity.session.user_id`
   (`identity/session/session.go:247-256`), and `SessionSubjects` sets `Subject.UserID` from it. **Intended space is
   the same (`identity.user_account.user_id`).** Not guaranteed by the schema: no foreign key (cross-schema), no
   check that the user exists or is active; custody accepts any well-formed uuid (test 850). The full
   import→login→own-read chain is **UNKNOWN** until tested end-to-end on the R7 harness.
8. **No organization key** on any `cases` or `custody` column (test asserts zero). Isolation is single-deployment.

## 4. Effect of changing `CurrentHolder` on contact-outcome writes

The contact-outcome write guard and `Eligible` consume `CurrentHolder` through `holderAdapter`. A local, uncommitted
mutation experiment (guard replaced by `parity != MATCH → not Known`) made all 7 tests fail and, in particular,
turned the actor's own lead (801, TARGET_UNKNOWN) from eligible into `DEPENDENCY_UNKNOWN`. Because every lead case is
TARGET_UNKNOWN, **a MATCH-only allow-list in `CurrentHolder` would disable the existing contact-outcome write for all
leads.** Narrowing the read predicate therefore either changes a current permission (needs explicit approval) or
requires a separate read predicate, which breaks "what I see = what I may act on". The mutation was reverted; the
diff contains no runtime change.

## 5. Unsupported or untested hypotheses

- Legacy "pending transfer" semantics: no transfer state exists in the target model; legacy behaviour unverified.
- Real-data class distribution (synthetic fixtures say nothing about volumes; ADR #18 governs real data).
- Whether production imports ever run custody with a spec not marked COMPLETE for long periods.
- Identity end-to-end (§3.7).

## 6. Pending security requirements (non-gating specifications)

Derived from Record 2 §7.2–§7.6 and SR-05/SR-06/SR-07/SR-11. To be implemented only after M2 approval and an
authorized engineering change; each will supersede the matching characterization assertion.

| Spec | Requirement | Today |
|---|---|---|
| PR-1 | An S1 own-read admits only an explicitly approved set of (derivation, parity); any other value fails closed (SR-05) | deny-list (fails) |
| PR-2 | An S1 own-read requires `work_case` and `case_source` PRESENT in addition to custody PRESENT | not checked by `CurrentHolder` |
| PR-3 | Out-of-scope, unknown-holder, absent and nonexistent cases produce the same 404 status and body (SR-06) | `Eligible` signals differ |
| PR-4 | Freshness: a holder fact older than the approved window is excluded or UNKNOWN; requires a last-seen fact | not representable today |
| PR-5 | Lists and counts come from one set-based scoped query; incomplete coverage → PARTIAL / "—", never 0 (SR-07) | no such query |
| PR-6 | Holder user id equality with the derived user id when both exist (key-level MATCH is insufficient) | not checked |

## 7. Human decisions still required

1. Admitted set for S1: are `TARGET_UNKNOWN` (all leads), `HISTORY_GAP`, and `derivation = UNKNOWN` acceptable?
2. Acceptable import-staleness window, and behaviour when exceeded (hide vs UNKNOWN).
3. Pending-transfer semantics in legacy and who sees a case meanwhile.
4. Cross-organization: accept single-deployment isolation as an explicit condition, or require an organization model.
5. Whether the contact-outcome write guard may change with the read predicate (§4), or the two must diverge.
6. PQ-1 (distribution-item cases in S1), PQ-6, PQ-4 / PQ-5 from Record 2 §8 remain open.

## 8. Engineering changes that would need later authorization

Allow-list in `CurrentHolder` or a new S1 resolver (changes the write guard if shared); a published set-based
held-by query and an index on `legacy_holder_user_id` (migration); a last-seen fact per assignment (migration +
import change); cases-presence check in the S1 path; 404-collapsing in `/me` handlers; user-id parity check (PR-6).
All are Group B work (B2a → B2 → B4) or need their own approval.

## 9. Reproduce

```bash
CRM_TEST_DATABASE_URL=postgres://…/postgres CRM_TEST_REQUIRE_POSTGRES=1 go test -v -run RME04N ./internal/custody/query/
```

Rollback: delete the test file and this document; nothing else depends on them.
