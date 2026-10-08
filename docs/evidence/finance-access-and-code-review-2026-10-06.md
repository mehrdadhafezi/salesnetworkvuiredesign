# Finance — limited access discovery and targeted code review

Date: 2026-10-06. Read-only discovery; no Finance operations executed.

## Access evidence

Chrome, test environment `https://crm.maximumclub.ir/crm-hr/`, existing Admin context. Workforce selected positions: معاون فروش (1)، مدیر فروش (2)، سرپرست ارشد (3)، سرپرست (4)، فروشنده (6)، MIS (9). No Finance account appeared in this observed inventory. Prior complete discovery in Role Access Map remains the accepted access evidence; this check is not a new full audit.

DOM-visible links: «پنل تایید مالی» → `/financial-approval/`; «ورود تایید مالی» → `/financial-login/`. Links demonstrate entry paths, not successful Finance impersonation. No Finance panel entered as Admin, no login/account/capability changed.

**FINANCE ROLE RUNTIME NOT LIVE VERIFIED.** Discovery LIVE VERIFIED only for the observed Admin page; financial workflow/runtime enforcement NOT LIVE VERIFIED.

## Targeted static evidence

Accepted Audit Finance inventory, Code Registry and Gate 0 reused; Sales/MIS/HR not re-audited.

- `includes/class-sn-plugin.php:33353`: review run guarded by sn_can_view_finance + nonce; approve records approval/actor/time/note/locked_at; approved run cannot be rejected in this branch. Not proof of all-path immutable snapshots.
- `:33388`: global posting activation flag write uses finance-view gate + nonce (F06).
- `:33400`: approved run/enabled flag/APPLY guards; query LIMIT 2000; existing transaction reference recovery/idempotent hit, per-item posted/error logs. Bounded processed summary does not prove full run coverage or atomic exactly-once concurrency.
- Accepted A/C: rules/runs/post/recalculate view/write issue F06; legacy wallet renderer conditional autopost F07; matrix/purpose engine has separate save/backfill authority and stage-credit/reversal behavior. No runtime autopost occurrence claimed.

No approve/reject/refund/posting/ledger/payment/rule/run/compensation action or export executed. Target permissions/statuses in Spec are logical contracts, not installed WP capability/status names.

## Local validation

Completed: Spec has 20 ordered sections (1–20), no broken local Markdown links, 46,892 bytes. All 248 baseline plugin files match workspace-before.json; zero changed code files. Input ZIP SHA256 remains `ae785ad4b27224e2ec015477188cf54409a251f3ad362f3a4d42181e39b03802`. Hash comparison pertains to local workspace, not installed site bytes. Only documentation artifacts created; no implementation or Claude work started.
