<?php
if ( ! defined( 'ABSPATH' ) ) { exit; }

class SN_Activator {
	const DB_VERSION = '2026-08-01-staged-payment-workflow-v1';

	public static function activate() {
		self::register_roles();
		self::register_report_caps();
		self::migrate_finance_user_caps();
		self::create_tables();
		self::register_report_state_schema();
		// A newly activated plugin is included after WordPress has already fired
		// plugins_loaded. Load and run the additive migration service explicitly so
		// HR/MIS tables exist before project and Dot modules seed their HR mappings.
		if ( ! class_exists( 'SN_Migration_Service' ) && defined( 'SN_PLUGIN_DIR' ) ) {
			require_once SN_PLUGIN_DIR . 'includes/class-sn-migration-service.php';
		}
		if ( class_exists( 'SN_Migration_Service' ) ) {
			( new SN_Migration_Service() )->migrate();
		}
		if ( class_exists( 'SN_Campaign_Tracking' ) ) {
			SN_Campaign_Tracking::activate();
		}
		if ( class_exists( 'SN_Projects' ) ) {
			SN_Projects::activate();
		}
		if ( class_exists( 'SN_Customer_Portal' ) ) {
			SN_Customer_Portal::activate();
		}
		if ( class_exists( 'SN_Operations_Flow' ) ) {
			SN_Operations_Flow::activate();
		}
		if ( class_exists( 'SN_Operations_Execution' ) ) {
			SN_Operations_Execution::activate();
		}
		if ( class_exists( 'SN_Seller_Flow' ) ) {
			SN_Seller_Flow::activate();
		}
		if ( class_exists( 'SN_Dot_Flow' ) ) {
			SN_Dot_Flow::activate();
		}
		if ( class_exists( 'SN_Product_Flow' ) ) {
			SN_Product_Flow::activate();
		}
		self::repair_stale_seller_status_ownership();
		self::insert_defaults();
		self::create_required_pages();
		update_option( 'sn_db_version', self::DB_VERSION );
	}

	public static function deactivate() {
		if ( class_exists( 'SN_Seller_Flow' ) ) {
			SN_Seller_Flow::deactivate();
		}
		if ( class_exists( 'SN_Dot_Flow' ) ) {
			SN_Dot_Flow::deactivate();
		}
		if ( class_exists( 'SN_Operations_Flow' ) ) {
			SN_Operations_Flow::deactivate();
		}
	}

	/** Additive permissions: existing installations are upgraded by the version check. */
	public static function register_report_caps(): void {
		$map = [
			'sn_seller' => [ 'sn_report_view_self', 'sn_report_export' ],
			'sn_supervisor' => [ 'sn_report_view_team', 'sn_report_export' ],
			'sn_senior_supervisor' => [ 'sn_report_view_team', 'sn_report_export' ],
			'sn_sales_manager' => [ 'sn_report_view_manager', 'sn_report_export' ],
			'sn_sales_deputy' => [ 'sn_report_view_all', 'sn_report_export' ],
			'sn_finance' => [ 'sn_report_view_all', 'sn_report_export', 'sn_report_audit' ],
			'sn_financial' => [ 'sn_report_view_all', 'sn_report_export', 'sn_report_audit' ],
			'sn_financial_approval' => [ 'sn_report_view_all', 'sn_report_export', 'sn_report_audit' ],
			'administrator' => [ 'sn_report_view_all', 'sn_report_export', 'sn_report_audit' ],
		];
		foreach ( $map as $name => $caps ) {
			$role = get_role( $name );
			if ( ! $role ) { continue; }
			foreach ( $caps as $cap ) { $role->add_cap( $cap ); }
		}
		update_option( 'sn_report_caps_version', '2.1.0', false );
	}

	/** Baseline for exact pending/rejected as-of queries from installation onward. */
	public static function register_report_state_schema(): void {
		global $wpdb;
		require_once ABSPATH . 'wp-admin/includes/upgrade.php';
		$table = $wpdb->prefix . 'sn_report_invoice_state_events';
		$charset = $wpdb->get_charset_collate();
		dbDelta( "CREATE TABLE {$table} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			invoice_id BIGINT UNSIGNED NOT NULL,
			seller_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
			state VARCHAR(60) NOT NULL,
			actor_user_id BIGINT UNSIGNED DEFAULT NULL,
			source_activity_id BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY source_activity_id (source_activity_id),
			KEY invoice_time (invoice_id,created_at),
			KEY state_time (state,created_at),
			KEY seller_time (seller_id,created_at)
		) {$charset};" );
		if ( $wpdb->get_var( $wpdb->prepare( 'SHOW TABLES LIKE %s', $table ) ) !== $table ) { return; }
		if ( get_option( 'sn_report_state_baseline_at' ) ) { return; }
		$at = current_time( 'mysql' );
		$invoices = $wpdb->prefix . 'sn_invoices';
		$result = $wpdb->query( $wpdb->prepare(
			"INSERT INTO {$table} (invoice_id,seller_id,state,created_at)
			 SELECT id,seller_id,CASE WHEN status='rejected' THEN 'rejected' ELSE 'pending_financial_approval' END,%s
			 FROM {$invoices} WHERE status IN ('rejected','pending_financial_approval')",
			$at
		) );
		if ( $result !== false ) { update_option( 'sn_report_state_baseline_at', $at, false ); }
	}

	private static function repair_stale_seller_status_ownership(): void {
		global $wpdb;
		$table = $wpdb->prefix . 'sn_distribution_items';
		if ( $wpdb->get_var( $wpdb->prepare( 'SHOW TABLES LIKE %s', $table ) ) !== $table ) { return; }
		$cols = (array) $wpdb->get_col( "SHOW COLUMNS FROM {$table}", 0 );
		$required = [ 'seller_contact_status', 'seller_updated_by', 'current_owner_user_id' ];
		foreach ( $required as $col ) { if ( ! in_array( $col, $cols, true ) ) { return; } }
		$sets = [];
		foreach ( [
			'seller_contact_status' => 'NULL',
			'seller_sale_probability' => '0',
			'seller_sale_probability_label' => 'NULL',
			'seller_next_followup_at' => 'NULL',
			'seller_notes' => 'NULL',
			'seller_updated_by' => 'NULL',
			'seller_updated_at' => 'NULL',
		] as $col => $val ) {
			if ( in_array( $col, $cols, true ) ) { $sets[] = "{$col}={$val}"; }
		}
		if ( empty( $sets ) ) { return; }
		$where = "seller_contact_status IS NOT NULL AND seller_contact_status <> '' AND COALESCE(seller_updated_by,0)>0 AND COALESCE(current_owner_user_id,0)>0 AND COALESCE(seller_updated_by,0)<>COALESCE(current_owner_user_id,0)";
		$before = (int) $wpdb->get_var( "SELECT COUNT(*) FROM {$table} WHERE {$where}" );
		$updated = $wpdb->query( "UPDATE {$table} SET " . implode( ', ', $sets ) . " WHERE {$where}" );
		update_option( 'sn_stale_seller_state_cleanup_report', [
			'version' => '1.0.134-force-stale-status-repair',
			'context' => 'activation',
			'stale_rows_before' => $before,
			'cleaned_rows' => is_numeric( $updated ) ? (int) $updated : 0,
			'last_error' => (string) $wpdb->last_error,
			'updated_at' => current_time( 'mysql' ),
		], false );
	}

