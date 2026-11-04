# Slice 15 — Final cutover readiness assessment

| Item | Value |
|---|---|
| Type | Audit / evidence. **Not** a cutover authorisation. |
| Baseline | `origin/main` = `b3bf31b5979edfa1ba9a5fb6a45521d21e7712c4` (PR #39 merged) |
| Legacy behavioural baseline | `plugin/current` 2.0.160 (`SN_VERSION 2.0.160`). Unchanged since the sync at `86859d7`. No newer baseline is substituted. |
| **Overall recommendation** | **NO-GO for Production** ([go-no-go-report.md](go-no-go-report.md)) |

## Contents

| # | Deliverable |
|---|---|
| 1 | This assessment |
| 2 | [module-readiness-matrix.md](module-readiness-matrix.md) |
| 3 | [feature-flag-and-capability-audit.md](feature-flag-and-capability-audit.md) |
| 4 | [data-reconciliation-register.md](data-reconciliation-register.md) |
| 5 | [financial-readiness.md](financial-readiness.md) |
| 6 | [biawin-integration-readiness.md](biawin-integration-readiness.md) |
| 7 | [infrastructure-operations-checklist.md](infrastructure-operations-checklist.md) |
| 8 | [end-to-end-journey-matrix.md](end-to-end-journey-matrix.md) |
| 9 | [decision-approval-register.md](decision-approval-register.md) |
| 10 | [cutover-runbook-DRAFT.md](cutover-runbook-DRAFT.md) |
| 11 | [rollback-and-recovery-DRAFT.md](rollback-and-recovery-DRAFT.md) |
| 12 | [go-no-go-report.md](go-no-go-report.md) |
| — | Supporting script: `scripts/cutover/readiness-evidence.sh` (read-only; refuses unsafe targets) |

## 1. Baseline verification

All items are VERIFIED.

| Check | Result |
|---|---|
| PR #39 merged | Yes: merge commit `b3bf31b` on `origin/main`. |
| Slices 0–14 present | Merge commits for PRs #25 (Slice 0) to #39 (Slice 14) are all in `main` history. |
| Repository clean | Working tree clean at the start; branch `audit/slice-15-cutover-readiness` created from `b3bf31b`. |
| `plugin/current` unchanged | `git log 86859d7..b3bf31b -- plugin/current` is empty. |
| CI evidence | CI runs on `pull_request` only. The `b3bf31b` tree is byte-identical to `26f92b1`, whose three jobs (Go gates, frontend, OpenAPI) passed on PR #39. |

**Baseline gates re-run locally at `b3bf31b`** (local PostgreSQL 16.15, synthetic data; CI uses a PostgreSQL 18 service):

| Gate | Result |
|---|---|
| `go vet ./...` | Clean. |
| `go test -count=1 ./...` | 70 packages ok, 0 failures. Run with `CRM_TEST_REQUIRE_POSTGRES=1`. |
| CI `crm-migrate` block on a fresh database | All invariants pass; 51 migrations applied. |
| Web | Typecheck, lint and build OK; 113/113 tests. |
| `readiness-evidence.sh static` | 8 verified, 7 findings. |
| `readiness-evidence.sh db` (disposable local database) | 0 flags, 6 grants, 0 staged snapshots. |

**Documents read:**
- Target architecture V1, with §30–34 in full.
- Gate-0 invariants, Cross-Role V1 §19, E0 freeze.
- Slice 11–14 command matrices. Slice 9 and 10 matrices exist only in PR descriptions (REPORTED).

**Tooling found:** `crm-migrate`, `crm-legacy-import`, the offline shadow jobs, and the flag store.

**Tooling absent:**
- deployment
- backup
- observability beyond health and logs
- a rollout operator procedure

## 2. Executive summary

The target CRM is a well-guarded **shadow** system.
- Every flag is OFF.
- Every new capability is granted to no role.
- Every financial and Biawin effect is blocked by design, in code, schema and contract.
- The fail-closed behaviour is VERIFIED by tests.

None of that makes it ready to take authority from legacy:

1. **Nobody can sign in.** Production requests resolve to an anonymous subject.
2. **No background process exists.** Nothing runs receipt finalisation, upload expiry, outbox relay,
   legacy write-back or provider reconciliation. Controlled writes would therefore diverge from legacy.
3. **No real data has ever been reconciled.** 55 of 89 legacy tables are not even specified for
   extraction.
4. **Core revenue commands are not implemented.** Invoice issue/return, Finance approve/reject, posting,
   refunds and compensations are absent. They wait on open owner decisions.
5. **No infrastructure, backup, monitoring or DR definition exists** in the repository.

**Recommendation: NO-GO.** The safest next steps are engineering prerequisites (E-1…E-7) and the
management decision packages, followed by a read-only shadow pilot.

## 3. Historical and runtime compatibility — Gate-0 invariants

**"Target code" column:**
- **ENFORCED**: implemented and VERIFIED by tests on synthetic data.
- **PARTIAL**: some paths enforced.
- **NOT IMPLEMENTED**: the operation does not exist in the target. The invariant is preserved only
  because no target write exists; legacy behaviour is unchanged.

**Production status: UNVERIFIED for all 36 invariants.** No real data and no real accounts.

| INV | Name | Target code | Required proof for production | Severity | Modules | Blocking scope | Decision owner |
|---|---|---|---|---|---|---|---|
| 001 | Stable case identity | ENFORCED (CaseRef independent of phone) | real-snapshot case mapping | HIGH | cases | P1 pilot | Engineering |
| 002 | Proven alias resolution | ENFORCED (no merges; RELATED/CONFLICT) | resolution report on real data | HIGH | customers | P1 | MIS + Sales (D-30) |
| 003 | Source provenance | PARTIAL (MIS/Dot/campaign sources unspecified) | specs for the 55 tables | MEDIUM | cases, custody | P1 | Engineering |
| 004 | Four owners | ENFORCED (separate custody/original/next/credit; Slice 13 credit owner fix) | real-data attribution parity | HIGH | cases, commission | P1/P6 | Sales + Finance |
| 005 | Traceable handoff | PARTIAL (no ACK model; OPD-02) | OPD-02 decision | MEDIUM | cases, HR | P2 | Product |
| 006 | All financial links | NOT IMPLEMENTED (no return) | F01 proof | HIGH | custody | P5 | F01 owners |
| 007 | Consumption guard | NOT IMPLEMENTED | OPD-03 | HIGH | custody | P5 | Sales/MIS/Finance |
| 008 | Apply recheck | ENFORCED for target commands (row lock + version recheck) | real-account tests | HIGH | all writes | P2 | Engineering |
| 009 | Return/invoice mutual safety | NOT IMPLEMENTED | ADR-01 | HIGH | cases, custody, billing | P5 | Human Owner + Eng |
| 010 | History conservation | ENFORCED (append-only triggers per module, CI) | — | HIGH | all | — | — |
| 011 | Separate financial objects | ENFORCED (invoice/stage/attempt/receipt/review distinct) | — | MEDIUM | billing, payments | — | — |
| 012 | Monetary conservation | PARTIAL (exact integers, one unit; OPD-06 boundary open) | OPD-06 + real totals | HIGH | billing, payments | P3/P4 | Finance |
| 013 | Completion proof | ENFORCED in reads (stage/receipt ≠ completed sale) | real report parity | HIGH | reporting | P1 | — |
| 014 | Rejection history | NOT IMPLEMENTED (no reject command); legacy history imported | OPD-08 | HIGH | finance | P4 | Finance |
| 015 | Refund distinction | NOT IMPLEMENTED | OPD-04 | HIGH | finance | P4 | Finance + Product |
| 016 | Namespaced status | ENFORCED (domain-scoped states) | — | MEDIUM | all | — | — |
| 017 | Preserve legacy status | ENFORCED (raw kept; unknown → UNKNOWN); `sn_lead_statuses` unspecified | spec + real dictionary | HIGH | cases, billing | P1 | Engineering |
| 018 | Complete KPI metadata | ENFORCED (metric contracts) | OPD-07 | MEDIUM | reporting | P1 | per §19 |
| 019 | Pre-invoice semantics | ENFORCED in projections | real parity | MEDIUM | reporting | P1 | — |
| 020 | Case coverage | PARTIAL (Dot, MIS missing) | specs + coverage report | HIGH | reporting | P1 | Engineering |
| 021 | Report parity | ENFORCED in tests (card = table = detail) | real parity | MEDIUM | reporting | P1 | — |
| 022 | Temporal semantics | ENFORCED (current / event-range / as-of) | — | MEDIUM | reporting | — | — |
| 023 | Four permission layers | ENFORCED (route/module/action/row), but no principals in production | E-1 | HIGH | all | P0 | Security |
| 024 | View is not write | ENFORCED (separate read and command capabilities) | real-account tests | HIGH | all | P1 | — |
| 025 | Split sensitive authorities | PARTIAL (maker ≠ checker evaluator; OPD-08) | OPD-08 | HIGH | finance, HR | P4 | Finance |
| 026 | Pure business read | ENFORCED (lint + GET fingerprint tests) | — | HIGH | all | — | — |
| 027 | Technical effects bounded | ENFORCED (offline jobs explicit; no startup migration) | — | HIGH | platform | — | — |
| 028 | Effective-dated HR | PARTIAL (history imported; change requests unspecified) | specs + HR runtime test | HIGH | organization | P2 | HR |
| 029 | Historical attribution | ENFORCED (snapshots; hierarchy never rewrites credit) | real parity | HIGH | commission, reporting | P6 | Finance |
| 030 | Termination accountability | NOT IMPLEMENTED | OPD-05 | HIGH | HR | P0 (P0-6) | HR + Sales + Finance |
| 031 | Locked approved input | ENFORCED in shadow (runs immutable) | — | HIGH | commission | P6 | — |
| 032 | Idempotent entitlement | ENFORCED in shadow (entitlement key) | OPD-09 | HIGH | commission | P6 | Finance |
| 033 | Explicit engine / correction | PARTIAL (engine routing UNKNOWN for history) | OPD-09 + wp option history | HIGH | commission | P6 | Finance |
| 034 | Truthful failure results | ENFORCED (SUCCESS/PARTIAL/UNKNOWN/BLOCKED distinct; UNKNOWN never success) | — | HIGH | all writes | — | — |
| 035 | Compatibility preservation | ENFORCED (native IDs, legacy refs, no historical rewrite) | real import | HIGH | all | P1 | — |
| 036 | Reconcile before cutover | **NOT MET** (no real reconciliation) | filled data register | HIGH | all | P0 (P0-3) | Human Owner |

### Compatibility facts (VERIFIED in code)

| Fact | Status |
|---|---|
| Legacy IDs | Preserved as `source_key` / `LegacyReference` per row. Stable target IDs come from `ids.Stable`. |
| Invoice references | Kept: `invoice_ref`, legacy invoice id. |
| Receipt links | Imported as links. Receipt **files** are not extracted. |
| Subscription links | Kept (Slice 10). |
| Custom statuses | Raw values kept. |
| Reporting definitions | Metric contracts exist. |
| Data-source completeness | **Incomplete** (§1 of the data register). |

## 4. Answers to the mission questions

| Question | Answer |
|---|---|
| Overall | **NO-GO** |
| Real-data parity coverage | **0 cohorts.** Synthetic only. |
| Rollback feasibility | Configuration rollback (class A) exists for new writes. Committed effects need compensation commands, which do not exist. Restore is NOT VERIFIED. |
| Safe cutover sequence | Runbook phases P0 → P9. Only P0 (prerequisites) may start; it is engineering and decision work, not a switch. |
| Additional implementation required | **Yes.** E-1…E-7 plus every decision-dependent command (approve/reject, posting, refunds, invoice issue/return, HR apply, Biawin connector). |

## 5. Safety statement for this audit

No production access occurred. Specifically, this audit:
- did not deploy anything or change any flag;
- did not run migrations against any non-disposable database;
- did not change any system of record;
- did not call any Biawin, gateway or legacy endpoint;
- did not approve any decision;
- changed no product code or schema. The only non-document file is the read-only evidence script.

`plugin/current` is unchanged.
