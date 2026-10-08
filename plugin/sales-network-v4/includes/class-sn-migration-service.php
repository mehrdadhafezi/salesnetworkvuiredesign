<?php
if ( ! defined( 'ABSPATH' ) ) { exit; }

class SN_Migration_Service {
	const DB_VERSION = '2026-09-16-operations-sales-hr-seed-v1';

	private wpdb $wpdb;
	private array $steps = [];
	private array $errors = [];

	public function __construct() {
		global $wpdb;
		$this->wpdb = $wpdb;
	}

	public function get_db_version(): string {
		return (string) get_option( 'sn_db_version', '' );
	}

	public function migrate(): array {
		require_once ABSPATH . 'wp-admin/includes/upgrade.php';
		$this->steps = [];
		$this->errors = [];
		$this->create_hr_tables();
		$this->create_hierarchy_tables();
		$this->create_mis_tables();
		$this->create_distribution_engine_tables();
		$this->create_after_sales_tables();
		$this->create_commission_tables();
		$this->create_hr_compensation_tables();
		$this->seed_default_data();
		update_option( 'sn_db_version', self::DB_VERSION, false );
		$report = [
			'version' => self::DB_VERSION,
			'steps' => $this->steps,
			'first_error_message' => $this->errors[0]['message'] ?? '',
			'last_error_message' => $this->errors ? $this->errors[ count( $this->errors ) - 1 ]['message'] : '',
			'failed_table' => $this->errors[0]['table'] ?? '',
			'failed_operation' => $this->errors[0]['operation'] ?? '',
			'table_errors' => $this->errors,
			'positions_total' => $this->count_rows( 'sn_hr_positions' ),
			'levels_total' => $this->count_rows( 'sn_hr_levels' ),
			'mappings_total' => $this->count_rows( 'sn_hr_position_role_mappings' ),
			'mis_batches_total' => $this->count_rows( 'sn_mis_import_batches' ),
			'after_sales_cases_total' => $this->count_rows( 'sn_after_sales_cases' ),
			'commission_rules_total' => $this->count_rows( 'sn_commission_rules' ),
			'hr_compensation_profiles_total' => $this->count_rows( 'sn_hr_compensation_profiles' ),
			'hr_compensation_history_total' => $this->count_rows( 'sn_hr_compensation_history' ),
			'distribution_chains_total' => $this->count_rows( 'sn_distribution_chains' ),
			'distribution_items_total' => $this->count_rows( 'sn_distribution_items' ),
			'updated_at' => current_time( 'mysql' ),
		];
		update_option( 'sn_db_migration_report', $report, false );
		return $report;
	}

	public function repair_default_seed_data(): array {
		require_once ABSPATH . 'wp-admin/includes/upgrade.php';
		$this->steps = [];
		$this->errors = [];
		$this->create_hr_tables();
		$this->repair_legacy_key_columns();
		$this->seed_default_positions();
		$this->seed_default_levels();
		$this->seed_default_role_mappings();
		$this->seed_default_distribution_chain();
		$report = [
			'version' => self::DB_VERSION,
			'repair' => true,
			'steps' => $this->steps,
			'positions_total' => $this->count_rows( 'sn_hr_positions' ),
			'levels_total' => $this->count_rows( 'sn_hr_levels' ),
			'mappings_total' => $this->count_rows( 'sn_hr_position_role_mappings' ),
			'updated_at' => current_time( 'mysql' ),
		];
		update_option( 'sn_db_migration_report', $report, false );
		return $report;
	}

	public function mis_tables_complete(): bool {
		foreach ( [ 'sn_mis_import_batches', 'sn_mis_data_rows', 'sn_mis_lead_pool', 'sn_mis_distribution_rules', 'sn_mis_distribution_plans', 'sn_mis_distribution_plan_items', 'sn_mis_distribution_logs' ] as $table ) {
			if ( ! $this->table_exists( $table ) ) {
				return false;
			}
		}
		return true;
	}

	public function distribution_tables_complete(): bool {
		foreach ( [ 'sn_distribution_chains', 'sn_distribution_chain_steps', 'sn_distribution_items', 'sn_distribution_item_logs' ] as $table ) {
			if ( ! $this->table_exists( $table ) ) {
				return false;
			}
		}
		return true;
	}

	public function after_sales_tables_complete(): bool {
		foreach ( [ 'sn_after_sales_cases', 'sn_after_sales_case_logs' ] as $table ) {
			if ( ! $this->table_exists( $table ) ) {
				return false;
			}
		}
		return true;
	}

	public function commission_tables_complete(): bool {
		foreach ( [ 'sn_commission_rules', 'sn_commission_dry_runs', 'sn_commission_dry_run_items', 'sn_commission_wallet_posting_logs', 'sn_wallet_settlement_batches', 'sn_wallet_settlement_items' ] as $table ) {
			if ( ! $this->table_exists( $table ) ) {
				return false;
			}
		}
		return true;
	}

	public function hr_compensation_tables_complete(): bool {
		return $this->table_exists( 'sn_hr_compensation_profiles' ) && $this->table_exists( 'sn_hr_compensation_history' );
	}

	public function repair_mis_schema(): array {
		require_once ABSPATH . 'wp-admin/includes/upgrade.php';
		$this->steps = [];
		$this->errors = [];
		$this->create_mis_tables();
		$report = [
			'version' => self::DB_VERSION,
			'repair' => true,
			'tables' => $this->mis_schema_status(),
			'steps' => $this->steps,
			'first_error_message' => $this->errors[0]['message'] ?? '',
			'last_error_message' => $this->errors ? $this->errors[ count( $this->errors ) - 1 ]['message'] : '',
			'failed_table' => $this->errors[0]['table'] ?? '',
			'failed_operation' => $this->errors[0]['operation'] ?? '',
			'table_errors' => $this->errors,
			'updated_at' => current_time( 'mysql' ),
		];
		update_option( 'sn_mis_schema_repair_report', $report, false );
		return $report;
	}

	public function mis_schema_status(): array {
		$status = [];
		foreach ( [ 'sn_mis_import_batches', 'sn_mis_data_rows', 'sn_mis_lead_pool', 'sn_mis_distribution_rules', 'sn_mis_distribution_plans', 'sn_mis_distribution_plan_items', 'sn_mis_distribution_logs', 'sn_distribution_chains', 'sn_distribution_chain_steps', 'sn_distribution_items', 'sn_distribution_item_logs' ] as $table ) {
			$full_table = $this->normalize_table_name( $table );
			$exists = $this->table_exists( $full_table );
			$status[ $table ] = [
				'exists' => $exists,
				'rows' => $exists ? (int) $this->wpdb->get_var( "SELECT COUNT(*) FROM {$full_table}" ) : 0,
			];
		}
		return $status;
	}

	public function default_seed_data_is_complete(): bool {
		if ( ! $this->table_exists( 'sn_hr_positions' )
			|| ! $this->table_exists( 'sn_hr_levels' )
			|| ! $this->table_exists( 'sn_hr_position_role_mappings' )
			|| ! $this->table_exists( 'sn_distribution_chains' )
			|| ! $this->table_exists( 'sn_distribution_chain_steps' )
			|| $this->count_rows( 'sn_hr_levels' ) < 4 ) {
			return false;
		}

		$positions = $this->wpdb->prefix . 'sn_hr_positions';
		$mappings  = $this->wpdb->prefix . 'sn_hr_position_role_mappings';
		foreach ( array_keys( $this->default_positions() ) as $slug ) {
			if ( ! (int) $this->wpdb->get_var( $this->wpdb->prepare( "SELECT id FROM {$positions} WHERE slug=%s LIMIT 1", $slug ) ) ) {
				return false;
			}
		}
		foreach ( $this->default_role_mappings() as $legacy_role => $position_slug ) {
			$mapped = (string) $this->wpdb->get_var( $this->wpdb->prepare(
				"SELECT p.slug FROM {$mappings} m INNER JOIN {$positions} p ON p.id=m.position_id WHERE m.legacy_role=%s LIMIT 1",
				$legacy_role
			) );
			if ( $mapped !== $position_slug ) {
				return false;
			}
		}
		return true;
	}

	public function table_exists( string $table ): bool {
		$table = $this->normalize_table_name( $table );
		return $this->wpdb->get_var( $this->wpdb->prepare( 'SHOW TABLES LIKE %s', $table ) ) === $table;
	}

	public function column_exists( string $table, string $column ): bool {
		$table = $this->normalize_table_name( $table );
		if ( ! $this->table_exists( $table ) ) { return false; }
		$cols = $this->wpdb->get_col( "SHOW COLUMNS FROM {$table}", 0 );
		return in_array( $column, $cols, true );
	}

	public function index_exists( string $table, string $index ): bool {
		$table = $this->normalize_table_name( $table );
		if ( ! $this->table_exists( $table ) ) { return false; }
		return (bool) $this->wpdb->get_var( $this->wpdb->prepare( 'SHOW INDEX FROM ' . $table . ' WHERE Key_name = %s', $index ) );
	}

	public function create_table_if_missing( string $table, string $sql ): bool {
		$table = $this->normalize_table_name( $table );
		if ( $this->table_exists( $table ) ) {
			$this->log_step( 'table exists: ' . $table, 'skipped' );
			return true;
		}
		if ( ! function_exists( 'dbDelta' ) ) {
			require_once ABSPATH . 'wp-admin/includes/upgrade.php';
		}
		dbDelta( $sql );
		$exists = $this->table_exists( $table );
		if ( ! $exists ) {
			$error = $this->wpdb->last_error ? sanitize_text_field( $this->wpdb->last_error ) : 'dbDelta returned without creating table';
			$this->record_error( $table, 'dbDelta_create_table', $error, $sql );
			$this->log_step( 'create_table_if_missing dbDelta failed: ' . $table . ' - ' . $error, 'error', [
				'operation' => 'dbDelta_create_table',
				'table' => $table,
				'mysql_error' => $error,
				'sanitized_query' => $this->sanitize_sql_for_report( $sql ),
				'table_exists_after_operation' => 'no',
			] );
			$direct_sql = preg_replace( '/^CREATE TABLE\s+/i', 'CREATE TABLE IF NOT EXISTS ', trim( $sql ) );
			if ( $direct_sql ) {
				$this->wpdb->query( $direct_sql );
				$exists = $this->table_exists( $table );
				if ( ! $exists ) {
					$error = $this->wpdb->last_error ? sanitize_text_field( $this->wpdb->last_error ) : 'direct CREATE TABLE fallback did not create table';
					$this->record_error( $table, 'direct_create_table', $error, (string) $direct_sql );
					$this->log_step( 'create_table_if_missing direct fallback failed: ' . $table . ' - ' . $error, 'error', [
						'operation' => 'direct_create_table',
						'table' => $table,
						'mysql_error' => $error,
						'sanitized_query' => $this->sanitize_sql_for_report( (string) $direct_sql ),
						'table_exists_after_operation' => 'no',
					] );
				} else {
					$this->log_step( 'create_table_if_missing direct fallback: ' . $table, 'ok', [
						'operation' => 'direct_create_table',
						'table' => $table,
						'mysql_error' => '',
						'sanitized_query' => '',
						'table_exists_after_operation' => 'yes',
					] );
				}
			}
		}
		$this->log_step( 'create_table_if_missing: ' . $table, $exists ? 'ok' : 'error' );
		return $exists;
	}

	public function add_column_if_missing( string $table, string $column, string $definition ): bool {
		$table = $this->normalize_table_name( $table );
		if ( $this->column_exists( $table, $column ) ) {
			$this->log_step( 'column exists: ' . $table . '.' . $column, 'skipped' );
			return false;
		}
		$sql = "ALTER TABLE {$table} ADD COLUMN {$definition}";
		$this->wpdb->query( $sql );
		$this->log_step( 'add_column: ' . $table . '.' . $column, $this->wpdb->last_error ? 'error' : 'ok', [
			'operation' => 'add_column',
			'table' => $table,
			'mysql_error' => $this->wpdb->last_error ? sanitize_text_field( $this->wpdb->last_error ) : '',
			'sanitized_query' => $this->wpdb->last_error ? $this->sanitize_sql_for_report( $sql ) : '',
			'table_exists_after_operation' => $this->table_exists( $table ) ? 'yes' : 'no',
		] );
		if ( $this->wpdb->last_error ) {
			$this->record_error( $table, 'add_column_' . $column, sanitize_text_field( $this->wpdb->last_error ), $sql );
		}
		return ! (bool) $this->wpdb->last_error;
	}

	public function add_index_if_missing( string $table, string $index, string $definition ): bool {
		$table = $this->normalize_table_name( $table );
		if ( $this->index_exists( $table, $index ) ) {
			$this->log_step( 'index exists: ' . $table . '.' . $index, 'skipped' );
			return false;
		}
		$sql = "ALTER TABLE {$table} ADD {$definition}";
		$this->wpdb->query( $sql );
		$this->log_step( 'add_index: ' . $table . '.' . $index, $this->wpdb->last_error ? 'error' : 'ok', [
			'operation' => 'add_index',
			'table' => $table,
			'mysql_error' => $this->wpdb->last_error ? sanitize_text_field( $this->wpdb->last_error ) : '',
			'sanitized_query' => $this->wpdb->last_error ? $this->sanitize_sql_for_report( $sql ) : '',
			'table_exists_after_operation' => $this->table_exists( $table ) ? 'yes' : 'no',
		] );
		if ( $this->wpdb->last_error ) {
			$this->record_error( $table, 'add_index_' . $index, sanitize_text_field( $this->wpdb->last_error ), $sql );
		}
		return ! (bool) $this->wpdb->last_error;
	}

	public function log_step( string $message, string $status = 'ok', array $context = [] ): void {
		$this->steps[] = array_merge( [
			'message' => $message,
			'status' => $status,
			'time' => current_time( 'mysql' ),
		], $context );
	}

