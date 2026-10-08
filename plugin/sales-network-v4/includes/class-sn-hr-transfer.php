<?php
if ( ! defined( 'ABSPATH' ) ) {
	exit;
}

/**
 * Import/export UI and CSV export for HR users.
 *
 * The existing people importer remains in SN_Plugin so upgrades keep the same
 * action and report format. This class owns the transport UI, scalable user
 * picker and export path to keep new concerns out of the legacy core class.
 */
final class SN_HR_Transfer {
	private static ?self $instance = null;
	private bool $hooks_registered = false;
	private bool $assets_enqueued = false;
	private array $table_exists_cache = [];

	public static function instance(): self {
		if ( null === self::$instance ) {
			self::$instance = new self();
		}
		return self::$instance;
	}

	public function register_hooks(): void {
		if ( $this->hooks_registered ) {
			return;
		}
		$this->hooks_registered = true;
		add_action( 'admin_menu', [ $this, 'register_admin_menu' ], 30 );
		add_action( 'admin_enqueue_scripts', [ $this, 'maybe_enqueue_admin_assets' ] );
		add_action( 'wp_enqueue_scripts', [ $this, 'maybe_enqueue_front_assets' ], 30 );
		add_action( 'admin_post_sn_hr_export_users', [ $this, 'handle_export' ] );
		add_action( 'wp_ajax_sn_hr_transfer_search_users', [ $this, 'ajax_search_users' ] );
	}

	public function maybe_enqueue_admin_assets(): void {
		$page = sanitize_key( wp_unslash( $_GET['page'] ?? '' ) );
		if ( 'sn-hr-transfer' === $page ) {
			$this->enqueue_assets();
		}
	}

	public function maybe_enqueue_front_assets(): void {
		if ( is_user_logged_in() && $this->is_hr_front_context() && $this->can_manage_hr() ) {
			$this->enqueue_assets();
		}
	}

	private function is_hr_front_context(): bool {
		if ( is_admin() ) {
			return false;
		}
		$hr_page_id = absint( get_option( 'sn_hr_panel_page_id', 0 ) );
		if ( $hr_page_id > 0 && is_page( $hr_page_id ) ) {
			return true;
		}
		global $post;
		if ( ! $post instanceof WP_Post ) {
			return false;
		}
		$content = (string) $post->post_content;
		return has_shortcode( $content, 'sn_hr_panel' ) || has_shortcode( $content, 'sn_my_panel' );
	}

	public function register_admin_menu(): void {
		add_submenu_page(
			'sn-dashboard',
			'ورودی/خروجی کاربران HR',
			'کاربران HR',
			'manage_options',
			'sn-hr-transfer',
			[ $this, 'render_admin_page' ]
		);
	}

	public function render_admin_page(): void {
		if ( ! current_user_can( 'manage_options' ) ) {
			wp_die( 'دسترسی غیرمجاز' );
		}
		$report = get_option( 'sn_hr_csv_dry_run_report', [] );
		echo '<div class="wrap sn-hr-transfer-admin" dir="rtl"><h1>ورودی/خروجی کاربران منابع انسانی</h1>';
		echo $this->render_module( is_array( $report ) ? $report : [], 'admin' ); // phpcs:ignore WordPress.Security.EscapeOutput.OutputNotEscaped
		echo '</div>';
	}

