# MIS — Targeted Live Reverification

Date: 2026-10-06, Asia/Tehran. Environment: https://crm.maximumclub.ir/.

Curated observations, not raw DOM or a write test. Customer identifiers, phone, credentials and links omitted.

## Access and inventory

Active Sales Network **2.0.123** was observed once. Existing Admin session → HR workforce “مشاهده پنل” for MIS #7 → `/crm-mis/`. Header confirmed MIS #7. Existing return control restored Admin after inspection. Matching version does not establish installed-source byte equality.

Eleven destinations: Today, Upload, Assignment/Return, Delivery to Sales, Number Report, Batches, Quick Tool, Distribution Planning, Controlled Lead Creation, Logs/Diagnostics, MIS Reports.

## Current observations

| Area | Observation | Limit |
|---|---|---|
| Today | Valid rows1, waiting assignment1, with Manager0, ready delivery0, with Sales1, returnable1. Per-batch waiting0, with Sales1, returnable1 | LIVE VERIFIED displays; measures overlap and do not form one partition |
| Return safety | Same source population has a hard-linked invoice in report, yet Today says one item has not become lead/invoice and is returnable | LIVE VERIFIED inconsistency; return not executed; source F01 remains accepted code evidence |
| Latest upload | Today labels previous quick-assignment operation as latest upload. Import view shows quick assignment summary1 but import counters0 | LIVE VERIFIED misleading operation/result context; not a new import run |
| Import | Warning: rows table missing or stale health report; “safe rebuild” control. Existing source/report data remains readable | LIVE VERIFIED contradiction; cause missing table/stale health not resolved; rebuild/import not executed |
| Batch maintenance | Create batch, cleanup deleted/duplicate traces, delete source file, delete entire batch controls present | Presence only; no maintenance action |
| Planning | Rule draft/manual/equal split; plan preview/save controls; text says no live Sales rows created. No existing rule/plan. Manager #3 selectable; text says no Seller in scope | LIVE VERIFIED form/text; actual planning correctness and candidate discrepancy NOT VERIFIED |
| Controlled materialization | Approved plan only; APPROVE/APPLY; dry-run/apply controls; text says no SMS, no updating/deleting existing leads | LIVE VERIFIED text/controls; handlers CODE VERIFIED only |
| MIS report | Opened tab; report auto-built without clicking build/export. One unique MIS row, Manager receive history0, Seller delivery history1, preinvoice history1, hard-linked invoice1, approved/completed0; current Seller #6 | LIVE VERIFIED one report sample |
| Financial report measures | Unique invoice1, amount15,000, confirmed paid0, remaining15,000, completed0; independent invoice/payment grouping | LIVE VERIFIED read; currency report text refers to stored CRM unit |
| Time semantics | UI explicitly says time filter selects cases with an event in range, statuses are current during gathering; gathering start/end shown. Current HR hierarchy and actual receive history shown separately | LIVE VERIFIED text; full historical reconstruction NOT VERIFIED |
| Export | “XLSX this report” control | Presence only; no export/download |

## Safety / evidence limits

Only role switching, tab navigation and read-only report observation. No import, assignment, return, repair, cleanup, deletion, batch/plan mutation, approval, live lead creation or export. Report opening auto-created a technical report job/snapshot; this is not a DB-pure read and is not claimed as zero internal writes. No business mutation was intentionally executed.

F02/F03/F04 reuse accepted evidence and lower-role specs; lower roles were not re-audited. F09 wording is explicit in current report; Event Range + current gathered state still is not Historical As-Of. Write enforcement, large-batch outcomes, negative scope tests and concurrency remain NOT VERIFIED.

## Local artifact validation

The specification has 20 ordered top-level sections and no broken local document links. All 248 baseline plugin file hashes remained unchanged. Input ZIP SHA256 remained `ae785ad4b27224e2ec015477188cf54409a251f3ad362f3a4d42181e39b03802`. Only the MIS specification and this evidence note were created in this stage; no implementation file was edited.
