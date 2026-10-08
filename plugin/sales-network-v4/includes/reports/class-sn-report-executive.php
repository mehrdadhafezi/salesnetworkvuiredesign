<?php
if ( ! defined( 'ABSPATH' ) ) { exit; }

/** Seven business questions, with one scoped query for cards, details and CSV. */
final class SN_Report_Executive {
	private $db;
	private ?array $seller_ids;
	private ?array $recipient_ids;
	private ?bool $invoice_has_issuer = null;

	public function __construct( $db, ?array $seller_ids, ?array $recipient_ids = null ) {
		$this->db = $db;
		$this->seller_ids = $seller_ids === null ? null : array_values( array_unique( array_filter( array_map( 'absint', $seller_ids ) ) ) );
		$this->recipient_ids = $recipient_ids === null ? null : array_values( array_unique( array_filter( array_map( 'absint', $recipient_ids ) ) ) );
	}

	/** Narrow a report to the selected hierarchy without widening the viewer scope. */
	public function under_person( array $seller_ids, array $recipient_ids ): self {
		$sellers = $this->seller_ids === null ? $seller_ids : array_intersect( $this->seller_ids, $seller_ids );
		$recipients = $this->recipient_ids === null ? $recipient_ids : array_intersect( $this->recipient_ids, $recipient_ids );
		return new self( $this->db, array_values( $sellers ), array_values( $recipients ) );
	}

	public static function definitions(): array {
		$invoice = [ 'invoice_code' => 'شماره پیش‌فاکتور', 'customer_name' => 'مشتری', 'customer_phone' => 'موبایل', 'issuer_name' => 'صادرکننده', 'seller_name' => 'فروشنده', 'amount' => 'مبلغ', 'status' => 'وضعیت', 'event_at' => 'زمان رویداد' ];
		return [
			'invoices_register' => [ 'title' => 'فاکتورها', 'headers' => [ 'invoice_code'=>'کد فاکتور', 'customer_name'=>'مشتری', 'customer_phone'=>'موبایل', 'product_label'=>'محصول', 'amount'=>'مبلغ', 'seller_name'=>'فروشنده', 'supervisor_name'=>'سرپرست', 'senior_supervisor_name'=>'سرپرست ارشد', 'sales_manager_name'=>'مدیر فروش', 'deputy_name'=>'معاونت', 'status'=>'وضعیت', 'pay_method'=>'روش پرداخت', 'payment_status'=>'وضعیت پرداخت', 'city'=>'شهر', 'province'=>'استان', 'event_at'=>'تاریخ صدور' ], 'note'=>'فهرست همهٔ فاکتورها در محدودهٔ دسترسی، بر اساس زمان صدور؛ برای دیدن مشتریانِ هر فاکتور به تب جزئیات پروفایل مراجعه کنید.' ],
			'customer_profiles' => [ 'title'=>'پروفایل مشتری‌ها', 'headers'=>[ 'customer_phone'=>'موبایل', 'customer_name'=>'مشتری', 'invoice_count'=>'تعداد فاکتور', 'amount'=>'جمع فاکتورها', 'paid_count'=>'تعداد پرداخت/تأیید', 'paid_amount'=>'فروش تأییدشده', 'invoice_code'=>'آخرین کد فاکتور', 'product_ids'=>'شناسه محصولات', 'seller_name'=>'فروشنده‌ها', 'supervisor_name'=>'سرپرست', 'senior_supervisor_name'=>'سرپرست ارشد', 'sales_manager_name'=>'مدیر فروش', 'deputy_name'=>'معاونت', 'event_at'=>'آخرین تاریخ' ], 'note'=>'هر مشتری در محدودهٔ انتخاب‌شده یک بار نمایش داده می‌شود؛ برای مشاهدهٔ فاکتورهای همان مشتری، تب جزئیات فاکتور مشتری را انتخاب کنید.' ],
			'customer_invoice_details' => [ 'title'=>'جزئیات فاکتور مشتری', 'headers'=>[ 'invoice_code'=>'کد فاکتور', 'customer_phone'=>'موبایل', 'customer_name'=>'مشتری', 'product_label'=>'محصول', 'amount'=>'مبلغ', 'seller_name'=>'فروشنده', 'status'=>'وضعیت', 'pay_method'=>'روش پرداخت', 'city'=>'شهر', 'province'=>'استان', 'event_at'=>'تاریخ صدور' ], 'note'=>'فاکتورها به تفکیک هر مشتری با امکان جستجوی شماره، نام یا کد فاکتور.' ],
			'mis_assignments' => [ 'title' => 'تخصیص دادهٔ MIS', 'headers' => [ 'record_id' => 'شناسه داده', 'batch_id' => 'پرونده MIS', 'actor' => 'تخصیص‌دهنده', 'recipient' => 'دریافت‌کننده', 'status' => 'نوع انتقال', 'event_at' => 'زمان تخصیص' ], 'note' => 'هر انتقال ثبت‌شده به یک گیرنده یک رویداد است؛ یک داده ممکن است در چند لایهٔ سازمانی منتقل شود.' ],
			'pre_invoices' => [ 'title' => 'پیش‌فاکتورهای صادرشده', 'headers' => $invoice, 'note' => 'شمارش بر اساس زمان ایجاد هر پیش‌فاکتور، مستقل از پرداخت.' ],
			'sales' => [ 'title' => 'فروش‌های تکمیل‌شده', 'headers' => $invoice, 'note' => 'فاکتورهای دارای وضعیت paid/approved و پرداخت کامل؛ تاریخ، زمان تکمیل پرداخت است.' ],
			'online_sales' => [ 'title' => 'فروش آنلاین', 'headers' => $invoice, 'note' => 'فاکتور دارای مرحلهٔ پرداخت آنلاین موفق؛ در هر بازه هر فاکتور یک بار و مبلغ مرحله‌های همان بازه جمع می‌شود.' ],
			'card_sales' => [ 'title' => 'فروش کارت‌به‌کارت تأییدشده', 'headers' => $invoice, 'note' => 'فاکتور دارای مرحلهٔ کارت‌به‌کارت با تأیید مالی؛ در هر بازه هر فاکتور یک بار شمرده می‌شود.' ],
			'finance_pending' => [ 'title' => 'پیش‌فاکتورهای در انتظار مالی', 'headers' => $invoice, 'note' => 'بدون تاریخ: صف فعلی. با بازه: ورودهای ثبت‌شده به صف. در حالت «وضعیت در لحظه»: تعداد موجود در صف در آن لحظه از زمان شروع ثبت تاریخچه.' ],
			'finance_rejected' => [ 'title' => 'پیش‌فاکتورهای ردشده', 'headers' => $invoice, 'note' => 'بدون تاریخ: موارد ردشدهٔ فعلی. با بازه: رویدادهای رد مالی. در حالت «وضعیت در لحظه»: موارد ردشده در آن لحظه از زمان شروع ثبت تاریخچه.' ],
		];
	}