	private static function register_roles() {
		add_role( 'sn_seller', 'فروشنده', [ 'read' => true ] );
		add_role( 'sn_supervisor', 'سرپرست فروش', [ 'read' => true ] );
		add_role( 'sn_senior_supervisor', 'سرپرست ارشد فروش', [ 'read' => true ] );
		add_role( 'sn_sales_deputy', 'معاون فروش', [ 'read' => true ] );
		add_role( 'sn_converter', 'تبدیل‌کننده', [ 'read' => true, 'sn_manage_dot_conversion' => true ] );
		add_role( 'sn_financial_approval', 'تایید مالی', [
			'read' => true,
		] );
		add_role( 'sn_financial', 'تایید مالی', [
			'read' => true,
		] );
		add_role( 'sn_finance', 'مالی', [ 'read' => true ] );
		add_role( 'sn_hr', 'منابع انسانی', [ 'read' => true ] );
		add_role( 'sn_mis', 'MIS', [ 'read' => true ] );
		add_role( 'sn_after_sales', 'خدمات پس از فروش', [ 'read' => true, 'sn_view_customer_profiles' => true ] );
		add_role( 'sn_sales_manager', 'مدیر فروش', [
			'read' => true,
			'sn_view_sales_reports' => true,
			'sn_manage_supervisor_leads' => true,
			'sn_export_sales_reports' => true,
		] );
		add_role( 'sn_campaign_partner', 'Campaign Partner (پارتنر کمپین)', [
			'read' => true,
			'sn_view_campaign_reports' => true,
		] );
			add_role( 'sn_shipping_expert', 'کارشناس ارسال', [ 'read' => true, 'sn_manage_shipping' => true ] );
		add_role( 'sn_operations_sales_manager', 'مدیر فروش عملیات', [ 'read' => true, 'sn_manage_operations_sales' => true ] );
		add_role( 'sn_operations_sales_supervisor', 'سرپرست فروش عملیات', [ 'read' => true, 'sn_supervise_operations_sales' => true ] );
		add_role( 'sn_operations_sales_expert', 'کارشناس فروش عملیات', [ 'read' => true, 'sn_work_operations_sales' => true ] );
		add_role( 'sn_operations_executive_manager', 'مدیر اجرایی عملیات', [ 'read' => true, 'sn_manage_operations_execution' => true ] );
		add_role( 'sn_operations_execution_expert', 'کارشناس اجرایی عملیات', [ 'read' => true, 'sn_work_operations_execution' => true ] );

		$finance = get_role( 'sn_financial_approval' );
		if ( $finance ) {
			$finance->add_cap( 'read' );
			foreach ( [ 'sn_view_payments', 'sn_approve_payments', 'sn_reject_payments', 'sn_view_finance', 'sn_approve_payment', 'sn_reject_payment' ] as $cap ) {
				$finance->remove_cap( $cap );
			}
		}
		$finance_alias = get_role( 'sn_financial' );
		if ( $finance_alias ) {
			$finance_alias->add_cap( 'read' );
			foreach ( [ 'sn_view_payments', 'sn_approve_payments', 'sn_reject_payments', 'sn_view_finance', 'sn_approve_payment', 'sn_reject_payment' ] as $cap ) {
				$finance_alias->remove_cap( $cap );
			}
		}

		// اطمینان از اینکه نقش‌های فرانت هیچ دسترسی به پیشخوان وردپرس ندارند.
			foreach ( [ 'sn_seller', 'sn_supervisor', 'sn_senior_supervisor', 'sn_sales_deputy', 'sn_converter', 'sn_hr', 'sn_mis', 'sn_finance', 'sn_after_sales', 'sn_sales_manager', 'sn_campaign_partner', 'sn_shipping_expert', 'sn_operations_sales_manager', 'sn_operations_sales_supervisor', 'sn_operations_sales_expert', 'sn_operations_executive_manager', 'sn_operations_execution_expert' ] as $role_key ) {
			$role = get_role( $role_key );
			if ( ! $role ) { continue; }
			$role->add_cap( 'read' );
			foreach ( [ 'edit_posts', 'delete_posts', 'publish_posts', 'upload_files', 'edit_pages', 'delete_pages', 'manage_options', 'list_users', 'create_users', 'edit_users', 'delete_users' ] as $cap ) {
				$role->remove_cap( $cap );
			}
		}
	}

	private static function migrate_finance_user_caps(): void {
		if ( get_option( 'sn_finance_user_caps_migrated' ) ) {
			return;
		}
		$users = get_users( [
			'role__in' => [ 'sn_financial', 'sn_financial_approval', 'sn_finance' ],
			'fields'   => 'all',
		] );
		foreach ( $users as $user ) {
			if ( ! $user instanceof WP_User ) {
				continue;
			}
			$user->add_cap( 'sn_view_finance' );
			$user->add_cap( 'sn_approve_payment' );
			$user->add_cap( 'sn_reject_payment' );
		}
		update_option( 'sn_finance_user_caps_migrated', current_time( 'mysql' ), false );
	}

