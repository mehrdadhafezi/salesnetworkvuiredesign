<?php
// Execute the production save method against a transactional in-memory wpdb double.
define('ABSPATH',__DIR__.'/'); define('ARRAY_A','ARRAY_A');
function esc_url_raw($url,...$args){return $url;}
function wp_unslash($x){return $x;} function sanitize_text_field($x){return $x;} function sanitize_key($x){return $x;}
function wp_json_encode($x){return json_encode($x);} function current_time($x){return '2026-10-06 12:00:00';} function get_current_user_id(){return 7;}
require __DIR__.'/../includes/class-sn-helpers.php';
require __DIR__.'/../includes/class-sn-payment-transactions.php';
$source=file_get_contents(__DIR__.'/../includes/class-sn-plugin.php');
$start=strpos($source,'private function sn_save_invoice_manual_payment(');
$end=strpos($source,'private function sn_supervisor_can_upload_receipt_for_status',$start);
$save=substr($source,$start,$end-$start);
$save=substr($save,0,strrpos($save,'/**'));
$save=preg_replace('/^private function/','public function',$save);
$stub= <<<'CODE'
class SaveHarness {
private function sn_card_to_card_enabled(){return true;}
private function sn_ensure_payment_migrations(){}
private function sn_invoice_payment_totals($invoice){return ['due'=>1000000.0,'remaining'=>1000000.0,'stage_no'=>1,'plan'=>'full'];}
private function sn_filter_existing_columns_fresh($table,$data){return $data;}
private function sn_log_activity(...$args){}
CODE;
eval($stub.$save.'}');
class MemoryDB {
 public $prefix='wp_', $last_error='', $insert_id=0, $payments=[], $fail_insert=false;
 public $invoice, $stage, $snapshot, $queries=[];
 function __construct(){ $this->invoice=(object)['id'=>1,'status'=>'pending','lead_id'=>2]; $this->stage=(object)['id'=>1,'status'=>'pending']; }
 function prepare($sql,...$args){ foreach($args as $v){ $sql=preg_replace('/%[ds]/',is_numeric($v)?(string)$v:"'".$v."'",$sql,1); } return $sql; }
 function query($sql){$this->queries[]=$sql; if($sql==='START TRANSACTION')$this->snapshot=serialize([$this->invoice,$this->stage,$this->payments]); if($sql==='ROLLBACK')[$this->invoice,$this->stage,$this->payments]=unserialize($this->snapshot);return 1;}
 function get_row($sql){if(strpos($sql,'FROM wp_sn_invoices')!==false)return clone $this->invoice; if(strpos($sql,'FROM wp_sn_invoice_payment_stages')!==false)return clone $this->stage; if(preg_match("/submission_key='([^']+)'/",$sql,$m))foreach($this->payments as $r)if($r['submission_key']===$m[1])return (object)$r; return null;}
 function get_results($sql,$mode){return array_values(array_filter($this->payments,fn($p)=>$p['status']==='pending'));}
 function update($table,$data,$where){$obj=$table==='wp_sn_invoices'?$this->invoice:$this->stage;foreach($data as $k=>$v)$obj->$k=$v;return 1;}
 function insert($table,$row){if($this->fail_insert)return false;$row['id']=++$this->insert_id;$this->payments[]=$row;return 1;}
}
function check($v,$name){if(!$v)throw new RuntimeException($name);echo "PASS $name\n";}
$wpdb=new MemoryDB();$h=new SaveHarness();
$_POST=['amount'=>'۴۰۰٬۰۰۰','submission_key'=>'first_transfer_key_123'];
$r=$h->sn_save_invoice_manual_payment(1,'seller_manual',['manual_transfer_type'=>'paya','manual_account_last6'=>null,'manual_account_owner'=>'First Owner','manual_tracking_last6'=>'123456']);
check($r['success']&&count($wpdb->payments)===1,'first bank transaction inserted: '.json_encode($r,JSON_UNESCAPED_UNICODE));
check($wpdb->payments[0]['submission_key']==='first_transfer_key_123','bank validation preserves idempotency key');
$_POST=['amount'=>'600000','submission_key'=>'second_transfer_key_456'];
$r=$h->sn_save_invoice_manual_payment(1,'supervisor_manual',['receipt_url'=>'https://example.test/receipt.pdf','receipt_urls'=>['https://example.test/receipt.pdf']]);
check($r['success']&&count($wpdb->payments)===2,'second receipt transaction appended');
check($wpdb->payments[0]['amount']===400000.0&&$wpdb->payments[1]['amount']===600000.0,'independent amounts retained');
check(json_decode($wpdb->payments[0]['proof_data'],true)['manual_account_owner']==='First Owner','earlier bank information retained');
check($wpdb->invoice->manual_amount===1000000.0,'invoice projection holds total');
$r=$h->sn_save_invoice_manual_payment(1,'supervisor_manual',['receipt_url'=>'https://example.test/retry.pdf']);
check($r['success']&&!empty($r['already'])&&count($wpdb->payments)===2,'retry after balance is full does not duplicate');
$_POST=['amount'=>'1','submission_key'=>'third_transfer_key_789'];
$r=$h->sn_save_invoice_manual_payment(1,'seller_manual',[]);
check(!$r['success']&&count($wpdb->payments)===2,'over-registration rolls back');
$wpdb=new MemoryDB();$wpdb->fail_insert=true;$_POST=['amount'=>'400000','submission_key'=>'failed_transfer_key_123'];
$r=$h->sn_save_invoice_manual_payment(1,'seller_manual',[]);
check(!$r['success']&&$wpdb->invoice->status==='pending'&&$wpdb->stage->status==='pending','failed payment insert rolls back invoice and stage');
check(end($wpdb->queries)==='ROLLBACK','failed save never commits');
