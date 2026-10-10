<?php
/**
 * Plugin Name: Sales Network (شبکه فروش)
 * Plugin URI:  https://example.com
 * Description: مدیریت فروشندگان، تخصیص شماره و صدور فاکتور با پرداخت آنلاین یا کارت به کارت
 * Version: 2.0.160
 * Author:      M.Hafezi & A.Nazari
 * Text Domain: sn
 * Domain Path: /languages
 * Requires at least: 6.0
 * Requires PHP: 8.0
 */

if ( ! defined( 'ABSPATH' ) ) {
	exit;
}

define( 'SN_VERSION', '2.0.160' );
define( 'SN_BUILD', '2026-10-10-finance-dialog-layout' );
define( 'SN_PERF_INDEX_VERSION', '2026-07-21-invoice-hierarchy-export-v1' );
define( 'SN_PLUGIN_FILE', __FILE__ );
define( 'SN_PLUGIN_DIR',  plugin_dir_path( __FILE__ ) );
define( 'SN_PLUGIN_URL',  plugin_dir_url( __FILE__ ) );

add_action( 'init', static function (): void {
	if ( get_option( 'sn_report_caps_version' ) !== '2.1.0' ) { SN_Activator::register_report_caps(); }
	if ( ! get_option( 'sn_report_state_baseline_at' ) ) { SN_Activator::register_report_state_schema(); }
}, 5 );

/**
 * Canonical CRM login URL used after staff logout.
 * Prefer the provisioned CRM login page, while keeping the documented slug as
 * a safe fallback for fresh installs or temporarily missing page options.
 */
function sn_crm_login_url(): string {
	$page_id = (int) get_option( 'sn_login_page_id', 0 );
	$url     = $page_id > 0 ? (string) get_permalink( $page_id ) : '';
	return $url !== '' ? $url : home_url( '/crm-login/' );
}

/**
 * Biavin Operations roles must always return to the unified CRM login after
 * logout, regardless of which Operations screen initiated the sign-out.
 */
function sn_biavin_logout_redirect( string $redirect_to, string $requested_redirect_to, $user ): string {
	if ( ! ( $user instanceof WP_User ) ) { return $redirect_to; }
	$biavin_roles = [
		'sn_operations_sales_manager',
		'sn_operations_sales_supervisor',
		'sn_operations_sales_expert',
		'sn_operations_executive_manager',
		'sn_operations_execution_expert',
	];
	return array_intersect( $biavin_roles, (array) $user->roles ) ? sn_crm_login_url() : $redirect_to;
}
add_filter( 'logout_redirect', 'sn_biavin_logout_redirect', 20, 3 );

require_once SN_PLUGIN_DIR . 'includes/class-sn-activator.php';
require_once SN_PLUGIN_DIR . 'includes/class-sn-helpers.php';
require_once SN_PLUGIN_DIR . 'includes/class-sn-payment-transactions.php';
require_once SN_PLUGIN_DIR . 'includes/class-sn-invoice-review.php';
SN_Invoice_Review::register_hooks();
require_once SN_PLUGIN_DIR . 'includes/class-sn-sales-catalog.php';
SN_Sales_Catalog::instance()->register_hooks();
require_once SN_PLUGIN_DIR . 'includes/class-sn-dot-flow.php';
require_once SN_PLUGIN_DIR . 'includes/class-sn-campaign-tracking.php';
require_once SN_PLUGIN_DIR . 'includes/class-sn-projects.php';
require_once SN_PLUGIN_DIR . 'includes/class-sn-customer-portal.php';
require_once SN_PLUGIN_DIR . 'includes/class-sn-operations-flow.php';
require_once SN_PLUGIN_DIR . 'includes/class-sn-operations-execution.php';
require_once SN_PLUGIN_DIR . 'includes/class-sn-product-card-sync.php';
require_once SN_PLUGIN_DIR . 'includes/class-sn-biavin-api.php';
require_once SN_PLUGIN_DIR . 'includes/class-sn-hr-transfer.php';
require_once SN_PLUGIN_DIR . 'includes/class-sn-seller-flow.php';
require_once SN_PLUGIN_DIR . 'includes/class-sn-product-flow.php';
require_once SN_PLUGIN_DIR . 'includes/class-sn-subscription-form.php';
require_once SN_PLUGIN_DIR . 'includes/class-sn-purpose-commission.php';

