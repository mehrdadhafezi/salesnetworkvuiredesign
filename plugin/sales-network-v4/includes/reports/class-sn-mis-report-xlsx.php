<?php
if (!defined('ABSPATH')) { exit; }
/** Disk-backed OOXML writer. Text is inlineStr, never an Excel formula. */
final class SN_MIS_Report_XLSX {
    private const SHEETS=['خلاصه','کمپین و دسته','افراد','پرونده‌ها','فاکتورها','مراحل پرداخت','تاریخچه تحویل','بازبینی دلیل','پیوند پرونده فاکتور','ادامه متن‌های بلند'];
    private string $dir; private array $counts=[];
    public function __construct(string $dir) { $this->dir=$dir; }
    private static function xml($v): string { return htmlspecialchars(preg_replace('/[^\x{0009}\x{000A}\x{000D}\x{0020}-\x{D7FF}\x{E000}-\x{FFFD}\x{10000}-\x{10FFFF}]/u','',(string)$v),ENT_XML1|ENT_QUOTES,'UTF-8'); }
    private static function column(int $n): string { $v=''; do { $v=chr(65+$n%26).$v; $n=intdiv($n,26)-1; } while ($n>=0); return $v; }
    private function write(string $path,string $content,int $flags=0): void { if (file_put_contents($path,$content,$flags)!==strlen($content)) { throw new RuntimeException('فضای دیسک برای خروجی کافی نیست یا نوشتن فایل مجاز نیست.'); } }
    public function row(int $sheet,array $values): void {
        $number=($this->counts[$sheet]??0)+1;
        if ($number>1048576) { throw new RuntimeException('تعداد ردیف یک شیت از سقف XLSX بیشتر است؛ خروجی را با فیلترهای کوچک‌تر تهیه کنید.'); }
        $xml='<row r="'.$number.'">';
        foreach (array_values($values) as $col=>$v) {
            if (is_string($v)) {
                preg_match_all('/.{1,30000}/us',$v,$chunks);
                if (count($chunks[0])>1 && $sheet!==10) {
                    foreach ($chunks[0] as $part=>$chunk) { $this->row(10,[self::SHEETS[$sheet-1],(string)$number,self::column($col),(string)($part+1),$chunk]); }
                    $v=$chunks[0][0].' [ادامه در شیت ادامه متن‌های بلند]';
                }
            }
            $ref=self::column($col).$number;
            $xml.=is_int($v)||is_float($v) ? '<c r="'.$ref.'"><v>'.self::xml($v).'</v></c>' : '<c r="'.$ref.'" t="inlineStr"><is><t xml:space="preserve">'.self::xml($v??'').'</t></is></c>';
        }
        $this->write($this->dir.'/sheet'.$sheet.'.xml',$xml.'</row>',FILE_APPEND); $this->counts[$sheet]=$number;
    }
    private function offsets(): array { $out=['counts'=>$this->counts,'sizes'=>[]]; for($s=1;$s<=10;$s++) { clearstatcache(true,$this->dir.'/sheet'.$s.'.xml'); $out['sizes'][$s]=filesize($this->dir.'/sheet'.$s.'.xml'); } return $out; }
    private function restore(array $saved): void {
        $this->counts=$saved['counts'];
        foreach ($saved['sizes'] as $s=>$size) { $file=fopen($this->dir.'/sheet'.(int)$s.'.xml','c+b'); if (!$file || !ftruncate($file,(int)$size)) { throw new RuntimeException('فایل موقت خروجی در دسترس نیست؛ خروجی تازه بسازید.'); } fclose($file); }
    }
    public static function remove(string $dir): void {
        $base=realpath(sys_get_temp_dir()); $real=realpath($dir);
        if (!$real || dirname($real)!==$base || !preg_match('/^sn-mis-report-[a-f0-9]{32}$/D',basename($real))) { return; }
        foreach (glob($real.'/*')?:[] as $file) { if (is_file($file)) { unlink($file); } } rmdir($real);
    }
    public static function begin(array $job,SN_MIS_Report_Store $store,SN_MIS_Report_Source $src): void {
        if (!class_exists('ZipArchive')) { throw new RuntimeException('افزونه PHP zip روی سرور فعال نیست؛ برای خروجی XLSX آن را فعال کنید.'); }
        $base=realpath(sys_get_temp_dir());
        foreach ([realpath(ABSPATH),realpath($_SERVER['DOCUMENT_ROOT']??ABSPATH)] as $web) { if ($web && ($base===$web || strpos($base,$web.DIRECTORY_SEPARATOR)===0)) { throw new RuntimeException('مسیر موقت PHP داخل مسیر عمومی وب است؛ یک مسیر موقت خصوصی تنظیم کنید.'); } }
        $dir=$base.'/sn-mis-report-'.bin2hex(random_bytes(16));
        if (!mkdir($dir,0700)) { throw new RuntimeException('ساخت پوشه خصوصی خروجی ممکن نیست.'); }
        $x=new self($dir);
        try {
            for($s=1;$s<=10;$s++) { $x->write($dir.'/sheet'.$s.'.xml','<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetViews><sheetView workbookViewId="0" rightToLeft="1"/></sheetViews><sheetData>'); }
            $headers=[1=>['گروه','شاخص','مقدار'],2=>['کمپین','شناسه دسته','دسته','پرونده یکتا','تحویل','جواب نداده','تماس مجدد','عدم خرید','پیگیری پیش‌فاکتور','ثبت نشده','دارای پیش‌فاکتور','پیوند قطعی','تأییدشده','تکمیل‌شده'],3=>['سطح جاری HR','شناسه شخص','نام','پرونده یکتا','تحویل','جواب نداده','تماس مجدد','عدم خرید','پیگیری پیش‌فاکتور','ثبت نشده','دارای پیش‌فاکتور','پیوند قطعی','تأییدشده','تکمیل‌شده'],4=>['شناسه ردیف','کمپین','شناسه دسته','عنوان دسته','دسته داده','شماره (متن)','نام','شهر','استان','اعتبار','مالک جاری','وضعیت توزیع','پیگیری','منشأ پیگیری','آخرین فعالیت','دلیل کامل','کنترل کیفیت','شناسه‌های فاکتور','کدها','وضعیت فاکتورها','پرداخت در پرونده','مانده در پرونده','پرونده با فروش تکمیل','شناسه مخزن','شناسه توزیع','اختلاف‌ها','زمان ورود','دلیل انصراف فاکتورها'],5=>['شناسه فاکتور','کد','شناسه فروشنده','وضعیت','گردش پرداخت','طرح پرداخت','مبلغ','پرداخت تأییدشده','مانده','تأیید','تکمیل','پرداخت ناقص','منشأ مبلغ','زمان صدور','زمان پرداخت','دلیل انصراف','اختلاف‌ها'],6=>['شناسه فاکتور','شناسه مرحله','شماره مرحله','نوع','مبلغ','وضعیت','روش','تأییدکننده','زمان تأیید','زمان پرداخت','زمان ایجاد'],7=>['شناسه پرونده','شناسه لاگ','شناسه آیتم','زمان','نوع رویداد','اقدام‌کننده','فرستنده','گیرنده','سمت فرستنده','سمت گیرنده','متن رویداد'],8=>['شناسه پرونده','کمپین','شناسه دسته','فروشنده','دلیل اصلی کامل','برچسب کیفیت','منشأ دلیل'],9=>['شناسه پرونده','شناسه فاکتور','منشأ پیوند قطعی'],10=>['شیت مبدأ','ردیف مبدأ','ستون مبدأ','بخش متن','متن کامل بخش']];
            foreach ($headers as $s=>$h) { $x->row($s,$h); }
            $summary=$store->summary($job['id'],$src);
            foreach ($summary['filters'] as $k=>$v) { $x->row(1,['فیلتر',$k,(string)$v]); }
            foreach (['created_at','finished_at'] as $k) { $x->row(1,['زمان گردآوری UTC',$k,$job[$k]??'']); }
            foreach ($summary['totals'] as $k=>$v) { $x->row(1,['پرونده',$k,(int)$v]); }
            foreach ($summary['invoice_totals'] as $k=>$v) { $x->row(1,['فاکتور یکتا',$k,(float)$v]); }
            foreach ($summary['breakdown'] as $k=>$rows) { foreach ($rows as $r) { $x->row(1,[$k,$r['label'],(int)$r['total']]); } }
            foreach ($summary['invoice_groups'] as $g) { foreach (['total_count','amount','paid','remaining','approved','completed','partial'] as $k) { $x->row(1,[$g['status'].' / '.$g['workflow'],$k,(float)$g[$k]]); } }
            foreach ($summary['notes'] as $note) { $x->row(1,['تعریف',$note,'']); }
            foreach ($summary['shared_invoices'] as $shared) { $x->row(1,['فاکتور مشترک',(string)$shared['invoice_id'],(int)$shared['case_count']]); }
            $metrics=['total','delivered','no_answer','callback','not_purchased','pre_invoice','unrecorded','pre_case','linked','approved','completed'];
            foreach ($summary['groups'] as $g) { $x->row(2,array_merge([$g['campaign'],(string)$g['batch_id'],$g['category']],array_map(static fn($k)=>(int)$g[$k],$metrics))); }
            foreach ($summary['people'] as $g) { $x->row(3,array_merge([$g['role'],(string)$g['person_id'],$g['name']],array_map(static fn($k)=>(int)$g[$k],$metrics))); }
            $x->row(3,[]); $x->row(3,['تحویل تاریخی از لاگ — مستقل از عملکرد جاری','شناسه شخص','نام','پرونده یکتا','تعداد رویداد','اولین دریافت','آخرین دریافت']);
            foreach ($summary['historical_people'] as $h) { $x->row(3,[$h['role'],(string)$h['person_id'],$h['name'],(int)$h['unique_cases'],(int)$h['events'],$h['first_at'],$h['last_at']]); }
            $store->checked($store->db->update($store->table('jobs'),['export_dir'=>$dir,'export_cursor'=>0,'export_status'=>'cases','export_offsets'=>wp_json_encode($x->offsets())],['id'=>$job['id']]));
        } catch (Throwable $e) { self::remove($dir); throw $e; }
    }
    public static function step(array $job,SN_MIS_Report_Store $store,SN_MIS_Report_Source $src): array {
        if ($job['export_status']==='ready') { return ['status'=>'ready']; }
        if (!$job['export_status']) { self::begin($job,$store,$src); return ['status'=>'cases']; }
        $x=new self($job['export_dir']); $saved=json_decode($job['export_offsets'],true); $x->restore($saved);
        try {
            $phase=$job['export_status']; $cursor=(int)$job['export_cursor'];
            if ($phase==='cases') {
                $rows=$src->query($store->db->prepare('SELECT payload FROM '.$store->table('cases').' WHERE job_id=%s AND row_id>%d ORDER BY row_id LIMIT 100',$job['id'],$cursor));
                foreach ($rows as $row) {
                    $c=json_decode($row['payload'],true); $cursor=$c['row_id'];
                    $x->row(4,[(string)$c['row_id'],$c['campaign'],(string)$c['batch_id'],$c['batch_title'],$c['category'],$c['phone'],$c['customer_name'],$c['city'],$c['province'],$c['validity'],$c['owner_name'],$c['distribution'],$c['flow'],$c['flow_source'],$c['last_activity'],$c['reason'],$c['reason_quality'],implode(',',array_column($c['invoices'],'id')),implode(',',array_column($c['invoices'],'invoice_code')),implode(',',array_column($c['invoices'],'status')),(float)$c['paid'],(float)$c['remaining'],(int)$c['completed'],implode(',',$c['pool_ids']),implode(',',$c['item_ids']),implode(' | ',$c['issues']),$c['imported_at'],$c['cancel_reasons']??'']);
                    foreach ($c['history'] as $l) { $x->row(7,[(string)$c['row_id'],(string)$l['id'],(string)$l['item_id'],$l['created_at'],$l['action'],$l['actor_user_id_name'],$l['from_user_id_name'],$l['to_user_id_name'],$l['from_position'],$l['to_position'],$l['note']]); }
                    foreach ($c['invoices'] as $i) { $x->row(9,[(string)$c['row_id'],(string)$i['id'],$i['link_source']]); }
                    if ($c['reason_review']) { $x->row(8,[(string)$c['row_id'],$c['campaign'],(string)$c['batch_id'],$c['seller_name'],$c['reason'],$c['reason_quality'],$c['reason_source']]); }
                }
                if (count($rows)<100) { $phase='invoices'; $cursor=0; }
            } elseif ($phase==='invoices') {
                $rows=$src->query($store->db->prepare('SELECT payload FROM '.$store->table('invoices').' WHERE job_id=%s AND invoice_id>%d ORDER BY invoice_id LIMIT 100',$job['id'],$cursor));
                foreach ($rows as $row) {
                    $i=json_decode($row['payload'],true); $cursor=$i['id'];
                    $x->row(5,[(string)$i['id'],$i['invoice_code'],(string)$i['seller_id'],$i['status'],$i['payment_workflow_status']??'',$i['payment_plan']??'',(float)$i['total'],(float)$i['paid'],(float)$i['remaining'],(int)$i['approved'],(int)$i['completed'],(int)$i['partial'],$i['payment_source'],$i['created_at'],$i['paid_at']??'',$i['payment_archive_reason']??'',implode(' | ',$i['issues'])]);
                    foreach ($i['stages'] as $s) { $x->row(6,[(string)$i['id'],(string)$s['id'],(string)$s['stage_no'],$s['stage_type'],(float)$s['requested_amount'],$s['status'],$s['pay_method'],(string)$s['approved_by_user_id'],$s['approved_at'],$s['paid_at'],$s['created_at']]); }
                }
                if (count($rows)<100) { $phase='finalize'; $cursor=0; }
            } elseif ($phase==='finalize') {
                $summary=$store->summary($job['id'],$src);
                if (($x->counts[4]-1)!==(int)$summary['totals']['total'] || ($x->counts[5]-1)!==(int)$summary['invoice_totals']['total_count'] || ($x->counts[8]-1)!==(int)$summary['totals']['reason_review']) { throw new RuntimeException('تطبیق شمارش اکسل ناموفق بود؛ فایل منتشر نشد.'); }
                $x->row(1,['کنترل تطبیق','پرونده / فاکتور / بازبینی','PASS']); $x->finish(); $phase='ready';
            }
            $store->checked($store->db->update($store->table('jobs'),['export_cursor'=>$cursor,'export_status'=>$phase,'export_offsets'=>wp_json_encode($x->offsets())],['id'=>$job['id']]));
            return ['status'=>$phase];
        } catch (Throwable $e) { $x->restore($saved); throw $e; }
    }
    private function finish(): void {
        $zip=new ZipArchive(); $file=$this->dir.'/report.partial';
        if ($zip->open($file,ZipArchive::CREATE|ZipArchive::OVERWRITE)!==true) { throw new RuntimeException('ساخت فایل ZIP خروجی ناموفق بود.'); }
        $types='<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>';
        $book='<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>';
        $rels='<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">';
        for($s=1;$s<=10;$s++) {
            $this->write($this->dir.'/sheet'.$s.'.xml','</sheetData></worksheet>',FILE_APPEND);
            if (!$zip->addFile($this->dir.'/sheet'.$s.'.xml','xl/worksheets/sheet'.$s.'.xml')) { throw new RuntimeException('افزودن شیت به خروجی ناموفق بود.'); }
            $types.='<Override PartName="/xl/worksheets/sheet'.$s.'.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>';
            $book.='<sheet name="'.self::xml(self::SHEETS[$s-1]).'" sheetId="'.$s.'" r:id="rId'.$s.'"/>';
            $rels.='<Relationship Id="rId'.$s.'" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet'.$s.'.xml"/>';
        }
        $zip->addFromString('[Content_Types].xml',$types.'</Types>');
        $zip->addFromString('xl/workbook.xml',$book.'</sheets></workbook>');
        $zip->addFromString('xl/_rels/workbook.xml.rels',$rels.'</Relationships>');
        $zip->addFromString('_rels/.rels','<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>');
        if (!$zip->close() || !rename($file,$this->dir.'/report.xlsx')) { throw new RuntimeException('نهایی‌سازی فایل اکسل ناموفق بود.'); }
    }
}
