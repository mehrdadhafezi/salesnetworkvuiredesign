# RM-E04H — Codex / Security / HR / Legal review handoff (R5-SEC-01 person identity)

**DRAFT — NOT APPROVED.** Independent review request for
`docs/security/rm-e04h-person-identity-decision-package.md`,
`docs/governance/proposals/rm-e04h-person-identity-decision-dashboard.fa.md` and
`docs/governance/rm_e04h_test.go`. No review has been performed or recorded. Only the named reviewers may change a
row's status, and only by recording their own result. **Production: NO-GO.** Group B and S4 stay BLOCKED.

| | |
|---|---|
| Status | **REVIEW PENDING.** |
| Requested reviewers | Codex (option fidelity, R6 schema, architecture); Security owner (identity, SoD, unlock routes); HR owner (person source, proofing, transfers); Legal owner (privacy) |
| Reviewer holders | All **UNASSIGNED** (RM-E04E §1.2; nominations register) — a review by a nominee is not a review by a verified office holder |
| Baseline | `main` @ `19ab171f2c2db9aa3e88dfc0ce4a13bb3d0f4657` |
| Scope | Documentation and governance tests only. No approval record, person id, account link, status, gate, grant, schema or runtime change |

Every row below is **PENDING**.

| # | Reviewer | Question | Reference | Status |
|---|---|---|---|---|
| H-RV-01 | Codex | Option fidelity: are A–D quoted verbatim and is OF-1 (B wording variance) correctly judged meaning-neutral? | package §2 | PENDING |
| H-RV-02 | Codex + inventory owner | Dependency cycle OF-4: which direction is authoritative between R5-SEC-01 and D-34 / D-42 / ADR #18? | package §2.3, HQ-10 | PENDING |
| H-RV-03 | HR + Security | Source of truth and owner for verified persons; is the legacy HR profile a usable candidate source? | package §3, §4, HQ-1, HQ-3 | PENDING |
| H-RV-04 | HR + Security | Identity proofing method and two-person link review | package §5.3 P2, HQ-2, HQ-6 | PENDING |
| H-RV-05 | Security | Resolver contract: states, precedence (`BLOCKED` pre-lookup; CONFLICT > UNKNOWN > VERIFIED), purpose limitation | package §5 | PENDING |
| H-RV-06 | Security + HR | Shared / non-personal accounts: who designates them; CONFLICT vs BLOCKED | package §5.2, HQ-4 | PENDING |
| H-RV-07 | Legal | Privacy constraints: stored attributes, document copies, retention, lawful basis, lookup audit | package §5.3 P17, HQ-5 | PENDING |
| H-RV-08 | Security + Codex | Shared person-independence precondition for all security-sensitive unlock routes; D-42 gaps G1–G5 (continues RM-E04G RV-09b) | package §7.2 | PENDING |
| H-RV-09 | Codex + Security | R6 identity extension (F-PR1): fields, optional vs mandatory, strict validation | package §6.1, §9 | PENDING |
| H-RV-10 | Codex | Historical v1 records: no rewrite; appended attestation only | package §6.3 | PENDING |
| H-RV-11 | Security | Bootstrap controls for R5-SEC-01's own approval (independent-channel identity check by T5) | package §6.4, HQ-8 | PENDING |
| H-RV-12 | Security | T5 independence: verifier pool and how its identity is established before a resolver exists | package §8, HQ-12 | PENDING |
| H-RV-13 | Security + HR | Failure and revocation semantics: expired / revoked evidence, reassignment, merge, split | package §5.2, §5.3 | PENDING |
| H-RV-14 | Codex | Migration and rollback: shadow-only projection, no rewrite of `organization.person`, interim guard replacement conditions | package §6.5, §9 | PENDING |

## Not claimed

- No Codex, Security, HR or Legal review, no human verification and no A3 post-merge sign-off exists for this change.
- R5-SEC-01 stays `READY_FOR_REVIEW`; no option is selected; no signer, recorder or T5 verifier is appointed.
- No person identity is resolved; no account is linked; the nominee stays NOMINATED with
  `identity_resolution = NOT_STARTED`.
- Group B gate closed; S4 manifest empty; RM-E04G-I1 unchanged; no approval record exists; no grant is active.
