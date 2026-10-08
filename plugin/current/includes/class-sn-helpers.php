<?php
if (! defined('ABSPATH')) {
	exit;
}

class SN_Helpers
{

	/** نرمال‌سازی کد فاکتور برای جلوگیری از خطای اعداد فارسی/فاصله/کاراکترهای نامرئی */
	public static function normalize_invoice_code(string $code): string
	{
		$code = self::to_english_nums($code);
		$code = trim($code);
		$code = preg_replace('/[\x{200C}\x{200D}\x{200E}\x{200F}\x{202A}-\x{202E}]/u', '', $code);
		$code = preg_replace('/\s+/u', '', (string) $code);
		return strtoupper((string) $code);
	}

	/** تولید کد فاکتور یکتا؛ از این نسخه به بعد تصادفی و غیرترتیبی است. */
	public static function generate_invoice_code(): string
	{
		for ($i = 0; $i < 500; $i++) {
			$code = (string) wp_rand(20000000, 99999999);
			if (! self::invoice_code_exists($code)) {
				return $code;
			}
		}
		throw new RuntimeException('invoice_code_random_pool_exhausted');
	}

	public static function invoice_code_exists(string $code): bool
	{
		global $wpdb;
		$code = self::normalize_invoice_code($code);
		return (bool) $wpdb->get_var($wpdb->prepare(
			"SELECT id FROM {$wpdb->prefix}sn_invoices WHERE UPPER(invoice_code) = %s LIMIT 1",
			$code
		));
	}

	public static function generate_access_token(): string
	{
		if (function_exists('random_bytes')) {
			return rtrim(strtr(base64_encode(random_bytes(32)), '+/', '-_'), '=');
		}
		return wp_generate_password(48, false, false);
	}

	public static function invoice_access_token_exists(string $token): bool
	{
		global $wpdb;
		return (bool) $wpdb->get_var($wpdb->prepare(
			"SELECT id FROM {$wpdb->prefix}sn_invoices WHERE access_token = %s LIMIT 1",
			$token
		));
	}

	public static function generate_unique_access_token(): string
	{
		do {
			$token = self::generate_access_token();
		} while (self::invoice_access_token_exists($token));
		return $token;
	}

	/** فرمت قیمت ریال */
	public static function format_price(float $price): string
	{
		return number_format($price, 0, '.', ',') . ' تومان';
	}

	/** برچسب فارسی وضعیت‌های فاکتور و پرداخت */
	public static function status_label(string $status): string
	{
		$map = [
			'pre_invoice' => 'پیش‌فاکتور',
			'pending' => 'در انتظار پرداخت',
			'pending_payment' => 'در انتظار پرداخت',
			'receipt_uploaded' => 'نیاز به بررسی فیش',
			'pending_financial_approval' => 'نیاز به بررسی فیش',
			'paid' => 'پرداخت‌شده درگاهی',
			'approved' => 'تایید شده مالی',
			'rejected' => 'رد شده',
			'cancelled' => 'لغوشده',
			'assigned' => 'تخصیص داده‌شده',
			'unassigned' => 'بدون تخصیص',
			'supervisor_pool' => 'در پنل سرپرست',
			'invoiced' => 'پیش‌فاکتور صادر شده',
			'follow_up' => 'پیگیری مجدد',
			'no_answer' => 'عدم پاسخگویی',
			'interested' => 'علاقه‌مند',
			'not_interested' => 'عدم تمایل',
			'converted' => 'تبدیل شده',
			'card' => 'کارت به کارت',
			'card_to_card' => 'کارت به کارت',
			'online' => 'پرداخت آنلاین',
			'gateway' => 'درگاه پرداخت',
			'customer_upload' => 'ثبت توسط مشتری',
			'supervisor_upload' => 'ثبت توسط سرپرست',
			'recontact_requested' => 'ارتباط مجدد با کارشناس',
			'partial_paid' => 'پیش‌پرداخت تایید شده',
			'awaiting_next_payment' => 'در انتظار صدور مرحله بعد',
			'payment_stage_issued' => 'مرحله پرداخت صادر شده',
			'payment_archived' => 'بایگانی انصرافی',
		];
		return $map[$status] ?? ($status !== '' ? $status : '—');
	}

	/** برچسب فارسی روش پرداخت */
	public static function pay_method_label(string $method): string
	{
		$map = [
			'card' => 'کارت به کارت',
			'card_to_card' => 'کارت به کارت',
			'online' => 'پرداخت آنلاین',
			'gateway' => 'درگاه پرداخت',
		];
		return $map[$method] ?? ($method !== '' ? $method : '—');
	}