	private static function create_tables() {
		global $wpdb;
		require_once ABSPATH . 'wp-admin/includes/upgrade.php';
		$charset = $wpdb->get_charset_collate();

		dbDelta( "CREATE TABLE {$wpdb->prefix}sn_leads (
			id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			phone        VARCHAR(20)     NOT NULL,
			customer_name VARCHAR(120)    DEFAULT NULL,
			national_id VARCHAR(20)       DEFAULT NULL,
			marketing_form_id VARCHAR(80) DEFAULT NULL,
			import_code  VARCHAR(80)     DEFAULT NULL,
			province     VARCHAR(60)     DEFAULT NULL,
			city         VARCHAR(60)     DEFAULT NULL,
			status       VARCHAR(20)     NOT NULL DEFAULT 'unassigned',
			lead_status  VARCHAR(60)     DEFAULT NULL,
			sales_prediction VARCHAR(120) DEFAULT NULL,
			destination_panel VARCHAR(50) DEFAULT NULL,
			destination_routed_at DATETIME DEFAULT NULL,
			note         TEXT            DEFAULT NULL,
			seller_id    BIGINT UNSIGNED DEFAULT NULL,
			supervisor_id BIGINT UNSIGNED DEFAULT NULL,
			imported_at  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
			assigned_at  DATETIME        DEFAULT NULL,
			updated_at   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY phone (phone),
			KEY seller_id (seller_id),
			KEY supervisor_id (supervisor_id),
			KEY import_code (import_code),
			KEY status (status),
			KEY lead_status (lead_status)
		) $charset;" );

		dbDelta( "CREATE TABLE {$wpdb->prefix}sn_lead_statuses (
			id         INT UNSIGNED NOT NULL AUTO_INCREMENT,
			label      VARCHAR(100) NOT NULL,
			color      VARCHAR(20)  NOT NULL DEFAULT '#6b7280',
			sort_order INT          NOT NULL DEFAULT 0,
			is_active  TINYINT(1)   NOT NULL DEFAULT 1,
			destination_panel VARCHAR(50) DEFAULT NULL,
			move_to_destination TINYINT(1) NOT NULL DEFAULT 0,
			PRIMARY KEY (id)
		) $charset;" );

		dbDelta( "CREATE TABLE {$wpdb->prefix}sn_invoices (
			id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			invoice_code    VARCHAR(20)     NOT NULL,
			access_token    VARCHAR(96)     DEFAULT NULL,
			short_code      VARCHAR(32)     DEFAULT NULL,
			seller_id       BIGINT UNSIGNED NOT NULL,
			issued_by_user_id BIGINT UNSIGNED DEFAULT NULL,
			lead_id         BIGINT UNSIGNED DEFAULT NULL,
			customer_wp_id  BIGINT UNSIGNED DEFAULT NULL,
			customer_name   VARCHAR(120)    NOT NULL,
			customer_phone  VARCHAR(20)     NOT NULL,
			customer_phone_secondary VARCHAR(20) DEFAULT NULL,
			province        VARCHAR(60)     DEFAULT NULL,
			city            VARCHAR(60)     DEFAULT NULL,
			customer_address TEXT           DEFAULT NULL,
			customer_postal_code VARCHAR(20) DEFAULT NULL,
			product_id      BIGINT UNSIGNED NOT NULL,
			product_price   DECIMAL(18,2)   NOT NULL DEFAULT 0,
			catalog_discounts_snapshot LONGTEXT DEFAULT NULL,
			catalog_manager_user_id BIGINT UNSIGNED DEFAULT NULL,
			pay_method      VARCHAR(20)     DEFAULT NULL,
			payment_card_number VARCHAR(32) DEFAULT NULL,
			payment_card_owner VARCHAR(191) DEFAULT NULL,
			payment_card_sales_manager_user_id BIGINT UNSIGNED DEFAULT NULL,
			payment_card_deputy_user_id BIGINT UNSIGNED DEFAULT NULL,
			status          VARCHAR(60)     NOT NULL DEFAULT 'pending',
			invoice_status  VARCHAR(60)     DEFAULT NULL,
			payment_status  VARCHAR(60)     DEFAULT NULL,
			receipt_url     VARCHAR(500)    DEFAULT NULL,
			receipt_file    VARCHAR(500)    DEFAULT NULL,
			receipt_urls LONGTEXT DEFAULT NULL,
			receipt_source  VARCHAR(50)     DEFAULT NULL,
			payment_source  VARCHAR(50)     DEFAULT NULL,
			manual_card_from VARCHAR(4)     DEFAULT NULL,
			manual_card_to  VARCHAR(4)      DEFAULT NULL,
			manual_card_to_number VARCHAR(32) DEFAULT NULL,
			manual_amount   DECIMAL(18,2)   DEFAULT NULL,
			manual_paid_at  DATETIME        DEFAULT NULL,
			manual_paid_at_jalali VARCHAR(30) DEFAULT NULL,
			deposit_card_from_last4 VARCHAR(4) DEFAULT NULL,
			deposit_card_to_last4 VARCHAR(4) DEFAULT NULL,
			deposit_amount  DECIMAL(18,2)   DEFAULT NULL,
			deposit_jalali_datetime VARCHAR(30) DEFAULT NULL,
			paid_at         DATETIME        DEFAULT NULL,
			approved_by     BIGINT UNSIGNED DEFAULT NULL,
			approved_at     DATETIME        DEFAULT NULL,
			rejected_by     BIGINT UNSIGNED DEFAULT NULL,
			rejected_at     DATETIME        DEFAULT NULL,
			rejected_reason TEXT            DEFAULT NULL,
			financial_reviewed_by BIGINT UNSIGNED DEFAULT NULL,
			financial_reviewed_at DATETIME  DEFAULT NULL,
			financial_reject_reason TEXT    DEFAULT NULL,
			financial_rejected_at DATETIME DEFAULT NULL,
			financial_rejected_by BIGINT UNSIGNED DEFAULT NULL,
			resend_to_financial_at DATETIME DEFAULT NULL,
			recontact_requested_at DATETIME DEFAULT NULL,
			recontact_note TEXT DEFAULT NULL,
			discount_amount DECIMAL(18,2) DEFAULT NULL,
			wheel_reward_summary TEXT DEFAULT NULL,
			coupon_code VARCHAR(100) DEFAULT NULL,
			coupon_discount_amount DECIMAL(18,2) DEFAULT NULL,
			original_total DECIMAL(18,2) DEFAULT NULL,
			discount_total DECIMAL(18,2) DEFAULT NULL,
			final_total DECIMAL(18,2) DEFAULT NULL,
			payment_plan VARCHAR(20) NOT NULL DEFAULT 'full',
			payment_total_amount DECIMAL(18,2) DEFAULT NULL,
			current_due_amount DECIMAL(18,2) DEFAULT NULL,
			paid_total_amount DECIMAL(18,2) NOT NULL DEFAULT 0,
			remaining_amount DECIMAL(18,2) DEFAULT NULL,
			current_payment_stage INT UNSIGNED NOT NULL DEFAULT 1,
			payment_workflow_status VARCHAR(60) NOT NULL DEFAULT 'awaiting_payment',
			current_stage_issued_by_user_id BIGINT UNSIGNED DEFAULT NULL,
			payment_completed_at DATETIME DEFAULT NULL,
			payment_archived_at DATETIME DEFAULT NULL,
			payment_archived_by BIGINT UNSIGNED DEFAULT NULL,
			payment_archive_reason TEXT DEFAULT NULL,
			referral_item_id BIGINT UNSIGNED DEFAULT NULL,
			wc_order_id BIGINT UNSIGNED DEFAULT NULL,
			financial_return_state VARCHAR(40) DEFAULT NULL,
			returned_to_seller_at DATETIME DEFAULT NULL,
			resent_after_return_at DATETIME DEFAULT NULL,
			created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY invoice_code (invoice_code),
			KEY seller_id (seller_id),
			KEY customer_wp_id (customer_wp_id),
			KEY customer_phone (customer_phone),
			KEY referral_item_id (referral_item_id),
			KEY status (status)
		) $charset;" );

		dbDelta( "CREATE TABLE {$wpdb->prefix}sn_invoice_payment_stages (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			invoice_id BIGINT UNSIGNED NOT NULL,
			stage_no INT UNSIGNED NOT NULL DEFAULT 1,
			stage_type VARCHAR(30) NOT NULL DEFAULT 'full',
			requested_amount DECIMAL(18,2) NOT NULL DEFAULT 0,
			status VARCHAR(60) NOT NULL DEFAULT 'pending',
			pay_method VARCHAR(30) DEFAULT NULL,
			payment_source VARCHAR(60) DEFAULT NULL,
			payment_ref_id VARCHAR(191) DEFAULT NULL,
			receipt_urls LONGTEXT DEFAULT NULL,
			issued_by_user_id BIGINT UNSIGNED DEFAULT NULL,
			approved_by_user_id BIGINT UNSIGNED DEFAULT NULL,
			approved_at DATETIME DEFAULT NULL,
			paid_at DATETIME DEFAULT NULL,
			note TEXT DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY invoice_stage (invoice_id, stage_no),
			KEY invoice_id (invoice_id),
			KEY status (status)
		) $charset;" );

		dbDelta( "CREATE TABLE {$wpdb->prefix}sn_payments (
			id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			invoice_id   BIGINT UNSIGNED NOT NULL,
			payment_stage_no INT UNSIGNED NOT NULL DEFAULT 1,
			authority    VARCHAR(100)    DEFAULT NULL,
			ref_id       VARCHAR(191)    DEFAULT NULL,
			amount       DECIMAL(18,2)   NOT NULL DEFAULT 0,
			status       VARCHAR(20)     NOT NULL DEFAULT 'pending',
			pay_method   VARCHAR(30)     DEFAULT NULL,
			payment_source VARCHAR(60)   DEFAULT NULL,
			proof_data LONGTEXT DEFAULT NULL,
			submission_key VARCHAR(64) DEFAULT NULL,
			uploaded_by_type VARCHAR(20) DEFAULT NULL,
			uploaded_by_user_id BIGINT UNSIGNED DEFAULT NULL,
			created_at   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			KEY invoice_id (invoice_id),
			KEY invoice_stage_status (invoice_id, payment_stage_no, status)
		) $charset;" );

		dbDelta( "CREATE TABLE {$wpdb->prefix}sn_invoice_items (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			invoice_id BIGINT UNSIGNED NOT NULL,
			product_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
			product_name VARCHAR(255) NOT NULL,
			product_type VARCHAR(30) DEFAULT NULL,
			qty INT UNSIGNED NOT NULL DEFAULT 1,
			unit_price DECIMAL(18,2) NOT NULL DEFAULT 0,
			total_price DECIMAL(18,2) NOT NULL DEFAULT 0,
			is_free TINYINT(1) NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			KEY invoice_id (invoice_id),
			KEY product_id (product_id)
		) $charset;" );

		dbDelta( "CREATE TABLE {$wpdb->prefix}sn_invoice_wheel (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			invoice_id BIGINT UNSIGNED NOT NULL,
			customer_id BIGINT UNSIGNED DEFAULT NULL,
			reward_type VARCHAR(40) DEFAULT NULL,
			reward_value VARCHAR(120) DEFAULT NULL,
			reward_payload LONGTEXT DEFAULT NULL,
			used_discount TINYINT(1) NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			KEY invoice_id (invoice_id),
			KEY customer_id (customer_id)
		) $charset;" );

		dbDelta( "CREATE TABLE {$wpdb->prefix}sn_payment_reward_claims (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			invoice_id BIGINT UNSIGNED NOT NULL,
			invoice_code VARCHAR(40) DEFAULT NULL,
			product_id BIGINT UNSIGNED DEFAULT NULL,
			product_name VARCHAR(255) DEFAULT NULL,
			customer_name VARCHAR(120) DEFAULT NULL,
			customer_phone VARCHAR(20) DEFAULT NULL,
			reward_phone VARCHAR(20) NOT NULL,
			sms_sent TINYINT(1) NOT NULL DEFAULT 0,
			sms_status VARCHAR(40) DEFAULT NULL,
			sms_pattern VARCHAR(120) DEFAULT NULL,
			sms_template TEXT DEFAULT NULL,
			sms_error TEXT DEFAULT NULL,
			ip_address VARCHAR(64) DEFAULT NULL,
			user_agent TEXT DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY invoice_id (invoice_id),
			KEY invoice_code (invoice_code),
			KEY short_code (short_code),
			KEY reward_phone (reward_phone),
			KEY customer_phone (customer_phone),
			KEY created_at (created_at)
		) $charset;" );


		dbDelta( "CREATE TABLE {$wpdb->prefix}sn_activity_logs (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			lead_id BIGINT UNSIGNED DEFAULT NULL,
			invoice_id BIGINT UNSIGNED DEFAULT NULL,
			user_id BIGINT UNSIGNED DEFAULT NULL,
			action VARCHAR(120) NOT NULL,
			old_value LONGTEXT DEFAULT NULL,
			new_value LONGTEXT DEFAULT NULL,
			description TEXT DEFAULT NULL,
			context LONGTEXT DEFAULT NULL,
			ip_address VARCHAR(64) DEFAULT NULL,
			user_agent TEXT DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			KEY lead_id (lead_id),
			KEY invoice_id (invoice_id),
			KEY user_id (user_id),
			KEY action (action),
			KEY created_at (created_at)
		) $charset;" );

		dbDelta( "CREATE TABLE {$wpdb->prefix}sn_lead_status_history (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			lead_id BIGINT UNSIGNED NOT NULL,
			user_id BIGINT UNSIGNED DEFAULT NULL,
			old_status VARCHAR(100) DEFAULT NULL,
			new_status VARCHAR(100) DEFAULT NULL,
			note TEXT DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			KEY lead_id (lead_id),
			KEY user_id (user_id),
			KEY created_at (created_at)
		) $charset;" );

		// Migration: ستون‌های جدید
		$cols = $wpdb->get_col( "SHOW COLUMNS FROM {$wpdb->prefix}sn_leads", 0 );
		if ( ! in_array( 'lead_status', $cols, true ) ) {
			$wpdb->query( "ALTER TABLE {$wpdb->prefix}sn_leads ADD COLUMN lead_status VARCHAR(60) DEFAULT NULL AFTER status" );
		}
		if ( ! in_array( 'note', $cols, true ) ) {
			$wpdb->query( "ALTER TABLE {$wpdb->prefix}sn_leads ADD COLUMN note TEXT DEFAULT NULL AFTER lead_status" );
		}
		if ( ! in_array( 'updated_at', $cols, true ) ) {
			$wpdb->query( "ALTER TABLE {$wpdb->prefix}sn_leads ADD COLUMN updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP AFTER assigned_at" );
		}
		if ( ! in_array( 'supervisor_id', $cols, true ) ) {
			$wpdb->query( "ALTER TABLE {$wpdb->prefix}sn_leads ADD COLUMN supervisor_id BIGINT UNSIGNED DEFAULT NULL AFTER seller_id" );
		}
		if ( ! in_array( 'import_code', $cols, true ) ) {
			$wpdb->query( "ALTER TABLE {$wpdb->prefix}sn_leads ADD COLUMN import_code VARCHAR(80) DEFAULT NULL AFTER phone" );
		}
		$lead_cols = $wpdb->get_col( "SHOW COLUMNS FROM {$wpdb->prefix}sn_leads", 0 );
		if ( ! in_array( 'customer_name', $lead_cols, true ) ) {
			$wpdb->query( "ALTER TABLE {$wpdb->prefix}sn_leads ADD COLUMN customer_name VARCHAR(120) DEFAULT NULL AFTER phone" );
		}
		if ( ! in_array( 'national_id', $lead_cols, true ) ) {
			$wpdb->query( "ALTER TABLE {$wpdb->prefix}sn_leads ADD COLUMN national_id VARCHAR(20) DEFAULT NULL AFTER customer_name" );
		}
		if ( ! in_array( 'marketing_form_id', $lead_cols, true ) ) {
			$wpdb->query( "ALTER TABLE {$wpdb->prefix}sn_leads ADD COLUMN marketing_form_id VARCHAR(80) DEFAULT NULL AFTER national_id" );
		}
		if ( ! in_array( 'sales_prediction', $lead_cols, true ) ) {
			$wpdb->query( "ALTER TABLE {$wpdb->prefix}sn_leads ADD COLUMN sales_prediction VARCHAR(120) DEFAULT NULL AFTER lead_status" );
		}
		if ( ! in_array( 'destination_panel', $lead_cols, true ) ) {
			$wpdb->query( "ALTER TABLE {$wpdb->prefix}sn_leads ADD COLUMN destination_panel VARCHAR(50) DEFAULT NULL AFTER lead_status" );
		}
		if ( ! in_array( 'destination_routed_at', $lead_cols, true ) ) {
			$wpdb->query( "ALTER TABLE {$wpdb->prefix}sn_leads ADD COLUMN destination_routed_at DATETIME DEFAULT NULL AFTER destination_panel" );
		}

		if ( ! in_array( 'customer_wp_id', $wpdb->get_col( "SHOW COLUMNS FROM {$wpdb->prefix}sn_invoices", 0 ), true ) ) {
			$wpdb->query( "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN customer_wp_id BIGINT UNSIGNED DEFAULT NULL AFTER lead_id" );
		}
		$invoice_cols = $wpdb->get_col( "SHOW COLUMNS FROM {$wpdb->prefix}sn_invoices", 0 );
		// Fix existing installations: old schema had status VARCHAR(20), but financial statuses can be longer.
		$wpdb->query( "ALTER TABLE {$wpdb->prefix}sn_invoices MODIFY COLUMN status VARCHAR(60) NOT NULL DEFAULT 'pending'" );
		$invoice_migrations = [
			'invoice_status' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN invoice_status VARCHAR(60) DEFAULT NULL AFTER status",
			'access_token'   => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN access_token VARCHAR(96) DEFAULT NULL AFTER invoice_code",
			'short_code'     => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN short_code VARCHAR(32) DEFAULT NULL AFTER access_token",
			'payment_status' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN payment_status VARCHAR(60) DEFAULT NULL AFTER invoice_status",
			'receipt_urls' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN receipt_urls LONGTEXT DEFAULT NULL",
			'receipt_file'    => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN receipt_file VARCHAR(500) DEFAULT NULL AFTER receipt_url",
			'receipt_source'  => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN receipt_source VARCHAR(50) DEFAULT NULL AFTER receipt_file",
			'payment_source'   => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN payment_source VARCHAR(50) DEFAULT NULL AFTER pay_method",
			'payment_card_number' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN payment_card_number VARCHAR(32) DEFAULT NULL AFTER pay_method",
			'payment_card_owner' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN payment_card_owner VARCHAR(191) DEFAULT NULL AFTER payment_card_number",
			'payment_card_sales_manager_user_id' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN payment_card_sales_manager_user_id BIGINT UNSIGNED DEFAULT NULL AFTER payment_card_owner",
			'payment_card_deputy_user_id' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN payment_card_deputy_user_id BIGINT UNSIGNED DEFAULT NULL AFTER payment_card_sales_manager_user_id",
			'manual_card_from' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN manual_card_from VARCHAR(4) DEFAULT NULL AFTER receipt_url",
			'manual_card_to'   => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN manual_card_to VARCHAR(4) DEFAULT NULL AFTER manual_card_from",
			'manual_card_to_number' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN manual_card_to_number VARCHAR(32) DEFAULT NULL AFTER manual_card_to",
			'manual_amount'    => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN manual_amount DECIMAL(18,2) DEFAULT NULL AFTER manual_card_to_number",
			'manual_paid_at'   => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN manual_paid_at DATETIME DEFAULT NULL AFTER manual_amount",
			'manual_paid_at_jalali' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN manual_paid_at_jalali VARCHAR(30) DEFAULT NULL AFTER manual_paid_at",
			'approved_by'      => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN approved_by BIGINT UNSIGNED DEFAULT NULL AFTER paid_at",
			'approved_at'      => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN approved_at DATETIME DEFAULT NULL AFTER approved_by",
			'rejected_by'      => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN rejected_by BIGINT UNSIGNED DEFAULT NULL AFTER approved_at",
			'rejected_at'      => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN rejected_at DATETIME DEFAULT NULL AFTER rejected_by",
			'rejected_reason'  => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN rejected_reason TEXT DEFAULT NULL AFTER rejected_at",
			'deposit_card_from_last4' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN deposit_card_from_last4 VARCHAR(4) DEFAULT NULL AFTER manual_paid_at_jalali",
			'deposit_card_to_last4'   => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN deposit_card_to_last4 VARCHAR(4) DEFAULT NULL AFTER deposit_card_from_last4",
			'deposit_amount'          => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN deposit_amount DECIMAL(18,2) DEFAULT NULL AFTER deposit_card_to_last4",
			'deposit_jalali_datetime' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN deposit_jalali_datetime VARCHAR(30) DEFAULT NULL AFTER deposit_amount",
			'financial_reviewed_by'   => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN financial_reviewed_by BIGINT UNSIGNED DEFAULT NULL AFTER rejected_reason",
			'financial_reviewed_at'   => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN financial_reviewed_at DATETIME DEFAULT NULL AFTER financial_reviewed_by",
			'financial_reject_reason' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN financial_reject_reason TEXT DEFAULT NULL AFTER financial_reviewed_at",
			'coupon_code' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN coupon_code VARCHAR(100) DEFAULT NULL AFTER wheel_reward_summary",
			'coupon_discount_amount' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN coupon_discount_amount DECIMAL(18,2) DEFAULT NULL AFTER coupon_code",
			'original_total' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN original_total DECIMAL(18,2) DEFAULT NULL AFTER coupon_discount_amount",
			'discount_total' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN discount_total DECIMAL(18,2) DEFAULT NULL AFTER original_total",
			'final_total' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN final_total DECIMAL(18,2) DEFAULT NULL AFTER discount_total",
		];
		foreach ( $invoice_migrations as $col => $sql ) {
			if ( ! in_array( $col, $invoice_cols, true ) ) {
				$wpdb->query( $sql );
			}
		}

		$status_cols = $wpdb->get_col( "SHOW COLUMNS FROM {$wpdb->prefix}sn_lead_statuses", 0 );
		if ( ! in_array( 'destination_panel', $status_cols, true ) ) {
			$wpdb->query( "ALTER TABLE {$wpdb->prefix}sn_lead_statuses ADD COLUMN destination_panel VARCHAR(50) DEFAULT NULL AFTER is_active" );
		}
		if ( ! in_array( 'move_to_destination', $status_cols, true ) ) {
			$wpdb->query( "ALTER TABLE {$wpdb->prefix}sn_lead_statuses ADD COLUMN move_to_destination TINYINT(1) NOT NULL DEFAULT 0 AFTER destination_panel" );
		}

		$invoice_cols = $wpdb->get_col( "SHOW COLUMNS FROM {$wpdb->prefix}sn_invoices", 0 );
		$workflow_invoice_migrations = [
			'financial_rejected_at' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN financial_rejected_at DATETIME DEFAULT NULL AFTER financial_reject_reason",
			'financial_rejected_by' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN financial_rejected_by BIGINT UNSIGNED DEFAULT NULL AFTER financial_rejected_at",
			'resend_to_financial_at' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN resend_to_financial_at DATETIME DEFAULT NULL AFTER financial_rejected_by",
			'recontact_requested_at' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN recontact_requested_at DATETIME DEFAULT NULL AFTER resend_to_financial_at",
			'recontact_note' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN recontact_note TEXT DEFAULT NULL AFTER recontact_requested_at",
			'discount_amount' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN discount_amount DECIMAL(18,2) DEFAULT NULL AFTER product_price",
			'wheel_reward_summary' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN wheel_reward_summary TEXT DEFAULT NULL AFTER discount_amount",
			'financial_return_state' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN financial_return_state VARCHAR(40) DEFAULT NULL AFTER wc_order_id",
			'returned_to_seller_at' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN returned_to_seller_at DATETIME DEFAULT NULL AFTER financial_return_state",
			'resent_after_return_at' => "ALTER TABLE {$wpdb->prefix}sn_invoices ADD COLUMN resent_after_return_at DATETIME DEFAULT NULL AFTER returned_to_seller_at",
		];
		foreach ( $workflow_invoice_migrations as $col => $sql ) {
			if ( ! in_array( $col, $invoice_cols, true ) ) { $wpdb->query( $sql ); }
		}

		self::backfill_invoice_access_tokens();
		update_option( 'sn_db_migration_report', [
			'version' => self::DB_VERSION,
			'added_columns' => [
				'sn_leads.customer_name',
				'sn_leads.sales_prediction',
				'sn_invoices.access_token',
				'sn_invoices.coupon_code',
				'sn_invoices.coupon_discount_amount',
				'sn_invoices.original_total',
				'sn_invoices.discount_total',
				'sn_invoices.final_total',
				'sn_invoices.wc_order_id',
			],
			'finance_permissions' => 'Existing sn_financial/sn_financial_approval users received user-level finance caps once; finance roles now keep only read.',
			'updated_at' => current_time( 'mysql' ),
		], false );

		$payment_cols = $wpdb->get_col( "SHOW COLUMNS FROM {$wpdb->prefix}sn_payments", 0 );
		if ( ! in_array( 'payment_stage_no', $payment_cols, true ) ) {
			$wpdb->query( "ALTER TABLE {$wpdb->prefix}sn_payments ADD COLUMN payment_stage_no INT UNSIGNED NOT NULL DEFAULT 1 AFTER invoice_id" );
		}
		if ( ! in_array( 'pay_method', $payment_cols, true ) ) {
			$wpdb->query( "ALTER TABLE {$wpdb->prefix}sn_payments ADD COLUMN pay_method VARCHAR(30) DEFAULT NULL AFTER status" );
		}
		if ( ! in_array( 'payment_source', $payment_cols, true ) ) {
			$wpdb->query( "ALTER TABLE {$wpdb->prefix}sn_payments ADD COLUMN payment_source VARCHAR(60) DEFAULT NULL AFTER pay_method" );
		}
		if ( ! in_array( 'uploaded_by_type', $payment_cols, true ) ) {
			$wpdb->query( "ALTER TABLE {$wpdb->prefix}sn_payments ADD COLUMN uploaded_by_type VARCHAR(20) DEFAULT NULL AFTER payment_source" );
		}
		if ( ! in_array( 'uploaded_by_user_id', $payment_cols, true ) ) {
			$wpdb->query( "ALTER TABLE {$wpdb->prefix}sn_payments ADD COLUMN uploaded_by_user_id BIGINT UNSIGNED DEFAULT NULL AFTER uploaded_by_type" );
		}
		if ( ! in_array( 'updated_at', $payment_cols, true ) ) {
			$wpdb->query( "ALTER TABLE {$wpdb->prefix}sn_payments ADD COLUMN updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP AFTER created_at" );
		}
		$payment_indexes = (array) $wpdb->get_results( "SHOW INDEX FROM {$wpdb->prefix}sn_payments", ARRAY_A );
		$payment_index_names = array_unique( array_map( static function( $row ) { return $row['Key_name'] ?? ''; }, $payment_indexes ) );
		if ( ! in_array( 'invoice_stage_status', $payment_index_names, true ) ) {
			$wpdb->query( "ALTER TABLE {$wpdb->prefix}sn_payments ADD INDEX invoice_stage_status (invoice_id, payment_stage_no, status)" );
		}
		$log_cols = $wpdb->get_col( "SHOW COLUMNS FROM {$wpdb->prefix}sn_activity_logs", 0 );
		$log_migrations = [
			'old_value' => "ALTER TABLE {$wpdb->prefix}sn_activity_logs ADD COLUMN old_value LONGTEXT DEFAULT NULL AFTER action",
			'new_value' => "ALTER TABLE {$wpdb->prefix}sn_activity_logs ADD COLUMN new_value LONGTEXT DEFAULT NULL AFTER old_value",
			'user_agent' => "ALTER TABLE {$wpdb->prefix}sn_activity_logs ADD COLUMN user_agent TEXT DEFAULT NULL AFTER ip_address",
		];
		foreach ( $log_migrations as $col => $sql ) {
			if ( ! in_array( $col, $log_cols, true ) ) {
				$wpdb->query( $sql );
			}
		}

		// 1.0.24: indexes for lightweight customer action timeline queries.
		$sn_log_indexes = [
			'sn_idx_log_invoice_action_id' => 'invoice_id, action, id',
			'sn_idx_log_invoice_id'        => 'invoice_id, id',
			'sn_idx_log_action_created'    => 'action, created_at',
		];
		foreach ( $sn_log_indexes as $sn_index => $sn_cols_sql ) {
			$exists = $wpdb->get_var( $wpdb->prepare( 'SHOW INDEX FROM ' . $wpdb->prefix . 'sn_activity_logs WHERE Key_name = %s', $sn_index ) );
			if ( ! $exists ) {
				$wpdb->query( "ALTER TABLE {$wpdb->prefix}sn_activity_logs ADD INDEX {$sn_index} ({$sn_cols_sql})" );
			}
		}

		// 1.0.16 stability-lite: safe indexes for financial tabs and invoice lookup on upgraded installs.
		$sn_index_specs = [
			"{$wpdb->prefix}sn_invoices" => [
				'sn_idx_inv_status_payment' => 'status, payment_status',
				'sn_idx_inv_financial_tabs' => 'payment_status, receipt_source, payment_source',
				'sn_idx_inv_seller_status' => 'seller_id, status',
			],
			"{$wpdb->prefix}sn_payments" => [
				'sn_idx_pay_status_uploaded' => 'status, uploaded_by_type',
				'sn_idx_pay_invoice_status' => 'invoice_id, status',
			],
		];
		foreach ( $sn_index_specs as $sn_table => $sn_indexes ) {
			foreach ( $sn_indexes as $sn_index => $sn_cols_sql ) {
				$exists = $wpdb->get_var( $wpdb->prepare( 'SHOW INDEX FROM ' . $sn_table . ' WHERE Key_name = %s', $sn_index ) );
				if ( ! $exists ) {
					$wpdb->query( "ALTER TABLE {$sn_table} ADD INDEX {$sn_index} ({$sn_cols_sql})" );
				}
			}
		}

	}

