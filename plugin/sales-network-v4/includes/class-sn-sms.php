<?php
if ( ! defined( 'ABSPATH' ) ) { exit; }

/**
 * سرویس ارسال پیامک
 * پشتیبانی از: کاوه‌نگار | فراز اس‌ام‌اس | ملی پیامک
 */
class SN_SMS {

	private string $last_error = '';

	public function get_last_error(): string {
		return $this->last_error;
	}

	/** Parse the documented Faraz response without treating HTTP success alone as acceptance. */
	private function faraz_response_ok( $response ): bool {
		$this->last_error = '';
		if ( is_wp_error( $response ) ) {
			$this->last_error = 'پاسخ سرویس پیامک دریافت نشد (' . sanitize_key( $response->get_error_code() ) . ')؛ نتیجه ارسال نامشخص است. پیش از ارسال مجدد، پنل پیامک را بررسی کنید.';
			return false;
		}
		$http_code = (int) wp_remote_retrieve_response_code( $response );
		$body = json_decode( (string) wp_remote_retrieve_body( $response ), true );
		$status = is_array( $body ) ? ( $body['status'] ?? null ) : null;
		$accepted = $status === true || ( is_string( $status ) && in_array( strtolower( trim( $status ) ), [ 'success', 'ok' ], true ) );
		if ( in_array( $http_code, [ 200, 201 ], true ) && $accepted ) { return true; }
		$messages = is_array( $body ) ? ( $body['messages'] ?? $body['message'] ?? [] ) : [];
		$parts = [];
		if ( is_string( $messages ) ) { $parts[] = $messages; }
		elseif ( is_array( $messages ) ) {
			array_walk_recursive( $messages, static function ( $value ) use ( &$parts ) {
				if ( is_string( $value ) && count( $parts ) < 5 ) { $parts[] = $value; }
			} );
		}
		$detail = sanitize_text_field( implode( '؛ ', $parts ) );
		foreach ( [ $this->api_key, $this->meli_password ] as $secret ) {
			if ( $secret !== '' ) { $detail = str_replace( $secret, '***', $detail ); }
		}
		$detail = function_exists( 'mb_substr' ) ? mb_substr( $detail, 0, 400 ) : substr( $detail, 0, 400 );
		$this->last_error = ! is_array( $body )
			? 'پاسخ سرویس پیامک معتبر نبود؛ نتیجه ارسال را در پنل پیامک بررسی کنید (HTTP ' . $http_code . ').'
			: 'سرویس پیامک ارسال را تأیید نکرد (HTTP ' . $http_code . ').' . ( $detail !== '' ? ' ' . $detail : ' تنظیمات خط ارسال، الگو و اعتبار پنل را بررسی کنید.' );
		return false;
	}

	private string $provider;
	private string $api_key;
	private string $sender;
	private string $invoice_pattern;
	private string $customer_otp_pattern;
	private string $customer_otp_variable;

	// --- ملی پیامک ---
	private string $meli_username;
	private string $meli_password;
	private string $meli_body_id_invoice;

	public function __construct() {
		$this->provider        = trim( (string) get_option( 'sn_sms_provider', 'kavenegar' ) );
		$this->api_key         = trim( (string) get_option( 'sn_sms_api_key', '' ) );
		$this->sender          = trim( (string) get_option( 'sn_sms_sender', '' ) );
		$this->invoice_pattern = trim( (string) get_option( 'sn_faraz_pattern_invoice', '' ) );
		$this->customer_otp_pattern = trim( (string) get_option( 'sn_faraz_pattern_customer_otp', '' ) );
		$this->customer_otp_variable = sanitize_key( (string) get_option( 'sn_faraz_pattern_customer_otp_variable', 'code' ) ) ?: 'code';

		// تنظیمات ملی پیامک (الگو)
		$this->meli_username         = trim( (string) get_option( 'sn_meli_username', '' ) );
		$this->meli_password         = trim( (string) get_option( 'sn_meli_password', '' ) );
		$this->meli_body_id_invoice  = trim( (string) get_option( 'sn_meli_body_id_invoice', '' ) );

		// مجاز کردن دامنه‌های فراز اس‌ام‌اس در صورت block بودن external requests
		add_filter( 'http_request_host_is_external', [ $this, 'allow_sms_hosts' ], 10, 2 );
	}

