# RM-B02 — Proposed role-to-capability matrix (8 roles × 43 capabilities)

**PROPOSAL — NOT APPROVED. Documentation only. No grant, bundle, flag, migration, account or configuration is created or changed.**

| | |
|---|---|
| Workstream | RM-B02 (R7 roadmap §7 B), associated with D-42 |
| Baseline | `main` @ `24f8a5badcefdfb6506df344bb686d71e5b0d634` (PR #66, RM-ACCESS-01, merged) |
| Branch | `docs/rm-b02-role-capability-matrix` |
| D-42 tracker status | `READY_FOR_REVIEW` (unchanged by this document) |
| Approvals recorded by this document | **0** |
| Production | **NO-GO** (unchanged) |

Companion documents: [scope and ownership](rm-b02-scope-and-ownership.md) · [separation of duties](rm-b02-separation-of-duties.md) · [D-42 decision package](rm-b02-d42-decision-package.md) · [executive summary (fa)](rm-b02-executive-summary.fa.md) · [evidence traceability](rm-b02-evidence-traceability.md).

## 1. What this document is and is not

It **is** an evidence-based proposal: for each of the eight business roles and each of the 43 existing backend capabilities, a documentation classification, a proposed analytical scope, and the conditions that would have to be satisfied first.

It **is not**:
- an authorization approval, a grant, or a runtime state. No cell here is `ACTIVE`. The classifications `PROPOSED_ALLOW`, `PROPOSED_CONDITIONAL`, `PROPOSED_DENY`, `DEFERRED_DECISION` and `NOT_APPLICABLE` exist only in this document;
- a decision on D-42. In R5 and M2, D-42 decides only **who may grant, edit and revoke** capabilities. Which role is eligible for which capability is a separate question with no decision ID yet (M2 proposed record 1). This document informs both, approves neither (see CF-01 in §10 and the [decision package](rm-b02-d42-decision-package.md));
- a design of new capabilities. Product actions with no backend capability are listed in §9 as **requirements**, never as implemented keys.

## 2. Existing system facts (CODE VERIFIED at the baseline)

| Fact | Evidence |
|---|---|
| The registry holds **43** capabilities; the database catalog and the OpenAPI enum hold the same 43 | `internal/identity/api/api.go` `vocabulary`; `identity.capability` (migrations 00001–00010); `api/openapi/components/hr.yaml#/Capability`; test `TestL_ContractEnumEqualsRegistry` (RM-ACCESS-01) |
| There are **9** target roles: the 8 business roles plus the technical `invoice_reviewer` | `api.go` `roles`; legacy `sn_invoice_reviewer` maps to it (`internal/integration/legacy/shadow/mapping.go:37`) |
| Only `hr` has a bundle: the 6 HR reads. The other 8 roles, including `invoice_reviewer`, have empty bundles | `api.go` `roles`; `00001_identity.sql:137-143`; `TestVocabularyShape` |
| **No ACTIVE grant exists or can exist**: the grant table only accepts `SHADOW` grants with basis `LEGACY_PROFILE_ROLE`; no grant, revoke or audit command exists | `00001_identity.sql:195` (`role_grant_shadow_only`); M2 D-42 analysis §1 |
| The evaluator is a **union**: every counting ACTIVE grant whose bundle holds the capability matches; all matched roles are reported, sorted, with no selection | `internal/authorization/authorization.go`; RM-ACCESS-01 test J |
| Every allowed decision is scope **ORGANIZATION**. There is no own, team or subtree scope in the evaluator | `authorization.go:163` (`ScopeOrganization`) |
| Row guards exist **inside** some commands and own-scope reads only: contact outcome (current holder), HR request create (descendant seller) and review (current reviewer), HR request read (requested or reviewing), payments upload/batch/submission (own), commission own (own user), fulfillment work (per-service scope) | Slice 11, 12, 13, 14 command matrices |
| Feature flags give **availability only**, never authority; `allowed` = ACTIVE grant **and** that capability's own flag | RM-ACCESS-01 §4–5 |
| No capability exists for: Finance approve/reject, posting, refund, settlement, commission approval, HR final apply, credential management, grant administration, invoice issuance, next stage, allocation/return, MIS import or maintenance, Biawin activation | `TestNoFinancialExecutionCapability`; Slice 11–14 blocked lists; registry comments |
| Real users can log in (R1) but every business read and write answers 403 for them | R7 §3; `web/e2e/seller-contact-outcome.spec.ts` |
| HR and Finance role runtime: **NOT LIVE VERIFIED** | AGENTS.md; Cross-Role V1 §1 |

## 3. Classification rules and global preconditions

### 3.1 Meaning of each classification

| Classification | Meaning in this document | Evidence rule |
|---|---|---|
| `PROPOSED_ALLOW` | The capability's **implemented** semantics and scope match a need stated in a frozen or accepted source for this role, with no role-specific authorization condition beyond the global preconditions. It may carry the key `LIV` only: an activation precondition (HR / Finance runtime verification with real role accounts), not an authorization condition | Must cite the source (§6 justification) |
| `PROPOSED_CONDITIONAL` | A stated need exists, but a named condition must be met first (usually a row scope that does not exist yet, a field policy or an open decision) | Must cite the source **and** name every condition key (§7) |
| `PROPOSED_DENY` | No source states a need for this role, or a source places it outside the role (`MUST NOT OWN`, `Should Hide`, an SoD conflict). Not proposed; a later decision may revisit it | Cites the excluding source where one exists |
| `DEFERRED_DECISION` | A need may exist, but the deciding question is open (or no product contract exists). The cell is left undecided on purpose | Names the open decision or gap |
| `NOT_APPLICABLE` | The capability is own-scoped to an activity the role never performs (an own-scope read that would return nothing) | Explained per cell |

Rules applied to every cell:
- **No rank inheritance.** A higher tier gets nothing because a lower tier has it. Each cell for Supervisor, Senior, Manager and Deputy was decided from that role's own source (Cross-Role V1 §1: higher position does not inherit Finance/HR/MIS authority or lower-tier writes).
- **Navigation is not authorization.** RM-E02 destination mappings are cited only as context; one of them conflicts with a role contract (CF-06).
- **Read scope is not write scope.** Writes are classified separately even when the role reads the same object.
- **Unknown stays unknown.** Where no source answers, the cell is `DEFERRED_DECISION`, not `ALLOW` or `DENY` by seniority.
- **Cross-domain writes are never proposed**: no Sales role receives a Finance, HR-apply or MIS write; HR receives no Finance or Sales capability; Finance receives no HR or Sales write; MIS receives no financial or HR capability.

### 3.2 Global preconditions (apply to every cell, not repeated)

| Key | Precondition | Source |
|---|---|---|
| G1 | D-42 decided (who grants, edits, revokes) **and** E-7 grant-change procedure with audit implemented (RM-B03) | R5 inventory; M2 §3 |
| G2 | A per-role capability-eligibility decision exists. For the seller this is M2 proposed record 1; for the other seven roles **no record is proposed yet** (gap GAP-01 in the decision package) | M2 §2, §4 |
| G3 | D-34 account provisioning decided and implemented (RM-B01) | R5; R7 §7 B |
| G4 | For any SoD-sensitive action: R5-SEC-01 decided with option A or B and implemented (RM-B04). Account-level grants never satisfy SoD | R5 P-SEC §2; M2 §2-2 |
| G5 | The capability's own feature flag and rollout mode stay separate availability gates; activation is separately authorized (R5 readiness ladder steps 4–6); Production remains NO-GO | RM-ACCESS-01 §4; R5 inventory |

### 3.3 Sources and their status (summary; full list in the traceability document)

| Source | Status at HEAD | Used as |
|---|---|---|
| Cross-Role Architecture V1 (XR) | FROZEN TARGET — IMPLEMENTATION DEFERRED | Product authority for role map (§2), ownership (§5), permission boundary (§12) |
| Gate 0 (G0) | FROZEN TARGET / IMPLEMENTATION DEFERRED | Invariants INV-023…025; Supervisor input §19 |
| Product-role Audit (A) and Decisions/Visibility appendix (DV) | ACCEPTED (cited as `Audit پذیرفته‌شده` by XR) | Seller/Supervisor contract basis (XR §1); visibility per role |
| Senior (SS), Manager (MG), Deputy (DP), MIS (MS), HR (HRS), Finance (FS) specs | FROZEN PRODUCT TARGET / IMPLEMENTATION DEFERRED (header of each) | Per-role permission concepts (§22 / §15 / §14 / §16) |
| Standalone Target Architecture V1 (ARCH) §4A, §17, §18 | Architecture freeze record; authorizes no slice | Module-level role → capability family intent |
| M2 intake and D-42 seller analysis | DRAFT — NOT APPROVED | Seller options and scope definitions |
| R5 inventory, R6 tracker | Inventory / tracker (statuses unchanged) | Decision IDs and statuses |
| RM-E02, RM-ACCESS-01, R7 audit | Merged engineering / audit records | Code facts and navigation context only |
| Seller Design V1, Supervisor design | Visual only | **Not** business or authorization authority |


## 4. Totals (documentation classifications, not runtime grant states)

| Classification | Cells |
|---|---|
| PROPOSED_ALLOW | 31 |
| PROPOSED_CONDITIONAL | 75 |
| PROPOSED_DENY | 147 |
| DEFERRED_DECISION | 84 |
| NOT_APPLICABLE | 7 |
| **Total (8 roles × 43 capabilities)** | **344** |

| Role | ALLOW | CONDITIONAL | DENY | DEFERRED | N/A |
|---|---|---|---|---|---|
| SEL (`seller`) | 1 | 8 | 24 | 10 | 0 |
| SUP (`supervisor`) | 4 | 11 | 16 | 12 | 0 |
| SNR (`senior_supervisor`) | 4 | 11 | 16 | 12 | 0 |
| MGR (`sales_manager`) | 4 | 11 | 16 | 12 | 0 |
| DEP (`sales_deputy`) | 1 | 12 | 19 | 10 | 1 |
| MIS (`mis`) | 4 | 9 | 16 | 12 | 2 |
| HR (`hr`) | 6 | 0 | 30 | 5 | 2 |
| FIN (`finance`) | 7 | 13 | 10 | 11 | 2 |

## 5. Capability inventory (existing system facts at HEAD)

Every row is an existing registry entry (`internal/identity/api/api.go` `vocabulary`), present in the database catalog (`identity.capability`) and in the OpenAPI `Capability` enum (`api/openapi/components/hr.yaml`). **Current role bundle** is the code fact; **activation** is the same for all 43: no ACTIVE grant exists or can exist today (`identity.role_grant` CHECK `role_grant_shadow_only`, `internal/identity/migrations/sql/00001_identity.sql:195`). Persian descriptions are translations of the English registry text, not new definitions.

| # | Capability | توضیح فارسی | Domain | Class | Current bundle | Feature flag (availability only) | Implemented row scope | Server enforcement | Sensitivity | Related decisions |
|---|---|---|---|---|---|---|---|---|---|---|
| 1 | `identity.user.read` | خواندن حساب‌های کاربری (مدل هویت سایه) | identity | READ | `hr` | `organization.hr_read_shadow` | ORGANIZATION (evaluator); no row filter | `internal/identity/http/http.go:40,78` | HIGH (account identity) | D-42, D-34, R5-SEC-01 |
| 2 | `org.person.read` | خواندن افراد (هویت منابع انسانی) | organization | READ | `hr` | `organization.hr_read_shadow` | ORGANIZATION (evaluator); no row filter | `internal/organization/http/http.go:154,200` | HIGH (personal data) | D-42, R5-SEC-01 |
| 3 | `org.employment.read` | خواندن ویژگی‌های چرخهٔ استخدام | organization | READ | `hr` | `organization.hr_read_shadow` | ORGANIZATION (evaluator); no row filter | `internal/organization/http/http.go:106,177` | HIGH (employment status) | D-42, OPD-05 |
| 4 | `org.position.read` | خواندن سمت‌ها و سطوح | organization | READ | `hr` | `organization.hr_read_shadow` | ORGANIZATION (evaluator); no row filter | `internal/organization/http/http.go:113,178,241` | MEDIUM | D-42 |
| 5 | `org.unit.read` | خواندن درخت واحدهای سازمانی | organization | READ | `hr` | `organization.hr_read_shadow` | ORGANIZATION (evaluator); no row filter | `internal/organization/http/http.go:117,179,298` | LOW-MEDIUM (structure) | D-42 |
| 6 | `org.assignment.read` | خواندن انتساب‌های گزارش‌دهی، تاریخچه و یافته‌های سلسله‌مراتب | organization | READ | `hr` | `organization.hr_read_shadow` | ORGANIZATION (evaluator); no row filter | `internal/organization/http/http.go:124,179,220,334` | MEDIUM-HIGH (hierarchy history) | D-42, OPD-05, SD-04 |
| 7 | `customers.customer.read` | خواندن مشتریان سایه، نقاط تماس پوشانده، شناسه‌ها و شواهد | customers | READ | none | `customers.identity_shadow_read` | ORGANIZATION (evaluator); contact points masked | `internal/customers/http/http.go:124,160,191` | HIGH (customer PII) | D-30, ADR #18, D-29 / D-31 |
| 8 | `customers.merge_case.read` | خواندن صف نامزدهای ادغام (بدون اختیار ادغام) | customers | READ | none | `customers.identity_shadow_read` | ORGANIZATION (evaluator) | `internal/customers/http/http.go:148,225,256` | HIGH (identity linkage) | D-30, OPD-01, ADR #18 |
| 9 | `customers.identity_report.read` | خواندن گزارش تطبیق هویت مشتری | customers | READ | none | `customers.identity_shadow_read` | ORGANIZATION (evaluator); aggregate | `internal/customers/http/http.go:290` | MEDIUM | D-30, OPD-01 |
| 10 | `cases.case.read` | خواندن پرونده‌های سایه، تبار، تاریخچهٔ وضعیت و عکس‌فوری انتساب | cases | READ | none | `sales.read_shadow` | ORGANIZATION (evaluator) — no `my cases` scope exists | `internal/cases/http/http.go:106,145,181,213` | HIGH (customer work data) | M2 proposed record 1 and 2 (no ID), OPD-01, ADR-01 |
| 11 | `custody.custody.read` | خواندن پیش‌نمایش‌های مشتق‌شدهٔ متولی، رویدادها و یافته‌ها | custody | READ | none | `sales.read_shadow` | ORGANIZATION (evaluator) | `internal/custody/http/http.go:85,115` | MEDIUM-HIGH | OPD-10, F01, OPD-05 |
| 12 | `sales.report.read` | خواندن گزارش پوشش منبع فروش و برابری متولی | sales | READ | none | `sales.read_shadow` | ORGANIZATION (evaluator); aggregate counts | `internal/cases/http/http.go:273; internal/custody/http/http.go:162` | MEDIUM | F02, F03, OPD-07 |
| 13 | `billing.invoice.read` | خواندن فاکتورهای سایه، اقلام، مراحل پرداخت، وضعیت و شواهد مبلغ | billing | READ | none | `billing_payments.read_shadow` | ORGANIZATION (evaluator) | `internal/billing/http/http.go:154,185,219` | HIGH (financial + customer) | FRS (registry cites D-25, D-26, D-28), OPD-10, D-32 |
| 14 | `payments.attempt.read` | خواندن تلاش‌های پرداخت سایه، شاهد درگاه (هش‌شده) و فرادادهٔ رسید | payments | READ | none | `billing_payments.read_shadow` | ORGANIZATION (evaluator); gateway evidence hashed | `internal/billing/http/http.go:316; internal/payments/http/http.go:104` | HIGH (payment evidence) | FRS, D-25, D-44, OPD-10 |
| 15 | `billing.reconciliation.read` | خواندن گزارش پوشش صورت‌حساب و پرداخت | billing | READ | none | `billing_payments.read_shadow` | ORGANIZATION (evaluator); aggregate | `internal/billing/http/http.go:390` | MEDIUM | F04, TAD #19 |
| 16 | `reporting.metrics.read` | خواندن قرارداد شاخص‌ها و پیش‌نمایش‌های گزارش تجمعی | reporting | READ | none | `reporting.read_shadow` | ORGANIZATION (evaluator); aggregate projections | `internal/reporting/http/http.go:284,321,357` | MEDIUM | OPD-07, OPD-01, F02, F03, F04, F09 |
| 17 | `reporting.mis.read` | خواندن نمای کلی MIS، کیفیت منبع و پیش‌نمایش‌های تطبیق | reporting | READ | none | `reporting.read_shadow` | ORGANIZATION (evaluator) | `internal/reporting/http/http.go:700,769,889` | MEDIUM | OPD-01, F09 |
| 18 | `reporting.invoice_review.read` | خواندن سطح پرس‌وجوی بازبینی فاکتور (با ممیزی؛ بدون خروجی) | reporting | READ | none | `reporting.read_shadow` | ORGANIZATION (evaluator); audited, no export | `internal/reporting/http/http.go:971,1024` | HIGH (PII) | D-23, D-24, D-45 |
| 19 | `finance.shadow.read` | خواندن اجراها، حساب‌ها و مانده‌های دفتر سایهٔ مالی (نه سامانهٔ مرجع) | finance | READ | none | `finance.shadow_ledger_read` | ORGANIZATION (evaluator) | `internal/finance/http/http.go:286,317,349` | HIGH (financial) | D-41, OPD-06, OPD-09 |
| 20 | `finance.ledger.read` | خواندن ثبت‌های روزنامهٔ سایهٔ مالی (نه سامانهٔ مرجع) | finance | READ | none | `finance.shadow_ledger_read` | ORGANIZATION (evaluator) | `internal/finance/http/http.go:422,470` | HIGH (financial) | D-41, OPD-06 |
| 21 | `finance.reconciliation.read` | خواندن تطبیق دفتر قدیمی با دفتر سایه (نه سامانهٔ مرجع) | finance | READ | none | `finance.shadow_ledger_read` | ORGANIZATION (evaluator) | `internal/finance/http/http.go:534` | HIGH (financial) | D-41, OPD-09 |
| 22 | `biawin.subscription.read` | خواندن اشتراک‌های سایهٔ بیاوین، اقلام، پیوند فاکتور و تاریخچه (بدون فعال‌سازی) | biawin | READ | none | `biawin_fulfillment.read_shadow` | ORGANIZATION (evaluator) | `internal/biawin/http/http.go:200,241,275` | MEDIUM-HIGH | D-35a, D-35b, D-35c, D-35d, D-35e, D-38 |
| 23 | `fulfillment.case.read` | خواندن پرونده‌های سایهٔ اجرای خدمت، مجریان، فرادادهٔ شواهد و تاریخچه | fulfillment | READ | none | `biawin_fulfillment.read_shadow` | ORGANIZATION (evaluator) | `internal/fulfillment/http/http.go:190,230,264` | MEDIUM | D-35a, D-35b, D-35c, D-35d, D-35e, OPD-02 |
| 24 | `biawin.parity.read` | خواندن گزارش برابری اشتراک‌های فعال قدیمی و هدف | biawin | READ | none | `biawin_fulfillment.read_shadow` | ORGANIZATION (evaluator); aggregate | `internal/biawin/http/http.go:314` | MEDIUM | D-35a, D-35b, D-35c, D-35d, D-35e, D-38 |
| 25 | `cases.contact_outcome.record` | ثبت پیامد تماس سرنخ در اختیار (بدون اثر فاکتور، متولی یا مالی) | cases | CREATE (event) | none | `cases.nonfinancial_write (+ rollout mode)` | Row guard implemented: actor == current legacy holder; unknown holder BLOCKED; any other actor (any rank) UNAUTHORIZED | `internal/cases/http/write.go:91,111,150` | MEDIUM | M2 proposed record 1 (no ID), D-29 / D-31, ADR-01, OPD-01, OPD-10 |
| 26 | `org.hr_request.create` | ایجاد درخواست انتقال یا قطع همکاری برای فروشندهٔ در دامنه (بدون تغییر پرسنلی) | organization | CREATE | none | `organization.hr_request_write (+ rollout mode)` | Row guard implemented: subject is a seller descendant of the actor via current REPORTS_TO; actor needs one LINKED ACTIVE employment with position slug in {supervisor, senior_supervisor, sales_manager, sales_deputy}; HR creator BLOCKED | `internal/organization/http/hrrequest.go:115,120,206` | MEDIUM-HIGH (personnel) | OPD-05, SD-04, R5-SEC-01 |
| 27 | `org.hr_request.review` | تأیید مرحله‌ای یا رد درخواست HR به‌عنوان بازبین فعلی (بدون اعمال نهایی) | organization | REVIEW | none | `organization.hr_request_write (+ rollout mode)` | Row guard implemented: actor == request's current reviewer snapshot, status pending_review; HR actor or pending_hr → BLOCKED | `internal/organization/http/hrrequest.go:55,58,248` | MEDIUM-HIGH (personnel) | OPD-05, SD-04, R5-SEC-01 |
| 28 | `org.hr_request.read` | خواندن درخواست‌های HR که فراخواننده ثبت کرده یا اکنون بازبین آن است | organization | READ | none | `organization.hr_request_write` | Row filter implemented: requested by or currently reviewed by the caller | `internal/organization/http/hrrequest.go:71` | MEDIUM-HIGH | OPD-05, SD-04 |
| 29 | `integration.writeback.read` | خواندن عیب‌یابی بازنویسی به سامانهٔ قدیمی (شمارش‌ها، وضعیت‌ها، تلاش‌ها؛ بدون محتوا) | integration | READ (diagnostic) | none | `integration.writeback_read` | ORGANIZATION (evaluator); no payloads | `internal/integration/writeback/http/http.go:44` | MEDIUM | E-3, P1-8 (engineering gap), E-7 |
| 30 | `payments.upload.stage` | قرارگیری فایل رسید در انبار قرنطینه (تا اتصال، شاهد نیست) | payments | CREATE (staging) | none | `payments.manual_write` | Row filter implemented: caller's own uploads | `internal/payments/http/write.go:80` | MEDIUM | D-44 |
| 31 | `payments.batch.submit` | ثبت دستهٔ پرداخت دستی برای مرحلهٔ جاری فاکتور متعلق (بدون بازبینی، ثبت دفتری یا تغییر مرحله) | payments | CREATE (submission) | none | `payments.manual_write (mode CONTROLLED_WRITEBACK)` | Row guard implemented: invoice.seller_user_id = actor AND issued_by_user_id = actor | `internal/payments/http/write.go:176,196,247` | HIGH (financial evidence) | D-43, D-28, OPD-06, OPD-10, OPD-08 |
| 32 | `payments.submission.read` | خواندن زمینهٔ ثبت پرداخت و دسته‌های پرداخت خود | payments | READ | none | `payments.manual_write` | Row filter implemented: caller's own batches | `internal/payments/http/write.go:143,303` | MEDIUM-HIGH | D-43, OPD-10 |
| 33 | `payments.evidence.read` | خواندن عیب‌یابی شواهد پرداخت و سوابق تطبیق شواهد | payments | READ (diagnostic) | none | `payments.manual_write` | ORGANIZATION (evaluator) | `internal/payments/http/write.go:330` | HIGH (receipt evidence) | D-44, OPD-08 |
| 34 | `finance.review.read` | خواندن صف بررسی مالی و دروازهٔ تصمیم آن (بدون اختیار تأیید/رد) | finance | READ | none | `finance.review_read` | ORGANIZATION (evaluator); no decision authority exists | `internal/finance/http/review.go:56` | HIGH (financial) | OPD-08, D-43, REJECT-VOCAB (R4 gap), R5-SEC-01 |
| 35 | `commission.shadow.read` | خواندن اجراهای سایهٔ پورسانت، محاسبات، برابری و یافته‌ها (بدون تأیید/ثبت) | commission | READ | none | `commission.read_shadow` | ORGANIZATION (evaluator) | `internal/commission/http/http.go:180,233,263` | HIGH (earnings of all recipients) | OPD-09, D-36, D-41 |
| 36 | `commission.policy.read` | خواندن نسخه‌های مشاهده‌شدهٔ سیاست پورسانت (هیچ‌کدام مصوب نیست) | commission | READ | none | `commission.read_shadow` | ORGANIZATION (evaluator) | `internal/commission/http/http.go:352` | MEDIUM | OPD-09, D-36 |
| 37 | `commission.own.read` | خواندن اعتبارهای تاریخی پورسانت خود فراخواننده (فقط دامنهٔ خود) | commission | READ | none | `commission.read_shadow` | Row filter implemented: authenticated user only; never broadened by hierarchy | `internal/commission/http/http.go:430` | MEDIUM (own earnings) | OPD-09, D-41 |
| 38 | `fulfillment.work.read` | خواندن پرونده‌های کار خدماتی که فراخواننده اکنون سرپرست یا مجری آن است | fulfillment | READ | none | `fulfillment.work_read` | Row filter implemented: per-service SUPERVISOR/EXECUTOR scope events (catalog), not HR hierarchy | `internal/fulfillment/http/work.go:108,165` | MEDIUM | D-35a, D-35b, D-35c, D-35d, D-35e, OPD-05, OPD-08 |
| 39 | `fulfillment.work.assign` | واگذاری پروندهٔ کار خدمتی که فراخواننده سرپرست آن است به مجری فعلی همان خدمت | fulfillment | UPDATE (assignment) | none | `fulfillment.work_write (+ writeback for CONTROLLED)` | Row guard implemented: actor is current SUPERVISOR of the service; assignee current EXECUTOR | `internal/fulfillment/http/work.go:323` | MEDIUM | OPD-05, D-35a, D-35b, D-35c, D-35d, D-35e |
| 40 | `fulfillment.work.execute` | ثبت اقدام، فرادادهٔ شاهد و پیامد پروندهٔ کار واگذارشده به فراخواننده | fulfillment | EXECUTE | none | `fulfillment.work_write (+ writeback for CONTROLLED)` | Row guard implemented: assignee with current EXECUTOR scope; financial behaviour NONE only | `internal/fulfillment/http/work.go:348,373,398` | MEDIUM | D-35a, D-35b, D-35c, D-35d, D-35e, D-38, OPD-04 |
| 41 | `biawin.activation.read` | خواندن تصمیم‌های فعال‌سازی سایهٔ بیاوین و زمینهٔ تطبیق آن‌ها | biawin | READ | none | `fulfillment.work_read` | ORGANIZATION (evaluator) | `internal/biawin/http/activation.go:31` | MEDIUM | D-35a, D-35b, D-35c, D-35d, D-35e, D-38 |
| 42 | `integration.provider_operation.read` | خواندن عملیات ارائه‌دهندهٔ بیرونی و وضعیت تطبیق آن‌ها | integration | READ | none | `fulfillment.work_read` | ORGANIZATION (evaluator) | `internal/integration/providerops/http/http.go:43` | MEDIUM | D-38, D-35d |
| 43 | `integration.provider_operation.reconcile` | ثبت شاهد تطبیق انسانی برای عملیات ارائه‌دهندهٔ نامعلوم (هرگز ارسال نمی‌کند) | integration | EXECUTE (reconciliation evidence) | none | `integration.provider_reconcile` | Actor rule implemented: any proven user id; no SoD check against the dispatcher/approver | `internal/integration/providerops/http/http.go:78` | HIGH (resolves UNKNOWN external outcome) | D-38, D-35d, R5-SEC-01 |

Additional gates on top of the flag (code facts): command capabilities also pass a rollout mode (`rollout.FromFlags`; SHADOW_COMPARE vs CONTROLLED / CONTROLLED_WRITEBACK) that is not reflected in effective-access (RM-ACCESS-01 §10.3); `payments.batch.submit` applies only in `CONTROLLED_WRITEBACK`; fulfillment commands apply only to target-native work cases.

## 6. Proposed role × capability matrix

Cell format: `code · proposed scope · condition keys`. Codes: **A** PROPOSED_ALLOW, **C** PROPOSED_CONDITIONAL, **X** PROPOSED_DENY, **D** DEFERRED_DECISION, **N** NOT_APPLICABLE. Condition keys are defined in §7. Every cell also carries the global preconditions G1–G5 (§3); they are not repeated. No cell is a grant.

Role columns: SEL = Seller (فروشنده), SUP = Supervisor (سرپرست), SNR = Senior Supervisor (سرپرست ارشد), MGR = Sales Manager (مدیر فروش), DEP = Sales Deputy (معاون فروش), MIS, HR (منابع انسانی), FIN = Finance (مالی).

### 6.1 Identity and organization (HR reads)

| # | Capability | SEL | SUP | SNR | MGR | DEP | MIS | HR | FIN |
|---|---|---|---|---|---|---|---|---|---|
| 1 | `identity.user.read` | X | X | X | X | X | X | A · ORGANIZATION · LIV | X |
| 2 | `org.person.read` | X | C · DIRECT_TEAM · RS, FP | C · AUTHORIZED_SUBTREE · RS, FP | C · AUTHORIZED_SUBTREE · RS, FP | C · GOVERNED_TERRITORY · RS, FP | C · ORGANIZATION (recipient eligibility only) · FP, MISS | A · ORGANIZATION · LIV | C · SPECIFIC_APPROVAL_SCOPE (subjects of a reviewed financial item) · LNK, FP, LIV |
| 3 | `org.employment.read` | X | C · DIRECT_TEAM · RS, FP | C · AUTHORIZED_SUBTREE · RS, FP | C · AUTHORIZED_SUBTREE · RS, FP | C · GOVERNED_TERRITORY · RS, FP | C · ORGANIZATION (active status only) · FP, MISS | A · ORGANIZATION · LIV | C · SPECIFIC_APPROVAL_SCOPE · LNK, FP, LIV |
| 4 | `org.position.read` | X | C · DIRECT_TEAM · RS | C · AUTHORIZED_SUBTREE · RS | C · AUTHORIZED_SUBTREE · RS | C · GOVERNED_TERRITORY · RS | C · ORGANIZATION (recipient eligibility only) · MISS | A · ORGANIZATION · LIV | C · SPECIFIC_APPROVAL_SCOPE · LNK, LIV |
| 5 | `org.unit.read` | X | C · DIRECT_TEAM · RS | C · AUTHORIZED_SUBTREE · RS | C · AUTHORIZED_SUBTREE · RS | C · GOVERNED_TERRITORY · RS | C · ORGANIZATION · MISS | A · ORGANIZATION · LIV | C · SPECIFIC_APPROVAL_SCOPE · LNK, LIV |
| 6 | `org.assignment.read` | X | C · DIRECT_TEAM · RS | C · AUTHORIZED_SUBTREE · RS | C · AUTHORIZED_SUBTREE · RS | C · GOVERNED_TERRITORY · RS | C · ORGANIZATION (recipient eligibility only) · MISS | A · ORGANIZATION · LIV | C · SPECIFIC_APPROVAL_SCOPE · LNK, LIV |

**Justification and evidence:**

- **1. `identity.user.read`.** HR bundle already lists it (api.go `roles`; migration 00001:137). HR spec §16 `view_access_state`; DV HR Must See (user/profile identity, effective access). No sales/MIS/Finance spec need; XR §2 MUST NOT OWN.
- **2. `org.person.read`.** HR bundle (existing). Sales tiers: G0 §19 Team scope / SS §22 `view_multiteam` (authorized subtree, compensation and credentials hidden) / MG §15 `view_scoped_teams` / DP §15 `view_manager_hierarchy`. MIS: XR §5 Workforce Subject `V eligible recipient`. Finance: XR §5 `V minimum input`. Seller: DV Should Hide (broad management structure).
- **3. `org.employment.read`.** Same basis as org.person.read. SS §25 Must See `activeprofiles`; MS DV Must See `recipient active`.
- **4. `org.position.read`.** Same basis as org.person.read.
- **5. `org.unit.read`.** Same basis as org.person.read. SS §9 Hierarchy/Structure; DP §7 Hierarchy & Performance.
- **6. `org.assignment.read`.** Same basis as org.person.read. XR §4 Hierarchy Assignment is not historical credit ownership.

### 6.2 Customers

| # | Capability | SEL | SUP | SNR | MGR | DEP | MIS | HR | FIN |
|---|---|---|---|---|---|---|---|---|---|
| 7 | `customers.customer.read` | C · ASSIGNED_CASES · RS, FP, CUS, M2R2 | C · DIRECT_TEAM (case-linked) · RS, FP, CUS | C · AUTHORIZED_SUBTREE (case-linked) · RS, FP, CUS | C · AUTHORIZED_SUBTREE (case-linked) · RS, FP, CUS | D | C · ORGANIZATION (provenance fields) · FP, CUS, MISS | X | C · SPECIFIC_APPROVAL_SCOPE (customer of a reviewed invoice) · LNK, FP, CUS, LIV |
| 8 | `customers.merge_case.read` | X | X | X | X | X | D | X | X |
| 9 | `customers.identity_report.read` | X | X | X | X | X | D | X | X |

**Justification and evidence:**

- **7. `customers.customer.read`.** Registry comment: no bundle because role assignment for customer identity is not evidenced (D-30 open). Seller: DV Sensitive (customer number/document only for an authorized case). Supervisor: G0 §19 Customer History destination. Senior: SS §22 `view_scoped_invoices` (scoped customer). Manager: MG §15 `view_customer_history`. Deputy: DV `sensitive detail only with need and policy` (policy undefined). MIS: MS §7 Case/Source Identity; DV Sensitive phone/source provenance. Finance: XR §9 linked context. HR: DV Should Hide (customer commercial history).
- **8. `customers.merge_case.read`.** D-30 options include `Merge by MIS with documented multi-factor evidence`; the queue reader is tied to who decides merges. XR §3 disputed identity is shared Product/MIS/Sales. No sales, HR or Finance spec need.
- **9. `customers.identity_report.read`.** Same as customers.merge_case.read.

### 6.3 Cases, custody and sales reports

| # | Capability | SEL | SUP | SNR | MGR | DEP | MIS | HR | FIN |
|---|---|---|---|---|---|---|---|---|---|
| 10 | `cases.case.read` | C · ASSIGNED_CASES (current holder) · RS, M2R1, M2R2 | C · DIRECT_TEAM · RS | C · AUTHORIZED_SUBTREE · RS | C · AUTHORIZED_SUBTREE · RS | C · GOVERNED_TERRITORY · RS | C · ORGANIZATION (source scope) · MISS, O01 | X | C · SPECIFIC_APPROVAL_SCOPE (case of a reviewed invoice) · LNK, LIV |
| 11 | `custody.custody.read` | X | C · DIRECT_TEAM + own custody · RS | C · AUTHORIZED_SUBTREE + own custody · RS | C · AUTHORIZED_SUBTREE + own custody · RS | C · GOVERNED_TERRITORY + own custody · RS | C · ORGANIZATION (governed source) · MISS | D | C · SPECIFIC_APPROVAL_SCOPE · LNK, LIV |
| 12 | `sales.report.read` | X | X | X | X | X | A · ORGANIZATION | X | X |

**Justification and evidence:**

- **10. `cases.case.read`.** Seller: M2 §4-1 option A (engineering proposal, conditional on a restricted scope) and §4-2 option 1 current holder (proposal, not selected); XR §2 Seller read domain `own Case`. Supervisor: XR §2 `direct team`. Senior: SS §22 `view_scoped_cases`. Manager: MG §15 `view_scoped_cases` KEEP. Deputy: DP §15 `view_scoped_cases` CONDITIONAL source-complete. MIS: MS §14 `view_source_cases` KEEP; XR §2. Finance: XR §9 linked context. HR: DV Should Hide (daily sales operations).
- **11. `custody.custody.read`.** Seller: M2 §5 explicitly excludes it from the proposed seller bundle (separate decision). Sales tiers: XR §5 Current Custody V/Δ. MIS: MS §14 `view_custody_history` KEEP. HR: XR §5 `V impact, not automatic Case Δ` (impact view for termination; OPD-05). Finance: XR §5 `V linked, not custody Δ`.
- **12. `sales.report.read`.** Coverage and parity diagnostics. MIS: MS §10 Data Quality/Reconciliation KEEP; XR §2 MIS read domain. Seller excluded by M2 §5. Sales tiers: DV Should Hide technical raw statuses / technical logs; their governed KPIs are reporting.metrics.read.

### 6.4 Billing and payments (reads)

| # | Capability | SEL | SUP | SNR | MGR | DEP | MIS | HR | FIN |
|---|---|---|---|---|---|---|---|---|---|
| 13 | `billing.invoice.read` | C · OWN (invoice seller) · RS | C · DIRECT_TEAM · RS, FP | C · AUTHORIZED_SUBTREE · RS, FP | C · AUTHORIZED_SUBTREE · RS, FP | C · GOVERNED_TERRITORY (read-only) · RS, FP | C · ORGANIZATION (proof fields) · FP, MISS | X | C · ORGANIZATION (financial domain) · FRS, LIV |
| 14 | `payments.attempt.read` | C · OWN · RS | C · DIRECT_TEAM · RS, FP, O10 | C · AUTHORIZED_SUBTREE · RS, FP, O10 | C · AUTHORIZED_SUBTREE · RS, FP, O10 | X | X | X | C · ORGANIZATION (financial domain) · FRS, LIV |
| 15 | `billing.reconciliation.read` | X | X | X | X | X | A · ORGANIZATION | X | A · ORGANIZATION · LIV |

**Justification and evidence:**

- **13. `billing.invoice.read`.** Seller: XR §2 read domain own invoice/stage. Supervisor: G0 §19 Invoice permissions scoped read. Senior: SS §22 `view_scoped_invoices`. Manager: MG §15 KEEP. Deputy: DP §15 KEEP readonly oversight. MIS: XR §9 `MIS needed proof`. Finance: FS §15 `view_scoped_financial_data`; XR §2 Finance read domain. HR: DV Should Hide (Finance receipt detail).
- **14. `payments.attempt.read`.** Seller: XR §12 invoice issue/evidence A. Supervisor/Senior/Manager: XR §12 C (scoped assist); SS §25 hide unrelated receipt bank details. Deputy: DV Should Hide direct receipt operations; XR §12 R baseline. MIS: DV Should Hide customer bank card detail (uses billing.reconciliation.read). Finance: FS §15 `review_payment_evidence`.
- **15. `billing.reconciliation.read`.** Coverage report. MIS: XR §2 read domain financial linkage; XR §6 MIS↔Finance discrepancy. Finance: FS §13 Reconciliation. No sales/HR need.

### 6.5 Reporting and MIS

| # | Capability | SEL | SUP | SNR | MGR | DEP | MIS | HR | FIN |
|---|---|---|---|---|---|---|---|---|---|
| 16 | `reporting.metrics.read` | D | C · DIRECT_TEAM · RS, O07 | C · AUTHORIZED_SUBTREE · RS, O07 | C · AUTHORIZED_SUBTREE · RS, O07 | C · GOVERNED_TERRITORY · RS, O07 | A · ORGANIZATION | X | C · ORGANIZATION (financial reports) · FRS, O07, LIV |
| 17 | `reporting.mis.read` | X | X | X | X | D | A · ORGANIZATION | X | C · SPECIFIC_APPROVAL_SCOPE (discrepancy context) · LNK, LIV |
| 18 | `reporting.invoice_review.read` | X | X | X | X | X | X | X | D |

**Justification and evidence:**

- **16. `reporting.metrics.read`.** Seller own KPIs exist (DV KPI rows) but no own-scope projection exists and OPD-07 is open. Sales tiers: SS §22/MG §15/DP §15 `view_reports` with source-specific scope. MIS: MS §11 Reports & KPI Trust KEEP (metrics already served labelled NOT FINAL, R5 OPD-07 safe behaviour). Finance: FS §14 Reports. HR: no spec need for sales metrics.
- **17. `reporting.mis.read`.** MIS: MS §5 Target IA; RE2 maps MIS today. Deputy: RE2 maps Deputy diagnostics → /mis/reconciliation, but DV Should Hide technical logs daily and XR §2 Deputy MUST NOT OWN MIS: conflict CF-06, so DEFERRED. Finance: XR §6 MIS↔Finance discrepancy (named Finance reviewer).
- **18. `reporting.invoice_review.read`.** ARCH §18 maps `sn_invoice_reviewer` to the invoice review read; ARCH §4A Invoice Reviewer is a separate technical role. D-23 approvers include Finance; D-24 asks who owns the surface.

### 6.6 Finance shadow ledger

| # | Capability | SEL | SUP | SNR | MGR | DEP | MIS | HR | FIN |
|---|---|---|---|---|---|---|---|---|---|
| 19 | `finance.shadow.read` | X | X | X | X | X | X | X | A · ORGANIZATION · LIV |
| 20 | `finance.ledger.read` | X | X | X | X | X | X | X | A · ORGANIZATION · LIV |
| 21 | `finance.reconciliation.read` | X | X | X | X | X | X | X | A · ORGANIZATION · LIV |

**Justification and evidence:**

- **19. `finance.shadow.read`.** Finance: FS §10 Ledger; XR §2 Finance read domain ledger; ARCH §4A. Shadow, labelled not the system of record. Sales `own ledger` needs are own-scope (commission.own.read), not this org-wide ledger. XR §2 MIS/HR MUST NOT OWN ledger.
- **20. `finance.ledger.read`.** Same as finance.shadow.read.
- **21. `finance.reconciliation.read`.** Finance: FS §13 Reconciliation. MIS reconciliation of financial linkage uses billing.reconciliation.read.

### 6.7 Biawin and fulfillment (reads)

| # | Capability | SEL | SUP | SNR | MGR | DEP | MIS | HR | FIN |
|---|---|---|---|---|---|---|---|---|---|
| 22 | `biawin.subscription.read` | D | D | D | D | D | D | X | D |
| 23 | `fulfillment.case.read` | D | D | D | D | D | D | X | D |
| 24 | `biawin.parity.read` | X | X | X | X | X | D | X | D |

**Justification and evidence:**

- **22. `biawin.subscription.read`.** XR §7: Biawin is not designed in this phase; no role spec defines Biawin/subscription access. G0 §19 mentions subscription context inside Supervisor invoices, but the mapping to target subscriptions (ARCH §4B #20) is not evidenced. HR: DV Should Hide customer commercial history.
- **23. `fulfillment.case.read`.** Same as biawin.subscription.read. Operations/execution actors are UNMAPPED legacy roles (mapping.go), outside the eight business roles.
- **24. `biawin.parity.read`.** Parity diagnostic. MIS or Finance might own it as reconciliation, but no spec assigns it. Sales tiers: DV Should Hide technical diagnostics.

### 6.8 Controlled non-financial writes and diagnostics

| # | Capability | SEL | SUP | SNR | MGR | DEP | MIS | HR | FIN |
|---|---|---|---|---|---|---|---|---|---|
| 25 | `cases.contact_outcome.record` | C · ASSIGNED_CASES (current holder; implemented) · M2R1, M2R2 | D | D | D | X | X | X | X |
| 26 | `org.hr_request.create` | X | A · AUTHORIZED_SUBTREE (descendant sellers; implemented) | A · AUTHORIZED_SUBTREE (descendant sellers; implemented) | A · AUTHORIZED_SUBTREE (descendant sellers; implemented) | C · AUTHORIZED_SUBTREE (descendant sellers; implemented) · DEPV | X | D | X |
| 27 | `org.hr_request.review` | X | A · SPECIFIC_APPROVAL_SCOPE (current reviewer; implemented) | A · SPECIFIC_APPROVAL_SCOPE (current reviewer; implemented) | A · SPECIFIC_APPROVAL_SCOPE (current reviewer; implemented) | C · SPECIFIC_APPROVAL_SCOPE (current reviewer; implemented) · DEPV | X | D | X |
| 28 | `org.hr_request.read` | X | A · ASSIGNED_QUEUE (requested or reviewing; implemented) | A · ASSIGNED_QUEUE (requested or reviewing; implemented) | A · ASSIGNED_QUEUE (requested or reviewing; implemented) | C · ASSIGNED_QUEUE (requested or reviewing; implemented) · DEPV | X | D | X |
| 29 | `integration.writeback.read` | X | X | X | X | X | D | X | D |

**Justification and evidence:**

- **25. `cases.contact_outcome.record`.** Seller: XR §12 `A own`; S11 SAFE; M2 §4-1 option A. Supervisor `C assigned/assist`, Senior `C assigned only`, Manager `C explicit action only` (XR §12): assist and personal receiving are OPD-10 (aliases SD-01..03) and XR §2 Senior MUST NOT OWN personal receiving from rank. Deputy R; MIS/HR —; Finance R needed (XR §12).
- **26. `org.hr_request.create`.** XR §12 HR request create: Sup A, Senior A, Manager A, Deputy C route/assigned step, HR A authorized HR workflow, Seller — baseline. SS §22 `submit_hr_request`; MG §15 KEEP; DP §15 `current entry NEEDS VALIDATION`; HRS §16 `current HR create entry conditional`. S11 SAFE (no personnel change).
- **27. `org.hr_request.review`.** XR §12 step review: Sup/Senior/Manager A, Deputy C, HR A authorized HR workflow. SS §22 `review_assigned_hr_request`; MG §15 KEEP (bulk HR review NOT ALLOWED); DP §15 CODE-supported. HR final apply has no capability (OPD-05, SD-04).
- **28. `org.hr_request.read`.** Same basis as org.hr_request.create/review. HR needs the whole pending_hr queue; the implemented filter does not serve that (HR read of the queue is tied to the final-apply workflow, OPD-05).
- **29. `integration.writeback.read`.** Technical diagnostics. XR §3 Configuration/Maintenance is a restricted technical boundary (Admin), not a business role. MIS (run diagnostics, MS §13) and Finance (diagnostics, rollout/maintenance in DV nav) have adjacent needs; no spec assigns write-back verification.

### 6.9 Controlled financial writes and Finance review

| # | Capability | SEL | SUP | SNR | MGR | DEP | MIS | HR | FIN |
|---|---|---|---|---|---|---|---|---|---|
| 30 | `payments.upload.stage` | C · OWN · D44 | D | D | D | X | X | X | X |
| 31 | `payments.batch.submit` | C · OWN (implemented) · D43, SEC | D | D | D | X | X | X | X |
| 32 | `payments.submission.read` | C · OWN (implemented) · PAIR | D | D | D | N | N | N | N |
| 33 | `payments.evidence.read` | X | X | X | X | X | X | X | C · ORGANIZATION (financial domain) · FRS, D44, LIV |
| 34 | `finance.review.read` | X | X | X | X | X | X | X | A · ORGANIZATION · LIV |

**Justification and evidence:**

- **30. `payments.upload.stage`.** Seller: XR §12 invoice issue/evidence A; S12 ACTIVE. Supervisor/Manager/Finance/converter submit is BLOCKED in S12 (OPD-10, OPD-08). Deputy R baseline. Finance must not submit evidence it reviews (W04).
- **31. `payments.batch.submit`.** Seller: XR §12 A; S12 ACTIVE for the owning seller only. Other submitters BLOCKED (S12: OPD-10, OPD-08). SoD with Finance review (W04).
- **32. `payments.submission.read`.** Pairs with payments.batch.submit. Own-scope read of a role that never submits returns nothing: NOT_APPLICABLE.
- **33. `payments.evidence.read`.** Finance: FS §15 `review_payment_evidence` (download protection independent). DV: hide receipt detail from HR and bank detail from MIS/Senior. Seller sees own receipts through payments.submission.read.
- **34. `finance.review.read`.** Finance: FS §15 `view_finance_queue`; FS §6 Review Queue; ARCH §4A payments (review); RE2 maps Finance review queue. MIS: XR §12 `R diagnosis` for stage decisions, but no MIS spec need for the pending queue. Sales: XR §2 MUST NOT OWN Finance review.

### 6.10 Commission

| # | Capability | SEL | SUP | SNR | MGR | DEP | MIS | HR | FIN |
|---|---|---|---|---|---|---|---|---|---|
| 35 | `commission.shadow.read` | X | X | X | X | X | X | X | A · ORGANIZATION · LIV |
| 36 | `commission.policy.read` | D | X | X | X | X | X | D | A · ORGANIZATION · LIV |
| 37 | `commission.own.read` | A · OWN (implemented) | A · OWN (implemented) | A · OWN (implemented) | A · OWN (implemented) | A · OWN (implemented) | N | N | N |

**Justification and evidence:**

- **35. `commission.shadow.read`.** Finance: FS §11-12 runs and entitlement boundary; ARCH §4A commission (run approval). HR: XR §5 `V effective input` is HR's own compensation input, not all recipients' results. Sales: own earnings only (commission.own.read).
- **36. `commission.policy.read`.** Finance: FS §11 rules. HR: FS §12 `HR owns compensation/structure temporal settings`; whether HR reads commission policy depends on OPD-09 routing. Seller: DV Nice to Have `own commission rule match` (own-scope match not implemented).
- **37. `commission.own.read`.** Seller: DV KPI `own wallet balances KEEP`; XR §5 Credit Owner `V own`. Supervisor: G0 §19 Own Wallet. Senior: SS §22 `view_own_wallet`. Manager: MG §15 `view_own_wallet` KEEP. Deputy: DP §15 `view_own_wallet` KEEP. ARCH §4A Seller commission (own earnings, read). MIS: XR §5 Credit Owner `—`. HR/Finance: not evidenced as commission recipients.

### 6.11 Fulfillment work, activation and provider operations

| # | Capability | SEL | SUP | SNR | MGR | DEP | MIS | HR | FIN |
|---|---|---|---|---|---|---|---|---|---|
| 38 | `fulfillment.work.read` | D | D | D | D | D | D | X | D |
| 39 | `fulfillment.work.assign` | D | D | D | D | D | D | X | D |
| 40 | `fulfillment.work.execute` | D | D | D | D | D | D | X | D |
| 41 | `biawin.activation.read` | D | D | D | D | D | D | X | D |
| 42 | `integration.provider_operation.read` | D | D | D | D | D | D | X | D |
| 43 | `integration.provider_operation.reconcile` | D | D | D | D | D | D | X | D |

**Justification and evidence:**

- **38. `fulfillment.work.read`.** S14 §5: per-service supervision; a supervisor role grants nothing by itself; no new human role. Operations/execution legacy roles are UNMAPPED (mapping.go). No eight-role spec assigns fulfillment work.
- **39. `fulfillment.work.assign`.** Same as fulfillment.work.read.
- **40. `fulfillment.work.execute`.** Same as fulfillment.work.read.
- **41. `biawin.activation.read`.** Same as biawin.subscription.read.
- **42. `integration.provider_operation.read`.** Same as biawin.subscription.read; provider reconciliation owner not named in any spec.
- **43. `integration.provider_operation.reconcile`.** Same as biawin.subscription.read. SoD W10: the reconciler must be independent of the activation path; no such check exists.

## 7. Condition keys

| Key | Condition that must be satisfied before the cell could become a grant |
|---|---|
| RS | Server-side row scope for this capability does not exist (the evaluator returns ORGANIZATION for every read). It must be implemented on every data-returning endpoint, including totals, counts, search and phone search, with out-of-scope rows indistinguishable from missing rows (M2 §5), before any grant. |
| FP | Field policy / masking required (PII, compensation, credentials, bank or card detail). Hiding in the UI is not a control (DV §1, G0 §10). |
| M2R1 | M2 proposed record 1 (seller capability eligibility; no decision ID) must be registered by the authorized inventory owner and decided. |
| M2R2 | M2 proposed record 2 (seller case read scope; no decision ID) must be registered and decided; engineering proposal is current holder only (not selected). |
| CUS | Customer-identity read assignment is not evidenced (registry comment cites D-30); a decision on who may read customer identity is needed (see CF-05). |
| MISS | The MIS row/source scope predicate is not defined (MS §14: `row/source scope`); organization-wide MIS reads need confirmation. |
| O01 | OPD-01: unknown or conflicting lineage stays UNKNOWN; logical Case coverage is conditional. |
| O07 | OPD-07: metrics are NOT FINAL and must be labelled; cohort/window/attribution undecided. |
| O10 | OPD-10: assist / delegation paths are not approved; read of evidence for assistance only. |
| DEPV | Deputy entry for HR requests is `NEEDS VALIDATION` (DP §15) and XR §12 rates Deputy `C route/assigned step`. |
| LNK | Scope `linked to the financial item under review` is not implemented; today the read would be organization-wide. |
| FRS | No decision record defines Finance row scope; the registry comment cites D-25, D-26 and D-28, which decide different questions (see CF-04). |
| D43 | D-43, D-28 and OPD-06 remain value-level blockers already enforced by the command. |
| D44 | D-44 evidence retention is open; purge runs in report mode only. |
| SEC | R5-SEC-01: person-level independence between submitter and Finance reviewer cannot be enforced today (W04). |
| PAIR | Only meaningful together with payments.batch.submit for the same role; follows that cell's conditions. |
| LIV | HR / Finance runtime is NOT LIVE VERIFIED: activation requires tests with dedicated real role accounts (AGENTS.md). |

## 8. Technical role `invoice_reviewer` (documented separately; not a ninth business role)

| Item | Fact or proposal |
|---|---|
| Runtime fact | Target role key `invoice_reviewer` exists with an **empty** bundle; legacy label `sn_invoice_reviewer` maps to it as SHADOW evidence only (`mapping.go:37`) |
| Architecture intent | ARCH §4A: "Invoice Reviewer → reporting (Invoice Review query surface)"; ARCH §18: `sn_invoice_reviewer` → invoice review read |
| Proposed cell for `reporting.invoice_review.read` | **DEFERRED_DECISION**: D-23 (visibility, PII, export) and D-24 (product/design owner) are open; D-45 governs audit-write failure (safe default fail closed) |
| Proposed cells for the other 42 capabilities | **PROPOSED_DENY**: no source states another need |
| Not proposed | Treating `invoice_reviewer` as a business role, or folding it into Finance. Whether a Finance person also holds it is a multi-role question (decision package §4) |

## 9. Product actions with no backend capability (requirements, not implemented keys)

The registry has no capability for the actions below. **Nothing here is implemented, and no key here is registered.** Where ARCH §18 already gives an *example* name, it is quoted as an example only; otherwise no name is proposed. Adding any capability needs, in one change: registry entry, catalog migration, OpenAPI enum value, `capabilityFlags` entry, and a navigation CONTRACT destination if navigable (RM-ACCESS-01 §11).

| Req | Product action | Roles needing it (source) | Nearest existing capability and gap | Blocking decisions / gaps |
|---|---|---|---|---|
| MC-01 | Grant, revoke and periodically review role grants | HR `manage_business_access` (HRS §16); Security (D-42 options) | none | D-42, E-7, R5-SEC-01 |
| MC-02 | Account provisioning; credential reset / send | HR `manage_credentials` RESTRICTED (HRS §13, §16) | `identity.user.read` is read only | D-34, R5-SEC-01 |
| MC-03 | Impersonation with purpose, expiry and trail | HR `impersonate_user` RESTRICTED (HRS §13) | none | D-34, R5-SEC-01 |
| MC-04 | HR final apply of transfer / termination with handover | HR (XR §12 `A specific authority`; HRS §9–11); ARCH §18 example name `org.assignment.apply` (not registered) | `org.hr_request.review` is step only; HR actor BLOCKED | OPD-05, SD-04, R5-SEC-01 |
| MC-05 | Workforce profile create / edit; current hierarchy edit | HR (HRS §16) | org reads only | F08, OPD-05 |
| MC-06 | Compensation view / edit | HR (HRS §14; G0 §10 `view_compensation`, `edit_compensation`) | none (no compensation read either) | OPD-09 (input boundary), F08 |
| MC-07 | Workforce import / export; HR audit view | HR (HRS §16) | none | ADR #18 (data handling) |
| MC-08 | Finance stage approve / reject, including bulk | Finance (FS §8, §15); ARCH §18 example name `finance.stage.approve` (not registered) | `finance.review.read` (read only) | OPD-08, D-43, REJECT-VOCAB (R4 gap), NO_DECISION_CAPABILITY (R4 gap), R5-SEC-01 |
| MC-09 | Refund request, approval, execution, confirmation | Finance (FS §9); Sales/after-sales request (XR §12 `request/context`) | none | OPD-04, OPD-08, R5-SEC-01, D-44 |
| MC-10 | Ledger posting, reversal, manual adjustment | Finance (FS §10) | `finance.*.read` (shadow read) | D-41, OPD-06, OPD-09, OPD-08 |
| MC-11 | Commission rule authoring, run review / approval, posting execution, settlement | Finance (FS §11; G0 §10 Rule Author / Run Reviewer / Posting Actor) | `commission.shadow.read`, `commission.policy.read` | OPD-09, D-36, D-41, OPD-08 |
| MC-12 | Rollout / posting-activation flag management | Restricted rollout owner (DV Finance error prevention; FS §11) | none; flags change out of band | E-7 |
| MC-13 | Sensitive export (finance, workforce, reports) as an audited command | All roles `export_*` CONDITIONAL (SS §22, MG §15, DP §15, MS §14, HRS §16, FS §15); ARCH §18 `CreateExport` | none | D-23 (invoice review), ADR #18, D-45 |
| MC-14 | Invoice issuance (incl. pre-invoice) and case claim for invoice | Seller A; Sup/Senior/Manager C (XR §12); ARCH §18 example name `billing.invoice.issue` (not registered) | none | ADR-01, F01, OPD-10 |
| MC-15 | Next payment stage issuance | Seller / converter (legacy) | none | D-32 |
| MC-16 | Allocation, distribution and transfer of case custody | Sup A, Senior/Manager A/C, Deputy C, MIS A governed (XR §12) | `custody.custody.read` (read only) | OPD-10 (alias SD-01), F01, ADR-01 |
| MC-17 | Ordinary return, release, reset | Same tiers (XR §12) | none | ADR-01, F01, OPD-03 |
| MC-18 | Ready conversion: mark ready, ready assignment, own conversion actions | Seller (ready event), Supervisor `A scoped` (XR §12); Senior read only | none (`MarkReady` BLOCKED, S11) | OPD-10 |
| MC-19 | Extra-number request (Seller) and decision (assigned Manager only) | XR §12; MG §12 | none | OPD-10 |
| MC-20 | Archive explorer read and archive revival | Manager (MG §11) | none | OPD-03 (revival NEEDS VALIDATION) |
| MC-21 | Invoice assist: resend link, receipt / prepayment assist, allowed sales correction | Sup/Senior/Manager (MG §15; SS §22 `assist_invoice_action`) | none | OPD-10 |
| MC-22 | Contact outcome value DUPLICATE | Seller | `cases.contact_outcome.record` blocks the value | D-29 / D-31 |
| MC-23 | Customer merge propose / resolve | MIS / named role (D-30 options) | `customers.merge_case.read` (read only) | D-30, ADR #18 |
| MC-24 | Own wallet balances (posted / earned / settled) | Seller and every sales tier (`view_own_wallet`) | `commission.own.read` covers historical commission credits only | D-41, OPD-09 |
| MC-25 | MIS import and batch / row views | MIS (MS §6, §14 `import_source_data`, `view_import_batches`) | none | OPD-01 |
| MC-26 | MIS source assignment, delivery, quick delivery | MIS (MS §14 `assign_source_cases`) | none | OPD-10, F01 |
| MC-27 | MIS return of source cases | MIS (MS §14 CONDITIONAL) | none | F01, OPD-03 |
| MC-28 | MIS data correction and maintenance actions | MIS (MS §13 MAINTENANCE ONLY) | `reporting.mis.read` (diagnosis only) | OPD-01, F01 |
| MC-29 | MIS planning | MIS (MS §12) | none | OPD-07 |
| MC-30 | Row-scoped variants of existing reads (own, direct team, subtree, territory, linked item) | All sales tiers, MIS, Finance (condition RS / LNK) | the 43 reads evaluate ORGANIZATION | M2 proposed record 2; GAP-02 (decision package) |
| MC-31 | Delegation / acting on behalf | Sales tiers (XR §6 skip-level, assist) | none; delegation not modelled (R5 P-SEC §1) | OPD-10, R5-SEC-01 |
| MC-32 | HR request queue for HR (`pending_hr`) | HR (HRS §11) | `org.hr_request.read` returns requested / reviewing only | OPD-05 |
| MC-33 | Deputy exceptions coordination view | Deputy (DP §10 `view_exceptions` ADD) | none | OPD-02 |
| MC-34 | Biawin production activation, wallet charge, membership lifecycle | not assigned to any of the eight roles | `biawin.activation.read` | D-35a … D-35e, D-38 |
| MC-35 | Fulfillment cancel / reopen / submit review | not assigned | none | OPD-04, OPD-08 |
| MC-36 | Operations upgrade payment | not assigned | none | D-33 |

## 10. Major access conflicts and tensions (recorded, not resolved)

| ID | Conflict | Sources in tension | Effect on this proposal |
|---|---|---|---|
| CF-01 | **Scope of D-42.** The RM-B02 brief treats the role-capability matrix and multi-role policy as D-42 work. R5 and M2 restrict D-42 to *who may grant, edit and revoke*; capability eligibility and seller read scope are separate (unnumbered) proposed records, and M2 states D-42 must not be widened | RM-B02 brief vs R5 inventory row D-42; M2 §2, §4 | The decision package keeps D-42's R5 question and options verbatim and lists the other questions as separate proposed records without IDs |
| CF-02 | **Role spec status labels.** Each role spec header says FROZEN PRODUCT TARGET / IMPLEMENTATION DEFERRED. R7 §8 says each role spec labels itself READY FOR REVIEW; no such label exists in `docs/source-of-truth/` at HEAD. RM-E02 uses READY_FOR_REVIEW for the **design** freeze | spec headers vs R7 §8 vs RM-E02 §7 | Product content of the specs is used as product authority; their design status is not relied on |
| CF-03 | **Supervisor freeze.** SS header calls Supervisor Design V1 frozen; RM-E02 marks it FREEZE_UNVERIFIED; G0 §19 says "READY FOR PRODUCT SPEC" and no standalone Supervisor product spec exists. XR §1 adopts A/G/D as the Supervisor contract | SS vs RM-E02 vs G0 §19 vs XR §1 | Supervisor cells rely on XR §2/§5/§12, G0 §19 and DV only |
| CF-04 | **Finance / billing row scope.** The registry comment says billing and payment reads stay unassigned because D-25, D-26 and D-28 are open. Those decisions cover gateway auto-approval, auto-login and partial payment, not read scope. No record decides Finance row scope | `api.go` Slice 7 comment vs R5 inventory rows D-25/D-26/D-28 | Finance billing/payment cells are CONDITIONAL on `FRS`; a separate scope record is proposed (GAP-03) |
| CF-05 | **Customer read assignment.** The registry ties customer reads to D-30, which decides merge authority, not read eligibility | `api.go` Slice 5 comment vs R5 row D-30 | Customer read cells carry `CUS`; GAP-04 proposes a record |
| CF-06 | **Deputy and MIS diagnostics.** RM-E02 maps Deputy diagnostics to `/mis/reconciliation` (`reporting.mis.read`). DV says Deputy should hide daily technical logs; XR §2 lists MIS repair among Deputy's MUST NOT OWN (repair, not read) | RM-E02 §5 vs DV vs XR §2 | Cell left DEFERRED_DECISION; navigation does not prove authority |
| CF-07 | **Position as an eligibility input.** `CreateHRRequest` requires the actor's HR position slug in {supervisor, senior_supervisor, sales_manager, sales_deputy} in addition to the capability (legacy guard port). The target rule is Position ≠ Role ≠ Capability; ARCH §18 allows hierarchy only as an input | S11 vs XR §12 / ARCH §18 | Not a violation (the capability is still required) but a role holder without that position is refused; must be part of the eligibility decision |
| CF-08 | **HR finalizing `pending_review`.** XR §11 says current HR may also finalize `pending_review`; Slice 11 blocks every HR action (final apply has no capability) | XR §11 vs S11 | HR cells for HR request capabilities DEFERRED (OPD-05, SD-04) |
| CF-09 | **Scope gap (fact vs target).** Every role spec requires own / team / subtree / territory scope; every implemented read is organization-wide. Granting any read to a sales tier today would expose the whole organization | `authorization.go:163` vs SS §22, MG §15, DP §15, M2 §5 | 46 CONDITIONAL cells carry `RS`; none can become a grant before server-side scope exists |
| CF-10 | **Architecture capability families with no capability.** ARCH §4A lists Seller billing issue, Supervisor custody allocation and ready assignment, Finance posting and run approval, HR organization (all) and credential policy; none of these writes exists | ARCH §4A vs registry | Listed in §9; ARCH §4A "organization (all)" is read as module ownership, not as one undivided grant (G0 §10 requires HR concepts to be independent) |
| CF-11 | **Own wallet vs commission credits.** Sales specs ask for own wallet balances; the only own-scope capability reads historical commission credits | specs vs registry | `commission.own.read` ALLOW covers part of the need; MC-24 records the rest |

## 11. What this document does not change

- No file under `internal/`, `api/`, `web/`, `cmd/` or any migration.
- No role bundle, grant, flag default, rollout mode, account, credential or configuration.
- No status in `docs/governance/r6-decision-tracker.json` or `docs/remediation/r5-decision-inventory.md`.
- No source-of-truth document.
- No decision ID is created; proposed records carry descriptive labels only.
