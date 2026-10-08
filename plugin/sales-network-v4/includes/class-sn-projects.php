<?php
if ( ! defined( 'ABSPATH' ) ) { exit; }

/**
 * Subscription projects, Biavin collaboration and attribution.
 *
 * All project data lives in additive tables. Existing WooCommerce products,
 * invoices and Dot cases are referenced by id and are never rewritten.
 */
final class SN_Projects {
	private const DB_VERSION = '2026-09-17-biavin-sales-chat-sync-v1';
	private const MAX_FILE_SIZE = 8388608;
	private static ?self $instance = null;
	private bool $hooks_registered = false;
	private array $table_exists_cache = [];
	private ?array $subscription_ids_cache = null;
	private ?array $content_ids_cache = null;

	public static function instance(): self {
		if ( null === self::$instance ) { self::$instance = new self(); }
		return self::$instance;
	}

	private function __construct() {}

	public static function activate(): void {
		$self = self::instance();
		$self->install_schema();
		$self->ensure_roles_and_hr();
	}

	public function register_hooks(): void {
		if ( $this->hooks_registered ) { return; }
		$this->hooks_registered = true;
		add_action( 'init', [ $this, 'maybe_upgrade' ], 8 );
		add_action( 'admin_init', [ $this, 'maybe_upgrade' ], 8 );
		add_action( 'init', [ $this, 'register_shortcodes' ], 2 );
		add_action( 'wp_enqueue_scripts', [ $this, 'enqueue_assets' ], 30 );
		add_filter( 'woocommerce_is_purchasable', [ $this, 'filter_content_purchasable' ], 20, 2 );
		add_filter( 'woocommerce_variation_is_purchasable', [ $this, 'filter_content_purchasable' ], 20, 2 );
		add_filter( 'woocommerce_product_is_visible', [ $this, 'filter_content_visible' ], 20, 2 );
		add_filter( 'woocommerce_add_to_cart_validation', [ $this, 'validate_content_add_to_cart' ], 20, 3 );
		add_action( 'sn_invoice_payment_stage_approved', [ $this, 'on_invoice_stage_approved' ], 20, 3 );
		add_action( 'sn_invoice_paid', [ $this, 'on_invoice_paid' ], 20, 2 );
		add_action( 'sn_invoice_financial_rejected', [ $this, 'on_invoice_reversed' ], 20, 3 );
		add_action( 'sn_invoice_financial_reopened', [ $this, 'on_invoice_reversed' ], 20, 3 );

		foreach ( [
			'sn_project_dashboard' => 'ajax_dashboard',
			'sn_project_assign_expert' => 'ajax_assign_expert',
			'sn_project_bulk_assign_expert' => 'ajax_bulk_assign_expert',
			'sn_project_update_status' => 'ajax_update_status',
			'sn_project_add_note' => 'ajax_add_note',
			'sn_project_thread' => 'ajax_thread',
			'sn_project_send_message' => 'ajax_send_message',
			'sn_project_notifications' => 'ajax_notifications',
			'sn_project_link_invoice' => 'ajax_link_invoice',
			'sn_project_wallet' => 'ajax_wallet',
			'sn_project_create_action' => 'ajax_create_action',
			'sn_project_retry_action_payment' => 'ajax_retry_action_payment',
			'sn_project_cancel_upsell' => 'ajax_cancel_upsell',
			'sn_project_fulfill_action' => 'ajax_fulfill_action',
			'sn_project_api_products' => 'ajax_api_products',
			'sn_project_api_purchase' => 'ajax_api_purchase',
			'sn_project_api_charge_wallet' => 'ajax_api_charge_wallet',
		] as $action => $method ) {
			add_action( 'wp_ajax_' . $action, [ $this, $method ] );
		}
		add_action( 'admin_post_sn_project_download', [ $this, 'handle_download' ] );
	}

	public function register_shortcodes(): void {
		add_shortcode( 'sn_project_manager_panel', [ $this, 'render_manager_panel' ] );
		add_shortcode( 'sn_project_expert_panel', [ $this, 'render_expert_panel' ] );
	}

	public function maybe_upgrade(): void {
		if ( (string) get_option( 'sn_projects_db_version', '' ) !== self::DB_VERSION ) {
			$this->install_schema();
		}
		if ( (string) get_option( 'sn_projects_hr_version', '' ) !== self::DB_VERSION ) {
			$this->ensure_roles_and_hr();
		}
	}

	private function tables(): array {
		global $wpdb;
		return [
			'subscriptions' => $wpdb->prefix . 'sn_project_subscriptions',
			'contents' => $wpdb->prefix . 'sn_project_subscription_contents',
			'managers' => $wpdb->prefix . 'sn_project_manager_products',
			'manager_routes' => $wpdb->prefix . 'sn_project_manager_sales_routes',
			'operations_managers' => $wpdb->prefix . 'sn_project_operations_manager_products',
			'operations_manager_routes' => $wpdb->prefix . 'sn_project_operations_sales_routes',
			'operations_supervisor_routes' => $wpdb->prefix . 'sn_project_operations_supervisor_sales_routes',
			'upgrade_rules' => $wpdb->prefix . 'sn_project_card_upgrade_rules',
			'memberships' => $wpdb->prefix . 'sn_project_memberships',
			'items' => $wpdb->prefix . 'sn_project_membership_items',
			'notes' => $wpdb->prefix . 'sn_project_notes',
			'messages' => $wpdb->prefix . 'sn_project_messages',
			'reads' => $wpdb->prefix . 'sn_project_message_reads',
			'invoice_links' => $wpdb->prefix . 'sn_project_invoice_links',
			'rules' => $wpdb->prefix . 'sn_project_commission_rules',
			'commissions' => $wpdb->prefix . 'sn_project_commissions',
			'events' => $wpdb->prefix . 'sn_project_events',
			'actions' => $wpdb->prefix . 'sn_project_actions',
			'api_configs' => $wpdb->prefix . 'sn_project_api_configs',
		];
	}

	private function table_exists( string $table ): bool {
		global $wpdb;
		if ( ! array_key_exists( $table, $this->table_exists_cache ) ) {
			$this->table_exists_cache[ $table ] = (string) $wpdb->get_var( $wpdb->prepare( 'SHOW TABLES LIKE %s', $table ) ) === $table;
		}
		return $this->table_exists_cache[ $table ];
	}

	private function install_schema(): void {
		global $wpdb;
		require_once ABSPATH . 'wp-admin/includes/upgrade.php';
		$t = $this->tables();
		$charset = $wpdb->get_charset_collate();

		dbDelta( "CREATE TABLE {$t['subscriptions']} (
			subscription_product_id BIGINT UNSIGNED NOT NULL,
			is_active TINYINT(1) NOT NULL DEFAULT 1,
			created_by BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (subscription_product_id),
			KEY is_active (is_active)
		) {$charset};" );

		dbDelta( "CREATE TABLE {$t['contents']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			subscription_product_id BIGINT UNSIGNED NOT NULL,
			content_product_id BIGINT UNSIGNED NOT NULL,
			sort_order INT NOT NULL DEFAULT 0,
			is_active TINYINT(1) NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY subscription_content (subscription_product_id,content_product_id),
			KEY content_product_id (content_product_id),
			KEY is_active (is_active)
		) {$charset};" );

		dbDelta( "CREATE TABLE {$t['managers']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			content_product_id BIGINT UNSIGNED NOT NULL,
			manager_user_id BIGINT UNSIGNED NOT NULL,
			created_by BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY content_product_id (content_product_id),
			KEY manager_user_id (manager_user_id)
		) {$charset};" );

		// Optional branch overrides. The original one-manager-per-content table is
		// intentionally retained as the safe default for old settings and invoices.
		dbDelta( "CREATE TABLE {$t['manager_routes']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			content_product_id BIGINT UNSIGNED NOT NULL,
			sales_manager_user_id BIGINT UNSIGNED NOT NULL,
			manager_user_id BIGINT UNSIGNED NOT NULL,
			created_by BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY content_sales_manager (content_product_id,sales_manager_user_id),
			KEY sales_manager_user_id (sales_manager_user_id),
			KEY manager_user_id (manager_user_id)
		) {$charset};" );

		// Operations sales ownership is deliberately separate from project-manager
		// ownership. A card gets one immutable destination snapshot when it is
		// issued, while these tables remain editable configuration for future cards.
		dbDelta( "CREATE TABLE {$t['operations_managers']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			content_product_id BIGINT UNSIGNED NOT NULL,
			operations_manager_user_id BIGINT UNSIGNED NOT NULL,
			created_by BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY content_product_id (content_product_id),
			KEY operations_manager_user_id (operations_manager_user_id)
		) {$charset};" );

		dbDelta( "CREATE TABLE {$t['operations_manager_routes']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			content_product_id BIGINT UNSIGNED NOT NULL,
			sales_manager_user_id BIGINT UNSIGNED NOT NULL,
			operations_manager_user_id BIGINT UNSIGNED NOT NULL,
			created_by BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY content_sales_manager (content_product_id,sales_manager_user_id),
			KEY sales_manager_user_id (sales_manager_user_id),
			KEY operations_manager_user_id (operations_manager_user_id)
		) {$charset};" );

		// Direct Operations routing for future cards. The Operations manager is
		// stored alongside the supervisor so each route can be validated against
		// the current HR hierarchy and snapshotted without another live lookup.
		dbDelta( "CREATE TABLE {$t['operations_supervisor_routes']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			content_product_id BIGINT UNSIGNED NOT NULL,
			sales_manager_user_id BIGINT UNSIGNED NOT NULL,
			operations_manager_user_id BIGINT UNSIGNED NOT NULL,
			operations_supervisor_user_id BIGINT UNSIGNED NOT NULL,
			created_by BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY content_sales_manager (content_product_id,sales_manager_user_id),
			KEY operations_manager_user_id (operations_manager_user_id),
			KEY operations_supervisor_user_id (operations_supervisor_user_id)
		) {$charset};" );

		// Product* upgrade definitions are additive configuration. They are
		// snapshotted into new membership cards and never rewrite old cards.
		dbDelta( "CREATE TABLE {$t['upgrade_rules']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			source_product_id BIGINT UNSIGNED NOT NULL,
			target_product_id BIGINT UNSIGNED NOT NULL,
			upgrade_amount DECIMAL(18,2) NOT NULL DEFAULT 0,
			is_active TINYINT(1) NOT NULL DEFAULT 1,
			created_by BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY source_target (source_product_id,target_product_id),
			KEY target_product_id (target_product_id),
			KEY is_active (is_active)
		) {$charset};" );

		dbDelta( "CREATE TABLE {$t['memberships']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			source_key VARCHAR(191) NOT NULL,
			source_type VARCHAR(30) NOT NULL,
			source_invoice_id BIGINT UNSIGNED NOT NULL,
			dot_case_id BIGINT UNSIGNED DEFAULT NULL,
			subscription_product_id BIGINT UNSIGNED NOT NULL,
			subscription_name_snapshot VARCHAR(255) NOT NULL,
			customer_wp_id BIGINT UNSIGNED DEFAULT NULL,
			customer_name VARCHAR(120) NOT NULL,
			customer_phone VARCHAR(20) NOT NULL,
			origin_user_id BIGINT UNSIGNED NOT NULL,
			origin_role VARCHAR(30) NOT NULL,
			source_sales_manager_user_id BIGINT UNSIGNED DEFAULT NULL,
			sales_chat_user_id BIGINT UNSIGNED DEFAULT NULL,
			sales_chat_role VARCHAR(30) DEFAULT NULL,
			first_payment_invoice_id BIGINT UNSIGNED NOT NULL,
			first_payment_stage_no INT UNSIGNED NOT NULL DEFAULT 1,
			first_payment_amount DECIMAL(18,2) NOT NULL DEFAULT 0,
			contents_snapshot_json LONGTEXT DEFAULT NULL,
			status VARCHAR(30) NOT NULL DEFAULT 'active',
			portal_invite_status VARCHAR(20) DEFAULT NULL,
			portal_invite_attempts INT UNSIGNED NOT NULL DEFAULT 0,
			portal_invite_attempted_at DATETIME DEFAULT NULL,
			portal_invite_sent_at DATETIME DEFAULT NULL,
			activated_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY source_key (source_key),
			KEY source_invoice_id (source_invoice_id),
			KEY dot_case_id (dot_case_id),
			KEY subscription_product_id (subscription_product_id),
			KEY customer_phone (customer_phone),
			KEY origin_user_id (origin_user_id),
			KEY source_sales_manager_user_id (source_sales_manager_user_id),
			KEY sales_chat_user_id (sales_chat_user_id),
			KEY status (status)
		) {$charset};" );
		$this->ensure_membership_chat_columns();

		dbDelta( "CREATE TABLE {$t['items']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			membership_id BIGINT UNSIGNED NOT NULL,
			content_product_id BIGINT UNSIGNED NOT NULL,
			content_name_snapshot VARCHAR(255) NOT NULL,
			base_credit_snapshot DECIMAL(18,2) DEFAULT NULL,
			upgrade_options_snapshot_json LONGTEXT DEFAULT NULL,
			manager_user_id BIGINT UNSIGNED DEFAULT NULL,
			expert_user_id BIGINT UNSIGNED DEFAULT NULL,
			workflow_status VARCHAR(30) NOT NULL DEFAULT 'waiting',
			cancel_reason TEXT DEFAULT NULL,
			last_contact_at DATETIME DEFAULT NULL,
			contact_attempts INT UNSIGNED NOT NULL DEFAULT 0,
			assigned_at DATETIME DEFAULT NULL,
			status_updated_at DATETIME DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY membership_content (membership_id,content_product_id),
			KEY manager_user_id (manager_user_id),
			KEY expert_user_id (expert_user_id),
			KEY workflow_status (workflow_status)
		) {$charset};" );
		$this->ensure_membership_item_columns();

		dbDelta( "CREATE TABLE {$t['notes']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			membership_item_id BIGINT UNSIGNED NOT NULL,
			author_user_id BIGINT UNSIGNED NOT NULL,
			note TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			KEY membership_item_id (membership_item_id),
			KEY author_user_id (author_user_id),
			KEY created_at (created_at)
		) {$charset};" );

		dbDelta( "CREATE TABLE {$t['messages']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			membership_item_id BIGINT UNSIGNED NOT NULL,
			sender_user_id BIGINT UNSIGNED NOT NULL,
			message_text TEXT DEFAULT NULL,
			attachment_token CHAR(64) DEFAULT NULL,
			attachment_original_name VARCHAR(255) DEFAULT NULL,
			attachment_storage_name VARCHAR(100) DEFAULT NULL,
			attachment_mime VARCHAR(100) DEFAULT NULL,
			attachment_size BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY attachment_token (attachment_token),
			KEY membership_item_id (membership_item_id),
			KEY sender_user_id (sender_user_id),
			KEY created_at (created_at)
		) {$charset};" );

		dbDelta( "CREATE TABLE {$t['reads']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			membership_item_id BIGINT UNSIGNED NOT NULL,
			user_id BIGINT UNSIGNED NOT NULL,
			last_message_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
			read_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY item_user (membership_item_id,user_id),
			KEY user_id (user_id)
		) {$charset};" );

		dbDelta( "CREATE TABLE {$t['invoice_links']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			membership_item_id BIGINT UNSIGNED NOT NULL,
			invoice_id BIGINT UNSIGNED NOT NULL,
			content_product_id BIGINT UNSIGNED NOT NULL,
			line_total DECIMAL(18,2) NOT NULL DEFAULT 0,
			status VARCHAR(30) NOT NULL DEFAULT 'verified',
			linked_by_user_id BIGINT UNSIGNED DEFAULT NULL,
			verified_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY item_invoice_content (membership_item_id,invoice_id,content_product_id),
			UNIQUE KEY invoice_id (invoice_id),
			KEY content_product_id (content_product_id),
			KEY status (status)
		) {$charset};" );

		dbDelta( "CREATE TABLE {$t['rules']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			content_product_id BIGINT UNSIGNED NOT NULL,
			commission_type VARCHAR(20) NOT NULL DEFAULT 'percent',
			commission_value DECIMAL(18,4) NOT NULL DEFAULT 0,
			is_active TINYINT(1) NOT NULL DEFAULT 1,
			created_by BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY content_product_id (content_product_id),
			KEY is_active (is_active)
		) {$charset};" );

		dbDelta( "CREATE TABLE {$t['commissions']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			membership_item_id BIGINT UNSIGNED NOT NULL,
			invoice_id BIGINT UNSIGNED NOT NULL,
			recipient_user_id BIGINT UNSIGNED NOT NULL,
			base_amount DECIMAL(18,2) NOT NULL DEFAULT 0,
			commission_amount DECIMAL(18,2) NOT NULL DEFAULT 0,
			status VARCHAR(30) NOT NULL DEFAULT 'pending',
			wallet_transaction_id BIGINT UNSIGNED DEFAULT NULL,
			reversal_transaction_id BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY item_invoice_recipient (membership_item_id,invoice_id,recipient_user_id),
			KEY invoice_id (invoice_id),
			KEY recipient_user_id (recipient_user_id),
			KEY status (status)
		) {$charset};" );

		dbDelta( "CREATE TABLE {$t['events']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			membership_id BIGINT UNSIGNED DEFAULT NULL,
			membership_item_id BIGINT UNSIGNED DEFAULT NULL,
			actor_user_id BIGINT UNSIGNED DEFAULT NULL,
			event_key VARCHAR(100) NOT NULL,
			details LONGTEXT DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			KEY membership_id (membership_id),
			KEY membership_item_id (membership_item_id),
			KEY event_key (event_key),
			KEY created_at (created_at)
		) {$charset};" );

		dbDelta( "CREATE TABLE {$t['actions']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			membership_item_id BIGINT UNSIGNED NOT NULL,
			actor_user_id BIGINT UNSIGNED NOT NULL,
			action_type VARCHAR(20) NOT NULL,
			fulfillment_mode VARCHAR(20) NOT NULL,
			target_product_id BIGINT UNSIGNED DEFAULT NULL,
			target_product_name VARCHAR(255) DEFAULT NULL,
			base_credit DECIMAL(18,2) NOT NULL DEFAULT 0,
			target_credit DECIMAL(18,2) NOT NULL DEFAULT 0,
			amount DECIMAL(18,2) NOT NULL DEFAULT 0,
			status VARCHAR(30) NOT NULL DEFAULT 'awaiting_payment',
			invoice_id BIGINT UNSIGNED DEFAULT NULL,
			payment_attempts INT UNSIGNED NOT NULL DEFAULT 0,
			last_payment_at DATETIME DEFAULT NULL,
			note TEXT DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			KEY membership_item_id (membership_item_id),
			KEY invoice_id (invoice_id),
			KEY status (status),
			KEY action_type (action_type)
		) {$charset};" );
		$this->ensure_action_api_columns();

		dbDelta( "CREATE TABLE {$t['api_configs']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			content_product_id BIGINT UNSIGNED NOT NULL,
			endpoint_url TEXT NOT NULL,
			api_key TEXT NOT NULL,
			project_code VARCHAR(100) NOT NULL,
			is_active TINYINT(1) NOT NULL DEFAULT 0,
			created_by BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY content_product_id (content_product_id),
			KEY project_code (project_code),
			KEY is_active (is_active)
		) {$charset};" );

		$this->ensure_wallet_tables();
		$this->table_exists_cache = [];
		$ready = true;
		foreach ( $t as $table ) { $ready = $ready && $this->table_exists( $table ); }
		if ( $ready ) {
			// Do not turn historic chat messages into a flood of unread alerts on
			// the first upgrade that introduces notifications. Messages created
			// from this point onward participate in unread counters.
			if ( (string) get_option( 'sn_project_notifications_started_at', '' ) === '' ) {
				update_option( 'sn_project_notifications_started_at', current_time( 'mysql' ), false );
			}
			update_option( 'sn_projects_db_version', self::DB_VERSION, false );
		}
	}

	/** Additive repair for installations upgraded from the original project schema. */
	private function ensure_membership_item_columns(): void {
		global $wpdb;
		$table = $this->tables()['items'];
		unset( $this->table_exists_cache[ $table ] );
		if ( ! $this->table_exists( $table ) ) { return; }
		$columns = array_flip( (array) $wpdb->get_col( "SHOW COLUMNS FROM {$table}", 0 ) );
		$alter = [];
		if ( ! isset( $columns['cancel_reason'] ) ) { $alter[] = 'ADD COLUMN cancel_reason TEXT NULL'; }
		if ( ! isset( $columns['last_contact_at'] ) ) { $alter[] = 'ADD COLUMN last_contact_at DATETIME NULL'; }
		if ( ! isset( $columns['contact_attempts'] ) ) { $alter[] = 'ADD COLUMN contact_attempts INT UNSIGNED NOT NULL DEFAULT 0'; }
		if ( ! isset( $columns['base_credit_snapshot'] ) ) { $alter[] = 'ADD COLUMN base_credit_snapshot DECIMAL(18,2) NULL AFTER content_name_snapshot'; }
		if ( ! isset( $columns['upgrade_options_snapshot_json'] ) ) { $alter[] = 'ADD COLUMN upgrade_options_snapshot_json LONGTEXT NULL AFTER base_credit_snapshot'; }
		if ( $alter ) { $wpdb->query( "ALTER TABLE {$table} " . implode( ', ', $alter ) ); }
	}

	/** Additive repair for remote product/order action metadata. */
	private function ensure_action_api_columns(): void {
		global $wpdb;
		$table = $this->tables()['actions'];
		unset( $this->table_exists_cache[ $table ] );
		if ( ! $this->table_exists( $table ) ) { return; }
		$columns = array_flip( (array) $wpdb->get_col( "SHOW COLUMNS FROM {$table}", 0 ) );
		$alter = [];
		if ( ! isset( $columns['remote_order_id'] ) ) { $alter[] = 'ADD COLUMN remote_order_id BIGINT UNSIGNED NULL'; }
		if ( ! isset( $columns['remote_order_key'] ) ) { $alter[] = 'ADD COLUMN remote_order_key VARCHAR(100) NULL'; }
		if ( ! isset( $columns['remote_payment_url'] ) ) { $alter[] = 'ADD COLUMN remote_payment_url TEXT NULL'; }
		if ( ! isset( $columns['remote_status'] ) ) { $alter[] = 'ADD COLUMN remote_status VARCHAR(30) NULL'; }
		if ( $alter ) { $wpdb->query( "ALTER TABLE {$table} " . implode( ', ', $alter ) ); }
	}

	/** Keep historic ownership immutable while adding routing/chat snapshots. */
	private function ensure_membership_chat_columns(): void {
		global $wpdb; $table = $this->tables()['memberships'];
		unset( $this->table_exists_cache[ $table ] );
		if ( ! $this->table_exists( $table ) ) { return; }
		$columns = array_flip( (array) $wpdb->get_col( "SHOW COLUMNS FROM {$table}", 0 ) );
		$alter = [];
		if ( ! isset( $columns['sales_chat_user_id'] ) ) { $alter[] = 'ADD COLUMN sales_chat_user_id BIGINT UNSIGNED NULL AFTER origin_role'; }
		if ( ! isset( $columns['sales_chat_role'] ) ) { $alter[] = 'ADD COLUMN sales_chat_role VARCHAR(30) NULL AFTER sales_chat_user_id'; }
		if ( $alter ) { $wpdb->query( "ALTER TABLE {$table} " . implode( ', ', $alter ) ); }
		$columns = array_flip( (array) $wpdb->get_col( "SHOW COLUMNS FROM {$table}", 0 ) );
		// Keep this in a separate ALTER so upgrades from every older column layout
		// are accepted without relying on multi-column AFTER ordering.
		if ( ! isset( $columns['source_sales_manager_user_id'] ) ) {
			$wpdb->query( "ALTER TABLE {$table} ADD COLUMN source_sales_manager_user_id BIGINT UNSIGNED NULL AFTER origin_role" );
			$columns = array_flip( (array) $wpdb->get_col( "SHOW COLUMNS FROM {$table}", 0 ) );
		}
		$indexes = array_flip( (array) $wpdb->get_col( "SHOW INDEX FROM {$table}", 2 ) );
		if ( isset( $columns['sales_chat_user_id'] ) ) {
			$wpdb->query( "UPDATE {$table} SET sales_chat_user_id=origin_user_id WHERE sales_chat_user_id IS NULL OR sales_chat_user_id=0" );
			if ( ! isset( $indexes['sales_chat_user_id'] ) ) { $wpdb->query( "ALTER TABLE {$table} ADD INDEX sales_chat_user_id (sales_chat_user_id)" ); }
		}
		if ( isset( $columns['source_sales_manager_user_id'] ) && ! isset( $indexes['source_sales_manager_user_id'] ) ) { $wpdb->query( "ALTER TABLE {$table} ADD INDEX source_sales_manager_user_id (source_sales_manager_user_id)" ); }
	}

