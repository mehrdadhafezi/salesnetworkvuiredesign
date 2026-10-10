# SN-203 QA report (Sales Deputy design)
Runner: `qa/qa-automated.js` (Playwright, Chromium). Serve `prototype/` on :8777, then `node qa/qa-automated.js`. Final run: **130 PASS, FAILS 0**.

| Pass | Scope | Result |
|---|---|---|
| A structural | 10 views × 1366 light; render, JS errors | PASS |
| B interaction | branch/class/exception drawers (owner, may/may-not, next actor, signal, UNKNOWN owner, 11 classes, no escalation command) · drill Manager→Senior→Supervisor→Seller with narrowing breadcrumb · snapshot/event never mixed · historical disabled · historical-unknown row · invoice mode current vs historical vs credit vs event · F02/F03/F04 diagnostics (no repair control, not-fixed wording, source matrix) · allocation: primary vs bypass, inactive/out-of-scope/unverified recipients disabled, bypass acknowledgement gate, preview fields, apply-recheck, count mode primary-only · result flows (partial 7-state strip reconciles, retry only failed, unknown→reconcile, conflict, recipient changed, out-of-scope, empty pool) · return preview (eligible, rank≠recall, fail-closed, conflict, consumed, cancelled/OPD-03, no commit) · invoice read-only (no assist/finance buttons, policy list) · HR (step vs final, reject needs reason, observe-only when not assigned, failed apply, OPD-05, no bulk) · monitor (bounded, null≠0, unauthorized/unavailable, no review controls) · reports (scope, bound, sensitive fields, export disabled, historical disabled, out-of-scope option disabled) · wallet pure-read · no finance-authority copy · 13 sims on 3 views, no JS errors · palette · drawer focus/return · landmarks | PASS |
| C matrix | 10 views × 1920/1366/1024/768/390 × (light·comfortable, dim·compact, dark·comfortable·focus): no horizontal page overflow, no JS errors, contrast ≥ 4.5:1 sampled at 1366 and 390 | PASS |
| D regression | Manager and Senior `qa-automated.js` re-run after the shared CSS addition: both FAILS 0; `crm-ext.css` change is additive (`.cov-matrix` only) | PASS |

Screenshots (`qa/*.png`): 01 overview · 02 perf invoice historical · 03 exceptions · 04 bypass review · 05 partial result · 06 blocked return · 07 diagnostics F03 · 08 HR drawer · 09 monitor unauthorized · 10 dark/compact/focus + branch failure · m01/m02 390 · t01 768 · d01 1920.
Not covered: real assistive technology; real data or a real multi-Manager Deputy account; HR/Finance live runtime.
