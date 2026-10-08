<?php
if ( ! defined( 'ABSPATH' ) ) { exit; }

/**
 * مدیریت فاکتور و درگاه‌های پرداخت
 */
class SN_Invoice {

	private bool   $zarinpal_sandbox;
	private string $zarinpal_merchant_id;
	private string $zibal_merchant;
	private bool   $zibal_test_mode;
	private int    $asan_config_id;
	private string $asan_username;
	private string $asan_password;
	private string $asan_key;
	private string $asan_iv;
	private bool   $asan_reconcile;
	private string $active_gateway;

	public function __construct() {
		$this->zarinpal_sandbox     = (bool) get_option( 'sn_zarinpal_sandbox', '1' );
		$this->zarinpal_merchant_id = (string) get_option( 'sn_zarinpal_merchant', '' );
		$this->zibal_merchant       = (string) get_option( 'sn_zibal_merchant', '' );
		$this->zibal_test_mode      = (bool) get_option( 'sn_zibal_test_mode', '1' );
		$this->asan_config_id       = absint( get_option( 'sn_asan_config_id', 0 ) );
		$this->asan_username        = trim( (string) get_option( 'sn_asan_username', '' ) );
		$this->asan_password        = trim( (string) get_option( 'sn_asan_password', '' ) );
		$this->asan_key             = trim( (string) get_option( 'sn_asan_key', '' ) );
		$this->asan_iv              = trim( (string) get_option( 'sn_asan_iv', '' ) );
		$this->asan_reconcile       = (bool) get_option( 'sn_asan_reconcile', '1' );
		$this->active_gateway       = sanitize_key( (string) get_option( 'sn_payment_gateway', 'zarinpal' ) );
		if ( ! in_array( $this->active_gateway, [ 'zarinpal', 'zibal', 'asanpardakht' ], true ) ) {
			$this->active_gateway = 'zarinpal';
		}
	}

	public function active_gateway(): string {
		return $this->active_gateway;
	}

	/** دریافت فاکتور با ID */
	public function get( int $id ): ?object {
		global $wpdb;
		return $wpdb->get_row( $wpdb->prepare(
			"SELECT * FROM {$wpdb->prefix}sn_invoices WHERE id = %d", $id
		) );
	}

	/** بروزرسانی وضعیت فاکتور */
	public function update_status( int $id, string $status, array $extra = [] ): void {
		global $wpdb;
		$data = array_merge( [ 'status' => $status ], $extra );
		$wpdb->update( $wpdb->prefix . 'sn_invoices', $data, [ 'id' => $id ] );
	}

	/** ایجاد درخواست پرداخت برای درگاه فعال؛ درخواست pending همان مرحله تا ۳۰ دقیقه دوباره استفاده می‌شود. */
	public function request_payment( int $invoice_id, float $amount, string $description, string $mobile, string $callback_url, string $order_id = '', int $payment_stage_no = 1 ): array {
		global $wpdb;
		$payment_stage_no = max( 1, $payment_stage_no );
		$lock_name = 'sn_pay_' . md5( $invoice_id . '|' . $payment_stage_no );
		$got_lock = (int) $wpdb->get_var( $wpdb->prepare( 'SELECT GET_LOCK(%s, %d)', $lock_name, 8 ) ) === 1;
		if ( ! $got_lock ) {
			return [ 'error' => 'درخواست پرداخت دیگری برای همین مرحله در حال پردازش است؛ چند ثانیه بعد دوباره تلاش کنید.' ];
		}
		try {
			$reused = $this->reusable_pending_request( $invoice_id, $amount, $payment_stage_no );
			if ( $reused ) {
				$reused['reused'] = true;
				return $reused;
			}
			$wpdb->query( $wpdb->prepare(
				"UPDATE {$wpdb->prefix}sn_payments SET status='expired', updated_at=%s WHERE invoice_id=%d AND payment_stage_no=%d AND status='pending' AND authority IS NOT NULL",
				current_time( 'mysql' ), $invoice_id, $payment_stage_no
			) );
			if ( $this->active_gateway === 'zibal' ) {
				return $this->zibal_request( $invoice_id, $amount, $description, $mobile, $callback_url, $order_id, $payment_stage_no );
			}
			if ( $this->active_gateway === 'asanpardakht' ) {
				return $this->asanpardakht_request( $invoice_id, $amount, $description, $mobile, $callback_url, $order_id, $payment_stage_no );
			}
			return $this->zarinpal_request( $invoice_id, $amount, $description, $mobile, $callback_url, $payment_stage_no );
		} finally {
			$wpdb->get_var( $wpdb->prepare( 'SELECT RELEASE_LOCK(%s)', $lock_name ) );
		}
	}

