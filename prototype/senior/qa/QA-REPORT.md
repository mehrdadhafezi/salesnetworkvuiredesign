# SN-201 QA report (Senior Supervisor design)
Runner: `qa/qa-automated.js` (Playwright, Chromium). Result of final run: **FAILS 0**.

| Pass | Scope | Result |
|---|---|---|
| A structural | 10 views × (1366, 390), light; JS errors, empty render, horizontal overflow | PASS (only a favicon 404) |
| B interaction | perf drilldown + 6 modes + historical unknown row; distribution preview (primary commit enabled / exception commit disabled + bypass + policy gap); blocked return (no commit, next actor); return confirm gating + result; flows partial/retry/unknown/conflict; ready readonly (no assign/escalate/bulk) + unresolved owner; invoice drawer (no approve); HR reject-needs-reason, step toast says not applied, chain queue; reports meta incl. export permission; case explorer caution; conditional personal module shown/hidden; 11 system states × 2 views no errors; team-failed partial banner; palette | PASS |
| C matrix | 9 views × widths 1920/1366/1024/768/390 × (light·comfortable, dim·compact, dark·comfortable·focus): no horizontal overflow, no JS errors; contrast ≥4.5:1 sampled on text at 1366 and 390 across themes | PASS |

Screenshots (representative, `qa/*.png`): 01 overview · 02 historical attribution (dim) · 03 exception path review · 04 blocked return · 05 ready readonly · 06 invoice drawer · 07 HR review · 08 team failed · 09 dark/compact/focus · m01/m02 phone · t01/t02 tablet.
Not covered: real assistive-technology run; real data; Seller/Supervisor pages were not re-rendered (their files are untouched).
