# ARCHITECTURE

## Top-level plugin structure
Known from available package and phase reports:

- `sales-network.php`
  - Plugin bootstrap.
  - Defines constants.
  - Always loads bounded lightweight modules (Campaign, Projects and HR Transfer).
  - Uses a selective bootstrap for CRM AJAX/admin/callback requests and a light shortcode proxy for normal frontend requests.
  - Registers activation/deactivation.
  - Loads and instantiates the legacy `SN_Plugin` core only when the request or rendered shortcode requires it.
- `includes/class-sn-plugin.php`
  - Main hook registration, shortcodes, admin pages, AJAX/admin-post handlers.
  - Large legacy file; should be split gradually.
- `includes/class-sn-activator.php`
  - Roles, activation schema, default pages/settings in older package.
- `includes/class-sn-helpers.php`
  - Shared helpers, upload helpers, escaping/sanitization helpers.
- `includes/class-sn-sms.php`
  - SMS provider integration.
- `includes/class-sn-invoice.php`
  - Invoice-related helper/model code.
- Newer v4 services from reports:
  - `class-sn-migration-service.php` — versioned/safe migrations and repairs.
  - `class-sn-hr-service.php` — HR profiles, positions, levels, panel resolver.
  - `class-sn-hierarchy-service.php` — manager/assignment hierarchy and history.
  - `class-sn-scope-service.php` — scope simulation/enforcement decisions.
  - `class-sn-seller-flow.php` — shared seller stage-one state/history, bounded archive cron, manager archive rendering and invoice product-type classification.
- `assets/js/*`
  - Frontend/admin panel behavior.
- `assets/css/*`
  - Public/admin panel styling.
- `docs/*`
  - Project documentation for future Codex tasks.

## Request flow
- WordPress loads `sales-network.php`.
- Lightweight modules register their own hooks without loading the legacy core.
- CRM admin, AJAX, callback, cron and CLI requests load the full core and call `SN_Plugin::run()`.
- Normal frontend requests register proxy shortcodes; rendering a CRM shortcode loads the core on demand.
- Shortcode detection inspects both `post_content` and Elementor `_elementor_data` so required assets are selected consistently.
- When loaded, `SN_Plugin::run()` registers:
  - admin menus.
  - AJAX actions (`wp_ajax_*`, some `wp_ajax_nopriv_*`).
  - `admin_post_*` form handlers.
  - shortcodes.
  - assets.
- Public/role panels render via shortcodes.
- AJAX returns JSON, generally via `wp_send_json_success()` / `wp_send_json_error()`.
- Form handlers redirect back with status/report options.

## Panel routing
- `[sn_my_panel]` resolves the logged-in user with `SN_HR_Service::resolve_panel_for_user($user_id)`.
- HR Position wins over WordPress role when HR profile exists.
- Fallback uses legacy role/WP role for compatibility.
- Guest users should see the plugin login form, not `wp-login.php`.

## Scope architecture
- `SN_Scope_Service` provides read-only audit and enforcement.
- Modes:
  - `off`
  - `audit_only`
  - `enforce_seller`
  - `enforce_seller_supervisor`
  - `enforce_sales_management`
- Enforcement is intended only for read/list/search paths unless explicitly expanded.

## Database architecture
- Legacy operational tables store leads, invoices, payments, wallet, logs.
- HR tables store dynamic positions, levels, profiles, assignments, and logs.
- MIS tables stage imported rows before any live lead creation.
- Migrations must be idempotent and repair missing columns/tables without destructive operations.

## Frontend/backend relationship
- Server renders many panels and tables.
- JavaScript handles form/AJAX UX in existing panels.
- User-controlled output must be escaped before HTML insertion.
- Public invoice page requires token-aware frontend requests.

## Needs confirmation
- Current v4 repository may have additional directories/classes not present in the uploaded older package.
- Exact hook names added after Phase 3E-4 must be verified in the active codebase.
