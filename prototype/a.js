/* Version A — current CRM structure, cleaned up. Demo only: nothing is saved or sent. */
(function () {
  'use strict';
  var M = window.MOCK;
  var $ = function (s, r) { return (r || document).querySelector(s); };
  var $$ = function (s, r) { return Array.prototype.slice.call((r || document).querySelectorAll(s)); };
  var esc = function (v) { return String(v == null ? '' : v).replace(/[&<>"']/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]; }); };
  var fa = function (v) { return String(v).replace(/\d/g, function (d) { return '۰۱۲۳۴۵۶۷۸۹'[d]; }); };
  var num = function (n) { return Number(n || 0).toLocaleString('fa-IR'); };
  var amt = function (n) { return '<span class="amt"><span class="num">' + num(n) + '</span><span class="u">تومان</span></span>'; };
  var P = {
    copy: '<rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>',
    users: '<path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/>',
    swap: '<path d="m16 3 4 4-4 4M20 7H4M8 21l-4-4 4-4M4 17h16"/>',
    repeat: '<path d="m17 2 4 4-4 4"/><path d="M3 11v-1a4 4 0 0 1 4-4h14M7 22l-4-4 4-4"/><path d="M21 13v1a4 4 0 0 1-4 4H3"/>',
    receipt: '<path d="M4 2v20l3-2 3 2 3-2 3 2 3-2 1 .7V2l-1 .7L16 1l-3 2-3-2-3 2-3-2z"/><path d="M8 8h8M8 12h8"/>',
    activity: '<path d="M22 12h-4l-3 9L9 3l-3 9H2"/>', plus: '<path d="M12 5v14M5 12h14"/>',
    wallet: '<path d="M20 7V5a2 2 0 0 0-2-2H5a2 2 0 0 0 0 4h15a1 1 0 0 1 1 1v4h-4a2 2 0 0 0 0 4h4v3a1 1 0 0 1-1 1H5a2 2 0 0 1-2-2V5"/>',
    edit: '<path d="M12 20h9"/><path d="M16.5 3.5a2.1 2.1 0 1 1 3 3L7 19l-4 1 1-4z"/>', send: '<path d="m22 2-7 20-4-9-9-4z"/><path d="M22 2 11 13"/>',
    link: '<path d="M10 13a5 5 0 0 0 7.5.5l3-3a5 5 0 0 0-7-7l-1.7 1.7"/><path d="M14 11a5 5 0 0 0-7.5-.5l-3 3a5 5 0 0 0 7 7l1.7-1.7"/>',
    history: '<path d="M3 3v5h5"/><path d="M3.05 13A9 9 0 1 0 6 5.3L3 8"/><path d="M12 7v5l4 2"/>', x: '<path d="M18 6 6 18M6 6l12 12"/>',
    chev: '<path d="m6 9 6 6 6-6"/>'
  };
  var ic = function (n) { return '<svg class="i" viewBox="0 0 24 24" aria-hidden="true">' + P[n] + '</svg>'; };
  var pill = function (l, t) { return '<span class="pill t-' + t + '">' + esc(l) + '</span>'; };

  var S = M.leadStatuses;
  var NAV = [
    { id: 'leads', label: 'شماره‌های من', icon: 'users', badge: function () { return M.leads.filter(function (l) { return l.status === 'none'; }).length; } },
    { id: 'conversions', label: 'آماده‌های تبدیل', icon: 'swap' },
    { id: 'repeat', label: 'نیاز به اقدام مجدد', icon: 'repeat' },
    { id: 'invoices', label: 'فاکتورها', icon: 'receipt' },
    { id: 'behavior', label: 'رفتار مشتریان', icon: 'activity' },
    { id: 'request', label: 'درخواست اضافه کردن شماره', icon: 'plus' },
    { id: 'wallet', label: 'کیف پول من', icon: 'wallet' }
  ];
  var qs = new URLSearchParams(location.search);
  var st = { view: (location.hash || '#leads').slice(1) || 'leads', filter: qs.get('filter') || 'none', open: null, flow: {}, invTab: 'all', caseOpen: null, caseStatus: '', walletTab: 'all' };

  function nav() {
    $('#side-nav').innerHTML = NAV.map(function (n) {
      var b = n.badge ? n.badge() : 0;
      return '<button type="button" class="nav-item" data-view="' + n.id + '"' + (st.view === n.id ? ' aria-current="page"' : '') + ' title="' + esc(n.label) + '"><span class="ico">' + ic(n.icon) + '</span><span class="t">' + esc(n.label) + '</span>' + (b ? '<span class="nav-badge">' + fa(b) + '</span>' : '') + '</button>';
    }).join('');
    // Demo switch keeps the current page when jumping to B.
    $('#sw-b').href = 'index.html#' + st.view;
  }

  /* ---------- شماره‌های من ---------- */
  function leads() {
    var list = M.leads.filter(function (l) {
      if (st.filter === 'all') return true;
      if (st.filter === 'requested') return l.source === 'درخواستی';
      return l.status === st.filter;
    });
    var done = M.leads.filter(function (l) { return l.status !== 'none'; }).length;
    var chips = [['none', 'بدون وضعیت', 'slate'], ['no_answer', 'جواب نداده', 'amber'], ['callback', 'تماس مجدد', 'blue'], ['duplicate', 'تکراری', 'slate'], ['not_purchased', 'عدم خرید', 'red'], ['invoiced', 'پیش‌فاکتور', 'violet'], ['requested', 'شماره‌های درخواستی', ''], ['all', 'همه', '']].map(function (c) {
      var n = c[0] === 'all' ? M.leads.length : c[0] === 'requested' ? M.leads.filter(function (l) { return l.source === 'درخواستی'; }).length : M.leads.filter(function (l) { return l.status === c[0]; }).length;
      var dot = c[2] ? '<span class="d" style="background:var(--' + c[2] + ')"></span>' : '';
      return '<button type="button" class="chip" data-filter="' + c[0] + '" aria-pressed="' + (st.filter === c[0]) + '">' + dot + c[1] + ' <span class="n">' + fa(n) + '</span></button>';
    }).join('');
    var rows = list.map(function (l) {
      var s = S[l.status], open = st.open === l.id;
      var info = ['<b>' + esc(l.name) + '</b>', esc(l.province) + ' — ' + esc(l.city)];
      if (l.prob) info.push('احتمال: ' + esc(l.prob));
      if (l.next) info.push('<span style="color:' + (l.next.warn ? 'var(--amber)' : 'inherit') + ';font-weight:700">' + esc(l.next.text) + '</span>');
      if (l.note) info.push('<span class="muted">📝 ' + esc(l.note) + '</span>');
      return '<tr class="row' + (open ? ' open' : '') + '" id="row-' + l.id + '">' +
        '<td><div class="phone"><span class="mono">' + fa(l.phone) + '</span><button type="button" class="copy" data-copy="' + l.phone + '" aria-label="کپی شماره">' + ic('copy') + '</button></div><div class="badges">' + pill(s.label, s.tone) + '<span class="src">' + (l.source === 'MIS' ? 'داده MIS تحویل‌شده' : esc(l.source)) + '</span></div></td>' +
        '<td class="date muted"><span class="num">' + esc(l.assigned) + '</span></td>' +
        '<td class="info"><div class="info-line">' + info.join('<span class="sep">|</span>') + '</div></td>' +
        '<td class="cell-actions"><button type="button" class="btn btn-sm ' + (open ? 'btn-primary' : 'btn-soft') + '" data-toggle="' + l.id + '" aria-expanded="' + open + '" aria-controls="ed-' + l.id + '">' + (open ? 'بستن' : 'مشاهده/ویرایش') + '</button></td></tr>' +
        (open ? '<tr class="editor-row" id="ed-' + l.id + '"><td colspan="4">' + editor(l) + '</td></tr>' : '');
    }).join('');
    var rev = 104250000;
    return '<div class="stack">' +
      '<details class="helper"><summary>ℹ️ راهنمای مرحله اول پیگیری شماره</summary><p>برای هر شماره وضعیت تماس را ثبت کنید و پیش‌فاکتور را از داخل همان شماره صادر کنید. «عدم خرید» دلیل اجباری دارد؛ در «جواب نداده» هر تماس را تا سقف سه بار جدا ثبت کنید.</p></details>' +
      '<div class="kpis">' +
      kpi('📞', '#fdf2f8', 'کل شماره‌ها', fa(M.leads.length), 'شماره‌های تخصیص‌یافته') +
      kpi('✅', '#ecfdf5', 'پیگیری‌شده', fa(done), fa(Math.round(done / M.leads.length * 100)) + '٪ از کل شماره‌ها') +
      kpi('🧾', '#eef2ff', 'پیش‌فاکتور', fa(5), 'تبدیل: ۸۳٪') +
      '<div class="kpi money"><span class="ico" style="background:#f0fdf4">💰</span><div class="body"><small>فروش تاییدشده</small><b><span class="num">' + num(rev) + '</span><span class="unit">تومان</span></b><em>۳۵ پرداخت موفق</em></div></div>' +
      '</div>' +
      '<section class="card"><div class="filters" role="group" aria-label="فیلتر وضعیت شماره‌ها">' + chips + '</div>' +
      '<div class="toolbar"><input class="input search" type="search" placeholder="جستجوی شماره، نام، شهر یا کد پرونده"><select class="input" aria-label="تعداد در صفحه"><option>۵۰ تایی</option><option>۲۰ تایی</option><option>۱۰۰ تایی</option></select></div></section>' +
      '<section class="card tbl-card"><div class="tbl-wrap"><table class="tbl lead-table"><thead><tr><th>شماره</th><th>تاریخ تخصیص</th><th>اطلاعات مشتری</th><th class="cell-actions"><span class="sr">عملیات</span></th></tr></thead><tbody>' +
      (rows || '<tr><td colspan="4" class="empty">شماره‌ای با این وضعیت یافت نشد.</td></tr>') + '</tbody></table></div>' +
      '<div class="pager"><span>نمایش <span class="num">' + fa(list.length) + '</span> از <span class="num">' + fa(list.length) + '</span></span><span class="grow"></span><button type="button">›</button><button type="button" aria-current="page">۱</button><button type="button">‹</button></div></section></div>';
  }
  function kpi(icon, bg, label, value, hint) {
    return '<div class="kpi"><span class="ico" style="background:' + bg + '">' + icon + '</span><div class="body"><small>' + label + '</small><b>' + value + '</b><em>' + hint + '</em></div></div>';
  }
  function editor(l) {
    var flow = st.flow[l.id] != null ? st.flow[l.id] : (l.status === 'none' ? '' : l.status);
    var opt = function (v, t) { return '<option value="' + v + '"' + (flow === v ? ' selected' : '') + '>' + t + '</option>'; };
    var third = l.attempts >= 2;
    return '<div class="editor">' +
      '<div class="editor-grid">' +
      '<label class="fld status">وضعیت تماس<select class="input" data-flow="' + l.id + '">' + opt('', '— تعیین وضعیت —') + opt('no_answer', 'جواب نداده') + opt('callback', 'تماس مجدد') + opt('not_purchased', 'عدم خرید') + opt('duplicate', 'تکراری') + opt('invoiced', 'پیش‌فاکتور') + '</select></label>' +
      '<label class="fld">احتمال فروش<select class="input"><option>' + (l.prob || 'انتخاب کنید') + '</option><option>ضعیف</option><option>متوسط</option><option>بالا</option><option>۱۰۰٪</option></select></label>' +
      '<label class="fld">نام مشتری<input class="input" value="' + esc(l.name) + '"></label>' +
      '<div class="fld wide flow' + (flow === 'no_answer' && third ? ' warn' : '') + '"' + (flow === 'no_answer' ? '' : ' hidden') + '><b>تماس بی‌پاسخ: ' + fa(l.attempts) + ' از ۳</b> ' + (l.attempts < 3 ? '<button type="button" class="btn btn-sm" data-demo="نمایشی: تلاش بعدی ثبت نشد" style="margin-right:8px">ثبت تماس بی‌پاسخ بعدی</button>' : '') + '<div style="font-weight:500;margin-top:6px">' + (third ? 'با تلاش سوم، اگر تا سه روز فعالیتی ثبت نشود شماره به بایگانی مدیر فروش منتقل می‌شود.' : 'هر تماس واقعیِ بی‌پاسخ را جدا ثبت کن؛ سقف این مرحله سه تلاش است.') + '</div></div>' +
      '<div class="fld wide flow"' + (flow === 'callback' ? '' : ' hidden') + '><div class="grid2"><label class="fld">تاریخ شمسی تماس مجدد <span style="color:var(--red)">*</span><input class="input" value="۱۴۰۵/۰۷/۱۰"></label><label class="fld">ساعت تهران <span style="color:var(--red)">*</span><span class="time"><select class="input" aria-label="ساعت"><option>10</option></select> : <select class="input" aria-label="دقیقه"><option>30</option></select></span></label></div></div>' +
      '<label class="fld wide flow"' + (flow === 'not_purchased' ? '' : ' hidden') + '>دلیل عدم خرید <span style="color:var(--red)">*</span><textarea class="input" placeholder="دلیل اعلام‌شده یا نتیجه نهایی مکالمه را بنویسید">' + esc(l.status === 'not_purchased' ? l.note : '') + '</textarea></label>' +
      '<label class="fld">استان<select class="input"><option>' + esc(l.province) + '</option></select></label>' +
      '<label class="fld">شهر<select class="input"><option>' + esc(l.city) + '</option></select></label>' +
      '<label class="fld wide">یادداشت<textarea class="input" placeholder="خلاصه مکالمه، نیاز مشتری، زمان پیگیری...">' + esc(l.note) + '</textarea></label>' +
      '</div>' +
      '<div class="editor-actions"><span class="saved">✓ تغییرات فیلدها خودکار ذخیره می‌شود</span><span class="grow"></span><button type="button" class="btn" data-demo="نمایشی: چیزی ذخیره نشد">ذخیره تغییرات</button><button type="button" class="btn btn-primary" data-demo="نمایشی: فرم صدور پیش‌فاکتور باز می‌شود">📄 صدور پیش‌فاکتور</button></div></div>';
  }

  /* ---------- فاکتورها ---------- */
  function invoices() {
    var tabs = [['all', 'همه'], ['pre_invoice', 'پیش‌فاکتور صادر شده'], ['approved', 'پرداخت شده'], ['staged', 'مرحله‌ای تکمیل‌نشده'], ['finance_review', 'در بررسی مالی'], ['rejected', 'رد شده'], ['cancelled', 'لغو شده'], ['recontact', 'ارتباط مجدد با کارشناس']];
    var list = M.invoices.filter(function (i) { return st.invTab === 'all' || i.status === st.invTab; });
    var rows = list.map(function (i) {
      var rem = i.total - i.paid;
      var main = i.next === 'register_payment' || i.next === 'send_link' ? '<button type="button" class="btn btn-sm btn-soft" data-pay="' + i.code + '">ثبت واریز و فیش</button>'
        : i.next === 'resend_finance' ? '<button type="button" class="btn btn-sm btn-soft" data-demo="نمایشی: فرم ارسال مجدد به مالی">ارسال مجدد برای مالی</button>' : '';
      var editable = i.paid === 0 && i.status === 'pre_invoice';
      return '<tr class="row"><td><span class="mono" style="font-weight:800">' + fa(i.code) + '</span></td><td><b>' + esc(i.customer) + '</b></td><td><span class="mono muted">' + fa(i.phone) + '</span></td><td>' + esc(i.product) + '</td>' +
        '<td><div class="money-lines"><span>کل: ' + amt(i.total) + '</span><span>پرداخت‌شده: ' + amt(i.paid) + '</span><strong>مانده: ' + (rem > 0 ? amt(rem) : '✓ تسویه') + '</strong><small>' + (i.stages > 1 ? 'مرحله ' + fa(i.stage) + ' از ' + fa(i.stages) : 'پرداخت یک‌جا') + '</small></div></td>' +
        '<td>' + pill(i.statusLabel, i.tone) + (i.rejectReason ? '<div class="reject">دلیل رد: ' + esc(i.rejectReason) + '</div>' : '') + '</td>' +
        '<td>' + (i.conversion ? '<div class="conv"><b>' + esc(i.conversion) + '</b><span class="muted">اشتراک VIP</span></div>' : '<span class="muted">—</span>') + '</td>' +
        '<td class="sticky-ops"><div class="ops">' + main +
        (editable ? '<button type="button" class="icon-btn sm" title="ویرایش پیش از پرداخت" aria-label="ویرایش پیش از پرداخت" data-demo="نمایشی: ویرایش پیش از پرداخت">' + ic('edit') + '</button>' : '') +
        (['approved', 'cancelled'].indexOf(i.status) === -1 ? '<button type="button" class="icon-btn sm" title="ارسال مجدد لینک پرداخت" aria-label="ارسال مجدد لینک پرداخت" data-demo="نمایشی: پیامکی ارسال نشد">' + ic('send') + '</button><button type="button" class="icon-btn sm" title="کپی لینک پرداخت" aria-label="کپی لینک پرداخت" data-demo="نمایشی: لینک کپی نشد">' + ic('link') + '</button>' : '') +
        '<button type="button" class="icon-btn sm" title="وضعیت و تاریخچه مالی" aria-label="وضعیت و تاریخچه مالی" data-history="' + i.code + '">' + ic('history') + '</button></div></td></tr>';
    }).join('');
    return '<div class="stack"><div class="subtabs" role="group" aria-label="وضعیت فاکتور">' + tabs.map(function (t) { return '<button type="button" class="subtab" data-invtab="' + t[0] + '" aria-pressed="' + (st.invTab === t[0]) + '">' + t[1] + '</button>'; }).join('') + '</div>' +
      '<section class="card"><label class="fld">جستجوی فاکتور<input class="input" type="search" placeholder="کد فاکتور، نام یا موبایل مشتری" style="max-width:520px"></label><div class="muted" style="font-size:12px;margin-top:6px">جستجو فقط در فاکتورهای داخلی خودتان انجام می‌شود.</div></section>' +
      '<section class="card tbl-card"><div class="tbl-wrap"><table class="tbl inv-table"><thead><tr><th>کد</th><th>مشتری</th><th>موبایل</th><th>محصول‌ها</th><th>مبلغ کل / پرداخت‌شده / مانده</th><th>وضعیت</th><th>نتیجه تبدیل اعتبارسنجی</th><th class="sticky-ops" style="text-align:left">عملیات</th></tr></thead><tbody>' + rows + '</tbody></table></div>' +
      '<div class="pager"><span>نمایش <span class="num">' + fa(list.length) + '</span> از <span class="num">' + fa(list.length) + '</span></span><span class="grow"></span></div></section></div>';
  }

  /* ---------- آماده‌های تبدیل ---------- */
  function conversions() {
    var cases = M.conversions.map(function (c) {
      var open = st.caseOpen === c.id, s = st.caseStatus;
      return '<div class="case"><div class="case-row"><div><b class="mono">#' + fa(c.id) + '</b><div class="muted" style="font-size:11.5px">فاکتور ' + fa(c.invoice) + '</div></div>' +
        '<div><b>' + esc(c.customer) + '</b><div class="muted mono" style="font-size:12px">' + fa(c.phone) + '</div></div><div class="opt"><span class="muted" style="font-size:11px;display:block">گزینه</span>' + esc(c.option) + '</div>' +
        '<div class="opt"><span class="muted" style="font-size:11px;display:block">مانده</span>' + (c.remaining ? amt(c.remaining) : '—') + '</div><div>' + pill(c.result, c.tone) + '</div>' +
        '<button type="button" class="btn btn-sm ' + (open ? 'btn-primary' : 'btn-soft') + '" data-case="' + c.id + '" aria-expanded="' + open + '">' + (open ? 'بستن پرونده' : 'باز کردن پرونده') + '</button></div>' +
        (open ? '<div class="case-body"><div class="facts"><div class="fact"><span>مبلغ گزینه</span><b>' + amt(c.remaining) + '</b></div><div class="fact"><span>پرداخت‌شده</span><b>' + amt(0) + '</b></div><div class="fact"><span>مانده</span><b>' + amt(c.remaining) + '</b></div><div class="fact"><span>آخرین فعالیت</span><b class="num">۱۴۰۵/۰۶/۱۷ — ۱۰:۴۰</b></div></div>' +
          '<div class="card" style="padding:14px"><div class="sec-title"><h2>۱. نتیجه تماس با مشتری</h2></div><div class="grid2">' +
          '<label class="fld">نتیجه تماس *<select class="input" data-case-status><option value="">انتخاب نتیجه</option><option value="no_answer"' + (s === 'no_answer' ? ' selected' : '') + '>جواب نداده (۰ بار ثبت‌شده)</option><option value="follow_up"' + (s === 'follow_up' ? ' selected' : '') + '>تماس مجدد</option><option value="customer_declined"' + (s === 'customer_declined' ? ' selected' : '') + '>انصراف</option><option value="payment_link"' + (s === 'payment_link' ? ' selected' : '') + '>ارسال لینک پرداخت</option></select></label>' +
          '<label class="fld">محصول نهایی<select class="input"><option>' + esc(c.option) + '</option></select></label>' +
          '<label class="fld wide"' + (s === 'customer_declined' ? '' : ' hidden') + ' style="grid-column:1/-1">علت انصراف <span style="color:var(--red)">*</span><textarea class="input" placeholder="علت دقیق انصراف مشتری را ثبت کنید"></textarea></label>' +
          '<label class="fld" style="grid-column:1/-1">یادداشت<textarea class="input" placeholder="مثلاً: مشتری فردا تماس می‌گیرد"></textarea></label></div>' +
          '<div class="editor-actions" style="margin-top:12px"><span class="grow"></span><button type="button" class="btn btn-primary" data-demo="نمایشی: نتیجه تماس ذخیره نشد">' + (s === 'payment_link' ? 'ثبت محصول و ادامه به ارسال لینک' : 'ذخیره نتیجه تماس') + '</button></div></div></div>' : '') + '</div>';
    }).join('');
    return '<div class="stack"><section class="card"><div class="sec-title"><h2>آماده‌های تبدیل من</h2></div><div class="muted" style="font-size:12.5px">فقط پرونده‌هایی که هنگام ثبت فاکتور «تبدیل توسط خودم» را انتخاب کرده‌اید نمایش داده می‌شوند.</div></section>' +
      '<div class="chips-kpi"><span><b>۳</b> کل پرونده</span><span><b>۱</b> نیازمند پیگیری</span><span><b>۰</b> تماس سررسیده</span><span><b>۲</b> دارای مانده</span><span><b>۰</b> تکمیل‌شده</span></div>' +
      '<section class="card"><div class="grid2" style="grid-template-columns:2fr 1fr 1fr"><label class="fld">جستجو<input class="input" placeholder="نام، موبایل، فاکتور یا شماره پرونده"></label><label class="fld">مرحله پرونده<select class="input"><option>همه مرحله‌ها</option></select></label><label class="fld">نتیجه تماس<select class="input"><option>همه نتایج</option></select></label></div></section>' +
      cases + '</div>';
  }

  function repeat() {
    return '<div class="stack"><section class="card"><div class="sec-title"><h2>نیاز به اقدام مجدد</h2></div><div class="helper" style="display:block">در پرداخت مرحله‌ای، مالک فروش و پورسانت پایه برای فروشنده صادرکننده اولیه حفظ می‌شود. فردی که مرحله بعد را انجام می‌دهد به‌عنوان مسئول اقدام در تاریخچه ثبت می‌شود.</div></section>' +
      '<section class="card tbl-card"><table class="tbl"><thead><tr><th>فاکتور مرجع</th><th>مشتری</th><th>فروشنده اصلی</th><th>وضعیت مالی</th><th>اقدام</th></tr></thead><tbody><tr><td colspan="5"><div class="empty">✓ فعلاً موردی برای اقدام مجدد وجود ندارد.</div></td></tr></tbody></table></section></div>';
  }

  function behavior() {
    var rows = M.customerActions.map(function (a) {
      return '<tr class="row"><td><span class="mono">' + fa(a.phone) + '</span></td><td><b>' + esc(a.name) + '</b></td><td><span class="mono">' + fa(a.invoice) + '</span></td><td>' + esc(a.last) + ' <span class="muted">· ' + esc(a.when) + '</span></td><td class="cell-actions"><button type="button" class="btn btn-sm btn-soft" data-behavior="' + a.invoice + '">مشاهده کامل</button></td></tr>';
    }).join('');
    return '<div class="stack"><section class="card"><div class="sec-title"><h2>رفتار مشتریان در صفحه فاکتور</h2></div><div class="muted" style="font-size:12.5px;margin-bottom:10px">باز شدن لینک، مشاهده اطلاعات محصول، کلیک روی گردونه، کد تخفیف، پرداخت و سایر اکشن‌های مشتری.</div>' +
      '<div class="toolbar" style="margin-top:0"><input class="input search" placeholder="جستجوی شماره مشتری یا کد فاکتور"><button type="button" class="btn">جستجو</button><button type="button" class="btn btn-ghost">پاک کردن</button></div></section>' +
      '<section class="card tbl-card"><div class="tbl-wrap"><table class="tbl"><thead><tr><th>شماره مشتری</th><th>اسم</th><th>شماره فاکتور</th><th>آخرین فعالیت</th><th class="cell-actions"><span class="sr">مشاهده</span></th></tr></thead><tbody>' + rows + '</tbody></table></div><div class="pager"><span>نمایش ۵ از ۲۰</span><span class="grow"></span><button type="button">›</button><button type="button" aria-current="page">۱</button><button type="button">۲</button><button type="button">‹</button></div></section></div>';
  }

  function request() {
    return '<section class="card" style="max-width:720px"><div class="sec-title"><h2>درخواست اضافه کردن شماره</h2></div><p class="muted" style="margin:0 0 14px;font-size:12.5px">اگر شماره‌ای خارج از لیست تخصیص‌یافته داری و می‌خواهی برای آن پیش‌فاکتور صادر کنی، اینجا ثبت کن. پس از تایید مدیر فروش، شماره با برچسب «درخواست شماره تایید شده» وارد لیست شماره‌های تو می‌شود.</p>' +
      '<div class="grid2"><label class="fld">شماره موبایل مشتری <span style="color:var(--red)">*</span><input class="input" dir="ltr" placeholder="09xxxxxxxxx" style="text-align:right"></label><label class="fld">نام مشتری<input class="input" placeholder="اختیاری"></label><label class="fld" style="grid-column:1/-1">یادداشت<textarea class="input" placeholder="علت درخواست یا توضیح کوتاه، اختیاری"></textarea></label></div>' +
      '<div class="editor-actions" style="margin-top:14px"><span class="grow"></span><button type="button" class="btn btn-primary" data-demo="نمایشی: درخواست ثبت نشد">ثبت درخواست</button></div></section>';
  }

  function wallet() {
    var W = M.wallet;
    var tx = W.tx.filter(function (t) { return st.walletTab === 'all' || (st.walletTab === 'online' ? t.channel === 'آنلاین' : t.channel !== 'آنلاین'); });
    return '<div class="stack"><div class="subtabs" style="width:max-content"><button type="button" class="subtab" aria-pressed="true">فروش و پورسانت</button><button type="button" class="subtab" data-demo="نمایشی: کیف پول پروژه‌ها">پروژه‌ها</button></div>' +
      '<div class="subtabs">' + [['all', 'همه'], ['online', 'کیف پول پرداختی‌های آنلاین'], ['card', 'کیف پول پرداخت‌های کارت‌به‌کارتی']].map(function (t) { return '<button type="button" class="subtab" data-wtab="' + t[0] + '" aria-pressed="' + (st.walletTab === t[0]) + '">' + t[1] + '</button>'; }).join('') + '</div>' +
      '<div class="w-stats"><div class="w-stat main"><span>موجودی قابل تسویه</span><b>' + amt(W.balance) + '</b></div><div class="w-stat"><span>کل بستانکاری</span><b>' + amt(W.credit) + '</b></div><div class="w-stat"><span>کل برداشت/تسویه</span><b>' + amt(W.debit) + '</b></div></div>' +
      '<section class="card tbl-card"><div class="sec-title" style="padding:14px 14px 0"><h2>آخرین تراکنش‌ها</h2></div><div class="tbl-wrap"><table class="tbl"><thead><tr><th>تاریخ</th><th>نوع</th><th>مبلغ</th><th>شرح</th></tr></thead><tbody>' +
      tx.map(function (t) { return '<tr class="row"><td class="muted"><span class="num">' + esc(t.date) + '</span></td><td>' + pill('بستانکار', 'green') + '<div class="muted" style="font-size:11px;margin-top:3px">' + esc(t.channel) + '</div></td><td><span class="plus">+' + amt(t.amount) + '</span></td><td>' + esc(t.desc) + '<div class="muted" style="font-size:11px">' + esc(t.rule) + '</div></td></tr>'; }).join('') +
      '</tbody></table></div><div class="pager"><span>نمایش ' + fa(tx.length) + ' از ۴۰</span><span class="grow"></span><button type="button">›</button><button type="button" aria-current="page">۱</button><button type="button">۲</button><button type="button">‹</button></div></section></div>';
  }

  /* ---------- Modals (focus stays inside; Esc closes; focus returns) ---------- */
  var lastOpener = null;
  function openModal(html, opener) {
    lastOpener = opener || document.activeElement;
    var box = $('#modal-box'); box.innerHTML = html; $('#modal').hidden = false;
    setTimeout(function () { box.focus(); }, 10);
  }
  function closeModal() { $('#modal').hidden = true; if (lastOpener && lastOpener.focus) lastOpener.focus(); }
  function historyModal(code) {
    var i = M.invoices.filter(function (x) { return x.code === code; })[0];
    return '<div class="mh"><h3 id="modal-title">وضعیت و سوابق مالی</h3><button type="button" class="icon-btn sm" data-close-modal aria-label="بستن">' + ic('x') + '</button></div>' +
      '<div class="mb"><div class="ctx"><span>فاکتور <b class="mono">' + fa(i.code) + '</b></span><span>مشتری <b>' + esc(i.customer) + '</b></span><span>وضعیت ' + pill(i.statusLabel, i.tone) + '</span></div>' +
      '<div class="grid2" style="grid-template-columns:repeat(3,1fr)"><div class="fact"><span>مبلغ کل</span><b>' + amt(i.total) + '</b></div><div class="fact"><span>واریزشده</span><b>' + amt(i.paid) + '</b></div><div class="fact"><span>مانده</span><b>' + amt(i.total - i.paid) + '</b></div></div>' +
      (i.rejectReason ? '<div class="flow warn" style="background:var(--red-bg);border-color:#fecaca;color:var(--red)"><b>دلیل رد فعلی:</b> ' + esc(i.rejectReason) + '</div>' : '') +
      '<h4>تاریخچه بررسی</h4><ul class="timeline">' + (i.status === 'rejected' ? '<li><b>رد توسط مالی</b><small>کارشناس مالی · دیروز ۱۴:۲۰</small></li>' : '') + (i.status === 'approved' ? '<li><b>تایید مالی</b><small>کارشناس مالی · ۱۴۰۵/۰۶/۱۸</small></li>' : '') + '<li><b>صدور پیش‌فاکتور</b><small>مهدی زاده · ۱۴۰۵/۰۶/۱۵</small></li></ul></div>' +
      '<div class="mf"><button type="button" class="btn" data-close-modal>بستن</button></div>';
  }
  function payModal(code) {
    var i = M.invoices.filter(function (x) { return x.code === code; })[0];
    return '<div class="mh"><h3 id="modal-title">ثبت اطلاعات واریز</h3><button type="button" class="icon-btn sm" data-close-modal aria-label="بستن">' + ic('x') + '</button></div>' +
      '<div class="mb"><div class="ctx"><span>مشتری <b>' + esc(i.customer) + '</b></span><span>کد فاکتور <b class="mono">' + fa(i.code) + '</b></span><span>وضعیت <b>' + esc(i.statusLabel) + '</b></span><span>مانده <b>' + amt(i.total - i.paid) + '</b></span></div>' +
      '<div class="grid2"><label class="fld">مبلغ واریز این مرحله (تومان)<input class="input" readonly value="' + num(Math.round(i.total / i.stages)) + '"></label><label class="fld">۴ رقم آخر کارت مبدا<input class="input" inputmode="numeric" maxlength="4" placeholder="مثلاً ۱۲۳۴"></label>' +
      '<label class="fld">۴ رقم آخر کارت مقصد (اختیاری)<input class="input" inputmode="numeric" maxlength="4"></label><label class="fld">تاریخ شمسی واریز<input class="input" placeholder="۱۴۰۵/۰۷/۰۱"></label>' +
      '<label class="fld">ساعت تهران (۲۴ ساعته)<span class="time"><select class="input" aria-label="ساعت"><option>14</option></select> : <select class="input" aria-label="دقیقه"><option>05</option></select></span></label><label class="fld">تصویر فیش (اختیاری)<input class="input" type="file" style="padding-top:6px"></label></div></div>' +
      '<div class="mf"><button type="button" class="btn btn-primary" data-demo="نمایشی: واریز ثبت نشد">ثبت اطلاعات واریز</button><button type="button" class="btn" data-close-modal>انصراف</button></div>';
  }
  function behaviorModal(code) {
    var a = M.customerActions.filter(function (x) { return x.invoice === code; })[0];
    return '<div class="mh"><h3 id="modal-title">جزئیات رفتار مشتری</h3><button type="button" class="icon-btn sm" data-close-modal aria-label="بستن">' + ic('x') + '</button></div><div class="mb"><div class="ctx"><span>مشتری <b>' + esc(a.name) + '</b></span><span>فاکتور <b class="mono">' + fa(a.invoice) + '</b></span></div><ul class="timeline"><li><b>' + esc(a.last) + '</b><small>' + esc(a.when) + '</small></li><li><b>مشاهده اطلاعات محصول</b><small>۲ ساعت پیش</small></li><li><b>باز کردن لینک فاکتور</b><small>دیروز</small></li></ul></div><div class="mf"><button type="button" class="btn" data-close-modal>بستن</button></div>';
  }

  /* ---------- Render / events ---------- */
  var V = { leads: leads, conversions: conversions, repeat: repeat, invoices: invoices, behavior: behavior, request: request, wallet: wallet };
  function render() { nav(); $('#content').innerHTML = (V[st.view] || leads)(); }
  function go(v) { st.view = v; st.open = null; history.replaceState(null, '', location.pathname + location.search + '#' + v); document.body.classList.remove('menu-open'); render(); window.scrollTo(0, 0); }
  var tt;
  function toast(m) { var t = $('#toast'); t.textContent = m; t.classList.add('on'); clearTimeout(tt); tt = setTimeout(function () { t.classList.remove('on'); }, 2200); }

  document.addEventListener('click', function (e) {
    var t = e.target.closest('button, a, [data-close-modal]'); if (!t) return;
    if (t.matches('[data-view]')) return go(t.getAttribute('data-view'));
    if (t.matches('[data-filter]')) { st.filter = t.getAttribute('data-filter'); st.open = null; render(); var b = $('[data-filter="' + st.filter + '"]'); if (b) b.focus(); return; }
    if (t.matches('[data-toggle]')) { var id = Number(t.getAttribute('data-toggle')); st.open = st.open === id ? null : id; render(); var tb = $('[data-toggle="' + id + '"]'); if (tb) tb.focus(); return; }
    if (t.matches('[data-invtab]')) { st.invTab = t.getAttribute('data-invtab'); render(); return; }
    if (t.matches('[data-wtab]')) { st.walletTab = t.getAttribute('data-wtab'); render(); return; }
    if (t.matches('[data-case]')) { var c = Number(t.getAttribute('data-case')); st.caseOpen = st.caseOpen === c ? null : c; st.caseStatus = ''; render(); return; }
    if (t.matches('[data-history]')) return openModal(historyModal(t.getAttribute('data-history')), t);
    if (t.matches('[data-pay]')) return openModal(payModal(t.getAttribute('data-pay')), t);
    if (t.matches('[data-behavior]')) return openModal(behaviorModal(t.getAttribute('data-behavior')), t);
    if (t.matches('[data-close-modal]')) return closeModal();
    if (t.matches('[data-copy]')) return toast('نمایشی: شماره ' + fa(t.getAttribute('data-copy')) + ' کپی شد');
    if (t.matches('[data-demo]')) return toast(t.getAttribute('data-demo'));
    if (t.id === 'side-toggle') { if (matchMedia('(max-width:760px)').matches) document.body.classList.remove('menu-open'); else $('#p-body').classList.toggle('collapsed'); return; }
    if (t.id === 'hamburger') { var o = document.body.classList.toggle('menu-open'); t.setAttribute('aria-expanded', String(o)); return; }
  });
  $('#scrim').addEventListener('click', function () { document.body.classList.remove('menu-open'); });
  document.addEventListener('change', function (e) {
    if (e.target.matches('[data-flow]')) { st.flow[e.target.getAttribute('data-flow')] = e.target.value; render(); var s = $('[data-flow="' + e.target.getAttribute('data-flow') + '"]'); if (s) s.focus(); }
    if (e.target.matches('[data-case-status]')) { st.caseStatus = e.target.value; render(); var cs = $('[data-case-status]'); if (cs) cs.focus(); }
  });
  document.addEventListener('keydown', function (e) {
    var modalOpen = !$('#modal').hidden;
    if (e.key === 'Escape') { if (modalOpen) closeModal(); else document.body.classList.remove('menu-open'); }
    if (e.key === 'Tab' && modalOpen) {
      var f = $$('#modal-box button, #modal-box input, #modal-box select, #modal-box textarea').filter(function (x) { return !x.disabled && x.offsetParent; });
      if (!f.length) return;
      var first = f[0], last = f[f.length - 1], a = document.activeElement;
      if (e.shiftKey && (a === first || a === $('#modal-box'))) { e.preventDefault(); last.focus(); }
      else if (!e.shiftKey && a === last) { e.preventDefault(); first.focus(); }
      else if (!$('#modal-box').contains(a)) { e.preventDefault(); first.focus(); }
    }
  });
  window.addEventListener('hashchange', function () { var v = location.hash.slice(1); if (V[v] && v !== st.view) go(v); });

  render();
  if (qs.get('open')) { st.open = Number(qs.get('open')); if (qs.get('flow')) st.flow[st.open] = qs.get('flow'); render(); }
  if (qs.get('history')) openModal(historyModal(qs.get('history')));
  if (qs.get('pay')) openModal(payModal(qs.get('pay')));
})();
