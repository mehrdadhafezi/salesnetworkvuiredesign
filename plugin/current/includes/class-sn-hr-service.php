<?php
if ( ! defined( 'ABSPATH' ) ) { exit; }

class SN_HR_Service {
	private wpdb $wpdb;
	private array $profile_by_user_id_cache = [];
	private array $position_slug_by_id_cache = [];
	private array $level_slug_by_id_cache = [];
	private array $resolved_panel_cache = [];
	private array $legacy_roles = [
		'sn_hr',
		'sn_mis',
		'sn_sales_deputy',
		'sn_sales_manager',
		'sas_sales_manager',
		'sn_senior_supervisor',
		'sas_senior_supervisor',
		'sn_supervisor',
		'sas_supervisor',
		'sn_converter',
		'sn_financial',
		'sn_financial_approval',
		'sn_finance',
		'sn_after_sales',
		'sn_project_manager',
		'sn_project_expert',
		'sn_shipping_expert',
		'sn_operations_sales_manager',
		'sn_operations_sales_supervisor',
		'sn_operations_sales_expert',
		'sn_operations_executive_manager',
		'sn_operations_execution_expert',
		'sn_seller',
		'sas_employee',
	];

	public function __construct() {
		global $wpdb;
		$this->wpdb = $wpdb;
	}

	/** Prime only the displayed workforce, avoiding per-row profile/lookup queries. */
	public function prime_workforce_rows(array $rows): void {
		$user_ids = [];
		foreach ($rows as $row) {
			$user_id = (int) ($row->user_id ?? 0);
			if (! $user_id) { continue; }
			$this->profile_by_user_id_cache[$user_id] = $row;
			unset($this->resolved_panel_cache[$user_id]);
			if (! empty($row->position_id)) { $this->position_slug_by_id_cache[(int) $row->position_id] = (string) ($row->position_slug ?? ''); }
			if (! empty($row->level_id)) { $this->level_slug_by_id_cache[(int) $row->level_id] = (string) ($row->level_slug ?? ''); }
			$user_ids[] = $user_id;
			if (! empty($row->manager_user_id)) { $user_ids[] = (int) $row->manager_user_id; }
		}
		if ($user_ids) { cache_users(array_values(array_unique($user_ids))); }
	}

	public function get_position_by_slug( string $slug ) {
		return $this->wpdb->get_row( $this->wpdb->prepare( "SELECT * FROM {$this->wpdb->prefix}sn_hr_positions WHERE slug=%s", sanitize_key( $slug ) ) );
	}

	public function get_level_by_slug( string $slug ) {
		return $this->wpdb->get_row( $this->wpdb->prepare( "SELECT * FROM {$this->wpdb->prefix}sn_hr_levels WHERE slug=%s", sanitize_key( $slug ) ) );
	}

	public function get_or_create_profile_for_user( int $user_id ) {
		$profile = $this->get_profile_by_user_id( $user_id );
		if ( $profile ) {
			return $profile;
		}
		$user = get_user_by( 'id', $user_id );
		$this->wpdb->insert( $this->wpdb->prefix . 'sn_hr_profiles', [
			'user_id' => $user_id,
			'hr_display_name' => $user instanceof WP_User ? (string) $user->display_name : '',
			'employment_status' => 'active',
			'is_active' => 1,
		] );
		$profile_id = (int) $this->wpdb->insert_id;
		unset( $this->profile_by_user_id_cache[ $user_id ], $this->resolved_panel_cache[ $user_id ] );
		$this->log_profile_event( $profile_id, $user_id, 0, 'profile_created', null, [ 'user_id' => $user_id ], [ 'source' => 'get_or_create_profile_for_user' ] );
		return $this->get_profile_by_user_id( $user_id );
	}

	public function get_profile_by_user_id( int $user_id ) {
		if ( array_key_exists( $user_id, $this->profile_by_user_id_cache ) ) {
			return $this->profile_by_user_id_cache[ $user_id ];
		}
		$profile = $this->wpdb->get_row( $this->wpdb->prepare( "SELECT * FROM {$this->wpdb->prefix}sn_hr_profiles WHERE user_id=%d LIMIT 1", $user_id ) );
		$this->profile_by_user_id_cache[ $user_id ] = $profile ?: null;
		return $this->profile_by_user_id_cache[ $user_id ];
	}

