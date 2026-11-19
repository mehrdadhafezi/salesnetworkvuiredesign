# RM-E04F — Codex / Security architecture review handoff

**DRAFT — NOT APPROVED.** Review questions for the RM-E04F authority and role-assignment architecture
(`docs/governance/proposals/rm-e04f-authority-role-assignment-architecture.md`). No review has been performed or
recorded. Only the reviewers may change a question's status. **Production: NO-GO.**

| | |
|---|---|
| Status | **REVIEW PENDING.** |
| Requested reviewers | Codex (architecture, workflow contract, data model); Security owner (SoD, threat model, privacy) |
| Baseline | `main` @ `b3dc3ed5fe8a7286bb2baf999da0d2ec62bba8c3` |
| Scope | Documentation, nomination register, governance tests. No runtime, schema, grant or gate change |

Every row below is **PENDING**.

| # | Area | Question | Design reference | Status |
|---|---|---|---|---|
| RQ-01 | Domain model compatibility | Is a new `governance` module the right owner of Position / AuthorityAssignment / Delegation, given that `organization` owns HR positions and `identity` owns accounts and grants? | §2.4, §3.2, §8.1 | PENDING |
| RQ-02 | Domain model compatibility | Should governance offices live in a separate catalogue (`position_kind = GOVERNANCE_OFFICE`) or extend `organization.position`? | §4.2 Position | PENDING |
| RQ-03 | Authority source of truth | Repository registry first, DB projection later, repository authoritative until a recorded decision (AD1) — acceptable? | §8.8, §11.3 | PENDING |
| RQ-04 | Authority source of truth | Root of trust: external legal instrument for the CEO position; no self-appointment (rule SA). Gaps? | §5.4 | PENDING |
| RQ-05 | Person identity / multi-account | Is `resolve(user_id, at)` with UNKNOWN / CONFLICT fail-closed the correct contract for R5-SEC-01 options A and B? What happens under option C or D? | §6.1, §11.1 | PENDING |
| RQ-06 | Person identity / multi-account | `SHARED_USER` → CONFLICT, `USER_NOT_*` → UNKNOWN: complete list? | §2.3 F3 | PENDING |
| RQ-07 | Separation of duties | Rules I1–I8: sufficient? Should the T5 verifier also differ from the appointer of any signer's authority? | §6.2 | PENDING |
| RQ-08 | Separation of duties | A delegated signature reserves both delegate and delegator (I4) — too strict or correct? | §5.3 DL9 | PENDING |
| RQ-09 | Revocation propagation | Position end → authority end → delegation end in one transaction; emergency revocation with compromise window flags only unverified decisions. Acceptable? | §5.1, §5.5 | PENDING |
| RQ-10 | Temporal integrity | `tstzrange` + exclusion constraints (`btree_gist`) as a new convention (AD8); activation never retroactive; revocation at commit time | §8.2, §8.3 | PENDING |
| RQ-11 | Database constraints | Are the CHECK / UNIQUE / EXCLUDE constraints and append-only / forward-only triggers sufficient, and which rules must stay in commands? | §8.2, §8.3 | PENDING |
| RQ-12 | Race conditions | `FOR SHARE` / `FOR UPDATE` + per-decision advisory lock + SERIALIZABLE — sufficient for sign / revoke / verify races? | §8.4 | PENDING |
| RQ-13 | R6 evidence compatibility | Additive optional `person_ref`, `authority_assignment_ref`, `sha256`, `verifier_person_ref`; existing records valid unchanged | §11.2 | PENDING |
| RQ-14 | R6 evidence compatibility | Name-keyed independence weakness (F1): accept the T5 checklist as interim control, or require F-PR2 before any multi-signer approval? | §2.3 F1 | PENDING |
| RQ-15 | Privilege escalation | Governance authority never reaches `authorization.Evaluate`; admin screens need a new capability (AD4) not in any existing bundle. Any path from authority to grant? | §3.1, §8.5, §11.1 | PENDING |
| RQ-16 | Privilege escalation | Can the nomination register or an R6 extension be abused to open the Group B gate or S4? | §11.1; `rm_e04f_test.go` | PENDING |
| RQ-17 | Safe migration / rollback | Shadow-only schema behind `governance.authority.shadow`, goose Down drops the schema; no existing row migrated | §8.8, §12 | PENDING |
| RQ-18 | Privacy | Hash + locator only; no document content; visibility rules PR1–PR5; link to ADR #18 / D-45 | §8.7 | PENDING |
| RQ-19 | Workflow | Is anything missing from T-A1…T-A8 (e.g. re-verification on evidence expiry, periodic attestation)? | §5.2 | PENDING |
| RQ-20 | Admin UX | Actor columns for screens 1–10 are proposals; which need their own decision? | §10 | PENDING |

## Not claimed

- No Codex review, Security review, architecture acceptance or human verification exists for this package.
- The nominations of مهرداد حافظی are provisional; they are not appointments or authority evidence.
- Group B gate closed; S4 manifest empty; no approval record exists; no grant is active.
