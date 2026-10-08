<?php
if ( ! defined( 'ABSPATH' ) ) { exit; }

/** Read-only reporting queries. Every view, count and download uses query(). */
final class SN_Report_Service {
	private $db;
	private ?array $seller_ids;

	public function __construct( $db, ?array $seller_ids ) {
		$this->db = $db;
		$this->seller_ids = $seller_ids === null ? null : array_values( array_unique( array_filter( array_map( 'absint', $seller_ids ) ) ) );
	}

	public static function definitions(): array {
		return [
			'leads' => [ 'title' => 'لیدها', 'table' => 'sn_leads', 'alias' => 'l', 'date' => 'imported_at', 'seller' => 'seller_id', 'columns' => [ 'id' => 'شناسه', 'customer_name' => 'مشتری', 'phone' => 'موبایل', 'province' => 'استان', 'city' => 'شهر', 'status' => 'وضعیت تخصیص', 'lead_status' => 'وضعیت مشتری', 'seller_name' => 'فروشنده', 'imported_at' => 'دریافت', 'assigned_at' => 'تخصیص' ], 'amount' => null ],
			'invoices' => [ 'title' => 'فاکتورهای ایجادشده', 'table' => 'sn_invoices', 'alias' => 'i', 'date' => 'created_at', 'seller' => 'seller_id', 'columns' => [ 'id' => 'شناسه', 'invoice_code' => 'فاکتور', 'customer_name' => 'مشتری', 'customer_phone' => 'موبایل', 'seller_name' => 'فروشنده', 'status' => 'وضعیت', 'final_total' => 'مبلغ نهایی', 'created_at' => 'ایجاد', 'paid_at' => 'پرداخت', 'approved_at' => 'تأیید' ], 'amount' => 'COALESCE(i.final_total,i.product_price,0)' ],
			'paid_invoices' => [ 'title' => 'فاکتورهای دارای زمان پرداخت', 'table' => 'sn_invoices', 'alias' => 'i', 'date' => 'paid_at', 'seller' => 'seller_id', 'condition' => 'i.paid_at IS NOT NULL', 'columns' => [ 'id' => 'شناسه', 'invoice_code' => 'فاکتور', 'seller_name' => 'فروشنده', 'status' => 'وضعیت', 'final_total' => 'مبلغ نهایی', 'paid_at' => 'زمان پرداخت' ], 'amount' => 'COALESCE(i.final_total,i.product_price,0)' ],
			'approved_invoices' => [ 'title' => 'فاکتورهای دارای زمان تأیید', 'table' => 'sn_invoices', 'alias' => 'i', 'date' => 'approved_at', 'seller' => 'seller_id', 'condition' => 'i.approved_at IS NOT NULL', 'columns' => [ 'id' => 'شناسه', 'invoice_code' => 'فاکتور', 'seller_name' => 'فروشنده', 'status' => 'وضعیت', 'final_total' => 'مبلغ نهایی', 'approved_at' => 'زمان تأیید' ], 'amount' => 'COALESCE(i.final_total,i.product_price,0)' ],
			'payments' => [ 'title' => 'تراکنش‌ها', 'table' => 'sn_payments', 'alias' => 'p', 'date' => 'created_at', 'seller' => 'seller_id', 'join' => 'sn_invoices', 'columns' => [ 'id' => 'شناسه', 'invoice_id' => 'فاکتور', 'amount' => 'مبلغ', 'status' => 'وضعیت', 'pay_method' => 'روش', 'created_at' => 'ثبت' ], 'amount' => 'p.amount' ],
			'audit' => [ 'title' => 'رویدادهای ثبت‌شده', 'table' => 'sn_activity_logs', 'alias' => 'a', 'date' => 'created_at', 'seller' => 'seller_id', 'join' => 'sn_invoices', 'columns' => [ 'id' => 'شناسه', 'invoice_id' => 'فاکتور', 'lead_id' => 'لید', 'user_id' => 'کاربر', 'action' => 'عمل', 'created_at' => 'زمان' ], 'amount' => null ],
		];
	}

	public static function filters( array $input ): array {
		$scalar = static fn( $value ): string => is_scalar( $value ) ? (string) $value : '';
		$parse_day = static function ( $value ): string {
			$value = sanitize_text_field( wp_unslash( is_scalar( $value ) ? (string) $value : '' ) );
			if ( preg_match( '/^\d{4}\/\d{2}\/\d{2}$/', $value ) ) { $value = SN_Helpers::jalali_to_gregorian_date( $value ); }
			return preg_match( '/^\d{4}-\d{2}-\d{2}$/', (string) $value ) && checkdate( (int) substr( $value, 5, 2 ), (int) substr( $value, 8, 2 ), (int) substr( $value, 0, 4 ) ) ? $value : '';
		};
		$time = static function ( $value ): string { $value = sanitize_text_field( wp_unslash( is_scalar( $value ) ? (string) $value : '' ) ); return preg_match( '/^(?:[01]\d|2[0-3]):[0-5]\d$/', $value ) ? $value : ''; };
		return [ 'date_from' => $parse_day( $input['date_from'] ?? '' ), 'date_to' => $parse_day( $input['date_to'] ?? '' ), 'time_from' => $time( $input['time_from'] ?? '' ), 'time_to' => $time( $input['time_to'] ?? '' ), 'seller_id' => absint( $scalar( $input['seller_id'] ?? 0 ) ), 'status' => sanitize_text_field( wp_unslash( $scalar( $input['status'] ?? '' ) ) ), 'province' => sanitize_text_field( wp_unslash( $scalar( $input['province'] ?? '' ) ) ), 'city' => sanitize_text_field( wp_unslash( $scalar( $input['city'] ?? '' ) ) ), 'pay_method' => sanitize_key( wp_unslash( $scalar( $input['pay_method'] ?? '' ) ) ), 'product_id' => absint( $scalar( $input['product_id'] ?? 0 ) ) ];
	}