	public function allow_sms_hosts( bool $allow, string $host ): bool {
		$allowed_hosts = [
			'api.iranpayamak.com',
			'rest.ippanel.com',
			'ippanel.com',
			'app.farazsms.com',
		];
		if ( in_array( strtolower( $host ), $allowed_hosts, true ) ) {
			return true;
		}
		return $allow;
	}

	/** ارسال پیامک ساده */
	public function send( string $to, string $message ): bool {
		$this->last_error = '';
		$to = SN_Helpers::normalize_mobile( $to );

		if ( empty( $to ) ) {
			$this->last_error = 'شماره مقصد پیامک خالی است.';
			return false;
		}
		if ( empty( $this->api_key ) && ! ( $this->provider === 'melipayamak' && $this->meli_username !== '' && $this->meli_password !== '' ) ) {
			$this->last_error = 'اطلاعات اتصال سرویس پیامک تنظیم نشده است.';
			return false;
		}

		switch ( $this->provider ) {
			case 'kavenegar':
				return $this->send_kavenegar( $to, $message );

			case 'faraz':
				return $this->send_faraz_simple( $to, $message );

			case 'melipayamak':
				return $this->send_melipayamak( $to, $message );
		}

		$this->last_error = 'سرویس‌دهنده پیامک انتخاب‌شده پشتیبانی نمی‌شود.';
		return false;
	}

	/**
	 * Send the customer-portal login code.
	 *
	 * Faraz must use a dedicated pattern here. The simple-message endpoint was
	 * returning false on the production account and the portal intentionally
	 * surfaced that provider failure as HTTP 502.
	 */
	public function send_customer_portal_otp( string $to, string $code ): bool {
		$to = SN_Helpers::normalize_mobile( $to );
		$code = preg_replace( '/\D+/', '', SN_Helpers::to_english_nums( $code ) );
		if ( empty( $this->api_key ) || ! SN_Helpers::is_valid_mobile( $to ) || ! preg_match( '/^\d{6}$/', $code ) ) { return false; }
		if ( $this->provider === 'faraz' ) {
			if ( $this->customer_otp_pattern === '' ) {
				error_log( 'SN Customer OTP Pattern: کد پترن فراز تنظیم نشده است.' );
				return false;
			}
			return $this->send_faraz_pattern_once(
				$to,
				$this->customer_otp_pattern,
				[ $this->customer_otp_variable => $code ]
			);
		}
		return $this->send( $to, "کد ورود به پروفایل بیاوین: {$code}\nاعتبار کد: ۵ دقیقه" );
	}

	/** One bounded Faraz pattern request; no second call to the same endpoint. */
	private function send_faraz_pattern_once( string $recipient, string $pattern_code, array $attributes ): bool {
		if ( $this->api_key === '' || $this->sender === '' || $pattern_code === '' ) {
			$this->last_error = 'کلید سرویس، خط ارسال یا کد الگوی پیامک تنظیم نشده است.';
			return false;
		}
		$payload = [
			'code'          => $pattern_code,
			'recipient'     => $recipient,
			'line_number'   => $this->sender,
			'attributes'    => $attributes,
			'number_format' => 'english',
		];
		$res = wp_remote_post( 'https://api.iranpayamak.com/ws/v1/sms/pattern', [
			'timeout'   => 10,
			'sslverify' => false,
			'headers'   => [
				'Accept'       => 'application/json',
				'Content-Type' => 'application/json',
				'Api-Key'      => $this->api_key,
			],
			'body' => wp_json_encode( $payload, JSON_UNESCAPED_UNICODE ),
		] );
		return $this->faraz_response_ok( $res );
	}