	public function update_profile_position_level( int $user_id, int $position_id, int $level_id = 0, int $actor_user_id = 0 ) {
		$profile = $this->get_or_create_profile_for_user( $user_id );
		if ( ! $profile ) { return null; }
		$old = [
			'position_id' => (int) $profile->position_id,
			'level_id' => (int) $profile->level_id,
		];
		$data = [
			'position_id' => $position_id ?: null,
			'level_id' => $level_id ?: null,
		];
		$this->wpdb->update( $this->wpdb->prefix . 'sn_hr_profiles', $data, [ 'id' => (int) $profile->id ] );
		unset( $this->profile_by_user_id_cache[ $user_id ], $this->resolved_panel_cache[ $user_id ] );
		$this->log_profile_event( (int) $profile->id, $user_id, $actor_user_id, 'position_level_updated', $old, $data, [] );
		return $this->get_profile_by_user_id( $user_id );
	}

	public function get_mapping_for_legacy_role( string $role ) {
		return $this->wpdb->get_row( $this->wpdb->prepare(
			"SELECT m.*, p.slug position_slug, p.label position_label, l.slug level_slug, l.label level_label
			FROM {$this->wpdb->prefix}sn_hr_position_role_mappings m
			LEFT JOIN {$this->wpdb->prefix}sn_hr_positions p ON p.id=m.position_id
			LEFT JOIN {$this->wpdb->prefix}sn_hr_levels l ON l.id=m.default_level_id
			WHERE m.legacy_role=%s AND m.is_active=1",
			sanitize_key( $role )
		) );
	}

	public function infer_primary_legacy_role( WP_User $user ): string {
		$user_roles = array_unique( array_merge( (array) $user->roles, array_keys( (array) $user->caps ) ) );
		foreach ( $this->legacy_roles as $role ) {
			if ( in_array( $role, $user_roles, true ) ) {
				return $role;
			}
		}
		return '';
	}

	public function sync_user_to_hr_profile( int $user_id, int $actor_user_id = 0, bool $force = false ): array {
		$user = get_user_by( 'id', $user_id );
		if ( ! $user instanceof WP_User ) {
			return [ 'status' => 'error', 'message' => 'user_not_found' ];
		}
		$legacy_role = $this->infer_primary_legacy_role( $user );
		if ( $legacy_role === '' ) {
			return [ 'status' => 'skipped', 'message' => 'no_legacy_role' ];
		}
		$mapping = $this->get_mapping_for_legacy_role( $legacy_role );
		if ( ! $mapping ) {
			return [ 'status' => 'skipped', 'message' => 'no_mapping', 'legacy_role' => $legacy_role ];
		}
		$profile = $this->get_profile_by_user_id( $user_id );
		$created = false;
		if ( ! $profile ) {
			$this->wpdb->insert( $this->wpdb->prefix . 'sn_hr_profiles', [
				'user_id' => $user_id,
				'employment_status' => 'active',
				'is_active' => 1,
				'legacy_role' => $legacy_role,
				'legacy_source' => 'wordpress_role',
				'position_id' => (int) $mapping->position_id,
				'level_id' => $mapping->default_level_id ? (int) $mapping->default_level_id : null,
			] );
			$profile = $this->get_profile_by_user_id( $user_id );
			$created = true;
			$this->log_profile_event( (int) $profile->id, $user_id, $actor_user_id, 'profile_synced_created', null, [
				'legacy_role' => $legacy_role,
				'position_id' => (int) $mapping->position_id,
				'level_id' => $mapping->default_level_id ? (int) $mapping->default_level_id : null,
			], [ 'source' => 'legacy_role_sync' ] );
			return [ 'status' => 'created', 'profile_id' => (int) $profile->id, 'legacy_role' => $legacy_role ];
		}

		$data = [
			'legacy_role' => $profile->legacy_role ?: $legacy_role,
			'legacy_source' => $profile->legacy_source ?: 'wordpress_role',
		];
		if ( $force || ! $profile->position_id ) {
			$data['position_id'] = (int) $mapping->position_id;
		}
		if ( $force || ! $profile->level_id ) {
			$data['level_id'] = $mapping->default_level_id ? (int) $mapping->default_level_id : null;
		}
		$old = [
			'legacy_role' => $profile->legacy_role,
			'legacy_source' => $profile->legacy_source,
			'position_id' => (int) $profile->position_id,
			'level_id' => (int) $profile->level_id,
		];
		$this->wpdb->update( $this->wpdb->prefix . 'sn_hr_profiles', $data, [ 'id' => (int) $profile->id ] );
		$changed = (bool) $this->wpdb->rows_affected;
		if ( $changed ) {
			$this->log_profile_event( (int) $profile->id, $user_id, $actor_user_id, 'profile_synced_updated', $old, $data, [ 'source' => 'legacy_role_sync' ] );
		}
		return [ 'status' => $changed ? 'updated' : 'skipped', 'profile_id' => (int) $profile->id, 'legacy_role' => $legacy_role, 'created' => $created ];
	}

