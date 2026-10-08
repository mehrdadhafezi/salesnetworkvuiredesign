<?php
if ( ! defined( 'ABSPATH' ) ) { exit; }

/**
 * Customer card activation and Operations workflow.
 *
 * This module is intentionally additive. It keeps the historic project item
 * status intact and stores every new Operations decision in its own table.
 */
final class SN_Operations_Flow {
	private const DB_VERSION = '2026-09-22-operations-contact-status-v8';
	private const CRON_HOOK = 'sn_operations_route_inactive_cards';
	private const FOLLOWUP_HOOK = 'sn_operations_followup_reminder';
	private static ?self $instance = null;
	private bool $hooks_registered = false;
	private array $columns_cache = [];
	private array $sales_filter_options = [];
	private array $sales_filter_values = [];
	private array $sales_filter_people = [];

	public static function instance(): self {
		if ( null === self::$instance ) { self::$instance = new self(); }
		return self::$instance;
	}

	private function __construct() {}

	public static function activate(): void {
		$self = self::instance();
		$self->install_schema();
		$self->ensure_roles_and_hr();
		if ( ! wp_next_scheduled( self::CRON_HOOK ) ) { wp_schedule_event( time() + 300, 'hourly', self::CRON_HOOK ); }
	}

	public static function deactivate(): void {
		$timestamp = wp_next_scheduled( self::CRON_HOOK );
		if ( $timestamp ) { wp_unschedule_event( $timestamp, self::CRON_HOOK ); }
		wp_unschedule_hook( self::FOLLOWUP_HOOK );
	}

	public function register_hooks(): void {
		if ( $this->hooks_registered ) { return; }
		$this->hooks_registered = true;
		add_action( 'init', [ $this, 'maybe_upgrade' ], 9 );
		add_action( 'admin_init', [ $this, 'maybe_upgrade' ], 9 );
		add_action( 'init', [ $this, 'register_shortcodes' ], 3 );
		add_action( 'wp_enqueue_scripts', [ $this, 'enqueue_assets' ], 32 );
		add_action( 'sn_project_membership_activated', [ $this, 'seed_membership_cards' ], 20, 3 );
		add_action( 'sn_invoice_paid', [ $this, 'on_invoice_paid' ], 30, 2 );
		add_action( 'sn_invoice_payment_stage_approved', [ $this, 'on_invoice_payment_stage_approved' ], 30, 3 );
		add_action( 'sn_invoice_financial_rejected', [ $this, 'on_invoice_reversed' ], 30, 3 );
		add_action( 'sn_invoice_financial_reopened', [ $this, 'on_invoice_reversed' ], 30, 3 );
		add_action( self::CRON_HOOK, [ $this, 'route_inactive_cards' ] );
		add_action( 'admin_post_sn_operations_customer_action', [ $this, 'handle_customer_action' ] );
		add_action( 'admin_post_sn_operations_staff_action', [ $this, 'handle_staff_action' ] );
	}

	public function register_shortcodes(): void {
		add_shortcode( 'sn_operations_sales_manager_panel', [ $this, 'render_sales_manager_panel' ] );
		add_shortcode( 'sn_operations_sales_supervisor_panel', [ $this, 'render_sales_supervisor_panel' ] );
		add_shortcode( 'sn_operations_sales_expert_panel', [ $this, 'render_sales_expert_panel' ] );
		add_shortcode( 'sn_operations_executive_manager_panel', [ $this, 'render_executive_manager_panel' ] );
	}

	private function table(): string { global $wpdb; return $wpdb->prefix . 'sn_project_operations'; }
	private function table_exists( string $table ): bool { global $wpdb; return (string) $wpdb->get_var( $wpdb->prepare( 'SHOW TABLES LIKE %s', $table ) ) === $table; }
	private function columns( string $table ): array {
		global $wpdb;
		if ( ! isset( $this->columns_cache[ $table ] ) ) { $this->columns_cache[ $table ] = (array) $wpdb->get_col( "SHOW COLUMNS FROM {$table}", 0 ); }
		return $this->columns_cache[ $table ];
	}
	private function filter_columns( string $table, array $data ): array { return array_intersect_key( $data, array_flip( $this->columns( $table ) ) ); }

	public function maybe_upgrade(): void {
		if ( (string) get_option( 'sn_operations_db_version', '' ) !== self::DB_VERSION ) { $this->install_schema(); }
		if ( (string) get_option( 'sn_operations_hr_version', '' ) !== self::DB_VERSION || ! $this->hr_contract_is_complete() ) { $this->ensure_roles_and_hr(); }
		if ( (string) get_option( 'sn_operations_followup_sms_disabled_v1', '' ) !== '1' ) {
			wp_unschedule_hook( self::FOLLOWUP_HOOK );
			update_option( 'sn_operations_followup_sms_disabled_v1', '1', false );
		}
		if ( ! wp_next_scheduled( self::CRON_HOOK ) ) { wp_schedule_event( time() + 300, 'hourly', self::CRON_HOOK ); }
		if ( ! get_transient( 'sn_operations_due_probe' ) ) {
			set_transient( 'sn_operations_due_probe', '1', 15 * MINUTE_IN_SECONDS );
			$this->route_inactive_cards( 20 );
			$this->reconcile_paid_upgrades( 20 );
		}
	}

	private function install_schema(): void {
		global $wpdb;
		require_once ABSPATH . 'wp-admin/includes/upgrade.php';
		$table = $this->table(); $charset = $wpdb->get_charset_collate();
		dbDelta( "CREATE TABLE {$table} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			membership_item_id BIGINT UNSIGNED NOT NULL,
			customer_choice VARCHAR(30) DEFAULT NULL,
			customer_choice_at DATETIME DEFAULT NULL,
			decision_due_at DATETIME DEFAULT NULL,
			auto_route_eligible TINYINT(1) NOT NULL DEFAULT 0,
			customer_self_activation_allowed TINYINT(1) NOT NULL DEFAULT 1,
			source_sales_manager_user_id BIGINT UNSIGNED DEFAULT NULL,
			operations_sales_manager_user_id BIGINT UNSIGNED DEFAULT NULL,
			sales_supervisor_user_id BIGINT UNSIGNED DEFAULT NULL,
			routing_source VARCHAR(30) NOT NULL DEFAULT 'legacy',
			activation_mode VARCHAR(20) DEFAULT NULL,
			base_credit DECIMAL(18,2) NOT NULL DEFAULT 0,
			current_credit DECIMAL(18,2) NOT NULL DEFAULT 0,
			upgrade_target_product_id BIGINT UNSIGNED DEFAULT NULL,
			upgrade_target_name VARCHAR(255) DEFAULT NULL,
			upgrade_amount DECIMAL(18,2) NOT NULL DEFAULT 0,
			upgrade_invoice_id BIGINT UNSIGNED DEFAULT NULL,
			upgrade_action_id BIGINT UNSIGNED DEFAULT NULL,
			payment_state VARCHAR(30) DEFAULT NULL,
			stage VARCHAR(40) NOT NULL DEFAULT 'awaiting_customer',
			sales_expert_user_id BIGINT UNSIGNED DEFAULT NULL,
			assigned_by_user_id BIGINT UNSIGNED DEFAULT NULL,
			assigned_at DATETIME DEFAULT NULL,
			supervisor_assigned_at DATETIME DEFAULT NULL,
			no_answer_count INT UNSIGNED NOT NULL DEFAULT 0,
			contact_status VARCHAR(30) DEFAULT NULL,
			last_contact_at DATETIME DEFAULT NULL,
			follow_up_at DATETIME DEFAULT NULL,
			follow_up_reminded_at DATETIME DEFAULT NULL,
			follow_up_reminder_attempts INT UNSIGNED NOT NULL DEFAULT 0,
			last_note TEXT DEFAULT NULL,
			cancel_reason TEXT DEFAULT NULL,
			cancelled_at DATETIME DEFAULT NULL,
			cancelled_by_user_id BIGINT UNSIGNED DEFAULT NULL,
			routed_at DATETIME DEFAULT NULL,
			completed_at DATETIME DEFAULT NULL,
			completed_by_user_id BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY membership_item_id (membership_item_id),
			KEY stage (stage),
			KEY operations_sales_manager_user_id (operations_sales_manager_user_id),
			KEY sales_supervisor_user_id (sales_supervisor_user_id),
			KEY source_sales_manager_user_id (source_sales_manager_user_id),
			KEY sales_expert_user_id (sales_expert_user_id),
			KEY contact_status (contact_status),
			KEY decision_due_at (decision_due_at),
			KEY follow_up_at (follow_up_at),
			KEY upgrade_invoice_id (upgrade_invoice_id)
		) {$charset};" );
		if ( $this->table_exists( $table ) ) { update_option( 'sn_operations_db_version', self::DB_VERSION, false ); }
	}

