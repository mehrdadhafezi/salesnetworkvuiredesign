/* Version B — Help Center, first-time welcome, guided tours and demo states. Prototype only.
   Tour progress is versioned (e.g. seller_tour_v1): bumping the version re-shows a tour after a UI change. */
(function () {
  'use strict';
  var B = window.SNB;
  if (!B) return;
  var $ = function (s, r) { return (r || document).querySelector(s); };
  var fa = function (v) { return String(v).replace(/\d/g, function (d) { return '۰۱۲۳۴۵۶۷۸۹'[d]; }); };
  var esc = function (v) { return String(v == null ? '' : v).replace(/[&<>"']/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]; }); };
  var reduced = window.matchMedia && matchMedia('(prefers-reduced-motion: reduce)').matches;
  var store = {
    get: function (k) { try { return localStorage.getItem('snproto.' + k); } catch (e) { return null; } },
    set: function (k, v) { try { localStorage.setItem('snproto.' + k, v); } catch (e) {} },
    del: function (k) { try { localStorage.removeItem('snproto.' + k); } catch (e) {} }
  };
  var IC = {
    help: '<circle cx="12" cy="12" r="9"/><path d="M9.6 9.3a2.5 2.5 0 0 1 4.8.9c0 1.7-2.4 2.3-2.4 3.8"/><path d="M12 17h.01"/>',
    x: '<path d="M18 6 6 18M6 6l12 12"/>', check: '<path d="M20 6 9 17l-5-5"/>', arrow: '<path d="m9 18 6-6-6-6"/>',
    compass: '<circle cx="12" cy="12" r="9"/><path d="m15.5 8.5-2 5-5 2 2-5z"/>', users: '<path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/>',
    receipt: '<path d="M4 2v20l3-2 3 2 3-2 3 2 3-2 1 .7V2l-1 .7L16 1l-3 2-3-2-3 2-3-2z"/><path d="M8 8h8M8 12h8"/>', swap: '<path d="m16 3 4 4-4 4M20 7H4M8 21l-4-4 4-4M4 17h16"/>',
    wallet: '<path d="M20 7V5a2 2 0 0 0-2-2H5a2 2 0 0 0 0 4h15a1 1 0 0 1 1 1v4h-4a2 2 0 0 0 0 4h4v3a1 1 0 0 1-1 1H5a2 2 0 0 1-2-2V5"/>',
    key: '<rect x="2" y="6" width="20" height="12" rx="2"/><path d="M6 10h.01M10 10h.01M14 10h.01M18 10h.01M8 14h8"/>', replay: '<path d="M3 3v5h5"/><path d="M3.05 13A9 9 0 1 0 6 5.3L3 8"/>',
    sparkle: '<path d="M12 3v4M12 17v4M3 12h4M17 12h4M6 6l2.5 2.5M15.5 15.5 18 18M6 18l2.5-2.5M15.5 8.5 18 6"/>', flask: '<path d="M9 3h6M10 3v6l-5 9a2 2 0 0 0 1.7 3h10.6a2 2 0 0 0 1.7-3l-5-9V3"/>'
  };
  var ic = function (n) { return '<svg class="i" viewBox="0 0 24 24" aria-hidden="true">' + (IC[n] || '') + '</svg>'; };
  var kbd = function (k) { return '<span class="kbd">' + k + '</span>'; };

  /* ---------- Tour registry (versioned) ---------- */
  var goLeads = function () { if (B.state.view !== 'leads') B.go('leads'); if (B.state.leadQueue !== 'all') B.switchQueue('all'); if (B.state.drawer) B.closeDrawer(); };
  var goView = function (v) { return function () { if (B.state.drawer) B.closeDrawer(); if (B.state.view !== v) B.go(v); }; };
  var noDrawer = function () { if (B.state.drawer) B.closeDrawer(); };
  var TOURS = {
    seller: { key: 'seller_tour_v1', title: 'آموزش سریع پنل فروشنده', desc: 'آشنایی با بخش‌های اصلی پنل در کمتر از ۲ دقیقه', dur: '۲ دقیقه', icon: 'compass', tone: '', steps: [
      { prep: goLeads, sel: ['#nav', '#bottom-nav'], title: 'ناوبری اصلی', body: 'از این بخش بین شماره‌ها، فاکتورها، تبدیل، کیف پول و سایر بخش‌های کاری جابه‌جا شوید.' },
      { prep: noDrawer, sel: ['.kpis'], title: 'شاخص‌های همین صفحه', body: 'این شاخص‌ها خلاصه وضعیت کاری همین صفحه را نشان می‌دهند. روی هر شاخص بزنید تا همان صف باز شود.' },
      { sel: ['.queues'], title: 'صف‌های پیگیری', body: 'با این تب‌ها شماره‌ها را بر اساس وضعیت پیگیری می‌کنید. کلیدهای ۱ تا ۶ هم صف را عوض می‌کنند.' },
      { sel: ['.toolbar'], title: 'جستجو و فیلتر', body: 'برای پیدا کردن مشتری یا محدود کردن فهرست از این بخش استفاده کنید. کلید F مستقیم به جستجو می‌رود.' },
      { sel: [function () { var t = $('.tbl'); if (!t || !visible(t)) return null; return [t.tHead].concat(Array.prototype.slice.call(t.tBodies[0].rows, 0, 4)); }, function () { return Array.prototype.slice.call(document.querySelectorAll('.card-list .lcard'), 0, 2); }], title: 'فهرست مشتریان', body: 'اطلاعات اصلی مشتری، وضعیت و «گام بعدی» در این جدول دیده می‌شود. با J و K بین ردیف‌ها حرکت کنید.' },
      { sel: ['.tbl tbody tr:first-child .row-actions .btn-soft', '.lcard .btn-soft'], title: 'اقدام اصلی', body: 'از این دکمه نتیجه تماس یا اقدام اصلی بعدی را ثبت می‌کنید. هر ردیف فقط یک اقدام اصلی دارد.' },
      { prep: function () { B.openDrawer('lead', B.firstLeadId()); }, wait: 260, sel: ['#drawer'], title: 'پنل جزئیات', body: 'اطلاعات کامل مشتری، یادداشت‌ها، تاریخچه و ثبت نتیجه در این پنل است؛ فهرست پشت آن سر جایش می‌ماند.' },
      { prep: noDrawer, wait: 220, sel: ['#help-btn'], title: 'راهنما و میانبرها', body: 'هر زمان لازم بود، راهنما، آموزش‌ها و میانبرهای کیبورد از این دکمه در دسترس هستند.' }
    ]},
    leads: { key: 'leads_tour_v1', title: 'آموزش شماره‌های من', desc: 'صف‌ها، جستجو، ردیف مشتری و ثبت نتیجه', dur: '۱ دقیقه', icon: 'users', tone: 'teal', steps: [
      { prep: goLeads, sel: ['.queues'], title: 'صف‌ها', body: 'هر صف یک مرحله از پیگیری است؛ عدد کنار آن تعداد کارهای همان صف است.' },
      { sel: ['#queue-filter'], title: 'جستجو در صف', body: 'نام، شماره یا شهر را بنویسید؛ فقط همین صف جستجو می‌شود. برای همه بخش‌ها Ctrl+K را بزنید.' },
      { sel: ['.toolbar .chip-btn'], title: 'فیلترها', body: 'منبع، احتمال فروش و تاریخ تخصیص فهرست را محدود می‌کنند و روی وضعیت مشتری اثری ندارند.' },
      { sel: ['.tbl tbody tr:first-child', '.lcard'], title: 'ردیف مشتری', body: 'وضعیت، نام و شماره، شهر و منبع در یک نگاه. روی ردیف بزنید تا جزئیات باز شود.' },
      { sel: [function () { var n = $('.tbl .next'); return n && n.closest('td'); }, '.lcard .next'], title: 'گام بعدی', body: 'اقدام پیشنهادی بعدی بر اساس وضعیت فعلی؛ رنگ کهربایی یعنی توجه، قرمز یعنی سررسید یا ریسک انتقال.' },
      { prep: function () { B.openDrawer('lead', 9043346); }, wait: 260, sel: ['#drawer'], title: 'جزئیات و ثبت نتیجه', body: 'نتیجه تماس را انتخاب کنید؛ فیلدهای لازم همان‌جا ظاهر می‌شوند و دکمه پایین پنل همیشه در دسترس است.' }
    ]},
    invoices: { key: 'invoices_tour_v1', title: 'آموزش فاکتورها', desc: 'وضعیت مالی، پیشرفت پرداخت و اقدام بعدی', dur: '۱ دقیقه', icon: 'receipt', tone: 'violet', steps: [
      { prep: goView('invoices'), wait: 160, sel: ['.queues'], title: 'وضعیت فاکتورها', body: 'فاکتورها بر اساس وضعیت مالی تفکیک شده‌اند: پیش‌فاکتور، مرحله‌ای، در بررسی مالی، تایید، رد و لغو.' },
      { sel: [function () { var r = $('.tbl tbody tr:first-child'); return r ? [r.children[0], r.children[1]] : null; }, '.lcard'], title: 'مشتری و کد فاکتور', body: 'کد فاکتور و مشتری مرتبط؛ کد برای پیگیری با مالی کافی است.' },
      { sel: ['.tbl tbody tr:first-child .prog .line', '.lcard .meta'], title: 'مبلغ', body: 'مبلغ کل فاکتور و این‌که پرداخت یک‌جا است یا مرحله‌ای.' },
      { sel: ['.tbl tbody tr:nth-child(3) .prog', '.lcard .meta'], title: 'پیشرفت پرداخت', body: 'نوار سبز سهم پرداخت‌شده را نشان می‌دهد؛ مانده در ستون بعدی است.' },
      { sel: ['.tbl tbody tr:first-child .row-actions .btn-soft', '.lcard .btn-soft'], title: 'اقدام بعدی', body: 'هر فاکتور فقط یک اقدام اصلی دارد؛ بقیه کارها در منوی سه‌نقطه هستند.' },
      { prep: function () { B.openDrawer('invoice', '71942055'); }, wait: 260, sel: ['#drawer'], title: 'سوابق مالی', body: 'جمع مبالغ، دلیل رد، مراحل پرداخت و تاریخچه بررسی مالی در این پنل است.' }
    ]},
    conversions: { key: 'conversions_tour_v1', title: 'آموزش آماده‌های تبدیل', desc: 'پرونده‌ها، مانده و ثبت نتیجه تماس', dur: '۱ دقیقه', icon: 'swap', tone: 'green', steps: [
      { prep: goView('conversions'), wait: 160, sel: ['.kpis'], title: 'خلاصه پرونده‌ها', body: 'تعداد پرونده‌ها، موارد نیازمند پیگیری و پرونده‌های دارای مانده.' },
      { sel: ['.toolbar'], title: 'جستجو و فیلتر', body: 'با نام، موبایل، فاکتور یا شماره پرونده پیدا کنید؛ مرحله و نتیجه تماس هم قابل فیلترند.' },
      { sel: ['.tbl tbody tr:first-child td:nth-child(4)'], title: 'مانده', body: 'مبلغی که هنوز از این پرونده باید وصول شود.' },
      { sel: ['.tbl tbody tr:first-child .row-actions .btn-soft'], title: 'باز کردن پرونده', body: 'پرونده در پنل کناری باز می‌شود و فهرست سر جایش می‌ماند.' },
      { prep: function () { B.openDrawer('case', B.firstCase()); }, wait: 260, sel: ['#drawer .outcomes', '#drawer'], title: 'ثبت نتیجه تماس', body: 'نتیجه را انتخاب کنید؛ برای «انصراف» علت الزامی است و برای لینک پرداخت، محصول و مبلغ مرحله.' }
    ]},
    wallet: { key: 'wallet_tour_v1', title: 'آموزش کیف پول', desc: 'موجودی، بستانکاری و تراکنش‌ها', dur: '۴۰ ثانیه', icon: 'wallet', tone: 'amber', steps: [
      { prep: goView('wallet'), wait: 160, sel: ['.wcard.hero'], title: 'موجودی قابل تسویه', body: 'مبلغی که الان قابل تسویه است.' },
      { sel: ['.wallet-sum .wcard:nth-child(2)'], title: 'بستانکاری', body: 'مجموع پورسانت‌هایی که تا امروز برای شما ثبت شده است.' },
      { sel: ['.tbl-wrap'], title: 'تراکنش‌ها', body: 'هر ردیف یک تراکنش با شرح و قانون مالی مرتبط است.' },
      { sel: [function () { return [$('.panel .toolbar'), $('.panel .tfoot')].filter(Boolean); }], title: 'فیلتر و صفحه‌بندی', body: 'نوع پرداخت و بازه تاریخ را انتخاب کنید؛ فهرست صفحه‌بندی شده است.' }
    ]}
  };
  var PAGE_TOUR = { leads: 'leads', invoices: 'invoices', conversions: 'conversions', wallet: 'wallet' };
  var isDone = function (id) { return store.get(TOURS[id].key + '_completed') === '1'; };

  /* ---------- Tour engine ---------- */
  var T = null; // { id, i, opener, els }
  function visible(el) { if (!el || !el.getClientRects().length) return false; var r = el.getBoundingClientRect(); return r.width > 0 && r.height > 0; }
  function resolve(sel) {
    for (var n = 0; n < sel.length; n++) {
      var s = sel[n], got = typeof s === 'function' ? s() : $(s);
      if (!got) continue;
      var list = Array.isArray(got) ? got.filter(visible) : (visible(got) ? [got] : []);
      if (list.length) return list;
    }
    return null;
  }
  function unionRect(els) {
    // The drawer may still be sliding in; use its final geometry (transform does not affect offset sizes).
    if (els.length === 1 && els[0].id === 'drawer') {
      var d = els[0];
      return innerWidth <= 760 ? { top: innerHeight - d.offsetHeight, left: 0, right: innerWidth, bottom: innerHeight } : { top: 0, left: 0, right: d.offsetWidth, bottom: innerHeight };
    }
    var r = { top: Infinity, left: Infinity, right: -Infinity, bottom: -Infinity };
    els.forEach(function (e) { var b = e.getBoundingClientRect(); r.top = Math.min(r.top, b.top); r.left = Math.min(r.left, b.left); r.right = Math.max(r.right, b.right); r.bottom = Math.max(r.bottom, b.bottom); });
    return r;
  }
  function startTour(id, at) {
    closeHelp(); closeDemo(); B.closeMenu();
    if ($('#palette') && !$('#palette').hidden) { $('#palette').hidden = true; }
    endTour(false, true);
    T = { id: id, i: Math.max(0, (at || 1) - 1), opener: document.activeElement };
    var catchEl = document.createElement('div'); catchEl.className = 'tour-catch'; catchEl.setAttribute('aria-hidden', 'true');
    var spot = document.createElement('div'); spot.className = 'tour-spot'; spot.setAttribute('aria-hidden', 'true');
    var coach = document.createElement('div'); coach.className = 'coach'; coach.setAttribute('role', 'dialog'); coach.setAttribute('aria-modal', 'true'); coach.setAttribute('aria-labelledby', 'coach-t'); coach.setAttribute('aria-describedby', 'coach-b'); coach.tabIndex = -1;
    document.body.appendChild(catchEl); document.body.appendChild(spot); document.body.appendChild(coach);
    T.els = { catchEl: catchEl, spot: spot, coach: coach };
    showStep();
  }
  function showStep() {
    if (!T) return;
    var tour = TOURS[T.id], st = tour.steps[T.i], total = tour.steps.length;
    if (st.prep) st.prep();
    var coach = T.els.coach, last = T.i === total - 1;
    coach.innerHTML = '<div class="c-top"><span class="c-step">' + fa(T.i + 1) + ' از ' + fa(total) + '</span><span class="c-bar" aria-hidden="true"><i style="width:' + Math.round((T.i + 1) / total * 100) + '%"></i></span></div>' +
      '<h3 id="coach-t">' + esc(st.title) + '</h3><p id="coach-b">' + esc(st.body) + '</p>' +
      '<span class="sr" role="status">مرحله ' + fa(T.i + 1) + ' از ' + fa(total) + ': ' + esc(st.title) + '</span>' +
      '<div class="c-acts"><button type="button" class="skip" data-tour="skip">رد کردن راهنما</button><span class="grow"></span>' +
      (T.i > 0 ? '<button type="button" class="btn btn-sm" data-tour="prev">قبلی</button>' : '') +
      '<button type="button" class="btn btn-sm btn-primary" data-tour="next">' + (last ? 'پایان آموزش' : 'بعدی') + '</button></div>';
    coach.style.visibility = 'hidden';
    setTimeout(function () { place(true); coach.style.visibility = ''; var n = coach.querySelector('[data-tour="next"]'); if (n) n.focus({ preventScroll: true }); }, st.wait || 90);
  }
  function place(scroll) {
    if (!T) return;
    var st = TOURS[T.id].steps[T.i], els = resolve(st.sel), spot = T.els.spot, coach = T.els.coach;
    var vw = innerWidth, vh = innerHeight, pad = 6;
    if (!els) { spot.style.display = 'none'; coach.dataset.pos = 'center'; coach.style.top = Math.round(vh / 2 - coach.offsetHeight / 2) + 'px'; coach.style.left = Math.round(vw / 2 - coach.offsetWidth / 2) + 'px'; return; }
    var r = unionRect(els);
    if (scroll && (r.top < 60 || r.bottom > vh - 20) && r.bottom - r.top < vh * 0.8 && !els[0].closest('#drawer')) {
      els[0].scrollIntoView({ block: 'center', behavior: 'auto' }); r = unionRect(els);
    }
    var top = Math.max(4, r.top - pad), left = Math.max(4, r.left - pad), right = Math.min(vw - 4, r.right + pad), bottom = Math.min(vh - 4, r.bottom + pad);
    spot.style.display = ''; spot.style.top = top + 'px'; spot.style.left = left + 'px'; spot.style.width = (right - left) + 'px'; spot.style.height = (bottom - top) + 'px';
    var cw = coach.offsetWidth, ch = coach.offsetHeight, gap = 12, pos, ct, cl;
    var inDrawer = els[0].id === 'drawer' || !!els[0].closest('#drawer');
    if (inDrawer && vw > 760 && right + gap + cw < vw - 8) { pos = 'side'; ct = Math.max(12, Math.min(vh - ch - 12, 120)); cl = right + gap + 8; }
    else if (vh - bottom >= ch + gap + 8) { pos = 'below'; ct = bottom + gap; }
    else if (top >= ch + gap + 8) { pos = 'above'; ct = top - gap - ch; }
    else { pos = 'center'; ct = Math.max(12, vh - ch - 16); }
    if (cl == null) { cl = Math.round(right - cw); cl = Math.max(12, Math.min(vw - cw - 12, cl)); }
    coach.dataset.pos = pos; coach.style.top = Math.round(ct) + 'px'; coach.style.left = Math.round(cl) + 'px';
    var targetMid = (left + right) / 2, arrowFromRight = Math.round(cl + cw - targetMid - 6);
    coach.style.setProperty('--arrow', Math.max(16, Math.min(cw - 28, arrowFromRight)) + 'px');
  }
  function endTour(completed, silent) {
    if (!T) return;
    var tour = TOURS[T.id], opener = T.opener;
    Object.keys(T.els).forEach(function (k) { T.els[k].remove(); });
    if (completed) store.set(tour.key + '_completed', '1'); else store.set(tour.key + '_dismissed', '1');
    T = null;
    if (silent) return;
    if (B.state.drawer) B.closeDrawer();
    if (completed) B.toast('آموزش «' + tour.title.replace('آموزش ', '') + '» تمام شد. هر زمان از دکمه راهنما قابل تکرار است.', 'success');
    var hb = $('#help-btn'); (opener && document.contains(opener) && opener !== document.body ? opener : hb).focus();
  }
  function moveTour(d) {
    if (!T) return;
    var total = TOURS[T.id].steps.length, n = T.i + d;
    if (n >= total) return endTour(true);
    if (n < 0) return;
    T.i = n; showStep();
  }

  /* ---------- Help Center ---------- */
  var hcView = 'home';
  function tourItem(id) {
    var t = TOURS[id], done = isDone(id);
    return '<button type="button" class="hc-item" data-start-tour="' + id + '"><span class="hc-ico ' + t.tone + '">' + ic(t.icon) + '</span><span class="hc-txt"><b>' + esc(t.title) + '</b><span>' + esc(t.desc) + '</span></span>' +
      '<span class="hc-meta"><span>' + fa(t.steps.length) + ' مرحله · ' + esc(t.dur) + '</span>' + (done ? '<span class="hc-done">' + ic('check') + 'دیده‌شده</span>' : '') + '</span></button>';
  }
  function helpHtml() {
    if (hcView === 'keys') {
      var row = function (keys, label) { return '<div class="kb-row"><span>' + label + '</span><span class="keys">' + keys + '</span></div>'; };
      return '<div class="hc-head"><button type="button" class="hc-back" data-hc="home">' + ic('arrow') + 'بازگشت</button><h2 id="hc-title">میانبرهای کیبورد</h2><button type="button" class="icon-btn" data-hc-close aria-label="بستن راهنما">' + ic('x') + '</button></div>' +
        '<div class="kb-list">' +
        row(kbd('Ctrl') + kbd('K') + ' یا ' + kbd('/'), 'جستجو و فرمان‌ها') +
        row(kbd('J') + kbd('K'), 'مشتری بعدی / قبلی') +
        row(kbd('Enter'), 'باز کردن ردیف یا تأیید اقدام اصلی') +
        row(kbd('Esc'), 'بستن پنل جزئیات، منو یا راهنما') +
        row(kbd('F'), 'رفتن به جستجوی صف') +
        row(kbd('۱') + '–' + kbd('۶') + ' ، ' + kbd('۰'), 'تغییر صف (۰ = همه)') +
        row(kbd('۱') + '–' + kbd('۵'), 'انتخاب نتیجه تماس داخل پنل جزئیات') +
        '</div>';
    }
    return '<div class="hc-head"><h2 id="hc-title">مرکز راهنما</h2><button type="button" class="icon-btn" data-hc-close aria-label="بستن راهنما">' + ic('x') + '</button></div>' +
      '<div class="hc-sub">آموزش‌های کوتاه تعاملی؛ هر زمان قابل تکرارند و هیچ داده‌ای را تغییر نمی‌دهند.</div>' +
      '<div class="hc-list">' + ['seller', 'leads', 'invoices', 'conversions', 'wallet'].map(tourItem).join('') + '<div class="hc-sep"></div>' +
      '<button type="button" class="hc-item" data-hc="keys"><span class="hc-ico neutral">' + ic('key') + '</span><span class="hc-txt"><b>میانبرهای کیبورد</b><span>کار سریع‌تر بدون ماوس</span></span><span class="hc-meta">' + kbd('?') + '</span></button>' +
      '<button type="button" class="hc-item" data-restart-tours><span class="hc-ico neutral">' + ic('replay') + '</span><span class="hc-txt"><b>شروع مجدد آموزش</b><span>آموزش سریع پنل را از ابتدا ببینید</span></span></button></div>' +
      '<div class="hc-foot"><span>راهنما · نسخه ۱</span><span class="grow"></span><button type="button" class="hc-back" data-reset-onboarding>بازنشانی آموزش (دمو)</button></div>';
  }
  function openHelp(view) {
    closeDemo(); B.closeMenu();
    hcView = view || 'home';
    var hc = $('#help-center');
    if (!hc) { hc = document.createElement('div'); hc.id = 'help-center'; hc.className = 'help-center'; hc.setAttribute('role', 'dialog'); hc.setAttribute('aria-labelledby', 'hc-title'); document.body.appendChild(hc); }
    hc.innerHTML = helpHtml(); hc.hidden = false;
    var b = $('#help-btn'), r = b.getBoundingClientRect();
    hc.style.top = (r.bottom + 8) + 'px'; hc.style.left = Math.max(8, Math.min(innerWidth - hc.offsetWidth - 8, r.left - 8)) + 'px';
    b.setAttribute('aria-expanded', 'true');
    var f = hc.querySelector('.hc-list .hc-item') || hc.querySelector('.hc-back') || hc.querySelector('[data-hc-close]'); if (f) f.focus({ preventScroll: true });
  }
  function closeHelp(restore) { var hc = $('#help-center'); if (!hc || hc.hidden) return; hc.hidden = true; $('#help-btn').setAttribute('aria-expanded', 'false'); if (restore) $('#help-btn').focus(); }

  /* ---------- Welcome ---------- */
  function showWelcome() {
    closeHelp(); closeDemo();
    var w = $('#welcome');
    if (!w) { w = document.createElement('div'); w.id = 'welcome'; w.className = 'welcome-wrap'; document.body.appendChild(w); }
    w.innerHTML = '<div class="welcome" role="dialog" aria-modal="true" aria-labelledby="wl-t" aria-describedby="wl-b"><div class="art">' + ic('sparkle') + '</div><h2 id="wl-t">به پنل فروش خوش آمدید</h2>' +
      '<p id="wl-b">در کمتر از ۲ دقیقه بخش‌های اصلی پنل و مسیر کار روزانه را به شما نشان می‌دهیم.</p>' +
      '<div class="acts"><button type="button" class="btn btn-primary btn-lg" data-welcome="start">شروع آموزش</button><button type="button" class="btn btn-ghost btn-lg" data-welcome="later">بعداً</button></div>' +
      '<div class="fine">آموزش همیشه از دکمه راهنما (؟) در بالای صفحه در دسترس است.</div></div>';
    w.hidden = false; w.querySelector('[data-welcome="start"]').focus();
  }
  function closeWelcome(later) {
    var w = $('#welcome'); if (!w) return; w.hidden = true; store.set('seller_welcome_v1_seen', '1');
    if (later) { B.toast('هر زمان خواستید، آموزش از دکمه راهنما (؟) در دسترس است.', 'info'); $('#help-btn').focus(); }
  }

  /* ---------- Demo panel ---------- */
  function demoHtml() {
    var b = function (act, label) { return '<button type="button" class="dp" data-dp="' + act + '">' + label + '</button>'; };
    return '<div class="dp-h">' + ic('flask') + '<span>حالت‌های نمایشی</span><button type="button" class="icon-btn" data-dp="close" aria-label="بستن">' + ic('x') + '</button></div>' +
      '<div class="dp-g">آموزش</div>' + b('welcome', 'نمایش خوش‌آمد') + b('tour:seller', 'آموزش پنل فروشنده') + b('tour:leads', 'آموزش شماره‌های من') + b('tour:invoices', 'آموزش فاکتورها') + b('tour:conversions', 'آموزش آماده‌های تبدیل') + b('tour:wallet', 'آموزش کیف پول') +
      '<div class="dp-g">وضعیت‌های سیستم</div>' + b('sim:pageError', 'خطای صفحه') + b('sim:tableError', 'خطای جدول') + b('sim:drawerError', 'خطای پنل جزئیات') + b('drawerWarn', 'هشدار داخل پنل جزئیات') + b('sim:empty', 'صف خالی') + b('sim:noresult', 'بدون نتیجه جستجو') + b('sim:loading', 'در حال بارگذاری (Skeleton)') + b('sim:offline', 'قطع اتصال') +
      '<div class="dp-g">اعلان‌ها</div>' + b('toast:success', 'Toast موفق') + b('toast:error', 'Toast خطا') + b('toast:warning', 'Toast هشدار') +
      '<div class="dp-g">بازنشانی</div>' + b('normal', 'بازگشت به حالت عادی') + b('reset', 'Reset onboarding') +
      '<div class="dp-note">این کنترل‌ها فقط برای بررسی نمونه هستند و بخشی از CRM نیستند.</div>';
  }
  function openDemo() {
    closeHelp(); B.closeMenu();
    var d = $('#demo-panel');
    if (!d) { d = document.createElement('div'); d.id = 'demo-panel'; d.className = 'demo-panel'; d.setAttribute('role', 'dialog'); d.setAttribute('aria-label', 'حالت‌های نمایشی'); document.body.appendChild(d); }
    d.innerHTML = demoHtml(); d.hidden = false;
    var u = $('#user-btn').getBoundingClientRect();
    d.style.top = (u.bottom + 8) + 'px'; d.style.left = Math.max(8, u.left) + 'px';
    d.querySelector('.dp').focus();
  }
  function closeDemo() { var d = $('#demo-panel'); if (d) d.hidden = true; }
  function resetOnboarding() {
    Object.keys(TOURS).forEach(function (id) { store.del(TOURS[id].key + '_completed'); store.del(TOURS[id].key + '_dismissed'); });
    store.del('seller_welcome_v1_seen');
    B.toast('وضعیت آموزش‌ها بازنشانی شد. با بارگذاری دوباره، پیام خوش‌آمد نمایش داده می‌شود.', 'info');
  }
  function runDemo(act) {
    closeDemo();
    if (act === 'close') return;
    if (act === 'welcome') return showWelcome();
    if (act.indexOf('tour:') === 0) return startTour(act.slice(5));
    if (act.indexOf('toast:') === 0) {
      var t = act.slice(6);
      return B.toast({ success: 'نتیجه تماس با موفقیت ثبت شد.', error: 'ثبت انجام نشد؛ اتصال را بررسی کنید و دوباره تلاش کنید.', warning: 'این پرونده تا ۲ روز دیگر به بایگانی منتقل می‌شود.' }[t], t);
    }
    if (act === 'drawerWarn') { B.setSim(null); if (B.state.view !== 'leads') B.go('leads'); return B.openDrawer('lead', 9043346); }
    if (act === 'normal') return B.setSim(null);
    if (act === 'reset') return resetOnboarding();
    if (act === 'sim:drawerError') { B.setSim(null); if (B.state.view !== 'leads') B.go('leads'); B.state.sim = 'drawerError'; return B.openDrawer('lead', B.firstLeadId()); }
    if (act.indexOf('sim:') === 0) return B.setSim(act.slice(4));
  }

  /* ---------- Events ---------- */
  document.addEventListener('click', function (e) {
    var t = e.target.closest('button, a, .tour-catch'); var inHelp = e.target.closest('#help-center'), inDemo = e.target.closest('#demo-panel');
    if (!t) { if (!inHelp) closeHelp(); if (!inDemo) closeDemo(); return; }
    if (t.matches('.tour-catch')) return;
    if (t.matches('[data-tour]')) { var a = t.getAttribute('data-tour'); if (a === 'next') moveTour(1); else if (a === 'prev') moveTour(-1); else endTour(false); return; }
    if (t.id === 'help-btn') { e.stopPropagation(); var hc = $('#help-center'); if (hc && !hc.hidden) closeHelp(true); else openHelp(); return; }
    if (t.matches('[data-help-open]')) { e.stopPropagation(); return openHelp(); }
    if (t.matches('[data-demo-panel]')) { e.stopPropagation(); return openDemo(); }
    if (t.matches('[data-hc-close]')) return closeHelp(true);
    if (t.matches('[data-hc]')) { hcView = t.getAttribute('data-hc'); var h = $('#help-center'); h.innerHTML = helpHtml(); var f = h.querySelector('.hc-back, .hc-item'); if (f) f.focus(); return; }
    if (t.matches('[data-start-tour]')) return startTour(t.getAttribute('data-start-tour'));
    if (t.matches('[data-restart-tours]')) { Object.keys(TOURS).forEach(function (id) { store.del(TOURS[id].key + '_completed'); }); return startTour('seller'); }
    if (t.matches('[data-reset-onboarding]')) { closeHelp(true); return resetOnboarding(); }
    if (t.matches('[data-page-tour]')) { e.stopPropagation(); return startTour(PAGE_TOUR[t.getAttribute('data-page-tour')] || 'seller'); }
    if (t.matches('[data-welcome]')) { var st = t.getAttribute('data-welcome') === 'start'; closeWelcome(!st); if (st) startTour('seller'); return; }
    if (t.matches('[data-dp]')) return runDemo(t.getAttribute('data-dp'));
    if (!inHelp) closeHelp(); if (!inDemo) closeDemo();
  }, true);

  // Overlays own the keyboard while open (capture phase, before the app's shortcuts).
  window.addEventListener('keydown', function (e) {
    var w = $('#welcome');
    if (w && !w.hidden) { if (e.key === 'Escape') { e.preventDefault(); closeWelcome(true); } else if (e.key === 'Tab') trap(e, w); e.stopImmediatePropagation(); return; }
    if (T) {
      if (e.key === 'Escape') { e.preventDefault(); endTour(false); }
      else if (e.key === 'ArrowLeft') { e.preventDefault(); moveTour(1); }
      else if (e.key === 'ArrowRight') { e.preventDefault(); moveTour(-1); }
      else if (e.key === 'Tab') trap(e, T.els.coach);
      e.stopImmediatePropagation(); return;
    }
    var hc = $('#help-center');
    if (hc && !hc.hidden) { if (e.key === 'Escape') { e.preventDefault(); closeHelp(true); } else if (e.key === 'Tab') trap(e, hc); if (e.key !== 'Enter' && e.key !== ' ') e.stopImmediatePropagation(); return; }
    var d = $('#demo-panel');
    if (d && !d.hidden) { if (e.key === 'Escape') { e.preventDefault(); closeDemo(); $('#user-btn').focus(); } else if (e.key === 'Tab') trap(e, d); if (e.key !== 'Enter' && e.key !== ' ') e.stopImmediatePropagation(); return; }
  }, true);
  function trap(e, box) {
    var f = Array.prototype.slice.call(box.querySelectorAll('button, [href], input, [tabindex]:not([tabindex="-1"])')).filter(function (x) { return !x.disabled && x.getClientRects().length; });
    if (!f.length) return;
    var first = f[0], last = f[f.length - 1], a = document.activeElement;
    if (e.shiftKey && (a === first || !box.contains(a))) { e.preventDefault(); last.focus(); }
    else if (!e.shiftKey && (a === last || !box.contains(a))) { e.preventDefault(); first.focus(); }
  }
  var raf = 0;
  var reflow = function () { if (!T) return; cancelAnimationFrame(raf); raf = requestAnimationFrame(function () { place(false); }); };
  window.addEventListener('resize', reflow); window.addEventListener('scroll', reflow, true);

  /* ---------- Boot: deep links, then first-visit welcome ---------- */
  var q = new URLSearchParams(location.search);
  var deep = ['lead', 'inv', 'case', 'menu', 'palette', 'sim', 'tour', 'help', 'demo', 'toast'].some(function (k) { return q.has(k); });
  setTimeout(function () {
    if (q.get('tour')) startTour(q.get('tour'), Number(q.get('step') || 1));
    else if (q.get('help')) openHelp(q.get('help') === 'shortcuts' ? 'keys' : 'home');
    else if (q.get('demo')) openDemo();
    if (q.get('toast')) { var tt = q.get('toast'); B.toast({ success: 'نتیجه تماس با موفقیت ثبت شد.', error: 'ثبت انجام نشد؛ اتصال را بررسی کنید و دوباره تلاش کنید.', warning: 'این پرونده تا ۲ روز دیگر به بایگانی منتقل می‌شود.' }[tt], tt, true); }
    if (q.get('welcome') === '1' || (!deep && q.get('welcome') !== '0' && store.get('seller_welcome_v1_seen') !== '1')) showWelcome();
  }, 60);
  window.SNHelp = { startTour: startTour, openHelp: openHelp, showWelcome: showWelcome, reset: resetOnboarding, tours: TOURS };
})();
