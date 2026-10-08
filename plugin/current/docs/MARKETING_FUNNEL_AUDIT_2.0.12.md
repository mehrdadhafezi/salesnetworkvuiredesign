# Marketing funnel audit — 2.0.12

## Outcome

Every valid click on the public “continue application” submit button creates or resumes one browser-owned Marketing Dot submission before any gateway redirect. A new page attempt creates a new submission even when the phone number already exists.

## Audited funnel

| Stage | Event | Stored result |
| --- | --- | --- |
| 10 | `continue_clicked` | Application started |
| 20 | `lead_saved` | Independent marketing lead persisted |
| 30 | `invoice_created` / `invoice_reused` | Source pre-invoice available |
| 40 | `confirmation_opened` | Confirmation popup shown |
| 50 | `confirmation_accepted` / `otp_requested` | Customer continued |
| 60 | `otp_verified` | Mobile verified when OTP is enabled |
| 70 | `gateway_requested` | Gateway request initiated |
| 80 | `gateway_started` | Customer redirected to gateway |
| 90 | `pending_finance` | Verified payment awaiting finance |
| 100 | `payment_paid` / `validation_gifted` | Flow completed |

Cancellation, gateway failure, OTP failure, rate limiting, finance rejection and invoice creation failure are immutable events. They do not delete the lead or its earlier history.

## Reporting

- Campaign analytics includes applications started, gateway entries and abandoned/failed applications.
- Marketing lead CSV includes Visitor ID, Session ID, First Touch UTM, Last Touch UTM, current stage from 0–100, every stage timestamp and the complete ordered event history.
- Campaign attribution falls back to the UTM values posted with the form when a cached page, blocked beacon or unavailable cookie prevents normal session attribution.
- Browser tracking retries transient non-2xx responses and sends a page-exit fallback beacon.

## Rate limiting

Public form and OTP limits use the browser-scoped campaign client key in addition to phone and IP. This prevents unrelated visitors behind one CDN/NAT address from exhausting one shared bucket while retaining per-phone abuse controls.

## QA matrix

1. Open a UTM link in a logged-out private window: Session and Unique become 1.
2. Submit a valid form and stop at the popup: Lead, Registration and Application Started become 1; the CSV stage is 40.
3. Close the popup: the lead remains and the event is `popup_cancelled`.
4. With OTP enabled, test send, wrong code, expiry, resend and successful verification; every result appears in history.
5. Continue to the gateway and cancel: stage remains 80 and status becomes `gateway_cancelled`.
6. Simulate gateway request failure: status becomes `gateway_failed` and the lead remains.
7. Complete direct payment: status becomes `payment_paid`, stage 100 and campaign sale metrics increase.
8. Complete finance-reviewed payment: status becomes `pending_finance`, then `payment_paid` after approval.
9. Reject finance review: status becomes `payment_rejected` without deleting the lead.
10. Reload the landing and submit the same phone again: a new marketing submission and a second application-start conversion are created.
