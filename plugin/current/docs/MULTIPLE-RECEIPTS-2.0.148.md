# Multiple receipts — 2.0.148

- Invoice, seller, supervisor, converter, manager legacy forms, and financial resubmission accept multiple JPEG/PNG/WEBP/PDF files.
- Each submission accepts up to 10 files, 5 MiB per file and 20 MiB in total. The host's PHP/web-server limits may be lower. The submitted count is checked to reject truncated batches.
- New uploads append to the current invoice payment stage. Explicit correction of a rejected invoice replaces that rejected receipt set; old/new links remain in the activity context.
- Invoice and payment-stage receipt collections are stored as JSON in additive `receipt_urls` columns. Legacy URL fields remain valid, and historical single receipts still render.
- Finance displays receipt counts and individual image/PDF files. Previous stage receipts can be opened from the finance receipt dialog.
- Multiple files do not create multiple payments or multiply the invoice/stage amount.
- Upload validation preserves the existing nonce, invoice access, payment-state and MIME checks. Uncommitted upload files are removed at request shutdown.

## Validation performed

- `node tools/test-multi-receipts.js`: 6 checks passed (all-file transport, rejecting a bad second file, limits, legacy links, link escaping, all picker/handler wiring).
- Actual Node FormData/Request multipart serialization verified three files plus the exact file-count field.
- Existing static audit: no new findings compared with the input archive. Five pre-existing audit findings concern Zibal exports, conversion-tab refresh and bulk-export timezone handling.
- PHP CLI, browser binary and WordPress/MySQL are unavailable in the build environment. PHP lint, browser rendering, database migration and end-to-end financial approval require validation on a WordPress test installation.

## WordPress acceptance checks

1. Upgrade the existing plugin and open its administration page to run additive migrations.
2. Submit two images and one PDF on one invoice; verify that finance shows all three and the amount is unchanged.
3. Add another receipt from the supervisor/converter panel while the same stage is under review; verify previous files remain visible.
4. Verify an older single-receipt invoice, a rejected invoice corrected with multiple files, and a subsequent payment stage.
5. Reject an oversized/unsupported/incomplete batch; it must not save a partial receipt set or mark payment successful.
