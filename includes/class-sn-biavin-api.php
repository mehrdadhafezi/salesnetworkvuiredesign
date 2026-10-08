<?php
if ( ! defined( 'ABSPATH' ) ) { exit; }

/**
 * Project-scoped client for the optional dot-wallet destination API.
 * Credentials are stored per content/project and are never exposed to JS.
 */
final class SN_Biavin_API {
	private static ?self $instance = null;

	public static function instance(): self {
		if ( null === self::$instance ) { self::$instance = new self(); }
		return self::$instance;
	}

	private function __construct() {}

	public function config_for_item( object $item ): ?array {
		global $wpdb;
		$table = $wpdb->prefix . 'sn_project_api_configs';
		if ( ! $this->table_exists( $table ) ) { return null; }
		$row = $wpdb->get_row( $wpdb->prepare( "SELECT endpoint_url,api_key,project_code,is_active FROM {$table} WHERE content_product_id=%d LIMIT 1", (int) $item->content_product_id ), ARRAY_A );
		if ( ! $row || empty( $row['is_active'] ) || trim( (string) $row['endpoint_url'] ) === '' || trim( (string) $row['api_key'] ) === '' || trim( (string) $row['project_code'] ) === '' ) { return null; }
		return [ 'endpoint_url' => untrailingslashit( trim( (string) $row['endpoint_url'] ) ), 'api_key' => trim( (string) $row['api_key'] ), 'project_code' => (string) $row['project_code'] ];
	}

	public function products( object $item, string $search = '' ): array|WP_Error {
		$config = $this->config_for_item( $item );
		if ( ! $config ) { return new WP_Error( 'sn_api_not_configured', 'برای این پروژه API مقصد تنظیم نشده است.' ); }
		$path = '/projects/' . rawurlencode( $config['project_code'] ) . '/products';
		if ( trim( $search ) !== '' ) { $path = add_query_arg( 'search', sanitize_text_field( $search ), $path ); }
		$result = $this->request( $config, 'GET', $path );
		if ( is_wp_error( $result ) ) { return $result; }
		$rows = isset( $result['products'] ) && is_array( $result['products'] ) ? $result['products'] : [];
		$out = [];
		foreach ( $rows as $row ) {
			if ( ! is_array( $row ) || absint( $row['id'] ?? 0 ) < 1 ) { continue; }
			$out[] = [ 'id' => absint( $row['id'] ), 'name' => sanitize_text_field( (string) ( $row['name'] ?? '' ) ), 'price' => (float) ( $row['price'] ?? 0 ), 'stock_status' => sanitize_key( (string) ( $row['stock_status'] ?? '' ) ) ];
		}
		return $out;
	}

	public function create_order( object $item, array $customer, int $product_id, int $quantity = 1, string $reference = '' ): array|WP_Error {
		$config = $this->config_for_item( $item );
		if ( ! $config ) { return new WP_Error( 'sn_api_not_configured', 'برای این پروژه API مقصد تنظیم نشده است.' ); }
		return $this->request( $config, 'POST', '/projects/' . rawurlencode( $config['project_code'] ) . '/orders', [
			'product_id' => $product_id,
			'quantity' => max( 1, min( 20, $quantity ) ),
			'customer' => [ 'name' => sanitize_text_field( (string) ( $customer['name'] ?? '' ) ), 'mobile' => sanitize_text_field( (string) ( $customer['mobile'] ?? '' ) ), 'email' => sanitize_email( (string) ( $customer['email'] ?? '' ) ) ],
			'reference' => sanitize_text_field( $reference ),
			'source' => 'sales-network',
		] );
	}

	public function charge_wallet( object $item, array $customer, string $wallet_type, float $amount, string $reference, string $note = '' ): array|WP_Error {
		$config = $this->config_for_item( $item );
		if ( ! $config ) { return new WP_Error( 'sn_api_not_configured', 'برای این پروژه API مقصد تنظیم نشده است.' ); }
		return $this->request( $config, 'POST', '/projects/' . rawurlencode( $config['project_code'] ) . '/wallet/charge', [
			'wallet_type' => in_array( $wallet_type, [ 'cash', 'installment' ], true ) ? $wallet_type : 'cash',
			'amount' => max( 0, $amount ),
			'customer' => [ 'name' => sanitize_text_field( (string) ( $customer['name'] ?? '' ) ), 'mobile' => sanitize_text_field( (string) ( $customer['mobile'] ?? '' ) ), 'email' => sanitize_email( (string) ( $customer['email'] ?? '' ) ) ],
			'reference' => sanitize_text_field( $reference ),
			'note' => sanitize_text_field( $note ),
			'source' => 'sales-network',
		] );
	}

	private function request( array $config, string $method, string $path, ?array $body = null ): array|WP_Error {
		$method = strtoupper( $method );
		$payload = $body !== null ? (string) wp_json_encode( $body, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ) : '';
		$timestamp = (string) time();
		$signature = hash_hmac( 'sha256', $method . "\n" . $path . "\n" . $timestamp . "\n" . $payload, (string) $config['api_key'] );
		$args = [ 'method' => $method, 'timeout' => 25, 'redirection' => 2, 'sslverify' => true, 'headers' => [ 'Accept' => 'application/json', 'Content-Type' => 'application/json', 'X-DCW-API-Key' => (string) $config['api_key'], 'X-DCW-Timestamp' => $timestamp, 'X-DCW-Signature' => $signature, 'X-DCW-Request-ID' => wp_generate_uuid4() ], 'body' => $payload ];
		$response = wp_remote_request( trailingslashit( (string) $config['endpoint_url'] ) . ltrim( $path, '/' ), $args );
		if ( is_wp_error( $response ) ) { return new WP_Error( 'sn_api_http', 'ارتباط با سایت مقصد برقرار نشد: ' . $response->get_error_message() ); }
		$code = (int) wp_remote_retrieve_response_code( $response );
		$data = json_decode( (string) wp_remote_retrieve_body( $response ), true );
		if ( ! is_array( $data ) ) { $data = []; }
		if ( $code < 200 || $code >= 300 || empty( $data['success'] ) ) { return new WP_Error( 'sn_api_remote', sanitize_text_field( (string) ( $data['message'] ?? 'سایت مقصد عملیات را نپذیرفت.' ) ), [ 'status' => $code, 'response' => $data ] ); }
		return $data;
	}

	private function table_exists( string $table ): bool {
		global $wpdb;
		return (string) $wpdb->get_var( $wpdb->prepare( 'SHOW TABLES LIKE %s', $table ) ) === $table;
	}
}