	public static function filters( array $input ): array {
		$base = SN_Report_Service::filters( $input );
		$asof = SN_Report_Service::filters( [ 'date_from' => $input['asof_date'] ?? '', 'time_from' => $input['asof_time'] ?? '' ] );
		$base['asof_date'] = $asof['date_from'];
		$base['asof_time'] = $asof['time_from'];
		$base['mode'] = ( ( $input['sn_center_mode'] ?? $input['mode'] ?? '' ) === 'asof' ) ? 'asof' : 'period';
		$scalar = static fn( $value ): string => is_scalar( $value ) ? sanitize_text_field( wp_unslash( (string)$value ) ) : '';
		$base['report_search'] = $scalar( $input['sn_report_search'] ?? '' );
		$base['report_status'] = sanitize_key( $scalar( $input['sn_report_status'] ?? '' ) );
		return $base;
	}

	private function table_exists( string $name ): bool {
		$table = $this->db->prefix . $name;
		return $this->db->get_var( $this->db->prepare( 'SHOW TABLES LIKE %s', $table ) ) === $table;
	}

	private function timestamp( string $date, string $time ): string {
		if ( $date === '' ) { return ''; }
		$tehran = new DateTimeZone( 'Asia/Tehran' );
		$site = function_exists( 'wp_timezone' ) ? wp_timezone() : $tehran;
		return ( new DateTimeImmutable( $date . ' ' . $time . ':00', $tehran ) )->setTimezone( $site )->format( 'Y-m-d H:i:s' );
	}

