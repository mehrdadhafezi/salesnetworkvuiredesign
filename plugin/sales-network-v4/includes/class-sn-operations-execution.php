<?php
if ( ! defined( 'ABSPATH' ) ) { exit; }

/**
 * Final execution layer for Biavin cards.
 *
 * Product settings are snapshotted when a card reaches this layer. Historic
 * cases never depend on the current WooCommerce product configuration.
 */
final class SN_Operations_Execution {
	private const DB_VERSION = '2026-09-21-operations-execution-codes-v2';
	private static ?self $instance = null;
	private bool $hooks_registered = false;
	private array $table_cache = [];

	public static function instance(): self {
		if ( null === self::$instance ) { self::$instance = new self(); }
		return self::$instance;
	}

	private function __construct() {}

	public static function activate(): void {
		$self = self::instance();
		$self->install_schema();
		$self->ensure_role_and_hr();
		$self->ensure_page();
	}

	public function register_hooks(): void {
		if ( $this->hooks_registered ) { return; }
		$this->hooks_registered = true;
		add_action( 'init', [ $this, 'maybe_upgrade' ], 9 );
		add_action( 'admin_init', [ $this, 'maybe_upgrade' ], 9 );
		add_action( 'admin_menu', [ $this, 'register_admin_page' ], 40 );
		add_action( 'add_meta_boxes_product', [ $this, 'register_product_metabox' ], 30 );
		add_action( 'save_post_product', [ $this, 'save_product_meta' ], 30, 2 );
		add_action( 'admin_post_sn_execution_save_form', [ $this, 'handle_save_form' ] );
		add_action( 'admin_post_sn_execution_save_integration', [ $this, 'handle_save_integration' ] );
		add_action( 'admin_post_sn_operations_execution_action', [ $this, 'handle_action' ] );
		add_action( 'admin_post_sn_operations_execution_export', [ $this, 'handle_manual_export' ] );
		add_action( 'admin_post_sn_operations_execution_shipping_update', [ $this, 'handle_shipping_update' ] );
		add_shortcode( 'sn_operations_execution_expert_panel', [ $this, 'render_expert_panel' ] );
	}

	private function tables(): array {
		global $wpdb;
		return [
			'codes' => $wpdb->prefix . 'sn_operations_wallet_codes',
			'cases' => $wpdb->prefix . 'sn_operations_execution_cases',
			'forms' => $wpdb->prefix . 'sn_operations_form_definitions',
			'submissions' => $wpdb->prefix . 'sn_operations_form_submissions',
			'events' => $wpdb->prefix . 'sn_operations_execution_events',
		];
	}

	private function table_exists( string $table ): bool {
		global $wpdb;
		if ( ! array_key_exists( $table, $this->table_cache ) ) {
			$this->table_cache[ $table ] = (string) $wpdb->get_var( $wpdb->prepare( 'SHOW TABLES LIKE %s', $table ) ) === $table;
		}
		return $this->table_cache[ $table ];
	}

	public function maybe_upgrade(): void {
		if ( (string) get_option( 'sn_operations_execution_db_version', '' ) !== self::DB_VERSION ) { $this->install_schema(); }
		if ( (string) get_option( 'sn_operations_execution_access_version', '' ) !== self::DB_VERSION ) {
			$this->ensure_role_and_hr();
			$this->ensure_page();
		}
	}

