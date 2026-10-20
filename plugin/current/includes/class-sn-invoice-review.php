<?php
if (!defined('ABSPATH')) { exit; }

/** Read-only, cross-team invoice inspection. No payment/mutation privileges. */
final class SN_Invoice_Review
{
    private const CAP = 'sn_review_invoices';
    private const TOTAL = "GREATEST(0,CASE WHEN COALESCE(i.catalog_discounts_snapshot,'')<>'' AND COALESCE(NULLIF(i.discount_total,0),i.discount_amount,0)>0 AND i.final_total>0 THEN i.final_total ELSE COALESCE(NULLIF(i.payment_total_amount,0),NULLIF(i.final_total,0),i.product_price,0) END)";

    // Match the current status shown by the CRM; legacy columns are fallbacks,
    // not independent historical matches that can reopen a completed invoice.
    private const STATUS = "COALESCE(NULLIF(i.status,''),NULLIF(i.invoice_status,''),NULLIF(i.payment_status,''),'')";
    private const ONLINE = "COALESCE(i.pay_method,'') IN ('online','gateway','zibal','zarinpal','asanpardakht')";

    private static function status_condition(string $status): array
    {
        $current = self::STATUS;
        $online = self::ONLINE;
        switch ($status) {
            case 'paid':
                // Gateway callbacks finalize as approved; partial payments remain partial_paid.
                return ["({$online} AND {$current} IN ('paid','approved','partial_paid'))", []];
            case 'needs_review':
                return ["{$current} IN ('receipt_uploaded','pending_financial_approval')", []];
            case 'pending_payment':
                // Issuing a stage deliberately stores pre_invoice + pending_payment.
                return ["({$current}='pending_payment' OR ({$current} IN ('pre_invoice','pending') AND i.payment_status='pending_payment'))", []];
            case 'awaiting_next_payment':
                return ["({$current}='awaiting_next_payment' OR ({$current}='partial_paid' AND i.payment_workflow_status IN ('awaiting_assignment','awaiting_next_payment')))", []];
            case 'payment_stage_issued':
                // The distribution queue owns payment_stage_issued; invoices use awaiting_payment.
                return ["({$current}='payment_stage_issued' OR ({$current} IN ('pre_invoice','pending','pending_payment') AND i.payment_workflow_status='awaiting_payment' AND i.current_payment_stage>1 AND i.current_due_amount>0))", []];
            default:
                return ["{$current}=%s", [$status]];
        }
    }

    public static function register_hooks(): void
    {
        add_action('init', [self::class, 'ensure_role'], 6);
        add_filter('login_redirect', [self::class, 'login_redirect'], 20, 3);
        add_filter('logout_redirect', [self::class, 'logout_redirect'], 20, 3);
        add_action('admin_init', [self::class, 'ensure_page'], 3);
        add_action('wp_enqueue_scripts', [self::class, 'enqueue_assets']);
        add_action('admin_post_sn_invoice_review_export', [self::class, 'export']);
        add_action('template_redirect', [self::class, 'protect_cache'], 1);
    }

    public static function login_redirect(string $redirect, string $requested, $user): string
    {
        if ($user instanceof WP_User && in_array('sn_invoice_reviewer', (array)$user->roles, true) && !user_can($user, 'manage_options')) {
            // wp-login.php uses the lightweight bootstrap; use the shared HR-aware resolver.
            return sn_bootstrap_plugin_instance(false)->seller_login_redirect($redirect, $requested, $user);
        }
        return $redirect;
    }

    public static function logout_redirect(string $redirect, string $requested, $user): string
    {
        return $user instanceof WP_User && in_array('sn_invoice_reviewer', (array)$user->roles, true)
            ? sn_crm_login_url() : $redirect;
    }

    public static function ensure_role(): void
    {
        // Works on ZIP replacement too, without requiring reactivation.
        if (!get_role('sn_invoice_reviewer')) {
            add_role('sn_invoice_reviewer', 'بررسی فاکتور', ['read' => true, self::CAP => true]);
        }
    }

    public static function ensure_page(): void
    {
        if (wp_doing_ajax() || !current_user_can('manage_options')) { return; }
        $id = (int)get_option('sn_invoice_review_panel_page_id', 0);
        if ($id && get_post_status($id) === 'publish') { return; }
        $page = get_page_by_path('crm-invoice-review');
        if ($page && $page->post_status === 'publish' && has_shortcode((string)$page->post_content, 'sn_invoice_review_panel')) {
            update_option('sn_invoice_review_panel_page_id', (int)$page->ID, false);
            return;
        }
        // Do not overwrite an unrelated page using the same slug.
        $id = wp_insert_post(['post_type'=>'page','post_status'=>'publish','post_title'=>'پنل بررسی فاکتور',
            'post_name'=>'crm-invoice-review','post_content'=>'[sn_invoice_review_panel]',
            'comment_status'=>'closed','ping_status'=>'closed'], true);
        if (is_wp_error($id) || !$id) { return; }
        update_option('sn_invoice_review_panel_page_id', (int)$id, false);
        update_post_meta((int)$id, '_sn_system_page', '1');
        update_post_meta((int)$id, '_wp_page_template', 'elementor_canvas');
    }

