<?php
if ( ! defined( 'ABSPATH' ) ) { exit; }

class SN_Hierarchy_Service {
	private wpdb $wpdb;
	private SN_HR_Service $hr;

	public function __construct() {
		global $wpdb;
		$this->wpdb = $wpdb;
		$this->hr = new SN_HR_Service();
	}

	public function get_current_parent_profile_id( int $child_profile_id ): int {
		return (int) $this->wpdb->get_var( $this->wpdb->prepare(
			"SELECT parent_profile_id FROM {$this->wpdb->prefix}sn_hr_assignments WHERE child_profile_id=%d AND relationship_type='reports_to' AND is_current=1 ORDER BY id DESC LIMIT 1",
			$child_profile_id
		) );
	}

	public function get_current_parent_user_id( int $child_user_id ): int {
		$profile = $this->hr->get_profile_by_user_id( $child_user_id );
		if ( ! $profile ) { return 0; }
		$parent_profile_id = $this->get_current_parent_profile_id( (int) $profile->id );
		if ( ! $parent_profile_id ) { return 0; }
		return (int) $this->wpdb->get_var( $this->wpdb->prepare( "SELECT user_id FROM {$this->wpdb->prefix}sn_hr_profiles WHERE id=%d", $parent_profile_id ) );
	}

	public function assign_parent( int $child_profile_id, int $parent_profile_id, int $actor_user_id = 0, string $source = 'manual', string $reason = '' ): array {
		$child_profile_id = absint( $child_profile_id );
		$parent_profile_id = absint( $parent_profile_id );
		if ( ! $child_profile_id || ! $parent_profile_id ) {
			return [ 'success' => false, 'message' => 'profile_missing' ];
		}
		if ( $child_profile_id === $parent_profile_id ) {
			return [ 'success' => false, 'message' => 'self_parent_not_allowed' ];
		}
		if ( ! $this->profile_exists( $child_profile_id ) || ! $this->profile_exists( $parent_profile_id ) ) {
			return [ 'success' => false, 'message' => 'profile_not_found' ];
		}
		if ( $this->would_create_cycle( $child_profile_id, $parent_profile_id ) ) {
			return [ 'success' => false, 'message' => 'cycle_not_allowed' ];
		}

		$current_parent = $this->get_current_parent_profile_id( $child_profile_id );
		if ( $current_parent === $parent_profile_id ) {
			return [ 'success' => true, 'message' => 'unchanged', 'skipped' => true ];
		}
		if ( $current_parent ) {
			$this->end_current_assignment( $child_profile_id, $actor_user_id, $reason ?: 'changed_parent' );
		}
		$this->wpdb->insert( $this->wpdb->prefix . 'sn_hr_assignments', [
			'child_profile_id' => $child_profile_id,
			'parent_profile_id' => $parent_profile_id,
			'relationship_type' => 'reports_to',
			'effective_from' => current_time( 'mysql' ),
			'is_current' => 1,
			'source' => sanitize_key( $source ),
			'created_by' => $actor_user_id ?: null,
		] );
		$assignment_id = (int) $this->wpdb->insert_id;
		$this->log_history( $assignment_id, $child_profile_id, $current_parent ?: null, $parent_profile_id, $actor_user_id, $current_parent ? 'changed' : 'created', $reason, [ 'source' => $source ] );
		return [ 'success' => true, 'message' => 'assigned', 'assignment_id' => $assignment_id ];
	}

	public function end_current_assignment( int $child_profile_id, int $actor_user_id = 0, string $reason = '' ): bool {
		$current = $this->wpdb->get_row( $this->wpdb->prepare(
			"SELECT * FROM {$this->wpdb->prefix}sn_hr_assignments WHERE child_profile_id=%d AND relationship_type='reports_to' AND is_current=1 ORDER BY id DESC LIMIT 1",
			$child_profile_id
		) );
		if ( ! $current ) { return false; }
		$this->wpdb->update( $this->wpdb->prefix . 'sn_hr_assignments', [
			'is_current' => 0,
			'effective_to' => current_time( 'mysql' ),
		], [ 'id' => (int) $current->id ] );
		$this->log_history( (int) $current->id, $child_profile_id, (int) $current->parent_profile_id, null, $actor_user_id, 'ended', $reason, [] );
		return true;
	}

	public function get_direct_children_profile_ids( int $parent_profile_id ): array {
		return array_map( 'intval', $this->wpdb->get_col( $this->wpdb->prepare(
			"SELECT child_profile_id FROM {$this->wpdb->prefix}sn_hr_assignments WHERE parent_profile_id=%d AND relationship_type='reports_to' AND is_current=1",
			$parent_profile_id
		) ) );
	}

