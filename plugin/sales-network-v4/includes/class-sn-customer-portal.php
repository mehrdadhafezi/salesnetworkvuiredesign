<?php
if ( ! defined( 'ABSPATH' ) ) { exit; }

/**
 * Customer-facing Biawin profile and passwordless mobile login.
 *
 * This deliberately stays separate from the internal Customer 360 screen.
 * Customers can only read memberships linked to their own WordPress account
 * (with a normalized-phone fallback for older rows that have no user id).
 */
final class SN_Customer_Portal {
	private const DB_VERSION = '2026-09-13-biavin-customer-invite-retry-v2';
	private const OTP_TTL = 300;
	private const OTP_MAX_ATTEMPTS = 5;
	private const INVITE_RETRY_HOOK = 'sn_customer_portal_retry_invite';
	private const INVITE_MAX_ATTEMPTS = 3;
	private static ?self $instance = null;
	private bool $hooks_registered = false;

	public static function instance(): self {
		if ( null === self::$instance ) { self::$instance = new self(); }
		return self::$instance;
	}

	private function __construct() {}

	public static function activate(): void {
		self::instance()->install_schema();
	}

	public function register_hooks(): void {
		if ( $this->hooks_registered ) { return; }
		$this->hooks_registered = true;
		add_action( 'init', [ $this, 'register_shortcode' ], 3 );
		add_action( 'init', [ $this, 'maybe_upgrade' ], 9 );
		add_action( 'wp_enqueue_scripts', [ $this, 'enqueue_assets' ], 31 );
		// Send only after Operations (priority 20) has snapshotted every card route.
		add_action( 'sn_project_membership_activated', [ $this, 'send_membership_invite' ], 30, 3 );
		add_action( self::INVITE_RETRY_HOOK, [ $this, 'send_membership_invite' ], 10, 2 );
		add_action( 'wp_ajax_sn_customer_portal_send_otp', [ $this, 'ajax_send_otp' ] );
		add_action( 'wp_ajax_nopriv_sn_customer_portal_send_otp', [ $this, 'ajax_send_otp' ] );
		add_action( 'wp_ajax_sn_customer_portal_verify_otp', [ $this, 'ajax_verify_otp' ] );
		add_action( 'wp_ajax_nopriv_sn_customer_portal_verify_otp', [ $this, 'ajax_verify_otp' ] );
		add_action( 'wp_ajax_sn_customer_purchased_subscriptions', [ $this, 'ajax_customer_purchased_subscriptions' ] );
		add_filter( 'show_admin_bar', [ $this, 'filter_admin_bar' ] );
		add_filter( 'wp_robots', [ $this, 'filter_portal_robots' ] );
	}

	public function register_shortcode(): void {
		add_shortcode( 'sn_biavin_customer_portal', [ $this, 'render' ] );
	}

	public function maybe_upgrade(): void {
		if ( (string) get_option( 'sn_customer_portal_db_version', '' ) !== self::DB_VERSION ) { $this->install_schema(); }
	}

	private function table_exists( string $table ): bool {
		global $wpdb;
		return (string) $wpdb->get_var( $wpdb->prepare( 'SHOW TABLES LIKE %s', $table ) ) === $table;
	}

	private function install_schema(): void {
		global $wpdb;
		$table = $wpdb->prefix . 'sn_project_memberships';
		if ( ! $this->table_exists( $table ) ) { return; }
		$columns = array_flip( (array) $wpdb->get_col( "SHOW COLUMNS FROM {$table}", 0 ) );
		$alter = [];
		if ( ! isset( $columns['portal_invite_status'] ) ) { $alter[] = "ADD COLUMN portal_invite_status VARCHAR(20) NULL AFTER status"; }
		if ( ! isset( $columns['portal_invite_attempts'] ) ) { $alter[] = 'ADD COLUMN portal_invite_attempts INT UNSIGNED NOT NULL DEFAULT 0 AFTER portal_invite_status'; }
		if ( ! isset( $columns['portal_invite_attempted_at'] ) ) { $alter[] = 'ADD COLUMN portal_invite_attempted_at DATETIME NULL AFTER portal_invite_attempts'; }
		if ( ! isset( $columns['portal_invite_sent_at'] ) ) { $alter[] = 'ADD COLUMN portal_invite_sent_at DATETIME NULL AFTER portal_invite_attempted_at'; }
		if ( $alter ) { $wpdb->query( "ALTER TABLE {$table} " . implode( ', ', $alter ) ); }
		$columns = array_flip( (array) $wpdb->get_col( "SHOW COLUMNS FROM {$table}", 0 ) );
		if ( isset( $columns['portal_invite_status'], $columns['portal_invite_attempts'], $columns['portal_invite_attempted_at'], $columns['portal_invite_sent_at'] ) ) {
			update_option( 'sn_customer_portal_db_version', self::DB_VERSION, false );
		}
	}

	private function portal_page_id(): int {
		$page_id = absint( get_option( 'sn_biavin_customer_portal_page_id', 0 ) );
		$post = $page_id ? get_post( $page_id ) : null;
		if ( $post instanceof WP_Post && $post->post_type === 'page' && $post->post_status !== 'trash' ) { return $page_id; }
		$by_slug = get_page_by_path( 'biawin', OBJECT, 'page' );
		return $by_slug instanceof WP_Post && $by_slug->post_status !== 'trash' ? (int) $by_slug->ID : 0;
	}

	private function portal_url(): string {
		$page_id = $this->portal_page_id();
		$url = $page_id > 0 ? (string) get_permalink( $page_id ) : '';
		return $url !== '' ? $url : home_url( '/biawin/' );
	}

	private function current_page_is_portal(): bool {
		global $post;
		if ( ! $post || ! is_singular() ) { return false; }
		if ( $this->portal_page_id() === (int) $post->ID ) { return true; }
		$content = (string) ( $post->post_content ?? '' );
		$elementor = (string) get_post_meta( (int) $post->ID, '_elementor_data', true );
		return has_shortcode( $content, 'sn_biavin_customer_portal' ) || strpos( $elementor, 'sn_biavin_customer_portal' ) !== false;
	}

	public function enqueue_assets(): void {
		if ( ! $this->current_page_is_portal() ) { return; }
		$css = SN_PLUGIN_DIR . 'assets/css/customer-portal.css';
		$js = SN_PLUGIN_DIR . 'assets/js/customer-portal.js';
		wp_enqueue_style( 'sn-customer-portal', SN_PLUGIN_URL . 'assets/css/customer-portal.css', [], SN_VERSION . '-' . ( file_exists( $css ) ? (string) filemtime( $css ) : '0' ) );
		wp_enqueue_script( 'sn-customer-portal', SN_PLUGIN_URL . 'assets/js/customer-portal.js', [], SN_VERSION . '-' . ( file_exists( $js ) ? (string) filemtime( $js ) : '0' ), true );
		wp_localize_script( 'sn-customer-portal', 'snCustomerPortal', [
			'ajaxUrl' => admin_url( 'admin-ajax.php' ),
			'nonce' => wp_create_nonce( 'sn_customer_portal_otp' ),
			'portalUrl' => $this->portal_url(),
			'otpLength' => 6,
		] );
	}

	public function filter_portal_robots( array $robots ): array {
		if ( ! $this->current_page_is_portal() ) { return $robots; }
		$robots['noindex'] = true;
		$robots['nofollow'] = true;
		$robots['noarchive'] = true;
		return $robots;
	}

	private function internal_roles(): array {
		return [ 'administrator', 'sn_seller', 'sas_employee', 'sn_supervisor', 'sas_supervisor', 'sn_converter', 'sn_senior_supervisor', 'sas_senior_supervisor', 'sn_sales_manager', 'sas_sales_manager', 'sn_sales_deputy', 'sn_financial', 'sn_financial_approval', 'sn_finance', 'sn_hr', 'sn_mis', 'sn_after_sales', 'sn_shipping_expert', 'sn_operations_sales_manager', 'sn_operations_sales_supervisor', 'sn_operations_sales_expert', 'sn_operations_executive_manager' ];
	}

	private function is_customer_user( WP_User $user ): bool {
		$roles = array_map( 'sanitize_key', (array) $user->roles );
		return in_array( 'customer', $roles, true ) && ! array_intersect( $roles, $this->internal_roles() ) && ! user_can( $user, 'manage_options' );
	}

	public function filter_admin_bar( bool $show ): bool {
		if ( ! is_user_logged_in() ) { return $show; }
		$user = wp_get_current_user();
		return $user instanceof WP_User && $this->is_customer_user( $user ) ? false : $show;
	}

	private function user_phones( WP_User $user ): array {
		$phones = [];
		foreach ( [ $user->user_login, get_user_meta( $user->ID, 'billing_phone', true ), get_user_meta( $user->ID, 'digits_phone', true ), get_user_meta( $user->ID, 'digits_phone_no', true ), get_user_meta( $user->ID, 'sn_customer_phone', true ) ] as $raw ) {
			$phone = SN_Helpers::normalize_mobile( (string) $raw );
			if ( SN_Helpers::is_valid_mobile( $phone ) ) { $phones[] = $phone; }
		}
		return array_values( array_unique( $phones ) );
	}

	private function phone_variants( string $phone ): array {
		$phone = SN_Helpers::normalize_mobile( $phone );
		if ( ! SN_Helpers::is_valid_mobile( $phone ) ) { return []; }
		$national = substr( $phone, 1 );
		return array_values( array_unique( [ $phone, $national, '98' . $national, '+98' . $national, '0098' . $national ] ) );
	}

	private function customer_has_membership( int $user_id, string $phone ): bool {
		global $wpdb; $table = $wpdb->prefix . 'sn_project_memberships';
		if ( ! $this->table_exists( $table ) ) { return false; }
		$variants = $this->phone_variants( $phone );
		if ( ! $variants ) { return false; }
		$phones_sql = implode( ',', array_fill( 0, count( $variants ), '%s' ) );
		return (bool) $wpdb->get_var( $wpdb->prepare(
			"SELECT id FROM {$table} WHERE customer_wp_id=%d OR (COALESCE(customer_wp_id,0)=0 AND customer_phone IN ({$phones_sql})) ORDER BY id DESC LIMIT 1",
			...array_merge( [ $user_id ], $variants )
		) );
	}

	private function customer_user_for_phone( string $phone ) {
		global $wpdb;
		$phone = SN_Helpers::normalize_mobile( $phone );
		$variants = $this->phone_variants( $phone );
		if ( ! $variants ) { return new WP_Error( 'invalid_phone', 'شماره موبایل معتبر نیست.' ); }
		$placeholders = implode( ',', array_fill( 0, count( $variants ), '%s' ) );
		$meta_keys = [ 'billing_phone', 'digits_phone', 'digits_phone_no', 'sn_customer_phone', 'mobile', 'phone' ];
		$key_placeholders = implode( ',', array_fill( 0, count( $meta_keys ), '%s' ) );
		$user_ids = $wpdb->get_col( $wpdb->prepare(
			"SELECT ID FROM {$wpdb->users} WHERE user_login IN ({$placeholders}) UNION SELECT DISTINCT user_id ID FROM {$wpdb->usermeta} WHERE meta_key IN ({$key_placeholders}) AND meta_value IN ({$placeholders})",
			...array_merge( $variants, $meta_keys, $variants )
		) ) ?: [];
		$matches = [];
		foreach ( array_unique( array_map( 'intval', $user_ids ) ) as $user_id ) {
			$user = get_user_by( 'id', $user_id );
			if ( ! $user instanceof WP_User || ! $this->is_customer_user( $user ) ) { continue; }
			if ( $this->customer_has_membership( $user_id, $phone ) ) { $matches[ $user_id ] = $user; }
		}
		if ( count( $matches ) > 1 ) { return new WP_Error( 'duplicate_customer', 'این شماره به بیش از یک حساب مشتری متصل است؛ با پشتیبانی تماس بگیرید.' ); }
		if ( ! $matches ) { return new WP_Error( 'membership_not_found', 'برای این شماره، اشتراک پرداخت‌شده‌ای پیدا نشد.' ); }
		return reset( $matches );
	}

