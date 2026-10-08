<?php
/** Run: php tools/test-kavenegar-switch.php [fixture-directory] */
define('ABSPATH', __DIR__);
define('SN_VERSION', '2.0.152');
error_reporting(E_ALL);
set_error_handler(static function($severity,$message,$file,$line) { throw new RuntimeException("$message at $file:$line"); });
class WP_Error {
    public function __construct(private string $code, private string $message) {}
    public function get_error_code() { return $this->code; }
    public function get_error_message() { return $this->message; }
}
class SN_Helpers {
    public static array $json = [];
    public static function normalize_mobile($value) { return str_replace('+98','0', self::to_english_nums($value)); }
    public static function to_english_nums($value) { return strtr((string)$value,['۰'=>'0','۱'=>'1','۲'=>'2','۳'=>'3','۴'=>'4','۵'=>'5','۶'=>'6','۷'=>'7','۸'=>'8','۹'=>'9']); }
    public static function is_valid_mobile($value) { return (bool)preg_match('/^09\d{9}$/',$value); }
    public static function format_price($value) { return number_format($value).' تومان'; }
    public static function send_json($success,$message,$data=[]) { self::$json = ['success'=>$success,'message'=>$message,'data'=>$data]; }
}
function get_option($name,$default=false) { return array_key_exists($name,$GLOBALS['options']) ? $GLOBALS['options'][$name] : $default; }
function update_option($name,$value,$autoload=null) { $GLOBALS['options'][$name]=$value; }
function is_wp_error($value) { return $value instanceof WP_Error; }
function sanitize_key($value) { return preg_replace('/[^a-z0-9_-]/','',strtolower($value)); }
function sanitize_text_field($value) { return trim(strip_tags($value)); }
function sanitize_textarea_field($value) { return trim(strip_tags($value)); }
function wp_unslash($value) { return is_string($value) ? stripslashes($value) : $value; }
function add_filter(...$args) {}
function wp_remote_post($url,$args) { $GLOBALS['calls'][]=['url'=>$url,'args'=>$args]; return $GLOBALS['response']; }
function wp_remote_retrieve_body($response) { return $response['body']; }
function wp_remote_retrieve_response_code($response) { return $response['response']['code']; }
function wp_json_encode($value,$flags=0) { return json_encode($value,$flags); }
function wp_parse_url($url,$component) { return parse_url($url,$component); }
function current_time($format) { return '2026-10-07 09:00:00'; }
function home_url($path) { return 'https://crm.example'.$path; }
function current_user_can($cap) { return $GLOBALS['authorized']; }
function check_ajax_referer(...$args) { return $GLOBALS['nonce_ok']; }
function esc_attr($value) { return htmlspecialchars((string)$value,ENT_QUOTES); }
function esc_html($value) { return esc_attr($value); }
function esc_textarea($value) { return esc_attr($value); }
function selected($a,$b) { if($a===$b) echo 'selected'; }
function checked($a,$b) { if($a===$b) echo 'checked'; }
function absint($value) { return abs((int)$value); }
$GLOBALS['options']=[]; $GLOBALS['calls']=[]; $GLOBALS['authorized']=true; $GLOBALS['nonce_ok']=true;
require dirname(__DIR__).'/includes/class-sn-sms.php';
require dirname(__DIR__).'/includes/class-sn-plugin.php';
$plugin=(new ReflectionClass(SN_Plugin::class))->newInstanceWithoutConstructor();
$checks=0;
function check($condition,$label) { global $checks; if(!$condition) throw new RuntimeException('FAIL '.$label); ++$checks; }
function accepted($status=5) { return ['response'=>['code'=>200],'body'=>json_encode(['return'=>['status'=>200,'message'=>'OK'],'entries'=>[['messageid'=>12345,'status'=>$status]]])]; }
function last_call() { return end($GLOBALS['calls']); }
function reset_requests($response=null) { $GLOBALS['calls']=[]; $GLOBALS['response']=$response ?? accepted(); }
$phone='09123456789';
// Migrate a shared-only active Faraz installation before switching.
$GLOBALS['options']=['sn_sms_provider'=>'faraz','sn_sms_api_key'=>'legacy-faraz','sn_sms_sender'=>'3000','sn_faraz_pattern_invoice'=>'FarazInvoice','sn_faraz_pattern_customer_otp'=>'FarazLogin'];
check(SN_SMS::provider_configuration('faraz')['api_key']==='legacy-faraz','legacy active key preserved');
check(SN_SMS::provider_configuration('kavenegar')['api_key']==='','inactive gateway cannot borrow legacy key');
$_POST=['sn_sms_provider'=>'kavenegar','sn_sms_api_key_kavenegar'=>'kave-key','sn_sms_sender_kavenegar'=>'1000','sn_kavenegar_template_customer_otp'=>'BiawinLogin','sn_kavenegar_template_invoice'=>'Invoice','sn_kavenegar_template_online_payment'=>'OnlinePaid','sn_kavenegar_template_card_payment'=>'CardPaid','sn_kavenegar_template_payment_reward'=>'Reward'];
$plugin->ajax_save_settings();
check(SN_Helpers::$json['success'],'authenticated settings save');
check(get_option('sn_sms_api_key_faraz')==='legacy-faraz','switch snapshots old legacy key');
check(get_option('sn_sms_api_key')==='kave-key','shared compatibility mirror updated');
check(get_option('sn_sms_credentials_provider')==='kavenegar','credential ownership recorded');
foreach(['customer_otp','invoice','online_payment','card_payment','payment_reward'] as $event) { check(get_option('sn_kavenegar_template_'.$event)!=='','template persists '.$event); }
$sms=new SN_SMS();
reset_requests(); check($sms->send($phone,'سلام'),'Kavenegar plain message accepted');
$call=last_call();
check($call['url']==='https://api.kavenegar.com/v1/kave-key/sms/send.json','correct simple endpoint and key');
check($call['args']['body']===['receptor'=>$phone,'message'=>'سلام','sender'=>'1000'],'simple payload');
check($call['args']['sslverify']===true && $call['args']['redirection']===0,'TLS verified and redirects disabled');
check($sms->allow_sms_hosts(false,'api.kavenegar.com'),'allowlisted Kavenegar host');
check(!$sms->allow_sms_hosts(false,'api.kavenegar.com.evil.example'),'exact host allowlist');
reset_requests(); check($sms->send_customer_portal_otp('+989123456789','۱۲۳۴۵۶'),'OTP normalization and lookup');
$call=last_call();
check(str_ends_with($call['url'],'/verify/lookup.json'),'OTP uses verify lookup');
check($call['args']['body']===['receptor'=>$phone,'template'=>'BiawinLogin','type'=>'sms','token'=>'123456'],'OTP payload');
check(count($GLOBALS['calls'])===1,'OTP single request');
reset_requests(); check(!$sms->send_customer_portal_otp($phone,'12345'),'invalid OTP rejected'); check(!$GLOBALS['calls'],'invalid OTP never sent');
reset_requests(); check($sms->send_invoice_link($phone,'INV-1','https://crm.example/i/abc/','مشتری تست','100,000','6037990000000000'),'invoice lookup');
$payload=last_call()['args']['body'];
check($payload['token']==='INV-1' && $payload['token2']==='https://crm.example/i/abc/' && $payload['token3']==='100000','invoice tokens normalized');
check($payload['token10']==='مشتری تست' && $payload['token20']==='6037990000000000','multiword name and card mappings');
$preview=$sms->build_invoice_link_preview($phone,'INV-1','https://crm.example/i/abc/','مشتری تست','100,000','6037990000000000');
check($preview['payload']===$payload,'preview matches outgoing invoice');
check(!str_contains(json_encode($preview),'kave-key'),'preview has no credential');
$GLOBALS['options']['sn_card_to_card_enabled']='0';
$preview=$sms->build_invoice_link_preview($phone,'INV-1','https://crm.example/i/abc/','مشتری تست','100000','6037990000000000');
check(!isset($preview['payload']['token20']),'card disabled does not leak payment card');
$GLOBALS['options']['sn_card_to_card_enabled']='1';
$payment=new ReflectionMethod($plugin,'sn_maybe_send_payment_sms');
$invoice=(object)['id'=>1,'lead_id'=>0,'invoice_code'=>'INV-1','customer_name'=>'مشتری تست','customer_phone'=>$phone,'product_price'=>100000,'short_code'=>'abc'];
reset_requests(); $payment->invoke($plugin,$invoice,'online');
check(last_call()['args']['body']['template']==='OnlinePaid','actual online-payment callback routes to Kavenegar');
reset_requests(); $payment->invoke($plugin,$invoice,'card_to_card');
check(last_call()['args']['body']['template']==='CardPaid' && last_call()['args']['body']['token3']==='card','actual card-payment callback routes to Kavenegar');
$reward=new ReflectionMethod($plugin,'sn_send_payment_reward_sms');
reset_requests(); $result=$reward->invoke($plugin,$phone,$invoice,'کفش راحتی');
check($result['sent']===0 && $result['status']==='disabled' && !$GLOBALS['calls'],'retired reward callback never sends SMS');
$has_reward = new ReflectionMethod($plugin, 'sn_invoice_has_payment_reward');
check(!$has_reward->invoke($plugin, $invoice), 'old invoice flags cannot enable retired rewards');
foreach (['faraz', 'kavenegar'] as $provider) {
    update_option('sn_sms_provider', $provider); update_option('sn_payment_reward_sms_enabled', '1');
    reset_requests(); $result = $reward->invoke($plugin, $phone, $invoice, 'کفش');
    check($result['status'] === 'disabled' && !$GLOBALS['calls'], 'no reward SMS through '.$provider);
}
update_option('sn_sms_provider', 'kavenegar');
$_POST=['invoice_code'=>'INV-1', 'reward_phone'=>$phone, 'nonce'=>'valid'];
$plugin->ajax_submit_payment_reward_contact();
check(!SN_Helpers::$json['success'] && str_contains(SN_Helpers::$json['message'], 'غیرفعال'), 'direct reward AJAX request rejected before data access');
reset_requests(); $_POST=['phone'=>$phone,'test_type'=>'otp']; $plugin->ajax_test_sms();
check(SN_Helpers::$json['success'] && SN_Helpers::$json['data']['provider']==='kavenegar','admin OTP test routes to selected provider');
check(last_call()['args']['body']['token']==='123456','test code is separate from login challenge');
foreach([['code'=>200,'status'=>403],['code'=>503,'status'=>200]] as $failure) {
    reset_requests(['response'=>['code'=>$failure['code']],'body'=>json_encode(['return'=>['status'=>$failure['status'],'message'=>'bad kave-key'],'entries'=>[['messageid'=>1,'status'=>5]]])]);
    check(!$sms->send($phone,'پیام'),'HTTP and API acceptance both required');
    check(count($GLOBALS['calls'])===1,'rejected response has no fallback/retry');
    check(!str_contains($sms->get_last_error(),'kave-key'),'API error redacts credential');
}
foreach([6,11,13,14,100] as $status) { reset_requests(accepted($status)); check(!$sms->send($phone,'پیام'),'delivery rejection '.$status); }
foreach([1,2,4,5,10] as $status) { reset_requests(accepted($status)); check($sms->send($phone,'پیام'),'accepted queued or delivered '.$status); }
foreach(['not-json',json_encode(['return'=>['status'=>200],'entries'=>[]]),json_encode(['return'=>['status'=>200],'entries'=>[['status'=>5]]])] as $body) {
    reset_requests(['response'=>['code'=>200],'body'=>$body]); check(!$sms->send($phone,'پیام'),'malformed success is not accepted');
}
reset_requests(new WP_Error('http_request_failed','https://api.kavenegar.com/v1/kave-key timeout'));
check(!$sms->send($phone,'پیام'),'transport timeout rejected');
check(!str_contains($sms->get_last_error(),'kave-key') && count($GLOBALS['calls'])===1,'timeout neither exposes key nor duplicates request');
reset_requests(); check(!$sms->send_kavenegar_pattern($phone,'Bad_name',['token'=>'123456']),'invalid template rejected locally');
check(!$GLOBALS['calls'],'invalid template has no request');
foreach([['token'=>'a b'],['token'=>str_repeat('x',101)],['token'=>['nested']],['token'=>'123','token10'=>"a\nb"],['token'=>'123','token20'=>'1 2 3 4 5 6 7 8 9 10']] as $tokens) {
    reset_requests(); check(!$sms->send_kavenegar_pattern($phone,'Template',$tokens),'invalid token rejected'); check(!$GLOBALS['calls'],'invalid token does not call service');
}
reset_requests(); check(!$sms->send_faraz_pattern($phone,'FarazPattern',['code'=>'123']),'Faraz boundary guarded on Kavenegar'); check(!$GLOBALS['calls'],'Faraz API never contacted with Kavenegar key');
// Switching back needs no re-entry and uses only Faraz's preserved credentials.
check(SN_SMS::save_provider_settings_from_request(['sn_sms_provider'=>'faraz'])===null,'switch back without credential entry');
$faraz=new SN_SMS(); reset_requests(['response'=>['code'=>201],'body'=>json_encode(['status'=>'success'])]);
check($faraz->send_customer_portal_otp($phone,'123456'),'existing Faraz OTP still sends');
check(last_call()['args']['headers']['Api-Key']==='legacy-faraz','Faraz uses its key');
check(json_decode(last_call()['args']['body'],true)['code']==='FarazLogin','Faraz keeps its template');
check(get_option('sn_sms_api_key_kavenegar')==='kave-key','switch back keeps inactive Kavenegar key');
// Clearing a gateway is explicit; never borrow the old shared key.
check(SN_SMS::save_provider_settings_from_request(['sn_sms_provider'=>'kavenegar','sn_sms_api_key_kavenegar'=>''])===null,'clear active key');
$blank=new SN_SMS(); reset_requests(); check(!$blank->send($phone,'پیام'),'missing key rejected'); check(!$GLOBALS['calls'],'missing key prevents outbound request');
check(get_option('sn_sms_api_key')==='','stale shared key cleared');
$before=$GLOBALS['options']; check(is_wp_error(SN_SMS::save_provider_settings_from_request(['sn_sms_provider'=>['kavenegar']])),'provider array rejected');
check(is_wp_error(SN_SMS::save_provider_settings_from_request(['sn_sms_provider'=>'invalid'])),'unknown provider rejected');
check($GLOBALS['options']===$before,'invalid provider does not mutate settings');
check(is_wp_error(SN_SMS::save_provider_settings_from_request(['sn_sms_provider'=>'faraz','sn_sms_api_key_faraz'=>['invalid']])),'malformed credential rejected');
// Legacy installations with empty individual fields must retain their active key.
$GLOBALS['options']=['sn_sms_provider'=>'faraz','sn_sms_api_key'=>'old-shared','sn_sms_sender'=>'3001','sn_sms_api_key_faraz'=>'','sn_sms_api_key_kavenegar'=>''];
check(SN_SMS::provider_configuration('faraz')['api_key']==='old-shared','empty legacy field preserves active shared credentials');
SN_SMS::save_provider_settings_from_request(['sn_sms_provider'=>'kavenegar']);
check(get_option('sn_sms_api_key_faraz')==='old-shared','empty legacy field migrated before switch');
reset_requests(); check(!(new SN_SMS())->send($phone,'پیام') && !$GLOBALS['calls'],'unconfigured secondary cannot borrow old primary key');
// Meli's legacy user:password credentials and line survive both switches.
$GLOBALS['options']=['sn_sms_provider'=>'melipayamak','sn_sms_api_key'=>'meli-user:meli-pass','sn_sms_sender'=>'5000','sn_sms_api_key_kavenegar'=>'kave-key','sn_sms_sender_kavenegar'=>'1000'];
reset_requests(['response'=>['code'=>200],'body'=>json_encode(['RetStatus'=>1])]);
check((new SN_SMS())->send($phone,'پیام'),'legacy Meli send works');
check(last_call()['args']['body']['username']==='meli-user' && last_call()['args']['body']['password']==='meli-pass','Meli legacy credentials resolved');
SN_SMS::save_provider_settings_from_request(['sn_sms_provider'=>'kavenegar']);
SN_SMS::save_provider_settings_from_request(['sn_sms_provider'=>'melipayamak']);
reset_requests(['response'=>['code'=>200],'body'=>json_encode(['RetStatus'=>1])]);
check((new SN_SMS())->send($phone,'پیام'),'Meli works after switching back');
check(last_call()['args']['body']['from']==='5000','Meli retains its sender');
check(get_option('sn_sms_api_key_kavenegar')==='kave-key','Meli switch keeps Kavenegar key');