SN_Campaign_Tracking::instance()->register_hooks();
SN_Projects::instance()->register_hooks();
SN_Customer_Portal::instance()->register_hooks();
SN_Operations_Flow::instance()->register_hooks();
SN_Operations_Execution::instance()->register_hooks();
SN_Product_Card_Sync::instance()->register_hooks();
SN_HR_Transfer::instance()->register_hooks();
SN_Seller_Flow::instance()->register_hooks();
SN_Product_Flow::instance()->register_hooks();
SN_Subscription_Form::register();
SN_Purpose_Commission::instance()->register_hooks();

register_activation_hook( __FILE__, [ 'SN_Activator', 'activate' ] );
register_deactivation_hook( __FILE__, [ 'SN_Activator', 'deactivate' ] );

function sn_bootstrap_request_action(): string {
	return sanitize_key( wp_unslash( $_REQUEST['action'] ?? '' ) );
}

function sn_bootstrap_is_admin_post_request(): bool {
	return basename( (string) ( $_SERVER['PHP_SELF'] ?? '' ) ) === 'admin-post.php';
}

function sn_bootstrap_is_sn_action_request(): bool {
	$action = sn_bootstrap_request_action();
	return $action !== '' && strpos( $action, 'sn_' ) === 0;
}

function sn_bootstrap_is_sn_rest_request(): bool {
	$route = (string) ( $_GET['rest_route'] ?? '' );
	$uri   = (string) ( $_SERVER['REQUEST_URI'] ?? '' );
	return strpos( $route, '/sn-crm/' ) === 0 || strpos( $uri, '/wp-json/sn-crm/' ) !== false;
}

function sn_bootstrap_is_callback_request(): bool {
	return ! empty( $_GET['sn_callback'] ) && ! empty( $_GET['invoice_id'] );
}

function sn_bootstrap_is_asan_redirect_request(): bool {
	return ! empty( $_GET['sn_asan_redirect'] ) && ! empty( $_GET['ref_id'] );
}

function sn_bootstrap_should_run_full(): bool {
	if ( defined( 'WP_CLI' ) && WP_CLI ) { return true; }
	if ( defined( 'DOING_CRON' ) && DOING_CRON ) { return true; }
	if ( wp_doing_ajax() ) {
		// Campaign landing beacons are intentionally lightweight and are handled
		// by SN_Campaign_Tracking without loading the multi-megabyte CRM core.
		// Project unread polling is also self-contained and runs every 30 seconds;
		// keeping it on the lightweight bootstrap avoids loading the full CRM for
		// every passive notification check.
		if ( in_array( sn_bootstrap_request_action(), [ 'sn_campaign_track', 'sn_project_notifications', 'sn_hr_transfer_search_users', 'sn_seller_flow_no_answer_attempt', 'sn_customer_portal_send_otp', 'sn_customer_portal_verify_otp' ], true ) ) { return false; }
		return sn_bootstrap_is_sn_action_request();
	}
	if ( sn_bootstrap_is_admin_post_request() ) {
		if ( sn_bootstrap_request_action() === 'sn_hr_export_users' ) { return false; }
		return sn_bootstrap_is_sn_action_request();
	}
	if ( sn_bootstrap_is_sn_rest_request() || sn_bootstrap_is_callback_request() || sn_bootstrap_is_asan_redirect_request() ) { return true; }
	if ( is_admin() ) { return true; }
	return false;
}