	private function reusable_pending_request( int $invoice_id, float $amount, int $payment_stage_no ): ?array {
		global $wpdb;
		$row = $wpdb->get_row( $wpdb->prepare(
			"SELECT * FROM {$wpdb->prefix}sn_payments
			 WHERE invoice_id=%d AND payment_stage_no=%d AND status='pending'
			   AND authority IS NOT NULL AND ABS(amount-%f)<0.5
			   AND created_at >= DATE_SUB(%s, INTERVAL 30 MINUTE)
			 ORDER BY id DESC LIMIT 1",
			$invoice_id, $payment_stage_no, $amount, current_time( 'mysql' )
		) );
		if ( ! $row ) { return null; }
		$authority = (string) ( $row->authority ?? '' );
		if ( $this->active_gateway === 'zibal' && strpos( $authority, 'zibal:' ) === 0 ) {
			$track_id = substr( $authority, 6 );
			return [ 'url' => 'https://gateway.zibal.ir/start/' . rawurlencode( $track_id ), 'authority' => $authority, 'gateway' => 'zibal' ];
		}
		if ( $this->active_gateway === 'asanpardakht' && strpos( $authority, 'asan:' ) === 0 && ! empty( $row->ref_id ) ) {
			$base = get_permalink( (int) get_option( 'sn_invoice_page_id' ) ) ?: home_url( '/' );
			return [ 'url' => add_query_arg( [ 'sn_asan_redirect' => '1', 'ref_id' => (string) $row->ref_id ], $base ), 'authority' => $authority, 'gateway' => 'asanpardakht' ];
		}
		if ( $this->active_gateway === 'zarinpal' && $authority !== '' && strpos( $authority, ':' ) === false ) {
			$url = $this->zarinpal_sandbox ? "https://sandbox.zarinpal.com/pg/StartPay/{$authority}" : "https://www.zarinpal.com/pg/StartPay/{$authority}";
			return [ 'url' => $url, 'authority' => $authority, 'gateway' => 'zarinpal' ];
		}
		return null;
	}