	public function sync_all_legacy_users( int $limit = 100, int $offset = 0, int $actor_user_id = 0 ): array {
		$users = get_users( [
			'number' => max( 1, min( 500, $limit ) ),
			'offset' => max( 0, $offset ),
			'orderby' => 'ID',
			'order' => 'ASC',
			'fields' => 'all',
		] );
		$summary = [
			'created' => 0,
			'updated' => 0,
			'skipped' => 0,
			'error' => 0,
			'processed' => 0,
			'skipped_details' => [],
		];
		foreach ( $users as $user ) {
			if ( ! $user instanceof WP_User ) { continue; }
			$result = $this->sync_user_to_hr_profile( (int) $user->ID, $actor_user_id );
			$summary['processed']++;
			$status = (string) ( $result['status'] ?? 'error' );
			if ( isset( $summary[ $status ] ) ) {
				$summary[ $status ]++;
			} else {
				$summary['error']++;
			}
			if ( $status === 'skipped' ) {
				$summary['skipped_details'][] = [
					'user_id' => (int) $user->ID,
					'username' => (string) $user->user_login,
					'display_name' => (string) $user->display_name,
					'roles' => array_values( (array) $user->roles ),
					'reason' => (string) ( $result['message'] ?? 'skipped' ),
				];
			}
		}
		update_option( 'sn_hr_last_sync_report', array_merge( $summary, [
			'limit' => $limit,
			'offset' => $offset,
			'actor_user_id' => $actor_user_id,
			'updated_at' => current_time( 'mysql' ),
		] ), false );
		return $summary;
	}

	public function count_legacy_users(): int {
		$count = 0;
		foreach ( get_users( [ 'fields' => 'all', 'number' => -1 ] ) as $user ) {
			if ( $user instanceof WP_User && $this->infer_primary_legacy_role( $user ) !== '' ) {
				$count++;
			}
		}
		return $count;
	}

	public function legacy_roles(): array {
		return $this->legacy_roles;
	}

	public function transition_legacy_supervisors_to_senior( bool $dry_run = true, int $actor_user_id = 0 ): array {
		$senior_position = $this->get_position_by_slug( 'senior_supervisor' );
		if ( ! $senior_position ) {
			return [ 'mode' => $dry_run ? 'dry_run' : 'apply', 'scanned' => 0, 'eligible' => 0, 'changed' => 0, 'skipped_already_senior' => 0, 'skipped_no_profile' => 0, 'errors' => 1, 'message' => 'senior_supervisor_position_missing', 'affected_users' => [] ];
		}
		$user_ids = array_map( 'intval', get_users( [ 'role' => 'sn_supervisor', 'fields' => 'ID', 'number' => -1 ] ) );
		$legacy_profile_user_ids = array_map( 'intval', $this->wpdb->get_col( "SELECT user_id FROM {$this->wpdb->prefix}sn_hr_profiles WHERE legacy_role='sn_supervisor'" ) );
		$user_ids = array_values( array_unique( array_merge( $user_ids, $legacy_profile_user_ids ) ) );
		sort( $user_ids );
		$report = [
			'mode' => $dry_run ? 'dry_run' : 'apply',
			'scanned' => 0,
			'eligible' => 0,
			'changed' => 0,
			'skipped_already_senior' => 0,
			'skipped_no_profile' => 0,
			'errors' => 0,
			'affected_users' => [],
			'updated_at' => current_time( 'mysql' ),
		];
		foreach ( $user_ids as $user_id ) {
			$report['scanned']++;
			$profile = $this->get_profile_by_user_id( $user_id );
			if ( ! $profile ) {
				$report['skipped_no_profile']++;
				continue;
			}
			$current_position = $this->position_slug_for_profile( (int) $profile->id );
			if ( $current_position === 'senior_supervisor' ) {
				$report['skipped_already_senior']++;
				continue;
			}
			if ( $current_position !== 'supervisor' ) {
				continue;
			}
			$report['eligible']++;
			$this->append_affected_user( $report, $user_id, $current_position, 'senior_supervisor' );
			if ( $dry_run ) {
				continue;
			}
			$updated = $this->update_profile_position_level( $user_id, (int) $senior_position->id, (int) ( $profile->level_id ?? 0 ), $actor_user_id );
			if ( $updated ) {
				$report['changed']++;
			} else {
				$report['errors']++;
			}
		}
		return $report;
	}

