<?php
if ( ! defined( 'ABSPATH' ) ) { exit; }

/** Additive Bale channel. Existing SMS outcomes remain authoritative. */
class SN_Notification_Service {
	private static ?self $instance = null;
	public static function instance(): self { return self::$instance ??= new self(); }
	private function __construct() {}

	public function bale_enabled(): bool { return (string) get_option( 'sn_bale_enabled', '0' ) === '1'; }
	public function mirror_enabled(): bool { return (string) get_option( 'sn_bale_mirror_sms', '0' ) === '1'; }
	public function otp_mirror_enabled(): bool { return (string) get_option( 'sn_bale_mirror_otp', '0' ) === '1'; }

	public function mirror_text( string $event, string $phone, string $text, array $context = [] ): array {
		if ( empty( $context['force'] ) && ( ! $this->bale_enabled() || ! $this->mirror_enabled() ) ) { return $this->skipped( 'mirror_disabled' ); }
		$message = [ 'text' => trim( $text ), 'secure' => ! empty( $context['secure'] ) ];
		if ( ! empty( $context['buttons'] ) && is_array( $context['buttons'] ) ) { $message['buttons'] = $context['buttons']; }
		if ( ! empty( $context['copy_text'] ) ) { $message['copy_text'] = (string) $context['copy_text']; }
		return $this->send_bale( $event, $phone, $message, $context );
	}

	public function mirror_otp( string $event, string $phone, string $otp, array $context = [] ): array {
		if ( empty( $context['force'] ) && ( ! $this->bale_enabled() || ! $this->otp_mirror_enabled() ) ) { return $this->skipped( 'otp_mirror_disabled' ); }
		$otp = preg_replace( '/\D+/', '', SN_Helpers::to_english_nums( $otp ) );
		if ( ! is_string( $otp ) || $otp === '' ) { return $this->failed_local( 'invalid_otp', 'OTP نامعتبر است.' ); }
		return $this->send_bale( $event, $phone, [ 'otp' => $otp ], $context + [ 'sensitive' => true ] );
	}

	public function mirror_invoice( string $event, string $phone, string $invoice_code, string $invoice_url, string $customer_name, $amount = '', string $card_number = '', array $context = [] ): array {
		$template = trim( (string) get_option( 'sn_bale_invoice_template', '' ) );
		if ( $template === '' ) { $template = trim( (string) get_option( 'sn_sms_invoice_template', '' ) ); }
		if ( $customer_name === '' ) { $customer_name = 'مشتری گرامی'; }
		if ( $template !== '' ) {
			$text = str_replace(
				[ '{customer_name}', '{invoice_code}', '{invoice_url}', '{amount}', '{card_number}', '{payment_type}' ],
				[ $customer_name, $invoice_code, $invoice_url, (string) $amount, $card_number, sanitize_text_field( (string) ( $context['payment_type'] ?? '' ) ) ],
				$template
			);
		} else {
			$text = "مشتری گرامی {$customer_name}\nفاکتور شما با کد {$invoice_code} آماده است.";
			if ( (string) $amount !== '' ) { $text .= "\nمبلغ: {$amount} تومان"; }
			$text .= "\nبرای مشاهده و پرداخت:\n{$invoice_url}";
			if ( $card_number !== '' ) { $text .= "\n\nشماره کارت: {$card_number}"; }
		}
		$buttons = [];
		if ( $invoice_url !== '' ) { $buttons[] = [ [ 'text' => 'مشاهده و پرداخت', 'url' => esc_url_raw( $invoice_url ) ] ]; }
		if ( $card_number !== '' ) { $buttons[] = [ [ 'text' => 'کپی شماره کارت', 'copy_text' => preg_replace( '/\s+/', '', $card_number ) ] ]; }
		$context['buttons'] = $buttons;
		return $this->mirror_text( $event, $phone, $text, $context );
	}