	/** ایجاد درخواست پرداخت زرین‌پال - برگرداندن URL */
	public function zarinpal_request( int $invoice_id, float $amount, string $description, string $mobile, string $callback_url, int $payment_stage_no = 1 ): array {
		if ( $this->zarinpal_merchant_id === '' ) {
			return [ 'error' => 'Merchant ID زرین‌پال تنظیم نشده است' ];
		}

		$api_url = $this->zarinpal_sandbox
			? 'https://sandbox.zarinpal.com/pg/v4/payment/request.json'
			: 'https://api.zarinpal.com/pg/v4/payment/request.json';

		$res = wp_remote_post( $api_url, [
			'timeout' => 20,
			'headers' => [ 'Content-Type' => 'application/json', 'Accept' => 'application/json' ],
			'body'    => wp_json_encode( [
				'merchant_id'  => $this->zarinpal_merchant_id,
				'amount'       => (int) round( $amount * 10 ), // تومان به ریال
				'description'  => $description,
				'callback_url' => $callback_url,
				'metadata'     => [ 'mobile' => $mobile ],
			] ),
		] );

		if ( is_wp_error( $res ) ) {
			return [ 'error' => $res->get_error_message() ];
		}

		$body = json_decode( wp_remote_retrieve_body( $res ), true );
		if ( empty( $body['data']['authority'] ) ) {
			return [ 'error' => 'خطا در ارتباط با درگاه زرین‌پال', 'raw' => $body ];
		}

		$authority = (string) $body['data']['authority'];

		// ذخیره در جدول پرداخت‌ها
		global $wpdb;
		$inserted = $wpdb->insert( $wpdb->prefix . 'sn_payments', [
			'invoice_id' => $invoice_id,
			'payment_stage_no' => max( 1, $payment_stage_no ),
			'authority'  => $authority,
			'amount'     => $amount,
			'status'     => 'pending',
			'pay_method' => 'online',
			'payment_source' => 'zarinpal',
			'created_at' => current_time( 'mysql' ),
			'updated_at' => current_time( 'mysql' ),
		] );
		if ( false === $inserted ) {
			return [ 'error' => 'ثبت درخواست پرداخت در دیتابیس انجام نشد: ' . ( $wpdb->last_error ?: 'خطای دیتابیس' ) ];
		}

		$gateway_url = $this->zarinpal_sandbox
			? "https://sandbox.zarinpal.com/pg/StartPay/{$authority}"
			: "https://www.zarinpal.com/pg/StartPay/{$authority}";

		return [ 'url' => $gateway_url, 'authority' => $authority, 'gateway' => 'zarinpal' ];
	}

	/** تایید پرداخت زرین‌پال */
	public function zarinpal_verify( string $authority, float $amount ): array {
		$api_url = $this->zarinpal_sandbox
			? 'https://sandbox.zarinpal.com/pg/v4/payment/verify.json'
			: 'https://api.zarinpal.com/pg/v4/payment/verify.json';

		$res = wp_remote_post( $api_url, [
			'timeout' => 20,
			'headers' => [ 'Content-Type' => 'application/json', 'Accept' => 'application/json' ],
			'body'    => wp_json_encode( [
				'merchant_id' => $this->zarinpal_merchant_id,
				'amount'      => (int) round( $amount * 10 ),
				'authority'   => $authority,
			] ),
		] );

		if ( is_wp_error( $res ) ) {
			return [ 'success' => false, 'error' => $res->get_error_message() ];
		}

		$body = json_decode( wp_remote_retrieve_body( $res ), true );
		$code = $body['data']['code'] ?? -1;

		if ( in_array( $code, [ 100, 101 ], true ) ) {
			$ref_id = $body['data']['ref_id'] ?? '';
			return [ 'success' => true, 'ref_id' => $ref_id, 'already_paid' => $code === 101, 'raw' => $body ];
		}

		return [ 'success' => false, 'code' => $code, 'raw' => $body ];
	}

