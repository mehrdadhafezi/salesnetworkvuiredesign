<?php
/** Focused tests with WordPress boundaries stubbed; run: php tools/test-sales-catalog.php */
define('ABSPATH', __DIR__);
class WP_Error { public function __construct(public string $code, public string $message) {} public function get_error_message(){return $this->message;} }
function is_wp_error($v){return $v instanceof WP_Error;}
function absint($v){return abs((int)$v);}
function user_can($id,$cap){return $id===999;}
function get_user_meta($id,$key,$single){return $GLOBALS['meta'][$id][$key]??'';}
function get_user_by($kind,$id){return (object)['ID'=>$id,'roles'=>$id===10||$id===20?['sn_sales_manager']:['sn_seller'],'display_name'=>'User '.$id];}
function get_the_title($id){return 'Product '.$id;}
function wp_json_encode($value){return json_encode($value);}
class SN_Helpers {
 public static function to_english_nums($s){return strtr($s,['۰'=>'0','۱'=>'1','۲'=>'2','۳'=>'3','۴'=>'4','۵'=>'5','۶'=>'6','۷'=>'7','۸'=>'8','۹'=>'9']);}
 public static function format_price($v){return $v.' تومان';}
}
class CatalogTestDb {
 public string $prefix='wp_'; public array $profiles=[]; public array $parents=[]; public bool $hr=false;
 public array $writes=[];
 function update($table,$data,$where){$this->writes[]=[$table,$data,$where];return 1;}
 function esc_like($s){return $s;}
 function prepare($sql,...$args){foreach($args as $arg){$sql=preg_replace('/%[ds]/',is_int($arg)?(string)$arg:"'".$arg."'",$sql,1);}return $sql;}
 function get_var($sql){if($this->hr && preg_match("/SHOW TABLES LIKE '(.*?)'/",$sql,$m))return $m[1];return null;}
 function get_row($sql){
   if(preg_match('/WHERE p.user_id=(\d+)/',$sql,$m))return $this->profiles[(int)$m[1]]??null;
   if(preg_match('/a.child_profile_id=(\d+)/',$sql,$m))return $this->profiles[$this->parents[(int)$m[1]]??0]??null;
   throw new RuntimeException('Unexpected query: '.$sql);
 }
}
$wpdb=new CatalogTestDb();
require dirname(__DIR__).'/includes/class-sn-sales-catalog.php';
$catalog=SN_Sales_Catalog::instance(); $checks=0;
function check($condition,$label){global $checks;if(!$condition){throw new RuntimeException('FAIL '.$label);} $checks++;}
$GLOBALS['meta']=[11=>['sn_supervisor_id'=>12],12=>['sn_sales_manager_id'=>10],21=>['sn_sales_manager_id'=>20]];
$policy=['mode'=>'selected','products'=>[100=>['discounts'=>['10000','20000']],101=>['discounts'=>['125']]],'revision'=>'a'];
$GLOBALS['meta'][10]['_sn_sales_catalog_policy']=$policy;
$GLOBALS['meta'][20]['_sn_sales_catalog_policy']=['mode'=>'selected','products'=>[200=>['discounts'=>['5000']]]];
check($catalog->manager_for_user(11)===10,'nested legacy team');
check($catalog->manager_for_user(10)===10,'manager himself');
check($catalog->product_allowed(100,11),'assigned product');
check(!$catalog->product_allowed(200,11),'other team product denied');
check(!$catalog->product_allowed(100,21),'product not shared between managers');
check($catalog->product_allowed(200,999),'admin product override');
check(is_wp_error($catalog->validate_products([100],999,21)),'admin acting for owner respects owner catalog');
check(count($catalog->filter_products([['id'=>100],['id'=>200]],11))===1,'picker filtering');
$items=[['product_id'=>100,'product_name'=>'A','qty'=>3,'unit_price'=>1000],['product_id'=>101,'product_name'=>'B','qty'=>2,'unit_price'=>10.25]];
$result=$catalog->price_items($items,['20000','125'],11);
check(!is_wp_error($result),'valid pricing');
check($result['original_total']===3020.5,'gross including decimals');
check($result['discount_total']===602.5,'per-unit multiplication');
check($result['final_total']===2418.0 || $result['final_total']===2418,'net exact');
check(count($result['snapshot'])===2,'one snapshot per discounted row');
check($result['snapshot'][0]['manager_user_id']===10,'snapshot owner');
check($catalog->price_items($items,[],11)['discount_total']===0,'no discount optional');
check(is_wp_error($catalog->price_items($items,['5000'],11)),'other manager discount rejected');
check(is_wp_error($catalog->price_items($items,['99999'],11)),'forged discount rejected');
check(is_wp_error($catalog->price_items($items,['1e4'],11)),'scientific notation rejected');
check(is_wp_error($catalog->price_items($items,'10000',11)),'malformed selections rejected');
check(is_wp_error($catalog->price_items($items,[[10000]],11)),'nested array rejected');
check(is_wp_error($catalog->price_items($items,[3=>'10000'],11)),'nonexistent line rejected');
check(is_wp_error($catalog->price_items($items,[-1=>'10000'],11)),'negative index rejected');
check(is_wp_error($catalog->price_items([['product_id'=>100,'qty'=>1,'unit_price'=>100]],['10000'],11)),'discount equal to price rejected');
check(is_wp_error($catalog->price_items([['product_id'=>100,'qty'=>1000000,'unit_price'=>1000000000000]],[],11)),'overflow rejected');
check(SN_Sales_Catalog::amount_key('۱۲۳٬۴۵۶٫۷۸')==='12345678','Persian money');
foreach(['0','-1','1e3','1.001',[], '10000000000000'] as $bad){check(SN_Sales_Catalog::amount_key($bad)==='','invalid money rejected');}
$saved=(object)['catalog_discounts_snapshot'=>json_encode($result['snapshot'])];
$GLOBALS['meta'][10]['_sn_sales_catalog_policy']['products'][100]['discounts']=[];
check(is_wp_error($catalog->price_items($items,['20000'],11)),'removed draft discount rejected');
check(SN_Sales_Catalog::invoice_snapshot($saved)===$result['snapshot'],'issued snapshot survives policy change');
$GLOBALS['meta'][10]['_sn_sales_catalog_policy']=['mode'=>'selected','products'=>[]];
check(!$catalog->product_allowed(100,11),'empty selected means no products');
$GLOBALS['meta'][10]['_sn_sales_catalog_policy']=['mode'=>'all','products'=>[]];
check($catalog->product_allowed(100,11),'all mode migration compatibility');
$wpdb->hr=true;
$wpdb->profiles=[31=>(object)['id'=>31,'user_id'=>31,'slug'=>'converter'],32=>(object)['id'=>32,'user_id'=>32,'slug'=>'supervisor'],30=>(object)['id'=>30,'user_id'=>30,'slug'=>'sales_manager'],41=>(object)['id'=>41,'user_id'=>41,'slug'=>'seller']];
$wpdb->parents=[31=>32,32=>30,41=>41];$GLOBALS['meta'][31]['sn_sales_manager_id']=10;$GLOBALS['meta'][41]['sn_sales_manager_id']=10;
check($catalog->manager_for_user(31)===30,'current HR overrides stale metadata');
check($catalog->manager_for_user(41)===0,'HR cycle terminates without stale fallback');
// Real Dot draft refresh: keep fixed discount while updating live gross price,
// then freeze both price and snapshot as soon as any payment link exists.
function get_option($key,$default=null){return $GLOBALS['options'][$key]??$default;}
function sanitize_key($v){return preg_replace('/[^a-z0-9_\-]/','',strtolower($v));}
function sanitize_hex_color($v){return $v;}
function current_time($format){return '2026-10-05 10:00:00';}
require dirname(__DIR__).'/includes/class-sn-dot-flow.php';
$GLOBALS['options']['sn_dot_config']=['options'=>[['key'=>'a','product_id'=>100,'price'=>1500,'active'=>1,'title'=>'A']]];
$case=(object)['id'=>5,'options_snapshot_json'=>json_encode([['key'=>'a','product_id'=>100,'price'=>1000]]),'selected_option_key'=>'a','selected_option_product_id'=>100,'selected_option_price'=>800,'paid_amount'=>0,'catalog_discount_snapshot'=>json_encode($result['snapshot'][0])];
// Dot cases are one unit; use an actual one-unit snapshot.
$draft=$result['snapshot'][0];$draft['qty']=1;$draft['discount_total']=200;$draft['original_line_total']=1000;$draft['final_line_total']=800;
$case->catalog_discount_snapshot=json_encode($draft);
$method=new ReflectionMethod(SN_Dot_Flow::class,'sync_live_case_option_prices');$method->setAccessible(true);
$case=$method->invoke(SN_Dot_Flow::instance(),$case,false);
check($case->selected_option_price===1300.0,'draft net follows live gross minus fixed amount');
check($case->remaining_amount===1300.0,'draft remaining uses discounted total');
check(json_decode($case->catalog_discount_snapshot,true)['original_unit_price']===1500,'draft snapshot updates gross');
$GLOBALS['options']['sn_dot_config']['options'][0]['price']=2200;
$before=serialize($case);$writeCount=count($wpdb->writes);
$case=$method->invoke(SN_Dot_Flow::instance(),$case,true);
check(serialize($case)===$before,'issued case ignores subsequent price changes');
check(count($wpdb->writes)===$writeCount,'issued case refresh does not write');
echo "PASS {$checks} catalog and conversion checks\n";