	public function assign_sales_deputy_position( int $user_id, bool $dry_run = true, int $actor_user_id = 0 ): array {
		$user = get_user_by( 'id', $user_id );
		$position = $this->get_position_by_slug( 'sales_deputy' );
		$report = [
			'mode' => $dry_run ? 'dry_run' : 'apply',
			'user_found' => $user instanceof WP_User,
			'user_id' => $user_id,
			'had_profile' => false,
			'old_position' => '',
			'new_position' => 'sales_deputy',
			'changed' => false,
			'created' => false,
			'errors' => 0,
			'updated_at' => current_time( 'mysql' ),
		];
		if ( ! $user instanceof WP_User || ! $position ) {
			$report['errors']++;
			$report['message'] = ! $position ? 'sales_deputy_position_missing' : 'user_not_found';
			return $report;
		}
		$profile = $this->get_profile_by_user_id( $user_id );
		$report['had_profile'] = (bool) $profile;
		if ( $profile ) {
			$report['old_position'] = $this->position_slug_for_profile( (int) $profile->id );
		}
		if ( $report['old_position'] === 'sales_deputy' ) {
			return $report;
		}
		if ( $dry_run ) {
			$report['changed'] = true;
			$report['created'] = ! $profile;
			return $report;
		}
		if ( ! $profile ) {
			$profile = $this->get_or_create_profile_for_user( $user_id );
			$report['created'] = (bool) $profile;
		}
		if ( ! $profile ) {
			$report['errors']++;
			$report['message'] = 'profile_create_failed';
			return $report;
		}
		$updated = $this->update_profile_position_level( $user_id, (int) $position->id, (int) ( $profile->level_id ?? 0 ), $actor_user_id );
		$report['changed'] = (bool) $updated;
		if ( ! $updated ) {
			$report['errors']++;
		}
		return $report;
	}