function sn_bootstrap_load_core(): void {
	static $loaded = false;
	if ( $loaded ) { return; }
	$loaded = true;
	require_once SN_PLUGIN_DIR . 'includes/class-sn-migration-service.php';
	require_once SN_PLUGIN_DIR . 'includes/class-sn-hr-service.php';
	require_once SN_PLUGIN_DIR . 'includes/class-sn-hierarchy-service.php';
	require_once SN_PLUGIN_DIR . 'includes/class-sn-scope-service.php';
	require_once SN_PLUGIN_DIR . 'includes/reports/class-sn-report-service.php';
	require_once SN_PLUGIN_DIR . 'includes/reports/class-sn-report-executive.php';
	require_once SN_PLUGIN_DIR . 'includes/class-sn-sms.php';
	require_once SN_PLUGIN_DIR . 'includes/class-sn-invoice.php';
	require_once SN_PLUGIN_DIR . 'includes/class-sn-plugin.php';
}

function sn_bootstrap_plugin_instance( bool $run = false ): SN_Plugin {
	static $plugin = null;
	sn_bootstrap_load_core();
	if ( ! $plugin ) { $plugin = new SN_Plugin(); }
	static $has_run = false;
	if ( $run && ! $has_run ) { $plugin->run(); $has_run = true; }
	return $plugin;
}

function sn_bootstrap_shortcodes(): array {
	return [
		'sn_seller_panel' => 'render_seller_panel',
		'sn_supervisor_panel' => 'render_supervisor_panel',
		'sn_senior_supervisor_panel' => 'render_senior_supervisor_panel',
		'sn_sales_manager_auth' => 'render_sales_manager_auth',
		'sn_sales_manager_panel' => 'render_sales_manager_panel',
		'sn_sales_deputy_panel' => 'render_sales_deputy_panel',
		'sn_after_sales_panel' => 'render_after_sales_panel',
		'sn_financial_auth' => 'render_financial_auth',
		'sn_financial_panel' => 'render_financial_panel',
		'sn_invoice_review_panel' => 'render_invoice_review_panel',
		'sn_hr_panel' => 'render_hr_panel',
		'sn_mis_panel' => 'render_mis_panel',
		'sn_customer_profile' => 'render_customer_profile',
		'sn_reports_panel' => 'render_reports_panel',
		'sn_portal_nav' => 'render_portal_nav',
		'sn_invoice_page' => 'render_invoice_page',
		'sn_login' => 'render_unified_login_page',
		'sn_unified_login' => 'render_unified_login_page',
		'sn_auth' => 'render_auth',
		'sn_supervisor_auth' => 'render_supervisor_auth',
		'sn_my_panel' => 'render_my_panel',
		'sn_my_password' => 'render_my_password',
		'sn_admin_front_dashboard' => 'render_admin_front_dashboard',
		'sn_dot_customer_flow' => 'render_dot_customer_flow',
		'sn_dot_converter_panel' => 'render_dot_converter_panel',
		'sn_dot_marketing_form' => 'render_dot_marketing_form',
		// Compatibility aliases for pages created by releases before 2.0.44.
		'sn_project_manager_panel' => 'render_operations_sales_manager_panel',
		'sn_project_expert_panel' => 'render_operations_sales_expert_panel',
		'sn_shipping_panel' => 'render_shipping_panel',
		'sn_operations_sales_manager_panel' => 'render_operations_sales_manager_panel',
		'sn_operations_sales_supervisor_panel' => 'render_operations_sales_supervisor_panel',
		'sn_operations_sales_expert_panel' => 'render_operations_sales_expert_panel',
		'sn_operations_executive_manager_panel' => 'render_operations_executive_manager_panel',
		'sn_operations_execution_expert_panel' => 'render_operations_execution_expert_panel',
	];
}

function sn_bootstrap_register_light_shortcodes(): void {
	foreach ( sn_bootstrap_shortcodes() as $shortcode => $method ) {
		add_shortcode( $shortcode, static function ( $atts = [] ) use ( $shortcode, $method ) {
			$plugin = sn_bootstrap_plugin_instance( false );
			if ( ! method_exists( $plugin, $method ) ) { return ''; }
			// The Marketing Dot shortcode uses its id attribute to resolve the exact
			// product/supervisor route. Dropping attributes here silently rendered
			// every custom shortcode as the default form.
			if ( $shortcode === 'sn_dot_marketing_form' ) {
				return (string) $plugin->{$method}( is_array( $atts ) ? $atts : [] );
			}
			return (string) $plugin->{$method}();
		} );
	}
}

