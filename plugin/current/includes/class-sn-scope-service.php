<?php
if ( ! defined( 'ABSPATH' ) ) { exit; }

class SN_Scope_Service {
	public const ENFORCEMENT_OPTION = 'sn_scope_enforcement_mode';
	public const ENFORCEMENT_AUDIT_OPTION = 'sn_scope_enforcement_audit_log';

	private wpdb $wpdb;
	private SN_HR_Service $hr;
	private SN_Hierarchy_Service $hierarchy;

	public function __construct() {
		global $wpdb;
		$this->wpdb = $wpdb;
		$this->hr = new SN_HR_Service();
		$this->hierarchy = new SN_Hierarchy_Service();
	}

	public function get_viewer_profile( int $viewer_user_id ) {
		return $this->hr->get_profile_by_user_id( $viewer_user_id );
	}

	public function get_viewer_position_slug( int $viewer_user_id ): string {
		$profile = $this->get_viewer_profile( $viewer_user_id );
		if ( ! $profile || ! $profile->position_id ) { return $this->legacy_position_slug( $viewer_user_id ); }
		$slug = $this->wpdb->get_var( $this->wpdb->prepare( "SELECT slug FROM {$this->wpdb->prefix}sn_hr_positions WHERE id=%d", (int) $profile->position_id ) );
		return $slug ? (string) $slug : $this->legacy_position_slug( $viewer_user_id );
	}

	public function get_enforcement_mode(): string {
		$mode = (string) get_option( self::ENFORCEMENT_OPTION, 'off' );
		$allowed = [ 'off', 'audit_only', 'enforce_seller', 'enforce_seller_supervisor', 'enforce_sales_management' ];
		return in_array( $mode, $allowed, true ) ? $mode : 'off';
	}

	public function should_enforce( string $context, int $viewer_user_id ): bool {
		$mode = $this->get_enforcement_mode();
		$seller_contexts = [ 'seller_leads', 'seller_invoices', 'seller_invoice_search' ];
		$supervisor_contexts = [ 'supervisor_leads', 'supervisor_invoices', 'supervisor_invoice_search', 'supervisor_sellers', 'supervisor_reports_readonly' ];
		$sales_manager_contexts = [ 'sales_manager_leads', 'sales_manager_invoices', 'sales_manager_invoice_search', 'sales_manager_sellers', 'sales_manager_reports_readonly' ];
		$sales_deputy_contexts = [ 'sales_deputy_leads', 'sales_deputy_invoices', 'sales_deputy_invoice_search', 'sales_deputy_sellers', 'sales_deputy_reports_readonly' ];
		if ( $mode === 'enforce_seller' ) {
			return in_array( $context, $seller_contexts, true ) && $this->can_enforce_seller_scope( $viewer_user_id );
		}
		if ( $mode === 'enforce_seller_supervisor' ) {
			if ( in_array( $context, $seller_contexts, true ) ) {
				return $this->can_enforce_seller_scope( $viewer_user_id );
			}
			return in_array( $context, $supervisor_contexts, true ) && $this->can_enforce_supervisor_scope( $viewer_user_id );
		}
		if ( $mode === 'enforce_sales_management' ) {
			if ( in_array( $context, $seller_contexts, true ) ) {
				return $this->can_enforce_seller_scope( $viewer_user_id );
			}
			if ( in_array( $context, $supervisor_contexts, true ) ) {
				return $this->can_enforce_supervisor_scope( $viewer_user_id );
			}
			if ( in_array( $context, $sales_manager_contexts, true ) ) {
				return $this->can_enforce_sales_manager_scope( $viewer_user_id );
			}
			return in_array( $context, $sales_deputy_contexts, true ) && $this->can_enforce_sales_deputy_scope( $viewer_user_id );
		}
		return false;
	}

	public function can_enforce_supervisor_scope( int $viewer_user_id ): bool {
		if ( $viewer_user_id < 1 || user_can( $viewer_user_id, 'manage_options' ) || user_can( $viewer_user_id, 'sn_view_finance' ) ) {
			return false;
		}
		$user = get_user_by( 'id', $viewer_user_id );
		if ( ! $user instanceof WP_User ) {
			return false;
		}
		$position = $this->get_viewer_position_slug( $viewer_user_id );
		return in_array( $position, [ 'supervisor', 'senior_supervisor' ], true );
	}

	public function can_enforce_sales_manager_scope( int $viewer_user_id ): bool {
		if ( $viewer_user_id < 1 || user_can( $viewer_user_id, 'manage_options' ) || user_can( $viewer_user_id, 'sn_view_finance' ) ) {
			return false;
		}
		$user = get_user_by( 'id', $viewer_user_id );
		if ( ! $user instanceof WP_User ) {
			return false;
		}
		return $this->get_viewer_position_slug( $viewer_user_id ) === 'sales_manager';
	}

	public function can_enforce_sales_deputy_scope( int $viewer_user_id ): bool {
		if ( $viewer_user_id < 1 || user_can( $viewer_user_id, 'manage_options' ) || user_can( $viewer_user_id, 'sn_view_finance' ) ) {
			return false;
		}
		$user = get_user_by( 'id', $viewer_user_id );
		if ( ! $user instanceof WP_User ) {
			return false;
		}
		return $this->get_viewer_position_slug( $viewer_user_id ) === 'sales_deputy';
	}

