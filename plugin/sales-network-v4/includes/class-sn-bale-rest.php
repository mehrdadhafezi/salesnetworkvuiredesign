<?php
if ( ! defined( 'ABSPATH' ) ) { exit; }

/** CRM endpoints consumed by the external Bale Gateway. */
class SN_Bale_REST {
	private static ?self $instance = null;
	public static function instance(): self { return self::$instance ??= new self(); }
	private function __construct() {}

	public function register_hooks(): void { add_action( 'rest_api_init', [ $this, 'register_routes' ] ); }

	public function register_routes(): void {
		register_rest_route( 'sn/v1', '/bale/link', [ 'methods' => 'POST', 'callback' => [ $this, 'link_account' ], 'permission_callback' => '__return_true' ] );
		register_rest_route( 'sn/v1', '/bale/status', [ 'methods' => 'POST', 'callback' => [ $this, 'status' ], 'permission_callback' => '__return_true' ] );
	}

	private function secret(): string {
		return defined( 'SN_BALE_GATEWAY_SHARED_SECRET' )
			? trim( (string) constant( 'SN_BALE_GATEWAY_SHARED_SECRET' ) )
			: trim( (string) get_option( 'sn_bale_gateway_shared_secret', '' ) );
	}

	private function authorize( WP_REST_Request $request, string $path ) {
		$secret = $this->secret();
		if ( strlen( $secret ) < 32 ) { return new WP_Error( 'bale_not_configured', 'Bale Gateway secret is not configured.', [ 'status' => 503 ] ); }
		$timestamp = trim( (string) $request->get_header( 'x-sn-timestamp' ) );
		$nonce = trim( (string) $request->get_header( 'x-sn-nonce' ) );
		$signature = strtolower( trim( (string) $request->get_header( 'x-sn-signature' ) ) );
		if ( $timestamp === '' || $nonce === '' || $signature === '' || ! ctype_digit( $timestamp ) || abs( time() - (int) $timestamp ) > 300 ) {
			return new WP_Error( 'bale_unauthorized', 'Unauthorized.', [ 'status' => 401 ] );
		}
		$raw = (string) $request->get_body();
		$canonical = $timestamp . "\n" . $nonce . "\nPOST\n" . $path . "\n" . hash( 'sha256', $raw );
		$expected = hash_hmac( 'sha256', $canonical, $secret );
		if ( ! hash_equals( $expected, $signature ) ) { return new WP_Error( 'bale_unauthorized', 'Unauthorized.', [ 'status' => 401 ] ); }
		$key = 'sn_bale_nonce_' . hash( 'sha256', $nonce );
		if ( get_transient( $key ) ) { return new WP_Error( 'bale_replay', 'Duplicate request.', [ 'status' => 409 ] ); }
		set_transient( $key, '1', 10 * MINUTE_IN_SECONDS );
		return true;
	}

