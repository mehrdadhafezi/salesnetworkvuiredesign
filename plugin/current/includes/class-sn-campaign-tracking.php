<?php
if ( ! defined( 'ABSPATH' ) ) { exit; }

/**
 * First-party campaign tracking, attribution, partner reporting and budgets.
 *
 * Customer-identifying values are never exposed to partners or the reporting
 * API. The bridge between a browser session and a later CRM record uses a
 * salted one-way identity hash.
 */
final class SN_Campaign_Tracking {
	const DB_VERSION       = '2026-08-31-campaign-attribution-v3';
	const VISITOR_COOKIE   = 'sn_campaign_visitor';
	const SESSION_COOKIE   = 'sn_campaign_session';
	const SESSION_TTL      = 1800;
	const DEFAULT_WINDOW   = 90;
	const DEFAULT_RETENTION = 730;

	private static $instance = null;
	private $hooks_registered = false;
	private $capturing = false;
	private $table_cache = [];

	public static function instance(): self {
		if ( ! self::$instance ) { self::$instance = new self(); }
		return self::$instance;
	}

	private function __construct() {}

	public function register_hooks(): void {
		if ( $this->hooks_registered ) { return; }
		$this->hooks_registered = true;

		add_action( 'init', [ $this, 'maybe_upgrade' ], 2 );
		add_action( 'init', [ $this, 'register_shortcode' ], 3 );
		add_action( 'template_redirect', [ $this, 'handle_friendly_api' ], -20 );
		add_action( 'template_redirect', [ $this, 'capture_landing_request' ], 0 );
		add_action( 'wp_enqueue_scripts', [ $this, 'enqueue_tracking_asset' ], 8 );
		add_action( 'wp_ajax_sn_campaign_track', [ $this, 'ajax_track' ] );
		add_action( 'wp_ajax_nopriv_sn_campaign_track', [ $this, 'ajax_track' ] );
		add_action( 'rest_api_init', [ $this, 'register_rest_routes' ] );
		add_filter( 'jwt_auth_do_jwt_check', [ $this, 'disable_jwt_for_campaign_api' ], 999 );

		add_action( 'admin_menu', [ $this, 'register_admin_menu' ], 30 );
		add_action( 'admin_enqueue_scripts', [ $this, 'enqueue_admin_asset' ] );
		add_action( 'admin_post_sn_campaign_save', [ $this, 'handle_save_campaign' ] );
		add_action( 'admin_post_sn_campaign_assign_partner', [ $this, 'handle_assign_partner' ] );
		add_action( 'admin_post_sn_campaign_generate_token', [ $this, 'handle_generate_token' ] );
		add_action( 'admin_post_sn_campaign_revoke_token', [ $this, 'handle_revoke_token' ] );
		add_action( 'admin_post_sn_campaign_save_settings', [ $this, 'handle_save_settings' ] );

		add_action( 'user_register', [ $this, 'on_user_register' ], 20, 2 );
		add_action( 'sn_invoice_paid', [ $this, 'on_invoice_paid' ], 20, 2 );
		add_action( 'admin_init', [ $this, 'block_partner_admin' ], 2 );
		add_action( 'after_setup_theme', [ $this, 'hide_partner_admin_bar' ], 2 );
		add_filter( 'login_redirect', [ $this, 'partner_login_redirect' ], 20, 3 );
	}

	public static function activate(): void {
		$self = self::instance();
		$self->register_role();
		$self->install_schema();
	}

	public function maybe_upgrade(): void {
		$this->register_role();
		if ( (string) get_option( 'sn_campaign_db_version', '' ) !== self::DB_VERSION ) {
			$this->install_schema();
		}
	}

	private function tables(): array {
		global $wpdb;
		return [
			'campaigns'    => $wpdb->prefix . 'sn_campaigns',
			'sessions'     => $wpdb->prefix . 'sn_campaign_sessions',
			'events'       => $wpdb->prefix . 'sn_campaign_events',
			'attributions' => $wpdb->prefix . 'sn_campaign_attributions',
			'conversions'  => $wpdb->prefix . 'sn_campaign_conversions',
			'assignments'  => $wpdb->prefix . 'sn_campaign_partner_campaigns',
			'tokens'       => $wpdb->prefix . 'sn_campaign_partner_tokens',
		];
	}

	private function table_exists( string $table ): bool {
		if ( isset( $this->table_cache[ $table ] ) ) { return $this->table_cache[ $table ]; }
		global $wpdb;
		$this->table_cache[ $table ] = $wpdb->get_var( $wpdb->prepare( 'SHOW TABLES LIKE %s', $table ) ) === $table;
		return $this->table_cache[ $table ];
	}

	public function install_schema(): void {
		global $wpdb;
		require_once ABSPATH . 'wp-admin/includes/upgrade.php';
		$charset = $wpdb->get_charset_collate();
		$t = $this->tables();

		dbDelta( "CREATE TABLE {$t['campaigns']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			campaign_key CHAR(64) NOT NULL,
			title VARCHAR(191) NOT NULL,
			utm_source VARCHAR(191) DEFAULT NULL,
			utm_medium VARCHAR(191) DEFAULT NULL,
			utm_campaign VARCHAR(191) DEFAULT NULL,
			utm_content VARCHAR(191) DEFAULT NULL,
			utm_term VARCHAR(191) DEFAULT NULL,
			budget_amount DECIMAL(20,2) NOT NULL DEFAULT 0,
			cost_amount DECIMAL(20,2) NOT NULL DEFAULT 0,
			currency VARCHAR(12) NOT NULL DEFAULT 'IRR',
			status VARCHAR(20) NOT NULL DEFAULT 'active',
			created_by BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY campaign_key (campaign_key),
			KEY utm_source (utm_source),
			KEY utm_medium (utm_medium),
			KEY utm_campaign (utm_campaign),
			KEY status (status)
		) $charset;" );

		dbDelta( "CREATE TABLE {$t['sessions']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			visitor_id CHAR(36) NOT NULL,
			session_id CHAR(36) NOT NULL,
			first_campaign_id BIGINT UNSIGNED DEFAULT NULL,
			last_campaign_id BIGINT UNSIGNED DEFAULT NULL,
			landing_page_url TEXT DEFAULT NULL,
			first_utm_source VARCHAR(191) DEFAULT NULL,
			first_utm_medium VARCHAR(191) DEFAULT NULL,
			first_utm_campaign VARCHAR(191) DEFAULT NULL,
			first_utm_content VARCHAR(191) DEFAULT NULL,
			first_utm_term VARCHAR(191) DEFAULT NULL,
			last_utm_source VARCHAR(191) DEFAULT NULL,
			last_utm_medium VARCHAR(191) DEFAULT NULL,
			last_utm_campaign VARCHAR(191) DEFAULT NULL,
			last_utm_content VARCHAR(191) DEFAULT NULL,
			last_utm_term VARCHAR(191) DEFAULT NULL,
			ip_address VARCHAR(45) DEFAULT NULL,
			user_agent VARCHAR(500) DEFAULT NULL,
			started_at DATETIME NOT NULL,
			last_seen_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY session_id (session_id),
			KEY visitor_started (visitor_id,started_at),
			KEY first_campaign_started (first_campaign_id,started_at),
			KEY last_campaign_started (last_campaign_id,started_at),
			KEY last_seen_at (last_seen_at)
		) $charset;" );

		dbDelta( "CREATE TABLE {$t['events']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			event_key CHAR(64) NOT NULL,
			visitor_id CHAR(36) NOT NULL,
			session_id CHAR(36) NOT NULL,
			campaign_id BIGINT UNSIGNED DEFAULT NULL,
			event_type VARCHAR(40) NOT NULL DEFAULT 'landing_view',
			landing_page_url TEXT DEFAULT NULL,
			utm_source VARCHAR(191) DEFAULT NULL,
			utm_medium VARCHAR(191) DEFAULT NULL,
			utm_campaign VARCHAR(191) DEFAULT NULL,
			utm_content VARCHAR(191) DEFAULT NULL,
			utm_term VARCHAR(191) DEFAULT NULL,
			ip_address VARCHAR(45) DEFAULT NULL,
			user_agent VARCHAR(500) DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY event_key (event_key),
			KEY session_created (session_id,created_at),
			KEY visitor_created (visitor_id,created_at),
			KEY campaign_created (campaign_id,created_at),
			KEY event_type (event_type)
		) $charset;" );

		dbDelta( "CREATE TABLE {$t['attributions']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			subject_type VARCHAR(40) NOT NULL,
			subject_id BIGINT UNSIGNED NOT NULL,
			identity_hash CHAR(64) DEFAULT NULL,
			visitor_id CHAR(36) DEFAULT NULL,
			session_id CHAR(36) DEFAULT NULL,
			first_campaign_id BIGINT UNSIGNED DEFAULT NULL,
			last_campaign_id BIGINT UNSIGNED DEFAULT NULL,
			first_touch_source VARCHAR(191) DEFAULT NULL,
			first_touch_medium VARCHAR(191) DEFAULT NULL,
			first_touch_campaign VARCHAR(191) DEFAULT NULL,
			first_touch_content VARCHAR(191) DEFAULT NULL,
			first_touch_term VARCHAR(191) DEFAULT NULL,
			last_touch_source VARCHAR(191) DEFAULT NULL,
			last_touch_medium VARCHAR(191) DEFAULT NULL,
			last_touch_campaign VARCHAR(191) DEFAULT NULL,
			last_touch_content VARCHAR(191) DEFAULT NULL,
			last_touch_term VARCHAR(191) DEFAULT NULL,
			attributed_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY subject (subject_type,subject_id),
			KEY identity_attributed (identity_hash,attributed_at),
			KEY first_campaign_attributed (first_campaign_id,attributed_at),
			KEY last_campaign_attributed (last_campaign_id,attributed_at)
		) $charset;" );

		dbDelta( "CREATE TABLE {$t['conversions']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			attribution_id BIGINT UNSIGNED NOT NULL,
			conversion_type VARCHAR(40) NOT NULL,
			subject_type VARCHAR(40) NOT NULL,
			subject_id BIGINT UNSIGNED NOT NULL,
			identity_hash CHAR(64) DEFAULT NULL,
			first_campaign_id BIGINT UNSIGNED DEFAULT NULL,
			last_campaign_id BIGINT UNSIGNED DEFAULT NULL,
			revenue_amount DECIMAL(20,2) NOT NULL DEFAULT 0,
			occurred_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY subject_conversion (conversion_type,subject_type,subject_id),
			KEY first_campaign_type_date (first_campaign_id,conversion_type,occurred_at),
			KEY last_campaign_type_date (last_campaign_id,conversion_type,occurred_at),
			KEY identity_type (identity_hash,conversion_type)
		) $charset;" );

		dbDelta( "CREATE TABLE {$t['assignments']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			partner_user_id BIGINT UNSIGNED NOT NULL,
			campaign_id BIGINT UNSIGNED NOT NULL,
			created_by BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY partner_campaign (partner_user_id,campaign_id),
			KEY campaign_partner (campaign_id,partner_user_id)
		) $charset;" );

		dbDelta( "CREATE TABLE {$t['tokens']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			partner_user_id BIGINT UNSIGNED NOT NULL,
			token_hash CHAR(64) NOT NULL,
			token_prefix VARCHAR(20) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'active',
			last_used_at DATETIME DEFAULT NULL,
			last_ip_hash CHAR(64) DEFAULT NULL,
			expires_at DATETIME DEFAULT NULL,
			created_by BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY token_hash (token_hash),
			KEY partner_status (partner_user_id,status),
			KEY status_expires (status,expires_at)
		) $charset;" );

		$this->table_cache = [];
		update_option( 'sn_campaign_db_version', self::DB_VERSION, false );
		if ( get_option( 'sn_campaign_tracking_enabled', null ) === null ) {
			add_option( 'sn_campaign_tracking_enabled', '1', '', false );
		}
	}

	private function register_role(): void {
		add_role( 'sn_campaign_partner', 'Campaign Partner (پارتنر کمپین)', [
			'read' => true,
			'sn_view_campaign_reports' => true,
		] );
		$role = get_role( 'sn_campaign_partner' );
		if ( ! $role ) { return; }
		$role->add_cap( 'read' );
		$role->add_cap( 'sn_view_campaign_reports' );
		foreach ( [ 'edit_posts','delete_posts','publish_posts','upload_files','edit_pages','delete_pages','manage_options','list_users','create_users','edit_users','delete_users' ] as $cap ) {
			$role->remove_cap( $cap );
		}
	}

	public function register_shortcode(): void {
		add_shortcode( 'sn_campaign_partner_portal', [ $this, 'render_partner_portal' ] );
	}

	private function tracking_enabled(): bool {
		return (string) get_option( 'sn_campaign_tracking_enabled', '1' ) !== '0';
	}

	private function should_track_front_request(): bool {
		if ( ! $this->tracking_enabled() || is_admin() || wp_doing_ajax() || wp_doing_cron() ) { return false; }
		if ( defined( 'REST_REQUEST' ) && REST_REQUEST ) { return false; }
		if ( is_feed() || is_robots() || is_trackback() ) { return false; }
		if ( is_user_logged_in() ) {
			$user = wp_get_current_user();
			$staff_roles = [ 'administrator','sn_seller','sn_supervisor','sn_converter','sn_after_sales','sn_sales_manager','sn_financial','sn_financial_approval','sn_campaign_partner' ];
			if ( current_user_can( 'manage_options' ) || array_intersect( $staff_roles, (array) $user->roles ) ) { return false; }
		}
		$ua = strtolower( (string) ( $_SERVER['HTTP_USER_AGENT'] ?? '' ) );
		return ! preg_match( '/bot|crawler|spider|slurp|bingpreview|facebookexternalhit/', $ua );
	}

	public function enqueue_tracking_asset(): void {
		$partner_page_id = (int) get_option( 'sn_campaign_partner_page_id', 0 );
		if ( $partner_page_id > 0 && is_page( $partner_page_id ) ) { $this->enqueue_portal_asset(); }
		if ( ! $this->should_track_front_request() ) { return; }
		$file = SN_PLUGIN_DIR . 'assets/js/campaign-tracking.js';
		wp_enqueue_script( 'sn-campaign-tracking', SN_PLUGIN_URL . 'assets/js/campaign-tracking.js', [], SN_VERSION . '-' . ( file_exists( $file ) ? (string) filemtime( $file ) : '0' ), true );
		wp_localize_script( 'sn-campaign-tracking', 'SN_CAMPAIGN_TRACKING', [
			'ajaxUrl' => admin_url( 'admin-ajax.php' ),
			'action' => 'sn_campaign_track',
		] );
	}

	private function normalize_utm_value( $value ): string {
		$value = sanitize_text_field( wp_unslash( (string) $value ) );
		$value = preg_replace( '/[\x00-\x1F\x7F]/u', '', $value );
		return function_exists( 'mb_substr' ) ? mb_substr( trim( $value ), 0, 191 ) : substr( trim( $value ), 0, 191 );
	}

	private function request_utm( ?array $source = null ): array {
		$source = is_array( $source ) ? $source : $_GET;
		$out = [];
		foreach ( [ 'source','medium','campaign','content','term' ] as $key ) {
			$out[ $key ] = $this->normalize_utm_value( $source[ 'utm_' . $key ] ?? '' );
		}
		return $out;
	}

	private function empty_utm(): array {
		return [ 'source' => '', 'medium' => '', 'campaign' => '', 'content' => '', 'term' => '' ];
	}

	/** Read campaign values from both a normal query string and legacy #form?utm links. */
	private function utm_from_landing_url( string $url ): array {
		$out = $this->empty_utm();
		$url = $this->safe_landing_url( $url );
		if ( $url === '' ) { return $out; }
		$candidates = [];
		$query = (string) wp_parse_url( $url, PHP_URL_QUERY );
		if ( $query !== '' ) { $candidates[] = $query; }
		$fragment = (string) wp_parse_url( $url, PHP_URL_FRAGMENT );
		if ( $fragment !== '' ) {
			$question = strpos( $fragment, '?' );
			$fragment_query = $question === false ? $fragment : substr( $fragment, $question + 1 );
			if ( strpos( $fragment_query, 'utm_' ) !== false ) { $candidates[] = $fragment_query; }
		}
		foreach ( $candidates as $candidate ) {
			$params = [];
			parse_str( $candidate, $params );
			$parsed = $this->request_utm( is_array( $params ) ? $params : [] );
			foreach ( $out as $key => $value ) {
				if ( $value === '' && (string) ( $parsed[ $key ] ?? '' ) !== '' ) { $out[ $key ] = (string) $parsed[ $key ]; }
			}
		}
		return $out;
	}

