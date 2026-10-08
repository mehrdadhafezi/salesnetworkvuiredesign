<?php
/** Explicit, resumable refresh of product financial snapshots on existing cards. */
if ( ! defined( 'ABSPATH' ) ) { exit; }

final class SN_Product_Card_Sync {
	private static $instance;
	private const BATCH_SIZE = 25;
	private const TTL = 7200;
	public static function instance(): self { return self::$instance ?? ( self::$instance = new self() ); }
	public function register_hooks(): void {
		add_action( 'admin_enqueue_scripts', [ $this, 'enqueue' ] );
		add_action( 'wp_ajax_sn_product_card_sync_preview', [ $this, 'ajax_preview' ] );
		add_action( 'wp_ajax_sn_product_card_sync_apply', [ $this, 'ajax_apply' ] );
	}
	public function enqueue(): void {
		$screen = get_current_screen();
		if ( $screen && $screen->id === 'product' && current_user_can( 'manage_options' ) ) {
			wp_enqueue_script( 'sn-product-card-sync', SN_PLUGIN_URL . 'assets/js/product-card-sync.js', [], SN_VERSION, true );
		}
	}
	public function render_controls( int $product_id ): void {
		if ( ! current_user_can( 'manage_options' ) ) { return; }
		?>
		<div data-sn-product-card-sync data-product-id="<?php echo esc_attr( (string) $product_id ); ?>" data-nonce="<?php echo esc_attr( wp_create_nonce( 'sn_product_card_sync_' . $product_id ) ); ?>" data-url="<?php echo esc_url( admin_url( 'admin-ajax.php' ) ); ?>" style="border:1px solid #c3c4c7;border-radius:6px;padding:10px;margin:12px 0;background:#f6f7f7">
			<strong>اعمال تنظیمات روی کارت‌های قبلی</strong>
			<p style="font-size:12px">ابتدا محصول را ذخیره کنید، سپس فیلدهای مورد نظر را انتخاب کنید. این ابزار کارت همین محصول را در اشتراک‌های قبلی و خریدهای مستقل به‌روزرسانی می‌کند.</p>
			<label style="display:block;margin:6px 0"><input type="checkbox" data-sync-field value="credit" checked> اعتبار قابل ارائه به مشتری (تومان)</label>
			<label style="display:block;margin:6px 0"><input type="checkbox" data-sync-field value="upgrade_options"> حالت‌های افزایشی و هزینه ارتقا</label>
			<label style="display:block;margin:8px 0">دامنه اشتراک‌ها <select data-sync-scope style="width:100%"><option value="active">اشتراک‌های فعال</option><option value="all">همه اشتراک‌ها، شامل تعلیق‌شده</option></select></label>
			<p style="font-size:11px">کارت دارای فاکتور یا ارتقای ثبت‌شده، کد فعال‌سازی یا پرونده اجرا تغییر نمی‌کند و در گزارش مشخص می‌شود. لینک، توضیحات و نمایش گزینه‌های افزایشی از تنظیمات فعلی محصول خوانده می‌شوند.</p>
			<button type="button" class="button" data-sync-preview>پیش‌نمایش</button>
			<button type="button" class="button button-primary" data-sync-apply disabled>اعمال روی کارت‌های قبلی</button>
			<button type="button" class="button" data-sync-pause hidden>توقف بعد از این مرحله</button>
			<div data-sync-report role="status" aria-live="polite" style="font-size:12px;white-space:pre-line;margin-top:8px"></div>
		</div>
		<?php
	}
	private function table( string $key ): string {
		global $wpdb;
		$names = [ 'items'=>'sn_project_membership_items', 'members'=>'sn_project_memberships', 'ops'=>'sn_project_operations', 'actions'=>'sn_project_actions', 'events'=>'sn_project_events', 'cases'=>'sn_operations_execution_cases', 'codes'=>'sn_operations_wallet_codes' ];
		return $wpdb->prefix . $names[$key];
	}
	private function authorize(): int {
		$id = absint( $_POST['product_id'] ?? 0 );
		if ( ! $id || ! current_user_can( 'manage_options' ) || ! current_user_can( 'edit_post', $id ) || get_post_type( $id ) !== 'product' ) {
			wp_send_json_error( [ 'message'=>'اجازه تغییر کارت‌های این محصول را ندارید.' ], 403 );
		}
		check_ajax_referer( 'sn_product_card_sync_' . $id, 'nonce' );
		return $id;
	}
	private function schema(): void {
		global $wpdb;
		foreach ( [ 'items','members','ops','actions','events','cases','codes' ] as $key ) {
			$status = $wpdb->get_row( $wpdb->prepare( 'SHOW TABLE STATUS WHERE Name=%s', $this->table( $key ) ), ARRAY_A );
			if ( ! $status || strtolower( (string) ( $status['Engine'] ?? '' ) ) !== 'innodb' ) {
				throw new RuntimeException( 'جدول‌های چرخه آماده نیستند یا ذخیره اتمی را پشتیبانی نمی‌کنند؛ همگام‌سازی انجام نشد.' );
			}
		}
		$index = $wpdb->get_var( "SHOW INDEX FROM {$this->table('items')} WHERE Key_name='content_product_cursor'" );
		if ( ! $index && false === $wpdb->query( "ALTER TABLE {$this->table('items')} ADD INDEX content_product_cursor (content_product_id,id)" ) ) {
			throw new RuntimeException( 'آماده‌سازی نمایه کارت‌ها ناموفق بود؛ همگام‌سازی شروع نشد.' );
		}
	}
	private function config( int $id ): array {
		global $wpdb;
		$credit = round( max( 0, (float) get_post_meta( $id, '_sn_product_credit_amount', true ) ), 2 );
		$wpdb->last_error = '';
		$options = SN_Projects::instance()->product_upgrade_rules( $id );
		if ( $wpdb->last_error !== '' ) { throw new RuntimeException( 'خواندن تنظیمات افزایشی محصول ناموفق بود.' ); }
		return [ 'credit'=>$credit, 'upgrade_options'=>$options ];
	}
	private function fingerprint( array $config ): string { return hash( 'sha256', wp_json_encode( $config ) ); }
	private function key( string $token ): string { return 'sn_card_sync_' . $token; }
	private function persist( string $token, array $job ): void {
		set_transient( $this->key( $token ), $job, self::TTL );
		if ( get_transient( $this->key( $token ) ) !== $job ) { throw new RuntimeException( 'ذخیره پیشرفت ممکن نشد؛ پیش‌نمایش را دوباره بگیرید. تغییرات ثبت‌شده در گزارش کارت محفوظ است.' ); }
	}
	private function reason_sql(): string {
		$actions = $this->table( 'actions' ); $cases = $this->table( 'cases' ); $codes = $this->table( 'codes' );
		return "CASE
			WHEN mi.workflow_status NOT IN ('waiting','no_answer','follow_up') THEN 'closed_card'
			WHEN COALESCE(o.upgrade_invoice_id,0)>0 OR COALESCE(o.upgrade_action_id,0)>0 OR COALESCE(o.activation_mode,'')='upsell' OR COALESCE(o.payment_state,'')<>'' THEN 'upgrade_or_payment'
			WHEN EXISTS(SELECT 1 FROM {$cases} c WHERE c.membership_item_id=mi.id) OR EXISTS(SELECT 1 FROM {$codes} c WHERE c.membership_item_id=mi.id) THEN 'execution'
			WHEN o.id IS NOT NULL AND o.stage NOT IN ('awaiting_customer','sales_manager_queue','sales_supervisor','sales_expert') THEN 'execution'
			WHEN EXISTS(SELECT 1 FROM {$actions} a WHERE a.membership_item_id=mi.id AND a.status NOT IN ('cancelled','cancelled_continue_normal','rejected','failed','remote_failed')) THEN 'recorded_action'
			WHEN o.id IS NOT NULL AND ABS(COALESCE(o.current_credit,0)-COALESCE(o.base_credit,0))>0.005 THEN 'custom_credit'
			ELSE 'eligible' END";
	}
	private function from_sql(): string {
		return ' FROM ' . $this->table( 'items' ) . ' mi INNER JOIN ' . $this->table( 'members' ) . ' m ON m.id=mi.membership_id LEFT JOIN ' . $this->table( 'ops' ) . ' o ON o.membership_item_id=mi.id ';
	}
	private function where_sql( array $job ): string {
		global $wpdb;
		return $wpdb->prepare( 'mi.content_product_id=%d AND mi.id<=%d', $job['product_id'], $job['max_id'] ) . ( $job['scope'] === 'active' ? " AND m.status='active'" : '' );
	}
	private function results( string $sql ): array {
		global $wpdb; $wpdb->last_error = '';
		$rows = $wpdb->get_results( $sql, ARRAY_A );
		if ( $wpdb->last_error !== '' || ! is_array( $rows ) ) { throw new RuntimeException( 'خواندن کارت‌های قبلی ناموفق بود؛ هیچ تغییر تأییدنشده‌ای ثبت نشد.' ); }
		return $rows;
	}
	public function ajax_preview(): void {
		$id = $this->authorize();
		$response = null; $error = '';
		try {
			$this->schema();
			$fields = array_values( array_intersect( [ 'credit','upgrade_options' ], (array) wp_unslash( $_POST['fields'] ?? [] ) ) );
			if ( ! $fields ) { throw new RuntimeException( 'حداقل یک فیلد را انتخاب کنید.' ); }
			global $wpdb;
			$config = $this->config( $id );
			$max = $this->results( $wpdb->prepare( 'SELECT COALESCE(MAX(id),0) max_id FROM ' . $this->table( 'items' ) . ' WHERE content_product_id=%d', $id ) );
			$job = [ 'product_id'=>$id, 'user_id'=>get_current_user_id(), 'scope'=>( $_POST['scope'] ?? '' ) === 'all' ? 'all' : 'active', 'fields'=>$fields, 'config'=>$config, 'fingerprint'=>$this->fingerprint( $config ), 'max_id'=>(int) $max[0]['max_id'], 'cursor'=>0, 'scanned'=>0, 'changed'=>0, 'unchanged'=>0, 'skipped'=>0, 'reasons'=>[], 'done'=>false ];
			$counts = $this->results( 'SELECT ' . $this->reason_sql() . ' reason,COUNT(*) total' . $this->from_sql() . ' WHERE ' . $this->where_sql( $job ) . ' GROUP BY reason' );
			$job['total'] = array_sum( array_map( static function( $row ) { return (int) $row['total']; }, $counts ) );
			$samples = $this->results( 'SELECT mi.id,mi.membership_id,mi.base_credit_snapshot,' . $this->reason_sql() . ' reason' . $this->from_sql() . ' WHERE ' . $this->where_sql( $job ) . ' ORDER BY mi.id LIMIT 5' );
			$token = str_replace( '-', '', wp_generate_uuid4() );
			$this->persist( $token, $job );
			$response = [ 'token'=>$token, 'counts'=>$counts, 'samples'=>$samples, 'total'=>$job['total'], 'credit'=>$config['credit'], 'fields'=>$fields, 'cursor'=>0 ];
		} catch ( Throwable $e ) { $error = $e->getMessage(); }
		if ( $error !== '' ) { wp_send_json_error( [ 'message'=>$error ], 400 ); }
		wp_send_json_success( $response );
	}
	/** Called under the per-product job lock; cursor replay returns the committed response. */
	private function apply_batch( string $token, int $product_id, int $cursor ): array {
		global $wpdb;
		$job = get_transient( $this->key( $token ) );
		if ( ! is_array( $job ) || $job['product_id'] !== $product_id || $job['user_id'] !== get_current_user_id() ) { throw new RuntimeException( 'پیش‌نمایش منقضی شده یا متعلق به شما نیست؛ دوباره پیش‌نمایش بگیرید.' ); }
		if ( isset( $job['last_cursor'] ) && $job['last_cursor'] === $cursor ) { return $job['last_response']; }
		if ( $job['cursor'] !== $cursor || $job['done'] ) { throw new RuntimeException( 'پیشرفت تغییر کرده است؛ دوباره پیش‌نمایش بگیرید.' ); }
		if ( $this->fingerprint( $this->config( $product_id ) ) !== $job['fingerprint'] ) { throw new RuntimeException( 'تنظیمات ذخیره‌شده محصول تغییر کرده؛ برای ادامه پیش‌نمایش تازه بگیرید.' ); }
		$this->schema();
		$rows = $this->results( 'SELECT mi.id' . $this->from_sql() . ' WHERE ' . $this->where_sql( $job ) . ' AND mi.id>' . (int) $cursor . ' ORDER BY mi.id LIMIT ' . self::BATCH_SIZE );
		if ( false === $wpdb->query( 'START TRANSACTION' ) ) { throw new RuntimeException( 'شروع ذخیره تغییرات ممکن نشد.' ); }
		try {
			foreach ( $rows as $candidate ) {
				$id = (int) $candidate['id'];
				// Match the lock order of upgrade/code handlers. A missing operation's
				// unique-key gap is locked too, so it cannot be seeded with stale credit.
				$this->results( $wpdb->prepare( 'SELECT id FROM ' . $this->table( 'ops' ) . ' WHERE membership_item_id=%d FOR UPDATE', $id ) );
				$this->results( $wpdb->prepare( 'SELECT id FROM ' . $this->table( 'items' ) . ' WHERE id=%d FOR UPDATE', $id ) );
				$current = $this->results( 'SELECT mi.*,m.status membership_status,m.contents_snapshot_json,o.id operation_id,o.base_credit operation_base_credit,o.current_credit operation_current_credit,' . $this->reason_sql() . ' reason' . $this->from_sql() . ' WHERE mi.id=' . $id . ' FOR UPDATE' );
				$job['cursor'] = $id; $job['scanned']++;
				if ( ! $current ) { $this->skip( $job, 'missing_card' ); continue; }
				$row = $current[0];
				if ( $job['scope'] === 'active' && $row['membership_status'] !== 'active' ) { $this->skip( $job, 'inactive' ); continue; }
				if ( $row['reason'] !== 'eligible' ) { $this->skip( $job, $row['reason'] ); continue; }
				// Read again after the parent lock: other product jobs may have just
				// refreshed a different card inside this same subscription JSON.
				$member = $this->results( 'SELECT contents_snapshot_json,status FROM ' . $this->table( 'members' ) . ' WHERE id=' . (int) $row['membership_id'] . ' FOR UPDATE' );
				if ( $job['scope'] === 'active' && $member[0]['status'] !== 'active' ) { $this->skip( $job, 'inactive' ); continue; }
				$raw = (string) ( $member[0]['contents_snapshot_json'] ?? '' );
				$contents = trim( $raw ) === '' ? [] : json_decode( $raw, true );
				if ( ! is_array( $contents ) ) { $this->skip( $job, 'invalid_snapshot' ); continue; }
				$changes = $this->changes( $row, $job );
				$parent_changed = false;
				foreach ( $contents as &$content ) {
					if ( ! is_array( $content ) || (int) ( $content['content_product_id'] ?? 0 ) !== $product_id ) { continue; }
					foreach ( $job['fields'] as $field ) {
						$key = $field === 'credit' ? 'base_credit' : 'upgrade_options'; $value = $job['config'][$field];
						if ( ! array_key_exists( $key, $content ) || ! $this->equal( $content[$key], $value ) ) { $content[$key] = $value; $parent_changed = true; }
					}
				} unset( $content );
				if ( ! $changes['item'] && ! $changes['op'] && ! $parent_changed ) { $job['unchanged']++; continue; }
				$now = current_time( 'mysql' );
				if ( $changes['item'] ) { $this->update( 'items', $changes['item'] + [ 'updated_at'=>$now ], [ 'id'=>$id ] ); }
				if ( $changes['op'] ) { $this->update( 'ops', $changes['op'] + [ 'updated_at'=>$now ], [ 'id'=>(int) $row['operation_id'] ] ); }
				if ( $parent_changed ) { $this->update( 'members', [ 'contents_snapshot_json'=>wp_json_encode( $contents, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ), 'updated_at'=>$now ], [ 'id'=>(int) $row['membership_id'] ] ); }
				$details = [ 'job'=>$token, 'product_id'=>$product_id, 'fields'=>$job['fields'], 'before'=>[ 'item_credit'=>$row['base_credit_snapshot'], 'operation_base_credit'=>$row['operation_base_credit'], 'operation_current_credit'=>$row['operation_current_credit'], 'upgrade_options'=>json_decode( (string) $row['upgrade_options_snapshot_json'], true ) ], 'after'=>$job['config'] ];
				if ( false === $wpdb->insert( $this->table( 'events' ), [ 'membership_id'=>(int) $row['membership_id'], 'membership_item_id'=>$id, 'actor_user_id'=>get_current_user_id(), 'event_key'=>'product_card_settings_synced', 'details'=>wp_json_encode( $details, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES ), 'created_at'=>$now ] ) ) { throw new RuntimeException( 'ثبت گزارش کارت ناموفق بود؛ تغییرات این مرحله برگشت داده شد.' ); }
				$job['changed']++;
			}
			if ( false === $wpdb->query( 'COMMIT' ) ) { throw new RuntimeException( 'تأیید ذخیره مرحله ناموفق بود؛ پیش‌نمایش تازه بگیرید.' ); }
		} catch ( Throwable $e ) { $wpdb->query( 'ROLLBACK' ); throw $e; }
		$job['done'] = count( $rows ) < self::BATCH_SIZE || $job['cursor'] >= $job['max_id'];
		$response = array_intersect_key( $job, array_flip( [ 'cursor','total','scanned','changed','unchanged','skipped','reasons','done' ] ) );
		$job['last_cursor'] = $cursor; $job['last_response'] = $response;
		$this->persist( $token, $job );
		return $response;
	}
	private function equal( $a, $b ): bool {
		if ( is_numeric( $a ) && is_numeric( $b ) ) { return abs( (float) $a - (float) $b ) < 0.005; }
		return wp_json_encode( $a, JSON_NUMERIC_CHECK ) === wp_json_encode( $b, JSON_NUMERIC_CHECK );
	}
	private function changes( array $row, array $job ): array {
		$item = []; $op = [];
		if ( in_array( 'credit', $job['fields'], true ) ) {
			$credit = $job['config']['credit'];
			if ( $row['base_credit_snapshot'] === null || ! $this->equal( $row['base_credit_snapshot'], $credit ) ) { $item['base_credit_snapshot'] = $credit; }
			if ( $row['operation_id'] ) {
				if ( ! $this->equal( $row['operation_base_credit'], $credit ) ) { $op['base_credit'] = $credit; }
				if ( ! $this->equal( $row['operation_current_credit'], $credit ) ) { $op['current_credit'] = $credit; }
			}
		}
		if ( in_array( 'upgrade_options', $job['fields'], true ) && ! $this->equal( json_decode( (string) $row['upgrade_options_snapshot_json'], true ), $job['config']['upgrade_options'] ) ) {
			$item['upgrade_options_snapshot_json'] = wp_json_encode( $job['config']['upgrade_options'], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES );
		}
		return [ 'item'=>$item, 'op'=>$op ];
	}
	private function update( string $table, array $data, array $where ): void {
		global $wpdb;
		if ( false === $wpdb->update( $this->table( $table ), $data, $where ) ) { throw new RuntimeException( 'ذخیره کارت ناموفق بود؛ تغییرات این مرحله برگشت داده شد.' ); }
	}
	private function skip( array &$job, string $reason ): void { $job['skipped']++; $job['reasons'][$reason] = ( $job['reasons'][$reason] ?? 0 ) + 1; }
	public function ajax_apply(): void {
		$id = $this->authorize();
		$token = sanitize_key( (string) ( $_POST['token'] ?? '' ) );
		if ( ! preg_match( '/^[a-f0-9]{32}$/', $token ) ) { wp_send_json_error( [ 'message'=>'شناسه پیش‌نمایش نامعتبر است.' ], 400 ); }
		global $wpdb; $lock = 'sn_card_sync_' . substr( hash( 'sha256', $wpdb->prefix . ':' . $id ), 0, 40 );
		$response = null; $error = ''; $acquired = false;
		try {
			$acquired = (int) $wpdb->get_var( $wpdb->prepare( 'SELECT GET_LOCK(%s,1)', $lock ) ) === 1;
			if ( ! $acquired ) { throw new RuntimeException( 'همگام‌سازی دیگری برای این محصول در حال اجراست؛ چند لحظه بعد ادامه دهید.' ); }
			$response = $this->apply_batch( $token, $id, absint( $_POST['cursor'] ?? 0 ) );
		} catch ( Throwable $e ) { $error = $e->getMessage(); }
		finally { if ( $acquired ) { $wpdb->get_var( $wpdb->prepare( 'SELECT RELEASE_LOCK(%s)', $lock ) ); } }
		if ( $error !== '' ) { wp_send_json_error( [ 'message'=>$error ], 400 ); }
		wp_send_json_success( $response );
	}
}
