# Sales Network 2.0.27 — Critical init-hook hotfix

## Incident

Version 2.0.26 registered `maybe_process_unpaid_archives()` on WordPress' `init`
action with the default accepted-argument count. WordPress passes a legacy empty
string placeholder to no-payload actions. On PHP 8.1 this reached an `int`-typed
parameter and caused a `TypeError` on every request.

## Fix

- The `init` callback now explicitly accepts zero hook arguments.
- The public callback defensively normalizes values supplied by direct callers.
- The strict integer contract remains inside `process_due_unpaid_archives()`.
- A static regression check prevents reintroducing the typed callback or the
  default WordPress accepted-argument behavior.

This hotfix does not delete, rewrite, or roll back any sales data or schema.

## Verification

- All 17 PHP files parsed with tree-sitter PHP: zero syntax error nodes.
- `node tools/static-audit.js --json`: PASS.
- ZIP inventory confirms all literal bootstrap dependencies, including
  `includes/class-sn-migration-service.php`, are present.

An end-to-end WordPress staging smoke test is still required because this build
environment does not include WordPress, MySQL, or PHP CLI.
