/* Sales Manager — role layer, part 2: read/analysis views (operations & exceptions, performance, invoices, archives, extra-number queue,
   customer context, wallet, reports). Presentation of Product-Spec-approved facts; undefined metrics render as «تعریف نشده», never as 0. */
(function () {
  'use strict';
  var X = window.MGRX, C = X.C, M = X.M, h = X.h, st = X.st, V = X.V;
  var esc = h.esc, fa = h.fa, num = h.num, money = h.money, ic = h.ic, pill = h.pill, hint = h.hint;
  var sim = X.sim, seller = X.seller, team = X.team, unit = X.unit;
  var na = function (t) { return '<span class="na tip" tabindex="0" data-tip="' + esc(t || 'داده در دسترس نیست') + '">—</span>'; };
  var n0 = function (v) { return v ? fa(v) : '<span class="muted">۰</span>'; };
  var failedU = function (uid) { return X.unitCov(uid) === 'failed'; };

  function topExc(uid) {
    if (uid === 'U3') return { t: 'مسئول جایگزین ثبت نشده', a: 'منابع انسانی', tone: 'amber' };
    var ts = X.teamsOf(uid);
    if (ts.some(function (t) { return !t.active; })) return { t: 'سرپرست غیرفعال در زیرمجموعه', a: 'منابع انسانی', tone: 'amber' };
    var sl = X.sellersOfUnit(uid);
    if (M.invoices.some(function (i) { return sl.some(function (s) { return s.id === i.seller; }) && i.inv === 'mismatch'; })) return { t: 'مغایرت فاکتور', a: 'MIS / مالک داده', tone: 'amber' };
    if (M.invoices.some(function (i) { return sl.some(function (s) { return s.id === i.seller; }) && i.review === 'rejected'; })) return { t: 'فیش ردشده', a: 'فروشنده', tone: 'red' };
    if (M.invoices.some(function (i) { return sl.some(function (s) { return s.id === i.seller; }) && i.review === 'pending'; })) return { t: 'در انتظار بررسی مالی', a: 'واحد مالی', tone: 'orange' };
    return null;
  }

  /* ================= Operations & Exceptions (overview) ================= */
  V.ov = function () {
    var A = X.attention(), bad = sim() === 'unitfail' ? 'unavailable' : sim() === 'stale' ? 'stale' : null;
    var pre = M.units.concat([{ id: 'M' }]).filter(function (u) { return !failedU(u.id); }).reduce(function (a, u) { return a + X.unit_f(u.id, 'current').openPre; }, 0);
    var xPend = M.xreq.filter(function (r) { return r.state === 'pending' && !st.xDone[r.id]; }).length;
    var hrWait = M.hr.filter(function (r) { return r.state === 'pending_review' && r.reviewer === 'self' && !st.hrDone[r.id]; }).length;
    var kpis = [
      { label: 'دسته استثنای باز', value: fa(A.length), color: 'orange', keep: true, meaning: 'هر دسته با مالک، مسئول بعدی و علامت حل مشخص است؛ شمار آیتم نیست', basis: 'وضعیت در لحظه · محدوده مجاز شما' },
      { label: 'موجودی قابل توزیع من', value: fa(X.poolOk().length), color: 'teal', keep: true, filter: 'dist:assign', meaning: 'پرونده‌های واجد شرایط تخصیص از پنل شخصی شما؛ جدا از بار واحدها', basis: 'وضعیت در لحظه' },
      { label: 'پیش‌فاکتور باز', value: fa(pre), color: 'violet', state: bad, filter: 'perf:unit', meaning: 'پیش‌فاکتورهای جاری بدون تکمیل یا لغو؛ با «پیش‌فاکتور صادرشده» یکی نیست', basis: 'وضعیت در لحظه' },
      { label: 'شماره اضافه منتظر شما', value: fa(xPend), color: 'blue', filter: 'xnum:pending', meaning: 'درخواست‌هایی که مدیر تعیین‌شده در ردیف شماست', basis: 'وضعیت در لحظه' },
      { label: 'HR منتظر مرحله شما', value: fa(hrWait), color: 'orange', meaning: 'فقط مرحله شما؛ اعمال نهایی با منابع انسانی است', basis: 'وضعیت در لحظه' }
    ];
    var attn = '<section class="panel attn-panel" aria-labelledby="attn-h"><div class="sec-h ph"><h2 id="attn-h">' + ic('inbox') + 'استثناهای نیازمند توجه</h2><span class="aside">' + fa(A.length) + ' دسته</span></div>' +
      (A.length ? '<ul class="attn">' + A.map(function (a) {
        return '<li><button type="button" class="attn-row" data-attn="' + a.id + '" style="--c:var(--' + a.tone + '-dot)"><span class="attn-ico t-' + a.tone + '">' + ic(a.icon) + '</span><span class="attn-txt"><b>' + esc(a.label) + '</b><span><span class="attn-own">مسئول: ' + esc(a.owner) + '</span>' + esc(a.state) + (a.subject ? ' · ' + esc(a.subject) : '') + '</span></span><span class="attn-n" aria-label="' + fa(a.n) + ' مورد">' + fa(a.n) + '</span>' + ic('arrowL') + '</button></li>';
      }).join('') + '</ul>' : h.stateBlock('empty', 'استثنای ثبت‌شده‌ای در محدوده کامل شما نیست', 'این نتیجه فقط برای محدوده و منابع دریافت‌شده است و به معنی «همه چیز قابل تخصیص یا آزاد است» نیست.')) +
      '<div class="attn-foot">' + ic('info') + '<span>«حل‌شده» لزوماً به معنی آزادشدن پرونده نیست؛ مصرف‌شده یا آرشیوشده ماندن می‌تواند نتیجه درست باشد. دستور ارجاع/تشدید و اعلان رسمی تعریف نشده (M-G11).</span></div></section>';

    var maxW = Math.max.apply(null, M.units.map(function (u) { return X.unitWorkload(u.id); }));
    var rows = M.units.map(function (u) {
      var cov = X.unitCov(u.id), f = X.unit_f(u.id, 'current'), ss = X.sellersOfUnit(u.id), act = ss.filter(function (s) { return s.active; }).length, w = X.unitWorkload(u.id), ex = topExc(u.id);
      if (cov === 'failed') return '<tr data-row="unit:' + u.id + '" tabindex="-1" class="failed"><td>' + X.who('سرپرست ارشد ' + u.name, X.rel('senior')) + '</td><td colspan="4"><div class="row-state">' + X.cov('failed') + '<span>دریافت آمار این واحد ناموفق بود؛ مقدار صفر فرض نشده است.</span></div></td><td>' + X.cov('failed') + '</td><td class="col-actions"><button type="button" class="btn btn-soft btn-sm" data-act="retry-unit">' + ic('refresh') + 'تلاش مجدد</button></td></tr>';
      return '<tr data-row="unit:' + u.id + '" tabindex="-1"' + (u.active ? '' : ' class="inactive"') + '><td>' + X.who('سرپرست ارشد ' + u.name, X.rel('senior') + (u.active ? '' : ' ' + pill('غیرفعال', 'slate', 'ban'))) + (u.active ? '' : '<div class="cell-sub">' + esc(u.inactiveNote) + '</div>') + '</td>' +
        '<td class="n">' + fa(act) + '<span class="of"> از ' + fa(ss.length) + '</span></td>' +
        '<td class="wl"><span class="wl-n">' + fa(w) + '</span><span class="wl-bar" aria-hidden="true"><i style="width:' + Math.round(w / maxW * 100) + '%"></i></span></td>' +
        '<td class="n col-opt">' + (cov === 'stale' ? '<span class="stale-n tip" data-tip="داده این واحد قدیمی است">' + fa(f.openPre) + '</span>' : fa(f.openPre)) + '</td>' +
        '<td>' + (ex ? '<div class="exc-cell"><span class="pill t-' + ex.tone + '">' + esc(ex.t) + '</span><span class="cell-sub">مسئول: ' + esc(ex.a) + '</span></div>' : '<span class="muted">—</span>') + '</td>' +
        '<td>' + X.cov(cov) + '</td><td class="col-actions"><button type="button" class="btn btn-soft btn-sm" data-act="open-unit:' + u.id + '">مشاهده</button></td></tr>';
    }).join('');
    var direct = X.teamsOf('M').map(function (t) {
      var f = X.team_f(t.id, 'current'), ss = X.sellersOf(t.id);
      return '<tr data-row="team:' + t.id + '" tabindex="-1" class="direct-row"><td>' + X.who(t.sup, X.rel('dsup') + ' <span class="cell-sub">بدون سرپرست ارشد میانی</span>') + '</td><td class="n">' + fa(ss.filter(function (s) { return s.active; }).length) + '<span class="of"> از ' + fa(ss.length) + '</span></td><td class="wl"><span class="wl-n">' + fa(X.teamWorkload(t.id)) + '</span></td><td class="n col-opt">' + fa(f.openPre) + '</td><td><span class="muted">—</span></td><td>' + X.cov('ok') + '</td><td class="col-actions"><button type="button" class="btn btn-soft btn-sm" data-act="open-team:' + t.id + '">مشاهده</button></td></tr>';
    }).join('') + M.sellers.filter(function (s) { return s.team === 'direct'; }).map(function (s) {
      return '<tr data-row="seller:' + s.id + '" tabindex="-1" class="direct-row"><td>' + X.who(s.name, X.rel('dseller') + ' <span class="cell-sub">بدون واسطه</span>') + '</td><td class="n">' + fa(1) + '<span class="of"> از ' + fa(1) + '</span></td><td class="wl"><span class="wl-n">' + fa(s.open) + '</span></td><td class="n col-opt">' + fa(X.seller_f(s.id).openPre) + '</td><td><span class="muted">—</span></td><td>' + X.cov('ok') + '</td><td class="col-actions"><button type="button" class="btn btn-soft btn-sm" data-act="open-seller:' + s.id + '">مشاهده</button></td></tr>';
    }).join('');
    var th = '<thead><tr><th>واحد و مسئول</th><th class="n">فروشنده فعال ' + hint('فروشندگان زیر این واحد؛ دیدن فروشنده غیرمستقیم اجازه نوشتن نمی‌سازد.', true) + '</th><th>پرونده نزد واحد ' + hint('مالک فعلی پرونده‌ها؛ این ستون رتبه یا ارزیابی عملکرد نیست.', true) + '</th><th class="n col-opt">پیش‌فاکتور باز</th><th>مهم‌ترین استثنا و مسئول آن</th><th>اعتماد به داده</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead>';
    var cap = '<section class="panel main cap-panel" aria-labelledby="cap-h"><div class="sec-h ph"><h2 id="cap-h">' + ic('users') + 'واحدهای تحت پوشش شما</h2>' + X.basis('current') + X.basis('snap') + X.grain('واحد مستقیم (سرپرست ارشد / سرپرست / فروشنده)') + '<span class="aside">به ترتیب نام · رتبه‌بندی نیست</span></div>' +
      '<div class="tbl-wrap"><table class="tbl" aria-label="واحدهای تحت پوشش"><caption class="sr">هر ردیف یک زیرمجموعه مستقیم شما با مسئول آن؛ ستون آخر وضعیت تازگی و پوشش داده است.</caption>' + th + '<tbody>' + rows + '<tr class="grp"><th colspan="7" scope="colgroup"><span>' + ic('user') + 'مستقیم زیر نظر شما (بدون سرپرست ارشد میانی)</span></th></tr>' + direct + '</tbody></table></div></section>';
    var pool = '<section class="panel pool-strip"><div class="sec-h ph"><h2>' + ic('inbox') + 'موجودی پنل من</h2><span class="scope-badge">' + ic('user') + 'شخصی · جدا از بار واحدها</span></div><div class="pool-body"><div><b class="pool-n">' + fa(M.pool.length) + '</b> پرونده <span class="muted">· ' + fa(X.poolOk().length) + ' قابل تخصیص · ' + fa(M.pool.length - X.poolOk().length) + ' نیازمند بررسی</span></div><button type="button" class="btn btn-soft" data-act="goto:dist">' + ic('send') + 'رفتن به توزیع</button></div></section>';
    var ident = '<section class="panel ind-panel"><div class="sec-h ph"><h2>' + ic('user') + 'هویت حساب و محدوده</h2></div><p class="ind-note">حساب: <b>' + esc(M.user.name) + '</b> · شناسه حساب <span class="mono">' + esc(M.user.id) + '</span> — شناسه فقط برای تشخیص حساب است، نه شاخص. ' + X.rel('senior') + ' ' + X.rel('dsup') + ' ' + X.rel('dseller') + ' ' + X.rel('indirect') + ' دیدن زیرمجموعه به معنی اجازه نوشتن یا تصمیم‌گیری مالی/HR/MIS نیست؛ سلسله‌مراتب مجوز نمی‌سازد.</p></section>';
    return h.pageHead({ title: 'عملیات و استثناها', sub: 'محدوده: ' + fa(M.units.length) + ' سرپرست ارشد · ' + fa(X.teamsOf('M').length) + ' سرپرست مستقیم · ' + fa(M.sellers.length) + ' فروشنده در محدوده · ارزیابی ' + esc(M.freshness.evaluated), kpis: kpis, fresh: X.freshPart('آمار واحدها'), scope: 'محدوده: زیرمجموعه مجاز شما' }) + X.banners() +
      '<div class="team-grid">' + attn + '<div class="team-main">' + cap + pool + ident + '</div></div>';
  };

  /* ================= Distribution (context) & Performance Explorer ================= */
  var MODES = [['unit', 'سرپرست ارشد', 'users'], ['sup', 'سرپرست', 'briefcase'], ['seller', 'فروشنده', 'user'], ['inv', 'فاکتور', 'receipt'], ['case', 'بار پرونده', 'inbox'], ['recon', 'تطبیق شاخص', 'swap']];
  function scopeSet() {
    var p = st.pscope; if (!p) return null; p = String(p);
    if (p === 'M') return { units: ['M'] };
    if (p.charAt(0) === 'U') return { units: [p] };
    if (p.charAt(0) === 'T') return { teams: [p] };
    return { sellers: [Number(p)] };
  }
  function inScopeUnit(uid) { var s = scopeSet(); if (!s) return true; if (s.units) return s.units.indexOf(uid) > -1; if (s.teams) return X.unitOfTeam(s.teams[0]) === uid; return X.unitOfTeam(seller(s.sellers[0]).team) === uid; }
  function inScopeTeam(tid) { var s = scopeSet(); if (!s) return true; if (s.units) return (tid === 'direct' ? 'M' : team(tid).parent) === s.units[0]; if (s.teams) return s.teams[0] === tid; return seller(s.sellers[0]).team === tid; }
  function inScopeSeller(sl) { var s = scopeSet(); if (!s) return true; if (s.units) return X.unitOfTeam(sl.team) === s.units[0]; if (s.teams) return sl.team === s.teams[0]; return s.sellers[0] === sl.id; }

  V.perf = function () {
    var pm = st.pm, basisBtn = pm === 'unit' || pm === 'inv';
    var dim = '<div class="dim-switch" role="group" aria-label="بعد تحلیل (واحد شمارش)">' + MODES.map(function (m) { return '<button type="button" data-pm="' + m[0] + '" aria-pressed="' + (pm === m[0]) + '">' + ic(m[2]) + m[1] + '</button>'; }).join('') + '</div>';
    var tb = (pm === 'unit' || pm === 'sup') ? '<div class="seg" role="group" aria-label="مبنای زمانی"><button type="button" data-tbs="snap" aria-pressed="' + (st.tb === 'snap') + '">وضعیت در لحظه</button><button type="button" data-tbs="event" aria-pressed="' + (st.tb === 'event') + '">رویداد در بازه</button><button type="button" disabled aria-disabled="true" class="tip" data-tip="وضعیت تاریخی (As-Of) فقط با سابقه معتبر ممکن است؛ داده امروز اثبات آن نیست (F09) و در این نمونه سابقه کافی ثبت نشده">وضعیت تاریخی</button></div>' : '';
    var basis = basisBtn ? '<div class="seg" role="group" aria-label="مبنای انتساب"><button type="button" data-pb="current" aria-pressed="' + (st.pb === 'current') + '">ساختار فعلی</button><button type="button" data-pb="hist" aria-pressed="' + (st.pb === 'hist') + '">انتساب تاریخی</button></div>' : '<span class="basis-fixed">' + (pm === 'seller' ? X.basis('event') + '<span class="muted">آمار فروشنده به خود او نسبت داده می‌شود و به والد امروز وابسته نیست.</span>' : pm === 'recon' ? '' : X.basis('current')) + '</span>';
    var body = ({ unit: perfUnit, sup: perfSup, seller: perfSeller, inv: perfInv, 'case': perfCase, recon: perfRecon })[pm]();
    var ctl = '<section class="panel explorer-ctl" aria-label="کنترل‌های کاوشگر">' + dim + tb + basis + '<button type="button" class="btn btn-sm btn-soft" data-act="open-metrics">' + ic('info') + 'تعریف شاخص‌ها</button></section>';
    return h.pageHead({ title: 'عملکرد و تطبیق', sub: 'سرپرست ارشد ← سرپرست ← فروشنده ← پرونده/فاکتور · هر حالت یک واحد شمارش دارد · بدون رتبه‌بندی، هدف فروش یا امتیاز', fresh: X.freshPart('داده'), scope: 'مبنا: ' + (basisBtn ? (st.pb === 'hist' ? 'انتساب تاریخی' : 'ساختار فعلی') : 'ثابت برای این حالت') }) + X.banners() + ctl +
      '<section class="panel main"><div class="ex-bar">' + X.crumb() + '<span class="grow"></span>' + X.grain(body.grain) + (basisBtn ? X.basis(st.pb === 'hist' ? 'hist' : 'current') : '') + '</div>' + (body.note ? '<div class="note info inset">' + ic('info') + '<span>' + body.note + '</span></div>' : '') + body.html + '</section>';
  };
  function covOrVal(tid, val) { return X.teamCov(tid) === 'failed' ? na('دریافت نشد') : val; }
  var TH = function (label, basisKey, tip) { return '<th class="n">' + label + ' <span class="tb-tag tb-' + basisKey + '">' + (basisKey === 'snap' ? 'لحظه' : 'رویداد') + '</span>' + (tip ? hint(tip, true) : '') + '</th>'; };

  // KPI columns per temporal concept (Product Spec §8): snapshot columns vs event columns never share a table view.
  function kpiCols(f, cov, dim) {
    var v = function (x) { return cov === 'failed' ? na('دریافت نشد') : x; };
    if (st.tb === 'snap') return [v(dim.cases), v(fa(f.openPre)), v(dim.legacy)];
    return [v(fa(f.created)), v(f.issued < f.created ? '<span class="partial-n tip" tabindex="0" data-tip="سابقه صدور برای ' + fa(f.created - f.issued) + ' فاکتور ثبت نشده؛ نبود سابقه صفر نیست">' + fa(f.issued) + '</span>' : fa(f.issued)), v(fa(f.completed)), v(cov === 'recon' ? (f.collected ? '<span class="recon-n tip" tabindex="0" data-tip="مغایرت بین منبع پرداخت و وضعیت فاکتور؛ مبلغ نهایی نیست">' + money(f.collected) + '</span>' : na('نیازمند تطبیق')) : money(f.collected))];
  }
  function kpiHead() {
    return st.tb === 'snap' ? TH('کل پرونده‌ها', 'snap', 'فقط منابع تأییدشده؛ پوشش کامل اثبات نشده و با شماره موبایل ادغام نشده.') + TH('پیش‌فاکتور باز', 'snap') + TH('فقط لیدهای قدیمی', 'snap', 'شمارش جدول قدیمی؛ برچسب «کل لیدها» نیست.')
      : TH('فاکتور صادرشده', 'event') + TH('پیش‌فاکتور صادرشده', 'event', 'کلید یکتای صدور؛ نبود سابقه صدور ≠ ۰.') + TH('فروش تکمیل‌شده', 'event', 'مرز تکمیل (OPD-06) و انتساب (OPD-07) باز است.') + TH('وصول تأییدشده', 'event', 'جمع مبلغ مراحل معتبر؛ مغایرت‌دار شمرده نمی‌شود.');
  }
  function perfUnit() {
    var hist = st.pb === 'hist', ids = M.units.map(function (u) { return u.id; }).concat(['M']).filter(inScopeUnit);
    var cols = st.tb === 'snap' ? 3 : 4;
    var rows = ids.map(function (uid) {
      var u = uid === 'M' ? null : unit(uid), f = X.unit_f(uid, hist ? 'hist' : 'current'), cov = X.unitCov(uid), ss = X.sellersOfUnit(uid);
      var nm = uid === 'M' ? X.who('مستقیم زیر نظر شما', X.rel('dsup') + ' ' + X.rel('dseller')) : X.who('سرپرست ارشد ' + u.name, X.rel('senior') + (u.active ? '' : ' ' + pill('غیرفعال', 'slate', 'ban')));
      var cur = hist ? na('فقط در ساختار فعلی معنا دارد') : null;
      var cells = kpiCols(f, cov, { cases: cur || fa(X.unitWorkload(uid)), legacy: cur || fa(ss.reduce(function (a, s) { return a + s.legacy; }, 0)) }).map(function (c) { return '<td class="n">' + c + '</td>'; }).join('');
      return '<tr data-row="unit:' + uid + '" tabindex="-1">' + '<td>' + nm + '</td><td class="n">' + (cur || fa(ss.filter(function (s) { return s.active; }).length) + '<span class="of"> از ' + fa(ss.length) + '</span>') + '</td>' + cells + '<td>' + X.cov(cov) + '</td><td class="col-actions"><button type="button" class="btn btn-sm" data-act="drill:' + uid + '">سرپرستان</button></td></tr>';
    }).join('');
    var unk = '';
    if (hist && !st.pscope) { var uf = X.facts(function (i) { return i.teamAtIssue == null; }); var uc = st.tb === 'snap' ? '<td class="n">' + na('') + '</td><td class="n">' + fa(uf.openPre) + '</td><td class="n">' + na('') + '</td>' : '<td class="n">' + fa(uf.created) + '</td><td class="n">' + na('سابقه صدور ثبت نشده') + '</td><td class="n">' + fa(uf.completed) + '</td><td class="n">' + money(uf.collected) + '</td>'; unk = '<tr class="unk-row"><td>' + X.who('واحد ناشناخته', '<span class="scope-badge view">' + ic('question') + 'غیرقابل بازیابی</span>') + '</td><td class="n">' + na('') + '</td>' + uc + '<td>' + X.cov('undef', 'ناشناخته') + '</td><td class="col-actions"></td></tr>'; }
    var all = X.facts(function () { return true; });
    var tot = st.pscope ? '' : '<tfoot><tr class="tot"><th scope="row">جمع یکتا ' + hint('هر فاکتور فقط یک‌بار شمرده می‌شود؛ جمع از زیرجمع‌های هم‌پوشان ساخته نشده است. واحدهای دریافت‌نشده در جمع نیستند.', true) + '</th><td></td>' + (st.tb === 'snap' ? '<td></td><td class="n">' + fa(all.openPre) + '</td><td></td>' : '<td class="n">' + fa(all.created) + '</td><td></td><td class="n">' + fa(all.completed) + '</td><td class="n">' + money(all.collected) + '</td>') + '<td colspan="2"></td></tr></tfoot>';
    return { grain: st.tb === 'snap' ? 'پرونده/پیش‌فاکتور (لحظه) به تفکیک واحد' : 'فاکتور (رویداد) به تفکیک واحد',
      note: (st.tb === 'snap' ? '<b>وضعیت در لحظه:</b> عکس فوری از بار و پیش‌فاکتورهای باز؛ تاریخچه را بازسازی نمی‌کند.' : '<b>رویداد در بازه:</b> رویدادها بر اساس زمان وقوع خودشان؛ زمان ایجاد/تخصیص جایگزین آن نیست.') + (hist ? ' <b>انتساب تاریخی:</b> هر فاکتور به واحدی که هنگام صدور ثبت شده نسبت داده می‌شود؛ سابقه‌نداشته «ناشناخته» می‌ماند.' : ' <b>ساختار فعلی:</b> فروشنده‌ای که اخیراً جابه‌جا شده زیر واحد فعلی‌اش دیده می‌شود؛ اعتبار گذشته را با «انتساب تاریخی» ببینید.') + ' مقادیر هر ستون با تعریف رسمی در «تعریف شاخص‌ها» تطبیق دارند؛ cohort تجاری باز است.',
      html: '<div class="tbl-wrap"><table class="tbl" aria-label="عملکرد به تفکیک واحد"><thead><tr><th>واحد</th><th class="n">فروشنده فعال</th>' + kpiHead() + '<th>اعتماد به داده</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + unk + '</tbody>' + tot + '</table></div>' };
  }
  function perfSup() {
    var list = M.teams.filter(function (t) { return inScopeTeam(t.id); });
    var rows = list.map(function (t) {
      var hist = st.pb === 'hist', f = X.team_f(t.id, 'current'), ss = X.sellersOf(t.id), cov = X.teamCov(t.id);
      var cells = kpiCols(f, cov, { cases: fa(X.teamWorkload(t.id)), legacy: fa(ss.reduce(function (a, s) { return a + s.legacy; }, 0)) }).map(function (c) { return '<td class="n">' + c + '</td>'; }).join('');
      var par = t.parent === 'M' ? '<span class="muted">مستقیم زیر نظر شما</span>' : esc('سرپرست ارشد ' + unit(t.parent).name);
      return '<tr data-row="team:' + t.id + '" tabindex="-1"' + (t.active ? '' : ' class="inactive"') + '><td>' + X.who(t.sup, t.active ? X.rel(t.parent === 'M' ? 'dsup' : 'sup') : pill('غیرفعال', 'slate', 'ban')) + (t.active ? '' : '<div class="cell-sub">' + esc(t.inactiveNote) + ' · مسئول جایگزین ثبت نشده</div>') + '</td><td>' + par + '</td><td class="n">' + fa(ss.filter(function (s) { return s.active; }).length) + '<span class="of"> از ' + fa(ss.length) + '</span></td>' + cells + '<td>' + X.cov(cov) + '</td><td class="col-actions"><button type="button" class="btn btn-sm" data-act="drill:' + t.id + '">فروشندگان</button></td></tr>';
    }).join('');
    return { grain: st.tb === 'snap' ? 'پرونده/پیش‌فاکتور (لحظه) به تفکیک سرپرست' : 'فاکتور (رویداد) به تفکیک سرپرست', note: 'سرپرستان با والد فعلی‌شان نمایش داده می‌شوند. این نما <b>پاسخ‌گویی و بار</b> را نشان می‌دهد، نه امتیاز عملکرد؛ بدون رتبه و هدف.',
      html: '<div class="tbl-wrap"><table class="tbl" aria-label="عملکرد به تفکیک سرپرست"><thead><tr><th>سرپرست</th><th>والد فعلی</th><th class="n">فروشنده فعال</th>' + kpiHead() + '<th>اعتماد به داده</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div>' };
  }
  function perfSeller() {
    var list = M.sellers.filter(inScopeSeller);
    var rows = list.map(function (s) {
      var f = X.seller_f(s.id), cov = X.teamCov(s.team), bad = cov === 'failed';
      var teamCell = (s.team === 'direct' ? X.rel('dseller') : X.rel('indirect') + '<div class="cell-sub">' + esc(X.teamName(s.team)) + ' · ' + esc(X.unitName(X.unitOfTeam(s.team))) + '</div>') + (s.moved ? '<div class="cell-sub moved">' + ic('swap') + 'منتقل‌شده از ' + esc(X.teamName(s.moved.from)) + ' · ' + esc(s.moved.when) + '</div>' : '');
      var legacyCell = cov === 'partial' ? '<span class="partial-n tip" tabindex="0" data-tip="پوشش ناقص: عدد نهایی نیست">' + fa(s.legacy) + '</span>' : fa(s.legacy);
      var v = function (x) { return bad ? na('دریافت نشد') : x; };
      return '<tr data-row="seller:' + s.id + '" tabindex="-1"' + (s.active ? '' : ' class="inactive"') + '><td>' + X.who(s.name, s.active ? '' : '<span class="warn-n">غیرفعال</span>') + '</td><td>' + teamCell + '</td><td class="n">' + v(fa(s.open)) + '</td><td class="n">' + v(legacyCell) + '</td><td class="n">' + v(fa(f.created)) + '</td><td class="n">' + v(n0(f.openPre)) + '</td><td class="n">' + v(n0(f.completed)) + '</td><td class="n">' + v(f.created ? money(f.collected) : na('فاکتوری ثبت نشده')) + '</td><td class="col-actions"><button type="button" class="btn btn-sm" data-act="open-seller:' + s.id + '">جزئیات</button></td></tr>';
    }).join('');
    return { grain: 'فروشنده', note: 'ستون «فقط لیدهای قدیمی» عدد <b>کل پرونده‌ها نیست</b>. «۰» فقط یعنی محاسبه کامل شده و مقدار صفر است؛ داده دریافت‌نشده «—» می‌ماند.',
      html: (list.length ? '<div class="tbl-wrap"><table class="tbl" aria-label="عملکرد به تفکیک فروشنده"><thead><tr><th>فروشنده</th><th>رابطه و تیم فعلی</th><th class="n">پرونده باز</th><th class="n">فقط لیدهای قدیمی</th><th class="n">فاکتور صادرشده</th><th class="n">پیش‌فاکتور باز</th><th class="n">فروش تکمیل‌شده</th><th class="n">وصول تأییدشده</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div>' : h.stateBlock('noresult', 'فروشنده‌ای در این محدوده نیست', 'محدوده را به «همه واحدها» برگردانید.', '<button type="button" class="btn btn-soft" data-act="pscope:all">همه واحدها</button>')) };
  }
  function perfInv() {
    var hist = st.pb === 'hist', s0 = scopeSet();
    var list = M.invoices.filter(function (i) { var sl = seller(i.seller); return !s0 || (s0.units ? X.unitOfTeam(sl.team) === s0.units[0] : s0.teams ? sl.team === s0.teams[0] : sl.id === s0.sellers[0]); });
    var rows = list.map(function (i) {
      var s = seller(i.seller), tn = X.teamIdOfInv(i, 'current'), th = i.teamAtIssue;
      var thCell = th == null ? '<span class="pill t-slate">' + ic('question') + 'ناشناخته</span><div class="cell-sub">' + esc(i.teamNote || '') + '</div>' : esc(X.teamName(th));
      var differs = th != null && th !== tn;
      return '<tr data-row="inv:' + i.code + '" tabindex="-1"><td><span class="mono case-id">' + fa(i.code) + '</span><div class="cell-sub">' + X.link(i.link) + '</div></td><td>' + X.who(s.name, '<span class="muted">عامل رویداد صدور و مالک اعتبار</span>') + '</td>' +
        '<td class="' + (hist ? 'dimcol' : 'hl') + '">' + esc(X.teamName(tn)) + '</td><td class="' + (hist ? 'hl' : 'dimcol') + '">' + thCell + (differs ? '<div class="cell-sub moved">' + ic('swap') + 'با تیم امروز فرق دارد</div>' : '') + '</td>' +
        '<td>' + pill(X.INVS[i.inv].label, X.INVS[i.inv].tone, X.INVS[i.inv].icon) + '</td><td class="col-opt muted">' + esc(i.issued) + '</td><td class="col-actions"><button type="button" class="btn btn-sm" data-act="open-inv:' + i.code + '">جزئیات</button></td></tr>';
    }).join('');
    return { grain: 'فاکتور (رویداد صدور)', note: 'سه مفهوم جدا: <b>فروشنده</b> = عامل رویداد/مالک اعتبار، <b>تیم فعلی</b> = ساختار امروز، <b>تیم هنگام صدور</b> = انتساب تاریخی. تیم امروز جای تیم تاریخی نمی‌نشیند.',
      html: '<div class="tbl-wrap"><table class="tbl" aria-label="فاکتورها به تفکیک تیم"><thead><tr><th>فاکتور و اطمینان اتصال</th><th>فروشنده (عامل رویداد)</th><th>تیم فعلی <span class="sr">(ساختار فعلی)</span></th><th>تیم هنگام صدور <span class="sr">(انتساب تاریخی)</span></th><th>وضعیت فاکتور</th><th class="col-opt">صدور</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div>' };
  }
  function perfCase() {
    var rows = M.units.map(function (u) {
      var atSen = M.held.filter(function (c) { return c.holder.kind === 'senior' && c.holder.unit === u.id; }).length;
      var atSup = M.held.filter(function (c) { return c.holder.kind === 'sup' && team(c.holder.team).parent === u.id; }).length;
      var cov = X.unitCov(u.id);
      return '<tr data-row="unit:' + u.id + '" tabindex="-1"><td>' + X.who('سرپرست ارشد ' + u.name, X.rel('senior')) + '</td><td class="n">' + n0(atSen) + '</td><td class="n">' + n0(atSup) + '</td><td class="n">' + (cov === 'failed' ? na('دریافت نشد') : fa(X.unitWorkload(u.id))) + '</td><td>' + X.cov(cov) + '</td></tr>';
    }).join('');
    var own = '<tr class="own-row"><td>' + X.who('پنل من (' + M.user.name + ')', '<span class="scope-badge">' + ic('user') + 'شخصی · موجودی قابل توزیع</span>') + '</td><td class="n"><span class="muted">—</span></td><td class="n"><span class="muted">—</span></td><td class="n">' + fa(M.pool.length) + '</td><td>' + X.cov('ok') + '</td></tr>';
    return { grain: 'پرونده یکتا نزد نگهدارنده فعلی', note: 'هر پرونده فقط یک‌بار و نزد <b>مسئول فعلی</b> شمرده می‌شود (نه مالک اولیه). پرونده با چند انتقال یک واقعیت است. بار فعلی، رویداد انتقال و فاکتور صادرشده سه گروه شمارش جدا هستند.',
      html: '<div class="tbl-wrap"><table class="tbl" aria-label="بار پرونده"><thead><tr><th>نگهدارنده فعلی</th><th class="n">نزد خود سرپرست ارشد</th><th class="n">نزد سرپرستان</th><th class="n">نزد فروشندگان (جمع بار)</th><th>اعتماد به داده</th></tr></thead><tbody>' + rows + own + '</tbody></table></div>' };
  }
  // F04: same-looking number disagrees between two surfaces. Cause NOT VERIFIED → analytic Conflict/Incomplete, never a "corrected" count.
  function perfRecon() {
    var R = M.recon;
    var card = function (x, tone) { return '<div class="rc-card"><span class="rc-n">' + esc(x.name) + '</span><b class="rc-v">' + fa(x.value) + '</b><span class="rc-m">واحد شمارش: ' + esc(x.grain) + ' · ' + esc(x.rows) + '</span><span class="pill t-' + tone + '">' + ic('question') + 'عدد نهایی تأیید نشده</span></div>'; };
    var checks = R.checks.map(function (c) { return [c[2], c[1] + ' — هم‌خوانی تأیید نشده', c[3]]; });
    var html = '<div class="rc-grid">' + card(R.panel, 'amber') + '<div class="rc-vs" aria-hidden="true">' + ic('swap') + '</div>' + card(R.tab, 'amber') + '</div>' +
      '<div class="rc-body">' + X.sec('آیا این دو عدد قابل مقایسه‌اند؟', 'پیش از هر نتیجه‌گیری', X.checks(checks), 'primary') +
      X.sec('علت', 'اثبات‌نشده', '<p class="ind-note" style="padding:0">' + ic('question') + ' علت اختلاف <b>تأیید نشده</b> است. فرضیه‌ها فقط برای بررسی هستند، نه نتیجه: ' + R.hypotheses.map(function (x) { return '<span class="tag">' + esc(x) + '</span>'; }).join(' ') + '</p>') +
      X.sec('چه چیزی ممنوع است؟', '', X.checks([['no', 'تغییر مبلغ/وضعیت برای هماهنگی ظاهری', 'عدد دستی اصلاح نمی‌شود'], ['no', 'اعلام «شاخص نهایی» تا پایان تطبیق', 'هر دو عدد «نیازمند تطبیق» می‌مانند'], ['info', 'مسئول تطبیق', 'مالک داده/گزارش موجود؛ مدیر فقط مغایرت و واحد/گروه شمارش را می‌بیند.']])) +
      '<div class="note warn inset" style="margin:0 16px 16px">' + ic('alert') + '<span>' + esc(R.label) + '</span></div></div>';
    return { grain: 'مقایسه دو نما (بدون جمع)', note: 'اختلاف ' + fa(R.panel.value) + ' در برابر ' + fa(R.tab.value) + ' <b>صفر صحیح نیست</b>؛ حالت تحلیلی «تعارض/ناقص» است. ابتدا محدوده، فیلتر، واحد شمارش، زمان و منبع را هم‌سان کنید.', html: html };
  }

  /* ================= Invoices & Exceptions ================= */
  var IQ = [
    { id: 'action', label: 'نیازمند هماهنگی', icon: 'alert', tone: 'red', key: '1', f: function (i) { return i.bucket === 'action' || i.review === 'rejected' || (i.next && i.next.who === 'mis'); } },
    { id: 'finance', label: 'در بررسی مالی', icon: 'hourglass', tone: 'orange', key: '2', f: function (i) { return i.review === 'pending'; } },
    { id: 'rejected', label: 'رد شده', icon: 'xCircle', tone: 'red', key: '3', f: function (i) { return i.review === 'rejected'; } },
    { id: 'pre', label: 'پیش‌فاکتور', icon: 'file', tone: 'violet', key: '4', f: function (i) { return i.inv === 'pre'; } },
    { id: 'staged', label: 'مرحله‌ای ناتمام', icon: 'layers', tone: 'teal', key: '5', f: function (i) { return i.inv === 'staged'; } },
    { id: 'all', label: 'همه', icon: 'receipt', tone: 'neutral', key: '6', f: function () { return true; } }
  ];
  X.IQ = IQ;
  V.inv = function () {
    var q = IQ.filter(function (x) { return x.id === st.iq; })[0] || IQ[0], list = M.invoices.filter(q.f);
    var tabs = h.queues(IQ.map(function (x) { return { id: x.id, label: x.label, icon: x.icon, tone: x.tone, key: x.key, n: M.invoices.filter(x.f).length }; }), q.id, 'data-iq');
    var rows = list.map(function (i) {
      var s = seller(i.seller), rv = X.REV[i.review];
      return '<tr data-row="inv:' + i.code + '" tabindex="-1"><td><span class="mono case-id">' + fa(i.code) + '</span><div class="cell-sub">' + esc(i.customer) + '</div><div class="cell-sub">' + X.link(i.link) + '</div></td>' +
        '<td>' + X.who(s.name, esc(X.teamName(s.team)) + ' · ' + esc(X.unitName(X.unitOfTeam(s.team)))) + '</td>' +
        '<td>' + pill(X.INVS[i.inv].label, X.INVS[i.inv].tone, X.INVS[i.inv].icon) + '</td>' +
        '<td><b>مرحله ' + fa(i.stage) + ' از ' + fa(i.stages) + '</b><div class="cell-sub">' + esc(X.STG[i.stg]) + ' · ' + esc(X.EVID[i.evidence]) + '</div></td>' +
        '<td>' + (rv ? pill(rv.label, rv.tone, rv.icon) : '<span class="muted">ارسال نشده</span>') + '</td>' +
        '<td class="amt-col">' + money(i.paid) + '<div class="cell-sub">مانده: ' + money(i.inv === 'cancelled' ? 0 : i.total - i.paid) + '</div></td>' +
        '<td>' + X.nextActor(i.next, i) + (i.reason ? '<div class="why-line">' + ic('alert') + esc(i.reason) + '</div>' : '') + (i.mismatch ? '<div class="why-line warn">' + ic('question') + esc(i.mismatch) + '</div>' : '') + '</td>' +
        '<td class="col-actions"><button type="button" class="btn btn-sm" data-act="open-inv:' + i.code + '">جزئیات</button></td></tr>';
    }).join('');
    var kpis = [
      { label: 'فاکتورهای محدوده', value: fa(M.invoices.length), color: 'neutral', keep: true, meaning: 'فاکتور یکتا در محدوده مجاز شما', basis: 'وضعیت در لحظه' },
      { label: 'نیازمند هماهنگی', value: fa(M.invoices.filter(IQ[0].f).length), color: 'red', tone: 'bad', filter: 'iq:action', meaning: 'رد مالی یا مغایرت؛ اقدام بعدی با فروشنده یا MIS', basis: 'وضعیت در لحظه' },
      { label: 'در بررسی مالی', value: fa(M.invoices.filter(IQ[1].f).length), color: 'orange', filter: 'iq:finance', meaning: 'منتظر واحد مالی؛ اقدام شما لازم نیست', basis: 'وضعیت در لحظه' },
      { label: 'فروش تکمیل‌شده', value: fa(X.facts(function () { return true; }).completed), color: 'green', meaning: 'پرداخت کامل و تأیید مالی؛ صدور یا رسید به‌تنهایی کافی نیست', basis: 'رویداد · مرز تکمیل: OPD-06' }
    ];
    return h.pageHead({ title: 'فاکتورها و استثناها', sub: 'وضعیت فاکتور، مرحله پرداخت و بررسی مالی سه چیز جدا هستند · تأیید/رد/بازپرداخت مالی از این پنل ممکن نیست', kpis: kpis, fresh: X.freshPart('فاکتورها') }) + X.banners() +
      '<section class="panel main">' + tabs + h.toolbar('کد فاکتور، مشتری یا فروشنده', [{ label: 'واحد', icon: 'users' }, { label: 'بازه صدور', icon: 'clock' }]) +
      (list.length ? '<div class="tbl-wrap"><table class="tbl" aria-label="فاکتورهای ' + esc(q.label) + '"><thead><tr><th>فاکتور و اتصال</th><th>فروشنده و واحد فعلی</th><th>وضعیت فاکتور</th><th>مرحله پرداخت</th><th>بررسی مالی</th><th>پرداخت‌شده / مانده</th><th>اقدام بعدی و دلیل</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div>' + h.tfoot(list.length, list.length)
        : h.stateBlock('empty', 'فاکتوری در این صف نیست', 'صف دیگری را انتخاب کنید.', '<button type="button" class="btn btn-soft" data-iq="all">همه فاکتورها</button>')) + '</section>';
  };

  /* ================= Archive Explorer (five reasons, distinct rules) ================= */
  V.arch = function () {
    var R = M.archReasons, keys = ['noanswer', 'assess', 'sub', 'product', 'productx'];
    var cnt = function (k) { return M.archives.filter(function (a) { return a.reason === k; }).length; };
    var qs = h.queues([{ id: 'all', label: 'همه علت‌ها', icon: 'inbox', tone: 'neutral', n: M.archives.length, key: '1' }].concat(keys.map(function (k, i) { return { id: k, label: R[k].label, icon: X.ARC[k][2], tone: X.ARC[k][1], n: cnt(k), key: String(i + 2) }; })), st.ar, 'data-ar');
    var list = M.archives.filter(function (a) { return st.ar === 'all' || a.reason === st.ar; });
    var rows = list.map(function (a) {
      var s = seller(a.seller), rc = X.ARC[a.reason];
      var fin = a.fin ? money(a.fin.total) + '<div class="cell-sub">پرداخت‌شده: ' + money(a.fin.paid) + ' · مانده: ' + money(a.fin.total - a.fin.paid) + '</div>' : '<span class="muted">بدون وابستگی مالی ثبت‌شده</span>';
      var res = a.resolve === 'late-payment' ? pill('حل با پرداخت دیرهنگام ممکن', 'teal', 'checkCircle') + '<div class="cell-sub">طبق قاعده موجود · اجرای واقعی تأیید نشده</div>' : a.resolve === 'partial-pay' ? pill('پرداخت جزئی ≠ احیا', 'amber', 'question') : '<span class="pill t-slate">' + ic('question') + 'قابل اثبات نیست</span>';
      return '<tr data-row="arch:' + a.id + '" tabindex="-1"><td><span class="mono case-id">' + esc(a.caseRef) + '</span><div class="cell-sub">' + X.src(a.src) + ' ' + X.link(a.link) + '</div></td><td>' + pill(rc[0], rc[1], rc[2]) + '</td><td class="n">' + fa(a.age) + ' روز<div class="cell-sub">بایگانی: ' + esc(a.archived) + '</div></td><td>' + X.who(s.name, esc(X.teamName(s.team))) + '</td><td class="amt-col">' + fin + '</td><td><span class="owner-shown">' + esc(a.owner[0]) + ': ' + esc(a.owner[1]) + '</span><div class="cell-sub">مالک نمایش‌داده‌شده؛ مسئول فعلی یا مالک اعتبار نیست</div></td><td>' + res + '</td><td class="col-actions"><button type="button" class="btn btn-sm" data-act="open-arch:' + a.id + '">بررسی</button></td></tr>';
    }).join('');
    var rule = st.ar !== 'all' ? '<div class="note info inset">' + ic('info') + '<span><b>قاعده ' + esc(R[st.ar].label) + ':</b> ' + esc(R[st.ar].rule) + ' این قاعده با علت‌های دیگر یکی نمی‌شود و فیلتر سن، آستانه جدیدی برای گردش‌کار نمی‌سازد.</span></div>' : '';
    return h.pageHead({ title: 'کاوشگر آرشیو', sub: 'آرشیو حذف نیست و وابستگی مالی را آزاد نمی‌کند · ابتدا مشاهده و بررسی؛ احیا مشروط و غیرفعال', kpis: [
      { label: 'پرونده آرشیوی در برش فعلی', value: fa(M.archives.length), color: 'blue', meaning: 'تعداد نمونه این محیط، نه معیار بار کار؛ سقف نمایش ۲۰۰ ردیف', basis: 'وضعیت در لحظه · برش نمونه' },
      { label: 'قابل بررسی برای حل با پرداخت', value: fa(M.archives.filter(function (a) { return a.resolve === 'late-payment'; }).length), color: 'teal', meaning: 'فقط یعنی قاعده پرداخت دیرهنگام وجود دارد؛ اجرا تأیید نشده', basis: 'وضعیت در لحظه' }
    ], fresh: { text: 'عکس لحظه‌ای آرشیو: ' + M.freshness.now, stale: false } }) + X.banners() +
      '<section class="panel main">' + qs + '<div class="note warn inset">' + ic('rows') + '<span><b>پوشش محدود.</b> فهرست آرشیو پرداخت حداکثر ۲۰۰ ردیف و فقط ردیف‌های حل‌نشده در محدوده مدیر را نشان می‌دهد؛ نبودِ موردِ بیشتر یا در دسترس‌نبودن منبع «خالی قطعی» نیست. مسیر بی‌پاسخ منبع و محدوده خودش را دارد.</span></div>' + rule +
      (list.length ? '<div class="tbl-wrap"><table class="tbl" aria-label="آرشیو"><thead><tr><th>پرونده و منبع</th><th>علت</th><th class="n">سن از زمان بایگانی</th><th>فروشنده / تیم</th><th>وابستگی مالی</th><th>مالک نمایش‌داده‌شده</th><th>قابلیت حل / احیا</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div>' + h.tfoot(list.length, list.length)
        : h.stateBlock('empty', 'در این برش و تا سقف نمایش، موردی نیست', 'این فقط برش نمونه (حداکثر ۲۰۰ ردیف) را توصیف می‌کند؛ ' + esc(R[st.ar] ? R[st.ar].label : '') + ' در محدوده کامل ممکن است مورد دیگری داشته باشد.', '<button type="button" class="btn btn-soft" data-ar="all">همه علت‌ها</button>')) +
      '<div class="bulk-note">' + ic('lock') + '<span>بررسی گروهی، احیا، تخصیص مجدد و حل گروهی: <b>نیازمند اعتبارسنجی / معوق</b> (M-G09، M-G10). حذف: در این نقش مجاز نیست.</span></div></section>';
  };

  /* ================= Extra-number Requests ================= */
  V.xnum = function () {
    var decided = function (r) { return st.xDone[r.id] ? st.xDone[r.id] : r.state; }, isOpen = function (d) { return d === 'pending' || d === 'unknown'; };
    var QS = [{ id: 'pending', label: 'منتظر تصمیم شما', icon: 'hourglass', tone: 'orange', key: '1', n: M.xreq.filter(function (r) { return isOpen(decided(r)); }).length }, { id: 'decided', label: 'تصمیم‌گرفته‌شده', icon: 'history', tone: 'neutral', key: '2', n: M.xreq.filter(function (r) { return !isOpen(decided(r)); }).length }];
    var list = M.xreq.filter(function (r) { return st.xq === 'pending' ? isOpen(decided(r)) : !isOpen(decided(r)); });
    var rows = list.map(function (r) {
      var s = seller(r.seller), d = decided(r), S = X.XRS[d];
      return '<tr data-row="xreq:' + r.id + '" tabindex="-1"><td><span class="mono case-id">' + r.id + '</span><div class="cell-sub">' + esc(r.at) + '</div></td><td>' + X.who(s.name, esc(X.teamName(s.team)) + (s.active ? '' : ' · <span class="warn-n">غیرفعال</span>')) + (r.flag ? '<div class="why-line warn">' + ic('question') + esc(r.flag) + '</div>' : '') + '</td><td><span class="mono phone-num tip" tabindex="0" data-tip="شماره فقط برای جستجوست و هویت پرونده نیست">' + fa(r.phone) + '</span>' + (r.name ? '<div class="cell-sub">' + esc(r.name) + (r.note ? ' · ' + esc(r.note) : '') + '</div>' : '<div class="cell-sub">نام/یادداشت ثبت نشده</div>') + '</td>' +
        '<td><span class="chk-ok">' + ic('checkCircle') + 'تکراری نیست</span><div class="cell-sub">فقط همان فروشنده در pending/approved</div></td><td>' + pill(S.label, S.tone, S.icon) + (!isOpen(d) ? '<div class="cell-sub">' + esc(st.xDone[r.id] ? 'شما · همین الان' : r.by) + '</div>' : d === 'unknown' ? '<div class="cell-sub">پاسخ نرسید؛ پیش از تکرار تطبیق کنید</div>' : '') + '</td>' +
        '<td>' + (d === 'approved' ? '<span class="mono">' + esc(r.lead || 'L-نمونه') + '</span><div class="cell-sub">سرنخ قدیمی · نگاشت CaseRef: ' + X.link('unresolved') + '</div>' : d === 'rejected' ? (r.reason ? '<span>' + esc(r.reason) + '</span>' : '<span class="muted">دلیل ثبت نشده</span>') : '<span class="muted">—</span>') + '</td><td class="col-actions"><button type="button" class="btn btn-sm' + (d === 'pending' ? ' btn-soft' : '') + '" data-act="open-xreq:' + r.id + '">' + (isOpen(d) ? (d === 'unknown' ? 'تطبیق و بررسی' : 'بررسی و تصمیم') : 'جزئیات') + '</button></td></tr>';
    }).join('');
    return h.pageHead({ title: 'درخواست شماره اضافه', sub: 'درخواست فروشنده برای افزودن شماره — نه ویرایش یک شماره موجود · تصمیم فقط برای ردیف‌هایی که مدیر تعیین‌شده شماست', kpis: [
      { label: 'منتظر تصمیم شما', value: fa(QS[0].n), color: 'orange', keep: true, meaning: 'ردیف‌هایی با مدیر تعیین‌شده = شما؛ جایگاه سازمانی به‌تنهایی کافی نیست', basis: 'وضعیت در لحظه' }
    ], fresh: X.freshPart('صف') }) + X.banners() +
      '<section class="panel main">' + h.queues(QS, st.xq, 'data-xq') + '<div class="note warn inset">' + ic('rows') + '<span><b>سقف نمایش.</b> فقط آخرین ۱۰۰ ردیف دیده می‌شود؛ کامل‌بودن تاریخچه اثبات نشده. بررسی گروهی ندارد (نیازمند اعتبارسنجی، M-G10) — هر درخواست جداگانه تصمیم‌گیری می‌شود.</span></div>' +
      (list.length ? '<div class="tbl-wrap"><table class="tbl" aria-label="درخواست‌های شماره اضافه"><thead><tr><th>درخواست</th><th>فروشنده درخواست‌دهنده</th><th>شماره (فقط جستجو) و یادداشت</th><th>بررسی تکراری ' + hint('فقط همان فروشنده + شماره نرمال‌شده در درخواست‌های pending/approved. تکراری بین فروشندگان یا منابع دیگر بررسی نمی‌شود.', true) + '</th><th>وضعیت و تصمیم‌گیرنده</th><th>نتیجه / دلیل</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div>' + h.tfoot(list.length, list.length)
        : h.stateBlock('empty', st.xq === 'pending' ? 'درخواستی منتظر تصمیم شما نیست' : 'درخواست تصمیم‌گرفته‌شده‌ای در این برش نیست', 'خالی بودن صف به معنی نبود قابلیت یا اختیار تصمیم نیست.', '')) +
      '<div class="bulk-note">' + ic('info') + '<span>اگر برای درخواستی مدیر بالادستی پیدا نشود، درخواست ثبت می‌شود و مسیر Admin مطرح است؛ مدیر آن را خودکار تصاحب نمی‌کند و این صف آن را نشان نمی‌دهد.</span></div></section>';
  };

  /* ================= Customer behaviour (contextual view of the Shared Customer Profile) ================= */
  var CS = [['partial', 'ناقص'], ['residual', 'بارگذاری باقی‌مانده'], ['loaded', 'بارگذاری شد'], ['error', 'خطا'], ['noresult', 'بدون نتیجه'], ['empty', 'خالی مجاز'], ['stale', 'قدیمی'], ['loading', 'در حال بارگذاری']];
  V.cust = function () {
    var K = M.cust, s = st.custSt;
    var seg = '<div class="seg" role="group" aria-label="حالت نمایش تاریخچه (دمو)">' + CS.map(function (c) { return '<button type="button" data-cs="' + c[0] + '" aria-pressed="' + (s === c[0]) + '">' + c[1] + '</button>'; }).join('') + '</div>';
    var rows = K.events.map(function (e) { return '<tr><td class="muted"><span class="num">' + esc(e.t) + '</span></td><td>' + pill('صدور توسط سامانه', 'slate', 'layers') + '<div class="cell-sub">منبع: ' + esc(e.src) + '</div></td><td>' + esc(e.text) + '</td><td class="muted">فاکتور ' + fa(K.inv) + '</td></tr>'; }).join('');
    var tbl = '<div class="tbl-wrap"><table class="tbl no-cursor" aria-label="رویدادهای مرتبط"><thead><tr><th>زمان رویداد</th><th>نوع / منبع</th><th>رویداد</th><th>زمینه</th></tr></thead><tbody>' + rows + '</tbody></table></div>';
    var pop = '<div class="note info inset">' + ic('layers') + '<span><b>دو جمعیت متفاوت.</b> رویدادهای «اقدام مشتری» برای این فاکتور: ' + fa(K.customerEvents.length) + ' · ورودی‌های «صدور توسط سامانه» در جزئیات: ' + fa(K.events.length) + '. این دو لزوماً یک جمعیت نیستند؛ اختلاف شمارنده در حال اعتبارسنجی است (M-G03) و «مشتری رفتاری نداشته» نتیجه گرفته نمی‌شود.</span></div>';
    var body;
    if (s === 'loading') body = '<div class="state" role="status"><div class="spin-ico" aria-hidden="true"></div><b>در حال دریافت تاریخچه…</b><span>درخواست خواندن هنوز در جریان است؛ هنوز چیزی درباره «وجود یا نبود رفتار» نمی‌توان گفت.</span></div>';
    else if (s === 'error') body = h.stateBlock('error', 'دریافت تاریخچه ناموفق بود', 'این به معنی «بدون فعالیت» نیست. خواندن را دوباره امتحان کنید؛ سابقه‌ای ساخته نمی‌شود.', '<button type="button" class="btn btn-primary" data-act="cust-retry">' + ic('refresh') + 'تلاش مجدد برای خواندن</button>');
    else if (s === 'noresult') body = h.stateBlock('noresult', 'با این فیلتر رویدادی نیست', 'فیلتر معتبر است ولی چیزی را تطبیق نمی‌دهد؛ این با «خالی بودن کل تاریخچه» فرق دارد.', '<button type="button" class="btn btn-soft" data-cs="partial">پاک‌کردن فیلتر</button>');
    else if (s === 'empty') body = h.stateBlock('empty', 'تاریخچه کامل و مجاز است و رویدادی در این جمعیت ندارد', 'محدوده و پرس‌وجو کامل بوده‌اند؛ هیچ رویدادی از جمعیت تعریف‌شده (اقدام مشتری) ثبت نشده است. هنوز به معنی قصد خرید یا نخریدن نیست.', '');
    else if (s === 'stale') body = h.banner('stale', '<b>این رویدادها از حافظه قدیمی هستند (' + M.freshness.stale + ').</b> زمان و پوشش در هر ردیف دیده می‌شود؛ پیش از تصمیم بازخوانی کنید.', '<button type="button" class="btn btn-sm" data-refresh>' + ic('refresh') + 'بازخوانی</button>') + pop + tbl;
    else if (s === 'partial') body = h.banner('incomplete', '<b>تاریخچه ناقص است:</b> فقط بخشی از حوزه‌های رویداد دریافت شده (صدور سیستم). حوزه «اقدام مشتری» دریافت نشده یا خالی است و پوشش باقی‌مانده مشخص نیست.', '<button type="button" class="btn btn-sm" data-act="cust-retry">' + ic('refresh') + 'تلاش مجدد</button>') + pop + tbl;
    else if (s === 'residual') body = '<div class="resid" role="status">' + ic('clock') + '<span><b>داده رسیده، اما نشانگر بارگذاری هنوز فعال است.</b> این حالت موفقیت کامل نیست (F05 باقی‌مانده). نمایش زیر آنچه واقعاً رسیده را نشان می‌دهد؛ علت ریشه‌ای اثبات نشده.</span><span class="spin-sm" aria-hidden="true"></span></div>' + pop + tbl;
    else body = pop + tbl;
    return h.pageHead({ title: 'زمینه رفتار مشتری', sub: 'نمای زمینه‌ای از پروفایل مشتریِ مشترک — فقط تاریخچه معتبر در محدوده فاکتور/مشتری مجاز · نه پایگاه مشتری موازی', fresh: X.freshPart('تاریخچه'), scope: 'فاکتور ' + fa(K.inv) + ' · ' + esc(K.customer) }) + X.banners() +
      '<section class="panel main"><div class="toolbar"><span class="muted">حالت نمایش (دمو):</span>' + seg + '</div><div class="note info inset">' + ic('shield') + '<span>«رفتار مشاهده‌شده» به معنی قصد خرید، تلاش تماس یا موفقیت پرداخت نیست. رویداد ساخته نمی‌شود، امتیازدهی به مشتری وجود ندارد و شماره موبایل هویت پرونده نیست.</span></div>' + body + '</section>';
  };

  /* ================= Wallet (Own Account) ================= */
  V.wallet = function () {
    var W = M.wallet, rows = W.tx.map(function (t) { return '<tr><td class="muted"><span class="num">' + esc(t.date) + '</span></td><td>' + pill('بستانکار', 'green', 'plus') + '</td><td><span class="tag' + (t.channel === 'آنلاین' ? ' teal' : '') + '">' + esc(t.channel) + '</span></td><td>' + esc(t.desc) + '<div class="muted" style="font-size:var(--t-micro)">' + esc(t.rule) + '</div></td><td style="text-align:left"><span class="credit">+<span class="num">' + num(t.amount) + '</span></span> <small class="muted">تومان</small></td></tr>'; }).join('');
    return h.pageHead({ title: 'کیف پول من', sub: 'حساب شخصی و فقط‌خواندنی — فقط حق‌الزحمه خودتان؛ درآمد یا پورسانت تیم‌ها و مالی سازمان اینجا نیست' }) +
      '<div class="wallet-sum"><div class="wcard hero"><span>موجودی ثبت‌شده</span><b><span class="num">' + num(W.balance) + '</span><small>تومان</small></b><div class="foot"><span>بدون تسویه‌نشده‌ی محاسبه‌شده</span></div></div><div class="wcard"><span>کل بستانکاری</span><b><span class="num">' + num(W.credit) + '</span><small>تومان</small></b><div class="foot"><span>' + fa(W.tx.length) + ' تراکنش</span></div></div><div class="wcard"><span>کل برداشت / تسویه</span><b><span class="num">' + num(W.debit) + '</span><small>تومان</small></b><div class="foot"><span>تسویه‌ای ثبت نشده</span></div></div></div>' +
      '<section class="panel main">' + h.banner('locked', '<b>خواندن محض.</b> باز کردن این صفحه هیچ اعتبار، ثبت یا محاسبه‌ای انجام نمی‌دهد (F07؛ فعال‌سازی فنی نیازمند جداسازی وابستگی). «ثبت‌شده» با «قابل تسویه» و «تسویه‌شده» یکی نیست.', '') + '<div class="tbl-wrap"><table class="tbl no-cursor"><thead><tr><th>تاریخ</th><th>نوع</th><th>کانال</th><th>شرح</th><th style="text-align:left">مبلغ</th></tr></thead><tbody>' + rows + '</tbody></table></div>' + h.tfoot(W.tx.length, W.tx.length) + '</section>';
  };

  /* ================= Shared Reports (Manager scope) ================= */
  V.rep = function () {
    var teamOpts = '<option value="all"' + (st.rteam === 'all' ? ' selected' : '') + '>همه واحدهای محدوده من</option>' + M.units.map(function (u) { return '<option value="' + u.id + '"' + (st.rteam === u.id ? ' selected' : '') + '>سرپرست ارشد ' + esc(u.name) + '</option>'; }).join('') + '<option value="M"' + (st.rteam === 'M' ? ' selected' : '') + '>مستقیم زیر نظر من</option>';
    var ctrl = '<div class="rep-ctrl"><div class="seg" role="group" aria-label="مبنای زمانی"><button type="button" data-rbasis="event" aria-pressed="' + (st.rbasis === 'event') + '">رویداد در بازه</button><button type="button" data-rbasis="now" aria-pressed="' + (st.rbasis === 'now') + '">وضعیت فعلی</button><button type="button" disabled aria-disabled="true" class="tip" data-tip="وضعیت تاریخی فقط با سابقه معتبر ممکن است؛ در این نمونه سابقه کافی ثبت نشده">وضعیت تاریخی</button></div>' +
      (st.rbasis === 'event' ? '<label class="lbl inline">از<input class="input" value="۱۴۰۵/۰۷/۰۱"></label><label class="lbl inline">تا<input class="input" value="۱۴۰۵/۰۷/۱۰"></label>' : '<span class="muted">وضعیت فعلی هنگام اجرای گزارش</span>') +
      '<label class="lbl inline">محدوده<select class="input" data-rteam>' + teamOpts + '</select></label><span class="scope-note">فقط زیرمجموعه مجاز شما فهرست می‌شود؛ نقش‌های بالاتر یا مالی انتخاب‌شدنی نیستند.</span></div>';
    var body;
    if (st.report) {
      var r = M.reports.filter(function (x) { return x.id === st.report; })[0];
      var lst = M.invoices.filter(function (i) { if (st.rteam === 'all') return true; return X.unitOfTeam(seller(i.seller).team) === st.rteam; });
      var rows = lst.map(function (i, n) { var sl = seller(i.seller); return '<tr><td><span class="mono">' + fa(i.code) + '</span></td><td>' + esc(i.customer) + '</td><td>' + esc(sl.name) + '</td><td>' + esc(X.teamName(sl.team)) + (n === 2 && r.id === 'invoices_register' ? '<div class="cell-sub why-line warn">' + ic('question') + 'برچسب سرپرست/فروشنده نیازمند اعتبارسنجی</div>' : '') + '</td><td>' + money(i.total) + '</td><td>' + pill(X.INVS[i.inv].label, X.INVS[i.inv].tone, X.INVS[i.inv].icon) + '</td><td class="muted">' + esc(i.issued) + '</td></tr>'; }).join('');
      var cov = sim() === 'incomplete' ? 'partial' : sim() === 'stale' ? 'stale' : 'ok';
      body = '<section class="panel main"><div class="rep-head"><button type="button" class="btn btn-ghost btn-sm" data-act="close-report">' + ic('arrowL') + 'همه گزارش‌ها</button><h2>' + esc(r.name) + '</h2><span class="grow"></span>' + h.freshness({ text: 'محاسبه‌شده: امروز ۱۰:۴۲', refresh: true }) + '<button type="button" class="btn btn-sm" disabled aria-disabled="true">' + ic('download') + 'خروجی (مشروط · مجوز تأیید نشده)</button></div>' +
        '<div class="rep-meta"><span class="chip-m">سطح: ' + esc(r.grain) + '</span><span class="chip-m">مبنا: ' + esc(st.rbasis === 'now' ? 'وضعیت فعلی' : r.basis) + '</span><span class="chip-m">محدوده: ' + (st.rteam === 'all' ? 'همه واحدهای مجاز' : esc(X.unitName(st.rteam))) + '</span><span class="chip-m">تعریف شاخص: نسخه ۱</span><span class="chip-m">واحد پول: تومان</span><span class="chip-m">' + X.cov(cov) + '</span><span class="chip-m">' + X.cov('bounded', 'نتیجه محدود ۶۰ تا ۱۲۰ ردیف') + '</span><span class="chip-m">خروجی: مجوز و فیلدها تأیید نشده</span></div>' +
        '<div class="note warn inset">' + ic('rows') + '<span><b>نتیجه محدود.</b> درخواست گزارش فقط ۶۰ تا ۱۲۰ ردیف برمی‌گرداند؛ این <b>کل قلمرو نیست</b> و با پوشش V4 + قدیمی کامل‌بودن منابع یا حذف تکرار قطعی اثبات نمی‌شود. اگر شمار گزارش با برگه فاکتورها نخواند، به «عملکرد و تطبیق ← تطبیق شاخص» بروید (F04).</span></div>' +
        '<div class="kpis rep-kpis">' + h.kpi({ label: 'تعداد در این برش', value: fa(lst.length), color: 'neutral', meaning: 'تعداد ' + r.grain + ' یکتا در برش نمایش‌داده‌شده؛ جمع کل نیست', basis: 'همان محدوده جدول' }) + h.kpi({ label: 'جمع مبلغ این برش', value: '<span class="num">' + num(lst.reduce(function (a, i) { return a + i.total; }, 0)) + '</span><small>تومان</small>', color: 'teal', meaning: 'جمع همان ردیف‌ها با همان تعریف', basis: 'همان محدوده جدول' }) + '</div>' +
        (lst.length ? '<div class="tbl-wrap"><table class="tbl no-cursor" aria-label="' + esc(r.name) + '"><thead><tr><th>کد</th><th>مشتری</th><th>فروشنده</th><th>تیم فعلی</th><th>مبلغ</th><th>وضعیت فاکتور</th><th>تاریخ صدور</th></tr></thead><tbody>' + rows + '</tbody></table></div>' + h.tfoot(lst.length, lst.length) : h.stateBlock('noresult', 'ردیفی در این محدوده نیست', 'محدوده را تغییر دهید؛ دسترسی بیرون از محدوده مجاز ایجاد نمی‌شود.', '<button type="button" class="btn btn-soft" data-act="rteam-all">همه واحدها</button>')) + '</section>';
    } else {
      body = '<div class="rep-grid">' + M.reports.map(function (r) { return '<button type="button" class="rep-card" data-act="open-report:' + r.id + '"><span class="rep-ico">' + ic(r.icon) + '</span><b>' + esc(r.name) + '</b><span class="rep-tags"><span>سطح: ' + esc(r.grain) + '</span><span>مبنا: ' + esc(r.basis) + '</span></span><span class="rep-go">مشاهده آمار و ریزداده ' + ic('arrowL') + '</span></button>'; }).join('') + '</div>';
    }
    return h.pageHead({ title: 'گزارش‌ها', sub: 'ورودی واحد گزارش‌های مشترک CRM با محدوده مدیر · داشتن گزارش مالی به معنی اقدام مالی نیست · هیچ گزارش یا خروجی اجرا نمی‌شود' }) + '<section class="panel rep-ctrl-panel">' + ctrl + '</section>' + body;
  };
})();