    public static function enqueue_assets(): void
    {
        global $post;
        if (!$post) { return; }
        $elementor = (string)get_post_meta((int)$post->ID, '_elementor_data', true);
        if (has_shortcode((string)$post->post_content, 'sn_invoice_review_panel') || strpos($elementor, 'sn_invoice_review_panel') !== false) {
            wp_enqueue_style('sn-invoice-review', SN_PLUGIN_URL . 'assets/css/invoice-review.css', [], SN_VERSION);
            wp_enqueue_script('sn-invoice-review', SN_PLUGIN_URL . 'assets/js/invoice-review.js', [], SN_VERSION, true);
        }
    }

    public static function allowed(): bool
    {
        return is_user_logged_in() && (current_user_can('manage_options') || current_user_can(self::CAP));
    }

    public static function protect_cache(): void
    {
        $id = (int)get_option('sn_invoice_review_panel_page_id', 0);
        global $post;
        if (($id && is_page($id)) || ($post && (has_shortcode((string)$post->post_content, 'sn_invoice_review_panel') || strpos((string)get_post_meta((int)$post->ID, '_elementor_data', true), 'sn_invoice_review_panel') !== false))) {
            if (!defined('DONOTCACHEPAGE')) { define('DONOTCACHEPAGE', true); }
            nocache_headers();
        }
    }

    private static function input(array $source, string $key, string $default = ''): string
    {
        $value = $source['ir_' . $key] ?? $default;
        return is_scalar($value) ? sanitize_text_field(wp_unslash((string)$value)) : $default;
    }

    private static function statuses(): array
    {
        return ['' => 'همه وضعیت‌ها', 'needs_review' => 'نیازمند بررسی مالی', 'pre_invoice' => 'پیش‌فاکتور',
            'pending' => 'در انتظار پرداخت', 'pending_payment' => 'در انتظار پرداخت (مرحله‌ای)',
            'receipt_uploaded' => 'فیش بارگذاری‌شده', 'pending_financial_approval' => 'در انتظار تأیید مالی',
            'paid' => 'پرداخت‌شده درگاهی', 'approved' => 'تأییدشده (مالی / خودکار درگاه)', 'rejected' => 'ردشده',
            'cancelled' => 'لغوشده', 'partial_paid' => 'پیش‌پرداخت تأییدشده',
            'awaiting_next_payment' => 'در انتظار مرحله بعد', 'payment_stage_issued' => 'مرحله پرداخت صادرشده',
            'payment_archived' => 'بایگانی انصرافی'];
    }

    private static function filters(array $source): array
    {
        $f = [];
        foreach (['search','status','method','plan','seller','product','from','to','min','max','sort','size','page'] as $key) {
            $f[$key] = SN_Helpers::to_english_nums(self::input($source, $key));
        }
        $f['errors'] = [];
        if (!array_key_exists($f['status'], self::statuses())) { $f['errors'][] = 'وضعیت انتخاب‌شده معتبر نیست.'; }
        foreach (['method' => ['', 'online', 'card'], 'plan' => ['', 'full', 'staged']] as $key => $allowed) {
            if (!in_array($f[$key], $allowed, true)) { $f['errors'][] = 'فیلتر روش یا نوع پرداخت معتبر نیست.'; }
        }
        $f['seller'] = absint($f['seller']);
        $f['product'] = absint($f['product']);
        $f['sort'] = in_array($f['sort'], ['newest','oldest','amount_desc','amount_asc'], true) ? $f['sort'] : 'newest';
        $f['size'] = in_array((int)$f['size'], [20,50,100], true) ? (int)$f['size'] : 20;
        $f['page'] = max(1, min(100000000, (int)$f['page']));
        foreach (['min','max'] as $key) {
            if ($f[$key] !== '') {
                $amount = SN_Payment_Transactions::parse_amount($f[$key]);
                if (!is_finite($amount) || $amount < 0 || $amount > 9999999999999999) { $f['errors'][] = 'محدوده مبلغ را به‌درستی وارد کنید.'; }
                else { $f[$key] = (string)$amount; }
            }
        }
        if ($f['min'] !== '' && $f['max'] !== '' && (float)$f['min'] > (float)$f['max']) { $f['errors'][] = 'حداقل مبلغ نباید بیشتر از حداکثر باشد.'; }
        foreach (['from','to'] as $key) {
            $f[$key . '_sql'] = '';
            if ($f[$key] === '') { continue; }
            $date = SN_Helpers::jalali_to_gregorian_date($f[$key]);
            if ($date === '') { $f['errors'][] = 'تاریخ را به‌درستی وارد کنید؛ نمونه: ۱۴۰۵/۰۷/۰۱'; continue; }
            // Inputs/display are Tehran; persisted DATETIME follows the WordPress timezone.
            $dt = new DateTimeImmutable($date . ' 00:00:00', new DateTimeZone('Asia/Tehran'));
            if ($key === 'to') { $dt = $dt->modify('+1 day'); }
            $f[$key . '_sql'] = $dt->setTimezone(wp_timezone())->format('Y-m-d H:i:s');
        }
        if ($f['from_sql'] !== '' && $f['to_sql'] !== '' && $f['from_sql'] >= $f['to_sql']) { $f['errors'][] = 'تاریخ پایان باید برابر یا بعد از تاریخ شروع باشد.'; }
        return $f;
    }

