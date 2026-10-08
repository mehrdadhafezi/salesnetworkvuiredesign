/* Sales Deputy — role layer, part 1: state, namespaced dictionaries, hierarchy/facts helpers, shared presentation helpers.
   Design prototype; mock data only. Nothing here assigns, returns, approves, posts, sends, archives or repairs anything.
   Product boundaries: SALES-DEPUTY-PRODUCT-SPEC.md + CRM-CROSS-ROLE-ARCHITECTURE-V1.md + GATE-0-PRODUCT-INVARIANTS.md.
   Deputy = leadership + cross-Manager coordination. Visibility never becomes write authority; hierarchy never becomes permission. */
(function () {
  'use strict';
  var C = window.CRM, M = window.DEP, h = C.h;
  var esc = h.esc, fa = h.fa, num = h.num, ic = h.ic, pill = h.pill, hint = h.hint;

  var X = window.DEPX = { C: C, M: M, h: h, V: {}, D: {} };

  /* ---------- Role state ---------- */
  X.st = {
    pm: 'mgr', pb: 'current', pscope: null, tb: 'event',                    // performance explorer: grain / attribution basis / drill scope / time basis
    aq: 'assign', amode: 'select', asel: {}, recip: null, recips: {}, per: 3, bypassOpen: false, bypassReason: '', bypassOk: false, // conditional direct allocation
    rf: 'all', rsel: {}, iq: 'action', hq: 'review', ec: 'all', em: 'all', dq: 'f02', mq: 'arch', monAuth: 'partial',
    rbasis: 'event', report: null, rscope: 'all', rgroup: 'mgr',
    flow: null, live: null, hrDone: {}, resent: {}
  };

  /* ---------- Namespaced status dictionaries (label + tone + icon; never colour alone) ---------- */
  X.ELIG = { ok: { label: 'قابل تخصیص', tone: 'teal', icon: 'checkCircle' }, blocked: { label: 'مسدود', tone: 'red', icon: 'lock' }, unknown: { label: 'نیاز به تطبیق', tone: 'amber', icon: 'question' }, conflict: { label: 'تغییر کرده', tone: 'orange', icon: 'swap' } };
  X.RET = { ok: { label: 'واجد شرایط (پیش‌نمایش)', tone: 'teal', icon: 'checkCircle' }, blocked: { label: 'مسدود', tone: 'red', icon: 'lock' }, unknown: { label: 'نامعلوم · نیاز به تطبیق', tone: 'amber', icon: 'question' }, conflict: { label: 'تعارض · تغییر کرده', tone: 'orange', icon: 'swap' } };
  X.INVS = { pre: { label: 'پیش‌فاکتور', tone: 'violet', icon: 'file' }, staged: { label: 'مرحله‌ای', tone: 'teal', icon: 'layers' }, completed: { label: 'تکمیل‌شده', tone: 'green', icon: 'checkCircle' }, cancelled: { label: 'لغو شده', tone: 'slate', icon: 'ban' }, mismatch: { label: 'مغایرت', tone: 'amber', icon: 'question' } };
  X.STG = { pending: 'در انتظار پرداخت', review: 'رسید در انتظار بررسی', rejected: 'رد شده', approved: 'تأییدشده', cancelled: 'لغو' };
  X.EVID = { none: '—', receipt: 'رسید ثبت‌شده', online: 'پرداخت آنلاین (درگاه)' };
  X.REV = { none: null, pending: { label: 'در انتظار مالی', tone: 'orange', icon: 'hourglass' }, approved: { label: 'تأیید مالی', tone: 'green', icon: 'checkCircle' }, rejected: { label: 'رد مالی', tone: 'red', icon: 'alert' } };
  X.OUT = {
    ok: { label: 'اعمال شد', tone: 'green', icon: 'checkCircle' }, skipped: { label: 'ارسال نشد', tone: 'orange', icon: 'swap' }, rejected: { label: 'ردشده توسط سامانه', tone: 'red', icon: 'ban' },
    failed: { label: 'ناموفق · قابل تکرار', tone: 'red', icon: 'refresh' }, unknown: { label: 'نامعلوم', tone: 'amber', icon: 'question' }
  };
  X.OPS = { complete: ['کامل', 'green', 'checkCircle'], partial: ['ناقص', 'orange', 'split'], failed: ['ناموفق', 'red', 'xCircle'], unknown: ['نامعلوم', 'amber', 'question'] };
  X.HRS = { pending_review: { label: 'در انتظار بررسی مرحله‌ای', tone: 'orange', icon: 'hourglass' }, pending_hr: { label: 'در انتظار منابع انسانی', tone: 'orange', icon: 'hourglass' }, approved: { label: 'اعمال‌شده', tone: 'green', icon: 'checkCircle' }, rejected: { label: 'رد شده', tone: 'red', icon: 'xCircle' }, failed: { label: 'اعمال ناموفق', tone: 'red', icon: 'alert' }, stepdone: { label: 'مرحله شما تأیید شد', tone: 'teal', icon: 'checkCircle' }, observe: { label: 'در مرحله دیگری', tone: 'slate', icon: 'eye' } };
  X.ARC = { noanswer: ['بی‌پاسخ', 'slate', 'phone'], assess: ['ارزیابی پرداخت‌نشده', 'amber', 'receipt'], sub: ['اشتراک پرداخت‌نشده', 'violet', 'receipt'], product: ['محصول پرداخت‌نشده', 'blue', 'receipt'], productx: ['محصول* پرداخت‌نشده', 'teal', 'receipt'] };

  /* ---------- Hierarchy helpers (Manager → Senior → Supervisor → Seller). Current structure only; history is never inferred from it. ---------- */
  var sim = X.sim = function () { return C.state.sim; };
  X.mgr = function (id) { return M.managers.filter(function (m) { return m.id === id; })[0]; };
  X.senior = function (id) { return M.seniors.filter(function (u) { return u.id === id; })[0]; };
  X.team = function (id) { return M.teams.filter(function (t) { return t.id === id; })[0]; };
  X.seller = function (id) { return M.sellers.filter(function (s) { return s.id === id; })[0]; };
  X.mgrOfTeam = function (tid) { if (tid == null) return null; if (tid.charAt(0) === 'd') return tid.slice(1); var p = X.team(tid).parent; return p.charAt(0) === 'U' ? X.senior(p).mgr : p; };
  X.seniorOfTeam = function (tid) { if (tid == null || tid.charAt(0) === 'd') return null; var p = X.team(tid).parent; return p.charAt(0) === 'U' ? p : null; };
  X.mgrOfSeller = function (s) { return X.mgrOfTeam(s.team); };
  X.teamName = function (id) { return id == null ? 'نامشخص' : id.charAt(0) === 'd' ? 'فروشنده مستقیم زیر ' + X.mgr(id.slice(1)).name : 'تیم ' + X.team(id).sup; };
  X.mgrName = function (id) { return id ? 'مدیر ' + X.mgr(id).name : '—'; };
  X.seniorName = function (id) { return id ? 'سرپرست ارشد ' + X.senior(id).name : 'بدون سرپرست ارشد میانی'; };
  X.teamsOfSenior = function (uid) { return M.teams.filter(function (t) { return t.parent === uid; }); };
  X.seniorsOfMgr = function (mid) { return M.seniors.filter(function (u) { return u.mgr === mid; }); };
  X.directTeams = function (mid) { return M.teams.filter(function (t) { return t.parent === mid; }); };
  X.sellersOfTeam = function (tid) { return M.sellers.filter(function (s) { return s.team === tid; }); };
  X.sellersOfSenior = function (uid) { return M.sellers.filter(function (s) { return X.seniorOfTeam(s.team) === uid; }); };
  X.sellersOfMgr = function (mid) { return M.sellers.filter(function (s) { return X.mgrOfTeam(s.team) === mid; }); };
  X.sum = function (list, k) { return list.reduce(function (a, s) { return a + (s[k] || 0); }, 0); };
  X.mgrLoad = function (mid) { return X.sum(X.sellersOfMgr(mid), 'open'); };
  X.seniorLoad = function (uid) { return X.sum(X.sellersOfSenior(uid), 'open'); };
  X.teamLoad = function (tid) { return X.sum(X.sellersOfTeam(tid), 'open'); };
  X.keys = function (o) { return Object.keys(o).filter(function (k) { return o[k]; }); };
  X.poolList = function () { return sim() === 'emptypool' ? [] : M.pool; };
  X.poolOk = function () { return X.poolList().filter(function (c) { return c.elig === 'ok'; }); };
  X.holderName = function (c) { var k = c.holder.kind; return k === 'manager' ? X.mgr(c.holder.mgr).name : k === 'senior' ? X.senior(c.holder.unit).name : k === 'sup' ? X.team(c.holder.team).sup : X.seller(c.holder.id).name; };
  X.holderLevel = function (c) { return { manager: 'مدیر فروش', senior: 'سرپرست ارشد', sup: 'سرپرست', seller: 'فروشنده' }[c.holder.kind]; };
  X.chainOfSeller = function (s) { var t = s.team; return { mgr: X.mgrOfTeam(t), senior: X.seniorOfTeam(t), team: t.charAt(0) === 'd' ? null : t }; };

  // Invoice-derived facts. Every aggregate on every screen is computed from the same list (cards/table/report parity).
  X.facts = function (filterFn, basis) {
    var list = M.invoices.filter(function (i) { return filterFn(i, basis); });
    var done = list.filter(function (i) { return i.inv === 'completed' && i.review === 'approved'; });
    return {
      created: list.length,
      openPre: list.filter(function (i) { return i.inv === 'pre'; }).length,
      completed: done.length,
      collected: list.filter(function (i) { return i.inv !== 'mismatch' && i.inv !== 'cancelled'; }).reduce(function (a, i) { return a + i.paid; }, 0),
      recon: list.filter(function (i) { return i.inv === 'mismatch'; }).length,
      list: list
    };
  };
  X.teamIdOfInv = function (i, basis) { return basis === 'hist' ? i.teamAtIssue : X.seller(i.seller).team; };
  X.mgr_f = function (mid, basis) { return X.facts(function (i) { var t = X.teamIdOfInv(i, basis); return t != null && X.mgrOfTeam(t) === mid; }, basis); };
  X.senior_f = function (uid, basis) { return X.facts(function (i) { var t = X.teamIdOfInv(i, basis); return t != null && X.seniorOfTeam(t) === uid; }, basis); };
  X.team_f = function (tid, basis) { return X.facts(function (i) { return X.teamIdOfInv(i, basis) === tid; }, basis); };
  X.seller_f = function (id) { return X.facts(function (i) { return i.seller === id; }); };
  X.allFacts = function () { return X.facts(function (i) { return !X.mgrFailed(X.mgrOfSeller(X.seller(i.seller))); }); };
  X.mgrFailed = function (mid) { return X.mgrCov(mid) === 'failed'; };

  // BranchCoverage state per Manager branch: one failed branch never erases the others, and never becomes zero.
  X.mgrCov = function (mid) {
    var s = sim();
    if (s === 'branchfail' && mid === 'MG3') return 'failed';
    if (s === 'stale' && mid === 'MG2') return 'stale';
    if (s === 'incomplete' && mid === 'MG1') return 'partial';
    if (s === 'parthier' && mid === 'MG3') return 'partial';
    if (X.mgr_f(mid, 'current').recon) return 'recon';
    return 'ok';
  };
  X.seniorCov = function (uid) { return X.mgrCov(X.senior(uid).mgr); };
  X.teamCov = function (tid) { var c = X.mgrCov(X.mgrOfTeam(tid)); if (c === 'recon') return X.team_f(tid, 'current').recon ? 'recon' : 'ok'; return c; };

  // Recipient resolution: relationship, bypassed levels and write-eligibility are separate facts (visible ≠ writable; hierarchy ≠ permission).
  // From the Deputy, the Manager level is the ordinary path. Everything below it bypasses at least one level and is EXCEPTIONAL + CONDITIONAL.
  X.recip = function (k) {
    var c = k.charAt(0);
    if (k === 'XM') return { k: k, kind: 'manager', name: 'پوریا آذر', level: 'مدیر فروش (شاخه مجاور)', bypass: [], active: true, ok: false, scopeOut: true, why: 'خارج از محدوده مجاز شما — دیدن نام یا گزارش‌های مشترک مجوز تخصیص نیست' };
    if (c === 'M') { var m = X.mgr(k); return { k: k, kind: 'manager', name: m.name, level: 'مدیر فروش', bypass: [], active: m.active, ok: m.active, why: m.active ? '' : 'غیرفعال — گیرنده نیست' }; }
    if (c === 'U') { var u = X.senior(k), bp = [X.mgrName(u.mgr)]; return { k: k, kind: 'senior', name: u.name, level: 'سرپرست ارشد', bypass: bp, active: u.active, ok: u.active, why: u.active ? '' : 'غیرفعال — گیرنده نیست (' + u.inactiveNote + ')' }; }
    if (c === 'T') {
      var t = X.team(k), b2 = [X.mgrName(X.mgrOfTeam(k))], un = X.seniorOfTeam(k); if (un) b2.push(X.seniorName(un));
      return { k: k, kind: 'sup', name: t.sup, level: 'سرپرست', bypass: b2, active: t.active, ok: t.active, why: t.active ? '' : 'غیرفعال — گیرنده نیست (' + t.inactiveNote + ')' };
    }
    var s = X.seller(Number(k.slice(1))), ch = X.chainOfSeller(s), b3 = [X.mgrName(ch.mgr)];
    if (ch.senior) b3.push(X.seniorName(ch.senior)); if (ch.team) b3.push('سرپرست ' + X.team(ch.team).sup);
    var ok = s.active && s.alloc === 'ok';
    return { k: k, kind: 'seller', name: s.name, level: 'فروشنده', bypass: b3, active: s.active, ok: ok, why: !s.active ? 'غیرفعال — گیرنده نیست' : s.alloc !== 'ok' ? 'اجازه تخصیص برای این فروشنده اثبات نشده؛ دیدن او مجوز نوشتن نمی‌سازد (SD-G06)' : '' };
  };
  X.isBypass = function (k) { return X.recip(k).bypass.length > 0; };

  /* ---------- Shared presentation helpers (reusable patterns; no business capability) ---------- */
  var COV = {
    ok: ['کامل', 'ok', 'checkCircle', 'همه منابع این محدوده با بازه و تعریف انتخاب‌شده دریافت شده‌اند.'],
    partial: ['پوشش ناقص', 'partial', 'layers', 'بخشی از منابع یا سلسله‌مراتب این محدوده دریافت نشده؛ جمع‌ها نهایی نیستند و صفر فرض نمی‌شوند.'],
    stale: ['قدیمی', 'stale', 'clock', 'آخرین داده معتبر قدیمی است؛ پیش از تصمیم بازخوانی کنید.'],
    recon: ['نیازمند تطبیق', 'recon', 'swap', 'بین دو منبع اختلاف وجود دارد؛ عدد دستی تغییر داده نمی‌شود و تطبیق باید انجام شود.'],
    undef: ['تعریف نشده', 'undef', 'question', 'تعریف این شاخص هنوز تصویب نشده است؛ صفر نمایش داده نمی‌شود.'],
    insuff: ['داده ناکافی', 'undef', 'question', 'تعداد یا بازه برای محاسبه کافی نیست؛ ۰ به معنی شکست نیست.'],
    failed: ['دریافت نشد', 'failed', 'xCircle', 'دریافت داده این شاخه ناموفق بود؛ بقیه شاخه‌ها معتبر هستند.'],
    bounded: ['سقف نمایش', 'partial', 'rows', 'فقط بخشی از ردیف‌ها نمایش داده می‌شود؛ نبود مورد بیشتر اثبات نمی‌شود و جمع کل نیست.'],
    unauth: ['غیرمجاز', 'undef', 'lock', 'دسترسی به این داده برای محدوده یا میدان شما تأیید نشده است؛ صفر یا خالی نمایش داده نمی‌شود.']
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
    var m = { manager: ['مدیر مستقیم', 'users', ''], senior: ['سرپرست ارشد · غیرمستقیم', 'users', ' view'], sup: ['سرپرست · غیرمستقیم', 'briefcase', ' view'], dsup: ['سرپرست زیر مدیر', 'briefcase', ' view'], seller: ['فروشنده · غیرمستقیم', 'user', ' view'], indirect: ['غیرمستقیم · فقط مشاهده', 'eye', ' view'], out: ['خارج از محدوده', 'lock', ' view'] }[k];
    return '<span class="scope-badge' + m[2] + '">' + ic(m[1]) + m[0] + (extra ? ' · ' + esc(extra) : '') + '</span>';
  };
  // AttributionBasis / TimeBasis — Current Hierarchy · Historical Attribution · Credit Owner · Event Actor; Current Snapshot · Event Range · Historical As-Of.
  var BASIS = {
    current: ['ساختار فعلی', 'compass', 'بر اساس درخت سازمانی امروز. برای بار کاری فعلی؛ اعتبار گذشته را تعیین نمی‌کند.'],
    hist: ['انتساب تاریخی', 'history', 'بر اساس تیمی که هنگام صدور/تخصیص ثبت شده. اگر سابقه ثبت نشده باشد «ناشناخته» می‌ماند و جای آن والد امروز نمی‌نشیند.'],
    event: ['عامل رویداد', 'user', 'کسی که آن رویداد را انجام داده؛ مالک فعلی یا مالک اعتبار نیست.'],
    credit: ['مالک اعتبار', 'wallet', 'طبق قوانین مالی فعلی؛ از گروه‌بندی گزارش یا والد امروز نتیجه نمی‌شود.'],
    snap: ['وضعیت در لحظه', 'clock', 'عکس فوری از وضعیت فعلی در زمان ارزیابی؛ سابقه گذشته را بازسازی نمی‌کند.'],
    range: ['رویداد در بازه', 'activity', 'رویدادهایی که زمان وقوعشان در بازه است؛ زمان ایجاد یا تخصیص جایگزین آن نمی‌شود.'],
    asof: ['وضعیت تاریخی', 'history', 'وضعیت بازسازی‌شده در لحظه‌ای گذشته؛ فقط با سابقه کافی. داده امروز اثبات آن نیست.']
  };
  X.basis = function (k) { var b = BASIS[k], cls = k === 'snap' ? 'current' : k === 'range' ? 'event' : k === 'asof' ? 'hist' : k === 'credit' ? 'event' : k; return '<span class="basis b-' + cls + ' tip" tabindex="0" data-tip="' + esc(b[2]) + '">' + ic(b[1]) + b[0] + '</span>'; };
  X.grain = function (text) { return '<span class="grain tip" tabindex="0" data-tip="هر عدد این جدول فقط در همین واحد شمارش، شمرده می‌شود؛ سطح‌ها (مدیر، سرپرست ارشد، سرپرست، فروشنده) با هم جمع نمی‌شوند.">واحد شمارش: <b>' + esc(text) + '</b></span>'; };
  X.who = function (name, sub) { return '<div class="who"><b>' + esc(name) + '</b>' + (sub ? '<span class="phone">' + sub + '</span>' : '') + '</div>'; };
  X.nextActor = function (n, inv) {
    if (!n) return '<span class="muted">—</span>';
    var lbl = { customer: 'مشتری', seller: 'فروشنده' + (inv ? ': ' + X.seller(inv.seller).name : ''), finance: 'واحد مالی', mis: 'MIS', sup: 'سرپرست مسئول', senior: 'سرپرست ارشد مسئول', manager: 'مدیر مسئول', self: 'شما', hr: 'منابع انسانی', sales: 'فروشنده / مسئول فروش', unknown: 'نامشخص' }[n.who];
    return '<div class="actor"><span class="actor-who a-' + n.who + '">' + ic(n.who === 'finance' || n.who === 'hr' ? 'hourglass' : n.who === 'mis' ? 'layers' : n.who === 'unknown' ? 'question' : 'user') + esc(lbl) + '</span><span class="actor-what">' + esc(n.text) + '</span></div>';
  };
  X.freshPart = function (label) { var s = sim() === 'stale'; return { text: label + ': ' + (s ? 'بخشی از شاخه‌ها ' + M.freshness.stale : 'به‌روزشده ' + (C.state.fresh === 'now' ? 'همین الان' : M.freshness.now)) + ' · ارزیابی ' + M.freshness.evaluated, stale: s }; };
  X.banners = function () {
    var s = sim();
    if (s === 'stale') return h.banner('stale', '<b>داده شاخه ' + esc(X.mgr('MG2').name) + ' ' + M.freshness.stale + ' محاسبه شده است.</b> بقیه شاخه‌ها به‌روز هستند؛ جمع کل شامل داده قدیمی آن شاخه است و نهایی نیست.', '<button type="button" class="btn btn-sm" data-refresh>' + ic('refresh') + 'بازخوانی</button>');
    if (s === 'incomplete') return h.banner('incomplete', '<b>پوشش گزارش ناقص است: لیدهای قدیمی شاخه ' + esc(X.mgr('MG1').name) + ' کامل دریافت نشد.</b> شاخص‌های وابسته «پوشش ناقص» نمایش داده می‌شوند و در جمع کل صفر حساب نمی‌شوند.', '<button type="button" class="btn btn-sm" data-refresh>' + ic('refresh') + 'تلاش مجدد</button>');
    if (s === 'branchfail') return h.banner('incomplete', '<b>دریافت داده شاخه ' + esc(X.mgr('MG3').name) + ' ناموفق بود؛ شاخه‌های دیگر بارگذاری شدند.</b> جمع کل فقط شاخه‌های دریافت‌شده را شامل می‌شود و «ناقص» است؛ آمار آن شاخه صفر فرض نشده است. موفقیت بقیه شاخه‌ها کل درخت را کامل نمی‌کند.', '<button type="button" class="btn btn-sm" data-act="retry-branch">' + ic('refresh') + 'تلاش مجدد برای این شاخه</button>');
    if (s === 'parthier') return h.banner('incomplete', '<b>سلسله‌مراتب شاخه ' + esc(X.mgr('MG3').name) + ' ناقص دریافت شد (سرپرست ارشد ' + esc(X.senior('U5').name) + ' و زیرمجموعه‌اش).</b> نبود ردیف به معنی غیرفعال‌بودن یا نبود فروش نیست؛ جمع‌ها نهایی نیستند.', '<button type="button" class="btn btn-sm" data-act="retry-branch">' + ic('refresh') + 'تلاش مجدد</button>');
    if (s === 'unknownhist' && X.st.pm !== 'case') return h.banner('incomplete', '<b>سابقه تیم هنگام صدور برای بخشی از فاکتورها ثبت نشده است.</b> این موارد در «انتساب تاریخی» ناشناخته می‌مانند و به ساختار امروز نسبت داده نمی‌شوند.', '');
    return '';
  };
  X.steps = function (list, cur) { return '<ol class="steps" aria-label="مراحل">' + list.map(function (s, i) { var c = i < cur ? 'done' : i === cur ? 'cur' : ''; return '<li class="' + c + '"' + (i === cur ? ' aria-current="step"' : '') + '><span class="sn">' + (i < cur ? ic('check') : fa(i + 1)) + '</span><span class="st-l">' + esc(s) + '</span></li>'; }).join('') + '</ol>'; };
  X.bulkbar = function (text, sub, actions) { return '<div class="bulkbar" role="region" aria-label="انتخاب گروهی"><div class="bb-txt"><b>' + text + '</b>' + (sub ? '<span>' + sub + '</span>' : '') + '</div><span class="grow"></span>' + actions + '</div>'; };
  // ScopeBreadcrumb — all permitted branches → Manager → Senior → Supervisor → Seller; narrows, never widens.
  X.chainOf = function (p) {
    p = String(p); var out = [];
    if (p.charAt(0) === 'M') return [[p, X.mgrName(p)]];
    if (p.charAt(0) === 'U') return [[X.senior(p).mgr, X.mgrName(X.senior(p).mgr)], [p, X.seniorName(p)]];
    if (p.charAt(0) === 'T') { var m = X.mgrOfTeam(p), u = X.seniorOfTeam(p); out.push([m, X.mgrName(m)]); if (u) out.push([u, X.seniorName(u)]); out.push([p, X.teamName(p)]); return out; }
    var s = X.seller(Number(p)), ch = X.chainOfSeller(s); out.push([ch.mgr, X.mgrName(ch.mgr)]); if (ch.senior) out.push([ch.senior, X.seniorName(ch.senior)]); if (ch.team) out.push([ch.team, X.teamName(ch.team)]); out.push([p, s.name]); return out;
  };
  X.crumb = function () {
    var p = X.st.pscope, parts = ['<button type="button" class="cr-i' + (p ? '' : ' cur') + '" data-act="pscope:all"' + (p ? '' : ' aria-current="true"') + '>همه شاخه‌های محدوده مجاز شما</button>'];
    if (p) X.chainOf(p).forEach(function (c, i, ch) { var last = i === ch.length - 1; parts.push(last ? '<span class="cr-i cur" aria-current="true">' + esc(c[1]) + '</span>' : '<button type="button" class="cr-i" data-act="pscope:' + c[0] + '">' + esc(c[1]) + '</button>'); });
    return '<nav class="scope-crumb" aria-label="محدوده فعلی">' + ic('compass') + parts.join('<span class="cr-sep" aria-hidden="true">‹</span>') + '</nav>';
  };
  X.top = function (title, extra) { return '<div class="sheet-grip" aria-hidden="true"></div><div class="dr-top"><span>' + esc(title) + '</span><span class="grow"></span>' + (extra || '') + '<button type="button" class="btn btn-ghost btn-icon btn-sm tip" data-tip="بستن (Esc)" data-close aria-label="بستن">' + ic('x') + '</button></div>'; };
  X.foot = function (primary, secondary, hintTxt) { return '<div class="dr-foot">' + (primary || '') + (secondary || '<button type="button" class="btn btn-lg btn-ghost" data-close>بستن</button>') + '<span class="grow"></span>' + (hintTxt ? '<span class="hint">' + hintTxt + '</span>' : '') + '</div>'; };
  X.tl = function (items) { return '<ul class="timeline">' + items.map(function (x) { return '<li>' + esc(x[0]) + '<span>' + esc(x[1]) + '</span></li>'; }).join('') + '</ul>'; };
  // OwnershipGrid — five independent concepts; never collapsed into one "owner".
  X.own = function (o) {
    return '<div class="own-grid">' + [['مسئول فعلی', 'پرونده الان نزد کیست (Current Custody)', o.custody, 'user'], ['مالک اولیه', 'نخستین دریافت‌کننده پرونده (Original Owner)', o.original, 'history'], ['اقدام بعدی با', 'چه کسی باید کار بعدی را انجام دهد (Next Actor)', o.next, 'arrowL'], ['عامل رویداد', 'کسی که آخرین انتقال/رویداد را انجام داده (Event Actor)', o.event, 'activity'], ['مالک اعتبار', 'طبق قوانین مالی فعلی؛ با تغییر مسئول یا ساختار عوض نمی‌شود (Credit Owner)', o.credit, 'wallet']].map(function (x) {
      return '<div class="own"><span class="own-l">' + ic(x[3]) + esc(x[0]) + hint(x[1]) + '</span><b>' + esc(x[2]) + '</b></div>';
    }).join('') + '</div>';
  };
  X.checks = function (list) { return '<ul class="elig">' + list.map(function (c) { var i = { ok: 'checkCircle', no: 'xCircle', warn: 'alert', q: 'question', info: 'info' }[c[0]]; return '<li class="e-' + c[0] + '">' + ic(i) + '<span><b>' + esc(c[1]) + '</b>' + (c[2] ? '<span>' + esc(c[2]) + '</span>' : '') + '</span></li>'; }).join('') + '</ul>'; };
  X.chain = function (stepsArr) { return '<ol class="chain">' + stepsArr.map(function (s) { var t = { done: 'check', cur: 'hourglass', rejected: 'xCircle', skip: 'dashed' }[s[1]]; return '<li class="c-' + s[1] + '"><span class="c-dot">' + ic(t) + '</span><span class="c-t"><b>' + esc(s[0]) + '</b><span>' + esc(s[2]) + '</span></span></li>'; }).join('') + '</ol>'; };
  X.sec = function (title, aside, inner, cls) { return '<section class="sec' + (cls ? ' ' + cls : '') + '"><div class="sec-h"><h3>' + title + '</h3>' + (aside ? '<span class="aside">' + aside + '</span>' : '') + '</div>' + inner + '</section>'; };
  X.details = function (title, count, inner, open) { return '<details class="sec"' + (open ? ' open' : '') + '><summary><h3>' + title + '</h3>' + (count != null ? '<span class="muted" style="margin-right:8px;font-size:var(--t-meta)">' + fa(count) + '</span>' : '') + '<span class="chev">' + ic('chev') + '</span></summary>' + inner + '</details>'; };
  // BulkOutcomeStrip (shared, SN-202) — requested · eligible · applied · skipped · rejected · failed · unknown (counts must reconcile).
  X.outcomeStrip = function (o) {
    var c = X.counts(o), cells = [['requested', 'درخواست‌شده', c.requested], ['eligible', 'واجد شرایط', c.eligible], ['applied', 'اعمال‌شده', c.ok], ['skipped', 'ارسال‌نشده', c.skipped], ['rejected', 'ردشده', c.rejected], ['failed', 'ناموفق', c.failed], ['unknown', 'نامعلوم', c.unknown]];
    return '<div class="outcome-strip" role="group" aria-label="خلاصه نتیجه گروهی">' + cells.map(function (x) { return '<div class="os os-' + x[0] + (x[2] ? '' : ' zero') + '"><span>' + x[1] + '</span><b>' + fa(x[2]) + '</b></div>'; }).join('') + '</div>' +
      '<p class="os-note">' + (c.ok + c.skipped + c.rejected + c.failed + c.unknown === c.requested ? ic('checkCircle') + 'مجموع «اعمال‌شده + ارسال‌نشده + ردشده + ناموفق + نامعلوم» برابر «درخواست‌شده» است.' : ic('alert') + 'جمع‌ها با هم نمی‌خوانند؛ پیش از هر اقدام بازخوانی کنید.') + '</p>';
  };
  X.counts = function (o) {
    var n = function (k) { return o.items.filter(function (x) { return x[1] === k; }).length; };
    return { requested: o.requested, eligible: o.requested - n('skipped') - n('rejected'), ok: n('ok'), skipped: n('skipped'), rejected: n('rejected'), failed: n('failed'), unknown: n('unknown') };
  };
  // BranchCoverage (shared presentation): which sources are really counted. Not-counted / unknown never renders as 0.
  var SRCST = { counted: ['شمرده می‌شود', 'ok', 'checkCircle'], notcounted: ['شمرده نمی‌شود', 'recon', 'xCircle'], unknown: ['پوشش اثبات نشده', 'undef', 'question'], failed: ['دریافت نشد', 'failed', 'xCircle'] };
  X.covMatrix = function (rows, caption) {
    return '<div class="cov-matrix" role="group" aria-label="' + esc(caption || 'پوشش منابع') + '">' + rows.map(function (r) {
      var s = SRCST[r.state] || SRCST.unknown;
      return '<div class="cm cm-' + s[1] + '"><span class="cm-n">' + esc(r.label) + '</span><span class="cm-s">' + ic(s[2]) + esc(s[0]) + '</span>' + (r.note ? '<span class="cm-t">' + esc(r.note) + '</span>' : '') + '</div>';
    }).join('') + '</div>';
  };
  X.sources = function () { return M.sources; };

  /* ---------- Exceptions (coordination only: each row names subject / owner / may / may-not / next actor / resolution signal; no escalation command) ---------- */
  X.exceptions = function () {
    var s = sim(), live = M.managers.filter(function (m) { return !X.mgrFailed(m.id); });
    var load = live.map(function (m) { return m.name + ' ' + fa(X.mgrLoad(m.id)); }).join(' · ');
    return M.exc.filter(function (e) { return !e.cond || e.cond === s; }).map(function (e) {
      if (e.id !== 'E2') return e;
      var c = {}; for (var k in e) c[k] = e[k];
      c.subject = 'پرونده باز نزد فروشندگان شاخه‌ها یکسان نیست: ' + load + (live.length < M.managers.length ? ' (شاخه دریافت‌نشده کنار گذاشته شد)' : '') + ' — فقط مقایسه واقعیت‌ها'; return c;
    });
  };
  X.excOf = function (id) { return X.exceptions().filter(function (e) { return e.id === id; })[0]; };
  X.attention = function () {
    var out = {}; X.exceptions().forEach(function (e) { (out[e.cls] = out[e.cls] || []).push(e); });
    return Object.keys(M.excClasses).filter(function (k) { return out[k]; }).map(function (k) {
      var list = out[k], c = M.excClasses[k], unk = list.some(function (e) { return !e.owner; });
      return { id: k, n: list.length, icon: c.icon, label: c.label, tone: unk ? 'slate' : (k === 'consumed' || k === 'invoice' ? 'red' : k === 'hr' || k === 'distconf' || k === 'mgrop' ? 'orange' : 'amber'), list: list, subject: list[0].subject, owner: unk ? 'مسئول نامشخص (UNKNOWN) برای بخشی از موارد' : list[0].owner, state: list[0].state };
    });
  };
  X.branchExc = function (mid) { return X.exceptions().filter(function (e) { return e.mgr === mid; }); };

  // Metric trust rows (Product Spec §8). A value is shown only when its definition and coverage allow it; otherwise a truthful state is shown.
  X.metricTrust = function () {
    var f = X.allFacts(), bad = M.managers.filter(function (m) { return X.mgrCov(m.id) !== 'ok' && X.mgrCov(m.id) !== 'recon'; }).length;
    var partial = bad ? 'partial' : 'ok';
    var openRows = M.invoices.filter(function (i) { return i.inv === 'pre'; }).length;
    var legacy = X.sum(M.sellers, 'legacy'), unkHist = M.invoices.filter(function (i) { return i.teamAtIssue == null; }).length;
    return [
      { id: 'created', name: 'فاکتور صادرشده', value: fa(f.created), cov: partial, covLabel: bad ? 'پوشش ناقص' : 'کامل', why: bad ? 'جمع فقط شاخه‌های دریافت‌شده را شامل می‌شود.' : 'شناسه یکتای فاکتور؛ نمونه نمایشی.', basis: 'range' },
      { id: 'openpre', name: 'پیش‌فاکتور باز', value: null, cov: 'recon', covLabel: 'نیازمند تطبیق (F02)', why: 'نمای کلی ' + fa(M.f02.tile) + ' نشان می‌دهد اما ' + fa(openRows) + ' ردیف پیش‌فاکتور در جدول فاکتورها هست؛ شرط شمارش «pre_invoice» را نمی‌شمارد.', basis: 'snap' },
      { id: 'issued', name: 'پیش‌فاکتور صادرشده', value: fa(f.created - unkHist), cov: 'partial', covLabel: 'سابقه ناقص', why: fa(unkHist) + ' مورد بدون سابقه انتساب هنگام صدور؛ ناشناخته می‌ماند و صفر نیست.', basis: 'range' },
      { id: 'done', name: 'فروش تکمیل‌شده', value: fa(f.completed), cov: 'undef', covLabel: 'تعریف نهایی نشده', why: 'مرز تکمیل و انتساب (OPD-06/07) باز است؛ رسید یا مرحله تأییدشده به‌تنهایی تکمیل نیست.', basis: 'range' },
      { id: 'cases', name: 'کل پرونده‌ها', value: null, cov: 'partial', covLabel: 'پوشش ناقص (F03)', why: 'فقط منبع قدیمی شمرده می‌شود؛ V4، MIS و Dot در شمارش پایه نیستند. صفر لید قدیمی به معنی صفر پرونده نیست.', basis: 'snap' },
      { id: 'legacy', name: 'فقط لیدهای قدیمی', value: fa(legacy), cov: bad ? 'partial' : 'ok', covLabel: bad ? 'پوشش ناقص' : 'فقط منبع قدیمی', why: 'با «کل پرونده‌ها» یکی نیست و آن را جایگزین نمی‌کند.', basis: 'snap' }
    ];
  };
})();
