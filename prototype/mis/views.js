/* MIS — role layer, part 2: page views (Today, Import & Data Quality, Source Cases, Custody & Delivery, Reconciliation & Reports,
   Planning, Quick Delivery, Maintenance, Diagnostics). Presentation of Product-Spec-approved facts; undefined / unreceived / unknown values
   render as a truthful state, never as 0. Read pages never carry side effects; ordinary work, advanced work and maintenance are visually separate. */
(function () {
  'use strict';
  var X = window.MISX, C = X.C, M = X.M, h = X.h, st = X.st, V = X.V;
  var esc = h.esc, fa = h.fa, ic = h.ic, pill = h.pill, hint = h.hint;
  var na = function (t) { return '<span class="na tip" tabindex="0" data-tip="' + esc(t || 'داده در دسترس نیست') + '">—</span>'; };
  var ph = function (icon, title, id, tags, aside) { return '<div class="sec-h ph"><h2 id="' + id + '">' + ic(icon) + title + '</h2>' + (tags || '') + (aside ? '<span class="aside">' + aside + '</span>' : '') + '</div>'; };
  var runPill = function (s) { var r = X.RUNS[s]; return pill(r[0], r[1], r[2]); };
  var valPill = function (v) { var r = X.VAL[v]; return pill(r.label, r.tone, r.icon); };
  var retPill = function (k) { var r = X.RET[k]; return pill(r.label, r.tone, r.icon); };
  var open = function (kind, id, label, cls) { return '<button type="button" class="btn ' + (cls || 'btn-soft') + ' btn-sm" data-act="open-' + kind + ':' + id + '">' + label + '</button>'; };
  var seg = function (attr, cur, opts, label) { return '<div class="seg" role="group" aria-label="' + esc(label) + '">' + opts.map(function (o) { return '<button type="button" ' + attr + '="' + o[0] + '" aria-pressed="' + (cur === o[0]) + '">' + esc(o[1]) + '</button>'; }).join('') + '</div>'; };
  var table = function (label, caption, head, body, cls) { return '<div class="tbl-wrap"><table class="tbl' + (cls ? ' ' + cls : '') + '" aria-label="' + esc(label) + '"><caption class="sr">' + esc(caption) + '</caption><thead><tr>' + head + '</tr></thead><tbody>' + body + '</tbody></table></div>'; };
  var notes = function (kind, text) { return '<div class="note ' + kind + '">' + ic(kind === 'danger' ? 'lock' : kind === 'info' ? 'info' : 'alert') + '<span>' + text + '</span></div>'; };

  /* ================= Today / Work Queue ================= */
  V.today = function () {
    var W = X.work(), cs = M.cases, rr = X.importRuns().filter(function (r) { return r.state === 'partial' || r.state === 'unknown'; }).length;
    var retBad = cs.filter(function (c) { return c.stage === 'delivered' && c.ret !== 'ok' && c.ret !== 'na'; }).length, linBad = cs.filter(function (c) { return c.val === 'valid' && c.lin !== 'verified'; }).length;
    var kpis = [
      { label: 'کار نیازمند اقدام', value: fa(W.length), color: 'orange', keep: true, meaning: 'هر مورد مسئول بعدی دارد؛ این عدد کل داده‌ها را افراز نمی‌کند', basis: 'وضعیت در لحظه' },
      { label: 'ورود بدون نتیجه قطعی', value: fa(rr), color: 'amber', keep: true, meaning: 'اجراهای ناقص یا نامعلوم؛ «کامل» نیست', basis: 'اجراهای ورود فایل' },
      { label: 'برگشت مسدود یا نامعلوم', value: fa(retBad), color: 'red', meaning: 'شامل مسدود، نامعلوم و تعارض؛ «قابل برگشت» اثبات‌شده نیست', basis: 'ردیف‌های تحویل‌شده' }
    ];
    var list = '<section class="panel attn-panel" aria-labelledby="wq-h">' + ph('inbox', 'صف کار MIS', 'wq-h', '', fa(W.length) + ' مورد') +
      '<ul class="attn">' + W.map(function (w) {
        return '<li><button type="button" class="attn-row" data-act="' + w.act + '" style="--c:var(--' + w.tone + '-dot)"><span class="attn-ico t-' + w.tone + '">' + ic(w.icon) + '</span><span class="attn-txt"><b>' + esc(w.title) + '</b><span><span class="attn-own">مسئول بعدی: ' + esc(w.own) + '</span>' + esc(w.detail) + '</span></span>' + ic('arrowL') + '</button></li>';
      }).join('') + '</ul><div class="attn-foot">' + ic('info') + '<span>این فهرست ترتیب کار است، نه رتبه‌بندی. «مسئول بعدی» حوزه‌ای است که باید ادامه دهد؛ MIS فروشنده، مالی یا منابع انسانی نمی‌شود و هیچ دکمه رفع همگانی وجود ندارد.</span></div></section>';
    var li = X.latestImport(), lo = X.latestOp();
    var latest = '<section class="panel" aria-labelledby="lt-h">' + ph('upload', 'آخرین ورود در برابر آخرین عملیات', 'lt-h', X.basis('snap'), 'دو چیز متفاوت') +
      '<div class="two-cards"><div class="mini-card"><span class="own-l">' + ic('upload') + 'آخرین ورود فایل</span><b class="mono">' + esc(li.id) + '</b>' + runPill(li.state) + '<span class="muted">' + esc(li.at) + ' · ' + fa(li.seen) + ' ردیف دیده‌شده</span>' + X.srcCtx({ file: li.file, batch: li.batch, run: li.id }) + '</div>' +
      '<div class="mini-card"><span class="own-l">' + ic('activity') + 'آخرین عملیات (هر نوع)</span><b class="mono">' + esc(lo.id) + '</b>' + pill(X.RUNTYPE[lo.type], 'slate', 'send') + '<span class="muted">' + esc(lo.at) + ' · ' + esc(lo.note) + '</span><span class="muted">شمارنده‌های ورود: ' + na('این اجرا ورود فایل نیست؛ شمارنده ورود «۰» نیست') + '</span></div></div>' +
      '<p class="ind-note">' + ic('info') + ' خلاصه عملیات اخیر نباید «نتیجه ورود» نام بگیرد؛ نوع اجرا، دسته، فایل و زمان همیشه کنار نتیجه نمایش داده می‌شوند.</p></section>';
    var H = M.health;
    var health = '<section class="panel" aria-labelledby="hc-h">' + ph('activity', 'آمادگی ورود و سلامت داده', 'hc-h', X.cov('undef', 'علت نامعلوم'), 'بررسی ' + esc(H.checkedAt)) +
      '<div class="hc-grid"><div class="hc-cell warn"><span class="own-l">' + ic('alert') + 'هشدار ذخیره‌شده</span><b>' + esc(H.msg) + '</b><span class="muted">منبع: ' + esc(H.source) + '</span></div><div class="hc-vs" aria-hidden="true">' + ic('swap') + '</div><div class="hc-cell ok"><span class="own-l">' + ic('checkCircle') + 'خواندن زنده گزارش</span><b>موفق · داده خوانا است</b><span class="muted">بدون درخواست ترمیم</span></div></div>' +
      '<p class="ind-note">' + ic('question') + ' دو سیگنال با هم نمی‌خوانند و علت تأیید نشده است. «بازسازی ساختار» از اینجا اجرا نمی‌شود (فقط در بخش نگهداری، پس از تشخیص تازه)؛ عدد «ورود ۰» هم نمایش داده نمی‌شود. <button type="button" class="linkish" data-act="goto:diag">جزئیات تشخیص</button></p></section>';
    var cnt = X.counters();
    var crow = cnt.map(function (c) {
      var tr = c.t === 'untrusted' ? pill('قابل اتکا نیست (F01)', 'red', 'lock') : c.t === 'diff' ? pill('شرط متفاوت', 'violet', 'swap') : pill('شرط روشن', 'teal', 'checkCircle');
      return '<tr><td><b>' + esc(c.n) + '</b></td><td class="muted wrap">' + esc(c.p) + '</td><td class="n mt-v">' + (c.v == null ? na('شمارش در دسترس نیست') : fa(c.v)) + '</td><td>' + tr + '</td><td class="col-opt muted wrap">' + esc(c.tip) + '</td></tr>';
    }).join('');
    var counters = '<section class="panel" aria-labelledby="ov-h">' + ph('layers', 'عددهای هم‌پوشان', 'ov-h', X.grain('ردیف منبع MIS'), 'جمع نمی‌شوند') +
      '<div class="note warn inset">' + ic('alert') + '<span><b>این عددها افراز یک کل نیستند.</b> «منتظر + نزد فروش + قابل برگشت» برابر «ردیف معتبر» نیست؛ هر ردیف شرط خودش را دارد و ممکن است در چند عدد بیاید. ناموجود یا محاسبه‌نشده صفر نیست.</span></div>' +
      table('عددهای هم‌پوشان', 'هر ردیف یک شرط شمارش مستقل با واحد ردیف منبع؛ جمع نکنید.', '<th>عدد</th><th>شرط شمارش</th><th class="n">مقدار</th><th>اعتماد</th><th class="col-opt">معنا</th>', crow, 'no-cursor') + '</section>';
    return h.pageHead({ title: 'امروز — صف کار', sub: 'ورود، کیفیت منبع، ردیابی تحویل و اعتماد گزارش · MIS اپراتور فروش یا مالی نیست', kpis: kpis, fresh: X.freshPart('صف کار'), scope: 'محدوده MIS · همه دسته‌های مجاز' }) + X.banners('today') +
      '<div class="team-grid">' + list + '<div class="team-main">' + latest + health + counters + '</div></div>';
  };

  /* ================= Import & Data Quality ================= */
  function concepts() {
    return '<div class="four-concepts">' + ic('layers') + '<span><b>چهار مفهوم جدا:</b>' + [['file', 'فایل', 'فایل بارگذاری‌شده'], ['batch', 'دسته', 'در نمای قدیمی «پرونده» نام دارد؛ پرونده تجاری نیست'], ['run', 'اجرای ورود', 'یک تلاش پردازش با نتیجه'], ['row', 'ردیف منبع', 'رکورد ذخیره‌شده با شناسه بومی']].map(function (x) { return '<span class="sc sc-' + x[0] + ' tip" tabindex="0" data-tip="' + esc(x[2]) + '"><small>' + x[1] + '</small></span>'; }).join('') + '</span></div>';
  }
  V.import = function () {
    var q = st.iq, runs = M.runs;
    var qs = h.queues([{ id: 'runs', label: 'اجراها و دسته‌ها', icon: 'activity', n: runs.length, key: '1' }, { id: 'files', label: 'فایل‌ها', icon: 'file', n: M.files.length, key: '2' }, { id: 'quality', label: 'کیفیت منبع', icon: 'shield', key: '3' }], q, 'data-iq');
    var body;
    if (q === 'runs') {
      var rows = runs.map(function (r) {
        var imp = r.type === 'import', f = function (v) { return imp ? fa(v) : na('این اجرا ورود فایل نیست'); };
        return '<tr data-row="run:' + r.id + '" tabindex="-1"><td>' + X.who(r.id, pill(X.RUNTYPE[r.type], imp ? 'blue' : 'slate', imp ? 'upload' : 'send')) + '</td><td>' + runPill(r.state) + '</td>' +
          '<td class="n">' + f(r.seen) + '</td><td class="n">' + f(r.imported) + '</td><td class="n col-opt">' + (imp ? fa(X.skippedOf(r)) + '<span class="of"> (' + fa(r.dup) + ' تکراری + ' + fa(r.invalid) + ' نامعتبر)</span>' : na('نامربوط')) + '</td><td class="n">' + (imp ? (r.failed ? '<b class="neg">' + fa(r.failed) + '</b>' : fa(0)) : na('نامربوط')) + '</td><td class="n">' + (imp ? (r.unknown ? '<b class="unk">' + fa(r.unknown) + '</b>' : fa(0)) : na('نامربوط')) + '</td>' +
          '<td class="col-opt">' + (imp ? (X.sumOk(r) ? '<span class="chk-ok">' + ic('checkCircle') + 'جمع‌ها می‌خوانند</span>' : pill('جمع‌ها نمی‌خوانند', 'orange', 'alert')) : '<span class="muted">—</span>') + '</td>' +
          '<td class="col-opt">' + X.srcCtx({ file: r.file, batch: r.batch }) + '<span class="cell-sub">' + esc(r.at) + '</span></td><td class="col-actions">' + open('run', r.id, 'مشاهده') + '</td></tr>';
      }).join('');
      body = table('اجراها و دسته‌های ورود', 'هر ردیف یک اجرا؛ شمارنده‌های ورود فقط برای اجرای نوع ورود فایل معنی دارند و برای بقیه صفر نیستند.', '<th>اجرا</th><th>نتیجه</th><th class="n">دیده‌شده</th><th class="n">واردشده</th><th class="n col-opt">نادیده‌گرفته‌شده</th><th class="n">ناموفق</th><th class="n">نامعلوم</th><th class="col-opt">تطبیق شمارها</th><th class="col-opt">فایل › دسته · زمان</th><th class="col-actions"><span class="sr">اقدام</span></th>', rows);
    } else if (q === 'files') {
      var frows = M.files.map(function (f) {
        return '<tr data-row="file:' + f.id + '" tabindex="-1"><td>' + X.who(f.name, '<span class="mono">' + f.id + '</span>') + '</td><td>' + pill(f.fmt, 'slate', 'file') + '</td><td class="n col-opt">' + esc(f.size) + '</td><td class="col-opt">' + esc(f.at) + '</td><td class="col-opt">' + esc(f.actor) + '</td><td>' + X.srcCtx({ batch: f.batch }) + '</td><td class="col-actions">' + open('file', f.id, 'مشاهده') + '</td></tr>';
      }).join('');
      body = table('فایل‌های بارگذاری‌شده', 'هر ردیف یک فایل؛ فایل با دسته و ردیف‌های منبع یکی نیست و حذف فایل رکوردها را حذف نمی‌کند.', '<th>فایل</th><th>قالب</th><th class="n col-opt">اندازه</th><th class="col-opt">بارگذاری</th><th class="col-opt">عامل</th><th>دسته</th><th class="col-actions"><span class="sr">اقدام</span></th>', frows) +
        '<p class="ind-note">' + ic('info') + ' سقف فعلی فایل ۵ مگابایت است؛ قالب‌های CSV و XLSX پشتیبانی می‌شوند. شماره‌ردیف ورودی (row_number) فقط محل ردیف در فایل است، شناسه سراسری یا پرونده نیست.</p>';
    } else {
      var rid = st.run || 'R-9921', r = X.run(rid) && X.run(rid).type === 'import' ? X.run(rid) : X.run('R-9921');
      var imps = X.importRuns();
      body = '<div class="ex-bar">' + seg('data-run', r.id, imps.map(function (x) { return [x.id, x.id + ' · ' + x.batch]; }), 'اجرای ورود برای بررسی کیفیت') + X.srcCtx({ file: r.file, batch: r.batch, run: r.id }) + '</div>' + qualityBody(r);
    }
    return h.pageHead({ title: 'ورود و کیفیت داده', sub: 'فایل ← دسته ← اجرا ← ردیف منبع · هر شمار با دلیل و تازگی · هیچ ادغام یا اصلاح خودکار انجام نمی‌شود', cta: '<button type="button" class="btn btn-primary" data-act="open-importnew">' + ic('upload') + 'ورود فایل جدید</button>', fresh: X.freshPart('اجراها'), scope: X.zone('normal', 'ورود فایل = عملیات عادی') }) +
      X.banners('import') + concepts() +
      h.banner('info', '<b>پیش‌نمایش ورود بدون اثر جانبی تضمین‌شده نیست.</b> در مسیر ورود فعلی، پاک‌سازی ردیف‌های تکراریِ هم‌دسته و ترمیم ساختار ممکن است پیش از تجزیه اجرا شوند (طبق کد بررسی‌شده؛ اجرا نشده است). حذف و پاک‌سازی عمدی فقط در «نگهداری» انجام می‌شود.', '') +
      '<section class="panel main" aria-labelledby="imp-h">' + ph('upload', 'ورود و کیفیت داده', 'imp-h', X.basis('snap'), '') + qs + (q === 'quality' ? '' : h.toolbar('جستجوی اجرا، دسته یا فایل…', [], '')) + body + (q === 'runs' ? h.tfoot(runs.length, runs.length) : '') + '</section>';
  };
  var qualityBody = X.qualityBody = function (r) {
    var eq = '<p class="eq-line mono">' + fa(r.seen) + ' = ' + fa(r.imported) + ' واردشده + ' + fa(r.dup) + ' تکراری + ' + fa(r.invalid) + ' نامعتبر + ' + fa(r.failed) + ' ناموفق + ' + fa(r.unknown) + ' نامعلوم ' + (X.sumOk(r) ? '<span class="chk-ok">' + ic('checkCircle') + 'می‌خواند</span>' : '') + '</p>';
    var facets = '<div class="facets q-facets">' + [['دیده‌شده (تجزیه)', r.seen, 'slate'], ['واردشده', r.imported, 'green'], ['تکراری', r.dup, 'orange'], ['نامعتبر', r.invalid, 'red'], ['ناموفق', r.failed, 'red'], ['نامعلوم', r.unknown, 'amber']].map(function (f) {
      return '<div class="facet"><span class="muted">' + f[0] + '</span><b class="mt-v">' + fa(f[1]) + '</b></div>';
    }).join('') + '</div>';
    var dc = r.dupClasses.length ? table('طبقه‌های تکراری', 'تعداد ردیف تکراری به تفکیک طبقه؛ شماره یکسان به معنی ادغام نیست.', '<th>طبقه تکرار</th><th class="n">تعداد</th><th>معنا</th>', r.dupClasses.map(function (d) { return '<tr><td>' + pill(d[1], 'orange', 'copy') + '</td><td class="n">' + fa(d[2]) + '</td><td class="muted wrap">' + (d[0] === 'legacy' ? 'مشابه لید قدیمی؛ مالک آن حوزه تصمیم می‌گیرد' : 'ردیف ذخیره می‌شود و ادغام نمی‌شود') + '</td></tr>'; }).join(''), 'no-cursor') : '<p class="ind-note">' + ic('info') + ' طبقه‌ای ثبت نشده است (تجزیه کامل نشد یا تکراری یافت نشد؛ این دو یکی نیست).</p>';
    var warn = r.warns.length ? '<ul class="elig warn-list">' + r.warns.map(function (w) { return '<li class="e-warn">' + ic('alert') + '<span><b>' + esc(w) + '</b></span></li>'; }).join('') + '</ul>' : '<p class="ind-note">' + ic('checkCircle') + ' هشدار تجزیه‌ای ثبت نشده است.</p>';
    return '<div class="q-body">' + X.sec('شمارنده‌ها و تطبیق', X.basis('snap'), facets + eq) + X.sec('طبقه‌های تکرار', '<button type="button" class="linkish" data-act="goto-cases:dup">مشاهده ردیف‌ها</button>', dc) + X.sec('هشدارهای تجزیه و دلیل‌ها', '', warn) +
      '<section class="sec"><div class="note warn">' + ic('alert') + '<span><b>نمونه محدود است.</b> فهرست ردیف‌های ردشده/نامعتبر فقط نمونه محدود نتیجه است و پوشش کامل نتیجه همه ردیف‌ها تأیید نشده؛ نگهداری کامل ردیف ردشده به سیاست فیلد/حریم خصوصی وابسته است.</span></div></section></div>';
 };

  /* ================= Source Cases (rows + lineage) ================= */
  V.cases = function () {
    var q = st.sq, all = M.cases;
    var filt = all.filter(function (c) { return q === 'all' ? true : q === 'lin' ? c.val === 'valid' && c.lin !== 'verified' : c.val === q; });
    var qs = h.queues([{ id: 'all', label: 'همه', icon: 'rows', n: all.length, key: '1' }, { id: 'valid', label: 'معتبر', icon: 'checkCircle', n: all.filter(function (c) { return c.val === 'valid'; }).length, key: '2' }, { id: 'dup', label: 'تکراری', icon: 'copy', n: all.filter(function (c) { return c.val === 'dup'; }).length, key: '3', tone: 'orange' }, { id: 'invalid', label: 'نامعتبر', icon: 'xCircle', n: all.filter(function (c) { return c.val === 'invalid'; }).length, key: '4', tone: 'red' }, { id: 'lin', label: 'اتصال غیرتأییدشده', icon: 'link', n: all.filter(function (c) { return c.val === 'valid' && c.lin !== 'verified'; }).length, key: '5', tone: 'amber' }], q, 'data-sq');
    var rows = filt.map(function (c) {
      var blockers = X.finBlockers(c).length;
      return '<tr data-row="case:' + c.id + '" tabindex="-1"><td>' + X.who('ردیف MIS ' + fa(c.id), '<span class="muted">دسته ' + c.batch + ' · ردیف فایل ' + fa(c.row) + '</span>') + '</td><td>' + valPill(c.val) + (c.reason ? '<span class="cell-sub">' + esc(c.reason) + '</span>' : '') + '</td><td>' + X.lin(c.lin) + '</td>' +
        '<td class="col-opt"><span class="cell-sub">' + esc(X.STAGE[c.stage]) + '</span></td><td class="col-opt">' + (c.val !== 'valid' ? '<span class="muted">—</span>' : blockers ? pill(fa(blockers) + ' حوزه با وابستگی/نامعلوم', 'amber', 'lock') : pill('هیچ وابستگی یافت نشد', 'teal', 'checkCircle')) + '</td>' +
        '<td class="col-opt"><span class="phone-tag tip" tabindex="0" data-tip="شماره فقط برای جستجو و کشف است؛ هویت پرونده یا مدرک اتصال مالی نیست.">' + ic('phone') + c.phone + '<small>کشف · نه هویت</small></span></td><td class="col-actions">' + open('case', c.id, 'بازرسی اتصال') + '</td></tr>';
    }).join('');
    return h.pageHead({ title: 'ردیف‌های منبع و اتصال', sub: 'شناسه بومی ← دسته ← اتصال به حوزه‌های دیگر · «پرونده منطقی» تعریف‌نشده است و ساخته یا حدس زده نمی‌شود', fresh: X.freshPart('ردیف‌ها'), scope: 'شماره موبایل فقط برای جستجو است' }) + X.banners('cases') +
      '<section class="panel main" aria-labelledby="cs-h">' + ph('link', 'ردیف‌های منبع', 'cs-h', X.grain('ردیف منبع MIS') + X.basis('snap'), '') + qs + h.toolbar('جستجو: شناسه ردیف، دسته یا شماره (کشف)…', [], '') +
      table('ردیف‌های منبع', 'هر ردیف یک رکورد منبع MIS با شناسه بومی؛ ستون اتصال میزان مدرک را نشان می‌دهد و شماره موبایل هویت نیست.', '<th>ردیف منبع (شناسه بومی)</th><th>اعتبار ردیف</th><th>اتصال</th><th class="col-opt">مرحله</th><th class="col-opt">وابستگی مالی</th><th class="col-opt">شماره (فقط کشف)</th><th class="col-actions"><span class="sr">اقدام</span></th>', rows) + h.tfoot(filt.length, all.length) + '</section>';
  };

  /* ================= Custody & Delivery ================= */
  function recipPanel() {
    var single = st.cq === 'assign';
    if (!single) return '<aside class="recips" aria-labelledby="rc-h"><div class="sec-h"><h3 id="rc-h">مقصد</h3><span class="scope-badge">' + ic('layers') + 'استخر</span></div><p class="recip-note">' + ic('info') + 'آماده‌سازی استخر ردیف منبع را برای تحویل آماده می‌کند؛ <b>لید زنده نمی‌سازد</b> و گیرنده‌ای انتخاب نمی‌شود. ایجاد لید زنده فقط از «برنامه‌ریزی» با تأیید جدا ممکن است.</p></aside>';
    return '<aside class="recips" aria-labelledby="rc-h"><div class="sec-h"><h3 id="rc-h">گیرنده (مدیر فروش)</h3><span class="scope-badge">' + ic('users') + 'مسیر عادی</span></div>' +
      M.managers.map(function (m) {
        var on = st.recip === m.id;
        return '<label class="recip' + (m.ok ? '' : ' off') + (on ? ' on' : '') + '"><input type="radio" name="recip" class="cbx" data-recip="' + m.id + '"' + (on ? ' checked' : '') + (m.ok ? '' : ' disabled') + '><span class="r-main"><b>' + esc(m.name) + '</b><span>' + esc(m.ok ? m.note : 'قابل انتخاب نیست — ' + m.why) + '</span></span></label>';
      }).join('') + '<p class="recip-note">' + ic('alert') + 'گیرنده باید مدیر فروش فعال باشد. نمایش یک نفر به معنی مجاز بودن نیست و جایگزین خودکار انتخاب نمی‌شود. تحویل پنج‌سطحی استثنایی در «تحویل سریع (مشروط)» است.</p></aside>';
  }
  V.custody = function () {
    var q = st.cq, body = '', pageSteps = '';
    var qs = h.queues([{ id: 'assign', label: 'تخصیص منبع به مدیر', icon: 'send', n: X.cand().length, key: '1' }, { id: 'pool', label: 'آماده‌سازی استخر', icon: 'layers', key: '2' }, { id: 'hist', label: 'سابقه تحویل', icon: 'history', key: '3' }, { id: 'return', label: 'برگشت (مشروط)', icon: 'lock', key: '4', tone: 'red' }], q, 'data-cq');
    if (q === 'assign' || q === 'pool') {
      var cand = X.cand(), sel = X.keys(st.asel).filter(function (k) { return X.cs(k); }), ready = q === 'assign' ? st.recip && sel.length : sel.length;
      var rows = cand.map(function (c) {
        return '<tr data-row="case:' + c.id + '" tabindex="-1"' + (st.asel[c.id] ? ' class="sel"' : '') + '><td class="col-sel">' + h.cbx('data-sel="asel" data-id="' + c.id + '"', st.asel[c.id], 'انتخاب ردیف ' + c.id) + '</td><td>' + X.who('ردیف MIS ' + fa(c.id), '<span class="muted">دسته ' + c.batch + ' · ردیف فایل ' + fa(c.row) + '</span>') + '</td><td>' + valPill(c.val) + '</td><td>' + X.lin(c.lin) + '</td><td class="col-opt"><span class="cell-sub">' + esc(X.STAGE[c.stage]) + '</span></td><td class="col-actions">' + open('case', c.id, 'اتصال') + '</td></tr>';
      }).join('');
      pageSteps = '<div class="flow-head">' + X.steps(['انتخاب ردیف‌ها', 'بررسی اثر', 'نتیجه'], sel.length ? 1 : 0) + '</div>';
      body = pageSteps + (q === 'assign' ? h.banner('info', '<b>تخصیص منبع فقط فیلد منبع (assigned_manager) و وضعیت دسته را تغییر می‌دهد.</b> تحویل فعلی، سابقه توزیع و لید زنده تغییر نمی‌کنند؛ این سه یکی نیستند.', '') : h.banner('info', '<b>آماده‌سازی استخر لید زنده نمی‌سازد.</b> این گام ردیف را برای تحویل آماده می‌کند و با تأیید و ایجاد برنامه جداست.', '')) +
        '<div class="split"><div class="split-main">' + (sel.length ? X.bulkbar(fa(sel.length) + ' ردیف انتخاب شد', q === 'assign' && !st.recip ? 'ابتدا یک مدیر مجاز انتخاب کنید' : 'پیش از ثبت، اثر هر ردیف جدا بررسی می‌شود', '<button type="button" class="btn" data-act="clear-asel">پاک کردن انتخاب</button><button type="button" class="btn btn-primary" data-act="review-assign"' + (ready ? '' : ' disabled aria-disabled="true"') + '>بررسی اثر (' + fa(sel.length) + ')</button>') : '') +
        table('ردیف‌های آماده تخصیص', 'ردیف‌های معتبر بدون تخصیص منبع؛ با انتخاب هر ردیف اثر آن جدا بررسی می‌شود.', '<th class="col-sel">' + h.cbx('data-selall="asel"', sel.length === cand.length && cand.length, 'انتخاب همه') + '</th><th>ردیف منبع</th><th>اعتبار</th><th>اتصال</th><th class="col-opt">مرحله</th><th class="col-actions"><span class="sr">اقدام</span></th>', rows) + h.tfoot(cand.length, cand.length) + '</div>' + recipPanel() + '</div>';
    } else if (q === 'hist') {
      var ev = M.cases.filter(function (c) { return c.stage !== 'none' && c.stage !== 'unassigned'; });
      var hrows = ev.map(function (c) {
        var l = c.hist[0];
        return '<tr data-row="tl:' + c.id + '" tabindex="-1"><td>' + X.who('ردیف MIS ' + fa(c.id), '<span class="muted">دسته ' + c.batch + '</span>') + '</td><td>' + esc(l[0]) + '</td><td class="col-opt">' + esc(l[1]) + '</td><td class="col-opt">' + esc(c.cust.event) + '</td><td>' + esc(c.cust.cur) + '</td><td class="col-opt"><span class="cell-sub">' + esc(c.cust.next) + '</span></td><td class="col-actions">' + open('tl', c.id, 'خط زمانی') + '</td></tr>';
      }).join('');
      body = h.banner('info', '<b>مسئول فعلی ≠ فیلد منبع ≠ مالک اولیه ≠ مالک اعتبار.</b> در خط زمانی هر رویداد با عامل، گیرنده، زمان و نتیجه جدا نمایش داده می‌شود؛ ساختار امروز سابقه را بازنویسی نمی‌کند.', '') + table('سابقه تحویل', 'آخرین رویداد هر ردیف تحویل‌شده یا تخصیص‌یافته؛ برای زنجیره کامل خط زمانی را باز کنید.', '<th>ردیف منبع</th><th>آخرین رویداد</th><th class="col-opt">زمان</th><th class="col-opt">عامل رویداد</th><th>مسئول فعلی</th><th class="col-opt">اقدام بعدی با</th><th class="col-actions"><span class="sr">اقدام</span></th>', hrows) + h.tfoot(ev.length, ev.length);
    } else {
      var del = M.cases.filter(function (c) { return c.stage === 'delivered'; }), f = st.rf, list = f === 'all' ? del : del.filter(function (c) { return c.ret === f; });
      var rsel = X.keys(st.rsel).filter(function (k) { var c = X.cs(k); return c && c.ret === 'ok'; });
      var cnt = function (k) { return del.filter(function (c) { return c.ret === k; }).length; };
      var rf = seg('data-rf', f, [['all', 'همه (' + fa(del.length) + ')'], ['ok', 'قابل برگشت — پیش‌نمایش (' + fa(cnt('ok')) + ')'], ['blocked', 'مسدود (' + fa(cnt('blocked')) + ')'], ['conflict', 'تعارض (' + fa(cnt('conflict')) + ')'], ['unknown', 'نامعلوم (' + fa(cnt('unknown')) + ')']], 'فیلتر وضعیت برگشت');
      var rrows = list.map(function (c) {
        var blockers = X.finBlockers(c);
        return '<tr data-row="ret:' + c.id + '" tabindex="-1"><td class="col-sel">' + h.cbx('data-sel="rsel" data-id="' + c.id + '"', st.rsel[c.id], 'انتخاب ردیف ' + c.id, c.ret !== 'ok') + '</td><td>' + X.who('ردیف MIS ' + fa(c.id), '<span class="muted">' + X.lin(c.lin).replace(/<[^>]*>/g, '') + '</span>') + '</td><td>' + esc(c.cust.cur) + '</td><td class="col-opt">' + (blockers.length ? pill(fa(blockers.length) + ' حوزه', 'amber', 'lock') : pill('هیچ‌کدام یافت نشد', 'teal', 'checkCircle')) + '</td><td>' + retPill(c.ret) + (c.retWhy ? '<span class="why-line ' + (c.ret === 'ok' ? 'warn' : '') + '">' + ic('alert') + esc(c.retWhy) + '</span>' : '') + '</td><td class="col-actions">' + open('ret', c.id, 'بررسی') + '</td></tr>';
      }).join('');
      body = h.banner('locked', '<b>برگشت مشروط و فقط پیش‌نمایش است (F01 / MIS-G01).</b> محافظ فعلی همه ارتباط‌های مالی را نمی‌بیند؛ «قابل برگشت» در این صفحه اجازه اجرا نیست. لغو/رد فاکتور به‌تنهایی آزادسازی نیست (OPD-03) و برای موارد مبهم هیچ دکمه بازنشانی وجود ندارد.', '') +
        '<div class="ex-bar">' + rf + '</div>' + (rsel.length ? X.bulkbar(fa(rsel.length) + ' ردیف برای پیش‌نمایش گروهی', 'ثبت برگشت گروهی غیرفعال است', '<button type="button" class="btn" data-act="clear-rsel">پاک کردن انتخاب</button><button type="button" class="btn btn-primary" data-act="review-return">پیش‌نمایش برگشت (' + fa(rsel.length) + ')</button>') : '') +
        table('وضعیت برگشت ردیف‌های تحویل‌شده', 'برای هر ردیف مسئول فعلی، وابستگی مالی به تفکیک حوزه و وضعیت برگشت؛ فقط «قابل برگشت (پیش‌نمایش)» انتخاب‌شدنی است.', '<th class="col-sel"><span class="sr">انتخاب</span></th><th>ردیف منبع</th><th>مسئول فعلی</th><th class="col-opt">وابستگی/نامعلوم</th><th>وضعیت برگشت</th><th class="col-actions"><span class="sr">اقدام</span></th>', rrows) + h.tfoot(list.length, del.length);
    }
    return h.pageHead({ title: 'مسئولیت و تحویل', sub: 'تخصیص منبع ← آماده‌سازی استخر ← تحویل ← برگشت (مشروط) · مالک منبع، مسئول فعلی، مالک اولیه، اقدام بعدی، مالک اعتبار، عامل رویداد و گیرنده جدا نمایش داده می‌شوند', fresh: X.freshPart('تحویل'), scope: 'تخصیص و استخر = عملیات عادی' }) + X.banners('custody') + '<section class="panel main has-bulk" aria-labelledby="cu-h">' + ph('send', 'مسئولیت و تحویل', 'cu-h', X.basis('snap'), '') + qs + body + '</section>';
  };

  /* ================= Reconciliation & Reports ================= */
  V.rec = function () {
    var q = st.rq, qs = h.queues([{ id: 'issues', label: 'مسائل تطبیق', icon: 'swap', n: M.issues.length, key: '1', tone: 'orange' }, { id: 'reports', label: 'گزارش‌ها و اعتماد', icon: 'chart', n: M.reports.length, key: '2' }], q, 'data-rq'), body = '';
    if (q === 'issues') {
      var list = st.icls === 'all' ? M.issues : M.issues.filter(function (i) { return i.cls === st.icls; }), sel = X.keys(st.isel);
      var clsChips = '<div class="ex-bar cls-bar" role="group" aria-label="نوع مسئله">' + [['all', 'همه نوع‌ها']].concat(Object.keys(M.classes).map(function (k) { return [k, M.classes[k].label]; })).map(function (c) { return '<button type="button" class="chip-btn' + (st.icls === c[0] ? ' on' : '') + '" data-icls="' + c[0] + '" aria-pressed="' + (st.icls === c[0]) + '">' + esc(c[1]) + '</button>'; }).join('') + '</div>';
      var rows = list.map(function (i) {
        var c = M.classes[i.cls], s = X.ISSTATE[i.state];
        return '<tr data-row="issue:' + i.id + '" tabindex="-1"><td class="col-sel">' + h.cbx('data-sel="isel" data-id="' + i.id + '"', st.isel[i.id], 'انتخاب مسئله ' + i.id) + '</td><td class="wrap"><div class="exc-cls">' + pill(c.label, i.sev, c.icon) + '</div><b class="exc-subj">' + esc(i.title) + '</b></td><td class="col-opt wrap"><span class="mono">' + esc(i.entity[1]) + '</span><span class="cell-sub">' + esc(i.entity[2]) + '</span></td><td class="col-opt wrap">' + esc(i.owner) + '</td><td>' + X.lin(i.conf) + '</td><td>' + pill(s[0], s[1], s[2]) + '</td><td class="col-opt wrap"><span class="allowed">' + ic('eye') + esc(i.allowed[0]) + '</span></td><td class="col-actions">' + open('issue', i.id, 'شواهد') + '</td></tr>';
      }).join('');
      body = h.banner('locked', '<b>فضای تشخیص است، نه ابزار ترمیم.</b> هر مسئله شواهد، مالک حوزه، اقدام مجاز MIS، اقدام غیرمجاز و مدرک حل دارد. «رفع همه» وجود ندارد و تطبیق گروهی فقط خواندنی است (مشروط).', '') + clsChips +
        (sel.length ? X.bulkbar(fa(sel.length) + ' مسئله برای تطبیق خواندنی', 'هیچ داده‌ای تغییر نمی‌کند · مشروط', '<button type="button" class="btn" data-act="clear-isel">پاک کردن انتخاب</button><button type="button" class="btn btn-primary" data-act="review-recon">پیش‌نمایش تطبیق خواندنی (' + fa(sel.length) + ')</button>') : '') +
        table('مسائل تطبیق', 'هر ردیف یک مسئله با نوع، موجودیت متأثر، مالک حوزه، اطمینان و اقدام مجاز MIS.', '<th class="col-sel"><span class="sr">انتخاب</span></th><th>مسئله</th><th class="col-opt">موجودیت متأثر</th><th class="col-opt">مالک / حوزه</th><th>اطمینان</th><th>وضعیت</th><th class="col-opt">اقدام مجاز MIS</th><th class="col-actions"><span class="sr">اقدام</span></th>', rows) + h.tfoot(list.length, M.issues.length);
    } else if (!st.rep) {
      body = '<div class="rep-grid cat-pad">' + M.reports.map(function (r) {
        return '<button type="button" class="rep-card" data-act="open-report:' + r.id + '"><span class="rep-ico">' + ic('chart') + '</span><b>' + esc(r.name) + '</b><span class="rep-tags">' + X.basis(r.basis) + X.grain(r.grain) + X.cov(r.fresh === 'stale' ? 'stale' : r.bounded ? 'bounded' : 'ok') + '</span><span class="rep-go">زمینه اعتماد و شاخص‌ها ' + ic('arrowL') + '</span></button>';
      }).join('') + '</div>';
    } else body = reportDetail();
    return h.pageHead({ title: 'گزارش‌ها و تطبیق', sub: 'مسئله‌ها با مدرک حل · هر گزارش با واحد شمارش، پوشش منبع، مبنای زمانی، تازگی و نسخه تعریف · هیچ ادعای برابری ساختگی نیست', fresh: X.freshPart('گزارش‌ها'), scope: X.zone('normal', 'تطبیق خواندنی') }) + X.banners('rec') + '<section class="panel main has-bulk" aria-labelledby="rc2-h">' + ph('swap', 'گزارش‌ها و تطبیق', 'rc2-h', '', '') + qs + body + '</section>';
  };
  function reportDetail() {
    var r = X.report(st.rep), b = st.tb;
    var tbSeg = '<div class="seg" role="group" aria-label="مبنای زمانی">' + [['snap', 'وضعیت در لحظه'], ['range', 'رویداد در بازه + وضعیت جاری گردآوری']].map(function (o) { return '<button type="button" data-tbs="' + o[0] + '" aria-pressed="' + (b === o[0]) + '">' + o[1] + '</button>'; }).join('') + '<button type="button" disabled aria-disabled="true" class="tip" data-tip="وضعیت تاریخی فقط با شواهد کافی بازسازی می‌شود؛ رویداد در بازه + وضعیت جاری هرگز به‌جای آن نمایش داده نمی‌شود (F09).">وضعیت تاریخی (فعال نیست)</button></div>';
    var shown = r.metrics.filter(function (m) { return m.basis === b; }), other = r.metrics.length - shown.length;
    var ctx = '<div class="trust-grid" role="group" aria-label="زمینه اعتماد گزارش">' + [
      ['واحد شمارش (grain)', r.grain], ['گروه (cohort)', r.cohort], ['محدوده', r.scope], ['نسخه تعریف شاخص', r.ver], ['بازه گردآوری', X.fmtTime(r.gather[0], r.gather[1])], ['تازگی', r.fresh === 'stale' ? 'قدیمی — بازخوانی لازم است' : 'امروز ' + M.freshness.now]
    ].map(function (x) { return '<div class="facet"><span class="muted">' + x[0] + '</span><b>' + esc(x[1]) + '</b></div>'; }).join('') + '</div>';
    return '<div class="rep-detail"><div class="ex-bar"><button type="button" class="btn btn-ghost" data-act="close-report">' + ic('arrowL') + 'همه گزارش‌ها</button><b>' + esc(r.name) + '</b>' + X.grain(r.grain) + '</div>' +
      '<div class="rep-ctrl-panel"><div class="rep-ctrl">' + tbSeg + X.basis(b) + '</div></div>' +
      (r.bounded ? h.banner('incomplete', '<b>گزارش محدود است.</b> فقط بخشی از جمعیت گردآوری شد؛ مجموع نهایی نیست و نبود مورد بیشتر اثبات نمی‌شود. گردآوری مرحله‌ای است و برش اتمیک یک لحظه نیست.', '') : '') +
      (r.fresh === 'stale' ? h.banner('stale', '<b>این گزارش قدیمی است.</b> بازخوانی خواندنی یک کار فنی است و داده منبع را تغییر نمی‌دهد.', '<button type="button" class="btn btn-sm" data-refresh>' + ic('refresh') + 'بازخوانی</button>') : '') +
      X.sec('زمینه اعتماد', X.cov(r.bounded ? 'bounded' : 'ok'), ctx) + X.sec('پوشش منبع', 'شمرده‌نشده یا اثبات‌نشده هرگز صفر نمایش داده نمی‌شود', X.covMatrix(r.cov, 'پوشش منبع گزارش')) +
      X.sec('شاخص‌ها در مبنای «' + (b === 'snap' ? 'وضعیت در لحظه' : 'رویداد در بازه + وضعیت جاری گردآوری') + '»', other ? fa(other) + ' شاخص در مبنای دیگر' : '', shown.length ? table('شاخص‌های گزارش', 'هر شاخص با پوشش و مبنای زمانی؛ مقدار تعریف‌نشده یا مغایر با خط تیره نمایش داده می‌شود نه صفر.', '<th>شاخص</th><th class="n">مقدار</th><th>اعتماد</th><th>مبنا</th><th class="col-opt">چرا</th>', X.metricRows(shown), 'no-cursor') : '<p class="ind-note">' + ic('info') + ' برای این گزارش شاخصی در این مبنا تعریف نشده است.</p>') +
      '<section class="sec"><div class="rep-foot"><button type="button" class="btn" disabled aria-disabled="true">' + ic('download') + 'خروجی XLSX — مشروط به مجوز صریح</button><span class="muted">خروجی همان محدوده و فهرست میدان‌های مجاز را با فراداده زمان، گروه، نسخه و پوشش می‌گیرد؛ اجرا و دانلود در این نمونه فعال نیست (MIS-G10).</span></div></section></div>';
  }

  /* ================= Planning (secondary, conditional) ================= */
  V.plan = function () {
    var rows = M.plans.map(function (p) {
      var idx = X.PLANST.map(function (s) { return s[0]; }).indexOf(p.stage);
      var chips = X.PLANST.map(function (s, i) { var done = i <= idx; return '<li class="' + (done ? 'done' : '') + (i === idx ? ' cur' : '') + '"' + (i === idx ? ' aria-current="step"' : '') + '><span class="sn">' + (done ? ic('check') : fa(i + 1)) + '</span><span class="st-l">' + s[1] + '</span></li>'; }).join('');
      var cons = p.stage === 'approved' ? pill('تأییدشده — هنوز لیدی ساخته نشده', 'amber', 'hourglass') : p.stage === 'materialized' ? pill('ایجاد لید: ناقص', 'orange', 'split') : p.stage === 'draft' ? pill('فقط دستورالعمل — چیزی تحویل نمی‌شود', 'slate', 'file') : pill('ذخیره‌شده', 'blue', 'file');
      return '<tr data-row="plan:' + p.id + '" tabindex="-1"><td>' + X.who(p.name, '<span class="mono">' + p.id + '</span> · دسته ' + p.batch) + '</td><td class="col-opt"><ol class="steps mini-steps" aria-label="مراحل برنامه">' + chips + '</ol></td><td>' + cons + '</td><td class="col-opt">' + esc(X.mgr(p.mgr).name) + '</td><td class="col-actions">' + open('plan', p.id, 'مشاهده') + '</td></tr>';
    }).join('');
    var rule = '<section class="panel" aria-labelledby="rd-h">' + ph('file', 'پیش‌نویس قانون', 'rd-h', X.zone('adv', 'برنامه‌ریزی'), '') + '<div class="rule-draft"><div class="facets"><div class="facet"><span class="muted">قانون</span><b>مساوی / دستی</b></div><div class="facet"><span class="muted">مدیر</span><b>' + esc(X.mgr('MG1').name) + '</b></div><div class="facet"><span class="muted">منبع</span><b>دسته B-1187</b></div></div>' +
      '<p class="ind-note">' + ic('info') + ' قانون یک دستورالعمل برنامه است، نه تخصیص فوری و نه پیش‌بینی. هیچ سهمیه، ظرفیت‌سنجی یا نمره برای فروشنده تعریف نشده است. کامل بودن فهرست کاندیدها «نیازمند اعتبارسنجی» است؛ «فروشنده‌ای در محدوده نیست» ثابت‌کننده نبود فروشنده نیست.</p></div></section>';
    return h.pageHead({ title: 'برنامه‌ریزی (مشروط)', sub: 'قانون ← پیش‌نمایش ← ذخیره ← تأیید ← ایجاد لید · هر مرحله جدا؛ تأیید برنامه به معنی ساخته‌شدن لید نیست', fresh: X.freshPart('برنامه‌ها'), scope: X.zone('adv', 'گاه‌به‌گاه، نه کار روزانه') }) + X.banners('plan') +
      h.banner('info', '<b>تأیید برنامه ≠ ایجاد لید.</b> تأیید فقط وضعیت برنامه را عوض می‌کند. ایجاد لید زنده گام جدا با تأیید جدا و نتیجه به‌تفکیک مورد است. پیش‌نمایش برنامه هم ممکن است اثر جانبی گزارش/پاک‌سازی داشته باشد و خالص‌خواندنی تضمین نمی‌شود.', '') + rule +
      '<section class="panel main" aria-labelledby="pl-h">' + ph('layers', 'برنامه‌ها', 'pl-h', '', '') + table('برنامه‌ها', 'هر ردیف یک برنامه با مرحله فعلی و پیامد مرحله؛ مرحله جلوتر به معنی انجام مرحله بعد نیست.', '<th>برنامه</th><th class="col-opt">مراحل</th><th>پیامد مرحله</th><th class="col-opt">مدیر</th><th class="col-actions"><span class="sr">اقدام</span></th>', rows) + h.tfoot(M.plans.length, M.plans.length) + '</section>';
  };

  /* ================= Quick Delivery (advanced / conditional) ================= */
  V.quick = function () {
    var b = M.quickBatches.filter(function (x) { return x.id === st.qb; })[0] || M.quickBatches[0], pos = M.positions.filter(function (p) { return p.id === st.qp; })[0];
    var okForm = pos && st.qr.trim().length > 3 && st.qn >= 1 && st.qn <= 500;
    var batches = '<div class="seg" role="group" aria-label="دسته منبع">' + M.quickBatches.map(function (x) { return '<button type="button" data-qb="' + x.id + '" aria-pressed="' + (st.qb === x.id) + '">' + x.id + ' · ' + esc(x.name) + '</button>'; }).join('') + '</div>';
    var recips = M.positions.map(function (p) { var on = st.qp === p.id; return '<label class="recip' + (on ? ' on' : '') + '"><input type="radio" name="qpos" class="cbx" data-qp="' + p.id + '"' + (on ? ' checked' : '') + '><span class="r-main"><b>' + esc(p.label) + '</b><span>گیرنده با این نقش — مسیر استثنایی</span></span></label>'; }).join('');
    return h.pageHead({ title: 'تحویل سریع (مشروط)', sub: 'مسیر مستقیم پنج‌سطحی · استثنایی و بدون سیاست مصوب برای دلیل/عبور از مدیر', scope: X.zone('adv', 'تحویل سریع'), fresh: X.freshPart('دسته‌ها') }) + X.banners('quick') +
      h.banner('stale', '<b>این مسیر پیش‌فرض تحویل نیست.</b> مسیر عادی «تخصیص منبع ← استخر ← تحویل» است. عبور از مدیر برای هر گیرنده یک استثنای مشروط است؛ سیاست دلیل و مجوز مصوب نیست (OPD)، و حذف یا تغییر مجوز فعلی انجام نشده است.', '<button type="button" class="btn btn-sm" data-act="goto:custody">مسیر عادی</button>') +
      '<section class="panel form-card" aria-labelledby="qd-h">' + ph('send', 'پارامترهای تحویل سریع', 'qd-h', X.zone('adv'), '') + '<div class="form-grid q-form">' +
      '<div><label class="lbl">دسته منبع</label>' + batches + '<p class="recip-note">' + ic('info') + 'موجودی قابل تحویل این دسته در پیش‌نمایش: <b>' + fa(b.avail) + '</b> ردیف (بررسی نهایی هنگام ثبت دوباره انجام می‌شود).</p></div>' +
      '<div><span class="lbl" id="qp-l">نقش گیرنده</span><div class="recips-inline" role="radiogroup" aria-labelledby="qp-l">' + recips + '</div><p class="recip-note">' + ic('alert') + 'هیچ گزینه‌ای پیش‌فرض نیست. گیرنده‌های پایین‌تر از مدیر یعنی «عبور از مدیر»؛ سطح‌های رد‌شده در بررسی اثر نمایش داده می‌شوند.</p></div>' +
      '<div><label class="lbl" for="q-n">تعداد (حداکثر ۵۰۰)</label><input class="input" id="q-n" type="number" min="1" max="500" value="' + st.qn + '" data-qn></div>' +
      '<div><label class="lbl" for="q-r">دلیل (برای ردیابی)</label><textarea class="input" id="q-r" rows="3" data-qr placeholder="دلیل این تحویل استثنایی را بنویسید">' + esc(st.qr) + '</textarea><p class="recip-note">' + ic('info') + 'دلیل ثبت می‌شود ولی سیاستی برای تأیید آن تعریف نشده و زنجیره تأیید جدید ساخته نشده است.</p></div>' +
      '<div class="form-acts"><button type="button" class="btn btn-primary btn-lg" data-act="review-quick"' + (okForm ? '' : ' disabled aria-disabled="true"') + '>بررسی اثر</button><span class="muted">' + (okForm ? '' : 'نقش گیرنده، تعداد معتبر و دلیل لازم است') + '</span></div></div></section>';
  };

  /* ================= Maintenance (advanced; visually separated) ================= */
  V.maint = function () {
    var zoneCol = function (k, items) { var z = X.ZONES[k]; return '<div class="zcol z-' + k + '"><h3>' + X.zone(k) + '</h3><ul>' + items.map(function (i) { return '<li>' + esc(i) + '</li>'; }).join('') + '</ul></div>'; };
    var legend = '<section class="panel" aria-labelledby="zl-h">' + ph('layers', 'مرز عملیات', 'zl-h', '', 'کدام کار کجاست') + '<div class="zone-legend">' +
      zoneCol('normal', ['مشاهده منبع، ورود و لاگ', 'تخصیص منبع و استخر', 'گزارش و تطبیق خواندنی']) + zoneCol('adv', ['تحویل سریع (مشروط)', 'برنامه‌ریزی', 'ایجاد لید کنترل‌شده', 'تخصیص مجدد استثنایی (نیازمند سیاست)']) + zoneCol('maint', ['حذف فایل', 'حذف کامل دسته', 'پاک‌سازی ردپا', 'بازسازی ساختار', 'ترمیم اتصال (مجوز تعریف نشده)']) + '</div></section>';
    var A = st.maintAuth === 'verified';
    var rows = M.maint.map(function (m) {
      var stt = m.block === 'forbidden' ? pill('در این طراحی در دسترس نیست', 'slate', 'ban') : m.block === 'protected' ? pill('مسدود · وابستگی محافظت‌شده', 'red', 'lock') : m.block === 'unknown' ? pill('مسدود · نامعلوم', 'amber', 'question') : m.block === 'partial' ? pill('بخشی مسدود', 'orange', 'split') : pill('قابل بررسی', 'teal', 'eye');
      return '<tr data-row="maint:' + m.id + '" tabindex="-1"><td>' + X.who(m.label, '<span class="muted">' + esc(m.target) + '</span>') + '</td><td>' + X.zone(m.zone) + '</td><td class="col-opt wrap muted">' + esc(m.desc) + '</td><td class="col-opt"><span class="mono">' + esc(m.auth) + '</span></td><td>' + stt + '</td><td class="col-actions">' + (m.block === 'forbidden' ? '<button type="button" class="btn btn-sm" disabled aria-disabled="true">' + ic('lock') + 'غیرفعال</button>' : open('maint', m.id, 'بررسی اثر', 'btn-soft')) + '</td></tr>';
    }).join('');
    var box = '<section class="panel main zone-box" aria-labelledby="mt-h"><div class="zone-box-h">' + ic('lock') + '<div><h2 id="mt-h">منطقه نگهداری و کارهای پیشرفته</h2><span>اینجا کار روزانه نیست. هر اقدام ابتدا بررسی اثر، وابستگی محافظت‌شده، دلیل و تأیید می‌خواهد؛ تخریبی‌ها عادی‌سازی نمی‌شوند.</span></div>' + X.zone('maint') + '</div>' +
      '<div class="auth-line">' + ic(A ? 'checkCircle' : 'lock') + '<span>اختیار نگهداری مستقل: <b>' + (A ? 'تأییدشده (حالت نمایشی دمو)' : 'تأیید نشده (MIS-G07)') + '</b> — ' + (A ? 'در واقعیت باید از سیاست مستقل بیاید؛ ورود به MIS یا دسترسی گسترده به داده این اختیار را نمی‌دهد.' : 'بررسی اثر ممکن است، ثبت اجرا غیرفعال است. ورود به MIS یا دسترسی گسترده به داده، اجازه حذف یا ترمیم نیست.') + '</span></div>' +
      table('اقدامات نگهداری', 'هر ردیف یک اقدام نگهداری یا پیشرفته با منطقه، اختیار و وضعیت محافظت؛ بررسی اثر هیچ داده‌ای را تغییر نمی‌دهد.', '<th>اقدام</th><th>منطقه</th><th class="col-opt">توضیح</th><th class="col-opt">اختیار فعلی (کد)</th><th>وضعیت</th><th class="col-actions"><span class="sr">اقدام</span></th>', rows) + '</section>';
    return h.pageHead({ title: 'نگهداری و پیشرفته', sub: 'حذف، پاک‌سازی، بازسازی و تخصیص استثنایی · جدا از کار روزانه · پشتیبانی نمی‌شود که از صفحه‌های گزارش یا خواندن اجرا شود', scope: X.zone('maint', 'پرمخاطره'), fresh: X.freshPart('وابستگی‌ها') }) + X.banners('maint') + legend + box;
  };

  /* ================= Diagnostics & operation log ================= */
  V.diag = function () {
    var q = st.dq, qs = h.queues([{ id: 'log', label: 'لاگ عملیات', icon: 'history', n: M.log.length, key: '1' }, { id: 'health', label: 'سلامت و آمادگی', icon: 'activity', key: '2' }, { id: 'map', label: 'نقشه مقصدهای قدیمی', icon: 'compass', key: '3' }], q, 'data-dq'), body;
    if (q === 'log') {
      var rows = M.log.map(function (l) {
        var r = X.RUNS[l.result];
        return '<tr data-row="log:' + l.id + '" tabindex="-1"><td>' + X.who(l.action, esc(l.at)) + '</td><td class="col-opt">' + esc(l.actor) + '</td><td>' + X.srcCtx({ batch: l.src.charAt(0) === 'B' ? l.src : '', run: l.run }) + (l.src.charAt(0) !== 'B' ? '<span class="cell-sub">' + esc(l.src) + '</span>' : '') + '</td><td class="col-opt wrap">' + esc(l.ref) + '</td><td>' + pill(r[0], r[1], r[2]) + '</td><td class="col-opt mono">' + esc(l.corr) + '</td><td class="col-actions">' + open('log', l.id, 'جزئیات') + '</td></tr>';
      }).join('');
      body = table('لاگ عملیات', 'هر ردیف یک عملیات با عامل، منبع، اجرا، ارجاع، نتیجه و شناسه همبستگی؛ جزئیات هر مورد در پنل است.', '<th>عملیات · زمان</th><th class="col-opt">عامل</th><th>اجرا / منبع</th><th class="col-opt">ارجاع‌ها</th><th>نتیجه</th><th class="col-opt">همبستگی</th><th class="col-actions"><span class="sr">اقدام</span></th>', rows) + h.tfoot(M.log.length, M.log.length) +
        '<p class="ind-note">' + ic('info') + ' لاگ و تاریخچه گزارش فعلی ممکن است کامل و تغییرناپذیر نباشند (آخرین گزارش بازنویسی می‌شود). نبود ردیف در این فهرست اثبات نمی‌کند که عملیاتی انجام نشده است.</p>';
    } else if (q === 'health') {
      var H = M.health;
      body = '<div class="q-body">' + X.sec('تناقض سلامت', X.cov('undef', 'علت نامعلوم'), '<div class="hc-grid"><div class="hc-cell warn"><span class="own-l">' + ic('alert') + 'هشدار ذخیره‌شده</span><b>' + esc(H.msg) + '</b><span class="muted">بررسی ' + esc(H.checkedAt) + ' · ' + esc(H.source) + '</span></div><div class="hc-vs" aria-hidden="true">' + ic('swap') + '</div><div class="hc-cell ok"><span class="own-l">' + ic('checkCircle') + 'خواندن زنده</span><b>گزارش خوانا است</b><span class="muted">همین الان</span></div></div>') +
        X.sec('چه کاری مجاز است؟', '', X.checks([['ok', 'نمایش زمان، منبع و خطای بررسی', 'برای تصمیم آگاهانه'], ['no', 'ترمیم یا بازسازی کور', 'علت نامعلوم است؛ فقط پس از تشخیص تازه و در بخش نگهداری'], ['no', 'نمایش «ورود ۰»', 'ناسالم بودن سلامت، صفر مشروع نمی‌سازد']])) + '</div>';
    } else {
      var map = [['mis-overview', 'امروز — صف کار', 'today', 'ساده‌سازی؛ شمارنده‌های هم‌پوشان افراز نیست'], ['mis-import', 'ورود و کیفیت داده', 'import', 'آمادگی جدا از سلامت ذخیره‌شده'], ['آخرین نتیجه ورود', 'اجراها (نوع اجرا مشخص)', 'import', 'تغییر نام: عملیات تخصیص «ورود» نیست'], ['mis-batches-list', 'مسئولیت و تحویل › تخصیص منبع', 'custody', 'برگشت مشروط به F01'], ['mis-pool', 'مسئولیت و تحویل › استخر', 'custody', 'استخر ≠ لید زنده'], ['mis-preview', 'ورود › کیفیت منبع', 'import', 'ادغام با گزارش/کیفیت'], ['mis-batches', 'ردیف‌های منبع + ورود › فایل‌ها', 'cases', 'چرخه دسته زیر ردیف‌ها'], ['حذف / پاک‌سازی', 'نگهداری و پیشرفته', 'maint', 'جدا از کار روزانه؛ قابلیت حذف نشده'], ['mis-quick', 'تحویل سریع (مشروط)', 'quick', 'نگه داشته شد؛ مسیر استثنایی'], ['mis-distribution + mis-live', 'برنامه‌ریزی (مشروط)', 'plan', 'برنامه ≠ تحویل واقعی'], ['mis-logs', 'عیب‌یابی › لاگ عملیات', 'diag', 'خالی بودن حذف نیست'], ['mis-report', 'گزارش‌ها و تطبیق', 'rec', 'ردیف ≠ پرونده سراسری']];
      body = table('نقشه مقصدهای قدیمی به ساختار جدید', 'هر ردیف یک مقصد فعلی و محل آن در ساختار جدید؛ مسیرها، شناسه‌ها و وضعیت‌ها تغییر نکرده‌اند.', '<th>مقصد فعلی</th><th>در ساختار جدید</th><th class="col-opt">تصمیم</th><th class="col-actions"><span class="sr">رفتن</span></th>', map.map(function (m) { return '<tr><td class="mono">' + esc(m[0]) + '</td><td>' + esc(m[1]) + '</td><td class="col-opt muted wrap">' + esc(m[3]) + '</td><td class="col-actions"><button type="button" class="btn btn-soft btn-sm" data-act="goto:' + m[2] + '">رفتن</button></td></tr>'; }).join(''), 'no-cursor');
    }
    return h.pageHead({ title: 'عیب‌یابی', sub: 'لاگ عملیات، سلامت و نقشه مقصدها · فقط خواندنی · بدون دکمه ترمیم', fresh: X.freshPart('عیب‌یابی'), scope: X.zone('normal', 'تشخیص خواندنی') }) + X.banners('diag') + '<section class="panel main" aria-labelledby="dg-h">' + ph('shield', 'عیب‌یابی', 'dg-h', '', '') + qs + body + '</section>';
  };
})();