    private static function where(array $f): array
    {
        global $wpdb;
        $parts = ['1=1']; $args = [];
        if ($f['search'] !== '') {
            $like = '%' . $wpdb->esc_like($f['search']) . '%';
            $parts[] = "(i.invoice_code LIKE %s OR i.customer_name LIKE %s OR i.customer_phone LIKE %s OR i.customer_phone_secondary LIKE %s OR i.city LIKE %s OR i.id=%d OR EXISTS (SELECT 1 FROM {$wpdb->prefix}sn_payments ps WHERE ps.invoice_id=i.id AND ps.ref_id LIKE %s))";
            array_push($args, $like, $like, $like, $like, $like, ctype_digit($f['search']) ? (int)$f['search'] : 0, $like);
        }
        if ($f['status'] !== '') {
            [$condition, $status_args] = self::status_condition($f['status']);
            $parts[] = '(' . $condition . ')';
            array_push($args, ...$status_args);
        }
        if ($f['method'] === 'online') { $parts[] = self::ONLINE; }
        if ($f['method'] === 'card') { $parts[] = "i.pay_method IN ('card','card_to_card','receipt','manual')"; }
        if ($f['plan'] === 'full') { $parts[] = "COALESCE(NULLIF(i.payment_plan,''),'full')='full'"; }
        if ($f['plan'] === 'staged') { $parts[] = "COALESCE(NULLIF(i.payment_plan,''),'full')<>'full'"; }
        if ($f['seller']) { $parts[] = 'i.seller_id=%d'; $args[] = $f['seller']; }
        if ($f['product']) {
            $parts[] = "(i.product_id=%d OR EXISTS (SELECT 1 FROM {$wpdb->prefix}sn_invoice_items pi WHERE pi.invoice_id=i.id AND pi.product_id=%d))";
            array_push($args, $f['product'], $f['product']);
        }
        foreach (['from' => '>=', 'to' => '<'] as $key => $operator) {
            if ($f[$key . '_sql'] !== '') { $parts[] = "i.created_at {$operator} %s"; $args[] = $f[$key . '_sql']; }
        }
        foreach (['min' => '>=', 'max' => '<='] as $key => $operator) {
            if ($f[$key] !== '') { $parts[] = self::TOTAL . " {$operator} %f"; $args[] = (float)$f[$key]; }
        }
        return [implode(' AND ', $parts), $args];
    }

    private static function query(string $sql, array $args): string
    {
        global $wpdb;
        return $args ? $wpdb->prepare($sql, $args) : $sql;
    }

    private static function order(array $f): string
    {
        return ['newest'=>'i.created_at DESC,i.id DESC','oldest'=>'i.created_at ASC,i.id ASC',
            'amount_desc'=>self::TOTAL.' DESC,i.id DESC','amount_asc'=>self::TOTAL.' ASC,i.id ASC'][$f['sort']];
    }

    private static function rows(array $f, int $offset, int $limit): array
    {
        global $wpdb;
        [$where, $args] = self::where($f);
        $sql = 'SELECT i.*, ' . self::TOTAL . " review_total, u.display_name seller_name FROM {$wpdb->prefix}sn_invoices i LEFT JOIN {$wpdb->users} u ON u.ID=i.seller_id WHERE {$where} ORDER BY " . self::order($f) . ' LIMIT %d OFFSET %d';
        array_push($args, $limit, $offset);
        return $wpdb->get_results(self::query($sql, $args), ARRAY_A) ?: [];
    }

    public static function url(): string
    {
        $id = (int)get_option('sn_invoice_review_panel_page_id', 0);
        return ($id ? get_permalink($id) : '') ?: home_url('/crm-invoice-review/');
    }

    private static function params(array $f): array
    {
        $out = [];
        foreach (['search','status','method','plan','seller','product','from','to','min','max','sort','size','page'] as $key) {
            if ($f[$key] !== '' && $f[$key] !== 0) { $out['ir_' . $key] = $f[$key]; }
        }
        return $out;
    }

