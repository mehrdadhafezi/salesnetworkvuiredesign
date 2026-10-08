<?php
/** Run php tools/test-biavin-callback-only.php [fixture-directory]. No live WordPress writes. */
define('ABSPATH', __DIR__);
error_reporting(E_ALL);
set_error_handler(static function($severity, $message, $file, $line) { throw new RuntimeException("$message at $file:$line"); });
class WP_Post { public function __construct(public int $ID) {} }
class WP_User { public function __construct(public int $ID = 42) {} }
class WP_Error { public function __construct(public string $code, public string $message) {} public function get_error_message() { return $this->message; } }
class CallbackRedirect extends RuntimeException {}
class CallbackDenied extends RuntimeException {}
function absint($v) { return abs((int)$v); }
function sanitize_key($v) { return preg_replace('/[^a-z0-9_-]/', '', strtolower($v)); }
function sanitize_text_field($v) { return trim(strip_tags($v)); }
function sanitize_textarea_field($v) { return trim(strip_tags($v)); }
function wp_unslash($v) { return is_string($v) ? stripslashes($v) : $v; }
function is_wp_error($v) { return $v instanceof WP_Error; }
function wp_timezone() { return new DateTimeZone('UTC'); }
function get_post_meta($id, $key, $single) { return $GLOBALS['meta'][$id][$key] ?? ''; }
function update_post_meta($id, $key, $v) { $GLOBALS['meta'][$id][$key] = $v; }
function get_option($key, $default = false) { return $default; }
function wp_verify_nonce($nonce, $action) { return $nonce === 'valid'; }
function check_admin_referer($action) { if (($_POST['_wpnonce'] ?? '') !== 'valid') { throw new CallbackDenied('nonce'); } }
function current_user_can($cap, ...$args) { return $GLOBALS['can_edit']; }
function is_user_logged_in() { return true; }
function get_current_user_id() { return 42; }
function wp_die($message) { throw new CallbackDenied($message); }
function current_time($type) { return gmdate('Y-m-d H:i:s'); }
function wp_json_encode($v, $flags = 0) { return json_encode($v, $flags); }
function esc_url_raw($v, $protocols = ['http','https']) { return in_array(parse_url($v, PHP_URL_SCHEME), $protocols, true) ? $v : ''; }
function wp_http_validate_url($v) { return filter_var($v, FILTER_VALIDATE_URL); }
function esc_attr($v) { return htmlspecialchars((string)$v, ENT_QUOTES); }
function esc_html($v) { return esc_attr($v); }
function esc_textarea($v) { return esc_attr($v); }
function esc_url($v) { return esc_attr($v); }
function selected($a, $b) { if ($a === $b) { echo 'selected'; } }
function checked($v) { if ($v) { echo 'checked="checked"'; } }
function number_format_i18n($v) { return (string)$v; }
function wp_nonce_field($action, $name = '_wpnonce') { echo '<input name="'.esc_attr($name).'" type="hidden" value="valid">'; }
function admin_url($v) { return 'https://crm.example/'.$v; }
function home_url($v) { return 'https://crm.example'.$v; }
function add_query_arg($args, $url) { return $url.(str_contains($url, '?') ? '&' : '?').http_build_query($args); }
function wp_parse_url($url, $component) { return parse_url($url, $component); }
function wp_get_referer() { return false; }
function wp_safe_redirect($url) { throw new CallbackRedirect($url); }
function wp_clear_scheduled_hook($hook, $args) { $GLOBALS['cleared_hooks'][] = [$hook, $args]; }
class CallbackDb {
    public string $prefix = 'wp_';
    public object $op;
    public int $owner = 42;
    public array $writes = [];
    public array $prepared = [];
    public array $events = [];
    public bool $race = false;
    public string $case_date = '';
    public function prepare($sql, ...$args) {
        $original = $sql;
        foreach ($args as $arg) { $sql = preg_replace_callback('/%[ds]/', fn($m) => $m[0] === '%d' ? (string)(int)$arg : "'".str_replace("'", "''", $arg)."'", $sql, 1); }
        $this->prepared[$sql] = [$original, $args];
        return $sql;
    }
    public function get_var($sql) {
        if (preg_match("/SHOW TABLES LIKE '([^']+)'/", $sql, $m)) { return $m[1]; }
        if (str_contains($sql, 'SELECT content_product_id')) { return 101; }
        if (str_contains($sql, 'SELECT membership_id')) { return 9; }
        if (str_contains($sql, 'SELECT follow_up_at')) { return $this->case_date; }
        if (str_contains($sql, 'SELECT activation_code')) { return 'SECRET-CODE'; }
        throw new RuntimeException('Unexpected scalar: '.$sql);
    }
    public function get_row($sql) {
        if (str_contains($sql, 'SELECT mi.*')) { return (object)['id'=>501, 'content_product_id'=>101, 'customer_wp_id'=>$this->owner, 'membership_status'=>'active']; }
        if (str_contains($sql, 'SELECT * FROM wp_sn_project_operations')) { return clone $this->op; }
        if (str_contains($sql, 'COUNT(*)')) { return (object)['total'=>3, 'available'=>2]; }
        throw new RuntimeException('Unexpected row: '.$sql);
    }
    public function get_results($sql) { return []; }
    public function query($sql) {
        $this->writes[] = $sql;
        [$source, $args] = $this->prepared[$sql] ?? ['', []];
        if (!str_contains($source, "SET customer_choice='expert',customer_choice_at=")) { throw new RuntimeException('Unexpected write: '.$sql); }
        if ($this->race) { $this->op->stage = 'completed'; }
        if ($this->op->stage !== $args[8] || !empty($this->op->follow_up_at)) { return 0; }
        $this->op->customer_choice = 'expert'; $this->op->follow_up_at = $args[1]; $this->op->stage = $args[5];
        return 1;
    }
    public function insert($table, $values) { $this->events[] = $values; return 1; }
}
$wpdb = new CallbackDb();
$GLOBALS['meta'] = []; $GLOBALS['can_edit'] = true; $GLOBALS['cleared_hooks'] = [];
require dirname(__DIR__).'/includes/class-sn-helpers.php';
require dirname(__DIR__).'/includes/class-sn-operations-execution.php';
require dirname(__DIR__).'/includes/class-sn-operations-flow.php';
$execution = SN_Operations_Execution::instance(); $flow = SN_Operations_Flow::instance();
$checks = 0;
function check($condition, $label) { global $checks; if (!$condition) { throw new RuntimeException('FAIL: '.$label); } ++$checks; }
function reset_op($stage = 'awaiting_customer', $date = null, $self_allowed = 1, $supervisor = 12) {
    global $wpdb;
    $wpdb->op = (object)['id'=>7, 'membership_item_id'=>501, 'stage'=>$stage, 'follow_up_at'=>$date, 'customer_choice'=>'', 'customer_self_activation_allowed'=>$self_allowed, 'operations_sales_manager_user_id'=>10, 'sales_supervisor_user_id'=>$supervisor, 'sales_expert_user_id'=>$stage === 'sales_expert' ? 23 : 0, 'auto_route_eligible'=>0, 'activation_mode'=>'normal', 'base_credit'=>100000, 'current_credit'=>150000, 'upgrade_invoice_id'=>99];
    $wpdb->writes = []; $wpdb->events = []; $wpdb->race = false; $wpdb->owner = 42; $wpdb->case_date = '';
}
function submit($choice, $date, $nonce = 'valid') {
    global $flow;
    $_POST = ['item_id'=>501, 'choice'=>$choice, 'followup_date'=>$date, '_wpnonce'=>$nonce];
    try { $flow->handle_customer_action(); } catch (CallbackRedirect $e) { return $e->getMessage(); }
    throw new RuntimeException('Handler did not redirect');
}
$future = new DateTimeImmutable('+3 days', SN_Helpers::tehran_timezone());
$jalali = SN_Helpers::gregorian_to_jalali_input_value($future->format('Y-m-d'));
$parsed = SN_Helpers::normalize_jalali_tehran_datetime($jalali);
$fa = strtr($jalali, ['0'=>'۰','1'=>'۱','2'=>'۲','3'=>'۳','4'=>'۴','5'=>'۵','6'=>'۶','7'=>'۷','8'=>'۸','9'=>'۹']);
$meta_key = '_sn_execution_callback_only';
foreach (['wallet_charge', 'form', 'physical_invoice', ''] as $type) {
    $_POST = ['sn_execution_product_nonce'=>'valid', 'sn_execution_fulfillment_type'=>$type, 'sn_execution_callback_only_present'=>'1', 'sn_execution_callback_only'=>'1', 'sn_execution_wallet_destination_url'=>'https://target.example/card'];
    $execution->save_product_meta(101, new WP_Post(101));
    check($execution->product_callback_only(101), 'setting enabled for every type');
    check($execution->product_destination_url(101) === '', 'configured destination hidden');
    check(get_post_meta(101, '_sn_execution_wallet_destination_url', true) === 'https://target.example/card', 'URL preserved');
    reset_op();
    $html = $flow->render_customer_card_controls(['id'=>501,'content_product_id'=>101], [], new WP_User());
    check(substr_count($html, '<form ') === 1 && str_contains($html, 'name="followup_date"'), 'one date form');
    check(!str_contains($html, 'choice" value="normal') && !str_contains($html, 'choice" value="upgrade') && !str_contains($html, 'کد فعال‌سازی'), 'no self activation, upgrades or code');
    check($execution->customer_wallet_destination(501) === '', 'legacy accessor hides destination');
}
ob_start(); $execution->render_product_metabox(new WP_Post(101)); $metabox = ob_get_clean();
check(str_contains($metabox, 'name="sn_execution_callback_only" value="1" checked="checked"'), 'checked setting rendered');
$_POST = ['sn_execution_product_nonce'=>'valid']; $execution->save_product_meta(101, new WP_Post(101));
check($execution->product_callback_only(101), 'old forms preserve new option');
$_POST['sn_execution_callback_only_present'] = '1'; $execution->save_product_meta(101, new WP_Post(101));
check(!$execution->product_callback_only(101) && $execution->product_destination_url(101) === 'https://target.example/card', 'uncheck restores stored URL');
$_POST['sn_execution_callback_only'] = '1'; $_POST['sn_execution_product_nonce'] = 'invalid'; $execution->save_product_meta(101, new WP_Post(101));
check(!$execution->product_callback_only(101), 'meta nonce enforced');
$_POST['sn_execution_product_nonce'] = 'valid'; $GLOBALS['can_edit'] = false; $execution->save_product_meta(101, new WP_Post(101));
check(!$execution->product_callback_only(101), 'meta edit permission enforced');
$GLOBALS['can_edit'] = true; $execution->save_product_meta(101, new WP_Post(101));
foreach (['awaiting_customer','sales_manager_queue','sales_supervisor','sales_expert','upgrade_payment','executive_manager_queue','executive_in_progress','execution_expert','execution_form_review'] as $stage) {
    reset_op($stage, null, 0);
    $before = clone $wpdb->op;
    check(str_contains($flow->render_customer_card_controls(['id'=>501], [], new WP_User()), 'ثبت تاریخ تماس'), 'old routed cards can pick date despite old self policy');
    $result = submit('expert', $fa);
    check(str_contains($result, 'sn_ops_kind=success') && str_contains($result, 'card=501'), 'date saved and returns exact card');
    check($wpdb->op->follow_up_at === $parsed['mysql'], 'Persian date stored in site timezone');
    check($wpdb->op->stage === ($stage === 'awaiting_customer' ? 'sales_supervisor' : $stage), 'only awaiting card advances, routing preserved');
    check($wpdb->op->base_credit === $before->base_credit && $wpdb->op->current_credit === $before->current_credit && $wpdb->op->upgrade_invoice_id === $before->upgrade_invoice_id && $wpdb->op->sales_expert_user_id === $before->sales_expert_user_id, 'financial state and assignee preserved');
    check(count($wpdb->events) === 1 && str_contains($wpdb->events[0]['details'], 'callback_only'), 'callback audit recorded');
    $html = $flow->render_customer_card_controls(['id'=>501], [], new WP_User());
    check(str_contains($html, $jalali) && str_contains($html, 'کارشناس در این تاریخ با شما تماس خواهد گرفت.') && !str_contains($html, '<form '), 'confirmation persists after reload');
    $previous_date = $wpdb->op->follow_up_at; $count = count($wpdb->writes);
    check(str_contains(submit('expert', $jalali), 'sn_ops_kind=error') && $wpdb->op->follow_up_at === $previous_date && count($wpdb->writes) === $count, 'duplicate does not replace date');
}
reset_op('awaiting_customer', null, 1, 0); submit('expert', $jalali);
check($wpdb->op->stage === 'sales_manager_queue', 'no direct supervisor routes to manager');
foreach (['normal','upgrade','continue_normal','retry_upgrade','unknown'] as $choice) {
    reset_op(); check(str_contains(submit($choice, $jalali), 'sn_ops_kind=error') && !$wpdb->writes, 'forged alternative rejected before write');
}
foreach (['', 'bad', '1405/13/01', '1405/07/99', '1400/01/01', ['1405/08/01']] as $bad_date) {
    reset_op(); check(str_contains(submit('expert', $bad_date), 'sn_ops_kind=error') && !$wpdb->writes, 'invalid, past and array date rejected');
}
foreach (['completed','cancelled','execution_cancelled','delivered','shipping_queue','customer_code_issued','execution_wallet_active'] as $stage) {
    reset_op($stage, $parsed['mysql']);
    $html = $flow->render_customer_card_controls(['id'=>501], [], new WP_User());
    check(!str_contains($html, '<form ') && !str_contains($html, 'کارشناس در این تاریخ'), 'finished cards have no new contact promise');
    check(str_contains(submit('expert', $jalali), 'sn_ops_kind=error') && !$wpdb->writes, 'finished cards cannot reopen');
}
reset_op(); $wpdb->race = true;
check(str_contains(submit('expert', $jalali), 'sn_ops_kind=error') && $wpdb->op->follow_up_at === null && !$wpdb->events, 'concurrent terminal transition respected');
check(str_contains($wpdb->writes[0], "AND stage='awaiting_customer' AND (follow_up_at IS NULL OR follow_up_at='')"), 'atomic stage and date guard');
reset_op('execution_expert'); $wpdb->case_date = $parsed['mysql'];
$html = $flow->render_customer_card_controls(['id'=>501], [], new WP_User());
check(str_contains($html, $jalali) && !str_contains($html, '<form '), 'existing execution expert date shown');
check(str_contains(submit('expert', $jalali), 'sn_ops_kind=error') && !$wpdb->writes, 'customer cannot overwrite expert date');
reset_op(); $GLOBALS['meta'][101][$meta_key] = '0'; $GLOBALS['meta'][101]['_sn_execution_fulfillment_type'] = 'form';
$html = $flow->render_customer_card_controls(['id'=>501], [], new WP_User());
check(str_contains($html, 'فعال‌سازی توسط خودم') && str_contains($html, 'فعال‌سازی توسط کارشناس'), 'other products keep original choices');
$GLOBALS['meta'][101][$meta_key] = '1';
reset_op();
try { submit('expert', $jalali, 'invalid'); check(false, 'invalid nonce'); } catch (CallbackDenied $e) { check(!$wpdb->writes, 'customer nonce enforced'); }
reset_op(); $wpdb->owner = 43;
try { submit('expert', $jalali); check(false, 'other customer'); } catch (CallbackDenied $e) { check(!$wpdb->writes, 'card ownership enforced'); }
check($flow->render_customer_card_controls(['id'=>501], [], new WP_User()) === '', 'unauthorized card controls hidden');
reset_op();
if (!empty($argv[1])) {
    file_put_contents($argv[1].'/callback-only.html', $flow->render_customer_card_controls(['id'=>501], [], new WP_User()));
    file_put_contents($argv[1].'/callback-metabox.html', $metabox);
    submit('expert', $jalali);
    file_put_contents($argv[1].'/callback-confirmation.html', $flow->render_customer_card_controls(['id'=>501], [], new WP_User()));
}
echo "PASS $checks Biavin callback-only regression checks\n";