	public function get_visible_user_ids_for_enforcement( int $viewer_user_id ): array {
		if ( $this->can_enforce_seller_scope( $viewer_user_id ) ) {
			return [ $viewer_user_id ];
		}
		return $this->visible_user_ids( $viewer_user_id );
	}

	public function apply_seller_scope_to_where( $where, int $viewer_user_id, string $seller_column = 'seller_id' ) {
		if ( ! in_array( $this->get_enforcement_mode(), [ 'enforce_seller', 'enforce_seller_supervisor', 'enforce_sales_management' ], true ) || ! $this->can_enforce_seller_scope( $viewer_user_id ) ) {
			return $where;
		}
		$seller_column = preg_replace( '/[^A-Za-z0-9_.]/', '', $seller_column );
		$condition = $seller_column . '=%d';
		if ( is_array( $where ) ) {
			$where[] = $condition;
			return $where;
		}
		$where = trim( (string) $where );
		return $where === '' ? $condition : '(' . $where . ') AND ' . $condition;
	}

	public function can_enforce_seller_scope( int $viewer_user_id ): bool {
		if ( $viewer_user_id < 1 || user_can( $viewer_user_id, 'manage_options' ) || user_can( $viewer_user_id, 'sn_view_finance' ) ) {
			return false;
		}
		$user = get_user_by( 'id', $viewer_user_id );
		if ( ! $user instanceof WP_User ) {
			return false;
		}
		$position = $this->get_viewer_position_slug( $viewer_user_id );
		return $position === 'seller' || in_array( 'sn_seller', (array) $user->roles, true );
	}

	public function scope_enforcement_decision( string $context, int $viewer_user_id ): array {
		$mode = $this->get_enforcement_mode();
		$seller_contexts = [ 'seller_leads', 'seller_invoices', 'seller_invoice_search' ];
		$supervisor_contexts = [ 'supervisor_leads', 'supervisor_invoices', 'supervisor_invoice_search', 'supervisor_sellers', 'supervisor_reports_readonly' ];
		$sales_manager_contexts = [ 'sales_manager_leads', 'sales_manager_invoices', 'sales_manager_invoice_search', 'sales_manager_sellers', 'sales_manager_reports_readonly' ];
		$sales_deputy_contexts = [ 'sales_deputy_leads', 'sales_deputy_invoices', 'sales_deputy_invoice_search', 'sales_deputy_sellers', 'sales_deputy_reports_readonly' ];
		$context_supported = in_array( $context, array_merge( $seller_contexts, $supervisor_contexts, $sales_manager_contexts, $sales_deputy_contexts ), true );
		$is_seller_context = in_array( $context, $seller_contexts, true );
		$is_supervisor_context = in_array( $context, $supervisor_contexts, true );
		$is_sales_manager_context = in_array( $context, $sales_manager_contexts, true );
		$is_sales_deputy_context = in_array( $context, $sales_deputy_contexts, true );
		$seller_allowed_in_mode = in_array( $mode, [ 'audit_only', 'enforce_seller', 'enforce_seller_supervisor', 'enforce_sales_management' ], true );
		$supervisor_allowed_in_mode = in_array( $mode, [ 'audit_only', 'enforce_seller_supervisor', 'enforce_sales_management' ], true );
		$sales_manager_allowed_in_mode = in_array( $mode, [ 'audit_only', 'enforce_sales_management' ], true );
		$sales_deputy_allowed_in_mode = in_array( $mode, [ 'audit_only', 'enforce_sales_management' ], true );
		$can_enforce = ( $is_seller_context && $seller_allowed_in_mode && $this->can_enforce_seller_scope( $viewer_user_id ) )
			|| ( $is_supervisor_context && $supervisor_allowed_in_mode && $this->can_enforce_supervisor_scope( $viewer_user_id ) )
			|| ( $is_sales_manager_context && $sales_manager_allowed_in_mode && $this->can_enforce_sales_manager_scope( $viewer_user_id ) )
			|| ( $is_sales_deputy_context && $sales_deputy_allowed_in_mode && $this->can_enforce_sales_deputy_scope( $viewer_user_id ) );
		$enforced = $this->should_enforce( $context, $viewer_user_id );
		if ( ! $context_supported ) {
			$reason = 'unsupported_context';
		} elseif ( $mode === 'off' ) {
			$reason = 'mode_off';
		} elseif ( $mode === 'audit_only' ) {
			$reason = $can_enforce ? 'audit_only_would_enforce' : 'audit_only_not_applicable';
		} elseif ( $mode === 'enforce_seller' ) {
			$reason = $enforced ? 'enforce_seller' : ( $is_supervisor_context ? 'supervisor_not_enabled_in_mode' : 'not_seller_or_privileged' );
		} elseif ( $mode === 'enforce_seller_supervisor' ) {
			$reason = $enforced ? 'enforce_seller_supervisor' : ( ( $is_sales_manager_context || $is_sales_deputy_context ) ? 'sales_management_not_enabled_in_mode' : 'not_scoped_role_or_privileged' );
		} elseif ( $mode === 'enforce_sales_management' ) {
			$reason = $enforced ? 'enforce_sales_management' : 'not_scoped_role_or_privileged';
		} else {
			$reason = 'mode_not_implemented';
		}
		if ( $enforced && ! $this->get_viewer_profile( $viewer_user_id ) && ( $is_sales_manager_context || $is_sales_deputy_context ) ) {
			$reason = 'enforce_sales_management_legacy_fallback';
		}
		$visible_seller_ids = $can_enforce ? $this->scope_visible_seller_ids( $viewer_user_id ) : [];
		$visible_user_ids = $can_enforce ? $this->visible_user_ids( $viewer_user_id ) : [];
		return [
			'context' => $context,
			'viewer_user_id' => $viewer_user_id,
			'viewer_position' => $this->get_viewer_position_slug( $viewer_user_id ),
			'mode' => $mode,
			'enforced' => $enforced,
			'would_enforce' => $can_enforce,
			'reason' => $reason,
			'visible_user_ids' => $enforced ? ( $is_seller_context ? [ $viewer_user_id ] : $visible_seller_ids ) : [],
			'visible_user_count' => count( $visible_user_ids ),
			'visible_seller_count' => count( $visible_seller_ids ),
		];
	}

