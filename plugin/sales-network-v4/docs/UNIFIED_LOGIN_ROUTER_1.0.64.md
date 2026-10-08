# Unified Login Router 1.0.64

- Added `[sn_login]` and `[sn_unified_login]` shortcodes.
- Added `sn_unified_login` public/admin-post handler.
- Login accepts mobile, username, or email, then resolves the user panel from HR position/profile.
- Added required page option `sn_login_page_id` with slug `crm-login`.
- Existing role-based login redirects now prefer the HR panel resolver.
