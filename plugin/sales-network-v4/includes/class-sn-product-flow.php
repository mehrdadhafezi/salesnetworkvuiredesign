<?php
if ( ! defined( 'ABSPATH' ) ) { exit; }

/**
 * Direct commercial product flows: supervisor-visible subscriptions and
 * physical-product shipping. All storage is additive and invoice snapshots
 * remain the source of truth for historical product type and price.
 */
final class SN_Product_Flow {
	private const DB_VERSION = '2026-09-05-biavin-project-workflow-v1';
	private static ?self $instance = null;
	private bool $hooks_registered = false;
	private array $table_exists_cache = [];

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
		add_action( 'sn_invoice_paid', [ $this, 'on_invoice_paid' ], 30, 2 );
		add_action( 'admin_post_sn_shipping_update_order', [ $this, 'handle_shipping_update' ] );
		add_action( 'wp_ajax_sn_supervisor_change_subscription', [ $this, 'handle_supervisor_change_subscription' ] );
	}

	private function tables(): array {
		global $wpdb;
		return [
			'orders' => $wpdb->prefix . 'sn_shipping_orders',
			'logs' => $wpdb->prefix . 'sn_shipping_order_logs',
			'events' => $wpdb->prefix . 'sn_product_flow_events',
		];
	}

	private function table_exists( string $table ): bool {
		global $wpdb;
		if ( ! array_key_exists( $table, $this->table_exists_cache ) ) {
			$this->table_exists_cache[ $table ] = (string) $wpdb->get_var( $wpdb->prepare( 'SHOW TABLES LIKE %s', $table ) ) === $table;
		}
		return $this->table_exists_cache[ $table ];
	}

	public function maybe_upgrade(): void {
		if ( (string) get_option( 'sn_product_flow_db_version', '' ) !== self::DB_VERSION ) { $this->install_schema(); }
		if ( (string) get_option( 'sn_product_flow_access_version', '' ) !== self::DB_VERSION ) {
			$this->ensure_role_and_hr();
			$this->ensure_page();
		}
	}

	private function install_schema(): void {
		global $wpdb;
		require_once ABSPATH . 'wp-admin/includes/upgrade.php';
		$t = $this->tables();
		$charset = $wpdb->get_charset_collate();
		dbDelta( "CREATE TABLE {$t['orders']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			invoice_id BIGINT UNSIGNED NOT NULL,
			seller_id BIGINT UNSIGNED NOT NULL,
			customer_wp_id BIGINT UNSIGNED DEFAULT NULL,
			customer_name VARCHAR(120) NOT NULL,
			customer_phone VARCHAR(20) NOT NULL,
			province VARCHAR(80) DEFAULT NULL,
			city VARCHAR(80) DEFAULT NULL,
			address TEXT DEFAULT NULL,
			postal_code VARCHAR(20) DEFAULT NULL,
			items_snapshot_json LONGTEXT DEFAULT NULL,
			total_amount DECIMAL(20,2) NOT NULL DEFAULT 0,
			contact_status VARCHAR(30) NOT NULL DEFAULT 'new',
			fulfillment_status VARCHAR(40) NOT NULL DEFAULT 'awaiting_review',
			shipping_method VARCHAR(80) DEFAULT NULL,
			carrier VARCHAR(120) DEFAULT NULL,
			tracking_code VARCHAR(120) DEFAULT NULL,
			expert_note TEXT DEFAULT NULL,
			failure_reason TEXT DEFAULT NULL,
			handled_by BIGINT UNSIGNED DEFAULT NULL,
			last_contact_at DATETIME DEFAULT NULL,
			shipped_at DATETIME DEFAULT NULL,
			delivered_at DATETIME DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY invoice_id (invoice_id),
			KEY fulfillment_status (fulfillment_status,updated_at),
			KEY handled_by (handled_by,fulfillment_status),
			KEY customer_phone (customer_phone)
		) {$charset};" );
		dbDelta( "CREATE TABLE {$t['logs']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			shipping_order_id BIGINT UNSIGNED NOT NULL,
			invoice_id BIGINT UNSIGNED NOT NULL,
			actor_user_id BIGINT UNSIGNED DEFAULT NULL,
			event_key VARCHAR(80) NOT NULL,
			old_status VARCHAR(40) DEFAULT NULL,
			new_status VARCHAR(40) DEFAULT NULL,
			details_json LONGTEXT DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			KEY shipping_order_id (shipping_order_id,id),
			KEY invoice_id (invoice_id),
			KEY event_key (event_key)
		) {$charset};" );
		dbDelta( "CREATE TABLE {$t['events']} (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			invoice_id BIGINT UNSIGNED NOT NULL,
			actor_user_id BIGINT UNSIGNED DEFAULT NULL,
			event_key VARCHAR(80) NOT NULL,
			details_json LONGTEXT DEFAULT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			KEY invoice_id (invoice_id,id),
			KEY event_key (event_key)
		) {$charset};" );
		$this->table_exists_cache = [];
		update_option( 'sn_product_flow_db_version', self::DB_VERSION, false );
	}

	private function ensure_role_and_hr(): void {
		global $wpdb;
		if ( ! get_role( 'sn_shipping_expert' ) ) { add_role( 'sn_shipping_expert', 'کارشناس ارسال', [ 'read' => true, 'sn_manage_shipping' => true ] ); }
		$role = get_role( 'sn_shipping_expert' );
		if ( $role ) {
			$role->add_cap( 'read' ); $role->add_cap( 'sn_manage_shipping' );
			foreach ( [ 'edit_posts', 'delete_posts', 'publish_posts', 'upload_files', 'edit_pages', 'delete_pages', 'manage_options', 'list_users', 'create_users', 'edit_users', 'delete_users' ] as $cap ) { $role->remove_cap( $cap ); }
		}
		$positions = $wpdb->prefix . 'sn_hr_positions';
		$positions_exists = (string) $wpdb->get_var( $wpdb->prepare( 'SHOW TABLES LIKE %s', $positions ) ) === $positions;
		if ( $positions_exists ) {
			$columns = (array) $wpdb->get_col( "SHOW COLUMNS FROM {$positions}", 0 );
			$data = [ 'slug' => 'shipping_expert', 'label' => 'کارشناس ارسال', 'panel_key' => 'shipping', 'sort_order' => 91, 'is_active' => 1 ];
			if ( in_array( 'position_key', $columns, true ) ) { $data['position_key'] = 'shipping_expert'; }
			$existing = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$positions} WHERE slug=%s LIMIT 1", 'shipping_expert' ) );
			if ( $existing > 0 ) { $wpdb->update( $positions, array_intersect_key( $data, array_flip( $columns ) ), [ 'id' => $existing ] ); }
			else { $wpdb->insert( $positions, array_intersect_key( $data, array_flip( $columns ) ) ); }
		}
		$mappings = $wpdb->prefix . 'sn_hr_position_role_mappings';
		if ( $positions_exists && (string) $wpdb->get_var( $wpdb->prepare( 'SHOW TABLES LIKE %s', $mappings ) ) === $mappings ) {
			$columns = (array) $wpdb->get_col( "SHOW COLUMNS FROM {$mappings}", 0 );
			$position_id = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$positions} WHERE slug=%s LIMIT 1", 'shipping_expert' ) );
			if ( $position_id > 0 && in_array( 'legacy_role', $columns, true ) && in_array( 'position_id', $columns, true ) ) {
				$existing = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$mappings} WHERE legacy_role=%s LIMIT 1", 'sn_shipping_expert' ) );
				$data = [ 'legacy_role' => 'sn_shipping_expert', 'position_id' => $position_id ];
				if ( in_array( 'is_active', $columns, true ) ) { $data['is_active'] = 1; }
				if ( $existing > 0 ) { $wpdb->update( $mappings, $data, [ 'id' => $existing ] ); } else { $wpdb->insert( $mappings, $data ); }
			}
		}
		update_option( 'sn_product_flow_access_version', self::DB_VERSION, false );
	}

	private function ensure_page(): void {
		$option = 'sn_shipping_panel_page_id';
		$page_id = absint( get_option( $option, 0 ) );
		$page = $page_id ? get_post( $page_id ) : null;
		if ( ! $page || $page->post_type !== 'page' || $page->post_status === 'trash' ) {
			$by_path = get_page_by_path( 'crm-shipping', OBJECT, 'page' );
			$page_id = $by_path ? (int) $by_path->ID : (int) wp_insert_post( [ 'post_title' => 'پنل ارسال', 'post_name' => 'crm-shipping', 'post_content' => '[sn_shipping_panel]', 'post_status' => 'publish', 'post_type' => 'page' ] );
			if ( $page_id > 0 ) { update_option( $option, $page_id ); update_post_meta( $page_id, '_sn_system_page', '1' ); }
		}
		if ( $page_id > 0 ) {
			$page = get_post( $page_id );
			if ( $page && ! has_shortcode( (string) $page->post_content, 'sn_shipping_panel' ) ) {
				wp_update_post( [ 'ID' => $page_id, 'post_content' => trim( (string) $page->post_content ) . "\n\n[sn_shipping_panel]" ] );
			}
		}
	}

	private function canonical_type( string $type ): string {
		$type = sanitize_key( $type );
		return $type === 'subscription_star' ? 'product_star' : $type;
	}

	private function invoice_items( int $invoice_id ): array {
		global $wpdb;
		return $wpdb->get_results( $wpdb->prepare( "SELECT * FROM {$wpdb->prefix}sn_invoice_items WHERE invoice_id=%d ORDER BY id", $invoice_id ), ARRAY_A ) ?: [];
	}

	public function on_invoice_paid( int $invoice_id, $invoice ): void {
		if ( ! is_object( $invoice ) ) { global $wpdb; $invoice = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$wpdb->prefix}sn_invoices WHERE id=%d", $invoice_id ) ); }
		if ( ! $invoice ) { return; }
		$items = $this->invoice_items( $invoice_id );
		$types = array_values( array_unique( array_map( fn( $row ) => $this->canonical_type( (string) ( $row['product_type'] ?? '' ) ), $items ) ) );
		if ( ! $items || $types !== [ 'product' ] ) { return; }
		global $wpdb; $t = $this->tables();
		if ( ! $this->table_exists( $t['orders'] ) ) { $this->install_schema(); }
		$existing = (int) $wpdb->get_var( $wpdb->prepare( "SELECT id FROM {$t['orders']} WHERE invoice_id=%d LIMIT 1", $invoice_id ) );
		if ( $existing > 0 ) { return; }
		$address = sanitize_textarea_field( (string) ( $invoice->customer_address ?? '' ) );
		$status = $address !== '' ? 'awaiting_review' : 'awaiting_address';
		$inserted = $wpdb->insert( $t['orders'], [
			'invoice_id' => $invoice_id, 'seller_id' => (int) ( $invoice->seller_id ?? 0 ),
			'customer_wp_id' => ! empty( $invoice->customer_wp_id ) ? (int) $invoice->customer_wp_id : null,
			'customer_name' => sanitize_text_field( (string) ( $invoice->customer_name ?? '' ) ),
			'customer_phone' => class_exists( 'SN_Helpers' ) ? SN_Helpers::normalize_mobile( (string) ( $invoice->customer_phone ?? '' ) ) : sanitize_text_field( (string) ( $invoice->customer_phone ?? '' ) ),
			'province' => sanitize_text_field( (string) ( $invoice->province ?? '' ) ), 'city' => sanitize_text_field( (string) ( $invoice->city ?? '' ) ),
			'address' => $address, 'postal_code' => sanitize_text_field( (string) ( $invoice->customer_postal_code ?? '' ) ),
			'items_snapshot_json' => wp_json_encode( $items, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ),
			'total_amount' => max( 0, (float) ( $invoice->final_total ?? $invoice->payment_total_amount ?? $invoice->product_price ?? 0 ) ),
			'contact_status' => 'new', 'fulfillment_status' => $status, 'created_at' => current_time( 'mysql' ), 'updated_at' => current_time( 'mysql' ),
		] );
		if ( $inserted ) { $this->log_order( (int) $wpdb->insert_id, $invoice_id, 'shipping_order_created', '', $status, [ 'items' => count( $items ) ] ); }
	}

	private function current_position(): string {
		global $wpdb; $user_id = get_current_user_id();
		if ( $user_id < 1 ) { return ''; }
		if ( ! $this->table_exists( $wpdb->prefix . 'sn_hr_profiles' ) || ! $this->table_exists( $wpdb->prefix . 'sn_hr_positions' ) ) { return ''; }
		return (string) $wpdb->get_var( $wpdb->prepare( "SELECT pos.slug FROM {$wpdb->prefix}sn_hr_profiles p INNER JOIN {$wpdb->prefix}sn_hr_positions pos ON pos.id=p.position_id WHERE p.user_id=%d AND p.is_active=1 ORDER BY p.id DESC LIMIT 1", $user_id ) );
	}

	private function can_manage_shipping(): bool {
		if ( current_user_can( 'manage_options' ) || current_user_can( 'sn_manage_shipping' ) ) { return true; }
		$user = wp_get_current_user();
		return in_array( 'sn_shipping_expert', (array) $user->roles, true ) || $this->current_position() === 'shipping_expert';
	}

	private function contact_statuses(): array {
		return [ 'new' => 'جدید', 'no_answer' => 'جواب نداده', 'contacted' => 'تماس انجام شد', 'follow_up' => 'تماس مجدد' ];
	}

	private function fulfillment_statuses(): array {
		return [ 'awaiting_address' => 'نیازمند تکمیل آدرس', 'awaiting_review' => 'در انتظار بررسی', 'preparing' => 'آماده‌سازی', 'shipped' => 'ارسال شد', 'delivered' => 'تحویل شد', 'not_sent' => 'ارسال نشد', 'returned' => 'مرجوع شد', 'cancelled' => 'لغو شد' ];
	}

	private function log_order( int $order_id, int $invoice_id, string $event, string $old, string $new, array $details = [] ): void {
		global $wpdb; $t = $this->tables();
		$wpdb->insert( $t['logs'], [ 'shipping_order_id' => $order_id, 'invoice_id' => $invoice_id, 'actor_user_id' => get_current_user_id() ?: null, 'event_key' => sanitize_key( $event ), 'old_status' => $old ?: null, 'new_status' => $new ?: null, 'details_json' => wp_json_encode( $details, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ), 'created_at' => current_time( 'mysql' ) ] );
	}

	private function redirect_shipping( string $message, bool $success ): void {
		$url = wp_get_referer();
		if ( ! $url ) { $page_id = (int) get_option( 'sn_shipping_panel_page_id', 0 ); $url = $page_id ? get_permalink( $page_id ) : home_url( '/' ); }
		wp_safe_redirect( add_query_arg( [ 'sn_shipping_result' => $success ? 'success' : 'error', 'sn_shipping_message' => $message ], $url ) );
		exit;
	}

	public function handle_shipping_update(): void {
		if ( ! $this->can_manage_shipping() ) { wp_die( 'دسترسی غیرمجاز', '', [ 'response' => 403 ] ); }
		$order_id = absint( $_POST['order_id'] ?? 0 );
		if ( ! wp_verify_nonce( sanitize_text_field( wp_unslash( $_POST['_wpnonce'] ?? '' ) ), 'sn_shipping_update_' . $order_id ) ) { $this->redirect_shipping( 'درخواست امنیتی نامعتبر است.', false ); }
		global $wpdb; $t = $this->tables();
		$order = $wpdb->get_row( $wpdb->prepare( "SELECT * FROM {$t['orders']} WHERE id=%d", $order_id ) );
		if ( ! $order ) { $this->redirect_shipping( 'سفارش ارسال یافت نشد.', false ); }
		$contact = sanitize_key( wp_unslash( $_POST['contact_status'] ?? 'new' ) );
		$status = sanitize_key( wp_unslash( $_POST['fulfillment_status'] ?? 'awaiting_review' ) );
		if ( ! isset( $this->contact_statuses()[ $contact ] ) || ! isset( $this->fulfillment_statuses()[ $status ] ) ) { $this->redirect_shipping( 'وضعیت انتخاب‌شده معتبر نیست.', false ); }
		$customer_name = sanitize_text_field( wp_unslash( $_POST['customer_name'] ?? '' ) );
		$customer_phone_raw = sanitize_text_field( wp_unslash( $_POST['customer_phone'] ?? '' ) );
		$customer_phone = class_exists( 'SN_Helpers' ) ? SN_Helpers::normalize_mobile( $customer_phone_raw ) : preg_replace( '/\D+/', '', $customer_phone_raw );
		if ( $customer_name === '' ) { $this->redirect_shipping( 'نام مشتری اجباری است.', false ); }
		if ( class_exists( 'SN_Helpers' ) && ! SN_Helpers::is_valid_mobile( $customer_phone ) ) { $this->redirect_shipping( 'شماره موبایل مشتری معتبر نیست.', false ); }
		if ( ! class_exists( 'SN_Helpers' ) && $customer_phone === '' ) { $this->redirect_shipping( 'شماره موبایل مشتری اجباری است.', false ); }
		$address = sanitize_textarea_field( wp_unslash( $_POST['address'] ?? '' ) );
		$carrier = sanitize_text_field( wp_unslash( $_POST['carrier'] ?? '' ) );
		$tracking = sanitize_text_field( wp_unslash( $_POST['tracking_code'] ?? '' ) );
		$failure = sanitize_textarea_field( wp_unslash( $_POST['failure_reason'] ?? '' ) );
		if ( in_array( $status, [ 'preparing', 'shipped', 'delivered' ], true ) && trim( $address ) === '' ) { $this->redirect_shipping( 'برای ادامه ارسال، ثبت آدرس کامل اجباری است.', false ); }
		if ( in_array( $status, [ 'shipped', 'delivered' ], true ) && ( $carrier === '' || $tracking === '' ) ) { $this->redirect_shipping( 'برای وضعیت ارسال‌شده، شرکت حمل و کد رهگیری اجباری است.', false ); }
		if ( in_array( $status, [ 'not_sent', 'returned', 'cancelled' ], true ) && trim( $failure ) === '' ) { $this->redirect_shipping( 'ثبت علت برای ارسال‌نشدن، مرجوعی یا لغو اجباری است.', false ); }
		$now = current_time( 'mysql' );
		$data = [
			'customer_name' => $customer_name,
			'customer_phone' => $customer_phone,
			'province' => sanitize_text_field( wp_unslash( $_POST['province'] ?? '' ) ), 'city' => sanitize_text_field( wp_unslash( $_POST['city'] ?? '' ) ),
			'address' => $address, 'postal_code' => sanitize_text_field( wp_unslash( $_POST['postal_code'] ?? '' ) ),
			'contact_status' => $contact, 'fulfillment_status' => $status,
			'shipping_method' => sanitize_text_field( wp_unslash( $_POST['shipping_method'] ?? '' ) ), 'carrier' => $carrier, 'tracking_code' => $tracking,
			'expert_note' => sanitize_textarea_field( wp_unslash( $_POST['expert_note'] ?? '' ) ), 'failure_reason' => $failure ?: null,
			'handled_by' => get_current_user_id(), 'updated_at' => $now,
		];
		if ( $contact !== (string) $order->contact_status ) { $data['last_contact_at'] = $now; }
		if ( $status === 'shipped' && empty( $order->shipped_at ) ) { $data['shipped_at'] = $now; }
		if ( $status === 'delivered' && empty( $order->delivered_at ) ) { $data['delivered_at'] = $now; }
		if ( false === $wpdb->update( $t['orders'], $data, [ 'id' => $order_id ] ) ) { $this->redirect_shipping( 'ذخیره تغییرات ارسال انجام نشد.', false ); }
		$this->log_order( $order_id, (int) $order->invoice_id, 'shipping_order_updated', (string) $order->fulfillment_status, $status, [ 'contact_status' => $contact, 'carrier' => $carrier, 'tracking_code' => $tracking ] );
		$this->redirect_shipping( 'اطلاعات ارسال ذخیره شد.', true );
	}

	public function render_shipping_panel(): string {
		if ( ! is_user_logged_in() ) { return '<div class="sn-notice">ابتدا وارد حساب کاربری شوید.</div>'; }
		if ( ! $this->can_manage_shipping() ) { return '<div class="sn-notice sn-error">این صفحه فقط برای کارشناس ارسال است.</div>'; }
		global $wpdb; $t = $this->tables();
		if ( ! $this->table_exists( $t['orders'] ) ) { $this->install_schema(); }
		$q = sanitize_text_field( wp_unslash( $_GET['shipping_q'] ?? '' ) );
		$status_filter = sanitize_key( wp_unslash( $_GET['shipping_status'] ?? '' ) );
		$where = [ '1=1' ]; $args = [];
		if ( $q !== '' ) { $like = '%' . $wpdb->esc_like( $q ) . '%'; $where[] = '(customer_name LIKE %s OR customer_phone LIKE %s OR invoice_id IN (SELECT id FROM ' . $wpdb->prefix . 'sn_invoices WHERE invoice_code LIKE %s) OR tracking_code LIKE %s)'; array_push( $args, $like, $like, $like, $like ); }
		if ( isset( $this->fulfillment_statuses()[ $status_filter ] ) ) { $where[] = 'fulfillment_status=%s'; $args[] = $status_filter; }
		$sql = "SELECT * FROM {$t['orders']} WHERE " . implode( ' AND ', $where ) . ' ORDER BY FIELD(fulfillment_status,\'awaiting_address\',\'awaiting_review\',\'preparing\',\'shipped\',\'not_sent\',\'returned\',\'delivered\',\'cancelled\'),updated_at DESC,id DESC LIMIT 200';
		$orders = $args ? ( $wpdb->get_results( $wpdb->prepare( $sql, ...$args ) ) ?: [] ) : ( $wpdb->get_results( $sql ) ?: [] );
		$counts = [];
		foreach ( $wpdb->get_results( "SELECT fulfillment_status,COUNT(*) total FROM {$t['orders']} GROUP BY fulfillment_status" ) ?: [] as $row ) { $counts[(string)$row->fulfillment_status] = (int) $row->total; }
		if ( class_exists( 'SN_Operations_Execution' ) ) { foreach ( SN_Operations_Execution::instance()->shipping_counts() as $key=>$total ) { $counts[$key]=(int)($counts[$key]??0)+(int)$total; } }
		ob_start(); ?>
			<div class="sn-panel sn-portal" id="sn-shipping-panel" dir="rtl">
				<div class="sn-panel-header"><div><h2>پنل ارسال</h2><p class="sn-muted">سفارش‌های محصول عادی و کارت‌های فاکتور کالا اینجا پیگیری می‌شوند.</p></div><a class="sn-btn" href="<?php echo esc_url( wp_logout_url( function_exists( 'sn_crm_login_url' ) ? sn_crm_login_url() : home_url( '/crm-login/' ) ) ); ?>">خروج</a></div>
			<?php if ( ! empty( $_GET['sn_shipping_message'] ) ) : ?><div class="sn-notice <?php echo (string) ( $_GET['sn_shipping_result'] ?? '' ) === 'success' ? 'sn-success' : 'sn-error'; ?>"><?php echo esc_html( sanitize_text_field( wp_unslash( $_GET['sn_shipping_message'] ) ) ); ?></div><?php endif; ?>
			<div class="sn-kpi-grid"><?php foreach ( $this->fulfillment_statuses() as $key => $label ) : ?><div class="sn-kpi"><strong><?php echo esc_html( number_format_i18n( (int) ( $counts[$key] ?? 0 ) ) ); ?></strong><span><?php echo esc_html( $label ); ?></span></div><?php endforeach; ?></div>
				<form method="get" class="sn-card sn-filter-bar"><label class="sn-grow">جستجو<input type="search" name="shipping_q" value="<?php echo esc_attr( $q ); ?>" placeholder="نام، موبایل، کد فاکتور یا رهگیری"></label><label>وضعیت<select name="shipping_status"><option value="">همه</option><?php foreach ( $this->fulfillment_statuses() as $key => $label ) : ?><option value="<?php echo esc_attr($key); ?>" <?php selected($status_filter,$key); ?>><?php echo esc_html($label); ?></option><?php endforeach; ?></select></label><button class="sn-btn sn-btn-primary">اعمال</button></form>
				<?php if ( class_exists( 'SN_Operations_Execution' ) ) { echo SN_Operations_Execution::instance()->render_shipping_handoffs(); } // phpcs:ignore WordPress.Security.EscapeOutput.OutputNotEscaped ?>
				<?php if ( ! $orders ) : ?><div class="sn-card sn-empty-state">سفارش محصول عادی با این فیلتر یافت نشد.</div><?php endif; ?>
			<?php foreach ( $orders as $order ) : $items = json_decode( (string) $order->items_snapshot_json, true ); ?>
			<details class="sn-card sn-section" <?php echo in_array((string)$order->fulfillment_status,['awaiting_address','awaiting_review'],true)?'open':''; ?>>
				<summary><strong>سفارش #<?php echo esc_html((string)$order->id); ?></strong> — فاکتور #<?php echo esc_html((string)$order->invoice_id); ?> — <?php echo esc_html((string)$order->customer_name); ?> — <span class="sn-badge"><?php echo esc_html($this->fulfillment_statuses()[(string)$order->fulfillment_status]??(string)$order->fulfillment_status); ?></span></summary>
				<div class="sn-table-wrap"><table class="sn-table"><thead><tr><th>محصول</th><th>تعداد</th><th>قیمت</th></tr></thead><tbody><?php foreach (is_array($items)?$items:[] as $item): ?><tr><td><?php echo esc_html((string)($item['product_name']??'—')); ?></td><td><?php echo esc_html(number_format_i18n((int)($item['qty']??1))); ?></td><td><?php echo esc_html(class_exists('SN_Helpers')?SN_Helpers::format_price((float)($item['total_price']??0)):(string)($item['total_price']??0)); ?></td></tr><?php endforeach; ?></tbody></table></div>
				<form method="post" action="<?php echo esc_url(admin_url('admin-post.php')); ?>" class="sn-form-grid"><input type="hidden" name="action" value="sn_shipping_update_order"><input type="hidden" name="order_id" value="<?php echo esc_attr((string)$order->id); ?>"><?php wp_nonce_field('sn_shipping_update_'.(int)$order->id); ?>
					<div class="sn-field"><label>نام مشتری<input name="customer_name" value="<?php echo esc_attr((string)$order->customer_name); ?>" required></label></div><div class="sn-field"><label>موبایل<input name="customer_phone" value="<?php echo esc_attr((string)$order->customer_phone); ?>" required></label></div>
					<div class="sn-field"><label>استان<input name="province" value="<?php echo esc_attr((string)$order->province); ?>"></label></div><div class="sn-field"><label>شهر<input name="city" value="<?php echo esc_attr((string)$order->city); ?>"></label></div>
					<div class="sn-field sn-full"><label>آدرس کامل<textarea name="address" rows="3"><?php echo esc_textarea((string)$order->address); ?></textarea></label></div><div class="sn-field"><label>کد پستی<input name="postal_code" value="<?php echo esc_attr((string)$order->postal_code); ?>"></label></div>
					<div class="sn-field"><label>نتیجه تماس<select name="contact_status"><?php foreach($this->contact_statuses() as $key=>$label): ?><option value="<?php echo esc_attr($key); ?>" <?php selected((string)$order->contact_status,$key); ?>><?php echo esc_html($label); ?></option><?php endforeach; ?></select></label></div>
					<div class="sn-field"><label>وضعیت ارسال<select name="fulfillment_status"><?php foreach($this->fulfillment_statuses() as $key=>$label): ?><option value="<?php echo esc_attr($key); ?>" <?php selected((string)$order->fulfillment_status,$key); ?>><?php echo esc_html($label); ?></option><?php endforeach; ?></select></label></div>
					<div class="sn-field"><label>روش ارسال<input name="shipping_method" value="<?php echo esc_attr((string)$order->shipping_method); ?>" placeholder="پست، تیپاکس، پیک..."></label></div><div class="sn-field"><label>شرکت حمل<input name="carrier" value="<?php echo esc_attr((string)$order->carrier); ?>"></label></div><div class="sn-field"><label>کد رهگیری<input name="tracking_code" value="<?php echo esc_attr((string)$order->tracking_code); ?>"></label></div>
					<div class="sn-field sn-full"><label>یادداشت کارشناس<textarea name="expert_note" rows="3"><?php echo esc_textarea((string)$order->expert_note); ?></textarea></label></div><div class="sn-field sn-full"><label>علت ارسال‌نشدن/مرجوعی/لغو<textarea name="failure_reason" rows="2"><?php echo esc_textarea((string)$order->failure_reason); ?></textarea></label></div>
					<div class="sn-field sn-full"><button class="sn-btn sn-btn-primary">ذخیره پیگیری ارسال</button></div>
				</form>
			</details><?php endforeach; ?>
		</div><?php return ob_get_clean();
	}

	private function descendant_seller_ids( int $actor_id ): array {
		global $wpdb; $profiles = $wpdb->prefix . 'sn_hr_profiles'; $assignments = $wpdb->prefix . 'sn_hr_assignments'; $positions = $wpdb->prefix . 'sn_hr_positions';
		if ( ! $this->table_exists( $profiles ) || ! $this->table_exists( $assignments ) || ! $this->table_exists( $positions ) ) {
			return array_map( 'intval', get_users( [ 'meta_key' => 'sn_supervisor_id', 'meta_value' => $actor_id, 'fields' => 'ID' ] ) );
		}
		$root_ids = array_map( 'intval', (array) $wpdb->get_col( $wpdb->prepare( "SELECT id FROM {$profiles} WHERE user_id=%d AND is_active=1", $actor_id ) ) );
		$seen = array_fill_keys( $root_ids, true ); $frontier = $root_ids; $seller_ids = [];
		for ( $depth = 0; $depth < 20 && $frontier; $depth++ ) {
			$sql = "SELECT child.id,child.user_id,pos.slug FROM {$assignments} a INNER JOIN {$profiles} child ON child.id=a.child_profile_id AND child.is_active=1 LEFT JOIN {$positions} pos ON pos.id=child.position_id WHERE a.is_current=1 AND a.parent_profile_id IN (" . implode( ',', array_map( 'intval', $frontier ) ) . ')';
			$next = [];
			foreach ( $wpdb->get_results( $sql ) ?: [] as $row ) { if ( isset( $seen[(int)$row->id] ) ) { continue; } $seen[(int)$row->id] = true; $next[] = (int)$row->id; if ( (string)$row->slug === 'seller' ) { $seller_ids[] = (int)$row->user_id; } }
			$frontier = $next;
		}
		$seller_ids = array_merge( $seller_ids, array_map( 'intval', get_users( [ 'meta_key' => 'sn_supervisor_id', 'meta_value' => $actor_id, 'fields' => 'ID' ] ) ) );
		return array_values( array_unique( array_filter( $seller_ids ) ) );
	}

	private function can_manage_subscription_invoice( object $invoice, int $actor_id ): bool {
		if ( current_user_can( 'manage_options' ) ) { return true; }
		if ( (int) ( $invoice->issued_by_user_id ?? 0 ) === $actor_id ) { return true; }
		return in_array( (int) ( $invoice->seller_id ?? 0 ), $this->descendant_seller_ids( $actor_id ), true );
	}

	private function subscription_products(): array {
		$ids = class_exists( 'SN_Projects' ) ? SN_Projects::instance()->subscription_product_ids() : [];
		$out = [];
		foreach ( $ids as $id ) { $product = function_exists('wc_get_product') ? wc_get_product((int)$id) : null; if($product && get_post_status((int)$id)==='publish' && (string)get_post_meta((int)$id,'_sn_enabled',true)==='1'){$out[(int)$id]=['name'=>(string)$product->get_name(),'price'=>(float)$product->get_price()];} }
		return $out;
	}

	public function render_supervisor_subscriptions( int $actor_id ): void {
		global $wpdb; $seller_ids = $this->descendant_seller_ids( $actor_id );
		$scope = [ 'i.issued_by_user_id=%d' ]; $args = [ $actor_id ];
		if ( $seller_ids ) { $scope[] = 'i.seller_id IN (' . implode(',',array_fill(0,count($seller_ids),'%d')) . ')'; array_push($args,...$seller_ids); }
		$sql = "SELECT i.*,ii.id item_id,ii.product_id subscription_product_id,ii.product_name subscription_name,ii.unit_price FROM {$wpdb->prefix}sn_invoices i INNER JOIN {$wpdb->prefix}sn_invoice_items ii ON ii.invoice_id=i.id AND ii.product_type='subscription' WHERE (".implode(' OR ',$scope).") AND COALESCE(i.status,'') NOT IN ('cancelled','payment_archived') ORDER BY i.id DESC LIMIT 100";
		$rows = $wpdb->get_results( $wpdb->prepare( $sql, ...$args ) ) ?: []; $products = $this->subscription_products();
		?><div class="sn-card sn-section"><div class="sn-card-head"><div><h3>اشتراک‌های مستقیم فروشنده</h3><p class="sn-note">سرپرست اشتراک انتخاب‌شده را می‌بیند و تا پیش از ثبت هر پرداخت می‌تواند آن را تغییر دهد. بعد از شروع پرداخت، نوع و قیمت برای حفاظت مالی قفل می‌شود.</p></div></div><div class="sn-table-wrap"><table class="sn-table"><thead><tr><th>فاکتور</th><th>مشتری</th><th>فروشنده</th><th>اشتراک</th><th>پرداخت</th><th>تغییر اشتراک</th></tr></thead><tbody>
		<?php if(!$rows): ?><tr><td colspan="6"><div class="sn-empty-state">اشتراک مستقیمی در محدوده شما ثبت نشده است.</div></td></tr><?php endif; ?>
		<?php foreach($rows as $row): $payment_started=(float)($row->paid_total_amount??0)>0.5 || in_array((string)($row->status??''),['paid','approved','completed','partial_paid','pending_financial_approval','receipt_uploaded'],true); ?><tr><td><strong><?php echo esc_html((string)$row->invoice_code); ?></strong><br><small>#<?php echo esc_html((string)$row->id); ?></small></td><td><?php echo esc_html((string)$row->customer_name); ?><br><code><?php echo esc_html((string)$row->customer_phone); ?></code></td><td><?php echo esc_html((string)get_the_author_meta('display_name',(int)$row->seller_id)); ?></td><td><strong><?php echo esc_html((string)$row->subscription_name); ?></strong><br><?php echo esc_html(class_exists('SN_Helpers')?SN_Helpers::format_price((float)$row->unit_price):(string)$row->unit_price); ?></td><td><span class="sn-badge"><?php echo esc_html(class_exists('SN_Helpers')?SN_Helpers::status_label((string)$row->status):(string)$row->status); ?></span></td><td><?php if(!$payment_started&&$products): ?><form class="sn-supervisor-subscription-change" method="post"><input type="hidden" name="action" value="sn_supervisor_change_subscription"><input type="hidden" name="invoice_id" value="<?php echo esc_attr((string)$row->id); ?>"><?php wp_nonce_field('sn_supervisor_change_subscription_'.(int)$row->id,'_wpnonce',false); ?><select name="product_id"><?php foreach($products as $pid=>$product): ?><option value="<?php echo esc_attr((string)$pid); ?>" <?php selected((int)$row->subscription_product_id,$pid); ?>><?php echo esc_html($product['name'].' — '.(class_exists('SN_Helpers')?SN_Helpers::format_price($product['price']):(string)$product['price'])); ?></option><?php endforeach; ?></select><button type="submit" class="sn-btn sn-btn-sm">ثبت تغییر</button></form><?php else: ?><small>پس از شروع پرداخت قفل است</small><?php endif; ?></td></tr><?php endforeach; ?>
		</tbody></table></div><div class="sn-supervisor-subscription-notice"></div></div><?php
	}

	public function handle_supervisor_change_subscription(): void {
		$invoice_id = absint( $_POST['invoice_id'] ?? 0 );
		if ( ! is_user_logged_in() || ! wp_verify_nonce( sanitize_text_field( wp_unslash( $_POST['_wpnonce'] ?? '' ) ), 'sn_supervisor_change_subscription_' . $invoice_id ) ) { wp_send_json_error(['message'=>'درخواست امنیتی نامعتبر است.'],403); }
		$new_product_id = absint( $_POST['product_id'] ?? 0 ); $actor_id = get_current_user_id();
		if ( ! class_exists('SN_Projects') || ! SN_Projects::instance()->is_subscription_product($new_product_id) ) { wp_send_json_error(['message'=>'اشتراک انتخاب‌شده معتبر نیست.'],400); }
		$product = function_exists('wc_get_product') ? wc_get_product($new_product_id) : null;
		if(!$product || get_post_status($new_product_id)!=='publish' || (string)get_post_meta($new_product_id,'_sn_enabled',true)!=='1'){wp_send_json_error(['message'=>'اشتراک انتخاب‌شده قابل فروش نیست.'],400);}
		global $wpdb; $invoice_table=$wpdb->prefix.'sn_invoices'; $item_table=$wpdb->prefix.'sn_invoice_items'; $stage_table=$wpdb->prefix.'sn_invoice_payment_stages'; $payment_table=$wpdb->prefix.'sn_payments';
		$wpdb->query('START TRANSACTION'); $invoice=$wpdb->get_row($wpdb->prepare("SELECT * FROM {$invoice_table} WHERE id=%d FOR UPDATE",$invoice_id));
		if(!$invoice || !$this->can_manage_subscription_invoice($invoice,$actor_id)){ $wpdb->query('ROLLBACK'); wp_send_json_error(['message'=>'دسترسی به این فاکتور مجاز نیست.'],403); }
		$items=$wpdb->get_results($wpdb->prepare("SELECT * FROM {$item_table} WHERE invoice_id=%d FOR UPDATE",$invoice_id))?:[];
		if(count($items)!==1 || (string)($items[0]->product_type??'')!=='subscription'){ $wpdb->query('ROLLBACK'); wp_send_json_error(['message'=>'این فاکتور از نوع اشتراک مستقیم نیست.'],409); }
		$payment_count=(int)$wpdb->get_var($wpdb->prepare("SELECT COUNT(*) FROM {$payment_table} WHERE invoice_id=%d AND status NOT IN ('cancelled','failed')",$invoice_id));
		$approved_stage=(int)$wpdb->get_var($wpdb->prepare("SELECT COUNT(*) FROM {$stage_table} WHERE invoice_id=%d AND status IN ('paid','approved','pending_financial_approval','receipt_uploaded')",$invoice_id));
		if($payment_count>0||$approved_stage>0||(float)($invoice->paid_total_amount??0)>0.5){$wpdb->query('ROLLBACK');wp_send_json_error(['message'=>'پرداخت این فاکتور شروع شده و اشتراک دیگر قابل تغییر نیست.'],409);}
		$price=(float)$product->get_price(); if($price<=0){$wpdb->query('ROLLBACK');wp_send_json_error(['message'=>'قیمت اشتراک معتبر نیست.'],400);}
		$stage=$wpdb->get_row($wpdb->prepare("SELECT * FROM {$stage_table} WHERE invoice_id=%d AND stage_no=%d FOR UPDATE",$invoice_id,max(1,(int)($invoice->current_payment_stage??1))));
		$due=(string)($invoice->payment_plan??'full')==='partial'?(float)($stage->requested_amount??$invoice->current_due_amount??0):$price;
		if((string)($invoice->payment_plan??'full')==='partial'&&($due<=0||$due>=$price)){$wpdb->query('ROLLBACK');wp_send_json_error(['message'=>'با قیمت اشتراک جدید، مبلغ پیش‌پرداخت معتبر نیست؛ ابتدا فاکتور جدید صادر کنید.'],409);}
		$now=current_time('mysql'); $item_ok=false!==$wpdb->update($item_table,['product_id'=>$new_product_id,'product_name'=>(string)$product->get_name(),'product_type'=>'subscription','qty'=>1,'unit_price'=>$price,'total_price'=>$price],['id'=>(int)$items[0]->id]);
		$invoice_ok=false!==$wpdb->update($invoice_table,['product_id'=>$new_product_id,'product_price'=>$price,'final_total'=>$price,'payment_total_amount'=>$price,'current_due_amount'=>$due,'remaining_amount'=>$price,'updated_at'=>$now],['id'=>$invoice_id]);
		$stage_ok=!$stage || false!==$wpdb->update($stage_table,['requested_amount'=>$due,'updated_at'=>$now],['id'=>(int)$stage->id]);
		if(!$item_ok||!$invoice_ok||!$stage_ok){$wpdb->query('ROLLBACK');wp_send_json_error(['message'=>'تغییر اشتراک به‌طور کامل ذخیره نشد.'],500);}
		$wpdb->query('COMMIT');
		$events=$this->tables()['events'];
		if($this->table_exists($events)){$wpdb->insert($events,['invoice_id'=>$invoice_id,'actor_user_id'=>$actor_id,'event_key'=>'supervisor_subscription_changed','details_json'=>wp_json_encode(['old_product_id'=>(int)$items[0]->product_id,'new_product_id'=>$new_product_id,'old_price'=>(float)$items[0]->unit_price,'new_price'=>$price],JSON_UNESCAPED_UNICODE|JSON_UNESCAPED_SLASHES),'created_at'=>current_time('mysql')]);}
		wp_send_json_success(['message'=>'اشتراک و مبلغ فاکتور با موفقیت به‌روزرسانی شد.','product_name'=>(string)$product->get_name(),'price'=>$price]);
	}
}