	private function record_error( string $table, string $operation, string $message, string $query = '' ): void {
		$this->errors[] = [
			'table' => $this->normalize_table_name( $table ),
			'operation' => sanitize_key( $operation ),
			'message' => sanitize_text_field( $message ),
			'sanitized_query' => $query ? $this->sanitize_sql_for_report( $query ) : '',
			'time' => current_time( 'mysql' ),
		];
	}

	private function sanitize_sql_for_report( string $sql ): string {
		$sql = preg_replace( '/\s+/', ' ', trim( $sql ) );
		return substr( sanitize_text_field( (string) $sql ), 0, 600 );
	}

	private function ensure_hr_lookup_schema(): void {
		$p = $this->wpdb->prefix;
		$positions = "{$p}sn_hr_positions";
		$levels = "{$p}sn_hr_levels";

		if ( $this->table_exists( $positions ) ) {
			$this->add_column_if_missing( $positions, 'slug', "slug varchar(80) NOT NULL DEFAULT ''" );
			$this->add_column_if_missing( $positions, 'label', "label varchar(120) NOT NULL DEFAULT ''" );
			$this->add_column_if_missing( $positions, 'description', 'description text NULL' );
			$this->add_column_if_missing( $positions, 'panel_key', 'panel_key varchar(80) DEFAULT NULL' );
			$this->add_column_if_missing( $positions, 'sort_order', 'sort_order int NOT NULL DEFAULT 0' );
			$this->add_column_if_missing( $positions, 'is_active', 'is_active tinyint(1) NOT NULL DEFAULT 1' );
			$this->add_column_if_missing( $positions, 'is_system', 'is_system tinyint(1) NOT NULL DEFAULT 0' );
			$this->add_column_if_missing( $positions, 'created_at', 'created_at datetime NULL' );
			$this->add_column_if_missing( $positions, 'updated_at', 'updated_at datetime NULL' );
			$this->backfill_hr_lookup_labels( $positions, 'position_key' );
		}

		if ( $this->table_exists( $levels ) ) {
			$this->add_column_if_missing( $levels, 'slug', "slug varchar(80) NOT NULL DEFAULT ''" );
			$this->add_column_if_missing( $levels, 'label', "label varchar(120) NOT NULL DEFAULT ''" );
			$this->add_column_if_missing( $levels, 'description', 'description text NULL' );
			$this->add_column_if_missing( $levels, 'sort_order', 'sort_order int NOT NULL DEFAULT 0' );
			$this->add_column_if_missing( $levels, 'is_active', 'is_active tinyint(1) NOT NULL DEFAULT 1' );
			$this->add_column_if_missing( $levels, 'is_system', 'is_system tinyint(1) NOT NULL DEFAULT 0' );
			$this->add_column_if_missing( $levels, 'created_at', 'created_at datetime NULL' );
			$this->add_column_if_missing( $levels, 'updated_at', 'updated_at datetime NULL' );
			$this->backfill_hr_lookup_labels( $levels, 'level_key' );
		}
	}

	private function backfill_hr_lookup_labels( string $table, string $legacy_key_column ): void {
		if ( ! $this->table_exists( $table ) || ! $this->column_exists( $table, 'label' ) || ! $this->column_exists( $table, 'slug' ) ) {
			return;
		}
		$candidates = [];
		foreach ( [ 'title', 'name', $legacy_key_column, 'slug' ] as $column ) {
			if ( $column && $this->column_exists( $table, $column ) ) {
				$candidates[] = $column;
			}
		}
		foreach ( $candidates as $column ) {
			$sql = "UPDATE {$table} SET label={$column} WHERE (label='' OR label IS NULL) AND {$column}<>'' AND {$column} IS NOT NULL";
			$this->wpdb->query( $sql );
			$this->log_step( 'backfill_lookup_label: ' . $table . '.' . $column, $this->wpdb->last_error ? 'error' : 'ok', [
				'operation' => 'backfill_lookup_label',
				'table' => $table,
				'mysql_error' => $this->wpdb->last_error ? sanitize_text_field( $this->wpdb->last_error ) : '',
				'sanitized_query' => $this->wpdb->last_error ? $this->sanitize_sql_for_report( $sql ) : '',
				'table_exists_after_operation' => $this->table_exists( $table ) ? 'yes' : 'no',
			] );
		}
	}

	private function create_hr_tables(): void {
		$charset = $this->wpdb->get_charset_collate();
		$p = $this->wpdb->prefix;

		$this->create_table_if_missing( "{$p}sn_hr_positions", "CREATE TABLE {$p}sn_hr_positions (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			slug VARCHAR(80) NOT NULL,
			label VARCHAR(120) NOT NULL,
			description TEXT DEFAULT NULL,
			panel_key VARCHAR(80) DEFAULT NULL,
			sort_order INT NOT NULL DEFAULT 0,
			is_active TINYINT(1) NOT NULL DEFAULT 1,
			is_system TINYINT(1) NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY  (id),
			UNIQUE KEY slug (slug),
			KEY is_active (is_active),
			KEY sort_order (sort_order)
		) {$charset};" );

		$this->create_table_if_missing( "{$p}sn_hr_levels", "CREATE TABLE {$p}sn_hr_levels (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			slug VARCHAR(80) NOT NULL,
			label VARCHAR(120) NOT NULL,
			description TEXT DEFAULT NULL,
			sort_order INT NOT NULL DEFAULT 0,
			is_active TINYINT(1) NOT NULL DEFAULT 1,
			is_system TINYINT(1) NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY  (id),
			UNIQUE KEY slug (slug),
			KEY is_active (is_active),
			KEY sort_order (sort_order)
		) {$charset};" );

		$this->ensure_hr_lookup_schema();

		$this->create_table_if_missing( "{$p}sn_hr_profiles", "CREATE TABLE {$p}sn_hr_profiles (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			user_id BIGINT UNSIGNED NOT NULL,
			position_id BIGINT UNSIGNED DEFAULT NULL,
			level_id BIGINT UNSIGNED DEFAULT NULL,
			employment_status VARCHAR(50) NOT NULL DEFAULT 'active',
			contract_status VARCHAR(50) DEFAULT NULL,
			training_status VARCHAR(50) DEFAULT NULL,
			legacy_role VARCHAR(120) DEFAULT NULL,
			legacy_source VARCHAR(120) DEFAULT NULL,
			is_active TINYINT(1) NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY  (id),
			UNIQUE KEY user_id (user_id),
			KEY position_id (position_id),
			KEY level_id (level_id),
			KEY legacy_role (legacy_role),
			KEY is_active (is_active)
		) {$charset};" );

		$this->create_table_if_missing( "{$p}sn_hr_position_role_mappings", "CREATE TABLE {$p}sn_hr_position_role_mappings (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			legacy_role VARCHAR(120) NOT NULL,
			position_id BIGINT UNSIGNED NOT NULL,
			default_level_id BIGINT UNSIGNED DEFAULT NULL,
			is_active TINYINT(1) NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY  (id),
			UNIQUE KEY legacy_role (legacy_role),
			KEY position_id (position_id),
			KEY default_level_id (default_level_id),
			KEY is_active (is_active)
		) {$charset};" );

		$this->create_table_if_missing( "{$p}sn_hr_profile_logs", "CREATE TABLE {$p}sn_hr_profile_logs (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			profile_id BIGINT UNSIGNED DEFAULT NULL,
			user_id BIGINT UNSIGNED DEFAULT NULL,
			actor_user_id BIGINT UNSIGNED DEFAULT NULL,
			action VARCHAR(120) NOT NULL,
			old_value LONGTEXT DEFAULT NULL,
			new_value LONGTEXT DEFAULT NULL,
			context LONGTEXT DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY  (id),
			KEY profile_id (profile_id),
			KEY user_id (user_id),
			KEY actor_user_id (actor_user_id),
			KEY action (action),
			KEY created_at (created_at)
		) {$charset};" );

		if ( $this->table_exists( "{$p}sn_hr_profiles" ) ) {
			foreach ( [
				'position_id' => 'position_id bigint unsigned DEFAULT NULL',
				'level_id' => 'level_id bigint unsigned DEFAULT NULL',
				'employee_code' => 'employee_code varchar(100) NULL',
				'employment_status' => "employment_status varchar(50) NOT NULL DEFAULT 'active'",
				'employment_type' => 'employment_type varchar(50) NULL',
				'hire_date' => 'hire_date date NULL',
				'termination_date' => 'termination_date date NULL',
				'base_salary' => 'base_salary bigint unsigned NULL',
				'salary_currency' => "salary_currency varchar(20) NOT NULL DEFAULT 'IRT'",
				'national_id' => 'national_id varchar(50) NULL',
				'work_phone' => 'work_phone varchar(50) NULL',
				'emergency_phone' => 'emergency_phone varchar(50) NULL',
				'department_id' => 'department_id bigint unsigned NULL',
				'unit_id' => 'unit_id bigint unsigned NULL',
				'team_id' => 'team_id bigint unsigned NULL',
				'notes' => 'notes text NULL',
			] as $column => $definition ) {
				$this->add_column_if_missing( "{$p}sn_hr_profiles", $column, $definition );
			}
			foreach ( [
				'position_id' => 'KEY position_id (position_id)',
				'level_id' => 'KEY level_id (level_id)',
				'employee_code' => 'KEY employee_code (employee_code)',
				'employment_status' => 'KEY employment_status (employment_status)',
				'employment_type' => 'KEY employment_type (employment_type)',
				'department_id' => 'KEY department_id (department_id)',
				'unit_id' => 'KEY unit_id (unit_id)',
				'team_id' => 'KEY team_id (team_id)',
			] as $index => $definition ) {
				$this->add_index_if_missing( "{$p}sn_hr_profiles", $index, $definition );
			}
		}

		$this->create_hr_structure_tables();
	}

	private function create_hr_structure_tables(): void {
		$charset = $this->wpdb->get_charset_collate();
		$p = $this->wpdb->prefix;
		$tables = [
			"{$p}sn_hr_departments" => [
				'parent' => '',
				'parent_column' => '',
			],
			"{$p}sn_hr_units" => [
				'parent' => 'department_id',
				'parent_column' => 'department_id BIGINT UNSIGNED DEFAULT NULL,',
			],
			"{$p}sn_hr_teams" => [
				'parent' => 'unit_id',
				'parent_column' => 'unit_id BIGINT UNSIGNED DEFAULT NULL,',
			],
		];
		foreach ( $tables as $table => $config ) {
			$parent_column = $config['parent_column'];
			$this->create_table_if_missing( $table, "CREATE TABLE {$table} (
				id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
				{$parent_column}
				slug VARCHAR(100) NOT NULL,
				label VARCHAR(191) NOT NULL,
				description TEXT DEFAULT NULL,
				is_active TINYINT(1) NOT NULL DEFAULT 1,
				sort_order INT NOT NULL DEFAULT 0,
				created_at DATETIME DEFAULT NULL,
				updated_at DATETIME DEFAULT NULL,
				PRIMARY KEY  (id),
				UNIQUE KEY slug (slug),
				KEY is_active (is_active),
				KEY sort_order (sort_order)
			) {$charset};" );
			if ( $this->table_exists( $table ) ) {
				if ( $config['parent'] ) {
					$this->add_column_if_missing( $table, $config['parent'], $config['parent'] . ' bigint unsigned NULL' );
					$this->add_index_if_missing( $table, $config['parent'], 'KEY ' . $config['parent'] . ' (' . $config['parent'] . ')' );
				}
				foreach ( [
					'slug' => 'slug varchar(100) NOT NULL DEFAULT \'\'',
					'label' => 'label varchar(191) NOT NULL DEFAULT \'\'',
					'description' => 'description text NULL',
					'is_active' => 'is_active tinyint(1) NOT NULL DEFAULT 1',
					'sort_order' => 'sort_order int NOT NULL DEFAULT 0',
					'created_at' => 'created_at datetime NULL',
					'updated_at' => 'updated_at datetime NULL',
				] as $column => $definition ) {
					$this->add_column_if_missing( $table, $column, $definition );
				}
				foreach ( [
					'slug' => 'UNIQUE KEY slug (slug)',
					'is_active' => 'KEY is_active (is_active)',
					'sort_order' => 'KEY sort_order (sort_order)',
				] as $index => $definition ) {
					$this->add_index_if_missing( $table, $index, $definition );
				}
			}
		}
		$this->log_step( 'create_hr_structure_tables' );
	}

	private function create_hierarchy_tables(): void {
		$charset = $this->wpdb->get_charset_collate();
		$p = $this->wpdb->prefix;

		$this->create_table_if_missing( "{$p}sn_hr_assignments", "CREATE TABLE {$p}sn_hr_assignments (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			child_profile_id BIGINT UNSIGNED NOT NULL,
			parent_profile_id BIGINT UNSIGNED DEFAULT NULL,
			relationship_type VARCHAR(80) NOT NULL DEFAULT 'reports_to',
			effective_from DATETIME DEFAULT NULL,
			effective_to DATETIME DEFAULT NULL,
			is_current TINYINT(1) NOT NULL DEFAULT 1,
			source VARCHAR(80) DEFAULT NULL,
			created_by BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY  (id),
			KEY child_profile_id (child_profile_id),
			KEY parent_profile_id (parent_profile_id),
			KEY relationship_type (relationship_type),
			KEY is_current (is_current),
			KEY effective_from (effective_from),
			KEY effective_to (effective_to)
		) {$charset};" );

		$this->create_table_if_missing( "{$p}sn_hr_assignment_history", "CREATE TABLE {$p}sn_hr_assignment_history (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			assignment_id BIGINT UNSIGNED DEFAULT NULL,
			child_profile_id BIGINT UNSIGNED DEFAULT NULL,
			old_parent_profile_id BIGINT UNSIGNED DEFAULT NULL,
			new_parent_profile_id BIGINT UNSIGNED DEFAULT NULL,
			actor_user_id BIGINT UNSIGNED DEFAULT NULL,
			action VARCHAR(120) NOT NULL,
			reason TEXT DEFAULT NULL,
			context LONGTEXT DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY  (id),
			KEY assignment_id (assignment_id),
			KEY child_profile_id (child_profile_id),
			KEY old_parent_profile_id (old_parent_profile_id),
			KEY new_parent_profile_id (new_parent_profile_id),
			KEY actor_user_id (actor_user_id),
			KEY action (action),
			KEY created_at (created_at)
		) {$charset};" );

		$this->log_step( 'create_hierarchy_tables' );
	}

