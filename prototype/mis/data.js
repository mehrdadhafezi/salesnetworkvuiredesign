/* MIS — mock data for the design prototype (SN-204). NOT business authority: every number/ID/name is fictional.
   Product boundaries live in docs/source-of-truth/MIS-PRODUCT-SPEC.md + GATE-0-PRODUCT-INVARIANTS.md.
   Nothing here is saved, sent or executed. Native IDs stay native; "CaseRef" is deliberately unresolved (resolver coverage is a product gap, MIS-G02). */
(function () {
  'use strict';
  var L = function (k, v, conf, note) { return { k: k, v: v, conf: conf, note: note || '' }; };

  var M = window.MIS = {
    user: { name: 'سمیرا کاظمی', id: '۱۹۰۰۷', pos: 'MIS' },
    freshness: { now: '۱۴:۲۰', evaluated: 'امروز ۱۴:۲۰ (+۰۳:۳۰)', stale: 'دیروز ۱۸:۱۰' },

    /* ---------- File · Batch · Import Run · Source Row — four different things ---------- */
    files: [
      { id: 'F-2041', name: 'leads-aban-1405.xlsx', fmt: 'XLSX', size: '۲٫۱ مگابایت', at: '۱۴۰۵/۰۷/۱۶ · ۰۹:۱۲', actor: 'سمیرا کاظمی', batch: 'B-1187' },
      { id: 'F-2040', name: 'campaign-ig-mehr.csv', fmt: 'CSV', size: '۶۴۰ کیلوبایت', at: '۱۴۰۵/۰۷/۱۵ · ۱۱:۰۵', actor: 'سمیرا کاظمی', batch: 'B-1186' },
      { id: 'F-2038', name: 'legacy-fix-resend.csv', fmt: 'CSV', size: '۱٫۴ مگابایت', at: '۱۴۰۵/۰۷/۱۲ · ۱۶:۴۰', actor: 'سمیرا کاظمی', batch: 'B-1180' },
      { id: 'F-2035', name: 'old-expo-1404.xlsx', fmt: 'XLSX', size: '۳٫۸ مگابایت', at: '۱۴۰۵/۰۶/۲۸ · ۱۰:۳۰', actor: 'امیر حاتمی', batch: 'B-1171' }
    ],
    batches: [
      { id: 'B-1187', name: 'آبان · اینستاگرام', file: 'F-2041', source: 'اینستاگرام', rows: 480, state: 'partial', assigned: 0, consumed: 0 },
      { id: 'B-1186', name: 'کمپین مهر · اینستاگرام', file: 'F-2040', source: 'اینستاگرام', rows: 220, state: 'complete', assigned: 120, consumed: 14 },
      { id: 'B-1180', name: 'ارسال مجدد لیدهای قدیمی', file: 'F-2038', source: 'خروجی قدیمی', rows: 300, state: 'unknown', assigned: 0, consumed: 0 },
      { id: 'B-1171', name: 'نمایشگاه ۱۴۰۴', file: 'F-2035', source: 'نمایشگاه', rows: 214, state: 'complete', assigned: 214, consumed: 36 }
    ],
    /* An import run reconciles: parsed = imported + duplicate + invalid + failed + unknown. «skipped» = duplicate + invalid. */
    runs: [
      { id: 'R-9922', type: 'assign', batch: 'B-1186', file: 'F-2040', actor: 'سمیرا کاظمی', at: 'امروز ۱۳:۵۵', state: 'complete', note: 'تخصیص منبع به مدیر · ۱۲۰ ردیف' },
      { id: 'R-9921', type: 'import', batch: 'B-1187', file: 'F-2041', actor: 'سمیرا کاظمی', at: 'امروز ۰۹:۱۴', state: 'partial', seen: 480, imported: 412, dup: 31, invalid: 22, failed: 9, unknown: 6,
        dupClasses: [['infile', 'تکراری در همان فایل', 9], ['samebatch', 'ورود مجدد در همین دسته', 12], ['othermis', 'ردیف MIS در دسته دیگر', 6], ['legacy', 'لید قدیمی', 4]],
        warns: ['۳ ردیف: ستون شماره دوم شناسایی شد ولی خالی بود', 'سربرگ «کمپین» با دو نام پیدا شد؛ نخستین استفاده شد', '۶ ردیف: پاسخ ثبت از پایگاه داده دریافت نشد'] },
      { id: 'R-9917', type: 'import', batch: 'B-1186', file: 'F-2040', actor: 'سمیرا کاظمی', at: '۱۴۰۵/۰۷/۱۵ · ۱۱:۰۸', state: 'complete', seen: 220, imported: 211, dup: 7, invalid: 2, failed: 0, unknown: 0,
        dupClasses: [['infile', 'تکراری در همان فایل', 3], ['samebatch', 'ورود مجدد در همین دسته', 0], ['othermis', 'ردیف MIS در دسته دیگر', 2], ['legacy', 'لید قدیمی', 2]], warns: [] },
      { id: 'R-9908', type: 'import', batch: 'B-1180', file: 'F-2038', actor: 'سمیرا کاظمی', at: '۱۴۰۵/۰۷/۱۲ · ۱۶:۴۴', state: 'unknown', seen: 300, imported: 212, dup: 0, invalid: 0, failed: 8, unknown: 80,
        dupClasses: [], warns: ['ارتباط با سرور در ردیف ۲۲۱ قطع شد؛ نتیجه ۸۰ ردیف آخر ثبت نشد'] },
      { id: 'R-9890', type: 'pool', batch: 'B-1186', file: 'F-2040', actor: 'سمیرا کاظمی', at: '۱۴۰۵/۰۷/۱۱ · ۱۰:۲۰', state: 'complete', note: 'آماده‌سازی استخر · ۴۰ ردیف' }
    ],
    health: { cached: 'warn', checkedAt: 'امروز ۱۳:۴۰', source: 'تشخیص سلامت ساختار (ذخیره‌شده، نه اجرای زنده)', msg: 'هشدار: یک جدول ساختار گم‌شده یا قدیمی گزارش شده', live: 'ok', cause: 'UNKNOWN' },

    /* ---------- Source rows (native = sn_mis_data_rows.id). ladder = identity nodes; fin = financial dependency per domain ---------- */
    cases: [
      { id: '884201', batch: 'B-1187', row: 3, val: 'valid', phone: '۰۹۱۲···۴۱۰۲', phoneMatches: 1, stage: 'unassigned', lin: 'verified',
        ladder: [L('mis', '۸۸۴۲۰۱', 'verified', 'شناسه بومی ردیف MIS'), L('case', 'تعریف نشده', 'unknown', 'حل‌کننده یکپارچه فعال نیست؛ پرونده منطقی ساخته یا حدس زده نمی‌شود')],
        fin: { legacy: ['none', 'لید قدیمی متصل نیست'], v4: ['none', 'آیتم توزیع V4 ندارد'], flow: ['none', 'رویداد جریان فروشنده ندارد'], dot: ['none', 'پرونده Dot ندارد'] }, ret: 'na',
        cust: { src: 'MIS · دسته B-1187', cur: 'نزد منبع — تحویل نشده', orig: 'MIS (ورود منبع)', next: 'MIS: تخصیص به مدیر', credit: 'هنوز فروشی ثبت نشده', event: 'ورود فایل · سمیرا کاظمی', recip: '—' },
        hist: [['ورود از فایل', 'امروز ۰۹:۱۴', 'اجرای R-9921 · ردیف ۳ فایل']] },
      { id: '884202', batch: 'B-1186', row: 12, val: 'valid', phone: '۰۹۳۵···۷۷۱۰', phoneMatches: 0, stage: 'assigned', lin: 'verified',
        ladder: [L('mis', '۸۸۴۲۰۲', 'verified', 'شناسه بومی ردیف MIS'), L('case', 'تعریف نشده', 'unknown', 'حل‌کننده فعال نیست')],
        fin: { legacy: ['none', 'متصل نیست'], v4: ['none', 'آیتم ندارد'], flow: ['none', 'رویداد ندارد'], dot: ['none', 'ندارد'] }, ret: 'na',
        cust: { src: 'مدیر فرهاد نجفی (assigned_manager منبع)', cur: 'نامشخص — از فیلد منبع حدس زده نمی‌شود', orig: 'MIS (ورود منبع)', next: 'MIS: آماده‌سازی استخر یا تحویل', credit: 'هنوز فروشی ثبت نشده', event: 'تخصیص منبع · سمیرا کاظمی', recip: 'مدیر فرهاد نجفی (در منبع)' },
        hist: [['تخصیص منبع به مدیر', 'امروز ۱۳:۵۵', 'اجرای R-9922 · فقط فیلد منبع تغییر کرد'], ['ورود از فایل', '۱۴۰۵/۰۷/۱۵ · ۱۱:۰۸', 'اجرای R-9917']] },
      { id: '884203', batch: 'B-1186', row: 31, val: 'valid', phone: '۰۹۱۹···۲۲۴۰', phoneMatches: 2, stage: 'delivered', lin: 'partial',
        ladder: [L('mis', '۸۸۴۲۰۳', 'verified', 'شناسه بومی ردیف MIS'), L('legacy', '۶۱۱۰۲', 'partial', 'فقط با شماره موبایل دیده شده؛ شناسه پایدار ثابت نشده'), L('alias', 'MIS ۸۸۴۲۰۳ ↔ لید ۶۱۱۰۲', 'partial', 'نوع رابطه: ادامه یا مرتبط‌اما‌متفاوت؟ (OPD-01) — اثبات کافی نیست'), L('case', 'تعریف نشده', 'unknown', 'ادغام یا حدس انجام نمی‌شود')],
        fin: { legacy: ['unknown', 'لید ۶۱۱۰۲ احراز نشده؛ بدون اتصال پایدار بررسی نمی‌شود'], v4: ['none', 'آیتم ندارد'], flow: ['none', 'رویداد ندارد'], dot: ['unknown', 'پوشش آداپتور Dot اعلام نشده'] }, ret: 'unknown', retWhy: 'اتصال جزئی؛ روابط مالی دو حوزه اثبات نشده (fail-closed)',
        cust: { src: 'مدیر فرهاد نجفی (منبع)', cur: 'فروشنده زهرا کریمی', orig: 'فروشنده زهرا کریمی', next: 'فروشنده: پیگیری', credit: 'هنوز فروشی ثبت نشده', event: 'تحویل سریع · سمیرا کاظمی', recip: 'فروشنده زهرا کریمی' },
        hist: [['تحویل سریع به فروشنده (مسیر استثنایی)', '۱۴۰۵/۰۷/۱۵ · ۱۵:۱۰', 'سطوح ردشده: مدیر، سرپرست ارشد، سرپرست · اجرای Q-77'], ['ورود از فایل', '۱۴۰۵/۰۷/۱۵ · ۱۱:۰۸', 'اجرای R-9917']] },
      { id: '884204', batch: 'B-1186', row: 44, val: 'valid', phone: '۰۹۱۲···۹۰۳۱', phoneMatches: 1, stage: 'delivered', lin: 'conflict', oldReturnable: true,
        ladder: [L('mis', '۸۸۴۲۰۴', 'verified', 'شناسه بومی ردیف MIS'), L('v4', 'آیتم ۷۷۱۲', 'verified', 'follow_invoice_id = ۹۰۳۴۵ (پیش‌فاکتور)، lead_id = ۰'), L('legacy', '۶۱۱۴۴', 'conflict', 'لید قدیمی می‌گوید فاکتوری ندارد؛ با آیتم V4 تناقض دارد'), L('dot', 'پرونده Dot ۵۵۰۲ / ۵۵۰۳', 'conflict', 'دو پرونده Dot ممکن است همین پرونده یا دو پرونده جدا باشند'), L('case', 'تعریف نشده', 'unknown', 'تناقض باید توسط مالک حوزه حل شود')],
        fin: { legacy: ['none', 'lead_id مجازی فاکتوری نشان نمی‌دهد'], v4: ['active', 'follow_invoice_id ← پیش‌فاکتور ۹۰۳۴۵ (فعال)'], flow: ['none', 'رویداد ندارد'], dot: ['unknown', 'دو پرونده Dot؛ ارتباط مالی اثبات نشده'] }, ret: 'blocked', retWhy: 'وابستگی مالی فعال از مسیر V4 (F01) — نمای قدیمی «قابل برگشت» می‌گوید',
        cust: { src: 'مدیر مهسا رضوی (منبع)', cur: 'فروشنده امین توکلی', orig: 'مدیر مهسا رضوی', next: 'فروشنده / مالی طبق وضعیت فاکتور', credit: 'طبق پیش‌فاکتور ۹۰۳۴۵ (قوانین مالی فعلی)', event: 'تحویل عادی · مدیر مهسا رضوی', recip: 'فروشنده امین توکلی' },
        hist: [['صدور پیش‌فاکتور', '۱۴۰۵/۰۷/۱۶ · ۰۸:۲۰', 'فروشنده امین توکلی · پیش‌فاکتور ۹۰۳۴۵'], ['تحویل به فروشنده', '۱۴۰۵/۰۷/۱۵ · ۱۲:۰۰', 'مدیر مهسا رضوی'], ['تخصیص منبع به مدیر', '۱۴۰۵/۰۷/۱۵ · ۱۱:۲۰', 'اجرای R-9899']] },
      { id: '884205', batch: 'B-1180', row: 7, val: 'valid', phone: '۰۹۲۱···۵۵۰۸', phoneMatches: 0, stage: 'delivered', lin: 'unknown',
        ladder: [L('mis', '۸۸۴۲۰۵', 'verified', 'شناسه بومی ردیف MIS'), L('case', 'تعریف نشده', 'unknown', 'هیچ مدرک اتصال به حوزه‌های دیگر یافت نشد؛ «بدون اتصال» نیز اثبات نشده')],
        fin: { legacy: ['unknown', 'پوشش اعلام نشده'], v4: ['unknown', 'پوشش اعلام نشده'], flow: ['unknown', 'پوشش اعلام نشده'], dot: ['unknown', 'پوشش اعلام نشده'] }, ret: 'unknown', retWhy: 'هیچ حوزه مالی پوشش داده نشده؛ نبود فاکتور در یک نما اثبات عدم مصرف نیست',
        cust: { src: 'مدیر فرهاد نجفی (منبع)', cur: 'سرپرست آرش یگانه', orig: 'مدیر فرهاد نجفی', next: 'نامشخص', credit: 'نامشخص', event: 'ثبت نشده', recip: 'سرپرست آرش یگانه' },
        hist: [['تحویل', '۱۴۰۵/۰۷/۱۲ · ۱۷:۳۰', 'رویداد انتقال کامل ثبت نشده — عامل نامشخص (UNKNOWN)']] },
      { id: '884206', batch: 'B-1187', row: 17, val: 'dup', dupClass: 'infile', reason: 'تکراری در همان فایل (ردیف ۱۷ با ردیف ۳)', phone: '۰۹۱۲···۴۱۰۲', phoneMatches: 1, stage: 'none', lin: 'unknown',
        ladder: [L('mis', '۸۸۴۲۰۶', 'verified', 'ردیف ذخیره‌شده با طبقه «تکراری»'), L('case', 'تعریف نشده', 'unknown', 'شماره یکسان اثبات یک پرونده نیست؛ ادغام انجام نمی‌شود')],
        fin: { legacy: ['none', '—'], v4: ['none', '—'], flow: ['none', '—'], dot: ['none', '—'] }, ret: 'na',
        cust: { src: 'MIS · دسته B-1187', cur: 'نزد منبع — تحویل نشده', orig: 'MIS', next: 'MIS: بررسی تکرار', credit: '—', event: 'ورود فایل', recip: '—' }, hist: [['ورود به‌عنوان تکراری', 'امروز ۰۹:۱۴', 'اجرای R-9921']] },
      { id: '884207', batch: 'B-1187', row: 52, val: 'dup', dupClass: 'othermis', reason: 'ردیف MIS دیگر (دسته B-1180، ردیف ۱۲۰)', phone: '۰۹۱۹···۶۶۱۴', phoneMatches: 1, stage: 'none', lin: 'partial',
        ladder: [L('mis', '۸۸۴۲۰۷', 'verified', 'ردیف جاری'), L('alias', 'MIS ۸۸۴۲۰۷ ↔ MIS ۸۷۹۰۱۰', 'partial', 'شماره یکسان؛ ادامه یا پرونده جدا ثابت نشده'), L('case', 'تعریف نشده', 'unknown')],
        fin: { legacy: ['none', '—'], v4: ['none', '—'], flow: ['none', '—'], dot: ['none', '—'] }, ret: 'na',
        cust: { src: 'MIS · دسته B-1187', cur: 'نزد منبع — تحویل نشده', orig: 'MIS', next: 'MIS: بررسی تکرار', credit: '—', event: 'ورود فایل', recip: '—' }, hist: [['ورود به‌عنوان تکراری', 'امروز ۰۹:۱۴', 'اجرای R-9921']] },
      { id: '884208', batch: 'B-1187', row: 61, val: 'invalid', reason: 'شماره موبایل ناقص (۹ رقم)', phone: '۰۹۱···۸۷۶', phoneMatches: 0, stage: 'none', lin: 'unknown',
        ladder: [L('mis', '۸۸۴۲۰۸', 'verified', 'ردیف رد‌شده'), L('case', 'تعریف نشده', 'unknown')],
        fin: { legacy: ['none', '—'], v4: ['none', '—'], flow: ['none', '—'], dot: ['none', '—'] }, ret: 'na',
        cust: { src: 'MIS · دسته B-1187', cur: 'نزد منبع — قابل تحویل نیست', orig: 'MIS', next: 'MIS: اصلاح فایل منبع', credit: '—', event: 'ورود فایل', recip: '—' }, hist: [['رد در کیفیت ورودی', 'امروز ۰۹:۱۴', 'اجرای R-9921 · نمونه محدود']] },
      { id: '884209', batch: 'B-1187', row: 70, val: 'dup', dupClass: 'legacy', reason: 'لید قدیمی ۶۱۰۰۷', phone: '۰۹۳۶···۳۳۰۱', phoneMatches: 1, stage: 'none', lin: 'partial',
        ladder: [L('mis', '۸۸۴۲۰۹', 'verified', 'ردیف جاری'), L('legacy', '۶۱۰۰۷', 'partial', 'فقط شماره یکسان؛ شناسه پایدار ثابت نشده'), L('case', 'تعریف نشده', 'unknown')],
        fin: { legacy: ['unknown', 'لید ۶۱۰۰۷ احراز نشده'], v4: ['none', '—'], flow: ['none', '—'], dot: ['unknown', 'پوشش اعلام نشده'] }, ret: 'na',
        cust: { src: 'MIS · دسته B-1187', cur: 'نزد منبع — تحویل نشده', orig: 'MIS', next: 'MIS: بررسی تکرار', credit: '—', event: 'ورود فایل', recip: '—' }, hist: [['ورود به‌عنوان تکراری', 'امروز ۰۹:۱۴', 'اجرای R-9921']] },
      { id: '884210', batch: 'B-1171', row: 90, val: 'valid', phone: '۰۹۱۴···۱۲۸۸', phoneMatches: 0, stage: 'delivered', lin: 'verified',
        ladder: [L('mis', '۸۸۴۲۱۰', 'verified', 'شناسه بومی'), L('v4', 'آیتم ۷۶۹۰', 'verified', 'follow_invoice_id = ۹۰۱۱۲'), L('dot', 'فاکتور Dot ۴۴۰۱ (لغوشده)', 'verified', 'فاکتور لغو شده'), L('case', 'تعریف نشده', 'unknown')],
        fin: { legacy: ['none', 'متصل نیست'], v4: ['cancelled', 'فاکتور ۹۰۱۱۲ لغو شده'], flow: ['none', '—'], dot: ['cancelled', 'فاکتور Dot ۴۴۰۱ لغو شده'] }, ret: 'blocked', retWhy: 'فاکتور لغوشده؛ قانون آزادسازی تعریف نشده (OPD-03) — لغو/رد آزادسازی نیست',
        cust: { src: 'مدیر مهسا رضوی (منبع)', cur: 'فروشنده سارا مرادی', orig: 'مدیر مهسا رضوی', next: 'فروشنده / مالی', credit: 'طبق فاکتور ۹۰۱۱۲ (قوانین مالی فعلی)', event: 'تحویل عادی · مدیر مهسا رضوی', recip: 'فروشنده سارا مرادی' },
        hist: [['لغو فاکتور', '۱۴۰۵/۰۷/۰۵', 'واحد مالی'], ['تحویل', '۱۴۰۵/۰۶/۲۹', 'مدیر مهسا رضوی']] },
      { id: '884211', batch: 'B-1186', row: 75, val: 'valid', phone: '۰۹۱۲···۰۶۵۲', phoneMatches: 0, stage: 'delivered', lin: 'verified',
        ladder: [L('mis', '۸۸۴۲۱۱', 'verified', 'شناسه بومی'), L('v4', 'آیتم ۷۷۳۵', 'verified', 'بدون follow_invoice_id'), L('case', 'تعریف نشده', 'unknown')],
        fin: { legacy: ['none', 'lead_id مجازی بدون فاکتور'], v4: ['none', 'follow_invoice_id ندارد (پوشش اثبات‌شده)'], flow: ['none', 'رویداد فاکتور ندارد'], dot: ['none', 'آداپتور Dot: ارتباطی نیست'] }, ret: 'ok', retWhy: '',
        cust: { src: 'مدیر فرهاد نجفی (منبع)', cur: 'مدیر فرهاد نجفی', orig: 'مدیر فرهاد نجفی', next: 'مدیر: تحویل به تیم', credit: 'هنوز فروشی ثبت نشده', event: 'تحویل · سمیرا کاظمی', recip: 'مدیر فرهاد نجفی' },
        hist: [['تحویل به مدیر', '۱۴۰۵/۰۷/۱۵ · ۱۴:۰۰', 'سمیرا کاظمی · اجرای R-9899']] },
      { id: '884212', batch: 'B-1186', row: 88, val: 'valid', phone: '۰۹۳۰···۴۴۱۹', phoneMatches: 0, stage: 'delivered', lin: 'verified',
        ladder: [L('mis', '۸۸۴۲۱۲', 'verified', 'شناسه بومی'), L('v4', 'آیتم ۷۷۴۰', 'verified', ''), L('case', 'تعریف نشده', 'unknown')],
        fin: { legacy: ['none', '—'], v4: ['none', '—'], flow: ['none', '—'], dot: ['none', '—'] }, ret: 'conflict', retWhy: 'مسئول فعلی پس از پیش‌نمایش تغییر کرده (انتقال تازه به سرپرست)',
        cust: { src: 'مدیر فرهاد نجفی (منبع)', cur: 'سرپرست آرش یگانه (تازه تغییر کرد)', orig: 'مدیر فرهاد نجفی', next: 'سرپرست: پیگیری', credit: 'هنوز فروشی ثبت نشده', event: 'انتقال · مدیر فرهاد نجفی (۱۴:۱۱)', recip: 'سرپرست آرش یگانه' },
        hist: [['انتقال به سرپرست', 'امروز ۱۴:۱۱', 'مدیر فرهاد نجفی — پس از پیش‌نمایش MIS'], ['تحویل به مدیر', '۱۴۰۵/۰۷/۱۵ · ۱۴:۰۰', 'سمیرا کاظمی']] },
      { id: '884213', batch: 'B-1186', row: 97, val: 'valid', phone: '۰۹۱۸···۷۷۰۰', phoneMatches: 0, stage: 'delivered', lin: 'verified', oldReturnable: false,
        ladder: [L('mis', '۸۸۴۲۱۳', 'verified', 'شناسه بومی'), L('v4', 'آیتم ۷۷۴۴', 'verified', 'follow_invoice_id = ۹۰۴۰۱'), L('dot', 'فاکتور Dot ۴۴۸۸', 'verified', 'تکمیل‌شده، پرداخت تأییدشده'), L('case', 'تعریف نشده', 'unknown')],
        fin: { legacy: ['none', '—'], v4: ['active', 'فاکتور ۹۰۴۰۱ فعال'], flow: ['active', 'رویداد پرداخت ثبت شده'], dot: ['active', 'فاکتور Dot ۴۴۸۸ تکمیل‌شده'] }, ret: 'blocked', retWhy: 'فاکتور فعال و پرداخت تأییدشده — محافظت‌شده',
        cust: { src: 'مدیر مهسا رضوی (منبع)', cur: 'فروشنده امین توکلی', orig: 'مدیر مهسا رضوی', next: 'فروشنده / مالی', credit: 'طبق فاکتور ۹۰۴۰۱', event: 'تحویل عادی', recip: 'فروشنده امین توکلی' },
        hist: [['پرداخت تأییدشده', '۱۴۰۵/۰۷/۱۴', 'واحد مالی'], ['تحویل به فروشنده', '۱۴۰۵/۰۷/۰۹', 'مدیر مهسا رضوی']] }
    ],
    managers: [
      { id: 'MG1', name: 'فرهاد نجفی', ok: true, note: 'فعال · نقش مدیر فروش' },
      { id: 'MG2', name: 'مهسا رضوی', ok: true, note: 'فعال · نقش مدیر فروش' },
      { id: 'MG3', name: 'کیان عباسی', ok: false, why: 'غیرفعال', note: 'غیرفعال' },
      { id: 'MG4', name: 'ندا سلیمی', ok: false, why: 'نقش مدیر فروش اثبات نشده', note: 'اثبات نشده' }
    ],

    /* ---------- Reconciliation issues. Each: evidence · affected entity · owner/domain · confidence · allowed MIS action · NOT allowed · resolution proof ---------- */
    classes: {
      fin: { label: 'وابستگی مالی / برگشت', icon: 'lock' }, identity: { label: 'ناهماهنگی هویت', icon: 'link' }, coverage: { label: 'ناهماهنگی پوشش منبع', icon: 'layers' }, dup: { label: 'ردیف منبع تکراری', icon: 'copy' },
      alias: { label: 'alias حل‌نشده', icon: 'unlink' }, custody: { label: 'مسئول فعلی ثبت‌نشده', icon: 'user' }, hier: { label: 'ناسازگاری سلسله‌مراتب', icon: 'users' }, invrep: { label: 'مغایرت فاکتور و گزارش', icon: 'receipt' },
      stale: { label: 'گزارش قدیمی', icon: 'clock' }, partial: { label: 'ورود ناقص', icon: 'split' }, unknownw: { label: 'نتیجه نوشتن نامعلوم', icon: 'question' }, orphan: { label: 'رابطه شبه‌یتیم', icon: 'unlink' }, health: { label: 'تناقض سلامت', icon: 'activity' }, semantic: { label: 'معنای زمانی گزارش', icon: 'history' }
    },
    issues: [
      { id: 'I1', cls: 'fin', title: 'نمای «قابل برگشت» با پیش‌فاکتور اتصال‌دار هم‌خوان نیست (F01)', entity: ['case', '884204', 'ردیف MIS ۸۸۴۲۰۴'], owner: 'MIS (منبع) + مالی (فاکتور)', conf: 'verified', state: 'open', sev: 'red',
        evidence: ['شمارنده قدیمی «قابل برگشت»: ۱ مورد، شامل این ردیف', 'آیتم V4 ۷۷۱۲: lead_id = ۰ و follow_invoice_id = ۹۰۳۴۵ (پیش‌فاکتور فعال)', 'محافظ فعلی فقط lead_id مجازی و فاکتورهای غیرلغو را می‌بیند'],
        allowed: ['خواندن اتحاد همه ارتباط‌های مالی', 'برچسب «مسدود» و ارجاع به مالک حوزه'], notAllowed: ['برگشت / بازنشانی / پاک‌کردن اتصال', 'اتصال مجدد فاکتور', 'تصمیم درباره وضعیت مالی'], proof: ['همه ارتباط‌های مالی (قدیمی، V4، جریان فروشنده، Dot) در یک اتحاد بررسی و مصرف فعال ثبت شده', 'محافظ برگشت و نمایش با هم یکسان شده‌اند (MIS-G01)'], next: 'مهندسی + مالی' },
      { id: 'I2', cls: 'identity', title: 'اتصال فقط با شماره موبایل دیده شده است', entity: ['case', '884203', 'ردیف MIS ۸۸۴۲۰۳'], owner: 'مالک حل‌کننده (resolver)', conf: 'partial', state: 'open', sev: 'amber',
        evidence: ['لید قدیمی ۶۱۱۰۲ هم‌شماره است؛ شناسه پایدار مشترک پیدا نشد', '۲ رکورد دیگر هم همین شماره را دارند'], allowed: ['نمایش شواهد و حفظ «جزئی»', 'ثبت درخواست اثبات برای مالک حوزه'], notAllowed: ['ادغام بر پایه شماره', 'ساخت CaseRef حدسی'], proof: ['رابطه alias با نوع و مدرک پایدار ثبت شود (OPD-01)'], next: 'مالک حوزه لید/Dot' },
      { id: 'I3', cls: 'coverage', title: '«کل پرونده‌ها» فقط لیدهای قدیمی را می‌شمارد (F03)', entity: ['report', 'rp1', 'گزارش مرور MIS'], owner: 'مالک گزارش / منبع', conf: 'verified', state: 'open', sev: 'amber',
        evidence: ['منابع شمرده‌شده: لید قدیمی', 'منابع شمرده‌نشده: V4، MIS، Dot'], allowed: ['علامت «ناقص» روی شاخص', 'نمایش ماتریس پوشش'], notAllowed: ['صفر فرض‌کردن منابع نشمرده', 'اختراع فرمول اتحاد'], proof: ['اتحاد منطقی پرونده با alias اثبات‌شده یا برچسب «فقط لیدهای قدیمی»'], next: 'مهندسی گزارش' },
      { id: 'I4', cls: 'dup', title: 'ردیف تکراری در همان فایل', entity: ['case', '884206', 'ردیف MIS ۸۸۴۲۰۶'], owner: 'MIS (منبع)', conf: 'verified', state: 'open', sev: 'slate',
        evidence: ['همان شماره در ردیف ۳ و ۱۷ فایل F-2041', 'هر دو ذخیره‌اند؛ هیچ ادغامی انجام نشده'], allowed: ['خواندن ردیابی تکرار', 'پذیرش یا نادیده‌گرفتن طبق سیاست ورود'], notAllowed: ['ادغام یا بازنویسی هویت', 'پاک‌سازی بدون بررسی وابستگی'], proof: ['شمار باقی‌مانده و ارجاع‌های اولیه حفظ شده'], next: 'MIS' },
      { id: 'I5', cls: 'alias', title: 'alias بین ردیف MIS و لید قدیمی حل‌نشده', entity: ['case', '884207', 'ردیف MIS ۸۸۴۲۰۷'], owner: 'مالک حل‌کننده', conf: 'unknown', state: 'awaiting', sev: 'slate',
        evidence: ['شماره یکسان با ردیف ۸۷۹۰۱۰ دسته B-1180', 'نوع رابطه ثبت نشده'], allowed: ['حفظ ورودی‌ها و «نامشخص»'], notAllowed: ['ادغام اجباری', 'ساخت alias جعلی'], proof: ['اثبات ادامه یا جدا بودن توسط مالک حوزه، قابل ردیابی'], next: 'مالک حوزه' },
      { id: 'I6', cls: 'custody', title: 'رویداد انتقال کامل ثبت نشده', entity: ['case', '884205', 'ردیف MIS ۸۸۴۲۰۵'], owner: 'مالک تحویل', conf: 'unknown', state: 'open', sev: 'amber',
        evidence: ['مسئول فعلی: سرپرست آرش یگانه', 'عامل انتقال و گیرنده اولیه در سابقه نیست'], allowed: ['مقایسه تخصیص منبع / استخر / آیتم / سابقه'], notAllowed: ['تخصیص مجدد دلخواه', 'حدس مالک اولیه'], proof: ['زنجیره انتقال معتبر با عامل و زمان'], next: 'مالک تحویل (فروش)' },
      { id: 'I7', cls: 'hier', title: 'دریافت تاریخی مدیر ۰ ولی ساختار امروز همه سطوح را دارد', entity: ['report', 'rp2', 'گزارش تحویل'], owner: 'منابع انسانی', conf: 'partial', state: 'awaiting', sev: 'slate',
        evidence: ['تحویل تاریخی به فروشنده: ۱', 'دریافت تاریخی مدیر: ۰', 'ساختار فعلی: مدیر ← سرپرست ارشد ← سرپرست ← فروشنده'], allowed: ['گزارش تناقض بدون ویرایش'], notAllowed: ['ویرایش سلسله‌مراتب (HR)', 'بازنویسی انتساب تاریخی'], proof: ['HR سابقه مؤثر را تأیید کند؛ انتساب اولیه حفظ شود'], next: 'منابع انسانی' },
      { id: 'I8', cls: 'invrep', title: 'گزارش مدیر ۰ و فاکتورها ۱ (F04)', entity: ['report', 'rp1', 'گزارش مرور MIS'], owner: 'مالک گزارش / مالی', conf: 'conflict', state: 'open', sev: 'orange',
        evidence: ['گزارش مدیر (تازه): ۰ فاکتور', 'نمونه MIS: ۱ فاکتور با همان بازه', 'علت: تأیید نشده'], allowed: ['هم‌ترازی واحد شمارش، زمان و منبع', 'علامت مغایرت'], notAllowed: ['اصلاح مبلغ', 'صفر یا یک جعلی'], proof: ['همان محدوده، گروه، زمان و منبع؛ توضیح دقیق اختلاف'], next: 'مالک گزارش' },
      { id: 'I9', cls: 'stale', title: 'گزارش ذخیره‌شده قدیمی است', entity: ['report', 'rp3', 'کیفیت منبع و ورود'], owner: 'سرویس گزارش', conf: 'verified', state: 'open', sev: 'amber',
        evidence: ['آخرین گردآوری: دیروز ۱۸:۱۰', 'منبع تغییرات بعد از آن ثبت شده'], allowed: ['بازسازی خواندنی گزارش (کار فنی)'], notAllowed: ['نمایش به‌عنوان وضعیت تاریخی', 'تغییر داده منبع'], proof: ['بازه گردآوری جدید و پوشش اعلام‌شده'], next: 'سرویس گزارش' },
      { id: 'I10', cls: 'partial', title: 'اجرای ورود R-9921 ناقص تمام شد', entity: ['run', 'R-9921', 'اجرای R-9921'], owner: 'MIS (منبع)', conf: 'verified', state: 'open', sev: 'orange',
        evidence: ['۴۸۰ دیده‌شده = ۴۱۲ واردشده + ۳۱ تکراری + ۲۲ نامعتبر + ۹ ناموفق + ۶ نامعلوم'], allowed: ['خواندن نتیجه هر ردیف', 'تکرار فقط ردیف‌های ناموفق معلوم'], notAllowed: ['ورود مجدد کل فایل', 'تکرار ردیف‌های نامعلوم قبل از تطبیق'], proof: ['شمارها پس از تطبیق با هم بخوانند؛ ورود دوباره ثبت نشده'], next: 'MIS' },
      { id: 'I11', cls: 'unknownw', title: 'نتیجه نوشتن ۸۰ ردیف R-9908 نامعلوم است', entity: ['run', 'R-9908', 'اجرای R-9908'], owner: 'سرویس عملیات', conf: 'unknown', state: 'open', sev: 'amber',
        evidence: ['ارتباط در ردیف ۲۲۱ قطع شد', 'ثبت‌شده: ۲۱۲ · ناموفق: ۸ · نامعلوم: ۸۰'], allowed: ['بازخوانی لاگ و شناسه‌های نتیجه'], notAllowed: ['تکرار کور', 'فرض موفقیت یا شکست'], proof: ['نتیجه قطعی با شناسه ارجاع برای هر ردیف'], next: 'سرویس عملیات' },
      { id: 'I12', cls: 'orphan', title: 'آیتم V4 ۷۸۰۲ ردیف MIS ندارد (شبه‌یتیم)', entity: ['case', '884204', 'مرتبط با ردیف ۸۸۴۲۰۴'], owner: 'مالک حوزه V4', conf: 'partial', state: 'awaiting', sev: 'slate',
        evidence: ['در نمای MIS دیده نمی‌شود', 'به این معنا نیست که استفاده‌نشده است'], allowed: ['بررسی ارتباط‌های سخت و سابقه'], notAllowed: ['حذف به‌خاطر نبود در نما', 'نام «یتیم» به‌عنوان نتیجه'], proof: ['مالک حوزه رابطه محافظت‌شده را تأیید کند'], next: 'مالک V4' },
      { id: 'I13', cls: 'health', title: 'هشدار سلامت ورود، درحالی‌که گزارش خوانا است', entity: ['health', 'H1', 'تشخیص سلامت ساختار'], owner: 'مالک تشخیص فنی', conf: 'unknown', state: 'open', sev: 'amber',
        evidence: ['آخرین بررسی: امروز ۱۳:۴۰ (ذخیره‌شده)', 'خواندن زنده گزارش موفق است', 'علت: تأیید نشده'], allowed: ['نمایش زمان و منبع بررسی'], notAllowed: ['ترمیم کور', 'نمایش «ورود ۰» به‌جای نامعلوم'], proof: ['تشخیص تازه با علت و منبع'], next: 'مالک فنی' },
      { id: 'I14', cls: 'semantic', title: 'بازه رویداد با وضعیت جاری گردآوری اشتباه «تاریخی» خوانده نشود (F09)', entity: ['report', 'rp1', 'گزارش مرور MIS'], owner: 'مالک معنایی گزارش', conf: 'verified', state: 'open', sev: 'slate',
        evidence: ['گروه پرونده‌ها با رویداد در بازه انتخاب شده', 'وضعیت‌ها در زمان گردآوری خوانده شده‌اند', 'گردآوری مرحله‌ای؛ برش اتمیک یک لحظه نیست'], allowed: ['نمایش دو زمان جدا'], notAllowed: ['ادعای «وضعیت تاریخی»'], proof: ['وضعیت تاریخی فقط با شواهد بازسازی'], next: 'مالک گزارش' }
    ],

    /* ---------- Reports ---------- */
    reports: [
      { id: 'rp1', name: 'مرور MIS (سطح ردیف منبع)', grain: 'ردیف منبع MIS', basis: 'range', cohort: 'ردیف‌هایی با رویداد در بازه', scope: 'محدوده MIS · همه دسته‌های مجاز', ver: 'v۱ (شناسه تعریف)', bounded: true, fresh: 'ok',
        gather: ['۱۴۰۵/۰۷/۱۶ · ۱۴:۱۲', '۱۴۰۵/۰۷/۱۶ · ۱۴:۲۰'],
        cov: [{ label: 'لید قدیمی', state: 'counted' }, { label: 'آیتم V4', state: 'notcounted', note: 'در این گزارش نیست' }, { label: 'ردیف MIS', state: 'counted' }, { label: 'پرونده/فاکتور Dot', state: 'unknown', note: 'پوشش آداپتور اعلام نشده' }],
        metrics: [
          { name: 'ردیف منبع یکتا', value: '۱۳', cov: 'ok', why: 'فقط شناسه ردیف MIS؛ پرونده منطقی سراسری نیست', basis: 'range' },
          { name: 'تحویل تاریخی به فروشنده', value: '۴', cov: 'ok', why: 'رویداد تحویل در بازه', basis: 'range' },
          { name: 'دریافت تاریخی مدیر', value: '۰', cov: 'recon', covLabel: 'ثبت‌شده؛ نیازمند تطبیق', why: 'تحویل مستقیم به فروشنده؛ ساختار امروز با سابقه یکی نیست (I7)', basis: 'range' },
          { name: 'فاکتور یکتای مرتبط', value: '۳', cov: 'partial', covLabel: 'پوشش ناقص', why: 'Dot پوشش داده نشده؛ مجموع نهایی نیست', basis: 'range' },
          { name: 'پیش‌فاکتور باز', value: null, cov: 'recon', covLabel: 'نیازمند تطبیق (F02)', why: 'شرط شمارش pre_invoice در بعضی نماها جا افتاده؛ صفر نمایش داده نمی‌شود', basis: 'snap' },
          { name: 'پیش‌فاکتور صادرشده', value: '۲', cov: 'partial', covLabel: 'سابقه ناقص', why: '۱ مورد بدون سابقه صدور؛ ناشناخته می‌ماند', basis: 'range' },
          { name: 'فروش تکمیل‌شده', value: null, cov: 'undef', covLabel: 'تعریف نهایی نشده', why: 'تکمیل و انتساب (OPD-06/07) باز است', basis: 'range' },
          { name: 'کل پرونده‌ها', value: null, cov: 'partial', covLabel: 'پوشش ناقص (F03)', why: 'اتحاد منطقی وجود ندارد؛ فقط لید قدیمی', basis: 'snap' },
          { name: 'فقط لیدهای قدیمی', value: '۷', cov: 'ok', covLabel: 'فقط منبع قدیمی', why: 'جایگزین «کل پرونده‌ها» نیست', basis: 'snap' }
        ] },
      { id: 'rp2', name: 'تحویل و سابقه مسئولیت', grain: 'رویداد تحویل', basis: 'range', cohort: 'تحویل‌های ثبت‌شده در بازه', scope: 'محدوده MIS', ver: 'v۱', bounded: false, fresh: 'ok', gather: ['۱۴۰۵/۰۷/۱۶ · ۱۴:۰۰', '۱۴۰۵/۰۷/۱۶ · ۱۴:۰۵'],
        cov: [{ label: 'لید قدیمی', state: 'notcounted', note: 'رویداد تحویل ندارد' }, { label: 'آیتم V4', state: 'counted' }, { label: 'ردیف MIS', state: 'counted' }, { label: 'Dot', state: 'unknown', note: 'اثبات نشده' }],
        metrics: [{ name: 'تحویل ثبت‌شده', value: '۹', cov: 'ok', why: 'هر رویداد یک بار', basis: 'range' }, { name: 'عامل رویداد نامشخص', value: '۱', cov: 'partial', covLabel: 'UNKNOWN', why: 'صفر نیست؛ سابقه ناقص', basis: 'range' }] },
      { id: 'rp3', name: 'کیفیت منبع و ورود', grain: 'دسته ورود', basis: 'snap', cohort: 'همه دسته‌ها', scope: 'محدوده MIS', ver: 'v۱', bounded: false, fresh: 'stale', gather: ['دیروز ۱۸:۰۵', 'دیروز ۱۸:۱۰'],
        cov: [{ label: 'ردیف MIS', state: 'counted' }, { label: 'لید قدیمی', state: 'counted' }, { label: 'V4', state: 'notcounted' }, { label: 'Dot', state: 'notcounted' }],
        metrics: [{ name: 'ردیف معتبر', value: '۸٬۴۱۱', cov: 'stale', why: 'گردآوری دیروز', basis: 'snap' }, { name: 'تکراری / نامعتبر', value: '۱٬۰۲۴ / ۳۸۰', cov: 'stale', why: 'قدیمی', basis: 'snap' }] },
      { id: 'rp4', name: 'پوشش وابستگی مالی (فقط خواندن)', grain: 'ارتباط مالی', basis: 'snap', cohort: 'ردیف‌های تحویل‌شده', scope: 'فقط نشانگر وابستگی؛ بدون مبلغ', ver: 'v۱', bounded: true, fresh: 'ok', gather: ['۱۴۰۵/۰۷/۱۶ · ۱۴:۱۵', '۱۴۰۵/۰۷/۱۶ · ۱۴:۲۰'],
        cov: [{ label: 'لید قدیمی', state: 'counted' }, { label: 'V4 follow_invoice', state: 'counted' }, { label: 'جریان فروشنده', state: 'counted' }, { label: 'Dot', state: 'unknown', note: 'اثبات نشده' }],
        metrics: [{ name: 'وابستگی فعال', value: '۲', cov: 'partial', covLabel: 'پوشش ناقص', why: 'Dot پوشش داده نشده', basis: 'snap' }, { name: 'نامعلوم', value: '۲', cov: 'partial', covLabel: 'UNKNOWN', why: 'نامعلوم = مسدود، نه آزاد', basis: 'snap' }] }
    ],

    /* ---------- Planning ---------- */
    plans: [
      { id: 'P-31', name: 'پیش‌نویس توزیع آبان · مدیر فرهاد نجفی', batch: 'B-1187', mgr: 'MG1', rule: 'equal', stage: 'draft', cand: 4, items: null, note: 'فروشنده‌ای برای این مدیر در محدوده نیست — کامل بودن فهرست کاندیدها نیازمند اعتبارسنجی' },
      { id: 'P-30', name: 'توزیع کمپین مهر · مدیر مهسا رضوی', batch: 'B-1186', mgr: 'MG2', rule: 'manual', stage: 'approved', cand: 5, items: null, approvedBy: 'سمیرا کاظمی', approvedAt: 'امروز ۱۱:۰۲', reason: 'بازبینی پیش‌نمایش پایان یافت' },
      { id: 'P-29', name: 'توزیع شهریور · مدیر فرهاد نجفی', batch: 'B-1171', mgr: 'MG1', rule: 'equal', stage: 'materialized', cand: 5, approvedBy: 'سمیرا کاظمی', approvedAt: '۱۴۰۵/۰۷/۰۲', reason: 'مطابق برنامه',
        items: [['C-1', 'ok'], ['C-2', 'ok'], ['C-3', 'ok'], ['C-4', 'ok'], ['C-5', 'ok'], ['C-6', 'ok'], ['C-7', 'ok'], ['C-8', 'ok'], ['C-9', 'ok'], ['C-10', 'ok'], ['C-11', 'ok'], ['C-12', 'ok'], ['C-13', 'ok'], ['C-14', 'ok'], ['C-15', 'ok'], ['C-16', 'ok'], ['C-17', 'ok'], ['C-18', 'ok'], ['C-19', 'skipped', 'لید موجود با همین ردیف'], ['C-20', 'skipped', 'لید موجود با همین ردیف'], ['C-21', 'skipped', 'لید موجود با همین ردیف'], ['C-22', 'skipped', 'تکراری'], ['C-23', 'failed', 'خطای ثبت؛ قابل تکرار'], ['C-24', 'failed', 'خطای ثبت؛ قابل تکرار'], ['C-25', 'unknown', 'پاسخ ثبت نرسید']] }
    ],

    /* ---------- Maintenance catalogue ---------- */
    maint: [
      { id: 'm1', zone: 'maint', label: 'حذف فایل منبع', icon: 'file', target: 'فایل F-2038 · legacy-fix-resend.csv', cls: 'delete_file', auth: 'DELETE_FILE', desc: 'فایل بارگذاری‌شده؛ رکوردها و منشأ آن حذف نمی‌شوند',
        impact: ['۱ فایل بارگذاری‌شده', '۰ ردیف منبع حذف می‌شود (فایل ≠ رکوردها)'], affected: [['فایل', 'F-2038', '۱'], ['ردیف منبع', 'B-1180', '۳۰۰ (دست‌نخورده)']], prot: [['ردیف‌ها', 'نامعلوم', '۸۰ ردیف نتیجه نامعلوم دارند؛ منشأ فایل برای تطبیق لازم است']], block: 'unknown',
        rollback: 'حذف فایل برگشت‌ناپذیر است؛ فایل اصلی باید دوباره بارگذاری شود.' },
      { id: 'm2', zone: 'maint', label: 'حذف کامل دسته', icon: 'layers', target: 'دسته B-1171 · نمایشگاه ۱۴۰۴', cls: 'delete_batch', auth: 'DELETE_BATCH', desc: 'پیش‌بررسی فقط قفل آیتم‌ها را می‌بیند؛ همه ارتباط‌ها را نمی‌بیند',
        impact: ['۲۱۴ ردیف منبع', '۳۶ ردیف مصرف‌شده'], affected: [['ردیف منبع', 'B-1171', '۲۱۴'], ['آیتم توزیع V4', '—', '۱۸۰'], ['لید قدیمی', '—', '۱۱']], prot: [['فاکتور فعال / مصرف', 'مسدود', '۳۶ ردیف'], ['اتصال نامعلوم', 'مسدود', '۶ ردیف']], block: 'protected',
        rollback: 'این عمل رکورد و منشأ را پاک می‌کند؛ برگشت از Git ممکن نیست.' },
      { id: 'm3', zone: 'maint', label: 'پاک‌سازی ردپای تکراری', icon: 'rows', target: 'دسته B-1187 · ۳۱ ردپای تکراری', cls: 'cleanup', auth: 'CLEANUP', desc: 'فقط ردپاهای تکراری بدون ارتباط محافظت‌شده',
        impact: ['۳۱ ردپای تکراری', '۴ مورد با لید قدیمی مرتبط'], affected: [['ردپای تکراری', 'B-1187', '۳۱'], ['لید قدیمی مرتبط', '—', '۴']], prot: [['لید قدیمی مرتبط', 'مسدود', '۴ ردپا'], ['بدون ارتباط', 'قابل بررسی', '۲۷ ردپا']], block: 'partial',
        rollback: 'پاک‌شدن ردپاها برگشت‌ناپذیر است؛ ارجاع اولیه در لاگ باقی می‌ماند.' },
      { id: 'm4', zone: 'maint', label: 'بازسازی ساختار (schema)', icon: 'activity', target: 'جدول‌های ساختار MIS', cls: 'schema', auth: 'RESTRICTED', desc: 'علت هشدار سلامت نامعلوم است؛ بازسازی کور توصیه نمی‌شود',
        impact: ['اثر روی جدول‌های ساختار'], affected: [['جدول ساختار', '—', 'نامعلوم']], prot: [['علت هشدار', 'نامعلوم', 'تشخیص تازه لازم است']], block: 'unknown', rollback: 'روش بازگردانی تعریف نشده.' },
      { id: 'm5', zone: 'maint', label: 'ترمیم اتصال / اتصال مجدد مالی', icon: 'link', target: 'اتصال‌های مالی ردیف‌ها', cls: 'relink', auth: 'NEEDS VALIDATION', desc: 'اتصال مجدد خودکار مالی ممنوع است؛ اختیار عمومی ترمیم برای MIS تعریف نشده', block: 'forbidden', impact: [], affected: [], prot: [], rollback: '' },
      { id: 'm6', zone: 'adv', label: 'تخصیص مجدد استثنایی', icon: 'swap', target: 'ردیف‌های تحویل‌شده', cls: 'reassign', auth: 'NEEDS POLICY', desc: 'سیاست و محافظ مالی مصوب نیست؛ بازپس‌گیری نامحدود نیست', block: 'forbidden', impact: [], affected: [], prot: [], rollback: '' },
      { id: 'm7', zone: 'maint', label: 'بازسازی داده مشتق', icon: 'refresh', target: 'گزارش‌های ذخیره‌شده', cls: 'reprocess', auth: 'NEEDS VALIDATION', desc: 'بازسازی خواندنی گزارش با ترمیم منبع یا بازپخش اثرات تجاری فرق دارد', block: 'forbidden', impact: [], affected: [], prot: [], rollback: '' }
    ],

    /* ---------- Operation log ---------- */
    log: [
      { id: 'L9', at: 'امروز ۱۳:۵۵', actor: 'سمیرا کاظمی', action: 'تخصیص منبع به مدیر', src: 'B-1186', run: 'R-9922', ref: 'SourceRef ۸۸۴۲۰۲ و ۱۱۹ ردیف دیگر', result: 'complete', reason: '—', corr: 'corr-5b81' },
      { id: 'L8', at: 'امروز ۰۹:۱۴', actor: 'سمیرا کاظمی', action: 'ورود فایل', src: 'F-2041', run: 'R-9921', ref: 'B-1187 · ۴۸۰ ردیف دیده‌شده', result: 'partial', reason: '—', corr: 'corr-5a1c' },
      { id: 'L7', at: '۱۴۰۵/۰۷/۱۵ · ۱۵:۱۰', actor: 'سمیرا کاظمی', action: 'تحویل سریع', src: 'B-1186', run: 'Q-77', ref: 'SourceRef ۸۸۴۲۰۳', result: 'complete', reason: 'مسیر استثنایی؛ دلیل ثبت نشد (سیاست مصوب نیست)', corr: 'corr-4f20' },
      { id: 'L6', at: '۱۴۰۵/۰۷/۱۵ · ۱۱:۰۸', actor: 'سمیرا کاظمی', action: 'ورود فایل', src: 'F-2040', run: 'R-9917', ref: 'B-1186 · ۲۲۰ ردیف', result: 'complete', reason: '—', corr: 'corr-4e07' },
      { id: 'L5', at: '۱۴۰۵/۰۷/۱۲ · ۱۶:۴۴', actor: 'سمیرا کاظمی', action: 'ورود فایل', src: 'F-2038', run: 'R-9908', ref: 'B-1180 · ۳۰۰ ردیف', result: 'unknown', reason: 'قطع ارتباط', corr: '—' },
      { id: 'L4', at: '۱۴۰۵/۰۷/۱۱ · ۱۰:۲۰', actor: 'سمیرا کاظمی', action: 'آماده‌سازی استخر', src: 'B-1186', run: 'R-9890', ref: '۴۰ ردیف', result: 'complete', reason: '—', corr: 'corr-4a11' },
      { id: 'L3', at: '۱۴۰۵/۰۷/۰۲ · ۱۰:۳۰', actor: 'سمیرا کاظمی', action: 'ایجاد لید از برنامه', src: 'P-29', run: 'P-29', ref: '۲۵ مورد درخواستی', result: 'partial', reason: 'مطابق برنامه', corr: 'corr-3d90' }
    ],

    /* ---------- Quick delivery ---------- */
    quickBatches: [{ id: 'B-1186', name: 'کمپین مهر', avail: 62 }, { id: 'B-1187', name: 'آبان · اینستاگرام', avail: 412 }],
    positions: [
      { id: 'manager', label: 'مدیر فروش', level: 1 }, { id: 'deputy', label: 'معاون فروش', level: 0 }, { id: 'senior', label: 'سرپرست ارشد', level: 2 }, { id: 'sup', label: 'سرپرست', level: 3 }, { id: 'seller', label: 'فروشنده', level: 4 }
    ]
  };
})();
