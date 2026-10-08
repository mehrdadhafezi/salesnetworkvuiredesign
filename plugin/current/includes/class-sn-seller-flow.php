<?php
if ( ! defined( 'ABSPATH' ) ) { exit; }

/**
 * Stage-one seller workflow.
 *
 * The workflow is intentionally additive. Legacy leads and MIS distribution
 * items keep their original rows; this module stores a shared state/history
 * layer and only hides rows from the seller after a traceable archive event.
 */
final class SN_Seller_Flow {
	private const DB_VERSION = '2026-09-27-seller-flow-callback-v2';
	private const CRON_HOOK = 'sn_seller_flow_archive_due';
	private const MAX_NO_ANSWER_ATTEMPTS = 3;
	private const ARCHIVE_IDLE_DAYS = 3;

	private static ?self $instance = null;
	private bool $hooks_registered = false;
	private bool $schema_ready = false;

	public static function instance(): self {
		if ( null === self::$instance ) { self::$instance = new self(); }
		return self::$instance;
	}

	private function __construct() {}

	public static function activate(): void {
		$self = self::instance();
		$self->install_schema();
		$self->schedule_archive_job();
	}

	public static function deactivate(): void {
		wp_clear_scheduled_hook( self::CRON_HOOK );
	}

	public function register_hooks(): void {
		if ( $this->hooks_registered ) { return; }
		$this->hooks_registered = true;
		add_action( 'init', [ $this, 'maybe_upgrade' ], 7 );
		add_action( 'admin_init', [ $this, 'maybe_upgrade' ], 7 );
		add_action( self::CRON_HOOK, [ $this, 'archive_due' ] );
		add_action( 'wp_ajax_sn_seller_flow_no_answer_attempt', [ $this, 'ajax_no_answer_attempt' ] );
	}

	public function maybe_upgrade(): void {
		if ( (string) get_option( 'sn_seller_flow_db_version', '' ) !== self::DB_VERSION ) {
			$this->install_schema();
		}
		$this->schedule_archive_job();
	}

	private function tables(): array {
		global $wpdb;
		return [
			'states' => $wpdb->prefix . 'sn_seller_flow_states',
			'events' => $wpdb->prefix . 'sn_seller_flow_events',
		];
	}

	private function install_schema(): void {
		global $wpdb;
		require_once ABSPATH . 'wp-admin/includes/upgrade.php';
		$t = $this->tables();
		$charset = $wpdb->get_charset_collate();

		dbDelta( "CREATE TABLE {$t['states']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			source_kind VARCHAR(30) NOT NULL,
			source_id BIGINT UNSIGNED NOT NULL,
			seller_user_id BIGINT UNSIGNED NOT NULL,
			sales_manager_user_id BIGINT UNSIGNED DEFAULT NULL,
			flow_status VARCHAR(30) NOT NULL DEFAULT '',
			no_answer_attempts TINYINT UNSIGNED NOT NULL DEFAULT 0,
			third_no_answer_at DATETIME DEFAULT NULL,
			callback_at DATETIME DEFAULT NULL,
			not_purchase_reason TEXT DEFAULT NULL,
			last_activity_at DATETIME NOT NULL,
			archived_at DATETIME DEFAULT NULL,
			archive_reason VARCHAR(60) DEFAULT NULL,
			archive_error VARCHAR(100) DEFAULT NULL,
			last_archive_attempt_at DATETIME DEFAULT NULL,
			last_invoice_id BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY source_ref (source_kind,source_id),
			KEY seller_active (seller_user_id,archived_at),
			KEY manager_archive (sales_manager_user_id,archived_at),
			KEY archive_due (flow_status,no_answer_attempts,archived_at,last_activity_at)
		) {$charset};" );
		$state_columns = (array) $wpdb->get_col( "SHOW COLUMNS FROM {$t['states']}", 0 );
		if ( ! in_array( 'callback_at', $state_columns, true ) ) {
			$wpdb->query( "ALTER TABLE {$t['states']} ADD COLUMN callback_at DATETIME DEFAULT NULL AFTER third_no_answer_at" );
		}

		dbDelta( "CREATE TABLE {$t['events']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			state_id BIGINT UNSIGNED NOT NULL,
			source_kind VARCHAR(30) NOT NULL,
			source_id BIGINT UNSIGNED NOT NULL,
			seller_user_id BIGINT UNSIGNED NOT NULL,
			actor_user_id BIGINT UNSIGNED DEFAULT NULL,
			event_key VARCHAR(60) NOT NULL,
			request_token VARCHAR(64) DEFAULT NULL,
			previous_status VARCHAR(30) DEFAULT NULL,
			new_status VARCHAR(30) DEFAULT NULL,
			attempt_number TINYINT UNSIGNED DEFAULT NULL,
			reason TEXT DEFAULT NULL,
			invoice_id BIGINT UNSIGNED DEFAULT NULL,
			payload_json LONGTEXT DEFAULT NULL,
			created_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			KEY state_created (state_id,created_at),
			KEY source_created (source_kind,source_id,created_at),
			KEY seller_created (seller_user_id,created_at),
			KEY event_key (event_key),
			UNIQUE KEY request_token (request_token)
		) {$charset};" );

		$invoice_items = $wpdb->prefix . 'sn_invoice_items';
		if ( (string) $wpdb->get_var( $wpdb->prepare( 'SHOW TABLES LIKE %s', $invoice_items ) ) === $invoice_items ) {
			$columns = (array) $wpdb->get_col( "SHOW COLUMNS FROM {$invoice_items}", 0 );
			if ( ! in_array( 'product_type', $columns, true ) ) {
				$wpdb->query( "ALTER TABLE {$invoice_items} ADD COLUMN product_type VARCHAR(30) DEFAULT NULL AFTER product_name" );
			}
		}
		$this->ensure_status_catalog();

		update_option( 'sn_seller_flow_db_version', self::DB_VERSION, false );
		$this->schema_ready = true;
	}

