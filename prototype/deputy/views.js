/* Sales Deputy — role layer, part 2: read/analysis views (leadership overview, hierarchy & performance, exceptions, scoped invoice context,
   archive/extra-number monitor, diagnostics, reports, wallet). Presentation of Product-Spec-approved facts; undefined or unreceived metrics
   render as a truthful state («تعریف نشده», «پوشش ناقص», «نیازمند تطبیق»), never as 0. Nothing here is a command. */
(function () {
  'use strict';
  var X = window.DEPX, C = X.C, M = X.M, h = X.h, st = X.st, V = X.V;
  var esc = h.esc, fa = h.fa, num = h.num, money = h.money, ic = h.ic, pill = h.pill, hint = h.hint;
  var sim = X.sim, seller = X.seller, team = X.team, mgr = X.mgr, senior = X.senior;
  var na = function (t) { return '<span class="na tip" tabindex="0" data-tip="' + esc(t || 'داده در دسترس نیست') + '">—</span>'; };
  var n0 = function (v) { return v ? fa(v) : '<span class="muted">۰</span>'; };
  var failedM = function (mid) { return X.mgrCov(mid) === 'failed'; };
  var branchCount = function () { return M.managers.length; };

  function topExc(mid) {
    var l = X.branchExc(mid); if (!l.length) return null;
    var pri = ['consumed', 'invoice', 'mgrop', 'distconf', 'hr', 'inactive', 'lineage', 'stale', 'f04', 'scope'];
    l.sort(function (a, b) { return pri.indexOf(a.cls) - pri.indexOf(b.cls); });
    var e = l[0]; return { t: M.excClasses[e.cls].label, a: e.owner ? e.owner.split(' · ')[0] : 'مسئول نامشخص (UNKNOWN)', unk: !e.owner, n: l.length, id: e.id };
  }
  function nextOf(mid) {
    var l = X.branchExc(mid); if (!l.length) return '<span class="muted">—</span>';
    return '<span class="actor-what">' + esc(l[0].next) + '</span>';
  }

  /* ================= Leadership Overview ================= */
  V.ov = function () {
    var A = X.attention(), MT = X.metricTrust(), bad = sim() === 'branchfail' ? 'unavailable' : sim() === 'stale' ? 'stale' : null;
    var needAttn = M.managers.filter(function (m) { return X.branchExc(m.id).length || X.mgrCov(m.id) !== 'ok'; }).length;
    var untrusted = MT.filter(function (m) { return m.cov !== 'ok'; }).length;
    var unk = X.exceptions().filter(function (e) { return !e.owner; }).length;
    var kpis = [
      { label: 'شاخه‌های نیازمند توجه', value: fa(needAttn) + '<small class="kstate">از ' + fa(branchCount()) + '</small>', color: 'orange', keep: true, meaning: 'شاخه‌ای که استثنای باز دارد یا داده‌اش کامل/به‌روز نیست؛ رتبه یا «ضعیف‌ترین» شاخه نیست', basis: 'وضعیت در لحظه · محدوده مجاز شما' },
      { label: 'دسته استثنای باز', value: fa(A.length), color: 'amber', keep: true, filter: 'exc:all', meaning: 'هر دسته مالک، مسئول بعدی و علامت حل دارد؛ شمار آیتم نیست', basis: 'وضعیت در لحظه' },
      { label: 'استثنای بدون مسئول مشخص', value: fa(unk), tone: unk ? 'warn' : '', color: 'slate', filter: 'exc:unk', meaning: 'مسئول UNKNOWN است؛ رتبه معاون مالک پیش‌فرض نیست', basis: 'وضعیت در لحظه' },
      { label: 'شاخص‌های قابل‌اتکا', value: fa(MT.length - untrusted) + '<small class="kstate">از ' + fa(MT.length) + '</small>', color: 'violet', state: bad, filter: 'diag:f02', meaning: 'شاخصی که تعریف، پوشش و تازگی‌اش کامل است؛ بقیه با وضعیت صادق نمایش داده می‌شوند', basis: 'تعریف + پوشش' }
    ];
    var attn = '<section class="panel attn-panel" aria-labelledby="attn-h"><div class="sec-h ph"><h2 id="attn-h">' + ic('inbox') + 'استثناهای نیازمند هماهنگی</h2><span class="aside">' + fa(A.length) + ' دسته</span></div>' +
      (A.length ? '<ul class="attn">' + A.map(function (a) {
        return '<li><button type="button" class="attn-row" data-attn="' + a.id + '" style="--c:var(--' + a.tone + '-dot)"><span class="attn-ico t-' + a.tone + '">' + ic(a.icon) + '</span><span class="attn-txt"><b>' + esc(a.label) + '</b><span><span class="attn-own">مسئول: ' + esc(a.owner) + '</span>' + esc(a.state) + '</span></span><span class="attn-n" aria-label="' + fa(a.n) + ' مورد">' + fa(a.n) + '</span>' + ic('arrowL') + '</button></li>';
      }).join('') + '</ul>' : h.stateBlock('empty', 'استثنای ثبت‌شده‌ای در محدوده کامل شما نیست', 'این نتیجه فقط برای محدوده و منابع دریافت‌شده است و به معنی «همه چیز آزاد یا حل‌شده است» نیست.')) +
      '<div class="attn-foot">' + ic('info') + '<span>شما مسئول هماهنگی‌اید، نه اپراتور صف مدیران. «حل‌شده» یعنی مسئول و وضعیت روشن شد، نه لزوماً انتقال پرونده یا آزادشدن مالی. دستور ارجاع/تشدید رسمی تعریف نشده (SD-G07).</span></div></section>';

    var maxW = Math.max.apply(null, M.managers.map(function (m) { return X.mgrLoad(m.id); }));
    var rows = M.managers.map(function (m) {
      var cov = X.mgrCov(m.id), ss = X.sellersOfMgr(m.id), act = ss.filter(function (s) { return s.active; }).length, w = X.mgrLoad(m.id), ex = topExc(m.id);
      var inactive = X.seniorsOfMgr(m.id).filter(function (u) { return !u.active; }).length + M.teams.filter(function (t) { return X.mgrOfTeam(t.id) === m.id && !t.active; }).length;
      var nm = X.who('مدیر ' + m.name, X.rel('manager') + ' ' + X.cov(cov));
      if (cov === 'failed') return '<tr data-row="mgr:' + m.id + '" tabindex="-1" class="failed"><td>' + nm + '</td><td colspan="3"><div class="row-state">' + X.cov('failed') + '<span>دریافت آمار این شاخه ناموفق بود؛ مقدار صفر فرض نشده است.</span></div></td><td><span class="muted">—</span></td><td>' + X.cov('failed') + '</td><td class="col-actions"><button type="button" class="btn btn-soft btn-sm" data-act="open-mgr:' + m.id + '">مشاهده</button></td></tr>';
      return '<tr data-row="mgr:' + m.id + '" tabindex="-1"><td>' + nm + '</td>' +
        '<td class="n">' + (cov === 'partial' ? '<span class="partial-n tip" tabindex="0" data-tip="سلسله‌مراتب یا منبع ناقص دریافت شده؛ عدد نهایی نیست">' + fa(act) + '</span>' : fa(act)) + '<span class="of"> از ' + fa(ss.length) + '</span></td>' +
        '<td class="wl"><span class="wl-n">' + fa(w) + '</span><span class="wl-bar" aria-hidden="true"><i style="width:' + Math.round(w / maxW * 100) + '%"></i></span></td>' +
        '<td class="col-opt">' + (inactive ? pill(fa(inactive) + ' مسئول غیرفعال', 'amber', 'user') : '<span class="muted">محدودیتی ثبت نشده</span>') + '</td>' +
        '<td>' + (ex ? '<div class="exc-cell"><span class="pill t-' + (ex.unk ? 'slate' : 'orange') + '">' + esc(ex.t) + (ex.n > 1 ? ' +' + fa(ex.n - 1) : '') + '</span><span class="cell-sub">مسئول: ' + esc(ex.a) + '</span></div>' : '<span class="muted">—</span>') + '</td>' +
        '<td>' + nextOf(m.id) + '</td><td class="col-actions"><button type="button" class="btn btn-soft btn-sm" data-act="open-mgr:' + m.id + '">مشاهده</button></td></tr>';
    }).join('');
    var th = '<thead><tr><th>شاخه مدیر</th><th class="n">فروشنده فعال ' + hint('فروشندگان زیر این شاخه؛ تعداد فروشنده فعال نماینده‌ای از ظرفیت است نه ظرفیت کامل. دیدن فروشنده غیرمستقیم اجازه نوشتن نمی‌سازد.', true) + '</th><th>پرونده نزد فروشندگان ' + hint('مالک فعلی پرونده‌ها؛ این ستون رتبه یا ارزیابی عملکرد نیست و جمع شاخه‌ها با هم مقایسه نمی‌شوند.', true) + '</th><th class="col-opt">محدودیت توزیع ' + hint('گیرنده غیرفعال در این شاخه؛ مسئول جایگزین خودکار تعیین نمی‌شود.', true) + '</th><th>بزرگ‌ترین استثنا و مسئول آن</th><th>اقدام بعدی با</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead>';
    var cap = '<section class="panel main cap-panel" aria-labelledby="cap-h"><div class="sec-h ph"><h2 id="cap-h">' + ic('users') + 'شاخه‌های مدیر در محدوده شما</h2>' + X.basis('current') + X.basis('snap') + X.grain('شاخه مدیر') + '<span class="aside">به ترتیب نام · رتبه‌بندی نیست</span></div>' +
      '<div class="tbl-wrap"><table class="tbl" aria-label="شاخه‌های مدیر"><caption class="sr">هر ردیف یک شاخه مدیر در محدوده شما با مسئول بزرگ‌ترین استثنا؛ نشان پوشش در ستون نخست وضعیت تازگی و کامل‌بودن داده را می‌گوید.</caption>' + th + '<tbody>' + rows + '</tbody></table></div></section>';

    var mrows = MT.map(function (m) {
      var val = m.value == null ? na(m.covLabel) : '<b class="mt-v">' + m.value + '</b>';
      return '<tr><td><b>' + esc(m.name) + '</b><div class="cell-sub">' + X.basis(m.basis) + '</div></td><td class="n">' + val + '</td><td>' + X.cov(m.cov, m.covLabel) + '</td><td class="mt-why">' + esc(m.why) + '</td></tr>';
    }).join('');
    var trust = '<section class="panel main" aria-labelledby="mt-h"><div class="sec-h ph"><h2 id="mt-h">' + ic('shield') + 'اعتماد به شاخص‌ها پیش از تصمیم</h2><span class="aside">عدد بدون پوشش و تعریف، تصمیم‌ساز نیست</span><button type="button" class="btn btn-sm btn-soft" data-act="open-metrics">' + ic('info') + 'تعریف شاخص‌ها</button></div>' +
      '<div class="tbl-wrap"><table class="tbl no-cursor stackable" aria-label="اعتماد به شاخص‌ها"><thead><tr><th>شاخص و مبنای زمانی</th><th class="n">مقدار</th><th>اعتماد</th><th>چرا</th></tr></thead><tbody>' + mrows + '</tbody></table></div>' +
      '<div class="bulk-note">' + ic('info') + '<span>هیچ رتبه‌بندی، هدف، نرخ تبدیل یا امتیازی نمایش داده نمی‌شود. F02 و F03 رفع‌شده فرض نشده‌اند؛ برای مقایسه هم‌گروه به <button type="button" class="linklike" data-act="goto:diag">عیب‌یابی شاخص و منشأ</button> بروید.</span></div></section>';

    var pool = '<section class="panel pool-strip"><div class="sec-h ph"><h2>' + ic('send') + 'تخصیص مستقیم (مشروط)</h2><span class="scope-badge">' + ic('user') + 'موجودی شخصی شما · جدا از بار شاخه‌ها</span></div><div class="pool-body"><div><b class="pool-n">' + fa(X.poolList().length) + '</b> پرونده <span class="muted">· ' + fa(X.poolOk().length) + ' قابل تخصیص · ' + fa(X.poolList().length - X.poolOk().length) + ' نیازمند بررسی · سیاست عبور از مدیر مصوب نیست (SD-G05)</span></div><button type="button" class="btn btn-soft" data-act="goto:alloc">' + ic('send') + 'رفتن به تخصیص مشروط</button></div></section>';
    var scope = '<section class="panel ind-panel"><div class="sec-h ph"><h2>' + ic('user') + 'هویت حساب و محدوده</h2></div><p class="ind-note">حساب: <b>' + esc(M.user.name) + '</b> · شناسه حساب <span class="mono">' + esc(M.user.id) + '</span> — شناسه فقط برای تشخیص حساب است، نه شاخص. محدوده: ' + fa(branchCount()) + ' شاخه مدیر · ' + fa(M.sellers.length) + ' فروشنده — <b>این کل سازمان نیست</b>. ' + X.rel('manager') + ' ' + X.rel('indirect') + ' دیدن زیرمجموعه به معنی اجازه نوشتن یا تصمیم‌گیری مالی/HR/MIS نیست؛ سلسله‌مراتب مجوز نمی‌سازد.</p></section>';
    return h.pageHead({ title: 'نمای رهبری', sub: 'کدام شاخه توجه می‌خواهد؟ کدام داده ناقص است؟ مسئول گام بعدی کیست؟ · بدون رتبه، امتیاز، هدف یا نرخ تبدیل', kpis: kpis, fresh: X.freshPart('آمار شاخه‌ها'), scope: 'محدوده: ' + fa(branchCount()) + ' شاخه مدیر (نه کل سازمان)' }) + X.banners() +
      '<div class="team-grid">' + attn + '<div class="team-main">' + cap + trust + pool + scope + '</div></div>';
  };

  /* ================= Hierarchy & Performance (one hierarchy-aware explorer) ================= */
  var MODES = [['mgr', 'مدیر', 'users'], ['senior', 'سرپرست ارشد', 'compass'], ['sup', 'سرپرست', 'briefcase'], ['seller', 'فروشنده', 'user'], ['inv', 'فاکتور', 'receipt'], ['case', 'بار پرونده', 'inbox']];
  function inScope(key) { var p = st.pscope; if (!p) return true; return X.chainOf(String(key)).map(function (c) { return String(c[0]); }).indexOf(String(p)) > -1; }
  V.perf = function () {
    var pm = st.pm, basisBtn = pm === 'mgr' || pm === 'senior' || pm === 'inv';
    var dim = '<div class="dim-switch" role="group" aria-label="بعد تحلیل (واحد شمارش)">' + MODES.map(function (m) { return '<button type="button" data-pm="' + m[0] + '" aria-pressed="' + (pm === m[0]) + '">' + ic(m[2]) + m[1] + '</button>'; }).join('') + '</div>';
    var tb = (pm === 'mgr' || pm === 'senior' || pm === 'sup') ? '<div class="seg" role="group" aria-label="مبنای زمانی"><button type="button" data-tbs="snap" aria-pressed="' + (st.tb === 'snap') + '">وضعیت در لحظه</button><button type="button" data-tbs="event" aria-pressed="' + (st.tb === 'event') + '">رویداد در بازه</button><button type="button" disabled aria-disabled="true" class="tip" data-tip="وضعیت تاریخی (As-Of) فقط با سابقه معتبر ممکن است؛ داده امروز اثبات آن نیست (F09) و در این نمونه سابقه کافی ثبت نشده">وضعیت تاریخی</button></div>' : '';
    var basis = basisBtn ? '<div class="seg" role="group" aria-label="مبنای انتساب"><button type="button" data-pb="current" aria-pressed="' + (st.pb === 'current') + '">ساختار فعلی</button><button type="button" data-pb="hist" aria-pressed="' + (st.pb === 'hist') + '">انتساب تاریخی</button></div>' : '<span class="basis-fixed">' + (pm === 'seller' ? X.basis('event') + '<span class="muted">آمار فروشنده به خود او نسبت داده می‌شود و به والد امروز وابسته نیست.</span>' : X.basis('current')) + '</span>';
    var body = ({ mgr: perfMgr, senior: perfSenior, sup: perfSup, seller: perfSeller, inv: perfInv, 'case': perfCase })[pm]();
    var ctl = '<section class="panel explorer-ctl" aria-label="کنترل‌های کاوشگر">' + dim + tb + basis + '<button type="button" class="btn btn-sm btn-soft" data-act="open-metrics">' + ic('info') + 'تعریف شاخص‌ها</button></section>';
    var four = '<div class="four-concepts" role="note">' + ic('shield') + '<span><b>چهار مفهوم جدا:</b> ' + X.basis('current') + ' برای بار و پیمایش · ' + X.basis('hist') + ' برای گذشته · ' + X.basis('event') + ' = انجام‌دهنده واقعی · ' + X.basis('credit') + ' = قوانین مالی. والد امروز اعتبار گذشته را تعیین نمی‌کند و عدد سطح‌ها با هم جمع نمی‌شود.</span></div>';
    return h.pageHead({ title: 'سلسله‌مراتب و عملکرد', sub: 'مدیر ← سرپرست ارشد ← سرپرست ← فروشنده ← پرونده/فاکتور · هر حالت یک واحد شمارش دارد · بدون رتبه‌بندی، هدف فروش یا امتیاز', fresh: X.freshPart('داده'), scope: 'مبنا: ' + (basisBtn ? (st.pb === 'hist' ? 'انتساب تاریخی' : 'ساختار فعلی') : 'ثابت برای این حالت') }) + X.banners() + ctl +
      '<section class="panel main"><div class="ex-bar">' + X.crumb() + '<span class="grow"></span>' + X.grain(body.grain) + (basisBtn ? X.basis(st.pb === 'hist' ? 'hist' : 'current') : '') + '</div>' + four + (body.note ? '<div class="note info inset">' + ic('info') + '<span>' + body.note + '</span></div>' : '') + body.html + '</section>';
  };
  var TH = function (label, basisKey, tip) { return '<th class="n">' + label + ' <span class="tb-tag tb-' + basisKey + '">' + (basisKey === 'snap' ? 'لحظه' : 'رویداد') + '</span>' + (tip ? hint(tip, true) : '') + '</th>'; };
  function kpiCols(f, cov, dim) {
    var v = function (x) { return cov === 'failed' ? na('دریافت نشد') : x; };
    if (st.tb === 'snap') return [v(dim.cases), v('<span class="recon-n tip" tabindex="0" data-tip="از ردیف‌های فاکتور شمرده شده؛ با شاخص نمای کلی هم‌خوان نیست (F02) و عدد نهایی تأیید نشده است">' + fa(f.openPre) + '</span>'), v(dim.legacy)];
    return [v(fa(f.created)), v(fa(f.created)), v(fa(f.completed)), v(cov === 'recon' ? (f.collected ? '<span class="recon-n tip" tabindex="0" data-tip="مغایرت بین منبع پرداخت و وضعیت فاکتور؛ مبلغ نهایی نیست">' + money(f.collected) + '</span>' : na('نیازمند تطبیق')) : money(f.collected))];
  }
  function kpiHead() {
    return st.tb === 'snap' ? TH('کل پرونده‌ها نزد فروشندگان', 'snap', 'فقط پرونده‌های دارای مسئول فعلی فروشنده؛ پوشش کامل منابع اثبات نشده (F03) و با شماره موبایل ادغام نشده.') + TH('پیش‌فاکتور باز', 'snap', 'از ردیف‌های فاکتور؛ شاخص نمای کلی با آن نمی‌خواند (F02).') + TH('فقط لیدهای قدیمی', 'snap', 'شمارش جدول قدیمی؛ برچسب «کل لیدها» نیست.')
      : TH('فاکتور صادرشده', 'event') + TH('پیش‌فاکتور صادرشده', 'event', 'کلید یکتای صدور؛ نبود سابقه صدور ≠ ۰.') + TH('فروش تکمیل‌شده', 'event', 'مرز تکمیل (OPD-06) و انتساب (OPD-07) باز است.') + TH('وصول تأییدشده', 'event', 'جمع مبلغ مراحل معتبر؛ مغایرت‌دار شمرده نمی‌شود.');
  }
  function noteBasis(hist) { return (st.tb === 'snap' ? '<b>وضعیت در لحظه:</b> عکس فوری از بار و پیش‌فاکتورهای باز؛ تاریخچه را بازسازی نمی‌کند.' : '<b>رویداد در بازه:</b> رویدادها بر اساس زمان وقوع خودشان؛ زمان ایجاد/تخصیص جایگزین آن نیست.') + (hist ? ' <b>انتساب تاریخی:</b> هر فاکتور به شاخه‌ای که هنگام صدور ثبت شده نسبت داده می‌شود؛ سابقه‌نداشته «ناشناخته» می‌ماند.' : ' <b>ساختار فعلی:</b> فروشنده‌ای که اخیراً جابه‌جا شده زیر والد فعلی‌اش دیده می‌شود؛ اعتبار گذشته را با «انتساب تاریخی» ببینید.'); }
  function perfMgr() {
    var hist = st.pb === 'hist', list = M.managers.filter(function (m) { return inScope(m.id); });
    var rows = list.map(function (m) {
      var f = X.mgr_f(m.id, hist ? 'hist' : 'current'), cov = X.mgrCov(m.id), ss = X.sellersOfMgr(m.id), cur = hist ? na('فقط در ساختار فعلی معنا دارد') : null;
      var cells = kpiCols(f, cov, { cases: cur || fa(X.mgrLoad(m.id)), legacy: cur || fa(X.sum(ss, 'legacy')) }).map(function (c) { return '<td class="n">' + c + '</td>'; }).join('');
      return '<tr data-row="mgr:' + m.id + '" tabindex="-1"><td>' + X.who('مدیر ' + m.name, X.rel('manager')) + '</td><td class="n">' + (cur || fa(ss.filter(function (s) { return s.active; }).length) + '<span class="of"> از ' + fa(ss.length) + '</span>') + '</td>' + cells + '<td>' + X.cov(cov) + '</td><td class="col-actions"><button type="button" class="btn btn-sm" data-act="drill:' + m.id + '">سرپرستان ارشد</button></td></tr>';
    }).join('');
    var unk = '';
    if (hist && !st.pscope) { var uf = X.facts(function (i) { return i.teamAtIssue == null; }); var uc = st.tb === 'snap' ? '<td class="n">' + na('') + '</td><td class="n">' + fa(uf.openPre) + '</td><td class="n">' + na('') + '</td>' : '<td class="n">' + fa(uf.created) + '</td><td class="n">' + na('سابقه صدور ثبت نشده') + '</td><td class="n">' + fa(uf.completed) + '</td><td class="n">' + money(uf.collected) + '</td>'; unk = '<tr class="unk-row"><td>' + X.who('شاخه ناشناخته', '<span class="scope-badge view">' + ic('question') + 'غیرقابل بازیابی</span>') + '</td><td class="n">' + na('') + '</td>' + uc + '<td>' + X.cov('undef', 'ناشناخته') + '</td><td class="col-actions"></td></tr>'; }
    var all = X.allFacts();
    var tot = st.pscope ? '' : '<tfoot><tr class="tot"><th scope="row">جمع یکتا ' + hint('هر فاکتور فقط یک‌بار شمرده می‌شود؛ جمع از زیرجمع‌های هم‌پوشان ساخته نشده است. شاخه‌های دریافت‌نشده در جمع نیستند.', true) + '</th><td></td>' + (st.tb === 'snap' ? '<td></td><td class="n">' + fa(all.openPre) + '</td><td></td>' : '<td class="n">' + fa(all.created) + '</td><td></td><td class="n">' + fa(all.completed) + '</td><td class="n">' + money(all.collected) + '</td>') + '<td colspan="2"></td></tr></tfoot>';
    return { grain: st.tb === 'snap' ? 'پرونده/پیش‌فاکتور (لحظه) به تفکیک مدیر' : 'فاکتور (رویداد) به تفکیک مدیر', note: noteBasis(hist) + ' مقادیر هر ستون با تعریف رسمی در «تعریف شاخص‌ها» تطبیق دارند؛ cohort تجاری باز است.',
      html: '<div class="tbl-wrap"><table class="tbl" aria-label="عملکرد به تفکیک مدیر"><thead><tr><th>مدیر شاخه</th><th class="n">فروشنده فعال</th>' + kpiHead() + '<th>اعتماد به داده</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + unk + '</tbody>' + tot + '</table></div>' };
  }
  function perfSenior() {
    var hist = st.pb === 'hist', cur = hist ? na('فقط در ساختار فعلی معنا دارد') : null;
    var rows = [];
    M.managers.forEach(function (m) {
      if (!inScope(m.id)) return;
      X.seniorsOfMgr(m.id).forEach(function (u) {
        if (!inScope(u.id)) return;
        var f = X.senior_f(u.id, hist ? 'hist' : 'current'), cov = X.seniorCov(u.id), ss = X.sellersOfSenior(u.id);
        var cells = kpiCols(f, cov, { cases: cur || fa(X.seniorLoad(u.id)), legacy: cur || fa(X.sum(ss, 'legacy')) }).map(function (c) { return '<td class="n">' + c + '</td>'; }).join('');
        rows.push('<tr data-row="senior:' + u.id + '" tabindex="-1"' + (u.active ? '' : ' class="inactive"') + '><td>' + X.who('سرپرست ارشد ' + u.name, X.rel('senior') + (u.active ? '' : ' ' + pill('غیرفعال', 'slate', 'ban'))) + (u.active ? '' : '<div class="cell-sub">' + esc(u.inactiveNote) + ' · مسئول جایگزین ثبت نشده</div>') + '</td><td><span class="muted">مدیر ' + esc(m.name) + '</span></td><td class="n">' + (cur || fa(ss.filter(function (s) { return s.active; }).length) + '<span class="of"> از ' + fa(ss.length) + '</span>') + '</td>' + cells + '<td>' + X.cov(cov) + '</td><td class="col-actions"><button type="button" class="btn btn-sm" data-act="drill:' + u.id + '">سرپرستان</button></td></tr>');
      });
      X.directTeams(m.id).filter(function (t) { return inScope(t.id); }).forEach(function (t) {
        var f = X.team_f(t.id, hist ? 'hist' : 'current'), cov = X.teamCov(t.id), ss = X.sellersOfTeam(t.id);
        var cells = kpiCols(f, cov, { cases: cur || fa(X.teamLoad(t.id)), legacy: cur || fa(X.sum(ss, 'legacy')) }).map(function (c) { return '<td class="n">' + c + '</td>'; }).join('');
        rows.push('<tr data-row="team:' + t.id + '" tabindex="-1" class="direct-row"><td>' + X.who('سرپرست ' + t.sup, X.rel('dsup') + ' <span class="cell-sub">بدون سرپرست ارشد میانی</span>') + '</td><td><span class="muted">مدیر ' + esc(m.name) + '</span></td><td class="n">' + (cur || fa(ss.filter(function (s) { return s.active; }).length) + '<span class="of"> از ' + fa(ss.length) + '</span>') + '</td>' + cells + '<td>' + X.cov(cov) + '</td><td class="col-actions"><button type="button" class="btn btn-sm" data-act="drill:' + t.id + '">فروشندگان</button></td></tr>');
      });
    });
    return { grain: st.tb === 'snap' ? 'پرونده/پیش‌فاکتور (لحظه) به تفکیک سرپرست ارشد' : 'فاکتور (رویداد) به تفکیک سرپرست ارشد', note: noteBasis(hist) + ' سرپرستانی که مستقیم زیر مدیر هستند با نشان جدا آمده‌اند؛ سطح میانی حدس زده نمی‌شود.',
      html: rows.length ? '<div class="tbl-wrap"><table class="tbl" aria-label="عملکرد به تفکیک سرپرست ارشد"><thead><tr><th>سرپرست ارشد</th><th>مدیر (والد فعلی)</th><th class="n">فروشنده فعال</th>' + kpiHead() + '<th>اعتماد به داده</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows.join('') + '</tbody></table></div>' : h.stateBlock('noresult', 'سرپرست ارشدی در این محدوده نیست', 'محدوده را به «همه شاخه‌ها» برگردانید.', '<button type="button" class="btn btn-soft" data-act="pscope:all">همه شاخه‌ها</button>') };
  }
  function perfSup() {
    var list = M.teams.filter(function (t) { return inScope(t.id); });
    var rows = list.map(function (t) {
      var f = X.team_f(t.id, 'current'), ss = X.sellersOfTeam(t.id), cov = X.teamCov(t.id), u = X.seniorOfTeam(t.id);
      var cells = kpiCols(f, cov, { cases: fa(X.teamLoad(t.id)), legacy: fa(X.sum(ss, 'legacy')) }).map(function (c) { return '<td class="n">' + c + '</td>'; }).join('');
      var par = u ? esc('سرپرست ارشد ' + senior(u).name) : '<span class="muted">مستقیم زیر مدیر</span>';
      return '<tr data-row="team:' + t.id + '" tabindex="-1"' + (t.active ? '' : ' class="inactive"') + '><td>' + X.who(t.sup, t.active ? X.rel(u ? 'sup' : 'dsup') : pill('غیرفعال', 'slate', 'ban')) + (t.active ? '' : '<div class="cell-sub">' + esc(t.inactiveNote) + ' · مسئول جایگزین ثبت نشده</div>') + '</td><td>' + esc(X.mgrName(X.mgrOfTeam(t.id))) + '<div class="cell-sub">' + par + '</div></td><td class="n">' + fa(ss.filter(function (s) { return s.active; }).length) + '<span class="of"> از ' + fa(ss.length) + '</span></td>' + cells + '<td>' + X.cov(cov) + '</td><td class="col-actions"><button type="button" class="btn btn-sm" data-act="drill:' + t.id + '">فروشندگان</button></td></tr>';
    }).join('');
    return { grain: st.tb === 'snap' ? 'پرونده/پیش‌فاکتور (لحظه) به تفکیک سرپرست' : 'فاکتور (رویداد) به تفکیک سرپرست', note: 'سرپرستان با زنجیره والد فعلی‌شان نمایش داده می‌شوند. این نما <b>پاسخ‌گویی و بار</b> را نشان می‌دهد، نه امتیاز عملکرد؛ بدون رتبه و هدف. عدد همین نما با عدد مدیر جمع نمی‌شود.',
      html: rows ? '<div class="tbl-wrap"><table class="tbl" aria-label="عملکرد به تفکیک سرپرست"><thead><tr><th>سرپرست</th><th>والد فعلی</th><th class="n">فروشنده فعال</th>' + kpiHead() + '<th>اعتماد به داده</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div>' : h.stateBlock('noresult', 'سرپرستی در این محدوده نیست', '', '<button type="button" class="btn btn-soft" data-act="pscope:all">همه شاخه‌ها</button>') };
  }
  function perfSeller() {
    var list = M.sellers.filter(function (s) { return inScope(s.id); });
    var rows = list.map(function (s) {
      var f = X.seller_f(s.id), cov = X.teamCov(s.team), bad = cov === 'failed', ch = X.chainOfSeller(s);
      var chain = X.mgrName(ch.mgr) + (ch.senior ? ' › ' + X.seniorName(ch.senior) : '') + (ch.team ? ' › ' + X.teamName(ch.team) : ' › مستقیم');
      var teamCell = X.rel('seller') + '<div class="cell-sub">' + esc(chain) + '</div>' + (s.moved ? '<div class="cell-sub moved">' + ic('swap') + 'منتقل‌شده از ' + esc(X.teamName(s.moved.from)) + ' · ' + esc(s.moved.when) + '</div>' : '');
      var legacyCell = cov === 'partial' ? '<span class="partial-n tip" tabindex="0" data-tip="پوشش ناقص: عدد نهایی نیست">' + fa(s.legacy) + '</span>' : fa(s.legacy);
      var v = function (x) { return bad ? na('دریافت نشد') : x; };
      return '<tr data-row="seller:' + s.id + '" tabindex="-1"' + (s.active ? '' : ' class="inactive"') + '><td>' + X.who(s.name, s.active ? '' : '<span class="warn-n">غیرفعال</span>') + '</td><td>' + teamCell + '</td><td class="n">' + v(fa(s.open)) + '</td><td class="n">' + v(legacyCell) + '</td><td class="n">' + v(fa(f.created)) + '</td><td class="n">' + v(n0(f.openPre)) + '</td><td class="n">' + v(n0(f.completed)) + '</td><td class="n">' + v(f.created ? money(f.collected) : na('فاکتوری ثبت نشده')) + '</td><td class="col-actions"><button type="button" class="btn btn-sm" data-act="open-seller:' + s.id + '">جزئیات</button></td></tr>';
    }).join('');
    return { grain: 'فروشنده', note: 'ستون «فقط لیدهای قدیمی» عدد <b>کل پرونده‌ها نیست</b>. «۰» فقط یعنی محاسبه کامل شده و مقدار صفر است؛ داده دریافت‌نشده «—» می‌ماند.',
      html: list.length ? '<div class="tbl-wrap"><table class="tbl" aria-label="عملکرد به تفکیک فروشنده"><thead><tr><th>فروشنده</th><th>رابطه و زنجیره فعلی</th><th class="n">پرونده باز</th><th class="n">فقط لیدهای قدیمی</th><th class="n">فاکتور صادرشده</th><th class="n">پیش‌فاکتور باز</th><th class="n">فروش تکمیل‌شده</th><th class="n">وصول تأییدشده</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div>' : h.stateBlock('noresult', 'فروشنده‌ای در این محدوده نیست', 'محدوده را به «همه شاخه‌ها» برگردانید.', '<button type="button" class="btn btn-soft" data-act="pscope:all">همه شاخه‌ها</button>') };
  }
  function perfInv() {
    var hist = st.pb === 'hist';
    var list = M.invoices.filter(function (i) { return inScope(i.seller); });
    var rows = list.map(function (i) {
      var s = seller(i.seller), tn = X.teamIdOfInv(i, 'current'), th = i.teamAtIssue;
      var cur = X.mgrName(X.mgrOfTeam(tn)) + ' › ' + esc(X.teamName(tn)), differs = th != null && th !== tn;
      var thCell = th == null ? '<span class="pill t-slate">' + ic('question') + 'ناشناخته</span><div class="cell-sub">' + esc(i.teamNote || '') + '</div>' : esc(X.mgrName(X.mgrOfTeam(th))) + ' › ' + esc(X.teamName(th));
      return '<tr data-row="inv:' + i.code + '" tabindex="-1"><td><span class="mono case-id">' + fa(i.code) + '</span><div class="cell-sub">' + X.link(i.link) + '</div></td><td>' + X.who(s.name, '<span class="muted">عامل رویداد صدور</span>') + '</td><td class="col-opt"><span class="muted">طبق قوانین مالی؛ از گروه‌بندی نتیجه نمی‌شود</span></td>' +
        '<td class="' + (hist ? 'dimcol' : 'hl') + '">' + cur + '</td><td class="' + (hist ? 'hl' : 'dimcol') + '">' + thCell + (differs ? '<div class="cell-sub moved">' + ic('swap') + 'با ساختار امروز فرق دارد</div>' : '') + '</td>' +
        '<td>' + pill(X.INVS[i.inv].label, X.INVS[i.inv].tone, X.INVS[i.inv].icon) + '</td><td class="col-actions"><button type="button" class="btn btn-sm" data-act="open-inv:' + i.code + '">جزئیات</button></td></tr>';
    }).join('');
    return { grain: 'فاکتور (رویداد صدور)', note: 'چهار مفهوم در چهار ستون جدا: <b>فروشنده</b> = عامل رویداد، <b>مالک اعتبار</b> = قوانین مالی، <b>زنجیره فعلی</b> = ساختار امروز، <b>زنجیره هنگام صدور</b> = انتساب تاریخی. ساختار امروز جای تاریخی نمی‌نشیند.',
      html: '<div class="tbl-wrap"><table class="tbl" aria-label="فاکتورها به تفکیک زنجیره"><thead><tr><th>فاکتور و اطمینان اتصال</th><th>فروشنده (عامل رویداد)</th><th class="col-opt">مالک اعتبار</th><th>زنجیره فعلی <span class="sr">(ساختار فعلی)</span></th><th>زنجیره هنگام صدور <span class="sr">(انتساب تاریخی)</span></th><th>وضعیت فاکتور</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div>' };
  }
  function perfCase() {
    var rows = M.managers.filter(function (m) { return inScope(m.id); }).map(function (m) {
      var atM = M.held.filter(function (c) { return c.holder.kind === 'manager' && c.holder.mgr === m.id; }).length;
      var atS = M.held.filter(function (c) { return c.holder.kind === 'senior' && senior(c.holder.unit).mgr === m.id; }).length;
      var atT = M.held.filter(function (c) { return c.holder.kind === 'sup' && X.mgrOfTeam(c.holder.team) === m.id; }).length;
      var cov = X.mgrCov(m.id);
      return '<tr data-row="mgr:' + m.id + '" tabindex="-1"><td>' + X.who('مدیر ' + m.name, X.rel('manager')) + '</td><td class="n">' + n0(atM) + '</td><td class="n">' + n0(atS) + '</td><td class="n">' + n0(atT) + '</td><td class="n">' + (cov === 'failed' ? na('دریافت نشد') : fa(X.mgrLoad(m.id))) + '</td><td>' + X.cov(cov === 'ok' ? 'partial' : cov, cov === 'ok' ? 'فقط منبع شمرده‌شده (F03)' : null) + '</td></tr>';
    }).join('');
    var own = '<tr class="own-row"><td>' + X.who('پنل من (' + M.user.name + ')', '<span class="scope-badge">' + ic('user') + 'شخصی · موجودی قابل تخصیص مشروط</span>') + '</td><td class="n"><span class="muted">—</span></td><td class="n"><span class="muted">—</span></td><td class="n"><span class="muted">—</span></td><td class="n">' + fa(X.poolList().length) + '</td><td>' + X.cov('ok') + '</td></tr>';
    return { grain: 'پرونده یکتا نزد نگهدارنده فعلی', note: 'هر پرونده فقط یک‌بار و نزد <b>مسئول فعلی</b> شمرده می‌شود (نه مالک اولیه). بار فعلی، رویداد انتقال و فاکتور صادرشده سه گروه شمارش جدا هستند. شمارش فقط منابع تأییدشده را در بر می‌گیرد (F03)؛ «کل پرونده» نهایی نیست.',
      html: '<div class="tbl-wrap"><table class="tbl" aria-label="بار پرونده"><thead><tr><th>شاخه</th><th class="n">نزد خود مدیر</th><th class="n">نزد سرپرستان ارشد</th><th class="n">نزد سرپرستان</th><th class="n">نزد فروشندگان (جمع بار)</th><th>اعتماد به داده</th></tr></thead><tbody>' + rows + own + '</tbody></table></div>' };
  }

  /* ================= Exceptions (owner-aware coordination) ================= */
  V.exc = function () {
    var all = X.exceptions(), cls = M.excClasses;
    var qItems = [{ id: 'all', label: 'همه دسته‌ها', icon: 'inbox', tone: 'neutral', n: all.length, key: '1' }].concat(Object.keys(cls).filter(function (k) { return all.some(function (e) { return e.cls === k; }); }).map(function (k, i) { return { id: k, label: cls[k].label, icon: cls[k].icon, tone: 'neutral', n: all.filter(function (e) { return e.cls === k; }).length, key: i < 8 ? String(i + 2) : undefined }; }));
    var list = all.filter(function (e) { return (st.ec === 'all' || e.cls === st.ec) && (st.ec !== 'unk' || !e.owner); });
    if (st.ec === 'unk') list = all.filter(function (e) { return !e.owner; });
    if (st.em !== 'all') list = list.filter(function (e) { return st.em === 'cross' ? !e.mgr : e.mgr === st.em; });
    var unk = all.filter(function (e) { return !e.owner; }).length;
    var rows = list.map(function (e) {
      var c = cls[e.cls];
      return '<tr data-row="exc:' + e.id + '" tabindex="-1"><td><div class="exc-cls">' + pill(c.label, 'neutral', c.icon) + '</div><div class="exc-subj">' + esc(e.subject) + '</div><div class="cell-sub">شاخه: ' + (e.mgr ? esc(X.mgrName(e.mgr)) : 'بین چند شاخه / بدون شاخه') + ' · ' + (e.since ? 'از ' + esc(e.since) : 'زمان معتبر ثبت نشده') + '</div><div class="exc-why">' + esc(e.why) + '</div></td>' +
        '<td>' + (e.owner ? esc(e.owner) : pill('نامشخص (UNKNOWN)', 'slate', 'question')) + '<div class="cell-sub">' + esc(e.state) + '</div></td>' +
        '<td><span class="allowed">' + ic('checkCircle') + esc(e.may[0]) + '</span>' + (e.may.length > 1 ? '<div class="cell-sub">+ ' + fa(e.may.length - 1) + ' اقدام مجاز دیگر</div>' : '') + '</td>' +
        '<td><span class="actor-what">' + esc(e.next) + '</span><div class="cell-sub">علامت حل: ' + esc(e.signal) + '</div></td>' +
        '<td class="col-actions"><button type="button" class="btn btn-sm btn-soft" data-act="open-exc:' + e.id + '">بررسی</button></td></tr>';
    }).join('');
    var mseg = '<div class="seg" role="group" aria-label="فیلتر شاخه">' + [['all', 'همه شاخه‌ها']].concat(M.managers.map(function (m) { return [m.id, m.name]; })).concat([['cross', 'بین چند شاخه']]).map(function (b) { return '<button type="button" data-em="' + b[0] + '" aria-pressed="' + (st.em === b[0]) + '">' + esc(b[1]) + '</button>'; }).join('') + '</div>';
    return h.pageHead({ title: 'استثناها و هماهنگی', sub: 'برای هر استثنا: موضوع، مالک، وضعیت، دلیل توجه، اقدام مجاز شما، مسئول بعدی و علامت حل · شما مسئول هماهنگی‌اید، نه اپراتور صف مدیران', kpis: [
      { label: 'استثنای باز', value: fa(all.length), color: 'orange', keep: true, meaning: 'موارد جداگانه در همه دسته‌ها', basis: 'وضعیت در لحظه' },
      { label: 'دسته‌های درگیر', value: fa(X.attention().length), color: 'amber', meaning: 'از ' + fa(Object.keys(cls).length) + ' دسته پشتیبانی‌شده در Product Spec', basis: 'وضعیت در لحظه' },
      { label: 'مسئول نامشخص', value: fa(unk), tone: unk ? 'warn' : '', color: 'slate', filter: 'exc:unk', meaning: 'UNKNOWN می‌ماند؛ رتبه معاون مالک پیش‌فرض نیست', basis: 'وضعیت در لحظه' }
    ], fresh: X.freshPart('استثناها') }) + X.banners() +
      '<section class="panel main">' + h.queues(qItems, st.ec === 'unk' ? 'all' : st.ec, 'data-ec') + '<div class="toolbar">' + mseg + '<span class="grow"></span><span class="scope-badge view">' + ic('lock') + 'اقدام گروهی روی استثناها: نیازمند اعتبارسنجی</span></div>' +
      '<div class="note info inset">' + ic('info') + '<span><b>ارجاع رسمی وجود ندارد.</b> «مسئول بعدی» همان مالک مسیر موجود است، نه دستور، اعلان یا SLA جدید (SD-G07). «حل‌شده» ممکن است یعنی «اقدام برای شما مجاز نیست و مسئول مشخص شد».</span></div>' +
      (list.length ? '<div class="tbl-wrap"><table class="tbl exc-tbl" aria-label="استثناها"><thead><tr><th style="width:38%">دسته، موضوع و دلیل توجه</th><th style="width:20%">مالک (مسئول) و وضعیت</th><th style="width:19%">اقدام مجاز شما</th><th style="width:19%">مسئول بعدی و علامت حل</th><th class="col-actions" style="width:80px"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div>' + h.tfoot(list.length, all.length)
        : h.stateBlock('noresult', 'استثنایی با این فیلتر نیست', 'یک شاخه یا دسته دیگر را انتخاب کنید.', '<button type="button" class="btn btn-soft" data-ec="all">همه دسته‌ها</button>')) + '</section>';
  };

  /* ================= Scoped Invoice Context (READ only) ================= */
  var IQ = [
    { id: 'action', label: 'نیازمند اقدام فروش', icon: 'alert', tone: 'red', key: '1', f: function (i) { return i.review === 'rejected' || (i.next && i.next.who === 'mis'); } },
    { id: 'finance', label: 'در بررسی مالی', icon: 'hourglass', tone: 'orange', key: '2', f: function (i) { return i.review === 'pending'; } },
    { id: 'pre', label: 'پیش‌فاکتور', icon: 'file', tone: 'violet', key: '3', f: function (i) { return i.inv === 'pre'; } },
    { id: 'staged', label: 'مرحله‌ای ناتمام', icon: 'layers', tone: 'teal', key: '4', f: function (i) { return i.inv === 'staged'; } },
    { id: 'all', label: 'همه', icon: 'receipt', tone: 'neutral', key: '5', f: function () { return true; } }
  ];
  X.IQ = IQ;
  V.inv = function () {
    var q = IQ.filter(function (x) { return x.id === st.iq; })[0] || IQ[0], visible = M.invoices.filter(function (i) { return !failedM(X.mgrOfSeller(seller(i.seller))); }), list = visible.filter(q.f);
    var tabs = h.queues(IQ.map(function (x) { return { id: x.id, label: x.label, icon: x.icon, tone: x.tone, key: x.key, n: visible.filter(x.f).length }; }), q.id, 'data-iq');
    var rows = list.map(function (i) {
      var s = seller(i.seller), rv = X.REV[i.review], ch = X.chainOfSeller(s);
      return '<tr data-row="inv:' + i.code + '" tabindex="-1"><td><span class="mono case-id">' + fa(i.code) + '</span><div class="cell-sub">' + esc(i.customer) + '</div></td>' +
        '<td><div class="ctx-chain"><span>' + esc(X.mgrName(ch.mgr)) + '</span>' + (ch.senior ? '<span>' + esc(X.seniorName(ch.senior)) + '</span>' : '') + (ch.team ? '<span>سرپرست ' + esc(team(ch.team).sup) + '</span>' : '') + '<b>' + esc(s.name) + '</b></div></td>' +
        '<td class="col-opt"><span class="cell-sub" style="margin:0">' + esc(i.case) + '</span><div class="cell-sub">' + X.link(i.link) + '</div></td>' +
        '<td>' + pill(X.INVS[i.inv].label, X.INVS[i.inv].tone, X.INVS[i.inv].icon) + '</td>' +
        '<td><b>مرحله ' + fa(i.stage) + ' از ' + fa(i.stages) + '</b><div class="cell-sub">' + esc(X.STG[i.stg]) + ' · ' + esc(X.EVID[i.evidence]) + '</div></td>' +
        '<td>' + (rv ? pill(rv.label, rv.tone, rv.icon) : '<span class="muted">ارسال نشده</span>') + '</td>' +
        '<td class="amt-col">' + money(i.total) + '<div class="cell-sub">پرداخت‌شده: ' + money(i.paid) + '</div><div class="cell-sub">مانده: ' + money(i.inv === 'cancelled' ? 0 : i.total - i.paid) + '</div></td>' +
        '<td>' + X.nextActor(i.next, i) + (i.reason ? '<div class="why-line">' + ic('alert') + esc(i.reason) + '</div>' : '') + (i.mismatch ? '<div class="why-line warn">' + ic('question') + esc(i.mismatch) + '</div>' : '') + '</td>' +
        '<td class="col-actions"><button type="button" class="btn btn-sm" data-act="open-inv:' + i.code + '">جزئیات</button></td></tr>';
    }).join('');
    return h.pageHead({ title: 'زمینه فاکتور', sub: 'نمای فقط‌خواندنی برای نظارت · وضعیت فاکتور، مرحله پرداخت و بررسی مالی سه چیز جدا هستند · هیچ اقدام مالی یا کمکی از این پنل ممکن نیست', kpis: [
      { label: 'فاکتورهای نمونه اخیر', value: fa(visible.length), color: 'neutral', keep: true, meaning: 'نمونه اخیر و محدود؛ جمع کل محدوده یا کل شاخه نیست (SD-G12)', basis: 'وضعیت در لحظه · نمونه' },
      { label: 'نیازمند اقدام فروش', value: fa(visible.filter(IQ[0].f).length), color: 'red', tone: 'bad', filter: 'iq:action', meaning: 'رد مالی یا مغایرت؛ اقدام بعدی با فروشنده یا MIS است، نه شما', basis: 'وضعیت در لحظه' },
      { label: 'در بررسی مالی', value: fa(visible.filter(IQ[1].f).length), color: 'orange', filter: 'iq:finance', meaning: 'منتظر واحد مالی؛ اقدام شما لازم نیست', basis: 'وضعیت در لحظه' }
    ], fresh: X.freshPart('فاکتورها') }) + X.banners() +
      '<section class="panel main">' + h.banner('locked', '<b>فقط‌خواندنی.</b> کنترل‌های کمکی و نوشتنی مدیر (ارسال/کپی لینک پرداخت، ویرایش رسید) به معاون منتقل نشده‌اند؛ تأیید/رد مالی، بازپرداخت، ثبت مالی و قوانین پورسانت خارج از نقش است.', '') + tabs +
      h.toolbar('کد فاکتور، مشتری یا فروشنده', [{ label: 'شاخه', icon: 'users' }, { label: 'بازه صدور', icon: 'clock' }]) +
      '<div class="note warn inset">' + ic('rows') + '<span><b>نمونه اخیر و محدود.</b> این جدول جمع کل نیست؛ مبلغ و تعداد با «واحد: تومان» و جدا از شمارش پرونده‌ها خوانده می‌شود. کامل‌بودن فیلدهای جزئیات هنوز اثبات نشده است.</span></div>' +
      (list.length ? '<div class="tbl-wrap"><table class="tbl" aria-label="فاکتورهای ' + esc(q.label) + '"><thead><tr><th>فاکتور</th><th>زنجیره شاخه › فروشنده</th><th class="col-opt">پرونده / منبع و اتصال</th><th>وضعیت فاکتور</th><th>مرحله پرداخت</th><th>بررسی مالی</th><th>کل · پرداخت‌شده · مانده (تومان)</th><th>اقدام بعدی با و دلیل</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div>' + h.tfoot(list.length, visible.length)
        : h.stateBlock('empty', 'فاکتوری در این صف نیست', 'صف دیگری را انتخاب کنید.', '<button type="button" class="btn btn-soft" data-iq="all">همه فاکتورها</button>')) + '</section>';
  };

  /* ================= Archive / Extra-number Monitor (CONDITIONAL READ — never a Manager queue) ================= */
  V.mon = function () {
    var A = st.monAuth, R = M.archReasons, keys = ['noanswer', 'assess', 'sub', 'product', 'productx'];
    var seg = '<div class="seg" role="group" aria-label="وضعیت دسترسی (دمو)">' + [['partial', 'مجاز جزئی'], ['unauth', 'غیرمجاز'], ['unavail', 'در دسترس نیست']].map(function (b) { return '<button type="button" data-monauth="' + b[0] + '" aria-pressed="' + (A === b[0]) + '">' + b[1] + '</button>'; }).join('') + '</div>';
    var QS = [{ id: 'arch', label: 'تجمیع آرشیو', icon: 'layers', tone: 'blue', key: '1' }, { id: 'archrec', label: 'ریز رکوردهای آرشیو', icon: 'rows', tone: 'amber', key: '2' }, { id: 'xnum', label: 'پس‌افتادگی شماره اضافه', icon: 'plus', tone: 'violet', key: '3' }, { id: 'xout', label: 'زمینه نتیجه درخواست', icon: 'history', tone: 'slate', key: '4' }];
    var blocked = A === 'unauth' ? h.stateBlock('locked', 'این داده برای محدوده شما مجاز نشده است', 'دسترسی تأیید نشده؛ هیچ عدد، صفر یا فهرست خالی نمایش داده نمی‌شود. مجوز با تصمیم محدوده/ماژول/ردیف/فیلد جدا ثبت می‌شود، نه با جایگاه شما در سلسله‌مراتب.', '') : A === 'unavail' ? h.stateBlock('error', 'منبع این داده در دسترس نیست', 'این به معنی «موردی وجود ندارد» نیست. بعداً دوباره بخوانید؛ داده ساخته نمی‌شود.', '<button type="button" class="btn btn-soft" data-refresh>' + ic('refresh') + 'تلاش مجدد</button>') : null;
    var body;
    if (st.mq === 'arch') {
      if (blocked) body = blocked;
      else {
        var rows = M.archAgg.map(function (a) {
          var nullAny = keys.some(function (k) { return a[k] == null; }), tot = keys.reduce(function (x, k) { return x + (a[k] || 0); }, 0);
          return '<tr><td>' + X.who(X.mgrName(a.mgr), nullAny && keys.every(function (k) { return a[k] == null; }) ? X.cov('unauth', 'مجاز نشده') : '') + '</td>' + keys.map(function (k) { return '<td class="n">' + (a[k] == null ? X.cov('unauth', 'مجاز نیست') : n0(a[k])) + '</td>'; }).join('') + '<td class="n">' + (nullAny ? X.cov('partial', 'جمع ناقص') : fa(tot)) + '</td></tr>';
        }).join('');
        body = '<div class="note warn inset">' + ic('rows') + '<span><b>برش نمونه، نه جمع کل.</b> تجمیع آرشیو هر مدیر فقط ردیف‌های حل‌نشده در محدوده خودِ او را شامل می‌شود (سقف ۲۰۰ ردیف)؛ نبود عدد برای یک شاخه «صفر» نیست. آرشیو حذف نیست و وابستگی مالی را آزاد نمی‌کند.</span></div><div class="tbl-wrap"><table class="tbl no-cursor" aria-label="تجمیع آرشیو به تفکیک مدیر"><thead><tr><th>شاخه مدیر</th>' + keys.map(function (k) { return '<th class="n">' + esc(R[k].label) + '</th>'; }).join('') + '<th class="n">جمع</th></tr></thead><tbody>' + rows + '</tbody></table></div>';
      }
    } else if (st.mq === 'xnum') {
      if (blocked) body = blocked;
      else {
        var xr = M.xnumAgg.map(function (a) { return '<tr><td>' + X.who(X.mgrName(a.mgr), a.pending == null ? X.cov('unauth', 'مجاز نشده') : '') + '</td><td class="n">' + (a.pending == null ? X.cov('unauth', 'مجاز نیست') : n0(a.pending)) + '</td><td class="n">' + (a.decided == null ? X.cov('unauth', 'مجاز نیست') : n0(a.decided)) + '</td></tr>'; }).join('');
        body = '<div class="note info inset">' + ic('info') + '<span><b>سیگنال پایش، نه صف تصمیم.</b> تأیید/رد شماره اضافه با مدیرِ تعیین‌شده در همان ردیف است؛ جایگاه معاون جای مالک ردیف را نمی‌گیرد. اینجا فقط تعداد پس‌افتادگی دیده می‌شود و هیچ ردیف، شماره یا دکمه‌ی تصمیمی نیست.</span></div><div class="tbl-wrap"><table class="tbl no-cursor" aria-label="پس‌افتادگی شماره اضافه به تفکیک مدیر"><thead><tr><th>شاخه مدیر</th><th class="n">منتظر تصمیم مدیر</th><th class="n">تصمیم‌گرفته‌شده (برش اخیر)</th></tr></thead><tbody>' + xr + '</tbody></table></div>';
      }
    } else {
      if (blocked) body = blocked;
      else body = h.stateBlock('locked', st.mq === 'archrec' ? 'ریز رکوردهای آرشیو: نیازمند اعتبارسنجی' : 'زمینه نتیجه درخواست: نیازمند اعتبارسنجی', st.mq === 'archrec' ? 'دسترسی ریز رکورد آرشیو به محدوده مدیرِ مالک وابسته است و از سلسله‌مراتب معاون نتیجه نمی‌شود. تا تأیید اعتبار و مجوز، هیچ رکوردی نمایش داده نمی‌شود.' : 'منشأ درخواست و سرنخ قدیمی حاصل از آن باید با محافظ میدان جدا اثبات شود؛ شماره موبایل فقط برای جستجوست و هویت پرونده نیست.', '');
    }
    return h.pageHead({ title: 'پایش آرشیو و شماره اضافه', sub: 'خواندن مشروط برای پایش رهبری · نه صف مدیر · بررسی، احیا، حذف و تأیید/رد این پنل ممکن نیست', fresh: { text: 'عکس لحظه‌ای: ' + M.freshness.now, stale: false }, scope: 'مشروط: با مجوز تأییدشده' }) + X.banners() +
      '<section class="panel main">' + h.banner('locked', '<b>مشروط (SD-G08).</b> وجود این صفحه مجوز خواندن نیست؛ نمایش واقعی فقط با داده و مجوز تأییدشده انجام می‌شود. در غیر این‌صورت «غیرمجاز» یا «در دسترس نیست» نمایش داده می‌شود، نه داده نمایشی یا صفر.', '') +
      '<div class="toolbar"><span class="muted">حالت نمایش (دمو):</span>' + seg + '</div>' + h.queues(QS, st.mq, 'data-mq') + body +
      '<div class="bulk-note">' + ic('lock') + '<span>بررسی آرشیو، احیا، حذف، بررسی شماره اضافه و هر اقدام گروهی روی آنها: <b>در این نقش نیست</b>. خواندن گروهی آرشیو: نیازمند اعتبارسنجی.</span></div></section>';
  };

  /* ================= Diagnostics (ADVANCED, read-only; no repair) ================= */
  V.diag = function () {
    var QS = [{ id: 'f02', label: 'پیش‌فاکتور (F02)', icon: 'file', tone: 'violet', key: '1' }, { id: 'f03', label: 'پوشش پرونده/لید (F03)', icon: 'layers', tone: 'amber', key: '2' }, { id: 'f04', label: 'تطبیق گزارش و فاکتور (F04)', icon: 'swap', tone: 'orange', key: '3' }, { id: 'lineage', label: 'منشأ و اتصال پرونده', icon: 'unlink', tone: 'slate', key: '4' }];
    var forbid = X.sec('چه چیزی ممنوع است؟', '', X.checks([['no', 'اصلاح عدد، مبلغ، وضعیت یا اتصال برای هماهنگی ظاهری', 'هیچ کنترل «اصلاح» یا «ادغام» در این صفحه نیست'], ['no', 'اعلام شاخص نهایی یا رفع‌شدن F02/F03/F04', 'عدد تا تطبیق «نیازمند تطبیق» می‌ماند'], ['info', 'مسئول رفع', 'مالک داده/گزارش موجود و MIS؛ معاون فقط مغایرت و گروه شمارش را می‌بیند.']]));
    var body;
    var card = function (x, tone) { return '<div class="rc-card"><span class="rc-n">' + esc(x.name) + '</span><b class="rc-v">' + fa(x.value) + '</b><span class="rc-m">واحد شمارش: ' + esc(x.grain) + ' · ' + esc(x.rows) + '</span><span class="pill t-' + tone + '">' + ic('question') + 'عدد نهایی تأیید نشده</span></div>'; };
    if (st.dq === 'f02') {
      var pre = M.invoices.filter(function (i) { return i.inv === 'pre'; });
      body = '<div class="rc-grid">' + card({ name: 'نمای کلی · شاخص پیش‌فاکتور باز', value: M.f02.tile, grain: 'فاکتور (شرط pending/draft/unpaid)', rows: 'محاسبه هنگام رندر' }, 'amber') + '<div class="rc-vs" aria-hidden="true">' + ic('swap') + '</div>' + card({ name: 'جدول فاکتورها · ردیف‌های پیش‌فاکتور', value: pre.length, grain: 'فاکتور (وضعیت pre_invoice)', rows: 'نمونه اخیر' }, 'amber') + '</div>' +
        '<div class="rc-body">' + X.sec('شاهد شرط شمارش', 'BOTH / HIGH', '<p class="ind-note" style="padding:0">' + ic('question') + ' ' + esc(M.f02.note) + ' این یک <b>شکاف پیاده‌سازی</b> است و با ظاهر جدید رفع‌شده فرض نشده است (SD-G02).</p>') +
        X.sec('ردیف‌های مرتبط', '', '<div class="mini-list">' + pre.map(function (i) { return '<button type="button" class="mini as-btn" data-act="open-inv:' + i.code + '"><div class="grow"><span class="mono">' + fa(i.code) + '</span><div class="muted" style="font-size:var(--t-meta)">' + esc(X.mgrName(X.mgrOfSeller(seller(i.seller)))) + ' · ' + esc(seller(i.seller).name) + '</div></div>' + pill(X.INVS[i.inv].label, X.INVS[i.inv].tone, X.INVS[i.inv].icon) + '</button>'; }).join('') + '</div>') + forbid + '</div>';
    } else if (st.dq === 'f03') {
      var legacy = X.sum(M.sellers, 'legacy');
      body = '<div class="rc-grid">' + '<div class="rc-card"><span class="rc-n">فقط لیدهای قدیمی (شمارش پایه)</span><b class="rc-v">' + fa(legacy) + '</b><span class="rc-m">واحد شمارش: لید قدیمی یکتا</span><span class="pill t-teal">' + ic('checkCircle') + 'فقط یک منبع</span></div><div class="rc-vs" aria-hidden="true">' + ic('swap') + '</div><div class="rc-card"><span class="rc-n">کل پرونده‌ها (پرونده منطقی)</span><b class="rc-v na">—</b><span class="rc-m">اجتماع منابع با اتصال اثبات‌شده</span><span class="pill t-amber">' + ic('layers') + 'پوشش ناقص — نهایی نیست</span></div></div>' +
        '<div class="rc-body">' + X.sec('کدام منبع شمرده می‌شود؟', 'F03 · CODE VERIFIED', X.covMatrix(X.sources(), 'پوشش منابع کل پرونده‌ها')) +
        X.sec('این به چه معناست؟', '', X.checks([['info', 'صفر لید قدیمی به معنی صفر پرونده نیست', 'V4، MIS و Dot امروز دوباره بازبینی نشده‌اند و در شمارش پایه نیستند.'], ['q', 'ادغام منابع با شماره موبایل ممنوع است', 'شماره فقط برای جستجوست؛ هویت پرونده با شناسه بومی و اتصال اثبات‌شده تعیین می‌شود (Gate 0).'], ['no', 'نبود پوشش، صفر نمایش داده نمی‌شود', 'ستون‌ها «پوشش ناقص» یا «—» می‌مانند']])) + forbid + '</div>';
    } else if (st.dq === 'f04') {
      var R = M.f04, deputyOv = X.allFacts().created, deputyTbl = M.invoices.length;
      body = '<div class="rc-grid">' + card(R.panel, 'amber') + '<div class="rc-vs" aria-hidden="true">' + ic('swap') + '</div>' + card(R.tab, 'amber') + '</div>' +
        '<div class="rc-body">' + X.sec('وضعیت در معاون امروز', 'بازتولید اعلام نمی‌شود', X.checks([['info', 'نمای رهبری ' + fa(deputyOv) + ' فاکتور · جدول فاکتورها ' + fa(deputyTbl) + ' ردیف', 'در این محدوده ناهمخوانی دیده نشده؛ این رفع F04 نیست و خطر ناهمخوانی سیستمی باقی است.']])) +
        X.sec('آیا این دو عدد قابل مقایسه‌اند؟', 'پیش از هر نتیجه‌گیری', X.checks(R.checks.map(function (c) { return [c[2], c[1] + ' — هم‌خوانی تأیید نشده', c[3]]; })), 'primary') +
        X.sec('علت', 'اثبات‌نشده', '<p class="ind-note" style="padding:0">' + ic('question') + ' علت <b>تأیید نشده</b> است. فرضیه‌ها فقط برای بررسی هستند، نه نتیجه: ' + R.hypotheses.map(function (x) { return '<span class="tag">' + esc(x) + '</span>'; }).join(' ') + '</p>') + forbid + '</div>';
    } else {
      var bad = M.invoices.filter(function (i) { return i.link !== 'confirmed'; }).map(function (i) { return { ref: 'فاکتور ' + fa(i.code), case_: i.case, link: i.link, mgr: X.mgrOfSeller(seller(i.seller)), src: /MIS/.test(i.case) ? 'mis' : /V4/.test(i.case) ? 'v4' : 'legacy', note: i.link === 'partial' ? 'تنها بخشی از منابع اتصال را تأیید می‌کنند' : i.link === 'conflict' ? i.mismatch : 'اتصال قابل تأیید نیست' }; })
        .concat(M.held.filter(function (c) { return c.link; }).map(function (c) { return { ref: 'پرونده ' + c.id, case_: c.source, link: c.link, mgr: c.holder.kind === 'manager' ? c.holder.mgr : c.holder.kind === 'senior' ? senior(c.holder.unit).mgr : c.holder.kind === 'sup' ? X.mgrOfTeam(c.holder.team) : X.mgrOfSeller(seller(c.holder.id)), src: c.legacy ? 'legacy' : 'mis', note: c.why }; }));
      var rows = bad.map(function (b) { return '<tr><td><span class="mono">' + esc(b.ref) + '</span><div class="cell-sub">' + X.src(b.src) + '</div></td><td>' + esc(b.case_) + '</td><td>' + X.link(b.link) + '</td><td>' + esc(X.mgrName(b.mgr)) + '</td><td>' + esc(b.note || '') + '</td><td><span class="pill t-slate">' + ic('question') + 'مالک نامشخص (UNKNOWN)</span><div class="cell-sub">مالک داده مجاز از مسیر موجود</div></td></tr>'; }).join('');
      body = '<div class="note info inset">' + ic('shield') + '<span><b>فقط تشخیص، بدون اصلاح.</b> مرجع‌ها (CaseRef / SourceRef) و ناقص‌بودن دیده می‌شوند؛ اتصال با شماره موبایل حدس زده نمی‌شود و دکمه ادغام/اصلاح خام MIS وجود ندارد. اقدام وابسته تا اثبات اتصال متوقف می‌ماند (fail-closed).</span></div>' +
        '<div class="tbl-wrap"><table class="tbl no-cursor stackable" aria-label="منشأ و اتصال نامطمئن"><thead><tr><th>مرجع و منبع</th><th>پرونده / منبع ثبت‌شده</th><th>اطمینان اتصال</th><th>شاخه</th><th>دلیل</th><th>مالک رفع</th></tr></thead><tbody>' + rows + '</tbody></table></div>' + h.tfoot(bad.length, bad.length);
    }
    return h.pageHead({ title: 'عیب‌یابی شاخص و منشأ', sub: 'پیشرفته · فقط خواندنی · چرا یک عدد قابل اعتماد نیست؟ · F02 و F03 و F04 رفع‌شده فرض نمی‌شوند', fresh: X.freshPart('عیب‌یابی'), scope: 'تشخیص، نه ترمیم' }) + X.banners() +
      '<section class="panel main">' + h.queues(QS, st.dq, 'data-dq') + body + '</section>';
  };

  /* ================= Wallet (Own Account) ================= */
  V.wallet = function () {
    var W = M.wallet, rows = W.tx.map(function (t) { return '<tr><td class="muted"><span class="num">' + esc(t.date) + '</span></td><td>' + pill('بستانکار', 'green', 'plus') + '</td><td><span class="tag' + (t.channel === 'آنلاین' ? ' teal' : '') + '">' + esc(t.channel) + '</span></td><td>' + esc(t.desc) + '<div class="muted" style="font-size:var(--t-micro)">' + esc(t.rule) + '</div></td><td style="text-align:left"><span class="credit">+<span class="num">' + num(t.amount) + '</span></span> <small class="muted">تومان</small></td></tr>'; }).join('');
    return h.pageHead({ title: 'کیف پول من', sub: 'حساب شخصی و فقط‌خواندنی — فقط حق‌الزحمه خودتان؛ درآمد یا پورسانت شاخه‌ها، تیم‌ها و مالی سازمان اینجا نیست' }) +
      '<div class="wallet-sum"><div class="wcard hero"><span>موجودی ثبت‌شده</span><b><span class="num">' + num(W.balance) + '</span><small>تومان</small></b><div class="foot"><span>بدون تسویه‌نشده‌ی محاسبه‌شده</span></div></div><div class="wcard"><span>کل بستانکاری</span><b><span class="num">' + num(W.credit) + '</span><small>تومان</small></b><div class="foot"><span>' + fa(W.tx.length) + ' تراکنش</span></div></div><div class="wcard"><span>کل برداشت / تسویه</span><b><span class="num">' + num(W.debit) + '</span><small>تومان</small></b><div class="foot"><span>تسویه‌ای ثبت نشده</span></div></div></div>' +
      '<section class="panel main">' + h.banner('locked', '<b>خواندن محض.</b> باز کردن این صفحه هیچ اعتبار، ثبت یا محاسبه‌ای انجام نمی‌دهد (F07؛ فعال‌سازی فنی نیازمند جداسازی وابستگی، SD-G11). «ثبت‌شده» با «قابل تسویه» و «تسویه‌شده» یکی نیست و پورسانت زیرمجموعه‌ها اینجا نمی‌آید.', '') + '<div class="tbl-wrap"><table class="tbl no-cursor"><thead><tr><th>تاریخ</th><th>نوع</th><th>کانال</th><th>شرح</th><th style="text-align:left">مبلغ</th></tr></thead><tbody>' + rows + '</tbody></table></div>' + h.tfoot(W.tx.length, W.tx.length) + '</section>';
  };

  /* ================= Shared Reports (broader scoped hierarchy; never all-org by default) ================= */
  V.rep = function () {
    var scopeOpts = '<option value="all"' + (st.rscope === 'all' ? ' selected' : '') + '>همه شاخه‌های محدوده مجاز من</option>' + M.managers.map(function (m) { return '<option value="' + m.id + '"' + (st.rscope === m.id ? ' selected' : '') + '>مدیر ' + esc(m.name) + '</option>'; }).join('') + '<option disabled>شاخه‌ای خارج از محدوده — قابل انتخاب نیست</option>';
    var ctrl = '<div class="rep-ctrl"><div class="seg" role="group" aria-label="مبنای زمانی"><button type="button" data-rbasis="event" aria-pressed="' + (st.rbasis === 'event') + '">رویداد در بازه</button><button type="button" data-rbasis="now" aria-pressed="' + (st.rbasis === 'now') + '">وضعیت فعلی</button><button type="button" disabled aria-disabled="true" class="tip" data-tip="وضعیت تاریخی فقط با سابقه معتبر ممکن است؛ داده امروز اثبات گذشته نیست (F09) و در این نمونه سابقه کافی ثبت نشده">وضعیت تاریخی</button></div>' +
      (st.rbasis === 'event' ? '<label class="lbl inline">از<input class="input" value="۱۴۰۵/۰۷/۰۱"></label><label class="lbl inline">تا<input class="input" value="۱۴۰۵/۰۷/۱۰"></label>' : '<span class="muted">وضعیت فعلی هنگام اجرای گزارش</span>') +
      '<label class="lbl inline">محدوده<select class="input" data-rscope>' + scopeOpts + '</select></label><span class="scope-note">فقط شاخه‌های مجاز شما فهرست می‌شود؛ «همه سازمان» پیش‌فرض نیست و نقش‌های مالی/HR انتخاب‌شدنی نیستند.</span></div>';
    var body;
    if (st.report) {
      var r = M.reports.filter(function (x) { return x.id === st.report; })[0];
      var ms = M.managers.filter(function (m) { return st.rscope === 'all' || st.rscope === m.id; });
      var inRep = function (i) { return ms.some(function (m) { return m.id === X.mgrOfSeller(seller(i.seller)); }); };
      var cov = sim() === 'incomplete' ? 'partial' : sim() === 'stale' ? 'stale' : sim() === 'branchfail' ? 'partial' : 'ok';
      var tbl, count, sum;
      if (r.id === 'perf_by_mgr') {
        var byMgr = st.rscope === 'all';
        var grp = byMgr ? ms.map(function (m) { return { key: m.id, name: 'مدیر ' + m.name, f: failedM(m.id) ? null : X.mgr_f(m.id, 'current'), drill: m.id }; })
          : X.seniorsOfMgr(st.rscope).map(function (u) { return { key: u.id, name: 'سرپرست ارشد ' + u.name, f: X.senior_f(u.id, 'current') }; }).concat(X.directTeams(st.rscope).map(function (t) { return { key: t.id, name: 'سرپرست ' + t.sup + ' (مستقیم زیر مدیر)', f: X.team_f(t.id, 'current') }; }));
        var gr = grp.map(function (g) { return '<tr><td>' + esc(g.name) + (g.drill ? '' : '<div class="cell-sub">' + X.link('confirmed') + '</div>') + '</td>' + (g.f ? '<td class="n">' + fa(g.f.created) + '</td><td class="n">' + fa(g.f.completed) + '</td><td class="n">' + money(g.f.collected) + '</td>' : '<td colspan="3">' + X.cov('failed') + '</td>') + '<td class="col-actions">' + (g.drill ? '<button type="button" class="btn btn-sm" data-act="rscope:' + g.drill + '">پیمایش زیرمجموعه</button>' : '') + '</td></tr>'; }).join('');
        count = grp.length; sum = grp.reduce(function (a, g) { return a + (g.f ? g.f.collected : 0); }, 0);
        tbl = '<div class="tbl-wrap"><table class="tbl no-cursor" aria-label="' + esc(r.name) + '"><thead><tr><th>' + (byMgr ? 'مدیر شاخه' : 'زیرمجموعه مدیر') + '</th><th class="n">فاکتور صادرشده</th><th class="n">فروش تکمیل‌شده</th><th class="n">وصول تأییدشده</th><th class="col-actions"><span class="sr">پیمایش</span></th></tr></thead><tbody>' + gr + '</tbody></table></div>';
      } else if (r.id === 'invoices_register') {
        var lst = M.invoices.filter(inRep);
        count = lst.length; sum = lst.reduce(function (a, i) { return a + i.total; }, 0);
        tbl = '<div class="tbl-wrap"><table class="tbl no-cursor" aria-label="' + esc(r.name) + '"><thead><tr><th>کد</th><th>مشتری</th><th>فروشنده</th><th>زنجیره فعلی</th><th>مبلغ</th><th>وضعیت فاکتور</th><th>تاریخ صدور</th></tr></thead><tbody>' + lst.map(function (i, n) { var sl = seller(i.seller); return '<tr><td><span class="mono">' + fa(i.code) + '</span></td><td>' + esc(i.customer) + '</td><td>' + esc(sl.name) + '</td><td>' + esc(X.mgrName(X.mgrOfSeller(sl))) + ' › ' + esc(X.teamName(sl.team)) + (n === 2 ? '<div class="cell-sub why-line warn">' + ic('question') + 'برچسب سرپرست/فروشنده نیازمند اعتبارسنجی</div>' : '') + '</td><td>' + money(i.total) + '</td><td>' + pill(X.INVS[i.inv].label, X.INVS[i.inv].tone, X.INVS[i.inv].icon) + '</td><td class="muted">' + esc(i.issued) + '</td></tr>'; }).join('') + '</tbody></table></div>';
      } else if (r.id === 'case_load') {
        var cl = ms.map(function (m) { return '<tr><td>' + esc('مدیر ' + m.name) + '</td><td class="n">' + (failedM(m.id) ? na('دریافت نشد') : fa(X.mgrLoad(m.id))) + '</td><td class="n">' + (failedM(m.id) ? na('دریافت نشد') : fa(X.sum(X.sellersOfMgr(m.id), 'legacy'))) + '</td><td>' + X.cov(failedM(m.id) ? 'failed' : 'partial', failedM(m.id) ? null : 'فقط منبع شمرده‌شده') + '</td></tr>'; }).join('');
        count = ms.length; sum = null;
        tbl = '<div class="note warn inset">' + ic('layers') + '<span><b>پوشش منبع.</b> «کل پرونده‌ها» نهایی نیست (F03): فقط منبع قدیمی و پرونده‌های دارای مسئول فعلی فروشنده شمرده می‌شود.</span></div>' + '<div class="tbl-wrap"><table class="tbl no-cursor" aria-label="' + esc(r.name) + '"><thead><tr><th>شاخه</th><th class="n">پرونده باز نزد فروشندگان</th><th class="n">فقط لیدهای قدیمی</th><th>پوشش منبع</th></tr></thead><tbody>' + cl + '</tbody></table></div>';
      } else {
        var hl = M.hr.filter(function (q) { return ms.some(function (m) { return m.id === X.mgrOfSeller(seller(q.seller)); }); });
        count = hl.length; sum = null;
        tbl = '<div class="tbl-wrap"><table class="tbl no-cursor" aria-label="' + esc(r.name) + '"><thead><tr><th>درخواست</th><th>فروشنده</th><th>وضعیت</th><th>بررسی‌کننده فعلی</th></tr></thead><tbody>' + hl.map(function (q) { var S = X.HRS[q.state]; return '<tr><td>#' + fa(q.id) + ' · ' + (q.type === 'transfer' ? 'جابجایی' : 'قطع همکاری') + '</td><td>' + esc(seller(q.seller).name) + '</td><td>' + pill(S.label, S.tone, S.icon) + '</td><td>' + esc(q.reviewer === 'self' ? 'شما' : q.reviewer === 'manager' ? 'مدیر شاخه' : 'منابع انسانی') + '</td></tr>'; }).join('') + '</tbody></table></div>';
      }
      body = '<section class="panel main"><div class="rep-head"><button type="button" class="btn btn-ghost btn-sm" data-act="close-report">' + ic('arrowL') + 'همه گزارش‌ها</button><h2>' + esc(r.name) + '</h2><span class="grow"></span>' + h.freshness({ text: 'محاسبه‌شده: امروز ۱۰:۴۸', refresh: true }) + '<button type="button" class="btn btn-sm" disabled aria-disabled="true">' + ic('download') + 'خروجی (مشروط · مجوز تأیید نشده)</button></div>' +
        '<div class="rep-meta"><span class="chip-m">سطح: ' + esc(r.grain) + '</span><span class="chip-m">مبنا: ' + esc(st.rbasis === 'now' ? 'وضعیت فعلی' : r.basis) + '</span><span class="chip-m">محدوده: ' + (st.rscope === 'all' ? 'همه شاخه‌های مجاز (نه کل سازمان)' : esc(X.mgrName(st.rscope))) + '</span><span class="chip-m">تعریف شاخص: نسخه ۱</span><span class="chip-m">واحد پول: تومان</span><span class="chip-m">' + X.cov(cov) + '</span><span class="chip-m">' + X.cov('bounded', 'نتیجه محدود ۶۰ تا ۱۲۰ ردیف') + '</span><span class="chip-m">خروجی: مجوز و فیلدها تأیید نشده</span><span class="chip-m tip" tabindex="0" data-tip="ستون‌های حقوق/پاداش، اطلاعات بانکی و رسید خام در این گزارش مخفی یا محدودند؛ مجوز مشاهده گزارش مجوز دیدن این میدان‌ها نیست">' + ic('lock') + 'میدان‌های حساس محدود</span></div>' +
        '<div class="note warn inset">' + ic('rows') + '<span><b>نتیجه محدود.</b> درخواست گزارش فقط ۶۰ تا ۱۲۰ ردیف برمی‌گرداند؛ این <b>کل محدوده</b> نیست و با پوشش V4 + قدیمی کامل‌بودن منابع اثبات نمی‌شود. اگر شمار گزارش با جدول فاکتورها نخواند به «عیب‌یابی شاخص و منشأ» بروید (F04). مشترک‌بودن گزارش دسترسی سراسری نمی‌سازد.</span></div>' +
        '<div class="kpis rep-kpis">' + h.kpi({ label: 'تعداد در این برش', value: fa(count), color: 'neutral', meaning: 'تعداد ردیف‌های یکتا در برش نمایش‌داده‌شده؛ جمع کل نیست', basis: 'همان محدوده جدول' }) + (sum != null ? h.kpi({ label: 'جمع مبلغ این برش', value: '<span class="num">' + num(sum) + '</span><small>تومان</small>', color: 'teal', meaning: 'جمع همان ردیف‌ها با همان تعریف', basis: 'همان محدوده جدول' }) : '') + '</div>' +
        (count ? tbl : h.stateBlock('noresult', 'ردیفی در این محدوده نیست', 'محدوده را تغییر دهید؛ دسترسی بیرون از محدوده ایجاد نمی‌شود.', '<button type="button" class="btn btn-soft" data-act="rscope:all">همه شاخه‌های مجاز</button>')) + '</section>';
    } else {
      body = '<div class="rep-grid">' + M.reports.map(function (r) { return '<button type="button" class="rep-card" data-act="open-report:' + r.id + '"><span class="rep-ico">' + ic(r.icon) + '</span><b>' + esc(r.name) + '</b><span class="rep-tags"><span>سطح: ' + esc(r.grain) + '</span><span>مبنا: ' + esc(r.basis) + '</span></span><span class="rep-go">مشاهده آمار و ریزداده ' + ic('arrowL') + '</span></button>'; }).join('') + '</div>';
    }
    return h.pageHead({ title: 'گزارش‌ها', sub: 'ورودی واحد گزارش‌های مشترک CRM با محدوده چندمدیره شما (نه کل سازمان) · داشتن گزارش مالی به معنی اقدام مالی نیست · هیچ گزارش یا خروجی اجرا نمی‌شود', fresh: X.freshPart('گزارش‌ها') }) + X.banners() + '<section class="panel rep-ctrl-panel">' + ctrl + '</section>' + body;
  };
})();
