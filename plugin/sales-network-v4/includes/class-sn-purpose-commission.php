<?php
if ( ! defined( 'ABSPATH' ) ) { exit; }

/**
 * Purpose-based commission matrix.
 *
 * A payment stage is the accounting event. The invoice purpose identifies why
 * the customer paid, while the resolved HR/flow snapshot identifies who should
 * receive each configured role share. Credits are idempotent per
 * invoice/stage/purpose/role/recipient.
 */
final class SN_Purpose_Commission {
	private const DB_VERSION = '2026-09-17-purpose-commission-v1';
	private static ?self $instance = null;
	private bool $hooks_registered = false;
	private array $table_exists_cache = [];
	private array $columns_cache = [];

	public static function instance(): self {
		if ( null === self::$instance ) { self::$instance = new self(); }
		return self::$instance;
	}

	private function __construct() {}

	public static function is_matrix_engine_enabled(): bool {
		return class_exists( __CLASS__ ) && (string) get_option( 'sn_purpose_commission_engine_enabled', '1' ) === '1';
	}

	public function register_hooks(): void {
		if ( $this->hooks_registered ) { return; }
		$this->hooks_registered = true;
		add_action( 'init', [ $this, 'maybe_upgrade' ], 7 );
		add_action( 'admin_init', [ $this, 'maybe_upgrade' ], 7 );
		add_action( 'admin_menu', [ $this, 'register_admin_page' ], 25 );
		add_action( 'admin_post_sn_save_purpose_commission_matrix', [ $this, 'handle_save_matrix' ] );
		add_action( 'admin_post_sn_backfill_purpose_commissions', [ $this, 'handle_backfill' ] );
		add_action( 'sn_backfill_purpose_commissions', [ $this, 'run_backfill' ] );
		add_action( 'sn_invoice_payment_stage_approved', [ $this, 'on_payment_stage_approved' ], 18, 3 );
		add_action( 'sn_invoice_financial_rejected', [ $this, 'on_invoice_financial_rejected' ], 28, 3 );
		add_action( 'sn_invoice_financial_reopened', [ $this, 'on_invoice_financial_reopened' ], 28, 3 );
	}

	public function maybe_upgrade(): void {
		if ( (string) get_option( 'sn_purpose_commission_db_version', '' ) === self::DB_VERSION ) { return; }
		$this->install_schema();
	}

	private function table(): string {
		global $wpdb;
		return $wpdb->prefix . 'sn_payment_commission_matrix';
	}

	private function table_exists( string $table ): bool {
		global $wpdb;
		if ( ! array_key_exists( $table, $this->table_exists_cache ) ) {
			$this->table_exists_cache[ $table ] = (string) $wpdb->get_var( $wpdb->prepare( 'SHOW TABLES LIKE %s', $table ) ) === $table;
		}
		return $this->table_exists_cache[ $table ];
	}

	private function columns( string $table ): array {
		global $wpdb;
		if ( ! isset( $this->columns_cache[ $table ] ) ) {
			$this->columns_cache[ $table ] = $this->table_exists( $table ) ? (array) $wpdb->get_col( "SHOW COLUMNS FROM {$table}", 0 ) : [];
		}
		return $this->columns_cache[ $table ];
	}

