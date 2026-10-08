# 2.0.138 — Payment queue inline actions and layout

- Fix lazy-rendered nonce fields embedding admin-ajax.php as the return URL. Nonce validation remains enabled; hidden referer fields are omitted.
- Register authenticated AJAX routes for assignment and next-stage issuance. Reuse existing ownership, nonce, stage-lock and transaction checks.
- Return JSON results and refreshed owner-scoped queue HTML; delegated capture listener handles dynamic forms, prevents duplicate submissions, restores controls on errors and shows inline feedback. Standard POST fallback rejects admin-ajax/admin-post return destinations.
- Correct stage success reporting: PHP array union previously preserved the initialized created=false value after commit.
- Replace stretched table presentation with responsive case cards, financial summary tiles, two-column stage form and separate cancellation action. Changes scoped to this queue.

Validation: PHP 8.0 parsing, both JavaScript syntax checks, 14 actual PHP rendering assertions retained. PHP 8.4 WebAssembly tests verify JSON success/failure, owner-context refreshed HTML, stage message, success flag assignment and absent lazy AJAX referer fields. No live WordPress/browser visual test available.