    private static function money($value): string { return SN_Helpers::format_price((float)$value); }
    private static function date($value): string { return SN_Helpers::gregorian_to_jalali_date((string)$value); }
    private static function user($value): string
    {
        $id = (int)$value;
        if (!$id) { return '—'; }
        $user = get_userdata($id);
        return $user ? $user->display_name . ' (#' . $id . ')' : '#' . $id . ' (کاربر حذف‌شده)';
    }
    private static function label($value): string
    {
        $map = ['full'=>'یکجا','staged'=>'مرحله‌ای','installment'=>'اقساطی','installments'=>'اقساطی',
            'deposit'=>'پیش‌پرداخت','partial'=>'پیش‌پرداخت','awaiting_payment'=>'در انتظار پرداخت',
            'completed'=>'تکمیل‌شده','verified'=>'پرداخت درگاهی ثبت‌شده','failed'=>'ناموفق','expired'=>'منقضی',
            'bank_transfer'=>'انتقال بانکی','paya'=>'پایا','satna'=>'ساتنا','pol'=>'پل'];
        return $map[$value] ?? SN_Helpers::status_label((string)$value);
    }

    public static function render(): string
    {
        if (!self::allowed()) {
            return '<div class="sn-panel" dir="rtl"><p>برای مشاهده فاکتورها با حساب دارای دسترسی «بررسی فاکتور» وارد شوید.</p><a href="' . esc_url(sn_crm_login_url()) . '">ورود به CRM</a></div>';
        }
        if (!defined('DONOTCACHEPAGE')) { define('DONOTCACHEPAGE', true); }
        if (!headers_sent()) { nocache_headers(); }
        wp_enqueue_style('sn-invoice-review', SN_PLUGIN_URL . 'assets/css/invoice-review.css', [], SN_VERSION);
        wp_enqueue_script('sn-invoice-review', SN_PLUGIN_URL . 'assets/js/invoice-review.js', [], SN_VERSION, true);
        $f = self::filters($_GET);
        ob_start(); ?>
        <section class="sn-ir" dir="rtl" aria-label="پنل بررسی فاکتور">
            <header class="sn-ir-header"><div><small>شبکه فروش · دسترسی مشاهده</small><h1>بررسی فاکتورها</h1><p>همه فاکتورها، اقلام، مراحل پرداخت و فیش‌های ثبت‌شده</p></div><a class="sn-ir-btn sn-ir-secondary" href="<?php echo esc_url(wp_logout_url(sn_crm_login_url())); ?>">خروج</a></header>
            <?php
            $id = absint(self::input($_GET, 'invoice'));
            if ($id) { self::detail($id, $f); }
            else { self::listing($f); }
            ?>
        </section>
        <?php return ob_get_clean();
    }

    private static function select(string $key, string $label, array $options, $selected): void
    {
        echo '<label>' . esc_html($label) . '<select name="ir_' . esc_attr($key) . '">';
        foreach ($options as $value => $text) { echo '<option value="' . esc_attr((string)$value) . '" ' . selected((string)$selected, (string)$value, false) . '>' . esc_html($text) . '</option>'; }
        echo '</select></label>';
    }