	/** ایجاد درخواست پرداخت زیبال - مبلغ ورودی تومان است و برای زیبال به ریال تبدیل می‌شود. */
	public function zibal_request( int $invoice_id, float $amount, string $description, string $mobile, string $callback_url, string $order_id = '', int $payment_stage_no = 1 ): array {
		$merchant = $this->zibal_test_mode ? 'zibal' : trim( $this->zibal_merchant );
		if ( $merchant === '' ) {
			return [ 'error' => 'Merchant زیبال تنظیم نشده است' ];
		}

		$payload = [
			'merchant'    => $merchant,
			'amount'      => (int) round( $amount * 10 ), // تومان به ریال
			'callbackUrl' => $callback_url,
			'description' => $description,
		];
		if ( $order_id !== '' ) {
			$payload['orderId'] = $order_id;
		}
		if ( $mobile !== '' ) {
			$payload['mobile'] = $mobile;
		}

		$res = wp_remote_post( 'https://gateway.zibal.ir/v1/request', [
			'timeout' => 20,
			'headers' => [ 'Content-Type' => 'application/json', 'Accept' => 'application/json' ],
			'body'    => wp_json_encode( $payload ),
		] );

		if ( is_wp_error( $res ) ) {
			return [ 'error' => $res->get_error_message() ];
		}

		$body   = json_decode( wp_remote_retrieve_body( $res ), true );
		$result = (int) ( $body['result'] ?? 0 );
		if ( $result !== 100 || empty( $body['trackId'] ) ) {
			$message = (string) ( $body['message'] ?? 'خطا در ارتباط با درگاه زیبال' );
			return [ 'error' => 'زیبال: ' . $message, 'code' => $result, 'raw' => $body ];
		}

		$track_id  = (string) $body['trackId'];
		$authority = 'zibal:' . $track_id;

		global $wpdb;
		$inserted = $wpdb->insert( $wpdb->prefix . 'sn_payments', [
			'invoice_id' => $invoice_id,
			'payment_stage_no' => max( 1, $payment_stage_no ),
			'authority'  => $authority,
			'amount'     => $amount,
			'status'     => 'pending',
			'pay_method' => 'online',
			'payment_source' => 'zibal',
			'created_at' => current_time( 'mysql' ),
			'updated_at' => current_time( 'mysql' ),
		] );
		if ( false === $inserted ) {
			return [ 'error' => 'ثبت درخواست پرداخت در دیتابیس انجام نشد: ' . ( $wpdb->last_error ?: 'خطای دیتابیس' ) ];
		}

		return [
			'url'       => 'https://gateway.zibal.ir/start/' . rawurlencode( $track_id ),
			'authority' => $authority,
			'track_id'  => $track_id,
			'gateway'   => 'zibal',
			'raw'       => $body,
		];
	}

	/** تایید پرداخت زیبال - مبلغ ورودی تومان است. */
	public function zibal_verify( string $track_id, float $amount ): array {
		$merchant = $this->zibal_test_mode ? 'zibal' : trim( $this->zibal_merchant );
		if ( $merchant === '' ) {
			return [ 'success' => false, 'error' => 'Merchant زیبال تنظیم نشده است' ];
		}

		$res = wp_remote_post( 'https://gateway.zibal.ir/v1/verify', [
			'timeout' => 20,
			'headers' => [ 'Content-Type' => 'application/json', 'Accept' => 'application/json' ],
			'body'    => wp_json_encode( [
				'merchant' => $merchant,
				'trackId'  => (int) $track_id,
			] ),
		] );

		if ( is_wp_error( $res ) ) {
			return [ 'success' => false, 'error' => $res->get_error_message() ];
		}

		$body   = json_decode( wp_remote_retrieve_body( $res ), true );
		$result = (int) ( $body['result'] ?? 0 );
		$status = isset( $body['status'] ) ? (int) $body['status'] : 0;
		$paid_amount = isset( $body['amount'] ) ? (int) $body['amount'] : 0;
		$expected_amount = (int) round( $amount * 10 );

		if ( in_array( $result, [ 100, 201 ], true ) && ( $status === 1 || $result === 201 ) ) {
			if ( $paid_amount > 0 && $expected_amount > 0 && $paid_amount !== $expected_amount ) {
				return [ 'success' => false, 'code' => 'amount_mismatch', 'raw' => $body ];
			}
			$ref_id = (string) ( $body['refNumber'] ?? $track_id );
			return [ 'success' => true, 'ref_id' => $ref_id, 'already_paid' => $result === 201, 'raw' => $body ];
		}

		return [ 'success' => false, 'code' => $result, 'status' => $status, 'raw' => $body ];
	}