	/** برچسب فارسی منبع ثبت فیش */
	public static function payment_source_label(string $source): string
	{
		$map = [
			'customer_upload' => 'ثبت توسط مشتری',
			'supervisor_upload' => 'ثبت توسط سرپرست',
			'supervisor_manual' => 'اطلاعات واریز توسط سرپرست',
			'seller_manual' => 'اطلاعات واریز توسط فروشنده',
			'seller_upload' => 'فیش توسط فروشنده',
			'converter_manual' => 'اطلاعات واریز توسط تبدیل‌کننده',
			'converter_upload' => 'فیش توسط تبدیل‌کننده',
			'recontact_requested' => 'ارتباط مجدد با کارشناس',
			'admin_upload' => 'ثبت توسط ادمین',
			'gateway' => 'درگاه پرداخت',
		];
		return $map[$source] ?? ($source !== '' ? $source : '—');
	}

	public static function payment_status_label(string $status): string
	{
		return self::status_label($status);
	}

	public static function invoice_status_label(string $status): string
	{
		return self::status_label($status);
	}

	public static function source_label(string $source): string
	{
		return self::payment_source_label($source);
	}

	public static function role_label(string $role): string
	{
		$map = [
			'administrator' => 'مدیر سایت',
			'sn_seller' => 'فروشنده',
			'sn_supervisor' => 'سرپرست فروش',
			'sn_financial_approval' => 'تایید مالی',
			'sn_financial' => 'تایید مالی',
			'sn_after_sales' => 'خدمات پس از فروش',
			'sn_sales_manager' => 'مدیر فروش',
			'sn_operations_sales_manager' => 'مدیر فروش عملیات',
			'sn_operations_sales_supervisor' => 'سرپرست فروش عملیات',
			'sn_operations_sales_expert' => 'کارشناس فروش عملیات',
			'sn_operations_executive_manager' => 'مدیر اجرایی عملیات',
			'customer' => 'مشتری',
			'subscriber' => 'مشترک',
		];
		return $map[$role] ?? ($role !== '' ? $role : '—');
	}

	/** تبدیل اعداد فارسی به انگلیسی */
	public static function to_english_nums(string $str): string
	{
		$persian = ['۰', '۱', '۲', '۳', '۴', '۵', '۶', '۷', '۸', '۹'];
		$arabic  = ['٠', '١', '٢', '٣', '٤', '٥', '٦', '٧', '٨', '٩'];
		$english = ['0', '1', '2', '3', '4', '5', '6', '7', '8', '9'];
		$str = str_replace($persian, $english, $str);
		$str = str_replace($arabic,  $english, $str);
		return $str;
	}

	public static function jalali_to_gregorian_date(string $date): string
	{
		$date = trim(self::to_english_nums($date));
		$date = str_replace(['-', '.', ' '], '/', $date);
		if (! preg_match('/^(\d{4})\/(\d{1,2})\/(\d{1,2})$/', $date, $m)) {
			return preg_match('/^\d{4}-\d{2}-\d{2}$/', $date) ? $date : '';
		}
		$jy = (int) $m[1];
		$jm = (int) $m[2];
		$jd = (int) $m[3];
		if ($jy > 1700) {
			return checkdate($jm, $jd, $jy) ? sprintf('%04d-%02d-%02d', $jy, $jm, $jd) : '';
		}
		if ($jy < 1300 || $jy > 1600 || $jm < 1 || $jm > 12 || $jd < 1 || $jd > ($jm <= 6 ? 31 : 30)) {
			return '';
		}
		$normalized_jalali = sprintf('%04d/%02d/%02d', $jy, $jm, $jd);
		$jy += 1595;
		$days = -355668 + (365 * $jy) + ((int) floor($jy / 33) * 8) + (int) floor((($jy % 33) + 3) / 4) + $jd + ($jm < 7 ? ($jm - 1) * 31 : (($jm - 7) * 30) + 186);
		$gy = 400 * (int) floor($days / 146097);
		$days %= 146097;
		if ($days > 36524) {
			$gy += 100 * (int) floor(--$days / 36524);
			$days %= 36524;
			if ($days >= 365) {
				$days++;
			}
		}
		$gy += 4 * (int) floor($days / 1461);
		$days %= 1461;
		if ($days > 365) {
			$gy += (int) floor(($days - 1) / 365);
			$days = ($days - 1) % 365;
		}
		$gd = $days + 1;
		$sal_a = [0, 31, (($gy % 4 === 0 && $gy % 100 !== 0) || ($gy % 400 === 0)) ? 29 : 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];
		for ($gm = 1; $gm <= 12 && $gd > $sal_a[$gm]; $gm++) {
			$gd -= $sal_a[$gm];
		}
		$result = sprintf('%04d-%02d-%02d', $gy, $gm, $gd);
		return self::gregorian_to_jalali_input_value($result) === $normalized_jalali ? $result : '';
	}


