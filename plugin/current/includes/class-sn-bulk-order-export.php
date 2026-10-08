<?php
if (! defined('ABSPATH')) { exit; }

/** SpreadsheetML export matching the supplied bulk_order_template.xls. */
class SN_Bulk_Order_Export
{
    public const HEADERS = [
        'شناسه فاکتور', 'موبایل مشتری بدون صفر', 'تلفن', 'نام مشتری', 'استان', 'شهر', 'آدرس', 'کد پستی',
        'نام کالا', 'تعداد', 'تخفیف کالا', 'نوع پرداخت', 'پرداخت نقدی', 'پرداخت چکی', 'هزینه ارسال',
        'پیش واریزی', 'تاریخ پیش واریزی', 'ساعت پیش واریزی', 'دقیقه پیش واریزی', 'کد پیگیری پیش واریزی',
        'شناسه حساب بانکی', 'تاریخ ارسال', 'زمان ارسال', 'جنسیت', 'فوری', 'توضیحات', 'مدل محصول',
    ];

    private static function xml(string $value): string
    {
        // XML 1.0 rejects control characters; all customer input remains literal text.
        $value = wp_check_invalid_utf8($value, true);
        $value = preg_replace('/[^\x{9}\x{A}\x{D}\x{20}-\x{D7FF}\x{E000}-\x{FFFD}\x{10000}-\x{10FFFF}]/u', '', $value);
        return htmlspecialchars($value, ENT_XML1 | ENT_QUOTES, 'UTF-8');
    }

    public static function row_xml(array $values, bool $header = false): string
    {
        if (count($values) !== count(self::HEADERS)) { throw new RuntimeException('Invalid bulk export column count'); }
        $xml = $header ? '<Row ss:StyleID="header">' : '<Row>';
        foreach (array_values($values) as $index => $value) {
            // Identifiers, phone numbers and postal codes must keep their digits exactly.
            $type = ! $header && in_array($index, [9,10,12,13,14,15], true) && $value !== '' && is_numeric($value) ? 'Number' : 'String';
            $xml .= '<Cell><Data ss:Type="' . $type . '">' . self::xml((string) $value) . '</Data></Cell>';
        }
        return $xml . '</Row>';
    }

    private static function date_parts(string $value): array
    {
        if ($value === '' || strpos($value, '0000-00-00') === 0) { return ['', '', '']; }
        $formatted = SN_Helpers::to_english_nums(SN_Helpers::gregorian_to_jalali_date($value));
        if (! preg_match('~^(\d{4}/\d{1,2}/\d{1,2})(?:\s+(\d{1,2}):(\d{2}))?~', $formatted, $m)) { return ['', '', '']; }
        return [$m[1], $m[2] ?? '', $m[3] ?? ''];
    }

    public static function values(array $invoice, array $item, array $shipping, array $deposit, bool $first): array
    {
        $phone = SN_Helpers::normalize_mobile((string) ($invoice['customer_phone'] ?? ''));
        if (preg_match('/^09\d{9}$/', $phone)) { $phone = substr($phone, 1); }
        $method = (string) ($invoice['pay_method'] ?? '');
        $methods = ['online'=>'آنلاین','card'=>'کارت‌به‌کارت','card_to_card'=>'کارت‌به‌کارت','receipt'=>'کارت‌به‌کارت','cash'=>'نقدی','cheque'=>'چکی','check'=>'چکی'];
        // Deposit means the first confirmed partial stage, never a pending receipt or the total invoice price.
        $deposit = $first ? $deposit : [];
        [$date, $hour, $minute] = self::date_parts((string) ($deposit['paid_at'] ?? ''));
        [$ship_date, $ship_hour, $ship_minute] = self::date_parts((string) ($shipping['shipped_at'] ?? ''));
        return [
            (string) $invoice['id'], $phone, (string) ($invoice['customer_phone_secondary'] ?? ''),
            (string) ($invoice['customer_name'] ?? ''),
            (string) (($shipping['province'] ?? '') ?: ($invoice['province'] ?? '')),
            (string) (($shipping['city'] ?? '') ?: ($invoice['city'] ?? '')),
            (string) (($shipping['address'] ?? '') ?: ($invoice['customer_address'] ?? '')),
            (string) (($shipping['postal_code'] ?? '') ?: ($invoice['customer_postal_code'] ?? '')),
            (string) ($item['product_name'] ?? ''), (int) ($item['qty'] ?? 1), '', $methods[$method] ?? $method,
            '', '', '', $deposit['requested_amount'] ?? '', $date, $hour, $minute,
            (string) ($deposit['payment_ref_id'] ?? ''), '', $ship_date,
            $ship_hour !== '' ? $ship_hour . ':' . $ship_minute : '', '', '',
            (string) ($shipping['expert_note'] ?? ''), '',
        ];
    }

    private static function query(string $sql, array $args = []): array
    {
        global $wpdb;
        $rows = $wpdb->get_results($args ? $wpdb->prepare($sql, ...$args) : $sql, ARRAY_A);
        if ($wpdb->last_error !== '') { throw new RuntimeException('Bulk export database read failed: ' . $wpdb->last_error); }
        return (array) $rows;
    }