	private function ensure_status_catalog(): void {
		global $wpdb;
		$table = $wpdb->prefix . 'sn_lead_statuses';
		if ( (string) $wpdb->get_var( $wpdb->prepare( 'SHOW TABLES LIKE %s', $table ) ) !== $table ) { return; }
		foreach ( self::statuses() as $status ) {
			$exists = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$table} WHERE label=%s LIMIT 1", (string) $status['label'] ) );
			if ( $exists > 0 ) { continue; }
			$wpdb->insert( $table, [
				'label' => (string) $status['label'],
				'color' => (string) $status['color'],
				'sort_order' => (int) $status['sort_order'],
				'is_active' => 1,
			] );
		}
	}

	private function ensure_schema(): bool {
		if ( $this->schema_ready ) { return true; }
		global $wpdb;
		$states = $this->tables()['states'];
		if ( (string) $wpdb->get_var( $wpdb->prepare( 'SHOW TABLES LIKE %s', $states ) ) !== $states ) {
			$this->install_schema();
		}
		$this->schema_ready = (string) $wpdb->get_var( $wpdb->prepare( 'SHOW TABLES LIKE %s', $states ) ) === $states;
		return $this->schema_ready;
	}

	private function schedule_archive_job(): void {
		if ( ! wp_next_scheduled( self::CRON_HOOK ) ) {
			wp_schedule_event( time() + 300, 'hourly', self::CRON_HOOK );
		}
	}

	public static function statuses(): array {
		return [
			'no_answer' => [ 'id' => 1001, 'label' => 'جواب نداده', 'color' => '#f59e0b', 'sort_order' => 1 ],
			'callback' => [ 'id' => 1002, 'label' => 'تماس مجدد', 'color' => '#3b82f6', 'sort_order' => 2 ],
			'duplicate' => [ 'id' => 1005, 'label' => 'تکراری', 'color' => '#64748b', 'sort_order' => 3 ],
			'not_purchased' => [ 'id' => 1003, 'label' => 'عدم خرید', 'color' => '#ef4444', 'sort_order' => 4 ],
			'pre_invoice' => [ 'id' => 1004, 'label' => 'پیش‌فاکتور', 'color' => '#0ea5e9', 'sort_order' => 5 ],
		];
	}

	public function statuses_for_ajax(): array {
		$rows = [];
		foreach ( self::statuses() as $slug => $status ) {
			$rows[] = $status + [
				'slug' => $slug,
				'destination_panel' => null,
				'move_to_destination' => 0,
			];
		}
		return $rows;
	}

	public function status_slug( string $label ): string {
		$label = sanitize_text_field( $label );
		$label = strtr( $label, [
			'ي' => 'ی', 'ى' => 'ی', 'ك' => 'ک',
			"\xE2\x80\x8C" => ' ', // ZWNJ
			"\xE2\x80\x8D" => ' ', // ZWJ
			'_' => ' ', '-' => ' ',
		] );
		$label = trim( preg_replace( '/\s+/u', ' ', $label ) );
		$aliases = [
			'no answer' => 'no_answer',
			'noanswer' => 'no_answer',
			'جواب نداده ها' => 'no_answer',
			'callback' => 'callback',
			'duplicate' => 'duplicate',
			'تکراری' => 'duplicate',
			'follow up' => 'callback',
			'تماس مجددها' => 'callback',
			'not purchased' => 'not_purchased',
			'عدم خریدها' => 'not_purchased',
			'pre invoice' => 'pre_invoice',
		];
		$alias_key = function_exists( 'mb_strtolower' ) ? mb_strtolower( $label, 'UTF-8' ) : strtolower( $label );
		if ( isset( $aliases[ $alias_key ] ) ) { return $aliases[ $alias_key ]; }
		if ( in_array( $label, [ 'پیش فاکتور', 'پیش فاکتور شده' ], true ) ) {
			return 'pre_invoice';
		}
		foreach ( self::statuses() as $slug => $status ) {
			$normalized_slug = str_replace( '_', ' ', $slug );
			$normalized_label = trim( preg_replace( '/\s+/u', ' ', str_replace( "\xE2\x80\x8C", ' ', (string) $status['label'] ) ) );
			if ( $alias_key === $slug || $alias_key === $normalized_slug || $label === $normalized_label ) { return $slug; }
		}
		return '';
	}

	public function status_label( string $slug ): string {
		$statuses = self::statuses();
		return isset( $statuses[ $slug ] ) ? (string) $statuses[ $slug ]['label'] : '';
	}

	public function validate_status_change( string $label, string $reason = '', bool $allow_pre_invoice = false ) {
		$slug = $this->status_slug( $label );
		if ( $slug === '' ) {
			return new WP_Error( 'sn_invalid_seller_status', 'وضعیت انتخاب‌شده معتبر نیست.' );
		}
		if ( $slug === 'not_purchased' && trim( $reason ) === '' ) {
			return new WP_Error( 'sn_not_purchase_reason_required', 'برای ثبت «عدم خرید»، واردکردن دلیل اجباری است.' );
		}
		if ( $slug === 'pre_invoice' && ! $allow_pre_invoice ) {
			return new WP_Error( 'sn_pre_invoice_requires_invoice', 'وضعیت «پیش‌فاکتور» فقط بعد از صدور موفق پیش‌فاکتور ثبت می‌شود.' );
		}
		return true;
	}

	private function normalize_source_kind( string $kind ): string {
		return in_array( $kind, [ 'v4_distribution_item', 'distribution_item' ], true ) ? 'v4_distribution_item' : 'legacy_lead';
	}

	public function get_state( string $source_kind, int $source_id ): ?array {
		if ( $source_id < 1 || ! $this->ensure_schema() ) { return null; }
		global $wpdb;
		$table = $this->tables()['states'];
		$row = $wpdb->get_row( $wpdb->prepare(
			"SELECT * FROM {$table} WHERE source_kind=%s AND source_id=%d LIMIT 1",
			$this->normalize_source_kind( $source_kind ),
			$source_id
		), ARRAY_A );
		return is_array( $row ) ? $row : null;
	}

	private function get_state_for_update( string $source_kind, int $source_id ): ?array {
		if ( $source_id < 1 || ! $this->ensure_schema() ) { return null; }
		global $wpdb;
		$table = $this->tables()['states'];
		$row = $wpdb->get_row( $wpdb->prepare(
			"SELECT * FROM {$table} WHERE source_kind=%s AND source_id=%d LIMIT 1 FOR UPDATE",
			$this->normalize_source_kind( $source_kind ),
			$source_id
		), ARRAY_A );
		return is_array( $row ) ? $row : null;
	}

	public function is_archived( string $source_kind, int $source_id ): bool {
		$state = $this->get_state( $source_kind, $source_id );
		return $state && ! empty( $state['archived_at'] );
	}

	/**
	 * Save workflow state. Callers that also update a source row should wrap both
	 * operations in the same database transaction.
	 *
	 * @return array|WP_Error
	 */
	public function apply_activity( array $args ) {
		if ( ! $this->ensure_schema() ) {
			return new WP_Error( 'sn_seller_flow_schema', 'ساختار مرحله فروشنده آماده نیست.' );
		}
		global $wpdb;
		$t = $this->tables();
		$source_kind = $this->normalize_source_kind( sanitize_key( (string) ( $args['source_kind'] ?? '' ) ) );
		$source_id = absint( $args['source_id'] ?? 0 );
		$seller_user_id = absint( $args['seller_user_id'] ?? 0 );
		$actor_user_id = absint( $args['actor_user_id'] ?? 0 );
		$status_changed = ! empty( $args['status_changed'] );
		$reason_changed = ! empty( $args['reason_changed'] );
		$force_attempt = ! empty( $args['force_no_answer_attempt'] );
		$invoice_id = absint( $args['invoice_id'] ?? 0 );
		$request_token = sanitize_key( (string) ( $args['request_token'] ?? '' ) );
		$changed_field = sanitize_key( (string) ( $args['changed_field'] ?? '' ) );
		$touch_activity = array_key_exists( 'touch_activity', $args )
			? ! empty( $args['touch_activity'] )
			: ! in_array( $changed_field, [ 'close_editor', 'leave_editor' ], true );
		$label = sanitize_text_field( (string) ( $args['status_label'] ?? '' ) );
		$reason = sanitize_textarea_field( (string) ( $args['not_purchase_reason'] ?? '' ) );
		$callback_input = sanitize_text_field( (string) ( $args['callback_at'] ?? '' ) );
		if ( $source_id < 1 || $seller_user_id < 1 ) {
			return new WP_Error( 'sn_seller_flow_source', 'مرجع یا فروشنده وضعیت نامعتبر است.' );
		}

		$current = $this->get_state_for_update( $source_kind, $source_id );
		if ( $request_token !== '' ) {
			$existing_event = $wpdb->get_row( $wpdb->prepare( "SELECT source_kind,source_id,seller_user_id FROM {$t['events']} WHERE request_token=%s LIMIT 1", $request_token ), ARRAY_A );
			if ( $existing_event ) {
				$same_request = (string) $existing_event['source_kind'] === $source_kind
					&& (int) $existing_event['source_id'] === $source_id
					&& (int) $existing_event['seller_user_id'] === $seller_user_id;
				if ( $same_request && $current ) { return $this->public_state( $current ) + [ 'idempotent' => true ]; }
				return new WP_Error( 'sn_seller_flow_token_conflict', 'شناسه ثبت تماس قبلاً برای درخواست دیگری استفاده شده است.' );
			}
		}
		if ( $current && ! empty( $current['archived_at'] ) ) {
			return new WP_Error( 'sn_seller_flow_archived', 'این شماره بایگانی شده و دیگر از پنل فروشنده قابل ویرایش نیست.' );
		}
		$previous_seller_user_id = (int) ( $current['seller_user_id'] ?? 0 );
		$owner_changed = $current && $previous_seller_user_id > 0 && $previous_seller_user_id !== $seller_user_id;
		$previous_status = (string) ( $current['flow_status'] ?? '' );
		$new_status = $previous_status;
		$attempts = (int) ( $current['no_answer_attempts'] ?? 0 );
		$third_no_answer_at = (string) ( $current['third_no_answer_at'] ?? '' );
		$stored_reason = (string) ( $current['not_purchase_reason'] ?? '' );
		$callback_at = (string) ( $current['callback_at'] ?? '' );
		$now = current_time( 'mysql' );
		if ( $owner_changed ) {
			// Source ownership has already been validated by the caller. A newly
			// assigned seller starts a fresh active state while the old events remain.
			$previous_status = '';
			$new_status = '';
			$attempts = 0;
			$third_no_answer_at = '';
			$stored_reason = '';
			$callback_at = '';
		}
		$inferred_status = $this->status_slug( $label );
		if ( ! $current && $inferred_status !== '' && ( ! $status_changed || $force_attempt ) ) {
			$previous_status = $inferred_status;
			$new_status = $inferred_status;
			if ( $inferred_status === 'no_answer' ) { $attempts = 1; }
			if ( $inferred_status === 'not_purchased' && $reason !== '' ) { $stored_reason = $reason; }
		}
		// Backward-safe repair: older releases could create a workflow-state row while
		// saving name/city/note, before a status was written.  Once such an empty row
		// existed, the raw source status was hidden from the status tabs forever.
		// Reconcile only an empty state; a non-empty workflow status remains authoritative.
		if ( $current && $new_status === '' && $inferred_status !== '' ) {
			$previous_status = '';
			$new_status = $inferred_status;
			if ( $inferred_status === 'no_answer' ) { $attempts = max( 1, $attempts ); }
			if ( $inferred_status === 'not_purchased' && $reason !== '' ) { $stored_reason = $reason; }
		}

		if ( $status_changed || $invoice_id > 0 ) {
			if ( $invoice_id > 0 ) { $label = 'پیش‌فاکتور'; }
			$valid = $this->validate_status_change( $label, $reason, $invoice_id > 0 );
			if ( is_wp_error( $valid ) ) { return $valid; }
			$new_status = $this->status_slug( $label );
			if ( $new_status === 'callback' ) {
				if ( ! preg_match( '/^(\d{4}\/\d{1,2}\/\d{1,2})\s+((?:[01]\d|2[0-3]):[0-5]\d)$/', SN_Helpers::to_english_nums( $callback_input ), $callback_parts ) ) {
					return new WP_Error( 'sn_callback_datetime_required', 'تاریخ و ساعت تماس مجدد را انتخاب کنید.' );
				}
				$parsed_callback = SN_Helpers::normalize_jalali_tehran_datetime( $callback_parts[1], $callback_parts[2], true );
				if ( is_wp_error( $parsed_callback ) ) { return $parsed_callback; }
				$callback_at = (string) $parsed_callback['mysql'];
			} else { $callback_at = ''; }
			if ( $new_status === 'no_answer' ) {
				if ( $force_attempt && $previous_status === 'no_answer' && $attempts >= self::MAX_NO_ANSWER_ATTEMPTS ) {
					return new WP_Error( 'sn_no_answer_attempt_limit', 'هر سه تماس بی‌پاسخ قبلاً ثبت شده است.' );
				}
				if ( $previous_status !== 'no_answer' ) {
					$attempts = 1;
				} elseif ( $force_attempt ) {
					$attempts = min( self::MAX_NO_ANSWER_ATTEMPTS, max( 1, $attempts ) + 1 );
				} else {
					$attempts = max( 1, $attempts );
				}
				if ( $attempts >= self::MAX_NO_ANSWER_ATTEMPTS && $third_no_answer_at === '' ) {
					$third_no_answer_at = $now;
				}
				$stored_reason = '';
			} else {
				$attempts = 0;
				$third_no_answer_at = '';
				$stored_reason = $new_status === 'not_purchased' ? $reason : '';
			}
		} elseif ( $reason_changed ) {
			if ( $previous_status !== 'not_purchased' ) {
				return new WP_Error( 'sn_reason_without_status', 'دلیل عدم خرید فقط برای وضعیت «عدم خرید» قابل ثبت است.' );
			}
			if ( trim( $reason ) === '' ) {
				return new WP_Error( 'sn_not_purchase_reason_required', 'دلیل عدم خرید نمی‌تواند خالی باشد.' );
			}
			$stored_reason = $reason;
		}
		if ( $new_status === 'not_purchased' && trim( $stored_reason ) === '' ) {
			return new WP_Error( 'sn_not_purchase_reason_required', 'برای ثبت «عدم خرید»، واردکردن دلیل اجباری است.' );
		}

		$activity_at = ( $touch_activity || ! $current ) ? $now : ( (string) ( $current['last_activity_at'] ?? '' ) ?: $now );
		$data = [
			'source_kind' => $source_kind,
			'source_id' => $source_id,
			'seller_user_id' => $seller_user_id,
			'flow_status' => $new_status,
			'no_answer_attempts' => min( self::MAX_NO_ANSWER_ATTEMPTS, max( 0, $attempts ) ),
			'third_no_answer_at' => $third_no_answer_at !== '' ? $third_no_answer_at : null,
			'callback_at' => $callback_at !== '' ? $callback_at : null,
			'not_purchase_reason' => $stored_reason !== '' ? $stored_reason : null,
			'last_activity_at' => $activity_at,
			'last_invoice_id' => $invoice_id > 0 ? $invoice_id : ( absint( $current['last_invoice_id'] ?? 0 ) ?: null ),
			'updated_at' => $now,
		];
		if ( $touch_activity || $status_changed || $invoice_id > 0 ) {
			$data['archive_error'] = null;
			$data['last_archive_attempt_at'] = null;
		}
		if ( $current ) {
			$ok = $wpdb->update( $t['states'], $data, [ 'id' => (int) $current['id'] ] );
			$state_id = (int) $current['id'];
		} else {
			$data['created_at'] = $now;
			$ok = $wpdb->insert( $t['states'], $data );
			$state_id = (int) $wpdb->insert_id;
		}
		if ( $ok === false || $state_id < 1 ) {
			error_log( 'SN seller flow state write failed for ' . $source_kind . ':' . $source_id . ': ' . sanitize_text_field( (string) $wpdb->last_error ) );
			return new WP_Error( 'sn_seller_flow_write', 'ذخیره وضعیت مرحله فروشنده انجام نشد؛ لطفاً دوباره تلاش کنید.' );
		}

		$event_key = '';
		if ( $invoice_id > 0 ) { $event_key = 'pre_invoice_created'; }
		elseif ( $force_attempt ) { $event_key = 'no_answer_attempt'; }
		elseif ( $status_changed && $previous_status !== $new_status ) { $event_key = 'status_changed'; }
		elseif ( $reason_changed ) { $event_key = 'not_purchase_reason_updated'; }
		elseif ( $owner_changed ) { $event_key = 'seller_reassigned'; }
		if ( $event_key !== '' ) {
			$event_ok = $wpdb->insert( $t['events'], [
				'state_id' => $state_id,
				'source_kind' => $source_kind,
				'source_id' => $source_id,
				'seller_user_id' => $seller_user_id,
				'actor_user_id' => $actor_user_id ?: null,
				'event_key' => $event_key,
				'request_token' => $request_token !== '' ? $request_token : null,
				'previous_status' => $previous_status !== '' ? $previous_status : null,
				'new_status' => $new_status !== '' ? $new_status : null,
				'attempt_number' => $new_status === 'no_answer' ? (int) $data['no_answer_attempts'] : null,
				'reason' => $new_status === 'callback' && $callback_at !== '' ? 'زمان تماس: ' . SN_Helpers::gregorian_to_jalali_date( $callback_at ) : ( $stored_reason !== '' ? $stored_reason : null ),
				'invoice_id' => $invoice_id ?: null,
				'payload_json' => wp_json_encode( [
					'changed_field' => $changed_field,
					'previous_seller_user_id' => $owner_changed ? $previous_seller_user_id : null,
				] ),
				'created_at' => $now,
			] );
			if ( $event_ok === false ) {
				error_log( 'SN seller flow event write failed for ' . $source_kind . ':' . $source_id . ': ' . sanitize_text_field( (string) $wpdb->last_error ) );
				return new WP_Error( 'sn_seller_flow_event', 'ثبت تاریخچه وضعیت انجام نشد؛ لطفاً دوباره تلاش کنید.' );
			}
		}

		$state = $this->get_state( $source_kind, $source_id );
		return $this->public_state( $state ?: $data );
	}

	private function public_state( array $state ): array {
		$status = (string) ( $state['flow_status'] ?? '' );
		$last_activity = (string) ( $state['last_activity_at'] ?? '' );
		$due_at = '';
		if ( $status === 'no_answer' && (int) ( $state['no_answer_attempts'] ?? 0 ) >= self::MAX_NO_ANSWER_ATTEMPTS && $last_activity !== '' ) {
			$last_activity_timestamp = SN_Helpers::site_mysql_timestamp( $last_activity, 0 );
			if ( $last_activity_timestamp > 0 ) {
				$due_at = SN_Helpers::site_mysql_from_timestamp( $last_activity_timestamp + ( self::ARCHIVE_IDLE_DAYS * DAY_IN_SECONDS ) );
			}
		}
		return [
			'seller_flow_status' => $status,
			'lead_status' => $this->status_label( $status ),
			'no_answer_attempts' => (int) ( $state['no_answer_attempts'] ?? 0 ),
			'no_answer_max_attempts' => self::MAX_NO_ANSWER_ATTEMPTS,
			'third_no_answer_at' => (string) ( $state['third_no_answer_at'] ?? '' ),
			'not_purchase_reason' => (string) ( $state['not_purchase_reason'] ?? '' ),
			'callback_at' => ! empty( $state['callback_at'] ) ? SN_Helpers::gregorian_to_jalali_date( (string) $state['callback_at'] ) : '',
			'last_activity_at' => $last_activity,
			'archive_due_at' => $due_at,
			'archived_at' => (string) ( $state['archived_at'] ?? '' ),
			'archive_error' => (string) ( $state['archive_error'] ?? '' ),
		];
	}

	public function enrich_and_filter_leads( array $leads, int $seller_user_id ): array {
		$this->maybe_archive_due( 50 );
		if ( ! $leads || ! $this->ensure_schema() ) { return $leads; }
		global $wpdb;
		$legacy_ids = [];
		$v4_ids = [];
		foreach ( $leads as $lead ) {
			if ( ! empty( $lead['distribution_item_id'] ) || (string) ( $lead['source_kind'] ?? '' ) === 'v4_distribution_item' ) {
				$v4_ids[] = absint( $lead['distribution_item_id'] ?? 0 );
			} else {
				$legacy_ids[] = absint( $lead['id'] ?? 0 );
			}
		}
		$clauses = [];
		$args = [];
		foreach ( [ 'legacy_lead' => array_filter( array_unique( $legacy_ids ) ), 'v4_distribution_item' => array_filter( array_unique( $v4_ids ) ) ] as $kind => $ids ) {
			if ( ! $ids ) { continue; }
			$clauses[] = '(source_kind=%s AND source_id IN (' . implode( ',', array_fill( 0, count( $ids ), '%d' ) ) . '))';
			$args[] = $kind;
			foreach ( $ids as $id ) { $args[] = $id; }
		}
		$state_map = [];
		$duplicate_actor_map = [];
		if ( $clauses ) {
			$table = $this->tables()['states'];
			$sql = "SELECT * FROM {$table} WHERE " . implode( ' OR ', $clauses );
			$rows = $wpdb->get_results( $wpdb->prepare( $sql, ...$args ), ARRAY_A ) ?: [];
			foreach ( $rows as $row ) { $state_map[ (string) $row['source_kind'] . ':' . (int) $row['source_id'] ] = $row; }
			$duplicate_state_ids = array_map( 'intval', array_column( array_filter( $rows, static fn( $row ) => (string) $row['flow_status'] === 'duplicate' ), 'id' ) );
			if ( $duplicate_state_ids ) {
				$events = $this->tables()['events'];
				$ids_sql = implode( ',', $duplicate_state_ids );
				$actors = $wpdb->get_results( "SELECT e.state_id,u.display_name,u.user_login FROM {$events} e INNER JOIN (SELECT state_id,MAX(id) id FROM {$events} WHERE state_id IN ({$ids_sql}) AND new_status='duplicate' GROUP BY state_id) latest ON latest.id=e.id LEFT JOIN {$wpdb->users} u ON u.ID=e.actor_user_id", ARRAY_A ) ?: [];
				foreach ( $actors as $actor ) {
					$duplicate_actor_map[ (int) $actor['state_id'] ] = (string) ( $actor['display_name'] ?: $actor['user_login'] ?: '' );
				}
			}
		}

		$result = [];
		foreach ( $leads as $lead ) {
			$is_v4 = ! empty( $lead['distribution_item_id'] ) || (string) ( $lead['source_kind'] ?? '' ) === 'v4_distribution_item';
			$kind = $is_v4 ? 'v4_distribution_item' : 'legacy_lead';
			$source_id = $is_v4 ? absint( $lead['distribution_item_id'] ?? 0 ) : absint( $lead['id'] ?? 0 );
			$state = $state_map[ $kind . ':' . $source_id ] ?? null;
			if ( $state && ! empty( $state['archived_at'] ) ) { continue; }
			if ( $state && (int) $state['seller_user_id'] !== $seller_user_id ) {
				// Ownership changed after the previous seller worked this source.
				// Do not leak that seller's outcome/reason into the new seller UI.
				$lead['lead_status'] = '';
				$lead['seller_flow_status'] = '';
				$lead['no_answer_attempts'] = 0;
				$lead['no_answer_max_attempts'] = self::MAX_NO_ANSWER_ATTEMPTS;
				$lead['not_purchase_reason'] = '';
				$lead['callback_at'] = '';
				$lead['archive_due_at'] = '';
				$lead['archive_error'] = '';
			} elseif ( $state ) {
				$lead = array_merge( $lead, $this->public_state( $state ) );
				if ( (string) $state['flow_status'] === 'duplicate' ) { $lead['duplicate_marked_by'] = (string) ( $duplicate_actor_map[ (int) $state['id'] ] ?? '' ); }
				// A blank historical state must not suppress a valid status already stored
				// on sn_leads/sn_distribution_items. This is read-only compatibility for
				// existing data and is persisted on the next ordinary edit.
				if ( (string) ( $lead['seller_flow_status'] ?? '' ) === '' ) {
					$mapped_slug = $this->status_slug( (string) ( $lead['lead_status'] ?? '' ) );
					if ( $mapped_slug !== '' ) {
						$lead['seller_flow_status'] = $mapped_slug;
						$lead['lead_status'] = $this->status_label( $mapped_slug );
						if ( $mapped_slug === 'no_answer' ) { $lead['no_answer_attempts'] = max( 1, (int) ( $lead['no_answer_attempts'] ?? 0 ) ); }
					}
				}
			} else {
				$legacy_status = (string) ( $lead['lead_status'] ?? '' );
				$mapped_slug = $this->status_slug( $legacy_status );
				if ( $mapped_slug !== '' ) { $lead['lead_status'] = $this->status_label( $mapped_slug ); }
				$lead['seller_flow_status'] = $mapped_slug;
				$lead['no_answer_attempts'] = $mapped_slug === 'no_answer' ? 1 : 0;
				$lead['no_answer_max_attempts'] = self::MAX_NO_ANSWER_ATTEMPTS;
				$lead['not_purchase_reason'] = '';
				$lead['callback_at'] = '';
				$lead['archive_due_at'] = '';
				$lead['archive_error'] = '';
			}
			$result[] = $lead;
		}
		return $result;
	}

	private function maybe_archive_due( int $limit ): int {
		$lock_key = 'sn_seller_flow_archive_check_lock';
		if ( get_transient( $lock_key ) ) { return 0; }
		set_transient( $lock_key, '1', MINUTE_IN_SECONDS );
		return $this->archive_due( $limit );
	}

	public function ajax_no_answer_attempt(): void {
		if ( ! is_user_logged_in() || ! check_ajax_referer( 'sn_public', 'nonce', false ) ) {
			SN_Helpers::send_json( false, 'دسترسی غیرمجاز' );
			return;
		}
		global $wpdb;
		$actor_id = get_current_user_id();
		if ( ! current_user_can( 'manage_options' ) && $this->user_position( $actor_id ) !== 'seller' ) {
			SN_Helpers::send_json( false, 'این عملیات فقط برای فروشنده مجاز است.' );
			return;
		}
		$kind = $this->normalize_source_kind( sanitize_key( wp_unslash( $_POST['source_kind'] ?? '' ) ) );
		$source_id = absint( $_POST['source_id'] ?? 0 );
		$distribution_id = absint( $_POST['distribution_item_id'] ?? 0 );
		$attempt_token = sanitize_key( wp_unslash( $_POST['attempt_token'] ?? '' ) );
		if ( $distribution_id > 0 ) { $kind = 'v4_distribution_item'; $source_id = $distribution_id; }
		if ( $source_id < 1 ) { SN_Helpers::send_json( false, 'شناسه شماره نامعتبر است.' ); return; }
		if ( strlen( $attempt_token ) < 12 ) { SN_Helpers::send_json( false, 'شناسه امن ثبت تماس نامعتبر است؛ صفحه را تازه‌سازی کنید.' ); return; }

		if ( $kind === 'v4_distribution_item' ) {
			$table = $wpdb->prefix . 'sn_distribution_items';
			$source = $wpdb->get_row( $wpdb->prepare( "SELECT id,current_owner_user_id FROM {$table} WHERE id=%d", $source_id ), ARRAY_A );
			$seller_id = absint( $source['current_owner_user_id'] ?? 0 );
			$where = [ 'id' => $source_id ];
			$source_data = [ 'seller_contact_status' => 'جواب نداده', 'seller_updated_by' => $actor_id, 'seller_updated_at' => current_time( 'mysql' ), 'updated_at' => current_time( 'mysql' ) ];
		} else {
			$table = $wpdb->prefix . 'sn_leads';
			$source = $wpdb->get_row( $wpdb->prepare( "SELECT id,seller_id FROM {$table} WHERE id=%d", $source_id ), ARRAY_A );
			$seller_id = absint( $source['seller_id'] ?? 0 );
			$where = [ 'id' => $source_id ];
			$source_data = [ 'lead_status' => 'جواب نداده', 'updated_at' => current_time( 'mysql' ) ];
		}
		if ( ! $source ) { SN_Helpers::send_json( false, 'شماره یافت نشد.' ); return; }
		if ( $seller_id !== $actor_id && ! current_user_can( 'manage_options' ) ) {
			SN_Helpers::send_json( false, 'این شماره در مالکیت فعلی شما نیست.' );
			return;
		}
		if ( $this->is_archived( $kind, $source_id ) ) {
			SN_Helpers::send_json( false, 'این شماره بایگانی شده است.' );
			return;
		}
		$source_data = $this->filter_existing_columns( $table, $source_data );
		if ( ! $source_data ) { SN_Helpers::send_json( false, 'ستون‌های ذخیره وضعیت این شماره آماده نیست.' ); return; }
		$wpdb->query( 'START TRANSACTION' );
		$updated = $wpdb->update( $table, $source_data, $where );
		if ( $updated === false ) {
			$wpdb->query( 'ROLLBACK' );
			SN_Helpers::send_json( false, 'ثبت تماس بی‌پاسخ انجام نشد.' );
			return;
		}
		$state = $this->apply_activity( [
			'source_kind' => $kind,
			'source_id' => $source_id,
			'seller_user_id' => $seller_id,
			'actor_user_id' => $actor_id,
			'status_label' => 'جواب نداده',
			'status_changed' => true,
			'force_no_answer_attempt' => true,
			'request_token' => $attempt_token,
			'changed_field' => 'no_answer_attempt',
		] );
		if ( is_wp_error( $state ) ) {
			$wpdb->query( 'ROLLBACK' );
			SN_Helpers::send_json( false, $state->get_error_message() );
			return;
		}
		// A network retry with the same token must not even refresh the source's
		// updated_at value, otherwise it would silently postpone auto-archiving.
		$wpdb->query( ! empty( $state['idempotent'] ) ? 'ROLLBACK' : 'COMMIT' );
		SN_Helpers::send_json( true, 'تماس بی‌پاسخ ثبت شد.', $state );
	}

	private function filter_existing_columns( string $table, array $data ): array {
		global $wpdb;
		$columns = (array) $wpdb->get_col( "SHOW COLUMNS FROM {$table}", 0 );
		return array_intersect_key( $data, array_flip( $columns ) );
	}

	public function archive_due( int $limit = 100 ): int {
		if ( ! $this->ensure_schema() ) { return 0; }
		global $wpdb;
		$t = $this->tables();
		$limit = max( 1, min( 500, $limit ) );
		$cutoff = SN_Helpers::site_mysql_from_timestamp( time() - ( self::ARCHIVE_IDLE_DAYS * DAY_IN_SECONDS ) );
		$rows = $wpdb->get_results( $wpdb->prepare(
			"SELECT * FROM {$t['states']} WHERE flow_status='no_answer' AND no_answer_attempts>=%d AND archived_at IS NULL AND last_activity_at<=%s ORDER BY last_activity_at ASC,id ASC LIMIT %d",
			self::MAX_NO_ANSWER_ATTEMPTS,
			$cutoff,
			$limit
		), ARRAY_A ) ?: [];
		$archived = 0;
		foreach ( $rows as $candidate ) {
			$wpdb->query( 'START TRANSACTION' );
			try {
				// Lock source first and shared state second. Seller writes use the
				// same order, preventing a stale archive from racing a new activity.
				$live = $this->live_source_snapshot( $candidate, true );
				$row = $this->get_state_for_update( (string) $candidate['source_kind'], (int) $candidate['source_id'] );
				if ( ! $row || ! empty( $row['archived_at'] ) || (string) $row['flow_status'] !== 'no_answer' || (int) $row['no_answer_attempts'] < self::MAX_NO_ANSWER_ATTEMPTS || (string) $row['last_activity_at'] > $cutoff ) {
					$wpdb->query( 'COMMIT' );
					continue;
				}

				$now = current_time( 'mysql' );
				if ( ! $live['found'] ) {
					$this->record_archive_error( (int) $row['id'], 'source_not_found', $now );
					$wpdb->query( 'COMMIT' );
					continue;
				}
				$live_owner_id = (int) $live['seller_user_id'];
				if ( $live_owner_id < 1 ) {
					$this->record_archive_error( (int) $row['id'], 'source_owner_not_found', $now );
					$wpdb->query( 'COMMIT' );
					continue;
				}
				if ( $live_owner_id !== (int) $row['seller_user_id'] ) {
					// Assignment may happen through older HR/MIS paths that do not know
					// about this additive state table. Never archive a newly assigned
					// seller's number using the previous seller's inactivity clock.
					$updated = $wpdb->update( $t['states'], [
						'seller_user_id' => $live_owner_id,
						'sales_manager_user_id' => null,
						'flow_status' => '',
						'no_answer_attempts' => 0,
						'third_no_answer_at' => null,
						'not_purchase_reason' => null,
						'last_activity_at' => $now,
						'archive_reason' => null,
						'archive_error' => null,
						'last_archive_attempt_at' => null,
						'updated_at' => $now,
					], [ 'id' => (int) $row['id'], 'archived_at' => null ] );
					$event_ok = $updated === false ? false : $wpdb->insert( $t['events'], [
						'state_id' => (int) $row['id'],
						'source_kind' => (string) $row['source_kind'],
						'source_id' => (int) $row['source_id'],
						'seller_user_id' => $live_owner_id,
						'event_key' => 'seller_reassigned_detected',
						'previous_status' => 'no_answer',
						'payload_json' => wp_json_encode( [ 'previous_seller_user_id' => (int) $row['seller_user_id'] ] ),
						'created_at' => $now,
					] );
					if ( $updated === false || $event_ok === false ) { throw new RuntimeException( 'seller reassignment reconciliation failed' ); }
					$wpdb->query( 'COMMIT' );
					continue;
				}

				$live_status = $this->status_slug( (string) $live['status_label'] );
				if ( $live_status !== 'no_answer' ) {
					// A compatible legacy path may have changed the source after the
					// shared state was written. Reconcile instead of archiving stale data.
					$reconciled_status = $live_status === 'not_purchased' ? '' : $live_status;
					$updated = $wpdb->update( $t['states'], [
						'flow_status' => $reconciled_status,
						'no_answer_attempts' => 0,
						'third_no_answer_at' => null,
						'not_purchase_reason' => null,
						'last_activity_at' => $now,
						'archive_error' => null,
						'last_archive_attempt_at' => null,
						'updated_at' => $now,
					], [ 'id' => (int) $row['id'], 'archived_at' => null ] );
					$event_ok = $updated === false ? false : $wpdb->insert( $t['events'], [
						'state_id' => (int) $row['id'],
						'source_kind' => (string) $row['source_kind'],
						'source_id' => (int) $row['source_id'],
						'seller_user_id' => (int) $row['seller_user_id'],
						'event_key' => 'source_status_reconciled',
						'previous_status' => 'no_answer',
						'new_status' => $reconciled_status !== '' ? $reconciled_status : null,
						'payload_json' => wp_json_encode( [ 'source_status_label' => (string) $live['status_label'] ] ),
						'created_at' => $now,
					] );
					if ( $updated === false || $event_ok === false ) { throw new RuntimeException( 'source status reconciliation failed' ); }
					$wpdb->query( 'COMMIT' );
					continue;
				}

				$live_activity_at = (string) $live['activity_at'];
				if ( $live_activity_at !== '' && $live_activity_at > (string) $row['last_activity_at'] ) {
					$updated = $wpdb->update( $t['states'], [
						'last_activity_at' => $live_activity_at,
						'archive_error' => null,
						'last_archive_attempt_at' => null,
						'updated_at' => $now,
					], [ 'id' => (int) $row['id'], 'archived_at' => null ] );
					if ( $updated === false ) { throw new RuntimeException( 'source activity reconciliation failed' ); }
					$wpdb->query( 'COMMIT' );
					continue;
				}

				$manager_id = $this->resolve_sales_manager( (int) $row['seller_user_id'], $row );
				if ( $manager_id < 1 ) {
					$this->record_archive_error( (int) $row['id'], 'sales_manager_not_found', $now );
					$wpdb->query( 'COMMIT' );
					continue;
				}
				$updated = $wpdb->query( $wpdb->prepare(
					"UPDATE {$t['states']} SET sales_manager_user_id=%d,archived_at=%s,archive_reason='no_answer',archive_error=NULL,last_archive_attempt_at=%s,updated_at=%s WHERE id=%d AND archived_at IS NULL",
					$manager_id,
					$now,
					$now,
					$now,
					(int) $row['id']
				) );
				$event_ok = $updated ? $wpdb->insert( $t['events'], [
					'state_id' => (int) $row['id'],
					'source_kind' => (string) $row['source_kind'],
					'source_id' => (int) $row['source_id'],
					'seller_user_id' => (int) $row['seller_user_id'],
					'event_key' => 'auto_archived_no_answer',
					'previous_status' => 'no_answer',
					'new_status' => 'no_answer',
					'attempt_number' => (int) $row['no_answer_attempts'],
					'payload_json' => wp_json_encode( [ 'manager_user_id' => $manager_id, 'idle_days' => self::ARCHIVE_IDLE_DAYS ] ),
					'created_at' => $now,
				] ) : false;
				if ( ! $updated || $event_ok === false ) { throw new RuntimeException( 'automatic archive transaction failed' ); }
				$wpdb->query( 'COMMIT' );
				$archived++;
			} catch ( Throwable $e ) {
				$wpdb->query( 'ROLLBACK' );
				error_log( 'SN seller flow archive failed for state ' . (int) ( $candidate['id'] ?? 0 ) . ': ' . $e->getMessage() );
			}
		}
		return $archived;
	}

	private function record_archive_error( int $state_id, string $error, string $now ): void {
		global $wpdb;
		$wpdb->update( $this->tables()['states'], [
			'archive_error' => sanitize_key( $error ),
			'last_archive_attempt_at' => $now,
			'updated_at' => $now,
		], [ 'id' => $state_id, 'archived_at' => null ] );
	}

	private function live_source_snapshot( array $state, bool $for_update = false ): array {
		global $wpdb;
		$source_id = absint( $state['source_id'] ?? 0 );
		$lock = $for_update ? ' FOR UPDATE' : '';
		if ( $source_id < 1 ) {
			return [ 'found' => false, 'seller_user_id' => 0, 'status_label' => '', 'activity_at' => '' ];
		}
		if ( (string) ( $state['source_kind'] ?? '' ) === 'legacy_lead' ) {
			$row = $wpdb->get_row( $wpdb->prepare(
				"SELECT seller_id,lead_status,updated_at FROM {$wpdb->prefix}sn_leads WHERE id=%d LIMIT 1{$lock}",
				$source_id
			), ARRAY_A );
			return [
				'found' => is_array( $row ),
				'seller_user_id' => absint( $row['seller_id'] ?? 0 ),
				'status_label' => (string) ( $row['lead_status'] ?? '' ),
				'activity_at' => (string) ( $row['updated_at'] ?? '' ),
			];
		}
		$row = $wpdb->get_row( $wpdb->prepare(
			"SELECT current_owner_user_id,seller_contact_status,seller_updated_at,updated_at FROM {$wpdb->prefix}sn_distribution_items WHERE id=%d LIMIT 1{$lock}",
			$source_id
		), ARRAY_A );
		return [
			'found' => is_array( $row ),
			'seller_user_id' => absint( $row['current_owner_user_id'] ?? 0 ),
			'status_label' => (string) ( $row['seller_contact_status'] ?? '' ),
			'activity_at' => (string) ( ( $row['seller_updated_at'] ?? '' ) ?: ( $row['updated_at'] ?? '' ) ),
		];
	}

	private function resolve_sales_manager( int $seller_user_id, array $state = [] ): int {
		$starts = [ $seller_user_id ];
		if ( (string) ( $state['source_kind'] ?? '' ) === 'legacy_lead' ) {
			global $wpdb;
			$supervisor_id = (int) $wpdb->get_var( $wpdb->prepare( "SELECT supervisor_id FROM {$wpdb->prefix}sn_leads WHERE id=%d", (int) ( $state['source_id'] ?? 0 ) ) );
			if ( $supervisor_id > 0 ) { $starts[] = $supervisor_id; }
		}
		$meta_supervisor = absint( get_user_meta( $seller_user_id, 'sn_supervisor_id', true ) );
		if ( $meta_supervisor > 0 ) { $starts[] = $meta_supervisor; }
		foreach ( array_unique( array_filter( $starts ) ) as $start_id ) {
			$resolved = $this->resolve_manager_from_ancestor_chain( (int) $start_id );
			if ( $resolved > 0 ) { return $resolved; }
			$resolved = $this->resolve_manager_from_legacy_meta_chain( (int) $start_id );
			if ( $resolved > 0 ) { return $resolved; }
		}
		foreach ( [ $seller_user_id, $meta_supervisor ] as $user_id ) {
			if ( $user_id < 1 ) { continue; }
			$manager_id = absint( get_user_meta( $user_id, 'sn_sales_manager_id', true ) );
			if ( $manager_id > 0 && $this->user_position( $manager_id ) === 'sales_manager' ) { return $manager_id; }
		}
		return 0;
	}

	/** Public, read-only hierarchy resolver used by the additive Dot archive. */
	public function sales_manager_for_user( int $user_id ): int {
		return $this->resolve_sales_manager( $user_id );
	}

	/** Hide the seller source when its assessment invoice remains unpaid. */
	public function mark_assessment_invoice_unpaid( int $invoice_id, int $manager_user_id, string $archived_at ): bool {
		return $this->mark_invoice_unpaid( $invoice_id, $manager_user_id, $archived_at, 'assessment_unpaid' );
	}

	public function mark_invoice_unpaid( int $invoice_id, int $manager_user_id, string $archived_at, string $reason ): bool {
		if ( $invoice_id < 1 || $manager_user_id < 1 || ! $this->ensure_schema() ) { return true; }
		global $wpdb;
		$table = $this->tables()['states'];
		$reason = sanitize_key( $reason );
		$result = $wpdb->query( $wpdb->prepare(
			"UPDATE {$table} SET sales_manager_user_id=%d,archived_at=%s,archive_reason=%s,archive_error=NULL,last_archive_attempt_at=%s,updated_at=%s WHERE last_invoice_id=%d AND archived_at IS NULL",
			$manager_user_id,
			$archived_at,
			$reason,
			$archived_at,
			$archived_at,
			$invoice_id
		) );
		return $result !== false;
	}

	/** Restore only the archive created by the assessment non-payment worker. */
	public function resolve_assessment_invoice_unpaid( int $invoice_id ): bool {
		return $this->resolve_invoice_unpaid( $invoice_id, 'assessment_unpaid' );
	}

	public function resolve_invoice_unpaid( int $invoice_id, string $reason ): bool {
		if ( $invoice_id < 1 || ! $this->ensure_schema() ) { return true; }
		global $wpdb;
		$now = current_time( 'mysql' );
		$table = $this->tables()['states'];
		$result = $wpdb->query( $wpdb->prepare(
			"UPDATE {$table} SET archived_at=NULL,archive_reason=NULL,archive_error=NULL,last_archive_attempt_at=NULL,last_activity_at=%s,updated_at=%s WHERE last_invoice_id=%d AND archive_reason=%s",
			$now,
			$now,
			$invoice_id,
			$reason
		) );
		return $result !== false;
	}

	private function resolve_manager_from_legacy_meta_chain( int $start_user_id ): int {
		$current = $start_user_id;
		$seen = [];
		for ( $depth = 0; $depth < 20 && $current > 0; $depth++ ) {
			if ( isset( $seen[ $current ] ) ) { break; }
			$seen[ $current ] = true;
			if ( $this->user_position( $current ) === 'sales_manager' ) { return $current; }
			$manager_id = absint( get_user_meta( $current, 'sn_sales_manager_id', true ) );
			if ( $manager_id > 0 && $this->user_position( $manager_id ) === 'sales_manager' ) { return $manager_id; }
			$current = absint( get_user_meta( $current, 'sn_supervisor_id', true ) );
		}
		return 0;
	}

	private function resolve_manager_from_ancestor_chain( int $start_user_id ): int {
		global $wpdb;
		$current_user_id = $start_user_id;
		$seen_profiles = [];
		for ( $depth = 0; $depth < 20 && $current_user_id > 0; $depth++ ) {
			if ( $this->user_position( $current_user_id ) === 'sales_manager' ) { return $current_user_id; }
			$profile_id = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$wpdb->prefix}sn_hr_profiles WHERE user_id=%d ORDER BY is_active DESC,id DESC LIMIT 1", $current_user_id ) );
			if ( $profile_id < 1 || isset( $seen_profiles[ $profile_id ] ) ) { break; }
			$seen_profiles[ $profile_id ] = true;
			$parent = $wpdb->get_row( $wpdb->prepare(
				"SELECT p.id,p.user_id,pos.slug position_slug FROM {$wpdb->prefix}sn_hr_assignments a INNER JOIN {$wpdb->prefix}sn_hr_profiles p ON p.id=a.parent_profile_id LEFT JOIN {$wpdb->prefix}sn_hr_positions pos ON pos.id=p.position_id WHERE a.child_profile_id=%d AND a.is_current=1 AND (a.relationship_type='reports_to' OR a.relationship_type='' OR a.relationship_type IS NULL) ORDER BY a.id DESC LIMIT 1",
				$profile_id
			), ARRAY_A );
			if ( ! $parent ) { break; }
			if ( (string) ( $parent['position_slug'] ?? '' ) === 'sales_manager' ) { return (int) $parent['user_id']; }
			$current_user_id = (int) $parent['user_id'];
		}
		return 0;
	}

	private function user_position( int $user_id ): string {
		global $wpdb;
		$position = (string) $wpdb->get_var( $wpdb->prepare(
			"SELECT pos.slug FROM {$wpdb->prefix}sn_hr_profiles p LEFT JOIN {$wpdb->prefix}sn_hr_positions pos ON pos.id=p.position_id WHERE p.user_id=%d ORDER BY p.is_active DESC,p.id DESC LIMIT 1",
			$user_id
		) );
		if ( $position !== '' ) { return $position; }
		$user = get_user_by( 'id', $user_id );
		$roles = $user instanceof WP_User ? (array) $user->roles : [];
		if ( in_array( 'sn_sales_manager', $roles, true ) ) { return 'sales_manager'; }
		if ( in_array( 'sn_supervisor', $roles, true ) ) { return 'supervisor'; }
		if ( in_array( 'sn_seller', $roles, true ) ) { return 'seller'; }
		return '';
	}

	public function render_manager_archives( int $manager_user_id, bool $show_all = false ): void {
		// Keep the lazy archive tab responsive. Hourly workers continue the backlog;
		// this visit-time probe only catches a small bounded batch.
		$this->maybe_archive_due( 5 );
		$payment_counts = [ 'assessment_unpaid' => 0, 'subscription_unpaid' => 0, 'product_unpaid' => 0, 'product_star_unpaid' => 0 ];
		if ( class_exists( 'SN_Dot_Flow' ) ) {
			SN_Dot_Flow::instance()->maybe_process_unpaid_archives( 5 );
			$payment_counts = array_merge( $payment_counts, SN_Dot_Flow::instance()->manager_archive_counts( $manager_user_id, $show_all ) );
		}
		global $wpdb;
		$t = $this->tables();
		$where = "archive_reason='no_answer' AND archived_at IS NOT NULL";
		$args = [];
		if ( ! $show_all ) { $where .= ' AND sales_manager_user_id=%d'; $args[] = $manager_user_id; }
		$sql = "SELECT * FROM {$t['states']} WHERE {$where} ORDER BY archived_at DESC,id DESC LIMIT 200";
		$count_sql = "SELECT COUNT(*) FROM {$t['states']} WHERE {$where}";
		$total_archived = $args ? (int) $wpdb->get_var( $wpdb->prepare( $count_sql, ...$args ) ) : (int) $wpdb->get_var( $count_sql );
		$states = $args ? ( $wpdb->get_results( $wpdb->prepare( $sql, ...$args ), ARRAY_A ) ?: [] ) : ( $wpdb->get_results( $sql, ARRAY_A ) ?: [] );
		if ( $states && function_exists( 'cache_users' ) ) { cache_users( array_values( array_unique( array_map( 'intval', array_column( $states, 'seller_user_id' ) ) ) ) ); }
		$source_summaries = $this->archive_source_summaries( $states );
		?>
		<div class="sn-card sn-section sn-manager-archives" dir="rtl">
			<div class="sn-card-head"><div><h3>بایگانی‌ها</h3><p class="sn-note">بایگانی‌های عملیاتی بدون حذف فاکتور، پرداخت یا سابقه تماس نگهداری می‌شوند.</p></div></div>
			<div class="sn-subtabs">
				<button type="button" class="sn-btn sn-btn-primary sn-btn-sm" aria-selected="true" data-sn-manager-archive-tab="no_answer">جواب نداده <span class="sn-tab-badge is-visible"><?php echo esc_html( number_format_i18n( $total_archived ) ); ?></span></button>
				<button type="button" class="sn-btn sn-btn-secondary sn-btn-sm" aria-selected="false" data-sn-manager-archive-tab="assessment_unpaid">عدم پرداخت اعتبارسنجی <span class="sn-tab-badge is-visible"><?php echo esc_html( number_format_i18n( (int) $payment_counts['assessment_unpaid'] ) ); ?></span></button>
				<button type="button" class="sn-btn sn-btn-secondary sn-btn-sm" aria-selected="false" data-sn-manager-archive-tab="subscription_unpaid">عدم پرداخت اشتراک <span class="sn-tab-badge is-visible"><?php echo esc_html( number_format_i18n( (int) $payment_counts['subscription_unpaid'] ) ); ?></span></button>
				<button type="button" class="sn-btn sn-btn-secondary sn-btn-sm" aria-selected="false" data-sn-manager-archive-tab="product_unpaid">عدم پرداخت محصول <span class="sn-tab-badge is-visible"><?php echo esc_html( number_format_i18n( (int) $payment_counts['product_unpaid'] ) ); ?></span></button>
				<button type="button" class="sn-btn sn-btn-secondary sn-btn-sm" aria-selected="false" data-sn-manager-archive-tab="product_star_unpaid">عدم پرداخت محصول* <span class="sn-tab-badge is-visible"><?php echo esc_html( number_format_i18n( (int) $payment_counts['product_star_unpaid'] ) ); ?></span></button>
			</div>
			<div data-sn-manager-archive-panel="no_answer">
			<p class="sn-note">شماره‌های سه‌بار بی‌پاسخ که پس از تلاش سوم، سه روز فعالیتی نداشته‌اند.</p>
			<div class="sn-table-wrap">
				<table class="sn-table"><thead><tr><th>شماره / مشتری</th><th>فروشنده</th><th>منبع</th><th>تلاش</th><th>آخرین فعالیت</th><th>بایگانی</th></tr></thead><tbody>
				<?php if ( ! $states ) : ?><tr><td colspan="6"><div class="sn-empty-state">موردی در بایگانی «جواب نداده» نیست.</div></td></tr><?php endif; ?>
				<?php foreach ( $states as $state ) : $source = $source_summaries[ (string) $state['source_kind'] . ':' . (int) $state['source_id'] ] ?? [ 'phone' => '', 'customer_name' => '', 'source_label' => 'منبع نامشخص' ]; ?>
				<tr>
					<td><strong><?php echo esc_html( $source['customer_name'] ?: 'بدون نام' ); ?></strong><br><code><?php echo esc_html( $source['phone'] ?: '—' ); ?></code></td>
					<td><?php echo esc_html( $this->user_label( (int) $state['seller_user_id'] ) ); ?></td>
					<td><?php echo esc_html( (string) $source['source_label'] ); ?> <span class="sn-muted">#<?php echo esc_html( (string) (int) $state['source_id'] ); ?></span></td>
					<td><strong><?php echo esc_html( number_format_i18n( (int) $state['no_answer_attempts'] ) ); ?> از ۳</strong></td>
					<td><?php echo esc_html( $this->date_label( (string) $state['last_activity_at'] ) ); ?></td>
					<td><?php echo esc_html( $this->date_label( (string) $state['archived_at'] ) ); ?></td>
				</tr>
				<?php endforeach; ?>
				</tbody></table>
			</div>
			<?php if ( $total_archived > 200 ) : ?><p class="sn-note">۲۰۰ مورد آخر از <?php echo esc_html( number_format_i18n( $total_archived ) ); ?> مورد نمایش داده شده است.</p><?php endif; ?>
			</div>
			<div data-sn-manager-archive-panel="assessment_unpaid" hidden>
				<?php if ( class_exists( 'SN_Dot_Flow' ) ) { SN_Dot_Flow::instance()->render_manager_payment_archive_tab( $manager_user_id, 'assessment_unpaid', $show_all ); } ?>
			</div>
			<div data-sn-manager-archive-panel="subscription_unpaid" hidden>
				<?php if ( class_exists( 'SN_Dot_Flow' ) ) { SN_Dot_Flow::instance()->render_manager_payment_archive_tab( $manager_user_id, 'subscription_unpaid', $show_all ); } ?>
			</div>
			<div data-sn-manager-archive-panel="product_unpaid" hidden>
				<?php if ( class_exists( 'SN_Dot_Flow' ) ) { SN_Dot_Flow::instance()->render_manager_payment_archive_tab( $manager_user_id, 'product_unpaid', $show_all ); } ?>
			</div>
			<div data-sn-manager-archive-panel="product_star_unpaid" hidden>
				<?php if ( class_exists( 'SN_Dot_Flow' ) ) { SN_Dot_Flow::instance()->render_manager_payment_archive_tab( $manager_user_id, 'product_star_unpaid', $show_all ); } ?>
			</div>
		</div>
		<?php
	}

	private function archive_rows_by_id( string $table, string $columns, array $ids ): array {
		global $wpdb;
		$ids = array_values( array_unique( array_filter( array_map( 'absint', $ids ) ) ) );
		if ( ! $ids ) { return []; }
		// Legacy installations may not have every historical source table. An
		// absent source must render as "unknown" instead of breaking all archives.
		$found_table = (string) $wpdb->get_var( $wpdb->prepare( 'SHOW TABLES LIKE %s', $table ) );
		if ( $found_table !== $table ) { return []; }
		$rows = $wpdb->get_results( "SELECT {$columns} FROM {$table} WHERE id IN (" . implode( ',', $ids ) . ')', ARRAY_A ) ?: [];
		$map = [];
		foreach ( $rows as $row ) {
			$map[ (int) $row['id'] ] = $row;
		}
		return $map;
	}

	private function archive_source_summaries( array $states ): array {
		global $wpdb;
		$legacy_ids = [];
		$distribution_ids = [];
		foreach ( $states as $state ) {
			if ( (string) ( $state['source_kind'] ?? '' ) === 'legacy_lead' ) { $legacy_ids[] = absint( $state['source_id'] ?? 0 ); }
			else { $distribution_ids[] = absint( $state['source_id'] ?? 0 ); }
		}
		$legacy = $this->archive_rows_by_id( $wpdb->prefix . 'sn_leads', 'id,phone,customer_name', $legacy_ids );
		$items = $this->archive_rows_by_id( $wpdb->prefix . 'sn_distribution_items', 'id,source_type,source_id,pool_item_id,seller_customer_name', $distribution_ids );

		$pool_ids = [];
		$data_row_ids = [];
		$invoice_ids = [];
		foreach ( $items as $item ) {
			$source_type = (string) ( $item['source_type'] ?? '' );
			$pool_id = absint( $item['pool_item_id'] ?? 0 );
			if ( $pool_id < 1 && $source_type === 'mis_pool' ) { $pool_id = absint( $item['source_id'] ?? 0 ); }
			if ( $pool_id > 0 ) { $pool_ids[] = $pool_id; }
			elseif ( $source_type === 'mis_data_row' ) { $data_row_ids[] = absint( $item['source_id'] ?? 0 ); }
			elseif ( $source_type === 'paid_invoice_referral' ) { $invoice_ids[] = absint( $item['source_id'] ?? 0 ); }
		}
		$pools = $this->archive_rows_by_id( $wpdb->prefix . 'sn_mis_lead_pool', 'id,customer_phone,normalized_phone,customer_name', $pool_ids );
		$data_rows = $this->archive_rows_by_id( $wpdb->prefix . 'sn_mis_data_rows', 'id,customer_phone,normalized_phone,customer_name', $data_row_ids );
		$invoices = $this->archive_rows_by_id( $wpdb->prefix . 'sn_invoices', 'id,customer_phone,customer_name', $invoice_ids );

		$summaries = [];
		foreach ( $states as $state ) {
			$source_kind = (string) ( $state['source_kind'] ?? '' );
			$source_id = absint( $state['source_id'] ?? 0 );
			$key = $source_kind . ':' . $source_id;
			if ( $source_kind === 'legacy_lead' ) {
				$row = $legacy[ $source_id ] ?? [];
				$summaries[ $key ] = [
					'phone' => (string) ( $row['phone'] ?? '' ),
					'customer_name' => (string) ( $row['customer_name'] ?? '' ),
					'source_label' => 'سرنخ قدیمی',
				];
				continue;
			}
			$item = $items[ $source_id ] ?? [];
			$source_type = (string) ( $item['source_type'] ?? '' );
			$source_row = [];
			$pool_id = absint( $item['pool_item_id'] ?? 0 );
			if ( $pool_id < 1 && $source_type === 'mis_pool' ) { $pool_id = absint( $item['source_id'] ?? 0 ); }
			if ( $pool_id > 0 ) { $source_row = $pools[ $pool_id ] ?? []; }
			elseif ( $source_type === 'mis_data_row' ) { $source_row = $data_rows[ absint( $item['source_id'] ?? 0 ) ] ?? []; }
			elseif ( $source_type === 'paid_invoice_referral' ) { $source_row = $invoices[ absint( $item['source_id'] ?? 0 ) ] ?? []; }
			$summaries[ $key ] = [
				'phone' => (string) ( ( $source_row['normalized_phone'] ?? '' ) ?: ( $source_row['customer_phone'] ?? '' ) ),
				'customer_name' => (string) ( ( $item['seller_customer_name'] ?? '' ) ?: ( $source_row['customer_name'] ?? '' ) ),
				'source_label' => 'توزیع MIS',
			];
		}
		return $summaries;
	}

	private function user_label( int $user_id ): string {
		$user = get_user_by( 'id', $user_id );
		return $user instanceof WP_User ? (string) ( $user->display_name ?: $user->user_login ) : '#' . $user_id;
	}

	private function date_label( string $mysql ): string {
		if ( $mysql === '' ) { return '—'; }
		return SN_Helpers::gregorian_to_jalali_date( $mysql );
	}

	public static function product_types(): array {
		return [
			'assessment' => 'اعتبارسنجی',
			'subscription' => 'اشتراک',
			'product' => 'محصول عادی',
			'product_star' => 'محصول*',
		];
	}

	public function product_type( int $product_id ): string {
		$explicit = sanitize_key( (string) get_post_meta( $product_id, '_sn_sales_product_type', true ) );
		// `subscription_star` was the temporary stage-one key. Keep reading it as
		// Product* so existing product metadata and invoice snapshots stay valid.
		if ( $explicit === 'subscription_star' ) { return 'product_star'; }
		// Project mapping is authoritative: a WooCommerce product may be both a
		// subscription content and a Product* sale, so an accidental manual
		// "محصول عادی" flag must not break Biavin routing.
		if ( class_exists( 'SN_Projects' ) && SN_Projects::instance()->is_content_product( $product_id ) ) { return 'product_star'; }
		if ( class_exists( 'SN_Projects' ) && SN_Projects::instance()->is_subscription_product( $product_id ) ) { return 'subscription'; }
		if ( class_exists( 'SN_Dot_Flow' ) && SN_Dot_Flow::instance()->is_assessment_product( $product_id ) ) { return 'assessment'; }
		if ( isset( self::product_types()[ $explicit ] ) ) { return $explicit; }
		return 'product';
	}

	public function decorate_products( array $products ): array {
		foreach ( $products as &$product ) {
			$type = $this->product_type( absint( $product['id'] ?? 0 ) );
			$product['sales_product_type'] = $type;
			$product['sales_product_type_label'] = (string) ( self::product_types()[ $type ] ?? '' );
		}
		unset( $product );
		return $products;
	}

	/** Missing metadata preserves visibility for existing Product* definitions. */
	public function standalone_sale_visible( int $product_id ): bool {
		return $this->product_type( $product_id ) !== 'product_star'
			|| (string) get_post_meta( $product_id, '_sn_product_star_standalone_visible', true ) !== '0';
	}

	public function validate_invoice_product_types( array $product_ids, array $quantities = [] ) {
		$product_ids = array_values( array_filter( array_map( 'absint', $product_ids ) ) );
		$types = [];
		foreach ( $product_ids as $product_id ) {
			$type = $this->product_type( $product_id );
			if ( ! isset( self::product_types()[ $type ] ) ) {
				return new WP_Error( 'sn_invalid_sales_product_type', 'نوع یکی از محصولات برای مرحله فروشنده مشخص نیست.' );
			}
			$types[] = $type;
		}
		$special = array_values( array_intersect( array_unique( $types ), [ 'assessment', 'subscription', 'product_star' ] ) );
		if ( $special && count( $product_ids ) !== 1 ) {
			return new WP_Error( 'sn_single_special_product', 'اعتبارسنجی، اشتراک و محصول* باید به‌تنهایی در پیش‌فاکتور ثبت شوند.' );
		}
		if ( $special && max( 1, absint( $quantities[0] ?? 1 ) ) !== 1 ) {
			return new WP_Error( 'sn_single_special_quantity', 'تعداد اعتبارسنجی، اشتراک یا محصول* باید دقیقاً یک باشد.' );
		}
		return true;
	}

}