	public static function gregorian_to_jalali_input_value(?string $datetime): string
	{
		$datetime = trim((string) $datetime);
		if ($datetime === '' || $datetime === '0000-00-00' || $datetime === '0000-00-00 00:00:00') {
			return '';
		}
		$converted = self::gregorian_to_jalali_date($datetime);
		if ($converted === '—' || $converted === '') {
			return '';
		}
		return substr($converted, 0, 10);
	}

	/**
	 * Business time zone used by every CRM panel.
	 *
	 * Database values are still stored in the WordPress time zone for backward
	 * compatibility; only parsing/display is normalized to Tehran.
	 */
	public static function tehran_timezone(): DateTimeZone
	{
		static $timezone = null;
		if (! $timezone instanceof DateTimeZone) {
			$timezone = new DateTimeZone('Asia/Tehran');
		}
		return $timezone;
	}

	/** Current instant represented explicitly in the CRM business timezone. */
	public static function tehran_now(): DateTimeImmutable
	{
		return (new DateTimeImmutable('@' . time()))->setTimezone(self::tehran_timezone());
	}

	/**
	 * Format an epoch in Tehran without wp_date filters.
	 *
	 * Persian-date plugins are allowed to filter wp_date(); using DateTime here
	 * keeps SQL keys, export names and comparisons Gregorian and deterministic.
	 */
	public static function tehran_format(string $format, ?int $timestamp = null): string
	{
		$timestamp = $timestamp ?? time();
		return (new DateTimeImmutable('@' . $timestamp))->setTimezone(self::tehran_timezone())->format($format);
	}

	/** Convert an epoch to the legacy WordPress-local MySQL storage format. */
	public static function site_mysql_from_timestamp(int $timestamp): string
	{
		$timezone = function_exists('wp_timezone') ? wp_timezone() : self::tehran_timezone();
		return (new DateTimeImmutable('@' . $timestamp))->setTimezone($timezone)->format('Y-m-d H:i:s');
	}

	/** Parse a legacy WordPress-local MySQL value without depending on server TZ. */
	public static function site_mysql_timestamp(?string $mysql, int $fallback = 0): int
	{
		$mysql = trim((string) $mysql);
		if ($mysql === '' || $mysql === '0000-00-00 00:00:00') {
			return $fallback;
		}
		try {
			$timezone = function_exists('wp_timezone') ? wp_timezone() : self::tehran_timezone();
			$date = DateTimeImmutable::createFromFormat('!Y-m-d H:i:s', $mysql, $timezone);
			$errors = DateTimeImmutable::getLastErrors();
			if (! $date || (is_array($errors) && ((int) $errors['warning_count'] > 0 || (int) $errors['error_count'] > 0))) {
				$date = new DateTimeImmutable($mysql, $timezone);
			}
			return $date->getTimestamp();
		} catch (Throwable $e) {
			return $fallback;
		}
	}

	/** Gregorian business date for SQL comparisons; presentation stays Jalali. */
	public static function tehran_today_gregorian(): string
	{
		return self::tehran_format('Y-m-d');
	}