	private static function backfill_invoice_access_tokens(): void {
		global $wpdb;
		$table = $wpdb->prefix . 'sn_invoices';
		if ( $wpdb->get_var( "SHOW TABLES LIKE '{$table}'" ) !== $table ) { return; }
		$cols = $wpdb->get_col( "SHOW COLUMNS FROM {$table}", 0 );
		if ( ! in_array( 'access_token', $cols, true ) ) { return; }
		$ids = $wpdb->get_col( "SELECT id FROM {$table} WHERE access_token IS NULL OR access_token='' LIMIT 1000" );
		foreach ( $ids as $id ) {
			$token = class_exists( 'SN_Helpers' ) ? SN_Helpers::generate_unique_access_token() : wp_generate_password( 48, false, false );
			$wpdb->update( $table, [ 'access_token' => $token ], [ 'id' => (int) $id ] );
		}
	}

	private static function insert_defaults() {
		global $wpdb;
		update_option( 'sn_flush_rewrite_needed', '1' );

		// وضعیت‌های پیش‌فرض
		$count = (int) $wpdb->get_var( "SELECT COUNT(*) FROM {$wpdb->prefix}sn_lead_statuses" );
		if ( $count === 0 ) {
			$defaults = [
				[ 'label' => 'جواب نداده',  'color' => '#f59e0b', 'sort_order' => 1 ],
				[ 'label' => 'تماس مجدد',  'color' => '#3b82f6', 'sort_order' => 2 ],
				[ 'label' => 'عدم خرید',    'color' => '#ef4444', 'sort_order' => 3 ],
				[ 'label' => 'پیش‌فاکتور',  'color' => '#0ea5e9', 'sort_order' => 4 ],
			];
			foreach ( $defaults as $s ) {
				$wpdb->insert( $wpdb->prefix . 'sn_lead_statuses', $s );
			}
		}

		$options = [
			'sn_zarinpal_merchant'  => '',
			'sn_zarinpal_sandbox'   => '1',
			'sn_sms_provider'       => 'faraz',
			'sn_sms_api_key'        => '',
			'sn_sms_sender'         => '',
				'sn_card_number'        => '',
				'sn_card_owner'         => '',
				'sn_card_to_card_enabled' => '1',
				'sn_sales_manager_cards' => [],
				'sn_sales_deputy_cards' => [],
			'sn_invoice_page_id'       => '',
			'sn_seller_panel_page_id'  => '',
			'sn_supervisor_panel_page_id' => '',
			'sn_supervisor_auth_page_id' => '',
			'sn_after_sales_panel_page_id' => '',
				'sn_shipping_panel_page_id' => '',
			'sn_operations_sales_manager_panel_page_id' => '',
			'sn_operations_sales_supervisor_panel_page_id' => '',
			'sn_operations_sales_expert_panel_page_id' => '',
			'sn_operations_executive_manager_panel_page_id' => '',
			'sn_operations_execution_expert_panel_page_id' => '',
			'sn_operations_customer_inactivity_days' => '3',
			'sn_sales_manager_auth_page_id' => '',
			'sn_sales_manager_panel_page_id' => '',
			'sn_financial_auth_page_id' => '',
			'sn_financial_panel_page_id' => '',
			'sn_auth_page_id'          => '',
			'sn_coupon_allow_on_sale' => '0',
			'sn_invoice_info_show_product_info' => '1',
			'sn_invoice_info_show_short_desc' => '1',
			'sn_invoice_info_show_price' => '1',
			'sn_invoice_info_show_lottery' => '1',
			'sn_invoice_info_show_coupon' => '1',
			'sn_lottery_text_template' => 'با پرداخت این فاکتور {count} شانس برای شرکت در قرعه‌کشی {company} دریافت می‌کنید.',
			'sn_recontact_popup_text' => 'اگر پیش از پرداخت فاکتور از کارشناس خود سوالی دارید، دکمه ارتباط مجدد با کارشناس را بزنید.',
			'sn_wheel_company_name' => '',
		];
		foreach ( $options as $key => $val ) {
			if ( false === get_option( $key ) ) {
				add_option( $key, $val );
			}
		}
	}