	private function ensure_table(): void {
		global $wpdb; $table = $wpdb->prefix . 'sn_bale_accounts';
		if ( $wpdb->get_var( $wpdb->prepare( 'SHOW TABLES LIKE %s', $table ) ) === $table ) { return; }
		require_once ABSPATH . 'wp-admin/includes/upgrade.php'; $charset = $wpdb->get_charset_collate();
		dbDelta( "CREATE TABLE {$table} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			bale_user_id BIGINT UNSIGNED NOT NULL,
			chat_id BIGINT NOT NULL,
			phone VARCHAR(20) DEFAULT NULL,
			wp_user_id BIGINT UNSIGNED DEFAULT NULL,
			first_name VARCHAR(191) DEFAULT NULL,
			last_name VARCHAR(191) DEFAULT NULL,
			username VARCHAR(191) DEFAULT NULL,
			status VARCHAR(30) NOT NULL DEFAULT 'verified',
			verified_at DATETIME DEFAULT NULL,
			last_interaction_at DATETIME DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id), UNIQUE KEY bale_user_id (bale_user_id), KEY phone (phone), KEY wp_user_id (wp_user_id), KEY status (status)
		) {$charset};" );
	}

	private function phone_variants( string $phone ): array {
		$phone = SN_Helpers::normalize_mobile( $phone );
		if ( ! SN_Helpers::is_valid_mobile( $phone ) ) { return []; }
		$n = substr( $phone, 1 ); return array_values( array_unique( [ $phone, $n, '98' . $n, '+98' . $n, '0098' . $n ] ) );
	}

	private function resolve_customer( string $phone ): array {
		global $wpdb; $variants = $this->phone_variants( $phone );
		if ( ! $variants ) { return [ 'user_id' => 0, 'display_name' => '', 'invoice' => null ]; }
		$invoice = null; $inv_table = $wpdb->prefix . 'sn_invoices';
		if ( $wpdb->get_var( $wpdb->prepare( 'SHOW TABLES LIKE %s', $inv_table ) ) === $inv_table ) {
			$ph = implode( ',', array_fill( 0, count( $variants ), '%s' ) );
			$invoice = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$inv_table} WHERE customer_phone IN ({$ph}) ORDER BY id DESC LIMIT 1", ...$variants ) );
		}
		$user_id = $invoice ? absint( $invoice->customer_wp_id ?? 0 ) : 0;
		$user = $user_id ? get_user_by( 'id', $user_id ) : false;
		if ( ! $user instanceof WP_User ) {
			$meta_keys = [ 'billing_phone', 'digits_phone', 'digits_phone_no', 'sn_customer_phone', 'mobile', 'phone' ];
			$ph = implode( ',', array_fill( 0, count( $variants ), '%s' ) ); $kh = implode( ',', array_fill( 0, count( $meta_keys ), '%s' ) );
			$ids = $wpdb->get_col( $wpdb->prepare(
				"SELECT ID FROM {$wpdb->users} WHERE user_login IN ({$ph}) UNION SELECT DISTINCT user_id ID FROM {$wpdb->usermeta} WHERE meta_key IN ({$kh}) AND meta_value IN ({$ph})",
				...array_merge( $variants, $meta_keys, $variants )
			) ) ?: [];
			$matches = [];
			foreach ( array_unique( array_map( 'absint', $ids ) ) as $id ) {
				$u = get_user_by( 'id', $id ); if ( ! $u instanceof WP_User ) { continue; }
				if ( in_array( 'customer', (array) $u->roles, true ) ) { $matches[ $id ] = $u; }
			}
			if ( count( $matches ) === 1 ) { $user = reset( $matches ); $user_id = (int) $user->ID; }
		}
		$name = $user instanceof WP_User ? (string) $user->display_name : (string) ( $invoice->customer_name ?? '' );
		return [ 'user_id' => $user_id, 'display_name' => sanitize_text_field( $name ), 'invoice' => $invoice ];
	}

	public function link_account( WP_REST_Request $request ) {
		$auth = $this->authorize( $request, '/wp-json/sn/v1/bale/link' ); if ( is_wp_error( $auth ) ) { return $auth; }
		$data = $request->get_json_params(); $data = is_array( $data ) ? $data : [];
		$bale_user_id = preg_replace( '/\D+/', '', (string) ( $data['bale_user_id'] ?? '' ) );
		$chat_id = preg_replace( '/[^0-9-]+/', '', (string) ( $data['chat_id'] ?? '' ) );
		$phone = SN_Helpers::normalize_mobile( sanitize_text_field( (string) ( $data['phone'] ?? '' ) ) );
		if ( $bale_user_id === '' || $chat_id === '' || ! SN_Helpers::is_valid_mobile( $phone ) ) { return new WP_REST_Response( [ 'success' => false, 'message' => 'اطلاعات اتصال بله نامعتبر است.' ], 400 ); }
		$customer = $this->resolve_customer( $phone );
		if ( empty( $customer['invoice'] ) && empty( $customer['user_id'] ) ) { return new WP_REST_Response( [ 'success' => false, 'message' => 'برای این شماره پرونده‌ای در CRM پیدا نشد.' ], 404 ); }
		$this->ensure_table(); global $wpdb; $table = $wpdb->prefix . 'sn_bale_accounts'; $now = current_time( 'mysql' );
		$existing = $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$table} WHERE bale_user_id=%s LIMIT 1", $bale_user_id ) );
		$row = [
			'bale_user_id' => $bale_user_id, 'chat_id' => $chat_id, 'phone' => $phone,
			'wp_user_id' => ! empty( $customer['user_id'] ) ? (int) $customer['user_id'] : null,
			'first_name' => sanitize_text_field( (string) ( $data['first_name'] ?? '' ) ), 'last_name' => sanitize_text_field( (string) ( $data['last_name'] ?? '' ) ),
			'username' => sanitize_text_field( (string) ( $data['username'] ?? '' ) ), 'status' => 'verified', 'verified_at' => $now, 'last_interaction_at' => $now, 'updated_at' => $now,
		];
		if ( $existing ) { $wpdb->update( $table, $row, [ 'id' => (int) $existing ] ); }
		else { $row['created_at'] = $now; $wpdb->insert( $table, $row ); }
		return new WP_REST_Response( [ 'success' => true, 'data' => [ 'customer_id' => (int) ( $customer['user_id'] ?? 0 ), 'display_name' => (string) ( $customer['display_name'] ?: 'مشتری' ) ] ], 200 );
	}

	public function status( WP_REST_Request $request ) {
		$auth = $this->authorize( $request, '/wp-json/sn/v1/bale/status' ); if ( is_wp_error( $auth ) ) { return $auth; }
		$data = $request->get_json_params(); $data = is_array( $data ) ? $data : [];
		$bale_user_id = preg_replace( '/\D+/', '', (string) ( $data['bale_user_id'] ?? '' ) ); $chat_id = preg_replace( '/[^0-9-]+/', '', (string) ( $data['chat_id'] ?? '' ) );
		$this->ensure_table(); global $wpdb; $table = $wpdb->prefix . 'sn_bale_accounts';
		$account = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$table} WHERE bale_user_id=%s AND chat_id=%s AND status='verified' LIMIT 1", $bale_user_id, $chat_id ) );
		if ( ! $account ) { return new WP_REST_Response( [ 'success' => false, 'message' => 'حساب بله هنوز به CRM متصل نشده است. ابتدا /start را اجرا کنید.' ], 404 ); }
		$wpdb->update( $table, [ 'last_interaction_at' => current_time( 'mysql' ), 'updated_at' => current_time( 'mysql' ) ], [ 'id' => (int) $account->id ] );
		$variants = $this->phone_variants( (string) $account->phone ); $inv_table = $wpdb->prefix . 'sn_invoices'; $invoice = null;
		if ( $wpdb->get_var( $wpdb->prepare( 'SHOW TABLES LIKE %s', $inv_table ) ) === $inv_table ) {
			$where = []; $args = [];
			if ( (int) $account->wp_user_id > 0 ) { $where[] = 'customer_wp_id=%d'; $args[] = (int) $account->wp_user_id; }
			if ( $variants ) { $where[] = 'customer_phone IN (' . implode( ',', array_fill( 0, count( $variants ), '%s' ) ) . ')'; $args = array_merge( $args, $variants ); }
			if ( $where ) { $invoice = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$inv_table} WHERE (" . implode( ' OR ', $where ) . ') ORDER BY id DESC LIMIT 1', ...$args ) ); }
		}
		if ( ! $invoice ) { return new WP_REST_Response( [ 'success' => true, 'data' => [ 'text' => 'حساب بله شما با CRM متصل است؛ هنوز فاکتور یا پرونده فعالی برای نمایش پیدا نشد.', 'buttons' => [] ] ], 200 ); }
		$status = (string) ( $invoice->status ?? $invoice->invoice_status ?? $invoice->payment_status ?? '' );
		$labels = [ 'pre_invoice'=>'پیش‌فاکتور / در انتظار پرداخت','pending'=>'در انتظار پرداخت','pending_payment'=>'در انتظار پرداخت','partial_paid'=>'پرداخت مرحله‌ای','receipt_uploaded'=>'فیش ثبت شده و در انتظار بررسی','pending_financial_approval'=>'در انتظار تایید مالی','paid'=>'پرداخت و تایید شده','approved'=>'تایید شده','rejected'=>'رد شده','financial_rejected'=>'رد مالی','cancelled'=>'لغو شده' ];
		$label = $labels[ $status ] ?? ( class_exists( 'SN_Helpers' ) && method_exists( 'SN_Helpers', 'status_label' ) ? SN_Helpers::status_label( $status ) : $status );
		$text = 'آخرین وضعیت شما' . ( ! empty( $invoice->customer_name ) ? '، ' . sanitize_text_field( (string) $invoice->customer_name ) : '' ) . ":\nفاکتور " . sanitize_text_field( (string) $invoice->invoice_code ) . "\nوضعیت: " . $label;
		$remaining = isset( $invoice->remaining_amount ) ? (float) $invoice->remaining_amount : 0; if ( $remaining > 0.5 ) { $text .= "\nمانده: " . number_format_i18n( $remaining ) . ' تومان'; }
		$url = '';
		if ( ! empty( $invoice->short_code ) ) { $url = home_url( '/i/' . rawurlencode( (string) $invoice->short_code ) . '/' ); }
		elseif ( ! empty( $invoice->access_token ) ) { $page = absint( get_option( 'sn_invoice_page_id', 0 ) ); $base = $page ? (string) get_permalink( $page ) : home_url( '/invoice/' ); $url = add_query_arg( [ 'invoice' => (string) $invoice->invoice_code, 'invoice_code' => (string) $invoice->invoice_code, 'access_token' => (string) $invoice->access_token ], $base ); }
		$buttons = $url !== '' ? [ [ [ 'text' => 'مشاهده پرونده / فاکتور', 'url' => esc_url_raw( $url ) ] ] ] : [];
		return new WP_REST_Response( [ 'success' => true, 'data' => [ 'text' => $text, 'buttons' => $buttons ] ], 200 );
	}
}
