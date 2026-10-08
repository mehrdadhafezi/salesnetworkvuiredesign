# Sales Network 2.0.87 — No-answer tab routing hotfix

## Problem

The seller panel saved the stage-one outcome successfully but filtered the status tabs by the Persian display label. The shared seller-flow state uses canonical slugs such as `no_answer`. On mixed legacy/MIS data this could leave a row saved as no-answer while the `جواب نداده` tab failed to return it.

## Fix

- Seller stage-one filter buttons now submit canonical slugs (`no_answer`, `callback`, `not_purchased`, `pre_invoice`).
- The server accepts both old Persian-label requests and canonical slugs, then normalizes them through `SN_Seller_Flow::status_slug()`.
- Stage-one filtering uses `seller_flow_status`, not the presentation label in `lead_status`.
- After a successful status save the UI switches/reloads using the canonical workflow status returned by the server.
- Legacy custom filters keep their old exact-label fallback.
- No database migration or destructive rewrite is included.

## Cross-panel audit

The other no-answer paths were checked and already use canonical status codes:

- Converter / self-conversion: `no_answer`
- Operations execution: `no_answer`
- Project membership workflow: `no_answer`
- Sales-manager no-answer archive: `archive_reason='no_answer'`

Static-audit guards now verify these contracts so a future change cannot silently reintroduce label-based routing.

## Runtime acceptance

On staging, test at least one legacy lead and one MIS distribution item: select `جواب نداده`, confirm attempt `1 از 3`, confirm the row appears in the `جواب نداده` tab, then record attempts 2 and 3. Manager archive behavior remains unchanged: it occurs only after the third no-answer attempt and the configured 3-day inactivity period.