	private static function shortcode_bracketed( string $shortcode ): string {
		$shortcode = trim( $shortcode );
		if ( '' === $shortcode ) { return ''; }
		return '[' === $shortcode[0] ? $shortcode : '[' . sanitize_key( $shortcode ) . ']';
	}

	private static function elementor_shortcode_data( string $shortcode ): string {
		$shortcode = self::shortcode_bracketed( $shortcode );
		$data = [ [
			'id'       => substr( md5( 'sn_section_' . $shortcode ), 0, 8 ),
			'elType'   => 'section',
			'isInner'  => false,
			'settings' => [
				'layout'          => 'full_width',
				'content_width'   => 'full',
				'stretch_section' => 'section-stretched',
			],
			'elements' => [ [
				'id'       => substr( md5( 'sn_column_' . $shortcode ), 0, 8 ),
				'elType'   => 'column',
				'isInner'  => false,
				'settings' => [ '_column_size' => 100, '_inline_size' => null ],
				'elements' => [ [
					'id'         => substr( md5( 'sn_shortcode_' . $shortcode ), 0, 8 ),
					'elType'     => 'widget',
					'widgetType' => 'shortcode',
					'settings'   => [ 'shortcode' => $shortcode ],
					'elements'   => [],
				] ],
			] ],
		] ];
		return wp_json_encode( $data, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ) ?: '[]';
	}