function sn_bootstrap_current_post_has_crm_shortcode(): bool {
	global $post;
	if ( ! $post || ! is_singular() ) { return false; }
	$content = (string) ( $post->post_content ?? '' );
	$elementor_data = (string) get_post_meta( (int) $post->ID, '_elementor_data', true );
	foreach ( array_keys( sn_bootstrap_shortcodes() ) as $shortcode ) {
		if ( has_shortcode( $content, $shortcode ) || strpos( $elementor_data, $shortcode ) !== false ) { return true; }
	}
	return false;
}

function sn_bootstrap_enqueue_public_assets(): void {
	global $post;
	$is_marketing_form = false;
	if ( $post && is_singular() ) {
		$post_content = (string) ( $post->post_content ?? '' );
		$elementor_data = (string) get_post_meta( (int) $post->ID, '_elementor_data', true );
		$is_marketing_form = has_shortcode( $post_content, 'sn_dot_marketing_form' )
			|| strpos( $elementor_data, 'sn_dot_marketing_form' ) !== false;
	}
	if ( $is_marketing_form ) {
		$marketing_css = SN_PLUGIN_DIR . 'assets/css/dot-marketing.css';
		$marketing_js  = SN_PLUGIN_DIR . 'assets/js/public-dot-marketing.js';
		wp_enqueue_style( 'sn-dot-marketing', SN_PLUGIN_URL . 'assets/css/dot-marketing.css', [], SN_VERSION . '-' . ( file_exists( $marketing_css ) ? (string) filemtime( $marketing_css ) : '0' ) );
		wp_enqueue_script( 'sn-dot-marketing', SN_PLUGIN_URL . 'assets/js/public-dot-marketing.js', [], SN_VERSION . '-' . ( file_exists( $marketing_js ) ? (string) filemtime( $marketing_js ) : '0' ), true );
	}
	if ( sn_bootstrap_current_post_has_crm_shortcode() || ( function_exists( 'is_account_page' ) && is_account_page() ) ) {
		sn_bootstrap_plugin_instance( false )->enqueue_public_assets();
	}
}

function sn_bootstrap_hide_admin_bar_for_front_roles(): void {
	if ( ! is_user_logged_in() || current_user_can( 'manage_options' ) ) { return; }
	$user = wp_get_current_user();
	$roles = (array) ( $user->roles ?? [] );
	if ( array_intersect( $roles, [ 'sn_supervisor', 'sn_senior_supervisor', 'sn_sales_deputy', 'sn_seller', 'sn_converter', 'sn_hr', 'sn_mis', 'sn_finance', 'sn_invoice_reviewer', 'sn_after_sales', 'sn_sales_manager', 'sn_financial', 'sn_financial_approval', 'sn_campaign_partner', 'sn_shipping_expert', 'sn_operations_sales_manager', 'sn_operations_sales_supervisor', 'sn_operations_sales_expert', 'sn_operations_executive_manager', 'sn_operations_execution_expert' ] ) ) {
		show_admin_bar( false );
	}
}

function sn_bootstrap_protect_dot_customer_link(): void {
	if ( empty( $_GET['dot_case'] ) ) { return; }
	if ( ! headers_sent() ) {
		header( 'Referrer-Policy: no-referrer' );
		header( 'Cache-Control: private, no-store, no-cache, must-revalidate, max-age=0' );
	}
	add_filter( 'wp_robots', static function ( array $robots ): array {
		$robots['noindex'] = true;
		$robots['nofollow'] = true;
		$robots['noarchive'] = true;
		return $robots;
	} );
}