	public function send_bale( string $event, string $phone, array $message, array $context = [] ): array {
		$phone = SN_Helpers::normalize_mobile( $phone );
		if ( ! SN_Helpers::is_valid_mobile( $phone ) ) { return $this->failed_local( 8, 'invalid_phone' ); }
		if ( empty( $message['otp'] ) && trim( (string) ( $message['text'] ?? '' ) ) === '' && empty( $message['template_id'] ) ) { return $this->failed_local( 'empty_message', 'empty_message' ); }
		$request_id = $this->request_id( $event, $phone, $message, $context );
		$prior = $this->delivery_by_request( 'bale', $request_id );
		if ( $prior && (string) ( $prior['status'] ?? '' ) === 'sent' ) { return $this->duplicate_result( $prior ); }
		$secure = ! empty( $message['secure'] ) || ! empty( $context['secure'] );
		$delivery_id = $this->start_delivery( $event, $phone, $request_id, $context, $secure, $prior );
		$payload = [
			'event_key' => $event,
			'request_id' => $request_id,
			'recipient_phone' => $phone,
			'entity_type' => sanitize_key( (string) ( $context['entity_type'] ?? '' ) ),
			'entity_id' => absint( $context['entity_id'] ?? 0 ),
			'message' => $message,
		];
		$result = ( new SN_Bale_Gateway() )->send_notification( $payload );
		$this->finish_delivery( $delivery_id, $result );
		return $result + [ 'delivery_id' => $delivery_id, 'request_id' => $request_id ];
	}

	private function request_id( string $event, string $phone, array $message, array $context ): string {
		$event = trim( (string) preg_replace( '/[^a-zA-Z0-9._:-]+/', '-', trim( $event ) ), '-' );
		$entity_type = sanitize_key( (string) ( $context['entity_type'] ?? '' ) );
		$entity_id = absint( $context['entity_id'] ?? 0 );
		$dedupe_key = trim( (string) ( $context['dedupe_key'] ?? '' ) );
		if ( $dedupe_key === '' ) {
			$fingerprint = ! empty( $context['sensitive'] ) || ! empty( $context['secure'] )
				? wp_json_encode( [ 'event' => $event, 'entity' => $entity_id ] )
				: wp_json_encode( $message, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES );
			$dedupe_key = hash( 'sha256', (string) $fingerprint );
		}
		return 'sn-' . substr( hash( 'sha256', $event . '|' . $phone . '|' . $entity_type . '|' . $entity_id . '|' . $dedupe_key ), 0, 56 );
	}

	private function metadata( array $context ): array {
		$allowed = [ 'case_id', 'invoice_id', 'stage_no', 'user_id', 'membership_id', 'operation_id', 'payment_type', 'source', 'resend' ];
		$out = [];
		foreach ( $allowed as $key ) {
			if ( ! array_key_exists( $key, $context ) ) { continue; }
			$value = $context[ $key ];
			if ( in_array( $key, [ 'payment_type', 'source' ], true ) ) { $out[ $key ] = sanitize_text_field( (string) $value ); }
			elseif ( $key === 'resend' ) { $out[ $key ] = ! empty( $value ) ? 1 : 0; }
			else { $out[ $key ] = absint( $value ); }
		}
		return $out;
	}

	private function ensure_table(): void {
		global $wpdb;
		$table = $wpdb->prefix . 'sn_message_deliveries';
		if ( $wpdb->get_var( $wpdb->prepare( 'SHOW TABLES LIKE %s', $table ) ) === $table ) { return; }
		require_once ABSPATH . 'wp-admin/includes/upgrade.php';
		$charset = $wpdb->get_charset_collate();
		dbDelta( "CREATE TABLE {$table} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			event_key VARCHAR(120) NOT NULL,
			entity_type VARCHAR(60) DEFAULT NULL,
			entity_id BIGINT UNSIGNED DEFAULT NULL,
			recipient_phone VARCHAR(20) NOT NULL,
			channel VARCHAR(20) NOT NULL,
			provider VARCHAR(40) NOT NULL,
			request_id VARCHAR(120) NOT NULL,
			external_message_id VARCHAR(191) DEFAULT NULL,
			status VARCHAR(30) NOT NULL DEFAULT 'pending',
			attempt_count INT UNSIGNED NOT NULL DEFAULT 0,
			retryable TINYINT(1) NOT NULL DEFAULT 0,
			error_code VARCHAR(40) DEFAULT NULL,
			error_message TEXT DEFAULT NULL,
			is_secure TINYINT(1) NOT NULL DEFAULT 0,
			metadata_json LONGTEXT DEFAULT NULL,
			sent_at DATETIME DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY channel_request (channel,request_id),
			KEY event_key (event_key),
			KEY entity_lookup (entity_type,entity_id),
			KEY recipient_phone (recipient_phone),
			KEY delivery_status (status,created_at)
		) {$charset};" );
	}

