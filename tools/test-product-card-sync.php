<?php
/** Local SQL integration via a SQLite bridge; no live WordPress data. */
define('ABSPATH', __DIR__); define('ARRAY_A','ARRAY_A'); define('OBJECT','OBJECT');
define('DAY_IN_SECONDS',86400);
error_reporting(E_ALL);
set_error_handler(static function($s,$m,$f,$l){throw new RuntimeException("$m at $f:$l");});
class JsonReply extends Exception { public function __construct(public bool $success,public array $data,public int $status=200){parent::__construct('JSON');} }
class WP_Error {public function __construct(public string $code,public string $message){} }
function absint($v){return abs((int)$v);}
function get_post_type($id){return $GLOBALS['post_type'];}
function current_user_can($cap,...$args){return $GLOBALS['allowed'];}
function get_current_user_id(){return $GLOBALS['user_id'];}
function check_ajax_referer($action,$key){if(($_POST[$key]??'')!=='valid')throw new JsonReply(false,['message'=>'nonce'],403);}
function wp_send_json_error($data,$status=200){throw new JsonReply(false,$data,$status);}
function wp_send_json_success($data){throw new JsonReply(true,$data);}
function wp_unslash($v){return $v;}
function wp_json_encode($v,$flags=0){return json_encode($v,$flags);}
function get_post_meta($id,$key,$single=true){return $key==='_sn_product_credit_amount'?$GLOBALS['credit']:'';}
function get_option($key,$default=false){return $default;}
class SN_Helpers {public static function site_mysql_from_timestamp($v){return date('Y-m-d H:i:s',$v);} }
function set_transient($key,$v,$ttl){$GLOBALS['jobs'][$key]=$v;return true;}
function get_transient($key){return $GLOBALS['jobs'][$key]??false;}
function wp_generate_uuid4(){return sprintf('%032x',++$GLOBALS['uuid']);}
function sanitize_key($v){return preg_replace('/[^a-z0-9_-]/','',strtolower($v));}
function sanitize_text_field($v){return trim(strip_tags($v));}
function current_time($type){return '2026-10-07 12:00:00';}
function get_the_title($id){return 'Destination';}
function esc_attr($v){return htmlspecialchars((string)$v,ENT_QUOTES);}
function esc_html($v){return esc_attr($v);}
function esc_url($v){return esc_attr($v);}
function wp_create_nonce($v){return 'valid';}
function admin_url($v){return 'https://crm.example/wp-admin/'.$v;}
class SN_Projects {
    public static function instance(){return new self;}
    public function product_upgrade_rules($id){return $GLOBALS['rules'];}
}
class SqlDb {
    public string $prefix='wp_', $last_error='';
    public array $queries=[],$pipes=[];
    public $process;
    public bool $lock=true,$fail_update=false,$fail_audit=false,$index=true;
    public string $engine='InnoDB';
    public int $insert_id=0;
    public bool $seed_race=false,$normal_race=false;
    public function __construct(){
        $python= <<<'PY'
import sqlite3,json,sys
db=sqlite3.connect(':memory:',isolation_level=None)
db.row_factory=sqlite3.Row
for line in sys.stdin:
    try:
        sql=json.loads(line)['sql']
        if sql == 'START TRANSACTION': sql='BEGIN'
        sql=sql.replace(' FOR UPDATE','').replace('INSERT IGNORE','INSERT OR IGNORE')
        c=db.execute(sql)
        print(json.dumps({'rows':[dict(r) for r in c.fetchall()] if c.description else [],'affected':max(c.rowcount,0),'insert_id':c.lastrowid or 0}),flush=True)
    except Exception as e: print(json.dumps({'error':str(e)}),flush=True)
PY;
        $this->process=proc_open(['python3','-u','-c',$python],[0=>['pipe','r'],1=>['pipe','w'],2=>['pipe','w']],$this->pipes);
        if(!is_resource($this->process))throw new RuntimeException('No SQL bridge');
        $this->query('CREATE TABLE wp_sn_project_memberships(id INTEGER PRIMARY KEY,status TEXT,contents_snapshot_json TEXT,source_sales_manager_user_id INTEGER DEFAULT 0,updated_at TEXT)');
        $this->query('CREATE TABLE wp_sn_project_membership_items(id INTEGER PRIMARY KEY,membership_id INTEGER,content_product_id INTEGER,base_credit_snapshot REAL,upgrade_options_snapshot_json TEXT,workflow_status TEXT,created_at TEXT,updated_at TEXT)');
        $this->query('CREATE TABLE wp_sn_project_operations(id INTEGER PRIMARY KEY,membership_item_id INTEGER UNIQUE,base_credit REAL,current_credit REAL,activation_mode TEXT,payment_state TEXT,upgrade_invoice_id INTEGER DEFAULT 0,upgrade_action_id INTEGER DEFAULT 0,stage TEXT,customer_choice TEXT DEFAULT \'\',customer_choice_at TEXT DEFAULT \'\',follow_up_at TEXT,routed_at TEXT,decision_due_at TEXT,auto_route_eligible INTEGER,customer_self_activation_allowed INTEGER,source_sales_manager_user_id INTEGER,operations_sales_manager_user_id INTEGER,sales_supervisor_user_id INTEGER,routing_source TEXT,created_at TEXT,updated_at TEXT)');
        $this->query('CREATE TABLE wp_sn_project_actions(id INTEGER PRIMARY KEY,membership_item_id INTEGER,status TEXT,amount REAL)');
        $this->query('CREATE TABLE wp_sn_operations_execution_cases(id INTEGER PRIMARY KEY,membership_item_id INTEGER,status TEXT)');
        $this->query('CREATE TABLE wp_sn_operations_wallet_codes(id INTEGER PRIMARY KEY,membership_item_id INTEGER,activation_code TEXT)');
        $this->query('CREATE TABLE wp_sn_project_events(id INTEGER PRIMARY KEY,membership_id INTEGER,membership_item_id INTEGER,actor_user_id INTEGER,event_key TEXT,details TEXT,created_at TEXT)');
    }
    public function __destruct(){foreach($this->pipes as $pipe)fclose($pipe);proc_close($this->process);}
    public function prepare($sql,...$args){foreach($args as $a)$sql=preg_replace_callback('/%[dfs]/',fn($m)=>$m[0]==='%s'?"'".str_replace("'","''",$a)."'":(string)(float)$a,$sql,1);return $sql;}
    public function run($sql){
        $this->queries[]=$sql;$this->last_error='';
        if(str_starts_with($sql,'ALTER TABLE')){$this->index=true;return ['rows'=>[],'affected'=>1,'insert_id'=>0];}
        fwrite($this->pipes[0],json_encode(['sql'=>$sql])."\n");
        $r=json_decode(fgets($this->pipes[1]),true);
        if(isset($r['error'])){$this->last_error=$r['error'];return false;}
        $this->insert_id=$r['insert_id'];return $r;
    }
    public function query($sql){$r=$this->run($sql);return $r===false?false:$r['affected'];}
    public function get_results($sql,$format=OBJECT){$r=$this->run($sql);if($r===false)return null;return $format===ARRAY_A?$r['rows']:array_map(fn($r)=>(object)$r,$r['rows']);}
    public function get_row($sql,$format=OBJECT){
        if(str_starts_with($sql,'SHOW TABLE STATUS'))return ['Engine'=>$this->engine];
        $rows=$this->get_results($sql,$format);
        if($this->seed_race&&str_starts_with($sql,'SELECT mi.id,mi.content_product_id,COALESCE')){
            $this->seed_race=false;$this->update('wp_sn_project_membership_items',['base_credit_snapshot'=>6000],['id'=>180]);
        }
        if($this->normal_race&&str_starts_with($sql,'SELECT * FROM wp_sn_project_operations')){
            $this->normal_race=false;$this->update('wp_sn_project_operations',['base_credit'=>500000,'current_credit'=>500000],['membership_item_id'=>182]);
        }
        return $rows[0]??null;
    }
    public function get_var($sql){
        $this->queries[]=$sql;
        if(str_contains($sql,'GET_LOCK'))return $this->lock?1:0;
        if(str_contains($sql,'RELEASE_LOCK'))return 1;
        if(str_starts_with($sql,'SHOW INDEX'))return $this->index?1:null;
        if(str_starts_with($sql,'SHOW TABLES LIKE')){preg_match("/LIKE '([^']+)'/",$sql,$m);return in_array($m[1],['wp_sn_project_membership_items','wp_sn_project_operations','wp_sn_project_events'],true)?$m[1]:null;}
        $rows=$this->get_results($sql,ARRAY_A);return $rows?array_values($rows[0])[0]:null;
    }
    public function value($v){return $v===null?'NULL':(is_numeric($v)?(string)$v:"'".str_replace("'","''",$v)."'");}
    public function insert($table,$data){
        if($this->fail_audit&&$table==='wp_sn_project_events'){$this->last_error='audit failure';return false;}
        return $this->query('INSERT INTO '.$table.' ('.implode(',',array_keys($data)).') VALUES ('.implode(',',array_map([$this,'value'],array_values($data))).')');
    }
    public function update($table,$data,$where){
        if($this->fail_update&&$table==='wp_sn_project_operations'){$this->last_error='write failure';return false;}
        $parts=[];foreach($data as $k=>$v)$parts[]=$k.'='.$this->value($v);
        $pred=[];foreach($where as $k=>$v)$pred[]=$k.'='.$this->value($v);
        return $this->query('UPDATE '.$table.' SET '.implode(',',$parts).' WHERE '.implode(' AND ',$pred));
    }
}
$GLOBALS['jobs']=[];$GLOBALS['uuid']=0;$GLOBALS['allowed']=true;$GLOBALS['post_type']='product';$GLOBALS['user_id']=9;$GLOBALS['credit']=9000000;
$GLOBALS['rules']=[['product_id'=>202,'name'=>'Destination','target_credit'=>12000000,'upgrade_amount'=>300000]];
$wpdb=new SqlDb;
require __DIR__.'/../includes/class-sn-product-card-sync.php';
require __DIR__.'/../includes/class-sn-operations-flow.php';
$sync=SN_Product_Card_Sync::instance();$checks=0;
function check($v,$m){global $checks;if(!$v)throw new RuntimeException('FAIL: '.$m);$checks++;}
function call_ajax($method,$post){global $sync;$_POST=$post+['product_id'=>101,'nonce'=>'valid'];try{$sync->$method();}catch(JsonReply $r){return $r;}throw new RuntimeException('Missing JSON');}
function preview($fields=['credit'],$scope='active'){return call_ajax('ajax_preview',['fields'=>$fields,'scope'=>$scope]);}
function batch($token,$cursor=0){return call_ajax('ajax_apply',['token'=>$token,'cursor'=>$cursor]);}
function row($table,$id){global $wpdb;return $wpdb->get_row('SELECT * FROM wp_'.$table.' WHERE id='.$id,ARRAY_A);}
function card($id,$credit=100000,$stage='awaiting_customer',$status='active',$product=101,$member=null){
    global $wpdb;$member=$member??$id;
    if(!row('sn_project_memberships',$member))$wpdb->insert('wp_sn_project_memberships',['id'=>$member,'status'=>$status,'contents_snapshot_json'=>json_encode([['content_product_id'=>$product,'base_credit'=>$credit,'upgrade_options'=>[],'route'=>['manager'=>44]],['content_product_id'=>999,'base_credit'=>77]])]);
    $wpdb->insert('wp_sn_project_membership_items',['id'=>$id,'membership_id'=>$member,'content_product_id'=>$product,'base_credit_snapshot'=>$credit,'upgrade_options_snapshot_json'=>'[]','workflow_status'=>'waiting']);
    if($stage!=='missing')$wpdb->insert('wp_sn_project_operations',['id'=>$id,'membership_item_id'=>$id,'base_credit'=>$credit,'current_credit'=>$credit,'activation_mode'=>'normal','payment_state'=>'','upgrade_invoice_id'=>0,'upgrade_action_id'=>0,'stage'=>$stage]);
}
card(1);card(2,100000,'sales_expert');card(3,100000,'missing');card(4,9000000);
card(5,100000,'awaiting_customer','suspended');card(6,100000,'completed');card(7);
$wpdb->update('wp_sn_project_operations',['activation_mode'=>'upsell','current_credit'=>15000000,'upgrade_invoice_id'=>123],['id'=>7]);
card(8);$wpdb->insert('wp_sn_project_actions',['id'=>8,'membership_item_id'=>8,'status'=>'awaiting_payment','amount'=>12345]);
card(9);$wpdb->insert('wp_sn_operations_execution_cases',['id'=>9,'membership_item_id'=>9,'status'=>'manager_queue']);
card(10);$wpdb->insert('wp_sn_operations_wallet_codes',['id'=>10,'membership_item_id'=>10,'activation_code'=>'SECRET']);
card(11);$wpdb->update('wp_sn_project_operations',['current_credit'=>999],['id'=>11]);
card(12);$wpdb->update('wp_sn_project_membership_items',['workflow_status'=>'purchased'],['id'=>12]);
card(13,100000,'awaiting_customer','active',303);
$p=preview();check($p->success,'preview succeeds');check($p->data['total']===11,'only this product + active memberships');
check(row('sn_project_membership_items',1)['base_credit_snapshot']==100000,'preview is read only');
$counts=array_column($p->data['counts'],'total','reason');check($counts['eligible']==4,'eligibility count');check($counts['execution']==3,'execution + code protected');
$before_actions=$wpdb->get_results('SELECT * FROM wp_sn_project_actions',ARRAY_A);
$r=batch($p->data['token']);check($r->success&&$r->data['done'],'batch completes');check($r->data['changed']===3&&$r->data['unchanged']===1,'correct changed/unchanged');check($r->data['skipped']===7,'protected cards reported');
foreach([1,2,3,4] as $id){check(row('sn_project_membership_items',$id)['base_credit_snapshot']==9000000,'existing item credit '.$id);if($id!==3){$op=row('sn_project_operations',$id);check($op['base_credit']==9000000&&$op['current_credit']==9000000,'operation credit '.$id);}}
$contents=json_decode(row('sn_project_memberships',1)['contents_snapshot_json'],true);
check($contents[0]['base_credit']==9000000,'parent snapshot updated');check($contents[1]['base_credit']==77&&$contents[0]['route']['manager']===44,'other cards + route preserved');
check(row('sn_project_membership_items',5)['base_credit_snapshot']==100000,'inactive scope preserved');
check(row('sn_project_operations',7)['current_credit']==15000000,'paid/upsell current credit preserved');
check(row('sn_project_membership_items',13)['base_credit_snapshot']==100000,'other product preserved');
check($wpdb->get_results('SELECT * FROM wp_sn_project_actions',ARRAY_A)===$before_actions,'invoices/actions not rewritten');
$events=$wpdb->get_results('SELECT * FROM wp_sn_project_events',ARRAY_A);check(count($events)===3,'audit for changed cards only');
$details=json_decode($events[0]['details'],true);check($details['before']['item_credit']==100000&&$details['after']['credit']==9000000&&$events[0]['actor_user_id']==9,'audit before/after/actor');
$replay=batch($p->data['token']);check($replay->success&&$replay->data===$r->data,'lost response replay same result');check(count($wpdb->get_results('SELECT * FROM wp_sn_project_events',ARRAY_A))===3,'replay no duplicate audits');
$GLOBALS['credit']=7000000;$p2=preview(['upgrade_options'],'all');$r2=batch($p2->data['token']);check($r2->success,'options-only sync');
check(row('sn_project_membership_items',1)['base_credit_snapshot']==9000000,'unselected credit preserved');
check(json_decode(row('sn_project_membership_items',1)['upgrade_options_snapshot_json'],true)===$GLOBALS['rules'],'upgrade options updated');
check(json_decode(row('sn_project_memberships',1)['contents_snapshot_json'],true)[0]['upgrade_options']===$GLOBALS['rules'],'parent options coherent');
check(row('sn_project_membership_items',7)['upgrade_options_snapshot_json']==='[]','upsell financial options preserved');
$p3=preview(['credit'],'all');$r3=batch($p3->data['token']);check($r3->success&&row('sn_project_operations',5)['current_credit']==7000000,'all scope includes suspended');
$p4=preview();$GLOBALS['credit']=6000000;$r4=batch($p4->data['token']);check(!$r4->success&&str_contains($r4->data['message'],'تغییر'),'changed saved product requires preview');
check(row('sn_project_membership_items',1)['base_credit_snapshot']==7000000,'stale preview does not update');
$p5=preview();$wpdb->fail_update=true;$failed=batch($p5->data['token']);check(!$failed->success,'failed operation write rejected');
check(row('sn_project_membership_items',1)['base_credit_snapshot']==7000000&&get_transient('sn_card_sync_'.$p5->data['token'])['cursor']===0,'item rollback + cursor not advanced');
$wpdb->fail_update=false;$wpdb->fail_audit=true;$failed=batch($p5->data['token']);check(!$failed->success&&row('sn_project_operations',1)['current_credit']==7000000,'audit failure rolls back financial writes');
$wpdb->fail_audit=false;$wpdb->lock=false;$failed=batch($p5->data['token']);check(!$failed->success,'overlapping job lock rejected');$wpdb->lock=true;
$GLOBALS['user_id']=10;check(!batch($p5->data['token'])->success,'job tied to creating user');$GLOBALS['user_id']=9;
check(!batch('bad')->success,'invalid token rejected');check(!batch(str_repeat('f',32))->success,'expired job rejected');
check(!batch($p5->data['token'],999)->success,'wrong cursor rejected');
$GLOBALS['allowed']=false;check(!preview()->success,'permission required');$GLOBALS['allowed']=true;
check(!call_ajax('ajax_preview',['nonce'=>'wrong','fields'=>['credit']])->success,'nonce required');
$GLOBALS['post_type']='page';check(!preview()->success,'product type required');$GLOBALS['post_type']='product';
check(!preview([])->success,'empty fields rejected');
$wpdb->engine='MyISAM';check(!preview()->success,'nontransactional table rejected');$wpdb->engine='InnoDB';
// 60 cards: keyset cursor + stable cutoff, no giant request and no OFFSET.
for($id=100;$id<160;$id++)card($id);
$p6=preview(['credit','upgrade_options']);card(160);
$cursor=0;$requests=0;do{$b=batch($p6->data['token'],$cursor);check($b->success,'multi batch '.$requests);$cursor=$b->data['cursor'];$requests++;}while(!$b->data['done']);
check($requests===3,'25-card bounded batches');check($b->data['changed']===64,'existing eligible cards updated across batches');
check(row('sn_project_membership_items',159)['base_credit_snapshot']==6000000,'last old card updated');check(row('sn_project_membership_items',160)['base_credit_snapshot']==100000,'new card outside preview cutoff');
check(count(array_filter($wpdb->queries,fn($q)=>str_contains($q,' LIMIT 25')))>=3,'bounded SQL scans');
check(count(array_filter($wpdb->queries,fn($q)=>str_contains($q,'OFFSET')))==0,'no offset scans');
card(170);$wpdb->update('wp_sn_project_memberships',['contents_snapshot_json'=>'{broken'],['id'=>170]);
$p7=preview();$b=batch($p7->data['token']);while($b->success&&!$b->data['done'])$b=batch($p7->data['token'],$b->data['cursor']);
check($b->success&&($b->data['reasons']['invalid_snapshot']??0)===1,'invalid JSON reported without destructive rewrite');
check(row('sn_project_memberships',170)['contents_snapshot_json']==='{broken'&&row('sn_project_membership_items',170)['base_credit_snapshot']==100000,'corrupt snapshot + credit intact');
// Revalidation is exercised on the actual Operations Flow helper after lock.
$ctx=(object)row('sn_project_membership_items',1);$option=['target_product_id'=>202,'name'=>'Destination','credit'=>12000000.0,'amount'=>300000.0];
$guard=new ReflectionMethod(SN_Operations_Flow::class,'upgrade_snapshot_unchanged');
check($guard->invoke(SN_Operations_Flow::instance(),1,$ctx,$option),'unchanged upgrade accepted');
$old=clone $ctx;$wpdb->update('wp_sn_project_membership_items',['base_credit_snapshot'=>1],['id'=>1]);
check(!$guard->invoke(SN_Operations_Flow::instance(),1,$old,$option),'credit changed while customer choosing upgrade rejected');
$wpdb->update('wp_sn_project_membership_items',['base_credit_snapshot'=>$old->base_credit_snapshot,'upgrade_options_snapshot_json'=>'[]'],['id'=>1]);
check(!$guard->invoke(SN_Operations_Flow::instance(),1,$old,$option),'removed upgrade rejected after lock');
card(180,5000,'missing');$wpdb->seed_race=true;
$ensure=new ReflectionMethod(SN_Operations_Flow::class,'ensure_operation');
$created=$ensure->invoke(SN_Operations_Flow::instance(),180,false);
check($created!==null&&$created->base_credit==6000&&$created->current_credit==6000,'operation seeded from current item credit after intervening sync');
check($created->auto_route_eligible===0,'lazy legacy operation retains inactivity protection');
card(181,8000,'missing');SN_Operations_Flow::instance()->seed_membership_cards(181);
$seeded=$wpdb->get_row('SELECT * FROM wp_sn_project_operations WHERE membership_item_id=181');
check($seeded->base_credit==8000&&$seeded->current_credit==8000&&$seeded->auto_route_eligible===1,'new subscription seed credit + inactivity flag');
card(182,100000);$wpdb->normal_race=true;
$route=new ReflectionMethod(SN_Operations_Flow::class,'route_normal_to_executive');
check($route->invoke(SN_Operations_Flow::instance(),182,9,'self'),'normal route after concurrent sync');
check(row('sn_project_operations',182)['current_credit']==500000,'normal route uses current database base credit');
$event=$wpdb->get_row("SELECT * FROM wp_sn_project_events WHERE membership_item_id=182 ORDER BY id DESC LIMIT 1",ARRAY_A);
check(json_decode($event['details'],true)['credit']==500000,'normal route audit uses committed credit');
$GLOBALS['allowed']=true;ob_start();$sync->render_controls(101);$html=ob_get_clean();
check(str_contains($html,'data-sync-preview')&&str_contains($html,'data-sync-apply disabled'),'preview before apply UI');
check(str_contains($html,'value="credit" checked')&&str_contains($html,'value="upgrade_options"'),'selective fields UI');
if(isset($argv[1])){if(!is_dir($argv[1]))mkdir($argv[1],0777,true);file_put_contents($argv[1].'/product-card-sync.html',$html);}
echo "PASS $checks product card sync checks (actual SQL predicates, transaction rollback, replay, cutoff, permissions, upgrade races).\n";
