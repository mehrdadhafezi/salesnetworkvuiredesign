/* Supervisor content registry for the Shared Help System (../shared/help-engine.js). Tours only navigate and open read-only panels. */
(function () {
  'use strict';
  var C = window.CRM, $ = C.h.$, $$ = C.h.$$;
  var S = function () { return window.SUPAPI; };
  var goView = function (v, fn) { return function () { if (C.state.drawer) C.closeDrawer(); if (fn) fn(); if (C.state.view !== v) C.go(v); else C.render(); }; };
  var noDrawer = function () { if (C.state.drawer) C.closeDrawer(); };
  var cells = function (tbl, from, to) { var t = $(tbl); if (!t || !t.getClientRects().length) return null; return Array.prototype.slice.call(t.tHead.rows[0].cells, from, to); };

  window.CRM_HELP = {
    version: 'سرپرست · نسخه ۱',
    welcome: { key: 'supervisor_welcome_v1_seen', title: 'به پنل سرپرست خوش آمدید', body: 'همان سامانه فروش، با ابزارهای مدیریت تیم: نیازمند توجه، ظرفیت تیم، تخصیص و برگشت امن، صف تبدیل و استثناهای فاکتور. در حدود ۲ دقیقه مسیر کار روزانه را نشان می‌دهیم.' },
    order: ['sup', 'assign', 'return', 'inv'],
    tours: {
      sup: { key: 'supervisor_tour_v1', title: 'آموزش سریع پنل سرپرست', desc: 'بخش‌ها و مسیر کار روزانه تیم', dur: '۲ دقیقه', icon: 'compass', tone: '', steps: [
        { prep: goView('team'), wait: 160, sel: ['#nav', '#bottom-nav'], title: 'ناوبری گروه‌بندی‌شده', body: 'بخش‌ها در سه گروه آمده‌اند: کار تیم، زمینه و درخواست، و بخش شخصی. اگر جا کم باشد، بقیه در «بیشتر» قرار می‌گیرند.' },
        { sel: ['.attn-panel'], title: 'نیازمند توجه', body: 'استثناهای تیم با تعداد و مسئول هر کدام. فقط دسته‌هایی نمایش داده می‌شوند که داده معتبر دارند.' },
        { sel: ['.cap-panel'], title: 'ظرفیت تیم مستقیم', body: 'بار کاری فعلی هر فروشنده برای تصمیم تخصیص. این جدول رتبه‌بندی یا هدف فروش نیست.' },
        { prep: goView('assign', function () { var s = S().st; s.aq = 'assign'; s.asel = {}; s.recip = null; }), wait: 160, sel: ['.flow-head'], title: 'تخصیص در چهار مرحله', body: 'انتخاب پرونده، انتخاب گیرنده، بررسی اثر و ثبت. هیچ تخصیصی بدون مرحله بررسی ثبت نمی‌شود.' },
        { prep: function () { S().preselect(); }, wait: 160, sel: ['.bulkbar'], title: 'انتخاب گروهی', body: 'تعداد انتخاب و گیرنده همیشه دیده می‌شود. پرونده‌های مسدود قابل انتخاب نیستند و دلیلشان کنار ردیف آمده است.' },
        { prep: goView('conv', function () { S().st.cq = 'team'; }), wait: 160, sel: ['.queues'], title: 'صف آماده تبدیل', body: 'صف تیم جدا از تبدیل‌های شخصی شماست. «آماده تبدیل» یعنی پرداخت اعتبارسنجی تأیید شده، نه فروش نهایی.' },
        { prep: goView('inv', function () { S().st.iq = 'action'; }), wait: 160, sel: [function () { return cells('.tbl', 3, 7); }, '.lcard'], title: 'استثناهای فاکتور', body: 'وضعیت فاکتور، مرحله و مدرک، بررسی مالی و مسئول اقدام بعدی جدا نمایش داده می‌شوند؛ «رسید ثبت‌شده» یعنی تأیید پرداخت نیست.' },
        { prep: goView('hr'), wait: 160, sel: ['.hr-list'], title: 'درخواست‌های HR', body: 'مسیر هر درخواست دیده می‌شود. تأیید مرحله‌ای با اعمال نهایی فرق دارد؛ اعمال فقط توسط منابع انسانی انجام می‌شود.' },
        { prep: goView('rep', function () { S().st.report = null; }), wait: 160, sel: ['.rep-grid'], title: 'گزارش‌ها', body: 'گزارش‌های مشترک CRM با محدوده تیم شما. هر گزارش سطح شمارش و مبنای زمانی خودش را دارد.' },
        { prep: noDrawer, sel: ['#help-btn'], title: 'راهنما و میانبرها', body: 'آموزش‌ها، واژه‌ها و میانبرهای کیبورد همیشه از این دکمه در دسترس هستند.' }
      ]},
      assign: { key: 'supervisor_assign_tour_v1', title: 'آموزش تخصیص پرونده', desc: 'انتخاب، گیرنده، بررسی اثر و نتیجه', dur: '۱ دقیقه', icon: 'send', tone: 'teal', steps: [
        { prep: goView('assign', function () { var s = S().st; s.aq = 'assign'; s.amode = 'select'; s.asel = {}; s.recip = null; }), wait: 160, sel: ['.steps'], title: 'مراحل', body: 'مرحله فعلی همیشه مشخص است؛ تا بررسی اثر انجام نشود، چیزی ثبت نمی‌شود.' },
        { sel: ['.flow-head .seg'], title: 'روش تخصیص', body: '«تعداد برای هر گیرنده» همان روش فعلی است؛ «انتخاب پرونده‌ها» برای انتخاب دقیق پرونده‌هاست.' },
        { sel: [function () { var t = $('.tbl.selectable'); if (!t) return null; return [t.tHead.rows[0].cells[5]].concat(Array.prototype.slice.call(t.tBodies[0].rows, 9, 12).map(function (r) { return r.cells[5]; })); }], title: 'قابلیت تخصیص', body: 'هر پرونده دلیل وضعیتش را دارد. پرونده‌های مسدود یا نیازمند تطبیق قابل انتخاب نیستند.' },
        { sel: ['.recips'], title: 'گیرنده', body: 'فقط فروشندگان مستقیم فعال. کنار هر نام بار کاری فعلی دیده می‌شود تا تخصیص آگاهانه باشد.' },
        { prep: function () { S().preselect(); C.openDrawer('assignReview', 'x'); }, wait: 280, sel: ['#drawer'], title: 'بررسی اثر', body: 'تعداد قابل تخصیص، تعارض‌ها و اثر بر گیرنده را می‌بینید. این پیش‌نمایش رزرو نیست و هنگام ثبت دوباره بررسی می‌شود.' }
      ]},
      return: { key: 'supervisor_return_tour_v1', title: 'آموزش برگشت امن پرونده', desc: 'قابلیت برگشت، دلیل و تأیید', dur: '۱ دقیقه', icon: 'repeat', tone: 'violet', steps: [
        { prep: goView('assign', function () { var s = S().st; s.aq = 'return'; s.rf = 'all'; s.rseller = null; }), wait: 160, sel: ['.note.inset'], title: 'قانون برگشت', body: 'پرونده‌هایی که فاکتور، پیش‌فاکتور یا پرداخت دارند با برگشت عادی آزاد نمی‌شوند.' },
        { sel: [function () { var t = $('.tbl.selectable'); if (!t) return null; return [t.tHead.rows[0].cells[6]].concat(Array.prototype.slice.call(t.tBodies[0].rows, 0, 4).map(function (r) { return r.cells[6]; })); }], title: 'قابلیت برگشت و دلیل', body: 'برای هر پرونده وضعیت و دلیل آن نوشته شده است: قابل برگشت، مسدود، تغییر کرده یا نیازمند تطبیق.' },
        { prep: function () { C.openDrawer('ret', 'C-20290'); }, wait: 280, sel: ['#drawer'], title: 'پرونده مسدود', body: 'مسئول فعلی، مالک اولیه، مسئول اقدام بعدی و مالک پورسانت جدا نمایش داده می‌شوند؛ وابستگی مالی با فاکتور مرتبط مشخص است.' },
        { prep: function () { C.openDrawer('ret', 'C-20310'); }, wait: 280, sel: ['#drawer'], title: 'تأیید برگشت', body: 'برای پرونده قابل برگشت، اثر برگشت نمایش داده می‌شود و ثبت فقط پس از تأیید صریح شما فعال است.' }
      ]},
      inv: { key: 'supervisor_inv_tour_v1', title: 'آموزش فاکتورها و استثناها', desc: 'وضعیت مالی تفکیک‌شده و مسئول اقدام', dur: '۱ دقیقه', icon: 'receipt', tone: 'amber', steps: [
        { prep: goView('inv', function () { S().st.iq = 'action'; }), wait: 160, sel: ['.queues'], title: 'صف‌ها', body: '«نیازمند اقدام» اول است؛ بقیه صف‌ها بر اساس وضعیت مالی تفکیک شده‌اند.' },
        { sel: [function () { return cells('.tbl', 3, 6); }, '.lcard'], title: 'وضعیت‌های جدا', body: 'فاکتور، مرحله و مدرک پرداخت، و بررسی مالی سه چیز متفاوت‌اند و در یک برچسب ادغام نمی‌شوند.' },
        { sel: [function () { var t = $('.tbl'); return t ? [t.tHead.rows[0].cells[6]].concat(Array.prototype.slice.call(t.tBodies[0].rows).map(function (r) { return r.cells[6]; })) : null; }, '.lcard .actor'], title: 'اقدام بعدی و مسئول', body: 'مشخص است کار بعدی با کیست: فروشنده، مشتری، واحد مالی یا MIS.' },
        { prep: function () { C.openDrawer('inv', '71942055'); }, wait: 280, sel: ['#drawer'], title: 'جزئیات و اقدام مجاز', body: 'دلیل رد، مراحل، اشتراک قفل‌شده و فقط اقدام‌های مجاز شما. تأیید یا رد پرداخت در اختیار واحد مالی است.' }
      ]}
    },
    glossary: [
      ['مسئول فعلی', 'پرونده الان نزد کیست و چه کسی آن را در دست دارد.'],
      ['مالک اولیه', 'نخستین دریافت‌کننده پرونده؛ با تغییر مسئول عوض نمی‌شود.'],
      ['اقدام بعدی با', 'چه کسی باید کار بعدی را انجام دهد: فروشنده، مشتری، واحد مالی یا MIS.'],
      ['مالک پورسانت', 'طبق قوانین مالی فعلی تعیین می‌شود؛ تخصیص یا برگشت آن را تعیین نمی‌کند.'],
      ['قابل برگشت', 'پرونده‌ای که وابستگی مالی ندارد و می‌تواند به پنل شما برگردد؛ هنگام ثبت دوباره بررسی می‌شود.'],
      ['وابستگی مالی', 'اتصال به فاکتور، پیش‌فاکتور یا پرداخت؛ چنین پرونده‌ای با برگشت عادی آزاد نمی‌شود.'],
      ['پیش‌نمایش', 'بررسی پیش از ثبت؛ پرونده‌ها را رزرو نمی‌کند.'],
      ['ناقص', 'بخشی از عملیات ثبت شد و بخشی نه؛ موفقیت کامل نیست.'],
      ['نامعلوم', 'پاسخ نرسید و ممکن است ثبت شده باشد؛ پیش از تکرار باید تطبیق شود.'],
      ['ثبت رسید', 'بارگذاری مدرک پرداخت؛ به معنی تأیید پرداخت توسط مالی نیست.']
    ],
    keys: [
      [['Ctrl', 'K', 'یا', '/'], 'جستجو و فرمان‌ها'], [['J', 'K'], 'ردیف بعدی / قبلی'], [['Enter'], 'باز کردن ردیف در پنل جزئیات'], [['Esc'], 'بستن پنل جزئیات، منو یا راهنما'],
      [['F'], 'رفتن به جستجوی صف'], [['۱', '–', '۶', '،', '۰'], 'تغییر صف در صفحه فعلی'], [['Space'], 'انتخاب ردیف وقتی روی کادر انتخاب هستید'], [['?'], 'همین راهنما']
    ],
    keysNote: 'پیشنهاد مستند (فعال نیست): X برای انتخاب ردیف جاری و Shift+A برای بررسی انتخاب — پیش از فعال‌سازی باید تصویب شود.',
    toasts: { success: 'درخواست ثبت شد (نمایشی).', error: 'ثبت انجام نشد؛ اتصال را بررسی کنید و دوباره تلاش کنید.', warning: 'وضعیت ۱ پرونده پس از بررسی تغییر کرده است.' },
    demo: [
      ['آموزش', [['welcome', 'نمایش خوش‌آمد'], ['tour:sup', 'آموزش سریع سرپرست'], ['tour:assign', 'آموزش تخصیص'], ['tour:return', 'آموزش برگشت'], ['tour:inv', 'آموزش فاکتورها']]],
      ['وضعیت صفحه', [['sim:loading', 'در حال بارگذاری'], ['sim:empty', 'خالی'], ['sim:noresult', 'بدون نتیجه'], ['sim:tableError', 'خطای جدول'], ['sim:pageError', 'خطای صفحه'], ['sim:drawerError', 'خطای پنل جزئیات'], ['sim:unauthorized', 'بدون دسترسی'], ['sim:stale', 'داده قدیمی'], ['sim:incomplete', 'داده ناقص'], ['sim:offline', 'قطع اتصال']]],
      ['وضعیت عملیات', [['flow:conflict', 'تغییر وضعیت پس از بررسی'], ['flow:partial', 'موفقیت ناقص'], ['flow:retry', 'خطای قابل تکرار'], ['flow:unknown', 'نتیجه نامعلوم'], ['retpick', 'انتخاب برای برگشت'], ['hrerr', 'خطای فرم HR'], ['flow:nomine', 'نمایش/پنهان تبدیل‌های من']]],
      ['اعلان‌ها', [['toast:success', 'Toast موفق'], ['toast:error', 'Toast خطا'], ['toast:warning', 'Toast هشدار']]],
      ['بازنشانی', [['normal', 'بازگشت به حالت عادی'], ['reset', 'Reset onboarding']]]
    ],
    runDemo: function (act) {
      if (act.indexOf('flow:') === 0) { C.setSim(null); S().flow(act.slice(5)); return true; }
      if (act === 'retpick') { C.setSim(null); S().returnPick(); return true; }
      if (act === 'hrerr') { C.setSim(null); C.go('hr'); var b = $('[data-act="hr-submit"]'); if (b) b.click(); return true; }
      if (act === 'sim:drawerError') { C.setSim(null); C.go('inv'); C.state.sim = 'drawerError'; C.openDrawer('inv', '71942055'); return true; }
      return false;
    }
  };
})();
