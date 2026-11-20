# RM-E04E — M2 Record 1 decision form: Seller capability eligibility

**DRAFT — NOT APPROVED.** Candidate form for registration. It is not an R5 inventory row, not an R6 tracker
entry and not an approval record.

| | |
|---|---|
| Baseline | `main` @ `238645206040b2e4a715df3855480890b01d2dd6` (PR #77 merged) |
| Official decision ID | **None.** "M2 Record 1" is a working label. Only the authorized inventory owner assigns an ID (registration packet §3) |
| Selection status | **None.** No box is ticked. The engineering recommendation in §9 is not a selection |
| Authority status | PROPOSED — not registered, not reviewed, not approved |
| Production | **NO-GO** |

Companions: row-scope form [`rm-e04e-m2-record-2-seller-row-scope-form.md`](rm-e04e-m2-record-2-seller-row-scope-form.md);
authority and registration packet [`rm-e04e-authority-and-registration-packet.md`](rm-e04e-authority-and-registration-packet.md);
evidence V1–V15 in [`../../implementation/rm-e04d-codex-security-review-package.md`](../../implementation/rm-e04d-codex-security-review-package.md) §1.

---

## 1. Decision question

Which target capabilities may the `seller` role bundle hold in the first Seller release?

This record decides **what** the Seller may do. It does not decide:
- **who** may grant it (D-42);
- how accounts are created (D-34);
- person-level SoD (R5-SEC-01);
- **which rows** "own" means (M2 Record 2).

Approving this record grants nothing to anyone. Real grants also need D-42, D-34 and a separate activation
authorization (Group C).

## 2. Terminology: options here vs historical M2 options

The M2 intake (`docs/remediation/m2-authorization-decision-intake-fa.md` §4-1, PR #50) was written with the
organization-wide capability `cases.case.read`. XR §2 restricts Seller reads to own data. RM-E04C/E04D therefore
replaced that capability with own-scoped capabilities. The letters below are **RM-E04E letters**. They do not mean
the same thing as the M2 letters.

| RM-E04E option | Capabilities | Nearest historical M2 §4-1 option | Difference |
|---|---|---|---|
| **A** — own-scoped reads | `cases.own.read`, `billing.own.read`, `payments.own.read`, `commission.own.read` | none (new model) | Four own-scoped reads replace any organization read |
| **B** — own-scoped case reads only | `cases.own.read` | M2 "ب" (`cases.case.read` alone) | Same screen coverage as M2 "ب", but own-scoped instead of organization-wide |
| **C** — own reads + contact-outcome recording | Option A + `cases.contact_outcome.record` | M2 "الف" (`cases.case.read` + `cases.contact_outcome.record`) | Adds a write. Read part is own-scoped instead of organization-wide |
| **D** — defer | none | M2 "ج" (none for now) | Same as M2 "ج". The formal form of D is an R6 `DEFERRAL` record, not a `DECISION` |

**Excluded from every option** (XR §2 forbids organization-wide Seller reads; changing that needs the XR
governance process, not this record): `cases.case.read`, `billing.invoice.read`, `payments.attempt.read`,
`custody.custody.read`, `sales.report.read`, `customers.customer.read`, `reporting.metrics.read`.
Also excluded: `payments.submission.read` and `payments.batch.submit` (OPD-10 / D-43 scope).

## 3. Options (none selected)

| ☐ | Option | Business value | Main risk |
|---|---|---|---|
| ☐ | **A** — `cases.own.read`, `billing.own.read`, `payments.own.read`, `commission.own.read` | Seller sees own cases, own invoices (once S4 is unblocked), own payments and own commission. Covers every screen that has a server source | Three new capability names (governance + code change). S4 part stays blocked by Record 2 / I-L prerequisites |
| ☐ | **B** — `cases.own.read` | Smallest surface: S1 and S2 attribution only | Seller cannot see invoices or commission. Low operational value |
| ☐ | **C** — A + `cases.contact_outcome.record` | Complete first workflow (see and act) | Adds a write: write flag, audit, D-42 / D-34 for real grants; possibly R5-SEC-01 if person-level SoD is later required |
| ☐ | **D** — defer | No risk | No value; Seller panel stays without a server source |

## 4. Capabilities: registered today vs proposed

| Capability | Kind | In code today? | In the `seller` bundle today? | A | B | C | D | Feature flag (existing) |
|---|---|---|---|---|---|---|---|---|
| `cases.own.read` | own read | **No** (proposed name; Codex naming review) | No | yes | yes | yes | — | `sales.read_shadow` |
| `billing.own.read` | own read | **No** (proposed) | No | yes | — | yes | — | `billing_payments.read_shadow` |
| `payments.own.read` | own read (invoice sub-resource) | **No** (proposed) | No | yes | — | yes | — | `billing_payments.read_shadow` |
| `commission.own.read` | own read | **Yes** (`CapCommissionOwnRead`, `internal/identity/api/api.go:101`) | No | yes | — | yes | — | `commission.read_shadow` |
| `cases.contact_outcome.record` | **write** | **Yes** (`CapContactOutcomeRecord`, `api.go:79`) | No | — | — | yes | — | `cases.nonfinancial_write` (+ `cases.nonfinancial_writeback` for legacy write-back) |

Today the `seller` bundle is empty (`{RoleSeller, "Seller", nil}`, `api.go:212`, V1). The database refuses
non-SHADOW grants (V2). The three unregistered names are guarded by `TestRME04CProposalsStayOutOfRuntime`: they
cannot appear in code while the Group B gate is closed.

## 5. Impact on Seller screens S1–S7

| Screen | Needs | A | B | C | D |
|---|---|---|---|---|---|
| S1 My numbers (cases) | `cases.own.read` | read | read | read + record outcome | none |
| S2 Ready to convert | `cases.own.read` (attribution sub-resource only); the list itself has no server source and no workflow contract | attribution only | attribution only | attribution only | none |
| S3 Re-engage | no server source | — | — | — | — |
| S4 Invoices + payments | `billing.own.read`, `payments.own.read` | granted on paper, **still blocked** (Record 2 §6) | — | as A | none |
| S5 Customer behaviour | no server source | — | — | — | — |
| S6 Number request | no workflow (OPD-10) | — | — | — | — |
| S7 Wallet (commission) | `commission.own.read` | read | — | read | none |

## 6. Read / write boundaries

- A and B contain **no write**. They need no write flag and no write audit.
- C adds exactly one write. It keeps the existing Slice 11 row control: only the current holder (`CurrentHolder`,
  V9) may record, unknown holder → `BLOCKED`, anyone else → `UNAUTHORIZED`. Read scope (Record 2, P2) equals this
  write scope only if Record 2 selects P2 for S1.
- View ≠ Write: approving a read never implies a write. Approving C does not approve any other write
  (`payments.batch.submit`, custody, customers).

## 7. Multi-role callers

- Each own capability serves only its own `/me/...` endpoint with scope `OWN`. Organization endpoints keep
  `ORGANIZATION`.
- An organization capability from another role never widens a `/me` result, and an own capability never narrows
  or widens an organization endpoint. The two are never unioned inside one response.
- A user who is both seller and supervisor sees two separate surfaces with separate checks.
- Hierarchy ≠ Permission: a manager's position grants nothing on `/me`.

## 8. Dependency and rollback matrix

| Item | Depends on | Needed for | Rollback |
|---|---|---|---|
| Record 1 approval | Record 2 approval (a capability without its predicate has no meaning); ADR acceptance (B1) | Group B gate | New R6 version (T7) or supersession; history stays |
| Own capability names in code (B-PR2) | Gate open with this record's exact list | B-PR3 | Revert code + catalogue down-migration; no grant data exists |
| `AccessDecision.scope` gains `OWN` | B-PR1 + Codex contract review | B-PR2 | Revert regenerated Go/TS; enum consumers re-checked |
| Real grant to a person | D-42, D-34, Group C authorization | Activation | Revoke grant (procedure from D-42 / E-7) |
| Option C write | `cases.nonfinancial_write` flag, audit, D-42 / D-34 | Contact outcome from Seller UI | Turn flag off. Recorded outcomes are append-only and stay |
| S4 part of A / C | Record 2 §6 blockers (conversion-payment contract, schema, tests, execution evidence) | S4 wiring | S4 stays out of the gate; nothing to roll back |

## 9. Engineering recommendation (not a selection)

**A**, subject to human approval:
- the smallest set that covers every Seller screen with a server source (S1, S2 attribution, S4, S7);
- no write, so no dependency on the write flag or its audit;
- consistent with XR §2;
- contact-outcome recording (C) fits better in a later version of this record, after D-42 and D-34.

This recommendation does not bind anyone. It is not entered in any authoritative field.

## 10. Required reviewers and approvers

| Role | Basis | Status |
|---|---|---|
| Product owner | M2 §4-1 / §6 proposal; Group B gate floor (`rmE04CGateFloorRoles`) | Holder **UNASSIGNED** |
| Sales owner | same | Holder **UNASSIGNED** |
| Security owner | same | Holder **UNASSIGNED** |
| Codex (architecture review of names and scope model) | AGENTS.md *Agent Ownership* | **CODEX_REVIEW_PENDING** |
| Recorder | R6 T4 (any role) | UNASSIGNED |
| T5 verifier | R6 T5: not an approver, not the recorder, not automation | UNASSIGNED |

## 11. Missing before a decision

- Official ID and R5 / R6 registration (packet §3).
- Authority evidence for each signer (packet §1).
- Codex review of the capability names (`*.own.read`) and of rewording the M2 text from `cases.case.read`.
- Record 2 decided in the same session (no capability without a predicate).

## 12. Proposed decision wording (fill only the chosen option; never pre-filled)

> For the first Seller release, the `seller` role is eligible for exactly these capabilities: ____ (option ____).
> Each is read only within the row scope approved in decision ____ (M2 Record 2). No organization-wide read
> capability is eligible. Eligibility grants nothing to any person. Real grants follow D-42 and D-34, and
> activation needs separate authorization. Conditions: ____.

**Outcome fields (empty until a real, verified decision exists):**

| Field | Value |
|---|---|
| Selected option |  |
| Full decision text |  |
| Approvers (name, role, authority evidence) |  |
| Decision date |  |
| Effective date |  |
| Conditions |  |
| Evidence reference / immutable_ref |  |
| Human verification |  |

## 13. RM-E04L owner preference (non-authoritative annotation)

**OWNER PREFERENCE — NOT APPROVED.** Recorded in RM-E04L (baseline `main` @ `a8f3c4df9215f4496f2547e89d950403f485ead3`)
from a conversation-level instruction of the project owner. It is not a selection in §3, not an outcome in §12, not
an R6 record and not implementation authorization. Report label: `M2_REC_1 = OWNER_PREFERRED_READ_ONLY_OWN_CASES`;
governance state: **`NOT_APPROVED_VERIFIED`** (record not registered; no official ID). Production **NO-GO**.

| Layer | Value |
|---|---|
| Owner preference | Seller reads **own cases only**, read-only, first phase |
| Nearest option in §3 | **B** (`cases.own.read` alone). `cases.own.read` is a **proposed candidate name**, not a registered capability (§4; Codex naming review pending) |
| Engineering recommendation (§9) | A — unchanged and still non-binding. The preference is narrower than A; the difference is a human decision |
| Official decision / approvers / T5 | **None** |

What the preference does **not** cover (each needs its own explicit eligibility decision; none is implied):

| Domain | Capability in the original options | Status under the preference |
|---|---|---|
| Invoices (S4) | `billing.own.read` | **DEFERRED** — also blocked by Record 2 §6 S4 readiness |
| Payments (S4 sub-resource) | `payments.own.read` | **DEFERRED** |
| Commission / wallet (S7) | `commission.own.read` | **DEFERRED** — the capability exists in code but is not in the `seller` bundle; S7 semantics stay blocked on D-41 / OPD-09 |
| Contact-outcome recording | `cases.contact_outcome.record` (write) | **EXCLUDED** — View ≠ Write (§6) |
| Case assignment, invoice issuing, any other write | — | **EXCLUDED** |
| Attribution sub-resource (S2) | `cases.own.read` (`listMyCaseAttribution`) | **REQUIRES EXPLICIT SCOPE** — it exposes original-owner / credit facts, which are not "current custody" (Record 2 §12) |
| Organization-wide reads | §2 exclusion list | **EXCLUDED** (XR §2) |

Options A–D in §3 stay open and unticked. Approvers may select any of them; the consequences in §3–§8 are unchanged.
Handoff and dependency chain: [`../../implementation/rm-e04l-seller-own-case-preference-handoff.md`](../../implementation/rm-e04l-seller-own-case-preference-handoff.md).