	public function get_descendant_profile_ids( int $parent_profile_id, int $max_depth = 10 ): array {
		$seen = [];
		$frontier = [ absint( $parent_profile_id ) ];
		for ( $depth = 0; $depth < max( 1, $max_depth ); $depth++ ) {
			$next = [];
			foreach ( $frontier as $profile_id ) {
				foreach ( $this->get_direct_children_profile_ids( $profile_id ) as $child_id ) {
					if ( ! isset( $seen[ $child_id ] ) ) {
						$seen[ $child_id ] = true;
						$next[] = $child_id;
					}
				}
			}
			if ( ! $next ) { break; }
			$frontier = $next;
		}
		return array_map( 'intval', array_keys( $seen ) );
	}

	public function get_ancestor_profile_ids( int $child_profile_id, int $max_depth = 10 ): array {
		$ancestors = [];
		$current = absint( $child_profile_id );
		for ( $depth = 0; $depth < max( 1, $max_depth ); $depth++ ) {
			$parent = $this->get_current_parent_profile_id( $current );
			if ( ! $parent || in_array( $parent, $ancestors, true ) ) { break; }
			$ancestors[] = $parent;
			$current = $parent;
		}
		return $ancestors;
	}

	public function would_create_cycle( int $child_profile_id, int $parent_profile_id ): bool {
		if ( $child_profile_id === $parent_profile_id ) { return true; }
		return in_array( $child_profile_id, $this->get_ancestor_profile_ids( $parent_profile_id, 20 ), true );
	}

	public function sync_legacy_supervisor_assignments( int $limit = 100, int $offset = 0, int $actor_user_id = 0 ): array {
		$users = get_users( [
			'number' => max( 1, min( 500, $limit ) ),
			'offset' => max( 0, $offset ),
			'orderby' => 'ID',
			'order' => 'ASC',
			'fields' => 'all',
		] );
		$summary = [ 'created' => 0, 'updated' => 0, 'skipped' => 0, 'error' => 0, 'processed' => 0 ];
		foreach ( $users as $user ) {
			if ( ! $user instanceof WP_User ) { continue; }
			$summary['processed']++;
			$supervisor_id = (int) get_user_meta( (int) $user->ID, 'sn_supervisor_id', true );
			if ( ! $supervisor_id ) { $summary['skipped']++; continue; }
			$this->hr->sync_user_to_hr_profile( (int) $user->ID, $actor_user_id );
			$this->hr->sync_user_to_hr_profile( $supervisor_id, $actor_user_id );
			$child = $this->hr->get_profile_by_user_id( (int) $user->ID );
			$parent = $this->hr->get_profile_by_user_id( $supervisor_id );
			if ( ! $child || ! $parent ) { $summary['error']++; continue; }
			$current = $this->get_current_parent_profile_id( (int) $child->id );
			$result = $this->assign_parent( (int) $child->id, (int) $parent->id, $actor_user_id, 'legacy_sn_supervisor_id', 'sync from user_meta sn_supervisor_id' );
			if ( empty( $result['success'] ) ) { $summary['error']++; continue; }
			if ( ! empty( $result['skipped'] ) ) { $summary['skipped']++; continue; }
			$summary[ $current ? 'updated' : 'created' ]++;
			$this->log_history( (int) ( $result['assignment_id'] ?? 0 ), (int) $child->id, $current ?: null, (int) $parent->id, $actor_user_id, 'synced_from_legacy', 'sync from user_meta sn_supervisor_id', [ 'child_user_id' => (int) $user->ID, 'supervisor_user_id' => $supervisor_id ] );
		}
		update_option( 'sn_hr_last_hierarchy_sync_report', array_merge( $summary, [
			'limit' => $limit,
			'offset' => $offset,
			'actor_user_id' => $actor_user_id,
			'updated_at' => current_time( 'mysql' ),
		] ), false );
		return $summary;
	}

