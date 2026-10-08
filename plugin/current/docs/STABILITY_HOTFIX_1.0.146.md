# Sales Network 1.0.146 Stability Hotfix

Base: `1.0.145-staged-payment-workflow`

## Fixes

- Senior-supervisor panel uses per-tab lazy loading instead of rendering all reports in the initial request.
- Senior seller performance metrics use grouped queries and bounded table output.
- Senior-supervisor invoice JavaScript now initializes on both supervisor and senior-supervisor panels.
- Senior invoices default to complete history and retain date/status/search filters.
- A single role-independent sidebar shell owns desktop collapse and mobile drawer behavior, preventing duplicate hamburger handlers.
- HR individual and bulk password endpoints return clean JSON, validate nonces without raw `-1` responses, verify password writes, and keep audit logging best-effort.
- HR Ajax response parsing now reports useful server errors even when a local PHP warning precedes JSON.

## Data safety

No destructive schema operation was added. The staged-payment workflow and sales-deputy card routing remain in place.
