# Sales Network 2.0.38 — Product* normal and upgrade definitions

## Outcome

Every WooCommerce product resolved as `Product*`—including every product used
inside a subscription—can now define zero or more upgrade destinations. Each
row stores:

- the destination Product*;
- the exact amount the customer must pay to upgrade;
- the destination credit, read from the destination product when a new card is
  created.

The source card's normal value remains `_sn_product_credit_amount`. Upgrade
destinations must have a strictly greater credit than the source card.

## Product editor

The existing Sales Network product metabox shows a repeatable section named
`حالت‌های افزایشی این کارت` whenever the effective product type is Product*.
Admins can add multiple destinations, remove rows and set a separate upgrade
amount for each destination. Draft/private Product* records remain selectable,
so preparing products before publication does not require changing old sales.

Server validation rejects self-upgrades, duplicate destinations, non-Product*
destinations, zero/negative amounts and targets whose credit is not greater
than the source credit. A failed rule submission leaves the complete previous
rule set and the previous source credit intact.

## Immutable customer-card snapshot

Live definitions are stored in the new additive
`sn_project_card_upgrade_rules` table. When a subscription or standalone
Product* reaches full payment and creates a membership, each membership item
snapshots:

- `base_credit_snapshot`;
- `upgrade_options_snapshot_json` (target id, name, target credit and required
  payment).

Later product edits therefore affect only cards created after the edit. No
existing invoice, membership, project assignment, commission or project action
is updated or deleted.

Older membership items intentionally keep the new columns as `NULL`. They use
the prior live/fallback behavior for compatibility; no backfill is performed.

## Action safety

For cards with a snapshot, an upsell can only target one of the snapshotted
destinations. The server ignores any browser-supplied upgrade amount and credit
and replaces them with the card's trusted snapshot values. The project UI
shows the configured payment next to each destination and makes the calculated
fields read-only.

## Scope boundary

This release implements the Product* definition, snapshot and current project
action foundation only. Customer OTP/profile/timeline and the customer's three
activation choices are intentionally left for the next Biavin phase.