	public function manual_position_correction( int $user_id, string $target_position_slug, bool $create_missing_profile = false, bool $dry_run = true, int $actor_user_id = 0, string $reason = '' ): array {
		$user = get_user_by( 'id', $user_id );
		$target_position = $this->get_position_by_slug( $target_position_slug );
		$profile = $this->get_profile_by_user_id( $user_id );
		$legacy_role = $user instanceof WP_User ? $this->infer_primary_legacy_role( $user ) : '';
		$old_position = $profile ? $this->position_slug_for_profile( (int) $profile->id ) : '';
		$report = [
			'mode' => $dry_run ? 'dry_run' : 'apply',
			'user_found' => $user instanceof WP_User,
			'user_id' => $user_id,
			'display_name' => $user instanceof WP_User ? (string) $user->display_name : '',
			'wp_roles' => $user instanceof WP_User ? array_values( (array) $user->roles ) : [],
			'legacy_role' => $profile && ! empty( $profile->legacy_role ) ? (string) $profile->legacy_role : $legacy_role,
			'had_profile' => (bool) $profile,
			'old_position' => $old_position,
			'new_position' => sanitize_key( $target_position_slug ),
			'would_change_position' => $old_position !== sanitize_key( $target_position_slug ),
			'changed_position' => false,
			'created_profile' => false,
			'errors' => 0,
			'logs_created' => 0,
			'warnings' => [],
			'updated_at' => current_time( 'mysql' ),
		];
		if ( ! $user instanceof WP_User ) {
			$report['errors']++;
			$report['message'] = 'user_not_found';
			return $report;
		}
		if ( ! $target_position ) {
			$report['errors']++;
			$report['message'] = 'target_position_not_found';
			return $report;
		}
		if ( ! $profile && ! $create_missing_profile ) {
			$report['errors']++;
			$report['message'] = 'profile_missing';
			$report['warnings'][] = 'HR profile does not exist; enable create_missing_profile to create one.';
			return $report;
		}
		if ( $dry_run ) {
			$report['created_profile'] = ! $profile && $create_missing_profile;
			return $report;
		}
		if ( ! $profile ) {
			$this->wpdb->insert( $this->wpdb->prefix . 'sn_hr_profiles', [
				'user_id' => $user_id,
				'employment_status' => 'active',
				'is_active' => 1,
				'legacy_role' => $legacy_role ?: null,
				'legacy_source' => $legacy_role ? 'wordpress_role' : 'manual_hr_position_correction',
				'position_id' => (int) $target_position->id,
			] );
			$profile = $this->get_profile_by_user_id( $user_id );
			$report['created_profile'] = (bool) $profile;
			$report['changed_position'] = (bool) $profile;
			if ( $profile ) {
				$this->log_profile_event( (int) $profile->id, $user_id, $actor_user_id, 'manual_hr_position_correction', null, [
					'position_id' => (int) $target_position->id,
					'position_slug' => sanitize_key( $target_position_slug ),
				], [ 'source' => 'manual_hr_position_correction', 'reason' => $reason ] );
				$report['logs_created']++;
			}
			return $report;
		}
		if ( $old_position === sanitize_key( $target_position_slug ) ) {
			return $report;
		}
		$old = [
			'position_id' => (int) $profile->position_id,
			'position_slug' => $old_position,
			'level_id' => (int) $profile->level_id,
			'legacy_role' => (string) $profile->legacy_role,
		];
		$data = [
			'position_id' => (int) $target_position->id,
			'level_id' => $profile->level_id ? (int) $profile->level_id : null,
		];
		$this->wpdb->update( $this->wpdb->prefix . 'sn_hr_profiles', $data, [ 'id' => (int) $profile->id ] );
		$report['changed_position'] = (bool) $this->wpdb->rows_affected;
		$this->log_profile_event( (int) $profile->id, $user_id, $actor_user_id, 'manual_hr_position_correction', $old, [
			'position_id' => (int) $target_position->id,
			'position_slug' => sanitize_key( $target_position_slug ),
			'level_id' => $profile->level_id ? (int) $profile->level_id : null,
			'legacy_role' => (string) $profile->legacy_role,
		], [ 'source' => 'manual_hr_position_correction', 'reason' => $reason ] );
		$report['logs_created']++;
		return $report;
	}

	public function target_organization_counts(): array {
		$positions = [];
		foreach ( [ 'sales_deputy', 'sales_manager', 'senior_supervisor', 'supervisor', 'converter', 'seller', 'finance', 'hr', 'after_sales', 'mis' ] as $slug ) {
			$positions[ $slug ] = (int) $this->wpdb->get_var( $this->wpdb->prepare(
				"SELECT COUNT(*) FROM {$this->wpdb->prefix}sn_hr_profiles p INNER JOIN {$this->wpdb->prefix}sn_hr_positions pos ON pos.id=p.position_id WHERE pos.slug=%s",
				$slug
			) );
		}
		return [
			'positions' => $positions,
			'legacy_sn_supervisor_users' => (int) count( get_users( [ 'role' => 'sn_supervisor', 'fields' => 'ID', 'number' => -1 ] ) ),
			'hr_supervisor_profiles' => $positions['supervisor'] ?? 0,
			'hr_senior_supervisor_profiles' => $positions['senior_supervisor'] ?? 0,
			'sales_manager_profiles' => $positions['sales_manager'] ?? 0,
			'sales_deputy_profiles' => $positions['sales_deputy'] ?? 0,
		];
	}