	private static function elementor_data_is_simple_shortcode( $raw ): bool {
		if ( ! is_string( $raw ) || '' === trim( $raw ) ) { return true; }
		$decoded = json_decode( $raw, true );
		if ( ! is_array( $decoded ) ) { return false; }
		return false !== strpos( $raw, '"widgetType":"shortcode"' );
	}

	private static function apply_page_layout_defaults( int $page_id, string $shortcode ): void {
		if ( $page_id < 1 ) { return; }
		$shortcode = self::shortcode_bracketed( $shortcode );
		if ( '' === $shortcode ) { return; }
		update_post_meta( $page_id, '_wp_page_template', 'elementor_canvas' );
		update_post_meta( $page_id, '_sn_system_page', '1' );
		$settings = get_post_meta( $page_id, '_elementor_page_settings', true );
		$settings = is_array( $settings ) ? $settings : [];
		$settings['page_layout'] = 'elementor_canvas';
		$settings['template'] = 'elementor_canvas';
		update_post_meta( $page_id, '_elementor_page_settings', $settings );
		update_post_meta( $page_id, '_elementor_edit_mode', 'builder' );
		update_post_meta( $page_id, '_elementor_template_type', 'wp-page' );
		$existing = (string) get_post_meta( $page_id, '_elementor_data', true );
		if ( self::elementor_data_is_simple_shortcode( $existing ) ) {
			update_post_meta( $page_id, '_elementor_data', self::elementor_shortcode_data( $shortcode ) );
		}
	}