	/**
	 * Validate a Jalali date plus Tehran wall-clock time and return legacy DB time.
	 * Existing rows remain in the WordPress timezone, so this performs a safe
	 * Tehran -> WordPress conversion instead of changing the storage contract.
	 *
	 * @return array|WP_Error
	 */
	public static function normalize_jalali_tehran_datetime(string $date_raw, string $time_raw = '09:00', bool $require_future = false, bool $allow_past = false)
	{
		$date_raw = trim(self::to_english_nums(wp_unslash($date_raw)));
		$date_raw = str_replace(['-', '.'], '/', $date_raw);
		$time_raw = trim(self::to_english_nums(wp_unslash($time_raw)));
		if (! preg_match('/^(\d{4})\/(\d{1,2})\/(\d{1,2})$/', $date_raw, $parts)) {
			return new WP_Error('sn_jalali_format', 'تاریخ شمسی معتبر نیست.');
		}
		$jalali = sprintf('%04d/%02d/%02d', (int) $parts[1], (int) $parts[2], (int) $parts[3]);
		$gregorian = self::jalali_to_gregorian_date($jalali);
		if ($gregorian === '' || self::gregorian_to_jalali_input_value($gregorian) !== $jalali) {
			return new WP_Error('sn_jalali_day', 'روز انتخاب‌شده در تقویم شمسی معتبر نیست.');
		}
		if (! preg_match('/^(?:[01]\d|2[0-3]):[0-5]\d$/', $time_raw)) {
			return new WP_Error('sn_tehran_time', 'ساعت تهران معتبر نیست.');
		}
		$normalized = $gregorian . ' ' . $time_raw;
		$date = DateTimeImmutable::createFromFormat('!Y-m-d H:i', $normalized, self::tehran_timezone());
		$errors = DateTimeImmutable::getLastErrors();
		if (! $date || (is_array($errors) && ((int) $errors['warning_count'] > 0 || (int) $errors['error_count'] > 0)) || $date->format('Y-m-d H:i') !== $normalized) {
			return new WP_Error('sn_tehran_datetime', 'تاریخ و ساعت تهران معتبر نیست.');
		}
		$today = self::tehran_today_gregorian();
		if ((! $allow_past && $gregorian < $today) || ($require_future && $date->getTimestamp() <= time() + 30)) {
			return new WP_Error('sn_tehran_datetime_past', $require_future ? 'زمان تماس مجدد باید در آینده باشد.' : 'تاریخ تماس نمی‌تواند گذشته باشد.');
		}
		$storage_timezone = function_exists('wp_timezone') ? wp_timezone() : self::tehran_timezone();
		return [
			'mysql' => $date->setTimezone($storage_timezone)->format('Y-m-d H:i:s'),
			'timestamp' => $date->getTimestamp(),
			'jalali' => $jalali,
			'time' => $time_raw,
		];
	}

	public static function gregorian_to_jalali_date(?string $datetime): string
	{
		$datetime = trim((string) $datetime);
		if ($datetime === '' || $datetime === '0000-00-00' || $datetime === '0000-00-00 00:00:00') {
			return '—';
		}
		if (preg_match('/^(13|14)\d{2}[\/\-.]\d{1,2}[\/\-.]\d{1,2}(?:\s+\d{1,2}:\d{2})?$/', self::to_english_nums($datetime))) {
			return str_replace(['-', '.'], '/', $datetime);
		}
		$has_time = (bool) preg_match('/\d{1,2}:\d{2}/', $datetime);
		try {
			$source_timezone = function_exists('wp_timezone') ? wp_timezone() : new DateTimeZone('UTC');
			$date = new DateTimeImmutable($datetime, $source_timezone);
			if ($has_time) {
				$date = $date->setTimezone(self::tehran_timezone());
			}
		} catch (Throwable $e) {
			return $datetime;
		}
		$gy = (int) $date->format('Y');
		$gm = (int) $date->format('n');
		$gd = (int) $date->format('j');
		$g_d_m = [0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334];
		$gy2 = $gm > 2 ? $gy + 1 : $gy;
		$days = 355666 + (365 * $gy) + (int) floor(($gy2 + 3) / 4) - (int) floor(($gy2 + 99) / 100) + (int) floor(($gy2 + 399) / 400) + $gd + $g_d_m[$gm - 1];
		$jy = -1595 + (33 * (int) floor($days / 12053));
		$days %= 12053;
		$jy += 4 * (int) floor($days / 1461);
		$days %= 1461;
		if ($days > 365) {
			$jy += (int) floor(($days - 1) / 365);
			$days = ($days - 1) % 365;
		}
		$jm = $days < 186 ? 1 + (int) floor($days / 31) : 7 + (int) floor(($days - 186) / 30);
		$jd = 1 + ($days < 186 ? $days % 31 : ($days - 186) % 30);
		return sprintf('%04d/%02d/%02d', $jy, $jm, $jd) . ($has_time ? ' ' . $date->format('H:i') : '');
	}

	/** اعتبارسنجی شماره موبایل ایران */
	public static function is_valid_mobile(string $phone): bool
	{
		$phone = self::to_english_nums($phone);
		$phone = preg_replace('/\D/', '', $phone);
		if (substr($phone, 0, 2) === '98') {
			$phone = '0' . substr($phone, 2);
		}
		return (bool) preg_match('/^09[0-9]{9}$/', $phone);
	}

	/** سازگاری با نسخه‌های قبلی کد که validate_mobile را صدا می‌زنند */
	public static function validate_mobile(string $phone): bool
	{
		return self::is_valid_mobile($phone);
	}

	/** نرمال‌سازی شماره موبایل */
	public static function normalize_mobile(string $phone): string
	{
		$phone = self::to_english_nums($phone);
		$phone = preg_replace('/\D/', '', $phone);
		if (substr($phone, 0, 2) === '98') {
			$phone = '0' . substr($phone, 2);
		}
		return $phone;
	}