	/** ایجاد درخواست پرداخت آسان پرداخت - مبلغ ورودی تومان است و طبق مستند آسان پرداخت به ریال ارسال می‌شود. */
	public function asanpardakht_request( int $invoice_id, float $amount, string $description, string $mobile, string $callback_url, string $order_id = '', int $payment_stage_no = 1 ): array {
		$ready = $this->asanpardakht_requirements_ok();
		if ( $ready !== true ) {
			return [ 'error' => $ready ];
		}

		$amount_rial = (int) round( $amount * 10 );
		if ( $amount_rial <= 0 ) {
			return [ 'error' => 'مبلغ پرداخت برای آسان پرداخت نامعتبر است' ];
		}

		$local_invoice_id = $invoice_id;
		$merchant_date    = current_time( 'Ymd His' );
		$additional_raw   = $description !== '' ? $description : ( 'invoice:' . ( $order_id ?: $invoice_id ) );
		$additional_data  = function_exists( 'mb_substr' ) ? mb_substr( $additional_raw, 0, 100 ) : substr( $additional_raw, 0, 100 );
		$payment_id       = '0';
		$request_string   = implode( ',', [
			'1',
			$this->asan_username,
			$this->asan_password,
			(string) $local_invoice_id,
			(string) $amount_rial,
			$merchant_date,
			$additional_data,
			$callback_url,
			$payment_id,
		] );

		$encrypted_request = $this->asanpardakht_encrypt( $request_string );
		if ( $encrypted_request === '' ) {
			return [ 'error' => 'رمزنگاری درخواست آسان پرداخت انجام نشد. کلید/IV یا افزونه mcrypt را بررسی کنید.' ];
		}

		$soap = $this->asanpardakht_soap_call( 'RequestOperation', [
			'merchantConfigurationID' => $this->asan_config_id,
			'encryptedRequest'        => $encrypted_request,
		] );
		if ( ! empty( $soap['error'] ) ) {
			return [ 'error' => 'آسان پرداخت: ' . $soap['error'] ];
		}

		$response = trim( (string) ( $soap['result'] ?? '' ) );
		$parts    = array_map( 'trim', explode( ',', $response, 2 ) );
		if ( count( $parts ) < 2 || $parts[0] !== '0' || $parts[1] === '' ) {
			return [ 'error' => 'آسان پرداخت: خطا در ایجاد توکن پرداخت' . ( $response !== '' ? ' - کد: ' . $response : '' ), 'raw' => $response ];
		}

		$ref_id    = $parts[1];
		$authority = 'asan:' . sha1( $ref_id );

		global $wpdb;
		$inserted = $wpdb->insert( $wpdb->prefix . 'sn_payments', [
			'invoice_id' => $invoice_id,
			'payment_stage_no' => max( 1, $payment_stage_no ),
			'authority'  => $authority,
			'ref_id'     => $ref_id,
			'amount'     => $amount,
			'status'     => 'pending',
			'pay_method' => 'online',
			'payment_source' => 'asanpardakht',
			'created_at' => current_time( 'mysql' ),
			'updated_at' => current_time( 'mysql' ),
		] );
		if ( false === $inserted ) {
			return [ 'error' => 'ثبت درخواست پرداخت در دیتابیس انجام نشد: ' . ( $wpdb->last_error ?: 'خطای دیتابیس' ) ];
		}

		$redirect_base = get_permalink( (int) get_option( 'sn_invoice_page_id' ) ) ?: home_url( '/' );
		$redirect_url  = add_query_arg( [
			'sn_asan_redirect' => '1',
			'ref_id'           => $ref_id,
		], $redirect_base );

		return [
			'url'       => $redirect_url,
			'authority' => $authority,
			'ref_id'    => $ref_id,
			'gateway'   => 'asanpardakht',
			'raw'       => $response,
		];
	}

