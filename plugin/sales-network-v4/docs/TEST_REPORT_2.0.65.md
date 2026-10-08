# Test Report — 2.0.65

## Scope
1. Explicit Sales Manager → Operations Sales Supervisor routing must stop at supervisor queue.
2. Operations case cards must be accordion UI in all five requested panels.
3. Dynamic form checkbox rendering/submission/persistence must be reliable.
4. Wallet/commission business logic is intentionally out of scope.

## Static checks
- PHP lint: PASS (20 PHP files).
- `node --check assets/js/operations-flow.js`: PASS.
- `node tools/static-audit.js`: PASS on version 2.0.65.

## Routing contract
- `sales_manager_supervisor_route` is treated as an explicit supervisor hand-off.
- Forced product routing does not auto-select the sole expert for that route.
- Destination remains `sales_supervisor`; `sales_expert_user_id` is not populated by this path.
- Legacy/default manager-only route keeps prior unique-supervisor/unique-expert convenience behavior.

## Accordion contract
- Common `.sn-ops-case` cards are collapsed by default.
- Toggle is injected into `.sn-ops-case-head` with `aria-expanded` state.
- Applies to operations sales manager, supervisor, expert, executive manager, and execution expert panels because all use the common Operations assets/card class.
- Bulk-selection checkbox stays in the visible header while the body is collapsed.

## Form checkbox contract
- Unchecked checkbox explicitly submits `0` through a hidden field.
- Checked checkbox submits `1` after the hidden field.
- Existing stored `1` re-renders checked; stored `0` re-renders unchecked.
- Required checkbox gets native `required` validation and remains protected by server-side required validation.
- Checkbox CSS overrides generic form input sizing so it renders as a normal compact checkbox.
