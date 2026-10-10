/* Senior Supervisor — role layer, part 1: state, namespaced dictionaries, shared presentation helpers.
   Design prototype; mock data only. Nothing here assigns, returns, approves, posts or sends anything.
   Product boundaries: SENIOR-SUPERVISOR-PRODUCT-SPEC.md + GATE-0-PRODUCT-INVARIANTS.md. */
(function () {
  'use strict';
  var C = window.CRM, M = window.SEN, h = C.h;
  var esc = h.esc, fa = h.fa, num = h.num, ic = h.ic, pill = h.pill, hint = h.hint;

  var X = window.SENX = { C: C, M: M, h: h, V: {}, D: {} };

  /* ---------- Role state ---------- */
  X.st = {
    pm: 'team', pb: 'current', pscope: null,                         // performance explorer: mode / attribution basis / drilldown scope
    aq: 'assign', amode: 'select', asel: {}, recip: null, recips: {}, per: 3, exc: false, excReason: '', // distribution
    rf: 'all', rsel: {}, iq: 'action', rg: 'team', hq: 'review', src: 'legacy',
    rbasis: 'event', report: null, rteam: 'all',
    flow: null, live: null, mineOn: false, hrForm: null, hrErr: null, hrDone: {}
  };

  /* ---------- Namespaced status dictionaries (label + tone + icon; never colour alone) ---------- */
  X.ELIG = { ok: { label: 'قابل تخصیص', tone: 'teal', icon: 'checkCircle' }, blocked: { label: 'مسدود', tone: 'red', icon: 'lock' }, unknown: { label: 'نیاز به تطبیق', tone: 'amber', icon: 'question' }, conflict: { label: 'تغییر کرده', tone: 'orange', icon: 'swap' } };
  X.RET = { ok: { label: 'قابل برگشت', tone: 'teal', icon: 'checkCircle' }, blocked: { label: 'مسدود', tone: 'red', icon: 'lock' }, unknown: { label: 'نامعلوم · نیاز به تطبیق', tone: 'amber', icon: 'question' }, conflict: { label: 'تغییر کرده', tone: 'orange', icon: 'swap' } };
  X.INVS = { pre: { label: 'پیش‌فاکتور', tone: 'violet', icon: 'file' }, staged: { label: 'مرحله‌ای', tone: 'teal', icon: 'layers' }, completed: { label: 'تکمیل‌شده', tone: 'green', icon: 'checkCircle' }, cancelled: { label: 'لغو شده', tone: 'slate', icon: 'ban' }, mismatch: { label: 'مغایرت', tone: 'amber', icon: 'question' } };
  X.STG = { pending: 'در انتظار پرداخت', review: 'رسید در انتظار بررسی', rejected: 'رد شده', approved: 'تأییدشده', cancelled: 'لغو' };
  X.EVID = { none: '—', receipt: 'رسید ثبت‌شده', online: 'پرداخت آنلاین (درگاه)' };
  X.REV = { none: null, pending: { label: 'در انتظار مالی', tone: 'orange', icon: 'hourglass' }, approved: { label: 'تأیید مالی', tone: 'green', icon: 'checkCircle' }, rejected: { label: 'رد مالی', tone: 'red', icon: 'alert' } };
  X.OUT = { ok: { label: 'ثبت شد', tone: 'green', icon: 'checkCircle' }, retry: { label: 'ناموفق · قابل تکرار', tone: 'red', icon: 'refresh' }, failed: { label: 'ناموفق', tone: 'red', icon: 'xCircle' }, conflict: { label: 'تعارض', tone: 'orange', icon: 'swap' }, unknown: { label: 'نامعلوم', tone: 'amber', icon: 'question' } };
  X.OPS = { complete: ['کامل', 'green', 'checkCircle'], partial: ['ناقص', 'orange', 'split'], failed: ['ناموفق', 'red', 'xCircle'], unknown: ['نامعلوم', 'amber', 'question'] };
  X.HRS = { pending_review: { label: 'در انتظار بررسی مرحله‌ای', tone: 'orange', icon: 'hourglass' }, pending_hr: { label: 'در انتظار منابع انسانی', tone: 'orange', icon: 'hourglass' }, approved: { label: 'اعمال‌شده', tone: 'green', icon: 'checkCircle' }, rejected: { label: 'رد شده', tone: 'red', icon: 'xCircle' }, failed: { label: 'اعمال ناموفق', tone: 'red', icon: 'alert' }, stepdone: { label: 'مرحله شما تأیید شد', tone: 'teal', icon: 'checkCircle' } };
  X.CONVS = { ready: { label: 'آماده تبدیل', tone: 'teal', icon: 'checkCircle' }, assigned: { label: 'تخصیص‌یافته به تبدیل‌کننده', tone: 'blue', icon: 'user' } };

  /* ---------- Data helpers ---------- */
  var sim = X.sim = function () { return C.state.sim; };
  X.seller = function (id) { return M.sellers.filter(function (s) { return s.id === id; })[0]; };
  X.team = function (id) { return M.teams.filter(function (t) { return t.id === id; })[0]; };
  X.teamName = function (id) { return id === 'direct' ? 'مستقیم (بدون سرپرست میانی)' : id ? 'تیم ' + X.team(id).sup : 'نامشخص'; };
  X.sellersOf = function (id) { return M.sellers.filter(function (s) { return s.team === id; }); };
  X.keys = function (o) { return Object.keys(o).filter(function (k) { return o[k]; }); };
  X.poolOk = function () { return M.pool.filter(function (c) { return c.elig === 'ok'; }); };
  X.phoneSafe = function (s) { return s; };
  X.invOf = function (code) { return M.invoices.filter(function (i) { return i.code === code; })[0]; };
  X.holderName = function (c) { return c.holder.kind === 'sup' ? X.team(c.holder.team).sup : X.seller(c.holder.id).name; };

  // Invoice-derived facts. Every aggregate on every screen is computed from the same list (cards/table/report parity).
  X.facts = function (filterFn, basis) {
    var list = M.invoices.filter(function (i) { return filterFn(i, basis); });
    var done = list.filter(function (i) { return i.inv === 'completed' && i.review === 'approved'; });
    var recon = list.filter(function (i) { return i.inv === 'mismatch'; });
    return {
      created: list.length,
      openPre: list.filter(function (i) { return i.inv === 'pre'; }).length,
      completed: done.length,
      collected: list.filter(function (i) { return i.inv !== 'mismatch' && i.inv !== 'cancelled'; }).reduce(function (a, i) { return a + i.paid; }, 0),
      recon: recon.length,
      last: list.reduce(function (a, i) { return i.issued > a ? i.issued : a; }, ''),
      list: list
    };
  };
  X.seller_f = function (id) { return X.facts(function (i) { return i.seller === id; }); };
  X.teamIdOfInv = function (i, basis) { return basis === 'hist' ? i.teamAtIssue : X.seller(i.seller).team; };
  X.team_f = function (tid, basis) { return X.facts(function (i) { return X.teamIdOfInv(i, basis) === tid; }, basis); };
  X.teamCov = function (tid) {
    if (sim() === 'teamfail' && tid === 'T3') return 'failed';
    if (sim() === 'stale' && tid === 'T3') return 'stale';
    if (sim() === 'incomplete' && tid === 'T2') return 'partial';
    if (X.team_f(tid, 'current').recon) return 'recon';
    return 'ok';
  };
  X.teamWorkload = function (tid) { return X.sellersOf(tid).reduce(function (a, s) { return a + s.open; }, 0); };
  X.readyOf = function (tid) { return M.ready.filter(function (r) { return r.team === tid; }); };

  /* ---------- Shared presentation helpers (new reusable patterns; no business capability) ---------- */
  var COV = {
    ok: ['کامل', 'ok', 'checkCircle', 'همه منابع این محدوده با بازه و تعریف انتخاب‌شده دریافت شده‌اند.'],
    partial: ['پوشش ناقص', 'partial', 'layers', 'بخشی از منابع این محدوده دریافت نشده؛ جمع‌ها نهایی نیستند و صفر فرض نمی‌شوند.'],
    stale: ['قدیمی', 'stale', 'clock', 'آخرین داده معتبر قدیمی است؛ پیش از تصمیم بازخوانی کنید.'],
    recon: ['نیازمند تطبیق', 'recon', 'swap', 'بین دو منبع اختلاف وجود دارد؛ عدد دستی تغییر داده نمی‌شود و تطبیق باید انجام شود.'],
    undef: ['تعریف نشده', 'undef', 'question', 'تعریف این شاخص هنوز تصویب نشده است؛ صفر نمایش داده نمی‌شود.'],
    insuff: ['داده ناکافی', 'undef', 'question', 'تعداد یا بازه برای محاسبه کافی نیست؛ ۰٪ به معنی شکست نیست.'],
    failed: ['دریافت نشد', 'failed', 'xCircle', 'دریافت داده این محدوده ناموفق بود؛ بقیه محدوده‌ها معتبر هستند.']
  };
  // MetricCoverage — operator wording for freshness / coverage / discrepancy (no technical internals).
  X.cov = function (k, label) { var c = COV[k] || COV.ok; return '<span class="cov c-' + c[1] + ' tip" tabindex="0" data-tip="' + esc(c[3]) + '">' + ic(c[2]) + esc(label || c[0]) + '</span>'; };
  var LINK = {
    confirmed: ['اتصال تأییدشده', 'ok', 'checkCircle', 'اتصال با شناسه پایدار پرونده/فاکتور اثبات شده است.'],
    partial: ['اتصال جزئی', 'partial', 'layers', 'اتصال فقط بخشی از منابع را پوشش می‌دهد یا فقط از شماره موبایل دیده می‌شود؛ شماره اثبات هویت پرونده نیست.'],
    unresolved: ['اتصال نامشخص', 'undef', 'question', 'اتصال قابل تأیید نیست و حدس زده نمی‌شود؛ نیاز به تطبیق دارد.'],
    conflict: ['تعارض اتصال', 'recon', 'swap', 'منابع درباره اتصال اختلاف دارند؛ تا تطبیق، وضعیت قطعی اعلام نمی‌شود.']
  };
  // LinkConfidence — how sure we are the record is linked to the Case/Invoice it appears under.
  X.link = function (k) { var c = LINK[k] || LINK.confirmed; return '<span class="lc c-' + c[1] + ' tip" tabindex="0" data-tip="' + esc(c[3]) + '">' + ic(c[2]) + esc(c[0]) + '</span>'; };
  var SRC = { legacy: ['سرنخ قدیمی', 'legacy', 'جدول لیدهای قدیمی؛ فقط همین منبع در نمای پایه شمرده می‌شود.'], v4: ['پیگیری فاکتور V4', '', 'پرونده مسیر جدید پیگیری فاکتور.'], mis: ['MIS', '', 'پرونده وارد‌شده از منبع MIS.'] };
  X.src = function (k) { var s = SRC[k] || [k, '', '']; return '<span class="tag src ' + s[1] + ' tip" tabindex="0" data-tip="' + esc(s[2]) + '">' + esc(s[0]) + '</span>'; };
  // RelationshipBadge (ScopeBadge variants): visibility never implies write authority.
  X.rel = function (k, extra) {
    var m = { sup: ['سرپرست مستقیم', 'users', ''], dseller: ['فروشنده مستقیم', 'user', ''], indirect: ['غیرمستقیم · فقط مشاهده', 'eye', ' view'] }[k];
    return '<span class="scope-badge' + m[2] + '">' + ic(m[1]) + m[0] + (extra ? ' · ' + esc(extra) : '') + '</span>';
  };
  // AttributionBasis — keeps Current Hierarchy / Historical Attribution / Event Actor visibly distinct.
  var BASIS = {
    current: ['ساختار فعلی', 'compass', 'بر اساس درخت سازمانی امروز. برای بار کاری فعلی؛ اعتبار گذشته را تعیین نمی‌کند.'],
    hist: ['انتساب تاریخی', 'history', 'بر اساس تیمی که هنگام صدور/تخصیص ثبت شده. اگر سابقه ثبت نشده باشد «ناشناخته» می‌ماند و جای آن والد امروز نمی‌نشیند.'],
    event: ['عامل رویداد', 'user', 'کسی که آن رویداد را انجام داده؛ مالک فعلی یا مالک اعتبار نیست.']
  };
  X.basis = function (k) { var b = BASIS[k]; return '<span class="basis b-' + k + ' tip" tabindex="0" data-tip="' + esc(b[2]) + '">' + ic(b[1]) + b[0] + '</span>'; };
  X.grain = function (text) { return '<span class="grain tip" tabindex="0" data-tip="هر عدد این جدول فقط در همین واحد شمرده می‌شود؛ واحدها با هم جمع یا مقایسه نمی‌شوند.">واحد شمارش: <b>' + esc(text) + '</b></span>'; };
  X.who = function (name, sub) { return '<div class="who"><b>' + esc(name) + '</b>' + (sub ? '<span class="phone">' + sub + '</span>' : '') + '</div>'; };
  X.nextActor = function (n, inv) {
    if (!n) return '<span class="muted">—</span>';
    var lbl = { customer: 'مشتری', seller: 'فروشنده' + (inv ? ': ' + X.seller(inv.seller).name : ''), finance: 'واحد مالی', mis: 'MIS', sup: 'سرپرست مسئول', self: 'شما', hr: 'منابع انسانی' }[n.who];
    return '<div class="actor"><span class="actor-who a-' + n.who + '">' + ic(n.who === 'finance' || n.who === 'hr' ? 'hourglass' : n.who === 'mis' ? 'layers' : 'user') + esc(lbl) + '</span><span class="actor-what">' + esc(n.text) + '</span></div>';
  };
  X.freshPart = function (label) { var s = sim() === 'stale'; return { text: label + ': ' + (s ? 'بخشی از تیم‌ها ' + M.freshness.stale : 'به‌روزشده ' + (C.state.fresh === 'now' ? 'همین الان' : M.freshness.now)) + ' · ارزیابی ' + M.freshness.evaluated, stale: s }; };
  X.banners = function () {
    var s = sim();
    if (s === 'stale') return h.banner('stale', '<b>شاخص‌های تیم ' + esc(X.team('T3').sup) + ' ' + M.freshness.stale + ' محاسبه شده‌اند.</b> بقیه تیم‌ها به‌روز هستند؛ جمع کل شامل داده قدیمی آن تیم است و نهایی نیست.', '<button type="button" class="btn btn-sm" data-refresh>' + ic('refresh') + 'بازخوانی</button>');
    if (s === 'incomplete') return h.banner('incomplete', '<b>پوشش منبع ناقص است: لیدهای قدیمی تیم ' + esc(X.team('T2').sup) + ' کامل دریافت نشد.</b> شاخص‌های وابسته «پوشش ناقص» نمایش داده می‌شوند و در جمع کل صفر حساب نمی‌شوند.', '<button type="button" class="btn btn-sm" data-refresh>' + ic('refresh') + 'تلاش مجدد</button>');
    if (s === 'teamfail') return h.banner('incomplete', '<b>دریافت داده تیم ' + esc(X.team('T3').sup) + ' ناموفق بود؛ سه تیم دیگر بارگذاری شدند.</b> جمع کل فقط شامل تیم‌های دریافت‌شده است و با «ناقص» علامت خورده؛ آمار آن تیم صفر فرض نشده است.', '<button type="button" class="btn btn-sm" data-act="retry-team">' + ic('refresh') + 'تلاش مجدد برای این تیم</button>');
    if (s === 'unknownhist' && X.st.pm !== 'x') return h.banner('incomplete', '<b>سابقه تیم هنگام صدور برای بخشی از فاکتورها ثبت نشده است.</b> این موارد در «انتساب تاریخی» ناشناخته می‌مانند و به تیم امروز نسبت داده نمی‌شوند.', '');
    return '';
  };
  X.steps = h.steps;
  X.bulkbar = h.bulkbar;
  // ScopeBreadcrumb — shows exactly which subset of the authorised subtree a table is filtered to.
  X.crumb = function () {
    var p = X.st.pscope, parts = ['<button type="button" class="cr-i' + (p ? '' : ' cur') + '" data-act="pscope:all"' + (p ? '' : ' aria-current="true"') + '>همه تیم‌ها (محدوده مجاز شما)</button>'];
    if (p) {
      var isT = String(p).charAt(0) === 'T', tid = isT ? p : X.seller(Number(p)).team;
      parts.push('<button type="button" class="cr-i' + (isT ? ' cur' : '') + '" data-act="pscope:' + tid + '"' + (isT ? ' aria-current="true"' : '') + '>' + esc(X.teamName(tid)) + '</button>');
      if (!isT) parts.push('<span class="cr-i cur" aria-current="true">' + esc(X.seller(Number(p)).name) + '</span>');
    }
    return '<nav class="scope-crumb" aria-label="محدوده فعلی">' + ic('compass') + parts.join('<span class="cr-sep" aria-hidden="true">‹</span>') + '</nav>';
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

  /* ---------- Exceptions (coordination only; every row names subject / responsible actor / state / reason / allowed next action) ---------- */
  X.attention = function () {
    var opIssues = M.ops.filter(function (o) { return o.failed || o.conflict || o.unknown; });
    var invRej = M.invoices.filter(function (i) { return i.review === 'rejected' || (i.review === 'pending'); });
    var waitMe = M.hr.filter(function (r) { return r.state === 'pending_review' && r.reviewer === 'self' && !X.st.hrDone[r.id]; });
    var readyWait = M.ready.filter(function (r) { return r.status === 'ready' && !r.direct; });
    var unres = M.held.filter(function (c) { return c.link === 'unresolved' || c.link === 'partial'; }).length + M.invoices.filter(function (i) { return i.link === 'partial' || i.link === 'unresolved' || i.link === 'conflict'; }).length;
    var consumed = M.held.filter(function (c) { return c.why === 'consumed'; });
    return [
      { id: 'var', n: 1, tone: 'blue', icon: 'users', label: 'تفاوت بار پرونده بین تیم‌ها', subject: 'مقایسه پرونده‌های نزد تیم‌ها', owner: 'سرپرست هر تیم · موجودی خودِ شما', state: 'اطلاع', reason: 'فقط شمار واقعی مقایسه می‌شود؛ آستانه یا امتیاز عدم‌توازن تعریف نشده است.', next: 'مقایسه و در صورت نیاز تخصیص مجاز', go: 'perf', pm: 'case' },
      { id: 'conf', n: opIssues.length, tone: 'orange', icon: 'split', label: 'عملیات تخصیص ناقص یا نامعلوم', subject: opIssues.map(function (o) { return o.ref; }).join('، '), owner: 'شما (عامل دستور) · مسئول فعلی پرونده', state: 'نیازمند رسیدگی', reason: 'تعارض یا پاسخ نرسیده؛ تکرار کورکورانه ممنوع است.', next: 'بازبینی دلیل و تطبیق پیش از تکرار', go: 'dist', aq: 'ops' },
      { id: 'stale', n: (sim() === 'stale' ? 1 : 0) + M.invoices.filter(function (i) { return i.inv === 'mismatch'; }).length + (sim() === 'incomplete' ? 1 : 0), tone: 'amber', icon: 'clock', label: 'شاخص قدیمی یا ناقص', subject: 'مغایرت فاکتور ۷۷۰۰۳۲۱۹ (تیم ' + X.team('T4').sup + ')' + (sim() === 'stale' ? '، داده قدیمی تیم ' + X.team('T3').sup : ''), owner: 'مالک داده (MIS) · مالک شاخص فروش', state: 'نیازمند تطبیق', reason: 'عدد دستی تغییر داده نمی‌شود؛ تا تطبیق «نیازمند تطبیق» می‌ماند.', next: 'ثبت نیاز به تطبیق؛ بازخوانی', drawer: 'diag' },
      { id: 'cons', n: consumed.length, tone: 'red', icon: 'lock', label: 'تعارض برگشت‌شده/مصرف‌شده', subject: consumed.map(function (c) { return c.id; }).join('، '), owner: 'درخواست‌دهنده · واحد مالی (وابستگی مالی)', state: 'مسدود', reason: 'پرونده به فاکتور مرحله‌ای V4 وابسته است؛ برگشت عادی انجام نمی‌شود.', next: 'مشاهده مدرک؛ تطبیق با مالی/MIS', go: 'dist', aq: 'return', rf: 'blocked' },
      { id: 'inv', n: invRej.length, tone: 'red', icon: 'receipt', label: 'فاکتور رد یا در انتظار بررسی مالی', subject: invRej.length + ' فاکتور', owner: 'فروشنده / واحد مالی (طبق اقدام بعدی)', state: 'در انتظار دیگران', reason: 'اقدام بعدی با فروشنده یا مالی است، نه شما.', next: 'نظارت؛ پیگیری از سرپرست مسئول', go: 'inv', iq: 'action' },
      { id: 'ready', n: readyWait.length, tone: 'violet', icon: 'repeat', label: 'آماده تبدیل در انتظار سرپرست', subject: readyWait.length + ' پرونده آماده', owner: 'سرپرست مسئول هر تیم', state: 'منتظر تخصیص توسط سرپرست', reason: 'تخصیص تبدیل از این پنل انجام نمی‌شود.', next: 'شناسایی مسئول و پیگیری', go: 'ready' },
      { id: 'inact', n: 1, tone: 'amber', icon: 'user', label: 'سرپرست/گیرنده غیرفعال', subject: 'تیم ' + X.team('T4').sup + ' · ' + fa(X.sellersOf('T4').length) + ' فروشنده فعال', owner: 'منابع انسانی · مدیر فروش', state: 'مسئول جایگزین ثبت نشده', reason: 'گیرنده غیرفعال انتخاب‌شدنی نیست و مالک جدید خودکار تعیین نمی‌شود.', next: 'انتخاب گیرنده واجد شرایط دیگر', go: 'perf', pm: 'sup' },
      { id: 'link', n: unres, tone: 'slate', icon: 'unlink', label: 'منبع یا اتصال نامشخص', subject: unres + ' مورد در پرونده‌ها و فاکتورها', owner: 'MIS · واحد مالی', state: 'نامعلوم — تصمیم‌گیری نشده', reason: 'اتصال با شماره موبایل حدس زده نمی‌شود.', next: 'نمایش به‌صورت نامشخص؛ تطبیق', go: 'case' },
      { id: 'hr', n: waitMe.length, tone: 'orange', icon: 'briefcase', label: 'درخواست HR منتظر بررسی شما', subject: waitMe.map(function (r) { return '#' + fa(r.id); }).join('، '), owner: 'شما (بررسی‌کننده فعلی) → منابع انسانی', state: 'در انتظار بررسی مرحله‌ای', reason: 'تأیید شما فقط مرحله را جلو می‌برد.', next: 'بررسی و تأیید این مرحله یا رد با دلیل', go: 'hr' }
    ].filter(function (a) { return a.n > 0; });
  };
})();