	public static function create_required_pages(): array {
		$pages = self::required_pages();
		$result = [];
		foreach ( $pages as $option_key => $page ) {
			$duplicates = [];
			$page_id = self::resolve_required_page_id( $option_key, $page, $duplicates );
			$post = $page_id ? get_post( $page_id ) : null;
			if ( ! $post ) {
				$new_id = wp_insert_post( [
					'post_title'     => $page['title'],
					'post_name'      => $page['slug'],
					'post_content'   => $page['shortcode'],
					'post_status'    => 'publish',
					'post_type'      => 'page',
					'comment_status' => 'closed',
					'ping_status'    => 'closed',
				], true );
				if ( is_wp_error( $new_id ) ) { $result[ $option_key ] = [ 'created' => false, 'error' => $new_id->get_error_message() ]; continue; }
				$page_id = (int) $new_id;
				$post = get_post( $page_id );
				update_post_meta( $page_id, '_sn_system_page', '1' );
			}
			if ( $post && 'publish' !== $post->post_status ) {
				wp_update_post( [ 'ID' => $page_id, 'post_status' => 'publish' ] );
				$post = get_post( $page_id );
			}
			if ( $post ) {
				self::ensure_page_shortcode_content( $post, $page['shortcode'] );
				$post = get_post( $page_id );
			}
			update_option( $option_key, $page_id );
			if ( $post && trim( (string) $post->post_content ) === $page['shortcode'] ) {
				update_post_meta( $page_id, '_sn_system_page', '1' );
			}
			self::apply_page_layout_defaults( (int) $page_id, (string) $page['shortcode'] );
			$result[ $option_key ] = [
				'created'    => true,
				'page_id'    => $page_id,
				'title'      => $page['title'],
				'slug'       => $page['slug'],
				'shortcode'  => $page['shortcode'],
				'duplicates' => $duplicates,
			];
		}
		update_option( 'sn_pages_initialized', current_time( 'mysql' ) );
		update_option( 'sn_flush_rewrite_needed', '1' );
		self::store_duplicate_report( $result );
		return $result;
	}