	public function resolve_panel_for_user( int $user_id ): array {
		if ( isset( $this->resolved_panel_cache[ $user_id ] ) ) {
			return $this->resolved_panel_cache[ $user_id ];
		}
		$user = get_user_by( 'id', $user_id );
		$profile = $this->get_profile_by_user_id( $user_id );
		$legacy_role = $user instanceof WP_User ? $this->infer_primary_legacy_role( $user ) : '';
		$position_slug = '';
		$level_slug = '';
		$fallback_used = false;
		$fallback_reason = '';

		if ( $profile && ! empty( $profile->position_id ) ) {
			$position_id = (int) $profile->position_id;
			if ( ! array_key_exists( $position_id, $this->position_slug_by_id_cache ) ) {
				$this->position_slug_by_id_cache[ $position_id ] = (string) $this->wpdb->get_var( $this->wpdb->prepare(
					"SELECT slug FROM {$this->wpdb->prefix}sn_hr_positions WHERE id=%d LIMIT 1",
					$position_id
				) );
			}
			$position_slug = (string) $this->position_slug_by_id_cache[ $position_id ];
		}
		if ( $profile && ! empty( $profile->level_id ) ) {
			$level_id = (int) $profile->level_id;
			if ( ! array_key_exists( $level_id, $this->level_slug_by_id_cache ) ) {
				$this->level_slug_by_id_cache[ $level_id ] = (string) $this->wpdb->get_var( $this->wpdb->prepare(
					"SELECT slug FROM {$this->wpdb->prefix}sn_hr_levels WHERE id=%d LIMIT 1",
					$level_id
				) );
			}
			$level_slug = (string) $this->level_slug_by_id_cache[ $level_id ];
		}

		if ( $position_slug === '' ) {
			$fallback_used = true;
			$fallback_reason = $profile ? 'missing_hr_position' : 'missing_hr_profile';
			$position_slug = $this->fallback_position_from_legacy_role( $legacy_role, $user );
		}

		$map = $this->panel_mapping();
		$resolved = $map[ $position_slug ] ?? $map['unknown'];
		$result = [
			'user_id' => $user_id,
			'wp_roles' => $user instanceof WP_User ? array_values( (array) $user->roles ) : [],
			'legacy_role' => $profile && ! empty( $profile->legacy_role ) ? (string) $profile->legacy_role : $legacy_role,
			'has_hr_profile' => (bool) $profile,
			'position_slug' => $position_slug,
			'level_slug' => $level_slug,
			'panel_key' => (string) $resolved['panel_key'],
			'panel_label' => (string) $resolved['panel_label'],
			'shortcode' => (string) $resolved['shortcode'],
			'route_type' => (string) $resolved['route_type'],
			'fallback_used' => $fallback_used,
			'fallback_reason' => $fallback_reason,
			'warnings' => $this->panel_resolver_warnings( $position_slug, $fallback_used, $fallback_reason ),
		];
		$this->resolved_panel_cache[ $user_id ] = $result;
		return $result;
	}