	private function otp_rate_limit( string $phone ) {
		$ip = sanitize_text_field( (string) ( $_SERVER['REMOTE_ADDR'] ?? 'unknown' ) );
		// Version the limiter namespace so stale counters from the earlier broken
		// login flow do not keep legitimate customers locked out after upgrading.
		$cooldown_key = 'sn_customer_otp_v2_cd_' . hash( 'sha256', $phone );
		if ( get_transient( $cooldown_key ) ) { return new WP_Error( 'otp_cooldown', 'برای ارسال مجدد کد، یک دقیقه صبر کنید.' ); }
		$phone_key = 'sn_customer_otp_v2_phone_' . hash( 'sha256', $phone );
		$ip_key = 'sn_customer_otp_v2_ip_' . hash( 'sha256', $ip );
		$phone_count = (int) get_transient( $phone_key );
		$ip_count = (int) get_transient( $ip_key );
		if ( $phone_count >= 6 || $ip_count >= 100 ) { return new WP_Error( 'otp_limited', 'تعداد درخواست‌ها بیش از حد مجاز است. کمی بعد دوباره تلاش کنید.' ); }
		set_transient( $cooldown_key, '1', MINUTE_IN_SECONDS );
		set_transient( $phone_key, $phone_count + 1, HOUR_IN_SECONDS );
		set_transient( $ip_key, $ip_count + 1, HOUR_IN_SECONDS );
		return true;
	}

	/** Return an unexpired challenge instead of sending another SMS. */
	private function reusable_otp( string $phone, int $user_id ): array {
		$active_key = $this->active_otp_key( $phone );
		$challenge = (string) get_transient( $active_key );
		if ( $challenge === '' ) { return []; }
		$key = $this->otp_key( $challenge );
		$data = get_transient( $key );
		$stored_phone = is_array( $data ) ? SN_Helpers::normalize_mobile( (string) ( $data['phone'] ?? '' ) ) : '';
		$created_at = is_array( $data ) ? (int) ( $data['created_at'] ?? 0 ) : 0;
		$valid = is_array( $data )
			&& absint( $data['user_id'] ?? 0 ) === $user_id
			&& $stored_phone !== ''
			&& hash_equals( $stored_phone, $phone )
			&& ! empty( $data['code_hash'] )
			&& (int) ( $data['attempts'] ?? 0 ) < self::OTP_MAX_ATTEMPTS
			&& $created_at > 0
			&& time() - $created_at < self::OTP_TTL;
		if ( ! $valid ) {
			delete_transient( $key );
			delete_transient( $active_key );
			return [];
		}
		return [
			'challenge' => $challenge,
			'expires_in' => max( 1, self::OTP_TTL - max( 0, time() - $created_at ) ),
		];
	}

	private function otp_key( string $challenge ): string {
		return 'sn_customer_portal_otp_' . hash( 'sha256', $challenge );
	}

	private function active_otp_key( string $phone ): string {
		return 'sn_customer_portal_active_otp_' . hash( 'sha256', $phone );
	}

	private function verify_otp_nonce(): bool {
		return (bool) check_ajax_referer( 'sn_customer_portal_otp', 'nonce', false );
	}

	private function send_sms( string $phone, string $message ): bool {
		if ( ! class_exists( 'SN_SMS' ) && defined( 'SN_PLUGIN_DIR' ) && file_exists( SN_PLUGIN_DIR . 'includes/class-sn-sms.php' ) ) {
			require_once SN_PLUGIN_DIR . 'includes/class-sn-sms.php';
		}
		$sent = class_exists( 'SN_SMS' ) && ( new SN_SMS() )->send( $phone, $message );
		if ( class_exists( 'SN_Notification_Service' ) ) { SN_Notification_Service::instance()->mirror_text( 'customer.portal_invite', $phone, $message, [ 'entity_type'=>'customer_portal', 'dedupe_key'=>'portal-invite-'.hash('sha256', $phone.'|'.$message), 'source'=>'customer_portal' ] ); }
		return $sent;
	}

	private function send_otp_sms( string $phone, string $code ): bool {
		if ( ! class_exists( 'SN_SMS' ) && defined( 'SN_PLUGIN_DIR' ) && file_exists( SN_PLUGIN_DIR . 'includes/class-sn-sms.php' ) ) {
			require_once SN_PLUGIN_DIR . 'includes/class-sn-sms.php';
		}
		$sent = class_exists( 'SN_SMS' ) && ( new SN_SMS() )->send_customer_portal_otp( $phone, $code );
		if ( $sent && class_exists( 'SN_Notification_Service' ) ) { SN_Notification_Service::instance()->mirror_otp( 'customer.portal_otp', $phone, $code, [ 'entity_type'=>'customer_portal', 'dedupe_key'=>'portal-otp-'.hash('sha256', $phone.'|'.$code), 'source'=>'customer_portal_otp' ] ); }
		return $sent;
	}

	public function ajax_send_otp(): void {
		nocache_headers();
		if ( ! $this->verify_otp_nonce() ) { wp_send_json( [ 'success' => false, 'message' => 'درخواست امنیتی نامعتبر است؛ صفحه را تازه کنید.' ], 403 ); }
		$phone = SN_Helpers::normalize_mobile( sanitize_text_field( wp_unslash( $_POST['phone'] ?? '' ) ) );
		if ( ! SN_Helpers::is_valid_mobile( $phone ) ) { wp_send_json( [ 'success' => false, 'message' => 'شماره موبایل معتبر نیست.' ], 400 ); }
		$user = $this->customer_user_for_phone( $phone );
		if ( is_wp_error( $user ) ) { wp_send_json( [ 'success' => false, 'message' => $user->get_error_message() ], 404 ); }
		$masked = substr( $phone, 0, 4 ) . '***' . substr( $phone, -4 );
		$reusable = $this->reusable_otp( $phone, (int) $user->ID );
		if ( $reusable ) {
			wp_send_json( [
				'success' => true,
				'message' => 'کد قبلی هنوز معتبر است؛ همان کد پیامک‌شده را وارد کنید.',
				'challenge' => (string) $reusable['challenge'],
				'masked_phone' => $masked,
				'expires_in' => (int) $reusable['expires_in'],
				'reused' => true,
			] );
		}
		$limited = $this->otp_rate_limit( $phone );
		if ( is_wp_error( $limited ) ) { wp_send_json( [ 'success' => false, 'message' => $limited->get_error_message() ], 429 ); }
		try { $code = (string) random_int( 100000, 999999 ); }
		catch ( Throwable $e ) { $code = (string) wp_rand( 100000, 999999 ); }
		$challenge = wp_generate_password( 48, false, false );
		$active_key = $this->active_otp_key( $phone );
		$previous_challenge = (string) get_transient( $active_key );
		if ( $previous_challenge !== '' ) { delete_transient( $this->otp_key( $previous_challenge ) ); }
		set_transient( $this->otp_key( $challenge ), [
			'user_id' => (int) $user->ID,
			'phone' => $phone,
			'code_hash' => wp_hash_password( $code ),
			'attempts' => 0,
			'created_at' => time(),
		], self::OTP_TTL );
		set_transient( $active_key, $challenge, self::OTP_TTL );
		if ( ! $this->send_otp_sms( $phone, $code ) ) {
			delete_transient( $this->otp_key( $challenge ) );
			delete_transient( $active_key );
			$provider = (string) get_option( 'sn_sms_provider', '' );
			$pattern = trim( (string) get_option( 'sn_faraz_pattern_customer_otp', '' ) );
			if ( $provider === 'faraz' && $pattern === '' ) {
				wp_send_json( [ 'success' => false, 'message' => 'کد پترن ورود مشتری در تنظیمات پیامک ثبت نشده است.' ], 422 );
			}
			wp_send_json( [ 'success' => false, 'message' => 'ارسال پترن ورود مشتری انجام نشد؛ کد پترن، نام متغیر و API Key فراز را بررسی کنید.' ], 503 );
		}
		wp_send_json( [ 'success' => true, 'message' => 'کد ورود ارسال شد.', 'challenge' => $challenge, 'masked_phone' => $masked, 'expires_in' => self::OTP_TTL ] );
	}

