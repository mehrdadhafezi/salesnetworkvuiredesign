# Release 2.0.81 — Customer activation and seller catalog

- Wallet cards show the one-time activation code prominently, with the complete destination URL and a separate visit button. URLs are escaped for display and links.
- After either self or expert activation form submission, the customer is redirected to the canonical profile URL with the submitted card ID. The portal opens that card automatically, then removes the temporary `card` query parameter from browser history. Existing nonce and ownership checks remain on the server.
- Products assigned to Dot marketing forms remain available in ordinary seller product selectors. The marketing shortcode and its system seller route remain available; disabled Dot assessment products still obey their previous visibility rule.

Verification: `node --check assets/js/customer-portal.js` and `node tools/static-audit.js` passed. PHP CLI and a live WordPress environment are unavailable here, so the activation flow should be smoke-tested on staging.