	public static function required_pages(): array {
		return [
			'sn_login_page_id' => [ 'title' => 'ورود یکپارچه CRM', 'slug' => 'crm-login', 'shortcode' => '[sn_login]' ],
			'sn_my_password_page_id' => [ 'title' => 'رمز من', 'slug' => 'crm-my-password', 'shortcode' => '[sn_my_password]' ],
			'sn_auth_page_id' => [ 'title' => 'ورود فروشنده', 'slug' => 'seller-login', 'shortcode' => '[sn_auth]' ],
			'sn_supervisor_auth_page_id' => [ 'title' => 'ورود سرپرست', 'slug' => 'supervisor-login', 'shortcode' => '[sn_supervisor_auth]' ],
			'sn_seller_panel_page_id' => [ 'title' => 'پنل فروشنده', 'slug' => 'seller-panel', 'shortcode' => '[sn_seller_panel]' ],
			'sn_supervisor_panel_page_id' => [ 'title' => 'پنل سرپرست', 'slug' => 'supervisor-panel', 'shortcode' => '[sn_supervisor_panel]' ],
			'sn_after_sales_panel_page_id' => [ 'title' => 'پنل خدمات پس از فروش', 'slug' => 'after-sales-panel', 'shortcode' => '[sn_after_sales_panel]' ],
			'sn_sales_manager_auth_page_id' => [ 'title' => 'ورود مدیر فروش', 'slug' => 'sales-manager-login', 'shortcode' => '[sn_sales_manager_auth]' ],
			'sn_sales_manager_panel_page_id' => [ 'title' => 'پنل مدیر فروش', 'slug' => 'sales-manager-panel', 'shortcode' => '[sn_sales_manager_panel]' ],
			'sn_financial_auth_page_id' => [ 'title' => 'ورود تایید مالی', 'slug' => 'financial-login', 'shortcode' => '[sn_financial_auth]' ],
			'sn_financial_panel_page_id' => [ 'title' => 'پنل تایید مالی', 'slug' => 'financial-approval', 'shortcode' => '[sn_financial_panel]' ],
			'sn_invoice_page_id' => [ 'title' => 'فاکتور', 'slug' => 'invoice', 'shortcode' => '[sn_invoice_page]' ],
			'sn_dot_customer_page_id' => [ 'title' => 'تایید اعتبارسنجی', 'slug' => 'credit-assessment', 'shortcode' => '[sn_dot_customer_flow]' ],
			'sn_dot_converter_panel_page_id' => [ 'title' => 'پنل تبدیل‌کننده', 'slug' => 'crm-converter', 'shortcode' => '[sn_dot_converter_panel]' ],
			'sn_campaign_partner_page_id' => [ 'title' => 'پرتال پارتنر کمپین', 'slug' => 'campaign-partner', 'shortcode' => '[sn_campaign_partner_portal]' ],
			'sn_biavin_customer_portal_page_id' => [ 'title' => 'پروفایل مشتری بیاوین', 'slug' => 'biawin', 'shortcode' => '[sn_biavin_customer_portal]' ],
				'sn_shipping_panel_page_id' => [ 'title' => 'پنل ارسال', 'slug' => 'crm-shipping', 'shortcode' => '[sn_shipping_panel]' ],
			'sn_operations_sales_manager_panel_page_id' => [ 'title' => 'پنل مدیر فروش عملیات', 'slug' => 'crm-operations-sales-manager', 'shortcode' => '[sn_operations_sales_manager_panel]' ],
			'sn_operations_sales_supervisor_panel_page_id' => [ 'title' => 'پنل سرپرست فروش عملیات', 'slug' => 'crm-operations-sales-supervisor', 'shortcode' => '[sn_operations_sales_supervisor_panel]' ],
			'sn_operations_sales_expert_panel_page_id' => [ 'title' => 'پنل کارشناس فروش عملیات', 'slug' => 'crm-operations-sales-expert', 'shortcode' => '[sn_operations_sales_expert_panel]' ],
			'sn_operations_executive_manager_panel_page_id' => [ 'title' => 'پنل مدیر اجرایی عملیات', 'slug' => 'crm-operations-executive-manager', 'shortcode' => '[sn_operations_executive_manager_panel]' ],
			'sn_operations_execution_expert_panel_page_id' => [ 'title' => 'پنل کارشناس اجرایی عملیات', 'slug' => 'crm-operations-execution-expert', 'shortcode' => '[sn_operations_execution_expert_panel]' ],
		];
	}

	private static function resolve_required_page_id( string $option_key, array $page, array &$duplicates ): int {
		$valid_statuses = [ 'publish', 'private', 'draft' ];
		$option_id = absint( get_option( $option_key, 0 ) );
		$post = $option_id ? get_post( $option_id ) : null;
		if ( $post && 'page' === $post->post_type && in_array( $post->post_status, $valid_statuses, true ) && self::content_has_shortcode( (string) $post->post_content, $page['shortcode'] ) ) {
			$duplicates = self::find_duplicate_pages( $page, $option_id );
			return $option_id;
		}

		$by_slug = get_page_by_path( $page['slug'], OBJECT, 'page' );
		if ( $by_slug && 'trash' !== $by_slug->post_status ) {
			$duplicates = self::find_duplicate_pages( $page, (int) $by_slug->ID );
			return (int) $by_slug->ID;
		}

		$by_shortcode = self::find_pages_by_shortcode( $page['shortcode'] );
		if ( ! empty( $by_shortcode ) ) {
			$chosen = (int) $by_shortcode[0]->ID;
			$duplicates = self::find_duplicate_pages( $page, $chosen );
			return $chosen;
		}

		$duplicates = self::find_duplicate_pages( $page, 0 );
		return 0;
	}

	private static function ensure_page_shortcode_content( WP_Post $post, string $shortcode ): void {
		$content = trim( (string) $post->post_content );
		if ( $content === $shortcode ) {
			return;
		}
		$has_shortcode = self::content_has_shortcode( $content, $shortcode );
		$wrapped_in_code = (bool) preg_match( '/<(code|pre)[^>]*>.*' . preg_quote( $shortcode, '/' ) . '.*<\/\1>/is', $content );
		$is_system_page = (string) get_post_meta( $post->ID, '_sn_system_page', true ) === '1';
		$is_elementor = (string) get_post_meta( $post->ID, '_elementor_edit_mode', true ) !== '';

		if ( ! $is_elementor && ( $content === '' || $wrapped_in_code || $is_system_page ) ) {
			wp_update_post( [
				'ID'           => $post->ID,
				'post_content' => $shortcode,
			] );
			return;
		}

		if ( ! $has_shortcode ) {
			wp_update_post( [
				'ID'           => $post->ID,
				'post_content' => $content . "\n\n" . $shortcode,
			] );
		}
	}

	private static function content_has_shortcode( string $content, string $shortcode ): bool {
		$tag = trim( $shortcode, '[]' );
		return has_shortcode( $content, $tag ) || false !== strpos( $content, $shortcode );
	}

	private static function find_pages_by_shortcode( string $shortcode ): array {
		global $wpdb;
		$like = '%' . $wpdb->esc_like( $shortcode ) . '%';
		$ids = $wpdb->get_col( $wpdb->prepare(
			"SELECT ID FROM {$wpdb->posts} WHERE post_type='page' AND post_status IN ('publish','private','draft') AND post_content LIKE %s ORDER BY post_date ASC LIMIT 20",
			$like
		) );
		return array_values( array_filter( array_map( 'get_post', array_map( 'absint', $ids ?: [] ) ) ) );
	}

	private static function find_duplicate_pages( array $page, int $chosen_id ): array {
		$ids = [];
		$slug_page = get_page_by_path( $page['slug'], OBJECT, 'page' );
		if ( $slug_page ) {
			$ids[] = (int) $slug_page->ID;
		}
		foreach ( self::find_pages_by_shortcode( $page['shortcode'] ) as $post ) {
			$ids[] = (int) $post->ID;
		}
		$ids = array_values( array_unique( array_filter( $ids, static fn( $id ) => $id && $id !== $chosen_id ) ) );
		$duplicates = [];
		foreach ( $ids as $id ) {
			$p = get_post( $id );
			if ( ! $p || 'page' !== $p->post_type || 'trash' === $p->post_status ) {
				continue;
			}
			$duplicates[] = [
				'id'     => $id,
				'title'  => get_the_title( $id ),
				'slug'   => $p->post_name,
				'status' => $p->post_status,
			];
		}
		return $duplicates;
	}

	private static function store_duplicate_report( array $result ): void {
		$report = [];
		foreach ( $result as $option_key => $item ) {
			if ( empty( $item['duplicates'] ) ) {
				continue;
			}
			$report[ $option_key ] = [
				'selected_page_id' => (int) ( $item['page_id'] ?? 0 ),
				'title'            => (string) ( $item['title'] ?? '' ),
				'duplicates'       => $item['duplicates'],
			];
		}
		update_option( 'sn_page_duplicate_report', $report, false );
		// Keep the report in the database, but do not spam error_log on every runtime repair.
		// The report is visible through the integration/tools UI when needed.
	}

}