	/** لیست استان‌های ایران */
	public static function get_provinces(): array
	{
		return [
			'آذربایجان شرقی',
			'آذربایجان غربی',
			'اردبیل',
			'اصفهان',
			'البرز',
			'ایلام',
			'بوشهر',
			'تهران',
			'چهارمحال و بختیاری',
			'خراسان جنوبی',
			'خراسان رضوی',
			'خراسان شمالی',
			'خوزستان',
			'زنجان',
			'سمنان',
			'سیستان و بلوچستان',
			'فارس',
			'قزوین',
			'قم',
			'کردستان',
			'کرمان',
			'کرمانشاه',
			'کهگیلویه و بویراحمد',
			'گلستان',
			'گیلان',
			'لرستان',
			'مازندران',
			'مرکزی',
			'هرمزگان',
			'همدان',
			'یزد',
		];
	}

	/** فهرست شهرهای فرم‌های عمومی، مشترک بین PHP و JavaScript. */
	public static function get_cities(): array
	{
		$cities = json_decode( <<<'JSON'
{"آذربایجان شرقی": ["تبریز", "مراغه", "مرند", "اهر", "بناب", "میانه", "سراب", "شبستر", "هشترود", "عجب‌شیر", "ملکان", "اسکو", "بستان‌آباد", "هریس", "کلیبر", "ورزقان", "خداآفرین", "چاراویماق"], "آذربایجان غربی": ["ارومیه", "خوی", "مهاباد", "بوکان", "میاندوآب", "اشنویه", "نقده", "سلماس", "پیرانشهر", "سردشت", "تکاب", "چالدران", "شاهین‌دژ", "ماکو", "پلدشت", "چایپاره"], "اردبیل": ["اردبیل", "پارس‌آباد", "خلخال", "مشگین‌شهر", "گرمی", "بیله‌سوار", "نمین", "نیر", "کوثر", "سرعین"], "اصفهان": ["اصفهان", "کاشان", "خمینی‌شهر", "نجف‌آباد", "شاهین‌شهر", "فلاورجان", "لنجان", "آران و بیدگل", "شهرضا", "مبارکه", "گلپایگان", "برخوار", "تیران و کرون", "سمیرم", "اردستان", "نائین", "خوانسار", "فریدن", "فریدونشهر", "دهاقان", "چادگان"], "البرز": ["کرج", "فردیس", "نظرآباد", "ساوجبلاغ", "طالقان", "محمدشهر", "هشتگرد"], "ایلام": ["ایلام", "دهلران", "ایوان", "مهران", "آبدانان", "دره‌شهر", "چرداول", "بدره", "ملکشاهی"], "بوشهر": ["بوشهر", "بندر گناوه", "برازجان", "بندر دیر", "خورموج", "کنگان", "جم", "دیلم"], "تهران": ["تهران", "شهریار", "پاکدشت", "ورامین", "دماوند", "فیروزکوه", "اسلامشهر", "رباط‌کریم", "قرچک", "ری", "ملارد", "بهارستان", "پردیس", "قدس"], "چهارمحال و بختیاری": ["شهرکرد", "بروجن", "فارسان", "لردگان", "اردل", "کوهرنگ", "سامان", "بن"], "خراسان جنوبی": ["بیرجند", "قاین", "نهبندان", "طبس", "سرایان", "فردوس", "درمیان", "سربیشه", "خوسف", "زیرکوه", "بشرویه"], "خراسان رضوی": ["مشهد", "سبزوار", "نیشابور", "تربت حیدریه", "کاشمر", "قوچان", "تربت جام", "چناران", "فریمان", "درگز", "تایباد", "خواف", "گناباد", "بردسکن", "جوین", "جغتای", "خلیل‌آباد", "مه‌ولات"], "خراسان شمالی": ["بجنورد", "شیروان", "اسفراین", "مانه و سملقان", "جاجرم", "گرمه", "فاروج"], "خوزستان": ["اهواز", "آبادان", "خرمشهر", "دزفول", "مسجدسلیمان", "بهبهان", "اندیمشک", "شوشتر", "شوش", "ماهشهر", "رامهرمز", "امیدیه", "ایذه", "باوی", "لالی", "هندیجان", "دشت آزادگان"], "زنجان": ["زنجان", "ابهر", "خدابنده", "قیدار", "ماهنشان", "سلطانیه", "طارم", "ایجرود"], "سمنان": ["سمنان", "شاهرود", "گرمسار", "دامغان", "مهدیشهر", "آرادان", "سرخه", "میامی"], "سیستان و بلوچستان": ["زاهدان", "چابهار", "زابل", "ایرانشهر", "خاش", "سراوان", "نیکشهر", "کنارک", "دلگان", "میرجاوه", "هیرمند", "قصرقند"], "فارس": ["شیراز", "مرودشت", "کازرون", "جهرم", "فسا", "لارستان", "داراب", "آباده", "نی‌ریز", "فیروزآباد", "استهبان", "اقلید", "ممسنی", "خرم‌بید", "پاسارگاد", "بوانات", "لامرد", "سپیدان", "گراش", "خنج"], "قزوین": ["قزوین", "البرز", "بویین‌زهرا", "تاکستان", "آوج"], "قم": ["قم"], "کردستان": ["سنندج", "سقز", "مریوان", "بانه", "قروه", "کامیاران", "بیجار", "دیواندره", "سروآباد", "دهگلان"], "کرمان": ["کرمان", "رفسنجان", "سیرجان", "جیرفت", "زرند", "شهربابک", "بافت", "بردسیر", "عنبرآباد", "کهنوج", "قلعه‌گنج", "منوجان", "نرماشیر", "فهرج"], "کرمانشاه": ["کرمانشاه", "اسلام‌آباد غرب", "کنگاور", "هرسین", "صحنه", "سنقر", "پاوه", "جوانرود", "روانسر", "دالاهو"], "کهگیلویه و بویراحمد": ["یاسوج", "گچساران", "دهدشت", "کهگیلویه", "بهمئی", "لنده", "باشت", "چرام"], "گلستان": ["گرگان", "گنبدکاووس", "آزادشهر", "علی‌آباد", "کردکوی", "بندرترکمن", "مینودشت", "رامیان", "گالیکش", "مراوه‌تپه", "کلاله", "آق‌قلا", "گمیشان"], "گیلان": ["رشت", "بندر انزلی", "لاهیجان", "لنگرود", "آستارا", "صومعه‌سرا", "رودبار", "رودسر", "تالش", "فومن", "شفت", "سیاهکل", "ماسال", "رضوانشهر"], "لرستان": ["خرم‌آباد", "بروجرد", "کوهدشت", "الیگودرز", "نورآباد", "ازنا", "دلفان", "سلسله", "رومشکان", "پلدختر"], "مازندران": ["ساری", "بابل", "آمل", "قائمشهر", "نوشهر", "بابلسر", "نکا", "چالوس", "تنکابن", "رامسر", "جویبار", "محمودآباد", "فریدونکنار", "بهشهر", "نور", "میاندورود", "سوادکوه", "کلاردشت"], "مرکزی": ["اراک", "ساوه", "خمین", "محلات", "دلیجان", "آشتیان", "شازند", "تفرش", "کمیجان", "زرندیه"], "هرمزگان": ["بندرعباس", "بندر لنگه", "قشم", "میناب", "حاجی‌آباد", "خمیر", "ابوموسی", "بستک", "پارسیان", "جاسک", "رودان"], "همدان": ["همدان", "ملایر", "نهاوند", "تویسرکان", "بهار", "اسدآباد", "کبودراهنگ", "رزن", "فامنین"], "یزد": ["یزد", "میبد", "اردکان", "بافق", "ابرکوه", "طبس", "مهریز", "خاتم", "تفت", "صدوق"]}
JSON
		, true );
		return is_array($cities) ? $cities : [];
	}

