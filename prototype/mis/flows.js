/* MIS — role layer, part 3: drawers and guarded flows (lineage inspector, custody timeline, return safety, reconciliation issue,
   import / assignment / quick-delivery / planning / maintenance reviews, bulk results with per-item truth).
   Prototype only: nothing is saved or sent. Preview ≠ apply; apply always re-checks; Outcome Unknown is reconciled before any retry. */
(function () {
  'use strict';
  var X = window.MISX, C = X.C, M = X.M, h = X.h, st = X.st, D = X.D;
  var esc = h.esc, fa = h.fa, ic = h.ic, pill = h.pill, hint = h.hint, $ = h.$;
  var top = X.top, foot = X.foot, sec = X.sec, tl = X.tl, checks = X.checks;
  var valPill = function (v) { var r = X.VAL[v]; return pill(r.label, r.tone, r.icon); };
  var retPill = function (k) { var r = X.RET[k]; return pill(r.label, r.tone, r.icon); };
  var head = function (title, badges, meta) { return '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">' + title + '</h2>' + (badges || '') + '</div>' + (meta ? '<div class="dr-meta">' + meta + '</div>' : '') + '</div>'; };
  var btn = function (cls, act, label, iconName, dis) { return '<button type="button" class="btn ' + cls + '" data-act="' + act + '"' + (dis ? ' disabled aria-disabled="true"' : '') + '>' + (iconName ? ic(iconName) : '') + label + '</button>'; };
  var runLine = function (r) { return X.srcCtx({ file: r.file, batch: r.batch, run: r.id }); };

  /* ---------- Operation engine: per-item truth. requested = applied + skipped + rejected + failed + unknown. ---------- */
  var seq = 100;
  var REASON = { skipped: 'تغییر پس از بررسی: همین ردیف قبلاً توسط عملیات دیگری تخصیص یافت', rejected: 'ردشده توسط سامانه: ردیف در حال حاضر معتبر نیست', failed: 'خطای ثبت؛ قابل تکرار پس از بررسی تازه', unknown: 'پاسخ ثبت نرسید؛ ممکن است ثبت شده باشد' };
  var PAT = ['ok', 'ok', 'skipped', 'ok', 'failed', 'unknown', 'rejected', 'ok'];
  X.newOp = function (o) { o.id = 'OP-' + (++seq); o.ts = 'همین الان'; st.ops.unshift(o); return o; };
  X.opOf = function (id) { return st.ops.filter(function (o) { return o.id === id; })[0]; };
  X.applyOk = function (op) {
    op.items.forEach(function (x) {
      var c = X.cs(x[0]); if (x[1] !== 'ok' || !c || x.applied) return; x.applied = true;
      if (op.kind === 'assign') { c.stage = 'assigned'; c.cust.src = op.to + ' (assigned_manager منبع)'; c.cust.cur = 'نامشخص — از فیلد منبع حدس زده نمی‌شود'; c.cust.recip = op.to + ' (در منبع)'; c.cust.next = 'MIS: آماده‌سازی استخر یا تحویل'; c.hist.unshift(['تخصیص منبع به مدیر', 'همین الان', 'اجرای ' + op.id + ' · فقط فیلد منبع']); }
      if (op.kind === 'pool') { c.pool = true; c.hist.unshift(['آماده‌سازی استخر', 'همین الان', 'اجرای ' + op.id + ' · لید زنده ساخته نشد']); }
    });
  };
  X.opState = function (op) { var c = X.counts(op); return c.unknown ? 'unknown' : c.ok === c.requested ? 'complete' : c.ok === 0 && c.requested ? 'failed' : 'partial'; };
  X.runAssign = function (flow) {
    var ids = X.keys(st.asel).filter(X.cs).sort(), isA = st.cq === 'assign';
    var items = ids.map(function (id, i) { var s = flow === 'unknown' ? 'unknown' : flow === 'retry' ? 'failed' : PAT[i % PAT.length]; return [id, s, REASON[s] || '']; });
    var op = X.newOp({ kind: isA ? 'assign' : 'pool', title: isA ? 'تخصیص منبع به ' + X.mgr(st.recip).name : 'آماده‌سازی استخر', to: isA ? 'مدیر ' + X.mgr(st.recip).name : 'استخر', requested: items.length, items: items, note: isA ? 'فقط فیلد منبع (assigned_manager) و ثبت اجرا تغییر کرد؛ تحویل فعلی و لید زنده تغییر نکرد.' : 'ردیف‌ها آماده تحویل شدند؛ لید زنده ساخته نشد.' });
    X.applyOk(op); st.asel = {}; st.flow = null; C.closeDrawer(); C.render(); C.openDrawer('result', op.id);
  };
  X.retryFailed = function (op) { op.items.forEach(function (x) { if (x[1] === 'failed') { x[1] = 'ok'; x[2] = 'تکرار پس از بررسی تازه'; } }); X.applyOk(op); };
  X.reconcileOp = function (op) { var n = 0; op.items.forEach(function (x) { if (x[1] === 'unknown') { x[1] = n++ % 2 === 0 ? 'ok' : 'failed'; x[2] = x[1] === 'ok' ? 'تطبیق: ثبت شده بود' : 'تطبیق: ثبت نشده بود؛ اکنون قابل تکرار'; } }); X.applyOk(op); op.note = (op.note || '') + ' نتیجه نامعلوم با خواندن لاگ و شناسه‌های نتیجه تطبیق شد.'; };

  D.result = function (id) {
    var op = X.opOf(id) || st.ops[0]; if (!op) return top('نتیجه عملیات') + '<div class="dr-body">' + h.stateBlock('empty', 'نتیجه‌ای نیست', '') + '</div>';
    var c = X.counts(op), s = X.OPS_S(X.opState(op)), groups = [['ok', 'اعمال‌شده'], ['skipped', 'ارسال‌نشده'], ['rejected', 'ردشده'], ['failed', 'ناموفق'], ['unknown', 'نامعلوم']];
    var gl = groups.filter(function (g) { return c[g[0]]; }).map(function (g) {
      var list = op.items.filter(function (x) { return x[1] === g[0]; }), o = X.OUT[g[0]];
      return '<details class="sec"' + (g[0] === 'ok' ? '' : ' open') + '><summary><h3>' + pill(o.label, o.tone, o.icon) + '</h3><span class="muted" style="margin-right:8px;font-size:var(--t-meta)">' + fa(list.length) + '</span><span class="chev">' + ic('chev') + '</span></summary><ul class="ir-list">' + list.map(function (x) { return '<li class="ir"><span class="mono">' + esc(x[0]) + '</span><span class="ir-why">' + esc(x[2] || '—') + '</span></li>'; }).join('') + '</ul></details>';
    }).join('');
    var acts = '';
    if (c.unknown) acts += btn('btn-primary btn-lg', 'reconcile-op:' + op.id, 'تطبیق نتیجه نامعلوم (خواندن لاگ)', 'refresh');
    if (c.failed && !c.unknown) acts += btn('btn-primary btn-lg', 'retry-op:' + op.id, 'تکرار فقط ناموفق‌های معلوم', 'refresh');
    return top('نتیجه عملیات') + head(esc(op.title), pill(s[0], s[1], s[2]), '<span class="mono">' + op.id + '</span><span>' + esc(op.ts) + '</span>') + '<div class="dr-body">' +
      '<section class="sec primary"><p class="result-line">' + ic(c.ok === c.requested ? 'checkCircle' : 'alert') + (c.ok === c.requested ? 'همه ' + fa(c.requested) + ' مورد اعمال شد.' : fa(c.ok) + ' از ' + fa(c.requested) + ' مورد اعمال شد؛ بقیه به‌تفکیک زیر آمده‌اند.') + '</p>' + X.outcomeStrip(op) + (c.unknown ? '<div class="note warn inset">' + ic('question') + '<span><b>نتیجه نامعلوم است.</b> این موارد ممکن است ثبت شده باشند؛ ابتدا تطبیق کنید و تکرار کور انجام ندهید.</span></div>' : '') + (op.note ? '<p class="ind-note">' + ic('info') + ' ' + esc(op.note) + '</p>' : '') + '</section>' + gl + '</div>' + foot(acts, null, 'شماره ارجاع ' + op.id);
  };
  X.OPS_S = function (k) { return { complete: ['کامل', 'green', 'checkCircle'], partial: ['ناقص (موفقیت جزئی)', 'orange', 'split'], failed: ['ناموفق', 'red', 'xCircle'], unknown: ['نتیجه نامعلوم', 'amber', 'question'] }[k]; };

  /* ---------- Import / run / file / batch drawers ---------- */
  D.run = function (id) {
    var r = X.run(id), isImp = r.type === 'import', s = X.RUNS[r.state], log = M.log.filter(function (l) { return l.run === r.id; })[0], acts = '';
    if (isImp && r.unknown) acts += btn('btn-primary btn-lg', 'reconcile-run:' + r.id, 'تطبیق نتیجه نامعلوم (خواندن لاگ)', 'refresh');
    if (isImp && r.failed) acts += btn('btn-lg', 'retry-run:' + r.id, 'تکرار فقط ' + fa(r.failed) + ' ناموفق معلوم', 'refresh');
    var body = isImp ? X.qualityBody(r) + (r.unknown ? sec('نتیجه نامعلوم', '', '<div class="note warn">' + ic('question') + '<span><b>' + fa(r.unknown) + ' ردیف نتیجه نامعلوم دارند.</b> ورود مجدد کل فایل ممنوع است: ابتدا لاگ و شناسه‌های نتیجه خوانده شود؛ سپس فقط ردیف‌های ناموفق معلوم با بررسی تازه تکرار می‌شوند.</div>') : '') : sec('خلاصه اجرا', '', '<p class="ind-note">' + ic('info') + ' ' + esc(r.note) + '. این اجرا ورود فایل نیست و شمارنده‌های ورود برایش معنی ندارد (صفر نیست).</p>');
    return top('اجرا ' + r.id) + head('<span class="mono">' + r.id + '</span>', pill(X.RUNTYPE[r.type], isImp ? 'blue' : 'slate', isImp ? 'upload' : 'send') + pill(s[0], s[1], s[2]), '<span>' + esc(r.at) + '</span><span>عامل: ' + esc(r.actor) + '</span>') +
      '<div class="dr-body"><section class="sec">' + runLine(r) + '</section>' + body + X.details('ردیابی و ممیزی', null, X.own7Audit(r, log), true) + '</div>' + foot(acts, null, '');
  };
  X.own7Audit = function (r, log) {
    return '<dl class="exc-dl"><div><dt>عامل</dt><dd>' + esc(r.actor) + '</dd></div><div><dt>زمان</dt><dd>' + esc(r.at) + ' (+۰۳:۳۰)</dd></div><div><dt>دسته / فایل</dt><dd class="mono">' + r.batch + ' / ' + r.file + '</dd></div><div><dt>شناسه همبستگی</dt><dd class="mono">' + esc(log ? log.corr : 'ثبت نشده') + '</dd></div><div><dt>دلیل</dt><dd>' + esc(log ? log.reason : '—') + '</dd></div></dl><p class="ind-note">' + ic('info') + ' تاریخچه کامل و تغییرناپذیر تأیید نشده است؛ فقط همین رکوردها نمایش داده می‌شوند.</p>';
  };
  D.file = function (id) {
    var f = X.file(id), b = X.batch(f.batch);
    return top('فایل ' + f.id) + head(esc(f.name), pill(f.fmt, 'slate', 'file'), '<span>' + esc(f.size) + '</span><span>' + esc(f.at) + '</span>') + '<div class="dr-body"><section class="sec">' + X.srcCtx({ file: f.id, batch: f.batch }) + '</section>' +
      sec('این فایل چه چیزی نیست', '', checks([['info', 'فایل ≠ دسته', 'دسته (در نمای قدیمی «پرونده») رکوردهای حاصل از آن را نگه می‌دارد'], ['info', 'فایل ≠ ردیف منبع', 'حذف فایل رکوردها و منشأ را حذف نمی‌کند'], ['warn', 'حذف فایل کار نگهداری است', 'از اینجا انجام نمی‌شود؛ بخش «نگهداری» بررسی اثر دارد']])) +
      sec('دسته مرتبط', '', '<div class="facets"><div class="facet"><span class="muted">دسته</span><b class="mono">' + b.id + '</b></div><div class="facet"><span class="muted">ردیف منبع</span><b>' + fa(b.rows) + '</b></div><div class="facet"><span class="muted">مصرف‌شده</span><b>' + fa(b.consumed) + '</b></div></div>') + '</div>' + foot(btn('btn-soft btn-lg', 'goto:maint', 'نگهداری و پیشرفته', 'lock'), null, '');
  };
  D.batch = function (id) {
    var b = X.batch(id), f = X.file(b.file), s = X.RUNS[b.state];
    return top('دسته ' + b.id) + head('<span class="mono">' + b.id + '</span> · ' + esc(b.name), pill(s[0], s[1], s[2]), '<span>منبع: ' + esc(b.source) + '</span>') + '<div class="dr-body"><section class="sec">' + X.srcCtx({ file: f.id, batch: b.id }) + '</section>' +
      sec('وضعیت', '', '<div class="facets"><div class="facet"><span class="muted">ردیف ذخیره‌شده</span><b>' + fa(b.rows) + '</b></div><div class="facet"><span class="muted">تخصیص منبع‌شده</span><b>' + fa(b.assigned) + '</b></div><div class="facet"><span class="muted">مصرف‌شده</span><b>' + fa(b.consumed) + '</b></div></div><p class="ind-note">' + ic('info') + ' «تخصیص منبع‌شده» فقط فیلد منبع است و تحویل فعلی را نمی‌گوید.</p>') + '</div>' + foot('', null, '');
  };

  D.importNew = function (id, keep) {
    keep = keep || C.state.drawer.keep; var step = keep.step || 0, FILES = { A: { n: 'leads-azar.xlsx', fmt: 'XLSX', size: '۱٫۸ مگابایت', c: [320, 301, 12, 7, 0, 0], note: 'نمونه سالم' }, B: { n: 'contacts-dup-heavy.csv', fmt: 'CSV', size: '۹۰۰ کیلوبایت', c: [180, 98, 41, 22, 11, 8], note: 'تکراری زیاد' }, C: { n: 'big-export.xlsx', fmt: 'XLSX', size: '۴٫۶ مگابایت', c: [620, 410, 0, 0, 10, 200], note: 'قطع ارتباط حین ورود' } };
    X.IMPFILES = FILES;
    var f = FILES[keep.file || 'A'], c = f.c, title = 'ورود فایل جدید', body;
    var stp = X.steps(['انتخاب فایل', 'تجزیه و بررسی', 'نتیجه'], step);
    if (step === 0) {
      body = '<div class="dr-body"><section class="sec">' + stp + '</section>' + sec('فایل نمونه', 'CSV یا XLSX · حداکثر ۵ مگابایت', '<div class="recips-inline" role="radiogroup" aria-label="فایل نمونه">' + Object.keys(FILES).map(function (k) { var x = FILES[k], on = (keep.file || 'A') === k; return '<label class="recip' + (on ? ' on' : '') + '"><input type="radio" name="impf" class="cbx" data-impfile="' + k + '"' + (on ? ' checked' : '') + '><span class="r-main"><b>' + esc(x.n) + '</b><span>' + x.fmt + ' · ' + x.size + ' · ' + fa(x.c[0]) + ' ردیف · ' + x.note + '</span></span></label>'; }).join('') + '</div>') +
        '<section class="sec"><div class="note warn">' + ic('alert') + '<span>گام بعد فایل را تجزیه می‌کند. <b>پیش‌نمایش ورود بدون اثر جانبی تضمین‌شده نیست</b> (ممکن است پاک‌سازی تکراری هم‌دسته و ترمیم ساختار پیش از تجزیه اجرا شود).</span></div></section></div>' + foot(btn('btn-primary btn-lg', 'imp-parse', 'تجزیه و بررسی', 'search'), null, '');
    } else if (step === 1) {
      body = '<div class="dr-body"><section class="sec">' + stp + '</section>' + sec('پیش‌نمایش طبقه‌بندی', esc(f.n), '<div class="facets q-facets">' + [['دیده‌شده', c[0]], ['معتبر (قابل ورود)', c[1]], ['تکراری', c[2]], ['نامعتبر', c[3]], ['ناموفق احتمالی', c[4]], ['نامعلوم احتمالی', c[5]]].map(function (x) { return '<div class="facet"><span class="muted">' + x[0] + '</span><b class="mt-v">' + fa(x[1]) + '</b></div>'; }).join('') + '</div>') +
        sec('قواعد', '', checks([['ok', 'هیچ ردیفی بر پایه شماره ادغام نمی‌شود', 'ردیف تکراری ذخیره و طبقه‌بندی می‌شود'], ['warn', 'نتیجه پیش‌نمایش ضمانت نتیجه نهایی نیست', 'ثبت واقعی هر ردیف جدا نتیجه می‌گیرد'], ['info', 'اگر ارتباط قطع شود، ردیف‌های بدون پاسخ «نامعلوم» می‌مانند', 'ورود مجدد کل فایل انجام نمی‌شود']])) +
        '<section class="sec"><label class="confirm"><input type="checkbox" class="cbx" data-impok' + (keep.ok ? ' checked' : '') + '><span>می‌دانم این ورود داده ثبت می‌کند، ردیف تکراری ادغام نمی‌شود و پاک‌سازی هم‌دسته ممکن است اجرا شود.</span></label></section></div>' + foot(btn('btn-primary btn-lg', 'imp-commit', 'ورود فایل', 'upload', !keep.ok), btn('btn-lg btn-ghost', 'imp-back', 'بازگشت'), 'ثبت واقعی در نمونه انجام نمی‌شود');
    } else {
      var r = X.run(keep.run), imp = { requested: r.seen, c: { requested: r.seen, eligible: r.seen - r.dup - r.invalid, ok: r.imported, skipped: r.dup, rejected: r.invalid, failed: r.failed, unknown: r.unknown } }, s = X.RUNS[r.state];
      body = '<div class="dr-body"><section class="sec">' + stp + '</section><section class="sec primary"><p class="result-line">' + ic(r.state === 'complete' ? 'checkCircle' : 'alert') + pill(s[0], s[1], s[2]) + '&nbsp;' + (r.state === 'complete' ? 'همه ردیف‌ها طبقه‌بندی شدند؛ جمع‌ها می‌خوانند.' : r.state === 'unknown' ? 'ارتباط قطع شد؛ نتیجه برخی ردیف‌ها نامعلوم است.' : 'ورود ناقص تمام شد؛ نتیجه هر ردیف جدا آمده است.') + '</p>' + X.outcomeStrip(imp) + '</section>' + sec('ردیابی', '', runLine(r)) + '</div>' + foot(btn('btn-primary btn-lg', 'open-run:' + r.id, 'مشاهده اجرا ' + r.id, 'arrowL'), null, '');
    }
    return top(title) + head(title, pill('عملیات عادی', 'teal', 'eye'), '<span>نمونه: ثبت واقعی انجام نمی‌شود</span>') + body;
  };
  X.commitImport = function (keep) {
    var f = X.IMPFILES[keep.file || 'A'], c = f.c, rid = 'R-' + (9923 + M.runs.length - 5), fid = 'F-' + (2042 + M.files.length - 4), bid = 'B-' + (1188 + M.batches.length - 4);
    M.files.unshift({ id: fid, name: f.n, fmt: f.fmt, size: f.size, at: 'همین الان', actor: M.user.name, batch: bid });
    var state = c[5] ? 'unknown' : (c[4] ? 'partial' : 'complete');
    M.batches.push({ id: bid, name: f.n, file: fid, source: 'بارگذاری نمونه', rows: c[1], state: state, assigned: 0, consumed: 0 });
    var run = { id: rid, type: 'import', batch: bid, file: fid, actor: M.user.name, at: 'همین الان', state: state, seen: c[0], imported: c[1], dup: c[2], invalid: c[3], failed: c[4], unknown: c[5], dupClasses: c[2] ? [['infile', 'تکراری در همان فایل', Math.ceil(c[2] / 2)], ['othermis', 'ردیف MIS در دسته دیگر', Math.floor(c[2] / 2)]] : [], warns: c[5] ? ['ارتباط با سرور حین ورود قطع شد؛ نتیجه آخرین ردیف‌ها ثبت نشد'] : [] };
    M.runs.unshift(run); keep.run = rid; keep.step = 2; return run;
  };
  X.reconcileRun = function (r) { var u = r.unknown, ok = Math.ceil(u * 0.75); r.imported += ok; r.failed += u - ok; r.unknown = 0; r.state = r.failed ? 'partial' : 'complete'; r.warns = r.warns.concat(['تطبیق با لاگ: ' + fa(ok) + ' ردیف ثبت شده بود، ' + fa(u - ok) + ' ردیف ثبت نشده بود']); };
  X.retryRun = function (r) { r.imported += r.failed; r.failed = 0; r.state = r.unknown ? 'partial' : 'complete'; };

  /* ---------- Lineage Inspector (Source Case) ---------- */
  D.case = function (id) {
    var c = X.cs(id), b = X.batch(c.batch), r = M.runs.filter(function (x) { return x.batch === c.batch && x.type === 'import'; })[0];
    var phoneBlock = '<div class="phone-box">' + ic('phone') + '<div><b class="mono">' + c.phone + '</b><span class="muted">فقط برای جستجو و کشف</span></div><div class="ph-note"><b>' + (c.phoneMatches ? fa(c.phoneMatches) + ' رکورد دیگر همین شماره را دارد' : 'رکورد هم‌شماره‌ای کشف نشد') + '</b><span>هم‌شماره بودن هویت، پرونده یا مدرک اتصال مالی نیست و ادغامی انجام نمی‌دهد.</span></div></div>';
    return top('بازرسی اتصال ردیف') + head('ردیف MIS ' + fa(c.id), valPill(c.val) + X.lin(c.lin), '<span>دسته ' + c.batch + '</span><span>ردیف فایل ' + fa(c.row) + '</span>') + '<div class="dr-body"><section class="sec">' + X.srcCtx({ file: b.file, batch: c.batch, run: r ? r.id : '', row: c.id }) + '</section>' +
      sec('سلسله هویت و اتصال', X.basis('snap'), X.ladder(c) + '<p class="ind-note">' + ic('info') + ' «پرونده منطقی (CaseRef)» تعریف نشده است: حل‌کننده یکپارچه فعال نیست (MIS-G02). شناسه‌های بومی بدون بازنویسی نمایش داده می‌شوند و شماره یکسان دو شناسه بومی را یکی نمی‌کند.</p>', 'primary') +
      sec('شماره موبایل (کشف)', '', phoneBlock) +
      (c.val === 'valid' || c.stage !== 'none' ? sec('وابستگی‌های مالی (اتحاد همه حوزه‌ها)', X.cov(X.finBlockers(c).length ? 'partial' : 'ok', X.finBlockers(c).length ? 'وابستگی/نامعلوم دارد' : 'یافت نشد'), X.finTable(c) + '<p class="ind-note">' + ic('alert') + ' نبود فاکتور در یک نما، اثبات «بدون مصرف» نیست. نامعلوم مانند مسدود رفتار می‌شود (fail-closed).</p>') : '') +
      sec('مسئولیت — هفت مفهوم جدا', '', X.own7(c.cust)) + X.details('سابقه (خط زمانی خلاصه)', c.hist.length, tl(c.hist), false) + '</div>' +
      foot(btn('btn-soft btn-lg', 'open-tl:' + c.id, 'خط زمانی کامل تحویل', 'history'), null, 'بازرسی خواندنی است؛ هیچ ادغام یا اتصال مجددی ثبت نمی‌شود');
  };

  /* ---------- Custody timeline ---------- */
  D.tl = function (id) {
    var c = X.cs(id);
    return top('خط زمانی مسئولیت') + head('ردیف MIS ' + fa(c.id), X.lin(c.lin), '<span>' + esc(X.STAGE[c.stage]) + '</span>') + '<div class="dr-body">' + sec('اکنون', '', X.own7(c.cust), 'primary') +
      sec('رویدادها (جدیدترین بالا)', '', c.hist.map(function (e, i) { return '<div class="ev"><span class="ev-dot" aria-hidden="true"></span><div><b>' + esc(e[0]) + '</b><span>' + esc(e[1]) + '</span><span class="muted">' + esc(e[2]) + '</span></div></div>'; }).join('')) +
      sec('چگونه خوانده شود', '', checks([['info', 'مسئول فعلی ≠ فیلد منبع', 'تخصیص منبع فقط فیلد منبع است؛ تحویل با رویداد جدا ثبت می‌شود'], ['info', 'ساختار امروز سابقه را بازنویسی نمی‌کند', 'تحویل مستقیم به فروشنده، حتی اگر امروز مدیر و سرپرست بالادست دارد، «دریافت مدیر» نمی‌شود'], ['info', 'مالک اعتبار با تحویل عوض نمی‌شود', 'طبق قوانین مالی فعلی تعیین می‌شود']])) + '</div>' + foot(btn('btn-soft btn-lg', 'open-case:' + c.id, 'بازرسی اتصال', 'link'), null, '');
  };

  /* ---------- Assignment / pool review ---------- */
  D.assignReview = function () {
    var ids = X.keys(st.asel).filter(X.cs).sort(), isA = st.cq === 'assign', m = isA ? X.mgr(st.recip) : null, conflict = st.flow === 'conflict' && ids.indexOf('884217') > -1;
    var rows = ids.map(function (id) {
      var c = X.cs(id), bad = conflict && id === '884217';
      return '<li class="mini"><div class="grow"><span class="mono">' + esc(id) + '</span><div class="muted" style="font-size:var(--t-meta)">دسته ' + c.batch + ' · ' + esc(X.STAGE[c.stage]) + '</div></div>' + (bad ? pill('تغییر کرده', 'orange', 'swap') : pill('واجد شرایط (پیش‌نمایش)', 'teal', 'checkCircle')) + '</li>';
    }).join('');
    var keep = C.state.drawer.keep;
    return top(isA ? 'بررسی اثر تخصیص منبع' : 'بررسی اثر آماده‌سازی استخر') + head((isA ? 'تخصیص ' : 'آماده‌سازی ') + fa(ids.length) + ' ردیف', pill('پیش‌نمایش', 'blue', 'eye'), '<span>' + (isA ? 'به: مدیر ' + esc(m.name) : 'به: استخر') + '</span>') +
      '<div class="dr-body">' + sec('چه تغییر می‌کند و چه نمی‌کند', '', checks(isA ? [['ok', 'فیلد منبع assigned_manager و ثبت اجرا', 'برای ' + fa(ids.length) + ' ردیف معتبر بدون تخصیص'], ['info', 'تحویل فعلی و سابقه توزیع تغییر نمی‌کند', 'تخصیص منبع ≠ مسئول فعلی'], ['info', 'لید زنده ساخته نمی‌شود', 'ایجاد لید در «برنامه‌ریزی» با تأیید جدا است']] : [['ok', 'ردیف‌ها به استخر آماده تحویل می‌روند', fa(ids.length) + ' ردیف'], ['info', 'لید زنده ساخته نمی‌شود', 'گام جدا و مشروط'], ['info', 'هیچ‌کس مالک جدید نمی‌شود', 'گیرنده انتخاب نمی‌شود']])) +
      sec('ردیف‌ها', conflict ? pill('۱ ردیف پس از بررسی تغییر کرد', 'orange', 'swap') : '', '<ul class="mini-list">' + rows + '</ul>' + (conflict ? '<div class="note conflict inset">' + ic('swap') + '<span><b>ردیف ۸۸۴۲۱۷ پس از بررسی توسط عملیات دیگری تخصیص یافت.</b> اطلاعات قبلی قدیمی است؛ بازخوانی کنید. بازنویسی یا گیرنده خودکار انجام نمی‌شود.</span></div>' : '')) +
      '<section class="sec"><div class="note warn">' + ic('alert') + '<span><b>هنگام ثبت دوباره بررسی می‌شود:</b> معتبر بودن، بدون تخصیص بودن، ' + (isA ? 'فعال بودن مدیر و ' : '') + 'تغییر ردیف. نتیجه هر ردیف جدا می‌آید و «همه موفق» فرض نمی‌شود.</span></div>' +
      '<label class="confirm"><input type="checkbox" class="cbx" data-confirm' + (keep.confirm ? ' checked' : '') + '><span>بررسی اثر را خواندم و می‌دانم که نتیجه ممکن است ناقص باشد.</span></label></section></div>' +
      foot(conflict ? btn('btn-primary btn-lg', 'refresh-review', 'بازخوانی بررسی', 'refresh') : btn('btn-primary btn-lg', 'commit-assign', 'ثبت ' + (isA ? 'تخصیص منبع' : 'آماده‌سازی'), 'send', !keep.confirm), null, 'پیش‌نمایش رزرو یا مجوز نهایی نیست');
  };

  /* ---------- Return safety ---------- */
  D.ret = function (id) {
    var c = X.cs(id), e = X.RET[c.ret], body, resp = null, fl = c.fin;
    var transfer = c.hist.length && c.cust.event !== 'ثبت نشده' ? ['ok', 'مدرک انتقال: ' + c.cust.event, c.hist[0][0] + ' · ' + c.hist[0][1]] : ['q', 'مدرک انتقال کامل نیست', 'عامل رویداد نامشخص (UNKNOWN)'];
    var linkRow = c.lin === 'verified' ? ['ok', 'اتصال منبع: تأییدشده', 'شناسه‌های بومی با مدرک پایدار'] : c.lin === 'conflict' ? ['no', 'اتصال منبع: متعارض', 'منابع اختلاف دارند'] : ['q', 'اتصال منبع: ' + (c.lin === 'partial' ? 'جزئی' : 'نامشخص'), 'با شماره موبایل اثبات نمی‌شود'];
    var dep = X.finBlockers(c).length ? ['no', 'وابستگی مالی: ' + fa(X.finBlockers(c).length) + ' حوزه فعال/لغوشده/نامعلوم', 'جدول اتحاد پایین'] : ['ok', 'وابستگی مالی: هیچ‌کدام در چهار حوزه یافت نشد', 'پوشش اثبات‌شده برای هر چهار حوزه'];
    if (c.ret === 'ok') { body = checks([transfer, linkRow, dep, ['warn', 'محافظ فعلی هنوز همه ارتباط‌ها را اجرا نمی‌کند (F01)', 'پس این فقط پیش‌نمایش است'], ['info', 'مالک اولیه، سابقه تماس و مالک اعتبار دست‌نخورده می‌ماند', '']]); }
    else if (c.ret === 'blocked') { body = checks([transfer, linkRow, dep, ['no', 'دلیل مسدودی: ' + c.retWhy, ''], ['info', 'برگشت عادی انجام نمی‌شود', 'پاک‌کردن اتصال یا بازنشانی ردیف، حل تعارض حساب نمی‌شود']]); resp = { who: 'مالی / مهندسی + مالک حوزه', text: c.retWhy.indexOf('لغو') > -1 ? 'تصمیم آزادسازی (OPD-03) با مالک حوزه' : 'اثبات و تطبیق ارتباط مالی' }; }
    else if (c.ret === 'conflict') { body = checks([['warn', 'وضعیت ردیف پس از پیش‌نمایش تغییر کرده است', c.retWhy], ['info', 'پیش از هر اقدام بازخوانی کنید', 'اطلاعات نمایش‌داده‌شده ممکن است قدیمی باشد؛ بازنویسی انجام نمی‌شود']]); }
    else { body = checks([transfer, linkRow, dep, ['q', 'نامعلوم · نیاز به تطبیق: ' + (c.retWhy || ''), 'تا تطبیق، برگشت انجام نمی‌شود (fail-closed)'], ['info', 'حدس با شماره موبایل انجام نمی‌شود', '']]); resp = { who: 'MIS + مالک حوزه', text: 'تطبیق ارتباط‌ها پیش از هر برگشت' }; }
    var primary = c.ret === 'conflict' ? btn('btn-primary btn-lg', 'refresh-ret:' + c.id, 'بازخوانی ردیف', 'refresh') : c.ret === 'ok' ? btn('btn-lg', 'x', 'ثبت برگشت غیرفعال — مشروط به F01', 'lock', true) : '';
    return top('بررسی ایمنی برگشت') + head('ردیف MIS ' + fa(c.id), retPill(c.ret) + X.lin(c.lin), '<span>نزد ' + esc(c.cust.cur) + '</span>') + '<div class="dr-body">' + sec('مسئولیت — هفت مفهوم جدا', '', X.own7(c.cust)) +
      sec('آیا این ردیف قابل برگشت است؟', '', body + (resp ? '<div class="resp-line"><span>مسئول اقدام بعدی:</span><b>' + esc(resp.who) + '</b><span class="muted">' + esc(resp.text) + '</span></div>' : ''), c.ret === 'ok' ? '' : 'primary') +
      sec('اتحاد وابستگی‌های مالی', X.zone('adv', 'برگشت'), X.finTable(c)) +
      '<section class="sec"><div class="note ' + (c.ret === 'ok' ? 'warn' : 'info') + '">' + ic('lock') + '<span><b>پیش‌نمایش است، نه مجوز اجرا.</b> در زمان ثبت، محدوده، وضعیت، مسئول فعلی و همه ارتباط‌های مالی دوباره بررسی می‌شود؛ هر ابهام، مسدود می‌ماند. لغو یا رد فاکتور به‌تنهایی آزادسازی نیست و برای موارد مبهم هیچ گزینه «بازنشانی» ارائه نمی‌شود.</span></div></section>' + X.details('سابقه', c.hist.length, tl(c.hist)) + '</div>' + foot(primary, btn('btn-soft btn-lg', 'open-case:' + c.id, 'بازرسی اتصال', 'link'), '');
  };
  D.returnReview = function () {
    var ids = X.keys(st.rsel).filter(function (k) { var c = X.cs(k); return c && c.ret === 'ok'; });
    return top('پیش‌نمایش برگشت گروهی') + head('برگشت ' + fa(ids.length) + ' ردیف', pill('مشروط · غیرفعال', 'amber', 'lock'), '<span>پیش‌نمایش</span>') + '<div class="dr-body">' + sec('ردیف‌ها (پیش‌نمایش)', '', '<ul class="mini-list">' + ids.map(function (i) { var c = X.cs(i); return '<li class="mini"><div class="grow"><span class="mono">' + i + '</span><div class="muted" style="font-size:var(--t-meta)">نزد ' + esc(c.cust.cur) + '</div></div>' + pill('واجد شرایط پیش‌نمایش', 'teal', 'checkCircle') + '</li>'; }).join('') + '</ul>', 'primary') +
      '<section class="sec"><div class="note warn">' + ic('lock') + '<span>ثبت برگشت گروهی تا تکمیل حفاظت F01 (MIS-G01) فعال نیست. وقتی فعال شود: نتیجه به‌تفکیک ردیف (اعمال‌شده / ارسال‌نشده / ردشده / ناموفق / نامعلوم) نمایش داده می‌شود و ابهام «موفق» حساب نمی‌شود.</span></div></section></div>' + foot(btn('btn-lg', 'x', 'ثبت برگشت — مشروط به F01', 'lock', true), null, '');
  };

  /* ---------- Reconciliation issue ---------- */
  D.issue = function (id) {
    var i = X.issue(id), cl = M.classes[i.cls], s = X.ISSTATE[i.state], e = i.entity;
    var ent = e[0] === 'case' ? btn('btn-soft', 'open-case:' + e[1], 'بازرسی اتصال ردیف', 'link') : e[0] === 'run' ? btn('btn-soft', 'open-run:' + e[1], 'مشاهده اجرا', 'activity') : e[0] === 'report' ? btn('btn-soft', 'open-report-from-issue:' + e[1], 'مشاهده گزارش', 'chart') : btn('btn-soft', 'goto-diag:health', 'مشاهده سلامت', 'activity');
    var ret = i.id === 'I1' ? btn('btn-soft', 'open-ret:884204', 'بررسی ایمنی برگشت', 'lock') : '';
    return top('مسئله تطبیق') + head(esc(i.title), pill(cl.label, i.sev, cl.icon) + pill(s[0], s[1], s[2]) + X.lin(i.conf), '<span class="mono">' + i.id + '</span><span>' + esc(e[2]) + '</span>') + '<div class="dr-body">' +
      sec('موجودیت متأثر', '', '<div class="facets"><div class="facet"><span class="muted">موجودیت</span><b class="mono">' + esc(e[1]) + '</b></div><div class="facet"><span class="muted">مالک / حوزه</span><b>' + esc(i.owner) + '</b></div><div class="facet"><span class="muted">مسئول گام بعد</span><b>' + esc(i.next) + '</b></div></div>' + '<div class="resp-line">' + ent + ret + '</div>', 'primary') +
      sec('شواهد', X.lin(i.conf), checks(i.evidence.map(function (x) { return ['info', x, '']; }))) +
      sec('اقدام مجاز MIS', '', checks(i.allowed.map(function (x) { return ['ok', x, '']; }))) + sec('اقدام غیرمجاز در این پنل', '', checks(i.notAllowed.map(function (x) { return ['no', x, '']; }))) +
      sec('مدرک حل', '', checks(i.proof.map(function (x) { return ['q', x, '']; }))) +
      '<section class="sec"><div class="note info">' + ic('info') + '<span>تشخیص مسئله به MIS اختیار اصلاح همه حوزه‌ها را نمی‌دهد. «حل‌شده» یعنی مدرک بالا فراهم شده، نه تغییر دلخواه داده. دستور ارجاع/تشدید رسمی تعریف نشده است (MIS-G12).</span></div></section></div>' + foot('', null, 'تشخیص ≠ مجوز نوشتن');
  };
  D.reconReview = function () {
    var ids = X.keys(st.isel), list = ids.map(X.issue).filter(Boolean);
    return top('پیش‌نمایش تطبیق خواندنی گروهی') + head(fa(list.length) + ' مسئله', pill('فقط خواندنی · مشروط', 'amber', 'eye'), '<span>هیچ داده‌ای تغییر نمی‌کند</span>') + '<div class="dr-body">' + sec('مسئله‌ها', '', '<ul class="mini-list">' + list.map(function (i) { var c = M.classes[i.cls]; return '<li class="mini"><div class="grow"><b>' + esc(i.title) + '</b><div class="muted" style="font-size:var(--t-meta)">' + esc(c.label) + ' · ' + esc(i.owner) + '</div></div>' + X.lin(i.conf) + '</li>'; }).join('') + '</ul>', 'primary') +
      '<section class="sec"><div class="note warn">' + ic('alert') + '<span>«اجرا» فقط مقایسه منابع را می‌خواند و برای هر مسئله می‌گوید مقایسه شد، به مالک حوزه واگذار می‌شود یا نامعلوم ماند. هیچ اصلاح، ادغام یا اتصال مجدد انجام نمی‌شود؛ اصلاح گروهی «فقط نگهداری» و نیازمند اعتبارسنجی است.</span></div></section></div>' +
      foot(btn('btn-primary btn-lg', 'run-recon', 'اجرای تطبیق خواندنی', 'search', !list.length), null, '');
  };
  X.runRecon = function () {
    var items = X.keys(st.isel).map(X.issue).filter(Boolean).map(function (i) { var s = i.conf === 'unknown' ? 'unknown' : i.state === 'awaiting' ? 'skipped' : 'ok'; return [i.id, s, s === 'ok' ? 'مقایسه خواندنی انجام شد؛ تغییری ثبت نشد' : s === 'skipped' ? 'به مالک حوزه واگذار شد: ' + i.owner : 'مدرک کافی نیست؛ نامعلوم ماند']; });
    var op = X.newOp({ kind: 'recon', title: 'تطبیق خواندنی گروهی', requested: items.length, items: items, note: 'اعمال‌شده یعنی مقایسه خواندنی انجام شد؛ هیچ داده‌ای تغییر نکرد.' });
    st.isel = {}; C.closeDrawer(); C.render(); C.openDrawer('result', op.id);
  };

  /* ---------- Planning ---------- */
  var stageIdx = function (p) { return X.PLANST.map(function (s) { return s[0]; }).indexOf(p.stage); };
  D.plan = function (id) {
    var p = X.plan(id), idx = stageIdx(p), m = X.mgr(p.mgr), acts = '';
    var texts = { draft: 'دستورالعمل ثبت شد', preview: 'پیش‌نمایش بررسی شد (خالص‌خواندنی تضمین نمی‌شود)', saved: 'برنامه ذخیره شد؛ چیزی تحویل نشد', approved: (p.approvedBy || '') + ' · ' + (p.approvedAt || '') + ' · فقط وضعیت برنامه', materialized: 'لیدها ساخته شد (نتیجه به‌تفکیک)' };
    var chain = X.chain(X.PLANST.map(function (s, i) { return [s[1], i <= idx ? 'done' : i === idx + 1 ? 'cur' : 'skip', i <= idx ? texts[s[0]] : i === idx + 1 ? 'مرحله بعد — هنوز انجام نشده' : 'هنوز انجام نشده']; }));
    if (p.stage === 'draft') acts = btn('btn-primary btn-lg', 'plan-save:' + p.id, 'پیش‌نمایش و ذخیره برنامه (نمایشی)', 'file');
    else if (p.stage === 'saved') acts = btn('btn-primary btn-lg', 'plan-approve:' + p.id, 'تأیید برنامه…', 'checkCircle');
    else if (p.stage === 'approved') acts = btn('btn-primary btn-lg', 'plan-mat:' + p.id, 'ایجاد لید زنده…', 'send');
    else acts = btn('btn-soft btn-lg', 'open-plan-result:' + p.id, 'نتیجه ایجاد لید', 'inbox');
    var cons = p.stage === 'approved' ? '<div class="note warn">' + ic('hourglass') + '<span><b>برنامه تأیید شده است، اما هنوز هیچ لیدی ساخته نشده.</b> «۰ لید ساخته‌شده» شکست نیست؛ مرحله ایجاد لید جدا و هنوز انجام نشده است.</span></div>' : p.stage === 'materialized' ? '<div class="note warn">' + ic('split') + '<span><b>ایجاد لید ناقص بود:</b> ' + fa(18) + ' ساخته شد، ' + fa(4) + ' ارسال نشد، ' + fa(2) + ' ناموفق، ' + fa(1) + ' نامعلوم از ' + fa(25) + ' مورد.</span></div>' : p.stage === 'draft' ? '<div class="note info">' + ic('info') + '<span>پیش‌نویس فقط دستورالعمل است؛ هیچ ردیفی تحویل یا هیچ لیدی ساخته نمی‌شود.</span></div>' : '<div class="note info">' + ic('info') + '<span>ذخیره برنامه تحویل یا لید زنده نمی‌سازد.</span></div>';
    return top('برنامه ' + p.id) + head(esc(p.name), pill(X.PLANST[idx][1], 'blue', 'file'), '<span>دسته ' + p.batch + '</span><span>مدیر: ' + esc(m.name) + '</span>') + '<div class="dr-body">' + sec('مراحل — هر مرحله جدا', '', chain, 'primary') + '<section class="sec">' + cons + '</section>' +
      sec('ورودی‌ها', '', '<div class="facets"><div class="facet"><span class="muted">قانون</span><b>' + (p.rule === 'equal' ? 'مساوی' : 'دستی') + '</b></div><div class="facet"><span class="muted">کاندید فروشنده</span><b>' + fa(p.cand) + '</b></div><div class="facet"><span class="muted">فهرست کاندیدها</span><b>نیازمند اعتبارسنجی</b></div></div>' + (p.note ? '<p class="ind-note">' + ic('alert') + esc(p.note) + '</p>' : '')) + '</div>' + foot(acts, null, 'بدون پیش‌بینی، سهمیه یا نمره ظرفیت');
  };
  D.planApprove = function (id) {
    var p = X.plan(id), keep = C.state.drawer.keep;
    return top('تأیید برنامه') + head('تأیید ' + p.id, pill('فقط وضعیت برنامه', 'amber', 'file'), '<span>' + esc(p.name) + '</span>') + '<div class="dr-body">' + sec('پیامد دقیق', '', checks([['ok', 'وضعیت برنامه «تأییدشده» می‌شود', 'با ثبت دلیل و شناسه ارجاع'], ['no', 'هیچ لیدی ساخته نمی‌شود', 'ایجاد لید گام جدا با تأیید جدا است'], ['no', 'هیچ ردیفی تحویل نمی‌شود', ''], ['info', 'موفقیت ثبت تأیید تحت خطای پایگاه داده اثبات نشده', 'نتیجه پس از ثبت بازخوانی می‌شود']]), 'primary') +
      '<section class="sec"><label class="lbl" for="ap-r">دلیل تأیید (الزامی)</label><textarea class="input" id="ap-r" rows="3" data-apreason placeholder="دلیل تأیید را بنویسید">' + esc(keep.reason || '') + '</textarea></section></div>' + foot(btn('btn-primary btn-lg', 'commit-approve:' + p.id, 'تأیید برنامه', 'checkCircle', !(keep.reason && keep.reason.trim().length > 2)), null, 'تأیید ≠ ایجاد لید');
  };
  D.planMat = function (id) {
    var p = X.plan(id), keep = C.state.drawer.keep, okTxt = (keep.apply || '').trim().toUpperCase() === 'APPLY';
    return top('ایجاد لید زنده') + head('ایجاد لید از ' + p.id, pill('پیشرفته · مشروط', 'amber', 'alert'), '<span>برنامه تأییدشده</span>') + '<div class="dr-body">' + sec('بررسی شرایط', '', checks([['ok', 'برنامه تأیید شده است', p.approvedBy + ' · ' + p.approvedAt], ['q', 'رقابت هم‌زمان و تکرار لیدهای موجود', 'بررسی تکرار وابسته به منبع است؛ رفتار کامل در رقابت تأیید نشده'], ['info', 'ایجاد لید زنده مسیر ویژه است', 'بدون پیامک؛ لید موجود ویرایش یا حذف نمی‌شود'], ['warn', 'نتیجه به‌تفکیک مورد می‌آید', 'موفقیت اتمیک کلی ادعا نمی‌شود']]), 'primary') +
      '<section class="sec"><label class="lbl" for="ap-w">برای تأیید، کلمه <span class="mono">APPLY</span> را بنویسید</label><input class="input mono" id="ap-w" type="text" dir="ltr" autocomplete="off" data-applyword value="' + esc(keep.apply || '') + '"></section></div>' + foot(btn('btn-primary btn-lg', 'commit-mat:' + p.id, 'ایجاد لید زنده', 'send', !okTxt), null, 'بازبینی نهایی هنگام ثبت');
  };
  X.planResult = function (p) {
    var oid = X.plansOps && X.plansOps[p.id], op = oid && X.opOf(oid);
    if (op) return op;
    op = X.newOp({ kind: 'plan', title: 'ایجاد لید زنده از ' + p.id, requested: p.items.length, items: p.items.map(function (x) { return x.slice(0, 3); }), note: 'برنامه تأیید شده بود؛ ایجاد لید گام جدا با نتیجه به‌تفکیک مورد است.' });
    X.plansOps = X.plansOps || {}; X.plansOps[p.id] = op.id; return op;
  };
  X.materialize = function (p) {
    var items = p.items || (M.plans.filter(function (x) { return x.id === 'P-29'; })[0].items);
    var op = X.newOp({ kind: 'plan', title: 'ایجاد لید زنده از ' + p.id, requested: items.length, items: items.map(function (x) { return x.slice(0, 3); }), note: 'برنامه تأیید شده بود؛ ایجاد لید گام جدا با نتیجه به‌تفکیک مورد است.' });
    X.plansOps = X.plansOps || {}; X.plansOps[p.id] = op.id; p.stage = 'materialized'; p.items = items; return op;
  };

  /* ---------- Quick delivery review ---------- */
  D.quickReview = function () {
    var b = M.quickBatches.filter(function (x) { return x.id === st.qb; })[0], pos = M.positions.filter(function (p) { return p.id === st.qp; })[0], keep = C.state.drawer.keep, okTxt = (keep.apply || '').trim().toUpperCase() === 'APPLY';
    var order = ['deputy', 'manager', 'senior', 'sup', 'seller'], tgt = order.indexOf(pos.id), steps = order.map(function (k, i) { var p = M.positions.filter(function (x) { return x.id === k; })[0]; return [p.label, i < 1 ? 'done' : i < tgt ? 'skip' : i === tgt ? 'cur' : 'skip', i === 0 ? 'مبدأ' : i < tgt ? 'عبور (رد‌شده)' : i === tgt ? 'گیرنده' : 'خارج از مسیر']; });
    var bypass = tgt > 1;
    return top('بررسی اثر تحویل سریع') + head('تحویل ' + fa(st.qn) + ' ردیف', pill('استثنایی · مشروط', 'amber', 'alert'), '<span>دسته ' + b.id + '</span><span>به: ' + esc(pos.label) + '</span>') + '<div class="dr-body">' + sec('مسیر', bypass ? pill('عبور از مدیر', 'amber', 'swap') : '', X.chain(steps), 'primary') +
      sec('بررسی', '', checks([['ok', 'دسته موجودی دارد', 'پیش‌نمایش: ' + fa(b.avail) + ' ردیف'], ['warn', bypass ? 'سطح(های) ردشده به‌خاطر ردیابی نمایش داده شد' : 'گیرنده در مسیر مدیر است', bypass ? 'مسئول سطح ردشده حذف نمی‌شود؛ سیاست مصوب برای دلیل/مجوز عبور نیست' : ''], ['info', 'دلیل ثبت‌شده: ' + st.qr, 'برای ردیابی؛ زنجیره تأییدی ساخته نشده'], ['warn', 'تعداد حداکثر ۵۰۰؛ نتیجه به‌تفکیک ردیف', 'مالک اعتبار و سابقه دست‌نخورده می‌ماند']])) +
      '<section class="sec"><label class="lbl" for="qk-w">برای تأیید، کلمه <span class="mono">APPLY</span> را بنویسید</label><input class="input mono" id="qk-w" type="text" dir="ltr" autocomplete="off" data-applyword value="' + esc(keep.apply || '') + '"></section></div>' + foot(btn('btn-primary btn-lg', 'commit-quick', 'ثبت تحویل سریع', 'send', !okTxt), null, 'ثبت واقعی در نمونه انجام نمی‌شود');
  };
  X.runQuick = function () {
    var n = Math.min(st.qn, 12), pos = M.positions.filter(function (p) { return p.id === st.qp; })[0], items = [];
    for (var i = 0; i < n; i++) { var s = PAT[i % PAT.length]; items.push(['Q-' + fa(500 + i), s, REASON[s] || '']); }
    var op = X.newOp({ kind: 'quick', title: 'تحویل سریع به ' + pos.label, requested: n, items: items, note: 'نمونه ۱۲ ردیف اول نمایش داده می‌شود؛ شمار واقعی تا ۵۰۰ است.' });
    C.closeDrawer(); C.render(); C.openDrawer('result', op.id);
  };

  /* ---------- Maintenance impact review ---------- */
  D.maint = function (id) {
    var m = X.maintOf(id), keep = C.state.drawer.keep, A = st.maintAuth === 'verified', sub = id === 'm3' && A, okTxt = (keep.word || '').trim() === (m.id), canGo = sub && keep.reason && keep.reason.trim().length > 2 && keep.ack && okTxt;
    var blocks = m.prot.length ? '<table class="tbl no-cursor" aria-label="وابستگی‌های محافظت‌شده"><caption class="sr">وابستگی‌های محافظت‌شده یا نامعلوم که اجرا را مسدود می‌کنند</caption><thead><tr><th>نوع</th><th>وضعیت</th><th>جزئیات</th></tr></thead><tbody>' + m.prot.map(function (p) { var tone = p[1] === 'قابل بررسی' ? 'teal' : p[1] === 'نامعلوم' ? 'amber' : 'red'; return '<tr><td>' + esc(p[0]) + '</td><td>' + pill(p[1], tone, tone === 'teal' ? 'checkCircle' : tone === 'amber' ? 'question' : 'lock') + '</td><td class="wrap">' + esc(p[2]) + '</td></tr>'; }).join('') + '</tbody></table>' : '';
    var outcome = m.block === 'protected' ? ['locked', 'مسدود: وابستگی محافظت‌شده وجود دارد. این نتیجه درست حفاظت است، نه خطا.'] : m.block === 'unknown' ? ['locked', 'مسدود: وابستگی نامعلوم است. نامعلوم یعنی «استفاده‌نشده» نیست.'] : ['locked', 'بخشی مسدود است: فقط ردپاهای بدون وابستگی قابل بررسی‌اند.'];
    var form = sub ? '<section class="sec"><label class="lbl" for="mt-r">دلیل (الزامی)</label><textarea class="input" id="mt-r" rows="2" data-mtreason>' + esc(keep.reason || '') + '</textarea>' +
      '<label class="lbl" for="mt-w" style="margin-top:12px">شناسه اقدام را برای تأیید بنویسید: <span class="mono">' + m.id + '</span></label><input class="input mono" id="mt-w" dir="ltr" data-mtword value="' + esc(keep.word || '') + '"><label class="confirm"><input type="checkbox" class="cbx" data-mtack' + (keep.ack ? ' checked' : '') + '><span>می‌دانم این عمل برگشت‌ناپذیر است و فقط ' + fa(27) + ' ردپای بدون وابستگی را اثر می‌دهد.</span></label></section>' : '';
    var auth = A ? 'تأییدشده (حالت نمایشی)' : 'تأیید نشده (MIS-G07)';
    return top('بررسی اثر نگهداری') + head(esc(m.label), X.zone('maint'), '<span>' + esc(m.target) + '</span>') + '<div class="dr-body"><section class="sec primary"><div class="note ' + 'danger' + '">' + ic('lock') + '<span>' + outcome[1] + '</span></div></section>' +
      sec('خلاصه اثر', '', checks(m.impact.map(function (x) { return ['info', x, '']; }))) + sec('موجودیت‌های متأثر', '', '<table class="tbl no-cursor" aria-label="موجودیت‌های متأثر"><caption class="sr">موجودیت‌های متأثر از اقدام</caption><thead><tr><th>نوع</th><th>شناسه</th><th class="n">تعداد</th></tr></thead><tbody>' + m.affected.map(function (a) { return '<tr><td>' + esc(a[0]) + '</td><td class="mono">' + esc(a[1]) + '</td><td class="n">' + esc(a[2]) + '</td></tr>'; }).join('') + '</tbody></table>') +
      sec('وابستگی‌های محافظت‌شده / نامعلوم', X.cov(m.block === 'partial' ? 'partial' : 'recon', m.block === 'partial' ? 'بخشی مسدود' : 'مسدود'), blocks) + form +
      sec('زمینه ممیزی', '', '<dl class="exc-dl"><div><dt>عامل</dt><dd>' + esc(M.user.name) + '</dd></div><div><dt>اختیار نگهداری مستقل</dt><dd>' + auth + '</dd></div><div><dt>شناسه همبستگی</dt><dd class="mono">پس از ثبت صادر می‌شود</dd></div></dl>') +
      '<section class="sec"><div class="note danger">' + ic('alert') + '<span><b>هشدار بازگشت:</b> ' + esc(m.rollback || '—') + ' برگشت از Git یا طراحی، تاریخچه پایگاه داده را بازنمی‌گرداند.</span></div></section></div>' +
      foot(sub ? btn('btn-danger btn-lg', 'commit-maint:' + m.id, 'اجرای فقط ' + fa(27) + ' ردپای بدون وابستگی', 'lock', !canGo) : btn('btn-lg', 'x', 'اجرا غیرفعال — ' + (A ? 'مسدود توسط وابستگی' : 'اختیار تأیید نشده'), 'lock', true), null, 'بررسی اثر هیچ داده‌ای را تغییر نمی‌دهد');
  };
  X.runMaint = function () {
    var items = []; for (var i = 0; i < 31; i++) items.push(['T-' + fa(i + 1), i < 25 ? 'ok' : i < 29 ? 'rejected' : i === 29 ? 'failed' : 'unknown', i < 25 ? '' : i < 29 ? 'رد شد: ارتباط با لید قدیمی محافظت‌شده' : i === 29 ? REASON.failed : REASON.unknown]);
    var op = X.newOp({ kind: 'maint', title: 'پاک‌سازی ردپای تکراری (فقط بدون وابستگی)', requested: 31, items: items, note: 'عمل برگشت‌ناپذیر است؛ ارجاع اولیه در لاگ باقی می‌ماند. ۴ مورد محافظت‌شده دست‌نخورده ماند.' });
    C.closeDrawer(); C.render(); C.openDrawer('result', op.id);
  };

  /* ---------- Log item + metric definitions ---------- */
  D.log = function (id) {
    var l = M.log.filter(function (x) { return x.id === id; })[0], r = X.RUNS[l.result];
    return top('جزئیات عملیات') + head(esc(l.action), pill(r[0], r[1], r[2]), '<span class="mono">' + l.id + '</span><span>' + esc(l.at) + '</span>') + '<div class="dr-body">' + sec('ردیابی', '', '<dl class="exc-dl"><div><dt>عامل</dt><dd>' + esc(l.actor) + '</dd></div><div><dt>اجرا / منبع</dt><dd class="mono">' + esc(l.run) + ' / ' + esc(l.src) + '</dd></div><div><dt>ارجاع‌ها (SourceRef / CaseRef)</dt><dd>' + esc(l.ref) + '</dd></div><div><dt>دلیل</dt><dd>' + esc(l.reason) + '</dd></div><div><dt>شناسه همبستگی</dt><dd class="mono">' + esc(l.corr) + '</dd></div></dl>', 'primary') +
      sec('نتیجه به‌تفکیک مورد', '', '<p class="ind-note">' + ic('info') + ' جزئیات هر ردیف در پنل اجرا می‌آید؛ خلاصه شمارنده‌ها نشان نمی‌دهد همه ردیف‌ها موفق بوده‌اند.</p>') + '</div>' + foot(M.runs.some(function (x) { return x.id === l.run; }) ? btn('btn-soft btn-lg', 'open-run:' + l.run, 'مشاهده اجرا', 'activity') : '', null, '');
  };
  D.metrics = function () {
    var defs = [['فاکتور صادرشده (Invoice Created)', 'شناسه یکتای فاکتور؛ گروه ایجاد در بازه', 'تعریف شده'], ['پیش‌فاکتور باز (Open Pre-invoices)', 'پیش‌فاکتور معتبر در لحظه؛ F02: بعضی نماها pre_invoice را نمی‌شمارند', 'نیازمند تطبیق'], ['پیش‌فاکتور صادرشده (Pre-invoices Issued)', 'تاریخچه صدور، جدا از «باز»', 'سابقه ناقص'], ['فروش تکمیل‌شده (Sales Completed)', 'تعریف نهایی نشده؛ مرحله تأییدشده معادل تکمیل کامل فاکتور نیست (OPD-06/07)', 'تعریف نشده'], ['کل پرونده‌ها (Total Cases)', 'اتحاد منطقی پرونده با alias اثبات‌شده؛ فعلاً وجود ندارد (F03)', 'پوشش ناقص'], ['فقط لیدهای قدیمی (Legacy Leads Only)', 'فقط منبع sn_leads؛ با «کل پرونده‌ها» یکی نیست', 'منبع محدود']];
    return top('تعریف شاخص‌ها') + head('تعریف شاخص‌ها', pill('نسخه تعریف v۱', 'slate', 'info'), '<span>برای هر شاخص: واحد شمارش، منبع، گروه، زمان، محدوده، تازگی</span>') + '<div class="dr-body">' + defs.map(function (d) { return '<details class="sec"><summary><h3>' + esc(d[0]) + '</h3><span class="chev">' + ic('chev') + '</span></summary><p class="ind-note">' + esc(d[1]) + '</p><p class="ind-note">وضعیت اعتماد: <b>' + esc(d[2]) + '</b></p></details>'; }).join('') +
      sec('سه معنای زمانی', '', checks([['info', 'وضعیت در لحظه', 'وضعیت جاری هنگام گردآوری'], ['info', 'رویداد در بازه + وضعیت جاری گردآوری (F09)', 'گروه با رویداد انتخاب می‌شود، وضعیت جاری خوانده می‌شود؛ این تاریخی نیست'], ['no', 'وضعیت تاریخی', 'فقط با مدرک کافی؛ فعلاً فعال نیست']])) + '</div>' + foot('', null, '');
  };
})();
