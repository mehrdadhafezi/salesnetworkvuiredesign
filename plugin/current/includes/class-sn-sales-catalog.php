<?php
if ( ! defined( 'ABSPATH' ) ) { exit; }

/** Manager-scoped product permissions and fixed, server-authorized discounts. */
final class SN_Sales_Catalog {
    private const META_KEY = '_sn_sales_catalog_policy';
    private const SCHEMA_VERSION = '2026-10-05-sales-catalog-v3-payment-stage-fix';
    private static ?self $instance = null;
    private array $manager_cache = [];

    public static function instance(): self {
        if ( self::$instance === null ) { self::$instance = new self(); }
        return self::$instance;
    }

    public function register_hooks(): void {
        add_action( 'admin_menu', [ $this, 'admin_menu' ], 30 );
        add_action( 'admin_enqueue_scripts', [ $this, 'admin_assets' ] );
        add_action( 'wp_enqueue_scripts', [ $this, 'staff_assets' ], 45 );
        add_action( 'init', [ $this, 'maybe_upgrade' ], 40 );
        add_action( 'wp_ajax_sn_sales_catalog_load', [ $this, 'ajax_load' ] );
        add_action( 'wp_ajax_sn_sales_catalog_save', [ $this, 'ajax_save' ] );
        add_action( 'wp_ajax_sn_sales_catalog_staff', [ $this, 'ajax_staff' ] );
    }

    private function table_exists( string $table ): bool {
        global $wpdb;
        return (string) $wpdb->get_var( $wpdb->prepare( 'SHOW TABLES LIKE %s', $wpdb->esc_like( $table ) ) ) === $table;
    }

    public function maybe_upgrade(): void {
        if ( class_exists( 'SN_Plugin', false ) && get_option( 'sn_sales_catalog_schema_version' ) !== self::SCHEMA_VERSION ) { $this->ensure_schema(); }
    }