	/** برگرداندن پیام JSON و توقف */
	public static function send_json(bool $success, string $message, array $data = []): void
	{
		$payload = wp_json_encode(
			array_merge(['success' => $success, 'message' => $message], $data)
		);
		// پاک کردن هر output قبلی
		while (ob_get_level() > 0) {
			ob_end_clean();
		}
		// ارسال response
		wp_send_json(json_decode($payload, true));
	}

	/** دریافت محصولات فعال‌شده برای شبکه فروش */
	public static function get_sn_products(int $user_id = 0, bool $unfiltered = false): array
	{
		if (! class_exists('WooCommerce')) {
			return [];
		}
		$query = new WP_Query([
			'post_type'      => 'product',
			'post_status'    => 'publish',
			'posts_per_page' => -1,
			'meta_query'     => [[
				'key'   => '_sn_enabled',
				'value' => '1',
			]],
		]);
		$products = [];
		foreach ($query->posts as $post) {
			$product = wc_get_product($post->ID);
			if ($product) {
				$products[] = [
					'id'    => $product->get_id(),
					'name'  => $product->get_name(),
					'price' => (float) $product->get_price(),
					// Product credit is separate from the invoice price. A missing
					// value intentionally remains zero for legacy products.
					'credit' => (float) get_post_meta($post->ID, '_sn_product_credit_amount', true),
				];
			}
		}
		return $unfiltered ? $products : SN_Sales_Catalog::instance()->filter_products($products, $user_id ?: get_current_user_id());
	}

