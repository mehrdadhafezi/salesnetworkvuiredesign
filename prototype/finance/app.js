/* Finance — role layer, part 4: events, palette, boot. Runs on the shared CRM runtime (../shared/crm-core.js).
   Prototype only: no data is saved or sent; no handler performs a real financial action. FINANCE ROLE RUNTIME NOT LIVE VERIFIED. */
(function () {
  'use strict';
  var X = window.FINX, C = X.C, M = X.M, h = X.h, st = X.st, V = X.V, D = X.D;
  var $ = h.$, esc = h.esc, fa = h.fa, ic = h.ic;
  var ok3 = function (s) { return (s || '').trim().length > 2; };
  var keepOf = function () { return C.state.drawer ? C.state.drawer.keep : {}; };
  function refocus(sel) { var n = $(sel); if (n) { n.focus({ preventScroll: true }); try { var l = n.value.length; n.setSelectionRange(l, l); } catch (e) {} } }
  function keepFocus(t) {
    var sel = t.hasAttribute('data-id') ? '[data-sel="' + t.getAttribute('data-sel') + '"][data-id="' + t.getAttribute('data-id') + '"]' : '#' + t.id;
    var y = window.scrollY; C.render(); window.scrollTo(0, y); var n = $(sel); if (n) n.focus({ preventScroll: true });
  }
  function result(op) { C.closeDrawer(); C.render(); C.openDrawer('res', op.id); }

  function onClick(t, e) {
    var tog = { 'data-rq': 'rq', 'data-recq': 'recq', 'data-lq': 'lq', 'data-runq': 'runq', 'data-ldom': 'ldom' };
    for (var attr in tog) if (t.hasAttribute(attr)) { st[tog[attr]] = t.getAttribute(attr); C.state.cursor = 0; if (C.state.drawer) C.closeDrawer(); C.render(); return true; }
    var a = t.getAttribute('data-act'); if (!a) return false;
    var arg = a.indexOf(':') > -1 ? a.slice(a.indexOf(':') + 1) : null, verb = arg != null ? a.slice(0, a.indexOf(':')) : a;
    e.preventDefault(); e.stopPropagation();
    if (t.disabled || t.getAttribute('aria-disabled') === 'true') return true;
    var k = keepOf(), r, op, sp, run;
    switch (verb) {
      case 'goto': C.closeMenu(); C.closeDrawer(); C.go(arg); return true;
      case 'open-rs': case 'open-ref': case 'open-tx': case 'open-run': case 'open-iss': case 'open-rep': case 'open-audit': C.closeMenu(); C.openDrawer(verb.slice(5), arg); return true;
      case 'open-res': C.openDrawer('res', arg); return true;
      case 'open-bulkrev': C.openDrawer('bulkrev', 'x'); return true;
      case 'open-bulkapp': C.openDrawer('bulkapp', 'x'); return true;
      case 'open-diag': C.openDrawer('diag', 'x'); return true;
      case 'clear-sel': st.sel = {}; C.render(); return true;
      case 'clear-isel': st.isel = {}; C.render(); return true;
      case 'view-evidence': C.toast('مشاهدهٔ مدرک تابع اختیار و محافظت جدا است؛ در نمونه فایل واقعی وجود ندارد.', 'info'); return true;
      case 'reload-stage': r = X.q(arg); st.local[arg] = { review: r.stale.res, reloaded: true, rev: r.stale.by, prior: r.prior.concat([{ res: r.stale.res, who: r.stale.by, at: r.stale.at, note: 'تصمیم همزمان بازبین دیگر' }]), next: 'گام بعد طبق تصمیم بازبین دیگر' }; C.render(); C.openDrawer('rs', arg); C.toast('وضعیت فعلی خوانده شد: مرحله قبلاً توسط بازبین دیگر ' + X.REV[r.stale.res].label + ' شده است. تصمیم جدیدی ثبت نشد.', 'warning'); return true;
      case 'rs-approve': X.openSens(X.approveSpec(X.q(arg))); return true;
      case 'rs-reject': X.openSens(X.rejectSpec(X.q(arg))); return true;
      case 'run-gen': X.openSens(X.runGenSpec(X.runView(X.run(arg)))); return true;
      case 'run-approve': X.openSens(X.runApproveSpec(X.runView(X.run(arg)))); return true;
      case 'run-post': X.openSens(X.runPostSpec(X.runView(X.run(arg)), false)); return true;
      case 'run-tail': X.openSens(X.runPostSpec(X.runView(X.run(arg)), true)); return true;
      case 'run-lookup': X.runLookup(arg); C.render(); C.openDrawer('run', arg); C.toast('وضعیت واقعی خوانده شد (فقط‌خواندنی، نمایشی). نتیجهٔ هر آیتم تطبیق شد؛ تکرار کور انجام نشد.', 'info'); return true;
      case 'commit-sens':
        sp = st.spec; if (!((!sp.needsReason || ok3(k.reason)) && k.ack)) { k.err = true; C.rerenderDrawer(); return true; } sp.reasonText = k.reason || '';
        if (sp.kind === 'approve' || sp.kind === 'reject') op = X.commitDecision(sp); else op = X.commitRun(sp);
        if (op) result(op); return true;
      case 'commit-bulkapp': if (!k.ack) { return true; } op = X.runBulkApp(); result(op); return true;
      case 'retry-op': op = X.opOf(arg); X.retryFailed(op); C.render(); C.openDrawer('res', arg); C.toast('تکرار فقط برای موارد ناموفق معلوم و پس از بازخوانی تازه انجام شد (نمایشی).', 'success'); return true;
      case 'reconcile-op': op = X.opOf(arg); X.reconcileOp(op); C.render(); C.openDrawer('res', arg); C.toast('تطبیق انجام شد؛ نتیجهٔ واقعی هر مورد مشخص شد (نمایشی).', 'info'); return true;
      case 'issue-handoff': C.closeDrawer(); C.toast('ارجاع با مالک، زمان و مسئول ثبت شد (نمایشی). مسئله هنوز حل‌شده نیست؛ مدرک حل لازم است.', 'info'); return true;
      case 'diag-ro': C.toast('تشخیص خواندنی انجام شد (نمایشی): هیچ داده یا مانده‌ای تغییر نکرد.', 'info'); return true;
      case 'perm': st.perm = arg; C.render(); C.toast(arg === 'view' ? 'شبیه‌سازی «فقط مشاهده»: اقدام‌های نوشتنی اکنون غیرفعال‌اند.' : 'شبیه‌سازی طرح هدف با اختیار فرضی؛ اجرای واقعی تأیید نشده است.', 'info'); return true;
      case 'x': return true;
    }
    return false;
  }
  document.addEventListener('change', function (e) {
    var t = e.target, k = keepOf();
    if (t.matches('[data-sel]')) { st[t.getAttribute('data-sel')][t.getAttribute('data-id')] = t.checked; keepFocus(t); return; }
    var chk = { 'data-sensack': 'ack', 'data-bulkack': 'ack' };
    for (var a in chk) if (t.hasAttribute(a)) { k[chk[a]] = t.checked; C.rerenderDrawer(); var n = $('#drawer [' + a + ']'); if (n) n.focus(); return; }
  });
  document.addEventListener('input', function (e) {
    var t = e.target, k = keepOf();
    if (t.hasAttribute('data-sensreason')) { k.reason = t.value; k.err = false; var id = t.id; C.rerenderDrawer(); refocus('#' + id); }
  });

  function palette(nq, match) {
    var out = [];
    if (!nq) return [
      { g: 'اقدام سریع', icon: 'receipt', label: 'صف بررسی', run: function () { C.go('rev'); } }, { g: 'اقدام سریع', icon: 'swap', label: 'تطبیق و استرداد', run: function () { C.go('rec'); } },
      { g: 'اقدام سریع', icon: 'wallet', label: 'دفتر کل', run: function () { C.go('led'); } }, { g: 'اقدام سریع', icon: 'layers', label: 'قوانین، اجراها و ثبت', run: function () { C.go('run'); } },
      { g: 'اقدام سریع', icon: 'chart', label: 'گزارش‌ها', run: function () { C.go('rep'); } }, { g: 'اقدام سریع', icon: 'history', label: 'ممیزی و تاریخچه', run: function () { C.go('aud'); } },
      { g: 'اقدام سریع', icon: 'shield', label: 'پیکربندی و اختیار اقدام‌ها', run: function () { C.go('cfg'); } }, { g: 'اقدام سریع', icon: 'lock', label: 'نگهداری و بازیابی (پیشرفته)', run: function () { C.go('mnt'); } }
    ];
    M.queue.forEach(function (r) { if (match(r.inv) || match(r.id) || (r.cs && match(r.cs))) out.push({ g: 'مراحل پرداخت', icon: 'receipt', label: r.inv + ' · ' + r.id, meta: r.purpose, run: function () { C.go('rev'); C.openDrawer('rs', r.id); } }); });
    M.ledger.forEach(function (t) { if (match(t.id) || match(t.key)) out.push({ g: 'تراکنش‌ها', icon: 'wallet', label: t.id + ' · ' + t.key, meta: X.LGS[t.st].label, run: function () { C.go('led'); C.openDrawer('tx', t.id === '—' ? t.key : t.id); } }); });
    M.runs.forEach(function (r) { if (match(r.id) || match(r.name)) out.push({ g: 'اجراها', icon: 'layers', label: r.id + ' · ' + r.name, meta: X.RUN[X.runView(r).st].label, run: function () { C.go('run'); C.openDrawer('run', r.id); } }); });
    M.issues.forEach(function (i) { if (match(i.title) || match(i.id)) out.push({ g: 'مسائل تطبیق', icon: 'alert', label: i.title, meta: i.id, run: function () { C.go('rec'); C.openDrawer('iss', i.id); } }); });
    return out;
  }

  C.boot({
    home: 'rev', titleSuffix: 'پنل مالی (نمونه)', palettePlaceholder: 'شناسهٔ فاکتور، مرحله، تراکنش، اجرا یا یک فرمان… (شماره تلفن شناسه مالی نیست)',
    nav: function () {
      var P = M.queue.filter(function (r) { return X.eff(r).review === 'pending'; }).length;
      return [
        { id: 'rev', label: 'صف بررسی', icon: 'receipt', group: 'pri', mobile: 'بررسی', count: function () { return P; }, alert: true },
        { id: 'rec', label: 'تطبیق و استرداد', icon: 'swap', group: 'pri', mobile: 'تطبیق', count: function () { return M.issues.length; }, alert: true },
        { id: 'led', label: 'دفتر کل', icon: 'wallet', group: 'pri', mobile: 'دفتر کل' },
        { id: 'run', label: 'قوانین، اجراها و ثبت', icon: 'layers', group: 'pri', mobile: 'اجراها', count: function () { return M.runs.filter(function (r) { return ['unknown', 'partial'].indexOf(X.runView(r).st) > -1; }).length; }, alert: true },
        { id: 'rep', label: 'گزارش‌ها', icon: 'chart', group: 'pri', mobile: 'گزارش' },
        { id: 'aud', label: 'ممیزی و تاریخچه', icon: 'history', group: 'cond' },
        { id: 'cfg', label: 'پیکربندی و اختیار', icon: 'shield', group: 'cond' },
        { id: 'mnt', label: 'نگهداری و بازیابی', icon: 'lock', group: 'adv' }
      ];
    },
    views: V, drawers: D, wideDrawers: ['rs', 'sens', 'bulkrev', 'bulkapp', 'ref', 'tx', 'run', 'iss', 'diag', 'rep', 'audit', 'res'],
    pageTours: { rev: 'finance', rec: 'recon', led: 'ledger', run: 'posting' },
    workload: function () { return '<span>' + ic('receipt') + '</span><span><b>' + fa(M.queue.filter(function (r) { return X.eff(r).review === 'pending'; }).length) + '</b> مرحله منتظر بررسی</span><span class="sep extra"></span><span class="due extra">' + ic('question') + ' <b style="color:inherit">' + fa(M.runs.filter(function (r) { return X.runView(r).st === 'unknown'; }).length) + '</b> نتیجه نامعلوم</span>'; },
    userMenuTop: function () { return [{ label: 'ممیزی و تاریخچه', icon: 'history', go: 'aud' }, { label: 'پیکربندی و اختیار اقدام‌ها', icon: 'shield', go: 'cfg' }]; },
    empty: {
      rev: ['مرحله‌ای در این صف نیست', 'این فقط برای صف بارگذاری‌شده است و نبود مرحله را اثبات نمی‌کند؛ با خطا یا نبود دسترسی هم فرق دارد.', '<button type="button" class="btn btn-soft" data-rq="needs">نیازمند بررسی</button>'], rec: ['مسئله‌ای نیست', 'فقط برای منابع دریافت‌شده؛ سلامت مالی را اثبات نمی‌کند.', ''], led: ['تراکنشی در این نما نیست', 'نبود رکورد با صفر یکی نیست.', ''],
      run: ['اجرایی نیست', 'نبود اجرا با نبود نتیجه نامعلوم یکی نیست.', ''], rep: ['گزارشی نیست', '', ''], aud: ['رویدادی نیست', '', ''], cfg: ['موردی نیست', '', ''], mnt: ['موردی نیست', '', '']
    },
    onClick: onClick, palette: palette,
    init: function (state, qs) { ['rq', 'recq', 'lq', 'runq', 'ldom'].forEach(function (k) { if (qs.get(k)) st[k] = qs.get(k); }); if (qs.get('perm') === 'view') st.perm = 'view'; },
    afterBoot: function (qs) { setTimeout(function () { if (qs.get('flow')) FINAPI.flow(qs.get('flow')); }, 0); }
  });

  var FINAPI = window.FINAPI = {
    st: st,
    flow: function (f) {
      if (C.state.drawer) C.closeDrawer();
      var O = function (view, kind, id, keep) { C.go(view); C.openDrawer(kind, id, keep); };
      if (f === 'item') return O('rev', 'rs', 'RS-2101');
      if (f === 'gateway') return O('rev', 'rs', 'RS-2103');
      if (f === 'noevidence') return O('rev', 'rs', 'RS-2104');
      if (f === 'mismatch') return O('rev', 'rs', 'RS-2105');
      if (f === 'unit') return O('rev', 'rs', 'RS-2107');
      if (f === 'dup') return O('rev', 'rs', 'RS-2108');
      if (f === 'link') return O('rev', 'rs', 'RS-2111');
      if (f === 'stale') return O('rev', 'rs', 'RS-2106');
      if (f === 'stalecommit') { st.flow = 'stalecommit'; C.go('rev'); return X.openSens(X.approveSpec(X.q('RS-2112'))); }
      if (f === 'approve') { C.go('rev'); return X.openSens(X.approveSpec(X.q('RS-2101'))); }
      if (f === 'reject') { C.go('rev'); return X.openSens(X.rejectSpec(X.q('RS-2101'))); }
      if (f === 'rejected') return O('rev', 'rs', 'RS-2109');
      if (f === 'approved') return O('rev', 'rs', 'RS-2110');
      if (f === 'bulkapp') { st.rq = 'needs'; st.sel = { 'RS-2101': true, 'RS-2102': true, 'RS-2112': true, 'RS-2113': true, 'RS-2114': true, 'RS-2105': true }; C.go('rev'); return C.openDrawer('bulkapp', 'x'); }
      if (f === 'bulkrev') { st.sel = { 'RS-2101': true, 'RS-2105': true, 'RS-2112': true }; C.go('rev'); return C.openDrawer('bulkrev', 'x'); }
      if (f === 'refund') return O('rec', 'ref', 'RF-31', (st.recq = 'refunds', {}));
      if (f === 'refundunknown') { st.recq = 'refunds'; return O('rec', 'ref', 'RF-34'); }
      if (f === 'issue') { st.recq = 'issues'; return O('rec', 'iss', 'I-8'); }
      if (f === 'ledger') return O('led', 'tx', 'T-9004');
      if (f === 'ledgerunk') return O('led', 'tx', 'RUN-305|INV-48012|S1|rec:HP-301');
      if (f === 'orphan') return O('led', 'tx', 'T-9012');
      if (f === 'runpreview') { st.runq = 'runs'; return O('run', 'run', 'RUN-311'); }
      if (f === 'runapproved') { st.runq = 'runs'; return O('run', 'run', 'RUN-309'); }
      if (f === 'runpartial') { st.runq = 'runs'; return O('run', 'run', 'RUN-307'); }
      if (f === 'rununknown') { st.runq = 'runs'; return O('run', 'run', 'RUN-305'); }
      if (f === 'runposted') { st.runq = 'runs'; return O('run', 'run', 'RUN-308'); }
      if (f === 'runfailed') { st.runq = 'runs'; return O('run', 'run', 'RUN-306'); }
      if (f === 'runrecon') { st.runq = 'runs'; return O('run', 'run', 'RUN-304'); }
      if (f === 'rundraft') { st.runq = 'runs'; return O('run', 'run', 'RUN-312'); }
      if (f === 'report') return O('rep', 'rep', 'REP-CUR');
      if (f === 'view') { st.perm = 'view'; return O('rev', 'rs', 'RS-2102'); }
    }
  };
})();