	private function create_mis_tables(): void {
		$charset = $this->wpdb->get_charset_collate();
		$p = $this->wpdb->prefix;

		$this->create_table_if_missing( "{$p}sn_mis_import_batches", "CREATE TABLE {$p}sn_mis_import_batches (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			batch_code VARCHAR(80) NOT NULL,
			title VARCHAR(190) NOT NULL,
			source_type VARCHAR(80) DEFAULT NULL,
			source_label VARCHAR(190) DEFAULT NULL,
			campaign_code VARCHAR(120) DEFAULT NULL,
			data_category VARCHAR(120) DEFAULT NULL,
			import_file_name VARCHAR(255) DEFAULT NULL,
			import_status VARCHAR(40) NOT NULL DEFAULT 'draft',
			assigned_manager_user_id BIGINT UNSIGNED DEFAULT NULL,
			total_rows INT NOT NULL DEFAULT 0,
			valid_rows INT NOT NULL DEFAULT 0,
			duplicate_rows INT NOT NULL DEFAULT 0,
			invalid_rows INT NOT NULL DEFAULT 0,
			created_by BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			notes TEXT DEFAULT NULL,
			PRIMARY KEY  (id),
			UNIQUE KEY batch_code (batch_code),
			KEY import_status (import_status),
			KEY assigned_manager_user_id (assigned_manager_user_id),
			KEY campaign_code (campaign_code),
			KEY data_category (data_category),
			KEY created_at (created_at)
		) {$charset};" );

		$rows_table = "{$p}sn_mis_data_rows";
		$this->repair_mis_data_rows_table( $rows_table, $charset );
		if ( ! $this->table_exists( $rows_table ) ) {
			$this->log_step( 'skipped_columns_because_table_missing: ' . $rows_table, 'skipped' );
			$this->log_step( 'skipped_indexes_because_table_missing: ' . $rows_table, 'skipped' );
			$this->record_error( $rows_table, 'create_table_final_check', 'MIS data rows table is still missing after minimal CREATE TABLE' );
		} else {
			$this->add_column_if_missing( $rows_table, 'batch_id', 'batch_id bigint unsigned NOT NULL DEFAULT 0' );
			$this->add_column_if_missing( $rows_table, 'row_number', '`row_number` int unsigned NOT NULL DEFAULT 0' );
			$this->add_column_if_missing( $rows_table, 'customer_name', 'customer_name varchar(191) NULL' );
			$this->add_column_if_missing( $rows_table, 'customer_phone', "customer_phone varchar(50) NOT NULL DEFAULT ''" );
			$this->add_column_if_missing( $rows_table, 'normalized_phone', "normalized_phone varchar(50) NOT NULL DEFAULT ''" );
			$this->add_column_if_missing( $rows_table, 'city', 'city varchar(100) NULL' );
			$this->add_column_if_missing( $rows_table, 'province', 'province varchar(100) NULL' );
			$this->add_column_if_missing( $rows_table, 'source_meta', 'source_meta longtext NULL' );
			$this->add_column_if_missing( $rows_table, 'row_status', "row_status varchar(30) NOT NULL DEFAULT 'valid'" );
			$this->add_column_if_missing( $rows_table, 'duplicate_reason', 'duplicate_reason varchar(191) NULL' );
			$this->add_column_if_missing( $rows_table, 'assigned_manager_user_id', 'assigned_manager_user_id bigint unsigned NULL' );
			$this->add_column_if_missing( $rows_table, 'assigned_at', 'assigned_at datetime NULL' );
			$this->add_column_if_missing( $rows_table, 'created_at', 'created_at datetime NULL' );
			$this->add_column_if_missing( $rows_table, 'updated_at', 'updated_at datetime NULL' );
			if ( $this->column_exists( $rows_table, 'batch_id' ) ) {
				$this->add_index_if_missing( $rows_table, 'batch_id', 'KEY batch_id (batch_id)' );
			}
			if ( $this->column_exists( $rows_table, 'normalized_phone' ) ) {
				$this->add_index_if_missing( $rows_table, 'normalized_phone', 'KEY normalized_phone (normalized_phone)' );
			}
			if ( $this->column_exists( $rows_table, 'assigned_manager_user_id' ) ) {
				$this->add_index_if_missing( $rows_table, 'assigned_manager_user_id', 'KEY assigned_manager_user_id (assigned_manager_user_id)' );
			}
			if ( $this->column_exists( $rows_table, 'row_status' ) ) {
				$this->add_index_if_missing( $rows_table, 'row_status', 'KEY row_status (row_status)' );
				$this->wpdb->query( "UPDATE {$rows_table} SET row_status='valid' WHERE row_status IS NULL OR row_status=''" );
			}
		}

		$pool_table = "{$p}sn_mis_lead_pool";
		$this->create_table_if_missing( $pool_table, "CREATE TABLE {$pool_table} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			source_row_id BIGINT UNSIGNED NOT NULL,
			batch_id BIGINT UNSIGNED NOT NULL,
			assigned_manager_user_id BIGINT UNSIGNED NOT NULL,
			assigned_seller_user_id BIGINT UNSIGNED DEFAULT NULL,
			pool_status VARCHAR(40) NOT NULL DEFAULT 'ready',
			normalized_phone VARCHAR(50) NOT NULL DEFAULT '',
			customer_phone VARCHAR(50) NOT NULL DEFAULT '',
			customer_name VARCHAR(191) DEFAULT NULL,
			city VARCHAR(100) DEFAULT NULL,
			province VARCHAR(100) DEFAULT NULL,
			source_meta LONGTEXT DEFAULT NULL,
			created_by BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME DEFAULT NULL,
			updated_at DATETIME DEFAULT NULL,
			notes TEXT DEFAULT NULL,
			PRIMARY KEY  (id),
			UNIQUE KEY source_row_id (source_row_id),
			KEY batch_id (batch_id),
			KEY assigned_manager_user_id (assigned_manager_user_id),
			KEY assigned_seller_user_id (assigned_seller_user_id),
			KEY pool_status (pool_status),
			KEY normalized_phone (normalized_phone)
		) {$charset};" );
		if ( $this->table_exists( $pool_table ) ) {
			$this->add_column_if_missing( $pool_table, 'source_row_id', 'source_row_id bigint unsigned NOT NULL DEFAULT 0' );
			$this->add_column_if_missing( $pool_table, 'batch_id', 'batch_id bigint unsigned NOT NULL DEFAULT 0' );
			$this->add_column_if_missing( $pool_table, 'assigned_manager_user_id', 'assigned_manager_user_id bigint unsigned NOT NULL DEFAULT 0' );
			$this->add_column_if_missing( $pool_table, 'assigned_seller_user_id', 'assigned_seller_user_id bigint unsigned NULL' );
			$this->add_column_if_missing( $pool_table, 'pool_status', "pool_status varchar(40) NOT NULL DEFAULT 'ready'" );
			$this->add_column_if_missing( $pool_table, 'normalized_phone', "normalized_phone varchar(50) NOT NULL DEFAULT ''" );
			$this->add_column_if_missing( $pool_table, 'customer_phone', "customer_phone varchar(50) NOT NULL DEFAULT ''" );
			$this->add_column_if_missing( $pool_table, 'customer_name', 'customer_name varchar(191) NULL' );
			$this->add_column_if_missing( $pool_table, 'city', 'city varchar(100) NULL' );
			$this->add_column_if_missing( $pool_table, 'province', 'province varchar(100) NULL' );
			$this->add_column_if_missing( $pool_table, 'source_meta', 'source_meta longtext NULL' );
			$this->add_column_if_missing( $pool_table, 'created_by', 'created_by bigint unsigned NULL' );
			$this->add_column_if_missing( $pool_table, 'created_at', 'created_at datetime NULL' );
			$this->add_column_if_missing( $pool_table, 'updated_at', 'updated_at datetime NULL' );
			$this->add_column_if_missing( $pool_table, 'notes', 'notes text NULL' );
			$this->add_index_if_missing( $pool_table, 'source_row_id', 'UNIQUE KEY source_row_id (source_row_id)' );
			$this->add_index_if_missing( $pool_table, 'batch_id', 'KEY batch_id (batch_id)' );
			$this->add_index_if_missing( $pool_table, 'assigned_manager_user_id', 'KEY assigned_manager_user_id (assigned_manager_user_id)' );
			$this->add_index_if_missing( $pool_table, 'assigned_seller_user_id', 'KEY assigned_seller_user_id (assigned_seller_user_id)' );
			$this->add_index_if_missing( $pool_table, 'pool_status', 'KEY pool_status (pool_status)' );
			$this->add_index_if_missing( $pool_table, 'normalized_phone', 'KEY normalized_phone (normalized_phone)' );
		}

		$rules_table = "{$p}sn_mis_distribution_rules";
		$this->create_table_if_missing( $rules_table, "CREATE TABLE {$rules_table} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			rule_name VARCHAR(191) NOT NULL,
			rule_type VARCHAR(50) NOT NULL DEFAULT 'equal_split',
			manager_user_id BIGINT UNSIGNED DEFAULT NULL,
			status VARCHAR(40) NOT NULL DEFAULT 'draft',
			config LONGTEXT DEFAULT NULL,
			created_by BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME DEFAULT NULL,
			updated_at DATETIME DEFAULT NULL,
			notes TEXT DEFAULT NULL,
			PRIMARY KEY  (id),
			KEY rule_type (rule_type),
			KEY manager_user_id (manager_user_id),
			KEY status (status)
		) {$charset};" );
		if ( $this->table_exists( $rules_table ) ) {
			$this->add_column_if_missing( $rules_table, 'rule_name', 'rule_name varchar(191) NOT NULL DEFAULT \'\'' );
			$this->add_column_if_missing( $rules_table, 'rule_type', "rule_type varchar(50) NOT NULL DEFAULT 'equal_split'" );
			$this->add_column_if_missing( $rules_table, 'manager_user_id', 'manager_user_id bigint unsigned NULL' );
			$this->add_column_if_missing( $rules_table, 'status', "status varchar(40) NOT NULL DEFAULT 'draft'" );
			$this->add_column_if_missing( $rules_table, 'config', 'config longtext NULL' );
			$this->add_column_if_missing( $rules_table, 'created_by', 'created_by bigint unsigned NULL' );
			$this->add_column_if_missing( $rules_table, 'created_at', 'created_at datetime NULL' );
			$this->add_column_if_missing( $rules_table, 'updated_at', 'updated_at datetime NULL' );
			$this->add_column_if_missing( $rules_table, 'notes', 'notes text NULL' );
			$this->add_index_if_missing( $rules_table, 'rule_type', 'KEY rule_type (rule_type)' );
			$this->add_index_if_missing( $rules_table, 'manager_user_id', 'KEY manager_user_id (manager_user_id)' );
			$this->add_index_if_missing( $rules_table, 'status', 'KEY status (status)' );
		}

		$plans_table = "{$p}sn_mis_distribution_plans";
		$this->create_table_if_missing( $plans_table, "CREATE TABLE {$plans_table} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			plan_code VARCHAR(100) NOT NULL,
			rule_id BIGINT UNSIGNED DEFAULT NULL,
			batch_id BIGINT UNSIGNED DEFAULT NULL,
			manager_user_id BIGINT UNSIGNED NOT NULL,
			plan_status VARCHAR(40) NOT NULL DEFAULT 'draft',
			total_pool_rows INT UNSIGNED NOT NULL DEFAULT 0,
			planned_rows INT UNSIGNED NOT NULL DEFAULT 0,
			skipped_rows INT UNSIGNED NOT NULL DEFAULT 0,
			created_by BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME DEFAULT NULL,
			updated_at DATETIME DEFAULT NULL,
			notes TEXT DEFAULT NULL,
			PRIMARY KEY  (id),
			UNIQUE KEY plan_code (plan_code),
			KEY rule_id (rule_id),
			KEY batch_id (batch_id),
			KEY manager_user_id (manager_user_id),
			KEY plan_status (plan_status)
		) {$charset};" );
		if ( $this->table_exists( $plans_table ) ) {
			$this->add_column_if_missing( $plans_table, 'plan_code', 'plan_code varchar(100) NOT NULL DEFAULT \'\'' );
			$this->add_column_if_missing( $plans_table, 'rule_id', 'rule_id bigint unsigned NULL' );
			$this->add_column_if_missing( $plans_table, 'batch_id', 'batch_id bigint unsigned NULL' );
			$this->add_column_if_missing( $plans_table, 'manager_user_id', 'manager_user_id bigint unsigned NOT NULL DEFAULT 0' );
			$this->add_column_if_missing( $plans_table, 'plan_status', "plan_status varchar(40) NOT NULL DEFAULT 'draft'" );
			$this->add_column_if_missing( $plans_table, 'total_pool_rows', 'total_pool_rows int unsigned NOT NULL DEFAULT 0' );
			$this->add_column_if_missing( $plans_table, 'planned_rows', 'planned_rows int unsigned NOT NULL DEFAULT 0' );
			$this->add_column_if_missing( $plans_table, 'skipped_rows', 'skipped_rows int unsigned NOT NULL DEFAULT 0' );
			$this->add_column_if_missing( $plans_table, 'created_by', 'created_by bigint unsigned NULL' );
			$this->add_column_if_missing( $plans_table, 'created_at', 'created_at datetime NULL' );
			$this->add_column_if_missing( $plans_table, 'updated_at', 'updated_at datetime NULL' );
			$this->add_column_if_missing( $plans_table, 'notes', 'notes text NULL' );
			$this->add_index_if_missing( $plans_table, 'plan_code', 'UNIQUE KEY plan_code (plan_code)' );
			$this->add_index_if_missing( $plans_table, 'rule_id', 'KEY rule_id (rule_id)' );
			$this->add_index_if_missing( $plans_table, 'batch_id', 'KEY batch_id (batch_id)' );
			$this->add_index_if_missing( $plans_table, 'manager_user_id', 'KEY manager_user_id (manager_user_id)' );
			$this->add_index_if_missing( $plans_table, 'plan_status', 'KEY plan_status (plan_status)' );
		}

		$items_table = "{$p}sn_mis_distribution_plan_items";
		$this->create_table_if_missing( $items_table, "CREATE TABLE {$items_table} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			plan_id BIGINT UNSIGNED NOT NULL,
			pool_id BIGINT UNSIGNED NOT NULL,
			source_row_id BIGINT UNSIGNED NOT NULL,
			manager_user_id BIGINT UNSIGNED NOT NULL,
			planned_seller_user_id BIGINT UNSIGNED DEFAULT NULL,
			item_status VARCHAR(40) NOT NULL DEFAULT 'planned',
			skip_reason VARCHAR(191) DEFAULT NULL,
			created_at DATETIME DEFAULT NULL,
			updated_at DATETIME DEFAULT NULL,
			PRIMARY KEY  (id),
			KEY plan_id (plan_id),
			KEY pool_id (pool_id),
			KEY manager_user_id (manager_user_id),
			KEY planned_seller_user_id (planned_seller_user_id),
			KEY item_status (item_status)
		) {$charset};" );
		if ( $this->table_exists( $items_table ) ) {
			$this->add_column_if_missing( $items_table, 'plan_id', 'plan_id bigint unsigned NOT NULL DEFAULT 0' );
			$this->add_column_if_missing( $items_table, 'pool_id', 'pool_id bigint unsigned NOT NULL DEFAULT 0' );
			$this->add_column_if_missing( $items_table, 'source_row_id', 'source_row_id bigint unsigned NOT NULL DEFAULT 0' );
			$this->add_column_if_missing( $items_table, 'manager_user_id', 'manager_user_id bigint unsigned NOT NULL DEFAULT 0' );
			$this->add_column_if_missing( $items_table, 'planned_seller_user_id', 'planned_seller_user_id bigint unsigned NULL' );
			$this->add_column_if_missing( $items_table, 'item_status', "item_status varchar(40) NOT NULL DEFAULT 'planned'" );
			$this->add_column_if_missing( $items_table, 'skip_reason', 'skip_reason varchar(191) NULL' );
			$this->add_column_if_missing( $items_table, 'created_lead_id', 'created_lead_id bigint unsigned NULL' );
			$this->add_column_if_missing( $items_table, 'lead_created_at', 'lead_created_at datetime NULL' );
			$this->add_column_if_missing( $items_table, 'lead_creation_status', 'lead_creation_status varchar(40) NULL' );
			$this->add_column_if_missing( $items_table, 'lead_creation_error', 'lead_creation_error varchar(191) NULL' );
			$this->add_column_if_missing( $items_table, 'created_at', 'created_at datetime NULL' );
			$this->add_column_if_missing( $items_table, 'updated_at', 'updated_at datetime NULL' );
			$this->add_index_if_missing( $items_table, 'plan_pool', 'UNIQUE KEY plan_pool (plan_id, pool_id)' );
			$this->add_index_if_missing( $items_table, 'plan_id', 'KEY plan_id (plan_id)' );
			$this->add_index_if_missing( $items_table, 'pool_id', 'KEY pool_id (pool_id)' );
			$this->add_index_if_missing( $items_table, 'manager_user_id', 'KEY manager_user_id (manager_user_id)' );
			$this->add_index_if_missing( $items_table, 'planned_seller_user_id', 'KEY planned_seller_user_id (planned_seller_user_id)' );
			$this->add_index_if_missing( $items_table, 'item_status', 'KEY item_status (item_status)' );
			$this->add_index_if_missing( $items_table, 'created_lead_id', 'KEY created_lead_id (created_lead_id)' );
			$this->add_index_if_missing( $items_table, 'lead_creation_status', 'KEY lead_creation_status (lead_creation_status)' );
		}

		$this->create_table_if_missing( "{$p}sn_mis_distribution_logs", "CREATE TABLE {$p}sn_mis_distribution_logs (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			batch_id BIGINT UNSIGNED DEFAULT NULL,
			actor_user_id BIGINT UNSIGNED DEFAULT NULL,
			action VARCHAR(120) NOT NULL,
			old_value LONGTEXT DEFAULT NULL,
			new_value LONGTEXT DEFAULT NULL,
			reason TEXT DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY  (id),
			KEY batch_id (batch_id),
			KEY actor_user_id (actor_user_id),
			KEY action (action),
			KEY created_at (created_at)
		) {$charset};" );

		$this->log_step( 'create_mis_tables', $this->mis_tables_complete() ? 'ok' : 'error' );
	}

