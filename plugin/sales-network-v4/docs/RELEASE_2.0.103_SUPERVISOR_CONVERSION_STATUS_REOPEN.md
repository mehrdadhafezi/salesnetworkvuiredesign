# Release 2.0.103 — Supervisor conversion status reopen

## Urgent fix

In Supervisor → «تبدیل‌های من», a case marked «انصراف» (`customer_declined`) was previously treated as irreversible by the converter update endpoint.

### Changed
- Removed the backend guard that blocked changing a declined case to another contact result.
- A declined case can now be changed to:
  - جواب نداده (`no_answer`)
  - تماس مجدد (`follow_up`)
  - ارسال لینک پرداخت (`payment_link`)
  - or remain انصراف (`customer_declined`)
- When reopening a declined case, current decline metadata (`converter_decline_reason`, `converter_declined_at`) is cleared so current state is consistent.
- Historical decline activity remains preserved in the case log.
- Completed or operationally archived cases remain protected and cannot be reopened by this change.

No schema migration is required.