	public function ajax_verify_otp(): void {
		nocache_headers();
		if ( ! $this->verify_otp_nonce() ) { wp_send_json( [ 'success' => false, 'message' => 'درخواست امنیتی نامعتبر است؛ صفحه را تازه کنید.' ], 403 ); }
		$phone = SN_Helpers::normalize_mobile( sanitize_text_field( wp_unslash( $_POST['phone'] ?? '' ) ) );
		$challenge = sanitize_text_field( wp_unslash( $_POST['challenge'] ?? '' ) );
		$code = preg_replace( '/\D+/', '', SN_Helpers::to_english_nums( sanitize_text_field( wp_unslash( $_POST['code'] ?? '' ) ) ) );
		if ( ! SN_Helpers::is_valid_mobile( $phone ) || strlen( $challenge ) < 30 || ! preg_match( '/^\d{6}$/', $code ) ) {
			wp_send_json( [ 'success' => false, 'message' => 'اطلاعات کد ورود معتبر نیست.' ], 400 );
		}
		$key = $this->otp_key( $challenge );
		$data = get_transient( $key );
		if ( ! is_array( $data ) ) { wp_send_json( [ 'success' => false, 'message' => 'کد منقضی شده است؛ کد جدید دریافت کنید.' ], 410 ); }
		$active_key = $this->active_otp_key( $phone );
		$active_challenge = (string) get_transient( $active_key );
		if ( $active_challenge === '' || ! hash_equals( $active_challenge, $challenge ) ) {
			delete_transient( $key );
			wp_send_json( [ 'success' => false, 'message' => 'این کد با ارسال کد جدید غیرفعال شده است.' ], 410 );
		}
		$stored_phone = SN_Helpers::normalize_mobile( (string) ( $data['phone'] ?? '' ) );
		if ( $stored_phone === '' || ! hash_equals( $stored_phone, $phone ) ) { delete_transient( $key ); delete_transient( $active_key ); wp_send_json( [ 'success' => false, 'message' => 'کد متعلق به این شماره نیست.' ], 400 ); }
		$attempts = (int) ( $data['attempts'] ?? 0 );
		if ( $attempts >= self::OTP_MAX_ATTEMPTS ) { delete_transient( $key ); delete_transient( $active_key ); wp_send_json( [ 'success' => false, 'message' => 'تعداد تلاش‌ها بیش از حد مجاز بود؛ کد جدید دریافت کنید.' ], 429 ); }
		if ( empty( $data['code_hash'] ) || ! wp_check_password( $code, (string) $data['code_hash'] ) ) {
			$data['attempts'] = $attempts + 1;
			if ( $data['attempts'] >= self::OTP_MAX_ATTEMPTS ) {
				delete_transient( $key );
				delete_transient( $active_key );
				wp_send_json( [ 'success' => false, 'message' => 'کد صحیح نبود و فرصت‌های ورود تمام شد؛ کد جدید دریافت کنید.', 'remaining_attempts' => 0 ], 429 );
			}
			$ttl = max( 1, self::OTP_TTL - max( 0, time() - (int) ( $data['created_at'] ?? time() ) ) );
			set_transient( $key, $data, $ttl );
			wp_send_json( [ 'success' => false, 'message' => 'کد واردشده صحیح نیست.', 'remaining_attempts' => max( 0, self::OTP_MAX_ATTEMPTS - $data['attempts'] ) ], 400 );
		}
		$user = get_user_by( 'id', absint( $data['user_id'] ?? 0 ) );
		if ( ! $user instanceof WP_User || ! $this->is_customer_user( $user ) || ! $this->customer_has_membership( (int) $user->ID, $phone ) ) {
			delete_transient( $key );
			delete_transient( $active_key );
			wp_send_json( [ 'success' => false, 'message' => 'حساب مشتری دیگر قابل ورود نیست.' ], 403 );
		}
		try {
			// Do not clear cookies for the normal anonymous OTP request. Clearing all
			// auth cookies fires host/security integrations and was one source of slow
			// upstream responses on the customer-login endpoint.
			if ( is_user_logged_in() && get_current_user_id() !== (int) $user->ID ) { wp_clear_auth_cookie(); }
			wp_set_current_user( (int) $user->ID );
			wp_set_auth_cookie( (int) $user->ID, true, is_ssl() );
			update_user_meta( (int) $user->ID, 'sn_biavin_last_login_at', current_time( 'mysql' ) );
		} catch ( Throwable $e ) {
			error_log( 'SN customer portal OTP login failed for user #' . (int) $user->ID . ': ' . $e->getMessage() );
			wp_set_current_user( 0 );
			wp_send_json( [ 'success' => false, 'message' => 'ایجاد نشست ورود انجام نشد؛ دوباره تلاش کنید.' ], 500 );
		}
		// Consume the OTP only after the authentication cookie has been created.
		delete_transient( $key );
		delete_transient( $active_key );
		// No login action is fired inside this AJAX response. Host integrations may
		// perform redirects or remote work on login hooks and prevent JSON from
		// reaching the browser before the upstream timeout.
		wp_send_json( [ 'success' => true, 'message' => 'ورود با موفقیت انجام شد.', 'redirect_url' => $this->portal_url() ] );
	}

	private function insert_event( int $membership_id, string $event_key, array $details = [] ): void {
		global $wpdb; $table = $wpdb->prefix . 'sn_project_events';
		if ( ! $this->table_exists( $table ) ) { return; }
		$wpdb->insert( $table, [
			'membership_id' => $membership_id,
			'membership_item_id' => null,
			'actor_user_id' => null,
			'event_key' => sanitize_key( $event_key ),
			'details' => wp_json_encode( $details, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ),
			'created_at' => current_time( 'mysql' ),
		] );
	}

