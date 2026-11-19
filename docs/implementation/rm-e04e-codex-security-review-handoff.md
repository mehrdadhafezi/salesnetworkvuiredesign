# RM-E04E — Codex and Security review handoff (Seller authorization)

**DRAFT — NOT APPROVED.** Review request only.

| | |
|---|---|
| Status | **CODEX_REVIEW_PENDING.** Security review: **PENDING.** Nothing in this file is a review result |
| Baseline | `main` @ `238645206040b2e4a715df3855480890b01d2dd6` (PR #77 merged) |
| Why pending | There is no authorized Codex review integration in this environment, and the earlier Codex attempt was blocked by usage limits. No result exists to report |
| Reviewers asked | Codex (architecture, workflow and data contracts, implementation and tests: AGENTS.md *Agent Ownership*); Security owner (holder UNASSIGNED, see packet §1) |
| Runtime change | none |
| Production | **NO-GO** |

How a review result becomes valid:
- **Codex architecture review:** recorded by Codex itself on the PR or in a committed review note that names the
  item, the verdict and the commit.
- **Security owner decisions** (S4 execution evidence, contract floor roles): recorded as R6 `DECISION` records
  with T5 verification.

Claude does not record, summarize or infer a verdict for either.

Prior package (still the technical basis, unchanged): [`rm-e04d-codex-security-review-package.md`](rm-e04d-codex-security-review-package.md).

---

## Review items

| # | Item | Question for the reviewer | Material | Blocks | Status |
|---|---|---|---|---|---|
| RV-1 | Own-scoped Seller architecture (`/me/...`, scope class `OWN` derived from capability, never from grant data) | Accept ADR Option C as written, or name the blocker | ADR `rm-e04c-seller-authorization-adr-PROPOSED.md`; review package §4 | B-PR1 (ADR acceptance) | CODEX_REVIEW_PENDING |
| RV-2 | M2 capability and row-scope model | Are `cases.own.read`, `billing.own.read`, `payments.own.read` the right names and granularity? Is one predicate per destination (form §3) implementable as a module-owned SQL fragment shared by list, count, search and sub-resources? | Record 1 form §4; Record 2 form §3, §7 | Registration wording; B-PR2 | CODEX_REVIEW_PENDING |
| RV-3 | I-L conversion-payment data contract | Approve the import of `sn_dot_invoice_links.flow_kind` and `sn_invoices.original_seller_id` into billing as facts with a link state and an explicit `UNKNOWN`; NOT NULL + CHECK; name the migration and column | V11, V13, V14; review package §4 last row; S4 manifest schema fields | B-PR4; S4 proof 1 and 2 | CODEX_REVIEW_PENDING; Finance + Security decision not registered |
| RV-4 | A3 payment 404 parity (PR #73, merged without the required pre-merge sign-off) | Does any downstream workflow contract rely on a non-owner receiving `ROW_SCOPE_DENIED` from the submission read? | review package §5 | Governance debt; does not block registration | **Post-merge sign-off owed — PENDING** |
| RV-5 | Scope enum and effective-access contract | Adding `OWN` to `AccessDecision.scope` (`[NONE, ORGANIZATION]`, V15): regenerated Go + TS, consumers, NT-07 rewrite and new NT-25 | review package §4 rows 3–4 | B-PR2 | CODEX_REVIEW_PENDING |
| RV-6 | S4 execution-evidence approval mechanism (E1–E5) | Approve the design: CI job running exactly the six I-L tests with `-json` artefact bound to SHA + run id; mutation evidence; Security `DECISION` + T5; then the governance-test change | review package §7; `TestRME04DS4ExecutionEvidencePending` | S4 proof 3b | CODEX_REVIEW_PENDING; Security PENDING |

## What each reviewer is asked to return

| Reviewer | Return |
|---|---|
| Codex | Per item: `ACCEPT`, `ACCEPT WITH CHANGES` (list), or `BLOCKER` (evidence). The commit SHA reviewed |
| Security owner | RV-3 and RV-6: whether the floor roles (Finance + Security) are sufficient; RV-2: whether the NULL / UNKNOWN / 404 rules in Record 2 §7 are acceptable |

## What is **not** being asked

- No reviewer is asked to approve M2 Records 1 or 2. That is a business decision of Product, Sales and Security
  (and possibly Legal).
- No reviewer can open the Group B gate. Only verified R6 records can.

## Effect of each outcome

| Outcome | Effect |
|---|---|
| RV-1 / RV-5 `BLOCKER` | Record 1 options stay valid as policy; the implementation route (B-PR1/B-PR2) is re-planned. A design blocker is recorded, not silently worked around |
| RV-2 change of names | Registration wording uses the reviewed names. Forms are updated before registration |
| RV-3 rejected | I-L stays recommended as policy but S4 stays blocked indefinitely; I-c remains the only fail-closed interim |
| RV-4 rejected | Revert of PR #73 is self-contained (no schema / data / flag) |
| RV-6 rejected | S4 stays blocked; `TestRME04DS4ExecutionEvidencePending` unchanged |