// Authorization remains at the actual AJAX endpoints.
$GLOBALS['authorized']=false; $before=$GLOBALS['options']; $_POST=['sn_sms_provider'=>'faraz'];
$plugin->ajax_save_settings(); check(!SN_Helpers::$json['success'] && $GLOBALS['options']===$before,'settings require admin authorization');
$_POST=['phone'=>$phone,'test_type'=>'otp']; reset_requests(); $plugin->ajax_test_sms();
check(!SN_Helpers::$json['success'] && !$GLOBALS['calls'],'send test requires admin authorization');
$GLOBALS['authorized']=true; $GLOBALS['nonce_ok']=false; $plugin->ajax_test_sms();
check(!SN_Helpers::$json['success'] && !$GLOBALS['calls'],'send test requires valid nonce');
$GLOBALS['nonce_ok']=true;
// Render the real SMS settings fragment for DOM tests, without a live WP site.
if(!empty($argv[1])) {
    $source=file_get_contents(dirname(__DIR__).'/includes/class-sn-plugin.php');
    $start=strpos($source,'<div class="sn-admin-tab-panel" id="sn-settings-sms">');
    $end=strpos($source,'<div class="sn-admin-tab-panel" id=', $start+1);
    $fragment=substr($source,$start,$end-$start);
    preg_match_all('/\$s\[\x27([^\x27]+)\x27\]/',$fragment,$keys);
    $s=[]; foreach(array_unique($keys[1]) as $key) { $s[$key]=get_option($key,''); }
    $s['sn_sms_provider']='faraz';
    ob_start(); eval('?>'.$fragment); $html=ob_get_clean();
    preg_match('/function showProvider\(\).*?showProvider\(\);.*?showProvider\);/s',$source,$js);
    file_put_contents($argv[1].'/sms-settings.html',$html.'<script>jQuery(function($){'.$js[0].'});</script>');
}
echo "PASS $checks Kavenegar/provider-switch regression checks\n";