    private static function listing(array $f): void
    {
        global $wpdb;
        $sellers = $wpdb->get_results("SELECT DISTINCT i.seller_id,u.display_name FROM {$wpdb->prefix}sn_invoices i LEFT JOIN {$wpdb->users} u ON u.ID=i.seller_id ORDER BY u.display_name,i.seller_id", ARRAY_A) ?: [];
        $options = [0=>'همه فروشندگان'];
        foreach ($sellers as $seller) { if ((int)$seller['seller_id'] > 0) { $options[(int)$seller['seller_id']] = ($seller['display_name'] ?: 'کاربر حذف‌شده') . ' (#' . $seller['seller_id'] . ')'; } }
        $products = $wpdb->get_results("SELECT product_id,MAX(product_name) name FROM {$wpdb->prefix}sn_invoice_items GROUP BY product_id UNION SELECT DISTINCT i.product_id,COALESCE(p.post_title,CONCAT('محصول #',i.product_id)) name FROM {$wpdb->prefix}sn_invoices i LEFT JOIN {$wpdb->posts} p ON p.ID=i.product_id WHERE NOT EXISTS (SELECT 1 FROM {$wpdb->prefix}sn_invoice_items ii WHERE ii.product_id=i.product_id) ORDER BY name", ARRAY_A) ?: [];
        $product_options = [0=>'همه محصولات'];
        foreach ($products as $product) { if ((int)$product['product_id'] > 0) { $product_options[(int)$product['product_id']] = $product['name']; } }
        ?>
        <form class="sn-ir-filters" method="get" action="<?php echo esc_url(self::url()); ?>">
            <?php // Preserve plain-permalink routing as well as pretty permalinks.
            $url_query = []; parse_str((string)parse_url(self::url(), PHP_URL_QUERY), $url_query);
            foreach ($url_query as $key => $value) { if (is_scalar($value)) { echo '<input type="hidden" name="'.esc_attr($key).'" value="'.esc_attr($value).'">'; } }
            ?>
            <label class="sn-ir-search">جست‌وجو<input type="search" name="ir_search" value="<?php echo esc_attr($f['search']); ?>" placeholder="کد یا شناسه فاکتور، نام، موبایل، شهر یا کد پیگیری"></label>
            <?php self::select('status','وضعیت',self::statuses(),$f['status']); self::select('seller','فروشنده',$options,$f['seller']); self::select('product','محصول',$product_options,$f['product']);
            self::select('method','روش پرداخت',[''=>'همه روش‌ها','online'=>'آنلاین','card'=>'کارت‌به‌کارت / ثبت دستی'],$f['method']);
            self::select('plan','نوع پرداخت',[''=>'همه انواع','full'=>'یکجا','staged'=>'مرحله‌ای / اقساطی'],$f['plan']); ?>
            <?php foreach (['from'=>'از تاریخ (شمسی)','to'=>'تا تاریخ (شمسی)','min'=>'حداقل مبلغ (تومان)','max'=>'حداکثر مبلغ (تومان)'] as $key=>$label): ?>
                <label><?php echo esc_html($label); ?><input name="ir_<?php echo esc_attr($key); ?>" value="<?php echo esc_attr($f[$key]); ?>" placeholder="<?php echo in_array($key,['from','to'],true) ? '۱۴۰۵/۰۷/۰۱' : '۰'; ?>" inputmode="<?php echo in_array($key,['from','to'],true) ? 'text' : 'decimal'; ?>"></label>
            <?php endforeach;
            self::select('sort','مرتب‌سازی',['newest'=>'جدیدترین','oldest'=>'قدیمی‌ترین','amount_desc'=>'بیشترین مبلغ','amount_asc'=>'کمترین مبلغ'],$f['sort']);
            self::select('size','تعداد در صفحه',[20=>'۲۰ فاکتور',50=>'۵۰ فاکتور',100=>'۱۰۰ فاکتور'],$f['size']); ?>
            <div class="sn-ir-actions"><button class="sn-ir-btn" type="submit">اعمال فیلتر و جست‌وجو</button><a class="sn-ir-btn sn-ir-secondary" href="<?php echo esc_url(self::url()); ?>">پاک‌کردن فیلترها</a></div>
        </form>
        <p class="sn-ir-hint">«پرداخت‌شده درگاهی» شامل پرداخت کامل یا پیش‌پرداخت تأییدشده با روش آنلاین است؛ به معنی تسویه کامل همه نتایج نیست. «تأییدشده» تأیید مالی و تأیید خودکار درگاه را شامل می‌شود. فیلترها وضعیت فعلی فاکتور را نشان می‌دهند.</p>
        <?php
        if ($f['errors']) { echo '<div class="sn-ir-error" role="alert">'.esc_html(implode(' ',array_unique($f['errors']))).'</div>'; return; }
        [$where,$args] = self::where($f);
        $summary = $wpdb->get_row(self::query('SELECT COUNT(*) count,COALESCE(SUM(' . self::TOTAL . "),0) total,COALESCE(SUM(i.paid_total_amount),0) paid FROM {$wpdb->prefix}sn_invoices i WHERE {$where}",$args), ARRAY_A);
        if ($wpdb->last_error || !$summary) { echo '<p class="sn-ir-error">خواندن فاکتورها انجام نشد؛ دوباره تلاش کنید.</p>'; return; }
        $count = (int)$summary['count']; $pages = max(1,(int)ceil($count/$f['size'])); $f['page'] = min($pages,$f['page']);
        $rows = self::rows($f,($f['page']-1)*$f['size'],$f['size']);
        if ($wpdb->last_error) { echo '<p class="sn-ir-error">خواندن فاکتورها انجام نشد؛ دوباره تلاش کنید.</p>'; return; }
        $export = wp_nonce_url(add_query_arg(array_merge(self::params($f),['action'=>'sn_invoice_review_export']),admin_url('admin-post.php')),'sn_invoice_review_export');
        ?>
        <div class="sn-ir-metrics"><div><span>تعداد نتایج</span><strong><?php echo esc_html(number_format($count)); ?></strong></div><div><span>مجموع مبلغ فاکتورها</span><strong><?php echo esc_html(self::money($summary['total'])); ?></strong></div><div><span>جمع پرداخت تأییدشده ثبت‌شده</span><strong><?php echo esc_html(self::money($summary['paid'])); ?></strong></div></div>
        <div class="sn-ir-toolbar"><p>صفحه <?php echo esc_html((string)$f['page']); ?> از <?php echo esc_html((string)$pages); ?> · ساعت‌ها به وقت تهران</p><a class="sn-ir-btn sn-ir-secondary" href="<?php echo esc_url($export); ?>">خروجی CSV همه نتایج</a></div>
        <div class="sn-ir-table-wrap"><table><caption class="screen-reader-text">فاکتورهای مطابق فیلتر</caption><thead><tr><th>فاکتور / تاریخ</th><th>مشتری</th><th>فروشنده</th><th>مبلغ کل</th><th>وضعیت</th><th>روش پرداخت</th><th>جزئیات</th></tr></thead><tbody>
        <?php foreach ($rows as $row): $url=add_query_arg(array_merge(self::params($f),['ir_invoice'=>(int)$row['id']]),self::url()); ?>
            <tr><td><strong><?php echo esc_html($row['invoice_code']); ?></strong><small>#<?php echo esc_html($row['id']); ?> · <?php echo esc_html(self::date($row['created_at'])); ?></small></td><td><?php echo esc_html($row['customer_name']); ?><small dir="ltr"><?php echo esc_html($row['customer_phone']); ?></small><small><?php echo esc_html($row['city'] ?? ''); ?></small></td><td><?php echo esc_html($row['seller_name'] ?: ('#'.$row['seller_id'])); ?></td><td><?php echo esc_html(self::money($row['review_total'])); ?></td><td><span class="sn-ir-badge"><?php echo esc_html(self::label(($row['status'] ?? '') ?: (($row['invoice_status'] ?? '') ?: ($row['payment_status'] ?? '')))); ?></span></td><td><?php echo esc_html(SN_Helpers::pay_method_label((string)$row['pay_method'])); ?></td><td><a class="sn-ir-btn sn-ir-secondary" href="<?php echo esc_url($url); ?>">مشاهده</a></td></tr>
        <?php endforeach; if (!$rows): ?><tr><td colspan="7" class="sn-ir-empty">فاکتوری با این فیلترها پیدا نشد.</td></tr><?php endif; ?>
        </tbody></table></div>
        <nav class="sn-ir-pagination" aria-label="صفحه‌بندی فاکتورها">
        <?php
        $numbers = array_unique([1,max(1,$f['page']-1),$f['page'],min($pages,$f['page']+1),$pages]); sort($numbers);
        foreach ($numbers as $page) {
            $params=self::params($f); $params['ir_page']=$page;
            echo '<a class="sn-ir-btn '.($page===$f['page']?'':'sn-ir-secondary').'" '.($page===$f['page']?'aria-current="page"':'').' href="'.esc_url(add_query_arg($params,self::url())).'">'.esc_html((string)$page).'</a>';
        }
        ?></nav><?php
    }

