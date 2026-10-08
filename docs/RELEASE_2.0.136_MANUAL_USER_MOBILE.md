# 2.0.136 — Reject duplicate mobile numbers in manual HR creation

Manual Add Person is now insert-only. Existing numbers produce a Persian error through the existing AJAX and POST report paths before account/profile updates. Existing names, phone metadata and access assignments are not overwritten.

The duplicate lookup checks account login, standard CRM/WooCommerce/Digits mobile metadata and HR work_phone. It recognizes Persian/Arabic digits, country prefixes, common separators and legacy user_ usernames. Emergency contacts are not treated as unique identity fields. International/no-leading-zero input is normalized in this form only. Username collisions no longer trigger suffixed account creation.

Validation: changed PHP syntax parsed for PHP 8.0. Actual PHP helper generated the SQL using PHP 8.4 WebAssembly; 73 SQLite fixture checks passed for legacy storage formats, metadata keys, nonduplicates and shared emergency contacts. Insert/update guard ordering inspected. No live WordPress/MySQL integration test was available. Other creation/import/edit flows are unchanged. Gateway export pagination limitation from 2.0.135 remains unresolved.
