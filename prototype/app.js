/* Seller Panel redesign (Version B) — operator-first iteration. Demo only: nothing is saved or sent anywhere. */
(function () {
  'use strict';
  var M = window.MOCK;
  var root = document.documentElement;
  var $ = function (s, r) { return (r || document).querySelector(s); };
  var $$ = function (s, r) { return Array.prototype.slice.call((r || document).querySelectorAll(s)); };
  var esc = function (v) { return String(v == null ? '' : v).replace(/[&<>"']/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]; }); };
  var fa = function (v) { return String(v).replace(/\d/g, function (d) { return '۰۱۲۳۴۵۶۷۸۹'[d]; }); };
  var num = function (n) { return Number(n || 0).toLocaleString('fa-IR'); };
  var money = function (n) { return '<span class="amount"><span class="num">' + num(n) + '</span><small>تومان</small></span>'; };
  var store = { get: function (k) { try { return localStorage.getItem('snproto.' + k); } catch (e) { return null; } }, set: function (k, v) { try { localStorage.setItem('snproto.' + k, v); } catch (e) {} } };

  /* ---------- Icons ---------- */
  var P = {
    phone: '<path d="M22 16.9v3a2 2 0 0 1-2.2 2 19.8 19.8 0 0 1-8.6-3.1 19.5 19.5 0 0 1-6-6A19.8 19.8 0 0 1 2.1 4.2 2 2 0 0 1 4.1 2h3a2 2 0 0 1 2 1.7c.1 1 .4 1.9.7 2.8a2 2 0 0 1-.5 2.1L8 9.9a16 16 0 0 0 6 6l1.3-1.3a2 2 0 0 1 2.1-.4c.9.3 1.8.6 2.8.7a2 2 0 0 1 1.7 2z"/>',
    phoneOff: '<path d="M10.7 13.3a16 16 0 0 0 3.4 2.6l1.3-1.3a2 2 0 0 1 2.1-.4c.9.3 1.8.6 2.8.7a2 2 0 0 1 1.7 2v3a2 2 0 0 1-2.2 2 19.8 19.8 0 0 1-8.6-3.1 19 19 0 0 1-3.3-2.7m-2.8-3.5A19.8 19.8 0 0 1 2.1 4.2 2 2 0 0 1 4.1 2h3a2 2 0 0 1 2 1.7c.1 1 .4 1.9.7 2.8a2 2 0 0 1-.5 2.1L8 9.9"/><path d="M22 2 2 22"/>',
    copy: '<rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>',
    more: '<circle cx="12" cy="5" r="1"/><circle cx="12" cy="12" r="1"/><circle cx="12" cy="19" r="1"/>',
    x: '<path d="M18 6 6 18M6 6l12 12"/>', up: '<path d="m18 15-6-6-6 6"/>', down: '<path d="m6 9 6 6 6-6"/>', chev: '<path d="m6 9 6 6 6-6"/>',
    clock: '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
    alert: '<path d="M12 9v4M12 17h.01"/><path d="M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0z"/>',
    file: '<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 2v6h6M8 13h8M8 17h5"/>',
    users: '<path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.9M16 3.1a4 4 0 0 1 0 7.8"/>',
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
    search: '<circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/>', target: '<circle cx="12" cy="12" r="9"/><circle cx="12" cy="12" r="5"/><circle cx="12" cy="12" r="1"/>',
    sun: '<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/>',
    moon: '<path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z"/>', key: '<rect x="2" y="6" width="20" height="12" rx="2"/><path d="M6 10h.01M10 10h.01M14 10h.01M18 10h.01M8 14h8"/>',
    logout: '<path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9"/>', lock: '<rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>',
    rows: '<path d="M3 6h18M3 12h18M3 18h18"/>', arrow: '<path d="m15 18-6-6 6-6"/>',
    info: '<circle cx="12" cy="12" r="9"/><path d="M12 11v5M12 8h.01"/>', help: '<circle cx="12" cy="12" r="9"/><path d="M9.6 9.3a2.5 2.5 0 0 1 4.8.9c0 1.7-2.4 2.3-2.4 3.8"/><path d="M12 17h.01"/>',
    wifiOff: '<path d="M2 2l20 20M8.5 16.5a5 5 0 0 1 7 0M5 13a10 10 0 0 1 5.2-2.8M19 13a10 10 0 0 0-2.3-1.6M2 8.8a15 15 0 0 1 4.2-2.7M22 8.8a15 15 0 0 0-10.4-4.3M12 20h.01"/>', refresh: '<path d="M21 12a9 9 0 1 1-2.6-6.4L21 8"/><path d="M21 3v5h-5"/>'
  };
  var ic = function (n) { return '<svg class="i" viewBox="0 0 24 24" aria-hidden="true">' + (P[n] || '') + '</svg>'; };

  /* ---------- Status system: colour + icon + text (never colour alone) ---------- */
  var LEAD = {
    none:          { label: 'بدون وضعیت', tone: 'neutral', icon: 'dashed', key: '1' },
    no_answer:     { label: 'جواب نداده', tone: 'amber',   icon: 'phoneOff', key: '2' },
    callback:      { label: 'تماس مجدد',  tone: 'blue',    icon: 'clock', key: '3' },
    not_purchased: { label: 'عدم خرید',   tone: 'red',     icon: 'xCircle', key: '4' },
    duplicate:     { label: 'تکراری',     tone: 'slate',   icon: 'copy', key: '5' },
    invoiced:      { label: 'پیش‌فاکتور', tone: 'violet',  icon: 'file', key: '6' }
  };
  var INV = {
    pre_invoice:    { tone: 'violet', icon: 'file' },
    staged:         { tone: 'teal',   icon: 'layers' },
    finance_review: { tone: 'orange', icon: 'hourglass' },
    approved:       { tone: 'green',  icon: 'checkCircle' },
    rejected:       { tone: 'red',    icon: 'alert' },
    cancelled:      { tone: 'slate',  icon: 'ban' }
  };
  var CASE_TONE = { green: { tone: 'green', icon: 'checkCircle' }, blue: { tone: 'teal', icon: 'send' }, neutral: { tone: 'neutral', icon: 'dashed' } };
  // Contextual help: small (i) button; tooltip shows on hover AND keyboard focus.
  var hint = function (text, below) { return '<button type="button" class="qhint tip' + (below ? ' below' : '') + '" data-tip="' + esc(text) + '" aria-label="' + esc(text) + '">' + ic('info') + '</button>'; };
  var pill = function (label, tone, icon) { return '<span class="pill t-' + tone + '">' + (icon ? ic(icon) : '') + esc(label) + '</span>'; };
  var leadPill = function (l) { var s = LEAD[l.status]; return pill(s.label, s.tone, s.icon); };
  var invPill = function (i) { var s = INV[i.status] || { tone: 'neutral' }; return pill(i.statusLabel, s.tone, s.icon); };
  var qStyle = function (tone) { return ' style="--qc:var(--' + tone + '-dot);--qbg:var(--' + tone + '-bg);--qfg:var(--' + tone + '-fg)"'; };
  // Next step: amber = needs attention soon, red = due now / at risk.
  var nextInfo = function (l) {
    if (!l.next) return null;
    var cls = l.next.kind === 'archive' || (l.next.kind === 'callback' && /امروز/.test(l.next.text)) ? 'due' : l.next.warn ? 'warn' : '';
    var icon = { callback: 'clock', attempts: 'phoneOff', archive: 'alert', invoice: 'file' }[l.next.kind] || 'clock';
    return { cls: cls, icon: icon, text: l.next.text };
  };
  var lastActivity = function (l) { return l.status === 'none' ? '—' : l.status === 'no_answer' ? '۲ روز پیش' : l.status === 'callback' ? 'دیروز' : l.assignedAgo; };

  /* ---------- Navigation config ---------- */
  var leadsCount = function (s) { return M.leads.filter(function (l) { return s === 'all' || l.status === s; }).length; };
  var NAV = [
    { id: 'leads', label: 'شماره‌های من', icon: 'users', count: function () { return leadsCount('none'); }, alert: true, mobile: 'شماره‌ها' },
    { id: 'conversions', label: 'آماده‌های تبدیل', icon: 'swap', count: function () { return M.conversions.length; }, mobile: 'تبدیل' },
    { id: 'repeat', label: 'نیاز به اقدام مجدد', icon: 'repeat', count: function () { return 0; } },
    { id: 'invoices', label: 'فاکتورها', icon: 'receipt', count: function () { return M.invoices.filter(function (i) { return i.next; }).length; }, mobile: 'فاکتورها' },
    { sep: true },
    { id: 'behavior', label: 'رفتار مشتریان', icon: 'activity' },
    { id: 'request', label: 'درخواست شماره', icon: 'plus' },
    { id: 'wallet', label: 'کیف پول', icon: 'wallet', mobile: 'کیف پول' }
  ];
  var QUEUE_ORDER = ['none', 'no_answer', 'callback', 'not_purchased', 'duplicate', 'invoiced', 'all'];

  /* ---------- State ---------- */
  var qs = new URLSearchParams(location.search);
  var state = {
    view: qs.get('view') || (location.hash || '#leads').slice(1) || 'leads',
    leadQueue: qs.get('queue') || 'none',
    invQueue: qs.get('iq') || 'all',
    walletFilter: 'all',
    drawer: null, outcome: qs.get('outcome') || null,
    sim: qs.get('sim') || null, offline: qs.get('sim') === 'offline',
    cursor: 0
  };
  var prefs = { theme: root.getAttribute('data-theme'), density: root.getAttribute('data-density'), focus: root.getAttribute('data-focus') === '1' };
  function setPref(k, v) {
    prefs[k] = v;
    if (k === 'focus') { if (v) root.setAttribute('data-focus', '1'); else root.removeAttribute('data-focus'); store.set('focus', v ? '1' : '0'); }
    else { root.setAttribute('data-' + k, v); store.set(k, v); }
    var fb = $('.focus-btn'); if (fb) fb.setAttribute('aria-pressed', String(prefs.focus));
  }

  /* ---------- Shell ---------- */
  function renderNav() {
    $('#nav').innerHTML = NAV.map(function (n) {
      if (n.sep) return '<span class="nav-sep" aria-hidden="true"></span>';
      var c = n.count ? n.count() : null;
      return '<a href="#' + n.id + '" data-view="' + n.id + '"' + (state.view === n.id ? ' aria-current="page"' : '') + '>' + ic(n.icon) + esc(n.label) + (c ? '<span class="count' + (n.alert ? ' alert' : '') + '">' + fa(c) + '</span>' : '') + '</a>';
    }).join('');
    var mob = NAV.filter(function (n) { return n.mobile; });
    $('#bottom-nav').innerHTML = mob.map(function (n) {
      var c = n.alert && n.count ? n.count() : 0;
      return '<a href="#' + n.id + '" data-view="' + n.id + '"' + (state.view === n.id ? ' aria-current="page"' : '') + '>' + ic(n.icon) + (c ? '<span class="b">' + fa(c) + '</span>' : '') + esc(n.mobile) + '</a>';
    }).join('') + '<a href="#more" data-more="1"' + (['repeat', 'behavior', 'request'].indexOf(state.view) > -1 ? ' aria-current="page"' : '') + '>' + ic('grid') + 'بیشتر</a>';
    var cur = NAV.filter(function (n) { return n.id === state.view; })[0];
    $('#crumb').textContent = cur ? cur.label : '';
    $('#m-title').textContent = cur ? cur.label : '';
    document.title = (cur ? cur.label + ' — ' : '') + 'پنل فروشنده (نمونه)';
    var swA = $('#sw-a'); if (swA) swA.href = 'a.html#' + state.view;
    // Orientation only — no targets, no ranking.
    var open = M.leads.filter(function (l) { return ['none', 'no_answer', 'callback'].indexOf(l.status) > -1; }).length;
    var done = M.leads.length - leadsCount('none');
    $('#workload').innerHTML = '<span>' + ic('rows') + '</span><span><b>' + fa(open) + '</b> مورد باز</span><span class="sep extra"></span><span class="due extra">' + ic('clock') + ' <b style="color:inherit">۱</b> تماس امروز</span><span class="sep extra"></span><span class="extra tip" data-tip="پیگیری‌شده از کل شماره‌ها">' + '<span class="meter" style="display:inline-block;vertical-align:middle"><i style="width:' + Math.round(done / M.leads.length * 100) + '%"></i></span></span>';
  }

  function pageHead(title, sub, kpis, cta) {
    var tourable = ['leads', 'invoices', 'conversions', 'wallet'].indexOf(state.view) > -1;
    return '<div class="page-head"><div><h1>' + esc(title) + (tourable ? '<button type="button" class="page-help tip below" data-tip="راهنمای این صفحه" aria-label="راهنمای این صفحه" data-page-tour="' + state.view + '">' + ic('help') + '</button>' : '') + '</h1>' + (sub ? '<div class="sub">' + sub + '</div>' : '') + '</div><div class="spacer"></div>' +
      (kpis ? '<div class="kpis" role="group" aria-label="شاخص‌های این صفحه">' + kpis.map(function (k) {
        var tag = k.filter ? 'button type="button" data-kpi-filter="' + k.filter + '" data-tip="نمایش همین صف"' : 'div';
        return '<' + tag + ' class="kpi' + (k.tone ? ' ' + k.tone : '') + (k.keep ? ' keep' : '') + '" style="--k:var(--' + (k.color || 'border') + '-dot)"><span>' + esc(k.label) + '</span><b>' + k.value + '</b></' + tag.split(' ')[0] + '>';
      }).join('') + '</div>' : '') +
      '<div class="head-cta">' + (cta || '') + '<button type="button" class="btn focus-btn tip" data-tip="عناصر فرعی کم‌رنگ می‌شوند" aria-pressed="' + prefs.focus + '" data-toggle-focus>' + ic('target') + 'حالت تمرکز</button></div></div>';
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
      (chips || []).map(function (c) { return '<button type="button" class="chip-btn" aria-label="' + esc(c.label) + '" data-demo="فیلتر «' + esc(c.label) + '» در نسخه واقعی یک منوی انتخاب باز می‌کند">' + ic(c.icon || 'plus') + '<span>' + esc(c.label) + '</span></button>'; }).join('') +
      '<span class="grow"></span>' + (right || '') + '</div>';
  }
  function tfoot(shown, total) {
    return '<div class="tfoot"><span>نمایش <b class="num">' + fa('1–' + shown) + '</b> از <b class="num">' + fa(total) + '</b></span>' +
      '<div class="pager"><button type="button" aria-label="صفحه قبلی">›</button><button type="button" aria-current="page">۱</button><button type="button" aria-label="صفحه بعدی">‹</button></div><span class="desk">۲۰ در صفحه</span></div>';
  }

  /* ---------- Views ---------- */
  var V = {};
  function currentLeads() { return M.leads.filter(function (l) { return state.leadQueue === 'all' || l.status === state.leadQueue; }); }

  V.leads = function () {
    var list = currentLeads();
    var qItems = QUEUE_ORDER.map(function (k) {
      if (k === 'all') return { id: 'all', label: 'همه', n: M.leads.length, tone: 'neutral', key: '0' };
      return { id: k, label: LEAD[k].label, n: leadsCount(k), tone: LEAD[k].tone, icon: LEAD[k].icon, key: LEAD[k].key };
    });
    var done = M.leads.length - leadsCount('none');
    var kpis = [
      { label: 'بدون وضعیت', value: fa(leadsCount('none')), filter: 'none', color: 'neutral', keep: true },
      { label: 'تماس مجدد امروز', value: fa(1), filter: 'callback', color: 'blue', keep: true },
      { label: 'نرخ پیگیری', value: fa(Math.round(done / M.leads.length * 100)) + '<small>٪</small>', color: 'teal' },
      { label: 'پیش‌فاکتور این هفته', value: fa(leadsCount('invoiced')), color: 'violet' }
    ];
    var rows = list.map(function (l, i) {
      var n = nextInfo(l);
      var closed = ['invoiced', 'not_purchased', 'duplicate'].indexOf(l.status) > -1;
      var sel = state.drawer && state.drawer.kind === 'lead' && state.drawer.id === l.id;
      return '<tr data-lead="' + l.id + '" tabindex="' + (i === state.cursor ? 0 : -1) + '" class="' + (sel ? 'sel' : '') + '" aria-selected="' + sel + '">' +
        '<td>' + leadPill(l) + '</td>' +
        '<td><div class="who"><b>' + esc(l.name) + '</b><span class="phone"><span class="mono phone-num">' + fa(l.phone) + '</span><button type="button" class="copy tip" data-tip="کپی شماره" data-copy="' + l.phone + '" aria-label="کپی شماره">' + ic('copy') + '</button></span></div></td>' +
        '<td>' + esc(l.city) + '</td>' +
        '<td class="col-opt"><span class="tag' + (l.source === 'MIS' ? '' : ' teal') + '">' + esc(l.source) + '</span></td>' +
        '<td class="col-opt muted">' + esc(l.assignedAgo) + '</td>' +
        '<td>' + (n ? '<span class="next ' + n.cls + '">' + ic(n.icon) + esc(n.text) + '</span>' + (l.next.kind === 'attempts' ? hint('این مشتری ' + fa(l.attempts) + ' بار پاسخ نداده است؛ سقف این مرحله ۳ تلاش است.') : '') : '<span class="muted">—</span>') + '</td>' +
        '<td class="col-opt">' + (l.prob ? esc(l.prob) : '<span class="muted">—</span>') + '</td>' +
        '<td class="col-actions"><div class="row-actions">' +
        '<a class="btn btn-ghost btn-icon btn-sm tip" data-tip="تماس" href="tel:' + l.phone + '" aria-label="تماس با ' + esc(l.name) + '" data-demo="نمایشی: روی موبایل شماره‌گیر باز می‌شود">' + ic('phone') + '</a>' +
        (closed ? '<button type="button" class="btn btn-sm" data-open-lead="' + l.id + '">' + (l.status === 'invoiced' ? 'مشاهده فاکتور' : 'مشاهده') + '</button>'
                : '<button type="button" class="btn btn-soft btn-sm tip" data-tip="Enter" data-open-lead="' + l.id + '">ثبت نتیجه</button>') +
        '<button type="button" class="btn btn-ghost btn-icon btn-sm tip" data-tip="بیشتر" data-more="lead" data-id="' + l.id + '" aria-label="اقدام‌های بیشتر">' + ic('more') + '</button>' +
        '</div></td></tr>';
    }).join('');
    var cards = list.map(function (l) {
      var n = nextInfo(l), closed = ['invoiced', 'not_purchased', 'duplicate'].indexOf(l.status) > -1;
      return '<div class="lcard" data-lead="' + l.id + '"><div class="top"><div class="who"><b>' + esc(l.name) + '</b><span class="phone"><span class="mono phone-num">' + fa(l.phone) + '</span></span></div>' + leadPill(l) + '</div>' +
        '<div class="meta"><span>' + esc(l.city) + '</span><span>' + esc(l.source) + '</span><span>' + esc(l.assignedAgo) + '</span>' + (n ? '<span class="next ' + n.cls + '">' + ic(n.icon) + esc(n.text) + '</span>' : '') + '</div>' +
        '<div class="acts"><a class="btn call" href="tel:' + l.phone + '" data-demo="نمایشی: شماره‌گیر باز می‌شود">' + ic('phone') + 'تماس</a>' + (closed ? '<button type="button" class="btn" data-open-lead="' + l.id + '">مشاهده</button>' : '<button type="button" class="btn btn-soft" data-open-lead="' + l.id + '">ثبت نتیجه</button>') + '</div></div>';
    }).join('');
    return pageHead('شماره‌های من', fa(M.leads.length) + ' شماره تخصیص‌یافته · به‌روز', kpis, '<button type="button" class="btn" data-go="request">' + ic('plus') + 'درخواست شماره</button>') +
      '<section class="panel has-cards">' + queues(qItems, state.leadQueue, 'data-lead-queue') +
      toolbar('جستجو در این صف: نام، شماره یا شهر', [{ label: 'منبع', icon: 'filter' }, { label: 'احتمال فروش', icon: 'filter' }, { label: 'تاریخ تخصیص', icon: 'clock' }],
        '<span class="muted desk">' + fa(list.length) + ' نتیجه</span><button type="button" class="btn btn-ghost btn-sm desk">' + ic('sort') + 'جدیدترین</button>') +
      (list.length ? '<div class="tbl-wrap"><table class="tbl" aria-label="فهرست شماره‌ها"><thead><tr><th>وضعیت</th><th>مشتری</th><th>شهر</th><th class="col-opt">منبع</th><th class="col-opt">تخصیص</th><th>گام بعدی ' + hint('اقدام پیشنهادی بعدی بر اساس وضعیت فعلی پرونده.', true) + '</th><th class="col-opt">احتمال ' + hint('برای اولویت‌بندی پیگیری است و وضعیت مشتری را تغییر نمی‌دهد.', true) + '</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div><div class="card-list">' + cards + '</div>' + tfoot(list.length, list.length)
        : '<div class="empty"><div class="ico">' + ic('check') + '</div><b>این صف خالی است</b><span>شماره‌ای با این وضعیت ندارید.</span></div>') + '</section>';
  };

  var invActions = { send_link: { label: 'ارسال لینک پرداخت', icon: 'send' }, register_payment: { label: 'ثبت واریز و فیش', icon: 'upload' }, resend_finance: { label: 'ارسال مجدد به مالی', icon: 'repeat' } };
  V.invoices = function () {
    var I = M.invoices, byQ = function (q) { return I.filter(function (i) { return q === 'all' || i.status === q; }); };
    var list = byQ(state.invQueue);
    var remaining = I.reduce(function (s, i) { return s + (i.status === 'cancelled' ? 0 : i.total - i.paid); }, 0);
    var kpis = [
      { label: 'در انتظار پرداخت', value: fa(byQ('pre_invoice').length + byQ('staged').length), filter: 'inv:pre_invoice', color: 'violet', keep: true },
      { label: 'در بررسی مالی', value: fa(byQ('finance_review').length), filter: 'inv:finance_review', color: 'orange' },
      { label: 'رد شده', value: fa(byQ('rejected').length), tone: 'bad', filter: 'inv:rejected', color: 'red', keep: true },
      { label: 'مانده قابل وصول', value: '<span class="num">' + num(remaining) + '</span><small>تومان</small>', color: 'teal' }
    ];
    var qItems = [{ id: 'all', label: 'همه', n: I.length, tone: 'neutral' }].concat(['pre_invoice', 'staged', 'finance_review', 'approved', 'rejected', 'cancelled'].map(function (k) {
      var lbl = { pre_invoice: 'پیش‌فاکتور', staged: 'مرحله‌ای', finance_review: 'در بررسی مالی', approved: 'تایید شده', rejected: 'رد شده', cancelled: 'لغو شده' }[k];
      return { id: k, label: lbl, n: byQ(k).length, tone: INV[k].tone, icon: INV[k].icon };
    }));
    var rows = list.map(function (i) {
      var pct = i.total ? Math.round(i.paid / i.total * 100) : 0, a = invActions[i.next];
      var primary = a ? '<button type="button" class="btn btn-soft btn-sm" data-open-inv="' + i.code + '">' + ic(a.icon) + esc(a.label) + '</button>' : '<button type="button" class="btn btn-sm" data-open-inv="' + i.code + '">جزئیات</button>';
      return '<tr data-inv="' + i.code + '" tabindex="-1"' + (state.drawer && state.drawer.kind === 'invoice' && state.drawer.id === i.code ? ' class="sel"' : '') + '>' +
        '<td><span class="mono" style="font-weight:500">' + fa(i.code) + '</span></td>' +
        '<td><div class="who"><b>' + esc(i.customer) + '</b><span class="phone"><span class="mono phone-num">' + fa(i.phone) + '</span></span></div></td>' +
        '<td class="col-opt">' + esc(i.product) + '</td>' +
        '<td><div class="prog"><div class="line"><span>' + money(i.total) + '</span><span>' + (i.stages > 1 ? 'مرحله ' + fa(i.stage) + ' از ' + fa(i.stages) : 'پرداخت یک‌جا') + '</span></div><div class="bar"><i style="width:' + pct + '%"></i></div></div></td>' +
        '<td>' + (i.status === 'cancelled' ? '<span class="muted">—</span>' : i.total - i.paid <= 0 ? '<span class="settled">✓ تسویه</span>' : money(i.total - i.paid)) + '</td>' +
        '<td>' + invPill(i) + (i.rejectReason ? '<div style="font-size:var(--t-micro);color:var(--red-fg);margin-top:4px">' + esc(i.rejectReason) + '</div>' : '') + '</td>' +
        '<td class="col-actions"><div class="row-actions">' + primary + '<button type="button" class="btn btn-ghost btn-icon btn-sm tip" data-tip="بیشتر" data-more="invoice" data-id="' + i.code + '" aria-label="اقدام‌های بیشتر">' + ic('more') + '</button></div></td></tr>';
    }).join('');
    var cards = list.map(function (i) {
      var a = invActions[i.next];
      return '<div class="lcard" data-inv="' + i.code + '"><div class="top"><div class="who"><b>' + esc(i.customer) + '</b><span class="phone">فاکتور <span class="mono">' + fa(i.code) + '</span></span></div>' + invPill(i) + '</div>' +
        '<div class="meta"><span>کل ' + money(i.total) + '</span><span>مانده ' + money(i.total - i.paid) + '</span></div>' +
        '<div class="acts"><button type="button" class="btn" data-more="invoice" data-id="' + i.code + '">' + ic('more') + 'بیشتر</button>' + (a ? '<button type="button" class="btn btn-soft" data-open-inv="' + i.code + '">' + esc(a.label) + '</button>' : '<button type="button" class="btn" data-open-inv="' + i.code + '">جزئیات</button>') + '</div></div>';
    }).join('');
    return pageHead('فاکتورها', 'فقط فاکتورهای صادرشده توسط شما', kpis) +
      '<section class="panel has-cards">' + queues(qItems, state.invQueue, 'data-inv-queue') +
      toolbar('کد فاکتور، نام یا موبایل مشتری', [{ label: 'بازه تاریخ', icon: 'clock' }, { label: 'محصول', icon: 'filter' }], '<span class="muted desk">' + fa(list.length) + ' فاکتور</span>') +
      '<div class="tbl-wrap"><table class="tbl" aria-label="فهرست فاکتورها"><thead><tr><th>کد</th><th>مشتری</th><th class="col-opt">محصول</th><th>مبلغ و پیشرفت پرداخت</th><th>مانده</th><th>وضعیت مالی ' + hint('وضعیت بررسی یا پرداخت فاکتور.', true) + '</th><th class="col-actions">اقدام بعدی</th></tr></thead><tbody>' + rows + '</tbody></table></div>' +
      '<div class="card-list">' + cards + '</div>' + tfoot(list.length, list.length) + '</section>';
  };

  V.conversions = function () {
    var C = M.conversions;
    var kpis = [{ label: 'کل پرونده', value: fa(C.length), color: 'neutral', keep: true }, { label: 'نیازمند پیگیری', value: fa(1), tone: 'warn', color: 'amber', keep: true }, { label: 'تماس سررسیده', value: fa(0), tone: 'bad', color: 'red' }, { label: 'دارای مانده', value: fa(C.filter(function (c) { return c.remaining > 0; }).length), color: 'teal' }];
    var rows = C.map(function (c) {
      var t = CASE_TONE[c.tone] || CASE_TONE.neutral;
      return '<tr data-case="' + c.id + '" tabindex="-1"><td><span class="mono" style="font-weight:500">#' + fa(c.id) + '</span><div class="muted" style="font-size:var(--t-micro)">فاکتور ' + fa(c.invoice) + '</div></td>' +
        '<td><div class="who"><b>' + esc(c.customer) + '</b><span class="phone"><span class="mono phone-num">' + fa(c.phone) + '</span></span></div></td>' +
        '<td>' + esc(c.option) + '</td><td>' + (c.remaining ? money(c.remaining) : '<span class="muted">—</span>') + '</td>' +
        '<td class="muted">' + esc(c.stage) + '</td><td>' + pill(c.result, t.tone, t.icon) + '</td>' +
        '<td class="col-actions"><div class="row-actions"><button type="button" class="btn btn-soft btn-sm" data-open-case="' + c.id + '">ثبت نتیجه تماس</button><button type="button" class="btn btn-ghost btn-icon btn-sm tip" data-tip="بیشتر" data-demo="منوی بیشتر: کپی لینک، باز کردن فاکتور" aria-label="بیشتر">' + ic('more') + '</button></div></td></tr>';
    }).join('');
    return pageHead('آماده‌های تبدیل', 'پرونده‌هایی که هنگام صدور فاکتور «تبدیل توسط خودم» را انتخاب کرده‌اید', kpis) +
      '<section class="panel">' + queues([{ id: 'all', label: 'همه', n: C.length, tone: 'neutral' }, { id: 'follow', label: 'نیازمند پیگیری', n: 1, tone: 'amber', icon: 'clock' }, { id: 'link', label: 'لینک ارسال شد', n: 1, tone: 'teal', icon: 'send' }, { id: 'done', label: 'تکمیل‌شده', n: 0, tone: 'green', icon: 'checkCircle' }], 'all', 'data-demo-queue') +
      toolbar('نام، موبایل، فاکتور یا شماره پرونده', [{ label: 'مرحله پرونده', icon: 'filter' }, { label: 'نتیجه تماس', icon: 'filter' }]) +
      '<div class="tbl-wrap"><table class="tbl"><thead><tr><th>پرونده</th><th>مشتری</th><th>گزینه</th><th>مانده</th><th>مرحله</th><th>نتیجه تماس</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div>' + tfoot(C.length, C.length) + '</section>';
  };

  V.repeat = function () {
    return pageHead('نیاز به اقدام مجدد', 'پرداخت‌های مرحله‌ای که مرحله بعدشان به شما ارجاع شده است', [{ label: 'باز', value: fa(0), color: 'neutral', keep: true }, { label: 'این ماه انجام‌شده', value: fa(4), tone: 'good', color: 'green' }]) +
      '<section class="panel"><div class="empty"><div class="ico">' + ic('check') + '</div><b>کاری در این صف نیست</b><span>وقتی مرحله بعدی یک پرداخت به شما ارجاع شود، اینجا نمایش داده می‌شود.<br>مالک فروش و پورسانت پایه برای فروشنده صادرکننده اولیه حفظ می‌شود.</span></div></section>';
  };

  V.behavior = function () {
    var rows = M.customerActions.map(function (a) {
      return '<tr data-behavior="' + a.invoice + '" tabindex="-1"><td><div class="who"><b>' + esc(a.name) + '</b><span class="phone"><span class="mono phone-num">' + fa(a.phone) + '</span></span></div></td>' +
        '<td><span class="mono">' + fa(a.invoice) + '</span></td><td>' + esc(a.last) + (a.hot ? ' <span class="hot">فعال</span>' : '') + '</td><td class="muted">' + esc(a.when) + '</td><td>' + fa(a.count) + ' رویداد</td>' +
        '<td class="col-actions"><button type="button" class="btn btn-sm" data-open-behavior="' + a.invoice + '">مشاهده روند</button></td></tr>';
    }).join('');
    return pageHead('رفتار مشتریان', 'باز شدن لینک، مشاهده محصول، گردونه، کد تخفیف و پرداخت روی صفحه فاکتور', [{ label: 'مشتری فعال امروز', value: fa(2), color: 'teal', keep: true }, { label: 'لینک باز شده (۷ روز)', value: fa(14), color: 'blue' }, { label: 'پرداخت پس از بازدید', value: fa(38) + '<small>٪</small>', tone: 'good', color: 'green' }]) +
      '<section class="panel">' + toolbar('شماره مشتری یا کد فاکتور', [{ label: 'نوع رویداد', icon: 'filter' }, { label: 'بازه زمانی', icon: 'clock' }]) +
      '<div class="tbl-wrap"><table class="tbl"><thead><tr><th>مشتری</th><th>فاکتور</th><th>آخرین فعالیت</th><th>زمان</th><th>تعداد</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div>' + tfoot(5, 20) + '</section>';
  };

  V.request = function () {
    var reqs = M.extraRequests.map(function (r) { return '<div class="mini"><div class="grow"><span class="mono phone-num">' + fa(r.phone) + '</span><div class="muted" style="font-size:var(--t-meta)">' + esc(r.name) + ' · ' + esc(r.when) + '</div></div>' + pill(r.status, r.tone === 'amber' ? 'orange' : r.tone, r.tone === 'green' ? 'checkCircle' : r.tone === 'red' ? 'xCircle' : 'hourglass') + '</div>'; }).join('');
    return pageHead('درخواست اضافه کردن شماره', 'برای شماره‌ای خارج از لیست تخصیص‌یافته؛ پس از تایید مدیر فروش به «شماره‌های من» اضافه می‌شود') +
      '<div class="form-card"><section class="panel"><div class="sec-h"><h3>درخواست جدید</h3></div>' +
      '<div style="display:grid;gap:12px"><label class="lbl">شماره موبایل مشتری <span class="req">*</span><input class="input" inputmode="tel" placeholder="09xxxxxxxxx" dir="ltr" style="text-align:right"></label>' +
      '<label class="lbl">نام مشتری <span class="muted">(اختیاری)</span><input class="input"></label><label class="lbl">یادداشت <span class="muted">(اختیاری)</span><textarea class="input" placeholder="علت درخواست یا توضیح کوتاه"></textarea></label>' +
      '<div><button type="button" class="btn btn-primary" data-demo="نمایشی: درخواست ثبت نشد">ثبت درخواست</button></div></div></section>' +
      '<section class="panel"><div class="sec-h"><h3>درخواست‌های اخیر من</h3><span class="aside">' + fa(M.extraRequests.length) + ' مورد</span></div><div class="mini-list">' + reqs + '</div></section></div>';
  };

  V.wallet = function () {
    var W = M.wallet;
    var tx = W.tx.filter(function (t) { return state.walletFilter === 'all' || (state.walletFilter === 'online' ? t.channel === 'آنلاین' : t.channel !== 'آنلاین'); });
    var rows = tx.map(function (t) {
      return '<tr><td class="muted"><span class="num">' + esc(t.date) + '</span></td><td>' + pill('بستانکار', 'green', 'plus') + '</td><td><span class="tag' + (t.channel === 'آنلاین' ? ' teal' : '') + '">' + esc(t.channel) + '</span></td>' +
        '<td>' + esc(t.desc) + '<div class="muted" style="font-size:var(--t-micro)">' + esc(t.rule) + '</div></td><td style="text-align:left"><span class="credit">+<span class="num">' + num(t.amount) + '</span></span> <small class="muted">تومان</small></td></tr>';
    }).join('');
    return pageHead('کیف پول', 'فروش و پورسانت') +
      '<div class="wallet-sum"><div class="wcard hero"><span>موجودی قابل تسویه</span><b><span class="num">' + num(W.balance) + '</span><small>تومان</small></b><div class="foot"><span>آخرین واریز: ۱۴۰۵/۰۶/۱۷</span><span class="seg"><button type="button" aria-pressed="true">فروش و پورسانت</button><button type="button" aria-pressed="false" data-demo="نمایشی: کیف پول پروژه‌ها">پروژه‌ها</button></span></div></div>' +
      '<div class="wcard"><span>کل بستانکاری</span><b><span class="num">' + num(W.credit) + '</span><small>تومان</small></b><div class="foot"><span>۸ تراکنش در ۳۰ روز اخیر</span></div></div>' +
      '<div class="wcard"><span>کل برداشت / تسویه</span><b><span class="num">' + num(W.debit) + '</span><small>تومان</small></b><div class="foot"><span>تسویه‌ای ثبت نشده</span></div></div></div>' +
      '<section class="panel"><div class="toolbar"><div class="seg" role="group" aria-label="نوع پرداخت">' +
      [['all', 'همه'], ['online', 'پرداخت آنلاین'], ['card', 'کارت‌به‌کارت']].map(function (s) { return '<button type="button" data-wallet="' + s[0] + '" aria-pressed="' + (state.walletFilter === s[0]) + '">' + s[1] + '</button>'; }).join('') +
      '</div><span class="grow"></span><button type="button" class="chip-btn" aria-label="بازه تاریخ">' + ic('clock') + '<span>بازه تاریخ</span></button></div>' +
      '<div class="tbl-wrap"><table class="tbl"><thead><tr><th>تاریخ</th><th>نوع</th><th>کانال</th><th>شرح</th><th style="text-align:left">مبلغ</th></tr></thead><tbody>' + rows + '</tbody></table></div>' + tfoot(tx.length, 40) + '</section>';
  };

  /* ---------- Drawer ---------- */
  var leadOutcomes = [
    { id: 'no_answer', label: 'جواب نداده', c: 'amber', icon: 'phoneOff', k: '1' },
    { id: 'callback', label: 'تماس مجدد', c: 'blue', icon: 'clock', hint: 'تاریخ و ساعت لازم', k: '2' },
    { id: 'not_purchased', label: 'عدم خرید', c: 'red', icon: 'xCircle', hint: 'دلیل الزامی', k: '3' },
    { id: 'duplicate', label: 'تکراری', c: 'slate', icon: 'copy', hint: 'قبلاً ثبت شده', k: '4' },
    { id: 'success', label: 'موفق — صدور پیش‌فاکتور', c: 'green', icon: 'checkCircle', hint: 'فرم پیش‌فاکتور همین‌جا باز می‌شود', wide: true, k: '5' }
  ];
  function outcomeBtn(x, o, hint) {
    return '<button type="button" class="outcome' + (x.wide ? ' wide' : '') + '" style="--c:var(--' + x.c + '-dot)" data-outcome="' + x.id + '" aria-pressed="' + (o === x.id) + '"><b>' + ic(x.icon) + esc(x.label) + (x.k ? '<span class="k">' + fa(x.k) + '</span>' : '') + '</b><small>' + esc(hint || x.hint || '') + '</small></button>';
  }
  function leadDrawer(l) {
    var S = LEAD[l.status], idx = M.leads.indexOf(l), attemptsNext = Math.min(3, l.attempts + 1), o = state.outcome, n = nextInfo(l);
    var outcomes = leadOutcomes.map(function (x) { return outcomeBtn(x, o, x.id === 'no_answer' ? 'تلاش ' + fa(attemptsNext) + ' از ۳' : null); }).join('');
    var cond = '' +
      '<div class="cond" data-cond="no_answer"' + (o === 'no_answer' ? '' : ' hidden') + '>' + (attemptsNext >= 3
        ? '<div class="note warn">' + ic('alert') + '<span>با ثبت این تلاش سقف ۳ تماس پر می‌شود. اگر تا ۳ روز فعالیت دیگری ثبت نشود، شماره به بایگانی مدیر فروش منتقل می‌شود.</span></div>'
        : '<div class="note info">' + ic('phoneOff') + '<span>هر تماس واقعیِ بی‌پاسخ جدا ثبت می‌شود. این تلاش ' + fa(attemptsNext) + ' از ۳ است.</span></div>') + '</div>' +
      '<div class="cond" data-cond="callback"' + (o === 'callback' ? '' : ' hidden') + '><div class="row2"><label class="lbl">تاریخ شمسی <span class="req">*</span><input class="input" value="۱۴۰۵/۰۷/۱۰"></label>' +
      '<label class="lbl">ساعت تهران <span class="req">*</span><span class="time"><input class="input" value="10" aria-label="ساعت"> : <input class="input" value="30" aria-label="دقیقه"></span></label></div><div class="muted" style="font-size:var(--t-meta)">یادآور در پنل برای همین زمان تنظیم می‌شود.</div></div>' +
      '<div class="cond" data-cond="not_purchased"' + (o === 'not_purchased' ? '' : ' hidden') + '><label class="lbl">دلیل عدم خرید <span class="req">*</span><textarea class="input" placeholder="دلیل اعلام‌شده یا نتیجه نهایی مکالمه"></textarea></label></div>' +
      '<div class="cond" data-cond="duplicate"' + (o === 'duplicate' ? '' : ' hidden') + '><div class="note info">' + ic('copy') + '<span>شماره به‌عنوان تکراری علامت می‌خورد و نام شما به‌عنوان ثبت‌کننده ذخیره می‌شود.</span></div></div>' +
      '<div class="cond" data-cond="success"' + (o === 'success' ? '' : ' hidden') + '><div class="row2"><label class="lbl">محصول <span class="req">*</span><select class="input"><option>اشتراک ۱۰ ستاره — ۱۰٬۰۰۰٬۰۰۰ تومان</option><option>اشتراک نقره‌ای</option></select></label>' +
      '<label class="lbl">نوع پرداخت<span class="seg"><button type="button" aria-pressed="true">پرداخت کامل</button><button type="button" aria-pressed="false">مرحله‌ای</button></span></label></div>' +
      '<div class="note info">' + ic('send') + '<span>پس از صدور، پیش‌فاکتور مطابق تنظیمات فعلی برای مشتری پیامک می‌شود.</span></div></div>';
    var ctaText = { no_answer: 'ثبت تماس بی‌پاسخ (' + fa(attemptsNext) + ' از ۳)', callback: 'ثبت تماس مجدد — ۱۴۰۵/۰۷/۱۰ ساعت ۱۰:۳۰', not_purchased: 'ثبت عدم خرید', duplicate: 'ثبت به‌عنوان تکراری', success: 'صدور پیش‌فاکتور' }[o];
    var invs = M.invoices.filter(function (i) { return i.phone === l.phone; });
    return '<div class="sheet-grip" aria-hidden="true"></div>' +
      '<div class="dr-top"><button type="button" class="btn btn-ghost btn-icon btn-sm tip" data-tip="قبلی (K)" data-step="-1" aria-label="مشتری قبلی">' + ic('up') + '</button><button type="button" class="btn btn-ghost btn-icon btn-sm tip" data-tip="بعدی (J)" data-step="1" aria-label="مشتری بعدی">' + ic('down') + '</button>' +
      '<span>' + fa(idx + 1) + ' از ' + fa(M.leads.length) + '</span><span class="grow"></span><span class="desk" style="font-size:var(--t-micro)">۱–۵ نتیجه · Enter ثبت · Esc بستن</span>' +
      '<button type="button" class="btn btn-ghost btn-icon btn-sm tip" data-tip="بستن (Esc)" data-close aria-label="بستن">' + ic('x') + '</button></div>' +
      '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">' + esc(l.name) + '</h2>' + leadPill(l) + '</div>' +
      '<div class="dr-meta"><span class="phone mono phone-num">' + fa(l.phone) + '</span><span>' + esc(l.city) + '، ' + esc(l.province) + '</span><span class="tag' + (l.source === 'MIS' ? '' : ' teal') + '">' + esc(l.source) + '</span></div>' +
      '<div class="dr-facts"><span class="fact-chip">' + ic('history') + 'آخرین فعالیت: ' + esc(lastActivity(l)) + '</span>' + (n ? '<span class="fact-chip ' + n.cls + '">' + ic(n.icon) + 'گام بعدی: ' + esc(n.text) + '</span>' : '<span class="fact-chip">' + ic('arrow') + 'گام بعدی: ثبت اولین نتیجه</span>') + '<span class="fact-chip">' + ic('clock') + 'تخصیص: ' + esc(l.assigned) + '</span></div>' +
      '<div class="dr-quick"><a class="btn" href="tel:' + l.phone + '" data-demo="نمایشی: شماره‌گیر باز می‌شود">' + ic('phone') + 'تماس</a><button type="button" class="btn" data-copy="' + l.phone + '">' + ic('copy') + 'کپی شماره</button></div></div>' +
      '<div class="dr-body">' +
      '<section class="sec primary"><div class="sec-h"><h3>ثبت نتیجه تماس</h3><span class="aside">وضعیت فعلی: ' + esc(S.label) + '</span></div>' + (l.attempts >= 2 && o !== 'no_answer' ? '<div class="note warn ctx-warn">' + ic('alert') + '<span>این مشتری ' + fa(l.attempts) + ' بار پاسخ نداده است. ثبت بی‌پاسخ بعدی می‌تواند باعث انتقال پرونده به بایگانی مدیر فروش شود.</span></div>' : '') + '<div class="outcomes" role="group" aria-label="نتیجه تماس">' + outcomes + '</div>' + cond + '</section>' +
      '<details class="sec" open><summary><h3>اطلاعات مشتری</h3><span class="saved" style="margin-right:8px">✓ ذخیره خودکار</span><span class="chev">' + ic('chev') + '</span></summary>' +
      '<div style="display:grid;gap:10px"><div class="row2"><label class="lbl">نام مشتری<input class="input" value="' + esc(l.name) + '"></label><label class="lbl">احتمال فروش ' + hint('برای اولویت‌بندی پیگیری است و وضعیت مشتری را تغییر نمی‌دهد.') + '<select class="input"><option>' + (l.prob || 'انتخاب کنید') + '</option><option>ضعیف</option><option>متوسط</option><option>بالا</option><option>۱۰۰٪</option></select></label></div>' +
      '<div class="row2"><label class="lbl">استان<select class="input"><option>' + esc(l.province) + '</option></select></label><label class="lbl">شهر<select class="input"><option>' + esc(l.city) + '</option></select></label></div></div></details>' +
      '<details class="sec" open><summary><h3>یادداشت</h3><span class="chev">' + ic('chev') + '</span></summary><textarea class="input" placeholder="خلاصه مکالمه، نیاز مشتری…">' + esc(l.note) + '</textarea></details>' +
      '<details class="sec"><summary><h3>تاریخچه</h3><span class="muted" style="margin-right:8px;font-size:var(--t-meta)">' + fa(3) + ' رویداد</span><span class="chev">' + ic('chev') + '</span></summary><ul class="timeline">' +
      (l.attempts ? '<li>تماس بی‌پاسخ (' + fa(l.attempts) + ' از ۳)<span>' + esc(M.user.name) + ' · ۲ روز پیش</span></li>' : '') + '<li>تخصیص به شما<span>سیستم توزیع MIS · ' + esc(l.assigned) + '</span></li><li>ورود شماره از MIS<span>دسته ورودی #۸۸۱۲</span></li></ul></details>' +
      '<details class="sec"' + (invs.length ? ' open' : '') + '><summary><h3>فاکتورهای این مشتری</h3><span class="muted" style="margin-right:8px;font-size:var(--t-meta)">' + fa(invs.length) + '</span><span class="chev">' + ic('chev') + '</span></summary>' +
      (invs.length ? '<div class="mini-list">' + invs.map(function (i) { return '<div class="mini"><div class="grow"><span class="mono">' + fa(i.code) + '</span><div class="muted" style="font-size:var(--t-meta)">' + esc(i.product) + '</div></div>' + money(i.total) + invPill(i) + '</div>'; }).join('') + '</div>' : '<div class="muted">هنوز فاکتوری برای این شماره صادر نشده است.</div>') + '</details></div>' +
      '<div class="dr-foot"><button type="button" class="btn btn-lg btn-primary" data-commit' + (o ? '' : ' disabled') + '>' + esc(ctaText || 'یک نتیجه انتخاب کنید') + '</button><button type="button" class="btn btn-lg btn-ghost" data-close>بستن</button><span class="grow"></span><span class="hint">چیزی بدون تأیید شما ثبت نمی‌شود</span></div>';
  }

  function invoiceDrawer(i) {
    var a = invActions[i.next], stages = [];
    for (var s = 1; s <= i.stages; s++) {
      var cls = s < i.stage || i.paid >= i.total ? 'done' : (s === i.stage ? 'cur' : '');
      stages.push('<div class="stage ' + cls + '"><span class="n">' + fa(s) + '</span><div class="grow">مرحله ' + fa(s) + '<div class="muted" style="font-size:var(--t-meta)">' + (cls === 'done' ? 'پرداخت و تایید شده' : cls === 'cur' ? 'مرحله جاری — قابل پرداخت' : 'بعدی') + '</div></div>' + money(Math.round(i.total / i.stages)) + '</div>');
    }
    var edit = i.paid === 0 && i.status === 'pre_invoice';
    return '<div class="sheet-grip" aria-hidden="true"></div><div class="dr-top"><span>فاکتور</span><span class="grow"></span><button type="button" class="btn btn-ghost btn-icon btn-sm tip" data-tip="بستن (Esc)" data-close aria-label="بستن">' + ic('x') + '</button></div>' +
      '<div class="dr-head"><div class="dr-title"><h2 id="dr-name" class="mono">' + fa(i.code) + '</h2>' + invPill(i) + '</div><div class="dr-meta"><span style="color:var(--text)">' + esc(i.customer) + '</span><span class="mono phone-num">' + fa(i.phone) + '</span><span>' + esc(i.product) + '</span></div>' +
      '<div class="dr-quick"><button type="button" class="btn" data-demo="نمایشی: لینک کپی نشد">' + ic('link') + 'کپی لینک پرداخت</button>' + (edit ? '<button type="button" class="btn" data-demo="نمایشی: ویرایش پیش از پرداخت">' + ic('edit') + 'ویرایش پیش از پرداخت</button>' : '') + '</div></div>' +
      '<div class="dr-body"><section class="sec primary"><div class="sum-grid"><div class="sum"><span>مبلغ کل</span><b>' + money(i.total) + '</b></div><div class="sum"><span>پرداخت‌شده</span><b style="color:var(--green-fg)">' + money(i.paid) + '</b></div><div class="sum"><span>مانده</span><b>' + money(i.total - i.paid) + '</b></div></div>' +
      (i.rejectReason ? '<div class="note danger" style="margin-top:12px">' + ic('alert') + '<span>دلیل رد مالی: ' + esc(i.rejectReason) + '</span></div>' : '') + '</section>' +
      '<details class="sec" open><summary><h3>مراحل پرداخت</h3><span class="muted" style="margin-right:8px;font-size:var(--t-meta)">' + fa(i.stages) + ' مرحله</span><span class="chev">' + ic('chev') + '</span></summary><div class="stages">' + stages.join('') + '</div></details>' +
      '<details class="sec" open><summary><h3>سوابق مالی</h3><span class="chev">' + ic('chev') + '</span></summary><ul class="timeline">' +
      (i.status === 'rejected' ? '<li>رد توسط مالی<span>کارشناس مالی · دیروز ۱۴:۲۰</span></li><li>ارسال فیش برای بررسی<span>' + esc(M.user.name) + ' · ۲ روز پیش</span></li>' : '') +
      (i.status === 'approved' ? '<li>تایید مالی<span>کارشناس مالی · ۱۴۰۵/۰۶/۱۸</span></li>' : '') + '<li>صدور پیش‌فاکتور<span>' + esc(M.user.name) + ' · ۱۴۰۵/۰۶/۱۵</span></li></ul></details></div>' +
      '<div class="dr-foot">' + (a ? '<button type="button" class="btn btn-lg btn-primary" data-commit>' + ic(a.icon) + esc(a.label) + '</button>' : '') + '<button type="button" class="btn btn-lg btn-ghost" data-close>بستن</button><span class="grow"></span></div>';
  }

  function caseDrawer(c) {
    var o = state.outcome, t = CASE_TONE[c.tone] || CASE_TONE.neutral;
    var opts = [{ id: 'no_answer', label: 'جواب نداده', c: 'amber', icon: 'phoneOff', k: '1' }, { id: 'follow_up', label: 'تماس مجدد', c: 'blue', icon: 'clock', hint: 'تاریخ و ساعت لازم', k: '2' }, { id: 'customer_declined', label: 'انصراف', c: 'red', icon: 'xCircle', hint: 'علت الزامی', k: '3' }, { id: 'payment_link', label: 'ادامه — ارسال لینک پرداخت', c: 'green', icon: 'send', hint: 'محصول نهایی و مبلغ مرحله', wide: true, k: '4' }];
    return '<div class="sheet-grip" aria-hidden="true"></div><div class="dr-top"><span>پرونده تبدیل #' + fa(c.id) + '</span><span class="grow"></span><button type="button" class="btn btn-ghost btn-icon btn-sm tip" data-tip="بستن (Esc)" data-close aria-label="بستن">' + ic('x') + '</button></div>' +
      '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">' + esc(c.customer) + '</h2>' + pill(c.result, t.tone, t.icon) + '</div><div class="dr-meta"><span class="phone mono phone-num">' + fa(c.phone) + '</span><span>فاکتور ' + fa(c.invoice) + '</span><span>' + esc(c.option) + '</span></div>' +
      '<div class="dr-facts"><span class="fact-chip">' + ic('history') + 'آخرین فعالیت: دیروز</span><span class="fact-chip">' + ic('layers') + 'مرحله: ' + esc(c.stage) + '</span></div></div>' +
      '<div class="dr-body"><section class="sec primary"><div class="sum-grid" style="margin-bottom:12px"><div class="sum"><span>مبلغ گزینه</span><b>' + money(c.remaining) + '</b></div><div class="sum"><span>پرداخت‌شده</span><b>' + money(0) + '</b></div><div class="sum"><span>مانده</span><b>' + money(c.remaining) + '</b></div></div>' +
      '<div class="sec-h"><h3>نتیجه تماس</h3></div><div class="outcomes" style="grid-template-columns:repeat(3,1fr)">' + opts.map(function (x) { return outcomeBtn(x, o); }).join('') + '</div>' +
      '<div class="cond" data-cond="customer_declined"' + (o === 'customer_declined' ? '' : ' hidden') + '><label class="lbl">علت انصراف <span class="req">*</span><textarea class="input" placeholder="علت دقیق انصراف مشتری"></textarea></label></div>' +
      '<div class="cond" data-cond="follow_up"' + (o === 'follow_up' ? '' : ' hidden') + '><div class="row2"><label class="lbl">تاریخ شمسی <span class="req">*</span><input class="input" placeholder="۱۴۰۵/۰۷/۱۰"></label><label class="lbl">ساعت تهران <span class="req">*</span><span class="time"><input class="input" value="10"> : <input class="input" value="00"></span></label></div></div>' +
      '<div class="cond" data-cond="payment_link"' + (o === 'payment_link' ? '' : ' hidden') + '><div class="row2"><label class="lbl">محصول نهایی<select class="input"><option>' + esc(c.option) + '</option></select></label><label class="lbl">مبلغ این مرحله<input class="input" placeholder="مثلاً ۲۰٬۰۰۰٬۰۰۰"></label></div><div class="note info">' + ic('send') + '<span>لینک پس از تأیید شما ساخته و برای مشتری پیامک می‌شود.</span></div></div>' +
      '</section><details class="sec"><summary><h3>یادداشت و تاریخچه</h3><span class="chev">' + ic('chev') + '</span></summary><ul class="timeline"><li>مشتری تأیید کرد<span>دیروز</span></li><li>تخصیص به شما<span>۱۴۰۵/۰۶/۲۰</span></li></ul></details></div>' +
      '<div class="dr-foot"><button type="button" class="btn btn-lg btn-primary" data-commit' + (o ? '' : ' disabled') + '>' + (o === 'payment_link' ? 'ثبت محصول و ادامه به ارسال لینک' : o ? 'ذخیره نتیجه تماس' : 'یک نتیجه انتخاب کنید') + '</button><button type="button" class="btn btn-lg btn-ghost" data-close>بستن</button></div>';
  }

  function behaviorDrawer(a) {
    return '<div class="sheet-grip" aria-hidden="true"></div><div class="dr-top"><span>روند رفتار مشتری</span><span class="grow"></span><button type="button" class="btn btn-ghost btn-icon btn-sm" data-close aria-label="بستن">' + ic('x') + '</button></div>' +
      '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">' + esc(a.name) + '</h2>' + (a.hot ? '<span class="hot">فعال</span>' : '') + '</div><div class="dr-meta"><span class="mono phone-num">' + fa(a.phone) + '</span><span>فاکتور ' + fa(a.invoice) + '</span></div></div>' +
      '<div class="dr-body"><section class="sec"><ul class="timeline"><li>' + esc(a.last) + '<span>' + esc(a.when) + '</span></li><li>مشاهده اطلاعات محصول<span>۲ ساعت پیش</span></li><li>کلیک روی گردونه<span>دیروز</span></li><li>باز کردن لینک فاکتور<span>دیروز</span></li><li>ارسال پیامک پیش‌فاکتور<span>۳ روز پیش</span></li></ul></section></div>' +
      '<div class="dr-foot"><button type="button" class="btn btn-lg" data-go-invoice="' + a.invoice + '">مشاهده فاکتور</button><button type="button" class="btn btn-lg btn-ghost" data-close>بستن</button></div>';
  }

  var lastOpener = null;
  function openDrawer(kind, id, keepOutcome) {
    var same = state.drawer && state.drawer.kind === kind && state.drawer.id === id;
    if (!same && !keepOutcome) state.outcome = null;
    if (!state.drawer) lastOpener = document.activeElement;
    state.drawer = { kind: kind, id: id };
    var el = $('#drawer'), html = '';
    if (kind === 'lead') html = leadDrawer(M.leads.filter(function (l) { return l.id === id; })[0]);
    if (kind === 'invoice') html = invoiceDrawer(M.invoices.filter(function (i) { return i.code === id; })[0]);
    if (kind === 'case') html = caseDrawer(M.conversions.filter(function (c) { return c.id === id; })[0]);
    if (kind === 'behavior') html = behaviorDrawer(M.customerActions.filter(function (a) { return a.invoice === id; })[0]);
    if (state.sim === 'drawerError') html = drawerState('error');
    else if (state.sim === 'loading') html = drawerState('loading');
    el.innerHTML = html; el.setAttribute('aria-hidden', 'false');
    document.body.classList.add('drawer-open');
    if (document.body.classList.contains('kb') && !el.contains(document.activeElement)) { el.tabIndex = -1; el.focus({ preventScroll: true }); }
    var attr = { lead: 'lead', invoice: 'inv', 'case': 'case', behavior: 'behavior' }[kind];
    $$('.tbl tbody tr').forEach(function (tr) { var on = String(tr.getAttribute('data-' + attr)) === String(id); tr.classList.toggle('sel', on); tr.setAttribute('aria-selected', String(on)); });
    if (kind === 'lead') { var i = currentLeads().map(function (l) { return l.id; }).indexOf(id); if (i > -1) setCursor(i, false); }
  }
  function drawerState(kind) {
    var top = '<div class="sheet-grip" aria-hidden="true"></div><div class="dr-top"><span>جزئیات</span><span class="grow"></span><button type="button" class="btn btn-ghost btn-icon btn-sm tip" data-tip="بستن (Esc)" data-close aria-label="بستن">' + ic('x') + '</button></div>';
    if (kind === 'loading') return top + '<div class="dr-head" aria-busy="true"><div class="sk sk-line" style="width:40%;height:18px"></div><div class="sk sk-line" style="width:65%;margin-top:12px"></div><div class="sk-row" style="margin-top:12px"><span class="sk sk-chip"></span><span class="sk sk-chip"></span><span class="sk sk-chip"></span></div></div><div class="dr-body"><div class="sec"><div class="sk sk-line" style="width:30%"></div><div class="sk-grid"><span class="sk sk-box"></span><span class="sk sk-box"></span><span class="sk sk-box"></span><span class="sk sk-box"></span></div></div><div class="sec"><div class="sk sk-line" style="width:80%"></div><div class="sk sk-line" style="width:60%;margin-top:8px"></div></div></div><span class="sr" role="status">در حال بارگذاری اطلاعات مشتری</span>';
    return top + '<div class="dr-body">' + stateBlock('error', 'اطلاعات مشتری بارگذاری نشد', 'اتصال یا سرور پاسخ نداد. پنجره باز می‌ماند تا دوباره تلاش کنید.', '<button type="button" class="btn btn-primary" data-retry="drawer">' + ic('refresh') + 'تلاش مجدد</button>') + '</div>';
  }
  function stateBlock(kind, title, text, actions) {
    var icon = { error: 'alert', empty: 'check', noresult: 'search', info: 'info' }[kind];
    return '<div class="state state-' + kind + '" role="' + (kind === 'error' ? 'alert' : 'status') + '"><div class="state-ico">' + ic(icon) + '</div><b>' + esc(title) + '</b>' + (text ? '<span>' + text + '</span>' : '') + (actions ? '<div class="state-acts">' + actions + '</div>' : '') + '</div>';
  }
  function closeDrawer() {
    state.drawer = null; state.outcome = null;
    document.body.classList.remove('drawer-open'); $('#drawer').setAttribute('aria-hidden', 'true');
    $$('.tbl tbody tr.sel').forEach(function (tr) { tr.classList.remove('sel'); tr.setAttribute('aria-selected', 'false'); });
    var row = $('.tbl tbody tr.cursor'); if (row) row.focus(); else if (lastOpener && lastOpener.focus && document.contains(lastOpener)) lastOpener.focus();
  }
  function rerenderDrawer() { if (state.drawer) openDrawer(state.drawer.kind, state.drawer.id, true); }
  function setCursor(i, focus) {
    var rows = $$('.tbl tbody tr[data-lead]'); if (!rows.length) return;
    state.cursor = Math.max(0, Math.min(rows.length - 1, i));
    rows.forEach(function (r, n) { r.classList.toggle('cursor', n === state.cursor); r.tabIndex = n === state.cursor ? 0 : -1; });
    if (focus !== false) { rows[state.cursor].focus({ preventScroll: true }); rows[state.cursor].scrollIntoView({ block: 'nearest' }); }
  }

  /* ---------- Menus, palette, toast ---------- */
  // Toasts: short, non-blocking, auto-dismiss (4.5s, paused on hover), manually closable, max 3 stacked.
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
  function closeMenu() { var m = $('.menu'); if (m) { m.remove(); $('#user-btn').setAttribute('aria-expanded', 'false'); } }
  function placeMenu(m, anchor) {
    var r = anchor.getBoundingClientRect(), top = r.bottom + 6;
    if (top + m.offsetHeight > innerHeight - 8) top = Math.max(8, r.top - m.offsetHeight - 6);
    m.style.top = top + 'px'; m.style.left = Math.max(8, Math.min(r.left, innerWidth - m.offsetWidth - 8)) + 'px';
  }
  function openMenu(anchor, items) {
    closeMenu();
    var m = document.createElement('div'); m.className = 'menu'; m.setAttribute('role', 'menu');
    m.innerHTML = items.map(function (it) { return it === '-' ? '<hr>' : '<button type="button" class="mi' + (it.danger ? ' danger' : '') + '" role="menuitem" data-demo="' + esc(it.demo || ('نمایشی: «' + it.label + '»')) + '"' + (it.go ? ' data-go="' + it.go + '"' : '') + '>' + ic(it.icon) + esc(it.label) + '</button>'; }).join('');
    document.body.appendChild(m); placeMenu(m, anchor);
    var first = m.querySelector('button'); if (first) first.focus();
  }
  function userMenuHtml() {
    var seg = function (key, opts) { return '<div class="opt-seg" role="group">' + opts.map(function (o) { return '<button type="button" data-pref="' + key + '" data-val="' + o[0] + '" aria-pressed="' + (prefs[key] === o[0]) + '">' + o[1] + '</button>'; }).join('') + '</div>'; };
    return '<div class="mh">پوسته</div>' + seg('theme', [['light', 'روشن'], ['dim', 'کم‌نور'], ['dark', 'تیره']]) +
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

  /* Command palette */
  var pal = { items: [], sel: 0 };
  function palItems(q) {
    q = (q || '').trim();
    var norm = function (s) { return String(s).replace(/[۰-۹]/g, function (d) { return '۰۱۲۳۴۵۶۷۸۹'.indexOf(d); }); };
    var nq = norm(q), out = [];
    var match = function (s) { return !nq || norm(s).indexOf(nq) > -1; };
    M.leads.forEach(function (l) { if (nq && (match(l.name) || match(l.phone))) out.push({ g: 'مشتریان', icon: 'users', label: l.name, meta: fa(l.phone) + ' · ' + LEAD[l.status].label, run: function () { go('leads'); state.leadQueue = 'all'; render(); openDrawer('lead', l.id); } }); });
    M.invoices.forEach(function (i) { if (nq && match(i.code)) out.push({ g: 'فاکتورها', icon: 'receipt', label: 'فاکتور ' + fa(i.code), meta: i.customer + ' · ' + i.statusLabel, run: function () { go('invoices'); openDrawer('invoice', i.code); } }); });
    NAV.filter(function (n) { return !n.sep; }).forEach(function (n) { if (match('رفتن به ' + n.label)) out.push({ g: 'رفتن به', icon: n.icon, label: n.label, run: function () { go(n.id); } }); });
    QUEUE_ORDER.forEach(function (k) { var lbl = k === 'all' ? 'همه شماره‌ها' : 'صف ' + LEAD[k].label; if (match(lbl)) out.push({ g: 'صف‌ها', icon: k === 'all' ? 'rows' : LEAD[k].icon, label: lbl, meta: 'کلید ' + fa(k === 'all' ? '0' : LEAD[k].key), run: function () { go('leads'); state.leadQueue = k; render(); } }); });
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
    $('.palette').innerHTML = '<div class="pal-in">' + ic('search') + '<input type="text" id="pal-q" placeholder="نام یا شماره مشتری، کد فاکتور، یا یک فرمان…" aria-label="جستجو و فرمان‌ها" autocomplete="off"><span class="kbd">Esc</span></div><div class="pal-list" role="listbox"></div>' +
      '<div class="pal-foot"><span><span class="kbd">↑↓</span> انتخاب</span><span><span class="kbd">Enter</span> اجرا</span><span><span class="kbd">/</span> یا <span class="kbd">Ctrl K</span> باز کردن</span><span><span class="kbd">J</span><span class="kbd">K</span> مشتری بعدی/قبلی</span><span><span class="kbd">۰–۶</span> تغییر صف</span><span><span class="kbd">F</span> جستجو در صف</span></div>';
    var input = $('#pal-q'); input.value = q || ''; renderPalette(input.value); input.focus();
  }
  function closePalette() { $('#palette').hidden = true; if (lastOpener && lastOpener.focus && document.contains(lastOpener)) lastOpener.focus(); }
  function runPal(n) { var it = pal.items[n]; if (!it) return; closePalette(); it.run(); }

  /* ---------- Render & events ---------- */
  var EMPTY = {
    leads: ['در این صف موردی وجود ندارد', 'شماره‌ای با این وضعیت ندارید.', '<button type="button" class="btn btn-soft" data-empty-all>مشاهده همه شماره‌ها</button>'],
    invoices: ['فاکتوری در این وضعیت وجود ندارد', 'وضعیت دیگری را انتخاب کنید یا جستجو را تغییر دهید.', '<button type="button" class="btn btn-soft" data-empty-all>همه فاکتورها</button>'],
    conversions: ['پرونده‌ای برای پیگیری وجود ندارد', 'پرونده‌هایی که «تبدیل توسط خودم» را برایشان انتخاب کنید اینجا می‌آیند.', ''],
    wallet: ['در این بازه تراکنشی ثبت نشده است', 'بازه تاریخ یا نوع پرداخت را تغییر دهید.', ''],
    behavior: ['رفتاری ثبت نشده است', 'هنوز مشتری‌ای لینک فاکتور را باز نکرده است.', '']
  };
  function skeletonPage() {
    var rows = '';
    for (var i = 0; i < 7; i++) rows += '<div class="sk-trow"><span class="sk sk-pill"></span><span class="sk sk-line" style="width:18%"></span><span class="sk sk-line" style="width:8%"></span><span class="sk sk-line" style="width:10%"></span><span class="sk sk-line" style="width:14%"></span><span class="grow"></span><span class="sk sk-btn"></span></div>';
    var kpi = '<div class="kpi"><span class="sk sk-line" style="width:70px"></span><span class="sk sk-line" style="width:36px;height:16px;margin-top:8px"></span></div>';
    return '<div class="page-head" aria-busy="true"><div><div class="sk sk-line" style="width:140px;height:20px"></div></div><div class="spacer"></div><div class="kpis sk-kpis">' + kpi + kpi + kpi + kpi + '</div></div>' +
      '<section class="panel" aria-busy="true"><div class="queues"><span class="sk sk-q"></span><span class="sk sk-q"></span><span class="sk sk-q"></span><span class="sk sk-q"></span></div><div class="toolbar"><span class="sk sk-input"></span></div>' + rows + '</section><span class="sr" role="status">در حال بارگذاری</span>';
  }
  function applyTableState() {
    var panel = $('#ws .panel'); if (!panel) return;
    var area = panel.querySelector('.tbl-wrap') || panel.querySelector('.empty'); if (!area) return;
    var html = '';
    if (state.sim === 'tableError') html = stateBlock('error', 'امکان دریافت لیست وجود ندارد', 'ممکن است اتصال لحظه‌ای قطع شده باشد. بقیه صفحه همچنان قابل استفاده است.', '<button type="button" class="btn btn-primary" data-retry="table">' + ic('refresh') + 'تلاش مجدد</button>');
    else if (state.sim === 'empty') { var e = EMPTY[state.view] || EMPTY.leads; html = stateBlock('empty', e[0], e[1], e[2]); }
    else if (state.sim === 'noresult') {
      var q = '09123456'; var f = $('#queue-filter'); if (f) f.value = fa(q);
      html = stateBlock('noresult', 'نتیجه‌ای برای «' + fa(q) + '» پیدا نشد', 'شماره را کامل‌تر وارد کنید یا جستجو را در همه صف‌ها انجام دهید.', '<button type="button" class="btn btn-soft" data-clear-search>پاک کردن جستجو</button><button type="button" class="btn" data-clear-filters>پاک کردن فیلترها</button>');
    } else return;
    $$('.card-list,.tfoot', panel).forEach(function (n) { n.remove(); });
    var d = document.createElement('div'); d.innerHTML = html; area.replaceWith(d.firstChild);
  }
  function render() {
    renderNav();
    $('#offline-bar').hidden = !state.offline; $('#net-chip').hidden = !state.offline;
    if (state.offline) $('#offline-bar').innerHTML = ic('wifiOff') + '<span><b>اتصال اینترنت قطع شده است.</b> برخی اطلاعات ممکن است به‌روز نباشند؛ تغییرات پس از اتصال دوباره ارسال می‌شوند.</span><button type="button" class="btn btn-sm" data-retry="net">' + ic('refresh') + 'تلاش مجدد</button>';
    if (state.sim === 'pageError') { $('#ws').innerHTML = '<section class="panel page-error">' + stateBlock('error', 'خطا در دریافت اطلاعات', 'پاسخی از سرور دریافت نشد. چند لحظه بعد دوباره تلاش کنید؛ داده‌ای از دست نرفته است.', '<button type="button" class="btn btn-primary" data-retry="page">' + ic('refresh') + 'تلاش مجدد</button>') + '</section>'; return; }
    if (state.sim === 'loading') { $('#ws').innerHTML = skeletonPage(); return; }
    $('#ws').innerHTML = (V[state.view] || V.leads)();
    applyTableState();
    // Label cells from their column header so non-card tables can stack on phones (CSS below 760px).
    $$('#ws .panel:not(.has-cards) .tbl').forEach(function (t) {
      var heads = $$('thead th', t).map(function (h) { return h.textContent.trim(); });
      $$('tbody tr', t).forEach(function (r) { Array.prototype.forEach.call(r.children, function (td, i) { if (heads[i]) td.setAttribute('data-l', heads[i]); }); });
      t.classList.add('stackable');
    });
    if (state.view === 'leads') setCursor(state.cursor, false);
    if (state.drawer) openDrawer(state.drawer.kind, state.drawer.id, true);
  }
  function go(view) { if (state.drawer) closeDrawer(); closeMenu(); state.view = view; state.cursor = 0; history.replaceState(null, '', location.pathname + location.search.replace(/([?&])view=[^&]*/, '$1view=' + view) + '#' + view); render(); window.scrollTo(0, 0); }
  // From the keyboard the focus goes to the first row so J/K keep working; from a click it stays on the tab.
  function switchQueue(k, fromKeys) { state.leadQueue = k; state.cursor = 0; if (state.drawer) closeDrawer(); render(); if (fromKeys) setCursor(0); else { var b = $('[data-lead-queue="' + k + '"]'); if (b) b.focus(); } }

  document.addEventListener('click', function (e) {
    var t = e.target.closest('button, a, tr, .lcard, [data-palette-close]');
    if (!t) { closeMenu(); return; }
    if (t.closest('.palette-wrap')) {
      if (t.matches('[data-palette-close]')) return closePalette();
      if (t.matches('[data-pal]')) return runPal(Number(t.getAttribute('data-pal')));
      return;
    }
    if (t.matches('[data-pref]')) { setPref(t.getAttribute('data-pref'), t.getAttribute('data-val')); var m = $('#user-menu'); if (m) { m.innerHTML = userMenuHtml(); var f = m.querySelector('[data-pref="' + t.getAttribute('data-pref') + '"][aria-pressed="true"]'); if (f) f.focus(); } return; }
    if (t.matches('[data-toggle-focus]')) { setPref('focus', !prefs.focus); var um = $('#user-menu'); if (um) um.innerHTML = userMenuHtml(); return; }
    if (t.matches('[data-open-shortcuts]')) { closeMenu(); if (window.SNHelp) window.SNHelp.openHelp('keys'); return; }
    if (!t.closest('.menu') && !t.matches('[data-more]') && !t.matches('#user-btn')) closeMenu();
    if (t.matches('#user-btn')) { e.stopPropagation(); if ($('#user-menu')) closeMenu(); else openUserMenu(); return; }
    if (t.matches('[data-palette]')) return openPalette('');
    if (t.matches('[data-view]')) { e.preventDefault(); return go(t.getAttribute('data-view')); }
    if (t.matches('[data-go]')) { e.preventDefault(); return go(t.getAttribute('data-go')); }
    if (t.matches('[data-more]')) {
      e.preventDefault(); e.stopPropagation();
      var kind = t.getAttribute('data-more');
      if (kind === '1') { openMenu(t, [{ label: 'نیاز به اقدام مجدد', icon: 'repeat', go: 'repeat' }, { label: 'رفتار مشتریان', icon: 'activity', go: 'behavior' }, { label: 'درخواست شماره', icon: 'plus', go: 'request' }, '-']); var mm = $('.menu'); mm.insertAdjacentHTML('beforeend', userMenuHtml()); mm.id = 'user-menu'; }
      else if (kind === 'lead') openMenu(t, [{ label: 'کپی شماره', icon: 'copy' }, { label: 'مشاهده تاریخچه', icon: 'history' }, { label: 'صدور پیش‌فاکتور', icon: 'file' }]);
      else if (kind === 'invoice') {
        var inv = M.invoices.filter(function (i) { return i.code === t.getAttribute('data-id'); })[0], items = [{ label: 'جزئیات و سوابق مالی', icon: 'history' }];
        if (inv.paid === 0 && inv.status === 'pre_invoice') items.push({ label: 'ویرایش پیش از پرداخت', icon: 'edit' });
        if (['approved', 'cancelled'].indexOf(inv.status) === -1) { items.push({ label: 'ارسال مجدد لینک پرداخت', icon: 'send' }); items.push({ label: 'کپی لینک پرداخت', icon: 'link' }); }
        openMenu(t, items);
      }
      return;
    }
    if (t.matches('[data-retry]')) {
      var r = t.getAttribute('data-retry');
      if (r === 'net') { state.offline = false; state.sim = null; render(); return toast('اتصال برقرار شد؛ اطلاعات به‌روز شد.', 'success'); }
      if (r === 'drawer') { state.sim = null; return openDrawer(state.drawer.kind, state.drawer.id, true); }
      state.sim = 'loading'; render(); setTimeout(function () { state.sim = null; render(); toast('اطلاعات دوباره بارگذاری شد.', 'success'); }, 900); return;
    }
    if (t.matches('[data-empty-all]')) { state.sim = null; if (state.view === 'invoices') { state.invQueue = 'all'; return render(); } return switchQueue('all'); }
    if (t.matches('[data-clear-search],[data-clear-filters]')) { state.sim = null; render(); return toast(t.matches('[data-clear-search]') ? 'جستجو پاک شد.' : 'فیلترها پاک شدند.', 'info'); }
    if (t.matches('#net-chip')) { state.offline = false; state.sim = null; render(); return toast('اتصال برقرار شد؛ اطلاعات به‌روز شد.', 'success'); }
    if (t.matches('[data-copy]')) { e.stopPropagation(); return toast('نمایشی: شماره ' + fa(t.getAttribute('data-copy')) + ' کپی شد'); }
    if (t.matches('[data-demo]')) { e.preventDefault(); e.stopPropagation(); toast(t.getAttribute('data-demo')); return closeMenu(); }
    if (t.matches('[data-lead-queue]')) return switchQueue(t.getAttribute('data-lead-queue'));
    if (t.matches('[data-inv-queue]')) { state.invQueue = t.getAttribute('data-inv-queue'); if (state.drawer) closeDrawer(); return render(); }
    if (t.matches('[data-kpi-filter]')) { var f2 = t.getAttribute('data-kpi-filter'); if (f2.indexOf('inv:') === 0) { state.invQueue = f2.slice(4); return render(); } return switchQueue(f2); }
    if (t.matches('[data-wallet]')) { state.walletFilter = t.getAttribute('data-wallet'); return render(); }
    if (t.matches('[data-open-lead]')) { e.stopPropagation(); return openDrawer('lead', Number(t.getAttribute('data-open-lead'))); }
    if (t.matches('[data-open-inv]')) { e.stopPropagation(); return openDrawer('invoice', t.getAttribute('data-open-inv')); }
    if (t.matches('[data-open-case]')) { e.stopPropagation(); return openDrawer('case', Number(t.getAttribute('data-open-case'))); }
    if (t.matches('[data-open-behavior]')) { e.stopPropagation(); return openDrawer('behavior', t.getAttribute('data-open-behavior')); }
    if (t.matches('[data-go-invoice]')) { var code = t.getAttribute('data-go-invoice'); go('invoices'); return openDrawer('invoice', code); }
    if (t.matches('[data-outcome]')) return pickOutcome(t.getAttribute('data-outcome'));
    if (t.matches('[data-commit]')) { if (!t.disabled) commit(); return; }
    if (t.matches('[data-close]')) return closeDrawer();
    if (t.matches('[data-step]')) return step(Number(t.getAttribute('data-step')));
    if (t.matches('tr[data-lead], .lcard[data-lead]')) return openDrawer('lead', Number(t.getAttribute('data-lead')));
    if (t.matches('tr[data-inv], .lcard[data-inv]')) return openDrawer('invoice', t.getAttribute('data-inv'));
    if (t.matches('tr[data-case]')) return openDrawer('case', Number(t.getAttribute('data-case')));
    if (t.matches('tr[data-behavior]')) return openDrawer('behavior', t.getAttribute('data-behavior'));
  });
  function pickOutcome(id) {
    state.outcome = id; rerenderDrawer();
    var c = $('.cond:not([hidden])'), inp = c && c.querySelector('input,textarea,select');
    if (inp) inp.focus(); else { var b = $('[data-outcome="' + id + '"]'); if (b) b.focus(); }
  }
  // Required inputs per outcome (mirrors the real rules: callback needs date+time, no-purchase/decline need a reason).
  var REQUIRED = { callback: '[data-cond="callback"] input', not_purchased: '[data-cond="not_purchased"] textarea', customer_declined: '[data-cond="customer_declined"] textarea', follow_up: '[data-cond="follow_up"] input' };
  function validateOutcome() {
    $$('#drawer .field-err').forEach(function (n) { n.remove(); });
    $$('#drawer [aria-invalid="true"]').forEach(function (n) { n.removeAttribute('aria-invalid'); n.removeAttribute('aria-describedby'); });
    var sel = REQUIRED[state.outcome]; if (!sel) return true;
    var bad = $$('#drawer ' + sel).filter(function (f) { return !String(f.value || '').trim(); });
    bad.forEach(function (f, n) {
      f.setAttribute('aria-invalid', 'true'); f.setAttribute('aria-describedby', 'err-' + n);
      var e = document.createElement('div'); e.className = 'field-err'; e.id = 'err-' + n; e.innerHTML = ic('alert') + '<span>' + (f.tagName === 'TEXTAREA' ? 'نوشتن دلیل برای این نتیجه الزامی است.' : 'این فیلد الزامی است.') + '</span>';
      (f.closest('.time') || f).insertAdjacentElement('afterend', e);
    });
    if (bad.length) { bad[0].focus(); return false; }
    return true;
  }
  function commit() {
    if (!validateOutcome()) return;
    var msg = state.drawer && state.drawer.kind === 'invoice' ? 'اقدام فاکتور ثبت شد.' : state.outcome === 'success' ? 'پیش‌فاکتور با موفقیت صادر شد.' : 'نتیجه تماس با موفقیت ثبت شد.';
    toast(msg + ' (نمایشی — چیزی ذخیره نشد)', 'success');
  }
  function step(d) {
    if (!state.drawer || state.drawer.kind !== 'lead') { setCursor(state.cursor + d); return; }
    var list = currentLeads(), i = list.map(function (l) { return l.id; }).indexOf(state.drawer.id);
    if (i < 0) list = M.leads, i = list.map(function (l) { return l.id; }).indexOf(state.drawer.id);
    var n = list[(i + d + list.length) % list.length]; state.outcome = null; openDrawer('lead', n.id);
  }
  $('#scrim').addEventListener('click', closeDrawer);
  $('#palette').addEventListener('input', function (e) { if (e.target.id === 'pal-q') { pal.sel = 0; renderPalette(e.target.value); } });

  // Keyboard-mode indicator: the row cursor ring only shows while navigating by keyboard.
  document.addEventListener('mousedown', function () { document.body.classList.remove('kb'); });
  document.addEventListener('keydown', function (e) {
    if (/^(Tab|Arrow|j|k|J|K|ت|ن|Enter)/.test(e.key)) document.body.classList.add('kb');
    var a = document.activeElement || {}, typing = /INPUT|TEXTAREA|SELECT/.test(a.tagName || '') || a.isContentEditable;
    // Letter/number shortcuts are ignored while a control (button, link, field) has focus; rows and the page are fine.
    var onControl = typing || !!(a.closest && a.closest('button, a[href], [role="button"], [role="menuitem"], [role="option"], summary'));
    var paletteOpen = !$('#palette').hidden;
    if (paletteOpen) {
      if (e.key === 'Escape') { e.preventDefault(); return closePalette(); }
      if (e.key === 'ArrowDown' || e.key === 'ArrowUp') { e.preventDefault(); pal.sel = (pal.sel + (e.key === 'ArrowDown' ? 1 : -1) + pal.items.length) % Math.max(1, pal.items.length); return renderPalette($('#pal-q').value); }
      if (e.key === 'Enter') { e.preventDefault(); return runPal(pal.sel); }
      if (e.key === 'Tab') { e.preventDefault(); $('#pal-q').focus(); }
      return;
    }
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') { e.preventDefault(); return openPalette(''); }
    if (e.key === 'Escape') { if ($('.menu')) { closeMenu(); $('#user-btn').focus(); return; } if (state.drawer) return closeDrawer(); if (typing) a.blur(); return; }
    if (e.ctrlKey || e.metaKey || e.altKey) return;
    if (e.key === '?' && !typing) { e.preventDefault(); if (window.SNHelp) window.SNHelp.openHelp('keys'); return; }
    if (onControl) return;
    var k = e.key.toLowerCase();
    var faDigit = '۰۱۲۳۴۵۶۷۸۹'.indexOf(e.key); if (faDigit > -1) k = String(faDigit);
    if (k === '/') { e.preventDefault(); return openPalette(''); }
    if (k === 'f' || e.key === 'ب') { var qf = $('#queue-filter'); if (qf) { e.preventDefault(); qf.focus(); } return; }
    if (k === 'j' || k === 'ت') { e.preventDefault(); return step(1); }
    if (k === 'k' || k === 'ن') { e.preventDefault(); return step(-1); }
    if (/^[0-9]$/.test(k)) {
      if (state.drawer && (state.drawer.kind === 'lead' || state.drawer.kind === 'case')) {
        var btn = $$('[data-outcome]')[Number(k) - 1]; if (btn) { e.preventDefault(); pickOutcome(btn.getAttribute('data-outcome')); }
        return;
      }
      if (state.view === 'leads') { var q = k === '0' ? 'all' : QUEUE_ORDER[Number(k) - 1]; if (q && q !== 'all' || k === '0') { e.preventDefault(); switchQueue(q, true); } }
      return;
    }
    if (e.key === 'Enter') {
      if (state.drawer) { var c = $('[data-commit]'); if (c && !c.disabled) { e.preventDefault(); commit(); } return; }
      if (a.matches && a.matches('tr[data-lead]')) { e.preventDefault(); openDrawer('lead', Number(a.getAttribute('data-lead'))); }
    }
  });
  window.addEventListener('hashchange', function () { var v = location.hash.slice(1); if (V[v] && v !== state.view) go(v); });

  /* ---------- Tooltip (shared): one fixed layer for every [data-tip]; hover after a short delay, keyboard focus at once ---------- */
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

  // Small API used by help.js (tours, help center, demo states). Prototype only.
  window.SNB = {
    state: state, prefs: prefs, setPref: setPref, render: render, go: go, switchQueue: switchQueue, toast: toast,
    openDrawer: openDrawer, closeDrawer: closeDrawer, closeMenu: closeMenu, openPalette: openPalette,
    setSim: function (sim) { state.sim = sim; state.offline = sim === 'offline'; if (state.drawer && sim !== 'drawerError') closeDrawer(); render(); },
    firstLeadId: function () { var l = currentLeads()[0] || M.leads[0]; return l.id; },
    firstInvoice: function () { return M.invoices[0].code; },
    firstCase: function () { return M.conversions[0].id; }
  };
  render();
  // Deep links used for screenshots: ?lead=… &outcome=… · ?inv=… · ?case=… · ?menu=invoice|user · ?palette=1
  if (qs.get('lead')) openDrawer('lead', Number(qs.get('lead')), true);
  if (qs.get('inv')) openDrawer('invoice', qs.get('inv'), true);
  if (qs.get('case')) openDrawer('case', Number(qs.get('case')), true);
  if (qs.get('menu') === 'user') openUserMenu();
  else if (qs.get('menu')) { var mb = $('[data-more="' + qs.get('menu') + '"]'); if (mb) mb.click(); }
  if (qs.get('palette')) openPalette(qs.get('palette') === '1' ? '' : qs.get('palette'));
})();
