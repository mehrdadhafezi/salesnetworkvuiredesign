/* Senior Supervisor — role layer, part 4: remaining drawers, events, palette, boot.
   Everything runs on the shared CRM runtime (../shared/crm-core.js). Prototype only: no data is saved or sent. */
(function () {
  'use strict';
  var X = window.SENX, C = X.C, M = X.M, h = X.h, st = X.st, V = X.V, D = X.D;
  var $ = h.$, esc = h.esc, fa = h.fa, money = h.money, ic = h.ic, pill = h.pill, hint = h.hint;
  var seller = X.seller, team = X.team, keys = X.keys, top = X.top, foot = X.foot, own = X.own, checks = X.checks, tl = X.tl;

  V.cases = V.cases; V.dist = V.dist;

  /* ---------- Remaining drawers ---------- */
  function facets(i) {
    var s = X.INVS[i.inv], rv = X.REV[i.review], done = i.inv === 'completed' && i.review === 'approved';
    var f = [['وضعیت فاکتور', pill(s.label, s.tone, s.icon)], ['مرحله پرداخت', '<b>مرحله ' + fa(i.stage) + ' از ' + fa(i.stages) + '</b><span>' + esc(X.STG[i.stg]) + '</span>'], ['مدرک پرداخت', '<b>' + esc(X.EVID[i.evidence]) + '</b>' + (i.evidence === 'receipt' ? '<span>ثبت رسید ≠ تأیید پرداخت</span>' : '')], ['بررسی مالی', rv ? pill(rv.label, rv.tone, rv.icon) : '<b class="muted">ارسال نشده</b>'], ['فروش تکمیل‌شده', done ? pill('بله', 'green', 'checkCircle') : '<b class="muted">خیر</b>'], ['اقدام بعدی با', i.next ? X.nextActor(i.next, i) : '<b class="muted">—</b>']];
    return '<div class="facets">' + f.map(function (x) { return '<div class="facet"><span class="f-l">' + x[0] + '</span><div class="f-v">' + x[1] + '</div></div>'; }).join('') + '</div>';
  }
  D.inv = function (code) {
    var i = X.invOf(code), s = seller(i.seller), tNow = s.team;
    var actions = [['خواندن', 'جزئیات، مراحل و سوابق مالی', 'ok', 'مجاز در محدوده شما'], ['کمک (نیازمند مجوز صریح)', 'کپی لینک پرداخت · ارسال مجدد لینک · ثبت واریز و فیش · ویرایش پیش از پرداخت', 'pending', 'مجوز تأیید نشده (SD-03)'], ['فقط واحد مالی', 'تأیید / رد / لغو / بازگشایی / بازپرداخت / ثبت پورسانت', 'locked', 'در این پنل ممکن نیست']];
    var acts = '<ul class="act-policy">' + actions.map(function (a) { return '<li class="ap-' + a[2] + '"><span class="ap-ic">' + ic(a[2] === 'ok' ? 'checkCircle' : a[2] === 'pending' ? 'question' : 'lock') + '</span><span class="ap-t"><b>' + a[0] + '</b><span>' + a[1] + '</span></span><span class="ap-s">' + a[3] + '</span></li>'; }).join('') + '</ul>';
    var waitTxt = i.review === 'pending' ? 'منتظر بررسی مالی — اقدامی از سمت شما لازم نیست' : i.inv === 'mismatch' ? 'منتظر تطبیق — مسئول: MIS / مالی' : i.next && i.next.who === 'seller' ? 'اقدام بعدی با ' + s.name : '';
    return top('فاکتور') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name" class="mono">' + fa(i.code) + '</h2>' + pill(X.INVS[i.inv].label, X.INVS[i.inv].tone, X.INVS[i.inv].icon) + X.link(i.link) + '</div><div class="dr-meta"><span style="color:var(--text)">' + esc(i.customer) + '</span><span>فروشنده: ' + esc(s.name) + '</span><span>تیم فعلی: ' + esc(X.teamName(tNow)) + '</span><span>صدور: ' + esc(i.issued) + '</span></div></div>' +
      '<div class="dr-body"><section class="sec primary"><div class="sum-grid"><div class="sum"><span>مبلغ کل</span><b>' + money(i.total) + '</b></div><div class="sum"><span>پرداخت‌شده</span><b style="color:var(--green-fg)">' + money(i.paid) + '</b></div><div class="sum"><span>مانده</span><b>' + money(i.inv === 'cancelled' ? 0 : i.total - i.paid) + '</b></div></div>' +
      (i.reason ? '<div class="note danger" style="margin-top:12px">' + ic('alert') + '<span>دلیل رد مالی: ' + esc(i.reason) + ' · اقدام بعدی با ' + esc(s.name) + '</span></div>' : '') + (i.mismatch ? '<div class="note warn" style="margin-top:12px">' + ic('question') + '<span>' + esc(i.mismatch) + '. تا تطبیق، وضعیت «تکمیل‌شده» قطعی اعلام نمی‌شود.</span></div>' : '') + (i.linkNote ? '<div class="note warn" style="margin-top:12px">' + ic('layers') + '<span>' + esc(i.linkNote) + '</span></div>' : '') + '</section>' +
      X.sec('وضعیت مالی به تفکیک', '', facets(i)) +
      X.sec('مالکیت و انتساب', '', own({ custody: s.name, original: s.name, next: i.next ? i.next.text : '—', event: s.name + ' (صدور · ' + i.issued + ')', credit: 'مالک اعتبار: ' + s.name + ' (طبق قوانین مالی فعلی)' }) + '<div class="basis-pair"><span>تیم فعلی: <b>' + esc(X.teamName(tNow)) + '</b> ' + X.basis('current') + '</span><span>تیم هنگام صدور: <b>' + (i.teamAtIssue == null ? 'ناشناخته' : esc(X.teamName(i.teamAtIssue))) + '</b> ' + X.basis('hist') + '</span></div>') +
      X.details('اقدام‌های مجاز شما', null, acts + '<div class="note info" style="margin-top:12px">' + ic('shield') + '<span>مجوز «کمک» از نقش‌های دیگر به ارث نمی‌رسد و نمایش یک کنترل به معنی اجرای موفق یا مجوز نیست. در این نمونه هیچ اقدامی اجرا نمی‌شود.</span></div>') +
      X.details('سوابق مالی', null, tl((i.review === 'rejected' ? [['رد توسط مالی: ' + i.reason, 'کارشناس مالی · ۱۴۰۵/۰۷/۰۵'], ['ثبت رسید مرحله ۱', s.name + ' · ۱۴۰۵/۰۷/۰۴']] : i.review === 'pending' ? [['ثبت رسید', s.name + ' · ۱۴۰۵/۰۷/۰۶']] : []).concat([['صدور ' + (i.inv === 'pre' ? 'پیش‌فاکتور' : 'فاکتور'), s.name + ' · ' + i.issued]]))) + '</div>' +
      foot(waitTxt ? '<span class="wait-note">' + ic('hourglass') + esc(waitTxt) + '</span>' : '', null);
  };
  D.rcase = function (id) {
    var r = M.ready.filter(function (x) { return x.id === id; })[0], s = seller(r.seller), t = r.team === 'direct' ? null : team(r.team), cs = X.CONVS[r.status];
    var respName = r.direct ? 'کامران صدری (شما)' : t.active ? t.sup : 'نامشخص';
    return top('آماده تبدیل — نمای تیم') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name" class="mono">' + r.id + '</h2>' + pill(cs.label, cs.tone, cs.icon) + X.src(r.source) + '</div><div class="dr-meta"><span style="color:var(--text)">' + esc(r.customer) + '</span><span>فروشنده: ' + esc(s.name) + '</span></div></div>' +
      '<div class="dr-body"><section class="sec primary"><div class="note info">' + ic('eye') + '<span><b>نمای نظارتی.</b> تخصیص تبدیل از این پنل ممکن نیست. مسئول این پرونده <b>' + esc(respName) + '</b> است.</span></div>' + (r.unresolved ? '<div class="note warn" style="margin-top:12px">' + ic('question') + '<span><b>مسئول نامشخص.</b> ' + esc(r.unresolved) + '؛ پرونده خودکار به شما واگذار نمی‌شود.</span></div>' : '') + '</section>' +
      '<section class="sec">' + own({ custody: s.name + ' (فروشنده)', original: s.name, next: r.next, event: 'تأیید مالی · ثبت‌شده در فاکتور', credit: 'مالک اعتبار: ' + s.name }) + '</section>' +
      X.sec('زمینه مالی مجاز', '', checks([['ok', r.stage, 'جزئیات حساس مالی در این نما نمایش داده نمی‌شود'], r.choice ? ['info', 'انتخاب مشتری: ' + r.choice, ''] : ['info', 'انتخاب مشتری: ثبت نشده', '«انتخاب نکرده» با آماده‌بودن تأییدشده یکی نیست']])) +
      X.sec('مدت انتظار', '', checks([['q', 'تعریف نشده', 'زمان شروع انتظار معتبر و سیاست سررسید هنوز تصویب نشده است (SD-05)؛ عدد ساخته نمی‌شود.']])) +
      X.sec('ارجاع رسمی', '', checks([['info', 'دستور ارجاع در این مرحله تعریف نشده است', 'فقط مسئول اقدام نشان داده می‌شود (SG-02).']])) + '</div>' + foot('', null);
  };
  D.ccase = function (id) {
    var c = M.cases.filter(function (x) { return x.id === id; })[0], s = c.seller ? seller(c.seller) : null;
    return top('پرونده و سوابق منبع') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name" class="mono">' + c.id + '</h2>' + X.src(c.src) + X.link(c.link) + '</div><div class="dr-meta"><span class="mono phone-num">' + fa(c.phone) + '</span><span>' + (s ? 'فروشنده: ' + esc(s.name) : esc(X.teamName(c.team))) + '</span></div></div>' +
      '<div class="dr-body"><section class="sec primary"><div class="note warn">' + ic('layers') + '<span><b>پوشش ناقص.</b> نمای یکپارچه تأیید نشده؛ فقط سوابق همین منبع نمایش داده می‌شود. اتصال به منابع دیگر حدس زده نمی‌شود.' + (c.note ? ' ' + esc(c.note) + '.' : '') + '</span></div></section>' +
      X.sec('وضعیت', '', '<div class="facets"><div class="facet"><span class="f-l">وضعیت خام منبع</span><div class="f-v"><b>' + esc(c.raw) + '</b></div></div><div class="facet"><span class="f-l">برچسب نگاشت‌شده</span><div class="f-v"><b>' + esc(c.mapped) + '</b><span>نگاشت اطمینان محدود دارد</span></div></div></div>') +
      X.sec('شماره موبایل', '', checks([['info', 'شماره فقط ابزار جستجوست', 'چند پرونده مجزا ممکن است شماره یکسان داشته باشند و ادغام نمی‌شوند']])) +
      X.details('سوابق منبع', c.hist.length, tl(c.hist), true) + '</div>' + foot('', null);
  };
  D.diag = function () {
    var rows = M.recon.map(function (r) { return '<tr><td>' + esc(r.metric) + '</td><td>' + esc(r.old) + '</td><td>' + esc(r.now) + '</td><td>' + esc(r.diff) + '</td><td>' + X.cov(r.state) + '<div class="cell-sub">' + esc(r.reason) + '</div></td></tr>'; }).join('');
    return top('تطبیق شاخص‌ها (پیشرفته)') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">تفاوت تعریف قدیم و جدید</h2></div><div class="dr-meta"><span>برای تشخیص نیازمند تطبیق · عدد دستی تغییر نمی‌کند</span></div></div><div class="dr-body">' +
      '<section class="sec primary"><div class="tbl-wrap"><table class="tbl no-cursor stackable" aria-label="تطبیق شاخص"><thead><tr><th>شاخص</th><th>تعریف قدیمی</th><th>تعریف جدید</th><th>تفاوت</th><th>وضعیت و دلیل</th></tr></thead><tbody>' + rows + '</tbody></table></div></section>' +
      X.sec('توجه', '', checks([['info', 'این صفحه ابزار تعمیر نیست', 'هیچ عددی بازنویسی یا تصحیح نمی‌شود؛ فقط علت اختلاف نشان داده می‌شود.'], ['info', 'مسئول تطبیق', 'MIS / مالک شاخص فروش / مهندسی']])) + '</div>' + foot('', null);
  };

  /* ---------- Events ---------- */
  function goAttn(a) {
    if (a.aq) st.aq = a.aq; if (a.rf) st.rf = a.rf; if (a.iq) st.iq = a.iq; if (a.pm) st.pm = a.pm; if (a.id === 'link') st.src = 'legacy';
    C.go(a.go === 'case' ? 'cases' : a.go);
  }
  function onClick(t, e) {
    var a = t.getAttribute('data-act');
    var tog = { 'data-aq': 'aq', 'data-iq': 'iq', 'data-rf': 'rf', 'data-amode': 'amode', 'data-rbasis': 'rbasis', 'data-pm': 'pm', 'data-pb': 'pb', 'data-rg': 'rg', 'data-hq': 'hq', 'data-src': 'src' };
    for (var attr in tog) if (t.hasAttribute(attr)) { st[tog[attr]] = t.getAttribute(attr); C.state.cursor = 0; if (C.state.drawer) C.closeDrawer(); C.render(); return true; }
    if (t.matches('[data-hrtype]')) { st.hrForm.type = t.getAttribute('data-hrtype'); st.hrErr = null; C.render(); return true; }
    if (t.matches('[data-kpi-filter]')) { var kf = t.getAttribute('data-kpi-filter').split(':'); if (kf[0] === 'iq') { st.iq = kf[1]; C.render(); } else if (kf[0] === 'perf') { st.pm = kf[1]; C.go('perf'); } else if (kf[0] === 'dist') { st.aq = kf[1]; C.go('dist'); } return true; }
    if (t.matches('[data-attn]')) { C.openDrawer('attn', t.getAttribute('data-attn')); return true; }
    if (!a) return false;
    var arg = a.indexOf(':') > -1 ? a.slice(a.indexOf(':') + 1) : null, verb = arg != null ? a.slice(0, a.indexOf(':')) : a;
    e.preventDefault(); e.stopPropagation();
    switch (verb) {
      case 'goto': C.closeMenu(); C.go(arg); return true;
      case 'attn-go': var A = X.attention().filter(function (x) { return x.id === arg; })[0]; if (A) goAttn(A); return true;
      case 'open-team': C.closeMenu(); C.openDrawer('team', arg); return true;
      case 'open-seller': C.closeMenu(); C.openDrawer('seller', arg); return true;
      case 'drill': C.closeMenu(); st.pscope = arg; st.pm = 'seller'; C.go('perf'); return true;
      case 'pscope': st.pscope = arg === 'all' ? null : arg; C.render(); return true;
      case 'assign-to': C.closeMenu(); st.aq = 'assign'; st.amode = 'select'; st.recip = arg; C.go('dist'); C.toast('گیرنده انتخاب شد: ' + team(arg).sup + '. پرونده‌ها را انتخاب کنید.', 'info'); return true;
      case 'hr-for': C.closeMenu(); st.hrForm = { seller: arg, type: 'transfer', dest: '', reason: '' }; st.hrErr = null; C.go('hr'); return true;
      case 'clear-asel': st.asel = {}; C.render(); return true;
      case 'clear-rsel': st.rsel = {}; C.render(); return true;
      case 'pick-one': st.asel[arg] = true; C.closeDrawer(); C.render(); return true;
      case 'review-assign': if (t.disabled) return true; C.openDrawer('assignReview', 'x'); return true;
      case 'refresh-review': delete st.asel['C-31003']; M.pool = M.pool.filter(function (c) { return c.id !== 'C-31003'; }); st.flow = null; C.render(); C.openDrawer('assignReview', 'x'); C.toast('بررسی بازخوانی شد؛ ۱ پرونده از انتخاب حذف شد.', 'warning'); return true;
      case 'commit-assign': if (t.disabled) return true; X.runAssign(st.flow); return true;
      case 'open-op': C.openDrawer('result', arg); return true;
      case 'retry': X.retryFailed(arg); C.render(); C.openDrawer('result', arg); C.toast('تلاش مجدد فقط برای موارد ناموفق انجام شد (نمایشی).', 'success'); return true;
      case 'reconcile': X.reconcile(arg); C.render(); C.openDrawer('result', arg); C.toast('تطبیق انجام شد؛ نتیجه واقعی هر مورد مشخص شد (نمایشی).', 'info'); return true;
      case 'open-ret': C.openDrawer('ret', arg); return true;
      case 'refresh-ret': var rc = M.held.filter(function (x) { return x.id === arg; })[0]; rc.elig = 'ok'; rc.why = null; C.render(); C.openDrawer('ret', arg); C.toast('پرونده بازخوانی شد؛ وضعیت فعلی نمایش داده می‌شود.', 'info'); return true;
      case 'review-return': C.openDrawer('returnReview', 'x'); return true;
      case 'commit-return': if (t.disabled) return true; X.runReturn(keys(st.rsel).filter(function (i) { return M.held.some(function (c) { return c.id === i; }); })); return true;
      case 'commit-return-one': if (t.disabled) return true; X.runReturn([arg]); return true;
      case 'open-inv': C.closeMenu(); C.openDrawer('inv', arg); return true;
      case 'open-rcase': C.openDrawer('rcase', arg); return true;
      case 'open-case': C.openDrawer('ccase', arg); return true;
      case 'open-diag': C.openDrawer('diag', 'x'); return true;
      case 'open-hr': C.openDrawer('hr', arg); return true;
      case 'hr-step': st.hrDone[arg] = 'step'; C.closeDrawer(); C.render(); C.toast('مرحله شما تأیید شد و درخواست به مرحله بعد رفت (نمایشی). هنوز اعمال نشده است؛ اعمال نهایی با منابع انسانی است.', 'success'); return true;
      case 'hr-reject': var note = $('[data-hrnote]'), nv = note ? note.value.trim() : ''; if (!nv) { C.state.drawer.keep.err = true; C.state.drawer.keep.note = ''; C.rerenderDrawer(); var nn = $('[data-hrnote]'); if (nn) nn.focus(); return true; } st.hrDone[arg] = 'rejected'; C.closeDrawer(); C.render(); C.toast('درخواست با دلیل رد شد (نمایشی).', 'info'); return true;
      case 'hr-submit':
        var f = st.hrForm, E = {};
        if (!f.seller) E.seller = 'یک فروشنده انتخاب کنید.';
        if (f.type === 'transfer' && !f.dest) E.dest = 'سرپرست مقصد را انتخاب کنید.';
        if (!String(f.reason).trim()) E.reason = 'نوشتن دلیل برای این درخواست الزامی است.';
        st.hrErr = Object.keys(E).length ? E : null; C.render();
        if (st.hrErr) { var first = $('[aria-invalid="true"]'); if (first) first.focus(); return true; }
        C.toast('درخواست ثبت شد (نمایشی). وضعیت: در انتظار بررسی مرحله‌ای — هنوز اعمال نشده است.', 'success'); st.hrForm = null; C.render(); return true;
      case 'open-report': st.report = arg; C.render(); return true;
      case 'close-report': st.report = null; C.render(); return true;
      case 'rteam-all': st.rteam = 'all'; C.render(); return true;
      case 'retry-team': C.setSim(null); C.toast('داده تیم ' + team('T3').sup + ' دوباره دریافت شد (نمایشی).', 'success'); return true;
    }
    return false;
  }
  document.addEventListener('change', function (e) {
    var t = e.target;
    if (t.matches('[data-sel]')) { st[t.getAttribute('data-sel')][t.getAttribute('data-id')] = t.checked; keepFocus(t); return; }
    if (t.matches('[data-selall]')) { var k = t.getAttribute('data-selall'), ids = k === 'asel' ? X.poolOk().map(function (c) { return c.id; }) : M.held.filter(function (c) { return c.elig === 'ok'; }).map(function (c) { return c.id; }); ids.forEach(function (id) { st[k][id] = t.checked; }); keepFocus(t); return; }
    if (t.matches('[data-recip]')) { var id = t.getAttribute('data-recip'); if (st.amode === 'select') { st.recip = id; if (id.charAt(0) === 'S') st.exc = true; } else st.recips[id] = t.checked; keepFocus(t); return; }
    if (t.matches('[data-confirm]')) { C.state.drawer.keep.confirm = t.checked; C.rerenderDrawer(); var c = $('#drawer [data-confirm]'); if (c) c.focus(); return; }
    if (t.matches('[data-hr]')) { st.hrForm[t.getAttribute('data-hr')] = t.value; if (st.hrErr) delete st.hrErr[t.getAttribute('data-hr')]; if (t.getAttribute('data-hr') === 'dest') C.render(); return; }
    if (t.matches('[data-per]')) { st.per = Math.max(1, Math.min(50, Number(t.value) || 1)); keepFocus(t); return; }
    if (t.matches('[data-rteam]')) { st.rteam = t.value; C.render(); }
  });
  document.addEventListener('input', function (e) { var t = e.target; if (t.matches('[data-hr="reason"]')) st.hrForm.reason = t.value; if (t.matches('[data-excreason]')) st.excReason = t.value; });
  document.addEventListener('toggle', function (e) { if (e.target.matches && e.target.matches('details[data-exc]')) st.exc = e.target.open; }, true);
  function keepFocus(t) {
    var sel = t.hasAttribute('data-id') ? '[data-sel="' + t.getAttribute('data-sel') + '"][data-id="' + t.getAttribute('data-id') + '"]' : t.hasAttribute('data-selall') ? '[data-selall="' + t.getAttribute('data-selall') + '"]' : t.hasAttribute('data-recip') ? '[data-recip="' + t.getAttribute('data-recip') + '"]' : '[data-per]';
    var y = window.scrollY; C.render(); window.scrollTo(0, y); var n = $(sel); if (n) n.focus({ preventScroll: true });
  }

  /* ---------- Palette ---------- */
  function palette(nq, match) {
    var out = [];
    if (!nq) return [
      { g: 'اقدام سریع', icon: 'users', label: 'نمای چندتیمی', run: function () { C.go('ov'); } }, { g: 'اقدام سریع', icon: 'send', label: 'تخصیص به سرپرست', run: function () { st.aq = 'assign'; C.go('dist'); } },
      { g: 'اقدام سریع', icon: 'repeat', label: 'بررسی برگشت پرونده', run: function () { st.aq = 'return'; C.go('dist'); } }, { g: 'اقدام سریع', icon: 'compass', label: 'کاوشگر عملکرد', run: function () { C.go('perf'); } },
      { g: 'اقدام سریع', icon: 'chart', label: 'تطبیق شاخص‌ها (پیشرفته)', run: function () { C.openDrawer('diag', 'x'); } }, { g: 'اقدام سریع', icon: 'briefcase', label: 'درخواست‌های HR منتظر بررسی', run: function () { st.hq = 'review'; C.go('hr'); } }
    ];
    M.teams.forEach(function (t) { if (match(t.sup)) out.push({ g: 'تیم‌ها', icon: 'users', label: 'تیم ' + t.sup, meta: t.active ? 'سرپرست مستقیم' : 'غیرفعال', run: function () { C.go('ov'); C.openDrawer('team', t.id); } }); });
    M.sellers.forEach(function (s) { if (match(s.name)) out.push({ g: 'فروشندگان', icon: 'user', label: s.name, meta: s.rel === 'direct' ? 'مستقیم' : 'غیرمستقیم · ' + X.teamName(s.team), run: function () { C.go('perf'); C.openDrawer('seller', s.id); } }); });
    M.invoices.forEach(function (i) { if (match(i.code) || match(i.customer)) out.push({ g: 'فاکتورها', icon: 'receipt', label: 'فاکتور ' + fa(i.code), meta: i.customer, run: function () { st.iq = 'all'; C.go('inv'); C.openDrawer('inv', i.code); } }); });
    M.held.forEach(function (c) { if (match(c.id)) out.push({ g: 'پرونده‌ها', icon: 'file', label: c.id, meta: 'نزد ' + X.holderName(c), run: function () { st.aq = 'return'; st.rf = 'all'; C.go('dist'); C.openDrawer('ret', c.id); } }); });
    return out;
  }

  /* ---------- Boot ---------- */
  C.boot({
    home: 'ov', titleSuffix: 'پنل سرپرست ارشد (نمونه)', palettePlaceholder: 'نام سرپرست یا فروشنده، شماره پرونده، کد فاکتور یا یک فرمان…',
    nav: function () {
      var A = X.attention().filter(function (a) { return a.id !== 'var'; }).length;
      return [
        { id: 'ov', label: 'نمای چندتیمی', icon: 'users', group: 'work', mobile: 'نما', count: function () { return A; }, alert: true },
        { id: 'perf', label: 'ساختار و عملکرد', icon: 'compass', group: 'work', mobile: 'عملکرد' },
        { id: 'dist', label: 'توزیع و برگشت', icon: 'swap', group: 'work', mobile: 'توزیع', count: function () { return X.poolOk().length; } },
        { id: 'ready', label: 'آماده‌های تبدیل تیم‌ها', icon: 'repeat', group: 'work', count: function () { return M.ready.filter(function (r) { return r.status === 'ready'; }).length; } },
        { id: 'inv', label: 'فاکتورها و استثناها', icon: 'receipt', group: 'work', mobile: 'فاکتورها', count: function () { return M.invoices.filter(X.IQ[0].f).length; }, alert: true },
        { id: 'hr', label: 'درخواست‌های HR', icon: 'briefcase', group: 'ctx', count: function () { return M.hr.filter(function (r) { return r.state === 'pending_review' && r.reviewer === 'self' && !st.hrDone[r.id]; }).length; }, alert: true },
        { id: 'rep', label: 'گزارش‌ها', icon: 'chart', group: 'ctx' },
        { id: 'cases', label: 'پرونده و سوابق منبع', icon: 'search', group: 'ctx' },
        { id: 'wallet', label: 'کیف پول من', icon: 'wallet', group: 'me' },
        { id: 'mine', label: 'تبدیل‌های من', icon: 'user', group: 'me', hidden: !st.mineOn && C.state.view !== 'mine' }
      ];
    },
    views: V, drawers: D, wideDrawers: ['inv', 'result', 'assignReview', 'diag'],
    pageTours: { ov: 'senior', perf: 'perf', dist: 'dist' },
    workload: function () { var A = X.attention().filter(function (a) { return a.id !== 'var'; }); return '<span>' + ic('users') + '</span><span><b>' + fa(M.teams.filter(function (t) { return t.active; }).length) + '</b> تیم فعال</span><span class="sep extra"></span><span class="due extra">' + ic('inbox') + ' <b style="color:inherit">' + fa(A.length) + '</b> نیازمند هماهنگی</span>'; },
    userMenuTop: function () { return [{ label: 'کیف پول من', icon: 'wallet', go: 'wallet' }, { label: 'تطبیق شاخص‌ها (پیشرفته)', icon: 'chart', act: 'open-diag' }]; },
    empty: {
      ov: ['تیمی در محدوده شما ثبت نشده', 'هنوز سرپرستی زیر نظر شما تعریف نشده است.', ''], perf: ['داده‌ای در این محدوده نیست', 'محدوده یا حالت دیگری را انتخاب کنید.', '<button type="button" class="btn btn-soft" data-act="pscope:all">همه تیم‌ها</button>'],
      dist: ['پرونده‌ای در پنل شما نیست', 'هنوز پرونده‌ای برای توزیع به شما تحویل نشده است.', ''], ready: ['پرونده آماده تبدیلی در تیم‌ها نیست', 'پرونده‌ها پس از تکمیل پرداخت و تأیید مالی اینجا می‌آیند.', ''],
      inv: ['فاکتوری در این صف نیست', 'صف دیگری را انتخاب کنید.', '<button type="button" class="btn btn-soft" data-iq="all">همه فاکتورها</button>'], hr: ['درخواستی نیست', '', ''], cases: ['پرونده‌ای در این منبع نیست', '', ''], wallet: ['تراکنشی ثبت نشده است', '', ''], rep: ['گزارشی در دسترس نیست', '', '']
    },
    onClick: onClick, palette: palette,
    init: function (state, qs) {
      ['pm', 'pb', 'aq', 'iq', 'hq', 'rf', 'rg', 'src', 'report', 'amode'].forEach(function (k) { if (qs.get(k)) st[k === 'report' ? 'report' : k] = qs.get(k); });
      if (qs.get('scope')) st.pscope = qs.get('scope'); if (qs.get('mine') === '1') st.mineOn = true;
    },
    afterBoot: function (qs) { setTimeout(function () { if (qs.get('flow')) SENAPI.flow(qs.get('flow')); if (qs.get('sel')) SENAPI.preselect(qs.get('sel')); }, 0); }
  });

  // Small API for tours / demo panel / deep links (prototype only).
  var SENAPI = window.SENAPI = {
    st: st,
    preselect: function (to) { st.aq = 'assign'; st.amode = 'select'; st.asel = { 'C-31001': true, 'C-31002': true, 'C-31003': true }; st.recip = to && to.charAt(0) === 'S' ? to : 'T2'; if (to && to.charAt(0) === 'S') st.exc = true; if (C.state.view !== 'dist') C.go('dist'); else C.render(); },
    flow: function (f) {
      if (C.state.drawer) C.closeDrawer();
      if (f === 'nomine') { st.mineOn = !st.mineOn; C.go('ov'); return C.toast(st.mineOn ? 'ماژول شخصی «تبدیل‌های من» نمایش داده می‌شود (مجوز دریافت فعال).' : 'مجوز دریافت تبدیل شخصی ندارید؛ «تبدیل‌های من» پنهان شد.', 'info'); }
      if (f === 'exception') { SENAPI.preselect('S44'); return C.openDrawer('assignReview', 'x'); }
      SENAPI.preselect(); st.flow = f;
      if (f === 'conflict') return C.openDrawer('assignReview', 'x');
      X.runAssign(f);
    },
    returnPick: function () { st.aq = 'return'; st.rf = 'all'; st.rsel = { 'C-30110': true, 'C-30111': true }; if (C.state.view !== 'dist') C.go('dist'); else C.render(); }
  };
})();
