# 2.0.129 — Sales-origin filters in Operations panels

Adds a native details/summary accordion titled Filter to the Operations Sales
Manager and Operations Sales Supervisor panels. Five independent selects:
seller, converter, supervisor, senior supervisor, sales manager. Multiple values
combine with AND. Apply and clear controls preserve normal panel navigation.

Options originate exclusively from the panel's authorized operations, not the
full user directory. Seller/converter provenance uses source invoice and Dot
case identifiers, with membership origin fallback. Supervisor/senior hierarchy
uses current HR reports-to relationships (legacy user metadata fallback); the
stored source sales-manager snapshot remains authoritative where available.
No new access is granted; original panel ownership predicates remain in SQL.

Filter resolution covers the entire authorized scope before the existing
500-card display limit. Existing stage/contact/card/text filters still apply to
the resulting displayed cards. Bad IDs return an empty result without exposing
out-of-scope names. Selection values are integers and output labels are escaped.

Validation: actual panel SQL exercised in SQLite with 602 synthetic cases;
old matching cards were found before LIMIT, manager and supervisor scopes
excluded out-of-scope cases, and empty results remained empty. Live PHP,
WordPress/MySQL and visual browser tests unavailable.