	private function dates( string $column, array $f, array &$where, array &$args ): void {
		if ( $f['mode'] === 'asof' ) {
			$at = $this->timestamp( $f['asof_date'], $f['asof_time'] ?: '23:59' );
			if ( $at !== '' ) { $where[] = "{$column}<=%s"; $args[] = ( new DateTimeImmutable( $at ) )->modify( '+59 seconds' )->format( 'Y-m-d H:i:s' ); }
			return;
		}
		$from = $this->timestamp( $f['date_from'], $f['time_from'] ?: '00:00' );
		$to = $this->timestamp( $f['date_to'], $f['time_to'] ?: '23:59' );
		if ( $from !== '' ) { $where[] = "{$column}>=%s"; $args[] = $from; }
		if ( $to !== '' ) { $where[] = "{$column}<=%s"; $args[] = ( new DateTimeImmutable( $to ) )->modify( '+59 seconds' )->format( 'Y-m-d H:i:s' ); }
	}

	private function scope( string $column, ?array $ids, array $f, array &$where, array &$args ): void {
		if ( $ids !== null ) {
			if ( ! $ids ) { $where[] = '1=0'; }
			else { $where[] = "{$column} IN (" . implode( ',', array_fill( 0, count( $ids ), '%d' ) ) . ')'; $args = array_merge( $args, $ids ); }
		}
		if ( $f['seller_id'] ) { $where[] = "{$column}=%d"; $args[] = $f['seller_id']; }
	}

	private function prepared( string $sql, array $args ): string { return $args ? $this->db->prepare( $sql, ...$args ) : $sql; }

