# 2.0.141 — Choose receipt upload or manual transfer information

Internal seller/supervisor/converter dialogs now start with two explicit paths. Receipt path displays current amount and required image/PDF only. Information path first asks card/Paya/Pol/account method, then displays matching fields, date and time. Back control returns to the initial selection. Shared helper and CSS provide matching controls, selectors and time fields across panels.

Receipt path uses existing authenticated manual-payment access checks, upload helper, stage gates and transactional save; preserves the internal submitting actor source. It clears stale manual identifiers. Explicit information-only submissions clear old receipt links; hidden selected files are never attached to this path. Legacy submissions without entry_mode keep their previous behavior.

Validation: modified PHP and JS syntax checks; actual PHP WebAssembly tests verify required receipt, upload path and actor source, stale-field clearing, separate information path and prior bank-field validations. No live site/browser visual test performed.
