<?php
if (!defined('ABSPATH')) { exit; }
require_once __DIR__.'/class-sn-mis-report-model.php';
require_once __DIR__.'/class-sn-mis-report-source.php';
require_once __DIR__.'/class-sn-mis-report-store.php';
require_once __DIR__.'/class-sn-mis-report-xlsx.php';
final class SN_MIS_Report {
    public static function register(): void {
        add_action('wp_ajax_sn_mis_report',[self::class,'ajax']);
        add_action('admin_post_sn_mis_report_download',[self::class,'download']);
        add_action('sn_mis_report_cleanup',static function() { (new SN_MIS_Report_Store())->cleanup(); });
        add_action('init',static function() { if (get_option('sn_mis_report_schema')===SN_MIS_Report_Store::VERSION && !wp_next_scheduled('sn_mis_report_cleanup')) { wp_schedule_event(time()+3600,'hourly','sn_mis_report_cleanup'); } });
        if (defined('WP_CLI') && WP_CLI) { WP_CLI::add_command('sn mis-report-install',static function() { (new SN_MIS_Report_Store())->install(); WP_CLI::success('MIS report schema and online indexes ready. No business records changed.'); }); }
    }
    public static function render(bool $embedded = false): string {
        try {
            new SN_MIS_Report_Source();
        } catch (Throwable $e) {
            return '<div class="sn-card sn-mis-report-state"><div class="sn-alert sn-error">'.esc_html($e->getMessage()).'</div></div>';
        }
        wp_enqueue_script('sn-mis-report',SN_PLUGIN_URL.'assets/js/mis-report.js',[],SN_VERSION,true);
        $config=['ajax'=>admin_url('admin-ajax.php'),'nonce'=>wp_create_nonce('sn_mis_report'),'download'=>admin_url('admin-post.php'),'downloadNonce'=>wp_create_nonce('sn_mis_report_download'),'canInstall'=>current_user_can('manage_options'),'ready'=>get_option('sn_mis_report_schema')===SN_MIS_Report_Store::VERSION,'timezone'=>wp_timezone_string()];
        wp_add_inline_script('sn-mis-report','window.snMisReport='.wp_json_encode($config).';','before');
        ob_start(); ?>
        <style><?php readfile(SN_PLUGIN_DIR.'assets/css/mis-report.css'); ?></style>
        <?php if (! $embedded) : ?><div id="sn-mis-panel" class="sn-panel sn-portal" dir="rtl"><?php endif; ?>
        <div id="sn-mis-report" class="sn-mis-report-tab" dir="rtl">
        <div class="sn-card sn-mis-report-intro">
            <div class="sn-mis-report-heading">
                <div>
                    <h3>گزارش‌های MIS</h3>
                    <p class="sn-note">گزارش پرونده‌ها، عملکرد فروشنده و نتیجه فروش. محاسبه فقط هنگام باز بودن همین تب انجام می‌شود.</p>
                </div>
            </div>
            <p class="sn-note">پرونده یکتا = شناسه ردیف MIS. پیش‌فاکتور، تأیید فاکتور و تکمیل پرداخت جدا محاسبه می‌شوند. مبالغ با واحد ذخیره‌شده در CRM هستند.</p>
        </div>
        <div class="sn-card sn-mis-report-filters-card">
        <form id="sn-mr-filters"><div class="sn-mr-grid">
        <label>کمپین<select name="campaign"><option value="">همه کمپین‌ها</option></select></label>
        <label>دسته واردات<select name="batch"><option value="0">همه دسته‌ها</option></select></label>
        <label>دسته داده<select name="category"><option value="">همه</option></select></label>
        <?php foreach (SN_MIS_Report_Model::ROLES as $role=>$label): ?><label><?php echo esc_html($label); ?><select name="<?php echo esc_attr($role); ?>"><option value="0">همه افراد مجاز این سطح</option></select></label><?php endforeach; ?>
        <label>اعتبار ردیف<input name="validity" list="sn-mr-validity" placeholder="همه وضعیت‌ها"><datalist id="sn-mr-validity"><option value="valid"><option value="duplicate"><option value="invalid"></datalist></label>
        <label>وضعیت توزیع<input name="distribution" list="sn-mr-distribution" placeholder="همه وضعیت‌ها"><datalist id="sn-mr-distribution"><option value="delivered_to_seller"><option value="distributed_forward"><option value="returned_to_mis"><option value="unassigned"></datalist></label>
        <label>پیگیری فروشنده<select name="flow"><option value="">همه</option><option value="no_answer">جواب نداده</option><option value="callback">تماس مجدد</option><option value="not_purchased">عدم خرید</option><option value="pre_invoice">پیش‌فاکتور</option><option value="unrecorded">ثبت نشده</option><option value="duplicate">تکراری</option></select></label>
        <label>وضعیت فاکتور<input name="invoice_status" list="sn-mr-invoice" placeholder="همه وضعیت‌ها"><datalist id="sn-mr-invoice"><option value="pre_invoice"><option value="pending_financial_approval"><option value="approved"><option value="paid"><option value="cancelled"><option value="rejected"><option value="unlinked"></datalist></label>
        <label>پرداخت<select name="payment"><option value="">همه</option><option value="completed">فروش تکمیل‌شده</option><option value="partial">پرداخت ناقص</option><option value="unpaid">بدون پرداخت ثبت‌شده</option><option value="awaiting_payment">منتظر پرداخت</option><option value="awaiting_assignment">منتظر تخصیص مرحله بعد</option></select></label>
        <label>کنترل دلیل<select name="reason_quality"><option value="">همه</option><option value="empty">خالی</option><option value="short">دو نویسه یا کمتر</option><option value="meaningless">نشانه‌ای / تکراری / مبهم</option><option value="recorded">ثبت‌شده؛ عبور از کنترل شکلی</option></select></label>
        <label>مبنای زمان<select name="basis"><option value="import">زمان ورود داده به MIS</option><option value="assignment">زمان رویداد تخصیص</option><option value="followup">زمان ثبت پیگیری</option><option value="invoice">زمان صدور فاکتور</option><option value="payment">زمان تأیید / تکمیل پرداخت</option></select></label>
        <label>از تاریخ و ساعت میلادی<input name="from" type="datetime-local"></label><label>تا تاریخ و ساعت میلادی<input name="to" type="datetime-local"></label>
        <label>جستجوی شماره، نام، ردیف یا کد فاکتور<input name="search" type="search"></label>
        </div><p class="sn-note">بازه خالی = همه زمان‌ها. منطقه زمانی: <b><?php echo esc_html(wp_timezone_string()); ?></b>. فیلتر زمان، پرونده‌های دارای رویداد در بازه را انتخاب می‌کند؛ وضعیت‌ها جاری هنگام گردآوری‌اند.</p>
        <div class="sn-actions sn-mr-actions"><button type="submit" class="sn-btn sn-btn-primary">ساخت / به‌روزرسانی گزارش</button><button type="reset" class="sn-btn sn-btn-secondary">پاک کردن فیلترها</button><button type="button" id="sn-mr-cancel" class="sn-btn sn-btn-secondary">لغو گزارش</button>
        <?php if (current_user_can('manage_options') && get_option('sn_mis_report_schema')!==SN_MIS_Report_Store::VERSION): ?><button type="button" id="sn-mr-install" class="sn-btn sn-btn-secondary">نصب ساختار گزارش</button><?php endif; ?>
        </div></form>
        <p id="sn-mr-status" class="sn-mis-report-status" role="status" aria-live="polite"></p>
        </div>
        <div id="sn-mr-summary"></div>
        <section id="sn-mr-results" hidden>
            <div class="sn-card sn-mr-export-card"><div class="sn-actions"><button type="button" id="sn-mr-export" class="sn-btn sn-btn-primary">ساخت خروجی XLSX همین گزارش</button><span id="sn-mr-export-status" role="status"></span><div id="sn-mr-download"></div></div></div>
            <div class="sn-card"><h3>ریز پرونده‌ها</h3><div id="sn-mr-table"></div><div class="sn-actions sn-mr-pager"><button type="button" id="sn-mr-prev" class="sn-btn sn-btn-secondary sn-btn-sm">قبلی</button><span id="sn-mr-page"></span><button type="button" id="sn-mr-next" class="sn-btn sn-btn-secondary sn-btn-sm">بعدی</button></div><section id="sn-mr-detail" aria-live="polite"></section></div>
        </section>
        </div>
        <?php if (! $embedded) : ?></div><?php endif; ?>
        <?php return ob_get_clean();
    }
    public static function ajax(): void {
        check_ajax_referer('sn_mis_report','nonce');
        nocache_headers();
        try {
            $op=sanitize_key($_POST['op']??''); $store=new SN_MIS_Report_Store();
            if ($op==='install') { $store->install(); wp_send_json_success(['installed'=>true]); }
            $src=new SN_MIS_Report_Source(); $id=sanitize_text_field(wp_unslash($_POST['job']??''));
            if ($op==='options') { wp_send_json_success($src->options()); }
            if ($op==='start') {
                $raw=json_decode(wp_unslash($_POST['filters']??'{}'),true); if (!is_array($raw)) { throw new RuntimeException('فیلتر نامعتبر است.'); }
                wp_send_json_success($store->progress($store->start($src,SN_MIS_Report_Model::filters($raw))));
            }
            if ($op==='step') { wp_send_json_success($store->step($id,$src)); }
            if ($op==='summary') { wp_send_json_success($store->summary($id,$src)); }
            if ($op==='page') {
                $data=$store->page($id,$src,max(1,absint($_POST['page']??1)));
                foreach ($data['rows'] as &$r) { unset($r['history'],$r['times']); foreach ($r['invoices'] as &$i) { unset($i['stages']); } unset($i); } unset($r);
                wp_send_json_success($data);
            }
            if ($op==='detail') {
                $store->job($id,$src); $row=$store->db->get_var($store->db->prepare('SELECT payload FROM '.$store->table('cases').' WHERE job_id=%s AND row_id=%d',$id,absint($_POST['row']??0))); $store->checked(true);
                if (!$row) { throw new RuntimeException('پرونده در این گزارش مجاز موجود نیست.'); } wp_send_json_success(json_decode($row,true));
            }
            if ($op==='cancel') { $store->cancel($id,$src); wp_send_json_success(['cancelled'=>true]); }
            if ($op==='export') {
                $store->lock($id);
                try { $j=$store->job($id,$src); if ($j['status']!=='ready') { throw new RuntimeException('گزارش هنوز کامل نشده است.'); } $data=SN_MIS_Report_XLSX::step($j,$store,$src); }
                finally { $store->unlock($id); }
                wp_send_json_success($data);
            }
            throw new RuntimeException('عملیات گزارش نامعتبر است.');
        } catch (Throwable $e) { wp_send_json_error(['message'=>$e->getMessage()],400); }
    }
    public static function download(): void {
        check_admin_referer('sn_mis_report_download');
        try {
            $src=new SN_MIS_Report_Source(); $store=new SN_MIS_Report_Store(); $id=sanitize_text_field(wp_unslash($_POST['job']??''));
            $j=$store->job($id,$src); if ($j['export_status']!=='ready') { throw new RuntimeException('خروجی آماده نیست.'); }
            $file=$j['export_dir'].'/report.xlsx'; if (!is_file($file)) { throw new RuntimeException('فایل خروجی منقضی شده؛ دوباره بسازید.'); }
            nocache_headers(); header('Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'); header('Content-Disposition: attachment; filename="mis-report-'.(new DateTimeImmutable('now',new DateTimeZone('UTC')))->format('Ymd-His').'.xlsx"'); header('X-Content-Type-Options: nosniff'); header('Content-Length: '.filesize($file)); readfile($file); exit;
        } catch (Throwable $e) { wp_die(esc_html($e->getMessage()),'گزارش MIS',['response'=>403]); }
    }
}
