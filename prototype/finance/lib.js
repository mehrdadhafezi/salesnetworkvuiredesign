/* Finance — role layer, part 1: state, namespaced dictionaries, financial-state helpers and shared presentation helpers.
   Design prototype; mock data only. FINANCE ROLE RUNTIME IS NOT LIVE VERIFIED. Nothing here approves, rejects, refunds, posts, reverses or exports.
   Finance = Financial Review + Reconciliation + Ledger Control + Ruled Posting. Visibility never becomes write authority (F06). Reads are pure (F07). */
(function () {
  'use strict';
  var C = window.CRM, M = window.FIN, h = C.h;
  var esc = h.esc, fa = h.fa, ic = h.ic, pill = h.pill, hint = h.hint, num = h.num;
  var X = window.FINX = { C: C, M: M, h: h, V: {}, D: {} };

  X.st = { rq: 'needs', sel: {}, lq: 'all', ldom: 'all', runq: 'runs', recq: 'issues', isel: {}, perm: 'full', ops: [], flow: null, local: {}, runLocal: {}, lookups: {}, auditSeen: 0, rep: null, spec: null };

  /* ---------- Namespaced dictionaries (label + tone + icon; never colour alone) ---------- */
  X.REV = { pending: { label: 'در انتظار بررسی مالی', tone: 'orange', icon: 'hourglass' }, approved: { label: 'تأیید مالی شد', tone: 'green', icon: 'checkCircle' }, rejected: { label: 'رد شد (بازگشت برای اصلاح)', tone: 'red', icon: 'xCircle' } };
  X.EVK = { receipt: { label: 'رسید بارگذاری‌شده', tone: 'blue', icon: 'receipt' }, gateway: { label: 'ثبت درگاهی', tone: 'violet', icon: 'link' }, none: { label: 'مدرک موجود نیست', tone: 'amber', icon: 'dashed' } };
  X.LK = { ok: ['پیوند تأییدشده', 'ok', 'checkCircle', 'Case/منبع با شناسهٔ پایدار اثبات شده است.'], unknown: ['پیوند نامعلوم', 'unknown', 'question', 'شواهد پیوند کافی نیست؛ اقدام وابسته مسدود است و با شماره تلفن حدس زده نمی‌شود.'], conflict: ['پیوند متعارض', 'conflict', 'swap', 'شواهد متناقض است؛ باید تطبیق شود.'] };
  X.INV = { 'در جریان وصول': 'slate', 'پرداخت کامل (طبق مبلغ معتبر)': 'teal' };
  X.RFS = { requested: { label: 'درخواست‌شده', tone: 'slate', icon: 'inbox' }, reported: { label: 'گزارش‌شده', tone: 'blue', icon: 'file' }, proof_pending: { label: 'مدرک در انتظار', tone: 'orange', icon: 'hourglass' }, confirmed: { label: 'تأییدشده', tone: 'green', icon: 'checkCircle' }, partial: { label: 'جزئی (ثبت‌شده)', tone: 'teal', icon: 'split' }, full: { label: 'کامل', tone: 'green', icon: 'checkCircle' }, unknown: { label: 'نامعلوم', tone: 'amber', icon: 'question' } };
  X.LGS = { committed: { label: 'ثبت‌شده (تراکنش)', tone: 'green', icon: 'checkCircle' }, preview: { label: 'پیش‌نمایش (تراکنش نیست)', tone: 'blue', icon: 'eye' }, unknown: { label: 'نامعلوم (ثبت‌شده فرض نمی‌شود)', tone: 'amber', icon: 'question' }, reconciled: { label: 'تطبیق‌شده', tone: 'teal', icon: 'checkCircle' } };
  X.RUN = {
    draft: { label: 'پیش‌نویس', tone: 'slate', icon: 'edit' }, preview: { label: 'پیش‌نمایش', tone: 'blue', icon: 'eye' }, approved: { label: 'تأییدشده (ثبت‌نشده)', tone: 'teal', icon: 'check' }, posting: { label: 'در حال ثبت', tone: 'orange', icon: 'refresh' },
    posted: { label: 'ثبت‌شده', tone: 'green', icon: 'checkCircle' }, partial: { label: 'ناقص (موفقیت جزئی)', tone: 'orange', icon: 'split' }, failed: { label: 'ناموفق', tone: 'red', icon: 'xCircle' }, unknown: { label: 'نتیجه نامعلوم', tone: 'amber', icon: 'question' }, reconciled: { label: 'تطبیق‌شده', tone: 'teal', icon: 'checkCircle' }
  };
  X.RUNORD = ['draft', 'preview', 'approved', 'posting', 'posted', 'partial', 'failed', 'unknown', 'reconciled'];
  X.ITEMR = { posted: ['ثبت‌شد', 'green', 'checkCircle'], existing: ['تراکنش موجود (اعتبار جدید ساخته نشد)', 'teal', 'link'], failed: ['ناموفق', 'red', 'xCircle'], unprocessed: ['پردازش‌نشده (دنباله)', 'orange', 'hourglass'], unknown: ['نامعلوم', 'amber', 'question'], notposted: ['هنوز ثبت نشده', 'slate', 'dashed'] };
  X.ISS = { amount: ['اختلاف مبلغ', 'wallet'], dup: ['پرداخت تکراری', 'copy'], mirror: ['نبود آینه', 'unlink'], stage: ['ناهمخوانی مرحله/بررسی', 'layers'], report: ['ناهمخوانی گزارش/فاکتور', 'chart'], refund: ['ناهمخوانی استرداد', 'repeat'], ledger: ['ناهمخوانی دفتر کل', 'wallet'], unknown: ['نتیجه ثبت نامعلوم', 'question'], unit: ['ابهام واحد/ارز', 'alert'], orphan: ['رابطهٔ شبه‌یتیم', 'unlink'] };
  X.ISST = { open: ['باز', 'orange', 'inbox'], awaiting: ['منتظر مالک', 'slate', 'hourglass'] };
  X.ZONES = { normal: ['روزانه', 'z-normal', 'eye'], cond: ['مشروط', 'z-adv', 'alert'], restr: ['پیشرفته · محدود', 'z-maint', 'lock'] };
  X.OUT = { ok: { label: 'ثبت شد', tone: 'green', icon: 'checkCircle' }, existing: { label: 'تراکنش موجود', tone: 'teal', icon: 'link' }, skipped: { label: 'ارسال نشد (واجد شرایط نبود)', tone: 'orange', icon: 'swap' }, failed: { label: 'ناموفق', tone: 'red', icon: 'xCircle' }, unknown: { label: 'نامعلوم', tone: 'amber', icon: 'question' } };
  X.OPS_S = function (k) { return { complete: ['کامل (با مدرک هر مورد)', 'green', 'checkCircle'], partial: ['ناقص (موفقیت جزئی)', 'orange', 'split'], failed: ['ناموفق (بدون اثر)', 'red', 'xCircle'], blocked: ['مسدود (بدون تغییر)', 'slate', 'ban'], unknown: ['نتیجه نامعلوم', 'amber', 'question'], conflict: ['تعارض (بدون تغییر)', 'red', 'swap'], unauthorized: ['غیرمجاز — UNAUTHORIZED (بدون تغییر)', 'red', 'lock'] }[k]; };

  /* ---------- Entity helpers ---------- */
  X.q = function (id) { return M.queue.filter(function (r) { return r.id === id; })[0]; };
  X.run = function (id) { return M.runs.filter(function (r) { return r.id === id; })[0]; };
  X.rf = function (id) { return M.refunds.filter(function (r) { return r.id === id; })[0]; };
  X.tx = function (id) { return M.ledger.filter(function (r) { return r.id === id; })[0]; };
  X.iss = function (id) { return M.issues.filter(function (r) { return r.id === id; })[0]; };
  X.rep = function (id) { return M.reports.filter(function (r) { return r.id === id; })[0]; };
  X.opOf = function (id) { return X.st.ops.filter(function (o) { return o.id === id; })[0]; };
  X.audit = function (id) { return M.audit.filter(function (r) { return r.id === id; })[0]; };
  X.sim = function () { return C.state.sim; };
  X.keys = function (o) { return Object.keys(o).filter(function (k) { return o[k]; }); };

  /* ---------- Action-aware permission awareness (F06): Can View ≠ Can Execute. Conceptual names only — not WordPress capabilities. ---------- */
  X.perm = function (k) { return M.perms.filter(function (p) { return p.k === k; })[0]; };
  X.can = function (k) {
    var p = X.perm(k), v = X.st.perm === 'view';
    if (v) return { ok: false, why: 'حالت «فقط مشاهده»: دیدن این صفحه به‌معنی اختیار اجرا نیست. اقدام مفهومی لازم: ' + p.concept + ' (نام‌ها مفهومی‌اند؛ مجوز واقعی ساخته نشده).' };
    if (p.onlyAdmin) return { ok: false, why: 'این اقدام در مرز فعلی فقط مدیر سیستم است؛ اختیار مالی داده نشده است (' + p.concept + ').' };
    if (p.cond === 'OPD-04') return { ok: false, why: 'مشروط: سیاست مدرک بانکی و تأییدکنندهٔ مجاز استرداد (OPD-04) تعریف نشده است؛ اجرای استرداد از این رابط وجود ندارد.' };
    if (p.cond === 'export') return { ok: false, why: 'خروجی گرفتن اختیار جدا با محدودهٔ ستون و ممیزی دارد و در این طرح فعال نیست.' };
    return { ok: true, why: '' };
  };
  // The COMMAND HANDLER is the action boundary: every commit re-checks authority, whatever opened the dialog (buttons, deep links, flow helpers).
  X.PKIND = { approve: 'approve', reject: 'reject', rungen: 'runapprove', runapprove: 'runapprove', runpost: 'post', runtail: 'post', runrecon: 'reconcile', bulkapp: 'approve', retry: 'approve', reconcileop: 'reconcile', handoff: 'reconcile' };
  X.denied = function (kind, title, quiet) {
    var g = X.can(X.PKIND[kind]); if (g.ok) return null;
    if (quiet) return { why: g.why };
    return X.newOp({ title: (title || 'اقدام') + ' — رد شد (UNAUTHORIZED)', kind: 'unauthorized', unauthorized: true, requested: 1, c: { requested: 1, eligible: 0, ok: 0, existing: 0, skipped: 1, failed: 0, unknown: 0 }, note: g.why + ' هیچ تغییری ثبت نشد.' });
  };
  X.guardBtn = function (k, cls, act, label, iconName, extraDis) {
    var g = X.can(k), dis = !g.ok || extraDis;
    var b = '<button type="button" class="btn ' + cls + (dis ? ' tip' : '') + '" data-act="' + act + '"' + (dis ? ' aria-disabled="true"' : '') + (!g.ok ? ' data-tip="' + esc(g.why) + '"' : '') + '>' + (iconName ? ic(iconName) : '') + label + '</button>';
    return b;
  };

  /* ---------- Money / unit: unknown amount or unit never renders as 0 or as a default unit ---------- */
  X.unitL = function (u) { return u === 'toman' ? 'تومان' : u === 'rial' ? 'ریال' : null; };
  X.na = function (t) { return '<span class="na tip" tabindex="0" data-tip="' + esc(t || 'داده در دسترس نیست') + '">—</span>'; };
  X.amt = function (n, unit, naText) {
    if (n == null) return X.na(naText || 'مبلغ نامعلوم است؛ صفر فرض نمی‌شود');
    var u = X.unitL(unit);
    return '<span class="amount"><span class="num">' + num(n) + '</span>' + (u ? '<small>' + u + '</small>' : '<small class="u-amb tip" tabindex="0" data-tip="واحد (تومان/ریال) مشخص نیست؛ بدون قرارداد مصوب (OPD-06) حدس زده نمی‌شود">واحد نامشخص</small>') + '</span>';
  };
  X.amtT = function (n, unit) { return n == null ? 'نامعلوم' : fa(num(n)) + ' ' + (X.unitL(unit) || '(واحد نامشخص)'); };

  /* ---------- Shared presentation helpers (reusable patterns; no business capability) ---------- */
  X.cov = function (k, label) {
    var c = { ok: ['کامل', 'ok', 'checkCircle', 'همهٔ منابع این نما دریافت شده‌اند.'], partial: ['ناقص', 'partial', 'layers', 'بخشی از داده یا منبع دریافت نشده؛ صفر فرض نمی‌شود.'], bounded: ['محدود (زیرمجموعه)', 'bounded', 'filter', 'نتیجه به سقف فراخوانی محدود است؛ خروجی کامل نیست.'], stale: ['قدیمی', 'stale', 'clock', 'داده قدیمی است؛ پیش از اتکا بازخوانی کنید.'] }[k];
    return '<span class="cov c-' + c[1] + ' tip" tabindex="0" data-tip="' + esc(c[3]) + '">' + ic(c[2]) + esc(label || c[0]) + '</span>';
  };
  X.lk = function (k, label) { var c = X.LK[k]; return '<span class="lc c-' + c[1] + ' tip" tabindex="0" data-tip="' + esc(c[3]) + '">' + ic(c[2]) + esc(label || c[0]) + '</span>'; };
  X.zone = function (k, extra) { var z = X.ZONES[k]; return '<span class="zone ' + z[1] + '">' + ic(z[2]) + z[0] + (extra ? ' · ' + esc(extra) : '') + '</span>'; };
  X.basis = function (k) {
    var b = { snap: ['وضعیت فعلی', 'clock', 'وضعیت مالی همین لحظه؛ تاریخ گذشته را بازسازی نمی‌کند.', 'current'], hist: ['تاریخی (As-Of)', 'history', 'فقط با شواهد بازیابی‌پذیر؛ در غیر این صورت نامعلوم.', 'hist'], range: ['رویداد در بازه', 'history', 'رویدادهای ثبت‌شده در یک بازه؛ وضعیت مالی آن روز نیست.', 'hist'] }[k];
    return '<span class="basis b-' + b[3] + ' tip" tabindex="0" data-tip="' + esc(b[2]) + '">' + ic(b[1]) + b[0] + '</span>';
  };
  X.grain = function (t) { return '<span class="grain tip" tabindex="0" data-tip="هر عدد فقط در همین واحد شمارش معتبر است؛ مرحله، فاکتور، تراکنش و اجرا با هم جمع نمی‌شوند.">واحد شمارش: <b>' + esc(t) + '</b></span>'; };
  X.freshPart = function (label) { var s = X.sim() === 'stale'; return { text: label + ': ' + (s ? M.freshness.stale + ' (قدیمی)' : 'امروز ' + M.freshness.now) + ' · ارزیابی ' + M.freshness.evaluated, stale: s }; };
  X.revPill = function (k) { var d = X.REV[k]; return pill(d.label, d.tone, d.icon); };
  X.runPill = function (k) { var d = X.RUN[k]; return pill(d.label, d.tone, d.icon); };
  X.lgPill = function (k) { var d = X.LGS[k]; return pill(d.label, d.tone, d.icon); };
  X.rfPill = function (k) { var d = X.RFS[k]; return pill(d.label, d.tone, d.icon); };

  /* FinancialStateLayers — independent dimensions, each with what it must NOT be inferred from. Not one giant status badge. */
  X.layers = function (r) {
    var ev = r.ev ? pill(X.EVK[r.ev.k].label, X.EVK[r.ev.k].tone, X.EVK[r.ev.k].icon) : pill(X.EVK.none.label, 'amber', 'dashed');
    var ld = { unposted: pill('ثبت‌نشده / تعریف‌نشده', 'slate', 'dashed'), unknown: pill('نامعلوم', 'amber', 'question'), posted: pill('ثبت‌شده', 'green', 'checkCircle') }[r.ledger];
    var rfs = r.refund === 'none' ? pill('بدون درخواست', 'slate', 'dashed') : X.rfPill(r.refund);
    var ent = { none: pill('ایجاد نشده', 'slate', 'dashed'), earned: pill('کسب‌شده (اعتبار کیف پول نیست)', 'amber', 'hourglass') }[r.ent];
    var cell = function (l, v, notFrom, extra) { return '<div class="fsl-c"><span class="fsl-l">' + l + '</span><b class="fsl-v">' + v + '</b>' + (extra ? '<span class="fsl-x">' + extra + '</span>' : '') + '<span class="fsl-n">' + ic('ban') + 'استنتاج نشود از: ' + notFrom + '</span></div>'; };
    return '<div class="fsl" role="group" aria-label="ابعاد مستقل وضعیت مالی">' +
      cell('وضعیت فاکتور', pill(r.invSt, X.INV[r.invSt] || 'slate', r.paid >= r.total ? 'checkCircle' : 'clock'), 'مرحله، مدرک یا دفتر کل') +
      cell('مرحلهٔ پرداخت', pill('مرحله ' + fa(r.stage[0]) + ' از ' + fa(r.stage[1]) + ' · ' + r.purpose, 'blue', 'layers'), 'تکمیل کل فاکتور') +
      cell('بررسی مالی', X.revPill(r.review), 'ثبت درگاهی', r.prior.length ? 'آخرین: ' + esc(r.prior[r.prior.length - 1].who) + ' · ' + esc(r.prior[r.prior.length - 1].at) : '') +
      cell('مدرک پرداخت', ev, 'وصول قطعی پول') +
      cell('وصول‌شده معتبر', X.amt(r.paid, r.unit), 'مجموع ادعاهای رسید یا اعتبار کیف پول') +
      cell('باقی‌مانده', X.amt(r.rem, r.unit), 'برچسب لغو یا نمایش صفر') +
      cell('وضعیت استرداد', rfs, 'لغو فاکتور') +
      cell('ثبت در دفتر کل', ld, 'اجرای تأییدشده یا پیش‌نمایش') +
      cell('استحقاق (کمیسیون)', ent, 'تأیید مرحله یا وصول کامل') + '</div>';
  };

  X.covRow = function (c, label) {
    var cells = [['intended', 'نیت شده (کل)', c.intended], ['processed', 'پردازش‌شده', c.processed], ['posted', 'ثبت‌شده', c.posted], ['existing', 'موجود/ردشده', c.existing], ['failed', 'ناموفق', c.failed], ['unprocessed', 'پردازش‌نشده', c.unprocessed], ['unknown', 'نامعلوم', c.unknown]];
    return '<div class="outcome-strip cov-strip" role="group" aria-label="' + esc(label || 'پوشش اجرا') + '">' + cells.map(function (x) { return '<div class="os os-' + x[0] + (x[2] ? '' : ' zero') + '"><span>' + x[1] + '</span><b>' + (x[2] == null ? '<span class="na tip" tabindex="0" data-tip="نامعلوم است؛ صفر فرض نمی‌شود">—</span>' : fa(num(x[2]))) + '</b></div>'; }).join('') + '</div>';
  };
  X.covNote = function (c) {
    if (c.processed == null) return '<p class="os-note">' + ic('question') + 'پوشش نامعلوم است؛ هیچ شمارشی ثبت‌شده فرض نمی‌شود.</p>';
    var ok = c.posted + c.existing + c.failed + c.unknown === c.processed && c.processed + c.unprocessed === c.intended;
    return '<p class="os-note">' + (ok ? ic('checkCircle') + 'ثبت‌شده + موجود + ناموفق + نامعلوم = پردازش‌شده؛ پردازش‌شده + پردازش‌نشده = کل نیت‌شده.' : ic('alert') + 'جمع‌ها با هم نمی‌خوانند؛ پیش از هر اقدام بازخوانی کنید.') + '</p>' + (c.unprocessed ? '<p class="os-note warn">' + ic('alert') + '«ok/partial» یک فراخوانی محدود به ۲۰۰۰ آیتم، موفقیت کل اجرا نیست؛ ' + fa(num(c.unprocessed)) + ' آیتم هنوز پردازش نشده است.</p>' : '');
  };

  /* OutcomeUnknownContext — when a financial effect may have committed: original intent · key · possible state · lookup requirement · retry eligibility · next safe action. Blind retry is never offered. */
  X.oux = function (o) {
    var row = function (l, v) { return '<div><dt>' + l + '</dt><dd>' + v + '</dd></div>'; };
    return '<div class="oux" role="group" aria-label="زمینهٔ نتیجه نامعلوم و تکرارپذیری"><div class="oux-h">' + ic('question') + '<b>نتیجه ممکن است ثبت شده باشد — تکرار کور ممنوع</b></div><dl class="oux-dl">' +
      row('نیت اصلی', o.intent) + row('شناسه تراکنش / کلید کسب‌وکار', '<span class="mono">' + o.key + '</span>') + row('وضعیت ممکن', o.possible) + row('الزام جستجو/تطبیق', o.lookup) + row('تراکنش موجود (اگر یافت شد)', o.existing) + row('شرط مجاز بودن تکرار', o.retry) + '</dl>' +
      '<div class="oux-next">' + ic('shield') + '<span><b>گام امن بعدی:</b> ' + o.next + '</span></div></div>';
  };

  X.top = h.dtop;
  X.foot = h.dfoot;
  X.sec = function (title, aside, inner, cls) { return '<section class="sec' + (cls ? ' ' + cls : '') + '"><div class="sec-h"><h3>' + title + '</h3>' + (aside ? '<span class="aside">' + aside + '</span>' : '') + '</div>' + inner + '</section>'; };
  X.checks = function (list) { return '<ul class="elig">' + list.map(function (c) { var i = { ok: 'checkCircle', no: 'xCircle', warn: 'alert', q: 'question', info: 'info' }[c[0]]; return '<li class="e-' + c[0] + '">' + ic(i) + '<span><b>' + esc(c[1]) + '</b>' + (c[2] ? '<span>' + esc(c[2]) + '</span>' : '') + '</span></li>'; }).join('') + '</ul>'; };
  X.tl = h.timeline;
  X.steps = h.steps;
  X.bulkbar = h.bulkbar;
  X.dl = function (rows, cls) { return '<dl class="exc-dl ' + (cls || 'hcp') + '">' + rows.map(function (e) { return '<div><dt>' + e[0] + '</dt><dd>' + e[1] + '</dd></div>'; }).join('') + '</dl>'; };

  /* ---------- Review eligibility: pure derivation from mock facts. Unknown/ambiguous inputs fail closed. ---------- */
  X.elig = function (r) {
    var cur = X.localOf(r), c = [], block = false, dec = null;
    if (cur.review !== 'pending') { c.push(['no', 'این مرحله اکنون قابل بررسی نیست', 'وضعیت فعلی: ' + X.REV[cur.review].label]); return { checks: c, blocked: true, reject: false, reasonTag: 'not-reviewable' }; }
    if (r.stale && !cur.reloaded) { c.push(['no', 'تعارض: وضعیت مرحله پس از بارگذاری تغییر کرده است', 'بازبین دیگر (' + r.stale.by + ') ' + r.stale.at + ' آن را «' + X.REV[r.stale.res].label + '» کرد؛ پیش از هر نیت جدید بازخوانی لازم است']); return { checks: c, blocked: true, reject: false, reasonTag: 'stale' }; }
    c.push(['ok', 'مرحله هنوز در انتظار بررسی است', 'دوباره هنگام ثبت بررسی می‌شود (نزدیک commit)']);
    var miss = !r.ev, mism = r.ev && r.ev.amt !== r.claimed, unit = !r.unit, link = r.lk !== 'ok', dup = r.flags.indexOf('dup') > -1, gw = r.flags.indexOf('gatewayOnly') > -1;
    if (miss) { c.push(['no', 'مدرک پرداخت موجود نیست', 'تأیید ممکن نیست؛ رد با دلیل «مدرک ناموجود» ممکن است']); block = true; } else c.push(['ok', 'مدرک موجود است', r.ev.ref + ' ' + r.ev.ver + ' · ' + r.ev.src + ' · ' + r.ev.at]);
    if (mism) { c.push(['no', 'مبلغ مدرک با مبلغ ادعاشده نمی‌خواند', 'مدرک ' + fa(num(r.ev.amt)) + ' — ادعا ' + fa(num(r.claimed)) + ' (' + X.unitL(r.unit) + ')']); block = true; } else if (!miss) c.push(['ok', 'مبلغ مدرک با ادعا می‌خواند', fa(num(r.claimed)) + ' ' + (X.unitL(r.unit) || '')]);
    if (unit) { c.push(['no', 'واحد مبلغ نامشخص است (OPD-06)', 'تومان/ریال حدس زده نمی‌شود؛ تا قرارداد مصوب، اقدام وابسته مسدود است']); block = true; } else c.push(['ok', 'واحد مشخص است', X.unitL(r.unit)]);
    if (link) { c.push(['no', 'پیوند Case/منبع ' + (r.lk === 'unknown' ? 'نامعلوم' : 'متعارض') + ' است', 'اقدام وابسته مسدود می‌ماند؛ اتصال با شماره تلفن انجام نمی‌شود']); block = true; } else c.push(['ok', 'پیوند Case/منبع اثبات‌شده است', r.cs]);
    if (dup) { c.push(['no', 'پرداخت تکراری احتمالی', r.dupOf]); block = true; }
    if (gw) c.push(['warn', 'فقط ثبت درگاهی است', 'پرداخت درگاهی ثبت‌شده ≠ تأیید مالی؛ تأیید دستی مالی جدا لازم است']);
    return { checks: c, blocked: block, reject: true, reasonTag: miss ? 'missing' : mism ? 'mismatch' : unit ? 'unit' : link ? 'link' : dup ? 'dup' : null };
  };
  // Local (prototype-only) decision overlay; never written back to mock data used by other roles.
  X.localOf = function (r) { var l = X.st.local[r.id]; return l ? l : { review: r.review, reloaded: false }; };
  X.eff = function (r) {
    var l = X.st.local[r.id], o = {}; for (var k in r) o[k] = r[k]; if (l) for (k in l) o[k] = l[k];
    // Invoice-level paid/remaining = loaded paid + every stage of the SAME invoice approved in this session (stage review state stays per stage).
    var add = 0; M.queue.forEach(function (s) { var sl = X.st.local[s.id]; if (s.inv === r.inv && sl && sl.approvedClaimed) add += sl.approvedClaimed; });
    o.paid = r.paid + add; o.rem = Math.max(0, r.total - o.paid); return o;
  };
  X.impact = function (r) {
    var v = r.claimed, np = r.paid + v, nr = Math.max(0, r.total - np), last = r.stage[0] === r.stage[1];
    return { np: np, nr: nr, last: last };
  };

  /* ---------- Operation engine: per-item truth. requested = ok + existing + skipped + failed + unknown. ---------- */
  var seq = 400;
  X.newOp = function (o) { o.id = 'FOP-' + (++seq); o.ts = 'همین الان'; X.st.ops.unshift(o); return o; };
  X.counts = function (o) {
    if (o.c) return o.c;
    var n = function (k) { return o.items.filter(function (x) { return x[1] === k; }).length; };
    return { requested: o.requested, eligible: o.requested - n('skipped'), ok: n('ok'), existing: n('existing'), skipped: n('skipped'), failed: n('failed'), unknown: n('unknown') };
  };
  X.opState = function (op) { if (op.unauthorized) return 'unauthorized'; if (op.conflict) return 'conflict'; if (op.blocked) return 'blocked'; var c = X.counts(op); if (c.unknown) return 'unknown'; var good = c.ok + c.existing; return good === c.requested ? 'complete' : good === 0 ? 'failed' : 'partial'; };
  X.outcomeStrip = function (o) { // Finance vocabulary: applied = ثبت‌شده (ruled posting), plus «موجود»; no bulk «ردشده» cell
    var c = X.counts(o);
    return h.outcomeStrip(c, { cells: [['requested', 'درخواست‌شده', c.requested], ['eligible', 'واجد شرایط', c.eligible], ['applied', 'ثبت‌شده', c.ok], ['existing', 'تراکنش/تصمیم موجود', c.existing], ['skipped', 'ارسال‌نشده', c.skipped], ['failed', 'ناموفق', c.failed], ['unknown', 'نامعلوم', c.unknown]], sumText: 'ثبت‌شده + موجود + ارسال‌نشده + ناموفق + نامعلوم = درخواست‌شده.' });
  };

  /* ---------- Page banners for simulated shared states ---------- */
  X.banners = function (view) {
    var s = X.sim(), b = function (k, t) { return h.banner(k, t, ''); }, out = '';
    if (X.st.perm === 'view') out += h.banner('locked', '<b>حالت فقط‌مشاهده:</b> دیدن صف و دفتر کل به‌معنی اختیار اجرا نیست. هر اقدام نوشتنی با اقدام مفهومی لازم و دلیل غیرفعال بودن نمایش داده می‌شود (F06؛ نام‌ها مفهومی‌اند و مجوز واقعی ساخته نشده).', '');
    if (s === 'stale') out += b('stale', '<b>صف قدیمی است (' + M.freshness.stale + ').</b> مرحله یا اجرا ممکن است عوض شده باشد؛ پیش از هر تصمیم بازخوانی کنید. اقدام روی ردیف کش‌شده انجام نمی‌شود.');
    if (s === 'incomplete' && (view === 'rev' || view === 'led' || view === 'rep')) out += b('incomplete', '<b>داده ناقص است:</b> بخشی از مدارک/تراکنش‌ها دریافت نشد. جمع‌ها نهایی نیستند و نبود مورد صفر بودن نیست.');
    if (s === 'unitamb' && (view === 'rev' || view === 'rep')) out += b('incomplete', '<b>ابهام واحد (OPD-06):</b> برخی مبالغ بدون واحد روشن‌اند. جمع بین تومان و ریال انجام نمی‌شود و اقدام وابسته مسدود است.');
    if (s === 'dupe' && (view === 'rev' || view === 'rec')) out += b('incomplete', '<b>پرداخت تکراری احتمالی:</b> دو مدرک به یک مرجع بانکی اشاره می‌کنند. هیچ اعتبار خودکار برای هر دو ساخته نمی‌شود؛ ابتدا بررسی تکراری لازم است.');
    if (s === 'runpartial' && view === 'run') out += b('incomplete', '<b>اجرای ناقص:</b> دست‌کم یک اجرا بخشی از آیتم‌ها را ثبت کرده و دنباله پردازش‌نشده دارد. پیام «ok» فراخوانی محدود، موفقیت کل اجرا نیست.');
    if (s === 'unknownrun' && view === 'run') out += b('incomplete', '<b>نتیجهٔ ثبت یک اجرا نامعلوم است:</b> ممکن است ثبت شده باشد. تکرار کور مسدود است؛ ابتدا وضعیت واقعی خوانده و تطبیق شود.');
    return out;
  };
})();