	private function create_distribution_engine_tables(): void {
		$charset = $this->wpdb->get_charset_collate();
		$p = $this->wpdb->prefix;

		$chains = "{$p}sn_distribution_chains";
		$this->create_table_if_missing( $chains, "CREATE TABLE {$chains} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			chain_code VARCHAR(100) NOT NULL,
			title VARCHAR(191) NOT NULL,
			description TEXT DEFAULT NULL,
			is_active TINYINT(1) NOT NULL DEFAULT 1,
			is_default TINYINT(1) NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT NULL,
			updated_at DATETIME DEFAULT NULL,
			PRIMARY KEY  (id),
			UNIQUE KEY chain_code (chain_code),
			KEY is_active (is_active),
			KEY is_default (is_default)
		) {$charset};" );
		if ( $this->table_exists( $chains ) ) {
			foreach ( [
				'chain_code' => "chain_code varchar(100) NOT NULL DEFAULT ''",
				'title' => "title varchar(191) NOT NULL DEFAULT ''",
				'description' => 'description text NULL',
				'is_active' => 'is_active tinyint(1) NOT NULL DEFAULT 1',
				'is_default' => 'is_default tinyint(1) NOT NULL DEFAULT 0',
				'created_at' => 'created_at datetime NULL',
				'updated_at' => 'updated_at datetime NULL',
			] as $column => $definition ) {
				$this->add_column_if_missing( $chains, $column, $definition );
			}
			$this->add_index_if_missing( $chains, 'chain_code', 'UNIQUE KEY chain_code (chain_code)' );
			$this->add_index_if_missing( $chains, 'is_active', 'KEY is_active (is_active)' );
			$this->add_index_if_missing( $chains, 'is_default', 'KEY is_default (is_default)' );
		}

		$steps = "{$p}sn_distribution_chain_steps";
		$this->create_table_if_missing( $steps, "CREATE TABLE {$steps} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			chain_id BIGINT UNSIGNED NOT NULL,
			step_order INT UNSIGNED NOT NULL DEFAULT 0,
			from_position_slug VARCHAR(100) NOT NULL,
			to_position_slug VARCHAR(100) NOT NULL,
			allow_skip_if_missing TINYINT(1) NOT NULL DEFAULT 1,
			is_final_delivery_step TINYINT(1) NOT NULL DEFAULT 0,
			creates_live_lead TINYINT(1) NOT NULL DEFAULT 0,
			is_active TINYINT(1) NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT NULL,
			updated_at DATETIME DEFAULT NULL,
			PRIMARY KEY  (id),
			KEY chain_id (chain_id),
			KEY step_order (step_order),
			KEY from_position_slug (from_position_slug),
			KEY to_position_slug (to_position_slug),
			KEY is_active (is_active)
		) {$charset};" );
		if ( $this->table_exists( $steps ) ) {
			foreach ( [
				'chain_id' => 'chain_id bigint unsigned NOT NULL DEFAULT 0',
				'step_order' => 'step_order int unsigned NOT NULL DEFAULT 0',
				'from_position_slug' => "from_position_slug varchar(100) NOT NULL DEFAULT ''",
				'to_position_slug' => "to_position_slug varchar(100) NOT NULL DEFAULT ''",
				'allow_skip_if_missing' => 'allow_skip_if_missing tinyint(1) NOT NULL DEFAULT 1',
				'is_final_delivery_step' => 'is_final_delivery_step tinyint(1) NOT NULL DEFAULT 0',
				'creates_live_lead' => 'creates_live_lead tinyint(1) NOT NULL DEFAULT 0',
				'is_active' => 'is_active tinyint(1) NOT NULL DEFAULT 1',
				'created_at' => 'created_at datetime NULL',
				'updated_at' => 'updated_at datetime NULL',
			] as $column => $definition ) {
				$this->add_column_if_missing( $steps, $column, $definition );
			}
			foreach ( [
				'chain_id' => 'KEY chain_id (chain_id)',
				'step_order' => 'KEY step_order (step_order)',
				'from_position_slug' => 'KEY from_position_slug (from_position_slug)',
				'to_position_slug' => 'KEY to_position_slug (to_position_slug)',
				'is_active' => 'KEY is_active (is_active)',
			] as $index => $definition ) {
				$this->add_index_if_missing( $steps, $index, $definition );
			}
		}

		$items = "{$p}sn_distribution_items";
		$this->create_table_if_missing( $items, "CREATE TABLE {$items} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			source_type VARCHAR(60) NOT NULL DEFAULT 'mis_pool',
			source_id BIGINT UNSIGNED NOT NULL,
			batch_id BIGINT UNSIGNED DEFAULT NULL,
			pool_item_id BIGINT UNSIGNED DEFAULT NULL,
			current_owner_user_id BIGINT UNSIGNED DEFAULT NULL,
			current_owner_position VARCHAR(100) DEFAULT NULL,
			current_step_order INT UNSIGNED NOT NULL DEFAULT 0,
			final_seller_user_id BIGINT UNSIGNED DEFAULT NULL,
			live_lead_id BIGINT UNSIGNED DEFAULT NULL,
			converted_by BIGINT UNSIGNED DEFAULT NULL,
			converted_at DATETIME DEFAULT NULL,
			conversion_status VARCHAR(60) DEFAULT NULL,
			conversion_error TEXT DEFAULT NULL,
			conversion_attempts INT UNSIGNED NOT NULL DEFAULT 0,
			seller_customer_name VARCHAR(255) DEFAULT NULL,
			seller_province VARCHAR(100) DEFAULT NULL,
			seller_city VARCHAR(100) DEFAULT NULL,
			seller_contact_status VARCHAR(60) DEFAULT NULL,
			seller_sale_probability TINYINT UNSIGNED NOT NULL DEFAULT 0,
			seller_next_followup_at DATETIME DEFAULT NULL,
			seller_notes TEXT DEFAULT NULL,
			seller_updated_by BIGINT UNSIGNED DEFAULT NULL,
			seller_updated_at DATETIME DEFAULT NULL,
			status VARCHAR(60) NOT NULL DEFAULT 'ready_for_distribution',
			created_at DATETIME DEFAULT NULL,
			updated_at DATETIME DEFAULT NULL,
			PRIMARY KEY  (id),
			UNIQUE KEY source_unique (source_type, source_id),
			KEY batch_id (batch_id),
			KEY pool_item_id (pool_item_id),
			KEY current_owner_user_id (current_owner_user_id),
			KEY current_owner_position (current_owner_position),
			KEY status (status),
			KEY final_seller_user_id (final_seller_user_id),
			KEY live_lead_id (live_lead_id),
			KEY conversion_status (conversion_status),
			KEY converted_at (converted_at),
			KEY seller_next_followup_at (seller_next_followup_at),
			KEY seller_contact_status (seller_contact_status)
		) {$charset};" );
		if ( $this->table_exists( $items ) ) {
			foreach ( [
				'source_type' => "source_type varchar(60) NOT NULL DEFAULT 'mis_pool'",
				'source_id' => 'source_id bigint unsigned NOT NULL DEFAULT 0',
				'batch_id' => 'batch_id bigint unsigned NULL',
				'pool_item_id' => 'pool_item_id bigint unsigned NULL',
				'current_owner_user_id' => 'current_owner_user_id bigint unsigned NULL',
				'current_owner_position' => 'current_owner_position varchar(100) NULL',
				'current_step_order' => 'current_step_order int unsigned NOT NULL DEFAULT 0',
				'final_seller_user_id' => 'final_seller_user_id bigint unsigned NULL',
				'live_lead_id' => 'live_lead_id bigint unsigned NULL',
				'converted_by' => 'converted_by bigint unsigned NULL',
				'converted_at' => 'converted_at datetime NULL',
				'conversion_status' => 'conversion_status varchar(60) NULL',
				'conversion_error' => 'conversion_error text NULL',
				'conversion_attempts' => 'conversion_attempts int unsigned NOT NULL DEFAULT 0',
				'seller_customer_name' => 'seller_customer_name varchar(255) NULL',
				'seller_province' => 'seller_province varchar(100) NULL',
				'seller_city' => 'seller_city varchar(100) NULL',
				'seller_contact_status' => 'seller_contact_status varchar(60) NULL',
				'seller_sale_probability' => 'seller_sale_probability tinyint unsigned NOT NULL DEFAULT 0',
				'seller_next_followup_at' => 'seller_next_followup_at datetime NULL',
				'seller_notes' => 'seller_notes text NULL',
				'seller_updated_by' => 'seller_updated_by bigint unsigned NULL',
				'seller_updated_at' => 'seller_updated_at datetime NULL',
				'status' => "status varchar(60) NOT NULL DEFAULT 'ready_for_distribution'",
				'created_at' => 'created_at datetime NULL',
				'updated_at' => 'updated_at datetime NULL',
			] as $column => $definition ) {
				$this->add_column_if_missing( $items, $column, $definition );
			}
			foreach ( [
				'source_unique' => 'UNIQUE KEY source_unique (source_type, source_id)',
				'batch_id' => 'KEY batch_id (batch_id)',
				'pool_item_id' => 'KEY pool_item_id (pool_item_id)',
				'current_owner_user_id' => 'KEY current_owner_user_id (current_owner_user_id)',
				'current_owner_position' => 'KEY current_owner_position (current_owner_position)',
				'status' => 'KEY status (status)',
				'final_seller_user_id' => 'KEY final_seller_user_id (final_seller_user_id)',
				'live_lead_id' => 'KEY live_lead_id (live_lead_id)',
				'conversion_status' => 'KEY conversion_status (conversion_status)',
				'converted_at' => 'KEY converted_at (converted_at)',
				'seller_next_followup_at' => 'KEY seller_next_followup_at (seller_next_followup_at)',
				'seller_contact_status' => 'KEY seller_contact_status (seller_contact_status)',
			] as $index => $definition ) {
				$this->add_index_if_missing( $items, $index, $definition );
			}
		}

		$logs = "{$p}sn_distribution_item_logs";
		$this->create_table_if_missing( $logs, "CREATE TABLE {$logs} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			item_id BIGINT UNSIGNED NOT NULL,
			action VARCHAR(100) NOT NULL,
			actor_user_id BIGINT UNSIGNED DEFAULT NULL,
			from_user_id BIGINT UNSIGNED DEFAULT NULL,
			to_user_id BIGINT UNSIGNED DEFAULT NULL,
			from_position VARCHAR(100) DEFAULT NULL,
			to_position VARCHAR(100) DEFAULT NULL,
			step_order INT UNSIGNED DEFAULT NULL,
			note TEXT DEFAULT NULL,
			payload_json LONGTEXT DEFAULT NULL,
			created_at DATETIME DEFAULT NULL,
			PRIMARY KEY  (id),
			KEY item_id (item_id),
			KEY action (action),
			KEY actor_user_id (actor_user_id),
			KEY to_user_id (to_user_id),
			KEY created_at (created_at)
		) {$charset};" );
		if ( $this->table_exists( $logs ) ) {
			foreach ( [
				'item_id' => 'item_id bigint unsigned NOT NULL DEFAULT 0',
				'action' => "action varchar(100) NOT NULL DEFAULT ''",
				'actor_user_id' => 'actor_user_id bigint unsigned NULL',
				'from_user_id' => 'from_user_id bigint unsigned NULL',
				'to_user_id' => 'to_user_id bigint unsigned NULL',
				'from_position' => 'from_position varchar(100) NULL',
				'to_position' => 'to_position varchar(100) NULL',
				'step_order' => 'step_order int unsigned NULL',
				'note' => 'note text NULL',
				'payload_json' => 'payload_json longtext NULL',
				'created_at' => 'created_at datetime NULL',
			] as $column => $definition ) {
				$this->add_column_if_missing( $logs, $column, $definition );
			}
			foreach ( [
				'item_id' => 'KEY item_id (item_id)',
				'action' => 'KEY action (action)',
				'actor_user_id' => 'KEY actor_user_id (actor_user_id)',
				'to_user_id' => 'KEY to_user_id (to_user_id)',
				'created_at' => 'KEY created_at (created_at)',
			] as $index => $definition ) {
				$this->add_index_if_missing( $logs, $index, $definition );
			}
		}

		$this->seed_default_distribution_chain();
		$this->log_step( 'create_distribution_engine_tables', $this->distribution_tables_complete() ? 'ok' : 'error' );
	}


