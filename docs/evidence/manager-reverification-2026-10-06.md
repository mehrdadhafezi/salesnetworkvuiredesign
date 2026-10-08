# Sales Manager — Targeted Live Reverification

Date: 2026-10-06, Asia/Tehran. Environment: https://crm.maximumclub.ir/

Curated browser observations; not a raw DOM export, write test or database audit. Customer identity, phone numbers, authentication material and payment links are omitted.

## Access and version

The environment initially required login. The user authenticated Admin. The installed plugins page showed **active Sales Network 2.0.123**. The existing HR workforce “مشاهده پنل” control for Manager #3 entered `/sales-manager-panel/`; the panel header confirmed Manager #3. After targeted inspection the existing “بازگشت به پنل HR” control restored Admin #1. Version equality does not establish byte-for-byte equality with the audited ZIP.

## Observations

| Area | Current observation | Verification limit |
|---|---|---|
| Inventory | Nine Manager destinations: overview, distribution, operational report, invoices, customer behavior, archives, extra-number requests, HR requests, wallet. Shared report center and `/crm-reports/` link also present | LIVE VERIFIED presence; not ten independent Manager tabs |
| Overview | Own balance 0, assigned by Manager 0; User #3 presented as a tile | LIVE VERIFIED display; not all-source workload |
| Distribution | Balance 0, quick allocation 0, active cases 0; three recipients: Senior #4, Supervisor #5, Seller #6; count, note, source/case, level and return controls | LIVE VERIFIED controls only; no selection or apply |
| Operational report | One result; legacy leads 0, V4 distribution 1, scoped invoices 0; text bounds each request to 60–120 rows | LIVE VERIFIED F04 display; cause and total coverage not established |
| Invoice tab | One pre-invoice; invoice amount 15,000 toman; paid/approved count 0 and confirmed-sales amount 0. Actions: resend payment link, copy link, financial status/history | LIVE VERIFIED presence; none of these actions executed |
| F04 | Report invoice metric 0 versus invoice tab count 1 in the same Manager session | LIVE VERIFIED discrepancy; filters/cohort/cache/query/build cause NOT VERIFIED |
| Customer behavior — changed from old F05 | List now loads one invoice, search/paging/details controls. Read-only details modal opened and showed two issuance-related timeline entries. List stated 0 events | LIVE VERIFIED list and detail read. Distinct event populations may explain counts; defect NOT established |
| Customer loading state | “در حال بارگذاری...” remained in the loaded section over successive reads. After closing details, the loading element had computed display:flex, visibility:visible and nonzero rectangle | LIVE VERIFIED residual loading presentation; binder failure/root cause NOT VERIFIED. Original permanently unrendered placeholder no longer reproduces |
| Archives — changed dataset | No-answer 0, assessment unpaid 1, subscription/product/product* unpaid 0. Assessment facet read: total 15,000, paid 0, remaining 15,000 toman, due/archive dates and owner context | LIVE VERIFIED one archived record. No mutation/revival control observed in this view |
| Archive semantics | No-answer explains three attempts plus three days idle. Assessment explains three-day incomplete active payment link and automatic resolution on qualifying late payment | LIVE VERIFIED text; successful automatic transition NOT LIVE VERIFIED |
| Extra-number requests | Empty table: Seller, phone, name, note, status, time, action | LIVE VERIFIED shell only; review behavior remains CODE VERIFIED / NOT LIVE VERIFIED |
| HR | Transfer/termination form, scoped Seller #6, destination Supervisor #5, reason, empty request table; text reserves final application for HR | LIVE VERIFIED controls; submit/review/final apply not executed |
| Shared reports | Shared center and time/role/person selectors present; current snapshot versus period mode | LIVE VERIFIED shell; report execution and export not repeated |
| Wallet | Navigation present. Wallet content was already rendered in the initial DOM without selecting its tab | Presence only. No intentional wallet visit; no inference of pure-read or posting success |

## Safety and evidence scope

Only page/tab navigation, existing impersonation/return and customer-history detail read were performed. No distribution, return, archive decision, invoice/payment/receipt mutation, SMS/link send, extra-number decision, HR request/review, import, export, permission edit or setting save was executed. No database snapshot exists, so internal renderer side effects are not certified absent. F07 remains governed by Gate 0.

Source evidence comes from the accepted local ZIP/source and appendices. Current runtime write enforcement, concurrency, multi-team scale, historical-as-of completeness and all-source identity resolution remain NOT VERIFIED. Old F05 evidence remains historical, superseded only for the specific loading observation above.

## Local artifact validation

The final specification has 20 ordered top-level sections and no broken local document links. All 248 plugin files in the existing workspace baseline retained their hashes. The input ZIP SHA256 remained `ae785ad4b27224e2ec015477188cf54409a251f3ad362f3a4d42181e39b03802`. Only the specification and this evidence note were created in this stage; no implementation files were edited.
