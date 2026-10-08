<?php
if (!defined('ABSPATH')) { exit; }
final class SN_MIS_Report_Store {
    public const VERSION='1';
    public const METRICS=['manager_assigned','delivered','returned','transferred','pre_case','linked','approved','completed','reason_review','issue_count'];
    public $db;
    public function __construct() { global $wpdb; $this->db=$wpdb; }
    public function table(string $name): string { return $this->db->prefix.'sn_mis_report_'.$name; }
    public function checked($result) { if ($result===false || $this->db->last_error) { $id=substr(wp_generate_uuid4(),0,8); error_log('SN MIS report write '.$id.' '.$this->db->last_error); throw new RuntimeException('خطای ذخیره گزارش؛ کد '.$id); } return $result; }
    public function install(): void {
        if (!current_user_can('manage_options') && !(defined('WP_CLI') && WP_CLI)) { throw new RuntimeException('نصب ساختار گزارش فقط برای مدیر سایت مجاز است.'); }
        if (get_option('sn_mis_report_schema')===self::VERSION) { return; }
        $lock='sn_mis_report_schema_'.substr(hash('sha256',$this->db->prefix),0,16);
        if ((int)$this->db->get_var($this->db->prepare('SELECT GET_LOCK(%s,0)',$lock))!==1) { throw new RuntimeException('مهاجرت دیگری در حال اجراست؛ دوباره تلاش کنید.'); }
        try {
            // Another installer may have completed between the first check and lock acquisition.
            // Read the database directly because the options cache can predate that installation.
            $installed=$this->db->get_var($this->db->prepare("SELECT option_value FROM {$this->db->options} WHERE option_name=%s",'sn_mis_report_schema')); $this->checked(true);
            if ($installed===self::VERSION) { return; }
            $charset=$this->db->get_charset_collate(); $j=$this->table('jobs'); $c=$this->table('cases'); $i=$this->table('invoices'); $l=$this->table('links'); $d=$this->table('deliveries');
            $this->checked($this->db->query("CREATE TABLE IF NOT EXISTS {$j} (id varchar(36) NOT NULL,user_id bigint unsigned NOT NULL,scope_hash char(64) NOT NULL,filters longtext NOT NULL,cursor_id bigint unsigned NOT NULL DEFAULT 0,max_id bigint unsigned NOT NULL DEFAULT 0,scanned bigint unsigned NOT NULL DEFAULT 0,status varchar(20) NOT NULL,created_at datetime NOT NULL,finished_at datetime NULL,expires_at datetime NOT NULL,export_cursor bigint unsigned NOT NULL DEFAULT 0,export_status varchar(20) NOT NULL DEFAULT '',export_dir varchar(500) NULL,export_offsets longtext NULL,PRIMARY KEY(id),KEY user_status(user_id,status),KEY expiry(expires_at)) ENGINE=InnoDB {$charset}"));
            $metrics=implode(',',array_map(static fn($k)=>"{$k} int NOT NULL DEFAULT 0",self::METRICS));
            $roles=implode(',',array_map(static fn($k)=>"{$k} bigint unsigned NOT NULL DEFAULT 0",array_keys(SN_MIS_Report_Model::ROLES)));
            $this->checked($this->db->query("CREATE TABLE IF NOT EXISTS {$c} (job_id varchar(36) NOT NULL,row_id bigint unsigned NOT NULL,batch_id bigint unsigned NOT NULL,campaign varchar(120) NOT NULL,category varchar(120) NOT NULL,validity varchar(100) NOT NULL,flow varchar(100) NOT NULL,distribution varchar(100) NOT NULL,reason_quality varchar(30) NOT NULL,{$metrics},{$roles},paid decimal(20,2) NOT NULL DEFAULT 0,remaining decimal(20,2) NOT NULL DEFAULT 0,payload longtext NOT NULL,PRIMARY KEY(job_id,row_id),KEY batch(job_id,batch_id),KEY campaign(job_id,campaign),KEY seller(job_id,seller),KEY review(job_id,reason_review)) ENGINE=InnoDB {$charset}"));
            $this->checked($this->db->query("CREATE TABLE IF NOT EXISTS {$i} (job_id varchar(36) NOT NULL,invoice_id bigint unsigned NOT NULL,status varchar(60) NOT NULL,workflow varchar(60) NOT NULL,approved tinyint NOT NULL,completed tinyint NOT NULL,partial tinyint NOT NULL,total decimal(20,2) NOT NULL,paid decimal(20,2) NOT NULL,remaining decimal(20,2) NOT NULL,payload longtext NOT NULL,PRIMARY KEY(job_id,invoice_id)) ENGINE=InnoDB {$charset}"));
            $this->checked($this->db->query("CREATE TABLE IF NOT EXISTS {$l} (job_id varchar(36) NOT NULL,row_id bigint unsigned NOT NULL,invoice_id bigint unsigned NOT NULL,PRIMARY KEY(job_id,row_id,invoice_id),KEY invoice(job_id,invoice_id)) ENGINE=InnoDB {$charset}"));
            $this->checked($this->db->query("CREATE TABLE IF NOT EXISTS {$d} (job_id varchar(36) NOT NULL,row_id bigint unsigned NOT NULL,person_id bigint unsigned NOT NULL,position varchar(100) NOT NULL,event_count int unsigned NOT NULL,first_at datetime NULL,last_at datetime NULL,PRIMARY KEY(job_id,row_id,person_id,position),KEY person(job_id,person_id)) ENGINE=InnoDB {$charset}"));
            $this->indexes();
            update_option('sn_mis_report_schema',self::VERSION,false);
        } finally { $this->db->get_var($this->db->prepare('SELECT RELEASE_LOCK(%s)',$lock)); }
    }
    /** Explicit install only. Never called by rendering, polling, exports or ordinary init. */
    private function indexes(): void {
        $required=[
            'sn_mis_data_rows'=>['sn_mr_batch_id'=>['batch_id','id']],
            'sn_mis_lead_pool'=>['sn_mr_source_row'=>['source_row_id']],
            'sn_distribution_items'=>['sn_mr_source'=>['source_type','source_id']],
            'sn_distribution_item_logs'=>['sn_mr_item_id'=>['item_id','id']],
            'sn_seller_flow_states'=>['sn_mr_source'=>['source_kind','source_id']],
            'sn_seller_flow_events'=>['sn_mr_source'=>['source_kind','source_id']],
            'sn_invoices'=>['sn_mr_lead_id'=>['lead_id','id'],'sn_mr_referral'=>['referral_item_id']],
            'sn_invoice_payment_stages'=>['sn_mr_invoice'=>['invoice_id']],
        ];
        $old=(int)$this->db->get_var('SELECT @@SESSION.lock_wait_timeout'); $this->checked(true);
        $this->checked($this->db->query('SET SESSION lock_wait_timeout=5'));
        try {
            foreach ($required as $name=>$indexes) {
                $table=$this->db->prefix.$name; $found=[];
                $rows=$this->db->get_results("SHOW INDEX FROM {$table}",ARRAY_A); $this->checked(true);
                foreach ($rows as $row) { $found[$row['Key_name']][(int)$row['Seq_in_index']]=$row['Column_name']; }
                foreach ($indexes as $index=>$columns) {
                    $covered=false;
                    foreach ($found as $parts) { ksort($parts); if (array_slice(array_values($parts),0,count($columns))===$columns) { $covered=true; break; } }
                    if ($covered) { continue; }
                    // Refuse blocking COPY fallback on engines without online index creation.
                    $this->checked($this->db->query("ALTER TABLE {$table} ADD INDEX {$index} (".implode(',',$columns)."), ALGORITHM=INPLACE, LOCK=NONE"));
                }
            }
        } finally { $this->db->query('SET SESSION lock_wait_timeout='.max(1,$old)); }
    }
    public function start(SN_MIS_Report_Source $src,array $f): array {
        if (get_option('sn_mis_report_schema')!==self::VERSION) { throw new RuntimeException('ساختار گزارش نصب نشده است؛ مدیر سایت باید «نصب ساختار گزارش» را اجرا کند.'); }
        $src->validate_filters($f);
        $count=(int)$this->db->get_var($this->db->prepare('SELECT COUNT(*) FROM '.$this->table('jobs')." WHERE user_id=%d AND status IN ('building','ready') AND expires_at>%s",get_current_user_id(),(new DateTimeImmutable('now',new DateTimeZone('UTC')))->format('Y-m-d H:i:s')));
        if ($count>=3) { throw new RuntimeException('سه گزارش فعال دارید؛ گزارش قبلی را لغو کنید یا تا انقضای آن صبر کنید.'); }
        $id=wp_generate_uuid4(); $max=(int)$this->db->get_var('SELECT MAX(id) FROM '.$this->db->prefix.'sn_mis_data_rows'); $this->checked(true);
        $j=['id'=>$id,'user_id'=>get_current_user_id(),'scope_hash'=>$src->scope_hash,'filters'=>wp_json_encode($f),'max_id'=>$max,'status'=>'building','created_at'=>(new DateTimeImmutable('now',new DateTimeZone('UTC')))->format('Y-m-d H:i:s'),'expires_at'=>(new DateTimeImmutable('now',new DateTimeZone('UTC')))->modify('+1 hour')->format('Y-m-d H:i:s')];
        $this->checked($this->db->insert($this->table('jobs'),$j)); return $this->job($id,$src);
    }
    public function job(string $id,SN_MIS_Report_Source $src): array {
        if (!preg_match('/^[a-f0-9-]{36}$/D',$id)) { throw new RuntimeException('شناسه گزارش نامعتبر است.'); }
        $j=$this->db->get_row($this->db->prepare('SELECT * FROM '.$this->table('jobs').' WHERE id=%s AND user_id=%d',$id,get_current_user_id()),ARRAY_A); $this->checked(true);
        if (!$j || $j['expires_at']<(new DateTimeImmutable('now',new DateTimeZone('UTC')))->format('Y-m-d H:i:s') || !hash_equals($j['scope_hash'],$src->scope_hash)) { throw new RuntimeException('گزارش منقضی شده یا محدوده دسترسی تغییر کرده است؛ گزارش تازه بسازید.'); }
        // Recheck custody on EVERY read/download: moving a case immediately revokes old snapshots.
        if ($src->allowed!==null) {
            $p=$this->db->prefix;
            $bad=$this->db->get_var($this->db->prepare('SELECT c.row_id FROM '.$this->table('cases')." c LEFT JOIN {$p}sn_mis_data_rows r ON r.id=c.row_id WHERE c.job_id=%s AND (r.id IS NULL OR NOT (".$src->scope_sql().')) LIMIT 1',$id)); $this->checked(true);
            if ($bad) { throw new RuntimeException('مالکیت پرونده‌ها تغییر کرده است؛ گزارش تازه بسازید.'); }
            $ids=SN_MIS_Report_Source::ids($src->allowed);
            $bad=$this->db->get_var($this->db->prepare('SELECT x.invoice_id FROM '.$this->table('invoices')." x LEFT JOIN {$p}sn_invoices i ON i.id=x.invoice_id WHERE x.job_id=%s AND (i.id IS NULL OR i.seller_id NOT IN ({$ids})) LIMIT 1",$id)); $this->checked(true);
            if ($bad) { throw new RuntimeException('دسترسی فاکتور تغییر کرده است؛ گزارش تازه بسازید.'); }
        }
        return $j;
    }
    public function lock(string $id): void {
        if ((int)$this->db->get_var($this->db->prepare('SELECT GET_LOCK(%s,0)','sn_mis_'.substr(hash('sha256',$this->db->prefix.$id),0,40)))!==1) { throw new RuntimeException('درخواست قبلی این گزارش هنوز در حال اجراست.'); }
    }
    public function unlock(string $id): void { $this->db->get_var($this->db->prepare('SELECT RELEASE_LOCK(%s)','sn_mis_'.substr(hash('sha256',$this->db->prefix.$id),0,40))); }
    public function step(string $id,SN_MIS_Report_Source $src): array {
        $this->lock($id);
        try {
            $j=$this->job($id,$src); if ($j['status']!=='building') { return $this->progress($j); }
            $f=json_decode($j['filters'],true); $rows=$src->rows((int)$j['cursor_id'],(int)$j['max_id'],$f);
            $cases=$src->hydrate($rows);
            $this->checked($this->db->query('START TRANSACTION'));
            try {
                foreach ($cases as $c) { if (SN_MIS_Report_Model::matches($c,$f)) { $this->save_case($id,$c); } }
                $cursor=$rows ? (int)end($rows)['id'] : (int)$j['max_id']; $done=count($rows)<100 || $cursor >= (int)$j['max_id'];
                $update=['cursor_id'=>$cursor,'scanned'=>(int)$j['scanned']+count($rows),'status'=>$done?'ready':'building'];
                if ($done) { $update['finished_at']=(new DateTimeImmutable('now',new DateTimeZone('UTC')))->format('Y-m-d H:i:s'); }
                $this->checked($this->db->update($this->table('jobs'),$update,['id'=>$id])); $this->checked($this->db->query('COMMIT'));
            } catch (Throwable $e) { $this->db->query('ROLLBACK'); throw $e; }
            return $this->progress(array_merge($j,$update));
        } finally { $this->unlock($id); }
    }
    private function save_case(string $job,array $c): void {
        // Invoice payload is frozen once per job, ensuring detail and XLSX use the same figures.
        foreach ($c['invoices'] as &$i) {
            $stored=$this->db->get_var($this->db->prepare('SELECT payload FROM '.$this->table('invoices').' WHERE job_id=%s AND invoice_id=%d',$job,$i['id'])); $this->checked(true);
            if ($stored) { $proof=$i['link_source']; $i=json_decode($stored,true); $i['link_source']=$proof; }
            else {
                $this->checked($this->db->insert($this->table('invoices'),['job_id'=>$job,'invoice_id'=>$i['id'],'status'=>$i['status'],'workflow'=>$i['payment_workflow_status']??'','approved'=>$i['approved'],'completed'=>$i['completed'],'partial'=>$i['partial'],'total'=>$i['total'],'paid'=>$i['paid'],'remaining'=>$i['remaining'],'payload'=>wp_json_encode($i)]));
            }
            $this->checked($this->db->replace($this->table('links'),['job_id'=>$job,'row_id'=>$c['row_id'],'invoice_id'=>$i['id']]));
        } unset($i);
        $c['paid']=array_sum(array_column($c['invoices'],'paid')); $c['remaining']=array_sum(array_column($c['invoices'],'remaining'));
        $c['approved']=(int)(array_sum(array_column($c['invoices'],'approved'))>0); $c['completed']=(int)(array_sum(array_column($c['invoices'],'completed'))>0);
        $recipients=[];
        foreach ($c['history'] as $log) {
            if (!(int)$log['to_user_id'] || (int)$log['to_user_id']===(int)$log['from_user_id']) { continue; }
            $key=$log['to_user_id'].':'.$log['to_position'];
            if (!isset($recipients[$key])) { $recipients[$key]=['job_id'=>$job,'row_id'=>$c['row_id'],'person_id'=>$log['to_user_id'],'position'=>$log['to_position'],'event_count'=>0,'first_at'=>$log['created_at'],'last_at'=>$log['created_at']]; }
            $recipients[$key]['event_count']++;
            if ($log['created_at']<$recipients[$key]['first_at']) { $recipients[$key]['first_at']=$log['created_at']; }
            if ($log['created_at']>$recipients[$key]['last_at']) { $recipients[$key]['last_at']=$log['created_at']; }
        }
        foreach ($recipients as $recipient) { $this->checked($this->db->replace($this->table('deliveries'),$recipient)); }
        $keys=array_merge(['row_id','batch_id','campaign','category','validity','flow','distribution','reason_quality','paid','remaining'],self::METRICS,array_keys(SN_MIS_Report_Model::ROLES));
        $record=array_intersect_key($c,array_flip($keys)); $record['job_id']=$job; $record['payload']=wp_json_encode($c);
        $this->checked($this->db->replace($this->table('cases'),$record));
    }
    public function progress(array $j): array { return ['job'=>$j['id'],'status'=>$j['status'],'scanned'=>(int)$j['scanned'],'created_at'=>$j['created_at'],'finished_at'=>$j['finished_at']??null]; }
    public function summary(string $id,SN_MIS_Report_Source $src): array {
        $j=$this->job($id,$src); if ($j['status']!=='ready') { throw new RuntimeException('گزارش هنوز کامل نشده است.'); }
        $table=$this->table('cases'); $q=$this->db->prepare('job_id=%s',$id);
        $sum='COUNT(*) total,'.implode(',',array_map(static fn($k)=>"COALESCE(SUM({$k}),0) {$k}",self::METRICS));
        $totals=$src->query("SELECT {$sum},COUNT(DISTINCT NULLIF(seller,0)) seller_count FROM {$table} WHERE {$q}")[0];
        $break=[];
        foreach (['validity','flow','distribution','reason_quality'] as $key) { $break[$key]=$src->query("SELECT {$key} label,COUNT(*) total FROM {$table} WHERE {$q} GROUP BY {$key}"); }
        $groups=$src->query("SELECT campaign,batch_id,category,{$sum},SUM(flow='no_answer') no_answer,SUM(flow='callback') callback,SUM(flow='not_purchased') not_purchased,SUM(flow='pre_invoice') pre_invoice,SUM(flow='unrecorded') unrecorded FROM {$table} WHERE {$q} GROUP BY campaign,batch_id,category ORDER BY campaign,batch_id");
        $people=[];
        foreach (SN_MIS_Report_Model::ROLES as $role=>$label) {
            foreach ($src->query("SELECT {$role} person_id,{$sum},SUM(flow='no_answer') no_answer,SUM(flow='callback') callback,SUM(flow='not_purchased') not_purchased,SUM(flow='pre_invoice') pre_invoice,SUM(flow='unrecorded') unrecorded FROM {$table} WHERE {$q} AND {$role}>0 GROUP BY {$role}") as $r) { $r['role']=$label; $r['name']=$src->name((int)$r['person_id']); $people[]=$r; }
        }
        $historical=$src->query('SELECT person_id,position,COUNT(*) unique_cases,SUM(event_count) events,MIN(first_at) first_at,MAX(last_at) last_at FROM '.$this->table('deliveries')." WHERE {$q} GROUP BY person_id,position ORDER BY person_id");
        foreach ($historical as &$h) { $h['name']=$src->name((int)$h['person_id']); $h['role']=SN_MIS_Report_Model::ROLES[$h['position']]??$h['position']; } unset($h);
        $it=$this->table('invoices');
        $invoice_groups=$src->query("SELECT status,workflow,COUNT(*) total_count,SUM(total) amount,SUM(paid) paid,SUM(remaining) remaining,SUM(approved) approved,SUM(completed) completed,SUM(partial) partial FROM {$it} WHERE {$q} GROUP BY status,workflow");
        $invoice_totals=$src->query("SELECT COUNT(*) total_count,COALESCE(SUM(total),0) amount,COALESCE(SUM(paid),0) paid,COALESCE(SUM(remaining),0) remaining,COALESCE(SUM(approved),0) approved,COALESCE(SUM(completed),0) completed,COALESCE(SUM(CASE WHEN completed=1 THEN total ELSE 0 END),0) completed_amount FROM {$it} WHERE {$q}")[0];
        $lt=$this->table('links'); $shared=$src->query("SELECT invoice_id,COUNT(*) case_count FROM {$lt} WHERE {$q} GROUP BY invoice_id HAVING COUNT(*)>1");
        return ['progress'=>$this->progress($j),'filters'=>json_decode($j['filters'],true),'totals'=>$totals,'breakdown'=>$break,'groups'=>$groups,'people'=>$people,'historical_people'=>$historical,'invoice_groups'=>$invoice_groups,'invoice_totals'=>$invoice_totals,'shared_invoices'=>$shared,'notes'=>['شمارش پرونده یکتا بر اساس sn_mis_data_rows.id؛ آمار تحویل و انتقال از کل تاریخچه پرونده منتخب است.','فیلتر زمان: پرونده دارای حداقل یک رویداد در بازه؛ اعداد وضعیت، آخرین وضعیت هنگام گردآوری‌اند، نه وضعیت تاریخی در انتهای بازه.','تفکیک عملکرد افراد بر اساس فروشنده نهایی و سلسله‌مراتب جاری HR؛ تاریخچه گیرندگان جدا در جزئیات و اکسل است.','فیلترها پرونده را انتخاب می‌کنند؛ آمار وضعیت و مبالغ شامل همه فاکتورهای قطعی همان پرونده‌هاست. زمان و وضعیت انتخابی فاکتور باید در یک فاکتور منطبق باشند.',
            'فاکتور مشترک بین چند پرونده در جمع کل فاکتورها و مبالغ فقط یک بار شمرده می‌شود.','دلیل recorded فقط از کنترل شکلی عبور کرده است؛ صحت معنایی تضمین نشده.','گزارش طی بازه گردآوری ساخته شده؛ برای دیدن تغییرات تازه، گزارش را به‌روزرسانی کنید.']];
    }
    public function page(string $id,SN_MIS_Report_Source $src,int $page,string $search=''): array {
        $j=$this->job($id,$src); if ($j['status']!=='ready') { throw new RuntimeException('ابتدا ساخت گزارش را کامل کنید.'); }
        $where=$this->db->prepare('job_id=%s',$id);
        if ($search!=='') { $where.=$this->db->prepare(' AND payload LIKE %s','%'.$this->db->esc_like($search).'%'); }
        $total=(int)$this->db->get_var('SELECT COUNT(*) FROM '.$this->table('cases').' WHERE '.$where); $this->checked(true);
        $rows=$src->query('SELECT payload FROM '.$this->table('cases').' WHERE '.$where.' ORDER BY row_id LIMIT 50 OFFSET '.(max(0,$page-1)*50));
        return ['rows'=>array_map(static fn($r)=>json_decode($r['payload'],true),$rows),'total'=>$total,'page'=>max(1,$page),'pages'=>max(1,(int)ceil($total/50))];
    }
    public function cancel(string $id,SN_MIS_Report_Source $src): void {
        $this->job($id,$src); $this->lock($id);
        try { $this->checked($this->db->update($this->table('jobs'),['status'=>'cancelled','expires_at'=>'2000-01-01 00:00:00'],['id'=>$id])); }
        finally { $this->unlock($id); }
    }
    private function delete(string $id): void {
        $j=$this->db->get_row($this->db->prepare('SELECT export_dir FROM '.$this->table('jobs').' WHERE id=%s',$id),ARRAY_A);
        if ($j && !empty($j['export_dir']) && class_exists('SN_MIS_Report_XLSX')) { SN_MIS_Report_XLSX::remove($j['export_dir']); }
        $remaining=false;
        foreach (['deliveries','links','invoices','cases'] as $t) {
            $table=$this->table($t);
            $this->checked($this->db->query($this->db->prepare("DELETE FROM {$table} WHERE job_id=%s LIMIT 2000",$id)));
            if ($this->db->get_var($this->db->prepare("SELECT 1 FROM {$table} WHERE job_id=%s LIMIT 1",$id))) { $remaining=true; }
            $this->checked(true);
        }
        if (!$remaining) { $this->checked($this->db->delete($this->table('jobs'),['id'=>$id])); }
    }
    public function cleanup(): void {
        if (get_option('sn_mis_report_schema')!==self::VERSION) { return; }
        $ids=$this->db->get_col("SELECT id FROM ".$this->table('jobs')." WHERE expires_at<'".(new DateTimeImmutable('now',new DateTimeZone('UTC')))->format('Y-m-d H:i:s')."' LIMIT 10");
        foreach ($ids?:[] as $id) { try { $this->lock($id); try { $this->delete($id); } finally { $this->unlock($id); } } catch (Throwable $e) { error_log('SN MIS report cleanup: '.$e->getMessage()); } }
    }
}
