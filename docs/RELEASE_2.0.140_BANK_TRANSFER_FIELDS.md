# 2.0.140 — Manual bank transfer details

Built on 2.0.139. Internal seller, supervisor and converter manual-payment dialogs now offer card, Paya, Pol and account-to-account transfer methods. Non-card methods expose account last six, tracking last six and source account holder with method-specific Persian guidance. Account/tracking are optional because source receipts differ; account holder is required. Digits are stored as strings, preserving leading zeroes. Persian/Arabic numbers are normalized by the existing helper. Unknown methods and nonempty identifiers other than six digits are rejected server-side.

Adds four nullable invoice columns using the existing versioned runtime migration. Save uses existing stage amount, authorization, financial review and transaction checks. Missing migrated columns abort non-card writes rather than silently discard fields. Transfer method is separate metadata; existing manual-payment routing remains intact. Metadata is visible in the finance receipt dialog, cleared when issuing/resetting the next stage and retained in submission activity history. Old card requests default to card and clear transfer-only fields.

Scope: internal manual-entry dialogs, not a change to the public customer payment form or bank reconciliation rules. Guidance does not derive an account number from an IBAN or assume matching Pol tracking IDs.

Validation: PHP 8.0 syntax parsed, four modified JavaScript files syntax checked. Actual PHP 8.4 WebAssembly validation covers all three methods, optional fields, leading zeroes/Persian digits, required holder, malformed lengths, unknown method and legacy card default. No live WordPress/MySQL or browser visual test performed.