	public function panel_mapping(): array {
		return [
			'seller' => [ 'panel_key' => 'seller_panel', 'panel_label' => 'پنل فروشنده', 'shortcode' => 'sn_seller_panel', 'route_type' => 'existing_shortcode' ],
			'supervisor' => [ 'panel_key' => 'supervisor_panel', 'panel_label' => 'پنل سرپرست', 'shortcode' => 'sn_supervisor_panel', 'route_type' => 'existing_shortcode' ],
			'converter' => [ 'panel_key' => 'converter_panel', 'panel_label' => 'پنل تبدیل‌کننده', 'shortcode' => 'sn_dot_converter_panel', 'route_type' => 'real_shortcode' ],
			'senior_supervisor' => [ 'panel_key' => 'senior_supervisor_panel', 'panel_label' => 'پنل سرپرست ارشد', 'shortcode' => 'sn_senior_supervisor_panel', 'route_type' => 'real_shortcode' ],
			'sales_manager' => [ 'panel_key' => 'sales_manager_panel', 'panel_label' => 'پنل مدیر فروش', 'shortcode' => 'sn_sales_manager_panel', 'route_type' => 'existing_shortcode' ],
			'sales_deputy' => [ 'panel_key' => 'sales_deputy_panel', 'panel_label' => 'پنل معاون فروش', 'shortcode' => 'sn_sales_deputy_panel', 'route_type' => 'real_shortcode' ],
			'finance' => [ 'panel_key' => 'finance_panel', 'panel_label' => 'پنل مالی', 'shortcode' => 'sn_financial_panel', 'route_type' => 'existing_shortcode' ],
			'hr' => [ 'panel_key' => 'hr_panel', 'panel_label' => 'پنل منابع انسانی', 'shortcode' => 'sn_hr_panel', 'route_type' => 'real_shortcode' ],
			'mis' => [ 'panel_key' => 'mis_panel', 'panel_label' => 'پنل MIS', 'shortcode' => 'sn_mis_panel', 'route_type' => 'real_shortcode' ],
			'after_sales' => [ 'panel_key' => 'after_sales_panel', 'panel_label' => 'پنل خدمات پس از فروش', 'shortcode' => 'sn_after_sales_panel', 'route_type' => 'real_shortcode' ],
			// Retired position aliases are kept only so an account missed by the
			// one-time migration still lands in the replacement operations panel.
			'project_manager' => [ 'panel_key' => 'operations_sales_manager_panel', 'panel_label' => 'پنل مدیر فروش عملیات', 'shortcode' => 'sn_operations_sales_manager_panel', 'route_type' => 'legacy_alias' ],
			'project_expert' => [ 'panel_key' => 'operations_sales_expert_panel', 'panel_label' => 'پنل کارشناس فروش عملیات', 'shortcode' => 'sn_operations_sales_expert_panel', 'route_type' => 'legacy_alias' ],
			'operations_sales_manager' => [ 'panel_key' => 'operations_sales_manager_panel', 'panel_label' => 'پنل مدیر فروش عملیات', 'shortcode' => 'sn_operations_sales_manager_panel', 'route_type' => 'real_shortcode' ],
			'operations_sales_supervisor' => [ 'panel_key' => 'operations_sales_supervisor_panel', 'panel_label' => 'پنل سرپرست فروش عملیات', 'shortcode' => 'sn_operations_sales_supervisor_panel', 'route_type' => 'real_shortcode' ],
			'operations_sales_expert' => [ 'panel_key' => 'operations_sales_expert_panel', 'panel_label' => 'پنل کارشناس فروش عملیات', 'shortcode' => 'sn_operations_sales_expert_panel', 'route_type' => 'real_shortcode' ],
			'operations_executive_manager' => [ 'panel_key' => 'operations_executive_manager_panel', 'panel_label' => 'پنل مدیر اجرایی عملیات', 'shortcode' => 'sn_operations_executive_manager_panel', 'route_type' => 'real_shortcode' ],
			'operations_execution_expert' => [ 'panel_key' => 'operations_execution_expert_panel', 'panel_label' => 'پنل کارشناس اجرایی عملیات', 'shortcode' => 'sn_operations_execution_expert_panel', 'route_type' => 'real_shortcode' ],
			'shipping_expert' => [ 'panel_key' => 'shipping_panel', 'panel_label' => 'پنل ارسال', 'shortcode' => 'sn_shipping_panel', 'route_type' => 'real_shortcode' ],
			'admin_dashboard' => [ 'panel_key' => 'admin_dashboard', 'panel_label' => 'داشبورد مدیریت شبکه فروش', 'shortcode' => 'sn_admin_front_dashboard', 'route_type' => 'frontend_admin_dashboard' ],
			'unknown' => [ 'panel_key' => 'unknown_panel', 'panel_label' => 'پنل نامشخص', 'shortcode' => '', 'route_type' => 'placeholder' ],
		];
	}