	public function seller_scope_decision( string $context, int $viewer_user_id ): array {
		return $this->scope_enforcement_decision( $context, $viewer_user_id );
	}

	public function record_enforcement_decision( string $context, int $viewer_user_id, array $decision = [], array $counts = [] ): void {
		$decision = $decision ?: $this->seller_scope_decision( $context, $viewer_user_id );
		$entry = [
			'time' => current_time( 'mysql' ),
			'context' => sanitize_key( $context ),
			'viewer_user_id' => (int) $viewer_user_id,
			'mode' => (string) ( $decision['mode'] ?? $this->get_enforcement_mode() ),
			'viewer_position' => sanitize_key( (string) ( $decision['viewer_position'] ?? $this->get_viewer_position_slug( $viewer_user_id ) ) ),
			'enforced' => ! empty( $decision['enforced'] ),
			'would_enforce' => ! empty( $decision['would_enforce'] ),
			'scope_visible_user_count' => (int) ( $counts['scope_visible_user_count'] ?? $counts['scope_count'] ?? $decision['visible_user_count'] ?? 0 ),
			'scope_visible_seller_count' => (int) ( $counts['scope_visible_seller_count'] ?? $counts['scope_count'] ?? $decision['visible_seller_count'] ?? 0 ),
			'legacy_visible_user_count' => (int) ( $counts['legacy_visible_user_count'] ?? $counts['legacy_count'] ?? 0 ),
			'legacy_visible_seller_count' => (int) ( $counts['legacy_visible_seller_count'] ?? $counts['legacy_count'] ?? 0 ),
			'reason' => sanitize_key( (string) ( $decision['reason'] ?? '' ) ),
		];
		foreach ( [ 'scope_active_seller_count', 'scope_inactive_seller_count' ] as $key ) {
			if ( isset( $counts[ $key ] ) ) {
				$entry[ $key ] = (int) $counts[ $key ];
			}
		}
		$log = get_option( self::ENFORCEMENT_AUDIT_OPTION, [] );
		if ( ! is_array( $log ) ) {
			$log = [];
		}
		array_unshift( $log, $entry );
		update_option( self::ENFORCEMENT_AUDIT_OPTION, array_slice( $log, 0, 20 ), false );
	}

	public function visible_profile_ids( int $viewer_user_id ): array {
		if ( user_can( $viewer_user_id, 'manage_options' ) ) {
			return array_map( 'intval', $this->wpdb->get_col( "SELECT id FROM {$this->wpdb->prefix}sn_hr_profiles WHERE is_active=1" ) );
		}
		$profile = $this->get_viewer_profile( $viewer_user_id );
		if ( ! $profile ) { return []; }
		$position = $this->get_viewer_position_slug( $viewer_user_id );
		if ( in_array( $position, [ 'sales_deputy' ], true ) ) {
			return array_values( array_unique( array_merge( [ (int) $profile->id ], $this->hierarchy->get_descendant_profile_ids( (int) $profile->id, 10 ) ) ) );
		}
		if ( in_array( $position, [ 'sales_manager', 'senior_supervisor', 'supervisor' ], true ) ) {
			return array_values( array_unique( array_merge( [ (int) $profile->id ], $this->hierarchy->get_descendant_profile_ids( (int) $profile->id, 10 ) ) ) );
		}
		if ( $position === 'seller' ) {
			return [ (int) $profile->id ];
		}
		return [ (int) $profile->id ];
	}

	public function visible_user_ids( int $viewer_user_id ): array {
		$profile_ids = $this->visible_profile_ids( $viewer_user_id );
		if ( ! $profile_ids ) {
			return $this->legacy_visible_user_ids( $viewer_user_id );
		}
		$ph = implode( ',', array_fill( 0, count( $profile_ids ), '%d' ) );
		return array_map( 'intval', $this->wpdb->get_col( $this->wpdb->prepare( "SELECT user_id FROM {$this->wpdb->prefix}sn_hr_profiles WHERE id IN ({$ph})", ...$profile_ids ) ) );
	}

