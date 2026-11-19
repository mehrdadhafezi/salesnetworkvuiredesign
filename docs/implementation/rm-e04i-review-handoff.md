# RM-E04I — Codex / Security / HR / Legal / Product review handoff (Option A readiness)

**DRAFT — NOT APPROVED.** Independent review request for
`docs/security/rm-e04i-option-a-implementation-contract.md`,
`docs/governance/proposals/rm-e04i-option-a-preference.NON-AUTHORITATIVE.json`,
`docs/governance/proposals/rm-e04i-dependency-graph.json` and `docs/governance/rm_e04i_test.go`.
No review has been performed or recorded. Only the named reviewers may change a row's status, and only by recording
their own result. **Production: NO-GO.** Group B and S4 stay BLOCKED. Real grants stay BLOCKED.

| | |
|---|---|
| Status | **REVIEW PENDING.** |
| Requested reviewers | Codex (architecture, R6 schema, PR sequence); Security owner (resolver, SoD, D-42 unlock); HR owner (person source, proofing, workflow); Legal owner (privacy, retention); Product owner (scope, ordering) |
| Reviewer holders | All **UNASSIGNED**. Nominations are provisional (`NOMINATED` / `UNASSIGNED`); HR owner and T5 verifier have no nominee; CEO appointment authority needs documentary proof. A review by a nominee is not a review by a verified office holder; no candidate name substitutes for a signed appointment |
| Authority floor for R5-SEC-01 | Security, HR, Legal and Product each independently represented; T5 independent of every signer and the recorder; minimum five distinct humans (RM-E04H §8) |
| Baseline | `main` @ `739db621bdf834cd92d7b33ce9a31f224572cc22` |
| Scope | Documentation, a non-authoritative preference record, a non-authoritative dependency graph and governance tests. No approval record, person, link, status, gate, grant, credential, schema, migration or runtime change |

Every row below is **PENDING**.

| # | Reviewer | Question | Reference | Status |
|---|---|---|---|---|
| I-RV-01 | Codex + Product | Preference vs decision separation: is the record unconsumable and clearly non-authoritative? | contract §2; preference record | PENDING |
| I-RV-02 | Codex + HR | Reuse inventory and the single new anchor (VerifiedPerson); no rewrite of `organization.person` | contract §3, §4 | PENDING |
| I-RV-03 | Codex + Security | Entity contracts E1–E7: identifiers, immutability, constraints, ownership (IQ-1) | contract §4 | PENDING |
| I-RV-04 | Security | Resolver algorithm, precedence, purposes (IQ-3), races RC-1…RC-9 | contract §5 | PENDING |
| I-RV-05 | HR + Security | Workflow W1–W11, separation of duties, bootstrap (IQ-2) | contract §6 | PENDING |
| I-RV-06 | Codex + Security | R6 / F1 integration: F-PR1 fields, adapter, historical v1, delegation | contract §7 | PENDING |
| I-RV-07 | Security + Codex | D-42 hardening H-1…H-8, change list, tests ND-1…ND-11 (IQ-5) | contract §8 | PENDING |
| I-RV-08 | Codex + inventory owner | Dependency graph and the UNRESOLVED cycle OF-4a / OF-4b; recommended ordering (HQ-10) | contract §9; graph JSON | PENDING |
| I-RV-09 | Codex | PR sequence PR-A…PR-F, PR-B before PR-C (IQ-4), NO-GO conditions | contract §10 | PENDING |
| I-RV-10 | Legal | Privacy classification, document copies, retention, purpose-limited audited reads | contract §4, §5.2 | PENDING |
| I-RV-11 | Product | Scope of PR-F interface; capabilities for it are a separate decision | contract §10 | PENDING |
| I-RV-12 | Codex | Governance tests guard safety properties without freezing legitimate future approvals | contract §12 | PENDING |

## Not claimed

- No Codex, Security, HR, Legal or Product review and no human verification exists for this change.
- R5-SEC-01 stays `READY_FOR_REVIEW`; no option is selected; Option A is only the project owner's preference.
- No signer, recorder or T5 verifier is appointed; no signature exists; no T5 verification is recorded.
- No person identity is resolved; no account is linked; no runtime resolver exists.
- RM-E04G-I1 unchanged; Group B gate closed; S4 manifest empty; D-42 not unlocked; no grant is active.