	public function bulk_assign_position_to_parent( int $parent_user_id, string $child_position_slug, bool $only_without_parent = true, bool $dry_run = true, int $actor_user_id = 0 ): array {
		$parent_profile = $this->hr->get_profile_by_user_id( $parent_user_id );
		$child_position_slug = sanitize_key( $child_position_slug );
		$report = [
			'mode' => $dry_run ? 'dry_run' : 'apply',
			'parent_user_id' => $parent_user_id,
			'child_position_slug' => $child_position_slug,
			'only_without_parent' => $only_without_parent,
			'scanned' => 0,
			'eligible' => 0,
			'assigned' => 0,
			'skipped_already_assigned' => 0,
			'skipped_cycle' => 0,
			'errors' => 0,
			'affected_users' => [],
			'updated_at' => current_time( 'mysql' ),
		];
		if ( ! $parent_profile ) {
			$report['errors']++;
			$report['message'] = 'parent_profile_missing';
			return $report;
		}
		$allowed_parent_positions = [
			'sales_manager' => [ 'sales_deputy' ],
			'senior_supervisor' => [ 'sales_manager' ],
			'supervisor' => [ 'sales_manager', 'senior_supervisor' ],
			'seller' => [ 'sales_manager', 'senior_supervisor', 'supervisor' ],
			'converter' => [ 'supervisor' ],
				'operations_sales_supervisor' => [ 'operations_sales_manager' ],
			'operations_sales_expert' => [ 'operations_sales_supervisor' ],
			'operations_execution_expert' => [ 'operations_executive_manager' ],
		][ $child_position_slug ] ?? [ 'sales_manager', 'supervisor' ];
		$parent_position = $this->position_slug_for_profile( (int) $parent_profile->id );
		if ( ! in_array( $parent_position, $allowed_parent_positions, true ) ) {
			$report['errors']++;
			$report['message'] = 'parent_position_not_allowed';
			$report['parent_position_slug'] = $parent_position;
			return $report;
		}
		$profiles = $this->profiles_by_position_slug( $child_position_slug );
		foreach ( $profiles as $profile ) {
			$report['scanned']++;
			$child_profile_id = (int) $profile->id;
			$child_user_id = (int) $profile->user_id;
			if ( $child_profile_id === (int) $parent_profile->id ) {
				$report['errors']++;
				continue;
			}
			$current_parent = $this->get_current_parent_profile_id( $child_profile_id );
			if ( $only_without_parent && $current_parent ) {
				continue;
			}
			$report['eligible']++;
			if ( $current_parent === (int) $parent_profile->id ) {
				$report['skipped_already_assigned']++;
				continue;
			}
			if ( $this->would_create_cycle( $child_profile_id, (int) $parent_profile->id ) ) {
				$report['skipped_cycle']++;
				continue;
			}
			$this->append_affected_user( $report, $child_user_id );
			if ( $dry_run ) {
				continue;
			}
			$result = $this->assign_parent( $child_profile_id, (int) $parent_profile->id, $actor_user_id, 'bulk_hierarchy_repair', 'bulk hierarchy repair' );
			if ( empty( $result['success'] ) ) {
				if ( ( $result['message'] ?? '' ) === 'cycle_not_allowed' ) {
					$report['skipped_cycle']++;
				} else {
					$report['errors']++;
				}
				continue;
			}
			if ( ! empty( $result['skipped'] ) ) {
				$report['skipped_already_assigned']++;
				continue;
			}
			$report['assigned']++;
		}
		return $report;
	}

	public function hierarchy_repair_audit_counts(): array {
		return [
			'top_level_supervisors' => (int) $this->wpdb->get_var(
				"SELECT COUNT(*) FROM {$this->wpdb->prefix}sn_hr_profiles p
				INNER JOIN {$this->wpdb->prefix}sn_hr_positions pos ON pos.id=p.position_id
				LEFT JOIN {$this->wpdb->prefix}sn_hr_assignments a ON a.child_profile_id=p.id AND a.relationship_type='reports_to' AND a.is_current=1
				WHERE pos.slug='supervisor' AND a.id IS NULL"
			),
			'unassigned_sellers' => (int) $this->wpdb->get_var(
				"SELECT COUNT(*) FROM {$this->wpdb->prefix}sn_hr_profiles p
				INNER JOIN {$this->wpdb->prefix}sn_hr_positions pos ON pos.id=p.position_id
				LEFT JOIN {$this->wpdb->prefix}sn_hr_assignments a ON a.child_profile_id=p.id AND a.relationship_type='reports_to' AND a.is_current=1
				WHERE pos.slug='seller' AND a.id IS NULL"
			),
			'sales_managers' => (int) $this->wpdb->get_var(
				"SELECT COUNT(*) FROM {$this->wpdb->prefix}sn_hr_profiles p
				INNER JOIN {$this->wpdb->prefix}sn_hr_positions pos ON pos.id=p.position_id
				WHERE pos.slug='sales_manager'"
			),
			'current_assignments' => (int) $this->wpdb->get_var(
				"SELECT COUNT(*) FROM {$this->wpdb->prefix}sn_hr_assignments WHERE relationship_type='reports_to' AND is_current=1"
			),
		];
	}

