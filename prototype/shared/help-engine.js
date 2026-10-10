/* Shared Help System — Help Center, welcome, guided tours, demo states. Same engine/behaviour as Seller V1 help.js;
   all content comes from a role registry (window.CRM_HELP). Tours never execute real actions. Prototype only. */
(function () {
  'use strict';
  var B = window.CRM, R = window.CRM_HELP;
  if (!B || !R) return;
  var h = B.h, $ = h.$, fa = h.fa, esc = h.esc, ic = h.ic, store = h.store;
  var kbd = function (k) { return '<span class="kbd">' + k + '</span>'; };
  /* SN-207: core cross-role terms (wording reused verbatim from the frozen role glossaries). A role's own entry always wins; a core term is
     appended only when the role glossary has no entry containing that title. Presentation terminology only. */
  var CORE_TERMS = [
    ['مسئول فعلی', 'پرونده الان نزد کیست (Current Custody).'],
    ['مالک اولیه', 'نخستین دریافت‌کننده پرونده؛ با تغییر مسئول عوض نمی‌شود.'],
    ['اقدام بعدی با', 'چه کسی یا کدام صف باید گام بعدی همین گردش‌کار را انجام دهد؛ طبق گردش‌کار همان حوزه تعیین می‌شود و حدس زده نمی‌شود.'],
    ['عامل رویداد', 'کسی که رویداد (مثلاً انتقال یا صدور) را انجام داده؛ مالک فعلی یا مالک اعتبار نیست.'],
    ['مالک اعتبار', 'طبق قوانین مالی فعلی تعیین می‌شود؛ تخصیص، برگشت یا تغییر والد آن را عوض نمی‌کند.'],
    ['نامعلوم', 'پاسخ نرسید و ممکن است ثبت شده باشد؛ پیش از تکرار باید تطبیق شود.']
  ];
  var GLOSS = (R.glossary || []).concat(CORE_TERMS.filter(function (c) { return !(R.glossary || []).some(function (g) { return g[0].indexOf(c[0]) > -1; }); }));
  var TOURS = R.tours;
  var isDone = function (id) { return store.get(TOURS[id].key + '_completed') === '1'; };

  /* ---------- Tour engine ---------- */
  var T = null;
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
    if (els.length === 1 && els[0].id === 'drawer') {
      var d = els[0];
      return innerWidth <= 760 ? { top: innerHeight - d.offsetHeight, left: 0, right: innerWidth, bottom: innerHeight } : { top: 0, left: 0, right: d.offsetWidth, bottom: innerHeight };
    }
    var r = { top: Infinity, left: Infinity, right: -Infinity, bottom: -Infinity };
    els.forEach(function (e) { var b = e.getBoundingClientRect(); r.top = Math.min(r.top, b.top); r.left = Math.min(r.left, b.left); r.right = Math.max(r.right, b.right); r.bottom = Math.max(r.bottom, b.bottom); });
    return r;
  }
  function startTour(id, at) {
    if (!TOURS[id]) return;
    closeHelp(); closeDemo(); B.closeMenu();
    if ($('#palette') && !$('#palette').hidden) $('#palette').hidden = true;
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
    if (scroll && (r.top < 60 || r.bottom > vh - 20) && r.bottom - r.top < vh * 0.8 && !els[0].closest('#drawer')) { els[0].scrollIntoView({ block: 'center', behavior: 'auto' }); r = unionRect(els); }
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
    store.set(tour.key + (completed ? '_completed' : '_dismissed'), '1');
    T = null;
    if (silent) return;
    if (B.state.drawer) B.closeDrawer();
    if (R.afterTour) R.afterTour();
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
      var row = function (r) { return '<div class="kb-row"><span>' + r[1] + '</span><span class="keys">' + r[0].map(function (k) { return k === 'یا' || k === '–' || k === '،' ? ' ' + k + ' ' : kbd(k); }).join('') + '</span></div>'; };
      return '<div class="hc-head"><button type="button" class="hc-back" data-hc="home">' + ic('arrowL') + 'بازگشت</button><h2 id="hc-title">میانبرهای کیبورد</h2><button type="button" class="icon-btn" data-hc-close aria-label="بستن راهنما">' + ic('x') + '</button></div>' +
        '<div class="kb-list">' + R.keys.map(row).join('') + '</div>' + (R.keysNote ? '<div class="hc-sub" style="padding-top:0">' + R.keysNote + '</div>' : '');
    }
    return '<div class="hc-head"><h2 id="hc-title">مرکز راهنما</h2><button type="button" class="icon-btn" data-hc-close aria-label="بستن راهنما">' + ic('x') + '</button></div>' +
      '<div class="hc-sub">آموزش‌های کوتاه تعاملی؛ هر زمان قابل تکرارند و هیچ داده‌ای را تغییر نمی‌دهند.</div>' +
      '<div class="hc-list">' + R.order.map(tourItem).join('') + '<div class="hc-sep"></div>' +
      (R.glossary ? '<button type="button" class="hc-item" data-hc="terms"><span class="hc-ico neutral">' + ic('info') + '</span><span class="hc-txt"><b>واژه‌ها و مفاهیم</b><span>مسئول فعلی، مالک اولیه، قابل برگشت و…</span></span></button>' : '') +
      '<button type="button" class="hc-item" data-hc="keys"><span class="hc-ico neutral">' + ic('key') + '</span><span class="hc-txt"><b>میانبرهای کیبورد</b><span>کار سریع‌تر بدون ماوس</span></span><span class="hc-meta">' + kbd('?') + '</span></button>' +
      '<button type="button" class="hc-item" data-restart-tours><span class="hc-ico neutral">' + ic('history') + '</span><span class="hc-txt"><b>شروع مجدد آموزش</b><span>آموزش سریع پنل را از ابتدا ببینید</span></span></button></div>' +
      '<div class="hc-foot"><span>راهنما · ' + esc(R.version || 'نسخه ۱') + '</span><span class="grow"></span><button type="button" class="hc-back" data-reset-onboarding>بازنشانی آموزش (دمو)</button></div>';
  }
  function termsHtml() {
    return '<div class="hc-head"><button type="button" class="hc-back" data-hc="home">' + ic('arrowL') + 'بازگشت</button><h2 id="hc-title">واژه‌ها و مفاهیم</h2><button type="button" class="icon-btn" data-hc-close aria-label="بستن راهنما">' + ic('x') + '</button></div>' +
      '<dl class="terms">' + GLOSS.map(function (g) { return '<dt>' + esc(g[0]) + '</dt><dd>' + esc(g[1]) + '</dd>'; }).join('') + '</dl>';
  }
  function openHelp(view) {
    closeDemo(); B.closeMenu();
    hcView = view || 'home';
    var hc = $('#help-center');
    if (!hc) { hc = document.createElement('div'); hc.id = 'help-center'; hc.className = 'help-center'; hc.setAttribute('role', 'dialog'); hc.setAttribute('aria-labelledby', 'hc-title'); document.body.appendChild(hc); }
    hc.innerHTML = hcView === 'terms' ? termsHtml() : helpHtml(); hc.hidden = false;
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
    w.innerHTML = '<div class="welcome" role="dialog" aria-modal="true" aria-labelledby="wl-t" aria-describedby="wl-b"><div class="art">' + ic('target') + '</div><h2 id="wl-t">' + esc(R.welcome.title) + '</h2>' +
      '<p id="wl-b">' + esc(R.welcome.body) + '</p>' +
      '<div class="acts"><button type="button" class="btn btn-primary btn-lg" data-welcome="start">شروع آموزش</button><button type="button" class="btn btn-ghost btn-lg" data-welcome="later">بعداً</button></div>' +
      '<div class="fine">آموزش همیشه از دکمه راهنما (؟) در بالای صفحه در دسترس است.</div></div>';
    w.hidden = false; w.querySelector('[data-welcome="start"]').focus();
  }
  function closeWelcome(later) {
    var w = $('#welcome'); if (!w) return; w.hidden = true; store.set(R.welcome.key, '1');
    if (later) { B.toast('هر زمان خواستید، آموزش از دکمه راهنما (؟) در دسترس است.', 'info'); $('#help-btn').focus(); }
  }

  /* ---------- Demo panel ---------- */
  function demoHtml() {
    var b = function (act, label) { return '<button type="button" class="dp" data-dp="' + act + '">' + label + '</button>'; };
    return '<div class="dp-h">' + ic('layers') + '<span>حالت‌های نمایشی</span><button type="button" class="icon-btn" data-dp="close" aria-label="بستن">' + ic('x') + '</button></div>' +
      R.demo.map(function (g) { return '<div class="dp-g">' + esc(g[0]) + '</div>' + g[1].map(function (x) { return b(x[0], x[1]); }).join(''); }).join('') +
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
    store.del(R.welcome.key);
    B.toast('وضعیت آموزش‌ها بازنشانی شد. با بارگذاری دوباره، پیام خوش‌آمد نمایش داده می‌شود.', 'info');
  }
  function runDemo(act) {
    closeDemo();
    if (act === 'close') return;
    if (act === 'welcome') return showWelcome();
    if (act === 'reset') return resetOnboarding();
    if (act.indexOf('tour:') === 0) return startTour(act.slice(5));
    if (act.indexOf('toast:') === 0) { var t = act.slice(6); return B.toast(R.toasts[t], t); }
    if (act === 'normal') return B.setSim(null);
    if (R.runDemo && R.runDemo(act) === true) return;
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
    if (t.matches('[data-hc]')) { e.stopPropagation(); return openHelp(t.getAttribute('data-hc')); }
    if (t.matches('[data-start-tour]')) return startTour(t.getAttribute('data-start-tour'));
    if (t.matches('[data-restart-tours]')) { Object.keys(TOURS).forEach(function (id) { store.del(TOURS[id].key + '_completed'); }); return startTour(R.order[0]); }
    if (t.matches('[data-reset-onboarding]')) { closeHelp(true); return resetOnboarding(); }
    if (t.matches('[data-page-tour]')) { e.stopPropagation(); return startTour(t.getAttribute('data-page-tour')); }
    if (t.matches('[data-welcome]')) { var st = t.getAttribute('data-welcome') === 'start'; closeWelcome(!st); if (st) startTour(R.order[0]); return; }
    if (t.matches('[data-dp]')) return runDemo(t.getAttribute('data-dp'));
    if (!inHelp) closeHelp(); if (!inDemo) closeDemo();
  }, true);

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

  /* ---------- Boot ---------- */
  var q = new URLSearchParams(location.search);
  var deep = ['open', 'menu', 'palette', 'sim', 'tour', 'help', 'demo', 'toast', 'flow'].some(function (k) { return q.has(k); });
  setTimeout(function () {
    if (q.get('tour')) startTour(q.get('tour'), Number(q.get('step') || 1));
    else if (q.get('help')) openHelp(q.get('help') === 'shortcuts' ? 'keys' : q.get('help') === 'terms' ? 'terms' : 'home');
    else if (q.get('demo')) openDemo();
    if (q.get('toast')) { var tt = q.get('toast'); B.toast(R.toasts[tt], tt, true); }
    if (q.get('welcome') === '1' || (!deep && q.get('welcome') !== '0' && store.get(R.welcome.key) !== '1')) showWelcome();
  }, 60);
  window.SNHelp = { startTour: startTour, openHelp: openHelp, showWelcome: showWelcome, reset: resetOnboarding, tours: TOURS };
})();
