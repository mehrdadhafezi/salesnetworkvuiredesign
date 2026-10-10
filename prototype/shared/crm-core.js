/* Shared CRM runtime — the role-agnostic part of the Seller V1 (frozen) shell, extracted for reuse by other roles.
   Seller V1 still runs its own frozen app.js; this file is the shared baseline for Supervisor and later roles.
   Demo only: nothing is saved or sent anywhere. */
(function () {
  'use strict';
  var root = document.documentElement;
  var $ = function (s, r) { return (r || document).querySelector(s); };
  var $$ = function (s, r) { return Array.prototype.slice.call((r || document).querySelectorAll(s)); };
  var esc = function (v) { return String(v == null ? '' : v).replace(/[&<>"']/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]; }); };
  var fa = function (v) { return String(v).replace(/\d/g, function (d) { return '۰۱۲۳۴۵۶۷۸۹'[d]; }); };
  var num = function (n) { return Number(n || 0).toLocaleString('fa-IR'); };
  var money = function (n) { return '<span class="amount"><span class="num">' + num(n) + '</span><small>تومان</small></span>'; };
  var store = { get: function (k) { try { return localStorage.getItem('snproto.' + k); } catch (e) { return null; } }, set: function (k, v) { try { localStorage.setItem('snproto.' + k, v); } catch (e) {} }, del: function (k) { try { localStorage.removeItem('snproto.' + k); } catch (e) {} } };

  /* ---------- Icons (Seller set + shared additions) ---------- */
  var P = {
    phone: '<path d="M22 16.9v3a2 2 0 0 1-2.2 2 19.8 19.8 0 0 1-8.6-3.1 19.5 19.5 0 0 1-6-6A19.8 19.8 0 0 1 2.1 4.2 2 2 0 0 1 4.1 2h3a2 2 0 0 1 2 1.7c.1 1 .4 1.9.7 2.8a2 2 0 0 1-.5 2.1L8 9.9a16 16 0 0 0 6 6l1.3-1.3a2 2 0 0 1 2.1-.4c.9.3 1.8.6 2.8.7a2 2 0 0 1 1.7 2z"/>',
    copy: '<rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>',
    more: '<circle cx="12" cy="5" r="1"/><circle cx="12" cy="12" r="1"/><circle cx="12" cy="19" r="1"/>',
    x: '<path d="M18 6 6 18M6 6l12 12"/>', up: '<path d="m18 15-6-6-6 6"/>', down: '<path d="m6 9 6 6 6-6"/>', chev: '<path d="m6 9 6 6 6-6"/>',
    clock: '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
    alert: '<path d="M12 9v4M12 17h.01"/><path d="M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0z"/>',
    file: '<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 2v6h6M8 13h8M8 17h5"/>',
    users: '<path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.9M16 3.1a4 4 0 0 1 0 7.8"/>',
    user: '<path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/>',
    swap: '<path d="m16 3 4 4-4 4M20 7H4M8 21l-4-4 4-4M4 17h16"/>',
    repeat: '<path d="m17 2 4 4-4 4"/><path d="M3 11v-1a4 4 0 0 1 4-4h14M7 22l-4-4 4-4"/><path d="M21 13v1a4 4 0 0 1-4 4H3"/>',
    receipt: '<path d="M4 2v20l3-2 3 2 3-2 3 2 3-2 1 .7V2l-1 .7L16 1l-3 2-3-2-3 2-3-2z"/><path d="M8 8h8M8 12h8M8 16h5"/>',
    activity: '<path d="M22 12h-4l-3 9L9 3l-3 9H2"/>', plus: '<path d="M12 5v14M5 12h14"/>',
    wallet: '<path d="M20 7V5a2 2 0 0 0-2-2H5a2 2 0 0 0 0 4h15a1 1 0 0 1 1 1v4h-4a2 2 0 0 0 0 4h4v3a1 1 0 0 1-1 1H5a2 2 0 0 1-2-2V5"/>',
    grid: '<rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/>',
    check: '<path d="M20 6 9 17l-5-5"/>', checkCircle: '<circle cx="12" cy="12" r="9"/><path d="m8.5 12 2.5 2.5 4.5-5"/>',
    xCircle: '<circle cx="12" cy="12" r="9"/><path d="m15 9-6 6M9 9l6 6"/>', dashed: '<circle cx="12" cy="12" r="9" stroke-dasharray="3 3"/>',
    hourglass: '<path d="M6 2h12M6 22h12M7 2v4a5 5 0 0 0 10 0V2M7 22v-4a5 5 0 0 1 10 0v4"/>', layers: '<path d="m12 2 10 5-10 5L2 7z"/><path d="m2 17 10 5 10-5M2 12l10 5 10-5"/>',
    ban: '<circle cx="12" cy="12" r="9"/><path d="m5.6 5.6 12.8 12.8"/>',
    send: '<path d="m22 2-7 20-4-9-9-4z"/><path d="M22 2 11 13"/>', link: '<path d="M10 13a5 5 0 0 0 7.5.5l3-3a5 5 0 0 0-7-7l-1.7 1.7"/><path d="M14 11a5 5 0 0 0-7.5-.5l-3 3a5 5 0 0 0 7 7l1.7-1.7"/>',
    history: '<path d="M3 3v5h5"/><path d="M3.05 13A9 9 0 1 0 6 5.3L3 8"/><path d="M12 7v5l4 2"/>', edit: '<path d="M12 20h9"/><path d="M16.5 3.5a2.1 2.1 0 1 1 3 3L7 19l-4 1 1-4z"/>',
    filter: '<path d="M22 3H2l8 9.5V19l4 2v-8.5z"/>', sort: '<path d="M3 6h18M6 12h12M10 18h4"/>', upload: '<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M17 8l-5-5-5 5M12 3v12"/>',
    download: '<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M7 10l5 5 5-5M12 15V3"/>',
    search: '<circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/>', target: '<circle cx="12" cy="12" r="9"/><circle cx="12" cy="12" r="5"/><circle cx="12" cy="12" r="1"/>',
    sun: '<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/>',
    moon: '<path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z"/>', key: '<rect x="2" y="6" width="20" height="12" rx="2"/><path d="M6 10h.01M10 10h.01M14 10h.01M18 10h.01M8 14h8"/>',
    logout: '<path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9"/>', lock: '<rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>',
    rows: '<path d="M3 6h18M3 12h18M3 18h18"/>', arrow: '<path d="m15 18-6-6 6-6"/>', arrowL: '<path d="m15 18-6-6 6-6"/>',
    info: '<circle cx="12" cy="12" r="9"/><path d="M12 11v5M12 8h.01"/>', help: '<circle cx="12" cy="12" r="9"/><path d="M9.6 9.3a2.5 2.5 0 0 1 4.8.9c0 1.7-2.4 2.3-2.4 3.8"/><path d="M12 17h.01"/>',
    wifiOff: '<path d="M2 2l20 20M8.5 16.5a5 5 0 0 1 7 0M5 13a10 10 0 0 1 5.2-2.8M19 13a10 10 0 0 0-2.3-1.6M2 8.8a15 15 0 0 1 4.2-2.7M22 8.8a15 15 0 0 0-10.4-4.3M12 20h.01"/>', refresh: '<path d="M21 12a9 9 0 1 1-2.6-6.4L21 8"/><path d="M21 3v5h-5"/>',
    inbox: '<path d="M22 12h-6l-2 3h-4l-2-3H2"/><path d="M5.5 5.1 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.5-6.9A2 2 0 0 0 16.8 4H7.2a2 2 0 0 0-1.7 1.1z"/>',
    flag: '<path d="M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1zM4 22v-7"/>',
    chart: '<path d="M3 3v18h18"/><path d="M7 15v2M12 11v6M17 7v10"/>', shield: '<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>',
    question: '<circle cx="12" cy="12" r="9"/><path d="M9.6 9.3a2.5 2.5 0 0 1 4.8.9c0 1.7-2.4 2.3-2.4 3.8M12 17h.01"/>',
    eye: '<path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12z"/><circle cx="12" cy="12" r="3"/>',
    briefcase: '<rect x="2" y="7" width="20" height="14" rx="2"/><path d="M16 7V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v2"/>',
    compass: '<circle cx="12" cy="12" r="9"/><path d="m15.5 8.5-2 5-5 2 2-5z"/>',
    split: '<path d="M16 3h5v5M8 3H3v5M21 3l-7 7M3 3l7 7M12 22v-8"/>', unlink: '<path d="m18.8 11.2 1.4-1.4a5 5 0 0 0-7-7l-1.4 1.4M5.2 12.8l-1.4 1.4a5 5 0 0 0 7 7l1.4-1.4M8 2v3M2 8h3M16 22v-3M22 16h-3"/>'
  };
  var ic = function (n) { return '<svg class="i" viewBox="0 0 24 24" aria-hidden="true">' + (P[n] || '') + '</svg>'; };

  /* ---------- Shared building blocks (markup only) ---------- */
  var hint = function (text, below) { return '<button type="button" class="qhint tip' + (below ? ' below' : '') + '" data-tip="' + esc(text) + '" aria-label="' + esc(text) + '">' + ic('info') + '</button>'; };
  var pill = function (label, tone, icon) { return '<span class="pill t-' + tone + '">' + (icon ? ic(icon) : '') + esc(label) + '</span>'; };
  var qStyle = function (tone) { return ' style="--qc:var(--' + tone + '-dot);--qbg:var(--' + tone + '-bg);--qfg:var(--' + tone + '-fg)"'; };

  // Metric: label + value + meaning/time-basis tooltip; stale/unavailable never render as zero.
  function kpiHtml(k) {
    var tipText = [k.meaning, k.basis].filter(Boolean).join(' · ');
    var tag = k.filter ? 'button type="button" data-kpi-filter="' + esc(k.filter) + '"' : 'div';
    var val = k.state === 'unavailable' ? '<span class="na">—</span><small class="kstate">نامعتبر</small>' : k.state === 'undefined' ? '<span class="na">—</span><small class="kstate">تعریف نشده</small>' : k.value + (k.state === 'stale' ? '<small class="kstate">قدیمی</small>' : '');
    return '<' + tag + ' class="kpi' + (k.tone ? ' ' + k.tone : '') + (k.keep ? ' keep' : '') + (k.state ? ' st-' + k.state : '') + (tipText ? ' tip' : '') + '"' + (tipText ? ' data-tip="' + esc(tipText) + '"' : '') + ' style="--k:var(--' + (k.color || 'border') + '-dot)"><span>' + esc(k.label) + '</span><b>' + val + '</b></' + tag.split(' ')[0] + '>';
  }
  function freshness(f) {
    if (!f) return '';
    return '<span class="fresh' + (f.stale ? ' stale' : '') + '" role="status">' + ic(f.stale ? 'alert' : 'clock') + '<span>' + esc(f.text) + '</span>' + (f.refresh !== false ? '<button type="button" class="btn btn-ghost btn-sm" data-refresh>' + ic('refresh') + 'بازخوانی</button>' : '') + '</span>';
  }
  function pageHead(o) {
    var tour = C.cfg && C.cfg.pageTours && C.cfg.pageTours[C.state.view];
    return '<div class="page-head"><div><h1>' + esc(o.title) + (tour ? '<button type="button" class="page-help tip below" data-tip="راهنمای این صفحه" aria-label="راهنمای این صفحه" data-page-tour="' + tour + '">' + ic('help') + '</button>' : '') + '</h1>' + (o.sub ? '<div class="sub">' + o.sub + '</div>' : '') + '</div><div class="spacer"></div>' +
      (o.kpis ? '<div class="kpis" role="group" aria-label="شاخص‌های این صفحه">' + o.kpis.map(kpiHtml).join('') + '</div>' : '') +
      '<div class="head-cta">' + (o.cta || '') + '<button type="button" class="btn focus-btn tip" data-tip="عناصر فرعی کم‌رنگ می‌شوند" aria-pressed="' + prefs.focus + '" data-toggle-focus>' + ic('target') + 'حالت تمرکز</button></div></div>' +
      (o.fresh ? '<div class="head-meta">' + freshness(o.fresh) + (o.scope ? '<span class="scope-note">' + o.scope + '</span>' : '') + '</div>' : '');
  }
  function queues(items, active, attr) {
    return '<div class="queues" role="group" aria-label="صف‌ها">' + items.map(function (q) {
      return '<button type="button" class="q" ' + attr + '="' + q.id + '" aria-pressed="' + (q.id === active) + '"' + (q.tone ? qStyle(q.tone) : '') + (q.key ? ' data-tip="کلید ' + fa(q.key) + '" aria-keyshortcuts="' + q.key + '"' : '') + '>' +
        (q.icon ? '<span class="ic">' + ic(q.icon) + '</span>' : '') + esc(q.label) + (q.n != null ? '<span class="n">' + fa(q.n) + '</span>' : '') + '</button>';
    }).join('') + '</div>';
  }
  function toolbar(placeholder, chips, right) {
    return '<div class="toolbar"><label class="field"><span class="sr">جستجو در این صف</span>' + ic('filter') +
      '<input class="input" type="search" id="queue-filter" placeholder="' + esc(placeholder) + '" aria-keyshortcuts="F"></label>' +
      (chips || []).map(function (c) { return '<button type="button" class="chip-btn' + (c.on ? ' on' : '') + '" aria-label="' + esc(c.label) + '" data-demo="فیلتر «' + esc(c.label) + '» در نسخه واقعی یک منوی انتخاب باز می‌کند">' + ic(c.icon || 'plus') + '<span>' + esc(c.label) + '</span></button>'; }).join('') +
      '<span class="grow"></span>' + (right || '') + '</div>';
  }
  function tfoot(shown, total) {
    return '<div class="tfoot"><span>نمایش <b class="num">' + fa('1–' + shown) + '</b> از <b class="num">' + fa(total) + '</b></span>' +
      '<div class="pager"><button type="button" aria-label="صفحه قبلی">›</button><button type="button" aria-current="page">۱</button><button type="button" aria-label="صفحه بعدی">‹</button></div><span class="desk">۲۰ در صفحه</span></div>';
  }
  // States are distinct on purpose: error ≠ empty ≠ unauthorized ≠ stale ≠ conflict ≠ partial ≠ unknown.
  var STATE_ICON = { error: 'alert', empty: 'check', noresult: 'search', info: 'info', locked: 'lock', stale: 'clock', incomplete: 'layers', conflict: 'swap', partial: 'split', retry: 'refresh', unknown: 'question' };
  function stateBlock(kind, title, text, actions) {
    return '<div class="state state-' + kind + '" role="' + (kind === 'error' || kind === 'conflict' ? 'alert' : 'status') + '"><div class="state-ico">' + ic(STATE_ICON[kind] || 'info') + '</div><b>' + esc(title) + '</b>' + (text ? '<span>' + text + '</span>' : '') + (actions ? '<div class="state-acts">' + actions + '</div>' : '') + '</div>';
  }
  function banner(kind, html, actions) { return '<div class="ws-banner b-' + kind + '" role="status">' + ic(kind === 'stale' ? 'clock' : kind === 'incomplete' ? 'layers' : kind === 'locked' ? 'lock' : 'info') + '<span>' + html + '</span>' + (actions || '') + '</div>'; }
  function cbx(attrs, checked, label, disabled) { return '<input type="checkbox" class="cbx" ' + attrs + (checked ? ' checked' : '') + (disabled ? ' disabled' : '') + ' aria-label="' + esc(label) + '">'; }

  /* ---------- State ---------- */
  var qs = new URLSearchParams(location.search);
  var state = { view: null, drawer: null, sim: qs.get('sim') || null, offline: qs.get('sim') === 'offline', cursor: 0, q: {} };
  var prefs = { theme: root.getAttribute('data-theme'), density: root.getAttribute('data-density'), focus: root.getAttribute('data-focus') === '1' };
  function setPref(k, v) {
    prefs[k] = v;
    if (k === 'focus') { if (v) root.setAttribute('data-focus', '1'); else root.removeAttribute('data-focus'); store.set('focus', v ? '1' : '0'); }
    else { root.setAttribute('data-' + k, v); store.set(k, v); }
    $$('.focus-btn').forEach(function (fb) { fb.setAttribute('aria-pressed', String(prefs.focus)); });
    fitNav();
  }

  var C = { state: state, prefs: prefs, cfg: null };

  /* ---------- Shell: role navigation with groups + priority overflow («بیشتر») ---------- */
  function navItems() { return C.cfg.nav().filter(function (n) { return !n.hidden; }); }
  function renderNav() {
    var items = navItems(), last = null;
    $('#nav').innerHTML = items.map(function (n) {
      var sep = last && n.group !== last ? '<span class="nav-sep" aria-hidden="true" data-sep></span>' : ''; last = n.group;
      var c = n.count ? n.count() : null;
      return sep + '<a href="#' + n.id + '" data-view="' + n.id + '"' + (state.view === n.id ? ' aria-current="page"' : '') + '>' + ic(n.icon) + esc(n.label) + (c ? '<span class="count' + (n.alert ? ' alert' : '') + '">' + fa(c) + '</span>' : '') + '</a>';
    }).join('') + '<button type="button" class="nav-more" id="nav-more" hidden aria-haspopup="menu" aria-expanded="false">' + ic('more') + 'بیشتر<span class="count" id="nav-more-n"></span></button><span class="nav-probe" id="nav-probe" aria-hidden="true">نمای تیم · فاکتورها</span>';
    // The probe's width changes only when the web font swaps in; re-measure overflow then.
    if (window.ResizeObserver) { if (!C.navRO) C.navRO = new ResizeObserver(function () { fitNav(); }); C.navRO.disconnect(); C.navRO.observe($('#nav-probe')); }
    var mob = items.filter(function (n) { return n.mobile; });
    var moreIds = items.filter(function (n) { return !n.mobile; }).map(function (n) { return n.id; });
    $('#bottom-nav').innerHTML = mob.map(function (n) {
      var c = n.alert && n.count ? n.count() : 0;
      return '<a href="#' + n.id + '" data-view="' + n.id + '"' + (state.view === n.id ? ' aria-current="page"' : '') + '>' + ic(n.icon) + (c ? '<span class="b">' + fa(c) + '</span>' : '') + esc(n.mobile) + '</a>';
    }).join('') + '<a href="#more" data-more="nav-mobile"' + (moreIds.indexOf(state.view) > -1 ? ' aria-current="page"' : '') + '>' + ic('grid') + 'بیشتر</a>';
    var cur = items.filter(function (n) { return n.id === state.view; })[0];
    $('#crumb').textContent = cur ? cur.label : '';
    $('#m-title').textContent = cur ? cur.label : '';
    document.title = (cur ? cur.label + ' — ' : '') + C.cfg.titleSuffix;
    if (C.cfg.workload) $('#workload').innerHTML = C.cfg.workload();
    fitNav();
  }
  // Priority+ overflow: trailing tabs move into «بیشتر» when the row does not fit; the current tab always stays visible.
  function fitNav() {
    var nav = $('#nav'); if (!nav || !C.cfg) return;
    var links = $$('a[data-view]', nav), seps = $$('[data-sep]', nav), more = $('#nav-more');
    links.forEach(function (a) { a.hidden = false; }); seps.forEach(function (s) { s.hidden = false; }); more.hidden = true;
    if (!nav.clientWidth || nav.scrollWidth <= nav.clientWidth + 1) return;
    more.hidden = false;
    for (var i = links.length - 1; i >= 0 && nav.scrollWidth > nav.clientWidth + 1; i--) {
      if (links[i].getAttribute('aria-current') === 'page') continue;
      links[i].hidden = true;
      var prev = links[i].previousElementSibling; if (prev && prev.hasAttribute('data-sep')) prev.hidden = true;
    }
    var hiddenN = links.filter(function (a) { return a.hidden; });
    $('#nav-more-n').textContent = fa(hiddenN.length);
  }
  function openNavMore(anchor) {
    var items = $$('#nav a[data-view]').filter(function (a) { return a.hidden; }).map(function (a) {
      var n = navItems().filter(function (x) { return x.id === a.getAttribute('data-view'); })[0];
      return { label: n.label, icon: n.icon, go: n.id };
    });
    openMenu(anchor, items); anchor.setAttribute('aria-expanded', 'true');
  }

  /* ---------- Drawer (shared detail drawer / mobile sheet) ---------- */
  var lastOpener = null;
  function openDrawer(kind, id, keep) {
    var same = state.drawer && state.drawer.kind === kind && String(state.drawer.id) === String(id);
    if (!state.drawer) lastOpener = document.activeElement;
    state.drawer = { kind: kind, id: id, keep: same ? (state.drawer.keep || {}) : (keep || {}) };
    var el = $('#drawer'), html = C.cfg.drawers[kind] ? C.cfg.drawers[kind](id, state.drawer.keep) : '';
    if (state.sim === 'drawerError') html = drawerState('error');
    else if (state.sim === 'loading') html = drawerState('loading');
    el.className = 'drawer' + (C.cfg.wideDrawers && C.cfg.wideDrawers.indexOf(kind) > -1 ? ' wide' : '');
    el.innerHTML = html; el.setAttribute('aria-hidden', 'false');
    document.body.classList.add('drawer-open');
    if (document.body.classList.contains('kb') && !el.contains(document.activeElement)) { el.tabIndex = -1; el.focus({ preventScroll: true }); }
    markRows();
  }
  function markRows() {
    var key = state.drawer ? state.drawer.kind + ':' + state.drawer.id : null;
    $$('#ws .tbl tbody tr[data-row]').forEach(function (tr) { var on = tr.getAttribute('data-row') === key; tr.classList.toggle('sel', on); tr.setAttribute('aria-selected', String(on)); });
  }
  function drawerState(kind) {
    var top = '<div class="sheet-grip" aria-hidden="true"></div><div class="dr-top"><span>جزئیات</span><span class="grow"></span><button type="button" class="btn btn-ghost btn-icon btn-sm tip" data-tip="بستن (Esc)" data-close aria-label="بستن">' + ic('x') + '</button></div>';
    if (kind === 'loading') return top + '<div class="dr-head" aria-busy="true"><div class="sk sk-line" style="width:40%;height:18px"></div><div class="sk sk-line" style="width:65%;margin-top:12px"></div><div class="sk-row" style="margin-top:12px"><span class="sk sk-chip"></span><span class="sk sk-chip"></span><span class="sk sk-chip"></span></div></div><div class="dr-body"><div class="sec"><div class="sk sk-line" style="width:30%"></div><div class="sk-grid"><span class="sk sk-box"></span><span class="sk sk-box"></span><span class="sk sk-box"></span><span class="sk sk-box"></span></div></div></div><span class="sr" role="status">در حال بارگذاری جزئیات</span>';
    return top + '<div class="dr-body">' + stateBlock('error', 'جزئیات بارگذاری نشد', 'اتصال یا سرور پاسخ نداد. پنجره باز می‌ماند تا دوباره تلاش کنید.', '<button type="button" class="btn btn-primary" data-retry="drawer">' + ic('refresh') + 'تلاش مجدد</button>') + '</div>';
  }
  function closeDrawer() {
    state.drawer = null;
    document.body.classList.remove('drawer-open'); $('#drawer').setAttribute('aria-hidden', 'true');
    markRows();
    var row = $('#ws .tbl tbody tr.cursor'); if (row) row.focus(); else if (lastOpener && lastOpener.focus && document.contains(lastOpener)) lastOpener.focus();
  }
  function rerenderDrawer() { if (state.drawer) openDrawer(state.drawer.kind, state.drawer.id); }

  /* ---------- Row cursor (J/K) ---------- */
  function rows() { return $$('#ws .tbl:not(.no-cursor) tbody tr[data-row]').filter(function (r) { return r.getClientRects().length; }); }
  function setCursor(i, focus) {
    var rs = rows(); if (!rs.length) return;
    state.cursor = Math.max(0, Math.min(rs.length - 1, i));
    rs.forEach(function (r, n) { r.classList.toggle('cursor', n === state.cursor); r.tabIndex = n === state.cursor ? 0 : -1; });
    if (focus !== false) { rs[state.cursor].focus({ preventScroll: true }); rs[state.cursor].scrollIntoView({ block: 'nearest' }); }
  }
  function openRow(tr) { var p = tr.getAttribute('data-row').split(':'); openDrawer(p[0], p.slice(1).join(':')); }

  /* ---------- Toast, menus, user menu ---------- */
  function toast(msg, type, persist) {
    type = type || 'info';
    var box = $('#toasts'), t = document.createElement('div');
    t.className = 'toast t-' + type; t.setAttribute('role', type === 'error' ? 'alert' : 'status');
    t.innerHTML = '<span class="t-ico">' + ic({ success: 'checkCircle', error: 'alert', warning: 'alert', info: 'info' }[type]) + '</span><span class="t-msg">' + esc(msg) + '</span><button type="button" class="t-x" aria-label="بستن اعلان">' + ic('x') + '</button>';
    box.appendChild(t); while (box.children.length > 3) box.removeChild(box.firstChild);
    var kill = function () { t.classList.add('out'); setTimeout(function () { t.remove(); }, 200); };
    var timer = persist ? 0 : setTimeout(kill, 4500);
    t.addEventListener('mouseenter', function () { clearTimeout(timer); });
    t.addEventListener('mouseleave', function () { if (!persist) timer = setTimeout(kill, 2500); });
    t.querySelector('.t-x').addEventListener('click', kill);
  }
  function closeMenu() { var m = $('.menu'); if (m) { m.remove(); $('#user-btn').setAttribute('aria-expanded', 'false'); var nm = $('#nav-more'); if (nm) nm.setAttribute('aria-expanded', 'false'); } }
  function placeMenu(m, anchor) {
    var r = anchor.getBoundingClientRect(), top = r.bottom + 6;
    if (top + m.offsetHeight > innerHeight - 8) top = Math.max(8, r.top - m.offsetHeight - 6);
    m.style.top = top + 'px'; m.style.left = Math.max(8, Math.min(r.left, innerWidth - m.offsetWidth - 8)) + 'px';
  }
  function menuItem(it) {
    if (it === '-') return '<hr>';
    if (it.head) return '<div class="mh">' + esc(it.head) + '</div>';
    var dis = it.disabled ? ' aria-disabled="true" data-tip="' + esc(it.why || '') + '"' : '';
    return '<button type="button" class="mi' + (it.danger ? ' danger' : '') + (it.disabled ? ' off' : '') + '" role="menuitem"' + dis + (it.go ? ' data-go="' + it.go + '"' : it.act ? ' data-act="' + esc(it.act) + '"' : ' data-demo="' + esc(it.demo || ('نمایشی: «' + it.label + '»')) + '"') + '>' + ic(it.icon) + esc(it.label) + (it.badge ? '<span class="mi-b">' + esc(it.badge) + '</span>' : '') + '</button>';
  }
  function openMenu(anchor, items, id) {
    closeMenu();
    var m = document.createElement('div'); m.className = 'menu'; m.setAttribute('role', 'menu'); if (id) m.id = id;
    m.innerHTML = items.map(menuItem).join('');
    document.body.appendChild(m); placeMenu(m, anchor);
    var first = m.querySelector('button:not([aria-disabled])'); if (first) first.focus();
    return m;
  }
  function userMenuHtml() {
    var seg = function (key, opts) { return '<div class="opt-seg" role="group">' + opts.map(function (o) { return '<button type="button" data-pref="' + key + '" data-val="' + o[0] + '" aria-pressed="' + (prefs[key] === o[0]) + '">' + o[1] + '</button>'; }).join('') + '</div>'; };
    return (C.cfg.userMenuTop ? C.cfg.userMenuTop().map(menuItem).join('') + '<hr>' : '') +
      '<div class="mh">پوسته</div>' + seg('theme', [['light', 'روشن'], ['dim', 'کم‌نور'], ['dark', 'تیره']]) +
      '<div class="mh">تراکم فهرست‌ها</div>' + seg('density', [['comfortable', 'راحت'], ['compact', 'فشرده']]) +
      '<button type="button" class="switch-row" role="menuitemcheckbox" aria-checked="' + prefs.focus + '" data-toggle-focus>' + ic('target') + 'حالت تمرکز<span class="switch"></span></button>' +
      '<button type="button" class="mi" data-open-shortcuts>' + ic('key') + 'میانبرهای صفحه‌کلید<span class="kbd" style="margin-right:auto">?</span></button>' +
      '<button type="button" class="mi" data-help-open>' + ic('help') + 'مرکز راهنما</button>' +
      '<button type="button" class="mi" data-demo-panel>' + ic('grid') + 'حالت‌های نمایشی (دمو)</button><hr>' +
      '<button type="button" class="mi" data-demo="نمایشی: تغییر رمز">' + ic('lock') + 'رمز من</button><button type="button" class="mi danger" data-demo="نمایشی: خروج انجام نشد">' + ic('logout') + 'خروج</button>';
  }
  function openUserMenu() {
    closeMenu();
    var b = $('#user-btn'), m = document.createElement('div');
    m.className = 'menu'; m.id = 'user-menu'; m.setAttribute('role', 'menu'); m.style.minWidth = '250px';
    m.innerHTML = userMenuHtml(); document.body.appendChild(m); placeMenu(m, b);
    b.setAttribute('aria-expanded', 'true');
    var f = m.querySelector('[aria-pressed="true"]'); if (f) f.focus();
  }

  /* ---------- Command palette (role supplies records; navigation/display are shared) ---------- */
  var pal = { items: [], sel: 0 };
  var norm = function (s) { return String(s).replace(/[۰-۹]/g, function (d) { return '۰۱۲۳۴۵۶۷۸۹'.indexOf(d); }); };
  function palItems(q) {
    var nq = norm((q || '').trim()), out = [];
    var match = function (s) { return !nq || norm(s).indexOf(nq) > -1; };
    if (C.cfg.palette) out = out.concat(C.cfg.palette(nq, match));
    navItems().forEach(function (n) { if (match('رفتن به ' + n.label)) out.push({ g: 'رفتن به', icon: n.icon, label: n.label, run: function () { go(n.id); } }); });
    [['light', 'پوسته روشن', 'sun'], ['dim', 'پوسته کم‌نور', 'moon'], ['dark', 'پوسته تیره', 'moon']].forEach(function (t) { if (match(t[1]) || match('تغییر پوسته')) out.push({ g: 'نمایش', icon: t[2], label: t[1], meta: prefs.theme === t[0] ? 'فعال' : '', run: function () { setPref('theme', t[0]); } }); });
    if (match('حالت تمرکز')) out.push({ g: 'نمایش', icon: 'target', label: prefs.focus ? 'خاموش کردن حالت تمرکز' : 'روشن کردن حالت تمرکز', run: function () { setPref('focus', !prefs.focus); } });
    if (match('تراکم فشرده راحت')) out.push({ g: 'نمایش', icon: 'rows', label: prefs.density === 'compact' ? 'تراکم راحت' : 'تراکم فشرده', run: function () { setPref('density', prefs.density === 'compact' ? 'comfortable' : 'compact'); } });
    return out.slice(0, 40);
  }
  function renderPalette(q) {
    pal.items = palItems(q); pal.sel = Math.min(pal.sel, Math.max(0, pal.items.length - 1));
    var html = '', g = null;
    pal.items.forEach(function (it, n) {
      if (it.g !== g) { g = it.g; html += '<div class="pal-group">' + esc(g) + '</div>'; }
      html += '<button type="button" class="pal-item" role="option" data-pal="' + n + '" aria-selected="' + (n === pal.sel) + '">' + ic(it.icon) + '<span class="grow">' + esc(it.label) + '</span>' + (it.meta ? '<small>' + esc(it.meta) + '</small>' : '') + '</button>';
    });
    $('.pal-list').innerHTML = html || '<div class="pal-empty">نتیجه‌ای پیدا نشد</div>';
    var s = $('.pal-item[aria-selected="true"]'); if (s) s.scrollIntoView({ block: 'nearest' });
  }
  function openPalette(q) {
    closeMenu();
    var w = $('#palette'); w.hidden = false; pal.sel = 0; lastOpener = document.activeElement;
    $('.palette').innerHTML = '<div class="pal-in">' + ic('search') + '<input type="text" id="pal-q" placeholder="' + esc(C.cfg.palettePlaceholder || 'جستجو یا یک فرمان…') + '" aria-label="جستجو و فرمان‌ها" autocomplete="off"><span class="kbd">Esc</span></div><div class="pal-list" role="listbox"></div>' +
      '<div class="pal-foot"><span><span class="kbd">↑↓</span> انتخاب</span><span><span class="kbd">Enter</span> اجرا</span><span><span class="kbd">/</span> یا <span class="kbd">Ctrl K</span> باز کردن</span><span><span class="kbd">J</span><span class="kbd">K</span> ردیف بعدی/قبلی</span><span><span class="kbd">۱–۹</span> تغییر صف</span><span><span class="kbd">F</span> جستجو در صف</span></div>';
    var input = $('#pal-q'); input.value = q || ''; renderPalette(input.value); input.focus();
  }
  function closePalette() { $('#palette').hidden = true; if (lastOpener && lastOpener.focus && document.contains(lastOpener)) lastOpener.focus(); }
  function runPal(n) { var it = pal.items[n]; if (!it) return; closePalette(); it.run(); }

  /* ---------- Page states ---------- */
  function skeletonPage() {
    var r = '';
    for (var i = 0; i < 7; i++) r += '<div class="sk-trow"><span class="sk sk-pill"></span><span class="sk sk-line" style="width:18%"></span><span class="sk sk-line" style="width:8%"></span><span class="sk sk-line" style="width:10%"></span><span class="sk sk-line" style="width:14%"></span><span class="grow"></span><span class="sk sk-btn"></span></div>';
    var kpi = '<div class="kpi"><span class="sk sk-line" style="width:70px"></span><span class="sk sk-line" style="width:36px;height:16px;margin-top:8px"></span></div>';
    return '<div class="page-head" aria-busy="true"><div><div class="sk sk-line" style="width:140px;height:20px"></div></div><div class="spacer"></div><div class="kpis sk-kpis">' + kpi + kpi + kpi + kpi + '</div></div>' +
      '<section class="panel" aria-busy="true"><div class="queues"><span class="sk sk-q"></span><span class="sk sk-q"></span><span class="sk sk-q"></span><span class="sk sk-q"></span></div><div class="toolbar"><span class="sk sk-input"></span></div>' + r + '</section><span class="sr" role="status">در حال بارگذاری</span>';
  }
  function applyTableState() {
    var panel = $('#ws .panel.main') || $('#ws .panel'); if (!panel) return;
    var area = panel.querySelector('.tbl-wrap') || panel.querySelector('.empty'); if (!area) return;
    var html = '', E = (C.cfg.empty || {})[state.view] || ['موردی برای نمایش وجود ندارد', '', ''];
    if (state.sim === 'tableError') html = stateBlock('error', 'امکان دریافت لیست وجود ندارد', 'ممکن است اتصال لحظه‌ای قطع شده باشد. بقیه صفحه همچنان قابل استفاده است.', '<button type="button" class="btn btn-primary" data-retry="table">' + ic('refresh') + 'تلاش مجدد</button>');
    else if (state.sim === 'empty') html = stateBlock('empty', E[0], E[1], E[2]);
    else if (state.sim === 'noresult') {
      var f = $('#queue-filter'); if (f) f.value = fa('09123456');
      html = stateBlock('noresult', 'نتیجه‌ای برای «' + fa('09123456') + '» پیدا نشد', 'عبارت را کامل‌تر وارد کنید یا فیلترها را پاک کنید.', '<button type="button" class="btn btn-soft" data-clear-search>پاک کردن جستجو</button><button type="button" class="btn" data-clear-filters>پاک کردن فیلترها</button>');
    } else return;
    $$('.card-list,.tfoot,.bulkbar', panel).forEach(function (n) { n.remove(); });
    var d = document.createElement('div'); d.innerHTML = html; area.replaceWith(d.firstChild);
  }
  function render() {
    renderNav();
    $('#offline-bar').hidden = !state.offline; $('#net-chip').hidden = !state.offline;
    if (state.offline) $('#offline-bar').innerHTML = ic('wifiOff') + '<span><b>اتصال اینترنت قطع شده است.</b> اطلاعات نمایش‌داده‌شده ممکن است به‌روز نباشند؛ هیچ عملیاتی تا اتصال دوباره ثبت نمی‌شود.</span><button type="button" class="btn btn-sm" data-retry="net">' + ic('refresh') + 'تلاش مجدد</button>';
    var ws = $('#ws');
    if (state.sim === 'pageError') { ws.innerHTML = '<section class="panel page-error">' + stateBlock('error', 'خطا در دریافت اطلاعات', 'پاسخی از سرور دریافت نشد. چند لحظه بعد دوباره تلاش کنید؛ داده‌ای از دست نرفته است.', '<button type="button" class="btn btn-primary" data-retry="page">' + ic('refresh') + 'تلاش مجدد</button>') + '</section>'; return; }
    if (state.sim === 'unauthorized') { ws.innerHTML = '<section class="panel page-error">' + stateBlock('locked', 'دسترسی به این بخش برای نقش شما فعال نیست', 'این صفحه برای حساب شما قابل مشاهده نیست. نمایش‌ندادن داده به معنی نبودن داده نیست؛ اگر برای کارتان لازم است، از مدیر مستقیم پیگیری کنید.', '<button type="button" class="btn" data-go="' + C.cfg.home + '">' + ic('arrowL') + 'بازگشت به ' + esc(navItems()[0].label) + '</button>') + '</section>'; return; }
    if (state.sim === 'loading') { ws.innerHTML = skeletonPage(); return; }
    ws.innerHTML = (C.cfg.views[state.view] || C.cfg.views[C.cfg.home])();
    applyTableState();
    // Non-card tables stack into labelled rows on phones.
    $$('#ws .panel:not(.has-cards) .tbl').forEach(function (t) {
      var heads = $$('thead th', t).map(function (h) { return h.getAttribute('data-l') != null ? h.getAttribute('data-l') : h.textContent.trim(); });
      $$('tbody tr', t).forEach(function (r) { Array.prototype.forEach.call(r.children, function (td, i) { if (heads[i] && !td.hasAttribute('data-l')) td.setAttribute('data-l', heads[i]); }); });
      t.classList.add('stackable');
    });
    setCursor(state.cursor, false);
    if (C.cfg.afterRender) C.cfg.afterRender();
    if (state.drawer) openDrawer(state.drawer.kind, state.drawer.id);
  }
  function go(view) { if (state.drawer) closeDrawer(); closeMenu(); state.view = view; state.cursor = 0; var s = location.search.replace(/([?&])view=[^&]*/, '$1view=' + view); history.replaceState(null, '', location.pathname + s + '#' + view); render(); window.scrollTo(0, 0); }

  /* ---------- Events ---------- */
  document.addEventListener('click', function (e) {
    var t = e.target.closest('button, a, tr, .lcard, input, label, [data-palette-close]');
    if (!t) { closeMenu(); return; }
    if (t.closest('.palette-wrap')) {
      if (t.matches('[data-palette-close]')) return closePalette();
      if (t.matches('[data-pal]')) return runPal(Number(t.getAttribute('data-pal')));
      return;
    }
    if (t.matches('[aria-disabled="true"]')) { e.preventDefault(); return; }
    if (t.matches('[data-pref]')) { setPref(t.getAttribute('data-pref'), t.getAttribute('data-val')); var m = $('#user-menu'); if (m) { m.innerHTML = userMenuHtml(); var f = m.querySelector('[data-pref="' + t.getAttribute('data-pref') + '"][aria-pressed="true"]'); if (f) f.focus(); } return; }
    if (t.matches('[data-toggle-focus]')) { setPref('focus', !prefs.focus); var um = $('#user-menu'); if (um) um.innerHTML = userMenuHtml(); return; }
    if (t.matches('[data-open-shortcuts]')) { closeMenu(); if (window.SNHelp) window.SNHelp.openHelp('keys'); return; }
    if (!t.closest('.menu') && !t.matches('[data-more]') && !t.matches('#user-btn') && !t.matches('#nav-more')) closeMenu();
    if (t.matches('#user-btn')) { e.stopPropagation(); if ($('#user-menu')) closeMenu(); else openUserMenu(); return; }
    if (t.matches('#nav-more')) { e.stopPropagation(); if ($('.menu')) closeMenu(); else openNavMore(t); return; }
    if (t.matches('[data-palette]')) return openPalette('');
    if (t.matches('[data-view]')) { e.preventDefault(); return go(t.getAttribute('data-view')); }
    if (t.matches('[data-go]')) { e.preventDefault(); return go(t.getAttribute('data-go')); }
    if (t.matches('[data-more="nav-mobile"]')) {
      e.preventDefault(); e.stopPropagation();
      var mm = openMenu(t, navItems().filter(function (n) { return !n.mobile; }).map(function (n) { return { label: n.label, icon: n.icon, go: n.id }; }).concat(['-']), 'user-menu');
      mm.insertAdjacentHTML('beforeend', userMenuHtml()); return;
    }
    if (t.matches('[data-retry]')) {
      var r = t.getAttribute('data-retry');
      if (r === 'net') { state.offline = false; state.sim = null; render(); return toast('اتصال برقرار شد؛ اطلاعات به‌روز شد.', 'success'); }
      if (r === 'drawer') { state.sim = null; return rerenderDrawer(); }
      state.sim = 'loading'; render(); setTimeout(function () { state.sim = null; render(); toast('اطلاعات دوباره بارگذاری شد.', 'success'); }, 900); return;
    }
    if (t.matches('[data-refresh]')) { state.sim = 'loading'; var keepDr = state.drawer; render(); setTimeout(function () { state.sim = null; state.fresh = 'now'; render(); if (keepDr) openDrawer(keepDr.kind, keepDr.id); toast('اطلاعات این صفحه بازخوانی شد.', 'success'); }, 700); return; }
    if (t.matches('[data-clear-search],[data-clear-filters]')) { state.sim = null; render(); return toast(t.matches('[data-clear-search]') ? 'جستجو پاک شد.' : 'فیلترها پاک شدند.', 'info'); }
    if (t.matches('#net-chip')) { state.offline = false; state.sim = null; render(); return toast('اتصال برقرار شد؛ اطلاعات به‌روز شد.', 'success'); }
    if (C.cfg.onClick && C.cfg.onClick(t, e) === true) return;
    if (t.matches('[data-copy]')) { e.stopPropagation(); return toast('نمایشی: ' + fa(t.getAttribute('data-copy')) + ' کپی شد'); }
    if (t.matches('[data-demo]')) { e.preventDefault(); e.stopPropagation(); toast(t.getAttribute('data-demo')); return closeMenu(); }
    if (t.matches('[data-close]')) return closeDrawer();
    if (t.matches('tr[data-row], .lcard[data-row]') && !e.target.closest('input,button,a,label')) return openRow(t);
  });
  $('#scrim').addEventListener('click', closeDrawer);
  $('#palette').addEventListener('input', function (e) { if (e.target.id === 'pal-q') { pal.sel = 0; renderPalette(e.target.value); } });

  // Keyboard: same architecture and guards as Seller V1. Letter/number shortcuts never fire inside controls.
  document.addEventListener('mousedown', function () { document.body.classList.remove('kb'); });
  document.addEventListener('keydown', function (e) {
    if (/^(Tab|Arrow|j|k|J|K|ت|ن|Enter)/.test(e.key)) document.body.classList.add('kb');
    var a = document.activeElement || {}, typing = /INPUT|TEXTAREA|SELECT/.test(a.tagName || '') || a.isContentEditable;
    var onControl = typing || !!(a.closest && a.closest('button, a[href], [role="button"], [role="menuitem"], [role="option"], summary, label'));
    if (!$('#palette').hidden) {
      if (e.key === 'Escape') { e.preventDefault(); return closePalette(); }
      if (e.key === 'ArrowDown' || e.key === 'ArrowUp') { e.preventDefault(); pal.sel = (pal.sel + (e.key === 'ArrowDown' ? 1 : -1) + pal.items.length) % Math.max(1, pal.items.length); return renderPalette($('#pal-q').value); }
      if (e.key === 'Enter') { e.preventDefault(); return runPal(pal.sel); }
      if (e.key === 'Tab') { e.preventDefault(); $('#pal-q').focus(); }
      return;
    }
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') { e.preventDefault(); return openPalette(''); }
    if (e.key === 'Escape') { if ($('.menu')) { closeMenu(); var ub = $('#user-btn'); if (ub) ub.focus(); return; } if (state.drawer) return closeDrawer(); if (typing) a.blur(); return; }
    if (e.ctrlKey || e.metaKey || e.altKey) return;
    if (e.key === '?' && !typing) { e.preventDefault(); if (window.SNHelp) window.SNHelp.openHelp('keys'); return; }
    if (onControl) return;
    var k = e.key.toLowerCase(), faDigit = '۰۱۲۳۴۵۶۷۸۹'.indexOf(e.key); if (faDigit > -1) k = String(faDigit);
    if (k === '/') { e.preventDefault(); return openPalette(''); }
    if (k === 'f' || e.key === 'ب') { var qf = $('#queue-filter'); if (qf) { e.preventDefault(); qf.focus(); } return; }
    if (k === 'j' || k === 'ت') { e.preventDefault(); return setCursor(state.cursor + 1); }
    if (k === 'k' || k === 'ن') { e.preventDefault(); return setCursor(state.cursor - 1); }
    if (/^[0-9]$/.test(k) && !state.drawer) { var qb = $('#ws .queues .q[aria-keyshortcuts="' + k + '"]'); if (qb) { e.preventDefault(); qb.click(); state.cursor = 0; setCursor(0); } return; }
    if (e.key === 'Enter' && !state.drawer && a.matches && a.matches('tr[data-row]')) { e.preventDefault(); openRow(a); }
  });
  window.addEventListener('hashchange', function () { var v = location.hash.slice(1); if (C.cfg.views[v] && v !== state.view) go(v); });
  var rz = 0; window.addEventListener('resize', function () { clearTimeout(rz); rz = setTimeout(fitNav, 80); });
  // Tab widths change once the web font arrives; measure again so overflow is never computed on fallback metrics.
  if (document.fonts) { if (document.fonts.ready) document.fonts.ready.then(function () { fitNav(); }); if (document.fonts.addEventListener) document.fonts.addEventListener('loadingdone', function () { fitNav(); }); }
  setTimeout(fitNav, 700);

  /* ---------- Tooltip (shared): one fixed layer for every [data-tip] ---------- */
  var tipEl = null, tipFor = null, tipTimer = 0;
  function hideTip() { clearTimeout(tipTimer); if (tipEl) { tipEl.remove(); tipEl = null; } if (tipFor) { if (tipFor.getAttribute('aria-describedby') === 'sn-tip') tipFor.removeAttribute('aria-describedby'); tipFor = null; } }
  function showTip(el) {
    hideTip();
    var text = el.getAttribute('data-tip'); if (!text || !document.contains(el)) return;
    tipFor = el; tipEl = document.createElement('div'); tipEl.className = 'tooltip'; tipEl.id = 'sn-tip'; tipEl.setAttribute('role', 'tooltip'); tipEl.textContent = text;
    document.body.appendChild(tipEl);
    if (el.getAttribute('aria-label') !== text && !el.hasAttribute('aria-describedby')) el.setAttribute('aria-describedby', 'sn-tip');
    var r = el.getBoundingClientRect(), w = tipEl.offsetWidth, h = tipEl.offsetHeight, top = r.top - h - 8;
    if (top < 8) top = r.bottom + 8;
    tipEl.style.top = Math.round(top) + 'px'; tipEl.style.left = Math.round(Math.max(8, Math.min(innerWidth - w - 8, r.left + r.width / 2 - w / 2))) + 'px';
  }
  document.addEventListener('mouseover', function (e) { var el = e.target.closest && e.target.closest('[data-tip]'); if (!el || el === tipFor) return; hideTip(); tipTimer = setTimeout(function () { showTip(el); }, 400); });
  document.addEventListener('mouseout', function (e) { var el = e.target.closest && e.target.closest('[data-tip]'); if (el && !el.contains(e.relatedTarget)) hideTip(); });
  document.addEventListener('focusin', function (e) { var el = e.target.closest && e.target.closest('[data-tip]'); if (el && document.body.classList.contains('kb')) showTip(el); else hideTip(); });
  document.addEventListener('focusout', hideTip);
  document.addEventListener('mousedown', hideTip, true);
  window.addEventListener('scroll', hideTip, true); window.addEventListener('resize', hideTip);
  document.addEventListener('keydown', function (e) { if (e.key === 'Escape' && tipEl) hideTip(); }, true);

  /* ---------- Public API ---------- */
  function boot(cfg) {
    C.cfg = cfg;
    state.view = qs.get('view') || (location.hash || '').slice(1) || cfg.home;
    if (!cfg.views[state.view]) state.view = cfg.home;
    if (cfg.init) cfg.init(state, qs);
    render();
    if (qs.get('open')) { var p = qs.get('open').split(':'); openDrawer(p[0], p.slice(1).join(':')); }
    if (qs.get('menu') === 'user') openUserMenu();
    if (qs.get('palette')) openPalette(qs.get('palette') === '1' ? '' : qs.get('palette'));
    if (cfg.afterBoot) cfg.afterBoot(qs);
  }

  /* ---------- SN-207 shared presentation helpers (promoted from six byte-identical per-role copies; markup unchanged) ---------- */
  var steps = function (list, cur) { return '<ol class="steps" aria-label="مراحل">' + list.map(function (s, i) { var c = i < cur ? 'done' : i === cur ? 'cur' : ''; return '<li class="' + c + '"' + (i === cur ? ' aria-current="step"' : '') + '><span class="sn">' + (i < cur ? ic('check') : fa(i + 1)) + '</span><span class="st-l">' + esc(s) + '</span></li>'; }).join('') + '</ol>'; };
  var bulkbar = function (text, sub, actions) { return '<div class="bulkbar" role="region" aria-label="انتخاب گروهی"><div class="bb-txt"><b>' + text + '</b>' + (sub ? '<span>' + sub + '</span>' : '') + '</div><span class="grow"></span>' + actions + '</div>'; };
  var dtop = function (title, extra) { return '<div class="sheet-grip" aria-hidden="true"></div><div class="dr-top"><span>' + esc(title) + '</span><span class="grow"></span>' + (extra || '') + '<button type="button" class="btn btn-ghost btn-icon btn-sm tip" data-tip="بستن (Esc)" data-close aria-label="بستن">' + ic('x') + '</button></div>'; };
  var dfoot = function (primary, secondary, hintTxt) { return '<div class="dr-foot">' + (primary || '') + (secondary || '<button type="button" class="btn btn-lg btn-ghost" data-close>بستن</button>') + '<span class="grow"></span>' + (hintTxt ? '<span class="hint">' + hintTxt + '</span>' : '') + '</div>'; };
  var timeline = function (items) { return '<ul class="timeline">' + items.map(function (x) { return '<li>' + esc(x[0]) + '<span>' + esc(x[1]) + (x[2] ? ' · ' + esc(x[2]) : '') + '</span></li>'; }).join('') + '</ul>'; };
  // BulkResult strip: requested · eligible · applied · skipped · rejected · failed · unknown. Domain vocabulary may relabel/replace a cell (opt.cells: [key, label, count]);
  // the note only asserts the sum over the cells after «eligible»; a mismatch never reads as success.
  var outcomeStrip = function (c, opt) {
    opt = opt || {};
    var cells = opt.cells || [['requested', 'درخواست‌شده', c.requested], ['eligible', 'واجد شرایط', c.eligible], ['applied', 'اعمال‌شده', c.ok], ['skipped', 'ارسال‌نشده', c.skipped], ['rejected', 'ردشده', c.rejected], ['failed', 'ناموفق', c.failed], ['unknown', 'نامعلوم', c.unknown]];
    var sum = cells.reduce(function (a, x) { return x[0] === 'requested' || x[0] === 'eligible' ? a : a + x[2]; }, 0) === c.requested;
    var eq = opt.sumText || 'مجموع «اعمال‌شده + ارسال‌نشده + ردشده + ناموفق + نامعلوم» برابر «درخواست‌شده» است.';
    return '<div class="outcome-strip" role="group" aria-label="خلاصه نتیجه گروهی">' + cells.map(function (x) { return '<div class="os os-' + x[0] + (x[2] ? '' : ' zero') + '"><span>' + x[1] + '</span><b>' + fa(x[2]) + '</b></div>'; }).join('') + '</div>' +
      '<p class="os-note">' + (sum ? ic('checkCircle') + eq : ic('alert') + 'جمع‌ها با هم نمی‌خوانند؛ پیش از هر اقدام بازخوانی کنید.') + '</p>';
  };

  C.h = { $: $, $$: $$, esc: esc, fa: fa, num: num, money: money, ic: ic, hint: hint, pill: pill, kpi: kpiHtml, freshness: freshness, pageHead: pageHead, queues: queues, toolbar: toolbar, tfoot: tfoot, stateBlock: stateBlock, banner: banner, cbx: cbx, store: store, steps: steps, bulkbar: bulkbar, dtop: dtop, dfoot: dfoot, timeline: timeline, outcomeStrip: outcomeStrip };
  C.boot = boot; C.render = render; C.go = go; C.toast = toast; C.openDrawer = openDrawer; C.closeDrawer = closeDrawer; C.rerenderDrawer = rerenderDrawer;
  C.openMenu = openMenu; C.closeMenu = closeMenu; C.openPalette = openPalette; C.setPref = setPref; C.setCursor = setCursor; C.fitNav = fitNav;
  C.setSim = function (sim) { state.sim = sim; state.offline = sim === 'offline'; if (state.drawer && sim !== 'drawerError') closeDrawer(); render(); };
  window.CRM = C;
  window.SNB = C; // help engine contract (same shape Seller's help.js expects: state, go, toast, openDrawer, closeDrawer, closeMenu)
})();
