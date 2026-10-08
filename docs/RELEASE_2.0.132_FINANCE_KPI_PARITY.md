# 2.0.132 — Finance summary and tab count parity

Replaces the old mixed summary metrics with seven cards in the same order and
using the same keys as the disjoint payment tabs. Pending review no longer shows
the combined pending/manual/receipt queue total. Adds manual-deposit, gateway and
all-invoice cards; removes returned/cancelled and search-result cards from this
summary. Search totals remain above the table. The pending amount now uses the
pending tab predicate too. Overview and payments share the same summary markup.

Validation: JS syntax passed. Actual renderKpi tested against screenshot counts
(0,10,7,7,1136,52; total 1212) and all-zero data. All seven card counts matched tab
inputs and the six category counts summed to all. No live site testing performed.
