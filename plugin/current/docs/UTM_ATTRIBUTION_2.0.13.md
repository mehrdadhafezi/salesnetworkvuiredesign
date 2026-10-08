# UTM attribution recovery — 2.0.13

## Goal

Every Marketing Dot submission is persisted before payment and linked to the
campaign that brought the browser to the form. Repeated submissions remain
independent leads.

## Persian date filters

The campaign report stores and queries timestamps in the Gregorian database
format. Its admin and partner inputs are Jalali dates, so the report converts
both Jalali (`1405/05/11` or `1405-05-11`) and ISO Gregorian (`2026-08-02`)
inputs to the same Gregorian SQL boundaries. This prevents a localized
`wp_date()` filter from making a valid current-period report query the year
1405 Gregorian and return zero metrics.

## Attribution inputs

The server resolves each UTM field using the following resilient sources:

1. Same-site HTTP referer query string (authoritative for normal campaign URLs).
2. Browser landing URL, including legacy `#form?utm_...` fragments.
3. Explicit `utm_*` form fields.
4. Existing valid campaign session when the submission contains no fresh UTM.

This order prevents a cached page from attributing a lead using stale hidden
fields while still supporting fragment-style legacy links.

## Durable lead snapshot

`sn_dot_marketing_leads` now stores the landing URL, all five UTM values,
attribution ID, attribution status and attribution time. A failed link is marked
`pending` and retried on the next funnel event or campaign report request.

## Expected test

For a new browser attempt opened with:

`?utm_source=jaryan&utm_medium=cpa&utm_campaign=social&utm_content=35935&utm_term=p-9s-15-0.5h#form`

clicking Continue must produce one Marketing Dot lead with status `attributed`.
The campaign report must increment Session, Unique, Registration, Lead and Start
Request for the same five-part UTM row. Cancelling payment keeps those counts and
increments the relevant abandonment stage; a successful payment later increments
the payment and revenue metrics.

## Audit output

Both Marketing Dot CSV exports expose attribution status, attribution ID,
landing URL and UTM values so a missing connection is visible without database
access.