	private function create_after_sales_tables(): void {
		$charset = $this->wpdb->get_charset_collate();
		$p = $this->wpdb->prefix;
		$cases = "{$p}sn_after_sales_cases";
		$logs = "{$p}sn_after_sales_case_logs";

		$this->create_table_if_missing( $cases, "CREATE TABLE {$cases} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			case_code VARCHAR(100) NOT NULL DEFAULT '',
			invoice_id BIGINT UNSIGNED DEFAULT NULL,
			invoice_code VARCHAR(100) DEFAULT NULL,
			lead_id BIGINT UNSIGNED DEFAULT NULL,
			customer_name VARCHAR(191) DEFAULT NULL,
			customer_phone VARCHAR(50) DEFAULT NULL,
			normalized_phone VARCHAR(50) DEFAULT NULL,
			seller_user_id BIGINT UNSIGNED DEFAULT NULL,
			assigned_after_sales_user_id BIGINT UNSIGNED DEFAULT NULL,
			case_type VARCHAR(50) DEFAULT NULL,
			case_status VARCHAR(50) NOT NULL DEFAULT 'open',
			priority VARCHAR(30) NOT NULL DEFAULT 'normal',
			source VARCHAR(50) NOT NULL DEFAULT 'manual',
			opened_at DATETIME DEFAULT NULL,
			due_at DATETIME DEFAULT NULL,
			resolved_at DATETIME DEFAULT NULL,
			created_by BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME DEFAULT NULL,
			updated_at DATETIME DEFAULT NULL,
			notes TEXT DEFAULT NULL,
			PRIMARY KEY  (id),
			KEY case_code (case_code),
			KEY invoice_id (invoice_id),
			KEY invoice_code (invoice_code),
			KEY lead_id (lead_id),
			KEY normalized_phone (normalized_phone),
			KEY seller_user_id (seller_user_id),
			KEY assigned_after_sales_user_id (assigned_after_sales_user_id),
			KEY case_status (case_status),
			KEY priority (priority),
			KEY due_at (due_at)
		) {$charset};" );

		if ( $this->table_exists( $cases ) ) {
			foreach ( [
				'case_code' => "case_code varchar(100) NOT NULL DEFAULT ''",
				'invoice_id' => 'invoice_id bigint unsigned NULL',
				'invoice_code' => 'invoice_code varchar(100) NULL',
				'lead_id' => 'lead_id bigint unsigned NULL',
				'customer_name' => 'customer_name varchar(191) NULL',
				'customer_phone' => 'customer_phone varchar(50) NULL',
				'normalized_phone' => 'normalized_phone varchar(50) NULL',
				'seller_user_id' => 'seller_user_id bigint unsigned NULL',
				'assigned_after_sales_user_id' => 'assigned_after_sales_user_id bigint unsigned NULL',
				'case_type' => 'case_type varchar(50) NULL',
				'case_status' => "case_status varchar(50) NOT NULL DEFAULT 'open'",
				'priority' => "priority varchar(30) NOT NULL DEFAULT 'normal'",
				'source' => "source varchar(50) NOT NULL DEFAULT 'manual'",
				'opened_at' => 'opened_at datetime NULL',
				'due_at' => 'due_at datetime NULL',
				'resolved_at' => 'resolved_at datetime NULL',
				'created_by' => 'created_by bigint unsigned NULL',
				'created_at' => 'created_at datetime NULL',
				'updated_at' => 'updated_at datetime NULL',
				'notes' => 'notes text NULL',
			] as $column => $definition ) {
				$this->add_column_if_missing( $cases, $column, $definition );
			}
			foreach ( [
				'case_code' => 'KEY case_code (case_code)',
				'invoice_id' => 'KEY invoice_id (invoice_id)',
				'invoice_code' => 'KEY invoice_code (invoice_code)',
				'lead_id' => 'KEY lead_id (lead_id)',
				'normalized_phone' => 'KEY normalized_phone (normalized_phone)',
				'seller_user_id' => 'KEY seller_user_id (seller_user_id)',
				'assigned_after_sales_user_id' => 'KEY assigned_after_sales_user_id (assigned_after_sales_user_id)',
				'case_status' => 'KEY case_status (case_status)',
				'priority' => 'KEY priority (priority)',
				'due_at' => 'KEY due_at (due_at)',
			] as $index => $definition ) {
				$this->add_index_if_missing( $cases, $index, $definition );
			}
		}

		$this->create_table_if_missing( $logs, "CREATE TABLE {$logs} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			case_id BIGINT UNSIGNED NOT NULL,
			actor_user_id BIGINT UNSIGNED DEFAULT NULL,
			action VARCHAR(80) NOT NULL,
			old_value LONGTEXT DEFAULT NULL,
			new_value LONGTEXT DEFAULT NULL,
			note TEXT DEFAULT NULL,
			created_at DATETIME DEFAULT NULL,
			PRIMARY KEY  (id),
			KEY case_id (case_id),
			KEY actor_user_id (actor_user_id),
			KEY action (action),
			KEY created_at (created_at)
		) {$charset};" );

		if ( $this->table_exists( $logs ) ) {
			foreach ( [
				'case_id' => 'case_id bigint unsigned NOT NULL DEFAULT 0',
				'actor_user_id' => 'actor_user_id bigint unsigned NULL',
				'action' => "action varchar(80) NOT NULL DEFAULT ''",
				'old_value' => 'old_value longtext NULL',
				'new_value' => 'new_value longtext NULL',
				'note' => 'note text NULL',
				'created_at' => 'created_at datetime NULL',
			] as $column => $definition ) {
				$this->add_column_if_missing( $logs, $column, $definition );
			}
			foreach ( [
				'case_id' => 'KEY case_id (case_id)',
				'actor_user_id' => 'KEY actor_user_id (actor_user_id)',
				'action' => 'KEY action (action)',
				'created_at' => 'KEY created_at (created_at)',
			] as $index => $definition ) {
				$this->add_index_if_missing( $logs, $index, $definition );
			}
		}

		$this->log_step( 'create_after_sales_tables', $this->after_sales_tables_complete() ? 'ok' : 'error' );
	}

