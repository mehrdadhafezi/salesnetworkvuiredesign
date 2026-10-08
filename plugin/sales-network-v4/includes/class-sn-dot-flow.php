<?php
if ( ! defined( 'ABSPATH' ) ) { exit; }

/**
 * Dot credit-assessment and conversion flow.
 *
 * This module is intentionally additive. It keeps all flow state in dedicated
 * tables and only attaches an existing CRM invoice to the flow by a link row.
 */
final class SN_Dot_Flow {
	private const DB_VERSION = '2026-09-11-conversion-outcomes-v7';
	private const CONFIG_OPTION = 'sn_dot_config';
	private const MARKETING_CONFIG_OPTION = 'sn_dot_marketing_config';
	private const MARKETING_RR_OPTION = 'sn_dot_marketing_rr_index';
	private const MARKETING_OTP_PREFIX = 'sn_dot_marketing_otp_';
	private const MARKETING_OTP_PROOF_PREFIX = 'sn_dot_marketing_otp_proof_';
	private const MARKETING_EVENT_TOKEN_PREFIX = 'sn_dot_marketing_event_';
	private const MARKETING_RETURN_PREFIX = 'sn_dot_marketing_return_';
	private const TEST_SMS_MODE_OPTION = 'sn_dot_sms_test_mode';
	private const CRON_HOOK = 'sn_dot_send_due_access_sms';
	private const ARCHIVE_CRON_HOOK = 'sn_dot_archive_unpaid_due';
	private const FOLLOWUP_CRON_HOOK = 'sn_dot_send_due_followup_reminder';
	private const ASSESSMENT_UNPAID_DAYS = 3;
	private const SUBSCRIPTION_UNPAID_DAYS = 5;
	// Customer access is intentionally long-lived for this business flow. The
	// SMS link itself never expires; the cookie only avoids asking for the code
	// again on the same browser.
	private const SESSION_TTL = 31536000;
	private const TEST_SMS_QUEUE_PREFIX = 'sn_dot_test_sms_queue_';
	private const TEST_SMS_QUEUE_TTL = 7200;

	private static ?self $instance = null;
	private bool $hooks_registered = false;
	private array $table_columns = [];
	private ?array $assessment_product_ids_cache = null;

	public static function instance(): self {
		if ( null === self::$instance ) { self::$instance = new self(); }
		return self::$instance;
	}

	private function __construct() {}

	public static function activate(): void {
		$self = self::instance();
		$self->install_schema();
		$self->ensure_role_and_hr();
		$self->ensure_flow_pages();
		$self->schedule_archive_job();
	}

	public static function deactivate(): void {
		wp_clear_scheduled_hook( self::ARCHIVE_CRON_HOOK );
		wp_clear_scheduled_hook( self::FOLLOWUP_CRON_HOOK );
	}

	public function register_hooks(): void {
		if ( $this->hooks_registered ) { return; }
		$this->hooks_registered = true;

		add_action( 'init', [ $this, 'maybe_upgrade' ], 7 );
		add_action( 'admin_init', [ $this, 'maybe_upgrade' ], 7 );
		add_action( 'init', [ $this, 'maybe_process_due_sms' ], 25 );
		// `init` has no business payload. Explicitly accept zero hook arguments so
		// WordPress' legacy empty-string action placeholder cannot reach the worker.
		add_action( 'init', [ $this, 'maybe_process_unpaid_archives' ], 26, 0 );
		// No payload is expected from init. Keep this explicit so future typed
		// parameters cannot repeat the 2.0.26 init-hook fatal regression.
		add_action( 'init', [ $this, 'maybe_process_due_followup_reminders' ], 27, 0 );
		add_action( 'wp_footer', [ $this, 'maybe_process_due_sms' ], 1 );
		add_action( 'admin_footer', [ $this, 'maybe_process_due_sms' ], 1 );
		add_action( 'shutdown', [ $this, 'maybe_process_due_sms' ], 1 );
		add_action( self::CRON_HOOK, [ $this, 'process_due_access_sms' ] );
		add_action( self::ARCHIVE_CRON_HOOK, [ $this, 'process_due_unpaid_archives' ] );
		add_action( self::FOLLOWUP_CRON_HOOK, [ $this, 'process_due_followup_reminders' ] );
		add_action( 'sn_invoice_paid', [ $this, 'on_invoice_paid' ], 5, 2 );
		add_action( 'sn_invoice_payment_stage_approved', [ $this, 'on_invoice_payment_stage_approved' ], 5, 3 );
		add_action( 'sn_invoice_financial_rejected', [ $this, 'on_financial_rejected' ], 10, 3 );

		foreach ( [
			'sn_dot_verify_code' => 'handle_verify_code',
			'sn_dot_select_option' => 'handle_select_option',
			'sn_dot_assign_converter' => 'handle_assign_converter',
			'sn_dot_converter_update' => 'handle_converter_update',
			'sn_dot_create_payment' => 'handle_create_payment',
			'sn_dot_resend_payment' => 'handle_resend_payment',
			'sn_dot_refresh_converter_panel' => 'handle_refresh_converter_panel',
		] as $action => $method ) {
			add_action( 'admin_post_' . $action, [ $this, $method ] );
			add_action( 'wp_ajax_' . $action, [ $this, $method ] );
		}
		add_action( 'admin_post_nopriv_sn_dot_verify_code', [ $this, 'handle_verify_code' ] );
		add_action( 'admin_post_nopriv_sn_dot_select_option', [ $this, 'handle_select_option' ] );
		add_action( 'admin_post_sn_dot_marketing_start_payment', [ $this, 'handle_marketing_start_payment' ] );
		add_action( 'admin_post_nopriv_sn_dot_marketing_start_payment', [ $this, 'handle_marketing_start_payment' ] );
		add_action( 'wp_ajax_sn_dot_marketing_send_otp', [ $this, 'handle_marketing_send_otp' ] );
		add_action( 'wp_ajax_nopriv_sn_dot_marketing_send_otp', [ $this, 'handle_marketing_send_otp' ] );
		add_action( 'wp_ajax_sn_dot_marketing_verify_otp', [ $this, 'handle_marketing_verify_otp' ] );
		add_action( 'wp_ajax_nopriv_sn_dot_marketing_verify_otp', [ $this, 'handle_marketing_verify_otp' ] );
		add_action( 'wp_ajax_sn_dot_marketing_save_lead', [ $this, 'handle_marketing_save_lead' ] );
		add_action( 'wp_ajax_nopriv_sn_dot_marketing_save_lead', [ $this, 'handle_marketing_save_lead' ] );
		add_action( 'wp_ajax_sn_dot_marketing_record_exit', [ $this, 'handle_marketing_record_exit' ] );
		add_action( 'wp_ajax_nopriv_sn_dot_marketing_record_exit', [ $this, 'handle_marketing_record_exit' ] );
		add_action( 'wp_ajax_sn_dot_marketing_record_event', [ $this, 'handle_marketing_record_event' ] );
		add_action( 'wp_ajax_nopriv_sn_dot_marketing_record_event', [ $this, 'handle_marketing_record_event' ] );
		add_action( 'wp_ajax_sn_dot_poll_test_sms', [ $this, 'handle_poll_test_sms' ] );
		add_action( 'wp_ajax_sn_dot_converter_invoices', [ $this, 'handle_converter_invoices' ] );
		add_action( 'wp_ajax_sn_dot_bulk_assign_converter', [ $this, 'handle_bulk_assign_converter' ] );
		add_action( 'wp_ajax_sn_dot_gift_validation_access', [ $this, 'handle_gift_validation_access' ] );
		add_action( 'admin_post_sn_dot_marketing_export_leads', [ $this, 'handle_marketing_export_leads' ] );
		add_action( 'admin_post_sn_dot_export_report', [ $this, 'handle_dot_export_report' ] );
	}

	public function maybe_upgrade(): void {
		if ( (string) get_option( 'sn_dot_db_version', '' ) !== self::DB_VERSION ) {
			$this->install_schema();
		}
		if ( (string) get_option( 'sn_dot_hr_version', '' ) !== self::DB_VERSION ) { $this->ensure_role_and_hr(); }
		if ( (string) get_option( 'sn_dot_pages_version', '' ) !== self::DB_VERSION ) { $this->ensure_flow_pages(); }
		$this->schedule_archive_job();
		$this->maybe_backfill_abandoned_marketing_preinvoices();
	}

	/**
	 * Recover pre-2.0.8 abandoned landing submissions in small bounded batches.
	 * A row is converted only when its saved form/product/supervisor snapshot still
	 * exactly matches the active form route; changed or deleted routes are skipped.
	 */
	private function maybe_backfill_abandoned_marketing_preinvoices(): void {
		$option_key = 'sn_dot_abandoned_preinvoice_backfill';
		$state = get_option( $option_key, [] );
		$state = is_array( $state ) ? $state : [];
		if ( (string) ( $state['version'] ?? '' ) === self::DB_VERSION && ! empty( $state['done'] ) ) { return; }
		if ( empty( $this->config()['enabled'] ) || get_transient( 'sn_dot_abandoned_preinvoice_backfill_lock' ) ) { return; }
		global $wpdb;
		$t = $this->tables();
		if ( ! $this->table_exists( $t['marketing_leads'] ) ) { return; }
		set_transient( 'sn_dot_abandoned_preinvoice_backfill_lock', '1', 45 );
		$is_current_state = (string) ( $state['version'] ?? '' ) === self::DB_VERSION;
		$last_id = $is_current_state ? absint( $state['last_id'] ?? 0 ) : 0;
		$created_count = $is_current_state ? absint( $state['created'] ?? 0 ) : 0;
		$skipped_count = $is_current_state ? absint( $state['skipped'] ?? 0 ) : 0;
		$rows = $wpdb->get_results( $wpdb->prepare(
			"SELECT * FROM {$t['marketing_leads']} WHERE id>%d AND invoice_id IS NULL AND status IN ('payment_not_done','popup_cancelled','otp_cancelled') ORDER BY id ASC LIMIT 20",
			$last_id
		) ) ?: [];
		foreach ( $rows as $row ) {
			$last_id = (int) $row->id;
			$config = $this->marketing_config_for_form( (string) $row->form_id );
			$route_matches = ! empty( $config['_form_exists'] )
				&& ! empty( $config['enabled'] )
				&& (int) $config['product_id'] === (int) $row->product_id
				&& in_array( (int) $row->supervisor_id, array_map( 'intval', (array) $config['supervisor_ids'] ), true );
			if ( ! $route_matches ) { $skipped_count++; continue; }
			$result = $this->ensure_marketing_source_invoice( $row, $config );
			if ( is_wp_error( $result ) ) { $skipped_count++; continue; }
			$created_count++;
		}
		$done = count( $rows ) < 20;
		update_option( $option_key, [
			'version' => self::DB_VERSION,
			'last_id' => $last_id,
			'created' => $created_count,
			'skipped' => $skipped_count,
			'done' => $done ? 1 : 0,
			'updated_at' => current_time( 'mysql' ),
		], false );
		delete_transient( 'sn_dot_abandoned_preinvoice_backfill_lock' );
	}

	/** Create only the two new flow pages on an in-place plugin update. */
	private function ensure_flow_pages(): void {
		$pages = [
			'sn_dot_customer_page_id' => [ 'title' => 'تایید اعتبارسنجی', 'slug' => 'credit-assessment', 'tag' => 'sn_dot_customer_flow' ],
			'sn_dot_converter_panel_page_id' => [ 'title' => 'پنل تبدیل‌کننده', 'slug' => 'crm-converter', 'tag' => 'sn_dot_converter_panel' ],
		];
		$changed = false;
		foreach ( $pages as $option_key => $definition ) {
			$page_id = absint( get_option( $option_key, 0 ) );
			$post = $page_id ? get_post( $page_id ) : null;
			if ( ! $post || 'page' !== $post->post_type || 'trash' === $post->post_status ) {
				$post = get_page_by_path( (string) $definition['slug'], OBJECT, 'page' );
				$page_id = $post && 'trash' !== $post->post_status ? (int) $post->ID : 0;
			}
			$shortcode = '[' . (string) $definition['tag'] . ']';
			if ( ! $page_id ) {
				$created = wp_insert_post( [
					'post_title' => (string) $definition['title'], 'post_name' => (string) $definition['slug'],
					'post_content' => $shortcode, 'post_status' => 'publish', 'post_type' => 'page',
					'comment_status' => 'closed', 'ping_status' => 'closed',
				], true );
				if ( is_wp_error( $created ) ) { continue; }
				$page_id = (int) $created;
				update_post_meta( $page_id, '_sn_system_page', '1' );
				$post = get_post( $page_id );
				$changed = true;
			}
			if ( $post && ! has_shortcode( (string) $post->post_content, (string) $definition['tag'] ) ) {
				wp_update_post( [ 'ID' => $page_id, 'post_content' => trim( (string) $post->post_content . "\n\n" . $shortcode ) ] );
				$changed = true;
			}
			if ( $post && 'publish' !== $post->post_status ) {
				wp_update_post( [ 'ID' => $page_id, 'post_status' => 'publish' ] );
				$changed = true;
			}
			update_option( $option_key, $page_id, false );
		}
		if ( $changed ) { update_option( 'sn_flush_rewrite_needed', '1', false ); }
		$customer_page = absint( get_option( 'sn_dot_customer_page_id', 0 ) );
		$converter_page = absint( get_option( 'sn_dot_converter_panel_page_id', 0 ) );
		if ( $customer_page && $converter_page && get_post( $customer_page ) && get_post( $converter_page ) ) {
			update_option( 'sn_dot_pages_version', self::DB_VERSION, false );
		}
	}

	private function tables(): array {
		global $wpdb;
		return [
			'cases' => $wpdb->prefix . 'sn_dot_cases',
			'links' => $wpdb->prefix . 'sn_dot_invoice_links',
			'payments' => $wpdb->prefix . 'sn_dot_payments',
			'logs' => $wpdb->prefix . 'sn_dot_case_logs',
			'archives' => $wpdb->prefix . 'sn_dot_operational_archives',
			'marketing_leads' => $wpdb->prefix . 'sn_dot_marketing_leads',
			'marketing_events' => $wpdb->prefix . 'sn_dot_marketing_events',
		];
	}

	private function install_schema(): void {
		global $wpdb;
		require_once ABSPATH . 'wp-admin/includes/upgrade.php';
		$t = $this->tables();
		$charset = $wpdb->get_charset_collate();

		dbDelta( "CREATE TABLE {$t['cases']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			referral_item_id BIGINT UNSIGNED DEFAULT NULL,
			source_invoice_id BIGINT UNSIGNED NOT NULL,
			source_product_id BIGINT UNSIGNED NOT NULL,
			seller_id BIGINT UNSIGNED NOT NULL,
			supervisor_id BIGINT UNSIGNED NOT NULL,
			converter_id BIGINT UNSIGNED DEFAULT NULL,
			conversion_route VARCHAR(30) DEFAULT NULL,
			access_sms_enabled TINYINT(1) NOT NULL DEFAULT 1,
			customer_wp_id BIGINT UNSIGNED DEFAULT NULL,
			customer_name VARCHAR(120) NOT NULL,
			customer_phone VARCHAR(20) NOT NULL,
			province VARCHAR(60) DEFAULT NULL,
			city VARCHAR(60) DEFAULT NULL,
			national_id VARCHAR(20) DEFAULT NULL,
			nominal_credit_amount DECIMAL(20,2) NOT NULL DEFAULT 0,
			nominal_credit_key VARCHAR(80) DEFAULT NULL,
			options_snapshot_json LONGTEXT DEFAULT NULL,
			status VARCHAR(60) NOT NULL DEFAULT 'awaiting_access_sms',
			selected_option_key VARCHAR(80) DEFAULT NULL,
			selected_option_product_id BIGINT UNSIGNED DEFAULT NULL,
			selected_option_title VARCHAR(191) DEFAULT NULL,
			selected_option_price DECIMAL(20,2) DEFAULT NULL,
			selected_option_html LONGTEXT DEFAULT NULL,
			selected_by BIGINT UNSIGNED DEFAULT NULL,
			selected_at DATETIME DEFAULT NULL,
			access_token_hash CHAR(64) DEFAULT NULL,
			access_code_hash VARCHAR(255) DEFAULT NULL,
			access_expires_at DATETIME DEFAULT NULL,
			access_failed_attempts INT UNSIGNED NOT NULL DEFAULT 0,
			access_locked_until DATETIME DEFAULT NULL,
			session_token_hash CHAR(64) DEFAULT NULL,
			session_expires_at DATETIME DEFAULT NULL,
			sms_due_at DATETIME DEFAULT NULL,
			sms_sent_at DATETIME DEFAULT NULL,
			sms_previewed_at DATETIME DEFAULT NULL,
			sms_delivery_mode VARCHAR(20) DEFAULT NULL,
			sms_attempts INT UNSIGNED NOT NULL DEFAULT 0,
			sms_last_error VARCHAR(191) DEFAULT NULL,
			customer_viewed_at DATETIME DEFAULT NULL,
			assigned_by BIGINT UNSIGNED DEFAULT NULL,
			assigned_at DATETIME DEFAULT NULL,
			converter_contact_status VARCHAR(40) DEFAULT NULL,
			converter_no_answer_attempts INT UNSIGNED NOT NULL DEFAULT 0,
			converter_note TEXT DEFAULT NULL,
			converter_decline_reason TEXT DEFAULT NULL,
			converter_declined_at DATETIME DEFAULT NULL,
			converter_followup_at DATETIME DEFAULT NULL,
			converter_followup_reminded_at DATETIME DEFAULT NULL,
			converter_followup_reminder_attempts TINYINT UNSIGNED NOT NULL DEFAULT 0,
			converter_followup_reminder_error VARCHAR(191) DEFAULT NULL,
			payment_mode VARCHAR(30) DEFAULT NULL,
			deposit_amount DECIMAL(20,2) NOT NULL DEFAULT 0,
			paid_amount DECIMAL(20,2) NOT NULL DEFAULT 0,
			remaining_amount DECIMAL(20,2) NOT NULL DEFAULT 0,
			completed_at DATETIME DEFAULT NULL,
			validation_fee_gifted TINYINT(1) NOT NULL DEFAULT 0,
			validation_gifted_by BIGINT UNSIGNED DEFAULT NULL,
			validation_gifted_at DATETIME DEFAULT NULL,
			validation_gift_sms_sent_at DATETIME DEFAULT NULL,
			validation_gift_sms_error VARCHAR(191) DEFAULT NULL,
			operational_archived_at DATETIME DEFAULT NULL,
			operational_archive_reason VARCHAR(60) DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY  (id),
			UNIQUE KEY referral_item_id (referral_item_id),
			UNIQUE KEY source_invoice_id (source_invoice_id),
			KEY supervisor_status (supervisor_id,status),
			KEY converter_status (converter_id,status),
			KEY converter_followup (converter_contact_status,converter_followup_at,converter_followup_reminded_at),
			KEY seller_route (seller_id,conversion_route,converter_id),
			KEY supervisor_converter_route (supervisor_id,converter_id,conversion_route),
			KEY customer_phone (customer_phone),
			KEY selected_option_product_id (selected_option_product_id),
			KEY sms_due_at (sms_due_at),
			KEY validation_fee_gifted (validation_fee_gifted),
			KEY operational_archive (operational_archived_at,operational_archive_reason)
		) {$charset};" );

		dbDelta( "CREATE TABLE {$t['links']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			invoice_id BIGINT UNSIGNED NOT NULL,
			case_id BIGINT UNSIGNED DEFAULT NULL,
			flow_kind VARCHAR(40) NOT NULL,
			payment_kind VARCHAR(30) NOT NULL DEFAULT 'source',
			marketing_form_id VARCHAR(80) DEFAULT NULL,
			supervisor_id BIGINT UNSIGNED NOT NULL,
			conversion_route VARCHAR(30) DEFAULT NULL,
			send_assessment_sms TINYINT(1) NOT NULL DEFAULT 1,
			requires_finance TINYINT(1) NOT NULL DEFAULT 0,
			sms_delay_minutes INT UNSIGNED NOT NULL DEFAULT 30,
			payment_verified_at DATETIME DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY  (id),
			UNIQUE KEY invoice_id (invoice_id),
			KEY case_id (case_id),
			KEY flow_kind (flow_kind),
			KEY marketing_form_id (marketing_form_id)
		) {$charset};" );

		dbDelta( "CREATE TABLE {$t['payments']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			case_id BIGINT UNSIGNED NOT NULL,
			invoice_id BIGINT UNSIGNED NOT NULL,
			payment_type VARCHAR(30) NOT NULL,
			requested_amount DECIMAL(20,2) NOT NULL DEFAULT 0,
			status VARCHAR(40) NOT NULL DEFAULT 'pending',
			sms_sent_at DATETIME DEFAULT NULL,
			created_by BIGINT UNSIGNED DEFAULT NULL,
			paid_at DATETIME DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY  (id),
			UNIQUE KEY invoice_id (invoice_id),
			KEY case_status (case_id,status),
			KEY payment_type (payment_type)
		) {$charset};" );

		dbDelta( "CREATE TABLE {$t['logs']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			case_id BIGINT UNSIGNED NOT NULL,
			actor_id BIGINT UNSIGNED DEFAULT NULL,
			event_key VARCHAR(100) NOT NULL,
			old_status VARCHAR(60) DEFAULT NULL,
			new_status VARCHAR(60) DEFAULT NULL,
			details LONGTEXT DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY  (id),
			KEY case_id (case_id),
			KEY event_key (event_key),
			KEY created_at (created_at)
		) {$charset};" );

		dbDelta( "CREATE TABLE {$t['archives']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			archive_type VARCHAR(40) NOT NULL,
			entity_kind VARCHAR(30) NOT NULL,
			entity_id BIGINT UNSIGNED NOT NULL,
			invoice_id BIGINT UNSIGNED DEFAULT NULL,
			case_id BIGINT UNSIGNED DEFAULT NULL,
			seller_id BIGINT UNSIGNED DEFAULT NULL,
			supervisor_id BIGINT UNSIGNED DEFAULT NULL,
			converter_id BIGINT UNSIGNED DEFAULT NULL,
			sales_manager_user_id BIGINT UNSIGNED NOT NULL,
			customer_name VARCHAR(120) DEFAULT NULL,
			customer_phone VARCHAR(20) DEFAULT NULL,
			product_title VARCHAR(191) DEFAULT NULL,
			total_amount DECIMAL(20,2) NOT NULL DEFAULT 0,
			paid_amount DECIMAL(20,2) NOT NULL DEFAULT 0,
			remaining_amount DECIMAL(20,2) NOT NULL DEFAULT 0,
			due_at DATETIME NOT NULL,
			archived_at DATETIME NOT NULL,
			resolved_at DATETIME DEFAULT NULL,
			resolved_reason VARCHAR(80) DEFAULT NULL,
			details_json LONGTEXT DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY archive_entity (archive_type,entity_kind,entity_id),
			KEY manager_active (sales_manager_user_id,archive_type,resolved_at,archived_at),
			KEY invoice_id (invoice_id),
			KEY case_id (case_id)
		) {$charset};" );

		dbDelta( "CREATE TABLE {$t['marketing_leads']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			crm_lead_id BIGINT UNSIGNED DEFAULT NULL,
			invoice_id BIGINT UNSIGNED DEFAULT NULL,
			form_id VARCHAR(80) NOT NULL DEFAULT 'default',
			form_title VARCHAR(191) DEFAULT NULL,
			product_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
			supervisor_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
			seller_id BIGINT UNSIGNED DEFAULT NULL,
			customer_name VARCHAR(120) NOT NULL,
			customer_phone VARCHAR(20) NOT NULL,
			national_id VARCHAR(20) DEFAULT NULL,
			province VARCHAR(60) DEFAULT NULL,
			city VARCHAR(60) DEFAULT NULL,
			campaign_attribution_id BIGINT UNSIGNED DEFAULT NULL,
			campaign_attribution_status VARCHAR(20) DEFAULT NULL,
			campaign_attributed_at DATETIME DEFAULT NULL,
			campaign_landing_url TEXT DEFAULT NULL,
			utm_source VARCHAR(191) DEFAULT NULL,
			utm_medium VARCHAR(191) DEFAULT NULL,
			utm_campaign VARCHAR(191) DEFAULT NULL,
			utm_content VARCHAR(191) DEFAULT NULL,
			utm_term VARCHAR(191) DEFAULT NULL,
			status VARCHAR(40) NOT NULL DEFAULT 'payment_not_done',
			funnel_stage SMALLINT UNSIGNED NOT NULL DEFAULT 0,
			last_event VARCHAR(40) DEFAULT NULL,
			last_event_reason VARCHAR(191) DEFAULT NULL,
			last_event_at DATETIME DEFAULT NULL,
			application_started_at DATETIME DEFAULT NULL,
			lead_saved_at DATETIME DEFAULT NULL,
			invoice_created_at DATETIME DEFAULT NULL,
			confirmation_opened_at DATETIME DEFAULT NULL,
			confirmation_accepted_at DATETIME DEFAULT NULL,
			otp_requested_at DATETIME DEFAULT NULL,
			otp_verified_at DATETIME DEFAULT NULL,
			gateway_requested_at DATETIME DEFAULT NULL,
			gateway_entered_at DATETIME DEFAULT NULL,
			gateway_returned_at DATETIME DEFAULT NULL,
			paid_at DATETIME DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY  (id),
			UNIQUE KEY invoice_id (invoice_id),
			UNIQUE KEY crm_lead_id (crm_lead_id),
			KEY form_status (form_id,status),
			KEY funnel_stage (funnel_stage),
			KEY last_event (last_event),
			KEY supervisor_id (supervisor_id),
			KEY campaign_attribution_id (campaign_attribution_id),
			KEY campaign_attribution_status (campaign_attribution_status),
			KEY utm_source (utm_source),
			KEY customer_phone (customer_phone),
			KEY created_at (created_at)
		) {$charset};" );

		dbDelta( "CREATE TABLE {$t['marketing_events']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			marketing_lead_id BIGINT UNSIGNED NOT NULL,
			event_key CHAR(64) NOT NULL,
			event_type VARCHAR(40) NOT NULL,
			event_status VARCHAR(20) NOT NULL DEFAULT 'success',
			reason VARCHAR(191) DEFAULT NULL,
			details LONGTEXT DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY event_key (event_key),
			KEY lead_created (marketing_lead_id,created_at),
			KEY event_created (event_type,created_at)
		) {$charset};" );

		// Older rows predate explicit funnel events. Preserve their current state as
		// an auditable snapshot instead of leaving the new report columns empty.
		if ( $this->table_exists( $t['marketing_leads'] ) ) {
			$wpdb->query( "UPDATE {$t['marketing_leads']} SET last_event=status,last_event_reason='legacy_status_snapshot',last_event_at=COALESCE(paid_at,updated_at,created_at) WHERE last_event IS NULL OR last_event=''" );
			$wpdb->query( "UPDATE {$t['marketing_leads']} SET gateway_returned_at=COALESCE(paid_at,updated_at,created_at) WHERE gateway_returned_at IS NULL AND status IN ('pending_finance','payment_paid','payment_rejected')" );
			$wpdb->query( "UPDATE {$t['marketing_leads']} SET application_started_at=COALESCE(application_started_at,created_at),lead_saved_at=COALESCE(lead_saved_at,created_at),invoice_created_at=IF(invoice_id IS NOT NULL,COALESCE(invoice_created_at,updated_at,created_at),invoice_created_at),funnel_stage=GREATEST(funnel_stage,CASE WHEN status IN ('payment_paid','validation_gifted') THEN 100 WHEN status='pending_finance' THEN 90 WHEN status IN ('gateway_started','gateway_cancelled','gateway_failed','payment_rejected') THEN 80 WHEN invoice_id IS NOT NULL THEN 30 ELSE 20 END)" );
		}
		if ( $this->table_exists( $t['marketing_events'] ) && $this->table_exists( $t['marketing_leads'] ) ) {
			$wpdb->query( "INSERT IGNORE INTO {$t['marketing_events']} (marketing_lead_id,event_key,event_type,event_status,reason,created_at) SELECT id,SHA2(CONCAT('legacy|',id),256),COALESCE(NULLIF(last_event,''),status),'legacy',COALESCE(last_event_reason,'legacy_status_snapshot'),COALESCE(last_event_at,updated_at,created_at) FROM {$t['marketing_leads']}" );
		}

		// Conversion payments always remain finance-reviewed. Source assessment
		// links keep their per-invoice snapshot because Marketing Dot Flow can be
		// configured for direct gateway approval.
		if ( $this->table_exists( $t['links'] ) ) {
			$wpdb->query( "UPDATE {$t['links']} SET requires_finance=1 WHERE flow_kind='conversion_payment' AND requires_finance<>1" );
		}

		$schema_ready = true;
		foreach ( $t as $table ) { $schema_ready = $schema_ready && $this->table_exists( $table ); }
		if ( $schema_ready ) {
			$this->migrate_legacy_test_sms_deliveries();
			$this->migrate_legacy_marketing_leads();
			$this->backfill_missing_marketing_source_invoices();
			update_option( 'sn_dot_db_version', self::DB_VERSION, false );
		}
		$this->table_columns = [];
	}

	/** Preserve pre-upgrade Marketing Dot Flow leads in the dedicated audit table. */
	private function migrate_legacy_marketing_leads(): void {
		if ( (string) get_option( 'sn_dot_marketing_leads_migration', '' ) === self::DB_VERSION ) { return; }
		global $wpdb;
		$t = $this->tables();
		$lead_table = $wpdb->prefix . 'sn_leads';
		$invoice_table = $wpdb->prefix . 'sn_invoices';
		if ( ! $this->table_exists( $t['marketing_leads'] ) || ! $this->table_exists( $lead_table ) || ! in_array( 'marketing_form_id', $this->columns( $lead_table ), true ) ) { return; }

		$last_id = 0;
		do {
			$rows = $wpdb->get_results( $wpdb->prepare(
				"SELECT * FROM {$lead_table} WHERE id>%d AND marketing_form_id IS NOT NULL AND marketing_form_id<>'' ORDER BY id ASC LIMIT 200",
				$last_id
			) ) ?: [];
			foreach ( $rows as $lead ) {
				$last_id = (int) $lead->id;
				$exists = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$t['marketing_leads']} WHERE crm_lead_id=%d LIMIT 1", (int) $lead->id ) );
				if ( $exists ) { continue; }
				$invoice = $this->table_exists( $invoice_table ) ? $wpdb->get_row( $wpdb->prepare( "SELECT id,product_id,seller_id,status,paid_at,created_at FROM {$invoice_table} WHERE lead_id=%d ORDER BY id DESC LIMIT 1", (int) $lead->id ) ) : null;
				$form_id = sanitize_key( (string) $lead->marketing_form_id ) ?: 'default';
				$form_config = $this->marketing_config_for_form( $form_id );
				$lead_status = sanitize_key( (string) ( $lead->lead_status ?? '' ) );
				$status = $lead_status === 'payment_paid' ? 'payment_paid' : 'payment_not_done';
				$legacy_event_at = (string) ( $lead->updated_at ?? '' );
				if ( $legacy_event_at === '' && $invoice ) { $legacy_event_at = (string) ( $invoice->paid_at ?: $invoice->created_at ); }
				if ( $legacy_event_at === '' ) { $legacy_event_at = current_time( 'mysql' ); }
				$wpdb->insert( $t['marketing_leads'], [
					'crm_lead_id' => (int) $lead->id,
					'invoice_id' => $invoice ? (int) $invoice->id : null,
					'form_id' => $form_id,
					'form_title' => (string) ( $form_config['_form_title'] ?? ( $form_id === 'default' ? 'فرم پیش‌فرض' : $form_id ) ),
					'product_id' => $invoice ? (int) $invoice->product_id : absint( $form_config['product_id'] ?? 0 ),
					'supervisor_id' => (int) ( $lead->supervisor_id ?? 0 ),
					'seller_id' => $invoice ? (int) $invoice->seller_id : (int) ( $lead->seller_id ?? 0 ),
					'customer_name' => (string) ( $lead->customer_name ?? '' ),
					'customer_phone' => (string) $lead->phone,
					'national_id' => (string) ( $lead->national_id ?? '' ),
					'province' => (string) ( $lead->province ?? '' ),
					'city' => (string) ( $lead->city ?? '' ),
					'status' => $status,
					'last_event' => $status,
					'last_event_reason' => 'legacy_lead_migration',
					'last_event_at' => $legacy_event_at,
					'paid_at' => $status === 'payment_paid' && $invoice ? ( $invoice->paid_at ?: null ) : null,
					'created_at' => (string) ( $lead->imported_at ?? ( $invoice->created_at ?? current_time( 'mysql' ) ) ),
					'updated_at' => (string) ( $lead->updated_at ?? current_time( 'mysql' ) ),
				] );
			}
		} while ( count( $rows ) === 200 );
		if ( ! $wpdb->last_error ) { update_option( 'sn_dot_marketing_leads_migration', self::DB_VERSION, false ); }
	}

	private function marketing_form_identity_for_product( int $product_id, string $preferred_form_id = '' ): array {
		$preferred_form_id = sanitize_key( $preferred_form_id );
		$forms = get_option( 'sn_dot_marketing_forms', [] );
		$forms = is_array( $forms ) ? $forms : [];
		if ( $preferred_form_id !== '' ) {
			if ( $preferred_form_id === 'default' ) { return [ 'id' => 'default', 'title' => 'فرم پیش‌فرض' ]; }
			$form = is_array( $forms[ $preferred_form_id ] ?? null ) ? $forms[ $preferred_form_id ] : [];
			return [
				'id' => $preferred_form_id,
				'title' => sanitize_text_field( (string) ( $form['title'] ?? $preferred_form_id ) ) ?: $preferred_form_id,
			];
		}

		$candidates = [];
		$default = $this->marketing_config();
		if ( $product_id > 0 && (int) ( $default['product_id'] ?? 0 ) === $product_id ) {
			$candidates['default'] = 'فرم پیش‌فرض';
		}
		foreach ( $forms as $form_id => $form ) {
			if ( ! is_array( $form ) || absint( $form['product_id'] ?? 0 ) !== $product_id ) { continue; }
			$form_id = sanitize_key( (string) ( $form['id'] ?? $form_id ) );
			if ( $form_id === '' || $form_id === 'default' ) { continue; }
			$candidates[ $form_id ] = sanitize_text_field( (string) ( $form['title'] ?? $form_id ) ) ?: $form_id;
		}
		if ( count( $candidates ) === 1 ) {
			$form_id = (string) array_key_first( $candidates );
			return [ 'id' => $form_id, 'title' => (string) $candidates[ $form_id ] ];
		}
		$form_id = $product_id > 0 ? 'legacy-product-' . $product_id : 'legacy-unknown';
		return [
			'id' => sanitize_key( $form_id ),
			'title' => $product_id > 0 ? 'بازیابی‌شده از محصول #' . $product_id : 'بازیابی‌شده از فاکتور قدیمی',
		];
	}

	private function marketing_status_from_invoice_record( $invoice ): string {
		$get = static function ( string $key ) use ( $invoice ) {
			if ( is_array( $invoice ) ) { return $invoice[ $key ] ?? ''; }
			return is_object( $invoice ) ? ( $invoice->{$key} ?? '' ) : '';
		};
		$statuses = array_values( array_filter( array_map( 'sanitize_key', [
			(string) $get( 'status' ),
			(string) $get( 'invoice_status' ),
			(string) $get( 'payment_status' ),
			(string) $get( 'payment_workflow_status' ),
		] ) ) );
		if ( array_intersect( $statuses, [ 'pending_financial_approval', 'receipt_uploaded', 'awaiting_financial_approval', 'pending_finance' ] ) ) {
			return 'pending_finance';
		}
		if ( array_intersect( $statuses, [ 'rejected', 'payment_rejected', 'financial_rejected', 'failed', 'cancelled', 'canceled' ] ) ) {
			return 'payment_rejected';
		}
		if ( trim( (string) $get( 'paid_at' ) ) !== '' || array_intersect( $statuses, [ 'paid', 'approved', 'completed', 'success' ] ) ) {
			return 'payment_paid';
		}
		return 'awaiting_payment';
	}

	/**
	 * Reconcile older Marketing Dot source invoices that predate the dedicated
	 * lead table. Inserts only missing invoice rows and never overwrites a lead.
	 */
	private function backfill_missing_marketing_source_invoices(): void {
		if ( (string) get_option( 'sn_dot_marketing_invoice_backfill', '' ) === self::DB_VERSION ) { return; }
		global $wpdb;
		$t = $this->tables();
		$invoice_table = $wpdb->prefix . 'sn_invoices';
		$lead_table = $wpdb->prefix . 'sn_leads';
		if ( ! $this->table_exists( $t['marketing_leads'] ) || ! $this->table_exists( $t['links'] ) || ! $this->table_exists( $invoice_table ) ) { return; }
		if ( ! in_array( 'marketing_form_id', $this->columns( $t['links'] ), true ) ) { return; }

		$ok = false !== $wpdb->query(
			"UPDATE {$t['links']} dl INNER JOIN {$t['marketing_leads']} m ON m.invoice_id=dl.invoice_id
			SET dl.marketing_form_id=m.form_id
			WHERE dl.flow_kind='assessment_source' AND (dl.marketing_form_id IS NULL OR dl.marketing_form_id='')"
		);
		$lead_exists = $this->table_exists( $lead_table );
		$lead_columns = $lead_exists ? $this->columns( $lead_table ) : [];
		$lead_form_exists = in_array( 'marketing_form_id', $lead_columns, true );
		$lead_join = $lead_exists ? "LEFT JOIN {$lead_table} l ON l.id=i.lead_id" : '';
		$lead_select = $lead_exists
			? "l.id crm_lead_id,l.customer_name lead_customer_name,l.phone lead_phone,l.national_id lead_national_id,l.province lead_province,l.city lead_city," . ( $lead_form_exists ? 'l.marketing_form_id lead_form_id' : "'' lead_form_id" )
			: "NULL crm_lead_id,'' lead_customer_name,'' lead_phone,'' lead_national_id,'' lead_province,'' lead_city,'' lead_form_id";
		$lead_route_condition = $lead_form_exists ? " OR (l.marketing_form_id IS NOT NULL AND l.marketing_form_id<>'')" : '';

		$last_link_id = 0;
		do {
			$sql = "SELECT i.*,dl.id source_link_id,dl.supervisor_id source_supervisor_id,dl.marketing_form_id source_form_id,{$lead_select}
				FROM {$t['links']} dl
				INNER JOIN {$invoice_table} i ON i.id=dl.invoice_id
				LEFT JOIN {$t['marketing_leads']} m ON m.invoice_id=dl.invoice_id
				{$lead_join}
				WHERE dl.id>%d AND dl.flow_kind='assessment_source' AND m.id IS NULL
				AND (
					(dl.marketing_form_id IS NOT NULL AND dl.marketing_form_id<>'')
					OR EXISTS (SELECT 1 FROM {$wpdb->usermeta} sm WHERE sm.user_id=i.seller_id AND sm.meta_key='_sn_dot_marketing_seller' AND sm.meta_value='1')
					{$lead_route_condition}
				)
				ORDER BY dl.id ASC LIMIT 200";
			$rows = $wpdb->get_results( $wpdb->prepare( $sql, $last_link_id ) ) ?: [];
			foreach ( $rows as $invoice ) {
				$last_link_id = (int) $invoice->source_link_id;
				$identity = $this->marketing_form_identity_for_product(
					(int) ( $invoice->product_id ?? 0 ),
					(string) ( $invoice->source_form_id ?: $invoice->lead_form_id )
				);
				$crm_lead_id = absint( $invoice->crm_lead_id ?? 0 );
				if ( $crm_lead_id > 0 ) {
					$already_linked = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$t['marketing_leads']} WHERE crm_lead_id=%d LIMIT 1", $crm_lead_id ) );
					if ( $already_linked > 0 ) { $crm_lead_id = 0; }
				}
				$status = $this->marketing_status_from_invoice_record( $invoice );
				$inserted = $wpdb->insert( $t['marketing_leads'], [
					'crm_lead_id' => $crm_lead_id > 0 ? $crm_lead_id : null,
					'invoice_id' => (int) $invoice->id,
					'form_id' => (string) $identity['id'],
					'form_title' => (string) $identity['title'],
					'product_id' => (int) ( $invoice->product_id ?? 0 ),
					'supervisor_id' => (int) $invoice->source_supervisor_id,
					'seller_id' => (int) ( $invoice->seller_id ?? 0 ),
					'customer_name' => (string) ( $invoice->customer_name ?: $invoice->lead_customer_name ),
					'customer_phone' => (string) ( $invoice->customer_phone ?: $invoice->lead_phone ),
					'national_id' => (string) ( $invoice->lead_national_id ?? '' ),
					'province' => (string) ( $invoice->province ?: $invoice->lead_province ),
					'city' => (string) ( $invoice->city ?: $invoice->lead_city ),
					'status' => $status,
					'last_event' => $status,
					'last_event_reason' => 'legacy_invoice_backfill',
					'last_event_at' => (string) ( $invoice->paid_at ?: $invoice->updated_at ?: $invoice->created_at ?: current_time( 'mysql' ) ),
					'gateway_returned_at' => in_array( $status, [ 'pending_finance', 'payment_paid', 'payment_rejected' ], true ) ? (string) ( $invoice->paid_at ?: $invoice->updated_at ?: current_time( 'mysql' ) ) : null,
					'paid_at' => $status === 'payment_paid' ? ( $invoice->paid_at ?: null ) : null,
					'created_at' => (string) ( $invoice->created_at ?: current_time( 'mysql' ) ),
					'updated_at' => current_time( 'mysql' ),
				] );
				if ( ! $inserted ) { $ok = false; continue; }
				if ( false === $wpdb->update( $t['links'], [ 'marketing_form_id' => (string) $identity['id'] ], [ 'id' => (int) $invoice->source_link_id ] ) ) {
					$ok = false;
				}
			}
		} while ( count( $rows ) === 200 );
		if ( $ok ) { update_option( 'sn_dot_marketing_invoice_backfill', self::DB_VERSION, false ); }
	}

	/**
	 * Older builds marked a test popup as a delivered SMS. Recover only recent,
	 * unopened cases from an installation that was still storing test mode as on.
	 */
	private function migrate_legacy_test_sms_deliveries(): void {
		if ( (string) get_option( 'sn_dot_test_delivery_migration', '' ) === self::DB_VERSION ) { return; }
		$stored = get_option( self::CONFIG_OPTION, [] );
		$legacy_test_mode = is_array( $stored ) && ! empty( $stored['sms_test_mode'] );
		if ( $legacy_test_mode ) {
			global $wpdb;
			$t = $this->tables();
			$cutoff = $this->local_mysql_from_timestamp( time() - DAY_IN_SECONDS );
			$now = current_time( 'mysql' );
			$wpdb->query( $wpdb->prepare(
				"UPDATE {$t['cases']} SET sms_previewed_at=sms_sent_at,sms_sent_at=NULL,sms_delivery_mode='test',sms_attempts=0,sms_last_error=NULL,sms_due_at=%s,updated_at=%s WHERE sms_sent_at IS NOT NULL AND (sms_delivery_mode IS NULL OR sms_delivery_mode='') AND customer_viewed_at IS NULL AND status='awaiting_customer_selection' AND created_at>=%s",
				$now,
				$now,
				$cutoff
			) );
		}
		if ( false === get_option( self::TEST_SMS_MODE_OPTION, false ) ) {
			add_option( self::TEST_SMS_MODE_OPTION, '0', '', false );
		}
		update_option( 'sn_dot_test_delivery_migration', self::DB_VERSION, false );
	}

	private function ensure_role_and_hr(): void {
		$role = get_role( 'sn_converter' );
		if ( ! $role ) {
			add_role( 'sn_converter', 'تبدیل‌کننده', [ 'read' => true, 'sn_manage_dot_conversion' => true ] );
			$role = get_role( 'sn_converter' );
		}
		if ( $role ) {
			$role->add_cap( 'read' );
			$role->add_cap( 'sn_manage_dot_conversion' );
			foreach ( [ 'edit_posts', 'delete_posts', 'publish_posts', 'upload_files', 'manage_options', 'list_users', 'edit_users' ] as $cap ) {
				$role->remove_cap( $cap );
			}
		}

		global $wpdb;
		$positions = $wpdb->prefix . 'sn_hr_positions';
		$mappings = $wpdb->prefix . 'sn_hr_position_role_mappings';
		if ( ! $this->table_exists( $positions ) || ! $this->table_exists( $mappings ) ) { return; }
		$position_id = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$positions} WHERE slug=%s LIMIT 1", 'converter' ) );
		if ( ! $position_id ) {
			$wpdb->insert( $positions, [
				'slug' => 'converter', 'label' => 'تبدیل‌کننده', 'description' => 'پیگیری آماده‌های تبدیل و ارسال لینک پرداخت',
				'panel_key' => 'converter', 'sort_order' => 45, 'is_active' => 1, 'is_system' => 1,
				'created_at' => current_time( 'mysql' ), 'updated_at' => current_time( 'mysql' ),
			] );
			$position_id = (int) $wpdb->insert_id;
		}
		if ( $position_id ) {
			$mapping_id = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$mappings} WHERE legacy_role=%s LIMIT 1", 'sn_converter' ) );
			$data = [ 'position_id' => $position_id, 'is_active' => 1, 'updated_at' => current_time( 'mysql' ) ];
			if ( $mapping_id ) { $wpdb->update( $mappings, $data, [ 'id' => $mapping_id ] ); }
			else { $wpdb->insert( $mappings, $data + [ 'legacy_role' => 'sn_converter', 'created_at' => current_time( 'mysql' ) ] ); }
		}
		if ( $position_id && ! $wpdb->last_error ) { update_option( 'sn_dot_hr_version', self::DB_VERSION, false ); }
	}

	private function table_exists( string $table ): bool {
		global $wpdb;
		return (string) $wpdb->get_var( $wpdb->prepare( 'SHOW TABLES LIKE %s', $table ) ) === $table;
	}

	private function columns( string $table ): array {
		global $wpdb;
		if ( ! isset( $this->table_columns[ $table ] ) ) {
			$this->table_columns[ $table ] = $this->table_exists( $table ) ? (array) $wpdb->get_col( "SHOW COLUMNS FROM {$table}", 0 ) : [];
		}
		return $this->table_columns[ $table ];
	}

	private function existing_columns( string $table, array $data ): array {
		return array_intersect_key( $data, array_flip( $this->columns( $table ) ) );
	}

	public function default_config(): array {
		return [
			'enabled' => 0,
			'sms_test_mode' => 0,
			'sms_test_run_id' => '',
			'sms_test_viewer_user_id' => 0,
			'sms_delay_minutes' => 30,
			'access_expiry_hours' => 0,
			'online_requires_finance' => 1,
			'product_supervisors' => [],
			'nominal_credits' => [
				[ 'key' => 'credit_1200', 'label' => 'اعتبار ۱ میلیارد و ۲۰۰ میلیون', 'amount' => 1200000000 ],
				[ 'key' => 'credit_950', 'label' => 'اعتبار ۹۵۰ میلیون', 'amount' => 950000000 ],
				[ 'key' => 'credit_850', 'label' => 'اعتبار ۸۵۰ میلیون', 'amount' => 850000000 ],
			],
			'digit_map' => [
				'0' => 'credit_1200', '1' => 'credit_950', '2' => 'credit_850', '3' => 'credit_1200', '4' => 'credit_950',
				'5' => 'credit_850', '6' => 'credit_1200', '7' => 'credit_950', '8' => 'credit_850', '9' => 'credit_1200',
			],
			'options' => [
				[ 'key' => 'option_1', 'title' => 'گزینه ۱', 'content' => '', 'color' => '#b8893e', 'product_id' => 0, 'price' => 0, 'active' => 1 ],
				[ 'key' => 'option_2', 'title' => 'گزینه ۲', 'content' => '', 'color' => '#137b72', 'product_id' => 0, 'price' => 0, 'active' => 1 ],
				[ 'key' => 'option_3', 'title' => 'گزینه ۳', 'content' => '', 'color' => '#27658c', 'product_id' => 0, 'price' => 0, 'active' => 1 ],
				[ 'key' => 'option_4', 'title' => 'گزینه ۴', 'content' => '', 'color' => '#9a4c3c', 'product_id' => 0, 'price' => 0, 'active' => 1 ],
			],
			'customer_ui' => [
				'hero_title' => 'سامانه اعلام اعتبار مشتریان',
				'hero_subtitle' => 'این اعتبارسنجی بر اساس فرمول‌های داخل سازمان به دست آمده است',
				'result_description' => "نتیجه اعتبارسنجی شما، مبلغ اعلام شده فوق می‌باشد که مجموعی از اعتبارات اقساطی، نقدی، تخفیفی و خرید رایگان است.\nدر زیر ۴ مدل پیشنهادی برای مصرف این اعتبار به شما داده شده است که می‌بایست یکی از آنها را انتخاب کنید. پس از انتخاب و ثبت، ظرف ۲۴ ساعت مشاور شما مشخص و با شما ارتباط خواهد گرفت.\nنکته مهم: از هرگونه پرداخت به‌جز حساب شرکت خودداری فرمایید.",
				'about_url' => '',
				'faq_url' => '',
				'contact_url' => '',
			],
			'sms' => [
				'access_pattern' => '',
				'access_template' => "{customer_name} گرامی\nاعتبارسنجی شما تایید شد.\nرمز اختصاصی: {access_code}\nاین دسترسی اختصاصی برای شماست.\n{access_link}",
				'gift_access_pattern' => '',
				'gift_access_template' => "{customer_name} گرامی\nهزینه اعتبارسنجی شما از طرف شرکت به‌صورت هدیه پرداخت شده است.\nبرای مشاهده نتیجه و انتخاب گزینه وارد شوید:\n{access_link}\nرمز اختصاصی: {access_code}",
				'payment_pattern' => '',
				'payment_template' => "{customer_name} گرامی\nلینک پرداخت {payment_type} به مبلغ {amount} تومان:\n{invoice_url}",
				'reject_pattern' => '',
				'reject_template' => "{seller_name} گرامی، پرداخت اعتبارسنجی مشتری {customer_name} با شماره {customer_phone} رد شد. دلیل: {reason}",
				'followup_pattern' => '',
				'followup_template' => "{staff_name} گرامی، زمان تماس مجدد با {customer_name} ({customer_phone}) برای پرونده #{case_id} رسیده است. موعد: {followup_at}",
			],
		];
	}

	public function config(): array {
		$stored = get_option( self::CONFIG_OPTION, [] );
		$stored = is_array( $stored ) ? $stored : [];
		$config = array_replace_recursive( $this->default_config(), $stored );
		// Numeric configuration lists must be replaced as a whole. Recursively
		// merging them would resurrect deleted default rows at their old indexes.
		foreach ( [ 'nominal_credits', 'options' ] as $list_key ) {
			if ( array_key_exists( $list_key, $stored ) && is_array( $stored[ $list_key ] ) ) {
				$config[ $list_key ] = array_values( $stored[ $list_key ] );
			}
		}
		// Option colors were added after the original Dot-flow rollout. Keep old
		// configurations valid and preserve the former four-color visual palette.
		$legacy_option_colors = [ '#b8893e', '#137b72', '#27658c', '#9a4c3c' ];
		foreach ( (array) $config['options'] as $option_index => $option_row ) {
			$option_row = is_array( $option_row ) ? $option_row : [];
			$color = sanitize_hex_color( (string) ( $option_row['color'] ?? '' ) );
			$option_row['color'] = $color ?: $legacy_option_colors[ $option_index % count( $legacy_option_colors ) ];
			$option_row['content'] = (string) ( $option_row['content'] ?? '' );
			$option_row['product_id'] = absint( $option_row['product_id'] ?? 0 );
			if ( $option_row['product_id'] > 0 && function_exists( 'wc_get_product' ) ) {
				$product = wc_get_product( $option_row['product_id'] );
				if ( $product && 'publish' === get_post_status( $option_row['product_id'] ) ) { $option_row['price'] = max( 0, (float) $product->get_price() ); }
			}
			$config['options'][ $option_index ] = $option_row;
		}
		// Upgrade only the former built-in copy; never overwrite a genuinely
		// customized administrator template.
		$legacy_access_template = "{customer_name} گرامی\nاعتبارسنجی شما تایید شد.\nرمز اختصاصی: {access_code}\nورود: {access_link}";
		if ( trim( (string) ( $config['sms']['access_template'] ?? '' ) ) === trim( $legacy_access_template ) ) {
			$config['sms']['access_template'] = (string) $this->default_config()['sms']['access_template'];
		}
		$config['online_requires_finance'] = 1;
		// A separate exact-value kill switch prevents an old nested option or a
		// cached settings form from silently keeping test mode alive.
		$explicit_test_mode = (string) get_option( self::TEST_SMS_MODE_OPTION, '0' ) === '1';
		$config['sms_test_mode'] = $explicit_test_mode && ! empty( $config['sms_test_mode'] ) ? 1 : 0;
		$config['sms_test_viewer_user_id'] = absint( $config['sms_test_viewer_user_id'] ?? 0 );
		return $config;
	}

	public function default_marketing_config(): array {
		return [
			'enabled' => 0,
			'product_id' => 0,
			'supervisor_ids' => [],
			'sms_delay_minutes' => 30,
			'otp_enabled' => 0,
			'online_requires_finance' => 1,
		];
	}

	public function marketing_config(): array {
		$stored = get_option( self::MARKETING_CONFIG_OPTION, [] );
		$stored = is_array( $stored ) ? $stored : [];
		$config = array_replace( $this->default_marketing_config(), $stored );
		$config['enabled'] = ! empty( $config['enabled'] ) ? 1 : 0;
		$config['product_id'] = absint( $config['product_id'] ?? 0 );
		$config['supervisor_ids'] = array_values( array_unique( array_filter( array_map( 'absint', (array) ( $config['supervisor_ids'] ?? [] ) ) ) ) );
		$config['sms_delay_minutes'] = min( 10080, max( 0, absint( $config['sms_delay_minutes'] ?? 30 ) ) );
		$config['otp_enabled'] = ! empty( $config['otp_enabled'] ) ? 1 : 0;
		$config['online_requires_finance'] = ! empty( $config['online_requires_finance'] ) ? 1 : 0;
		return $config;
	}

	public function marketing_config_for_form( string $form_id = 'default' ): array {
		$config = $this->marketing_config();
		$form_id = sanitize_key( $form_id ) ?: 'default';
		$forms = get_option( 'sn_dot_marketing_forms', [] );
		$config['_form_id'] = $form_id;
		$config['_form_title'] = 'فرم پیش‌فرض';
		$config['_form_exists'] = $form_id === 'default' ? 1 : 0;
		if ( $form_id !== 'default' && is_array( $forms ) && ! empty( $forms[ $form_id ] ) && is_array( $forms[ $form_id ] ) ) {
			$config = array_replace( $config, $forms[ $form_id ] );
			$config['product_id'] = absint( $config['product_id'] ?? 0 );
			$config['supervisor_ids'] = array_values( array_unique( array_filter( array_map( 'absint', (array) ( $config['supervisor_ids'] ?? [] ) ) ) ) );
			$config['enabled'] = ! empty( $config['enabled'] ) ? 1 : 0;
			$config['otp_enabled'] = ! empty( $config['otp_enabled'] ) ? 1 : 0;
			$config['online_requires_finance'] = ! empty( $config['online_requires_finance'] ) ? 1 : 0;
			$config['_form_id'] = $form_id;
			$config['_form_title'] = sanitize_text_field( (string) ( $forms[ $form_id ]['title'] ?? $form_id ) ) ?: $form_id;
			$config['_form_exists'] = 1;
		} elseif ( $form_id !== 'default' ) {
			// Never let a misspelled or deleted shortcode silently fall back to the
			// default product/supervisor route.
			$config['enabled'] = 0;
			$config['product_id'] = 0;
			$config['supervisor_ids'] = [];
			$config['_form_title'] = $form_id;
		}
		return $config;
	}

	private function marketing_rr_option_name( string $form_id ): string {
		$form_id = sanitize_key( $form_id ) ?: 'default';
		return $form_id === 'default' ? self::MARKETING_RR_OPTION : self::MARKETING_RR_OPTION . '_' . md5( $form_id );
	}

	private function is_marketing_seller_id( int $user_id ): bool {
		return $user_id > 0 && (string) get_user_meta( $user_id, '_sn_dot_marketing_seller', true ) === '1';
	}

	private function marketing_seller_for_supervisor( int $supervisor_id ): int {
		$supervisor_id = absint( $supervisor_id );
		if ( $supervisor_id < 1 || ! get_user_by( 'id', $supervisor_id ) ) { return 0; }
		$existing = get_users( [
			'fields' => 'ids',
			'number' => 5,
			'meta_query' => [
				'relation' => 'AND',
				[ 'key' => '_sn_dot_marketing_seller', 'value' => '1' ],
				[ 'key' => '_sn_dot_marketing_supervisor', 'value' => (string) $supervisor_id ],
			],
		] );
		$user_id = ! empty( $existing ) ? absint( $existing[0] ) : 0;
		$supervisor = get_user_by( 'id', $supervisor_id );
		$display_name = 'مارکتینگ دات فلو — ' . ( $supervisor instanceof WP_User ? $supervisor->display_name : ( '#' . $supervisor_id ) );
		if ( $user_id > 0 ) {
			$user = get_user_by( 'id', $user_id );
			if ( $user instanceof WP_User ) {
				if ( ! in_array( 'sn_seller', (array) $user->roles, true ) ) { $user->add_role( 'sn_seller' ); }
				if ( (string) $user->display_name !== $display_name ) { wp_update_user( [ 'ID' => $user_id, 'display_name' => $display_name ] ); }
				update_user_meta( $user_id, 'sn_supervisor_id', $supervisor_id );
				update_user_meta( $user_id, '_sn_dot_marketing_supervisor', $supervisor_id );
				update_user_meta( $user_id, '_sn_dot_marketing_seller', '1' );
				return $user_id;
			}
		}

		$base = 'sn_dot_marketing_' . $supervisor_id;
		$login = $base;
		$suffix = 2;
		while ( username_exists( $login ) ) { $login = $base . '_' . $suffix++; }
		$created = wp_insert_user( [
			'user_login' => $login,
			'user_pass' => wp_generate_password( 40, true, true ),
			'display_name' => $display_name,
			'role' => 'sn_seller',
		] );
		if ( is_wp_error( $created ) ) { return 0; }
		$user_id = absint( $created );
		update_user_meta( $user_id, 'sn_supervisor_id', $supervisor_id );
		update_user_meta( $user_id, '_sn_dot_marketing_supervisor', $supervisor_id );
		update_user_meta( $user_id, '_sn_dot_marketing_seller', '1' );
		update_user_meta( $user_id, 'show_admin_bar_front', 'false' );
		return $user_id;
	}

	private function next_marketing_supervisor( ?array $config = null, string $form_id = 'default' ): int {
		$config = is_array( $config ) ? $config : $this->marketing_config();
		$form_id = sanitize_key( (string) ( $config['_form_id'] ?? $form_id ) ) ?: 'default';
		$rr_option = $this->marketing_rr_option_name( $form_id );
		$allowed = [];
		foreach ( (array) $config['supervisor_ids'] as $supervisor_id ) {
			if ( get_user_by( 'id', (int) $supervisor_id ) ) { $allowed[] = (int) $supervisor_id; }
		}
		$allowed = array_values( array_unique( $allowed ) );
		if ( ! $allowed ) { return 0; }

		// One configured supervisor is a hard route: every marketing request goes
		// to that supervisor and the round-robin cursor is kept at zero.
		if ( count( $allowed ) === 1 ) {
			if ( (int) get_option( $rr_option, 0 ) !== 0 ) {
				update_option( $rr_option, 0, false );
			}
			return (int) $allowed[0];
		}

		// Atomic DB-side increment keeps true round-robin ordering even when two
		// landing submissions arrive at the same moment. LAST_INSERT_ID(expr) is
		// connection-local, so each request receives the exact counter value that
		// its own atomic UPDATE produced without trusting a front-end cursor.
		global $wpdb;
		add_option( $rr_option, '0', '', false );
		$updated = $wpdb->query( $wpdb->prepare(
			"UPDATE {$wpdb->options} SET option_value=LAST_INSERT_ID(CAST(option_value AS UNSIGNED)+1) WHERE option_name=%s",
			$rr_option
		) );
		if ( false === $updated ) { return 0; }
		$next_counter = max( 1, (int) $wpdb->get_var( 'SELECT LAST_INSERT_ID()' ) );
		wp_cache_delete( $rr_option, 'options' );
		$index = $next_counter - 1;
		return (int) $allowed[ $index % count( $allowed ) ];
	}

	public function is_sms_test_mode(): bool {
		return ! empty( $this->config()['sms_test_mode'] );
	}

	/** Runtime identity used to reject stale/cached test-mode pages. */
	public function sms_test_runtime_id(): string {
		$config = $this->config();
		return ! empty( $config['sms_test_mode'] ) ? $this->test_sms_run_id( $config ) : '';
	}

	/**
	 * Sanitize administrator-authored option markup for safe front-end output.
	 *
	 * WordPress' wp_kses_post() removes <script> tags but intentionally keeps
	 * their text nodes. Older Dot-flow settings can therefore contain orphaned
	 * JavaScript text at the end of the saved HTML. Remove executable blocks
	 * with their contents first, then trim the known legacy feature-accordion
	 * script fragment before applying the normal post HTML allow-list.
	 */
	private function sanitize_option_content( string $content ): string {
		$content = str_replace( "\0", '', $content );
		$without_scripts = preg_replace( '#<(script|noscript)\\b[^>]*>.*?</\\1\\s*>#is', '', $content );
		if ( is_string( $without_scripts ) ) { $content = $without_scripts; }

		// Repair content already saved by older versions after wp_kses_post()
		// stripped the <script> wrapper but left its accordion JavaScript visible.
		$markers = [
			"document.querySelectorAll('.feature",
			'document.querySelectorAll(".feature',
			"btn.addEventListener('click'",
			'btn.addEventListener("click"',
		];
		$cut_at = null;
		foreach ( $markers as $marker ) {
			$pos = strpos( $content, $marker );
			if ( $pos !== false && ( $cut_at === null || $pos < $cut_at ) ) { $cut_at = $pos; }
		}
		if ( $cut_at !== null ) { $content = rtrim( substr( $content, 0, $cut_at ) ); }

		return wp_kses_post( $content );
	}

	/**
	 * Preserve administrator-authored option documents for isolated iframe output.
	 *
	 * Unlike the legacy parent-DOM renderer, the iframe renderer can safely keep
	 * document-level markup such as <style>, <input>, <label> and inline scripts.
	 * The iframe itself is sandboxed and receives a restrictive CSP at render time.
	 */
	private function normalize_option_document_content( string $content ): string {
		$content = str_replace( "\0", '', $content );
		return trim( $content );
	}

	/** Build a self-contained, sandbox-friendly document for an option popup. */
	private function option_iframe_document( string $content ): string {
		$content = $this->normalize_option_document_content( $content );

		// Our CSP must be authoritative; remove any author-supplied CSP meta first.
		$without_csp = preg_replace( '#<meta\b[^>]*http-equiv\s*=\s*(["\']?)Content-Security-Policy\1[^>]*>#i', '', $content );
		if ( is_string( $without_csp ) ) { $content = $without_csp; }

		$security_head = '<meta charset="utf-8">'
			. '<meta name="viewport" content="width=device-width,initial-scale=1,maximum-scale=1">'
			. "<meta http-equiv=\"Content-Security-Policy\" content=\"default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; img-src data: blob: https: http:; font-src data: https: http:; media-src data: blob: https: http:; connect-src 'none'; frame-src 'none'; object-src 'none'; form-action 'none'; base-uri 'none';\">";

		if ( preg_match( '#<head\b[^>]*>#i', $content ) ) {
			$document = preg_replace( '#(<head\b[^>]*>)#i', '$1' . $security_head, $content, 1 );
			return is_string( $document ) ? $document : $content;
		}
		if ( preg_match( '#<html\b[^>]*>#i', $content ) ) {
			$document = preg_replace( '#(<html\b[^>]*>)#i', '$1<head>' . $security_head . '</head>', $content, 1 );
			return is_string( $document ) ? $document : $content;
		}

		return '<!doctype html><html lang="fa" dir="rtl"><head>' . $security_head . '</head><body>' . $content . '</body></html>';
	}

	private function parse_money( $value ): float {
		$value = SN_Helpers::to_english_nums( (string) $value );
		$value = str_replace( [ ',', '٬', '،', ' ', 'تومان' ], '', $value );
		return max( 0, (float) preg_replace( '/[^0-9.]/', '', $value ) );
	}

	public function save_settings_from_request( array $request ) {
		if ( empty( $request['sn_dot_settings_present'] ) || ! isset( $request['sn_dot'] ) || ! is_array( $request['sn_dot'] ) ) { return true; }
		$raw = isset( $request['sn_dot'] ) && is_array( $request['sn_dot'] ) ? wp_unslash( $request['sn_dot'] ) : [];
		$requested_sms_provider = sanitize_key( (string) ( $request['sn_sms_provider'] ?? get_option( 'sn_sms_provider', '' ) ) );
		$previous_config = $this->config();
		$config = $this->default_config();
		$config['enabled'] = ! empty( $raw['enabled'] ) ? 1 : 0;
		$test_mode_raw = $request['sn_dot_sms_test_mode'] ?? '0';
		if ( is_array( $test_mode_raw ) ) { $test_mode_raw = end( $test_mode_raw ); }
		$config['sms_test_mode'] = (string) $test_mode_raw === '1' ? 1 : 0;
		$config['sms_test_run_id'] = $config['sms_test_mode']
			? ( ! empty( $previous_config['sms_test_mode'] ) && ! empty( $previous_config['sms_test_run_id'] ) ? sanitize_text_field( (string) $previous_config['sms_test_run_id'] ) : wp_generate_uuid4() )
			: '';
		$config['sms_test_viewer_user_id'] = $config['sms_test_mode'] ? get_current_user_id() : 0;
		// Financial approval is a hard business rule for every Dot-flow payment.
		// Keep the stored key for backward compatibility, but never allow an
		// assessment, full, deposit, or balance payment to bypass finance.
		$config['online_requires_finance'] = 1;
		$config['sms_delay_minutes'] = min( 10080, max( 0, absint( $raw['sms_delay_minutes'] ?? 30 ) ) );
		// Retain the key for backward-compatible option shape; it is deliberately
		// disabled and no longer participates in customer access validation.
		$config['access_expiry_hours'] = 0;

		$config['nominal_credits'] = [];
		$credit_keys = [];
		foreach ( (array) ( $raw['nominal_credits'] ?? [] ) as $index => $row ) {
			$row = is_array( $row ) ? $row : [];
			$key = substr( sanitize_key( (string) ( $row['key'] ?? '' ) ), 0, 70 );
			if ( $key === '' ) { $key = 'credit_' . ( $index + 1 ); }
			$base_key = $key; $suffix = 2;
			while ( isset( $credit_keys[ $key ] ) ) { $key = substr( $base_key, 0, 66 ) . '_' . $suffix++; }
			$amount = $this->parse_money( $row['amount'] ?? 0 );
			if ( $amount <= 0 ) { continue; }
			$credit_keys[ $key ] = true;
			$label = sanitize_text_field( (string) ( $row['label'] ?? '' ) );
			$config['nominal_credits'][] = [
				'key' => $key,
				'label' => $label !== '' ? $label : $key,
				'amount' => $amount,
			];
		}

		$config['digit_map'] = [];
		foreach ( range( 0, 9 ) as $digit ) {
			$key = sanitize_key( (string) ( $raw['digit_map'][ (string) $digit ] ?? '' ) );
			$config['digit_map'][ (string) $digit ] = isset( $credit_keys[ $key ] ) ? $key : '';
		}

		$config['options'] = [];
		$option_keys = [];
		$previous_option_colors = [];
		$previous_options_by_key = [];
		foreach ( (array) ( $previous_config['options'] ?? [] ) as $previous_option ) {
			$previous_key = sanitize_key( (string) ( $previous_option['key'] ?? '' ) );
			$previous_color = sanitize_hex_color( (string) ( $previous_option['color'] ?? '' ) );
			if ( $previous_key !== '' ) {
				$previous_options_by_key[ $previous_key ] = (array) $previous_option;
				if ( $previous_color ) { $previous_option_colors[ $previous_key ] = $previous_color; }
			}
		}
		foreach ( array_slice( (array) ( $raw['options'] ?? [] ), 0, 20 ) as $index => $row ) {
			$row = is_array( $row ) ? $row : [];
			$key = substr( sanitize_key( (string) ( $row['key'] ?? '' ) ), 0, 70 );
			if ( $key === '' ) { $key = 'option_' . ( $index + 1 ); }
			$base_key = $key; $suffix = 2;
			while ( isset( $option_keys[ $key ] ) ) { $key = substr( $base_key, 0, 66 ) . '_' . $suffix++; }
			$title = sanitize_text_field( (string) ( $row['title'] ?? '' ) );
			if ( $title === '' ) { continue; }
			$option_keys[ $key ] = true;
			$color = sanitize_hex_color( (string) ( $row['color'] ?? '' ) );
			if ( ! $color ) { $color = $previous_option_colors[ $key ] ?? '#b8893e'; }
			$product_id = absint( $row['product_id'] ?? ( $previous_options_by_key[ $key ]['product_id'] ?? 0 ) );
			$legacy_price = $this->parse_money( $row['price'] ?? ( $previous_options_by_key[ $key ]['price'] ?? 0 ) );
			$product_price = 0.0;
			if ( $product_id > 0 && function_exists( 'wc_get_product' ) ) {
				$product = wc_get_product( $product_id );
				if ( ! $product || 'publish' !== get_post_status( $product_id ) || (string) get_post_meta( $product_id, '_sn_enabled', true ) !== '1' ) {
					return new WP_Error( 'sn_dot_option_product', 'محصول انتخاب‌شده برای گزینه «' . $title . '» معتبر یا فعال شبکه فروش نیست.' );
				}
				$product_price = max( 0, (float) $product->get_price() );
			}
			$config['options'][] = [
				'key' => $key,
				'title' => $title,
				'content' => $this->normalize_option_document_content( (string) ( $row['content'] ?? '' ) ),
				'color' => $color,
				'product_id' => $product_id,
				'price' => $product_id > 0 ? $product_price : $legacy_price,
				'active' => ! empty( $row['active'] ) ? 1 : 0,
			];
		}

		// Customer-facing selection-page copy and destination links are editable
		// independently from case/payment snapshots. Preserve existing values if
		// an older cached settings form is submitted without these newer fields.
		if ( isset( $raw['customer_ui'] ) && is_array( $raw['customer_ui'] ) ) {
			$ui_raw = $raw['customer_ui'];
			$config['customer_ui'] = [
				'hero_title' => sanitize_text_field( (string) ( $ui_raw['hero_title'] ?? '' ) ),
				'hero_subtitle' => sanitize_text_field( (string) ( $ui_raw['hero_subtitle'] ?? '' ) ),
				'result_description' => sanitize_textarea_field( (string) ( $ui_raw['result_description'] ?? '' ) ),
				'about_url' => esc_url_raw( (string) ( $ui_raw['about_url'] ?? '' ) ),
				'faq_url' => esc_url_raw( (string) ( $ui_raw['faq_url'] ?? '' ) ),
				'contact_url' => esc_url_raw( (string) ( $ui_raw['contact_url'] ?? '' ) ),
			];
		} else {
			$config['customer_ui'] = (array) ( $previous_config['customer_ui'] ?? $config['customer_ui'] );
		}

		// Supervisor inclusion is no longer configurable: every supervisor and
		// their HR branch participate in regular Dot Flow. Retain the old map only
		// as a read-only product-identification fallback for products created before
		// the explicit `assessment` product type existed.
		$config['product_supervisors'] = (array) ( $previous_config['product_supervisors'] ?? [] );

		$sms_raw = isset( $raw['sms'] ) && is_array( $raw['sms'] ) ? $raw['sms'] : [];
		$config['sms'] = [
			'access_pattern' => sanitize_text_field( (string) ( $sms_raw['access_pattern'] ?? '' ) ),
			'access_template' => sanitize_textarea_field( (string) ( $sms_raw['access_template'] ?? '' ) ),
			// Preserve the gift pattern when an older cached settings form is
			// submitted without the fields introduced in 2.0.8.
			'gift_access_pattern' => sanitize_text_field( (string) ( $sms_raw['gift_access_pattern'] ?? ( $previous_config['sms']['gift_access_pattern'] ?? '' ) ) ),
			'gift_access_template' => sanitize_textarea_field( (string) ( $sms_raw['gift_access_template'] ?? ( $previous_config['sms']['gift_access_template'] ?? $this->default_config()['sms']['gift_access_template'] ) ) ),
			'payment_pattern' => sanitize_text_field( (string) ( $sms_raw['payment_pattern'] ?? '' ) ),
			'payment_template' => sanitize_textarea_field( (string) ( $sms_raw['payment_template'] ?? '' ) ),
			'reject_pattern' => sanitize_text_field( (string) ( $sms_raw['reject_pattern'] ?? '' ) ),
			'reject_template' => sanitize_textarea_field( (string) ( $sms_raw['reject_template'] ?? '' ) ),
			// Preserve reminder settings when an older cached settings form is submitted.
			'followup_pattern' => sanitize_text_field( (string) ( $sms_raw['followup_pattern'] ?? ( $previous_config['sms']['followup_pattern'] ?? '' ) ) ),
			'followup_template' => sanitize_textarea_field( (string) ( $sms_raw['followup_template'] ?? ( $previous_config['sms']['followup_template'] ?? $this->default_config()['sms']['followup_template'] ) ) ),
		];

		if ( $config['enabled'] ) {
			if ( ! $config['nominal_credits'] || in_array( '', $config['digit_map'], true ) ) { return new WP_Error( 'sn_dot_credits', 'مبالغ اعتبار و نگاشت هر ۱۰ رقم پایانی موبایل باید کامل باشد.' ); }
			$active_options = array_values( array_filter( $config['options'], static fn( $row ) => ! empty( $row['active'] ) ) );
			if ( ! $active_options ) { return new WP_Error( 'sn_dot_options', 'حداقل یک گزینه فعال تعریف کنید.' ); }
			foreach ( $active_options as $active_option ) {
				if ( absint( $active_option['product_id'] ?? 0 ) < 1 ) {
					return new WP_Error( 'sn_dot_option_product_required', 'برای گزینه فعال «' . (string) ( $active_option['title'] ?? '' ) . '» محصول ووکامرس اشتراک را انتخاب کنید.' );
				}
				if ( (float) ( $active_option['price'] ?? 0 ) <= 0 ) {
					return new WP_Error( 'sn_dot_option_price', 'برای گزینه فعال «' . (string) ( $active_option['title'] ?? '' ) . '» یک محصول ووکامرس دارای قیمت انتخاب کنید.' );
				}
			}
			foreach ( [ 'access', 'gift_access', 'payment', 'reject' ] as $sms_type ) {
				$pattern = trim( (string) ( $config['sms'][ $sms_type . '_pattern' ] ?? '' ) );
				$template = trim( (string) ( $config['sms'][ $sms_type . '_template' ] ?? '' ) );
				if ( $pattern === '' && $template === '' ) {
					return new WP_Error( 'sn_dot_sms_' . $sms_type, 'برای هر رویداد پیامک، حداقل پترن یا متن جایگزین تعریف کنید.' );
				}
				if ( $requested_sms_provider !== 'faraz' && $template === '' ) {
					return new WP_Error( 'sn_dot_sms_template_' . $sms_type, 'برای سرویس‌دهنده فعلی، متن جایگزین همه پیامک‌های دات باید تکمیل باشد.' );
				}
			}
		}

		update_option( self::TEST_SMS_MODE_OPTION, $config['sms_test_mode'] ? '1' : '0', false );
		update_option( self::CONFIG_OPTION, $config, false );
		if ( empty( $config['sms_test_mode'] ) ) {
			// A panel may still be open when the switch is turned off. Clearing the
			// known viewers prevents an old queued preview from surviving the change.
			$this->clear_test_sms_queue_for_user( get_current_user_id() );
			$this->clear_test_sms_queue_for_user( absint( $previous_config['sms_test_viewer_user_id'] ?? 0 ) );
			// Test previews are not real deliveries. Release the probe so the
			// shutdown worker can send them after all provider settings are saved.
			delete_transient( 'sn_dot_due_sms_probe' );
		}
		return true;
	}

	public function save_marketing_settings_from_request( array $request ) {
		if ( empty( $request['sn_dot_marketing_settings_present'] ) ) { return true; }
		$raw = isset( $request['sn_dot_marketing'] ) && is_array( $request['sn_dot_marketing'] ) ? wp_unslash( $request['sn_dot_marketing'] ) : [];
		$config = $this->default_marketing_config();
		$config['enabled'] = ! empty( $raw['enabled'] ) ? 1 : 0;
		$config['product_id'] = absint( $raw['product_id'] ?? 0 );
		$config['sms_delay_minutes'] = min( 10080, max( 0, absint( $raw['sms_delay_minutes'] ?? 30 ) ) );
		$config['otp_enabled'] = ! empty( $raw['otp_enabled'] ) ? 1 : 0;
		$config['online_requires_finance'] = array_key_exists( 'online_requires_finance', $raw ) ? ( ! empty( $raw['online_requires_finance'] ) ? 1 : 0 ) : 1;

		$valid_supervisor_ids = array_map( static fn( $user ) => (int) $user->ID, $this->supervisors() );
		$requested_supervisors = array_values( array_unique( array_filter( array_map( 'absint', (array) ( $raw['supervisor_ids'] ?? [] ) ) ) ) );
		$config['supervisor_ids'] = array_values( array_intersect( $requested_supervisors, $valid_supervisor_ids ) );

		if ( $config['enabled'] ) {
			if ( empty( $this->config()['enabled'] ) ) {
				return new WP_Error( 'sn_dot_marketing_dot_disabled', 'برای فعال‌سازی مارکتینگ دات فلو، ابتدا خود «دات ستینگ» را فعال کنید.' );
			}
			if ( $config['product_id'] < 1 || ! function_exists( 'wc_get_product' ) ) {
				return new WP_Error( 'sn_dot_marketing_product', 'یک محصول ثابت معتبر برای مارکتینگ دات فلو انتخاب کنید.' );
			}
			$product = wc_get_product( $config['product_id'] );
			if ( ! $product || get_post_status( $config['product_id'] ) !== 'publish' || (string) get_post_meta( $config['product_id'], '_sn_enabled', true ) !== '1' ) {
				return new WP_Error( 'sn_dot_marketing_product', 'محصول انتخاب‌شده باید منتشرشده و برای شبکه فروش فعال باشد.' );
			}
			if ( (float) $product->get_price() <= 0 ) {
				return new WP_Error( 'sn_dot_marketing_product_price', 'قیمت محصول ثابت مارکتینگ باید بیشتر از صفر باشد.' );
			}
			if ( ! $config['supervisor_ids'] ) {
				return new WP_Error( 'sn_dot_marketing_supervisors', 'حداقل یک سرپرست برای مارکتینگ دات فلو انتخاب کنید.' );
			}
			foreach ( $config['supervisor_ids'] as $supervisor_id ) {
				if ( $this->marketing_seller_for_supervisor( (int) $supervisor_id ) < 1 ) {
					return new WP_Error( 'sn_dot_marketing_seller', 'ساخت فروشنده سیستمی مارکتینگ برای یکی از سرپرست‌ها انجام نشد.' );
				}
			}
		}

		// Additional landing forms. Each form has its own product and supervisor route.
		$forms = [];
		$raw_forms = isset( $request['sn_dot_marketing_forms'] ) && is_array( $request['sn_dot_marketing_forms'] ) ? wp_unslash( $request['sn_dot_marketing_forms'] ) : [];
		foreach ( $raw_forms as $raw_form ) {
			if ( ! is_array( $raw_form ) ) { continue; }
			$form_id = sanitize_key( (string) ( $raw_form['id'] ?? '' ) );
			if ( $form_id === '' || $form_id === 'default' ) { continue; }
			if ( isset( $forms[ $form_id ] ) ) { return new WP_Error( 'sn_dot_marketing_form_duplicate', 'شناسه فرم «' . $form_id . '» تکراری است.' ); }
			$title = trim( sanitize_text_field( (string) ( $raw_form['title'] ?? $form_id ) ) );
			$product_id = absint( $raw_form['product_id'] ?? 0 );
			$form_supervisors = array_values( array_intersect( array_values( array_unique( array_filter( array_map( 'absint', (array) ( $raw_form['supervisor_ids'] ?? [] ) ) ) ) ), $valid_supervisor_ids ) );
			$enabled = ! empty( $raw_form['enabled'] ) ? 1 : 0;
			if ( $enabled ) {
				if ( $product_id < 1 || ! function_exists( 'wc_get_product' ) ) { return new WP_Error( 'sn_dot_marketing_form_product', 'برای فرم «' . ( $title ?: $form_id ) . '» یک محصول معتبر انتخاب کنید.' ); }
				$form_product = wc_get_product( $product_id );
				if ( ! $form_product || get_post_status( $product_id ) !== 'publish' || (string) get_post_meta( $product_id, '_sn_enabled', true ) !== '1' || (float) $form_product->get_price() <= 0 ) { return new WP_Error( 'sn_dot_marketing_form_product', 'محصول فرم «' . ( $title ?: $form_id ) . '» معتبر یا فعال نیست.' ); }
				if ( ! $form_supervisors ) { return new WP_Error( 'sn_dot_marketing_form_supervisors', 'برای فرم «' . ( $title ?: $form_id ) . '» حداقل یک سرپرست انتخاب کنید.' ); }
			}
			if ( $enabled ) {
				foreach ( $form_supervisors as $form_supervisor ) {
					if ( $this->marketing_seller_for_supervisor( (int) $form_supervisor ) < 1 ) { return new WP_Error( 'sn_dot_marketing_form_seller', 'فروشنده سیستمی فرم «' . ( $title ?: $form_id ) . '» ساخته نشد.' ); }
				}
			}
			$forms[ $form_id ] = [ 'id' => $form_id, 'title' => $title ?: $form_id, 'enabled' => $enabled, 'product_id' => $product_id, 'supervisor_ids' => $form_supervisors ];
		}
		$has_enabled_custom_form = (bool) array_filter( $forms, static fn( $form ) => ! empty( $form['enabled'] ) );
		if ( $has_enabled_custom_form && empty( $this->config()['enabled'] ) ) {
			return new WP_Error( 'sn_dot_marketing_custom_dot_disabled', 'برای فعال‌سازی فرم‌های مستقل مارکتینگ، ابتدا خود «دات ستینگ» را فعال کنید.' );
		}

		$previous = $this->marketing_config();
		$previous_forms = get_option( 'sn_dot_marketing_forms', [] );
		$previous_forms = is_array( $previous_forms ) ? $previous_forms : [];
		update_option( self::MARKETING_CONFIG_OPTION, $config, false );
		update_option( 'sn_dot_marketing_forms', $forms, false );
		if ( array_values( (array) $previous['supervisor_ids'] ) !== array_values( (array) $config['supervisor_ids'] ) || count( $config['supervisor_ids'] ) <= 1 ) {
			update_option( self::MARKETING_RR_OPTION, 0, false );
		}
		foreach ( $forms as $form_id => $form ) {
			$old_supervisors = array_values( array_map( 'intval', (array) ( $previous_forms[ $form_id ]['supervisor_ids'] ?? [] ) ) );
			$new_supervisors = array_values( array_map( 'intval', (array) ( $form['supervisor_ids'] ?? [] ) ) );
			if ( $old_supervisors !== $new_supervisors || count( $new_supervisors ) <= 1 ) {
				update_option( $this->marketing_rr_option_name( (string) $form_id ), 0, false );
			}
		}
		foreach ( array_diff( array_keys( $previous_forms ), array_keys( $forms ) ) as $removed_form_id ) {
			delete_option( $this->marketing_rr_option_name( (string) $removed_form_id ) );
		}
		return true;
	}

	private function supervisors(): array {
		global $wpdb;
		$out = [];
		$profiles = $wpdb->prefix . 'sn_hr_profiles';
		$positions = $wpdb->prefix . 'sn_hr_positions';
		if ( $this->table_exists( $profiles ) && $this->table_exists( $positions ) ) {
			$ids = $wpdb->get_col( "SELECT hp.user_id FROM {$profiles} hp INNER JOIN {$positions} pos ON pos.id=hp.position_id WHERE pos.slug='supervisor' AND hp.is_active=1 ORDER BY hp.user_id" );
			foreach ( (array) $ids as $id ) { $out[ (int) $id ] = get_user_by( 'id', (int) $id ); }
		}
		foreach ( get_users( [ 'role' => 'sn_supervisor', 'number' => 2000, 'fields' => 'all' ] ) as $user ) { $out[ (int) $user->ID ] = $user; }
		return array_values( array_filter( $out, static fn( $user ) => $user instanceof WP_User ) );
	}

	public function render_settings_panel(): void {
		$config = $this->config();
		$products = class_exists( 'SN_Helpers' ) ? SN_Helpers::get_sn_products() : [];
		$report_counts = $this->dot_report_counts();
		$credits = (array) $config['nominal_credits'];
		$options = (array) $config['options'];
		$product_options_html = '<option value="0" data-price="0">— انتخاب محصول اشتراک —</option>';
		foreach ( $products as $dot_product ) {
			$product_options_html .= '<option value="' . (int) $dot_product['id'] . '" data-price="' . esc_attr( (string) (float) $dot_product['price'] ) . '">' . esc_html( (string) $dot_product['name'] . ' (#' . (int) $dot_product['id'] . ')' ) . '</option>';
		}
		$sms_examples = [
			'access' => "%customer_name% عزیز، اعتبارسنجی شما تأیید شد.\nرمز اختصاصی: %access_code%\nاین دسترسی اختصاصی برای شماست.\n%access_link%",
			'gift_access' => "%customer_name% عزیز، هزینه اعتبارسنجی شما از طرف شرکت به‌صورت هدیه پرداخت شده است.\nبرای مشاهده نتیجه و انتخاب گزینه وارد شوید:\n%access_link%\nرمز اختصاصی: %access_code%",
			'payment' => "%customer_name% عزیز، لینک %payment_type% برای «%option_title%» به مبلغ %amount% تومان:\n%invoice_url%",
			'reject' => "%seller_name% عزیز، پرداخت اعتبارسنجی مشتری %customer_name% با شماره %customer_phone% رد شد.\nکد فاکتور: %invoice_code%\nدلیل: %reason%",
			'followup' => "%staff_name% عزیز، زمان تماس مجدد با %customer_name% (%customer_phone%) برای پرونده #%case_id% رسیده است.\nموعد: %followup_at%",
		];
		?>
		<input type="hidden" name="sn_dot_settings_present" value="1">
		<div class="sn-dot-admin-section">
			<div class="sn-dot-admin-hero">
				<div><span class="sn-dot-admin-kicker">DOT SETTINGS</span><h2>فلو اعتبارسنجی و تبدیل</h2><p>مدیریت یکپارچه محصول اعتبارسنجی، اعتبار نمایشی، انتخاب مشتری، پیامک و تبدیل.</p></div>
				<div class="sn-dot-admin-states">
					<div class="sn-dot-admin-state <?php echo ! empty( $config['enabled'] ) ? 'is-active' : ''; ?>"><span></span><?php echo ! empty( $config['enabled'] ) ? 'فلو فعال است' : 'فلو غیرفعال است'; ?></div>
					<?php if ( ! empty( $config['sms_test_mode'] ) ) : ?><div class="sn-dot-admin-state is-test"><span></span>پیامک آزمایشی فعال</div><?php endif; ?>
				</div>
			</div>
			<div class="sn-dot-admin-note"><strong>سازگاری با سیستم فعلی</strong><span>کاربران و ارتباط سرپرست/نیرو از HR قدیمی خوانده می‌شوند و محصولات عادی همچنان از فلوهای قبلی عبور می‌کنند.</span></div>
			<div class="sn-dot-admin-subsection" style="margin-top:20px">
				<h3>گزارش‌های دات فلو معمولی</h3>
				<p class="description">خروجی لیدها، مسیر ورود و فاکتور اولیه را نشان می‌دهد. خروجی انتخاب و پرداخت فقط پرونده‌های ایجادشده را با گزینه انتخابی، تبدیل‌کننده و تمام وضعیت‌های پرداخت ارائه می‌کند.</p>
				<p><a class="button button-primary" href="<?php echo esc_url( $this->dot_report_export_url( 'regular', 'leads' ) ); ?>">خروجی لیدهای دات معمولی (<?php echo esc_html( number_format_i18n( (int) $report_counts['regular_leads'] ) ); ?>)</a> <a class="button" href="<?php echo esc_url( $this->dot_report_export_url( 'regular', 'outcomes' ) ); ?>">خروجی انتخاب و پرداخت (<?php echo esc_html( number_format_i18n( (int) $report_counts['regular_outcomes'] ) ); ?>)</a></p>
			</div>
			<table class="form-table">
				<tr><th>فعال‌سازی فلو</th><td><label><input type="checkbox" name="sn_dot[enabled]" value="1" <?php checked( ! empty( $config['enabled'] ) ); ?>> فعال</label></td></tr>
				<tr class="sn-dot-test-mode-row"><th>پیامک آزمایشی فلو</th><td><input type="hidden" name="sn_dot_sms_test_mode" value="0"><label class="sn-dot-test-toggle"><input type="checkbox" name="sn_dot_sms_test_mode" value="1" <?php checked( ! empty( $config['sms_test_mode'] ) ); ?>><span><strong>نمایش آزمایشی روی صفحه</strong><small>فقط پیامک‌های همین فلو به‌صورت پاپ‌آپ کوچک نمایش داده شوند.</small></span></label><div class="sn-dot-test-warning"><strong>مخصوص دمو و تست</strong><span>در حالت فعال، پیامک واقعی این فلو به سرویس‌دهنده ارسال نمی‌شود. متن نهایی، شماره مقصد، رمز و لینک فقط برای مدیری که این حالت را ذخیره کرده و افراد مرتبط همان پرونده در پنل CRM نمایش داده می‌شود؛ پیامک سایر بخش‌ها هیچ تغییری نمی‌کند.</span></div></td></tr>
				<tr><th>زمان ارسال پیامک ورود</th><td><div class="sn-dot-delay-control"><input id="sn-dot-sms-delay" type="number" min="0" max="10080" name="sn_dot[sms_delay_minutes]" value="<?php echo esc_attr( (string) $config['sms_delay_minutes'] ); ?>"><span>دقیقه پس از پرداخت موفق</span></div><div class="sn-dot-delay-presets" aria-label="انتخاب سریع زمان"><button type="button" class="button" data-sn-delay="0">فوری</button><button type="button" class="button" data-sn-delay="15">۱۵ دقیقه</button><button type="button" class="button" data-sn-delay="30">۳۰ دقیقه</button><button type="button" class="button" data-sn-delay="60">۱ ساعت</button><button type="button" class="button" data-sn-delay="120">۲ ساعت</button></div><p class="description">از ۰ تا ۱۰٬۰۸۰ دقیقه قابل تنظیم است. اگر تأیید مالی دیرتر از این زمان انجام شود، پیامک بلافاصله پس از تأیید مالی ارسال می‌شود.</p></td></tr>
				<tr><th>لینک ورود مشتری</th><td><span class="sn-dot-rule-chip">بدون تاریخ انقضا</span><p class="description">لینک و رمز اختصاصی مشتری با گذشت زمان منقضی نمی‌شوند و ارسال مجدد نیز همان دسترسی پایدار را ایجاد می‌کند.</p></td></tr>
				<tr><th>تأیید مالی پرداخت‌ها</th><td><span class="sn-dot-rule-chip">الزامی و غیرقابل دور زدن</span><p class="description">پرداخت اولیه اعتبارسنجی و تمام پرداخت‌های کامل، پیش‌پرداخت و تسویه—چه درگاهی و چه کارت‌به‌کارت—پس از ثبت موفق وارد صف تأیید مالی می‌شوند.</p></td></tr>
			</table>

			<h3>متن و لینک صفحه انتخاب مشتری</h3>
			<p class="description">متن‌های این بخش مستقیماً در صفحه نتیجه و انتخاب مشتری نمایش داده می‌شوند. لینک‌های خالی در صفحه به‌صورت دکمه غیرفعال دیده می‌شوند تا زمانی که مقصدشان تعیین شود.</p>
			<table class="form-table sn-dot-customer-ui-settings">
				<tr><th>عنوان باکس اول</th><td><input class="large-text" type="text" name="sn_dot[customer_ui][hero_title]" value="<?php echo esc_attr( (string) ( $config['customer_ui']['hero_title'] ?? '' ) ); ?>"></td></tr>
				<tr><th>زیرعنوان باکس اول</th><td><input class="large-text" type="text" name="sn_dot[customer_ui][hero_subtitle]" value="<?php echo esc_attr( (string) ( $config['customer_ui']['hero_subtitle'] ?? '' ) ); ?>"></td></tr>
				<tr><th>متن توضیحات اعتبار</th><td><textarea class="large-text" rows="7" name="sn_dot[customer_ui][result_description]"><?php echo esc_textarea( (string) ( $config['customer_ui']['result_description'] ?? '' ) ); ?></textarea><p class="description">خط‌های جدید در همین کادر در صفحه مشتری نیز حفظ می‌شوند.</p></td></tr>
				<tr><th>لینک «درباره بیاوین»</th><td><input class="large-text code" type="url" dir="ltr" name="sn_dot[customer_ui][about_url]" value="<?php echo esc_attr( (string) ( $config['customer_ui']['about_url'] ?? '' ) ); ?>" placeholder="https://..."></td></tr>
				<tr><th>لینک «سوالات متداول»</th><td><input class="large-text code" type="url" dir="ltr" name="sn_dot[customer_ui][faq_url]" value="<?php echo esc_attr( (string) ( $config['customer_ui']['faq_url'] ?? '' ) ); ?>" placeholder="https://..."></td></tr>
				<tr><th>لینک «راه‌های ارتباطی بیاوین»</th><td><input class="large-text code" type="url" dir="ltr" name="sn_dot[customer_ui][contact_url]" value="<?php echo esc_attr( (string) ( $config['customer_ui']['contact_url'] ?? '' ) ); ?>" placeholder="https://..."></td></tr>
			</table>

			<h3>حوزه سرپرست‌ها</h3>
			<div class="sn-dot-admin-note"><strong>همه سرپرست‌ها مشمول‌اند</strong><span>برای محصولات نوع «اعتبارسنجی» نیازی به انتخاب سرپرست نیست. هر فاکتور پس از تسویه و تأیید مالی، بر اساس زنجیره HR فروشنده به صف سرپرست مستقیم او می‌رود.</span></div>

			<h3>اعتبارها</h3>
			<table class="widefat striped" id="sn-dot-credits"><thead><tr><th>کلید</th><th>عنوان مدیریتی</th><th>مبلغ (تومان)</th><th></th></tr></thead><tbody>
			<?php foreach ( $credits as $i => $credit ) : ?><tr><td><input name="sn_dot[nominal_credits][<?php echo esc_attr( (string) $i ); ?>][key]" value="<?php echo esc_attr( (string) $credit['key'] ); ?>" dir="ltr"></td><td><input class="regular-text" name="sn_dot[nominal_credits][<?php echo esc_attr( (string) $i ); ?>][label]" value="<?php echo esc_attr( (string) $credit['label'] ); ?>"></td><td><input class="regular-text" name="sn_dot[nominal_credits][<?php echo esc_attr( (string) $i ); ?>][amount]" value="<?php echo esc_attr( (string) $credit['amount'] ); ?>" inputmode="numeric"></td><td><button type="button" class="button sn-dot-remove-credit">حذف</button></td></tr><?php endforeach; ?>
			</tbody></table><p><button type="button" class="button" id="sn-dot-add-credit">+ افزودن مبلغ اعتبار</button></p>

			<h3>شرط رقم آخر موبایل</h3>
			<table class="widefat striped"><thead><tr><th>رقم آخر</th><th>اعتبار نمایشی</th></tr></thead><tbody><?php foreach ( range( 0, 9 ) as $digit ) : ?><tr><td><strong><?php echo esc_html( (string) $digit ); ?></strong></td><td><select name="sn_dot[digit_map][<?php echo esc_attr( (string) $digit ); ?>]"><option value="">انتخاب کنید</option><?php foreach ( $credits as $credit ) : ?><option value="<?php echo esc_attr( (string) $credit['key'] ); ?>" <?php selected( (string) ( $config['digit_map'][ (string) $digit ] ?? '' ), (string) $credit['key'] ); ?>><?php echo esc_html( (string) $credit['label'] . ' — ' . number_format_i18n( (float) $credit['amount'] ) . ' تومان' ); ?></option><?php endforeach; ?></select></td></tr><?php endforeach; ?></tbody></table>

			<h3>اشتراک‌های قابل انتخاب مشتری</h3>
			<p class="description">هر گزینه به یک محصول واقعی ووکامرس متصل است و قیمت مستقیماً از همان محصول خوانده می‌شود. همان محصول باید در تب «پروژه‌ها» به‌عنوان اشتراک و همراه محتویاتش تعریف شود. عنوان، رنگ و توضیحات فقط نحوه نمایش در صفحه انتخاب را کنترل می‌کنند. ردیف‌های قدیمی بدون محصول تا زمان انتخاب محصول برای حفظ سوابق باقی می‌مانند.</p>
			<table class="widefat striped" id="sn-dot-options"><thead><tr><th>فعال</th><th>کلید</th><th>عنوان</th><th>رنگ</th><th>توضیحات اشتراک</th><th>محصول ووکامرس و قیمت</th><th></th></tr></thead><tbody>
			<?php foreach ( $options as $i => $option ) : $option_color = sanitize_hex_color( (string) ( $option['color'] ?? '' ) ) ?: '#b8893e'; ?>
				<tr><td><input type="checkbox" name="sn_dot[options][<?php echo esc_attr( (string) $i ); ?>][active]" value="1" <?php checked( ! empty( $option['active'] ) ); ?>></td><td><input name="sn_dot[options][<?php echo esc_attr( (string) $i ); ?>][key]" value="<?php echo esc_attr( (string) $option['key'] ); ?>" dir="ltr"></td><td><input name="sn_dot[options][<?php echo esc_attr( (string) $i ); ?>][title]" value="<?php echo esc_attr( (string) $option['title'] ); ?>"></td><td><input type="color" class="sn-dot-option-color" name="sn_dot[options][<?php echo esc_attr( (string) $i ); ?>][color]" value="<?php echo esc_attr( $option_color ); ?>" aria-label="رنگ <?php echo esc_attr( (string) $option['title'] ); ?>"></td><td><textarea rows="3" class="large-text code" name="sn_dot[options][<?php echo esc_attr( (string) $i ); ?>][content]" dir="rtl"><?php echo esc_textarea( (string) ( $option['content'] ?? '' ) ); ?></textarea></td><td><select class="sn-dot-option-product" name="sn_dot[options][<?php echo esc_attr( (string) $i ); ?>][product_id]"><option value="0" data-price="<?php echo esc_attr( (string) (float) ( $option['price'] ?? 0 ) ); ?>">— محصول قدیمی/انتخاب نشده —</option><?php foreach ( $products as $product ) : ?><option value="<?php echo esc_attr( (string) (int) $product['id'] ); ?>" data-price="<?php echo esc_attr( (string) (float) $product['price'] ); ?>" <?php selected( (int) ( $option['product_id'] ?? 0 ), (int) $product['id'] ); ?>><?php echo esc_html( (string) $product['name'] . ' (#' . (int) $product['id'] . ')' ); ?></option><?php endforeach; ?></select><input type="hidden" class="sn-dot-option-legacy-price" name="sn_dot[options][<?php echo esc_attr( (string) $i ); ?>][price]" value="<?php echo esc_attr( (string) (float) ( $option['price'] ?? 0 ) ); ?>"><small class="sn-dot-option-live-price"><?php echo esc_html( number_format_i18n( (float) ( $option['price'] ?? 0 ) ) ); ?> تومان</small></td><td><button type="button" class="button sn-dot-remove-option">حذف</button></td></tr>
			<?php endforeach; ?>
			</tbody></table><p><button type="button" class="button" id="sn-dot-add-option">+ افزودن گزینه</button></p>

			<h3>پترن‌ها و متن‌های پیامک</h3>
			<p class="description">در فیلد «پترن» کد پترن ساخته‌شده در پنل پیامک را وارد کنید. نمونه‌های زیر متن پیشنهادی برای ساخت همان پترن هستند؛ نام متغیرها باید دقیقاً حفظ شود.</p>
			<table class="form-table">
				<tr><th>پترن پیامک ورود</th><td><input class="regular-text" name="sn_dot[sms][access_pattern]" value="<?php echo esc_attr( (string) $config['sms']['access_pattern'] ); ?>"><p class="description">متغیرها: customer_name, access_code, code, access_note, access_link, link, nominal_credit, invoice_code</p><div class="sn-dot-sms-example"><span>نمونه متن پیشنهادی</span><code><?php echo esc_html( $sms_examples['access'] ); ?></code></div></td></tr>
				<tr><th>متن جایگزین ورود</th><td><textarea class="large-text" rows="4" name="sn_dot[sms][access_template]"><?php echo esc_textarea( (string) $config['sms']['access_template'] ); ?></textarea></td></tr>
				<tr><th>پترن «اعتبارسنجی هدیه»</th><td><input class="regular-text" name="sn_dot[sms][gift_access_pattern]" value="<?php echo esc_attr( (string) ( $config['sms']['gift_access_pattern'] ?? '' ) ); ?>"><p class="description">این پیامک فقط با دکمه «ارسال لینک انتخاب» سرپرست ارسال می‌شود. متغیرها: customer_name, access_code, code, access_note, gift_message, access_link, link, nominal_credit, invoice_code</p><div class="sn-dot-sms-example"><span>نمونه متن پیشنهادی</span><code><?php echo esc_html( $sms_examples['gift_access'] ); ?></code></div></td></tr>
				<tr><th>متن جایگزین «اعتبارسنجی هدیه»</th><td><textarea class="large-text" rows="5" name="sn_dot[sms][gift_access_template]"><?php echo esc_textarea( (string) ( $config['sms']['gift_access_template'] ?? '' ) ); ?></textarea><p class="description">اگر سرویس پیامک پترنی نباشد یا ارسال پترن ناموفق شود، این متن استفاده می‌شود.</p></td></tr>
				<tr><th>پترن لینک پرداخت</th><td><input class="regular-text" name="sn_dot[sms][payment_pattern]" value="<?php echo esc_attr( (string) $config['sms']['payment_pattern'] ); ?>"><p class="description">متغیرها: customer_name, option_title, amount, payment_type, invoice_url, remaining_amount</p><div class="sn-dot-sms-example"><span>نمونه متن پیشنهادی</span><code><?php echo esc_html( $sms_examples['payment'] ); ?></code></div></td></tr>
				<tr><th>متن جایگزین پرداخت</th><td><textarea class="large-text" rows="4" name="sn_dot[sms][payment_template]"><?php echo esc_textarea( (string) $config['sms']['payment_template'] ); ?></textarea></td></tr>
				<tr><th>پترن رد مالی فروشنده</th><td><input class="regular-text" name="sn_dot[sms][reject_pattern]" value="<?php echo esc_attr( (string) $config['sms']['reject_pattern'] ); ?>"><p class="description">متغیرها: seller_name, customer_name, customer_phone, invoice_code, reason</p><div class="sn-dot-sms-example"><span>نمونه متن پیشنهادی</span><code><?php echo esc_html( $sms_examples['reject'] ); ?></code></div></td></tr>
				<tr><th>متن جایگزین رد مالی</th><td><textarea class="large-text" rows="4" name="sn_dot[sms][reject_template]"><?php echo esc_textarea( (string) $config['sms']['reject_template'] ); ?></textarea></td></tr>
				<tr><th>پترن یادآور تماس مجدد</th><td><input class="regular-text" name="sn_dot[sms][followup_pattern]" value="<?php echo esc_attr( (string) ( $config['sms']['followup_pattern'] ?? '' ) ); ?>"><p class="description">متغیرها: staff_name, customer_name, customer_phone, case_id, followup_at</p><div class="sn-dot-sms-example"><span>نمونه متن پیشنهادی</span><code><?php echo esc_html( $sms_examples['followup'] ); ?></code></div></td></tr>
				<tr><th>متن جایگزین یادآور تماس مجدد</th><td><textarea class="large-text" rows="4" name="sn_dot[sms][followup_template]"><?php echo esc_textarea( (string) ( $config['sms']['followup_template'] ?? '' ) ); ?></textarea><p class="description">در موعد انتخاب‌شده یک‌بار برای مسئول فعلی پرونده ارسال می‌شود؛ یادآور داخل پنل حتی در صورت خطای پیامک باقی می‌ماند.</p></td></tr>
			</table>
		</div>
		<script>
		(function(){
			var delayInput=document.getElementById('sn-dot-sms-delay');
			document.querySelectorAll('[data-sn-delay]').forEach(function(button){
				button.addEventListener('click',function(){
					if(!delayInput){return;}
					delayInput.value=String(button.getAttribute('data-sn-delay')||'0');
					delayInput.dispatchEvent(new Event('change',{bubbles:true}));
					delayInput.focus();
				});
			});
			var creditTable=document.querySelector('#sn-dot-credits tbody');
			var addCredit=document.getElementById('sn-dot-add-credit');
			var nextCreditIndex=0;
			function creditRows(){return creditTable?Array.prototype.slice.call(creditTable.querySelectorAll('tr')):[];}
			function creditData(){
				return creditRows().map(function(row){
					var key=row.querySelector('[name$="[key]"]'), label=row.querySelector('[name$="[label]"]'), amount=row.querySelector('[name$="[amount]"]');
					return {key:key?String(key.value||'').trim():'',label:label?String(label.value||'').trim():'',amount:amount?String(amount.value||'').trim():''};
				}).filter(function(item){return item.key!=='';});
			}
			function rebuildCreditChoices(replacements){
				var data=creditData();
				document.querySelectorAll('select[name^="sn_dot[digit_map]"]').forEach(function(select){
					var selected=String(select.value||'');
					if(replacements&&Object.prototype.hasOwnProperty.call(replacements,selected)){selected=replacements[selected];}
					while(select.firstChild){select.removeChild(select.firstChild);}
					select.appendChild(new Option('انتخاب کنید',''));
					data.forEach(function(item){select.appendChild(new Option((item.label||item.key)+' — '+(item.amount||'0')+' تومان',item.key));});
					select.value=selected;
				});
			}
			if(creditTable){
				creditTable.querySelectorAll('[name^="sn_dot[nominal_credits]"]').forEach(function(field){
					var match=String(field.name||'').match(/sn_dot\[nominal_credits\]\[(\d+)\]/);
					if(match){nextCreditIndex=Math.max(nextCreditIndex,Number(match[1])+1);}
				});
				creditRows().forEach(function(row){var key=row.querySelector('[name$="[key]"]');row.dataset.snCreditKey=key?String(key.value||''):'';});
				creditTable.addEventListener('input',function(e){
					var row=e.target.closest('tr');if(!row){return;}
					var keyField=row.querySelector('[name$="[key]"]'), oldKey=String(row.dataset.snCreditKey||''), newKey=keyField?String(keyField.value||'').trim():'';
					var replacements={};if(oldKey&&oldKey!==newKey){replacements[oldKey]=newKey;}
					row.dataset.snCreditKey=newKey;rebuildCreditChoices(replacements);
				});
			}
			if(addCredit&&creditTable){
				addCredit.addEventListener('click',function(){
					var i=nextCreditIndex++, number=creditRows().length+1, tr=document.createElement('tr');
					tr.innerHTML='<td><input name="sn_dot[nominal_credits]['+i+'][key]" value="credit_'+number+'" dir="ltr"></td><td><input class="regular-text" name="sn_dot[nominal_credits]['+i+'][label]" value="اعتبار '+number+'"></td><td><input class="regular-text" name="sn_dot[nominal_credits]['+i+'][amount]" inputmode="numeric"></td><td><button type="button" class="button sn-dot-remove-credit">حذف</button></td>';
					tr.dataset.snCreditKey='credit_'+number;creditTable.appendChild(tr);rebuildCreditChoices();
				});
			}
			var table=document.querySelector('#sn-dot-options tbody');
			var add=document.getElementById('sn-dot-add-option');
			var productOptionsHtml=<?php echo wp_json_encode( $product_options_html, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ); ?>;
			var nextIndex=0;
			if(table){
				table.querySelectorAll('[name^="sn_dot[options]"]').forEach(function(field){
					var match=String(field.name||'').match(/sn_dot\[options\]\[(\d+)\]/);
					if(match){nextIndex=Math.max(nextIndex,Number(match[1])+1);}
				});
			}
			if(add&&table){
				add.addEventListener('click',function(){
					var i=nextIndex++, number=table.querySelectorAll('tr').length+1, tr=document.createElement('tr');
					var palette=['#b8893e','#137b72','#27658c','#9a4c3c'], color=palette[(number-1)%palette.length];
					tr.innerHTML='<td><input type="checkbox" name="sn_dot[options]['+i+'][active]" value="1" checked></td><td><input name="sn_dot[options]['+i+'][key]" value="option_'+number+'" dir="ltr"></td><td><input name="sn_dot[options]['+i+'][title]" value="گزینه '+number+'"></td><td><input type="color" class="sn-dot-option-color" name="sn_dot[options]['+i+'][color]" value="'+color+'" aria-label="رنگ گزینه '+number+'"></td><td><textarea rows="3" class="large-text code" name="sn_dot[options]['+i+'][content]"></textarea></td><td><select class="sn-dot-option-product" name="sn_dot[options]['+i+'][product_id]">'+productOptionsHtml+'</select><input type="hidden" class="sn-dot-option-legacy-price" name="sn_dot[options]['+i+'][price]" value="0"><small class="sn-dot-option-live-price">۰ تومان</small></td><td><button type="button" class="button sn-dot-remove-option">حذف</button></td>';
					table.appendChild(tr);
				});
			}
			document.addEventListener('click',function(e){
				if(e.target&&e.target.classList.contains('sn-dot-remove-credit')){
					var creditRow=e.target.closest('tr');if(creditRow){creditRow.remove();rebuildCreditChoices();}
				}
				if(e.target&&e.target.classList.contains('sn-dot-remove-option')){
					var row=e.target.closest('tr');
					if(row){row.remove();}
				}
			});
			document.addEventListener('change',function(e){
				if(!e.target||!e.target.classList.contains('sn-dot-option-product')){return;}
				var row=e.target.closest('tr'), option=e.target.options[e.target.selectedIndex], price=Number(option&&option.dataset?option.dataset.price:0)||0;
				if(row){var hidden=row.querySelector('.sn-dot-option-legacy-price'), label=row.querySelector('.sn-dot-option-live-price');if(hidden){hidden.value=String(price);}if(label){label.textContent=new Intl.NumberFormat('fa-IR').format(price)+' تومان';}}
			});
		})();
		</script>
		<?php
	}

	private function marketing_lead_counts(): array {
		global $wpdb;
		$t = $this->tables();
		if ( ! $this->table_exists( $t['marketing_leads'] ) ) { return []; }
		$rows = $wpdb->get_results( "SELECT form_id,COUNT(*) total,SUM(status='payment_paid') paid,SUM(status IN ('payment_not_done','popup_cancelled','otp_cancelled','awaiting_payment','gateway_started','gateway_cancelled','gateway_failed')) abandoned,SUM(status='pending_finance') pending_finance FROM {$t['marketing_leads']} GROUP BY form_id", ARRAY_A ) ?: [];
		$out = [];
		foreach ( $rows as $row ) {
			$form_id = sanitize_key( (string) ( $row['form_id'] ?? 'default' ) ) ?: 'default';
			$out[ $form_id ] = [
				'total' => (int) ( $row['total'] ?? 0 ),
				'paid' => (int) ( $row['paid'] ?? 0 ),
				'abandoned' => (int) ( $row['abandoned'] ?? 0 ),
				'pending_finance' => (int) ( $row['pending_finance'] ?? 0 ),
			];
		}
		return $out;
	}

	private function marketing_export_url( string $form_id = 'all' ): string {
		$form_id = $form_id === 'all' ? 'all' : ( sanitize_key( $form_id ) ?: 'default' );
		$url = add_query_arg( [ 'action' => 'sn_dot_marketing_export_leads', 'form' => $form_id ], admin_url( 'admin-post.php' ) );
		return wp_nonce_url( $url, 'sn_dot_marketing_export_leads' );
	}

	private function marketing_lead_status_label( string $status ): string {
		return [
			'payment_not_done' => 'پرداخت انجام نشده',
			'popup_cancelled' => 'انصراف در پاپ‌آپ قبل از درگاه',
			'otp_cancelled' => 'انصراف در مرحله تأیید شماره',
			'awaiting_payment' => 'فاکتور ساخته شده / در انتظار پرداخت',
			'gateway_started' => 'هدایت به درگاه / پرداخت تکمیل نشده',
			'gateway_cancelled' => 'انصراف از درگاه پرداخت',
			'gateway_failed' => 'خطا یا پرداخت ناموفق در درگاه',
			'pending_finance' => 'در انتظار تأیید مالی',
			'payment_paid' => 'پرداخت تأیید شده',
			'payment_rejected' => 'پرداخت رد شده',
			'validation_gifted' => 'اعتبارسنجی هدیه / آماده انتخاب',
		][ $status ] ?? $status;
	}

	private function marketing_lead_event_label( string $event ): string {
		$labels = [
			'continue_clicked' => 'دکمه ادامه درخواست زده شد',
			'lead_saved' => 'لید و اطلاعات فرم ثبت شد',
			'invoice_created' => 'پیش‌فاکتور پرداخت ساخته شد',
			'invoice_reused' => 'پیش‌فاکتور قبلی همین درخواست استفاده شد',
			'invoice_failed' => 'ساخت پیش‌فاکتور ناموفق بود',
			'confirmation_opened' => 'پاپ‌آپ تأیید نمایش داده شد',
			'confirmation_accepted' => 'ادامه در پاپ‌آپ تأیید شد',
			'otp_requested' => 'کد تأیید ارسال شد',
			'otp_verified' => 'شماره موبایل تأیید شد',
			'otp_failed' => 'مرحله تأیید شماره ناموفق بود',
			'gateway_requested' => 'درخواست ساخت لینک درگاه ارسال شد',
			'rate_limited' => 'درخواست به‌علت محدودیت موقت متوقف شد',
			'form_error' => 'خطای پردازش فرم',
		];
		if ( isset( $labels[ $event ] ) ) { return $labels[ $event ]; }
		return $this->marketing_lead_status_label( $event );
	}

	private function marketing_funnel_stage_label( int $stage, string $status = '' ): string {
		if ( $status === 'payment_paid' ) { return 'پرداخت تأییدشده'; }
		if ( $status === 'validation_gifted' ) { return 'اعتبارسنجی هدیه‌شده'; }
		if ( $status === 'payment_rejected' ) { return 'پرداخت ردشده'; }
		if ( $status === 'gateway_cancelled' ) { return 'انصراف از درگاه'; }
		if ( $status === 'gateway_failed' ) { return 'خطا در درگاه'; }
		if ( $status === 'popup_cancelled' ) { return 'انصراف در پاپ‌آپ'; }
		if ( $status === 'otp_cancelled' ) { return 'انصراف در OTP'; }
		if ( $stage >= 100 ) { return 'تکمیل‌شده'; }
		if ( $stage >= 90 ) { return 'برگشت از درگاه / بررسی مالی'; }
		if ( $stage >= 80 ) { return 'وارد درگاه شده'; }
		if ( $stage >= 70 ) { return 'در حال دریافت لینک درگاه'; }
		if ( $stage >= 60 ) { return 'شماره موبایل تأییدشده'; }
		if ( $stage >= 50 ) { return 'تأیید پاپ‌آپ / OTP'; }
		if ( $stage >= 40 ) { return 'پاپ‌آپ تأیید نمایش داده شده'; }
		if ( $stage >= 30 ) { return 'پیش‌فاکتور ساخته شده'; }
		if ( $stage >= 20 ) { return 'لید ذخیره شده'; }
		if ( $stage >= 10 ) { return 'شروع درخواست'; }
		return 'بدون مرحله ثبت‌شده';
	}

	private function marketing_lead_event_reason_label( string $reason ): string {
		return [
			'cancel_button' => 'دکمه انصراف',
			'close_button' => 'دکمه بستن پاپ‌آپ',
			'backdrop' => 'کلیک بیرون پاپ‌آپ',
			'escape' => 'کلید Escape',
			'page_exit' => 'خروج یا بستن صفحه در زمان نمایش پاپ‌آپ',
			'gateway_request_failed' => 'عدم دریافت لینک از درگاه',
			'rate_limited' => 'محدودیت موقت تعداد درخواست',
			'gateway_rate_limit' => 'محدودیت موقت هنگام ورود به درگاه',
			'sms_failed' => 'ارسال پیامک ناموفق',
			'expired' => 'کد منقضی شده',
			'phone_mismatch' => 'عدم تطابق شماره موبایل',
			'form_mismatch' => 'عدم تطابق فرم',
			'attempt_limit' => 'عبور از سقف تلاش OTP',
			'invalid_code' => 'کد تأیید اشتباه',
			'proof_invalid' => 'اثبات تأیید شماره نامعتبر یا منقضی',
			'invalid_gateway' => 'درگاه نامعتبر',
			'payment_request_not_found' => 'درخواست پرداخت پیدا نشد',
			'payment_invoice_mismatch' => 'عدم تطابق درخواست و فاکتور',
			'payment_gateway_mismatch' => 'عدم تطابق درگاه',
			'stage_not_found' => 'مرحله پرداخت پیدا نشد',
			'payment_amount_mismatch' => 'عدم تطابق مبلغ پرداخت',
			'no_active_stage' => 'مرحله پرداخت فعالی وجود ندارد',
			'validation_gifted' => 'هزینه اعتبارسنجی توسط شرکت هدیه شده است',
			'dot_finance_hold' => 'خطا در ثبت انتظار تأیید مالی',
			'stage_finalize' => 'خطا در نهایی‌سازی پرداخت',
			'asanpardakht_cancelled' => 'لغو یا بازگشت ناقص از آسان‌پرداخت',
			'asanpardakht_verify_failed' => 'تأیید آسان‌پرداخت ناموفق بود',
			'payment_stage_mismatch' => 'عدم تطابق مرحله پرداخت',
			'zibal_cancelled' => 'لغو پرداخت زیبال',
			'zibal_verify_failed' => 'تأیید زیبال ناموفق بود',
			'zarinpal_callback_missing' => 'بازگشت ناقص از زرین‌پال',
			'zarinpal_cancelled' => 'لغو پرداخت زرین‌پال',
			'zarinpal_verify_failed' => 'تأیید زرین‌پال ناموفق بود',
			'legacy_lead_migration' => 'بازیابی از لید نسخه قدیمی',
			'legacy_invoice_backfill' => 'بازیابی از فاکتور نسخه قدیمی',
			'legacy_status_snapshot' => 'وضعیت موجود هنگام ارتقای افزونه',
			'supervisor_gift' => 'هدیه اعتبارسنجی توسط سرپرست',
		][ $reason ] ?? $reason;
	}

	private function marketing_csv_cell( $value ): string {
		$value = str_replace( "\0", '', (string) $value );
		return preg_match( '/^[=+\-@]/u', $value ) ? "'" . $value : $value;
	}

	/** Stream every Marketing Dot submission without an artificial row limit. */
	public function handle_marketing_export_leads(): void {
		if ( ! current_user_can( 'manage_options' ) ) { wp_die( 'دسترسی غیرمجاز است.', '', [ 'response' => 403 ] ); }
		check_admin_referer( 'sn_dot_marketing_export_leads' );
		global $wpdb;
		$t = $this->tables();
		if ( ! $this->table_exists( $t['marketing_leads'] ) ) { wp_die( 'جدول لیدهای مارکتینگ آماده نیست.', '', [ 'response' => 500 ] ); }
		$form_id = sanitize_key( wp_unslash( $_GET['form'] ?? 'all' ) );
		if ( $form_id === '' ) { $form_id = 'all'; }
		$filename = 'dot-marketing-leads-' . ( $form_id === 'all' ? 'all' : $form_id ) . '-' . SN_Helpers::tehran_format( 'Y-m-d-His' ) . '.csv';
		while ( ob_get_level() ) { ob_end_clean(); }
		nocache_headers();
		header( 'Content-Type: text/csv; charset=UTF-8' );
		header( 'Content-Disposition: attachment; filename="' . sanitize_file_name( $filename ) . '"' );
		$out = fopen( 'php://output', 'w' );
		if ( ! is_resource( $out ) ) { wp_die( 'ساخت خروجی ممکن نشد.', '', [ 'response' => 500 ] ); }
		fwrite( $out, "\xEF\xBB\xBF" );
		$headers = [ 'شناسه لید مارکتینگ','شناسه فرم','عنوان فرم','وضعیت اتصال کمپین','Campaign Attribution ID','Landing URL','Visitor ID','Session ID','First Source','First Medium','First Campaign','First Content','First Term','Last Source','Last Medium','Last Campaign','Last Content','Last Term','نام و نام خانوادگی','موبایل','کد ملی','استان','شهر','وضعیت لید مارکتینگ','مرحله فعلی قیف','شماره مرحله از ۱۰۰','آخرین رویداد','کلید آخرین رویداد','علت/جزئیات آخرین رویداد','زمان آخرین رویداد','زمان شروع درخواست','زمان ذخیره لید','زمان ساخت پیش‌فاکتور','زمان نمایش پاپ‌آپ','زمان تأیید پاپ‌آپ','زمان درخواست OTP','زمان تأیید OTP','زمان درخواست درگاه','زمان هدایت به درگاه','زمان بازگشت از درگاه','تاریخچه کامل رویدادها','تاریخ ثبت لید','تاریخ پرداخت','شناسه محصول','نام محصول','شناسه سرپرست','نام سرپرست','شناسه فروشنده سیستمی','نام فروشنده سیستمی','شناسه لید CRM','وضعیت سیستمی CRM','وضعیت پیگیری CRM','یادداشت CRM','شناسه فاکتور','کد فاکتور','مبلغ فاکتور','وضعیت فاکتور','وضعیت بررسی فاکتور','وضعیت پرداخت فاکتور','روش پرداخت','درگاه','تاریخ ساخت فاکتور','تاریخ پرداخت فاکتور','شناسه پرونده دات','وضعیت پرونده دات','اعتبار اسمی','گزینه انتخابی','هزینه گزینه','شناسه تبدیل‌کننده','نام تبدیل‌کننده','پرداخت تأییدشده گزینه','مانده گزینه','تاریخ تکمیل پرونده' ];
		fputcsv( $out, $headers );
		$last_id = 0;
		do {
			$where = 'm.id>%d';
			$args = [ $last_id ];
			if ( $form_id !== 'all' ) { $where .= ' AND m.form_id=%s'; $args[] = $form_id; }
			$sql = "SELECT m.*,m.invoice_created_at marketing_invoice_event_at,l.status crm_status,l.lead_status crm_lead_status,l.note crm_note,
				ca.visitor_id campaign_visitor_id,ca.session_id campaign_session_id,
				COALESCE(NULLIF(ca.first_touch_source,''),m.utm_source,'') first_touch_source,COALESCE(NULLIF(ca.first_touch_medium,''),m.utm_medium,'') first_touch_medium,COALESCE(NULLIF(ca.first_touch_campaign,''),m.utm_campaign,'') first_touch_campaign,COALESCE(NULLIF(ca.first_touch_content,''),m.utm_content,'') first_touch_content,COALESCE(NULLIF(ca.first_touch_term,''),m.utm_term,'') first_touch_term,
				COALESCE(NULLIF(ca.last_touch_source,''),m.utm_source,'') last_touch_source,COALESCE(NULLIF(ca.last_touch_medium,''),m.utm_medium,'') last_touch_medium,COALESCE(NULLIF(ca.last_touch_campaign,''),m.utm_campaign,'') last_touch_campaign,COALESCE(NULLIF(ca.last_touch_content,''),m.utm_content,'') last_touch_content,COALESCE(NULLIF(ca.last_touch_term,''),m.utm_term,'') last_touch_term,
				su.display_name supervisor_name,se.display_name seller_name,cv.display_name converter_name,
				i.invoice_code,i.product_price,i.status invoice_status,i.invoice_status invoice_review_status,i.payment_status invoice_payment_status,i.pay_method,i.payment_source,i.created_at invoice_created_at,i.paid_at invoice_paid_at,
				p.post_title product_name,dl.case_id,c.status dot_case_status,c.nominal_credit_amount,c.selected_option_title,c.selected_option_price,c.converter_id,c.paid_amount option_paid_amount,c.remaining_amount option_remaining_amount,c.completed_at
				FROM {$t['marketing_leads']} m
				LEFT JOIN {$wpdb->prefix}sn_campaign_attributions ca ON ca.subject_type='marketing_lead' AND ca.subject_id=m.id
				LEFT JOIN {$wpdb->prefix}sn_leads l ON l.id=m.crm_lead_id
				LEFT JOIN {$wpdb->users} su ON su.ID=m.supervisor_id
				LEFT JOIN {$wpdb->users} se ON se.ID=m.seller_id
				LEFT JOIN {$wpdb->prefix}sn_invoices i ON i.id=m.invoice_id
				LEFT JOIN {$wpdb->posts} p ON p.ID=m.product_id
				LEFT JOIN {$t['links']} dl ON dl.invoice_id=m.invoice_id AND dl.flow_kind='assessment_source'
				LEFT JOIN {$t['cases']} c ON c.id=dl.case_id
				LEFT JOIN {$wpdb->users} cv ON cv.ID=c.converter_id
				WHERE {$where} ORDER BY m.id ASC LIMIT 500";
			$rows = $wpdb->get_results( $wpdb->prepare( $sql, ...$args ), ARRAY_A ) ?: [];
			$event_history = [];
			if ( $rows && $this->table_exists( $t['marketing_events'] ) ) {
				$lead_ids = array_values( array_filter( array_map( 'absint', array_column( $rows, 'id' ) ) ) );
				if ( $lead_ids ) {
					$id_sql = implode( ',', $lead_ids );
					$events = $wpdb->get_results( "SELECT marketing_lead_id,event_type,event_status,reason,created_at FROM {$t['marketing_events']} WHERE marketing_lead_id IN ({$id_sql}) ORDER BY marketing_lead_id ASC,id ASC", ARRAY_A ) ?: [];
					foreach ( $events as $event_row ) {
						$lead_event_id = (int) $event_row['marketing_lead_id'];
						$event_label = $this->marketing_lead_event_label( sanitize_key( (string) $event_row['event_type'] ) );
						$reason_label = $this->marketing_lead_event_reason_label( (string) ( $event_row['reason'] ?? '' ) );
						$event_history[ $lead_event_id ][] = trim( (string) $event_row['created_at'] . ' — ' . $event_label . ( $reason_label !== '' ? ' (' . $reason_label . ')' : '' ) );
					}
				}
			}
			foreach ( $rows as $row ) {
				$last_id = (int) $row['id'];
				$last_event_key = sanitize_key( (string) ( $row['last_event'] ?: $row['status'] ) );
				$last_event_reason = (string) ( $row['last_event_reason'] ?? '' );
				$values = [
					$row['id'],$row['form_id'],$row['form_title'],$row['campaign_attribution_status'],$row['campaign_attribution_id'],$row['campaign_landing_url'],$row['campaign_visitor_id'],$row['campaign_session_id'],$row['first_touch_source'],$row['first_touch_medium'],$row['first_touch_campaign'],$row['first_touch_content'],$row['first_touch_term'],$row['last_touch_source'],$row['last_touch_medium'],$row['last_touch_campaign'],$row['last_touch_content'],$row['last_touch_term'],$row['customer_name'],$row['customer_phone'],$row['national_id'],$row['province'],$row['city'],$this->marketing_lead_status_label( (string) $row['status'] ),
					$this->marketing_funnel_stage_label( (int) ( $row['funnel_stage'] ?? 0 ), (string) $row['status'] ),(int) ( $row['funnel_stage'] ?? 0 ),$this->marketing_lead_event_label( $last_event_key ),$last_event_key,$this->marketing_lead_event_reason_label( $last_event_reason ),$row['last_event_at'],
					$row['application_started_at'] ?? '',$row['lead_saved_at'] ?? '',$row['marketing_invoice_event_at'] ?? '',$row['confirmation_opened_at'] ?? '',$row['confirmation_accepted_at'] ?? '',$row['otp_requested_at'] ?? '',$row['otp_verified_at'] ?? '',$row['gateway_requested_at'] ?? '',$row['gateway_entered_at'],$row['gateway_returned_at'],implode( ' | ', $event_history[ (int) $row['id'] ] ?? [] ),
					$row['created_at'],$row['paid_at'],$row['product_id'],$row['product_name'],$row['supervisor_id'],$row['supervisor_name'],$row['seller_id'],$row['seller_name'],$row['crm_lead_id'],$row['crm_status'],$row['crm_lead_status'],$row['crm_note'],$row['invoice_id'],$row['invoice_code'],$row['product_price'],$row['invoice_status'],$row['invoice_review_status'],$row['invoice_payment_status'],$row['pay_method'],$row['payment_source'],$row['invoice_created_at'],$row['invoice_paid_at'],$row['case_id'],$row['dot_case_status'],$row['nominal_credit_amount'],$row['selected_option_title'],$row['selected_option_price'],$row['converter_id'],$row['converter_name'],$row['option_paid_amount'],$row['option_remaining_amount'],$row['completed_at'],
				];
				fputcsv( $out, array_map( [ $this, 'marketing_csv_cell' ], $values ) );
			}
			if ( function_exists( 'flush' ) ) { flush(); }
		} while ( count( $rows ) === 500 );
		fclose( $out );
		exit;
	}

	private function dot_report_counts(): array {
		global $wpdb;
		$t = $this->tables();
		$counts = [
			'regular_leads' => 0,
			'regular_outcomes' => 0,
			'marketing_leads' => 0,
			'marketing_outcomes' => 0,
		];
		$invoice_table = $wpdb->prefix . 'sn_invoices';
		if ( ! $this->table_exists( $t['links'] ) || ! $this->table_exists( $t['cases'] ) || ! $this->table_exists( $t['marketing_leads'] ) || ! $this->table_exists( $invoice_table ) ) { return $counts; }
		$is_marketing = "(m.id IS NOT NULL OR (dl.marketing_form_id IS NOT NULL AND dl.marketing_form_id<>'') OR EXISTS (SELECT 1 FROM {$wpdb->usermeta} sm WHERE sm.user_id=i.seller_id AND sm.meta_key='_sn_dot_marketing_seller' AND sm.meta_value='1'))";
		$counts['regular_leads'] = (int) $wpdb->get_var(
			"SELECT COUNT(*) FROM {$t['links']} dl INNER JOIN {$invoice_table} i ON i.id=dl.invoice_id LEFT JOIN {$t['marketing_leads']} m ON m.invoice_id=dl.invoice_id WHERE dl.flow_kind='assessment_source' AND NOT {$is_marketing}"
		);
		$counts['marketing_leads'] = (int) $wpdb->get_var( "SELECT COUNT(*) FROM {$t['marketing_leads']}" );
		$counts['marketing_leads'] += (int) $wpdb->get_var(
			"SELECT COUNT(*) FROM {$t['links']} dl INNER JOIN {$invoice_table} i ON i.id=dl.invoice_id LEFT JOIN {$t['marketing_leads']} m ON m.invoice_id=dl.invoice_id WHERE dl.flow_kind='assessment_source' AND m.id IS NULL AND {$is_marketing}"
		);
		$case_counts = $wpdb->get_row(
			"SELECT
				SUM(CASE WHEN {$is_marketing} THEN 0 ELSE 1 END) regular_total,
				SUM(CASE WHEN {$is_marketing} THEN 1 ELSE 0 END) marketing_total
			FROM {$t['cases']} c
			INNER JOIN {$t['links']} dl ON dl.invoice_id=c.source_invoice_id AND dl.flow_kind='assessment_source'
			INNER JOIN {$invoice_table} i ON i.id=c.source_invoice_id
			LEFT JOIN {$t['marketing_leads']} m ON m.invoice_id=c.source_invoice_id",
			ARRAY_A
		);
		$counts['regular_outcomes'] = (int) ( $case_counts['regular_total'] ?? 0 );
		$counts['marketing_outcomes'] = (int) ( $case_counts['marketing_total'] ?? 0 );
		return $counts;
	}

	private function dot_report_export_url( string $scope, string $report ): string {
		$scope = in_array( $scope, [ 'regular', 'marketing' ], true ) ? $scope : 'regular';
		$report = in_array( $report, [ 'leads', 'outcomes' ], true ) ? $report : 'leads';
		$url = add_query_arg( [
			'action' => 'sn_dot_export_report',
			'scope' => $scope,
			'report' => $report,
		], admin_url( 'admin-post.php' ) );
		return wp_nonce_url( $url, 'sn_dot_export_report' );
	}

	private function open_dot_csv_output( string $filename, array $headers ) {
		while ( ob_get_level() ) { ob_end_clean(); }
		nocache_headers();
		header( 'Content-Type: text/csv; charset=UTF-8' );
		header( 'Content-Disposition: attachment; filename="' . sanitize_file_name( $filename ) . '"' );
		$out = fopen( 'php://output', 'w' );
		if ( ! is_resource( $out ) ) { wp_die( 'ساخت خروجی ممکن نشد.', '', [ 'response' => 500 ] ); }
		fwrite( $out, "\xEF\xBB\xBF" );
		fputcsv( $out, $headers );
		return $out;
	}

	/** Stream one of the four administrator reports without an artificial row limit. */
	public function handle_dot_export_report(): void {
		if ( ! current_user_can( 'manage_options' ) ) { wp_die( 'دسترسی غیرمجاز است.', '', [ 'response' => 403 ] ); }
		check_admin_referer( 'sn_dot_export_report' );
		$scope = sanitize_key( wp_unslash( $_GET['scope'] ?? '' ) );
		$report = sanitize_key( wp_unslash( $_GET['report'] ?? '' ) );
		if ( ! in_array( $scope, [ 'regular', 'marketing' ], true ) || ! in_array( $report, [ 'leads', 'outcomes' ], true ) ) {
			wp_die( 'نوع خروجی معتبر نیست.', '', [ 'response' => 400 ] );
		}
		$t = $this->tables();
		foreach ( [ 'links', 'cases', 'payments', 'marketing_leads' ] as $table_key ) {
			if ( ! $this->table_exists( $t[ $table_key ] ) ) { wp_die( 'جداول دات فلو آماده نیستند.', '', [ 'response' => 500 ] ); }
		}
		if ( $report === 'leads' ) {
			$this->stream_dot_lead_report( $scope );
		}
		$this->stream_dot_outcome_report( $scope );
	}

	private function stream_dot_lead_report( string $scope ): void {
		global $wpdb;
		$t = $this->tables();
		$campaign_attribution_table = $wpdb->prefix . 'sn_campaign_attributions';
		$has_campaign_attributions = $this->table_exists( $campaign_attribution_table );
		$campaign_attribution_join = $has_campaign_attributions
			? "LEFT JOIN {$campaign_attribution_table} ca ON ca.subject_type='marketing_lead' AND ca.subject_id=m.id"
			: '';
		$marketing_campaign_fields = $has_campaign_attributions
			? "COALESCE(NULLIF(m.campaign_attribution_status,''),CASE WHEN ca.id IS NULL THEN '' ELSE 'attributed' END) campaign_attribution_status,COALESCE(NULLIF(m.campaign_attribution_id,0),ca.id,0) campaign_attribution_id,m.campaign_landing_url,
				COALESCE(NULLIF(m.utm_source,''),NULLIF(ca.first_touch_source,''),NULLIF(ca.last_touch_source,''),'') utm_source,
				COALESCE(NULLIF(m.utm_medium,''),NULLIF(ca.first_touch_medium,''),NULLIF(ca.last_touch_medium,''),'') utm_medium,
				COALESCE(NULLIF(m.utm_campaign,''),NULLIF(ca.first_touch_campaign,''),NULLIF(ca.last_touch_campaign,''),'') utm_campaign,
				COALESCE(NULLIF(m.utm_content,''),NULLIF(ca.first_touch_content,''),NULLIF(ca.last_touch_content,''),'') utm_content,
				COALESCE(NULLIF(m.utm_term,''),NULLIF(ca.first_touch_term,''),NULLIF(ca.last_touch_term,''),'') utm_term,"
			: "m.campaign_attribution_status,m.campaign_attribution_id,m.campaign_landing_url,m.utm_source,m.utm_medium,m.utm_campaign,m.utm_content,m.utm_term,";
		$recovered_campaign_fields = $has_campaign_attributions
			? "CASE WHEN ca.id IS NULL THEN '' ELSE 'attributed' END campaign_attribution_status,COALESCE(ca.id,0) campaign_attribution_id,'' campaign_landing_url,
				COALESCE(NULLIF(ca.first_touch_source,''),NULLIF(ca.last_touch_source,''),'') utm_source,
				COALESCE(NULLIF(ca.first_touch_medium,''),NULLIF(ca.last_touch_medium,''),'') utm_medium,
				COALESCE(NULLIF(ca.first_touch_campaign,''),NULLIF(ca.last_touch_campaign,''),'') utm_campaign,
				COALESCE(NULLIF(ca.first_touch_content,''),NULLIF(ca.last_touch_content,''),'') utm_content,
				COALESCE(NULLIF(ca.first_touch_term,''),NULLIF(ca.last_touch_term,''),'') utm_term,"
			: "'' campaign_attribution_status,0 campaign_attribution_id,'' campaign_landing_url,'' utm_source,'' utm_medium,'' utm_campaign,'' utm_content,'' utm_term,";
		$headers = [
			'ردیف منبع','نوع ورودی','شناسه لید مارکتینگ','شناسه فرم','عنوان فرم','شناسه لید CRM',
				'وضعیت اتصال کمپین','شناسه انتساب کمپین','صفحه ورود','منبع UTM','رسانه UTM','کمپین UTM','محتوای UTM','کلیدواژه UTM',
			'نام و نام خانوادگی','موبایل','کد ملی','استان','شهر','وضعیت لید مارکتینگ','آخرین رویداد','کلید آخرین رویداد','علت/جزئیات آخرین رویداد',
			'زمان آخرین رویداد','زمان هدایت به درگاه','زمان بازگشت از درگاه','وضعیت CRM','وضعیت پیگیری CRM',
			'شناسه محصول','نام محصول','شناسه فاکتور منبع','کد فاکتور','مبلغ فاکتور','وضعیت فاکتور','وضعیت بررسی فاکتور','وضعیت پرداخت فاکتور',
			'روش پرداخت','درگاه','شناسه فروشنده','نام فروشنده','شناسه سرپرست','نام سرپرست','نیاز به تأیید مالی',
			'تأخیر پیامک ثبت‌شده (دقیقه)','زمان تأیید پرداخت','تاریخ ساخت فاکتور','تاریخ پرداخت فاکتور','تاریخ ثبت لید',
		];
		$filename = 'dot-' . $scope . '-leads-' . SN_Helpers::tehran_format( 'Y-m-d-His' ) . '.csv';
		$out = $this->open_dot_csv_output( $filename, $headers );
		$last_id = 0;
		do {
			if ( $scope === 'marketing' ) {
					$sql = "SELECT m.id row_id,m.id marketing_lead_id,m.form_id,m.form_title,m.status marketing_status,m.last_event,m.last_event_reason,m.last_event_at,m.gateway_entered_at,m.gateway_returned_at,m.crm_lead_id,
						{$marketing_campaign_fields}
					m.customer_name,m.customer_phone,m.national_id,m.province,m.city,m.product_id,m.supervisor_id,m.seller_id,m.created_at lead_created_at,
					l.status crm_status,l.lead_status crm_lead_status,
					i.id invoice_id,i.invoice_code,i.product_price,i.status invoice_status,i.invoice_status invoice_review_status,i.payment_status invoice_payment_status,
					i.pay_method,i.payment_source,i.created_at invoice_created_at,i.paid_at invoice_paid_at,
					p.post_title product_name,se.display_name seller_name,su.display_name supervisor_name,
					dl.requires_finance,dl.sms_delay_minutes,dl.payment_verified_at
				FROM {$t['marketing_leads']} m
					LEFT JOIN {$wpdb->prefix}sn_leads l ON l.id=m.crm_lead_id
					{$campaign_attribution_join}
				LEFT JOIN {$wpdb->prefix}sn_invoices i ON i.id=m.invoice_id
				LEFT JOIN {$wpdb->posts} p ON p.ID=m.product_id
				LEFT JOIN {$wpdb->users} se ON se.ID=m.seller_id
				LEFT JOIN {$wpdb->users} su ON su.ID=m.supervisor_id
				LEFT JOIN {$t['links']} dl ON dl.invoice_id=m.invoice_id AND dl.flow_kind='assessment_source'
				WHERE m.id>%d ORDER BY m.id ASC LIMIT 500";
			} else {
				$sql = "SELECT dl.id row_id,0 marketing_lead_id,'' form_id,'' form_title,'' marketing_status,'' last_event,'' last_event_reason,NULL last_event_at,NULL gateway_entered_at,NULL gateway_returned_at,i.lead_id crm_lead_id,
					'' campaign_attribution_status,NULL campaign_attribution_id,'' campaign_landing_url,'' utm_source,'' utm_medium,'' utm_campaign,'' utm_content,'' utm_term,
					i.customer_name,i.customer_phone,l.national_id,i.province,i.city,i.product_id,dl.supervisor_id,i.seller_id,dl.created_at lead_created_at,
					l.status crm_status,l.lead_status crm_lead_status,
					i.id invoice_id,i.invoice_code,i.product_price,i.status invoice_status,i.invoice_status invoice_review_status,i.payment_status invoice_payment_status,
					i.pay_method,i.payment_source,i.created_at invoice_created_at,i.paid_at invoice_paid_at,
					p.post_title product_name,se.display_name seller_name,su.display_name supervisor_name,
					dl.requires_finance,dl.sms_delay_minutes,dl.payment_verified_at
				FROM {$t['links']} dl
				INNER JOIN {$wpdb->prefix}sn_invoices i ON i.id=dl.invoice_id
				LEFT JOIN {$t['marketing_leads']} m ON m.invoice_id=dl.invoice_id
				LEFT JOIN {$wpdb->prefix}sn_leads l ON l.id=i.lead_id
				LEFT JOIN {$wpdb->posts} p ON p.ID=i.product_id
				LEFT JOIN {$wpdb->users} se ON se.ID=i.seller_id
				LEFT JOIN {$wpdb->users} su ON su.ID=dl.supervisor_id
				WHERE dl.id>%d AND dl.flow_kind='assessment_source' AND m.id IS NULL
				AND (dl.marketing_form_id IS NULL OR dl.marketing_form_id='')
				AND NOT EXISTS (SELECT 1 FROM {$wpdb->usermeta} sm WHERE sm.user_id=i.seller_id AND sm.meta_key='_sn_dot_marketing_seller' AND sm.meta_value='1')
				ORDER BY dl.id ASC LIMIT 500";
			}
			$rows = $wpdb->get_results( $wpdb->prepare( $sql, $last_id ), ARRAY_A ) ?: [];
			foreach ( $rows as $row ) {
				$last_id = (int) $row['row_id'];
				$last_event_key = sanitize_key( (string) ( $row['last_event'] ?: $row['marketing_status'] ) );
				$last_event_reason = (string) ( $row['last_event_reason'] ?? '' );
				$values = [
					$row['row_id'],$scope === 'marketing' ? 'مارکتینگ دات' : 'دات معمولی',$row['marketing_lead_id'],$row['form_id'],$row['form_title'],$row['crm_lead_id'],
					$row['campaign_attribution_status'],$row['campaign_attribution_id'],$row['campaign_landing_url'],$row['utm_source'],$row['utm_medium'],$row['utm_campaign'],$row['utm_content'],$row['utm_term'],
					$row['customer_name'],$row['customer_phone'],$row['national_id'],$row['province'],$row['city'],
					$row['marketing_status'] !== '' ? $this->marketing_lead_status_label( (string) $row['marketing_status'] ) : '',
					$last_event_key !== '' ? $this->marketing_lead_event_label( $last_event_key ) : '',$last_event_key,$this->marketing_lead_event_reason_label( $last_event_reason ),$row['last_event_at'],$row['gateway_entered_at'],$row['gateway_returned_at'],
					$row['crm_status'],$row['crm_lead_status'],
					$row['product_id'],$row['product_name'],$row['invoice_id'],$row['invoice_code'],$row['product_price'],$row['invoice_status'],$row['invoice_review_status'],$row['invoice_payment_status'],
					$row['pay_method'],$row['payment_source'],$row['seller_id'],$row['seller_name'],$row['supervisor_id'],$row['supervisor_name'],
					(int) ( $row['requires_finance'] ?? 0 ),$row['sms_delay_minutes'],$row['payment_verified_at'],$row['invoice_created_at'],$row['invoice_paid_at'],$row['lead_created_at'],
				];
				fputcsv( $out, array_map( [ $this, 'marketing_csv_cell' ], $values ) );
			}
			if ( function_exists( 'flush' ) ) { flush(); }
		} while ( count( $rows ) === 500 );
		// A previous build did not persist every Marketing Dot submission in the
		// dedicated audit table. Include every remaining marketing source invoice
		// so the CSV reconciles with the supervisor invoice list.
		if ( $scope === 'marketing' ) {
			$last_id = 0;
			do {
					$sql = "SELECT dl.id row_id,0 marketing_lead_id,dl.marketing_form_id link_form_id,l.marketing_form_id lead_form_id,
						i.lead_id crm_lead_id,
						{$recovered_campaign_fields}
						i.customer_name,i.customer_phone,l.national_id,i.province,i.city,i.product_id,dl.supervisor_id,i.seller_id,dl.created_at lead_created_at,
					l.status crm_status,l.lead_status crm_lead_status,
					i.id invoice_id,i.invoice_code,i.product_price,i.status invoice_status,i.invoice_status invoice_review_status,i.payment_status invoice_payment_status,
					i.payment_workflow_status,i.pay_method,i.payment_source,i.created_at invoice_created_at,i.paid_at invoice_paid_at,
					p.post_title product_name,se.display_name seller_name,su.display_name supervisor_name,
					dl.requires_finance,dl.sms_delay_minutes,dl.payment_verified_at
				FROM {$t['links']} dl
				INNER JOIN {$wpdb->prefix}sn_invoices i ON i.id=dl.invoice_id
				LEFT JOIN {$t['marketing_leads']} m ON m.invoice_id=dl.invoice_id
					LEFT JOIN {$wpdb->prefix}sn_leads l ON l.id=i.lead_id
						" . ( $has_campaign_attributions ? "LEFT JOIN {$campaign_attribution_table} ca ON ca.subject_type='lead' AND ca.subject_id=i.lead_id" : '' ) . "
				LEFT JOIN {$wpdb->posts} p ON p.ID=i.product_id
				LEFT JOIN {$wpdb->users} se ON se.ID=i.seller_id
				LEFT JOIN {$wpdb->users} su ON su.ID=dl.supervisor_id
				WHERE dl.id>%d AND dl.flow_kind='assessment_source' AND m.id IS NULL
				AND (
					(dl.marketing_form_id IS NOT NULL AND dl.marketing_form_id<>'')
					OR EXISTS (SELECT 1 FROM {$wpdb->usermeta} sm WHERE sm.user_id=i.seller_id AND sm.meta_key='_sn_dot_marketing_seller' AND sm.meta_value='1')
				)
				ORDER BY dl.id ASC LIMIT 500";
				$rows = $wpdb->get_results( $wpdb->prepare( $sql, $last_id ), ARRAY_A ) ?: [];
				foreach ( $rows as $row ) {
					$last_id = (int) $row['row_id'];
						$identity = $this->marketing_form_identity_for_product(
						(int) $row['product_id'],
						(string) ( $row['link_form_id'] ?: $row['lead_form_id'] )
					);
					$marketing_status = $this->marketing_status_from_invoice_record( [
						'status' => $row['invoice_status'],
						'invoice_status' => $row['invoice_review_status'],
						'payment_status' => $row['invoice_payment_status'],
						'payment_workflow_status' => $row['payment_workflow_status'],
						'paid_at' => $row['invoice_paid_at'],
					] );
					$values = [
							$row['row_id'],'مارکتینگ دات — بازیابی از فاکتور',0,$identity['id'],$identity['title'],$row['crm_lead_id'],
							$row['campaign_attribution_status'],$row['campaign_attribution_id'],$row['campaign_landing_url'],$row['utm_source'],$row['utm_medium'],$row['utm_campaign'],$row['utm_content'],$row['utm_term'],
						$row['customer_name'],$row['customer_phone'],$row['national_id'],$row['province'],$row['city'],
						$this->marketing_lead_status_label( $marketing_status ),$this->marketing_lead_event_label( $marketing_status ),$marketing_status,'بازیابی از فاکتور قدیمی',($row['invoice_paid_at'] ?: $row['invoice_created_at']),'','',$row['crm_status'],$row['crm_lead_status'],
						$row['product_id'],$row['product_name'],$row['invoice_id'],$row['invoice_code'],$row['product_price'],$row['invoice_status'],$row['invoice_review_status'],$row['invoice_payment_status'],
						$row['pay_method'],$row['payment_source'],$row['seller_id'],$row['seller_name'],$row['supervisor_id'],$row['supervisor_name'],
						(int) ( $row['requires_finance'] ?? 0 ),$row['sms_delay_minutes'],$row['payment_verified_at'],$row['invoice_created_at'],$row['invoice_paid_at'],$row['lead_created_at'],
					];
					fputcsv( $out, array_map( [ $this, 'marketing_csv_cell' ], $values ) );
				}
				if ( function_exists( 'flush' ) ) { flush(); }
			} while ( count( $rows ) === 500 );
		}
		fclose( $out );
		exit;
	}

	private function stream_dot_outcome_report( string $scope ): void {
		global $wpdb;
		$t = $this->tables();
		$headers = [
			'شناسه پرونده','نوع ورودی','شناسه لید مارکتینگ','شناسه فرم','عنوان فرم','شناسه فاکتور منبع','کد فاکتور منبع',
			'وضعیت پرداخت فاکتور منبع','تاریخ پرداخت فاکتور منبع','وضعیت لید مارکتینگ','آخرین رویداد مارکتینگ','زمان آخرین رویداد مارکتینگ',
			'شناسه محصول','نام محصول','نام مشتری','موبایل','استان','شهر','شناسه فروشنده','نام فروشنده','شناسه سرپرست','نام سرپرست',
			'وضعیت پرونده','کلید وضعیت پرونده','اعتبار اسمی','موعد پیامک ورود','زمان ارسال پیامک','زمان مشاهده مشتری',
			'کلید گزینه انتخابی','عنوان گزینه انتخابی','هزینه گزینه','زمان انتخاب','شناسه تبدیل‌کننده','نام تبدیل‌کننده',
			'وضعیت تماس','یادداشت تبدیل‌کننده','روش پرداخت انتخابی','پیش‌پرداخت','مبلغ تأییدشده','مانده','زمان تکمیل',
			'تعداد لینک‌های پرداخت','تعداد پرداخت تأییدشده','تعداد در انتظار','تعداد ردشده','جمع مراحل تأییدشده',
			'شناسه آخرین پرداخت','نوع آخرین پرداخت','وضعیت آخرین پرداخت','مبلغ آخرین پرداخت','زمان پرداخت آخرین مرحله','تاریخ ایجاد پرونده',
		];
		$filename = 'dot-' . $scope . '-selections-payments-' . SN_Helpers::tehran_format( 'Y-m-d-His' ) . '.csv';
		$out = $this->open_dot_csv_output( $filename, $headers );
		$last_id = 0;
		$is_marketing = "(m.id IS NOT NULL OR (dl.marketing_form_id IS NOT NULL AND dl.marketing_form_id<>'') OR EXISTS (SELECT 1 FROM {$wpdb->usermeta} sm WHERE sm.user_id=i.seller_id AND sm.meta_key='_sn_dot_marketing_seller' AND sm.meta_value='1'))";
		$scope_condition = $scope === 'marketing' ? $is_marketing : 'NOT ' . $is_marketing;
		do {
			$sql = "SELECT c.*,m.id marketing_lead_id,COALESCE(NULLIF(m.form_id,''),NULLIF(dl.marketing_form_id,''),'') form_id,m.form_title,
				m.status marketing_status,m.last_event marketing_last_event,m.last_event_at marketing_last_event_at,
				i.invoice_code,i.payment_status source_payment_status,i.paid_at source_paid_at,p.post_title product_name,se.display_name seller_name,su.display_name supervisor_name,cv.display_name converter_name,
				COALESCE(ps.payment_count,0) payment_count,COALESCE(ps.paid_payment_count,0) paid_payment_count,
				COALESCE(ps.pending_payment_count,0) pending_payment_count,COALESCE(ps.rejected_payment_count,0) rejected_payment_count,
				COALESCE(ps.paid_payment_amount,0) paid_payment_amount,
				lp.id latest_payment_id,lp.payment_type latest_payment_type,lp.status latest_payment_status,
				lp.requested_amount latest_payment_amount,lp.paid_at latest_payment_paid_at
			FROM {$t['cases']} c
			INNER JOIN {$t['links']} dl ON dl.invoice_id=c.source_invoice_id AND dl.flow_kind='assessment_source'
			INNER JOIN {$wpdb->prefix}sn_invoices i ON i.id=c.source_invoice_id
			LEFT JOIN {$t['marketing_leads']} m ON m.invoice_id=c.source_invoice_id
			LEFT JOIN {$wpdb->posts} p ON p.ID=c.source_product_id
			LEFT JOIN {$wpdb->users} se ON se.ID=c.seller_id
			LEFT JOIN {$wpdb->users} su ON su.ID=c.supervisor_id
			LEFT JOIN {$wpdb->users} cv ON cv.ID=c.converter_id
			LEFT JOIN (
				SELECT case_id,COUNT(*) payment_count,
					SUM(CASE WHEN status='paid' THEN 1 ELSE 0 END) paid_payment_count,
					SUM(CASE WHEN status IN ('pending','pending_finance') THEN 1 ELSE 0 END) pending_payment_count,
					SUM(CASE WHEN status='rejected' THEN 1 ELSE 0 END) rejected_payment_count,
					SUM(CASE WHEN status='paid' THEN requested_amount ELSE 0 END) paid_payment_amount,
					MAX(id) latest_payment_id
				FROM {$t['payments']} GROUP BY case_id
			) ps ON ps.case_id=c.id
			LEFT JOIN {$t['payments']} lp ON lp.id=ps.latest_payment_id
			WHERE c.id>%d AND {$scope_condition} ORDER BY c.id ASC LIMIT 500";
			$rows = $wpdb->get_results( $wpdb->prepare( $sql, $last_id ), ARRAY_A ) ?: [];
			foreach ( $rows as $row ) {
				$last_id = (int) $row['id'];
				$latest_status = (string) ( $row['latest_payment_status'] ?? '' );
				if ( $scope === 'marketing' && (string) $row['form_title'] === '' ) {
					$identity = $this->marketing_form_identity_for_product( (int) $row['source_product_id'], (string) $row['form_id'] );
					$row['form_id'] = (string) $identity['id'];
					$row['form_title'] = (string) $identity['title'];
				}
				$values = [
					$row['id'],$scope === 'marketing' ? 'مارکتینگ دات' : 'دات معمولی',$row['marketing_lead_id'],$row['form_id'],$row['form_title'],$row['source_invoice_id'],$row['invoice_code'],
					$row['source_payment_status'],$row['source_paid_at'],$row['marketing_status'] ? $this->marketing_lead_status_label( (string) $row['marketing_status'] ) : '',$row['marketing_last_event'] ? $this->marketing_lead_event_label( (string) $row['marketing_last_event'] ) : '',$row['marketing_last_event_at'],
					$row['source_product_id'],$row['product_name'],$row['customer_name'],$row['customer_phone'],$row['province'],$row['city'],$row['seller_id'],$row['seller_name'],$row['supervisor_id'],$row['supervisor_name'],
					$this->status_label( (string) $row['status'] ),$row['status'],$row['nominal_credit_amount'],$row['sms_due_at'],$row['sms_sent_at'] ?: $row['sms_previewed_at'],$row['customer_viewed_at'],
					$row['selected_option_key'],$row['selected_option_title'],$row['selected_option_price'],$row['selected_at'],$row['converter_id'],$row['converter_name'],
					$this->contact_status_label( (string) ( $row['converter_contact_status'] ?: 'new' ) ),$row['converter_note'],$row['payment_mode'],$row['deposit_amount'],$row['paid_amount'],$row['remaining_amount'],$row['completed_at'],
					$row['payment_count'],$row['paid_payment_count'],$row['pending_payment_count'],$row['rejected_payment_count'],$row['paid_payment_amount'],
					$row['latest_payment_id'],$row['latest_payment_type'] ? $this->payment_type_label( (string) $row['latest_payment_type'] ) : '',$latest_status !== '' ? $this->payment_status_label( $latest_status ) : '',$row['latest_payment_amount'],$row['latest_payment_paid_at'],$row['created_at'],
				];
				fputcsv( $out, array_map( [ $this, 'marketing_csv_cell' ], $values ) );
			}
			if ( function_exists( 'flush' ) ) { flush(); }
		} while ( count( $rows ) === 500 );
		fclose( $out );
		exit;
	}

	public function render_marketing_settings_panel(): void {
		$config = $this->marketing_config();
		$products = class_exists( 'SN_Helpers' ) ? SN_Helpers::get_sn_products() : [];
		$supervisors = $this->supervisors();
		$selected_supervisors = array_map( 'intval', (array) $config['supervisor_ids'] );
		$lead_counts = $this->marketing_lead_counts();
		$report_counts = $this->dot_report_counts();
		$saved_forms = get_option( 'sn_dot_marketing_forms', [] );
		$saved_forms = is_array( $saved_forms ) ? $saved_forms : [];
		$active_form_count = ! empty( $config['enabled'] ) ? 1 : 0;
		$active_form_count += count( array_filter( $saved_forms, static fn( $form ) => is_array( $form ) && ! empty( $form['enabled'] ) ) );
		?>
		<input type="hidden" name="sn_dot_marketing_settings_present" value="1">
		<div class="sn-dot-admin-section sn-dot-marketing-admin">
			<div class="sn-dot-admin-hero">
				<div><span class="sn-dot-admin-kicker">DOT MARKETING</span><h2>مارکتینگ دات فلو</h2><p>ورودی مستقیم لندینگ تبلیغاتی به دات فلو با محصول ثابت و پرداخت فقط از درگاه.</p></div>
				<div class="sn-dot-admin-states"><div class="sn-dot-admin-state <?php echo $active_form_count > 0 ? 'is-active' : ''; ?>"><span></span><?php echo $active_form_count > 0 ? esc_html( number_format_i18n( $active_form_count ) . ' فرم فعال' ) : 'همه فرم‌ها غیرفعال‌اند'; ?></div></div>
			</div>
			<div class="sn-dot-admin-note"><strong>مسیر این فرم</strong><span>ثبت اطلاعات مشتری → ساخت فاکتور Source دات → انتقال مستقیم به درگاه → تعیین تکلیف پرداخت طبق گزینه «تأیید پرداخت درگاهی» → ارسال پیامک اعتبارسنجی در زمان تنظیم‌شده → انتخاب مشتری → صف همان سرپرست → تخصیص به تبدیل‌کننده.</span></div>
			<div class="sn-dot-admin-subsection" style="margin-top:20px">
				<h3>گزارش‌های مارکتینگ دات فلو</h3>
				<p class="description">خروجی لیدها تمام ثبت‌ها—از انصراف داخل پاپ‌آپ تا هدایت، لغو یا خطای درگاه—را با فرم، محصول، سرپرست و زمان آخرین رویداد نشان می‌دهد. خروجی انتخاب و پرداخت، گزینه مشتری و ادامه مسیر پرونده تا پرداخت و تکمیل را ارائه می‌کند.</p>
				<p><a class="button button-primary" href="<?php echo esc_url( $this->dot_report_export_url( 'marketing', 'leads' ) ); ?>">خروجی لیدهای مارکتینگ (<?php echo esc_html( number_format_i18n( (int) $report_counts['marketing_leads'] ) ); ?>)</a> <a class="button" href="<?php echo esc_url( $this->dot_report_export_url( 'marketing', 'outcomes' ) ); ?>">خروجی انتخاب و پرداخت (<?php echo esc_html( number_format_i18n( (int) $report_counts['marketing_outcomes'] ) ); ?>)</a></p>
			</div>
			<table class="form-table">
				<tr><th>فعال‌سازی</th><td><label><input type="checkbox" name="sn_dot_marketing[enabled]" value="1" <?php checked( ! empty( $config['enabled'] ) ); ?>> فرم مارکتینگ فعال باشد</label></td></tr>
				<tr><th>شورتکد فرم</th><td><code style="font-size:14px;direction:ltr;display:inline-block;padding:8px 12px">[sn_dot_marketing_form]</code><p class="description">این شورتکد را در هر برگه، المنتور یا لندینگ وردپرس قرار دهید.</p></td></tr>
				<tr><th>محصول ثابت</th><td><select name="sn_dot_marketing[product_id]" style="min-width:340px"><option value="0">انتخاب محصول</option><?php foreach ( $products as $product ) : $pid = absint( $product['id'] ?? 0 ); ?><option value="<?php echo esc_attr( (string) $pid ); ?>" <?php selected( (int) $config['product_id'], $pid ); ?>><?php echo esc_html( (string) ( $product['name'] ?? ( '#' . $pid ) ) ); ?> — <?php echo esc_html( number_format_i18n( (float) ( $product['price'] ?? 0 ) ) . ' تومان' ); ?></option><?php endforeach; ?></select><p class="description">بازدیدکننده امکان تغییر محصول یا مبلغ را ندارد؛ هر دو فقط سمت سرور از همین تنظیم خوانده می‌شوند.</p></td></tr>
				<tr><th>سرپرست‌های این فلو</th><td><select name="sn_dot_marketing[supervisor_ids][]" multiple size="7" style="min-width:360px"><?php foreach ( $supervisors as $supervisor ) : ?><option value="<?php echo esc_attr( (string) $supervisor->ID ); ?>" <?php selected( in_array( (int) $supervisor->ID, $selected_supervisors, true ) ); ?>><?php echo esc_html( (string) $supervisor->display_name . ' (#' . $supervisor->ID . ')' ); ?></option><?php endforeach; ?></select><p class="description">اگر فقط یک سرپرست انتخاب شود، تمام ورودی‌ها برای همان سرپرست ثبت می‌شوند. اگر چند سرپرست انتخاب شوند، ورودی‌ها دقیقاً به‌صورت صفی و Round‑Robin یکی‌یکی بین آن‌ها توزیع می‌شوند. برای هر سرپرست یک فروشنده سیستمی اختصاصی «مارکتینگ دات فلو» نگهداری می‌شود تا فاکتورهای لندینگ با فروشندگان واقعی مخلوط نشوند.</p></td></tr>
				<tr><th>تأیید شماره با OTP</th><td><select name="sn_dot_marketing[otp_enabled]"><option value="0" <?php selected( empty( $config['otp_enabled'] ) ); ?>>غیرفعال</option><option value="1" <?php selected( ! empty( $config['otp_enabled'] ) ); ?>>فعال</option></select><p class="description">در حالت فعال، مشتری ابتدا اطلاعات را در پاپ‌آپ تأیید می‌کند، کد ۶ رقمی برای همان شماره ارسال می‌شود و فقط پس از تأیید کد به درگاه می‌رود. در حالت غیرفعال، پاپ‌آپ تأیید نمایش داده می‌شود و سپس مستقیماً به درگاه منتقل می‌شود.</p></td></tr>
				<tr><th>تأیید پرداخت درگاهی</th><td><select name="sn_dot_marketing[online_requires_finance]"><option value="1" <?php selected( ! empty( $config['online_requires_finance'] ) ); ?>>ارسال به تأیید مالی</option><option value="0" <?php selected( empty( $config['online_requires_finance'] ) ); ?>>تأیید مستقیم پس از Verify درگاه</option></select><p class="description">این انتخاب برای تراکنش‌های جدید این فرم ذخیره می‌شود. در حالت مستقیم، Verify موفق درگاه پرداخت را نهایی و پرونده دات را بدون انتظار برای واحد مالی ایجاد می‌کند.</p></td></tr>
				<tr><th>زمان ارسال SMS اعتبارسنجی</th><td><div class="sn-dot-delay-control"><input type="number" min="0" max="10080" name="sn_dot_marketing[sms_delay_minutes]" value="<?php echo esc_attr( (string) $config['sms_delay_minutes'] ); ?>"><span>دقیقه پس از پرداخت موفق درگاه</span></div><p class="description">اگر تأیید مالی انتخاب شده باشد و واحد مالی بعد از زمان تعیین‌شده پرداخت را تأیید کند، SMS بلافاصله پس از تأیید مالی ارسال می‌شود.</p></td></tr>
			</table>
			<div class="sn-dot-admin-subsection" style="margin-top:24px">
				<h3>فرم‌های مستقل لندینگ</h3>
				<p class="description">برای هر لندینگ یک شناسه یکتا، محصول و سرپرست انتخاب کنید. سپس شورتکد همان ردیف را در برگه لندینگ قرار دهید.</p>
				<table class="widefat striped" id="sn-dot-marketing-forms-table"><thead><tr><th>فعال</th><th>شناسه فرم</th><th>عنوان</th><th>محصول</th><th>سرپرست</th><th>شورتکد</th><th>لیدها / خروجی</th><th></th></tr></thead><tbody>
				<?php $form_row = 0; foreach ( $saved_forms as $saved_form ) : $saved_form = is_array( $saved_form ) ? $saved_form : []; $form_row++; $saved_id = sanitize_key( (string) ( $saved_form['id'] ?? '' ) ); ?>
				<tr><td><input type="checkbox" name="sn_dot_marketing_forms[<?php echo esc_attr( (string) $form_row ); ?>][enabled]" value="1" <?php checked( ! empty( $saved_form['enabled'] ) ); ?>></td><td><input type="text" name="sn_dot_marketing_forms[<?php echo esc_attr( (string) $form_row ); ?>][id]" value="<?php echo esc_attr( $saved_id ); ?>" pattern="[a-zA-Z0-9_-]+" required></td><td><input type="text" name="sn_dot_marketing_forms[<?php echo esc_attr( (string) $form_row ); ?>][title]" value="<?php echo esc_attr( (string) ( $saved_form['title'] ?? '' ) ); ?>"></td><td><select name="sn_dot_marketing_forms[<?php echo esc_attr( (string) $form_row ); ?>][product_id]"><option value="0">انتخاب محصول</option><?php foreach ( $products as $product ) : $pid = absint( $product['id'] ?? 0 ); ?><option value="<?php echo esc_attr( (string) $pid ); ?>" <?php selected( absint( $saved_form['product_id'] ?? 0 ), $pid ); ?>><?php echo esc_html( (string) ( $product['name'] ?? ( '#' . $pid ) ) ); ?> — <?php echo esc_html( number_format_i18n( (float) ( $product['price'] ?? 0 ) ) . ' تومان' ); ?></option><?php endforeach; ?></select></td><td><select name="sn_dot_marketing_forms[<?php echo esc_attr( (string) $form_row ); ?>][supervisor_ids][]" multiple size="3"><?php foreach ( $supervisors as $supervisor ) : ?><option value="<?php echo esc_attr( (string) $supervisor->ID ); ?>" <?php selected( in_array( (int) $supervisor->ID, array_map( 'intval', (array) ( $saved_form['supervisor_ids'] ?? [] ) ), true ) ); ?>><?php echo esc_html( (string) $supervisor->display_name ); ?></option><?php endforeach; ?></select></td><td><code>[sn_dot_marketing_form id="<?php echo esc_attr( $saved_id ); ?>"]</code></td><td><a class="button button-small" href="<?php echo esc_url( $this->marketing_export_url( $saved_id ) ); ?>">CSV (<?php echo esc_html( number_format_i18n( (int) ( $lead_counts[ $saved_id ]['total'] ?? 0 ) ) ); ?>)</a></td><td><button type="button" class="button sn-dot-remove-marketing-form">حذف</button></td></tr>
				<?php endforeach; ?>
				</tbody></table>
				<p><button type="button" class="button" id="sn-dot-add-marketing-form">افزودن فرم جدید</button></p>
				<script>(function(){var table=document.querySelector('#sn-dot-marketing-forms-table tbody'),add=document.querySelector('#sn-dot-add-marketing-form'),products=<?php echo wp_json_encode( array_values( $products ) ); ?>,supervisors=<?php echo wp_json_encode( array_map( static function( $u ){ return [ 'id'=>(int) $u->ID, 'name'=>(string) $u->display_name ]; }, $supervisors ) ); ?>,i=<?php echo (int) $form_row; ?>;if(!table||!add){return;}function esc(v){var d=document.createElement('div');d.textContent=v==null?'':String(v);return d.innerHTML;}function row(){i++;var p='<option value="0">انتخاب محصول</option>';products.forEach(function(x){p+='<option value="'+esc(x.id)+'">'+esc(x.name||('#'+x.id))+' — '+esc(x.price||0)+' تومان</option>';});var s='';supervisors.forEach(function(x){s+='<option value="'+esc(x.id)+'">'+esc(x.name)+'</option>';});var n=String(i),tr=document.createElement('tr');tr.innerHTML='<td><input type="checkbox" name="sn_dot_marketing_forms['+n+'][enabled]" value="1"></td><td><input type="text" name="sn_dot_marketing_forms['+n+'][id]" pattern="[a-zA-Z0-9_-]+" required></td><td><input type="text" name="sn_dot_marketing_forms['+n+'][title]"></td><td><select name="sn_dot_marketing_forms['+n+'][product_id]">'+p+'</select></td><td><select name="sn_dot_marketing_forms['+n+'][supervisor_ids][]" multiple size="3">'+s+'</select></td><td>بعد از ذخیره نمایش داده می‌شود</td><td>پس از ذخیره</td><td><button type="button" class="button sn-dot-remove-marketing-form">حذف</button></td>';table.appendChild(tr);}add.addEventListener('click',row);table.addEventListener('click',function(e){if(e.target.classList.contains('sn-dot-remove-marketing-form')){e.target.closest('tr').remove();}});})();</script>
			</div>
			<div class="sn-dot-admin-note"><strong>روش پرداخت</strong><span>در این ورودی هیچ لینک صدور فاکتور و هیچ گزینه کارت‌به‌کارت به مشتری نمایش داده نمی‌شود؛ پس از ثبت فرم مستقیماً به درگاه فعال شبکه فروش منتقل می‌شود.</span></div>
		</div>
		<?php
	}

	public function assessment_product_ids(): array {
		if ( is_array( $this->assessment_product_ids_cache ) ) { return $this->assessment_product_ids_cache; }
		$ids = array_values( array_filter( array_unique( array_filter( array_map( 'absint', array_keys( (array) $this->config()['product_supervisors'] ) ) ) ), static function ( int $product_id ): bool {
			$explicit = sanitize_key( (string) get_post_meta( $product_id, '_sn_sales_product_type', true ) );
			return ! in_array( $explicit, [ 'subscription', 'subscription_star', 'product', 'product_star' ], true );
		} ) );
		$explicit = get_posts( [
			'post_type' => 'product', 'post_status' => [ 'publish', 'private', 'draft' ], 'fields' => 'ids',
			'posts_per_page' => -1, 'no_found_rows' => true, 'meta_key' => '_sn_sales_product_type', 'meta_value' => 'assessment',
		] );
		$ids = array_merge( $ids, array_map( 'absint', is_array( $explicit ) ? $explicit : [] ) );
		$marketing = $this->marketing_config();
		if ( ! empty( $marketing['enabled'] ) && (int) $marketing['product_id'] > 0 ) { $ids[] = (int) $marketing['product_id']; }
		$forms = get_option( 'sn_dot_marketing_forms', [] );
		foreach ( is_array( $forms ) ? $forms : [] as $form ) {
			if ( is_array( $form ) && ! empty( $form['enabled'] ) && absint( $form['product_id'] ?? 0 ) > 0 ) { $ids[] = absint( $form['product_id'] ); }
		}
		$this->assessment_product_ids_cache = array_values( array_unique( array_filter( array_map( 'absint', $ids ) ) ) );
		return $this->assessment_product_ids_cache;
	}

	public function is_assessment_product( int $product_id ): bool {
		$explicit = sanitize_key( (string) get_post_meta( $product_id, '_sn_sales_product_type', true ) );
		if ( in_array( $explicit, [ 'assessment', 'subscription', 'subscription_star', 'product', 'product_star' ], true ) ) { return $explicit === 'assessment'; }
		return in_array( $product_id, $this->assessment_product_ids(), true );
	}

	public function filter_products_for_user( array $products, int $user_id ): array {
		$config = $this->config();
		$marketing = $this->marketing_config();
		$marketing_product_ids = [];
		if ( ! empty( $marketing['enabled'] ) && absint( $marketing['product_id'] ?? 0 ) > 0 ) { $marketing_product_ids[] = absint( $marketing['product_id'] ); }
		$marketing_forms = get_option( 'sn_dot_marketing_forms', [] );
		foreach ( is_array( $marketing_forms ) ? $marketing_forms : [] as $marketing_form ) {
			if ( is_array( $marketing_form ) && ! empty( $marketing_form['enabled'] ) && absint( $marketing_form['product_id'] ?? 0 ) > 0 ) { $marketing_product_ids[] = absint( $marketing_form['product_id'] ); }
		}
		$marketing_product_ids = array_values( array_unique( $marketing_product_ids ) );
		$assessment_ids = array_values( array_diff( $this->assessment_product_ids(), $marketing_product_ids ) );
		if ( empty( $config['enabled'] ) ) {
			return array_values( array_filter( $products, static fn( $row ) => ! in_array( (int) ( $row['id'] ?? 0 ), $assessment_ids, true ) ) );
		}
		// Marketing forms add a sales channel; they do not reserve the product.
		return array_values( $products );
	}

	/**
	 * Resolve the operational owner of a regular assessment queue.
	 *
	 * The seller's nearest ordinary supervisor owns the operational queue. A
	 * senior supervisor is retained only as a compatibility fallback when an old
	 * HR branch has no ordinary supervisor, so existing paid invoices are never
	 * stranded while the hierarchy is being completed.
	 */
	private function resolve_default_supervisor_for_seller( int $seller_id ): int {
		if ( $this->is_supervisor( $seller_id ) && ! $this->is_senior_supervisor( $seller_id ) ) { return $seller_id; }
		global $wpdb;
		$profiles = $wpdb->prefix . 'sn_hr_profiles';
		$positions = $wpdb->prefix . 'sn_hr_positions';
		$assignments = $wpdb->prefix . 'sn_hr_assignments';
		$fallback_senior_id = 0;
		if ( $this->table_exists( $profiles ) && $this->table_exists( $positions ) && $this->table_exists( $assignments ) ) {
			$profile_id = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$profiles} WHERE user_id=%d ORDER BY is_active DESC,id DESC LIMIT 1", $seller_id ) );
			$seen = [];
			for ( $depth = 0; $profile_id > 0 && $depth < 15; $depth++ ) {
				$parent = $wpdb->get_row( $wpdb->prepare(
					"SELECT p.id,p.user_id,pos.slug FROM {$assignments} a INNER JOIN {$profiles} p ON p.id=a.parent_profile_id LEFT JOIN {$positions} pos ON pos.id=p.position_id WHERE a.child_profile_id=%d AND a.is_current=1 AND a.relationship_type='reports_to' ORDER BY a.id DESC LIMIT 1",
					$profile_id
				) );
				if ( ! $parent || isset( $seen[ (int) $parent->id ] ) ) { break; }
				$seen[ (int) $parent->id ] = true;
				$parent_user_id = (int) $parent->user_id;
				if ( (string) $parent->slug === 'supervisor' || ( $this->is_supervisor( $parent_user_id ) && ! $this->is_senior_supervisor( $parent_user_id ) ) ) { return $parent_user_id; }
				if ( ! $fallback_senior_id && ( (string) $parent->slug === 'senior_supervisor' || $this->is_senior_supervisor( $parent_user_id ) ) ) { $fallback_senior_id = $parent_user_id; }
				$profile_id = (int) $parent->id;
			}
		}
		$legacy_id = absint( get_user_meta( $seller_id, 'sn_supervisor_id', true ) );
		$legacy_seen = [];
		for ( $depth = 0; $legacy_id > 0 && $depth < 10; $depth++ ) {
			if ( isset( $legacy_seen[ $legacy_id ] ) ) { break; }
			$legacy_seen[ $legacy_id ] = true;
			if ( $this->is_supervisor( $legacy_id ) && ! $this->is_senior_supervisor( $legacy_id ) ) { return $legacy_id; }
			if ( ! $fallback_senior_id && $this->is_senior_supervisor( $legacy_id ) ) { $fallback_senior_id = $legacy_id; }
			$legacy_id = absint( get_user_meta( $legacy_id, 'sn_supervisor_id', true ) );
		}
		return $fallback_senior_id;
	}

	private function normalize_conversion_route( string $route ): string {
		$route = sanitize_key( $route );
		return in_array( $route, [ 'supervisor_queue', 'seller_self' ], true ) ? $route : 'supervisor_queue';
	}

	public function prepare_invoice_context( array $product_ids, int $seller_id, string $payment_plan, array $quantities = [], string $conversion_route = 'supervisor_queue', bool $send_assessment_sms = true ) {
		$config = $this->config();
		$assessment = array_values( array_filter( array_map( 'absint', $product_ids ), fn( int $product_id ): bool => $this->is_assessment_product( $product_id ) ) );
		if ( ! $assessment ) { return []; }
		if ( empty( $config['enabled'] ) ) { return new WP_Error( 'sn_dot_disabled', 'فلو اعتبارسنجی هنوز از تنظیمات دات فعال نشده است.' ); }
		if ( count( $product_ids ) !== 1 || count( $assessment ) !== 1 ) { return new WP_Error( 'sn_dot_single_product', 'محصول اعتبارسنجی باید به‌تنهایی در یک پیش‌فاکتور ثبت شود.' ); }
		if ( (int) ( $quantities[0] ?? 1 ) !== 1 ) { return new WP_Error( 'sn_dot_single_quantity', 'تعداد محصول اعتبارسنجی باید دقیقاً یک باشد.' ); }
		if ( $payment_plan !== 'full' ) { return new WP_Error( 'sn_dot_assessment_full_payment', 'اعتبارسنجی فقط با پرداخت کامل قابل صدور است.' ); }
		$product_id = (int) $assessment[0];
		$supervisor_id = $this->resolve_default_supervisor_for_seller( $seller_id );
		if ( ! $supervisor_id ) { return new WP_Error( 'sn_dot_scope', 'سرپرست مستقیم (یا سرپرست ارشد پشتیبان) این فروشنده در ساختار HR تعیین نشده است.' ); }
		$t = $this->tables();
		if ( ! $this->table_exists( $t['links'] ) ) { $this->install_schema(); }
		return [
			'product_id' => $product_id,
			'supervisor_id' => $supervisor_id,
			'conversion_route' => $this->normalize_conversion_route( $conversion_route ),
			'send_assessment_sms' => $send_assessment_sms ? 1 : 0,
			'requires_finance' => ! empty( $config['online_requires_finance'] ) ? 1 : 0,
			'sms_delay_minutes' => (int) $config['sms_delay_minutes'],
		];
	}

	public function prepare_marketing_invoice_context( int $product_id, int $seller_id, int $supervisor_id, ?array $marketing_override = null ) {
		$main = $this->config();
		$marketing = is_array( $marketing_override ) ? $marketing_override : $this->marketing_config();
		$form_id = sanitize_key( (string) ( $marketing['_form_id'] ?? 'default' ) ) ?: 'default';
		if ( empty( $main['enabled'] ) || empty( $marketing['enabled'] ) || ( isset( $marketing['_form_exists'] ) && empty( $marketing['_form_exists'] ) ) ) {
			return new WP_Error( 'sn_dot_marketing_disabled', 'مارکتینگ دات فلو فعال نیست.' );
		}
		if ( $product_id < 1 || $product_id !== (int) $marketing['product_id'] ) {
			return new WP_Error( 'sn_dot_marketing_product', 'محصول درخواست با محصول ثابت مارکتینگ مطابقت ندارد.' );
		}
		if ( ! in_array( $supervisor_id, array_map( 'intval', (array) $marketing['supervisor_ids'] ), true ) ) {
			return new WP_Error( 'sn_dot_marketing_scope', 'سرپرست انتخاب‌شده در محدوده مارکتینگ دات فلو نیست.' );
		}
		if ( ! $this->is_marketing_seller_id( $seller_id ) || (int) get_user_meta( $seller_id, '_sn_dot_marketing_supervisor', true ) !== $supervisor_id ) {
			return new WP_Error( 'sn_dot_marketing_seller', 'فروشنده سیستمی این سرپرست معتبر نیست.' );
		}
		$t = $this->tables();
		if ( ! $this->table_exists( $t['links'] ) ) { $this->install_schema(); }
		return [
			'product_id' => $product_id,
			'marketing_form_id' => $form_id,
			'supervisor_id' => $supervisor_id,
			'send_assessment_sms' => 1,
			'requires_finance' => ! empty( $marketing['online_requires_finance'] ) ? 1 : 0,
			'sms_delay_minutes' => (int) $marketing['sms_delay_minutes'],
		];
	}

	public function register_source_invoice_in_transaction( int $invoice_id, array $context ): bool {
		if ( ! $context ) { return true; }
		global $wpdb;
		$t = $this->tables();
		$data = [
			'invoice_id' => $invoice_id,
			'case_id' => null,
			'flow_kind' => 'assessment_source',
			'payment_kind' => 'source',
			'marketing_form_id' => ! empty( $context['marketing_form_id'] ) ? sanitize_key( (string) $context['marketing_form_id'] ) : null,
			'supervisor_id' => (int) $context['supervisor_id'],
			'conversion_route' => isset( $context['conversion_route'] ) ? $this->normalize_conversion_route( (string) $context['conversion_route'] ) : null,
			// Missing means a legacy caller/invoice and must preserve the established
			// behavior: send the customer subscription-selection SMS after approval.
			'send_assessment_sms' => ! array_key_exists( 'send_assessment_sms', $context ) || ! empty( $context['send_assessment_sms'] ) ? 1 : 0,
			'requires_finance' => ! empty( $context['requires_finance'] ) ? 1 : 0,
			'sms_delay_minutes' => (int) $context['sms_delay_minutes'],
			'created_at' => current_time( 'mysql' ),
			'updated_at' => current_time( 'mysql' ),
		];
		$ok = $wpdb->insert( $t['links'], $this->existing_columns( $t['links'], $data ) );
		return (bool) $ok;
	}

	public function is_dot_invoice_id( int $invoice_id ): bool {
		global $wpdb;
		$t = $this->tables();
		if ( $invoice_id < 1 || ! $this->table_exists( $t['links'] ) ) { return false; }
		return (bool) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$t['links']} WHERE invoice_id=%d LIMIT 1", $invoice_id ) );
	}

	/**
	 * Source assessment invoices retain the plugin's established paid-invoice
	 * effects (Woo order, configured commission, and ordinary payment receipt).
	 * Conversion invoices are synthetic option payments and must never be
	 * posted as a second product sale.
	 */
	public function should_bypass_core_paid_flow( int $invoice_id ): bool {
		global $wpdb;
		$t = $this->tables();
		if ( $invoice_id < 1 || ! $this->table_exists( $t['links'] ) ) { return false; }
		return 'conversion_payment' === (string) $wpdb->get_var( $wpdb->prepare( "SELECT flow_kind FROM {$t['links']} WHERE invoice_id=%d LIMIT 1", $invoice_id ) );
	}

	public function hold_verified_online_payment_for_finance( int $invoice_id, int $stage_no, string $source, string $ref_id, string $authority ): ?array {
		global $wpdb;
		$t = $this->tables();
		if ( $this->is_gifted_assessment_invoice( $invoice_id ) ) {
			return [ 'success' => false, 'message' => 'هزینه اعتبارسنجی این فاکتور قبلاً توسط شرکت هدیه شده و پرداخت آن بسته است.' ];
		}
		$link = $this->table_exists( $t['links'] ) ? $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['links']} WHERE invoice_id=%d AND flow_kind IN ('assessment_source','conversion_payment') LIMIT 1", $invoice_id ) ) : null;
		if ( ! $link ) { return null; }
		if ( empty( $link->requires_finance ) ) { return null; }
		$invoice_table = $wpdb->prefix . 'sn_invoices';
		$stage_table = $wpdb->prefix . 'sn_invoice_payment_stages';
		$payments_table = $wpdb->prefix . 'sn_payments';
		$now = current_time( 'mysql' );
		$wpdb->query( 'START TRANSACTION' );
		$invoice = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$invoice_table} WHERE id=%d FOR UPDATE", $invoice_id ) );
		$stage = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$stage_table} WHERE invoice_id=%d AND stage_no=%d FOR UPDATE", $invoice_id, $stage_no ) );
		if ( ! $invoice || ! $stage ) { $wpdb->query( 'ROLLBACK' ); return [ 'success' => false, 'message' => 'فاکتور یا مرحله پرداخت یافت نشد.' ]; }
		if ( in_array( (string) $stage->status, [ 'paid', 'approved' ], true ) ) { $wpdb->query( 'COMMIT' ); return [ 'success' => true, 'completed' => true ]; }
		$dot_payment = null;
		if ( (string) $link->flow_kind === 'conversion_payment' ) {
			$dot_payment = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['payments']} WHERE invoice_id=%d FOR UPDATE", $invoice_id ) );
			if ( ! $dot_payment || ! in_array( (string) $dot_payment->status, [ 'pending', 'pending_finance', 'rejected' ], true ) ) {
				$wpdb->query( 'ROLLBACK' );
				return [ 'success' => false, 'message' => 'این لینک پرداخت فعال نیست یا با لینک جدیدتری جایگزین شده است.' ];
			}
		}
		if ( (string) $stage->status === 'rejected' && $authority !== '' ) {
			$request_status = (string) $wpdb->get_var( $wpdb->prepare( "SELECT status FROM {$payments_table} WHERE authority=%s ORDER BY id DESC LIMIT 1", $authority ) );
			if ( $request_status !== 'pending' ) {
				$wpdb->query( 'ROLLBACK' );
				return [ 'success' => false, 'message' => 'این بازگشت درگاه مربوط به پرداختی است که قبلاً توسط مالی رد شده است.' ];
			}
		}
		$stage_data = [ 'status' => 'pending_financial_approval', 'pay_method' => 'online', 'payment_source' => sanitize_key( $source ), 'payment_ref_id' => sanitize_text_field( $ref_id ), 'updated_at' => $now ];
		$invoice_data = $this->existing_columns( $invoice_table, [
			'status' => 'pending_financial_approval', 'invoice_status' => 'pending_financial_approval', 'payment_status' => 'pending_financial_approval',
			'pay_method' => 'online', 'payment_source' => sanitize_key( $source ), 'payment_workflow_status' => 'awaiting_financial_approval', 'updated_at' => $now,
		] );
		$ok = false !== $wpdb->update( $stage_table, $stage_data, [ 'id' => (int) $stage->id ] )
			&& false !== $wpdb->update( $invoice_table, $invoice_data, [ 'id' => $invoice_id ] )
			&& false !== $wpdb->update( $t['links'], [ 'payment_verified_at' => $now, 'updated_at' => $now ], [ 'id' => (int) $link->id ] );
		if ( (string) $link->flow_kind === 'assessment_source' && $this->table_exists( $t['marketing_leads'] ) ) {
			$ok = $ok && false !== $wpdb->update( $t['marketing_leads'], [
				'status' => 'pending_finance',
				'last_event' => 'pending_finance',
				'last_event_reason' => null,
				'last_event_at' => $now,
				'gateway_returned_at' => $now,
				'updated_at' => $now,
			], [ 'invoice_id' => $invoice_id ] );
		}
		if ( $dot_payment ) {
			$ok = $ok && false !== $wpdb->update( $t['payments'], [ 'status' => 'pending_finance', 'updated_at' => $now ], [ 'id' => (int) $dot_payment->id ] );
		}
		if ( $authority !== '' ) {
			$ok = $ok && false !== $wpdb->update( $payments_table, $this->existing_columns( $payments_table, [ 'status' => 'verified', 'ref_id' => $ref_id, 'pay_method' => 'online', 'payment_source' => sanitize_key( $source ), 'updated_at' => $now ] ), [ 'authority' => $authority ] );
		}
		if ( ! $ok ) { $wpdb->query( 'ROLLBACK' ); return [ 'success' => false, 'message' => 'ثبت انتظار تأیید مالی انجام نشد.' ]; }
		$wpdb->query( 'COMMIT' );
		if ( (string) $link->flow_kind === 'assessment_source' && $this->table_exists( $t['marketing_leads'] ) ) {
			$marketing_submission_id = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$t['marketing_leads']} WHERE invoice_id=%d LIMIT 1", $invoice_id ) );
			if ( $marketing_submission_id > 0 ) { $this->record_marketing_event( $marketing_submission_id, 'pending_finance', 'success', sanitize_key( $source ), [ 'invoice_id' => $invoice_id ], 'invoice-' . $invoice_id ); }
		}
		if ( $dot_payment && (int) $link->case_id > 0 ) {
			$this->log_case( (int) $link->case_id, 0, 'online_payment_pending_finance', '', 'payment_link_sent', [ 'invoice_id' => $invoice_id, 'payment_type' => (string) $dot_payment->payment_type, 'amount' => (float) $dot_payment->requested_amount, 'gateway' => sanitize_key( $source ) ] );
		}
		return [ 'success' => true, 'pending_finance' => true ];
	}

	public function on_invoice_paid( int $invoice_id, $invoice ): void {
		global $wpdb;
		$t = $this->tables();
		if ( $this->is_gifted_assessment_invoice( $invoice_id ) ) { return; }
		if ( ! $this->table_exists( $t['links'] ) ) { return; }
		$link = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['links']} WHERE invoice_id=%d LIMIT 1", $invoice_id ) );
		if ( ! $link ) { return; }
		if ( (string) $link->flow_kind === 'assessment_source' ) {
			$this->resolve_operational_archive( 'assessment_unpaid', 'invoice', $invoice_id, 'assessment_paid' );
		}
		$inv = is_object( $invoice ) ? $invoice : (object) $invoice;
		$fresh = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$wpdb->prefix}sn_invoices WHERE id=%d LIMIT 1", $invoice_id ) );
		if ( $fresh ) { $inv = $fresh; }
		if ( (string) $link->flow_kind === 'assessment_source' ) {
			// Regular seller invoices always follow the seller's current direct HR
			// supervisor. Marketing links keep their explicit landing-form route.
			$link = $this->reconcile_regular_assessment_link_supervisor( $link, $inv );
		}
		$marketing_submission_id = $this->table_exists( $t['marketing_leads'] )
			? (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$t['marketing_leads']} WHERE invoice_id=%d LIMIT 1", $invoice_id ) )
			: 0;
		if ( ! empty( $inv->lead_id ) && ( $marketing_submission_id > 0 || $this->is_marketing_seller_id( (int) ( $inv->seller_id ?? 0 ) ) ) ) {
			$wpdb->update( $wpdb->prefix . 'sn_leads', [ 'lead_status' => 'payment_paid', 'note' => 'Marketing Dot Flow: پرداخت انجام شد' ], [ 'id' => (int) $inv->lead_id ], [ '%s', '%s' ], [ '%d' ] );
		}
		if ( $marketing_submission_id > 0 ) {
			$paid_now = current_time( 'mysql' );
			$wpdb->update( $t['marketing_leads'], [
				'status' => 'payment_paid',
				'last_event' => 'payment_paid',
				'last_event_reason' => null,
				'last_event_at' => $paid_now,
				'gateway_returned_at' => $paid_now,
				'paid_at' => $paid_now,
				'updated_at' => $paid_now,
			], [ 'invoice_id' => $invoice_id ] );
			$this->record_marketing_event( $marketing_submission_id, 'payment_paid', 'success', '', [ 'invoice_id' => $invoice_id ], 'invoice-' . $invoice_id );
		}
		$wpdb->query( $wpdb->prepare( "UPDATE {$wpdb->prefix}sn_payments SET status='paid',updated_at=%s WHERE invoice_id=%d AND status='verified'", current_time( 'mysql' ), $invoice_id ) );
		if ( (string) $link->flow_kind === 'assessment_source' ) {
			// The link is the immutable invoice-time snapshot. A later settings
			// change must not strand an already-paid assessment outside the queue.
			$case_id = $this->create_case_for_source_invoice( $link, $inv );
			if ( $case_id < 1 ) {
				error_log( 'SN Dot: paid assessment #' . $invoice_id . ' could not be materialized in the supervisor conversion queue.' );
			}
			return;
		}
		if ( (string) $link->flow_kind === 'conversion_payment' ) {
			$this->complete_conversion_payment( $link, $inv );
		}
	}

	private function nominal_for_phone( string $phone, array $config ): array {
		$digits = preg_replace( '/\D+/', '', SN_Helpers::to_english_nums( $phone ) );
		$digit = $digits !== '' ? substr( $digits, -1 ) : '0';
		$key = sanitize_key( (string) ( $config['digit_map'][ $digit ] ?? '' ) );
		foreach ( (array) $config['nominal_credits'] as $credit ) {
			if ( (string) ( $credit['key'] ?? '' ) === $key ) {
				return [ 'key' => $key, 'amount' => (float) ( $credit['amount'] ?? 0 ) ];
			}
		}
		$first = (array) ( $config['nominal_credits'][0] ?? [] );
		return [ 'key' => (string) ( $first['key'] ?? '' ), 'amount' => (float) ( $first['amount'] ?? 0 ) ];
	}

	private function local_mysql_from_timestamp( int $timestamp ): string {
		return SN_Helpers::site_mysql_from_timestamp( $timestamp );
	}

	private function mysql_timestamp( string $mysql ): int {
		return SN_Helpers::site_mysql_timestamp( $mysql, time() );
	}

	/**
	 * Align one regular assessment link/case with the seller's direct supervisor.
	 *
	 * The invoice and payment ledgers remain untouched. An already-assigned
	 * supervisor-queue case is deliberately not moved mid-conversion; missing,
	 * unassigned and seller-self cases are safe to repair.
	 */
	private function reconcile_regular_assessment_link_supervisor( object $link, object $invoice ): object {
		global $wpdb;
		$t = $this->tables();
		if ( (string) ( $link->flow_kind ?? '' ) !== 'assessment_source' || trim( (string) ( $link->marketing_form_id ?? '' ) ) !== '' ) { return $link; }
		$seller_id = absint( $invoice->seller_id ?? 0 );
		// Legacy marketing links may predate marketing_form_id. Their explicit
		// landing/round-robin supervisor must never be replaced by an HR parent.
		if ( $this->is_marketing_seller_id( $seller_id ) ) { return $link; }
		$resolved_supervisor_id = $this->resolve_default_supervisor_for_seller( $seller_id );
		if ( $resolved_supervisor_id < 1 || ! $this->is_supervisor( $resolved_supervisor_id ) || $resolved_supervisor_id === (int) ( $link->supervisor_id ?? 0 ) ) { return $link; }

		$case = null;
		if ( $this->table_exists( $t['cases'] ) ) {
			$case = $wpdb->get_row( $wpdb->prepare(
				"SELECT * FROM {$t['cases']} WHERE id=%d OR source_invoice_id=%d ORDER BY (id=%d) DESC LIMIT 1",
				absint( $link->case_id ?? 0 ),
				absint( $invoice->id ?? 0 ),
				absint( $link->case_id ?? 0 )
			) );
		}
		$is_seller_self = $case && (string) ( $case->conversion_route ?? $link->conversion_route ?? '' ) === 'seller_self';
		if ( $case && ! $is_seller_self && (int) ( $case->converter_id ?? 0 ) > 0 ) { return $link; }

		$old_supervisor_id = (int) ( $link->supervisor_id ?? 0 );
		$now = current_time( 'mysql' );
		$wpdb->query( 'START TRANSACTION' );
		$link_update = [ 'supervisor_id' => $resolved_supervisor_id, 'updated_at' => $now ];
		if ( $case && empty( $link->case_id ) ) { $link_update['case_id'] = (int) $case->id; }
		$link_ok = false !== $wpdb->update( $t['links'], $this->existing_columns( $t['links'], $link_update ), [ 'id' => (int) $link->id ] );
		$case_ok = ! $case || false !== $wpdb->update( $t['cases'], [ 'supervisor_id' => $resolved_supervisor_id, 'updated_at' => $now ], [ 'id' => (int) $case->id ] );
		if ( ! $link_ok || ! $case_ok ) { $wpdb->query( 'ROLLBACK' ); return $link; }
		$wpdb->query( 'COMMIT' );

		$link->supervisor_id = $resolved_supervisor_id;
		if ( $case && empty( $link->case_id ) ) { $link->case_id = (int) $case->id; }
		if ( $case ) {
			$this->log_case( (int) $case->id, 0, 'supervisor_scope_reconciled', (string) $case->status, (string) $case->status, [
				'old_supervisor_id' => $old_supervisor_id,
				'new_supervisor_id' => $resolved_supervisor_id,
				'seller_id' => $seller_id,
			] );
		}
		return $link;
	}

	private function create_case_for_source_invoice( object $link, object $invoice ): int {
		global $wpdb;
		$t = $this->tables();
		// Links created before this field existed intentionally retain the old
		// behavior (SMS enabled). New invoices snapshot the seller's exact choice.
		$send_access_sms = ! isset( $link->send_assessment_sms ) || (int) $link->send_assessment_sms === 1;
		$existing = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$t['cases']} WHERE source_invoice_id=%d LIMIT 1", (int) $invoice->id ) );
		if ( $existing ) {
			$wpdb->update( $t['links'], [ 'case_id' => $existing, 'updated_at' => current_time( 'mysql' ) ], [ 'id' => (int) $link->id ] );
			if ( $send_access_sms ) {
				$this->dispatch_case_sms( $existing );
			} else {
				$existing_case = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['cases']} WHERE id=%d LIMIT 1", $existing ) );
				$direct_update = [ 'access_sms_enabled' => 0, 'sms_due_at' => null, 'updated_at' => current_time( 'mysql' ) ];
				if ( $existing_case && (string) $existing_case->status === 'awaiting_access_sms' ) { $direct_update['status'] = 'ready_for_conversion'; }
				$wpdb->update( $t['cases'], $this->existing_columns( $t['cases'], $direct_update ), [ 'id' => $existing ] );
				wp_clear_scheduled_hook( self::CRON_HOOK, [ $existing ] );
				if ( function_exists( 'as_unschedule_all_actions' ) ) { as_unschedule_all_actions( self::CRON_HOOK, [ $existing ], 'sales-network' ); }
			}
			return $existing;
		}
		$config = $this->config();
		$nominal = $this->nominal_for_phone( (string) $invoice->customer_phone, $config );
		$options = array_values( array_filter( (array) $config['options'], static fn( $row ) => ! empty( $row['active'] ) ) );
		$paid_at = (string) ( $link->payment_verified_at ?: ( $invoice->manual_paid_at ?? '' ) ?: ( $invoice->paid_at ?? '' ) ?: current_time( 'mysql' ) );
		// The delay is captured on the source link when that exact form creates the
		// invoice. Never re-read a later global/form setting here: changing another
		// form or editing settings while finance is reviewing must not alter this lead.
		$delay_minutes = max( 0, (int) ( $link->sms_delay_minutes ?? 0 ) );
		$configured_due_timestamp = $this->mysql_timestamp( $paid_at ) + ( $delay_minutes * MINUTE_IN_SECONDS );
		$due_timestamp = max( time(), $configured_due_timestamp );
		$due_at = $this->local_mysql_from_timestamp( $due_timestamp );
		$now = current_time( 'mysql' );
		$conversion_route = isset( $link->conversion_route ) && (string) $link->conversion_route !== ''
			? $this->normalize_conversion_route( (string) $link->conversion_route )
			: 'supervisor_queue';
		$seller_self = 'seller_self' === $conversion_route && (int) $invoice->seller_id > 0;
		$data = [
			'source_invoice_id' => (int) $invoice->id,
			'source_product_id' => (int) $invoice->product_id,
			'seller_id' => (int) $invoice->seller_id,
			'supervisor_id' => (int) $link->supervisor_id,
			'converter_id' => $seller_self ? (int) $invoice->seller_id : null,
			'conversion_route' => $conversion_route,
			'access_sms_enabled' => $send_access_sms ? 1 : 0,
			'customer_wp_id' => ! empty( $invoice->customer_wp_id ) ? (int) $invoice->customer_wp_id : null,
			'customer_name' => (string) $invoice->customer_name,
			'customer_phone' => (string) $invoice->customer_phone,
			'province' => (string) ( $invoice->province ?? '' ),
			'city' => (string) ( $invoice->city ?? '' ),
			'nominal_credit_amount' => (float) $nominal['amount'],
			'nominal_credit_key' => (string) $nominal['key'],
			'options_snapshot_json' => wp_json_encode( $options, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ),
			'status' => $send_access_sms ? 'awaiting_access_sms' : 'ready_for_conversion',
			'assigned_by' => $seller_self ? (int) $invoice->seller_id : null,
			'assigned_at' => $seller_self ? $now : null,
			'sms_due_at' => $send_access_sms ? $due_at : null,
			'created_at' => $now,
			'updated_at' => $now,
		];
		$wpdb->query( 'START TRANSACTION' );
		// Filter against the live schema so an in-flight request during an
		// incremental dbDelta upgrade cannot make an otherwise valid sale fail.
		$inserted = $wpdb->insert( $t['cases'], $this->existing_columns( $t['cases'], $data ) );
		if ( ! $inserted ) {
			$wpdb->query( 'ROLLBACK' );
			error_log( 'SN Dot: case insert failed for source invoice #' . (int) $invoice->id . ': ' . $wpdb->last_error );
			return 0;
		}
		$case_id = (int) $wpdb->insert_id;
		if ( false === $wpdb->update( $t['links'], [ 'case_id' => $case_id, 'updated_at' => $now ], [ 'id' => (int) $link->id ] ) ) {
			$wpdb->query( 'ROLLBACK' );
			return 0;
		}
		$wpdb->query( 'COMMIT' );
		$initial_status = $send_access_sms ? 'awaiting_access_sms' : 'ready_for_conversion';
		$this->log_case( $case_id, 0, 'case_created', '', $initial_status, [
			'source_invoice_id' => (int) $invoice->id,
			'send_assessment_sms' => $send_access_sms ? 1 : 0,
			'sms_due_at' => $send_access_sms ? $due_at : null,
			'conversion_route' => $conversion_route,
			'converter_id' => $seller_self ? (int) $invoice->seller_id : 0,
		] );
		if ( $send_access_sms ) { $this->dispatch_case_sms( $case_id ); }
		return $case_id;
	}

	/**
	 * Move a legacy paid-invoice referral into the shared conversion workspace.
	 *
	 * The distribution row is retained as an immutable origin record. Linking is
	 * idempotent by referral_item_id and falls back to the source-invoice unique
	 * key when that invoice already owns a Dot case.
	 */
	public function ensure_paid_referral_conversion_case( int $referral_item_id, int $responsible_user_id = 0 ): int {
		global $wpdb;
		$referral_item_id = absint( $referral_item_id );
		$responsible_user_id = absint( $responsible_user_id );
		$t = $this->tables();
		$distribution_table = $wpdb->prefix . 'sn_distribution_items';
		$invoice_table = $wpdb->prefix . 'sn_invoices';
		$invoice_items_table = $wpdb->prefix . 'sn_invoice_items';
		if ( $referral_item_id < 1 || ! $this->table_exists( $distribution_table ) || ! $this->table_exists( $invoice_table ) ) { return 0; }
		if ( ! $this->table_exists( $t['cases'] ) || ! in_array( 'referral_item_id', $this->columns( $t['cases'] ), true ) ) {
			$this->install_schema();
			$this->table_columns = [];
		}
		if ( ! $this->table_exists( $t['cases'] ) || ! in_array( 'referral_item_id', $this->columns( $t['cases'] ), true ) ) { return 0; }

		$item = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$distribution_table} WHERE id=%d LIMIT 1", $referral_item_id ) );
		if ( ! $item || (string) ( $item->source_type ?? '' ) !== 'paid_invoice_referral' || (string) ( $item->workflow_kind ?? '' ) === 'invoice_payment_completion' ) { return 0; }
		if ( in_array( (string) ( $item->status ?? '' ), [ 'referral_cancelled', 'referral_invoice_created' ], true ) ) { return 0; }
		$invoice_id = absint( ( $item->ref_invoice_id ?? 0 ) ?: ( $item->source_id ?? 0 ) );
		$invoice = $invoice_id > 0 ? $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$invoice_table} WHERE id=%d LIMIT 1", $invoice_id ) ) : null;
		if ( ! $invoice ) { return 0; }

		$existing = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['cases']} WHERE referral_item_id=%d LIMIT 1", $referral_item_id ) );
		if ( ! $existing ) { $existing = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['cases']} WHERE source_invoice_id=%d LIMIT 1", $invoice_id ) ); }
		if ( $existing ) {
			$wpdb->query( 'START TRANSACTION' );
			$case_update = empty( $existing->referral_item_id ) ? [ 'referral_item_id' => $referral_item_id, 'updated_at' => current_time( 'mysql' ) ] : [];
			$case_ok = ! $case_update || false !== $wpdb->update( $t['cases'], $case_update, [ 'id' => (int) $existing->id ] );
			$item_ok = false !== $wpdb->update( $distribution_table, [ 'status' => 'converted_to_dot_case', 'updated_at' => current_time( 'mysql' ) ], [ 'id' => $referral_item_id ] );
			if ( ! $case_ok || ! $item_ok ) { $wpdb->query( 'ROLLBACK' ); return 0; }
			$wpdb->query( 'COMMIT' );
			return (int) $existing->id;
		}

		$seller_id = absint( ( $item->referral_original_seller_id ?? 0 ) ?: ( $item->final_seller_user_id ?? 0 ) ?: ( $invoice->seller_id ?? 0 ) );
		// Legacy rows may still be owned by another hierarchy level. New shared
		// conversion cases use the seller's direct supervisor whenever it is known.
		$supervisor_id = $this->resolve_default_supervisor_for_seller( $seller_id );
		if ( ! $this->is_supervisor( $supervisor_id ) ) { $supervisor_id = $responsible_user_id ?: absint( $item->current_owner_user_id ?? 0 ); }
		if ( $seller_id < 1 || $supervisor_id < 1 || ! $this->is_supervisor( $supervisor_id ) ) { return 0; }

		$product_id = absint( ( $item->ref_product_id ?? 0 ) ?: ( $invoice->product_id ?? 0 ) );
		$product = $product_id > 0 && function_exists( 'wc_get_product' ) ? wc_get_product( $product_id ) : null;
		$product_name = $product && method_exists( $product, 'get_name' ) ? trim( (string) $product->get_name() ) : '';
		$product_price = $product && method_exists( $product, 'get_price' ) ? max( 0, (float) $product->get_price() ) : 0.0;
		if ( $this->table_exists( $invoice_items_table ) ) {
			$snapshot = $wpdb->get_row( $wpdb->prepare( "SELECT product_name,unit_price,total_price,qty FROM {$invoice_items_table} WHERE invoice_id=%d AND product_id=%d ORDER BY id ASC LIMIT 1", $invoice_id, $product_id ) );
			if ( $snapshot ) {
				if ( $product_name === '' ) { $product_name = trim( (string) $snapshot->product_name ); }
				if ( $product_price <= 0 ) { $product_price = max( 0, (float) ( $snapshot->unit_price ?: $snapshot->total_price ) ); }
			}
		}
		if ( $product_name === '' && $product_id > 0 ) { $product_name = trim( (string) get_the_title( $product_id ) ); }
		if ( $product_name === '' ) { $product_name = $product_id > 0 ? 'محصول ارجاعی #' . $product_id : 'محصول ارجاعی'; }
		$option_key = sanitize_key( 'paid_referral_' . $referral_item_id );
		$note = sanitize_textarea_field( (string) ( $item->referral_note ?? '' ) );
		$option = [
			'key' => $option_key,
			'title' => $product_name,
			'product_id' => $product_id,
			'price' => $product_price,
			'content' => $note,
			'active' => 1,
		];
		$now = current_time( 'mysql' );
		$data = [
			'referral_item_id' => $referral_item_id,
			'source_invoice_id' => $invoice_id,
			'source_product_id' => $product_id,
			'seller_id' => $seller_id,
			'supervisor_id' => $supervisor_id,
			'converter_id' => null,
			'conversion_route' => 'supervisor_queue',
			'customer_wp_id' => ! empty( $invoice->customer_wp_id ) ? (int) $invoice->customer_wp_id : null,
			'customer_name' => sanitize_text_field( (string) ( ( $item->seller_customer_name ?? '' ) ?: ( $invoice->customer_name ?? '' ) ) ),
			'customer_phone' => SN_Helpers::normalize_mobile( (string) ( ( $item->normalized_phone ?? '' ) ?: ( $item->customer_phone ?? '' ) ?: ( $invoice->customer_phone ?? '' ) ) ),
			'province' => sanitize_text_field( (string) ( ( $item->seller_province ?? '' ) ?: ( $invoice->province ?? '' ) ) ),
			'city' => sanitize_text_field( (string) ( ( $item->seller_city ?? '' ) ?: ( $invoice->city ?? '' ) ) ),
			'nominal_credit_amount' => $product_price,
			'nominal_credit_key' => 'paid_referral',
			'options_snapshot_json' => wp_json_encode( [ $option ], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ),
			'status' => 'ready_for_conversion',
			'selected_option_key' => $option_key,
			'selected_option_product_id' => $product_id ?: null,
			'selected_option_title' => $product_name,
			'selected_option_price' => $product_price,
			'selected_option_html' => $this->sanitize_option_content( $note ),
			'selected_at' => (string) ( ( $item->created_at ?? '' ) ?: $now ),
			'converter_contact_status' => 'new',
			'converter_note' => $note,
			'remaining_amount' => $product_price,
			'created_at' => (string) ( ( $item->created_at ?? '' ) ?: $now ),
			'updated_at' => $now,
		];
		$wpdb->query( 'START TRANSACTION' );
		$inserted = $wpdb->insert( $t['cases'], $this->existing_columns( $t['cases'], $data ) );
		if ( ! $inserted ) { $wpdb->query( 'ROLLBACK' ); return 0; }
		$case_id = (int) $wpdb->insert_id;
		if ( false === $wpdb->update( $distribution_table, [ 'status' => 'converted_to_dot_case', 'updated_at' => $now ], [ 'id' => $referral_item_id ] ) ) {
			$wpdb->query( 'ROLLBACK' );
			return 0;
		}
		$wpdb->query( 'COMMIT' );
		$this->log_case( $case_id, $responsible_user_id, 'paid_referral_migrated_to_conversion', '', 'ready_for_conversion', [
			'referral_item_id' => $referral_item_id,
			'source_invoice_id' => $invoice_id,
			'product_id' => $product_id,
		] );
		return $case_id;
	}

	/** Convert bounded legacy supervisor-owned rows before rendering the queue. */
	public function migrate_paid_referrals_for_actor( int $actor_id, int $limit = 100 ): int {
		global $wpdb;
		$actor_id = absint( $actor_id );
		$limit = max( 1, min( 200, $limit ) );
		$table = $wpdb->prefix . 'sn_distribution_items';
		if ( $actor_id < 1 || ! $this->is_supervisor( $actor_id ) || ! $this->table_exists( $table ) ) { return 0; }
		$owner_ids = [ $actor_id ];
		if ( $this->is_senior_supervisor( $actor_id ) && class_exists( 'SN_Scope_Service' ) ) {
			foreach ( ( new SN_Scope_Service() )->visible_user_ids( $actor_id ) as $visible_user_id ) {
				$visible_user_id = absint( $visible_user_id );
				if ( $visible_user_id > 0 && $this->is_supervisor( $visible_user_id ) ) { $owner_ids[] = $visible_user_id; }
			}
		}
		$owner_ids = array_values( array_unique( array_filter( $owner_ids ) ) );
		$statuses = [ 'needs_repeat_action', 'pending_supervisor_action', 'referral_self_invoice_pending', 'assigned_to_role', 'paid_referral_routed', 'delivered_to_seller' ];
		$owner_placeholders = implode( ',', array_fill( 0, count( $owner_ids ), '%d' ) );
		$status_placeholders = implode( ',', array_fill( 0, count( $statuses ), '%s' ) );
		$item_ids = array_map( 'absint', (array) $wpdb->get_col( $wpdb->prepare(
			"SELECT id FROM {$table} WHERE source_type='paid_invoice_referral' AND COALESCE(workflow_kind,'')<>'invoice_payment_completion' AND current_owner_user_id IN ({$owner_placeholders}) AND status IN ({$status_placeholders}) ORDER BY id ASC LIMIT %d",
			...array_merge( $owner_ids, $statuses, [ $limit ] )
		) ) );
		$migrated = 0;
		foreach ( $item_ids as $item_id ) { if ( $this->ensure_paid_referral_conversion_case( $item_id, $actor_id ) > 0 ) { $migrated++; } }
		return $migrated;
	}

	/** Exact Dot source-invoice scope used by the supervisor invoice tab. */
	public function source_invoice_ids_for_supervisor( int $supervisor_id, int $limit = 250 ): array {
		global $wpdb;
		$t = $this->tables();
		$supervisor_id = absint( $supervisor_id );
		$limit = max( 1, min( 500, $limit ) );
		if ( $supervisor_id < 1 || ! $this->table_exists( $t['links'] ) ) { return []; }
		return array_values( array_unique( array_filter( array_map( 'absint', (array) $wpdb->get_col( $wpdb->prepare(
			"SELECT invoice_id FROM {$t['links']} WHERE supervisor_id=%d AND flow_kind='assessment_source' ORDER BY id DESC LIMIT %d",
			$supervisor_id,
			$limit
		) ) ) ) ) );
	}

	/** Aggregate-only gift metadata; no customer data is returned. */
	public function invoice_gift_states( array $invoice_ids, int $viewer_id ): array {
		global $wpdb;
		$t = $this->tables();
		$invoice_ids = array_values( array_unique( array_filter( array_map( 'absint', $invoice_ids ) ) ) );
		if ( ! $invoice_ids || ! $this->table_exists( $t['links'] ) || ! $this->table_exists( $t['cases'] ) ) { return []; }
		$has_gift_columns = in_array( 'validation_fee_gifted', $this->columns( $t['cases'] ), true );
		$gift_select = $has_gift_columns
			? 'COALESCE(c.validation_fee_gifted,0) validation_fee_gifted,c.validation_gifted_at,c.validation_gift_sms_sent_at,c.validation_gift_sms_error,c.sms_sent_at,c.sms_previewed_at'
			: "0 validation_fee_gifted,NULL validation_gifted_at,NULL validation_gift_sms_sent_at,NULL validation_gift_sms_error,c.sms_sent_at,c.sms_previewed_at";
		$placeholders = implode( ',', array_fill( 0, count( $invoice_ids ), '%d' ) );
		$sql = "SELECT dl.invoice_id,dl.case_id,dl.supervisor_id,{$gift_select}
			FROM {$t['links']} dl
			LEFT JOIN {$t['cases']} c ON c.id=dl.case_id
			WHERE dl.flow_kind='assessment_source' AND dl.invoice_id IN ({$placeholders})";
		$rows = $wpdb->get_results( $wpdb->prepare( $sql, ...$invoice_ids ) ) ?: [];
		$is_admin = $viewer_id > 0 && user_can( $viewer_id, 'manage_options' );
		$out = [];
		foreach ( $rows as $row ) {
			$invoice_id = (int) $row->invoice_id;
			$out[ $invoice_id ] = [
				'is_dot_assessment' => true,
				'case_id' => (int) $row->case_id,
				'validation_gifted' => ! empty( $row->validation_fee_gifted ),
				'validation_gifted_at' => (string) ( $row->validation_gifted_at ?? '' ),
				'validation_gift_sms_sent_at' => (string) ( $row->validation_gift_sms_sent_at ?? '' ),
				'validation_gift_sms_error' => (string) ( $row->validation_gift_sms_error ?? '' ),
				'sms_delivered' => ! empty( $row->sms_sent_at ) || ! empty( $row->sms_previewed_at ),
				'can_manage_gift' => $is_admin || ( $viewer_id > 0 && (int) $row->supervisor_id === $viewer_id ),
			];
		}
		return $out;
	}

	public function is_gifted_assessment_invoice( int $invoice_id ): bool {
		global $wpdb;
		$t = $this->tables();
		if ( $invoice_id < 1 || ! $this->table_exists( $t['links'] ) || ! $this->table_exists( $t['cases'] ) || ! in_array( 'validation_fee_gifted', $this->columns( $t['cases'] ), true ) ) { return false; }
		return (bool) $wpdb->get_var( $wpdb->prepare(
			"SELECT c.validation_fee_gifted FROM {$t['links']} dl INNER JOIN {$t['cases']} c ON c.id=dl.case_id WHERE dl.invoice_id=%d AND dl.flow_kind='assessment_source' LIMIT 1",
			$invoice_id
		) );
	}

	public function handle_gift_validation_access(): void {
		if ( ! is_user_logged_in() || ! check_ajax_referer( 'sn_public', 'nonce', false ) ) {
			wp_send_json_error( [ 'message' => 'نشست کاربری معتبر نیست؛ صفحه را تازه‌سازی کنید.' ], 403 );
		}
		$invoice_id = absint( $_POST['invoice_id'] ?? 0 );
		if ( $invoice_id < 1 ) { wp_send_json_error( [ 'message' => 'شناسه پیش‌فاکتور معتبر نیست.' ], 400 ); }
		$result = $this->gift_validation_for_invoice( $invoice_id, get_current_user_id() );
		if ( empty( $result['success'] ) ) {
			wp_send_json_error( [ 'message' => (string) ( $result['message'] ?? 'فعال‌سازی اعتبارسنجی هدیه انجام نشد.' ) ], (int) ( $result['status'] ?? 400 ) );
		}
		wp_send_json_success( $result );
	}

	/**
	 * Skip only the assessment fee. No paid amount, sale or finance approval is
	 * fabricated. The source invoice remains a pre-invoice and receives an audit
	 * workflow marker while the Dot case is opened for customer selection.
	 */
	private function gift_validation_for_invoice( int $invoice_id, int $actor_id ): array {
		global $wpdb;
		$t = $this->tables();
		if ( empty( $this->config()['enabled'] ) ) { return [ 'success' => false, 'message' => 'دات فلو در تنظیمات فعال نیست.', 'status' => 409 ]; }
		if ( ! $this->table_exists( $t['links'] ) || ! $this->table_exists( $t['cases'] ) || ! in_array( 'validation_fee_gifted', $this->columns( $t['cases'] ), true ) ) { $this->install_schema(); }
		if ( ! in_array( 'validation_fee_gifted', $this->columns( $t['cases'] ), true ) ) { return [ 'success' => false, 'message' => 'ستون‌های اعتبارسنجی هدیه در دیتابیس ساخته نشدند.', 'status' => 500 ]; }
		$invoice_table = $wpdb->prefix . 'sn_invoices';
		$stage_table = $wpdb->prefix . 'sn_invoice_payment_stages';
		$payment_request_table = $wpdb->prefix . 'sn_payments';
		$now = current_time( 'mysql' );

		$wpdb->query( 'START TRANSACTION' );
		$link = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['links']} WHERE invoice_id=%d AND flow_kind='assessment_source' LIMIT 1 FOR UPDATE", $invoice_id ) );
		if ( ! $link ) { $wpdb->query( 'ROLLBACK' ); return [ 'success' => false, 'message' => 'این پیش‌فاکتور مربوط به محصول اعتبارسنجی دات نیست.', 'status' => 404 ]; }
		if ( ! user_can( $actor_id, 'manage_options' ) && (int) $link->supervisor_id !== $actor_id ) {
			$wpdb->query( 'ROLLBACK' );
			return [ 'success' => false, 'message' => 'فقط سرپرست همین پیش‌فاکتور اجازه ارسال لینک انتخاب را دارد.', 'status' => 403 ];
		}
		$invoice = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$invoice_table} WHERE id=%d LIMIT 1 FOR UPDATE", $invoice_id ) );
		if ( ! $invoice ) { $wpdb->query( 'ROLLBACK' ); return [ 'success' => false, 'message' => 'پیش‌فاکتور پیدا نشد.', 'status' => 404 ]; }

		$existing_case = (int) $link->case_id > 0 ? $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['cases']} WHERE id=%d LIMIT 1 FOR UPDATE", (int) $link->case_id ) ) : null;
		if ( ! $existing_case ) {
			$existing_case = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['cases']} WHERE source_invoice_id=%d LIMIT 1 FOR UPDATE", $invoice_id ) );
		}
		if ( $existing_case ) {
			if ( empty( $existing_case->validation_fee_gifted ) ) {
				$wpdb->query( 'ROLLBACK' );
				return [ 'success' => false, 'message' => 'برای این پیش‌فاکتور قبلاً پرونده اعتبارسنجی از مسیر پرداخت ساخته شده است.', 'status' => 409 ];
			}
			if ( (int) $link->case_id !== (int) $existing_case->id ) {
				$wpdb->update( $t['links'], [ 'case_id' => (int) $existing_case->id, 'updated_at' => $now ], [ 'id' => (int) $link->id ] );
			}
			$wpdb->query( 'COMMIT' );
			if ( empty( $existing_case->sms_sent_at ) && empty( $existing_case->sms_previewed_at ) ) {
				$wpdb->update( $t['cases'], [ 'sms_due_at' => $now, 'sms_attempts' => 0, 'sms_last_error' => null, 'validation_gift_sms_error' => null, 'updated_at' => $now ], [ 'id' => (int) $existing_case->id ] );
				$this->dispatch_case_sms( (int) $existing_case->id );
			}
			$fresh_existing = $this->get_case( (int) $existing_case->id );
			$delivered = $fresh_existing && ( ! empty( $fresh_existing->sms_sent_at ) || ! empty( $fresh_existing->sms_previewed_at ) );
			return [
				'success' => true,
				'already_applied' => true,
				'case_id' => (int) $existing_case->id,
				'sms_sent' => $fresh_existing && ! empty( $fresh_existing->sms_sent_at ),
				'sms_previewed' => $fresh_existing && ! empty( $fresh_existing->sms_previewed_at ),
				'message' => $delivered ? 'اعتبارسنجی قبلاً هدیه شده و لینک انتخاب برای مشتری ارسال شده است.' : 'اعتبارسنجی قبلاً هدیه شده است؛ ارسال پیامک هنوز موفق نشده و دوباره در صف قرار گرفت.',
			];
		}

		$invoice_statuses = array_values( array_filter( array_map( 'sanitize_key', [
			(string) ( $invoice->status ?? '' ),
			(string) ( $invoice->invoice_status ?? '' ),
			(string) ( $invoice->payment_status ?? '' ),
		] ) ) );
		$blocked_statuses = [ 'paid', 'approved', 'completed', 'partial_paid', 'receipt_uploaded', 'pending_financial_approval', 'financial_approved', 'gateway_paid', 'online_paid' ];
		if ( array_intersect( $invoice_statuses, $blocked_statuses ) || (float) ( $invoice->paid_total_amount ?? 0 ) > 0.5 || ! empty( $invoice->paid_at ) || ! empty( $invoice->approved_at ) ) {
			$wpdb->query( 'ROLLBACK' );
			return [ 'success' => false, 'message' => 'این پیش‌فاکتور وارد پرداخت یا تأیید مالی شده و دیگر قابل تبدیل به اعتبارسنجی هدیه نیست.', 'status' => 409 ];
		}
		$verified_request = $this->table_exists( $payment_request_table ) ? (int) $wpdb->get_var( $wpdb->prepare( "SELECT COUNT(*) FROM {$payment_request_table} WHERE invoice_id=%d AND status IN ('verified','paid')", $invoice_id ) ) : 0;
		if ( $verified_request > 0 ) {
			$wpdb->query( 'ROLLBACK' );
			return [ 'success' => false, 'message' => 'پرداخت این پیش‌فاکتور در درگاه تأیید شده و باید ابتدا توسط مالی تعیین تکلیف شود.', 'status' => 409 ];
		}

		$config = $this->config();
		$nominal = $this->nominal_for_phone( (string) $invoice->customer_phone, $config );
		$options = array_values( array_filter( (array) $config['options'], static fn( $row ) => ! empty( $row['active'] ) ) );
		if ( ! $options ) { $wpdb->query( 'ROLLBACK' ); return [ 'success' => false, 'message' => 'هیچ گزینه فعالی در دات ستینگ تعریف نشده است.', 'status' => 409 ]; }
		$case_data = $this->existing_columns( $t['cases'], [
			'source_invoice_id' => $invoice_id,
			'source_product_id' => (int) $invoice->product_id,
			'seller_id' => (int) $invoice->seller_id,
			'supervisor_id' => (int) $link->supervisor_id,
			'converter_id' => null,
			'customer_wp_id' => ! empty( $invoice->customer_wp_id ) ? (int) $invoice->customer_wp_id : null,
			'customer_name' => (string) $invoice->customer_name,
			'customer_phone' => (string) $invoice->customer_phone,
			'province' => (string) ( $invoice->province ?? '' ),
			'city' => (string) ( $invoice->city ?? '' ),
			'nominal_credit_amount' => (float) $nominal['amount'],
			'nominal_credit_key' => (string) $nominal['key'],
			'options_snapshot_json' => wp_json_encode( $options, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ),
			'status' => 'awaiting_access_sms',
			'sms_due_at' => $now,
			'validation_fee_gifted' => 1,
			'validation_gifted_by' => $actor_id,
			'validation_gifted_at' => $now,
			'created_at' => $now,
			'updated_at' => $now,
		] );
		if ( ! $wpdb->insert( $t['cases'], $case_data ) ) {
			$wpdb->query( 'ROLLBACK' );
			return [ 'success' => false, 'message' => 'ساخت پرونده انتخاب انجام نشد: ' . ( $wpdb->last_error ?: 'خطای دیتابیس' ), 'status' => 500 ];
		}
		$case_id = (int) $wpdb->insert_id;
		$link_updated = $wpdb->update( $t['links'], [ 'case_id' => $case_id, 'updated_at' => $now ], [ 'id' => (int) $link->id ] );
		$invoice_update = $this->existing_columns( $invoice_table, [
			'status' => 'pre_invoice',
			'invoice_status' => 'pre_invoice',
			'payment_status' => 'pre_invoice',
			'payment_workflow_status' => 'validation_gifted',
			'current_due_amount' => 0,
			'updated_at' => $now,
		] );
		$invoice_updated = $wpdb->update( $invoice_table, $invoice_update, [ 'id' => $invoice_id ] );
		$stage_updated = true;
		if ( $this->table_exists( $stage_table ) ) {
			$stage_updated = false !== $wpdb->query( $wpdb->prepare(
				"UPDATE {$stage_table} SET status='gifted',note=%s,updated_at=%s WHERE invoice_id=%d AND status NOT IN ('paid','approved','pending_financial_approval')",
				'هزینه اعتبارسنجی توسط شرکت هدیه شد؛ هیچ پرداختی از مشتری دریافت نشد.',
				$now,
				$invoice_id
			) );
		}
		if ( false === $link_updated || false === $invoice_updated || ! $stage_updated ) {
			$wpdb->query( 'ROLLBACK' );
			return [ 'success' => false, 'message' => 'ثبت وضعیت اعتبارسنجی هدیه کامل نشد.', 'status' => 500 ];
		}
		if ( $this->table_exists( $payment_request_table ) ) {
			$payment_requests_updated = $wpdb->query( $wpdb->prepare( "UPDATE {$payment_request_table} SET status='cancelled_gifted',updated_at=%s WHERE invoice_id=%d AND status='pending'", $now, $invoice_id ) );
			if ( false === $payment_requests_updated ) { $wpdb->query( 'ROLLBACK' ); return [ 'success' => false, 'message' => 'بستن درخواست پرداخت قبلی انجام نشد.', 'status' => 500 ]; }
		}
		$marketing_crm_lead_id = 0;
		$marketing_submission_id = 0;
		if ( $this->table_exists( $t['marketing_leads'] ) ) {
			$marketing_row = $wpdb->get_row( $wpdb->prepare( "SELECT id,crm_lead_id FROM {$t['marketing_leads']} WHERE invoice_id=%d LIMIT 1 FOR UPDATE", $invoice_id ) );
			if ( $marketing_row ) {
				$marketing_submission_id = (int) $marketing_row->id;
				$marketing_crm_lead_id = (int) $marketing_row->crm_lead_id;
				$marketing_updated = $wpdb->update( $t['marketing_leads'], [
					'status' => 'validation_gifted',
					'last_event' => 'validation_gifted',
					'last_event_reason' => 'supervisor_gift',
					'last_event_at' => $now,
					'updated_at' => $now,
				], [ 'id' => (int) $marketing_row->id ] );
				if ( false === $marketing_updated ) { $wpdb->query( 'ROLLBACK' ); return [ 'success' => false, 'message' => 'ثبت وضعیت هدیه روی لید مارکتینگ انجام نشد.', 'status' => 500 ]; }
			}
		}
		$wpdb->query( 'COMMIT' );

		if ( $marketing_submission_id > 0 ) { $this->record_marketing_event( $marketing_submission_id, 'validation_gifted', 'success', 'supervisor_gift', [ 'invoice_id' => $invoice_id ], 'invoice-' . $invoice_id ); }
		if ( $marketing_crm_lead_id > 0 ) { $this->update_marketing_crm_event_note( $marketing_crm_lead_id, 'validation_gifted' ); }
		$this->log_case( $case_id, $actor_id, 'validation_fee_gifted', '', 'awaiting_access_sms', [
			'source_invoice_id' => $invoice_id,
			'payment_received' => false,
		] );
		$this->dispatch_case_sms( $case_id );
		$fresh_case = $this->get_case( $case_id );
		$sms_sent = $fresh_case && ! empty( $fresh_case->sms_sent_at );
		$sms_previewed = $fresh_case && ! empty( $fresh_case->sms_previewed_at );
		$message = $sms_sent
			? 'هزینه اعتبارسنجی به‌صورت هدیه رد شد و لینک انتخاب برای مشتری پیامک شد.'
			: ( $sms_previewed ? 'اعتبارسنجی هدیه فعال شد و پیامک در حالت آزمایشی نمایش داده شد.' : 'اعتبارسنجی هدیه فعال شد، اما ارسال پیامک موفق نبود و برای تلاش مجدد در صف قرار گرفت.' );
		return [
			'success' => true,
			'already_applied' => false,
			'case_id' => $case_id,
			'sms_sent' => $sms_sent,
			'sms_previewed' => $sms_previewed,
			'message' => $message,
		];
	}

	/**
	 * Send a due access SMS in the finance-approval request itself. WP-Cron is
	 * retained for future-due messages and as a recovery path only.
	 */
	private function dispatch_case_sms( int $case_id ): void {
		global $wpdb;
		$t = $this->tables();
		$case = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['cases']} WHERE id=%d LIMIT 1", $case_id ) );
		if ( ! $case || ( isset( $case->access_sms_enabled ) && ! (int) $case->access_sms_enabled ) || ! empty( $case->sms_sent_at ) || (int) $case->sms_attempts >= 3 ) { return; }
		if ( $this->is_sms_test_mode() && ! empty( $case->sms_previewed_at ) ) { return; }

		if ( empty( $case->sms_due_at ) ) {
			$case->sms_due_at = current_time( 'mysql' );
			$wpdb->update( $t['cases'], [ 'sms_due_at' => (string) $case->sms_due_at, 'updated_at' => current_time( 'mysql' ) ], [ 'id' => $case_id ] );
		}
		$due_timestamp = $this->mysql_timestamp( (string) $case->sms_due_at );
		if ( $due_timestamp > time() ) {
			$this->schedule_case_sms( $case_id, $due_timestamp );
			return;
		}

		$this->process_due_access_sms( $case_id );

		// If another worker briefly held the lock, guarantee a near-term retry.
		$state = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['cases']} WHERE id=%d LIMIT 1", $case_id ) );
		$preview_complete = $this->is_sms_test_mode() && $state && ! empty( $state->sms_previewed_at );
		if ( $state && ! $preview_complete && empty( $state->sms_sent_at ) && (int) $state->sms_attempts < 3 && ! wp_next_scheduled( self::CRON_HOOK, [ $case_id ] ) ) {
			$this->schedule_case_sms( $case_id, time() + 30 );
		}
	}

	private function schedule_case_sms( int $case_id, int $timestamp = 0 ): void {
		global $wpdb;
		$t = $this->tables();
		if ( ! $timestamp ) {
			$due = (string) $wpdb->get_var( $wpdb->prepare( "SELECT sms_due_at FROM {$t['cases']} WHERE id=%d", $case_id ) );
			$timestamp = $due !== '' ? $this->mysql_timestamp( $due ) : time() + 5;
		}
		$timestamp = max( time() + 5, $timestamp );
		$next = wp_next_scheduled( self::CRON_HOOK, [ $case_id ] );
		if ( $next && (int) $next > $timestamp + MINUTE_IN_SECONDS ) {
			wp_unschedule_event( (int) $next, self::CRON_HOOK, [ $case_id ] );
			$next = false;
		}
		if ( ! $next ) {
			wp_schedule_single_event( $timestamp, self::CRON_HOOK, [ $case_id ] );
		}
		// WooCommerce ships Action Scheduler on most installations. Registering
		// the same idempotent worker there removes sole reliance on WP-Cron.
		if ( function_exists( 'as_next_scheduled_action' ) && function_exists( 'as_schedule_single_action' ) ) {
			$as_next = as_next_scheduled_action( self::CRON_HOOK, [ $case_id ], 'sales-network' );
			if ( ! $as_next ) {
				as_schedule_single_action( $timestamp, self::CRON_HOOK, [ $case_id ], 'sales-network', true );
			}
		}
	}

	public function maybe_process_due_sms(): void {
		if ( get_transient( 'sn_dot_due_sms_probe' ) ) { return; }
		set_transient( 'sn_dot_due_sms_probe', '1', 30 );
		$this->reconcile_paid_links();
		if ( empty( $this->config()['enabled'] ) ) { return; }
		global $wpdb;
		$t = $this->tables();
		if ( ! $this->table_exists( $t['cases'] ) ) { return; }
		$preview_filter = $this->is_sms_test_mode() && in_array( 'sms_previewed_at', $this->columns( $t['cases'] ), true ) ? ' AND sms_previewed_at IS NULL' : '';
		$due = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$t['cases']} WHERE sms_sent_at IS NULL AND sms_attempts<3 AND sms_due_at IS NOT NULL AND sms_due_at<=%s{$preview_filter} ORDER BY sms_due_at ASC LIMIT 1", current_time( 'mysql' ) ) );
		if ( $due ) { $this->process_due_access_sms(); }
	}

	/** Parse a Jalali date and Tehran time, then store it in the legacy WP timezone. */
	private function normalize_followup_datetime( string $raw ) {
		$value = trim( SN_Helpers::to_english_nums( wp_unslash( $raw ) ) );
		$value = str_replace( 'T', ' ', substr( $value, 0, 24 ) );
		if ( ! preg_match( '/^(\d{4}[\/\-.]\d{1,2}[\/\-.]\d{1,2})\s+(\d{1,2}:\d{2})$/', $value, $parts ) ) {
			return new WP_Error( 'sn_dot_followup_format', 'تاریخ و ساعت تماس مجدد معتبر نیست.' );
		}
		return SN_Helpers::normalize_jalali_tehran_datetime( (string) $parts[1], (string) $parts[2], true );
	}

	private function clear_followup_reminder_schedule( int $case_id ): void {
		if ( $case_id < 1 ) { return; }
		wp_clear_scheduled_hook( self::FOLLOWUP_CRON_HOOK, [ $case_id ] );
		if ( function_exists( 'as_unschedule_all_actions' ) ) {
			as_unschedule_all_actions( self::FOLLOWUP_CRON_HOOK, [ $case_id ], 'sales-network' );
		}
	}

	private function schedule_followup_reminder( int $case_id, int $timestamp ): void {
		if ( $case_id < 1 ) { return; }
		$this->clear_followup_reminder_schedule( $case_id );
		$timestamp = max( time() + 5, $timestamp );
		wp_schedule_single_event( $timestamp, self::FOLLOWUP_CRON_HOOK, [ $case_id ] );
		if ( function_exists( 'as_schedule_single_action' ) ) {
			as_schedule_single_action( $timestamp, self::FOLLOWUP_CRON_HOOK, [ $case_id ], 'sales-network', true );
		}
	}

	/** Lightweight fallback for sites where a scheduled request has been delayed. */
	public function maybe_process_due_followup_reminders(): void {
		if ( get_transient( 'sn_dot_followup_reminder_probe' ) ) { return; }
		set_transient( 'sn_dot_followup_reminder_probe', '1', 30 );
		global $wpdb;
		$t = $this->tables();
		if ( ! $this->table_exists( $t['cases'] ) || ! in_array( 'converter_followup_at', $this->columns( $t['cases'] ), true ) ) { return; }
		$due = (int) $wpdb->get_var( $wpdb->prepare(
			"SELECT id FROM {$t['cases']} WHERE converter_contact_status='follow_up' AND converter_followup_at IS NOT NULL AND converter_followup_at<=%s AND converter_followup_reminded_at IS NULL AND converter_followup_reminder_attempts<3 AND operational_archived_at IS NULL AND status NOT IN ('completed','customer_declined','archived_unpaid_subscription') ORDER BY converter_followup_at ASC LIMIT 1",
			current_time( 'mysql' )
		) );
		if ( $due > 0 ) { $this->process_due_followup_reminders(); }
	}

	/** Send one idempotent reminder to the role currently responsible for conversion. */
	public function process_due_followup_reminders( int $case_id = 0 ): void {
		global $wpdb;
		$t = $this->tables();
		if ( ! $this->table_exists( $t['cases'] ) || ! in_array( 'converter_followup_at', $this->columns( $t['cases'] ), true ) ) { return; }
		$lock_key = 'sn_dot_followup_reminder_worker_lock';
		$lock = (int) get_option( $lock_key, 0 );
		if ( $lock && $lock > time() - 120 ) { return; }
		if ( $lock ) { delete_option( $lock_key ); }
		if ( ! add_option( $lock_key, time(), '', false ) ) { return; }
		try {
			$where_case = $case_id > 0 ? $wpdb->prepare( ' AND id=%d', $case_id ) : '';
			$rows = $wpdb->get_results( $wpdb->prepare(
				"SELECT * FROM {$t['cases']} WHERE converter_contact_status='follow_up' AND converter_followup_at IS NOT NULL AND converter_followup_at<=%s AND converter_followup_reminded_at IS NULL AND converter_followup_reminder_attempts<3 AND operational_archived_at IS NULL AND status NOT IN ('completed','customer_declined','archived_unpaid_subscription'){$where_case} ORDER BY converter_followup_at ASC LIMIT 20",
				current_time( 'mysql' )
			) ) ?: [];
			foreach ( $rows as $case ) {
				$staff_id = (int) ( $case->converter_id ?? 0 );
				$staff = $staff_id > 0 ? get_user_by( 'id', $staff_id ) : null;
				$phone = $staff_id > 0 ? $this->user_mobile( $staff_id ) : '';
				$attempt = (int) ( $case->converter_followup_reminder_attempts ?? 0 ) + 1;
				$sent = $phone !== '' && $this->send_configured_sms( 'followup', $phone, [
					'staff_name' => $staff instanceof WP_User ? (string) ( $staff->display_name ?: $staff->user_login ) : 'همکار',
					'customer_name' => (string) $case->customer_name,
					'customer_phone' => (string) $case->customer_phone,
					'case_id' => (int) $case->id,
					'followup_at' => $this->jalali_datetime( (string) $case->converter_followup_at ),
				], [ 'case_id' => (int) $case->id, 'audience_user_ids' => [ $staff_id ] ] );
				$now = current_time( 'mysql' );
				$wpdb->update( $t['cases'], [
					'converter_followup_reminder_attempts' => $attempt,
					'converter_followup_reminded_at' => $sent ? $now : null,
					'converter_followup_reminder_error' => $sent ? null : ( $phone === '' ? 'staff_mobile_not_found' : 'sms_provider_failed' ),
					'updated_at' => $now,
				], [ 'id' => (int) $case->id, 'converter_followup_reminded_at' => null ] );
				$this->log_case( (int) $case->id, 0, $sent ? 'followup_reminder_sent' : 'followup_reminder_failed', (string) $case->status, (string) $case->status, [ 'staff_user_id' => $staff_id, 'attempt' => $attempt, 'due_at' => (string) $case->converter_followup_at ] );
				if ( ! $sent && $attempt < 3 ) { $this->schedule_followup_reminder( (int) $case->id, time() + 15 * MINUTE_IN_SECONDS ); }
			}
		} finally {
			delete_option( $lock_key );
		}
	}

	private function schedule_archive_job(): void {
		if ( ! wp_next_scheduled( self::ARCHIVE_CRON_HOOK ) ) {
			wp_schedule_event( time() + 600, 'hourly', self::ARCHIVE_CRON_HOOK );
		}
	}

	/** Run a bounded archive probe on operational panel visits as a WP-Cron fallback. */
	public function maybe_process_unpaid_archives( $limit = 50 ): void {
		// Keep this public callback defensive: direct calls may still supply a
		// numeric limit, while action dispatchers can supply no value or a legacy
		// empty-string placeholder. Never let an incidental hook value take down
		// the site under PHP 8 strict argument checks.
		$limit = is_numeric( $limit ) ? (int) $limit : 50;
		if ( get_transient( 'sn_dot_unpaid_archive_probe' ) ) { return; }
		set_transient( 'sn_dot_unpaid_archive_probe', '1', MINUTE_IN_SECONDS );
		$this->process_due_unpaid_archives( max( 1, min( 50, $limit ) ) );
	}

	/**
	 * Additive archive worker. It never deletes or invalidates an invoice/payment;
	 * a later approved payment resolves the archive and restores the queue.
	 */
	public function process_due_unpaid_archives( int $limit = 100 ): int {
		global $wpdb;
		$t = $this->tables();
		$limit = max( 1, min( 300, $limit ) );
		if ( ! $this->table_exists( $t['archives'] ) || ! in_array( 'operational_archived_at', $this->columns( $t['cases'] ), true ) ) {
			$this->install_schema();
		}
		if ( ! $this->table_exists( $t['archives'] ) ) { return 0; }
		$lock_key = 'sn_dot_unpaid_archive_worker_lock';
		$lock = (int) get_option( $lock_key, 0 );
		if ( $lock && $lock > time() - 300 ) { return 0; }
		if ( $lock ) { delete_option( $lock_key ); }
		if ( ! add_option( $lock_key, time(), '', false ) ) { return 0; }
		try {
			return $this->archive_due_assessment_invoices( $limit ) + $this->archive_due_subscription_payments( $limit ) + $this->archive_due_direct_product_invoices( $limit );
		} finally {
			delete_option( $lock_key );
		}
	}

	private function sales_manager_for_case( int $seller_id, int $supervisor_id = 0, int $converter_id = 0 ): int {
		if ( ! class_exists( 'SN_Seller_Flow' ) ) { return 0; }
		foreach ( array_unique( array_filter( [ $seller_id, $supervisor_id, $converter_id ] ) ) as $user_id ) {
			$manager_id = SN_Seller_Flow::instance()->sales_manager_for_user( (int) $user_id );
			if ( $manager_id > 0 ) { return $manager_id; }
		}
		return 0;
	}

	private function archive_is_active( string $archive_type, string $entity_kind, int $entity_id ): bool {
		global $wpdb;
		$table = $this->tables()['archives'];
		return (bool) $wpdb->get_var( $wpdb->prepare(
			"SELECT id FROM {$table} WHERE archive_type=%s AND entity_kind=%s AND entity_id=%d AND resolved_at IS NULL LIMIT 1",
			$archive_type,
			$entity_kind,
			$entity_id
		) );
	}

	private function save_operational_archive_in_transaction( array $data ): bool {
		global $wpdb;
		$table = $this->tables()['archives'];
		$type = sanitize_key( (string) ( $data['archive_type'] ?? '' ) );
		$kind = sanitize_key( (string) ( $data['entity_kind'] ?? '' ) );
		$entity_id = absint( $data['entity_id'] ?? 0 );
		if ( $type === '' || $kind === '' || $entity_id < 1 ) { return false; }
		$existing_id = (int) $wpdb->get_var( $wpdb->prepare(
			"SELECT id FROM {$table} WHERE archive_type=%s AND entity_kind=%s AND entity_id=%d LIMIT 1 FOR UPDATE",
			$type,
			$kind,
			$entity_id
		) );
		$payload = [
			'archive_type' => $type,
			'entity_kind' => $kind,
			'entity_id' => $entity_id,
			'invoice_id' => absint( $data['invoice_id'] ?? 0 ) ?: null,
			'case_id' => absint( $data['case_id'] ?? 0 ) ?: null,
			'seller_id' => absint( $data['seller_id'] ?? 0 ) ?: null,
			'supervisor_id' => absint( $data['supervisor_id'] ?? 0 ) ?: null,
			'converter_id' => absint( $data['converter_id'] ?? 0 ) ?: null,
			'sales_manager_user_id' => absint( $data['sales_manager_user_id'] ?? 0 ),
			'customer_name' => sanitize_text_field( (string) ( $data['customer_name'] ?? '' ) ),
			'customer_phone' => SN_Helpers::normalize_mobile( (string) ( $data['customer_phone'] ?? '' ) ),
			'product_title' => sanitize_text_field( (string) ( $data['product_title'] ?? '' ) ),
			'total_amount' => max( 0, (float) ( $data['total_amount'] ?? 0 ) ),
			'paid_amount' => max( 0, (float) ( $data['paid_amount'] ?? 0 ) ),
			'remaining_amount' => max( 0, (float) ( $data['remaining_amount'] ?? 0 ) ),
			'due_at' => (string) ( $data['due_at'] ?? current_time( 'mysql' ) ),
			'archived_at' => (string) ( $data['archived_at'] ?? current_time( 'mysql' ) ),
			'resolved_at' => null,
			'resolved_reason' => null,
			'details_json' => wp_json_encode( (array) ( $data['details'] ?? [] ), JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ),
			'updated_at' => current_time( 'mysql' ),
		];
		if ( $payload['sales_manager_user_id'] < 1 ) { return false; }
		if ( $existing_id > 0 ) { return false !== $wpdb->update( $table, $payload, [ 'id' => $existing_id ] ); }
		$payload['created_at'] = current_time( 'mysql' );
		return (bool) $wpdb->insert( $table, $payload );
	}

	private function archive_due_assessment_invoices( int $limit ): int {
		global $wpdb;
		$t = $this->tables();
		$invoice_table = $wpdb->prefix . 'sn_invoices';
		$stage_table = $wpdb->prefix . 'sn_invoice_payment_stages';
		if ( ! $this->table_exists( $t['links'] ) || ! $this->table_exists( $invoice_table ) || ! $this->table_exists( $stage_table ) ) { return 0; }
		$cutoff = $this->local_mysql_from_timestamp( time() - ( self::ASSESSMENT_UNPAID_DAYS * DAY_IN_SECONDS ) );
		$rows = $wpdb->get_results( $wpdb->prepare(
			"SELECT dl.invoice_id FROM {$t['links']} dl
			 INNER JOIN {$invoice_table} i ON i.id=dl.invoice_id
			 LEFT JOIN {$stage_table} s ON s.invoice_id=i.id AND s.stage_no=COALESCE(i.current_payment_stage,1)
			 WHERE dl.flow_kind='assessment_source' AND dl.case_id IS NULL
			 AND COALESCE(i.remaining_amount,i.payment_total_amount,i.final_total,i.product_price,0)>0.5
			 AND COALESCE(i.status,'') NOT IN ('paid','approved','completed','pending_financial_approval','receipt_uploaded','payment_archived','cancelled')
			 AND ((s.id IS NOT NULL AND s.status='pending' AND s.created_at<=%s)
			      OR (s.id IS NULL AND i.created_at<=%s))
			 ORDER BY COALESCE(s.created_at,i.created_at) ASC,dl.invoice_id ASC LIMIT %d",
			$cutoff,
			$cutoff,
			$limit
		) ) ?: [];
		$archived = 0;
		foreach ( $rows as $candidate ) {
			$invoice_id = (int) $candidate->invoice_id;
			if ( $this->archive_is_active( 'assessment_unpaid', 'invoice', $invoice_id ) ) { continue; }
			$wpdb->query( 'START TRANSACTION' );
			$link = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['links']} WHERE invoice_id=%d AND flow_kind='assessment_source' LIMIT 1 FOR UPDATE", $invoice_id ) );
			$invoice = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$invoice_table} WHERE id=%d LIMIT 1 FOR UPDATE", $invoice_id ) );
			$stage = $invoice ? $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$stage_table} WHERE invoice_id=%d AND stage_no=%d LIMIT 1 FOR UPDATE", $invoice_id, max( 1, (int) ( $invoice->current_payment_stage ?? 1 ) ) ) ) : null;
			$anchor = $stage ? (string) $stage->created_at : (string) ( $invoice->created_at ?? '' );
			$status = (string) ( $invoice->status ?? '' );
			$remaining = $invoice ? max( 0, (float) ( $invoice->remaining_amount ?? $invoice->payment_total_amount ?? $invoice->final_total ?? $invoice->product_price ?? 0 ) ) : 0;
			$eligible = $link && $invoice && empty( $link->case_id ) && $remaining > 0.5 && $anchor !== '' && $anchor <= $cutoff
				&& ! in_array( $status, [ 'paid', 'approved', 'completed', 'pending_financial_approval', 'receipt_uploaded', 'payment_archived', 'cancelled' ], true )
				&& ( ! $stage || (string) $stage->status === 'pending' );
			if ( ! $eligible ) { $wpdb->query( 'COMMIT' ); continue; }
			$manager_id = $this->sales_manager_for_case( (int) $invoice->seller_id, (int) $link->supervisor_id );
			if ( $manager_id < 1 ) { $wpdb->query( 'COMMIT' ); continue; }
			$total = max( 0, (float) ( $invoice->payment_total_amount ?? $invoice->final_total ?? $invoice->product_price ?? 0 ) );
			$paid = max( 0, (float) ( $invoice->paid_total_amount ?? 0 ) );
			$now = current_time( 'mysql' );
			$saved = $this->save_operational_archive_in_transaction( [
				'archive_type' => 'assessment_unpaid', 'entity_kind' => 'invoice', 'entity_id' => $invoice_id,
				'invoice_id' => $invoice_id, 'seller_id' => (int) $invoice->seller_id, 'supervisor_id' => (int) $link->supervisor_id,
				'sales_manager_user_id' => $manager_id, 'customer_name' => (string) $invoice->customer_name, 'customer_phone' => (string) $invoice->customer_phone,
				'product_title' => (string) get_the_title( (int) $invoice->product_id ), 'total_amount' => $total, 'paid_amount' => $paid,
				'remaining_amount' => $remaining, 'due_at' => $this->local_mysql_from_timestamp( $this->mysql_timestamp( $anchor ) + ( self::ASSESSMENT_UNPAID_DAYS * DAY_IN_SECONDS ) ),
				'archived_at' => $now, 'details' => [ 'current_stage' => (int) ( $invoice->current_payment_stage ?? 1 ), 'invoice_status' => $status ],
			] );
			$source_saved = ! class_exists( 'SN_Seller_Flow' ) || SN_Seller_Flow::instance()->mark_assessment_invoice_unpaid( $invoice_id, $manager_id, $now );
			if ( ! $saved || ! $source_saved ) { $wpdb->query( 'ROLLBACK' ); continue; }
			$wpdb->query( 'COMMIT' );
			$archived++;
		}
		return $archived;
	}

	private function archive_due_subscription_payments( int $limit ): int {
		global $wpdb;
		$t = $this->tables();
		if ( ! $this->table_exists( $t['payments'] ) || ! $this->table_exists( $t['cases'] ) ) { return 0; }
		$cutoff = $this->local_mysql_from_timestamp( time() - ( self::SUBSCRIPTION_UNPAID_DAYS * DAY_IN_SECONDS ) );
		$rows = $wpdb->get_results( $wpdb->prepare(
			"SELECT p.id payment_id,p.case_id FROM {$t['payments']} p INNER JOIN {$t['cases']} c ON c.id=p.case_id
			 WHERE p.status='pending' AND p.created_at<=%s AND c.operational_archived_at IS NULL
			 AND c.status NOT IN ('completed','customer_declined','archived_unpaid_subscription')
			 AND NOT EXISTS (SELECT 1 FROM {$t['payments']} newer WHERE newer.case_id=p.case_id AND newer.id>p.id)
			 ORDER BY p.created_at ASC,p.id ASC LIMIT %d",
			$cutoff,
			$limit
		) ) ?: [];
		$archived = 0;
		foreach ( $rows as $candidate ) {
			$payment_id = (int) $candidate->payment_id;
			$case_id = (int) $candidate->case_id;
			$wpdb->query( 'START TRANSACTION' );
			$payment = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['payments']} WHERE id=%d LIMIT 1 FOR UPDATE", $payment_id ) );
			$case = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['cases']} WHERE id=%d LIMIT 1 FOR UPDATE", $case_id ) );
			$newer = $payment ? (int) $wpdb->get_var( $wpdb->prepare( "SELECT COUNT(*) FROM {$t['payments']} WHERE case_id=%d AND id>%d", $case_id, $payment_id ) ) : 1;
			if ( ! $payment || ! $case || (string) $payment->status !== 'pending' || (string) $payment->created_at > $cutoff || $newer > 0 || ! empty( $case->operational_archived_at ) || in_array( (string) $case->status, [ 'completed', 'customer_declined', 'archived_unpaid_subscription' ], true ) ) {
				$wpdb->query( 'COMMIT' );
				continue;
			}
			$manager_id = $this->sales_manager_for_case( (int) $case->seller_id, (int) $case->supervisor_id, (int) $case->converter_id );
			if ( $manager_id < 1 ) { $wpdb->query( 'COMMIT' ); continue; }
			$total = max( 0, (float) $case->selected_option_price );
			$paid = max( 0, (float) $case->paid_amount );
			$remaining = max( 0, $total - $paid );
			$now = current_time( 'mysql' );
			$saved = $this->save_operational_archive_in_transaction( [
				'archive_type' => 'subscription_unpaid', 'entity_kind' => 'dot_case', 'entity_id' => $case_id,
				'invoice_id' => (int) $payment->invoice_id, 'case_id' => $case_id, 'seller_id' => (int) $case->seller_id,
				'supervisor_id' => (int) $case->supervisor_id, 'converter_id' => (int) $case->converter_id, 'sales_manager_user_id' => $manager_id,
				'customer_name' => (string) $case->customer_name, 'customer_phone' => (string) $case->customer_phone,
				'product_title' => (string) ( $case->selected_option_title ?: 'اشتراک انتخاب‌نشده' ), 'total_amount' => $total,
				'paid_amount' => $paid, 'remaining_amount' => $remaining,
				'due_at' => $this->local_mysql_from_timestamp( $this->mysql_timestamp( (string) $payment->created_at ) + ( self::SUBSCRIPTION_UNPAID_DAYS * DAY_IN_SECONDS ) ),
				'archived_at' => $now, 'details' => [ 'payment_id' => $payment_id, 'payment_type' => (string) $payment->payment_type, 'previous_case_status' => (string) $case->status ],
			] );
			$case_saved = false !== $wpdb->update( $t['cases'], [
				'status' => 'archived_unpaid_subscription', 'operational_archived_at' => $now,
				'operational_archive_reason' => 'subscription_unpaid', 'updated_at' => $now,
			], [ 'id' => $case_id, 'operational_archived_at' => null ] );
			if ( ! $saved || ! $case_saved ) { $wpdb->query( 'ROLLBACK' ); continue; }
			$wpdb->query( 'COMMIT' );
			$this->log_case( $case_id, 0, 'auto_archived_subscription_unpaid', (string) $case->status, 'archived_unpaid_subscription', [ 'payment_id' => $payment_id, 'idle_days' => self::SUBSCRIPTION_UNPAID_DAYS, 'manager_user_id' => $manager_id ] );
			$archived++;
		}
		return $archived;
	}

	/** Archive unpaid direct subscription/normal/product* invoices after five days. */
	private function archive_due_direct_product_invoices( int $limit ): int {
		global $wpdb;
		$archives = $this->tables()['archives'];
		$invoices = $wpdb->prefix . 'sn_invoices';
		$stages = $wpdb->prefix . 'sn_invoice_payment_stages';
		$items = $wpdb->prefix . 'sn_invoice_items';
		if ( ! $this->table_exists( $invoices ) || ! $this->table_exists( $stages ) || ! $this->table_exists( $items ) ) { return 0; }
		$links = $this->tables()['links'];
		$link_join = $this->table_exists( $links ) ? "LEFT JOIN {$links} dl ON dl.invoice_id=i.id" : '';
		$link_where = $this->table_exists( $links ) ? ' AND dl.id IS NULL' : '';
		$cutoff = $this->local_mysql_from_timestamp( time() - ( self::SUBSCRIPTION_UNPAID_DAYS * DAY_IN_SECONDS ) );
		$rows = $wpdb->get_results( $wpdb->prepare( "SELECT i.*,s.created_at stage_created,MIN(ii.product_type) product_type,MIN(ii.product_name) product_name FROM {$invoices} i INNER JOIN {$stages} s ON s.invoice_id=i.id AND s.stage_no=COALESCE(i.current_payment_stage,1) INNER JOIN {$items} ii ON ii.invoice_id=i.id {$link_join} WHERE s.status='pending' AND s.created_at<=%s AND COALESCE(i.remaining_amount,i.payment_total_amount,i.final_total,i.product_price,0)>0.5 AND COALESCE(i.status,'') NOT IN ('paid','approved','completed','partial_paid','pending_financial_approval','receipt_uploaded','payment_archived','cancelled'){$link_where} GROUP BY i.id ORDER BY s.created_at ASC,i.id ASC LIMIT %d", $cutoff, $limit ) ) ?: [];
		$count = 0;
		foreach ( $rows as $candidate ) {
			$type = (string) $candidate->product_type;
			$archive_type = $type === 'subscription' ? 'subscription_unpaid' : ( in_array( $type, [ 'product_star', 'subscription_star' ], true ) ? 'product_star_unpaid' : ( $type === 'product' ? 'product_unpaid' : '' ) );
			if ( $archive_type === '' || $this->archive_is_active( $archive_type, 'invoice', (int) $candidate->id ) ) { continue; }
			$manager_id = class_exists( 'SN_Seller_Flow' ) ? SN_Seller_Flow::instance()->sales_manager_for_user( (int) $candidate->seller_id ) : 0;
			if ( $manager_id < 1 ) { continue; }
			$wpdb->query( 'START TRANSACTION' );
			$invoice = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$invoices} WHERE id=%d FOR UPDATE", (int) $candidate->id ) );
			$stage = $invoice ? $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$stages} WHERE invoice_id=%d AND stage_no=%d FOR UPDATE", (int) $candidate->id, max( 1, (int) ( $invoice->current_payment_stage ?? 1 ) ) ) ) : null;
			$remaining = $invoice ? max( 0, (float) ( $invoice->remaining_amount ?? $invoice->payment_total_amount ?? $invoice->final_total ?? $invoice->product_price ?? 0 ) ) : 0;
			if ( ! $invoice || ! $stage || (string) $stage->status !== 'pending' || (string) $stage->created_at > $cutoff || $remaining <= 0.5 || in_array( (string) $invoice->status, [ 'paid','approved','completed','partial_paid','pending_financial_approval','receipt_uploaded','payment_archived','cancelled' ], true ) ) { $wpdb->query( 'COMMIT' ); continue; }
			$now = current_time( 'mysql' );
			$saved = $this->save_operational_archive_in_transaction( [ 'archive_type' => $archive_type, 'entity_kind' => 'invoice', 'entity_id' => (int) $invoice->id, 'invoice_id' => (int) $invoice->id, 'seller_id' => (int) $invoice->seller_id, 'sales_manager_user_id' => $manager_id, 'customer_name' => (string) $invoice->customer_name, 'customer_phone' => (string) $invoice->customer_phone, 'product_title' => (string) $candidate->product_name, 'total_amount' => (float) ( $invoice->payment_total_amount ?? $invoice->final_total ?? $invoice->product_price ?? 0 ), 'paid_amount' => (float) ( $invoice->paid_total_amount ?? 0 ), 'remaining_amount' => $remaining, 'due_at' => $this->local_mysql_from_timestamp( $this->mysql_timestamp( (string) $stage->created_at ) + ( self::SUBSCRIPTION_UNPAID_DAYS * DAY_IN_SECONDS ) ), 'archived_at' => $now, 'details' => [ 'product_type' => $type, 'invoice_status' => (string) $invoice->status ] ] );
			$source_saved = ! class_exists( 'SN_Seller_Flow' ) || SN_Seller_Flow::instance()->mark_invoice_unpaid( (int) $invoice->id, $manager_id, $now, $archive_type );
			if ( ! $saved || ! $source_saved ) { $wpdb->query( 'ROLLBACK' ); continue; }
			$wpdb->query( 'COMMIT' ); $count++;
		}
		return $count;
	}

	private function resolve_operational_archive( string $archive_type, string $entity_kind, int $entity_id, string $reason ): void {
		global $wpdb;
		$t = $this->tables();
		if ( $entity_id < 1 || ! $this->table_exists( $t['archives'] ) ) { return; }
		$now = current_time( 'mysql' );
		$wpdb->update( $t['archives'], [ 'resolved_at' => $now, 'resolved_reason' => sanitize_key( $reason ), 'updated_at' => $now ], [
			'archive_type' => sanitize_key( $archive_type ), 'entity_kind' => sanitize_key( $entity_kind ), 'entity_id' => $entity_id, 'resolved_at' => null,
		] );
		if ( $archive_type === 'assessment_unpaid' && class_exists( 'SN_Seller_Flow' ) ) {
			SN_Seller_Flow::instance()->resolve_assessment_invoice_unpaid( $entity_id );
		} elseif ( in_array( $archive_type, [ 'subscription_unpaid', 'product_unpaid', 'product_star_unpaid' ], true ) && class_exists( 'SN_Seller_Flow' ) ) {
			SN_Seller_Flow::instance()->resolve_invoice_unpaid( $entity_id, $archive_type );
		}
	}

	public function on_invoice_payment_stage_approved( int $invoice_id, $invoice, array $context = [] ): void {
		$this->resolve_operational_archive( 'assessment_unpaid', 'invoice', $invoice_id, 'payment_stage_approved' );
		foreach ( [ 'subscription_unpaid', 'product_unpaid', 'product_star_unpaid' ] as $type ) { $this->resolve_operational_archive( $type, 'invoice', $invoice_id, 'payment_stage_approved' ); }
	}

	public function manager_archive_counts( int $manager_user_id, bool $show_all = false ): array {
		global $wpdb;
		$table = $this->tables()['archives'];
		$counts = [ 'assessment_unpaid' => 0, 'subscription_unpaid' => 0, 'product_unpaid' => 0, 'product_star_unpaid' => 0 ];
		if ( ! $this->table_exists( $table ) ) { return $counts; }
		$where = 'resolved_at IS NULL';
		$args = [];
		if ( ! $show_all ) { $where .= ' AND sales_manager_user_id=%d'; $args[] = $manager_user_id; }
		$sql = "SELECT archive_type,COUNT(*) total FROM {$table} WHERE {$where} GROUP BY archive_type";
		$rows = $args ? $wpdb->get_results( $wpdb->prepare( $sql, ...$args ) ) : $wpdb->get_results( $sql );
		foreach ( $rows ?: [] as $row ) {
			$type = sanitize_key( (string) $row->archive_type );
			if ( array_key_exists( $type, $counts ) ) { $counts[ $type ] = (int) $row->total; }
		}
		return $counts;
	}

	public function render_manager_payment_archive_tab( int $manager_user_id, string $archive_type, bool $show_all = false ): void {
		global $wpdb;
		$table = $this->tables()['archives'];
		$archive_type = in_array( $archive_type, [ 'assessment_unpaid', 'subscription_unpaid', 'product_unpaid', 'product_star_unpaid' ], true ) ? $archive_type : 'assessment_unpaid';
		$where = 'archive_type=%s AND resolved_at IS NULL';
		$args = [ $archive_type ];
		if ( ! $show_all ) { $where .= ' AND sales_manager_user_id=%d'; $args[] = $manager_user_id; }
		$rows = $this->table_exists( $table ) ? ( $wpdb->get_results( $wpdb->prepare( "SELECT * FROM {$table} WHERE {$where} ORDER BY archived_at DESC,id DESC LIMIT 200", ...$args ) ) ?: [] ) : [];
		$title = [ 'assessment_unpaid' => 'عدم پرداخت اعتبارسنجی', 'subscription_unpaid' => 'عدم پرداخت اشتراک', 'product_unpaid' => 'عدم پرداخت محصول', 'product_star_unpaid' => 'عدم پرداخت محصول*' ][ $archive_type ];
		$days = $archive_type === 'assessment_unpaid' ? self::ASSESSMENT_UNPAID_DAYS : self::SUBSCRIPTION_UNPAID_DAYS;
		?>
		<p class="sn-note"><?php echo esc_html( sprintf( 'مواردی که لینک پرداخت فعال آن‌ها طی %d روز تکمیل نشده است. پرداخت دیرهنگام، بایگانی را خودکار حل می‌کند.', $days ) ); ?></p>
		<div class="sn-table-wrap"><table class="sn-table"><thead><tr><th>مشتری</th><th>محصول</th><th>مالک پرونده</th><th>مبلغ کل</th><th>پرداخت‌شده</th><th>مانده</th><th>موعد</th><th>بایگانی</th></tr></thead><tbody>
		<?php if ( ! $rows ) : ?><tr><td colspan="8"><div class="sn-empty-state">موردی در بایگانی «<?php echo esc_html( $title ); ?>» نیست.</div></td></tr><?php endif; ?>
		<?php foreach ( $rows as $row ) :
			$owner_id = (int) ( $row->converter_id ?: ( $row->supervisor_id ?: $row->seller_id ) );
			$owner = $owner_id > 0 ? get_user_by( 'id', $owner_id ) : null;
		?>
		<tr><td><strong><?php echo esc_html( (string) ( $row->customer_name ?: 'بدون نام' ) ); ?></strong><br><code><?php echo esc_html( (string) ( $row->customer_phone ?: '—' ) ); ?></code></td><td><strong><?php echo esc_html( (string) ( $row->product_title ?: '—' ) ); ?></strong><br><small>#<?php echo esc_html( (string) ( $row->case_id ?: $row->invoice_id ) ); ?></small></td><td><?php echo esc_html( $owner instanceof WP_User ? (string) $owner->display_name : ( $owner_id > 0 ? '#' . $owner_id : '—' ) ); ?></td><td><?php echo esc_html( number_format_i18n( (float) $row->total_amount ) ); ?> تومان</td><td><?php echo esc_html( number_format_i18n( (float) $row->paid_amount ) ); ?> تومان</td><td><strong><?php echo esc_html( number_format_i18n( (float) $row->remaining_amount ) ); ?> تومان</strong></td><td><?php echo esc_html( $this->jalali_datetime( (string) $row->due_at ) ); ?></td><td><?php echo esc_html( $this->jalali_datetime( (string) $row->archived_at ) ); ?></td></tr>
		<?php endforeach; ?>
		</tbody></table></div>
		<?php
	}

	/** Recover a small batch after an interrupted hook or transient DB error. */
	private function reconcile_paid_links(): void {
		global $wpdb;
		$t = $this->tables();
		if ( ! $this->table_exists( $t['links'] ) || ! $this->table_exists( $t['cases'] ) || ! $this->table_exists( $t['payments'] ) ) { return; }
		$invoice_table = $wpdb->prefix . 'sn_invoices';
		if ( ! empty( $this->config()['enabled'] ) ) {
			$sources = $wpdb->get_results( "SELECT dl.* FROM {$t['links']} dl INNER JOIN {$invoice_table} i ON i.id=dl.invoice_id WHERE dl.flow_kind='assessment_source' AND dl.case_id IS NULL AND i.status IN ('paid','approved') ORDER BY dl.id ASC LIMIT 5" ) ?: [];
			foreach ( $sources as $link ) {
				$invoice = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$invoice_table} WHERE id=%d", (int) $link->invoice_id ) );
				if ( $invoice ) { $this->create_case_for_source_invoice( $link, $invoice ); }
			}
		}
		$conversions = $wpdb->get_results( "SELECT dl.* FROM {$t['links']} dl INNER JOIN {$invoice_table} i ON i.id=dl.invoice_id INNER JOIN {$t['payments']} dp ON dp.invoice_id=dl.invoice_id WHERE dl.flow_kind='conversion_payment' AND dp.status<>'paid' AND i.status IN ('paid','approved') ORDER BY dl.id ASC LIMIT 5" ) ?: [];
		foreach ( $conversions as $link ) {
			$invoice = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$invoice_table} WHERE id=%d", (int) $link->invoice_id ) );
			if ( $invoice ) { $this->complete_conversion_payment( $link, $invoice ); }
		}
	}

	/**
	 * Repair paid assessment links belonging to one supervisor before rendering.
	 *
	 * This is intentionally independent of customer option selection: payment and
	 * finance approval make the assessment an actionable conversion case. An empty
	 * selected_option_key is rendered as «انتخاب نکرده» and can still be assigned.
	 */
	private function repair_paid_assessment_cases_for_supervisor( int $supervisor_id, int $limit = 20 ): int {
		global $wpdb;
		$t = $this->tables();
		$supervisor_id = absint( $supervisor_id );
		$limit = max( 1, min( 50, $limit ) );
		$invoice_table = $wpdb->prefix . 'sn_invoices';
		if ( $supervisor_id < 1 || ! $this->is_supervisor( $supervisor_id ) || ! $this->table_exists( $t['links'] ) || ! $this->table_exists( $t['cases'] ) || ! $this->table_exists( $invoice_table ) ) { return 0; }

		$paid_statuses = "'paid','approved','financial_approved','completed','success','gateway_paid','online_paid'";
		$links = $wpdb->get_results( $wpdb->prepare(
			"SELECT dl.* FROM {$t['links']} dl
			 INNER JOIN {$invoice_table} i ON i.id=dl.invoice_id
			 LEFT JOIN {$t['cases']} linked_case ON linked_case.id=dl.case_id
			 LEFT JOIN {$t['cases']} source_case ON source_case.source_invoice_id=dl.invoice_id
			 WHERE dl.flow_kind='assessment_source'
			 AND dl.supervisor_id=%d
			 AND linked_case.id IS NULL
			 AND source_case.id IS NULL
			 AND (
				i.status IN ({$paid_statuses})
				OR i.invoice_status IN ({$paid_statuses})
				OR i.payment_status IN ({$paid_statuses})
				OR (COALESCE(i.payment_workflow_status,'')='completed' AND COALESCE(i.remaining_amount,0)<=0.5)
			 )
			 ORDER BY dl.id ASC LIMIT %d",
			$supervisor_id,
			$limit
		) ) ?: [];

		$repaired = 0;
		foreach ( $links as $link ) {
			$invoice = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$invoice_table} WHERE id=%d LIMIT 1", (int) $link->invoice_id ) );
			if ( $invoice && $this->create_case_for_source_invoice( $link, $invoice ) > 0 ) { $repaired++; }
		}
		return $repaired;
	}

	/**
	 * Adopt legacy paid cases that were routed to an ordinary/configured supervisor
	 * instead of the seller's direct supervisor. The seller scope is authoritative and
	 * every row is rechecked by resolve_default_supervisor_for_seller().
	 */
	private function reconcile_paid_assessment_scope_for_supervisor( int $supervisor_id, int $limit = 100 ): int {
		global $wpdb;
		$t = $this->tables();
		$supervisor_id = absint( $supervisor_id );
		$limit = max( 1, min( 250, $limit ) );
		$invoice_table = $wpdb->prefix . 'sn_invoices';
		if ( $supervisor_id < 1 || ! $this->is_supervisor( $supervisor_id ) || ! $this->table_exists( $t['links'] ) || ! $this->table_exists( $t['cases'] ) || ! $this->table_exists( $invoice_table ) ) { return 0; }

		$seller_ids = [];
		if ( class_exists( 'SN_Scope_Service' ) ) {
			$seller_ids = ( new SN_Scope_Service() )->scope_visible_seller_ids( $supervisor_id );
		}
		if ( ! $seller_ids ) {
			$seller_ids = get_users( [ 'role' => 'sn_seller', 'fields' => 'ids', 'meta_key' => 'sn_supervisor_id', 'meta_value' => $supervisor_id ] );
		}
		$seller_ids = array_values( array_unique( array_filter( array_map( 'absint', (array) $seller_ids ) ) ) );
		if ( ! $seller_ids ) { return 0; }

		$placeholders = implode( ',', array_fill( 0, count( $seller_ids ), '%d' ) );
		$paid_statuses = "'paid','approved','financial_approved','completed','success','gateway_paid','online_paid'";
		$args = array_merge( $seller_ids, [ $supervisor_id, $limit ] );
		$link_ids = $wpdb->get_col( $wpdb->prepare(
			"SELECT dl.id FROM {$t['links']} dl
			 INNER JOIN {$invoice_table} i ON i.id=dl.invoice_id
			 LEFT JOIN {$t['cases']} source_case ON source_case.source_invoice_id=dl.invoice_id
			 WHERE dl.flow_kind='assessment_source'
			 AND COALESCE(dl.marketing_form_id,'')=''
			 AND i.seller_id IN ({$placeholders})
			 AND COALESCE(dl.supervisor_id,0)<>%d
			 AND (source_case.id IS NULL OR COALESCE(source_case.converter_id,0)=0 OR source_case.conversion_route='seller_self')
			 AND (
				i.status IN ({$paid_statuses})
				OR i.invoice_status IN ({$paid_statuses})
				OR i.payment_status IN ({$paid_statuses})
				OR (COALESCE(i.payment_workflow_status,'')='completed' AND COALESCE(i.remaining_amount,0)<=0.5)
			 )
			 ORDER BY dl.id DESC LIMIT %d",
			...$args
		) ) ?: [];

		$reconciled = 0;
		foreach ( array_map( 'absint', $link_ids ) as $link_id ) {
			$link = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['links']} WHERE id=%d LIMIT 1", $link_id ) );
			$invoice = $link ? $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$invoice_table} WHERE id=%d LIMIT 1", (int) $link->invoice_id ) ) : null;
			if ( ! $link || ! $invoice || $this->resolve_default_supervisor_for_seller( (int) $invoice->seller_id ) !== $supervisor_id ) { continue; }
			$before = (int) $link->supervisor_id;
			$link = $this->reconcile_regular_assessment_link_supervisor( $link, $invoice );
			if ( $before !== (int) $link->supervisor_id && (int) $link->supervisor_id === $supervisor_id ) { $reconciled++; }
		}
		// Referral cases created by older versions have no assessment link to
		// reconcile. Move only still-unassigned rows; their immutable origin row,
		// timestamps, invoice ledger and any active assignment remain untouched.
		$remaining_limit = max( 0, $limit - $reconciled );
		if ( $remaining_limit > 0 && in_array( 'referral_item_id', $this->columns( $t['cases'] ), true ) ) {
			$case_args = array_merge( $seller_ids, [ $supervisor_id, $remaining_limit ] );
			$legacy_cases = $wpdb->get_results( $wpdb->prepare(
				"SELECT id,supervisor_id,status,seller_id FROM {$t['cases']}
				 WHERE referral_item_id IS NOT NULL
				 AND seller_id IN ({$placeholders})
				 AND COALESCE(supervisor_id,0)<>%d
				 AND COALESCE(converter_id,0)=0
				 AND operational_archived_at IS NULL
				 ORDER BY id DESC LIMIT %d",
				...$case_args
			) ) ?: [];
			foreach ( $legacy_cases as $legacy_case ) {
				$updated = $wpdb->update( $t['cases'], [ 'supervisor_id' => $supervisor_id, 'updated_at' => current_time( 'mysql' ) ], [ 'id' => (int) $legacy_case->id, 'supervisor_id' => (int) $legacy_case->supervisor_id ] );
				if ( 1 === $updated ) {
					$reconciled++;
					$this->log_case( (int) $legacy_case->id, 0, 'supervisor_queue_scope_reconciled', (string) $legacy_case->status, (string) $legacy_case->status, [
						'old_supervisor_id' => (int) $legacy_case->supervisor_id,
						'new_supervisor_id' => $supervisor_id,
						'seller_id' => (int) $legacy_case->seller_id,
					] );
				}
			}
		}
		return $reconciled;
	}

	/** Recover paid seller-self assessments when the original paid hook was missed. */
	private function repair_paid_assessment_cases_for_seller( int $seller_id, int $limit = 20 ): int {
		global $wpdb;
		$t = $this->tables();
		$seller_id = absint( $seller_id );
		$limit = max( 1, min( 50, $limit ) );
		$invoice_table = $wpdb->prefix . 'sn_invoices';
		if ( $seller_id < 1 || ! $this->is_seller( $seller_id ) || ! $this->table_exists( $t['links'] ) || ! $this->table_exists( $t['cases'] ) || ! $this->table_exists( $invoice_table ) ) { return 0; }

		$paid_statuses = "'paid','approved','financial_approved','completed','success','gateway_paid','online_paid'";
		$links = $wpdb->get_results( $wpdb->prepare(
			"SELECT dl.* FROM {$t['links']} dl
			 INNER JOIN {$invoice_table} i ON i.id=dl.invoice_id
			 LEFT JOIN {$t['cases']} linked_case ON linked_case.id=dl.case_id
			 LEFT JOIN {$t['cases']} source_case ON source_case.source_invoice_id=dl.invoice_id
			 WHERE dl.flow_kind='assessment_source'
			 AND dl.conversion_route='seller_self'
			 AND i.seller_id=%d
			 AND linked_case.id IS NULL
			 AND source_case.id IS NULL
			 AND (
				i.status IN ({$paid_statuses})
				OR i.invoice_status IN ({$paid_statuses})
				OR i.payment_status IN ({$paid_statuses})
				OR (COALESCE(i.payment_workflow_status,'')='completed' AND COALESCE(i.remaining_amount,0)<=0.5)
			 )
			 ORDER BY dl.id ASC LIMIT %d",
			$seller_id,
			$limit
		) ) ?: [];

		$repaired = 0;
		foreach ( $links as $link ) {
			$invoice = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$invoice_table} WHERE id=%d LIMIT 1", (int) $link->invoice_id ) );
			if ( $invoice && $this->create_case_for_source_invoice( $link, $invoice ) > 0 ) { $repaired++; }
		}
		return $repaired;
	}

	public function process_due_access_sms( int $case_id = 0 ): void {
		global $wpdb;
		$t = $this->tables();
		if ( empty( $this->config()['enabled'] ) || ! $this->table_exists( $t['cases'] ) ) { return; }
		$lock_key = 'sn_dot_sms_worker_lock';
		$lock = (int) get_option( $lock_key, 0 );
		if ( $lock && $lock > time() - 120 ) { return; }
		if ( $lock ) { delete_option( $lock_key ); }
		if ( ! add_option( $lock_key, time(), '', false ) ) { return; }
		try {
			$where = "sms_sent_at IS NULL AND sms_attempts<3 AND sms_due_at IS NOT NULL AND sms_due_at<=" . $wpdb->prepare( '%s', current_time( 'mysql' ) );
			if ( in_array( 'access_sms_enabled', $this->columns( $t['cases'] ), true ) ) { $where .= ' AND access_sms_enabled=1'; }
			if ( $this->is_sms_test_mode() && in_array( 'sms_previewed_at', $this->columns( $t['cases'] ), true ) ) { $where .= ' AND sms_previewed_at IS NULL'; }
			if ( $case_id > 0 ) { $where .= $wpdb->prepare( ' AND id=%d', $case_id ); }
			$cases = $wpdb->get_results( "SELECT * FROM {$t['cases']} WHERE {$where} ORDER BY sms_due_at ASC LIMIT 20" ) ?: [];
			foreach ( $cases as $case ) { $this->send_access_sms( $case ); }
		} finally {
			delete_option( $lock_key );
		}
	}

	private function customer_flow_url( int $case_id, string $token ): string {
		$page_id = (int) get_option( 'sn_dot_customer_page_id', 0 );
		$base = $page_id ? get_permalink( $page_id ) : home_url( '/credit-assessment/' );
		return add_query_arg( [ 'dot_case' => $case_id, 'dot_token' => $token ], $base ?: home_url( '/' ) );
	}

	private function short_access_signature( object $case ): string {
		$seed = implode( '|', [ (int) $case->id, (int) $case->source_invoice_id, SN_Helpers::normalize_mobile( (string) $case->customer_phone ) ] );
		return substr( hash_hmac( 'sha256', 'sn-dot-short-link|' . $seed, wp_salt( 'auth' ) ), 0, 14 );
	}

	private function customer_short_url( object $case ): string {
		return home_url( '/d/' . (int) $case->id . '-' . $this->short_access_signature( $case ) . '/' );
	}

	public function resolve_short_access_url( int $case_id, string $signature ): string {
		$case = $this->get_case( $case_id );
		if ( ! $case || $signature === '' || ! hash_equals( $this->short_access_signature( $case ), strtolower( $signature ) ) ) { return ''; }
		$token = $this->persistent_access_token( $case );
		global $wpdb;
		$t = $this->tables();
		$wpdb->update( $t['cases'], [
			'access_token_hash' => hash( 'sha256', $token ),
			'access_expires_at' => null,
			'access_locked_until' => null,
			'updated_at' => current_time( 'mysql' ),
		], [ 'id' => $case_id ] );
		return $this->customer_flow_url( $case_id, $token );
	}

	private function persistent_access_token( object $case ): string {
		$seed = implode( '|', [ (int) $case->id, (int) $case->source_invoice_id, SN_Helpers::normalize_mobile( (string) $case->customer_phone ) ] );
		return hash_hmac( 'sha256', 'sn-dot-customer-link|' . $seed, wp_salt( 'auth' ) );
	}

	private function persistent_access_code( object $case ): string {
		$seed = implode( '|', [ (int) $case->id, (int) $case->source_invoice_id, SN_Helpers::normalize_mobile( (string) $case->customer_phone ) ] );
		$hash = hash_hmac( 'sha256', 'sn-dot-customer-code|' . $seed, wp_salt( 'auth' ) );
		return (string) ( 100000 + ( hexdec( substr( $hash, 0, 7 ) ) % 900000 ) );
	}

	private function send_access_sms( object $case ): bool {
		global $wpdb;
		$t = $this->tables();
		if ( isset( $case->access_sms_enabled ) && ! (int) $case->access_sms_enabled ) { return false; }
		$is_test_preview = $this->is_sms_test_mode();
		$is_validation_gift = ! empty( $case->validation_fee_gifted );
		$sms_type = $is_validation_gift ? 'gift_access' : 'access';
		$has_preview_columns = in_array( 'sms_previewed_at', $this->columns( $t['cases'] ), true ) && in_array( 'sms_delivery_mode', $this->columns( $t['cases'] ), true );
		$token = $this->persistent_access_token( $case );
		$code = $this->persistent_access_code( $case );
		$now = current_time( 'mysql' );
		$credential_data = [
			'access_token_hash' => hash( 'sha256', $token ),
			'access_code_hash' => wp_hash_password( $code ),
			'access_expires_at' => null,
			'access_failed_attempts' => 0,
			'access_locked_until' => null,
			'sms_attempts' => (int) $case->sms_attempts + ( $is_test_preview ? 0 : 1 ),
			'updated_at' => $now,
		];
		$stored = $wpdb->update( $t['cases'], $credential_data, [ 'id' => (int) $case->id ] );
		if ( false === $stored ) { return false; }
		$link = $this->customer_short_url( $case );
		$vars = [
			'customer_name' => (string) $case->customer_name,
			'access_code' => $code,
			'code' => $code,
			'access_note' => $is_validation_gift ? 'هزینه اعتبارسنجی از طرف شرکت به‌صورت هدیه پرداخت شده است.' : 'این دسترسی اختصاصی برای شماست.',
			'gift_message' => $is_validation_gift ? 'هزینه اعتبارسنجی از طرف شرکت به‌صورت هدیه پرداخت شده است.' : '',
			'access_link' => $link,
			'link' => $link,
			'nominal_credit' => number_format( (float) $case->nominal_credit_amount, 0, '.', ',' ),
			'invoice_code' => $this->source_invoice_code( (int) $case->source_invoice_id ),
		];
		$sent = $this->send_configured_sms( $sms_type, (string) $case->customer_phone, $vars, [
			'case_id' => (int) $case->id,
			'invoice_id' => (int) $case->source_invoice_id,
		] );
		if ( $sent ) {
			if ( $is_test_preview && $has_preview_columns ) {
				$preview_delivery = [ 'sms_previewed_at' => $now, 'sms_delivery_mode' => 'test', 'sms_last_error' => null, 'status' => 'awaiting_customer_selection', 'updated_at' => $now ];
				if ( $is_validation_gift ) { $preview_delivery['validation_gift_sms_error'] = null; }
				$wpdb->update( $t['cases'], $this->existing_columns( $t['cases'], $preview_delivery ), [ 'id' => (int) $case->id ] );
				$this->log_case( (int) $case->id, 0, 'access_sms_previewed', (string) $case->status, 'awaiting_customer_selection' );
			} else {
				$delivery = [ 'sms_sent_at' => $now, 'sms_last_error' => null, 'status' => 'awaiting_customer_selection', 'updated_at' => $now ];
				if ( $is_validation_gift ) { $delivery['validation_gift_sms_sent_at'] = $now; $delivery['validation_gift_sms_error'] = null; }
				if ( $has_preview_columns ) { $delivery['sms_delivery_mode'] = 'real'; }
				$wpdb->update( $t['cases'], $this->existing_columns( $t['cases'], $delivery ), [ 'id' => (int) $case->id ] );
				$this->log_case( (int) $case->id, 0, 'access_sms_sent', (string) $case->status, 'awaiting_customer_selection', [ 'attempt' => (int) $case->sms_attempts + 1 ] );
			}
			return true;
		}
		$attempts = (int) $case->sms_attempts + ( $is_test_preview ? 0 : 1 );
		$failure = [ 'sms_last_error' => 'sms_provider_failed', 'updated_at' => $now ];
		if ( $is_validation_gift ) { $failure['validation_gift_sms_error'] = 'sms_provider_failed'; }
		$wpdb->update( $t['cases'], $this->existing_columns( $t['cases'], $failure ), [ 'id' => (int) $case->id ] );
		if ( $attempts < 3 ) {
			$retry_at = time() + 10 * MINUTE_IN_SECONDS;
			$wpdb->update( $t['cases'], [ 'sms_due_at' => $this->local_mysql_from_timestamp( $retry_at ) ], [ 'id' => (int) $case->id ] );
			$this->schedule_case_sms( (int) $case->id, $retry_at );
		}
		$this->log_case( (int) $case->id, 0, 'access_sms_failed', (string) $case->status, (string) $case->status, [ 'attempt' => $attempts ] );
		return false;
	}

	private function source_invoice_code( int $invoice_id ): string {
		global $wpdb;
		return (string) $wpdb->get_var( $wpdb->prepare( "SELECT invoice_code FROM {$wpdb->prefix}sn_invoices WHERE id=%d", $invoice_id ) );
	}

	private function test_sms_run_id( array $config ): string {
		$run_id = sanitize_text_field( (string) ( $config['sms_test_run_id'] ?? '' ) );
		if ( $run_id !== '' ) { return $run_id; }
		return substr( hash_hmac( 'sha256', 'sn-dot-test-sms', wp_salt( 'nonce' ) ), 0, 24 );
	}

	private function render_sms_message( string $type, string $template, array $vars ): string {
		if ( trim( $template ) === '' ) {
			$template = (string) ( $this->default_config()['sms'][ $type . '_template' ] ?? '' );
		}
		$replace = [];
		foreach ( $vars as $key => $value ) {
			$replace[ '{' . $key . '}' ] = (string) $value;
			$replace[ '%' . $key . '%' ] = (string) $value;
		}
		return trim( strtr( $template, $replace ) );
	}

	private function test_sms_type_label( string $type ): string {
		return [
			'access' => 'ورود اعتبارسنجی',
			'gift_access' => 'اعتبارسنجی هدیه و لینک انتخاب',
			'payment' => 'لینک پرداخت',
			'reject' => 'رد تأیید مالی',
			'followup' => 'یادآور تماس مجدد',
		][ $type ] ?? 'پیامک فلو';
	}

	private function test_sms_audience( array $context ): array {
		$ids = array_map( 'absint', (array) ( $context['audience_user_ids'] ?? [] ) );
		$current_user_id = get_current_user_id();
		if ( ! empty( $context['include_current_user'] ) && $current_user_id > 0 ) { $ids[] = $current_user_id; }

		$case_id = absint( $context['case_id'] ?? 0 );
		if ( $case_id > 0 ) {
			$case = $this->get_case( $case_id );
			if ( $case ) {
				$ids[] = (int) $case->supervisor_id;
				$ids[] = (int) $case->converter_id;
				if ( ! empty( $context['include_case_seller'] ) ) { $ids[] = (int) $case->seller_id; }
			}
		}

		$invoice_id = absint( $context['invoice_id'] ?? 0 );
		if ( $invoice_id > 0 ) {
			global $wpdb;
			$t = $this->tables();
			$link = $this->table_exists( $t['links'] ) ? $wpdb->get_row( $wpdb->prepare( "SELECT supervisor_id,case_id FROM {$t['links']} WHERE invoice_id=%d LIMIT 1", $invoice_id ) ) : null;
			if ( $link ) {
				$ids[] = (int) $link->supervisor_id;
				if ( ! $case_id && (int) $link->case_id > 0 ) {
					$case = $this->get_case( (int) $link->case_id );
					if ( $case ) { $ids[] = (int) $case->converter_id; }
				}
			}
		}

		return array_values( array_unique( array_filter( $ids ) ) );
	}

	private function queue_test_sms( string $type, string $phone, string $message, array $context, array $config ): void {
		$run_id = $this->test_sms_run_id( $config );
		$now = time();
		$test_viewer_user_id = absint( $config['sms_test_viewer_user_id'] ?? 0 );
		if ( $test_viewer_user_id > 0 ) {
			$context['audience_user_ids'] = array_merge( (array) ( $context['audience_user_ids'] ?? [] ), [ $test_viewer_user_id ] );
		}
		$item = [
			'id' => wp_generate_password( 12, false, false ),
			'run_id' => $run_id,
			'type' => sanitize_key( $type ),
			'type_label' => $this->test_sms_type_label( $type ),
			'phone' => SN_Helpers::normalize_mobile( $phone ),
			'message' => sanitize_textarea_field( $message ),
			'case_id' => absint( $context['case_id'] ?? 0 ),
			'invoice_id' => absint( $context['invoice_id'] ?? 0 ),
			'created_at' => SN_Helpers::tehran_format( 'c', $now ),
			'created_label' => SN_Helpers::tehran_format( 'H:i', $now ),
			'expires_at' => $now + self::TEST_SMS_QUEUE_TTL,
		];
		foreach ( $this->test_sms_audience( $context ) as $user_id ) {
			$queue_key = self::TEST_SMS_QUEUE_PREFIX . $user_id;
			$queue = get_transient( $queue_key );
			$queue = is_array( $queue ) ? $queue : [];
			$queue = array_values( array_filter( $queue, static function ( $queued ) use ( $now, $run_id ) {
				return is_array( $queued ) && (int) ( $queued['expires_at'] ?? 0 ) > $now && hash_equals( $run_id, (string) ( $queued['run_id'] ?? '' ) );
			} ) );
			$queue[] = $item;
			set_transient( $queue_key, array_slice( $queue, -12 ), self::TEST_SMS_QUEUE_TTL );
		}
	}

	private function clear_test_sms_queue_for_user( int $user_id ): void {
		if ( $user_id > 0 ) { delete_transient( self::TEST_SMS_QUEUE_PREFIX . $user_id ); }
	}

	private function drain_test_sms_queue( int $user_id ): array {
		$config = $this->config();
		if ( empty( $config['sms_test_mode'] ) || $user_id <= 0 ) { return []; }
		$run_id = $this->test_sms_run_id( $config );
		$queue_key = self::TEST_SMS_QUEUE_PREFIX . $user_id;
		$queue = get_transient( $queue_key );
		$queue = is_array( $queue ) ? $queue : [];
		$now = time();
		$messages = [];
		$remaining = [];
		foreach ( $queue as $item ) {
			if ( ! is_array( $item ) || (int) ( $item['expires_at'] ?? 0 ) <= $now || ! hash_equals( $run_id, (string) ( $item['run_id'] ?? '' ) ) ) { continue; }
			if ( count( $messages ) < 5 ) {
				unset( $item['run_id'], $item['expires_at'] );
				$messages[] = $item;
			} else {
				$remaining[] = $item;
			}
		}
		if ( $remaining ) { set_transient( $queue_key, $remaining, self::TEST_SMS_QUEUE_TTL ); }
		else { delete_transient( $queue_key ); }
		return $messages;
	}

	public function handle_poll_test_sms(): void {
		check_ajax_referer( 'sn_dot_test_sms', 'nonce' );
		nocache_headers();
		if ( ! is_user_logged_in() ) { wp_send_json_error( [ 'message' => 'نشست کاربری معتبر نیست.' ], 401 ); }
		$config = $this->config();
		if ( empty( $config['sms_test_mode'] ) ) {
			$this->clear_test_sms_queue_for_user( get_current_user_id() );
			wp_send_json_success( [ 'enabled' => false, 'run_id' => '', 'messages' => [] ] );
		}
		$current_run_id = $this->test_sms_run_id( $config );
		$request_run_id = sanitize_text_field( wp_unslash( $_POST['run_id'] ?? '' ) );
		if ( $request_run_id === '' || ! hash_equals( $current_run_id, $request_run_id ) ) {
			// Do not drain a current queue into a stale browser tab.
			wp_send_json_success( [ 'enabled' => true, 'run_id' => $current_run_id, 'messages' => [] ] );
		}
		wp_send_json_success( [
			'enabled' => true,
			'run_id' => $current_run_id,
			'messages' => $this->drain_test_sms_queue( get_current_user_id() ),
		] );
	}

	private function send_configured_sms( string $type, string $phone, array $vars, array $context = [] ): bool {
		$config = $this->config();
		$sms_config = (array) $config['sms'];
		$pattern = trim( (string) ( $sms_config[ $type . '_pattern' ] ?? '' ) );
		$template = (string) ( $sms_config[ $type . '_template' ] ?? '' );
		$message = $this->render_sms_message( $type, $template, $vars );
		if ( ! empty( $config['sms_test_mode'] ) ) {
			if ( $message === '' ) { return false; }
			$this->queue_test_sms( $type, $phone, $message, $context, $config );
			return true;
		}
		$sms = new SN_SMS();
		$sms_sent = false;
		if ( $pattern !== '' && (string) get_option( 'sn_sms_provider', '' ) === 'faraz' ) {
			$sms_sent = (bool) $sms->send_faraz_pattern( SN_Helpers::normalize_mobile( $phone ), $pattern, array_map( 'strval', $vars ) );
			// render_sms_message() already supplies the safe built-in template when
			// the optional custom template is blank. Do not suppress that fallback
			// merely because a configured Faraz pattern failed.
		}
		if ( ! $sms_sent && $message !== '' ) { $sms_sent = (bool) $sms->send( $phone, $message ); }
		if ( $message !== '' && class_exists( 'SN_Notification_Service' ) ) {
			$buttons = [];
			if ( ! empty( $vars['invoice_url'] ) ) { $buttons[] = [ [ 'text' => 'مشاهده و پرداخت', 'url' => esc_url_raw( (string) $vars['invoice_url'] ) ] ]; }
			elseif ( ! empty( $vars['access_link'] ) ) { $buttons[] = [ [ 'text' => 'مشاهده', 'url' => esc_url_raw( (string) $vars['access_link'] ) ] ]; }
			$entity_id = absint( $context['case_id'] ?? $context['invoice_id'] ?? 0 );
			SN_Notification_Service::instance()->mirror_text( 'dot.' . sanitize_key( $type ), $phone, $message, [
				'entity_type' => ! empty( $context['case_id'] ) ? 'dot_case' : ( ! empty( $context['invoice_id'] ) ? 'invoice' : 'dot' ),
				'entity_id' => $entity_id,
				'case_id' => absint( $context['case_id'] ?? 0 ),
				'invoice_id' => absint( $context['invoice_id'] ?? 0 ),
				'dedupe_key' => 'dot-' . sanitize_key( $type ) . '-' . $entity_id . '-' . hash( 'sha256', $message ),
				'source' => 'dot_flow',
				'buttons' => $buttons,
			] );
		}
		return $sms_sent;
	}

	/**
	 * Send an invoice link through the canonical Sales Cycle payment SMS setup.
	 *
	 * Other modules must use this boundary instead of rebuilding a plain-text
	 * payment message, otherwise the configured Faraz pattern and test mode are
	 * silently bypassed.
	 */
	public function send_sales_cycle_payment_sms( string $phone, array $vars, array $context = [] ): bool {
		$defaults = [
			'customer_name'   => 'مشتری گرامی',
			'option_title'    => '',
			'amount'          => '0',
			'payment_type'    => 'پرداخت',
			'invoice_url'     => '',
			'remaining_amount'=> '0',
		];
		return $this->send_configured_sms( 'payment', $phone, array_merge( $defaults, $vars ), $context );
	}

	public function on_financial_rejected( int $invoice_id, $invoice, string $reason ): void {
		global $wpdb;
		$t = $this->tables();
		$link = $this->table_exists( $t['links'] ) ? $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['links']} WHERE invoice_id=%d LIMIT 1", $invoice_id ) ) : null;
		if ( ! $link ) { return; }
		$inv = is_object( $invoice ) ? $invoice : (object) $invoice;
		$wpdb->query( $wpdb->prepare( "UPDATE {$wpdb->prefix}sn_payments SET status='rejected',updated_at=%s WHERE invoice_id=%d AND status='verified'", current_time( 'mysql' ), $invoice_id ) );
		if ( $this->table_exists( $t['marketing_leads'] ) ) {
			$rejected_now = current_time( 'mysql' );
			$marketing_submission_id = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$t['marketing_leads']} WHERE invoice_id=%d LIMIT 1", $invoice_id ) );
			$wpdb->update( $t['marketing_leads'], [
				'status' => 'payment_rejected',
				'last_event' => 'payment_rejected',
				'last_event_reason' => sanitize_text_field( $reason ),
				'last_event_at' => $rejected_now,
				'gateway_returned_at' => $rejected_now,
				'updated_at' => $rejected_now,
			], [ 'invoice_id' => $invoice_id ] );
			if ( $marketing_submission_id > 0 ) { $this->record_marketing_event( $marketing_submission_id, 'payment_rejected', 'failed', sanitize_text_field( $reason ), [ 'invoice_id' => $invoice_id ], 'invoice-' . $invoice_id ); }
		}
		if ( (string) $link->flow_kind === 'assessment_source' ) {
			$seller = get_user_by( 'id', (int) ( $inv->seller_id ?? 0 ) );
			$phone = $seller instanceof WP_User ? $this->user_mobile( (int) $seller->ID ) : '';
			if ( $phone !== '' ) {
				$this->send_configured_sms( 'reject', $phone, [
					'seller_name' => $seller instanceof WP_User ? (string) $seller->display_name : 'فروشنده',
					'customer_name' => (string) ( $inv->customer_name ?? '' ),
					'customer_phone' => (string) ( $inv->customer_phone ?? '' ),
					'invoice_code' => (string) ( $inv->invoice_code ?? '' ),
					'reason' => $reason,
				], [
					'invoice_id' => $invoice_id,
					'audience_user_ids' => [ (int) $seller->ID, (int) $link->supervisor_id, get_current_user_id() ],
				] );
			}
			return;
		}
		if ( (string) $link->flow_kind === 'conversion_payment' && (int) $link->case_id > 0 ) {
			$payment = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['payments']} WHERE invoice_id=%d LIMIT 1", $invoice_id ) );
			if ( $payment ) { $wpdb->update( $t['payments'], [ 'status' => 'rejected', 'updated_at' => current_time( 'mysql' ) ], [ 'id' => (int) $payment->id ] ); }
			$case = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['cases']} WHERE id=%d", (int) $link->case_id ) );
			if ( $case ) {
				$new_status = (float) $case->paid_amount > 0 ? 'deposit_paid' : 'payment_rejected';
				$wpdb->update( $t['cases'], [ 'status' => $new_status, 'updated_at' => current_time( 'mysql' ) ], [ 'id' => (int) $case->id ] );
				$this->log_case( (int) $case->id, get_current_user_id(), 'payment_financial_rejected', (string) $case->status, $new_status, [ 'invoice_id' => $invoice_id, 'reason' => $reason ] );
			}
		}
	}

	private function user_mobile( int $user_id ): string {
		$user = get_user_by( 'id', $user_id );
		$candidates = [
			get_user_meta( $user_id, 'billing_phone', true ), get_user_meta( $user_id, 'mobile', true ),
			get_user_meta( $user_id, 'phone', true ), $user instanceof WP_User ? $user->user_login : '',
		];
		foreach ( $candidates as $candidate ) {
			$phone = SN_Helpers::normalize_mobile( (string) $candidate );
			if ( SN_Helpers::is_valid_mobile( $phone ) ) { return $phone; }
		}
		return '';
	}

	private function log_case( int $case_id, int $actor_id, string $event, string $old_status, string $new_status, array $details = [] ): void {
		global $wpdb;
		$t = $this->tables();
		$wpdb->insert( $t['logs'], [
			'case_id' => $case_id, 'actor_id' => $actor_id ?: null, 'event_key' => sanitize_key( $event ),
			'old_status' => $old_status ?: null, 'new_status' => $new_status ?: null,
			'details' => $details ? wp_json_encode( $details, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ) : null,
			'created_at' => current_time( 'mysql' ),
		] );
	}

	private function get_case( int $case_id ) {
		global $wpdb;
		$t = $this->tables();
		return $case_id > 0 && $this->table_exists( $t['cases'] ) ? $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['cases']} WHERE id=%d LIMIT 1", $case_id ) ) : null;
	}

	/**
	 * Keep Dot option prices current until the first conversion payment is
	 * issued.  A payment row is created when the converter issues the link,
	 * so that row is the immutable-price boundary (including rejected links).
	 *
	 * The selected option key remains the case's source of truth.  Only prices
	 * are refreshed from the current Dot configuration; once any payment row
	 * exists, the value stored on the case is deliberately left untouched.
	 */
	private function sync_live_case_option_prices( object $case, ?bool $has_payment = null ): object {
		global $wpdb;
		$t = $this->tables();
		$case_id = (int) ( $case->id ?? 0 );
		if ( $case_id < 1 || ! $this->table_exists( $t['cases'] ) || ! $this->table_exists( $t['payments'] ) ) { return $case; }
		if ( null === $has_payment ) {
			$has_payment = (int) $wpdb->get_var( $wpdb->prepare( "SELECT COUNT(*) FROM {$t['payments']} WHERE case_id=%d", $case_id ) ) > 0;
		}
		if ( $has_payment ) { return $case; }

		$configured_prices = [];
		$configured_products = [];
		foreach ( (array) ( $this->config()['options'] ?? [] ) as $configured_option ) {
			$key = sanitize_key( (string) ( $configured_option['key'] ?? '' ) );
			if ( $key !== '' ) {
				$configured_prices[ $key ] = (float) ( $configured_option['price'] ?? 0 );
				$configured_products[ $key ] = absint( $configured_option['product_id'] ?? 0 );
			}
		}
		if ( ! $configured_prices ) { return $case; }

		$options = json_decode( (string) ( $case->options_snapshot_json ?? '' ), true );
		$options_changed = false;
		if ( is_array( $options ) ) {
			foreach ( $options as &$option ) {
				$key = sanitize_key( (string) ( $option['key'] ?? '' ) );
				if ( $key === '' || ! array_key_exists( $key, $configured_prices ) ) { continue; }
				$live_price = $configured_prices[ $key ];
				if ( abs( (float) ( $option['price'] ?? 0 ) - $live_price ) > 0.0001 ) {
					$option['price'] = $live_price;
					$options_changed = true;
				}
				$live_product_id = (int) ( $configured_products[ $key ] ?? 0 );
				if ( (int) ( $option['product_id'] ?? 0 ) !== $live_product_id ) { $option['product_id'] = $live_product_id; $options_changed = true; }
			}
			unset( $option );
		}

		$data = [];
		if ( $options_changed ) {
			$data['options_snapshot_json'] = wp_json_encode( $options, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES );
		}
		$selected_key = sanitize_key( (string) ( $case->selected_option_key ?? '' ) );
		if ( $selected_key !== '' && array_key_exists( $selected_key, $configured_prices ) ) {
			$live_selected_price = $configured_prices[ $selected_key ];
			$live_selected_product_id = (int) ( $configured_products[ $selected_key ] ?? 0 );
			if ( abs( (float) ( $case->selected_option_price ?? 0 ) - $live_selected_price ) > 0.0001 ) {
				$data['selected_option_price'] = $live_selected_price;
				$data['remaining_amount'] = max( 0, $live_selected_price - (float) ( $case->paid_amount ?? 0 ) );
			}
			if ( (int) ( $case->selected_option_product_id ?? 0 ) !== $live_selected_product_id ) { $data['selected_option_product_id'] = $live_selected_product_id ?: null; }
		}
		if ( ! $data ) { return $case; }
		$data['updated_at'] = current_time( 'mysql' );
		if ( false === $wpdb->update( $t['cases'], $data, [ 'id' => $case_id ] ) ) { return $case; }
		foreach ( $data as $column => $value ) { $case->{$column} = $value; }
		return $case;
	}

	/**
	 * Dot conversion state for assessment invoices owned by one seller.
	 *
	 * Results are keyed by the original assessment invoice. Both incomplete and
	 * completed cases are returned so the seller can distinguish "not sold yet"
	 * from a fully-paid subscription without seeing any other seller's records.
	 */
	public function completed_results_for_seller_invoices( array $source_invoice_ids, int $seller_id ): array {
		global $wpdb;
		$t = $this->tables();
		$invoice_ids = array_values( array_unique( array_filter( array_map( 'absint', $source_invoice_ids ) ) ) );
		if ( $seller_id < 1 || empty( $invoice_ids ) || ! $this->table_exists( $t['cases'] ) ) { return []; }

		$placeholders = implode( ',', array_fill( 0, count( $invoice_ids ), '%d' ) );
		$args = array_merge( [ $seller_id ], $invoice_ids );
		$sql = "SELECT id,source_invoice_id,status,selected_option_product_id,selected_option_title,selected_option_price,paid_amount,remaining_amount,completed_at,updated_at
			FROM {$t['cases']}
			WHERE seller_id=%d
				AND source_invoice_id IN ({$placeholders})
			ORDER BY id ASC";
		$rows = $wpdb->get_results( $wpdb->prepare( $sql, ...$args ) ) ?: [];
		$results = [];
		foreach ( $rows as $row ) {
			$paid = max( 0, (float) $row->paid_amount );
			$total = max( 0, (float) $row->selected_option_price );
			$status = sanitize_key( (string) $row->status );
			$has_selected_subscription = (int) $row->selected_option_product_id > 0 || trim( (string) $row->selected_option_title ) !== '';
			// Status and balance are authoritative. Do not hide a migrated/legacy
			// completed case merely because its historical completed_at is empty;
			// however a malformed case with no selected subscription is never a sale.
			$fully_paid = $has_selected_subscription && $status === 'completed' && (float) $row->remaining_amount <= 0.5 && $paid + 0.5 >= $total;
			$closed_without_sale = in_array( $status, [ 'customer_declined', 'archived_unpaid_subscription' ], true );
			$sale_state = $fully_paid ? 'sold' : ( $closed_without_sale ? 'not_sold' : 'in_progress' );
			$results[ (int) $row->source_invoice_id ] = [
				'case_id' => (int) $row->id,
				'option_product_id' => (int) $row->selected_option_product_id,
				'option_title' => (string) ( $row->selected_option_title ?: 'هنوز انتخاب نشده' ),
				'conversion_status' => $status,
				'conversion_status_label' => $this->status_label( $status ),
				'sale_state' => $sale_state,
				'sale_state_label' => $fully_paid ? 'اشتراک فروخته شد' : ( $closed_without_sale ? 'اشتراک فروخته نشد' : 'هنوز فروخته نشده' ),
				'paid_amount' => $paid,
				'paid_amount_fmt' => SN_Helpers::format_price( $paid ),
				'total_amount' => $total,
				'total_amount_fmt' => SN_Helpers::format_price( $total ),
				'completed_at_jalali' => $fully_paid ? $this->jalali_datetime( (string) ( $row->completed_at ?: $row->updated_at ) ) : '',
				'updated_at_jalali' => $this->jalali_datetime( (string) $row->updated_at ),
				'is_fully_paid' => $fully_paid,
				'subscription_sold' => $fully_paid,
			];
		}
		return $results;
	}

	private function valid_link_case( int $case_id, string $token ) {
		$case = $this->get_case( $case_id );
		if ( ! $case || $token === '' || empty( $case->access_token_hash ) ) { return null; }
		$token_hash = hash( 'sha256', $token );
		if ( ! hash_equals( (string) $case->access_token_hash, $token_hash ) ) {
			// Self-heal a migrated case when the compact signed route supplied the
			// deterministic current token but the stored hash is from an older SMS.
			if ( ! hash_equals( $this->persistent_access_token( $case ), $token ) ) { return null; }
			global $wpdb; $t = $this->tables();
			$wpdb->update( $t['cases'], [ 'access_token_hash' => $token_hash, 'access_expires_at' => null, 'updated_at' => current_time( 'mysql' ) ], [ 'id' => $case_id ] );
		}
		return $case;
	}

	private function session_cookie_name( int $case_id ): string {
		return 'sn_dot_session_' . $case_id;
	}

	private function valid_session_case( int $case_id ) {
		$case = $this->get_case( $case_id );
		if ( ! $case || empty( $case->session_token_hash ) || empty( $case->session_expires_at ) ) { return null; }
		$cookie_name = $this->session_cookie_name( $case_id );
		$token = isset( $_COOKIE[ $cookie_name ] ) ? sanitize_text_field( wp_unslash( $_COOKIE[ $cookie_name ] ) ) : '';
		if ( $token === '' || ! hash_equals( (string) $case->session_token_hash, hash( 'sha256', $token ) ) ) { return null; }
		if ( $this->mysql_timestamp( (string) $case->session_expires_at ) < time() ) { return null; }
		return $case;
	}

	private function set_session_cookie( int $case_id, string $token ): void {
		$name = $this->session_cookie_name( $case_id );
		$options = [
			'expires' => time() + self::SESSION_TTL,
			'path' => COOKIEPATH ?: '/',
			'domain' => COOKIE_DOMAIN ?: '',
			'secure' => is_ssl(),
			'httponly' => true,
			'samesite' => 'Lax',
		];
		setcookie( $name, $token, $options );
		$_COOKIE[ $name ] = $token;
	}

	private function customer_page_base(): string {
		$page_id = (int) get_option( 'sn_dot_customer_page_id', 0 );
		return $page_id ? (string) get_permalink( $page_id ) : home_url( '/credit-assessment/' );
	}

	private function marketing_return_url( string $raw = '' ): string {
		$raw = esc_url_raw( $raw );
		$fallback = home_url( '/' );
		return $raw !== '' ? wp_validate_redirect( $raw, $fallback ) : $fallback;
	}

	private function capture_marketing_campaign_context( array $source ): void {
		if ( ! class_exists( 'SN_Campaign_Tracking' ) ) { return; }
		$url = esc_url_raw( wp_unslash( (string) ( $source['sn_campaign_landing_url'] ?? '' ) ) );
		SN_Campaign_Tracking::capture_submission_context( $source, $url );
	}

	/** Keep the UTM source beside the lead so attribution can be audited and retried. */
	private function marketing_campaign_snapshot( array $source ): array {
		if ( class_exists( 'SN_Campaign_Tracking' ) ) {
			return SN_Campaign_Tracking::submission_snapshot(
				$source,
				esc_url_raw( wp_unslash( (string) ( $source['sn_campaign_landing_url'] ?? '' ) ) )
			);
		}
		$snapshot = [ 'sn_campaign_landing_url' => '' ];
		foreach ( [ 'utm_source','utm_medium','utm_campaign','utm_content','utm_term' ] as $key ) {
			$snapshot[ $key ] = sanitize_text_field( wp_unslash( (string) ( $source[ $key ] ?? '' ) ) );
		}
		return $snapshot;
	}

	private function marketing_snapshot_has_utm( array $snapshot ): bool {
		foreach ( [ 'utm_source','utm_medium','utm_campaign','utm_content','utm_term' ] as $key ) {
			if ( trim( (string) ( $snapshot[ $key ] ?? '' ) ) !== '' ) { return true; }
		}
		return false;
	}

	private function link_marketing_submission_campaign( int $submission_id, int $crm_lead_id, string $phone, array $snapshot ): int {
		if ( $submission_id < 1 || ! class_exists( 'SN_Campaign_Tracking' ) ) { return 0; }
		$attribution_id = SN_Campaign_Tracking::capture_marketing_registration( $submission_id, $crm_lead_id, $phone, $snapshot );
		global $wpdb;
		$t = $this->tables();
		$has_utm = $this->marketing_snapshot_has_utm( $snapshot );
		$update = [
			'campaign_attribution_id' => $attribution_id > 0 ? $attribution_id : null,
			'campaign_attribution_status' => $attribution_id > 0 ? 'attributed' : ( $has_utm ? 'pending' : 'untracked' ),
			'campaign_attributed_at' => $attribution_id > 0 ? current_time( 'mysql' ) : null,
			'updated_at' => current_time( 'mysql' ),
		];
		$wpdb->update( $t['marketing_leads'], $this->existing_columns( $t['marketing_leads'], $update ), [ 'id' => $submission_id ] );
		return $attribution_id;
	}

	private function marketing_error_redirect( string $message, string $return_url = '' ): void {
		$target = $this->marketing_return_url( $return_url );
		$target = remove_query_arg( [ 'sn_dot_marketing_error' ], $target );
		wp_safe_redirect( add_query_arg( 'sn_dot_marketing_error', $message, $target ) );
		exit;
	}

	private function normalize_marketing_client_key( $value ): string {
		$value = sanitize_text_field( wp_unslash( (string) $value ) );
		return preg_match( '/^[A-Za-z0-9_-]{16,80}$/', $value ) ? $value : '';
	}

	private function marketing_rate_limit_ok( string $phone, string $form_id = 'default', string $client_key = '' ): bool {
		$ip = sanitize_text_field( (string) ( $_SERVER['REMOTE_ADDR'] ?? '' ) );
		$form_id = sanitize_key( $form_id ) ?: 'default';
		$client_key = $this->normalize_marketing_client_key( $client_key );
		$ip_key = 'sn_dot_mkt_v2_ip_' . md5( $ip ?: 'unknown' );
		$phone_key = 'sn_dot_mkt_v2_phone_' . md5( $form_id . '|' . $phone );
		$pair_key = 'sn_dot_mkt_v3_pair_' . md5( ( $client_key ?: ( $ip ?: 'unknown' ) ) . '|' . $form_id . '|' . $phone );
		$ip_count = (int) get_transient( $ip_key );
		$phone_count = (int) get_transient( $phone_key );
		$pair_count = (int) get_transient( $pair_key );
		if ( $ip_count >= 800 || $phone_count >= 20 || $pair_count >= 8 ) { return false; }
		set_transient( $ip_key, $ip_count + 1, 15 * MINUTE_IN_SECONDS );
		set_transient( $phone_key, $phone_count + 1, HOUR_IN_SECONDS );
		set_transient( $pair_key, $pair_count + 1, 15 * MINUTE_IN_SECONDS );
		return true;
	}

	private function marketing_snapshot_rate_limit_ok( string $phone, string $form_id = 'default', string $client_key = '' ): bool {
		$ip = sanitize_text_field( (string) ( $_SERVER['REMOTE_ADDR'] ?? '' ) );
		$form_id = sanitize_key( $form_id ) ?: 'default';
		$client_key = $this->normalize_marketing_client_key( $client_key );
		$ip_key = 'sn_dot_mkt_snapshot_v2_ip_' . md5( $ip ?: 'unknown' );
		$phone_key = 'sn_dot_mkt_snapshot_v2_phone_' . md5( $form_id . '|' . $phone );
		$pair_key = 'sn_dot_mkt_snapshot_v3_pair_' . md5( ( $client_key ?: ( $ip ?: 'unknown' ) ) . '|' . $form_id . '|' . $phone );
		$ip_count = (int) get_transient( $ip_key );
		$phone_count = (int) get_transient( $phone_key );
		$pair_count = (int) get_transient( $pair_key );
		if ( $ip_count >= 1200 || $phone_count >= 40 || $pair_count >= 20 ) { return false; }
		set_transient( $ip_key, $ip_count + 1, 15 * MINUTE_IN_SECONDS );
		set_transient( $phone_key, $phone_count + 1, HOUR_IN_SECONDS );
		set_transient( $pair_key, $pair_count + 1, 15 * MINUTE_IN_SECONDS );
		return true;
	}

	/**
	 * Keep the legacy CRM lead link when it is safe to do so. The core lead table
	 * has a unique phone key, so an unrelated CRM lead must never be overwritten
	 * merely because the same person submitted a Marketing Dot landing form.
	 */
	private function save_marketing_crm_lead( string $name, string $phone, string $national_id, string $province, string $city, int $supervisor_id, string $form_id ): int {
		global $wpdb;
		$table = $wpdb->prefix . 'sn_leads';
		if ( ! $this->table_exists( $table ) ) { return 0; }
		$form_id = sanitize_key( $form_id ) ?: 'default';
		$existing = $wpdb->get_row( $wpdb->prepare( "SELECT id,marketing_form_id,lead_status FROM {$table} WHERE phone=%s LIMIT 1", $phone ) );
		$data = [ 'phone'=>$phone, 'customer_name'=>$name, 'national_id'=>$national_id, 'marketing_form_id'=>$form_id, 'province'=>$province, 'city'=>$city, 'status'=>'unassigned', 'lead_status'=>'payment_not_done', 'destination_panel'=>'supervisor_leads', 'supervisor_id'=>$supervisor_id, 'note'=>'Marketing Dot Flow: پرداخت نشده' ];
		$formats = [ '%s','%s','%s','%s','%s','%s','%s','%s','%s','%d','%s' ];
		if ( $existing ) {
			if ( sanitize_key( (string) $existing->marketing_form_id ) !== $form_id ) { return 0; }
			if ( sanitize_key( (string) $existing->lead_status ) === 'payment_not_done' ) {
				$wpdb->update( $table, $data, [ 'id'=>(int)$existing->id ], $formats, [ '%d' ] );
			}
			return (int) $existing->id;
		}
		if ( $wpdb->insert( $table, $data, $formats ) ) { return (int) $wpdb->insert_id; }
		// A concurrent request may have won the unique phone insert. Link only if
		// that row belongs to the same marketing form.
		$existing = $wpdb->get_row( $wpdb->prepare( "SELECT id,marketing_form_id FROM {$table} WHERE phone=%s LIMIT 1", $phone ) );
		return $existing && sanitize_key( (string) $existing->marketing_form_id ) === $form_id ? (int) $existing->id : 0;
	}

	private function get_marketing_submission( int $submission_id ) {
		global $wpdb;
		$t = $this->tables();
		return $submission_id > 0 && $this->table_exists( $t['marketing_leads'] )
			? $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['marketing_leads']} WHERE id=%d LIMIT 1", $submission_id ) )
			: null;
	}

	private function owned_marketing_submission( int $submission_id, string $phone, string $form_id, string $event_token ) {
		if ( $submission_id < 1 || $event_token === '' ) { return null; }
		$token_data = get_transient( self::MARKETING_EVENT_TOKEN_PREFIX . hash( 'sha256', $event_token ) );
		if ( ! is_array( $token_data )
			|| (int) ( $token_data['submission_id'] ?? 0 ) !== $submission_id
			|| ! hash_equals( (string) ( $token_data['phone'] ?? '' ), $phone )
			|| ! hash_equals( (string) ( $token_data['form_id'] ?? '' ), $form_id ) ) {
			return null;
		}
		$row = $this->get_marketing_submission( $submission_id );
		return $row && hash_equals( (string) $row->customer_phone, $phone ) && hash_equals( (string) $row->form_id, $form_id ) ? $row : null;
	}

	private function marketing_event_stage( string $event ): int {
		return [
			'continue_clicked' => 10,
			'lead_saved' => 20,
			'invoice_failed' => 25,
			'invoice_created' => 30,
			'invoice_reused' => 30,
			'confirmation_opened' => 40,
			'confirmation_accepted' => 50,
			'otp_requested' => 50,
			'otp_verified' => 60,
			'gateway_requested' => 70,
			'gateway_started' => 80,
			'gateway_cancelled' => 80,
			'gateway_failed' => 80,
			'pending_finance' => 90,
			'payment_rejected' => 90,
			'payment_paid' => 100,
			'validation_gifted' => 100,
		][ $event ] ?? 0;
	}

	private function marketing_event_timestamp_column( string $event ): string {
		return [
			'continue_clicked' => 'application_started_at',
			'lead_saved' => 'lead_saved_at',
			'invoice_created' => 'invoice_created_at',
			'invoice_reused' => 'invoice_created_at',
			'confirmation_opened' => 'confirmation_opened_at',
			'confirmation_accepted' => 'confirmation_accepted_at',
			'otp_requested' => 'otp_requested_at',
			'otp_verified' => 'otp_verified_at',
			'gateway_requested' => 'gateway_requested_at',
			'gateway_started' => 'gateway_entered_at',
			'gateway_cancelled' => 'gateway_returned_at',
			'gateway_failed' => 'gateway_returned_at',
			'pending_finance' => 'gateway_returned_at',
			'payment_rejected' => 'gateway_returned_at',
			'payment_paid' => 'paid_at',
		][ $event ] ?? '';
	}

	/** Append one immutable funnel event and keep the lead's current stage in sync. */
	private function record_marketing_event( int $submission_id, string $event, string $event_status = 'success', string $reason = '', array $details = [], string $event_seed = '' ): bool {
		$event = sanitize_key( $event );
		$allowed = [ 'continue_clicked','lead_saved','invoice_created','invoice_reused','invoice_failed','confirmation_opened','confirmation_accepted','popup_cancelled','otp_requested','otp_verified','otp_failed','otp_cancelled','gateway_requested','gateway_started','gateway_cancelled','gateway_failed','pending_finance','payment_paid','payment_rejected','validation_gifted','rate_limited','form_error' ];
		if ( $submission_id < 1 || ! in_array( $event, $allowed, true ) ) { return false; }
		$row = $this->get_marketing_submission( $submission_id );
		if ( ! $row ) { return false; }
		$event_status = in_array( $event_status, [ 'success','failed','cancelled','info','legacy' ], true ) ? $event_status : 'info';
		$reason = sanitize_text_field( $reason );
		$reason = function_exists( 'mb_substr' ) ? mb_substr( $reason, 0, 191 ) : substr( $reason, 0, 191 );
		$now = current_time( 'mysql' );
		$event_seed = sanitize_text_field( $event_seed );
		if ( $event_seed === '' ) { $event_seed = SN_Helpers::tehran_format( 'YmdHi' ); }
		$event_key = hash( 'sha256', implode( '|', [ $submission_id, $event, $reason, $event_seed ] ) );
		global $wpdb;
		$t = $this->tables();
		if ( ! $this->table_exists( $t['marketing_events'] ) ) { $this->install_schema(); }
		if ( ! $this->table_exists( $t['marketing_events'] ) ) { return false; }
		$details_json = $details ? wp_json_encode( $details, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ) : null;
		$inserted = $wpdb->query( $wpdb->prepare(
			"INSERT IGNORE INTO {$t['marketing_events']} (marketing_lead_id,event_key,event_type,event_status,reason,details,created_at) VALUES (%d,%s,%s,%s,%s,%s,%s)",
			$submission_id,
			$event_key,
			$event,
			$event_status,
			$reason ?: null,
			$details_json,
			$now
		) );
		$stage = $this->marketing_event_stage( $event );
		$current_stage = (int) ( $row->funnel_stage ?? 0 );
		$data = [ 'updated_at' => $now ];
		$stale_browser_milestone = $event === 'confirmation_opened' && $stage < $current_stage;
		if ( ! $stale_browser_milestone ) {
			$data['last_event'] = $event;
			$data['last_event_reason'] = $reason ?: null;
			$data['last_event_at'] = $now;
		}
		$timestamp_column = $this->marketing_event_timestamp_column( $event );
		if ( $timestamp_column !== '' && empty( $row->{$timestamp_column} ) ) { $data[ $timestamp_column ] = $now; }
		$wpdb->update( $t['marketing_leads'], $this->existing_columns( $t['marketing_leads'], $data ), [ 'id' => $submission_id ] );
		if ( $stage > 0 && in_array( 'funnel_stage', $this->columns( $t['marketing_leads'] ), true ) ) {
			$wpdb->query( $wpdb->prepare( "UPDATE {$t['marketing_leads']} SET funnel_stage=GREATEST(funnel_stage,%d) WHERE id=%d", $stage, $submission_id ) );
		}
		if ( $inserted && class_exists( 'SN_Campaign_Tracking' ) && method_exists( 'SN_Campaign_Tracking', 'capture_marketing_funnel_event' ) ) {
			SN_Campaign_Tracking::capture_marketing_funnel_event( $submission_id, $event, $now );
		}
		return false !== $inserted;
	}

	private function update_marketing_crm_event_note( int $crm_lead_id, string $event ): void {
		if ( $crm_lead_id < 1 ) { return; }
		global $wpdb;
		$table = $wpdb->prefix . 'sn_leads';
		if ( ! $this->table_exists( $table ) ) { return; }
		$wpdb->update(
			$table,
			[ 'note' => 'Marketing Dot Flow: ' . $this->marketing_lead_event_label( $event ) ],
			[ 'id' => $crm_lead_id ],
			[ '%s' ],
			[ '%d' ]
		);
	}

	/** Record the best-known result of a Marketing Dot gateway redirect/callback. */
	public function mark_marketing_gateway_result( int $invoice_id, string $result, string $reason = '' ): void {
		$invoice_id = absint( $invoice_id );
		$result = sanitize_key( $result );
		if ( $invoice_id < 1 || ! in_array( $result, [ 'gateway_started', 'gateway_cancelled', 'gateway_failed' ], true ) ) { return; }
		global $wpdb;
		$t = $this->tables();
		if ( ! $this->table_exists( $t['marketing_leads'] ) ) { return; }
		$row = $wpdb->get_row( $wpdb->prepare( "SELECT id,crm_lead_id,status FROM {$t['marketing_leads']} WHERE invoice_id=%d LIMIT 1", $invoice_id ) );
		if ( ! $row || in_array( (string) $row->status, [ 'pending_finance', 'payment_paid', 'payment_rejected', 'validation_gifted' ], true ) ) { return; }
		$now = current_time( 'mysql' );
		$data = [
			'status' => $result,
			'last_event' => $result,
			'last_event_reason' => sanitize_key( $reason ) ?: null,
			'last_event_at' => $now,
			'updated_at' => $now,
		];
		if ( $result === 'gateway_started' ) {
			$data['gateway_entered_at'] = $now;
			$data['gateway_returned_at'] = null;
		} else {
			$data['gateway_returned_at'] = $now;
		}
		if ( false !== $wpdb->update( $t['marketing_leads'], $data, [ 'id' => (int) $row->id ] ) ) {
			$this->record_marketing_event( (int) $row->id, $result, $result === 'gateway_started' ? 'success' : ( $result === 'gateway_cancelled' ? 'cancelled' : 'failed' ), sanitize_key( $reason ), [ 'invoice_id' => $invoice_id ], 'invoice-' . $invoice_id . '-' . $result );
			$this->update_marketing_crm_event_note( (int) $row->crm_lead_id, $result );
		}
	}

	/** Persist every landing submission before payment, including abandoned payments. */
	private function save_marketing_submission( string $name, string $phone, string $national_id, string $province, string $city, int $supervisor_id, array $config, int $submission_id = 0, array $campaign_source = [] ): int {
		global $wpdb;
		$t = $this->tables();
		if ( ! $this->table_exists( $t['marketing_leads'] ) ) { $this->install_schema(); }
		if ( ! $this->table_exists( $t['marketing_leads'] ) ) { return 0; }
		$form_id = sanitize_key( (string) ( $config['_form_id'] ?? 'default' ) ) ?: 'default';
		$reusable_statuses = [ 'payment_not_done', 'popup_cancelled', 'otp_cancelled', 'awaiting_payment', 'gateway_started', 'gateway_cancelled', 'gateway_failed' ];
		$row = $submission_id > 0 ? $this->get_marketing_submission( $submission_id ) : null;
		if ( ! $row || (string) $row->customer_phone !== $phone || (string) $row->form_id !== $form_id || ! in_array( (string) $row->status, $reusable_statuses, true ) ) {
			$row = null;
		}
		$campaign_snapshot = $this->marketing_campaign_snapshot( $campaign_source );
		// No phone/form fallback is intentional. A validated submission_id updates
		// the same browser attempt; an empty/new submission_id always inserts a new
		// Marketing Dot lead, even when that phone submitted earlier.
		$crm_lead_id = $row ? (int) $row->crm_lead_id : 0;
		if ( $crm_lead_id < 1 ) {
			$candidate = $this->save_marketing_crm_lead( $name, $phone, $national_id, $province, $city, $supervisor_id, $form_id );
			$already_linked = $candidate > 0 ? (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$t['marketing_leads']} WHERE crm_lead_id=%d LIMIT 1", $candidate ) ) : 0;
			$crm_lead_id = $already_linked && ( ! $row || $already_linked !== (int) $row->id ) ? 0 : $candidate;
		}
		$data = [
			'crm_lead_id' => $crm_lead_id > 0 ? $crm_lead_id : null,
			'form_id' => $form_id,
			'form_title' => sanitize_text_field( (string) ( $config['_form_title'] ?? ( $form_id === 'default' ? 'فرم پیش‌فرض' : $form_id ) ) ),
			'product_id' => absint( $config['product_id'] ?? 0 ),
			'supervisor_id' => $supervisor_id,
			'customer_name' => $name,
			'customer_phone' => $phone,
			'national_id' => $national_id,
			'province' => $province,
			'city' => $city,
			'status' => 'payment_not_done',
			'last_event' => 'lead_saved',
			'last_event_reason' => null,
			'last_event_at' => current_time( 'mysql' ),
			'gateway_entered_at' => null,
			'gateway_returned_at' => null,
			'updated_at' => current_time( 'mysql' ),
		];
		$landing_url = esc_url_raw( (string) ( $campaign_snapshot['sn_campaign_landing_url'] ?? '' ) );
		if ( ! $row || $landing_url !== '' ) { $data['campaign_landing_url'] = $landing_url !== '' ? $landing_url : null; }
		foreach ( [ 'utm_source','utm_medium','utm_campaign','utm_content','utm_term' ] as $utm_key ) {
			$value = sanitize_text_field( (string) ( $campaign_snapshot[ $utm_key ] ?? '' ) );
			if ( ! $row || $value !== '' ) { $data[ $utm_key ] = $value !== '' ? $value : null; }
		}
		if ( $row ) {
			if ( false === $wpdb->update( $t['marketing_leads'], $data, [ 'id' => (int) $row->id ] ) ) { return 0; }
			$this->update_marketing_crm_event_note( $crm_lead_id, 'lead_saved' );
			$this->link_marketing_submission_campaign( (int) $row->id, $crm_lead_id, $phone, $campaign_snapshot );
			$this->record_marketing_event( (int) $row->id, 'continue_clicked', 'success', '', [], 'submission-' . (int) $row->id );
			$this->record_marketing_event( (int) $row->id, 'lead_saved', 'success', '', [], 'submission-' . (int) $row->id );
			return (int) $row->id;
		}
		$data['created_at'] = current_time( 'mysql' );
		if ( ! $wpdb->insert( $t['marketing_leads'], $data ) ) { return 0; }
		$new_submission_id = (int) $wpdb->insert_id;
		$this->update_marketing_crm_event_note( $crm_lead_id, 'lead_saved' );
		$this->link_marketing_submission_campaign( $new_submission_id, $crm_lead_id, $phone, $campaign_snapshot );
		$this->record_marketing_event( $new_submission_id, 'continue_clicked', 'success', '', [], 'submission-' . $new_submission_id );
		$this->record_marketing_event( $new_submission_id, 'lead_saved', 'success', '', [], 'submission-' . $new_submission_id );
		return $new_submission_id;
	}

	/**
	 * Create the source pre-invoice as soon as the landing data is saved. This is
	 * what makes both popup abandonment and gateway abandonment visible in the
	 * supervisor invoice tab. Repeated attempts reuse the same pre-invoice.
	 */
	private function ensure_marketing_source_invoice( object $submission, array $config ) {
		global $wpdb;
		$t = $this->tables();
		$form_id = sanitize_key( (string) ( $config['_form_id'] ?? $submission->form_id ?? 'default' ) ) ?: 'default';
		$product_id = absint( $config['product_id'] ?? 0 );
		$supervisor_id = (int) ( $submission->supervisor_id ?? 0 );
		if ( $product_id < 1 || $supervisor_id < 1 || (string) ( $submission->form_id ?? '' ) !== $form_id ) {
			return new WP_Error( 'sn_dot_marketing_invoice_route', 'اتصال فرم، محصول یا سرپرست برای ساخت پیش‌فاکتور معتبر نیست.' );
		}

		$invoice_id = absint( $submission->invoice_id ?? 0 );
		if ( $invoice_id > 0 ) {
			$invoice = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$wpdb->prefix}sn_invoices WHERE id=%d LIMIT 1", $invoice_id ) );
			$link = $this->table_exists( $t['links'] ) ? $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['links']} WHERE invoice_id=%d AND flow_kind='assessment_source' LIMIT 1", $invoice_id ) ) : null;
			if ( ! $invoice || ! $link || (int) $invoice->product_id !== $product_id || (int) $link->supervisor_id !== $supervisor_id || sanitize_key( (string) ( $link->marketing_form_id ?? 'default' ) ) !== $form_id ) {
				return new WP_Error( 'sn_dot_marketing_invoice_mismatch', 'پیش‌فاکتور قبلی با فرم، محصول یا سرپرست فعلی مطابقت ندارد.' );
			}
			return [
				'invoice_id' => $invoice_id,
				'invoice_code' => (string) $invoice->invoice_code,
				'amount' => (float) ( ( $invoice->final_total ?? 0 ) ?: ( $invoice->product_price ?? 0 ) ),
				'seller_id' => (int) $invoice->seller_id,
				'reused' => true,
			];
		}

		$seller_id = $this->marketing_seller_for_supervisor( $supervisor_id );
		if ( $seller_id < 1 ) { return new WP_Error( 'sn_dot_marketing_seller', 'فروشنده سیستمی مارکتینگ قابل ایجاد نیست.' ); }
		if ( ! function_exists( 'sn_bootstrap_plugin_instance' ) ) { return new WP_Error( 'sn_dot_marketing_core', 'هسته شبکه فروش در دسترس نیست.' ); }
		$plugin = sn_bootstrap_plugin_instance( false );
		if ( ! method_exists( $plugin, 'create_dot_marketing_source_invoice' ) ) { return new WP_Error( 'sn_dot_marketing_invoice_method', 'مسیر صدور پیش‌فاکتور مارکتینگ در دسترس نیست.' ); }
		$created = $plugin->create_dot_marketing_source_invoice( [
			'customer_name' => (string) $submission->customer_name,
			'customer_phone' => (string) $submission->customer_phone,
			'province' => (string) $submission->province,
			'city' => (string) $submission->city,
			'national_id' => (string) $submission->national_id,
			'lead_id' => (int) $submission->crm_lead_id,
			'marketing_form_id' => $form_id,
		], $product_id, $seller_id, $supervisor_id );
		if ( empty( $created['success'] ) ) { return new WP_Error( 'sn_dot_marketing_invoice_create', (string) ( $created['message'] ?? 'ساخت پیش‌فاکتور انجام نشد.' ) ); }
		$data = (array) ( $created['data'] ?? [] );
		$invoice_id = absint( $data['invoice_id'] ?? 0 );
		$invoice_code = sanitize_text_field( (string) ( $data['invoice_code'] ?? '' ) );
		$amount = (float) ( $data['amount'] ?? 0 );
		if ( $invoice_id < 1 || $invoice_code === '' || $amount <= 0 ) { return new WP_Error( 'sn_dot_marketing_invoice_incomplete', 'اطلاعات پیش‌فاکتور ساخته‌شده ناقص است.' ); }
		$invoice_created_at = current_time( 'mysql' );
		$updated = $wpdb->update( $t['marketing_leads'], [
			'invoice_id' => $invoice_id,
			'seller_id' => $seller_id,
			'status' => 'awaiting_payment',
			'last_event' => 'invoice_created',
			'last_event_reason' => null,
			'last_event_at' => $invoice_created_at,
			'gateway_entered_at' => null,
			'gateway_returned_at' => null,
			'updated_at' => $invoice_created_at,
		], [ 'id' => (int) $submission->id ] );
		if ( false === $updated ) { return new WP_Error( 'sn_dot_marketing_invoice_link', 'پیش‌فاکتور ساخته شد اما اتصال آن به ثبت فرم انجام نشد.' ); }
		return [
			'invoice_id' => $invoice_id,
			'invoice_code' => $invoice_code,
			'amount' => $amount,
			'seller_id' => $seller_id,
			'reused' => false,
		];
	}

	public function handle_marketing_save_lead(): void {
		check_ajax_referer( 'sn_dot_marketing_start_payment', 'nonce' );
		$name = trim( sanitize_text_field( wp_unslash( $_POST['customer_name'] ?? '' ) ) );
		$phone = SN_Helpers::normalize_mobile( sanitize_text_field( wp_unslash( $_POST['customer_phone'] ?? '' ) ) );
		$national_id = preg_replace( '/\\D+/', '', (string) SN_Helpers::to_english_nums( wp_unslash( $_POST['national_id'] ?? '' ) ) );
		$province = trim( sanitize_text_field( wp_unslash( $_POST['province'] ?? '' ) ) );
		$city = trim( sanitize_text_field( wp_unslash( $_POST['city'] ?? '' ) ) );
		if ( $name === '' || ! SN_Helpers::is_valid_mobile( $phone ) || strlen( $national_id ) !== 10 || $province === '' || $city === '' ) {
			wp_send_json_error( [ 'message' => 'اطلاعات فرم را کامل وارد کنید.' ], 400 );
		}
		$form_id = sanitize_key( wp_unslash( $_POST['sn_dot_marketing_form_id'] ?? 'default' ) ) ?: 'default';
		$client_key = $this->normalize_marketing_client_key( $_POST['sn_campaign_client_key'] ?? '' );
		$form_config = $this->marketing_config_for_form( $form_id );
		if ( empty( $form_config['_form_exists'] ) || empty( $form_config['enabled'] ) || empty( $this->config()['enabled'] ) || absint( $form_config['product_id'] ?? 0 ) < 1 || empty( $form_config['supervisor_ids'] ) ) {
			wp_send_json_error( [ 'message' => 'تنظیمات این فرم کامل یا فعال نیست.' ], 400 );
		}
		$this->capture_marketing_campaign_context( $_POST );
		if ( ! $this->marketing_snapshot_rate_limit_ok( $phone, $form_id, $client_key ) ) {
			wp_send_json_error( [ 'message' => 'تعداد ثبت درخواست بیش از حد مجاز است. کمی بعد دوباره تلاش کنید.' ], 429 );
		}
		$submission_id = absint( $_POST['sn_dot_marketing_submission_id'] ?? 0 );
		$event_token = sanitize_text_field( wp_unslash( $_POST['event_token'] ?? '' ) );
		$submission = $submission_id > 0 ? $this->owned_marketing_submission( $submission_id, $phone, $form_id, $event_token ) : null;
		if ( ! $submission ) { $submission_id = 0; }
		if ( $submission && (string) $submission->customer_phone === $phone && (string) $submission->form_id === $form_id && ! empty( $submission->invoice_id ) && $this->is_gifted_assessment_invoice( (int) $submission->invoice_id ) ) {
			wp_send_json_error( [ 'message' => 'هزینه این درخواست قبلاً از طرف شرکت پرداخت شده است؛ لطفاً از لینک انتخابی که برایتان پیامک شده استفاده کنید.' ], 409 );
		}
		$allowed_supervisors = array_map( 'intval', (array) $form_config['supervisor_ids'] );
		$supervisor_id = $submission && (string) $submission->customer_phone === $phone && (string) $submission->form_id === $form_id && in_array( (int) $submission->supervisor_id, $allowed_supervisors, true )
			? (int) $submission->supervisor_id
			: $this->next_marketing_supervisor( $form_config, $form_id );
		if ( $supervisor_id < 1 ) { wp_send_json_error( [ 'message' => 'سرپرست فعالی برای این فرم تنظیم نشده است.' ], 400 ); }
		$submission_id = $this->save_marketing_submission( $name, $phone, $national_id, $province, $city, $supervisor_id, $form_config, $submission_id, $_POST );
		if ( $submission_id < 1 ) { wp_send_json_error( [ 'message' => 'ذخیره اطلاعات انجام نشد.' ], 500 ); }
		$saved = $this->get_marketing_submission( $submission_id );
		if ( ! $saved ) { wp_send_json_error( [ 'message' => 'اطلاعات ثبت‌شده قابل بازیابی نیست.' ], 500 ); }
		$invoice_result = $this->ensure_marketing_source_invoice( $saved, $form_config );
		if ( is_wp_error( $invoice_result ) ) {
			$this->record_marketing_event( $submission_id, 'invoice_failed', 'failed', sanitize_key( $invoice_result->get_error_code() ) );
			wp_send_json_error( [ 'message' => $invoice_result->get_error_message() ], 500 );
		}
		$this->record_marketing_event( $submission_id, ! empty( $invoice_result['reused'] ) ? 'invoice_reused' : 'invoice_created', 'success', '', [ 'invoice_id' => (int) $invoice_result['invoice_id'] ], 'invoice-' . (int) $invoice_result['invoice_id'] );
		$saved = $this->get_marketing_submission( $submission_id );
		$event_token = wp_generate_password( 48, false, false );
		set_transient( self::MARKETING_EVENT_TOKEN_PREFIX . hash( 'sha256', $event_token ), [
			'submission_id' => $submission_id,
			'phone' => $phone,
			'form_id' => $form_id,
		], 2 * HOUR_IN_SECONDS );
		wp_send_json_success( [
			'submission_id' => $submission_id,
			'lead_id' => $saved ? (int) $saved->crm_lead_id : 0,
			'invoice_id' => (int) $invoice_result['invoice_id'],
			'invoice_code' => (string) $invoice_result['invoice_code'],
			'campaign_attributed' => $saved && (int) ( $saved->campaign_attribution_id ?? 0 ) > 0,
			'campaign_attribution_id' => $saved ? (int) ( $saved->campaign_attribution_id ?? 0 ) : 0,
			'event_token' => $event_token,
		] );
	}

	/** Persist a deliberate exit from the pre-gateway confirmation/OTP popup. */
	public function handle_marketing_record_exit(): void {
		check_ajax_referer( 'sn_dot_marketing_start_payment', 'nonce' );
		$submission_id = absint( $_POST['sn_dot_marketing_submission_id'] ?? 0 );
		$form_id = sanitize_key( wp_unslash( $_POST['sn_dot_marketing_form_id'] ?? 'default' ) ) ?: 'default';
		$phone = SN_Helpers::normalize_mobile( sanitize_text_field( wp_unslash( $_POST['customer_phone'] ?? '' ) ) );
		$event = sanitize_key( wp_unslash( $_POST['event'] ?? '' ) );
		$reason = sanitize_key( wp_unslash( $_POST['reason'] ?? '' ) );
		$event_token = sanitize_text_field( wp_unslash( $_POST['event_token'] ?? '' ) );
		if ( $submission_id < 1 || ! SN_Helpers::is_valid_mobile( $phone ) || ! in_array( $event, [ 'popup_cancelled', 'otp_cancelled' ], true ) ) {
			wp_send_json_error( [ 'message' => 'رویداد خروج معتبر نیست.' ], 400 );
		}
		$row = $this->owned_marketing_submission( $submission_id, $phone, $form_id, $event_token );
		if ( ! $row ) {
			wp_send_json_error( [ 'message' => 'مجوز ثبت رویداد خروج معتبر نیست.' ], 403 );
		}
		if ( ! in_array( (string) $row->status, [ 'payment_not_done', 'awaiting_payment', 'popup_cancelled', 'otp_cancelled' ], true ) ) {
			wp_send_json_success( [ 'recorded' => false ] );
		}
		$reason = in_array( $reason, [ 'cancel_button', 'close_button', 'backdrop', 'escape', 'page_exit' ], true ) ? $reason : 'close_button';
		$now = current_time( 'mysql' );
		global $wpdb;
		$t = $this->tables();
		$updated = $wpdb->update( $t['marketing_leads'], [
			'status' => $event,
			'updated_at' => $now,
		], [ 'id' => $submission_id ] );
		if ( false === $updated ) { wp_send_json_error( [ 'message' => 'ثبت رویداد خروج انجام نشد.' ], 500 ); }
		$this->record_marketing_event( $submission_id, $event, 'cancelled', $reason );
		$this->update_marketing_crm_event_note( (int) $row->crm_lead_id, $event );
		wp_send_json_success( [ 'recorded' => true ] );
	}

	/** Record non-terminal browser funnel milestones after the lead exists. */
	public function handle_marketing_record_event(): void {
		check_ajax_referer( 'sn_dot_marketing_start_payment', 'nonce' );
		$submission_id = absint( $_POST['sn_dot_marketing_submission_id'] ?? 0 );
		$form_id = sanitize_key( wp_unslash( $_POST['sn_dot_marketing_form_id'] ?? 'default' ) ) ?: 'default';
		$phone = SN_Helpers::normalize_mobile( sanitize_text_field( wp_unslash( $_POST['customer_phone'] ?? '' ) ) );
		$event = sanitize_key( wp_unslash( $_POST['event'] ?? '' ) );
		$reason = sanitize_key( wp_unslash( $_POST['reason'] ?? '' ) );
		$event_token = sanitize_text_field( wp_unslash( $_POST['event_token'] ?? '' ) );
		if ( ! in_array( $event, [ 'confirmation_opened','confirmation_accepted' ], true ) || ! SN_Helpers::is_valid_mobile( $phone ) ) {
			wp_send_json_error( [ 'message' => 'رویداد مرحله فرم معتبر نیست.' ], 400 );
		}
		$row = $this->owned_marketing_submission( $submission_id, $phone, $form_id, $event_token );
		if ( ! $row ) { wp_send_json_error( [ 'message' => 'مجوز ثبت مرحله فرم معتبر نیست.' ], 403 ); }
		$recorded = $this->record_marketing_event( $submission_id, $event, 'success', $reason );
		wp_send_json_success( [ 'recorded' => $recorded ] );
	}

	private function marketing_otp_rate_limit( string $phone, string $client_key = '' ) {
		$ip = sanitize_text_field( (string) ( $_SERVER['REMOTE_ADDR'] ?? '' ) );
		$client_key = $this->normalize_marketing_client_key( $client_key );
		$cooldown_key = 'sn_dot_mkt_otp_cd_' . md5( $phone );
		if ( get_transient( $cooldown_key ) ) {
			return new WP_Error( 'sn_dot_marketing_otp_cooldown', 'برای ارسال مجدد کد، کمی صبر کنید.' );
		}
		$ip_key = 'sn_dot_mkt_otp_ip_' . md5( $ip ?: 'unknown' );
		$phone_key = 'sn_dot_mkt_otp_phone_' . md5( $phone );
		$client_rate_key = 'sn_dot_mkt_otp_client_' . md5( $client_key ?: ( $ip ?: 'unknown' ) );
		$ip_count = (int) get_transient( $ip_key );
		$phone_count = (int) get_transient( $phone_key );
		$client_count = (int) get_transient( $client_rate_key );
		if ( $ip_count >= 300 || $phone_count >= 6 || $client_count >= 20 ) {
			return new WP_Error( 'sn_dot_marketing_otp_limit', 'تعداد درخواست کد بیش از حد مجاز است. کمی بعد دوباره تلاش کنید.' );
		}
		set_transient( $cooldown_key, '1', MINUTE_IN_SECONDS );
		set_transient( $ip_key, $ip_count + 1, HOUR_IN_SECONDS );
		set_transient( $phone_key, $phone_count + 1, HOUR_IN_SECONDS );
		set_transient( $client_rate_key, $client_count + 1, HOUR_IN_SECONDS );
		return true;
	}

	private function marketing_otp_key( string $challenge ): string {
		return self::MARKETING_OTP_PREFIX . hash( 'sha256', $challenge );
	}

	private function marketing_otp_proof_key( string $proof ): string {
		return self::MARKETING_OTP_PROOF_PREFIX . hash( 'sha256', $proof );
	}

	private function consume_marketing_otp_proof( string $proof, string $phone, string $form_id ): bool {
		$proof = sanitize_text_field( $proof );
		if ( $proof === '' ) { return false; }
		$key = $this->marketing_otp_proof_key( $proof );
		$data = get_transient( $key );
		if ( ! is_array( $data ) ) { return false; }
		$stored_phone = SN_Helpers::normalize_mobile( (string) ( $data['phone'] ?? '' ) );
		if ( $stored_phone === '' || ! hash_equals( $stored_phone, $phone ) ) { return false; }
		$stored_form_id = sanitize_key( (string) ( $data['form_id'] ?? 'default' ) ) ?: 'default';
		$form_id = sanitize_key( $form_id ) ?: 'default';
		if ( ! hash_equals( $stored_form_id, $form_id ) ) { return false; }
		delete_transient( $key );
		return true;
	}

	public function handle_marketing_send_otp(): void {
		check_ajax_referer( 'sn_dot_marketing_start_payment', 'nonce' );
		nocache_headers();
		$form_id = sanitize_key( wp_unslash( $_POST['sn_dot_marketing_form_id'] ?? 'default' ) ) ?: 'default';
		$config = $this->marketing_config_for_form( $form_id );
		if ( empty( $config['_form_exists'] ) || empty( $config['enabled'] ) || empty( $config['otp_enabled'] ) || empty( $this->config()['enabled'] ) ) {
			wp_send_json_error( [ 'message' => 'تأیید پیامکی این فرم فعال نیست.' ], 400 );
		}
		$phone = SN_Helpers::normalize_mobile( sanitize_text_field( wp_unslash( $_POST['phone'] ?? '' ) ) );
		if ( ! SN_Helpers::is_valid_mobile( $phone ) ) {
			wp_send_json_error( [ 'message' => 'شماره موبایل معتبر نیست.' ], 400 );
		}
		$client_key = $this->normalize_marketing_client_key( $_POST['sn_campaign_client_key'] ?? '' );
		$submission_id = absint( $_POST['sn_dot_marketing_submission_id'] ?? 0 );
		$event_token = sanitize_text_field( wp_unslash( $_POST['event_token'] ?? '' ) );
		$event_submission = $this->owned_marketing_submission( $submission_id, $phone, $form_id, $event_token );
		$limited = $this->marketing_otp_rate_limit( $phone, $client_key );
		if ( is_wp_error( $limited ) ) {
			if ( $event_submission ) { $this->record_marketing_event( $submission_id, 'otp_failed', 'failed', 'rate_limited' ); }
			wp_send_json_error( [ 'message' => $limited->get_error_message() ], 429 );
		}
		try {
			$code = (string) random_int( 100000, 999999 );
		} catch ( Throwable $e ) {
			$code = (string) wp_rand( 100000, 999999 );
		}
		$challenge = wp_generate_password( 40, false, false );
		$payload = [
			'phone' => $phone,
			'form_id' => $form_id,
			'code_hash' => wp_hash_password( $code ),
			'attempts' => 0,
			'created_at' => time(),
		];
		set_transient( $this->marketing_otp_key( $challenge ), $payload, 5 * MINUTE_IN_SECONDS );
		$message = "کد تأیید شماره موبایل: {$code}\nاین کد تا ۵ دقیقه معتبر است\nدات فلو";
		$sms = new SN_SMS();
		if ( ! $sms->send( $phone, $message ) ) {
			delete_transient( $this->marketing_otp_key( $challenge ) );
			if ( $event_submission ) { $this->record_marketing_event( $submission_id, 'otp_failed', 'failed', 'sms_failed' ); }
			wp_send_json_error( [ 'message' => 'ارسال کد تأیید انجام نشد. تنظیمات سرویس پیامک را بررسی کنید.' ], 502 );
		}
		if ( class_exists( 'SN_Notification_Service' ) ) { SN_Notification_Service::instance()->mirror_otp( 'dot.marketing_otp', $phone, $code, [ 'entity_type'=>'dot_marketing', 'entity_id'=>$submission_id, 'dedupe_key'=>'dot-marketing-otp-'.hash('sha256',$challenge), 'source'=>'dot_marketing_otp' ] ); }
		if ( $event_submission ) { $this->record_marketing_event( $submission_id, 'otp_requested', 'success', '', [], 'challenge-' . hash( 'sha256', $challenge ) ); }
		$masked = substr( $phone, 0, 4 ) . '***' . substr( $phone, -4 );
		wp_send_json_success( [ 'challenge' => $challenge, 'phone' => $masked, 'expires_in' => 300 ] );
	}

	public function handle_marketing_verify_otp(): void {
		check_ajax_referer( 'sn_dot_marketing_start_payment', 'nonce' );
		nocache_headers();
		$form_id = sanitize_key( wp_unslash( $_POST['sn_dot_marketing_form_id'] ?? 'default' ) ) ?: 'default';
		$config = $this->marketing_config_for_form( $form_id );
		if ( empty( $config['_form_exists'] ) || empty( $config['enabled'] ) || empty( $config['otp_enabled'] ) || empty( $this->config()['enabled'] ) ) {
			wp_send_json_error( [ 'message' => 'تأیید پیامکی این فرم فعال نیست.' ], 400 );
		}
		$phone = SN_Helpers::normalize_mobile( sanitize_text_field( wp_unslash( $_POST['phone'] ?? '' ) ) );
		$submission_id = absint( $_POST['sn_dot_marketing_submission_id'] ?? 0 );
		$event_token = sanitize_text_field( wp_unslash( $_POST['event_token'] ?? '' ) );
		$event_submission = $this->owned_marketing_submission( $submission_id, $phone, $form_id, $event_token );
		$challenge = sanitize_text_field( wp_unslash( $_POST['challenge'] ?? '' ) );
		$code = preg_replace( '/\D+/', '', SN_Helpers::to_english_nums( sanitize_text_field( wp_unslash( $_POST['code'] ?? '' ) ) ) );
		if ( ! SN_Helpers::is_valid_mobile( $phone ) || strlen( $challenge ) < 20 || ! preg_match( '/^\d{6}$/', $code ) ) {
			wp_send_json_error( [ 'message' => 'اطلاعات کد تأیید معتبر نیست.' ], 400 );
		}
		$key = $this->marketing_otp_key( $challenge );
		$data = get_transient( $key );
		if ( ! is_array( $data ) ) {
			if ( $event_submission ) { $this->record_marketing_event( $submission_id, 'otp_failed', 'failed', 'expired' ); }
			wp_send_json_error( [ 'message' => 'کد منقضی شده است. کد جدید دریافت کنید.' ], 410 );
		}
		$stored_phone = SN_Helpers::normalize_mobile( (string) ( $data['phone'] ?? '' ) );
		if ( $stored_phone === '' || ! hash_equals( $stored_phone, $phone ) ) {
			delete_transient( $key );
			if ( $event_submission ) { $this->record_marketing_event( $submission_id, 'otp_failed', 'failed', 'phone_mismatch' ); }
			wp_send_json_error( [ 'message' => 'شماره موبایل با درخواست کد مطابقت ندارد.' ], 400 );
		}
		$stored_form_id = sanitize_key( (string) ( $data['form_id'] ?? 'default' ) ) ?: 'default';
		if ( ! hash_equals( $stored_form_id, $form_id ) ) {
			delete_transient( $key );
			if ( $event_submission ) { $this->record_marketing_event( $submission_id, 'otp_failed', 'failed', 'form_mismatch' ); }
			wp_send_json_error( [ 'message' => 'کد تأیید متعلق به این فرم نیست.' ], 400 );
		}
		$attempts = (int) ( $data['attempts'] ?? 0 );
		if ( $attempts >= 5 ) {
			delete_transient( $key );
			if ( $event_submission ) { $this->record_marketing_event( $submission_id, 'otp_failed', 'failed', 'attempt_limit' ); }
			wp_send_json_error( [ 'message' => 'تعداد تلاش‌ها بیش از حد مجاز بود. کد جدید دریافت کنید.' ], 429 );
		}
		if ( empty( $data['code_hash'] ) || ! wp_check_password( $code, (string) $data['code_hash'] ) ) {
			$data['attempts'] = $attempts + 1;
			$remaining_ttl = max( 1, ( 5 * MINUTE_IN_SECONDS ) - max( 0, time() - (int) ( $data['created_at'] ?? time() ) ) );
			set_transient( $key, $data, $remaining_ttl );
			if ( $event_submission ) { $this->record_marketing_event( $submission_id, 'otp_failed', 'failed', 'invalid_code', [ 'attempt' => $attempts + 1 ], 'attempt-' . ( $attempts + 1 ) ); }
			wp_send_json_error( [ 'message' => 'کد واردشده صحیح نیست.' ], 400 );
		}
		delete_transient( $key );
		$proof = wp_generate_password( 48, false, false );
		set_transient( $this->marketing_otp_proof_key( $proof ), [ 'phone' => $phone, 'form_id' => $form_id, 'verified_at' => time() ], 10 * MINUTE_IN_SECONDS );
		if ( $event_submission ) { $this->record_marketing_event( $submission_id, 'otp_verified', 'success', '', [], 'challenge-' . hash( 'sha256', $challenge ) ); }
		wp_send_json_success( [ 'proof' => $proof ] );
	}

	private function remember_marketing_return_url( int $invoice_id, string $return_url ): void {
		if ( $invoice_id < 1 ) { return; }
		set_transient( self::MARKETING_RETURN_PREFIX . $invoice_id, $this->marketing_return_url( $return_url ), 2 * DAY_IN_SECONDS );
	}

	/** Return a landing URL only for Marketing Dot Flow source invoices. */
	public function marketing_gateway_return_url( int $invoice_id ): string {
		$invoice_id = absint( $invoice_id );
		if ( $invoice_id < 1 ) { return ''; }
		global $wpdb;
		$seller_id = (int) $wpdb->get_var( $wpdb->prepare( "SELECT seller_id FROM {$wpdb->prefix}sn_invoices WHERE id=%d", $invoice_id ) );
		if ( ! $this->is_marketing_seller_id( $seller_id ) ) { return ''; }
		$links = $this->tables()['links'];
		if ( ! $this->table_exists( $links ) || ! (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$links} WHERE invoice_id=%d AND flow_kind='assessment_source' LIMIT 1", $invoice_id ) ) ) { return ''; }
		$stored = get_transient( self::MARKETING_RETURN_PREFIX . $invoice_id );
		return $this->marketing_return_url( is_string( $stored ) ? $stored : home_url( '/' ) );
	}

	public function render_marketing_form( array $atts = [] ): string {
		$form_id = sanitize_key( $atts['id'] ?? $atts['form'] ?? 'default' ) ?: 'default';
		$config = $this->marketing_config_for_form( $form_id );
		$main = $this->config();
		if ( empty( $config['_form_exists'] ) || empty( $config['enabled'] ) || empty( $main['enabled'] ) ) {
			return '<div class="sn-dot-marketing-unavailable" dir="rtl">فرم درخواست در حال حاضر فعال نیست.</div>';
		}
		$product_id = absint( $config['product_id'] ?? 0 );
		$product = function_exists( 'wc_get_product' ) ? wc_get_product( $product_id ) : null;
		if ( ! $product || get_post_status( $product_id ) !== 'publish' || (string) get_post_meta( $product_id, '_sn_enabled', true ) !== '1' || (float) $product->get_price() <= 0 ) {
			return '<div class="sn-dot-marketing-unavailable" dir="rtl">این فرم هنوز به‌درستی تنظیم نشده است.</div>';
		}
		$marketing_amount = (float) $product->get_price();
		$css = SN_PLUGIN_DIR . 'assets/css/dot-marketing.css';
		$js = SN_PLUGIN_DIR . 'assets/js/public-dot-marketing.js';
		wp_enqueue_style( 'sn-dot-marketing', SN_PLUGIN_URL . 'assets/css/dot-marketing.css', [], SN_VERSION . '-' . ( file_exists( $css ) ? filemtime( $css ) : '0' ) );
		wp_enqueue_script( 'sn-dot-marketing', SN_PLUGIN_URL . 'assets/js/public-dot-marketing.js', [], SN_VERSION . '-' . ( file_exists( $js ) ? filemtime( $js ) : '0' ), true );

		$provinces = class_exists( 'SN_Helpers' ) && method_exists( 'SN_Helpers', 'get_provinces' ) ? SN_Helpers::get_provinces() : [];
		$return_url = get_permalink( get_queried_object_id() );
		if ( ! $return_url ) { $return_url = home_url( '/' ); }
		$landing_utm = [];
		foreach ( [ 'utm_source','utm_medium','utm_campaign','utm_content','utm_term' ] as $utm_key ) {
			$landing_utm[ $utm_key ] = sanitize_text_field( wp_unslash( (string) ( $_GET[ $utm_key ] ?? '' ) ) );
		}
		$landing_url = add_query_arg( array_filter( $landing_utm, static function ( $value ) { return $value !== ''; } ), $return_url );
		$error = sanitize_text_field( wp_unslash( $_GET['sn_dot_marketing_error'] ?? '' ) );
		$pay_result = sanitize_key( wp_unslash( $_GET['pay_result'] ?? '' ) );
		ob_start();
		?>
		<div class="sn-dot-marketing-wrap dot-credit-form-wrap" dir="rtl">
			<form class="sn-dot-marketing-form dot-credit-form" method="post" action="<?php echo esc_url( admin_url( 'admin-post.php' ) ); ?>" data-otp-enabled="<?php echo ! empty( $config['otp_enabled'] ) ? '1' : '0'; ?>" data-ajax-url="<?php echo esc_url( admin_url( 'admin-ajax.php' ) ); ?>">
				<input type="hidden" name="action" value="sn_dot_marketing_start_payment">
				<input type="hidden" name="sn_dot_marketing_return" value="<?php echo esc_attr( $return_url ); ?>">
				<input type="hidden" name="sn_dot_marketing_form_id" value="<?php echo esc_attr( $form_id ); ?>">
				<input type="hidden" name="sn_dot_marketing_submission_id" value="">
				<input type="hidden" name="sn_dot_marketing_event_token" value="">
				<input type="hidden" name="sn_dot_marketing_otp_proof" value="">
				<?php foreach ( $landing_utm as $utm_key => $utm_value ) : ?><input type="hidden" name="<?php echo esc_attr( $utm_key ); ?>" value="<?php echo esc_attr( $utm_value ); ?>"><?php endforeach; ?>
				<input type="hidden" name="sn_campaign_landing_url" value="<?php echo esc_url( $landing_url ); ?>">
				<input type="hidden" name="sn_campaign_client_key" value="">
				<?php wp_nonce_field( 'sn_dot_marketing_start_payment', 'sn_dot_marketing_nonce' ); ?>
				<div class="sn-dot-marketing-hp" aria-hidden="true"><label>وب‌سایت<input type="text" name="sn_dot_marketing_website" tabindex="-1" autocomplete="off"></label></div>
				<div class="dot-credit-header">
					<span class="dot-credit-eyebrow">درخواست آنلاین</span>
					<h3>درخواست خود را شروع کنید</h3>
					<p>اطلاعات شما برای بررسی درخواست دریافت می‌شود.</p>
				</div>
				<?php if ( $error !== '' ) : ?><div class="sn-dot-marketing-alert dot-credit-alert"><?php echo esc_html( $error ); ?></div><?php endif; ?>
			<?php if ( $pay_result === 'failed' ) : ?><div class="sn-dot-marketing-alert dot-credit-alert">پرداخت انجام نشد یا توسط شما لغو شد. می‌توانید دوباره اطلاعات را بررسی و پرداخت را تکرار کنید.</div><?php elseif ( $pay_result === 'pending_finance' ) : ?><div class="sn-dot-marketing-alert dot-credit-alert is-success">پرداخت درگاهی با موفقیت دریافت و به‌صورت خودکار تأیید مالی شد.</div><?php elseif ( in_array( $pay_result, [ 'success', 'partial_success' ], true ) ) : ?><div class="sn-dot-marketing-alert dot-credit-alert is-success">پرداخت با موفقیت تأیید شد و درخواست شما ثبت گردید.</div><?php endif; ?>
				<div class="sn-dot-marketing-alert dot-credit-alert" data-mkt-form-message hidden aria-live="polite"></div>
				<div class="sn-dot-marketing-grid dot-credit-grid">
					<label class="dot-credit-field"><span>نام و نام خانوادگی</span><input class="dot-credit-input" type="text" name="customer_name" autocomplete="name" maxlength="120" required placeholder="مثلاً محمود رضایی"></label>
					<label class="dot-credit-field"><span>شماره موبایل</span><input class="dot-credit-input" type="tel" name="customer_phone" autocomplete="tel" inputmode="numeric" maxlength="16" required placeholder="0912xxxxxxx"></label>
					<label class="dot-credit-field"><span>کد ملی</span><input class="dot-credit-input" type="text" name="national_id" inputmode="numeric" maxlength="10" required placeholder="کد ملی ۱۰ رقمی"></label>
					<label class="dot-credit-field"><span>استان</span><select class="dot-credit-input" name="province" required><option value="">انتخاب استان</option><?php foreach ( (array) $provinces as $province ) : ?><option value="<?php echo esc_attr( (string) $province ); ?>"><?php echo esc_html( (string) $province ); ?></option><?php endforeach; ?></select></label>
					<label class="dot-credit-field"><span>شهر</span><input class="dot-credit-input" type="text" name="city" autocomplete="address-level2" maxlength="60" required placeholder="نام شهر را وارد کنید"></label>
				</div>
				<button class="sn-dot-marketing-submit dot-credit-submit" type="submit">ادامه درخواست</button>
				<p class="sn-dot-marketing-foot dot-credit-privacy"><span aria-hidden="true">🔒</span> اطلاعات شما امن و محرمانه پردازش می‌شود</p>
			</form>
			<div class="sn-dot-marketing-modal" hidden aria-hidden="true">
				<div class="sn-dot-marketing-modal-backdrop" data-mkt-close></div>
				<div class="sn-dot-marketing-modal-card" role="dialog" aria-modal="true" aria-labelledby="sn-dot-mkt-modal-title">
					<button type="button" class="sn-dot-marketing-modal-close" data-mkt-close aria-label="بستن">×</button>
					<div class="sn-dot-marketing-confirm-step">
						<span class="sn-dot-marketing-modal-kicker">تکمیل درخواست</span>
						<h4><?php echo esc_html( $product->get_name() ); ?></h4>
						<p>مبلغ قابل پرداخت: <strong><?php echo esc_html( number_format_i18n( $marketing_amount ) ); ?> تومان</strong></p>
						<div class="sn-dot-marketing-modal-actions"><button type="button" class="sn-dot-marketing-modal-primary" data-mkt-continue>ادامه</button><button type="button" class="sn-dot-marketing-modal-secondary" data-mkt-cancel>انصراف</button></div>
					</div>
					<div class="sn-dot-marketing-otp-step" hidden>
						<span class="sn-dot-marketing-modal-kicker">تأیید شماره موبایل</span>
						<h4 id="sn-dot-mkt-modal-title">کد ۶ رقمی را وارد کنید</h4>
						<p class="sn-dot-marketing-otp-caption">کد برای <strong dir="ltr" data-mkt-otp-phone></strong> ارسال شد.</p>
						<input type="text" class="sn-dot-marketing-otp-input" inputmode="numeric" autocomplete="one-time-code" maxlength="6" pattern="[0-9۰-۹٠-٩]{6}" placeholder="------">
						<div class="sn-dot-marketing-otp-message" aria-live="polite"></div>
						<button type="button" class="sn-dot-marketing-modal-primary" data-mkt-verify>تأیید کد و ورود به درگاه</button>
						<button type="button" class="sn-dot-marketing-modal-secondary" data-mkt-resend disabled>ارسال مجدد کد <span data-mkt-countdown></span></button>
					</div>
				</div>
			</div>
		</div>
		<?php
		return ob_get_clean();
	}

	public function handle_marketing_start_payment(): void {
		$return_url = $this->marketing_return_url( sanitize_text_field( wp_unslash( $_POST['sn_dot_marketing_return'] ?? '' ) ) );
		if ( ! wp_verify_nonce( sanitize_text_field( wp_unslash( $_POST['sn_dot_marketing_nonce'] ?? '' ) ), 'sn_dot_marketing_start_payment' ) ) {
			$this->marketing_error_redirect( 'فرم منقضی شده است؛ صفحه را تازه‌سازی و دوباره تلاش کنید.', $return_url );
		}
		if ( trim( (string) wp_unslash( $_POST['sn_dot_marketing_website'] ?? '' ) ) !== '' ) {
			$this->marketing_error_redirect( 'درخواست نامعتبر است.', $return_url );
		}
		$form_id = sanitize_key( wp_unslash( $_POST['sn_dot_marketing_form_id'] ?? 'default' ) ) ?: 'default';
		$client_key = $this->normalize_marketing_client_key( $_POST['sn_campaign_client_key'] ?? '' );
		$config = $this->marketing_config_for_form( $form_id );
		if ( empty( $config['_form_exists'] ) || empty( $config['enabled'] ) || empty( $this->config()['enabled'] ) ) {
			$this->marketing_error_redirect( 'ورودی مارکتینگ دات فلو در حال حاضر فعال نیست.', $return_url );
		}
		$this->capture_marketing_campaign_context( $_POST );
		$name = trim( sanitize_text_field( wp_unslash( $_POST['customer_name'] ?? '' ) ) );
		$phone = SN_Helpers::normalize_mobile( sanitize_text_field( wp_unslash( $_POST['customer_phone'] ?? '' ) ) );
		$province = trim( sanitize_text_field( wp_unslash( $_POST['province'] ?? '' ) ) );
		$city = trim( sanitize_text_field( wp_unslash( $_POST['city'] ?? '' ) ) );
		$national_id = preg_replace( '/\\D+/', '', (string) SN_Helpers::to_english_nums( wp_unslash( $_POST['national_id'] ?? '' ) ) );
		if ( $name === '' ) { $this->marketing_error_redirect( 'نام و نام خانوادگی را وارد کنید.', $return_url ); }
		if ( ! SN_Helpers::is_valid_mobile( $phone ) ) { $this->marketing_error_redirect( 'شماره موبایل واردشده معتبر نیست.', $return_url ); }
		if ( $province === '' ) { $this->marketing_error_redirect( 'استان را انتخاب کنید.', $return_url ); }
		if ( $city === '' ) { $this->marketing_error_redirect( 'نام شهر را وارد کنید.', $return_url ); }
		if ( strlen( $national_id ) !== 10 ) { $this->marketing_error_redirect( 'کد ملی باید ۱۰ رقم باشد.', $return_url ); }
		$submission_id = absint( $_POST['sn_dot_marketing_submission_id'] ?? 0 );
		$event_token = sanitize_text_field( wp_unslash( $_POST['sn_dot_marketing_event_token'] ?? '' ) );
		$submission = $this->owned_marketing_submission( $submission_id, $phone, $form_id, $event_token );
		if ( ! $this->marketing_rate_limit_ok( $phone, $form_id, $client_key ) ) {
			if ( $submission ) { $this->record_marketing_event( $submission_id, 'rate_limited', 'failed', 'gateway_rate_limit' ); }
			$this->marketing_error_redirect( 'تعداد درخواست‌ها بیش از حد مجاز است. کمی بعد دوباره تلاش کنید.', $return_url );
		}
		if ( ! empty( $config['otp_enabled'] ) ) {
			$proof = sanitize_text_field( wp_unslash( $_POST['sn_dot_marketing_otp_proof'] ?? '' ) );
			if ( ! $this->consume_marketing_otp_proof( $proof, $phone, $form_id ) ) {
				if ( $submission ) { $this->record_marketing_event( $submission_id, 'otp_failed', 'failed', 'proof_invalid' ); }
				$this->marketing_error_redirect( 'شماره موبایل تأیید نشده یا کد تأیید منقضی شده است.', $return_url );
			}
		}
		if ( ! $submission ) {
			$submission_id = 0;
		}
		if ( $submission && ! empty( $submission->invoice_id ) && $this->is_gifted_assessment_invoice( (int) $submission->invoice_id ) ) {
			$this->marketing_error_redirect( 'هزینه این درخواست از طرف شرکت پرداخت شده است؛ برای ادامه از لینک پیامک‌شده استفاده کنید.', $return_url );
		}
		$allowed_supervisors = array_map( 'intval', (array) $config['supervisor_ids'] );
		$supervisor_id = $submission && (string) $submission->customer_phone === $phone && (string) $submission->form_id === $form_id && in_array( (int) $submission->supervisor_id, $allowed_supervisors, true )
			? (int) $submission->supervisor_id
			: $this->next_marketing_supervisor( $config, $form_id );
		if ( $supervisor_id < 1 ) {
			$this->marketing_error_redirect( 'سرپرست فعالی برای این فلو تنظیم نشده است.', $return_url );
		}
		$submission_id = $this->save_marketing_submission( $name, $phone, $national_id, $province, $city, $supervisor_id, $config, $submission_id, $_POST );
		$submission = $this->get_marketing_submission( $submission_id );
		if ( ! $submission ) { $this->marketing_error_redirect( 'ذخیره لید مارکتینگ انجام نشد.', $return_url ); }
		$invoice_result = $this->ensure_marketing_source_invoice( $submission, $config );
		if ( is_wp_error( $invoice_result ) ) {
			$this->record_marketing_event( $submission_id, 'invoice_failed', 'failed', sanitize_key( $invoice_result->get_error_code() ) );
			$this->marketing_error_redirect( $invoice_result->get_error_message(), $return_url );
		}
		$this->record_marketing_event( $submission_id, ! empty( $invoice_result['reused'] ) ? 'invoice_reused' : 'invoice_created', 'success', '', [ 'invoice_id' => (int) $invoice_result['invoice_id'] ], 'invoice-' . (int) $invoice_result['invoice_id'] );
		$invoice_id = absint( $invoice_result['invoice_id'] ?? 0 );
		$invoice_code = sanitize_text_field( (string) ( $invoice_result['invoice_code'] ?? '' ) );
		$amount = (float) ( $invoice_result['amount'] ?? 0 );
		$this->record_marketing_event( $submission_id, 'confirmation_accepted', 'success', '', [], 'submission-' . $submission_id );
		if ( $this->is_gifted_assessment_invoice( $invoice_id ) ) {
			$this->marketing_error_redirect( 'هزینه این درخواست از طرف شرکت پرداخت شده است؛ برای ادامه از لینک پیامک‌شده استفاده کنید.', $return_url );
		}
		if ( $event_token !== '' ) { delete_transient( self::MARKETING_EVENT_TOKEN_PREFIX . hash( 'sha256', $event_token ) ); }
		$this->remember_marketing_return_url( $invoice_id, $return_url );
		$gw = new SN_Invoice();
		$page_id = (int) get_option( 'sn_invoice_page_id' );
		$gateway = $gw->active_gateway();
		$callback_url = add_query_arg( [
			'sn_callback' => '1',
			'sn_gateway' => $gateway,
			'invoice_id' => $invoice_id,
			'sn_stage' => 1,
			'orderId' => $invoice_code,
		], $page_id ? get_permalink( $page_id ) : home_url( '/' ) );
		$this->record_marketing_event( $submission_id, 'gateway_requested', 'success', sanitize_key( $gateway ), [ 'invoice_id' => $invoice_id ], 'invoice-' . $invoice_id );
		$payment = $gw->request_payment( $invoice_id, $amount, 'پرداخت درخواست دات فلو ' . $invoice_code, $phone, $callback_url, $invoice_code, 1 );
		if ( ! empty( $payment['error'] ) || empty( $payment['url'] ) ) {
			$this->mark_marketing_gateway_result( $invoice_id, 'gateway_failed', 'gateway_request_failed' );
			$this->marketing_error_redirect( (string) ( $payment['error'] ?? 'ارتباط با درگاه پرداخت انجام نشد.' ), $return_url );
		}
		$this->mark_marketing_gateway_result( $invoice_id, 'gateway_started' );
		wp_redirect( esc_url_raw( (string) $payment['url'] ) );
		exit;
	}

	public function render_customer_flow(): string {
		$case_id = absint( $_GET['dot_case'] ?? 0 );
		$token = sanitize_text_field( wp_unslash( $_GET['dot_token'] ?? '' ) );
		$notice = sanitize_key( wp_unslash( $_GET['dot_notice'] ?? '' ) );
		$just_selected = absint( $_GET['dot_selected'] ?? 0 ) === 1;
		$customer_config = $this->config();
		$customer_ui = isset( $customer_config['customer_ui'] ) && is_array( $customer_config['customer_ui'] ) ? $customer_config['customer_ui'] : [];
		if ( $case_id < 1 ) {
			return '<div class="sn-dot-wrap" dir="rtl"><div class="sn-notice sn-error">لینک اعتبارسنجی ناقص است.</div></div>';
		}
		$session_case = $this->valid_session_case( $case_id );
		$link_case = $token !== '' ? $this->valid_link_case( $case_id, $token ) : null;
		if ( ! $session_case && $link_case && ! empty( $link_case->customer_viewed_at ) ) { $session_case = $link_case; }
		ob_start();
		?>
		<div class="sn-dot-wrap" dir="rtl" id="sn-dot-customer-flow">
			<div class="sn-dot-hero sn-dot-customer-intro"><h1><?php echo esc_html( (string) ( $customer_ui['hero_title'] ?? '' ) ); ?></h1><p><?php echo esc_html( (string) ( $customer_ui['hero_subtitle'] ?? '' ) ); ?></p></div>
			<?php if ( ! $session_case ) : ?>
				<?php if ( ! $link_case ) : ?>
					<div class="sn-notice sn-error">لینک ورود کامل نیست. لطفاً لینک درج‌شده در پیامک را دوباره باز کنید.</div>
				<?php else : ?>
					<div class="sn-dot-card sn-dot-code-card">
						<h2>ورود با رمز اختصاصی</h2>
						<?php if ( $notice === 'wrong_code' ) : ?><div class="sn-notice sn-error">رمز واردشده صحیح نیست؛ دوباره تلاش کنید.</div><?php elseif ( $notice === 'locked' ) : ?><div class="sn-notice sn-error">به‌دلیل چند تلاش ناموفق، ورود ۱۵ دقیقه قفل شده است.</div><?php endif; ?>
						<form method="post" action="<?php echo esc_url( admin_url( 'admin-post.php' ) ); ?>">
							<input type="hidden" name="action" value="sn_dot_verify_code"><input type="hidden" name="case_id" value="<?php echo esc_attr( (string) $case_id ); ?>"><input type="hidden" name="dot_token" value="<?php echo esc_attr( $token ); ?>">
							<label>رمز ۶ رقمی<input type="text" name="access_code" inputmode="numeric" pattern="[0-9۰-۹٠-٩]{6}" maxlength="6" autocomplete="one-time-code" required autofocus></label>
							<button class="sn-btn sn-btn-primary sn-dot-code-submit" type="submit"><span>ورود و مشاهده نتیجه</span><b aria-hidden="true">←</b></button>
						</form>
					</div>
				<?php endif; ?>
			<?php else :
				$case = $session_case;
				$options = json_decode( (string) $case->options_snapshot_json, true );
				$options = is_array( $options ) ? $options : [];
				// Financial fields remain frozen in the case snapshot, but editable
				// presentation fields should update for cases that are still open.
				// Match only by stable option key and fall back to the snapshot when
				// an option no longer exists in the current configuration.
				$current_option_presentation = [];
				foreach ( (array) ( $customer_config['options'] ?? [] ) as $configured_option ) {
					$configured_key = sanitize_key( (string) ( $configured_option['key'] ?? '' ) );
					if ( $configured_key === '' ) { continue; }
					$current_option_presentation[ $configured_key ] = [
						'content' => (string) ( $configured_option['content'] ?? '' ),
						'color'   => sanitize_hex_color( (string) ( $configured_option['color'] ?? '' ) ),
					];
				}
			?>
				<div class="sn-dot-card sn-dot-profile-card">
					<div class="sn-dot-profile-head">
						<div><span>پرونده مشتری</span></div>
					</div>
					<div class="sn-dot-profile-grid">
						<div class="sn-dot-profile-item"><span>نام و نام خانوادگی</span><strong><?php echo esc_html( (string) $case->customer_name ); ?></strong></div>
						<div class="sn-dot-profile-item"><span>شماره موبایل</span><strong dir="ltr"><?php echo esc_html( (string) $case->customer_phone ); ?></strong></div>
						<div class="sn-dot-profile-item"><span>استان</span><strong><?php echo esc_html( (string) ( $case->province ?: '—' ) ); ?></strong></div>
						<div class="sn-dot-profile-item"><span>شهر</span><strong><?php echo esc_html( (string) ( $case->city ?: '—' ) ); ?></strong></div>
					</div>
				</div>
				<div class="sn-dot-credit sn-dot-credit-standalone"><span>اعتبار شما</span><strong><?php echo esc_html( number_format_i18n( (float) $case->nominal_credit_amount ) ); ?> تومان</strong><small>معتبر در باشگاه مشتریان بیاوین</small></div>
				<div class="sn-dot-card sn-dot-explanation-card">
					<div class="sn-dot-explanation-copy"><?php echo nl2br( esc_html( (string) ( $customer_ui['result_description'] ?? '' ) ) ); ?></div>
				</div>
				<?php if ( ! empty( $case->selected_option_key ) || in_array( (string) $case->status, [ 'ready_for_conversion', 'assigned', 'contacted', 'payment_link_sent', 'deposit_paid', 'completed' ], true ) ) : ?>
					<div class="sn-dot-card sn-dot-success-card sn-dot-post-selection" id="sn-dot-after-selection" data-auto-scroll="<?php echo $just_selected ? '1' : '0'; ?>">
						<div class="sn-dot-success-hero">
							<span class="sn-dot-success-icon" aria-hidden="true">✓</span>
							<div class="sn-dot-success-copy">
								<h2>انتخاب شما با موفقیت ثبت شد</h2>
								<p class="sn-dot-success-subtitle">در صورت نیاز می‌توانید از مسیرهای زیر با بیاوین بیشتر آشنا شوید.</p>
							</div>
						</div>
						<div class="sn-dot-post-selection-links">
							<?php $about_url = esc_url( (string) ( $customer_ui['about_url'] ?? '' ) ); if ( $about_url !== '' ) : ?><a class="sn-dot-info-link" href="<?php echo $about_url; ?>" target="_blank" rel="noopener noreferrer"><span class="sn-dot-info-link-title">درباره بیاوین</span><small>معرفی باشگاه مشتریان</small></a><?php else : ?><span class="sn-dot-info-link is-disabled" aria-disabled="true"><span class="sn-dot-info-link-title">درباره بیاوین</span><small>معرفی باشگاه مشتریان</small></span><?php endif; ?>
							<?php $faq_url = esc_url( (string) ( $customer_ui['faq_url'] ?? '' ) ); if ( $faq_url !== '' ) : ?><a class="sn-dot-info-link" href="<?php echo $faq_url; ?>" target="_blank" rel="noopener noreferrer"><span class="sn-dot-info-link-title">سوالات متداول</span><small>پاسخ سوالات پرتکرار</small></a><?php else : ?><span class="sn-dot-info-link is-disabled" aria-disabled="true"><span class="sn-dot-info-link-title">سوالات متداول</span><small>پاسخ سوالات پرتکرار</small></span><?php endif; ?>
							<?php $contact_url = esc_url( (string) ( $customer_ui['contact_url'] ?? '' ) ); if ( $contact_url !== '' ) : ?><a class="sn-dot-info-link" href="<?php echo $contact_url; ?>" target="_blank" rel="noopener noreferrer"><span class="sn-dot-info-link-title">راه‌های ارتباطی بیاوین</span><small>مسیرهای تماس و پشتیبانی</small></a><?php else : ?><span class="sn-dot-info-link is-disabled" aria-disabled="true"><span class="sn-dot-info-link-title">راه‌های ارتباطی بیاوین</span><small>مسیرهای تماس و پشتیبانی</small></span><?php endif; ?>
						</div>
						<p class="sn-dot-advisor-message">مشاور شما مشخص و با شما ارتباط خواهد گرفت.</p>
					</div>
				<?php else : ?>
					<form class="sn-dot-options-form" method="post" action="<?php echo esc_url( admin_url( 'admin-post.php' ) ); ?>">
						<input type="hidden" name="action" value="sn_dot_select_option"><input type="hidden" name="case_id" value="<?php echo esc_attr( (string) $case_id ); ?>"><input type="hidden" name="dot_token" value="<?php echo esc_attr( $token ); ?>">
						<div class="sn-dot-section-head"><span>مرحله انتخاب</span><p>برای انتخاب، روی عنوان گزینه موردنظر بزنید.</p></div>
						<div class="sn-dot-option-grid">
						<?php $option_number = 0; $option_symbols = [ '◆', '✦', '◈', '◎' ]; foreach ( $options as $option ) : if ( empty( $option['active'] ) ) { continue; } $option_number++; $option_key = sanitize_key( (string) $option['key'] ); $input_id = 'sn-dot-option-' . $case_id . '-' . sanitize_html_class( $option_key ); $modal_id = 'sn-dot-option-modal-' . $case_id . '-' . sanitize_html_class( $option_key ); $live_presentation = $current_option_presentation[ $option_key ] ?? []; $option_color = sanitize_hex_color( (string) ( $live_presentation['color'] ?? '' ) ) ?: sanitize_hex_color( (string) ( $option['color'] ?? '' ) ); if ( ! $option_color ) { $option_color = '#b8893e'; } $option_content = array_key_exists( 'content', $live_presentation ) ? (string) $live_presentation['content'] : (string) ( $option['content'] ?? '' ); ?>
							<article class="sn-dot-option" data-option-key="<?php echo esc_attr( $option_key ); ?>" data-option-title="<?php echo esc_attr( (string) $option['title'] ); ?>" style="--sn-option-accent:<?php echo esc_attr( $option_color ); ?>">
								<input class="sn-dot-option-radio" id="<?php echo esc_attr( $input_id ); ?>" type="radio" name="option_key" value="<?php echo esc_attr( $option_key ); ?>" required>
								<button class="sn-dot-option-body sn-dot-option-open" type="button" aria-haspopup="dialog" aria-controls="<?php echo esc_attr( $modal_id ); ?>">
									<span class="sn-dot-option-symbol" aria-hidden="true"><?php echo esc_html( $option_symbols[ ( $option_number - 1 ) % count( $option_symbols ) ] ); ?></span>
									<strong><?php echo esc_html( (string) $option['title'] ); ?></strong>
								</button>
								<div class="sn-dot-option-modal sn-dot-choice-modal" id="<?php echo esc_attr( $modal_id ); ?>" role="dialog" aria-modal="true" aria-labelledby="<?php echo esc_attr( $modal_id . '-title' ); ?>" hidden>
									<div class="sn-dot-option-modal-backdrop" data-sn-dot-modal-close></div>
									<div class="sn-dot-option-modal-panel" role="document" tabindex="-1" style="--sn-option-accent:<?php echo esc_attr( $option_color ); ?>">
										<header><div><span>گزینه <?php echo esc_html( number_format_i18n( $option_number ) ); ?></span><h3 id="<?php echo esc_attr( $modal_id . '-title' ); ?>"><?php echo esc_html( (string) $option['title'] ); ?></h3></div><button class="sn-dot-option-modal-close" type="button" data-sn-dot-modal-close aria-label="بستن پنجره">×</button></header>
										<div class="sn-dot-option-fullscreen-body">
											<?php if ( trim( $option_content ) !== '' ) : ?>
												<iframe class="sn-dot-option-document-frame" title="<?php echo esc_attr( 'جزئیات ' . (string) $option['title'] ); ?>" sandbox="allow-scripts" referrerpolicy="no-referrer" srcdoc="<?php echo esc_attr( $this->option_iframe_document( $option_content ) ); ?>"></iframe>
											<?php endif; ?>
										</div>
										<footer><button class="sn-btn sn-btn-primary sn-dot-modal-select" type="button" data-option-key="<?php echo esc_attr( $option_key ); ?>" data-option-title="<?php echo esc_attr( (string) $option['title'] ); ?>">انتخاب این گزینه</button><button class="sn-btn sn-btn-secondary" type="button" data-sn-dot-modal-close>بستن</button></footer>
									</div>
								</div>
							</article>
						<?php endforeach; ?>
						</div>
						<div class="sn-dot-option-modal sn-dot-choice-modal sn-dot-confirm-modal" id="sn-dot-option-confirm-<?php echo esc_attr( (string) $case_id ); ?>" role="dialog" aria-modal="true" aria-labelledby="sn-dot-option-confirm-title-<?php echo esc_attr( (string) $case_id ); ?>" hidden>
							<div class="sn-dot-option-modal-backdrop" data-sn-dot-modal-close></div>
							<div class="sn-dot-option-modal-panel" role="document" tabindex="-1">
								<header><div><span>تأیید نهایی</span><h3 id="sn-dot-option-confirm-title-<?php echo esc_attr( (string) $case_id ); ?>">آیا از انتخاب این گزینه مطمئن هستید؟</h3></div><button class="sn-dot-option-modal-close" type="button" data-sn-dot-modal-close aria-label="بستن پنجره">×</button></header>
								<div class="sn-dot-confirm-content"><strong data-sn-dot-confirm-option></strong><p>بعد از ثبت، کارشناسان ما برای ادامه فرایند با شما تماس خواهند گرفت.</p></div>
								<footer><button class="sn-btn sn-btn-primary sn-dot-confirm-yes" type="button">بله، ثبت شود</button><button class="sn-btn sn-btn-secondary" type="button" data-sn-dot-modal-close>خیر</button></footer>
							</div>
						</div>
					</form>
				<?php endif; ?>
			<?php endif; ?>
		</div>
		<?php
		if ( $session_case && empty( $session_case->customer_viewed_at ) ) {
			global $wpdb; $t = $this->tables();
			$wpdb->update( $t['cases'], [ 'customer_viewed_at' => current_time( 'mysql' ), 'updated_at' => current_time( 'mysql' ) ], [ 'id' => $case_id ] );
		}
		return ob_get_clean();
	}

	public function handle_verify_code(): void {
		$case_id = absint( $_POST['case_id'] ?? 0 );
		$token = sanitize_text_field( wp_unslash( $_POST['dot_token'] ?? '' ) );
		$code = preg_replace( '/\D+/', '', SN_Helpers::to_english_nums( sanitize_text_field( wp_unslash( $_POST['access_code'] ?? '' ) ) ) );
		$base = $this->customer_page_base();
		// The persistent SMS link plus its six-digit code are sufficient for this
		// customer-facing flow. Do not expire access because a page was left open.
		$case = $this->valid_link_case( $case_id, $token );
		if ( ! $case ) { wp_safe_redirect( add_query_arg( [ 'dot_case' => $case_id, 'dot_notice' => 'invalid' ], $base ) ); exit; }
		global $wpdb; $t = $this->tables();
		$locked_until = trim( (string) ( $case->access_locked_until ?? '' ) );
		if ( $locked_until !== '' && $this->mysql_timestamp( $locked_until ) > time() ) {
			wp_safe_redirect( add_query_arg( [ 'dot_case' => $case_id, 'dot_token' => $token, 'dot_notice' => 'locked' ], $base ) ); exit;
		}
		if ( strlen( $code ) !== 6 || ! wp_check_password( $code, (string) $case->access_code_hash ) ) {
			$attempts = $locked_until !== '' ? 0 : (int) $case->access_failed_attempts;
			$attempts++;
			$lock_until = $attempts >= 5 ? $this->local_mysql_from_timestamp( time() + 15 * MINUTE_IN_SECONDS ) : null;
			$wpdb->update( $t['cases'], [ 'access_failed_attempts' => $attempts, 'access_locked_until' => $lock_until, 'updated_at' => current_time( 'mysql' ) ], [ 'id' => $case_id ] );
			wp_safe_redirect( add_query_arg( [ 'dot_case' => $case_id, 'dot_token' => $token, 'dot_notice' => $lock_until ? 'locked' : 'wrong_code' ], $base ) ); exit;
		}
		$session_token = bin2hex( random_bytes( 24 ) );
		$expires = $this->local_mysql_from_timestamp( time() + self::SESSION_TTL );
		$wpdb->update( $t['cases'], [ 'session_token_hash' => hash( 'sha256', $session_token ), 'session_expires_at' => $expires, 'customer_viewed_at' => current_time( 'mysql' ), 'access_failed_attempts' => 0, 'access_locked_until' => null, 'updated_at' => current_time( 'mysql' ) ], [ 'id' => $case_id ] );
		$this->set_session_cookie( $case_id, $session_token );
		$this->log_case( $case_id, 0, 'customer_access_verified', (string) $case->status, (string) $case->status );
		wp_safe_redirect( add_query_arg( [ 'dot_case' => $case_id, 'dot_token' => $this->persistent_access_token( $case ) ], $base ) ); exit;
	}

	public function handle_select_option(): void {
		$case_id = absint( $_POST['case_id'] ?? 0 );
		$token = sanitize_text_field( wp_unslash( $_POST['dot_token'] ?? '' ) );
		$base = $this->customer_page_base();
		$case = $this->valid_session_case( $case_id );
		if ( ! $case && $token !== '' ) {
			$link_case = $this->valid_link_case( $case_id, $token );
			if ( $link_case && ! empty( $link_case->customer_viewed_at ) ) { $case = $link_case; }
		}
		if ( ! $case ) { wp_safe_redirect( add_query_arg( [ 'dot_case' => $case_id, 'dot_notice' => 'invalid' ], $base ) ); exit; }
		if ( ! empty( $case->selected_option_key ) ) { wp_safe_redirect( add_query_arg( 'dot_case', $case_id, $base ) ); exit; }
		$key = sanitize_key( wp_unslash( $_POST['option_key'] ?? '' ) );
		$options = json_decode( (string) $case->options_snapshot_json, true );
		$selected = null;
		foreach ( is_array( $options ) ? $options : [] as $option ) {
			if ( ! empty( $option['active'] ) && (string) ( $option['key'] ?? '' ) === $key ) { $selected = $option; break; }
		}
		if ( ! $selected ) { wp_safe_redirect( add_query_arg( [ 'dot_case' => $case_id, 'dot_notice' => 'invalid' ], $base ) ); exit; }
		// Keep the price/title decision tied to the original case snapshot, while
		// storing the same live HTML that the customer just reviewed. If the
		// configured option was removed, preserve the historical snapshot HTML.
		$selected_html = (string) ( $selected['content'] ?? '' );
		foreach ( (array) $this->config()['options'] as $configured_option ) {
			if ( sanitize_key( (string) ( $configured_option['key'] ?? '' ) ) === $key ) {
				$selected_html = (string) ( $configured_option['content'] ?? '' );
				break;
			}
		}
		global $wpdb; $t = $this->tables(); $now = current_time( 'mysql' );
		$updated = $wpdb->update( $t['cases'], [
			'selected_option_key' => $key,
			'selected_option_product_id' => ! empty( $selected['product_id'] ) ? absint( $selected['product_id'] ) : null,
			'selected_option_title' => sanitize_text_field( (string) $selected['title'] ),
			'selected_option_price' => (float) $selected['price'],
			'selected_option_html' => $this->sanitize_option_content( $selected_html ),
			'selected_by' => ! empty( $case->customer_wp_id ) ? (int) $case->customer_wp_id : null,
			'selected_at' => $now,
			'status' => 'ready_for_conversion',
			'remaining_amount' => (float) $selected['price'],
			'updated_at' => $now,
		], [ 'id' => $case_id, 'selected_option_key' => null ] );
		if ( 1 === $updated ) {
			$this->log_case( $case_id, 0, 'customer_option_selected', (string) $case->status, 'ready_for_conversion', [ 'option_key' => $key ] );
			$redirect_args = [ 'dot_case' => $case_id, 'dot_selected' => 1 ];
			if ( $token !== '' ) { $redirect_args['dot_token'] = $token; }
			wp_safe_redirect( add_query_arg( $redirect_args, $base ) . '#sn-dot-after-selection' ); exit;
		}
		wp_safe_redirect( add_query_arg( 'dot_case', $case_id, $base ) ); exit;
	}

	private function hr_position( int $user_id ): string {
		global $wpdb;
		$profiles = $wpdb->prefix . 'sn_hr_profiles';
		$positions = $wpdb->prefix . 'sn_hr_positions';
		if ( $user_id < 1 || ! $this->table_exists( $profiles ) || ! $this->table_exists( $positions ) ) { return ''; }
		return (string) $wpdb->get_var( $wpdb->prepare( "SELECT pos.slug FROM {$profiles} hp INNER JOIN {$positions} pos ON pos.id=hp.position_id WHERE hp.user_id=%d LIMIT 1", $user_id ) );
	}

	private function is_converter( int $user_id ): bool {
		$user = get_user_by( 'id', $user_id );
		// HR is authoritative, but retain the capability fallback for sites where
		// an existing user was assigned the converter capability before the
		// legacy role was synchronized. Without it, assigned cases render but all
		// status updates are rejected by the ownership gate.
		return $user instanceof WP_User && ( in_array( 'sn_converter', (array) $user->roles, true ) || $this->hr_position( $user_id ) === 'converter' || user_can( $user, 'sn_manage_dot_conversion' ) );
	}

	private function is_seller( int $user_id ): bool {
		$user = get_user_by( 'id', $user_id );
		return $user instanceof WP_User && ( in_array( 'sn_seller', (array) $user->roles, true ) || $this->hr_position( $user_id ) === 'seller' );
	}

	private function is_senior_supervisor( int $user_id ): bool {
		$user = get_user_by( 'id', $user_id );
		return $user instanceof WP_User && ( in_array( 'sn_senior_supervisor', (array) $user->roles, true ) || $this->hr_position( $user_id ) === 'senior_supervisor' );
	}

	private function is_supervisor( int $user_id ): bool {
		$user = get_user_by( 'id', $user_id );
		if ( ! $user instanceof WP_User ) { return false; }
		return (bool) array_intersect( [ 'sn_supervisor', 'sn_senior_supervisor' ], (array) $user->roles ) || in_array( $this->hr_position( $user_id ), [ 'supervisor', 'senior_supervisor' ], true );
	}

	/** Null is the legacy value and intentionally keeps the established supervisor route. */
	private function case_conversion_route( object $case ): string {
		$route = trim( (string) ( $case->conversion_route ?? '' ) );
		return $route !== '' ? $this->normalize_conversion_route( $route ) : 'supervisor_queue';
	}

	private function is_seller_self_case( object $case ): bool {
		return 'seller_self' === $this->case_conversion_route( $case );
	}

	private function converter_candidates( int $supervisor_id ): array {
		global $wpdb;
		$out = [];
		$profiles = $wpdb->prefix . 'sn_hr_profiles';
		$positions = $wpdb->prefix . 'sn_hr_positions';
		$assignments = $wpdb->prefix . 'sn_hr_assignments';
		if ( $this->table_exists( $profiles ) && $this->table_exists( $positions ) && $this->table_exists( $assignments ) ) {
			$root = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$profiles} WHERE user_id=%d LIMIT 1", $supervisor_id ) );
			$seen = [];
			$frontier = $root ? [ $root ] : [];
			for ( $depth = 0; $frontier && $depth < 10; $depth++ ) {
				$ids_sql = implode( ',', array_map( 'absint', $frontier ) );
				$children = array_map( 'intval', (array) $wpdb->get_col( "SELECT child_profile_id FROM {$assignments} WHERE parent_profile_id IN ({$ids_sql}) AND relationship_type='reports_to' AND is_current=1" ) );
				$frontier = [];
				foreach ( $children as $profile_id ) {
					if ( isset( $seen[ $profile_id ] ) ) { continue; }
					$seen[ $profile_id ] = true;
					$frontier[] = $profile_id;
				}
			}
			if ( $seen ) {
				$ids_sql = implode( ',', array_map( 'absint', array_keys( $seen ) ) );
				$users = $wpdb->get_col( "SELECT hp.user_id FROM {$profiles} hp INNER JOIN {$positions} pos ON pos.id=hp.position_id WHERE hp.id IN ({$ids_sql}) AND pos.slug='converter' AND hp.is_active=1 AND (hp.employment_status IS NULL OR hp.employment_status NOT IN ('inactive','suspended','resigned','terminated'))" );
				foreach ( (array) $users as $user_id ) { $out[ (int) $user_id ] = get_user_by( 'id', (int) $user_id ); }
			}
		}
		foreach ( get_users( [ 'role' => 'sn_converter', 'meta_key' => 'sn_supervisor_id', 'meta_value' => $supervisor_id, 'number' => 2000, 'fields' => 'all' ] ) as $user ) { $out[ (int) $user->ID ] = $user; }
		return array_values( array_filter( $out, static fn( $user ) => $user instanceof WP_User ) );
	}

	/**
	 * Assignment targets shown to a supervisor. The supervisor is an explicit
	 * self-assignment target, while the regular converter list remains limited
	 * to active subordinate converters for team statistics.
	 */
	private function converter_assignment_targets( int $supervisor_id ): array {
		$targets = [];
		foreach ( $this->converter_candidates( $supervisor_id ) as $user ) {
			if ( $user instanceof WP_User ) { $targets[ (int) $user->ID ] = $user; }
		}
		$supervisor = get_user_by( 'id', $supervisor_id );
		if ( $supervisor instanceof WP_User ) { $targets[ $supervisor_id ] = $supervisor; }
		return array_values( $targets );
	}

	private function status_label( string $status ): string {
		$labels = [
			'awaiting_access_sms' => 'در انتظار پیامک ورود', 'awaiting_customer_selection' => 'در انتظار انتخاب مشتری',
			'ready_for_conversion' => 'آماده تبدیل', 'assigned' => 'تخصیص به تبدیل‌کننده', 'contacted' => 'تماس انجام شد',
			'customer_declined' => 'فعلاً منصرف', 'payment_link_sent' => 'لینک پرداخت ارسال شد', 'payment_rejected' => 'پرداخت رد شد',
			'deposit_paid' => 'پیش‌پرداخت تأیید شد', 'completed' => 'تکمیل پرداخت', 'archived_unpaid_subscription' => 'بایگانی عدم پرداخت اشتراک',
		];
		return $labels[ $status ] ?? $status;
	}

	private function contact_status_label( string $status ): string {
		$labels = [
			'new' => 'جدید',
			'no_answer' => 'جواب نداده',
			'contacted' => 'تماس انجام شد',
			'follow_up' => 'تماس مجدد',
			'confirmed' => 'مشتری تأیید کرد',
			'customer_declined' => 'انصراف',
			'payment_link' => 'آماده ارسال لینک پرداخت',
		];
		return $labels[ $status ] ?? $labels['new'];
	}

	private function jalali_datetime( ?string $value ): string {
		$value = trim( (string) $value );
		if ( $value === '' ) { return '—'; }
		$label = SN_Helpers::gregorian_to_jalali_date( $value );
		return preg_replace( '/\s+(\d{2}:\d{2})$/', ' — $1', $label ) ?: $label;
	}

	private function access_sms_delivery( object $case ): array {
		$sent_at = trim( (string) ( $case->sms_sent_at ?? '' ) );
		$previewed_at = trim( (string) ( $case->sms_previewed_at ?? '' ) );
		if ( $sent_at !== '' ) {
			return [ 'at' => $sent_at, 'label' => 'ارسال واقعی', 'sent' => true ];
		}
		if ( $previewed_at !== '' ) {
			return [ 'at' => $previewed_at, 'label' => 'نمایش آزمایشی', 'sent' => true ];
		}
		return [ 'at' => '', 'label' => 'هنوز ارسال نشده', 'sent' => false ];
	}

	private function case_payment_summary( object $case ): array {
		$total = max( 0, (float) ( $case->selected_option_price ?? 0 ) );
		$paid = max( 0, (float) ( $case->paid_amount ?? 0 ) );
		if ( $total > 0 ) { $paid = min( $total, $paid ); }
		$remaining = max( 0, $total - $paid );
		$completed = $total > 0 && ( $remaining <= 0.5 || (string) ( $case->status ?? '' ) === 'completed' );
		$percent = $total > 0 ? (int) round( ( $paid / $total ) * 100 ) : 0;
		if ( $completed ) { $percent = 100; }
		return [
			'total' => $total,
			'paid' => $paid,
			'remaining' => $remaining,
			'percent' => max( 0, min( 100, $percent ) ),
			'completed' => $completed,
		];
	}

	private function payment_rows_for_cases( array $case_ids ): array {
		global $wpdb;
		$t = $this->tables();
		$case_ids = array_values( array_unique( array_filter( array_map( 'absint', $case_ids ) ) ) );
		if ( ! $case_ids ) { return []; }
		$placeholders = implode( ',', array_fill( 0, count( $case_ids ), '%d' ) );
		$sql = $wpdb->prepare(
			"SELECT dp.*,i.invoice_code,i.short_code,i.status invoice_status,i.pay_method,i.payment_source
			 FROM {$t['payments']} dp
			 INNER JOIN {$wpdb->prefix}sn_invoices i ON i.id=dp.invoice_id
			 WHERE dp.case_id IN ({$placeholders})
			 ORDER BY dp.case_id ASC,dp.id DESC",
			...$case_ids
		);
		$grouped = [];
		foreach ( $wpdb->get_results( $sql ) ?: [] as $payment ) {
			$grouped[ (int) $payment->case_id ][] = $payment;
		}
		return $grouped;
	}

	private function render_case_payment_overview( object $case, array $payments, bool $allow_resend = false, bool $compact = false ): void {
		$summary = $this->case_payment_summary( $case );
		$ordered = array_reverse( $payments );
		$latest_payment_id = $payments ? max( array_map( static fn( $payment ) => (int) $payment->id, $payments ) ) : 0;
		?>
		<div class="sn-dot-payment-overview <?php echo $summary['completed'] ? 'is-completed' : 'is-incomplete'; ?><?php echo $compact ? ' is-compact' : ''; ?>">
			<div class="sn-dot-payment-overview-head">
				<div><span>وضعیت پرداخت پرونده</span><strong><?php echo $summary['completed'] ? 'پرداخت ۱۰۰٪ و تکمیل‌شده' : esc_html( number_format_i18n( $summary['percent'] ) . '٪ تکمیل‌شده' ); ?></strong></div>
				<span class="sn-dot-payment-completion <?php echo $summary['completed'] ? 'is-done' : 'is-open'; ?>"><?php echo $summary['completed'] ? 'تسویه کامل' : 'تکمیل‌نشده'; ?></span>
			</div>
			<div class="sn-dot-payment-track" aria-label="<?php echo esc_attr( 'پیشرفت پرداخت ' . $summary['percent'] . ' درصد' ); ?>"><span style="width:<?php echo esc_attr( (string) $summary['percent'] ); ?>%"></span></div>
			<div class="sn-dot-payment-totals">
				<span><small>مبلغ کل</small><strong><?php echo esc_html( number_format_i18n( $summary['total'] ) ); ?> تومان</strong></span>
				<span><small>پرداخت تأییدشده</small><strong><?php echo esc_html( number_format_i18n( $summary['paid'] ) ); ?> تومان</strong></span>
				<span><small>مانده</small><strong><?php echo esc_html( number_format_i18n( $summary['remaining'] ) ); ?> تومان</strong></span>
			</div>
			<div class="sn-dot-payment-stage-title"><strong>مراحل پرداخت</strong><span><?php echo esc_html( number_format_i18n( count( $ordered ) ) ); ?> مرحله ثبت‌شده</span></div>
			<?php if ( ! $ordered ) : ?>
				<div class="sn-dot-payment-empty">هنوز لینک یا مرحله پرداختی برای این پرونده ساخته نشده است.</div>
			<?php else : ?>
				<ol class="sn-dot-payment-timeline">
				<?php foreach ( $ordered as $index => $payment ) :
					$is_paid = (string) $payment->status === 'paid';
					$is_rejected = (string) $payment->status === 'rejected';
					$is_open = in_array( (string) $payment->status, [ 'pending', 'pending_finance' ], true );
					$stage_class = $is_paid ? 'is-paid' : ( $is_rejected ? 'is-rejected' : ( $is_open ? 'is-pending' : 'is-muted' ) );
					$event_at = $is_paid && ! empty( $payment->paid_at ) ? (string) $payment->paid_at : (string) ( $payment->updated_at ?: $payment->created_at );
				?>
					<li class="<?php echo esc_attr( $stage_class ); ?>">
						<span class="sn-dot-payment-stage-marker"><?php echo $is_paid ? '✓' : esc_html( number_format_i18n( $index + 1 ) ); ?></span>
						<div class="sn-dot-payment-stage-main"><span>مرحله <?php echo esc_html( number_format_i18n( $index + 1 ) ); ?></span><strong><?php echo esc_html( $this->payment_type_label( (string) $payment->payment_type ) ); ?></strong><small>فاکتور <code><?php echo esc_html( (string) $payment->invoice_code ); ?></code> · <?php echo esc_html( $this->jalali_datetime( $event_at ) ); ?></small><button type="button" class="sn-btn sn-btn-sm sn-btn-secondary sn-open-invoice-link" data-invoice-id="<?php echo esc_attr( (string) $payment->invoice_id ); ?>">باز کردن پیش‌فاکتور و ثبت فیش</button><small class="sn-open-invoice-link-msg" aria-live="polite"></small></div>
						<div class="sn-dot-payment-stage-side"><strong><?php echo esc_html( number_format_i18n( (float) $payment->requested_amount ) ); ?> تومان</strong><span><?php echo esc_html( $this->payment_status_label( (string) $payment->status ) ); ?></span></div>
						<?php if ( $allow_resend && (int) $payment->id === $latest_payment_id && in_array( (string) $payment->status, [ 'pending', 'pending_finance', 'rejected' ], true ) ) : ?>
							<form class="sn-dot-action-form sn-dot-payment-resend" method="post" action="<?php echo esc_url( admin_url( 'admin-post.php' ) ); ?>"><input type="hidden" name="action" value="sn_dot_resend_payment"><input type="hidden" name="payment_id" value="<?php echo esc_attr( (string) $payment->id ); ?>"><?php wp_nonce_field( 'sn_dot_resend_payment_' . (int) $payment->id ); ?><button type="submit" class="sn-btn sn-btn-sm sn-btn-ghost">ارسال مجدد لینک</button></form>
						<?php endif; ?>
					</li>
				<?php endforeach; ?>
				</ol>
			<?php endif; ?>
		</div>
		<?php
	}

	public function render_supervisor_ready_tab( int $viewer_id, bool $read_only = false ): void {
		global $wpdb;
		$viewer_id = absint( $viewer_id );
		$queue_owner_id = $viewer_id;
		// Move only safe legacy rows (missing, unassigned, or seller-self) to the
		// direct-supervisor queue. Assigned cases keep their original owner/history.
		if ( ! $read_only ) { $this->reconcile_paid_assessment_scope_for_supervisor( $queue_owner_id, 100 ); }
		// Recover a missed finance hook before reading the queue. No customer
		// option is required; unselected cases remain visible and assignable.
		if ( ! $read_only ) { $this->repair_paid_assessment_cases_for_supervisor( $queue_owner_id, 20 ); }
		// Older paid-invoice referrals used a parallel "repeat action" table/UI.
		// Convert only this actor's bounded pending rows, keeping their origin rows.
		$this->migrate_paid_referrals_for_actor( $viewer_id, 100 );
		$t = $this->tables();
		// `created_at` is the immutable moment the case entered this conversion
		// workspace. Selection, assignment and later edits must not reshuffle rows.
		if ( $read_only ) {
			$seller_ids = class_exists( 'SN_Scope_Service' ) ? ( new SN_Scope_Service() )->scope_visible_seller_ids( $viewer_id ) : [];
			if ( ! $seller_ids ) { $seller_ids = get_users( [ 'role' => 'sn_seller', 'fields' => 'ids', 'meta_key' => 'sn_supervisor_id', 'meta_value' => $viewer_id ] ); }
			$seller_ids = array_values( array_unique( array_filter( array_map( 'absint', (array) $seller_ids ) ) ) );
			if ( $seller_ids ) {
				$placeholders = implode( ',', array_fill( 0, count( $seller_ids ), '%d' ) );
				$cases = $wpdb->get_results( $wpdb->prepare( "SELECT * FROM {$t['cases']} WHERE seller_id IN ({$placeholders}) AND operational_archived_at IS NULL AND status<>'customer_declined' ORDER BY created_at DESC,id DESC LIMIT 300", ...$seller_ids ) ) ?: [];
			} else {
				$cases = [];
			}
		} else {
			$cases = $wpdb->get_results( $wpdb->prepare( "SELECT * FROM {$t['cases']} WHERE supervisor_id=%d AND operational_archived_at IS NULL AND status<>'customer_declined' ORDER BY created_at DESC,id DESC LIMIT 300", $queue_owner_id ) ) ?: [];
		}
		$converters = $this->converter_candidates( $viewer_id );
		$assignment_targets = $read_only ? [] : $this->converter_assignment_targets( $queue_owner_id );
		$ready = array_values( $cases );
		$waiting = array_values( array_filter( $cases, static fn( $case ) => empty( $case->selected_option_key ) ) );
		$selected_count = count( $ready ) - count( $waiting );
		$assignable_count = $read_only ? 0 : count( array_filter( $ready, fn( $case ) => ! $this->is_seller_self_case( $case ) && (string) $case->status !== 'completed' ) );
		$payments_by_case = $this->payment_rows_for_cases( array_map( static fn( $case ) => (int) $case->id, $ready ) );
		foreach ( $ready as $index => $case ) {
			if ( ! empty( $case->selected_option_key ) ) { $ready[ $index ] = $this->sync_live_case_option_prices( $case, ! empty( $payments_by_case[ (int) $case->id ] ) ); }
		}
		?>
		<div class="sn-dot-supervisor" dir="rtl">
			<div class="sn-card sn-dot-supervisor-summary"><div class="sn-dot-section-title"><div><span>مرکز تبدیل مشتریان</span><h3>آماده‌های تبدیل</h3><p><?php echo esc_html( $read_only ? 'پرونده‌های آماده تبدیل فروشندگان زیرمجموعه را به‌صورت تجمیعی مشاهده می‌کنید؛ تخصیص و تغییر مسئول در پنل سرپرست مستقیم انجام می‌شود.' : 'فاکتورهای اعتبارسنجی پس از تکمیل پرداخت و تأیید مالی در صف شما قرار می‌گیرند و می‌توانید آن‌ها را به خودتان یا تبدیل‌کننده تخصیص دهید.' ); ?></p></div></div><div class="sn-dot-kpis"><div><strong><?php echo esc_html( number_format_i18n( $selected_count ) ); ?></strong><span>انتخاب ثبت‌شده</span></div><div><strong><?php echo esc_html( number_format_i18n( count( $waiting ) ) ); ?></strong><span>انتخاب نکرده</span></div><div><strong><?php echo esc_html( number_format_i18n( count( $converters ) ) ); ?></strong><span>تبدیل‌کننده فعال زیرمجموعه</span></div></div></div>
			<div class="sn-dot-assign-notice" aria-live="polite"></div>
			<?php if ( $read_only ) : ?><div class="sn-notice sn-info">این نمای سرپرست ارشد فقط نظارتی است. مسئولیت تخصیص هر پرونده با سرپرست مستقیم همان فروشنده است.</div><?php elseif ( ! $converters ) : ?><div class="sn-notice sn-info">تبدیل‌کننده فعالی زیر این سرپرست تعریف نشده است؛ می‌توانید پرونده را با گزینه «تخصیص به خودم» پیگیری کنید.</div><?php endif; ?>
			<section class="sn-card sn-dot-supervisor-group is-ready"><div class="sn-dot-section-title"><div><span>صف عملیاتی</span><h3>پرونده‌های آماده تبدیل</h3><p><?php echo esc_html( $read_only ? 'پرونده‌های دارای تگ «انتخاب نکرده» نیز برای نظارت نمایش داده می‌شوند و سرپرست مستقیم آن‌ها را تخصیص می‌دهد.' : 'پروندهٔ دارای تگ «انتخاب نکرده» را هم می‌توانید به خودتان یا تبدیل‌کننده تخصیص دهید تا محصول نهایی هنگام تماس انتخاب شود.' ); ?></p></div><strong><?php echo esc_html( number_format_i18n( count( $ready ) ) ); ?> پرونده</strong></div>
				<?php if ( $assignable_count > 0 && $assignment_targets ) : ?>
				<div class="sn-dot-bulk-assign" data-sn-dot-bulk-assign>
					<div class="sn-dot-bulk-assign-copy"><strong>تخصیص گروهی</strong><span><b data-sn-dot-selected-count>۰</b> پرونده انتخاب شده</span></div>
					<select class="sn-dot-bulk-converter" aria-label="انتخاب تبدیل‌کننده یا تخصیص به خودم برای پرونده‌های انتخاب‌شده"><option value="">تبدیل‌کننده یا خودم</option><?php foreach ( $assignment_targets as $target ) : ?><option value="<?php echo esc_attr( (string) $target->ID ); ?>"><?php echo esc_html( (int) $target->ID === $queue_owner_id ? 'تخصیص به خودم — ' . $target->display_name . ' (#' . $target->ID . ')' : (string) $target->display_name . ' (#' . $target->ID . ')' ); ?></option><?php endforeach; ?></select>
					<button type="button" class="sn-btn sn-btn-primary sn-dot-bulk-assign-btn" disabled>تخصیص پرونده‌های انتخاب‌شده</button>
				</div>
				<?php endif; ?>
				<div class="sn-table-wrap"><table class="sn-table sn-dot-supervisor-ready-table"><thead><tr><th class="sn-dot-select-col"><?php if ( $assignable_count > 0 ) : ?><input type="checkbox" class="sn-dot-select-all" aria-label="انتخاب همه پرونده‌های قابل تخصیص"><?php else : ?>—<?php endif; ?></th><th>پرونده</th><th>مشتری</th><th>اعتبار</th><th>انتخاب مشتری</th><th>وضعیت</th><th>پرداخت</th><th>تبدیل‌کننده</th><th><?php echo $read_only ? 'دسترسی' : 'اقدام'; ?></th></tr></thead><tbody>
			<?php if ( ! $ready ) : ?><tr><td colspan="9"><div class="sn-empty-state">پرونده آماده تبدیلی وجود ندارد.</div></td></tr><?php endif; ?>
			<?php foreach ( $ready as $case ) : $payments = $payments_by_case[ (int) $case->id ] ?? []; $payment_summary = $this->case_payment_summary( $case ); $is_completed = (string) $case->status === 'completed'; $seller_self = $this->is_seller_self_case( $case ); ?>
				<tr data-sn-dot-case-row="<?php echo esc_attr( (string) $case->id ); ?>"><td class="sn-dot-select-col"><?php if ( ! $read_only && ! $is_completed && ! $seller_self ) : ?><input type="checkbox" class="sn-dot-case-check" value="<?php echo esc_attr( (string) $case->id ); ?>" aria-label="انتخاب پرونده <?php echo esc_attr( (string) $case->id ); ?>"><?php elseif ( $seller_self ) : ?><span class="sn-dot-check-done" title="این پرونده توسط فروشنده تبدیل می‌شود">◉</span><?php elseif ( $is_completed ) : ?><span class="sn-dot-check-done">✓</span><?php else : ?>—<?php endif; ?></td><td>#<?php echo esc_html( (string) $case->id ); ?><br><small title="زمان ورود به صف تبدیل"><?php echo esc_html( $this->jalali_datetime( (string) $case->created_at ) ); ?></small></td><td><strong><?php echo esc_html( (string) $case->customer_name ); ?></strong><br><code><?php echo esc_html( (string) $case->customer_phone ); ?></code></td><td><?php echo esc_html( number_format_i18n( (float) $case->nominal_credit_amount ) ); ?> تومان</td><td><?php if ( ! empty( $case->selected_option_key ) ) : ?><strong><?php echo esc_html( (string) $case->selected_option_title ); ?></strong><br><small><?php echo esc_html( number_format_i18n( (float) $case->selected_option_price ) ); ?> تومان</small><?php else : ?><span class="sn-badge">انتخاب نکرده</span><br><small>توسط مسئول تماس انتخاب شود</small><?php endif; ?></td><td><span class="sn-badge"><?php echo esc_html( $this->status_label( (string) $case->status ) ); ?></span><?php if ( ! empty( $case->validation_fee_gifted ) ) : ?><span class="sn-dot-validation-gift-tag">اعتبارسنجی هدیه</span><?php endif; ?></td><td><div class="sn-dot-supervisor-payment-cell"><span class="sn-dot-payment-completion <?php echo $payment_summary['completed'] ? 'is-done' : 'is-open'; ?>"><?php echo $payment_summary['completed'] ? 'پرداخت ۱۰۰٪' : esc_html( number_format_i18n( $payment_summary['percent'] ) . '٪ پرداخت' ); ?></span><small><?php echo esc_html( number_format_i18n( $payment_summary['paid'] ) ); ?> از <?php echo esc_html( number_format_i18n( $payment_summary['total'] ) ); ?> تومان</small><details class="sn-dot-supervisor-payment-details"><summary>مشاهده مراحل پرداخت</summary><?php $this->render_case_payment_overview( $case, $payments, false, true ); ?></details></div></td><td class="sn-dot-assigned-converter"><?php if ( $seller_self ) : ?><span class="sn-badge">فروشنده</span><br><small><?php echo esc_html( (string) get_the_author_meta( 'display_name', (int) $case->seller_id ) ); ?></small><?php else : ?><?php echo (int) $case->converter_id > 0 ? esc_html( (string) get_the_author_meta( 'display_name', (int) $case->converter_id ) ) : '—'; ?><?php endif; ?></td><td>
					<?php if ( $read_only ) : ?><span class="sn-badge">فقط مشاهده</span><?php elseif ( $seller_self ) : ?><span class="sn-badge">نظارتی — بدون تخصیص مجدد</span><?php elseif ( $is_completed ) : ?>تکمیل‌شده<?php else : ?><form class="sn-dot-action-form sn-dot-assign-form" method="post" action="<?php echo esc_url( admin_url( 'admin-post.php' ) ); ?>"><input type="hidden" name="action" value="sn_dot_assign_converter"><input type="hidden" name="case_id" value="<?php echo esc_attr( (string) $case->id ); ?>"><?php wp_nonce_field( 'sn_dot_assign_' . (int) $case->id ); ?><select name="converter_id" required><option value="">تبدیل‌کننده یا خودم</option><?php foreach ( $assignment_targets as $target ) : ?><option value="<?php echo esc_attr( (string) $target->ID ); ?>" <?php selected( (int) $case->converter_id, (int) $target->ID ); ?>><?php echo esc_html( (int) $target->ID === $queue_owner_id ? 'تخصیص به خودم — ' . $target->display_name . ' (#' . $target->ID . ')' : (string) $target->display_name . ' (#' . $target->ID . ')' ); ?></option><?php endforeach; ?></select><button class="sn-btn sn-btn-sm sn-btn-primary" type="submit"><?php echo (int) $case->converter_id ? 'تغییر تخصیص' : 'تخصیص'; ?></button></form><?php endif; ?>
				</td></tr>
			<?php endforeach; ?>
			</tbody></table></div></section>
		</div>
		<?php
	}

	public function render_supervisor_converters_tab( int $viewer_id ): void {
		global $wpdb;
		$t = $this->tables();
		$viewer_id = absint( $viewer_id );
		$is_senior_view = $this->is_senior_supervisor( $viewer_id );
		$active_users = $this->converter_candidates( $viewer_id );
		$active_ids = [];
		$users = [];
		foreach ( $active_users as $user ) {
			$active_ids[ (int) $user->ID ] = true;
			$users[ (int) $user->ID ] = $user;
		}
		if ( $is_senior_view ) {
			$seller_ids = class_exists( 'SN_Scope_Service' ) ? ( new SN_Scope_Service() )->scope_visible_seller_ids( $viewer_id ) : [];
			if ( ! $seller_ids ) { $seller_ids = get_users( [ 'role' => 'sn_seller', 'fields' => 'ids', 'meta_key' => 'sn_supervisor_id', 'meta_value' => $viewer_id ] ); }
			$seller_ids = array_values( array_unique( array_filter( array_map( 'absint', (array) $seller_ids ) ) ) );
			$scope_where = $seller_ids ? 'seller_id IN (' . implode( ',', $seller_ids ) . ')' : '1=0';
		} else {
			$scope_where = $wpdb->prepare( 'supervisor_id=%d', $viewer_id );
		}
		$rows = $wpdb->get_results(
			"SELECT converter_id,
				COUNT(*) assigned_count,
				SUM(CASE WHEN COALESCE(converter_contact_status,'') IN ('contacted','follow_up','confirmed','customer_declined') OR status IN ('contacted','customer_declined','payment_link_sent','deposit_paid','completed') THEN 1 ELSE 0 END) contacted_count,
				SUM(CASE WHEN status IN ('payment_link_sent','payment_rejected','deposit_paid','completed') THEN 1 ELSE 0 END) payment_count,
				SUM(CASE WHEN status='completed' THEN 1 ELSE 0 END) completed_count,
				COALESCE(SUM(paid_amount),0) paid_total,
				MAX(updated_at) last_activity
			FROM {$t['cases']}
			WHERE {$scope_where} AND converter_id IS NOT NULL AND converter_id>0 AND (conversion_route IS NULL OR conversion_route<>'seller_self')
			GROUP BY converter_id"
		) ?: [];

		$stats = [];
		foreach ( $rows as $row ) {
			$converter_id = (int) $row->converter_id;
			if ( ! isset( $users[ $converter_id ] ) ) {
				$user = get_user_by( 'id', $converter_id );
				if ( $user instanceof WP_User ) { $users[ $converter_id ] = $user; }
			}
			$stats[ $converter_id ] = [
				'id' => $converter_id,
				'assigned' => (int) $row->assigned_count,
				'contacted' => (int) $row->contacted_count,
				'payment' => (int) $row->payment_count,
				'completed' => (int) $row->completed_count,
				'paid_total' => (float) $row->paid_total,
				'last_activity' => (string) $row->last_activity,
			];
			if ( ! isset( $users[ $converter_id ] ) ) { unset( $stats[ $converter_id ] ); }
		}
		foreach ( array_keys( $active_ids ) as $converter_id ) {
			if ( isset( $stats[ $converter_id ] ) ) { continue; }
			$stats[ $converter_id ] = [
				'id' => $converter_id,
				'assigned' => 0,
				'contacted' => 0,
				'payment' => 0,
				'completed' => 0,
				'paid_total' => 0.0,
				'last_activity' => '',
			];
		}
		$stats = array_values( $stats );
		usort( $stats, static function ( array $a, array $b ): int {
			if ( $a['completed'] !== $b['completed'] ) { return $b['completed'] <=> $a['completed']; }
			if ( $a['assigned'] !== $b['assigned'] ) { return $b['assigned'] <=> $a['assigned']; }
			return $a['id'] <=> $b['id'];
		} );
		$total_assigned = array_sum( array_column( $stats, 'assigned' ) );
		$total_completed = array_sum( array_column( $stats, 'completed' ) );
		$total_paid = array_sum( array_column( $stats, 'paid_total' ) );
		$overall_rate = $total_assigned > 0 ? (int) round( ( $total_completed / $total_assigned ) * 100 ) : 0;
		?>
		<div class="sn-dot-supervisor sn-dot-converter-analytics" dir="rtl">
			<div class="sn-card sn-dot-analytics-summary">
				<div class="sn-dot-analytics-title"><div><span>عملکرد تیم تبدیل</span><h3>تبدیل‌کنندگان و آمار</h3></div><small><?php echo esc_html( $is_senior_view ? 'آمار تجمیعی صف‌های سرپرستان زیرمجموعه' : 'آمار صف تبدیل سرپرست' ); ?></small></div>
				<div class="sn-dot-kpis sn-dot-kpis-four">
					<div><strong><?php echo esc_html( number_format_i18n( count( $active_ids ) ) ); ?></strong><span>تبدیل‌کننده فعال</span></div>
					<div><strong><?php echo esc_html( number_format_i18n( $total_assigned ) ); ?></strong><span>پرونده تخصیص‌یافته</span></div>
					<div><strong><?php echo esc_html( number_format_i18n( $total_completed ) ); ?></strong><span>تبدیل تکمیل‌شده</span></div>
					<div><strong><?php echo esc_html( number_format_i18n( $overall_rate ) ); ?>٪</strong><span>نرخ تبدیل کل</span></div>
				</div>
			</div>
			<div class="sn-card">
				<div class="sn-dot-analytics-list-head"><div><h3>عملکرد اعضای تیم</h3><p>از تماس اولیه تا تکمیل پرداخت</p></div><strong><?php echo esc_html( number_format_i18n( $total_paid ) ); ?> تومان پرداخت تأییدشده</strong></div>
				<?php if ( ! $stats ) : ?><div class="sn-empty-state">هنوز تبدیل‌کننده یا پرونده‌ای برای نمایش وجود ندارد.</div><?php endif; ?>
				<div class="sn-dot-converter-stat-list">
				<?php foreach ( $stats as $stat ) :
					$converter_id = (int) $stat['id'];
					$user = $users[ $converter_id ] ?? null;
					$name = $user instanceof WP_User ? (string) $user->display_name : 'کاربر #' . $converter_id;
					$rate = $stat['assigned'] > 0 ? (int) round( ( $stat['completed'] / $stat['assigned'] ) * 100 ) : 0;
					$is_active = isset( $active_ids[ $converter_id ] );
				?>
					<article class="sn-dot-converter-stat-card">
						<div class="sn-dot-stat-person"><span class="sn-dot-stat-avatar" aria-hidden="true">↗</span><div><h4><?php echo esc_html( $name ); ?></h4><small>#<?php echo esc_html( (string) $converter_id ); ?> · <?php echo $is_active ? 'عضو فعال تیم' : 'عضو سابق تیم'; ?></small></div><span class="sn-dot-stat-state <?php echo $is_active ? 'is-active' : 'is-former'; ?>"><?php echo $is_active ? 'فعال' : 'سابق'; ?></span></div>
						<div class="sn-dot-stat-numbers"><span><small>تخصیص</small><strong><?php echo esc_html( number_format_i18n( $stat['assigned'] ) ); ?></strong></span><span><small>تماس</small><strong><?php echo esc_html( number_format_i18n( $stat['contacted'] ) ); ?></strong></span><span><small>ورود به پرداخت</small><strong><?php echo esc_html( number_format_i18n( $stat['payment'] ) ); ?></strong></span><span><small>تکمیل</small><strong><?php echo esc_html( number_format_i18n( $stat['completed'] ) ); ?></strong></span></div>
						<div class="sn-dot-stat-progress"><div><span>نرخ تبدیل</span><strong><?php echo esc_html( number_format_i18n( $rate ) ); ?>٪</strong></div><div class="sn-dot-stat-track"><span style="width:<?php echo esc_attr( (string) $rate ); ?>%"></span></div></div>
						<div class="sn-dot-stat-footer"><span><?php echo esc_html( number_format_i18n( $stat['paid_total'] ) ); ?> تومان پرداخت تأییدشده</span><small>آخرین فعالیت: <?php echo esc_html( $stat['last_activity'] !== '' ? SN_Helpers::gregorian_to_jalali_date( $stat['last_activity'] ) : '—' ); ?></small></div>
					</article>
				<?php endforeach; ?>
				</div>
			</div>
		</div>
		<?php
	}

	private function can_manage_supervisor_case( object $case, int $user_id ): bool {
		return current_user_can( 'manage_options' ) || (int) $case->supervisor_id === $user_id;
	}

	private function can_manage_converter_case( object $case, int $user_id ): bool {
		if ( current_user_can( 'manage_options' ) ) { return true; }
		if ( ! empty( $case->operational_archived_at ) ) { return false; }
		if ( (int) $case->converter_id !== $user_id ) { return false; }
		if ( $this->is_seller_self_case( $case ) ) {
			return (int) $case->seller_id === $user_id && $this->is_seller( $user_id );
		}
		// A supervisor may act as a converter only on cases explicitly assigned
		// to that same supervisor; this does not grant access to other converters'
		// queues or to unrelated cases.
		return $this->is_converter( $user_id ) || ( $this->is_supervisor( $user_id ) && (int) $case->supervisor_id === $user_id );
	}

	private function finish_action( bool $success, string $message, string $tab = '' ): void {
		if ( wp_doing_ajax() ) {
			$payload = [
				'message' => $message,
				'action' => sanitize_key( wp_unslash( $_POST['action'] ?? '' ) ),
				'case_id' => absint( $_POST['case_id'] ?? 0 ),
				'payment_id' => absint( $_POST['payment_id'] ?? 0 ),
			];
			if ( $success ) { wp_send_json_success( $payload ); }
			wp_send_json_error( $payload, 400 );
		}
		$back = wp_get_referer() ?: home_url( '/' );
		if ( $tab === 'needs-action' && $this->is_senior_supervisor( get_current_user_id() ) ) { $tab = 'ss-repeat-actions'; }
		$args = [ 'sn_dot_result' => $success ? 'success' : 'error', 'sn_dot_message' => $message ];
		if ( $tab !== '' ) { $args['sn_dot_tab'] = $tab; }
		wp_safe_redirect( add_query_arg( $args, $back ) );
		exit;
	}

	public function handle_assign_converter(): void {
		$case_id = absint( $_POST['case_id'] ?? 0 );
		if ( ! wp_verify_nonce( sanitize_text_field( wp_unslash( $_POST['_wpnonce'] ?? '' ) ), 'sn_dot_assign_' . $case_id ) ) { $this->finish_action( false, 'درخواست امنیتی نامعتبر است.', 'needs-action' ); }
		$case = $this->get_case( $case_id );
		$user_id = get_current_user_id();
		if ( ! $case || ! $this->can_manage_supervisor_case( $case, $user_id ) || ! empty( $case->operational_archived_at ) || in_array( (string) $case->status, [ 'completed', 'customer_declined', 'archived_unpaid_subscription' ], true ) ) { $this->finish_action( false, 'دسترسی یا وضعیت این پرونده برای تخصیص معتبر نیست.', 'needs-action' ); }
		if ( $this->is_seller_self_case( $case ) ) { $this->finish_action( false, 'این پرونده با انتخاب فروشنده در مسیر «تبدیل توسط خودم» است و برای سرپرست فقط حالت نظارتی دارد.', 'needs-action' ); }
		$converter_id = absint( $_POST['converter_id'] ?? 0 );
		$valid_ids = array_map( static fn( $user ) => (int) $user->ID, $this->converter_assignment_targets( (int) $case->supervisor_id ) );
		if ( ! in_array( $converter_id, $valid_ids, true ) ) { $this->finish_action( false, 'تبدیل‌کننده انتخاب‌شده زیرمجموعه این سرپرست نیست.', 'needs-action' ); }
		global $wpdb; $t = $this->tables(); $now = current_time( 'mysql' );
		$wpdb->query( 'START TRANSACTION' );
		$locked_case = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['cases']} WHERE id=%d FOR UPDATE", $case_id ) );
		if ( ! $locked_case || ! $this->can_manage_supervisor_case( $locked_case, $user_id ) || ! empty( $locked_case->operational_archived_at ) || in_array( (string) $locked_case->status, [ 'completed', 'customer_declined', 'archived_unpaid_subscription' ], true ) ) {
			$wpdb->query( 'ROLLBACK' );
			$this->finish_action( false, 'وضعیت یا دسترسی پرونده در حین تخصیص تغییر کرده است.', 'needs-action' );
		}
		if ( $this->is_seller_self_case( $locked_case ) ) {
			$wpdb->query( 'ROLLBACK' );
			$this->finish_action( false, 'مسیر این پرونده به «تبدیل توسط فروشنده» تغییر کرده و تخصیص مجدد مجاز نیست.', 'needs-action' );
		}
		$case = $locked_case;
		$new_status = in_array( (string) $case->status, [ 'payment_link_sent', 'payment_rejected', 'deposit_paid' ], true ) ? (string) $case->status : 'assigned';
		$updated = $wpdb->update( $t['cases'], [ 'converter_id' => $converter_id, 'assigned_by' => $user_id, 'assigned_at' => $now, 'status' => $new_status, 'updated_at' => $now ], [ 'id' => $case_id ] );
		if ( false === $updated ) { $wpdb->query( 'ROLLBACK' ); $this->finish_action( false, 'ذخیره تخصیص انجام نشد.', 'needs-action' ); }
		$wpdb->query( 'COMMIT' );
		$this->log_case( $case_id, $user_id, 'converter_assigned', (string) $case->status, $new_status, [ 'old_converter_id' => (int) $case->converter_id, 'converter_id' => $converter_id ] );
		if ( class_exists( 'SN_Purpose_Commission' ) ) { SN_Purpose_Commission::reconcile_dot_case( $case_id ); }
		$this->finish_action( true, $converter_id === (int) $case->supervisor_id ? 'پرونده به خودتان تخصیص داده شد و در تب «تبدیل‌های من» قابل پیگیری است.' : 'پرونده به تبدیل‌کننده تخصیص داده شد.', 'needs-action' );
	}

	/** Ajax-only atomic bulk assignment of multiple Dot cases to one converter. */
	public function handle_bulk_assign_converter(): void {
		if ( ! is_user_logged_in() || ! check_ajax_referer( 'sn_public', 'nonce', false ) ) {
			wp_send_json_error( [ 'message' => 'نشست کاربری معتبر نیست؛ صفحه را تازه‌سازی کنید.' ], 403 );
		}
		$user_id = get_current_user_id();
		$raw_ids = isset( $_POST['case_ids'] ) && is_array( $_POST['case_ids'] ) ? wp_unslash( $_POST['case_ids'] ) : [];
		$case_ids = array_values( array_unique( array_filter( array_map( 'absint', $raw_ids ) ) ) );
		$case_ids = array_slice( $case_ids, 0, 300 );
		$converter_id = absint( $_POST['converter_id'] ?? 0 );
		if ( ! $case_ids || $converter_id < 1 ) {
			wp_send_json_error( [ 'message' => 'حداقل یک پرونده و یک تبدیل‌کننده انتخاب کنید.' ], 400 );
		}

		global $wpdb;
		$t = $this->tables();
		$placeholders = implode( ',', array_fill( 0, count( $case_ids ), '%d' ) );
		$wpdb->query( 'START TRANSACTION' );
		$locked = $wpdb->get_results( $wpdb->prepare( "SELECT * FROM {$t['cases']} WHERE id IN ({$placeholders}) FOR UPDATE", ...$case_ids ) ) ?: [];
		if ( count( $locked ) !== count( $case_ids ) ) {
			$wpdb->query( 'ROLLBACK' );
			wp_send_json_error( [ 'message' => 'یک یا چند پرونده دیگر در دسترس نیستند. فهرست را تازه‌سازی کنید.' ], 409 );
		}

		$candidate_cache = [];
		$changes = [];
		$now = current_time( 'mysql' );
		foreach ( $locked as $case ) {
			$case_id = (int) $case->id;
			if ( ! $this->can_manage_supervisor_case( $case, $user_id ) || ! empty( $case->operational_archived_at ) || in_array( (string) $case->status, [ 'completed', 'customer_declined', 'archived_unpaid_subscription' ], true ) ) {
				$wpdb->query( 'ROLLBACK' );
				wp_send_json_error( [ 'message' => 'پرونده #' . $case_id . ' دیگر برای تخصیص معتبر نیست.' ], 409 );
			}
			if ( $this->is_seller_self_case( $case ) ) {
				$wpdb->query( 'ROLLBACK' );
				wp_send_json_error( [ 'message' => 'پرونده #' . $case_id . ' توسط فروشنده تبدیل می‌شود و فقط برای نظارت سرپرست نمایش داده شده است.' ], 409 );
			}
			$supervisor_id = (int) $case->supervisor_id;
			if ( ! isset( $candidate_cache[ $supervisor_id ] ) ) {
				$candidate_cache[ $supervisor_id ] = array_map( static fn( $user ) => (int) $user->ID, $this->converter_assignment_targets( $supervisor_id ) );
			}
			if ( ! in_array( $converter_id, $candidate_cache[ $supervisor_id ], true ) ) {
				$wpdb->query( 'ROLLBACK' );
				wp_send_json_error( [ 'message' => 'تبدیل‌کننده انتخاب‌شده برای همه پرونده‌های انتخابی مجاز نیست.' ], 403 );
			}
			$new_status = in_array( (string) $case->status, [ 'payment_link_sent', 'payment_rejected', 'deposit_paid' ], true ) ? (string) $case->status : 'assigned';
			$updated = $wpdb->update( $t['cases'], [ 'converter_id' => $converter_id, 'assigned_by' => $user_id, 'assigned_at' => $now, 'status' => $new_status, 'updated_at' => $now ], [ 'id' => $case_id ] );
			if ( false === $updated ) {
				$wpdb->query( 'ROLLBACK' );
				wp_send_json_error( [ 'message' => 'ذخیره تخصیص پرونده #' . $case_id . ' انجام نشد؛ هیچ پرونده‌ای تغییر نکرد.' ], 500 );
			}
			$changes[] = [ 'case_id' => $case_id, 'old_status' => (string) $case->status, 'new_status' => $new_status, 'old_converter_id' => (int) $case->converter_id ];
		}
		$wpdb->query( 'COMMIT' );
		foreach ( $changes as $change ) {
			$this->log_case( (int) $change['case_id'], $user_id, 'converter_assigned_bulk', (string) $change['old_status'], (string) $change['new_status'], [ 'old_converter_id' => (int) $change['old_converter_id'], 'converter_id' => $converter_id, 'batch_count' => count( $changes ) ] );
			if ( class_exists( 'SN_Purpose_Commission' ) ) { SN_Purpose_Commission::reconcile_dot_case( (int) $change['case_id'] ); }
		}
		$converter = get_user_by( 'id', $converter_id );
		wp_send_json_success( [
			'message' => number_format_i18n( count( $changes ) ) . ( $converter_id === $user_id ? ' پرونده به خودتان تخصیص داده شد و در تب «تبدیل‌های من» قابل پیگیری است.' : ' پرونده بدون بارگذاری مجدد صفحه تخصیص داده شد.' ),
			'case_ids' => array_map( static fn( $change ) => (int) $change['case_id'], $changes ),
			'converter_id' => $converter_id,
			'converter_name' => $converter instanceof WP_User ? (string) $converter->display_name : ( 'کاربر #' . $converter_id ),
		] );
	}

	/** Return a fresh converter workspace after an Ajax mutation. */
	public function handle_refresh_converter_panel(): void {
		if ( ! is_user_logged_in() || ! check_ajax_referer( 'sn_public', 'nonce', false ) ) {
			wp_send_json_error( [ 'message' => 'نشست کاربری معتبر نیست؛ صفحه را تازه‌سازی کنید.' ], 403 );
		}
		$user_id = get_current_user_id();
		$context = sanitize_key( wp_unslash( $_POST['context'] ?? '' ) );
		$context = in_array( $context, [ 'seller', 'supervisor' ], true ) ? $context : '';
		$allowed = current_user_can( 'manage_options' )
			|| ( 'seller' === $context && $this->is_seller( $user_id ) )
			|| ( 'supervisor' === $context && $this->is_supervisor( $user_id ) )
			|| ( '' === $context && ( $this->is_converter( $user_id ) || $this->is_supervisor( $user_id ) ) );
		if ( ! $allowed ) {
			wp_send_json_error( [ 'message' => 'دسترسی به فهرست تبدیل مجاز نیست.' ], 403 );
		}
		wp_send_json_success( [ 'html' => $this->render_converter_panel( $context ) ] );
	}

	private function payment_rows( int $case_id ): array {
		$grouped = $this->payment_rows_for_cases( [ $case_id ] );
		return $grouped[ $case_id ] ?? [];
	}

	/**
	 * Customer-safe financial context for a synthetic Dot conversion invoice.
	 * The payable amount still belongs to the current invoice, while totals and
	 * progress belong to the whole conversion case.
	 */
	public function invoice_conversion_payment_context( int $invoice_id ): array {
		global $wpdb;
		$t = $this->tables();
		if ( $invoice_id < 1 || ! $this->table_exists( $t['links'] ) || ! $this->table_exists( $t['cases'] ) || ! $this->table_exists( $t['payments'] ) ) { return []; }
		$link = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['links']} WHERE invoice_id=%d AND flow_kind='conversion_payment' LIMIT 1", $invoice_id ) );
		if ( ! $link || (int) $link->case_id < 1 ) { return []; }
		$case = $this->get_case( (int) $link->case_id );
		if ( ! $case ) { return []; }
		$payments = array_reverse( $this->payment_rows( (int) $case->id ) );
		$summary = $this->case_payment_summary( $case );
		$stages = [];
		$current_stage = 1;
		$current_payment_type = sanitize_key( (string) $link->payment_kind );
		$status_map = [
			'pending' => 'pending',
			'pending_finance' => 'pending_financial_approval',
			'rejected' => 'rejected',
			'superseded' => 'expired',
			'paid' => 'paid',
		];
		$type_map = [ 'full' => 'full', 'deposit' => 'partial', 'balance' => 'remaining' ];
		foreach ( $payments as $index => $payment ) {
			$stage_no = $index + 1;
			$event_at = (string) ( $payment->paid_at ?: ( $payment->updated_at ?: $payment->created_at ) );
			$mapped_status = $status_map[ (string) $payment->status ] ?? (string) $payment->status;
			$stages[] = [
				'stage_no' => $stage_no,
				'stage_type' => $type_map[ (string) $payment->payment_type ] ?? 'partial',
				'stage_type_label' => $this->payment_type_label( (string) $payment->payment_type ),
				'requested_amount' => (float) $payment->requested_amount,
				'status' => $mapped_status,
				'status_label' => $this->payment_status_label( (string) $payment->status ),
				'pay_method' => (string) ( $payment->pay_method ?? '' ),
				'payment_source' => (string) ( $payment->payment_source ?? '' ),
				'display_date_jalali' => $this->jalali_datetime( $event_at ),
			];
		}
		// The customer must see the current state of the whole case, even when an
		// older deposit URL is reopened. The newest recorded payment is the active
		// stage; the invoice itself still controls whether a payment action is open.
		if ( $payments ) {
			$latest_payment = $payments[ count( $payments ) - 1 ];
			$current_stage = count( $payments );
			$current_payment_type = (string) $latest_payment->payment_type;
		}
		return [
			'is_dot_conversion_payment' => true,
			'case_id' => (int) $case->id,
			'option_title' => (string) $case->selected_option_title,
			'payment_type' => $current_payment_type,
			'payment_type_label' => $this->payment_type_label( $current_payment_type ),
			// Reissued/retried full-payment links are still one-time payments; only
			// deposit/balance modes make the customer flow genuinely staged.
			'payment_plan' => in_array( $current_payment_type, [ 'deposit', 'balance' ], true ) ? 'partial' : 'full',
			'payment_total_amount' => (float) $summary['total'],
			'paid_total_amount' => (float) $summary['paid'],
			'remaining_amount' => (float) $summary['remaining'],
			'progress_percent' => (int) $summary['percent'],
			'is_completed' => ! empty( $summary['completed'] ),
			'current_payment_stage' => $current_stage,
			'payment_stages' => $stages,
		];
	}


	private function converter_invoice_items_label( int $invoice_id, int $fallback_product_id = 0 ): string {
		global $wpdb;
		$item_table = $wpdb->prefix . 'sn_invoice_items';
		$parts = [];
		if ( $this->table_exists( $item_table ) ) {
			$items = $wpdb->get_results( $wpdb->prepare( "SELECT product_name,qty FROM {$item_table} WHERE invoice_id=%d ORDER BY id ASC", $invoice_id ), ARRAY_A ) ?: [];
			foreach ( $items as $item ) {
				$name = trim( (string) ( $item['product_name'] ?? '' ) );
				$qty = max( 1, (int) ( $item['qty'] ?? 1 ) );
				if ( $name !== '' ) { $parts[] = $qty > 1 ? $name . ' × ' . $qty : $name; }
			}
		}
		if ( ! $parts && $fallback_product_id > 0 ) {
			$title = get_the_title( $fallback_product_id );
			if ( is_string( $title ) && trim( $title ) !== '' ) { $parts[] = trim( $title ); }
		}
		return $parts ? implode( '، ', $parts ) : '—';
	}

	private function converter_invoice_effective_status( array $row ): string {
		if ( (string) ( $row['source_type'] ?? '' ) === 'conversion' ) {
			// Manual information/receipt submission updates the shared invoice and
			// finance-stage rows first. Prefer those review/final states so the
			// converter does not keep seeing a stale Dot "pending" label.
			foreach ( [ 'payment_status', 'invoice_status', 'status' ] as $invoice_key ) {
				$invoice_status = sanitize_key( (string) ( $row[ $invoice_key ] ?? '' ) );
				if ( in_array( $invoice_status, [ 'receipt_uploaded', 'pending_financial_approval', 'paid', 'approved', 'rejected', 'cancelled', 'payment_archived' ], true ) ) {
					return $invoice_status;
				}
			}
			$status = sanitize_key( (string) ( $row['dot_payment_status'] ?? '' ) );
			return [
				'pending' => 'pending',
				'pending_finance' => 'pending_financial_approval',
				'rejected' => 'rejected',
				'superseded' => 'expired',
				'paid' => 'paid',
			][ $status ] ?? $status;
		}
		foreach ( [ 'payment_status', 'invoice_status', 'status' ] as $key ) {
			$value = sanitize_key( (string) ( $row[ $key ] ?? '' ) );
			if ( $value !== '' ) { return $value; }
		}
		return 'pending';
	}

	private function converter_invoice_status_label( string $status ): string {
		$labels = [
			'pre_invoice' => 'پیش‌فاکتور', 'pending' => 'در انتظار پرداخت', 'pending_payment' => 'در انتظار پرداخت',
			'partial_paid' => 'پرداخت مرحله‌ای', 'receipt_uploaded' => 'رسید ارسال شده',
			'pending_financial_approval' => 'در انتظار تأیید مالی', 'paid' => 'پرداخت شده', 'approved' => 'تأیید شده',
			'rejected' => 'رد شده', 'expired' => 'منقضی / جایگزین شده', 'cancelled' => 'لغو شده',
			'recontact_requested' => 'نیازمند تماس مجدد',
		];
		return $labels[ $status ] ?? ( class_exists( 'SN_Helpers' ) ? SN_Helpers::status_label( $status ) : $status );
	}

	public function handle_converter_invoices(): void {
		if ( ! is_user_logged_in() || ! check_ajax_referer( 'sn_public', 'nonce', false ) ) {
			wp_send_json( [ 'success' => false, 'message' => 'دسترسی غیرمجاز است.' ], 403 );
		}
		$user_id = get_current_user_id();
		if ( ! current_user_can( 'manage_options' ) && ! $this->is_converter( $user_id ) && ! $this->is_supervisor( $user_id ) ) {
			wp_send_json( [ 'success' => false, 'message' => 'این بخش فقط برای تبدیل‌کننده یا سرپرست در دسترس است.' ], 403 );
		}
		global $wpdb;
		$t = $this->tables();
		$invoice_table = $wpdb->prefix . 'sn_invoices';
		if ( ! $this->table_exists( $invoice_table ) || ! $this->table_exists( $t['links'] ) || ! $this->table_exists( $t['cases'] ) || ! $this->table_exists( $t['payments'] ) ) {
			wp_send_json( [ 'success' => true, 'invoices' => [], 'summary' => [ 'all'=>0, 'pre_invoice'=>0, 'paid'=>0, 'rejected'=>0 ] ] );
		}
		$tab = sanitize_key( wp_unslash( $_POST['tab'] ?? 'all' ) );
		if ( ! in_array( $tab, [ 'all', 'pre_invoice', 'paid', 'rejected', 'staged_completed', 'staged_incomplete' ], true ) ) { $tab = 'all'; }
		$q = sanitize_text_field( wp_unslash( $_POST['q'] ?? '' ) );
		$limit = min( 100, max( 10, absint( $_POST['limit'] ?? 50 ) ) );
		$page = max( 1, absint( $_POST['page'] ?? 1 ) );
		$offset = ( $page - 1 ) * $limit;

		// Keep historical seller ownership untouched, but show each conversion
		// payment to both the current case converter and the user who issued it.
		// The latter keeps an invoice reachable after a later reassignment.
		$base_where = "((i.seller_id=%d AND dl.id IS NULL) OR (dl.id IS NOT NULL AND (c.converter_id=%d OR i.issued_by_user_id=%d)))";
		$args = [ $user_id, $user_id, $user_id ];
		$where = [ $base_where ];
		if ( $q !== '' ) {
			$like = '%' . $wpdb->esc_like( $q ) . '%';
			$where[] = '(i.invoice_code LIKE %s OR i.customer_name LIKE %s OR i.customer_phone LIKE %s OR c.selected_option_title LIKE %s OR CAST(c.id AS CHAR) LIKE %s)';
			array_push( $args, $like, $like, $like, $like, $like );
		}
		$status_sql = "CASE WHEN dl.id IS NOT NULL THEN COALESCE(dp.status,'pending') ELSE COALESCE(NULLIF(i.payment_status,''),NULLIF(i.invoice_status,''),NULLIF(i.status,''),'pending') END";
		if ( $tab === 'pre_invoice' ) {
			$where[] = "({$status_sql} IN ('pre_invoice','pending','pending_payment','partial_paid','receipt_uploaded','pending_finance','pending_financial_approval'))";
		} elseif ( $tab === 'paid' ) {
			$where[] = "({$status_sql} IN ('paid','approved'))";
		} elseif ( $tab === 'rejected' ) {
			$where[] = "({$status_sql} IN ('rejected','expired','superseded','cancelled'))";
		} elseif ( in_array( $tab, [ 'staged_completed', 'staged_incomplete' ], true ) ) {
			// A conversion can issue separate invoices for deposit and balance. Its
			// completion belongs to the case, rather than one paid stage invoice.
			$staged = "((dl.id IS NOT NULL AND dl.payment_kind IN ('deposit','balance')) OR (dl.id IS NULL AND (i.payment_plan='partial' OR i.current_payment_stage>1)))";
			$complete = "((dl.id IS NOT NULL AND COALESCE(c.remaining_amount,0)<=0.5 AND COALESCE(c.selected_option_price,0)>0 AND COALESCE(c.paid_amount,0)>=COALESCE(c.selected_option_price,0)-0.5) OR (dl.id IS NULL AND i.status IN ('paid','approved') AND i.payment_workflow_status='completed' AND COALESCE(i.remaining_amount,0)<=0.5))";
			$where[] = $staged . ' AND ' . ( $tab === 'staged_completed' ? $complete : 'NOT ' . $complete );
		}
		$sql_where = implode( ' AND ', array_map( static fn( $part ) => '(' . $part . ')', $where ) );
		$select = "SELECT i.*, dl.id AS dot_link_id, dl.case_id AS dot_case_id, dl.payment_kind AS dot_payment_kind, c.selected_option_title AS dot_option_title, c.selected_option_price, c.paid_amount AS case_paid_amount, c.status AS dot_case_status, dp.status AS dot_payment_status, dp.requested_amount AS dot_requested_amount, dp.paid_at AS dot_paid_at, CASE WHEN dl.id IS NOT NULL THEN 'conversion' ELSE 'manual' END AS source_type
			FROM {$invoice_table} i
			LEFT JOIN {$t['links']} dl ON dl.invoice_id=i.id AND dl.flow_kind='conversion_payment'
			LEFT JOIN {$t['cases']} c ON c.id=dl.case_id
			LEFT JOIN {$t['payments']} dp ON dp.invoice_id=i.id";
		$rows = $wpdb->get_results( $wpdb->prepare( "{$select} WHERE {$sql_where} ORDER BY i.id DESC LIMIT %d OFFSET %d", ...array_merge( $args, [ $limit, $offset ] ) ), ARRAY_A ) ?: [];
		$total = (int) $wpdb->get_var( $wpdb->prepare( "SELECT COUNT(*) FROM {$invoice_table} i LEFT JOIN {$t['links']} dl ON dl.invoice_id=i.id AND dl.flow_kind='conversion_payment' LEFT JOIN {$t['cases']} c ON c.id=dl.case_id LEFT JOIN {$t['payments']} dp ON dp.invoice_id=i.id WHERE {$sql_where}", ...$args ) );
		$project_results = ($rows && class_exists( 'SN_Projects' )) ? SN_Projects::instance()->project_summaries_for_invoices( array_column( $rows, 'id' ), $user_id ) : [];

		$out = [];
		$card_to_card_enabled = (string) get_option( 'sn_card_to_card_enabled', '1' ) === '1';
		$payment_action_statuses = [
			'pre_invoice', 'pending', 'pending_payment', 'awaiting_payment', 'partial_paid',
			'rejected', 'financial_rejected', 'payment_rejected', 'recontact_requested',
			'receipt_uploaded', 'pending_financial_approval', 'needs_finance_review',
		];
		foreach ( $rows as $row ) {
			$source_type = (string) ( $row['source_type'] ?? 'manual' );
			$status = $this->converter_invoice_effective_status( $row );
			$amount = $source_type === 'conversion' && (float) ( $row['dot_requested_amount'] ?? 0 ) > 0
				? (float) $row['dot_requested_amount'] : (float) ( $row['product_price'] ?? 0 );
			$product_label = $source_type === 'conversion'
				? ( (string) ( $row['dot_option_title'] ?? '' ) ?: $this->converter_invoice_items_label( (int) $row['id'], (int) ( $row['product_id'] ?? 0 ) ) )
				: $this->converter_invoice_items_label( (int) $row['id'], (int) ( $row['product_id'] ?? 0 ) );
			$source_label = $source_type === 'conversion' ? 'تبدیل دات‌فلو' : 'صدور دستی';
			if ( $source_type === 'conversion' ) {
				$kind = sanitize_key( (string) ( $row['dot_payment_kind'] ?? '' ) );
				$source_label .= ' · ' . ( [ 'full'=>'پرداخت کامل', 'deposit'=>'پرداخت مرحله‌ای', 'balance'=>'تسویه کامل مانده' ][ $kind ] ?? 'پرداخت' );
			}
			$manual_total = (float) ( ( $row['payment_total_amount'] ?? 0 ) ?: ( ( $row['final_total'] ?? 0 ) ?: ( $row['product_price'] ?? 0 ) ) );
			$manual_paid = max( 0, (float) ( $row['paid_total_amount'] ?? 0 ) );
			$manual_remaining = isset( $row['remaining_amount'] ) && $row['remaining_amount'] !== null && $row['remaining_amount'] !== '' ? max( 0, (float) $row['remaining_amount'] ) : max( 0, $manual_total - $manual_paid );
			$manual_due = max( 0, (float) ( $row['current_due_amount'] ?? 0 ) );
			$manual_workflow = sanitize_key( (string) ( $row['payment_workflow_status'] ?? '' ) );
			$is_conversion = $source_type === 'conversion';
			$case_total = max( 0, (float) ( $row['selected_option_price'] ?? 0 ) );
			$display_total = $is_conversion ? max( $case_total, $amount ) : $manual_total;
			$display_paid = $is_conversion ? min( $display_total, max( 0, (float) ( $row['case_paid_amount'] ?? 0 ) ) ) : min( $manual_total, $manual_paid );
			$display_remaining = max( 0, $display_total - $display_paid );
			$can_issue_next_stage = $source_type === 'manual' && (int) ( $row['seller_id'] ?? 0 ) === $user_id && $manual_remaining > 0.5 && $manual_due <= 0.5 && $manual_workflow === 'awaiting_assignment';
			$project = $project_results[(int) $row['id']] ?? null;
			$can_submit_payment = $card_to_card_enabled && in_array( $status, $payment_action_statuses, true );
			$out[] = [
				'id' => (int) $row['id'],
				'invoice_code' => (string) ( $row['invoice_code'] ?? '' ),
				'customer_name' => (string) ( $row['customer_name'] ?? '' ),
				'customer_phone' => (string) ( $row['customer_phone'] ?? '' ),
				'product_name' => $product_label,
				'amount' => $amount,
				'amount_fmt' => SN_Helpers::format_price( $amount ),
				'status' => $status,
				'status_label' => $this->converter_invoice_status_label( $status ),
				'source_type' => $source_type,
				'source_label' => $source_label,
				'case_id' => (int) ( $row['dot_case_id'] ?? 0 ),
				'created_at' => (string) ( $row['created_at'] ?? '' ),
				'created_at_label' => $this->jalali_datetime( (string) ( $row['created_at'] ?? '' ) ),
				'rejected_reason' => (string) ( $row['financial_reject_reason'] ?? ( $row['rejected_reason'] ?? '' ) ),
				'payment_total_amount' => $display_total,
				'paid_total_amount' => $display_paid,
				'remaining_amount' => $display_remaining,
				'current_due_amount' => $manual_due,
				'current_payment_stage' => max( 1, (int) ( $row['current_payment_stage'] ?? 1 ) ),
				'payment_workflow_status' => $manual_workflow,
				'can_issue_next_stage' => $can_issue_next_stage,
				'can_submit_manual_payment' => $can_submit_payment,
				'can_upload_receipt' => $can_submit_payment,
				'receipt_url' => (string) ( $row['receipt_url'] ?? ( $row['receipt_file'] ?? '' ) ),
				'project' => $project,
			];
		}

		// Lightweight summaries use the already-scoped ownership relation and do not run seller-scope services.
		$summary_rows = $wpdb->get_results( $wpdb->prepare(
			"SELECT {$status_sql} AS effective_status, COUNT(*) cnt FROM {$invoice_table} i LEFT JOIN {$t['links']} dl ON dl.invoice_id=i.id AND dl.flow_kind='conversion_payment' LEFT JOIN {$t['cases']} c ON c.id=dl.case_id LEFT JOIN {$t['payments']} dp ON dp.invoice_id=i.id WHERE {$base_where} GROUP BY effective_status",
			$user_id, $user_id, $user_id
		), ARRAY_A ) ?: [];
		$summary = [ 'all'=>0, 'pre_invoice'=>0, 'paid'=>0, 'rejected'=>0, 'staged_completed'=>0, 'staged_incomplete'=>0 ];
		foreach ( $summary_rows as $sum ) {
			$count = (int) ( $sum['cnt'] ?? 0 ); $summary['all'] += $count;
			$st = sanitize_key( (string) ( $sum['effective_status'] ?? '' ) );
			if ( in_array( $st, [ 'paid','approved' ], true ) ) { $summary['paid'] += $count; }
			elseif ( in_array( $st, [ 'rejected','expired','superseded','cancelled' ], true ) ) { $summary['rejected'] += $count; }
			else { $summary['pre_invoice'] += $count; }
		}
		$stage_counts = $wpdb->get_row( $wpdb->prepare(
			"SELECT COUNT(DISTINCT CASE WHEN ((dl.id IS NOT NULL AND dl.payment_kind IN ('deposit','balance')) OR (dl.id IS NULL AND (i.payment_plan='partial' OR i.current_payment_stage>1))) AND ((dl.id IS NOT NULL AND COALESCE(c.selected_option_price,0)>0 AND COALESCE(c.paid_amount,0)>=c.selected_option_price-0.5) OR (dl.id IS NULL AND i.status IN ('paid','approved') AND i.payment_workflow_status='completed' AND COALESCE(i.remaining_amount,0)<=0.5)) THEN i.id END) completed,
			COUNT(DISTINCT CASE WHEN ((dl.id IS NOT NULL AND dl.payment_kind IN ('deposit','balance')) OR (dl.id IS NULL AND (i.payment_plan='partial' OR i.current_payment_stage>1))) AND NOT ((dl.id IS NOT NULL AND COALESCE(c.selected_option_price,0)>0 AND COALESCE(c.paid_amount,0)>=c.selected_option_price-0.5) OR (dl.id IS NULL AND i.status IN ('paid','approved') AND i.payment_workflow_status='completed' AND COALESCE(i.remaining_amount,0)<=0.5)) THEN i.id END) incomplete
			FROM {$invoice_table} i LEFT JOIN {$t['links']} dl ON dl.invoice_id=i.id AND dl.flow_kind='conversion_payment' LEFT JOIN {$t['cases']} c ON c.id=dl.case_id WHERE {$base_where}",
			$user_id, $user_id, $user_id
		), ARRAY_A );
		$summary['staged_completed'] = (int) ( $stage_counts['completed'] ?? 0 );
		$summary['staged_incomplete'] = (int) ( $stage_counts['incomplete'] ?? 0 );
		wp_send_json( [ 'success'=>true, 'invoices'=>$out, 'items'=>$out, 'page'=>$page, 'limit'=>$limit, 'total'=>$total, 'summary'=>$summary ] );
	}

	public function render_converter_panel( string $context = '' ): string {
		if ( ! is_user_logged_in() ) {
			$login_id = (int) get_option( 'sn_login_page_id', 0 );
			$url = $login_id ? get_permalink( $login_id ) : wp_login_url();
			return '<script>window.location.href=' . wp_json_encode( $url ) . ';</script><div class="sn-notice">در حال انتقال به صفحه ورود...</div>';
		}
		$user_id = get_current_user_id();
		$is_converter_user = $this->is_converter( $user_id );
		$is_seller_user = $this->is_seller( $user_id );
		$is_supervisor = $this->is_supervisor( $user_id );
		$context = sanitize_key( $context );
		$context = in_array( $context, [ 'seller', 'supervisor' ], true ) ? $context : '';
		$is_embedded = $context !== '';
		$has_context_access = ( 'seller' === $context && $is_seller_user ) || ( 'supervisor' === $context && $is_supervisor );
		if ( ! current_user_can( 'manage_options' ) && ( $is_embedded ? ! $has_context_access : ( ! $is_converter_user && ! $is_supervisor ) ) ) {
			return '<div class="sn-notice sn-error" dir="rtl">دسترسی به این فهرست تبدیل مجاز نیست.</div>';
		}
		global $wpdb; $t = $this->tables();
		if ( 'seller' === $context ) {
			// A finance callback can be missed by a request failure. Repair only this
			// seller's fully-paid seller_self links before reading their own queue.
			$this->repair_paid_assessment_cases_for_seller( $user_id, 20 );
		}
		// The converter queue is ordered by the moment a case entered the
		// converter's responsibility, not by later customer/payment activity.
		// The id tie-breaker keeps rows deterministic when assignments share a
		// second-level timestamp.
		$order_by = "COALESCE(assigned_at, created_at) DESC, id DESC";
		if ( 'seller' === $context ) {
			$sql = $wpdb->prepare( "SELECT * FROM {$t['cases']} WHERE seller_id=%d AND converter_id=%d AND conversion_route='seller_self' AND operational_archived_at IS NULL ORDER BY {$order_by} LIMIT 300", $user_id, $user_id );
		} elseif ( 'supervisor' === $context ) {
			$sql = $wpdb->prepare( "SELECT * FROM {$t['cases']} WHERE supervisor_id=%d AND converter_id=%d AND (conversion_route IS NULL OR conversion_route<>'seller_self') AND operational_archived_at IS NULL ORDER BY {$order_by} LIMIT 300", $user_id, $user_id );
		} else {
			$sql = current_user_can( 'manage_options' )
				? "SELECT * FROM {$t['cases']} WHERE converter_id IS NOT NULL AND operational_archived_at IS NULL ORDER BY {$order_by} LIMIT 300"
				: ( $is_supervisor && ! $is_converter_user
					? $wpdb->prepare( "SELECT * FROM {$t['cases']} WHERE converter_id=%d AND supervisor_id=%d AND operational_archived_at IS NULL ORDER BY {$order_by} LIMIT 300", $user_id, $user_id )
					: $wpdb->prepare( "SELECT * FROM {$t['cases']} WHERE converter_id=%d AND operational_archived_at IS NULL ORDER BY {$order_by} LIMIT 300", $user_id ) );
		}
		$cases = $wpdb->get_results( $sql ) ?: [];
		$source_invoice_codes = [];
		$source_invoice_ids = array_values( array_unique( array_filter( array_map( static fn( $case ) => absint( $case->source_invoice_id ?? 0 ), $cases ) ) ) );
		if ( $source_invoice_ids ) {
			$source_placeholders = implode( ',', array_fill( 0, count( $source_invoice_ids ), '%d' ) );
			foreach ( $wpdb->get_results( $wpdb->prepare( "SELECT id,invoice_code FROM {$wpdb->prefix}sn_invoices WHERE id IN ({$source_placeholders})", ...$source_invoice_ids ) ) ?: [] as $source_invoice ) {
				$source_invoice_codes[ (int) $source_invoice->id ] = (string) $source_invoice->invoice_code;
			}
		}
		$payments_by_case = $this->payment_rows_for_cases( array_map( static fn( $case ) => (int) $case->id, $cases ) );
		foreach ( $cases as $index => $case ) {
			$cases[ $index ] = $this->sync_live_case_option_prices( $case, ! empty( $payments_by_case[ (int) $case->id ] ) );
		}
		$notice = wp_doing_ajax() ? '' : sanitize_text_field( wp_unslash( $_GET['sn_dot_message'] ?? '' ) );
		$status_filters = [ 'assigned', 'contacted', 'customer_declined', 'payment_link_sent', 'payment_rejected', 'deposit_paid', 'completed' ];
		$contact_filters = [ 'new', 'no_answer', 'contacted', 'follow_up', 'confirmed', 'customer_declined', 'payment_link' ];
		$completed_count = count( array_filter( $cases, static fn( $c ) => (string) $c->status === 'completed' ) );
		$settlement_count = count( array_filter( $cases, static fn( $c ) => (string) $c->status === 'deposit_paid' ) );
		$follow_up_count = count( array_filter( $cases, static fn( $c ) => in_array( (string) ( $c->converter_contact_status ?: 'new' ), [ 'new', 'no_answer', 'follow_up' ], true ) && ! in_array( (string) $c->status, [ 'completed', 'customer_declined' ], true ) ) );
		$due_followup_count = count( array_filter( $cases, fn( $c ) => (string) ( $c->converter_contact_status ?? '' ) === 'follow_up' && ! empty( $c->converter_followup_at ) && $this->mysql_timestamp( (string) $c->converter_followup_at ) <= time() && ! in_array( (string) $c->status, [ 'completed', 'customer_declined' ], true ) ) );

		// Seller-like workspace context for optional manual invoice issuance.
		$can_issue_invoice = false; // CRM-30: no standalone pre-invoice tab for converters.
		$products = [];
		$provinces = [];
		if ( $can_issue_invoice ) {
			$products = SN_Helpers::get_sn_products();
			$products = $this->filter_products_for_user( $products, $user_id );
			if ( class_exists( 'SN_Projects' ) ) { $products = SN_Projects::instance()->filter_sellable_products( $products, $user_id ); }
			if ( class_exists( 'SN_Seller_Flow' ) ) { $products = SN_Seller_Flow::instance()->decorate_products( $products ); }
			$provinces = SN_Helpers::get_provinces();
		}
		$max_invoice_products = max( 0, (int) get_option( 'sn_invoice_max_products', '2' ) );
		$payment_presets_raw = (string) get_option( 'sn_partial_payment_presets', '1000000,2000000,3000000,5000000' );
		$payment_presets = [];
		foreach ( preg_split( '/[,،;\n\r]+/u', $payment_presets_raw ) ?: [] as $preset_raw ) {
			$preset = (float) preg_replace( '/[^0-9.]/', '', strtr( trim( (string) $preset_raw ), [ '۰'=>'0','۱'=>'1','۲'=>'2','۳'=>'3','۴'=>'4','۵'=>'5','۶'=>'6','۷'=>'7','۸'=>'8','۹'=>'9','٠'=>'0','١'=>'1','٢'=>'2','٣'=>'3','٤'=>'4','٥'=>'5','٦'=>'6','٧'=>'7','٨'=>'8','٩'=>'9' ] ) );
			if ( $preset > 0 ) { $payment_presets[ (string) (int) round( $preset ) ] = $preset; }
		}
		$payment_presets = array_values( $payment_presets );
		sort( $payment_presets, SORT_NUMERIC );
		if ( ! $payment_presets ) { $payment_presets = [ 1000000, 2000000, 3000000, 5000000 ]; }

		$current_user = wp_get_current_user();
		$display_name = trim( (string) $current_user->display_name ) ?: (string) $current_user->user_login;
		ob_start(); ?>
		<div class="sn-panel sn-portal sn-dot-converter sn-converter-seller-ui<?php echo $is_embedded ? ' sn-dot-converter-embedded' : ''; ?>" dir="rtl" id="sn-dot-converter-panel" data-sn-context="<?php echo esc_attr( $context ); ?>" data-sn-max-products="<?php echo esc_attr( (string) $max_invoice_products ); ?>" data-sn-can-issue-invoice="<?php echo $can_issue_invoice ? '1' : '0'; ?>">
			<?php if ( ! $is_embedded ) : ?>
			<div class="sn-panel-header">
				<h2><?php echo $is_supervisor && ! $is_converter_user ? 'پنل تبدیل‌کننده (تخصیص‌های خودم)' : 'پنل تبدیل‌کننده'; ?> — <?php echo esc_html( $display_name ); ?></h2>
				<div class="sn-panel-user-actions">
					<div class="sn-panel-user-chip"><span>کاربر</span><strong><?php echo esc_html( $display_name ); ?></strong><small><?php echo $is_supervisor && ! $is_converter_user ? 'سرپرست — در نقش تبدیل‌کننده' : 'تبدیل‌کننده'; ?> | #<?php echo esc_html( (string) $user_id ); ?></small></div>
					<form method="post" action="<?php echo esc_url( admin_url( 'admin-post.php' ) ); ?>" class="sn-panel-logout-form"><input type="hidden" name="action" value="sn_seller_logout"><button type="submit" class="sn-btn sn-btn-sm sn-btn-secondary">خروج</button></form>
				</div>
			</div>

			<div class="sn-tabs sn-converter-main-tabs">
				<button type="button" class="sn-tab active" data-tab="converter-cases">پرونده‌های من <span class="sn-tab-badge is-visible"><?php echo esc_html( number_format_i18n( count( $cases ) ) ); ?></span></button>
				<?php if ( $can_issue_invoice ) : ?><button type="button" class="sn-tab" data-tab="new-invoice">صدور پیش‌فاکتور</button><?php endif; ?>
				<button type="button" class="sn-tab" data-tab="invoices">فاکتورهای من</button>
				<button type="button" class="sn-tab" data-tab="customer-actions">رفتار مشتریان</button>
				<button type="button" class="sn-tab" data-tab="wallet">کیف پول من</button>
			</div>
			<?php endif; ?>

			<div id="sn-tab-converter-cases" class="sn-tab-content active">
				<?php if ( $is_embedded ) : ?>
				<div class="sn-card sn-dot-embedded-heading"><div><span>دات‌فلو</span><h3><?php echo 'seller' === $context ? 'آماده‌های تبدیل من' : 'تبدیل‌های من'; ?></h3><p><?php echo 'seller' === $context ? 'فقط پرونده‌هایی که هنگام ثبت فاکتور «تبدیل توسط خودم» را انتخاب کرده‌اید نمایش داده می‌شوند.' : 'فقط پرونده‌هایی که با «تخصیص به خودم» برداشته‌اید نمایش داده می‌شوند.'; ?></p></div></div>
				<?php endif; ?>
				<?php if ( $notice !== '' ) : ?><div class="sn-notice <?php echo sanitize_key( $_GET['sn_dot_result'] ?? '' ) === 'success' ? 'sn-success' : 'sn-error'; ?>"><?php echo esc_html( $notice ); ?></div><?php endif; ?>
				<div class="sn-converter-simple-summary" aria-label="خلاصه پرونده‌ها">
					<span><strong><?php echo esc_html( number_format_i18n( count( $cases ) ) ); ?></strong> کل پرونده</span>
					<span><strong><?php echo esc_html( number_format_i18n( $follow_up_count ) ); ?></strong> نیازمند پیگیری</span>
					<span class="<?php echo $due_followup_count > 0 ? 'is-overdue' : ''; ?>"><strong><?php echo esc_html( number_format_i18n( $due_followup_count ) ); ?></strong> تماس سررسیدشده</span>
					<span><strong><?php echo esc_html( number_format_i18n( $settlement_count ) ); ?></strong> دارای مانده</span>
					<span><strong><?php echo esc_html( number_format_i18n( $completed_count ) ); ?></strong> تکمیل‌شده</span>
				</div>
				<?php if ( $cases ) : ?>
				<div class="sn-converter-case-toolbar sn-converter-case-toolbar-simple">
					<div class="sn-converter-toolbar-title"><strong>پرونده‌های من</strong><span><strong class="sn-dot-filter-result-count"><?php echo esc_html( number_format_i18n( count( $cases ) ) ); ?></strong> مورد</span></div>
					<div class="sn-converter-toolbar-fields sn-converter-toolbar-fields-simple">
						<label class="sn-converter-search"><span>جست‌وجو</span><input type="search" data-sn-dot-filter="search" placeholder="نام، موبایل، فاکتور یا شماره پرونده" autocomplete="off"></label>
						<label><span>مرحله پرونده</span><select data-sn-dot-filter="status"><option value="">همه مرحله‌ها</option><?php foreach ( $status_filters as $status_filter ) : ?><option value="<?php echo esc_attr( $status_filter ); ?>"><?php echo esc_html( $this->status_label( $status_filter ) ); ?></option><?php endforeach; ?></select></label>
						<label><span>نتیجه تماس</span><select data-sn-dot-filter="contact"><option value="">همه نتایج</option><?php foreach ( $contact_filters as $contact_filter ) : ?><option value="<?php echo esc_attr( $contact_filter ); ?>"><?php echo esc_html( $this->contact_status_label( $contact_filter ) ); ?></option><?php endforeach; ?></select></label>
						<button type="button" class="sn-btn sn-btn-secondary sn-dot-filter-reset">پاک کردن فیلتر</button>
					</div>
				</div>
				<?php else : ?><div class="sn-card sn-empty-state"><?php echo 'seller' === $context ? 'هنوز فاکتور اعتبارسنجی آماده‌ای با مسیر «تبدیل توسط خودم» ندارید.' : ( 'supervisor' === $context ? 'هنوز پرونده‌ای را به خودتان تخصیص نداده‌اید.' : 'هنوز پرونده‌ای به شما تخصیص داده نشده است.' ); ?></div><?php endif; ?>

				<div class="sn-dot-converter-case-list sn-converter-case-list-simple">
				<?php foreach ( $cases as $case ) :
					$options = json_decode( (string) $case->options_snapshot_json, true ); $options = is_array( $options ) ? $options : [];
					$payments = $payments_by_case[ (int) $case->id ] ?? []; $has_payment = ! empty( $payments );
					$payment_summary = $this->case_payment_summary( $case ); $remaining = (float) $payment_summary['remaining'];
					$contact_status = sanitize_key( (string) ( $case->converter_contact_status ?: 'new' ) );
					$has_converter_action = $contact_status !== 'new' || trim( (string) ( $case->converter_note ?? '' ) ) !== '' || ! in_array( (string) $case->status, [ 'assigned', 'ready_for_conversion' ], true );
					$content_id = 'sn-dot-case-content-' . (int) $case->id;
					$location = trim( trim( (string) $case->province ) . ( trim( (string) $case->city ) !== '' ? '، ' . trim( (string) $case->city ) : '' ), '، ' );
					$search_text = implode( ' ', [ (string) $case->id, (string) $case->customer_name, (string) $case->customer_phone, (string) $case->province, (string) $case->city, (string) $case->selected_option_title ] );
					$entered_timestamp = $this->mysql_timestamp( (string) ( $case->assigned_at ?: $case->created_at ?: $case->updated_at ) );
					$open_payments = array_filter( $payments, static fn( $p ) => in_array( (string) $p->status, [ 'pending', 'pending_finance' ], true ) );
					$source_invoice_code = (string) ( $source_invoice_codes[ (int) $case->source_invoice_id ] ?? '' );
					$source_invoice_label = $source_invoice_code !== '' ? $source_invoice_code : ( '#' . (int) $case->source_invoice_id );
					$search_text .= ' ' . $source_invoice_label;
					$followup_at = trim( (string) ( $case->converter_followup_at ?? '' ) );
					$followup_due = $contact_status === 'follow_up' && $followup_at !== '' && $this->mysql_timestamp( $followup_at ) <= time();
					$followup_date_input = $followup_at !== '' ? SN_Helpers::gregorian_to_jalali_input_value( $followup_at ) : '';
					$followup_timestamp = SN_Helpers::site_mysql_timestamp( $followup_at, 0 );
					$followup_time_input = $followup_timestamp > 0 ? SN_Helpers::tehran_format( 'H:i', $followup_timestamp ) : '';
					$no_answer_attempts = max( 0, (int) ( $case->converter_no_answer_attempts ?? 0 ) );
				?>
					<article class="sn-dot-converter-case sn-converter-case-simple<?php echo $has_converter_action ? '' : ' is-no-action'; ?><?php echo $followup_due ? ' is-followup-due' : ''; ?>" id="dot-case-<?php echo esc_attr( (string) $case->id ); ?>" data-sn-dot-case data-search="<?php echo esc_attr( $search_text ); ?>" data-status="<?php echo esc_attr( (string) $case->status ); ?>" data-contact="<?php echo esc_attr( $contact_status ); ?>" data-no-action="<?php echo $has_converter_action ? '0' : '1'; ?>" data-updated="<?php echo esc_attr( (string) $entered_timestamp ); ?>" data-progress="<?php echo esc_attr( (string) $payment_summary['percent'] ); ?>">
						<button type="button" class="sn-dot-case-toggle sn-converter-case-simple-row" aria-expanded="false" aria-controls="<?php echo esc_attr( $content_id ); ?>">
							<span class="sn-converter-simple-person"><span class="sn-converter-case-id">پرونده #<?php echo esc_html( (string) $case->id ); ?><small>فاکتور <bdi dir="ltr"><?php echo esc_html( $source_invoice_label ); ?></bdi></small></span><span><strong><?php echo esc_html( (string) $case->customer_name ); ?></strong><small><bdi dir="ltr"><?php echo esc_html( (string) $case->customer_phone ); ?></bdi><?php echo $location !== '' ? ' · ' . esc_html( $location ) : ''; ?></small></span></span>
							<span class="sn-converter-simple-option"><small>گزینه</small><strong><?php echo esc_html( (string) ( $case->selected_option_title ?: 'انتخاب نشده' ) ); ?></strong></span>
							<span class="sn-converter-simple-payment"><small><?php echo $payment_summary['completed'] ? 'وضعیت پرداخت' : 'مانده'; ?></small><strong><?php echo $payment_summary['completed'] ? 'تسویه شده' : esc_html( number_format_i18n( $remaining ) . ' تومان' ); ?></strong></span>
							<span class="sn-converter-simple-status"><span class="sn-badge"><?php echo esc_html( $this->contact_status_label( $contact_status ) ); ?><?php echo $contact_status === 'no_answer' ? ' · ' . esc_html( number_format_i18n( $no_answer_attempts ) ) . ' بار' : ''; ?></span><?php if ( $followup_due ) : ?><span class="sn-dot-followup-due-tag">یادآور تماس</span><?php elseif ( $contact_status === 'follow_up' && $followup_at !== '' ) : ?><small><?php echo esc_html( $this->jalali_datetime( $followup_at ) ); ?></small><?php endif; ?><?php if ( ! empty( $case->validation_fee_gifted ) ) : ?><span class="sn-dot-validation-gift-tag sn-converter-gift-tag">اعتبارسنجی هدیه</span><?php endif; ?></span>
							<span class="sn-converter-simple-open">باز کردن پرونده <i aria-hidden="true">⌄</i></span>
						</button>
						<div class="sn-dot-case-content sn-converter-case-simple-content" id="<?php echo esc_attr( $content_id ); ?>" hidden>
							<div class="sn-converter-simple-details">
								<div class="sn-converter-simple-facts">
									<div><span>مبلغ گزینه</span><strong><?php echo esc_html( number_format_i18n( (float) $case->selected_option_price ) ); ?> تومان</strong></div>
									<div><span>پرداخت‌شده</span><strong><?php echo esc_html( number_format_i18n( (float) $payment_summary['paid'] ) ); ?> تومان</strong></div>
									<div><span>مانده</span><strong><?php echo esc_html( number_format_i18n( $remaining ) ); ?> تومان</strong></div>
									<div><span>آخرین فعالیت</span><strong><?php echo esc_html( $this->jalali_datetime( (string) $case->updated_at ) ); ?></strong></div>
									<div><span>تماس مجدد</span><strong><?php echo esc_html( $followup_at !== '' ? $this->jalali_datetime( $followup_at ) : '—' ); ?></strong></div>
								</div>
								<?php if ( $followup_due ) : ?><div class="sn-notice sn-error sn-dot-followup-reminder"><strong>یادآور:</strong> زمان تماس مجدد این مشتری رسیده است.</div><?php endif; ?>

								<section class="sn-converter-simple-step">
									<div class="sn-converter-simple-step-head"><span>۱</span><div><strong>نتیجه تماس با مشتری</strong><small>جواب نداده، تماس مجدد، انصراف یا ادامه به ارسال لینک پرداخت را مشخص کنید.</small></div></div>
									<form class="sn-dot-action-form sn-dot-converter-update" method="post" action="<?php echo esc_url( admin_url( 'admin-post.php' ) ); ?>">
										<input type="hidden" name="action" value="sn_dot_converter_update"><input type="hidden" name="case_id" value="<?php echo esc_attr( (string) $case->id ); ?>"><?php wp_nonce_field( 'sn_dot_converter_update_' . (int) $case->id ); ?>
										<div class="sn-converter-simple-followup-grid">
											<label><span>نتیجه تماس *</span><select name="contact_status" required><option value="" <?php selected( in_array( $contact_status, [ 'new', 'contacted', 'confirmed' ], true ) ); ?>>انتخاب نتیجه</option><option value="no_answer" <?php selected( $contact_status, 'no_answer' ); ?>>جواب نداده (<?php echo esc_html( number_format_i18n( $no_answer_attempts ) ); ?> بار ثبت‌شده)</option><option value="follow_up" <?php selected( $contact_status, 'follow_up' ); ?>>تماس مجدد</option><option value="customer_declined" <?php selected( $contact_status, 'customer_declined' ); ?>>انصراف</option><option value="payment_link" <?php selected( $contact_status, 'payment_link' ); ?>>ارسال لینک پرداخت</option></select></label>
											<label class="sn-dot-followup-at-field" data-sn-dot-status-field="follow_up" hidden><span>تاریخ شمسی و ساعت تهران *</span><span class="sn-dot-followup-date-time"><input type="text" class="sn-jalali-date" name="followup_date" value="<?php echo esc_attr( $followup_date_input ); ?>" placeholder="۱۴۰۵/۰۶/۲۰" autocomplete="off" aria-label="تاریخ شمسی تماس مجدد"><input type="time" name="followup_time" value="<?php echo esc_attr( $followup_time_input ); ?>" step="300" aria-label="ساعت تهران تماس مجدد"></span><input type="hidden" name="followup_at" value=""><small>فقط پس از انتخاب کامل تاریخ شمسی و ساعت تهران، ثبت تماس مجدد فعال می‌شود و یادآور پنل و پیامک مسئول فعلی تنظیم خواهد شد.</small></label>
											<?php if ( ! $has_payment ) : ?><label><span>محصول نهایی</span><select name="option_key"><option value="">هنوز انتخاب نشده</option><?php foreach ( $options as $option ) : if ( empty( $option['active'] ) ) { continue; } ?><option value="<?php echo esc_attr( (string) $option['key'] ); ?>" <?php selected( (string) $case->selected_option_key, (string) $option['key'] ); ?>><?php echo esc_html( (string) $option['title'] . ' — ' . number_format_i18n( (float) $option['price'] ) . ' تومان' ); ?></option><?php endforeach; ?></select></label><?php else : ?><input type="hidden" name="option_key" value="<?php echo esc_attr( (string) $case->selected_option_key ); ?>"><?php endif; ?>
											<label class="sn-converter-simple-note"><span>یادداشت</span><textarea name="note" rows="2" placeholder="مثلاً: مشتری فردا تماس می‌گیرد"><?php echo esc_textarea( (string) $case->converter_note ); ?></textarea></label>
											<label class="sn-converter-simple-note sn-dot-decline-reason-field" data-sn-dot-status-field="customer_declined"><span>علت انصراف *</span><textarea name="decline_reason" rows="2" placeholder="علت دقیق انصراف مشتری را ثبت کنید"><?php echo esc_textarea( (string) ( $case->converter_decline_reason ?? '' ) ); ?></textarea></label>
										</div>
										<button class="sn-btn sn-btn-primary sn-converter-simple-save" type="submit">ذخیره نتیجه تماس</button>
									</form>
								</section>

								<section class="sn-converter-simple-step sn-dot-payment-step" data-sn-dot-payment-step data-has-payment="<?php echo $has_payment ? '1' : '0'; ?>"<?php echo $has_payment || $contact_status === 'payment_link' ? '' : ' hidden'; ?>>
									<div class="sn-converter-simple-step-head"><span>۲</span><div><strong>دریافت وجه</strong><small><?php echo $payment_summary['completed'] ? 'این پرونده تسویه شده است.' : 'مبلغ مرحله بعد را مشخص کنید و لینک را برای مشتری بفرستید.'; ?></small></div></div>
									<div class="sn-converter-simple-pay-status"><span>پرداخت‌شده <strong><?php echo esc_html( number_format_i18n( (float) $payment_summary['paid'] ) ); ?> تومان</strong></span><span>مانده <strong><?php echo esc_html( number_format_i18n( $remaining ) ); ?> تومان</strong></span><span><?php echo esc_html( number_format_i18n( $payment_summary['percent'] ) ); ?>٪</span></div>
									<?php if ( ! in_array( (string) $case->status, [ 'completed', 'customer_declined' ], true ) ) : ?>
										<?php if ( $open_payments ) : ?><div class="sn-notice sn-info sn-converter-simple-open-payment">یک لینک پرداخت فعال است. ابتدا همان پرداخت را پیگیری کنید؛ بعد از تعیین تکلیف، مرحله بعد باز می‌شود.</div><?php endif; ?>
										<?php if ( ! $open_payments && $remaining > 0 ) : ?>
										<div class="sn-converter-simple-payment-actions">
											<form class="sn-dot-action-form sn-dot-deposit-form sn-converter-simple-stage-form" method="post" action="<?php echo esc_url( admin_url( 'admin-post.php' ) ); ?>">
												<input type="hidden" name="action" value="sn_dot_create_payment"><input type="hidden" name="case_id" value="<?php echo esc_attr( (string) $case->id ); ?>"><input type="hidden" name="payment_type" value="deposit"><?php wp_nonce_field( 'sn_dot_create_payment_' . (int) $case->id ); ?>
												<label><span><?php echo (float) $case->paid_amount > 0 ? 'مبلغ مرحله بعد' : 'مبلغ مرحله اول'; ?></span><input name="deposit_amount" inputmode="numeric" required placeholder="مثلاً ۲۰,۰۰۰,۰۰۰"></label>
												<button class="sn-btn sn-btn-primary" type="submit">ارسال لینک این مرحله</button>
											</form>
											<form class="sn-dot-action-form sn-converter-simple-full-form" method="post" action="<?php echo esc_url( admin_url( 'admin-post.php' ) ); ?>"><input type="hidden" name="action" value="sn_dot_create_payment"><input type="hidden" name="case_id" value="<?php echo esc_attr( (string) $case->id ); ?>"><input type="hidden" name="payment_type" value="<?php echo (float) $case->paid_amount > 0 ? 'balance' : 'full'; ?>"><?php wp_nonce_field( 'sn_dot_create_payment_' . (int) $case->id ); ?><button class="sn-btn sn-btn-secondary" type="submit">تسویه کامل مانده — <?php echo esc_html( number_format_i18n( $remaining ) ); ?> تومان</button></form>
										</div>
										<?php endif; ?>
									<?php endif; ?>
									<?php if ( $payments ) : ?><details class="sn-converter-simple-history"><summary>مشاهده سوابق و لینک‌های پرداخت</summary><div><?php $this->render_case_payment_overview( $case, $payments, true ); ?></div></details><?php endif; ?>
								</section>
							</div>
						</div>
					</article>
				<?php endforeach; ?>
					<div class="sn-card sn-empty-state sn-dot-filter-empty" hidden>پرونده‌ای با این فیلتر پیدا نشد.</div>
				</div>
			</div>

			<?php if ( ! $is_embedded ) : ?>
			<?php if ( $can_issue_invoice ) : ?>
			<div id="sn-tab-new-invoice" class="sn-tab-content">
				<div class="sn-card sn-converter-invoice-card">
					<h3>ثبت فاکتور جدید</h3>
					<p class="sn-note">این فاکتور مستقل از پرونده‌های دات‌فلو صادر می‌شود و در تب «فاکتورهای من» خودتان نمایش داده خواهد شد.</p>
					<div id="sn-converter-invoice-notice"></div>
					<div class="sn-form-grid">
						<div class="sn-field"><label>نام مشتری *</label><input type="text" id="sn-converter-cust-name" placeholder="نام و نام خانوادگی"></div>
						<div class="sn-field"><label>شماره موبایل مشتری *</label><input type="tel" id="sn-converter-cust-phone" inputmode="tel" placeholder="09xxxxxxxxx"></div>
						<div class="sn-field"><label>شماره دوم مشتری (اختیاری؛ فقط اطلاعات تماس)</label><input type="tel" id="sn-converter-cust-phone-secondary" inputmode="tel" placeholder="09xxxxxxxxx"></div>
						<div class="sn-field"><label>استان</label><select id="sn-converter-cust-prov"><option value="">انتخاب استان</option><?php foreach ( $provinces as $province ) : ?><option value="<?php echo esc_attr( $province ); ?>"><?php echo esc_html( $province ); ?></option><?php endforeach; ?></select></div>
						<div class="sn-field"><label>شهر</label><input type="text" id="sn-converter-cust-city" placeholder="نام شهر"></div>
						<div class="sn-field sn-full sn-converter-shipping-field" hidden><label>آدرس کامل مشتری *</label><textarea id="sn-converter-cust-address" rows="3" autocomplete="street-address" placeholder="استان، شهر، خیابان، کوچه و پلاک"></textarea><small class="sn-muted">برای محصول عادی/فیزیکی اجباری است.</small></div>
						<div class="sn-field sn-converter-shipping-field" hidden><label>کد پستی</label><input type="text" id="sn-converter-cust-postal" inputmode="numeric" autocomplete="postal-code" maxlength="10" placeholder="۱۰ رقم (اختیاری)"></div>
						<div class="sn-field sn-full">
							<label>محصول‌ها *</label>
							<div id="sn-converter-products-multi" class="sn-products-multi">
								<div class="sn-product-row">
									<select class="sn-converter-product-select"><option value="">انتخاب محصول</option><?php foreach ( $products as $product ) : ?><option value="<?php echo esc_attr( (string) $product['id'] ); ?>" data-price="<?php echo esc_attr( (string) $product['price'] ); ?>" data-sn-product-type="<?php echo esc_attr( (string) ( $product['sales_product_type'] ?? 'product' ) ); ?>"><?php echo esc_html( (string) $product['name'] ); ?> — <?php echo esc_html( SN_Helpers::format_price( (float) $product['price'] ) ); ?></option><?php endforeach; ?></select>
									<input type="number" class="sn-converter-product-qty" min="1" value="1" aria-label="تعداد">
									<button type="button" class="sn-btn sn-btn-ghost sn-converter-remove-product" style="display:none">حذف</button>
								</div>
							</div>
							<button type="button" id="sn-converter-add-product" class="sn-btn sn-btn-secondary sn-btn-sm" data-max-products="<?php echo esc_attr( (string) $max_invoice_products ); ?>">+ افزودن محصول دیگر</button>
							<small class="sn-muted"><?php echo $max_invoice_products > 0 ? esc_html( sprintf( 'حداکثر %d محصول در هر پیش‌فاکتور', $max_invoice_products ) ) : 'تعداد محصولات بدون محدودیت است'; ?></small>
							<div id="sn-converter-products-total" class="sn-products-total">جمع: ۰ تومان</div>
						</div>
						<div class="sn-field"><label>نوع پرداخت مشتری *</label><select id="sn-converter-payment-plan"><option value="full">پرداخت کامل فاکتور</option><option value="partial">پیش‌پرداخت</option></select></div>
						<div class="sn-field sn-converter-prepayment-fields" hidden>
							<label>مبلغ پیش‌پرداخت این مرحله *</label>
							<select id="sn-converter-prepayment-choice"><option value="">انتخاب مبلغ پیش‌پرداخت</option><?php foreach ( $payment_presets as $preset ) : ?><option value="<?php echo esc_attr( (string) (int) $preset ); ?>"><?php echo esc_html( SN_Helpers::format_price( (float) $preset ) ); ?></option><?php endforeach; ?><option value="custom">سایر — ورود مبلغ</option></select>
							<input type="text" id="sn-converter-prepayment-custom" inputmode="numeric" placeholder="مبلغ پیش‌پرداخت به تومان" hidden>
							<small class="sn-muted">مبلغ این مرحله باید بیشتر از صفر و حداکثر برابر جمع کل فاکتور باشد.</small>
						</div>
					</div>
					<button type="button" id="sn-converter-create-invoice" class="sn-btn sn-btn-primary">صدور پیش‌فاکتور و ارسال پیامک</button>
				</div>
			</div>
			<?php endif; ?>

			<div id="sn-tab-invoices" class="sn-tab-content">
				<div class="sn-subtabs sn-converter-invoice-status-tabs">
					<button type="button" class="sn-subtab active" data-status="all">همه <span class="sn-invoice-tab-count">—</span></button>
					<button type="button" class="sn-subtab" data-status="pre_invoice">در جریان <span class="sn-invoice-tab-count">—</span></button>
					<button type="button" class="sn-subtab" data-status="paid">پرداخت شده <span class="sn-invoice-tab-count">—</span></button>
					<button type="button" class="sn-subtab" data-status="staged_incomplete">مرحله‌ای تکمیل‌نشده <span class="sn-invoice-tab-count">—</span></button>
					<button type="button" class="sn-subtab" data-status="staged_completed">مرحله‌ای تکمیل‌شده <span class="sn-invoice-tab-count">—</span></button>
					<button type="button" class="sn-subtab" data-status="rejected">رد / لغو <span class="sn-invoice-tab-count">—</span></button>
				</div>
				<div class="sn-card sn-converter-invoice-search-card"><label>جستجوی فاکتور<br><input type="search" id="sn-converter-invoice-search" placeholder="کد فاکتور، نام یا موبایل مشتری"></label><p class="sn-note">فاکتورهای دستی شما و همه پرداخت‌های دات‌فلو مربوط به پرونده‌هایی که تبدیل کرده‌اید، در این بخش نمایش داده می‌شوند.</p></div>
				<div id="sn-converter-invoices-notice"></div>
				<div id="sn-converter-invoices-loading" class="sn-loading" hidden>در حال بارگذاری...</div>
				<div id="sn-converter-invoices-list"></div>
			</div>

			<div id="sn-tab-customer-actions" class="sn-tab-content">
				<div class="sn-card sn-staff-customer-actions" data-page="1" data-loaded="0">
					<h3>رفتار مشتریان در صفحه فاکتور</h3>
					<p class="sn-muted">رفتار مشتری در فاکتورهای دستی شما و فاکتورهای مبدأ پرونده‌هایی که به شما تخصیص داده شده‌اند.</p>
					<div class="sn-staff-customer-actions-toolbar"><input type="search" class="sn-staff-customer-actions-search" placeholder="کد فاکتور، نام یا موبایل مشتری"><button type="button" class="sn-btn sn-btn-secondary sn-staff-customer-actions-filter">جستجو</button></div>
					<div class="sn-staff-customer-actions-loading sn-loading" hidden>در حال بارگذاری...</div>
					<div class="sn-staff-customer-actions-list"></div><div class="sn-staff-customer-actions-pager"></div>
				</div>
			</div>

			<div id="sn-tab-wallet" class="sn-tab-content">
				<?php if ( class_exists( 'SN_Purpose_Commission' ) ) { echo SN_Purpose_Commission::render_dual_wallet_cards( $user_id, 25 ); } else { ?>
				<section class="sn-project-wallet sn-project-wallet-integrated"><header><div><span>پورسانت پروژه‌ها</span><h3>کیف پول بیاوین</h3></div><button type="button" class="sn-project-wallet-refresh">بروزرسانی</button></header><div class="sn-project-wallet-body" data-loaded="0">در حال دریافت کیف پول...</div></section>
				<?php } ?>
			</div>
			<?php endif; ?>
		</div>
		<?php return ob_get_clean();
	}

	public function handle_converter_update(): void {
		$case_id = absint( $_POST['case_id'] ?? 0 );
		if ( ! wp_verify_nonce( sanitize_text_field( wp_unslash( $_POST['_wpnonce'] ?? '' ) ), 'sn_dot_converter_update_' . $case_id ) ) { $this->finish_action( false, 'درخواست امنیتی نامعتبر است.' ); }
		$case = $this->get_case( $case_id ); $user_id = get_current_user_id();
		if ( ! $case || ! $this->can_manage_converter_case( $case, $user_id ) || ! empty( $case->operational_archived_at ) || (string) $case->status === 'completed' ) { $this->finish_action( false, 'دسترسی به پرونده مجاز نیست.' ); }
		global $wpdb; $t = $this->tables();
		$wpdb->query( 'START TRANSACTION' );
		$locked_case = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['cases']} WHERE id=%d FOR UPDATE", $case_id ) );
		if ( ! $locked_case || ! $this->can_manage_converter_case( $locked_case, $user_id ) || ! empty( $locked_case->operational_archived_at ) || (string) $locked_case->status === 'completed' ) {
			$wpdb->query( 'ROLLBACK' );
			$this->finish_action( false, 'تخصیص پرونده در حین ویرایش تغییر کرده است.' );
		}
		$case = $locked_case;
		$status = sanitize_key( wp_unslash( $_POST['contact_status'] ?? '' ) );
		$allowed = [ 'no_answer', 'follow_up', 'customer_declined', 'payment_link' ];
		if ( ! in_array( $status, $allowed, true ) ) {
			$wpdb->query( 'ROLLBACK' );
			$this->finish_action( false, 'یکی از نتیجه‌های تماس را انتخاب کنید.' );
		}
		$decline_reason = sanitize_textarea_field( wp_unslash( $_POST['decline_reason'] ?? '' ) );
		// انصراف یک نتیجه تماس قابل اصلاح است، نه وضعیت نهایی غیرقابل بازگشت.
		// سرپرست/تبدیل‌کننده مجاز است بعداً همان پرونده را به جواب نداده،
		// تماس مجدد یا ارسال لینک پرداخت برگرداند. تاریخچه انصراف در لاگ پرونده حفظ می‌شود.
		if ( $status === 'customer_declined' && trim( $decline_reason ) === '' ) {
			$wpdb->query( 'ROLLBACK' );
			$this->finish_action( false, 'برای ثبت انصراف، واردکردن علت انصراف اجباری است.' );
		}
		$followup = null;
		if ( $status === 'follow_up' ) {
			$followup_raw = (string) ( $_POST['followup_at'] ?? '' );
			if ( trim( $followup_raw ) === '' ) {
				$followup_date = sanitize_text_field( wp_unslash( $_POST['followup_date'] ?? '' ) );
				$followup_time = sanitize_text_field( wp_unslash( $_POST['followup_time'] ?? '' ) );
				$followup_raw = $followup_date !== '' && $followup_time !== '' ? $followup_date . 'T' . $followup_time : '';
			}
			$followup = $this->normalize_followup_datetime( $followup_raw );
			if ( is_wp_error( $followup ) ) {
				$wpdb->query( 'ROLLBACK' );
				$this->finish_action( false, $followup->get_error_message() );
			}
		}
		$now = current_time( 'mysql' );
		$data = [ 'converter_contact_status' => $status, 'converter_note' => sanitize_textarea_field( wp_unslash( $_POST['note'] ?? '' ) ), 'updated_at' => $now ];
		if ( $status === 'no_answer' ) {
			$data['converter_no_answer_attempts'] = max( 0, (int) ( $case->converter_no_answer_attempts ?? 0 ) ) + 1;
		}
		$old_followup_at = trim( (string) ( $case->converter_followup_at ?? '' ) );
		if ( $status === 'follow_up' && is_array( $followup ) ) {
			$data['converter_followup_at'] = (string) $followup['mysql'];
			if ( $old_followup_at !== (string) $followup['mysql'] ) {
				$data['converter_followup_reminded_at'] = null;
				$data['converter_followup_reminder_attempts'] = 0;
				$data['converter_followup_reminder_error'] = null;
			}
		} else {
			$data['converter_followup_at'] = null;
			$data['converter_followup_reminded_at'] = null;
			$data['converter_followup_reminder_attempts'] = 0;
			$data['converter_followup_reminder_error'] = null;
		}
		$payment_count = (int) $wpdb->get_var( $wpdb->prepare( "SELECT COUNT(*) FROM {$t['payments']} WHERE case_id=%d", $case_id ) );
		if ( $status === 'customer_declined' && $payment_count > 0 ) {
			$wpdb->query( 'ROLLBACK' );
			$this->finish_action( false, 'برای پرونده‌ای که وارد چرخه پرداخت شده، انصراف مستقیم مجاز نیست؛ ابتدا پرداخت باز را تعیین تکلیف کنید.' );
		}
		if ( $status === 'customer_declined' ) {
			$data['converter_decline_reason'] = $decline_reason;
			$data['converter_declined_at'] = $now;
		} else {
			// If a previously declined case is reopened, current-state decline metadata
			// must not keep the case looking declined. Historical details remain in case logs.
			$data['converter_decline_reason'] = null;
			$data['converter_declined_at'] = null;
		}
		$case = $this->sync_live_case_option_prices( $case, $payment_count > 0 );
		$key = sanitize_key( wp_unslash( $_POST['option_key'] ?? '' ) );
		if ( $status === 'payment_link' && $key === '' && empty( $case->selected_option_key ) ) {
			$wpdb->query( 'ROLLBACK' );
			$this->finish_action( false, 'برای ارسال لینک پرداخت، ابتدا محصول نهایی را انتخاب کنید.' );
		}
		if ( ! $payment_count && $key !== '' && $key !== (string) $case->selected_option_key ) {
			$options = json_decode( (string) $case->options_snapshot_json, true );
			$matched_option = false;
			foreach ( is_array( $options ) ? $options : [] as $option ) {
				if ( ! empty( $option['active'] ) && (string) $option['key'] === $key ) {
					$data += [ 'selected_option_key' => $key, 'selected_option_product_id' => ! empty( $option['product_id'] ) ? absint( $option['product_id'] ) : null, 'selected_option_title' => sanitize_text_field( (string) $option['title'] ), 'selected_option_price' => (float) $option['price'], 'selected_option_html' => $this->sanitize_option_content( (string) $option['content'] ), 'selected_by' => $user_id, 'selected_at' => current_time( 'mysql' ), 'remaining_amount' => (float) $option['price'] ];
					$matched_option = true;
					break;
				}
			}
			if ( ! $matched_option ) { $wpdb->query( 'ROLLBACK' ); $this->finish_action( false, 'گزینه انتخاب‌شده معتبر نیست.' ); }
		}
		if ( ! in_array( (string) $case->status, [ 'completed', 'archived_unpaid_subscription' ], true ) ) { $data['status'] = $status === 'customer_declined' ? 'customer_declined' : 'contacted'; }
		if ( false === $wpdb->update( $t['cases'], $data, [ 'id' => $case_id ] ) ) { $wpdb->query( 'ROLLBACK' ); $this->finish_action( false, 'ذخیره پیگیری انجام نشد.' ); }
		$wpdb->query( 'COMMIT' );
		$this->clear_followup_reminder_schedule( $case_id );
		if ( $status === 'follow_up' && is_array( $followup ) && ( empty( $case->converter_followup_reminded_at ) || $old_followup_at !== (string) $followup['mysql'] ) ) {
			$this->schedule_followup_reminder( $case_id, (int) $followup['timestamp'] );
		}
		$this->log_case( $case_id, $user_id, $status === 'customer_declined' ? 'customer_declined_with_reason' : ( $status === 'follow_up' ? 'converter_followup_scheduled' : 'converter_case_updated' ), (string) $case->status, (string) ( $data['status'] ?? $case->status ), [ 'contact_status' => $status, 'option_key' => (string) ( $data['selected_option_key'] ?? $case->selected_option_key ), 'decline_reason' => $status === 'customer_declined' ? $decline_reason : '', 'followup_at' => $status === 'follow_up' && is_array( $followup ) ? (string) $followup['mysql'] : '' ] );
		$this->finish_action( true, $status === 'customer_declined' ? 'انصراف مشتری همراه با علت ثبت شد.' : ( $status === 'follow_up' ? 'زمان تماس مجدد ذخیره و یادآور آن فعال شد.' : ( $status === 'payment_link' ? 'محصول نهایی ثبت شد؛ اکنون مبلغ را وارد و لینک پرداخت را ارسال کنید.' : 'پیگیری پرونده ذخیره شد.' ) ) );
	}

	private function payment_type_label( string $type ): string {
		return [ 'full' => 'پرداخت کامل', 'deposit' => 'پرداخت مرحله‌ای', 'balance' => 'تسویه کامل مانده' ][ $type ] ?? $type;
	}

	private function payment_status_label( string $status ): string {
		return [
			'pending' => 'در انتظار پرداخت', 'pending_finance' => 'در انتظار تأیید مالی', 'rejected' => 'رد مالی',
			'superseded' => 'جایگزین‌شده', 'paid' => 'تأییدشده',
		][ $status ] ?? $status;
	}

	private function generate_invoice_code(): string {
		global $wpdb;
		for ( $attempt = 0; $attempt < 500; $attempt++ ) {
			$code = (string) wp_rand( 20000000, 99999999 );
			if ( ! $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$wpdb->prefix}sn_invoices WHERE invoice_code=%s LIMIT 1", $code ) ) ) { return $code; }
		}
		throw new RuntimeException( 'dot_invoice_code_pool_exhausted' );
	}

	private function generate_short_code(): string {
		global $wpdb;
		$alphabet = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789';
		$max = strlen( $alphabet ) - 1;
		for ( $attempt = 0; $attempt < 30; $attempt++ ) {
			$code = '';
			for ( $i = 0; $i < 9; $i++ ) { $code .= $alphabet[ random_int( 0, $max ) ]; }
			if ( ! $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$wpdb->prefix}sn_invoices WHERE short_code=%s LIMIT 1", $code ) ) ) { return $code; }
		}
		return wp_generate_password( 14, false, false );
	}

	private function invoice_short_url( object $invoice ): string {
		if ( ! empty( $invoice->short_code ) ) { return home_url( '/i/' . rawurlencode( (string) $invoice->short_code ) . '/' ); }
		$page_id = (int) get_option( 'sn_invoice_page_id', 0 );
		$base = $page_id ? get_permalink( $page_id ) : home_url( '/invoice/' );
		return add_query_arg( [ 'invoice_code' => (string) $invoice->invoice_code, 'invoice' => (string) $invoice->invoice_code, 'access_token' => (string) $invoice->access_token ], $base );
	}

	private function create_conversion_invoice( object $case, string $payment_type, float $amount, int $actor_id ) {
		global $wpdb;
		$t = $this->tables();
		$invoice_table = $wpdb->prefix . 'sn_invoices';
		$item_table = $wpdb->prefix . 'sn_invoice_items';
		$stage_table = $wpdb->prefix . 'sn_invoice_payment_stages';
		$source = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$invoice_table} WHERE id=%d LIMIT 1", (int) $case->source_invoice_id ) );
		if ( ! $source ) { return new WP_Error( 'sn_dot_source_missing', 'فاکتور مبنای اعتبارسنجی یافت نشد.' ); }
		try { $invoice_code = $this->generate_invoice_code(); $short_code = $this->generate_short_code(); }
		catch ( Throwable $e ) { return new WP_Error( 'sn_dot_code', 'ساخت کد فاکتور پرداخت ممکن نشد.' ); }
		$access_token = SN_Helpers::generate_unique_access_token();
		$now = current_time( 'mysql' );
		$wpdb->query( 'START TRANSACTION' );
		$locked_case = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['cases']} WHERE id=%d FOR UPDATE", (int) $case->id ) );
		if ( ! $locked_case || ! $this->can_manage_converter_case( $locked_case, $actor_id ) || empty( $locked_case->selected_option_key ) || (string) $locked_case->status === 'completed' ) {
			$wpdb->query( 'ROLLBACK' );
			return new WP_Error( 'sn_dot_case_state', 'وضعیت پرونده برای ساخت لینک پرداخت معتبر نیست.' );
		}
		$open_payment = (int) $wpdb->get_var( $wpdb->prepare( "SELECT COUNT(*) FROM {$t['payments']} WHERE case_id=%d AND status IN ('pending','pending_finance')", (int) $locked_case->id ) );
		if ( $open_payment > 0 ) {
			$wpdb->query( 'ROLLBACK' );
			return new WP_Error( 'sn_dot_open_payment', 'یک لینک پرداخت باز برای این پرونده وجود دارد.' );
		}
		$payment_count = (int) $wpdb->get_var( $wpdb->prepare( "SELECT COUNT(*) FROM {$t['payments']} WHERE case_id=%d", (int) $locked_case->id ) );
		$case = $this->sync_live_case_option_prices( $locked_case, $payment_count > 0 );
		$subscription_product_id = absint( $case->selected_option_product_id ?? 0 );
		if ( $subscription_product_id > 0 ) {
			$product = function_exists( 'wc_get_product' ) ? wc_get_product( $subscription_product_id ) : null;
			if ( ! $product || 'publish' !== get_post_status( $subscription_product_id ) || (string) get_post_meta( $subscription_product_id, '_sn_enabled', true ) !== '1' ) {
				$wpdb->query( 'ROLLBACK' );
				return new WP_Error( 'sn_dot_subscription_unavailable', 'محصول انتخاب‌شده پیش‌نویس، حذف‌شده یا غیرفعال شبکه فروش است و برای فروش جدید قابل استفاده نیست؛ سابقه پرونده حفظ شده است.' );
			}
			$is_paid_referral_case = ! empty( $case->referral_item_id );
			if ( ! $is_paid_referral_case && class_exists( 'SN_Projects' ) && ! SN_Projects::instance()->is_subscription_product( $subscription_product_id ) ) {
				$wpdb->query( 'ROLLBACK' );
				return new WP_Error( 'sn_dot_subscription_not_mapped', 'این محصول در تب «پروژه‌ها» به‌عنوان اشتراک و محتویاتش تعریف نشده است.' );
			}
		}
		$total = (float) $case->selected_option_price;
		$paid = (float) $case->paid_amount;
		$remaining = max( 0, $total - $paid );
		$expected_amount = 0.0;
		if ( $payment_type === 'full' && $paid <= 0 && $total > 0 ) {
			$expected_amount = $total;
		} elseif ( $payment_type === 'deposit' && $remaining > 0 && $amount > 0 && $amount < $remaining ) {
			// A partial payment can be issued at any point, not only as the first deposit.
			$expected_amount = $amount;
		} elseif ( $payment_type === 'balance' && $remaining > 0 ) {
			$expected_amount = $remaining;
		}
		if ( $expected_amount <= 0 || abs( $amount - $expected_amount ) > 0.5 ) {
			$wpdb->query( 'ROLLBACK' );
			return new WP_Error( 'sn_dot_payment_amount', 'مبلغ یا نوع پرداخت با مانده فعلی پرونده سازگار نیست.' );
		}

		// Once a replacement payment is issued, old rejected invoices must not
		// remain payable; otherwise a customer could accidentally pay both links.
		$rejected = $wpdb->get_results( $wpdb->prepare( "SELECT * FROM {$t['payments']} WHERE case_id=%d AND status='rejected' FOR UPDATE", (int) $case->id ) ) ?: [];
		foreach ( $rejected as $old_payment ) {
			$archive_data = $this->existing_columns( $invoice_table, [
				'status' => 'payment_archived', 'invoice_status' => 'payment_archived', 'payment_status' => 'payment_archived',
				'payment_workflow_status' => 'archived', 'payment_archived_at' => $now, 'payment_archived_by' => $actor_id,
				'payment_archive_reason' => 'جایگزینی لینک ردشده در دات فلو', 'updated_at' => $now,
			] );
			if ( false === $wpdb->update( $invoice_table, $archive_data, [ 'id' => (int) $old_payment->invoice_id ] )
				|| false === $wpdb->update( $t['payments'], [ 'status' => 'superseded', 'updated_at' => $now ], [ 'id' => (int) $old_payment->id ] ) ) {
				$wpdb->query( 'ROLLBACK' );
				return new WP_Error( 'sn_dot_archive_old_payment', 'بایگانی لینک پرداخت قبلی انجام نشد.' );
			}
		}
		$invoice_product_id = $subscription_product_id > 0 ? $subscription_product_id : (int) $case->source_product_id;
		$invoice_data = $this->existing_columns( $invoice_table, [
			'invoice_code' => $invoice_code, 'access_token' => $access_token, 'short_code' => $short_code,
			'seller_id' => (int) $case->seller_id, 'lead_id' => ! empty( $source->lead_id ) ? (int) $source->lead_id : null,
			'customer_wp_id' => ! empty( $case->customer_wp_id ) ? (int) $case->customer_wp_id : null,
			'customer_name' => (string) $case->customer_name, 'customer_phone' => (string) $case->customer_phone,
			'province' => (string) $case->province, 'city' => (string) $case->city,
			'product_id' => $invoice_product_id, 'product_price' => $amount, 'original_total' => $amount, 'final_total' => $amount,
			'payment_plan' => 'full', 'payment_total_amount' => $amount, 'current_due_amount' => $amount, 'paid_total_amount' => 0,
			'remaining_amount' => $amount, 'current_payment_stage' => 1, 'payment_workflow_status' => 'awaiting_payment',
			'current_stage_issued_by_user_id' => $actor_id, 'issued_by_user_id' => $actor_id, 'original_seller_id' => (int) $case->seller_id,
			'payment_card_number' => (string) ( $source->payment_card_number ?? '' ), 'payment_card_owner' => (string) ( $source->payment_card_owner ?? '' ),
			'payment_card_sales_manager_user_id' => ! empty( $source->payment_card_sales_manager_user_id ) ? (int) $source->payment_card_sales_manager_user_id : null,
			'payment_card_deputy_user_id' => ! empty( $source->payment_card_deputy_user_id ) ? (int) $source->payment_card_deputy_user_id : null,
			'status' => 'pre_invoice', 'invoice_status' => 'pre_invoice', 'payment_status' => 'pre_invoice', 'created_at' => $now, 'updated_at' => $now,
		] );
		$item_name = (string) $case->selected_option_title . ' — ' . $this->payment_type_label( $payment_type );
		$stage_type = [ 'full' => 'full', 'deposit' => 'partial', 'balance' => 'remaining' ][ $payment_type ] ?? 'partial';
		if ( ! $wpdb->insert( $invoice_table, $invoice_data ) ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_dot_invoice_insert', 'ذخیره فاکتور پرداخت انجام نشد: ' . ( $wpdb->last_error ?: 'خطای دیتابیس' ) ); }
		$invoice_id = (int) $wpdb->insert_id;
		$ok = $wpdb->insert( $item_table, [ 'invoice_id' => $invoice_id, 'product_id' => $invoice_product_id, 'product_name' => $item_name, 'qty' => 1, 'unit_price' => $amount, 'total_price' => $amount, 'is_free' => 0, 'created_at' => $now ] )
			&& $wpdb->insert( $stage_table, [ 'invoice_id' => $invoice_id, 'stage_no' => 1, 'stage_type' => $stage_type, 'requested_amount' => $amount, 'status' => 'pending', 'issued_by_user_id' => $actor_id, 'note' => 'دات فلو: ' . $this->payment_type_label( $payment_type ), 'created_at' => $now, 'updated_at' => $now ] )
			&& $wpdb->insert( $t['links'], [ 'invoice_id' => $invoice_id, 'case_id' => (int) $case->id, 'flow_kind' => 'conversion_payment', 'payment_kind' => $payment_type, 'supervisor_id' => (int) $case->supervisor_id, 'requires_finance' => 1, 'sms_delay_minutes' => 0, 'created_at' => $now, 'updated_at' => $now ] )
			&& $wpdb->insert( $t['payments'], [ 'case_id' => (int) $case->id, 'invoice_id' => $invoice_id, 'payment_type' => $payment_type, 'requested_amount' => $amount, 'status' => 'pending', 'created_by' => $actor_id, 'created_at' => $now, 'updated_at' => $now ] );
		if ( ! $ok ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_dot_payment_insert', 'ساخت لینک پرداخت کامل نشد: ' . ( $wpdb->last_error ?: 'خطای دیتابیس' ) ); }
		$payment_id = (int) $wpdb->insert_id;
		$case_update = [ 'status' => 'payment_link_sent', 'payment_mode' => $payment_type, 'remaining_amount' => max( 0, (float) $case->selected_option_price - (float) $case->paid_amount ), 'updated_at' => $now ];
		if ( false === $wpdb->update( $t['cases'], $case_update, [ 'id' => (int) $case->id ] ) ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_dot_case_payment', 'اتصال لینک پرداخت به پرونده انجام نشد.' ); }
		$wpdb->query( 'COMMIT' );
		$invoice = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$invoice_table} WHERE id=%d", $invoice_id ) );
		$this->log_case( (int) $case->id, $actor_id, 'payment_link_created', (string) $case->status, 'payment_link_sent', [ 'payment_id' => $payment_id, 'invoice_id' => $invoice_id, 'payment_type' => $payment_type, 'amount' => $amount ] );
		return [ 'invoice' => $invoice, 'payment_id' => $payment_id, 'payment_type' => $payment_type, 'amount' => $amount, 'case' => $case ];
	}

	private function send_payment_link_sms( object $case, object $invoice, string $payment_type, float $amount ): bool {
		$url = $this->invoice_short_url( $invoice );
		$remaining = max( 0, (float) $case->selected_option_price - (float) $case->paid_amount - $amount );
		$vars = [
			'customer_name' => (string) $case->customer_name,
			'option_title' => (string) $case->selected_option_title,
			'amount' => number_format( $amount, 0, '.', ',' ),
			'payment_type' => $this->payment_type_label( $payment_type ),
			'invoice_url' => $url,
			'remaining_amount' => number_format( $remaining, 0, '.', ',' ),
		];
		$context = [
			'case_id' => (int) $case->id,
			'invoice_id' => (int) $invoice->id,
			'include_current_user' => true,
		];
		$sent = $this->send_configured_sms( 'payment', (string) $case->customer_phone, $vars, $context );
		if ( $sent ) {
			$this->log_case( (int) $case->id, get_current_user_id(), $this->is_sms_test_mode() ? 'payment_sms_previewed' : 'payment_sms_sent', (string) $case->status, (string) $case->status, [
				'invoice_id' => (int) $invoice->id,
				'delivery_route' => $this->is_sms_test_mode() ? 'test_preview' : 'dot_payment',
			] );
			return true;
		}
		// Never turn an explicit test/preview run into a real SMS fallback.
		if ( $this->is_sms_test_mode() ) {
			$this->log_case( (int) $case->id, get_current_user_id(), 'payment_sms_preview_failed', (string) $case->status, (string) $case->status, [ 'invoice_id' => (int) $invoice->id ] );
			return false;
		}
		// Conversion payments historically used only the Dot-specific pattern.
		// Fall back to the canonical invoice sender so a missing/invalid Dot
		// payment pattern cannot strand an otherwise valid payment invoice.
		$phone = SN_Helpers::normalize_mobile( (string) $case->customer_phone );
		$fallback_sent = false;
		if ( class_exists( 'SN_SMS' ) && SN_Helpers::is_valid_mobile( $phone ) && $url !== '' ) {
			$card_number = '';
			if ( (string) get_option( 'sn_card_to_card_enabled', '1' ) === '1' ) {
				$card_number = (string) ( $invoice->payment_card_number ?? '' );
				if ( $card_number === '' ) { $card_number = (string) get_option( 'sn_card_number', '' ); }
			}
			$fallback_sent = (new SN_SMS())->send_invoice_link(
				$phone,
				(string) $invoice->invoice_code,
				$url,
				(string) $case->customer_name,
				number_format( max( 0, $amount ), 0, '', ',' ),
				$card_number
			);
		}
		$this->log_case( (int) $case->id, get_current_user_id(), $fallback_sent ? 'payment_sms_fallback_sent' : 'payment_sms_failed', (string) $case->status, (string) $case->status, [
			'invoice_id' => (int) $invoice->id,
			'delivery_route' => $fallback_sent ? 'canonical_invoice' : 'all_routes_failed',
		] );
		return $fallback_sent;
	}

	public function handle_create_payment(): void {
		$case_id = absint( $_POST['case_id'] ?? 0 );
		if ( ! wp_verify_nonce( sanitize_text_field( wp_unslash( $_POST['_wpnonce'] ?? '' ) ), 'sn_dot_create_payment_' . $case_id ) ) { $this->finish_action( false, 'درخواست امنیتی نامعتبر است.' ); }
		$case = $this->get_case( $case_id ); $user_id = get_current_user_id();
		if ( ! $case || ! $this->can_manage_converter_case( $case, $user_id ) || empty( $case->selected_option_key ) || in_array( (string) $case->status, [ 'customer_declined', 'archived_unpaid_subscription' ], true ) ) { $this->finish_action( false, 'دسترسی به پرونده یا گزینه نهایی معتبر نیست.' ); }
		if ( (string) $case->status === 'completed' ) { $this->finish_action( false, 'پرداخت این پرونده قبلاً کامل شده است.' ); }
		global $wpdb; $t = $this->tables();
		$case = $this->sync_live_case_option_prices( $case );
		$open = (int) $wpdb->get_var( $wpdb->prepare( "SELECT COUNT(*) FROM {$t['payments']} WHERE case_id=%d AND status IN ('pending','pending_finance')", $case_id ) );
		if ( $open ) { $this->finish_action( false, 'ابتدا لینک پرداخت باز فعلی باید پرداخت یا رد شود.' ); }
		$type = sanitize_key( wp_unslash( $_POST['payment_type'] ?? '' ) );
		$total = (float) $case->selected_option_price;
		$paid = (float) $case->paid_amount;
		$remaining = max( 0, $total - $paid );
		$amount = 0.0;
		if ( $type === 'full' && $paid <= 0 ) {
			$amount = $total;
		} elseif ( $type === 'deposit' && $remaining > 0 ) {
			$amount = $this->parse_money( $_POST['deposit_amount'] ?? 0 );
			if ( $amount <= 0 || $amount - $remaining > 0.5 ) {
				$this->finish_action( false, 'مبلغ این مرحله باید بیشتر از صفر و حداکثر برابر مانده فعلی پرونده باشد.' );
			}
			if ( abs( $amount - $remaining ) <= 0.5 ) { $amount = $remaining; $type = $paid > 0 ? 'balance' : 'full'; }
		} elseif ( $type === 'balance' && $remaining > 0 ) {
			$amount = $remaining;
		} else {
			$this->finish_action( false, 'نوع پرداخت با وضعیت فعلی پرونده سازگار نیست.' );
		}
		$result = $this->create_conversion_invoice( $case, $type, $amount, $user_id );
		if ( is_wp_error( $result ) ) { $this->finish_action( false, $result->get_error_message() ); }
		$case = $result['case'] ?? $case;
		$invoice = $result['invoice'];
		$is_test_preview = $this->is_sms_test_mode();
		$sent = $invoice ? $this->send_payment_link_sms( $case, $invoice, $type, $amount ) : false;
		if ( $sent && ! $is_test_preview ) { $wpdb->update( $t['payments'], [ 'sms_sent_at' => current_time( 'mysql' ), 'updated_at' => current_time( 'mysql' ) ], [ 'id' => (int) $result['payment_id'] ] ); }
		$message = $is_test_preview
			? ( $sent ? 'پیش‌نمایش آزمایشی پیامک ساخته شد؛ پیامک واقعی ارسال نشد.' : 'ساخت پیش‌نمایش آزمایشی پیامک ناموفق بود.' )
			: ( $sent ? 'لینک پرداخت ساخته و پیامک شد.' : 'لینک پرداخت ساخته شد اما ارسال پیامک ناموفق بود؛ می‌توانید از جدول پرداخت دوباره ارسال کنید.' );
		$this->finish_action( true, $message );
	}

	public function handle_resend_payment(): void {
		$payment_id = absint( $_POST['payment_id'] ?? 0 );
		if ( ! wp_verify_nonce( sanitize_text_field( wp_unslash( $_POST['_wpnonce'] ?? '' ) ), 'sn_dot_resend_payment_' . $payment_id ) ) { $this->finish_action( false, 'درخواست امنیتی نامعتبر است.' ); }
		global $wpdb; $t = $this->tables();
		$payment = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['payments']} WHERE id=%d LIMIT 1", $payment_id ) );
		$case = $payment ? $this->get_case( (int) $payment->case_id ) : null;
		if ( ! $payment || ! $case || ! $this->can_manage_converter_case( $case, get_current_user_id() ) ) { $this->finish_action( false, 'دسترسی به این پرداخت مجاز نیست.' ); }
		if ( (string) $payment->status === 'paid' ) { $this->finish_action( false, 'این پرداخت قبلاً تأیید شده است.' ); }
		if ( ! in_array( (string) $payment->status, [ 'pending', 'pending_finance', 'rejected' ], true ) ) { $this->finish_action( false, 'این لینک با لینک جدید جایگزین شده و قابل ارسال نیست.' ); }
		$newer = (int) $wpdb->get_var( $wpdb->prepare( "SELECT COUNT(*) FROM {$t['payments']} WHERE case_id=%d AND id>%d", (int) $payment->case_id, $payment_id ) );
		if ( $newer > 0 ) { $this->finish_action( false, 'این لینک با یک پرداخت جدیدتر جایگزین شده است.' ); }
		$invoice = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$wpdb->prefix}sn_invoices WHERE id=%d LIMIT 1", (int) $payment->invoice_id ) );
		$is_test_preview = $this->is_sms_test_mode();
		$sent = $invoice ? $this->send_payment_link_sms( $case, $invoice, (string) $payment->payment_type, (float) $payment->requested_amount ) : false;
		if ( $sent && ! $is_test_preview ) { $wpdb->update( $t['payments'], [ 'sms_sent_at' => current_time( 'mysql' ), 'updated_at' => current_time( 'mysql' ) ], [ 'id' => $payment_id ] ); }
		if ( $is_test_preview ) {
			$this->finish_action( $sent, $sent ? 'پیش‌نمایش آزمایشی ساخته شد؛ پیامک واقعی ارسال نشد.' : 'ساخت پیش‌نمایش آزمایشی ناموفق بود.' );
		}
		$this->finish_action( $sent, $sent ? 'لینک پرداخت دوباره پیامک شد.' : 'ارسال مجدد پیامک ناموفق بود.' );
	}

	private function complete_conversion_payment( object $link, object $invoice ): void {
		global $wpdb; $t = $this->tables(); $now = current_time( 'mysql' );
		$wpdb->query( 'START TRANSACTION' );
		$payment = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['payments']} WHERE invoice_id=%d FOR UPDATE", (int) $invoice->id ) );
		if ( ! $payment ) { $wpdb->query( 'ROLLBACK' ); return; }
		if ( (string) $payment->status === 'paid' ) { $wpdb->query( 'COMMIT' ); return; }
		$case = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['cases']} WHERE id=%d FOR UPDATE", (int) $payment->case_id ) );
		if ( ! $case ) { $wpdb->query( 'ROLLBACK' ); return; }
		$total = (float) $case->selected_option_price;
		$new_paid = min( $total, (float) $case->paid_amount + (float) $payment->requested_amount );
		$remaining = max( 0, $total - $new_paid );
		$new_status = $remaining <= 0.5 ? 'completed' : 'deposit_paid';
		$case_data = [
			'paid_amount' => $new_paid,
			'remaining_amount' => $remaining,
			'status' => $new_status,
			'operational_archived_at' => null,
			'operational_archive_reason' => null,
			'updated_at' => $now,
		];
		if ( (string) $payment->payment_type === 'deposit' && (float) ( $case->deposit_amount ?? 0 ) <= 0 ) { $case_data['deposit_amount'] = (float) $payment->requested_amount; }
		if ( $new_status === 'completed' ) { $case_data['completed_at'] = $now; }
		if ( $new_status === 'completed' ) {
			$other_payments = $wpdb->get_results( $wpdb->prepare( "SELECT * FROM {$t['payments']} WHERE case_id=%d AND id<>%d AND status IN ('pending','pending_finance','rejected') FOR UPDATE", (int) $case->id, (int) $payment->id ) ) ?: [];
			foreach ( $other_payments as $other_payment ) {
				$archive_data = $this->existing_columns( $wpdb->prefix . 'sn_invoices', [
					'status' => 'payment_archived', 'invoice_status' => 'payment_archived', 'payment_status' => 'payment_archived',
					'payment_workflow_status' => 'archived', 'payment_archived_at' => $now,
					'payment_archive_reason' => 'تکمیل پرونده از یک لینک پرداخت دیگر', 'updated_at' => $now,
				] );
				if ( false === $wpdb->update( $wpdb->prefix . 'sn_invoices', $archive_data, [ 'id' => (int) $other_payment->invoice_id ] )
					|| false === $wpdb->update( $t['payments'], [ 'status' => 'superseded', 'updated_at' => $now ], [ 'id' => (int) $other_payment->id ] ) ) {
					$wpdb->query( 'ROLLBACK' );
					return;
				}
			}
		}
		$ok = false !== $wpdb->update( $t['payments'], [ 'status' => 'paid', 'paid_at' => $now, 'updated_at' => $now ], [ 'id' => (int) $payment->id ] )
			&& false !== $wpdb->update( $t['cases'], $case_data, [ 'id' => (int) $case->id ] )
			&& false !== $wpdb->update( $t['links'], [ 'payment_verified_at' => (string) ( $invoice->paid_at ?? $now ), 'updated_at' => $now ], [ 'id' => (int) $link->id ] );
		if ( ! $ok ) { $wpdb->query( 'ROLLBACK' ); return; }
		$wpdb->query( 'COMMIT' );
		$this->resolve_operational_archive( 'subscription_unpaid', 'dot_case', (int) $case->id, 'subscription_payment_approved' );
		$this->log_case( (int) $case->id, 0, 'conversion_payment_paid', (string) $case->status, $new_status, [ 'invoice_id' => (int) $invoice->id, 'payment_type' => (string) $payment->payment_type, 'amount' => (float) $payment->requested_amount, 'paid_amount' => $new_paid, 'remaining_amount' => $remaining ] );
	}


	/**
	 * Guard a Finance "reopen approved" action.  Assessment-source invoices are
	 * not reversible once a Dot case exists because that case is already a live
	 * business object.  Conversion payments can be reconciled transactionally.
	 */
	public function financial_reopen_guard( int $invoice_id ): array {
		global $wpdb;
		$t = $this->tables();
		if ( $invoice_id < 1 || ! $this->table_exists( $t['links'] ) ) { return [ 'allowed' => true ]; }
		$link = $wpdb->get_row( $wpdb->prepare( "SELECT flow_kind,case_id FROM {$t['links']} WHERE invoice_id=%d LIMIT 1", $invoice_id ) );
		if ( ! $link ) { return [ 'allowed' => true ]; }
		if ( (string) $link->flow_kind === 'assessment_source' && (int) $link->case_id > 0 ) {
			return [
				'allowed' => false,
				'message' => 'این فاکتور اعتبارسنجی قبلاً پرونده دات‌فلو ساخته است. بازگشت تایید مالی مستقیم برای جلوگیری از ناقص شدن پرونده مسدود است؛ در صورت نیاز ابتدا وضعیت پرونده دات باید تعیین تکلیف شود.',
			];
		}
		if ( (string) $link->flow_kind === 'conversion_payment' && (int) $link->case_id > 0 ) {
			if ( ! $this->table_exists( $t['payments'] ) || ! $this->table_exists( $t['cases'] ) ) {
				return [ 'allowed' => false, 'message' => 'جداول دات‌فلو برای بازگشت هماهنگ پرداخت در دسترس نیستند.' ];
			}
			$payment_exists = (int) $wpdb->get_var( $wpdb->prepare( "SELECT COUNT(*) FROM {$t['payments']} WHERE invoice_id=%d", $invoice_id ) );
			$case_exists = (int) $wpdb->get_var( $wpdb->prepare( "SELECT COUNT(*) FROM {$t['cases']} WHERE id=%d", (int) $link->case_id ) );
			if ( $payment_exists < 1 || $case_exists < 1 ) {
				return [ 'allowed' => false, 'message' => 'ارتباط فاکتور تبدیل با پرونده دات ناقص است و بازگشت برای جلوگیری از اختلاف مبلغ متوقف شد.' ];
			}
		}
		return [ 'allowed' => true ];
	}

	/**
	 * Apply Dot reconciliation inside the caller's existing SQL transaction.
	 * Do not START/COMMIT here.
	 */
	public function financial_reopen_apply_in_transaction( int $invoice_id, string $reason = '' ): array {
		global $wpdb;
		$t = $this->tables();
		if ( $invoice_id < 1 || ! $this->table_exists( $t['links'] ) ) { return [ 'success' => true, 'handled' => false ]; }
		$link = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['links']} WHERE invoice_id=%d LIMIT 1 FOR UPDATE", $invoice_id ) );
		if ( ! $link || (string) $link->flow_kind !== 'conversion_payment' || (int) $link->case_id < 1 ) {
			return [ 'success' => true, 'handled' => false ];
		}
		$payment = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['payments']} WHERE invoice_id=%d FOR UPDATE", $invoice_id ) );
		$case = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['cases']} WHERE id=%d FOR UPDATE", (int) $link->case_id ) );
		if ( ! $payment || ! $case ) { return [ 'success' => false, 'message' => 'پرداخت یا پرونده دات برای بازگشت یافت نشد.' ]; }
		$old_status = (string) $case->status;
		$was_paid = (string) $payment->status === 'paid';
		$new_paid = max( 0, (float) $case->paid_amount - ( $was_paid ? (float) $payment->requested_amount : 0 ) );
		$total = max( 0, (float) $case->selected_option_price );
		$remaining = max( 0, $total - $new_paid );
		$new_status = $new_paid > 0.5 ? 'deposit_paid' : 'payment_link_sent';
		$now = current_time( 'mysql' );
		$payment_data = [ 'status' => 'pending_finance', 'paid_at' => null, 'updated_at' => $now ];
		$ok = false !== $wpdb->update( $t['payments'], $payment_data, [ 'id' => (int) $payment->id ] )
			&& false !== $wpdb->update( $t['cases'], [
				'paid_amount' => $new_paid,
				'remaining_amount' => $remaining,
				'status' => $new_status,
				'completed_at' => null,
				'updated_at' => $now,
			], [ 'id' => (int) $case->id ] );
		if ( ! $ok ) { return [ 'success' => false, 'message' => 'برگشت مبلغ پرونده دات انجام نشد.' ]; }
		$this->log_case( (int) $case->id, get_current_user_id(), 'conversion_payment_finance_reopened', $old_status, $new_status, [
			'invoice_id' => $invoice_id,
			'reason' => sanitize_textarea_field( $reason ),
			'payment_was_paid' => $was_paid,
			'reversed_amount' => $was_paid ? (float) $payment->requested_amount : 0,
			'paid_amount' => $new_paid,
			'remaining_amount' => $remaining,
		] );
		return [ 'success' => true, 'handled' => true ];
	}


	/**
	 * Keep Dot conversion-payment state aligned when a rejected invoice is
	 * corrected and resent to Finance. Caller already owns the SQL transaction.
	 */
	public function financial_resubmit_apply_in_transaction( int $invoice_id, string $note = '' ): array {
		global $wpdb;
		$t = $this->tables();
		if ( $invoice_id < 1 || ! $this->table_exists( $t['links'] ) ) { return [ 'success' => true, 'handled' => false ]; }
		$link = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['links']} WHERE invoice_id=%d LIMIT 1 FOR UPDATE", $invoice_id ) );
		if ( ! $link || (string) $link->flow_kind !== 'conversion_payment' || (int) $link->case_id < 1 ) {
			return [ 'success' => true, 'handled' => false ];
		}
		if ( ! $this->table_exists( $t['payments'] ) || ! $this->table_exists( $t['cases'] ) ) {
			return [ 'success' => false, 'message' => 'جداول پرونده و پرداخت دات برای ارسال مجدد در دسترس نیستند.' ];
		}
		$payment = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['payments']} WHERE invoice_id=%d FOR UPDATE", $invoice_id ) );
		$case = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['cases']} WHERE id=%d FOR UPDATE", (int) $link->case_id ) );
		if ( ! $payment || ! $case ) { return [ 'success' => false, 'message' => 'ارتباط پرداخت ردشده با پرونده دات ناقص است.' ]; }
		if ( (string) $payment->status === 'paid' ) { return [ 'success' => false, 'message' => 'این پرداخت دات قبلاً تأیید شده است.' ]; }
		if ( ! in_array( (string) $payment->status, [ 'rejected', 'pending', 'pending_finance' ], true ) ) {
			return [ 'success' => false, 'message' => 'پرداخت دات در وضعیت قابل ارسال مجدد نیست.' ];
		}
		$now = current_time( 'mysql' );
		$old_case_status = (string) $case->status;
		$new_case_status = (float) $case->paid_amount > 0.5 ? 'deposit_paid' : 'payment_link_sent';
		$ok = false !== $wpdb->update( $t['payments'], [ 'status' => 'pending_finance', 'updated_at' => $now ], [ 'id' => (int) $payment->id ] )
			&& false !== $wpdb->update( $t['cases'], [ 'status' => $new_case_status, 'updated_at' => $now ], [ 'id' => (int) $case->id ] );
		if ( ! $ok ) { return [ 'success' => false, 'message' => 'وضعیت پرداخت دات برای بررسی مجدد همگام نشد.' ]; }
		$this->log_case( (int) $case->id, get_current_user_id(), 'conversion_payment_resent_to_finance', $old_case_status, $new_case_status, [
			'invoice_id' => $invoice_id,
			'payment_id' => (int) $payment->id,
			'note' => sanitize_textarea_field( $note ),
		] );
		return [ 'success' => true, 'handled' => true ];
	}

	/**
	 * Guard Finance cancellation from orphaning an already-created assessment case.
	 * Conversion-payment invoices remain cancellable and are reconciled by
	 * on_financial_cancelled().
	 */
	public function financial_cancellation_guard( int $invoice_id ): array {
		global $wpdb;
		$t = $this->tables();
		if ( $invoice_id < 1 || ! $this->table_exists( $t['links'] ) ) { return [ 'allowed' => true ]; }
		$link = $wpdb->get_row( $wpdb->prepare( "SELECT flow_kind,case_id FROM {$t['links']} WHERE invoice_id=%d LIMIT 1", $invoice_id ) );
		if ( ! $link ) { return [ 'allowed' => true ]; }
		if ( (string) $link->flow_kind === 'assessment_source' && (int) $link->case_id > 0 ) {
			return [
				'allowed' => false,
				'message' => 'برای این فاکتور اعتبارسنجی، پرونده دات‌فلو قبلاً ساخته شده است. برای جلوگیری از ناقص شدن پرونده، لغو مستقیم مالی مسدود شده و باید ابتدا وضعیت پرونده دات تعیین تکلیف شود.',
			];
		}
		if ( (string) $link->flow_kind === 'conversion_payment' && (int) $link->case_id > 0 ) {
			if ( ! $this->table_exists( $t['payments'] ) || ! $this->table_exists( $t['cases'] ) ) {
				return [ 'allowed' => false, 'message' => 'جداول دات‌فلو برای تطبیق لغو مالی در دسترس نیستند؛ لغو برای جلوگیری از ناسازگاری متوقف شد.' ];
			}
			$payment_exists = (int) $wpdb->get_var( $wpdb->prepare( "SELECT COUNT(*) FROM {$t['payments']} WHERE invoice_id=%d", $invoice_id ) );
			$case_exists = (int) $wpdb->get_var( $wpdb->prepare( "SELECT COUNT(*) FROM {$t['cases']} WHERE id=%d", (int) $link->case_id ) );
			if ( $payment_exists < 1 || $case_exists < 1 ) {
				return [ 'allowed' => false, 'message' => 'ارتباط این فاکتور تبدیل با پرونده دات‌فلو ناقص است؛ برای جلوگیری از اختلاف مبالغ، لغو مالی تا تعمیر ارتباط مسدود شد.' ];
			}
		}
		return [ 'allowed' => true ];
	}

	/**
	 * Keep Dot case totals consistent when Finance cancels/refunds a synthetic
	 * conversion-payment invoice. Returns true when the invoice belonged to a
	 * Dot conversion payment and was handled here.
	 */
	public function on_financial_cancelled( int $invoice_id, string $reason = '' ): bool {
		global $wpdb;
		$t = $this->tables();
		if ( $invoice_id < 1 || ! $this->table_exists( $t['links'] ) || ! $this->table_exists( $t['payments'] ) ) { return false; }
		$link = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['links']} WHERE invoice_id=%d AND flow_kind='conversion_payment' LIMIT 1", $invoice_id ) );
		if ( ! $link || (int) $link->case_id < 1 ) { return false; }
		$now = current_time( 'mysql' );
		$wpdb->query( 'START TRANSACTION' );
		$payment = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['payments']} WHERE invoice_id=%d FOR UPDATE", $invoice_id ) );
		$case = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['cases']} WHERE id=%d FOR UPDATE", (int) $link->case_id ) );
		if ( ! $payment || ! $case ) { $wpdb->query( 'ROLLBACK' ); return true; }
		$old_case_status = (string) $case->status;
		$was_paid = (string) $payment->status === 'paid';
		$new_paid = max( 0, (float) $case->paid_amount - ( $was_paid ? (float) $payment->requested_amount : 0 ) );
		$total = max( 0, (float) $case->selected_option_price );
		$remaining = max( 0, $total - $new_paid );
		$new_status = $new_paid > 0.5 ? 'deposit_paid' : 'payment_rejected';
		$case_data = [ 'paid_amount' => $new_paid, 'remaining_amount' => $remaining, 'status' => $new_status, 'completed_at' => null, 'updated_at' => $now ];
		$ok = false !== $wpdb->update( $t['payments'], [ 'status' => 'rejected', 'updated_at' => $now ], [ 'id' => (int) $payment->id ] )
			&& false !== $wpdb->update( $t['cases'], $case_data, [ 'id' => (int) $case->id ] );
		if ( ! $ok ) { $wpdb->query( 'ROLLBACK' ); return true; }
		$wpdb->query( 'COMMIT' );
		$this->log_case( (int) $case->id, get_current_user_id(), 'conversion_payment_cancelled_by_finance', $old_case_status, $new_status, [
			'invoice_id' => $invoice_id,
			'reason' => sanitize_textarea_field( $reason ),
			'payment_was_paid' => $was_paid,
			'reversed_case_amount' => $was_paid ? (float) $payment->requested_amount : 0,
			'paid_amount' => $new_paid,
			'remaining_amount' => $remaining,
		] );
		return true;
	}

}
