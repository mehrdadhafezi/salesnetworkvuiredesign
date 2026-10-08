# Release 2.0.71 — Operations Header & Wallet Modal UI

## Scope
UI/UX polish for Biavin operations panels only. No accounting, commission, routing, chat, shipping, or payment business rules were changed.

## Changes
- Removed the always-visible Biavin wallet card from the top of operations panels.
- Added a compact "کیف پول بیاوین" launcher inside the operations hero actions.
- Wallet balance and transaction history now open in a centered viewport modal.
- Modal supports backdrop close, explicit close button, Escape key, focus return, and body scroll lock.
- Reworked operations hero actions into a compact horizontal action row: card count, wallet, notifications, logout.
- Improved responsive behavior of the hero, KPI cards, and filter toolbar.
- Explicit wallet text colors were added to avoid theme/header color leakage.

## Compatibility
- Existing `render_wallet_card()` calls remain compatible; a new optional context argument was added.
- Sales wallet views are unchanged.
- Existing wallet balances and transactions are unchanged.