	private function install_schema(): void {
		global $wpdb;
		require_once ABSPATH . 'wp-admin/includes/upgrade.php';
		$charset = $wpdb->get_charset_collate();
		$table = $this->table();
		dbDelta( "CREATE TABLE {$table} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			payment_purpose VARCHAR(60) NOT NULL,
			role_key VARCHAR(80) NOT NULL,
			rate_percent DECIMAL(7,4) NOT NULL DEFAULT 0,
			updated_by BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY purpose_role (payment_purpose, role_key),
			KEY payment_purpose (payment_purpose),
			KEY role_key (role_key)
		) {$charset};" );
		$this->table_exists_cache[ $table ] = true;

		$invoice_table = $wpdb->prefix . 'sn_invoices';
		if ( $this->table_exists( $invoice_table ) ) {
			$cols = $this->columns( $invoice_table );
			if ( ! in_array( 'payment_purpose', $cols, true ) ) {
				$wpdb->query( "ALTER TABLE {$invoice_table} ADD COLUMN payment_purpose VARCHAR(60) DEFAULT NULL AFTER payment_source" );
				unset( $this->columns_cache[ $invoice_table ] );
			}
			$idx = (array) $wpdb->get_col( "SHOW INDEX FROM {$invoice_table}", 2 );
			if ( ! in_array( 'sn_idx_invoice_payment_purpose', $idx, true ) ) {
				$wpdb->query( "ALTER TABLE {$invoice_table} ADD INDEX sn_idx_invoice_payment_purpose (payment_purpose)" );
			}
		}
		$this->ensure_wallet_tables();
		update_option( 'sn_purpose_commission_engine_enabled', '1', false );
		update_option( 'sn_purpose_commission_db_version', self::DB_VERSION, false );
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
			PRIMARY KEY (id), UNIQUE KEY user_wallet (user_id,wallet_type), KEY wallet_type (wallet_type)
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
			idempotency_key VARCHAR(191) DEFAULT NULL,
			created_by BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id), KEY wallet_id (wallet_id), KEY user_id (user_id), KEY invoice_id (invoice_id), UNIQUE KEY idempotency_key (idempotency_key), KEY type (type), KEY created_at (created_at)
		) {$charset};" );
	}

	public static function purposes(): array {
		return [
			'assessment_fee' => [ 'label' => 'اعتبارسنجی', 'cycle' => 'sales', 'description' => 'پرداخت هزینه محصول اعتبارسنجی' ],
			'subscription_direct' => [ 'label' => 'خرید مستقیم اشتراک', 'cycle' => 'sales', 'description' => 'فروش مستقیم اشتراک خارج از تبدیل دات' ],
			'conversion_purchase' => [ 'label' => 'خرید/تبدیل پس از اعتبارسنجی', 'cycle' => 'sales', 'description' => 'پرداخت گزینه انتخاب‌شده در فرآیند تبدیل' ],
			'product_sale' => [ 'label' => 'فروش محصول عادی', 'cycle' => 'sales', 'description' => 'فروش محصول معمولی شبکه فروش' ],
			'product_star_sale' => [ 'label' => 'فروش اولیه محصول*', 'cycle' => 'sales', 'description' => 'پرداخت اولیه خود کارت / Product*' ],
			'product_star_upgrade' => [ 'label' => 'افزایشی / ارتقای اعتبار محصول*', 'cycle' => 'biavin', 'description' => 'پرداخت افزایش اعتبار در چرخه بیاوین' ],
		];
	}

	public static function roles(): array {
		return [
			'seller' => [ 'label' => 'فروشنده', 'family' => 'sales' ],
			'supervisor' => [ 'label' => 'سرپرست', 'family' => 'sales' ],
			'converter' => [ 'label' => 'تبدیل‌کننده', 'family' => 'sales' ],
			'senior_supervisor' => [ 'label' => 'سرپرست ارشد', 'family' => 'sales' ],
			'sales_manager' => [ 'label' => 'مدیر فروش', 'family' => 'sales' ],
			'sales_deputy' => [ 'label' => 'معاون فروش', 'family' => 'sales' ],
			'operations_sales_expert' => [ 'label' => 'کارشناس فروش عملیات', 'family' => 'biavin' ],
			'operations_sales_supervisor' => [ 'label' => 'سرپرست فروش عملیات', 'family' => 'biavin' ],
			'operations_sales_manager' => [ 'label' => 'مدیر فروش عملیات', 'family' => 'biavin' ],
			'operations_executive_manager' => [ 'label' => 'مدیر اجرایی عملیات', 'family' => 'biavin' ],
			'operations_execution_expert' => [ 'label' => 'کارشناس اجرایی عملیات', 'family' => 'biavin' ],
		];
	}

	public function register_admin_page(): void {
		add_submenu_page( 'sn-dashboard', 'پورسانت', 'پورسانت', 'sn_view_wallet_commission', 'sn-commission-matrix', [ $this, 'render_admin_page' ] );
	}

	private function rates(): array {
		global $wpdb;
		$this->maybe_upgrade();
		$out = [];
		if ( ! $this->table_exists( $this->table() ) ) { return $out; }
		$rows = $wpdb->get_results( "SELECT payment_purpose,role_key,rate_percent FROM {$this->table()}", ARRAY_A ) ?: [];
		foreach ( $rows as $row ) {
			$out[ (string) $row['payment_purpose'] ][ (string) $row['role_key'] ] = (float) $row['rate_percent'];
		}
		return $out;
	}

	public function render_admin_page(): void {
		if ( ! current_user_can( 'sn_view_wallet_commission' ) && ! current_user_can( 'manage_options' ) ) { wp_die( 'دسترسی غیرمجاز' ); }
		$this->maybe_upgrade();
		$rates = $this->rates();
		$purposes = self::purposes();
		$roles = self::roles();
		$saved = isset( $_GET['sn_commission_saved'] );
		$backfill = get_option( 'sn_purpose_commission_backfill', [] );
		?>
		<div class="wrap sn-purpose-commission-admin" dir="rtl">
			<h1>پورسانت بر اساس قصد پرداخت</h1>
			<?php if ( $saved ) : ?><div class="notice notice-success is-dismissible"><p>جدول پورسانت ذخیره شد.</p></div><?php endif; ?>
			<?php if ( is_array( $backfill ) && $backfill ) : ?><div class="notice notice-info"><p>بازبینی سوابق پورسانت: <?php echo esc_html( (string) ( $backfill['status'] ?? '' ) ); ?> — فاکتور بررسی‌شده: <?php echo esc_html( number_format_i18n( (int) ( $backfill['scanned'] ?? 0 ) ) ); ?>، تراکنش جدید: <?php echo esc_html( number_format_i18n( (int) ( $backfill['created'] ?? 0 ) ) ); ?>، تراکنش موجود: <?php echo esc_html( number_format_i18n( (int) ( $backfill['existing'] ?? 0 ) ) ); ?></p></div><?php endif; ?>
			<div class="notice notice-info"><p><strong>مبنای محاسبه:</strong> مبلغ هر مرحله پرداخت تأییدشده × درصد نقش. مقدار ۰ یعنی آن نقش برای آن قصد پرداخت پورسانت ندارد. درصد قابل تعریف از ۰٫۰۱ تا ۱۰۰ است.</p><p><strong>کیف پول:</strong> سهم نقش‌های فروش از پرداخت‌های چرخه فروش → «فروش». هر سهم مربوط به چرخه بیاوین و همچنین سهم نقش‌های عملیاتی → «بیاوین».</p></div>
			<form method="post" action="<?php echo esc_url( admin_url( 'admin-post.php' ) ); ?>">
				<input type="hidden" name="action" value="sn_save_purpose_commission_matrix">
				<?php wp_nonce_field( 'sn_save_purpose_commission_matrix' ); ?>
				<div style="overflow:auto;background:#fff;border:1px solid #dcdcde;border-radius:12px;box-shadow:0 1px 2px rgba(0,0,0,.04)">
				<table class="widefat striped" style="min-width:1900px;border:0">
					<thead><tr><th style="position:sticky;right:0;z-index:3;background:#f6f7f7;min-width:250px">قصد پرداخت</th><?php foreach ( $roles as $role ) : ?><th style="min-width:145px;text-align:center"><?php echo esc_html( $role['label'] ); ?></th><?php endforeach; ?></tr></thead>
					<tbody>
					<?php foreach ( $purposes as $purpose_key => $purpose ) : ?>
					<tr>
						<th style="position:sticky;right:0;z-index:2;background:#fff"><strong><?php echo esc_html( $purpose['label'] ); ?></strong><br><small><?php echo esc_html( $purpose['description'] ); ?></small><br><span style="display:inline-block;margin-top:5px;padding:2px 7px;border-radius:999px;background:<?php echo $purpose['cycle'] === 'biavin' ? '#e7f3ff' : '#edf7ed'; ?>"><?php echo $purpose['cycle'] === 'biavin' ? 'چرخه بیاوین' : 'چرخه فروش'; ?></span></th>
						<?php foreach ( $roles as $role_key => $role ) : $value = (float) ( $rates[ $purpose_key ][ $role_key ] ?? 0 ); ?>
						<td style="text-align:center"><div style="display:flex;align-items:center;gap:5px;justify-content:center"><input type="number" name="rates[<?php echo esc_attr( $purpose_key ); ?>][<?php echo esc_attr( $role_key ); ?>]" min="0" max="100" step="0.01" value="<?php echo esc_attr( $value > 0 ? number_format( $value, 2, '.', '' ) : '0' ); ?>" style="width:90px;text-align:center"><span>%</span></div></td>
						<?php endforeach; ?>
					</tr>
					<?php endforeach; ?>
					</tbody>
				</table></div>
				<p class="submit"><button type="submit" class="button button-primary button-large">ذخیره جدول پورسانت</button></p>
			</form>
			<form method="post" action="<?php echo esc_url( admin_url( 'admin-post.php' ) ); ?>"><input type="hidden" name="action" value="sn_backfill_purpose_commissions"><?php wp_nonce_field( 'sn_backfill_purpose_commissions' ); ?><button class="button" type="submit">بازبینی دوبارهٔ فاکتورهای قدیمی</button></form>
			<div class="card" style="max-width:none"><h2>قاعده ثبت تراکنش</h2><p>هر تراکنش با فاکتور، شماره مرحله پرداخت، قصد پرداخت، نقش، درصد، مبلغ مبنا و کاربر گیرنده Snapshot می‌شود. به همین دلیل Refresh، Callback تکراری یا تأیید دوباره، پورسانت تکراری ایجاد نمی‌کند.</p></div>
		</div>
		<?php
	}

	public function render_settings_tab(): void {
		$this->maybe_upgrade();
		$rates = $this->rates();
		$purposes = self::purposes();
		$roles = self::roles();
		?>
		<div class="sn-commission-settings-tab" dir="rtl">
			<div class="notice notice-info inline"><p><strong>مبنای محاسبه:</strong> مبلغ هر مرحله پرداخت تأییدشده × درصد نقش. مقدار ۰ یعنی بدون پورسانت و هر مقدار مثبت باید بین ۰٫۰۱ تا ۱۰۰ باشد.</p><p><strong>مسیر کیف پول:</strong> پورسانت نقش‌های فروش از پرداخت چرخه فروش در «فروش» ثبت می‌شود؛ پورسانت چرخه بیاوین و تمام پورسانت نقش‌های عملیات در «بیاوین» ثبت می‌شود.</p></div>
			<div style="overflow:auto;background:#fff;border:1px solid #dcdcde;border-radius:12px">
			<table class="widefat striped" style="min-width:1900px;border:0">
				<thead><tr><th style="position:sticky;right:0;z-index:3;background:#f6f7f7;min-width:250px">قصد پرداخت</th><?php foreach ( $roles as $role ) : ?><th style="min-width:145px;text-align:center"><?php echo esc_html( $role['label'] ); ?></th><?php endforeach; ?></tr></thead>
				<tbody><?php foreach ( $purposes as $purpose_key => $purpose ) : ?><tr>
					<th style="position:sticky;right:0;z-index:2;background:#fff"><strong><?php echo esc_html( $purpose['label'] ); ?></strong><br><small><?php echo esc_html( $purpose['description'] ); ?></small><br><span style="display:inline-block;margin-top:5px;padding:2px 7px;border-radius:999px;background:<?php echo $purpose['cycle'] === 'biavin' ? '#e7f3ff' : '#edf7ed'; ?>"><?php echo $purpose['cycle'] === 'biavin' ? 'چرخه بیاوین' : 'چرخه فروش'; ?></span></th>
					<?php foreach ( $roles as $role_key => $_role ) : $value = (float) ( $rates[ $purpose_key ][ $role_key ] ?? 0 ); ?><td style="text-align:center"><div style="display:flex;align-items:center;gap:5px;justify-content:center"><input type="number" name="sn_commission_rates[<?php echo esc_attr( $purpose_key ); ?>][<?php echo esc_attr( $role_key ); ?>]" min="0" max="100" step="0.01" value="<?php echo esc_attr( $value > 0 ? number_format( $value, 2, '.', '' ) : '0' ); ?>" style="width:90px;text-align:center"><span>%</span></div></td><?php endforeach; ?>
				</tr><?php endforeach; ?></tbody>
			</table></div>
			<p class="description" style="margin-top:10px">با تعریف درصد جدید، فاکتورهای پرداخت‌شدهٔ قبلی نیز به‌تدریج بررسی می‌شوند و سهم‌های ثبت‌نشده به سوابق کیف پول افزوده می‌شوند. تراکنش‌های قبلاً ثبت‌شده با درصد و مبلغ اولیه حفظ می‌شوند.</p>
		</div>
		<?php
	}

	private function persist_rates( array $raw ) {
		global $wpdb;
		$this->maybe_upgrade();
		$previous = $this->rates();
		$changed = false;
		$purposes = self::purposes(); $roles = self::roles(); $now = current_time( 'mysql' ); $actor = get_current_user_id();
		$wpdb->query( 'START TRANSACTION' );
		foreach ( $purposes as $purpose_key => $_purpose ) {
			foreach ( $roles as $role_key => $_role ) {
				$value = isset( $raw[ $purpose_key ][ $role_key ] ) ? (float) str_replace( ',', '.', sanitize_text_field( (string) $raw[ $purpose_key ][ $role_key ] ) ) : 0.0;
				if ( $value < 0 ) { $value = 0; }
				if ( $value > 100 ) { $value = 100; }
				if ( $value > 0 && $value < 0.01 ) { $value = 0.01; }
				if ( $value > 0 && abs( $value - (float) ( $previous[ $purpose_key ][ $role_key ] ?? 0 ) ) > 0.0001 ) { $changed = true; }
				$sql = $wpdb->prepare(
					"INSERT INTO {$this->table()} (payment_purpose,role_key,rate_percent,updated_by,created_at,updated_at) VALUES (%s,%s,%f,%d,%s,%s) ON DUPLICATE KEY UPDATE rate_percent=VALUES(rate_percent),updated_by=VALUES(updated_by),updated_at=VALUES(updated_at)",
					$purpose_key, $role_key, $value, $actor ?: 0, $now, $now
				);
				if ( false === $wpdb->query( $sql ) ) { $error = $wpdb->last_error; $wpdb->query( 'ROLLBACK' ); return new WP_Error( 'sn_commission_save_failed', 'ذخیره جدول پورسانت انجام نشد: ' . $error ); }
			}
		}
		$wpdb->query( 'COMMIT' );
		if ( $changed && self::is_matrix_engine_enabled() ) { $this->queue_backfill(); }
		return true;
	}

	/** Scan older invoices in small cron batches; existing wallet snapshots remain immutable. */
	public function queue_backfill(): void {
		global $wpdb;
		$table = $wpdb->prefix . 'sn_invoices';
		if ( ! $this->table_exists( $table ) ) { return; }
		$token = wp_generate_uuid4();
		update_option( 'sn_purpose_commission_backfill', [
			'token' => $token, 'cursor' => 0, 'max_id' => (int) $wpdb->get_var( "SELECT MAX(id) FROM {$table}" ),
			'scanned' => 0, 'created' => 0, 'existing' => 0, 'status' => 'در حال بررسی',
		], false );
		wp_schedule_single_event( time() + 10, 'sn_backfill_purpose_commissions', [ $token ] );
	}

	public function run_backfill( string $token ): void {
		global $wpdb;
		$state = get_option( 'sn_purpose_commission_backfill', [] );
		if ( ! is_array( $state ) || ( $state['token'] ?? '' ) !== $token || ! self::is_matrix_engine_enabled() ) { return; }
		$table = $wpdb->prefix . 'sn_invoices';
		$rows = $wpdb->get_col( $wpdb->prepare( "SELECT id FROM {$table} WHERE id>%d AND id<=%d ORDER BY id ASC LIMIT 25", (int) $state['cursor'], (int) $state['max_id'] ) );
		if ( null === $rows ) { $state['status'] = 'خطا در خواندن فاکتورها'; update_option( 'sn_purpose_commission_backfill', $state, false ); return; }
		foreach ( $rows as $id ) {
			$result = $this->reconcile_invoice( (int) $id );
			$state['cursor'] = (int) $id;
			$state['scanned']++;
			$state['created'] += (int) ( $result['created'] ?? 0 );
			$state['existing'] += (int) ( $result['existing'] ?? 0 );
		}
		if ( count( $rows ) < 25 ) { $state['status'] = 'تکمیل شد'; $state['completed_at'] = current_time( 'mysql' ); }
		update_option( 'sn_purpose_commission_backfill', $state, false );
		if ( count( $rows ) === 25 ) { wp_schedule_single_event( time() + 10, 'sn_backfill_purpose_commissions', [ $token ] ); }
	}

	public function handle_backfill(): void {
		if ( ! current_user_can( 'manage_options' ) && ! current_user_can( 'sn_manage_wallets' ) ) { wp_die( 'دسترسی غیرمجاز' ); }
		check_admin_referer( 'sn_backfill_purpose_commissions' );
		$this->queue_backfill();
		wp_safe_redirect( admin_url( 'admin.php?page=sn-commission-matrix' ) );
		exit;
	}

	public function save_settings_from_request( array $request ) {
		$raw = isset( $request['sn_commission_rates'] ) && is_array( $request['sn_commission_rates'] ) ? wp_unslash( $request['sn_commission_rates'] ) : [];
		return $this->persist_rates( $raw );
	}

	public function handle_save_matrix(): void {
		if ( ! current_user_can( 'manage_options' ) && ! current_user_can( 'sn_manage_wallets' ) ) { wp_die( 'دسترسی غیرمجاز' ); }
		check_admin_referer( 'sn_save_purpose_commission_matrix' );
		$raw = isset( $_POST['rates'] ) && is_array( $_POST['rates'] ) ? wp_unslash( $_POST['rates'] ) : [];
		$result = $this->persist_rates( $raw );
		if ( is_wp_error( $result ) ) { wp_die( esc_html( $result->get_error_message() ) ); }
		wp_safe_redirect( add_query_arg( 'sn_commission_saved', '1', admin_url( 'admin.php?page=sn-commission-matrix' ) ) );
		exit;
	}

	private function get_rate( string $purpose, string $role_key ): float {
		global $wpdb;
		if ( ! isset( self::purposes()[ $purpose ], self::roles()[ $role_key ] ) ) { return 0.0; }
		$value = $wpdb->get_var( $wpdb->prepare( "SELECT rate_percent FROM {$this->table()} WHERE payment_purpose=%s AND role_key=%s LIMIT 1", $purpose, $role_key ) );
		return max( 0.0, min( 100.0, (float) $value ) );
	}

	private function resolve_purpose( int $invoice_id, $invoice = null ): string {
		global $wpdb;
		$invoice_table = $wpdb->prefix . 'sn_invoices';
		$inv = is_object( $invoice ) ? $invoice : ( is_array( $invoice ) ? (object) $invoice : null );
		if ( ! $inv ) { $inv = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$invoice_table} WHERE id=%d", $invoice_id ) ); }
		$stored = sanitize_key( (string) ( $inv->payment_purpose ?? '' ) );
		if ( isset( self::purposes()[ $stored ] ) ) { return $stored; }

		$purpose = '';
		$ops_table = $wpdb->prefix . 'sn_project_operations';
		if ( $this->table_exists( $ops_table ) ) {
			$is_upgrade = (int) $wpdb->get_var( $wpdb->prepare( "SELECT COUNT(*) FROM {$ops_table} WHERE upgrade_invoice_id=%d", $invoice_id ) );
			if ( $is_upgrade > 0 ) { $purpose = 'product_star_upgrade'; }
		}
		$link_table = $wpdb->prefix . 'sn_dot_invoice_links';
		if ( $purpose === '' && $this->table_exists( $link_table ) ) {
			$flow_kind = sanitize_key( (string) $wpdb->get_var( $wpdb->prepare( "SELECT flow_kind FROM {$link_table} WHERE invoice_id=%d ORDER BY id DESC LIMIT 1", $invoice_id ) ) );
			if ( $flow_kind === 'conversion_payment' ) { $purpose = 'conversion_purchase'; }
			elseif ( $flow_kind === 'assessment_source' ) { $purpose = 'assessment_fee'; }
		}
		$item_table = $wpdb->prefix . 'sn_invoice_items';
		if ( $purpose === '' && $this->table_exists( $item_table ) ) {
			$item_cols = $this->columns( $item_table );
			if ( in_array( 'product_type', $item_cols, true ) ) {
				$types = array_values( array_unique( array_filter( array_map( 'sanitize_key', (array) $wpdb->get_col( $wpdb->prepare( "SELECT product_type FROM {$item_table} WHERE invoice_id=%d ORDER BY id", $invoice_id ) ) ) ) ) );
				foreach ( [ 'project_upgrade' => 'product_star_upgrade', 'assessment' => 'assessment_fee', 'subscription' => 'subscription_direct', 'product_star' => 'product_star_sale', 'subscription_star' => 'product_star_sale', 'product' => 'product_sale' ] as $type => $mapped ) {
					if ( in_array( $type, $types, true ) ) { $purpose = $mapped; break; }
				}
			}
		}
		if ( $purpose === '' ) {
			$product_id = (int) ( $inv->product_id ?? 0 );
			if ( $product_id > 0 ) {
				$type = sanitize_key( (string) get_post_meta( $product_id, '_sn_product_type', true ) );
				$purpose = [ 'assessment' => 'assessment_fee', 'subscription' => 'subscription_direct', 'product_star' => 'product_star_sale', 'subscription_star' => 'product_star_sale', 'product' => 'product_sale' ][ $type ] ?? '';
			}
		}
		if ( $purpose !== '' && isset( self::purposes()[ $purpose ] ) && in_array( 'payment_purpose', $this->columns( $invoice_table ), true ) ) {
			$wpdb->update( $invoice_table, [ 'payment_purpose' => $purpose ], [ 'id' => $invoice_id ] );
		}
		return $purpose;
	}

	private function user_position( int $user_id ): string {
		global $wpdb;
		if ( $user_id < 1 ) { return ''; }
		$profiles = $wpdb->prefix . 'sn_hr_profiles'; $positions = $wpdb->prefix . 'sn_hr_positions';
		if ( $this->table_exists( $profiles ) && $this->table_exists( $positions ) ) {
			$slug = (string) $wpdb->get_var( $wpdb->prepare( "SELECT pos.slug FROM {$profiles} p INNER JOIN {$positions} pos ON pos.id=p.position_id WHERE p.user_id=%d AND p.is_active=1 ORDER BY p.id DESC LIMIT 1", $user_id ) );
			if ( $slug !== '' ) { return sanitize_key( $slug ); }
		}
		$user = get_user_by( 'id', $user_id );
		$map = [ 'sn_seller'=>'seller','sn_supervisor'=>'supervisor','sn_converter'=>'converter','sn_senior_supervisor'=>'senior_supervisor','sn_sales_manager'=>'sales_manager','sn_sales_deputy'=>'sales_deputy','sn_operations_sales_expert'=>'operations_sales_expert','sn_operations_sales_supervisor'=>'operations_sales_supervisor','sn_operations_sales_manager'=>'operations_sales_manager','sn_operations_executive_manager'=>'operations_executive_manager','sn_operations_execution_expert'=>'operations_execution_expert' ];
		foreach ( (array) ( $user ? $user->roles : [] ) as $role ) { if ( isset( $map[ $role ] ) ) { return $map[ $role ]; } }
		return '';
	}

	private function ancestor_for_position( int $user_id, string $target_position ): int {
		global $wpdb;
		if ( $user_id < 1 ) { return 0; }
		if ( $this->user_position( $user_id ) === $target_position ) { return $user_id; }
		$profiles = $wpdb->prefix . 'sn_hr_profiles'; $positions = $wpdb->prefix . 'sn_hr_positions'; $assignments = $wpdb->prefix . 'sn_hr_assignments';
		if ( $this->table_exists( $profiles ) && $this->table_exists( $positions ) && $this->table_exists( $assignments ) ) {
			$current = $user_id; $seen = [];
			for ( $depth = 0; $depth < 12 && $current > 0; $depth++ ) {
				if ( isset( $seen[ $current ] ) ) { break; } $seen[ $current ] = true;
				$profile_id = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$profiles} WHERE user_id=%d AND is_active=1 ORDER BY id DESC LIMIT 1", $current ) );
				if ( $profile_id < 1 ) { break; }
				$parent = $wpdb->get_row( $wpdb->prepare( "SELECT pp.user_id,pos.slug FROM {$assignments} a INNER JOIN {$profiles} pp ON pp.id=a.parent_profile_id AND pp.is_active=1 INNER JOIN {$positions} pos ON pos.id=pp.position_id WHERE a.child_profile_id=%d AND a.is_current=1 ORDER BY a.id DESC LIMIT 1", $profile_id ) );
				if ( ! $parent ) { break; }
				if ( sanitize_key( (string) $parent->slug ) === $target_position ) { return (int) $parent->user_id; }
				$current = (int) $parent->user_id;
			}
		}
		$current = $user_id; $seen = [];
		for ( $depth = 0; $depth < 12 && $current > 0; $depth++ ) {
			if ( isset( $seen[ $current ] ) ) { break; } $seen[ $current ] = true;
			$next = 0;
			foreach ( [ 'sn_supervisor_id', 'sn_sales_manager_id', 'sn_sales_deputy_id' ] as $meta_key ) {
				$candidate = absint( get_user_meta( $current, $meta_key, true ) );
				if ( $candidate > 0 ) { $next = $candidate; if ( $this->user_position( $candidate ) === $target_position ) { return $candidate; } break; }
			}
			$current = $next;
		}
		return 0;
	}

	private function resolve_recipients( int $invoice_id, $invoice = null ): array {
		global $wpdb;
		$inv = is_object( $invoice ) ? $invoice : ( is_array( $invoice ) ? (object) $invoice : null );
		if ( ! $inv ) { $inv = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$wpdb->prefix}sn_invoices WHERE id=%d", $invoice_id ) ); }
		if ( ! $inv ) { return []; }
		$out = array_fill_keys( array_keys( self::roles() ), 0 );
		$seller_id = (int) ( $inv->original_seller_id ?? 0 ) ?: (int) ( $inv->seller_id ?? 0 );
		$supervisor_id = 0; $converter_id = 0;
		$link_table = $wpdb->prefix . 'sn_dot_invoice_links'; $case_table = $wpdb->prefix . 'sn_dot_cases';
		if ( $this->table_exists( $link_table ) && $this->table_exists( $case_table ) ) {
			$dot = $wpdb->get_row( $wpdb->prepare( "SELECT dl.flow_kind,dl.supervisor_id link_supervisor,c.seller_id,c.supervisor_id,c.converter_id FROM {$link_table} dl LEFT JOIN {$case_table} c ON c.id=dl.case_id WHERE dl.invoice_id=%d ORDER BY dl.id DESC LIMIT 1", $invoice_id ) );
			if ( $dot ) {
				if ( (int) $dot->seller_id > 0 ) { $seller_id = (int) $dot->seller_id; }
				$supervisor_id = (int) $dot->supervisor_id ?: (int) $dot->link_supervisor;
				$converter_id = (int) $dot->converter_id;
			}
		}
		$lead_id = (int) ( $inv->lead_id ?? 0 );
		if ( $lead_id > 0 && $this->table_exists( $wpdb->prefix . 'sn_leads' ) ) {
			$lead_cols = $this->columns( $wpdb->prefix . 'sn_leads' );
			$select = [ 'seller_id' ]; if ( in_array( 'supervisor_id', $lead_cols, true ) ) { $select[] = 'supervisor_id'; }
			$lead = $wpdb->get_row( $wpdb->prepare( 'SELECT ' . implode( ',', $select ) . " FROM {$wpdb->prefix}sn_leads WHERE id=%d", $lead_id ) );
			if ( $lead ) { if ( $seller_id < 1 && (int) ( $lead->seller_id ?? 0 ) > 0 ) { $seller_id = (int) $lead->seller_id; } if ( $supervisor_id < 1 && (int) ( $lead->supervisor_id ?? 0 ) > 0 ) { $supervisor_id = (int) $lead->supervisor_id; } }
		}
		if ( $converter_id < 1 ) {
			$issued = (int) ( $inv->current_stage_issued_by_user_id ?? 0 ) ?: (int) ( $inv->issued_by_user_id ?? 0 );
			if ( $this->user_position( $issued ) === 'converter' ) { $converter_id = $issued; }
		}
		if ( $supervisor_id < 1 && $seller_id > 0 ) { $supervisor_id = $this->ancestor_for_position( $seller_id, 'supervisor' ); }
		$out['seller'] = $seller_id;
		$out['supervisor'] = $supervisor_id;
		$out['converter'] = $converter_id;
		$out['senior_supervisor'] = $this->ancestor_for_position( $supervisor_id ?: $seller_id, 'senior_supervisor' );
		$out['sales_manager'] = $this->ancestor_for_position( $out['senior_supervisor'] ?: ( $supervisor_id ?: $seller_id ), 'sales_manager' );
		$out['sales_deputy'] = $this->ancestor_for_position( $out['sales_manager'] ?: ( $supervisor_id ?: $seller_id ), 'sales_deputy' );

		$operations = $wpdb->prefix . 'sn_project_operations';
		$op = null;
		if ( $this->table_exists( $operations ) ) {
			/*
			 * Upgrade payments point to the operation directly. The initial
			 * Product* purchase points to the membership which later owns the
			 * operation. Supporting both is essential because operations roles may
			 * only become known after the initial invoice has already been paid.
			 */
			$op = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$operations} WHERE upgrade_invoice_id=%d ORDER BY id DESC LIMIT 1", $invoice_id ) );
			if ( ! $op ) {
				$items_table = $wpdb->prefix . 'sn_project_membership_items';
				$members_table = $wpdb->prefix . 'sn_project_memberships';
				if ( $this->table_exists( $items_table ) && $this->table_exists( $members_table ) ) {
					$op = $wpdb->get_row( $wpdb->prepare(
						"SELECT o.* FROM {$operations} o INNER JOIN {$items_table} mi ON mi.id=o.membership_item_id INNER JOIN {$members_table} m ON m.id=mi.membership_id WHERE m.source_invoice_id=%d ORDER BY o.id DESC LIMIT 1",
						$invoice_id
					) );
				}
			}
		}
		if ( $op ) {
			/*
			 * An upgrade invoice is usually issued inside operations, therefore its
			 * own seller/issuer columns are not a reliable snapshot of the original
			 * sales chain. Restore that chain from the membership / Dot case which
			 * created this Product* operation before resolving sales-role shares.
			 */
			$items_table = $wpdb->prefix . 'sn_project_membership_items';
			$members_table = $wpdb->prefix . 'sn_project_memberships';
			$membership = null;
			if ( $this->table_exists( $items_table ) && $this->table_exists( $members_table ) && (int) ( $op->membership_item_id ?? 0 ) > 0 ) {
				$membership = $wpdb->get_row( $wpdb->prepare(
					"SELECT m.origin_user_id,m.origin_role,m.dot_case_id,m.source_invoice_id,m.source_sales_manager_user_id FROM {$items_table} mi INNER JOIN {$members_table} m ON m.id=mi.membership_id WHERE mi.id=%d LIMIT 1",
					(int) $op->membership_item_id
				) );
			}
			if ( $membership ) {
				$origin_id = (int) ( $membership->origin_user_id ?? 0 );
				$origin_role = sanitize_key( (string) ( $membership->origin_role ?? '' ) );
				$dot_case_id = (int) ( $membership->dot_case_id ?? 0 );
				if ( $dot_case_id > 0 && $this->table_exists( $case_table ) ) {
					$origin_case = $wpdb->get_row( $wpdb->prepare( "SELECT seller_id,supervisor_id,converter_id FROM {$case_table} WHERE id=%d LIMIT 1", $dot_case_id ) );
					if ( $origin_case ) {
						if ( (int) $origin_case->seller_id > 0 ) { $seller_id = (int) $origin_case->seller_id; }
						if ( (int) $origin_case->supervisor_id > 0 ) { $supervisor_id = (int) $origin_case->supervisor_id; }
						if ( (int) $origin_case->converter_id > 0 ) { $converter_id = (int) $origin_case->converter_id; }
					}
				}
				if ( $origin_id > 0 ) {
					if ( $origin_role === 'seller' || $this->user_position( $origin_id ) === 'seller' ) { $seller_id = $origin_id; }
					elseif ( $origin_role === 'converter' || $this->user_position( $origin_id ) === 'converter' ) { $converter_id = $origin_id; }
					elseif ( $origin_role === 'supervisor' || $this->user_position( $origin_id ) === 'supervisor' ) { $supervisor_id = $origin_id; }
				}
				/* Fall back to the original source invoice for seller/supervisor. */
				$source_invoice_id = (int) ( $membership->source_invoice_id ?? 0 );
				if ( $source_invoice_id > 0 && ( $seller_id < 1 || $supervisor_id < 1 ) ) {
					$source_inv = $wpdb->get_row( $wpdb->prepare( "SELECT seller_id,original_seller_id,lead_id FROM {$wpdb->prefix}sn_invoices WHERE id=%d LIMIT 1", $source_invoice_id ) );
					if ( $source_inv ) {
						if ( $seller_id < 1 ) { $seller_id = (int) ( $source_inv->original_seller_id ?? 0 ) ?: (int) ( $source_inv->seller_id ?? 0 ); }
						if ( $supervisor_id < 1 && (int) ( $source_inv->lead_id ?? 0 ) > 0 && $this->table_exists( $wpdb->prefix . 'sn_leads' ) && in_array( 'supervisor_id', $this->columns( $wpdb->prefix . 'sn_leads' ), true ) ) {
							$supervisor_id = (int) $wpdb->get_var( $wpdb->prepare( "SELECT supervisor_id FROM {$wpdb->prefix}sn_leads WHERE id=%d", (int) $source_inv->lead_id ) );
						}
					}
				}
			}

			if ( $supervisor_id < 1 && $seller_id > 0 ) { $supervisor_id = $this->ancestor_for_position( $seller_id, 'supervisor' ); }
			$out['seller'] = $seller_id;
			$out['supervisor'] = $supervisor_id;
			$out['converter'] = $converter_id;
			$out['senior_supervisor'] = $this->ancestor_for_position( $supervisor_id ?: $seller_id, 'senior_supervisor' );
			$source_manager = (int) ( $op->source_sales_manager_user_id ?? 0 );
			if ( $source_manager < 1 && $membership ) { $source_manager = (int) ( $membership->source_sales_manager_user_id ?? 0 ); }
			$out['sales_manager'] = $source_manager > 0 ? $source_manager : $this->ancestor_for_position( $out['senior_supervisor'] ?: ( $supervisor_id ?: $seller_id ), 'sales_manager' );
			$out['sales_deputy'] = $this->ancestor_for_position( $out['sales_manager'] ?: ( $supervisor_id ?: $seller_id ), 'sales_deputy' );

			$out['operations_sales_manager'] = (int) ( $op->operations_sales_manager_user_id ?? 0 );
			$out['operations_sales_supervisor'] = (int) ( $op->sales_supervisor_user_id ?? 0 );
			$out['operations_sales_expert'] = (int) ( $op->sales_expert_user_id ?? 0 );
			$exec_table = $wpdb->prefix . 'sn_operations_execution_cases';
			if ( $this->table_exists( $exec_table ) ) {
				$exec = $wpdb->get_row( $wpdb->prepare( "SELECT manager_user_id,expert_user_id FROM {$exec_table} WHERE operation_id=%d LIMIT 1", (int) $op->id ) );
				if ( $exec ) { $out['operations_executive_manager'] = (int) $exec->manager_user_id; $out['operations_execution_expert'] = (int) $exec->expert_user_id; }
			}
		}
		return array_map( 'intval', $out );
	}

	private function wallet_type_for( string $purpose, string $role_key ): string {
		$purpose_def = self::purposes()[ $purpose ] ?? [];
		$role_def = self::roles()[ $role_key ] ?? [];
		if ( (string) ( $purpose_def['cycle'] ?? '' ) === 'sales' && (string) ( $role_def['family'] ?? '' ) === 'sales' ) { return 'seller'; }
		return 'biavin';
	}

	private function wallet_id( int $user_id, string $wallet_type ): int {
		global $wpdb;
		$this->ensure_wallet_tables();
		$table = $wpdb->prefix . 'sn_wallets';
		$id = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$table} WHERE user_id=%d AND wallet_type=%s LIMIT 1", $user_id, $wallet_type ) );
		if ( $id > 0 ) { return $id; }
		$wpdb->query( $wpdb->prepare( "INSERT IGNORE INTO {$table} (user_id,wallet_type,balance,total_credit,total_debit) VALUES (%d,%s,0,0,0)", $user_id, $wallet_type ) );
		return (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$table} WHERE user_id=%d AND wallet_type=%s LIMIT 1", $user_id, $wallet_type ) );
	}

	private function post_transaction( int $user_id, string $wallet_type, float $amount, string $direction, string $type, string $description, int $invoice_id, ?int $lead_id, array $meta, string $idempotency_key ): int {
		global $wpdb;
		if ( $user_id < 1 || $amount <= 0 ) { return 0; }
		$wallet_id = $this->wallet_id( $user_id, $wallet_type ); if ( $wallet_id < 1 ) { return 0; }
		$tx = $wpdb->prefix . 'sn_wallet_transactions';
		$existing = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$tx} WHERE idempotency_key=%s LIMIT 1", $idempotency_key ) );
		if ( $existing > 0 ) { return $existing; }
		$inserted = $wpdb->insert( $tx, [
			'wallet_id'=>$wallet_id,'user_id'=>$user_id,'wallet_type'=>$wallet_type,'invoice_id'=>$invoice_id ?: null,'lead_id'=>$lead_id ?: null,
			'amount'=>$amount,'direction'=>$direction === 'debit' ? 'debit' : 'credit','type'=>sanitize_key( $type ),'status'=>'approved','description'=>$description,
			'meta'=>wp_json_encode( $meta, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ),'idempotency_key'=>substr( $idempotency_key, 0, 191 ),
			'created_by'=>get_current_user_id() ?: null,'created_at'=>current_time( 'mysql' ),
		] );
		if ( ! $inserted ) {
			return (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$tx} WHERE idempotency_key=%s LIMIT 1", $idempotency_key ) );
		}
		$tx_id = (int) $wpdb->insert_id; $sign = $direction === 'debit' ? -1 : 1;
		$updated = $wpdb->query( $wpdb->prepare( "UPDATE {$wpdb->prefix}sn_wallets SET balance=balance+%f,total_credit=total_credit+%f,total_debit=total_debit+%f,updated_at=%s WHERE id=%d", $sign*$amount, $direction==='debit'?0:$amount, $direction==='debit'?$amount:0, current_time('mysql'), $wallet_id ) );
		if ( $updated === false || $updated === 0 ) { $wpdb->delete( $tx, [ 'id'=>$tx_id ] ); return 0; }
		return $tx_id;
	}

	public function on_payment_stage_approved( int $invoice_id, $invoice = null, array $context = [] ): void {
		if ( ! self::is_matrix_engine_enabled() ) { return; }
		$this->maybe_upgrade();
		$stage_no = max( 1, absint( $context['stage_no'] ?? 1 ) );
		$stage_amount = (float) ( $context['stage_amount'] ?? 0 );
		if ( $stage_amount <= 0 ) { return; }
		$this->post_commissions_for_stage( $invoice_id, $stage_no, $stage_amount, $invoice );
	}

	private function post_commissions_for_stage( int $invoice_id, int $stage_no, float $stage_amount, $invoice = null ): array {
		global $wpdb;
		$report = [ 'created'=>0, 'existing'=>0, 'unresolved'=>[], 'purpose'=>'' ];
		$purpose = $this->resolve_purpose( $invoice_id, $invoice ); $report['purpose'] = $purpose;
		if ( ! isset( self::purposes()[ $purpose ] ) || $stage_amount <= 0 ) { return $report; }
		$recipients = $this->resolve_recipients( $invoice_id, $invoice );
		$invoice_row = is_object( $invoice ) ? $invoice : $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$wpdb->prefix}sn_invoices WHERE id=%d", $invoice_id ) );
		$invoice_code = (string) ( $invoice_row->invoice_code ?? ( '#' . $invoice_id ) );
		$lead_id = (int) ( $invoice_row->lead_id ?? 0 ); $purpose_label = (string) self::purposes()[ $purpose ]['label'];
		foreach ( self::roles() as $role_key => $role_def ) {
			$rate = $this->get_rate( $purpose, $role_key ); if ( $rate <= 0 ) { continue; }
			$user_id = (int) ( $recipients[ $role_key ] ?? 0 );
			if ( $user_id < 1 ) { $report['unresolved'][] = $role_key; continue; }
			$amount = round( $stage_amount * $rate / 100, 2 ); if ( $amount <= 0 ) { continue; }
			$wallet_type = $this->wallet_type_for( $purpose, $role_key );
			$key = implode( ':', [ 'purpose-commission', $invoice_id, 'stage', $stage_no, $purpose, $role_key, 'user', $user_id ] );
			// A later HR assignment must not credit the same invoice, stage and role twice.
			$role_prefix = implode( ':', [ 'purpose-commission', $invoice_id, 'stage', $stage_no, $purpose, $role_key, 'user', '' ] );
			$existing = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$wpdb->prefix}sn_wallet_transactions WHERE idempotency_key LIKE %s LIMIT 1", $wpdb->esc_like( $role_prefix ) . '%' ) );
			if ( $existing > 0 ) { $report['existing']++; continue; }
			$description = sprintf( 'پورسانت %s — %s — مرحله %d فاکتور %s (%.2f%%)', $purpose_label, (string) $role_def['label'], $stage_no, $invoice_code, $rate );
			$tx_id = $this->post_transaction( $user_id, $wallet_type, $amount, 'credit', 'purpose_commission', $description, $invoice_id, $lead_id ?: null, [
				'engine'=>'purpose_matrix','payment_purpose'=>$purpose,'payment_purpose_label'=>$purpose_label,'payment_cycle'=>(string) self::purposes()[$purpose]['cycle'],
				'role_key'=>$role_key,'role_label'=>(string)$role_def['label'],'rate_percent'=>$rate,'base_amount'=>$stage_amount,'stage_no'=>$stage_no,
				'invoice_code'=>$invoice_code,'recipient_user_id'=>$user_id,'wallet_label'=>$wallet_type==='seller'?'فروش':'بیاوین','snapshot_version'=>defined('SN_VERSION')?SN_VERSION:'',
			], $key );
			if ( $tx_id > 0 ) { $report['created']++; }
		}
		return $report;
	}

	public function reconcile_invoice( int $invoice_id ): array {
		global $wpdb;
		$report = [ 'stages'=>0,'created'=>0,'existing'=>0 ];
		if ( $invoice_id < 1 || ! self::is_matrix_engine_enabled() ) { return $report; }
		$invoice = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$wpdb->prefix}sn_invoices WHERE id=%d", $invoice_id ) ); if ( ! $invoice ) { return $report; }
		foreach ( [ 'status', 'invoice_status', 'payment_status' ] as $field ) {
			if ( in_array( (string) ( $invoice->$field ?? '' ), [ 'rejected', 'cancelled', 'canceled', 'failed', 'refunded', 'trash' ], true ) ) { return $report; }
		}
		$stage_table = $wpdb->prefix . 'sn_invoice_payment_stages';
		$stages = $this->table_exists( $stage_table ) ? $wpdb->get_results( $wpdb->prepare( "SELECT stage_no,requested_amount,status FROM {$stage_table} WHERE invoice_id=%d AND status IN ('paid','approved') ORDER BY stage_no", $invoice_id ) ) : [];
		if ( ! $stages && ( in_array( (string) ( $invoice->status ?? '' ), [ 'paid', 'approved' ], true ) || in_array( (string) ( $invoice->payment_status ?? '' ), [ 'paid', 'approved' ], true ) ) ) {
			$amount = (float) ( $invoice->paid_total_amount ?? 0 );
			if ( $amount <= 0 ) { $amount = (float) ( $invoice->final_total ?? 0 ) ?: (float) ( $invoice->product_price ?? 0 ); }
			if ( $amount > 0 ) { $stages = [ (object) [ 'stage_no' => 1, 'requested_amount' => $amount, 'status' => 'paid' ] ]; }
		}
		foreach ( $stages as $stage ) { $one = $this->post_commissions_for_stage( $invoice_id, max(1,(int)$stage->stage_no), (float)$stage->requested_amount, $invoice ); $report['stages']++; $report['created'] += (int)$one['created']; $report['existing'] += (int)$one['existing']; }
		return $report;
	}

	public static function reconcile_operation( int $operation_id ): array {
		global $wpdb;
		$out = [ 'created'=>0, 'invoices'=>0, 'existing'=>0 ];
		if ( $operation_id < 1 ) { return $out; }
		$self = self::instance();
		$operations = $wpdb->prefix . 'sn_project_operations';
		$items = $wpdb->prefix . 'sn_project_membership_items';
		$members = $wpdb->prefix . 'sn_project_memberships';
		$row = $wpdb->get_row( $wpdb->prepare(
			"SELECT o.upgrade_invoice_id,m.source_invoice_id FROM {$operations} o LEFT JOIN {$items} mi ON mi.id=o.membership_item_id LEFT JOIN {$members} m ON m.id=mi.membership_id WHERE o.id=%d LIMIT 1",
			$operation_id
		) );
		if ( ! $row ) { return $out; }
		$invoice_ids = array_values( array_unique( array_filter( [ (int) ( $row->source_invoice_id ?? 0 ), (int) ( $row->upgrade_invoice_id ?? 0 ) ] ) ) );
		foreach ( $invoice_ids as $invoice_id ) {
			$r = $self->reconcile_invoice( $invoice_id );
			$out['created'] += (int) ( $r['created'] ?? 0 );
			$out['existing'] += (int) ( $r['existing'] ?? 0 );
			$out['invoices']++;
		}
		return $out;
	}

	public static function reconcile_dot_case( int $case_id ): array {
		global $wpdb;
		if ( $case_id < 1 ) { return []; }
		$ids = [];
		$link_table = $wpdb->prefix . 'sn_dot_invoice_links';
		if ( self::instance()->table_exists( $link_table ) ) { $ids = array_map( 'intval', (array) $wpdb->get_col( $wpdb->prepare( "SELECT invoice_id FROM {$link_table} WHERE case_id=%d ORDER BY id", $case_id ) ) ); }
		$out = [ 'created'=>0,'invoices'=>0 ]; foreach ( array_unique( array_filter( $ids ) ) as $invoice_id ) { $r=self::instance()->reconcile_invoice($invoice_id);$out['created']+=(int)($r['created']??0);$out['invoices']++; } return $out;
	}

	private function reverse_commission_credits( int $invoice_id, string $reason, bool $latest_stage_only = false ): void {
		global $wpdb;
		$tx = $wpdb->prefix . 'sn_wallet_transactions'; if ( ! $this->table_exists( $tx ) ) { return; }
		$credits = $wpdb->get_results( $wpdb->prepare( "SELECT * FROM {$tx} WHERE invoice_id=%d AND direction='credit' AND status='approved' AND type='purpose_commission' ORDER BY id DESC", $invoice_id ), ARRAY_A ) ?: [];
		if ( ! $credits ) { return; }
		$max_stage = 0;
		if ( $latest_stage_only ) { foreach ( $credits as $credit ) { $meta=json_decode((string)($credit['meta']??''),true);$max_stage=max($max_stage,(int)($meta['stage_no']??0)); } }
		foreach ( $credits as $credit ) {
			$meta = json_decode( (string) ( $credit['meta'] ?? '' ), true ); $meta = is_array($meta)?$meta:[];
			if ( $latest_stage_only && $max_stage > 0 && (int)($meta['stage_no']??0) !== $max_stage ) { continue; }
			$credit_id=(int)$credit['id'];$amount=(float)$credit['amount'];$user_id=(int)$credit['user_id']; if($credit_id<1||$amount<=0||$user_id<1){continue;}
			$reversal_key='purpose-commission-reversal:'.$credit_id;
			$reversal_id=$this->post_transaction($user_id,(string)$credit['wallet_type'],$amount,'debit','purpose_commission_reversal','برگشت پورسانت: '.sanitize_text_field($reason),$invoice_id,!empty($credit['lead_id'])?(int)$credit['lead_id']:null,[ 'original_credit_transaction_id'=>$credit_id,'reason'=>$reason,'original_meta'=>$meta ],$reversal_key);
			if($reversal_id>0){$wpdb->update($tx,['type'=>'purpose_commission_reversed','idempotency_key'=>null],['id'=>$credit_id]);}
		}
	}

	public function on_invoice_financial_rejected( int $invoice_id, $invoice = null, string $reason = '' ): void {
		/*
		 * Rejecting a pending stage does not invalidate older approved stages.
		 * A full financial cancellation is different: the caller prefixes the
		 * reason with «لغو توسط مالی», so all purpose commissions must be reversed.
		 */
		$normalized = trim( (string) $reason );
		if ( strpos( $normalized, 'لغو توسط مالی' ) !== false || strpos( strtolower( $normalized ), 'cancel' ) !== false ) {
			$this->reverse_commission_credits( $invoice_id, $normalized ?: 'لغو مالی', false );
		}
	}
	public function on_invoice_financial_reopened( int $invoice_id, $invoice = null, string $reason = '' ): void {
		$this->reverse_commission_credits( $invoice_id, $reason ?: 'بازگشت به بررسی مالی', true );
	}

	public static function render_dual_wallet_cards( int $user_id, int $limit = 20 ): string {
		if ( $user_id < 1 ) { return ''; }
		return '<div class="sn-purpose-wallet-grid">' . self::render_wallet_card( $user_id, 'seller', $limit ) . self::render_wallet_card( $user_id, 'biavin', $limit ) . '</div>'
			. '<style>.sn-purpose-wallet-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px;margin:14px 0}.sn-purpose-wallet-grid .sn-purpose-wallet-card{margin:0}@media(max-width:980px){.sn-purpose-wallet-grid{grid-template-columns:1fr}}</style>';
	}

	public static function render_wallet_card( int $user_id, string $wallet_type = 'biavin', int $limit = 20, string $context = 'standalone' ): string {
		$self = self::instance(); $self->maybe_upgrade(); global $wpdb;
		$wallet_type = $wallet_type === 'seller' ? 'seller' : 'biavin';
		$wallet_id = $self->wallet_id( $user_id, $wallet_type );
		$wallet = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$wpdb->prefix}sn_wallets WHERE id=%d", $wallet_id ), ARRAY_A ) ?: [];
		$transactions = $wpdb->get_results( $wpdb->prepare( "SELECT * FROM {$wpdb->prefix}sn_wallet_transactions WHERE wallet_id=%d ORDER BY id DESC LIMIT %d", $wallet_id, max(1,min(50,$limit)) ), ARRAY_A ) ?: [];
		$title = $wallet_type === 'seller' ? 'کیف پول فروش' : 'کیف پول بیاوین';
		$card_class = $context === 'modal' ? 'sn-purpose-wallet-card is-modal' : 'sn-purpose-wallet-card';
		ob_start(); ?>
		<section class="<?php echo esc_attr( $card_class ); ?>" dir="rtl">
			<header><div><small>پورسانت و تراکنش‌ها</small><h3><?php echo esc_html($title); ?></h3></div><div class="sn-purpose-wallet-balance"><span>موجودی</span><strong><?php echo esc_html(class_exists('SN_Helpers')?SN_Helpers::format_price((float)($wallet['balance']??0)):number_format_i18n((float)($wallet['balance']??0))); ?></strong></div></header>
			<div class="sn-purpose-wallet-history"><h4>سابقه تراکنش‌ها</h4>
			<?php if(!$transactions):?><div class="sn-purpose-wallet-empty">هنوز تراکنشی در این کیف پول ثبت نشده است.</div><?php else:?>
			<?php foreach($transactions as $row):?><div class="sn-purpose-wallet-row"><div><strong><?php echo esc_html((string)($row['description']?:'تراکنش کیف پول'));?></strong><small><?php echo esc_html(class_exists('SN_Helpers')?SN_Helpers::gregorian_to_jalali_date((string)$row['created_at']):(string)$row['created_at']);?><?php echo !empty($row['invoice_id'])?' · فاکتور #'.esc_html((string)(int)$row['invoice_id']):'';?></small></div><b class="<?php echo $row['direction']==='debit'?'is-debit':'is-credit';?>"><?php echo $row['direction']==='debit'?'− ':'+ ';?><?php echo esc_html(class_exists('SN_Helpers')?SN_Helpers::format_price((float)$row['amount']):number_format_i18n((float)$row['amount']));?></b></div><?php endforeach;?>
			<?php endif;?></div>
		</section>
		<style>.sn-purpose-wallet-card{margin:16px 0;padding:18px;color:#153247;border:1px solid #dbe8e7;border-radius:18px;background:linear-gradient(135deg,#f7fbfb,#eef7f6);box-shadow:0 8px 28px rgba(24,62,69,.06)}.sn-purpose-wallet-card>header{display:flex;align-items:center;justify-content:space-between;gap:16px;color:#153247}.sn-purpose-wallet-card h3{margin:2px 0 0;color:#153247}.sn-purpose-wallet-card header small{color:#647778}.sn-purpose-wallet-balance{min-width:190px;padding:10px 14px;border-radius:13px;background:#fff;border:1px solid #dce8e7}.sn-purpose-wallet-balance span,.sn-purpose-wallet-balance strong{display:block}.sn-purpose-wallet-balance span{font-size:11px;color:#6b7b7c}.sn-purpose-wallet-balance strong{font-size:19px;margin-top:3px;color:#153247}.sn-purpose-wallet-history{margin-top:14px}.sn-purpose-wallet-history h4{margin:0 0 8px;color:#153247}.sn-purpose-wallet-row{display:flex;align-items:center;justify-content:space-between;gap:12px;padding:11px 4px;border-top:1px solid #dfe9e8}.sn-purpose-wallet-row strong,.sn-purpose-wallet-row small{display:block}.sn-purpose-wallet-row small{margin-top:4px;color:#738183}.sn-purpose-wallet-row b{white-space:nowrap}.sn-purpose-wallet-row .is-credit{color:#13724b}.sn-purpose-wallet-row .is-debit{color:#ad2d3b}.sn-purpose-wallet-empty{padding:18px;text-align:center;border:1px dashed #cbdcda;border-radius:12px;background:#fff;color:#718081}.sn-purpose-wallet-card.is-modal{margin:0;padding:0;border:0;border-radius:0;background:transparent;box-shadow:none}.sn-purpose-wallet-card.is-modal>header{padding:0 0 16px;border-bottom:1px solid #e2ecee}.sn-purpose-wallet-card.is-modal .sn-purpose-wallet-history{margin-top:16px}.sn-purpose-wallet-card.is-modal .sn-purpose-wallet-row{padding:12px 2px}@media(max-width:640px){.sn-purpose-wallet-card>header{align-items:stretch;flex-direction:column}.sn-purpose-wallet-balance{min-width:0}.sn-purpose-wallet-row{align-items:flex-start;flex-direction:column}}</style>
		<?php return (string)ob_get_clean();
	}

	public static function render_wallet_launcher( int $user_id, string $wallet_type = 'biavin', int $limit = 20 ): string {
		if ( $user_id < 1 ) { return ''; }
		$self = self::instance(); $self->maybe_upgrade(); global $wpdb;
		$wallet_type = $wallet_type === 'seller' ? 'seller' : 'biavin';
		$wallet_id = $self->wallet_id( $user_id, $wallet_type );
		$balance = (float) $wpdb->get_var( $wpdb->prepare( "SELECT balance FROM {$wpdb->prefix}sn_wallets WHERE id=%d", $wallet_id ) );
		$title = $wallet_type === 'seller' ? 'کیف پول فروش' : 'کیف پول بیاوین';
		$modal_id = 'sn-purpose-wallet-modal-' . $wallet_type . '-' . $user_id;
		$balance_text = class_exists( 'SN_Helpers' ) ? SN_Helpers::format_price( $balance ) : number_format_i18n( $balance );
		ob_start(); ?>
		<button type="button" class="sn-ops-wallet-trigger" data-sn-wallet-open="<?php echo esc_attr( $modal_id ); ?>" aria-haspopup="dialog" aria-label="<?php echo esc_attr( 'باز کردن ' . $title . '؛ موجودی ' . $balance_text ); ?>">
			<span class="sn-ops-wallet-trigger-icon" aria-hidden="true"><svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M4 7.5h14.5A1.5 1.5 0 0 1 20 9v9a2 2 0 0 1-2 2H5a3 3 0 0 1-3-3V6a3 3 0 0 1 3-3h11"/><path d="M20 12h-5a2 2 0 0 0 0 4h5"/><circle cx="15" cy="14" r=".5" fill="currentColor" stroke="none"/></svg></span>
			<span class="sn-ops-wallet-trigger-copy"><strong>کیف پول</strong></span>
		</button>
		<div id="<?php echo esc_attr( $modal_id ); ?>" class="sn-purpose-wallet-modal" data-sn-wallet-modal hidden>
			<div class="sn-purpose-wallet-modal-backdrop" data-sn-wallet-close></div>
			<section class="sn-purpose-wallet-modal-dialog" role="dialog" aria-modal="true" aria-label="<?php echo esc_attr( $title ); ?>">
				<button type="button" class="sn-purpose-wallet-modal-close" data-sn-wallet-close aria-label="بستن کیف پول">×</button>
				<?php echo self::render_wallet_card( $user_id, $wallet_type, $limit, 'modal' ); // phpcs:ignore WordPress.Security.EscapeOutput.OutputNotEscaped ?>
			</section>
		</div>
		<?php return (string) ob_get_clean();
	}
}