	private function create_commission_tables(): void {
		$charset = $this->wpdb->get_charset_collate();
		$p = $this->wpdb->prefix;
		$rules = "{$p}sn_commission_rules";
		$runs = "{$p}sn_commission_dry_runs";
		$items = "{$p}sn_commission_dry_run_items";
		$logs = "{$p}sn_commission_wallet_posting_logs";
		$settlement_batches = "{$p}sn_wallet_settlement_batches";
		$settlement_items = "{$p}sn_wallet_settlement_items";

		$this->create_table_if_missing( $rules, "CREATE TABLE {$rules} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			rule_code VARCHAR(100) NOT NULL DEFAULT '',
			title VARCHAR(191) NOT NULL DEFAULT '',
			description TEXT DEFAULT NULL,
			applies_to_position VARCHAR(80) NOT NULL DEFAULT '',
			applies_to_level VARCHAR(80) NOT NULL DEFAULT '',
			payment_method VARCHAR(80) NOT NULL DEFAULT '',
			invoice_status VARCHAR(120) NOT NULL DEFAULT '',
			employment_status VARCHAR(80) NOT NULL DEFAULT '',
			employment_type VARCHAR(80) NOT NULL DEFAULT '',
			calculation_type VARCHAR(60) NOT NULL DEFAULT 'invoice_percent_fixed',
			rate_percent DECIMAL(10,4) NOT NULL DEFAULT 0,
			fixed_amount DECIMAL(18,2) NOT NULL DEFAULT 0,
			daily_base_amount DECIMAL(18,6) DEFAULT NULL,
			attendance_first_before TIME DEFAULT NULL,
			attendance_last_after TIME DEFAULT NULL,
			min_invoice_amount DECIMAL(18,2) DEFAULT NULL,
			max_invoice_amount DECIMAL(18,2) DEFAULT NULL,
			effective_from DATE DEFAULT NULL,
			effective_to DATE DEFAULT NULL,
			is_active TINYINT(1) NOT NULL DEFAULT 1,
			priority INT NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT NULL,
			updated_at DATETIME DEFAULT NULL,
			PRIMARY KEY  (id),
			UNIQUE KEY rule_code (rule_code),
			KEY applies_to_position (applies_to_position),
			KEY calculation_type (calculation_type),
			KEY is_active (is_active),
			KEY priority (priority)
		) {$charset};" );

		if ( $this->table_exists( $rules ) ) {
			foreach ( [
				'rule_code' => "rule_code varchar(100) NOT NULL DEFAULT ''",
				'title' => "title varchar(191) NOT NULL DEFAULT ''",
				'description' => 'description text NULL',
				'applies_to_position' => "applies_to_position varchar(80) NOT NULL DEFAULT ''",
				'applies_to_level' => "applies_to_level varchar(80) NOT NULL DEFAULT ''",
				'payment_method' => "payment_method varchar(80) NOT NULL DEFAULT ''",
				'invoice_status' => "invoice_status varchar(120) NOT NULL DEFAULT ''",
				'employment_status' => "employment_status varchar(80) NOT NULL DEFAULT ''",
				'employment_type' => "employment_type varchar(80) NOT NULL DEFAULT ''",
				'calculation_type' => "calculation_type varchar(60) NOT NULL DEFAULT 'invoice_percent_fixed'",
				'rate_percent' => 'rate_percent decimal(10,4) NOT NULL DEFAULT 0',
				'fixed_amount' => 'fixed_amount decimal(18,2) NOT NULL DEFAULT 0',
				'daily_base_amount' => 'daily_base_amount decimal(18,6) NULL',
				'attendance_first_before' => 'attendance_first_before time NULL',
				'attendance_last_after' => 'attendance_last_after time NULL',
				'min_invoice_amount' => 'min_invoice_amount decimal(18,2) NULL',
				'max_invoice_amount' => 'max_invoice_amount decimal(18,2) NULL',
				'effective_from' => 'effective_from date NULL',
				'effective_to' => 'effective_to date NULL',
				'is_active' => 'is_active tinyint(1) NOT NULL DEFAULT 1',
				'priority' => 'priority int NOT NULL DEFAULT 0',
				'created_at' => 'created_at datetime NULL',
				'updated_at' => 'updated_at datetime NULL',
			] as $column => $definition ) {
				$this->add_column_if_missing( $rules, $column, $definition );
			}
			foreach ( [
				'rule_code' => 'UNIQUE KEY rule_code (rule_code)',
				'applies_to_position' => 'KEY applies_to_position (applies_to_position)',
				'calculation_type' => 'KEY calculation_type (calculation_type)',
				'is_active' => 'KEY is_active (is_active)',
				'priority' => 'KEY priority (priority)',
			] as $index => $definition ) {
				$this->add_index_if_missing( $rules, $index, $definition );
			}
		}

		$this->create_table_if_missing( $runs, "CREATE TABLE {$runs} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			run_code VARCHAR(100) NOT NULL DEFAULT '',
			title VARCHAR(191) NOT NULL DEFAULT '',
			date_from DATE DEFAULT NULL,
			date_to DATE DEFAULT NULL,
			status VARCHAR(40) NOT NULL DEFAULT 'completed',
			created_by BIGINT UNSIGNED DEFAULT NULL,
			summary_json LONGTEXT DEFAULT NULL,
			approval_status VARCHAR(40) NOT NULL DEFAULT 'generated',
			approved_by BIGINT UNSIGNED DEFAULT NULL,
			approved_at DATETIME DEFAULT NULL,
			rejected_by BIGINT UNSIGNED DEFAULT NULL,
			rejected_at DATETIME DEFAULT NULL,
			approval_note TEXT DEFAULT NULL,
			locked_at DATETIME DEFAULT NULL,
			created_at DATETIME DEFAULT NULL,
			PRIMARY KEY  (id),
			UNIQUE KEY run_code (run_code),
			KEY status (status),
			KEY approval_status (approval_status),
			KEY created_by (created_by),
			KEY created_at (created_at)
		) {$charset};" );

		if ( $this->table_exists( $runs ) ) {
			foreach ( [
				'run_code' => "run_code varchar(100) NOT NULL DEFAULT ''",
				'title' => "title varchar(191) NOT NULL DEFAULT ''",
				'date_from' => 'date_from date NULL',
				'date_to' => 'date_to date NULL',
				'status' => "status varchar(40) NOT NULL DEFAULT 'completed'",
				'created_by' => 'created_by bigint unsigned NULL',
				'summary_json' => 'summary_json longtext NULL',
				'approval_status' => "approval_status varchar(40) NOT NULL DEFAULT 'generated'",
				'approved_by' => 'approved_by bigint unsigned NULL',
				'approved_at' => 'approved_at datetime NULL',
				'rejected_by' => 'rejected_by bigint unsigned NULL',
				'rejected_at' => 'rejected_at datetime NULL',
				'approval_note' => 'approval_note text NULL',
				'locked_at' => 'locked_at datetime NULL',
				'created_at' => 'created_at datetime NULL',
			] as $column => $definition ) {
				$this->add_column_if_missing( $runs, $column, $definition );
			}
			foreach ( [
				'run_code' => 'UNIQUE KEY run_code (run_code)',
				'status' => 'KEY status (status)',
				'approval_status' => 'KEY approval_status (approval_status)',
				'created_by' => 'KEY created_by (created_by)',
				'created_at' => 'KEY created_at (created_at)',
			] as $index => $definition ) {
				$this->add_index_if_missing( $runs, $index, $definition );
			}
		}

		$this->create_table_if_missing( $items, "CREATE TABLE {$items} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			run_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
			invoice_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
			invoice_code VARCHAR(100) NOT NULL DEFAULT '',
			seller_user_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
			seller_position VARCHAR(80) NOT NULL DEFAULT '',
			seller_level VARCHAR(80) NOT NULL DEFAULT '',
			invoice_amount DECIMAL(18,2) NOT NULL DEFAULT 0,
			payment_method VARCHAR(80) NOT NULL DEFAULT '',
			invoice_status VARCHAR(80) NOT NULL DEFAULT '',
			matched_rule_id BIGINT UNSIGNED DEFAULT NULL,
			calculated_commission DECIMAL(18,2) NOT NULL DEFAULT 0,
			legacy_commission DECIMAL(18,2) DEFAULT NULL,
			difference_amount DECIMAL(18,2) DEFAULT NULL,
			item_status VARCHAR(40) NOT NULL DEFAULT 'no_rule',
			notes TEXT DEFAULT NULL,
			wallet_transaction_id BIGINT UNSIGNED DEFAULT NULL,
			posted_at DATETIME DEFAULT NULL,
			posting_status VARCHAR(40) NOT NULL DEFAULT 'pending',
			posting_error VARCHAR(191) DEFAULT NULL,
			compensation_history_id BIGINT UNSIGNED DEFAULT NULL,
			compensation_effective_from DATE DEFAULT NULL,
			compensation_effective_to DATE DEFAULT NULL,
			rule_source VARCHAR(60) DEFAULT NULL,
			warning_code VARCHAR(120) DEFAULT NULL,
			calculation_date DATE DEFAULT NULL,
			created_at DATETIME DEFAULT NULL,
			PRIMARY KEY  (id),
			KEY run_id (run_id),
			KEY invoice_id (invoice_id),
			KEY seller_user_id (seller_user_id),
			KEY matched_rule_id (matched_rule_id),
			KEY item_status (item_status),
			KEY wallet_transaction_id (wallet_transaction_id),
			KEY posting_status (posting_status),
			KEY compensation_history_id (compensation_history_id),
			KEY rule_source (rule_source),
			KEY calculation_date (calculation_date)
		) {$charset};" );

		if ( $this->table_exists( $items ) ) {
			foreach ( [
				'run_id' => 'run_id bigint unsigned NOT NULL DEFAULT 0',
				'invoice_id' => 'invoice_id bigint unsigned NOT NULL DEFAULT 0',
				'invoice_code' => "invoice_code varchar(100) NOT NULL DEFAULT ''",
				'seller_user_id' => 'seller_user_id bigint unsigned NOT NULL DEFAULT 0',
				'seller_position' => "seller_position varchar(80) NOT NULL DEFAULT ''",
				'seller_level' => "seller_level varchar(80) NOT NULL DEFAULT ''",
				'invoice_amount' => 'invoice_amount decimal(18,2) NOT NULL DEFAULT 0',
				'payment_method' => "payment_method varchar(80) NOT NULL DEFAULT ''",
				'invoice_status' => "invoice_status varchar(80) NOT NULL DEFAULT ''",
				'matched_rule_id' => 'matched_rule_id bigint unsigned NULL',
				'calculated_commission' => 'calculated_commission decimal(18,2) NOT NULL DEFAULT 0',
				'legacy_commission' => 'legacy_commission decimal(18,2) NULL',
				'difference_amount' => 'difference_amount decimal(18,2) NULL',
				'item_status' => "item_status varchar(40) NOT NULL DEFAULT 'no_rule'",
				'notes' => 'notes text NULL',
				'wallet_transaction_id' => 'wallet_transaction_id bigint unsigned NULL',
				'posted_at' => 'posted_at datetime NULL',
				'posting_status' => "posting_status varchar(40) NOT NULL DEFAULT 'pending'",
				'posting_error' => 'posting_error varchar(191) NULL',
				'compensation_history_id' => 'compensation_history_id bigint unsigned NULL',
				'compensation_effective_from' => 'compensation_effective_from date NULL',
				'compensation_effective_to' => 'compensation_effective_to date NULL',
				'rule_source' => 'rule_source varchar(60) NULL',
				'warning_code' => 'warning_code varchar(120) NULL',
				'calculation_date' => 'calculation_date date NULL',
				'created_at' => 'created_at datetime NULL',
			] as $column => $definition ) {
				$this->add_column_if_missing( $items, $column, $definition );
			}
			foreach ( [
				'run_id' => 'KEY run_id (run_id)',
				'invoice_id' => 'KEY invoice_id (invoice_id)',
				'seller_user_id' => 'KEY seller_user_id (seller_user_id)',
				'matched_rule_id' => 'KEY matched_rule_id (matched_rule_id)',
				'item_status' => 'KEY item_status (item_status)',
				'wallet_transaction_id' => 'KEY wallet_transaction_id (wallet_transaction_id)',
				'posting_status' => 'KEY posting_status (posting_status)',
				'compensation_history_id' => 'KEY compensation_history_id (compensation_history_id)',
				'rule_source' => 'KEY rule_source (rule_source)',
				'calculation_date' => 'KEY calculation_date (calculation_date)',
			] as $index => $definition ) {
				$this->add_index_if_missing( $items, $index, $definition );
			}
		}

		$this->create_table_if_missing( $logs, "CREATE TABLE {$logs} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			run_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
			item_id BIGINT UNSIGNED DEFAULT NULL,
			wallet_transaction_id BIGINT UNSIGNED DEFAULT NULL,
			actor_user_id BIGINT UNSIGNED DEFAULT NULL,
			action VARCHAR(80) NOT NULL DEFAULT '',
			status VARCHAR(40) NOT NULL DEFAULT '',
			old_value LONGTEXT DEFAULT NULL,
			new_value LONGTEXT DEFAULT NULL,
			message TEXT DEFAULT NULL,
			created_at DATETIME DEFAULT NULL,
			PRIMARY KEY  (id),
			KEY run_id (run_id),
			KEY item_id (item_id),
			KEY wallet_transaction_id (wallet_transaction_id),
			KEY action (action),
			KEY status (status)
		) {$charset};" );
		if ( $this->table_exists( $logs ) ) {
			foreach ( [
				'run_id' => 'run_id bigint unsigned NOT NULL DEFAULT 0',
				'item_id' => 'item_id bigint unsigned NULL',
				'wallet_transaction_id' => 'wallet_transaction_id bigint unsigned NULL',
				'actor_user_id' => 'actor_user_id bigint unsigned NULL',
				'action' => "action varchar(80) NOT NULL DEFAULT ''",
				'status' => "status varchar(40) NOT NULL DEFAULT ''",
				'old_value' => 'old_value longtext NULL',
				'new_value' => 'new_value longtext NULL',
				'message' => 'message text NULL',
				'created_at' => 'created_at datetime NULL',
			] as $column => $definition ) {
				$this->add_column_if_missing( $logs, $column, $definition );
			}
			foreach ( [
				'run_id' => 'KEY run_id (run_id)',
				'item_id' => 'KEY item_id (item_id)',
				'wallet_transaction_id' => 'KEY wallet_transaction_id (wallet_transaction_id)',
				'action' => 'KEY action (action)',
				'status' => 'KEY status (status)',
			] as $index => $definition ) {
				$this->add_index_if_missing( $logs, $index, $definition );
			}
		}

		$this->create_table_if_missing( $settlement_batches, "CREATE TABLE {$settlement_batches} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			batch_code VARCHAR(100) NOT NULL DEFAULT '',
			status VARCHAR(40) NOT NULL DEFAULT 'draft',
			created_by BIGINT UNSIGNED DEFAULT NULL,
			total_items INT UNSIGNED NOT NULL DEFAULT 0,
			total_amount DECIMAL(18,2) NOT NULL DEFAULT 0,
			notes TEXT DEFAULT NULL,
			created_at DATETIME DEFAULT NULL,
			updated_at DATETIME DEFAULT NULL,
			PRIMARY KEY  (id),
			UNIQUE KEY batch_code (batch_code),
			KEY status (status),
			KEY created_by (created_by)
		) {$charset};" );
		if ( $this->table_exists( $settlement_batches ) ) {
			foreach ( [
				'batch_code' => "batch_code varchar(100) NOT NULL DEFAULT ''",
				'status' => "status varchar(40) NOT NULL DEFAULT 'draft'",
				'created_by' => 'created_by bigint unsigned NULL',
				'total_items' => 'total_items int unsigned NOT NULL DEFAULT 0',
				'total_amount' => 'total_amount decimal(18,2) NOT NULL DEFAULT 0',
				'notes' => 'notes text NULL',
				'created_at' => 'created_at datetime NULL',
				'updated_at' => 'updated_at datetime NULL',
			] as $column => $definition ) {
				$this->add_column_if_missing( $settlement_batches, $column, $definition );
			}
			foreach ( [
				'batch_code' => 'UNIQUE KEY batch_code (batch_code)',
				'status' => 'KEY status (status)',
				'created_by' => 'KEY created_by (created_by)',
			] as $index => $definition ) {
				$this->add_index_if_missing( $settlement_batches, $index, $definition );
			}
		}

		$this->create_table_if_missing( $settlement_items, "CREATE TABLE {$settlement_items} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			batch_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
			user_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
			wallet_id BIGINT UNSIGNED DEFAULT NULL,
			wallet_type VARCHAR(30) NOT NULL DEFAULT 'seller',
			suggested_amount DECIMAL(18,2) NOT NULL DEFAULT 0,
			item_status VARCHAR(40) NOT NULL DEFAULT 'ready',
			created_at DATETIME DEFAULT NULL,
			updated_at DATETIME DEFAULT NULL,
			PRIMARY KEY  (id),
			KEY batch_id (batch_id),
			KEY user_id (user_id),
			KEY wallet_id (wallet_id),
			KEY item_status (item_status)
		) {$charset};" );
		if ( $this->table_exists( $settlement_items ) ) {
			foreach ( [
				'batch_id' => 'batch_id bigint unsigned NOT NULL DEFAULT 0',
				'user_id' => 'user_id bigint unsigned NOT NULL DEFAULT 0',
				'wallet_id' => 'wallet_id bigint unsigned NULL',
				'wallet_type' => "wallet_type varchar(30) NOT NULL DEFAULT 'seller'",
				'suggested_amount' => 'suggested_amount decimal(18,2) NOT NULL DEFAULT 0',
				'item_status' => "item_status varchar(40) NOT NULL DEFAULT 'ready'",
				'created_at' => 'created_at datetime NULL',
				'updated_at' => 'updated_at datetime NULL',
			] as $column => $definition ) {
				$this->add_column_if_missing( $settlement_items, $column, $definition );
			}
			foreach ( [
				'batch_id' => 'KEY batch_id (batch_id)',
				'user_id' => 'KEY user_id (user_id)',
				'wallet_id' => 'KEY wallet_id (wallet_id)',
				'item_status' => 'KEY item_status (item_status)',
			] as $index => $definition ) {
				$this->add_index_if_missing( $settlement_items, $index, $definition );
			}
		}

		$this->log_step( 'create_commission_tables', $this->commission_tables_complete() ? 'ok' : 'error' );
	}