    /** Schema upgrade and targeted normalization for catalog-discount invoices. */
    public function ensure_schema(): bool {
        global $wpdb;
        $definitions = [
            'sn_invoices' => [
                'catalog_discounts_snapshot' => 'LONGTEXT DEFAULT NULL',
                'catalog_manager_user_id' => 'BIGINT UNSIGNED DEFAULT NULL',
                'original_total' => 'DECIMAL(18,2) DEFAULT NULL',
                'discount_amount' => 'DECIMAL(18,2) DEFAULT NULL',
                'discount_total' => 'DECIMAL(18,2) DEFAULT NULL',
            ],
            'sn_dot_cases' => [ 'catalog_discount_snapshot' => 'LONGTEXT DEFAULT NULL' ],
        ];
        $ok = true;
        foreach ( $definitions as $suffix => $columns ) {
            $table = $wpdb->prefix . $suffix;
            if ( ! $this->table_exists( $table ) ) { $ok = false; continue; }
            $existing = (array) $wpdb->get_col( "SHOW COLUMNS FROM {$table}", 0 );
            foreach ( $columns as $column => $definition ) {
                if ( ! in_array( $column, $existing, true ) && false === $wpdb->query( "ALTER TABLE {$table} ADD COLUMN {$column} {$definition}" ) ) { $ok = false; }
            }
        }
        if ( $ok ) {
            // product_price is a legacy payable-total field throughout the CRM.
            // v2.0.142 stored the pre-discount gross amount there for catalog-discount
            // invoices, so older payment/report paths could ignore final_total. Keep
            // original_total as the gross audit value and normalize product_price to
            // the actual payable amount for both existing and future compatibility.
            $invoice_table = $wpdb->prefix . 'sn_invoices';
            if ( $this->table_exists( $invoice_table ) ) {
                $repair = $wpdb->query( "UPDATE {$invoice_table}
                    SET product_price = final_total,
                        payment_total_amount = final_total,
                        remaining_amount = GREATEST(0, final_total - COALESCE(paid_total_amount, 0)),
                        current_due_amount = CASE
                            WHEN current_due_amount IS NULL THEN NULL
                            ELSE LEAST(GREATEST(0, current_due_amount), GREATEST(0, final_total - COALESCE(paid_total_amount, 0)))
                        END
                    WHERE catalog_discounts_snapshot IS NOT NULL
                      AND catalog_discounts_snapshot <> ''
                      AND GREATEST(COALESCE(discount_total, 0), COALESCE(discount_amount, 0)) > 0
                      AND final_total IS NOT NULL
                      AND final_total > 0
                      AND (
                          ABS(COALESCE(product_price, 0) - final_total) > 0.01
                          OR ABS(COALESCE(payment_total_amount, 0) - final_total) > 0.01
                          OR ABS(COALESCE(remaining_amount, 0) - GREATEST(0, final_total - COALESCE(paid_total_amount, 0))) > 0.01
                          OR (current_due_amount IS NOT NULL AND current_due_amount > GREATEST(0, final_total - COALESCE(paid_total_amount, 0)) + 0.01)
                      )" );
                if ( $repair === false ) { $ok = false; }

                // v2.0.142 may already have created the active payment-stage row
                // with the gross pre-discount amount. Keep unpaid stages aligned
                // with the invoice's authoritative due amount, otherwise the
                // gateway callback can reject the correctly discounted payment as
                // a stage/payment amount mismatch.
                $stage_table = $wpdb->prefix . 'sn_invoice_payment_stages';
                if ( $this->table_exists( $stage_table ) ) {
                    $stage_repair = $wpdb->query( "UPDATE {$stage_table} s
                        INNER JOIN {$invoice_table} i ON i.id=s.invoice_id
                            AND s.stage_no=GREATEST(1, COALESCE(i.current_payment_stage, 1))
                        SET s.requested_amount=LEAST(
                                GREATEST(0, COALESCE(i.current_due_amount, i.final_total)),
                                GREATEST(0, i.final_total - COALESCE(i.paid_total_amount, 0))
                            ),
                            s.stage_type=CASE
                                WHEN LEAST(
                                    GREATEST(0, COALESCE(i.current_due_amount, i.final_total)),
                                    GREATEST(0, i.final_total - COALESCE(i.paid_total_amount, 0))
                                ) >= GREATEST(0, i.final_total - COALESCE(i.paid_total_amount, 0)) - 0.01
                                THEN 'full'
                                ELSE s.stage_type
                            END,
                            s.updated_at=NOW()
                        WHERE i.catalog_discounts_snapshot IS NOT NULL
                          AND i.catalog_discounts_snapshot <> ''
                          AND GREATEST(COALESCE(i.discount_total, 0), COALESCE(i.discount_amount, 0)) > 0
                          AND i.final_total IS NOT NULL
                          AND i.final_total > 0
                          AND s.status NOT IN ('paid','approved')
                          AND ABS(s.requested_amount - LEAST(
                                GREATEST(0, COALESCE(i.current_due_amount, i.final_total)),
                                GREATEST(0, i.final_total - COALESCE(i.paid_total_amount, 0))
                              )) > 0.01" );
                    if ( $stage_repair === false ) { $ok = false; }
                }
            }
            if ( $ok ) { update_option( 'sn_sales_catalog_schema_version', self::SCHEMA_VERSION, false ); }
        }
        return $ok;
    }

    /** Current HR reports_to chain wins; legacy metadata is used only without HR. */
    public function manager_for_user( int $user_id ): int {
        if ( $user_id < 1 ) { return 0; }
        if ( array_key_exists( $user_id, $this->manager_cache ) ) { return $this->manager_cache[ $user_id ]; }
        global $wpdb;
        $profiles = $wpdb->prefix . 'sn_hr_profiles';
        $positions = $wpdb->prefix . 'sn_hr_positions';
        $assignments = $wpdb->prefix . 'sn_hr_assignments';
        if ( $this->table_exists( $profiles ) && $this->table_exists( $positions ) && $this->table_exists( $assignments ) ) {
            $profile = $wpdb->get_row( $wpdb->prepare( "SELECT p.id,p.user_id,pos.slug FROM {$profiles} p LEFT JOIN {$positions} pos ON pos.id=p.position_id WHERE p.user_id=%d ORDER BY p.id DESC LIMIT 1", $user_id ) );
            if ( $profile ) {
                $seen = [];
                for ( $depth = 0; $profile && $depth < 20; $depth++ ) {
                    if ( isset( $seen[ (int) $profile->id ] ) ) { break; }
                    $seen[ (int) $profile->id ] = true;
                    if ( (string) $profile->slug === 'sales_manager' ) { return $this->manager_cache[ $user_id ] = (int) $profile->user_id; }
                    $profile = $wpdb->get_row( $wpdb->prepare( "SELECT p.id,p.user_id,pos.slug FROM {$assignments} a INNER JOIN {$profiles} p ON p.id=a.parent_profile_id LEFT JOIN {$positions} pos ON pos.id=p.position_id WHERE a.child_profile_id=%d AND a.relationship_type='reports_to' AND a.is_current=1 ORDER BY a.id DESC LIMIT 1", (int) $profile->id ) );
                }
                return $this->manager_cache[ $user_id ] = 0;
            }
        }
        $seen = []; $cursor = $user_id;
        for ( $depth = 0; $cursor > 0 && $depth < 12; $depth++ ) {
            if ( isset( $seen[ $cursor ] ) ) { break; }
            $seen[ $cursor ] = true;
            $user = get_user_by( 'id', $cursor );
            if ( $user && array_intersect( [ 'sn_sales_manager', 'sas_sales_manager' ], (array) $user->roles ) ) { return $this->manager_cache[ $user_id ] = $cursor; }
            $parent = 0;
            foreach ( [ 'sn_supervisor_id', 'sn_senior_supervisor_id', 'sn_sales_manager_id' ] as $key ) {
                $candidate = absint( get_user_meta( $cursor, $key, true ) );
                if ( $candidate > 0 && $candidate !== $cursor ) { $parent = $candidate; break; }
            }
            $cursor = $parent;
        }
        return $this->manager_cache[ $user_id ] = 0;
    }

    public function managers(): array {
        global $wpdb;
        $users = get_users( [ 'role__in' => [ 'sn_sales_manager', 'sas_sales_manager' ], 'fields' => 'ids', 'number' => -1 ] );
        $profiles = $wpdb->prefix . 'sn_hr_profiles'; $positions = $wpdb->prefix . 'sn_hr_positions';
        if ( $this->table_exists( $profiles ) && $this->table_exists( $positions ) ) {
            $users = array_merge( $users, (array) $wpdb->get_col( "SELECT p.user_id FROM {$profiles} p INNER JOIN {$positions} pos ON pos.id=p.position_id WHERE pos.slug='sales_manager'" ) );
        }
        $out = [];
        foreach ( array_unique( array_map( 'intval', $users ) ) as $id ) {
            $user = get_user_by( 'id', $id );
            if ( $user && $this->manager_for_user( $id ) === $id ) { $out[] = [ 'id' => $id, 'name' => (string) $user->display_name ]; }
        }
        usort( $out, static fn( $a, $b ) => strcmp( $a['name'], $b['name'] ) );
        return $out;
    }

    public function policy( int $manager_id ): array {
        $raw = $manager_id > 0 ? get_user_meta( $manager_id, self::META_KEY, true ) : [];
        return is_array( $raw ) ? array_merge( [ 'mode' => 'all', 'products' => [], 'revision' => '' ], $raw ) : [ 'mode' => 'all', 'products' => [], 'revision' => '' ];
    }

    public function product_allowed( int $product_id, int $user_id ): bool {
        if ( $user_id < 1 || user_can( $user_id, 'manage_options' ) ) { return true; }
        $policy = $this->policy( $this->manager_for_user( $user_id ) );
        return $policy['mode'] !== 'selected' || array_key_exists( $product_id, (array) $policy['products'] );
    }

    public function filter_products( array $products, int $user_id ): array {
        return array_values( array_filter( $products, fn( $product ) => $this->product_allowed( absint( $product['id'] ?? 0 ), $user_id ) ) );
    }

    public function validate_products( array $product_ids, int $actor_id, int $owner_id = 0 ) {
        foreach ( $product_ids as $product_id ) {
            if ( ! $this->product_allowed( (int) $product_id, $actor_id ) || ( $owner_id > 0 && ! $this->product_allowed( (int) $product_id, $owner_id ) ) ) {
                return new WP_Error( 'sn_catalog_product_forbidden', 'محصول «' . get_the_title( (int) $product_id ) . '» در فهرست محصولات مجاز مدیر فروش این نیرو نیست.' );
            }
        }
        return true;
    }

    /** Decimal money is validated strictly, then compared as integer hundredths. */
    public static function amount_key( $raw ): string {
        if ( ! is_scalar( $raw ) ) { return ''; }
        $value = str_replace( [ ',', '٬', ' ', '٫' ], [ '', '', '', '.' ], SN_Helpers::to_english_nums( trim( (string) $raw ) ) );
        if ( ! preg_match( '/^\d{1,12}(?:\.\d{1,2})?$/D', $value ) ) { return ''; }
        $parts = explode( '.', $value );
        $cents = (int) $parts[0] * 100 + (int) str_pad( $parts[1] ?? '', 2, '0' );
        return $cents > 0 ? (string) $cents : '';
    }

    private function discount_manager( int $actor_id, int $owner_id ): int {
        return user_can( $actor_id, 'manage_options' ) && $owner_id > 0 ? $this->manager_for_user( $owner_id ) : $this->manager_for_user( $actor_id );
    }

    public function discounts( int $product_id, int $actor_id, int $owner_id = 0 ): array {
        if ( is_wp_error( $this->validate_products( [ $product_id ], $actor_id, $owner_id ) ) ) { return []; }
        $policy = $this->policy( $this->discount_manager( $actor_id, $owner_id ) );
        $out = [];
        foreach ( (array) ( $policy['products'][ $product_id ]['discounts'] ?? [] ) as $key ) {
            if ( preg_match( '/^\d+$/D', (string) $key ) && (int) $key > 0 ) { $out[] = [ 'id' => (string) $key, 'amount' => (int) $key / 100, 'label' => SN_Helpers::format_price( (int) $key / 100 ) ]; }
        }
        return $out;
    }

    /** Select one fixed discount for each line; never trust a client-supplied amount. */
    public function price_items( array $items, $selections, int $actor_id, int $owner_id = 0 ) {
        if ( ! is_array( $selections ) || count( $selections ) > count( $items ) ) { return new WP_Error( 'sn_catalog_discount_input', 'گزینه‌های تخفیف معتبر نیستند.' ); }
        foreach ( $selections as $index => $value ) {
            if ( ! preg_match( '/^\d+$/D', (string) $index ) || ! array_key_exists( (int) $index, $items ) || ! is_scalar( $value ) ) { return new WP_Error( 'sn_catalog_discount_input', 'گزینه‌های تخفیف معتبر نیستند.' ); }
        }
        $valid = $this->validate_products( array_column( $items, 'product_id' ), $actor_id, $owner_id );
        if ( is_wp_error( $valid ) ) { return $valid; }
        $gross = 0; $discount = 0; $snapshot = [];
        foreach ( $items as $index => $item ) {
            $qty = max( 1, (int) ( $item['qty'] ?? 1 ) );
            $unit_cents = (int) round( (float) ( $item['unit_price'] ?? 0 ) * 100 );
            if ( $unit_cents <= 0 || $qty > 1000000 || $unit_cents > 100000000000000 || $unit_cents > intdiv( 999999999999999999 - $gross, $qty ) ) { return new WP_Error( 'sn_catalog_price', 'قیمت یا تعداد محصول معتبر نیست.' ); }
            $line_gross = $unit_cents * $qty; $gross += $line_gross;
            $selection = trim( (string) ( $selections[ $index ] ?? '' ) );
            if ( $selection === '' || $selection === '0' ) { continue; }
            $matched = null;
            foreach ( $this->discounts( (int) $item['product_id'], $actor_id, $owner_id ) as $option ) { if ( $selection === $option['id'] ) { $matched = $option; break; } }
            if ( ! $matched ) { return new WP_Error( 'sn_catalog_discount_forbidden', 'تخفیف انتخاب‌شده برای این محصول و مدیر فروش مجاز نیست؛ فهرست تخفیف‌ها را تازه‌سازی کنید.' ); }
            $unit_discount = (int) $matched['id'];
            if ( $unit_discount >= $unit_cents ) { return new WP_Error( 'sn_catalog_discount_excess', 'تخفیف هر واحد باید کمتر از قیمت همان محصول باشد.' ); }
            $line_discount = $unit_discount * $qty; $discount += $line_discount;
            $snapshot[] = [ 'line_index' => $index, 'product_id' => (int) $item['product_id'], 'product_name' => (string) ( $item['product_name'] ?? '' ), 'qty' => $qty, 'option_id' => $selection, 'unit_discount' => $unit_discount / 100, 'discount_total' => $line_discount / 100, 'original_unit_price' => $unit_cents / 100, 'original_line_total' => $line_gross / 100, 'final_line_total' => ( $line_gross - $line_discount ) / 100, 'manager_user_id' => $this->discount_manager( $actor_id, $owner_id ) ];
        }
        return [ 'original_total' => $gross / 100, 'discount_total' => $discount / 100, 'final_total' => ( $gross - $discount ) / 100, 'snapshot' => $snapshot, 'manager_user_id' => $this->discount_manager( $actor_id, $owner_id ) ];
    }

    public static function invoice_snapshot( $invoice ): array {
        $raw = is_array( $invoice ) ? ( $invoice['catalog_discounts_snapshot'] ?? '' ) : ( $invoice->catalog_discounts_snapshot ?? '' );
        $snapshot = json_decode( (string) $raw, true );
        return is_array( $snapshot ) ? $snapshot : [];
    }

    public function admin_menu(): void {
        add_submenu_page( 'sn-dashboard', 'تخفیف‌ها و محصولات', 'تخفیف‌ها و محصولات', 'manage_options', 'sn-sales-catalog', [ $this, 'render_page' ] );
    }

    public function admin_assets(): void {
        if ( ! current_user_can( 'manage_options' ) || ! in_array( sanitize_key( $_GET['page'] ?? '' ), [ 'sn-settings', 'sn-sales-catalog' ], true ) ) { return; }
        wp_enqueue_style( 'sn-sales-catalog', SN_PLUGIN_URL . 'assets/css/sales-catalog.css', [], SN_VERSION );
        wp_enqueue_script( 'sn-sales-catalog-admin', SN_PLUGIN_URL . 'assets/js/sales-catalog-admin.js', [ 'jquery' ], SN_VERSION, true );
        wp_localize_script( 'sn-sales-catalog-admin', 'snSalesCatalogAdmin', [ 'ajaxUrl' => admin_url( 'admin-ajax.php' ), 'nonce' => wp_create_nonce( 'sn_sales_catalog_admin' ) ] );
    }

    public function staff_assets(): void {
        if ( wp_script_is( 'sn-sales-catalog-staff', 'enqueued' ) ) { return; }
        if ( ! is_user_logged_in() || ! wp_script_is( 'sn-public-seller', 'enqueued' ) && ! wp_script_is( 'sn-public-converter', 'enqueued' ) && ! wp_script_is( 'sn-public-supervisor', 'enqueued' ) && ! wp_script_is( 'sn-public-base', 'enqueued' ) && ! wp_script_is( 'sn-public-dot', 'enqueued' ) ) { return; }
        wp_enqueue_style( 'sn-sales-catalog', SN_PLUGIN_URL . 'assets/css/sales-catalog.css', [], SN_VERSION );
        wp_enqueue_script( 'sn-sales-catalog-staff', SN_PLUGIN_URL . 'assets/js/sales-catalog-staff.js', [ 'jquery' ], SN_VERSION, true );
        wp_localize_script( 'sn-sales-catalog-staff', 'snSalesCatalog', [ 'ajaxUrl' => admin_url( 'admin-ajax.php' ), 'nonce' => wp_create_nonce( 'sn_sales_catalog_staff' ) ] );
    }

    public function render_page(): void {
        if ( ! current_user_can( 'manage_options' ) ) { wp_die( 'دسترسی مجاز نیست.' ); }
        echo '<div class="wrap" dir="rtl"><h1>تخفیف‌ها و محصولات</h1>';
        $this->render_settings_panel();
        echo '</div>';
    }

    public function render_settings_panel(): void {
        if ( ! current_user_can( 'manage_options' ) ) { return; }
        $managers = $this->managers();
        ?>
        <section class="sn-sales-catalog" dir="rtl">
            <div class="sn-catalog-heading"><h2>محصولات و تخفیف‌های تیم فروش</h2><p>محصولات مجاز مدیر فروش و تمام نیروهای زیرمجموعه‌اش را تعیین کنید. برای هر محصول چند مبلغ تخفیف ثابت به تومان تعریف کنید؛ در پیش‌فاکتور یک گزینه برای هر ردیف قابل انتخاب است.</p></div>
            <div class="sn-catalog-toolbar"><label>مدیر فروش<select class="sn-catalog-manager"><option value="">انتخاب مدیر فروش</option><?php foreach ( $managers as $manager ) : ?><option value="<?php echo (int) $manager['id']; ?>"><?php echo esc_html( $manager['name'] . ' — #' . $manager['id'] ); ?></option><?php endforeach; ?></select></label><button type="button" class="button sn-catalog-load">بارگذاری</button></div>
            <?php if ( ! $managers ) : ?><p>مدیر فروشی یافت نشد. ابتدا مدیر فروش و ارتباط نیروهایش را در منابع انسانی تعریف کنید.</p><?php endif; ?>
            <div class="sn-catalog-notice" role="status" aria-live="polite"></div>
            <div class="sn-catalog-editor" hidden>
                <div class="sn-catalog-mode"><label><input type="radio" class="sn-catalog-mode-input" name="sn_catalog_mode_ui" value="all"> همه محصولات CRM</label><label><input type="radio" class="sn-catalog-mode-input" name="sn_catalog_mode_ui" value="selected"> فقط محصولات انتخاب‌شده</label></div>
                <p>تخفیف ثابت برای هر واحد محصول محاسبه می‌شود. فروشنده می‌تواند «بدون تخفیف» یا یکی از مبالغ مجاز را انتخاب کند. محدودیت‌های قبلی نوع محصول و پروژه نیز اعمال می‌شوند.</p>
                <div class="sn-catalog-tools"><input type="search" class="sn-catalog-search" placeholder="جستجوی نام محصول"><button type="button" class="button sn-catalog-select-all">انتخاب همه</button><button type="button" class="button sn-catalog-select-none">لغو انتخاب همه</button></div>
                <div class="sn-catalog-table-wrap"><table class="widefat striped sn-catalog-table"><thead><tr><th>اجازه فروش</th><th>محصول CRM</th><th>قیمت فعلی</th><th>تخفیف‌های ثابت هر واحد (تومان)</th></tr></thead><tbody></tbody></table></div>
                <p class="sn-catalog-footnote">در حالت «فقط محصولات انتخاب‌شده»، خالی‌بودن فهرست یعنی هیچ محصولی مجاز نیست. مدیرانی که هنوز تنظیم نشده‌اند، همه محصولات فعلی را بدون تخفیف جدید می‌بینند.</p>
                <button type="button" class="button button-primary sn-catalog-save">ذخیره محصولات و تخفیف‌های این مدیر</button>
            </div>
        </section>
        <?php
    }

    private function verify_admin(): void {
        if ( ! current_user_can( 'manage_options' ) ) { wp_send_json_error( [ 'message' => 'دسترسی مجاز نیست.' ], 403 ); }
        check_ajax_referer( 'sn_sales_catalog_admin', 'nonce' );
    }

    public function ajax_load(): void {
        $this->verify_admin();
        $id = absint( $_POST['manager_id'] ?? 0 );
        if ( $id < 1 || ! get_user_by( 'id', $id ) || $this->manager_for_user( $id ) !== $id ) { wp_send_json_error( [ 'message' => 'مدیر فروش معتبر نیست.' ], 400 ); }
        wp_send_json_success( [ 'manager_id' => $id, 'policy' => $this->policy( $id ), 'products' => SN_Helpers::get_sn_products( 0, true ) ] );
    }

    public function ajax_save(): void {
        $this->verify_admin();
        $id = absint( $_POST['manager_id'] ?? 0 );
        if ( $id < 1 || ! get_user_by( 'id', $id ) || $this->manager_for_user( $id ) !== $id ) { wp_send_json_error( [ 'message' => 'مدیر فروش معتبر نیست.' ], 400 ); }
        $raw = wp_unslash( $_POST['policy'] ?? '' );
        if ( ! is_string( $raw ) || strlen( $raw ) > 1000000 ) { wp_send_json_error( [ 'message' => 'حجم تنظیمات معتبر نیست.' ], 400 ); }
        $input = json_decode( $raw, true );
        if ( ! is_array( $input ) || ! in_array( $input['mode'] ?? '', [ 'all', 'selected' ], true ) || ! is_array( $input['products'] ?? null ) ) { wp_send_json_error( [ 'message' => 'ساختار تنظیمات معتبر نیست.' ], 400 ); }
        $previous_meta = get_user_meta( $id, self::META_KEY, true );
        $old = is_array( $previous_meta ) ? array_merge( [ 'revision' => '' ], $previous_meta ) : [ 'revision' => '' ];
        if ( ! hash_equals( (string) $old['revision'], (string) ( $input['revision'] ?? '' ) ) ) { wp_send_json_error( [ 'message' => 'تنظیمات توسط مدیر دیگری تغییر کرده؛ دوباره بارگذاری کنید.' ], 409 ); }
        $available = [];
        foreach ( SN_Helpers::get_sn_products( 0, true ) as $product ) { $available[ (int) $product['id'] ] = $product; }
        $products = [];
        foreach ( $input['products'] as $pid => $row ) {
            $pid = absint( $pid );
            if ( ! isset( $available[ $pid ] ) || ! is_array( $row ) || ! is_array( $row['discounts'] ?? [] ) ) { wp_send_json_error( [ 'message' => 'یکی از محصولات دیگر در CRM فعال نیست؛ دوباره بارگذاری کنید.' ], 400 ); }
            if ( count( $row['discounts'] ?? [] ) > 30 ) { wp_send_json_error( [ 'message' => 'برای هر محصول حداکثر ۳۰ مبلغ تخفیف تعریف کنید.' ], 400 ); }
            $discounts = [];
            foreach ( $row['discounts'] ?? [] as $amount ) {
                if ( $amount === '' ) { continue; }
                $key = self::amount_key( $amount );
                if ( $key === '' || (int) $key >= (int) round( (float) $available[ $pid ]['price'] * 100 ) ) { wp_send_json_error( [ 'message' => 'تخفیف محصول «' . $available[ $pid ]['name'] . '» باید مثبت و کمتر از قیمت هر واحد باشد.' ], 400 ); }
                $discounts[ $key ] = (string) $key;
            }
            $discounts = array_values( $discounts );
            usort( $discounts, static fn( $a, $b ) => (int) $a <=> (int) $b );
            $products[ $pid ] = [ 'discounts' => $discounts ];
        }
        $policy = [ 'mode' => $input['mode'], 'products' => $products, 'revision' => wp_generate_uuid4(), 'updated_by' => get_current_user_id(), 'updated_at' => current_time( 'mysql' ) ];
        // Compare against the exact version read before validation.
        $saved = $previous_meta === '' ? add_user_meta( $id, self::META_KEY, $policy, true ) : update_user_meta( $id, self::META_KEY, $policy, $previous_meta );
        if ( ! $saved || get_user_meta( $id, self::META_KEY, true ) !== $policy ) { wp_send_json_error( [ 'message' => 'ذخیره انجام نشد یا تنظیمات هم‌زمان تغییر کرده؛ دوباره بارگذاری کنید.' ], 409 ); }
        wp_send_json_success( [ 'message' => 'محصولات و تخفیف‌های مدیر فروش و نیروهایش ذخیره شد.', 'policy' => $policy ] );
    }

    private function staff_user( int $id ): bool {
        $user = get_user_by( 'id', $id );
        if ( ! $user ) { return false; }
        if ( user_can( $id, 'manage_options' ) ) { return true; }
        if ( class_exists( 'SN_HR_Service' ) ) {
            $resolved = ( new SN_HR_Service() )->resolve_panel_for_user( $id );
            if ( in_array( $resolved['position_slug'] ?? '', [ 'seller', 'converter', 'supervisor', 'senior_supervisor', 'sales_manager', 'sales_deputy' ], true ) ) { return true; }
        }
        return (bool) array_intersect( [ 'sn_seller', 'sn_converter', 'sn_supervisor', 'sn_senior_supervisor', 'sn_sales_manager', 'sn_sales_deputy' ], (array) $user->roles );
    }

    public function ajax_staff(): void {
        check_ajax_referer( 'sn_sales_catalog_staff', 'nonce' );
        $actor = get_current_user_id();
        if ( ! $this->staff_user( $actor ) ) { wp_send_json_error( [ 'message' => 'دسترسی مجاز نیست.' ], 403 ); }
        $owner = absint( $_POST['seller_id'] ?? 0 ) ?: $actor;
        if ( $owner !== $actor && ! current_user_can( 'manage_options' ) ) {
            $allowed = false;
            if ( class_exists( 'SN_Hierarchy_Service' ) && class_exists( 'SN_HR_Service' ) ) {
                $profile = ( new SN_HR_Service() )->get_profile_by_user_id( $actor );
                $target = ( new SN_HR_Service() )->get_profile_by_user_id( $owner );
                $allowed = $profile && $target && in_array( (int) $target->id, ( new SN_Hierarchy_Service() )->get_descendant_profile_ids( (int) $profile->id, 20 ), true );
            }
            if ( ! $allowed ) { wp_send_json_error( [ 'message' => 'این نیرو در محدوده شما نیست.' ], 403 ); }
        }
        $products = [];
        foreach ( SN_Helpers::get_sn_products( 0, true ) as $product ) {
            if ( is_wp_error( $this->validate_products( [ (int) $product['id'] ], $actor, $owner ) ) ) { continue; }
            $product['discounts'] = $this->discounts( (int) $product['id'], $actor, $owner );
            $products[] = $product;
        }
        wp_send_json_success( [ 'products' => $products, 'manager_id' => $this->discount_manager( $actor, $owner ) ] );
    }

    public function render_case_discount( object $case, array $options, bool $has_payment, int $actor_id ): void {
        if ( $has_payment ) {
            $snapshot = json_decode( (string) ( $case->catalog_discount_snapshot ?? '' ), true );
            if ( ! empty( $snapshot['discount_total'] ) ) { echo '<p class="sn-catalog-case-discount">تخفیف ثبت‌شده محصول: <strong>' . esc_html( SN_Helpers::format_price( (float) $snapshot['discount_total'] ) ) . '</strong></p>'; }
            return;
        }
        $map = [];
        foreach ( $options as $option ) {
            if ( empty( $option['active'] ) ) { continue; }
            $pid = absint( $option['product_id'] ?? 0 );
            if ( $pid > 0 ) { $map[ (string) $option['key'] ] = $this->discounts( $pid, $actor_id ); }
        }
        ?><label class="sn-catalog-case-discount" data-selected="<?php $saved = json_decode( (string) ( $case->catalog_discount_snapshot ?? '' ), true ); echo esc_attr( (string) ( $saved['option_id'] ?? '' ) ); ?>" data-discounts="<?php echo esc_attr( wp_json_encode( $map ) ); ?>"><span>تخفیف محصول نهایی</span><select name="catalog_discount_id"><option value="">بدون تخفیف</option></select><small>پس از ثبت اولین لینک پرداخت، تخفیف این پرونده ثابت می‌ماند.</small></label><?php
    }
}