    private static function fields(array $row, array $fields): void
    {
        echo '<dl class="sn-ir-fields">';
        foreach ($fields as $key=>$spec) {
            [$label,$type] = is_array($spec) ? $spec : [$spec,'text'];
            $value = $row[$key] ?? '';
            if (!is_scalar($value) || $value === '') { $value='—'; }
            elseif ($type==='money') { $value=self::money($value); }
            elseif ($type==='date') { $value=self::date($value); }
            elseif ($type==='user') { $value=self::user($value); }
            elseif ($type==='status') { $value=self::label($value); }
            echo '<div><dt>'.esc_html($label).'</dt><dd>'.nl2br(esc_html((string)$value)).'</dd></div>';
        }
        echo '</dl>';
    }

    private static function detail(int $id, array $f): void
    {
        global $wpdb;
        // Called only after the render capability check. Never expose the public payment token.
        $row=$wpdb->get_row($wpdb->prepare("SELECT * FROM {$wpdb->prefix}sn_invoices WHERE id=%d",$id),ARRAY_A);
        echo '<div class="sn-ir-toolbar"><a class="sn-ir-btn sn-ir-secondary" href="'.esc_url(add_query_arg(self::params($f),self::url())).'">بازگشت به نتایج</a><button type="button" class="sn-ir-btn sn-ir-secondary" data-ir-print>چاپ جزئیات</button></div>';
        if (!$row) { echo '<p class="sn-ir-error">فاکتور پیدا نشد یا خواندن آن ممکن نیست.</p>'; return; }
        echo '<h2>فاکتور '.esc_html($row['invoice_code']).' <small>#'.esc_html((string)$id).'</small></h2>';
        echo '<section class="sn-ir-card"><h3>مشتری و مشخصات فاکتور</h3>';
        self::fields($row,['customer_name'=>'نام مشتری','customer_phone'=>'موبایل','customer_phone_secondary'=>'تلفن دوم','province'=>'استان','city'=>'شهر','customer_address'=>'نشانی','customer_postal_code'=>'کد پستی','seller_id'=>['فروشنده','user'],'issued_by_user_id'=>['صادرکننده','user'],'customer_wp_id'=>'شناسه کاربر مشتری','lead_id'=>'شناسه سرنخ','wc_order_id'=>'شناسه سفارش','created_at'=>['تاریخ صدور','date'],'updated_at'=>['آخرین تغییر','date']]);
        echo '</section><section class="sn-ir-card"><h3>مبلغ و وضعیت پرداخت</h3>';
        self::fields($row,['status'=>['وضعیت اصلی','status'],'invoice_status'=>['وضعیت فاکتور','status'],'payment_status'=>['وضعیت پرداخت','status'],'payment_workflow_status'=>['روند پرداخت','status'],'pay_method'=>['روش پرداخت','status'],'payment_plan'=>['نوع پرداخت','status'],'product_price'=>['مبلغ محصول','money'],'original_total'=>['جمع اولیه','money'],'discount_total'=>['جمع تخفیف','money'],'discount_amount'=>['تخفیف','money'],'coupon_code'=>'کد تخفیف','coupon_discount_amount'=>['تخفیف کد','money'],'final_total'=>['مبلغ نهایی','money'],'payment_total_amount'=>['کل تعهد پرداخت','money'],'current_due_amount'=>['مبلغ مرحله جاری','money'],'paid_total_amount'=>['پرداخت تأییدشده ثبت‌شده','money'],'remaining_amount'=>['مانده ثبت‌شده','money'],'current_payment_stage'=>'شماره مرحله جاری','paid_at'=>['زمان پرداخت','date'],'payment_completed_at'=>['زمان تکمیل پرداخت','date'],'wheel_reward_summary'=>'پاداش و هدایا']);
        echo '</section><section class="sn-ir-card"><h3>اقلام فاکتور</h3>';
        $items=$wpdb->get_results($wpdb->prepare("SELECT * FROM {$wpdb->prefix}sn_invoice_items WHERE invoice_id=%d ORDER BY id",$id),ARRAY_A) ?: [];
        if (!$items) { $items=[['product_name'=>get_the_title((int)$row['product_id']) ?: ('محصول #'.$row['product_id']),'product_id'=>$row['product_id'],'qty'=>1,'unit_price'=>$row['product_price'],'total_price'=>$row['product_price'],'is_free'=>0]]; }
        echo '<div class="sn-ir-table-wrap"><table><thead><tr><th>محصول</th><th>تعداد</th><th>قیمت واحد</th><th>جمع</th><th>هدیه</th></tr></thead><tbody>';
        foreach ($items as $item) { echo '<tr><td>'.esc_html($item['product_name']).'<small>#'.esc_html($item['product_id']).'</small></td><td>'.esc_html($item['qty']).'</td><td>'.esc_html(self::money($item['unit_price'])).'</td><td>'.esc_html(self::money($item['total_price'])).'</td><td>'.(!empty($item['is_free'])?'بله':'خیر').'</td></tr>'; }
        echo '</tbody></table></div></section><section class="sn-ir-card"><h3>مراحل پرداخت</h3>';
        $stages=$wpdb->get_results($wpdb->prepare("SELECT * FROM {$wpdb->prefix}sn_invoice_payment_stages WHERE invoice_id=%d ORDER BY stage_no,id",$id),ARRAY_A) ?: [];
        if (!$stages) { echo '<p>مرحلهٔ جداگانه‌ای ثبت نشده است.</p>'; }
        foreach ($stages as $stage) {
            echo '<details open><summary>مرحله '.esc_html($stage['stage_no']).' · '.esc_html(self::label($stage['status'])).' · '.esc_html(self::money($stage['requested_amount'])).'</summary>';
            self::fields($stage,['stage_type'=>['نوع مرحله','status'],'pay_method'=>['روش پرداخت','status'],'payment_source'=>'منبع پرداخت','payment_ref_id'=>'کد پیگیری','issued_by_user_id'=>['صادرکننده','user'],'approved_by_user_id'=>['تأییدکننده','user'],'created_at'=>['ایجاد','date'],'paid_at'=>['پرداخت','date'],'approved_at'=>['تأیید','date'],'note'=>'یادداشت']);
            echo SN_Helpers::receipt_links($stage).'</details>';
        }
        echo '</section><section class="sn-ir-card"><h3>تراکنش‌ها و فیش‌های هر پرداخت</h3>';
        $payments=SN_Payment_Transactions::rows((object)$row);
        if (!$payments) { echo '<p>تراکنش جداگانه‌ای ثبت نشده است.</p>'; }
        foreach ($payments as $payment) {
            echo '<details open><summary>تراکنش #'.esc_html($payment['id']).' · مرحله '.esc_html($payment['stage_no']).' · '.esc_html($payment['status_label']).' · '.esc_html(self::money($payment['amount'])).'</summary>';
            self::fields($payment,['ref_id'=>'کد پیگیری','pay_method'=>['روش پرداخت','status'],'source_label'=>'منبع ثبت','uploaded_by_user_id'=>['ثبت‌کننده','user'],'created_at'=>['ایجاد','date'],'updated_at'=>['به‌روزرسانی','date']]);
            self::proof_fields((array)$payment['proof']);
            echo SN_Helpers::receipt_links($payment).'</details>';
        }
        echo '</section><section class="sn-ir-card"><h3>اطلاعات و فیش‌های ثبت‌شده روی فاکتور</h3>';
        self::proof_fields($row);
        self::fields($row,['payment_card_number'=>'کارت مقصد ثبت‌شده','payment_card_owner'=>'صاحب کارت مقصد','manual_amount'=>['مبلغ واریز دستی','money'],'deposit_amount'=>['مبلغ پیش‌پرداخت','money'],'deposit_card_from_last4'=>'چهار رقم کارت مبدأ پیش‌پرداخت','deposit_card_to_last4'=>'چهار رقم کارت مقصد پیش‌پرداخت','deposit_jalali_datetime'=>'تاریخ پیش‌پرداخت']);
        echo SN_Helpers::receipt_links($row) ?: '<p>فیشی روی خود فاکتور ثبت نشده است.</p>';
        echo '</section><section class="sn-ir-card"><h3>سوابق بررسی مالی و پیگیری</h3>';
        self::fields($row,['approved_by'=>['تأییدکننده','user'],'approved_at'=>['زمان تأیید','date'],'rejected_by'=>['ردکننده','user'],'rejected_at'=>['زمان رد','date'],'rejected_reason'=>'دلیل رد','financial_reviewed_by'=>['بررسی‌کننده مالی','user'],'financial_reviewed_at'=>['زمان بررسی مالی','date'],'financial_reject_reason'=>'دلیل رد مالی','financial_rejected_by'=>['ردکننده مالی','user'],'financial_rejected_at'=>['زمان رد مالی','date'],'resend_to_financial_at'=>['ارسال مجدد به مالی','date'],'recontact_requested_at'=>['درخواست تماس مجدد','date'],'recontact_note'=>'یادداشت پیگیری','financial_return_state'=>'وضعیت بازگشت از مالی','returned_to_seller_at'=>['بازگشت به فروشنده','date'],'resent_after_return_at'=>['ارسال پس از بازگشت','date'],'payment_archived_at'=>['زمان بایگانی','date'],'payment_archived_by'=>['بایگانی‌کننده','user'],'payment_archive_reason'=>'دلیل بایگانی']);
        echo '</section>';
    }