	public function render_module( array $report = [], string $context = 'front' ): string {
		if ( ! $this->can_manage_hr() ) {
			return '<div class="sn-notice sn-error">دسترسی به انتقال کاربران HR ندارید.</div>';
		}
		$this->enqueue_assets();
		$total_profiles = $this->profile_count();
		$instance_id = wp_unique_id( 'sn-hr-transfer-' );
		$message = sanitize_key( wp_unslash( $_GET['sn_transfer_msg'] ?? '' ) );
		ob_start();
		?>
		<div id="<?php echo esc_attr( $instance_id ); ?>" class="sn-hr-transfer" data-sn-hr-transfer data-context="<?php echo esc_attr( $context ); ?>">
			<?php if ( 'apply_confirmation_missing' === $message ) : ?>
				<div class="sn-notice sn-error">ورود انجام نشد؛ برای Apply باید تأیید تغییر اطلاعات را فعال کنید.</div>
			<?php endif; ?>
			<div class="sn-hr-transfer-intro">
				<strong><?php echo esc_html( number_format_i18n( $total_profiles ) ); ?> پروفایل HR آماده خروجی است.</strong>
				<span>فایل خروجی برای ورود دوباره با حالت «کامل / فایل خروجی HR» سازگار است. رمز، هش، توکن و نشست کاربر صادر نمی‌شود.</span>
			</div>
			<div class="sn-hr-transfer-grid">
				<section class="sn-hr-transfer-card">
					<h3>خروجی کاربران</h3>
					<p>همه کاربران HR یا افراد مشخص را به CSV استاندارد UTF-8 صادر کنید.</p>
					<form method="post" action="<?php echo esc_url( admin_url( 'admin-post.php' ) ); ?>" data-sn-export-form>
						<input type="hidden" name="action" value="sn_hr_export_users">
						<?php wp_nonce_field( 'sn_hr_export_users' ); ?>
						<div class="sn-hr-transfer-scope" role="radiogroup" aria-label="دامنه خروجی">
							<label><input type="radio" name="export_scope" value="all" checked> همه کاربران منابع انسانی</label>
							<label><input type="radio" name="export_scope" value="selected"> انتخاب کاربران</label>
						</div>
						<div class="sn-hr-transfer-picker" data-sn-picker hidden>
							<label class="sn-hr-transfer-search-label" for="<?php echo esc_attr( $instance_id ); ?>-search">جست‌وجوی نام، نام کاربری، ایمیل، موبایل یا کد پرسنلی</label>
							<div class="sn-hr-transfer-search">
								<input id="<?php echo esc_attr( $instance_id ); ?>-search" type="search" autocomplete="off" data-sn-user-search placeholder="حداقل ۲ حرف یا شناسه کاربر">
								<button class="button sn-btn sn-btn-secondary" type="button" data-sn-search-button>جست‌وجو</button>
							</div>
							<div class="sn-hr-transfer-results-head"><span data-sn-search-status>برای نمایش کاربران جست‌وجو کنید.</span><button class="sn-hr-transfer-link" type="button" data-sn-add-visible hidden>افزودن نتایج نمایش‌داده‌شده</button></div>
							<div class="sn-hr-transfer-results" data-sn-search-results aria-live="polite"></div>
							<div class="sn-hr-transfer-selected-head"><strong>انتخاب‌شده‌ها</strong><span data-sn-selected-count>۰ نفر</span></div>
							<div class="sn-hr-transfer-selected" data-sn-selected-users><span class="sn-hr-transfer-empty">هنوز کاربری انتخاب نشده است.</span></div>
							<div data-sn-selected-inputs></div>
						</div>
						<button class="button button-primary sn-btn" type="submit">دانلود CSV</button>
					</form>
				</section>

				<section class="sn-hr-transfer-card">
					<h3>ورود و بازیابی کاربران</h3>
					<p>ابتدا Dry-run را اجرا کنید؛ Apply فقط با تأیید صریح انجام می‌شود.</p>
					<div class="sn-hr-download-links">
						<a class="button sn-btn sn-btn-secondary" href="<?php echo esc_url( SN_PLUGIN_URL . 'docs/samples/sn-hr-people-advanced-sample.xlsx' ); ?>" download>نمونه کامل</a>
						<a class="button sn-btn sn-btn-secondary" href="<?php echo esc_url( SN_PLUGIN_URL . 'docs/samples/sn-hr-people-simple-sample.xlsx' ); ?>" download>نمونه ساده</a>
					</div>
					<form method="post" action="<?php echo esc_url( admin_url( 'admin-post.php' ) ); ?>" enctype="multipart/form-data" data-sn-import-form>
						<input type="hidden" name="action" value="sn_hr_people_import">
						<?php wp_nonce_field( 'sn_hr_people_import' ); ?>
						<label>نوع ورود
							<select name="import_mode">
								<option value="roundtrip_people" selected>کامل / فایل خروجی HR</option>
								<option value="advanced_people">پیشرفته / ساخت شبکه فروش</option>
								<option value="simple_people">ساده / افراد و سمت</option>
							</select>
						</label>
						<label>فایل CSV یا XLSX
							<input type="file" name="hr_csv" accept=".csv,.xlsx,text/csv,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" required>
						</label>
						<label class="sn-hr-check"><input type="checkbox" name="allow_create_wp_user" value="1"> کاربر وردپرسِ پیدا‌نشده ساخته شود</label>
						<label class="sn-hr-check"><input type="checkbox" name="send_user_notification" value="1"> برای کاربر جدید ایمیل وردپرس ارسال شود</label>
						<label class="sn-hr-check sn-hr-transfer-confirm"><input type="checkbox" name="confirm_apply" value="APPLY" data-sn-apply-confirm> تأیید می‌کنم Apply می‌تواند پروفایل، سمت، مدیر و اطلاعات جبران خدمات را تغییر دهد.</label>
						<div class="sn-actions">
							<button class="button sn-btn sn-btn-secondary" type="submit" name="run_mode" value="dry_run">Dry-run بدون اعمال تغییر</button>
							<button class="button button-primary sn-btn" type="submit" name="run_mode" value="apply">Apply تغییرات معتبر</button>
						</div>
						<p class="sn-note">حداکثر حجم فایل ۵ مگابایت و حداکثر تعداد داده ۱۰٬۰۰۰ ردیف است. فایل XLS قدیمی را ابتدا به CSV UTF-8 یا XLSX تبدیل کنید.</p>
					</form>
				</section>
			</div>

			<?php $this->render_report( $report ); ?>
		</div>
		<?php
		return (string) ob_get_clean();
	}