	public function can_view_user( int $viewer_user_id, int $target_user_id ): bool {
		if ( user_can( $viewer_user_id, 'manage_options' ) ) { return true; }
		return in_array( $target_user_id, $this->visible_user_ids( $viewer_user_id ), true );
	}

	public function can_view_lead( int $viewer_user_id, int $lead_id ): bool {
		if ( user_can( $viewer_user_id, 'manage_options' ) ) { return true; }
		$lead = $this->wpdb->get_row( $this->wpdb->prepare( "SELECT seller_id, supervisor_id FROM {$this->wpdb->prefix}sn_leads WHERE id=%d", $lead_id ) );
		if ( ! $lead ) { return false; }
		$visible = $this->visible_user_ids( $viewer_user_id );
		return in_array( (int) $lead->seller_id, $visible, true ) || in_array( (int) $lead->supervisor_id, $visible, true );
	}

	public function can_view_invoice( int $viewer_user_id, int $invoice_id ): bool {
		if ( user_can( $viewer_user_id, 'manage_options' ) ) { return true; }
		$invoice = $this->wpdb->get_row( $this->wpdb->prepare( "SELECT seller_id, lead_id FROM {$this->wpdb->prefix}sn_invoices WHERE id=%d", $invoice_id ) );
		if ( ! $invoice ) { return false; }
		if ( in_array( (int) $invoice->seller_id, $this->visible_user_ids( $viewer_user_id ), true ) ) { return true; }
		return $invoice->lead_id ? $this->can_view_lead( $viewer_user_id, (int) $invoice->lead_id ) : false;
	}

	public function can_assign_lead_to_user( int $viewer_user_id, int $target_user_id ): bool {
		if ( user_can( $viewer_user_id, 'manage_options' ) ) { return true; }
		$position = $this->get_viewer_position_slug( $viewer_user_id );
		if ( ! in_array( $position, [ 'sales_deputy', 'sales_manager', 'senior_supervisor', 'supervisor' ], true ) ) { return false; }
		return $this->can_view_user( $viewer_user_id, $target_user_id );
	}

	public function can_view_wallet( int $viewer_user_id, int $target_user_id ): bool {
		if ( user_can( $viewer_user_id, 'manage_options' ) ) { return true; }
		return $viewer_user_id === $target_user_id || $this->can_view_user( $viewer_user_id, $target_user_id );
	}

	public function explain_scope( int $viewer_user_id ): array {
		$user = get_user_by( 'id', $viewer_user_id );
		$profile = $this->get_viewer_profile( $viewer_user_id );
		$position = $this->get_viewer_position_slug( $viewer_user_id );
		$visible_profiles = $this->visible_profile_ids( $viewer_user_id );
		$visible_users = $this->visible_user_ids( $viewer_user_id );
		$seller_status_counts = $this->visible_seller_status_counts( $viewer_user_id );
		$parent_user_id = $profile ? $this->hierarchy->get_current_parent_user_id( $viewer_user_id ) : 0;
		$direct_children = $profile ? $this->hierarchy->get_direct_children_profile_ids( (int) $profile->id ) : [];
		$descendants = $profile ? $this->hierarchy->get_descendant_profile_ids( (int) $profile->id, 10 ) : [];
		$fallback_used = ! $profile || ! $visible_profiles;
		$rules = [];
		if ( user_can( $viewer_user_id, 'manage_options' ) ) {
			$rules[] = 'administrator/manage_options: all active HR profiles are visible';
		} elseif ( in_array( $position, [ 'sales_deputy' ], true ) ) {
			$rules[] = 'sales_deputy: all sales hierarchy descendants';
		} elseif ( in_array( $position, [ 'sales_manager', 'senior_supervisor', 'supervisor' ], true ) ) {
			$rules[] = "{$position}: self profile plus HR descendants";
		} elseif ( $position === 'seller' ) {
			$rules[] = 'seller: self only';
		} elseif ( $fallback_used ) {
			$rules[] = 'fallback: legacy WordPress role/user_meta visibility';
		} else {
			$rules[] = "{$position}: self profile only";
		}
		return [
			'viewer_user_id' => $viewer_user_id,
			'has_hr_profile' => (bool) $profile,
			'profile_id' => $profile ? (int) $profile->id : 0,
			'position_slug' => $position,
			'legacy_role' => $profile ? (string) ( $profile->legacy_role ?? '' ) : $this->legacy_position_slug( $viewer_user_id ),
			'wp_roles' => $user instanceof WP_User ? array_values( (array) $user->roles ) : [],
			'direct_parent_user_id' => $parent_user_id,
			'direct_children_count' => count( $direct_children ),
			'descendant_profile_count' => count( $descendants ),
			'visible_user_count' => count( $visible_users ),
			'visible_profile_count' => count( $visible_profiles ),
			'visible_seller_count' => (int) $seller_status_counts['visible_seller_count'],
			'active_seller_count' => (int) $seller_status_counts['active_seller_count'],
			'inactive_seller_count' => (int) $seller_status_counts['inactive_seller_count'],
			'fallback_used' => $fallback_used,
			'fallback_reason' => $fallback_used ? ( ! $profile ? 'missing_hr_profile' : 'empty_visible_profile_ids' ) : '',
			'has_hierarchy_assignment' => $parent_user_id > 0,
			'rules_applied' => $rules,
			'target_hierarchy_warnings' => $this->target_hierarchy_warnings( $viewer_user_id, $profile, $position ),
		];
	}

