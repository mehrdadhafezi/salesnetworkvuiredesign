/* Sales Manager — role layer, part 4: events, palette, boot.
   Everything runs on the shared CRM runtime (../shared/crm-core.js). Prototype only: no data is saved or sent. */
(function () {
  'use strict';
  var X = window.MGRX, C = X.C, M = X.M, h = X.h, st = X.st, V = X.V, D = X.D;
  var $ = h.$, esc = h.esc, fa = h.fa, ic = h.ic;
  var seller = X.seller, team = X.team, keys = X.keys;

  function goAttn(a) {
    if (a.aq) st.aq = a.aq; if (a.rf) st.rf = a.rf; if (a.iq) st.iq = a.iq; if (a.pm) st.pm = a.pm;
    C.go(a.go);
  }
  function onClick(t, e) {
    var a = t.getAttribute('data-act');
    var tog = { 'data-aq': 'aq', 'data-iq': 'iq', 'data-rf': 'rf', 'data-amode': 'amode', 'data-rbasis': 'rbasis', 'data-pm': 'pm', 'data-pb': 'pb', 'data-tbs': 'tb', 'data-hq': 'hq', 'data-xq': 'xq', 'data-ar': 'ar', 'data-cs': 'custSt' };
    for (var attr in tog) if (t.hasAttribute(attr)) { st[tog[attr]] = t.getAttribute(attr); C.state.cursor = 0; if (C.state.drawer) C.closeDrawer(); C.render(); return true; }
    if (t.matches('[data-hrtype]')) { st.hrForm.type = t.getAttribute('data-hrtype'); st.hrErr = null; C.render(); return true; }
    if (t.matches('[data-kpi-filter]')) { var kf = t.getAttribute('data-kpi-filter').split(':'); if (kf[0] === 'iq') { st.iq = kf[1]; C.render(); } else if (kf[0] === 'perf') { st.pm = kf[1]; C.go('perf'); } else if (kf[0] === 'dist') { st.aq = kf[1]; C.go('dist'); } else if (kf[0] === 'xnum') { st.xq = kf[1]; C.go('xnum'); } return true; }
    if (t.matches('[data-attn]')) { C.openDrawer('attn', t.getAttribute('data-attn')); return true; }
    if (!a) return false;
    var arg = a.indexOf(':') > -1 ? a.slice(a.indexOf(':') + 1) : null, verb = arg != null ? a.slice(0, a.indexOf(':')) : a;
    e.preventDefault(); e.stopPropagation();
    switch (verb) {
      case 'goto': C.closeMenu(); C.go(arg); return true;
      case 'goto-cust': C.closeDrawer(); C.go('cust'); return true;
      case 'attn-go': var A = X.attention().filter(function (x) { return x.id === arg; })[0]; if (A) { C.closeDrawer(); goAttn(A); } return true;
      case 'open-unit': C.closeMenu(); C.openDrawer('unit', arg); return true;
      case 'open-team': C.closeMenu(); C.openDrawer('team', arg); return true;
      case 'open-seller': C.closeMenu(); C.openDrawer('seller', arg); return true;
      case 'drill': C.closeMenu(); C.closeDrawer(); st.pscope = arg; st.pm = String(arg).charAt(0) === 'T' ? 'seller' : 'sup'; C.go('perf'); return true;
      case 'pscope': st.pscope = arg === 'all' ? null : arg; C.render(); return true;
      case 'open-metrics': C.openDrawer('metrics', 'x'); return true;
      case 'assign-to': C.closeMenu(); st.aq = 'assign'; st.amode = 'select'; st.recip = arg; st.skipOk = false; C.go('dist'); C.toast('گیرنده انتخاب شد: ' + X.recip(arg).name + '. پرونده‌ها را انتخاب کنید.', 'info'); return true;
      case 'hr-for': C.closeMenu(); st.hrForm = { seller: arg, type: 'transfer', dest: '', reason: '' }; st.hrErr = null; C.go('hr'); return true;
      case 'clear-asel': st.asel = {}; C.render(); return true;
      case 'clear-rsel': st.rsel = {}; C.render(); return true;
      case 'pick-one': st.asel[arg] = true; C.closeDrawer(); C.render(); return true;
      case 'review-assign': if (t.disabled) return true; C.openDrawer('assignReview', 'x'); return true;
      case 'refresh-review': delete st.asel['C-41003']; M.pool = M.pool.filter(function (c) { return c.id !== 'C-41003'; }); st.flow = null; C.render(); C.openDrawer('assignReview', 'x'); C.toast('بررسی بازخوانی شد؛ ۱ پرونده از انتخاب حذف شد.', 'warning'); return true;
      case 'commit-assign': if (t.disabled) return true; X.runAssign(st.flow); return true;
      case 'open-op': C.openDrawer('result', arg); return true;
      case 'retry': X.retryFailed(arg); C.render(); C.openDrawer('result', arg); C.toast('تلاش مجدد فقط برای موارد ناموفق انجام شد (نمایشی).', 'success'); return true;
      case 'reconcile': X.reconcile(arg); C.render(); C.openDrawer('result', arg); C.toast('تطبیق انجام شد؛ نتیجه واقعی هر مورد مشخص شد (نمایشی).', 'info'); return true;
      case 'open-ret': C.openDrawer('ret', arg); return true;
      case 'refresh-ret': var rc = M.held.filter(function (x) { return x.id === arg; })[0]; rc.elig = 'ok'; rc.why = null; C.render(); C.openDrawer('ret', arg); C.toast('پرونده بازخوانی شد؛ وضعیت فعلی نمایش داده می‌شود.', 'info'); return true;
      case 'review-return': C.openDrawer('returnReview', 'x'); return true;
      case 'open-inv': C.closeMenu(); C.openDrawer('inv', arg); return true;
      case 'copy-link': C.toast('لینک پرداخت فاکتور ' + fa(arg) + ' کپی شد (نمایشی). کپی تغییر مالی نیست.', 'success'); return true;
      case 'resend': if (t.disabled) return true; st.resent[arg] = true; C.rerenderDrawer(); C.toast('درخواست ارسال ثبت شد (نمایشی). نتیجه تحویل تأیید نشده است.', 'info'); return true;
      case 'open-arch': C.openDrawer('arch', arg); return true;
      case 'open-xreq': C.openDrawer('xreq', arg); return true;
      case 'x-approve':
        if (t.disabled) return true;
        if (st.xFlow === 'fail') { st.xFlow = null; st.xErr[arg] = 'ساخت سرنخ قدیمی انجام نشد و تغییرات برگردانده شد.'; C.rerenderDrawer(); C.toast('ثبت انجام نشد؛ درخواست همچنان در انتظار است.', 'error'); return true; }
        if (st.xFlow === 'unknown') { st.xFlow = null; st.xDone[arg] = 'unknown'; delete st.xErr[arg]; C.closeDrawer(); C.render(); C.openDrawer('xreq', arg); C.toast('پاسخ تأیید نرسید؛ نتیجه نامعلوم است. پیش از تکرار تطبیق کنید.', 'warning'); return true; }
        st.xDone[arg] = 'approved'; delete st.xErr[arg]; C.closeDrawer(); C.render(); C.toast('درخواست تأیید شد و یک سرنخ قدیمی برای فروشنده ساخته شد (نمایشی). نگاشت به CaseRef نامشخص می‌ماند.', 'success'); return true;
      case 'x-reject': var xn = $('[data-xnote]'), xv = xn ? xn.value.trim() : ''; var xr = M.xreq.filter(function (x) { return x.id === arg; })[0]; xr.reason = xv; xr.by = 'شما · همین الان'; st.xDone[arg] = 'rejected'; C.closeDrawer(); C.render(); C.toast('درخواست رد شد (نمایشی).' + (xv ? '' : ' دلیلی ثبت نشد.'), 'info'); return true;
      case 'x-reconcile': delete st.xDone[arg]; delete st.xErr[arg]; C.rerenderDrawer(); C.render(); C.toast('تطبیق: تصمیم ثبت نشده بود؛ اکنون دوباره قابل بررسی است.', 'info'); return true;
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
      case 'retry-unit': C.setSim(null); C.toast('داده سرپرست ارشد ' + X.unit('U2').name + ' دوباره دریافت شد (نمایشی).', 'success'); return true;
      case 'cust-retry': st.custSt = 'loading'; C.render(); setTimeout(function () { st.custSt = 'loaded'; C.render(); C.toast('تاریخچه دوباره خوانده شد (نمایشی).', 'success'); }, 700); return true;
    }
    return false;
  }
  document.addEventListener('change', function (e) {
    var t = e.target;
    if (t.matches('[data-sel]')) { st[t.getAttribute('data-sel')][t.getAttribute('data-id')] = t.checked; keepFocus(t); return; }
    if (t.matches('[data-selall]')) { var k = t.getAttribute('data-selall'), ids = k === 'asel' ? X.poolOk().map(function (c) { return c.id; }) : M.held.filter(function (c) { return c.elig === 'ok'; }).map(function (c) { return c.id; }); ids.forEach(function (id) { st[k][id] = t.checked; }); keepFocus(t); return; }
    if (t.matches('[data-recip]')) { var id = t.getAttribute('data-recip'); if (st.amode === 'select') { st.recip = id; if (X.isSkip(id)) st.skipOpen = true; } else st.recips[id] = t.checked; keepFocus(t); return; }
    if (t.matches('[data-confirm]')) { C.state.drawer.keep.confirm = t.checked; C.rerenderDrawer(); var c = $('#drawer [data-confirm]'); if (c) c.focus(); return; }
    if (t.matches('[data-hr]')) { st.hrForm[t.getAttribute('data-hr')] = t.value; if (st.hrErr) delete st.hrErr[t.getAttribute('data-hr')]; if (t.getAttribute('data-hr') === 'dest') C.render(); return; }
    if (t.matches('[data-per]')) { st.per = Math.max(1, Math.min(50, Number(t.value) || 1)); keepFocus(t); return; }
    if (t.matches('[data-rteam]')) { st.rteam = t.value; C.render(); }
  });
  document.addEventListener('input', function (e) { var t = e.target; if (t.matches('[data-hr="reason"]')) st.hrForm.reason = t.value; if (t.matches('[data-skipreason]')) st.skipReason = t.value; });
  document.addEventListener('toggle', function (e) { if (e.target.matches && e.target.matches('details[data-exc]')) st.skipOpen = e.target.open; }, true);
  function keepFocus(t) {
    var sel = t.hasAttribute('data-id') ? '[data-sel="' + t.getAttribute('data-sel') + '"][data-id="' + t.getAttribute('data-id') + '"]' : t.hasAttribute('data-selall') ? '[data-selall="' + t.getAttribute('data-selall') + '"]' : t.hasAttribute('data-recip') ? '[data-recip="' + t.getAttribute('data-recip') + '"]' : '[data-per]';
    var y = window.scrollY; C.render(); window.scrollTo(0, y); var n = $(sel); if (n) n.focus({ preventScroll: true });
  }

  /* ---------- Palette ---------- */
  function palette(nq, match) {
    var out = [];
    if (!nq) return [
      { g: 'اقدام سریع', icon: 'inbox', label: 'عملیات و استثناها', run: function () { C.go('ov'); } }, { g: 'اقدام سریع', icon: 'send', label: 'تخصیص پرونده', run: function () { st.aq = 'assign'; C.go('dist'); } },
      { g: 'اقدام سریع', icon: 'compass', label: 'عملکرد و تطبیق', run: function () { C.go('perf'); } }, { g: 'اقدام سریع', icon: 'inbox', label: 'کاوشگر آرشیو', run: function () { C.go('arch'); } },
      { g: 'اقدام سریع', icon: 'plus', label: 'درخواست‌های شماره اضافه', run: function () { st.xq = 'pending'; C.go('xnum'); } }, { g: 'اقدام سریع', icon: 'briefcase', label: 'درخواست‌های HR منتظر بررسی', run: function () { st.hq = 'review'; C.go('hr'); } },
      { g: 'اقدام سریع', icon: 'chart', label: 'تعریف شاخص‌ها', run: function () { C.openDrawer('metrics', 'x'); } }
    ];
    M.units.forEach(function (u) { if (match(u.name)) out.push({ g: 'واحدها', icon: 'users', label: 'سرپرست ارشد ' + u.name, meta: u.active ? 'مستقیم' : 'غیرفعال', run: function () { C.go('ov'); C.openDrawer('unit', u.id); } }); });
    M.teams.forEach(function (t) { if (match(t.sup)) out.push({ g: 'سرپرستان', icon: 'briefcase', label: t.sup, meta: t.parent === 'M' ? 'مستقیم' : 'غیرمستقیم', run: function () { C.go('ov'); C.openDrawer('team', t.id); } }); });
    M.sellers.forEach(function (s) { if (match(s.name)) out.push({ g: 'فروشندگان', icon: 'user', label: s.name, meta: s.team === 'direct' ? 'مستقیم' : 'غیرمستقیم · ' + X.teamName(s.team), run: function () { C.go('perf'); C.openDrawer('seller', s.id); } }); });
    M.invoices.forEach(function (i) { if (match(i.code) || match(i.customer)) out.push({ g: 'فاکتورها', icon: 'receipt', label: 'فاکتور ' + fa(i.code), meta: i.customer, run: function () { st.iq = 'all'; C.go('inv'); C.openDrawer('inv', i.code); } }); });
    M.held.forEach(function (c) { if (match(c.id)) out.push({ g: 'پرونده‌ها', icon: 'file', label: c.id, meta: 'نزد ' + X.holderName(c), run: function () { st.aq = 'return'; st.rf = 'all'; C.go('dist'); C.openDrawer('ret', c.id); } }); });
    M.archives.forEach(function (a) { if (match(a.caseRef) || match(a.id)) out.push({ g: 'آرشیو', icon: 'inbox', label: a.caseRef, meta: M.archReasons[a.reason].label, run: function () { st.ar = 'all'; C.go('arch'); C.openDrawer('arch', a.id); } }); });
    M.xreq.forEach(function (r) { if (match(r.id)) out.push({ g: 'شماره اضافه', icon: 'plus', label: r.id, meta: X.XRS[r.state].label, run: function () { st.xq = r.state === 'pending' ? 'pending' : 'decided'; C.go('xnum'); C.openDrawer('xreq', r.id); } }); });
    return out;
  }

  /* ---------- Boot ---------- */
  C.boot({
    home: 'ov', titleSuffix: 'پنل مدیر فروش (نمونه)', palettePlaceholder: 'نام سرپرست ارشد، سرپرست یا فروشنده، شماره پرونده، کد فاکتور یا یک فرمان…',
    nav: function () {
      var A = X.attention().length;
      return [
        { id: 'ov', label: 'عملیات و استثناها', icon: 'inbox', group: 'work', mobile: 'عملیات', count: function () { return A; }, alert: true },
        { id: 'dist', label: 'توزیع', icon: 'swap', group: 'work', mobile: 'توزیع', count: function () { return X.poolOk().length; } },
        { id: 'perf', label: 'عملکرد و تطبیق', icon: 'compass', group: 'work', mobile: 'عملکرد' },
        { id: 'inv', label: 'فاکتورها و استثناها', icon: 'receipt', group: 'work', mobile: 'فاکتورها', count: function () { return M.invoices.filter(X.IQ[0].f).length; }, alert: true },
        { id: 'arch', label: 'کاوشگر آرشیو', icon: 'layers', group: 'work', count: function () { return M.archives.length; } },
        { id: 'xnum', label: 'شماره اضافه', icon: 'plus', group: 'req', count: function () { return M.xreq.filter(function (r) { return r.state === 'pending' && !st.xDone[r.id]; }).length; }, alert: true },
        { id: 'hr', label: 'درخواست‌های HR', icon: 'briefcase', group: 'req', count: function () { return M.hr.filter(function (r) { return r.state === 'pending_review' && r.reviewer === 'self' && !st.hrDone[r.id]; }).length; }, alert: true },
        { id: 'rep', label: 'گزارش‌ها', icon: 'chart', group: 'req' },
        { id: 'cust', label: 'زمینه رفتار مشتری', icon: 'activity', group: 'ctx' },
        { id: 'wallet', label: 'کیف پول من', icon: 'wallet', group: 'me' }
      ];
    },
    views: V, drawers: D, wideDrawers: ['inv', 'result', 'assignReview', 'metrics', 'arch', 'xreq'],
    pageTours: { ov: 'manager', perf: 'perf', dist: 'dist' },
    workload: function () { var A = X.attention(); return '<span>' + ic('users') + '</span><span><b>' + fa(M.units.filter(function (u) { return u.active; }).length) + '</b> سرپرست ارشد فعال</span><span class="sep extra"></span><span class="due extra">' + ic('inbox') + ' <b style="color:inherit">' + fa(A.length) + '</b> دسته استثنا</span>'; },
    userMenuTop: function () { return [{ label: 'کیف پول من', icon: 'wallet', go: 'wallet' }, { label: 'تعریف شاخص‌ها', icon: 'chart', act: 'open-metrics' }]; },
    empty: {
      ov: ['واحدی در محدوده شما ثبت نشده', 'هنوز زیرمجموعه‌ای زیر نظر شما تعریف نشده است.', ''], perf: ['داده‌ای در این محدوده نیست', 'محدوده یا حالت دیگری را انتخاب کنید.', '<button type="button" class="btn btn-soft" data-act="pscope:all">همه واحدها</button>'],
      dist: ['پرونده‌ای در پنل شما نیست', 'هنوز پرونده‌ای برای توزیع به شما تحویل نشده است.', ''], inv: ['فاکتوری در این صف نیست', 'صف دیگری را انتخاب کنید.', '<button type="button" class="btn btn-soft" data-iq="all">همه فاکتورها</button>'],
      arch: ['در این برش مورد آرشیوی نیست', 'این فقط برش نمونه را توصیف می‌کند، نه کل محدوده.', ''], xnum: ['درخواستی نیست', 'خالی بودن صف به معنی حذف قابلیت نیست.', ''],
      hr: ['درخواستی نیست', '', ''], cust: ['رویدادی نیست', '', ''], wallet: ['تراکنشی ثبت نشده است', '', ''], rep: ['گزارشی در دسترس نیست', '', '']
    },
    onClick: onClick, palette: palette,
    init: function (state, qs) {
      ['pm', 'pb', 'aq', 'iq', 'hq', 'xq', 'ar', 'rf', 'src', 'report', 'amode', 'tb'].forEach(function (k) { if (qs.get(k)) st[k] = qs.get(k); });
      if (qs.get('cs')) st.custSt = qs.get('cs'); if (qs.get('scope')) st.pscope = qs.get('scope');
    },
    afterBoot: function (qs) { setTimeout(function () { if (qs.get('flow')) MGRAPI.flow(qs.get('flow')); if (qs.get('sel')) MGRAPI.preselect(qs.get('sel')); }, 0); }
  });

  // Small API for tours / demo panel / deep links (prototype only).
  var MGRAPI = window.MGRAPI = {
    st: st,
    preselect: function (to) { st.aq = 'assign'; st.amode = 'select'; st.asel = { 'C-41001': true, 'C-41002': true, 'C-41003': true, 'C-41004': true, 'C-41005': true, 'C-41006': true }; st.recip = to && /^[UTS]/.test(to) ? to : 'U1'; if (X.isSkip(st.recip)) st.skipOpen = true; if (C.state.view !== 'dist') C.go('dist'); else C.render(); },
    flow: function (f) {
      if (C.state.drawer) C.closeDrawer();
      if (f === 'skip') { MGRAPI.preselect('T1'); return C.openDrawer('assignReview', 'x'); }
      if (f === 'skipseller') { MGRAPI.preselect('S6'); return C.openDrawer('assignReview', 'x'); }
      if (f === 'xfail' || f === 'xunknown') { st.xFlow = f === 'xfail' ? 'fail' : 'unknown'; st.xq = 'pending'; C.go('xnum'); return C.openDrawer('xreq', 'XR-1201'); }
      MGRAPI.preselect(); st.flow = f;
      if (f === 'conflict') return C.openDrawer('assignReview', 'x');
      X.runAssign(f);
    },
    returnPick: function () { st.aq = 'return'; st.rf = 'all'; st.rsel = { 'C-40110': true, 'C-40111': true }; if (C.state.view !== 'dist') C.go('dist'); else C.render(); }
  };
})();