	private function render_report( array $report ): void {
		if ( empty( $report ) ) {
			return;
		}
		$labels = [
			'total_rows' => 'کل ردیف‌ها',
			'valid' => 'معتبر',
			'invalid' => 'نامعتبر',
			'created_users' => 'کاربر جدید',
			'matched_users' => 'کاربر موجود',
			'profiles_created' => 'پروفایل جدید',
			'profiles_updated' => 'پروفایل بروزرسانی‌شده',
			'manager_resolved' => 'مدیر پیدا شد',
			'manager_in_file' => 'مدیر داخل فایل',
			'would_assign' => 'قابل انتساب',
			'assignments_changed' => 'انتساب تغییر کرد',
			'assignments_skipped' => 'انتساب رد شد',
			'duplicate_employee_code' => 'کد پرسنلی تکراری',
			'duplicate_person' => 'کاربر تکراری',
			'identity_conflict' => 'تعارض هویت',
			'warning_count' => 'هشدارها',
			'invalid_mobile' => 'موبایل نامعتبر',
			'user_not_found' => 'کاربر پیدا نشد',
			'position_not_found' => 'سمت پیدا نشد',
			'manager_not_found' => 'مدیر پیدا نشد',
		];
		?>
		<section class="sn-hr-transfer-report">
			<h3>آخرین گزارش Import</h3>
			<div class="sn-hr-report-grid">
				<?php foreach ( $labels as $key => $label ) : ?>
					<div class="sn-hr-mini-stat"><strong><?php echo esc_html( (string) (int) ( $report[ $key ] ?? 0 ) ); ?></strong><span><?php echo esc_html( $label ); ?></span></div>
				<?php endforeach; ?>
			</div>
			<?php if ( ! empty( $report['errors'] ) ) : ?>
				<div class="sn-notice sn-error">خطاها: <?php echo esc_html( implode( '، ', array_map( 'strval', (array) $report['errors'] ) ) ); ?></div>
			<?php endif; ?>
			<?php if ( ! empty( $report['warnings'] ) ) : ?>
				<div class="sn-notice sn-warning">هشدارها: <?php echo esc_html( implode( '، ', array_slice( array_map( 'strval', (array) $report['warnings'] ), 0, 20 ) ) ); ?></div>
			<?php endif; ?>
			<?php if ( current_user_can( 'manage_options' ) ) : ?>
				<details><summary>جزئیات فنی گزارش</summary><pre><?php echo esc_html( wp_json_encode( $report, JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE ) ); ?></pre></details>
			<?php endif; ?>
		</section>
		<?php
	}

