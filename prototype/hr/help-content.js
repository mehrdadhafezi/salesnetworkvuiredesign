/* HR content registry for the Shared Help System (../shared/help-engine.js). Tours only navigate and open read-only panels — they never perform a business action. */
(function () {
  'use strict';
  var C = window.CRM, $ = C.h.$;
  var S = function () { return window.HRAPI; };
  var goView = function (v, fn) { return function () { if (C.state.drawer) C.closeDrawer(); if (fn) fn(); if (C.state.view !== v) C.go(v); else C.render(); }; };
  var noDrawer = function () { if (C.state.drawer) C.closeDrawer(); };

  window.CRM_HELP = {
    version: 'منابع انسانی · نسخه ۱',
    welcome: { key: 'hr_welcome_v1_seen', title: 'به پنل منابع انسانی خوش آمدید', body: 'این پنل برای چرخه نیرو، ساختار، دسترسی، اعمال درخواست و تاریخچه زمانی است: چه کسی تغییر می‌کند، وضعیت فعلی و درخواستی، تاریخ اثر و زمان اعمال، و چه چیزی هنوز حل‌نشده است. منابع انسانی مدیر فروش، مالی، MIS یا مدیر کل وردپرس نیست و تغییر ساختار فعلی، انتساب و اعتبار گذشته را بازنویسی نمی‌کند. نقش HR هنوز در محیط واقعی تأیید نشده (NOT LIVE VERIFIED) و این نمونه هیچ داده‌ای ثبت نمی‌کند.' },
    order: ['hr', 'transfer', 'term', 'restricted'],
    tours: {
      hr: { key: 'hr_tour_v1', title: 'آموزش سریع پنل منابع انسانی', desc: 'نیروها، ورود، انتقال، درخواست‌ها، دسترسی، جبران خدمات، استثناها و اعتبارنامه', dur: '۴ دقیقه', icon: 'compass', tone: '', steps: [
        { prep: goView('work', function () { S().st.wq = 'all'; }), wait: 160, sel: ['.tbl', '.idcell'], title: '۱. نیروها', body: 'شخص، پروفایل نیرو و حساب وردپرس سه چیز جدا هستند و پیوند هر کدام وضعیت خودش را دارد. سمت، نقش و مدیر فعلی هم جدا هستند.' },
        { prep: goView('onb', function () { S().st.oq = 'onb'; }), wait: 160, sel: ['.onb-st', '.ws-banner'], title: '۲. ورود نیرو', body: 'مراحل مرحله‌ای‌اند ولی گام ساختگی ندارند. تعارض هویت پیش از ادامه باید صریح حل شود و اعتبارنامه جزو ورود نیرو نیست.' },
        { prep: function () { S().st.oq = 'transfer'; C.go('onb'); C.openDrawer('xfer', 'HP-301', { target: 'HP-120' }); }, wait: 300, sel: ['#drawer .exc-dl', '#drawer .recips-inline'], title: '۳. انتقال', body: 'پیش‌نمایش تغییر مدیر: فرد، مدیر فعلی و مقصد، تاریخ اثر درخواستی در برابر زمان اعمال، اثر روی دسترسی و کار باز، و حفظ تاریخچه. پیش‌نمایش ثبت نهایی نیست.' },
        { prep: goView('req', function () { var s = S().st; s.rq = 'review'; }), wait: 160, sel: ['.tbl'], title: '۴. درخواست‌ها', body: 'تأیید مرحله‌ای با تغییر اعمال‌شده یکی نیست؛ درخواستی و واقعی دو ستون جدا دارند و اعمال ناموفق جداگانه می‌ماند.' },
        { prep: goView('acc', function () { S().st.aq = 'matrix'; }), wait: 160, sel: ['.tbl'], title: '۵. ساختار و دسترسی', body: 'سمت کاری، مجوز محصولی، نقش/قابلیت و اعتبارنامه جدا هستند. ماتریس منبع، ارث‌بری یا مستقیم و وضعیت مؤثر را نشان می‌دهد؛ ویرایش قابلیت وردپرس نیست.' },
        { prep: goView('comp'), wait: 160, sel: ['.ws-banner', '.tbl'], title: '۶. جبران خدمات', body: 'فقط برای مجازها و جدا از کیف پول، مالی و موتور کمیسیون. تغییر اینجا اعتبار کسب‌شده گذشته را دوباره محاسبه نمی‌کند.' },
        { prep: goView('exc', function () { S().st.xq = 'exc'; }), wait: 160, sel: ['.tbl'], title: '۷. استثناها', body: 'هر مورد مالک، اقدام مجاز، اقدام ممنوع و مدرک حل دارد. SLA یا فرمان تشدید ساخته نشده است.' },
        { prep: goView('cred'), wait: 160, sel: ['.ws-banner', '.tbl'], title: '۸. اعتبارنامه (محدود)', body: 'مسیر حساس جدا از ویرایش نیرو. نمایش به‌جای کاربر فقط‌خواندنی نیست و هیچ گذرواژه یا رازی نمایش داده نمی‌شود.' },
        { prep: noDrawer, sel: ['#help-btn'], title: 'راهنما و میانبرها', body: 'آموزش‌ها، واژه‌ها و میانبرهای کیبورد همیشه از این دکمه در دسترس هستند.' }
      ]},
      transfer: { key: 'hr_transfer_tour_v1', title: 'آموزش تغییر مدیر', desc: 'پیش‌نمایش اثر و تأیید حساس', dur: '۱ دقیقه', icon: 'swap', tone: 'violet', steps: [
        { prep: function () { C.go('onb'); C.openDrawer('xfer', 'HP-301', { target: 'HP-120' }); }, wait: 300, sel: ['#drawer .exc-dl'], title: 'خلاصه تغییر', body: 'تاریخ اثر درخواستی (پشتیبانی تأیید نشده) و زمان اعمال واقعی دو چیز جدا هستند.' },
        { sel: ['#drawer .elig'], title: 'مسئولیت‌های عملیاتی', body: 'دیده‌شدن زیر مدیر جدید انتقال پرونده یا استحقاق مالی نیست؛ تحویل کار باز (OPD-05) تعریف نشده.' },
        { sel: ['#drawer .dr-foot'], title: 'بعدی: تأیید با بررسی اثر', body: 'چه چیزی تغییر می‌کند، چه چیزی نمی‌کند، موجودیت‌های متأثر، زمان اثر و دلیل ممیزی؛ نه فقط «مطمئنید؟».' }
      ]},
      term: { key: 'hr_term_tour_v1', title: 'آموزش پایان همکاری و درخواست‌ها', desc: 'اثر تحویل و اعمال نهایی', dur: '۱ دقیقه', icon: 'lock', tone: 'teal', steps: [
        { prep: function () { C.go('req'); C.openDrawer('term', 'HP-304'); }, wait: 300, sel: ['#drawer .rva'], title: 'دو چیز جدا', body: 'وضعیت اشتغال و وضعیت دسترسی دو چیز جدا هستند؛ لغو نقش و نشست از این اقدام نتیجه نمی‌شود.' },
        { sel: ['#drawer .tbl'], title: 'کار باز', body: 'مسئول بعدی نامشخص است (OPD-05). بازتخصیص خودکار وجود ندارد و اعمال تا تعیین آن مسدود/مشروط است.' },
        { prep: function () { C.openDrawer('req', 'R-502'); }, wait: 300, sel: ['#drawer .rva', '#drawer .steps'], title: 'درخواستی در برابر اعمال‌شده', body: 'تأیید مرحله‌ای تغییر اعمال‌شده نیست؛ اعمال نهایی گام جدا با بازبینی تازه است.' }
      ]},
      restricted: { key: 'hr_restricted_tour_v1', title: 'آموزش مسیر محدود', desc: 'اعتبارنامه و نمایش به‌جای کاربر', dur: '۱ دقیقه', icon: 'shield', tone: 'orange', steps: [
        { prep: goView('cred'), wait: 160, sel: ['.ws-banner'], title: 'مسیر محدود', body: 'جدا از ویرایش نیرو؛ سیاست دقیق هدف مجاز و ممیزی ماندگار هنوز تعریف نشده است.' },
        { prep: function () { C.openDrawer('impstart', 'HP-302'); }, wait: 300, sel: ['#drawer .elig'], title: 'نمایش به‌جای کاربر', body: 'فقط‌خواندنی نیست: عامل، هدف، هدف جلسه، جلسه فعال، بازگشت و الزام ممیزی روشن نمایش داده می‌شود.' },
        { prep: function () { C.openDrawer('credreset', 'HP-301'); }, wait: 300, sel: ['#drawer .form-grid'], title: 'بازنشانی گذرواژه', body: 'مقدار گذرواژه هرگز نمایش یا ثبت نمی‌شود؛ بازنشانی و پیامک دو نتیجه جدا هستند.' }
      ]}
    },
    glossary: [
      ['شخص / پروفایل نیرو / حساب وردپرس', 'سه شناسه مستقل. برابری عدد در دو جدول یکی بودن نیست؛ هویت شخص سراسری هنوز تعریف نشده.'],
      ['سمت / سطح / نقش / مجوز محصولی / اعتبارنامه', 'سمت کاری، طبقه، نقش فنی، نتیجه دسترسی و ورود، پنج چیز جدا.'],
      ['مدیر فعلی در برابر مدیر تاریخی', 'مدیر فعلی ساختار امروز است و اعتبار گذشته را تعیین نمی‌کند؛ نبود سابقه «نامعلوم» است.'],
      ['تاریخ اثر / زمان اعمال / عامل رویداد', 'تاریخ اثر درخواستی، زمان واقعی اعمال و کسی که انجام داده، سه چیز جدا.'],
      ['تأیید مرحله‌ای ≠ اعمال‌شده', 'تأیید یک مرحله بررسی فقط مسیر را جلو می‌برد؛ اعمال نهایی با منابع انسانی و جدا است.'],
      ['اعمال ناموفق', 'وضعیت جدا با اثر واقعی نامعلوم؛ اعمال دوباره کور ممنوع است و ابتدا اثر خوانده می‌شود.'],
      ['پایان همکاری ≠ غیرفعال‌کردن کاربر', 'وضعیت اشتغال، دسترسی و تحویل کار سه چیز جدا هستند؛ تحویل کامل بدون گیرنده تعریف‌شده (OPD-05) ممکن نیست.'],
      ['OPD-05', 'گیرنده و زمان تحویل کار باز هنوز تعیین نشده؛ هیچ بازتخصیص خودکار وجود ندارد.'],
      ['ماتریس وضعیت دسترسی', 'برای هر مجوز: منبع، ارث‌بری یا مستقیم، وضعیت مؤثر و دلیل استثنا.'],
      ['اعتبارنامه / نمایش به‌جای کاربر', 'مسیر محدود حساس؛ نمایش به‌جای کاربر فقط‌خواندنی نیست.'],
      ['جبران خدمات', 'پایه، واحد و دوره اثر؛ جدا از کیف پول، مالی و موتور کمیسیون.'],
      ['درخواست‌شده / واجد شرایط / اعمال‌شده / ارسال‌نشده / ردشده / ناموفق / نامعلوم', 'زبان نتیجه گروهی؛ جمع باید با درخواست‌شده بخواند.'],
      ['نامعلوم', 'پاسخ نرسید و ممکن است ثبت شده باشد؛ پیش از تکرار باید تطبیق شود.']
    ],
    keys: [
      [['Ctrl', 'K', 'یا', '/'], 'جستجو و فرمان‌ها'], [['J', 'K'], 'ردیف بعدی / قبلی'], [['Enter'], 'باز کردن ردیف در پنل جزئیات'], [['Esc'], 'بستن پنل جزئیات، منو یا راهنما'],
      [['F'], 'رفتن به جستجوی صف'], [['۱', '–', '۴'], 'تغییر صف در صفحه فعلی'], [['Space'], 'انتخاب وقتی روی کادر انتخاب هستید'], [['?'], 'همین راهنما']
    ],
    keysNote: 'پیشنهاد مستند (فعال نیست): X برای انتخاب ردیف جاری و Shift+A برای بررسی انتخاب — پیش از فعال‌سازی باید تصویب شود.',
    toasts: { success: 'درخواست ثبت شد (نمایشی).', error: 'ثبت انجام نشد؛ اتصال را بررسی کنید و دوباره تلاش کنید.', warning: 'وضعیت درخواست پس از بررسی تغییر کرده است.' },
    demo: [
      ['آموزش', [['welcome', 'نمایش خوش‌آمد'], ['tour:hr', 'آموزش سریع منابع انسانی'], ['tour:transfer', 'آموزش تغییر مدیر'], ['tour:term', 'آموزش پایان همکاری'], ['tour:restricted', 'آموزش مسیر محدود']]],
      ['وضعیت صفحه', [['sim:loading', 'در حال بارگذاری'], ['sim:empty', 'خالی'], ['sim:noresult', 'بدون نتیجه'], ['sim:tableError', 'خطای جدول'], ['sim:pageError', 'خطای صفحه'], ['sim:drawerError', 'خطای پنل جزئیات'], ['sim:unauthorized', 'بدون دسترسی'], ['sim:offline', 'آفلاین'], ['sim:stale', 'فهرست قدیمی'], ['sim:incomplete', 'داده ناقص'], ['sim:reqchanged', 'درخواست پیش از بررسی تغییر کرد'], ['sim:targetinactive', 'مدیر مقصد غیرفعال'], ['sim:dupprofile', 'پروفایل تکراری'], ['sim:accesspartial', 'اعمال دسترسی ناقص'], ['sim:termincomplete', 'تحویل پایان همکاری ناتمام'], ['sim:effconflict', 'تعارض تاریخ اثر'], ['sim:histunavail', 'تاریخچه انتساب در دسترس نیست'], ['sim:credpartial', 'بازنشانی موفق / اعلان ناموفق'], ['flow:compdenied', 'جبران خدمات: بدون دسترسی']]],
      ['وضعیت عملیات', [['flow:onb', 'ورود نیرو (تعارض هویت)'], ['flow:xfer', 'پیش‌نمایش تغییر مدیر'], ['flow:targetinactive', 'مدیر مقصد غیرفعال'], ['flow:effconflict', 'تعارض تاریخ اثر'], ['flow:xpartial', 'تغییر مدیر: اعمال دسترسی ناقص'], ['flow:xunknown', 'تغییر مدیر: نتیجه نامعلوم'], ['flow:term', 'پایان همکاری: تحویل ناتمام'], ['flow:termclear', 'پایان همکاری: بدون کار باز'], ['flow:reqstep', 'درخواست: تأیید مرحله‌ای'], ['flow:reqapply', 'درخواست: اعمال نهایی'], ['flow:applyfail', 'درخواست: اعمال ناموفق'], ['flow:reqfailed', 'درخواست ناموفق: اثر نامعلوم'], ['flow:reqtermblock', 'درخواست پایان همکاری مسدود'], ['flow:reqchanged', 'درخواست تغییر کرده'], ['flow:reqoos', 'درخواست خارج از محدوده'], ['flow:access', 'مقایسه دسترسی'], ['flow:accpartial', 'تغییر سمت: دسترسی ناقص'], ['flow:comp', 'تاریخچه جبران خدمات'], ['flow:compunknown', 'جبران خدمات: نتیجه نامعلوم'], ['flow:cred', 'بازنشانی گذرواژه'], ['flow:credlead', 'بازنشانی: هدف مدیریتی'], ['flow:credpartial', 'بازنشانی: پیامک ناموفق'], ['flow:imp', 'نمایش به‌جای کاربر'], ['flow:impactive', 'جلسه نمایش فعال'], ['flow:bulkwf', 'گروهی: به‌روزرسانی نیروها'], ['flow:bulkcred', 'گروهی: بازنشانی (محدود)']]],
      ['اعلان‌ها', [['toast:success', 'Toast موفق'], ['toast:error', 'Toast خطا'], ['toast:warning', 'Toast هشدار']]],
      ['بازنشانی', [['normal', 'بازگشت به حالت عادی'], ['reset', 'Reset onboarding']]]
    ],
    runDemo: function (act) {
      var s = S().st;
      if (act.indexOf('flow:') === 0) { C.setSim(null); s.flow = null; S().flow(act.slice(5)); return true; }
      var VIEW = { stale: 'work', incomplete: 'work', reqchanged: 'req', targetinactive: 'onb', dupprofile: 'work', accesspartial: 'acc', termincomplete: 'req', effconflict: 'onb', histunavail: 'work', credpartial: 'cred' };
      var k = act.indexOf('sim:') === 0 ? act.slice(4) : null;
      if (k && VIEW[k]) { C.setSim(null); s.flow = null; if (k === 'reqchanged') s.rq = 'chain'; if (k === 'targetinactive' || k === 'effconflict') s.oq = 'transfer'; C.state.sim = k; C.go(VIEW[k]); return true; }
      if (act === 'sim:drawerError') { C.setSim(null); C.go('work'); C.state.sim = 'drawerError'; C.openDrawer('staff', 'HP-303'); return true; }
      if (act === 'normal') { s.compAuth = 'authorized'; s.flow = null; s.imp = null; return false; }
      return false;
    }
  };
})();
