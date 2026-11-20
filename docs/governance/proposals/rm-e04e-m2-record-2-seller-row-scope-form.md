# RM-E04E — M2 Record 2 decision form: Seller row-scope (ownership predicate)

**DRAFT — NOT APPROVED.** Candidate form for registration. It is not an R5 inventory row, not an R6 tracker
entry and not an approval record.

| | |
|---|---|
| Baseline | `main` @ `238645206040b2e4a715df3855480890b01d2dd6` (PR #77 merged) |
| Official decision ID | **None.** "M2 Record 2" is a working label. Only the authorized inventory owner assigns an ID |
| Selection status | **None.** No box is ticked. Recommendations (§5) are engineering advice, not selections |
| Authority status | PROPOSED — not registered, not reviewed, not approved |
| Production | **NO-GO** |

Companions: capability form [`rm-e04e-m2-record-1-seller-capability-form.md`](rm-e04e-m2-record-1-seller-capability-form.md);
packet [`rm-e04e-authority-and-registration-packet.md`](rm-e04e-authority-and-registration-packet.md);
evidence V1–V15 and predicates in [`../../implementation/rm-e04d-codex-security-review-package.md`](../../implementation/rm-e04d-codex-security-review-package.md) §1, §3.

---

## 1. Decision question

For each Seller destination with a server source (S1, S2 attribution, S4, S7), which **ownership fact** defines
"my rows"?

Reference rule: XR §2. Seller reads are own-only. Organization-wide Seller reads are not an option in this record.

Ownership facets are never collapsed (AGENTS.md): Current Custody, Original Owner, Next Actor, Event Actor and
Credit Owner are different facts. This record picks one fact per destination. It never derives ownership from
hierarchy, UI role, phone or customer information.

## 2. Terminology: predicate codes vs historical M2 options

The M2 intake (§4-2) numbered its options 1–5. RM-E04D found that RM-E04A/E04C attached the label "M2 option 1"
to the wrong column. This form therefore names predicates **by column**:

| Code | Historical M2 §4-2 option | Meaning |
|---|---|---|
| P1 | option 1 "current holder" (derived) | Holder derived by the custody engine from events |
| **P2** | option 2 "current assignment" (legacy column) **plus** the existing contradiction guard | Legacy holder column, the same as `CurrentHolder` (V9) |
| P3 | none (stricter than 1 and 2) | P1 and P2 agree |
| — | option 3 (current + historical) | Not proposed (privacy) |
| — | option 4 (organization) | Excluded by XR §2 |
| — | option 5 (later) | Equivalent to deferring the record (R6 `DEFERRAL`) |
| I-a … I-L | none (S4 was not in the M2 intake) | Invoice predicates (§3) |
| C | none | Commission entitlement holder (exists) |

## 3. Predicate catalogue (none selected)

All SQL runs inside the scoped query, never as a post-filter. `:me` is the authenticated `Subject.UserID`. A
caller without a proven user id gets 403 `ROW_SCOPE_DENIED` (V8 precedent).

| Code | Exact predicate | Tables / relationship | Screens |
|---|---|---|---|
| P1 | `derivation = 'DERIVED' AND derived_user_id = :me AND presence = 'PRESENT'` | `custody.assignment` per case | S1 |
| P2 | `legacy_holder_user_id = :me AND presence = 'PRESENT' AND derivation <> 'MULTIPLE' AND parity NOT IN ('CONFLICT','MULTIPLE_CURRENT')` | `custody.assignment` per case; legacy source: lead cases ← `sn_leads.seller_id`, distribution-item cases ← `current_owner_user_id` (V10) | S1, S2 attribution |
| P3 | P1 AND P2 AND `parity = 'MATCH'` | same | S1 |
| I-a | `seller_user_id = :me` | `billing` invoice | S4 |
| I-b | `issued_by_user_id = :me` | `billing` invoice | S4 |
| I-c | `seller_user_id = :me AND issued_by_user_id = :me` | `billing` invoice | S4 (fail-closed interim) |
| I-d | linked case passes P2 AND `case_link_state = 'RESOLVED'` | invoice → case link → `custody.assignment` | S4 |
| I-L | `seller_user_id = :me AND (conversion_payment IS FALSE OR issued_by_user_id = :me)`; NULL / unknown conversion state is treated as conversion | invoice + conversion-payment fact (**not imported today**, V11) | S4 |
| C (`COMMISSION_HOLDER`) | `holder_user_id = :me` | `commission.historical_entitlement` | S7 |

## 4. Options per destination (none selected)

| Destination | ☐ Options | Not proposed / excluded |
|---|---|---|
| S1 My numbers | ☐ P1 · ☐ P2 · ☐ P3 · ☐ defer | historical assignment; original seller; organization |
| S2 Ready to convert (attribution sub-resource) | ☐ inherit the approved S1 predicate · ☐ defer | any own predicate for the S2 **list** (no source, no workflow contract) |
| S4 Invoices + payments | ☐ I-a · ☐ I-b · ☐ I-c · ☐ I-d · ☐ I-L · ☐ defer | entitlement holder (does not admit invoices) |
| S7 Wallet | ☐ C (`COMMISSION_HOLDER`) · ☐ defer | — |

## 5. Engineering recommendation (not a selection)

| Destination | Recommended | Status of the recommendation |
|---|---|---|
| S1 | **P2** (legacy holder + existing contradiction guard) | Recommended, **not approved**. Implementable once Group B opens |
| S2 | **Inherit the approved S1 predicate** for the attribution sub-resource. The conversions **list** stays **BLOCKED** (no workflow contract) | Recommended, not approved |
| S4 | **I-L** (legacy-equivalent) | Recommended, **not approved and not implementable today**. See §6 |
| S7 | **C** (`COMMISSION_HOLDER`), with the financial-semantics limits in §7.7 | Recommended, not approved. Server scope already exists (V8) |

Why P2 for S1: it is the exact guard of the contact-outcome write (V9), so "what I see" = "what I may act on"; for
lead cases it equals legacy 2.0.160 `WHERE l.seller_id = me` (V12); it excludes contradicted cases.

Why I-L for S4: legacy 2.0.160 shows invoices by seller and requires issuer = me only for conversion-payment
invoices, "preventing another converter's payment links from leaking to the seller" (V13). I-a leaks those; I-c
hides almost everything (synthetic 0 of 4).

## 6. S4 stays BLOCKED even if Record 2 selects I-L

S4 may not enter the Group B gate until **all** of these exist (`docs/governance/rm-e04d-s4-readiness.json`,
`rm_e04d_test.go`):

| # | Prerequisite | Today |
|---|---|---|
| 1 | Conversion-payment data contract registered and `APPROVED_VERIFIED`, signed at least by Finance owner and Security owner | Not registered |
| 2 | Schema facts: named billing migration, column `NOT NULL`, `CHECK` with an explicit unknown state | Absent (V11) |
| 3a | Static registration of six named I-L tests | Empty |
| 3b | Real CI execution on the implementing commit + mutation evidence (review package §7, E1–E3) | **PENDING by construction**; no mechanism exists |
| 4 | Independent Security review recorded as R6 `DECISION` + T5 verification (E4), then the governance-test change (E5) | Not possible yet |

Approving I-L in this record answers the **policy** question only. It does not satisfy any row above.

## 7. Behaviour of each recommended predicate

### 7.1 Reassignment

| Predicate | Behaviour |
|---|---|
| P2 (S1, S2) | After the import batch that moves the case, the previous seller loses it on the next request. The new holder gains it. Attribution history is never rewritten. P2 is batch-dependent (stale between imports); each response states the evaluation time; no scope cache |
| I-L (S4) | Bound to the invoice, not to the case. A seller keeps seeing invoices where they are seller (and issuer for conversion payments) after the case moves. Product question PQ-3 |
| C (S7) | Bound to the entitlement. Reassigning a case does not move historical entitlements |

### 7.2 Conflicts

P2 excludes `derivation = 'MULTIPLE'` and `parity IN ('CONFLICT','MULTIPLE_CURRENT')`. Excluded cases are visible to
no seller and need the management path. I-L and C have no multi-owner state today. A future conflict state must be
excluded the same way (SR-05).

### 7.3 NULL / UNKNOWN

Rows whose ownership fact is NULL, UNKNOWN, CONFLICT, MULTIPLE or NOT_RECOVERABLE are **not shown and not counted**.
The number of hidden rows is not reported. For I-L, an unknown conversion state counts as conversion, so the
issuer condition applies (fail closed). Inside a visible row, unknown business facts render as UNKNOWN, never zero.

### 7.4 Search and pagination

- Search runs inside the predicate. Case or invoice code: exact or prefix of at least 4 characters.
- Phone search only after GAP-04, exact match only. Phone is never a key or ownership proof.
- Cursor pagination with a sort name per `/me` endpoint. The predicate is re-applied on every page.

### 7.5 Counts and totals

Computed by the same scoped query. Not computable → "—", never 0. Totals are per currency unit (IRT / IRR), never
summed across units. Incomplete data → no total shown.

### 7.6 Sub-resources and 404 parity

- Every sub-resource (case history, contact outcomes, attribution; invoice stages, totals, payments) re-applies
  the parent predicate in its own query.
- An out-of-scope row is indistinguishable from a missing row: same status, same problem body (A3 precedent, V6).
- An S7 entitlement that references an invoice does not admit that invoice into S4; following the link must pass
  the S4 predicate.

### 7.7 S7 financial-semantics limits (carried, not decided here)

`COMMISSION_HOLDER` defines **which rows** are mine. It does not define what they mean financially. Until D-41 /
OPD-09 / D-36 are decided:
- lines are shown as historical credit (shadow), not as settleable;
- no "confirmed", "withdrawable" or settlement value is shown;
- units are never summed together;
- whether the credit owner for the wallet equals the entitlement holder is an OPD-09 question.

### 7.8 Multi-role callers

Own predicates apply only on `/me` endpoints with scope `OWN`. An organization capability held through another
role never widens `/me` results, and the two scopes are never merged in one response.

### 7.9 Privacy and staff identity

- Other staff: only "you / someone else" (`is_caller`). No other staff id or name until RM-E04A-Q4 is decided.
- Customer: `customer_id` only until GAP-04. Name and phone later through a separate audited, fail-closed
  sub-resource (D-45).
- Legal (privacy) sign-off on this record is an open question (PQ-4; packet §1).

## 8. Unresolved product questions (visible; not decided here)

| # | Question | Why it matters |
|---|---|---|
| PQ-1 | Are distribution-item cases (`current_owner_user_id`) part of Seller S1, or only lead cases? | P2 uses one column for both kinds |
| PQ-2 | Should a conversion-payment invoice that the seller did not issue appear in their S4? (legacy: no) | I-a vs I-L |
| PQ-3 | After reassignment, does the previous seller still see their own invoices? (I-L: yes) | User expectation |
| PQ-4 | Must Legal (privacy) sign Record 2? | It sets access to personal data |
| PQ-5 | Is field scope (e.g. masked vs full phone) part of this record or a separate one? | Server control type (M2 Q9) |
| PQ-6 | Should the Seller see contact outcomes they recorded after the case moved away? | P2 hides them |

## 9. Dependencies

| Depends on | Why |
|---|---|
| Record 1 | Predicates apply to the capabilities Record 1 approves |
| GAP-04 | Customer fields and phone search |
| OPD-10 | Issuance assistance / delegation semantics for S4 |
| Conversion-payment data contract (new record, not registered) | Required for I-L implementation |
| ADR #18 | Real-data rates (seller ≠ issuer, unknown holders) cannot be measured without it |
| D-41 / OPD-09 / D-36 | S7 meaning, not S7 scope |

## 10. Required reviewers and approvers

| Role | Basis | Status |
|---|---|---|
| Product owner | M2 §4-2 / §6; gate floor | Holder **UNASSIGNED** |
| Sales owner | same | Holder **UNASSIGNED** |
| Security owner | same | Holder **UNASSIGNED** |
| Legal owner (privacy) | **Open** (PQ-4, M2 Q8) | Requirement undetermined; holder UNASSIGNED |
| Codex (scope model, SQL predicates, data contract) | AGENTS.md | **CODEX_REVIEW_PENDING** |
| Finance owner | Conversion-payment contract (S4 prerequisite), not Record 2 itself | UNASSIGNED |
| Recorder / T5 verifier | R6 | UNASSIGNED |

## 11. Proposed decision wording (fill only the chosen predicates; never pre-filled)

> Seller own-scope is defined per destination as follows: S1 ____; S2 attribution ____ (the S2 list remains
> without scope until a workflow contract exists); S4 ____; S7 ____. Rows whose ownership fact is NULL, UNKNOWN,
> CONFLICT, MULTIPLE or NOT_RECOVERABLE are neither shown nor counted. Out-of-scope rows are indistinguishable from
> missing rows. Ownership is never inferred from hierarchy, UI role, phone or customer data. Approval of the S4
> predicate does not authorize S4 implementation before the S4 readiness proofs exist. Answers to PQ-1…PQ-6:
> ____. Conditions: ____.

**Outcome fields (empty until a real, verified decision exists):**

| Field | Value |
|---|---|
| Selected predicates |  |
| Full decision text |  |
| Approvers (name, role, authority evidence) |  |
| Decision date |  |
| Effective date |  |
| Conditions |  |
| Evidence reference / immutable_ref |  |
| Human verification |  |

## 12. RM-E04L owner preference (non-authoritative annotation)

**OWNER PREFERENCE — NOT APPROVED.** Recorded in RM-E04L (baseline `main` @ `a8f3c4df9215f4496f2547e89d950403f485ead3`)
from a conversation-level instruction of the project owner. It ticks nothing in §4, fills nothing in §11 and opens no
gate. Report label: `M2_REC_2 = OWNER_PREFERRED_CUSTODY_CURRENT_HOLDER`; governance state: **`NOT_APPROVED_VERIFIED`**
(record not registered; no official ID). Production **NO-GO**.

**Preference as stated:** Seller-owned cases are those where the caller is the verified current custody holder of the
requested case, as established by authoritative server-side custody facts (`CUSTODY_CURRENT_HOLDER`, the basis name
used by the API proposal).

**Normalization against §3.** `CUSTODY_CURRENT_HOLDER` is a basis *label*, not a column. It can be realised by P1
(derived), P2 (legacy holder + contradiction guard) or P3 (both agree). The only resolver that exists today is
`CurrentHolder` (`internal/custody/query/query.go`), which is **P2** and fails closed with a named basis
(`NO_CUSTODY_RECORD`, `CASE_ABSENT_IN_LATEST`, `NO_RESOLVED_LEGACY_HOLDER`, `CONTRADICTING_HOLDER_EVIDENCE`). The
engineering mapping is therefore **preference ≈ P2 for S1**, but which fact is "authoritative" is an approver
decision: P1 and P3 stay open alternatives with their §3 consequences. S2 attribution, S4 and S7 are **not covered**
by this preference.

**Predicate completeness check (fail closed wherever UNRESOLVED):**

| Case | Defined by | State |
|---|---|---|
| Unknown / NULL holder | §7.3, SR-05; `CurrentHolder` `Known=false` | Defined — not shown, not counted |
| Historical vs current holder | §1, §7.1; historical option not proposed | Defined — current only. PQ-6 (outcomes the Seller recorded after the case moved) **open** |
| Pending transfer | — | **UNRESOLVED.** The target custody model has no transfer / pending state (`internal/custody/custody.go`: no Transfer); legacy in-flight semantics are unverified. Until defined, only the published holder counts and nothing else widens scope |
| Conflicting custody evidence | §7.2; `CONTRADICTING_HOLDER_EVIDENCE` | Defined — excluded for every Seller |
| Missing caller identity | §3 (V8 precedent) | Defined — 403 `ROW_SCOPE_DENIED`. Scope is per account, not per person (R5-SEC-01 stays separate) |
| Reassignment / revoked custody | §7.1; SR-11 | Defined — lost on the next request after the moving import batch; stale between batches. The **acceptable staleness window is not decided** |
| Cross-user access | §7.6, SR-06 | Defined — out-of-scope ≡ missing |
| Cross-organization access | — | **UNRESOLVED.** Case and custody tables carry no organization key; isolation is single-deployment only. Approvers must state this as a condition or require a model |
| Aggregate counts | §7.5, SR-07 | Defined — same scoped query; not computable → ABSENT, never 0 |
| Detail and history sub-resources | §7.6, SR-06 | Defined — each re-applies the parent predicate. Attribution is S2 (above) |
| Generic 404 parity | §7.6; A3 precedent | Defined — same status and body as a missing id |

PQ-1 (distribution-item cases in S1), PQ-4 (Legal sign-off) and PQ-6 remain open and must be answered in the decision
text. Handoff: [`../../implementation/rm-e04l-seller-own-case-preference-handoff.md`](../../implementation/rm-e04l-seller-own-case-preference-handoff.md).