	/** Send one idempotent portal invite after a new fully-paid membership exists. */
	public function send_membership_invite( int $membership_id, int $invoice_id, $invoice = null ): void {
		global $wpdb; $table = $wpdb->prefix . 'sn_project_memberships';
		if ( $membership_id < 1 || ! $this->table_exists( $table ) ) { return; }
		$this->maybe_upgrade();
		$row = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$table} WHERE id=%d LIMIT 1", $membership_id ) );
		if ( ! $row || (string) $row->status !== 'active' || ! empty( $row->portal_invite_sent_at ) || (int) ( $row->portal_invite_attempts ?? 0 ) >= self::INVITE_MAX_ATTEMPTS ) { return; }
		if ( (string) get_option( 'sn_customer_portal_invite_sms_enabled', '1' ) !== '1' ) {
			wp_clear_scheduled_hook( self::INVITE_RETRY_HOOK, [ $membership_id, $invoice_id ] );
			$wpdb->update( $table, [ 'portal_invite_status' => 'disabled', 'updated_at' => current_time( 'mysql' ) ], [ 'id' => $membership_id ] );
			$this->insert_event( $membership_id, 'customer_portal_invite_skipped', [ 'invoice_id' => $invoice_id, 'reason' => 'setting_disabled' ] );
			return;
		}
		$now = current_time( 'mysql' );
		$retry_before = SN_Helpers::site_mysql_from_timestamp( time() - 300 );
		$claimed = $wpdb->query( $wpdb->prepare(
			"UPDATE {$table} SET portal_invite_status='sending',portal_invite_attempts=portal_invite_attempts+1,portal_invite_attempted_at=%s,updated_at=%s WHERE id=%d AND portal_invite_sent_at IS NULL AND portal_invite_attempts<%d AND (portal_invite_attempted_at IS NULL OR portal_invite_attempted_at<%s)",
			$now, $now, $membership_id, self::INVITE_MAX_ATTEMPTS, $retry_before
		) );
		if ( 1 !== (int) $claimed ) { return; }
		$attempt = (int) ( $row->portal_invite_attempts ?? 0 ) + 1;
		$phone = SN_Helpers::normalize_mobile( (string) $row->customer_phone );
		$user_id = (int) ( $row->customer_wp_id ?? 0 );
		$user = $user_id > 0 ? get_user_by( 'id', $user_id ) : null;
		if ( ! $user instanceof WP_User && SN_Helpers::is_valid_mobile( $phone ) ) {
			$resolved = $this->customer_user_for_phone( $phone );
			if ( $resolved instanceof WP_User ) { $user = $resolved; }
		}
		if ( ! SN_Helpers::is_valid_mobile( $phone ) || ! $user instanceof WP_User || ! $this->is_customer_user( $user ) ) {
			$wpdb->update( $table, [ 'portal_invite_status' => 'failed', 'updated_at' => current_time( 'mysql' ) ], [ 'id' => $membership_id ] );
			$this->insert_event( $membership_id, 'customer_portal_invite_failed', [ 'invoice_id' => $invoice_id, 'reason' => 'customer_identity_unavailable', 'attempt' => $attempt ] );
			$this->schedule_invite_retry( $membership_id, $invoice_id, $attempt );
			return;
		}
		// The paid hook may receive an invoice object created before the core hook
		// attached its customer account. Persist the safely resolved identity here so
		// future OTP/profile reads use the immutable user id instead of phone fallback.
		if ( $user_id < 1 ) {
			$wpdb->query( $wpdb->prepare(
				"UPDATE {$table} SET customer_wp_id=%d,updated_at=%s WHERE id=%d AND COALESCE(customer_wp_id,0)=0",
				(int) $user->ID,
				current_time( 'mysql' ),
				$membership_id
			) );
		}
		$name = sanitize_text_field( (string) ( $row->customer_name ?: $user->display_name ) ) ?: 'مشتری گرامی';
		$message = "{$name} عزیز\nخرید شما با موفقیت وارد چرخه بیاوین شد.\nبرای مشاهده کارت‌ها و ادامه فرآیند فعال‌سازی، از لینک زیر وارد پروفایل شوید:\n" . $this->portal_url();
		$sent = $this->send_sms( $phone, $message );
		if ( $sent ) {
			wp_clear_scheduled_hook( self::INVITE_RETRY_HOOK, [ $membership_id, $invoice_id ] );
			$wpdb->update( $table, [ 'portal_invite_status' => 'sent', 'portal_invite_sent_at' => current_time( 'mysql' ), 'updated_at' => current_time( 'mysql' ) ], [ 'id' => $membership_id ] );
			$this->insert_event( $membership_id, 'customer_portal_invite_sent', [ 'invoice_id' => $invoice_id, 'attempt' => $attempt ] );
		} else {
			$wpdb->update( $table, [ 'portal_invite_status' => 'failed', 'updated_at' => current_time( 'mysql' ) ], [ 'id' => $membership_id ] );
			$this->insert_event( $membership_id, 'customer_portal_invite_failed', [ 'invoice_id' => $invoice_id, 'reason' => 'sms_failed', 'attempt' => $attempt ] );
			$this->schedule_invite_retry( $membership_id, $invoice_id, $attempt );
		}
	}

	private function schedule_invite_retry( int $membership_id, int $invoice_id, int $attempt ): void {
		if ( $attempt >= self::INVITE_MAX_ATTEMPTS || wp_next_scheduled( self::INVITE_RETRY_HOOK, [ $membership_id, $invoice_id ] ) ) { return; }
		wp_schedule_single_event( time() + ( 10 * max( 1, $attempt ) * MINUTE_IN_SECONDS ), self::INVITE_RETRY_HOOK, [ $membership_id, $invoice_id ] );
	}

	private function customer_memberships( WP_User $user ): array {
		global $wpdb; $memberships = $wpdb->prefix . 'sn_project_memberships'; $items = $wpdb->prefix . 'sn_project_membership_items';
		if ( ! $this->table_exists( $memberships ) || ! $this->table_exists( $items ) ) { return []; }
		$phones = [];
		foreach ( $this->user_phones( $user ) as $phone ) { $phones = array_merge( $phones, $this->phone_variants( $phone ) ); }
		$phones = array_values( array_unique( $phones ) );
		$where = 'm.customer_wp_id=%d'; $args = [ (int) $user->ID ];
		if ( $phones ) {
			$where .= ' OR (COALESCE(m.customer_wp_id,0)=0 AND m.customer_phone IN (' . implode( ',', array_fill( 0, count( $phones ), '%s' ) ) . ')';
			$args = array_merge( $args, $phones );
			$where .= ')';
		}
		$sql = "SELECT m.* FROM {$memberships} m WHERE ({$where}) ORDER BY m.activated_at DESC,m.id DESC LIMIT 50";
		$rows = $wpdb->get_results( $wpdb->prepare( $sql, ...$args ), ARRAY_A ) ?: [];
		$subscription_colors = [];
		if ( class_exists( 'SN_Dot_Flow' ) ) {
			$dot_config = SN_Dot_Flow::instance()->config();
			foreach ( (array) ( $dot_config['options'] ?? [] ) as $dot_option ) {
				$product_id = absint( $dot_option['product_id'] ?? 0 );
				$color = sanitize_hex_color( (string) ( $dot_option['color'] ?? '' ) );
				if ( $product_id > 0 && $color ) { $subscription_colors[ $product_id ] = $color; }
			}
		}
		foreach ( $rows as &$membership ) {
			$membership_id = (int) $membership['id'];
			$operations = $wpdb->prefix . 'sn_project_operations';
			$operation_select = $this->table_exists( $operations ) ? ',o.stage operations_stage,o.activation_mode operations_mode,o.current_credit operations_current_credit' : ",'awaiting_customer' operations_stage,NULL operations_mode,NULL operations_current_credit";
			$operation_join = $this->table_exists( $operations ) ? " LEFT JOIN {$operations} o ON o.membership_item_id=mi.id" : '';
			$membership['items'] = $wpdb->get_results( $wpdb->prepare(
				"SELECT mi.*{$operation_select},(SELECT a.status FROM {$wpdb->prefix}sn_project_actions a WHERE a.membership_item_id=mi.id ORDER BY a.id DESC LIMIT 1) latest_action_status,(SELECT a.action_type FROM {$wpdb->prefix}sn_project_actions a WHERE a.membership_item_id=mi.id ORDER BY a.id DESC LIMIT 1) latest_action_type FROM {$items} mi{$operation_join} WHERE mi.membership_id=%d ORDER BY mi.id",
				$membership_id
			), ARRAY_A ) ?: [];
			foreach ( $membership['items'] as &$item ) {
				$content_product_id = absint( $item['content_product_id'] ?? 0 );
				$content_product = $content_product_id && function_exists( 'wc_get_product' ) ? wc_get_product( $content_product_id ) : null;
				$options = json_decode( (string) ( $item['upgrade_options_snapshot_json'] ?? '' ), true );
				$item['upgrade_options'] = is_array( $options ) ? $options : [];
				$item['base_credit'] = null !== ( $item['base_credit_snapshot'] ?? null ) ? max( 0, (float) $item['base_credit_snapshot'] ) : max( 0, (float) get_post_meta( $content_product_id, '_sn_product_credit_amount', true ) );
				$item['credit'] = null !== ( $item['operations_current_credit'] ?? null ) ? max( 0, (float) $item['operations_current_credit'] ) : $item['base_credit'];
				$image_id = $content_product && method_exists( $content_product, 'get_image_id' ) ? absint( $content_product->get_image_id() ) : absint( get_post_thumbnail_id( $content_product_id ) );
				$item['image'] = $image_id ? (string) wp_get_attachment_image_url( $image_id, 'large' ) : '';
				$description = $content_product && method_exists( $content_product, 'get_description' ) ? (string) $content_product->get_description() : (string) get_post_field( 'post_content', $content_product_id );
				if ( $description !== '' && function_exists( 'do_blocks' ) ) { $description = do_blocks( $description ); }
				$item['description_html'] = $description !== '' ? wp_kses_post( wpautop( $description ) ) : '';
			}
			unset( $item );
			$membership['image'] = (string) get_the_post_thumbnail_url( (int) $membership['subscription_product_id'], 'large' );
			$membership['theme_color'] = $subscription_colors[ absint( $membership['subscription_product_id'] ?? 0 ) ] ?? '#177c91';
		}
		unset( $membership );
		return $rows;
	}

	private function customer_invoices( WP_User $user ): array {
		global $wpdb; $table = $wpdb->prefix . 'sn_invoices';
		if ( ! $this->table_exists( $table ) ) { return []; }
		$phones = [];
		foreach ( $this->user_phones( $user ) as $phone ) { $phones = array_merge( $phones, $this->phone_variants( $phone ) ); }
		$phones = array_values( array_unique( $phones ) );
		$where = 'customer_wp_id=%d'; $args = [ (int) $user->ID ];
		if ( $phones ) { $where .= ' OR (COALESCE(customer_wp_id,0)=0 AND customer_phone IN (' . implode( ',', array_fill( 0, count( $phones ), '%s' ) ) . '))'; $args = array_merge( $args, $phones ); }
		$rows = $wpdb->get_results( $wpdb->prepare(
			"SELECT id,invoice_code,wc_order_id,status,invoice_status,payment_status,payment_workflow_status,pay_method,payment_source,province,city,COALESCE(final_total,payment_total_amount,product_price,0) amount,created_at,paid_at,approved_at,payment_completed_at FROM {$table} WHERE ({$where}) ORDER BY id DESC LIMIT 100",
			...$args
		), ARRAY_A ) ?: [];
		$invoice_ids = array_values( array_filter( array_map( static fn( $row ) => absint( $row['id'] ?? 0 ), $rows ) ) );
		$items_table = $wpdb->prefix . 'sn_invoice_items';
		$items_by_invoice = [];
		if ( $invoice_ids && $this->table_exists( $items_table ) ) {
			$placeholders = implode( ',', array_fill( 0, count( $invoice_ids ), '%d' ) );
			$items = $wpdb->get_results( $wpdb->prepare(
				"SELECT invoice_id,product_id,product_name,product_type,qty FROM {$items_table} WHERE invoice_id IN ({$placeholders}) ORDER BY id ASC",
				...$invoice_ids
			), ARRAY_A ) ?: [];
			foreach ( $items as $item ) { $items_by_invoice[ absint( $item['invoice_id'] ?? 0 ) ][] = $item; }
		}
		foreach ( $rows as &$row ) {
			$row['items'] = $items_by_invoice[ absint( $row['id'] ?? 0 ) ] ?? [];
			$row['purchase_title'] = $this->invoice_purchase_title( $row );
		}
		unset( $row );
		return $rows;
	}

	private function invoice_purchase_title( array $invoice ): string {
		$parts = [];
		foreach ( (array) ( $invoice['items'] ?? [] ) as $item ) {
			$name = sanitize_text_field( (string) ( $item['product_name'] ?? '' ) );
			if ( $name === '' ) { continue; }
			$qty = max( 1, absint( $item['qty'] ?? 1 ) );
			$parts[] = $name . ( $qty > 1 ? ' × ' . number_format_i18n( $qty ) : '' );
		}
		return $parts ? implode( '، ', $parts ) : 'خرید ثبت‌شده در فاکتور';
	}

	private function payment_reason( array $invoice, int $stage_no, string $stage_type ): string {
		$type_labels = [
			'subscription' => 'خرید اشتراک', 'assessment' => 'هزینه اعتبارسنجی',
			'product' => 'خرید محصول', 'product_star' => 'خرید محصول ویژه',
			'subscription_star' => 'خرید محصول ویژه', 'project_upgrade' => 'افزایش اعتبار کارت',
		];
		$reasons = [];
		foreach ( (array) ( $invoice['items'] ?? [] ) as $item ) {
			$name = sanitize_text_field( (string) ( $item['product_name'] ?? '' ) );
			$type = sanitize_key( (string) ( $item['product_type'] ?? '' ) );
			$label = $type_labels[ $type ] ?? 'پرداخت بابت';
			$reasons[] = trim( $label . ' ' . $name );
		}
		$reason = $reasons ? implode( '، ', array_values( array_unique( $reasons ) ) ) : 'پرداخت فاکتور ' . sanitize_text_field( (string) ( $invoice['invoice_code'] ?? '' ) );
		$stage_label = [ 'partial' => 'پیش‌پرداخت', 'remaining' => 'تکمیل مانده', 'full' => 'پرداخت کامل' ][ sanitize_key( $stage_type ) ] ?? '';
		if ( $stage_label === '' && $stage_no > 1 ) { $stage_label = 'مرحله ' . number_format_i18n( $stage_no ); }
		return $stage_label !== '' ? $stage_label . ' — ' . $reason : $reason;
	}

	private function customer_location( WP_User $user, array $invoices, array $memberships ): array {
		global $wpdb;
		$first_meta = static function ( int $user_id, array $keys ): string {
			foreach ( $keys as $key ) {
				$value = sanitize_text_field( (string) get_user_meta( $user_id, $key, true ) );
				if ( $value !== '' ) { return $value; }
			}
			return '';
		};
		$province = $first_meta( (int) $user->ID, [ 'sn_customer_province', 'billing_state', 'shipping_state', 'province', 'state' ] );
		$city = $first_meta( (int) $user->ID, [ 'sn_customer_city', 'billing_city', 'shipping_city', 'city' ] );

		// Membership ownership has already been scoped to this authenticated
		// customer. Its immutable source invoice is therefore the safest fallback
		// when old invoices were created before customer_wp_id was populated.
		$source_invoice_ids = array_values( array_unique( array_filter( array_map( static fn( $row ) => absint( $row['source_invoice_id'] ?? 0 ), $memberships ) ) ) );
		$dot_case_ids = array_values( array_unique( array_filter( array_map( static fn( $row ) => absint( $row['dot_case_id'] ?? 0 ), $memberships ) ) ) );
		$scoped_invoices = [];
		if ( $source_invoice_ids ) {
			$invoice_table = $wpdb->prefix . 'sn_invoices';
			if ( $this->table_exists( $invoice_table ) ) {
				$placeholders = implode( ',', array_fill( 0, count( $source_invoice_ids ), '%d' ) );
				$scoped_invoices = $wpdb->get_results( $wpdb->prepare(
					"SELECT id,wc_order_id,province,city FROM {$invoice_table} WHERE id IN ({$placeholders}) ORDER BY id DESC",
					...$source_invoice_ids
				), ARRAY_A ) ?: [];
			}
		}
		$location_rows = array_merge( $scoped_invoices, $invoices );
		foreach ( $location_rows as $invoice ) {
			if ( $province === '' ) { $province = sanitize_text_field( (string) ( $invoice['province'] ?? '' ) ); }
			if ( $city === '' ) { $city = sanitize_text_field( (string) ( $invoice['city'] ?? '' ) ); }
			if ( ( $province === '' || $city === '' ) && ! empty( $invoice['wc_order_id'] ) && function_exists( 'wc_get_order' ) ) {
				$order = wc_get_order( absint( $invoice['wc_order_id'] ) );
				if ( $order ) {
					if ( $province === '' ) { $province = sanitize_text_field( (string) $order->get_billing_state() ); }
					if ( $city === '' ) { $city = sanitize_text_field( (string) $order->get_billing_city() ); }
				}
			}
			if ( $province !== '' && $city !== '' ) { break; }
		}
		if ( ( $province === '' || $city === '' ) && $dot_case_ids ) {
			$dot_table = $wpdb->prefix . 'sn_dot_cases';
			if ( $this->table_exists( $dot_table ) ) {
				$placeholders = implode( ',', array_fill( 0, count( $dot_case_ids ), '%d' ) );
				$dot_rows = $wpdb->get_results( $wpdb->prepare(
					"SELECT province,city FROM {$dot_table} WHERE id IN ({$placeholders}) ORDER BY id DESC",
					...$dot_case_ids
				), ARRAY_A ) ?: [];
				foreach ( $dot_rows as $row ) {
					if ( $province === '' ) { $province = sanitize_text_field( (string) ( $row['province'] ?? '' ) ); }
					if ( $city === '' ) { $city = sanitize_text_field( (string) ( $row['city'] ?? '' ) ); }
					if ( $province !== '' && $city !== '' ) { break; }
				}
			}
		}
		if ( $province !== '' && function_exists( 'WC' ) && WC() && isset( WC()->countries ) ) {
			$states = (array) WC()->countries->get_states( 'IR' );
			if ( isset( $states[ $province ] ) ) { $province = sanitize_text_field( (string) $states[ $province ] ); }
		}
		return [ $province, $city ];
	}

	/**
	 * Payment records are resolved only through invoices already authorized for
	 * the current customer. This prevents phone-like searches from exposing a
	 * transaction that belongs to another account.
	 */
	private function customer_payments( array $invoices ): array {
		global $wpdb;
		$invoice_ids = array_values( array_unique( array_filter( array_map( static fn( $row ) => absint( $row['id'] ?? 0 ), $invoices ) ) ) );
		if ( ! $invoice_ids ) { return []; }

		$payments_table = $wpdb->prefix . 'sn_payments';
		$invoice_table = $wpdb->prefix . 'sn_invoices';
		$rows = [];
		$invoices_by_id = [];
		foreach ( $invoices as $invoice ) { $invoices_by_id[ absint( $invoice['id'] ?? 0 ) ] = $invoice; }
		if ( $this->table_exists( $payments_table ) ) {
			$placeholders = implode( ',', array_fill( 0, count( $invoice_ids ), '%d' ) );
			$stage_table = $wpdb->prefix . 'sn_invoice_payment_stages';
			$stage_select = $this->table_exists( $stage_table ) ? ',s.stage_type' : ",'' stage_type";
			$stage_join = $this->table_exists( $stage_table ) ? " LEFT JOIN {$stage_table} s ON s.invoice_id=p.invoice_id AND s.stage_no=p.payment_stage_no" : '';
			$sql = "SELECT p.id,p.invoice_id,p.payment_stage_no,p.ref_id,p.amount,p.status,p.pay_method,p.payment_source,p.created_at,p.updated_at,i.invoice_code{$stage_select}
				FROM {$payments_table} p
				INNER JOIN {$invoice_table} i ON i.id=p.invoice_id
				{$stage_join}
				WHERE p.invoice_id IN ({$placeholders})
				ORDER BY p.id DESC LIMIT 100";
			$rows = $wpdb->get_results( $wpdb->prepare( $sql, ...$invoice_ids ), ARRAY_A ) ?: [];
		}

		$successful_invoice_ids = [];
		foreach ( $rows as &$row ) {
			$row['invoice_id'] = absint( $row['invoice_id'] ?? 0 );
			$row['payment_stage_no'] = max( 1, absint( $row['payment_stage_no'] ?? 1 ) );
			$row['amount'] = max( 0, (float) ( $row['amount'] ?? 0 ) );
			$row['status'] = sanitize_key( (string) ( $row['status'] ?? '' ) );
			$row['display_at'] = (string) ( $row['updated_at'] ?: $row['created_at'] );
			$row['reason'] = $this->payment_reason( $invoices_by_id[ $row['invoice_id'] ] ?? [], $row['payment_stage_no'], (string) ( $row['stage_type'] ?? '' ) );
			if ( in_array( $row['status'], [ 'paid', 'approved', 'verified' ], true ) ) { $successful_invoice_ids[ $row['invoice_id'] ] = true; }
		}
		unset( $row );

		// Older manual payments may have completed the invoice before sn_payments
		// became mandatory. Add one explicit fallback so those purchases are not lost.
		foreach ( $invoices as $invoice ) {
			$invoice_id = absint( $invoice['id'] ?? 0 );
			$statuses = array_map( 'sanitize_key', [
				(string) ( $invoice['status'] ?? '' ),
				(string) ( $invoice['invoice_status'] ?? '' ),
				(string) ( $invoice['payment_status'] ?? '' ),
			] );
			$confirmed_status = in_array( 'approved', $statuses, true ) ? 'approved' : ( in_array( 'paid', $statuses, true ) ? 'paid' : '' );
			if ( $invoice_id < 1 || $confirmed_status === '' || isset( $successful_invoice_ids[ $invoice_id ] ) ) { continue; }
			$rows[] = [
				'id' => 'invoice-' . $invoice_id,
				'invoice_id' => $invoice_id,
				'invoice_code' => sanitize_text_field( (string) ( $invoice['invoice_code'] ?? '' ) ),
				'payment_stage_no' => 1,
				'ref_id' => '',
				'amount' => max( 0, (float) ( $invoice['amount'] ?? 0 ) ),
				'status' => $confirmed_status,
				'pay_method' => sanitize_key( (string) ( $invoice['pay_method'] ?? '' ) ),
				'payment_source' => sanitize_key( (string) ( $invoice['payment_source'] ?? '' ) ),
				'stage_type' => 'full',
				'reason' => $this->payment_reason( $invoice, 1, 'full' ),
				'created_at' => (string) ( $invoice['created_at'] ?? '' ),
				'updated_at' => (string) ( $invoice['payment_completed_at'] ?: ( $invoice['paid_at'] ?: ( $invoice['approved_at'] ?: $invoice['created_at'] ) ) ),
				'display_at' => (string) ( $invoice['payment_completed_at'] ?: ( $invoice['paid_at'] ?: ( $invoice['approved_at'] ?: $invoice['created_at'] ) ) ),
				'legacy_fallback' => true,
			];
		}

		usort( $rows, static fn( $a, $b ) => strcmp( (string) ( $b['display_at'] ?? '' ), (string) ( $a['display_at'] ?? '' ) ) );
		return array_slice( $rows, 0, 100 );
	}

	/**
	 * Return exact subscription product ids already purchased by one customer.
	 * Membership snapshots are authoritative; confirmed legacy invoices are a fallback.
	 */
	public function purchased_subscription_product_ids( string $phone, int $customer_wp_id = 0, int $exclude_invoice_id = 0 ): array {
		global $wpdb;
		$phone = SN_Helpers::normalize_mobile( $phone );
		$variants = $this->phone_variants( $phone );
		if ( ! $variants && $customer_wp_id < 1 ) { return []; }
		$product_ids = [];
		$identity = [];
		$args = [];
		if ( $customer_wp_id > 0 ) { $identity[] = 'customer_wp_id=%d'; $args[] = $customer_wp_id; }
		if ( $variants ) {
			$identity[] = 'customer_phone IN (' . implode( ',', array_fill( 0, count( $variants ), '%s' ) ) . ')';
			$args = array_merge( $args, $variants );
		}
		$memberships = $wpdb->prefix . 'sn_project_memberships';
		if ( $identity && $this->table_exists( $memberships ) ) {
			$where = '(' . implode( ' OR ', $identity ) . ')';
			if ( $exclude_invoice_id > 0 ) { $where .= ' AND COALESCE(source_invoice_id,0)<>%d'; $args[] = $exclude_invoice_id; }
			$sql = "SELECT DISTINCT subscription_product_id FROM {$memberships} WHERE subscription_product_id>0 AND {$where}";
			$product_ids = array_merge( $product_ids, array_map( 'intval', (array) $wpdb->get_col( $wpdb->prepare( $sql, ...$args ) ) ) );
		}

		$invoices = $wpdb->prefix . 'sn_invoices';
		$items = $wpdb->prefix . 'sn_invoice_items';
		if ( $identity && $this->table_exists( $invoices ) && $this->table_exists( $items ) ) {
			$invoice_args = [];
			$invoice_identity = [];
			if ( $customer_wp_id > 0 ) { $invoice_identity[] = 'i.customer_wp_id=%d'; $invoice_args[] = $customer_wp_id; }
			if ( $variants ) {
				$invoice_identity[] = 'i.customer_phone IN (' . implode( ',', array_fill( 0, count( $variants ), '%s' ) ) . ')';
				$invoice_args = array_merge( $invoice_args, $variants );
			}
			$where = '(' . implode( ' OR ', $invoice_identity ) . ") AND (i.status IN ('paid','approved','completed') OR i.invoice_status IN ('paid','approved','completed') OR i.payment_status IN ('paid','approved','completed'))";
			// Dot conversion creates one invoice per deposit/balance. A paid deposit
			// is not a completed subscription purchase and must never make the next
			// payment look like a duplicate purchase. Completed Dot subscriptions
			// are represented by the membership snapshot above, so every synthetic
			// conversion-payment invoice is excluded from this legacy fallback.
			$dot_links = $wpdb->prefix . 'sn_dot_invoice_links';
			if ( $this->table_exists( $dot_links ) ) {
				$where .= " AND NOT EXISTS (SELECT 1 FROM {$dot_links} dl WHERE dl.invoice_id=i.id AND dl.flow_kind='conversion_payment')";
			}
			if ( $exclude_invoice_id > 0 ) { $where .= ' AND i.id<>%d'; $invoice_args[] = $exclude_invoice_id; }
			$sql = "SELECT DISTINCT ii.product_id,ii.product_type FROM {$items} ii INNER JOIN {$invoices} i ON i.id=ii.invoice_id WHERE {$where}";
			foreach ( $wpdb->get_results( $wpdb->prepare( $sql, ...$invoice_args ), ARRAY_A ) ?: [] as $legacy_item ) {
				$product_id = absint( $legacy_item['product_id'] ?? 0 );
				$type = sanitize_key( (string) ( $legacy_item['product_type'] ?? '' ) );
				if ( $type === '' && $product_id && class_exists( 'SN_Seller_Flow' ) ) { $type = SN_Seller_Flow::instance()->product_type( $product_id ); }
				if ( $product_id && $type === 'subscription' ) { $product_ids[] = $product_id; }
			}
		}
		$product_ids = array_values( array_unique( array_filter( array_map( 'absint', $product_ids ) ) ) );
		sort( $product_ids, SORT_NUMERIC );
		return $product_ids;
	}

	private function current_user_can_check_customer_purchases(): bool {
		if ( current_user_can( 'manage_options' ) ) { return true; }
		$user = wp_get_current_user();
		if ( array_intersect( [ 'sn_seller', 'sn_converter' ], (array) $user->roles ) ) { return true; }
		global $wpdb;
		$profiles = $wpdb->prefix . 'sn_hr_profiles';
		$positions = $wpdb->prefix . 'sn_hr_positions';
		if ( ! $this->table_exists( $profiles ) || ! $this->table_exists( $positions ) ) { return false; }
		$slug = (string) $wpdb->get_var( $wpdb->prepare( "SELECT pos.slug FROM {$profiles} p INNER JOIN {$positions} pos ON pos.id=p.position_id WHERE p.user_id=%d AND p.is_active=1 LIMIT 1", get_current_user_id() ) );
		return in_array( $slug, [ 'seller', 'converter' ], true );
	}

	public function ajax_customer_purchased_subscriptions(): void {
		if ( ! is_user_logged_in() || ! $this->current_user_can_check_customer_purchases() ) { wp_send_json_error( [ 'message' => 'دسترسی مجاز نیست.' ], 403 ); }
		$nonce = sanitize_text_field( wp_unslash( $_POST['nonce'] ?? '' ) );
		if ( ! wp_verify_nonce( $nonce, 'sn_public' ) && ! wp_verify_nonce( $nonce, 'sn_admin' ) ) { wp_send_json_error( [ 'message' => 'درخواست امنیتی نامعتبر است.' ], 403 ); }
		$phone = SN_Helpers::normalize_mobile( sanitize_text_field( wp_unslash( $_POST['phone'] ?? '' ) ) );
		if ( ! SN_Helpers::is_valid_mobile( $phone ) ) { wp_send_json_error( [ 'message' => 'شماره موبایل معتبر نیست.' ], 400 ); }
		$product_ids = $this->purchased_subscription_product_ids( $phone );
		wp_send_json_success( [ 'product_ids' => $product_ids, 'count' => count( $product_ids ) ] );
	}

	private function payment_status_label( string $status ): string {
		return [
			'approved' => 'تأییدشده', 'paid' => 'پرداخت‌شده', 'verified' => 'تأیید درگاه',
			'pending' => 'در انتظار پرداخت', 'receipt_uploaded' => 'در انتظار تأیید فیش',
			'pending_financial_approval' => 'در انتظار تأیید مالی', 'reconcile_error' => 'نیازمند بررسی',
			'rejected' => 'ردشده', 'cancelled' => 'لغوشده', 'expired' => 'منقضی‌شده', 'failed' => 'ناموفق',
		][ sanitize_key( $status ) ] ?? ( $status !== '' ? $status : 'نامشخص' );
	}

	private function payment_status_kind( string $status ): string {
		$status = sanitize_key( $status );
		if ( in_array( $status, [ 'approved', 'paid', 'verified' ], true ) ) { return 'success'; }
		if ( in_array( $status, [ 'rejected', 'cancelled', 'expired', 'failed', 'reconcile_error' ], true ) ) { return 'failed'; }
		return 'pending';
	}

	private function payment_channel_label( array $payment ): string {
		$method = sanitize_key( (string) ( $payment['pay_method'] ?? '' ) );
		$source = sanitize_key( (string) ( $payment['payment_source'] ?? '' ) );
		$method_label = [ 'card' => 'کارت‌به‌کارت', 'card_to_card' => 'کارت‌به‌کارت', 'online' => 'پرداخت آنلاین', 'gateway' => 'پرداخت آنلاین' ][ $method ] ?? '';
		$source_label = [
			'zarinpal' => 'زرین‌پال', 'zibal' => 'زیبال', 'asanpardakht' => 'آسان‌پرداخت',
			'customer_upload' => 'فیش مشتری', 'supervisor_upload' => 'ثبت سرپرست', 'admin_upload' => 'ثبت مدیریت',
			'seller_resubmit' => 'ارسال مجدد فیش', 'gateway' => 'درگاه پرداخت',
		][ $source ] ?? '';
		if ( $method_label !== '' && $source_label !== '' && $method_label !== $source_label ) { return $method_label . ' / ' . $source_label; }
		return $method_label ?: ( $source_label ?: 'ثبت سیستمی' );
	}

	private function status_label( string $status ): string {
		return [
			'active' => 'فعال', 'suspended' => 'معلق', 'waiting' => 'آماده فعال‌سازی', 'no_answer' => 'در انتظار تماس',
			'follow_up' => 'پیگیری مجدد', 'cancelled' => 'لغوشده', 'normal' => 'فعال‌سازی عادی', 'upsell' => 'فعال‌سازی افزایشی',
			'ready' => 'آماده انجام', 'awaiting_payment' => 'در انتظار پرداخت', 'retry_requested' => 'فرصت مجدد پرداخت',
			'paid' => 'پرداخت تأیید شد', 'fulfilled' => 'فعال‌شده', 'remote_completed' => 'فعال‌شده',
		][ sanitize_key( $status ) ] ?? ( $status !== '' ? $status : 'آماده فعال‌سازی' );
	}

	private function timeline( WP_User $user, array $memberships, array $invoices ): array {
		global $wpdb; $timeline = [];
		foreach ( $invoices as $invoice ) {
			$code = sanitize_text_field( (string) ( $invoice['invoice_code'] ?? '' ) );
			if ( ! empty( $invoice['created_at'] ) ) { $timeline[] = [ 'at' => (string) $invoice['created_at'], 'type' => 'invoice', 'title' => 'فاکتور صادر شد', 'text' => $code !== '' ? 'شماره فاکتور: ' . $code : 'فاکتور جدید برای شما ثبت شد.' ]; }
			if ( ! empty( $invoice['payment_completed_at'] ) ) { $timeline[] = [ 'at' => (string) $invoice['payment_completed_at'], 'type' => 'payment', 'title' => 'پرداخت نهایی تأیید شد', 'text' => SN_Helpers::format_price( (float) ( $invoice['amount'] ?? 0 ) ) ]; }
		}
		$membership_ids = array_values( array_filter( array_map( static fn( $row ) => absint( $row['id'] ?? 0 ), $memberships ) ) );
		if ( $membership_ids && $this->table_exists( $wpdb->prefix . 'sn_project_events' ) ) {
			$ids_sql = implode( ',', array_map( 'intval', $membership_ids ) );
			$events = $wpdb->get_results( "SELECT event_key,details,created_at FROM {$wpdb->prefix}sn_project_events WHERE membership_id IN ({$ids_sql}) ORDER BY id DESC LIMIT 250", ARRAY_A ) ?: [];
			$labels = [
				'membership_activated' => [ 'membership', 'اشتراک فعال شد', 'اشتراک و کارت‌های شما به پروفایل اضافه شد.' ],
				'product_star_activated' => [ 'membership', 'کارت جدید فعال شد', 'محصول خریداری‌شده به پروفایل شما اضافه شد.' ],
				'membership_reactivated' => [ 'membership', 'عضویت دوباره فعال شد', 'وضعیت مالی عضویت تأیید و دسترسی دوباره برقرار شد.' ],
				'membership_suspended_finance' => [ 'warning', 'عضویت موقتاً معلق شد', 'عضویت برای بررسی وضعیت مالی در حالت تعلیق قرار گرفت.' ],
				'customer_portal_invite_sent' => [ 'message', 'پیامک ورود ارسال شد', 'لینک ورود به پروفایل بیاوین برای شما ارسال شد.' ],
				'workflow_status_changed' => [ 'card', 'وضعیت کارت تغییر کرد', 'فرآیند کارت توسط تیم پروژه بروزرسانی شد.' ],
				'project_action_created' => [ 'card', 'درخواست کارت ثبت شد', 'فرآیند فعال‌سازی یا ارتقای کارت آغاز شد.' ],
				'project_action_paid' => [ 'payment', 'پرداخت کارت تأیید شد', 'پرداخت مربوط به اقدام کارت با موفقیت ثبت شد.' ],
				'project_action_fulfilled' => [ 'success', 'فعال‌سازی کارت تکمیل شد', 'خدمت انتخاب‌شده برای کارت شما تکمیل شد.' ],
				'project_remote_order_created' => [ 'card', 'سفارش پروژه ثبت شد', 'سفارش شما در سامانه پروژه ایجاد شد.' ],
				'project_remote_wallet_charged' => [ 'success', 'اعتبار پروژه اعمال شد', 'اعتبار کارت در سامانه پروژه ثبت شد.' ],
				'project_upsell_cancelled_continue_normal' => [ 'card', 'ادامه با حالت عادی', 'درخواست افزایشی لغو و کارت با حالت عادی ادامه یافت.' ],
				'operations_customer_requested_expert' => [ 'card', 'درخواست فعال‌سازی توسط کارشناس', 'کارت برای بررسی مدیر فروش عملیات ارسال شد.' ],
				'operations_assigned_to_sales_supervisor' => [ 'card', 'ارجاع به سرپرست عملیات', 'مدیر فروش عملیات کارت را به سرپرست مربوطه تخصیص داد.' ],
				'operations_assigned_to_sales_expert' => [ 'card', 'ارجاع به کارشناس عملیات', 'سرپرست فروش عملیات کارت را به کارشناس تخصیص داد.' ],
				'operations_follow_up_scheduled' => [ 'card', 'تماس مجدد برنامه‌ریزی شد', 'زمان تماس با شما در پرونده ثبت یا ویرایش شد.' ],
				'operations_supervisor_self_assigned' => [ 'card', 'پیگیری توسط سرپرست آغاز شد', 'سرپرست فروش عملیات پیگیری این کارت را بر عهده گرفت.' ],
				'operations_no_answer' => [ 'warning', 'پاسخی دریافت نشد', 'کارشناس تلاش ناموفق برای تماس را ثبت کرد.' ],
				'operations_customer_cancelled' => [ 'warning', 'انصراف ثبت شد', 'انصراف از ادامه فعال‌سازی توسط کارشناس ثبت شد.' ],
				'operations_upgrade_invoice_created' => [ 'invoice', 'فاکتور افزایشی صادر شد', 'لینک پرداخت افزایش اعتبار کارت ایجاد شد.' ],
				'operations_upgrade_paid_routed_to_execution' => [ 'payment', 'افزایش اعتبار تأیید شد', 'کارت افزایشی برای اجرای عملیات ارسال شد.' ],
				'operations_upgrade_payment_reversed' => [ 'warning', 'پرداخت افزایشی به بررسی برگشت', 'اعتبار افزایشی موقتاً برداشته شد و امکان پیگیری دوباره یا ادامه با حالت عادی وجود دارد.' ],
				'operations_routed_normal_to_execution' => [ 'card', 'فعال‌سازی عادی تأیید شد', 'کارت با اعتبار عادی برای اجرای عملیات ارسال شد.' ],
				'operations_auto_routed_after_inactivity' => [ 'card', 'ارجاع خودکار به فروش عملیات', 'به دلیل عدم انتخاب در مهلت تعیین‌شده، کارت وارد صف مدیر فروش عملیات شد.' ],
				'operations_execution_started' => [ 'card', 'اجرای کارت آغاز شد', 'مدیر اجرایی عملیات انجام خدمت کارت را آغاز کرد.' ],
				'operations_execution_completed' => [ 'success', 'اجرای کارت تکمیل شد', 'فرآیند اجرایی کارت با موفقیت تکمیل شد.' ],
				'operations_execution_assigned' => [ 'card', 'ارجاع به کارشناس اجرایی', 'مدیر اجرایی عملیات کارت را برای انجام خدمت تخصیص داد.' ],
				'operations_execution_no_answer' => [ 'warning', 'تماس اجرایی بی‌پاسخ بود', 'تلاش کارشناس اجرایی برای تماس در پرونده ثبت شد.' ],
				'operations_execution_follow_up' => [ 'card', 'تماس اجرایی برنامه‌ریزی شد', 'تاریخ تماس مجدد کارشناس اجرایی ثبت شد.' ],
				'operations_execution_cancelled' => [ 'warning', 'اجرای کارت لغو شد', 'انصراف از ادامه اجرای کارت در پرونده ثبت شد.' ],
				'operations_wallet_charged' => [ 'success', 'اعتبار کیف پول شارژ شد', 'اعتبار این کارت در سامانه مقصد ثبت شد.' ],
				'operations_physical_sent_to_shipping' => [ 'card', 'کارت وارد واحد ارسال شد', 'مشخصات کارت فیزیکی برای آماده‌سازی و ارسال ثبت شد.' ],
				'operations_form_completed' => [ 'card', 'فرم کارت تکمیل شد', 'اطلاعات فرم تلفنی ثبت و برای تصمیم نهایی آماده شد.' ],
				'operations_form_executed' => [ 'success', 'فرم اجرا شد', 'اجرای نهایی فرم کارت تأیید شد.' ],
				'operations_form_not_executed' => [ 'warning', 'فرم اجرا نشد', 'عدم اجرای نهایی فرم در پرونده ثبت شد.' ],
				'operations_shipping_status_changed' => [ 'card', 'وضعیت ارسال به‌روز شد', 'آخرین وضعیت ارسال کارت در پرونده شما ثبت شد.' ],
				'verified_invoice_attributed' => [ 'payment', 'خرید مرتبط تأیید شد', 'یک خرید تأییدشده به کارت شما متصل شد.' ],
			];
			foreach ( $events as $event ) {
				$key = sanitize_key( (string) ( $event['event_key'] ?? '' ) );
				if ( ! isset( $labels[ $key ] ) ) { continue; }
				[ $type, $title, $text ] = $labels[ $key ];
				$timeline[] = [ 'at' => (string) $event['created_at'], 'type' => $type, 'title' => $title, 'text' => $text ];
			}
		}
		usort( $timeline, static fn( $a, $b ) => strcmp( (string) $b['at'], (string) $a['at'] ) );
		$unique = []; $out = [];
		foreach ( $timeline as $row ) {
			$key = md5( (string) $row['at'] . '|' . (string) $row['title'] . '|' . (string) $row['text'] );
			if ( isset( $unique[ $key ] ) ) { continue; }
			$unique[ $key ] = true; $out[] = $row;
			if ( count( $out ) >= 100 ) { break; }
		}
		return $out;
	}

	private function render_login(): string {
		ob_start(); ?>
		<div class="sn-customer-portal sn-customer-login" dir="rtl" data-sn-customer-login>
			<div class="sn-customer-login-shell">
				<div class="sn-customer-login-brand"><span class="sn-customer-brand-mark">ب</span><div><strong>پروفایل بیاوین</strong><small>همه اشتراک‌ها و کارت‌های شما، یکجا</small></div></div>
				<div class="sn-customer-login-card">
					<div class="sn-customer-login-icon">✦</div>
					<h1>خوش آمدید</h1>
					<p>شماره موبایلی که هنگام خرید ثبت کرده‌اید وارد کنید.</p>
					<form data-sn-customer-phone-form>
						<label for="sn-customer-phone">شماره موبایل</label>
						<input id="sn-customer-phone" name="phone" type="tel" inputmode="numeric" autocomplete="tel" maxlength="14" placeholder="۰۹۱۲۱۲۳۴۵۶۷" required>
						<button type="submit">ارسال کد ورود</button>
					</form>
					<form data-sn-customer-otp-form hidden>
						<button type="button" class="sn-customer-back" data-sn-customer-back>تغییر شماره</button>
						<p class="sn-customer-otp-caption">کد ۶ رقمی ارسال‌شده به <strong data-sn-customer-masked></strong> را وارد کنید.</p>
						<input name="code" class="sn-customer-otp-input" type="text" inputmode="numeric" autocomplete="one-time-code" maxlength="6" placeholder="------" required>
						<button type="submit">ورود به پروفایل</button>
						<button type="button" class="sn-customer-resend" data-sn-customer-resend disabled>ارسال مجدد تا ۶۰ ثانیه</button>
					</form>
					<div class="sn-customer-login-message" data-sn-customer-message aria-live="polite"></div>
					<div class="sn-customer-login-trust"><span>ورود امن با رمز یک‌بارمصرف</span><span>اطلاعات فقط برای شما نمایش داده می‌شود</span></div>
				</div>
			</div>
		</div>
		<?php return (string) ob_get_clean();
	}

	private function render_upgrade_options( array $options ): string {
		if ( ! $options ) { return '<span class="sn-customer-card-no-upgrade">حالت افزایشی تعریف نشده</span>'; }
		$html = '<details class="sn-customer-upgrades"><summary>' . esc_html( 'مشاهده ' . number_format_i18n( count( $options ) ) . ' حالت افزایشی' ) . '</summary><div>';
		foreach ( $options as $option ) {
			$name = sanitize_text_field( (string) ( $option['name'] ?? '' ) );
			$credit = max( 0, (float) ( $option['credit'] ?? $option['target_credit'] ?? 0 ) );
			$price = max( 0, (float) ( $option['price'] ?? $option['upgrade_amount'] ?? 0 ) );
			$html .= '<article><strong>' . esc_html( $name ?: 'کارت افزایشی' ) . '</strong><span>اعتبار ' . esc_html( SN_Helpers::format_price( $credit ) ) . '</span><small>مبلغ ارتقا: ' . esc_html( SN_Helpers::format_price( $price ) ) . '</small></article>';
		}
		return $html . '</div></details>';
	}

	public function render(): string {
		if ( ! is_user_logged_in() ) { return $this->render_login(); }
		$user = wp_get_current_user();
		if ( ! $user instanceof WP_User || ! $this->is_customer_user( $user ) ) {
			return '<div class="sn-customer-portal" dir="rtl"><div class="sn-customer-access-error"><strong>این صفحه مخصوص حساب مشتری است.</strong><p>برای مشاهده اشتراک‌ها، با شماره مشتری وارد شوید.</p><a href="' . esc_url( wp_logout_url( $this->portal_url() ) ) . '">خروج و ورود با حساب مشتری</a></div></div>';
		}
		$memberships = $this->customer_memberships( $user );
		$invoices = $this->customer_invoices( $user );
		$payments = $this->customer_payments( $invoices );
		$name = sanitize_text_field( (string) ( $user->display_name ?: get_user_meta( $user->ID, 'billing_first_name', true ) ) ) ?: 'مشتری بیاوین';
		$phones = $this->user_phones( $user ); $phone = (string) ( $phones[0] ?? '' );
		[ $province, $city ] = $this->customer_location( $user, $invoices, $memberships );
		$location = implode( '، ', array_values( array_filter( [ $province, $city ] ) ) );
		$card_count = 0; $total_credit = 0.0; $active_memberships = 0;
		foreach ( $memberships as $membership ) { if ( (string) $membership['status'] === 'active' ) { $active_memberships++; } foreach ( (array) $membership['items'] as $item ) { $card_count++; $total_credit += (float) ( $item['credit'] ?? 0 ); } }
		$confirmed_payment_count = 0; $confirmed_payment_total = 0.0; $last_confirmed_payment_at = '';
		foreach ( $payments as $payment ) {
			if ( ! in_array( sanitize_key( (string) ( $payment['status'] ?? '' ) ), [ 'approved', 'paid', 'verified' ], true ) ) { continue; }
			$confirmed_payment_count++;
			$confirmed_payment_total += max( 0, (float) ( $payment['amount'] ?? 0 ) );
			if ( $last_confirmed_payment_at === '' ) { $last_confirmed_payment_at = (string) ( $payment['display_at'] ?? '' ); }
		}
		ob_start(); ?>
		<div class="sn-customer-portal" dir="rtl">
			<?php if ( ! empty( $_GET['sn_ops_notice'] ) ) : $notice_error = sanitize_key( (string) ( $_GET['sn_ops_kind'] ?? '' ) ) === 'error'; ?><div class="sn-customer-global-notice <?php echo $notice_error ? 'is-error' : ''; ?>" role="status"><b><?php echo esc_html( $notice_error ? 'ناموفق' : 'موفق' ); ?></b><button type="button" class="sn-customer-notice-close" data-sn-customer-notice-close aria-label="بستن اعلان">×</button></div><?php endif; ?>
			<header class="sn-customer-hero">
				<div class="sn-customer-hero-glow"></div>
				<nav><div class="sn-customer-logo"><span>ب</span><strong>بیاوین</strong></div><div class="sn-customer-nav-links"><a href="#sn-customer-account">حساب</a><a href="#sn-customer-cards">کارت‌ها</a><a href="#sn-customer-purchases">خریدها</a><a href="#sn-customer-payments">پرداخت‌ها</a><a class="is-logout" href="<?php echo esc_url( wp_logout_url( $this->portal_url() ) ); ?>">خروج</a></div></nav>
				<div class="sn-customer-welcome"><div><small>سلام، خوش آمدید</small><h1><?php echo esc_html( $name ); ?></h1><p>اشتراک‌ها، کارت‌ها و مسیر خدمات شما در این پروفایل نگهداری می‌شود.</p></div><div class="sn-customer-avatar"><?php echo esc_html( function_exists( 'mb_substr' ) ? mb_substr( $name, 0, 1 ) : substr( $name, 0, 1 ) ); ?></div></div>
			</header>

			<main class="sn-customer-main">
				<section class="sn-customer-static-section sn-customer-account-section" id="sn-customer-account">
					<header class="sn-customer-static-section-head"><span class="sn-customer-accordion-heading"><small>پروفایل من</small><strong>اطلاعات حساب و اعتبار</strong></span></header>
					<div class="sn-customer-static-section-body">
				<section class="sn-customer-identity" aria-label="اطلاعات مشتری">
					<div><i aria-hidden="true">م</i><span>نام و نام خانوادگی</span><strong><?php echo esc_html( $name ); ?></strong></div>
					<div><i aria-hidden="true">۰۹</i><span>شماره موبایل</span><strong dir="ltr"><?php echo esc_html( $phone ?: '—' ); ?></strong></div>
					<div><i aria-hidden="true">⌖</i><span>استان و شهر</span><strong><?php echo esc_html( $location ?: 'ثبت نشده' ); ?></strong></div>
				</section>

				<section class="sn-customer-credit-wallet" aria-label="اعتبار کارت‌های بیاوین">
					<div class="sn-customer-credit-wallet-top"><div><small>اعتبار کارت‌های بیاوین</small><strong>اعتبار قابل استفاده</strong></div></div>
					<?php if ( $total_credit > 0 ) : ?><div class="sn-customer-credit-wallet-balance"><small>مجموع اعتبار جاری</small><div><strong><?php echo esc_html( SN_Helpers::format_price( $total_credit ) ); ?></strong></div></div><?php endif; ?>
					<div class="sn-customer-credit-wallet-actions"><a href="#sn-customer-cards"><span aria-hidden="true">＋</span> مشاهده کارت‌ها</a><a href="#sn-customer-payments"><span aria-hidden="true">≡</span> گردش پرداخت‌ها</a></div>
				</section>


					</div>
				</section>

				<details class="sn-customer-accordion sn-customer-section" id="sn-customer-cards" open>
					<summary><span class="sn-customer-accordion-heading"><small>دارایی‌های من</small><strong>اشتراک‌ها و کارت‌ها</strong></span><span class="sn-customer-accordion-help">اطلاعات هر کارت مطابق شرایط زمان خرید شما ثابت نگه داشته می‌شود.</span><i class="sn-customer-accordion-icon" aria-hidden="true"></i></summary>
					<div class="sn-customer-accordion-body">
					<?php if ( ! $memberships ) : ?><div class="sn-customer-empty"><span>◇</span><strong>هنوز اشتراکی در پروفایل شما نیست</strong><p>پس از پرداخت نهایی و تأیید مالی، اشتراک و کارت‌ها اینجا نمایش داده می‌شوند.</p></div><?php endif; ?>
					<div class="sn-customer-memberships">
					<?php foreach ( $memberships as $membership ) : $membership_theme = sanitize_hex_color( (string) ( $membership['theme_color'] ?? '' ) ) ?: '#177c91'; ?>
						<article class="sn-customer-membership <?php echo (string) $membership['status'] === 'active' ? 'is-active' : 'is-suspended'; ?>" style="--sn-membership-accent:<?php echo esc_attr( $membership_theme ); ?>">
							<header><?php if ( ! empty( $membership['image'] ) ) : ?><img src="<?php echo esc_url( $membership['image'] ); ?>" alt=""><?php else : ?><div class="sn-customer-membership-placeholder">✦</div><?php endif; ?><div class="sn-customer-membership-copy"><span>اشتراک من</span><h3><?php echo esc_html( (string) $membership['subscription_name_snapshot'] ); ?></h3><small>فعال از <?php echo esc_html( SN_Helpers::gregorian_to_jalali_date( (string) $membership['activated_at'] ) ); ?></small></div><b><?php echo esc_html( $this->status_label( (string) $membership['status'] ) ); ?></b></header>
							<div class="sn-customer-cards">
							<?php foreach ( (array) $membership['items'] as $item ) :
								$item_id = absint( $item['id'] ?? 0 );
								$modal_id = 'sn-customer-card-modal-' . $item_id;
								$title_id = $modal_id . '-title';
								$operations_stage = sanitize_key( (string) ( $item['operations_stage'] ?? '' ) );
								$card_state = $operations_stage === 'customer_code_issued' ? 'کد آماده استفاده' : ( $operations_stage === 'awaiting_customer' ? 'آماده فعال‌سازی' : ( $operations_stage === 'completed' ? 'تکمیل‌شده' : 'در حال پیگیری' ) );
								$state_kind = in_array( $operations_stage, [ 'completed', 'customer_code_issued' ], true ) ? 'success' : ( in_array( $operations_stage, [ 'awaiting_customer', 'upgrade_payment' ], true ) ? 'attention' : 'progress' );
								$mode_label = (string) ( $item['operations_mode'] ?? '' ) === 'upsell' ? 'افزایشی' : ( (string) ( $item['operations_mode'] ?? '' ) === 'normal' ? 'عادی' : '' );
							?>
								<article class="sn-customer-card">
								<div class="sn-customer-card-image"><?php if ( ! empty( $item['image'] ) ) : ?><img src="<?php echo esc_url( $item['image'] ); ?>" alt="<?php echo esc_attr( (string) $item['content_name_snapshot'] ); ?>" loading="lazy"><?php else : ?><span>بدون تصویر محصول</span><?php endif; ?></div>
									<div class="sn-customer-card-content"><div class="sn-customer-card-meta"><span class="sn-customer-card-state is-<?php echo esc_attr( $state_kind ); ?>"><?php echo esc_html( $card_state ); ?></span></div><h4><?php echo esc_html( (string) $item['content_name_snapshot'] ); ?></h4><?php if ( (float) $item['credit'] > 0 ) : ?><div class="sn-customer-card-credit"><small><?php echo ! empty( $item['operations_mode'] ) ? 'اعتبار جاری کارت' : 'اعتبار پایه کارت'; ?></small><strong><?php echo esc_html( SN_Helpers::format_price( (float) $item['credit'] ) ); ?></strong></div><?php endif; ?><button type="button" class="sn-customer-card-open" data-sn-customer-card-open="<?php echo esc_attr( $modal_id ); ?>" aria-haspopup="dialog" aria-controls="<?php echo esc_attr( $modal_id ); ?>" aria-expanded="false"><span>مشاهده</span><i aria-hidden="true">←</i></button></div>
								</article>
								<div class="sn-customer-card-modal" id="<?php echo esc_attr( $modal_id ); ?>" data-sn-customer-card-modal hidden aria-hidden="true">
									<button type="button" class="sn-customer-card-modal-backdrop" data-sn-customer-card-close aria-label="بستن پنجره"></button>
									<section class="sn-customer-card-dialog" role="dialog" aria-modal="true" aria-labelledby="<?php echo esc_attr( $title_id ); ?>" tabindex="-1">
									<header><div class="sn-customer-card-dialog-heading"><div class="sn-customer-card-dialog-thumb"><?php if ( ! empty( $item['image'] ) ) : ?><img src="<?php echo esc_url( $item['image'] ); ?>" alt="<?php echo esc_attr( (string) $item['content_name_snapshot'] ); ?>"><?php else : ?><span>بدون تصویر</span><?php endif; ?></div><div><span class="sn-customer-card-state is-<?php echo esc_attr( $state_kind ); ?>"><?php echo esc_html( $card_state ); ?></span><h3 id="<?php echo esc_attr( $title_id ); ?>"><?php echo esc_html( (string) $item['content_name_snapshot'] ); ?></h3></div></div><button type="button" class="sn-customer-card-modal-close" data-sn-customer-card-close aria-label="بستن">×</button></header>
									<div class="sn-customer-card-dialog-body"><div class="sn-customer-card-dialog-section"><span>توضیحات کارت</span></div><div class="sn-customer-card-description"><?php if ( ! empty( $item['description_html'] ) ) : ?><?php echo $item['description_html']; // phpcs:ignore WordPress.Security.EscapeOutput.OutputNotEscaped ?><?php else : ?><p class="sn-customer-card-description-empty">برای این محصول توضیحاتی ثبت نشده است.</p><?php endif; ?></div><div class="sn-customer-card-dialog-section"><span>گزینه‌ها و عملیات کارت</span></div><?php if ( class_exists( 'SN_Operations_Flow' ) ) { echo SN_Operations_Flow::instance()->render_customer_card_controls( $item, $membership, $user ); } // phpcs:ignore WordPress.Security.EscapeOutput.OutputNotEscaped ?></div>
									</section>
								</div>
							<?php endforeach; ?>
							</div>
						</article>
					<?php endforeach; ?>
					</div>
					</div>
				</details>

				<details class="sn-customer-accordion sn-customer-section sn-customer-purchases-section" id="sn-customer-purchases">
					<summary><span class="sn-customer-accordion-heading"><small>خریدهای من</small><strong>سفارش‌ها و فاکتورها</strong></span><span class="sn-customer-accordion-help">محصول، مبلغ، وضعیت و تاریخ هر خرید</span><i class="sn-customer-accordion-icon" aria-hidden="true"></i></summary>
					<div class="sn-customer-accordion-body">
					<?php if ( $invoices ) : ?><div class="sn-customer-purchase-list">
						<?php foreach ( $invoices as $invoice ) : ?>
							<article><span class="sn-customer-purchase-symbol" aria-hidden="true">◇</span><div><strong><?php echo esc_html( (string) ( $invoice['purchase_title'] ?? 'خرید ثبت‌شده' ) ); ?></strong><small>فاکتور <?php echo esc_html( (string) ( $invoice['invoice_code'] ?: '#' . absint( $invoice['id'] ?? 0 ) ) ); ?> · <?php echo esc_html( SN_Helpers::gregorian_to_jalali_date( (string) ( $invoice['created_at'] ?? '' ) ) ); ?></small></div><div class="sn-customer-purchase-end"><b><?php echo esc_html( SN_Helpers::format_price( max( 0, (float) ( $invoice['amount'] ?? 0 ) ) ) ); ?></b><span><?php echo esc_html( SN_Helpers::status_label( (string) ( $invoice['status'] ?? '' ) ) ); ?></span></div></article>
						<?php endforeach; ?>
					</div><?php else : ?><div class="sn-customer-empty"><strong>هنوز خریدی در حساب شما ثبت نشده است.</strong></div><?php endif; ?>
					</div>
				</details>

				<details class="sn-customer-accordion sn-customer-section sn-customer-payments-section" id="sn-customer-payments">
					<summary><span class="sn-customer-accordion-heading"><small>امور مالی من</small><strong>سوابق پرداخت</strong></span><span class="sn-customer-accordion-help">جزئیات پرداخت‌های درگاهی و کارت‌به‌کارت</span><i class="sn-customer-accordion-icon" aria-hidden="true"></i></summary>
					<div class="sn-customer-accordion-body">
					<div class="sn-customer-payment-summary">
						<div><span>پرداخت تأییدشده</span><strong><?php echo esc_html( number_format_i18n( $confirmed_payment_count ) ); ?></strong></div>
						<div><span>مجموع مبالغ تأییدشده</span><strong><?php echo esc_html( SN_Helpers::format_price( $confirmed_payment_total ) ); ?></strong></div>
						<div><span>آخرین پرداخت تأییدشده</span><strong><?php echo esc_html( $last_confirmed_payment_at !== '' ? SN_Helpers::gregorian_to_jalali_date( $last_confirmed_payment_at ) : '—' ); ?></strong></div>
					</div>
					<?php if ( $payments ) : ?>
						<div class="sn-customer-payment-list" role="table" aria-label="سوابق پرداخت">
							<div class="sn-customer-payment-head" role="row"><span>فاکتور</span><span>دلیل پرداخت</span><span>مبلغ</span><span>روش پرداخت</span><span>وضعیت</span><span>تاریخ</span></div>
							<?php foreach ( $payments as $payment ) :
								$status = sanitize_key( (string) ( $payment['status'] ?? '' ) );
								$reference = sanitize_text_field( (string) ( $payment['ref_id'] ?? '' ) );
								$invoice_code = sanitize_text_field( (string) ( $payment['invoice_code'] ?? '' ) );
							?>
								<article class="sn-customer-payment-row is-<?php echo esc_attr( $this->payment_status_kind( $status ) ); ?>" role="row">
									<div data-label="فاکتور"><strong><?php echo esc_html( $invoice_code !== '' ? $invoice_code : '#' . absint( $payment['invoice_id'] ?? 0 ) ); ?></strong><small>مرحله <?php echo esc_html( number_format_i18n( max( 1, absint( $payment['payment_stage_no'] ?? 1 ) ) ) ); ?><?php if ( $reference !== '' ) : ?> · پیگیری: <bdi><?php echo esc_html( $reference ); ?></bdi><?php endif; ?></small></div>
									<div data-label="دلیل پرداخت" class="sn-customer-payment-reason"><strong><?php echo esc_html( (string) ( $payment['reason'] ?? 'پرداخت فاکتور' ) ); ?></strong></div>
									<div data-label="مبلغ"><strong><?php echo esc_html( SN_Helpers::format_price( max( 0, (float) ( $payment['amount'] ?? 0 ) ) ) ); ?></strong></div>
									<div data-label="روش پرداخت"><span><?php echo esc_html( $this->payment_channel_label( $payment ) ); ?></span></div>
									<div data-label="وضعیت"><span class="sn-customer-payment-status"><?php echo esc_html( $this->payment_status_label( $status ) ); ?></span></div>
									<div data-label="تاریخ"><time><?php echo esc_html( SN_Helpers::gregorian_to_jalali_date( (string) ( $payment['display_at'] ?? '' ) ) ); ?></time></div>
								</article>
							<?php endforeach; ?>
						</div>
					<?php else : ?>
						<div class="sn-customer-empty sn-customer-payment-empty"><strong>هنوز سابقه پرداختی ثبت نشده است.</strong><p>پس از ایجاد درخواست پرداخت یا ثبت فیش، جزئیات آن در این بخش نمایش داده می‌شود.</p></div>
					<?php endif; ?>
					</div>
				</details>

				<div class="sn-customer-profile-logout"><a href="<?php echo esc_url( wp_logout_url( $this->portal_url() ) ); ?>"><span aria-hidden="true">←</span> خروج از حساب کاربری</a></div>
			</main>
		</div>
		<?php return (string) ob_get_clean();
	}
}