function sn_bootstrap_handle_short_invoice_link(): void {
	$uri_path = (string) wp_parse_url( (string) ( $_SERVER['REQUEST_URI'] ?? '' ), PHP_URL_PATH );
	$path = trim( rawurldecode( $uri_path ), '/' );
	if ( ! preg_match( '#^i/([A-Za-z0-9]{6,32})/?$#', $path, $m ) ) {
		return;
	}
	$short_code = sanitize_text_field( $m[1] );
	global $wpdb;
	$table = $wpdb->prefix . 'sn_invoices';
	if ( $wpdb->get_var( $wpdb->prepare( 'SHOW TABLES LIKE %s', $table ) ) !== $table ) {
		return;
	}
	$cols = $wpdb->get_col( "SHOW COLUMNS FROM {$table}", 0 );
	if ( ! in_array( 'short_code', $cols, true ) ) {
		return;
	}
	$invoice = $wpdb->get_row( $wpdb->prepare( "SELECT id, invoice_code, access_token FROM {$table} WHERE short_code=%s LIMIT 1", $short_code ) );
	if ( ! $invoice || empty( $invoice->invoice_code ) ) {
		return;
	}
	$access_token = (string) ( $invoice->access_token ?? '' );
	if ( $access_token === '' ) {
		$access_token = class_exists( 'SN_Helpers' ) ? SN_Helpers::generate_unique_access_token() : wp_generate_password( 48, false, false );
		$wpdb->update( $table, [ 'access_token' => $access_token ], [ 'id' => (int) $invoice->id ] );
	}
	$page_id = (int) get_option( 'sn_invoice_page_id' );
	$base = $page_id ? get_permalink( $page_id ) : home_url( '/' );
	$target = add_query_arg( [
		'invoice_code'  => (string) $invoice->invoice_code,
		'invoice'       => (string) $invoice->invoice_code,
		'access_token'  => $access_token,
	], $base ?: home_url( '/' ) );
	wp_safe_redirect( $target, 302 );
	exit;
}

/** Resolve compact Dot customer links before WordPress renders a 404 page. */
function sn_bootstrap_handle_dot_short_link(): void {
	$uri_path = (string) wp_parse_url( (string) ( $_SERVER['REQUEST_URI'] ?? '' ), PHP_URL_PATH );
	$path = trim( rawurldecode( $uri_path ), '/' );
	$home_path = trim( (string) wp_parse_url( home_url( '/' ), PHP_URL_PATH ), '/' );
	if ( $home_path !== '' && strpos( $path, $home_path . '/' ) === 0 ) {
		$path = substr( $path, strlen( $home_path ) + 1 );
	}
	if ( ! preg_match( '#^d/([0-9]+)-([a-f0-9]{14})/?$#i', $path, $matches ) ) { return; }
	$target = SN_Dot_Flow::instance()->resolve_short_access_url( absint( $matches[1] ), strtolower( (string) $matches[2] ) );
	if ( $target === '' ) { return; }
	if ( ! headers_sent() ) {
		header( 'Referrer-Policy: no-referrer' );
		header( 'Cache-Control: private, no-store, no-cache, must-revalidate, max-age=0' );
	}
	wp_safe_redirect( $target, 302, 'Sales Network Dot Access' );
	exit;
}

add_action( 'init', 'sn_bootstrap_register_light_shortcodes', 1 );
add_action( 'init', static function () {
	if ( is_user_logged_in() && ! get_option( 'sn_my_password_page_id' ) ) {
		SN_Activator::create_required_pages();
	}
}, 20 );
add_action( 'wp_enqueue_scripts', 'sn_bootstrap_enqueue_public_assets', 9 );
add_action( 'wp_footer', static function () {
	if ( is_user_logged_in() && is_singular() ) {
		sn_bootstrap_plugin_instance( false )->sn_render_other_panel_password_link();
	}
} );
add_action( 'template_redirect', 'sn_bootstrap_handle_dot_short_link', 0 );
add_action( 'template_redirect', 'sn_bootstrap_handle_short_invoice_link', 1 );
add_action( 'template_redirect', 'sn_bootstrap_protect_dot_customer_link', 0 );
add_action( 'after_setup_theme', 'sn_bootstrap_hide_admin_bar_for_front_roles', 1 );

add_action( 'plugins_loaded', static function () {
	if ( wp_doing_ajax() && sn_bootstrap_is_sn_action_request() ) {
		add_filter( 'jwt_auth_do_jwt_check', '__return_false', 999 );
		if ( ! ob_get_level() ) { ob_start(); }
	}
	if ( sn_bootstrap_should_run_full() ) {
		sn_bootstrap_plugin_instance( true );
	}
}, 0 );