	/**
	 * Produce one authoritative form snapshot. URL values override hidden POST
	 * fields because a full-page cache can serve stale hidden inputs. The same-site
	 * HTTP referer is evaluated last for a normal query, while the browser landing
	 * URL still recovers legacy UTM values kept inside a hash fragment.
	 */
	public static function submission_snapshot( array $source, string $url = '' ): array {
		$self = self::instance();
		$explicit = $self->request_utm( $source );
		$url_candidates = [
			$url,
			(string) ( $source['sn_campaign_landing_url'] ?? '' ),
			(string) ( $_SERVER['HTTP_REFERER'] ?? '' ),
		];
		$landing_url = '';
		foreach ( $url_candidates as $candidate ) {
			$candidate = $self->safe_landing_url( esc_url_raw( wp_unslash( (string) $candidate ) ) );
			if ( $candidate === '' ) { continue; }
			if ( $landing_url === '' ) { $landing_url = $candidate; }
			$from_url = $self->utm_from_landing_url( $candidate );
			foreach ( $explicit as $key => $value ) {
				if ( (string) ( $from_url[ $key ] ?? '' ) !== '' ) { $explicit[ $key ] = (string) $from_url[ $key ]; }
			}
		}
		$snapshot = [ 'sn_campaign_landing_url' => $landing_url ];
		foreach ( $explicit as $key => $value ) { $snapshot[ 'utm_' . $key ] = (string) $value; }
		return $snapshot;
	}

	private function has_utm( array $utm ): bool {
		foreach ( $utm as $value ) { if ( (string) $value !== '' ) { return true; } }
		return false;
	}

	private function valid_uuid( $value ): string {
		$value = strtolower( sanitize_text_field( wp_unslash( (string) $value ) ) );
		return preg_match( '/^[a-f0-9]{8}-[a-f0-9]{4}-[1-5][a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$/', $value ) ? $value : '';
	}

	private function set_tracking_cookie( string $name, string $value, int $expires ): void {
		if ( headers_sent() ) { return; }
		$options = [
			'expires' => $expires,
			'path' => ( defined( 'COOKIEPATH' ) && COOKIEPATH ) ? COOKIEPATH : '/',
			'domain' => ( defined( 'COOKIE_DOMAIN' ) && COOKIE_DOMAIN ) ? COOKIE_DOMAIN : '',
			'secure' => is_ssl(),
			'httponly' => true,
			'samesite' => 'Lax',
		];
		setcookie( $name, $value, $options );
		$_COOKIE[ $name ] = $value;
	}

	private function current_ip(): string {
		if ( (string) get_option( 'sn_campaign_store_ip', '0' ) !== '1' ) { return ''; }
		$ip = sanitize_text_field( (string) ( $_SERVER['REMOTE_ADDR'] ?? '' ) );
		return filter_var( $ip, FILTER_VALIDATE_IP ) ? $ip : '';
	}

	private function current_user_agent(): string {
		$ua = sanitize_text_field( (string) ( $_SERVER['HTTP_USER_AGENT'] ?? '' ) );
		return function_exists( 'mb_substr' ) ? mb_substr( $ua, 0, 500 ) : substr( $ua, 0, 500 );
	}

	private function safe_landing_url( string $raw = '' ): string {
		if ( $raw === '' ) {
			$uri = wp_unslash( (string) ( $_SERVER['REQUEST_URI'] ?? '/' ) );
			$scheme = (string) wp_parse_url( home_url( '/' ), PHP_URL_SCHEME );
			$host = (string) wp_parse_url( home_url( '/' ), PHP_URL_HOST );
			$port = (int) wp_parse_url( home_url( '/' ), PHP_URL_PORT );
			$raw = $scheme . '://' . $host . ( $port > 0 ? ':' . $port : '' ) . '/' . ltrim( $uri, '/' );
		}
		$raw = esc_url_raw( $raw );
		if ( $raw === '' ) { return ''; }
		$home_host = strtolower( (string) wp_parse_url( home_url( '/' ), PHP_URL_HOST ) );
		$url_host = strtolower( (string) wp_parse_url( $raw, PHP_URL_HOST ) );
		if ( $home_host === '' || $url_host !== $home_host ) { return ''; }
		$raw = remove_query_arg( [ 'access_token','token','password','pass','otp','code','nonce','_wpnonce','key','authorization' ], $raw );
		return function_exists( 'mb_substr' ) ? mb_substr( $raw, 0, 2000 ) : substr( $raw, 0, 2000 );
	}

	private function campaign_key( array $utm ): string {
		$normalized = [];
		foreach ( [ 'source','medium','campaign','content','term' ] as $key ) {
			$value = (string) ( $utm[ $key ] ?? '' );
			$normalized[ $key ] = function_exists( 'mb_strtolower' ) ? mb_strtolower( trim( $value ) ) : strtolower( trim( $value ) );
		}
		return hash( 'sha256', wp_json_encode( $normalized, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ) );
	}