	/** پیدا کردن فاکتور با کد */
	public static function get_invoice_by_code(string $code): ?object
	{
		global $wpdb;
		$code = self::normalize_invoice_code($code);
		if ($code === '') {
			return null;
		}
		return $wpdb->get_row($wpdb->prepare(
			"SELECT * FROM {$wpdb->prefix}sn_invoices WHERE UPPER(invoice_code) = %s LIMIT 1",
			$code
		));
	}

	/** Uploaded files are removed at shutdown unless their DB transaction commits. */
	private static $pending_receipt_files = [];

	public static function receipt_urls($record): array
	{
		$record = (array) $record;
		$urls = $record['receipt_urls'] ?? [];
		if (is_string($urls)) { $urls = json_decode($urls, true); }
		$urls = is_array($urls) ? $urls : [];
		foreach (['receipt_url', 'receipt_file'] as $key) {
			if (!empty($record[$key]) && is_string($record[$key])) { $urls[] = $record[$key]; }
		}
		return array_values(array_unique(array_filter(array_map(static function($url) {
			return is_string($url) ? esc_url_raw($url, ['http', 'https']) : '';
		}, $urls))));
	}

	public static function receipt_links($record): string
	{
		$links = [];
		foreach (self::receipt_urls($record) as $index => $url) {
			$links[] = '<a class="sn-btn sn-btn-sm sn-btn-secondary" href="'.esc_url($url).'" target="_blank" rel="noopener">فیش '.esc_html((string)($index + 1)).'</a>';
		}
		return $links ? '<div class="sn-receipt-links">'.implode(' ', $links).'</div>' : '';
	}

	public static function has_receipt_upload(array $file): bool
	{
		$errors = $file['error'] ?? UPLOAD_ERR_NO_FILE;
		foreach ((array)$errors as $error) {
			if (is_array($error) || (int)$error !== UPLOAD_ERR_NO_FILE) { return true; }
		}
		return false;
	}

	public static function commit_receipt_uploads(array $urls): void
	{
		foreach ($urls as $url) { unset(self::$pending_receipt_files[$url]); }
	}

	public static function cleanup_receipt_uploads(): void
	{
		foreach (self::$pending_receipt_files as $path) { wp_delete_file($path); }
		self::$pending_receipt_files = [];
	}

	/** Accept both legacy receipt and receipt[]; a partial batch must never succeed. */
	public static function upload_receipt(array $file): array
	{
		$files = [];
		if (isset($file['name']) && is_array($file['name'])) {
			foreach ($file['name'] as $key => $name) {
				$item = [];
				foreach (['name','type','tmp_name','error','size'] as $field) {
					if (!isset($file[$field]) || !is_array($file[$field]) || !array_key_exists($key, $file[$field]) || !is_scalar($file[$field][$key])) {
						return ['success'=>false,'message'=>'ساختار فایل‌های فیش معتبر نیست'];
					}
					$item[$field] = $file[$field][$key];
				}
				$files[] = $item;
			}
		} else { $files[] = $file; }
		$count = count($files);
		$expected = isset($_POST['receipt_count']) && is_scalar($_POST['receipt_count']) ? (int)$_POST['receipt_count'] : $count;
		if ($count < 1 || $count > 10 || $expected !== $count) {
			return ['success'=>false,'message'=>'حداکثر ۱۰ فیش در هر ارسال مجاز است؛ همه فایل‌ها باید به سرور برسند'];
		}
		$total = 0;
		foreach ($files as $item) {
			if (!isset($item['name'],$item['tmp_name'],$item['size'],$item['error']) || !is_string($item['name']) || !is_string($item['tmp_name']) || !is_numeric($item['size']) || !is_numeric($item['error']) || (int)$item['error'] !== UPLOAD_ERR_OK) {
				return ['success'=>false,'message'=>'یک یا چند فیش کامل دریافت نشد؛ حجم فایل‌ها و محدودیت سرور را بررسی کنید'];
			}
			$total += (int)$item['size'];
		}
		if ($total > 20 * 1024 * 1024) { return ['success'=>false,'message'=>'مجموع حجم فیش‌ها در هر ارسال حداکثر ۲۰ مگابایت است']; }
		$urls = [];
		foreach ($files as $item) {
			$result = self::upload_single_receipt($item);
			if (empty($result['success']) || empty($result['url'])) {
				self::cleanup_receipt_uploads();
				return ['success'=>false,'message'=>(string)($result['message'] ?? 'آپلود فیش انجام نشد')];
			}
			$urls[] = $result['url'];
		}
		return ['success'=>true,'url'=>$urls[0],'urls'=>$urls];
	}