    private static function write($stream, string $text): void
    {
        if (fwrite($stream, $text) !== strlen($text)) { throw new RuntimeException('Bulk export temporary file write failed'); }
    }

    public static function download(string $where, array $args, array $selected): void
    {
        global $wpdb;
        $stream = tmpfile();
        if ($stream === false) { wp_die('فضای موقت برای ساخت خروجی در دسترس نیست.'); }
        try {
            if ($selected) {
                $where .= ' AND id IN (' . implode(',', array_fill(0, count($selected), '%d')) . ')';
                $args = array_merge($args, $selected);
            }
            $tables = [];
            foreach (['sn_invoice_items','sn_invoice_payment_stages','sn_shipping_orders'] as $suffix) {
                $table = $wpdb->prefix . $suffix;
                $tables[$suffix] = (string) $wpdb->get_var($wpdb->prepare('SHOW TABLES LIKE %s', $wpdb->esc_like($table))) === $table;
                if ($wpdb->last_error !== '') { throw new RuntimeException('Bulk export schema read failed'); }
            }
            self::write($stream, "\xEF\xBB\xBF" . '<?xml version="1.0" encoding="UTF-8"?><?mso-application progid="Excel.Sheet"?>');
            self::write($stream, '<Workbook xmlns="urn:schemas-microsoft-com:office:spreadsheet" xmlns:ss="urn:schemas-microsoft-com:office:spreadsheet"><Styles><Style ss:ID="Default" ss:Name="Normal"><Alignment ss:Horizontal="Center" ss:Vertical="Center" ss:ReadingOrder="RightToLeft"/><Font ss:FontName="Tahoma"/></Style><Style ss:ID="header"><Font ss:FontName="Tahoma" ss:Bold="1"/></Style></Styles><Worksheet ss:Name="Sheet1" ss:RightToLeft="1"><Table>');
            self::write($stream, self::row_xml(self::HEADERS, true));
            $last_id = 0;
            $row_count = 0;
            do {
                $batch = self::query("SELECT * FROM {$wpdb->prefix}sn_invoices WHERE ({$where}) AND id > %d ORDER BY id ASC LIMIT 200", array_merge($args, [$last_id]));
                if (! $batch) { break; }
                $ids = array_map('intval', array_column($batch, 'id'));
                $in = implode(',', $ids);
                $items = $shipping = $deposits = [];
                if ($tables['sn_invoice_items']) {
                    foreach (self::query("SELECT * FROM {$wpdb->prefix}sn_invoice_items WHERE invoice_id IN ({$in}) ORDER BY id ASC") as $item) { $items[(int) $item['invoice_id']][] = $item; }
                }
                if ($tables['sn_shipping_orders']) {
                    foreach (self::query("SELECT * FROM {$wpdb->prefix}sn_shipping_orders WHERE invoice_id IN ({$in})") as $ship) { $shipping[(int) $ship['invoice_id']] = $ship; }
                }
                if ($tables['sn_invoice_payment_stages']) {
                    foreach (self::query("SELECT * FROM {$wpdb->prefix}sn_invoice_payment_stages WHERE invoice_id IN ({$in}) AND stage_no=1 AND stage_type='partial' AND status IN ('paid','approved')") as $stage) { $deposits[(int) $stage['invoice_id']] = $stage; }
                }
                foreach ($batch as $invoice) {
                    $id = (int) $invoice['id'];
                    $lines = $items[$id] ?? [['product_name'=>! empty($invoice['product_id']) ? get_the_title((int) $invoice['product_id']) : '', 'qty'=>1]];
                    foreach ($lines as $index => $item) {
                        if (++$row_count > 65535) { throw new LengthException('برای خروجی XLS حداکثر ۶۵۵۳۵ ردیف کالا مجاز است؛ بازهٔ فیلتر را کوچک‌تر کنید.'); }
                        self::write($stream, self::row_xml(self::values($invoice, $item, $shipping[$id] ?? [], $deposits[$id] ?? [], $index === 0)));
                    }
                    $last_id = $id;
                }
            } while (count($batch) === 200);
            if ($row_count === 0) { throw new LengthException('فاکتوری مطابق فیلترها و انتخاب شما پیدا نشد.'); }
            self::write($stream, '</Table><WorksheetOptions xmlns="urn:schemas-microsoft-com:office:excel"><DisplayRightToLeft/><ProtectObjects>False</ProtectObjects><ProtectScenarios>False</ProtectScenarios></WorksheetOptions></Worksheet></Workbook>');
        } catch (Throwable $e) {
            fclose($stream);
            error_log('SN bulk order export: ' . $e->getMessage());
            wp_die($e instanceof LengthException ? esc_html($e->getMessage()) : 'ساخت خروجی کامل نشد؛ جزئیات در گزارش خطای سرور ثبت شد.');
            return;
        }
        rewind($stream);
        while (ob_get_level()) { if (! ob_end_clean()) { break; } }
        nocache_headers();
        header('Content-Type: application/vnd.ms-excel; charset=UTF-8');
        header('Content-Disposition: attachment; filename="bulk-orders-' . gmdate('Y-m-d-His') . '.xls"');
        fpassthru($stream);
        fclose($stream);
        exit;
    }
}