	private function is_short_invoice_url( string $invoice_url ): bool {
		$invoice_url = trim( (string) $invoice_url );
		$path = (string) wp_parse_url( $invoice_url, PHP_URL_PATH );
		if ( $path === '' && preg_match( '~(?:^|/)(i/[A-Za-z0-9_-]+/?)(?:$|[?#])~', $invoice_url, $m ) ) {
			$path = '/' . trim( (string) $m[1], '/' ) . '/';
		}
		return (bool) preg_match( '~^/i/[A-Za-z0-9_-]+/?$~', $path );
	}

	private function normalize_invoice_sms_amount( $amount ): string {
		$amount = trim( SN_Helpers::to_english_nums( (string) $amount ) );
		if ( $amount === '' ) {
			return '';
		}
		$amount = str_replace( [ ',', '٬', '،', ' ' ], '', $amount );
		if ( preg_match( '/^-?\d+(?:\.\d+)?$/', $amount ) ) {
			return (string) (int) floor( (float) $amount );
		}
		if ( strpos( $amount, '.' ) !== false ) {
			$amount = strtok( $amount, '.' );
		}
		return (string) preg_replace( '/[^0-9]/', '', $amount );
	}

	/** ساخت پیش‌نمایش دقیق پیامک فاکتور بدون ارسال واقعی */
	public function build_invoice_link_preview( string $phone, string $invoice_code, string $invoice_url, string $customer_name, $amount = '', $card_number = '' ): array {
		$phone         = SN_Helpers::normalize_mobile( $phone );
		$invoice_code  = trim( $invoice_code );
		$invoice_url   = trim( $invoice_url );
		$customer_name = trim( $customer_name );
		$amount        = trim( (string) $amount );
		$card_number   = trim( (string) $card_number );
		$normalized_amount = $this->normalize_invoice_sms_amount( $amount );
		if ( empty( $customer_name ) ) {
			$customer_name = 'مشتری گرامی';
		}
		if ( (string) get_option( 'sn_card_to_card_enabled', '1' ) !== '1' ) {
			$card_number = '';
		} elseif ( empty( $card_number ) ) {
			$card_number = get_option( 'sn_card_number', '' );
		}
		$base = [
			'preview_version' => defined( 'SN_VERSION' ) ? SN_VERSION : 'unknown',
			'sms_class_file'  => __FILE__,
			'provider'      => $this->provider,
			'to'            => $phone,
			'invoice_code'  => $invoice_code,
			'invoice_url'   => $invoice_url,
			'is_short_url'  => $this->is_short_invoice_url( $invoice_url ),
			'customer_name' => $customer_name,
			'amount'        => $amount,
			'normalized_amount' => $normalized_amount,
			'card_number'   => $card_number,
			'sender'        => $this->sender,
		];

		if ( $this->provider === 'faraz' && ! empty( $this->invoice_pattern ) ) {
			$attributes = [
				'customer_name' => $customer_name,
				'invoice_code'  => $invoice_code,
				'invoice_url'   => $invoice_url,
				'amount'        => $normalized_amount,
				'card_number'   => $card_number,
			];
			return $base + [
				'mode'         => 'faraz_pattern',
				'endpoint'     => 'https://api.iranpayamak.com/ws/v1/sms/pattern',
				'pattern_code' => $this->invoice_pattern,
				'payload'      => [
					'code'          => $this->invoice_pattern,
					'recipient'     => $phone,
					'line_number'   => $this->sender,
					'attributes'    => $attributes,
					'number_format' => 'english',
				],
			];
		}

		if ( $this->provider === 'melipayamak' && ! empty( $this->meli_body_id_invoice ) ) {
			return $base + [
				'mode'    => 'melipayamak_pattern',
				'endpoint'=> 'https://rest.payamak-panel.com/api/SendSMS/BaseServiceNumber',
				'body_id' => (int) $this->meli_body_id_invoice,
				'vars'    => [ $customer_name, $invoice_code, $invoice_url, $amount, $card_number ],
			];
		}

		$template = get_option( 'sn_sms_invoice_template', '' );
		if ( ! empty( $template ) ) {
			$message = str_replace(
				[ '{customer_name}', '{invoice_code}', '{invoice_url}', '{amount}', '{card_number}' ],
				[ $customer_name, $invoice_code, $invoice_url, $amount, $card_number ],
				$template
			);
		} else {
			$message  = "مشتری گرامی {$customer_name}\n";
			$message .= "فاکتور شما با کد {$invoice_code} آماده است.\n";
			if ( ! empty( $amount ) ) {
				$message .= "مبلغ: {$amount} تومان\n";
			}
			$message .= "برای مشاهده و پرداخت:\n{$invoice_url}";
			if ( ! empty( $card_number ) ) {
				$message .= "\n\nیا کارت به کارت به شماره:\n{$card_number}";
			}
		}

		return $base + [
			'mode'    => $this->provider === 'kavenegar' ? 'kavenegar_simple' : 'simple',
			'message' => $message,
		];
	}