	private function enqueue_assets(): void {
		if ( $this->assets_enqueued ) {
			return;
		}
		$this->assets_enqueued = true;
		$css = SN_PLUGIN_DIR . 'assets/css/hr-transfer.css';
		$js  = SN_PLUGIN_DIR . 'assets/js/hr-transfer.js';
		wp_enqueue_style( 'sn-hr-transfer', SN_PLUGIN_URL . 'assets/css/hr-transfer.css', [], SN_VERSION . '-' . ( file_exists( $css ) ? (string) filemtime( $css ) : '0' ) );
		wp_enqueue_script( 'sn-hr-transfer', SN_PLUGIN_URL . 'assets/js/hr-transfer.js', [], SN_VERSION . '-' . ( file_exists( $js ) ? (string) filemtime( $js ) : '0' ), true );
		wp_localize_script( 'sn-hr-transfer', 'snHrTransfer', [
			'ajaxurl' => admin_url( 'admin-ajax.php' ),
			'nonce' => wp_create_nonce( 'sn_hr_transfer_search_users' ),
			'messages' => [
				'searching' => 'در حال جست‌وجو…',
				'minimumSearch' => 'برای جست‌وجوی متنی حداقل ۲ حرف وارد کنید.',
				'noResults' => 'کاربری پیدا نشد.',
				'error' => 'جست‌وجو انجام نشد؛ دوباره تلاش کنید.',
				'needSelection' => 'حداقل یک کاربر را انتخاب کنید.',
				'needApplyConfirm' => 'برای Apply باید تأیید تغییر اطلاعات را فعال کنید.',
				'person' => 'نفر',
			],
		] );
	}

	public function ajax_search_users(): void {
		if ( ! $this->can_manage_hr() ) {
			wp_send_json_error( [ 'message' => 'دسترسی غیرمجاز' ], 403 );
		}
		check_ajax_referer( 'sn_hr_transfer_search_users', 'nonce' );
		global $wpdb;
		if ( ! $this->table_exists( $wpdb->prefix . 'sn_hr_profiles' ) || ! $this->table_exists( $wpdb->prefix . 'sn_hr_positions' ) ) {
			wp_send_json_error( [ 'message' => 'جداول منابع انسانی هنوز آماده نیستند.' ], 503 );
		}
		$q = sanitize_text_field( wp_unslash( $_POST['q'] ?? '' ) );
		$query_length = function_exists( 'mb_strlen' ) ? mb_strlen( $q, 'UTF-8' ) : strlen( $q );
		if ( '' !== $q && ! ctype_digit( $q ) && $query_length < 2 ) {
			wp_send_json_success( [ 'users' => [] ] );
		}
		$where = '1=1';
		$args = [];
		if ( '' !== $q ) {
			$like = '%' . $wpdb->esc_like( $q ) . '%';
			$where = '(CAST(p.user_id AS CHAR)=%s OR COALESCE(NULLIF(p.hr_display_name,\'\'),u.display_name) LIKE %s OR u.user_login LIKE %s OR u.user_email LIKE %s OR p.employee_code LIKE %s OR p.work_phone LIKE %s)';
			array_push( $args, $q, $like, $like, $like, $like, $like );
		}
		$sql = "SELECT p.user_id, COALESCE(NULLIF(p.hr_display_name,''),u.display_name,CONCAT('#',p.user_id)) display_name, u.user_login, u.user_email, p.employee_code, pos.label position_label
			FROM {$wpdb->prefix}sn_hr_profiles p
			LEFT JOIN {$wpdb->users} u ON u.ID=p.user_id
			LEFT JOIN {$wpdb->prefix}sn_hr_positions pos ON pos.id=p.position_id
			WHERE {$where}
			ORDER BY p.is_active DESC, display_name ASC, p.user_id ASC
			LIMIT 30";
		$rows = $args ? $wpdb->get_results( $wpdb->prepare( $sql, ...$args ), ARRAY_A ) : $wpdb->get_results( $sql, ARRAY_A );
		$users = [];
		foreach ( (array) $rows as $row ) {
			$users[] = [
				'id' => (int) $row['user_id'],
				'name' => sanitize_text_field( (string) $row['display_name'] ),
				'login' => sanitize_text_field( (string) $row['user_login'] ),
				'email' => sanitize_email( (string) $row['user_email'] ),
				'employeeCode' => sanitize_text_field( (string) $row['employee_code'] ),
				'position' => sanitize_text_field( (string) $row['position_label'] ),
			];
		}
		wp_send_json_success( [ 'users' => $users ] );
	}