	/** تایید پرداخت آسان پرداخت. amount بر حسب تومان است. */
	public function asanpardakht_verify( int $invoice_id, float $amount, string $returning_params ): array {
		$ready = $this->asanpardakht_requirements_ok();
		if ( $ready !== true ) {
			return [ 'success' => false, 'error' => $ready ];
		}

		$returning_params = trim( $returning_params );
		$plain = $this->asanpardakht_decrypt( $returning_params );
		if ( $plain === '' && strpos( $returning_params, ',' ) !== false ) {
			$plain = $returning_params;
		}
		if ( $plain === '' ) {
			return [ 'success' => false, 'error' => 'رمزگشایی پاسخ آسان پرداخت انجام نشد' ];
		}

		$parts = array_map( 'trim', explode( ',', $plain ) );
		if ( count( $parts ) < 8 ) {
			return [ 'success' => false, 'error' => 'ساختار پاسخ آسان پرداخت نامعتبر است', 'raw' => $plain ];
		}

		[$paid_amount_rial, $sale_order_id, $ref_id, $res_code, $message_text, $pay_gate_tran_id, $rrn, $pan_last4] = array_slice( $parts, 0, 8 );
		$expected_amount_rial = (int) round( $amount * 10 );
		if ( (int) $sale_order_id !== $invoice_id ) {
			return [ 'success' => false, 'code' => 'invoice_mismatch', 'raw' => $plain ];
		}
		if ( $expected_amount_rial > 0 && (int) $paid_amount_rial !== $expected_amount_rial ) {
			return [ 'success' => false, 'code' => 'amount_mismatch', 'raw' => $plain ];
		}
		if ( ! in_array( (string) $res_code, [ '0', '00' ], true ) ) {
			return [ 'success' => false, 'code' => $res_code, 'message' => $message_text, 'raw' => $plain ];
		}

		$credentials = $this->asanpardakht_encrypt( $this->asan_username . ',' . $this->asan_password );
		if ( $credentials === '' ) {
			return [ 'success' => false, 'error' => 'رمزنگاری اعتبارنامه آسان پرداخت انجام نشد' ];
		}

		$verify = $this->asanpardakht_soap_call( 'RequestVerification', [
			'merchantConfigurationID' => $this->asan_config_id,
			'encryptedCredentials'    => $credentials,
			'payGateTranID'           => (int) $pay_gate_tran_id,
		] );
		if ( ! empty( $verify['error'] ) ) {
			return [ 'success' => false, 'error' => 'آسان پرداخت Verify: ' . $verify['error'], 'raw' => $plain ];
		}
		$verify_code = (int) ( $verify['result'] ?? 0 );
		if ( ! in_array( $verify_code, [ 500, 504, 505, 507 ], true ) ) {
			return [ 'success' => false, 'code' => $verify_code, 'stage' => 'verify', 'raw' => $plain ];
		}

		$reconcile_code = null;
		if ( $this->asan_reconcile ) {
			$reconcile = $this->asanpardakht_soap_call( 'RequestReconciliation', [
				'merchantConfigurationID' => $this->asan_config_id,
				'encryptedCredentials'    => $credentials,
				'payGateTranID'           => (int) $pay_gate_tran_id,
			] );
			if ( ! empty( $reconcile['error'] ) ) {
				return [ 'success' => false, 'error' => 'آسان پرداخت Reconciliation: ' . $reconcile['error'], 'raw' => $plain ];
			}
			$reconcile_code = (int) ( $reconcile['result'] ?? 0 );
			if ( ! in_array( $reconcile_code, [ 600, 606, 609 ], true ) ) {
				return [ 'success' => false, 'code' => $reconcile_code, 'stage' => 'reconciliation', 'raw' => $plain ];
			}
		}

		return [
			'success'          => true,
			'ref_id'           => (string) ( $rrn !== '' ? $rrn : $pay_gate_tran_id ),
			'authority'        => 'asan:' . sha1( (string) $ref_id ),
			'pay_gate_tran_id' => (string) $pay_gate_tran_id,
			'rrn'              => (string) $rrn,
			'pan_last4'        => (string) $pan_last4,
			'verify_code'      => $verify_code,
			'reconcile_code'   => $reconcile_code,
			'raw'              => $plain,
		];
	}

