# Release 2.0.147 — HR manual user creation fix

- Fixed a frontend class mismatch that made HR AJAX success/error messages invisible.
- Manual user creation now immediately shows a visible “در حال ثبت نیروی جدید...” state after submit.
- Server validation errors such as duplicate mobile, invalid position, expired nonce, invalid hierarchy, and WordPress user creation failures are shown directly inside the form instead of appearing to do nothing.
- Successful creation keeps the existing reload behavior and returns the created user ID in the AJAX response for diagnostics.
- No invoice, discount, payment, catalog, or gateway logic was changed.