	public function handle_export(): void {
		if ( ! $this->can_manage_hr() ) {
			wp_die( 'دسترسی غیرمجاز' );
		}
		check_admin_referer( 'sn_hr_export_users' );
		$scope = sanitize_key( wp_unslash( $_POST['export_scope'] ?? 'all' ) );
		$selected_ids = array_values( array_unique( array_filter( array_map( 'absint', (array) ( $_POST['selected_user_ids'] ?? [] ) ) ) ) );
		if ( 'selected' === $scope && empty( $selected_ids ) ) {
			wp_die( 'حداقل یک کاربر HR را برای خروجی انتخاب کنید.' );
		}
		if ( count( $selected_ids ) > 5000 ) {
			wp_die( 'در خروجی انتخابی حداکثر ۵۰۰۰ کاربر قابل انتخاب است؛ برای تعداد بیشتر گزینه همه کاربران را بزنید.' );
		}
		if ( ! in_array( $scope, [ 'all', 'selected' ], true ) ) {
			$scope = 'all';
		}
		global $wpdb;
		foreach ( [ 'sn_hr_profiles', 'sn_hr_positions', 'sn_hr_levels', 'sn_hr_assignments' ] as $required_table ) {
			if ( ! $this->table_exists( $wpdb->prefix . $required_table ) ) {
				wp_die( 'جداول منابع انسانی کامل نیستند؛ ابتدا مهاجرت/فعال‌سازی افزونه را اجرا کنید.' );
			}
		}
		$last_profile_id = 0;
		$rows = $this->fetch_export_rows( $scope, $selected_ids, $last_profile_id, 500 );
		if ( '' !== (string) $wpdb->last_error ) {
			wp_die( 'خواندن اطلاعات خروجی انجام نشد. جزئیات در لاگ وردپرس ثبت شده است.' );
		}
		$this->send_csv_header( 'sn-hr-users-' . SN_Helpers::tehran_format( 'Y-m-d-His' ) . '.csv' );
		$output = fopen( 'php://output', 'wb' );
		if ( false === $output ) {
			wp_die( 'ساخت فایل خروجی ممکن نشد.' );
		}
		$headers = $this->export_headers();
		fputcsv( $output, $headers, ',', '"', '\\' );
		$exported = 0;
		while ( true ) {
			foreach ( $rows as $row ) {
				$last_profile_id = max( $last_profile_id, (int) $row['profile_id'] );
				$values = [];
				foreach ( $headers as $header ) {
					$values[] = $this->safe_csv_cell( (string) ( $row[ $header ] ?? '' ) );
				}
				fputcsv( $output, $values, ',', '"', '\\' );
				$exported++;
			}
			if ( count( $rows ) < 500 ) {
				break;
			}
			$rows = $this->fetch_export_rows( $scope, $selected_ids, $last_profile_id, 500 );
			if ( '' !== (string) $wpdb->last_error ) {
				error_log( 'SN HR export stopped after ' . $exported . ' rows: ' . sanitize_text_field( (string) $wpdb->last_error ) );
				break;
			}
		}
		fclose( $output );
		$this->log_export( $scope, $selected_ids, $exported );
		exit;
	}