	private function sql( string $type, array $filters ): array {
		$def = self::definitions()[ $type ] ?? null;
		if ( ! $def ) { throw new InvalidArgumentException( 'Unknown report' ); }
		$a = $def['alias']; $table = $this->db->prefix . $def['table'];
		$join = '';
		if ( isset( $def['join'] ) ) { $join = " LEFT JOIN {$this->db->prefix}sn_invoices i ON i.id={$a}.invoice_id"; }
		$where = [ '1=1' ]; $args = [];
		if ( isset( $def['condition'] ) ) { $where[] = $def['condition']; }
		$seller = isset( $def['join'] ) ? 'i.seller_id' : "{$a}.seller_id";
		if ( $type === 'audit' ) {
			$join .= " LEFT JOIN {$this->db->prefix}sn_leads l ON l.id={$a}.lead_id";
			$seller = 'COALESCE(i.seller_id,l.seller_id)';
			$where[] = '(i.id IS NOT NULL OR l.id IS NOT NULL)';
		}
		if ( $type !== 'audit' ) { $join .= " LEFT JOIN {$this->db->users} u ON u.ID={$seller}"; }
		if ( $this->seller_ids !== null ) {
			if ( ! $this->seller_ids ) { $where[] = '1=0'; }
			else { $where[] = $seller . ' IN (' . implode( ',', array_fill( 0, count( $this->seller_ids ), '%d' ) ) . ')'; $args = array_merge( $args, $this->seller_ids ); }
		}
		if ( $filters['seller_id'] ) { $where[] = "{$seller}=%d"; $args[] = $filters['seller_id']; }
		$date = "{$a}.{$def['date']}";
		if ( $filters['date_from'] ) { $where[] = "{$date} >= %s"; $args[] = $filters['date_from'] . ' ' . ( $filters['time_from'] ?: '00:00' ) . ':00'; }
		if ( $filters['date_to'] ) { $where[] = "{$date} <= %s"; $args[] = $filters['date_to'] . ' ' . ( $filters['time_to'] ?: '23:59' ) . ':59'; }
		if ( $filters['status'] && $type !== 'audit' ) { $where[] = $type === 'leads' ? "COALESCE(NULLIF(l.lead_status,''),l.status)=%s" : "{$a}.status=%s"; $args[] = $filters['status']; }
		$invoice = in_array( $type, [ 'invoices', 'paid_invoices', 'approved_invoices' ], true );
		if ( $filters['province'] && ( $type === 'leads' || $invoice ) ) { $where[] = "{$a}.province=%s"; $args[] = $filters['province']; }
		if ( $filters['city'] && ( $type === 'leads' || $invoice ) ) { $where[] = "{$a}.city=%s"; $args[] = $filters['city']; }
		if ( $filters['pay_method'] && ( $invoice || $type === 'payments' ) ) { $where[] = "{$a}.pay_method=%s"; $args[] = $filters['pay_method']; }
		if ( $filters['product_id'] && $invoice ) { $where[] = 'i.product_id=%d'; $args[] = $filters['product_id']; }
		return [ " FROM {$table} {$a}{$join} WHERE " . implode( ' AND ', $where ), $args, $def ];
	}

	public function query( string $type, array $filters, int $page = 1, int $size = 50 ): array {
		[ $from, $args, $def ] = $this->sql( $type, $filters );
		$amount = $def['amount'] ? ",COALESCE(SUM({$def['amount']}),0) amount" : '';
		$stats = $this->db->get_row( $this->prepare( "SELECT COUNT(DISTINCT {$def['alias']}.id) total{$amount}{$from}", $args ), ARRAY_A );
		$page = max( 1, $page ); $size = max( 1, min( 500, $size ) );
		$select = implode( ',', array_map( static fn( $col ) => $col === 'seller_name' ? 'u.display_name AS seller_name' : "{$def['alias']}.{$col}", array_keys( $def['columns'] ) ) );
		$rows = $this->db->get_results( $this->prepare( "SELECT {$select}{$from} ORDER BY {$def['alias']}.id DESC LIMIT %d OFFSET %d", array_merge( $args, [ $size, ( $page - 1 ) * $size ] ) ), ARRAY_A );
		return [ 'title' => $def['title'], 'headers' => $def['columns'], 'total' => (int) ( $stats['total'] ?? 0 ), 'amount' => $def['amount'] ? (string) ( $stats['amount'] ?? '0' ) : null, 'rows' => $rows ?: [], 'page' => $page, 'page_size' => $size ];
	}

	private function prepare( string $sql, array $args ): string { return $args ? $this->db->prepare( $sql, ...$args ) : $sql; }
}
