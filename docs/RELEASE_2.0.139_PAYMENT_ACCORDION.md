# 2.0.139 — Compact accordion payment queue

Each case is a closed native details/summary row showing customer, invoice, seller, remaining amount and status. Click or keyboard activation expands the existing financial details and payment actions. Only one case remains open per queue. Successful AJAX actions reopen the same case after refreshing its HTML. Layout adapts to narrow screens; existing permissions and payment logic are unchanged.

Validation: PHP syntax and both JS syntax checks passed. Existing 14 PHP-rendered action/permission assertions passed. No live site or browser visual test performed.
