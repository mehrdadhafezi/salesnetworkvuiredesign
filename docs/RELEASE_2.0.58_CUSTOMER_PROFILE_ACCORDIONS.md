# Release 2.0.58 — Customer Profile Accordions

## Scope

- Each membership card resolves its image from the featured image of the linked WooCommerce `content_product_id`.
- Card images and modal thumbnails use a fixed landscape 3:2 ratio with `object-fit: cover`.
- The card modal displays the main WooCommerce product description under «توضیحات کارت».
- Missing images and missing descriptions have explicit fallback states.
- Account information, cards, purchases and payments are now separate titled accordions.
- Only one top-level profile accordion stays open; «اشتراک‌ها و کارت‌ها» is open initially.
- Profile navigation and wallet links open and scroll to the target accordion.

## Data and safety

- No new database column or migration is required.
- Product content is read live from WooCommerce and safely filtered before output.
- Existing membership snapshots, card operations, payment history, duplicate-subscription warning and invoice flow are unchanged.

## Staging acceptance

1. Open a customer containing at least two cards with different WooCommerce featured images.
2. Confirm each card shows its own product image and the visual container remains 3:2 on desktop and mobile.
3. Open each card and compare «توضیحات کارت» with the main WooCommerce product description.
4. Confirm the fallback message appears for a product without a description.
5. Confirm account, cards, purchases and payments have visible accordion titles.
6. Open each accordion and confirm the previously open top-level accordion closes.
7. Use the top navigation and wallet links and confirm the destination accordion opens and scrolls into view.
8. Recheck card action forms and modal keyboard close/focus behavior.