	public function legacy_visible_seller_ids( int $viewer_user_id ): array {
		if ( user_can( $viewer_user_id, 'manage_options' ) ) {
			return array_map( 'intval', get_users( [ 'role' => 'sn_seller', 'fields' => 'ID' ] ) );
		}
		$position = $this->legacy_position_slug( $viewer_user_id );
		if ( $position === 'seller' ) { return [ $viewer_user_id ]; }
		if ( $position === 'supervisor' ) {
			return array_map( 'intval', get_users( [ 'role' => 'sn_seller', 'fields' => 'ID', 'meta_key' => 'sn_supervisor_id', 'meta_value' => $viewer_user_id ] ) );
		}
		if ( $position === 'sales_manager' ) {
			return array_map( 'intval', get_users( [ 'role' => 'sn_seller', 'fields' => 'ID' ] ) );
		}
		return [];
	}

	public function scope_visible_seller_ids( int $viewer_user_id ): array {
		$profile_ids = $this->visible_profile_ids( $viewer_user_id );
		if ( $profile_ids ) {
			$profile_ids = array_values( array_unique( array_map( 'intval', $profile_ids ) ) );
			$ph = implode( ',', array_fill( 0, count( $profile_ids ), '%d' ) );
			return array_map( 'intval', $this->wpdb->get_col( $this->wpdb->prepare(
				"SELECT p.user_id
				FROM {$this->wpdb->prefix}sn_hr_profiles p
				INNER JOIN {$this->wpdb->prefix}sn_hr_positions pos ON pos.id=p.position_id
				WHERE p.id IN ({$ph}) AND pos.slug='seller'",
				...$profile_ids
			) ) );
		}
		$visible = $this->visible_user_ids( $viewer_user_id );
		if ( ! $visible ) { return []; }
		$sellers = [];
		foreach ( $visible as $user_id ) {
			$user = get_user_by( 'id', (int) $user_id );
			if ( $user instanceof WP_User && in_array( 'sn_seller', (array) $user->roles, true ) ) {
				$sellers[] = (int) $user_id;
			}
		}
		return array_values( array_unique( $sellers ) );
	}

	public function compare_visible_sellers( int $viewer_user_id ): array {
		$legacy = $this->legacy_visible_seller_ids( $viewer_user_id );
		$scope = $this->scope_visible_seller_ids( $viewer_user_id );
		$matched = array_values( array_intersect( $legacy, $scope ) );
		$only_legacy = array_values( array_diff( $legacy, $scope ) );
		$only_scope = array_values( array_diff( $scope, $legacy ) );
		$position = $this->get_viewer_position_slug( $viewer_user_id );
		return [
			'legacy_count' => count( $legacy ),
			'scope_count' => count( $scope ),
			'matched_count' => count( $matched ),
			'only_in_legacy_count' => count( $only_legacy ),
			'only_in_scope_count' => count( $only_scope ),
			'legacy_ids' => $legacy,
			'scope_ids' => $scope,
			'only_in_legacy_ids' => $only_legacy,
			'only_in_scope_ids' => $only_scope,
			'informational_note' => $position === 'sales_deputy' ? 'sales_deputy is new; legacy comparison is informational only.' : '',
		];
	}

	public function visible_seller_status_counts( int $viewer_user_id ): array {
		$seller_ids = $this->scope_visible_seller_ids( $viewer_user_id );
		$active = 0;
		$inactive = 0;
		foreach ( $seller_ids as $seller_id ) {
			$is_active = get_user_meta( (int) $seller_id, 'sn_is_active', true );
			if ( $is_active === '0' ) {
				$inactive++;
			} else {
				$active++;
			}
		}
		return [
			'visible_seller_count' => count( $seller_ids ),
			'active_seller_count' => $active,
			'inactive_seller_count' => $inactive,
		];
	}

	public function enforcement_count_summary( int $viewer_user_id, array $legacy_seller_ids = [], ?array $scope_seller_ids = null ): array {
		$legacy_seller_ids = $legacy_seller_ids ? array_values( array_unique( array_map( 'intval', $legacy_seller_ids ) ) ) : $this->legacy_visible_seller_ids( $viewer_user_id );
		$scope_seller_ids = is_array( $scope_seller_ids ) ? array_values( array_unique( array_map( 'intval', $scope_seller_ids ) ) ) : $this->scope_visible_seller_ids( $viewer_user_id );
		$position = $this->get_viewer_position_slug( $viewer_user_id );
		$legacy_visible_user_count = count( $legacy_seller_ids );
		if ( $position !== 'seller' && ! user_can( $viewer_user_id, 'manage_options' ) ) {
			$legacy_visible_user_count++;
		}
		$status_counts = $this->visible_seller_status_counts( $viewer_user_id );
		return [
			'legacy_visible_user_count' => $legacy_visible_user_count,
			'legacy_visible_seller_count' => count( $legacy_seller_ids ),
			'scope_visible_user_count' => count( $this->visible_user_ids( $viewer_user_id ) ),
			'scope_visible_seller_count' => count( $scope_seller_ids ),
			'scope_active_seller_count' => (int) $status_counts['active_seller_count'],
			'scope_inactive_seller_count' => (int) $status_counts['inactive_seller_count'],
		];
	}