	private function delivery_by_request( string $channel, string $request_id ): ?array {
		$this->ensure_table(); global $wpdb; $table = $wpdb->prefix . 'sn_message_deliveries';
		$row = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$table} WHERE channel=%s AND request_id=%s LIMIT 1", $channel, $request_id ), ARRAY_A );
		return is_array( $row ) ? $row : null;
	}

	private function start_delivery( string $event, string $phone, string $request_id, array $context, bool $secure, ?array $prior ): int {
		$this->ensure_table(); global $wpdb; $table = $wpdb->prefix . 'sn_message_deliveries'; $now = current_time( 'mysql' );
		$data = [
			'event_key' => substr( sanitize_text_field( $event ), 0, 120 ),
			'entity_type' => sanitize_key( (string) ( $context['entity_type'] ?? '' ) ) ?: null,
			'entity_id' => absint( $context['entity_id'] ?? 0 ) ?: null,
			'recipient_phone' => $phone,
			'channel' => 'bale', 'provider' => 'gateway_safir', 'request_id' => $request_id,
			'status' => 'pending', 'attempt_count' => (int) ( $prior['attempt_count'] ?? 0 ) + 1,
			'retryable' => 0, 'error_code' => null, 'error_message' => null,
			'is_secure' => $secure ? 1 : 0,
			'metadata_json' => wp_json_encode( $this->metadata( $context ), JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ),
			'updated_at' => $now,
		];
		if ( $prior && ! empty( $prior['id'] ) ) { $wpdb->update( $table, $data, [ 'id' => (int) $prior['id'] ] ); return (int) $prior['id']; }
		$data['created_at'] = $now; $wpdb->insert( $table, $data ); return (int) $wpdb->insert_id;
	}

	private function finish_delivery( int $delivery_id, array $result ): void {
		if ( $delivery_id < 1 ) { return; }
		global $wpdb; $table = $wpdb->prefix . 'sn_message_deliveries'; $success = ! empty( $result['success'] );
		$wpdb->update( $table, [
			'external_message_id' => $success ? sanitize_text_field( (string) ( $result['message_id'] ?? '' ) ) : null,
			'status' => $success ? 'sent' : sanitize_key( (string) ( $result['status'] ?? 'failed' ) ),
			'retryable' => ! empty( $result['retryable'] ) ? 1 : 0,
			'error_code' => $success ? null : substr( sanitize_text_field( (string) ( $result['error_code'] ?? '' ) ), 0, 40 ),
			'error_message' => $success ? null : sanitize_text_field( (string) ( $result['error_message'] ?? '' ) ),
			'sent_at' => $success ? current_time( 'mysql' ) : null,
			'updated_at' => current_time( 'mysql' ),
		], [ 'id' => $delivery_id ] );
	}

	private function skipped( string $reason ): array { return [ 'success' => false, 'status' => 'skipped', 'message_id' => '', 'error_code' => $reason, 'error_message' => '', 'retryable' => false ]; }
	private function failed_local( $code, string $message ): array { return [ 'success' => false, 'status' => 'failed', 'message_id' => '', 'error_code' => $code, 'error_message' => $message, 'retryable' => false ]; }
	private function duplicate_result( array $row ): array { return [ 'success' => true, 'status' => 'duplicate', 'message_id' => (string) ( $row['external_message_id'] ?? '' ), 'error_code' => null, 'error_message' => '', 'retryable' => false, 'delivery_id' => (int) ( $row['id'] ?? 0 ), 'request_id' => (string) ( $row['request_id'] ?? '' ) ]; }
}
