# Staged Invoice Payment Workflow — 1.0.145

## Scope

This release adds full-payment and staged prepayment support to CRM invoices while preserving the original seller as the commercial owner of the invoice.

## Core invariants

- One invoice code and one customer link are used for all payment stages.
- `seller_id` / `original_seller_id` remain tied to the seller who originally issued the invoice.
- The current workflow assignee is stored separately in `sn_distribution_items.current_owner_user_id`.
- The latest assigner is stored in `sn_distribution_items.workflow_assigned_by_user_id`; the original seller, current assignee and latest assigner can monitor the case and submit final cancellation, while only the current assignee can issue the next stage.
- Every routing, assignment, stage issuance, approval, completion and final cancellation is logged.
- WooCommerce order creation, wallet credit and final commission actions run only when the remaining balance reaches zero.
- Earlier approved stages are immutable and cannot be overwritten by a later stage.

## State flow

1. Seller issues an invoice and chooses:
   - Full payment
   - Prepayment from configured presets
   - Custom prepayment
2. Customer pays the active stage using online payment or card-to-card.
3. Online payment is verified automatically. Card-to-card is approved or rejected by finance.
4. If a balance remains, the invoice enters `partial_paid / awaiting_assignment` and a repeat-action item is created for the original seller.
5. The seller can:
   - Assign the completion to self
   - Send it to the direct supervisor
6. The supervisor can:
   - Assign it to self
   - Assign it to an eligible descendant user
7. The assignee issues the next stage on the same invoice:
   - Entire remaining balance
   - Another partial payment
8. When the remaining balance reaches zero, the invoice becomes paid/approved and final sales side effects run.
9. Before completion, the original seller, current responsible user or latest assigner can archive the invoice with a mandatory cancellation reason.

## Database additions

### `sn_invoices`

- `payment_plan`
- `payment_total_amount`
- `current_due_amount`
- `paid_total_amount`
- `remaining_amount`
- `current_payment_stage`
- `payment_workflow_status`
- `current_stage_issued_by_user_id`
- `payment_completed_at`
- `payment_archived_at`
- `payment_archived_by`
- `payment_archive_reason`

### `sn_invoice_payment_stages`

Stores each requested payment stage, amount, status, issuer, approval, payment method and reference ID. `(invoice_id, stage_no)` is unique.

### `sn_payments`

- `payment_stage_no`

### `sn_distribution_items`

- `workflow_kind`
- `workflow_assigned_by_user_id`

## Safety and concurrency

- Invoice creation, initial payment stage creation and invoice-item insertion are transactional.
- Payment-stage finalization locks the invoice row before changing totals.
- Next-stage issuance locks the invoice first and the workflow item second.
- Final cancellation uses the same lock order and rechecks the current balance after acquiring locks.
- An invoice with a card receipt pending finance review cannot be archived until finance approves or rejects that receipt.
- A verified online callback that was already in-flight before archival is reconciled instead of silently discarding a successful charge.
- Gateway callbacks are bound to the persisted invoice, payment stage and requested amount.
- Duplicate gateway callbacks are idempotent.
- No destructive schema operations (`DROP TABLE` / `TRUNCATE`) are introduced.

## Admin setting

`sn_partial_payment_presets` accepts comma-separated amounts in تومان. Sellers can also select “custom” and enter another amount.

## Required smoke test after update

1. Back up database and plugin folder.
2. Replace the existing `sales-network-v4` plugin with this ZIP.
3. Open one CRM/admin page once so additive migrations run.
4. Save prepayment presets.
5. Issue a 10,000,000 تومان invoice with a 2,000,000 تومان prepayment.
6. Test one online approval and one card-to-card finance approval.
7. Confirm the customer page shows total, approved amount, remaining amount and stage history.
8. Test seller self-assignment.
9. Test seller-to-supervisor routing, then supervisor self-assignment.
10. Test supervisor assignment to another eligible user.
11. Issue a second partial stage and then a full remaining stage.
12. Confirm WooCommerce order, wallet and final commission actions occur only after the balance reaches zero.
13. Test final cancellation on an incomplete invoice and confirm a completed invoice cannot be archived.