	/** ارسال لینک فاکتور به مشتری */
	public function send_invoice_link( string $phone, string $invoice_code, string $invoice_url, string $customer_name, $amount = '', $card_number = '' ): bool {
		$this->last_error = '';
		$phone         = SN_Helpers::normalize_mobile( $phone );
		$invoice_code  = trim( $invoice_code );
		$invoice_url   = trim( $invoice_url );
		$customer_name = trim( $customer_name );
		$amount        = trim( (string) $amount );
		$card_number   = trim( (string) $card_number );

		if ( empty( $phone ) || empty( $invoice_url ) ) {
			$this->last_error = 'شماره مشتری یا لینک فاکتور خالی است.';
			return false;
		}

		if ( empty( $customer_name ) ) {
			$customer_name = 'مشتری گرامی';
		}
		
		// شماره کارت پیش‌فرض فقط وقتی قابلیت کارت‌به‌کارت روشن است وارد
		// پیامک شود؛ مقدار خالی در حالت خاموش نباید دوباره از تنظیمات پر شود.
		if ( (string) get_option( 'sn_card_to_card_enabled', '1' ) !== '1' ) {
			$card_number = '';
		} elseif ( empty( $card_number ) ) {
			$card_number = get_option( 'sn_card_number', '' );
		}

		$sn_invoice_sms_debug_payload = $this->build_invoice_link_preview( $phone, $invoice_code, $invoice_url, $customer_name, $amount, $card_number );
		update_option( 'sn_last_invoice_sms_debug', [
			'preview_only' => false,
			'created_at'   => current_time( 'mysql' ),
			'payload'      => $sn_invoice_sms_debug_payload,
		], false );
		error_log( 'SN Invoice SMS Preview | ' . wp_json_encode( $sn_invoice_sms_debug_payload, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ) );

		// اگر سرویس‌دهنده فراز باشد و کد پترن تعریف شده باشد، با پترن بفرست
		if ( $this->provider === 'faraz' && ! empty( $this->invoice_pattern ) ) {
			return $this->send_faraz_pattern_once(
				$phone,
				$this->invoice_pattern,
				[
					'customer_name' => $customer_name,
					'invoice_code'  => $invoice_code,
					'invoice_url'   => $invoice_url,
					'amount'        => $this->normalize_invoice_sms_amount( $amount ),
					'card_number'   => $card_number,
				]
			);
		}

		// اگر سرویس‌دهنده ملی پیامک باشد و bodyId فاکتور تعریف شده باشد، با الگو بفرست
		if ( $this->provider === 'melipayamak' && ! empty( $this->meli_body_id_invoice ) ) {
			return $this->send_melipayamak_pattern(
				$phone,
				(int) $this->meli_body_id_invoice,
				[ $customer_name, $invoice_code, $invoice_url, $amount, $card_number ]
			);
		}

		// استفاده از template از تنظیمات
		$template = get_option( 'sn_sms_invoice_template', '' );
		
		// اگر template داریم از اون استفاده کن، وگرنه متن پیش‌فرض
		if ( ! empty( $template ) ) {
			$message = str_replace(
				[ '{customer_name}', '{invoice_code}', '{invoice_url}', '{amount}', '{card_number}' ],
				[ $customer_name, $invoice_code, $invoice_url, $amount, $card_number ],
				$template
			);
		} else {
			// متن پیش‌فرض
			$message  = "مشتری گرامی {$customer_name}\n";
			$message .= "فاکتور شما با کد {$invoice_code} آماده است.\n";
			
			if ( ! empty( $amount ) ) {
				$message .= "مبلغ: {$amount} تومان\n";
			}
			
			$message .= "برای مشاهده و پرداخت:\n{$invoice_url}";
			
			if ( ! empty( $card_number ) ) {
				$message .= "\n\nیا کارت به کارت به شماره:\n{$card_number}";
			}
		}

		return $this->send( $phone, $message );
	}

