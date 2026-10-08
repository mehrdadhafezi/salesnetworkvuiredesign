# 2.0.125 — Finance interaction feedback

- Compact Persian-number badges on all seven payment tabs show full tab totals,
  independent of the search text/page size, refreshed with every list response.
- Finance mutations show a spinner and working label. Conflicting buttons are
  disabled while a mutation is in flight; completion/error restores their state.
- Active tab shows loading feedback. Buttons have brief press and focus states.
- Receipt modal keeps the media column fixed and scrolls only the right details
  column. On narrow screens both columns remain visible with single-column
  metadata. Images fit inside the fixed media area; PDFs retain native scrolling.

Checks: JavaScript syntax passed; feedback/duplicate-request/retry unit checks
passed; prior 11 SQL routing scenarios passed. Live WordPress/PHP and visual
browser tests remain unavailable in this environment.