	private function export_headers(): array {
		return [
			'export_schema', 'source_user_id', 'user_login', 'email', 'first_name', 'last_name', 'display_name', 'mobile',
			'position', 'position_label', 'level', 'level_label', 'manager_username', 'manager_name', 'manager_mobile', 'source_manager_user_id',
			'employee_code', 'employment_status', 'employment_type', 'is_active', 'hire_date', 'termination_date',
			'base_salary', 'salary_currency', 'commission_enabled', 'commission_rule_code', 'commission_rule_id', 'commission_override_mode', 'effective_from',
			'department', 'department_label', 'unit', 'unit_label', 'team', 'team_label',
			'national_id', 'work_phone', 'emergency_phone', 'notes',
		];
	}

	private function fetch_export_rows( string $scope, array $selected_ids, int $last_profile_id, int $limit ): array {
		global $wpdb;
		$profiles = $wpdb->prefix . 'sn_hr_profiles';
		$compensation = $wpdb->prefix . 'sn_hr_compensation_profiles';
		$commission_rules = $wpdb->prefix . 'sn_commission_rules';
		$departments = $wpdb->prefix . 'sn_hr_departments';
		$units = $wpdb->prefix . 'sn_hr_units';
		$teams = $wpdb->prefix . 'sn_hr_teams';
		$has_comp = $this->table_exists( $compensation );
		$has_rules = $this->table_exists( $commission_rules );
		$has_departments = $this->table_exists( $departments );
		$has_units = $this->table_exists( $units );
		$has_teams = $this->table_exists( $teams );
		$comp_select = $has_comp
			? "COALESCE(cp.base_salary,p.base_salary) base_salary, COALESCE(NULLIF(cp.salary_currency,''),p.salary_currency,'IRT') salary_currency, cp.commission_enabled, cp.default_commission_rule_id commission_rule_id, cp.commission_override_mode, cp.effective_from"
			: "p.base_salary, COALESCE(NULLIF(p.salary_currency,''),'IRT') salary_currency, '' commission_enabled, '' commission_rule_id, '' commission_override_mode, '' effective_from";
		$comp_join = $has_comp ? "LEFT JOIN {$compensation} cp ON cp.id=(SELECT MAX(cp2.id) FROM {$compensation} cp2 WHERE cp2.profile_id=p.id AND cp2.is_active=1)" : '';
		$rule_select = $has_comp && $has_rules ? 'cr.rule_code commission_rule_code' : "'' commission_rule_code";
		$rule_join = $has_comp && $has_rules ? "LEFT JOIN {$commission_rules} cr ON cr.id=cp.default_commission_rule_id" : '';
		$department_select = $has_departments ? 'd.slug department, d.label department_label' : "'' department, '' department_label";
		$unit_select = $has_units ? 'un.slug unit, un.label unit_label' : "'' unit, '' unit_label";
		$team_select = $has_teams ? 'tm.slug team, tm.label team_label' : "'' team, '' team_label";
		$department_join = $has_departments ? "LEFT JOIN {$departments} d ON d.id=p.department_id" : '';
		$unit_join = $has_units ? "LEFT JOIN {$units} un ON un.id=p.unit_id" : '';
		$team_join = $has_teams ? "LEFT JOIN {$teams} tm ON tm.id=p.team_id" : '';
		$where = [ 'p.id>%d' ];
		$args = [ $last_profile_id ];
		if ( 'selected' === $scope ) {
			$placeholders = implode( ',', array_fill( 0, count( $selected_ids ), '%d' ) );
			$where[] = "p.user_id IN ({$placeholders})";
			array_push( $args, ...$selected_ids );
		}
		$args[] = max( 1, min( 500, $limit ) );
		$sql = "SELECT
			p.id profile_id, 'sn_hr_users_v1' export_schema, p.user_id source_user_id,
			u.user_login, u.user_email email, um_first.meta_value first_name, um_last.meta_value last_name,
			COALESCE(NULLIF(p.hr_display_name,''),u.display_name,CONCAT('#',p.user_id)) display_name,
			COALESCE(NULLIF(um_billing.meta_value,''),NULLIF(um_mobile.meta_value,''),NULLIF(um_phone.meta_value,''),NULLIF(um_sn_phone.meta_value,''),p.work_phone,'') mobile,
			pos.slug position, pos.label position_label, lvl.slug level, lvl.label level_label,
			pu.user_login manager_username, COALESCE(NULLIF(parent.hr_display_name,''),pu.display_name,'') manager_name,
			COALESCE(NULLIF(pum_billing.meta_value,''),NULLIF(pum_mobile.meta_value,''),NULLIF(pum_phone.meta_value,''),NULLIF(pum_sn_phone.meta_value,''),parent.work_phone,'') manager_mobile,
			parent.user_id source_manager_user_id,
			p.employee_code, p.employment_status, p.employment_type, p.is_active, p.hire_date, p.termination_date,
			{$comp_select}, {$rule_select}, {$department_select}, {$unit_select}, {$team_select},
			p.national_id, p.work_phone, p.emergency_phone, p.notes
			FROM {$profiles} p
			LEFT JOIN {$wpdb->users} u ON u.ID=p.user_id
			LEFT JOIN {$wpdb->usermeta} um_first ON um_first.umeta_id=(SELECT MAX(mx.umeta_id) FROM {$wpdb->usermeta} mx WHERE mx.user_id=p.user_id AND mx.meta_key='first_name')
			LEFT JOIN {$wpdb->usermeta} um_last ON um_last.umeta_id=(SELECT MAX(mx.umeta_id) FROM {$wpdb->usermeta} mx WHERE mx.user_id=p.user_id AND mx.meta_key='last_name')
			LEFT JOIN {$wpdb->usermeta} um_billing ON um_billing.umeta_id=(SELECT MAX(mx.umeta_id) FROM {$wpdb->usermeta} mx WHERE mx.user_id=p.user_id AND mx.meta_key='billing_phone')
			LEFT JOIN {$wpdb->usermeta} um_mobile ON um_mobile.umeta_id=(SELECT MAX(mx.umeta_id) FROM {$wpdb->usermeta} mx WHERE mx.user_id=p.user_id AND mx.meta_key='mobile')
			LEFT JOIN {$wpdb->usermeta} um_phone ON um_phone.umeta_id=(SELECT MAX(mx.umeta_id) FROM {$wpdb->usermeta} mx WHERE mx.user_id=p.user_id AND mx.meta_key='phone')
			LEFT JOIN {$wpdb->usermeta} um_sn_phone ON um_sn_phone.umeta_id=(SELECT MAX(mx.umeta_id) FROM {$wpdb->usermeta} mx WHERE mx.user_id=p.user_id AND mx.meta_key='sn_phone')
			LEFT JOIN {$wpdb->prefix}sn_hr_positions pos ON pos.id=p.position_id
			LEFT JOIN {$wpdb->prefix}sn_hr_levels lvl ON lvl.id=p.level_id
			LEFT JOIN {$wpdb->prefix}sn_hr_assignments a ON a.id=(
				SELECT MAX(a2.id) FROM {$wpdb->prefix}sn_hr_assignments a2
				WHERE a2.child_profile_id=p.id AND a2.relationship_type='reports_to' AND a2.is_current=1
			)
			LEFT JOIN {$profiles} parent ON parent.id=a.parent_profile_id
			LEFT JOIN {$wpdb->users} pu ON pu.ID=parent.user_id
			LEFT JOIN {$wpdb->usermeta} pum_billing ON pum_billing.umeta_id=(SELECT MAX(mx.umeta_id) FROM {$wpdb->usermeta} mx WHERE mx.user_id=parent.user_id AND mx.meta_key='billing_phone')
			LEFT JOIN {$wpdb->usermeta} pum_mobile ON pum_mobile.umeta_id=(SELECT MAX(mx.umeta_id) FROM {$wpdb->usermeta} mx WHERE mx.user_id=parent.user_id AND mx.meta_key='mobile')
			LEFT JOIN {$wpdb->usermeta} pum_phone ON pum_phone.umeta_id=(SELECT MAX(mx.umeta_id) FROM {$wpdb->usermeta} mx WHERE mx.user_id=parent.user_id AND mx.meta_key='phone')
			LEFT JOIN {$wpdb->usermeta} pum_sn_phone ON pum_sn_phone.umeta_id=(SELECT MAX(mx.umeta_id) FROM {$wpdb->usermeta} mx WHERE mx.user_id=parent.user_id AND mx.meta_key='sn_phone')
			{$comp_join} {$rule_join} {$department_join} {$unit_join} {$team_join}
			WHERE " . implode( ' AND ', $where ) . '
			ORDER BY p.id ASC LIMIT %d';
		return $wpdb->get_results( $wpdb->prepare( $sql, ...$args ), ARRAY_A ) ?: [];
	}