	// --- Kavenegar ---
	private function send_kavenegar( string $to, string $message ): bool {
		$url = "https://api.kavenegar.com/v1/{$this->api_key}/sms/send.json";

		$res = wp_remote_post( $url, [
			'timeout' => 15,
			'body'    => [
				'receptor' => $to,
				'message'  => $message,
				'sender'   => $this->sender,
			],
		] );

		if ( is_wp_error( $res ) ) {
			error_log( 'SN Kavenegar Error: ' . $res->get_error_message() );
			$this->last_error = 'پاسخ کاوه‌نگار دریافت نشد؛ وضعیت ارسال را در پنل پیامک بررسی کنید.';
			return false;
		}

		$body = json_decode( wp_remote_retrieve_body( $res ), true );
		$ok = isset( $body['return']['status'] ) && (int) $body['return']['status'] === 200;
		if ( ! $ok ) { $this->last_error = 'کاوه‌نگار ارسال را تأیید نکرد؛ کد پاسخ: ' . (int) ( $body['return']['status'] ?? wp_remote_retrieve_response_code( $res ) ); }
		return $ok;
	}

	// --- Faraz SMS : Simple (IPPanel v1) ---
	private function send_faraz_simple( string $to, string $message ): bool {
		// داکیومنت: https://docs.iranpayamak.com/send-simple-sms-13909967e0.md
		$url = 'https://api.iranpayamak.com/ws/v1/sms/simple';

		// iranpayamak فرمت 09xx میخواد
		$originator = $this->sender;

		$payload = [
			'text'          => $message,
			'line_number'   => $originator,
			'recipients'    => [ $to ],
			'number_format' => 'english',
			'schedule'      => null,
		];

		$res = wp_remote_post( $url, [
			'timeout'   => 20,
			'sslverify' => false,
			'headers'   => [
				'Accept'        => 'application/json',
				'Content-Type'  => 'application/json',
				'Api-Key'      => $this->api_key,
			],
			'body' => wp_json_encode( $payload, JSON_UNESCAPED_UNICODE ),
		] );

		return $this->faraz_response_ok( $res );
	}

	// --- Faraz SMS : Pattern ---
	// فراز SMS از IPPanel استفاده میکنه
	// اگر rest.ippanel.com بلاک بود (سرورهای ایران)، fallback به endpoint داخلی ippanel.com:8080
	public function send_faraz_pattern( string $to, string $pattern_code, array $attributes ): bool {
		if ( empty( $pattern_code ) || empty( $to ) ) {
			error_log( 'SN Faraz Pattern: pattern_code یا to خالی است' );
			return false;
		}

		// تبدیل شماره: 09xx → +989xx
		// iranpayamak فرمت 09xxxxxxxxx میخواد (بدون +98)
		$recipient  = $to;  // همان 09xx که از SN_Helpers آمده
		$originator = $this->sender;

		// --- روش اول: REST API (rest.ippanel.com) ---
		$result = $this->send_faraz_pattern_rest( $recipient, $originator, $pattern_code, $attributes );
		if ( $result !== null ) {
			return $result;
		}

		// --- روش دوم: endpoint داخلی ایران (ippanel.com:8080) ---
		error_log( 'SN Faraz: REST بلاک شد، تلاش با endpoint داخلی...' );
		return $this->send_faraz_pattern_domestic( $recipient, $pattern_code, $attributes );
	}

