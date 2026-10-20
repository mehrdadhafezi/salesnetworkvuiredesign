<?php
/** Focused regression tests. Run: php tools/test-biavin-live-product.php */
define('ABSPATH', __DIR__);
define('ARRAY_A', 'ARRAY_A');
error_reporting(E_ALL);
set_error_handler(static function($severity, $message, $file, $line) { throw new RuntimeException("$message at $file:$line"); });
class WP_Post { public function __construct(public int $ID) {} }
class WP_User { public int $ID = 42; public string $user_login = 'customer'; }
class SN_Helpers {
    public static function normalize_mobile($value) { return $value; }
    public static function is_valid_mobile($value) { return false; }
    public static function format_price($value) { return number_format($value,0).' تومان'; }
}
function absint($value) { return abs((int)$value); }
function sanitize_key($value) { return preg_replace('/[^a-z0-9_-]/', '', strtolower($value)); }
function sanitize_text_field($value) { return trim(strip_tags($value)); }
function sanitize_textarea_field($value) { return trim(strip_tags($value)); }
function wp_unslash($value) { return is_string($value) ? stripslashes($value) : $value; }
function wp_verify_nonce($nonce, $action) { return $nonce === 'valid' && $action === 'sn_execution_product_meta'; }
function current_user_can($cap, ...$args) { return $GLOBALS['can_edit']; }
function get_post_meta($id, $key, $single) { return $GLOBALS['meta'][$id][$key] ?? ''; }
function update_post_meta($id, $key, $value) { $GLOBALS['meta'][$id][$key] = $value; }
function get_user_meta($id, $key, $single) { return ''; }
function esc_url_raw($value, $protocols = ['http', 'https']) {
    return in_array(strtolower((string)parse_url($value, PHP_URL_SCHEME)), $protocols, true) ? $value : '';
}
function wp_http_validate_url($value) { return filter_var($value, FILTER_VALIDATE_URL) ? $value : false; }
function esc_attr($value) { return htmlspecialchars((string)$value, ENT_QUOTES); }
function esc_html($value) { return esc_attr($value); }
function esc_textarea($value) { return esc_attr($value); }
function esc_url($value) { return esc_attr(esc_url_raw($value)); }
function selected($a, $b) { if ($a === $b) { echo 'selected'; } }
function checked($value) { if ($value) { echo 'checked="checked"'; } }
function wp_nonce_field($action, $name) { echo '<input type="hidden" name="'.esc_attr($name).'" value="valid">'; }
function number_format_i18n($value) { return (string)$value; }
function admin_url($value) { return 'https://crm.example/'.$value; }
function get_post_thumbnail_id($id) { return 0; }
function get_the_post_thumbnail_url($id, $size) { return ''; }
function wp_get_attachment_image_url($id, $size) { return 'https://crm.example/image-'.$id.'.jpg'; }
function get_post_field($field, $id) { return $GLOBALS['fields'][$id][$field] ?? ''; }
function do_blocks($value) { return $value; }
function wpautop($value) { return '<p>'.$value.'</p>'; }
function wp_kses_post($value) { return preg_replace('#<script\b[^>]*>.*?</script>#is', '', $value); }
function wc_get_product($id) {
    if (!isset($GLOBALS['products'][$id])) { return null; }
    return new class($id) {
        public function __construct(private int $id) {}
        public function get_image_id() { return $GLOBALS['products'][$this->id]['image_id'] ?? 0; }
        public function get_name() { return $GLOBALS['products'][$this->id]['name'] ?? ''; }
        public function get_description() { return $GLOBALS['products'][$this->id]['full']; }
        public function get_short_description() { return $GLOBALS['products'][$this->id]['short']; }
    };
}
class LiveProductDb {
    public string $prefix = 'wp_';
    public array $items = [];
    public array $writes = [];
    public array $queries = [];
    public function prepare($sql, ...$args) {
        foreach ($args as $arg) { $sql = preg_replace_callback('/%[ds]/', fn($m) => $m[0] === '%d' ? (string)(int)$arg : "'".str_replace("'", "''", $arg)."'", $sql, 1); }
        return $sql;
    }
    public function get_var($sql) {
        $this->queries[] = $sql;
        if (preg_match("/SHOW TABLES LIKE '([^']+)'/", $sql, $m)) { return $m[1]; }
        if (preg_match('/SELECT content_product_id FROM wp_sn_project_membership_items WHERE id=(\d+)/', $sql, $m)) {
            foreach ($this->items as $item) { if ($item['id'] === (int)$m[1]) { return $item['content_product_id']; } }
            return null;
        }
        throw new RuntimeException('Unexpected query: '.$sql);
    }
    public function get_row($sql) { return (object)['total' => 3, 'available' => 2]; }
    public function get_results($sql, $format = null) {
        if (str_contains($sql, 'SELECT m.* FROM wp_sn_project_memberships')) { return [['id'=>9, 'subscription_product_id'=>90]]; }
        if (str_contains($sql, 'SELECT mi.*')) { return $this->items; }
        if (str_contains($sql, 'sn_operations_form_definitions')) { return []; }
        throw new RuntimeException('Unexpected result query: '.$sql);
    }
    public function query($sql) { $this->writes[] = $sql; return 1; }
}
$wpdb = new LiveProductDb();
$GLOBALS['meta'] = []; $GLOBALS['products'] = []; $GLOBALS['can_edit'] = true;
require dirname(__DIR__).'/includes/class-sn-operations-execution.php';
require dirname(__DIR__).'/includes/class-sn-customer-portal.php';
$execution = SN_Operations_Execution::instance(); $portal = SN_Customer_Portal::instance();
$checks = 0;
function check($condition, $label) { global $checks; if (!$condition) { throw new RuntimeException('FAIL: '.$label); } ++$checks; }
$key = '_sn_execution_wallet_destination_url';
foreach (['wallet_charge','form','physical_invoice',''] as $index => $type) {
    $id = 100 + $index;
    $_POST = ['sn_execution_product_nonce'=>'valid', 'sn_execution_fulfillment_type'=>$type, 'sn_execution_wallet_destination_url'=>'https://new.example/product-'.$id];
    $execution->save_product_meta($id, new WP_Post($id));
    check(get_post_meta($id, $key, true) === $_POST['sn_execution_wallet_destination_url'], 'save link for '.$type);
    check($execution->product_destination_url($id) === $_POST['sn_execution_wallet_destination_url'], 'read link for '.$type);
    ob_start(); $execution->render_product_metabox(new WP_Post($id)); $html = ob_get_clean();
    $start = strpos($html, '<div data-sn-execution-field="wallet_charge"');
    $end = strpos($html, '</div>', $start);
    check(strpos($html, 'name="sn_execution_wallet_destination_url"') > $end, 'destination outside wallet-only fields '.$type);
    check(substr_count($html, 'name="sn_execution_wallet_destination_url"') === 1, 'one shared link input '.$type);
    if (!empty($argv[1])) { file_put_contents($argv[1].'/metabox-'.($type ?: 'none').'.html', $html); }
}
$id = 101; $post = new WP_Post($id);
$_POST = ['sn_execution_product_nonce'=>'invalid', 'sn_execution_wallet_destination_url'=>'https://forged.example'];
$previous = $GLOBALS['meta']; $execution->save_product_meta($id, $post);
check($GLOBALS['meta'] === $previous, 'invalid nonce cannot change meta');
$_POST['sn_execution_product_nonce'] = 'valid'; $GLOBALS['can_edit'] = false;
$execution->save_product_meta($id, $post); check($GLOBALS['meta'] === $previous, 'permission required');
$GLOBALS['can_edit'] = true;
$_POST = ['sn_execution_product_nonce'=>'valid', 'sn_execution_fulfillment_type'=>'form'];
$execution->save_product_meta($id, $post); check($GLOBALS['meta'][$id][$key] === $previous[$id][$key], 'omitted destination preserves link');
$_POST['sn_execution_wallet_destination_url'] = '';
$execution->save_product_meta($id, $post); check($execution->product_destination_url($id) === '', 'explicit empty removes link');
foreach (['javascript:alert(1)', 'ftp://files.example/a', 'not a URL'] as $bad) {
    $_POST['sn_execution_wallet_destination_url'] = $bad; $execution->save_product_meta($id, $post);
    check($execution->product_destination_url($id) === '', 'invalid or unsafe link rejected');
}
$_POST['sn_execution_wallet_destination_url'] = ['https://array.example'];
$execution->save_product_meta($id, $post); check($execution->product_destination_url($id) === '', 'array input ignored safely');
// These rows predate every product edit; no membership/case/code rewrite occurs.
foreach ([100,101,102] as $index => $id) {
    $wpdb->items[] = ['id'=>500+$index,'content_product_id'=>$id,'content_name_snapshot'=>'Original card','base_credit_snapshot'=>100000,'upgrade_options_snapshot_json'=>'[]','operations_stage'=>$index===0?'customer_code_issued':'completed','operations_current_credit'=>200000];
    $GLOBALS['meta'][$id][$key] = 'https://old.example/'.$id;
    $GLOBALS['products'][$id] = ['name'=>'Old product '.$id,'full'=>'Old description '.$id,'short'=>'Short '.$id];
}
$items_before = $wpdb->items; $writes_before = $wpdb->writes;
$memberships = new ReflectionMethod($portal, 'customer_memberships');
$user = new WP_User();
$before = $memberships->invoke($portal, $user)[0]['items'];
foreach ([100,101,102] as $index => $id) {
    check($before[$index]['destination_url'] === 'https://old.example/'.$id, 'initial live link');
    check($before[$index]['display_name'] === 'Old product '.$id, 'initial live card title');
    $GLOBALS['meta'][$id][$key] = 'https://updated.example/'.$id.'?a=1&b=2';
    $GLOBALS['products'][$id]['full'] = '<strong>New description '.$id.'</strong><script>alert(1)</script>';
    $GLOBALS['products'][$id]['name'] = 'نام جدید کارت '.$id;
}
$after = $memberships->invoke($portal, $user)[0]['items'];
foreach ([100,101,102] as $index => $id) {
    check($after[$index]['destination_url'] === $GLOBALS['meta'][$id][$key], 'old card reads edited destination');
    check(str_contains($after[$index]['description_html'], 'New description '.$id), 'old card reads edited description');
    check(!str_contains($after[$index]['description_html'], '<script'), 'description sanitized');
    check($after[$index]['base_credit'] === 100000.0 && $after[$index]['credit'] === 200000.0, 'financial snapshot intact');
    check($after[$index]['content_name_snapshot'] === 'Original card', 'card identity intact');
    check($after[$index]['display_name'] === 'نام جدید کارت '.$id, 'renamed product updates old card title at every stage');
    check($execution->customer_wallet_destination(500+$index) === $GLOBALS['meta'][$id][$key], 'compatibility accessor live without assigned code');
}
$GLOBALS['meta'][100][$key] = '';
$cleared = $memberships->invoke($portal, $user)[0]['items'];
check($cleared[0]['destination_url'] === '', 'old card hides cleared destination');
check($execution->customer_wallet_destination(500) === '', 'cleared link never reverts to code snapshot');
check($execution->customer_wallet_destination(9999) === '', 'missing item has no destination');
check($execution->product_destination_url(0) === '', 'missing product has no destination');
$GLOBALS['products'][101]['full'] = ''; $GLOBALS['products'][101]['short'] = 'Updated short description';
$short = $memberships->invoke($portal, $user)[0]['items'];
check(str_contains($short[1]['description_html'], 'Updated short description'), 'short description fallback is live');
$GLOBALS['products'][101]['short'] = '';
check($memberships->invoke($portal, $user)[0]['items'][1]['description_html'] === '', 'cleared descriptions disappear');
unset($GLOBALS['products'][102]); $GLOBALS['fields'][102]['post_content'] = 'Current post content';
check(str_contains($memberships->invoke($portal, $user)[0]['items'][2]['description_html'], 'Current post content'), 'post content fallback');
check($wpdb->items === $items_before && $wpdb->writes === $writes_before, 'display refresh does not mutate cards or execution records');
foreach ([100,101,102] as $index => $id) {
    $GLOBALS['meta'][$id]['_sn_execution_callback_only'] = '1';
    check($memberships->invoke($portal, $user)[0]['items'][$index]['destination_url'] === '', 'callback-only hides link on existing card');
    check($execution->customer_wallet_destination(500+$index) === '', 'callback-only compatibility accessor hides link');
    $GLOBALS['meta'][$id]['_sn_execution_callback_only'] = '0';
    check($memberships->invoke($portal, $user)[0]['items'][$index]['destination_url'] === $execution->product_destination_url($id), 'turning off callback-only restores current product link');
}
check($wpdb->items === $items_before && $wpdb->writes === $writes_before, 'callback-only display never rewrites historical cards');
$destination = new ReflectionMethod($portal, 'render_card_destination');
$html = $destination->invoke($portal, $after[2]['destination_url']);
check(substr_count($html, 'href="https://updated.example/102?a=1&amp;b=2"') === 2, 'destination links escaped');
check(str_contains($html, 'rel="noopener noreferrer"'), 'safe new tab link');
check($destination->invoke($portal, '') === '', 'empty URL produces no destination block');
if (!empty($argv[1])) { file_put_contents($argv[1].'/card-destination.html', $html); }
// Product-less/legacy cards keep the purchased title; no database update required.
$GLOBALS['fields'][102]['post_title'] = 'عنوان فعلی بدون شیء ووکامرس';
check($memberships->invoke($portal,$user)[0]['items'][2]['display_name']==='عنوان فعلی بدون شیء ووکامرس','post-title fallback when WooCommerce product unavailable');
unset($GLOBALS['fields'][102]['post_title']);
check($memberships->invoke($portal,$user)[0]['items'][2]['display_name']==='Original card','missing product retains historic title');
$GLOBALS['products'][101]['name']='  ';
check($memberships->invoke($portal,$user)[0]['items'][1]['display_name']==='Original card','blank current title retains historic name');
$GLOBALS['products'][101]['name']='0';
check($memberships->invoke($portal,$user)[0]['items'][1]['display_name']==='0','zero is a valid nonblank product title');
// Render the actual card template, including list heading, dialog and both alt texts.
$GLOBALS['products'][100]['name']='کارت جدید <script>alert("x")</script> & "عنوان"';
$GLOBALS['products'][100]['image_id']=42;
$membership=$memberships->invoke($portal,$user)[0];
$source=file_get_contents(dirname(__DIR__).'/includes/class-sn-customer-portal.php');
$start=strpos($source,'<?php foreach ( (array) $membership[\'items\'] as $item ) :');
check($start!==false,'actual card template located');
$end=strpos($source,'<?php endforeach; ?>',$start)+strlen('<?php endforeach; ?>');
$markup=substr($source,$start,$end-$start);
$render=function($membership,$user)use($markup){ob_start();eval('?>'.$markup);return ob_get_clean();};
$html=$render->call($portal,$membership,$user);
$safe=esc_html($GLOBALS['products'][100]['name']);
check(str_contains($html,'<h4>'.$safe.'</h4>'),'list heading renders latest escaped name');
check(str_contains($html,'id="sn-customer-card-modal-500-title">'.$safe.'</h3>'),'dialog heading renders latest escaped name');
check(substr_count($html,'alt="'.esc_attr($GLOBALS['products'][100]['name']).'"')===2,'both image alt texts follow current name');
check(!str_contains($html,'<script>'),'title cannot inject HTML');
check($wpdb->items===$items_before&&$wpdb->writes===$writes_before,'renaming never mutates purchase or financial snapshots');
if (!empty($argv[1])) { file_put_contents($argv[1].'/live-card-title.html', $html); }
echo "PASS $checks Biavin live-product regression checks\n";
