/* MIS — role layer, part 1: state, namespaced dictionaries, source/lineage helpers and shared presentation helpers.
   Design prototype; mock data only. Nothing here imports, assigns, returns, deletes, repairs, approves or posts anything.
   Product boundaries: MIS-PRODUCT-SPEC.md + CRM-CROSS-ROLE-ARCHITECTURE-V1.md + GATE-0-PRODUCT-INVARIANTS.md.
   MIS = Data Operations + Source Quality + Custody Traceability + Governed Reporting. Visibility never becomes repair authority. */
(function () {
  'use strict';
  var C = window.CRM, M = window.MIS, h = C.h;
  var esc = h.esc, fa = h.fa, ic = h.ic, pill = h.pill, hint = h.hint;
  var X = window.MISX = { C: C, M: M, h: h, V: {}, D: {} };

  /* ---------- Role state ---------- */
  X.st = {
    iq: 'runs', vq: 'all', sq: 'all',               // import queue / source-row validity filter / source-cases filter
    cq: 'assign', asel: {}, recip: null, rf: 'all', rsel: {},   // custody: queue / selection / recipient / return filter / return selection
    rq: 'issues', icls: 'all', isel: {}, rid: 'rp1', tb: 'range',   // reconciliation + reports
    plan: 'P-30', qb: 'B-1186', qp: null, qn: 50, qr: '', qok: false,   // planning / quick delivery
    maintAuth: 'unverified', dq: 'log', flow: null, ops: [], done: {}, rep: null, run: null
  };

  /* ---------- Namespaced dictionaries (label + tone + icon; never colour alone) ---------- */
  X.VAL = { valid: { label: 'معتبر', tone: 'teal', icon: 'checkCircle' }, dup: { label: 'تکراری', tone: 'orange', icon: 'copy' }, invalid: { label: 'نامعتبر', tone: 'red', icon: 'xCircle' } };
  X.DUPC = { infile: 'تکراری در همان فایل', samebatch: 'ورود مجدد در همین دسته', othermis: 'ردیف MIS در دسته دیگر', legacy: 'لید قدیمی' };
  X.RUNS = { complete: ['کامل', 'green', 'checkCircle'], partial: ['ناقص (موفقیت جزئی)', 'orange', 'split'], failed: ['ناموفق', 'red', 'xCircle'], unknown: ['نتیجه نامعلوم', 'amber', 'question'] };
  X.RUNTYPE = { import: 'ورود فایل', assign: 'تخصیص منبع به مدیر', pool: 'آماده‌سازی استخر', quick: 'تحویل سریع' };
  X.RET = { ok: { label: 'قابل برگشت (فقط پیش‌نمایش)', tone: 'teal', icon: 'checkCircle' }, blocked: { label: 'مسدود', tone: 'red', icon: 'lock' }, unknown: { label: 'نامعلوم · نیاز به تطبیق', tone: 'amber', icon: 'question' }, conflict: { label: 'تعارض · تغییر کرده', tone: 'orange', icon: 'swap' }, na: { label: 'تحویل نشده · برگشت نامربوط', tone: 'slate', icon: 'dashed' } };
  X.OUT = {
    ok: { label: 'اعمال شد', tone: 'green', icon: 'checkCircle' }, skipped: { label: 'ارسال نشد', tone: 'orange', icon: 'swap' }, rejected: { label: 'ردشده توسط سامانه', tone: 'red', icon: 'ban' },
    failed: { label: 'ناموفق · قابل تکرار', tone: 'red', icon: 'refresh' }, unknown: { label: 'نامعلوم', tone: 'amber', icon: 'question' }
  };
  X.FIN = { none: ['یافت نشد', 'teal', 'checkCircle'], active: ['فعال · محافظت‌شده', 'red', 'lock'], cancelled: ['لغوشده · آزادسازی تعریف نشده', 'orange', 'ban'], unknown: ['نامعلوم · مسدودکننده', 'amber', 'question'] };
  X.FINDOM = { legacy: ['لید قدیمی (lead_id)', 'file'], v4: ['آیتم V4 (follow_invoice_id)', 'layers'], flow: ['جریان فروشنده / آخرین فاکتور', 'activity'], dot: ['ارتباط‌های مالی Dot', 'receipt'] };
  X.NODE = { mis: ['ردیف MIS', 'شناسه بومی'], legacy: ['لید قدیمی', 'شناسه بومی'], v4: ['آیتم توزیع V4', 'شناسه بومی'], dot: ['پرونده / فاکتور Dot', 'شناسه بومی'], case: ['پرونده منطقی (CaseRef)', 'شناسه منطقی'], alias: ['رابطه alias', 'نوع رابطه + مدرک'] };
  X.STAGE = { unassigned: 'نزد منبع — تحویل نشده', assigned: 'تخصیص منبع به مدیر (فیلد منبع)', delivered: 'تحویل‌شده', none: '—' };
  X.ISSTATE = { open: ['باز', 'orange', 'inbox'], awaiting: ['منتظر مالک حوزه', 'slate', 'hourglass'] };
  X.ZONES = { normal: ['عادی', 'teal', 'eye'], adv: ['پیشرفته · مشروط', 'amber', 'alert'], maint: ['نگهداری', 'red', 'lock'] };
  X.PLANST = [['draft', 'پیش‌نویس قانون'], ['preview', 'پیش‌نمایش برنامه'], ['saved', 'ذخیره برنامه'], ['approved', 'تأیید برنامه'], ['materialized', 'ایجاد لید زنده']];

  /* ---------- Entity helpers ---------- */
  (function genCandidates() {
    var base = [['B-1187', 8, 'verified'], ['B-1187', 9, 'verified'], ['B-1187', 10, 'partial'], ['B-1187', 11, 'verified'], ['B-1187', 14, 'verified'], ['B-1187', 15, 'unknown'], ['B-1187', 19, 'verified']];
    base.forEach(function (b, i) {
      var id = String(884214 + i);
      M.cases.push({ id: id, batch: b[0], row: b[1], val: 'valid', phone: '۰۹۱۲···' + fa(String(3000 + i * 71)), phoneMatches: 0, stage: 'unassigned', lin: b[2],
        ladder: [{ k: 'mis', v: fa(id), conf: 'verified', note: 'شناسه بومی ردیف MIS' }, { k: 'case', v: 'تعریف نشده', conf: 'unknown', note: 'حل‌کننده یکپارچه فعال نیست' }],
        fin: { legacy: ['none', '—'], v4: ['none', '—'], flow: ['none', '—'], dot: ['none', '—'] }, ret: 'na',
        cust: { src: 'MIS · دسته ' + b[0], cur: 'نزد منبع — تحویل نشده', orig: 'MIS (ورود منبع)', next: 'MIS: تخصیص به مدیر', credit: 'هنوز فروشی ثبت نشده', event: 'ورود فایل · سمیرا کاظمی', recip: '—' },
        hist: [['ورود از فایل', 'امروز ۰۹:۱۴', 'اجرای R-9921']] });
    });
  })();
  X.cs = function (id) { return M.cases.filter(function (c) { return c.id === String(id); })[0]; };
  X.run = function (id) { return M.runs.filter(function (r) { return r.id === id; })[0]; };
  X.batch = function (id) { return M.batches.filter(function (b) { return b.id === id; })[0]; };
  X.file = function (id) { return M.files.filter(function (f) { return f.id === id; })[0]; };
  X.issue = function (id) { return M.issues.filter(function (i) { return i.id === id; })[0]; };
  X.report = function (id) { return M.reports.filter(function (r) { return r.id === id; })[0]; };
  X.plan = function (id) { return M.plans.filter(function (p) { return p.id === id; })[0]; };
  X.mgr = function (id) { return M.managers.filter(function (m) { return m.id === id; })[0]; };
  X.maintOf = function (id) { return M.maint.filter(function (m) { return m.id === id; })[0]; };
  X.keys = function (o) { return Object.keys(o).filter(function (k) { return o[k]; }); };
  X.sim = function () { return C.state.sim; };
  X.cand = function () { return M.cases.filter(function (c) { return c.stage === 'unassigned' && c.val === 'valid'; }); };
  X.runsOf = function (type) { return M.runs.filter(function (r) { return r.type === type; }); };
  X.importRuns = function () { return X.runsOf('import'); };
  X.latestImport = function () { return X.importRuns()[0]; };
  X.latestOp = function () { return M.runs[0]; };
  X.skippedOf = function (r) { return r.dup + r.invalid; };
  X.sumOk = function (r) { return r.imported + r.dup + r.invalid + r.failed + r.unknown === r.seen; };
  X.finBlockers = function (c) { var o = []; Object.keys(c.fin).forEach(function (k) { if (c.fin[k][0] !== 'none') o.push(k); }); return o; };

  /* ---------- Shared presentation helpers (reusable patterns; no business capability) ---------- */
  var COV = {
    ok: ['کامل', 'ok', 'checkCircle', 'همه منابع این گزارش با بازه و تعریف انتخاب‌شده دریافت شده‌اند.'],
    partial: ['پوشش ناقص', 'partial', 'layers', 'بخشی از منابع دریافت نشده؛ جمع‌ها نهایی نیستند و صفر فرض نمی‌شوند.'],
    stale: ['قدیمی', 'stale', 'clock', 'آخرین گردآوری قدیمی است؛ پیش از تصمیم بازخوانی کنید.'],
    recon: ['نیازمند تطبیق', 'recon', 'swap', 'بین دو منبع اختلاف وجود دارد؛ عدد دستی تغییر داده نمی‌شود.'],
    undef: ['تعریف نشده', 'undef', 'question', 'تعریف این شاخص هنوز تصویب نشده؛ صفر نمایش داده نمی‌شود.'],
    failed: ['دریافت نشد', 'failed', 'xCircle', 'دریافت داده ناموفق بود؛ صفر فرض نشده است.'],
    bounded: ['سقف نمایش', 'partial', 'rows', 'فقط بخشی از ردیف‌ها گردآوری شده؛ نبود موارد بیشتر اثبات نمی‌شود.'],
    unauth: ['غیرمجاز', 'undef', 'lock', 'دسترسی به این داده تأیید نشده؛ صفر یا خالی نمایش داده نمی‌شود.']
  };
  X.cov = function (k, label) { var c = COV[k] || COV.ok; return '<span class="cov c-' + c[1] + ' tip" tabindex="0" data-tip="' + esc(c[3]) + '">' + ic(c[2]) + esc(label || c[0]) + '</span>'; };
  // LinkConfidence — operator-friendly: Verified · Partial · Conflicting · Unknown. Phone alone never reaches «تأییدشده».
  var LIN = {
    verified: ['تأییدشده', 'ok', 'checkCircle', 'اتصال با شناسه پایدار و مدرک ثبت‌شده اثبات شده است.'],
    partial: ['جزئی', 'partial', 'layers', 'مدرک بخشی از اتصال را پوشش می‌دهد یا فقط شماره موبایل دیده شده؛ شماره اثبات هویت پرونده نیست.'],
    conflict: ['متعارض', 'recon', 'swap', 'منابع درباره اتصال اختلاف دارند؛ تا تعیین مالک حوزه، وضعیت قطعی اعلام نمی‌شود.'],
    unknown: ['نامشخص', 'undef', 'question', 'مدرک کافی نیست و حدس زده نمی‌شود؛ «نامشخص» یعنی بدون اتصال نیست و بدون مصرف هم نیست.']
  };
  X.lin = function (k) { var c = LIN[k] || LIN.unknown; return '<span class="lc c-' + c[1] + ' tip" tabindex="0" data-tip="' + esc(c[3]) + '">' + ic(c[2]) + esc(c[0]) + '</span>'; };
  var BASIS = {
    snap: ['وضعیت در لحظه', 'clock', 'عکس فوری از وضعیت جاری هنگام گردآوری؛ سابقه گذشته را بازسازی نمی‌کند.'],
    range: ['رویداد در بازه + وضعیت جاری گردآوری', 'activity', 'پرونده‌هایی که رویدادشان در بازه است انتخاب می‌شوند؛ وضعیت آن‌ها در زمان گردآوری خوانده می‌شود. این «وضعیت تاریخی» نیست.'],
    asof: ['وضعیت تاریخی', 'history', 'وضعیت بازسازی‌شده در لحظه‌ای گذشته؛ فقط با مدرک کافی. داده جاری اثبات آن نیست.']
  };
  X.basis = function (k) { var b = BASIS[k], cls = k === 'snap' ? 'current' : k === 'asof' ? 'hist' : 'event'; return '<span class="basis b-' + cls + ' tip" tabindex="0" data-tip="' + esc(b[2]) + '">' + ic(b[1]) + b[0] + '</span>'; };
  X.grain = function (text) { return '<span class="grain tip" tabindex="0" data-tip="هر عدد فقط در همین واحد شمارش معتبر است؛ ردیف منبع، دسته، فاکتور و رویداد با هم جمع نمی‌شوند.">واحد شمارش: <b>' + esc(text) + '</b></span>'; };
  X.zone = function (k, extra) { var z = X.ZONES[k]; return '<span class="zone z-' + k + '">' + ic(z[2]) + z[0] + (extra ? ' · ' + esc(extra) : '') + '</span>'; };
  X.who = function (name, sub) { return '<div class="who"><b>' + esc(name) + '</b>' + (sub ? '<span class="phone">' + sub + '</span>' : '') + '</div>'; };
  // SourceRunContext — File › Batch › Import Run › Source Row are four different things; each is labelled, none is called «پرونده».
  X.srcCtx = function (o) {
    var seg = function (k, label, val, act) { return val ? '<span class="sc sc-' + k + '"><small>' + label + '</small>' + (act ? '<button type="button" class="linkish" data-act="' + act + '">' + esc(val) + '</button>' : '<b>' + esc(val) + '</b>') + '</span>' : ''; };
    return '<nav class="src-ctx" aria-label="زمینه منبع: فایل، دسته، اجرا، ردیف">' + [seg('file', 'فایل', o.file, o.file ? 'open-file:' + o.file : ''), seg('batch', 'دسته', o.batch, o.batch ? 'open-batch:' + o.batch : ''), seg('run', 'اجرا', o.run, o.run ? 'open-run:' + o.run : ''), seg('row', 'ردیف منبع', o.row, o.row ? 'open-case:' + o.row.replace(/\D/g, '') : '')].filter(Boolean).join('<span class="sc-sep" aria-hidden="true">‹</span>') + '</nav>';
  };
  X.freshPart = function (label) { var s = X.sim() === 'stale'; return { text: label + ': ' + (s ? M.freshness.stale + ' (قدیمی)' : (C.state.fresh === 'now' ? 'همین الان' : 'امروز ' + M.freshness.now)) + ' · ارزیابی ' + M.freshness.evaluated, stale: s }; };
  X.banners = function (view) {
    var s = X.sim(); if (!s) return '';
    if (s === 'stale') return h.banner('stale', '<b>گزارش‌های ذخیره‌شده قدیمی‌اند (' + M.freshness.stale + ').</b> عددها هنوز نمایش داده می‌شوند ولی وضعیت جاری را نشان نمی‌دهند؛ پیش از تصمیم بازخوانی کنید.', '<button type="button" class="btn btn-sm" data-refresh>' + ic('refresh') + 'بازخوانی</button>');
    if (s === 'incomplete' && (view === 'rec' || view === 'today')) return h.banner('incomplete', '<b>گزارش محدود/ناقص است.</b> فقط بخشی از جمعیت گردآوری شد؛ جمع‌ها نهایی نیستند و نبود مورد بیشتر اثبات نمی‌شود.', '<button type="button" class="btn btn-sm" data-refresh>' + ic('refresh') + 'تلاش مجدد</button>');
    if (s === 'healthcontra' && (view === 'import' || view === 'today' || view === 'diag')) return h.banner('incomplete', '<b>تناقض سلامت:</b> هشدار ذخیره‌شده می‌گوید ساختار ورود مشکل دارد ولی خواندن زنده گزارش موفق است. علت <b>نامعلوم</b> است؛ ترمیم کور پیشنهاد نمی‌شود و شمار «ورود ۰» نمایش داده نمی‌شود.', '');
    if (s === 'partialparse' && view === 'import') return h.banner('incomplete', '<b>تجزیه فایل ناقص است:</b> بخشی از ردیف‌ها خوانده نشد. این «فایل خالی» نیست؛ جمع‌ها نهایی نیستند و ورود مجدد کل فایل پیشنهاد نمی‌شود.', '');
    if (s === 'lineageconflict' && (view === 'cases' || view === 'rec')) return h.banner('incomplete', '<b>تعارض اتصال:</b> دو منبع درباره یک ردیف اختلاف دارند. هیچ ادغام یا اتصال مجدد خودکار انجام نمی‌شود؛ مالک حوزه باید اثبات کند.', '');
    if (s === 'reconmismatch' && view === 'rec') return h.banner('incomplete', '<b>عدم‌تطابق تطبیق:</b> مقایسه دو منبع با هم نمی‌خواند. این یک تشخیص است، نه مجوز نوشتن؛ اصلاح با مالک حوزه است.', '');
    if (s === 'maintblocked' && view === 'maint') return h.banner('locked', '<b>نگهداری مسدود است.</b> وابستگی محافظت‌شده یا نامعلوم وجود دارد؛ این نتیجه درست حفاظت است، نه خطا.', '');
    return '';
  };
  X.steps = h.steps;
  X.bulkbar = h.bulkbar;
  X.top = h.dtop;
  X.foot = h.dfoot;
  X.tl = h.timeline;
  // OwnershipGrid (7 concepts) — Source Owner · Current Custody · Original Owner · Next Actor · Credit Owner · Event Actor · Recipient. Never collapsed.
  X.own7 = function (o) {
    var cells = [['مالک منبع', 'actor/حوزه مسئول منبع و منشأ؛ مالک فعلی فروش نیست (Source Owner)', o.src, 'layers'], ['مسئول فعلی', 'پرونده الان نزد کیست؛ از فیلد assigned_manager منبع به‌تنهایی حدس زده نمی‌شود (Current Custody)', o.cur, 'user'], ['مالک اولیه', 'نخستین دریافت‌کننده طبق مدرک؛ با ساختار امروز عوض نمی‌شود (Original Owner)', o.orig, 'history'], ['اقدام بعدی با', 'چه کسی باید گام بعد را بردارد؛ نامشخص ← UNKNOWN (Next Actor)', o.next, 'arrowL'], ['مالک اعتبار', 'انتساب/استحقاق تاریخی؛ با تحویل یا تغییر مسئول عوض نمی‌شود (Credit Owner)', o.credit, 'wallet'], ['عامل رویداد', 'انجام‌دهنده آخرین انتقال با زمان و نتیجه (Event Actor)', o.event, 'activity'], ['گیرنده', 'گیرنده واقعی انتقال (Recipient)', o.recip, 'send']];
    return '<div class="own-grid own7">' + cells.map(function (x) { return '<div class="own"><span class="own-l">' + ic(x[3]) + esc(x[0]) + hint(x[1]) + '</span><b>' + esc(x[2]) + '</b></div>'; }).join('') + '</div>';
  };
  X.checks = function (list) { return '<ul class="elig">' + list.map(function (c) { var i = { ok: 'checkCircle', no: 'xCircle', warn: 'alert', q: 'question', info: 'info' }[c[0]]; return '<li class="e-' + c[0] + '">' + ic(i) + '<span><b>' + esc(c[1]) + '</b>' + (c[2] ? '<span>' + esc(c[2]) + '</span>' : '') + '</span></li>'; }).join('') + '</ul>'; };
  X.chain = function (stepsArr) { return '<ol class="chain">' + stepsArr.map(function (s) { var t = { done: 'check', cur: 'hourglass', rejected: 'xCircle', skip: 'dashed' }[s[1]]; return '<li class="c-' + s[1] + '"><span class="c-dot">' + ic(t) + '</span><span class="c-t"><b>' + esc(s[0]) + '</b><span>' + esc(s[2]) + '</span></span></li>'; }).join('') + '</ol>'; };
  X.sec = function (title, aside, inner, cls) { return '<section class="sec' + (cls ? ' ' + cls : '') + '"><div class="sec-h"><h3>' + title + '</h3>' + (aside ? '<span class="aside">' + aside + '</span>' : '') + '</div>' + inner + '</section>'; };
  X.details = function (title, count, inner, open) { return '<details class="sec"' + (open ? ' open' : '') + '><summary><h3>' + title + '</h3>' + (count != null ? '<span class="muted" style="margin-right:8px;font-size:var(--t-meta)">' + fa(count) + '</span>' : '') + '<span class="chev">' + ic('chev') + '</span></summary>' + inner + '</details>'; };
  // BulkOutcomeStrip (shared, SN-202) — requested · eligible · applied · skipped · rejected · failed · unknown (counts must reconcile).
  X.counts = function (o) {
    if (o.c) return o.c;
    var n = function (k) { return o.items.filter(function (x) { return x[1] === k; }).length; };
    return { requested: o.requested, eligible: o.requested - n('skipped') - n('rejected'), ok: n('ok'), skipped: n('skipped'), rejected: n('rejected'), failed: n('failed'), unknown: n('unknown') };
  };
  X.outcomeStrip = function (o) { return h.outcomeStrip(X.counts(o)); };
  // SourceCoverage matrix (shared SN-203 pattern): which sources are really counted. Not-counted / unknown never renders as 0.
  var SRCST = { counted: ['شمرده می‌شود', 'ok', 'checkCircle'], notcounted: ['شمرده نمی‌شود', 'recon', 'xCircle'], unknown: ['پوشش اثبات نشده', 'undef', 'question'], failed: ['دریافت نشد', 'failed', 'xCircle'] };
  X.covMatrix = function (rows, caption) {
    return '<div class="cov-matrix" role="group" aria-label="' + esc(caption || 'پوشش منابع') + '">' + rows.map(function (r) {
      var s = SRCST[r.state] || SRCST.unknown;
      return '<div class="cm cm-' + s[1] + '"><span class="cm-n">' + esc(r.label) + '</span><span class="cm-s">' + ic(s[2]) + esc(s[0]) + '</span>' + (r.note ? '<span class="cm-t">' + esc(r.note) + '</span>' : '') + '</div>';
    }).join('') + '</div>';
  };
  // LineageLadder — one node per identity domain, each with its own confidence. Phone is never a node.
  X.ladder = function (c) {
    return '<ol class="lin-ladder" aria-label="سلسله هویت و اتصال">' + c.ladder.map(function (n) {
      var k = X.NODE[n.k];
      return '<li class="ln ln-' + n.conf + ' ln-k-' + n.k + '"><span class="ln-k">' + esc(k[0]) + '<small>' + esc(k[1]) + '</small></span><span class="ln-v mono">' + esc(n.v) + '</span><span class="ln-c">' + X.lin(n.conf) + '</span>' + (n.note ? '<span class="ln-n">' + esc(n.note) + '</span>' : '') + '</li>';
    }).join('') + '</ol>';
  };
  // FinancialDependency union — every domain is shown; an absent row in one view is never proof of «not consumed».
  X.finTable = function (c) {
    return '<table class="tbl no-cursor fin-tbl" aria-label="اتحاد وابستگی‌های مالی"><caption class="sr">وابستگی مالی به تفکیک حوزه برای ردیف ' + esc(c.id) + '</caption><thead><tr><th>حوزه</th><th>نتیجه</th><th>شاهد</th></tr></thead><tbody>' + Object.keys(X.FINDOM).map(function (k) {
      var f = c.fin[k], d = X.FIN[f[0]], dm = X.FINDOM[k];
      return '<tr><td>' + ic(dm[1]) + ' ' + esc(dm[0]) + '</td><td>' + pill(d[0], d[1], d[2]) + '</td><td>' + esc(f[1]) + '</td></tr>';
    }).join('') + '</tbody></table>';
  };
  X.metricRows = function (list) {
    return list.map(function (m) {
      return '<tr><td><b>' + esc(m.name) + '</b></td><td class="n mt-v">' + (m.value == null ? '<span class="na tip" tabindex="0" data-tip="' + esc(m.why) + '">—</span>' : esc(m.value)) + '</td><td>' + X.cov(m.cov, m.covLabel) + '</td><td>' + X.basis(m.basis) + '</td><td class="col-opt muted">' + esc(m.why) + '</td></tr>';
    }).join('');
  };
  X.fmtTime = function (a, b) { return 'از ' + a + ' تا ' + b + ' (+۰۳:۳۰)'; };

  /* ---------- Today / Work Queue (derived from facts above; no invented counters) ---------- */
  X.counters = function () {
    var cs = M.cases, valid = cs.filter(function (c) { return c.val === 'valid'; }).length, waiting = X.cand().length, withSales = cs.filter(function (c) { return c.stage === 'delivered'; }).length, oldRet = cs.filter(function (c) { return c.oldReturnable; }).length;
    var b = M.batches.filter(function (x) { return x.id === 'B-1187'; })[0];
    return [
      { n: 'ردیف معتبر', p: 'val = valid', v: valid, t: 'ok', tip: 'همه ردیف‌های معتبر ذخیره‌شده، صرف‌نظر از تحویل' },
      { n: 'منتظر تحویل', p: 'معتبر و بدون تخصیص منبع', v: waiting, t: 'ok', tip: 'فقط فیلد منبع assigned_manager تهی است' },
      { n: 'نزد فروش', p: 'تحویل‌شده (رویداد انتقال)', v: withSales, t: 'ok', tip: 'ردیف‌هایی که حداقل یک انتقال دارند؛ مسئول فعلی را نمی‌گوید' },
      { n: 'قابل برگشت (نمای قدیمی)', p: 'فقط lead_id مجازی + فاکتور غیرلغو', v: oldRet, t: 'untrusted', tip: 'شرط ناقص (F01)؛ مجوز برگشت نیست' },
      { n: 'منتظر در سطح دسته', p: 'محاسبه جدا برای هر دسته', v: b && b.assigned === 0 ? 0 : null, t: 'diff', tip: 'شرط شمارش متفاوت از «منتظر تحویل»؛ با آن جمع یا تفاضل نمی‌شود' }
    ];
  };
  X.work = function () {
    var cands = X.cand().length, notVer = M.cases.filter(function (c) { return c.val === 'valid' && c.lin !== 'verified'; }).length, openIssues = M.issues.length;
    return [
      { id: 'w1', kind: 'import', tone: 'orange', icon: 'split', title: 'اجرای ورود R-9921 ناقص تمام شد', detail: '۹ ناموفق قابل تکرار · ۶ نامعلوم (ابتدا تطبیق) · ۲۲ نامعتبر · ۳۱ تکراری', own: 'MIS', act: 'open-run:R-9921' },
      { id: 'w2', kind: 'import', tone: 'amber', icon: 'question', title: 'نتیجه ۸۰ ردیف R-9908 نامعلوم است', detail: 'ورود مجدد قبل از تطبیق ممنوع است', own: 'MIS ← سرویس عملیات', act: 'open-run:R-9908' },
      { id: 'w3', kind: 'rows', tone: 'teal', icon: 'inbox', title: fa(cands) + ' ردیف معتبر منتظر تخصیص منبع', detail: 'تخصیص منبع با تحویل فعلی یکی نیست', own: 'MIS', act: 'goto:custody' },
      { id: 'w4', kind: 'return', tone: 'red', icon: 'lock', title: 'برگشت ردیف ۸۸۴۲۰۴ مسدود است (F01)', detail: 'نمای قدیمی «قابل برگشت» می‌گوید، ولی پیش‌فاکتور V4 فعال است', own: 'مهندسی + مالی', act: 'open-ret:884204' },
      { id: 'w5', kind: 'custody', tone: 'amber', icon: 'user', title: 'سابقه انتقال ردیف ۸۸۴۲۰۵ ناقص است', detail: 'عامل رویداد نامشخص (UNKNOWN)', own: 'مالک تحویل (فروش)', act: 'open-case:884205' },
      { id: 'w6', kind: 'custody', tone: 'orange', icon: 'swap', title: 'مسئول فعلی ردیف ۸۸۴۲۱۲ پس از پیش‌نمایش تغییر کرد', detail: 'پیش از هر اقدام بازخوانی شود', own: 'MIS', act: 'open-ret:884212' },
      { id: 'w7', kind: 'trust', tone: 'amber', icon: 'chart', title: 'اعتماد گزارش: F02 · F03 · F04 · گزارش قدیمی', detail: 'شاخص‌های وابسته «نیازمند تطبیق/ناقص» نمایش داده می‌شوند', own: 'مالک گزارش', act: 'goto-rec:reports' },
      { id: 'w8', kind: 'identity', tone: 'slate', icon: 'link', title: fa(notVer) + ' ردیف معتبر با اتصال غیرتأییدشده', detail: 'جزئی/متعارض/نامشخص؛ هیچ ادغامی انجام نمی‌شود', own: 'مالک حل‌کننده', act: 'goto-cases:lin' },
      { id: 'w9', kind: 'health', tone: 'amber', icon: 'activity', title: 'تناقض سلامت: هشدار ذخیره‌شده در برابر خواندن زنده', detail: 'علت نامعلوم · آخرین بررسی ' + M.health.checkedAt, own: 'مالک فنی', act: 'goto:diag' }
    ].concat(openIssues ? [] : []);
  };
})();
