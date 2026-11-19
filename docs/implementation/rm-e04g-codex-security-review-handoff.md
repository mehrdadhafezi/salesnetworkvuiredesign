# RM-E04G — Codex / Security review handoff (F1 person-identity independence)

**DRAFT — NOT APPROVED.** Independent review request for
`docs/security/rm-e04g-f1-person-identity-independence.md` and `docs/governance/rm_e04g_test.go`. No review has been
performed or recorded. Only the reviewers may change a question's status. **Production: NO-GO.** Group B and S4 stay
BLOCKED.

| | |
|---|---|
| Status | **REVIEW PENDING.** |
| Requested reviewers | Codex (workflow contract, R6 compatibility, architecture); Security owner (identity, SoD, threat model) |
| Baseline | `main` @ `162ff8981c3331598616cdc38cb122dc9ab17eb9` |
| Scope | Governance tests and documentation only. No approval record, person id, account link, decision status, gate, grant, schema or runtime change |

Every row below is **PENDING**.

| # | Decision requested | Options / proposal | Reference | Status |
|---|---|---|---|---|
| RV-01 | R5-SEC-01 identity source of truth | R5-SEC-01 options A / B / C / D are unchanged; this PR picks none. Confirm the evaluator contract `resolve(account) → VERIFIED person_id / UNKNOWN / CONFLICT` fits each | §5, §6 | PENDING |
| RV-02 | Interim approval-readiness rule | Accept RM-E04G-I1 (Group B needs verified person independence; repository resolver resolves nothing) as the interim control, or replace it | §7 | PENDING |
| RV-03 | Final `person_id`-based independence model | Rules S1–S12 and precedence CONFLICT > UNKNOWN > VIOLATED; keep R6 rule I7 (recorder may be an approver)? | §5 | PENDING |
| RV-04 | Optional versus mandatory R6 identity fields | Optional (RM-E04F §11.2) vs mandatory for decisions that feed a gate | §6, §11 | PENDING |
| RV-05 | Legacy record compatibility | Zero records today; future v1 records valid for R6 but UNKNOWN for gates — acceptable, or require migration review per record? | §8 | PENDING |
| RV-06 | Multiple accounts and `SHARED_USER` | Several accounts → one person; `SHARED_USER` → CONFLICT; `USER_NOT_*` → UNKNOWN | §4.3, §5 | PENDING |
| RV-07 | Delegation independence | Delegate and delegator both reserved; self-delegation refused; delegation never adds independence | §5 S8 | PENDING |
| RV-08 | Human T5 verification and evidence | Verifier must be a resolved person distinct from signers, principals and recorder; evidence form for T5 | §5 S4r, S5 | PENDING |
| RV-09a | Group B gate integration | Interim rule as a separate readiness function on top of `rmE04CGateState` (existing gate test expectations unchanged) | §7.1 | PENDING |
| RV-09b | Other verified-decision routes | Should RT-3 (D-42 `APPROVED_VERIFIED` unlocks `identity.role_grant` migrations) and RT-4 (follow-up eligibility) also require person independence? Not changed here | §4.2 | PENDING |
| RV-10 | Rollback and migration guarantees | Interim rule relaxed only by replacing two named functions in a reviewed PR; F1-I guard relaxed only with an approved schema; no historical rewrite | §7, §8 | PENDING |
| RV-11 | F1-I unknown-field guard | Accept refusing any non-R6-v1 key (except `_comment`) in committed approval records | §7.3 | PENDING |

## Not claimed

- No Codex review, Security review, human verification or A3 post-merge sign-off exists for this change.
- No person identity is resolved; the nominee of RM-E04F stays NOMINATED with `identity_resolution = NOT_STARTED`.
- R5-SEC-01 stays `READY_FOR_REVIEW`; Group B gate closed; S4 manifest empty; no approval record exists; no grant is
  active.
