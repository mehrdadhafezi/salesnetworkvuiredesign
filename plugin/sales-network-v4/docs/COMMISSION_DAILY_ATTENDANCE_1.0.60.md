# v1.0.60 - Daily attendance commission formula

Adds a commission formula for fixed-salary daily qualification plus invoice percentage.

Business rule:
- Monthly base salary: 22,000,000 Toman
- Working days: 26
- Daily qualified amount: 846,153.846154 Toman
- Qualification window:
  - first issued invoice of the seller/day must be at or before 10:30
  - last issued invoice of the seller/day must be at or after 15:00
- Invoice percentage component can be set, e.g. 3%, and is still calculated per approved/paid invoice.

Implementation:
- New commission rule type: `daily_attendance_plus_percent`
- New columns on `sn_commission_rules`:
  - `calculation_type`
  - `daily_base_amount`
  - `attendance_first_before`
  - `attendance_last_after`
- The daily base is added once per seller per day as a synthetic commission item.
- The daily rule's `rate_percent` is calculated per invoice, while `daily_base_amount` is calculated once per qualified day.
- Regular `invoice_percent_fixed` rules are unchanged.

Recommended rule:
- Title: حقوق روزانه مشروط + ۳٪ فروش تاییدشده
- Type: حقوق روزانه مشروط + درصد فاکتور
- Position: فروشنده
- Invoice status: paid,approved
- Rate percent: 3
- Daily amount: 846153.846154
- First invoice cutoff: 10:30
- Last invoice cutoff: 15:00