	public function target_hierarchy_summary(): array {
		return [
			'sales_deputy_to_sales_manager' => $this->count_current_relationship( 'sales_deputy', 'sales_manager' ),
			'sales_manager_to_senior_supervisor' => $this->count_current_relationship( 'sales_manager', 'senior_supervisor' ),
			'senior_supervisor_to_supervisor' => $this->count_current_relationship( 'senior_supervisor', 'supervisor' ),
			'supervisor_to_converter' => $this->count_current_relationship( 'supervisor', 'converter' ),
			'supervisor_to_seller' => $this->count_current_relationship( 'supervisor', 'seller' ),
			'senior_supervisor_to_seller_direct' => $this->count_current_relationship( 'senior_supervisor', 'seller' ),
			'sales_manager_to_seller_direct' => $this->count_current_relationship( 'sales_manager', 'seller' ),
		];
	}

	private function profile_exists( int $profile_id ): bool {
		return (bool) $this->wpdb->get_var( $this->wpdb->prepare( "SELECT id FROM {$this->wpdb->prefix}sn_hr_profiles WHERE id=%d", $profile_id ) );
	}

	private function position_slug_for_profile( int $profile_id ): string {
		return (string) $this->wpdb->get_var( $this->wpdb->prepare(
			"SELECT pos.slug FROM {$this->wpdb->prefix}sn_hr_profiles p INNER JOIN {$this->wpdb->prefix}sn_hr_positions pos ON pos.id=p.position_id WHERE p.id=%d",
			$profile_id
		) );
	}

	private function profiles_by_position_slug( string $position_slug ): array {
		return $this->wpdb->get_results( $this->wpdb->prepare(
			"SELECT p.* FROM {$this->wpdb->prefix}sn_hr_profiles p
			INNER JOIN {$this->wpdb->prefix}sn_hr_positions pos ON pos.id=p.position_id
			WHERE pos.slug=%s AND p.is_active=1
			ORDER BY p.user_id ASC",
			$position_slug
		) );
	}

	private function append_affected_user( array &$report, int $user_id ): void {
		if ( count( $report['affected_users'] ) >= 100 ) {
			return;
		}
		$user = get_user_by( 'id', $user_id );
		$report['affected_users'][] = [
			'user_id' => $user_id,
			'display_name' => $user instanceof WP_User ? (string) $user->display_name : '',
			'username' => $user instanceof WP_User ? (string) $user->user_login : '',
		];
	}

	private function count_current_relationship( string $parent_position_slug, string $child_position_slug ): int {
		return (int) $this->wpdb->get_var( $this->wpdb->prepare(
			"SELECT COUNT(*) FROM {$this->wpdb->prefix}sn_hr_assignments a
			INNER JOIN {$this->wpdb->prefix}sn_hr_profiles child ON child.id=a.child_profile_id
			INNER JOIN {$this->wpdb->prefix}sn_hr_positions child_pos ON child_pos.id=child.position_id
			INNER JOIN {$this->wpdb->prefix}sn_hr_profiles parent ON parent.id=a.parent_profile_id
			INNER JOIN {$this->wpdb->prefix}sn_hr_positions parent_pos ON parent_pos.id=parent.position_id
			WHERE a.relationship_type='reports_to' AND a.is_current=1 AND parent_pos.slug=%s AND child_pos.slug=%s",
			$parent_position_slug,
			$child_position_slug
		) );
	}

	private function log_history( ?int $assignment_id, ?int $child_profile_id, ?int $old_parent_profile_id, ?int $new_parent_profile_id, int $actor_user_id, string $action, string $reason, array $context ): void {
		$this->wpdb->insert( $this->wpdb->prefix . 'sn_hr_assignment_history', [
			'assignment_id' => $assignment_id ?: null,
			'child_profile_id' => $child_profile_id ?: null,
			'old_parent_profile_id' => $old_parent_profile_id ?: null,
			'new_parent_profile_id' => $new_parent_profile_id ?: null,
			'actor_user_id' => $actor_user_id ?: null,
			'action' => sanitize_key( $action ),
			'reason' => $reason ?: null,
			'context' => $context ? wp_json_encode( $context ) : null,
		] );
	}
}