	/** Returns [SQL select with normalized columns, arguments, availability note]. */
	private function source( string $type, array $f ): array {
		$p = $this->db->prefix; $users = $this->db->users;
		$invoice = "{$p}sn_invoices"; $where = [ '1=1' ]; $args = [];
		$state = in_array( $type, [ 'finance_pending', 'finance_rejected' ], true );
		if ( in_array( $type, [ 'invoices_register', 'customer_profiles', 'customer_invoice_details' ], true ) ) {
			$this->scope( 'i.seller_id', $this->seller_ids, $f, $where, $args );
			$this->dates( 'i.created_at', $f, $where, $args );
			if ( $f['report_search'] !== '' ) {
				$like = '%' . $this->db->esc_like( $f['report_search'] ) . '%';
				$where[] = '(i.invoice_code LIKE %s OR i.customer_name LIKE %s OR i.customer_phone LIKE %s OR i.city LIKE %s OR i.province LIKE %s)';
				$args = array_merge( $args, array_fill( 0, 5, $like ) );
			}
			if ( in_array( $f['report_status'], [ 'pre_invoice', 'pending_financial_approval', 'paid', 'approved', 'rejected', 'cancelled' ], true ) ) { $where[] = 'i.status=%s'; $args[] = $f['report_status']; }
			if ( $type === 'customer_profiles' ) {
				$key = "COALESCE(NULLIF(TRIM(i.customer_phone),''),CONCAT('#',i.id))";
				$paid = "i.status IN ('paid','approved')";
				return [ "SELECT MIN(i.id) id,{$key} customer_phone,MAX(i.customer_name) customer_name,
				 COUNT(*) invoice_count,SUM(COALESCE(i.final_total,i.product_price,0)) amount,
				 SUM(CASE WHEN {$paid} THEN 1 ELSE 0 END) paid_count,
				 SUM(CASE WHEN {$paid} THEN COALESCE(i.final_total,i.product_price,0) ELSE 0 END) paid_amount,
				 SUBSTRING_INDEX(GROUP_CONCAT(i.invoice_code ORDER BY i.created_at DESC,i.id DESC SEPARATOR ','),',',1) invoice_code,
				 GROUP_CONCAT(DISTINCT i.product_id ORDER BY i.product_id SEPARATOR '، ') product_ids,
				 GROUP_CONCAT(DISTINCT i.seller_id ORDER BY i.seller_id SEPARATOR ',') seller_ids,
				 GROUP_CONCAT(DISTINCT u.display_name ORDER BY u.display_name SEPARATOR '، ') seller_name,
				 MAX(i.created_at) event_at FROM {$invoice} i LEFT JOIN {$users} u ON u.ID=i.seller_id
				 WHERE " . implode( ' AND ', $where ) . " GROUP BY {$key}", $args, '' ];
			}
			return [ "SELECT i.id,i.invoice_code,i.customer_name,i.customer_phone,i.product_id,i.seller_id,i.city,i.province,
			 COALESCE(u.display_name,CONCAT('#',i.seller_id)) seller_name,
			 COALESCE(i.final_total,i.product_price,0) amount,i.status,i.pay_method,i.payment_status,i.created_at event_at
			 FROM {$invoice} i LEFT JOIN {$users} u ON u.ID=i.seller_id WHERE " . implode( ' AND ', $where ), $args, '' ];
		}
		if ( $type === 'mis_assignments' ) {
			if ( ! $this->table_exists( 'sn_distribution_item_logs' ) || ! $this->table_exists( 'sn_distribution_items' ) ) { return [ '', [], 'جدول رویدادهای توزیع MIS موجود نیست.' ]; }
			$where[] = "d.source_type IN ('mis_pool','mis_data_row')";
			$where[] = "l.action IN ('distributed_forward','delivered_to_seller','mis_direct_delivery_to_seller','mis_quick_assignment_to_role')";
			$this->scope( 'l.to_user_id', $this->recipient_ids, $f, $where, $args );
			$this->dates( 'l.created_at', $f, $where, $args );
			$sql = "SELECT l.id,l.item_id record_id,d.batch_id,l.actor_user_id,l.to_user_id,
			 COALESCE(ua.display_name,CONCAT('#',l.actor_user_id)) actor,
			 COALESCE(ur.display_name,CONCAT('#',l.to_user_id)) recipient,
			 l.action status,l.created_at event_at,0 amount
			 FROM {$p}sn_distribution_item_logs l INNER JOIN {$p}sn_distribution_items d ON d.id=l.item_id
			 LEFT JOIN {$users} ua ON ua.ID=l.actor_user_id LEFT JOIN {$users} ur ON ur.ID=l.to_user_id
			 WHERE " . implode( ' AND ', $where );
			if ( $this->table_exists( 'sn_mis_data_rows' ) ) {
				$row_where = [ 'r.assigned_manager_user_id>0', 'r.assigned_at IS NOT NULL' ]; $row_args = [];
				$this->scope( 'r.assigned_manager_user_id', $this->recipient_ids, $f, $row_where, $row_args );
				$this->dates( 'r.assigned_at', $f, $row_where, $row_args );
				$row_where[] = "NOT EXISTS (SELECT 1 FROM {$p}sn_distribution_items dx INNER JOIN {$p}sn_distribution_item_logs lx ON lx.item_id=dx.id AND lx.action='mis_quick_assignment_to_role' WHERE dx.source_type='mis_data_row' AND dx.source_id=r.id AND lx.to_user_id=r.assigned_manager_user_id)";
				$sql .= " UNION ALL SELECT 1000000000000+r.id id,r.id record_id,r.batch_id,NULL actor_user_id,r.assigned_manager_user_id to_user_id,
				 'MIS' actor,COALESCE(ur.display_name,CONCAT('#',r.assigned_manager_user_id)) recipient,
				 'assigned_to_manager' status,r.assigned_at event_at,0 amount
				 FROM {$p}sn_mis_data_rows r LEFT JOIN {$users} ur ON ur.ID=r.assigned_manager_user_id
				 WHERE " . implode( ' AND ', $row_where );
				$args = array_merge( $args, $row_args );
			}
			return [ $sql, $args, '' ];
		}
		if ( $this->invoice_has_issuer === null ) { $this->invoice_has_issuer = (bool) $this->db->get_var( $this->db->prepare( "SHOW COLUMNS FROM {$invoice} LIKE %s", 'issued_by_user_id' ) ); }
		$issuer_id = $this->invoice_has_issuer ? 'COALESCE(NULLIF(i.issued_by_user_id,0),i.seller_id)' : 'i.seller_id';
		$status_expr = $state && ( $f['mode'] === 'asof' || $f['date_from'] || $f['date_to'] ) ? 'e.state' : 'i.status';
		$cols = "i.id,i.invoice_code,i.customer_name,i.customer_phone,i.seller_id,
		 COALESCE(issuer.display_name,CONCAT('#',{$issuer_id})) issuer_name,
		 COALESCE(u.display_name,CONCAT('#',i.seller_id)) seller_name,
		 COALESCE(i.final_total,i.product_price,0) amount,{$status_expr} status";
		if ( $state && $f['mode'] === 'asof' ) {
			$baseline = (string) get_option( 'sn_report_state_baseline_at', '' );
			$at = $this->timestamp( $f['asof_date'], $f['asof_time'] ?: '23:59' );
			if ( $at !== '' ) { $at = ( new DateTimeImmutable( $at ) )->modify( '+59 seconds' )->format( 'Y-m-d H:i:s' ); }
			if ( $at === '' || $baseline === '' || $at < $baseline || ! $this->table_exists( 'sn_report_invoice_state_events' ) ) { return [ '', [], 'وضعیت تاریخی پیش از شروع ثبت رویدادها یا بدون تاریخ و ساعت معتبر قابل بازسازی نیست.' ]; }
			$this->scope( 'i.seller_id', $this->seller_ids, $f, $where, $args );
			$where[] = 'e.state=%s'; $args[] = $type === 'finance_pending' ? 'pending_financial_approval' : 'rejected';
			$sql = "SELECT {$cols},e.created_at event_at FROM {$p}sn_report_invoice_state_events e
			 INNER JOIN (SELECT invoice_id,MAX(id) id FROM {$p}sn_report_invoice_state_events WHERE created_at<=%s GROUP BY invoice_id) last ON last.id=e.id
			 INNER JOIN {$invoice} i ON i.id=e.invoice_id LEFT JOIN {$users} u ON u.ID=i.seller_id
			 LEFT JOIN {$users} issuer ON issuer.ID={$issuer_id}
			 WHERE " . implode( ' AND ', $where );
			return [ $sql, array_merge( [ $at ], $args ), '' ];
		}
		if ( $state && ( $f['date_from'] || $f['date_to'] ) ) {
			if ( ! $this->table_exists( 'sn_report_invoice_state_events' ) ) { return [ '', [], 'جدول تاریخچهٔ مالی موجود نیست.' ]; }
			$baseline = (string) get_option( 'sn_report_state_baseline_at', '' );
			$until = $this->timestamp( $f['date_to'], $f['time_to'] ?: '23:59' );
			if ( $baseline === '' || ( $until !== '' && ( new DateTimeImmutable( $until ) )->modify( '+59 seconds' )->format( 'Y-m-d H:i:s' ) < $baseline ) ) { return [ '', [], 'رویدادهای مالی پیش از فعال‌سازی تاریخچه قابل شمارش نیستند.' ]; }
			$where[] = 'e.source_activity_id IS NOT NULL';
			$where[] = 'e.state=%s'; $args[] = $type === 'finance_pending' ? 'pending_financial_approval' : 'rejected';
			$this->scope( 'i.seller_id', $this->seller_ids, $f, $where, $args );
			$this->dates( 'e.created_at', $f, $where, $args );
			return [ "SELECT {$cols},e.created_at event_at FROM {$p}sn_report_invoice_state_events e INNER JOIN {$invoice} i ON i.id=e.invoice_id LEFT JOIN {$users} u ON u.ID=i.seller_id LEFT JOIN {$users} issuer ON issuer.ID={$issuer_id} WHERE " . implode( ' AND ', $where ), $args, '' ];
		}
		if ( $type === 'pre_invoices' || $type === 'sales' || $state ) {
			$this->scope( 'i.seller_id', $this->seller_ids, $f, $where, $args );
			if ( $type === 'sales' ) { $where[] = "i.status IN ('paid','approved') AND (i.payment_workflow_status='completed' OR i.paid_at IS NOT NULL)"; $date = 'COALESCE(i.payment_completed_at,i.paid_at)'; }
			elseif ( $state ) { $where[] = 'i.status=%s'; $args[] = $type === 'finance_pending' ? 'pending_financial_approval' : 'rejected'; $date = $type === 'finance_pending' ? 'i.updated_at' : 'COALESCE(i.financial_rejected_at,i.rejected_at)'; }
			else { $date = 'i.created_at'; }
			if ( ! $state ) { $this->dates( $date, $f, $where, $args ); }
			return [ "SELECT {$cols},{$date} event_at FROM {$invoice} i LEFT JOIN {$users} u ON u.ID=i.seller_id LEFT JOIN {$users} issuer ON issuer.ID={$issuer_id} WHERE " . implode( ' AND ', $where ), $args, '' ];
		}
		if ( ! $this->table_exists( 'sn_invoice_payment_stages' ) ) { return [ '', [], 'جدول مرحله‌های پرداخت موجود نیست.' ]; }
		$card = $type === 'card_sales';
		$where[] = $card ? "s.pay_method='card' AND s.status='approved' AND s.approved_at IS NOT NULL" : "s.pay_method='online' AND s.status IN ('paid','approved') AND s.paid_at IS NOT NULL";
		$this->scope( 'i.seller_id', $this->seller_ids, $f, $where, $args );
		$date = $card ? 's.approved_at' : 's.paid_at';
		$this->dates( $date, $f, $where, $args );
		$sql = "SELECT MIN(i.id) id,i.invoice_code,MAX(i.customer_name) customer_name,MAX(i.customer_phone) customer_phone,
		 i.seller_id,MAX(u.display_name) seller_name,MAX(issuer.display_name) issuer_name,SUM(s.requested_amount) amount,
		 MAX(i.status) status,MAX({$date}) event_at
		 FROM {$p}sn_invoice_payment_stages s INNER JOIN {$invoice} i ON i.id=s.invoice_id
		 LEFT JOIN {$users} u ON u.ID=i.seller_id LEFT JOIN {$users} issuer ON issuer.ID={$issuer_id} WHERE " . implode( ' AND ', $where ) . ' GROUP BY i.id,i.invoice_code,i.seller_id';
		return [ $sql, $args, '' ];
	}

	public function query( string $type, array $filters, int $page = 1, int $size = 50, bool $include_breakdown = true ): array {
		$def = self::definitions()[$type] ?? null;
		if ( ! $def ) { throw new InvalidArgumentException( 'Unknown report' ); }
		[ $source, $args, $error ] = $this->source( $type, $filters );
		$page = max( 1, $page ); $size = max( 1, min( 500, $size ) );
		$result = [ 'title' => $def['title'], 'headers' => $def['headers'], 'note' => $error ?: $def['note'], 'available' => $error === '', 'total' => 0, 'amount' => null, 'rows' => [], 'breakdown' => [], 'page' => $page, 'page_size' => $size ];
		if ( ! $error && in_array( $type, [ 'finance_pending', 'finance_rejected' ], true ) && $filters['mode'] === 'period' && $filters['date_from'] ) {
			$baseline = (string) get_option( 'sn_report_state_baseline_at', '' );
			if ( $baseline && $this->timestamp( $filters['date_from'], $filters['time_from'] ?: '00:00' ) < $baseline ) { $result['note'] .= ' بخش پیش از فعال‌سازی تاریخچه در این شمارش موجود نیست.'; }
		}
		if ( $filters['mode'] === 'asof' && $filters['asof_date'] === '' ) { $result['available'] = false; $result['note'] = 'برای وضعیت در لحظه، تاریخ مشخص کنید.'; return $result; }
		if ( $error ) { return $result; }
		$base = $this->prepared( $source, $args );
		$stats = $this->db->get_row( "SELECT COUNT(*) total,COALESCE(SUM(q.amount),0) amount FROM ({$base}) q", ARRAY_A );
		if ( $this->db->last_error ) { $result['available'] = false; $result['note'] = 'پرس‌وجوی گزارش در این نصب اجرا نشد؛ ساختار جدول‌ها باید بررسی شود.'; return $result; }
		$result['total'] = (int) ( $stats['total'] ?? 0 );
		$result['amount'] = in_array( $type, [ 'mis_assignments', 'finance_pending', 'finance_rejected' ], true ) ? null : (string) ( $stats['amount'] ?? '0' );
		if ( $size > 1 ) {
			$result['rows'] = $this->db->get_results( $this->db->prepare( "SELECT * FROM ({$base}) q ORDER BY q.event_at DESC,q.id DESC LIMIT %d OFFSET %d", $size, ( $page - 1 ) * $size ), ARRAY_A ) ?: [];
			if ( $this->db->last_error ) { $result['available'] = false; $result['note'] = 'ریز داده‌ها در این نصب قابل خواندن نیستند.'; return $result; }
			if ( $include_breakdown && $type !== 'customer_profiles' ) {
				$person = $type === 'mis_assignments' ? 'recipient' : ( $type === 'pre_invoices' ? 'issuer_name' : 'seller_name' );
				$result['breakdown'] = $this->db->get_results( "SELECT q.{$person} person,COUNT(*) total,COALESCE(SUM(q.amount),0) amount FROM ({$base}) q GROUP BY q.{$person} ORDER BY total DESC", ARRAY_A ) ?: [];
				if ( $this->db->last_error ) { $result['available'] = false; $result['note'] = 'تفکیک افراد در این نصب قابل خواندن نیست.'; }
			}
		}
		return $result;
	}
}