	public function compare_lead_invoice_counts( int $viewer_user_id ): array {
		$legacy_sellers = $this->legacy_visible_seller_ids( $viewer_user_id );
		$scope_sellers = $this->scope_visible_seller_ids( $viewer_user_id );
		return [
			'leads' => [
				'legacy_count' => $this->count_records_for_sellers( 'sn_leads', $legacy_sellers ),
				'scope_count' => $this->count_records_for_sellers( 'sn_leads', $scope_sellers ),
			],
			'invoices' => [
				'legacy_count' => $this->count_records_for_sellers( 'sn_invoices', $legacy_sellers ),
				'scope_count' => $this->count_records_for_sellers( 'sn_invoices', $scope_sellers ),
			],
		];
	}

	public function describe_seller_visibility_gap( int $viewer_user_id, array $seller_ids, int $limit = 100 ): array {
		$seller_ids = array_values( array_unique( array_map( 'intval', $seller_ids ) ) );
		$visible_user_ids = $this->visible_user_ids( $viewer_user_id );
		$rows = [];
		$summary = [
			'missing_due_to_no_assignment' => 0,
			'missing_due_to_supervisor_not_under_viewer' => 0,
			'missing_due_to_no_hr_profile' => 0,
		];
		foreach ( $seller_ids as $seller_id ) {
			$row = $this->describe_seller_visibility_gap_row( $viewer_user_id, $seller_id, $visible_user_ids );
			if ( $row['suggested_reason'] === 'no_hr_assignment' ) {
				$summary['missing_due_to_no_assignment']++;
			} elseif ( $row['suggested_reason'] === 'legacy_supervisor_not_under_viewer' ) {
				$summary['missing_due_to_supervisor_not_under_viewer']++;
			} elseif ( $row['suggested_reason'] === 'no_hr_profile' ) {
				$summary['missing_due_to_no_hr_profile']++;
			}
			if ( count( $rows ) < max( 1, min( 100, $limit ) ) ) {
				$rows[] = $row;
			}
		}
		return [
			'total_count' => count( $seller_ids ),
			'displayed_count' => count( $rows ),
			'summary' => $summary,
			'rows' => $rows,
		];
	}

	public function target_hierarchy_warnings( int $viewer_user_id, $profile = null, string $position = '' ): array {
		$warnings = [];
		$legacy_supervisors_still_supervisor = $this->count_legacy_supervisors_still_supervisor();
		if ( $legacy_supervisors_still_supervisor > 0 ) {
			$warnings[] = [
				'code' => 'legacy_supervisors_still_supervisor',
				'count' => $legacy_supervisors_still_supervisor,
				'message' => 'هنوز سرپرست‌های قدیمی با سمت supervisor وجود دارند. قبل از ساختار هدف باید به senior_supervisor تبدیل شوند.',
			];
		}
		if ( ! $profile ) {
			return $warnings;
		}
		$visible_profiles = $this->visible_profile_ids( $viewer_user_id );
		if ( $position === 'sales_manager' ) {
			$senior_descendants = $this->count_visible_profiles_by_position( $visible_profiles, 'senior_supervisor' );
			$parent_slug = $this->parent_position_slug( (int) $profile->id );
			if ( $senior_descendants > 0 && $parent_slug !== 'sales_deputy' ) {
				$warnings[] = [
					'code' => 'sales_manager_missing_sales_deputy_parent',
					'count' => $senior_descendants,
					'message' => 'مدیر فروش هنوز زیر معاون فروش قرار نگرفته است.',
				];
			}
		}
		$senior_to_seller_direct = $this->count_current_relationship_visible( 'senior_supervisor', 'seller', $visible_profiles );
		if ( $senior_to_seller_direct > 0 ) {
			$warnings[] = [
				'code' => 'senior_supervisor_direct_sellers',
				'count' => $senior_to_seller_direct,
				'message' => 'برخی فروشنده‌ها مستقیم زیر سرپرست ارشد هستند. این حالت موقتاً مجاز است تا سرپرست‌های جدید تعریف شوند.',
			];
		}
		$sales_manager_to_seller_direct = $this->count_current_relationship_visible( 'sales_manager', 'seller', $visible_profiles );
		if ( $sales_manager_to_seller_direct > 0 ) {
			$warnings[] = [
				'code' => 'sales_manager_direct_sellers',
				'count' => $sales_manager_to_seller_direct,
				'message' => 'برخی فروشنده‌ها مستقیم زیر مدیر فروش هستند. این حالت فقط برای گذار از legacy مجاز است.',
			];
		}
		return $warnings;
	}