	private function ensure_roles_and_hr(): void {
		$roles = [
			'sn_operations_sales_manager' => [ 'مدیر فروش عملیات', 'sn_manage_operations_sales' ],
			'sn_operations_sales_supervisor' => [ 'سرپرست فروش عملیات', 'sn_supervise_operations_sales' ],
			'sn_operations_sales_expert' => [ 'کارشناس فروش عملیات', 'sn_work_operations_sales' ],
			'sn_operations_executive_manager' => [ 'مدیر اجرایی عملیات', 'sn_manage_operations_execution' ],
			'sn_operations_execution_expert' => [ 'کارشناس اجرایی عملیات', 'sn_work_operations_execution' ],
		];
		foreach ( $roles as $key => $row ) {
			if ( ! get_role( $key ) ) { add_role( $key, $row[0], [ 'read' => true, $row[1] => true ] ); }
			$role = get_role( $key ); if ( ! $role ) { continue; }
			$role->add_cap( 'read' ); $role->add_cap( $row[1] );
			foreach ( [ 'edit_posts','delete_posts','publish_posts','upload_files','edit_pages','delete_pages','manage_options','list_users','create_users','edit_users','delete_users' ] as $cap ) { $role->remove_cap( $cap ); }
		}
		global $wpdb; $positions = $wpdb->prefix . 'sn_hr_positions'; $mappings = $wpdb->prefix . 'sn_hr_position_role_mappings';
		if ( ! $this->table_exists( $positions ) || ! $this->table_exists( $mappings ) ) { return; }
		$defs = [
			'operations_sales_manager' => [ 'مدیر فروش عملیات', 'operations_sales_manager', 94, 'sn_operations_sales_manager' ],
			'operations_sales_supervisor' => [ 'سرپرست فروش عملیات', 'operations_sales_supervisor', 95, 'sn_operations_sales_supervisor' ],
			'operations_sales_expert' => [ 'کارشناس فروش عملیات', 'operations_sales_expert', 96, 'sn_operations_sales_expert' ],
			'operations_executive_manager' => [ 'مدیر اجرایی عملیات', 'operations_executive_manager', 97, 'sn_operations_executive_manager' ],
			'operations_execution_expert' => [ 'کارشناس اجرایی عملیات', 'operations_execution_expert', 98, 'sn_operations_execution_expert' ],
		];
		foreach ( $defs as $slug => $def ) {
			$position_id = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$positions} WHERE slug=%s LIMIT 1", $slug ) );
			if ( ! $position_id ) {
				$data = $this->filter_columns( $positions, [ 'slug'=>$slug, 'label'=>$def[0], 'panel_key'=>$def[1], 'sort_order'=>$def[2], 'is_active'=>1, 'is_system'=>1 ] );
				$wpdb->insert( $positions, $data ); $position_id = (int) $wpdb->insert_id;
			}
			$mapping_id = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$mappings} WHERE legacy_role=%s LIMIT 1", $def[3] ) );
			if ( $position_id && ! $mapping_id ) {
				$wpdb->insert( $mappings, $this->filter_columns( $mappings, [ 'legacy_role'=>$def[3], 'position_id'=>$position_id, 'is_active'=>1 ] ) );
			} elseif ( $position_id && $mapping_id ) {
				// Operations roles are system contracts. Repair only their mapping row;
				// profiles, assignments and historical ownership remain untouched.
				$wpdb->update( $mappings, $this->filter_columns( $mappings, [ 'position_id'=>$position_id, 'is_active'=>1 ] ), [ 'id'=>$mapping_id ] );
			}
		}
		update_option( 'sn_operations_hr_version', self::DB_VERSION, false );
	}

	private function hr_contract_is_complete(): bool {
		foreach ( [
			'sn_operations_sales_manager',
			'sn_operations_sales_supervisor',
			'sn_operations_sales_expert',
			'sn_operations_executive_manager',
			'sn_operations_execution_expert',
		] as $role_key ) {
			if ( ! get_role( $role_key ) ) { return false; }
		}
		global $wpdb;
		$positions = $wpdb->prefix . 'sn_hr_positions';
		$mappings  = $wpdb->prefix . 'sn_hr_position_role_mappings';
		if ( ! $this->table_exists( $positions ) || ! $this->table_exists( $mappings ) ) { return false; }
		$required = [
			'sn_operations_sales_manager' => 'operations_sales_manager',
			'sn_operations_sales_supervisor' => 'operations_sales_supervisor',
			'sn_operations_sales_expert' => 'operations_sales_expert',
			'sn_operations_executive_manager' => 'operations_executive_manager',
			'sn_operations_execution_expert' => 'operations_execution_expert',
		];
		foreach ( $required as $role_key => $position_slug ) {
			$mapped_slug = (string) $wpdb->get_var( $wpdb->prepare(
				"SELECT p.slug FROM {$mappings} m INNER JOIN {$positions} p ON p.id=m.position_id WHERE m.legacy_role=%s LIMIT 1",
				$role_key
			) );
			if ( $mapped_slug !== $position_slug ) { return false; }
		}
		return true;
	}

	private function inactivity_days(): int { return min( 365, max( 0, absint( get_option( 'sn_operations_customer_inactivity_days', 3 ) ) ) ); }

	/** Build a database-local deadline without mixing UTC timestamps with the site timezone. */
	private function inactivity_due_at( int $days ): string {
		$days = min( 365, max( 0, $days ) );
		return SN_Helpers::site_mysql_from_timestamp( time() + ( $days * DAY_IN_SECONDS ) );
	}

	/** Preserve the wait period which was snapshotted on the operation itself. */
	private function operation_wait_days( object $op ): int {
		$created = trim( (string) ( $op->created_at ?? '' ) );
		$due = trim( (string) ( $op->decision_due_at ?? '' ) );
		if ( $created === '' || $due === '' ) { return $this->inactivity_days(); }
		try {
			$timezone = wp_timezone();
			$created_at = new DateTimeImmutable( $created, $timezone );
			$due_at = new DateTimeImmutable( $due, $timezone );
			return min( 365, max( 0, (int) round( ( $due_at->getTimestamp() - $created_at->getTimestamp() ) / DAY_IN_SECONDS ) ) );
		} catch ( Throwable $error ) {
			return $this->inactivity_days();
		}
	}

	/** Resolve the destination for a future card without changing any older card. */
	private function resolve_operations_manager( int $content_product_id, int $source_sales_manager_user_id ): array {
		global $wpdb;
		$supervisor_routes = $wpdb->prefix . 'sn_project_operations_supervisor_sales_routes';
		$routes = $wpdb->prefix . 'sn_project_operations_sales_routes';
		$defaults = $wpdb->prefix . 'sn_project_operations_manager_products';
		if ( $content_product_id > 0 && $source_sales_manager_user_id > 0 && $this->table_exists( $supervisor_routes ) ) {
			$route = $wpdb->get_row( $wpdb->prepare(
				"SELECT operations_manager_user_id,operations_supervisor_user_id FROM {$supervisor_routes} WHERE content_product_id=%d AND sales_manager_user_id=%d LIMIT 1",
				$content_product_id,
				$source_sales_manager_user_id
			) );
			if ( $route && (int) $route->operations_manager_user_id > 0 && (int) $route->operations_supervisor_user_id > 0 ) {
				return [ 'manager_id' => (int) $route->operations_manager_user_id, 'supervisor_id' => (int) $route->operations_supervisor_user_id, 'source' => 'sales_manager_supervisor_route' ];
			}
		}
		if ( $content_product_id > 0 && $source_sales_manager_user_id > 0 && $this->table_exists( $routes ) ) {
			$manager_id = (int) $wpdb->get_var( $wpdb->prepare(
				"SELECT operations_manager_user_id FROM {$routes} WHERE content_product_id=%d AND sales_manager_user_id=%d LIMIT 1",
				$content_product_id,
				$source_sales_manager_user_id
			) );
			if ( $manager_id > 0 ) { return [ 'manager_id' => $manager_id, 'supervisor_id' => 0, 'source' => 'sales_manager_route' ]; }
		}
		if ( $content_product_id > 0 && $this->table_exists( $defaults ) ) {
			$manager_id = (int) $wpdb->get_var( $wpdb->prepare(
				"SELECT operations_manager_user_id FROM {$defaults} WHERE content_product_id=%d LIMIT 1",
				$content_product_id
			) );
			if ( $manager_id > 0 ) { return [ 'manager_id' => $manager_id, 'supervisor_id' => 0, 'source' => 'card_default' ]; }
		}
		return [ 'manager_id' => 0, 'supervisor_id' => 0, 'source' => 'unconfigured' ];
	}

	private function product_self_activation_allowed( int $product_id ): bool {
		if ( $product_id < 1 ) { return true; }
		if ( class_exists( 'SN_Seller_Flow' ) && SN_Seller_Flow::instance()->product_type( $product_id ) !== 'product_star' ) { return true; }
		$value = (string) get_post_meta( $product_id, '_sn_operations_self_activation_allowed', true );
		return $value === '' || $value === '1';
	}

	private function customer_upgrade_options_visible( int $product_id ): bool {
		// Products configured before this switch was added keep their existing customer view.
		return $product_id < 1 || (string) get_post_meta( $product_id, '_sn_customer_upgrade_options_visible', true ) !== '0';
	}

	/** Return true HR direct reports without the administrator-wide picker shortcut. */
	private function exact_direct_reports( int $parent_user_id, string $child_position ): array {
		global $wpdb;
		$profiles=$wpdb->prefix.'sn_hr_profiles';$positions=$wpdb->prefix.'sn_hr_positions';$assignments=$wpdb->prefix.'sn_hr_assignments';
		if(!$this->table_exists($profiles)||!$this->table_exists($positions)||!$this->table_exists($assignments)){return [];}
		$ids=$wpdb->get_col($wpdb->prepare("SELECT child.user_id FROM {$assignments} a INNER JOIN {$profiles} parent ON parent.id=a.parent_profile_id INNER JOIN {$profiles} child ON child.id=a.child_profile_id INNER JOIN {$positions} pos ON pos.id=child.position_id WHERE parent.user_id=%d AND a.relationship_type='reports_to' AND a.is_current=1 AND COALESCE(child.is_active,1)=1 AND pos.slug=%s ORDER BY child.user_id",$parent_user_id,$child_position))?:[];
		return array_values(array_unique(array_map('intval',$ids)));
	}

	/** Immediately route cards whose product forbids customer self activation. */
	private function route_forced_sales_expert( object $op ): bool {
		if ( (string) ( $op->stage ?? '' ) !== 'awaiting_customer' ) { return false; }
		$op = $this->hydrate_operation_route( $op );
		$manager_id = (int) ( $op->operations_sales_manager_user_id ?? 0 );
		$supervisor_id = (int) ( $op->sales_supervisor_user_id ?? 0 );
		$expert_id = 0;
		$routing_source = (string) ( $op->routing_source ?? 'legacy' );

		/*
		 * An explicit Sales Manager -> Operations Sales Supervisor route is a
		 * hard hand-off boundary. Never skip the supervisor merely because that
		 * supervisor currently has exactly one direct expert. The supervisor must
		 * see the case in their own queue and choose/claim the assignee manually.
		 *
		 * For older/default manager-only routes we keep the previous convenience:
		 * if HR has exactly one supervisor and exactly one expert below it, the
		 * forced product route may still land directly on that expert.
		 */
		$explicit_supervisor_route = $supervisor_id > 0 && $routing_source === 'sales_manager_supervisor_route';
		if ( ! $explicit_supervisor_route && $supervisor_id > 0 ) {
			$experts = $this->exact_direct_reports( $supervisor_id, 'operations_sales_expert' );
			if ( count( $experts ) === 1 ) { $expert_id = (int) $experts[0]; }
		}
		if ( $supervisor_id < 1 && $manager_id > 0 ) {
			$supervisors = $this->exact_direct_reports( $manager_id, 'operations_sales_supervisor' );
			if ( count( $supervisors ) === 1 ) {
				$supervisor_id = (int) $supervisors[0];
				$experts = $this->exact_direct_reports( $supervisor_id, 'operations_sales_expert' );
				if ( count( $experts ) === 1 ) { $expert_id = (int) $experts[0]; }
			}
		}
		$stage = $expert_id > 0 ? 'sales_expert' : ( $supervisor_id > 0 ? 'sales_supervisor' : 'sales_manager_queue' );
		$now = current_time( 'mysql' ); global $wpdb; $table = $this->table();
		$data = [ 'customer_choice'=>'product_forced_expert', 'customer_choice_at'=>$now, 'auto_route_eligible'=>0, 'stage'=>$stage, 'routed_at'=>$now, 'updated_at'=>$now ];
		if ( $supervisor_id > 0 ) { $data['sales_supervisor_user_id']=$supervisor_id; $data['supervisor_assigned_at']=$now; }
		if ( $expert_id > 0 ) { $data['sales_expert_user_id']=$expert_id; $data['assigned_by_user_id']=$manager_id?:null; $data['assigned_at']=$now; }
		$updated = $wpdb->update( $table, $data, [ 'id'=>(int)$op->id, 'stage'=>'awaiting_customer' ] );
		if ( $updated ) {
			$this->event( (int)$op->membership_item_id, 'operations_product_forced_sales_route', [
				'destination_stage'=>$stage,
				'manager_user_id'=>$manager_id,
				'supervisor_user_id'=>$supervisor_id,
				'expert_user_id'=>$expert_id,
				'explicit_supervisor_handoff'=>$explicit_supervisor_route ? 1 : 0,
			], 0 );
			return true;
		}
		return false;
	}

	private function hydrate_operation_route( object $op ): object {
		if ( (int) ( $op->operations_sales_manager_user_id ?? 0 ) > 0 ) { return $op; }
		$ctx = $this->item_context( (int) $op->membership_item_id );
		if ( ! $ctx ) { return $op; }
		$route = $this->resolve_operations_manager( (int) $ctx->content_product_id, (int) ( $ctx->source_sales_manager_user_id ?? 0 ) );
		if ( (int) $route['manager_id'] < 1 ) { return $op; }
		global $wpdb; $table = $this->table();
		$wpdb->query( $wpdb->prepare(
			"UPDATE {$table} SET source_sales_manager_user_id=NULLIF(%d,0),operations_sales_manager_user_id=%d,sales_supervisor_user_id=NULLIF(%d,0),routing_source=%s,updated_at=%s WHERE id=%d AND operations_sales_manager_user_id IS NULL",
			(int) ( $ctx->source_sales_manager_user_id ?? 0 ),
			(int) $route['manager_id'],
			(int) ( $route['supervisor_id'] ?? 0 ),
			(string) $route['source'],
			current_time( 'mysql' ),
			(int) $op->id
		) );
		return $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$table} WHERE id=%d LIMIT 1", (int) $op->id ) ) ?: $op;
	}

	public function seed_membership_cards( int $membership_id, int $invoice_id = 0, $invoice = null ): void {
		global $wpdb; $items = $wpdb->prefix . 'sn_project_membership_items'; $table = $this->table();
		if ( $membership_id < 1 || ! $this->table_exists( $items ) || ! $this->table_exists( $table ) ) { return; }
		$now = current_time( 'mysql' );
		$inactivity_days = $this->inactivity_days();
		$due = $this->inactivity_due_at( $inactivity_days );
		$memberships = $wpdb->prefix . 'sn_project_memberships';
		$rows = $wpdb->get_results( $wpdb->prepare(
			"SELECT mi.id,mi.content_product_id,COALESCE(mi.base_credit_snapshot,0) base_credit,COALESCE(m.source_sales_manager_user_id,0) source_sales_manager_user_id FROM {$items} mi INNER JOIN {$memberships} m ON m.id=mi.membership_id WHERE mi.membership_id=%d",
			$membership_id
		) ) ?: [];
		foreach ( $rows as $row ) {
			$route = $this->resolve_operations_manager( (int) $row->content_product_id, (int) $row->source_sales_manager_user_id );
			$inserted = $wpdb->query( $wpdb->prepare(
				"INSERT IGNORE INTO {$table} (membership_item_id,decision_due_at,auto_route_eligible,customer_self_activation_allowed,source_sales_manager_user_id,operations_sales_manager_user_id,sales_supervisor_user_id,routing_source,base_credit,current_credit,stage,created_at,updated_at) SELECT mi.id,%s,1,%d,NULLIF(%d,0),NULLIF(%d,0),NULLIF(%d,0),%s,COALESCE(mi.base_credit_snapshot,0),COALESCE(mi.base_credit_snapshot,0),'awaiting_customer',%s,%s FROM {$items} mi WHERE mi.id=%d",
				$due,
				$this->product_self_activation_allowed( (int) $row->content_product_id ) ? 1 : 0,
				(int) $row->source_sales_manager_user_id,
				(int) $route['manager_id'],
				(int) ( $route['supervisor_id'] ?? 0 ),
				(string) $route['source'],
				$now,
				$now,
				(int) $row->id
			) );
			if ( $inserted ) {
				$this->event( (int) $row->id, 'operations_route_snapshotted', [
					'invoice_id' => $invoice_id,
					'source_sales_manager_user_id' => (int) $row->source_sales_manager_user_id,
					'operations_sales_manager_user_id' => (int) $route['manager_id'],
					'operations_sales_supervisor_user_id' => (int) ( $route['supervisor_id'] ?? 0 ),
					'routing_source' => (string) $route['source'],
				], 0 );
				$operation = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$table} WHERE membership_item_id=%d LIMIT 1", (int) $row->id ) );
				if ( $operation && (int) $operation->customer_self_activation_allowed !== 1 ) {
					$this->route_forced_sales_expert( $operation );
				} elseif ( $operation && 0 === $inactivity_days ) {
					$this->route_operation_to_sales_manager( $operation, 0 );
				}
				if ( $operation && class_exists( 'SN_Purpose_Commission' ) ) { SN_Purpose_Commission::reconcile_operation( (int) $operation->id ); }
			}
		}
	}

	private function ensure_operation( int $item_id, bool $auto_eligible = false ) {
		global $wpdb; $table = $this->table(); $items = $wpdb->prefix . 'sn_project_membership_items';
		$row = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$table} WHERE membership_item_id=%d LIMIT 1", $item_id ) );
		if ( $row ) { return $row; }
		$memberships = $wpdb->prefix . 'sn_project_memberships';
		$item = $wpdb->get_row( $wpdb->prepare( "SELECT mi.id,mi.content_product_id,COALESCE(mi.base_credit_snapshot,0) base_credit,mi.created_at,COALESCE(m.source_sales_manager_user_id,0) source_sales_manager_user_id FROM {$items} mi INNER JOIN {$memberships} m ON m.id=mi.membership_id WHERE mi.id=%d LIMIT 1", $item_id ) );
		if ( ! $item ) { return null; }
		$now = current_time( 'mysql' );
		$due = $this->inactivity_due_at( $this->inactivity_days() );
		$route = $this->resolve_operations_manager( (int) $item->content_product_id, (int) $item->source_sales_manager_user_id );
		$wpdb->query( $wpdb->prepare( "INSERT IGNORE INTO {$table} (membership_item_id,decision_due_at,auto_route_eligible,customer_self_activation_allowed,source_sales_manager_user_id,operations_sales_manager_user_id,sales_supervisor_user_id,routing_source,base_credit,current_credit,stage,created_at,updated_at) SELECT mi.id,%s,%d,%d,NULLIF(%d,0),NULLIF(%d,0),NULLIF(%d,0),%s,COALESCE(mi.base_credit_snapshot,0),COALESCE(mi.base_credit_snapshot,0),'awaiting_customer',%s,%s FROM {$items} mi WHERE mi.id=%d", $due, $auto_eligible ? 1 : 0, $this->product_self_activation_allowed( (int) $item->content_product_id ) ? 1 : 0, (int) $item->source_sales_manager_user_id, (int) $route['manager_id'], (int) ( $route['supervisor_id'] ?? 0 ), (string) $route['source'], $now, $now, $item_id ) );
		$created = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$table} WHERE membership_item_id=%d LIMIT 1", $item_id ) );
		if ( $created && (int) $created->customer_self_activation_allowed !== 1 ) { $this->route_forced_sales_expert( $created ); $created = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$table} WHERE membership_item_id=%d LIMIT 1", $item_id ) ); }
		return $created;
	}

	private function item_context( int $item_id ) {
		global $wpdb; $items=$wpdb->prefix.'sn_project_membership_items'; $memberships=$wpdb->prefix.'sn_project_memberships';
		if ( ! $this->table_exists( $items ) || ! $this->table_exists( $memberships ) ) { return null; }
		return $wpdb->get_row( $wpdb->prepare( "SELECT mi.*,m.customer_wp_id,m.customer_name,m.customer_phone,m.origin_user_id,m.source_sales_manager_user_id,m.source_invoice_id,m.status membership_status,m.subscription_name_snapshot FROM {$items} mi INNER JOIN {$memberships} m ON m.id=mi.membership_id WHERE mi.id=%d LIMIT 1", $item_id ) );
	}

	private function customer_owns_item( int $item_id, int $user_id ): bool {
		$ctx = $this->item_context( $item_id ); if ( ! $ctx || (string) $ctx->membership_status !== 'active' ) { return false; }
		if ( (int) $ctx->customer_wp_id > 0 ) { return (int) $ctx->customer_wp_id === $user_id; }
		$user = get_user_by( 'id', $user_id ); if ( ! $user instanceof WP_User || ! in_array( 'customer', (array) $user->roles, true ) ) { return false; }
		$target = SN_Helpers::normalize_mobile( (string) $ctx->customer_phone );
		foreach ( [ $user->user_login, get_user_meta( $user_id, 'billing_phone', true ), get_user_meta( $user_id, 'sn_customer_phone', true ) ] as $raw ) {
			if ( $target !== '' && SN_Helpers::normalize_mobile( (string) $raw ) === $target ) { return true; }
		}
		return false;
	}

	private function event( int $item_id, string $key, array $details = [], int $actor = 0 ): void {
		global $wpdb; $events=$wpdb->prefix.'sn_project_events'; $items=$wpdb->prefix.'sn_project_membership_items';
		if ( ! $this->table_exists( $events ) ) { return; }
		$membership_id = (int) $wpdb->get_var( $wpdb->prepare( "SELECT membership_id FROM {$items} WHERE id=%d", $item_id ) );
		$wpdb->insert( $events, [ 'membership_id'=>$membership_id ?: null, 'membership_item_id'=>$item_id ?: null, 'actor_user_id'=>$actor ?: null, 'event_key'=>sanitize_key($key), 'details'=>wp_json_encode($details,JSON_UNESCAPED_UNICODE|JSON_UNESCAPED_SLASHES), 'created_at'=>current_time('mysql') ] );
	}

	private function portal_url( int $item_id = 0, array $args = [] ): string {
		$page_id = absint( get_option( 'sn_biavin_customer_portal_page_id', 0 ) );
		$url = $page_id ? (string) get_permalink( $page_id ) : home_url( '/biawin/' );
		if ( $item_id ) { $args['card'] = $item_id; }
		return $args ? add_query_arg( $args, $url ) : $url;
	}

	private function redirect_back( string $message = '', string $kind = 'success', string $fallback = '' ): void {
		// Customer actions return to the exact card so the refreshed result is visible.
		$target_query = [];
		parse_str( (string) wp_parse_url( $fallback, PHP_URL_QUERY ), $target_query );
		$card_id = absint( $target_query['card'] ?? 0 );
		$url = $card_id > 0 ? $fallback : ( wp_get_referer() ?: $fallback ?: home_url( '/' ) );
		$kind = $kind === 'error' ? 'error' : 'success';
		$message = sanitize_text_field( $message );
		$args = $message !== '' ? [ 'sn_ops_notice'=>'1', 'sn_ops_kind'=>$kind ] : [];
		wp_safe_redirect( $args ? add_query_arg( $args, $url ) : $url ); exit;
	}

	public function handle_customer_action(): void {
		if ( ! is_user_logged_in() ) { auth_redirect(); }
		$item_id = absint( $_POST['item_id'] ?? 0 ); $choice = sanitize_key( wp_unslash( $_POST['choice'] ?? '' ) );
		check_admin_referer( 'sn_operations_customer_' . $item_id );
		$user_id = get_current_user_id();
		if ( ! $this->customer_owns_item( $item_id, $user_id ) ) { wp_die( 'دسترسی به این کارت مجاز نیست.' ); }
		$op = $this->ensure_operation( $item_id, false );
		if ( ! $op ) { $this->redirect_back( 'زیرساخت کارت آماده نیست.', 'error', $this->portal_url($item_id) ); }
		$op = $this->maybe_route_operation_if_due( $op );
		$ctx = $this->item_context( $item_id );
		if ( $ctx && class_exists( 'SN_Operations_Execution' ) && SN_Operations_Execution::instance()->product_callback_only( (int) $ctx->content_product_id ) ) {
			// Enforce the live policy on the server as well as hiding the buttons.
			if ( $choice !== 'expert' ) { $this->redirect_back( 'برای این کارت فقط انتخاب تاریخ تماس امکان‌پذیر است.', 'error', $this->portal_url( $item_id ) ); }
			$date_raw = $_POST['followup_date'] ?? '';
			$parsed = $this->normalize_followup_datetime( is_scalar( $date_raw ) ? (string) wp_unslash( $date_raw ) : '' );
			if ( is_wp_error( $parsed ) ) { $this->redirect_back( $parsed->get_error_message(), 'error', $this->portal_url( $item_id ) ); }
			$updated = $this->request_customer_callback( $op, $parsed['mysql'], $user_id );
			$this->redirect_back( $updated ? 'تاریخ تماس ثبت شد. کارشناس در این تاریخ با شما تماس خواهد گرفت.' : 'وضعیت یا تاریخ تماس کارت تغییر کرده است؛ صفحه را تازه کنید.', $updated ? 'success' : 'error', $this->portal_url( $item_id ) );
		}
		if ( (int) ( $op->customer_self_activation_allowed ?? 1 ) !== 1 ) { $this->redirect_back( 'این کارت طبق تنظیم محصول مستقیماً وارد فلو کارشناس فروش عملیات شده است.', 'success', $this->portal_url($item_id) ); }
		if ( (string) $op->stage !== 'awaiting_customer' && in_array( $choice, [ 'expert', 'normal', 'upgrade' ], true ) ) {
			$this->redirect_back( 'مهلت انتخاب این کارت پایان یافته و کارت برای مدیر فروش عملیات ارسال شده است.', 'error', $this->portal_url($item_id) );
		}
		if ( $choice === 'expert' ) {
			$parsed = $this->normalize_followup_datetime( (string) ( $_POST['followup_date'] ?? '' ) );
			if ( is_wp_error( $parsed ) ) { $this->redirect_back( $parsed->get_error_message(), 'error', $this->portal_url($item_id) ); }
			$op = $this->hydrate_operation_route( $op );
			$direct_supervisor_id = (int) ( $op->sales_supervisor_user_id ?? 0 );
			$destination_stage = $direct_supervisor_id > 0 ? 'sales_supervisor' : 'sales_manager_queue';
			$transition = [ 'customer_choice'=>'expert','customer_choice_at'=>current_time('mysql'),'follow_up_at'=>$parsed['mysql'],'follow_up_reminded_at'=>null,'follow_up_reminder_attempts'=>0,'stage'=>$destination_stage,'routed_at'=>current_time('mysql') ];
			if ( $direct_supervisor_id > 0 ) { $transition['supervisor_assigned_at'] = current_time( 'mysql' ); }
			$updated = $this->transition_awaiting( $item_id, $transition );
			if ( $updated ) { $this->event( $item_id, 'operations_customer_requested_expert', [ 'follow_up_at' => $parsed['mysql'], 'requested_by_customer' => 1, 'timezone' => 'Asia/Tehran', 'destination_stage' => $destination_stage, 'operations_supervisor_user_id' => $direct_supervisor_id ], $user_id ); }
			$this->redirect_back( $updated ? 'درخواست فعال‌سازی توسط کارشناس ثبت شد و تاریخ انتخابی شما برای تماس ذخیره شد.' : 'وضعیت کارت تغییر کرده است؛ صفحه را تازه کنید.', $updated ? 'success' : 'error', $this->portal_url($item_id) );
		}
		if ( in_array( $choice, [ 'normal', 'continue_normal' ], true ) ) {
			if ( $choice === 'normal' ) {
				$card_type = $this->customer_card_type( $item_id );
				if ( $card_type === 'wallet_charge' ) {
					$issued = $this->issue_customer_wallet_code( $item_id, $user_id );
					$this->redirect_back( $issued === 'issued' ? 'کد فعال‌سازی این کارت صادر شد و در همین پنجره نمایش داده می‌شود.' : ( $issued === 'queued' ? 'درخواست شما ثبت شد. در حال حاضر کدی برای این کارت موجود نیست؛ کارشناسان با شما تماس خواهند گرفت.' : 'درخواست فعال‌سازی ثبت نشد؛ دوباره تلاش کنید.' ), $issued === 'error' ? 'error' : 'success', $this->portal_url( $item_id ) );
				}
				if ( in_array( $card_type, [ 'form', 'physical_invoice' ], true ) ) {
					$updated = $this->route_self_to_executive( $item_id, $user_id );
					$this->redirect_back( $updated ? 'درخواست شما ثبت شد و کارشناسان ما با شما تماس خواهند گرفت.' : 'وضعیت کارت تغییر کرده است؛ صفحه را تازه کنید.', $updated ? 'success' : 'error', $this->portal_url( $item_id ) );
				}
			}
			$updated = $this->route_normal_to_executive( $item_id, $user_id, $choice === 'normal' ? 'self' : 'customer_continue_normal' );
			$this->redirect_back( $updated ? 'کارت با اعتبار عادی برای مدیر اجرایی عملیات ارسال شد.' : 'وضعیت کارت تغییر کرده است؛ صفحه را تازه کنید.', $updated ? 'success' : 'error', $this->portal_url($item_id) );
		}
		if ( in_array( $choice, [ 'upgrade', 'retry_upgrade' ], true ) ) {
		if ( $choice === 'upgrade' ) {
			$item = $this->item_context( $item_id );
			if ( ! $item || ! $this->customer_upgrade_options_visible( (int) $item->content_product_id ) ) {
				$this->redirect_back( 'انتخاب افزایشی این کارت در پروفایل مشتری غیرفعال است.', 'error', $this->portal_url( $item_id ) );
			}
		}
			if ( $choice === 'retry_upgrade' && (int) $op->upgrade_invoice_id > 0 ) {
				$url = $this->invoice_url( (int) $op->upgrade_invoice_id );
				if ( $url !== '' ) { wp_safe_redirect( $url ); exit; }
			}
			$target_id = absint( $_POST['target_product_id'] ?? 0 );
			$result = $this->create_upgrade_invoice( $item_id, $target_id, $user_id, 'customer' );
			if ( is_wp_error( $result ) ) { $this->redirect_back( $result->get_error_message(), 'error', $this->portal_url($item_id) ); }
			if ( ! empty( $result['free'] ) ) { $this->redirect_back( 'ارتقای رایگان ثبت شد و کارت با اعتبار جدید برای اجرای عملیات ارسال شد.', 'success', $this->portal_url( $item_id ) ); }
			wp_safe_redirect( (string) $result['url'] ); exit;
		}
		$this->redirect_back( 'انتخاب نامعتبر است.', 'error', $this->portal_url($item_id) );
	}

	private function customer_callback_stages(): array {
		return [ 'awaiting_customer', 'sales_manager_queue', 'sales_supervisor', 'sales_expert', 'upgrade_payment', 'executive_manager_queue', 'executive_in_progress', 'execution_expert', 'execution_form_review' ];
	}

	/** Record one customer date without reassigning an already routed card. */
	private function request_customer_callback( object $op, string $date, int $actor ): bool {
		$stage = (string) $op->stage;
		$date_before = SN_Operations_Execution::instance()->customer_callback_date( (int) $op->membership_item_id, (string) ( $op->follow_up_at ?? '' ) );
		if ( ! in_array( $stage, $this->customer_callback_stages(), true ) || $date_before !== '' ) { return false; }
		if ( $stage === 'awaiting_customer' ) { $op = $this->hydrate_operation_route( $op ); }
		$supervisor_id = (int) ( $op->sales_supervisor_user_id ?? 0 );
		$destination = $stage === 'awaiting_customer' ? ( $supervisor_id > 0 ? 'sales_supervisor' : 'sales_manager_queue' ) : $stage;
		$now = current_time( 'mysql' );
		global $wpdb; $table = $this->table();
		$updated = $wpdb->query( $wpdb->prepare(
			"UPDATE {$table} SET customer_choice='expert',customer_choice_at=%s,follow_up_at=%s,follow_up_reminded_at=NULL,follow_up_reminder_attempts=0,routed_at=CASE WHEN stage='awaiting_customer' THEN %s ELSE routed_at END,supervisor_assigned_at=CASE WHEN stage='awaiting_customer' AND %d>0 THEN %s ELSE supervisor_assigned_at END,stage=%s,updated_at=%s WHERE id=%d AND stage=%s AND (follow_up_at IS NULL OR follow_up_at='')",
			$now, $date, $now, $supervisor_id, $now, $destination, $now, (int) $op->id, $stage
		) );
		if ( $updated !== 1 ) { return false; }
		$this->event( (int) $op->membership_item_id, 'operations_customer_requested_expert', [ 'follow_up_at'=>$date, 'requested_by_customer'=>1, 'callback_only'=>1, 'timezone'=>'Asia/Tehran', 'destination_stage'=>$destination ], $actor );
		if ( (int) ( $op->sales_expert_user_id ?? 0 ) > 0 ) { $this->schedule_existing_followup( (int) $op->id, (int) $op->sales_expert_user_id, $date ); }
		return true;
	}

	private function render_customer_callback_controls( int $item_id, object $op ): string {
		$contactable = in_array( (string) $op->stage, $this->customer_callback_stages(), true );
		$date = SN_Operations_Execution::instance()->customer_callback_date( $item_id, (string) ( $op->follow_up_at ?? '' ) );
		ob_start(); ?>
		<div class="sn-customer-card-actions sn-customer-callback-only" id="sn-card-<?php echo esc_attr( (string) $item_id ); ?>">
			<?php if ( $contactable && $date === '' ) : ?>
				<form method="post" action="<?php echo esc_url( admin_url( 'admin-post.php' ) ); ?>" class="sn-customer-expert-choice sn-ops-followup-form" data-sn-ops-followup>
					<input type="hidden" name="action" value="sn_operations_customer_action"><input type="hidden" name="item_id" value="<?php echo esc_attr( (string) $item_id ); ?>"><input type="hidden" name="choice" value="expert"><?php wp_nonce_field( 'sn_operations_customer_' . $item_id ); ?>
					<strong>انتخاب تاریخ تماس</strong><p>کارشناس در تاریخ انتخابی با شما تماس خواهد گرفت.</p>
					<label><span>تاریخ تماس</span><input class="sn-ops-jalali-date" name="followup_date" placeholder="تاریخ را انتخاب کنید" pattern="[۰-۹0-9]{4}[\/-][۰-۹0-9]{1,2}[\/-][۰-۹0-9]{1,2}" autocomplete="off" required></label><button type="submit" data-sn-ops-followup-submit>ثبت تاریخ تماس</button>
				</form>
			<?php elseif ( $contactable ) : ?>
				<div class="sn-customer-callback-time" role="status"><span>تاریخ تماس</span><strong><?php echo esc_html( $this->jalali_date( $date ) ); ?></strong><p>کارشناس در این تاریخ با شما تماس خواهد گرفت.</p></div>
			<?php else : ?><div class="sn-customer-operation-state"><span><?php echo esc_html( $this->status_label( (string) $op->stage ) ); ?></span></div><?php endif; ?>
		</div>
		<?php return (string) ob_get_clean();
	}

	private function customer_card_type( int $item_id ): string {
		global $wpdb;
		$product_id = (int) $wpdb->get_var( $wpdb->prepare( "SELECT content_product_id FROM {$wpdb->prefix}sn_project_membership_items WHERE id=%d", $item_id ) );
		return $product_id ? sanitize_key( (string) get_post_meta( $product_id, '_sn_execution_fulfillment_type', true ) ) : '';
	}

	private function route_self_to_executive( int $item_id, int $actor ): bool {
		$now = current_time( 'mysql' );
		$updated = $this->transition_awaiting( $item_id, [ 'customer_choice'=>'self', 'customer_choice_at'=>$now, 'activation_mode'=>'normal', 'stage'=>'executive_manager_queue', 'routed_at'=>$now ] );
		if ( $updated ) {
			$this->event( $item_id, 'operations_customer_self_requested_callback', [], $actor );
			global $wpdb; $op = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$this->table()} WHERE membership_item_id=%d", $item_id ) );
			if ( $op && class_exists( 'SN_Operations_Execution' ) ) { SN_Operations_Execution::instance()->ensure_case( $op ); }
		}
		return $updated;
	}

	private function issue_customer_wallet_code( int $item_id, int $actor ): string {
		global $wpdb;
		if ( ! class_exists( 'SN_Operations_Execution' ) ) { return 'error'; }
		$wpdb->query( 'START TRANSACTION' );
		$op = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$this->table()} WHERE membership_item_id=%d FOR UPDATE", $item_id ) );
		if ( ! $op || (string) $op->stage !== 'awaiting_customer' ) { $wpdb->query( 'ROLLBACK' ); return 'error'; }
		$product_id = (int) $wpdb->get_var( $wpdb->prepare( "SELECT content_product_id FROM {$wpdb->prefix}sn_project_membership_items WHERE id=%d", $item_id ) );
		if ( ! $product_id || $this->customer_card_type( $item_id ) !== 'wallet_charge' ) { $wpdb->query( 'ROLLBACK' ); return 'error'; }
		$destination = esc_url_raw( (string) get_post_meta( $product_id, '_sn_execution_wallet_destination_url', true ) );
		$code = wp_http_validate_url( $destination ) ? SN_Operations_Execution::instance()->reserve_wallet_code( $product_id, $item_id ) : null;
		$now = current_time( 'mysql' ); $stage = $code === null ? 'executive_manager_queue' : 'customer_code_issued';
		$ok = $wpdb->query( $wpdb->prepare( "UPDATE {$this->table()} SET customer_choice='self',customer_choice_at=%s,activation_mode='normal',stage=%s,routed_at=%s,updated_at=%s WHERE id=%d AND stage='awaiting_customer'", $now, $stage, $now, $now, (int) $op->id ) );
		if ( $ok !== 1 ) { $wpdb->query( 'ROLLBACK' ); return 'error'; }
		$wpdb->query( 'COMMIT' );
		$this->event( $item_id, $code === null ? 'operations_wallet_code_unavailable' : 'operations_wallet_code_issued', [ 'product_id'=>$product_id ], $actor );
		if ( $code === null ) {
			$op->stage = $stage;
			SN_Operations_Execution::instance()->ensure_case( $op );
			return 'queued';
		}
		return 'issued';
	}

	private function transition_awaiting( int $item_id, array $data ): bool {
		global $wpdb; $table = $this->table(); $data['updated_at'] = current_time('mysql');
		$set=[]; $args=[]; foreach ( $data as $key=>$value ) { $set[]="{$key}=".(is_null($value)?'NULL':'%s'); if(!is_null($value)){$args[]=$value;} }
		$args[]=$item_id;
		return 1 === (int) $wpdb->query( $wpdb->prepare( "UPDATE {$table} SET ".implode(',',$set)." WHERE membership_item_id=%d AND stage='awaiting_customer'", ...$args ) );
	}

	private function route_normal_to_executive( int $item_id, int $actor, string $source ): bool {
		global $wpdb; $table=$this->table(); $op=$this->ensure_operation($item_id,false); if(!$op){return false;} $now=current_time('mysql');
		if ( (float) $op->base_credit <= 0 ) { return false; }
		if ( in_array( $source, [ 'refer_normal', 'revert_normal' ], true ) && ! $this->can_work_case( $op, $actor ) ) { return false; }
		$allowed = [ 'awaiting_customer','upgrade_payment','sales_expert' ];
		if ( ! in_array( (string) $op->stage, $allowed, true ) ) { return false; }
		if ( (int) $op->upgrade_action_id ) { $wpdb->query($wpdb->prepare("UPDATE {$wpdb->prefix}sn_project_actions SET status='cancelled_continue_normal',updated_at=%s WHERE id=%d AND status IN ('awaiting_payment','retry_requested','payment_reversed')",$now,(int)$op->upgrade_action_id)); }
		$choice=(string)$op->customer_choice?:($source==='self'?'self':'expert');$choice_at=(string)$op->customer_choice_at?:$now;$payment_state=(int)$op->upgrade_invoice_id?'cancelled_continue_normal':'';
		$updated=$wpdb->query($wpdb->prepare("UPDATE {$table} SET customer_choice=%s,customer_choice_at=%s,activation_mode='normal',current_credit=base_credit,payment_state=%s,stage='executive_manager_queue',follow_up_at=NULL,routed_at=%s,updated_at=%s WHERE membership_item_id=%d AND stage IN ('awaiting_customer','upgrade_payment','sales_expert') AND base_credit>0 AND COALESCE(payment_state,'')<>'paid'",$choice,$choice_at,$payment_state,$now,$now,$item_id));
		if($updated){$execution_op=$wpdb->get_row($wpdb->prepare("SELECT * FROM {$table} WHERE membership_item_id=%d LIMIT 1",$item_id));$this->event($item_id,'operations_routed_normal_to_execution',[ 'source'=>$source,'credit'=>(float)($execution_op->base_credit??$op->base_credit) ],$actor);if($execution_op&&class_exists('SN_Operations_Execution')){SN_Operations_Execution::instance()->ensure_case($execution_op);}return true;} return false;
	}

	private function upgrade_option( object $ctx, int $target_id ) {
		$options = json_decode( (string) ( $ctx->upgrade_options_snapshot_json ?? '' ), true );
		foreach ( is_array( $options ) ? $options : [] as $row ) {
			if ( ! is_array( $row ) ) { continue; }
			$id = absint( $row['target_product_id'] ?? $row['product_id'] ?? $row['id'] ?? 0 );
			$raw_credit = $row['target_credit'] ?? $row['credit'] ?? null;
			$raw_amount = $row['upgrade_amount'] ?? $row['price'] ?? null;
			if ( $id !== $target_id || ! is_numeric( $raw_credit ) || (float) $raw_credit < 0 || ! is_numeric( $raw_amount ) || (float) $raw_amount < 0 ) { continue; }
			return [
				'target_product_id' => $id,
				'name' => sanitize_text_field( (string) ( $row['name'] ?? $row['target_name'] ?? get_the_title( $id ) ) ),
				'credit' => (float) $raw_credit,
				'amount' => (float) $raw_amount,
			];
		}
		return null;
	}

	/** Recheck the selected snapshot after the operation lock, against admin sync. */
	private function upgrade_snapshot_unchanged( int $item_id, object $ctx, array $option ): bool {
		global $wpdb;
		$fresh = $wpdb->get_row( $wpdb->prepare( "SELECT base_credit_snapshot,upgrade_options_snapshot_json FROM {$wpdb->prefix}sn_project_membership_items WHERE id=%d FOR UPDATE", $item_id ) );
		if ( ! $fresh || abs( (float) $fresh->base_credit_snapshot - (float) ( $ctx->base_credit_snapshot ?? 0 ) ) > 0.005 ) { return false; }
		$current = $this->upgrade_option( $fresh, (int) $option['target_product_id'] );
		return $current !== null && $current === $option;
	}

	/** Apply a zero-cost upgrade without creating a payment invoice or paid sale. */
	private function apply_free_upgrade( int $item_id, object $ctx, array $option, int $actor_id, string $source ) {
		global $wpdb;
		$table = $this->table();
		$actions = $wpdb->prefix . 'sn_project_actions';
		if ( ! $this->table_exists( $actions ) ) { return new WP_Error( 'sn_ops_action_schema', 'جدول اقدامات پروژه آماده نیست.' ); }
		if ( false === $wpdb->query( 'START TRANSACTION' ) ) { return new WP_Error( 'sn_ops_transaction', 'شروع ثبت ارتقای رایگان ممکن نشد.' ); }
		$op = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$table} WHERE membership_item_id=%d FOR UPDATE", $item_id ) );
		if ( ! $op || ! in_array( (string) $op->stage, $source === 'customer' ? [ 'awaiting_customer' ] : [ 'sales_expert' ], true ) || (int) $op->upgrade_invoice_id > 0 ) {
			$wpdb->query( 'ROLLBACK' );
			return new WP_Error( 'sn_ops_state_changed', 'وضعیت کارت یا فاکتور افزایشی قبلی تغییر کرده است؛ صفحه را تازه کنید.' );
		}
		if ( ( $source === 'customer' && ( (int) $op->customer_self_activation_allowed !== 1 || ! $this->customer_upgrade_options_visible( (int) $ctx->content_product_id ) ) ) || ( $source === 'expert' && ! $this->can_work_case( $op, $actor_id ) ) ) {
			$wpdb->query( 'ROLLBACK' );
			return new WP_Error( 'sn_ops_scope_changed', 'دسترسی به ارتقای این کارت تغییر کرده است.' );
		}
		if ( ! $this->upgrade_snapshot_unchanged( $item_id, $ctx, $option ) ) {
			$wpdb->query( 'ROLLBACK' );
			return new WP_Error( 'sn_ops_product_changed', 'اعتبار یا گزینه افزایشی این کارت تغییر کرده است؛ صفحه را تازه کنید.' );
		}
		$now = current_time( 'mysql' );
		$action = $this->filter_columns( $actions, [
			'membership_item_id'=>$item_id, 'actor_user_id'=>$actor_id, 'action_type'=>'upsell',
			'fulfillment_mode'=>$source === 'customer' ? 'self' : 'expert',
			'target_product_id'=>(int) $option['target_product_id'], 'target_product_name'=>$option['name'],
			'base_credit'=>(float) $op->base_credit, 'target_credit'=>(float) $option['credit'],
			'amount'=>0, 'status'=>'ready', 'note'=>'ارتقای رایگان بدون فاکتور پرداخت',
			'created_at'=>$now, 'updated_at'=>$now,
		] );
		if ( ! $wpdb->insert( $actions, $action ) ) {
			$wpdb->query( 'ROLLBACK' );
			return new WP_Error( 'sn_ops_free_action', 'ثبت اقدام ارتقای رایگان انجام نشد.' );
		}
		$action_id = (int) $wpdb->insert_id;
		$updated = $wpdb->update( $table, [
			'customer_choice'=>$source === 'customer' ? 'self' : ( (string) $op->customer_choice ?: 'expert' ),
			'customer_choice_at'=>(string) $op->customer_choice_at ?: $now,
			'activation_mode'=>'upsell', 'current_credit'=>(float) $option['credit'],
			'upgrade_target_product_id'=>(int) $option['target_product_id'],
			'upgrade_target_name'=>$option['name'], 'upgrade_amount'=>0,
			'upgrade_invoice_id'=>null, 'upgrade_action_id'=>$action_id,
			'payment_state'=>'not_required', 'stage'=>'executive_manager_queue',
			'follow_up_at'=>null, 'routed_at'=>$now, 'updated_at'=>$now,
		], [ 'id'=>(int) $op->id ] );
		if ( $updated !== 1 || false === $wpdb->query( 'COMMIT' ) ) {
			$wpdb->query( 'ROLLBACK' );
			return new WP_Error( 'sn_ops_free_upgrade', 'ثبت ارتقای رایگان کارت کامل نشد.' );
		}
		$this->event( $item_id, 'operations_upgrade_free_routed_to_execution', [ 'target_product_id'=>(int) $option['target_product_id'], 'current_credit'=>(float) $option['credit'], 'source'=>$source, 'action_id'=>$action_id ], $actor_id );
		if ( class_exists( 'SN_Operations_Execution' ) ) {
			$execution_op = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$table} WHERE id=%d LIMIT 1", (int) $op->id ) );
			if ( $execution_op ) { SN_Operations_Execution::instance()->ensure_case( $execution_op ); }
		}
		return [ 'free'=>true, 'action_id'=>$action_id ];
	}

	private function generate_invoice_code(): string {
		global $wpdb; $table=$wpdb->prefix.'sn_invoices';
		for($i=0;$i<50;$i++){try{$code=(string)random_int(100000,999999);}catch(Throwable $e){$code=(string)wp_rand(100000,999999);} if(!(int)$wpdb->get_var($wpdb->prepare("SELECT id FROM {$table} WHERE invoice_code=%s LIMIT 1",$code))){return $code;}}
		throw new RuntimeException('invoice_code_collision');
	}

	private function generate_invoice_short_code( int $length = 8 ): string {
		global $wpdb; $table=$wpdb->prefix.'sn_invoices'; $alphabet='ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789'; $max=strlen($alphabet)-1;
		for($attempt=0;$attempt<20;$attempt++){$code='';for($i=0;$i<$length;$i++){try{$index=random_int(0,$max);}catch(Throwable $e){$index=wp_rand(0,$max);}$code.=$alphabet[$index];}if(!(int)$wpdb->get_var($wpdb->prepare("SELECT id FROM {$table} WHERE short_code=%s LIMIT 1",$code))){return $code;}}
		return wp_generate_password(max(12,$length),false,false);
	}


	private function parse_money_value( $value ): float {
		$value = strtr( (string) $value, [ '۰'=>'0','۱'=>'1','۲'=>'2','۳'=>'3','۴'=>'4','۵'=>'5','۶'=>'6','۷'=>'7','۸'=>'8','۹'=>'9','٠'=>'0','١'=>'1','٢'=>'2','٣'=>'3','٤'=>'4','٥'=>'5','٦'=>'6','٧'=>'7','٨'=>'8','٩'=>'9' ] );
		$value = preg_replace( '/[^0-9.]/', '', $value );
		return max( 0, (float) $value );
	}

	private function partial_payment_presets(): array {
		$raw = (string) get_option( 'sn_partial_payment_presets', '1000000,2000000,3000000,5000000' );
		$parts = preg_split( '/[,،;\n\r]+/u', $raw ) ?: [];
		$out = [];
		foreach ( $parts as $part ) {
			$amount = $this->parse_money_value( $part );
			if ( $amount > 0 ) { $out[ (string) (int) round( $amount ) ] = $amount; }
		}
		$out = array_values( $out );
		sort( $out, SORT_NUMERIC );
		return $out ?: [ 1000000, 2000000, 3000000, 5000000 ];
	}

	private function requested_initial_payment( float $total, string $source ): array {
		// Customer self-activation keeps the legacy one-stage behavior. The new staged
		// controls intentionally belong to the Operations sales expert workflow.
		if ( $source !== 'expert' ) { return [ 'plan'=>'full', 'due'=>$total ]; }
		$plan = sanitize_key( (string) wp_unslash( $_POST['payment_plan'] ?? 'full' ) );
		if ( $plan !== 'partial' ) { return [ 'plan'=>'full', 'due'=>$total ]; }
		$choice = sanitize_text_field( wp_unslash( $_POST['prepayment_choice'] ?? '' ) );
		$due = $choice === 'custom'
			? $this->parse_money_value( $_POST['prepayment_custom'] ?? 0 )
			: $this->parse_money_value( $choice );
		if ( $due <= 0.5 ) { return [ 'error'=>new WP_Error( 'sn_ops_partial_amount', 'مبلغ مرحله اول باید بیشتر از صفر باشد.' ) ]; }
		if ( $due - $total > 0.5 ) { return [ 'error'=>new WP_Error( 'sn_ops_partial_amount', 'مبلغ مرحله اول نمی‌تواند از مبلغ کل افزایشی بیشتر باشد.' ) ]; }
		if ( abs( $due - $total ) <= 0.5 ) { return [ 'plan'=>'full', 'due'=>$total ]; }
		return [ 'plan'=>'partial', 'due'=>$due ];
	}

	private function upgrade_payment_summary( int $invoice_id ): array {
		global $wpdb;
		$empty = [ 'exists'=>false, 'total'=>0.0, 'paid'=>0.0, 'remaining'=>0.0, 'due'=>0.0, 'stage_no'=>0, 'workflow'=>'', 'status'=>'', 'can_issue_next'=>false ];
		if ( $invoice_id < 1 ) { return $empty; }
		$table = $wpdb->prefix . 'sn_invoices';
		if ( ! $this->table_exists( $table ) ) { return $empty; }
		$invoice = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$table} WHERE id=%d LIMIT 1", $invoice_id ) );
		if ( ! $invoice ) { return $empty; }
		$total = max( 0, (float) ( ( $invoice->payment_total_amount ?? 0 ) ?: ( ( $invoice->final_total ?? 0 ) ?: ( $invoice->product_price ?? 0 ) ) ) );
		$paid = min( $total, max( 0, (float) ( $invoice->paid_total_amount ?? 0 ) ) );
		$status = sanitize_key( (string) ( $invoice->status ?? '' ) );
		$workflow = sanitize_key( (string) ( $invoice->payment_workflow_status ?? '' ) );
		if ( $paid <= 0.5 && in_array( $status, [ 'paid','approved' ], true ) && $workflow !== 'awaiting_assignment' ) { $paid = $total; }
		$remaining = max( 0, $total - $paid );
		$due = min( $remaining, max( 0, (float) ( $invoice->current_due_amount ?? 0 ) ) );
		$stage_no = max( 1, (int) ( $invoice->current_payment_stage ?? 1 ) );
		return [
			'exists'=>true,
			'total'=>$total,
			'paid'=>$paid,
			'remaining'=>$remaining,
			'due'=>$due,
			'stage_no'=>$stage_no,
			'workflow'=>$workflow,
			'status'=>$status,
			'can_issue_next'=>$remaining > 0.5 && $due <= 0.5 && $workflow === 'awaiting_assignment',
		];
	}

	private function issue_upgrade_payment_stage( object $op, int $actor_id, string $mode, float $requested_amount = 0 ) {
		global $wpdb;
		if ( ! $this->can_work_case( $op, $actor_id ) || (string) $op->stage !== 'upgrade_payment' || (int) $op->upgrade_invoice_id < 1 ) {
			return new WP_Error( 'sn_ops_stage_scope', 'این پرونده برای صدور مرحله بعد پرداخت در اختیار شما نیست.' );
		}
		$invoice_id = (int) $op->upgrade_invoice_id;
		$invoice_table = $wpdb->prefix . 'sn_invoices';
		$stage_table = $wpdb->prefix . 'sn_invoice_payment_stages';
		if ( ! $this->table_exists( $invoice_table ) || ! $this->table_exists( $stage_table ) ) { return new WP_Error( 'sn_ops_stage_schema', 'زیرساخت پرداخت مرحله‌ای آماده نیست.' ); }
		$mode = in_array( $mode, [ 'partial','remaining' ], true ) ? $mode : 'partial';
		$wpdb->query( 'START TRANSACTION' );
		$invoice = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$invoice_table} WHERE id=%d FOR UPDATE", $invoice_id ) );
		$locked_op = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$this->table()} WHERE id=%d FOR UPDATE", (int) $op->id ) );
		if ( ! $invoice || ! $locked_op ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_ops_stage_missing', 'فاکتور یا پرونده پرداخت یافت نشد.' ); }
		if ( ! $this->can_work_case( $locked_op, $actor_id ) || (string) $locked_op->stage !== 'upgrade_payment' || (int) $locked_op->upgrade_invoice_id !== $invoice_id ) {
			$wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_ops_stage_changed', 'تخصیص یا وضعیت پرونده تغییر کرده است؛ صفحه را تازه کنید.' );
		}
		$total = max( 0, (float) ( ( $invoice->payment_total_amount ?? 0 ) ?: ( ( $invoice->final_total ?? 0 ) ?: ( $invoice->product_price ?? 0 ) ) ) );
		$paid = min( $total, max( 0, (float) ( $invoice->paid_total_amount ?? 0 ) ) );
		$remaining = max( 0, $total - $paid );
		$due = min( $remaining, max( 0, (float) ( $invoice->current_due_amount ?? 0 ) ) );
		$workflow = sanitize_key( (string) ( $invoice->payment_workflow_status ?? '' ) );
		if ( $remaining <= 0.5 || $workflow === 'completed' ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_ops_stage_paid', 'این فاکتور قبلاً کامل تسویه شده است.' ); }
		if ( $workflow !== 'awaiting_assignment' || $due > 0.5 ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_ops_stage_open', 'مرحله فعلی هنوز تعیین تکلیف نشده یا مرحله بعد قبلاً صادر شده است.' ); }
		$amount = $mode === 'remaining' ? $remaining : max( 0, $requested_amount );
		if ( $amount <= 0.5 || $amount - $remaining > 0.5 ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_ops_stage_amount', 'مبلغ مرحله جدید باید بیشتر از صفر و حداکثر برابر مانده فاکتور باشد.' ); }
		if ( abs( $amount - $remaining ) <= 0.5 ) { $mode = 'remaining'; $amount = $remaining; }
		$next_stage = max( 1, (int) ( $invoice->current_payment_stage ?? 1 ) ) + 1;
		$exists = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$stage_table} WHERE invoice_id=%d AND stage_no=%d LIMIT 1", $invoice_id, $next_stage ) );
		if ( $exists ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_ops_stage_duplicate', 'مرحله بعد قبلاً ایجاد شده است؛ صفحه را تازه کنید.' ); }
		$now = current_time( 'mysql' );
		$stage_data = $this->filter_columns( $stage_table, [
			'invoice_id'=>$invoice_id,
			'stage_no'=>$next_stage,
			'stage_type'=>$mode === 'remaining' ? 'remaining' : 'partial',
			'requested_amount'=>$amount,
			'status'=>'pending',
			'issued_by_user_id'=>$actor_id,
			'note'=>$mode === 'remaining' ? 'تسویه کامل مانده توسط کارشناس فروش عملیات' : 'مرحله پرداخت جدید توسط کارشناس فروش عملیات',
			'created_at'=>$now,
			'updated_at'=>$now,
		] );
		if ( ! $wpdb->insert( $stage_table, $stage_data ) ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_ops_stage_insert', 'مرحله پرداخت جدید ذخیره نشد: ' . ( $wpdb->last_error ?: 'خطای دیتابیس' ) ); }
		$invoice_update = $this->filter_columns( $invoice_table, [
			'current_due_amount'=>$amount,
			'current_payment_stage'=>$next_stage,
			'payment_workflow_status'=>'awaiting_payment',
			'current_stage_issued_by_user_id'=>$actor_id,
			'status'=>'pre_invoice',
			'invoice_status'=>'pre_invoice',
			'payment_status'=>'pending_payment',
			'receipt_url'=>null, 'receipt_urls'=>null,
			'receipt_file'=>null,
			'receipt_source'=>null,
			'manual_card_from'=>null,
			'manual_card_to'=>null,
			'manual_amount'=>null,
			'manual_paid_at'=>null,
			'manual_paid_at_jalali'=>null,
			'rejected_by'=>null,
			'rejected_at'=>null,
			'rejected_reason'=>null,
			'updated_at'=>$now,
		] );
		if ( false === $wpdb->update( $invoice_table, $invoice_update, [ 'id'=>$invoice_id ] ) ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_ops_stage_invoice', 'وضعیت مرحله جدید روی فاکتور ذخیره نشد.' ); }
		if ( false === $wpdb->update( $this->table(), [ 'payment_state'=>'awaiting_payment', 'updated_at'=>$now ], [ 'id'=>(int) $locked_op->id ] ) ) { $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_ops_stage_case', 'وضعیت پرداخت پرونده ذخیره نشد.' ); }
		if ( (int) $locked_op->upgrade_action_id > 0 && $this->table_exists( $wpdb->prefix . 'sn_project_actions' ) ) {
			$action_result = $wpdb->query( $wpdb->prepare( "UPDATE {$wpdb->prefix}sn_project_actions SET status='awaiting_payment',payment_attempts=payment_attempts+1,last_payment_at=%s,updated_at=%s WHERE id=%d", $now, $now, (int) $locked_op->upgrade_action_id ) );
			if ( false === $action_result ) {
				$wpdb->query( 'ROLLBACK' );
				return new WP_Error( 'sn_ops_stage_action', 'وضعیت مرحله پرداخت در اکشن پروژه ذخیره نشد: ' . ( $wpdb->last_error ?: 'خطای دیتابیس' ) );
			}
		}
		$wpdb->query( 'COMMIT' );
		$ctx = $this->item_context( (int) $locked_op->membership_item_id );
		$url = $this->invoice_url( $invoice_id );
		$code = (string) ( $invoice->invoice_code ?? '' );
		$sms_sent = $ctx && $url !== '' ? $this->send_invoice_sms( (string) $ctx->customer_phone, (string) $ctx->customer_name, $code, $url, $amount, (string) $locked_op->upgrade_target_name, $invoice_id ) : false;
		$this->event( (int) $locked_op->membership_item_id, 'operations_upgrade_payment_stage_issued', [ 'invoice_id'=>$invoice_id, 'stage_no'=>$next_stage, 'amount'=>$amount, 'remaining_before'=>$remaining, 'mode'=>$mode, 'sms_sent'=>$sms_sent ], $actor_id );
		return [ 'invoice_id'=>$invoice_id, 'stage_no'=>$next_stage, 'amount'=>$amount, 'remaining_before'=>$remaining, 'sms_sent'=>$sms_sent ];
	}

	private function create_upgrade_invoice( int $item_id, int $target_id, int $actor_id, string $source ) {
		global $wpdb; $table=$this->table(); $ctx=$this->item_context($item_id); $op=$this->ensure_operation($item_id,false);
		if(!$ctx||!$op||!in_array((string)$op->stage,[ 'awaiting_customer','sales_expert','upgrade_payment' ],true)){return new WP_Error('sn_ops_state','این کارت در این مرحله قابل ارتقا نیست.');}
		if($source==='customer'&&!$this->customer_owns_item($item_id,$actor_id)){return new WP_Error('sn_ops_scope','دسترسی مجاز نیست.');}
		if($source==='expert'&&!$this->can_work_case($op,$actor_id)){return new WP_Error('sn_ops_scope','این کارت به شما تخصیص ندارد.');}
		$option=$this->upgrade_option($ctx,$target_id); if(!$option){return new WP_Error('sn_ops_upgrade','گزینه افزایشی معتبر نیست یا اعتبار مقصد آن تعریف نشده است.');}
		if ( (float) $option['amount'] <= 0 ) { return $this->apply_free_upgrade( $item_id, $ctx, $option, $actor_id, $source ); }
		$invoice_table=$wpdb->prefix.'sn_invoices'; $item_table=$wpdb->prefix.'sn_invoice_items'; $stage_table=$wpdb->prefix.'sn_invoice_payment_stages'; $action_table=$wpdb->prefix.'sn_project_actions';
		foreach([$invoice_table,$item_table,$stage_table,$action_table] as $required){if(!$this->table_exists($required)){return new WP_Error('sn_ops_schema','جدول‌های فاکتور یا پروژه آماده نیستند.');}}
		try{$code=$this->generate_invoice_code();}catch(Throwable $e){return new WP_Error('sn_ops_code','ساخت کد فاکتور ممکن نشد.');}
		$token=SN_Helpers::generate_unique_access_token(); $short_code=in_array('short_code',$this->columns($invoice_table),true)?$this->generate_invoice_short_code():''; $now=current_time('mysql'); $amount=(float)$option['amount']; $seller_id=max(1,(int)$ctx->origin_user_id);
		$payment_request=$this->requested_initial_payment($amount,$source); if(!empty($payment_request['error'])&&is_wp_error($payment_request['error'])){return $payment_request['error'];} $payment_plan=(string)($payment_request['plan']??'full'); $initial_due=(float)($payment_request['due']??$amount);
		$source_invoice=(int)$ctx->source_invoice_id>0?$wpdb->get_row($wpdb->prepare("SELECT * FROM {$invoice_table} WHERE id=%d LIMIT 1",(int)$ctx->source_invoice_id)):null;
		$invoice_data=$this->filter_columns($invoice_table,[ 'invoice_code'=>$code,'access_token'=>$token,'short_code'=>$short_code,'seller_id'=>$seller_id,'customer_wp_id'=>(int)$ctx->customer_wp_id?:null,'customer_name'=>(string)$ctx->customer_name,'customer_phone'=>(string)$ctx->customer_phone,'product_id'=>(int)$option['target_product_id'],'product_price'=>$amount,'final_total'=>$amount,'original_total'=>$amount,'payment_plan'=>$payment_plan,'payment_total_amount'=>$amount,'current_due_amount'=>$initial_due,'paid_total_amount'=>0,'remaining_amount'=>$amount,'current_payment_stage'=>1,'payment_workflow_status'=>'awaiting_payment','current_stage_issued_by_user_id'=>$actor_id,'payment_card_number'=>(string)($source_invoice->payment_card_number??''),'payment_card_owner'=>(string)($source_invoice->payment_card_owner??''),'payment_card_sales_manager_user_id'=>(int)($source_invoice->payment_card_sales_manager_user_id??0)?:null,'payment_card_deputy_user_id'=>(int)($source_invoice->payment_card_deputy_user_id??0)?:null,'status'=>'pre_invoice','invoice_status'=>'pre_invoice','payment_status'=>'awaiting_payment','issued_by_user_id'=>$actor_id,'original_seller_id'=>$seller_id,'created_at'=>$now,'updated_at'=>$now ]);
		$wpdb->query('START TRANSACTION');
		$locked=$wpdb->get_row($wpdb->prepare("SELECT * FROM {$table} WHERE membership_item_id=%d FOR UPDATE",$item_id));
		if(!$locked||!in_array((string)$locked->stage,[ 'awaiting_customer','sales_expert','upgrade_payment' ],true)){$wpdb->query('ROLLBACK');return new WP_Error('sn_ops_state_changed','وضعیت کارت هم‌زمان تغییر کرده است؛ صفحه را تازه کنید.');}
		if($source==='expert'&&!$this->can_work_case($locked,$actor_id)){$wpdb->query('ROLLBACK');return new WP_Error('sn_ops_scope_changed','تخصیص این کارت تغییر کرده است؛ صفحه را تازه کنید.');}
		if((string)$locked->stage==='upgrade_payment'&&(int)$locked->upgrade_invoice_id>0){$existing_invoice_id=(int)$locked->upgrade_invoice_id;$existing_url=$this->invoice_url($existing_invoice_id);$existing_row=$wpdb->get_row($wpdb->prepare("SELECT * FROM {$invoice_table} WHERE id=%d",$existing_invoice_id));$existing_code=(string)($existing_row->invoice_code??'');$existing_due=$existing_row?$this->upgrade_payment_summary($existing_invoice_id):[];$sms_amount=max(0,(float)($existing_due['due']??0));if($sms_amount<=0.5){$sms_amount=max(0,(float)($existing_due['remaining']??0));}$wpdb->query('COMMIT');$sms_sent=$sms_amount>0.5&&$existing_url!==''?$this->send_invoice_sms((string)$ctx->customer_phone,(string)$ctx->customer_name,$existing_code,$existing_url,$sms_amount,(string)$locked->upgrade_target_name,$existing_invoice_id):false;return[ 'invoice_id'=>$existing_invoice_id,'url'=>$existing_url,'sms_sent'=>$sms_sent,'payment_plan'=>(string)($existing_row->payment_plan??'full'),'current_due_amount'=>$sms_amount ];}
		if(!$this->upgrade_snapshot_unchanged($item_id,$ctx,$option)){$wpdb->query('ROLLBACK');return new WP_Error('sn_ops_product_changed','اعتبار یا گزینه افزایشی این کارت تغییر کرده است؛ صفحه را تازه کنید.');}
		if(!$wpdb->insert($invoice_table,$invoice_data)){ $wpdb->query('ROLLBACK'); return new WP_Error('sn_ops_invoice','فاکتور افزایشی ذخیره نشد: '.($wpdb->last_error?:'خطای دیتابیس')); }
		$invoice_id=(int)$wpdb->insert_id;
		$ok_item=$wpdb->insert($item_table,$this->filter_columns($item_table,[ 'invoice_id'=>$invoice_id,'product_id'=>(int)$option['target_product_id'],'product_name'=>'ارتقای کارت '.$ctx->content_name_snapshot.' به '.$option['name'],'product_type'=>'project_upgrade','qty'=>1,'unit_price'=>$amount,'total_price'=>$amount,'is_free'=>0,'created_at'=>$now ]));
		$ok_stage=$wpdb->insert($stage_table,$this->filter_columns($stage_table,[ 'invoice_id'=>$invoice_id,'stage_no'=>1,'stage_type'=>$payment_plan,'requested_amount'=>$initial_due,'status'=>'pending','issued_by_user_id'=>$actor_id,'note'=>($payment_plan==='partial'?'پیش‌پرداخت اولیه افزایشی کارت #':'پرداخت کامل افزایشی کارت #').$item_id,'created_at'=>$now,'updated_at'=>$now ]));
		$ok_action=$wpdb->insert($action_table,$this->filter_columns($action_table,[ 'membership_item_id'=>$item_id,'actor_user_id'=>$actor_id,'action_type'=>'upsell','fulfillment_mode'=>$source==='customer'?'self':'expert','target_product_id'=>(int)$option['target_product_id'],'target_product_name'=>$option['name'],'base_credit'=>(float)$ctx->base_credit_snapshot,'target_credit'=>(float)$option['credit'],'amount'=>$amount,'status'=>'awaiting_payment','invoice_id'=>$invoice_id,'payment_attempts'=>1,'last_payment_at'=>$now,'note'=>'فاکتور افزایشی عملیات','created_at'=>$now,'updated_at'=>$now ])); $action_id=(int)$wpdb->insert_id;
		$ok_op=$wpdb->update($table,[ 'customer_choice'=>$source==='customer'?'self':((string)$op->customer_choice?:'expert'),'customer_choice_at'=>(string)$op->customer_choice_at?:$now,'activation_mode'=>'upsell','upgrade_target_product_id'=>(int)$option['target_product_id'],'upgrade_target_name'=>$option['name'],'upgrade_amount'=>$amount,'upgrade_invoice_id'=>$invoice_id,'upgrade_action_id'=>$action_id,'payment_state'=>'awaiting_payment','stage'=>'upgrade_payment','follow_up_at'=>null,'updated_at'=>$now ],[ 'membership_item_id'=>$item_id ]);
		if(!$ok_item||!$ok_stage||!$ok_action||false===$ok_op){$wpdb->query('ROLLBACK');return new WP_Error('sn_ops_invoice_parts','اجزای فاکتور افزایشی کامل ذخیره نشد: '.($wpdb->last_error?:'خطای دیتابیس'));}
		$wpdb->query('COMMIT');
		$url=$this->invoice_url($invoice_id); $sms_sent=$this->send_invoice_sms((string)$ctx->customer_phone,(string)$ctx->customer_name,$code,$url,$initial_due,(string)$option['name'],$invoice_id);
		$this->event($item_id,'operations_upgrade_invoice_created',[ 'invoice_id'=>$invoice_id,'amount'=>$amount,'payment_plan'=>$payment_plan,'initial_due'=>$initial_due,'target_credit'=>(float)$option['credit'],'source'=>$source,'sms_sent'=>$sms_sent ],$actor_id);
		return[ 'invoice_id'=>$invoice_id,'url'=>$url,'sms_sent'=>$sms_sent,'payment_plan'=>$payment_plan,'current_due_amount'=>$initial_due ];
	}

	private function invoice_url( int $invoice_id ): string {
		global $wpdb; $table=$wpdb->prefix.'sn_invoices'; $has_short_code=in_array('short_code',$this->columns($table),true); $select=$has_short_code?'invoice_code,access_token,short_code':'invoice_code,access_token';
		$row=$wpdb->get_row($wpdb->prepare("SELECT {$select} FROM {$table} WHERE id=%d",$invoice_id)); if(!$row){return '';}
		if($has_short_code&&empty($row->short_code)){$generated=$this->generate_invoice_short_code();$updated=$wpdb->update($table,[ 'short_code'=>$generated ],[ 'id'=>$invoice_id ]);if(false!==$updated){$row->short_code=(string)$wpdb->get_var($wpdb->prepare("SELECT short_code FROM {$table} WHERE id=%d",$invoice_id));}}
		if($has_short_code&&!empty($row->short_code)){return home_url('/i/'.rawurlencode((string)$row->short_code).'/');}
		$page_id=absint(get_option('sn_invoice_page_id',0)); $base=$page_id?(string)get_permalink($page_id):home_url('/invoice/');
		return add_query_arg([ 'invoice'=>(string)$row->invoice_code,'invoice_code'=>(string)$row->invoice_code,'access_token'=>(string)$row->access_token ],$base);
	}

	private function send_invoice_sms( string $phone, string $name, string $code, string $url, float $amount, string $option_title = '', int $invoice_id = 0 ): bool {
		$phone=SN_Helpers::normalize_mobile($phone); if(!SN_Helpers::is_valid_mobile($phone)){return false;}
		if(!class_exists('SN_SMS')){return false;}
		$card_number='';
		if((string)get_option('sn_card_to_card_enabled','1')==='1'){
			global $wpdb;
			if($invoice_id>0){$card_number=(string)$wpdb->get_var($wpdb->prepare("SELECT payment_card_number FROM {$wpdb->prefix}sn_invoices WHERE id=%d LIMIT 1",$invoice_id));}
			if($card_number===''){$card_number=(string)get_option('sn_card_number','');}
		}
		return (new SN_SMS())->send_invoice_link(
			$phone,
			sanitize_text_field($code),
			esc_url_raw($url),
			sanitize_text_field($name)?:'مشتری گرامی',
			number_format(max(0,$amount),0,'',','),
			$card_number
		);
	}

	public function is_upgrade_invoice( int $invoice_id ): bool { global $wpdb; $table=$this->table(); return $invoice_id>0&&$this->table_exists($table)&&(bool)$wpdb->get_var($wpdb->prepare("SELECT id FROM {$table} WHERE upgrade_invoice_id=%d LIMIT 1",$invoice_id)); }
	public function customer_return_url( int $invoice_id ): string { global $wpdb; $table=$this->table(); $item_id=(int)$wpdb->get_var($wpdb->prepare("SELECT membership_item_id FROM {$table} WHERE upgrade_invoice_id=%d LIMIT 1",$invoice_id)); return $item_id?$this->portal_url($item_id):''; }


	public function on_invoice_payment_stage_approved( int $invoice_id, $invoice = null, array $context = [] ): void {
		if ( ! $this->is_upgrade_invoice( $invoice_id ) || ! empty( $context['completed'] ) ) { return; }
		global $wpdb;
		$table = $this->table();
		$actions = $wpdb->prefix . 'sn_project_actions';
		$op = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$table} WHERE upgrade_invoice_id=%d LIMIT 1", $invoice_id ) );
		if ( ! $op || (string) $op->stage !== 'upgrade_payment' ) { return; }
		$now = current_time( 'mysql' );
		$remaining = max( 0, (float) ( $context['remaining_amount'] ?? 0 ) );
		$paid = max( 0, (float) ( $context['paid_total'] ?? 0 ) );
		$stage_no = max( 1, (int) ( $context['stage_no'] ?? 1 ) );
		$wpdb->update( $table, [ 'payment_state'=>'awaiting_next_payment', 'updated_at'=>$now ], [ 'id'=>(int) $op->id ] );
		if ( (int) $op->upgrade_action_id > 0 && $this->table_exists( $actions ) ) {
			$wpdb->query( $wpdb->prepare( "UPDATE {$actions} SET status='awaiting_payment',last_payment_at=%s,updated_at=%s WHERE id=%d", $now, $now, (int) $op->upgrade_action_id ) );
		}
		$this->event( (int) $op->membership_item_id, 'operations_upgrade_partial_payment_approved', [
			'invoice_id'=>$invoice_id,
			'stage_no'=>$stage_no,
			'paid_total'=>$paid,
			'remaining_amount'=>$remaining,
		], (int) ( $context['approved_by'] ?? 0 ) );
	}

	public function on_invoice_paid( int $invoice_id, $invoice = null ): void {
		if ( ! $this->is_upgrade_invoice( $invoice_id ) ) { return; }
		global $wpdb; $table = $this->table(); $actions = $wpdb->prefix . 'sn_project_actions'; $now = current_time( 'mysql' );
		$op = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$table} WHERE upgrade_invoice_id=%d LIMIT 1", $invoice_id ) );
		if ( ! $op || (int) $op->upgrade_action_id < 1 ) { return; }
		$target_credit = (float) $wpdb->get_var( $wpdb->prepare( "SELECT target_credit FROM {$actions} WHERE id=%d LIMIT 1", (int) $op->upgrade_action_id ) );
		if ( (float) $op->upgrade_amount <= 0 ) { return; }

		$wpdb->query( 'START TRANSACTION' );
		$locked = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$table} WHERE id=%d AND upgrade_invoice_id=%d LIMIT 1 FOR UPDATE", (int) $op->id, $invoice_id ) );
		if ( ! $locked ) { $wpdb->query( 'ROLLBACK' ); return; }
		$action_updated = $wpdb->query( $wpdb->prepare(
			"UPDATE {$actions} SET status='paid',last_payment_at=%s,updated_at=%s WHERE id=%d AND status IN ('awaiting_payment','retry_requested','payment_reversed','cancelled_continue_normal','cancelled_by_customer')",
			$now,
			$now,
			(int) $locked->upgrade_action_id
		) );
		$updated = $wpdb->query( $wpdb->prepare(
			"UPDATE {$table} SET activation_mode='upsell',current_credit=%f,payment_state='paid',stage='executive_manager_queue',follow_up_at=NULL,routed_at=%s,updated_at=%s WHERE id=%d AND upgrade_invoice_id=%d AND COALESCE(payment_state,'')<>'paid'",
			$target_credit,
			$now,
			$now,
			(int) $locked->id,
			$invoice_id
		) );
		if ( false === $action_updated || false === $updated ) { $wpdb->query( 'ROLLBACK' ); return; }
		$wpdb->query( 'COMMIT' );

		// Repeating the paid hook is harmless and also retries an interrupted
		// commission post. The project commission writer has its own unique key.
		if ( $updated || (string) $locked->payment_state === 'paid' ) {
			if ( class_exists( 'SN_Projects' ) ) { SN_Projects::instance()->credit_operations_upgrade_commission( (int) $locked->membership_item_id, $invoice_id, (float) $locked->upgrade_amount ); }
		}
		if ( $updated ) {
			$this->event( (int) $locked->membership_item_id, 'operations_upgrade_paid_routed_to_execution', [ 'invoice_id' => $invoice_id, 'current_credit' => $target_credit ], 0 );
			if ( class_exists( 'SN_Operations_Execution' ) ) { $execution_op=$wpdb->get_row($wpdb->prepare("SELECT * FROM {$table} WHERE id=%d LIMIT 1",(int)$locked->id));if($execution_op){SN_Operations_Execution::instance()->ensure_case($execution_op);} }
		}
	}

	public function on_invoice_reversed( int $invoice_id, $invoice = null, string $reason = '' ): void {
		if ( ! $this->is_upgrade_invoice( $invoice_id ) ) { return; }
		global $wpdb; $table = $this->table(); $actions = $wpdb->prefix . 'sn_project_actions'; $now = current_time( 'mysql' );
		$wpdb->query( 'START TRANSACTION' );
		$op = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$table} WHERE upgrade_invoice_id=%d LIMIT 1 FOR UPDATE", $invoice_id ) );
		if ( ! $op ) { $wpdb->query( 'ROLLBACK' ); return; }
		$back = (string) $op->customer_choice === 'expert' && (int) $op->sales_expert_user_id > 0 ? 'sales_expert' : 'upgrade_payment';
		$action_updated = true;
		if ( (int) $op->upgrade_action_id > 0 ) {
			$action_updated = false !== $wpdb->query( $wpdb->prepare(
				"UPDATE {$actions} SET status='payment_reversed',updated_at=%s WHERE id=%d AND status IN ('paid','fulfilled','awaiting_payment','retry_requested')",
				$now,
				(int) $op->upgrade_action_id
			) );
		}
		$updated = $wpdb->update( $table, [
			'current_credit' => (float) $op->base_credit,
			'payment_state' => 'rejected',
			'stage' => $back,
			'routed_at' => null,
			'updated_at' => $now,
		], [ 'id' => (int) $op->id, 'upgrade_invoice_id' => $invoice_id ] );
		if ( ! $action_updated || false === $updated ) { $wpdb->query( 'ROLLBACK' ); return; }
		$wpdb->query( 'COMMIT' );
		if ( $updated ) { $this->event( (int) $op->membership_item_id, 'operations_upgrade_payment_reversed', [ 'invoice_id' => $invoice_id, 'reason' => sanitize_textarea_field( $reason ) ], 0 ); }
	}

	/** Repair only objectively completed upgrade invoices whose paid hook stopped midway. */
	public function reconcile_paid_upgrades( int $limit = 20 ): void {
		global $wpdb; $table = $this->table(); $invoices = $wpdb->prefix . 'sn_invoices';
		if ( ! $this->table_exists( $table ) || ! $this->table_exists( $invoices ) ) { return; }
		if ( array_diff( [ 'remaining_amount', 'status', 'payment_workflow_status' ], $this->columns( $invoices ) ) ) { return; }
		$limit = min( 100, max( 1, $limit ) );
		$ids = $wpdb->get_col( $wpdb->prepare(
			"SELECT o.upgrade_invoice_id FROM {$table} o INNER JOIN {$invoices} i ON i.id=o.upgrade_invoice_id WHERE o.upgrade_invoice_id IS NOT NULL AND o.stage='upgrade_payment' AND COALESCE(o.payment_state,'')<>'paid' AND COALESCE(i.remaining_amount,0)<=0.5 AND (i.status IN ('paid','approved') OR i.payment_workflow_status='completed') ORDER BY o.id ASC LIMIT %d",
			$limit
		) ) ?: [];
		foreach ( $ids as $paid_invoice_id ) { $this->on_invoice_paid( (int) $paid_invoice_id ); }
	}

	private function operation_is_due( object $op ): bool {
		if ( (string) ( $op->stage ?? '' ) !== 'awaiting_customer' || (int) ( $op->auto_route_eligible ?? 0 ) !== 1 ) { return false; }
		if ( 0 === $this->inactivity_days() ) { return true; }
		$due = trim( (string) ( $op->decision_due_at ?? '' ) );
		return $due !== '' && $due <= current_time( 'mysql' );
	}

	private function route_operation_to_sales_manager( object $op, ?int $wait_days = null ): bool {
		if ( (string) ( $op->stage ?? '' ) !== 'awaiting_customer' || (int) ( $op->auto_route_eligible ?? 0 ) !== 1 ) { return false; }
		$op = $this->hydrate_operation_route( $op );
		global $wpdb; $table = $this->table(); $now = current_time( 'mysql' );
		$direct_supervisor_id = (int) ( $op->sales_supervisor_user_id ?? 0 );
		$destination_stage = $direct_supervisor_id > 0 ? 'sales_supervisor' : 'sales_manager_queue';
		if ( $direct_supervisor_id > 0 ) {
			$updated = $wpdb->query( $wpdb->prepare(
				"UPDATE {$table} SET customer_choice='auto_inactivity',customer_choice_at=%s,stage='sales_supervisor',supervisor_assigned_at=%s,routed_at=%s,updated_at=%s WHERE id=%d AND stage='awaiting_customer' AND auto_route_eligible=1",
				$now, $now, $now, $now, (int) $op->id
			) );
		} else {
			$updated = $wpdb->query( $wpdb->prepare(
				"UPDATE {$table} SET customer_choice='auto_inactivity',customer_choice_at=%s,stage='sales_manager_queue',supervisor_assigned_at=NULL,routed_at=%s,updated_at=%s WHERE id=%d AND stage='awaiting_customer' AND auto_route_eligible=1",
				$now, $now, $now, (int) $op->id
			) );
		}
		if ( $updated ) {
			$effective_wait_days = null === $wait_days ? $this->operation_wait_days( $op ) : max( 0, $wait_days );
			$this->event( (int) $op->membership_item_id, 'operations_auto_routed_after_inactivity', [
				'days' => $effective_wait_days,
				'immediate' => 0 === $effective_wait_days ? 1 : 0,
				'decision_due_at' => (string) ( $op->decision_due_at ?? '' ),
					'operations_sales_manager_user_id' => (int) ( $op->operations_sales_manager_user_id ?? 0 ),
					'operations_sales_supervisor_user_id' => $direct_supervisor_id,
					'destination_stage' => $destination_stage,
				'routing_source' => (string) ( $op->routing_source ?? 'unconfigured' ),
			], 0 );
			return true;
		}
		return false;
	}

	private function maybe_route_operation_if_due( object $op ): object {
		if ( ! $this->operation_is_due( $op ) ) { return $op; }
		$this->route_operation_to_sales_manager( $op );
		global $wpdb; $table = $this->table();
		return $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$table} WHERE id=%d LIMIT 1", (int) $op->id ) ) ?: $op;
	}

	public function route_inactive_cards( int $limit=200 ): void {
		global $wpdb; $table=$this->table(); if(!$this->table_exists($table)){return;} $now=current_time('mysql'); $limit=min(500,max(1,$limit));
		$items=$wpdb->prefix.'sn_project_membership_items';$members=$wpdb->prefix.'sn_project_memberships';
		if ( 0 === $this->inactivity_days() ) {
			$ids=$wpdb->get_col($wpdb->prepare("SELECT o.id FROM {$table} o INNER JOIN {$items} mi ON mi.id=o.membership_item_id INNER JOIN {$members} m ON m.id=mi.membership_id WHERE m.status='active' AND o.stage='awaiting_customer' AND o.auto_route_eligible=1 ORDER BY o.id ASC LIMIT %d",$limit))?:[];
		} else {
			$ids=$wpdb->get_col($wpdb->prepare("SELECT o.id FROM {$table} o INNER JOIN {$items} mi ON mi.id=o.membership_item_id INNER JOIN {$members} m ON m.id=mi.membership_id WHERE m.status='active' AND o.stage='awaiting_customer' AND o.auto_route_eligible=1 AND o.decision_due_at IS NOT NULL AND o.decision_due_at<=%s ORDER BY o.decision_due_at ASC LIMIT %d",$now,$limit))?:[];
		}
		foreach($ids as $id){$op=$wpdb->get_row($wpdb->prepare("SELECT * FROM {$table} WHERE id=%d",(int)$id)); if(!$op){continue;} $this->route_operation_to_sales_manager($op,0===$this->inactivity_days()?0:null);}
	}

	private function position_for_user( int $user_id ): string {
		global $wpdb; $positions=$wpdb->prefix.'sn_hr_positions'; $profiles=$wpdb->prefix.'sn_hr_profiles';
		if($this->table_exists($positions)&&$this->table_exists($profiles)){$slug=(string)$wpdb->get_var($wpdb->prepare("SELECT p.slug FROM {$profiles} hp INNER JOIN {$positions} p ON p.id=hp.position_id WHERE hp.user_id=%d AND COALESCE(hp.is_active,1)=1 LIMIT 1",$user_id)); if($slug!==''){return $slug;}}
		$user=get_user_by('id',$user_id); $roles=$user instanceof WP_User?(array)$user->roles:[];
		$map=[ 'sn_operations_sales_manager'=>'operations_sales_manager','sn_operations_sales_supervisor'=>'operations_sales_supervisor','sn_operations_sales_expert'=>'operations_sales_expert','sn_operations_executive_manager'=>'operations_executive_manager','sn_operations_execution_expert'=>'operations_execution_expert' ]; foreach($map as $role=>$position){if(in_array($role,$roles,true)){return $position;}} return '';
	}

	private function can_manage_sales( int $user_id ): bool { return current_user_can('manage_options')||current_user_can('sn_manage_operations_sales')||$this->position_for_user($user_id)==='operations_sales_manager'; }
	private function can_supervise_sales( int $user_id ): bool { return current_user_can('manage_options')||current_user_can('sn_supervise_operations_sales')||$this->position_for_user($user_id)==='operations_sales_supervisor'; }
	private function can_execute( int $user_id ): bool { return current_user_can('manage_options')||current_user_can('sn_manage_operations_execution')||$this->position_for_user($user_id)==='operations_executive_manager'; }
	private function can_work_case( object $op, int $user_id ): bool {
		if ( current_user_can( 'manage_options' ) ) { return true; }
		if ( (int) ( $op->sales_expert_user_id ?? 0 ) !== $user_id ) { return false; }
		$position = $this->position_for_user( $user_id );
		if ( current_user_can( 'sn_work_operations_sales' ) || $position === 'operations_sales_expert' ) { return true; }
		return $position === 'operations_sales_supervisor' && (int) ( $op->sales_supervisor_user_id ?? 0 ) === $user_id;
	}
	private function can_manage_case( object $op, int $user_id ): bool {
		if ( current_user_can( 'manage_options' ) ) { return true; }
		if ( $this->position_for_user( $user_id ) !== 'operations_sales_manager' && ! current_user_can( 'sn_manage_operations_sales' ) ) { return false; }
		return (int) ( $op->operations_sales_manager_user_id ?? 0 ) === $user_id;
	}
	private function can_supervise_case( object $op, int $user_id ): bool {
		return current_user_can( 'manage_options' ) || ( $this->can_supervise_sales( $user_id ) && (int) ( $op->sales_supervisor_user_id ?? 0 ) === $user_id );
	}

	private function operations_users( string $position ): array {
		$role_map = [
			'operations_sales_manager' => 'sn_operations_sales_manager',
			'operations_sales_supervisor' => 'sn_operations_sales_supervisor',
			'operations_sales_expert' => 'sn_operations_sales_expert',
			'operations_executive_manager' => 'sn_operations_executive_manager',
			'operations_execution_expert' => 'sn_operations_execution_expert',
		];
		$users = [];
		if ( isset( $role_map[ $position ] ) ) {
			foreach ( get_users( [ 'role' => $role_map[ $position ], 'number' => 2000, 'orderby' => 'display_name', 'order' => 'ASC' ] ) as $user ) {
				if ( $user instanceof WP_User ) { $users[ (int) $user->ID ] = $user; }
			}
		}
		global $wpdb; $positions=$wpdb->prefix.'sn_hr_positions';$profiles=$wpdb->prefix.'sn_hr_profiles';
		if($this->table_exists($positions)&&$this->table_exists($profiles)){$ids=$wpdb->get_col($wpdb->prepare("SELECT hp.user_id FROM {$profiles} hp INNER JOIN {$positions} p ON p.id=hp.position_id WHERE p.slug=%s AND COALESCE(hp.is_active,1)=1",$position))?:[]; foreach($ids as $id){$u=get_user_by('id',(int)$id);if($u instanceof WP_User){$users[(int)$u->ID]=$u;}}}
		uasort($users,static fn($a,$b)=>strcasecmp((string)$a->display_name,(string)$b->display_name));
		return array_values($users);
	}

	/** @deprecated Kept only for the inert 2.0.40 renderer during safe upgrades. */
	private function sales_experts(): array { return $this->operations_users( 'operations_sales_expert' ); }

	private function direct_reports( int $parent_user_id, string $child_position ): array {
		if ( current_user_can( 'manage_options' ) ) { return $this->operations_users( $child_position ); }
		global $wpdb; $profiles=$wpdb->prefix.'sn_hr_profiles';$positions=$wpdb->prefix.'sn_hr_positions';$assignments=$wpdb->prefix.'sn_hr_assignments';
		if(!$this->table_exists($profiles)||!$this->table_exists($positions)||!$this->table_exists($assignments)){return [];}
		$ids=$wpdb->get_col($wpdb->prepare("SELECT child.user_id FROM {$assignments} a INNER JOIN {$profiles} parent ON parent.id=a.parent_profile_id INNER JOIN {$profiles} child ON child.id=a.child_profile_id INNER JOIN {$positions} pos ON pos.id=child.position_id WHERE parent.user_id=%d AND a.relationship_type='reports_to' AND a.is_current=1 AND COALESCE(child.is_active,1)=1 AND pos.slug=%s ORDER BY child.user_id",$parent_user_id,$child_position))?:[];
		$out=[];foreach($ids as $id){$user=get_user_by('id',(int)$id);if($user instanceof WP_User){$out[(int)$user->ID]=$user;}}
		return array_values($out);
	}

	private function valid_direct_report( int $parent_user_id, int $child_user_id, string $child_position ): bool {
		foreach ( $this->direct_reports( $parent_user_id, $child_position ) as $user ) { if ( (int) $user->ID === $child_user_id ) { return true; } }
		return false;
	}

	private function normalize_followup_datetime( string $date_raw, string $legacy_time_raw = '' ) {
		$parsed = SN_Helpers::normalize_jalali_tehran_datetime( $date_raw, '09:00', false );
		if ( is_wp_error( $parsed ) ) { return $parsed; }
		$parsed['date'] = (string) $parsed['jalali'];
		return $parsed;
	}

	private function schedule_existing_followup( int $op_id, int $assignee_id, ?string $follow_up_at ): void {
		if ( $op_id < 1 || $assignee_id < 1 ) { return; }
		wp_clear_scheduled_hook( self::FOLLOWUP_HOOK, [ $op_id, $assignee_id ] );
	}

	private function handle_bulk_assignment( string $action, int $actor ): void {
		if ( ! in_array( $action, [ 'bulk_assign_supervisor', 'bulk_assign_expert' ], true ) ) { wp_die( 'عملیات تخصیص گروهی معتبر نیست.' ); }
		$raw_ids = isset( $_POST['operation_ids'] ) && is_array( $_POST['operation_ids'] ) ? wp_unslash( $_POST['operation_ids'] ) : [];
		if ( count( $raw_ids ) > 500 ) { $this->redirect_back( 'در هر تخصیص گروهی حداکثر ۵۰۰ کارت قابل پردازش است.', 'error' ); }
		$operation_ids = [];
		foreach ( $raw_ids as $raw_id ) {
			if ( ! is_scalar( $raw_id ) ) { continue; }
			$operation_id = absint( $raw_id );
			if ( $operation_id > 0 ) { $operation_ids[ $operation_id ] = $operation_id; }
		}
		$operation_ids = array_values( $operation_ids );
		if ( ! $operation_ids ) { $this->redirect_back( 'حداقل یک کارت را انتخاب کنید.', 'error' ); }

		$is_manager_assignment = 'bulk_assign_supervisor' === $action;
		if ( $is_manager_assignment ) {
			if ( ! $this->can_manage_sales( $actor ) ) { wp_die( 'دسترسی تخصیص گروهی مدیر عملیات مجاز نیست.' ); }
			$target_raw = $_POST['supervisor_user_id'] ?? 0;
			$target_id = is_scalar( $target_raw ) ? absint( $target_raw ) : 0;
			$target_position = 'operations_sales_supervisor';
			$target_label = 'سرپرست فروش عملیات';
		} else {
			if ( ! $this->can_supervise_sales( $actor ) ) { wp_die( 'دسترسی تخصیص گروهی سرپرست عملیات مجاز نیست.' ); }
			$target_raw = $_POST['expert_user_id'] ?? 0;
			$target_id = is_scalar( $target_raw ) ? absint( $target_raw ) : 0;
			$target_position = 'operations_sales_expert';
			$target_label = 'کارشناس فروش عملیات';
		}
		if ( $target_id < 1 ) { $this->redirect_back( $target_label . ' مقصد را انتخاب کنید.', 'error' ); }
		if ( ! current_user_can( 'manage_options' ) && ! $this->valid_direct_report( $actor, $target_id, $target_position ) ) {
			$this->redirect_back( $target_label . ' انتخاب‌شده زیرمجموعه مستقیم شما نیست.', 'error' );
		}

		global $wpdb; $table = $this->table(); $now = current_time( 'mysql' );
		$assigned = 0; $skipped = 0;
		foreach ( $operation_ids as $op_id ) {
			$op = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$table} WHERE id=%d LIMIT 1", $op_id ) );
			if ( ! $op ) { $skipped++; continue; }
			if ( $is_manager_assignment ) {
				$op = $this->hydrate_operation_route( $op );
				$manager_id = (int) ( $op->operations_sales_manager_user_id ?? 0 );
				if ( (string) $op->stage !== 'sales_manager_queue' || $manager_id < 1 || ! $this->can_manage_case( $op, $actor ) || ! $this->valid_direct_report( $manager_id, $target_id, $target_position ) ) { $skipped++; continue; }
				$updated = $wpdb->query( $wpdb->prepare(
					"UPDATE {$table} SET sales_supervisor_user_id=%d,supervisor_assigned_at=%s,stage='sales_supervisor',updated_at=%s WHERE id=%d AND stage='sales_manager_queue' AND operations_sales_manager_user_id=%d",
					$target_id, $now, $now, $op_id, $manager_id
				) );
				if ( $updated ) {
					$assigned++;
					$this->event( (int) $op->membership_item_id, 'operations_assigned_to_sales_supervisor', [ 'manager_user_id'=>$manager_id, 'supervisor_user_id'=>$target_id, 'bulk'=>1 ], $actor );
				if ( class_exists( 'SN_Purpose_Commission' ) ) { SN_Purpose_Commission::reconcile_operation( $op_id ); }
				} else { $skipped++; }
				continue;
			}

			$supervisor_id = (int) ( $op->sales_supervisor_user_id ?? 0 );
			if ( (string) $op->stage !== 'sales_supervisor' || $supervisor_id < 1 || ! $this->can_supervise_case( $op, $actor ) || ! $this->valid_direct_report( $supervisor_id, $target_id, $target_position ) ) { $skipped++; continue; }
			$updated = $wpdb->query( $wpdb->prepare(
				"UPDATE {$table} SET sales_expert_user_id=%d,assigned_by_user_id=%d,assigned_at=%s,stage='sales_expert',updated_at=%s WHERE id=%d AND stage='sales_supervisor' AND sales_supervisor_user_id=%d",
				$target_id, $actor, $now, $now, $op_id, $supervisor_id
			) );
			if ( $updated ) {
				$assigned++;
				$this->schedule_existing_followup( $op_id, $target_id, (string) ( $op->follow_up_at ?? '' ) );
				$this->event( (int) $op->membership_item_id, 'operations_assigned_to_sales_expert', [ 'supervisor_user_id'=>$supervisor_id, 'expert_user_id'=>$target_id, 'bulk'=>1 ], $actor );
				if ( class_exists( 'SN_Purpose_Commission' ) ) { SN_Purpose_Commission::reconcile_operation( $op_id ); }
			} else { $skipped++; }
		}

		$message = number_format_i18n( $assigned ) . ' کارت به ' . $target_label . ' تخصیص یافت.';
		if ( $skipped ) { $message .= ' ' . number_format_i18n( $skipped ) . ' کارت به‌دلیل تغییر وضعیت یا خارج‌بودن از محدوده رد شد.'; }
		$this->redirect_back( $message, $assigned ? 'success' : 'error' );
	}

	public function handle_staff_action(): void {
		if ( ! is_user_logged_in() ) { auth_redirect(); }
		$actor = get_current_user_id();
		$op_id = absint( $_POST['operation_id'] ?? 0 );
		$action = sanitize_key( wp_unslash( $_POST['operation_action'] ?? '' ) );
		if ( in_array( $action, [ 'bulk_assign_supervisor', 'bulk_assign_expert' ], true ) ) {
			check_admin_referer( 'sn_operations_bulk_assign' );
			$this->handle_bulk_assignment( $action, $actor );
			return;
		}
		check_admin_referer( 'sn_operations_staff_' . $op_id );
		global $wpdb; $table = $this->table(); $now = current_time( 'mysql' );
		$op = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$table} WHERE id=%d LIMIT 1", $op_id ) );
		if ( ! $op ) { wp_die( 'کارت عملیات یافت نشد.' ); }

		if ( $action === 'route_manager' ) {
			if ( ! current_user_can( 'manage_options' ) || (string) $op->stage !== 'sales_manager_queue' ) { wp_die( 'دسترسی غیرمجاز' ); }
			$manager_id = absint( $_POST['operations_manager_user_id'] ?? 0 );
			$valid = false;
			foreach ( $this->operations_users( 'operations_sales_manager' ) as $manager ) { if ( (int) $manager->ID === $manager_id ) { $valid = true; break; } }
			if ( ! $valid ) { $this->redirect_back( 'مدیر فروش عملیات معتبر نیست.', 'error' ); }
			$updated = $wpdb->query( $wpdb->prepare( "UPDATE {$table} SET operations_sales_manager_user_id=%d,routing_source='manual_admin',updated_at=%s WHERE id=%d AND stage='sales_manager_queue'", $manager_id, $now, $op_id ) );
			if ( $updated ) { $this->event( (int) $op->membership_item_id, 'operations_manager_routed_manually', [ 'operations_sales_manager_user_id' => $manager_id ], $actor ); if ( class_exists( 'SN_Purpose_Commission' ) ) { SN_Purpose_Commission::reconcile_operation( $op_id ); } }
			$this->redirect_back( $updated ? 'مسیر مدیر فروش عملیات ثبت شد.' : 'وضعیت کارت تغییر کرده است.', $updated ? 'success' : 'error' );
		}

		if ( $action === 'assign_supervisor' ) {
			$op = $this->hydrate_operation_route( $op );
			if ( ! $this->can_manage_case( $op, $actor ) || (string) $op->stage !== 'sales_manager_queue' ) { wp_die( 'این کارت در محدوده مدیر فروش عملیات شما نیست.' ); }
			$manager_id = (int) ( $op->operations_sales_manager_user_id ?? 0 );
			if ( $manager_id < 1 ) { $this->redirect_back( 'ابتدا مدیر فروش عملیات این کارت را مشخص کنید.', 'error' ); }
			$supervisor_id = absint( $_POST['supervisor_user_id'] ?? 0 );
			if ( ! $this->valid_direct_report( $manager_id, $supervisor_id, 'operations_sales_supervisor' ) ) { $this->redirect_back( 'سرپرست انتخاب‌شده زیرمجموعه این مدیر عملیات نیست.', 'error' ); }
			$updated = $wpdb->query( $wpdb->prepare(
				"UPDATE {$table} SET operations_sales_manager_user_id=%d,sales_supervisor_user_id=%d,supervisor_assigned_at=%s,stage='sales_supervisor',updated_at=%s WHERE id=%d AND stage='sales_manager_queue' AND operations_sales_manager_user_id=%d",
				$manager_id, $supervisor_id, $now, $now, $op_id, $manager_id
			) );
			if ( $updated ) { $this->event( (int) $op->membership_item_id, 'operations_assigned_to_sales_supervisor', [ 'manager_user_id' => $manager_id, 'supervisor_user_id' => $supervisor_id ], $actor ); if ( class_exists( 'SN_Purpose_Commission' ) ) { SN_Purpose_Commission::reconcile_operation( $op_id ); } }
			$this->redirect_back( $updated ? 'کارت به سرپرست فروش عملیات تخصیص یافت.' : 'وضعیت کارت هم‌زمان تغییر کرده است.', $updated ? 'success' : 'error' );
		}

		if ( $action === 'assign_expert' ) {
			if ( ! $this->can_supervise_case( $op, $actor ) || (string) $op->stage !== 'sales_supervisor' ) { wp_die( 'این کارت به سرپرستی شما تخصیص ندارد.' ); }
			$supervisor_id = (int) $op->sales_supervisor_user_id;
			$expert_id = absint( $_POST['expert_user_id'] ?? 0 );
			if ( ! $this->valid_direct_report( $supervisor_id, $expert_id, 'operations_sales_expert' ) ) { $this->redirect_back( 'کارشناس انتخاب‌شده زیرمجموعه این سرپرست نیست.', 'error' ); }
			$updated = $wpdb->query( $wpdb->prepare( "UPDATE {$table} SET sales_expert_user_id=%d,assigned_by_user_id=%d,assigned_at=%s,stage='sales_expert',updated_at=%s WHERE id=%d AND stage='sales_supervisor' AND sales_supervisor_user_id=%d", $expert_id, $actor, $now, $now, $op_id, $supervisor_id ) );
			if ( $updated ) {
				$this->schedule_existing_followup( $op_id, $expert_id, (string) ( $op->follow_up_at ?? '' ) );
				$this->event( (int) $op->membership_item_id, 'operations_assigned_to_sales_expert', [ 'supervisor_user_id' => $supervisor_id, 'expert_user_id' => $expert_id ], $actor );
				if ( class_exists( 'SN_Purpose_Commission' ) ) { SN_Purpose_Commission::reconcile_operation( $op_id ); }
			}
			$this->redirect_back( $updated ? 'کارت به کارشناس فروش عملیات تخصیص یافت.' : 'وضعیت کارت هم‌زمان تغییر کرده است.', $updated ? 'success' : 'error' );
		}

		if ( $action === 'assign_self' ) {
			if ( ! $this->can_supervise_case( $op, $actor ) || (string) $op->stage !== 'sales_supervisor' ) { wp_die( 'این کارت به سرپرستی شما تخصیص ندارد.' ); }
			$supervisor_id = (int) $op->sales_supervisor_user_id;
			$updated = $wpdb->query( $wpdb->prepare( "UPDATE {$table} SET sales_expert_user_id=%d,assigned_by_user_id=%d,assigned_at=%s,stage='sales_expert',updated_at=%s WHERE id=%d AND stage='sales_supervisor' AND sales_supervisor_user_id=%d", $actor, $actor, $now, $now, $op_id, $supervisor_id ) );
			if ( $updated ) {
				$this->schedule_existing_followup( $op_id, $actor, (string) ( $op->follow_up_at ?? '' ) );
				$this->event( (int) $op->membership_item_id, 'operations_supervisor_self_assigned', [ 'supervisor_user_id' => $supervisor_id ], $actor );
				if ( class_exists( 'SN_Purpose_Commission' ) ) { SN_Purpose_Commission::reconcile_operation( $op_id ); }
			}
			$this->redirect_back( $updated ? 'کارت به خود شما تخصیص یافت و آماده پیگیری است.' : 'وضعیت کارت هم‌زمان تغییر کرده است.', $updated ? 'success' : 'error' );
		}

		if ( $action === 'follow_up' ) {
			$allowed_stages = [ 'sales_manager_queue', 'sales_supervisor', 'sales_expert', 'upgrade_payment' ];
			$can_schedule = $this->can_manage_case( $op, $actor ) || $this->can_supervise_case( $op, $actor ) || $this->can_work_case( $op, $actor );
			if ( ! $can_schedule || ! in_array( (string) $op->stage, $allowed_stages, true ) ) { wp_die( 'ثبت زمان تماس برای این پرونده مجاز نیست.' ); }
			$parsed = $this->normalize_followup_datetime( (string) ( $_POST['followup_date'] ?? '' ) );
			if ( is_wp_error( $parsed ) ) { $this->redirect_back( $parsed->get_error_message(), 'error' ); }
			$note = sanitize_textarea_field( wp_unslash( $_POST['note'] ?? '' ) );
			$updated = $wpdb->query( $wpdb->prepare( "UPDATE {$table} SET contact_status='follow_up',follow_up_at=%s,follow_up_reminded_at=NULL,follow_up_reminder_attempts=0,last_note=%s,updated_at=%s WHERE id=%d AND stage IN ('sales_manager_queue','sales_supervisor','sales_expert','upgrade_payment')", $parsed['mysql'], $note, $now, $op_id ) );
			if ( false === $updated ) { $this->redirect_back( 'ذخیره زمان تماس به‌دلیل خطای پایگاه داده انجام نشد.', 'error' ); }
			$assignee_id = (int) ( $op->sales_expert_user_id ?? 0 );
			if ( $assignee_id > 0 ) { $this->schedule_existing_followup( $op_id, $assignee_id, $parsed['mysql'] ); }
			$this->event( (int) $op->membership_item_id, 'operations_follow_up_scheduled', [ 'follow_up_at' => $parsed['mysql'], 'timezone' => 'Asia/Tehran', 'scheduled_by_user_id' => $actor ], $actor );
			$this->redirect_back( 'زمان تماس با مشتری ثبت شد و در پروفایل او نمایش داده می‌شود.' );
		}

		if ( in_array( $action, [ 'no_answer', 'cancel', 'refer_normal', 'create_upgrade', 'retry_payment', 'issue_payment_stage', 'revert_normal' ], true ) ) {
			if ( ! $this->can_work_case( $op, $actor ) ) { wp_die( 'این کارت به شما تخصیص ندارد.' ); }
			$admin_override = current_user_can( 'manage_options' ) ? 1 : 0;
			$case_expert_id = (int) ( $op->sales_expert_user_id ?? 0 ) ?: $actor;
			if ( $action === 'no_answer' ) {
				$updated = $wpdb->query( $wpdb->prepare( "UPDATE {$table} SET contact_status='no_answer',no_answer_count=no_answer_count+1,last_contact_at=%s,last_note=%s,follow_up_at=NULL,follow_up_reminded_at=NULL,follow_up_reminder_attempts=0,updated_at=%s WHERE id=%d AND stage IN ('sales_expert','upgrade_payment') AND (sales_expert_user_id=%d OR %d=1)", $now, sanitize_textarea_field( wp_unslash( $_POST['note'] ?? '' ) ), $now, $op_id, $actor, $admin_override ) );
				if ( $updated ) {
					wp_clear_scheduled_hook( self::FOLLOWUP_HOOK, [ $op_id, $case_expert_id ] );
					$this->event( (int) $op->membership_item_id, 'operations_no_answer', [ 'count' => (int) $op->no_answer_count + 1 ], $actor );
				}
				$this->redirect_back( $updated ? 'عدم پاسخ ثبت شد و کارت در صف باقی ماند.' : 'وضعیت کارت تغییر کرده است.', $updated ? 'success' : 'error' );
			}
			if ( $action === 'cancel' ) {
				$reason = trim( sanitize_textarea_field( wp_unslash( $_POST['cancel_reason'] ?? '' ) ) );
				if ( $reason === '' ) { $this->redirect_back( 'ثبت دلیل انصراف اجباری است.', 'error' ); }
				$updated = $wpdb->query( $wpdb->prepare( "UPDATE {$table} SET stage='cancelled',contact_status='cancelled',cancel_reason=%s,cancelled_at=%s,cancelled_by_user_id=%d,last_contact_at=%s,last_note=%s,follow_up_at=NULL,updated_at=%s WHERE id=%d AND stage IN ('sales_expert','upgrade_payment') AND (sales_expert_user_id=%d OR %d=1)", $reason, $now, $actor, $now, $reason, $now, $op_id, $actor, $admin_override ) );
				if ( $updated && (int) $op->upgrade_action_id > 0 ) { $wpdb->query( $wpdb->prepare( "UPDATE {$wpdb->prefix}sn_project_actions SET status='cancelled_by_customer',note=%s,updated_at=%s WHERE id=%d AND status IN ('awaiting_payment','retry_requested','payment_reversed')", $reason, $now, (int) $op->upgrade_action_id ) ); }
				if ( $updated ) { wp_clear_scheduled_hook( self::FOLLOWUP_HOOK, [ $op_id, $case_expert_id ] ); $this->event( (int) $op->membership_item_id, 'operations_customer_cancelled', [ 'reason' => $reason ], $actor ); }
				$this->redirect_back( $updated ? 'انصراف با دلیل ثبت شد؛ پرونده در سوابق پنل باقی می‌ماند.' : 'وضعیت کارت تغییر کرده است.', $updated ? 'success' : 'error' );
			}
			if ( $action === 'refer_normal' || $action === 'revert_normal' ) {
				$updated = $this->route_normal_to_executive( (int) $op->membership_item_id, $actor, $action );
				$this->redirect_back( $updated ? 'کارت با اعتبار عادی برای مدیر اجرایی عملیات ارسال شد.' : 'وضعیت کارت تغییر کرده است.', $updated ? 'success' : 'error' );
			}
			if ( $action === 'create_upgrade' ) {
				$result = $this->create_upgrade_invoice( (int) $op->membership_item_id, absint( $_POST['target_product_id'] ?? 0 ), $actor, 'expert' );
				if ( is_wp_error( $result ) ) { $this->redirect_back( $result->get_error_message(), 'error' ); }
				if ( ! empty( $result['free'] ) ) { $this->redirect_back( 'ارتقای رایگان ثبت شد و کارت با اعتبار جدید برای اجرای عملیات ارسال شد.' ); }
				$staged = (string) ( $result['payment_plan'] ?? 'full' ) === 'partial';
				$ok_message = $staged ? 'مرحله اول پرداخت افزایشی صادر و لینک پرداخت برای مشتری ارسال شد.' : 'فاکتور افزایشی صادر و لینک پرداخت برای مشتری ارسال شد.';
				$this->redirect_back( ! empty( $result['sms_sent'] ) ? $ok_message : 'فاکتور صادر شد، اما ارسال پیامک ناموفق بود؛ از «ارسال مجدد لینک» استفاده کنید.', ! empty( $result['sms_sent'] ) ? 'success' : 'error' );
			}
			if ( $action === 'issue_payment_stage' ) {
				$mode = sanitize_key( (string) wp_unslash( $_POST['payment_stage_mode'] ?? 'partial' ) );
				$amount = $mode === 'remaining' ? 0 : $this->parse_money_value( $_POST['payment_stage_amount'] ?? 0 );
				$result = $this->issue_upgrade_payment_stage( $op, $actor, $mode, $amount );
				if ( is_wp_error( $result ) ) { $this->redirect_back( $result->get_error_message(), 'error' ); }
				$this->redirect_back( ! empty( $result['sms_sent'] ) ? 'مرحله بعد پرداخت صادر و لینک برای مشتری ارسال شد.' : 'مرحله بعد پرداخت صادر شد، اما ارسال پیامک ناموفق بود.', ! empty( $result['sms_sent'] ) ? 'success' : 'error' );
			}
			if ( $action === 'retry_payment' ) {
				$invoice_id = (int) $op->upgrade_invoice_id; $ctx = $this->item_context( (int) $op->membership_item_id ); $url = $this->invoice_url( $invoice_id );
				if ( ! $ctx || $url === '' ) { $this->redirect_back( 'فاکتور پرداخت پیدا نشد.', 'error' ); }
				$wpdb->query( $wpdb->prepare( "UPDATE {$wpdb->prefix}sn_project_actions SET status='retry_requested',payment_attempts=payment_attempts+1,last_payment_at=%s,updated_at=%s WHERE id=%d AND status IN ('awaiting_payment','retry_requested','payment_reversed')", $now, $now, (int) $op->upgrade_action_id ) );
				$code = (string) $wpdb->get_var( $wpdb->prepare( "SELECT invoice_code FROM {$wpdb->prefix}sn_invoices WHERE id=%d", $invoice_id ) );
				$payment = $this->upgrade_payment_summary( $invoice_id );
				$sms_amount = (float) ( $payment['due'] ?? 0 ); if ( $sms_amount <= 0.5 ) { $sms_amount = (float) $op->upgrade_amount; }
				$sms_sent = $this->send_invoice_sms( (string) $ctx->customer_phone, (string) $ctx->customer_name, $code, $url, $sms_amount, (string) $op->upgrade_target_name, $invoice_id );
				$this->redirect_back( $sms_sent ? 'لینک پرداخت دوباره ارسال شد.' : 'ارسال پیامک ناموفق بود؛ تنظیمات یا گزارش پیامک را بررسی کنید.', $sms_sent ? 'success' : 'error' );
			}
		}

		if ( in_array( $action, [ 'execution_start', 'execution_complete' ], true ) ) {
			if ( ! $this->can_execute( $actor ) ) { wp_die( 'دسترسی غیرمجاز' ); }
			if ( $action === 'execution_start' ) {
				$updated = $wpdb->query( $wpdb->prepare( "UPDATE {$table} SET stage='executive_in_progress',updated_at=%s WHERE id=%d AND stage='executive_manager_queue'", $now, $op_id ) );
				if ( $updated ) { $this->event( (int) $op->membership_item_id, 'operations_execution_started', [], $actor ); }
				$this->redirect_back( $updated ? 'اجرای کارت آغاز شد.' : 'وضعیت کارت تغییر کرده است.', $updated ? 'success' : 'error' );
			}
			$updated = $wpdb->query( $wpdb->prepare( "UPDATE {$table} SET stage='completed',completed_at=%s,completed_by_user_id=%d,updated_at=%s WHERE id=%d AND stage='executive_in_progress'", $now, $actor, $now, $op_id ) );
			if ( $updated ) {
				if ( (int) $op->upgrade_action_id ) { $wpdb->query( $wpdb->prepare( "UPDATE {$wpdb->prefix}sn_project_actions SET status='fulfilled',updated_at=%s WHERE id=%d AND status='paid'", $now, (int) $op->upgrade_action_id ) ); }
				$this->event( (int) $op->membership_item_id, 'operations_execution_completed', [ 'credit' => (float) $op->current_credit, 'mode' => (string) $op->activation_mode ], $actor );
			}
			$this->redirect_back( $updated ? 'اجرای کارت تکمیل شد.' : 'ابتدا اجرای کارت را آغاز کنید.', $updated ? 'success' : 'error' );
		}
		wp_die( 'عملیات نامعتبر است.' );
	}

	public function send_followup_reminder( int $op_id, int $expert_id ): void {
		wp_clear_scheduled_hook( self::FOLLOWUP_HOOK, [ $op_id, $expert_id ] );
	}

	private function sales_filter_labels(): array {
		return ['seller'=>'فروشنده','converter'=>'تبدیل‌کننده','supervisor'=>'سرپرست','senior_supervisor'=>'سرپرست ارشد','sales_manager'=>'مدیر فروش'];
	}

	private function sales_filter_person( int $id ): array {
		if ( $id < 1 ) { return ['position'=>'','parent'=>0,'name'=>'']; }
		if ( isset($this->sales_filter_people[$id]) ) { return $this->sales_filter_people[$id]; }
		$scope = class_exists('SN_Scope_Service') ? new SN_Scope_Service() : null;
		$hierarchy = class_exists('SN_Hierarchy_Service') ? new SN_Hierarchy_Service() : null;
		$position = $scope ? $scope->get_viewer_position_slug($id) : '';
		$parent = $hierarchy ? $hierarchy->get_current_parent_user_id($id) : 0;
		if (!$parent) {
			foreach (['sn_supervisor_id','sn_senior_supervisor_id','sn_sales_manager_id'] as $key) {
				$candidate = absint(get_user_meta($id,$key,true));
				if ($candidate && $candidate !== $id) { $parent=$candidate; break; }
			}
		}
		$user = get_userdata($id);
		return $this->sales_filter_people[$id] = ['position'=>$position,'parent'=>$parent,'name'=>$user ? (string)($user->display_name ?: $user->user_login) : ('کاربر #'.$id)];
	}

	/** Read provenance from the whole authorized scope, before the 500-card limit. */
	private function sales_filter_where( string $where ): string {
		global $wpdb;
		$table=$this->table(); $items=$wpdb->prefix.'sn_project_membership_items'; $members=$wpdb->prefix.'sn_project_memberships';
		$invoices=$wpdb->prefix.'sn_invoices'; $cases=$wpdb->prefix.'sn_dot_cases';
		$this->sales_filter_options=[]; $this->sales_filter_values=[];
		foreach ($this->sales_filter_labels() as $key=>$label) {
			$this->sales_filter_options[$key]=[];
			$value=$_GET['sn_ops_sales_'.$key] ?? 0;
			$this->sales_filter_values[$key]=is_scalar($value) ? absint($value) : 0;
		}
		$select='o.id,m.origin_user_id,m.origin_role,COALESCE(NULLIF(o.source_sales_manager_user_id,0),m.source_sales_manager_user_id,0) branch_manager';
		$joins='';
		if ($this->table_exists($invoices)) {
			$joins.=" LEFT JOIN {$invoices} fi ON fi.id=m.source_invoice_id";
			$cols=$this->columns($invoices);
			foreach (['original_seller_id','commission_seller_id','seller_id','converter_id'] as $col) {
				$select.=in_array($col,$cols,true) ? ",fi.{$col} invoice_{$col}" : ",0 invoice_{$col}";
			}
		} else { $select.=',0 invoice_original_seller_id,0 invoice_commission_seller_id,0 invoice_seller_id,0 invoice_converter_id'; }
		if ($this->table_exists($cases)) {
			$joins.=" LEFT JOIN {$cases} dc ON dc.id=m.dot_case_id";
			$select.=',dc.seller_id dot_seller_id,dc.converter_id dot_converter_id,dc.supervisor_id dot_supervisor_id';
		} else { $select.=',0 dot_seller_id,0 dot_converter_id,0 dot_supervisor_id'; }
		$source_rows=$wpdb->get_results("SELECT {$select} FROM {$table} o INNER JOIN {$items} mi ON mi.id=o.membership_item_id INNER JOIN {$members} m ON m.id=mi.membership_id {$joins} WHERE ({$where})") ?: [];
		$matching=[];
		foreach ($source_rows as $row) {
			$values=array_fill_keys(array_keys($this->sales_filter_labels()),0);
			$origin=(int)$row->origin_user_id;
			$origin_position=(string)$row->origin_role;
			if (str_starts_with($origin_position,'sn_')) { $origin_position=substr($origin_position,3); }
			$values['seller']=(int)($row->invoice_original_seller_id ?: $row->dot_seller_id ?: $row->invoice_commission_seller_id);
			$values['converter']=(int)($row->dot_converter_id ?: $row->invoice_converter_id);
			if (!$values['seller'] && $origin_position==='seller') { $values['seller']=$origin; }
			if (!$values['converter'] && $origin_position==='converter') { $values['converter']=$origin; }
			$issuer=(int)$row->invoice_seller_id;
			$issuer_person=$this->sales_filter_person($issuer);
			if (!$values['seller'] && $issuer_person['position']==='seller') { $values['seller']=$issuer; }
			if (!$values['converter'] && $issuer_person['position']==='converter') { $values['converter']=$issuer; }
			$current=$values['seller'] ?: $origin; $seen=[];
			for ($depth=0;$current>0 && $depth<20 && !isset($seen[$current]);$depth++) {
				$seen[$current]=true; $person=$this->sales_filter_person($current); $position=$person['position'];
				if (in_array($position,['supervisor','senior_supervisor','sales_manager'],true) && !$values[$position]) { $values[$position]=$current; }
				$current=(int)$person['parent'];
			}
			if (!$values['supervisor']) { $values['supervisor']=(int)$row->dot_supervisor_id; }
			if ((int)$row->branch_manager>0) { $values['sales_manager']=(int)$row->branch_manager; }
			$matches=true;
			foreach ($values as $key=>$id) {
				if ($id>0) { $person=$this->sales_filter_person($id); $this->sales_filter_options[$key][$id]=$person['name'].' (#'.$id.')'; }
				if ($this->sales_filter_values[$key]>0 && $this->sales_filter_values[$key]!==$id) { $matches=false; }
			}
			if ($matches) { $matching[]=(int)$row->id; }
		}
		foreach ($this->sales_filter_options as &$options) { asort($options,SORT_NATURAL); } unset($options);
		if (!array_filter($this->sales_filter_values)) { return $where; }
		return $where . ($matching ? ' AND o.id IN ('.implode(',',$matching).')' : ' AND 1=0');
	}

	private function render_card_filters(array $stages, array $card_types): string {
		ob_start(); ?>
		<div class="sn-ops-toolbar"><label><span>جستجو در کارت‌ها</span><input type="search" data-sn-ops-search placeholder="نام، موبایل، کارت یا مدیر فروش…" autocomplete="off"></label><label><span>فیلتر بر اساس مرحله</span><select data-sn-ops-status><option value="">همه مرحله‌ها</option><?php foreach ( $stages as $stage_key => $stage_label ) : ?><option value="<?php echo esc_attr( $stage_key ); ?>"><?php echo esc_html( $stage_label ); ?></option><?php endforeach; ?></select></label><label><span>نتیجه تماس</span><select data-sn-ops-contact-status><option value="">همه نتایج</option><option value="no_answer">جواب نداده</option><option value="follow_up">تماس مجدد</option><option value="cancelled">انصراف</option></select></label><label><span>فیلتر نوع کارت</span><select data-sn-ops-card-filter><option value="">همه کارت‌ها</option><?php foreach ( $card_types as $product_id => $card_name ) : ?><option value="<?php echo esc_attr( (string) $product_id ); ?>"><?php echo esc_html( $card_name ); ?></option><?php endforeach; ?></select></label></div>
		<?php return (string)ob_get_clean();
	}

	private function render_sales_filters(array $stages, array $card_types): string {
		$active=(bool)array_filter($this->sales_filter_values);
		ob_start(); ?>
		<details class="sn-ops-sales-filters" <?php echo $active ? 'open' : ''; ?>>
			<summary><span class="sn-ops-filter-symbol" aria-hidden="true"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7"><path d="M4 7h16M4 17h16"/><circle cx="9" cy="7" r="3" fill="currentColor" stroke="none"/><circle cx="16" cy="17" r="3" fill="currentColor" stroke="none"/></svg></span><span class="sn-ops-filter-heading"><strong>فیلتر</strong><span>جستجو و محدودکردن پرونده‌ها</span></span><?php if ($active) : ?><small>فعال</small><?php endif; ?><span class="sn-ops-filter-chevron" aria-hidden="true"></span></summary>
			<form method="get" action="<?php echo esc_url(get_permalink()); ?>">
				<?php if (isset($_GET['page_id']) && is_scalar($_GET['page_id'])) : ?><input type="hidden" name="page_id" value="<?php echo esc_attr((string)absint($_GET['page_id'])); ?>"><?php endif; ?>
				<div class="sn-ops-filter-section-title">عوامل فروش</div><p>برای نمایش پرونده‌های یک فرد یا تیم، گزینه‌های زیر را انتخاب کنید.</p>
				<div class="sn-ops-sales-filter-grid">
				<?php foreach ($this->sales_filter_labels() as $key=>$label) : $value=$this->sales_filter_values[$key] ?? 0; ?>
					<label><span><?php echo esc_html($label); ?></span><select name="<?php echo esc_attr('sn_ops_sales_'.$key); ?>">
					<option value="0">همه</option>
					<?php if ($value && !isset($this->sales_filter_options[$key][$value])) : ?><option value="<?php echo esc_attr((string)$value); ?>" selected>انتخاب فعلی — بدون پرونده در این پنل</option><?php endif; ?>
					<?php foreach ($this->sales_filter_options[$key] ?? [] as $id=>$name) : ?><option value="<?php echo esc_attr((string)$id); ?>" <?php selected($value,$id); ?>><?php echo esc_html($name); ?></option><?php endforeach; ?>
					</select></label>
				<?php endforeach; ?>
				</div>
				<div class="sn-ops-sales-filter-actions"><button type="submit">اعمال فیلتر</button><a href="<?php echo esc_url(get_permalink()); ?>">پاک‌کردن فیلترها</a></div>
			</form>
			<div class="sn-ops-filter-local"><div class="sn-ops-filter-section-title">جستجو در نتایج این صفحه</div><?php echo $this->render_card_filters($stages,$card_types); ?></div>
		</details>
		<?php return (string)ob_get_clean();
	}

	private function panel_rows( string $type, int $user_id ): array {
		global $wpdb; $table=$this->table(); $items=$wpdb->prefix.'sn_project_membership_items'; $members=$wpdb->prefix.'sn_project_memberships'; $users=$wpdb->users; $where='1=0';
		$is_admin = current_user_can( 'manage_options' );
		if ( $type === 'manager' ) {
			$where = $is_admin
				? "o.stage<>'awaiting_customer' AND (o.operations_sales_manager_user_id IS NOT NULL OR o.stage='sales_manager_queue')"
				: $wpdb->prepare( "o.operations_sales_manager_user_id=%d AND o.stage<>'awaiting_customer'", $user_id );
		} elseif ( $type === 'supervisor' ) {
			$where = $is_admin ? 'o.sales_supervisor_user_id IS NOT NULL' : $wpdb->prepare( 'o.sales_supervisor_user_id=%d', $user_id );
		} elseif ( $type === 'expert' ) {
			$where = $is_admin ? 'o.sales_expert_user_id IS NOT NULL' : $wpdb->prepare( 'o.sales_expert_user_id=%d', $user_id );
		} elseif ( $type === 'executive' ) {
			$where = "o.stage IN ('executive_manager_queue','execution_expert','execution_wallet_active','execution_form_review','shipping_queue','shipping_processing','shipped','delivered','shipping_not_sent','shipping_returned','shipping_cancelled','executive_in_progress','completed','execution_not_executed','execution_cancelled')";
		}
		if (in_array($type,['manager','supervisor'],true)) { $where=$this->sales_filter_where($where); }
		$sql = "SELECT o.*,mi.membership_id,mi.content_product_id,mi.content_name_snapshot,mi.upgrade_options_snapshot_json,m.customer_name,m.customer_phone,m.subscription_name_snapshot,m.activated_at,m.status membership_status,source_manager.display_name source_sales_manager_name,operations_manager.display_name operations_manager_name,supervisor.display_name supervisor_name,expert.display_name expert_name FROM {$table} o INNER JOIN {$items} mi ON mi.id=o.membership_item_id INNER JOIN {$members} m ON m.id=mi.membership_id LEFT JOIN {$users} source_manager ON source_manager.ID=o.source_sales_manager_user_id LEFT JOIN {$users} operations_manager ON operations_manager.ID=o.operations_sales_manager_user_id LEFT JOIN {$users} supervisor ON supervisor.ID=o.sales_supervisor_user_id LEFT JOIN {$users} expert ON expert.ID=o.sales_expert_user_id WHERE ({$where}) ORDER BY CASE o.stage WHEN 'sales_manager_queue' THEN 0 WHEN 'sales_supervisor' THEN 0 WHEN 'sales_expert' THEN 0 WHEN 'upgrade_payment' THEN 0 WHEN 'executive_manager_queue' THEN 1 WHEN 'executive_in_progress' THEN 1 WHEN 'completed' THEN 2 WHEN 'cancelled' THEN 3 ELSE 4 END,o.updated_at DESC,o.id DESC LIMIT 500";
		return $wpdb->get_results( $sql ) ?: [];
	}

	private function notice_html(): string {
		if ( empty( $_GET['sn_ops_notice'] ) ) { return ''; }
		$kind = sanitize_key( (string) ( $_GET['sn_ops_kind'] ?? 'success' ) );
		$error = $kind === 'error';
		return '<div class="sn-ops-notice is-' . esc_attr( $error ? 'error' : 'success' ) . '" role="status"><b>' . esc_html( $error ? 'ناموفق' : 'موفق' ) . '</b><button type="button" class="sn-ops-notice-close" data-sn-ops-notice-close aria-label="بستن اعلان">×</button></div>';
	}

	private function status_label( string $stage ): string { return[ 'awaiting_customer'=>'در انتظار انتخاب مشتری','sales_manager_queue'=>'در صف مدیر فروش عملیات','sales_supervisor'=>'در صف سرپرست فروش عملیات','sales_expert'=>'در حال پیگیری کارشناس','upgrade_payment'=>'در انتظار پرداخت افزایشی','executive_manager_queue'=>'آماده تخصیص اجرایی','execution_expert'=>'در حال اجرای کارشناس','execution_wallet_active'=>'کیف پول شارژ شد','execution_form_review'=>'فرم تکمیل و منتظر تصمیم','shipping_queue'=>'در صف ارسال','shipping_processing'=>'آماده‌سازی ارسال','shipped'=>'ارسال شد','delivered'=>'تحویل شد','shipping_not_sent'=>'ارسال نشد','shipping_returned'=>'مرجوع شد','shipping_cancelled'=>'لغو ارسال','executive_in_progress'=>'در حال اجرا','completed'=>'تکمیل‌شده','execution_not_executed'=>'عدم اجرا','execution_cancelled'=>'انصراف اجرایی','cancelled'=>'انصراف' ][$stage]??$stage; }
	public function customer_stage_label( string $stage ): string { return $this->status_label( sanitize_key( $stage ) ); }
	public function public_notice_html(): string { return $this->notice_html(); }
	private function mode_label( object $row ): string { return (string)$row->activation_mode==='upsell'?'افزایشی':((string)$row->activation_mode==='normal'?'عادی':'در انتظار تعیین حالت'); }
	private function routing_label( string $source ): string { return [ 'sales_manager_supervisor_route'=>'مسیر مستقیم شاخه فروش به سرپرست عملیات','sales_manager_route'=>'قانون قدیمی مدیر فروش','card_default'=>'مدیر پیش‌فرض کارت','manual_admin'=>'تعیین دستی مدیر سایت','legacy'=>'رکورد نسخه قبل','unconfigured'=>'مسیر تنظیم نشده' ][ $source ] ?? $source; }
	private function jalali_datetime( ?string $value ): string {
		$value = trim( (string) $value ); if ( $value === '' ) { return '—'; }
		$label = SN_Helpers::gregorian_to_jalali_date( $value );
		return preg_replace( '/\s+(\d{2}:\d{2})$/', ' — $1', $label ) ?: $label;
	}
	private function jalali_date( ?string $value ): string {
		$value = trim( (string) $value ); if ( $value === '' ) { return '—'; }
		try { $date = ( new DateTimeImmutable( $value, wp_timezone() ) )->setTimezone( SN_Helpers::tehran_timezone() ); }
		catch ( Throwable $e ) { return '—'; }
		return SN_Helpers::gregorian_to_jalali_input_value( $date->format( 'Y-m-d' ) );
	}
	private function followup_input_values( ?string $value ): array {
		$value = trim( (string) $value ); if ( $value === '' ) { return [ '' ]; }
		try { $date = new DateTimeImmutable( $value, wp_timezone() ); $date = $date->setTimezone( SN_Helpers::tehran_timezone() ); }
		catch ( Throwable $e ) { return [ '' ]; }
		return [ SN_Helpers::gregorian_to_jalali_input_value( $date->format( 'Y-m-d' ) ) ];
	}

	private function render_followup_editor( object $row, string $action_url, string $nonce ): string {
		if ( ! in_array( (string) $row->stage, [ 'sales_manager_queue', 'sales_supervisor', 'sales_expert', 'upgrade_payment' ], true ) ) { return ''; }
		[ $followup_date ] = $this->followup_input_values( (string) ( $row->follow_up_at ?? '' ) );
		ob_start(); ?>
		<details class="sn-ops-followup-editor"><summary><?php echo empty( $row->follow_up_at ) ? 'تعیین تاریخ تماس با مشتری' : 'ویرایش تاریخ تماس با مشتری'; ?></summary>
			<form method="post" action="<?php echo $action_url; ?>" class="sn-ops-form sn-ops-followup-form" data-sn-ops-followup>
				<input type="hidden" name="action" value="sn_operations_staff_action"><input type="hidden" name="operation_id" value="<?php echo esc_attr( (string) $row->id ); ?>"><input type="hidden" name="operation_action" value="follow_up"><?php echo $nonce; // phpcs:ignore WordPress.Security.EscapeOutput.OutputNotEscaped ?>
				<label>تاریخ شمسی<input class="sn-ops-jalali-date" name="followup_date" value="<?php echo esc_attr( $followup_date ); ?>" placeholder="۱۴۰۵/۰۶/۲۳" pattern="[۰-۹0-9]{4}[\/-][۰-۹0-9]{1,2}[\/-][۰-۹0-9]{1,2}" autocomplete="off" required></label>
				<textarea name="note" placeholder="یادداشت تماس (اختیاری)"><?php echo esc_textarea( (string) ( $row->last_note ?? '' ) ); ?></textarea>
				<small>تاریخ انتخابی در پروفایل مشتری هم نمایش داده می‌شود.</small><button type="submit" data-sn-ops-followup-submit>ذخیره تاریخ تماس</button>
			</form>
		</details>
		<?php return (string) ob_get_clean();
	}

	private function render_case_work_actions( object $row, array $options, string $action_url, string $nonce ): string {
		if ( ! in_array( (string) $row->stage, [ 'sales_expert', 'upgrade_payment' ], true ) ) { return ''; }
		$valid_upgrades = [];
		foreach ( $options as $option ) {
			$target = absint( $option['target_product_id'] ?? $option['product_id'] ?? 0 );
			$credit = (float) ( $option['target_credit'] ?? $option['credit'] ?? 0 );
			$price = (float) ( $option['upgrade_amount'] ?? $option['price'] ?? 0 );
			if ( $target && $credit > (float) $row->base_credit && $price > 0 ) {
				$valid_upgrades[] = [
					'target' => $target,
					'name' => (string) ( $option['name'] ?? 'افزایشی' ),
					'credit' => $credit,
					'price' => $price,
				];
			}
		}
		$payment = (string) $row->stage === 'upgrade_payment' ? $this->upgrade_payment_summary( (int) $row->upgrade_invoice_id ) : [];
		$presets = $this->partial_payment_presets();
		ob_start(); ?>
		<?php if ( ! empty( $payment['exists'] ) ) : ?>
			<div class="sn-ops-payment-summary">
				<span>مبلغ کل <strong><?php echo esc_html( SN_Helpers::format_price( (float) $payment['total'] ) ); ?></strong></span>
				<span>پرداخت تأییدشده <strong><?php echo esc_html( SN_Helpers::format_price( (float) $payment['paid'] ) ); ?></strong></span>
				<span>مانده <strong><?php echo esc_html( SN_Helpers::format_price( (float) $payment['remaining'] ) ); ?></strong></span>
				<span>مرحله جاری <strong><?php echo esc_html( number_format_i18n( (int) $payment['stage_no'] ) ); ?></strong></span>
				<?php if ( (float) $payment['due'] > 0.5 ) : ?><span>مبلغ مرحله جاری <strong><?php echo esc_html( SN_Helpers::format_price( (float) $payment['due'] ) ); ?></strong></span><?php endif; ?>
			</div>
		<?php endif; ?>
		<form method="post" action="<?php echo $action_url; ?>" class="sn-ops-form sn-ops-case-status-form" data-sn-ops-case-status>
			<input type="hidden" name="action" value="sn_operations_staff_action"><input type="hidden" name="operation_id" value="<?php echo esc_attr( (string) $row->id ); ?>"><?php echo $nonce; // phpcs:ignore WordPress.Security.EscapeOutput.OutputNotEscaped ?>
			<label class="sn-ops-status-main"><span>تعیین وضعیت پرونده</span><select name="operation_action" data-sn-ops-status-select required><option value="">— انتخاب وضعیت —</option><option value="no_answer">جواب نداده (<?php echo esc_html( number_format_i18n( (int) $row->no_answer_count ) ); ?> بار ثبت‌شده)</option><option value="follow_up">تماس مجدد</option><option value="cancel">انصراف مشتری</option><?php if ( (string) $row->stage === 'sales_expert' ) : ?><?php if ( (float) $row->base_credit > 0 ) : ?><option value="refer_normal">ارجاع عادی به واحد اجرا</option><?php endif; ?><?php if ( $valid_upgrades ) : ?><option value="create_upgrade">افزایشی و ارسال لینک پرداخت</option><?php endif; ?><?php else : ?><?php if ( ! empty( $payment['can_issue_next'] ) ) : ?><option value="issue_payment_stage">صدور مرحله بعد پرداخت</option><?php elseif ( (float) ( $payment['due'] ?? 0 ) > 0.5 ) : ?><option value="retry_payment">ارسال مجدد لینک مرحله جاری</option><?php endif; ?><?php if ( (float) $row->base_credit > 0 ) : ?><option value="revert_normal">بازگشت به عادی و ارجاع اجرا</option><?php endif; ?><?php endif; ?></select></label>
			<div class="sn-ops-status-fields" data-sn-ops-status-fields="no_answer" hidden><p>تعداد فعلی عدم پاسخ: <strong><?php echo esc_html( number_format_i18n( (int) $row->no_answer_count ) ); ?></strong></p><textarea name="note" placeholder="یادداشت اختیاری" disabled></textarea></div>
			<div class="sn-ops-status-fields" data-sn-ops-status-fields="follow_up" hidden><label>تاریخ شمسی<input class="sn-ops-jalali-date" name="followup_date" placeholder="۱۴۰۵/۰۶/۲۳" pattern="[۰-۹0-9]{4}[\/-][۰-۹0-9]{1,2}[\/-][۰-۹0-9]{1,2}" autocomplete="off" data-sn-ops-required disabled></label><textarea name="note" placeholder="یادداشت تماس (اختیاری)" disabled></textarea></div>
			<div class="sn-ops-status-fields" data-sn-ops-status-fields="cancel" hidden><textarea name="cancel_reason" placeholder="دلیل انصراف مشتری (اجباری)" minlength="3" data-sn-ops-required disabled></textarea></div>
			<?php if ( $valid_upgrades && (string) $row->stage === 'sales_expert' ) : ?>
				<div class="sn-ops-status-fields" data-sn-ops-status-fields="create_upgrade" hidden>
					<label><span>اعتبار افزایشی</span><select name="target_product_id" data-sn-ops-required disabled><option value="">انتخاب اعتبار افزایشی</option><?php foreach ( $valid_upgrades as $upgrade ) : ?><option value="<?php echo esc_attr( (string) $upgrade['target'] ); ?>" data-price="<?php echo esc_attr( (string) $upgrade['price'] ); ?>"><?php echo esc_html( $upgrade['name'] . ' · اعتبار ' . SN_Helpers::format_price( $upgrade['credit'] ) . ' · هزینه ' . SN_Helpers::format_price( $upgrade['price'] ) ); ?></option><?php endforeach; ?></select></label>
					<label><span>روش پرداخت</span><select name="payment_plan" data-sn-ops-payment-plan disabled><option value="full">پرداخت کامل</option><option value="partial">پرداخت مرحله‌ای</option></select></label>
					<div class="sn-ops-partial-payment" data-sn-ops-partial-payment hidden>
						<label><span>مبلغ مرحله اول</span><select name="prepayment_choice" data-sn-ops-prepayment-choice disabled><option value="">انتخاب مبلغ</option><?php foreach ( $presets as $preset ) : ?><option value="<?php echo esc_attr( (string) (int) round( $preset ) ); ?>"><?php echo esc_html( SN_Helpers::format_price( $preset ) ); ?></option><?php endforeach; ?><option value="custom">سایر / مبلغ دلخواه</option></select></label>
						<label data-sn-ops-prepayment-custom-wrap hidden><span>مبلغ دلخواه مرحله اول</span><input type="text" name="prepayment_custom" inputmode="numeric" data-sn-ops-money placeholder="مثلاً 2,000,000" disabled></label>
						<small>بعد از تأیید این مرحله، مانده روی همین فاکتور باقی می‌ماند و مرحله بعد را کارشناس فروش عملیات صادر می‌کند.</small>
					</div>
				</div>
			<?php endif; ?>
			<?php if ( (string) $row->stage === 'upgrade_payment' && ! empty( $payment['can_issue_next'] ) ) : ?>
				<div class="sn-ops-status-fields" data-sn-ops-status-fields="issue_payment_stage" hidden>
					<p>مانده قابل صدور: <strong><?php echo esc_html( SN_Helpers::format_price( (float) $payment['remaining'] ) ); ?></strong></p>
					<label><span>نوع مرحله جدید</span><select name="payment_stage_mode" data-sn-ops-stage-mode disabled><option value="partial">مبلغ دلخواه از مانده</option><option value="remaining">تسویه کل مانده</option></select></label>
					<label data-sn-ops-stage-amount-wrap><span>مبلغ مرحله جدید</span><input type="text" name="payment_stage_amount" inputmode="numeric" data-sn-ops-money data-remaining="<?php echo esc_attr( (string) $payment['remaining'] ); ?>" placeholder="مبلغ دلخواه" disabled></label>
				</div>
			<?php endif; ?>
			<button type="submit" data-sn-ops-status-submit disabled>ثبت وضعیت</button>
		</form>
		<?php return (string) ob_get_clean();
	}

	public function render_customer_card_controls( array $item, array $membership, WP_User $user ): string {
		$item_id=absint($item['id']??0);if(!$item_id||!$this->customer_owns_item($item_id,(int)$user->ID)){return '';} $op=$this->ensure_operation($item_id,false);if(!$op){return '<p class="sn-ops-inline-error">فلو عملیات این کارت هنوز آماده نشده است.</p>';} $op=$this->maybe_route_operation_if_due($op);
		$ctx = $this->item_context( $item_id );
		if ( $ctx && class_exists( 'SN_Operations_Execution' ) && SN_Operations_Execution::instance()->product_callback_only( (int) $ctx->content_product_id ) ) {
			return $this->render_customer_callback_controls( $item_id, $op );
		}
		$card_type = $this->customer_card_type( $item_id );
		if ( in_array( $card_type, [ 'wallet_charge', 'form', 'physical_invoice' ], true ) ) {
			$code = class_exists( 'SN_Operations_Execution' ) ? SN_Operations_Execution::instance()->customer_wallet_code( $item_id ) : '';
			ob_start(); ?>
			<div class="sn-customer-card-actions" id="sn-card-<?php echo esc_attr( (string) $item_id ); ?>">
			<?php if ( (string) $op->stage === 'awaiting_customer' ) : ?>
				<p>روش فعال‌سازی این کارت را انتخاب کنید:</p>
				<form method="post" action="<?php echo esc_url( admin_url( 'admin-post.php' ) ); ?>">
					<input type="hidden" name="action" value="sn_operations_customer_action"><input type="hidden" name="item_id" value="<?php echo esc_attr( (string) $item_id ); ?>"><input type="hidden" name="choice" value="normal">
					<?php wp_nonce_field( 'sn_operations_customer_' . $item_id ); ?><button type="submit">فعال‌سازی توسط خودم</button>
				</form>
				<details class="sn-customer-assisted-choice">
					<summary>فعال‌سازی توسط کارشناس</summary>
					<form method="post" action="<?php echo esc_url( admin_url( 'admin-post.php' ) ); ?>" class="sn-customer-expert-choice sn-ops-followup-form" data-sn-ops-followup>
						<input type="hidden" name="action" value="sn_operations_customer_action"><input type="hidden" name="item_id" value="<?php echo esc_attr( (string) $item_id ); ?>"><input type="hidden" name="choice" value="expert"><?php wp_nonce_field( 'sn_operations_customer_' . $item_id ); ?>
						<p>زمان مناسب تماس را از تقویم انتخاب کنید.</p><label><span>تاریخ تماس</span><input class="sn-ops-jalali-date" name="followup_date" placeholder="تاریخ را انتخاب کنید" pattern="[۰-۹0-9]{4}[\/-][۰-۹0-9]{1,2}[\/-][۰-۹0-9]{1,2}" autocomplete="off" required></label><button type="submit" data-sn-ops-followup-submit>ثبت درخواست تماس</button>
					</form>
				</details>
			<?php elseif ( $code !== '' && $card_type === 'wallet_charge' ) : ?>
				<div class="sn-customer-activation-result" role="status"><span class="sn-customer-activation-label">کد فعال‌سازی شما</span><strong class="sn-customer-activation-code" dir="ltr"><bdi><?php echo esc_html( $code ); ?></bdi></strong>
				</div>
			<?php elseif ( (string) $op->customer_choice === 'self' ) : ?>
				<div class="sn-customer-activation-pending" role="status"><?php if ( $card_type === 'wallet_charge' ) : ?><small>در حال حاضر کدی برای این کارت موجود نیست.</small><strong>کارشناسان با شما تماس خواهند گرفت.</strong><?php else : ?><strong>درخواست شما ثبت شد و کارشناسان ما با شما تماس خواهند گرفت.</strong><?php endif; ?></div>
			<?php else : ?><p>درخواست فعال‌سازی شما در حال پیگیری است.</p><?php endif; ?>
			</div>
			<?php return (string) ob_get_clean();
		}
		$options=$this->customer_upgrade_options_visible(absint($item['content_product_id']??0))?(array)($item['upgrade_options']??[]):[];$action=esc_url(admin_url('admin-post.php'));$nonce=wp_nonce_field('sn_operations_customer_'.$item_id,'_wpnonce',true,false);ob_start(); ?>
		<div class="sn-customer-card-actions" id="sn-card-<?php echo esc_attr((string)$item_id); ?>">
			<?php if((string)$op->stage==='awaiting_customer'): ?><p>روش فعال‌سازی این کارت را انتخاب کنید:</p><div class="sn-customer-choice-grid">
				<?php if((float)$op->base_credit>0 || $options): ?><details><summary>فعال‌سازی کارت توسط خودم</summary><div class="sn-customer-choice-body"><?php if((float)$op->base_credit>0): ?><div class="sn-customer-normal-box"><strong>حالت عادی</strong><span>اعتبار <?php echo esc_html(SN_Helpers::format_price((float)$op->base_credit)); ?></span><form method="post" action="<?php echo $action; ?>"><input type="hidden" name="action" value="sn_operations_customer_action"><input type="hidden" name="item_id" value="<?php echo esc_attr((string)$item_id); ?>"><input type="hidden" name="choice" value="normal"><?php echo $nonce; // phpcs:ignore ?><button type="submit">تأیید و ادامه عادی</button></form></div><?php endif; ?>
				<?php if($options): ?><div class="sn-customer-upgrade-box"><strong>حالت افزایشی</strong><?php foreach($options as $option): $target=absint($option['target_product_id']??$option['product_id']??0);$credit=max(0,(float)($option['target_credit']??$option['credit']??0));$price=max(0,(float)($option['upgrade_amount']??$option['price']??0));if(!$target||(!array_key_exists('upgrade_amount',$option)&&!array_key_exists('price',$option))){continue;} ?><form method="post" action="<?php echo $action; ?>"><input type="hidden" name="action" value="sn_operations_customer_action"><input type="hidden" name="item_id" value="<?php echo esc_attr((string)$item_id); ?>"><input type="hidden" name="choice" value="upgrade"><input type="hidden" name="target_product_id" value="<?php echo esc_attr((string)$target); ?>"><?php echo $nonce; // phpcs:ignore ?><span><?php echo esc_html((string)($option['name']??'ارتقای کارت')); ?> — اعتبار <?php echo esc_html(SN_Helpers::format_price($credit)); ?></span><small><?php echo $price > 0 ? 'هزینه تبدیل: ' . esc_html(SN_Helpers::format_price($price)) : 'ارتقای رایگان بدون پرداخت'; ?></small><button type="submit"><?php echo $price > 0 ? 'پرداخت و فعال‌سازی افزایشی' : 'ارتقای رایگان کارت'; ?></button></form><?php endforeach; ?></div><?php endif; ?></div></details><?php endif; ?>
				<form method="post" action="<?php echo $action; ?>" class="sn-customer-expert-choice sn-ops-followup-form" data-sn-ops-followup><input type="hidden" name="action" value="sn_operations_customer_action"><input type="hidden" name="item_id" value="<?php echo esc_attr((string)$item_id); ?>"><input type="hidden" name="choice" value="expert"><?php echo $nonce; // phpcs:ignore ?><strong>فعال‌سازی توسط کارشناس</strong><p>تاریخ مناسب تماس را انتخاب کنید.</p><label>تاریخ تماس<input class="sn-ops-jalali-date" name="followup_date" placeholder="۱۴۰۵/۰۶/۲۳" pattern="[۰-۹0-9]{4}[\/-][۰-۹0-9]{1,2}[\/-][۰-۹0-9]{1,2}" autocomplete="off" required></label><button type="submit" data-sn-ops-followup-submit>ثبت درخواست فعال‌سازی</button></form>
			</div><?php elseif((string)$op->stage==='upgrade_payment'): ?><div class="sn-customer-payment-pending"><strong>پرداخت افزایشی در انتظار تکمیل است</strong><p>هزینه: <?php echo esc_html(SN_Helpers::format_price((float)$op->upgrade_amount)); ?></p><div><form method="post" action="<?php echo $action; ?>"><input type="hidden" name="action" value="sn_operations_customer_action"><input type="hidden" name="item_id" value="<?php echo esc_attr((string)$item_id); ?>"><input type="hidden" name="choice" value="retry_upgrade"><?php echo $nonce; // phpcs:ignore ?><button type="submit">تلاش مجدد پرداخت</button></form><?php if((float)$op->base_credit>0): ?><form method="post" action="<?php echo $action; ?>"><input type="hidden" name="action" value="sn_operations_customer_action"><input type="hidden" name="item_id" value="<?php echo esc_attr((string)$item_id); ?>"><input type="hidden" name="choice" value="continue_normal"><?php echo $nonce; // phpcs:ignore ?><button type="submit" class="is-secondary">ادامه با حالت عادی</button></form><?php endif; ?></div></div><?php else: ?><div class="sn-customer-operation-state"><?php if((string)$op->activation_mode!==''): ?><b><?php echo esc_html($this->mode_label($op)); ?></b><?php endif; ?><span><?php echo esc_html($this->status_label((string)$op->stage)); ?></span><?php if((float)$op->current_credit>0): ?><small>اعتبار جاری: <?php echo esc_html(SN_Helpers::format_price((float)$op->current_credit)); ?></small><?php endif; ?></div><?php endif; ?>
			<?php if ( (string) $op->stage !== 'awaiting_customer' && ! empty( $op->follow_up_at ) ) : ?><div class="sn-customer-callback-time"><span>تاریخ تماس هماهنگ‌شده</span><strong><?php echo esc_html( $this->jalali_date( (string) $op->follow_up_at ) ); ?></strong></div><?php endif; ?>
		</div><?php return (string)ob_get_clean();
	}

	private function render_legacy_panel( string $type ): string {
		if(!is_user_logged_in()){return '<div class="sn-ops-panel" dir="rtl">برای ورود به پنل ابتدا وارد حساب شوید.</div>';}$uid=get_current_user_id();$allowed=$type==='manager'?$this->can_manage_sales($uid):($type==='expert'?(current_user_can('manage_options')||current_user_can('sn_work_operations_sales')||$this->position_for_user($uid)==='operations_sales_expert'):$this->can_execute($uid));if(!$allowed){return '<div class="sn-ops-panel" dir="rtl">دسترسی به این پنل مجاز نیست.</div>';}$rows=$this->panel_rows($type,$uid);$experts=$type==='manager'?$this->sales_experts():[];$title=$type==='manager'?'مدیر فروش عملیات':($type==='expert'?'کارشناس فروش عملیات':'مدیر اجرایی عملیات');$action=esc_url(admin_url('admin-post.php'));ob_start(); ?><div class="sn-ops-panel" dir="rtl"><header><div><span>بیاوین</span><h2><?php echo esc_html($title); ?></h2></div><b><?php echo esc_html(number_format_i18n(count($rows))); ?> کارت</b></header><?php echo $this->notice_html(); // phpcs:ignore ?><div class="sn-ops-list"><?php if(!$rows): ?><div class="sn-ops-empty">کارت فعالی در این صف نیست.</div><?php endif; ?><?php foreach($rows as $row):$nonce=wp_nonce_field('sn_operations_staff_'.(int)$row->id,'_wpnonce',true,false);$options=json_decode((string)$row->upgrade_options_snapshot_json,true);$options=is_array($options)?$options:[]; ?><article class="sn-ops-case"><div class="sn-ops-case-head"><div><small>#<?php echo esc_html((string)$row->id); ?> · <?php echo esc_html((string)$row->subscription_name_snapshot); ?></small><h3><?php echo esc_html((string)$row->content_name_snapshot); ?></h3><p><?php echo esc_html((string)$row->customer_name); ?> · <span dir="ltr"><?php echo esc_html((string)$row->customer_phone); ?></span></p></div><div class="sn-ops-badges"><b><?php echo esc_html($this->status_label((string)$row->stage)); ?></b><?php if((string)$row->activation_mode): ?><em class="is-<?php echo esc_attr((string)$row->activation_mode); ?>"><?php echo esc_html($this->mode_label($row)); ?></em><?php endif; ?></div></div><div class="sn-ops-credit"><span>اعتبار عادی <strong><?php echo esc_html(SN_Helpers::format_price((float)$row->base_credit)); ?></strong></span><span>اعتبار جاری <strong><?php echo esc_html(SN_Helpers::format_price((float)$row->current_credit)); ?></strong></span><?php if((int)$row->no_answer_count): ?><span>جواب نداده <strong><?php echo esc_html(number_format_i18n((int)$row->no_answer_count)); ?> بار</strong></span><?php endif; ?></div>
			<?php if($type==='manager'): ?><form method="post" action="<?php echo $action; ?>" class="sn-ops-form"><input type="hidden" name="action" value="sn_operations_staff_action"><input type="hidden" name="operation_id" value="<?php echo esc_attr((string)$row->id); ?>"><input type="hidden" name="operation_action" value="assign_expert"><?php echo $nonce; // phpcs:ignore ?><select name="expert_user_id" required><option value="">انتخاب کارشناس فروش عملیات</option><?php foreach($experts as $expert): ?><option value="<?php echo esc_attr((string)$expert->ID); ?>"><?php echo esc_html((string)$expert->display_name); ?></option><?php endforeach; ?></select><button type="submit">تخصیص</button></form><?php endif; ?>
			<?php if($type==='expert'): ?><div class="sn-ops-expert-actions"><details><summary>جواب نداده</summary><form method="post" action="<?php echo $action; ?>" class="sn-ops-form"><input type="hidden" name="action" value="sn_operations_staff_action"><input type="hidden" name="operation_id" value="<?php echo esc_attr((string)$row->id); ?>"><input type="hidden" name="operation_action" value="no_answer"><?php echo $nonce; // phpcs:ignore ?><textarea name="note" placeholder="یادداشت اختیاری"></textarea><button type="submit">ثبت بدون بایگانی</button></form></details><details><summary>تماس مجدد</summary><form method="post" action="<?php echo $action; ?>" class="sn-ops-form"><input type="hidden" name="action" value="sn_operations_staff_action"><input type="hidden" name="operation_id" value="<?php echo esc_attr((string)$row->id); ?>"><input type="hidden" name="operation_action" value="follow_up"><?php echo $nonce; // phpcs:ignore ?><input class="sn-ops-jalali-date" name="followup_date" placeholder="۱۴۰۵/۰۶/۲۳" pattern="[۰-۹0-9]{4}[\/-][۰-۹0-9]{1,2}[\/-][۰-۹0-9]{1,2}" required><textarea name="note" placeholder="یادداشت تماس"></textarea><button type="submit">ثبت تماس مجدد</button></form></details><?php if((string)$row->stage==='sales_expert'): ?><details><summary>ارتقای افزایشی کارت</summary><form method="post" action="<?php echo $action; ?>" class="sn-ops-form"><input type="hidden" name="action" value="sn_operations_staff_action"><input type="hidden" name="operation_id" value="<?php echo esc_attr((string)$row->id); ?>"><input type="hidden" name="operation_action" value="create_upgrade"><?php echo $nonce; // phpcs:ignore ?><select name="target_product_id" required><option value="">انتخاب اعتبار افزایشی</option><?php foreach($options as $option):$target=absint($option['target_product_id']??$option['product_id']??0);$credit=(float)($option['target_credit']??$option['credit']??0);$price=(float)($option['upgrade_amount']??$option['price']??0);if(!$target||(!array_key_exists('upgrade_amount',$option)&&!array_key_exists('price',$option))){continue;} ?><option value="<?php echo esc_attr((string)$target); ?>"><?php echo esc_html((string)($option['name']??'افزایشی').' · '.SN_Helpers::format_price($credit).' · '.($price>0?'هزینه '.SN_Helpers::format_price($price):'رایگان')); ?></option><?php endforeach; ?></select><button type="submit">ثبت ارتقای افزایشی</button></form></details><form method="post" action="<?php echo $action; ?>" class="sn-ops-form is-inline"><input type="hidden" name="action" value="sn_operations_staff_action"><input type="hidden" name="operation_id" value="<?php echo esc_attr((string)$row->id); ?>"><input type="hidden" name="operation_action" value="refer_normal"><?php echo $nonce; // phpcs:ignore ?><button type="submit" class="is-secondary">ارجاع عادی به مدیر اجرایی</button></form><?php else: ?><form method="post" action="<?php echo $action; ?>" class="sn-ops-form is-inline"><input type="hidden" name="action" value="sn_operations_staff_action"><input type="hidden" name="operation_id" value="<?php echo esc_attr((string)$row->id); ?>"><input type="hidden" name="operation_action" value="retry_payment"><?php echo $nonce; // phpcs:ignore ?><button type="submit">ارسال مجدد لینک پرداخت</button></form><form method="post" action="<?php echo $action; ?>" class="sn-ops-form is-inline"><input type="hidden" name="action" value="sn_operations_staff_action"><input type="hidden" name="operation_id" value="<?php echo esc_attr((string)$row->id); ?>"><input type="hidden" name="operation_action" value="revert_normal"><?php echo $nonce; // phpcs:ignore ?><button type="submit" class="is-secondary">برگشت به عادی و ارجاع</button></form><?php endif; ?></div><?php endif; ?>
			<?php if($type==='executive'&&(string)$row->stage!=='completed'): ?><form method="post" action="<?php echo $action; ?>" class="sn-ops-form is-inline"><input type="hidden" name="action" value="sn_operations_staff_action"><input type="hidden" name="operation_id" value="<?php echo esc_attr((string)$row->id); ?>"><input type="hidden" name="operation_action" value="<?php echo (string)$row->stage==='executive_manager_queue'?'execution_start':'execution_complete'; ?>"><?php echo $nonce; // phpcs:ignore ?><button type="submit"><?php echo (string)$row->stage==='executive_manager_queue'?'شروع اجرا':'ثبت تکمیل'; ?></button></form><?php endif; ?></article><?php endforeach; ?></div></div><?php return (string)ob_get_clean();
	}

	private function render_panel( string $type ): string {
		if ( ! is_user_logged_in() ) { return '<div class="sn-ops-panel" dir="rtl">برای ورود به پنل ابتدا وارد حساب شوید.</div>'; }
		$uid = get_current_user_id();
		$allowed = false;
		if ( $type === 'manager' ) { $allowed = $this->can_manage_sales( $uid ); }
		elseif ( $type === 'supervisor' ) { $allowed = $this->can_supervise_sales( $uid ); }
		elseif ( $type === 'expert' ) { $allowed = current_user_can( 'manage_options' ) || current_user_can( 'sn_work_operations_sales' ) || $this->position_for_user( $uid ) === 'operations_sales_expert'; }
		elseif ( $type === 'executive' ) { $allowed = $this->can_execute( $uid ); }
		if ( ! $allowed ) { return '<div class="sn-ops-panel" dir="rtl">دسترسی به این پنل مجاز نیست.</div>'; }

		$titles = [
			'manager' => [ 'مدیر فروش عملیات', 'مسیریابی و تخصیص کارت‌ها به سرپرستان فروش عملیات' ],
			'supervisor' => [ 'سرپرست فروش عملیات', 'تخصیص کارت‌های دریافتی به کارشناسان زیرمجموعه' ],
			'expert' => [ 'کارشناس فروش عملیات', 'پیگیری فعال‌سازی عادی یا افزایشی مشتری' ],
			'executive' => [ 'مدیر اجرایی عملیات', 'صف نهایی اجرای کارت‌های عادی و افزایشی' ],
		];
		$rows = $this->panel_rows( $type, $uid );
		$followups = 0; $completed = 0; $actionable = 0; $stages = [];
		foreach ( $rows as $row ) {
			$stage = (string) $row->stage; $stages[ $stage ] = $this->status_label( $stage );
			if ( ! empty( $row->follow_up_at ) && in_array( $stage, [ 'sales_manager_queue', 'sales_supervisor', 'sales_expert', 'upgrade_payment' ], true ) ) { $followups++; }
			if ( in_array( $stage, [ 'completed', 'cancelled' ], true ) ) { $completed++; }
			$supervisor_work = $type === 'supervisor' && in_array( $stage, [ 'sales_expert', 'upgrade_payment' ], true ) && (int) $row->sales_expert_user_id === $uid;
			if ( ( $type === 'manager' && $stage === 'sales_manager_queue' ) || ( $type === 'supervisor' && $stage === 'sales_supervisor' ) || $supervisor_work || ( $type === 'expert' && in_array( $stage, [ 'sales_expert', 'upgrade_payment' ], true ) ) || ( $type === 'executive' && $stage === 'executive_manager_queue' ) ) { $actionable++; }
		}
		asort( $stages, SORT_NATURAL );
		$action_url = esc_url( admin_url( 'admin-post.php' ) );
		$is_admin = current_user_can( 'manage_options' );
		$all_operations_managers = $is_admin ? $this->operations_users( 'operations_sales_manager' ) : [];
		$bulk_enabled = in_array( $type, [ 'manager', 'supervisor' ], true );
		$bulk_action = $type === 'manager' ? 'bulk_assign_supervisor' : 'bulk_assign_expert';
		$bulk_target_name = $type === 'manager' ? 'supervisor_user_id' : 'expert_user_id';
		$bulk_target_label = $type === 'manager' ? 'سرپرست فروش عملیات' : 'کارشناس فروش عملیات';
		$bulk_target_position = $type === 'manager' ? 'operations_sales_supervisor' : 'operations_sales_expert';
		$bulk_targets = $bulk_enabled ? ( $is_admin ? $this->operations_users( $bulk_target_position ) : $this->direct_reports( $uid, $bulk_target_position ) ) : [];
		$bulk_form_id = wp_unique_id( 'sn-ops-bulk-' . $type . '-' . $uid . '-' );
		$card_types = [];
		foreach ( $rows as $row ) { $card_types[ (int) $row->content_product_id ] = (string) $row->content_name_snapshot; }
		asort( $card_types, SORT_NATURAL );
		ob_start(); ?>
		<div class="sn-ops-panel sn-ops-panel-<?php echo esc_attr( $type ); ?>" dir="rtl">
			<header class="sn-ops-hero">
				<div class="sn-ops-hero-copy"><span class="sn-ops-eyebrow">چرخه بیاوین</span><h2><?php echo esc_html( $titles[ $type ][0] ); ?></h2><p><?php echo esc_html( $titles[ $type ][1] ); ?></p></div>
				<div class="sn-ops-hero-actions">
					<?php if (in_array($type,['manager','supervisor'],true)) { echo sn_bootstrap_plugin_instance(false)->sn_operations_password_actions_markup(); } ?>
					<div class="sn-ops-hero-count" aria-label="تعداد کارت‌های این نما"><strong><?php echo esc_html( number_format_i18n( count( $rows ) ) ); ?></strong><span>کارت</span></div>
					<?php if ( class_exists( 'SN_Purpose_Commission' ) ) { echo SN_Purpose_Commission::render_wallet_launcher( $uid, 'biavin', 20 ); } // phpcs:ignore WordPress.Security.EscapeOutput.OutputNotEscaped ?>
					<a class="sn-ops-logout" href="<?php echo esc_url( wp_logout_url( sn_crm_login_url() ) ); ?>"><span aria-hidden="true">↪</span><strong>خروج</strong></a>
				</div>
			</header>
			<?php echo $this->notice_html(); // phpcs:ignore WordPress.Security.EscapeOutput.OutputNotEscaped ?>
			<section class="sn-ops-kpis" aria-label="خلاصه وضعیت">
				<div><span>کل کارت‌ها</span><strong><?php echo esc_html( number_format_i18n( count( $rows ) ) ); ?></strong></div>
				<div><span>نیازمند اقدام من</span><strong><?php echo esc_html( number_format_i18n( $actionable ) ); ?></strong></div>
				<div><span>تاریخ تماس ثبت‌شده</span><strong><?php echo esc_html( number_format_i18n( $followups ) ); ?></strong></div>
				<div><span>مختومه در سوابق</span><strong><?php echo esc_html( number_format_i18n( $completed ) ); ?></strong></div>
			</section>
			<?php echo in_array($type,['manager','supervisor'],true) ? $this->render_sales_filters($stages,$card_types) : $this->render_card_filters($stages,$card_types); ?>
			<?php if ( $bulk_enabled ) : ?><section class="sn-ops-bulk-bar" data-sn-ops-bulk-bar>
				<div class="sn-ops-bulk-selectors"><label class="sn-ops-select-all"><input type="checkbox" data-sn-ops-select-all> انتخاب همه کارت‌های قابل تخصیص در نتایج فعلی</label><button type="button" data-sn-ops-select-visible>انتخاب همه نتایج فیلترشده</button><button type="button" class="is-secondary" data-sn-ops-clear-selection>پاک‌کردن انتخاب‌ها</button><span><strong data-sn-ops-selected-count>۰</strong> کارت انتخاب شده</span></div>
				<form id="<?php echo esc_attr( $bulk_form_id ); ?>" method="post" action="<?php echo $action_url; ?>" class="sn-ops-bulk-form" data-sn-ops-bulk-form>
					<input type="hidden" name="action" value="sn_operations_staff_action"><input type="hidden" name="operation_action" value="<?php echo esc_attr( $bulk_action ); ?>"><?php wp_nonce_field( 'sn_operations_bulk_assign' ); ?>
					<select name="<?php echo esc_attr( $bulk_target_name ); ?>" data-sn-ops-bulk-target required <?php disabled( ! $bulk_targets ); ?>><option value="">انتخاب <?php echo esc_html( $bulk_target_label ); ?></option><?php foreach ( $bulk_targets as $target ) : ?><option value="<?php echo esc_attr( (string) $target->ID ); ?>"><?php echo esc_html( (string) $target->display_name ); ?></option><?php endforeach; ?></select>
					<button type="submit" data-sn-ops-bulk-submit disabled>تخصیص گروهی</button><?php if ( ! $bulk_targets ) : ?><small>ابتدا نیروی زیرمجموعه را در HR تعریف کنید.</small><?php endif; ?>
				</form>
			</section><?php endif; ?>
			<?php if ( $type === 'executive' && class_exists( 'SN_Operations_Execution' ) ) { echo SN_Operations_Execution::instance()->render_manager_bulk( $rows, $uid, $bulk_form_id ); } // phpcs:ignore WordPress.Security.EscapeOutput.OutputNotEscaped ?>
			<div class="sn-ops-list" data-sn-ops-list>
				<?php if ( ! $rows ) : ?><div class="sn-ops-empty">کارت فعالی در این صف نیست.</div><?php endif; ?>
				<?php foreach ( $rows as $row ) :
					$nonce = wp_nonce_field( 'sn_operations_staff_' . (int) $row->id, '_wpnonce', true, false );
					$options = json_decode( (string) $row->upgrade_options_snapshot_json, true ); $options = is_array( $options ) ? $options : [];
					$bulk_selectable = ( $type === 'manager' && (string) $row->stage === 'sales_manager_queue' && (int) $row->operations_sales_manager_user_id > 0 ) || ( $type === 'supervisor' && (string) $row->stage === 'sales_supervisor' );
					$contact_status = sanitize_key( (string) ( $row->contact_status ?? '' ) );
					if ( $contact_status === '' ) {
						// Existing rows predate the dedicated contact_status column. Infer only
						// for display/filtering; no historic workflow data is rewritten.
						if ( (string) $row->stage === 'cancelled' ) { $contact_status = 'cancelled'; }
						elseif ( ! empty( $row->follow_up_at ) ) { $contact_status = 'follow_up'; }
						elseif ( (int) $row->no_answer_count > 0 ) { $contact_status = 'no_answer'; }
					}
					$search = implode( ' ', [ $row->customer_name, $row->customer_phone, $row->content_name_snapshot, $row->subscription_name_snapshot, $row->source_sales_manager_name, $row->operations_manager_name, $row->supervisor_name, $row->expert_name ] );
					$search = function_exists( 'mb_strtolower' ) ? mb_strtolower( $search, 'UTF-8' ) : strtolower( $search );
				?>
				<article class="sn-ops-case" data-sn-ops-card data-search="<?php echo esc_attr( $search ); ?>" data-status="<?php echo esc_attr( (string) $row->stage ); ?>" data-contact-status="<?php echo esc_attr( $contact_status ); ?>" data-card-type="<?php echo esc_attr( (string) (int) $row->content_product_id ); ?>">
					<div class="sn-ops-case-head">
						<div class="sn-ops-case-title"><?php if ( $bulk_selectable ) : ?><label class="sn-ops-card-select"><input type="checkbox" name="operation_ids[]" value="<?php echo esc_attr( (string) $row->id ); ?>" form="<?php echo esc_attr( $bulk_form_id ); ?>" data-sn-ops-bulk-checkbox><span>انتخاب</span></label><?php endif; ?><div class="sn-ops-case-summary-copy"><small class="sn-ops-case-reference">#<?php echo esc_html( (string) $row->id ); ?> · <?php echo esc_html( (string) $row->subscription_name_snapshot ); ?></small><h3><span>کارت</span><?php echo esc_html( (string) $row->content_name_snapshot ); ?></h3><div class="sn-ops-customer-identity"><span><small>مشتری</small><strong><?php echo esc_html( (string) $row->customer_name ); ?></strong></span><span><small>شماره تماس</small><a dir="ltr" href="tel:<?php echo esc_attr( (string) $row->customer_phone ); ?>"><?php echo esc_html( (string) $row->customer_phone ); ?></a></span><?php $header_credit = max( 0, (float) ( (string) $row->activation_mode === 'upsell' ? $row->current_credit : $row->base_credit ) ); if ( $header_credit > 0 ) : ?><span class="sn-ops-customer-credit"><small>اعتبار</small><strong><?php echo esc_html( SN_Helpers::format_price( $header_credit ) ); ?></strong></span><?php endif; ?></div></div></div>
						<div class="sn-ops-badges"><b><?php echo esc_html( $this->status_label( (string) $row->stage ) ); ?></b><?php if ( (string) $row->activation_mode !== '' ) : ?><em class="is-<?php echo esc_attr( (string) $row->activation_mode ); ?>"><?php echo esc_html( $this->mode_label( $row ) ); ?></em><?php endif; ?></div>
					</div>
					<div class="sn-ops-route-line">
						<span><small>مدیر فروش مبدأ</small><strong><?php echo esc_html( (string) ( $row->source_sales_manager_name ?: 'نامشخص' ) ); ?></strong></span>
						<span><small>مدیر فروش عملیات</small><strong><?php echo esc_html( (string) ( $row->operations_manager_name ?: 'تعیین نشده' ) ); ?></strong></span>
						<span><small>سرپرست عملیات</small><strong><?php echo esc_html( (string) ( $row->supervisor_name ?: 'تخصیص نشده' ) ); ?></strong></span>
						<span><small>کارشناس عملیات</small><strong><?php echo esc_html( (string) ( $row->expert_name ?: 'تخصیص نشده' ) ); ?></strong></span>
					</div>
					<div class="sn-ops-credit"><?php if ( (float) $row->base_credit > 0 ) : ?><span>اعتبار عادی <strong><?php echo esc_html( SN_Helpers::format_price( (float) $row->base_credit ) ); ?></strong></span><?php else : ?><span><strong>مسیر افزایشی</strong></span><?php endif; ?><?php if ( (float) $row->current_credit > 0 ) : ?><span>اعتبار جاری <strong><?php echo esc_html( SN_Helpers::format_price( (float) $row->current_credit ) ); ?></strong></span><?php endif; ?><span>ورود به چرخه <strong><?php echo esc_html( $this->jalali_datetime( (string) $row->activated_at ) ); ?></strong></span><?php if ( (int) $row->no_answer_count ) : ?><span>جواب نداده <strong><?php echo esc_html( number_format_i18n( (int) $row->no_answer_count ) ); ?> بار</strong></span><?php endif; ?></div>
					<div class="sn-ops-route-caption">مبنای مسیر: <?php echo esc_html( $this->routing_label( (string) ( $row->routing_source ?? 'legacy' ) ) ); ?></div>
					<?php if ( ! empty( $row->follow_up_at ) ) : ?><div class="sn-ops-followup-current"><strong>تاریخ تماس مجدد:</strong> <?php echo esc_html( $this->jalali_date( (string) $row->follow_up_at ) ); ?></div><?php endif; ?>
					<?php $chat_is_mine = in_array( $type, [ 'expert', 'supervisor' ], true ) && (int) $row->sales_expert_user_id === $uid; if ( $chat_is_mine ) : ?>
						<div class="sn-ops-chat-entry"><div><strong>گفت‌وگو با چرخه فروش</strong><small>طرف فروش: آخرین صادرکننده فاکتور اشتراک این پرونده</small></div><button type="button" class="sn-project-chat-btn sn-ops-chat-button" data-membership="<?php echo esc_attr( (string) $row->membership_id ); ?>" data-item="<?php echo esc_attr( (string) $row->membership_item_id ); ?>">چت با فروش</button></div>
					<?php endif; ?>

					<?php if ( $type === 'manager' ) : ?>
						<?php if ( (string) $row->stage === 'sales_manager_queue' && $is_admin && (int) $row->operations_sales_manager_user_id < 1 ) : ?>
							<div class="sn-ops-warning">برای این کارت مسیر مدیر فروش عملیات تنظیم نشده است.</div>
							<form method="post" action="<?php echo $action_url; ?>" class="sn-ops-form"><input type="hidden" name="action" value="sn_operations_staff_action"><input type="hidden" name="operation_id" value="<?php echo esc_attr( (string) $row->id ); ?>"><input type="hidden" name="operation_action" value="route_manager"><?php echo $nonce; // phpcs:ignore ?><select name="operations_manager_user_id" required><option value="">انتخاب مدیر فروش عملیات</option><?php foreach ( $all_operations_managers as $manager ) : ?><option value="<?php echo esc_attr( (string) $manager->ID ); ?>"><?php echo esc_html( (string) $manager->display_name ); ?></option><?php endforeach; ?></select><button type="submit">ثبت مسیر کارت</button></form>
						<?php elseif ( (string) $row->stage === 'sales_manager_queue' ) : $manager_id = (int) $row->operations_sales_manager_user_id ?: $uid; $supervisors = $this->direct_reports( $manager_id, 'operations_sales_supervisor' ); ?>
							<form method="post" action="<?php echo $action_url; ?>" class="sn-ops-form"><input type="hidden" name="action" value="sn_operations_staff_action"><input type="hidden" name="operation_id" value="<?php echo esc_attr( (string) $row->id ); ?>"><input type="hidden" name="operation_action" value="assign_supervisor"><?php echo $nonce; // phpcs:ignore ?><select name="supervisor_user_id" required <?php disabled( ! $supervisors ); ?>><option value="">انتخاب سرپرست فروش عملیات</option><?php foreach ( $supervisors as $supervisor ) : ?><option value="<?php echo esc_attr( (string) $supervisor->ID ); ?>"><?php echo esc_html( (string) $supervisor->display_name ); ?></option><?php endforeach; ?></select><button type="submit" <?php disabled( ! $supervisors ); ?>>تخصیص به سرپرست</button><?php if ( ! $supervisors ) : ?><small>ابتدا در HR یک سرپرست عملیات زیرمجموعه این مدیر تعریف کنید.</small><?php endif; ?></form>
						<?php endif; ?>
						<?php echo $this->render_followup_editor( $row, $action_url, $nonce ); // phpcs:ignore WordPress.Security.EscapeOutput.OutputNotEscaped ?>
					<?php elseif ( $type === 'supervisor' ) : ?>
						<?php if ( (string) $row->stage === 'sales_supervisor' ) : $experts = $this->direct_reports( (int) $row->sales_supervisor_user_id, 'operations_sales_expert' ); ?>
							<div class="sn-ops-assignment-options"><form method="post" action="<?php echo $action_url; ?>" class="sn-ops-form"><input type="hidden" name="action" value="sn_operations_staff_action"><input type="hidden" name="operation_id" value="<?php echo esc_attr( (string) $row->id ); ?>"><input type="hidden" name="operation_action" value="assign_expert"><?php echo $nonce; // phpcs:ignore ?><select name="expert_user_id" required <?php disabled( ! $experts ); ?>><option value="">انتخاب کارشناس فروش عملیات</option><?php foreach ( $experts as $expert ) : ?><option value="<?php echo esc_attr( (string) $expert->ID ); ?>"><?php echo esc_html( (string) $expert->display_name ); ?></option><?php endforeach; ?></select><button type="submit" <?php disabled( ! $experts ); ?>>تخصیص به کارشناس</button><?php if ( ! $experts ) : ?><small>کارشناس زیرمجموعه تعریف نشده؛ می‌توانید پرونده را خودتان بردارید.</small><?php endif; ?></form><form method="post" action="<?php echo $action_url; ?>" class="sn-ops-form is-inline sn-ops-self-assign"><input type="hidden" name="action" value="sn_operations_staff_action"><input type="hidden" name="operation_id" value="<?php echo esc_attr( (string) $row->id ); ?>"><input type="hidden" name="operation_action" value="assign_self"><?php echo $nonce; // phpcs:ignore ?><button type="submit">تخصیص پرونده به خودم</button></form></div>
						<?php endif; ?>
						<?php $supervisor_self_work = (int) $row->sales_supervisor_user_id === $uid && (int) $row->sales_expert_user_id === $uid; if ( ! $supervisor_self_work ) { echo $this->render_followup_editor( $row, $action_url, $nonce ); } // phpcs:ignore WordPress.Security.EscapeOutput.OutputNotEscaped ?>
						<?php if ( $supervisor_self_work ) { echo $this->render_case_work_actions( $row, $options, $action_url, $nonce ); } // phpcs:ignore WordPress.Security.EscapeOutput.OutputNotEscaped ?>
					<?php elseif ( $type === 'expert' ) : ?>
						<?php echo $this->render_case_work_actions( $row, $options, $action_url, $nonce ); // phpcs:ignore WordPress.Security.EscapeOutput.OutputNotEscaped ?>
					<?php elseif ( $type === 'executive' && class_exists( 'SN_Operations_Execution' ) ) : ?>
						<?php echo SN_Operations_Execution::instance()->render_manager_case( $row, $uid, $bulk_form_id ); // phpcs:ignore WordPress.Security.EscapeOutput.OutputNotEscaped ?>
					<?php endif; ?>
				</article>
				<?php endforeach; ?>
				<div class="sn-ops-empty" data-sn-ops-no-results hidden>نتیجه‌ای با عبارت جستجو پیدا نشد.</div>
			</div>
		</div>
		<?php return (string) ob_get_clean();
	}

	public function render_sales_manager_panel(): string { return $this->render_panel('manager'); }
	public function render_sales_supervisor_panel(): string { return $this->render_panel('supervisor'); }
	public function render_sales_expert_panel(): string { return $this->render_panel('expert'); }
	public function render_executive_manager_panel(): string { return $this->render_panel('executive'); }

	public function enqueue_assets(): void {
		global $post;if(!$post||!is_singular()){return;}$content=(string)($post->post_content??'');$elementor=(string)get_post_meta((int)$post->ID,'_elementor_data',true);$found=false;foreach([ 'sn_operations_sales_manager_panel','sn_operations_sales_supervisor_panel','sn_operations_sales_expert_panel','sn_operations_executive_manager_panel','sn_operations_execution_expert_panel','sn_project_manager_panel','sn_project_expert_panel','sn_shipping_panel','sn_biavin_customer_portal' ]as$sc){if(has_shortcode($content,$sc)||strpos($elementor,$sc)!==false){$found=true;break;}}if(!$found){return;}$css=SN_PLUGIN_DIR.'assets/css/operations-flow.css';$js=SN_PLUGIN_DIR.'assets/js/operations-flow.js';wp_enqueue_style('sn-operations-flow',SN_PLUGIN_URL.'assets/css/operations-flow.css',[],SN_VERSION.'-'.(file_exists($css)?filemtime($css):'0'));wp_enqueue_script('sn-operations-flow',SN_PLUGIN_URL.'assets/js/operations-flow.js',[],SN_VERSION.'-'.(file_exists($js)?filemtime($js):'0'),true);
	}
}
