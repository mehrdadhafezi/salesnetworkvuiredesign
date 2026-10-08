/* Sales Deputy — role layer, part 4: events, palette, boot.
   Everything runs on the shared CRM runtime (../shared/crm-core.js). Prototype only: no data is saved or sent. */
(function () {
  'use strict';
  var X = window.DEPX, C = X.C, M = X.M, h = X.h, st = X.st, V = X.V, D = X.D;
  var $ = h.$, esc = h.esc, fa = h.fa, ic = h.ic;
  var seller = X.seller, keys = X.keys;

  function nextPm(key) { key = String(key); return key.charAt(0) === 'M' ? 'senior' : key.charAt(0) === 'U' ? 'sup' : 'seller'; }
  function goExc(e) {
    if (e.pm) st.pm = e.pm; if (e.scope) st.pscope = e.scope; if (e.aq) st.aq = e.aq; if (e.rf) st.rf = e.rf; if (e.iq) st.iq = e.iq; if (e.dq) st.dq = e.dq;
    C.go(e.go);
    if (e.open) { var p = e.open.split(':'); C.openDrawer(p[0], p.slice(1).join(':')); }
  }
  function onClick(t, e) {
    var a = t.getAttribute('data-act');
    var tog = { 'data-aq': 'aq', 'data-iq': 'iq', 'data-rf': 'rf', 'data-amode': 'amode', 'data-rbasis': 'rbasis', 'data-pm': 'pm', 'data-pb': 'pb', 'data-tbs': 'tb', 'data-hq': 'hq', 'data-ec': 'ec', 'data-em': 'em', 'data-dq': 'dq', 'data-mq': 'mq', 'data-monauth': 'monAuth' };
    for (var attr in tog) if (t.hasAttribute(attr)) { st[tog[attr]] = t.getAttribute(attr); C.state.cursor = 0; if (C.state.drawer) C.closeDrawer(); C.render(); return true; }
    if (t.matches('[data-kpi-filter]')) { var kf = t.getAttribute('data-kpi-filter').split(':'); if (kf[0] === 'iq') { st.iq = kf[1]; C.render(); } else if (kf[0] === 'exc') { st.ec = kf[1]; st.em = 'all'; C.go('exc'); } else if (kf[0] === 'diag') { st.dq = kf[1]; C.go('diag'); } return true; }
    if (t.matches('[data-attn]')) { C.openDrawer('attn', t.getAttribute('data-attn')); return true; }
    if (!a) return false;
    var arg = a.indexOf(':') > -1 ? a.slice(a.indexOf(':') + 1) : null, verb = arg != null ? a.slice(0, a.indexOf(':')) : a;
    e.preventDefault(); e.stopPropagation();
    switch (verb) {
      case 'goto': C.closeMenu(); C.closeDrawer(); C.go(arg); return true;
      case 'goto-exc': C.closeDrawer(); st.ec = arg; st.em = 'all'; C.go('exc'); return true;
      case 'goto-diag': C.closeDrawer(); st.dq = arg; C.go('diag'); return true;
      case 'open-mgr': C.closeMenu(); C.openDrawer('mgr', arg); return true;
      case 'open-senior': C.closeMenu(); C.openDrawer('senior', arg); return true;
      case 'open-team': C.closeMenu(); C.openDrawer('team', arg); return true;
      case 'open-seller': C.closeMenu(); C.openDrawer('seller', arg); return true;
      case 'open-exc': C.openDrawer('exc', arg); return true;
      case 'exc-go': var E = X.excOf(arg); if (E) { C.closeDrawer(); goExc(E); } return true;
      case 'drill': C.closeMenu(); C.closeDrawer(); st.pscope = arg; st.pm = nextPm(arg); C.go('perf'); return true;
      case 'pscope': st.pscope = arg === 'all' ? null : arg; C.render(); return true;
      case 'open-metrics': C.openDrawer('metrics', 'x'); return true;
      case 'assign-to': C.closeMenu(); C.closeDrawer(); st.aq = 'assign'; st.amode = 'select'; st.recip = arg; st.bypassOk = false; if (X.isBypass(arg)) st.bypassOpen = true; C.go('alloc'); C.toast('گیرنده انتخاب شد: ' + X.recip(arg).name + (X.isBypass(arg) ? ' (مسیر استثنایی · مشروط)' : '') + '. پرونده‌ها را انتخاب کنید.', 'info'); return true;
      case 'clear-asel': st.asel = {}; C.render(); return true;
      case 'clear-rsel': st.rsel = {}; C.render(); return true;
      case 'pick-one': st.asel[arg] = true; C.closeDrawer(); C.render(); return true;
      case 'review-assign': if (t.disabled) return true; C.openDrawer('assignReview', 'x'); return true;
      case 'refresh-review': delete st.asel['C-51003']; M.pool = M.pool.filter(function (c) { return c.id !== 'C-51003'; }); st.flow = null; C.render(); C.openDrawer('assignReview', 'x'); C.toast('بررسی بازخوانی شد؛ ۱ پرونده از انتخاب حذف شد.', 'warning'); return true;
      case 'refresh-recip': st.flow = null; st.recip = null; st.recips = {}; C.closeDrawer(); C.render(); C.toast('گیرنده بازخوانی شد؛ گیرنده‌ی واجد شرایط را دوباره انتخاب کنید. گیرنده‌ی جایگزین خودکار تعیین نمی‌شود.', 'warning'); return true;
      case 'commit-assign': if (t.disabled) return true; X.runAssign(st.flow); return true;
      case 'open-op': C.openDrawer('result', arg); return true;
      case 'retry': X.retryFailed(arg); C.render(); C.openDrawer('result', arg); C.toast('تلاش مجدد فقط برای موارد ناموفق انجام شد (نمایشی).', 'success'); return true;
      case 'reconcile': X.reconcile(arg); C.render(); C.openDrawer('result', arg); C.toast('تطبیق انجام شد؛ نتیجه واقعی هر مورد مشخص شد (نمایشی).', 'info'); return true;
      case 'open-ret': C.openDrawer('ret', arg); return true;
      case 'refresh-ret': var rc = M.held.filter(function (x) { return x.id === arg; })[0]; rc.elig = 'ok'; rc.why = null; C.render(); C.openDrawer('ret', arg); C.toast('پرونده بازخوانی شد؛ وضعیت فعلی نمایش داده می‌شود.', 'info'); return true;
      case 'review-return': C.openDrawer('returnReview', 'x'); return true;
      case 'open-inv': C.closeMenu(); C.openDrawer('inv', arg); return true;
      case 'open-hr': C.openDrawer('hr', arg); return true;
      case 'hr-step': st.hrDone[arg] = 'step'; C.closeDrawer(); C.render(); C.toast('مرحله شما تأیید شد و درخواست به مرحله بعد رفت (نمایشی). هنوز اعمال نشده است؛ اعمال نهایی با منابع انسانی است.', 'success'); return true;
      case 'hr-reject': var note = $('[data-hrnote]'), nv = note ? note.value.trim() : ''; if (!nv) { C.state.drawer.keep.err = true; C.state.drawer.keep.note = ''; C.rerenderDrawer(); var nn = $('[data-hrnote]'); if (nn) nn.focus(); return true; } st.hrDone[arg] = 'rejected'; C.closeDrawer(); C.render(); C.toast('درخواست با دلیل رد شد (نمایشی).', 'info'); return true;
      case 'open-report': st.report = arg; C.render(); return true;
      case 'close-report': st.report = null; C.render(); return true;
      case 'rscope': st.rscope = arg; C.render(); return true;
      case 'retry-branch': C.setSim(null); C.toast('داده شاخه دوباره دریافت شد (نمایشی).', 'success'); return true;
    }
    return false;
  }
  document.addEventListener('change', function (e) {
    var t = e.target;
    if (t.matches('[data-sel]')) { st[t.getAttribute('data-sel')][t.getAttribute('data-id')] = t.checked; keepFocus(t); return; }
    if (t.matches('[data-selall]')) { var k = t.getAttribute('data-selall'), ids = k === 'asel' ? X.poolOk().map(function (c) { return c.id; }) : M.held.filter(function (c) { return c.elig === 'ok'; }).map(function (c) { return c.id; }); ids.forEach(function (id) { st[k][id] = t.checked; }); keepFocus(t); return; }
    if (t.matches('[data-recip]')) { var id = t.getAttribute('data-recip'); if (st.amode === 'select') { st.recip = id; if (X.isBypass(id)) st.bypassOpen = true; } else st.recips[id] = t.checked; keepFocus(t); return; }
    if (t.matches('[data-confirm]')) { C.state.drawer.keep.confirm = t.checked; C.rerenderDrawer(); var c = $('#drawer [data-confirm]'); if (c) c.focus(); return; }
    if (t.matches('[data-per]')) { st.per = Math.max(1, Math.min(50, Number(t.value) || 1)); keepFocus(t); return; }
    if (t.matches('[data-rscope]')) { st.rscope = t.value; C.render(); }
  });
  document.addEventListener('input', function (e) { var t = e.target; if (t.matches('[data-bypassreason]')) st.bypassReason = t.value; });
  document.addEventListener('toggle', function (e) { if (e.target.matches && e.target.matches('details[data-exc]')) st.bypassOpen = e.target.open; }, true);
  function keepFocus(t) {
    var sel = t.hasAttribute('data-id') ? '[data-sel="' + t.getAttribute('data-sel') + '"][data-id="' + t.getAttribute('data-id') + '"]' : t.hasAttribute('data-selall') ? '[data-selall="' + t.getAttribute('data-selall') + '"]' : t.hasAttribute('data-recip') ? '[data-recip="' + t.getAttribute('data-recip') + '"]' : '[data-per]';
    var y = window.scrollY; C.render(); window.scrollTo(0, y); var n = $(sel); if (n) n.focus({ preventScroll: true });
  }

  /* ---------- Palette ---------- */
  function palette(nq, match) {
    var out = [];
    if (!nq) return [
      { g: 'اقدام سریع', icon: 'compass', label: 'نمای رهبری', run: function () { C.go('ov'); } }, { g: 'اقدام سریع', icon: 'users', label: 'سلسله‌مراتب و عملکرد', run: function () { C.go('perf'); } },
      { g: 'اقدام سریع', icon: 'inbox', label: 'استثناها و هماهنگی', run: function () { C.go('exc'); } }, { g: 'اقدام سریع', icon: 'chart', label: 'گزارش‌ها', run: function () { C.go('rep'); } },
      { g: 'اقدام سریع', icon: 'send', label: 'تخصیص مستقیم (مشروط)', run: function () { st.aq = 'assign'; C.go('alloc'); } }, { g: 'اقدام سریع', icon: 'briefcase', label: 'زمینه درخواست HR', run: function () { st.hq = 'review'; C.go('hr'); } },
      { g: 'اقدام سریع', icon: 'shield', label: 'عیب‌یابی شاخص و منشأ', run: function () { C.go('diag'); } }, { g: 'اقدام سریع', icon: 'info', label: 'تعریف شاخص‌ها', run: function () { C.openDrawer('metrics', 'x'); } }
    ];
    M.managers.forEach(function (m) { if (match(m.name)) out.push({ g: 'شاخه‌ها', icon: 'users', label: 'مدیر ' + m.name, meta: 'شاخه', run: function () { C.go('ov'); C.openDrawer('mgr', m.id); } }); });
    M.seniors.forEach(function (u) { if (match(u.name)) out.push({ g: 'سرپرستان ارشد', icon: 'compass', label: u.name, meta: X.mgrName(u.mgr), run: function () { C.go('perf'); C.openDrawer('senior', u.id); } }); });
    M.teams.forEach(function (t) { if (match(t.sup)) out.push({ g: 'سرپرستان', icon: 'briefcase', label: t.sup, meta: X.mgrName(X.mgrOfTeam(t.id)), run: function () { C.go('perf'); C.openDrawer('team', t.id); } }); });
    M.sellers.forEach(function (s) { if (match(s.name)) out.push({ g: 'فروشندگان', icon: 'user', label: s.name, meta: X.mgrName(X.mgrOfSeller(s)), run: function () { C.go('perf'); C.openDrawer('seller', String(s.id)); } }); });
    M.invoices.forEach(function (i) { if (match(i.code) || match(i.customer)) out.push({ g: 'فاکتورها', icon: 'receipt', label: 'فاکتور ' + fa(i.code), meta: i.customer, run: function () { st.iq = 'all'; C.go('inv'); C.openDrawer('inv', i.code); } }); });
    M.held.forEach(function (c) { if (match(c.id)) out.push({ g: 'پرونده‌ها', icon: 'file', label: c.id, meta: 'نزد ' + X.holderName(c), run: function () { st.aq = 'return'; st.rf = 'all'; C.go('alloc'); C.openDrawer('ret', c.id); } }); });
    M.exc.forEach(function (e) { if (match(e.subject)) out.push({ g: 'استثناها', icon: 'inbox', label: M.excClasses[e.cls].label, meta: e.state, run: function () { C.go('exc'); C.openDrawer('exc', e.id); } }); });
    return out;
  }

  /* ---------- Boot ---------- */
  C.boot({
    home: 'ov', titleSuffix: 'پنل معاون فروش (نمونه)', palettePlaceholder: 'نام مدیر، سرپرست ارشد، سرپرست یا فروشنده، شماره پرونده، کد فاکتور یا یک فرمان…',
    nav: function () {
      var A = X.exceptions().length;
      return [
        { id: 'ov', label: 'نمای رهبری', icon: 'compass', group: 'lead', mobile: 'رهبری' },
        { id: 'perf', label: 'سلسله‌مراتب و عملکرد', icon: 'users', group: 'lead', mobile: 'عملکرد' },
        { id: 'exc', label: 'استثناها', icon: 'inbox', group: 'lead', mobile: 'استثناها', count: function () { return A; }, alert: true },
        { id: 'rep', label: 'گزارش‌ها', icon: 'chart', group: 'lead', mobile: 'گزارش‌ها' },
        { id: 'alloc', label: 'تخصیص مستقیم (مشروط)', icon: 'send', group: 'cond' },
        { id: 'hr', label: 'زمینه HR (مشروط)', icon: 'briefcase', group: 'cond', count: function () { return M.hr.filter(function (r) { return r.state === 'pending_review' && r.reviewer === 'self' && !st.hrDone[r.id]; }).length; }, alert: true },
        { id: 'mon', label: 'پایش آرشیو و شماره اضافه (مشروط)', icon: 'layers', group: 'cond' },
        { id: 'inv', label: 'زمینه فاکتور', icon: 'receipt', group: 'ctx' },
        { id: 'diag', label: 'عیب‌یابی شاخص و منشأ', icon: 'shield', group: 'adv' },
        { id: 'wallet', label: 'کیف پول من', icon: 'wallet', group: 'me' }
      ];
    },
    views: V, drawers: D, wideDrawers: ['inv', 'result', 'assignReview', 'metrics', 'exc', 'attn', 'mgr'],
    pageTours: { ov: 'deputy', perf: 'perf', alloc: 'alloc' },
    workload: function () { var A = X.attention(); return '<span>' + ic('users') + '</span><span><b>' + fa(M.managers.length) + '</b> شاخه مدیر</span><span class="sep extra"></span><span class="due extra">' + ic('inbox') + ' <b style="color:inherit">' + fa(A.length) + '</b> دسته استثنا</span>'; },
    userMenuTop: function () { return [{ label: 'کیف پول من', icon: 'wallet', go: 'wallet' }, { label: 'تعریف شاخص‌ها', icon: 'chart', act: 'open-metrics' }]; },
    empty: {
      ov: ['شاخه‌ای در محدوده شما ثبت نشده', 'هنوز شاخه مدیری زیر نظر شما تعریف نشده است.', ''], perf: ['داده‌ای در این محدوده نیست', 'محدوده یا حالت دیگری را انتخاب کنید.', '<button type="button" class="btn btn-soft" data-act="pscope:all">همه شاخه‌ها</button>'],
      exc: ['استثنای باز نیست', 'این فقط برای محدوده و داده دریافت‌شده است.', ''], rep: ['گزارشی در دسترس نیست', '', ''],
      alloc: ['پرونده‌ای در موجودی شخصی شما نیست', 'صفر بودن موجودی ثبت دریافت را اثبات یا رد نمی‌کند.', ''], hr: ['درخواستی نیست', '', ''], mon: ['داده‌ای نیست', 'مجاز نبودن با خالی بودن فرق دارد.', ''],
      inv: ['فاکتوری در این صف نیست', 'صف دیگری را انتخاب کنید.', '<button type="button" class="btn btn-soft" data-iq="all">همه فاکتورها</button>'], diag: ['موردی برای عیب‌یابی نیست', '', ''], wallet: ['تراکنشی ثبت نشده است', '', '']
    },
    onClick: onClick, palette: palette,
    init: function (state, qs) {
      ['pm', 'pb', 'aq', 'iq', 'hq', 'ec', 'em', 'dq', 'mq', 'rf', 'report', 'amode', 'tb', 'monAuth', 'rscope'].forEach(function (k) { if (qs.get(k)) st[k] = qs.get(k); });
      if (qs.get('scope')) st.pscope = qs.get('scope');
    },
    afterBoot: function (qs) { setTimeout(function () { if (qs.get('flow')) DEPAPI.flow(qs.get('flow')); if (qs.get('sel')) DEPAPI.preselect(qs.get('sel')); }, 0); }
  });

  // Small API for tours / demo panel / deep links (prototype only).
  var DEPAPI = window.DEPAPI = {
    st: st,
    preselect: function (to) { st.aq = 'assign'; st.amode = 'select'; st.asel = { 'C-51001': true, 'C-51002': true, 'C-51003': true, 'C-51004': true, 'C-51005': true, 'C-51006': true }; st.recip = to && /^(MG|U|T|S)/.test(to) ? to : 'MG1'; if (X.isBypass(st.recip)) st.bypassOpen = true; if (C.state.view !== 'alloc') C.go('alloc'); else C.render(); },
    flow: function (f) {
      if (C.state.drawer) C.closeDrawer();
      if (f === 'bypass') { DEPAPI.preselect('U1'); return C.openDrawer('assignReview', 'x'); }
      if (f === 'bypassseller') { DEPAPI.preselect('S38'); return C.openDrawer('assignReview', 'x'); }
      if (f === 'recipchg') { DEPAPI.preselect('MG2'); st.flow = 'recipchg'; return C.openDrawer('assignReview', 'x'); }
      if (f === 'outscope') { DEPAPI.preselect('MG1'); st.flow = 'outscope'; return C.openDrawer('assignReview', 'x'); }
      DEPAPI.preselect(); st.flow = f;
      if (f === 'conflict') return C.openDrawer('assignReview', 'x');
      X.runAssign(f);
    },
    returnPick: function () { st.aq = 'return'; st.rf = 'all'; st.rsel = { 'C-50110': true, 'C-50111': true }; if (C.state.view !== 'alloc') C.go('alloc'); else C.render(); }
  };
})();
