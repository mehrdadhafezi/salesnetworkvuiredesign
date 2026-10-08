<?php
if ( ! defined( 'ABSPATH' ) ) { exit; }

/** Public subscription purchase, using the CRM invoice and paid-membership hooks. */
final class SN_Subscription_Form {
	public static function register(): void {
		add_action( 'add_meta_boxes_product', [ __CLASS__, 'metabox' ] );
		add_action( 'save_post_product', [ __CLASS__, 'save' ], 35, 2 );
		add_shortcode( 'sn_subscription_form', [ __CLASS__, 'render' ] );
		add_action( 'admin_post_nopriv_sn_subscription_form_pay', [ __CLASS__, 'submit' ] );
		add_action( 'admin_post_sn_subscription_form_pay', [ __CLASS__, 'submit' ] );
	}

	public static function metabox(): void {
		add_meta_box( 'sn-subscription-form', 'فروش فرمی اشتراک', [ __CLASS__, 'fields' ], 'product', 'side' );
	}

	public static function fields( WP_Post $post ): void {
		$supervisor = absint( get_post_meta( $post->ID, '_sn_subscription_form_supervisor', true ) );
		wp_nonce_field( 'sn_subscription_form_product', 'sn_subscription_form_nonce' );
		?>
		<p>برای اشتراک تعریف‌شده در تب پروژه‌ها، سرپرست فروش این فرم را انتخاب کنید.</p>
		<label>سرپرست فروش<br><select name="sn_subscription_form_supervisor" style="width:100%"><option value="0">غیرفعال</option>
		<?php foreach ( get_users( [ 'role__in' => [ 'sn_supervisor', 'administrator' ], 'orderby' => 'display_name', 'number' => 300 ] ) as $user ) : ?>
		<option value="<?php echo esc_attr( (string) $user->ID ); ?>" <?php selected( $supervisor, (int) $user->ID ); ?>><?php echo esc_html( $user->display_name ); ?> (#<?php echo (int) $user->ID; ?>)</option>
		<?php endforeach; ?></select></label>
		<p>شورتکد: <code>[sn_subscription_form product_id="<?php echo (int) $post->ID; ?>"]</code></p>
		<?php
	}

	public static function save( int $post_id, WP_Post $post ): void {
		if ( ! isset( $_POST['sn_subscription_form_nonce'] ) || ! wp_verify_nonce( sanitize_text_field( wp_unslash( $_POST['sn_subscription_form_nonce'] ) ), 'sn_subscription_form_product' ) || ! current_user_can( 'edit_post', $post_id ) || ( defined( 'DOING_AUTOSAVE' ) && DOING_AUTOSAVE ) ) { return; }
		$supervisor = absint( $_POST['sn_subscription_form_supervisor'] ?? 0 );
		$user = $supervisor ? get_user_by( 'id', $supervisor ) : null;
		update_post_meta( $post_id, '_sn_subscription_form_supervisor', $user instanceof WP_User && array_intersect( [ 'sn_supervisor', 'administrator' ], (array) $user->roles ) ? $supervisor : 0 );
	}

	private static function product( int $id ) {
		if ( $id < 1 || ! function_exists( 'wc_get_product' ) || ! class_exists( 'SN_Projects' ) || ! SN_Projects::instance()->is_subscription_product( $id ) || get_post_status( $id ) !== 'publish' || (string) get_post_meta( $id, '_sn_enabled', true ) !== '1' || ! absint( get_post_meta( $id, '_sn_subscription_form_supervisor', true ) ) ) { return null; }
		$product = wc_get_product( $id );
		return $product && (float) $product->get_price() > 0 ? $product : null;
	}

	private static function seller_for_supervisor( int $supervisor ): int {
		if ( $supervisor < 1 || ! get_user_by( 'id', $supervisor ) ) { return 0; }
		$ids = get_users( [ 'fields'=>'ids', 'number'=>1, 'meta_query'=>[ 'relation'=>'AND', [ 'key'=>'_sn_subscription_form_seller', 'value'=>'1' ], [ 'key'=>'_sn_subscription_form_supervisor', 'value'=>(string)$supervisor ] ] ] );
		if ( $ids ) { return absint( $ids[0] ); }
		$login = 'sn_subscription_form_' . $supervisor;
		$index = 2;
		while ( username_exists( $login ) ) { $login = 'sn_subscription_form_' . $supervisor . '_' . $index++; }
		$user_id = wp_insert_user( [ 'user_login'=>$login, 'user_pass'=>wp_generate_password( 40, true, true ), 'display_name'=>'فروش فرمی اشتراک', 'role'=>'sn_seller' ] );
		if ( is_wp_error( $user_id ) ) { return 0; }
		update_user_meta( $user_id, '_sn_subscription_form_seller', '1' );
		update_user_meta( $user_id, '_sn_subscription_form_supervisor', $supervisor );
		update_user_meta( $user_id, 'sn_supervisor_id', $supervisor );
		return (int) $user_id;
	}

	public static function render( $atts = [] ): string {
		$id = absint( is_array( $atts ) ? ( $atts['product_id'] ?? 0 ) : 0 );
		$product = self::product( $id );
		if ( ! $product ) { return '<p dir="rtl">فرم خرید این اشتراک فعال نیست.</p>'; }
		$error = sanitize_key( (string) ( $_GET['sn_subscription_error'] ?? '' ) );
		$token = wp_generate_password( 48, false, false );
		set_transient( 'sn_subscription_token_' . hash( 'sha256', $token ), [ 'product_id' => $id ], HOUR_IN_SECONDS );
		$messages = [ 'invalid'=>'اطلاعات فرم معتبر نیست.', 'unavailable'=>'اشتراک در دسترس نیست.', 'invoice'=>'صدور فاکتور انجام نشد. دوباره تلاش کنید.', 'limit'=>'لطفاً چند دقیقه دیگر تلاش کنید.' ];
		ob_start(); ?>
		<form dir="rtl" method="post" action="<?php echo esc_url( admin_url( 'admin-post.php' ) ); ?>" class="sn-dot-marketing-wrap dot-credit-form-wrap" style="max-width:650px;margin:auto">
			<h3><?php echo esc_html( $product->get_name() ); ?></h3><p>مبلغ اشتراک: <strong><?php echo esc_html( SN_Helpers::format_price( (float) $product->get_price() ) ); ?></strong></p>
			<?php if ( isset( $messages[ $error ] ) ) : ?><p role="alert"><?php echo esc_html( $messages[ $error ] ); ?></p><?php endif; ?>
			<input type="hidden" name="action" value="sn_subscription_form_pay"><input type="hidden" name="product_id" value="<?php echo (int) $id; ?>"><input type="hidden" name="form_token" value="<?php echo esc_attr( $token ); ?>"><?php wp_nonce_field( 'sn_subscription_form_' . $id ); ?>
			<p style="display:none"><label>وب‌سایت<input name="website" tabindex="-1" autocomplete="off"></label></p>
			<p><label>نام و نام خانوادگی <input name="customer_name" required maxlength="120" autocomplete="name"></label></p>
			<p><label>شماره موبایل <input name="customer_phone" type="tel" required inputmode="numeric" maxlength="16" autocomplete="tel"></label></p>
			<p><label>کد ملی <input name="national_id" required inputmode="numeric" maxlength="10" pattern="[0-9۰-۹٠-٩]{10}"></label></p>
			<p><label>استان <input name="province" required maxlength="80"></label></p><p><label>شهر <input name="city" required maxlength="80"></label></p>
			<button type="submit">ثبت اطلاعات و پرداخت اشتراک</button>
		</form><?php return (string) ob_get_clean();
	}

	public static function submit(): void {
		$id = absint( $_POST['product_id'] ?? 0 );
		$ref = wp_get_referer(); $back = $ref && wp_validate_redirect( $ref, false ) ? $ref : home_url( '/' );
		$fail = static function ( string $reason ) use ( $back ): void { wp_safe_redirect( add_query_arg( 'sn_subscription_error', $reason, $back ) ); exit; };
		if ( ! wp_verify_nonce( sanitize_text_field( wp_unslash( $_POST['_wpnonce'] ?? '' ) ), 'sn_subscription_form_' . $id ) || trim( (string) wp_unslash( $_POST['website'] ?? '' ) ) !== '' ) { $fail( 'invalid' ); }
		if ( ! self::product( $id ) ) { $fail( 'unavailable' ); }
		$name = sanitize_text_field( wp_unslash( $_POST['customer_name'] ?? '' ) );
		$phone = SN_Helpers::normalize_mobile( sanitize_text_field( wp_unslash( $_POST['customer_phone'] ?? '' ) ) );
		$national_id = preg_replace( '/\D+/', '', SN_Helpers::to_english_nums( (string) wp_unslash( $_POST['national_id'] ?? '' ) ) );
		$province = sanitize_text_field( wp_unslash( $_POST['province'] ?? '' ) ); $city = sanitize_text_field( wp_unslash( $_POST['city'] ?? '' ) );
		if ( $name === '' || ! SN_Helpers::is_valid_mobile( $phone ) || strlen( $national_id ) !== 10 || $province === '' || $city === '' ) { $fail( 'invalid' ); }
		$token = sanitize_text_field( wp_unslash( $_POST['form_token'] ?? '' ) );
		if ( ! preg_match( '/^[A-Za-z0-9]{48}$/', $token ) ) { $fail( 'invalid' ); }
		$key = 'sn_subscription_token_' . hash( 'sha256', $token );
		$existing = get_transient( $key );
		if ( ! is_array( $existing ) || (int) ( $existing['product_id'] ?? 0 ) !== $id ) { $fail( 'invalid' ); }
		if ( is_array( $existing ) && ! empty( $existing['invoice_code'] ) && ! empty( $existing['access_token'] ) ) { self::invoice_redirect( $existing ); }
		$ip_key = 'sn_subscription_ip_' . md5( (string) ( $_SERVER['REMOTE_ADDR'] ?? '' ) );
		if ( get_transient( $ip_key ) ) { $fail( 'limit' ); }
		set_transient( $ip_key, 1, MINUTE_IN_SECONDS );
		$claim = 'sn_subscription_claim_' . hash( 'sha256', $token );
		$claimed_at = (int) get_option( $claim, 0 );
		if ( $claimed_at > 0 && $claimed_at + 300 < time() ) { delete_option( $claim ); }
		if ( ! add_option( $claim, (string) time(), '', false ) ) { $fail( 'limit' ); }
		$supervisor = absint( get_post_meta( $id, '_sn_subscription_form_supervisor', true ) );
		$seller = self::seller_for_supervisor( $supervisor );
		if ( ! $seller || ! function_exists( 'sn_bootstrap_plugin_instance' ) ) { delete_option( $claim ); $fail( 'invoice' ); }
		$result = sn_bootstrap_plugin_instance( false )->create_dot_marketing_source_invoice( [ 'customer_name'=>$name, 'customer_phone'=>$phone, 'national_id'=>$national_id, 'province'=>$province, 'city'=>$city ], $id, $seller, $supervisor, true );
		if ( empty( $result['success'] ) || empty( $result['data']['invoice_code'] ) || empty( $result['data']['access_token'] ) ) { delete_option( $claim ); $fail( 'invoice' ); }
		set_transient( $key, array_merge( $result['data'], [ 'product_id' => $id ] ), HOUR_IN_SECONDS );
		delete_option( $claim );
		self::invoice_redirect( $result['data'] );
	}

	private static function invoice_redirect( array $invoice ): void {
		$page_id = absint( get_option( 'sn_invoice_page_id', 0 ) );
		$base = $page_id ? get_permalink( $page_id ) : home_url( '/invoice/' );
		wp_safe_redirect( add_query_arg( [ 'invoice'=>(string)$invoice['invoice_code'], 'access_token'=>(string)$invoice['access_token'] ], $base ) ); exit;
	}
}
