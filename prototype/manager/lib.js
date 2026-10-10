/* Sales Manager — role layer, part 1: state, namespaced dictionaries, shared presentation helpers.
   Design prototype; mock data only. Nothing here assigns, returns, approves, posts, sends or archives anything.
   Product boundaries: SALES-MANAGER-PRODUCT-SPEC.md + CRM-CROSS-ROLE-ARCHITECTURE-V1.md + GATE-0-PRODUCT-INVARIANTS.md. */
(function () {
  'use strict';
  var C = window.CRM, M = window.MGR, h = C.h;
  var esc = h.esc, fa = h.fa, num = h.num, ic = h.ic, pill = h.pill, hint = h.hint;

  var X = window.MGRX = { C: C, M: M, h: h, V: {}, D: {} };

  /* ---------- Role state ---------- */
  X.st = {
    pm: 'unit', pb: 'current', pscope: null, tb: 'event',             // performance explorer: grain / attribution basis / drilldown scope / time basis
    aq: 'assign', amode: 'select', asel: {}, recip: null, recips: {}, per: 3, skipReason: '', skipOk: false, // distribution
    rf: 'all', rsel: {}, iq: 'action', hq: 'review', xq: 'pending', ar: 'all', xDone: {}, custSt: 'partial',
    rbasis: 'event', report: null, rteam: 'all',
    flow: null, live: null, hrForm: null, hrErr: null, hrDone: {}, xErr: {}, xFlow: null, resent: {}
  };

  /* ---------- Namespaced status dictionaries (label + tone + icon; never colour alone) ---------- */
  X.ELIG = { ok: { label: 'قابل تخصیص', tone: 'teal', icon: 'checkCircle' }, blocked: { label: 'مسدود', tone: 'red', icon: 'lock' }, unknown: { label: 'نیاز به تطبیق', tone: 'amber', icon: 'question' }, conflict: { label: 'تغییر کرده', tone: 'orange', icon: 'swap' } };
  X.RET = { ok: { label: 'واجد شرایط (پیش‌نمایش)', tone: 'teal', icon: 'checkCircle' }, blocked: { label: 'مسدود', tone: 'red', icon: 'lock' }, unknown: { label: 'نامعلوم · نیاز به تطبیق', tone: 'amber', icon: 'question' }, conflict: { label: 'تغییر کرده', tone: 'orange', icon: 'swap' } };
  X.INVS = { pre: { label: 'پیش‌فاکتور', tone: 'violet', icon: 'file' }, staged: { label: 'مرحله‌ای', tone: 'teal', icon: 'layers' }, completed: { label: 'تکمیل‌شده', tone: 'green', icon: 'checkCircle' }, cancelled: { label: 'لغو شده', tone: 'slate', icon: 'ban' }, mismatch: { label: 'مغایرت', tone: 'amber', icon: 'question' } };
  X.STG = { pending: 'در انتظار پرداخت', review: 'رسید در انتظار بررسی', rejected: 'رد شده', approved: 'تأییدشده', cancelled: 'لغو' };
  X.EVID = { none: '—', receipt: 'رسید ثبت‌شده', online: 'پرداخت آنلاین (درگاه)' };
  X.REV = { none: null, pending: { label: 'در انتظار مالی', tone: 'orange', icon: 'hourglass' }, approved: { label: 'تأیید مالی', tone: 'green', icon: 'checkCircle' }, rejected: { label: 'رد مالی', tone: 'red', icon: 'alert' } };
  // Bulk outcome language (shared): requested · eligible · applied · skipped · rejected · failed · unknown.
  X.OUT = {
    ok: { label: 'اعمال شد', tone: 'green', icon: 'checkCircle' }, skipped: { label: 'ارسال نشد', tone: 'orange', icon: 'swap' }, rejected: { label: 'ردشده توسط سامانه', tone: 'red', icon: 'ban' },
    failed: { label: 'ناموفق · قابل تکرار', tone: 'red', icon: 'refresh' }, unknown: { label: 'نامعلوم', tone: 'amber', icon: 'question' }
  };
  X.OPS = { complete: ['کامل', 'green', 'checkCircle'], partial: ['ناقص', 'orange', 'split'], failed: ['ناموفق', 'red', 'xCircle'], unknown: ['نامعلوم', 'amber', 'question'] };
  X.HRS = { pending_review: { label: 'در انتظار بررسی مرحله‌ای', tone: 'orange', icon: 'hourglass' }, pending_hr: { label: 'در انتظار منابع انسانی', tone: 'orange', icon: 'hourglass' }, approved: { label: 'اعمال‌شده', tone: 'green', icon: 'checkCircle' }, rejected: { label: 'رد شده', tone: 'red', icon: 'xCircle' }, failed: { label: 'اعمال ناموفق', tone: 'red', icon: 'alert' }, stepdone: { label: 'مرحله شما تأیید شد', tone: 'teal', icon: 'checkCircle' } };
  X.XRS = { pending: { label: 'در انتظار تصمیم شما', tone: 'orange', icon: 'hourglass' }, approved: { label: 'تأیید شد', tone: 'green', icon: 'checkCircle' }, rejected: { label: 'رد شد', tone: 'red', icon: 'xCircle' }, unknown: { label: 'نتیجه ثبت نامعلوم', tone: 'amber', icon: 'question' } };
  X.ARC = { noanswer: ['بی‌پاسخ', 'slate', 'phone'], assess: ['ارزیابی پرداخت‌نشده', 'amber', 'receipt'], sub: ['اشتراک پرداخت‌نشده', 'violet', 'receipt'], product: ['محصول پرداخت‌نشده', 'blue', 'receipt'], productx: ['محصول* پرداخت‌نشده', 'teal', 'receipt'] };

  /* ---------- Data helpers ---------- */
  var sim = X.sim = function () { return C.state.sim; };
  X.seller = function (id) { return M.sellers.filter(function (s) { return s.id === id; })[0]; };
  X.team = function (id) { return M.teams.filter(function (t) { return t.id === id; })[0]; };
  X.unit = function (id) { return M.units.filter(function (u) { return u.id === id; })[0]; };
  X.unitOfTeam = function (tid) { return tid === 'direct' ? 'M' : X.team(tid).parent; };
  X.unitName = function (id) { return id === 'M' ? 'مستقیم زیر نظر شما' : 'سرپرست ارشد ' + X.unit(id).name; };
  X.teamName = function (id) { return id === 'direct' ? 'فروشنده مستقیم (بدون سرپرست میانی)' : id ? 'تیم ' + X.team(id).sup : 'نامشخص'; };
  X.sellersOf = function (id) { return M.sellers.filter(function (s) { return s.team === id; }); };
  X.teamsOf = function (uid) { return M.teams.filter(function (t) { return t.parent === uid; }); };
  X.sellersOfUnit = function (uid) { return M.sellers.filter(function (s) { return X.unitOfTeam(s.team) === uid; }); };
  X.keys = function (o) { return Object.keys(o).filter(function (k) { return o[k]; }); };
  X.poolOk = function () { return M.pool.filter(function (c) { return c.elig === 'ok'; }); };
  X.invOf = function (code) { return M.invoices.filter(function (i) { return i.code === code; })[0]; };
  X.holderName = function (c) { return c.holder.kind === 'senior' ? X.unit(c.holder.unit).name : c.holder.kind === 'sup' ? X.team(c.holder.team).sup : X.seller(c.holder.id).name; };
  X.holderLevel = function (c) { return { senior: 'سرپرست ارشد', sup: 'سرپرست', seller: 'فروشنده' }[c.holder.kind]; };

  // Invoice-derived facts. Every aggregate on every screen is computed from the same list (cards/table/report parity).
  X.facts = function (filterFn, basis) {
    var list = M.invoices.filter(function (i) { return filterFn(i, basis); });
    var done = list.filter(function (i) { return i.inv === 'completed' && i.review === 'approved'; });
    return {
      created: list.length,
      openPre: list.filter(function (i) { return i.inv === 'pre'; }).length,
      issued: list.length,
      completed: done.length,
      collected: list.filter(function (i) { return i.inv !== 'mismatch' && i.inv !== 'cancelled'; }).reduce(function (a, i) { return a + i.paid; }, 0),
      recon: list.filter(function (i) { return i.inv === 'mismatch'; }).length,
      last: list.reduce(function (a, i) { return i.issued > a ? i.issued : a; }, ''),
      list: list
    };
  };
  X.seller_f = function (id) { return X.facts(function (i) { return i.seller === id; }); };
  X.teamIdOfInv = function (i, basis) { return basis === 'hist' ? i.teamAtIssue : X.seller(i.seller).team; };
  X.team_f = function (tid, basis) { return X.facts(function (i) { return X.teamIdOfInv(i, basis) === tid; }, basis); };
  X.unit_f = function (uid, basis) { return X.facts(function (i) { var t = X.teamIdOfInv(i, basis); return t != null && X.unitOfTeam(t) === uid; }, basis); };
  X.unitCov = function (uid) {
    if (sim() === 'unitfail' && uid === 'U2') return 'failed';
    if (sim() === 'stale' && uid === 'U2') return 'stale';
    if (sim() === 'incomplete' && uid === 'U1') return 'partial';
    if (X.unit_f(uid, 'current').recon) return 'recon';
    return 'ok';
  };
  X.teamCov = function (tid) { var c = X.unitCov(X.unitOfTeam(tid)); if (c === 'recon') return X.team_f(tid, 'current').recon ? 'recon' : 'ok'; return c; };
  X.teamWorkload = function (tid) { return X.sellersOf(tid).reduce(function (a, s) { return a + s.open; }, 0); };
  X.unitWorkload = function (uid) { return X.sellersOfUnit(uid).reduce(function (a, s) { return a + s.open; }, 0); };

  // Recipient resolution: relationship, bypassed levels and write-eligibility are separate facts (visible ≠ writable).
  X.recip = function (k) {
    var c = k.charAt(0);
    if (c === 'U') { var u = X.unit(k); return { k: k, kind: 'senior', name: u.name, level: 'سرپرست ارشد', bypass: [], active: u.active, ok: u.active, why: u.active ? '' : 'غیرفعال — گیرنده نیست (' + u.inactiveNote + ')' }; }
    if (c === 'T') { var t = X.team(k), par = t.parent === 'M' ? [] : [X.unit(t.parent)]; var bad = par.length && !par[0].active; return { k: k, kind: 'sup', name: t.sup, level: 'سرپرست', bypass: par.map(function (p) { return 'سرپرست ارشد ' + p.name; }), active: t.active, ok: t.active, why: t.active ? '' : 'غیرفعال — گیرنده نیست (' + t.inactiveNote + ')', bypassInactive: bad }; }
    var s = X.seller(Number(k.slice(1))), tm = s.team === 'direct' ? null : X.team(s.team), bp = [];
    if (tm) { bp.push('سرپرست ' + tm.sup); if (tm.parent !== 'M') bp.push('سرپرست ارشد ' + X.unit(tm.parent).name); }
    var ok = s.active && s.alloc === 'ok';
    return { k: k, kind: 'seller', name: s.name, level: 'فروشنده', bypass: bp, active: s.active, ok: ok, why: !s.active ? 'غیرفعال — گیرنده نیست' : s.alloc !== 'ok' ? 'اجازه تخصیص برای این فروشنده اثبات نشده؛ دیدن او مجوز نوشتن نمی‌سازد (M-G05)' : '' };
  };
  X.isSkip = function (k) { return X.recip(k).bypass.length > 0; };

  /* ---------- Shared presentation helpers (new reusable patterns; no business capability) ---------- */
  var COV = {
    ok: ['کامل', 'ok', 'checkCircle', 'همه منابع این محدوده با بازه و تعریف انتخاب‌شده دریافت شده‌اند.'],
    partial: ['پوشش ناقص', 'partial', 'layers', 'بخشی از منابع این محدوده دریافت نشده؛ جمع‌ها نهایی نیستند و صفر فرض نمی‌شوند.'],
    stale: ['قدیمی', 'stale', 'clock', 'آخرین داده معتبر قدیمی است؛ پیش از تصمیم بازخوانی کنید.'],
    recon: ['نیازمند تطبیق', 'recon', 'swap', 'بین دو منبع اختلاف وجود دارد؛ عدد دستی تغییر داده نمی‌شود و تطبیق باید انجام شود.'],
    undef: ['تعریف نشده', 'undef', 'question', 'تعریف این شاخص هنوز تصویب نشده است؛ صفر نمایش داده نمی‌شود.'],
    insuff: ['داده ناکافی', 'undef', 'question', 'تعداد یا بازه برای محاسبه کافی نیست؛ ۰٪ به معنی شکست نیست.'],
    failed: ['دریافت نشد', 'failed', 'xCircle', 'دریافت داده این محدوده ناموفق بود؛ بقیه محدوده‌ها معتبر هستند.'],
    bounded: ['سقف نمایش', 'partial', 'rows', 'فقط بخشی از ردیف‌ها نمایش داده می‌شود؛ نبود مورد بیشتر اثبات نمی‌شود و جمع کل نیست.']
  };
  X.cov = function (k, label) { var c = COV[k] || COV.ok; return '<span class="cov c-' + c[1] + ' tip" tabindex="0" data-tip="' + esc(c[3]) + '">' + ic(c[2]) + esc(label || c[0]) + '</span>'; };
  var LINK = {
    confirmed: ['اتصال تأییدشده', 'ok', 'checkCircle', 'اتصال با شناسه پایدار پرونده/فاکتور اثبات شده است.'],
    partial: ['اتصال جزئی', 'partial', 'layers', 'اتصال فقط بخشی از منابع را پوشش می‌دهد یا فقط از شماره موبایل دیده می‌شود؛ شماره اثبات هویت پرونده نیست.'],
    unresolved: ['اتصال نامشخص', 'undef', 'question', 'اتصال قابل تأیید نیست و حدس زده نمی‌شود؛ نیاز به تطبیق دارد.'],
    conflict: ['تعارض اتصال', 'recon', 'swap', 'منابع درباره اتصال اختلاف دارند؛ تا تطبیق، وضعیت قطعی اعلام نمی‌شود.']
  };
  X.link = function (k) { var c = LINK[k] || LINK.confirmed; return '<span class="lc c-' + c[1] + ' tip" tabindex="0" data-tip="' + esc(c[3]) + '">' + ic(c[2]) + esc(c[0]) + '</span>'; };
  var SRC = { legacy: ['سرنخ قدیمی', 'legacy', 'جدول لیدهای قدیمی؛ فقط همین منبع در نمای پایه شمرده می‌شود.'], v4: ['پیگیری فاکتور V4', '', 'پرونده مسیر جدید پیگیری فاکتور.'], mis: ['MIS', '', 'پرونده وارد‌شده از منبع MIS.'] };
  X.src = function (k) { var s = SRC[k] || [k, '', '']; return '<span class="tag src ' + s[1] + ' tip" tabindex="0" data-tip="' + esc(s[2]) + '">' + esc(s[0]) + '</span>'; };
  // RelationshipBadge: visibility never implies write authority.
  X.rel = function (k, extra) {
    var m = { senior: ['سرپرست ارشد مستقیم', 'users', ''], sup: ['سرپرست', 'briefcase', ''], dsup: ['سرپرست مستقیم', 'briefcase', ''], dseller: ['فروشنده مستقیم', 'user', ''], indirect: ['غیرمستقیم · فقط مشاهده', 'eye', ' view'] }[k];
    return '<span class="scope-badge' + m[2] + '">' + ic(m[1]) + m[0] + (extra ? ' · ' + esc(extra) : '') + '</span>';
  };
  // AttributionBasis / TimeBasis — Current Hierarchy · Historical Attribution · Event Actor; Current Snapshot · Event Range · Historical As-Of.
  var BASIS = {
    current: ['ساختار فعلی', 'compass', 'بر اساس درخت سازمانی امروز. برای بار کاری فعلی؛ اعتبار گذشته را تعیین نمی‌کند.'],
    hist: ['انتساب تاریخی', 'history', 'بر اساس تیمی که هنگام صدور/تخصیص ثبت شده. اگر سابقه ثبت نشده باشد «ناشناخته» می‌ماند و جای آن والد امروز نمی‌نشیند.'],
    event: ['عامل رویداد', 'user', 'کسی که آن رویداد را انجام داده؛ مالک فعلی یا مالک اعتبار نیست.'],
    snap: ['وضعیت در لحظه', 'clock', 'عکس فوری از وضعیت فعلی در زمان ارزیابی؛ سابقه گذشته را بازسازی نمی‌کند.'],
    range: ['رویداد در بازه', 'activity', 'رویدادهایی که زمان وقوعشان در بازه است؛ زمان ایجاد یا تخصیص جایگزین آن نمی‌شود.'],
    asof: ['وضعیت تاریخی', 'history', 'وضعیت بازسازی‌شده در لحظه‌ای گذشته؛ فقط با سابقه کافی. داده امروز اثبات آن نیست.']
  };
  X.basis = function (k) { var b = BASIS[k], cls = k === 'snap' ? 'current' : k === 'range' ? 'event' : k === 'asof' ? 'hist' : k; return '<span class="basis b-' + cls + ' tip" tabindex="0" data-tip="' + esc(b[2]) + '">' + ic(b[1]) + b[0] + '</span>'; };
  X.grain = function (text) { return '<span class="grain tip" tabindex="0" data-tip="هر عدد این جدول فقط در همین واحد شمرده می‌شود؛ واحدها با هم جمع یا مقایسه نمی‌شوند.">واحد شمارش: <b>' + esc(text) + '</b></span>'; };
  X.who = function (name, sub) { return '<div class="who"><b>' + esc(name) + '</b>' + (sub ? '<span class="phone">' + sub + '</span>' : '') + '</div>'; };
  X.nextActor = function (n, inv) {
    if (!n) return '<span class="muted">—</span>';
    var lbl = { customer: 'مشتری', seller: 'فروشنده' + (inv ? ': ' + X.seller(inv.seller).name : ''), finance: 'واحد مالی', mis: 'MIS', sup: 'سرپرست مسئول', senior: 'سرپرست ارشد مسئول', self: 'شما', hr: 'منابع انسانی', sales: 'فروشنده / مسئول فروش' }[n.who];
    return '<div class="actor"><span class="actor-who a-' + n.who + '">' + ic(n.who === 'finance' || n.who === 'hr' ? 'hourglass' : n.who === 'mis' ? 'layers' : 'user') + esc(lbl) + '</span><span class="actor-what">' + esc(n.text) + '</span></div>';
  };
  X.freshPart = function (label) { var s = sim() === 'stale'; return { text: label + ': ' + (s ? 'بخشی از واحدها ' + M.freshness.stale : 'به‌روزشده ' + (C.state.fresh === 'now' ? 'همین الان' : M.freshness.now)) + ' · ارزیابی ' + M.freshness.evaluated, stale: s }; };
  X.banners = function () {
    var s = sim();
    if (s === 'stale') return h.banner('stale', '<b>شاخص‌های سرپرست ارشد ' + esc(X.unit('U2').name) + ' ' + M.freshness.stale + ' محاسبه شده‌اند.</b> بقیه واحدها به‌روز هستند؛ جمع کل شامل داده قدیمی آن واحد است و نهایی نیست.', '<button type="button" class="btn btn-sm" data-refresh>' + ic('refresh') + 'بازخوانی</button>');
    if (s === 'incomplete') return h.banner('incomplete', '<b>پوشش منبع ناقص است: لیدهای قدیمی زیرمجموعه ' + esc(X.unit('U1').name) + ' کامل دریافت نشد.</b> شاخص‌های وابسته «پوشش ناقص» نمایش داده می‌شوند و در جمع کل صفر حساب نمی‌شوند.', '<button type="button" class="btn btn-sm" data-refresh>' + ic('refresh') + 'تلاش مجدد</button>');
    if (s === 'unitfail') return h.banner('incomplete', '<b>دریافت داده سرپرست ارشد ' + esc(X.unit('U2').name) + ' ناموفق بود؛ بقیه واحدها بارگذاری شدند.</b> جمع کل فقط شامل واحدهای دریافت‌شده است و با «ناقص» علامت خورده؛ آمار آن واحد صفر فرض نشده است.', '<button type="button" class="btn btn-sm" data-act="retry-unit">' + ic('refresh') + 'تلاش مجدد برای این واحد</button>');
    if (s === 'unknownhist' && X.st.pm !== 'case') return h.banner('incomplete', '<b>سابقه تیم هنگام صدور برای بخشی از فاکتورها ثبت نشده است.</b> این موارد در «انتساب تاریخی» ناشناخته می‌مانند و به تیم امروز نسبت داده نمی‌شوند.', '');
    return '';
  };
  X.steps = h.steps;
  X.bulkbar = h.bulkbar;
  // ScopeBreadcrumb — Manager → سرپرست ارشد → Supervisor → Seller; narrows, never widens.
  X.crumb = function () {
    var p = X.st.pscope, parts = ['<button type="button" class="cr-i' + (p ? '' : ' cur') + '" data-act="pscope:all"' + (p ? '' : ' aria-current="true"') + '>همه واحدها (محدوده مجاز شما)</button>'];
    if (p) {
      var ch = X.chainOf(p);
      ch.forEach(function (c, i) { var last = i === ch.length - 1; parts.push(last ? '<span class="cr-i cur" aria-current="true">' + esc(c[1]) + '</span>' : '<button type="button" class="cr-i" data-act="pscope:' + c[0] + '">' + esc(c[1]) + '</button>'); });
    }
    return '<nav class="scope-crumb" aria-label="محدوده فعلی">' + ic('compass') + parts.join('<span class="cr-sep" aria-hidden="true">‹</span>') + '</nav>';
  };
  X.chainOf = function (p) {
    p = String(p); var out = [];
    if (p === 'M') return [['M', X.unitName('M')]];
    if (p.charAt(0) === 'U') return [[p, X.unitName(p)]];
    if (p.charAt(0) === 'T') { var t = X.team(p); if (t.parent !== 'M') out.push([t.parent, X.unitName(t.parent)]); else out.push(['M', X.unitName('M')]); out.push([p, X.teamName(p)]); return out; }
    var s = X.seller(Number(p)), u = X.unitOfTeam(s.team); out.push([u, X.unitName(u)]); if (s.team !== 'direct') out.push([s.team, X.teamName(s.team)]); out.push([p, s.name]); return out;
  };
  X.top = h.dtop;
  X.foot = h.dfoot;
  X.tl = h.timeline;
  // OwnershipGrid — five independent concepts; never collapsed into one "owner".
  X.own = function (o) {
    return '<div class="own-grid">' + [['مسئول فعلی', 'پرونده الان نزد کیست (Current Custody)', o.custody, 'user'], ['مالک اولیه', 'نخستین دریافت‌کننده پرونده (Original Owner)', o.original, 'history'], ['اقدام بعدی با', 'چه کسی باید کار بعدی را انجام دهد (Next Actor)', o.next, 'arrowL'], ['عامل رویداد', 'کسی که آخرین انتقال/رویداد را انجام داده (Event Actor)', o.event, 'activity'], ['مالک اعتبار', 'طبق قوانین مالی فعلی؛ با تغییر مسئول عوض نمی‌شود (Credit Owner)', o.credit, 'wallet']].map(function (x) {
      return '<div class="own"><span class="own-l">' + ic(x[3]) + esc(x[0]) + hint(x[1]) + '</span><b>' + esc(x[2]) + '</b></div>';
    }).join('') + '</div>';
  };
  X.checks = function (list) { return '<ul class="elig">' + list.map(function (c) { var i = { ok: 'checkCircle', no: 'xCircle', warn: 'alert', q: 'question', info: 'info' }[c[0]]; return '<li class="e-' + c[0] + '">' + ic(i) + '<span><b>' + esc(c[1]) + '</b>' + (c[2] ? '<span>' + esc(c[2]) + '</span>' : '') + '</span></li>'; }).join('') + '</ul>'; };
  X.chain = function (stepsArr) { return '<ol class="chain">' + stepsArr.map(function (s) { var t = { done: 'check', cur: 'hourglass', rejected: 'xCircle', skip: 'dashed' }[s[1]]; return '<li class="c-' + s[1] + '"><span class="c-dot">' + ic(t) + '</span><span class="c-t"><b>' + esc(s[0]) + '</b><span>' + esc(s[2]) + '</span></span></li>'; }).join('') + '</ol>'; };
  X.sec = function (title, aside, inner, cls) { return '<section class="sec' + (cls ? ' ' + cls : '') + '"><div class="sec-h"><h3>' + title + '</h3>' + (aside ? '<span class="aside">' + aside + '</span>' : '') + '</div>' + inner + '</section>'; };
  X.details = function (title, count, inner, open) { return '<details class="sec"' + (open ? ' open' : '') + '><summary><h3>' + title + '</h3>' + (count != null ? '<span class="muted" style="margin-right:8px;font-size:var(--t-meta)">' + fa(count) + '</span>' : '') + '<span class="chev">' + ic('chev') + '</span></summary>' + inner + '</details>'; };
  // BulkOutcomeStrip — requested · eligible · applied · skipped · rejected · failed · unknown (counts must reconcile).
  X.outcomeStrip = function (o) { return h.outcomeStrip(X.counts(o)); };
  X.counts = function (o) {
    var n = function (k) { return o.items.filter(function (x) { return x[1] === k; }).length; };
    return { requested: o.requested, eligible: o.requested - n('skipped') - n('rejected'), ok: n('ok'), skipped: n('skipped'), rejected: n('rejected'), failed: n('failed'), unknown: n('unknown') };
  };

  /* ---------- Operations & Exceptions (coordination only; each row names subject / owner / may / may-not / next actor / resolution signal) ---------- */
  X.attention = function () {
    var opIssues = M.ops.filter(function (o) { var c = X.counts(o); return c.failed || c.skipped || c.rejected || c.unknown; });
    var heldConf = M.held.filter(function (c) { return c.elig === 'conflict'; });
    var inactiveRec = M.units.filter(function (u) { return !u.active; }).length + M.teams.filter(function (t) { return !t.active; }).length;
    var consumed = M.held.filter(function (c) { return c.why === 'consumed' || c.why === 'fin'; });
    var unres = M.held.filter(function (c) { return c.link === 'unresolved' || c.link === 'partial'; }).length + M.invoices.filter(function (i) { return i.link === 'partial' || i.link === 'unresolved' || i.link === 'conflict'; }).length;
    var stale = (sim() === 'stale' ? 1 : 0) + (sim() === 'incomplete' ? 1 : 0) + (sim() === 'unitfail' ? 1 : 0);
    var finRej = M.invoices.filter(function (i) { return i.review === 'rejected'; });
    var xPend = M.xreq.filter(function (r) { return r.state === 'pending' && !X.st.xDone[r.id]; });
    var waitMe = M.hr.filter(function (r) { return r.state === 'pending_review' && r.reviewer === 'self' && !X.st.hrDone[r.id]; });
    return [
      { id: 'dist', n: opIssues.length + heldConf.length, tone: 'orange', icon: 'split', label: 'تعارض یا نتیجه ناقص/نامعلوم توزیع', subject: opIssues.map(function (o) { return o.ref; }).concat(heldConf.map(function (c) { return c.id; })).join('، '), owner: 'مسئول فعلی پرونده (Custody) · شما به‌عنوان عامل دستور', state: 'نیازمند رسیدگی',
        may: ['بررسی مسئول فعلی و واجد شرایط بودن', 'پیش‌نمایش خواندنی/بازخوانی پیش از اقدام مجاز'], mayNot: ['بازنویسی مالکیت', 'تکرار کورکورانه'], next: 'مسئول فعلی پرونده', signal: 'نتیجه معتبر در محدوده یا تعارض صریح', go: 'dist', aq: 'ops' },
      { id: 'inactive', n: inactiveRec, tone: 'amber', icon: 'user', label: 'گیرنده غیرفعال / مسئول جایگزین ثبت نشده', subject: M.units.filter(function (u) { return !u.active; }).map(function (u) { return 'سرپرست ارشد ' + u.name; }).concat(M.teams.filter(function (t) { return !t.active; }).map(function (t) { return 'سرپرست ' + t.sup; })).join('، '), owner: 'منابع انسانی · مالک ساختار', state: 'مسئول جایگزین ثبت نشده',
        may: ['انتخاب گیرنده فعال و مجاز دیگر', 'گزارش محدودیت'], mayNot: ['فعال‌کردن کاربر', 'تغییر ساختار سازمانی'], next: 'منابع انسانی (actor مجاز)', signal: 'واجد شرایط معتبر یا انتخاب گیرنده جایگزین', go: 'perf', pm: 'unit' },
      { id: 'cons', n: consumed.length, tone: 'red', icon: 'lock', label: 'تعارض برگشت / مصرف‌شده (وابستگی مالی)', subject: consumed.map(function (c) { return c.id; }).join('، '), owner: 'فروش / مالی مرتبط (وابستگی مالی و پرونده)', state: 'مسدود',
        may: ['مشاهده دلیل مسدودی', 'حفظ پرونده'], mayNot: ['بازپس‌گیری فقط به اتکای رتبه', 'پاک‌کردن اتصال فاکتور'], next: 'مسئول فروش یا مالی مرتبط', signal: 'وضعیت محافظت‌شده روشن؛ لزوماً برگشت نیست', go: 'dist', aq: 'return', rf: 'blocked' },
      { id: 'link', n: unres, tone: 'slate', icon: 'unlink', label: 'اتصال پرونده نامشخص (منبع)', subject: unres + ' مورد در پرونده‌ها و فاکتورها', owner: 'MIS / مسئول فنی مجاز (از مسیر موجود)', state: 'نامعلوم — تصمیم‌گیری نشده',
        may: ['دیدن نام‌های جایگزین و ناقص‌بودن', 'توقف اقدام وابسته'], mayNot: ['ادغام بر اساس شماره موبایل', 'اتصال مجدد فاکتور'], next: 'actor فنی/MIS مجاز', signal: 'اثبات قابل ردیابی؛ مالک تشدید جدید تعریف نشده (M-G11)', go: 'inv', iq: 'all' },
      { id: 'stale', n: stale, tone: 'amber', icon: 'clock', label: 'شاخص قدیمی یا ناقص', subject: sim() === 'stale' ? 'داده قدیمی سرپرست ارشد ' + X.unit('U2').name : sim() === 'incomplete' ? 'لیدهای قدیمی زیرمجموعه ' + X.unit('U1').name : sim() === 'unitfail' ? 'دریافت سرپرست ارشد ' + X.unit('U2').name + ' ناموفق' : '', owner: 'مسئول گزارش', state: 'نیازمند بازخوانی',
        may: ['دیدن پوشش/بازه/فیلتر', 'رفتن به رکوردهای معتبر'], mayNot: ['تبدیل ناقص به صفر', 'قضاوت عملکرد'], next: 'مسئول گزارش', signal: 'پوشش و تازگی اعلام‌شده', go: 'perf', pm: 'unit' },
      { id: 'f04', n: 1, tone: 'amber', icon: 'swap', label: 'مغایرت گزارش و برگه فاکتور (F04)', subject: 'شاخص فاکتور ' + fa(M.recon.panel.value) + ' در گزارش در برابر ' + fa(M.recon.tab.value) + ' در برگه فاکتورها', owner: 'مالک داده / مالک گزارش', state: 'نیازمند تطبیق · علت اثبات‌نشده',
        may: ['مقایسه واحد، گروه شمارش و محدوده', 'نمایش مغایرت'], mayNot: ['اصلاح مبلغ/وضعیت برای هماهنگی ظاهری', 'اعلام شاخص نهایی'], next: 'مالک داده/گزارش موجود', signal: 'تطبیق روی گروه شمارش هم‌واحد', go: 'perf', pm: 'recon' },
      { id: 'cust', n: 1, tone: 'violet', icon: 'activity', label: 'تاریخچه رفتار مشتری ناقص (F05 باقی‌مانده)', subject: 'فاکتور ' + fa(M.cust.inv) + ' · نشانگر بارگذاری پس از رسیدن داده', owner: 'سرویس تاریخچه', state: 'ناقص · بررسی نیاز دارد',
        may: ['تلاش مجدد خواندن', 'استفاده از زمینه موجود با برچسب ناقص'], mayNot: ['ساخت رویداد', 'ادعای «هیچ رفتاری ندارد»'], next: 'مالک سرویس', signal: 'بارگذاری معتبر یا خطای صریح', go: 'cust' },
      { id: 'arch', n: M.archives.length, tone: 'blue', icon: 'inbox', label: 'بررسی آرشیو عملیاتی', subject: M.archives.length + ' پرونده آرشیوشده', owner: 'فروش / پرداخت (actor فعلی)', state: 'در حال بررسی',
        may: ['بررسی علت، سن، مالک و وابستگی مالی', 'مسیر actor موجود'], mayNot: ['حذف', 'احیا بدون سیاست مصوب'], next: 'actor فروش/پرداخت فعلی', signal: 'رویداد حل‌شده واجد شرط یا ماندن در آرشیو', go: 'arch' },
      { id: 'xnum', n: xPend.length, tone: 'orange', icon: 'plus', label: 'درخواست شماره اضافه در انتظار تصمیم شما', subject: xPend.map(function (r) { return r.id; }).join('، '), owner: 'شما (مدیر تعیین‌شده در ردیف)', state: 'در انتظار تصمیم',
        may: ['بررسی ردیف و تأیید/رد در محدوده کد موجود'], mayNot: ['عبور از محدوده', 'ساخت پرونده خارج از منشأ'], next: 'فروشنده (پس از تأیید) / ردیف رد‌شده', signal: 'نتیجه ثبت‌شده یا خطای روشن', go: 'xnum' },
      { id: 'hr', n: waitMe.length, tone: 'orange', icon: 'briefcase', label: 'درخواست HR منتظر بررسی شما', subject: waitMe.map(function (r) { return '#' + fa(r.id); }).join('، '), owner: 'شما (بررسی‌کننده فعلی) ← مرحله بعد ← منابع انسانی', state: 'در انتظار بررسی مرحله‌ای',
        may: ['تأیید/رد مرحله خودتان'], mayNot: ['اعمال نهایی', 'تغییر دسترسی/نقش', 'عبور از زنجیره'], next: 'بررسی‌کننده بعدی، سپس منابع انسانی', signal: 'گذار ثبت‌شده یا اعمال ناموفق مشخص', go: 'hr' },
      { id: 'fin', n: finRej.length, tone: 'red', icon: 'receipt', label: 'نتیجه مالی نیازمند اقدام فروش', subject: finRej.length + ' فاکتور', owner: 'فروشنده / مسئول اقدام فروش', state: 'در انتظار دیگران',
        may: ['دیدن دلیل و مرحله', 'کمک مجاز (کپی/ارسال لینک)'], mayNot: ['تأیید/رد/بازپرداخت/ثبت مالی'], next: 'فروشنده مسئول', signal: 'ارسال مجدد معتبر یا وضعیت نهایی مالی', go: 'inv', iq: 'action' }
    ].filter(function (a) { return a.n > 0; });
  };
})();