    private static function proof_fields(array $row): void
    {
        self::fields($row,['manual_transfer_type'=>['نوع انتقال','status'],'manual_card_from'=>'چهار رقم کارت مبدأ','manual_card_to'=>'چهار رقم کارت مقصد','manual_account_last6'=>'شش رقم حساب','manual_tracking_last6'=>'شش رقم پیگیری','manual_account_owner'=>'صاحب حساب','manual_paid_at'=>['زمان واریز','date'],'manual_paid_at_jalali'=>'زمان واریز ثبت‌شده (شمسی)']);
    }

    /** Defuse formula cells when opened by Excel/LibreOffice. */
    private static function csv_cell($value): string
    {
        $value=(string)$value;
        return preg_match('/^[\s\x00-\x1f]*[=+@-]/u',$value) ? "'".$value : $value;
    }

    public static function export(): void
    {
        if (!self::allowed()) { wp_die('دسترسی غیرمجاز', '', ['response'=>403]); }
        check_admin_referer('sn_invoice_review_export');
        $f=self::filters($_GET);
        if ($f['errors']) { wp_die(esc_html(implode(' ',$f['errors'])), '', ['response'=>400]); }
        nocache_headers();
        header('Content-Type: text/csv; charset=UTF-8');
        header('Content-Disposition: attachment; filename="invoice-review-'.gmdate('Ymd-His').'.csv"');
        $out=fopen('php://output','w');
        fwrite($out,"\xEF\xBB\xBF");
        fputcsv($out,['شناسه','کد فاکتور','مشتری','موبایل','تلفن دوم','استان','شهر','نشانی','کد پستی','فروشنده','شناسه فروشنده','تاریخ صدور (تهران)','وضعیت اصلی','وضعیت فاکتور','وضعیت پرداخت','مبلغ کل (تومان)','پرداخت تأییدشده ثبت‌شده','مانده ثبت‌شده','روش پرداخت','نوع پرداخت','مرحله جاری'],',','"','');
        // Stream every matching page; no historical 100-row export cap.
        for ($offset=0; ; $offset+=500) {
            $rows=self::rows($f,$offset,500);
            global $wpdb;
            if ($wpdb->last_error) { fputcsv($out,['خطا در خواندن ادامه نتایج؛ خروجی کامل نیست.'],',','"',''); break; }
            foreach ($rows as $row) {
                $cells=[$row['id'],$row['invoice_code'],$row['customer_name'],$row['customer_phone'],$row['customer_phone_secondary'],$row['province'],$row['city'],$row['customer_address'],$row['customer_postal_code'],$row['seller_name'],$row['seller_id'],self::date($row['created_at']),self::label($row['status']),self::label($row['invoice_status']),self::label($row['payment_status']),$row['review_total'],$row['paid_total_amount'],$row['remaining_amount'],SN_Helpers::pay_method_label((string)$row['pay_method']),self::label($row['payment_plan']),$row['current_payment_stage']];
                fputcsv($out,array_map([self::class,'csv_cell'],$cells),',','"','');
            }
            if (count($rows)<500) { break; }
        }
        fclose($out); exit;
    }
}
