<?php
/** Local regressions: actual HR render/query methods with a recording WordPress database double. */
define('ABSPATH', __DIR__ . '/');
define('ARRAY_A', 'ARRAY_A');
define('OBJECT', 'OBJECT');
error_reporting(E_ALL);
set_error_handler(static function($severity, $message, $file, $line) { throw new RuntimeException("$message at $file:$line"); });
class wpdb {
    public string $prefix='wp_', $users='wp_users', $usermeta='wp_usermeta', $last_error='';
    public array $queries=[], $rows=[];
    public int $total=0;
    public bool $fail_indexes=false;
    public function prepare($sql,...$args) { foreach($args as $arg) $sql=preg_replace_callback('/%[ds]/',fn($m)=>$m[0]==='%d'?(string)(int)$arg:"'".str_replace("'","''",$arg)."'",$sql,1); return $sql; }
    public function esc_like($v) { return addcslashes($v,'_%'); }
    public function get_var($sql) {
        $this->queries[]=$sql;
        if(preg_match("/SHOW TABLES LIKE '([^']+)'/",$sql,$m)) return $m[1];
        if(str_starts_with($sql,'SHOW INDEX') && $this->fail_indexes) return null;
        if(str_starts_with($sql,'SHOW COLUMNS') || str_starts_with($sql,'SHOW INDEX')) return 1;
        if(str_contains($sql,'SELECT COUNT(*) FROM wp_sn_hr_profiles')) return $this->total;
        return 0;
    }
    public function get_results($sql,$format=null) {
        $this->queries[]=$sql;
        if(str_contains($sql,'SELECT p.id, p.user_id') && str_contains($sql,'OFFSET')) {
            preg_match('/LIMIT (\d+) OFFSET (\d+)/',$sql,$m);
            return array_slice($this->rows,(int)$m[2],(int)$m[1]);
        }
        return [];
    }
    public function get_row($sql,$format=null) { $this->queries[]=$sql; return null; }
    public function get_col($sql,$col=0) { $this->queries[]=$sql; return []; }
    public function query($sql) { $this->queries[]=$sql; if($this->fail_indexes && str_starts_with($sql,'ALTER')){$this->last_error='index denied';return false;} return 1; }
    public function get_charset_collate() { return ''; }
}
class WP_User { public int $ID=999; public array $roles=['administrator'], $caps=[]; public string $display_name='HR', $user_login='hr'; public function exists(){return false;} }
class SN_Helpers {
    public static function to_english_nums($v) { return strtr($v,['۱'=>'1','۲'=>'2']); }
    public static function tehran_today_gregorian(){return '2026-10-07';}
    public static function __callStatic($name,$args){return (string)($args[0]??'');}
}
function absint($v){return abs((int)$v);}
if (!function_exists('ctype_digit')) { function ctype_digit($v){return preg_match('/^[0-9]+$/',(string)$v)===1;} }
function sanitize_key($v){return preg_replace('/[^a-z0-9_-]/','',strtolower($v));}
function sanitize_text_field($v){return trim(strip_tags($v));}
function sanitize_textarea_field($v){return sanitize_text_field($v);}
function wp_unslash($v){return is_array($v)?array_map('wp_unslash',$v):(is_string($v)?stripslashes($v):$v);}
function wp_slash($v){return is_array($v)?array_map('wp_slash',$v):(is_string($v)?addslashes($v):$v);}
function get_option($k,$d=false){return $GLOBALS['options'][$k]??$d;}
function update_option($k,$v,$autoload=false){$GLOBALS['options'][$k]=$v;$GLOBALS['writes'][]=$k;return true;}
function delete_option($k){unset($GLOBALS['options'][$k]);}
function current_time($type){return '2026-10-07 12:00:00';}
function current_user_can($cap,...$args){return $GLOBALS['allowed'];}
function user_can($id,$cap){return false;}
function is_user_logged_in(){return true;}
function is_admin(){return false;}
function wp_doing_ajax(){return false;}
function get_current_user_id(){return 999;}
function wp_get_current_user(){return new WP_User;}
function get_user_by($field,$id){return new WP_User;}
function get_user_meta($id,$key,$single=true){return '';}
function update_meta_cache($type,$ids){$GLOBALS['meta_batches'][]=$ids;}
function cache_users($ids){$GLOBALS['user_batches'][]=$ids;}
function get_users($args=[]){return [];}
function get_page_by_path($path){return null;}
function wp_json_encode($v,$flags=0){return json_encode($v,$flags);}
function esc_attr($v){return htmlspecialchars((string)$v,ENT_QUOTES);}
function esc_html($v){return esc_attr($v);}
function esc_url($v){return esc_attr($v);}
function esc_textarea($v){return esc_attr($v);}
function esc_sql($v){return addslashes($v);}
function wp_kses_post($v){return $v;}
function admin_url($v){return 'https://crm.example/wp-admin/'.$v;}
function home_url($v='/'){return 'https://crm.example'.$v;}
function wp_nonce_field($action,$name='_wpnonce'){echo '<input type="hidden" name="'.$name.'" value="valid">';}
function wp_create_nonce($action){return 'valid';}
function selected($a,$b,$echo=true){$s=(string)$a===(string)$b?' selected':'';if($echo)echo $s;return $s;}
function checked($a,$b=true,$echo=true){return selected($a,$b,$echo);}
function number_format_i18n($v){return (string)$v;}
function wp_parse_url($v,$part=-1){return parse_url($v,$part);}
function add_query_arg($key,$value=null,$url=null){
    if(is_array($key)){ $args=$key; $base=$value??'https://crm.example/hr/'; } else {$args=[$key=>$value];$base=$url??'https://crm.example/hr/';}
    $parts=parse_url($base);parse_str($parts['query']??'',$q);$q=array_merge($q,$args);
    return 'https://crm.example'.($parts['path']??'/hr/').($q?'?'.http_build_query($q):'');
}
function remove_query_arg($keys,$url=null){return $url??'https://crm.example/hr/';}
function dbDelta($sql){$GLOBALS['wpdb']->query($sql);}
$GLOBALS['allowed']=true;$GLOBALS['writes']=[];$GLOBALS['meta_batches']=[];$GLOBALS['user_batches']=[];
$GLOBALS['wpdb']=$wpdb=new wpdb;
require __DIR__.'/../includes/class-sn-plugin.php';
require __DIR__.'/../includes/class-sn-migration-service.php';
require __DIR__.'/../includes/class-sn-hr-service.php';
$GLOBALS['options']=[
    'sn_db_version'=>SN_Migration_Service::DB_VERSION,
    'sn_db_migration_report'=>['version'=>SN_Migration_Service::DB_VERSION,'table_errors'=>[]],
    'sn_hr_profile_runtime_version'=>'2026-10-07-hr-performance-v1',
    'sn_hr_builtin_lookup_version'=>'2026-10-07-hr-lookups-v1',
];
$checks=0;
function check($v,$message){global $checks;if(!$v)throw new RuntimeException($message);$checks++;}
function plugin(){return (new ReflectionClass(SN_Plugin::class))->newInstanceWithoutConstructor();}
function call_private($object,$name,...$args){return (new ReflectionMethod($object,$name))->invoke($object,...$args);}
$service=new SN_Migration_Service;
$service->migrate_if_needed();check(!$wpdb->queries && !$GLOBALS['writes'],'warm migration performs no SQL/writes');
$p=plugin();$p->sn_ensure_hr_profile_schema_runtime();$p->sn_hr_purge_builtin_lookup_rows_once();
check(!$wpdb->queries && !$GLOBALS['writes'],'warm schema/lookups perform no SQL/writes');
class ProbeMigration extends SN_Migration_Service {public int $runs=0;public function migrate():array{$this->runs++;return [];}}
$probe=new ProbeMigration;$saved=$GLOBALS['options'];
unset($GLOBALS['options']['sn_db_version']);$probe->migrate_if_needed();check($probe->runs===1,'missing DB version runs upgrade');
$GLOBALS['options']=$saved;$GLOBALS['options']['sn_db_migration_report']['table_errors']=['failure'];$probe->migrate_if_needed();check($probe->runs===2,'failed migration is retried');
$GLOBALS['options']=$saved;unset($GLOBALS['options']['sn_hr_profile_runtime_version']);$wpdb->fail_indexes=true;
plugin()->sn_ensure_hr_profile_schema_runtime();check(!isset($GLOBALS['options']['sn_hr_profile_runtime_version']),'failed index installation is not marked ready');
$wpdb->fail_indexes=false;$wpdb->last_error='';plugin()->sn_ensure_hr_profile_schema_runtime();check(isset($GLOBALS['options']['sn_hr_profile_runtime_version']),'successful upgrade is marked ready');
$n=count($wpdb->queries);plugin()->sn_ensure_hr_profile_schema_runtime();check(count($wpdb->queries)===$n,'later request skips full schema check');
$GLOBALS['writes']=[];$wpdb->queries=[];
$_GET=['sn_hr_q'=>'۱۲'];$filters=call_private($p,'sn_hr_panel_profile_filters');check($filters['q']==='12','Persian digits normalized');
for($i=1;$i<=123;$i++)$wpdb->rows[]=(object)['user_id'=>$i];$wpdb->total=123;
$filters['q']='';
$rows=call_private($p,'sn_hr_workforce_rows',$filters,50,50);
check(count($rows)===50 && $rows[0]->user_id===51 && $rows[49]->user_id===100,'page two has correct 50 rows');
$sql=end($wpdb->queries);
check(str_contains($sql,'LIMIT 50 OFFSET 50'),'database pagination');
check(str_contains($sql,'cp_latest') && str_contains($sql,'ch_latest') && str_contains($sql,'a_latest'),'one latest active relationship/history/comp row');
check(count($GLOBALS['meta_batches'])===1 && count($GLOBALS['meta_batches'][0])===50,'metadata is fetched once for displayed page');
check(!in_array('sn_hr_workforce_query_report',$GLOBALS['writes'],true),'successful reads do not write diagnostics');
$n=count($wpdb->queries);call_private($p,'sn_hr_workforce_rows',$filters,50,50);check(count($wpdb->queries)===$n,'page cache avoids repeated SQL');
$filters['q']="O'Reilly";$filters['department_id']=7;$filters['legacy_role']='sn_seller';
call_private(plugin(),'sn_hr_workforce_rows',$filters,50,0);$sql=end($wpdb->queries);
check(str_contains($sql,'p.hr_display_name LIKE') && str_contains($sql,'p.department_id=7') && str_contains($sql,"p.legacy_role='sn_seller'"),'search and organization filters preserved');
$raw=['42'=>['first_name'=>"O'Reilly",'notes'=>"C:\\files",'position_id'=>1]];
$_POST=['rows_json'=>wp_slash(json_encode($raw))];check(wp_unslash(call_private($p,'sn_hr_workforce_rows_from_post'))===$raw,'JSON preserves quotes and backslashes');
$_POST=['rows_json'=>wp_slash('{bad')];check(call_private($p,'sn_hr_workforce_rows_from_post')===[],'malformed JSON fails closed');
$_POST=['rows'=>wp_slash($raw)];check(wp_unslash(call_private($p,'sn_hr_workforce_rows_from_post'))===$raw,'legacy POST supported');
$hr=new SN_HR_Service;$rows=[(object)['user_id'=>42,'position_id'=>1,'position_slug'=>'seller','level_id'=>2,'level_slug'=>'gold','legacy_role'=>'sn_seller']];
$hr->prime_workforce_rows($rows);$n=count($wpdb->queries);$resolved=$hr->resolve_panel_for_user(42);
check(count($wpdb->queries)===$n && $resolved['panel_key']==='seller_panel' && $resolved['level_slug']==='gold','panel resolver uses primed rows without per-user SQL');
$wpdb->rows=[];$wpdb->total=0;$_POST=[];
$tabs=['hr-workforce','hr-manual-add','hr-csv','hr-bulk','hr-change-requests','hr-compensation','hr-positions','hr-extra','hr-overview','hr-structure','hr-hierarchy','hr-logs'];
$out=$argv[1]??null;if($out && !is_dir($out))mkdir($out,0700,true);
foreach($tabs as $tab){
    $_GET=['sn_hr_tab'=>$tab];$wpdb->queries=[];$GLOBALS['writes']=[];
    $html=plugin()->render_hr_panel();
    check(substr_count($html,'data-sn-tab-panel=')===1,'only one section rendered: '.$tab);
    check(str_contains($html,'data-sn-tab-panel="'.$tab.'"') && str_contains($html,'sn-tab-panel-active'),'requested section is active: '.$tab);
    check(substr_count($html,'data-sn-hr-tab=')===12,'all 12 tabs reachable: '.$tab);
    check(!array_filter($wpdb->queries,fn($q)=>str_starts_with($q,'ALTER') || str_starts_with($q,'CREATE')),'no repeated DDL: '.$tab);
    if($tab==='hr-logs')check(!array_filter($wpdb->queries,fn($q)=>str_contains($q,'SELECT p.id, p.user_id')),'logs do not query workforce');
    if($out)file_put_contents($out.'/'.$tab.'.html',$html);
}
$defaults=array_fill_keys(['employee_code','employment_type','work_phone','emergency_phone','hire_date','user_email','comp_salary_currency','rule_title','rule_code','comp_effective_from','history_effective_from'], '');
$defaults+=array_fill_keys(['manager_user_id','manager_position_slug','manager_position_label','comp_base_salary','base_salary','default_commission_rule_id','current_history_id'],0);
$defaults+=['position_id'=>1,'position_slug'=>'seller','position_label'=>'فروشنده','level_id'=>2,'level_slug'=>'gold','level_label'=>'طلایی','employment_status'=>'active','salary_currency'=>'IRT','is_active'=>1,'commission_enabled'=>1,'commission_override_mode'=>'inherit','legacy_role'=>'sn_seller'];
for($i=1;$i<=123;$i++)$wpdb->rows[]=(object)array_merge($defaults,['id'=>$i,'user_id'=>$i,'user_login'=>'0912'.$i,'display_name'=>'نیرو '.$i]);
$wpdb->total=123;$_GET=['sn_hr_tab'=>'hr-workforce','sn_hr_page'=>'2'];$wpdb->queries=[];
$html=plugin()->render_hr_panel();
check(substr_count($html,'<tr data-sn-user-id=')===50,'real populated renderer generates only 50 editing forms');
check(str_contains($html,'<tr data-sn-user-id="51"') && !str_contains($html,'<tr data-sn-user-id="1"'),'real renderer serves page two');
check(str_contains($html,'صفحه 2 از 3') && str_contains($html,'sn_hr_page=3'),'pagination reaches all 123 users');
check(!array_filter($wpdb->queries,fn($q)=>str_contains($q,'SELECT * FROM wp_sn_hr_profiles WHERE user_id=')),'populated renderer does not query a profile for each user');
if($out)file_put_contents($out.'/hr-workforce-page2.html',$html);
$_GET=['sn_hr_tab'=>'hr-workforce','sn_hr_page'=>'9999'];$html=plugin()->render_hr_panel();check(substr_count($html,'<tr data-sn-user-id=')===23 && str_contains($html,'صفحه 3 از 3'),'out-of-range page clamps to last page');
$wpdb->rows=[];$wpdb->total=0;
$_GET=['sn_hr_tab'=>'<script>'];$html=plugin()->render_hr_panel();check(str_contains($html,'data-sn-active-tab="hr-workforce"'),'invalid tab falls back');
$GLOBALS['allowed']=false;$wpdb->queries=[];$html=plugin()->render_hr_panel();check(str_contains($html,'دسترسی') && !$wpdb->queries,'unauthorized users perform no HR SQL');
echo "PASS $checks HR PHP performance/regression checks\n";