	private function send_csv_header( string $filename ): void {
		while ( ob_get_level() > 0 ) {
			ob_end_clean();
		}
		nocache_headers();
		header( 'Content-Type: text/csv; charset=utf-8' );
		header( 'Content-Disposition: attachment; filename="' . sanitize_file_name( $filename ) . '"' );
		header( 'X-Content-Type-Options: nosniff' );
		echo "\xEF\xBB\xBF";
	}

	private function safe_csv_cell( string $value ): string {
		$value = str_replace( "\0", '', $value );
		return preg_match( '/^[\s]*[=+\-@]/u', $value ) ? "'" . $value : $value;
	}

	private function log_export( string $scope, array $selected_ids, int $count ): void {
		global $wpdb;
		$table = $wpdb->prefix . 'sn_hr_profile_logs';
		if ( ! $this->table_exists( $table ) ) {
			return;
		}
		$wpdb->insert( $table, [
			'profile_id' => null,
			'user_id' => null,
			'actor_user_id' => get_current_user_id() ?: null,
			'action' => 'hr_users_exported',
			'old_value' => null,
			'new_value' => null,
			'context' => wp_json_encode( [
				'scope' => $scope,
				'count' => $count,
				'selected_user_ids' => 'selected' === $scope ? array_slice( $selected_ids, 0, 5000 ) : [],
				'plugin_version' => defined( 'SN_VERSION' ) ? SN_VERSION : '',
			], JSON_UNESCAPED_UNICODE ),
			'created_at' => current_time( 'mysql' ),
		] );
	}

