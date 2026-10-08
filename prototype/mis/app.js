/* MIS — role layer, part 4: events, palette, boot. Everything runs on the shared CRM runtime (../shared/crm-core.js).
   Prototype only: no data is saved or sent; no handler here performs a real business action. */
(function () {
  'use strict';
  var X = window.MISX, C = X.C, M = X.M, h = X.h, st = X.st, V = X.V, D = X.D;
  var $ = h.$, esc = h.esc, fa = h.fa, ic = h.ic;
  var keys = X.keys;

  function drawerKeep() { return C.state.drawer ? C.state.drawer.keep : {}; }
  function refocus(sel) { var n = $(sel); if (n) { n.focus({ preventScroll: true }); try { var l = n.value.length; n.setSelectionRange(l, l); } catch (e) {} } }
  function keepFocus(t) {
    var sel = t.hasAttribute('data-id') ? '[data-sel="' + t.getAttribute('data-sel') + '"][data-id="' + t.getAttribute('data-id') + '"]' : t.hasAttribute('data-selall') ? '[data-selall="' + t.getAttribute('data-selall') + '"]' : t.hasAttribute('data-recip') ? '[data-recip="' + t.getAttribute('data-recip') + '"]' : t.hasAttribute('data-qp') ? '[data-qp="' + t.getAttribute('data-qp') + '"]' : '#' + t.id;
    var y = window.scrollY; C.render(); window.scrollTo(0, y); var n = $(sel); if (n) n.focus({ preventScroll: true });
  }

  function onClick(t, e) {
    var a = t.getAttribute('data-act');
    var tog = { 'data-iq': 'iq', 'data-sq': 'sq', 'data-cq': 'cq', 'data-rf': 'rf', 'data-rq': 'rq', 'data-dq': 'dq', 'data-run': 'run', 'data-icls': 'icls', 'data-tbs': 'tb', 'data-qb': 'qb' };
    for (var attr in tog) if (t.hasAttribute(attr)) { st[tog[attr]] = t.getAttribute(attr); if (attr === 'data-rq') st.rep = null; C.state.cursor = 0; if (C.state.drawer) C.closeDrawer(); C.render(); return true; }
    if (!a) return false;
    var arg = a.indexOf(':') > -1 ? a.slice(a.indexOf(':') + 1) : null, verb = arg != null ? a.slice(0, a.indexOf(':')) : a;
    e.preventDefault(); e.stopPropagation();
    if (t.disabled || t.getAttribute('aria-disabled') === 'true') return true;
    var keep = drawerKeep(), op, p, c;
    switch (verb) {
      case 'goto': C.closeMenu(); C.closeDrawer(); C.go(arg); return true;
      case 'goto-rec': C.closeDrawer(); st.rq = arg; st.rep = null; C.go('rec'); return true;
      case 'goto-cases': C.closeDrawer(); st.sq = arg; C.go('cases'); return true;
      case 'goto-diag': C.closeDrawer(); st.dq = arg; C.go('diag'); return true;
      case 'open-run': case 'open-file': case 'open-batch': case 'open-case': case 'open-tl': case 'open-issue': case 'open-plan': case 'open-maint': case 'open-log':
        C.closeMenu(); C.openDrawer(verb.slice(5), arg); return true;
      case 'open-ret': C.closeMenu(); C.openDrawer('ret', arg); return true;
      case 'open-importnew': C.openDrawer('importNew', 'x', { step: 0, file: 'A' }); return true;
      case 'open-metrics': C.openDrawer('metrics', 'x'); return true;
      case 'open-op': C.openDrawer('result', arg); return true;
      case 'imp-parse': keep.step = 1; C.rerenderDrawer(); return true;
      case 'imp-back': keep.step = 0; C.rerenderDrawer(); return true;
      case 'imp-commit': X.commitImport(keep); C.render(); C.rerenderDrawer(); C.toast('اجرای ورود ثبت شد (نمایشی)؛ نتیجه هر ردیف جدا آمده است.', 'info'); return true;
      case 'review-assign': C.openDrawer('assignReview', 'x'); return true;
      case 'commit-assign': if (!keep.confirm) return true; X.runAssign(st.flow); return true;
      case 'refresh-review': c = X.cs('884217'); if (c) { c.stage = 'assigned'; c.cust.src = 'مدیر مهسا رضوی (تخصیص هم‌زمان)'; } delete st.asel['884217']; st.flow = null; C.render(); C.openDrawer('assignReview', 'x'); C.toast('بررسی بازخوانی شد؛ ۱ ردیف از انتخاب حذف شد. گیرنده یا ردیف جایگزین خودکار تعیین نمی‌شود.', 'warning'); return true;
      case 'clear-asel': st.asel = {}; C.render(); return true;
      case 'clear-rsel': st.rsel = {}; C.render(); return true;
      case 'clear-isel': st.isel = {}; C.render(); return true;
      case 'retry-op': op = X.opOf(arg); X.retryFailed(op); C.render(); C.openDrawer('result', arg); C.toast('تکرار فقط برای موارد ناموفق معلوم و پس از بررسی تازه انجام شد (نمایشی).', 'success'); return true;
      case 'reconcile-op': op = X.opOf(arg); X.reconcileOp(op); C.render(); C.openDrawer('result', arg); C.toast('تطبیق انجام شد؛ نتیجه واقعی هر مورد مشخص شد (نمایشی).', 'info'); return true;
      case 'reconcile-run': X.reconcileRun(X.run(arg)); C.render(); C.openDrawer('run', arg); C.toast('نتیجه نامعلوم با لاگ تطبیق شد (نمایشی).', 'info'); return true;
      case 'retry-run': X.retryRun(X.run(arg)); C.render(); C.openDrawer('run', arg); C.toast('فقط ردیف‌های ناموفق معلوم دوباره امتحان شدند (نمایشی).', 'success'); return true;
      case 'refresh-ret': c = X.cs(arg); c.ret = 'unknown'; c.retWhy = 'پس از بازخوانی: مسئول فعلی سرپرست است؛ مدرک انتقال برای این مسیر کامل نیست'; C.render(); C.openDrawer('ret', arg); C.toast('ردیف بازخوانی شد؛ وضعیت فعلی نمایش داده می‌شود و نیازمند تطبیق است.', 'info'); return true;
      case 'review-return': C.openDrawer('returnReview', 'x'); return true;
      case 'review-recon': C.openDrawer('reconReview', 'x'); return true;
      case 'run-recon': X.runRecon(); C.toast('تطبیق خواندنی اجرا شد؛ هیچ داده‌ای تغییر نکرد (نمایشی).', 'info'); return true;
      case 'open-report': st.rq = 'reports'; st.rep = arg; st.tb = X.report(arg).basis === 'snap' ? 'snap' : 'range'; C.render(); return true;
      case 'close-report': st.rep = null; C.render(); return true;
      case 'open-report-from-issue': C.closeDrawer(); st.rq = 'reports'; st.rep = arg; st.tb = X.report(arg).basis === 'snap' ? 'snap' : 'range'; C.go('rec'); return true;
      case 'plan-save': p = X.plan(arg); p.stage = 'saved'; C.render(); C.openDrawer('plan', arg); C.toast('برنامه ذخیره شد (نمایشی)؛ چیزی تحویل نشد.', 'info'); return true;
      case 'plan-approve': C.openDrawer('planApprove', arg); return true;
      case 'commit-approve': p = X.plan(arg); p.stage = 'approved'; p.approvedBy = M.user.name; p.approvedAt = 'همین الان'; p.reason = keep.reason; C.render(); C.openDrawer('plan', arg); C.toast('برنامه تأیید شد (نمایشی). هیچ لیدی ساخته نشد؛ ایجاد لید گام جدا است.', 'info'); return true;
      case 'plan-mat': C.openDrawer('planMat', arg); return true;
      case 'commit-mat': p = X.plan(arg); op = X.materialize(p); C.closeDrawer(); C.render(); C.openDrawer('result', op.id); return true;
      case 'open-plan-result': op = X.planResult(X.plan(arg)); C.openDrawer('result', op.id); return true;
      case 'review-quick': C.openDrawer('quickReview', 'x'); return true;
      case 'commit-quick': X.runQuick(); return true;
      case 'commit-maint': X.runMaint(); return true;
      case 'x': return true;
    }
    return false;
  }
  document.addEventListener('change', function (e) {
    var t = e.target;
    if (t.matches('[data-sel]')) { st[t.getAttribute('data-sel')][t.getAttribute('data-id')] = t.checked; keepFocus(t); return; }
    if (t.matches('[data-selall]')) { X.cand().forEach(function (c) { st.asel[c.id] = t.checked; }); keepFocus(t); return; }
    if (t.matches('[data-recip]')) { st.recip = t.getAttribute('data-recip'); keepFocus(t); return; }
    if (t.matches('[data-qp]')) { st.qp = t.getAttribute('data-qp'); keepFocus(t); return; }
    if (t.matches('[data-confirm]')) { drawerKeep().confirm = t.checked; C.rerenderDrawer(); var c = $('#drawer [data-confirm]'); if (c) c.focus(); return; }
    if (t.matches('[data-impfile]')) { drawerKeep().file = t.getAttribute('data-impfile'); C.rerenderDrawer(); var f = $('#drawer [data-impfile="' + t.getAttribute('data-impfile') + '"]'); if (f) f.focus(); return; }
    if (t.matches('[data-impok]')) { drawerKeep().ok = t.checked; C.rerenderDrawer(); var o = $('#drawer [data-impok]'); if (o) o.focus(); return; }
    if (t.matches('[data-mtack]')) { drawerKeep().ack = t.checked; C.rerenderDrawer(); var m = $('#drawer [data-mtack]'); if (m) m.focus(); return; }
    if (t.matches('[data-qn],[data-qr]')) { C.render(); }
  });
  document.addEventListener('input', function (e) {
    var t = e.target, k = drawerKeep();
    if (t.matches('[data-qn]')) { st.qn = Math.max(0, Math.min(999, Number(t.value) || 0)); return; }
    if (t.matches('[data-qr]')) { st.qr = t.value; return; }
    var map = { 'data-apreason': 'reason', 'data-applyword': 'apply', 'data-mtreason': 'reason', 'data-mtword': 'word' };
    for (var a in map) if (t.hasAttribute(a)) { k[map[a]] = t.value; var id = t.id; C.rerenderDrawer(); refocus('#' + id); return; }
  });

  /* ---------- Palette (phone appears only as «discovery», never as identity) ---------- */
  function palette(nq, match) {
    var out = [];
    if (!nq) return [
      { g: 'اقدام سریع', icon: 'inbox', label: 'امروز — صف کار', run: function () { C.go('today'); } }, { g: 'اقدام سریع', icon: 'upload', label: 'ورود و کیفیت داده', run: function () { C.go('import'); } },
      { g: 'اقدام سریع', icon: 'link', label: 'ردیف‌های منبع و اتصال', run: function () { C.go('cases'); } }, { g: 'اقدام سریع', icon: 'send', label: 'مسئولیت و تحویل', run: function () { C.go('custody'); } },
      { g: 'اقدام سریع', icon: 'swap', label: 'مسائل تطبیق', run: function () { st.rq = 'issues'; C.go('rec'); } }, { g: 'اقدام سریع', icon: 'chart', label: 'گزارش‌ها و اعتماد', run: function () { st.rq = 'reports'; st.rep = null; C.go('rec'); } },
      { g: 'اقدام سریع', icon: 'lock', label: 'نگهداری و پیشرفته', run: function () { C.go('maint'); } }, { g: 'اقدام سریع', icon: 'info', label: 'تعریف شاخص‌ها', run: function () { C.openDrawer('metrics', 'x'); } }
    ];
    M.cases.forEach(function (c) { if (match(c.id) || match(c.batch)) out.push({ g: 'ردیف‌های منبع', icon: 'link', label: 'ردیف MIS ' + fa(c.id), meta: c.batch, run: function () { C.go('cases'); C.openDrawer('case', c.id); } }); });
    if (/^[0-9۰-۹]{3,}$/.test(nq)) M.cases.forEach(function (c) { if (match(c.phone)) out.push({ g: 'کشف با شماره (هویت نیست)', icon: 'phone', label: c.phone + ' ← ردیف ' + fa(c.id), meta: 'فقط کشف', run: function () { C.go('cases'); C.openDrawer('case', c.id); } }); });
    M.runs.forEach(function (r) { if (match(r.id) || match(r.batch)) out.push({ g: 'اجراها', icon: 'activity', label: r.id + ' · ' + X.RUNTYPE[r.type], meta: r.batch, run: function () { C.go('import'); C.openDrawer('run', r.id); } }); });
    M.files.forEach(function (f) { if (match(f.name) || match(f.id)) out.push({ g: 'فایل‌ها', icon: 'file', label: f.name, meta: f.id, run: function () { st.iq = 'files'; C.go('import'); C.openDrawer('file', f.id); } }); });
    M.issues.forEach(function (i) { if (match(i.title) || match(i.id)) out.push({ g: 'مسائل تطبیق', icon: 'swap', label: i.title, meta: i.id, run: function () { st.rq = 'issues'; C.go('rec'); C.openDrawer('issue', i.id); } }); });
    M.plans.forEach(function (p) { if (match(p.name) || match(p.id)) out.push({ g: 'برنامه‌ها', icon: 'layers', label: p.name, meta: p.id, run: function () { C.go('plan'); C.openDrawer('plan', p.id); } }); });
    return out;
  }

  /* ---------- Boot ---------- */
  C.boot({
    home: 'today', titleSuffix: 'پنل MIS (نمونه)', palettePlaceholder: 'شناسه ردیف، دسته، اجرا، فایل، مسئله یا یک فرمان… (شماره موبایل فقط برای کشف)',
    nav: function () {
      var W = X.work().length, R = X.importRuns().filter(function (r) { return r.state === 'partial' || r.state === 'unknown'; }).length;
      return [
        { id: 'today', label: 'امروز', icon: 'inbox', group: 'pri', mobile: 'امروز', count: function () { return W; }, alert: true },
        { id: 'import', label: 'ورود و کیفیت داده', icon: 'upload', group: 'pri', mobile: 'ورود', count: function () { return R; }, alert: true },
        { id: 'cases', label: 'ردیف‌های منبع', icon: 'link', group: 'pri', mobile: 'ردیف‌ها' },
        { id: 'custody', label: 'مسئولیت و تحویل', icon: 'send', group: 'pri', mobile: 'تحویل' },
        { id: 'rec', label: 'گزارش‌ها و تطبیق', icon: 'swap', group: 'pri', mobile: 'گزارش', count: function () { return M.issues.length; }, alert: true },
        { id: 'plan', label: 'برنامه‌ریزی (مشروط)', icon: 'layers', group: 'cond' },
        { id: 'quick', label: 'تحویل سریع (مشروط)', icon: 'alert', group: 'cond' },
        { id: 'maint', label: 'نگهداری', icon: 'lock', group: 'adv' },
        { id: 'diag', label: 'عیب‌یابی', icon: 'shield', group: 'adv' }
      ];
    },
    views: V, drawers: D, wideDrawers: ['case', 'tl', 'assignReview', 'result', 'ret', 'issue', 'plan', 'maint', 'importNew', 'run', 'quickReview', 'planMat', 'metrics', 'reconReview', 'returnReview'],
    pageTours: { today: 'mis', cases: 'lineage', custody: 'ret', maint: 'boundary' },
    workload: function () { return '<span>' + ic('inbox') + '</span><span><b>' + fa(X.work().length) + '</b> مورد در صف کار</span><span class="sep extra"></span><span class="due extra">' + ic('activity') + ' <b style="color:inherit">' + fa(X.importRuns().filter(function (r) { return r.state === 'partial' || r.state === 'unknown'; }).length) + '</b> ورود بدون نتیجه قطعی</span>'; },
    userMenuTop: function () { return [{ label: 'تعریف شاخص‌ها', icon: 'chart', act: 'open-metrics' }, { label: 'عیب‌یابی', icon: 'shield', go: 'diag' }]; },
    empty: {
      today: ['کاری در صف نیست', 'این فقط برای منابع و محدوده دریافت‌شده است و «همه چیز درست است» را اثبات نمی‌کند.', ''],
      import: ['اجرایی ثبت نشده', 'نبود اجرا با «ورود ناموفق» یکی نیست.', '<button type="button" class="btn btn-primary" data-act="open-importnew">ورود فایل جدید</button>'],
      cases: ['ردیفی در این صف نیست', 'صف دیگری را انتخاب کنید.', '<button type="button" class="btn btn-soft" data-sq="all">همه ردیف‌ها</button>'],
      custody: ['ردیفی برای این گام نیست', 'خالی بودن لیست ثبت یا عدم ثبت تحویل را اثبات نمی‌کند.', ''],
      rec: ['مسئله‌ای ثبت نشده', 'این نتیجه فقط برای منابع دریافت‌شده است و تطبیق کامل را اثبات نمی‌کند.', ''],
      plan: ['برنامه‌ای نیست', '', ''], quick: ['موردی نیست', '', ''], maint: ['اقدام نگهداری ثبت نشده', 'مجاز نبودن با خالی بودن فرق دارد.', ''], diag: ['موردی برای عیب‌یابی نیست', '', '']
    },
    onClick: onClick, palette: palette,
    init: function (state, qs) {
      ['iq', 'sq', 'cq', 'rf', 'rq', 'icls', 'tb', 'dq', 'run', 'qb', 'qp', 'maintAuth'].forEach(function (k) { if (qs.get(k)) st[k] = qs.get(k); });
      if (qs.get('rep')) { st.rep = qs.get('rep'); st.rq = 'reports'; if (!qs.get('tb') && X.report(st.rep)) st.tb = X.report(st.rep).basis === 'snap' ? 'snap' : 'range'; }
      if (qs.get('plan')) st.plan = qs.get('plan');
    },
    afterBoot: function (qs) { setTimeout(function () { if (qs.get('flow')) MISAPI.flow(qs.get('flow')); if (qs.get('sel')) MISAPI.preselect(); }, 0); }
  });

  // Small API for tours / demo panel / deep links (prototype only).
  var MISAPI = window.MISAPI = {
    st: st,
    preselect: function () { st.cq = 'assign'; st.recip = 'MG1'; st.asel = {}; ['884214', '884215', '884216', '884217', '884218', '884219'].forEach(function (id) { st.asel[id] = true; }); if (C.state.view !== 'custody') C.go('custody'); else C.render(); },
    flow: function (f) {
      if (C.state.drawer) C.closeDrawer();
      if (f === 'conflict') { MISAPI.preselect(); st.flow = 'conflict'; return C.openDrawer('assignReview', 'x', { confirm: false }); }
      if (f === 'partial' || f === 'retry' || f === 'unknown') { MISAPI.preselect(); return X.runAssign(f); }
      if (f === 'importnew') return C.openDrawer('importNew', 'x', { step: 0, file: 'A' });
      if (f === 'importpartial') { C.go('import'); return C.openDrawer('run', 'R-9921'); }
      if (f === 'importunknown') { C.go('import'); return C.openDrawer('run', 'R-9908'); }
      if (f === 'blocked') { st.cq = 'return'; st.rf = 'all'; C.go('custody'); return C.openDrawer('ret', '884204'); }
      if (f === 'retconflict') { st.cq = 'return'; C.go('custody'); return C.openDrawer('ret', '884212'); }
      if (f === 'retunknown') { st.cq = 'return'; C.go('custody'); return C.openDrawer('ret', '884205'); }
      if (f === 'lineage') { C.go('cases'); return C.openDrawer('case', '884204'); }
      if (f === 'planapprove') { C.go('plan'); return C.openDrawer('plan', 'P-30'); }
      if (f === 'planmat') { C.go('plan'); return C.openDrawer('planMat', 'P-30'); }
      if (f === 'maintblocked') { C.go('maint'); return C.openDrawer('maint', 'm2'); }
      if (f === 'maintauth') { st.maintAuth = 'verified'; C.go('maint'); return C.openDrawer('maint', 'm3'); }
      if (f === 'quick') { st.qb = 'B-1186'; st.qp = 'seller'; st.qn = 50; st.qr = 'تحویل فوری درخواست مدیر'; C.go('quick'); return C.openDrawer('quickReview', 'x'); }
      if (f === 'issue') { st.rq = 'issues'; C.go('rec'); return C.openDrawer('issue', 'I1'); }
      if (f === 'recon') { st.rq = 'issues'; st.isel = { I2: true, I5: true, I8: true, I11: true }; C.go('rec'); return C.openDrawer('reconReview', 'x'); }
    }
  };
})();