	private function get_or_create_campaign( array $utm ): int {
		if ( ! $this->has_utm( $utm ) ) { return 0; }
		global $wpdb;
		$t = $this->tables();
		if ( ! $this->table_exists( $t['campaigns'] ) ) { $this->install_schema(); }
		$key = $this->campaign_key( $utm );
		$id = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$t['campaigns']} WHERE campaign_key=%s LIMIT 1", $key ) );
		if ( $id > 0 ) { return $id; }
		$title = (string) ( $utm['campaign'] ?: $utm['source'] ?: 'کمپین بدون نام' );
		$wpdb->insert( $t['campaigns'], [
			'campaign_key' => $key,
			'title' => $title,
			'utm_source' => $utm['source'] ?: null,
			'utm_medium' => $utm['medium'] ?: null,
			'utm_campaign' => $utm['campaign'] ?: null,
			'utm_content' => $utm['content'] ?: null,
			'utm_term' => $utm['term'] ?: null,
			'currency' => function_exists( 'get_woocommerce_currency' ) ? get_woocommerce_currency() : 'IRR',
			'created_at' => current_time( 'mysql' ),
			'updated_at' => current_time( 'mysql' ),
		] );
		if ( $wpdb->insert_id ) { return (int) $wpdb->insert_id; }
		return (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$t['campaigns']} WHERE campaign_key=%s LIMIT 1", $key ) );
	}

	private function mysql_timestamp( string $value ): int {
		return SN_Helpers::site_mysql_timestamp( $value, 0 );
	}

	/**
	 * Return a Gregorian Tehran-business date without going through wp_date().
	 *
	 * Some Persian date plugins filter wp_date() to return a Jalali value. The
	 * campaign tables, however, store DATETIME values in the Gregorian format.
	 * Using wp_date() for SQL boundaries therefore turns 1405/05/11 into the
	 * Gregorian year 1405 and makes every report metric appear as zero.
	 */
	private function site_date( string $format = 'Y-m-d', int $offset_days = 0 ): string {
		try {
			$date = SN_Helpers::tehran_now();
			if ( $offset_days !== 0 ) { $date = $date->modify( ( $offset_days > 0 ? '+' : '' ) . $offset_days . ' days' ); }
			return $date->format( $format );
		} catch ( Throwable $e ) {
			return SN_Helpers::tehran_format( $format, time() + ( $offset_days * DAY_IN_SECONDS ) );
		}
	}

	/** Normalize either a Jalali UI date or an ISO Gregorian date for SQL. */
	private function report_date( $value, string $fallback ): string {
		$value = sanitize_text_field( wp_unslash( (string) $value ) );
		if ( class_exists( 'SN_Helpers' ) && method_exists( 'SN_Helpers', 'to_english_nums' ) ) {
			$value = SN_Helpers::to_english_nums( $value );
		}
		$value = trim( str_replace( [ '/', '.', ' ' ], '-', $value ) );
		if ( ! preg_match( '/^(\d{4})-(\d{1,2})-(\d{1,2})$/', $value, $parts ) ) { return $fallback; }
		$input_year = (int) $parts[1];
		$input_month = (int) $parts[2];
		$input_day = (int) $parts[3];
		if ( $input_month < 1 || $input_month > 12 || $input_day < 1 ) { return $fallback; }
		if ( $input_year <= 1700 ) {
			$max_day = $input_month <= 6 ? 31 : ( $input_month <= 11 ? 30 : 30 );
			if ( $input_day > $max_day ) { return $fallback; }
		}
		$converted = class_exists( 'SN_Helpers' ) && method_exists( 'SN_Helpers', 'jalali_to_gregorian_date' )
			? SN_Helpers::jalali_to_gregorian_date( $value )
			: $value;
		if ( ! preg_match( '/^\d{4}-\d{2}-\d{2}$/', (string) $converted ) ) { return $fallback; }
		try {
			$parsed = DateTimeImmutable::createFromFormat( '!Y-m-d', (string) $converted, new DateTimeZone( 'UTC' ) );
			$errors = DateTimeImmutable::getLastErrors();
			if ( ! $parsed || ( $errors !== false && ( $errors['warning_count'] > 0 || $errors['error_count'] > 0 ) ) || $parsed->format( 'Y-m-d' ) !== $converted ) {
				return $fallback;
			}
		} catch ( Throwable $e ) {
			return $fallback;
		}
		return (string) $converted;
	}

	/** Render the internal Gregorian boundary as the site's Jalali input value. */
	private function report_date_input_value( string $gregorian ): string {
		if ( class_exists( 'SN_Helpers' ) && method_exists( 'SN_Helpers', 'gregorian_to_jalali_input_value' ) ) {
			$jalali = SN_Helpers::gregorian_to_jalali_input_value( $gregorian );
			if ( $jalali !== '' ) { return $jalali; }
		}
		return $gregorian;
	}

	public function capture_landing_request(): void {
		if ( ! $this->should_track_front_request() ) { return; }
		$this->capture_request( $this->safe_landing_url(), $this->request_utm(), 'server' );
	}

	private function capture_request( string $url, array $utm, string $source ): bool {
		if ( $this->capturing || ! $this->tracking_enabled() ) { return false; }
		$this->capturing = true;
		global $wpdb;
		$t = $this->tables();
		if ( ! $this->table_exists( $t['sessions'] ) ) { $this->install_schema(); }
		$url = $this->safe_landing_url( $url );
		$utm = $this->request_utm( [
			'utm_source' => $utm['source'] ?? '',
			'utm_medium' => $utm['medium'] ?? '',
			'utm_campaign' => $utm['campaign'] ?? '',
			'utm_content' => $utm['content'] ?? '',
			'utm_term' => $utm['term'] ?? '',
		] );
		$has_utm = $this->has_utm( $utm );
		$visitor_id = $this->valid_uuid( $_COOKIE[ self::VISITOR_COOKIE ] ?? '' );
		$session_id = $this->valid_uuid( $_COOKIE[ self::SESSION_COOKIE ] ?? '' );
		$session = $session_id !== '' ? $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['sessions']} WHERE session_id=%s LIMIT 1", $session_id ) ) : null;
		$active_cutoff = time() - self::SESSION_TTL;
		if ( $session && $this->mysql_timestamp( (string) $session->last_seen_at ) < $active_cutoff ) { $session = null; $session_id = ''; }
		if ( ! $has_utm && ! $session ) { $this->capturing = false; return false; }
		if ( $session && $this->valid_uuid( (string) $session->visitor_id ) !== '' ) { $visitor_id = (string) $session->visitor_id; }
		if ( $visitor_id === '' ) { $visitor_id = wp_generate_uuid4(); }
		if ( $session_id === '' ) { $session_id = wp_generate_uuid4(); }
		$campaign_id = $has_utm ? $this->get_or_create_campaign( $utm ) : (int) ( $session->last_campaign_id ?? 0 );
		if ( $campaign_id < 1 ) { $this->capturing = false; return false; }
		$now = current_time( 'mysql' );
		$ip = $this->current_ip();
		$ua = $this->current_user_agent();

		if ( ! $session ) {
			$insert = [
				'visitor_id' => $visitor_id,
				'session_id' => $session_id,
				'first_campaign_id' => $campaign_id,
				'last_campaign_id' => $campaign_id,
				'landing_page_url' => $url,
				'ip_address' => $ip ?: null,
				'user_agent' => $ua ?: null,
				'started_at' => $now,
				'last_seen_at' => $now,
				'created_at' => $now,
				'updated_at' => $now,
			];
			foreach ( [ 'source','medium','campaign','content','term' ] as $key ) {
				$insert[ 'first_utm_' . $key ] = $utm[ $key ] ?: null;
				$insert[ 'last_utm_' . $key ] = $utm[ $key ] ?: null;
			}
			if ( ! $wpdb->insert( $t['sessions'], $insert ) ) { $this->capturing = false; return false; }
			$session = (object) $insert;
		} else {
			$update = [ 'last_seen_at' => $now, 'updated_at' => $now ];
			if ( $has_utm ) {
				$update['last_campaign_id'] = $campaign_id;
				foreach ( [ 'source','medium','campaign','content','term' ] as $key ) { $update[ 'last_utm_' . $key ] = $utm[ $key ] ?: null; }
			} else {
				foreach ( [ 'source','medium','campaign','content','term' ] as $key ) { $utm[ $key ] = (string) ( $session->{ 'last_utm_' . $key } ?? '' ); }
			}
			$wpdb->update( $t['sessions'], $update, [ 'session_id' => $session_id ] );
			$campaign_id = (int) ( $update['last_campaign_id'] ?? $session->last_campaign_id );
		}

		$minute_bucket = $this->site_date( 'YmdHi' );
		$event_type = $source === 'browser' ? 'landing_view_browser' : ( $source === 'form' ? 'form_context' : ( $source === 'exit' ? 'landing_exit' : 'landing_view' ) );
		// A page can fire both visibilitychange and pagehide. Keep one exit event
		// per browser session/campaign so the report is a visitor/session metric,
		// not a count of duplicate unload callbacks.
		$event_key = $source === 'exit'
			? hash( 'sha256', implode( '|', [ 'landing_exit', $session_id, $campaign_id ] ) )
			: hash( 'sha256', implode( '|', [ $session_id, $url, $campaign_id, $minute_bucket ] ) );
		$wpdb->query( $wpdb->prepare(
			"INSERT IGNORE INTO {$t['events']} (event_key,visitor_id,session_id,campaign_id,event_type,landing_page_url,utm_source,utm_medium,utm_campaign,utm_content,utm_term,ip_address,user_agent,created_at) VALUES (%s,%s,%s,%d,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s)",
			$event_key, $visitor_id, $session_id, $campaign_id, $event_type, $url,
			$utm['source'], $utm['medium'], $utm['campaign'], $utm['content'], $utm['term'], $ip, $ua, $now
		) );
		$this->set_tracking_cookie( self::VISITOR_COOKIE, $visitor_id, time() + YEAR_IN_SECONDS );
		$this->set_tracking_cookie( self::SESSION_COOKIE, $session_id, time() + self::SESSION_TTL );
		$this->capturing = false;
		return true;
	}

	private function valid_tracking_origin(): bool {
		$home_host = strtolower( (string) wp_parse_url( home_url( '/' ), PHP_URL_HOST ) );
		foreach ( [ 'HTTP_ORIGIN','HTTP_REFERER' ] as $header ) {
			if ( empty( $_SERVER[ $header ] ) ) { continue; }
			$host = strtolower( (string) wp_parse_url( esc_url_raw( wp_unslash( (string) $_SERVER[ $header ] ) ), PHP_URL_HOST ) );
			return $host !== '' && hash_equals( $home_host, $host );
		}
		return false;
	}

	private function tracking_rate_limit_ok( string $client_key = '' ): bool {
		$ip = sanitize_text_field( (string) ( $_SERVER['REMOTE_ADDR'] ?? 'unknown' ) );
		$client_key = sanitize_text_field( $client_key );
		if ( ! preg_match( '/^[A-Za-z0-9_-]{16,80}$/', $client_key ) ) { $client_key = 'anonymous'; }
		// A browser-scoped key prevents every visitor behind the same CDN/NAT IP
		// from sharing one tiny rate-limit bucket. Origin validation still guards
		// the public endpoint and the IP remains part of the server-side key.
		$key = 'sn_campaign_track_v2_' . hash_hmac( 'sha256', $ip . '|' . $client_key, wp_salt( 'nonce' ) );
		$count = (int) get_transient( $key );
		if ( $count >= 120 ) { return false; }
		set_transient( $key, $count + 1, 15 * MINUTE_IN_SECONDS );
		return true;
	}

	public function ajax_track(): void {
		if ( strtoupper( (string) ( $_SERVER['REQUEST_METHOD'] ?? '' ) ) !== 'POST' || ! $this->valid_tracking_origin() ) {
			wp_send_json_error( [ 'message' => 'invalid_request' ], 403 );
		}
		if ( ! $this->tracking_rate_limit_ok( (string) ( $_POST['client_key'] ?? '' ) ) ) { wp_send_json_error( [ 'message' => 'rate_limited' ], 429 ); }
		$url = $this->safe_landing_url( esc_url_raw( wp_unslash( (string) ( $_POST['url'] ?? '' ) ) ) );
		$snapshot = self::submission_snapshot( $_POST, $url );
		$url = $this->safe_landing_url( (string) ( $snapshot['sn_campaign_landing_url'] ?? $url ) );
		$event = sanitize_key( wp_unslash( (string) ( $_POST['event'] ?? '' ) ) );
		$source = $event === 'landing_exit' ? 'exit' : 'browser';
		$captured = $this->capture_request( $url, $this->request_utm( $snapshot ), $source );
		wp_send_json_success( [ 'captured' => $captured ] );
	}

	/**
	 * Capture UTM values posted with a public form before its lead is persisted.
	 * This is the reliable fallback for cached landing pages, blocked beacons and
	 * legacy links whose UTM query was placed after the hash fragment.
	 */
	public static function capture_submission_context( array $source, string $url = '' ): bool {
		$self = self::instance();
		if ( ! $self->tracking_enabled() ) { return false; }
		$snapshot = self::submission_snapshot( $source, $url );
		$utm = $self->request_utm( $snapshot );
		if ( ! $self->has_utm( $utm ) ) { return false; }
		$url = $self->safe_landing_url( (string) ( $snapshot['sn_campaign_landing_url'] ?? '' ) );
		if ( $url === '' ) { $url = home_url( '/' ); }
		return $self->capture_request( $url, $utm, 'form' );
	}

	private function identity_hash( string $identity ): string {
		$identity = trim( $identity );
		if ( $identity === '' ) { return ''; }
		if ( is_email( $identity ) ) {
			$normalized = strtolower( sanitize_email( $identity ) );
		} else {
			$english = class_exists( 'SN_Helpers' ) ? SN_Helpers::to_english_nums( $identity ) : $identity;
			$digits = preg_replace( '/\D+/', '', $english );
			$normalized = $digits !== '' ? $digits : strtolower( sanitize_text_field( $identity ) );
		}
		return $normalized === '' ? '' : hash_hmac( 'sha256', $normalized, wp_salt( 'auth' ) );
	}

	private function attribution_window_days(): int {
		return max( 1, min( 3650, (int) get_option( 'sn_campaign_attribution_window_days', self::DEFAULT_WINDOW ) ) );
	}

	private function attribution_from_row( $row ): array {
		if ( ! $row ) { return []; }
		return [
			'attribution_id' => (int) ( $row->id ?? 0 ),
			'identity_hash' => (string) ( $row->identity_hash ?? '' ),
			'visitor_id' => (string) ( $row->visitor_id ?? '' ),
			'session_id' => (string) ( $row->session_id ?? '' ),
			'first_campaign_id' => (int) ( $row->first_campaign_id ?? 0 ),
			'last_campaign_id' => (int) ( $row->last_campaign_id ?? 0 ),
			'first_touch_source' => (string) ( $row->first_touch_source ?? '' ),
			'first_touch_medium' => (string) ( $row->first_touch_medium ?? '' ),
			'first_touch_campaign' => (string) ( $row->first_touch_campaign ?? '' ),
			'first_touch_content' => (string) ( $row->first_touch_content ?? '' ),
			'first_touch_term' => (string) ( $row->first_touch_term ?? '' ),
			'last_touch_source' => (string) ( $row->last_touch_source ?? '' ),
			'last_touch_medium' => (string) ( $row->last_touch_medium ?? '' ),
			'last_touch_campaign' => (string) ( $row->last_touch_campaign ?? '' ),
			'last_touch_content' => (string) ( $row->last_touch_content ?? '' ),
			'last_touch_term' => (string) ( $row->last_touch_term ?? '' ),
		];
	}

	private function session_attribution_context(): array {
		$session_id = $this->valid_uuid( $_COOKIE[ self::SESSION_COOKIE ] ?? '' );
		if ( $session_id === '' ) { return []; }
		global $wpdb;
		$t = $this->tables();
		if ( ! $this->table_exists( $t['sessions'] ) ) { return []; }
		$session = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['sessions']} WHERE session_id=%s LIMIT 1", $session_id ) );
		if ( ! $session || (int) $session->last_campaign_id < 1 || $this->mysql_timestamp( (string) $session->last_seen_at ) < time() - self::SESSION_TTL ) { return []; }
		$window_start = $this->site_date( 'Y-m-d H:i:s', -$this->attribution_window_days() );
		$first = $wpdb->get_row( $wpdb->prepare(
			"SELECT * FROM {$t['sessions']} WHERE visitor_id=%s AND first_campaign_id IS NOT NULL AND started_at>=%s ORDER BY started_at ASC,id ASC LIMIT 1",
			(string) $session->visitor_id,
			$window_start
		) );
		if ( ! $first ) { $first = $session; }
		return [
			'visitor_id' => (string) $session->visitor_id,
			'session_id' => (string) $session->session_id,
			'first_campaign_id' => (int) $first->first_campaign_id,
			'last_campaign_id' => (int) $session->last_campaign_id,
			'first_touch_source' => (string) $first->first_utm_source,
			'first_touch_medium' => (string) $first->first_utm_medium,
			'first_touch_campaign' => (string) $first->first_utm_campaign,
			'first_touch_content' => (string) $first->first_utm_content,
			'first_touch_term' => (string) $first->first_utm_term,
			'last_touch_source' => (string) $session->last_utm_source,
			'last_touch_medium' => (string) $session->last_utm_medium,
			'last_touch_campaign' => (string) $session->last_utm_campaign,
			'last_touch_content' => (string) $session->last_utm_content,
			'last_touch_term' => (string) $session->last_utm_term,
		];
	}

	/** Build a conversion-safe attribution fallback when a cached page or blocked cookie prevents a session row. */
	private function direct_attribution_context( array $source ): array {
		$utm = $this->request_utm( $source );
		if ( ! $this->has_utm( $utm ) ) { return []; }
		$campaign_id = $this->get_or_create_campaign( $utm );
		if ( $campaign_id < 1 ) { return []; }
		$visitor_id = $this->valid_uuid( $_COOKIE[ self::VISITOR_COOKIE ] ?? '' );
		$session_id = $this->valid_uuid( $_COOKIE[ self::SESSION_COOKIE ] ?? '' );
		if ( $visitor_id === '' ) { $visitor_id = wp_generate_uuid4(); }
		if ( $session_id === '' ) { $session_id = wp_generate_uuid4(); }
		$context = [
			'visitor_id' => $visitor_id,
			'session_id' => $session_id,
			'first_campaign_id' => $campaign_id,
			'last_campaign_id' => $campaign_id,
		];
		foreach ( [ 'source','medium','campaign','content','term' ] as $key ) {
			$context[ 'first_touch_' . $key ] = (string) $utm[ $key ];
			$context[ 'last_touch_' . $key ] = (string) $utm[ $key ];
		}
		return $context;
	}

	/** Current form UTM is always the last touch; a valid browser session may retain the first touch. */
	private function submission_attribution_context( array $snapshot ): array {
		$direct = $this->direct_attribution_context( $snapshot );
		$session = $this->session_attribution_context();
		if ( ! $direct ) { return $session; }
		if ( $session ) {
			$direct['visitor_id'] = (string) ( $session['visitor_id'] ?? $direct['visitor_id'] );
			$direct['session_id'] = (string) ( $session['session_id'] ?? $direct['session_id'] );
			$direct['first_campaign_id'] = (int) ( $session['first_campaign_id'] ?? $direct['first_campaign_id'] );
			foreach ( [ 'source','medium','campaign','content','term' ] as $key ) {
				$direct[ 'first_touch_' . $key ] = (string) ( $session[ 'first_touch_' . $key ] ?? $direct[ 'first_touch_' . $key ] );
			}
		}
		return $direct;
	}

	private function find_attribution_by_identity( string $identity_hash ) {
		if ( $identity_hash === '' ) { return null; }
		global $wpdb;
		$t = $this->tables();
		$window_start = $this->site_date( 'Y-m-d H:i:s', -$this->attribution_window_days() );
		return $wpdb->get_row( $wpdb->prepare(
			"SELECT * FROM {$t['attributions']} WHERE identity_hash=%s AND attributed_at>=%s ORDER BY attributed_at DESC,id DESC LIMIT 1",
			$identity_hash,
			$window_start
		) );
	}

	private function capture_subject_internal( string $subject_type, int $subject_id, string $identity = '', int $preferred_attribution_id = 0, array $context_override = [] ): int {
		$subject_type = sanitize_key( $subject_type );
		if ( $subject_type === '' || $subject_id < 1 ) { return 0; }
		global $wpdb;
		$t = $this->tables();
		if ( ! $this->table_exists( $t['attributions'] ) ) { $this->install_schema(); }
		$identity_hash = $this->identity_hash( $identity );
		$existing = $wpdb->get_row( $wpdb->prepare(
			"SELECT * FROM {$t['attributions']} WHERE subject_type=%s AND subject_id=%d LIMIT 1",
			$subject_type,
			$subject_id
		) );
		$preferred = $preferred_attribution_id > 0
			? $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['attributions']} WHERE id=%d LIMIT 1", $preferred_attribution_id ) )
			: null;
		if ( ! $preferred && $identity_hash !== '' ) { $preferred = $this->find_attribution_by_identity( $identity_hash ); }
		$current_context = $context_override ?: $this->session_attribution_context();
		$existing_context = $this->attribution_from_row( $existing );
		$preferred_context = $this->attribution_from_row( $preferred );
		if ( ! $current_context && ! $existing_context && ! $preferred_context ) { return 0; }

		$first = $existing_context ?: $preferred_context ?: $current_context;
		$last = $current_context ?: $preferred_context ?: $existing_context;
		$data = [
			'identity_hash' => $identity_hash ?: ( $existing_context['identity_hash'] ?? $preferred_context['identity_hash'] ?? null ),
			'visitor_id' => (string) ( $last['visitor_id'] ?? $first['visitor_id'] ?? '' ) ?: null,
			'session_id' => (string) ( $last['session_id'] ?? $first['session_id'] ?? '' ) ?: null,
			'first_campaign_id' => (int) ( $first['first_campaign_id'] ?? 0 ) ?: null,
			'last_campaign_id' => (int) ( $last['last_campaign_id'] ?? 0 ) ?: null,
			'attributed_at' => current_time( 'mysql' ),
			'updated_at' => current_time( 'mysql' ),
		];
		foreach ( [ 'source','medium','campaign','content','term' ] as $key ) {
			$data[ 'first_touch_' . $key ] = (string) ( $first[ 'first_touch_' . $key ] ?? '' ) ?: null;
			$data[ 'last_touch_' . $key ] = (string) ( $last[ 'last_touch_' . $key ] ?? '' ) ?: null;
		}
		if ( $existing ) {
			if ( false === $wpdb->update( $t['attributions'], $data, [ 'id' => (int) $existing->id ] ) ) { return 0; }
			return (int) $existing->id;
		}
		$data['subject_type'] = $subject_type;
		$data['subject_id'] = $subject_id;
		$data['created_at'] = current_time( 'mysql' );
		if ( ! $wpdb->insert( $t['attributions'], $data ) ) { return 0; }
		return (int) $wpdb->insert_id;
	}

	private function record_conversion( int $attribution_id, string $conversion_type, string $subject_type, int $subject_id, float $revenue = 0.0, string $occurred_at = '' ): bool {
		if ( $attribution_id < 1 || $subject_id < 1 ) { return false; }
		$conversion_type = sanitize_key( $conversion_type );
		$subject_type = sanitize_key( $subject_type );
		if ( ! in_array( $conversion_type, [
			'registration','lead','approved_lead','active_customer','sale','application_started','otp_verified',
			'gateway_started','gateway_cancelled','funnel_abandoned','lead_saved','invoice_created',
			'awaiting_payment','popup_cancelled','payment_paid','pending_finance','payment_rejected',
			'gateway_failed','otp_cancelled','invoice_failed',
		], true ) ) { return false; }
		global $wpdb;
		$t = $this->tables();
		$attr = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['attributions']} WHERE id=%d LIMIT 1", $attribution_id ) );
		if ( ! $attr ) { return false; }
		$occurred_at = $occurred_at !== '' ? $occurred_at : current_time( 'mysql' );
		$sql = "INSERT INTO {$t['conversions']} (attribution_id,conversion_type,subject_type,subject_id,identity_hash,first_campaign_id,last_campaign_id,revenue_amount,occurred_at,created_at,updated_at)
			VALUES (%d,%s,%s,%d,%s,%d,%d,%f,%s,%s,%s)
			ON DUPLICATE KEY UPDATE attribution_id=VALUES(attribution_id),identity_hash=VALUES(identity_hash),first_campaign_id=VALUES(first_campaign_id),last_campaign_id=VALUES(last_campaign_id),revenue_amount=VALUES(revenue_amount),occurred_at=VALUES(occurred_at),updated_at=VALUES(updated_at)";
		return false !== $wpdb->query( $wpdb->prepare(
			$sql,
			$attribution_id,
			$conversion_type,
			$subject_type,
			$subject_id,
			(string) $attr->identity_hash,
			(int) $attr->first_campaign_id,
			(int) $attr->last_campaign_id,
			max( 0, $revenue ),
			$occurred_at,
			current_time( 'mysql' ),
			current_time( 'mysql' )
		) );
	}

	/**
	 * Return the campaign metrics represented by one Marketing Dot event.
	 * Aggregate funnel metrics are kept for compatibility while the exact
	 * statuses requested by the dashboard are stored as separate conversions.
	 */
	private function marketing_event_conversion_types( string $event ): array {
		$event = sanitize_key( $event );
		$map = [
			'continue_clicked' => [ 'application_started' ],
			'lead_saved' => [ 'application_started', 'lead_saved' ],
			'invoice_created' => [ 'invoice_created', 'awaiting_payment' ],
			'invoice_reused' => [ 'awaiting_payment' ],
			'invoice_failed' => [ 'invoice_failed', 'funnel_abandoned' ],
			'popup_cancelled' => [ 'popup_cancelled', 'funnel_abandoned' ],
			'otp_cancelled' => [ 'otp_cancelled', 'funnel_abandoned' ],
			'otp_verified' => [ 'otp_verified' ],
			'gateway_started' => [ 'gateway_started' ],
			'gateway_cancelled' => [ 'gateway_cancelled', 'funnel_abandoned' ],
			'gateway_failed' => [ 'gateway_failed', 'funnel_abandoned' ],
			'pending_finance' => [ 'pending_finance' ],
			'payment_rejected' => [ 'payment_rejected', 'funnel_abandoned' ],
			'payment_paid' => [ 'payment_paid' ],
			'awaiting_payment' => [ 'awaiting_payment' ],
			'payment_not_done' => [ 'lead_saved' ],
		];
		return $map[ $event ] ?? [];
	}

	/** Backfill exact status conversions for events written before this report version. */
	private function repair_marketing_funnel_conversions( int $limit = 500 ): void {
		global $wpdb;
		$t = $this->tables();
		$marketing_events = $wpdb->prefix . 'sn_dot_marketing_events';
		if ( ! $this->table_exists( $marketing_events ) || ! $this->table_exists( $t['attributions'] ) || ! $this->table_exists( $t['conversions'] ) ) { return; }
		$limit = max( 1, min( 2000, $limit ) );
		$rows = $wpdb->get_results( $wpdb->prepare(
			"SELECT e.marketing_lead_id,e.event_type,e.created_at,a.id attribution_id
			FROM {$marketing_events} e
			INNER JOIN {$t['attributions']} a ON a.subject_type='marketing_lead' AND a.subject_id=e.marketing_lead_id
			WHERE e.event_type IN ('continue_clicked','lead_saved','invoice_created','invoice_reused','invoice_failed','popup_cancelled','otp_cancelled','otp_verified','gateway_started','gateway_cancelled','gateway_failed','pending_finance','payment_rejected','payment_paid','awaiting_payment','payment_not_done')
			ORDER BY e.id DESC LIMIT %d",
			$limit
		) ) ?: [];
		foreach ( $rows as $row ) {
			foreach ( $this->marketing_event_conversion_types( (string) $row->event_type ) as $conversion_type ) {
				$this->record_conversion( (int) $row->attribution_id, $conversion_type, 'marketing_lead', (int) $row->marketing_lead_id, 0, (string) $row->created_at );
			}
		}
	}

	public static function capture_lead( int $lead_id, string $identity = '' ): int {
		$self = self::instance();
		$attr_id = $self->capture_subject_internal( 'lead', $lead_id, $identity );
		if ( $attr_id > 0 ) { $self->record_conversion( $attr_id, 'lead', 'lead', $lead_id ); }
		return $attr_id;
	}

	public static function mark_lead_status( int $lead_id, string $status ): void {
		$status = trim( sanitize_text_field( $status ) );
		if ( $lead_id < 1 || $status === '' ) { return; }
		$lower = function_exists( 'mb_strtolower' ) ? mb_strtolower( $status ) : strtolower( $status );
		if ( preg_match( '/نشده|رد|ناموفق|rejected|not[_ -]?approved|unqualified/u', $lower ) ) { return; }
		if ( ! preg_match( '/تایید|تأیید|واجد|qualified|approved|converted/u', $lower ) ) { return; }
		$self = self::instance();
		global $wpdb;
		$t = $self->tables();
		$attr_id = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$t['attributions']} WHERE subject_type='lead' AND subject_id=%d LIMIT 1", $lead_id ) );
		if ( $attr_id > 0 ) { $self->record_conversion( $attr_id, 'approved_lead', 'lead', $lead_id ); }
	}

	public static function capture_marketing_registration( int $submission_id, int $crm_lead_id = 0, string $identity = '', array $source = [] ): int {
		$self = self::instance();
		$context = [];
		if ( $source ) {
			$url = esc_url_raw( wp_unslash( (string) ( $source['sn_campaign_landing_url'] ?? '' ) ) );
			$snapshot = self::submission_snapshot( $source, $url );
			self::capture_submission_context( $snapshot, (string) ( $snapshot['sn_campaign_landing_url'] ?? '' ) );
			$context = $self->submission_attribution_context( $snapshot );
		}
		$attr_id = $self->capture_subject_internal( 'marketing_lead', $submission_id, $identity, 0, $context );
		if ( $attr_id > 0 ) {
			$self->record_conversion( $attr_id, 'lead', 'marketing_lead', $submission_id );
			$self->record_conversion( $attr_id, 'registration', 'marketing_registration', $submission_id );
		}
		if ( $crm_lead_id > 0 ) {
			// Preserve the CRM attribution bridge for a later invoice, but do not
			// record a second lead conversion for the same form submission. The
			// marketing submission itself is the canonical lead event and permits
			// repeat submissions from the same phone to be counted independently.
			$self->capture_subject_internal( 'lead', $crm_lead_id, $identity, $attr_id );
		}
		return $attr_id;
	}

	/** Retry missing attribution and idempotently restore missing lead conversions. */
	private function repair_unattributed_marketing_submissions( int $limit = 100 ): void {
		global $wpdb;
		$t = $this->tables();
		$marketing_table = $wpdb->prefix . 'sn_dot_marketing_leads';
		if ( ! $this->table_exists( $marketing_table ) || ! $this->table_exists( $t['attributions'] ) || ! $this->table_exists( $t['conversions'] ) ) { return; }
		$columns = (array) $wpdb->get_col( "SHOW COLUMNS FROM {$marketing_table}" );
		$required = [ 'campaign_attribution_id','campaign_attribution_status','campaign_attributed_at','campaign_landing_url','utm_source','utm_medium','utm_campaign','utm_content','utm_term' ];
		if ( array_diff( $required, $columns ) ) { return; }
		$limit = max( 1, min( 500, $limit ) );
		$rows = $wpdb->get_results( $wpdb->prepare(
			"SELECT m.id,m.crm_lead_id,m.customer_phone,m.campaign_landing_url,m.utm_source,m.utm_medium,m.utm_campaign,m.utm_content,m.utm_term
			FROM {$marketing_table} m
			LEFT JOIN {$t['attributions']} a ON a.subject_type='marketing_lead' AND a.subject_id=m.id
			LEFT JOIN {$t['conversions']} lc ON lc.conversion_type='lead' AND lc.subject_type='marketing_lead' AND lc.subject_id=m.id
			LEFT JOIN {$t['conversions']} rc ON rc.conversion_type='registration' AND rc.subject_type='marketing_registration' AND rc.subject_id=m.id
			WHERE (a.id IS NULL AND (COALESCE(m.utm_source,'')<>'' OR COALESCE(m.utm_medium,'')<>'' OR COALESCE(m.utm_campaign,'')<>'' OR COALESCE(m.utm_content,'')<>'' OR COALESCE(m.utm_term,'')<>''))
			OR (a.id IS NOT NULL AND (lc.id IS NULL OR rc.id IS NULL))
			ORDER BY m.id DESC LIMIT %d",
			$limit
		), ARRAY_A ) ?: [];
		foreach ( $rows as $row ) {
			$source = [
				'sn_campaign_landing_url' => (string) $row['campaign_landing_url'],
				'utm_source' => (string) $row['utm_source'],
				'utm_medium' => (string) $row['utm_medium'],
				'utm_campaign' => (string) $row['utm_campaign'],
				'utm_content' => (string) $row['utm_content'],
				'utm_term' => (string) $row['utm_term'],
			];
			$attr_id = self::capture_marketing_registration( (int) $row['id'], (int) $row['crm_lead_id'], (string) $row['customer_phone'], $source );
			$wpdb->update( $marketing_table, [
				'campaign_attribution_id' => $attr_id > 0 ? $attr_id : null,
				'campaign_attribution_status' => $attr_id > 0 ? 'attributed' : 'pending',
				'campaign_attributed_at' => $attr_id > 0 ? current_time( 'mysql' ) : null,
			], [ 'id' => (int) $row['id'] ] );
		}
	}

	public static function capture_marketing_funnel_event( int $submission_id, string $event, string $occurred_at = '' ): void {
		$event = sanitize_key( $event );
		if ( $submission_id < 1 ) { return; }
		$self = self::instance();
		global $wpdb;
		$t = $self->tables();
		$self->repair_unattributed_marketing_submissions( 25 );
		$attr_id = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$t['attributions']} WHERE subject_type='marketing_lead' AND subject_id=%d LIMIT 1", $submission_id ) );
		if ( $attr_id < 1 ) { return; }
		foreach ( $self->marketing_event_conversion_types( $event ) as $conversion_type ) {
			$self->record_conversion( $attr_id, $conversion_type, 'marketing_lead', $submission_id, 0, $occurred_at );
		}
	}

	public function on_user_register( int $user_id, array $userdata = [] ): void {
		$user = get_user_by( 'id', $user_id );
		if ( ! $user ) { return; }
		$identity = (string) get_user_meta( $user_id, 'billing_phone', true );
		if ( $identity === '' ) { $identity = (string) ( $user->user_email ?: $user->user_login ); }
		$attr_id = $this->capture_subject_internal( 'registration', $user_id, $identity );
		if ( $attr_id > 0 ) { $this->record_conversion( $attr_id, 'registration', 'registration', $user_id ); }
	}

	private function invoice_amount( object $invoice ): float {
		foreach ( [ 'payment_total_amount','final_total','original_total','product_price' ] as $field ) {
			$value = isset( $invoice->{$field} ) ? (float) $invoice->{$field} : 0.0;
			if ( $value > 0 ) { return $value; }
		}
		return 0.0;
	}

	public function on_invoice_paid( int $invoice_id, $invoice ): void {
		global $wpdb;
		$invoice = is_object( $invoice ) ? $invoice : null;
		if ( ! $invoice || (int) ( $invoice->id ?? 0 ) !== $invoice_id ) {
			$invoice = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$wpdb->prefix}sn_invoices WHERE id=%d LIMIT 1", $invoice_id ) );
		}
		if ( ! $invoice ) { return; }
		$t = $this->tables();
		$preferred_id = 0;
		$marketing_submission_id = 0;
		$lead_id = (int) ( $invoice->lead_id ?? 0 );
		if ( $lead_id > 0 ) {
			$preferred_id = (int) $wpdb->get_var( $wpdb->prepare(
				"SELECT id FROM {$t['attributions']} WHERE subject_type='lead' AND subject_id=%d LIMIT 1",
				$lead_id
			) );
		}
		$identity = (string) ( $invoice->customer_phone ?? '' );
		$attr_id = $this->capture_subject_internal( 'invoice', $invoice_id, $identity, $preferred_id );
		$marketing_table = $wpdb->prefix . 'sn_dot_marketing_leads';
		if ( $this->table_exists( $marketing_table ) ) {
			$marketing_submission_id = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$marketing_table} WHERE invoice_id=%d LIMIT 1", $invoice_id ) );
			if ( $attr_id < 1 && $marketing_submission_id > 0 ) {
					$preferred_id = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$t['attributions']} WHERE subject_type='marketing_lead' AND subject_id=%d LIMIT 1", $marketing_submission_id ) );
					$attr_id = $this->capture_subject_internal( 'invoice', $invoice_id, $identity, $preferred_id );
			}
		}
		if ( $attr_id < 1 ) { return; }
		$occurred = (string) ( $invoice->payment_completed_at ?? $invoice->paid_at ?? $invoice->approved_at ?? current_time( 'mysql' ) );
		$this->record_conversion( $attr_id, 'active_customer', 'invoice', $invoice_id, 0, $occurred );
		$this->record_conversion( $attr_id, 'sale', 'invoice', $invoice_id, $this->invoice_amount( $invoice ), $occurred );
		if ( $lead_id > 0 ) {
			$lead_attr = $preferred_id ?: $this->capture_subject_internal( 'lead', $lead_id, $identity, $attr_id );
			if ( $lead_attr > 0 ) { $this->record_conversion( $lead_attr, 'approved_lead', 'lead', $lead_id, 0, $occurred ); }
		}
		if ( $marketing_submission_id > 0 ) {
			$marketing_attr = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$t['attributions']} WHERE subject_type='marketing_lead' AND subject_id=%d LIMIT 1", $marketing_submission_id ) );
			if ( $marketing_attr > 0 ) { $this->record_conversion( $marketing_attr, 'approved_lead', 'marketing_lead', $marketing_submission_id, 0, $occurred ); }
		}
	}

	private function normalize_filters( array $source ): array {
		$today = $this->site_date( 'Y-m-d' );
		$default_from = $this->site_date( 'Y-m-d', -29 );
		$from = $this->report_date( $source['date_from'] ?? '', $default_from );
		$to = $this->report_date( $source['date_to'] ?? '', $today );
		if ( $from > $to ) { $tmp = $from; $from = $to; $to = $tmp; }
		$filters = [
			'date_from' => $from,
			'date_to' => $to,
			'attribution_model' => sanitize_key( wp_unslash( (string) ( $source['attribution_model'] ?? 'last' ) ) ) === 'first' ? 'first' : 'last',
		];
		foreach ( [ 'source','medium','campaign','content','term' ] as $key ) {
			$filters[ $key ] = $this->normalize_utm_value( $source[ $key ] ?? '' );
		}
		return $filters;
	}

	private function campaign_ids_for_scope( array $filters, int $partner_user_id = 0 ): array {
		global $wpdb;
		$t = $this->tables();
		$where = [ '1=1' ];
		$args = [];
		$join = '';
		if ( $partner_user_id > 0 ) {
			$join = " INNER JOIN {$t['assignments']} a ON a.campaign_id=c.id ";
			$where[] = 'a.partner_user_id=%d';
			$args[] = $partner_user_id;
		}
		foreach ( [ 'source','medium','campaign','content','term' ] as $key ) {
			if ( (string) ( $filters[ $key ] ?? '' ) === '' ) { continue; }
			$where[] = 'c.utm_' . $key . '=%s';
			$args[] = (string) $filters[ $key ];
		}
		$sql = "SELECT DISTINCT c.id FROM {$t['campaigns']} c {$join} WHERE " . implode( ' AND ', $where ) . ' ORDER BY c.id ASC';
		$ids = $args ? $wpdb->get_col( $wpdb->prepare( $sql, ...$args ) ) : $wpdb->get_col( $sql );
		return array_values( array_unique( array_filter( array_map( 'absint', (array) $ids ) ) ) );
	}

	private function campaign_rows_by_ids( array $ids ): array {
		if ( ! $ids ) { return []; }
		global $wpdb;
		$t = $this->tables();
		$id_sql = implode( ',', array_map( 'absint', $ids ) );
		$rows = $wpdb->get_results( "SELECT * FROM {$t['campaigns']} WHERE id IN ({$id_sql}) ORDER BY id DESC", ARRAY_A ) ?: [];
		$out = [];
		foreach ( $rows as $row ) { $out[ (int) $row['id'] ] = $row; }
		return $out;
	}

	private function conversion_identity_sql(): string {
		return "CASE WHEN identity_hash IS NULL OR identity_hash='' THEN CONCAT(subject_type,':',subject_id) ELSE identity_hash END";
	}

	private function conversion_subject_sql(): string {
		return "CONCAT(subject_type,':',subject_id)";
	}

	private function analytics( array $filters, int $partner_user_id = 0 ): array {
		global $wpdb;
		$t = $this->tables();
		$this->repair_unattributed_marketing_submissions( 100 );
		$this->repair_marketing_funnel_conversions( 1000 );
		$ids = $this->campaign_ids_for_scope( $filters, $partner_user_id );
		$empty_totals = [
			'sessions' => 0, 'visitors' => 0, 'registrations' => 0, 'conversion_rate' => 0.0,
			'leads' => 0, 'approved_leads' => 0, 'active_customers' => 0, 'sales_count' => 0,
			'application_started' => 0, 'otp_verified' => 0, 'gateway_started' => 0, 'funnel_abandoned' => 0,
			'page_closed_without_lead' => 0, 'lead_saved' => 0, 'invoice_created' => 0, 'awaiting_payment' => 0,
			'popup_cancelled' => 0, 'gateway_cancelled' => 0, 'payment_paid' => 0, 'pending_finance' => 0, 'payment_rejected' => 0,
			'sales_amount' => 0.0, 'budget_amount' => 0.0, 'cost_amount' => 0.0, 'cpl' => 0.0, 'cpa' => 0.0,
		];
		if ( ! $ids ) { return [ 'filters' => $filters, 'campaign_ids' => [], 'rows' => [], 'totals' => $empty_totals ]; }
		$id_sql = implode( ',', array_map( 'absint', $ids ) );
		$model_column = $filters['attribution_model'] === 'first' ? 'first_campaign_id' : 'last_campaign_id';
		$from = $filters['date_from'] . ' 00:00:00';
		$to = $filters['date_to'] . ' 23:59:59';
		$campaigns = $this->campaign_rows_by_ids( $ids );
		$session_map = [];
		$session_sql = "SELECT {$model_column} campaign_id,COUNT(*) sessions,COUNT(DISTINCT visitor_id) visitors FROM {$t['sessions']} WHERE started_at BETWEEN %s AND %s AND {$model_column} IN ({$id_sql}) GROUP BY {$model_column}";
		foreach ( (array) $wpdb->get_results( $wpdb->prepare( $session_sql, $from, $to ), ARRAY_A ) as $row ) {
			$session_map[ (int) $row['campaign_id'] ] = [ 'sessions' => (int) $row['sessions'], 'visitors' => (int) $row['visitors'] ];
		}
		$page_exit_map = [];
		if ( $this->table_exists( $t['events'] ) && $this->table_exists( $t['attributions'] ) && $this->table_exists( $t['conversions'] ) ) {
			// landing_exit is emitted by the browser on pagehide. Exclude sessions
			// that later produced a Marketing Dot submission so this metric means
			// "opened and left without submitting a lead" rather than every exit.
			$page_exit_sql = "SELECT e.campaign_id,COUNT(DISTINCT e.session_id) total
				FROM {$t['events']} e
				WHERE e.event_type='landing_exit' AND e.created_at BETWEEN %s AND %s AND e.campaign_id IN ({$id_sql})
				AND NOT EXISTS (
					SELECT 1 FROM {$t['attributions']} a
					INNER JOIN {$t['conversions']} c ON c.attribution_id=a.id
					WHERE a.session_id=e.session_id AND c.conversion_type IN ('lead','registration','lead_saved')
				)
				GROUP BY e.campaign_id";
			foreach ( (array) $wpdb->get_results( $wpdb->prepare( $page_exit_sql, $from, $to ), ARRAY_A ) as $row ) {
				$page_exit_map[ (int) $row['campaign_id'] ] = (int) $row['total'];
			}
		}

		$conversion_map = [];
		$identity_expr = $this->conversion_identity_sql();
		$subject_expr = $this->conversion_subject_sql();
		$marketing_table = $wpdb->prefix . 'sn_dot_marketing_leads';
		$exclude_linked_marketing_crm = $this->table_exists( $marketing_table )
			? " AND NOT (c.conversion_type IN ('lead','approved_lead') AND c.subject_type='lead' AND EXISTS (SELECT 1 FROM {$marketing_table} ml WHERE ml.crm_lead_id=c.subject_id))"
			: '';
		// Registration/lead metrics are submission counts, not unique-phone
		// counts. Every Marketing Dot form attempt therefore remains visible even
		// when the same person submits again. The linked CRM conversion is excluded
		// to avoid counting one submission twice.
		$conv_sql = "SELECT c.{$model_column} campaign_id,c.conversion_type,COUNT(DISTINCT {$subject_expr}) total,SUM(c.revenue_amount) revenue FROM {$t['conversions']} c WHERE c.occurred_at BETWEEN %s AND %s AND c.{$model_column} IN ({$id_sql}) {$exclude_linked_marketing_crm} GROUP BY c.{$model_column},c.conversion_type";
		foreach ( (array) $wpdb->get_results( $wpdb->prepare( $conv_sql, $from, $to ), ARRAY_A ) as $row ) {
			$conversion_map[ (int) $row['campaign_id'] ][ (string) $row['conversion_type'] ] = [
				'total' => (int) $row['total'],
				'revenue' => (float) $row['revenue'],
			];
		}
		// "Active customer" remains a unique paid person rather than an invoice
		// count. Successful invoice count is separately available as sales_count.
		$customer_sql = "SELECT {$model_column} campaign_id,COUNT(DISTINCT {$identity_expr}) total FROM {$t['conversions']} WHERE conversion_type='active_customer' AND occurred_at BETWEEN %s AND %s AND {$model_column} IN ({$id_sql}) GROUP BY {$model_column}";
		foreach ( (array) $wpdb->get_results( $wpdb->prepare( $customer_sql, $from, $to ), ARRAY_A ) as $row ) {
			$conversion_map[ (int) $row['campaign_id'] ]['active_customer']['total'] = (int) $row['total'];
		}

		$rows = [];
		foreach ( $campaigns as $campaign_id => $campaign ) {
			$sessions = (int) ( $session_map[ $campaign_id ]['sessions'] ?? 0 );
			$registrations = (int) ( $conversion_map[ $campaign_id ]['registration']['total'] ?? 0 );
			$leads = (int) ( $conversion_map[ $campaign_id ]['lead']['total'] ?? 0 );
			$approved = (int) ( $conversion_map[ $campaign_id ]['approved_lead']['total'] ?? 0 );
			$customers = (int) ( $conversion_map[ $campaign_id ]['active_customer']['total'] ?? 0 );
			$sales_count = (int) ( $conversion_map[ $campaign_id ]['sale']['total'] ?? 0 );
			$application_started = (int) ( $conversion_map[ $campaign_id ]['application_started']['total'] ?? 0 );
			$otp_verified = (int) ( $conversion_map[ $campaign_id ]['otp_verified']['total'] ?? 0 );
			$gateway_started = (int) ( $conversion_map[ $campaign_id ]['gateway_started']['total'] ?? 0 );
			$funnel_abandoned = (int) ( $conversion_map[ $campaign_id ]['funnel_abandoned']['total'] ?? 0 );
			$lead_saved = (int) ( $conversion_map[ $campaign_id ]['lead_saved']['total'] ?? 0 );
			$invoice_created = (int) ( $conversion_map[ $campaign_id ]['invoice_created']['total'] ?? 0 );
			$awaiting_payment = (int) ( $conversion_map[ $campaign_id ]['awaiting_payment']['total'] ?? 0 );
			$popup_cancelled = (int) ( $conversion_map[ $campaign_id ]['popup_cancelled']['total'] ?? 0 );
			$gateway_cancelled = (int) ( $conversion_map[ $campaign_id ]['gateway_cancelled']['total'] ?? 0 );
			$payment_paid = (int) ( $conversion_map[ $campaign_id ]['payment_paid']['total'] ?? 0 );
			$pending_finance = (int) ( $conversion_map[ $campaign_id ]['pending_finance']['total'] ?? 0 );
			$payment_rejected = (int) ( $conversion_map[ $campaign_id ]['payment_rejected']['total'] ?? 0 );
			$sales_amount = (float) ( $conversion_map[ $campaign_id ]['sale']['revenue'] ?? 0 );
			$recorded_cost = max( 0, (float) ( $campaign['cost_amount'] ?? 0 ) );
			$cost = $recorded_cost > 0 ? $recorded_cost : max( 0, (float) ( $campaign['budget_amount'] ?? 0 ) );
			$rows[] = array_merge( $campaign, [
				'sessions' => $sessions,
				'visitors' => (int) ( $session_map[ $campaign_id ]['visitors'] ?? 0 ),
				'registrations' => $registrations,
				'conversion_rate' => $sessions > 0 ? round( ( $registrations / $sessions ) * 100, 2 ) : 0.0,
				'leads' => $leads,
				'approved_leads' => $approved,
				'active_customers' => $customers,
				'sales_count' => $sales_count,
				'application_started' => $application_started,
				'otp_verified' => $otp_verified,
				'gateway_started' => $gateway_started,
					'funnel_abandoned' => $funnel_abandoned,
					'page_closed_without_lead' => (int) ( $page_exit_map[ $campaign_id ] ?? 0 ),
					'lead_saved' => $lead_saved,
					'invoice_created' => $invoice_created,
					'awaiting_payment' => $awaiting_payment,
					'popup_cancelled' => $popup_cancelled,
					'gateway_cancelled' => $gateway_cancelled,
					'payment_paid' => $payment_paid,
					'pending_finance' => $pending_finance,
					'payment_rejected' => $payment_rejected,
				'sales_amount' => $sales_amount,
				'recorded_cost_amount' => $recorded_cost,
				'cost_amount' => $cost,
				'cpl' => $leads > 0 ? $cost / $leads : 0.0,
				'cpa' => $customers > 0 ? $cost / $customers : 0.0,
			] );
		}
		usort( $rows, static function ( array $a, array $b ): int {
			$by_sessions = (int) $b['sessions'] <=> (int) $a['sessions'];
			return $by_sessions !== 0 ? $by_sessions : ( (int) $b['id'] <=> (int) $a['id'] );
		} );

		$totals = $empty_totals;
		$totals['sessions'] = array_sum( array_column( $rows, 'sessions' ) );
		$totals['budget_amount'] = array_sum( array_map( 'floatval', array_column( $rows, 'budget_amount' ) ) );
		$totals['cost_amount'] = array_sum( array_map( 'floatval', array_column( $rows, 'cost_amount' ) ) );
		$totals['sales_amount'] = array_sum( array_map( 'floatval', array_column( $rows, 'sales_amount' ) ) );
		$totals['sales_count'] = array_sum( array_map( 'intval', array_column( $rows, 'sales_count' ) ) );
		$totals['registrations'] = array_sum( array_map( 'intval', array_column( $rows, 'registrations' ) ) );
		$totals['leads'] = array_sum( array_map( 'intval', array_column( $rows, 'leads' ) ) );
		$totals['approved_leads'] = array_sum( array_map( 'intval', array_column( $rows, 'approved_leads' ) ) );
		$totals['application_started'] = array_sum( array_map( 'intval', array_column( $rows, 'application_started' ) ) );
		$totals['otp_verified'] = array_sum( array_map( 'intval', array_column( $rows, 'otp_verified' ) ) );
		$totals['gateway_started'] = array_sum( array_map( 'intval', array_column( $rows, 'gateway_started' ) ) );
		$totals['funnel_abandoned'] = array_sum( array_map( 'intval', array_column( $rows, 'funnel_abandoned' ) ) );
		foreach ( [ 'page_closed_without_lead','lead_saved','invoice_created','awaiting_payment','popup_cancelled','gateway_cancelled','payment_paid','pending_finance','payment_rejected' ] as $metric ) {
			$totals[ $metric ] = array_sum( array_map( 'intval', array_column( $rows, $metric ) ) );
		}
		$totals['visitors'] = (int) $wpdb->get_var( $wpdb->prepare(
			"SELECT COUNT(DISTINCT visitor_id) FROM {$t['sessions']} WHERE started_at BETWEEN %s AND %s AND {$model_column} IN ({$id_sql})",
			$from,
			$to
		) );
		$totals['active_customers'] = (int) $wpdb->get_var( $wpdb->prepare(
			"SELECT COUNT(DISTINCT {$identity_expr}) FROM {$t['conversions']} WHERE conversion_type='active_customer' AND occurred_at BETWEEN %s AND %s AND {$model_column} IN ({$id_sql})",
			$from,
			$to
		) );
		$totals['conversion_rate'] = $totals['sessions'] > 0 ? round( ( $totals['registrations'] / $totals['sessions'] ) * 100, 2 ) : 0.0;
		$totals['cpl'] = $totals['leads'] > 0 ? $totals['cost_amount'] / $totals['leads'] : 0.0;
		$totals['cpa'] = $totals['active_customers'] > 0 ? $totals['cost_amount'] / $totals['active_customers'] : 0.0;
		return [ 'filters' => $filters, 'campaign_ids' => $ids, 'rows' => $rows, 'totals' => $totals ];
	}

	private function token_from_headers( $request = null ): string {
		$authorization = '';
		$custom = '';
		if ( $request instanceof WP_REST_Request ) {
			$authorization = (string) $request->get_header( 'authorization' );
			$custom = (string) $request->get_header( 'x-campaign-token' );
		} else {
			$authorization = (string) ( $_SERVER['HTTP_AUTHORIZATION'] ?? $_SERVER['REDIRECT_HTTP_AUTHORIZATION'] ?? '' );
			$custom = (string) ( $_SERVER['HTTP_X_CAMPAIGN_TOKEN'] ?? '' );
		}
		$raw = trim( $custom );
		if ( $raw === '' && preg_match( '/^Bearer\s+(.+)$/i', trim( $authorization ), $m ) ) { $raw = trim( $m[1] ); }
		if ( strlen( $raw ) < 32 || strlen( $raw ) > 191 || strpos( $raw, 'sncp_' ) !== 0 ) { return ''; }
		return sanitize_text_field( $raw );
	}

	private function authenticate_api_token( string $raw ) {
		if ( $raw === '' ) { return null; }
		global $wpdb;
		$t = $this->tables();
		$hash = hash( 'sha256', $raw );
		$row = $wpdb->get_row( $wpdb->prepare(
			"SELECT * FROM {$t['tokens']} WHERE token_hash=%s AND status='active' AND (expires_at IS NULL OR expires_at='' OR expires_at>%s) LIMIT 1",
			$hash,
			current_time( 'mysql' )
		) );
		if ( ! $row || ! user_can( (int) $row->partner_user_id, 'sn_view_campaign_reports' ) ) { return null; }
		$rate_key = 'sn_campaign_api_' . (int) $row->id;
		$count = (int) get_transient( $rate_key );
		if ( $count >= 300 ) { return new WP_Error( 'sn_campaign_rate_limited', 'تعداد درخواست بیش از حد مجاز است.', [ 'status' => 429 ] ); }
		set_transient( $rate_key, $count + 1, 5 * MINUTE_IN_SECONDS );
		$ip = sanitize_text_field( (string) ( $_SERVER['REMOTE_ADDR'] ?? '' ) );
		$wpdb->update( $t['tokens'], [
			'last_used_at' => current_time( 'mysql' ),
			'last_ip_hash' => $ip !== '' ? hash_hmac( 'sha256', $ip, wp_salt( 'auth' ) ) : null,
			'updated_at' => current_time( 'mysql' ),
		], [ 'id' => (int) $row->id ] );
		return $row;
	}

	private function api_payload( array $source, int $partner_user_id ): array {
		$filters = $this->normalize_filters( $source );
		$analytics = $this->analytics( $filters, $partner_user_id );
		$page = max( 1, absint( $source['page'] ?? 1 ) );
		$per_page = max( 1, min( 200, absint( $source['per_page'] ?? 100 ) ) );
		$total = count( $analytics['rows'] );
		$rows = array_slice( $analytics['rows'], ( $page - 1 ) * $per_page, $per_page );
		$data = [];
		foreach ( $rows as $row ) {
			$data[] = [
				'source' => (string) ( $row['utm_source'] ?? '' ),
				'medium' => (string) ( $row['utm_medium'] ?? '' ),
				'campaign' => (string) ( $row['utm_campaign'] ?? '' ),
				'content' => (string) ( $row['utm_content'] ?? '' ),
				'term' => (string) ( $row['utm_term'] ?? '' ),
				'sessions' => (int) $row['sessions'],
				'unique_visitors' => (int) $row['visitors'],
				'registrations' => (int) $row['registrations'],
				'conversion_rate' => (float) $row['conversion_rate'],
				'leads_count' => (int) $row['leads'],
				'applications_started_count' => (int) $row['application_started'],
				'otp_verified_count' => (int) $row['otp_verified'],
					'gateway_started_count' => (int) $row['gateway_started'],
					'funnel_abandoned_count' => (int) $row['funnel_abandoned'],
					'page_closed_without_lead_count' => (int) $row['page_closed_without_lead'],
					'lead_saved_count' => (int) $row['lead_saved'],
					'invoice_created_count' => (int) $row['invoice_created'],
					'awaiting_payment_count' => (int) $row['awaiting_payment'],
					'popup_cancelled_count' => (int) $row['popup_cancelled'],
					'gateway_cancelled_count' => (int) $row['gateway_cancelled'],
					'payment_paid_count' => (int) $row['payment_paid'],
					'pending_finance_count' => (int) $row['pending_finance'],
					'payment_rejected_count' => (int) $row['payment_rejected'],
				'approved_leads_count' => (int) $row['approved_leads'],
				'paid_customers_count' => (int) $row['active_customers'],
				'successful_payments_count' => (int) $row['sales_count'],
				'sales_amount' => (float) $row['sales_amount'],
			];
		}
		return [
			'data' => $data,
			'totals' => [
				'sessions' => (int) $analytics['totals']['sessions'],
				'unique_visitors' => (int) $analytics['totals']['visitors'],
				'registrations' => (int) $analytics['totals']['registrations'],
				'leads_count' => (int) $analytics['totals']['leads'],
				'applications_started_count' => (int) $analytics['totals']['application_started'],
				'otp_verified_count' => (int) $analytics['totals']['otp_verified'],
					'gateway_started_count' => (int) $analytics['totals']['gateway_started'],
					'funnel_abandoned_count' => (int) $analytics['totals']['funnel_abandoned'],
					'page_closed_without_lead_count' => (int) $analytics['totals']['page_closed_without_lead'],
					'lead_saved_count' => (int) $analytics['totals']['lead_saved'],
					'invoice_created_count' => (int) $analytics['totals']['invoice_created'],
					'awaiting_payment_count' => (int) $analytics['totals']['awaiting_payment'],
					'popup_cancelled_count' => (int) $analytics['totals']['popup_cancelled'],
					'gateway_cancelled_count' => (int) $analytics['totals']['gateway_cancelled'],
					'payment_paid_count' => (int) $analytics['totals']['payment_paid'],
					'pending_finance_count' => (int) $analytics['totals']['pending_finance'],
					'payment_rejected_count' => (int) $analytics['totals']['payment_rejected'],
				'approved_leads_count' => (int) $analytics['totals']['approved_leads'],
				'paid_customers_count' => (int) $analytics['totals']['active_customers'],
				'successful_payments_count' => (int) $analytics['totals']['sales_count'],
				'sales_amount' => (float) $analytics['totals']['sales_amount'],
			],
			'meta' => [
				'date_from' => $filters['date_from'],
				'date_to' => $filters['date_to'],
				'attribution_model' => $filters['attribution_model'],
				'page' => $page,
				'per_page' => $per_page,
				'total' => $total,
			],
		];
	}

	public function register_rest_routes(): void {
		register_rest_route( 'sn-crm/v1', '/campaign/report', [
			'methods' => WP_REST_Server::READABLE,
			'callback' => [ $this, 'rest_campaign_report' ],
			'permission_callback' => '__return_true',
		] );
	}

	public function disable_jwt_for_campaign_api( $check ) {
		$uri = (string) ( $_SERVER['REQUEST_URI'] ?? '' );
		if ( strpos( $uri, '/sn-crm/v1/campaign/report' ) !== false || strpos( $uri, '/api/campaign/report' ) !== false ) { return false; }
		return $check;
	}

	public function rest_campaign_report( WP_REST_Request $request ) {
		$token = $this->authenticate_api_token( $this->token_from_headers( $request ) );
		if ( is_wp_error( $token ) ) { return $token; }
		if ( ! $token ) { return new WP_Error( 'sn_campaign_unauthorized', 'توکن گزارش معتبر نیست.', [ 'status' => 401 ] ); }
		$response = new WP_REST_Response( $this->api_payload( $request->get_params(), (int) $token->partner_user_id ), 200 );
		$response->header( 'Cache-Control', 'private, no-store, max-age=0' );
		return $response;
	}

	private function is_friendly_api_request(): bool {
		$path = trim( rawurldecode( (string) wp_parse_url( (string) ( $_SERVER['REQUEST_URI'] ?? '' ), PHP_URL_PATH ) ), '/' );
		$home_path = trim( (string) wp_parse_url( home_url( '/' ), PHP_URL_PATH ), '/' );
		if ( $home_path !== '' && strpos( $path, $home_path . '/' ) === 0 ) { $path = substr( $path, strlen( $home_path ) + 1 ); }
		return $path === 'api/campaign/report';
	}

	public function handle_friendly_api(): void {
		if ( ! $this->is_friendly_api_request() ) { return; }
		if ( strtoupper( (string) ( $_SERVER['REQUEST_METHOD'] ?? 'GET' ) ) !== 'GET' ) {
			status_header( 405 );
			header( 'Allow: GET' );
			wp_send_json( [ 'code' => 'method_not_allowed', 'message' => 'این API فقط Read Only است.' ], 405 );
		}
		$token = $this->authenticate_api_token( $this->token_from_headers() );
		if ( is_wp_error( $token ) ) { wp_send_json( [ 'code' => $token->get_error_code(), 'message' => $token->get_error_message() ], (int) ( $token->get_error_data()['status'] ?? 429 ) ); }
		if ( ! $token ) { wp_send_json( [ 'code' => 'unauthorized', 'message' => 'توکن گزارش معتبر نیست.' ], 401 ); }
		nocache_headers();
		header( 'Content-Type: application/json; charset=' . get_option( 'blog_charset' ) );
		header( 'X-Content-Type-Options: nosniff' );
		wp_send_json( $this->api_payload( $_GET, (int) $token->partner_user_id ), 200 );
	}

	public function register_admin_menu(): void {
		add_submenu_page(
			'sn-dashboard',
			'تحلیل کمپین‌ها',
			'تحلیل کمپین‌ها',
			'manage_options',
			'sn-campaign-analytics',
			[ $this, 'render_admin_dashboard' ]
		);
	}

	public function enqueue_admin_asset( string $hook ): void {
		if ( strpos( $hook, 'sn-campaign-analytics' ) === false ) { return; }
		$file = SN_PLUGIN_DIR . 'assets/css/campaign-analytics.css';
		wp_enqueue_style( 'sn-campaign-analytics', SN_PLUGIN_URL . 'assets/css/campaign-analytics.css', [], SN_VERSION . '-' . ( file_exists( $file ) ? (string) filemtime( $file ) : '0' ) );
	}

	private function enqueue_portal_asset(): void {
		$file = SN_PLUGIN_DIR . 'assets/css/campaign-analytics.css';
		wp_enqueue_style( 'sn-campaign-analytics', SN_PLUGIN_URL . 'assets/css/campaign-analytics.css', [], SN_VERSION . '-' . ( file_exists( $file ) ? (string) filemtime( $file ) : '0' ) );
	}

	private function partner_page_url(): string {
		$page_id = (int) get_option( 'sn_campaign_partner_page_id', 0 );
		$url = $page_id > 0 ? (string) get_permalink( $page_id ) : '';
		if ( $url === '' ) {
			$page = get_page_by_path( 'campaign-partner', OBJECT, 'page' );
			$url = $page instanceof WP_Post ? (string) get_permalink( $page->ID ) : '';
		}
		return $url !== '' ? $url : home_url( '/' );
	}

	public function block_partner_admin(): void {
		if ( ! is_user_logged_in() || current_user_can( 'manage_options' ) || ! current_user_can( 'sn_view_campaign_reports' ) ) { return; }
		if ( wp_doing_ajax() || ( defined( 'DOING_CRON' ) && DOING_CRON ) ) { return; }
		$script = basename( (string) ( $_SERVER['PHP_SELF'] ?? '' ) );
		if ( in_array( $script, [ 'admin-ajax.php','admin-post.php' ], true ) ) { return; }
		wp_safe_redirect( $this->partner_page_url() );
		exit;
	}

	public function hide_partner_admin_bar(): void {
		if ( is_user_logged_in() && ! current_user_can( 'manage_options' ) && current_user_can( 'sn_view_campaign_reports' ) ) { show_admin_bar( false ); }
	}

	public function partner_login_redirect( string $redirect_to, string $requested_redirect_to, $user ): string {
		if ( $user instanceof WP_User && ! user_can( $user, 'manage_options' ) && user_can( $user, 'sn_view_campaign_reports' ) ) { return $this->partner_page_url(); }
		return $redirect_to;
	}

	private function metric_cards( array $totals ): string {
		$cards = [
			[ 'جلسه', number_format_i18n( (int) $totals['sessions'] ), 'sessions' ],
			[ 'بازدیدکننده یکتا', number_format_i18n( (int) $totals['visitors'] ), 'visitors' ],
			[ 'ثبت‌نام', number_format_i18n( (int) $totals['registrations'] ), 'registrations' ],
			[ 'نرخ تبدیل', number_format_i18n( (float) $totals['conversion_rate'], 2 ) . '٪', 'rate' ],
				[ 'کل لید', number_format_i18n( (int) $totals['leads'] ), 'leads' ],
				[ 'بازدید و بستن صفحه بدون لید', number_format_i18n( (int) $totals['page_closed_without_lead'] ), 'page-exit' ],
				[ 'لید ثبت‌شده', number_format_i18n( (int) $totals['lead_saved'] ), 'lead-saved' ],
				[ 'پیش‌فاکتور ساخته‌شده', number_format_i18n( (int) $totals['invoice_created'] ), 'invoice-created' ],
				[ 'در انتظار پرداخت', number_format_i18n( (int) $totals['awaiting_payment'] ), 'awaiting-payment' ],
				[ 'شروع درخواست', number_format_i18n( (int) $totals['application_started'] ), 'applications' ],
				[ 'ورود به درگاه', number_format_i18n( (int) $totals['gateway_started'] ), 'gateway' ],
				[ 'انصراف از پاپ‌آپ', number_format_i18n( (int) $totals['popup_cancelled'] ), 'popup-cancelled' ],
				[ 'انصراف از درگاه', number_format_i18n( (int) $totals['gateway_cancelled'] ), 'gateway-cancelled' ],
				[ 'پرداخت تأییدشده', number_format_i18n( (int) $totals['payment_paid'] ), 'payment-paid' ],
				[ 'در انتظار تأیید مالی', number_format_i18n( (int) $totals['pending_finance'] ), 'pending-finance' ],
				[ 'پرداخت ردشده', number_format_i18n( (int) $totals['payment_rejected'] ), 'payment-rejected' ],
			[ 'انصراف/ناموفق', number_format_i18n( (int) $totals['funnel_abandoned'] ), 'abandoned' ],
			[ 'لید تأییدشده', number_format_i18n( (int) $totals['approved_leads'] ), 'approved' ],
			[ 'پرداخت‌کننده یکتا', number_format_i18n( (int) $totals['active_customers'] ), 'customers' ],
			[ 'فاکتورهای موفق', number_format_i18n( (int) $totals['sales_count'] ), 'payments' ],
			[ 'فروش ایجادشده', $this->format_money( (float) $totals['sales_amount'] ), 'sales' ],
			[ 'بودجه کمپین', $this->format_money( (float) $totals['budget_amount'] ), 'budget' ],
			[ 'هزینه کمپین', $this->format_money( (float) $totals['cost_amount'] ), 'cost' ],
			[ 'CPL', $this->format_money( (float) $totals['cpl'] ), 'cpl' ],
			[ 'CPA', $this->format_money( (float) $totals['cpa'] ), 'cpa' ],
		];
		ob_start();
		?><div class="sn-campaign-cards"><?php foreach ( $cards as $card ) : ?>
			<div class="sn-campaign-card sn-campaign-card-<?php echo esc_attr( $card[2] ); ?>"><span><?php echo esc_html( $card[0] ); ?></span><strong><?php echo esc_html( $card[1] ); ?></strong></div>
		<?php endforeach; ?></div><?php
		return (string) ob_get_clean();
	}

	private function format_money( float $amount, string $currency = '' ): string {
		$currency = $currency !== '' ? $currency : ( function_exists( 'get_woocommerce_currency' ) ? get_woocommerce_currency() : 'IRR' );
		return number_format_i18n( max( 0, $amount ), 0 ) . ' ' . $currency;
	}

	private function filter_options( string $key, int $partner_user_id = 0 ): array {
		if ( ! in_array( $key, [ 'source','medium','campaign','content','term' ], true ) ) { return []; }
		global $wpdb;
		$t = $this->tables();
		$column = 'utm_' . $key;
		if ( $partner_user_id > 0 ) {
			$sql = $wpdb->prepare( "SELECT DISTINCT c.{$column} FROM {$t['campaigns']} c INNER JOIN {$t['assignments']} a ON a.campaign_id=c.id WHERE a.partner_user_id=%d AND c.{$column} IS NOT NULL AND c.{$column}<>'' ORDER BY c.{$column} ASC LIMIT 1000", $partner_user_id );
		} else {
			$sql = "SELECT DISTINCT {$column} FROM {$t['campaigns']} WHERE {$column} IS NOT NULL AND {$column}<>'' ORDER BY {$column} ASC LIMIT 1000";
		}
		return array_values( array_filter( array_map( 'strval', (array) $wpdb->get_col( $sql ) ) ) );
	}

	private function render_filters( array $filters, int $partner_user_id = 0, bool $admin = false ): void {
		$labels = [ 'source' => 'منبع (UTM Source)','medium' => 'رسانه (UTM Medium)','campaign' => 'کمپین (UTM Campaign)','content' => 'محتوای تبلیغ (UTM Content)','term' => 'کلیدواژه (UTM Term)' ];
		$display_date_from = $this->report_date_input_value( (string) $filters['date_from'] );
		$display_date_to = $this->report_date_input_value( (string) $filters['date_to'] );
		?>
		<form method="get" class="sn-campaign-filters">
			<?php if ( $admin ) : ?><input type="hidden" name="page" value="sn-campaign-analytics"><?php endif; ?>
			<label>از تاریخ<input type="text" class="sn-jalali-date" name="date_from" value="<?php echo esc_attr( $display_date_from ); ?>" placeholder="1403/02/18" autocomplete="off"></label>
			<label>تا تاریخ<input type="text" class="sn-jalali-date" name="date_to" value="<?php echo esc_attr( $display_date_to ); ?>" placeholder="1403/02/18" autocomplete="off"></label>
			<?php foreach ( $labels as $key => $label ) : $options = $this->filter_options( $key, $partner_user_id ); ?>
				<label><?php echo esc_html( $label ); ?><select name="<?php echo esc_attr( $key ); ?>"><option value="">همه</option><?php foreach ( $options as $option ) : ?><option value="<?php echo esc_attr( $option ); ?>" <?php selected( $filters[ $key ], $option ); ?>><?php echo esc_html( $option ); ?></option><?php endforeach; ?></select></label>
			<?php endforeach; ?>
			<label>مدل انتساب<select name="attribution_model"><option value="last" <?php selected( $filters['attribution_model'], 'last' ); ?>>آخرین تعامل</option><option value="first" <?php selected( $filters['attribution_model'], 'first' ); ?>>اولین تعامل</option></select></label>
			<button type="submit" class="button button-primary">اعمال فیلتر</button>
		</form>
		<?php
	}

	private function render_report_table( array $rows, bool $show_costs = true ): void {
		$colspan = $show_costs ? 30 : 26;
		$columns = [
			[ 'منبع', 'منبع ورود کاربر در UTM Source؛ مثلاً jaryan.' ],
			[ 'رسانه', 'نوع رسانه یا مدل همکاری در UTM Medium؛ مثلاً cpa.' ],
			[ 'کمپین', 'نام کمپین در UTM Campaign؛ گزارش‌ها بر اساس این شناسه تفکیک می‌شوند.' ],
			[ 'محتوای تبلیغ', 'شناسه یا محتوای خلاقه تبلیغ در UTM Content.' ],
			[ 'کلیدواژه', 'عبارت یا شناسه تکمیلی تبلیغ در UTM Term.' ],
			[ 'جلسه', 'تعداد Sessionهای ثبت‌شده در بازه انتخابی برای این کمپین.' ],
			[ 'بازدیدکننده یکتا', 'تعداد بازدیدکننده‌های متمایز در بازه انتخابی؛ هر نفر یک‌بار شمارش می‌شود.' ],
			[ 'ثبت‌نام', 'تعداد ثبت‌نام‌های موفق منتسب به این کمپین.' ],
			[ 'نرخ تبدیل', 'درصد ثبت‌نام نسبت به تعداد جلسه‌ها: ثبت‌نام تقسیم بر جلسه.' ],
			[ 'تعداد لید (ارسال فرم)', 'تعداد دفعاتی که فرم مارکتینگ با موفقیت ارسال شده است.' ],
			[ 'بازدید و بستن صفحه بدون لید', 'کاربر با UTM وارد شده، صفحه را بسته یا ترک کرده و در همان Session لید ثبت نکرده است.' ],
			[ 'لید ثبت‌شده', 'رویداد lead_saved در مارکتینگ دات‌فلو؛ یعنی اطلاعات لید ذخیره شده است.' ],
			[ 'پیش‌فاکتور ساخته‌شده', 'رویداد invoice_created؛ برای لید پیش‌فاکتور ساخته شده است.' ],
			[ 'در انتظار پرداخت', 'لید به مرحله انتظار پرداخت رسیده، اما پرداخت نهایی هنوز تأیید نشده است.' ],
			[ 'شروع درخواست', 'رویداد شروع فرایند فرم یا application_started.' ],
			[ 'ورود به درگاه', 'کاربر از فرایند مارکتینگ وارد درگاه پرداخت شده است.' ],
			[ 'انصراف از پاپ‌آپ', 'کاربر پاپ‌آپ یا مرحله تأیید را بسته و ادامه نداده است.' ],
			[ 'انصراف از درگاه', 'کاربر به درگاه رفته اما فرایند پرداخت را لغو کرده یا برگشته است.' ],
			[ 'انصراف/ناموفق', 'مجموع رویدادهای ناموفق یا انصرافی ثبت‌شده در قیف؛ شامل خروج ساده از صفحه نیست.' ],
			[ 'لید تأییدشده', 'لیدی که در CRM تأیید یا واجد شرایط اعلام شده است.' ],
			[ 'در انتظار تأیید مالی', 'پرداخت یا فاکتور ثبت شده و منتظر بررسی واحد مالی است.' ],
			[ 'پرداخت ردشده', 'پرداخت یا درخواست مالی توسط سیستم یا واحد مالی رد شده است.' ],
			[ 'پرداخت‌کننده یکتا', 'تعداد مشتری‌های متمایزی که پرداخت موفق داشته‌اند.' ],
			[ 'فاکتورهای موفق', 'تعداد فاکتورهایی که به Conversion فروش موفق رسیده‌اند.' ],
			[ 'پرداخت تأییدشده', 'تعداد رویدادهای payment_paid در مارکتینگ دات‌فلو.' ],
			[ 'فروش', 'مجموع مبلغ فروش منتسب به این کمپین در بازه انتخابی.' ],
			[ 'بودجه', 'بودجه تعریف‌شده برای کمپین.' ],
			[ 'هزینه', 'هزینه واقعی ثبت‌شده کمپین؛ اگر ثبت نشده باشد از بودجه استفاده می‌شود.' ],
			[ 'هزینه هر لید (CPL)', 'هزینه کمپین تقسیم بر تعداد لیدهای ارسال‌شده.' ],
			[ 'هزینه هر مشتری (CPA)', 'هزینه کمپین تقسیم بر تعداد پرداخت‌کننده‌های یکتا.' ],
		];
		?>
		<div class="sn-campaign-table-wrap"><table class="widefat striped sn-campaign-table"><thead><tr>
			<?php foreach ( $columns as $index => $column ) : ?>
				<?php if ( ! $show_costs && $index >= 26 ) { continue; } ?>
				<th><span class="sn-campaign-th-label"><?php echo esc_html( $column[0] ); ?></span><button type="button" class="sn-campaign-help" title="<?php echo esc_attr( $column[1] ); ?>" aria-label="توضیح <?php echo esc_attr( $column[0] ); ?>" data-tooltip="<?php echo esc_attr( $column[1] ); ?>">?</button></th>
			<?php endforeach; ?>
		</tr></thead><tbody>
		<?php if ( ! $rows ) : ?><tr><td colspan="<?php echo esc_attr( (string) $colspan ); ?>">در بازه و فیلتر انتخاب‌شده داده‌ای وجود ندارد.</td></tr><?php endif; ?>
		<?php foreach ( $rows as $row ) : ?>
			<tr>
				<td><?php echo esc_html( (string) ( $row['utm_source'] ?: '—' ) ); ?></td>
				<td><?php echo esc_html( (string) ( $row['utm_medium'] ?: '—' ) ); ?></td>
				<td><strong><?php echo esc_html( (string) ( $row['utm_campaign'] ?: $row['title'] ) ); ?></strong></td>
				<td><?php echo esc_html( (string) ( $row['utm_content'] ?: '—' ) ); ?></td>
				<td><?php echo esc_html( (string) ( $row['utm_term'] ?: '—' ) ); ?></td>
				<td><?php echo esc_html( number_format_i18n( (int) $row['sessions'] ) ); ?></td>
				<td><?php echo esc_html( number_format_i18n( (int) $row['visitors'] ) ); ?></td>
				<td><?php echo esc_html( number_format_i18n( (int) $row['registrations'] ) ); ?></td>
				<td><?php echo esc_html( number_format_i18n( (float) $row['conversion_rate'], 2 ) ); ?>٪</td>
				<td><?php echo esc_html( number_format_i18n( (int) $row['leads'] ) ); ?></td>
				<td><?php echo esc_html( number_format_i18n( (int) $row['page_closed_without_lead'] ) ); ?></td>
				<td><?php echo esc_html( number_format_i18n( (int) $row['lead_saved'] ) ); ?></td>
				<td><?php echo esc_html( number_format_i18n( (int) $row['invoice_created'] ) ); ?></td>
				<td><?php echo esc_html( number_format_i18n( (int) $row['awaiting_payment'] ) ); ?></td>
				<td><?php echo esc_html( number_format_i18n( (int) $row['application_started'] ) ); ?></td>
				<td><?php echo esc_html( number_format_i18n( (int) $row['gateway_started'] ) ); ?></td>
				<td><?php echo esc_html( number_format_i18n( (int) $row['popup_cancelled'] ) ); ?></td>
				<td><?php echo esc_html( number_format_i18n( (int) $row['gateway_cancelled'] ) ); ?></td>
				<td><?php echo esc_html( number_format_i18n( (int) $row['funnel_abandoned'] ) ); ?></td>
				<td><?php echo esc_html( number_format_i18n( (int) $row['approved_leads'] ) ); ?></td>
				<td><?php echo esc_html( number_format_i18n( (int) $row['pending_finance'] ) ); ?></td>
				<td><?php echo esc_html( number_format_i18n( (int) $row['payment_rejected'] ) ); ?></td>
				<td><?php echo esc_html( number_format_i18n( (int) $row['active_customers'] ) ); ?></td>
				<td><?php echo esc_html( number_format_i18n( (int) $row['sales_count'] ) ); ?></td>
				<td><?php echo esc_html( number_format_i18n( (int) $row['payment_paid'] ) ); ?></td>
				<td><?php echo esc_html( $this->format_money( (float) $row['sales_amount'], (string) $row['currency'] ) ); ?></td>
				<?php if ( $show_costs ) : ?>
					<td><?php echo esc_html( $this->format_money( (float) $row['budget_amount'], (string) $row['currency'] ) ); ?></td>
					<td><?php echo esc_html( $this->format_money( (float) $row['cost_amount'], (string) $row['currency'] ) ); ?></td>
					<td><?php echo esc_html( $this->format_money( (float) $row['cpl'], (string) $row['currency'] ) ); ?></td>
					<td><?php echo esc_html( $this->format_money( (float) $row['cpa'], (string) $row['currency'] ) ); ?></td>
				<?php endif; ?>
			</tr>
		<?php endforeach; ?>
		</tbody></table></div>
		<?php
	}

	private function admin_notice(): void {
		$notice = sanitize_key( wp_unslash( (string) ( $_GET['sn_campaign_notice'] ?? '' ) ) );
		$messages = [
			'saved' => [ 'success', 'کمپین و هزینه‌ها ذخیره شد.' ],
			'assigned' => [ 'success', 'دسترسی کمپین‌های پارتنر بروزرسانی شد.' ],
			'token_created' => [ 'success', 'توکن جدید ساخته شد. مقدار کامل فقط همین بار نمایش داده می‌شود.' ],
			'token_revoked' => [ 'success', 'توکن پارتنر باطل شد.' ],
			'settings_saved' => [ 'success', 'تنظیمات Tracking ذخیره شد.' ],
			'error' => [ 'error', 'عملیات انجام نشد؛ ورودی‌ها و لاگ دیتابیس را بررسی کنید.' ],
		];
		if ( isset( $messages[ $notice ] ) ) { printf( '<div class="notice notice-%s is-dismissible"><p>%s</p></div>', esc_attr( $messages[ $notice ][0] ), esc_html( $messages[ $notice ][1] ) ); }
	}

	public function render_admin_dashboard(): void {
		if ( ! current_user_can( 'manage_options' ) ) { wp_die( 'دسترسی غیرمجاز.' ); }
		$this->maybe_upgrade();
		$filters = $this->normalize_filters( $_GET );
		$analytics = $this->analytics( $filters );
		global $wpdb;
		$t = $this->tables();
		$edit_id = absint( $_GET['edit_campaign'] ?? 0 );
		$edit = $edit_id > 0 ? $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['campaigns']} WHERE id=%d LIMIT 1", $edit_id ), ARRAY_A ) : [];
		$partners = get_users( [ 'role' => 'sn_campaign_partner', 'number' => 1000, 'orderby' => 'display_name', 'order' => 'ASC' ] );
		$all_campaigns = $wpdb->get_results( "SELECT * FROM {$t['campaigns']} ORDER BY id DESC LIMIT 5000", ARRAY_A ) ?: [];
		$assignments = [];
		foreach ( (array) $wpdb->get_results( "SELECT partner_user_id,campaign_id FROM {$t['assignments']}", ARRAY_A ) as $row ) { $assignments[ (int) $row['partner_user_id'] ][] = (int) $row['campaign_id']; }
		$tokens = [];
		foreach ( (array) $wpdb->get_results( "SELECT * FROM {$t['tokens']} ORDER BY id DESC", ARRAY_A ) as $row ) { $tokens[ (int) $row['partner_user_id'] ][] = $row; }
		?>
		<div class="wrap sn-campaign-admin" dir="rtl">
			<h1>تحلیل کمپین و گزارش پارتنرها</h1>
			<p class="description">گزارش مسیر کاربر از اولین ورود با UTM تا ثبت‌نام، لید، پرداخت و فروش. مدل انتساب را از «اولین تعامل» یا «آخرین تعامل» انتخاب کنید.</p>
			<?php $this->admin_notice(); ?>
			<section class="sn-campaign-section">
				<h2>داشبورد تحلیل کمپین</h2>
				<p class="description">«بازدید و بستن صفحه بدون لید» یعنی کاربر با لینک UTM وارد شده، صفحه را ترک کرده و در همان نشست هیچ ارسال فرم مارکتینگ داتی ثبت نشده است.</p>
				<?php $this->render_filters( $filters, 0, true ); ?>
				<?php echo $this->metric_cards( $analytics['totals'] ); // phpcs:ignore WordPress.Security.EscapeOutput.OutputNotEscaped ?>
				<?php $this->render_report_table( $analytics['rows'], true ); ?>
			</section>

			<section class="sn-campaign-section" id="campaign-editor">
				<h2><?php echo $edit ? 'ویرایش کمپین #' . esc_html( (string) $edit_id ) : 'تعریف کمپین و هزینه'; ?></h2>
				<form method="post" action="<?php echo esc_url( admin_url( 'admin-post.php' ) ); ?>" class="sn-campaign-editor">
					<input type="hidden" name="action" value="sn_campaign_save"><input type="hidden" name="campaign_id" value="<?php echo esc_attr( (string) $edit_id ); ?>">
					<?php wp_nonce_field( 'sn_campaign_save' ); ?>
					<label>عنوان داخلی<input type="text" name="title" maxlength="191" value="<?php echo esc_attr( (string) ( $edit['title'] ?? '' ) ); ?>"></label>
					<?php foreach ( [ 'source' => 'منبع (utm_source)','medium' => 'رسانه (utm_medium)','campaign' => 'کمپین (utm_campaign)','content' => 'محتوای تبلیغ (utm_content)','term' => 'کلیدواژه (utm_term)' ] as $name => $column ) : ?><label><?php echo esc_html( $column ); ?><input type="text" name="<?php echo esc_attr( $name ); ?>" maxlength="191" value="<?php echo esc_attr( (string) ( $edit[ 'utm_' . $name ] ?? '' ) ); ?>"></label><?php endforeach; ?>
					<label>بودجه<input type="number" min="0" step="0.01" name="budget_amount" value="<?php echo esc_attr( (string) ( $edit['budget_amount'] ?? '0' ) ); ?>"></label>
					<label>هزینه کمپین<input type="number" min="0" step="0.01" name="cost_amount" value="<?php echo esc_attr( (string) ( $edit['cost_amount'] ?? '0' ) ); ?>"><small>مبنای هزینه هر لید و هزینه هر مشتری؛ اگر صفر باشد بودجه استفاده می‌شود.</small></label>
					<label>واحد پول<input type="text" name="currency" maxlength="12" value="<?php echo esc_attr( (string) ( $edit['currency'] ?? ( function_exists( 'get_woocommerce_currency' ) ? get_woocommerce_currency() : 'IRR' ) ) ); ?>"></label>
					<label>وضعیت<select name="status"><option value="active" <?php selected( (string) ( $edit['status'] ?? 'active' ), 'active' ); ?>>فعال</option><option value="inactive" <?php selected( (string) ( $edit['status'] ?? '' ), 'inactive' ); ?>>غیرفعال</option></select></label>
					<button class="button button-primary" type="submit">ذخیره کمپین</button>
				</form>
				<?php if ( $all_campaigns ) : ?><div class="sn-campaign-compact-list"><?php foreach ( $all_campaigns as $campaign ) : ?><a href="<?php echo esc_url( add_query_arg( [ 'page' => 'sn-campaign-analytics', 'edit_campaign' => (int) $campaign['id'] ], admin_url( 'admin.php' ) ) ); ?>#campaign-editor"><strong><?php echo esc_html( (string) $campaign['title'] ); ?></strong><span><?php echo esc_html( (string) ( $campaign['utm_source'] ?: '—' ) . ' / ' . (string) ( $campaign['utm_campaign'] ?: '—' ) ); ?></span></a><?php endforeach; ?></div><?php endif; ?>
			</section>

			<section class="sn-campaign-section">
				<h2>دسترسی پارتنرها</h2>
				<p>کاربر را از بخش کاربران وردپرس با Role «پارتنر کمپین» بسازید؛ سپس فقط کمپین‌های مجاز او را اینجا انتخاب کنید.</p>
				<?php if ( ! $partners ) : ?><div class="notice notice-info inline"><p>هنوز کاربری با Role پارتنر کمپین ساخته نشده است.</p></div><?php endif; ?>
				<?php foreach ( $partners as $partner ) : $allowed = $assignments[ (int) $partner->ID ] ?? []; ?>
					<div class="sn-campaign-partner-card">
						<div><h3><?php echo esc_html( $partner->display_name ?: $partner->user_login ); ?></h3><code><?php echo esc_html( $partner->user_login ); ?></code></div>
						<form method="post" action="<?php echo esc_url( admin_url( 'admin-post.php' ) ); ?>">
							<input type="hidden" name="action" value="sn_campaign_assign_partner"><input type="hidden" name="partner_user_id" value="<?php echo esc_attr( (string) $partner->ID ); ?>"><?php wp_nonce_field( 'sn_campaign_assign_partner_' . (int) $partner->ID ); ?>
							<div class="sn-campaign-assignment-list"><?php foreach ( $all_campaigns as $campaign ) : ?><label><input type="checkbox" name="campaign_ids[]" value="<?php echo esc_attr( (string) $campaign['id'] ); ?>" <?php checked( in_array( (int) $campaign['id'], $allowed, true ) ); ?>><?php echo esc_html( (string) $campaign['title'] ); ?> <small><?php echo esc_html( (string) ( $campaign['utm_source'] ?: '—' ) . ' / ' . (string) ( $campaign['utm_campaign'] ?: '—' ) ); ?></small></label><?php endforeach; ?></div>
							<button class="button button-primary" type="submit">ذخیره دسترسی‌ها</button>
						</form>
						<div class="sn-campaign-token-actions">
							<form method="post" action="<?php echo esc_url( admin_url( 'admin-post.php' ) ); ?>"><input type="hidden" name="action" value="sn_campaign_generate_token"><input type="hidden" name="partner_user_id" value="<?php echo esc_attr( (string) $partner->ID ); ?>"><?php wp_nonce_field( 'sn_campaign_generate_token_' . (int) $partner->ID ); ?><button class="button" type="submit">ساخت توکن جدید</button></form>
							<?php foreach ( $tokens[ (int) $partner->ID ] ?? [] as $token ) : ?><div class="sn-campaign-token-row"><code><?php echo esc_html( (string) $token['token_prefix'] ); ?>…</code><span><?php echo esc_html( (string) $token['status'] ); ?></span><span>آخرین استفاده: <?php echo esc_html( (string) ( $token['last_used_at'] ?: '—' ) ); ?></span><?php if ( $token['status'] === 'active' ) : ?><form method="post" action="<?php echo esc_url( admin_url( 'admin-post.php' ) ); ?>"><input type="hidden" name="action" value="sn_campaign_revoke_token"><input type="hidden" name="token_id" value="<?php echo esc_attr( (string) $token['id'] ); ?>"><?php wp_nonce_field( 'sn_campaign_revoke_token_' . (int) $token['id'] ); ?><button class="button-link-delete" type="submit">ابطال</button></form><?php endif; ?></div><?php endforeach; ?>
						</div>
					</div>
				<?php endforeach; ?>
				<div class="sn-campaign-api-help"><strong>API Read Only:</strong> <code><?php echo esc_html( home_url( '/api/campaign/report' ) ); ?></code><br><small>هدر پیشنهادی: <code>Authorization: Bearer TOKEN</code>. توکن در Query String پذیرفته نمی‌شود.</small></div>
			</section>

			<section class="sn-campaign-section">
				<h2>تنظیمات Tracking و حریم خصوصی</h2>
				<form method="post" action="<?php echo esc_url( admin_url( 'admin-post.php' ) ); ?>" class="sn-campaign-settings-form"><input type="hidden" name="action" value="sn_campaign_save_settings"><?php wp_nonce_field( 'sn_campaign_save_settings' ); ?>
					<label><input type="checkbox" name="tracking_enabled" value="1" <?php checked( $this->tracking_enabled() ); ?>> Tracking فعال باشد</label>
					<label><input type="checkbox" name="store_ip" value="1" <?php checked( (string) get_option( 'sn_campaign_store_ip', '0' ), '1' ); ?>> IP خام ذخیره شود <small>به‌صورت پیش‌فرض خاموش است.</small></label>
					<label>پنجره Attribution (روز)<input type="number" name="window_days" min="1" max="3650" value="<?php echo esc_attr( (string) $this->attribution_window_days() ); ?>"></label>
					<button class="button button-primary" type="submit">ذخیره تنظیمات</button>
				</form>
			</section>
		</div>
		<?php
	}

	public function render_partner_portal(): string {
		$this->enqueue_portal_asset();
		if ( ! is_user_logged_in() ) {
			$login_id = (int) get_option( 'sn_login_page_id', 0 );
			$login_url = $login_id > 0 ? (string) get_permalink( $login_id ) : wp_login_url( $this->partner_page_url() );
			return '<div class="sn-campaign-portal sn-campaign-login" dir="rtl"><h2>پرتال گزارش کمپین</h2><p>برای مشاهده گزارش کمپین‌های اختصاص‌یافته وارد شوید.</p><a class="sn-campaign-primary-link" href="' . esc_url( $login_url ) . '">ورود به CRM</a></div>';
		}
		if ( ! current_user_can( 'manage_options' ) && ! current_user_can( 'sn_view_campaign_reports' ) ) {
			return '<div class="sn-campaign-portal" dir="rtl"><div class="sn-campaign-alert">دسترسی به گزارش کمپین برای این حساب فعال نیست.</div></div>';
		}
		$user = wp_get_current_user();
		$partner_id = current_user_can( 'manage_options' ) ? 0 : (int) $user->ID;
		$filters = $this->normalize_filters( $_GET );
		$analytics = $this->analytics( $filters, $partner_id );
		global $wpdb;
		$t = $this->tables();
		$token_rows = $partner_id > 0 ? $wpdb->get_results( $wpdb->prepare( "SELECT token_prefix,status,last_used_at,created_at FROM {$t['tokens']} WHERE partner_user_id=%d ORDER BY id DESC LIMIT 10", $partner_id ), ARRAY_A ) : [];
		ob_start();
		?>
		<div class="sn-campaign-portal" dir="rtl">
		<header class="sn-campaign-portal-header"><div><span>گزارش عملکرد کمپین</span><h1>گزارش کمپین‌های <?php echo esc_html( $user->display_name ?: $user->user_login ); ?></h1><p>فقط کمپین‌های اختصاص‌یافته به این حساب در محاسبات و API قابل مشاهده‌اند.</p></div><a href="<?php echo esc_url( wp_logout_url( $this->partner_page_url() ) ); ?>">خروج</a></header>
			<section class="sn-campaign-section">
				<p class="description">«بازدید و بستن صفحه بدون لید» یعنی ورود با لینک UTM و خروج بدون ارسال فرم در همان نشست.</p>
				<?php $this->render_filters( $filters, $partner_id, false ); ?>
				<?php echo $this->metric_cards( $analytics['totals'] ); // phpcs:ignore WordPress.Security.EscapeOutput.OutputNotEscaped ?>
				<?php $this->render_report_table( $analytics['rows'], true ); ?>
			</section>
			<section class="sn-campaign-section sn-campaign-api-partner">
				<h2>خروجی API گزارش (فقط خواندنی)</h2>
				<code><?php echo esc_html( home_url( '/api/campaign/report' ) ); ?></code>
				<p>توکن را در هدر <code>Authorization: Bearer TOKEN</code> ارسال کنید. توکن کامل فقط هنگام ساخت توسط مدیر نمایش داده می‌شود.</p>
				<?php if ( ! $token_rows ) : ?><p class="sn-campaign-muted">توکن فعالی برای این حساب ثبت نشده است.</p><?php endif; ?>
				<?php foreach ( $token_rows as $token ) : ?><div class="sn-campaign-token-row"><code><?php echo esc_html( (string) $token['token_prefix'] ); ?>…</code><span><?php echo esc_html( (string) $token['status'] ); ?></span><span>آخرین استفاده: <?php echo esc_html( (string) ( $token['last_used_at'] ?: '—' ) ); ?></span></div><?php endforeach; ?>
			</section>
		</div>
		<?php
		return (string) ob_get_clean();
	}

	private function admin_post_guard( string $action ): void {
		if ( ! current_user_can( 'manage_options' ) ) { wp_die( 'دسترسی غیرمجاز.', 'خطای دسترسی', [ 'response' => 403 ] ); }
		check_admin_referer( $action );
	}

	private function admin_redirect( string $notice, array $extra = [] ): void {
		$args = array_merge( [ 'page' => 'sn-campaign-analytics', 'sn_campaign_notice' => sanitize_key( $notice ) ], $extra );
		wp_safe_redirect( add_query_arg( $args, admin_url( 'admin.php' ) ) );
		exit;
	}

	private function decimal_value( $value ): float {
		$value = wp_unslash( (string) $value );
		if ( class_exists( 'SN_Helpers' ) ) { $value = SN_Helpers::to_english_nums( $value ); }
		$value = str_replace( [ ',', '٬', ' ' ], '', $value );
		return max( 0, is_numeric( $value ) ? (float) $value : 0.0 );
	}

	public function handle_save_campaign(): void {
		$this->admin_post_guard( 'sn_campaign_save' );
		global $wpdb;
		$t = $this->tables();
		$id = absint( $_POST['campaign_id'] ?? 0 );
		$utm = [];
		foreach ( [ 'source','medium','campaign','content','term' ] as $key ) { $utm[ $key ] = $this->normalize_utm_value( $_POST[ $key ] ?? '' ); }
		if ( ! $this->has_utm( $utm ) ) { $this->admin_redirect( 'error' ); }
		$title = $this->normalize_utm_value( $_POST['title'] ?? '' );
		if ( $title === '' ) { $title = $utm['campaign'] ?: $utm['source'] ?: 'کمپین بدون نام'; }
		$currency = strtoupper( sanitize_key( wp_unslash( (string) ( $_POST['currency'] ?? 'IRR' ) ) ) );
		$data = [
			'campaign_key' => $this->campaign_key( $utm ),
			'title' => $title,
			'utm_source' => $utm['source'] ?: null,
			'utm_medium' => $utm['medium'] ?: null,
			'utm_campaign' => $utm['campaign'] ?: null,
			'utm_content' => $utm['content'] ?: null,
			'utm_term' => $utm['term'] ?: null,
			'budget_amount' => $this->decimal_value( $_POST['budget_amount'] ?? 0 ),
			'cost_amount' => $this->decimal_value( $_POST['cost_amount'] ?? 0 ),
			'currency' => $currency !== '' ? substr( $currency, 0, 12 ) : 'IRR',
			'status' => sanitize_key( wp_unslash( (string) ( $_POST['status'] ?? 'active' ) ) ) === 'inactive' ? 'inactive' : 'active',
			'updated_at' => current_time( 'mysql' ),
		];
		if ( $id > 0 ) {
			$ok = false !== $wpdb->update( $t['campaigns'], $data, [ 'id' => $id ] );
		} else {
			$existing = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$t['campaigns']} WHERE campaign_key=%s LIMIT 1", $data['campaign_key'] ) );
			if ( $existing > 0 ) { $ok = false !== $wpdb->update( $t['campaigns'], $data, [ 'id' => $existing ] ); }
			else { $data['created_by'] = get_current_user_id(); $data['created_at'] = current_time( 'mysql' ); $ok = (bool) $wpdb->insert( $t['campaigns'], $data ); }
		}
		$this->admin_redirect( $ok ? 'saved' : 'error' );
	}

	public function handle_assign_partner(): void {
		$partner_id = absint( $_POST['partner_user_id'] ?? 0 );
		$this->admin_post_guard( 'sn_campaign_assign_partner_' . $partner_id );
		if ( $partner_id < 1 || ! user_can( $partner_id, 'sn_view_campaign_reports' ) ) { $this->admin_redirect( 'error' ); }
		$campaign_ids = array_values( array_unique( array_filter( array_map( 'absint', (array) ( $_POST['campaign_ids'] ?? [] ) ) ) ) );
		global $wpdb;
		$t = $this->tables();
		if ( $campaign_ids ) {
			$id_sql = implode( ',', $campaign_ids );
			$valid = array_map( 'absint', (array) $wpdb->get_col( "SELECT id FROM {$t['campaigns']} WHERE id IN ({$id_sql})" ) );
			$campaign_ids = array_values( array_intersect( $campaign_ids, $valid ) );
		}
		$wpdb->query( 'START TRANSACTION' );
		$ok = false !== $wpdb->delete( $t['assignments'], [ 'partner_user_id' => $partner_id ] );
		foreach ( $campaign_ids as $campaign_id ) {
			if ( ! $ok ) { break; }
			$ok = (bool) $wpdb->insert( $t['assignments'], [ 'partner_user_id' => $partner_id, 'campaign_id' => $campaign_id, 'created_by' => get_current_user_id(), 'created_at' => current_time( 'mysql' ) ] );
		}
		$wpdb->query( $ok ? 'COMMIT' : 'ROLLBACK' );
		$this->admin_redirect( $ok ? 'assigned' : 'error' );
	}

	private function generate_raw_token(): string {
		try { $random = bin2hex( random_bytes( 32 ) ); }
		catch ( Throwable $e ) { $random = wp_generate_password( 64, false, false ); }
		return 'sncp_' . $random;
	}

	private function render_token_once( string $raw, int $partner_id ): void {
		while ( ob_get_level() ) { ob_end_clean(); }
		nocache_headers();
		header( 'Referrer-Policy: no-referrer' );
		header( "Content-Security-Policy: default-src 'none'; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'" );
		header( 'Content-Type: text/html; charset=' . get_option( 'blog_charset' ) );
		$partner = get_user_by( 'id', $partner_id );
		$back = add_query_arg( [ 'page' => 'sn-campaign-analytics', 'sn_campaign_notice' => 'token_created' ], admin_url( 'admin.php' ) );
		?><!doctype html><html lang="fa" dir="rtl"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>توکن جدید Campaign Partner</title><style>body{background:#f4f6fb;color:#172033;font-family:Tahoma,Arial,sans-serif;margin:0;padding:40px}.card{background:#fff;border:1px solid #e3e7ef;border-radius:18px;box-shadow:0 12px 36px #17203314;margin:auto;max-width:820px;padding:28px}code{background:#f3f5ff;border:1px solid #d7dcff;border-radius:10px;direction:ltr;display:block;font-size:15px;line-height:1.8;margin:18px 0;overflow-wrap:anywhere;padding:15px;text-align:left}.warning{background:#fff6dd;border-radius:10px;padding:12px}.button{background:#3b4cca;border-radius:9px;color:#fff;display:inline-block;margin-top:16px;padding:11px 18px;text-decoration:none}</style></head><body><main class="card"><h1>توکن جدید ساخته شد</h1><p>پارتنر: <strong><?php echo esc_html( $partner ? ( $partner->display_name ?: $partner->user_login ) : (string) $partner_id ); ?></strong></p><code><?php echo esc_html( $raw ); ?></code><p class="warning">این مقدار فقط در همین پاسخ نمایش داده می‌شود، در دیتابیس به‌صورت خام ذخیره نشده و بعد از خروج قابل بازیابی نیست. اکنون آن را در محل امن کپی کنید.</p><a class="button" href="<?php echo esc_url( $back ); ?>">بازگشت به تحلیل کمپین‌ها</a></main></body></html><?php
		exit;
	}

	public function handle_generate_token(): void {
		$partner_id = absint( $_POST['partner_user_id'] ?? 0 );
		$this->admin_post_guard( 'sn_campaign_generate_token_' . $partner_id );
		if ( $partner_id < 1 || ! user_can( $partner_id, 'sn_view_campaign_reports' ) ) { $this->admin_redirect( 'error' ); }
		global $wpdb;
		$t = $this->tables();
		$raw = $this->generate_raw_token();
		$now = current_time( 'mysql' );
		$wpdb->query( 'START TRANSACTION' );
		$ok = false !== $wpdb->update( $t['tokens'], [ 'status' => 'revoked', 'updated_at' => $now ], [ 'partner_user_id' => $partner_id, 'status' => 'active' ] );
		if ( $ok ) {
			$ok = (bool) $wpdb->insert( $t['tokens'], [
				'partner_user_id' => $partner_id,
				'token_hash' => hash( 'sha256', $raw ),
				'token_prefix' => substr( $raw, 0, 17 ),
				'status' => 'active',
				'created_by' => get_current_user_id(),
				'created_at' => $now,
				'updated_at' => $now,
			] );
		}
		$wpdb->query( $ok ? 'COMMIT' : 'ROLLBACK' );
		if ( $ok ) { $this->render_token_once( $raw, $partner_id ); }
		$this->admin_redirect( 'error' );
	}

	public function handle_revoke_token(): void {
		$token_id = absint( $_POST['token_id'] ?? 0 );
		$this->admin_post_guard( 'sn_campaign_revoke_token_' . $token_id );
		global $wpdb;
		$t = $this->tables();
		$ok = $token_id > 0 && false !== $wpdb->update( $t['tokens'], [ 'status' => 'revoked', 'updated_at' => current_time( 'mysql' ) ], [ 'id' => $token_id ] );
		$this->admin_redirect( $ok ? 'token_revoked' : 'error' );
	}

	public function handle_save_settings(): void {
		$this->admin_post_guard( 'sn_campaign_save_settings' );
		update_option( 'sn_campaign_tracking_enabled', ! empty( $_POST['tracking_enabled'] ) ? '1' : '0', false );
		update_option( 'sn_campaign_store_ip', ! empty( $_POST['store_ip'] ) ? '1' : '0', false );
		$days = max( 1, min( 3650, absint( $_POST['window_days'] ?? self::DEFAULT_WINDOW ) ) );
		update_option( 'sn_campaign_attribution_window_days', $days, false );
		$this->admin_redirect( 'settings_saved' );
	}
}