	private function create_hr_compensation_tables(): void {
		$charset = $this->wpdb->get_charset_collate();
		$p = $this->wpdb->prefix;
		$table = "{$p}sn_hr_compensation_profiles";
		$history = "{$p}sn_hr_compensation_history";

		$this->create_table_if_missing( $table, "CREATE TABLE {$table} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			profile_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
			user_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
			base_salary BIGINT UNSIGNED DEFAULT NULL,
			salary_currency VARCHAR(20) NOT NULL DEFAULT 'IRT',
			commission_enabled TINYINT(1) NOT NULL DEFAULT 0,
			default_commission_rule_id BIGINT UNSIGNED DEFAULT NULL,
			commission_override_mode VARCHAR(30) NOT NULL DEFAULT 'inherit',
			effective_from DATE DEFAULT NULL,
			effective_to DATE DEFAULT NULL,
			notes TEXT DEFAULT NULL,
			is_active TINYINT(1) NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT NULL,
			updated_at DATETIME DEFAULT NULL,
			PRIMARY KEY  (id),
			UNIQUE KEY profile_id (profile_id),
			KEY user_id (user_id),
			KEY commission_enabled (commission_enabled),
			KEY default_commission_rule_id (default_commission_rule_id),
			KEY is_active (is_active)
		) {$charset};" );

		if ( $this->table_exists( $table ) ) {
			foreach ( [
				'profile_id' => 'profile_id bigint unsigned NOT NULL DEFAULT 0',
				'user_id' => 'user_id bigint unsigned NOT NULL DEFAULT 0',
				'base_salary' => 'base_salary bigint unsigned NULL',
				'salary_currency' => "salary_currency varchar(20) NOT NULL DEFAULT 'IRT'",
				'commission_enabled' => 'commission_enabled tinyint(1) NOT NULL DEFAULT 0',
				'default_commission_rule_id' => 'default_commission_rule_id bigint unsigned NULL',
				'commission_override_mode' => "commission_override_mode varchar(30) NOT NULL DEFAULT 'inherit'",
				'effective_from' => 'effective_from date NULL',
				'effective_to' => 'effective_to date NULL',
				'notes' => 'notes text NULL',
				'is_active' => 'is_active tinyint(1) NOT NULL DEFAULT 1',
				'created_at' => 'created_at datetime NULL',
				'updated_at' => 'updated_at datetime NULL',
			] as $column => $definition ) {
				$this->add_column_if_missing( $table, $column, $definition );
			}
			foreach ( [
				'profile_id' => 'UNIQUE KEY profile_id (profile_id)',
				'user_id' => 'KEY user_id (user_id)',
				'commission_enabled' => 'KEY commission_enabled (commission_enabled)',
				'default_commission_rule_id' => 'KEY default_commission_rule_id (default_commission_rule_id)',
				'commission_override_mode' => 'KEY commission_override_mode (commission_override_mode)',
				'is_active' => 'KEY is_active (is_active)',
			] as $index => $definition ) {
				$this->add_index_if_missing( $table, $index, $definition );
			}
		}

		$this->create_table_if_missing( $history, "CREATE TABLE {$history} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			profile_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
			user_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
			base_salary BIGINT UNSIGNED DEFAULT NULL,
			salary_currency VARCHAR(20) NOT NULL DEFAULT 'IRT',
			commission_enabled TINYINT(1) NOT NULL DEFAULT 0,
			commission_rule_id BIGINT UNSIGNED DEFAULT NULL,
			commission_override_mode VARCHAR(30) NOT NULL DEFAULT 'inherit',
			effective_from DATE NOT NULL,
			effective_to DATE DEFAULT NULL,
			employment_type VARCHAR(50) DEFAULT NULL,
			employment_status VARCHAR(50) DEFAULT NULL,
			position_slug_snapshot VARCHAR(80) DEFAULT NULL,
			level_slug_snapshot VARCHAR(80) DEFAULT NULL,
			notes TEXT DEFAULT NULL,
			is_active TINYINT(1) NOT NULL DEFAULT 1,
			locked_after_payroll TINYINT(1) NOT NULL DEFAULT 0,
			created_by BIGINT UNSIGNED DEFAULT NULL,
			updated_by BIGINT UNSIGNED DEFAULT NULL,
			change_reason TEXT DEFAULT NULL,
			created_at DATETIME DEFAULT NULL,
			updated_at DATETIME DEFAULT NULL,
			PRIMARY KEY  (id),
			KEY profile_id (profile_id),
			KEY user_id (user_id),
			KEY commission_rule_id (commission_rule_id),
			KEY commission_override_mode (commission_override_mode),
			KEY effective_from (effective_from),
			KEY effective_to (effective_to),
			KEY is_active (is_active)
		) {$charset};" );

		if ( $this->table_exists( $history ) ) {
			foreach ( [
				'profile_id' => 'profile_id bigint unsigned NOT NULL DEFAULT 0',
				'user_id' => 'user_id bigint unsigned NOT NULL DEFAULT 0',
				'base_salary' => 'base_salary bigint unsigned NULL',
				'salary_currency' => "salary_currency varchar(20) NOT NULL DEFAULT 'IRT'",
				'commission_enabled' => 'commission_enabled tinyint(1) NOT NULL DEFAULT 0',
				'commission_rule_id' => 'commission_rule_id bigint unsigned NULL',
				'commission_override_mode' => "commission_override_mode varchar(30) NOT NULL DEFAULT 'inherit'",
				'effective_from' => 'effective_from date NOT NULL',
				'effective_to' => 'effective_to date NULL',
				'employment_type' => 'employment_type varchar(50) NULL',
				'employment_status' => 'employment_status varchar(50) NULL',
				'position_slug_snapshot' => 'position_slug_snapshot varchar(80) NULL',
				'level_slug_snapshot' => 'level_slug_snapshot varchar(80) NULL',
				'notes' => 'notes text NULL',
				'is_active' => 'is_active tinyint(1) NOT NULL DEFAULT 1',
				'locked_after_payroll' => 'locked_after_payroll tinyint(1) NOT NULL DEFAULT 0',
				'created_by' => 'created_by bigint unsigned NULL',
				'updated_by' => 'updated_by bigint unsigned NULL',
				'change_reason' => 'change_reason text NULL',
				'created_at' => 'created_at datetime NULL',
				'updated_at' => 'updated_at datetime NULL',
			] as $column => $definition ) {
				$this->add_column_if_missing( $history, $column, $definition );
			}
			foreach ( [
				'profile_id' => 'KEY profile_id (profile_id)',
				'user_id' => 'KEY user_id (user_id)',
				'commission_rule_id' => 'KEY commission_rule_id (commission_rule_id)',
				'commission_override_mode' => 'KEY commission_override_mode (commission_override_mode)',
				'effective_from' => 'KEY effective_from (effective_from)',
				'effective_to' => 'KEY effective_to (effective_to)',
				'is_active' => 'KEY is_active (is_active)',
			] as $index => $definition ) {
				$this->add_index_if_missing( $history, $index, $definition );
			}
		}

		$this->log_step( 'create_hr_compensation_tables', $this->hr_compensation_tables_complete() ? 'ok' : 'error' );
	}

