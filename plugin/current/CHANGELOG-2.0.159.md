# 2.0.159 — Invoice review status filters

- Fixed gateway-paid filtering: this version's verified gateway callback saves `approved`, not `paid`. Match online/gateway/Zibal/Zarinpal/AsanPardakht invoices with current `paid`, `approved`, or `partial_paid` status.
- Gateway-paid includes confirmed partial payments, not necessarily fully settled invoices. Added explanatory panel text; approved filter label now explicitly includes automatic gateway approval.
- Map next-payment filtering to `partial_paid` + `awaiting_assignment` and stage-issued filtering to a later active stage (`current_payment_stage > 1`, positive due, `awaiting_payment`).
- Preserve the intentional `pre_invoice` + `pending_payment` combination when filtering pending stage payments.
- Prefer the current main status, falling back to invoice/payment status only when empty, matching the CRM main-status display. Conflicting stale secondary values no longer put completed/rejected/cancelled invoices in unrelated status results.
- List, count/amount summary and CSV export retain the shared filter builder. Stored financial data, gateway callbacks and permissions are unchanged.

Validation: `python3 tools/test-invoice-review-filters.py` passes against 23 synthetic invoice fixtures covering 14 status filters, aggregate consistency and method intersection. Tests execute the SQL literals extracted from the PHP source using SQLite. PHP CLI and a live WordPress/MySQL installation were unavailable; PHP runtime, site UI and real customer data have not been tested.
