# Senior Supervisor — Live Reverification Record

Date: 2026-10-05, Asia/Tehran. Environment: https://crm.maximumclub.ir/

This is a curated record of browser observations in this session, not a raw DOM export or an execution test. Customer names, phone numbers and authentication material are omitted.

## Access and version

The user supplied an authenticated Supervisor session. Header showed Supervisor #5. The existing “بازگشت به پنل HR” control restored Admin #1. The HR workforce table showed Senior Supervisor #4; its “مشاهده پنل” control entered `/crm-senior-supervisor/`. Header confirmed Senior Supervisor #4. After inspection, the existing return control restored Admin #1. `/wp-admin/plugins.php` showed active Sales Network version **2.0.123**. No plugin actions were clicked. Matching version numbers do not prove identical installed source bytes.

## Inventory and observations

| View | Observation | Evidence type |
|---|---|---|
| Navigation | 11 destinations: overview, team, distribution, sellers, invoices, ready conversion, converter stats, my conversions, HR requests, leads, wallet; report center separate | LIVE VERIFIED |
| Overview | sellers 1, active 1, inactive 0, leads 0, invoices 1, pre-invoices 0, paid/approved 0, rejected 0, invoice amount 15,000 toman, paid 0 | LIVE VERIFIED |
| Team | direct Seller 0, indirect Seller 1, direct Supervisor 1; Seller #6 under Supervisor #5 | LIVE VERIFIED |
| Distribution | own balance 0, quick allocation 0, recipients 2 (Supervisor #5 / Seller #6), active cases 0; count/note/source/level controls; allocation disabled; return count field 50 | LIVE VERIFIED controls only |
| Seller performance | one active Seller, legacy lead count 0, invoices 1, pre-invoices 0, paid 0, total 15,000 toman | LIVE VERIFIED |
| Invoices | one pre-invoice; stage 1, due 15,000, paid 0, remaining 15,000 toman; receipt registration/upload, resend, copy, prepayment edit and history controls | LIVE VERIFIED controls/data only |
| Financial history | read-only dialog opened; pre-invoice and total/paid/remaining confirmed; no review history for this sample | LIVE VERIFIED read |
| Ready conversion | explicit readonly supervision message; assignment/change of responsibility belongs to direct Supervisor; empty case queue | LIVE VERIFIED |
| Converter stats | converters 0, assigned 0, completed 0, amount 0, conversion rate displayed 0%; empty dataset | LIVE VERIFIED display; not evidence of poor performance |
| Personal conversions | own queue empty; text states only self-assigned cases are displayed | LIVE VERIFIED; actual receiving eligibility NOT VERIFIED |
| HR requests | transfer/termination form; subject Seller #6; destination Supervisor #5; reason field; requests empty | LIVE VERIFIED controls; create/review NOT VERIFIED |
| Leads | legacy view summary/latest scoped leads empty, date/Seller/status/code filters | LIVE VERIFIED display; legacy source definition CODE VERIFIED from accepted ZIP |
| Wallet | navigation present; initial role landing unexpectedly displayed empty wallet histories before any wallet-tab click | LIVE VERIFIED presence only; no intentional revisit, no posting verification |
| Shared reports | ten report entries and time/role/person selectors; invoices_register selected, count 1, total 15,000 toman, pre_invoice row and hierarchy columns | LIVE VERIFIED one report only |

F02 is still observable: pre-invoice row exists, overview/performance pre-invoice count is 0. The missing predicate was verified in the accepted source audit; installed source bytes have not been extracted. F03 legacy-only code remains source evidence; V4 workload in MIS/Seller was not rechecked today.

## Limits and safety

No assignment, return, invoice edit, receipt submission, payment, HR approval, import, export, SMS/link send, permission edit, or setting save was executed. The history dialog and invoice report were read. Role switching used the existing product mechanism. Page renderers may have internal technical or conditional business effects; no database before/after snapshot exists, so zero internal writes are not asserted. Read purity remains a target invariant.

Runtime write success, negative authorization enforcement, concurrent return/invoice safety, real multi-team data, historical-as-of correctness, exports and personal conversion assignment remain NOT VERIFIED.

Local validation: 248 plugin files matched the existing baseline hashes; input ZIP SHA256 remained `ae785ad4b27224e2ec015477188cf54409a251f3ad362f3a4d42181e39b03802`. The specification has 30 ordered sections and no broken local document links at validation time. Only documentation was created/edited in this stage.