	private function install_schema(): void {
		global $wpdb;
		require_once ABSPATH . 'wp-admin/includes/upgrade.php';
		$t = $this->tables(); $charset = $wpdb->get_charset_collate();
		dbDelta( "CREATE TABLE {$t['codes']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			product_id BIGINT UNSIGNED NOT NULL,
			code_hash CHAR(64) NOT NULL,
			activation_code VARCHAR(191) NOT NULL,
			destination_url TEXT DEFAULT NULL,
			membership_item_id BIGINT UNSIGNED DEFAULT NULL,
			assigned_at DATETIME DEFAULT NULL,
			created_at DATETIME NOT NULL,
			PRIMARY KEY (id), UNIQUE KEY code_hash (code_hash), UNIQUE KEY membership_item_id (membership_item_id), KEY product_available (product_id,membership_item_id)
		) {$charset};" );
		dbDelta( "CREATE TABLE {$t['forms']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			title VARCHAR(191) NOT NULL,
			slug VARCHAR(191) NOT NULL,
			description TEXT DEFAULT NULL,
			fields_json LONGTEXT NOT NULL,
			is_active TINYINT(1) NOT NULL DEFAULT 1,
			created_by BIGINT UNSIGNED DEFAULT NULL,
			updated_by BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id), UNIQUE KEY slug (slug), KEY is_active (is_active)
		) {$charset};" );
		dbDelta( "CREATE TABLE {$t['cases']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			operation_id BIGINT UNSIGNED NOT NULL,
			membership_item_id BIGINT UNSIGNED NOT NULL,
			content_product_id BIGINT UNSIGNED NOT NULL,
			fulfillment_type VARCHAR(30) NOT NULL DEFAULT 'unconfigured',
			form_definition_id BIGINT UNSIGNED DEFAULT NULL,
			config_snapshot_json LONGTEXT DEFAULT NULL,
			form_schema_snapshot_json LONGTEXT DEFAULT NULL,
			manager_user_id BIGINT UNSIGNED DEFAULT NULL,
			expert_user_id BIGINT UNSIGNED DEFAULT NULL,
			assigned_by_user_id BIGINT UNSIGNED DEFAULT NULL,
			assigned_at DATETIME DEFAULT NULL,
			status VARCHAR(40) NOT NULL DEFAULT 'manager_queue',
			no_answer_count INT UNSIGNED NOT NULL DEFAULT 0,
			follow_up_at DATETIME DEFAULT NULL,
			last_contact_at DATETIME DEFAULT NULL,
			last_note TEXT DEFAULT NULL,
			cancel_reason TEXT DEFAULT NULL,
			cancelled_at DATETIME DEFAULT NULL,
			wallet_external_id VARCHAR(191) DEFAULT NULL,
			wallet_activation_code VARCHAR(191) DEFAULT NULL,
			wallet_status VARCHAR(40) DEFAULT NULL,
			wallet_usage_json LONGTEXT DEFAULT NULL,
			wallet_customer_url TEXT DEFAULT NULL,
			wallet_sms_sent_at DATETIME DEFAULT NULL,
			shipping_status VARCHAR(40) DEFAULT NULL,
			shipping_province VARCHAR(100) DEFAULT NULL,
			shipping_city VARCHAR(100) DEFAULT NULL,
			shipping_address TEXT DEFAULT NULL,
			shipping_postal_code VARCHAR(30) DEFAULT NULL,
			shipping_carrier VARCHAR(120) DEFAULT NULL,
			shipping_tracking_code VARCHAR(191) DEFAULT NULL,
			form_submission_id BIGINT UNSIGNED DEFAULT NULL,
			final_decision VARCHAR(30) DEFAULT NULL,
			completed_at DATETIME DEFAULT NULL,
			completed_by_user_id BIGINT UNSIGNED DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id), UNIQUE KEY operation_id (operation_id), KEY expert_status (expert_user_id,status),
			KEY fulfillment_type (fulfillment_type,status), KEY membership_item_id (membership_item_id), KEY follow_up_at (follow_up_at)
		) {$charset};" );
		dbDelta( "CREATE TABLE {$t['submissions']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			execution_case_id BIGINT UNSIGNED NOT NULL,
			form_definition_id BIGINT UNSIGNED DEFAULT NULL,
			schema_snapshot_json LONGTEXT NOT NULL,
			values_json LONGTEXT NOT NULL,
			submitted_by_user_id BIGINT UNSIGNED NOT NULL,
			submitted_at DATETIME NOT NULL,
			final_decision VARCHAR(30) DEFAULT NULL,
			final_note TEXT DEFAULT NULL,
			decided_by_user_id BIGINT UNSIGNED DEFAULT NULL,
			decided_at DATETIME DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id), UNIQUE KEY execution_case_id (execution_case_id), KEY form_definition_id (form_definition_id)
		) {$charset};" );
		dbDelta( "CREATE TABLE {$t['events']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			execution_case_id BIGINT UNSIGNED NOT NULL,
			operation_id BIGINT UNSIGNED NOT NULL,
			membership_item_id BIGINT UNSIGNED NOT NULL,
			actor_user_id BIGINT UNSIGNED DEFAULT NULL,
			event_key VARCHAR(90) NOT NULL,
			details_json LONGTEXT DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id), KEY case_events (execution_case_id,id), KEY operation_id (operation_id), KEY event_key (event_key)
		) {$charset};" );
		$this->table_cache = [];
		update_option( 'sn_operations_execution_db_version', self::DB_VERSION, false );
	}

	private function ensure_role_and_hr(): void {
		if ( ! get_role( 'sn_operations_execution_expert' ) ) { add_role( 'sn_operations_execution_expert', 'کارشناس اجرایی عملیات', [ 'read' => true, 'sn_work_operations_execution' => true ] ); }
		$role = get_role( 'sn_operations_execution_expert' );
		if ( $role ) {
			$role->add_cap( 'read' ); $role->add_cap( 'sn_work_operations_execution' );
			foreach ( [ 'edit_posts','delete_posts','publish_posts','upload_files','edit_pages','delete_pages','manage_options','list_users','create_users','edit_users','delete_users' ] as $cap ) { $role->remove_cap( $cap ); }
		}
		global $wpdb; $positions = $wpdb->prefix . 'sn_hr_positions'; $mappings = $wpdb->prefix . 'sn_hr_position_role_mappings';
		if ( ! $this->table_exists( $positions ) ) { return; }
		$columns = (array) $wpdb->get_col( "SHOW COLUMNS FROM {$positions}", 0 );
		$data = [ 'slug'=>'operations_execution_expert', 'label'=>'کارشناس اجرایی عملیات', 'panel_key'=>'operations_execution_expert', 'sort_order'=>98, 'is_active'=>1, 'is_system'=>1 ];
		$position_id = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$positions} WHERE slug=%s LIMIT 1", 'operations_execution_expert' ) );
		if ( $position_id ) { $wpdb->update( $positions, array_intersect_key( $data, array_flip( $columns ) ), [ 'id'=>$position_id ] ); }
		else { $wpdb->insert( $positions, array_intersect_key( $data, array_flip( $columns ) ) ); $position_id = (int) $wpdb->insert_id; }
		if ( $position_id && $this->table_exists( $mappings ) ) {
			$mcols = (array) $wpdb->get_col( "SHOW COLUMNS FROM {$mappings}", 0 );
			$existing = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$mappings} WHERE legacy_role=%s LIMIT 1", 'sn_operations_execution_expert' ) );
			$mapping = array_intersect_key( [ 'legacy_role'=>'sn_operations_execution_expert', 'position_id'=>$position_id, 'is_active'=>1 ], array_flip( $mcols ) );
			if ( $existing ) { $wpdb->update( $mappings, $mapping, [ 'id'=>$existing ] ); } else { $wpdb->insert( $mappings, $mapping ); }
		}
		update_option( 'sn_operations_execution_access_version', self::DB_VERSION, false );
	}

	private function ensure_page(): void {
		$option = 'sn_operations_execution_expert_panel_page_id';
		$page_id = absint( get_option( $option, 0 ) ); $page = $page_id ? get_post( $page_id ) : null;
		if ( ! $page || $page->post_type !== 'page' || $page->post_status === 'trash' ) {
			$existing = get_page_by_path( 'crm-operations-execution-expert', OBJECT, 'page' );
			$page_id = $existing ? (int) $existing->ID : (int) wp_insert_post( [ 'post_title'=>'پنل کارشناس اجرایی عملیات', 'post_name'=>'crm-operations-execution-expert', 'post_content'=>'[sn_operations_execution_expert_panel]', 'post_status'=>'publish', 'post_type'=>'page' ] );
			if ( $page_id ) { update_option( $option, $page_id ); update_post_meta( $page_id, '_sn_system_page', '1' ); }
		}
	}

	public function register_admin_page(): void {
		add_submenu_page( 'sn-dashboard', 'فرم‌های عملیات', 'فرم‌های عملیات', 'manage_options', 'sn-operations-forms', [ $this, 'render_admin_forms' ] );
	}

	private function fulfillment_types(): array {
		return [
			'wallet_charge' => 'شارژ کیف پول',
			'physical_invoice' => 'فاکتور کالا',
			'form' => 'فرم',
		];
	}

	public function fulfillment_label( string $type ): string { return $this->fulfillment_types()[ sanitize_key( $type ) ] ?? 'نوع اجرا تعیین نشده'; }

	private function active_forms(): array {
		global $wpdb; $table = $this->tables()['forms'];
		if ( ! $this->table_exists( $table ) ) { return []; }
		return $wpdb->get_results( "SELECT * FROM {$table} WHERE is_active=1 ORDER BY title ASC,id DESC" ) ?: [];
	}

	public function register_product_metabox(): void {
		add_meta_box( 'sn-operations-execution-product', 'نوع اجرای محصول*', [ $this, 'render_product_metabox' ], 'product', 'side', 'default' );
	}

	public function render_product_metabox( WP_Post $post ): void {
		$type = sanitize_key( (string) get_post_meta( $post->ID, '_sn_execution_fulfillment_type', true ) );
		$form_id = absint( get_post_meta( $post->ID, '_sn_execution_form_id', true ) );
		$service_key = sanitize_key( (string) get_post_meta( $post->ID, '_sn_execution_wallet_service_key', true ) );
		$wallet_type = sanitize_key( (string) get_post_meta( $post->ID, '_sn_execution_wallet_type', true ) );
		if ( ! in_array( $wallet_type, [ 'cash', 'installment' ], true ) ) { $wallet_type = 'cash'; }
		$instructions = (string) get_post_meta( $post->ID, '_sn_execution_instructions', true );
		$destination_url = (string) get_post_meta( $post->ID, '_sn_execution_wallet_destination_url', true );
		global $wpdb;
		$codes_table = $this->tables()['codes'];
		$code_counts = $this->table_exists( $codes_table ) ? $wpdb->get_row( $wpdb->prepare( "SELECT COUNT(*) total, SUM(membership_item_id IS NULL) available FROM {$codes_table} WHERE product_id=%d", $post->ID ) ) : null;
		wp_nonce_field( 'sn_execution_product_meta', 'sn_execution_product_nonce' ); ?>
		<div dir="rtl" data-sn-execution-product-meta>
			<p style="font-size:11px;color:#646970">این تنظیم فقط برای محصول* استفاده و هنگام ورود کارت به واحد اجرا Snapshot می‌شود.</p>
			<label style="display:block;font-weight:700;margin-bottom:5px">نوع اجرای کارت</label>
			<select name="sn_execution_fulfillment_type" data-sn-execution-type style="width:100%"><option value="">— انتخاب نوع —</option><?php foreach ( $this->fulfillment_types() as $key=>$label ) : ?><option value="<?php echo esc_attr( $key ); ?>" <?php selected( $type, $key ); ?>><?php echo esc_html( $label ); ?></option><?php endforeach; ?></select>
			<div data-sn-execution-field="wallet_charge" style="margin-top:10px" <?php echo $type === 'wallet_charge' ? '' : 'hidden'; ?>>
				<label style="display:block;font-weight:600;margin-bottom:4px">نوع کیف پول مقصد</label>
				<select name="sn_execution_wallet_type" style="width:100%;margin-bottom:8px"><option value="cash" <?php selected( $wallet_type, 'cash' ); ?>>کیف پول نقدی</option><option value="installment" <?php selected( $wallet_type, 'installment' ); ?>>کیف پول اقساطی</option></select>
				<label style="display:block;font-weight:600;margin-bottom:4px">شناسه خدمت در کیف پول مقصد</label><input name="sn_execution_wallet_service_key" value="<?php echo esc_attr( $service_key ); ?>" style="width:100%" placeholder="مثال: shoe-credit"><small>برای نوع «شارژ کیف پول» اجباری است و همراه نوع کیف پول Snapshot می‌شود.</small>
				<label style="display:block;margin-top:8px">لینک سایت مقصد<input type="url" name="sn_execution_wallet_destination_url" value="<?php echo esc_attr( $destination_url ); ?>" style="width:100%" placeholder="https://example.com"></label>
				<label style="display:block;margin-top:8px">افزودن کدهای یک‌بارمصرف (هر خط یک کد)<textarea name="sn_execution_wallet_new_codes" rows="5" style="width:100%" autocomplete="off"></textarea></label>
				<small>موجودی: <?php echo esc_html( number_format_i18n( (int) ( $code_counts->available ?? 0 ) ) ); ?> کد آزاد از <?php echo esc_html( number_format_i18n( (int) ( $code_counts->total ?? 0 ) ) ); ?> کد. کدهای مصرف‌شده حفظ می‌شوند؛ ثبت کد تکراری اثری ندارد.</small>
			</div>
			<div data-sn-execution-field="form" style="margin-top:10px" <?php echo $type === 'form' ? '' : 'hidden'; ?>><label>فرم این کارت</label><select name="sn_execution_form_id" style="width:100%"><option value="0">— انتخاب فرم —</option><?php foreach ( $this->active_forms() as $form ) : ?><option value="<?php echo esc_attr( (string) $form->id ); ?>" <?php selected( $form_id, (int) $form->id ); ?>><?php echo esc_html( (string) $form->title ); ?></option><?php endforeach; ?></select><small><a href="<?php echo esc_url( admin_url( 'admin.php?page=sn-operations-forms' ) ); ?>">مدیریت فرم‌ها</a></small></div>
			<label style="display:block;margin-top:10px">راهنمای کارشناس<textarea name="sn_execution_instructions" rows="3" style="width:100%"><?php echo esc_textarea( $instructions ); ?></textarea></label>
			<p style="font-size:10px;color:#8a5a00">تغییر این تنظیم روی کارت‌های قبلی اثر ندارد.</p>
		</div><script>(function(){var root=document.querySelector('[data-sn-execution-product-meta]');if(!root)return;var type=root.querySelector('[data-sn-execution-type]');function sync(){root.querySelectorAll('[data-sn-execution-field]').forEach(function(el){el.hidden=el.getAttribute('data-sn-execution-field')!==type.value;});}type.addEventListener('change',sync);sync();}());</script>
		<?php
	}

	public function save_product_meta( int $post_id, WP_Post $post ): void {
		if ( ! isset( $_POST['sn_execution_product_nonce'] ) || ! wp_verify_nonce( sanitize_text_field( wp_unslash( $_POST['sn_execution_product_nonce'] ) ), 'sn_execution_product_meta' ) || ! current_user_can( 'edit_post', $post_id ) || ( defined( 'DOING_AUTOSAVE' ) && DOING_AUTOSAVE ) ) { return; }
		$type = sanitize_key( wp_unslash( $_POST['sn_execution_fulfillment_type'] ?? '' ) );
		if ( ! isset( $this->fulfillment_types()[ $type ] ) ) { $type = ''; }
		$wallet_type = sanitize_key( wp_unslash( $_POST['sn_execution_wallet_type'] ?? 'cash' ) );
		if ( ! in_array( $wallet_type, [ 'cash', 'installment' ], true ) ) { $wallet_type = 'cash'; }
		$service_key = sanitize_key( wp_unslash( $_POST['sn_execution_wallet_service_key'] ?? '' ) );
		update_post_meta( $post_id, '_sn_execution_fulfillment_type', $type );
		update_post_meta( $post_id, '_sn_execution_wallet_type', $type === 'wallet_charge' ? $wallet_type : '' );
		update_post_meta( $post_id, '_sn_execution_wallet_service_key', $type === 'wallet_charge' ? $service_key : '' );
		update_post_meta( $post_id, '_sn_execution_form_id', $type === 'form' ? absint( $_POST['sn_execution_form_id'] ?? 0 ) : 0 );
		update_post_meta( $post_id, '_sn_execution_instructions', sanitize_textarea_field( wp_unslash( $_POST['sn_execution_instructions'] ?? '' ) ) );
		if ( $type === 'wallet_charge' ) {
			$url = esc_url_raw( trim( (string) wp_unslash( $_POST['sn_execution_wallet_destination_url'] ?? '' ) ) );
			update_post_meta( $post_id, '_sn_execution_wallet_destination_url', wp_http_validate_url( $url ) ? $url : '' );
			$codes = preg_split( '/\r\n|\r|\n/', (string) wp_unslash( $_POST['sn_execution_wallet_new_codes'] ?? '' ) );
			global $wpdb; $codes_table = $this->tables()['codes'];
			foreach ( array_slice( (array) $codes, 0, 500 ) as $raw_code ) {
				$code = trim( sanitize_text_field( $raw_code ) );
				if ( $code === '' || strlen( $code ) > 191 ) { continue; }
				$wpdb->query( $wpdb->prepare( "INSERT IGNORE INTO {$codes_table} (product_id,code_hash,activation_code,created_at) VALUES (%d,%s,%s,%s)", $post_id, hash( 'sha256', $code ), $code, current_time( 'mysql' ) ) );
			}
		}
	}

	/** Reserve one code under a row lock. Caller must own the item and hold an active DB transaction. */
	public function reserve_wallet_code( int $product_id, int $item_id ): ?string {
		global $wpdb; $table = $this->tables()['codes'];
		if ( ! $this->table_exists( $table ) ) { return null; }
		$existing = $wpdb->get_var( $wpdb->prepare( "SELECT activation_code FROM {$table} WHERE membership_item_id=%d LIMIT 1", $item_id ) );
		if ( $existing !== null ) { return (string) $existing; }
		$row = $wpdb->get_row( $wpdb->prepare( "SELECT id,activation_code FROM {$table} WHERE product_id=%d AND membership_item_id IS NULL ORDER BY id ASC LIMIT 1 FOR UPDATE", $product_id ) );
		if ( ! $row ) { return null; }
		$url = esc_url_raw( (string) get_post_meta( $product_id, '_sn_execution_wallet_destination_url', true ) );
		$updated = $wpdb->query( $wpdb->prepare( "UPDATE {$table} SET membership_item_id=%d,destination_url=%s,assigned_at=%s WHERE id=%d AND membership_item_id IS NULL", $item_id, $url, current_time( 'mysql' ), (int) $row->id ) );
		return $updated === 1 ? (string) $row->activation_code : null;
	}

	public function customer_wallet_code( int $item_id ): string {
		global $wpdb; $table = $this->tables()['codes'];
		return $this->table_exists( $table ) ? (string) $wpdb->get_var( $wpdb->prepare( "SELECT activation_code FROM {$table} WHERE membership_item_id=%d LIMIT 1", $item_id ) ) : '';
	}

	public function customer_wallet_destination( int $item_id ): string {
		global $wpdb; $table = $this->tables()['codes'];
		return $this->table_exists( $table ) ? (string) $wpdb->get_var( $wpdb->prepare( "SELECT destination_url FROM {$table} WHERE membership_item_id=%d LIMIT 1", $item_id ) ) : '';
	}

	private function sanitize_form_fields( array $rows ) {
		$out = []; $allowed = [ 'text','textarea','number','date','select','radio','checkbox' ];
		foreach ( array_slice( $rows, 0, 80 ) as $row ) {
			if ( ! is_array( $row ) ) { continue; }
			$label = sanitize_text_field( $row['label'] ?? '' ); $key = sanitize_key( $row['key'] ?? '' ); $type = sanitize_key( $row['type'] ?? 'text' );
			if ( $label === '' || $key === '' || ! in_array( $type, $allowed, true ) ) { continue; }
			$options = array_values( array_filter( array_map( 'sanitize_text_field', preg_split( '/[\r\n,،]+/', (string) ( $row['options'] ?? '' ) ) ?: [] ) ) );
			$out[ $key ] = [ 'key'=>$key, 'label'=>$label, 'type'=>$type, 'required'=>! empty( $row['required'] ) ? 1 : 0, 'options'=>array_slice( $options, 0, 50 ) ];
		}
		return array_values( $out );
	}

	public function handle_save_form(): void {
		if ( ! current_user_can( 'manage_options' ) ) { wp_die( 'دسترسی غیرمجاز' ); }
		check_admin_referer( 'sn_execution_save_form' );
		global $wpdb; $table = $this->tables()['forms']; $id = absint( $_POST['form_id'] ?? 0 );
		$title = sanitize_text_field( wp_unslash( $_POST['title'] ?? '' ) ); $slug = sanitize_title( wp_unslash( $_POST['slug'] ?? $title ) );
		$fields = $this->sanitize_form_fields( (array) wp_unslash( $_POST['fields'] ?? [] ) );
		if ( $title === '' || $slug === '' || ! $fields ) { $this->admin_redirect( 'عنوان و حداقل یک فیلد معتبر اجباری است.', false ); }
		$data = [ 'title'=>$title, 'slug'=>$slug, 'description'=>sanitize_textarea_field( wp_unslash( $_POST['description'] ?? '' ) ), 'fields_json'=>wp_json_encode( $fields, JSON_UNESCAPED_UNICODE|JSON_UNESCAPED_SLASHES ), 'is_active'=>! empty( $_POST['is_active'] ) ? 1 : 0, 'updated_by'=>get_current_user_id(), 'updated_at'=>current_time( 'mysql' ) ];
		if ( $id ) { $ok = $wpdb->update( $table, $data, [ 'id'=>$id ] ); }
		else { $data['created_by']=get_current_user_id(); $data['created_at']=current_time( 'mysql' ); $ok=$wpdb->insert( $table, $data ); }
		$this->admin_redirect( false === $ok ? 'ذخیره فرم انجام نشد؛ نامک باید یکتا باشد.' : 'فرم ذخیره شد.', false !== $ok );
	}

	public function handle_save_integration(): void {
		if ( ! current_user_can( 'manage_options' ) ) { wp_die( 'دسترسی غیرمجاز' ); }
		check_admin_referer( 'sn_execution_save_integration' );
		$endpoint = esc_url_raw( wp_unslash( $_POST['wallet_endpoint'] ?? '' ) );
		if ( $endpoint !== '' && ( ! wp_http_validate_url( $endpoint ) || strtolower( (string) wp_parse_url( $endpoint, PHP_URL_SCHEME ) ) !== 'https' ) ) { $this->admin_redirect( 'نشانی اتصال کیف پول باید یک آدرس معتبر HTTPS باشد.', false ); }
		update_option( 'sn_operations_wallet_endpoint', $endpoint, false );
		$token = trim( (string) wp_unslash( $_POST['wallet_bearer'] ?? '' ) );
		if ( $token !== '' && $token !== '********' ) { update_option( 'sn_operations_wallet_bearer', sanitize_text_field( $token ), false ); }
		if ( ! empty( $_POST['clear_wallet_bearer'] ) ) { delete_option( 'sn_operations_wallet_bearer' ); }
		$this->admin_redirect( 'تنظیمات اتصال ذخیره شد.', true );
	}

	private function admin_redirect( string $message, bool $success ): void {
		wp_safe_redirect( add_query_arg( [ 'sn_execution_notice'=>rawurlencode( $message ), 'sn_execution_kind'=>$success?'success':'error' ], admin_url( 'admin.php?page=sn-operations-forms' ) ) ); exit;
	}

	public function render_admin_forms(): void {
		if ( ! current_user_can( 'manage_options' ) ) { return; }
		global $wpdb; $table = $this->tables()['forms']; $forms = $this->table_exists( $table ) ? ( $wpdb->get_results( "SELECT * FROM {$table} ORDER BY id DESC" ) ?: [] ) : [];
		$editing_id = absint( $_GET['edit_form'] ?? 0 ); $editing = $editing_id ? $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$table} WHERE id=%d", $editing_id ) ) : null;
		$fields = $editing ? json_decode( (string) $editing->fields_json, true ) : []; $fields = is_array( $fields ) ? $fields : [];
		?>
		<div class="wrap" dir="rtl"><h1>فرم‌های عملیات کارت</h1>
		<?php if ( ! empty( $_GET['sn_execution_notice'] ) ) : ?><div class="notice notice-<?php echo (string) ( $_GET['sn_execution_kind'] ?? '' ) === 'success' ? 'success' : 'error'; ?>"><p><?php echo esc_html( sanitize_text_field( rawurldecode( (string) wp_unslash( $_GET['sn_execution_notice'] ) ) ) ); ?></p></div><?php endif; ?>
		<div style="display:grid;grid-template-columns:minmax(420px,1fr) minmax(300px,.7fr);gap:18px;align-items:start">
		<form method="post" action="<?php echo esc_url( admin_url( 'admin-post.php' ) ); ?>" class="card" style="max-width:none"><input type="hidden" name="action" value="sn_execution_save_form"><input type="hidden" name="form_id" value="<?php echo esc_attr( (string) $editing_id ); ?>"><?php wp_nonce_field( 'sn_execution_save_form' ); ?><h2><?php echo $editing ? 'ویرایش فرم' : 'فرم جدید'; ?></h2>
		<p><label>عنوان<br><input class="regular-text" name="title" required value="<?php echo esc_attr( (string) ( $editing->title ?? '' ) ); ?>"></label></p><p><label>نامک یکتا<br><input class="regular-text" name="slug" value="<?php echo esc_attr( (string) ( $editing->slug ?? '' ) ); ?>"></label></p><p><label>توضیح<br><textarea class="large-text" name="description" rows="2"><?php echo esc_textarea( (string) ( $editing->description ?? '' ) ); ?></textarea></label></p>
		<table class="widefat striped" data-sn-form-fields><thead><tr><th>عنوان</th><th>کلید</th><th>نوع</th><th>گزینه‌ها</th><th>اجباری</th><th></th></tr></thead><tbody><?php foreach ( $fields as $index=>$field ) : $this->render_field_row( (int) $index, $field ); endforeach; ?></tbody></table><p><button type="button" class="button" data-sn-add-field>+ افزودن فیلد</button></p><p><label><input type="checkbox" name="is_active" value="1" <?php checked( ! $editing || ! empty( $editing->is_active ) ); ?>> فعال</label></p><button class="button button-primary">ذخیره فرم</button></form>
		<div><div class="card" style="max-width:none"><h2>فرم‌های تعریف‌شده</h2><?php if ( ! $forms ) : ?><p>هنوز فرمی تعریف نشده است.</p><?php endif; ?><?php foreach ( $forms as $form ) : ?><p><strong><?php echo esc_html( (string) $form->title ); ?></strong> <code><?php echo esc_html( (string) $form->slug ); ?></code> · <?php echo (int) $form->is_active ? 'فعال' : 'غیرفعال'; ?> <a href="<?php echo esc_url( add_query_arg( 'edit_form', (int) $form->id, admin_url( 'admin.php?page=sn-operations-forms' ) ) ); ?>">ویرایش</a></p><?php endforeach; ?></div>
		<form method="post" action="<?php echo esc_url( admin_url( 'admin-post.php' ) ); ?>" class="card" style="max-width:none"><input type="hidden" name="action" value="sn_execution_save_integration"><?php wp_nonce_field( 'sn_execution_save_integration' ); ?><h2>اتصال افزونه کیف پول مقصد</h2><p>تا زمان دریافت افزونه ثالث، این اتصال غیرفعال می‌ماند و هیچ شارژ صوری ثبت نمی‌شود.</p><p><label>Endpoint امن HTTPS<br><input class="large-text" type="url" name="wallet_endpoint" value="<?php echo esc_attr( (string) get_option( 'sn_operations_wallet_endpoint', '' ) ); ?>"></label></p><p><label>Bearer Token<br><input class="large-text" type="password" name="wallet_bearer" value="<?php echo get_option( 'sn_operations_wallet_bearer', '' ) !== '' ? '********' : ''; ?>" autocomplete="new-password"></label></p><p><label><input type="checkbox" name="clear_wallet_bearer" value="1"> حذف توکن فعلی</label></p><button class="button button-primary">ذخیره اتصال</button></form></div></div></div>
		<script>(function(){var table=document.querySelector('[data-sn-form-fields] tbody'),add=document.querySelector('[data-sn-add-field]');if(!table||!add)return;var index=table.children.length;function row(i){return '<tr><td><input name="fields['+i+'][label]" required></td><td><input name="fields['+i+'][key]" dir="ltr" required></td><td><select name="fields['+i+'][type]"><option value="text">متن</option><option value="textarea">متن بلند</option><option value="number">عدد</option><option value="date">تاریخ</option><option value="select">انتخابی</option><option value="radio">رادیویی</option><option value="checkbox">تیک</option></select></td><td><input name="fields['+i+'][options]" placeholder="هر گزینه با ویرگول"></td><td><input type="checkbox" name="fields['+i+'][required]" value="1"></td><td><button type="button" class="button-link-delete" data-sn-remove-field>حذف</button></td></tr>';}add.addEventListener('click',function(){table.insertAdjacentHTML('beforeend',row(index++));});document.addEventListener('click',function(e){if(e.target.matches('[data-sn-remove-field]'))e.target.closest('tr').remove();});if(!table.children.length)add.click();}());</script>
		<?php
	}

	private function render_field_row( int $index, array $field ): void {
		$options = implode( '، ', (array) ( $field['options'] ?? [] ) ); ?>
		<tr><td><input name="fields[<?php echo $index; ?>][label]" value="<?php echo esc_attr( (string) ( $field['label'] ?? '' ) ); ?>" required></td><td><input name="fields[<?php echo $index; ?>][key]" dir="ltr" value="<?php echo esc_attr( (string) ( $field['key'] ?? '' ) ); ?>" required></td><td><select name="fields[<?php echo $index; ?>][type]"><?php foreach ( [ 'text'=>'متن','textarea'=>'متن بلند','number'=>'عدد','date'=>'تاریخ','select'=>'انتخابی','radio'=>'رادیویی','checkbox'=>'تیک' ] as $key=>$label ) : ?><option value="<?php echo esc_attr( $key ); ?>" <?php selected( (string) ( $field['type'] ?? '' ), $key ); ?>><?php echo esc_html( $label ); ?></option><?php endforeach; ?></select></td><td><input name="fields[<?php echo $index; ?>][options]" value="<?php echo esc_attr( $options ); ?>"></td><td><input type="checkbox" name="fields[<?php echo $index; ?>][required]" value="1" <?php checked( ! empty( $field['required'] ) ); ?>></td><td><button type="button" class="button-link-delete" data-sn-remove-field>حذف</button></td></tr>
		<?php
	}

	private function position_for_user( int $user_id ): string {
		global $wpdb; $profiles=$wpdb->prefix.'sn_hr_profiles'; $positions=$wpdb->prefix.'sn_hr_positions';
		if ( $this->table_exists( $profiles ) && $this->table_exists( $positions ) ) {
			$slug=(string)$wpdb->get_var($wpdb->prepare("SELECT p.slug FROM {$profiles} hp INNER JOIN {$positions} p ON p.id=hp.position_id WHERE hp.user_id=%d AND COALESCE(hp.is_active,1)=1 LIMIT 1",$user_id)); if($slug!==''){return $slug;}
		}
		$user=get_user_by('id',$user_id); return $user instanceof WP_User && in_array('sn_operations_execution_expert',(array)$user->roles,true)?'operations_execution_expert':'';
	}

	private function direct_experts( int $manager_id ): array {
		if ( current_user_can( 'manage_options' ) ) { return $this->execution_experts(); }
		global $wpdb; $profiles=$wpdb->prefix.'sn_hr_profiles';$positions=$wpdb->prefix.'sn_hr_positions';$assignments=$wpdb->prefix.'sn_hr_assignments';
		if(!$this->table_exists($profiles)||!$this->table_exists($positions)||!$this->table_exists($assignments)){return [];}
		$ids=$wpdb->get_col($wpdb->prepare("SELECT child.user_id FROM {$assignments} a INNER JOIN {$profiles} parent ON parent.id=a.parent_profile_id INNER JOIN {$profiles} child ON child.id=a.child_profile_id INNER JOIN {$positions} pos ON pos.id=child.position_id WHERE parent.user_id=%d AND a.relationship_type='reports_to' AND a.is_current=1 AND COALESCE(child.is_active,1)=1 AND pos.slug='operations_execution_expert' ORDER BY child.user_id",$manager_id))?:[];
		$out=[];foreach($ids as $id){$user=get_user_by('id',(int)$id);if($user instanceof WP_User){$out[]=$user;}}return $out;
	}

	private function execution_experts(): array {
		$out=[];foreach(get_users(['role'=>'sn_operations_execution_expert','number'=>2000,'orderby'=>'display_name','order'=>'ASC']) as $u){$out[(int)$u->ID]=$u;}
		global $wpdb;$profiles=$wpdb->prefix.'sn_hr_profiles';$positions=$wpdb->prefix.'sn_hr_positions';if($this->table_exists($profiles)&&$this->table_exists($positions)){$ids=$wpdb->get_col("SELECT hp.user_id FROM {$profiles} hp INNER JOIN {$positions} p ON p.id=hp.position_id WHERE p.slug='operations_execution_expert' AND COALESCE(hp.is_active,1)=1")?:[];foreach($ids as $id){$u=get_user_by('id',(int)$id);if($u instanceof WP_User){$out[(int)$id]=$u;}}}return array_values($out);
	}

	private function operation_context( int $operation_id ) {
		global $wpdb; $operations=$wpdb->prefix.'sn_project_operations';$items=$wpdb->prefix.'sn_project_membership_items';$members=$wpdb->prefix.'sn_project_memberships';$invoices=$wpdb->prefix.'sn_invoices';
		return $wpdb->get_row($wpdb->prepare("SELECT o.*,mi.content_product_id,mi.content_name_snapshot,m.id membership_id,m.customer_wp_id,m.customer_name,m.customer_phone,m.source_invoice_id,m.subscription_name_snapshot,i.province,i.city,i.customer_address,i.customer_postal_code FROM {$operations} o INNER JOIN {$items} mi ON mi.id=o.membership_item_id INNER JOIN {$members} m ON m.id=mi.membership_id LEFT JOIN {$invoices} i ON i.id=m.source_invoice_id WHERE o.id=%d LIMIT 1",$operation_id));
	}

	private function repair_pending_configuration( object $case, object $operation ): object {
		if ( (string)($operation->stage??'') !== 'executive_manager_queue' || (int)($case->expert_user_id??0) > 0 || $this->configuration_issue($case)==='' ) { return $case; }
		global $wpdb;$t=$this->tables();$ctx=$this->operation_context((int)$case->operation_id);if(!$ctx){return$case;}
		$product_id=(int)$ctx->content_product_id;$type=sanitize_key((string)get_post_meta($product_id,'_sn_execution_fulfillment_type',true));if(!isset($this->fulfillment_types()[$type])){return$case;}
		$form_id=$type==='form'?absint(get_post_meta($product_id,'_sn_execution_form_id',true)):0;$form=null;if($form_id){$form=$wpdb->get_row($wpdb->prepare("SELECT * FROM {$t['forms']} WHERE id=%d AND is_active=1 LIMIT 1",$form_id));}
		if($type==='form'&&!$form){return$case;}
		$wallet_type=sanitize_key((string)get_post_meta($product_id,'_sn_execution_wallet_type',true));if(!in_array($wallet_type,['cash','installment'],true)){$wallet_type='cash';}$config=['service_key'=>sanitize_key((string)get_post_meta($product_id,'_sn_execution_wallet_service_key',true)),'wallet_type'=>$wallet_type,'instructions'=>sanitize_textarea_field((string)get_post_meta($product_id,'_sn_execution_instructions',true))];
		$wpdb->update($t['cases'],['fulfillment_type'=>$type,'form_definition_id'=>$form_id?:null,'config_snapshot_json'=>wp_json_encode($config,JSON_UNESCAPED_UNICODE|JSON_UNESCAPED_SLASHES),'form_schema_snapshot_json'=>$form?(string)$form->fields_json:null,'updated_at'=>current_time('mysql')],['id'=>(int)$case->id]);
		return$wpdb->get_row($wpdb->prepare("SELECT * FROM {$t['cases']} WHERE id=%d LIMIT 1",(int)$case->id))?:$case;
	}

	public function ensure_case( object $operation, int $manager_id = 0 ) {
		global $wpdb; $t=$this->tables(); $op_id=(int)($operation->id??0); if($op_id<1){return null;}
		$case=$wpdb->get_row($wpdb->prepare("SELECT * FROM {$t['cases']} WHERE operation_id=%d LIMIT 1",$op_id));if($case){return $this->repair_pending_configuration($case,$operation);}
		$ctx=$this->operation_context($op_id);if(!$ctx){return null;}$product_id=(int)$ctx->content_product_id;$type=sanitize_key((string)get_post_meta($product_id,'_sn_execution_fulfillment_type',true));if(!isset($this->fulfillment_types()[$type])){$type='unconfigured';}
		$form_id=$type==='form'?absint(get_post_meta($product_id,'_sn_execution_form_id',true)):0;$form=null;if($form_id){$form=$wpdb->get_row($wpdb->prepare("SELECT * FROM {$t['forms']} WHERE id=%d AND is_active=1 LIMIT 1",$form_id));}
		$wallet_type=sanitize_key((string)get_post_meta($product_id,'_sn_execution_wallet_type',true));if(!in_array($wallet_type,['cash','installment'],true)){$wallet_type='cash';}$config=['service_key'=>sanitize_key((string)get_post_meta($product_id,'_sn_execution_wallet_service_key',true)),'wallet_type'=>$wallet_type,'instructions'=>sanitize_textarea_field((string)get_post_meta($product_id,'_sn_execution_instructions',true))];
		$stage=sanitize_key((string)($operation->stage??'executive_manager_queue'));$case_status=['executive_manager_queue'=>'manager_queue','execution_expert'=>'assigned','executive_in_progress'=>'assigned','execution_wallet_active'=>'wallet_charged','execution_form_review'=>'form_completed','shipping_queue'=>'shipping_queue','shipping_processing'=>'shipping_processing','shipped'=>'shipped','delivered'=>'delivered','shipping_not_sent'=>'shipping_not_sent','shipping_returned'=>'shipping_returned','shipping_cancelled'=>'shipping_cancelled','completed'=>'completed','execution_not_executed'=>'not_executed','execution_cancelled'=>'cancelled'][$stage]??'manager_queue';
		$wpdb->insert($t['cases'],['operation_id'=>$op_id,'membership_item_id'=>(int)$ctx->membership_item_id,'content_product_id'=>$product_id,'fulfillment_type'=>$type,'form_definition_id'=>$form_id?:null,'config_snapshot_json'=>wp_json_encode($config,JSON_UNESCAPED_UNICODE|JSON_UNESCAPED_SLASHES),'form_schema_snapshot_json'=>$form?(string)$form->fields_json:null,'manager_user_id'=>$manager_id?:null,'status'=>$case_status,'shipping_province'=>sanitize_text_field((string)$ctx->province),'shipping_city'=>sanitize_text_field((string)$ctx->city),'shipping_address'=>sanitize_textarea_field((string)$ctx->customer_address),'shipping_postal_code'=>sanitize_text_field((string)$ctx->customer_postal_code),'created_at'=>current_time('mysql'),'updated_at'=>current_time('mysql')]);
		return $wpdb->get_row($wpdb->prepare("SELECT * FROM {$t['cases']} WHERE operation_id=%d LIMIT 1",$op_id));
	}

	public function is_assignable( object $operation, int $manager_id ): bool {
		if((string)($operation->stage??'')!=='executive_manager_queue'){return false;}$case=$this->ensure_case($operation,$manager_id);return $case&&$this->configuration_issue($case)==='';
	}

	private function configuration_issue( object $case ): string {
		if ( (string) $case->fulfillment_type === 'unconfigured' ) { return 'نوع اجرای این محصول* هنوز تعیین نشده است.'; }
		if ( (string) $case->fulfillment_type === 'form' ) {
			$schema = json_decode( (string) $case->form_schema_snapshot_json, true );
			if ( ! is_array( $schema ) || ! $schema ) { return 'فرم معتبر برای این محصول* انتخاب نشده است.'; }
		}
		if ( (string) $case->fulfillment_type === 'wallet_charge' ) {
			$config = json_decode( (string) $case->config_snapshot_json, true );
			$config = is_array( $config ) ? $config : [];
			if ( sanitize_key( (string) ( $config['service_key'] ?? '' ) ) === '' ) { return 'شناسه خدمت کیف پول مقصد برای این محصول* تعریف نشده است.'; }
			$wallet_type = sanitize_key( (string) ( $config['wallet_type'] ?? 'cash' ) );
			if ( ! in_array( $wallet_type, [ 'cash', 'installment' ], true ) ) { return 'نوع کیف پول مقصد برای این محصول* معتبر نیست.'; }
		}
		return '';
	}

	public function assign_operations( array $operation_ids, int $expert_id, int $actor ): array {
		$operation_ids=array_values(array_unique(array_filter(array_map('absint',array_slice($operation_ids,0,500)))));$valid=false;foreach($this->direct_experts($actor) as $expert){if((int)$expert->ID===$expert_id){$valid=true;break;}}if(current_user_can('manage_options')){$valid=false;foreach($this->execution_experts() as $expert){if((int)$expert->ID===$expert_id){$valid=true;break;}}}
		if(!$valid){return['success'=>false,'message'=>'کارشناس اجرایی انتخاب‌شده زیرمجموعه این مدیر نیست.','assigned'=>0];}
		global $wpdb;$operations=$wpdb->prefix.'sn_project_operations';$assigned=0;$skipped=0;$now=current_time('mysql');
		foreach($operation_ids as $op_id){$op=$wpdb->get_row($wpdb->prepare("SELECT * FROM {$operations} WHERE id=%d LIMIT 1",$op_id));if(!$op||(string)$op->stage!=='executive_manager_queue'){$skipped++;continue;}$case=$this->ensure_case($op,$actor);if(!$case||$this->configuration_issue($case)!==''){$skipped++;continue;}$wpdb->query('START TRANSACTION');$changed=$wpdb->query($wpdb->prepare("UPDATE {$operations} SET stage='execution_expert',assigned_by_user_id=%d,assigned_at=%s,updated_at=%s WHERE id=%d AND stage='executive_manager_queue'",$actor,$now,$now,$op_id));if($changed){$ok=$wpdb->update($this->tables()['cases'],['manager_user_id'=>$actor,'expert_user_id'=>$expert_id,'assigned_by_user_id'=>$actor,'assigned_at'=>$now,'status'=>'assigned','updated_at'=>$now],['id'=>(int)$case->id]);if(false!==$ok){$wpdb->query('COMMIT');$assigned++;$this->log_event((int)$case->id,'operations_execution_assigned',['expert_user_id'=>$expert_id],$actor);$this->project_event((int)$case->membership_item_id,'operations_execution_assigned',['expert_user_id'=>$expert_id],$actor);if(class_exists('SN_Purpose_Commission')){SN_Purpose_Commission::reconcile_operation((int)$op_id);}continue;}}$wpdb->query('ROLLBACK');$skipped++;}
		return['success'=>$assigned>0,'message'=>$assigned>0?number_format_i18n($assigned).' کارت به کارشناس اجرایی تخصیص یافت.':'هیچ کارتی تخصیص نیافت؛ نوع اجرای محصول* یا وضعیت صف را بررسی کنید.','assigned'=>$assigned,'skipped'=>$skipped];
	}

	public function render_manager_bulk( array $rows, int $manager_id, string $form_id ): string {
		$experts=$this->direct_experts($manager_id);$assignable=0;foreach($rows as $row){if($this->is_assignable($row,$manager_id)){$assignable++;}}
		ob_start();?><section class="sn-ops-bulk-bar sn-exec-bulk" data-sn-ops-bulk-bar><div class="sn-ops-bulk-selectors"><label class="sn-ops-select-all"><input type="checkbox" data-sn-ops-select-all> انتخاب همه کارت‌های قابل تخصیص</label><button type="button" data-sn-ops-select-visible>انتخاب نتایج فیلترشده</button><button type="button" class="is-secondary" data-sn-ops-clear-selection>پاک‌کردن</button><span><strong data-sn-ops-selected-count>۰</strong> از <?php echo esc_html(number_format_i18n($assignable)); ?> کارت</span></div><form id="<?php echo esc_attr($form_id); ?>" method="post" action="<?php echo esc_url(admin_url('admin-post.php')); ?>" class="sn-ops-bulk-form" data-sn-ops-bulk-form><input type="hidden" name="action" value="sn_operations_execution_action"><input type="hidden" name="execution_action" value="assign"><?php wp_nonce_field('sn_operations_execution_assign');?><select name="expert_user_id" data-sn-ops-bulk-target required <?php disabled(!$experts);?>><option value="">انتخاب کارشناس اجرایی عملیات</option><?php foreach($experts as $expert):?><option value="<?php echo esc_attr((string)$expert->ID);?>"><?php echo esc_html((string)$expert->display_name);?></option><?php endforeach;?></select><button data-sn-ops-bulk-submit disabled>تخصیص گروهی</button><?php if(!$experts):?><small>ابتدا در HR کارشناس اجرایی را زیرمجموعه مدیر اجرایی تعریف کنید.</small><?php endif;?></form></section><?php return(string)ob_get_clean();
	}

	public function render_manager_case( object $row, int $manager_id, string $bulk_form_id ): string {
		if((string)($row->stage??'')!=='executive_manager_queue'){global$wpdb;$cases_table=$this->tables()['cases'];$existing_case=$wpdb->get_var($wpdb->prepare("SELECT id FROM {$cases_table} WHERE operation_id=%d LIMIT 1",(int)($row->id??0)));if(!$existing_case){return'<div class="sn-exec-manager-box"><span>این رکورد مربوط به اجرای نسخه قبل است و بدون بازنویسی نگهداری شده است.</span></div>';}}
		$prechecked_case = $this->ensure_case( $row, $manager_id );
		$configuration_issue = $prechecked_case ? $this->configuration_issue( $prechecked_case ) : '';
		if ( $configuration_issue !== '' && (string) $prechecked_case->fulfillment_type !== 'unconfigured' ) {
			return '<div class="sn-exec-manager-box"><strong>' . esc_html( $this->fulfillment_label( (string) $prechecked_case->fulfillment_type ) ) . '</strong><div class="sn-ops-warning">' . esc_html( $configuration_issue ) . ' سوابق فعلی بدون تغییر باقی مانده‌اند.</div></div>';
		}
		$case=$this->ensure_case($row,$manager_id);if(!$case){return'<div class="sn-ops-warning">ساخت پرونده اجرایی انجام نشد.</div>';}$experts=$this->direct_experts($manager_id);ob_start();?><div class="sn-exec-manager-box"><strong><?php echo esc_html($this->fulfillment_label((string)$case->fulfillment_type));?></strong><?php if((string)$case->fulfillment_type==='unconfigured'):?><div class="sn-ops-warning">نوع اجرای این محصول* هنوز در صفحه محصول تعیین نشده است؛ سوابق فعلی بدون تغییر باقی مانده‌اند.</div><?php elseif((string)$row->stage==='executive_manager_queue'):?><label class="sn-ops-card-select"><input type="checkbox" name="operation_ids[]" value="<?php echo esc_attr((string)$row->id);?>" form="<?php echo esc_attr($bulk_form_id);?>" data-sn-ops-bulk-checkbox><span>انتخاب برای تخصیص</span></label><form method="post" action="<?php echo esc_url(admin_url('admin-post.php'));?>" class="sn-ops-form"><input type="hidden" name="action" value="sn_operations_execution_action"><input type="hidden" name="execution_action" value="assign"><input type="hidden" name="operation_ids[]" value="<?php echo esc_attr((string)$row->id);?>"><?php wp_nonce_field('sn_operations_execution_assign');?><select name="expert_user_id" required <?php disabled(!$experts);?>><option value="">انتخاب کارشناس اجرایی عملیات</option><?php foreach($experts as $expert):?><option value="<?php echo esc_attr((string)$expert->ID);?>"><?php echo esc_html((string)$expert->display_name);?></option><?php endforeach;?></select><button <?php disabled(!$experts);?>>تخصیص کارت</button></form><?php else:$expert=$case->expert_user_id?get_user_by('id',(int)$case->expert_user_id):null;?><span>کارشناس: <b><?php echo esc_html($expert instanceof WP_User?(string)$expert->display_name:'تخصیص نشده');?></b></span><span>وضعیت اجرا: <b><?php echo esc_html($this->case_status_label((string)$case->status));?></b></span><?php endif;?></div><?php return(string)ob_get_clean();
	}

	private function can_work_case( object $case, int $user_id ): bool { return current_user_can('manage_options')||((int)$case->expert_user_id===$user_id&&(current_user_can('sn_work_operations_execution')||$this->position_for_user($user_id)==='operations_execution_expert')); }

	private function normalize_followup( string $raw ) {
		$parsed=SN_Helpers::normalize_jalali_tehran_datetime($raw,'09:00',false);return is_wp_error($parsed)?$parsed:(string)$parsed['mysql'];
	}

	private function action_redirect( string $message, bool $success, string $fallback='' ): void { $url=wp_get_referer()?:$fallback?:home_url('/');wp_safe_redirect(add_query_arg(['sn_ops_notice'=>'1','sn_ops_kind'=>$success?'success':'error'],$url));exit; }

	public function handle_action(): void {
		if(!is_user_logged_in()){auth_redirect();}$actor=get_current_user_id();$action=sanitize_key(wp_unslash($_POST['execution_action']??''));
		if($action==='assign'){check_admin_referer('sn_operations_execution_assign');if(!(current_user_can('manage_options')||current_user_can('sn_manage_operations_execution')||$this->position_for_user($actor)==='operations_executive_manager')){wp_die('دسترسی غیرمجاز');}$result=$this->assign_operations((array)wp_unslash($_POST['operation_ids']??[]),absint($_POST['expert_user_id']??0),$actor);$this->action_redirect((string)$result['message'],!empty($result['success']));}
		$case_id=absint($_POST['case_id']??0);check_admin_referer('sn_operations_execution_case_'.$case_id);global $wpdb;$t=$this->tables();$case=$wpdb->get_row($wpdb->prepare("SELECT * FROM {$t['cases']} WHERE id=%d LIMIT 1",$case_id));if(!$case||!$this->can_work_case($case,$actor)){wp_die('این کارت به شما تخصیص ندارد.');}$ctx=$this->operation_context((int)$case->operation_id);if(!$ctx){$this->action_redirect('اطلاعات کارت پیدا نشد.',false);}$now=current_time('mysql');
		$terminal_statuses = [ 'completed', 'cancelled', 'not_executed', 'shipping_not_sent', 'shipping_returned', 'shipping_cancelled', 'delivered' ];
		if ( in_array( (string) $case->status, $terminal_statuses, true ) ) { $this->action_redirect( 'این پرونده قبلاً بسته شده است.', false ); }
		if ( in_array( $action, [ 'no_answer', 'follow_up', 'cancel' ], true ) && ! in_array( (string) $case->status, [ 'assigned', 'no_answer', 'follow_up' ], true ) ) { $this->action_redirect( 'این کارت قبلاً به مرحله بعد رفته است.', false ); }
		if($action==='no_answer'){$wpdb->query($wpdb->prepare("UPDATE {$t['cases']} SET status='no_answer',no_answer_count=no_answer_count+1,last_contact_at=%s,last_note=%s,updated_at=%s WHERE id=%d AND status NOT IN ('completed','cancelled','not_executed')",$now,sanitize_textarea_field(wp_unslash($_POST['note']??'')),$now,$case_id));$this->log_event($case_id,'execution_no_answer',['count'=>(int)$case->no_answer_count+1],$actor);$this->project_event((int)$case->membership_item_id,'operations_execution_no_answer',['count'=>(int)$case->no_answer_count+1],$actor);$this->action_redirect('عدم پاسخ ثبت شد.',true);}
		if($action==='follow_up'){$follow=$this->normalize_followup((string)($_POST['followup_date']??''));if(is_wp_error($follow)){$this->action_redirect($follow->get_error_message(),false);}$wpdb->update($t['cases'],['follow_up_at'=>$follow,'last_note'=>sanitize_textarea_field(wp_unslash($_POST['note']??'')),'status'=>'follow_up','updated_at'=>$now],['id'=>$case_id]);$this->project_event((int)$case->membership_item_id,'operations_execution_follow_up',['follow_up_at'=>$follow],$actor);$this->action_redirect('تاریخ تماس مجدد ثبت شد.',true);}
		if($action==='cancel'){$reason=trim(sanitize_textarea_field(wp_unslash($_POST['cancel_reason']??'')));if($reason===''){$this->action_redirect('دلیل انصراف اجباری است.',false);}$wpdb->update($t['cases'],['status'=>'cancelled','cancel_reason'=>$reason,'cancelled_at'=>$now,'last_note'=>$reason,'follow_up_at'=>null,'updated_at'=>$now],['id'=>$case_id]);$wpdb->update($wpdb->prefix.'sn_project_operations',['stage'=>'execution_cancelled','cancel_reason'=>$reason,'cancelled_at'=>$now,'cancelled_by_user_id'=>$actor,'updated_at'=>$now],['id'=>(int)$case->operation_id]);$this->project_event((int)$case->membership_item_id,'operations_execution_cancelled',['reason'=>$reason],$actor);$this->action_redirect('انصراف ثبت شد.',true);}
		if($action==='wallet_charge'){
			if((string)$case->fulfillment_type!=='wallet_charge'){$this->action_redirect('نوع این کارت شارژ کیف پول نیست.',false);}
			$config_issue=$this->configuration_issue($case);if($config_issue!==''){$this->action_redirect($config_issue,false);}
			if((string)$case->wallet_activation_code!==''||in_array((string)$case->wallet_status,['charged','charging','sync_pending','used'],true)){$this->action_redirect('درخواست شارژ این کارت قبلاً ثبت شده یا در حال پردازش است.',false);}
			$locked=$wpdb->query($wpdb->prepare("UPDATE {$t['cases']} SET wallet_status='charging',updated_at=%s WHERE id=%d AND (wallet_status IS NULL OR wallet_status='' OR wallet_status='failed') AND (wallet_activation_code IS NULL OR wallet_activation_code='')",$now,$case_id));
			if(!$locked){$this->action_redirect('درخواست شارژ هم‌اکنون در حال پردازش است.',false);}
			$result=$this->wallet_request($case,$ctx,'charge');
			if(empty($result['success'])){$wpdb->update($t['cases'],['wallet_status'=>'failed','last_note'=>sanitize_text_field((string)($result['message']??'')),'updated_at'=>$now],['id'=>$case_id]);$this->action_redirect((string)($result['message']??'شارژ کیف پول ناموفق بود.'),false);}
			$code=sanitize_text_field((string)($result['activation_code']??''));
			$external_id=sanitize_text_field((string)($result['external_id']??''));
			if($code===''){$wpdb->update($t['cases'],['wallet_status'=>'sync_pending','wallet_external_id'=>$external_id,'last_note'=>'پاسخ موفق بدون کد فعال‌سازی دریافت شد.','updated_at'=>$now],['id'=>$case_id]);$this->action_redirect('شارژ در مقصد ثبت شد اما کد فعال‌سازی برنگشت؛ از به‌روزرسانی گزارش استفاده کنید.',false);}
			$customer_url=$this->validated_https_url((string)($result['customer_url']??''));
			$data=['wallet_external_id'=>$external_id,'wallet_activation_code'=>$code,'wallet_status'=>'charged','wallet_usage_json'=>wp_json_encode((array)($result['usage']??[]),JSON_UNESCAPED_UNICODE|JSON_UNESCAPED_SLASHES),'wallet_customer_url'=>$customer_url,'status'=>'wallet_charged','last_contact_at'=>$now,'updated_at'=>$now];
			$link_line=$customer_url!==''?"\nلینک استفاده: {$customer_url}":'';$wallet_message="اعتبار کارت «{$ctx->content_name_snapshot}» شارژ شد.\nکد فعال‌سازی: {$code}{$link_line}";$sms=(new SN_SMS())->send((string)$ctx->customer_phone,$wallet_message);if(class_exists('SN_Notification_Service')){SN_Notification_Service::instance()->mirror_text('operations.wallet_charged',(string)$ctx->customer_phone,$wallet_message,[ 'secure'=>true,'sensitive'=>true,'entity_type'=>'operation','entity_id'=>(int)$case->operation_id,'operation_id'=>(int)$case->operation_id,'dedupe_key'=>'wallet-charged-'.(int)$case->operation_id.'-'.hash('sha256',(string)$code),'source'=>'operations_execution','buttons'=>$customer_url!==''?[[['text'=>'مشاهده کارت','url'=>esc_url_raw($customer_url)]]]:[] ]);}if($sms){$data['wallet_sms_sent_at']=$now;}
			$wpdb->update($t['cases'],$data,['id'=>$case_id]);$wpdb->update($wpdb->prefix.'sn_project_operations',['stage'=>'execution_wallet_active','updated_at'=>$now],['id'=>(int)$case->operation_id]);$this->project_event((int)$case->membership_item_id,'operations_wallet_charged',['sms_sent'=>$sms?1:0,'credit'=>(float)$ctx->current_credit],$actor);$this->action_redirect($sms?'کیف پول شارژ و کد برای مشتری ارسال شد.':'کیف پول شارژ شد؛ ارسال پیامک ناموفق بود.',true);
		}
		if($action==='wallet_refresh'){
			if((string)$case->fulfillment_type!=='wallet_charge'||((string)$case->wallet_activation_code===''&&(string)$case->wallet_status!=='sync_pending')){$this->action_redirect('هنوز شارژ قابل استعلامی برای این کارت ثبت نشده است.',false);}
			$result=$this->wallet_request($case,$ctx,'status');if(empty($result['success'])){$this->action_redirect((string)($result['message']??'دریافت گزارش استفاده ناموفق بود.'),false);}
			$data=['wallet_status'=>sanitize_key((string)($result['status']??$case->wallet_status)),'wallet_usage_json'=>wp_json_encode((array)($result['usage']??[]),JSON_UNESCAPED_UNICODE|JSON_UNESCAPED_SLASHES),'updated_at'=>$now];
			if(!empty($result['activation_code'])){$data['wallet_activation_code']=sanitize_text_field((string)$result['activation_code']);$data['wallet_status']='charged';$data['status']='wallet_charged';}
			if(!empty($result['customer_url'])){$data['wallet_customer_url']=$this->validated_https_url((string)$result['customer_url']);}
			$wpdb->update($t['cases'],$data,['id'=>$case_id]);$this->action_redirect('گزارش استفاده به‌روز شد.',true);
		}
		if($action==='wallet_assisted_login'){if((string)$case->fulfillment_type!=='wallet_charge'||(string)$case->wallet_activation_code===''){$this->action_redirect('ابتدا شارژ کیف پول باید با موفقیت تکمیل شود.',false);}$result=$this->wallet_request($case,$ctx,'assisted_login');$url=$this->validated_https_url((string)($result['url']??''));if(empty($result['success'])||$url===''){$this->action_redirect('سایت مقصد لینک ورود امن یک‌بارمصرف معتبر برنگرداند.',false);}wp_redirect($url);exit;}
		if($action==='physical_confirm'){if((string)$case->fulfillment_type!=='physical_invoice'||!in_array((string)$case->status,['assigned','no_answer','follow_up'],true)){$this->action_redirect('این کارت قبلاً تعیین تکلیف شده است.',false);}$wpdb->update($t['cases'],['status'=>'shipping_queue','shipping_status'=>trim((string)$case->shipping_address)!==''?'awaiting_review':'awaiting_address','last_contact_at'=>$now,'updated_at'=>$now],['id'=>$case_id]);$wpdb->update($wpdb->prefix.'sn_project_operations',['stage'=>'shipping_queue','updated_at'=>$now],['id'=>(int)$case->operation_id]);$this->project_event((int)$case->membership_item_id,'operations_physical_sent_to_shipping',['card'=>(string)$ctx->content_name_snapshot],$actor);$this->action_redirect('کارت وارد پنل ارسال شد.',true);}
		if($action==='form_save'){if((string)$case->fulfillment_type!=='form'||!in_array((string)$case->status,['assigned','no_answer','follow_up','form_completed'],true)){$this->action_redirect('امکان ثبت فرم برای این کارت وجود ندارد.',false);}$result=$this->save_submission($case,$actor,(array)wp_unslash($_POST['fields']??[]),sanitize_textarea_field(wp_unslash($_POST['note']??'')));if(is_wp_error($result)){$this->action_redirect($result->get_error_message(),false);}$wpdb->update($wpdb->prefix.'sn_project_operations',['stage'=>'execution_form_review','updated_at'=>$now],['id'=>(int)$case->operation_id]);$this->project_event((int)$case->membership_item_id,'operations_form_completed',['submission_id'=>(int)$result],$actor);$this->action_redirect('فرم ذخیره شد؛ اکنون اجرای نهایی یا عدم اجرا را ثبت کنید.',true);}
		if(in_array($action,['form_execute','form_skip'],true)){if((string)$case->fulfillment_type!=='form'||(string)$case->status!=='form_completed'){$this->action_redirect('ابتدا فرم را کامل و ذخیره کنید.',false);}$submission=$wpdb->get_row($wpdb->prepare("SELECT * FROM {$t['submissions']} WHERE execution_case_id=%d LIMIT 1",$case_id));if(!$submission){$this->action_redirect('ابتدا فرم را کامل و ذخیره کنید.',false);}$decision=$action==='form_execute'?'executed':'not_executed';$note=sanitize_textarea_field(wp_unslash($_POST['note']??''));if($decision==='not_executed'&&trim($note)===''){$this->action_redirect('دلیل عدم اجرا اجباری است.',false);}$wpdb->update($t['submissions'],['final_decision'=>$decision,'final_note'=>$note,'decided_by_user_id'=>$actor,'decided_at'=>$now,'updated_at'=>$now],['id'=>(int)$submission->id]);$wpdb->update($t['cases'],['status'=>$decision==='executed'?'completed':'not_executed','final_decision'=>$decision,'completed_at'=>$now,'completed_by_user_id'=>$actor,'updated_at'=>$now],['id'=>$case_id]);$wpdb->update($wpdb->prefix.'sn_project_operations',['stage'=>$decision==='executed'?'completed':'execution_not_executed','completed_at'=>$now,'completed_by_user_id'=>$actor,'updated_at'=>$now],['id'=>(int)$case->operation_id]);$this->project_event((int)$case->membership_item_id,$decision==='executed'?'operations_form_executed':'operations_form_not_executed',['note'=>$note],$actor);$this->action_redirect($decision==='executed'?'اجرای نهایی ثبت شد.':'عدم اجرا ثبت شد.',true);}
		wp_die('عملیات نامعتبر است.');
	}

	private function validated_https_url( string $url ): string {
		$url = esc_url_raw( $url );
		return $url !== '' && wp_http_validate_url( $url ) && strtolower( (string) wp_parse_url( $url, PHP_URL_SCHEME ) ) === 'https' ? $url : '';
	}

	private function wallet_payload( object $case, object $ctx, string $action ): array {
		$config=json_decode((string)$case->config_snapshot_json,true);$config=is_array($config)?$config:[];
		$wallet_type=sanitize_key((string)($config['wallet_type']??'cash'));if(!in_array($wallet_type,['cash','installment'],true)){$wallet_type='cash';}
		return ['action'=>$action,'request_id'=>'sn-execution-'.(int)$case->id,'operation_id'=>(int)$case->operation_id,'customer'=>['wp_id'=>(int)$ctx->customer_wp_id,'name'=>(string)$ctx->customer_name,'phone'=>SN_Helpers::normalize_mobile((string)$ctx->customer_phone)],'card'=>['product_id'=>(int)$ctx->content_product_id,'name'=>(string)$ctx->content_name_snapshot,'credit'=>(float)$ctx->current_credit,'mode'=>(string)$ctx->activation_mode],'wallet_type'=>$wallet_type,'service_key'=>sanitize_key((string)($config['service_key']??'')),'external_id'=>(string)$case->wallet_external_id];
	}

	public function handle_manual_export(): void {
		if(!is_user_logged_in()){auth_redirect();}
		$case_id=absint($_POST['case_id']??0);check_admin_referer('sn_operations_execution_case_'.$case_id);$actor=get_current_user_id();global $wpdb;$t=$this->tables();
		$case=$wpdb->get_row($wpdb->prepare("SELECT * FROM {$t['cases']} WHERE id=%d LIMIT 1",$case_id));
		if(!$case||!$this->can_work_case($case,$actor)){wp_die('این کارت به شما تخصیص ندارد.','',[ 'response'=>403 ]);}
		if((string)$case->fulfillment_type!=='wallet_charge'){wp_die('خروجی دستی فقط برای مرحله اتصال کیف پول/API در دسترس است.','',[ 'response'=>400 ]);}
		$config_issue=$this->configuration_issue($case);if($config_issue!==''){wp_die(esc_html($config_issue),'',[ 'response'=>400 ]);}
		$ctx=$this->operation_context((int)$case->operation_id);if(!$ctx){wp_die('اطلاعات کارت پیدا نشد.','',[ 'response'=>404 ]);}
		$payload=$this->wallet_payload($case,$ctx,'charge');
		$this->log_event($case_id,'execution_manual_api_export',['request_id'=>(string)$payload['request_id']],$actor);
		$filename='sn-manual-api-case-'.$case_id.'-'.SN_Helpers::tehran_format('Ymd-His').'.csv';
		nocache_headers();header('Content-Type: text/csv; charset=UTF-8');header('Content-Disposition: attachment; filename="'.sanitize_file_name($filename).'"');
		$out=fopen('php://output','w');if(false===$out){exit;}fwrite($out,"\xEF\xBB\xBF");
		fputcsv($out,['request_id','operation_id','execution_case_id','customer_wp_id','customer_name','customer_phone','product_id','card_name','credit','activation_mode','wallet_type','service_key','external_id']);
		fputcsv($out,[(string)$payload['request_id'],(int)$payload['operation_id'],$case_id,(int)$payload['customer']['wp_id'],(string)$payload['customer']['name'],(string)$payload['customer']['phone'],(int)$payload['card']['product_id'],(string)$payload['card']['name'],(float)$payload['card']['credit'],(string)$payload['card']['mode'],(string)$payload['wallet_type'],(string)$payload['service_key'],(string)$payload['external_id']]);
		fclose($out);exit;
	}

	private function wallet_request( object $case, object $ctx, string $action ): array {
		$payload=$this->wallet_payload($case,$ctx,$action);
		if((string)$payload['service_key']===''){return['success'=>false,'message'=>'شناسه خدمت کیف پول مقصد برای این محصول* تعریف نشده است.'];}
		if(!in_array((string)$payload['wallet_type'],['cash','installment'],true)){return['success'=>false,'message'=>'نوع کیف پول مقصد معتبر نیست.'];}
		$filtered=apply_filters('sn_operations_wallet_request',null,$payload,$case,$ctx);if(is_array($filtered)){return$filtered;}
		$endpoint=$this->validated_https_url((string)get_option('sn_operations_wallet_endpoint',''));if($endpoint===''){return['success'=>false,'message'=>'اتصال HTTPS افزونه کیف پول هنوز تنظیم نشده است.'];}$token=trim((string)get_option('sn_operations_wallet_bearer',''));$response=wp_remote_post($endpoint,['timeout'=>20,'sslverify'=>true,'redirection'=>0,'headers'=>array_filter(['Accept'=>'application/json','Content-Type'=>'application/json','Authorization'=>$token!==''?'Bearer '.$token:'']),'body'=>wp_json_encode($payload,JSON_UNESCAPED_UNICODE|JSON_UNESCAPED_SLASHES)]);if(is_wp_error($response)){return['success'=>false,'message'=>'ارتباط با افزونه کیف پول برقرار نشد.'];}$code=(int)wp_remote_retrieve_response_code($response);$body=json_decode((string)wp_remote_retrieve_body($response),true);if(!is_array($body)||$code<200||$code>=300||empty($body['success'])){return['success'=>false,'message'=>sanitize_text_field((string)($body['message']??'پاسخ سرویس کیف پول نامعتبر است.'))];}return$body;
	}

	private function save_submission( object $case, int $actor, array $input, string $note ) {
		$schema=json_decode((string)$case->form_schema_snapshot_json,true);if(!is_array($schema)||!$schema){return new WP_Error('schema','فرم این کارت تعریف نشده است.');}$values=[];foreach($schema as $field){$key=sanitize_key((string)($field['key']??''));if($key===''){continue;}$raw=$input[$key]??'';$type=sanitize_key((string)($field['type']??'text'));if($type==='checkbox'){$value=!empty($raw)?1:0;}elseif($type==='textarea'){$value=sanitize_textarea_field($raw);}elseif($type==='number'){$value=is_numeric(SN_Helpers::to_english_nums((string)$raw))?(float)SN_Helpers::to_english_nums((string)$raw):'';}elseif($type==='date'){$value=str_replace(['-','.'],'/',trim(SN_Helpers::to_english_nums((string)$raw)));if($value!==''&&(!preg_match('/^(\\d{4})\\/(\\d{1,2})\\/(\\d{1,2})$/',$value,$date_match)||SN_Helpers::gregorian_to_jalali_input_value(SN_Helpers::jalali_to_gregorian_date(sprintf('%04d/%02d/%02d',(int)($date_match[1]??0),(int)($date_match[2]??0),(int)($date_match[3]??0))))!==sprintf('%04d/%02d/%02d',(int)($date_match[1]??0),(int)($date_match[2]??0),(int)($date_match[3]??0)))){return new WP_Error('date','تاریخ واردشده در فرم معتبر نیست.');}if($value!==''){$value=sprintf('%04d/%02d/%02d',(int)$date_match[1],(int)$date_match[2],(int)$date_match[3]);}}else{$value=sanitize_text_field($raw);}if(!empty($field['required'])&&($value===''||$value===0)) {return new WP_Error('required','تکمیل فیلد «'.sanitize_text_field((string)$field['label']).'» اجباری است.');}if(in_array($type,['select','radio'],true)&&$value!==''&&!in_array($value,(array)($field['options']??[]),true)){return new WP_Error('option','مقدار انتخابی فرم معتبر نیست.');}$values[$key]=$value;}
		global $wpdb;$t=$this->tables();$now=current_time('mysql');$existing=(int)$wpdb->get_var($wpdb->prepare("SELECT id FROM {$t['submissions']} WHERE execution_case_id=%d LIMIT 1",(int)$case->id));$data=['form_definition_id'=>(int)$case->form_definition_id?:null,'schema_snapshot_json'=>(string)$case->form_schema_snapshot_json,'values_json'=>wp_json_encode($values,JSON_UNESCAPED_UNICODE|JSON_UNESCAPED_SLASHES),'submitted_by_user_id'=>$actor,'submitted_at'=>$now,'final_note'=>$note,'updated_at'=>$now];if($existing){$ok=$wpdb->update($t['submissions'],$data,['id'=>$existing]);$id=$existing;}else{$data['execution_case_id']=(int)$case->id;$data['created_at']=$now;$ok=$wpdb->insert($t['submissions'],$data);$id=(int)$wpdb->insert_id;}if(false===$ok){return new WP_Error('db','ذخیره فرم انجام نشد.');}$wpdb->update($t['cases'],['form_submission_id'=>$id,'status'=>'form_completed','last_note'=>$note,'updated_at'=>$now],['id'=>(int)$case->id]);return$id;
	}

	private function log_event( int $case_id, string $key, array $details, int $actor ): void { global $wpdb;$t=$this->tables();$case=$wpdb->get_row($wpdb->prepare("SELECT operation_id,membership_item_id FROM {$t['cases']} WHERE id=%d",$case_id));if(!$case){return;}$wpdb->insert($t['events'],['execution_case_id'=>$case_id,'operation_id'=>(int)$case->operation_id,'membership_item_id'=>(int)$case->membership_item_id,'actor_user_id'=>$actor?:null,'event_key'=>sanitize_key($key),'details_json'=>wp_json_encode($details,JSON_UNESCAPED_UNICODE|JSON_UNESCAPED_SLASHES),'created_at'=>current_time('mysql')]); }

	private function project_event( int $item_id, string $key, array $details, int $actor ): void { global $wpdb;$items=$wpdb->prefix.'sn_project_membership_items';$events=$wpdb->prefix.'sn_project_events';if(!$this->table_exists($events)){return;}$membership_id=(int)$wpdb->get_var($wpdb->prepare("SELECT membership_id FROM {$items} WHERE id=%d",$item_id));$wpdb->insert($events,['membership_id'=>$membership_id,'membership_item_id'=>$item_id,'actor_user_id'=>$actor?:null,'event_key'=>sanitize_key($key),'details'=>wp_json_encode($details,JSON_UNESCAPED_UNICODE|JSON_UNESCAPED_SLASHES),'created_at'=>current_time('mysql')]); }

	private function case_status_label( string $status ): string { return['manager_queue'=>'در صف مدیر اجرایی','assigned'=>'تخصیص به کارشناس','no_answer'=>'جواب نداده','follow_up'=>'تماس مجدد','wallet_charged'=>'کیف پول شارژ شد','shipping_queue'=>'در صف ارسال','shipping_processing'=>'در حال آماده‌سازی ارسال','shipped'=>'ارسال شد','delivered'=>'تحویل شد','shipping_not_sent'=>'ارسال نشد','shipping_returned'=>'مرجوع شد','shipping_cancelled'=>'لغو ارسال','form_completed'=>'فرم تکمیل شد','completed'=>'اجرای نهایی','not_executed'=>'عدم اجرا','cancelled'=>'انصراف'][$status]??$status; }

	private function panel_rows( int $user_id ): array { global $wpdb;$t=$this->tables();$operations=$wpdb->prefix.'sn_project_operations';$items=$wpdb->prefix.'sn_project_membership_items';$members=$wpdb->prefix.'sn_project_memberships';$users=$wpdb->users;$where=current_user_can('manage_options')?'1=1':$wpdb->prepare('c.expert_user_id=%d',$user_id);return$wpdb->get_results("SELECT c.*,o.current_credit,o.base_credit,o.activation_mode,o.stage,mi.content_name_snapshot,m.subscription_name_snapshot,m.customer_name,m.customer_phone,u.display_name expert_name FROM {$t['cases']} c INNER JOIN {$operations} o ON o.id=c.operation_id INNER JOIN {$items} mi ON mi.id=c.membership_item_id INNER JOIN {$members} m ON m.id=mi.membership_id LEFT JOIN {$users} u ON u.ID=c.expert_user_id WHERE {$where} ORDER BY FIELD(c.status,'assigned','follow_up','form_completed','wallet_charged','shipping_queue','completed','not_executed','cancelled'),c.updated_at DESC,c.id DESC LIMIT 500")?:[]; }

	public function render_expert_panel(): string {
		if(!is_user_logged_in()){return'<div class="sn-ops-panel">ابتدا وارد شوید.</div>';}$uid=get_current_user_id();if(!(current_user_can('manage_options')||current_user_can('sn_work_operations_execution')||$this->position_for_user($uid)==='operations_execution_expert')){return'<div class="sn-ops-panel">دسترسی غیرمجاز است.</div>';}$rows=$this->panel_rows($uid);$stages=[];$types=[];foreach($rows as$row){$stages[$row->status]=$this->case_status_label((string)$row->status);$types[$row->fulfillment_type]=$this->fulfillment_label((string)$row->fulfillment_type);}ob_start();?><div class="sn-ops-panel sn-execution-expert-panel" dir="rtl"><header class="sn-ops-hero"><div class="sn-ops-hero-copy"><span class="sn-ops-eyebrow">چرخه بیاوین</span><h2>کارشناس اجرایی عملیات</h2><p>اجرای کارت‌های شارژ کیف پول، فاکتور کالا و فرم</p></div><div class="sn-ops-hero-actions"><div class="sn-ops-hero-count" aria-label="تعداد کارت‌های این نما"><strong><?php echo esc_html(number_format_i18n(count($rows)));?></strong><span>کارت</span></div><?php if(class_exists('SN_Purpose_Commission')){echo SN_Purpose_Commission::render_wallet_launcher($uid,'biavin',20);}//phpcs:ignore?><a class="sn-ops-logout" href="<?php echo esc_url(wp_logout_url(sn_crm_login_url()));?>"><span aria-hidden="true">↪</span><strong>خروج</strong></a></div></header><?php echo class_exists('SN_Operations_Flow')?SN_Operations_Flow::instance()->public_notice_html():'';//phpcs:ignore?><div class="sn-ops-toolbar"><label><span>جستجو</span><input type="search" data-sn-ops-search placeholder="نام، موبایل یا کارت…"></label><label><span>وضعیت</span><select data-sn-ops-status><option value="">همه</option><?php foreach($stages as$key=>$label):?><option value="<?php echo esc_attr($key);?>"><?php echo esc_html($label);?></option><?php endforeach;?></select></label><label><span>نوع کارت</span><select data-sn-ops-card-filter><option value="">همه</option><?php foreach($types as$key=>$label):?><option value="<?php echo esc_attr($key);?>"><?php echo esc_html($label);?></option><?php endforeach;?></select></label></div><div class="sn-ops-list" data-sn-ops-list><?php if(!$rows):?><div class="sn-ops-empty">کارتی به شما تخصیص داده نشده است.</div><?php endif;?><?php foreach($rows as$row):$search=implode(' ',[$row->customer_name,$row->customer_phone,$row->content_name_snapshot,$row->subscription_name_snapshot]);?><article class="sn-ops-case" data-sn-ops-card data-search="<?php echo esc_attr($search);?>" data-status="<?php echo esc_attr((string)$row->status);?>" data-card-type="<?php echo esc_attr((string)$row->fulfillment_type);?>"><div class="sn-ops-case-head"><div class="sn-ops-case-summary-copy"><small class="sn-ops-case-reference">#<?php echo esc_html((string)$row->id);?> · <?php echo esc_html((string)$row->subscription_name_snapshot);?></small><h3><span>کارت</span><?php echo esc_html((string)$row->content_name_snapshot);?></h3><div class="sn-ops-customer-identity"><span><small>مشتری</small><strong><?php echo esc_html((string)$row->customer_name);?></strong></span><span><small>شماره تماس</small><a dir="ltr" href="tel:<?php echo esc_attr((string)$row->customer_phone);?>"><?php echo esc_html((string)$row->customer_phone);?></a></span><?php if((float)$row->current_credit>0):?><span class="sn-ops-customer-credit"><small>اعتبار</small><strong><?php echo esc_html(SN_Helpers::format_price((float)$row->current_credit));?></strong></span><?php endif;?></div></div><div class="sn-ops-badges"><b><?php echo esc_html($this->case_status_label((string)$row->status));?></b><em class="is-<?php echo esc_attr((string)$row->activation_mode);?>"><?php echo esc_html((string)$row->activation_mode==='upsell'?'افزایشی':'عادی');?></em><em><?php echo esc_html($this->fulfillment_label((string)$row->fulfillment_type));?></em></div></div><div class="sn-ops-credit"><?php if((float)$row->current_credit>0):?><span>اعتبار کارت <strong><?php echo esc_html(SN_Helpers::format_price((float)$row->current_credit));?></strong></span><?php endif;?><span>جواب نداده <strong><?php echo esc_html(number_format_i18n((int)$row->no_answer_count));?> بار</strong></span></div><?php if($row->follow_up_at):?><div class="sn-ops-followup-current">تماس مجدد: <?php echo esc_html(SN_Helpers::gregorian_to_jalali_date((string)$row->follow_up_at));?></div><?php endif;?><?php $this->render_expert_actions($row);?></article><?php endforeach;?><div class="sn-ops-empty" data-sn-ops-no-results hidden>نتیجه‌ای پیدا نشد.</div></div></div><?php return(string)ob_get_clean();
	}

	private function render_expert_actions( object $row ): void { $nonce=wp_nonce_field('sn_operations_execution_case_'.(int)$row->id,'_wpnonce',true,false);$url=esc_url(admin_url('admin-post.php'));$contactable=in_array((string)$row->status,['assigned','no_answer','follow_up'],true);if(in_array((string)$row->status,['completed','not_executed','cancelled','shipping_not_sent','shipping_returned','shipping_cancelled','delivered'],true)){return;}?>
		<?php if($contactable):?><form method="post" action="<?php echo $url;?>" class="sn-ops-form sn-ops-case-status-form" data-sn-ops-case-status><input type="hidden" name="action" value="sn_operations_execution_action"><input type="hidden" name="case_id" value="<?php echo esc_attr((string)$row->id);?>"><?php echo $nonce;//phpcs:ignore?><label class="sn-ops-status-main"><span>تعیین وضعیت</span><select name="execution_action" data-sn-ops-status-select required><option value="">— انتخاب —</option><option value="no_answer">جواب نداده</option><option value="follow_up">تماس مجدد</option><option value="cancel">انصراف</option></select></label><div class="sn-ops-status-fields" data-sn-ops-status-fields="no_answer" hidden><textarea name="note" placeholder="یادداشت اختیاری"></textarea></div><div class="sn-ops-status-fields" data-sn-ops-status-fields="follow_up" hidden><input class="sn-ops-jalali-date" name="followup_date" data-sn-ops-required placeholder="۱۴۰۵/۰۶/۲۳"><textarea name="note" placeholder="یادداشت"></textarea></div><div class="sn-ops-status-fields" data-sn-ops-status-fields="cancel" hidden><textarea name="cancel_reason" placeholder="دلیل انصراف (اجباری)" minlength="3" data-sn-ops-required></textarea></div><button data-sn-ops-status-submit disabled>ثبت وضعیت</button></form><?php endif;?>
		<div class="sn-execution-primary-actions"><?php if((string)$row->fulfillment_type==='wallet_charge'):?><form method="post" action="<?php echo $url;?>" class="sn-ops-form is-inline"><input type="hidden" name="action" value="sn_operations_execution_export"><input type="hidden" name="case_id" value="<?php echo esc_attr((string)$row->id);?>"><?php echo $nonce;//phpcs:ignore?><button type="submit" class="is-secondary">خروجی CSV برای ثبت دستی در سیستم مقصد</button></form><?php if((string)$row->wallet_status==='sync_pending'):?><div class="sn-ops-warning">شارژ در مقصد ثبت شده اما دریافت کد فعال‌سازی نیازمند همگام‌سازی است.</div><form method="post" action="<?php echo $url;?>" class="sn-ops-form is-inline"><input type="hidden" name="action" value="sn_operations_execution_action"><input type="hidden" name="execution_action" value="wallet_refresh"><input type="hidden" name="case_id" value="<?php echo esc_attr((string)$row->id);?>"><?php echo $nonce;//phpcs:ignore?><button>همگام‌سازی کد و گزارش</button></form><?php elseif((string)$row->wallet_status!=='charged'&&$contactable):?><form method="post" action="<?php echo $url;?>" class="sn-ops-form is-inline"><input type="hidden" name="action" value="sn_operations_execution_action"><input type="hidden" name="execution_action" value="wallet_charge"><input type="hidden" name="case_id" value="<?php echo esc_attr((string)$row->id);?>"><?php echo $nonce;//phpcs:ignore?><button>تأیید و شارژ کیف پول</button></form><?php else:?><div class="sn-exec-code">کد فعال‌سازی: <strong><?php echo esc_html((string)$row->wallet_activation_code);?></strong></div><form method="post" action="<?php echo $url;?>" class="sn-ops-form is-inline"><input type="hidden" name="action" value="sn_operations_execution_action"><input type="hidden" name="execution_action" value="wallet_refresh"><input type="hidden" name="case_id" value="<?php echo esc_attr((string)$row->id);?>"><?php echo $nonce;//phpcs:ignore?><button>به‌روزرسانی گزارش استفاده</button></form><form method="post" action="<?php echo $url;?>" class="sn-ops-form is-inline" target="_blank"><input type="hidden" name="action" value="sn_operations_execution_action"><input type="hidden" name="execution_action" value="wallet_assisted_login"><input type="hidden" name="case_id" value="<?php echo esc_attr((string)$row->id);?>"><?php echo $nonce;//phpcs:ignore?><button class="is-secondary">ورود امن یک‌بارمصرف به حساب مشتری</button></form><?php if($row->wallet_usage_json):?><pre class="sn-exec-usage"><?php echo esc_html(wp_json_encode(json_decode((string)$row->wallet_usage_json,true),JSON_PRETTY_PRINT|JSON_UNESCAPED_UNICODE));?></pre><?php endif;?><?php endif;?><?php elseif((string)$row->fulfillment_type==='physical_invoice'):?><?php if($contactable):?><form method="post" action="<?php echo $url;?>" class="sn-ops-form is-inline"><input type="hidden" name="action" value="sn_operations_execution_action"><input type="hidden" name="execution_action" value="physical_confirm"><input type="hidden" name="case_id" value="<?php echo esc_attr((string)$row->id);?>"><?php echo $nonce;//phpcs:ignore?><button>تأیید و ارسال به پنل ارسال</button></form><?php endif;?><?php elseif((string)$row->fulfillment_type==='form'):?><?php if($contactable||(string)$row->status==='form_completed'){$this->render_case_form($row,$url,$nonce);}?><?php endif;?></div>
		<?php }

	private function render_case_form( object $row, string $url, string $nonce ): void { $schema=json_decode((string)$row->form_schema_snapshot_json,true);$schema=is_array($schema)?$schema:[];global$wpdb;$submissions_table=$this->tables()['submissions'];$submission=$wpdb->get_row($wpdb->prepare("SELECT * FROM {$submissions_table} WHERE execution_case_id=%d LIMIT 1",(int)$row->id));$values=$submission?json_decode((string)$submission->values_json,true):[];$values=is_array($values)?$values:[];?><details class="sn-exec-form" <?php echo !$submission?'open':'';?>><summary><?php echo $submission?'ویرایش فرم تکمیل‌شده':'تکمیل فرم کارت';?></summary><form method="post" action="<?php echo $url;?>" class="sn-ops-form sn-exec-dynamic-form"><input type="hidden" name="action" value="sn_operations_execution_action"><input type="hidden" name="execution_action" value="form_save"><input type="hidden" name="case_id" value="<?php echo esc_attr((string)$row->id);?>"><?php echo $nonce;//phpcs:ignore?><?php foreach($schema as$field):$key=sanitize_key((string)($field['key']??''));$type=sanitize_key((string)($field['type']??'text'));$value=$values[$key]??'';?><label class="<?php echo $type==='checkbox'?'sn-exec-checkbox-field':'';?>"><span><?php echo esc_html((string)($field['label']??$key));?><?php echo !empty($field['required'])?' *':'';?></span><?php if($type==='textarea'):?><textarea name="fields[<?php echo esc_attr($key);?>]" <?php echo !empty($field['required'])?'required':'';?>><?php echo esc_textarea((string)$value);?></textarea><?php elseif(in_array($type,['select','radio'],true)):?><select name="fields[<?php echo esc_attr($key);?>]" <?php echo !empty($field['required'])?'required':'';?>><option value="">— انتخاب —</option><?php foreach((array)($field['options']??[]) as$option):?><option value="<?php echo esc_attr((string)$option);?>" <?php selected((string)$value,(string)$option);?>><?php echo esc_html((string)$option);?></option><?php endforeach;?></select><?php elseif($type==='checkbox'):?><input type="hidden" name="fields[<?php echo esc_attr($key);?>]" value="0"><input type="checkbox" name="fields[<?php echo esc_attr($key);?>]" value="1" <?php checked((string)$value==='1'||$value===1||$value===true);?> <?php echo !empty($field['required'])?'required':'';?>><?php elseif($type==='date'):?><input type="text" class="sn-ops-jalali-date" data-sn-ops-date-any name="fields[<?php echo esc_attr($key);?>]" value="<?php echo esc_attr((string)$value);?>" placeholder="۱۴۰۵/۰۶/۲۴" pattern="[۰-۹0-9]{4}[\\/-][۰-۹0-9]{1,2}[\\/-][۰-۹0-9]{1,2}" autocomplete="off" <?php echo !empty($field['required'])?'required':'';?>><?php else:?><input type="<?php echo esc_attr($type==='number'?'number':'text');?>" name="fields[<?php echo esc_attr($key);?>]" value="<?php echo esc_attr((string)$value);?>" <?php echo !empty($field['required'])?'required':'';?>><?php endif;?></label><?php endforeach;?><textarea name="note" placeholder="یادداشت فرم"></textarea><button>ذخیره فرم</button></form></details><?php if($submission):?><div class="sn-exec-final"><form method="post" action="<?php echo $url;?>" class="sn-ops-form is-inline"><input type="hidden" name="action" value="sn_operations_execution_action"><input type="hidden" name="execution_action" value="form_execute"><input type="hidden" name="case_id" value="<?php echo esc_attr((string)$row->id);?>"><?php echo $nonce;//phpcs:ignore?><button>اجرای نهایی</button></form><form method="post" action="<?php echo $url;?>" class="sn-ops-form"><input type="hidden" name="action" value="sn_operations_execution_action"><input type="hidden" name="execution_action" value="form_skip"><input type="hidden" name="case_id" value="<?php echo esc_attr((string)$row->id);?>"><?php echo $nonce;//phpcs:ignore?><textarea name="note" required placeholder="دلیل عدم اجرا"></textarea><button class="is-danger">عدم اجرا</button></form></div><?php endif;?><?php }

	public function render_customer_case_history( int $item_id ): string {
		if ( ! is_user_logged_in() || $item_id < 1 ) { return ''; }
		global $wpdb; $t=$this->tables();$items=$wpdb->prefix.'sn_project_membership_items';$members=$wpdb->prefix.'sn_project_memberships';
		$owner_id=(int)$wpdb->get_var($wpdb->prepare("SELECT m.customer_wp_id FROM {$items} mi INNER JOIN {$members} m ON m.id=mi.membership_id WHERE mi.id=%d LIMIT 1",$item_id));
		if ( ! current_user_can('manage_options') && $owner_id !== get_current_user_id() ) { return ''; }
		$case=$wpdb->get_row($wpdb->prepare("SELECT c.*,o.current_credit,o.activation_mode FROM {$t['cases']} c INNER JOIN {$wpdb->prefix}sn_project_operations o ON o.id=c.operation_id WHERE c.membership_item_id=%d LIMIT 1",$item_id));
		if(!$case){return'';}
		$event_labels=['operations_execution_assigned'=>'ارجاع به کارشناس اجرایی','operations_execution_no_answer'=>'تماس بی‌پاسخ','operations_execution_follow_up'=>'ثبت تماس مجدد','operations_execution_cancelled'=>'ثبت انصراف','operations_wallet_charged'=>'شارژ کیف پول','operations_physical_sent_to_shipping'=>'ارسال به واحد ارسال','operations_form_completed'=>'تکمیل فرم','operations_form_executed'=>'اجرای نهایی فرم','operations_form_not_executed'=>'عدم اجرای فرم','operations_shipping_status_changed'=>'به‌روزرسانی وضعیت ارسال'];
		$events_table=$wpdb->prefix.'sn_project_events';$events=[];
		if($this->table_exists($events_table)){$events=$wpdb->get_results($wpdb->prepare("SELECT event_key,created_at FROM {$events_table} WHERE membership_item_id=%d AND event_key LIKE 'operations_%%' ORDER BY id DESC LIMIT 20",$item_id))?:[];}
		ob_start();?><div class="sn-customer-execution-track"><div class="sn-customer-card-dialog-section"><span>پیگیری اجرای کارت</span></div><div class="sn-customer-execution-current"><span><small>نوع خدمت</small><strong><?php echo esc_html($this->fulfillment_label((string)$case->fulfillment_type));?></strong></span><span><small>وضعیت فعلی</small><strong><?php echo esc_html($this->case_status_label((string)$case->status));?></strong></span><?php if((float)$case->current_credit>0):?><span><small>اعتبار</small><strong><?php echo esc_html(SN_Helpers::format_price((float)$case->current_credit));?></strong></span><?php endif;?></div><?php if($case->follow_up_at):?><p class="sn-customer-execution-note">تاریخ تماس مجدد: <strong><?php echo esc_html(SN_Helpers::gregorian_to_jalali_date((string)$case->follow_up_at));?></strong></p><?php endif;?><?php if($case->wallet_activation_code):?><p class="sn-customer-execution-note">کد فعال‌سازی: <strong><?php echo esc_html((string)$case->wallet_activation_code);?></strong></p><?php endif;?><?php if($case->shipping_tracking_code):?><p class="sn-customer-execution-note">کد رهگیری ارسال: <strong><bdi><?php echo esc_html((string)$case->shipping_tracking_code);?></bdi></strong></p><?php endif;?><?php if($events):?><ol class="sn-customer-execution-events"><?php foreach($events as$event):$key=sanitize_key((string)$event->event_key);if(!isset($event_labels[$key])){continue;}?><li><span><?php echo esc_html($event_labels[$key]);?></span><time><?php echo esc_html(SN_Helpers::gregorian_to_jalali_date((string)$event->created_at));?></time></li><?php endforeach;?></ol><?php endif;?></div><?php return(string)ob_get_clean();
	}

	private function shipping_statuses(): array {
		return [ 'awaiting_address'=>'نیازمند تکمیل آدرس', 'awaiting_review'=>'در انتظار بررسی', 'preparing'=>'آماده‌سازی', 'shipped'=>'ارسال شد', 'delivered'=>'تحویل شد', 'not_sent'=>'ارسال نشد', 'returned'=>'مرجوع شد', 'cancelled'=>'لغو شد' ];
	}

	public function shipping_counts(): array {
		if ( ! ( current_user_can( 'manage_options' ) || current_user_can( 'sn_manage_shipping' ) ) ) { return []; }
		global $wpdb;$table=$this->tables()['cases'];$counts=[];
		foreach($wpdb->get_results("SELECT shipping_status,COUNT(*) total FROM {$table} WHERE fulfillment_type='physical_invoice' AND shipping_status IS NOT NULL GROUP BY shipping_status")?:[] as$row){$counts[(string)$row->shipping_status]=(int)$row->total;}
		return$counts;
	}

	public function handle_shipping_update(): void {
		if ( ! is_user_logged_in() ) { auth_redirect(); }
		if ( ! ( current_user_can( 'manage_options' ) || current_user_can( 'sn_manage_shipping' ) ) ) { wp_die( 'دسترسی غیرمجاز', '', [ 'response'=>403 ] ); }
		$case_id = absint( $_POST['case_id'] ?? 0 );
		check_admin_referer( 'sn_operations_execution_shipping_' . $case_id );
		global $wpdb; $t = $this->tables();
		$case = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['cases']} WHERE id=%d AND fulfillment_type='physical_invoice' LIMIT 1", $case_id ) );
		if ( ! $case || empty( $case->shipping_status ) ) { $this->shipping_redirect( 'پرونده ارسال کارت پیدا نشد.', false ); }
		$status = sanitize_key( wp_unslash( $_POST['shipping_status'] ?? '' ) );
		if ( ! isset( $this->shipping_statuses()[ $status ] ) ) { $this->shipping_redirect( 'وضعیت ارسال معتبر نیست.', false ); }
		$address = sanitize_textarea_field( wp_unslash( $_POST['shipping_address'] ?? '' ) );
		$carrier = sanitize_text_field( wp_unslash( $_POST['shipping_carrier'] ?? '' ) );
		$tracking = sanitize_text_field( wp_unslash( $_POST['shipping_tracking_code'] ?? '' ) );
		$note = sanitize_textarea_field( wp_unslash( $_POST['shipping_note'] ?? '' ) );
		if ( in_array( $status, [ 'preparing','shipped','delivered' ], true ) && trim( $address ) === '' ) { $this->shipping_redirect( 'برای ادامه ارسال، آدرس کامل اجباری است.', false ); }
		if ( in_array( $status, [ 'shipped','delivered' ], true ) && ( $carrier === '' || $tracking === '' ) ) { $this->shipping_redirect( 'شرکت حمل و کد رهگیری اجباری است.', false ); }
		if ( in_array( $status, [ 'not_sent','returned','cancelled' ], true ) && trim( $note ) === '' ) { $this->shipping_redirect( 'ثبت علت برای این وضعیت اجباری است.', false ); }
		$case_status_map = [ 'awaiting_address'=>'shipping_queue','awaiting_review'=>'shipping_queue','preparing'=>'shipping_processing','shipped'=>'shipped','delivered'=>'delivered','not_sent'=>'shipping_not_sent','returned'=>'shipping_returned','cancelled'=>'shipping_cancelled' ];
		$now = current_time( 'mysql' );
		$data = [ 'status'=>$case_status_map[$status], 'shipping_status'=>$status, 'shipping_province'=>sanitize_text_field( wp_unslash( $_POST['shipping_province'] ?? '' ) ), 'shipping_city'=>sanitize_text_field( wp_unslash( $_POST['shipping_city'] ?? '' ) ), 'shipping_address'=>$address, 'shipping_postal_code'=>sanitize_text_field( wp_unslash( $_POST['shipping_postal_code'] ?? '' ) ), 'shipping_carrier'=>$carrier, 'shipping_tracking_code'=>$tracking, 'last_note'=>$note, 'updated_at'=>$now ];
		if ( $status === 'delivered' ) { $data['completed_at']=$now; $data['completed_by_user_id']=get_current_user_id(); }
		if ( false === $wpdb->update( $t['cases'], $data, [ 'id'=>$case_id ] ) ) { $this->shipping_redirect( 'ذخیره وضعیت ارسال انجام نشد.', false ); }
		$wpdb->update( $wpdb->prefix.'sn_project_operations', [ 'stage'=>$case_status_map[$status], 'updated_at'=>$now ], [ 'id'=>(int)$case->operation_id ] );
		$this->project_event( (int)$case->membership_item_id, 'operations_shipping_status_changed', [ 'old_status'=>(string)$case->shipping_status, 'status'=>$status, 'tracking_code'=>$tracking ], get_current_user_id() );
		$this->shipping_redirect( 'وضعیت ارسال کارت ذخیره شد.', true );
	}

	private function shipping_redirect( string $message, bool $success ): void {
		$url = wp_get_referer();
		if ( ! $url ) { $page_id=absint(get_option('sn_shipping_panel_page_id',0));$url=$page_id?get_permalink($page_id):home_url('/'); }
		wp_safe_redirect( add_query_arg( [ 'sn_shipping_result'=>$success?'success':'error', 'sn_shipping_message'=>$message ], $url ) ); exit;
	}

	public function render_shipping_handoffs(): string {
		if ( ! ( current_user_can( 'manage_options' ) || current_user_can( 'sn_manage_shipping' ) ) ) { return ''; }
		global $wpdb; $t=$this->tables();
		$q=sanitize_text_field(wp_unslash($_GET['shipping_q']??''));$status_filter=sanitize_key(wp_unslash($_GET['shipping_status']??''));
		$where=["c.fulfillment_type='physical_invoice'","c.shipping_status IS NOT NULL"];$args=[];
		if($q!==''){$like='%'.$wpdb->esc_like($q).'%';$where[]='(mi.content_name_snapshot LIKE %s OR m.customer_name LIKE %s OR m.customer_phone LIKE %s OR c.shipping_tracking_code LIKE %s)';array_push($args,$like,$like,$like,$like);}
		if(isset($this->shipping_statuses()[$status_filter])){$where[]='c.shipping_status=%s';$args[]=$status_filter;}
		$sql="SELECT c.*,mi.content_name_snapshot,m.customer_name,m.customer_phone,o.current_credit,o.activation_mode FROM {$t['cases']} c INNER JOIN {$wpdb->prefix}sn_project_operations o ON o.id=c.operation_id INNER JOIN {$wpdb->prefix}sn_project_membership_items mi ON mi.id=c.membership_item_id INNER JOIN {$wpdb->prefix}sn_project_memberships m ON m.id=mi.membership_id WHERE ".implode(' AND ',$where)." ORDER BY FIELD(c.shipping_status,'awaiting_address','awaiting_review','preparing','shipped','not_sent','returned','delivered','cancelled'),c.updated_at DESC,c.id DESC LIMIT 200";
		$rows=$args?($wpdb->get_results($wpdb->prepare($sql,...$args))?:[]):($wpdb->get_results($sql)?:[]);
		ob_start();?><section class="sn-card sn-execution-shipping"><div class="sn-execution-shipping-head"><div><h3>کارت‌های فاکتور کالا از چرخه بیاوین</h3><p>هر کارت مستقل از فاکتور اصلی پیگیری می‌شود.</p></div><a class="sn-btn" href="<?php echo esc_url(wp_logout_url(sn_crm_login_url()));?>">خروج</a></div><?php if(!$rows):?><p>کارت فیزیکی مطابق این فیلتر پیدا نشد.</p><?php endif;?><?php foreach($rows as$row):$location=implode('، ',array_filter([(string)$row->shipping_province,(string)$row->shipping_city]));?><details class="sn-exec-shipping-case" <?php echo in_array((string)$row->shipping_status,['awaiting_address','awaiting_review'],true)?'open':'';?>><summary><strong><?php echo esc_html((string)$row->content_name_snapshot);?></strong> — <?php echo esc_html((string)$row->customer_name);?> — <span class="sn-badge"><?php echo esc_html($this->shipping_statuses()[(string)$row->shipping_status]??(string)$row->shipping_status);?></span></summary><div class="sn-exec-shipping-meta"><span><bdi><?php echo esc_html((string)$row->customer_phone);?></bdi></span><span><?php echo esc_html($location?:'موقعیت ثبت نشده');?></span><span>اعتبار: <?php echo esc_html(SN_Helpers::format_price((float)$row->current_credit));?></span></div><form method="post" action="<?php echo esc_url(admin_url('admin-post.php'));?>" class="sn-form-grid"><input type="hidden" name="action" value="sn_operations_execution_shipping_update"><input type="hidden" name="case_id" value="<?php echo esc_attr((string)$row->id);?>"><?php wp_nonce_field('sn_operations_execution_shipping_'.(int)$row->id);?><div class="sn-field"><label>استان<input name="shipping_province" value="<?php echo esc_attr((string)$row->shipping_province);?>"></label></div><div class="sn-field"><label>شهر<input name="shipping_city" value="<?php echo esc_attr((string)$row->shipping_city);?>"></label></div><div class="sn-field sn-full"><label>آدرس<textarea name="shipping_address" rows="2"><?php echo esc_textarea((string)$row->shipping_address);?></textarea></label></div><div class="sn-field"><label>کد پستی<input name="shipping_postal_code" value="<?php echo esc_attr((string)$row->shipping_postal_code);?>"></label></div><div class="sn-field"><label>وضعیت<select name="shipping_status"><?php foreach($this->shipping_statuses() as$key=>$label):?><option value="<?php echo esc_attr($key);?>" <?php selected((string)$row->shipping_status,$key);?>><?php echo esc_html($label);?></option><?php endforeach;?></select></label></div><div class="sn-field"><label>شرکت حمل<input name="shipping_carrier" value="<?php echo esc_attr((string)$row->shipping_carrier);?>"></label></div><div class="sn-field"><label>کد رهگیری<input name="shipping_tracking_code" value="<?php echo esc_attr((string)$row->shipping_tracking_code);?>"></label></div><div class="sn-field sn-full"><label>یادداشت / علت<textarea name="shipping_note" rows="2"><?php echo esc_textarea((string)$row->last_note);?></textarea></label></div><div class="sn-field sn-full"><button class="sn-btn sn-btn-primary">ذخیره وضعیت کارت</button></div></form></details><?php endforeach;?></section><?php return(string)ob_get_clean();
	}
}