	private function send_faraz_pattern_rest( string $recipient, string $originator, string $pattern_code, array $attributes ): ?bool {
		// داکیومنت رسمی: https://docs.iranpayamak.com/send-pattern-based-sms-13925177e0.md
		// route: POST https://api.iranpayamak.com/ws/v1/sms/pattern
		$url = 'https://api.iranpayamak.com/ws/v1/sms/pattern';

		$payload = [
			'code'          => $pattern_code,     // نه patternCode
			'recipient'     => $recipient,         // فرمت 09xxxxxxxxx
			'line_number'   => $originator,        // نه originator
			'attributes'    => $attributes,        // نه values — آرایه associative
			'number_format' => 'english',
		];

		$res = wp_remote_post( $url, [
			'timeout'   => 20,
			'sslverify' => false,
			'headers'   => [
				'Accept'       => 'application/json',
				'Content-Type' => 'application/json',
				'Api-Key'      => $this->api_key,
			],
			'body' => wp_json_encode( $payload, JSON_UNESCAPED_UNICODE ),
		] );

		if ( is_wp_error( $res ) ) {
			error_log( 'SN Faraz Pattern WP_Error: ' . $res->get_error_message() );
			return null;
		}

		$http_code = (int) wp_remote_retrieve_response_code( $res );
		$raw       = wp_remote_retrieve_body( $res );
		$body      = json_decode( $raw, true );

		error_log( 'SN Faraz Pattern | HTTP:' . $http_code . ' | to:' . $recipient . ' | code:' . $pattern_code . ' | Response:' . $raw );

		// 201 = موفق (Created)
		if ( $http_code === 201 || $http_code === 200 ) {
			if ( isset( $body['status'] ) ) {
				return in_array( $body['status'], [ 'success', 'OK', true ], true );
			}
			return true;
		}
		if ( $http_code === 401 || $http_code === 403 ) {
			error_log( 'SN Faraz Pattern Auth FAILED — API Key نامعتبر' );
			return false;
		}
		return null;
	}

	private function send_faraz_pattern_domestic( string $recipient, string $pattern_code, array $attributes ): bool {
		// fallback: همان API جدید iranpayamak با timeout بیشتر
		$url     = 'https://api.iranpayamak.com/ws/v1/sms/pattern';
		$payload = [
			'code'          => $pattern_code,
			'recipient'     => $recipient,
			'line_number'   => $this->sender,
			'attributes'    => $attributes,
			'number_format' => 'english',
		];
		$res = wp_remote_post( $url, [
			'timeout'   => 30,
			'sslverify' => false,
			'headers'   => [
				'Accept'       => 'application/json',
				'Content-Type' => 'application/json',
				'Api-Key'      => $this->api_key,
			],
			'body' => wp_json_encode( $payload, JSON_UNESCAPED_UNICODE ),
		] );
		if ( is_wp_error( $res ) ) {
			error_log( 'SN Faraz Fallback WP_Error: ' . $res->get_error_message() );
			return false;
		}
		$http_code = (int) wp_remote_retrieve_response_code( $res );
		$raw       = wp_remote_retrieve_body( $res );
		error_log( 'SN Faraz Fallback | HTTP:' . $http_code . ' | ' . $raw );
		$body = json_decode( $raw, true );
		if ( $http_code === 200 || $http_code === 201 ) {
			return ! isset( $body['status'] ) || in_array( $body['status'], [ 'success', 'OK', true ], true );
		}
		return false;
	}

