/* HR — role layer, part 4: events, palette, boot. Runs on the shared CRM runtime (../shared/crm-core.js).
   Prototype only: no data is saved or sent; no handler performs a real HR action. HR ROLE RUNTIME NOT LIVE VERIFIED. */
(function () {
  'use strict';
  var X = window.HRX, C = X.C, M = X.M, h = X.h, st = X.st, V = X.V, D = X.D;
  var $ = h.$, esc = h.esc, fa = h.fa, ic = h.ic;
  var ok3 = function (s) { return (s || '').trim().length > 2; };
  var keepOf = function () { return C.state.drawer ? C.state.drawer.keep : {}; };
  var PURPOSES = ['بررسی مشکل پشتیبانی گزارش‌شده', 'تأیید پیکربندی دسترسی پس از تغییر سمت'];
  function refocus(sel) { var n = $(sel); if (n) { n.focus({ preventScroll: true }); try { var l = n.value.length; n.setSelectionRange(l, l); } catch (e) {} } }
  // Persistent impersonation boundary: lives in the shell (outside #ws and the drawer), so it survives navigation, drawers and page-state simulations.
  X.syncImp = function () {
    var n = $('#imp-shell'); if (!n) return; var html = st.imp ? X.impBar() : '';
    if (n.getAttribute('data-h') !== html) { n.innerHTML = html; n.setAttribute('data-h', html); }
    document.body.classList.toggle('imp-on', !!st.imp);
    document.body.style.setProperty('--imp-h', st.imp ? n.offsetHeight + 'px' : '0px');
  };
  window.addEventListener('resize', function () { if (st.imp) X.syncImp(); });
  function result(op) { C.closeDrawer(); C.render(); C.openDrawer('result', op.id); }

  function onClick(t, e) {
    var tog = { 'data-wq': 'wq', 'data-oq': 'oq', 'data-rq': 'rq', 'data-aq': 'aq', 'data-xq': 'xq', 'data-dq': 'dq' };
    for (var attr in tog) if (t.hasAttribute(attr)) { st[tog[attr]] = t.getAttribute(attr); C.state.cursor = 0; if (C.state.drawer) C.closeDrawer(); C.render(); return true; }
    var a = t.getAttribute('data-act'); if (!a) return false;
    var arg = a.indexOf(':') > -1 ? a.slice(a.indexOf(':') + 1) : null, verb = arg != null ? a.slice(0, a.indexOf(':')) : a;
    e.preventDefault(); e.stopPropagation();
    if (t.disabled || t.getAttribute('aria-disabled') === 'true') return true;
    var k = keepOf(), p, r, sp, op;
    switch (verb) {
      case 'goto': C.closeMenu(); C.closeDrawer(); C.go(arg); return true;
      case 'open-staff': case 'open-req': case 'open-access': case 'open-comp': case 'open-exc': case 'open-audit': case 'open-pos': case 'open-term': case 'open-credreset': case 'open-impstart':
        C.closeMenu(); C.openDrawer(verb.slice(5), arg); return true;
      case 'open-xfer': C.closeMenu(); C.openDrawer('xfer', arg, { target: st.flow === 'targetinactive' ? 'HP-130' : null }); return true;
      case 'open-onbwiz': C.openDrawer('onbwiz', 'x', { step: 0 }); return true;
      case 'open-bulkwf': C.openDrawer('bulkwf', 'x'); return true;
      case 'open-bulkcred': C.openDrawer('bulkcred', 'x'); return true;
      case 'open-export': C.openDrawer('export', 'x'); return true;
      case 'open-op': C.openDrawer('result', arg); return true;
      case 'xfer-review': p = X.s(arg); X.openSens(X.xferSpec(p, X.s(k.target))); return true;
      case 'term-review': X.openSens(X.termSpec(X.s(arg))); return true;
      case 'access-review': p = X.s(arg); X.openSens(X.accessSpec(p, k.after)); return true;
      case 'comp-review': p = X.s(arg); X.openSens(X.compSpec(p, k)); return true;
      case 'cred-review': p = X.s(arg); sp = X.credSpec(p, { sms: !!($('[data-sms]') || {}).checked }); sp.reasonText = ($('#pw-r') || {}).value || ''; sp.smsFail = st.flow === 'credpartial'; X.openSens(sp); return true;
      case 'imp-review': p = X.s(arg); X.openSens(X.impSpec(p, k)); return true;
      case 'req-apply': X.openSens(X.reqApplySpec(X.req(arg))); return true;
      case 'req-step': r = X.req(arg); X.reqStep(r); C.render(); C.openDrawer('req', arg); C.toast('تأیید مرحله‌ای ثبت شد (نمایشی). هنوز اعمال نشده است؛ اعمال نهایی با منابع انسانی و جدا است.', 'info'); return true;
      case 'req-reject': r = X.req(arg); if (!ok3(k.note)) { k.err = true; C.rerenderDrawer(); refocus('#rj-n'); return true; } X.reqReject(r, k.note); C.render(); C.openDrawer('req', arg); C.toast('درخواست با دلیل رد شد (نمایشی).', 'info'); return true;
      case 'req-reconcile': st.recon = st.recon || {}; st.recon[arg] = true; C.rerenderDrawer(); C.toast('اثر واقعی خوانده شد؛ اعمال دوباره کور انجام نمی‌شود و سیاست بازیابی تعریف نشده است.', 'info'); return true;
      case 'commit-sens':
        sp = st.spec; if (!(ok3(k.reason) && k.ack)) { k.err = true; C.rerenderDrawer(); return true; } sp.reasonText = k.reason;
        if (sp.kind === 'xfer') op = X.commitXfer(sp); else if (sp.kind === 'term') op = X.commitTerm(sp); else if (sp.kind === 'reqapply') op = X.commitReqApply(sp); else if (sp.kind === 'access') op = X.commitAccess(sp); else if (sp.kind === 'comp') op = X.commitComp(sp); else if (sp.kind === 'cred') op = X.commitCred(sp);
        else if (sp.kind === 'imp') { st.imp = { target: sp.p, purpose: sp.purpose, at: 'همین الان' }; X.syncImp(); C.closeDrawer(); C.go('cred'); C.toast('جلسه نمایش آغاز شد (نمایشی). این جلسه فقط‌خواندنی نیست؛ بازگشت لازم است.', 'warning'); return true; }
        if (op) result(op); return true;
      case 'imp-end': st.imp = null; X.syncImp(); C.render(); C.toast('به حساب منابع انسانی بازگشتید (نمایشی). پایان جلسه ثبت شد؛ ممیزی ماندگار تأیید نشده است.', 'info'); return true;
      case 'commit-bulkwf': X.runBulkWf(); return true;
      case 'commit-bulkcred': X.runBulkCred(); return true;
      case 'retry-op': op = X.opOf(arg); X.retryFailed(op); C.render(); C.openDrawer('result', arg); C.toast('تکرار فقط برای موارد ناموفق معلوم و پس از بررسی تازه انجام شد (نمایشی).', 'success'); return true;
      case 'reconcile-op': op = X.opOf(arg); X.reconcileOp(op); C.render(); C.openDrawer('result', arg); C.toast('تطبیق انجام شد؛ نتیجه واقعی هر مورد مشخص شد (نمایشی).', 'info'); return true;
      case 'onb-next': k.step = (k.step || 0) + 1; C.rerenderDrawer(); return true;
      case 'onb-back': k.step = Math.max(0, (k.step || 0) - 1); C.rerenderDrawer(); return true;
      case 'onb-commit': if (!(ok3(k.reason) && k.ack)) { k.err = true; C.rerenderDrawer(); return true; } k.reasonText = k.reason; X.commitOnb(k); C.render(); C.rerenderDrawer(); C.toast('ورود نیرو ثبت شد (نمایشی)؛ برخی اثرها باز مانده‌اند.', 'info'); return true;
      case 'comp-auth': st.compAuth = arg; C.render(); return true;
      case 'x': return true;
    }
    return false;
  }
  document.addEventListener('change', function (e) {
    var t = e.target, k = keepOf(), g = function (a) { return t.getAttribute(a); };
    var set = function (prop, val) { k[prop] = val; C.rerenderDrawer(); var n = $('#drawer input:checked, #drawer [data-focus]'); if (n) n.focus(); };
    if (t.hasAttribute('data-xt')) return set('target', g('data-xt'));
    if (t.hasAttribute('data-after')) return set('after', g('data-after'));
    if (t.hasAttribute('data-ochoice')) return set('choice', g('data-ochoice'));
    if (t.hasAttribute('data-opos')) return set('pos', g('data-opos'));
    if (t.hasAttribute('data-opar')) return set('par', g('data-opar'));
    if (t.hasAttribute('data-purpose')) return set('purpose', PURPOSES[Number(g('data-purpose'))]);
    var chk = { 'data-sensack': 'ack', 'data-bulkack': 'ack', 'data-ack0': 'ack0' };
    for (var a in chk) if (t.hasAttribute(a)) { k[chk[a]] = t.checked; C.rerenderDrawer(); var n = $('#drawer [' + a + ']'); if (n) n.focus(); return; }
    if (t.hasAttribute('data-pw') || t.hasAttribute('data-pw2')) { credCheck(true); return; }
  });
  function credCheck(touch) {
    var p1 = $('#pw-1'), p2 = $('#pw-2'), r = $('#pw-r'), b = $('[data-act^="cred-review"]'); if (!p1 || !b) return;
    var v1 = p1.value.length >= 8, v2 = p1.value === p2.value && p2.value.length > 0, vr = ok3(r.value);
    if (touch || p1.getAttribute('aria-invalid')) { p1.setAttribute('aria-invalid', String(!v1 && p1.value.length > 0)); $('#pw-1e').hidden = v1 || !p1.value.length; if (!v1 && p1.value.length) p1.setAttribute('aria-describedby', 'pw-1e'); else p1.removeAttribute('aria-describedby'); }
    if (touch || p2.getAttribute('aria-invalid')) { var bad2 = !v2 && p2.value.length > 0; p2.setAttribute('aria-invalid', String(bad2)); $('#pw-2e').hidden = !bad2; if (bad2) p2.setAttribute('aria-describedby', 'pw-2e'); else p2.removeAttribute('aria-describedby'); }
    var ok = v1 && v2 && vr; b.disabled = !ok; b.setAttribute('aria-disabled', String(!ok));
  }
  document.addEventListener('input', function (e) {
    var t = e.target, k = keepOf();
    if (t.matches('[data-pw],[data-pw2],[data-pwreason]')) { credCheck(false); return; }   // password values are never stored or re-rendered
    var map = { 'data-sensreason': 'reason', 'data-rejnote': 'note', 'data-cpbase': 'base', 'data-cpfrom': 'from', 'data-cpreason': 'reason' };
    for (var a in map) if (t.hasAttribute(a)) { k[map[a]] = t.value; if (a === 'data-rejnote' || a === 'data-sensreason') k.err = false; var id = t.id; C.rerenderDrawer(); refocus('#' + id); return; }
  });

  /* ---------- Palette (mobile/phone appears only as «discovery», never as identity) ---------- */
  function palette(nq, match) {
    var out = [];
    if (!nq) return [
      { g: 'اقدام سریع', icon: 'users', label: 'نیروها', run: function () { C.go('work'); } }, { g: 'اقدام سریع', icon: 'swap', label: 'ورود نیرو و انتقال', run: function () { C.go('onb'); } },
      { g: 'اقدام سریع', icon: 'inbox', label: 'درخواست‌ها', run: function () { C.go('req'); } }, { g: 'اقدام سریع', icon: 'key', label: 'ساختار و دسترسی', run: function () { C.go('acc'); } },
      { g: 'اقدام سریع', icon: 'wallet', label: 'جبران خدمات', run: function () { C.go('comp'); } }, { g: 'اقدام سریع', icon: 'alert', label: 'استثناها و ممیزی', run: function () { C.go('exc'); } },
      { g: 'اقدام سریع', icon: 'lock', label: 'اعتبارنامه (محدود)', run: function () { C.go('cred'); } }
    ];
    M.staff.forEach(function (p) { if (match(p.name) || match(p.id)) out.push({ g: 'نیروها', icon: 'user', label: p.name, meta: p.id, run: function () { C.go('work'); C.openDrawer('staff', p.id); } }); });
    if (/^[0-9۰-۹]{3,}$/.test(nq)) M.staff.forEach(function (p) { if (match(p.mob)) out.push({ g: 'کشف با شماره (هویت نیست)', icon: 'phone', label: p.mob + ' ← ' + p.name, meta: 'فقط کشف', run: function () { C.go('work'); C.openDrawer('staff', p.id); } }); });
    M.requests.forEach(function (r) { if (match(r.id) || match(X.name(r.subj))) out.push({ g: 'درخواست‌ها', icon: 'inbox', label: r.id + ' · ' + X.name(r.subj), meta: X.REQ[X.reqSt(r)].label, run: function () { st.rq = ['pending_hr'].indexOf(X.reqSt(r)) > -1 ? 'review' : ['approved', 'rejected', 'failed'].indexOf(X.reqSt(r)) > -1 ? 'closed' : 'chain'; C.go('req'); C.openDrawer('req', r.id); } }); });
    M.exc.forEach(function (x) { if (match(x.title)) out.push({ g: 'استثناها', icon: 'alert', label: x.title, meta: x.id, run: function () { C.go('exc'); C.openDrawer('exc', x.id); } }); });
    return out;
  }

  C.boot({
    home: 'work', titleSuffix: 'پنل منابع انسانی (نمونه)', palettePlaceholder: 'نام نیرو، شناسه پروفایل، درخواست یا یک فرمان… (موبایل فقط برای کشف)',
    nav: function () {
      var P = M.requests.filter(function (r) { return X.reqSt(r) === 'pending_hr'; }).length;
      return [
        { id: 'work', label: 'نیروها', icon: 'users', group: 'pri', mobile: 'نیروها', count: function () { return M.staff.filter(function (p) { return p.conf.length; }).length; }, alert: true },
        { id: 'onb', label: 'ورود نیرو و انتقال', icon: 'swap', group: 'pri', mobile: 'ورود' },
        { id: 'req', label: 'درخواست‌ها', icon: 'inbox', group: 'pri', mobile: 'درخواست', count: function () { return P; }, alert: true },
        { id: 'acc', label: 'ساختار و دسترسی', icon: 'key', group: 'pri', mobile: 'دسترسی' },
        { id: 'comp', label: 'جبران خدمات', icon: 'wallet', group: 'pri' },
        { id: 'exc', label: 'استثناها و ممیزی', icon: 'alert', group: 'pri', mobile: 'استثنا', count: function () { return M.exc.length; }, alert: true },
        { id: 'cred', label: 'اعتبارنامه (محدود)', icon: 'lock', group: 'adv' },
        { id: 'bulk', label: 'ورود/گروهی/خروجی', icon: 'layers', group: 'cond' },
        { id: 'diag', label: 'نگاشت قدیمی', icon: 'shield', group: 'cond' }
      ];
    },
    views: V, drawers: D, wideDrawers: ['staff', 'xfer', 'term', 'req', 'access', 'comp', 'sens', 'result', 'onbwiz', 'exc', 'audit', 'credreset', 'impstart', 'bulkwf', 'bulkcred', 'pos'],
    pageTours: { work: 'hr', onb: 'transfer', req: 'term', cred: 'restricted' },
    workload: function () { return '<span>' + ic('inbox') + '</span><span><b>' + fa(M.requests.filter(function (r) { return X.reqSt(r) === 'pending_hr'; }).length) + '</b> منتظر اعمال منابع انسانی</span><span class="sep extra"></span><span class="due extra">' + ic('alert') + ' <b style="color:inherit">' + fa(M.exc.length) + '</b> استثنا</span>'; },
    userMenuTop: function () { return [{ label: 'استثناها و ممیزی', icon: 'alert', go: 'exc' }, { label: 'اعتبارنامه (محدود)', icon: 'lock', go: 'cred' }]; },
    empty: {
      work: ['نیرویی در این صف نیست', 'این فقط برای فهرست بارگذاری‌شده است و نبود نیرو را اثبات نمی‌کند.', '<button type="button" class="btn btn-soft" data-wq="all">همه نیروها</button>'], onb: ['موردی نیست', 'خالی بودن لیست، نبود ورود یا انتقال را اثبات نمی‌کند.', ''],
      req: ['درخواستی در این صف نیست', 'صف دیگری را انتخاب کنید.', '<button type="button" class="btn btn-soft" data-rq="review">منتظر اعمال</button>'], acc: ['موردی نیست', '', ''], comp: ['جبران خدمات ثبت نشده', 'نبود رکورد با صفر یکی نیست.', ''],
      exc: ['استثنایی ثبت نشده', 'فقط برای منابع دریافت‌شده؛ سلامت کامل را اثبات نمی‌کند.', ''], cred: ['حسابی نیست', '', ''], bulk: ['موردی نیست', '', ''], diag: ['موردی نیست', '', '']
    },
    onClick: onClick, palette: palette,
    init: function (state, qs) { ['wq', 'oq', 'rq', 'aq', 'xq', 'dq', 'compAuth'].forEach(function (k) { if (qs.get(k)) st[k] = qs.get(k); }); if (qs.get('imp')) st.imp = { target: 'HP-302', purpose: PURPOSES[0], at: 'همین الان' }; },
    afterRender: function () { X.syncImp(); },
    afterBoot: function (qs) { X.syncImp(); setTimeout(function () { if (qs.get('flow')) HRAPI.flow(qs.get('flow')); }, 0); }
  });

  var HRAPI = window.HRAPI = {
    st: st,
    flow: function (f) {
      if (C.state.drawer) C.closeDrawer();
      var O = function (view, kind, id, keep) { C.go(view); C.openDrawer(kind, id, keep); };
      if (f === 'xfer') return O('onb', 'xfer', 'HP-301', { target: 'HP-120' });
      if (f === 'targetinactive') { st.flow = 'targetinactive'; return O('onb', 'xfer', 'HP-303', { target: 'HP-130' }); }
      if (f === 'effconflict') { st.flow = 'effconflict'; return O('onb', 'xfer', 'HP-301', { target: 'HP-120' }); }
      if (f === 'xpartial' || f === 'xunknown') { st.flow = f; C.go('onb'); return X.openSens(X.xferSpec(X.s('HP-301'), X.s('HP-120'))); }
      if (f === 'term') return O('req', 'term', 'HP-304');
      if (f === 'termclear') return O('req', 'term', 'HP-311');
      if (f === 'reqstep') { st.rq = 'chain'; return O('req', 'req', 'R-501'); }
      if (f === 'reqapply') return O('req', 'req', 'R-502');
      if (f === 'applyfail') { st.flow = 'applyfail'; return O('req', 'req', 'R-502'); }
      if (f === 'reqtermblock') return O('req', 'req', 'R-504');
      if (f === 'reqfailed') { st.rq = 'closed'; return O('req', 'req', 'R-506'); }
      if (f === 'reqchanged') { st.rq = 'chain'; return O('req', 'req', 'R-508'); }
      if (f === 'reqoos') { st.rq = 'chain'; return O('req', 'req', 'R-509'); }
      if (f === 'reqapplied') { st.rq = 'closed'; return O('req', 'req', 'R-505'); }
      if (f === 'access') return O('acc', 'access', 'HP-301', { after: 'sup' });
      if (f === 'accpartial') { st.flow = 'accpartial'; return O('acc', 'access', 'HP-301', { after: 'sup' }); }
      if (f === 'comp') return O('comp', 'comp', 'HP-303');
      if (f === 'compunknown') { st.flow = 'compunknown'; return O('comp', 'comp', 'HP-301'); }
      if (f === 'compdenied') { st.compAuth = 'denied'; return C.go('comp'); }
      if (f === 'cred') return O('cred', 'credreset', 'HP-301');
      if (f === 'credlead') return O('cred', 'credreset', 'HP-110');
      if (f === 'credpartial') { st.flow = 'credpartial'; return O('cred', 'credreset', 'HP-302'); }
      if (f === 'imp') return O('cred', 'impstart', 'HP-302');
      if (f === 'implead') return O('cred', 'impstart', 'HP-110');
      if (f === 'impactive') { st.imp = { target: 'HP-302', purpose: PURPOSES[0], at: 'همین الان' }; X.syncImp(); return C.go('cred'); }
      if (f === 'onb') return C.openDrawer('onbwiz', 'x', { step: 0 });
      if (f === 'staff') return O('work', 'staff', 'HP-303');
      if (f === 'dup') return O('work', 'staff', 'HP-305');
      if (f === 'bulkwf') return O('bulk', 'bulkwf', 'x');
      if (f === 'bulkcred') return O('bulk', 'bulkcred', 'x');
      if (f === 'exc') return O('exc', 'exc', 'X8');
    }
  };
})();
