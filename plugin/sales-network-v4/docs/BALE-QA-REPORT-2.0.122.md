# Bale / Safir QA Report — 2.0.122

مبنای Merge:
- Base: 2.0.117
- Bale reference: 2.0.119 (Production-tested Bale/Safir integration + seller checkbox)
- Target: 2.0.121
- Output: 2.0.122

## نتایج
- PHP lint: 27/27 PASS
- JavaScript syntax: PASS (`public-seller.js`, `public-shell.js`, `public.js`)
- Sales Network static audit: PASS
- Git diff whitespace check: PASS
- Conflict marker scan: PASS
- Outbound CRM -> Gateway HMAC contract: PASS
- Inbound Gateway -> CRM HMAC contract: PASS
- Replay protection: PASS
- Deterministic request_id / idempotency: PASS

## حفظ قابلیت‌های نسخه مقصد
- 2.0.118 HR manual invoice access/SMS handling: preserved
- 2.0.119 HR access save: preserved
- 2.0.120 SMS provider result/error reporting: preserved
- 2.0.121 standalone Product* sale visibility/activation behavior: preserved
- `includes/class-sn-sms.php` is the 2.0.121 implementation, not rolled back.

## Bale coverage
- seller invoice create + explicit send_bale checkbox
- invoice stage / payment-link resend
- customer portal invite + OTP
- Dot messages + marketing OTP
- operations upgrade invoice
- operations wallet charged secure message
- HR login credentials + password reset secure message
- invoice payment completed
- payment reward
- admin Bale test
- account link/status REST endpoints
- delivery/account tables and admin settings

## Production note
Live Safir/Bot credentials are intentionally not embedded in the plugin. The external Gateway keeps Bot/Safir secrets; CRM keeps only Gateway URL + HMAC shared secret (or wp-config constants).
