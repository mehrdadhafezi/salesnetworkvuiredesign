# Sales Deputy — Targeted Live Reverification

Date: 2026-10-06, Asia/Tehran. Environment: https://crm.maximumclub.ir/.

Curated observations, not a full DOM export or execution test. Customer identity, phone numbers, credentials and payment links omitted.

## Access / version

The existing Admin session was available. Plugins page showed active Sales Network **2.0.123** once. HR workforce “مشاهده پنل” for Deputy #2 entered `/crm-sales-deputy/`; header confirmed Deputy #2. The existing return control restored Admin after inspection. Version equality does not prove identical installed source bytes.

## Inventory / evidence

| Area | Observation | Limit |
|---|---|---|
| Navigation | Nine destinations: overview, hierarchy, Managers, Seniors, Supervisors, Sellers, distribution, invoices, wallet; shared report center separate | LIVE VERIFIED presence |
| Overview | Visible Sellers 1, active 1, inactive 0; leads 0, invoices 1, pre-invoices 0, paid/approved 0, rejected/returned 0; total invoice amount 15,000 toman, paid 0 | LIVE VERIFIED display; not final KPI definitions |
| Filters | Date from/to, Manager, Senior, free-text invoice status | LIVE VERIFIED controls only |
| Hierarchy | Manager #3, Senior #4; direct Sellers of Manager 0, total descendant Sellers 1 | LIVE VERIFIED single branch, no cross-Manager runtime dataset |
| Performance | Manager #3 / Senior #4 / Supervisor #5 / Seller #6 rows each show the same descendant sample: Sellers 1, active 1, leads 0, invoice 1, paid/rejected 0, total 15,000 and paid 0 toman | Read of rendered tables; no summing across ancestor levels |
| Distribution | Own balance 0, quick allocation 0, active cases 0; four eligible recipient controls: Manager #3, Senior #4, Supervisor #5, Seller #6; count/note/source/level/return controls | LIVE VERIFIED controls; no selection, preview execution, apply or return |
| Invoices | One pre-invoice, 15,000 toman; columns code/Seller/state/amount/payment status/date; no action buttons or links in this table | LIVE VERIFIED read-only sample; no inference of all possible backend rights |
| F02 | Pre-invoice count 0 versus a pre-invoice row in invoice table | LIVE VERIFIED discrepancy; accepted source predicate omits pre_invoice |
| HR / archives / extra-number | No destinations or controls for these queues in Deputy navigation/panel | LIVE VERIFIED absence in inspected panel; shared/backend reachability not globally disproven |
| Reports | Shared report center with period/current-snapshot and role/person selectors | LIVE VERIFIED shell; no report/export executed |
| Wallet | Destination present; intentionally not opened | Presence only; F07/pure-read behavior not retested |

F03 remains CODE VERIFIED: Deputy lead counter uses sn_leads only. V4 sources were not re-audited. F04 remains the accepted Manager discrepancy, freshly documented in manager-reverification-2026-10-06.md; Deputy sample invoice total and list both showed 1, so F04 is not claimed newly reproduced in Deputy.

## Safety / confidence

Only navigation, existing role impersonation/return and reads were performed. No allocation, assignment, return, invoice/payment action, HR decision, archive decision, export, import, permission or data mutation was executed. No DB before/after snapshot exists; internal renderer effects are not certified absent. Role-specific writes, negative enforcement, races, multi-Manager scale and historical-as-of completeness remain NOT VERIFIED.

Targeted source inspection covered only Deputy renderer/scope, metric predicates and shared HR eligibility. Archive/extra-number and return boundaries reuse accepted code evidence and the Manager Spec; no unrelated role audit was restarted.

## Local validation

The final specification has 20 ordered top-level sections and no broken local document links. All 248 baseline plugin file hashes remained unchanged. Input ZIP SHA256 remained `ae785ad4b27224e2ec015477188cf54409a251f3ad362f3a4d42181e39b03802`. Only this evidence note and the Deputy specification were created; implementation files were not edited.