	private function fallback_position_from_legacy_role( string $legacy_role, $user ): string {
		if ( in_array( $legacy_role, [ 'sn_seller', 'sas_employee' ], true ) ) { return 'seller'; }
		if ( in_array( $legacy_role, [ 'sn_supervisor', 'sas_supervisor' ], true ) ) { return 'supervisor'; }
		if ( $legacy_role === 'sn_converter' ) { return 'converter'; }
		if ( in_array( $legacy_role, [ 'sn_senior_supervisor', 'sas_senior_supervisor' ], true ) ) { return 'senior_supervisor'; }
		if ( in_array( $legacy_role, [ 'sn_sales_manager', 'sas_sales_manager' ], true ) ) { return 'sales_manager'; }
		if ( $legacy_role === 'sn_sales_deputy' ) { return 'sales_deputy'; }
		if ( in_array( $legacy_role, [ 'sn_financial', 'sn_financial_approval', 'sn_finance' ], true ) ) { return 'finance'; }
		if ( $legacy_role === 'sn_hr' ) { return 'hr'; }
		if ( $legacy_role === 'sn_mis' ) { return 'mis'; }
		if ( $legacy_role === 'sn_after_sales' ) { return 'after_sales'; }
		if ( $legacy_role === 'sn_project_manager' ) { return 'operations_sales_manager'; }
		if ( $legacy_role === 'sn_project_expert' ) { return 'operations_sales_expert'; }
		if ( $legacy_role === 'sn_shipping_expert' ) { return 'shipping_expert'; }
		if ( $legacy_role === 'sn_operations_sales_manager' ) { return 'operations_sales_manager'; }
		if ( $legacy_role === 'sn_operations_sales_supervisor' ) { return 'operations_sales_supervisor'; }
		if ( $legacy_role === 'sn_operations_sales_expert' ) { return 'operations_sales_expert'; }
		if ( $legacy_role === 'sn_operations_executive_manager' ) { return 'operations_executive_manager'; }
		if ( $legacy_role === 'sn_operations_execution_expert' ) { return 'operations_execution_expert'; }
		if ( $user instanceof WP_User && user_can( $user, 'manage_options' ) ) { return 'admin_dashboard'; }
		return 'unknown';
	}

	private function panel_resolver_warnings( string $position_slug, bool $fallback_used, string $fallback_reason ): array {
		$warnings = [];
		if ( $fallback_used ) {
			$warnings[] = 'user has no complete HR profile and legacy fallback was used: ' . $fallback_reason;
		}
		if ( $position_slug === 'senior_supervisor' ) {
			$warnings[] = 'پنل سرپرست ارشد فعلاً فقط خواندنی است.';
		}
		if ( $position_slug === 'sales_deputy' ) {
			$warnings[] = 'پنل معاون فروش فعلاً فقط خواندنی است.';
		}
		if ( $position_slug === 'hr' ) {
			$warnings[] = 'پنل منابع انسانی در فاز پایه ساختار سازمانی است.';
		}
		if ( $position_slug === 'mis' ) {
			$warnings[] = 'پنل MIS در فاز پایه ورود و staging دیتا است.';
		}
		if ( $position_slug === 'after_sales' ) {
			$warnings[] = 'پنل خدمات پس از فروش در فاز پایه پیگیری و ثبت پرونده است.';
		}
		return $warnings;
	}

	private function log_profile_event( int $profile_id, int $user_id, int $actor_user_id, string $action, $old_value, $new_value, array $context ): void {
		$this->wpdb->insert( $this->wpdb->prefix . 'sn_hr_profile_logs', [
			'profile_id' => $profile_id ?: null,
			'user_id' => $user_id ?: null,
			'actor_user_id' => $actor_user_id ?: null,
			'action' => sanitize_key( $action ),
			'old_value' => null === $old_value ? null : wp_json_encode( $old_value ),
			'new_value' => null === $new_value ? null : wp_json_encode( $new_value ),
			'context' => $context ? wp_json_encode( $context ) : null,
		] );
	}

	private function position_slug_for_profile( int $profile_id ): string {
		return (string) $this->wpdb->get_var( $this->wpdb->prepare(
			"SELECT pos.slug FROM {$this->wpdb->prefix}sn_hr_profiles p INNER JOIN {$this->wpdb->prefix}sn_hr_positions pos ON pos.id=p.position_id WHERE p.id=%d",
			$profile_id
		) );
	}

	private function append_affected_user( array &$report, int $user_id, string $old_position, string $new_position ): void {
		if ( count( $report['affected_users'] ) >= 100 ) {
			return;
		}
		$user = get_user_by( 'id', $user_id );
		$report['affected_users'][] = [
			'user_id' => $user_id,
			'display_name' => $user instanceof WP_User ? (string) $user->display_name : '',
			'username' => $user instanceof WP_User ? (string) $user->user_login : '',
			'old_position' => $old_position,
			'new_position' => $new_position,
		];
	}
}