	private function asanpardakht_requirements_ok() {
		if ( $this->asan_config_id <= 0 || $this->asan_username === '' || $this->asan_password === '' ) {
			return 'اطلاعات اصلی پذیرنده آسان پرداخت کامل نیست؛ Merchant Config ID، Username و Password را وارد کنید.';
		}
		if ( ( $this->asan_key === '' && $this->asan_iv !== '' ) || ( $this->asan_key !== '' && $this->asan_iv === '' ) ) {
			return 'برای آسان پرداخت، کلید رمزنگاری و IV باید یا هر دو خالی باشند یا هر دو تکمیل شوند.';
		}
		if ( ! class_exists( 'SoapClient' ) ) {
			return 'افزونه PHP SOAP روی سرور فعال نیست';
		}
		if ( $this->asanpardakht_encryption_enabled() && ( ! function_exists( 'mcrypt_encrypt' ) || ! defined( 'MCRYPT_RIJNDAEL_256' ) || ! defined( 'MCRYPT_MODE_CBC' ) ) ) {
			return 'برای حالت رمزنگاری‌شده آسان پرداخت، افزونه mcrypt/Rijndael-256 سرور باید فعال باشد. اگر پذیرنده شما Key/IV ندارد، هر دو فیلد Encryption Key و Encryption IV را خالی بگذارید.';
		}
		return true;
	}

	private function asanpardakht_encryption_enabled(): bool {
		return $this->asan_key !== '' && $this->asan_iv !== '';
	}

	private function asanpardakht_key_bytes( string $value ): string {
		$decoded = base64_decode( $value, true );
		return $decoded !== false ? $decoded : $value;
	}

	private function asanpardakht_pkcs7_pad( string $value, int $block_size = 32 ): string {
		$pad = $block_size - ( strlen( $value ) % $block_size );
		return $value . str_repeat( chr( $pad ), $pad );
	}

	private function asanpardakht_pkcs7_unpad( string $value ): string {
		if ( $value === '' ) { return ''; }
		$pad = ord( substr( $value, -1 ) );
		if ( $pad < 1 || $pad > 32 ) { return ''; }
		if ( substr( $value, -$pad ) !== str_repeat( chr( $pad ), $pad ) ) { return ''; }
		return substr( $value, 0, -$pad );
	}

	private function asanpardakht_encrypt( string $plain ): string {
		if ( ! $this->asanpardakht_encryption_enabled() ) {
			return $plain;
		}
		$key = $this->asanpardakht_key_bytes( $this->asan_key );
		$iv  = $this->asanpardakht_key_bytes( $this->asan_iv );
		$encrypted = @mcrypt_encrypt( MCRYPT_RIJNDAEL_256, $key, $this->asanpardakht_pkcs7_pad( $plain, 32 ), MCRYPT_MODE_CBC, $iv );
		return $encrypted === false ? '' : base64_encode( $encrypted );
	}

	private function asanpardakht_decrypt( string $encrypted ): string {
		if ( ! $this->asanpardakht_encryption_enabled() ) {
			return trim( $encrypted );
		}
		$key = $this->asanpardakht_key_bytes( $this->asan_key );
		$iv  = $this->asanpardakht_key_bytes( $this->asan_iv );
		$raw = base64_decode( $encrypted, true );
		if ( $raw === false ) { return ''; }
		$plain = @mcrypt_decrypt( MCRYPT_RIJNDAEL_256, $key, $raw, MCRYPT_MODE_CBC, $iv );
		return $plain === false ? '' : $this->asanpardakht_pkcs7_unpad( $plain );
	}

	private function asanpardakht_soap_call( string $method, array $params ): array {
		try {
			$client = new SoapClient( 'https://services.asanpardakht.net/paygate/merchantservices.asmx?WSDL', [
				'encoding'           => 'UTF-8',
				'connection_timeout' => 20,
				'trace'              => false,
				'exceptions'         => true,
			] );
			try {
				$result = $client->__soapCall( $method, [ $params ] );
			} catch ( Throwable $e ) {
				$result = $client->__soapCall( $method, array_values( $params ) );
			}
			$key = $method . 'Result';
			if ( is_object( $result ) && isset( $result->{$key} ) ) {
				$result = $result->{$key};
			}
			return [ 'result' => is_scalar( $result ) ? (string) $result : wp_json_encode( $result ) ];
		} catch ( Throwable $e ) {
			return [ 'error' => $e->getMessage() ];
		}
	}

}