	// --- Meli Payamak : Simple ---
	private function send_melipayamak( string $to, string $message ): bool {
		$url      = 'https://rest.payamak-panel.com/api/SendSMS/SendSMS';
		$username = $this->meli_username ?: explode( ':', $this->api_key . ':' )[0];
		$password = $this->meli_password ?: ( explode( ':', $this->api_key . ':' )[1] ?? '' );

		$res = wp_remote_post( $url, [
			'timeout' => 15,
			'body'    => [
				'username' => $username,
				'password' => $password,
				'to'       => $to,
				'from'     => $this->sender,
				'text'     => $message,
				'isFlash'  => 'false',
			],
		] );

		if ( is_wp_error( $res ) ) {
			error_log( 'SN MeliPayamak Error: ' . $res->get_error_message() );
			$this->last_error = 'پاسخ ملی‌پیامک دریافت نشد؛ وضعیت ارسال را در پنل پیامک بررسی کنید.';
			return false;
		}

		$body = json_decode( wp_remote_retrieve_body( $res ), true );
		$ok = isset( $body['RetStatus'] ) && (int) $body['RetStatus'] === 1;
		if ( ! $ok ) { $this->last_error = 'ملی‌پیامک ارسال را تأیید نکرد؛ کد پاسخ: ' . (int) ( $body['RetStatus'] ?? wp_remote_retrieve_response_code( $res ) ); }
		return $ok;
	}

	// --- Meli Payamak : Pattern (الگوی اشتراکی) ---
	public function send_melipayamak_pattern( string $to, int $body_id, array $vars ): bool {
		if ( empty( $this->meli_username ) || empty( $this->meli_password ) || ! $body_id ) {
			$this->last_error = 'نام کاربری، رمز اتصال یا شناسه الگوی ملی‌پیامک تنظیم نشده است.';
			error_log( 'SN MeliPayamak Pattern: تنظیمات ناقص (username/password/bodyId)' );
			return false;
		}

		// متغیرها با ; جدا می‌شوند
		$text = implode( ';', $vars );

		$payload = [
			'username' => $this->meli_username,
			'password' => $this->meli_password,
			'text'     => $text,
			'to'       => $to,
			'bodyId'   => $body_id,
		];

		$res = wp_remote_post(
			'https://rest.payamak-panel.com/api/SendSMS/BaseServiceNumber',
			[
				'timeout' => 15,
				'headers' => [
					'Content-Type' => 'application/json',
					'Accept'       => 'application/json',
				],
				'body' => wp_json_encode( $payload, JSON_UNESCAPED_UNICODE ),
			]
		);

		if ( is_wp_error( $res ) ) {
			error_log( 'SN MeliPayamak Pattern Error: ' . $res->get_error_message() );
			$this->last_error = 'پاسخ ملی‌پیامک دریافت نشد؛ وضعیت ارسال را در پنل پیامک بررسی کنید.';
			return false;
		}

		$body = json_decode( wp_remote_retrieve_body( $res ), true );

		// ارسال موفق: RetStatus=1 و Value بیشتر از ۱۵ رقم
		if (
			isset( $body['RetStatus'] ) &&
			(int) $body['RetStatus'] === 1 &&
			isset( $body['Value'] ) &&
			strlen( (string) $body['Value'] ) > 15
		) {
			return true;
		}

		$error_code = $body['Value'] ?? 'unknown';
		$this->last_error = 'ملی‌پیامک ارسال الگو را تأیید نکرد؛ کد پاسخ: ' . (int) ( $body['RetStatus'] ?? wp_remote_retrieve_response_code( $res ) ) . ( is_numeric( $error_code ) ? ' / ' . (string) $error_code : '' );
		error_log( 'SN MeliPayamak Pattern Failed — کد خطا: ' . $error_code . ' | پاسخ: ' . wp_json_encode( $body, JSON_UNESCAPED_UNICODE ) );
		return false;
	}
}