	private function legacy_position_slug( int $user_id ): string {
		$user = get_user_by( 'id', $user_id );
		if ( ! $user instanceof WP_User ) { return ''; }
		$roles = (array) $user->roles;
		if ( in_array( 'sn_hr', $roles, true ) ) { return 'hr'; }
		if ( in_array( 'sn_mis', $roles, true ) ) { return 'mis'; }
		if ( in_array( 'sn_sales_deputy', $roles, true ) ) { return 'sales_deputy'; }
		if ( in_array( 'sn_sales_manager', $roles, true ) || in_array( 'sas_sales_manager', $roles, true ) ) { return 'sales_manager'; }
		if ( in_array( 'sn_senior_supervisor', $roles, true ) || in_array( 'sas_senior_supervisor', $roles, true ) ) { return 'senior_supervisor'; }
		if ( in_array( 'sn_supervisor', $roles, true ) || in_array( 'sas_supervisor', $roles, true ) ) { return 'supervisor'; }
		if ( in_array( 'sn_converter', $roles, true ) ) { return 'converter'; }
		if ( in_array( 'sn_financial', $roles, true ) || in_array( 'sn_financial_approval', $roles, true ) || in_array( 'sn_finance', $roles, true ) ) { return 'finance'; }
		if ( in_array( 'sn_after_sales', $roles, true ) ) { return 'after_sales'; }
		if ( in_array( 'sn_project_manager', $roles, true ) ) { return 'project_manager'; }
		if ( in_array( 'sn_project_expert', $roles, true ) ) { return 'project_expert'; }
		if ( in_array( 'sn_shipping_expert', $roles, true ) ) { return 'shipping_expert'; }
		if ( in_array( 'sn_operations_sales_manager', $roles, true ) ) { return 'operations_sales_manager'; }
		if ( in_array( 'sn_operations_sales_supervisor', $roles, true ) ) { return 'operations_sales_supervisor'; }
		if ( in_array( 'sn_operations_sales_expert', $roles, true ) ) { return 'operations_sales_expert'; }
		if ( in_array( 'sn_operations_executive_manager', $roles, true ) ) { return 'operations_executive_manager'; }
		if ( in_array( 'sn_operations_execution_expert', $roles, true ) ) { return 'operations_execution_expert'; }
		if ( in_array( 'sn_seller', $roles, true ) || in_array( 'sas_employee', $roles, true ) ) { return 'seller'; }
		return '';
	}

	private function legacy_visible_user_ids( int $viewer_user_id ): array {
		if ( user_can( $viewer_user_id, 'manage_options' ) ) {
			return array_map( 'intval', get_users( [ 'fields' => 'ID' ] ) );
		}
		$position = $this->legacy_position_slug( $viewer_user_id );
		if ( $position === 'seller' ) { return [ $viewer_user_id ]; }
		if ( $position === 'supervisor' ) {
			$ids = array_map( 'intval', get_users( [ 'fields' => 'ID', 'meta_key' => 'sn_supervisor_id', 'meta_value' => $viewer_user_id ] ) );
			$ids[] = $viewer_user_id;
			return array_values( array_unique( $ids ) );
		}
		if ( $position === 'sales_manager' ) {
			return array_map( 'intval', get_users( [ 'fields' => 'ID' ] ) );
		}
		return [ $viewer_user_id ];
	}

	private function count_records_for_sellers( string $table, array $seller_ids ): int {
		$full = $this->wpdb->prefix . $table;
		if ( user_can( get_current_user_id(), 'manage_options' ) && empty( $seller_ids ) ) {
			return 0;
		}
		if ( ! $seller_ids ) { return 0; }
		$seller_ids = array_values( array_unique( array_map( 'intval', $seller_ids ) ) );
		$ph = implode( ',', array_fill( 0, count( $seller_ids ), '%d' ) );
		return (int) $this->wpdb->get_var( $this->wpdb->prepare( "SELECT COUNT(*) FROM {$full} WHERE seller_id IN ({$ph})", ...$seller_ids ) );
	}

	private function parent_position_slug( int $child_profile_id ): string {
		$parent_profile_id = $this->hierarchy->get_current_parent_profile_id( $child_profile_id );
		if ( ! $parent_profile_id ) { return ''; }
		return (string) $this->wpdb->get_var( $this->wpdb->prepare(
			"SELECT pos.slug FROM {$this->wpdb->prefix}sn_hr_profiles p INNER JOIN {$this->wpdb->prefix}sn_hr_positions pos ON pos.id=p.position_id WHERE p.id=%d",
			$parent_profile_id
		) );
	}

	private function count_visible_profiles_by_position( array $profile_ids, string $position_slug ): int {
		$profile_ids = array_values( array_unique( array_map( 'intval', $profile_ids ) ) );
		if ( ! $profile_ids ) { return 0; }
		$ph = implode( ',', array_fill( 0, count( $profile_ids ), '%d' ) );
		return (int) $this->wpdb->get_var( $this->wpdb->prepare(
			"SELECT COUNT(*) FROM {$this->wpdb->prefix}sn_hr_profiles p INNER JOIN {$this->wpdb->prefix}sn_hr_positions pos ON pos.id=p.position_id WHERE p.id IN ({$ph}) AND pos.slug=%s",
			...array_merge( $profile_ids, [ $position_slug ] )
		) );
	}

