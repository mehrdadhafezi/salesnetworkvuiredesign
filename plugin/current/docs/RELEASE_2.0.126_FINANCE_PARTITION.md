# 2.0.126 — Non-overlapping finance tabs

Supersedes the overlapping source tabs introduced in 2.0.124.
One SQL CASE assigns each eligible invoice to exactly one tab, in priority order:
rejected, financially approved, successful gateway payment, pending uploaded
receipt, pending manual deposit, remaining pending review.

The all tab uses the union of these six buckets, not the full CRM invoice table.
Pre-invoices and other records outside these buckets remain stored unchanged.
Counts are collected in a single grouped query; the all badge is their sum.
The overview queue KPI still includes all three pending buckets.
Search stays inside the selected bucket. Badges represent full bucket totals.

Validation: 11 SQL fixture cases passed, including reviewed receipts, gateway
approval, partial approval, resubmission and excluded pre-invoices/cancellations.
All 15 pairs of tabs had zero overlap; sum of counts equalled the all-list count.
JavaScript syntax passed. Live WordPress/MySQL and PHP lint unavailable.
