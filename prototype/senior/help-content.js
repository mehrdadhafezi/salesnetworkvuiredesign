/* Senior Supervisor content registry for the Shared Help System (../shared/help-engine.js). Tours only navigate and open read-only panels — they never perform a business action. */
(function () {
  'use strict';
  var C = window.CRM, $ = C.h.$;
  var S = function () { return window.SENAPI; };
  var goView = function (v, fn) { return function () { if (C.state.drawer) C.closeDrawer(); if (fn) fn(); if (C.state.view !== v) C.go(v); else C.render(); }; };
  var noDrawer = function () { if (C.state.drawer) C.closeDrawer(); };
  var cells = function (tbl, from, to) { var t = $(tbl); if (!t || !t.getClientRects().length) return null; return Array.prototype.slice.call(t.tHead.rows[0].cells, from, to); };

  window.CRM_HELP = {
    version: 'سرپرست ارشد · نسخه ۱',
    welcome: { key: 'senior_welcome_v1_seen', title: 'به پنل سرپرست ارشد خوش آمدید', body: 'این پنل برای هماهنگی چند تیم است: ببینید کدام تیم و سرپرست مسئول چه کاری است، پرونده‌های خود را به سرپرستان توزیع کنید و استثناها را به مسئول درست برسانید. دیدن زیرمجموعه به معنی اجازه نوشتن برای آن‌ها نیست. در حدود ۳ دقیقه مسیر کار روزانه را نشان می‌دهیم.' },
    order: ['senior', 'perf', 'dist'],
    tours: {
      senior: { key: 'senior_tour_v1', title: 'آموزش سریع پنل سرپرست ارشد', desc: 'تیم‌ها، توزیع، آماده تبدیل، فاکتور، HR و گزارش', dur: '۳ دقیقه', icon: 'compass', tone: '', steps: [
        { prep: goView('ov'), wait: 160, sel: ['#nav', '#bottom-nav'], title: 'ناوبری گروه‌بندی‌شده', body: 'بخش‌ها در سه گروه آمده‌اند: کار تیم‌ها، زمینه و درخواست، و بخش شخصی. اگر جا کم باشد، بقیه در «بیشتر» می‌روند. تبدیل‌های شخصی فقط در صورت مجوز دیده می‌شود.' },
        { sel: ['.attn-panel'], title: 'نیازمند هماهنگی', body: 'هر استثنا موضوع، مسئول اقدام، وضعیت، دلیل و اقدام مجاز شما را دارد. دستور ارجاع رسمی هنوز تعریف نشده و ساخته نمی‌شود.' },
        { sel: ['.cap-panel'], title: 'تیم‌های تحت پوشش', body: 'هر ردیف یک تیم و سرپرست مسئول آن است. ستون «اعتماد به داده» نشان می‌دهد آمار کامل، قدیمی، ناقص یا نیازمند تطبیق است. این جدول رتبه‌بندی نیست.' },
        { prep: goView('perf', function () { var s = S().st; s.pm = 'team'; s.pb = 'current'; s.pscope = null; }), wait: 160, sel: ['.explorer-ctl'], title: 'کاوشگر ساختار و عملکرد', body: 'هر بعد یک واحد شمارش دارد و با بعد دیگر مخلوط نمی‌شود. مبنای انتساب (ساختار فعلی یا تاریخی) همیشه دیده می‌شود.' },
        { prep: goView('dist', function () { var s = S().st; s.aq = 'assign'; s.amode = 'select'; s.asel = {}; s.recip = null; }), wait: 160, sel: ['.recips'], title: 'توزیع: مسیر اصلی سرپرست است', body: 'گیرنده اصلی سرپرستان هستند. مسیر «مستقیم به فروشنده» استثنایی و مشروط است و سیاست آن تعریف نشده.' },
        { prep: goView('ready'), wait: 160, sel: ['.tbl'], title: 'آماده‌های تبدیل (فقط‌خواندنی)', body: 'اینجا فقط می‌بینید مسئول هر پرونده کیست. تخصیص تبدیل از این پنل ممکن نیست و دکمه ارجاعی وجود ندارد.' },
        { prep: goView('inv', function () { S().st.iq = 'action'; }), wait: 160, sel: [function () { return cells('.tbl', 2, 6); }, '.lcard'], title: 'فاکتور، مرحله و بررسی مالی', body: 'وضعیت فاکتور، مرحله پرداخت و بررسی مالی سه چیز جدا هستند. تأیید، رد یا بازپرداخت مالی در اختیار واحد مالی است.' },
        { prep: goView('hr', function () { S().st.hq = 'review'; }), wait: 160, sel: ['.hr-list'], title: 'درخواست‌های HR', body: 'تأیید شما فقط یک مرحله از مسیر است؛ اعمال نهایی را منابع انسانی انجام می‌دهد. بررسی گروهی وجود ندارد.' },
        { prep: goView('rep', function () { S().st.report = null; }), wait: 160, sel: ['.rep-ctrl'], title: 'گزارش‌ها', body: 'محدوده، مبنای زمانی، پوشش و تازگی هر گزارش مشخص است. خروجی فقط با مجوز صریح ممکن است.' },
        { prep: noDrawer, sel: ['#help-btn'], title: 'راهنما و میانبرها', body: 'آموزش‌ها، واژه‌ها و میانبرهای کیبورد همیشه از این دکمه در دسترس هستند.' }
      ]},
      perf: { key: 'senior_perf_tour_v1', title: 'آموزش کاوشگر عملکرد', desc: 'بعدها، مبنای انتساب و محدوده', dur: '۱ دقیقه', icon: 'chart', tone: 'violet', steps: [
        { prep: goView('perf', function () { var s = S().st; s.pm = 'team'; s.pb = 'current'; s.pscope = null; }), wait: 160, sel: ['.dim-switch'], title: 'بعد تحلیل', body: 'تیم، سرپرست، فروشنده، تبدیل، فاکتور و بار پرونده هر کدام واحد شمارش خودش را دارند؛ جمع و مقایسه بین آن‌ها معنا ندارد.' },
        { sel: ['.explorer-ctl .seg'], title: 'ساختار فعلی یا انتساب تاریخی', body: 'ساختار فعلی برای بار کاری امروز است. انتساب تاریخی هر فاکتور را به تیمی می‌دهد که هنگام صدور ثبت شده؛ سابقه‌نداشته «ناشناخته» می‌ماند.' },
        { prep: function () { S().st.pb = 'hist'; C.render(); }, wait: 160, sel: ['.unk-row'], title: 'ناشناخته به والد امروز تبدیل نمی‌شود', body: 'اگر سابقه تیم ثبت نشده باشد، ردیف «ناشناخته» می‌ماند و جمع یکتا همچنان درست است.' },
        { prep: function () { S().st.pb = 'current'; S().st.pm = 'seller'; C.render(); }, wait: 160, sel: ['.scope-crumb'], title: 'مسیر محدوده', body: 'هر drilldown محدوده را به زیرمجموعه‌ای از حوزه مجاز شما محدود می‌کند و هرگز آن را وسیع‌تر نمی‌کند.' },
        { sel: [function () { var t = $('.tbl'); return t ? [t.tHead.rows[0].cells[3]] : null; }], title: 'فقط لیدهای قدیمی', body: 'این ستون «کل پرونده‌ها» نیست. پوشش منابع جدید هنوز تأیید نشده است.' }
      ]},
      dist: { key: 'senior_dist_tour_v1', title: 'آموزش توزیع و برگشت', desc: 'گیرنده اصلی، مسیر استثنایی و برگشت امن', dur: '۲ دقیقه', icon: 'send', tone: 'teal', steps: [
        { prep: goView('dist', function () { var s = S().st; s.aq = 'assign'; s.amode = 'select'; s.asel = {}; s.recip = null; s.exc = false; }), wait: 160, sel: ['.steps'], title: 'مراحل', body: 'تا بررسی اثر انجام نشود چیزی ثبت نمی‌شود و پیش‌نمایش ثبت را تضمین نمی‌کند.' },
        { sel: ['.recips'], title: 'گیرنده', body: 'سرپرستان گیرنده اصلی هستند. سرپرست غیرفعال انتخاب‌شدنی نیست.' },
        { prep: function () { S().st.exc = true; C.render(); }, wait: 160, sel: ['.exc-path'], title: 'مسیر استثنایی (مشروط)', body: 'عبور از سرپرست عادی نیست. رابطه، سطح ردشده و دلیل نمایش داده می‌شود اما ثبت تا تعیین سیاست غیرفعال است.' },
        { prep: function () { S().st.exc = false; S().st.aq = 'return'; S().st.rf = 'all'; C.render(); }, wait: 160, sel: ['.note.inset'], title: 'قانون برگشت', body: 'فقط انتقال‌های خودتان قابل برگشت است. انتقال سرپرستان و منبع MIS با خودشان است؛ رتبه بالاتر اجازه بازپس‌گیری همه چیز نیست.' },
        { prep: function () { C.openDrawer('ret', 'C-30120'); }, wait: 280, sel: ['#drawer'], title: 'پرونده مسدود', body: 'مسئول فعلی، مالک اولیه، اقدام بعدی، عامل رویداد و مالک اعتبار جدا نمایش داده می‌شوند و مسئول اقدام بعدی معلوم است.' },
        { prep: function () { C.openDrawer('ret', 'C-30110'); }, wait: 280, sel: ['#drawer'], title: 'تأیید برگشت', body: 'برای پرونده قابل برگشت، اثر نمایش داده می‌شود و ثبت فقط پس از تأیید صریح شما فعال است.' }
      ]}
    },
    glossary: [
      ['مسئول فعلی', 'پرونده الان نزد کیست (Current Custody).'],
      ['مالک اولیه', 'نخستین دریافت‌کننده پرونده؛ با تغییر مسئول عوض نمی‌شود.'],
      ['اقدام بعدی با', 'چه کسی باید کار بعدی را انجام دهد: سرپرست، فروشنده، مشتری، واحد مالی یا MIS.'],
      ['عامل رویداد', 'کسی که رویداد (مثلاً انتقال یا صدور) را انجام داده؛ مالک فعلی یا مالک اعتبار نیست.'],
      ['مالک اعتبار', 'طبق قوانین مالی فعلی تعیین می‌شود؛ تخصیص، برگشت یا تغییر والد آن را عوض نمی‌کند.'],
      ['ساختار فعلی', 'درخت سازمانی امروز؛ برای بار کاری فعلی.'],
      ['انتساب تاریخی', 'تیمی که هنگام رویداد ثبت شده؛ ساختار امروز آن را بازنویسی نمی‌کند و ناشناخته می‌تواند بماند.'],
      ['مستقیم / غیرمستقیم', 'سرپرست و فروشنده‌ای که مستقیم زیر نظر شما هستند، در برابر فروشندگان زیر سرپرست. دیدن به معنی اجازه نوشتن نیست.'],
      ['مسیر استثنایی', 'تخصیص مستقیم به فروشنده با عبور از سرپرست؛ مشروط و بدون سیاست تعریف‌شده.'],
      ['پوشش ناقص', 'بخشی از منابع دریافت نشده؛ جمع نهایی نیست و صفر فرض نمی‌شود.'],
      ['نیازمند تطبیق', 'دو منبع اختلاف دارند؛ عدد دستی تغییر نمی‌کند.'],
      ['تعریف نشده', 'تعریف شاخص هنوز تصویب نشده است؛ ۰ نمایش داده نمی‌شود.'],
      ['اطمینان اتصال', 'چقدر مطمئنیم رکورد به همان پرونده/فاکتور وصل است؛ شماره موبایل اثبات هویت پرونده نیست.'],
      ['تأیید این مرحله', 'فقط مرحله بررسی شما را جلو می‌برد؛ اعمال نهایی توسط منابع انسانی جداست.'],
      ['نامعلوم', 'پاسخ نرسید و ممکن است ثبت شده باشد؛ پیش از تکرار باید تطبیق شود.']
    ],
    keys: [
      [['Ctrl', 'K', 'یا', '/'], 'جستجو و فرمان‌ها'], [['J', 'K'], 'ردیف بعدی / قبلی'], [['Enter'], 'باز کردن ردیف در پنل جزئیات'], [['Esc'], 'بستن پنل جزئیات، منو یا راهنما'],
      [['F'], 'رفتن به جستجوی صف'], [['۱', '–', '۶'], 'تغییر صف در صفحه فعلی'], [['Space'], 'انتخاب ردیف وقتی روی کادر انتخاب هستید'], [['?'], 'همین راهنما']
    ],
    keysNote: 'پیشنهاد مستند (فعال نیست): X برای انتخاب ردیف جاری و Shift+A برای بررسی انتخاب — پیش از فعال‌سازی باید تصویب شود.',
    toasts: { success: 'درخواست ثبت شد (نمایشی).', error: 'ثبت انجام نشد؛ اتصال را بررسی کنید و دوباره تلاش کنید.', warning: 'وضعیت ۱ پرونده پس از بررسی تغییر کرده است.' },
    demo: [
      ['آموزش', [['welcome', 'نمایش خوش‌آمد'], ['tour:senior', 'آموزش سریع سرپرست ارشد'], ['tour:perf', 'آموزش کاوشگر عملکرد'], ['tour:dist', 'آموزش توزیع و برگشت']]],
      ['وضعیت صفحه', [['sim:loading', 'در حال بارگذاری'], ['sim:empty', 'خالی'], ['sim:noresult', 'بدون نتیجه'], ['sim:tableError', 'خطای جدول'], ['sim:pageError', 'خطای صفحه'], ['sim:drawerError', 'خطای پنل جزئیات'], ['sim:unauthorized', 'بدون دسترسی'], ['sim:stale', 'داده قدیمی یک تیم'], ['sim:incomplete', 'پوشش ناقص منبع'], ['sim:teamfail', 'یک تیم ناموفق · بقیه بارگذاری'], ['sim:unknownhist', 'انتساب تاریخی ناشناخته'], ['sim:offline', 'قطع اتصال']]],
      ['وضعیت عملیات', [['flow:conflict', 'تغییر وضعیت پس از بررسی'], ['flow:partial', 'موفقیت ناقص'], ['flow:retry', 'خطای قابل تکرار'], ['flow:unknown', 'نتیجه نامعلوم'], ['flow:exception', 'مسیر استثنایی (مستقیم به فروشنده)'], ['retpick', 'انتخاب برای برگشت'], ['hrerr', 'خطای فرم HR'], ['flow:nomine', 'نمایش/پنهان تبدیل‌های من (شرطی)']]],
      ['اعلان‌ها', [['toast:success', 'Toast موفق'], ['toast:error', 'Toast خطا'], ['toast:warning', 'Toast هشدار']]],
      ['بازنشانی', [['normal', 'بازگشت به حالت عادی'], ['reset', 'Reset onboarding']]]
    ],
    runDemo: function (act) {
      if (act.indexOf('flow:') === 0) { C.setSim(null); S().flow(act.slice(5)); return true; }
      if (act === 'retpick') { C.setSim(null); S().returnPick(); return true; }
      if (act === 'hrerr') { C.setSim(null); C.go('hr'); var b = $('[data-act="hr-submit"]'); if (b) b.click(); return true; }
      if (act === 'sim:unknownhist') { C.setSim(null); S().st.pm = 'team'; S().st.pb = 'hist'; C.state.sim = 'unknownhist'; C.go('perf'); return true; }
      if (act === 'sim:drawerError') { C.setSim(null); C.go('inv'); C.state.sim = 'drawerError'; C.openDrawer('inv', '71942055'); return true; }
      return false;
    }
  };
})();
