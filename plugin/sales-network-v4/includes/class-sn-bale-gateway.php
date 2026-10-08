<?php
if ( ! defined( 'ABSPATH' ) ) { exit; }

/**
 * Signed CRM -> Bale Gateway client.
 *
 * Safir credentials and the Bale Bot token stay on the external gateway.
 * CRM only stores/reads the Gateway URL + shared HMAC secret.
 */
class SN_Bale_Gateway {
	private string $base_url;
	private string $secret;
	private int $timeout;

	public function __construct() {
		$this->base_url = defined( 'SN_BALE_GATEWAY_URL' )
			? trim( (string) constant( 'SN_BALE_GATEWAY_URL' ) )
			: trim( (string) get_option( 'sn_bale_gateway_url', 'https://bale.andishesazanco.ir' ) );
		$this->secret = defined( 'SN_BALE_GATEWAY_SHARED_SECRET' )
			? trim( (string) constant( 'SN_BALE_GATEWAY_SHARED_SECRET' ) )
			: trim( (string) get_option( 'sn_bale_gateway_shared_secret', '' ) );
		$this->timeout = max( 3, min( 30, absint( get_option( 'sn_bale_gateway_timeout', 15 ) ) ) );
		add_filter( 'http_request_host_is_external', [ $this, 'allow_gateway_host' ], 10, 2 );
	}

	public function allow_gateway_host( bool $allow, string $host ): bool {
		$configured_host = strtolower( (string) wp_parse_url( $this->base_url, PHP_URL_HOST ) );
		return $configured_host !== '' && strtolower( trim( $host ) ) === $configured_host ? true : $allow;
	}

	public function is_configured(): bool {
		return $this->base_url !== '' && stripos( $this->base_url, 'https://' ) === 0 && strlen( $this->secret ) >= 32;
	}

	public function secret_source(): string {
		if ( defined( 'SN_BALE_GATEWAY_SHARED_SECRET' ) && trim( (string) constant( 'SN_BALE_GATEWAY_SHARED_SECRET' ) ) !== '' ) { return 'constant'; }
		return $this->secret !== '' ? 'option' : 'missing';
	}

	public function gateway_url(): string { return $this->base_url; }

	public function send_notification( array $payload ): array {
		if ( ! $this->is_configured() ) {
			return $this->error_result( 'not_configured', 'آدرس Gateway یا Shared Secret بله کامل تنظیم نشده است.', false );
		}
		$path = '/api/v1/notify';
		$raw = wp_json_encode( $payload, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES );
		if ( ! is_string( $raw ) || $raw === '' ) {
			return $this->error_result( 'json_encode_failed', 'ساخت payload بله ناموفق بود.', false );
		}
		$timestamp = (string) time();
		try { $nonce = bin2hex( random_bytes( 16 ) ); }
		catch ( Throwable $e ) { $nonce = wp_generate_password( 32, false, false ); }
		$canonical = $timestamp . "\n" . $nonce . "\nPOST\n" . $path . "\n" . hash( 'sha256', $raw );
		$signature = hash_hmac( 'sha256', $canonical, $this->secret );

		$response = wp_remote_post( rtrim( $this->base_url, '/' ) . $path, [
			'timeout' => $this->timeout,
			'redirection' => 0,
			'sslverify' => true,
			'headers' => [
				'Accept' => 'application/json',
				'Content-Type' => 'application/json',
				'X-SN-Timestamp' => $timestamp,
				'X-SN-Nonce' => $nonce,
				'X-SN-Signature' => $signature,
			],
			'body' => $raw,
		] );
		if ( is_wp_error( $response ) ) {
			return $this->error_result( 'transport_error', sanitize_text_field( $response->get_error_message() ), true );
		}
		$http = (int) wp_remote_retrieve_response_code( $response );
		$body = json_decode( (string) wp_remote_retrieve_body( $response ), true );
		$body = is_array( $body ) ? $body : [];
		if ( $http >= 200 && $http < 300 && ! empty( $body['success'] ) ) {
			return [
				'success' => true,
				'status' => 'sent',
				'message_id' => sanitize_text_field( (string) ( $body['message_id'] ?? '' ) ),
				'error_code' => null,
				'error_message' => '',
				'retryable' => false,
				'http_code' => $http,
			];
		}

		$errors = isset( $body['errors'] ) && is_array( $body['errors'] ) ? $body['errors'] : [];
		$first = ! empty( $errors[0] ) && is_array( $errors[0] ) ? $errors[0] : [];
		$code = $first['code'] ?? ( $body['error'] ?? ( $http >= 500 ? 'http_' . $http : 'gateway_error' ) );
		$description = sanitize_text_field( (string) ( $first['description'] ?? ( $body['message'] ?? ( $body['error'] ?? 'Bale Gateway request failed' ) ) ) );
		$retryable = in_array( (string) $code, [ '2', '3', 'transport_error' ], true ) || $http === 429 || $http >= 500;
		$status = (string) $code === '17' ? 'skipped' : 'failed';
		$result = $this->error_result( $code, $description, $retryable );
		$result['status'] = $status;
		$result['http_code'] = $http;
		return $result;
	}

	private function error_result( $code, string $message, bool $retryable ): array {
		return [
			'success' => false,
			'status' => 'failed',
			'message_id' => '',
			'error_code' => $code,
			'error_message' => sanitize_text_field( $message ),
			'retryable' => $retryable,
			'http_code' => 0,
		];
	}
}
