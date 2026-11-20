# RM-E04J — Seller S1 contact-outcome interactive preview

Baseline: `main` @ `e8f34e7d3290aa8b1fbfe815a4c210b86bfe22fa` (PR #82 merged).
Scope: DEV/TEST-only Seller preview (`/dev/seller/leads`), SYNTHETIC fixtures. **UI functionality only — not authorized business functionality.**

## A7 status

RM-E04C Group A item A7 was already implemented (`state/uiState.ts`, `components/states.tsx`, `seller.ui-states.test.tsx`).
It was verified, reused unchanged and **not** reimplemented. One A7-adjacent gap was closed: toasts (context-level) could outlive
a transition into a blocking answer for up to 4.5 s; `setSim` now clears them when the new state `blocksContent`.

## What changed

| Step | Before | After |
|---|---|---|
| Outcome + fields | `OutcomeGrid` + `validateOutcome` | unchanged (same grid, labels, fields, rules, keys 1–5) |
| «ثبت …» CTA | validate → demo toast | validate → **review step** (invalid drafts cannot enter it) |
| Review | — | lead ref (synthetic id), customer (or «محدود»), outcome, entered facts, CRM status «(بدون تغییر)», simulation notice; ویرایش / دورانداختن پیش‌نویس / تأیید شبیه‌سازی |
| Confirm | — | **simulated result**: labelled «شبیه‌سازی»; one simulated timeline entry marked «جزو تاریخچه CRM نیست · بدون زمان سرور · بدون شناسه رویداد»; fixture history/status untouched; شروع دوباره resets |

- Draft model: `web/src/modules/seller/state/outcomeDraft.ts` (pure reducer; steps are presentation steps, not business statuses).
- The «موفق» product / payment-type controls became controlled so the review shows what was chosen (same options and labels).
- Focus: review/result heading on entry; chosen outcome button on «ویرایش»; Esc still closes the drawer (draft dropped).
- In review/result, digit keys and Enter cannot change or confirm the draft; a double click on «تأیید» lands on a disabled placeholder, never on «شروع دوباره».

## Invariants kept

No request (fetch / XHR / beacon asserted absent), no storage, no `cases.contact_outcome.record` wiring, no production route,
no runtime flag. Blocking answers (401 / 403 ACTION_DENIED / 403 ROW_SCOPE_DENIED / FEATURE_DISABLED) unmount the drawer and the
draft; recovering does not resurrect it. FIELD_DENIED review shows «محدود» only. 404 stays generic.

`R5-SEC-01 = PREFERRED_A / NOT_APPROVED · F1 = INTERIM_GUARD_ONLY · Group B / S4 / D-42 / Real grants = BLOCKED · Production = NO-GO` — unchanged.

## Evidence

`docs/evidence/rm-e04j/` — `{desktop 1366, wide 1920, tablet 768, mobile 390}-{s1-list, drawer-open, validation, review, simulated, review-field-denied, blocked-*}.png`
Regenerate: `cd web && RM_E04J_EVIDENCE_DIR=../docs/evidence/rm-e04j npx playwright test -c playwright.seller.config.ts -g "E04J"`

## Rollback

Revert the PR commit. No data, schema, permission or configuration state to undo.