	private function repair_mis_data_rows_table( string $table, string $charset ): bool {
		$table = $this->normalize_table_name( $table );
		if ( $this->table_exists( $table ) ) {
			$this->log_step( 'minimal_table_exists: ' . $table, 'skipped', [
				'operation' => 'minimal_create_table',
				'table' => $table,
				'mysql_error' => '',
				'sanitized_query' => '',
				'table_exists_after_operation' => 'yes',
			] );
			return true;
		}

		$sql = "CREATE TABLE IF NOT EXISTS {$table} (
			id bigint unsigned NOT NULL AUTO_INCREMENT,
			batch_id bigint unsigned NOT NULL DEFAULT 0,
			row_number int unsigned NOT NULL DEFAULT 0,
			customer_name varchar(191) NULL,
			customer_phone varchar(50) NOT NULL DEFAULT '',
			normalized_phone varchar(50) NOT NULL DEFAULT '',
			city varchar(100) NULL,
			province varchar(100) NULL,
			source_meta longtext NULL,
			row_status varchar(30) NOT NULL DEFAULT 'valid',
			duplicate_reason varchar(191) NULL,
			assigned_manager_user_id bigint unsigned NULL,
			assigned_at datetime NULL,
			created_at datetime NULL,
			updated_at datetime NULL,
			PRIMARY KEY  (id),
			KEY batch_id (batch_id),
			KEY normalized_phone (normalized_phone),
			KEY row_status (row_status),
			KEY assigned_manager_user_id (assigned_manager_user_id)
		) {$charset}";
		$this->wpdb->query( $sql );
		$exists = $this->table_exists( $table );
		$error = $this->wpdb->last_error ? sanitize_text_field( $this->wpdb->last_error ) : '';
		$this->log_step( 'minimal_create_table: ' . $table, $exists ? 'ok' : 'error', [
			'operation' => 'minimal_create_table',
			'table' => $table,
			'mysql_error' => $exists ? '' : ( $error ?: 'minimal CREATE TABLE did not create table' ),
			'sanitized_query' => $exists ? '' : $this->sanitize_sql_for_report( $this->wpdb->last_query ?: $sql ),
			'table_exists_after_operation' => $exists ? 'yes' : 'no',
		] );
		if ( ! $exists ) {
			$this->record_error( $table, 'minimal_create_table', $error ?: 'minimal CREATE TABLE did not create table', $this->wpdb->last_query ?: $sql );
		}
		return $exists;
	}

	private function seed_default_data(): void {
		$this->repair_legacy_key_columns();
		$this->seed_default_positions();
		$this->seed_default_levels();
		$this->seed_default_role_mappings();
		$this->seed_default_distribution_chain();
	}

	private function seed_default_distribution_chain(): void {
		$chains = $this->wpdb->prefix . 'sn_distribution_chains';
		$steps = $this->wpdb->prefix . 'sn_distribution_chain_steps';
		if ( ! $this->table_exists( $chains ) || ! $this->table_exists( $steps ) ) {
			return;
		}
		$now = current_time( 'mysql' );
		$chain_id = (int) $this->wpdb->get_var( $this->wpdb->prepare( "SELECT id FROM {$chains} WHERE chain_code=%s", 'default_sales_hierarchy' ) );
		if ( ! $chain_id ) {
			$this->wpdb->insert( $chains, [
				'chain_code' => 'default_sales_hierarchy',
				'title' => 'زنجیره پیش‌فرض پخش فروش',
				'description' => 'مدیر فروش به سرپرست ارشد، سرپرست ارشد به سرپرست، سرپرست به فروشنده. عبور از لایه خالی طبق تنظیم هر مرحله مجاز است.',
				'is_active' => 1,
				'is_default' => 1,
				'created_at' => $now,
				'updated_at' => $now,
			] );
			$chain_id = (int) $this->wpdb->insert_id;
			$this->log_step( 'seed_default_distribution_chain inserted chain' );
		} else {
			// Preserve custom chain settings on plugin updates. Updates must not re-enable, re-default, or rename an existing chain.
			$this->log_step( 'seed_default_distribution_chain existing chain preserved' );
		}
		if ( ! $chain_id ) {
			$this->log_step( 'seed_default_distribution_chain missing chain id', 'error' );
			return;
		}
		$default_steps = [
			[ 10, 'sales_manager', 'senior_supervisor', 1, 0, 0 ],
			[ 20, 'senior_supervisor', 'supervisor', 1, 0, 0 ],
			[ 30, 'supervisor', 'seller', 1, 1, 0 ],
		];
		$inserted_steps = 0;
		$preserved_steps = 0;
		foreach ( $default_steps as $row ) {
			[ $order, $from, $to, $skip, $final, $creates ] = $row;
			$existing = (int) $this->wpdb->get_var( $this->wpdb->prepare(
				"SELECT id FROM {$steps} WHERE chain_id=%d AND step_order=%d AND from_position_slug=%s AND to_position_slug=%s",
				$chain_id,
				$order,
				$from,
				$to
			) );
			if ( $existing ) {
				$preserved_steps++;
				continue;
			}
			$data = [
				'chain_id' => $chain_id,
				'step_order' => $order,
				'from_position_slug' => $from,
				'to_position_slug' => $to,
				'allow_skip_if_missing' => $skip,
				'is_final_delivery_step' => $final,
				'creates_live_lead' => $creates,
				'is_active' => 1,
				'created_at' => $now,
				'updated_at' => $now,
			];
			$this->wpdb->insert( $steps, $data );
			$inserted_steps++;
		}
		$this->log_step( "seed_default_distribution_chain steps inserted={$inserted_steps} preserved={$preserved_steps}" );
	}

	private function seed_default_positions(): void {
		$inserted = 0;
		$updated = 0;
		$errors = 0;
		foreach ( $this->default_positions() as $slug => $row ) {
			$result = $this->upsert_by_slug( 'sn_hr_positions', [
				'slug' => $slug,
				'label' => $row['label'],
				'panel_key' => $row['panel_key'],
				'sort_order' => $row['sort_order'],
				'is_active' => 1,
				'is_system' => 1,
			], [ 'label', 'description', 'panel_key', 'sort_order', 'is_active' ] );
			if ( $result === 'inserted' ) { $inserted++; }
			if ( $result === 'updated' ) { $updated++; }
			if ( $result === 'error' ) { $errors++; }
		}
		$this->log_step( "seed_default_positions inserted={$inserted} updated={$updated} errors={$errors}" );
	}

	private function seed_default_levels(): void {
		$inserted = 0;
		$updated = 0;
		$errors = 0;
		foreach ( $this->default_levels() as $slug => $row ) {
			$result = $this->upsert_by_slug( 'sn_hr_levels', [
				'slug' => $slug,
				'label' => $row['label'],
				'sort_order' => $row['sort_order'],
				'is_active' => 1,
				'is_system' => 1,
			], [ 'label', 'description', 'sort_order', 'is_active' ] );
			if ( $result === 'inserted' ) { $inserted++; }
			if ( $result === 'updated' ) { $updated++; }
			if ( $result === 'error' ) { $errors++; }
		}
		$this->log_step( "seed_default_levels inserted={$inserted} updated={$updated} errors={$errors}" );
	}

	private function seed_default_role_mappings(): void {
		$table = $this->wpdb->prefix . 'sn_hr_position_role_mappings';
		$inserted = 0;
		$updated = 0;
		$skipped = 0;
		$errors = 0;
		foreach ( $this->default_role_mappings() as $legacy_role => $position_slug ) {
			$position_id = $this->get_id_by_slug( 'sn_hr_positions', $position_slug );
			if ( ! $position_id ) {
				$skipped++;
				$this->log_step( 'seed_default_role_mappings missing position: ' . $legacy_role . ' -> ' . $position_slug, 'error' );
				continue;
			}
			$existing = $this->wpdb->get_var( $this->wpdb->prepare( "SELECT id FROM {$table} WHERE legacy_role=%s", $legacy_role ) );
			$data = [
				'legacy_role' => $legacy_role,
				'position_id' => $position_id,
				'is_active' => 1,
			];
			if ( $existing ) {
				// Preserve role mapping changes made in HR settings. Seed data may insert missing mappings only.
				$skipped++;
				continue;
			} else {
				$this->wpdb->insert( $table, $data );
				if ( $this->wpdb->last_error ) {
					$errors++;
					$this->log_step( 'seed_default_role_mappings insert failed: ' . $legacy_role . ' - ' . $this->wpdb->last_error, 'error' );
				} else {
					$inserted++;
				}
			}
		}
		$this->log_step( "seed_default_role_mappings inserted={$inserted} updated={$updated} skipped={$skipped} errors={$errors}" );
	}

	private function upsert_by_slug( string $table, array $data, array $preserve_existing_fields = [] ): string {
		$table = $this->normalize_table_name( $table );
		$slug = (string) ( $data['slug'] ?? '' );
		if ( $slug === '' ) { return 'error'; }
		$legacy_key_column = $this->legacy_key_column_for_table( $table );
		if ( $legacy_key_column && ! isset( $data[ $legacy_key_column ] ) ) {
			$data[ $legacy_key_column ] = $slug;
		}
		$existing = $this->wpdb->get_var( $this->wpdb->prepare( "SELECT id FROM {$table} WHERE slug=%s", $slug ) );
		if ( ! $existing && $legacy_key_column ) {
			$existing = $this->wpdb->get_var( $this->wpdb->prepare( "SELECT id FROM {$table} WHERE {$legacy_key_column}=%s", $slug ) );
		}
		if ( $existing ) {
			$update = $data;
			unset( $update['slug'] );
			foreach ( $preserve_existing_fields as $field ) {
				unset( $update[ $field ] );
			}
			if ( empty( $update ) ) {
				return 'skipped';
			}
			$this->wpdb->update( $table, $update, [ 'id' => (int) $existing ] );
			if ( $this->wpdb->last_error ) {
				$this->log_step( 'upsert update failed: ' . $table . '.' . $slug . ' - ' . $this->wpdb->last_error, 'error' );
				return 'error';
			}
			return 'updated';
		} else {
			$this->wpdb->insert( $table, $data );
			if ( $this->wpdb->last_error ) {
				$this->log_step( 'upsert insert failed: ' . $table . '.' . $slug . ' - ' . $this->wpdb->last_error, 'error' );
				return 'error';
			}
			return 'inserted';
		}
	}

	private function get_id_by_slug( string $table, string $slug ): int {
		$table = $this->normalize_table_name( $table );
		$id = (int) $this->wpdb->get_var( $this->wpdb->prepare( "SELECT id FROM {$table} WHERE slug=%s", $slug ) );
		if ( $id ) {
			return $id;
		}
		$legacy_key_column = $this->legacy_key_column_for_table( $table );
		if ( $legacy_key_column ) {
			return (int) $this->wpdb->get_var( $this->wpdb->prepare( "SELECT id FROM {$table} WHERE {$legacy_key_column}=%s", $slug ) );
		}
		return 0;
	}

	private function normalize_table_name( string $table ): string {
		if ( strpos( $table, $this->wpdb->prefix ) === 0 ) {
			return $table;
		}
		return $this->wpdb->prefix . ltrim( $table, '_' );
	}

	private function count_rows( string $table ): int {
		$table = $this->normalize_table_name( $table );
		if ( ! $this->table_exists( $table ) ) {
			return 0;
		}
		return (int) $this->wpdb->get_var( "SELECT COUNT(*) FROM {$table}" );
	}

	private function repair_legacy_key_columns(): void {
		$positions_table = $this->wpdb->prefix . 'sn_hr_positions';
		$levels_table = $this->wpdb->prefix . 'sn_hr_levels';
		$has_position_key = $this->column_exists( $positions_table, 'position_key' );
		$has_level_key = $this->column_exists( $levels_table, 'level_key' );
		$this->log_step( 'detected_position_key_column: ' . ( $has_position_key ? 'yes' : 'no' ) );
		$this->log_step( 'detected_level_key_column: ' . ( $has_level_key ? 'yes' : 'no' ) );
		if ( $has_position_key ) {
			$this->repair_slug_key_pair( $positions_table, 'slug', 'position_key', 'position' );
		}
		if ( $has_level_key ) {
			$this->repair_slug_key_pair( $levels_table, 'slug', 'level_key', 'level' );
		}
	}

	private function repair_slug_key_pair( string $table, string $slug_column, string $key_column, string $label ): void {
		$repaired_empty_key = 0;
		$repaired_empty_slug = 0;
		if ( ! $this->table_exists( $table ) || ! $this->column_exists( $table, $slug_column ) || ! $this->column_exists( $table, $key_column ) ) {
			return;
		}

		$rows = $this->wpdb->get_results( "SELECT id, {$slug_column} AS slug_value, {$key_column} AS key_value FROM {$table} WHERE {$slug_column}<>'' AND ({$key_column}='' OR {$key_column} IS NULL)" );
		foreach ( $rows as $row ) {
			$slug = (string) $row->slug_value;
			if ( $slug === '' ) { continue; }
			$duplicate = (int) $this->wpdb->get_var( $this->wpdb->prepare( "SELECT id FROM {$table} WHERE {$key_column}=%s AND id<>%d LIMIT 1", $slug, (int) $row->id ) );
			if ( $duplicate ) { continue; }
			$this->wpdb->update( $table, [ $key_column => $slug ], [ 'id' => (int) $row->id ] );
			if ( $this->wpdb->last_error ) {
				$this->log_step( "repair_{$label}_empty_{$key_column}_failed: id={$row->id} - " . $this->wpdb->last_error, 'error' );
			} else {
				$repaired_empty_key++;
			}
		}

		$rows = $this->wpdb->get_results( "SELECT id, {$slug_column} AS slug_value, {$key_column} AS key_value FROM {$table} WHERE {$key_column}<>'' AND ({$slug_column}='' OR {$slug_column} IS NULL)" );
		foreach ( $rows as $row ) {
			$key = (string) $row->key_value;
			if ( $key === '' ) { continue; }
			$duplicate = (int) $this->wpdb->get_var( $this->wpdb->prepare( "SELECT id FROM {$table} WHERE {$slug_column}=%s AND id<>%d LIMIT 1", $key, (int) $row->id ) );
			if ( $duplicate ) { continue; }
			$this->wpdb->update( $table, [ $slug_column => $key ], [ 'id' => (int) $row->id ] );
			if ( $this->wpdb->last_error ) {
				$this->log_step( "repair_{$label}_empty_{$slug_column}_failed: id={$row->id} - " . $this->wpdb->last_error, 'error' );
			} else {
				$repaired_empty_slug++;
			}
		}

		$this->log_step( "repaired_empty_{$key_column}_count: {$repaired_empty_key}" );
		$this->log_step( "repaired_empty_{$slug_column}_count: {$repaired_empty_slug}" );
	}

	private function legacy_key_column_for_table( string $table ): string {
		$table = $this->normalize_table_name( $table );
		if ( substr( $table, -strlen( 'sn_hr_positions' ) ) === 'sn_hr_positions' && $this->column_exists( $table, 'position_key' ) ) {
			return 'position_key';
		}
		if ( substr( $table, -strlen( 'sn_hr_levels' ) ) === 'sn_hr_levels' && $this->column_exists( $table, 'level_key' ) ) {
			return 'level_key';
		}
		return '';
	}

	private function default_positions(): array {
		return [
			'sales_deputy' => [ 'label' => 'معاون فروش', 'panel_key' => 'sales_deputy', 'sort_order' => 10 ],
			'sales_manager' => [ 'label' => 'مدیر فروش', 'panel_key' => 'sales_manager', 'sort_order' => 20 ],
			'senior_supervisor' => [ 'label' => 'سرپرست ارشد', 'panel_key' => 'senior_supervisor', 'sort_order' => 30 ],
			'supervisor' => [ 'label' => 'سرپرست', 'panel_key' => 'supervisor', 'sort_order' => 40 ],
			'converter' => [ 'label' => 'تبدیل‌کننده', 'panel_key' => 'converter', 'sort_order' => 45 ],
			'seller' => [ 'label' => 'فروشنده', 'panel_key' => 'seller', 'sort_order' => 50 ],
			'finance' => [ 'label' => 'مالی', 'panel_key' => 'finance', 'sort_order' => 60 ],
			'hr' => [ 'label' => 'منابع انسانی', 'panel_key' => 'hr', 'sort_order' => 70 ],
			'mis' => [ 'label' => 'MIS', 'panel_key' => 'mis', 'sort_order' => 80 ],
			'after_sales' => [ 'label' => 'خدمات پس از فروش', 'panel_key' => 'after_sales', 'sort_order' => 90 ],
			'shipping_expert' => [ 'label' => 'کارشناس ارسال', 'panel_key' => 'shipping', 'sort_order' => 91 ],
			'operations_sales_manager' => [ 'label' => 'مدیر فروش عملیات', 'panel_key' => 'operations_sales_manager', 'sort_order' => 94 ],
			'operations_sales_supervisor' => [ 'label' => 'سرپرست فروش عملیات', 'panel_key' => 'operations_sales_supervisor', 'sort_order' => 95 ],
			'operations_sales_expert' => [ 'label' => 'کارشناس فروش عملیات', 'panel_key' => 'operations_sales_expert', 'sort_order' => 96 ],
			'operations_executive_manager' => [ 'label' => 'مدیر اجرایی عملیات', 'panel_key' => 'operations_executive_manager', 'sort_order' => 97 ],
			'operations_execution_expert' => [ 'label' => 'کارشناس اجرایی عملیات', 'panel_key' => 'operations_execution_expert', 'sort_order' => 98 ],
		];
	}

	private function default_levels(): array {
		return [
			'trainee' => [ 'label' => 'آموزشی', 'sort_order' => 10 ],
			'standard' => [ 'label' => 'قراردادی', 'sort_order' => 20 ],
			'senior' => [ 'label' => 'ارشد', 'sort_order' => 30 ],
			'manager' => [ 'label' => 'مدیریتی', 'sort_order' => 40 ],
		];
	}

	private function default_role_mappings(): array {
		return [
			'sn_seller' => 'seller',
			'sas_employee' => 'seller',
			'sn_supervisor' => 'supervisor',
			'sn_converter' => 'converter',
			'sas_supervisor' => 'supervisor',
			'sn_senior_supervisor' => 'senior_supervisor',
			'sas_senior_supervisor' => 'senior_supervisor',
			'sn_sales_manager' => 'sales_manager',
			'sas_sales_manager' => 'sales_manager',
			'sn_sales_deputy' => 'sales_deputy',
			'sn_financial' => 'finance',
			'sn_financial_approval' => 'finance',
			'sn_finance' => 'finance',
			'sn_hr' => 'hr',
			'sn_mis' => 'mis',
			'sn_after_sales' => 'after_sales',
			'sn_shipping_expert' => 'shipping_expert',
			'sn_operations_sales_manager' => 'operations_sales_manager',
			'sn_operations_sales_supervisor' => 'operations_sales_supervisor',
			'sn_operations_sales_expert' => 'operations_sales_expert',
			'sn_operations_executive_manager' => 'operations_executive_manager',
			'sn_operations_execution_expert' => 'operations_execution_expert',
		];
	}

}