	private function ensure_roles_and_hr(): void {
		// Releases before 2.0.44 exposed a second, parallel project hierarchy.
		// Consolidate those accounts into the operations hierarchy, while retaining
		// historical item/user ids and legacy tables for read compatibility.
		foreach ( [
			'sn_operations_sales_manager' => [ 'مدیر فروش عملیات', 'sn_manage_operations_sales' ],
			'sn_operations_sales_expert' => [ 'کارشناس فروش عملیات', 'sn_work_operations_sales' ],
		] as $role_key => $role_def ) {
			if ( ! get_role( $role_key ) ) { add_role( $role_key, $role_def[0], [ 'read' => true, $role_def[1] => true ] ); }
		}
		$role_migrations = [
			'sn_project_manager' => 'sn_operations_sales_manager',
			'sn_project_expert' => 'sn_operations_sales_expert',
		];
		foreach ( $role_migrations as $legacy_role => $target_role ) {
			foreach ( get_users( [ 'role' => $legacy_role, 'number' => -1, 'fields' => 'all' ] ) as $user ) {
				if ( ! $user instanceof WP_User ) { continue; }
				$user->add_role( $target_role );
				$user->remove_role( $legacy_role );
			}
			remove_role( $legacy_role );
		}

		global $wpdb;
		$positions = $wpdb->prefix . 'sn_hr_positions';
		$profiles = $wpdb->prefix . 'sn_hr_profiles';
		$mappings = $wpdb->prefix . 'sn_hr_position_role_mappings';
		if ( ! $this->table_exists( $positions ) || ! $this->table_exists( $profiles ) || ! $this->table_exists( $mappings ) ) { return; }

		$position_columns = array_flip( (array) $wpdb->get_col( "SHOW COLUMNS FROM {$positions}", 0 ) );
		foreach ( [
			'operations_sales_manager' => [ 'مدیر فروش عملیات', 'operations_sales_manager', 94 ],
			'operations_sales_expert' => [ 'کارشناس فروش عملیات', 'operations_sales_expert', 96 ],
		] as $slug => $row ) {
			if ( (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$positions} WHERE slug=%s LIMIT 1", $slug ) ) ) { continue; }
			$data = [ 'slug' => $slug, 'label' => $row[0], 'panel_key' => $row[1], 'sort_order' => $row[2], 'is_active' => 1, 'is_system' => 1 ];
			$wpdb->insert( $positions, array_intersect_key( $data, $position_columns ) );
		}

		foreach ( [
			'project_manager' => 'operations_sales_manager',
			'project_expert' => 'operations_sales_expert',
		] as $legacy_slug => $target_slug ) {
			$legacy_id = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$positions} WHERE slug=%s LIMIT 1", $legacy_slug ) );
			$target_id = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$positions} WHERE slug=%s LIMIT 1", $target_slug ) );
			if ( $legacy_id > 0 && $target_id > 0 ) {
				$wpdb->update( $profiles, [ 'position_id' => $target_id ], [ 'position_id' => $legacy_id ] );
				$wpdb->update( $positions, [ 'is_active' => 0 ], [ 'id' => $legacy_id ] );
			}
		}
		foreach ( array_keys( $role_migrations ) as $legacy_role ) {
			$wpdb->update( $mappings, [ 'is_active' => 0 ], [ 'legacy_role' => $legacy_role ] );
		}
		update_option( 'sn_projects_hr_version', self::DB_VERSION, false );
	}

	private function ensure_wallet_tables(): void {
		global $wpdb;
		require_once ABSPATH . 'wp-admin/includes/upgrade.php';
		$charset = $wpdb->get_charset_collate();
		dbDelta( "CREATE TABLE {$wpdb->prefix}sn_wallets (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			user_id BIGINT UNSIGNED NOT NULL,
			wallet_type VARCHAR(30) NOT NULL DEFAULT 'seller',
			balance DECIMAL(18,2) NOT NULL DEFAULT 0,
			total_credit DECIMAL(18,2) NOT NULL DEFAULT 0,
			total_debit DECIMAL(18,2) NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY user_wallet (user_id,wallet_type),
			KEY wallet_type (wallet_type)
		) {$charset};" );
		dbDelta( "CREATE TABLE {$wpdb->prefix}sn_wallet_transactions (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			wallet_id BIGINT UNSIGNED NOT NULL,
			user_id BIGINT UNSIGNED NOT NULL,
			wallet_type VARCHAR(30) NOT NULL DEFAULT 'seller',
			invoice_id BIGINT UNSIGNED DEFAULT NULL,
			lead_id BIGINT UNSIGNED DEFAULT NULL,
			amount DECIMAL(18,2) NOT NULL DEFAULT 0,
			direction VARCHAR(10) NOT NULL DEFAULT 'credit',
			type VARCHAR(60) NOT NULL DEFAULT 'commission',
			status VARCHAR(30) NOT NULL DEFAULT 'approved',
			description TEXT DEFAULT NULL,
			meta LONGTEXT DEFAULT NULL,
			created_by BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			KEY wallet_id (wallet_id),
			KEY user_id (user_id),
			KEY invoice_id (invoice_id),
			KEY type (type),
			KEY created_at (created_at)
		) {$charset};" );
		unset( $this->table_exists_cache[ $wpdb->prefix . 'sn_wallets' ], $this->table_exists_cache[ $wpdb->prefix . 'sn_wallet_transactions' ] );
	}

	private function product_exists( int $product_id ): bool {
		if ( $product_id < 1 || 'publish' !== get_post_status( $product_id ) ) { return false; }
		if ( (string) get_post_meta( $product_id, '_sn_enabled', true ) !== '1' ) { return false; }
		return function_exists( 'wc_get_product' ) && (bool) wc_get_product( $product_id );
	}

	public function subscription_product_ids(): array {
		global $wpdb; $t = $this->tables();
		if ( null !== $this->subscription_ids_cache ) { return $this->subscription_ids_cache; }
		if ( ! $this->table_exists( $t['subscriptions'] ) ) { return []; }
		$this->subscription_ids_cache = array_map( 'intval', (array) $wpdb->get_col( "SELECT subscription_product_id FROM {$t['subscriptions']} WHERE is_active=1 ORDER BY subscription_product_id" ) );
		return $this->subscription_ids_cache;
	}

	public function content_product_ids(): array {
		global $wpdb; $t = $this->tables();
		if ( null !== $this->content_ids_cache ) { return $this->content_ids_cache; }
		if ( ! $this->table_exists( $t['contents'] ) || ! $this->table_exists( $t['subscriptions'] ) ) { return []; }
		$this->content_ids_cache = array_map( 'intval', (array) $wpdb->get_col( "SELECT DISTINCT c.content_product_id FROM {$t['contents']} c INNER JOIN {$t['subscriptions']} s ON s.subscription_product_id=c.subscription_product_id AND s.is_active=1 WHERE c.is_active=1 ORDER BY c.content_product_id" ) );
		return $this->content_ids_cache;
	}

	public function is_subscription_product( int $product_id ): bool {
		return in_array( $product_id, $this->subscription_product_ids(), true );
	}

	public function is_content_product( int $product_id ): bool {
		return in_array( $product_id, $this->content_product_ids(), true );
	}

	private function actor_is_sales_user( int $user_id ): bool {
		$user = get_user_by( 'id', $user_id );
		if ( ! $user instanceof WP_User ) { return false; }
		$roles = (array) $user->roles;
		if ( array_intersect( $roles, [ 'sn_seller', 'sn_converter' ] ) ) { return true; }
		return in_array( $this->position_for_user( $user_id ), [ 'seller', 'converter' ], true );
	}

	public function filter_sellable_products( array $products, int $user_id = 0 ): array {
		if ( ! class_exists( 'SN_Seller_Flow' ) ) { return $products; }
		return array_values( array_filter( $products, static function ( $product ) {
			return SN_Seller_Flow::instance()->standalone_sale_visible( absint( $product['id'] ?? 0 ) );
		} ) );
	}

	public function validate_invoice_products( array $product_ids, int $actor_user_id ) {
		$product_ids = array_values( array_unique( array_filter( array_map( 'absint', $product_ids ) ) ) );
		if ( count( array_intersect( $product_ids, $this->subscription_product_ids() ) ) > 1 ) {
			return new WP_Error( 'sn_project_one_subscription', 'در هر فاکتور فقط یک محصول اشتراک قابل انتخاب است.' );
		}
		return true;
	}

	public function filter_content_purchasable( bool $purchasable, $product ): bool {
		$product_id = is_object( $product ) && method_exists( $product, 'get_id' ) ? (int) $product->get_id() : 0;
		if ( $product_id > 0 && function_exists( 'wc_get_product' ) ) {
			$parent_id = is_object( $product ) && method_exists( $product, 'get_parent_id' ) ? (int) $product->get_parent_id() : 0;
			if ( $this->is_content_product( $product_id ) || ( $parent_id > 0 && $this->is_content_product( $parent_id ) ) ) { return false; }
		}
		return $purchasable;
	}

	public function filter_content_visible( bool $visible, int $product_id ): bool {
		if ( $this->is_content_product( $product_id ) ) { return false; }
		if ( function_exists( 'wc_get_product' ) ) {
			$product = wc_get_product( $product_id );
			$parent_id = $product && method_exists( $product, 'get_parent_id' ) ? (int) $product->get_parent_id() : 0;
			if ( $parent_id > 0 && $this->is_content_product( $parent_id ) ) { return false; }
		}
		return $visible;
	}

	public function validate_content_add_to_cart( bool $passed, int $product_id, int $quantity ): bool {
		if ( $this->is_content_product( $product_id ) ) {
			if ( function_exists( 'wc_add_notice' ) ) { wc_add_notice( 'این محصول فقط یکی از محتویات اشتراک است و جداگانه قابل خرید نیست.', 'error' ); }
			return false;
		}
		return $passed;
	}

	private function subscription_contents( int $subscription_product_id ): array {
		global $wpdb; $t = $this->tables();
		if ( ! $this->is_subscription_product( $subscription_product_id ) ) { return []; }
		$rows = $wpdb->get_results( $wpdb->prepare(
			"SELECT c.content_product_id,c.sort_order,COALESCE(mp.manager_user_id,0) manager_user_id FROM {$t['contents']} c LEFT JOIN {$t['managers']} mp ON mp.content_product_id=c.content_product_id WHERE c.subscription_product_id=%d AND c.is_active=1 ORDER BY c.sort_order,c.id",
			$subscription_product_id
		), ARRAY_A ) ?: [];
		foreach ( $rows as &$row ) {
			$row['content_product_id'] = (int) $row['content_product_id'];
			$row['manager_user_id'] = (int) $row['manager_user_id'];
			$row['name'] = (string) get_the_title( $row['content_product_id'] );
		}
		return $rows;
	}

	private function position_for_user( int $user_id ): string {
		global $wpdb;
		if ( $user_id < 1 ) { return ''; }
		$profiles = $wpdb->prefix . 'sn_hr_profiles'; $positions = $wpdb->prefix . 'sn_hr_positions';
		if ( $this->table_exists( $profiles ) && $this->table_exists( $positions ) ) {
			$slug = (string) $wpdb->get_var( $wpdb->prepare( "SELECT pos.slug FROM {$profiles} hp INNER JOIN {$positions} pos ON pos.id=hp.position_id WHERE hp.user_id=%d AND hp.is_active=1 LIMIT 1", $user_id ) );
			if ( $slug !== '' ) { return $slug; }
		}
		$user = get_user_by( 'id', $user_id );
		$roles = $user instanceof WP_User ? (array) $user->roles : [];
		if ( in_array( 'sn_project_manager', $roles, true ) ) { return 'project_manager'; }
		if ( in_array( 'sn_project_expert', $roles, true ) ) { return 'project_expert'; }
		if ( in_array( 'sn_operations_sales_manager', $roles, true ) ) { return 'operations_sales_manager'; }
		if ( in_array( 'sn_operations_sales_supervisor', $roles, true ) ) { return 'operations_sales_supervisor'; }
		if ( in_array( 'sn_operations_sales_expert', $roles, true ) ) { return 'operations_sales_expert'; }
		if ( in_array( 'sn_operations_executive_manager', $roles, true ) ) { return 'operations_executive_manager'; }
		if ( in_array( 'sn_operations_execution_expert', $roles, true ) ) { return 'operations_execution_expert'; }
		if ( in_array( 'sn_sales_deputy', $roles, true ) ) { return 'sales_deputy'; }
		if ( in_array( 'sn_sales_manager', $roles, true ) || in_array( 'sas_sales_manager', $roles, true ) ) { return 'sales_manager'; }
		if ( in_array( 'sn_senior_supervisor', $roles, true ) || in_array( 'sas_senior_supervisor', $roles, true ) ) { return 'senior_supervisor'; }
		if ( in_array( 'sn_supervisor', $roles, true ) || in_array( 'sas_supervisor', $roles, true ) ) { return 'supervisor'; }
		if ( in_array( 'sn_converter', $roles, true ) ) { return 'converter'; }
		if ( in_array( 'sn_seller', $roles, true ) || in_array( 'sas_employee', $roles, true ) ) { return 'seller'; }
		return '';
	}

	/** Resolve the sales manager above a sales-side user without mutating HR. */
	private function sales_manager_for_user( int $user_id ): int {
		if ( $user_id < 1 ) { return 0; }
		if ( $this->position_for_user( $user_id ) === 'sales_manager' ) { return $user_id; }
		if ( class_exists( 'SN_Seller_Flow' ) && method_exists( SN_Seller_Flow::instance(), 'sales_manager_for_user' ) ) {
			$resolved = (int) SN_Seller_Flow::instance()->sales_manager_for_user( $user_id );
			if ( $resolved > 0 ) { return $resolved; }
		}

		global $wpdb;
		$profiles = $wpdb->prefix . 'sn_hr_profiles';
		$positions = $wpdb->prefix . 'sn_hr_positions';
		$assignments = $wpdb->prefix . 'sn_hr_assignments';
		if ( $this->table_exists( $profiles ) && $this->table_exists( $positions ) && $this->table_exists( $assignments ) ) {
			$current_user_id = $user_id; $seen_profiles = [];
			for ( $depth = 0; $depth < 20 && $current_user_id > 0; $depth++ ) {
				$profile_id = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$profiles} WHERE user_id=%d ORDER BY is_active DESC,id DESC LIMIT 1", $current_user_id ) );
				if ( $profile_id < 1 || isset( $seen_profiles[ $profile_id ] ) ) { break; }
				$seen_profiles[ $profile_id ] = true;
				$parent = $wpdb->get_row( $wpdb->prepare(
					"SELECT p.id,p.user_id,pos.slug position_slug FROM {$assignments} a INNER JOIN {$profiles} p ON p.id=a.parent_profile_id LEFT JOIN {$positions} pos ON pos.id=p.position_id WHERE a.child_profile_id=%d AND a.is_current=1 AND (a.relationship_type='reports_to' OR a.relationship_type='' OR a.relationship_type IS NULL) ORDER BY a.id DESC LIMIT 1",
					$profile_id
				), ARRAY_A );
				if ( ! $parent ) { break; }
				if ( (string) ( $parent['position_slug'] ?? '' ) === 'sales_manager' ) { return (int) $parent['user_id']; }
				$current_user_id = (int) ( $parent['user_id'] ?? 0 );
			}
		}

		$current = $user_id; $seen_users = [];
		for ( $depth = 0; $depth < 20 && $current > 0; $depth++ ) {
			if ( isset( $seen_users[ $current ] ) ) { break; }
			$seen_users[ $current ] = true;
			if ( $this->position_for_user( $current ) === 'sales_manager' ) { return $current; }
			$direct_manager = absint( get_user_meta( $current, 'sn_sales_manager_id', true ) );
			if ( $direct_manager > 0 && $this->position_for_user( $direct_manager ) === 'sales_manager' ) { return $direct_manager; }
			$current = absint( get_user_meta( $current, 'sn_supervisor_id', true ) );
		}
		return 0;
	}

	/**
	 * Sales branch ownership is based on the original seller, not on the user
	 * who later converts or issues the payment invoice.
	 */
	private function resolve_source_sales_manager( object $invoice, int $fallback_user_id = 0 ): int {
		$candidates = [
			(int) ( $invoice->original_seller_id ?? 0 ),
			(int) ( $invoice->commission_seller_id ?? 0 ),
			// For a Dot conversion this is the immutable case seller. It must win
			// over the converter/supervisor who may have issued the payment invoice.
			$fallback_user_id,
			(int) ( $invoice->seller_id ?? 0 ),
		];
		foreach ( array_values( array_unique( array_filter( $candidates ) ) ) as $candidate ) {
			$manager_id = $this->sales_manager_for_user( (int) $candidate );
			if ( $manager_id > 0 ) { return $manager_id; }
		}
		return 0;
	}

	/** Exact sales-manager override, followed by the legacy/default manager. */
	private function manager_for_sales_branch( int $content_product_id, int $sales_manager_user_id, int $default_manager_user_id ): int {
		global $wpdb; $table = $this->tables()['manager_routes'];
		if ( $content_product_id > 0 && $sales_manager_user_id > 0 && $this->table_exists( $table ) ) {
			$routed = (int) $wpdb->get_var( $wpdb->prepare(
				"SELECT manager_user_id FROM {$table} WHERE content_product_id=%d AND sales_manager_user_id=%d LIMIT 1",
				$content_product_id,
				$sales_manager_user_id
			) );
			if ( $routed > 0 ) { return $routed; }
		}
		return max( 0, $default_manager_user_id );
	}

	private function apply_sales_branch_routes( array $contents, int $sales_manager_user_id ): array {
		foreach ( $contents as &$content ) {
			$content_id = (int) ( $content['content_product_id'] ?? 0 );
			$default_manager_id = (int) ( $content['manager_user_id'] ?? 0 );
			$content['default_manager_user_id'] = $default_manager_id;
			$content['source_sales_manager_user_id'] = $sales_manager_user_id;
			$content['manager_user_id'] = $this->manager_for_sales_branch( $content_id, $sales_manager_user_id, $default_manager_id );
			$content['manager_route'] = $content['manager_user_id'] !== $default_manager_id ? 'sales_manager_override' : 'default';
		}
		unset( $content );
		return $contents;
	}

	private function origin_role( int $user_id ): string {
		$position = $this->position_for_user( $user_id );
		return $position === 'converter' ? 'converter' : 'seller';
	}

	private function resolve_sales_chat_user( object $invoice, int $fallback_user_id ): int {
		// Chat ownership belongs to the latest sales-side actor who issued the
		// subscription payment. In staged payments the current stage issuer is
		// therefore preferred over the original invoice issuer.
		$candidates = [
			(int) ( $invoice->current_stage_issued_by_user_id ?? 0 ),
			(int) ( $invoice->issued_by_user_id ?? 0 ),
			(int) ( $invoice->seller_id ?? 0 ),
			$fallback_user_id,
		];
		foreach ( array_values( array_unique( array_filter( $candidates ) ) ) as $candidate ) {
			$position = $this->position_for_user( (int) $candidate );
			if ( in_array( $position, [ 'seller', 'converter', 'supervisor', 'senior_supervisor', 'sales_manager', 'sales_deputy' ], true ) ) { return (int) $candidate; }
		}
		return $fallback_user_id;
	}

	/**
	 * Resolve the immutable sales-side owner of a project conversation.
	 * Supervisors/admins may issue an invoice on behalf of a seller, so the
	 * invoice owner is preferred over the user who clicked the issue button.
	 */
	private function resolve_origin_user( object $invoice, int $preferred_user_id = 0 ): int {
		$candidates = [
			$preferred_user_id,
			(int) ( $invoice->seller_id ?? 0 ),
			(int) ( $invoice->original_seller_id ?? 0 ),
			(int) ( $invoice->commission_seller_id ?? 0 ),
			(int) ( $invoice->issued_by_user_id ?? 0 ),
			(int) ( $invoice->current_stage_issued_by_user_id ?? 0 ),
		];
		foreach ( array_values( array_unique( array_filter( $candidates ) ) ) as $candidate ) {
			if ( $this->actor_is_sales_user( (int) $candidate ) ) { return (int) $candidate; }
		}
		return 0;
	}

	private function normalize_phone( string $phone ): string {
		return class_exists( 'SN_Helpers' ) ? SN_Helpers::normalize_mobile( $phone ) : preg_replace( '/\D+/', '', $phone );
	}

	private function event( int $membership_id, int $item_id, string $key, array $details = [], int $actor_id = 0 ): void {
		global $wpdb; $t = $this->tables();
		$wpdb->insert( $t['events'], [
			'membership_id' => $membership_id ?: null,
			'membership_item_id' => $item_id ?: null,
			'actor_user_id' => $actor_id ?: ( get_current_user_id() ?: null ),
			'event_key' => sanitize_key( $key ),
			'details' => wp_json_encode( $details, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ),
			'created_at' => current_time( 'mysql' ),
		] );
	}

	private function invoice_items( int $invoice_id ): array {
		global $wpdb;
		return $wpdb->get_results( $wpdb->prepare( "SELECT * FROM {$wpdb->prefix}sn_invoice_items WHERE invoice_id=%d ORDER BY id", $invoice_id ), ARRAY_A ) ?: [];
	}

	/** Activate a paid Product*, with or without a parent subscription. */
	private function activate_single_content_from_invoice( int $invoice_id, object $invoice, array $context, array $item ): bool {
		global $wpdb; $t = $this->tables();
		$content_id = absint( $item['product_id'] ?? 0 );
		$item_type = sanitize_key( (string) ( $item['product_type'] ?? '' ) );
		if ( $content_id < 1 || ( ! in_array( $item_type, [ 'product_star', 'subscription_star' ], true ) && ! $this->is_product_star_definition( $content_id ) ) ) { return false; }
		$parent = $wpdb->get_row( $wpdb->prepare(
			"SELECT c.subscription_product_id,COALESCE(mp.manager_user_id,0) manager_user_id FROM {$t['contents']} c INNER JOIN {$t['subscriptions']} s ON s.subscription_product_id=c.subscription_product_id AND s.is_active=1 LEFT JOIN {$t['managers']} mp ON mp.content_product_id=c.content_product_id WHERE c.content_product_id=%d AND c.is_active=1 ORDER BY c.sort_order,c.id LIMIT 1",
			$content_id
		) );
		if ( ! $parent ) {
			// Standalone cards keep subscription id zero; their own product and snapshots remain authoritative.
			$default_manager = (int) $wpdb->get_var( $wpdb->prepare( "SELECT manager_user_id FROM {$t['managers']} WHERE content_product_id=%d LIMIT 1", $content_id ) );
			$parent = (object) [ 'subscription_product_id' => 0, 'manager_user_id' => $default_manager ];
		}
		$origin_user_id = $this->resolve_origin_user( $invoice );
		if ( $origin_user_id < 1 ) { return false; }
		$source_sales_manager_user_id = $this->resolve_source_sales_manager( $invoice, $origin_user_id );
		$sales_chat_user_id = $this->resolve_sales_chat_user( $invoice, $origin_user_id );
		$subscription_id = (int) $parent->subscription_product_id;
		$source_key = 'product_star:' . $invoice_id . ':' . $content_id;
		$existing = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$t['memberships']} WHERE source_key=%s LIMIT 1", $source_key ) );
		if ( $existing > 0 ) {
			$wpdb->update( $t['memberships'], [ 'sales_chat_user_id' => $sales_chat_user_id ?: $origin_user_id, 'sales_chat_role' => $this->position_for_user( $sales_chat_user_id ?: $origin_user_id ), 'updated_at' => current_time( 'mysql' ) ], [ 'id' => $existing ] );
			return true;
		}
		$stage_no = max( 1, absint( $context['stage_no'] ?? $invoice->current_payment_stage ?? 1 ) );
		$stage_amount = max( 0, (float) ( $context['stage_amount'] ?? 0 ) );
		$content = [
			'content_product_id' => $content_id,
			'default_manager_user_id' => (int) $parent->manager_user_id,
			'source_sales_manager_user_id' => $source_sales_manager_user_id,
			'manager_user_id' => $this->manager_for_sales_branch( $content_id, $source_sales_manager_user_id, (int) $parent->manager_user_id ),
			'name' => sanitize_text_field( (string) ( $item['product_name'] ?? get_the_title( $content_id ) ) ),
			'unit_price' => (float) ( $item['unit_price'] ?? 0 ),
			'sale_mode' => 'single_product_star',
		];
		$content['base_credit'] = $this->product_credit( $content_id );
		$content['upgrade_options'] = $this->product_upgrade_rules( $content_id );
		$content['manager_route'] = $content['manager_user_id'] !== $content['default_manager_user_id'] ? 'sales_manager_override' : 'default';
		$now = current_time( 'mysql' );
		$wpdb->query( 'START TRANSACTION' );
		$inserted = $wpdb->insert( $t['memberships'], [
			'source_key' => $source_key, 'source_type' => 'product_star', 'source_invoice_id' => $invoice_id,
			'dot_case_id' => null, 'subscription_product_id' => $subscription_id,
			'subscription_name_snapshot' => 'فروش تکی: ' . $content['name'],
			'customer_wp_id' => ! empty( $invoice->customer_wp_id ) ? (int) $invoice->customer_wp_id : null,
			'customer_name' => sanitize_text_field( (string) ( $invoice->customer_name ?? '' ) ),
			'customer_phone' => $this->normalize_phone( (string) ( $invoice->customer_phone ?? '' ) ),
			'origin_user_id' => $origin_user_id, 'origin_role' => $this->origin_role( $origin_user_id ),
			'source_sales_manager_user_id' => $source_sales_manager_user_id ?: null,
			'sales_chat_user_id' => $sales_chat_user_id ?: $origin_user_id, 'sales_chat_role' => $this->position_for_user( $sales_chat_user_id ?: $origin_user_id ),
			'first_payment_invoice_id' => $invoice_id, 'first_payment_stage_no' => $stage_no,
			'first_payment_amount' => $stage_amount, 'contents_snapshot_json' => wp_json_encode( [ $content ], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ),
			'status' => 'active', 'activated_at' => $now, 'created_at' => $now, 'updated_at' => $now,
		] );
		if ( ! $inserted ) { $wpdb->query( 'ROLLBACK' ); return false; }
		$membership_id = (int) $wpdb->insert_id;
		$inserted_item = $wpdb->insert( $t['items'], [
			'membership_id' => $membership_id, 'content_product_id' => $content_id,
			'content_name_snapshot' => $content['name'],
			'base_credit_snapshot' => $content['base_credit'],
			'upgrade_options_snapshot_json' => wp_json_encode( $content['upgrade_options'], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ),
			'manager_user_id' => $content['manager_user_id'] ?: null,
			'workflow_status' => 'waiting', 'created_at' => $now, 'updated_at' => $now,
		] );
		if ( ! $inserted_item ) { $wpdb->query( 'ROLLBACK' ); return false; }
		$wpdb->query( 'COMMIT' );
		$this->event( $membership_id, (int) $wpdb->insert_id, 'product_star_activated', [ 'invoice_id' => $invoice_id, 'content_product_id' => $content_id, 'subscription_product_id' => $subscription_id, 'price' => $content['unit_price'], 'source_sales_manager_user_id' => $source_sales_manager_user_id, 'manager_user_id' => $content['manager_user_id'], 'manager_route' => $content['manager_route'] ], $origin_user_id );
		do_action( 'sn_project_membership_activated', $membership_id, $invoice_id, $invoice );
		return true;
	}

	public function on_invoice_stage_approved( int $invoice_id, $invoice, array $stage_context = [] ): void {
		if ( ! is_object( $invoice ) ) {
			global $wpdb; $invoice = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$wpdb->prefix}sn_invoices WHERE id=%d", $invoice_id ) );
		}
		if ( ! $invoice ) { return; }
		if ( $this->is_project_action_invoice( $invoice_id ) ) {
			if ( ! empty( $stage_context['completed'] ) ) { $this->mark_action_invoice_paid( $invoice_id ); }
			return;
		}
		// A project case belongs in Biavin only after the invoice is fully paid.
		// Keep this hook for restoring reversed links, but do not create a new
		// membership for an approved instalment/prepayment.
		if ( empty( $stage_context['completed'] ) ) {
			$this->restore_reversed_invoice_links( $invoice_id, $invoice );
			return;
		}
		$this->activate_from_invoice( $invoice_id, $invoice, $stage_context );
		$this->restore_reversed_invoice_links( $invoice_id, $invoice );
	}

	public function on_invoice_paid( int $invoice_id, $invoice ): void {
		if ( ! is_object( $invoice ) ) {
			global $wpdb; $invoice = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$wpdb->prefix}sn_invoices WHERE id=%d", $invoice_id ) );
		}
		if ( ! $invoice ) { return; }
		$this->mark_action_invoice_paid( $invoice_id );
		if ( $this->is_project_action_invoice( $invoice_id ) ) { return; }
		$this->activate_from_invoice( $invoice_id, $invoice, [ 'stage_no' => (int) ( $invoice->current_payment_stage ?? 1 ), 'stage_amount' => (float) ( $invoice->current_due_amount ?? $invoice->product_price ?? 0 ) ] );
		$this->restore_reversed_invoice_links( $invoice_id, $invoice );
	}

	private function activate_from_invoice( int $invoice_id, object $invoice, array $context ): void {
		global $wpdb; $t = $this->tables();
		$invoice_items = $this->invoice_items( $invoice_id );
		$activated_single_cards = false;
		foreach ( $invoice_items as $invoice_item ) {
			$item_type = sanitize_key( (string) ( $invoice_item['product_type'] ?? '' ) );
			if ( in_array( $item_type, [ 'product_star', 'subscription_star' ], true ) || $this->is_content_product( (int) ( $invoice_item['product_id'] ?? 0 ) ) ) {
				// One invoice may contain more than one Product*. Create every paid
				// card idempotently; a line that is actually a subscription simply
				// returns false here and is handled by the subscription path below.
				$activated_single_cards = $this->activate_single_content_from_invoice( $invoice_id, $invoice, $context, $invoice_item ) || $activated_single_cards;
			}
		}
		if ( $activated_single_cards ) { return; }
		$dot_link_table = $wpdb->prefix . 'sn_dot_invoice_links';
		$dot_case_table = $wpdb->prefix . 'sn_dot_cases';
		$subscription_id = 0; $source_key = ''; $source_type = 'direct'; $dot_case_id = 0; $dot_converter_id = 0; $dot_seller_id = 0;
		if ( $this->table_exists( $dot_link_table ) && $this->table_exists( $dot_case_table ) ) {
			$dot = $wpdb->get_row( $wpdb->prepare( "SELECT dl.case_id,c.selected_option_product_id,c.converter_id,c.seller_id FROM {$dot_link_table} dl INNER JOIN {$dot_case_table} c ON c.id=dl.case_id WHERE dl.invoice_id=%d AND dl.flow_kind='conversion_payment' LIMIT 1", $invoice_id ) );
			if ( $dot ) {
				$subscription_id = (int) ( $dot->selected_option_product_id ?? 0 );
				$dot_case_id = (int) $dot->case_id;
				$dot_converter_id = (int) ( $dot->converter_id ?? 0 );
				$dot_seller_id = (int) ( $dot->seller_id ?? 0 );
				$source_type = 'dot';
				$source_key = 'dot:' . $dot_case_id . ':' . $subscription_id;
			}
		}
		if ( ! $subscription_id ) {
			foreach ( $invoice_items as $item ) {
				$pid = (int) ( $item['product_id'] ?? 0 );
				if ( $this->is_subscription_product( $pid ) ) { $subscription_id = $pid; break; }
			}
			if ( ! $subscription_id && $this->is_subscription_product( (int) ( $invoice->product_id ?? 0 ) ) ) { $subscription_id = (int) $invoice->product_id; }
			$source_key = 'direct:' . $invoice_id . ':' . $subscription_id;
		}
		if ( $subscription_id < 1 || ! $this->is_subscription_product( $subscription_id ) ) { return; }
		$origin_user_id = $this->resolve_origin_user( $invoice, $dot_converter_id );
		$source_sales_manager_user_id = $this->resolve_source_sales_manager( $invoice, $dot_seller_id ?: $origin_user_id );
		$sales_chat_user_id = $this->resolve_sales_chat_user( $invoice, $origin_user_id );
		$contents = $this->apply_sales_branch_routes( $this->subscription_contents( $subscription_id ), $source_sales_manager_user_id );
		foreach ( $contents as &$content ) {
			$content_id = (int) ( $content['content_product_id'] ?? 0 );
			$content['base_credit'] = $this->product_credit( $content_id );
			$content['upgrade_options'] = $this->product_upgrade_rules( $content_id );
		}
		unset( $content );
		if ( ! $contents || $origin_user_id < 1 ) { return; }
		$now = current_time( 'mysql' );
		$stage_no = max( 1, absint( $context['stage_no'] ?? $invoice->current_payment_stage ?? 1 ) );
		$stage_amount = max( 0, (float) ( $context['stage_amount'] ?? 0 ) );
		if ( $stage_amount <= 0 ) {
			$stage_amount = (float) $wpdb->get_var( $wpdb->prepare( "SELECT requested_amount FROM {$wpdb->prefix}sn_invoice_payment_stages WHERE invoice_id=%d AND stage_no=%d LIMIT 1", $invoice_id, $stage_no ) );
		}
		$existing = $wpdb->get_row( $wpdb->prepare( "SELECT id,status FROM {$t['memberships']} WHERE source_key=%s LIMIT 1", $source_key ) );
		if ( $existing ) {
			$wpdb->update( $t['memberships'], [ 'sales_chat_user_id' => $sales_chat_user_id ?: $origin_user_id, 'sales_chat_role' => $this->position_for_user( $sales_chat_user_id ?: $origin_user_id ), 'updated_at' => $now ], [ 'id' => (int) $existing->id ] );
			if ( (string) $existing->status === 'suspended' ) {
				$wpdb->update( $t['memberships'], [
					'source_invoice_id' => $invoice_id,
					'first_payment_invoice_id' => $invoice_id,
					'first_payment_stage_no' => $stage_no,
					'first_payment_amount' => $stage_amount,
					'status' => 'active',
					'activated_at' => $now,
					'updated_at' => $now,
				], [ 'id' => (int) $existing->id, 'status' => 'suspended' ] );
				$this->event( (int) $existing->id, 0, 'membership_reactivated', [ 'invoice_id' => $invoice_id ], $origin_user_id );
			}
			return;
		}
		$wpdb->query( 'START TRANSACTION' );
		$inserted = $wpdb->insert( $t['memberships'], [
			'source_key' => $source_key,
			'source_type' => $source_type,
			'source_invoice_id' => $invoice_id,
			'dot_case_id' => $dot_case_id ?: null,
			'subscription_product_id' => $subscription_id,
			'subscription_name_snapshot' => (string) get_the_title( $subscription_id ),
			'customer_wp_id' => ! empty( $invoice->customer_wp_id ) ? (int) $invoice->customer_wp_id : null,
			'customer_name' => sanitize_text_field( (string) ( $invoice->customer_name ?? '' ) ),
			'customer_phone' => $this->normalize_phone( (string) ( $invoice->customer_phone ?? '' ) ),
			'origin_user_id' => $origin_user_id,
			'origin_role' => $this->origin_role( $origin_user_id ),
			'source_sales_manager_user_id' => $source_sales_manager_user_id ?: null,
			'sales_chat_user_id' => $sales_chat_user_id ?: $origin_user_id,
			'sales_chat_role' => $this->position_for_user( $sales_chat_user_id ?: $origin_user_id ),
			'first_payment_invoice_id' => $invoice_id,
			'first_payment_stage_no' => $stage_no,
			'first_payment_amount' => $stage_amount,
			'contents_snapshot_json' => wp_json_encode( $contents, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ),
			'status' => 'active', 'activated_at' => $now, 'created_at' => $now, 'updated_at' => $now,
		] );
		if ( ! $inserted ) { $wpdb->query( 'ROLLBACK' ); return; }
		$membership_id = (int) $wpdb->insert_id;
		foreach ( $contents as $content ) {
			$ok = $wpdb->insert( $t['items'], [
				'membership_id' => $membership_id,
				'content_product_id' => (int) $content['content_product_id'],
				'content_name_snapshot' => sanitize_text_field( (string) $content['name'] ),
				'base_credit_snapshot' => max( 0, (float) ( $content['base_credit'] ?? 0 ) ),
				'upgrade_options_snapshot_json' => wp_json_encode( (array) ( $content['upgrade_options'] ?? [] ), JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ),
				'manager_user_id' => ! empty( $content['manager_user_id'] ) ? (int) $content['manager_user_id'] : null,
				'workflow_status' => 'waiting', 'created_at' => $now, 'updated_at' => $now,
			] );
			if ( ! $ok ) { $wpdb->query( 'ROLLBACK' ); return; }
		}
		$wpdb->query( 'COMMIT' );
		$this->event( $membership_id, 0, 'membership_activated', [ 'source_type' => $source_type, 'invoice_id' => $invoice_id, 'subscription_product_id' => $subscription_id, 'origin_user_id' => $origin_user_id, 'source_sales_manager_user_id' => $source_sales_manager_user_id ] );
		do_action( 'sn_project_membership_activated', $membership_id, $invoice_id, $invoice );
	}

	private function invoice_is_verified( object $invoice ): bool {
		$statuses = array_filter( array_map( 'sanitize_key', [
			(string) ( $invoice->status ?? '' ), (string) ( $invoice->invoice_status ?? '' ),
			(string) ( $invoice->payment_status ?? '' ), (string) ( $invoice->payment_workflow_status ?? '' ),
		] ) );
		return (bool) array_intersect( $statuses, [ 'paid', 'approved', 'completed' ] );
	}

	private function invoice_total( object $invoice ): float {
		foreach ( [ 'final_total', 'payment_total_amount', 'product_price', 'original_total' ] as $field ) {
			if ( isset( $invoice->{$field} ) && (float) $invoice->{$field} > 0 ) { return (float) $invoice->{$field}; }
		}
		return 0.0;
	}

	private function restore_reversed_invoice_links( int $invoice_id, object $invoice ): void {
		if ( ! $this->invoice_is_verified( $invoice ) ) { return; }
		global $wpdb; $t = $this->tables();
		$rows = $wpdb->get_results( $wpdb->prepare(
			"SELECT l.*,mi.membership_id,m.origin_user_id FROM {$t['invoice_links']} l INNER JOIN {$t['items']} mi ON mi.id=l.membership_item_id INNER JOIN {$t['memberships']} m ON m.id=mi.membership_id WHERE l.invoice_id=%d AND l.status='reversed' ORDER BY l.id",
			$invoice_id
		) ) ?: [];
		foreach ( $rows as $row ) {
			$now = current_time( 'mysql' );
			$this->credit_biavin_commission( (int) $row->membership_item_id, (int) $row->origin_user_id, $invoice_id, (int) $row->content_product_id, (float) $row->line_total );
			$commission_status = $wpdb->get_var( $wpdb->prepare( "SELECT status FROM {$t['commissions']} WHERE membership_item_id=%d AND invoice_id=%d AND recipient_user_id=%d LIMIT 1", (int) $row->membership_item_id, $invoice_id, (int) $row->origin_user_id ) );
			if ( null !== $commission_status && (string) $commission_status !== 'credited' ) { continue; }
			$updated = $wpdb->update( $t['invoice_links'], [ 'status' => 'verified', 'verified_at' => $now, 'updated_at' => $now ], [ 'id' => (int) $row->id, 'status' => 'reversed' ] );
			if ( false === $updated || 0 === $updated ) { continue; }
			$this->event( (int) $row->membership_id, (int) $row->membership_item_id, 'verified_invoice_reactivated', [ 'invoice_id' => $invoice_id, 'line_total' => (float) $row->line_total ] );
		}
	}

	private function wallet_id( int $user_id ): int {
		global $wpdb; $table = $wpdb->prefix . 'sn_wallets';
		if ( ! $this->table_exists( $table ) || ! $this->table_exists( $wpdb->prefix . 'sn_wallet_transactions' ) ) { $this->ensure_wallet_tables(); }
		$id = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$table} WHERE user_id=%d AND wallet_type='biavin' LIMIT 1", $user_id ) );
		if ( $id ) { return $id; }
		$wpdb->query( $wpdb->prepare(
			"INSERT IGNORE INTO {$table} (user_id,wallet_type,balance,total_credit,total_debit) VALUES (%d,'biavin',0,0,0)",
			$user_id
		) );
		return (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$table} WHERE user_id=%d AND wallet_type='biavin' LIMIT 1", $user_id ) );
	}

	private function post_wallet_transaction( int $user_id, int $invoice_id, float $amount, string $direction, string $type, string $description, array $meta ): int {
		if ( $user_id < 1 || $amount <= 0 ) { return 0; }
		global $wpdb; $wallet_id = $this->wallet_id( $user_id );
		if ( $wallet_id < 1 ) { return 0; }
		$ok = $wpdb->insert( $wpdb->prefix . 'sn_wallet_transactions', [
			'wallet_id' => $wallet_id, 'user_id' => $user_id, 'wallet_type' => 'biavin',
			'invoice_id' => $invoice_id ?: null, 'amount' => $amount,
			'direction' => $direction === 'debit' ? 'debit' : 'credit', 'type' => sanitize_key( $type ),
			'status' => 'approved', 'description' => sanitize_text_field( $description ),
			'meta' => wp_json_encode( $meta, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ),
			'created_by' => get_current_user_id() ?: null, 'created_at' => current_time( 'mysql' ),
		] );
		if ( ! $ok ) { return 0; }
		$transaction_id = (int) $wpdb->insert_id;
		$sign = $direction === 'debit' ? -1 : 1;
		$updated = $wpdb->query( $wpdb->prepare(
			"UPDATE {$wpdb->prefix}sn_wallets SET balance=balance+%f,total_credit=total_credit+%f,total_debit=total_debit+%f WHERE id=%d",
			$sign * $amount, $direction === 'debit' ? 0 : $amount, $direction === 'debit' ? $amount : 0, $wallet_id
		) );
		return false === $updated ? 0 : $transaction_id;
	}

	private function credit_biavin_commission( int $item_id, int $recipient_id, int $invoice_id, int $content_id, float $base ): void {
		if ( class_exists( 'SN_Purpose_Commission' ) && SN_Purpose_Commission::is_matrix_engine_enabled() ) { return; }
		if ( $base <= 0 || $recipient_id < 1 ) { return; }
		global $wpdb; $t = $this->tables();
		if ( ! $this->table_exists( $wpdb->prefix . 'sn_wallets' ) || ! $this->table_exists( $wpdb->prefix . 'sn_wallet_transactions' ) ) { $this->ensure_wallet_tables(); }
		$wpdb->query( 'START TRANSACTION' );
		$commission = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['commissions']} WHERE membership_item_id=%d AND invoice_id=%d AND recipient_user_id=%d LIMIT 1 FOR UPDATE", $item_id, $invoice_id, $recipient_id ) );
		if ( ! $commission ) {
			$rule = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['rules']} WHERE content_product_id=%d AND is_active=1 LIMIT 1", $content_id ) );
			if ( ! $rule || (float) $rule->commission_value <= 0 ) { $wpdb->query( 'ROLLBACK' ); return; }
			$amount = (string) $rule->commission_type === 'fixed' ? (float) $rule->commission_value : round( $base * (float) $rule->commission_value / 100, 2 );
			if ( $amount <= 0 ) { $wpdb->query( 'ROLLBACK' ); return; }
			$inserted = $wpdb->insert( $t['commissions'], [
				'membership_item_id' => $item_id, 'invoice_id' => $invoice_id, 'recipient_user_id' => $recipient_id,
				'base_amount' => $base, 'commission_amount' => $amount, 'status' => 'pending',
				'created_at' => current_time( 'mysql' ), 'updated_at' => current_time( 'mysql' ),
			] );
			if ( ! $inserted ) { $wpdb->query( 'ROLLBACK' ); return; }
			$commission = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['commissions']} WHERE id=%d FOR UPDATE", (int) $wpdb->insert_id ) );
		}
		if ( ! $commission || ! in_array( (string) $commission->status, [ 'pending', 'reversed' ], true ) ) { $wpdb->query( 'COMMIT' ); return; }
		$is_recredit = (string) $commission->status === 'reversed';
		$tx_id = $this->post_wallet_transaction(
			$recipient_id,
			$invoice_id,
			(float) $commission->commission_amount,
			'credit',
			$is_recredit ? 'biavin_commission_recredit' : 'biavin_commission',
			( $is_recredit ? 'بازگردانی ' : '' ) . 'پورسانت بیاوین فاکتور #' . $invoice_id,
			[ 'project_commission_id' => (int) $commission->id, 'membership_item_id' => $item_id, 'content_product_id' => $content_id ]
		);
		if ( ! $tx_id ) { $wpdb->query( 'ROLLBACK' ); return; }
		$commission_update = [ 'status' => 'credited', 'wallet_transaction_id' => $tx_id, 'updated_at' => current_time( 'mysql' ) ];
		if ( $is_recredit ) { $commission_update['reversal_transaction_id'] = null; }
		$updated = $wpdb->update( $t['commissions'], $commission_update, [ 'id' => (int) $commission->id, 'status' => (string) $commission->status ] );
		if ( false === $updated || 0 === $updated ) { $wpdb->query( 'ROLLBACK' ); return; }
		$wpdb->query( 'COMMIT' );
	}

	/** Idempotently post the configured Biawin card commission for an Operations upgrade. */
	public function credit_operations_upgrade_commission( int $item_id, int $invoice_id, float $base ): void {
		if ( $item_id < 1 || $invoice_id < 1 || $base <= 0 ) { return; }
		global $wpdb; $t = $this->tables();
		$row = $wpdb->get_row( $wpdb->prepare(
			"SELECT mi.content_product_id,m.origin_user_id FROM {$t['items']} mi INNER JOIN {$t['memberships']} m ON m.id=mi.membership_id WHERE mi.id=%d LIMIT 1",
			$item_id
		) );
		if ( ! $row ) { return; }
		$this->credit_biavin_commission( $item_id, (int) $row->origin_user_id, $invoice_id, (int) $row->content_product_id, $base );
	}

	public function on_invoice_reversed( int $invoice_id, $invoice = null, string $reason = '' ): void {
		global $wpdb; $t = $this->tables();
		$fresh = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$wpdb->prefix}sn_invoices WHERE id=%d LIMIT 1", $invoice_id ) );
		$status = sanitize_key( (string) ( $fresh->status ?? '' ) );
		$paid_total = (float) ( $fresh->paid_total_amount ?? 0 );
		$should_suspend = $paid_total <= 0.5 || in_array( $status, [ 'rejected', 'cancelled', 'canceled', 'void', 'refunded' ], true );
		if ( $should_suspend && $this->table_exists( $t['memberships'] ) ) {
			$membership_ids = array_map( 'intval', (array) $wpdb->get_col( $wpdb->prepare( "SELECT id FROM {$t['memberships']} WHERE first_payment_invoice_id=%d AND status='active'", $invoice_id ) ) );
			if ( $membership_ids ) {
				$wpdb->query( $wpdb->prepare( "UPDATE {$t['memberships']} SET status='suspended',updated_at=%s WHERE first_payment_invoice_id=%d AND status='active'", current_time( 'mysql' ), $invoice_id ) );
				foreach ( $membership_ids as $membership_id ) { $this->event( $membership_id, 0, 'membership_suspended_finance', [ 'invoice_id' => $invoice_id, 'reason' => $reason ] ); }
			}
		}
		if ( $this->table_exists( $t['commissions'] ) ) {
			if ( ! $this->table_exists( $wpdb->prefix . 'sn_wallets' ) || ! $this->table_exists( $wpdb->prefix . 'sn_wallet_transactions' ) ) { $this->ensure_wallet_tables(); }
			$row_ids = array_map( 'intval', (array) $wpdb->get_col( $wpdb->prepare( "SELECT id FROM {$t['commissions']} WHERE invoice_id=%d AND status='credited' AND reversal_transaction_id IS NULL", $invoice_id ) ) );
			foreach ( $row_ids as $row_id ) {
				$wpdb->query( 'START TRANSACTION' );
				$row = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['commissions']} WHERE id=%d AND status='credited' AND reversal_transaction_id IS NULL LIMIT 1 FOR UPDATE", $row_id ) );
				if ( ! $row ) { $wpdb->query( 'COMMIT' ); continue; }
				$tx_id = $this->post_wallet_transaction( (int) $row->recipient_user_id, $invoice_id, (float) $row->commission_amount, 'debit', 'biavin_commission_reversal', 'برگشت پورسانت بیاوین: ' . sanitize_text_field( $reason ), [ 'project_commission_id' => (int) $row->id, 'original_transaction_id' => (int) $row->wallet_transaction_id ] );
				if ( ! $tx_id ) { $wpdb->query( 'ROLLBACK' ); continue; }
				$updated = $wpdb->update( $t['commissions'], [ 'status' => 'reversed', 'reversal_transaction_id' => $tx_id, 'updated_at' => current_time( 'mysql' ) ], [ 'id' => (int) $row->id, 'status' => 'credited' ] );
				if ( false === $updated || 0 === $updated ) { $wpdb->query( 'ROLLBACK' ); continue; }
				$wpdb->query( 'COMMIT' );
			}
		}
		if ( $this->table_exists( $t['invoice_links'] ) ) {
			$wpdb->update( $t['invoice_links'], [ 'status' => 'reversed', 'updated_at' => current_time( 'mysql' ) ], [ 'invoice_id' => $invoice_id ] );
		}
	}

	private function project_users( string $position ): array {
		global $wpdb;
		$position = sanitize_key( $position );
		$role = $position === 'project_manager' ? 'sn_project_manager' : 'sn_project_expert';
		$users = [];
		foreach ( get_users( [ 'role' => $role, 'number' => 2000, 'fields' => 'all' ] ) as $user ) {
			if ( $user instanceof WP_User ) { $users[ (int) $user->ID ] = $user; }
		}
		$profiles = $wpdb->prefix . 'sn_hr_profiles'; $positions = $wpdb->prefix . 'sn_hr_positions';
		if ( $this->table_exists( $profiles ) && $this->table_exists( $positions ) ) {
			$ids = $wpdb->get_col( $wpdb->prepare( "SELECT hp.user_id FROM {$profiles} hp INNER JOIN {$positions} pos ON pos.id=hp.position_id WHERE pos.slug=%s AND hp.is_active=1", $position ) );
			foreach ( (array) $ids as $id ) {
				$user = get_user_by( 'id', (int) $id );
				if ( $user instanceof WP_User ) { $users[ (int) $user->ID ] = $user; }
			}
		}
		uasort( $users, static fn( $a, $b ) => strcasecmp( (string) $a->display_name, (string) $b->display_name ) );
		return array_values( $users );
	}

	private function sales_manager_users(): array {
		global $wpdb; $users = [];
		foreach ( get_users( [ 'role__in' => [ 'sn_sales_manager', 'sas_sales_manager' ], 'number' => 2000, 'fields' => 'all' ] ) as $user ) {
			if ( $user instanceof WP_User ) { $users[ (int) $user->ID ] = $user; }
		}
		$profiles = $wpdb->prefix . 'sn_hr_profiles'; $positions = $wpdb->prefix . 'sn_hr_positions';
		if ( $this->table_exists( $profiles ) && $this->table_exists( $positions ) ) {
			$ids = $wpdb->get_col( "SELECT hp.user_id FROM {$profiles} hp INNER JOIN {$positions} pos ON pos.id=hp.position_id WHERE pos.slug='sales_manager' AND hp.is_active=1" );
			foreach ( (array) $ids as $id ) {
				$user = get_user_by( 'id', (int) $id );
				if ( $user instanceof WP_User ) { $users[ (int) $user->ID ] = $user; }
			}
		}
		uasort( $users, static fn( $a, $b ) => strcasecmp( (string) $a->display_name, (string) $b->display_name ) );
		return array_values( $users );
	}

	private function operations_sales_manager_users(): array {
		global $wpdb; $users = [];
		foreach ( get_users( [ 'role' => 'sn_operations_sales_manager', 'number' => 2000, 'fields' => 'all' ] ) as $user ) {
			if ( $user instanceof WP_User ) { $users[ (int) $user->ID ] = $user; }
		}
		$profiles = $wpdb->prefix . 'sn_hr_profiles'; $positions = $wpdb->prefix . 'sn_hr_positions';
		if ( $this->table_exists( $profiles ) && $this->table_exists( $positions ) ) {
			$ids = $wpdb->get_col( "SELECT hp.user_id FROM {$profiles} hp INNER JOIN {$positions} pos ON pos.id=hp.position_id WHERE pos.slug='operations_sales_manager' AND hp.is_active=1" );
			foreach ( (array) $ids as $id ) {
				$user = get_user_by( 'id', (int) $id );
				if ( $user instanceof WP_User ) { $users[ (int) $user->ID ] = $user; }
			}
		}
		uasort( $users, static fn( $a, $b ) => strcasecmp( (string) $a->display_name, (string) $b->display_name ) );
		return array_values( $users );
	}

	/** Active Operations supervisors annotated with their current direct manager. */
	private function operations_sales_supervisor_users(): array {
		global $wpdb; $users = [];
		foreach ( get_users( [ 'role' => 'sn_operations_sales_supervisor', 'number' => 2000, 'fields' => 'all' ] ) as $user ) {
			if ( $user instanceof WP_User ) { $users[ (int) $user->ID ] = $user; }
		}
		$profiles = $wpdb->prefix . 'sn_hr_profiles'; $positions = $wpdb->prefix . 'sn_hr_positions'; $assignments = $wpdb->prefix . 'sn_hr_assignments';
		if ( $this->table_exists( $profiles ) && $this->table_exists( $positions ) ) {
			$ids = $wpdb->get_col( "SELECT hp.user_id FROM {$profiles} hp INNER JOIN {$positions} pos ON pos.id=hp.position_id WHERE pos.slug='operations_sales_supervisor' AND hp.is_active=1" );
			foreach ( (array) $ids as $id ) {
				$user = get_user_by( 'id', (int) $id );
				if ( $user instanceof WP_User ) { $users[ (int) $user->ID ] = $user; }
			}
		}
		$parent_map = [];
		if ( $users && $this->table_exists( $assignments ) && $this->table_exists( $profiles ) ) {
			$user_ids = implode( ',', array_map( 'intval', array_keys( $users ) ) );
			foreach ( $wpdb->get_results( "SELECT child.user_id child_user_id,parent.user_id parent_user_id FROM {$assignments} a INNER JOIN {$profiles} child ON child.id=a.child_profile_id INNER JOIN {$profiles} parent ON parent.id=a.parent_profile_id WHERE child.user_id IN ({$user_ids}) AND a.is_current=1 AND (a.relationship_type='reports_to' OR a.relationship_type='' OR a.relationship_type IS NULL) ORDER BY a.id DESC", ARRAY_A ) ?: [] as $row ) {
				$child_id = (int) $row['child_user_id'];
				if ( ! isset( $parent_map[ $child_id ] ) ) { $parent_map[ $child_id ] = (int) $row['parent_user_id']; }
			}
		}
		foreach ( $users as $user_id => $user ) { $user->sn_operations_manager_user_id = (int) ( $parent_map[ $user_id ] ?? 0 ); }
		uasort( $users, static fn( $a, $b ) => strcasecmp( (string) $a->display_name, (string) $b->display_name ) );
		return array_values( $users );
	}

	private function settings_maps(): array {
		global $wpdb; $t = $this->tables();
		$subscriptions = [];
		if ( $this->table_exists( $t['subscriptions'] ) ) {
			$rows = $wpdb->get_results( "SELECT s.subscription_product_id,c.content_product_id,c.sort_order FROM {$t['subscriptions']} s LEFT JOIN {$t['contents']} c ON c.subscription_product_id=s.subscription_product_id AND c.is_active=1 WHERE s.is_active=1 ORDER BY s.subscription_product_id,c.sort_order,c.id", ARRAY_A ) ?: [];
			foreach ( $rows as $row ) {
				$sid = (int) $row['subscription_product_id'];
				if ( ! isset( $subscriptions[ $sid ] ) ) { $subscriptions[ $sid ] = []; }
				if ( ! empty( $row['content_product_id'] ) ) { $subscriptions[ $sid ][] = (int) $row['content_product_id']; }
			}
		}
		$managers = [];
		if ( $this->table_exists( $t['managers'] ) ) {
			foreach ( $wpdb->get_results( "SELECT content_product_id,manager_user_id FROM {$t['managers']}", ARRAY_A ) ?: [] as $row ) { $managers[ (int) $row['content_product_id'] ] = (int) $row['manager_user_id']; }
		}
		$manager_routes = [];
		if ( $this->table_exists( $t['manager_routes'] ) ) {
			foreach ( $wpdb->get_results( "SELECT content_product_id,sales_manager_user_id,manager_user_id FROM {$t['manager_routes']} ORDER BY content_product_id,sales_manager_user_id,id", ARRAY_A ) ?: [] as $row ) {
				$content_id = (int) $row['content_product_id'];
				if ( ! isset( $manager_routes[ $content_id ] ) ) { $manager_routes[ $content_id ] = []; }
				$manager_routes[ $content_id ][] = [
					'sales_manager_user_id' => (int) $row['sales_manager_user_id'],
					'manager_user_id' => (int) $row['manager_user_id'],
				];
			}
		}
		$operations_managers = [];
		if ( $this->table_exists( $t['operations_managers'] ) ) {
			foreach ( $wpdb->get_results( "SELECT content_product_id,operations_manager_user_id FROM {$t['operations_managers']}", ARRAY_A ) ?: [] as $row ) {
				$operations_managers[ (int) $row['content_product_id'] ] = (int) $row['operations_manager_user_id'];
			}
		}
		$operations_manager_routes = [];
		if ( $this->table_exists( $t['operations_manager_routes'] ) ) {
			foreach ( $wpdb->get_results( "SELECT content_product_id,sales_manager_user_id,operations_manager_user_id FROM {$t['operations_manager_routes']} ORDER BY content_product_id,sales_manager_user_id,id", ARRAY_A ) ?: [] as $row ) {
				$content_id = (int) $row['content_product_id'];
				if ( ! isset( $operations_manager_routes[ $content_id ] ) ) { $operations_manager_routes[ $content_id ] = []; }
				$operations_manager_routes[ $content_id ][] = [
					'sales_manager_user_id' => (int) $row['sales_manager_user_id'],
					'operations_manager_user_id' => (int) $row['operations_manager_user_id'],
				];
			}
		}
		$operations_supervisor_routes = [];
		if ( $this->table_exists( $t['operations_supervisor_routes'] ) ) {
			foreach ( $wpdb->get_results( "SELECT content_product_id,sales_manager_user_id,operations_manager_user_id,operations_supervisor_user_id FROM {$t['operations_supervisor_routes']} ORDER BY content_product_id,sales_manager_user_id,id", ARRAY_A ) ?: [] as $row ) {
				$content_id = (int) $row['content_product_id'];
				if ( ! isset( $operations_supervisor_routes[ $content_id ] ) ) { $operations_supervisor_routes[ $content_id ] = []; }
				$operations_supervisor_routes[ $content_id ][] = [
					'sales_manager_user_id' => (int) $row['sales_manager_user_id'],
					'operations_manager_user_id' => (int) $row['operations_manager_user_id'],
					'operations_supervisor_user_id' => (int) $row['operations_supervisor_user_id'],
				];
			}
		}
		$rules = [];
		if ( $this->table_exists( $t['rules'] ) ) {
			foreach ( $wpdb->get_results( "SELECT * FROM {$t['rules']}", ARRAY_A ) ?: [] as $row ) { $rules[ (int) $row['content_product_id'] ] = $row; }
		}
		$api_configs = [];
		if ( $this->table_exists( $t['api_configs'] ) ) {
			foreach ( $wpdb->get_results( "SELECT * FROM {$t['api_configs']}", ARRAY_A ) ?: [] as $row ) { $api_configs[ (int) $row['content_product_id'] ] = $row; }
		}
		return [
			'subscriptions' => $subscriptions,
			'managers' => $managers,
			'manager_routes' => $manager_routes,
			'operations_managers' => $operations_managers,
			'operations_manager_routes' => $operations_manager_routes,
			'operations_supervisor_routes' => $operations_supervisor_routes,
			'rules' => $rules,
			'api_configs' => $api_configs,
		];
	}

	public function save_settings_from_request( array $request ) {
		if ( isset( $request['sn_operations_customer_inactivity_days'] ) ) {
			$days = min( 365, max( 0, absint( $request['sn_operations_customer_inactivity_days'] ) ) );
			update_option( 'sn_operations_customer_inactivity_days', $days, false );
			if ( 0 === $days && class_exists( 'SN_Operations_Flow' ) ) { SN_Operations_Flow::instance()->route_inactive_cards( 500 ); }
		}
		if ( empty( $request['sn_projects_settings_present'] ) || ! isset( $request['sn_projects'] ) || ! is_array( $request['sn_projects'] ) ) { return true; }
		$raw = isset( $request['sn_projects'] ) && is_array( $request['sn_projects'] ) ? wp_unslash( $request['sn_projects'] ) : [];
		$route_controls_present = ! empty( $request['sn_project_manager_routes_present'] );
		$operations_route_controls_present = ! empty( $request['sn_operations_manager_routes_present'] );
		$operations_supervisor_route_controls_present = ! empty( $request['sn_operations_supervisor_routes_present'] );
		$compact_rows = isset( $raw['subscription_rows'] ) && is_array( $raw['subscription_rows'] ) ? $raw['subscription_rows'] : null;
		$subscription_rows = is_array( $compact_rows ) ? $compact_rows : ( isset( $raw['subscriptions'] ) && is_array( $raw['subscriptions'] ) ? $raw['subscriptions'] : [] );
		$selected = []; $content_set = [];
		foreach ( $subscription_rows as $subscription_key => $row ) {
			$row = is_array( $row ) ? $row : [];
			$subscription_id = is_array( $compact_rows ) ? absint( $row['product_id'] ?? 0 ) : absint( $subscription_key );
			if ( $subscription_id < 1 || empty( $row['active'] ) ) { continue; }
			if ( ! $this->product_exists( $subscription_id ) ) { return new WP_Error( 'sn_project_subscription_missing', 'یکی از محصولات اشتراک معتبر، منتشرشده یا «فعال در شبکه فروش» نیست.' ); }
			if ( isset( $selected[ $subscription_id ] ) ) { return new WP_Error( 'sn_project_subscription_duplicate', 'یک محصول اشتراک بیش از یک بار در جدول پروژه‌ها انتخاب شده است.' ); }
			$contents = array_values( array_unique( array_filter( array_map( 'absint', (array) ( $row['contents'] ?? [] ) ) ) ) );
			$contents = array_values( array_diff( $contents, [ $subscription_id ] ) );
			if ( ! $contents ) { return new WP_Error( 'sn_project_contents_empty', 'برای اشتراک «' . get_the_title( $subscription_id ) . '» حداقل یک محصول محتوا انتخاب کنید.' ); }
			foreach ( $contents as $content_id ) {
				if ( ! $this->product_exists( $content_id ) ) { return new WP_Error( 'sn_project_content_missing', 'یکی از محصولات محتوای انتخاب‌شده معتبر، منتشرشده یا «فعال در شبکه فروش» نیست.' ); }
				$content_set[ $content_id ] = true;
			}
			$selected[ $subscription_id ] = $contents;
		}
		foreach ( array_keys( $selected ) as $subscription_id ) {
			if ( isset( $content_set[ $subscription_id ] ) ) { return new WP_Error( 'sn_project_product_overlap', 'یک محصول نمی‌تواند هم‌زمان اشتراک و محتوای اشتراک باشد.' ); }
		}

		$manager_raw = isset( $raw['managers'] ) && is_array( $raw['managers'] ) ? $raw['managers'] : [];
		$route_raw = isset( $raw['manager_routes'] ) && is_array( $raw['manager_routes'] ) ? $raw['manager_routes'] : [];
		$operations_manager_raw = isset( $raw['operations_managers'] ) && is_array( $raw['operations_managers'] ) ? $raw['operations_managers'] : [];
		$operations_route_raw = isset( $raw['operations_manager_routes'] ) && is_array( $raw['operations_manager_routes'] ) ? $raw['operations_manager_routes'] : [];
		$operations_supervisor_route_raw = isset( $raw['operations_supervisor_routes'] ) && is_array( $raw['operations_supervisor_routes'] ) ? $raw['operations_supervisor_routes'] : [];
		$rule_raw = isset( $raw['rules'] ) && is_array( $raw['rules'] ) ? $raw['rules'] : [];
		$api_raw = isset( $raw['apis'] ) && is_array( $raw['apis'] ) ? $raw['apis'] : [];
		$existing_maps = $this->settings_maps();
		$existing_api = $existing_maps['api_configs'];
		$valid_manager_ids = array_flip( array_map( static fn( $u ) => (int) $u->ID, $this->project_users( 'project_manager' ) ) );
		$valid_sales_manager_ids = array_flip( array_map( static fn( $u ) => (int) $u->ID, $this->sales_manager_users() ) );
		$valid_operations_manager_ids = array_flip( array_map( static fn( $u ) => (int) $u->ID, $this->operations_sales_manager_users() ) );
		$operations_supervisors = $this->operations_sales_supervisor_users();
		$valid_operations_supervisors = [];
		foreach ( $operations_supervisors as $supervisor ) { $valid_operations_supervisors[ (int) $supervisor->ID ] = (int) ( $supervisor->sn_operations_manager_user_id ?? 0 ); }
		$managers = []; $manager_routes = []; $operations_managers = []; $operations_manager_routes = []; $operations_supervisor_routes = []; $rules = []; $apis = [];
		foreach ( array_keys( $content_set ) as $content_id ) {
			if ( $route_controls_present ) {
				$manager_id = absint( $manager_raw[ $content_id ] ?? 0 );
				if ( $manager_id < 1 ) { return new WP_Error( 'sn_project_manager_required', 'برای محتوای «' . get_the_title( $content_id ) . '» مدیر پروژه را انتخاب کنید.' ); }
				if ( ! isset( $valid_manager_ids[ $manager_id ] ) ) { return new WP_Error( 'sn_project_manager_scope', 'مدیر انتخاب‌شده برای «' . get_the_title( $content_id ) . '» نقش مدیر پروژه فعال ندارد.' ); }
				$managers[ $content_id ] = $manager_id;
				$seen_sales_managers = [];
				foreach ( (array) ( $route_raw[ $content_id ] ?? [] ) as $route ) {
					$route = is_array( $route ) ? $route : [];
					$sales_manager_id = absint( $route['sales_manager_user_id'] ?? 0 );
					$route_manager_id = absint( $route['manager_user_id'] ?? 0 );
					if ( $sales_manager_id < 1 && $route_manager_id < 1 ) { continue; }
					if ( $sales_manager_id < 1 || $route_manager_id < 1 ) { return new WP_Error( 'sn_project_manager_route_incomplete', 'در مسیر اختصاصی «' . get_the_title( $content_id ) . '» هم مدیر فروش مبدأ و هم مدیر پروژه مقصد را انتخاب کنید.' ); }
					if ( ! isset( $valid_sales_manager_ids[ $sales_manager_id ] ) ) { return new WP_Error( 'sn_project_sales_manager_scope', 'مدیر فروش انتخاب‌شده برای «' . get_the_title( $content_id ) . '» فعال یا معتبر نیست.' ); }
					if ( ! isset( $valid_manager_ids[ $route_manager_id ] ) ) { return new WP_Error( 'sn_project_route_manager_scope', 'مدیر پروژه مقصد در یکی از مسیرهای «' . get_the_title( $content_id ) . '» فعال یا معتبر نیست.' ); }
					if ( isset( $seen_sales_managers[ $sales_manager_id ] ) ) { return new WP_Error( 'sn_project_manager_route_duplicate', 'برای یک مدیر فروش در محتوای «' . get_the_title( $content_id ) . '» بیش از یک مقصد ثبت شده است.' ); }
					$seen_sales_managers[ $sales_manager_id ] = true;
					$manager_routes[ $content_id ][] = [ 'sales_manager_user_id' => $sales_manager_id, 'manager_user_id' => $route_manager_id ];
				}
			}
			if ( $operations_route_controls_present ) {
				$operations_manager_id = absint( $operations_manager_raw[ $content_id ] ?? 0 );
				if ( $operations_manager_id < 1 ) {
					return new WP_Error( 'sn_operations_manager_required', 'برای کارت «' . get_the_title( $content_id ) . '» مدیر فروش عملیات پیش‌فرض را انتخاب کنید.' );
				}
				if ( ! isset( $valid_operations_manager_ids[ $operations_manager_id ] ) ) {
					return new WP_Error( 'sn_operations_manager_scope', 'مدیر فروش عملیات پیش‌فرض «' . get_the_title( $content_id ) . '» فعال یا معتبر نیست.' );
				}
				$operations_managers[ $content_id ] = $operations_manager_id;
				$seen_operations_sources = [];
				foreach ( (array) ( $operations_route_raw[ $content_id ] ?? [] ) as $route ) {
					$route = is_array( $route ) ? $route : [];
					$sales_manager_id = absint( $route['sales_manager_user_id'] ?? 0 );
					$route_manager_id = absint( $route['operations_manager_user_id'] ?? 0 );
					if ( $sales_manager_id < 1 && $route_manager_id < 1 ) { continue; }
					if ( $sales_manager_id < 1 || $route_manager_id < 1 ) {
						return new WP_Error( 'sn_operations_manager_route_incomplete', 'در مسیر عملیات کارت «' . get_the_title( $content_id ) . '» مدیر فروش مبدأ و مدیر فروش عملیات مقصد را کامل انتخاب کنید.' );
					}
					if ( ! isset( $valid_sales_manager_ids[ $sales_manager_id ] ) ) {
						return new WP_Error( 'sn_operations_source_manager_scope', 'مدیر فروش مبدأ انتخاب‌شده برای کارت «' . get_the_title( $content_id ) . '» فعال یا معتبر نیست.' );
					}
					if ( ! isset( $valid_operations_manager_ids[ $route_manager_id ] ) ) {
						return new WP_Error( 'sn_operations_route_manager_scope', 'مدیر فروش عملیات مقصد در یکی از مسیرهای کارت «' . get_the_title( $content_id ) . '» فعال یا معتبر نیست.' );
					}
					if ( isset( $seen_operations_sources[ $sales_manager_id ] ) ) {
						return new WP_Error( 'sn_operations_manager_route_duplicate', 'برای یک مدیر فروش مبدأ در کارت «' . get_the_title( $content_id ) . '» بیش از یک مدیر عملیات ثبت شده است.' );
					}
					$seen_operations_sources[ $sales_manager_id ] = true;
					$operations_manager_routes[ $content_id ][] = [
						'sales_manager_user_id' => $sales_manager_id,
						'operations_manager_user_id' => $route_manager_id,
					];
				}
			}
			if ( $operations_supervisor_route_controls_present ) {
				$operations_manager_id = absint( $operations_manager_raw[ $content_id ] ?? 0 );
				if ( $operations_manager_id < 1 || ! isset( $valid_operations_manager_ids[ $operations_manager_id ] ) ) {
					return new WP_Error( 'sn_operations_supervisor_manager_required', 'ابتدا مدیر فروش عملیات معتبر کارت «' . get_the_title( $content_id ) . '» را انتخاب کنید.' );
				}
				$operations_managers[ $content_id ] = $operations_manager_id;
				$seen_operations_sources = [];
				foreach ( (array) ( $operations_supervisor_route_raw[ $content_id ] ?? [] ) as $route ) {
					$route = is_array( $route ) ? $route : [];
					$sales_manager_id = absint( $route['sales_manager_user_id'] ?? 0 );
					$supervisor_id = absint( $route['operations_supervisor_user_id'] ?? 0 );
					if ( $sales_manager_id < 1 && $supervisor_id < 1 ) { continue; }
					if ( $sales_manager_id < 1 || $supervisor_id < 1 ) {
						return new WP_Error( 'sn_operations_supervisor_route_incomplete', 'در مسیر کارت «' . get_the_title( $content_id ) . '» مدیر فروش مبدأ و سرپرست عملیات مقصد را کامل انتخاب کنید.' );
					}
					if ( ! isset( $valid_sales_manager_ids[ $sales_manager_id ] ) ) {
						return new WP_Error( 'sn_operations_supervisor_source_scope', 'مدیر فروش مبدأ مسیر کارت «' . get_the_title( $content_id ) . '» معتبر نیست.' );
					}
					if ( ! isset( $valid_operations_supervisors[ $supervisor_id ] ) || (int) $valid_operations_supervisors[ $supervisor_id ] !== $operations_manager_id ) {
						return new WP_Error( 'sn_operations_supervisor_parent_scope', 'سرپرست عملیات انتخاب‌شده برای کارت «' . get_the_title( $content_id ) . '» زیرمجموعه مستقیم مدیر عملیات همین کارت نیست.' );
					}
					if ( isset( $seen_operations_sources[ $sales_manager_id ] ) ) {
						return new WP_Error( 'sn_operations_supervisor_route_duplicate', 'برای یک مدیر فروش مبدأ در کارت «' . get_the_title( $content_id ) . '» بیش از یک سرپرست عملیات ثبت شده است.' );
					}
					$seen_operations_sources[ $sales_manager_id ] = true;
					$operations_supervisor_routes[ $content_id ][] = [
						'sales_manager_user_id' => $sales_manager_id,
						'operations_manager_user_id' => $operations_manager_id,
						'operations_supervisor_user_id' => $supervisor_id,
					];
				}
			}
			$rule = isset( $rule_raw[ $content_id ] ) && is_array( $rule_raw[ $content_id ] ) ? $rule_raw[ $content_id ] : [];
			$type = sanitize_key( (string) ( $rule['type'] ?? 'percent' ) );
			if ( ! in_array( $type, [ 'percent', 'fixed' ], true ) ) { $type = 'percent'; }
			$value_raw = class_exists( 'SN_Helpers' ) ? SN_Helpers::to_english_nums( (string) ( $rule['value'] ?? 0 ) ) : (string) ( $rule['value'] ?? 0 );
			$value = max( 0, (float) str_replace( [ ',', '٬', '،', ' ' ], '', $value_raw ) );
			if ( $type === 'percent' ) { $value = min( 100, $value ); }
			$rules[ $content_id ] = [ 'type' => $type, 'value' => $value, 'active' => ! empty( $rule['active'] ) && $value > 0 ? 1 : 0 ];
			$api = isset( $api_raw[ $content_id ] ) && is_array( $api_raw[ $content_id ] ) ? $api_raw[ $content_id ] : [];
			$old_api = (array) ( $existing_api[ $content_id ] ?? [] );
			$endpoint = esc_url_raw( trim( (string) ( $api['endpoint_url'] ?? '' ) ) );
			$project_code = preg_replace( '/[^A-Za-z0-9_.:-]/', '', (string) ( $api['project_code'] ?? '' ) );
			$project_code = substr( (string) $project_code, 0, 100 );
			$api_key = trim( (string) ( $api['api_key'] ?? '' ) );
			if ( $api_key === '' ) { $api_key = (string) ( $old_api['api_key'] ?? '' ); }
			$apis[ $content_id ] = [ 'endpoint_url' => $endpoint, 'api_key' => $api_key, 'project_code' => $project_code, 'is_active' => ! empty( $api['is_active'] ) && $endpoint !== '' && $api_key !== '' && $project_code !== '' ? 1 : 0 ];
		}

		global $wpdb; $t = $this->tables(); $now = current_time( 'mysql' ); $actor = get_current_user_id();
		if ( $route_controls_present && ! $this->table_exists( $t['manager_routes'] ) ) { return new WP_Error( 'sn_project_manager_route_table', 'جدول مسیریابی مدیران فروش آماده نیست؛ یک‌بار صفحه را تازه‌سازی و دوباره ذخیره کنید.' ); }
		if ( $operations_route_controls_present && ( ! $this->table_exists( $t['operations_managers'] ) || ! $this->table_exists( $t['operations_manager_routes'] ) ) ) {
			return new WP_Error( 'sn_operations_manager_route_table', 'جدول مسیریابی مدیران فروش عملیات آماده نیست؛ یک‌بار صفحه را تازه‌سازی و دوباره ذخیره کنید.' );
		}
		if ( $operations_supervisor_route_controls_present && ! $this->table_exists( $t['operations_supervisor_routes'] ) ) {
			return new WP_Error( 'sn_operations_supervisor_route_table', 'جدول مسیریابی سرپرستان فروش عملیات آماده نیست؛ یک‌بار صفحه را تازه‌سازی و دوباره ذخیره کنید.' );
		}
		$wpdb->query( 'START TRANSACTION' );
		if ( false === $wpdb->query( "UPDATE {$t['subscriptions']} SET is_active=0,updated_at='" . esc_sql( $now ) . "'" )
			|| false === $wpdb->query( "UPDATE {$t['contents']} SET is_active=0,updated_at='" . esc_sql( $now ) . "'" )
			|| false === $wpdb->query( "UPDATE {$t['rules']} SET is_active=0,updated_at='" . esc_sql( $now ) . "'" ) ) {
			$wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_project_settings_reset', 'آماده‌سازی نگاشت پروژه انجام نشد.' );
		}
		foreach ( $selected as $subscription_id => $contents ) {
			$sql = $wpdb->prepare( "INSERT INTO {$t['subscriptions']} (subscription_product_id,is_active,created_by,created_at,updated_at) VALUES (%d,1,%d,%s,%s) ON DUPLICATE KEY UPDATE is_active=1,updated_at=VALUES(updated_at)", $subscription_id, $actor, $now, $now );
			if ( false === $wpdb->query( $sql ) ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_project_subscription_save', 'ذخیره محصول اشتراک انجام نشد.' ); }
			foreach ( $contents as $sort => $content_id ) {
				$sql = $wpdb->prepare( "INSERT INTO {$t['contents']} (subscription_product_id,content_product_id,sort_order,is_active,created_at,updated_at) VALUES (%d,%d,%d,1,%s,%s) ON DUPLICATE KEY UPDATE sort_order=VALUES(sort_order),is_active=1,updated_at=VALUES(updated_at)", $subscription_id, $content_id, $sort, $now, $now );
				if ( false === $wpdb->query( $sql ) ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_project_content_save', 'ذخیره محتوای اشتراک انجام نشد.' ); }
			}
		}
		if ( $route_controls_present ) {
			if ( $content_set ) {
				$ids_sql = implode( ',', array_map( 'intval', array_keys( $content_set ) ) );
				if ( false === $wpdb->query( "DELETE FROM {$t['managers']} WHERE content_product_id NOT IN ({$ids_sql})" ) ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_project_manager_cleanup', 'پاک‌سازی نگاشت مدیر انجام نشد.' ); }
				if ( false === $wpdb->query( "DELETE FROM {$t['manager_routes']} WHERE content_product_id NOT IN ({$ids_sql})" ) ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_project_manager_route_cleanup', 'پاک‌سازی مسیرهای مدیر فروش انجام نشد.' ); }
			} else {
				if ( false === $wpdb->query( "DELETE FROM {$t['managers']}" ) || false === $wpdb->query( "DELETE FROM {$t['manager_routes']}" ) ) {
					$wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_project_manager_route_cleanup', 'پاک‌سازی مسیرهای مدیر فروش انجام نشد.' );
				}
			}
		}
		if ( $operations_route_controls_present ) {
			if ( $content_set ) {
				$ids_sql = implode( ',', array_map( 'intval', array_keys( $content_set ) ) );
				if ( false === $wpdb->query( "DELETE FROM {$t['operations_managers']} WHERE content_product_id NOT IN ({$ids_sql})" )
					|| false === $wpdb->query( "DELETE FROM {$t['operations_manager_routes']} WHERE content_product_id NOT IN ({$ids_sql})" ) ) {
					$wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_operations_manager_cleanup', 'پاک‌سازی تنظیمات قدیمی مسیر عملیات انجام نشد.' );
				}
			} elseif ( false === $wpdb->query( "DELETE FROM {$t['operations_managers']}" ) || false === $wpdb->query( "DELETE FROM {$t['operations_manager_routes']}" ) ) {
				$wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_operations_manager_cleanup', 'پاک‌سازی تنظیمات مسیر عملیات انجام نشد.' );
			}
		}
		if ( $operations_supervisor_route_controls_present ) {
			if ( $content_set ) {
				$ids_sql = implode( ',', array_map( 'intval', array_keys( $content_set ) ) );
				if ( false === $wpdb->query( "DELETE FROM {$t['operations_supervisor_routes']} WHERE content_product_id NOT IN ({$ids_sql})" ) ) {
					$wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_operations_supervisor_cleanup', 'پاک‌سازی مسیرهای قدیمی سرپرستان عملیات انجام نشد.' );
				}
			} elseif ( false === $wpdb->query( "DELETE FROM {$t['operations_supervisor_routes']}" ) ) {
				$wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_operations_supervisor_cleanup', 'پاک‌سازی مسیرهای سرپرستان عملیات انجام نشد.' );
			}
		}
		foreach ( array_keys( $content_set ) as $content_id ) {
			if ( $route_controls_present ) {
				if ( isset( $managers[ $content_id ] ) ) {
					$sql = $wpdb->prepare( "INSERT INTO {$t['managers']} (content_product_id,manager_user_id,created_by,created_at,updated_at) VALUES (%d,%d,%d,%s,%s) ON DUPLICATE KEY UPDATE manager_user_id=VALUES(manager_user_id),updated_at=VALUES(updated_at)", $content_id, $managers[ $content_id ], $actor, $now, $now );
					if ( false === $wpdb->query( $sql ) ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_project_manager_save', 'ذخیره مدیر پروژه انجام نشد.' ); }
				}
				if ( false === $wpdb->delete( $t['manager_routes'], [ 'content_product_id' => $content_id ] ) ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_project_manager_route_reset', 'بازنشانی مسیرهای مدیر فروش انجام نشد.' ); }
				foreach ( (array) ( $manager_routes[ $content_id ] ?? [] ) as $route ) {
					$ok = $wpdb->insert( $t['manager_routes'], [
						'content_product_id' => $content_id,
						'sales_manager_user_id' => (int) $route['sales_manager_user_id'],
						'manager_user_id' => (int) $route['manager_user_id'],
						'created_by' => $actor ?: null,
						'created_at' => $now,
						'updated_at' => $now,
					] );
					if ( false === $ok ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_project_manager_route_save', 'ذخیره مسیر اختصاصی مدیر فروش انجام نشد.' ); }
				}
			}
			if ( $operations_route_controls_present ) {
				$operations_manager_id = (int) ( $operations_managers[ $content_id ] ?? 0 );
				$sql = $wpdb->prepare( "INSERT INTO {$t['operations_managers']} (content_product_id,operations_manager_user_id,created_by,created_at,updated_at) VALUES (%d,%d,%d,%s,%s) ON DUPLICATE KEY UPDATE operations_manager_user_id=VALUES(operations_manager_user_id),updated_at=VALUES(updated_at)", $content_id, $operations_manager_id, $actor, $now, $now );
				if ( false === $wpdb->query( $sql ) ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_operations_manager_save', 'ذخیره مدیر فروش عملیات پیش‌فرض انجام نشد.' ); }
				if ( false === $wpdb->delete( $t['operations_manager_routes'], [ 'content_product_id' => $content_id ] ) ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_operations_manager_route_reset', 'بازنشانی مسیرهای عملیات انجام نشد.' ); }
				foreach ( (array) ( $operations_manager_routes[ $content_id ] ?? [] ) as $route ) {
					$ok = $wpdb->insert( $t['operations_manager_routes'], [
						'content_product_id' => $content_id,
						'sales_manager_user_id' => (int) $route['sales_manager_user_id'],
						'operations_manager_user_id' => (int) $route['operations_manager_user_id'],
						'created_by' => $actor ?: null,
						'created_at' => $now,
						'updated_at' => $now,
					] );
					if ( false === $ok ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_operations_manager_route_save', 'ذخیره مسیر اختصاصی مدیر فروش عملیات انجام نشد.' ); }
				}
			}
			if ( $operations_supervisor_route_controls_present ) {
				$operations_manager_id = (int) ( $operations_managers[ $content_id ] ?? 0 );
				$sql = $wpdb->prepare( "INSERT INTO {$t['operations_managers']} (content_product_id,operations_manager_user_id,created_by,created_at,updated_at) VALUES (%d,%d,%d,%s,%s) ON DUPLICATE KEY UPDATE operations_manager_user_id=VALUES(operations_manager_user_id),updated_at=VALUES(updated_at)", $content_id, $operations_manager_id, $actor, $now, $now );
				if ( false === $wpdb->query( $sql ) ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_operations_supervisor_manager_save', 'ذخیره مدیر فروش عملیات کارت انجام نشد.' ); }
				if ( false === $wpdb->delete( $t['operations_supervisor_routes'], [ 'content_product_id' => $content_id ] ) ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_operations_supervisor_route_reset', 'بازنشانی مسیرهای سرپرستان عملیات انجام نشد.' ); }
				// The legacy manager-to-manager overrides are superseded by this card's
				// direct supervisor routes. Removing them avoids a hidden fallback that
				// the administrator can no longer see in this screen.
				if ( $this->table_exists( $t['operations_manager_routes'] ) ) { $wpdb->delete( $t['operations_manager_routes'], [ 'content_product_id' => $content_id ] ); }
				foreach ( (array) ( $operations_supervisor_routes[ $content_id ] ?? [] ) as $route ) {
					$ok = $wpdb->insert( $t['operations_supervisor_routes'], [
						'content_product_id' => $content_id,
						'sales_manager_user_id' => (int) $route['sales_manager_user_id'],
						'operations_manager_user_id' => (int) $route['operations_manager_user_id'],
						'operations_supervisor_user_id' => (int) $route['operations_supervisor_user_id'],
						'created_by' => $actor ?: null,
						'created_at' => $now,
						'updated_at' => $now,
					] );
					if ( false === $ok ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_operations_supervisor_route_save', 'ذخیره مسیر مستقیم سرپرست فروش عملیات انجام نشد.' ); }
				}
			}
			$rule = $rules[ $content_id ];
			$sql = $wpdb->prepare( "INSERT INTO {$t['rules']} (content_product_id,commission_type,commission_value,is_active,created_by,created_at,updated_at) VALUES (%d,%s,%f,%d,%d,%s,%s) ON DUPLICATE KEY UPDATE commission_type=VALUES(commission_type),commission_value=VALUES(commission_value),is_active=VALUES(is_active),updated_at=VALUES(updated_at)", $content_id, $rule['type'], $rule['value'], $rule['active'], $actor, $now, $now );
			if ( false === $wpdb->query( $sql ) ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_project_rule_save', 'ذخیره قانون پورسانت بیاوین انجام نشد.' ); }
			$api = $apis[ $content_id ] ?? [ 'endpoint_url' => '', 'api_key' => '', 'project_code' => '', 'is_active' => 0 ];
			$sql = $wpdb->prepare( "INSERT INTO {$t['api_configs']} (content_product_id,endpoint_url,api_key,project_code,is_active,created_by,created_at,updated_at) VALUES (%d,%s,%s,%s,%d,%d,%s,%s) ON DUPLICATE KEY UPDATE endpoint_url=VALUES(endpoint_url),api_key=VALUES(api_key),project_code=VALUES(project_code),is_active=VALUES(is_active),updated_at=VALUES(updated_at)", $content_id, $api['endpoint_url'], $api['api_key'], $api['project_code'], $api['is_active'], $actor, $now, $now );
			if ( false === $wpdb->query( $sql ) ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_project_api_save', 'ذخیره اتصال API پروژه انجام نشد.' ); }
		}
		$wpdb->query( 'COMMIT' );
		$this->subscription_ids_cache = null; $this->content_ids_cache = null;
		$this->event( 0, 0, 'project_settings_saved', [
			'subscriptions' => array_keys( $selected ),
			'contents' => array_keys( $content_set ),
			'manager_routes_updated' => $route_controls_present,
			'manager_route_count' => array_sum( array_map( 'count', $manager_routes ) ),
			'operations_routes_updated' => $operations_route_controls_present,
			'operations_route_count' => array_sum( array_map( 'count', $operations_manager_routes ) ),
			'operations_supervisor_routes_updated' => $operations_supervisor_route_controls_present,
			'operations_supervisor_route_count' => array_sum( array_map( 'count', $operations_supervisor_routes ) ),
		], $actor );
		return true;
	}

	public function render_settings_panel(): void {
		$products = class_exists( 'SN_Helpers' ) ? SN_Helpers::get_sn_products() : [];
		$maps = $this->settings_maps();
		$sales_managers = $this->sales_manager_users();
		$operations_managers = $this->operations_sales_manager_users();
		$operations_supervisors = $this->operations_sales_supervisor_users();
		$subscription_rows = (array) $maps['subscriptions'];
		if ( ! $subscription_rows ) { $subscription_rows = [ 0 => [] ]; }
		$product_options_html = '<option value="0">— انتخاب محصول اشتراک —</option>';
		$content_options_html = '';
		$sales_manager_options_html = '<option value="0">— مدیر فروش مبدأ —</option>';
		$operations_supervisor_options_html = '<option value="0">— سرپرست فروش عملیات مقصد —</option>';
		foreach ( $sales_managers as $sales_manager ) { $sales_manager_options_html .= '<option value="' . esc_attr( (string) $sales_manager->ID ) . '">' . esc_html( (string) $sales_manager->display_name . ' (#' . (int) $sales_manager->ID . ')' ) . '</option>'; }
		foreach ( $operations_supervisors as $supervisor ) { $operations_supervisor_options_html .= '<option value="' . esc_attr( (string) $supervisor->ID ) . '" data-manager="' . esc_attr( (string) (int) ( $supervisor->sn_operations_manager_user_id ?? 0 ) ) . '">' . esc_html( (string) $supervisor->display_name . ' (#' . (int) $supervisor->ID . ')' ) . '</option>'; }
		foreach ( $products as $row_product ) {
			$pid = (int) $row_product['id'];
			$label = esc_html( (string) $row_product['name'] . ' (#' . $pid . ')' );
			$product_options_html .= '<option value="' . esc_attr( (string) $pid ) . '">' . $label . '</option>';
			$content_options_html .= '<option value="' . esc_attr( (string) $pid ) . '">' . $label . '</option>';
		}
		?>
		<input type="hidden" name="sn_projects_settings_present" value="1">
		<input type="hidden" name="sn_operations_supervisor_routes_present" value="1">
		<div class="sn-project-admin-settings" dir="rtl">
			<h2>پروژه‌ها و اشتراک‌های بیاوین</h2>
			<p class="description">محصول اشتراک قابل فروش است؛ محصولات محتوا فقط داخل پروژه مصرف می‌شوند و برای فروشنده/تبدیل‌کننده در صدور فاکتور نمایش داده نخواهند شد. تغییر این صفحه روی عضویت‌های قبلی اثر نمی‌گذارد.</p>
			<table class="form-table" style="max-width:820px"><tr><th>عدم انجام عمل توسط مشتری روی کارت</th><td><input type="number" name="sn_operations_customer_inactivity_days" min="0" max="365" step="1" value="<?php echo esc_attr( (string) min( 365, max( 0, absint( get_option( 'sn_operations_customer_inactivity_days', 3 ) ) ) ) ); ?>" class="small-text"> روز<p class="description">پس از این تعداد روز، کارتِ بدون انتخاب وارد صف مدیر فروش عملیات می‌شود. مقدار صفر یعنی ارسال فوری همه کارت‌های خودکارِ در انتظار و حذف امکان فعال‌سازی توسط مشتری. رکوردهای تاریخی که مجوز مسیریابی خودکار ندارند جابه‌جا نمی‌شوند.</p></td></tr></table>
			<table class="widefat striped" id="sn-project-subscription-table">
				<thead><tr><th style="width:70px">فعال</th><th>محصول ووکامرس اشتراک</th><th>محتویات اشتراک (چند انتخابی)</th><th style="width:80px"></th></tr></thead>
				<tbody>
				<?php if ( ! $products ) : ?><tr class="sn-project-no-products"><td colspan="4">محصول فعال شبکه فروش یافت نشد.</td></tr><?php endif; ?>
				<?php $row_index = 0; foreach ( $subscription_rows as $subscription_id => $selected_contents ) : $subscription_id = (int) $subscription_id; ?>
				<tr class="sn-project-subscription-row">
					<td><input type="checkbox" name="sn_projects[subscription_rows][<?php echo esc_attr( (string) $row_index ); ?>][active]" value="1" checked></td>
					<td><select class="sn-project-subscription-product" name="sn_projects[subscription_rows][<?php echo esc_attr( (string) $row_index ); ?>][product_id]" style="width:100%;max-width:420px"><option value="0">— انتخاب محصول اشتراک —</option><?php foreach ( $products as $product ) : $pid = (int) $product['id']; ?><option value="<?php echo esc_attr( (string) $pid ); ?>" <?php selected( $subscription_id, $pid ); ?>><?php echo esc_html( (string) $product['name'] . ' (#' . $pid . ')' ); ?></option><?php endforeach; ?></select></td>
					<td><select class="sn-project-subscription-contents" multiple size="5" name="sn_projects[subscription_rows][<?php echo esc_attr( (string) $row_index ); ?>][contents][]" style="width:100%;max-width:620px"><?php foreach ( $products as $content ) : $cid = (int) $content['id']; ?><option value="<?php echo esc_attr( (string) $cid ); ?>" <?php selected( in_array( $cid, (array) $selected_contents, true ) ); ?>><?php echo esc_html( (string) $content['name'] . ' (#' . $cid . ')' ); ?></option><?php endforeach; ?></select></td>
					<td><button type="button" class="button sn-project-remove-subscription">حذف</button></td>
				</tr>
				<?php $row_index++; endforeach; ?>
				</tbody>
			</table>
			<p><button type="button" class="button" id="sn-project-add-subscription">+ افزودن اشتراک</button></p>

			<h3 style="margin-top:28px">مدیر و سرپرست فروش عملیات هر کارت، API مقصد و پورسانت بیاوین</h3>
			<p class="description">برای هر کارت یک مدیر فروش عملیات انتخاب کنید؛ سپس مشخص کنید فروش زیرمجموعه هر مدیر فروش مستقیماً به کدام سرپرستِ زیرمجموعه همان مدیر عملیات برسد. مقصد هنگام ایجاد کارت ذخیره می‌شود و تغییر تنظیمات فقط روی خریدهای بعدی اثر دارد.</p>
			<table class="widefat striped">
				<thead><tr><th>محصول محتوا / کارت</th><th style="min-width:460px">مدیران فروش عملیات و مسیر شاخه‌های فروش</th><th>آدرس API مقصد</th><th>کد پروژه</th><th>کلید API</th><th>API فعال</th><th>نوع پورسانت</th><th>مقدار</th><th>پورسانت فعال</th></tr></thead>
				<tbody>
				<?php foreach ( $products as $product ) : $pid = (int) $product['id']; $rule = (array) ( $maps['rules'][ $pid ] ?? [] ); $api = (array) ( $maps['api_configs'][ $pid ] ?? [] ); ?>
				<tr class="sn-project-content-config-row" data-content-product="<?php echo esc_attr( (string) $pid ); ?>">
					<td><strong><?php echo esc_html( (string) $product['name'] ); ?></strong><br><code>#<?php echo esc_html( (string) $pid ); ?></code></td>
					<td>
						<div class="sn-project-route-box sn-operations-route-box">
						<label><strong>مدیر فروش عملیات کارت</strong><br><select class="sn-operations-card-manager" name="sn_projects[operations_managers][<?php echo esc_attr( (string) $pid ); ?>]" style="width:100%;max-width:420px"><option value="0">— انتخاب مدیر فروش عملیات —</option><?php foreach ( $operations_managers as $manager ) : ?><option value="<?php echo esc_attr( (string) $manager->ID ); ?>" <?php selected( (int) ( $maps['operations_managers'][ $pid ] ?? 0 ), (int) $manager->ID ); ?>><?php echo esc_html( (string) $manager->display_name . ' (#' . (int) $manager->ID . ')' ); ?></option><?php endforeach; ?></select></label>
						<p class="description" style="margin:6px 0">اگر برای شاخه فروش مسیر مستقیم تعریف نشده باشد، کارت ابتدا وارد صف همین مدیر عملیات می‌شود.</p>
						<div><strong>مدیر فروش مبدأ ← سرپرست فروش عملیات مقصد</strong><div class="sn-operations-manager-route-list" data-content-product="<?php echo esc_attr( (string) $pid ); ?>">
							<?php $operations_route_index = 0; foreach ( (array) ( $maps['operations_supervisor_routes'][ $pid ] ?? [] ) as $route ) : ?>
							<div class="sn-operations-manager-route-row" style="display:flex;gap:6px;align-items:center;margin-top:6px;flex-wrap:wrap">
								<select name="sn_projects[operations_supervisor_routes][<?php echo esc_attr( (string) $pid ); ?>][<?php echo esc_attr( (string) $operations_route_index ); ?>][sales_manager_user_id]" style="min-width:180px"><option value="0">— مدیر فروش مبدأ —</option><?php foreach ( $sales_managers as $sales_manager ) : ?><option value="<?php echo esc_attr( (string) $sales_manager->ID ); ?>" <?php selected( (int) ( $route['sales_manager_user_id'] ?? 0 ), (int) $sales_manager->ID ); ?>><?php echo esc_html( (string) $sales_manager->display_name . ' (#' . (int) $sales_manager->ID . ')' ); ?></option><?php endforeach; ?></select>
								<span>←</span>
								<select class="sn-operations-route-supervisor" name="sn_projects[operations_supervisor_routes][<?php echo esc_attr( (string) $pid ); ?>][<?php echo esc_attr( (string) $operations_route_index ); ?>][operations_supervisor_user_id]" style="min-width:220px"><option value="0">— سرپرست فروش عملیات مقصد —</option><?php foreach ( $operations_supervisors as $supervisor ) : ?><option value="<?php echo esc_attr( (string) $supervisor->ID ); ?>" data-manager="<?php echo esc_attr( (string) (int) ( $supervisor->sn_operations_manager_user_id ?? 0 ) ); ?>" <?php selected( (int) ( $route['operations_supervisor_user_id'] ?? 0 ), (int) $supervisor->ID ); ?>><?php echo esc_html( (string) $supervisor->display_name . ' (#' . (int) $supervisor->ID . ')' ); ?></option><?php endforeach; ?></select>
								<button type="button" class="button-link-delete sn-project-remove-operations-route" aria-label="حذف مسیر عملیات">حذف</button>
							</div>
							<?php $operations_route_index++; endforeach; ?>
						</div><button type="button" class="button button-small sn-project-add-operations-route" data-content-product="<?php echo esc_attr( (string) $pid ); ?>" data-next-index="<?php echo esc_attr( (string) $operations_route_index ); ?>" style="margin-top:7px">+ افزودن مسیر سرپرست عملیات</button></div></div>
					</td>
					<td><input type="url" name="sn_projects[apis][<?php echo esc_attr( (string) $pid ); ?>][endpoint_url]" value="<?php echo esc_attr( (string) ( $api['endpoint_url'] ?? '' ) ); ?>" placeholder="https://target.example/wp-json/dcw/v1/sn" style="min-width:240px" dir="ltr"></td>
					<td><input type="text" name="sn_projects[apis][<?php echo esc_attr( (string) $pid ); ?>][project_code]" value="<?php echo esc_attr( (string) ( $api['project_code'] ?? '' ) ); ?>" placeholder="shoe-project" pattern="[A-Za-z0-9_.:-]+" dir="ltr" style="width:130px"></td>
					<td><input type="password" name="sn_projects[apis][<?php echo esc_attr( (string) $pid ); ?>][api_key]" value="" placeholder="<?php echo ! empty( $api['api_key'] ) ? '•••••••••••• (ثبت شده)' : 'کلید مقصد'; ?>" autocomplete="new-password" dir="ltr" style="width:150px"></td>
					<td><input type="checkbox" name="sn_projects[apis][<?php echo esc_attr( (string) $pid ); ?>][is_active]" value="1" <?php checked( ! empty( $api['is_active'] ) ); ?>></td>
					<td><select name="sn_projects[rules][<?php echo esc_attr( (string) $pid ); ?>][type]"><option value="percent" <?php selected( (string) ( $rule['commission_type'] ?? 'percent' ), 'percent' ); ?>>درصد از فروش تاییدشده</option><option value="fixed" <?php selected( (string) ( $rule['commission_type'] ?? '' ), 'fixed' ); ?>>مبلغ ثابت برای فاکتور</option></select></td>
					<td><input type="number" min="0" step="0.01" name="sn_projects[rules][<?php echo esc_attr( (string) $pid ); ?>][value]" value="<?php echo esc_attr( (string) ( $rule['commission_value'] ?? 0 ) ); ?>"></td>
					<td><input type="checkbox" name="sn_projects[rules][<?php echo esc_attr( (string) $pid ); ?>][active]" value="1" <?php checked( ! empty( $rule['is_active'] ) ); ?>></td>
				</tr>
				<?php endforeach; ?>
				</tbody>
			</table>
		</div>
		<script>
		(function(){
			var table=document.querySelector('#sn-project-subscription-table tbody'),add=document.getElementById('sn-project-add-subscription'),nextIndex=<?php echo (int) $row_index; ?>;
			var productOptions=<?php echo wp_json_encode( $product_options_html, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ); ?>;
			var contentOptions=<?php echo wp_json_encode( $content_options_html, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ); ?>;
			var salesManagerOptions=<?php echo wp_json_encode( $sales_manager_options_html, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ); ?>;
			var operationsSupervisorOptions=<?php echo wp_json_encode( $operations_supervisor_options_html, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ); ?>;
			function filterOperationsSupervisors(box){var manager=box.querySelector('.sn-operations-card-manager'),managerId=manager?String(manager.value||'0'):'0';box.querySelectorAll('.sn-operations-route-supervisor').forEach(function(select){var selected=select.value;select.querySelectorAll('option').forEach(function(option){var allowed=!option.value||String(option.dataset.manager||'0')===managerId;option.hidden=!allowed;option.disabled=!allowed;});if(selected&&select.options[select.selectedIndex]&&select.options[select.selectedIndex].disabled){select.value='0';}});}
			function refreshContentRows(){
				var selected={};document.querySelectorAll('.sn-project-subscription-contents option:checked').forEach(function(option){selected[String(option.value)]=true;});
				document.querySelectorAll('.sn-project-content-config-row').forEach(function(row){row.style.display=selected[String(row.dataset.contentProduct||'')]?'':'none';});
			}
			if(add&&table){add.addEventListener('click',function(){var i=nextIndex++,row=document.createElement('tr');row.className='sn-project-subscription-row';row.innerHTML='<td><input type="checkbox" name="sn_projects[subscription_rows]['+i+'][active]" value="1" checked></td><td><select class="sn-project-subscription-product" name="sn_projects[subscription_rows]['+i+'][product_id]" style="width:100%;max-width:420px">'+productOptions+'</select></td><td><select class="sn-project-subscription-contents" multiple size="5" name="sn_projects[subscription_rows]['+i+'][contents][]" style="width:100%;max-width:620px">'+contentOptions+'</select></td><td><button type="button" class="button sn-project-remove-subscription">حذف</button></td>';table.appendChild(row);});}
			document.addEventListener('click',function(event){
				if(!event.target){return;}
				if(event.target.classList.contains('sn-project-remove-subscription')){var subscriptionRow=event.target.closest('tr');if(subscriptionRow){subscriptionRow.remove();refreshContentRows();}return;}
				if(event.target.classList.contains('sn-project-remove-operations-route')){var operationsRouteRow=event.target.closest('.sn-operations-manager-route-row');if(operationsRouteRow){operationsRouteRow.remove();}return;}
				if(event.target.classList.contains('sn-project-add-operations-route')){var operationsButton=event.target,operationsPid=String(operationsButton.dataset.contentProduct||''),operationsList=operationsButton.parentNode.querySelector('.sn-operations-manager-route-list'),operationsIndex=parseInt(operationsButton.dataset.nextIndex||'0',10);if(!operationsPid||!operationsList){return;}operationsButton.dataset.nextIndex=String(operationsIndex+1);var operationsRoute=document.createElement('div');operationsRoute.className='sn-operations-manager-route-row';operationsRoute.style.cssText='display:flex;gap:6px;align-items:center;margin-top:6px;flex-wrap:wrap';operationsRoute.innerHTML='<select name="sn_projects[operations_supervisor_routes]['+operationsPid+']['+operationsIndex+'][sales_manager_user_id]" style="min-width:180px">'+salesManagerOptions+'</select><span>←</span><select class="sn-operations-route-supervisor" name="sn_projects[operations_supervisor_routes]['+operationsPid+']['+operationsIndex+'][operations_supervisor_user_id]" style="min-width:220px">'+operationsSupervisorOptions+'</select><button type="button" class="button-link-delete sn-project-remove-operations-route" aria-label="حذف مسیر عملیات">حذف</button>';operationsList.appendChild(operationsRoute);filterOperationsSupervisors(operationsButton.closest('.sn-operations-route-box'));}
			});
			document.addEventListener('change',function(event){if(!event.target){return;}if(event.target.classList.contains('sn-project-subscription-contents')){refreshContentRows();}if(event.target.classList.contains('sn-operations-card-manager')){filterOperationsSupervisors(event.target.closest('.sn-operations-route-box'));}});
			document.querySelectorAll('.sn-operations-route-box').forEach(filterOperationsSupervisors);
			refreshContentRows();
		})();
		</script>
		<?php
	}

	private function current_role(): string {
		if ( current_user_can( 'manage_options' ) ) { return 'admin'; }
		$position = $this->position_for_user( get_current_user_id() );
		if ( in_array( $position, [ 'project_manager', 'project_expert', 'seller', 'converter', 'operations_sales_expert', 'operations_sales_supervisor' ], true ) ) { return $position; }
		if ( in_array( $position, [ 'supervisor', 'senior_supervisor', 'sales_manager', 'sales_deputy' ], true ) ) { return 'sales_supervisor'; }
		return '';
	}

	private function sales_chat_scope_user_ids( int $viewer_id ): array {
		$ids = [ $viewer_id ];
		global $wpdb;
		$profiles = $wpdb->prefix . 'sn_hr_profiles'; $assignments = $wpdb->prefix . 'sn_hr_assignments';
		if ( $this->table_exists( $profiles ) && $this->table_exists( $assignments ) ) {
			$frontier = [ $viewer_id ]; $seen = [];
			while ( $frontier && count( $seen ) < 2000 ) {
				$parent_user = array_shift( $frontier ); if ( isset( $seen[ $parent_user ] ) ) { continue; } $seen[ $parent_user ] = true;
				$child_ids = $wpdb->get_col( $wpdb->prepare( "SELECT child.user_id FROM {$assignments} a INNER JOIN {$profiles} parent ON parent.id=a.parent_profile_id INNER JOIN {$profiles} child ON child.id=a.child_profile_id WHERE parent.user_id=%d AND a.relationship_type='reports_to' AND a.is_current=1 AND child.is_active=1", $parent_user ) );
				foreach ( (array) $child_ids as $child_id ) { $child_id = (int) $child_id; if ( $child_id > 0 && ! isset( $seen[ $child_id ] ) ) { $ids[] = $child_id; $frontier[] = $child_id; } }
			}
		}
		$legacy = get_users( [ 'role__in' => [ 'sn_seller', 'sn_converter' ], 'fields' => 'ID', 'number' => 2000, 'meta_key' => 'sn_supervisor_id', 'meta_value' => $viewer_id ] );
		return array_values( array_unique( array_filter( array_map( 'intval', array_merge( $ids, (array) $legacy ) ) ) ) );
	}

	private function verify_ajax(): void {
		if ( ! is_user_logged_in() ) { wp_send_json( [ 'success' => false, 'message' => 'ابتدا وارد حساب کاربری شوید.' ], 401 ); }
		if ( ! check_ajax_referer( 'sn_projects', 'nonce', false ) ) { wp_send_json( [ 'success' => false, 'message' => 'درخواست امنیتی نامعتبر است.' ], 403 ); }
	}

	private function item_context( int $item_id ) {
		global $wpdb; $t = $this->tables();
		if ( $item_id < 1 ) { return null; }
		return $wpdb->get_row( $wpdb->prepare(
			"SELECT mi.*,m.source_type,m.source_invoice_id,m.dot_case_id,m.subscription_product_id,m.subscription_name_snapshot,m.customer_wp_id,m.customer_name,m.customer_phone,m.origin_user_id,m.origin_role,m.sales_chat_user_id,m.sales_chat_role,m.status membership_status,m.activated_at FROM {$t['items']} mi INNER JOIN {$t['memberships']} m ON m.id=mi.membership_id WHERE mi.id=%d LIMIT 1",
			$item_id
		) );
	}

	private function product_credit( int $product_id ): float {
		return max( 0, (float) get_post_meta( $product_id, '_sn_product_credit_amount', true ) );
	}

	private function is_product_star_definition( int $product_id ): bool {
		if ( $product_id < 1 || get_post_type( $product_id ) !== 'product' ) { return false; }
		if ( class_exists( 'SN_Seller_Flow' ) ) {
			return SN_Seller_Flow::instance()->product_type( $product_id ) === 'product_star';
		}
		$type = sanitize_key( (string) get_post_meta( $product_id, '_sn_sales_product_type', true ) );
		return in_array( $type, [ 'product_star', 'subscription_star' ], true ) || $this->is_content_product( $product_id );
	}

	/** Product* choices available while editing a source card. */
	public function upgrade_target_products( int $exclude_product_id = 0 ): array {
		$ids = get_posts( [
			'post_type' => 'product',
			'post_status' => [ 'publish', 'draft', 'pending', 'private', 'future' ],
			'numberposts' => -1,
			'orderby' => 'title',
			'order' => 'ASC',
			'fields' => 'ids',
			'suppress_filters' => false,
		] );
		$out = [];
		foreach ( (array) $ids as $product_id ) {
			$product_id = (int) $product_id;
			if ( $product_id === $exclude_product_id || ! $this->is_product_star_definition( $product_id ) ) { continue; }
			$out[] = [
				'id' => $product_id,
				'name' => sanitize_text_field( (string) get_the_title( $product_id ) ),
				'credit' => $this->product_credit( $product_id ),
				'post_status' => sanitize_key( (string) get_post_status( $product_id ) ),
			];
		}
		return $out;
	}

	/** Live upgrade rules for a Product*. Existing card snapshots use their own copy. */
	public function product_upgrade_rules( int $source_product_id ): array {
		global $wpdb; $table = $this->tables()['upgrade_rules'];
		if ( $source_product_id < 1 || ! $this->table_exists( $table ) ) { return []; }
		$rows = $wpdb->get_results( $wpdb->prepare(
			"SELECT target_product_id,upgrade_amount FROM {$table} WHERE source_product_id=%d AND is_active=1 ORDER BY id",
			$source_product_id
		), ARRAY_A ) ?: [];
		$out = [];
		foreach ( $rows as $row ) {
			$target_id = absint( $row['target_product_id'] ?? 0 );
			$amount = max( 0, (float) ( $row['upgrade_amount'] ?? 0 ) );
			$target_credit = $this->product_credit( $target_id );
			if ( $target_id < 1 || get_post_status( $target_id ) === 'trash' || ! $this->is_product_star_definition( $target_id ) ) { continue; }
			$out[] = [
				'id' => $target_id,
				'product_id' => $target_id,
				'name' => sanitize_text_field( (string) get_the_title( $target_id ) ),
				'credit' => $target_credit,
				'target_credit' => $target_credit,
				'price' => $amount,
				'upgrade_amount' => $amount,
			];
		}
		return $out;
	}

	/**
	 * Replace only the live rules of one Product*. Memberships/actions are not
	 * touched. Invalid submissions leave the complete previous set intact.
	 */
	public function save_product_upgrade_rules( int $source_product_id, array $submitted_rows, int $actor_user_id ) {
		if ( ! $this->is_product_star_definition( $source_product_id ) ) {
			return new WP_Error( 'sn_upgrade_source_type', 'تعریف حالت افزایشی فقط برای محصول نوع «محصول*» امکان‌پذیر است.' );
		}
		global $wpdb; $table = $this->tables()['upgrade_rules'];
		if ( ! $this->table_exists( $table ) ) { $this->install_schema(); }
		if ( ! $this->table_exists( $table ) ) { return new WP_Error( 'sn_upgrade_schema', 'جدول تنظیمات افزایشی در دسترس نیست.' ); }
		$valid = []; $seen = [];
		foreach ( $submitted_rows as $row ) {
			if ( ! is_array( $row ) ) { continue; }
			$target_id = absint( $row['target_product_id'] ?? 0 );
			$raw = class_exists( 'SN_Helpers' ) ? SN_Helpers::to_english_nums( (string) ( $row['upgrade_amount'] ?? '' ) ) : (string) ( $row['upgrade_amount'] ?? '' );
			$amount = max( 0, (float) str_replace( [ ',', '٬', '،', ' ' ], '', $raw ) );
			if ( $target_id < 1 && $amount <= 0 ) { continue; }
			if ( $target_id < 1 || $target_id === $source_product_id || ! $this->is_product_star_definition( $target_id ) ) {
				return new WP_Error( 'sn_upgrade_target', 'یکی از محصولات مقصد افزایشی نامعتبر است.' );
			}
			if ( isset( $seen[ $target_id ] ) ) { return new WP_Error( 'sn_upgrade_duplicate', 'یک محصول مقصد را نمی‌توان دو بار برای یک کارت تعریف کرد.' ); }
			if ( trim( $raw ) === '' || ! is_numeric( str_replace( [ ',', '٬', '،', ' ' ], '', $raw ) ) || (float) str_replace( [ ',', '٬', '،', ' ' ], '', $raw ) < 0 ) {
				return new WP_Error( 'sn_upgrade_amount', 'مبلغ ارتقا باید صفر یا یک عدد مثبت باشد.' );
			}
			$seen[ $target_id ] = true;
			$valid[] = [ 'target_product_id' => $target_id, 'upgrade_amount' => $amount ];
		}
		$now = current_time( 'mysql' );
		if ( false === $wpdb->query( 'START TRANSACTION' ) ) { return new WP_Error( 'sn_upgrade_transaction', 'شروع ذخیره امن تنظیمات افزایشی انجام نشد.' ); }
		$deleted = $wpdb->delete( $table, [ 'source_product_id' => $source_product_id ], [ '%d' ] );
		if ( false === $deleted ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_upgrade_delete', 'ذخیره تنظیمات افزایشی انجام نشد.' ); }
		foreach ( $valid as $row ) {
			$ok = $wpdb->insert( $table, [
				'source_product_id' => $source_product_id,
				'target_product_id' => $row['target_product_id'],
				'upgrade_amount' => $row['upgrade_amount'],
				'is_active' => 1,
				'created_by' => $actor_user_id ?: null,
				'created_at' => $now,
				'updated_at' => $now,
			] );
			if ( ! $ok ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_upgrade_insert', 'ذخیره یکی از گزینه‌های افزایشی انجام نشد.' ); }
		}
		if ( false === $wpdb->query( 'COMMIT' ) ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_upgrade_commit', 'ثبت نهایی تنظیمات افزایشی انجام نشد.' ); }
		return true;
	}

	private function item_has_upgrade_snapshot( object $item ): bool {
		return property_exists( $item, 'upgrade_options_snapshot_json' ) && null !== $item->upgrade_options_snapshot_json && (string) $item->upgrade_options_snapshot_json !== '';
	}

	private function item_base_credit( object $item ): float {
		if ( property_exists( $item, 'base_credit_snapshot' ) && null !== $item->base_credit_snapshot ) {
			return max( 0, (float) $item->base_credit_snapshot );
		}
		return $this->product_credit( (int) ( $item->content_product_id ?? 0 ) );
	}

	private function item_upgrade_options( object $item ): array {
		if ( $this->item_has_upgrade_snapshot( $item ) ) {
			$decoded = json_decode( (string) $item->upgrade_options_snapshot_json, true );
			return is_array( $decoded ) ? $decoded : [];
		}
		return $this->product_upgrade_rules( (int) ( $item->content_product_id ?? 0 ) );
	}

	private function mark_action_invoice_paid( int $invoice_id ): void {
		if ( $invoice_id < 1 ) { return; }
		global $wpdb; $table = $this->tables()['actions'];
		if ( ! $this->table_exists( $table ) ) { return; }
		$now = current_time( 'mysql' );
		$rows = $wpdb->get_results( $wpdb->prepare( "SELECT id,membership_item_id FROM {$table} WHERE invoice_id=%d AND status IN ('awaiting_payment','retry_requested')", $invoice_id ) ) ?: [];
		$updated = $wpdb->query( $wpdb->prepare( "UPDATE {$table} SET status='paid',last_payment_at=%s,updated_at=%s WHERE invoice_id=%d AND status IN ('awaiting_payment','retry_requested')", $now, $now, $invoice_id ) );
		if ( $updated && $rows ) {
			foreach ( $rows as $row ) { $this->event( 0, (int) $row->membership_item_id, 'project_action_paid', [ 'action_id' => (int) $row->id, 'invoice_id' => $invoice_id ] ); }
		}
	}

	private function is_project_action_invoice( int $invoice_id ): bool {
		if ( $invoice_id < 1 ) { return false; }
		global $wpdb; $table = $this->tables()['actions'];
		return $this->table_exists( $table ) && (bool) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$table} WHERE invoice_id=%d LIMIT 1", $invoice_id ) );
	}

	/** Current Operations-sales chat participant for a membership item. */
	private function operations_sales_chat_assignment( int $item_id ): array {
		if ( $item_id < 1 ) { return [ 'user_id' => 0, 'assigned_at' => '' ]; }
		global $wpdb;
		$table = $wpdb->prefix . 'sn_project_operations';
		if ( ! $this->table_exists( $table ) ) { return [ 'user_id' => 0, 'assigned_at' => '' ]; }
		$row = $wpdb->get_row( $wpdb->prepare( "SELECT sales_expert_user_id,assigned_at FROM {$table} WHERE membership_item_id=%d LIMIT 1", $item_id ) );
		return [
			'user_id' => (int) ( $row->sales_expert_user_id ?? 0 ),
			'assigned_at' => (string) ( $row->assigned_at ?? '' ),
		];
	}

	private function can_access_item( object $item, int $user_id ): bool {
		if ( current_user_can( 'manage_options' ) ) { return true; }
		if ( $user_id < 1 ) { return false; }
		$sales_chat_user_id = (int) ( $item->sales_chat_user_id ?? 0 );
		if ( $sales_chat_user_id < 1 ) { $sales_chat_user_id = (int) $item->origin_user_id; }
		if ( $user_id === $sales_chat_user_id || in_array( $user_id, [ (int) $item->manager_user_id, (int) $item->expert_user_id ], true ) ) { return true; }
		$operations = $this->operations_sales_chat_assignment( (int) $item->id );
		if ( (int) $operations['user_id'] === $user_id ) { return true; }
		$position = $this->position_for_user( $user_id );
		return in_array( $position, [ 'supervisor', 'senior_supervisor', 'sales_manager', 'sales_deputy' ], true ) && in_array( $sales_chat_user_id, $this->sales_chat_scope_user_ids( $user_id ), true );
	}

	private function can_manage_item( object $item, int $user_id ): bool {
		if ( (string) ( $item->membership_status ?? '' ) !== 'active' ) { return false; }
		if ( current_user_can( 'manage_options' ) ) { return true; }
		$position = $this->position_for_user( $user_id );
		if ( $position === 'project_manager' ) { return (int) $item->manager_user_id === $user_id; }
		if ( $position === 'project_expert' ) { return (int) $item->expert_user_id === $user_id; }
		return false;
	}

	private function direct_experts_for_manager( int $manager_user_id ): array {
		if ( current_user_can( 'manage_options' ) ) { return $this->project_users( 'project_expert' ); }
		global $wpdb;
		$profiles = $wpdb->prefix . 'sn_hr_profiles'; $positions = $wpdb->prefix . 'sn_hr_positions'; $assignments = $wpdb->prefix . 'sn_hr_assignments';
		if ( ! $this->table_exists( $profiles ) || ! $this->table_exists( $positions ) || ! $this->table_exists( $assignments ) ) { return []; }
		$ids = $wpdb->get_col( $wpdb->prepare(
			"SELECT child.user_id FROM {$assignments} a INNER JOIN {$profiles} parent ON parent.id=a.parent_profile_id INNER JOIN {$profiles} child ON child.id=a.child_profile_id INNER JOIN {$positions} pos ON pos.id=child.position_id WHERE parent.user_id=%d AND a.relationship_type='reports_to' AND a.is_current=1 AND child.is_active=1 AND pos.slug='project_expert' ORDER BY child.user_id",
			$manager_user_id
		) );
		$out = [];
		foreach ( (array) $ids as $id ) { $user = get_user_by( 'id', (int) $id ); if ( $user instanceof WP_User ) { $out[] = $user; } }
		return $out;
	}

	private function purchase_summary( int $item_id ): array {
		$all = $this->purchase_summaries( [ $item_id ] );
		return $all[ $item_id ] ?? [ 'invoice_count' => 0, 'total' => 0.0 ];
	}

	private function purchase_summaries( array $item_ids ): array {
		global $wpdb; $t = $this->tables();
		$item_ids = array_values( array_unique( array_filter( array_map( 'absint', $item_ids ) ) ) );
		if ( ! $item_ids ) { return []; }
		$ids_sql = implode( ',', array_map( 'intval', $item_ids ) );
		$rows = $wpdb->get_results( "SELECT membership_item_id,COUNT(*) invoice_count,COALESCE(SUM(line_total),0) total FROM {$t['invoice_links']} WHERE status='verified' AND membership_item_id IN ({$ids_sql}) GROUP BY membership_item_id", ARRAY_A ) ?: [];
		$out = [];
		foreach ( $item_ids as $item_id ) { $out[ $item_id ] = [ 'invoice_count' => 0, 'total' => 0.0 ]; }
		foreach ( $rows as $row ) { $out[ (int) $row['membership_item_id'] ] = [ 'invoice_count' => (int) $row['invoice_count'], 'total' => (float) $row['total'] ]; }
		return $out;
	}

	private function linked_invoices( int $item_id ): array {
		global $wpdb; $t = $this->tables();
		$rows = $wpdb->get_results( $wpdb->prepare(
			"SELECT l.invoice_id,l.line_total,l.verified_at,i.invoice_code,i.status,i.paid_at FROM {$t['invoice_links']} l INNER JOIN {$wpdb->prefix}sn_invoices i ON i.id=l.invoice_id WHERE l.membership_item_id=%d AND l.status='verified' ORDER BY l.verified_at DESC,l.id DESC",
			$item_id
		), ARRAY_A ) ?: [];
		foreach ( $rows as &$row ) { $row['invoice_id'] = (int) $row['invoice_id']; $row['line_total'] = (float) $row['line_total']; $row['amount_fmt'] = class_exists( 'SN_Helpers' ) ? SN_Helpers::format_price( $row['line_total'] ) : number_format_i18n( $row['line_total'] ); }
		return $rows;
	}

	private function status_label( string $status ): string {
		return [
			'waiting' => 'در انتظار',
			'no_answer' => 'جواب نداده',
			'follow_up' => 'تماس مجدد',
			'cancelled' => 'انصراف',
			'normal' => 'عادی',
			'upsell' => 'افزایشی',
			// Legacy labels are retained for rows created by earlier releases.
			'purchased' => 'خرید کرده',
			'regretted' => 'پشیمان شده',
		][ $status ] ?? 'در انتظار';
	}

	private function serialize_item( object $item, int $viewer_id, bool $with_invoices = false, ?array $purchase = null ): array {
		$manager = ! empty( $item->manager_user_id ) ? get_user_by( 'id', (int) $item->manager_user_id ) : null;
		$expert = ! empty( $item->expert_user_id ) ? get_user_by( 'id', (int) $item->expert_user_id ) : null;
		$origin = get_user_by( 'id', (int) $item->origin_user_id );
		$sales_chat_user = ! empty( $item->sales_chat_user_id ) ? get_user_by( 'id', (int) $item->sales_chat_user_id ) : null;
		$summary = is_array( $purchase ) ? $purchase : $this->purchase_summary( (int) $item->id );
		$operations_chat = $this->operations_sales_chat_assignment( (int) $item->id );
		$operations_expert = (int) $operations_chat['user_id'] > 0 ? get_user_by( 'id', (int) $operations_chat['user_id'] ) : null;
		$out = [
			'id' => (int) $item->id, 'membership_id' => (int) $item->membership_id,
			'content_product_id' => (int) $item->content_product_id,
			'content_name' => (string) $item->content_name_snapshot,
			'content_credit' => $this->item_base_credit( $item ),
			'subscription_product_id' => (int) $item->subscription_product_id,
			'subscription_name' => (string) $item->subscription_name_snapshot,
			'customer_name' => (string) $item->customer_name,
			'customer_phone' => (string) $item->customer_phone,
			'origin_user_id' => (int) $item->origin_user_id,
			'origin_name' => $origin instanceof WP_User ? (string) $origin->display_name : '#' . (int) $item->origin_user_id,
			'origin_role' => (string) $item->origin_role,
			'sales_chat_user_id' => (int) ( $item->sales_chat_user_id ?? $item->origin_user_id ),
			'sales_chat_name' => $sales_chat_user instanceof WP_User ? (string) $sales_chat_user->display_name : ( $origin instanceof WP_User ? (string) $origin->display_name : '—' ),
			'membership_status' => (string) ( $item->membership_status ?? 'active' ),
			'manager_user_id' => (int) $item->manager_user_id,
			'manager_name' => $manager instanceof WP_User ? (string) $manager->display_name : 'تخصیص‌داده‌نشده',
			'expert_user_id' => (int) $item->expert_user_id,
			'expert_name' => $expert instanceof WP_User ? (string) $expert->display_name : 'تخصیص‌داده‌نشده',
			'operations_sales_expert_user_id' => (int) $operations_chat['user_id'],
			'operations_sales_expert_name' => $operations_expert instanceof WP_User ? (string) $operations_expert->display_name : 'تخصیص‌داده‌نشده',
			'workflow_status' => (string) $item->workflow_status,
			'workflow_status_label' => $this->status_label( (string) $item->workflow_status ),
			'cancel_reason' => (string) ( $item->cancel_reason ?? '' ),
			'last_contact_at' => (string) ( $item->last_contact_at ?? '' ),
			'contact_attempts' => (int) ( $item->contact_attempts ?? 0 ),
			'purchase_verified' => $summary['invoice_count'] > 0,
			'purchase_invoice_count' => $summary['invoice_count'],
			'purchase_total' => $summary['total'],
			'purchase_total_fmt' => class_exists( 'SN_Helpers' ) ? SN_Helpers::format_price( $summary['total'] ) : number_format_i18n( $summary['total'] ),
			'activated_at' => (string) $item->activated_at,
			'can_manage' => $this->can_manage_item( $item, $viewer_id ),
		];
		$out['latest_action'] = $this->latest_action( (int) $item->id );
		$out['action_catalog'] = $out['can_manage'] ? $this->action_catalog( $item ) : [];
		$out['upgrade_options'] = $out['can_manage'] ? $this->item_upgrade_options( $item ) : [];
		$out['upgrade_rules_locked'] = $this->item_has_upgrade_snapshot( $item ) || ! empty( $out['upgrade_options'] );
		$api_config = class_exists( 'SN_Biavin_API' ) ? SN_Biavin_API::instance()->config_for_item( $item ) : null;
		$out['api_enabled'] = is_array( $api_config );
		$out['api_project_code'] = is_array( $api_config ) ? (string) $api_config['project_code'] : '';
		if ( $with_invoices ) { $out['invoices'] = $this->linked_invoices( (int) $item->id ); }
		return $out;
	}

	private function dashboard_scope( int $user_id, string $role ): array {
		$where = [ "m.status='active'" ]; $args = [];
		if ( $role === 'project_manager' ) { $where[] = 'mi.manager_user_id=%d'; $args[] = $user_id; }
		elseif ( $role === 'project_expert' ) { $where[] = 'mi.expert_user_id=%d'; $args[] = $user_id; }
		elseif ( in_array( $role, [ 'seller', 'converter' ], true ) ) { $where[] = 'm.origin_user_id=%d'; $args[] = $user_id; }
		elseif ( $role === 'sales_supervisor' ) {
			$ids = $this->sales_chat_scope_user_ids( $user_id );
			if ( ! $ids ) { return [ '0=1', [] ]; }
			$placeholders = implode( ',', array_fill( 0, count( $ids ), '%d' ) );
			$where[] = "(m.sales_chat_user_id IN ({$placeholders}) OR m.origin_user_id IN ({$placeholders}))";
			$args = array_merge( $args, $ids, $ids );
		}
		elseif ( $role !== 'admin' ) { return [ '0=1', [] ]; }
		return [ implode( ' AND ', $where ), $args ];
	}

	private function dashboard_items( int $user_id, string $role, int $page = 1, int $per_page = 100, string $query = '', string $status = '' ): array {
		global $wpdb; $t = $this->tables();
		[ $where, $args ] = $this->dashboard_scope( $user_id, $role );
		$query = sanitize_text_field( $query );
		if ( $query !== '' ) {
			$like = '%' . $wpdb->esc_like( $query ) . '%';
			$where .= ' AND (m.customer_name LIKE %s OR m.customer_phone LIKE %s OR m.subscription_name_snapshot LIKE %s OR mi.content_name_snapshot LIKE %s OR origin.display_name LIKE %s OR expert.display_name LIKE %s)';
			array_push( $args, $like, $like, $like, $like, $like, $like );
		}
		if ( in_array( $status, [ 'waiting', 'no_answer', 'follow_up', 'cancelled', 'normal', 'upsell', 'purchased', 'regretted' ], true ) ) { $where .= ' AND mi.workflow_status=%s'; $args[] = $status; }
		$page = max( 1, $page ); $per_page = max( 20, min( 150, $per_page ) ); $offset = ( $page - 1 ) * $per_page;
		$joins = " INNER JOIN {$t['memberships']} m ON m.id=mi.membership_id LEFT JOIN {$wpdb->users} origin ON origin.ID=m.origin_user_id LEFT JOIN {$wpdb->users} expert ON expert.ID=mi.expert_user_id";
		$count_sql = "SELECT COUNT(*) FROM {$t['items']} mi {$joins} WHERE {$where}";
		$total = (int) $wpdb->get_var( $args ? $wpdb->prepare( $count_sql, ...$args ) : $count_sql );
		$sql = "SELECT mi.*,m.source_type,m.source_invoice_id,m.dot_case_id,m.subscription_product_id,m.subscription_name_snapshot,m.customer_wp_id,m.customer_name,m.customer_phone,m.origin_user_id,m.origin_role,m.sales_chat_user_id,m.sales_chat_role,m.status membership_status,m.activated_at FROM {$t['items']} mi {$joins} WHERE {$where} ORDER BY m.activated_at DESC,mi.id DESC LIMIT %d OFFSET %d";
		$query_args = array_merge( $args, [ $per_page, $offset ] );
		$rows = $wpdb->get_results( $wpdb->prepare( $sql, ...$query_args ) );
		$purchase_map = $this->purchase_summaries( array_map( static fn( $row ) => (int) $row->id, $rows ?: [] ) );
		$out = [];
		foreach ( $rows ?: [] as $row ) { $out[] = $this->serialize_item( $row, $user_id, false, $purchase_map[ (int) $row->id ] ?? null ); }
		return [ 'items' => $out, 'total' => $total, 'page' => $page, 'per_page' => $per_page, 'has_more' => $offset + count( $out ) < $total ];
	}

	private function dashboard_stats( int $user_id, string $role ): array {
		global $wpdb; $t = $this->tables();
		[ $where, $args ] = $this->dashboard_scope( $user_id, $role );
		$sql = "SELECT COUNT(*) total,SUM(mi.workflow_status='waiting') waiting,SUM(mi.workflow_status='no_answer') no_answer,SUM(mi.workflow_status='follow_up') follow_up,SUM(mi.workflow_status='cancelled') cancelled,SUM(mi.workflow_status='normal') normal,SUM(mi.workflow_status='upsell') upsell,SUM(mi.workflow_status='purchased') purchased,SUM(mi.workflow_status='regretted') regretted FROM {$t['items']} mi INNER JOIN {$t['memberships']} m ON m.id=mi.membership_id WHERE {$where}";
		$row = $wpdb->get_row( $args ? $wpdb->prepare( $sql, ...$args ) : $sql, ARRAY_A ) ?: [];
		$sum_sql = "SELECT COALESCE(SUM(l.line_total),0) FROM {$t['invoice_links']} l INNER JOIN {$t['items']} mi ON mi.id=l.membership_item_id INNER JOIN {$t['memberships']} m ON m.id=mi.membership_id WHERE l.status='verified' AND {$where}";
		$verified_total = (float) $wpdb->get_var( $args ? $wpdb->prepare( $sum_sql, ...$args ) : $sum_sql );
		return [
			'total' => (int) ( $row['total'] ?? 0 ), 'waiting' => (int) ( $row['waiting'] ?? 0 ),
			'no_answer' => (int) ( $row['no_answer'] ?? 0 ), 'purchased' => (int) ( $row['purchased'] ?? 0 ),
			'follow_up' => (int) ( $row['follow_up'] ?? 0 ), 'cancelled' => (int) ( $row['cancelled'] ?? 0 ),
			'normal' => (int) ( $row['normal'] ?? 0 ), 'upsell' => (int) ( $row['upsell'] ?? 0 ),
			'regretted' => (int) ( $row['regretted'] ?? 0 ), 'verified_total' => $verified_total,
		];
	}

	public function ajax_dashboard(): void {
		$this->verify_ajax(); $user_id = get_current_user_id(); $role = $this->current_role();
		if ( ! in_array( $role, [ 'admin', 'project_manager', 'project_expert', 'sales_supervisor' ], true ) ) { wp_send_json( [ 'success' => false, 'message' => 'دسترسی به پنل پروژه ندارید.' ], 403 ); }
		$page = max( 1, absint( $_POST['page'] ?? 1 ) );
		$query = sanitize_text_field( wp_unslash( $_POST['query'] ?? '' ) );
		$status = sanitize_key( wp_unslash( $_POST['status'] ?? '' ) );
		$result = $this->dashboard_items( $user_id, $role, $page, 100, $query, $status );
		$experts = [];
		if ( in_array( $role, [ 'admin', 'project_manager' ], true ) ) {
			foreach ( $this->direct_experts_for_manager( $user_id ) as $expert ) { $experts[] = [ 'id' => (int) $expert->ID, 'name' => (string) $expert->display_name ]; }
		}
		wp_send_json( [
			'success' => true, 'items' => $result['items'], 'stats' => $this->dashboard_stats( $user_id, $role ),
			'experts' => $experts, 'role' => $role, 'pagination' => [
				'page' => $result['page'], 'per_page' => $result['per_page'], 'total' => $result['total'], 'has_more' => $result['has_more'],
			],
		] );
	}

	public function ajax_assign_expert(): void {
		$this->verify_ajax(); $item_id = absint( $_POST['item_id'] ?? 0 ); $expert_id = absint( $_POST['expert_id'] ?? 0 ); $user_id = get_current_user_id();
		$item = $this->item_context( $item_id );
		if ( ! $item || (string) ( $item->membership_status ?? '' ) !== 'active' || ( ! current_user_can( 'manage_options' ) && ( $this->position_for_user( $user_id ) !== 'project_manager' || (int) $item->manager_user_id !== $user_id ) ) ) { wp_send_json( [ 'success' => false, 'message' => 'اجازه تخصیص این پرونده را ندارید.' ], 403 ); }
		$allowed = array_map( static fn( $u ) => (int) $u->ID, $this->direct_experts_for_manager( $user_id ) );
		if ( $expert_id > 0 && ! current_user_can( 'manage_options' ) && ! in_array( $expert_id, $allowed, true ) ) { wp_send_json( [ 'success' => false, 'message' => 'کارشناس باید در HR زیرمجموعه مستقیم شما باشد.' ], 403 ); }
		if ( $expert_id > 0 && $this->position_for_user( $expert_id ) !== 'project_expert' ) { wp_send_json( [ 'success' => false, 'message' => 'کاربر انتخاب‌شده کارشناس پروژه نیست.' ], 400 ); }
		global $wpdb; $t = $this->tables(); $updated = $wpdb->update( $t['items'], [ 'expert_user_id' => $expert_id ?: null, 'assigned_at' => $expert_id ? current_time( 'mysql' ) : null, 'updated_at' => current_time( 'mysql' ) ], [ 'id' => $item_id ] );
		if ( false === $updated ) { wp_send_json( [ 'success' => false, 'message' => 'تخصیص ذخیره نشد.' ], 500 ); }
		$this->event( (int) $item->membership_id, $item_id, 'expert_assigned', [ 'expert_user_id' => $expert_id ], $user_id );
		wp_send_json( [ 'success' => true, 'message' => $expert_id ? 'کارشناس پروژه تخصیص داده شد.' : 'تخصیص کارشناس برداشته شد.' ] );
	}

	public function ajax_bulk_assign_expert(): void {
		$this->verify_ajax();
		$user_id = get_current_user_id();
		$role = $this->current_role();
		if ( ! in_array( $role, [ 'admin', 'project_manager' ], true ) ) {
			wp_send_json( [ 'success' => false, 'message' => 'تخصیص گروهی فقط برای مدیر پروژه در دسترس است.' ], 403 );
		}
		$raw_ids = isset( $_POST['item_ids'] ) ? (array) wp_unslash( $_POST['item_ids'] ) : [];
		$item_ids = array_slice( array_values( array_unique( array_filter( array_map( 'absint', $raw_ids ) ) ) ), 0, 200 );
		$expert_id = absint( $_POST['expert_id'] ?? 0 );
		if ( ! $item_ids ) { wp_send_json( [ 'success' => false, 'message' => 'حداقل یک مشتری را انتخاب کنید.' ], 400 ); }
		if ( $expert_id > 0 && $this->position_for_user( $expert_id ) !== 'project_expert' ) {
			wp_send_json( [ 'success' => false, 'message' => 'کاربر انتخاب‌شده کارشناس پروژه نیست.' ], 400 );
		}
		$allowed_experts = array_map( static fn( $user ) => (int) $user->ID, $this->direct_experts_for_manager( $user_id ) );
		if ( $expert_id > 0 && $role !== 'admin' && ! in_array( $expert_id, $allowed_experts, true ) ) {
			wp_send_json( [ 'success' => false, 'message' => 'کارشناس باید در HR زیرمجموعه مستقیم شما باشد.' ], 403 );
		}

		global $wpdb; $t = $this->tables(); $now = current_time( 'mysql' );
		$ids_sql = implode( ',', array_map( 'intval', $item_ids ) );
		if ( false === $wpdb->query( 'START TRANSACTION' ) ) { wp_send_json( [ 'success' => false, 'message' => 'شروع عملیات گروهی ممکن نشد.' ], 500 ); }
		$rows = $wpdb->get_results( "SELECT mi.id,mi.membership_id,mi.manager_user_id,mi.expert_user_id,m.status membership_status FROM {$t['items']} mi INNER JOIN {$t['memberships']} m ON m.id=mi.membership_id WHERE mi.id IN ({$ids_sql}) FOR UPDATE" );
		if ( $wpdb->last_error !== '' ) {
			$wpdb->query( 'ROLLBACK' );
			wp_send_json( [ 'success' => false, 'message' => 'خواندن امن پرونده‌های انتخاب‌شده انجام نشد.' ], 500 );
		}
		$by_id = [];
		foreach ( $rows ?: [] as $row ) { $by_id[ (int) $row->id ] = $row; }
		$assigned = 0; $unchanged = 0; $skipped = 0;
		foreach ( $item_ids as $item_id ) {
			$item = $by_id[ $item_id ] ?? null;
			if ( ! $item || (string) $item->membership_status !== 'active' || ( $role !== 'admin' && (int) $item->manager_user_id !== $user_id ) ) { $skipped++; continue; }
			if ( (int) $item->expert_user_id === $expert_id ) { $unchanged++; continue; }
			$updated = $wpdb->update( $t['items'], [
				'expert_user_id' => $expert_id ?: null,
				'assigned_at' => $expert_id ? $now : null,
				'updated_at' => $now,
			], [ 'id' => $item_id ] );
			if ( false === $updated ) {
				$wpdb->query( 'ROLLBACK' );
				wp_send_json( [ 'success' => false, 'message' => 'تخصیص گروهی ذخیره نشد؛ هیچ تغییری اعمال نشد.' ], 500 );
			}
			$this->event( (int) $item->membership_id, $item_id, 'expert_bulk_assigned', [ 'old_expert_user_id' => (int) $item->expert_user_id, 'expert_user_id' => $expert_id ], $user_id );
			$assigned++;
		}
		if ( false === $wpdb->query( 'COMMIT' ) ) {
			$wpdb->query( 'ROLLBACK' );
			wp_send_json( [ 'success' => false, 'message' => 'ثبت نهایی تخصیص گروهی انجام نشد.' ], 500 );
		}
		$message = $expert_id > 0 ? sprintf( '%s مورد به کارشناس انتخاب‌شده تخصیص یافت.', number_format_i18n( $assigned ) ) : sprintf( 'تخصیص کارشناس از %s مورد برداشته شد.', number_format_i18n( $assigned ) );
		if ( $unchanged > 0 ) { $message .= ' ' . sprintf( '%s مورد از قبل همین وضعیت را داشت.', number_format_i18n( $unchanged ) ); }
		if ( $skipped > 0 ) { $message .= ' ' . sprintf( '%s مورد به‌دلیل تغییر دسترسی یا وضعیت رد شد.', number_format_i18n( $skipped ) ); }
		wp_send_json( [ 'success' => true, 'message' => $message, 'assigned_count' => $assigned, 'unchanged_count' => $unchanged, 'skipped_count' => $skipped ] );
	}

	public function ajax_update_status(): void {
		$this->verify_ajax(); $item_id = absint( $_POST['item_id'] ?? 0 ); $status = sanitize_key( wp_unslash( $_POST['status'] ?? '' ) ); $user_id = get_current_user_id();
		if ( ! in_array( $status, [ 'waiting', 'no_answer', 'follow_up', 'cancelled', 'normal', 'upsell', 'purchased', 'regretted' ], true ) ) { wp_send_json( [ 'success' => false, 'message' => 'وضعیت نامعتبر است.' ], 400 ); }
		$item = $this->item_context( $item_id );
		if ( ! $item || ! $this->can_manage_item( $item, $user_id ) ) { wp_send_json( [ 'success' => false, 'message' => 'اجازه تغییر این پرونده را ندارید.' ], 403 ); }
		$cancel_reason = sanitize_textarea_field( wp_unslash( $_POST['cancel_reason'] ?? '' ) );
		if ( $status === 'cancelled' && $cancel_reason === '' && (string) ( $item->cancel_reason ?? '' ) === '' ) {
			wp_send_json( [ 'success' => false, 'message' => 'برای وضعیت انصراف، ثبت علت اجباری است.' ], 400 );
		}
		global $wpdb; $t = $this->tables();
		$now = current_time( 'mysql' );
		$contact_statuses = [ 'no_answer', 'follow_up', 'cancelled', 'normal', 'upsell' ];
		$update = [
			'workflow_status' => $status,
			'cancel_reason' => $status === 'cancelled' ? ( $cancel_reason !== '' ? $cancel_reason : (string) $item->cancel_reason ) : null,
			'last_contact_at' => in_array( $status, $contact_statuses, true ) ? $now : (string) ( $item->last_contact_at ?? '' ),
			'contact_attempts' => (int) ( $item->contact_attempts ?? 0 ) + ( in_array( $status, $contact_statuses, true ) ? 1 : 0 ),
			'status_updated_at' => $now,
			'updated_at' => $now,
		];
		if ( false === $wpdb->update( $t['items'], $update, [ 'id' => $item_id ] ) ) { wp_send_json( [ 'success' => false, 'message' => 'وضعیت ذخیره نشد.' ], 500 ); }
		$this->event( (int) $item->membership_id, $item_id, 'workflow_status_changed', [ 'old' => (string) $item->workflow_status, 'new' => $status, 'cancel_reason' => $update['cancel_reason'], 'contact_attempts' => $update['contact_attempts'] ], $user_id );
		wp_send_json( [ 'success' => true, 'message' => 'وضعیت ذخیره شد.', 'status' => $status, 'status_label' => $this->status_label( $status ), 'cancel_reason' => (string) $update['cancel_reason'], 'contact_attempts' => (int) $update['contact_attempts'] ] );
	}

	private function latest_action( int $item_id ): ?array {
		global $wpdb; $table = $this->tables()['actions'];
		if ( $item_id < 1 || ! $this->table_exists( $table ) ) { return null; }
		$row = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$table} WHERE membership_item_id=%d ORDER BY id DESC LIMIT 1", $item_id ), ARRAY_A );
		if ( ! $row ) { return null; }
		foreach ( [ 'id', 'membership_item_id', 'actor_user_id', 'target_product_id', 'invoice_id', 'remote_order_id', 'payment_attempts' ] as $key ) { $row[ $key ] = (int) ( $row[ $key ] ?? 0 ); }
		foreach ( [ 'base_credit', 'target_credit', 'amount' ] as $key ) { $row[ $key ] = (float) ( $row[ $key ] ?? 0 ); }
		return $row;
	}

	private function action_catalog( object $item ): array {
		$rows = apply_filters( 'sn_biavin_action_catalog', [], $item );
		if ( ! is_array( $rows ) || ! $rows ) {
			global $wpdb; $manager_id = (int) ( $item->manager_user_id ?? 0 ); $manager_table = $this->tables()['managers'];
			if ( $manager_id > 0 && $this->table_exists( $manager_table ) ) {
				$rows = $wpdb->get_results( $wpdb->prepare( "SELECT content_product_id id FROM {$manager_table} WHERE manager_user_id=%d ORDER BY content_product_id", $manager_id ), ARRAY_A ) ?: [];
			}
		}
		$out = [];
		foreach ( is_array( $rows ) ? $rows : [] as $row ) {
			if ( ! is_array( $row ) ) { continue; }
			$id = absint( $row['id'] ?? $row['product_id'] ?? 0 );
			$credit = max( 0, (float) ( $row['credit'] ?? $row['target_credit'] ?? 0 ) );
			if ( $credit <= 0 && $id > 0 ) { $credit = $this->product_credit( $id ); }
			$name = sanitize_text_field( (string) ( $row['name'] ?? $row['title'] ?? ( $id ? get_the_title( $id ) : '' ) ) );
			if ( $name === '' || $credit <= 0 ) { continue; }
			$out[] = [ 'id' => $id, 'name' => $name, 'credit' => $credit, 'price' => max( 0, (float) ( $row['price'] ?? 0 ) ) ];
		}
		return $out;
	}

	public function ajax_create_action(): void {
		$this->verify_ajax();
		$item_id = absint( $_POST['item_id'] ?? 0 ); $user_id = get_current_user_id();
		$item = $this->item_context( $item_id );
		if ( ! $item || ! $this->can_manage_item( $item, $user_id ) ) { wp_send_json( [ 'success' => false, 'message' => 'اجازه ثبت اقدام بیاوین را ندارید.' ], 403 ); }
		$action_type = sanitize_key( wp_unslash( $_POST['action_type'] ?? '' ) );
		$mode = sanitize_key( wp_unslash( $_POST['fulfillment_mode'] ?? '' ) );
		if ( ! in_array( $action_type, [ 'normal', 'upsell' ], true ) || ! in_array( $mode, [ 'product', 'wallet' ], true ) ) { wp_send_json( [ 'success' => false, 'message' => 'نوع اقدام یا روش تحویل نامعتبر است.' ], 400 ); }
		$target_id = absint( $_POST['target_product_id'] ?? 0 );
		$target_name = sanitize_text_field( wp_unslash( $_POST['target_product_name'] ?? '' ) );
		$base_credit = $this->item_base_credit( $item );
		$target_credit = max( 0, (float) ( $_POST['target_credit'] ?? 0 ) );
		$amount_raw = class_exists( 'SN_Helpers' ) ? SN_Helpers::to_english_nums( (string) ( $_POST['amount'] ?? 0 ) ) : (string) ( $_POST['amount'] ?? 0 );
		$amount = max( 0, (float) str_replace( [ ',', '٬', '،', ' ' ], '', $amount_raw ) );
		$catalog = $this->action_catalog( $item );
		$free_upgrade = false;
		$upgrade_rules_locked = false;
		foreach ( $catalog as $row ) {
			if ( $target_id > 0 && (int) $row['id'] === $target_id ) { $target_name = $row['name']; $target_credit = $row['credit']; break; }
		}
		if ( $action_type === 'upsell' ) {
			$upgrade_options = $this->item_upgrade_options( $item );
			$upgrade_rules_locked = $this->item_has_upgrade_snapshot( $item ) || ! empty( $upgrade_options );
			if ( $upgrade_rules_locked ) {
				$matched_upgrade = null;
				foreach ( $upgrade_options as $option ) {
					if ( $target_id > 0 && (int) ( $option['id'] ?? $option['product_id'] ?? 0 ) === $target_id ) { $matched_upgrade = $option; break; }
				}
				if ( ! is_array( $matched_upgrade ) ) { wp_send_json( [ 'success' => false, 'message' => 'این مقصد در گزینه‌های افزایشی ثبت‌شده کارت وجود ندارد.' ], 400 ); }
				$target_name = sanitize_text_field( (string) ( $matched_upgrade['name'] ?? get_the_title( $target_id ) ) );
				$target_credit = max( 0, (float) ( $matched_upgrade['credit'] ?? $matched_upgrade['target_credit'] ?? 0 ) );
				$amount = max( 0, (float) ( $matched_upgrade['price'] ?? $matched_upgrade['upgrade_amount'] ?? 0 ) );
				$free_upgrade = $amount === 0.0 && ( array_key_exists( 'price', $matched_upgrade ) || array_key_exists( 'upgrade_amount', $matched_upgrade ) );
			}
		}
		if ( $target_name === '' && $target_id > 0 ) { $target_name = sanitize_text_field( (string) get_the_title( $target_id ) ); }
		if ( $mode === 'product' && $target_name === '' ) { wp_send_json( [ 'success' => false, 'message' => 'برای انتخاب محصول، نام یا شناسه محصول مقصد را وارد کنید.' ], 400 ); }
		if ( $target_credit <= 0 && ! $upgrade_rules_locked ) { wp_send_json( [ 'success' => false, 'message' => 'اعتبار محصول مقصد باید بیشتر از صفر باشد.' ], 400 ); }
		if ( $action_type === 'upsell' && $amount <= 0 && ! $free_upgrade ) { wp_send_json( [ 'success' => false, 'message' => 'مبلغ افزایشی را وارد کنید.' ], 400 ); }
		$existing_action = $this->latest_action( $item_id );
		if ( $existing_action && in_array( (string) ( $existing_action['status'] ?? '' ), [ 'ready', 'awaiting_payment', 'retry_requested', 'paid' ], true ) ) { wp_send_json( [ 'success' => false, 'message' => 'برای این محتوا یک اقدام باز وجود دارد؛ ابتدا آن را تکمیل یا لغو کنید.' ], 409 ); }
		global $wpdb; $table = $this->tables()['actions']; $now = current_time( 'mysql' );
		$initial_status = $action_type === 'normal' || $free_upgrade ? 'ready' : 'awaiting_payment';
		$ok = $wpdb->insert( $table, [ 'membership_item_id' => $item_id, 'actor_user_id' => $user_id, 'action_type' => $action_type, 'fulfillment_mode' => $mode, 'target_product_id' => $target_id ?: null, 'target_product_name' => $target_name ?: null, 'base_credit' => $base_credit, 'target_credit' => $target_credit, 'amount' => $amount, 'status' => $initial_status, 'payment_attempts' => 0, 'note' => sanitize_textarea_field( wp_unslash( $_POST['note'] ?? '' ) ), 'created_at' => $now, 'updated_at' => $now ] );
		if ( ! $ok ) { wp_send_json( [ 'success' => false, 'message' => 'ثبت اقدام انجام نشد.' ], 500 ); }
		$action_id = (int) $wpdb->insert_id;
		$this->event( (int) $item->membership_id, $item_id, 'project_action_created', [ 'action_id' => $action_id, 'action_type' => $action_type, 'fulfillment_mode' => $mode, 'target_product_id' => $target_id, 'target_credit' => $target_credit, 'amount' => $amount ], $user_id );
		do_action( 'sn_project_action_created', $action_id, $item, [ 'action_type' => $action_type, 'fulfillment_mode' => $mode, 'target_product_id' => $target_id, 'target_credit' => $target_credit, 'amount' => $amount ] );
		wp_send_json( [ 'success' => true, 'message' => $free_upgrade ? 'اقدام افزایشی رایگان بدون پرداخت آماده تکمیل شد.' : ( $action_type === 'normal' ? 'اقدام عادی آماده تکمیل و ارسال به سامانه مقصد شد.' : 'اقدام افزایشی ثبت شد و در انتظار پرداخت مشتری است.' ), 'action' => $this->latest_action( $item_id ) ] );
	}

	private function api_customer_for_item( object $item ): array {
		$user = ! empty( $item->customer_wp_id ) ? get_user_by( 'id', (int) $item->customer_wp_id ) : null;
		return [
			'name' => (string) ( $item->customer_name ?? '' ),
			'mobile' => (string) ( $item->customer_phone ?? '' ),
			'email' => $user instanceof WP_User ? (string) $user->user_email : '',
		];
	}

	public function ajax_api_products(): void {
		$this->verify_ajax();
		$item_id = absint( $_POST['item_id'] ?? 0 ); $user_id = get_current_user_id();
		$item = $this->item_context( $item_id );
		if ( ! $item || ! $this->can_manage_item( $item, $user_id ) ) { wp_send_json( [ 'success' => false, 'message' => 'اجازه دریافت محصولات این پروژه را ندارید.' ], 403 ); }
		if ( ! class_exists( 'SN_Biavin_API' ) ) { wp_send_json( [ 'success' => false, 'message' => 'ماژول اتصال API در دسترس نیست.' ], 503 ); }
		$products = SN_Biavin_API::instance()->products( $item, sanitize_text_field( wp_unslash( $_POST['search'] ?? '' ) ) );
		if ( is_wp_error( $products ) ) { wp_send_json( [ 'success' => false, 'message' => $products->get_error_message() ], 502 ); }
		wp_send_json( [ 'success' => true, 'products' => $products ] );
	}

	public function ajax_api_purchase(): void {
		$this->verify_ajax();
		$item_id = absint( $_POST['item_id'] ?? 0 ); $user_id = get_current_user_id();
		$item = $this->item_context( $item_id );
		if ( ! $item || ! $this->can_manage_item( $item, $user_id ) ) { wp_send_json( [ 'success' => false, 'message' => 'اجازه خرید مستقیم برای این پرونده را ندارید.' ], 403 ); }
		if ( ! in_array( (string) $item->workflow_status, [ 'normal', 'upsell' ], true ) ) { wp_send_json( [ 'success' => false, 'message' => 'ابتدا وضعیت پرونده را عادی یا افزایشی انتخاب کنید.' ], 400 ); }
		$product_id = absint( $_POST['product_id'] ?? 0 ); $quantity = max( 1, min( 20, absint( $_POST['quantity'] ?? 1 ) ) );
		if ( $product_id < 1 ) { wp_send_json( [ 'success' => false, 'message' => 'محصول سایت مقصد را انتخاب کنید.' ], 400 ); }
		if ( ! class_exists( 'SN_Biavin_API' ) || ! SN_Biavin_API::instance()->config_for_item( $item ) ) { wp_send_json( [ 'success' => false, 'message' => 'API این پروژه فعال نیست.' ], 503 ); }
		global $wpdb; $table = $this->tables()['actions']; $now = current_time( 'mysql' );
		$ok = $wpdb->insert( $table, [ 'membership_item_id' => $item_id, 'actor_user_id' => $user_id, 'action_type' => (string) $item->workflow_status, 'fulfillment_mode' => 'product', 'target_product_id' => $product_id, 'target_product_name' => sanitize_text_field( wp_unslash( $_POST['product_name'] ?? '' ) ), 'base_credit' => $this->item_base_credit( $item ), 'target_credit' => 0, 'amount' => 0, 'status' => 'remote_pending', 'payment_attempts' => 0, 'note' => sanitize_textarea_field( wp_unslash( $_POST['note'] ?? '' ) ), 'created_at' => $now, 'updated_at' => $now ] );
		if ( ! $ok ) { wp_send_json( [ 'success' => false, 'message' => 'ثبت درخواست خرید انجام نشد.' ], 500 ); }
		$action_id = (int) $wpdb->insert_id;
		$reference = 'sn-' . (int) get_current_blog_id() . '-project-action-' . $action_id . '-order';
		$result = SN_Biavin_API::instance()->create_order( $item, $this->api_customer_for_item( $item ), $product_id, $quantity, $reference );
		if ( is_wp_error( $result ) ) {
			$wpdb->update( $table, [ 'status' => 'remote_failed', 'remote_status' => 'failed', 'note' => $result->get_error_message(), 'updated_at' => current_time( 'mysql' ) ], [ 'id' => $action_id ] );
			$this->event( (int) $item->membership_id, $item_id, 'project_remote_order_failed', [ 'action_id' => $action_id, 'error' => $result->get_error_message() ], $user_id );
			wp_send_json( [ 'success' => false, 'message' => $result->get_error_message(), 'action_id' => $action_id ], 502 );
		}
		$order = isset( $result['order'] ) && is_array( $result['order'] ) ? $result['order'] : $result;
		$remote_status = sanitize_key( (string) ( $order['status'] ?? 'pending' ) );
		$local_status = in_array( $remote_status, [ 'processing', 'completed', 'paid' ], true ) ? 'fulfilled' : 'awaiting_payment';
		$wpdb->update( $table, [ 'status' => $local_status, 'remote_order_id' => absint( $order['id'] ?? 0 ) ?: null, 'remote_order_key' => sanitize_text_field( (string) ( $order['key'] ?? '' ) ), 'remote_payment_url' => esc_url_raw( (string) ( $order['payment_url'] ?? '' ) ), 'remote_status' => $remote_status, 'target_product_name' => sanitize_text_field( (string) ( $order['product_name'] ?? $_POST['product_name'] ?? '' ) ), 'amount' => max( 0, (float) ( $order['total'] ?? 0 ) ), 'updated_at' => current_time( 'mysql' ) ], [ 'id' => $action_id ] );
		$this->event( (int) $item->membership_id, $item_id, 'project_remote_order_created', [ 'action_id' => $action_id, 'remote_order_id' => absint( $order['id'] ?? 0 ), 'remote_status' => $remote_status ], $user_id );
		wp_send_json( [ 'success' => true, 'message' => 'سفارش مشتری در سایت مقصد ثبت شد.', 'action' => $this->latest_action( $item_id ), 'order' => $order, 'payment_url' => esc_url_raw( (string) ( $order['payment_url'] ?? '' ) ) ] );
	}

	public function ajax_api_charge_wallet(): void {
		$this->verify_ajax();
		$item_id = absint( $_POST['item_id'] ?? 0 ); $user_id = get_current_user_id();
		$item = $this->item_context( $item_id );
		if ( ! $item || ! $this->can_manage_item( $item, $user_id ) ) { wp_send_json( [ 'success' => false, 'message' => 'اجازه شارژ کیف پول این مشتری را ندارید.' ], 403 ); }
		if ( ! in_array( (string) $item->workflow_status, [ 'normal', 'upsell' ], true ) ) { wp_send_json( [ 'success' => false, 'message' => 'ابتدا وضعیت پرونده را عادی یا افزایشی انتخاب کنید.' ], 400 ); }
		$amount_raw = class_exists( 'SN_Helpers' ) ? SN_Helpers::to_english_nums( (string) ( $_POST['amount'] ?? 0 ) ) : (string) ( $_POST['amount'] ?? 0 );
		$amount = max( 0, (float) str_replace( [ ',', '٬', '،', ' ' ], '', $amount_raw ) );
		$wallet_type = sanitize_key( wp_unslash( $_POST['wallet_type'] ?? 'cash' ) );
		if ( $amount <= 0 || ! in_array( $wallet_type, [ 'cash', 'installment' ], true ) ) { wp_send_json( [ 'success' => false, 'message' => 'نوع کیف پول و مبلغ شارژ را درست وارد کنید.' ], 400 ); }
		if ( ! class_exists( 'SN_Biavin_API' ) || ! SN_Biavin_API::instance()->config_for_item( $item ) ) { wp_send_json( [ 'success' => false, 'message' => 'API این پروژه فعال نیست.' ], 503 ); }
		global $wpdb; $table = $this->tables()['actions']; $now = current_time( 'mysql' );
		$ok = $wpdb->insert( $table, [ 'membership_item_id' => $item_id, 'actor_user_id' => $user_id, 'action_type' => (string) $item->workflow_status, 'fulfillment_mode' => 'wallet', 'target_product_id' => null, 'target_product_name' => $wallet_type === 'cash' ? 'شارژ کیف پول نقدی' : 'شارژ کیف پول اقساطی', 'base_credit' => $this->item_base_credit( $item ), 'target_credit' => $amount, 'amount' => 0, 'status' => 'remote_pending', 'payment_attempts' => 0, 'note' => sanitize_textarea_field( wp_unslash( $_POST['note'] ?? '' ) ), 'created_at' => $now, 'updated_at' => $now ] );
		if ( ! $ok ) { wp_send_json( [ 'success' => false, 'message' => 'ثبت درخواست شارژ انجام نشد.' ], 500 ); }
		$action_id = (int) $wpdb->insert_id;
		$reference = 'sn-' . (int) get_current_blog_id() . '-project-action-' . $action_id . '-wallet-' . $wallet_type;
		$result = SN_Biavin_API::instance()->charge_wallet( $item, $this->api_customer_for_item( $item ), $wallet_type, $amount, $reference, sanitize_text_field( wp_unslash( $_POST['note'] ?? '' ) ) );
		if ( is_wp_error( $result ) ) {
			$wpdb->update( $table, [ 'status' => 'remote_failed', 'remote_status' => 'failed', 'note' => $result->get_error_message(), 'updated_at' => current_time( 'mysql' ) ], [ 'id' => $action_id ] );
			$this->event( (int) $item->membership_id, $item_id, 'project_remote_wallet_failed', [ 'action_id' => $action_id, 'error' => $result->get_error_message() ], $user_id );
			wp_send_json( [ 'success' => false, 'message' => $result->get_error_message(), 'action_id' => $action_id ], 502 );
		}
		$wallet = isset( $result['wallet'] ) && is_array( $result['wallet'] ) ? $result['wallet'] : $result;
		$wpdb->update( $table, [ 'status' => 'fulfilled', 'remote_status' => 'charged', 'updated_at' => current_time( 'mysql' ) ], [ 'id' => $action_id ] );
		$this->event( (int) $item->membership_id, $item_id, 'project_remote_wallet_charged', [ 'action_id' => $action_id, 'wallet_type' => $wallet_type, 'amount' => $amount, 'remaining' => (float) ( $wallet['remaining'] ?? 0 ) ], $user_id );
		wp_send_json( [ 'success' => true, 'message' => 'کیف پول مشتری در سایت مقصد شارژ شد.', 'action' => $this->latest_action( $item_id ), 'wallet' => $wallet ] );
	}

	public function ajax_retry_action_payment(): void {
		$this->verify_ajax(); $action_id = absint( $_POST['action_id'] ?? 0 ); $user_id = get_current_user_id();
		global $wpdb; $table = $this->tables()['actions']; $action = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$table} WHERE id=%d LIMIT 1", $action_id ) );
		$item = $action ? $this->item_context( (int) $action->membership_item_id ) : null;
		if ( ! $action || ! $item || ! $this->can_manage_item( $item, $user_id ) ) { wp_send_json( [ 'success' => false, 'message' => 'اجازه تکرار پرداخت این اقدام را ندارید.' ], 403 ); }
		if ( ! in_array( (string) $action->status, [ 'unpaid', 'awaiting_payment', 'retry_requested' ], true ) ) { wp_send_json( [ 'success' => false, 'message' => 'این اقدام در وضعیت قابل تکرار نیست.' ], 400 ); }
		$now = current_time( 'mysql' ); $attempts = (int) $action->payment_attempts + 1;
		$wpdb->update( $table, [ 'status' => 'retry_requested', 'payment_attempts' => $attempts, 'updated_at' => $now ], [ 'id' => $action_id ] );
		$this->event( (int) $item->membership_id, (int) $item->id, 'project_action_payment_retry', [ 'action_id' => $action_id, 'payment_attempts' => $attempts ], $user_id );
		do_action( 'sn_project_action_payment_retry', $action_id, $item, $action );
		wp_send_json( [ 'success' => true, 'message' => 'درخواست فرصت مجدد ثبت شد؛ پیامک پرداخت توسط اتصال پرداخت ارسال می‌شود.', 'payment_attempts' => $attempts ] );
	}

	public function ajax_cancel_upsell(): void {
		$this->verify_ajax(); $action_id = absint( $_POST['action_id'] ?? 0 ); $user_id = get_current_user_id();
		global $wpdb; $table = $this->tables()['actions']; $action = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$table} WHERE id=%d LIMIT 1", $action_id ) );
		$item = $action ? $this->item_context( (int) $action->membership_item_id ) : null;
		if ( ! $action || (string) $action->action_type !== 'upsell' || ! $item || ! $this->can_manage_item( $item, $user_id ) ) { wp_send_json( [ 'success' => false, 'message' => 'اجازه لغو افزایشی این پرونده را ندارید.' ], 403 ); }
		$reason = sanitize_textarea_field( wp_unslash( $_POST['reason'] ?? '' ) );
		$now = current_time( 'mysql' );
		$wpdb->update( $table, [ 'status' => 'cancelled_continue_normal', 'note' => $reason ?: (string) $action->note, 'updated_at' => $now ], [ 'id' => $action_id ] );
		$wpdb->update( $this->tables()['items'], [ 'workflow_status' => 'normal', 'status_updated_at' => $now, 'updated_at' => $now ], [ 'id' => (int) $item->id ] );
		$this->event( (int) $item->membership_id, (int) $item->id, 'project_upsell_cancelled_continue_normal', [ 'action_id' => $action_id, 'reason' => $reason ], $user_id );
		wp_send_json( [ 'success' => true, 'message' => 'افزایشی لغو شد و پرونده با روند عادی ادامه می‌یابد.' ] );
	}

	public function ajax_fulfill_action(): void {
		$this->verify_ajax(); $action_id = absint( $_POST['action_id'] ?? 0 ); $user_id = get_current_user_id();
		global $wpdb; $table = $this->tables()['actions']; $action = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$table} WHERE id=%d LIMIT 1", $action_id ) );
		$item = $action ? $this->item_context( (int) $action->membership_item_id ) : null;
		if ( ! $action || ! $item || ! $this->can_manage_item( $item, $user_id ) ) { wp_send_json( [ 'success' => false, 'message' => 'اجازه تکمیل این اقدام را ندارید.' ], 403 ); }
		if ( ! in_array( (string) $action->status, [ 'ready', 'paid' ], true ) ) { wp_send_json( [ 'success' => false, 'message' => 'اقدام باید ابتدا آماده یا پرداخت‌شده باشد.' ], 400 ); }
		$now = current_time( 'mysql' ); $wpdb->update( $table, [ 'status' => 'fulfilled', 'updated_at' => $now ], [ 'id' => $action_id, 'status' => 'paid' ] );
		$this->event( (int) $item->membership_id, (int) $item->id, 'project_action_fulfilled', [ 'action_id' => $action_id, 'mode' => (string) $action->fulfillment_mode, 'target_product_id' => (int) $action->target_product_id ], $user_id );
		do_action( 'sn_project_action_fulfilled', $action_id, $item, $action );
		wp_send_json( [ 'success' => true, 'message' => 'اقدام تکمیل شد.' ] );
	}

	public function ajax_add_note(): void {
		$this->verify_ajax(); $item_id = absint( $_POST['item_id'] ?? 0 ); $note = sanitize_textarea_field( wp_unslash( $_POST['note'] ?? '' ) ); $user_id = get_current_user_id();
		if ( $note === '' ) { wp_send_json( [ 'success' => false, 'message' => 'متن یادداشت خالی است.' ], 400 ); }
		$item = $this->item_context( $item_id );
		if ( ! $item || ! $this->can_manage_item( $item, $user_id ) ) { wp_send_json( [ 'success' => false, 'message' => 'اجازه یادداشت‌گذاری این پرونده را ندارید.' ], 403 ); }
		global $wpdb; $t = $this->tables();
		if ( ! $wpdb->insert( $t['notes'], [ 'membership_item_id' => $item_id, 'author_user_id' => $user_id, 'note' => $note, 'created_at' => current_time( 'mysql' ) ] ) ) { wp_send_json( [ 'success' => false, 'message' => 'یادداشت ذخیره نشد.' ], 500 ); }
		$this->event( (int) $item->membership_id, $item_id, 'internal_note_added', [], $user_id );
		wp_send_json( [ 'success' => true, 'message' => 'یادداشت داخلی ذخیره شد.' ] );
	}

	private function private_upload_base(): string {
		$uploads = wp_upload_dir( null, false );
		return trailingslashit( (string) $uploads['basedir'] ) . 'sn-project-private';
	}

	private function ensure_private_upload_dir(): bool {
		$base = $this->private_upload_base();
		if ( ! wp_mkdir_p( $base ) ) { return false; }
		$guards = [
			$base . '/.htaccess' => "Deny from all\n",
			$base . '/web.config' => '<?xml version="1.0" encoding="UTF-8"?><configuration><system.webServer><security><authorization><remove users="*" roles="" verbs=""/><add accessType="Deny" users="*"/></authorization></security></system.webServer></configuration>',
			$base . '/index.php' => "<?php http_response_code(404); exit;\n",
		];
		foreach ( $guards as $path => $content ) {
			if ( ! file_exists( $path ) ) { @file_put_contents( $path, $content, LOCK_EX ); }
		}
		return is_dir( $base ) && is_writable( $base );
	}

	private function store_attachment( array $file ) {
		if ( empty( $file['tmp_name'] ) || ! is_uploaded_file( (string) $file['tmp_name'] ) ) { return new WP_Error( 'sn_project_file_upload', 'فایل بارگذاری‌شده معتبر نیست.' ); }
		$size = absint( $file['size'] ?? 0 );
		if ( $size < 1 || $size > self::MAX_FILE_SIZE ) { return new WP_Error( 'sn_project_file_size', 'حجم فایل باید کمتر از ۸ مگابایت باشد.' ); }
		$original = sanitize_file_name( (string) ( $file['name'] ?? 'file' ) );
		$extension = strtolower( pathinfo( $original, PATHINFO_EXTENSION ) );
		$allowed = [
			'jpg' => [ 'image/jpeg' ], 'jpeg' => [ 'image/jpeg' ], 'png' => [ 'image/png' ], 'webp' => [ 'image/webp' ],
			'pdf' => [ 'application/pdf' ], 'txt' => [ 'text/plain' ],
			'doc' => [ 'application/msword', 'application/octet-stream' ],
			'docx' => [ 'application/vnd.openxmlformats-officedocument.wordprocessingml.document', 'application/zip', 'application/x-zip-compressed', 'application/octet-stream' ],
			'xls' => [ 'application/vnd.ms-excel', 'application/octet-stream' ],
			'xlsx' => [ 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet', 'application/zip', 'application/x-zip-compressed', 'application/octet-stream' ],
		];
		if ( ! isset( $allowed[ $extension ] ) ) { return new WP_Error( 'sn_project_file_type', 'این نوع فایل مجاز نیست. فرمت‌های مجاز: تصویر، PDF، TXT، Word و Excel.' ); }
		$mime = '';
		if ( class_exists( 'finfo' ) ) { $finfo = new finfo( FILEINFO_MIME_TYPE ); $mime = (string) $finfo->file( (string) $file['tmp_name'] ); }
		if ( $mime === '' ) { $mime = sanitize_mime_type( (string) ( $file['type'] ?? '' ) ); }
		if ( ! in_array( $mime, $allowed[ $extension ], true ) ) { return new WP_Error( 'sn_project_file_mime', 'محتوای واقعی فایل با پسوند آن سازگار نیست.' ); }
		if ( ! $this->ensure_private_upload_dir() ) { return new WP_Error( 'sn_project_file_dir', 'فضای امن بارگذاری فایل آماده نیست.' ); }
		try { $random = bin2hex( random_bytes( 24 ) ); $token = hash( 'sha256', random_bytes( 32 ) ); }
		catch ( Throwable $e ) { return new WP_Error( 'sn_project_file_random', 'ساخت شناسه امن فایل ممکن نشد.' ); }
		$relative = substr( $random, 0, 2 ) . '/' . $random . '.bin';
		$directory = $this->private_upload_base() . '/' . substr( $random, 0, 2 );
		if ( ! wp_mkdir_p( $directory ) || ! move_uploaded_file( (string) $file['tmp_name'], $this->private_upload_base() . '/' . $relative ) ) { return new WP_Error( 'sn_project_file_move', 'ذخیره امن فایل انجام نشد.' ); }
		@chmod( $this->private_upload_base() . '/' . $relative, 0640 );
		return [ 'token' => $token, 'original_name' => $original ?: 'file.' . $extension, 'storage_name' => $relative, 'mime' => $mime, 'size' => $size ];
	}

	private function message_rows( int $item_id, int $viewer_id ): array {
		global $wpdb; $t = $this->tables();
		$rows = $wpdb->get_results( $wpdb->prepare( "SELECT * FROM {$t['messages']} WHERE membership_item_id=%d ORDER BY id DESC LIMIT 200", $item_id ), ARRAY_A ) ?: [];
		$rows = array_reverse( $rows ); $out = [];
		foreach ( $rows as $row ) {
			$user = get_user_by( 'id', (int) $row['sender_user_id'] );
			$download = '';
			if ( ! empty( $row['attachment_token'] ) ) {
				$download = add_query_arg( [ 'action' => 'sn_project_download', 'token' => (string) $row['attachment_token'], '_wpnonce' => wp_create_nonce( 'sn_project_download_' . (string) $row['attachment_token'] ) ], admin_url( 'admin-post.php' ) );
			}
			$out[] = [
				'id' => (int) $row['id'], 'sender_user_id' => (int) $row['sender_user_id'],
				'sender_name' => $user instanceof WP_User ? (string) $user->display_name : '#' . (int) $row['sender_user_id'],
				'is_self' => (int) $row['sender_user_id'] === $viewer_id,
				'text' => (string) ( $row['message_text'] ?? '' ), 'created_at' => (string) $row['created_at'],
				'attachment' => $download !== '' ? [ 'name' => (string) $row['attachment_original_name'], 'size' => (int) $row['attachment_size'], 'mime' => (string) $row['attachment_mime'], 'url' => $download ] : null,
			];
		}
		$last_id = $out ? (int) end( $out )['id'] : 0;
		$wpdb->query( $wpdb->prepare( "INSERT INTO {$t['reads']} (membership_item_id,user_id,last_message_id,read_at) VALUES (%d,%d,%d,%s) ON DUPLICATE KEY UPDATE last_message_id=GREATEST(last_message_id,VALUES(last_message_id)),read_at=VALUES(read_at)", $item_id, $viewer_id, $last_id, current_time( 'mysql' ) ) );
		return $out;
	}

	private function unread_notifications( int $user_id, string $role ): array {
		global $wpdb; $t = $this->tables();
		$started_at = (string) get_option( 'sn_project_notifications_started_at', '' );
		if ( $started_at === '' ) {
			$started_at = current_time( 'mysql' );
			update_option( 'sn_project_notifications_started_at', $started_at, false );
		}
		$scope_sql = '';
		$scope_args = [];
		$operations_table = $wpdb->prefix . 'sn_project_operations';
		$operations_join = $this->table_exists( $operations_table ) ? " LEFT JOIN {$operations_table} ops ON ops.membership_item_id=mi.id" : '';
		if ( $role === 'sales_supervisor' ) {
			$ids = $this->sales_chat_scope_user_ids( $user_id );
			if ( ! $ids ) { return [ 'total' => 0, 'threads' => [] ]; }
			$placeholders = implode( ',', array_fill( 0, count( $ids ), '%d' ) );
			$scope_sql = " AND COALESCE(NULLIF(m.sales_chat_user_id,0),m.origin_user_id) IN ({$placeholders})";
			$scope_args = $ids;
		} elseif ( in_array( $role, [ 'operations_sales_expert', 'operations_sales_supervisor' ], true ) ) {
			if ( $operations_join === '' ) { return [ 'total' => 0, 'threads' => [] ]; }
			// A reassigned expert sees the complete thread when opened, but only
			// receives unread alerts for messages created after their assignment.
			$scope_sql = ' AND ops.sales_expert_user_id=%d AND msg.created_at>=COALESCE(ops.assigned_at,m.activated_at)';
			$scope_args = [ $user_id ];
		} elseif ( in_array( $role, [ 'seller', 'converter' ], true ) ) {
			$scope_sql = ' AND COALESCE(NULLIF(m.sales_chat_user_id,0),m.origin_user_id)=%d';
			$scope_args = [ $user_id ];
		} elseif ( $role !== 'admin' ) {
			$scope_sql = ' AND (mi.manager_user_id=%d OR mi.expert_user_id=%d)';
			$scope_args = [ $user_id, $user_id ];
		}
		$joins = " INNER JOIN {$t['memberships']} m ON m.id=mi.membership_id{$operations_join} INNER JOIN {$t['messages']} msg ON msg.membership_item_id=mi.id LEFT JOIN {$t['reads']} rd ON rd.membership_item_id=mi.id AND rd.user_id=%d";
		$where = "m.status='active' AND msg.sender_user_id<>%d AND msg.created_at>=%s AND msg.id>COALESCE(rd.last_message_id,0){$scope_sql}";
		$args = array_merge( [ $user_id, $user_id, $started_at ], $scope_args );
		$total_sql = "SELECT COUNT(msg.id) FROM {$t['items']} mi {$joins} WHERE {$where}";
		$total = (int) $wpdb->get_var( $wpdb->prepare( $total_sql, ...$args ) );
		$list_sql = "SELECT mi.id item_id,mi.membership_id,mi.content_name_snapshot,m.customer_name,m.subscription_name_snapshot,COUNT(msg.id) unread_count,MAX(msg.id) last_message_id,MAX(msg.created_at) last_message_at FROM {$t['items']} mi {$joins} WHERE {$where} GROUP BY mi.id,mi.membership_id,mi.content_name_snapshot,m.customer_name,m.subscription_name_snapshot ORDER BY last_message_id DESC LIMIT 20";
		$rows = $wpdb->get_results( $wpdb->prepare( $list_sql, ...$args ), ARRAY_A ) ?: [];
		$message_ids = array_values( array_unique( array_filter( array_map( static fn( $row ) => absint( $row['last_message_id'] ?? 0 ), $rows ) ) ) );
		$last_messages = [];
		if ( $message_ids ) {
			$ids_sql = implode( ',', array_map( 'intval', $message_ids ) );
			foreach ( $wpdb->get_results( "SELECT id,sender_user_id,message_text,attachment_original_name,created_at FROM {$t['messages']} WHERE id IN ({$ids_sql})", ARRAY_A ) ?: [] as $message ) {
				$last_messages[ (int) $message['id'] ] = $message;
			}
		}
		$out = [];
		foreach ( $rows as $row ) {
			$message = $last_messages[ (int) $row['last_message_id'] ] ?? [];
			$sender = ! empty( $message['sender_user_id'] ) ? get_user_by( 'id', (int) $message['sender_user_id'] ) : null;
			$preview = trim( (string) ( $message['message_text'] ?? '' ) );
			if ( $preview === '' && ! empty( $message['attachment_original_name'] ) ) { $preview = 'فایل: ' . (string) $message['attachment_original_name']; }
			if ( function_exists( 'mb_substr' ) ) { $preview = mb_substr( $preview, 0, 120 ); } else { $preview = substr( $preview, 0, 120 ); }
			$out[] = [
				'item_id' => (int) $row['item_id'], 'membership_id' => (int) $row['membership_id'],
				'customer_name' => (string) $row['customer_name'], 'subscription_name' => (string) $row['subscription_name_snapshot'],
				'content_name' => (string) $row['content_name_snapshot'], 'unread_count' => (int) $row['unread_count'],
				'last_message_id' => (int) $row['last_message_id'], 'last_message_at' => (string) ( $message['created_at'] ?? $row['last_message_at'] ),
				'last_sender_name' => $sender instanceof WP_User ? (string) $sender->display_name : 'کاربر پروژه', 'preview' => $preview,
			];
		}
		return [ 'total' => $total, 'threads' => $out ];
	}

	public function ajax_notifications(): void {
		$this->verify_ajax();
		$user_id = get_current_user_id(); $role = $this->current_role();
		if ( ! in_array( $role, [ 'admin', 'project_manager', 'project_expert', 'seller', 'converter', 'sales_supervisor', 'operations_sales_expert', 'operations_sales_supervisor' ], true ) ) {
			wp_send_json( [ 'success' => false, 'message' => 'اعلان‌های پروژه برای این نقش در دسترس نیست.' ], 403 );
		}
		$notifications = $this->unread_notifications( $user_id, $role );
		wp_send_json( [ 'success' => true, 'total' => $notifications['total'], 'threads' => $notifications['threads'] ] );
	}

	private function note_rows( int $item_id ): array {
		global $wpdb; $t = $this->tables();
		$rows = $wpdb->get_results( $wpdb->prepare( "SELECT * FROM {$t['notes']} WHERE membership_item_id=%d ORDER BY id DESC LIMIT 100", $item_id ), ARRAY_A ) ?: [];
		foreach ( $rows as &$row ) { $user = get_user_by( 'id', (int) $row['author_user_id'] ); $row['author_name'] = $user instanceof WP_User ? (string) $user->display_name : '#' . (int) $row['author_user_id']; $row['id'] = (int) $row['id']; }
		return $rows;
	}

	public function ajax_thread(): void {
		$this->verify_ajax(); $membership_id = absint( $_POST['membership_id'] ?? 0 ); $requested_item_id = absint( $_POST['item_id'] ?? 0 ); $user_id = get_current_user_id();
		global $wpdb; $t = $this->tables();
		$item_ids = $wpdb->get_col( $wpdb->prepare( "SELECT id FROM {$t['items']} WHERE membership_id=%d ORDER BY id", $membership_id ) );
		$items = []; $contexts = [];
		foreach ( (array) $item_ids as $item_id ) {
			$context = $this->item_context( (int) $item_id );
			if ( $context && $this->can_access_item( $context, $user_id ) ) { $contexts[ (int) $item_id ] = $context; $items[] = $this->serialize_item( $context, $user_id, true ); }
		}
		if ( ! $items ) { wp_send_json( [ 'success' => false, 'message' => 'پرونده‌ای در محدوده دسترسی شما یافت نشد.' ], 403 ); }
		$item_id = $requested_item_id && isset( $contexts[ $requested_item_id ] ) ? $requested_item_id : (int) $items[0]['id'];
		$selected = $contexts[ $item_id ];
		wp_send_json( [
			'success' => true, 'membership_id' => $membership_id, 'items' => $items, 'selected_item_id' => $item_id,
			'messages' => $this->message_rows( $item_id, $user_id ),
			'notes' => $this->can_manage_item( $selected, $user_id ) ? $this->note_rows( $item_id ) : [],
			'can_manage' => $this->can_manage_item( $selected, $user_id ),
		] );
	}

	public function ajax_send_message(): void {
		$this->verify_ajax(); $item_id = absint( $_POST['item_id'] ?? 0 ); $user_id = get_current_user_id();
		$rate_key = 'sn_project_msg_rate_' . $user_id;
		$rate_count = (int) get_transient( $rate_key );
		if ( $rate_count >= 30 ) { wp_send_json( [ 'success' => false, 'message' => 'تعداد پیام‌های شما در یک دقیقه بیش از حد مجاز است؛ کمی بعد دوباره تلاش کنید.' ], 429 ); }
		$text = sanitize_textarea_field( wp_unslash( $_POST['message'] ?? '' ) );
		if ( function_exists( 'mb_substr' ) ) { $text = mb_substr( $text, 0, 5000 ); } else { $text = substr( $text, 0, 5000 ); }
		$item = $this->item_context( $item_id );
		if ( ! $item || ! $this->can_access_item( $item, $user_id ) ) { wp_send_json( [ 'success' => false, 'message' => 'اجازه ارسال پیام در این گفتگو را ندارید.' ], 403 ); }
		$attachment = null;
		if ( ! empty( $_FILES['attachment']['name'] ) ) {
			$attachment = $this->store_attachment( $_FILES['attachment'] );
			if ( is_wp_error( $attachment ) ) { wp_send_json( [ 'success' => false, 'message' => $attachment->get_error_message() ], 400 ); }
		}
		if ( $text === '' && ! is_array( $attachment ) ) { wp_send_json( [ 'success' => false, 'message' => 'پیام یا فایل را وارد کنید.' ], 400 ); }
		$data = [ 'membership_item_id' => $item_id, 'sender_user_id' => $user_id, 'message_text' => $text !== '' ? $text : null, 'created_at' => current_time( 'mysql' ) ];
		if ( is_array( $attachment ) ) {
			$data += [ 'attachment_token' => $attachment['token'], 'attachment_original_name' => $attachment['original_name'], 'attachment_storage_name' => $attachment['storage_name'], 'attachment_mime' => $attachment['mime'], 'attachment_size' => $attachment['size'] ];
		}
		global $wpdb; $t = $this->tables();
		if ( ! $wpdb->insert( $t['messages'], $data ) ) {
			if ( is_array( $attachment ) ) { @unlink( $this->private_upload_base() . '/' . $attachment['storage_name'] ); }
			wp_send_json( [ 'success' => false, 'message' => 'ارسال پیام انجام نشد.' ], 500 );
		}
		set_transient( $rate_key, $rate_count + 1, MINUTE_IN_SECONDS );
		$this->event( (int) $item->membership_id, $item_id, 'chat_message_sent', [ 'message_id' => (int) $wpdb->insert_id, 'has_attachment' => is_array( $attachment ) ], $user_id );
		wp_send_json( [ 'success' => true, 'message' => 'پیام ارسال شد.' ] );
	}

	public function handle_download(): void {
		if ( ! is_user_logged_in() ) { status_header( 401 ); exit; }
		$token = sanitize_text_field( wp_unslash( $_GET['token'] ?? '' ) );
		if ( ! preg_match( '/^[a-f0-9]{64}$/', $token ) || ! wp_verify_nonce( sanitize_text_field( wp_unslash( $_GET['_wpnonce'] ?? '' ) ), 'sn_project_download_' . $token ) ) { status_header( 403 ); exit; }
		global $wpdb; $t = $this->tables();
		$message = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['messages']} WHERE attachment_token=%s LIMIT 1", $token ) );
		if ( ! $message ) { status_header( 404 ); exit; }
		$item = $this->item_context( (int) $message->membership_item_id );
		if ( ! $item || ! $this->can_access_item( $item, get_current_user_id() ) ) { status_header( 403 ); exit; }
		$relative = (string) $message->attachment_storage_name;
		if ( ! preg_match( '#^[a-f0-9]{2}/[a-f0-9]{48}\.bin$#', $relative ) ) { status_header( 404 ); exit; }
		$path = $this->private_upload_base() . '/' . $relative;
		if ( ! is_file( $path ) || ! is_readable( $path ) ) { status_header( 404 ); exit; }
		while ( ob_get_level() > 0 ) { ob_end_clean(); }
		$mime = sanitize_mime_type( (string) $message->attachment_mime );
		if ( $mime === '' ) { $mime = 'application/octet-stream'; }
		nocache_headers(); header( 'X-Content-Type-Options: nosniff' ); header( 'Content-Type: ' . $mime );
		$filename = sanitize_file_name( (string) $message->attachment_original_name ) ?: 'attachment';
		header( "Content-Disposition: attachment; filename=\"download\"; filename*=UTF-8''" . rawurlencode( $filename ) );
		header( 'Content-Length: ' . (string) filesize( $path ) ); readfile( $path ); exit;
	}

	private function explicit_link_invoice( object $item, object $invoice, int $actor_id ): array {
		if ( ! $this->invoice_is_verified( $invoice ) ) { return [ 'success' => false, 'message' => 'فاکتور باید پرداخت‌شده یا تاییدشده مالی باشد.' ]; }
		if ( $this->normalize_phone( (string) $invoice->customer_phone ) !== $this->normalize_phone( (string) $item->customer_phone ) ) { return [ 'success' => false, 'message' => 'شماره مشتری فاکتور با پرونده پروژه یکسان نیست.' ]; }
		global $wpdb; $t = $this->tables(); $now = current_time( 'mysql' );
		if ( (int) $invoice->id === (int) $item->source_invoice_id ) { return [ 'success' => false, 'message' => 'فاکتور اولیه خرید اشتراک را نمی‌توان به‌عنوان خرید حاصل از محتوا ثبت کرد.' ]; }
		if ( ! empty( $invoice->created_at ) && ! empty( $item->activated_at ) && (string) $invoice->created_at < (string) $item->activated_at ) { return [ 'success' => false, 'message' => 'این فاکتور پیش از فعال‌شدن اشتراک مشتری ساخته شده است.' ]; }
		$invoice_products = [];
		foreach ( $this->invoice_items( (int) $invoice->id ) as $row ) { $invoice_products[] = (int) ( $row['product_id'] ?? 0 ); }
		$invoice_products[] = (int) ( $invoice->product_id ?? 0 );
		if ( array_intersect( array_filter( $invoice_products ), $this->subscription_product_ids() ) ) { return [ 'success' => false, 'message' => 'فاکتور فروش اشتراک قابل انتساب به محتوای پروژه نیست؛ یک فاکتور خرید بعدی مشتری را انتخاب کنید.' ]; }
		$existing_link = $wpdb->get_row( $wpdb->prepare( "SELECT membership_item_id,status FROM {$t['invoice_links']} WHERE invoice_id=%d LIMIT 1", (int) $invoice->id ) );
		if ( $existing_link && (int) $existing_link->membership_item_id !== (int) $item->id ) { return [ 'success' => false, 'message' => 'این فاکتور قبلاً به یک محتوای دیگر نسبت داده شده و برای جلوگیری از پورسانت تکراری قابل استفاده مجدد نیست.' ]; }
		$line_total = max( 0, $this->invoice_total( $invoice ) );
		if ( $line_total <= 0 ) { return [ 'success' => false, 'message' => 'مبلغ نهایی معتبر برای این فاکتور پیدا نشد.' ]; }
		$sql = $wpdb->prepare( "INSERT INTO {$t['invoice_links']} (membership_item_id,invoice_id,content_product_id,line_total,status,linked_by_user_id,verified_at,created_at,updated_at) VALUES (%d,%d,%d,%f,'verified',%d,%s,%s,%s) ON DUPLICATE KEY UPDATE line_total=VALUES(line_total),status='verified',linked_by_user_id=VALUES(linked_by_user_id),verified_at=VALUES(verified_at),updated_at=VALUES(updated_at)", (int) $item->id, (int) $invoice->id, (int) $item->content_product_id, $line_total, $actor_id, $now, $now, $now );
		if ( false === $wpdb->query( $sql ) ) { return [ 'success' => false, 'message' => 'اتصال فاکتور ذخیره نشد.' ]; }
		// Chat ownership stays bound to the latest actor who issued the
		// subscription payment. A later content/product invoice must not steal
		// the conversation from that sales-side owner.
		$this->credit_biavin_commission( (int) $item->id, (int) $item->origin_user_id, (int) $invoice->id, (int) $item->content_product_id, $line_total );
		$this->event( (int) $item->membership_id, (int) $item->id, 'verified_invoice_attributed', [ 'invoice_id' => (int) $invoice->id, 'invoice_total' => $line_total ], $actor_id );
		return [ 'success' => true, 'message' => 'فاکتور تاییدشده به این محتوا نسبت داده شد.', 'line_total' => $line_total ];
	}

	public function ajax_link_invoice(): void {
		$this->verify_ajax(); $item_id = absint( $_POST['item_id'] ?? 0 ); $invoice_ref = sanitize_text_field( wp_unslash( $_POST['invoice'] ?? '' ) ); $user_id = get_current_user_id();
		$item = $this->item_context( $item_id );
		if ( ! $item || ! $this->can_manage_item( $item, $user_id ) ) { wp_send_json( [ 'success' => false, 'message' => 'اجازه اتصال فاکتور به این پرونده را ندارید.' ], 403 ); }
		global $wpdb;
		$invoice = ctype_digit( $invoice_ref ) ? $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$wpdb->prefix}sn_invoices WHERE id=%d OR invoice_code=%s ORDER BY id DESC LIMIT 1", (int) $invoice_ref, $invoice_ref ) ) : $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$wpdb->prefix}sn_invoices WHERE invoice_code=%s LIMIT 1", $invoice_ref ) );
		if ( ! $invoice ) { wp_send_json( [ 'success' => false, 'message' => 'فاکتور یافت نشد.' ], 404 ); }
		$result = $this->explicit_link_invoice( $item, $invoice, $user_id );
		wp_send_json( $result, ! empty( $result['success'] ) ? 200 : 400 );
	}

	private function wallet_summary( int $user_id ): array {
		global $wpdb; $wallet_id = $this->wallet_id( $user_id );
		$wallet = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$wpdb->prefix}sn_wallets WHERE id=%d", $wallet_id ), ARRAY_A ) ?: [];
		$transactions = $wpdb->get_results( $wpdb->prepare( "SELECT id,invoice_id,amount,direction,type,description,created_at FROM {$wpdb->prefix}sn_wallet_transactions WHERE wallet_id=%d ORDER BY id DESC LIMIT 50", $wallet_id ), ARRAY_A ) ?: [];
		foreach ( $transactions as &$row ) { $row['id'] = (int) $row['id']; $row['invoice_id'] = (int) $row['invoice_id']; $row['amount'] = (float) $row['amount']; $row['amount_fmt'] = class_exists( 'SN_Helpers' ) ? SN_Helpers::format_price( $row['amount'] ) : number_format_i18n( $row['amount'] ); }
		return [
			'balance' => (float) ( $wallet['balance'] ?? 0 ), 'total_credit' => (float) ( $wallet['total_credit'] ?? 0 ), 'total_debit' => (float) ( $wallet['total_debit'] ?? 0 ),
			'balance_fmt' => class_exists( 'SN_Helpers' ) ? SN_Helpers::format_price( (float) ( $wallet['balance'] ?? 0 ) ) : number_format_i18n( (float) ( $wallet['balance'] ?? 0 ) ),
			'transactions' => $transactions,
		];
	}

	public function ajax_wallet(): void {
		$this->verify_ajax(); $user_id = get_current_user_id(); $role = $this->current_role();
		if ( ! in_array( $role, [ 'seller', 'converter', 'sales_supervisor', 'admin' ], true ) ) { wp_send_json( [ 'success' => false, 'message' => 'کیف پول بیاوین برای این نقش در دسترس نیست.' ], 403 ); }
		if ( $role === 'admin' && ! empty( $_POST['user_id'] ) ) { $user_id = absint( $_POST['user_id'] ); }
		wp_send_json( [ 'success' => true, 'wallet' => $this->wallet_summary( $user_id ) ] );
	}

	public function project_summary_for_invoice( int $invoice_id, int $origin_user_id = 0 ): ?array {
		$all = $this->project_summaries_for_invoices( [ $invoice_id ], $origin_user_id );
		return $all[ $invoice_id ] ?? null;
	}

	public function project_summaries_for_invoices( array $invoice_ids, int $origin_user_id = 0 ): array {
		global $wpdb; $t = $this->tables();
		$invoice_ids = array_values( array_unique( array_filter( array_map( 'absint', $invoice_ids ) ) ) );
		if ( ! $invoice_ids || ! $this->table_exists( $t['memberships'] ) ) { return []; }
		$ids_sql = implode( ',', $invoice_ids ); $memberships = []; $invoice_membership = [];
		$current_role = $this->current_role();
		// For direct sales actors the invoice UI must expose the chat only to the
		// current sales-side conversation owner. This may be different from the
		// original seller (for example a converter or supervisor who issued the
		// latest subscription payment). Higher sales roles keep their scoped
		// visibility and the thread endpoint still performs the final ACL check.
		$restrict_chat_owner = $origin_user_id > 0 && in_array( $current_role, [ 'seller', 'converter' ], true );
		$membership_visible = static function ( object $membership ) use ( $restrict_chat_owner, $origin_user_id ): bool {
			if ( ! $restrict_chat_owner ) { return true; }
			$chat_owner = (int) ( $membership->sales_chat_user_id ?? 0 );
			if ( $chat_owner < 1 ) { $chat_owner = (int) ( $membership->origin_user_id ?? 0 ); }
			return $chat_owner === $origin_user_id;
		};
		$direct_rows = $wpdb->get_results( "SELECT * FROM {$t['memberships']} WHERE source_invoice_id IN ({$ids_sql}) ORDER BY id DESC" ) ?: [];
		foreach ( $direct_rows as $membership ) {
			if ( ! $membership_visible( $membership ) ) { continue; }
			$memberships[ (int) $membership->id ] = $membership;
			if ( ! isset( $invoice_membership[ (int) $membership->source_invoice_id ] ) ) { $invoice_membership[ (int) $membership->source_invoice_id ] = (int) $membership->id; }
		}
		$dot_links = $wpdb->prefix . 'sn_dot_invoice_links';
		if ( $this->table_exists( $dot_links ) ) {
			$case_by_invoice = [];
			foreach ( $wpdb->get_results( "SELECT invoice_id,case_id FROM {$dot_links} WHERE invoice_id IN ({$ids_sql}) AND flow_kind='conversion_payment'", ARRAY_A ) ?: [] as $row ) { $case_by_invoice[ (int) $row['invoice_id'] ] = (int) $row['case_id']; }
			$case_ids = array_values( array_unique( array_filter( $case_by_invoice ) ) );
			if ( $case_ids ) {
				$case_sql = implode( ',', array_map( 'intval', $case_ids ) ); $membership_by_case = [];
				foreach ( $wpdb->get_results( "SELECT * FROM {$t['memberships']} WHERE dot_case_id IN ({$case_sql}) ORDER BY id DESC" ) ?: [] as $membership ) {
					if ( ! $membership_visible( $membership ) ) { continue; }
					$memberships[ (int) $membership->id ] = $membership;
					if ( ! isset( $membership_by_case[ (int) $membership->dot_case_id ] ) ) { $membership_by_case[ (int) $membership->dot_case_id ] = (int) $membership->id; }
				}
				foreach ( $case_by_invoice as $invoice_id => $case_id ) { if ( isset( $membership_by_case[ $case_id ] ) ) { $invoice_membership[ $invoice_id ] = $membership_by_case[ $case_id ]; } }
			}
		}
		// A later verified invoice may be linked to a specific subscription
		// content (for example the final product* invoice). Include that link so
		// the invoice row exposes the same per-content chat entry point.
		if ( $this->table_exists( $t['invoice_links'] ) ) {
			$linked_rows = $wpdb->get_results( "SELECT il.invoice_id,m.* FROM {$t['invoice_links']} il INNER JOIN {$t['items']} mi ON mi.id=il.membership_item_id INNER JOIN {$t['memberships']} m ON m.id=mi.membership_id WHERE il.invoice_id IN ({$ids_sql}) AND il.status='verified' ORDER BY il.id DESC" ) ?: [];
			foreach ( $linked_rows as $membership ) {
				if ( ! $membership_visible( $membership ) ) { continue; }
				$memberships[ (int) $membership->id ] = $membership;
				if ( ! isset( $invoice_membership[ (int) $membership->invoice_id ] ) ) { $invoice_membership[ (int) $membership->invoice_id ] = (int) $membership->id; }
			}
		}
		if ( ! $memberships ) { return []; }
		$membership_ids_sql = implode( ',', array_map( 'intval', array_keys( $memberships ) ) ); $items_by_membership = [];
		foreach ( $wpdb->get_results( "SELECT * FROM {$t['items']} WHERE membership_id IN ({$membership_ids_sql}) ORDER BY id", ARRAY_A ) ?: [] as $row ) {
			$items_by_membership[ (int) $row['membership_id'] ][] = [ 'id' => (int) $row['id'], 'name' => (string) $row['content_name_snapshot'], 'status' => (string) $row['workflow_status'], 'status_label' => $this->status_label( (string) $row['workflow_status'] ), 'expert_user_id' => (int) $row['expert_user_id'], 'manager_user_id' => (int) $row['manager_user_id'], 'cancel_reason' => (string) ( $row['cancel_reason'] ?? '' ), 'contact_attempts' => (int) ( $row['contact_attempts'] ?? 0 ) ];
		}
		$out = [];
		foreach ( $invoice_membership as $invoice_id => $membership_id ) {
			$membership = $memberships[ $membership_id ] ?? null; if ( ! $membership ) { continue; }
			$is_active = (string) $membership->status === 'active';
			$out[ $invoice_id ] = [
				'membership_id' => $membership_id,
				'subscription_name' => (string) $membership->subscription_name_snapshot,
				'items' => $items_by_membership[ $membership_id ] ?? [],
				'status' => (string) $membership->status,
				'tag' => $is_active ? 'اشتراک فعال · بیاوین' : 'اشتراک معلق · بیاوین',
			];
		}
		return $out;
	}

	/** Backward-compatible shortcode targets for pages created before 2.0.44. */
	public function render_manager_panel(): string {
		return class_exists( 'SN_Operations_Flow' ) ? SN_Operations_Flow::instance()->render_sales_manager_panel() : '';
	}
	public function render_expert_panel(): string {
		return class_exists( 'SN_Operations_Flow' ) ? SN_Operations_Flow::instance()->render_sales_expert_panel() : '';
	}

	private function render_project_panel( string $required_role ): string {
		if ( ! is_user_logged_in() ) { $login = (int) get_option( 'sn_login_page_id', 0 ); $url = $login ? get_permalink( $login ) : wp_login_url(); return '<script>window.location.href=' . wp_json_encode( $url ) . ';</script><div class="sn-notice">در حال انتقال به صفحه ورود...</div>'; }
		$role = $this->current_role();
		if ( $role !== 'admin' && $role !== $required_role ) { return '<div class="sn-notice sn-error" dir="rtl">این صفحه برای نقش ' . esc_html( $required_role === 'project_manager' ? 'مدیر پروژه' : 'کارشناس پروژه' ) . ' است.</div>'; }
		$title = $required_role === 'project_manager' ? 'پنل مدیر پروژه' : 'پنل کارشناس پروژه';
		ob_start(); ?>
		<div class="sn-project-panel" data-sn-project-panel="<?php echo esc_attr( $required_role ); ?>" dir="rtl">
			<header class="sn-project-panel-head"><div><h1><?php echo esc_html( $title ); ?></h1><p>مدیریت مشتریان اشتراک، محتواها، خریدهای تاییدشده و گفت‌وگوی بیاوین</p></div><button type="button" class="sn-project-refresh button">بروزرسانی</button></header>
			<div class="sn-project-stats" aria-live="polite"></div>
			<div class="sn-project-filters"><input type="search" class="sn-project-search" placeholder="جستجو نام، شماره، اشتراک یا محتوا"><select class="sn-project-status-filter"><option value="">همه وضعیت‌ها</option><option value="waiting">در انتظار</option><option value="no_answer">جواب نداده</option><option value="follow_up">تماس مجدد</option><option value="cancelled">انصراف</option><option value="normal">عادی</option><option value="upsell">افزایشی</option><option value="purchased">خرید کرده (قدیمی)</option><option value="regretted">پشیمان شده (قدیمی)</option></select></div>
			<div class="sn-project-list"><div class="sn-project-loading">در حال دریافت پرونده‌ها...</div></div>
		</div>
		<?php return (string) ob_get_clean();
	}

	private function current_post_has_project_assets(): bool {
		global $post;
		if ( ! $post || ! is_singular() ) { return false; }
		$page_ids = array_filter( array_map( 'absint', [
			get_option( 'sn_seller_panel_page_id', 0 ),
			get_option( 'sn_dot_converter_panel_page_id', 0 ),
			get_option( 'sn_supervisor_panel_page_id', 0 ),
			get_option( 'sn_sales_manager_panel_page_id', 0 ),
			get_option( 'sn_project_manager_panel_page_id', 0 ),
			get_option( 'sn_project_expert_panel_page_id', 0 ),
			get_option( 'sn_operations_sales_supervisor_panel_page_id', 0 ),
			get_option( 'sn_operations_sales_expert_panel_page_id', 0 ),
			get_option( 'sn_my_panel_page_id', 0 ),
		] ) );
		if ( in_array( (int) $post->ID, $page_ids, true ) ) { return true; }
		$content = (string) ( $post->post_content ?? '' );
		$elementor = (string) get_post_meta( (int) $post->ID, '_elementor_data', true );
		foreach ( [ 'sn_seller_panel', 'sn_dot_converter_panel', 'sn_supervisor_panel', 'sn_senior_supervisor_panel', 'sn_sales_manager_panel', 'sn_project_manager_panel', 'sn_project_expert_panel', 'sn_operations_sales_supervisor_panel', 'sn_operations_sales_expert_panel', 'sn_my_panel', 'sn_portal_nav' ] as $shortcode ) {
			if ( has_shortcode( $content, $shortcode ) || strpos( $elementor, $shortcode ) !== false ) { return true; }
		}
		return false;
	}

	public function enqueue_assets(): void {
		if ( ! $this->current_post_has_project_assets() ) { return; }
		$css = SN_PLUGIN_DIR . 'assets/css/projects.css'; $js = SN_PLUGIN_DIR . 'assets/js/public-projects.js';
		if ( ! wp_style_is( 'sn-public', 'enqueued' ) ) { wp_enqueue_style( 'sn-public', SN_PLUGIN_URL . 'assets/css/public.css', [], SN_VERSION . '-' . ( file_exists( SN_PLUGIN_DIR . 'assets/css/public.css' ) ? (string) filemtime( SN_PLUGIN_DIR . 'assets/css/public.css' ) : '0' ) ); }
		wp_enqueue_style( 'sn-projects', SN_PLUGIN_URL . 'assets/css/projects.css', [ 'sn-public' ], SN_VERSION . '-' . ( file_exists( $css ) ? (string) filemtime( $css ) : '0' ) );
		wp_enqueue_script( 'sn-projects', SN_PLUGIN_URL . 'assets/js/public-projects.js', [ 'jquery' ], SN_VERSION . '-' . ( file_exists( $js ) ? (string) filemtime( $js ) : '0' ), true );
		wp_localize_script( 'sn-projects', 'snProjects', [ 'ajaxUrl' => admin_url( 'admin-ajax.php' ), 'nonce' => wp_create_nonce( 'sn_projects' ), 'role' => $this->current_role(), 'maxFileSize' => self::MAX_FILE_SIZE ] );
	}
}
