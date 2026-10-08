# Campaign Tracking & Partner Analytics — 2.0.7

## Scope

- First-party UTM capture on public landing pages.
- Persistent anonymous `visitor_id` and 30-minute `session_id` cookies.
- First-touch and last-touch attribution to registrations, CRM leads, Marketing Dot submissions and paid invoices.
- Aggregate CRM dashboard with date and UTM filters.
- Campaign budget/cost, CPL and CPA.
- Restricted `sn_campaign_partner` role and frontend portal.
- Read-only, token-authenticated partner report API.

## Data model

- `sn_campaigns`: canonical UTM combinations, budget and actual campaign cost.
- `sn_campaign_sessions`: one row per 30-minute visit session.
- `sn_campaign_events`: deduplicated landing/page-view events.
- `sn_campaign_attributions`: subject-to-first/last-touch bridge.
- `sn_campaign_conversions`: registration, lead, approved lead, active customer and sale milestones.
- `sn_campaign_partner_campaigns`: partner allowlist.
- `sn_campaign_partner_tokens`: SHA-256 token hashes only; raw tokens are never stored.

Phone/email identity bridges use a WordPress-salted HMAC and are not exposed in the partner portal or API. Raw IP storage is disabled by default.

## Metric definitions

- Session: distinct campaign session row in the selected period.
- Unique Visitor: distinct anonymous visitor ID.
- Registration: WordPress registration or Marketing Dot registration submission.
- Conversion Rate: registrations / sessions.
- Lead: attributed CRM or Marketing Dot lead.
- Approved Lead: an attributed lead with an approved/qualified status, or a lead linked to a fully paid invoice.
- Active Customer: distinct attributed identity with a fully paid CRM invoice.
- Sales: sum of the final paid invoice amounts.
- CPL: Campaign Cost / Leads.
- CPA: Campaign Cost / Active Customers.

When actual Campaign Cost is zero/not entered, Budget is used as the effective cost for CPL and CPA.

Both the dashboard and portal support First Touch and Last Touch models. Last Touch is the default.

## 2.0.11 repeat-lead reliability update

- Every new Marketing Dot form attempt creates an independent marketing submission, even when the same phone submitted before.
- The AJAX snapshot and final gateway POST update the same owned submission; they do not create two rows for one attempt.
- UTM values are posted with the form and captured server-side before the lead conversion, so cached landing pages or a failed tracking beacon cannot detach the lead from its campaign.
- Standard `?utm_...#form` links and legacy `#form?utm_...` links are both parsed in the browser.
- Campaign Lead/Registration metrics count submissions instead of deduplicating by phone. Paid Customer remains a unique-person metric and Successful Payments counts invoices.
- Marketing form rate limits use versioned per-phone/per-browser-network buckets with a high IP safety ceiling, preventing a shared CDN/NAT address from blocking normal campaign traffic.

## Partner access

1. Create a WordPress user with role `Campaign Partner / پارتنر کمپین`.
2. Open CRM > تحلیل کمپین‌ها.
3. Assign only the campaigns that user may see.
4. Generate a token and copy it immediately; only its prefix and hash remain stored.

The partner portal is created at `/campaign-partner/`. Partner users are redirected away from `wp-admin` and cannot access CRM settings or customer-level data.

## API

Friendly endpoint:

`GET /api/campaign/report`

WordPress REST equivalent:

`GET /wp-json/sn-crm/v1/campaign/report`

Authentication header:

`Authorization: Bearer sncp_...`

Tokens in query strings are deliberately rejected. Supported filters: `date_from`, `date_to`, `source`, `medium`, `campaign`, `content`, `term`, `attribution_model`, `page`, and `per_page` (maximum 200).

Every API query is intersected with the token owner's campaign assignments before aggregation.
