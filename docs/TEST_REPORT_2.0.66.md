# Test Report 2.0.66

## Scope
UI/UX-only polish for Operations case summaries/accordion controls and Customer Profile card detail modal.

## Automated checks
- PHP lint: PASS across 20 PHP files.
- JavaScript syntax: PASS for operations-flow.js and customer-portal.js.
- tools/static-audit.js: PASS, version 2.0.66.
- Release diff reviewed: no wallet/commission implementation files changed.

## Expected behavior
- All five Operations role panels show prominent customer/card/phone/positive-credit summary while cases are collapsed.
- Accordion uses a labeled control instead of a small standalone chevron.
- Customer Profile card CTA is labeled «مشاهده».
- Card modal is moved to document.body before display and centered relative to viewport on desktop and mobile.
