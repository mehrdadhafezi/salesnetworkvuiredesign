<?php
if (!defined('ABSPATH')) { exit; }

/** Independent bank transactions; invoice fields are only a compatibility projection. */
class SN_Payment_Transactions
{
    public static function rows($invoice): array
    {
        global $wpdb;
        $invoice = (object)$invoice;
        $rows = $wpdb->get_results($wpdb->prepare("SELECT * FROM {$wpdb->prefix}sn_payments WHERE invoice_id=%d ORDER BY payment_stage_no ASC,id ASC", (int)$invoice->id), ARRAY_A) ?: [];
        foreach ($rows as &$row) {
            $proof = json_decode((string)($row['proof_data'] ?? ''), true);
            $row['is_transaction'] = is_array($proof);
            $row['is_gateway'] = !empty($row['authority']);
            if (!is_array($proof)) {
                // Historical rows never had their own proof snapshot.
                $proof = (int)($row['payment_stage_no'] ?? 1) === (int)($invoice->current_payment_stage ?? 1) && empty($row['authority']) ? (array)$invoice : [];
            }
            $row['proof'] = array_intersect_key($proof, array_flip(self::proof_fields()));
            $row['receipt_urls'] = SN_Helpers::receipt_urls($proof);
            $row['amount_fmt'] = SN_Helpers::format_price((float)$row['amount']);
            $row['stage_no'] = (int)($row['payment_stage_no'] ?? 1);
            $row['source_label'] = SN_Helpers::payment_source_label((string)($row['payment_source'] ?? ''));
            $row['status_label'] = ['pending'=>'در انتظار بررسی','approved'=>'تأیید شده','paid'=>'پرداخت شده','verified'=>'پرداخت درگاهی ثبت شده','rejected'=>'رد شده','failed'=>'ناموفق','expired'=>'منقضی'][$row['status']] ?? (string)$row['status'];
            $row['created_at_jalali'] = SN_Helpers::gregorian_to_jalali_date((string)$row['created_at']);
            unset($row['proof_data'], $row['submission_key'], $row['authority']);
        }
        unset($row);
        return $rows;
    }

    public static function proof_fields(): array
    {
        return ['entry_mode','manual_transfer_type','manual_account_last6','manual_tracking_last6','manual_account_owner','manual_card_from','manual_card_to','manual_card_to_number','manual_paid_at','manual_paid_at_jalali','receipt_url','receipt_file','receipt_urls'];
    }

    public static function pending_rows(int $invoice_id, int $stage_no): array
    {
        global $wpdb;
        return $wpdb->get_results($wpdb->prepare("SELECT * FROM {$wpdb->prefix}sn_payments WHERE invoice_id=%d AND payment_stage_no=%d AND status='pending' AND COALESCE(authority,'')='' ORDER BY id ASC", $invoice_id, $stage_no), ARRAY_A) ?: [];
    }

    public static function summary(array $rows, int $stage_no, float $due): array
    {
        $pending = 0.0; $recorded = 0.0; $count = 0; $independent = false;
        foreach ($rows as $row) {
            if ((int)($row['payment_stage_no'] ?? $row['stage_no'] ?? 1) !== $stage_no) { continue; }
            if (!empty($row['is_transaction']) || !empty($row['proof_data'])) { $independent = true; }
            $is_gateway = !empty($row['authority']) || !empty($row['is_gateway']) || in_array((string)($row['pay_method'] ?? ''), ['online','gateway','zibal','zarinpal'], true);
            // An unfinished gateway request is not a registered bank transfer.
            if ((string)($row['status'] ?? '') === 'pending' && $is_gateway) { continue; }
            if (in_array((string)($row['status'] ?? ''),['pending','approved','paid','verified'],true)) { $recorded += max(0,(float)$row['amount']); }
            if ((string)($row['status'] ?? '') !== 'pending' || !empty($row['authority']) || !empty($row['is_gateway'])) { continue; }
            if (in_array((string)($row['pay_method'] ?? ''), ['online','gateway','zibal','zarinpal'], true)) { continue; }
            $pending += max(0, (float)$row['amount']); $count++;
        }
        return ['recorded_amount'=>round($recorded,2),'pending_amount'=>round($pending,2),'pending_count'=>$count,'available_amount'=>max(0,round($due-$pending,2)), 'independent'=>$independent, 'ready'=>$count>0 && abs($pending-$due)<0.005];
    }

    public static function parse_amount($value): float
    {
        if (!is_scalar($value)) { return -1; }
        $value = SN_Helpers::to_english_nums((string)$value);
        $value = preg_replace('/[,٬\s]+/u','',$value);
        return preg_match('/^[0-9]+(?:\.[0-9]{1,2})?$/D',$value) ? (float)$value : -1;
    }

    public static function amount_error(float $amount, float $due, float $pending): string
    {
        if (!is_finite($amount) || $amount <= 0 || round($amount,2) !== $amount) { return 'مبلغ این تراکنش را به‌درستی وارد کنید'; }
        if ($amount - max(0,$due-$pending) > 0.005) { return 'مبلغ تراکنش از مانده قابل ثبت این مرحله بیشتر است؛ تراکنش‌های قبلی را بررسی کنید'; }
        return '';
    }
}