	private function count_current_relationship_visible( string $parent_position_slug, string $child_position_slug, array $visible_profile_ids ): int {
		$visible_profile_ids = array_values( array_unique( array_map( 'intval', $visible_profile_ids ) ) );
		if ( ! $visible_profile_ids ) { return 0; }
		$ph = implode( ',', array_fill( 0, count( $visible_profile_ids ), '%d' ) );
		return (int) $this->wpdb->get_var( $this->wpdb->prepare(
			"SELECT COUNT(*) FROM {$this->wpdb->prefix}sn_hr_assignments a
			INNER JOIN {$this->wpdb->prefix}sn_hr_profiles child ON child.id=a.child_profile_id
			INNER JOIN {$this->wpdb->prefix}sn_hr_positions child_pos ON child_pos.id=child.position_id
			INNER JOIN {$this->wpdb->prefix}sn_hr_profiles parent ON parent.id=a.parent_profile_id
			INNER JOIN {$this->wpdb->prefix}sn_hr_positions parent_pos ON parent_pos.id=parent.position_id
			WHERE a.relationship_type='reports_to' AND a.is_current=1 AND child.id IN ({$ph}) AND parent_pos.slug=%s AND child_pos.slug=%s",
			...array_merge( $visible_profile_ids, [ $parent_position_slug, $child_position_slug ] )
		) );
	}

	private function count_legacy_supervisors_still_supervisor(): int {
		$count = (int) $this->wpdb->get_var(
			"SELECT COUNT(*) FROM {$this->wpdb->prefix}sn_hr_profiles p
			INNER JOIN {$this->wpdb->prefix}sn_hr_positions pos ON pos.id=p.position_id
			WHERE p.legacy_role='sn_supervisor' AND pos.slug='supervisor'"
		);
		$wp_supervisor_ids = array_map( 'intval', get_users( [ 'role' => 'sn_supervisor', 'fields' => 'ID', 'number' => -1 ] ) );
		if ( $wp_supervisor_ids ) {
			$ph = implode( ',', array_fill( 0, count( $wp_supervisor_ids ), '%d' ) );
			$count = max( $count, (int) $this->wpdb->get_var( $this->wpdb->prepare(
				"SELECT COUNT(*) FROM {$this->wpdb->prefix}sn_hr_profiles p
				INNER JOIN {$this->wpdb->prefix}sn_hr_positions pos ON pos.id=p.position_id
				WHERE p.user_id IN ({$ph}) AND pos.slug='supervisor'",
				...$wp_supervisor_ids
			) ) );
		}
		return $count;
	}

	private function describe_seller_visibility_gap_row( int $viewer_user_id, int $seller_id, array $visible_user_ids ): array {
		$user = get_user_by( 'id', $seller_id );
		$profile = $this->hr->get_profile_by_user_id( $seller_id );
		$position_slug = '';
		if ( $profile && ! empty( $profile->position_id ) ) {
			$position_slug = (string) $this->wpdb->get_var( $this->wpdb->prepare(
				"SELECT slug FROM {$this->wpdb->prefix}sn_hr_positions WHERE id=%d",
				(int) $profile->position_id
			) );
		}
		$current_manager_user_id = $profile ? $this->hierarchy->get_current_parent_user_id( $seller_id ) : 0;
		$current_manager = $current_manager_user_id ? get_user_by( 'id', $current_manager_user_id ) : null;
		$legacy_supervisor_id = (int) get_user_meta( $seller_id, 'sn_supervisor_id', true );
		$legacy_supervisor = $legacy_supervisor_id ? get_user_by( 'id', $legacy_supervisor_id ) : null;
		$legacy_supervisor_profile = $legacy_supervisor_id ? $this->hr->get_profile_by_user_id( $legacy_supervisor_id ) : null;
		$suggested_reason = 'unknown';
		if ( ! $profile ) {
			$suggested_reason = 'no_hr_profile';
		} elseif ( $legacy_supervisor_id && ! $legacy_supervisor_profile ) {
			$suggested_reason = 'legacy_supervisor_missing_hr_profile';
		} elseif ( ! $current_manager_user_id ) {
			$suggested_reason = 'no_hr_assignment';
		} elseif ( $legacy_supervisor_id && ! in_array( $legacy_supervisor_id, $visible_user_ids, true ) ) {
			$suggested_reason = 'legacy_supervisor_not_under_viewer';
		}
		return [
			'user_id' => $seller_id,
			'display_name' => $user instanceof WP_User ? (string) $user->display_name : '',
			'username' => $user instanceof WP_User ? (string) $user->user_login : '',
			'wp_roles' => $user instanceof WP_User ? array_values( (array) $user->roles ) : [],
			'legacy_role' => $profile ? (string) ( $profile->legacy_role ?? '' ) : '',
			'has_hr_profile' => (bool) $profile,
			'hr_position' => $position_slug,
			'current_manager_user_id' => $current_manager_user_id,
			'current_manager_display_name' => $current_manager instanceof WP_User ? (string) $current_manager->display_name : '',
			'legacy_supervisor_id' => $legacy_supervisor_id,
			'legacy_supervisor_display_name' => $legacy_supervisor instanceof WP_User ? (string) $legacy_supervisor->display_name : '',
			'suggested_reason' => $suggested_reason,
		];
	}
}
