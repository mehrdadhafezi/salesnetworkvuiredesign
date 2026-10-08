# 2.0.124 — Finance tabs and deposit modal

Payment review tabs, in order: pending review, deposit details, uploaded receipt,
gateway payments, financially approved, financially rejected, all invoices.

Deposit details lists manually entered deposits without a receipt file. Uploaded
receipt lists records with an actual receipt URL/file. These two source tabs
retain reviewed records; approval/rejection tabs classify the review outcome.
All invoices includes pre-invoices, cancelled and archived records as well.
Search stays inside the selected tab; use all invoices for a global search.

Reviewed deposits, including partial payments with retained receipt/manual data,
leave the pending queue. A newly submitted pending stage can re-enter the queue.
Financially approved includes partial payments with recorded approval metadata.
Existing single/bulk approval and rejection callbacks reload the active list.

The receipt/deposit modal now uses a shrinking flex card, fixed header and one
scrollable body, with viewport-relative height limits and no conflicting grid
minimum height. Reopening the modal resets the body scroll position.

Validation: 11 synthetic SQL predicate scenarios passed using SQLite; tab-scoped
search preservation checked; changed JavaScript passed node --check.
WordPress/MySQL integration and PHP lint were not run (PHP/site unavailable).
A Playwright visual test was prepared but could not run: browser binary absent.
Verify modal scrolling on desktop/mobile and an approve/reject/resubmit cycle
on a staging WordPress site before production rollout.