	private function profile_count(): int {
		global $wpdb;
		$table = $wpdb->prefix . 'sn_hr_profiles';
		return $this->table_exists( $table ) ? (int) $wpdb->get_var( "SELECT COUNT(*) FROM {$table}" ) : 0;
	}

	private function table_exists( string $table ): bool {
		if ( array_key_exists( $table, $this->table_exists_cache ) ) {
			return $this->table_exists_cache[ $table ];
		}
		global $wpdb;
		$this->table_exists_cache[ $table ] = (string) $wpdb->get_var( $wpdb->prepare( 'SHOW TABLES LIKE %s', $wpdb->esc_like( $table ) ) ) === $table;
		return $this->table_exists_cache[ $table ];
	}

	private function can_manage_hr(): bool {
		if ( current_user_can( 'manage_options' ) ) {
			return true;
		}
		if ( ! is_user_logged_in() ) {
			return false;
		}
		$user = wp_get_current_user();
		if ( in_array( 'sn_hr', (array) $user->roles, true ) ) {
			return true;
		}
		global $wpdb;
		$profiles = $wpdb->prefix . 'sn_hr_profiles';
		$positions = $wpdb->prefix . 'sn_hr_positions';
		if ( ! $this->table_exists( $profiles ) || ! $this->table_exists( $positions ) ) {
			return false;
		}
		return 'hr' === (string) $wpdb->get_var( $wpdb->prepare(
			"SELECT pos.slug FROM {$profiles} p INNER JOIN {$positions} pos ON pos.id=p.position_id WHERE p.user_id=%d AND p.is_active=1 LIMIT 1",
			get_current_user_id()
		) );
	}
}
