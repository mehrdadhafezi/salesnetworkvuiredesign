<?php
if (!defined('ABSPATH')) { exit; }
final class SN_MIS_Report_Source {
    public $db; public array $people = []; public ?array $allowed; public string $scope_hash;
    private array $columns = [];
    public function __construct() {
        global $wpdb; $this->db = $wpdb;
        if (!is_user_logged_in()) { throw new RuntimeException('ورود به حساب لازم است.'); }
        $svc = new SN_Scope_Service(); $uid = get_current_user_id();
        $position = $svc->get_viewer_position_slug($uid);
        if (!current_user_can('manage_options') && !in_array($position, array_merge(['mis'],array_keys(SN_MIS_Report_Model::ROLES)), true)) { throw new RuntimeException('مجوز مشاهده گزارش MIS ندارید.'); }
        $p = $wpdb->prefix;
        $profiles = $this->query("SELECT h.id,h.user_id,h.is_active,pos.slug,u.display_name FROM {$p}sn_hr_profiles h JOIN {$p}sn_hr_positions pos ON pos.id=h.position_id JOIN {$wpdb->users} u ON u.ID=h.user_id");
        $profile_users = [];
        foreach ($profiles as $h) { $profile_users[$h['id']] = (int)$h['user_id']; $this->people[(int)$h['user_id']] = ['id'=>(int)$h['user_id'],'name'=>$h['display_name'],'role'=>$h['slug'],'active'=>(int)$h['is_active'],'parent'=>0]; }
        foreach ($this->query("SELECT child_profile_id,parent_profile_id FROM {$p}sn_hr_assignments WHERE is_current=1 AND relationship_type='reports_to' ORDER BY id") as $a) {
            $child = $profile_users[$a['child_profile_id']] ?? 0;
            if ($child) { $this->people[$child]['parent'] = $profile_users[$a['parent_profile_id']] ?? 0; }
        }
        if (!current_user_can('manage_options') && isset($this->people[$uid]) && !$this->people[$uid]['active']) { throw new RuntimeException('حساب سازمانی غیرفعال است.'); }
        $this->allowed = current_user_can('manage_options') || $position === 'mis' ? null : $this->descendants($uid);
        if ($this->allowed !== null && $position !== 'seller' && !isset($this->people[$uid])) { $this->allowed = []; }
        $this->scope_hash = hash('sha256', wp_json_encode([$uid,$position,$this->allowed,$this->people,current_user_can('manage_options')]));
    }
    public function descendants(int $id): array {
        $found = [$id=>true]; $again = true;
        while ($again) { $again = false; foreach ($this->people as $uid=>$person) { if (!isset($found[$uid]) && isset($found[$person['parent']])) { $found[$uid] = true; $again = true; } } }
        return array_map('intval',array_keys($found));
    }
    public function name(int $id): string { return $id ? (($this->people[$id]['name'] ?? 'کاربر') . ' #' . $id) : ''; }
    public function query(string $sql): array {
        $start = microtime(true); $out = $this->db->get_results($sql, ARRAY_A);
        if ($this->db->last_error) { $id = substr(wp_generate_uuid4(),0,8); error_log('SN MIS report DB '.$id.' '. $this->db->last_error); throw new RuntimeException('خطای پایگاه داده گزارش؛ کد پیگیری '.$id); }
        if (microtime(true)-$start > 3) { error_log('SN MIS report slow query '.round(microtime(true)-$start,2).'s fingerprint='.substr(hash('sha256',$sql),0,12)); }
        return $out ?: [];
    }
    public function cols(string $table): array {
        if (!isset($this->columns[$table])) { $this->columns[$table] = array_column($this->query('SHOW COLUMNS FROM '.$this->db->prefix.$table),'Field'); }
        return $this->columns[$table];
    }
    public static function ids(array $ids): string { return implode(',',array_unique(array_map('intval',$ids))) ?: '0'; }
    public function scope_sql(string $r='r'): string {
        if ($this->allowed === null) { return '1=1'; }
        $p=$this->db->prefix; $ids=self::ids($this->allowed);
        // Use current custody, never an old manager assignment when an item exists.
        $item="(d.source_type='mis_data_row' AND d.source_id={$r}.id) OR (d.source_type='mis_pool' AND d.source_id IN (SELECT id FROM {$p}sn_mis_lead_pool WHERE source_row_id={$r}.id))";
        $newer=str_replace("d.","dn.",$item);
        $latest="NOT EXISTS (SELECT 1 FROM {$p}sn_distribution_items dn WHERE ({$newer}) AND (COALESCE(dn.updated_at,dn.created_at,'')>COALESCE(d.updated_at,d.created_at,'') OR (COALESCE(dn.updated_at,dn.created_at,'')=COALESCE(d.updated_at,d.created_at,'') AND dn.id>d.id)))";
        $owner="CASE WHEN d.current_owner_user_id>0 THEN d.current_owner_user_id WHEN d.status NOT IN ('returned_to_mis','cancelled','archived') THEN COALESCE(d.final_seller_user_id,0) ELSE 0 END";
        return "(EXISTS (SELECT 1 FROM {$p}sn_distribution_items d WHERE ({$item}) AND {$latest} AND {$owner} IN ({$ids})) OR (NOT EXISTS (SELECT 1 FROM {$p}sn_distribution_items d WHERE {$item}) AND COALESCE((SELECT COALESCE(NULLIF(p.assigned_seller_user_id,0),NULLIF(p.assigned_manager_user_id,0)) FROM {$p}sn_mis_lead_pool p WHERE p.source_row_id={$r}.id LIMIT 1),{$r}.assigned_manager_user_id,0) IN ({$ids})))";
    }
    public function options(): array {
        $p=$this->db->prefix;
        $batches=$this->query("SELECT b.id,b.title,b.campaign_code,b.data_category FROM {$p}sn_mis_import_batches b WHERE EXISTS (SELECT 1 FROM {$p}sn_mis_data_rows r WHERE r.batch_id=b.id AND ".$this->scope_sql().") ORDER BY b.id DESC");
        $people=[]; foreach ($this->people as $id=>$person) { if (isset(SN_MIS_Report_Model::ROLES[$person['role']]) && ($this->allowed===null || in_array($id,$this->allowed,true))) { $people[]=$person; } }
        return ['batches'=>$batches,'people'=>$people,'roles'=>SN_MIS_Report_Model::ROLES];
    }
    public function validate_filters(array $f): void {
        $allowed=$this->allowed;
        foreach (SN_MIS_Report_Model::ROLES as $role=>$label) {
            $id=$f[$role]; if (!$id) { continue; }
            if (!isset($this->people[$id]) || $this->people[$id]['role']!==$role || ($allowed!==null && !in_array($id,$allowed,true))) { throw new RuntimeException('فرد انتخاب‌شده خارج از سلسله‌مراتب مجاز است.'); }
            $allowed=$this->descendants($id);
        }
    }
    public function rows(int $after,int $max,array $f,int $limit=100): array {
        $p=$this->db->prefix; $where=['r.id>'.(int)$after,'r.id<='.(int)$max,$this->scope_sql()];
        foreach (['campaign'=>'b.campaign_code','category'=>'b.data_category','batch'=>'r.batch_id'] as $key=>$column) { if ($f[$key]!=='' && $f[$key]!==0) { $where[]=$this->db->prepare("{$column}=%s",$f[$key]); } }
        if ($f['validity']!=='') { $where[]=$this->db->prepare('r.row_status=%s',$f['validity']); }
        return $this->query("SELECT r.*,b.title batch_title,b.campaign_code,b.data_category,b.created_at batch_created_at FROM {$p}sn_mis_data_rows r LEFT JOIN {$p}sn_mis_import_batches b ON b.id=r.batch_id WHERE ".implode(' AND ',$where).' ORDER BY r.id LIMIT '.max(1,min(200,$limit)));
    }
    public function hydrate(array $rows): array {
        if (!$rows) { return []; } $p=$this->db->prefix;
        $rowids=self::ids(array_column($rows,'id')); $pools=$this->query("SELECT * FROM {$p}sn_mis_lead_pool WHERE source_row_id IN ({$rowids})"); $poolmap=[];
        foreach ($pools as $pool) { $poolmap[$pool['id']]=$pool; }
        $poolids=self::ids(array_column($pools,'id'));
        $items=$this->query("SELECT * FROM {$p}sn_distribution_items WHERE (source_type='mis_data_row' AND source_id IN ({$rowids})) OR (source_type='mis_pool' AND source_id IN ({$poolids}))");
        $itemids=self::ids(array_column($items,'id')); $leadids=self::ids(array_filter(array_column($items,'live_lead_id')));
        $states=$this->query("SELECT * FROM {$p}sn_seller_flow_states WHERE (source_kind='v4_distribution_item' AND source_id IN ({$itemids})) OR (source_kind='legacy_lead' AND source_id IN ({$leadids}))");
        $events=$this->query("SELECT id,source_kind,source_id,invoice_id,event_key,created_at FROM {$p}sn_seller_flow_events WHERE (source_kind='v4_distribution_item' AND source_id IN ({$itemids})) OR (source_kind='legacy_lead' AND source_id IN ({$leadids})) ORDER BY id");
        $logs=$this->query("SELECT id,item_id,action,actor_user_id,from_user_id,to_user_id,from_position,to_position,created_at,note FROM {$p}sn_distribution_item_logs WHERE item_id IN ({$itemids}) ORDER BY id");
        $links=[]; $state_map=[]; $logsmap=[]; $eventsmap=[];
        foreach ($states as $s) { $state_map[$s['source_kind'].':'.$s['source_id']]=$s; }
        foreach ($events as $e) { $eventsmap[$e['source_kind'].':'.$e['source_id']][]=$e; }
        foreach ($logs as $l) { $logsmap[$l['item_id']][]=$l; }
        $itemmap=[]; $byrow=[]; $leadmap=[];
        foreach ($items as $d) {
            $rid=$d['source_type']==='mis_data_row' ? (int)$d['source_id'] : (int)($poolmap[$d['source_id']]['source_row_id']??0);
            if (!$rid) { continue; } $itemmap[$d['id']]=$rid; $byrow[$rid][]=$d;
            if (!empty($d['live_lead_id'])) { $leadmap[(int)$d['live_lead_id']][]=$rid; }
            $refs=[];
            if (!empty($d['follow_invoice_id'])) { $refs[(int)$d['follow_invoice_id']][]='sn_distribution_items.follow_invoice_id#'.$d['id']; }
            foreach (['v4_distribution_item:'.$d['id'],'legacy_lead:'.($d['live_lead_id']??0)] as $key) {
                if (!empty($state_map[$key]['last_invoice_id'])) { $refs[(int)$state_map[$key]['last_invoice_id']][]='sn_seller_flow_states.last_invoice_id#'.$state_map[$key]['id']; }
                foreach ($eventsmap[$key]??[] as $e) { if ($e['invoice_id']) { $refs[(int)$e['invoice_id']][]='sn_seller_flow_events.invoice_id#'.$e['id']; } }
            }
            foreach ($refs as $iid=>$proof) { $links[$rid][$iid]=array_merge($links[$rid][$iid]??[],$proof); }
        }
        $invoiceids=[]; foreach ($links as $link) { $invoiceids=array_merge($invoiceids,array_keys($link)); }
        $ids=self::ids($invoiceids); $icol=$this->cols('sn_invoices');
        $select=array_intersect(['id','invoice_code','seller_id','lead_id','referral_item_id','status','invoice_status','payment_status','payment_workflow_status','payment_plan','created_at','paid_at','approved_at','payment_completed_at','payment_archive_reason','updated_at','payment_total_amount','final_total','product_price','paid_total_amount','remaining_amount'],$icol);
        $branches=[]; $projection='SELECT '.implode(',',$select)." FROM {$p}sn_invoices WHERE ";
        if ($ids!=='0') { $branches[]=$projection."id IN ({$ids})"; }
        if ($leadids!=='0') { $branches[]=$projection."lead_id IN ({$leadids})"; }
        if ($itemids!=='0' && in_array('referral_item_id',$icol,true)) { $branches[]=$projection."referral_item_id IN ({$itemids})"; }
        $invoices=$branches ? $this->query(implode(' UNION ',$branches)) : [];
        $allids=self::ids(array_column($invoices,'id')); $stagemap=[];
        foreach ($this->query("SELECT id,invoice_id,stage_no,stage_type,requested_amount,status,pay_method,approved_by_user_id,approved_at,paid_at,created_at FROM {$p}sn_invoice_payment_stages WHERE invoice_id IN ({$allids}) ORDER BY invoice_id,stage_no") as $s) { $stagemap[$s['invoice_id']][]=$s; }
        $invmap=[];
        foreach ($invoices as $i) {
            $iid=(int)$i['id']; $invmap[$iid]=SN_MIS_Report_Model::invoice($i,$stagemap[$iid]??[]);
            foreach ($leadmap[$i['lead_id']??0]??[] as $rid) { $links[$rid][$iid][]='sn_invoices.lead_id=sn_distribution_items.live_lead_id'; }
            if (isset($itemmap[$i['referral_item_id']??0])) { $links[$itemmap[$i['referral_item_id']]][$iid][]='sn_invoices.referral_item_id'; }
        }
        $out=[];
        foreach ($rows as $r) {
            $rid=(int)$r['id']; $ds=$byrow[$rid]??[];
            usort($ds,static fn($a,$b)=>strcmp(($b['updated_at']??$b['created_at']??'').sprintf('%020d',$b['id']),($a['updated_at']??$a['created_at']??'').sprintf('%020d',$a['id'])));
            $d=$ds[0]??[]; $pool=[]; foreach ($pools as $x) { if ((int)$x['source_row_id']===$rid) { $pool=$x; break; } }
            $owner=(int)($d['current_owner_user_id']??0);
            if (!$owner && $d && !in_array($d['status'],['returned_to_mis','cancelled','archived'],true)) { $owner=(int)($d['final_seller_user_id']??0); }
            if (!$d) { $owner=(int)(($pool['assigned_seller_user_id']??0)?:($pool['assigned_manager_user_id']??0)?:($r['assigned_manager_user_id']??0)); }
            if ($this->allowed!==null && !in_array($owner,$this->allowed,true)) { continue; }
            $s=$state_map['v4_distribution_item:'.($d['id']??0)]??$state_map['legacy_lead:'.($d['live_lead_id']??0)]??null;
            $seller=(int)(($d['final_seller_user_id']??0)?:($pool['assigned_seller_user_id']??0));
            if (($d['current_owner_position']??'')==='seller') { $seller=$owner; }
            if (!$seller && $s) { $seller=(int)$s['seller_user_id']; }
            $raw=$s!==null ? (string)$s['flow_status'] : (string)($d['seller_contact_status']??'');
            $flow=SN_Seller_Flow::instance()->status_slug($raw); if (!$flow) { $flow=$raw!=='' ? 'other:'.$raw : 'unrecorded'; }
            $reason=$s!==null ? (string)($s['not_purchase_reason']??'') : (string)($d['not_purchase_reason']??'');
            $issues=[];
            if (count($ds)>1) { $issues[]='چند آیتم توزیع؛ مالک و پیگیری از تازه‌ترین updated_at/id، همه آیتم‌ها در جزئیات'; }
            if ($s && $this->allowed!==null && !in_array((int)$s['seller_user_id'],$this->allowed,true)) { $reason=''; $raw=''; $flow='restricted'; $issues[]='پیگیری فروشنده قبلی خارج از محدوده؛ نمایش داده نشد'; }
            $history=[]; $followtimes=[]; $assignment=[];
            foreach ($ds as $di) {
                foreach ($logsmap[$di['id']]??[] as $log) {
                    foreach (['actor_user_id','from_user_id','to_user_id'] as $k) { $log[$k.'_name']=$this->name((int)$log[$k]); }
                    $history[]=$log;
                    if (((int)$log['to_user_id']>0 && (int)$log['to_user_id']!==(int)$log['from_user_id']) || in_array($log['action'],['returned_to_mis','returned_to_manager','returned_to_owner'],true)) { $assignment[]=$log['created_at']; }
                }
                foreach (['v4_distribution_item:'.$di['id'],'legacy_lead:'.($di['live_lead_id']??0)] as $key) {
                    foreach ($eventsmap[$key]??[] as $e) { $followtimes[]=$e['created_at']; }
                }
            }
            if ($ds && !$history) { $issues[]='آیتم توزیع بدون لاگ تاریخی؛ سابقه قابل بازسازی نیست'; }
            $manager_assigned=(int)!empty($r['assigned_manager_user_id']); $delivered=(int)($seller>0); $returned=0; $transferred=0;
            foreach ($history as $l) {
                if (isset(SN_MIS_Report_Model::ROLES[$l['to_position']]) && $l['to_position']!=='seller' && $l['to_user_id']) { $manager_assigned=1; }
                if ($l['to_position']==='seller' && $l['to_user_id']) { $delivered=1; }
                if (in_array($l['action'],['returned_to_mis','returned_to_manager','returned_to_supervisor','returned_to_owner'],true)) { $returned=1; }
                if ($l['from_user_id'] && $l['to_user_id'] && $l['from_user_id']!==$l['to_user_id']) { $transferred=1; }
            }
            if (SN_MIS_Report_Model::valid_time($r['assigned_at']??null)) { $assignment[]=$r['assigned_at']; }
            if ($s && SN_MIS_Report_Model::valid_time($s['last_activity_at']??null)) { $followtimes[]=$s['last_activity_at']; }
            if (!$s && SN_MIS_Report_Model::valid_time($d['seller_updated_at']??null)) { $followtimes[]=$d['seller_updated_at']; }
            $cinvoices=[]; $missing=[];
            foreach ($links[$rid]??[] as $iid=>$proof) {
                if (!isset($invmap[$iid])) { $missing[]=(string)$iid; continue; }
                $i=$invmap[$iid];
                if ($this->allowed!==null && !in_array((int)$i['seller_id'],$this->allowed,true)) { $issues[]='فاکتور خارج از محدوده؛ جزئیات نمایش داده نشد'; continue; }
                $i['link_source']=implode(' | ',array_unique($proof)); $cinvoices[]=$i;
            }
            if (!$delivered && $flow==='unrecorded') { $flow='not_delivered'; }
            if ($missing) { $issues[]='شناسه قطعی با فاکتور مفقود: '.implode(',',$missing); }
            if ($flow==='pre_invoice' && !$cinvoices) { $issues[]='پیگیری پیش‌فاکتور بدون فاکتور قابل مشاهده با پیوند قطعی'; }
            $roles=array_fill_keys(array_keys(SN_MIS_Report_Model::ROLES),0); $cursor=$seller?:$owner; $seen=[];
            while ($cursor && !isset($seen[$cursor])) { $seen[$cursor]=true; $person=$this->people[$cursor]??null; if (!$person) { break; } if (isset($roles[$person['role']])) { $roles[$person['role']]=$cursor; } $cursor=(int)$person['parent']; }
            if ($seller) { $roles['seller']=$seller; }
            $times=['import'=>[$r['created_at']?:$r['batch_created_at']],'assignment'=>$assignment,'followup'=>$followtimes,'invoice'=>array_column($cinvoices,'created_at'),'payment'=>[]];
            foreach ($cinvoices as $i) { $times['payment']=array_merge($times['payment'],$i['payment_times']); }
            $activity=array_filter(array_merge($followtimes,$assignment,array_column($cinvoices,'created_at'),$times['payment'],array_column($cinvoices,'updated_at'),array_column($history,'created_at'),[$d['updated_at']??''])); sort($activity);
            $c=array_merge($roles,['row_id'=>$rid,'batch_id'=>(int)$r['batch_id'],'batch_title'=>$r['batch_title']??'دسته مفقود','campaign'=>(string)($r['campaign_code']??''),'category'=>(string)($r['data_category']??''),'phone'=>(string)$r['customer_phone'],'customer_name'=>(string)(($d['seller_customer_name']??'')?:$r['customer_name']),'customer_name_source'=>!empty($d['seller_customer_name'])?'sn_distribution_items.seller_customer_name':'sn_mis_data_rows.customer_name','city'=>(string)($r['city']??''),'province'=>(string)($r['province']??''),'validity'=>(string)$r['row_status'],'duplicate_reason'=>(string)($r['duplicate_reason']??''),'owner'=>$owner,'owner_name'=>$this->name($owner),'seller_name'=>$this->name($seller),'distribution'=>(string)($d['status']??$pool['pool_status']??'unassigned'),'flow'=>$flow,'flow_raw'=>$raw,'flow_source'=>$s!==null?'sn_seller_flow_states#'.$s['id']:'sn_distribution_items.seller_contact_status (fallback)','reason'=>$reason,'reason_source'=>$s!==null?'sn_seller_flow_states.not_purchase_reason':'sn_distribution_items.not_purchase_reason (if present); seller_notes excluded','reason_quality'=>$flow==='not_purchased'?SN_MIS_Report_Model::quality($reason):'not_applicable','last_activity'=>end($activity)?:'','manager_assigned'=>$manager_assigned,'delivered'=>$delivered,'returned'=>$returned,'transferred'=>$transferred,'times'=>$times,'invoices'=>$cinvoices,'history'=>$history,'issues'=>$issues,'pool_ids'=>array_values(array_map('intval',array_column(array_filter($pools,static fn($x)=>(int)$x['source_row_id']===$rid),'id'))),'item_ids'=>array_map('intval',array_column($ds,'id')),'missing_invoice_ids'=>$missing,'imported_at'=>$times['import'][0]??'','assigned_at'=>$r['assigned_at']??'']);
            $c['cancel_reasons']=implode(' | ',array_filter(array_map(static fn($i)=>!empty($i['payment_archive_reason']) ? '#'.$i['id'].': '.$i['payment_archive_reason'] : '',$cinvoices)));
            $c['linked']=(int)(count($cinvoices)>0); $c['pre_case']=(int)($flow==='pre_invoice'||$c['linked']);
            $c['approved']=(int)(array_sum(array_column($cinvoices,'approved'))>0); $c['completed']=(int)(array_sum(array_column($cinvoices,'completed'))>0);
            $c['paid']=array_sum(array_column($cinvoices,'paid')); $c['remaining']=array_sum(array_column($cinvoices,'remaining'));
            $c['reason_review']=(int)($flow==='not_purchased' && $c['reason_quality']!=='recorded');
            $c['issue_count']=count($issues)+count(array_filter($cinvoices,static fn($i)=>count($i['issues'])>0));
            $out[]=$c;
        }
        return $out;
    }
}
