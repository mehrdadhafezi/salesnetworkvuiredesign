# 2.0.137 — Restore the payment-completion queue for supervisors

Root cause: the supervisor repeat-actions AJAX route rendered the Dot conversion queue. Dot deliberately excludes invoice_payment_completion items. Payment ownership could therefore move to a supervisor while the only working payment-stage renderer remained in the seller panel.

Changes:
- Dedicated «نیاز به اقدام مجدد / تکمیل پرداخت» tabs for supervisor and senior supervisor, using the existing owner-aware paid-referral renderer and authorization.
- Existing needs_payment_assignment, payment_assigned and payment_stage_issued records appear through the existing distribution query; no financial or ownership migration is performed.
- Owner can select self/eligible assignee, issue the next stage on the same invoice, then open that invoice to register its receipt.
- Non-owner message explains where the current owner should follow up.
- New queues refresh whenever their tab is opened.

Validation: PHP 8.0 syntax parse and JavaScript syntax checks passed. Fourteen assertions against actual PHP-rendered HTML under PHP 8.4 WebAssembly cover owner assignment, next-stage form, 25m/5m/20m amounts, same-invoice receipt link, non-owner restrictions and AJAX routes. Fixture uses actor IDs 35654/35706 and invoice code 57720489 but does not assert the live database matches this fixture. No live server or authenticated payment test performed.

Install and open the supervisor panel as the current owner, then the new payment-completion tab. If assignment is pending select «ارجاع به خودم», or select an eligible seller. Issue the remaining/partial stage once, then use the same-invoice receipt link. All prior changes retained; the previously disclosed gateway report pagination limitation remains.
