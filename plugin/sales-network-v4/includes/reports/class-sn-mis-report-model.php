<?php
if (!defined('ABSPATH')) { exit; }

/** Pure report rules. Never writes back to CRM business records. */
final class SN_MIS_Report_Model {
    public const ROLES = ['sales_deputy'=>'معاون فروش','sales_manager'=>'مدیر فروش','senior_supervisor'=>'سرپرست ارشد','supervisor'=>'سرپرست','seller'=>'فروشنده'];
    public static function valid_time($value): bool {
        if (!is_string($value) || !preg_match('/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/D', $value)) { return false; }
        $d = DateTimeImmutable::createFromFormat('!Y-m-d H:i:s', $value);
        return $d && $d->format('Y-m-d H:i:s') === $value;
    }
    public static function quality(string $reason): string {
        $trim = trim(preg_replace('/[\s\x{200c}\x{200d}\x{200e}\x{200f}]+/u', ' ', $reason));
        if ($trim === '') { return 'empty'; }
        preg_match_all('/./us', $trim, $chars);
        if (count($chars[0]) <= 2) { return 'short'; }
        $letters = preg_replace('/[^\p{L}\p{N}]/u', '', $trim);
        if ($letters === '' || preg_match('/^(.)\1+$/us', $letters) || in_array($trim, ['تست','test','ندارد','نامشخص','---','...'], true)) { return 'meaningless'; }
        return 'recorded'; // Syntactic check only, never a semantic guarantee.
    }
    public static function invoice(array $i, array $stages): array {
        $total = (float)($i['payment_total_amount'] ?? $i['final_total'] ?? $i['product_price'] ?? 0);
        $total = max(0, $total); $approved = in_array($i['status'], ['paid','approved'], true);
        $workflow = (string)($i['payment_workflow_status'] ?? '');
        $evidence = $workflow === 'completed' || self::valid_time($i['paid_at'] ?? null);
        $stage_paid = 0; $times = []; $pending = false;
        foreach ($stages as $s) {
            if (in_array($s['status'], ['paid','approved'], true)) {
                $stage_paid += (float)$s['requested_amount'];
                foreach (['approved_at','paid_at'] as $key) { if (self::valid_time($s[$key] ?? null)) { $times[] = $s[$key]; } }
            } elseif (!in_array($s['status'], ['cancelled','rejected','payment_archived'], true)) { $pending = true; }
        }
        $stored = max(0, (float)($i['paid_total_amount'] ?? 0)); $issues = [];
        if ($stages) {
            $paid = max(0, $stage_paid); $source = 'sn_invoice_payment_stages:paid/approved';
            if (abs($stored - $paid) > 0.01) { $issues[] = 'اختلاف جمع مراحل با paid_total_amount'; }
        } elseif ($stored > 0) { $paid = $stored; $source = 'sn_invoices.paid_total_amount'; }
        elseif ($approved && $evidence && ($i['payment_plan'] ?? 'full') !== 'partial' && (float)($i['remaining_amount'] ?? 0) <= 0) {
            $paid = $total; $source = 'legacy:status+paid_at/completed';
        } else { $paid = 0; $source = 'بدون مبلغ پرداخت تأییدشده'; }
        $remaining = max(0, $total - $paid);
        if (isset($i['remaining_amount']) && abs((float)$i['remaining_amount'] - $remaining) > 0.01) { $issues[] = 'اختلاف مانده ذخیره‌شده با جمع پرداخت'; }
        $stored_partial = isset($i['remaining_amount']) && (float)$i['remaining_amount'] > 0.01;
        $complete = $approved && $evidence && $remaining <= 0.01 && !$stored_partial && !$pending;
        if ($approved && !$complete) { $issues[] = 'تأیید فاکتور بدون احراز تکمیل پرداخت'; }
        if ($paid > $total + 0.01) { $issues[] = 'پرداخت بیشتر از مبلغ فاکتور'; }
        foreach (['paid_at','payment_completed_at','approved_at'] as $key) { if (self::valid_time($i[$key] ?? null)) { $times[] = $i[$key]; } }
        $i = array_intersect_key($i, array_flip(['id','invoice_code','seller_id','status','invoice_status','payment_status','payment_workflow_status','payment_plan','created_at','paid_at','approved_at','payment_completed_at','payment_archive_reason','updated_at']));
        return array_merge($i, ['total'=>$total,'paid'=>$paid,'remaining'=>$remaining,'approved'=>(int)$approved,'completed'=>(int)$complete,'partial'=>(int)($paid>0 && !$complete),'payment_source'=>$source,'payment_times'=>array_values(array_unique($times)),'issues'=>$issues,'stages'=>$stages]);
    }
    public static function filters(array $raw): array {
        $f = [];
        foreach (['campaign','category','validity','distribution','flow','invoice_status','payment','reason_quality','search','basis','from','to'] as $key) {
            if (isset($raw[$key]) && !is_scalar($raw[$key])) { throw new RuntimeException('قالب فیلتر نامعتبر است.'); }
            $f[$key] = trim((string)($raw[$key] ?? ''));
            if (strlen($f[$key]) > 300) { throw new RuntimeException('فیلتر بیش از حد طولانی است.'); }
        }
        foreach (array_merge(['batch'], array_keys(self::ROLES)) as $key) { $f[$key] = max(0, (int)($raw[$key] ?? 0)); }
        if (!in_array($f['basis'], ['import','assignment','followup','invoice','payment'], true)) { $f['basis'] = 'import'; }
        foreach (['from','to'] as $key) {
            if ($f[$key] === '') { continue; }
            $f[$key] = str_replace('T',' ', $f[$key]);
            if (strlen($f[$key]) === 16) { $f[$key] .= $key === 'to' ? ':59' : ':00'; }
            if (!self::valid_time($f[$key])) { throw new RuntimeException('تاریخ و ساعت میلادی نامعتبر است.'); }
        }
        if ($f['from'] && $f['to'] && $f['from'] > $f['to']) { throw new RuntimeException('ابتدای بازه بعد از انتهای بازه است.'); }
        return $f;
    }
    public static function matches(array $c, array $f): bool {
        foreach (['validity','distribution','flow','reason_quality'] as $key) { if ($f[$key] !== '' && $f[$key] !== (string)$c[$key]) { return false; } }
        foreach (self::ROLES as $role=>$label) { if ($f[$role] && (int)$c[$role] !== $f[$role]) { return false; } }
        if ($f['search'] !== '' && strpos(implode(' ', [$c['row_id'],$c['phone'],$c['customer_name'],implode(' ',array_column($c['invoices'],'invoice_code'))]), $f['search']) === false) { return false; }
        $candidates=$c['invoices'];
        if ($f['invoice_status'] === 'unlinked' && count($candidates)) { return false; }
        if ($f['invoice_status'] !== '' && $f['invoice_status'] !== 'unlinked') { $candidates=array_filter($candidates,static fn($i)=>$i['status']===$f['invoice_status']); }
        if ($f['payment'] !== '') {
            $candidates=array_filter($candidates,static fn($i)=>
                ($f['payment']==='completed' && $i['completed']) || ($f['payment']==='partial' && $i['partial']) ||
                ($f['payment']==='unpaid' && !$i['paid']) || ($i['payment_workflow_status']??'')===$f['payment']);
        }
        if (($f['invoice_status']!=='' && $f['invoice_status']!=='unlinked' || $f['payment']!=='') && !$candidates) { return false; }
        if ($f['from'] || $f['to']) {
            $times = $c['times'][$f['basis']] ?? []; $found = false;
            if ($f['basis']==='invoice') { $times=array_column($candidates,'created_at'); }
            if ($f['basis']==='payment') { $times=[]; foreach ($candidates as $candidate) { $times=array_merge($times,$candidate['payment_times']); } }
            foreach ($times as $time) { if (self::valid_time($time) && (!$f['from'] || $time >= $f['from']) && (!$f['to'] || $time <= $f['to'])) { $found = true; break; } }
            if (!$found) { return false; }
        }
        return true;
    }
}