	/** آپلود فایل فیش پرداخت */
	private static function upload_single_receipt(array $file): array
	{
		if (! function_exists('wp_handle_upload')) {
			require_once ABSPATH . 'wp-admin/includes/file.php';
		}
		if (! function_exists('wp_check_filetype_and_ext')) {
			require_once ABSPATH . 'wp-admin/includes/file.php';
		}

		if (empty($file) || ! empty($file['error'])) {
			return ['success' => false, 'message' => 'فایل آپلود نشد'];
		}

		$max_size = (int) apply_filters('sn_receipt_upload_max_size', 5 * 1024 * 1024);
		if (! empty($file['size']) && (int) $file['size'] > $max_size) {
			return ['success' => false, 'message' => 'حجم فایل فیش بیش از حد مجاز است'];
		}

		$allowed_mimes = [
			'jpg'  => 'image/jpeg',
			'jpeg' => 'image/jpeg',
			'png'  => 'image/png',
			'webp' => 'image/webp',
			'pdf'  => 'application/pdf',
		];

		$name = isset($file['name']) ? sanitize_file_name((string) $file['name']) : '';
		$file['name'] = $name;
		$check = wp_check_filetype_and_ext((string) $file['tmp_name'], $name, $allowed_mimes);
		$ext = strtolower((string) ($check['ext'] ?? ''));
		$type = strtolower((string) ($check['type'] ?? ''));
		if (! $ext || ! $type || ! isset($allowed_mimes[$ext]) || $allowed_mimes[$ext] !== $type) {
			return ['success' => false, 'message' => 'نوع فایل فیش مجاز نیست'];
		}
		if (preg_match('/\.(php|phtml|phar|js|html?|svg|exe|sh|bat|cmd)$/i', $name)) {
			return ['success' => false, 'message' => 'نوع فایل فیش مجاز نیست'];
		}

		add_filter('upload_dir', [__CLASS__, 'receipt_upload_dir']);
		$uploaded = wp_handle_upload($file, [
			'test_form' => false,
			'mimes'     => $allowed_mimes,
		]);
		remove_filter('upload_dir', [__CLASS__, 'receipt_upload_dir']);

		if (! empty($uploaded['error'])) {
			return ['success' => false, 'message' => sanitize_text_field((string) $uploaded['error'])];
		}

		$url = esc_url_raw((string) ($uploaded['url'] ?? ''));
		if ($url === '' || empty($uploaded['file'])) { return ['success'=>false,'message'=>'مسیر فیش ذخیره‌شده معتبر نیست']; }
		static $cleanup_registered = false;
		if (!$cleanup_registered) { register_shutdown_function([__CLASS__, 'cleanup_receipt_uploads']); $cleanup_registered = true; }
		self::$pending_receipt_files[$url] = (string)$uploaded['file'];
		return ['success' => true, 'url' => $url];
	}

	public static function receipt_upload_dir(array $dirs): array
	{
		$subdir = '/sn-receipts' . (empty($dirs['subdir']) ? '' : $dirs['subdir']);
		$dirs['path'] = $dirs['basedir'] . $subdir;
		$dirs['url'] = $dirs['baseurl'] . $subdir;
		$dirs['subdir'] = $subdir;
		return $dirs;
	}

	public static function client_ip(): string
	{
		$ip = isset($_SERVER['REMOTE_ADDR']) ? sanitize_text_field(wp_unslash($_SERVER['REMOTE_ADDR'])) : '';
		return preg_match('/^[a-f0-9:\.]+$/i', $ip) ? $ip : 'unknown';
	}

	public static function rate_limit(string $action, string $subject = '', int $limit = 10, int $window = 600): array
	{
		$user = get_current_user_id();
		$key = 'sn_rl_' . md5($action . '|' . self::client_ip() . '|' . $user . '|' . $subject);
		$data = get_transient($key);
		$data = is_array($data) ? $data : ['count' => 0, 'reset' => time() + $window];
		$data['count'] = (int) ($data['count'] ?? 0) + 1;
		if (empty($data['reset']) || (int) $data['reset'] < time()) {
			$data = ['count' => 1, 'reset' => time() + $window];
		}
		set_transient($key, $data, max(1, (int) $data['reset'] - time()));
		if ((int) $data['count'] > $limit) {
			return ['allowed' => false, 'message' => 'تعداد درخواست‌ها بیش از حد مجاز است. کمی بعد دوباره تلاش کنید.', 'retry_after' => max(1, (int) $data['reset'] - time())];
		}
		return ['allowed' => true, 'remaining' => max(0, $limit - (int) $data['count'])];
	}
}
