/* Senior Supervisor — role layer, part 2: read/analysis views (overview, explorer, ready, invoices, cases, reports, personal).
   Everything here is presentation of Product-Spec-approved facts; undefined metrics render as «تعریف نشده», never as 0. */
(function () {
  'use strict';
  var X = window.SENX, C = X.C, M = X.M, h = X.h, st = X.st, V = X.V;
  var esc = h.esc, fa = h.fa, num = h.num, money = h.money, ic = h.ic, pill = h.pill, hint = h.hint;
  var sim = X.sim, seller = X.seller, team = X.team;
  var na = function (t) { return '<span class="na tip" tabindex="0" data-tip="' + esc(t || 'داده در دسترس نیست') + '">—</span>'; };
  var n0 = function (v) { return v ? fa(v) : '<span class="muted">۰</span>'; };

  function topExc(tid) {
    if (tid === 'T4') return { t: 'مسئول جایگزین ثبت نشده', a: 'منابع انسانی / مدیر فروش', tone: 'amber' };
    var rej = X.sellersOf(tid).some(function (s) { return M.invoices.some(function (i) { return i.seller === s.id && i.review === 'rejected'; }); });
    if (rej) return { t: 'فیش ردشده', a: 'فروشنده', tone: 'red' };
    if (M.ready.some(function (r) { return r.team === tid && r.status === 'ready'; })) return { t: 'آماده تبدیل در انتظار', a: 'سرپرست تیم', tone: 'violet' };
    if (M.invoices.some(function (i) { return X.seller(i.seller).team === tid && i.review === 'pending'; })) return { t: 'در انتظار بررسی مالی', a: 'واحد مالی', tone: 'orange' };
    return null;
  }

  /* ================= Multi-Team Overview ================= */
  V.ov = function () {
    var A = X.attention(), failed = function (tid) { return X.teamCov(tid) === 'failed'; };
    var okTeams = M.teams.filter(function (t) { return !failed(t.id); });
    var allSellers = M.sellers, actAll = allSellers.filter(function (s) { return s.active; }).length;
    var wl = okTeams.reduce(function (a, t) { return a + X.teamWorkload(t.id); }, 0) + X.teamWorkload('direct');
    var pre = okTeams.reduce(function (a, t) { return a + X.team_f(t.id, 'current').openPre; }, 0) + X.team_f('direct', 'current').openPre;
    var bad = sim() === 'teamfail' ? 'unavailable' : sim() === 'stale' ? 'stale' : null;
    var kpis = [
      { label: 'سرپرست مستقیم فعال', value: fa(M.teams.filter(function (t) { return t.active; }).length) + '<small>از ' + fa(M.teams.length) + '</small>', color: 'teal', keep: true, meaning: 'سرپرستانی که مستقیم زیر نظر شما هستند', basis: 'ساختار فعلی' },
      { label: 'فروشنده فعال', value: fa(actAll) + '<small>از ' + fa(allSellers.length) + '</small>', color: 'neutral', keep: true, meaning: 'مستقیم و غیرمستقیم؛ مشاهده، اجازه نوشتن نمی‌سازد', basis: 'ساختار فعلی' },
      { label: 'بار پرونده تیم‌ها', value: fa(wl), color: 'blue', state: bad, meaning: 'پرونده‌های فعال نزد تیم‌ها (مالک فعلی)؛ موجودی پنل شما جداست', basis: 'وضعیت در لحظه' + (sim() === 'teamfail' ? ' · بدون یک تیم' : '') },
      { label: 'پیش‌فاکتور باز', value: fa(pre), color: 'violet', state: bad, filter: 'perf:team', meaning: 'پیش‌فاکتورهای فعلی بدون تکمیل یا لغو', basis: 'وضعیت در لحظه' },
      { label: 'موجودی پنل من', value: fa(M.pool.length), color: 'neutral', keep: true, filter: 'dist:assign', meaning: 'پرونده‌هایی که به شما تحویل شده و هنوز تخصیص نیافته‌اند؛ جدا از بار تیم‌ها', basis: 'وضعیت در لحظه' }
    ];
    var attn = '<section class="panel attn-panel" aria-labelledby="attn-h"><div class="sec-h ph"><h2 id="attn-h">' + ic('inbox') + 'نیازمند هماهنگی</h2><span class="aside">' + fa(A.length) + ' دسته</span></div>' +
      (A.length ? '<ul class="attn">' + A.map(function (a) {
        return '<li><button type="button" class="attn-row" data-attn="' + a.id + '" style="--c:var(--' + a.tone + '-dot)"><span class="attn-ico t-' + a.tone + '">' + ic(a.icon) + '</span><span class="attn-txt"><b>' + esc(a.label) + '</b><span><span class="attn-own">مسئول: ' + esc(a.owner) + '</span>' + esc(a.state) + ' · ' + esc(a.subject) + '</span></span><span class="attn-n" aria-label="' + fa(a.n) + ' مورد">' + fa(a.n) + '</span>' + ic('arrowL') + '</button></li>';
      }).join('') + '</ul>' : h.stateBlock('empty', 'موردی نیازمند هماهنگی نیست', 'استثنای ثبت‌شده‌ای در تیم‌های شما وجود ندارد.')) +
      '<div class="attn-foot">' + ic('info') + '<span>این فهرست فقط استثناهای دارای داده معتبر را نشان می‌دهد؛ کانال «ارجاع» رسمی تعریف نشده و دستوری برای آن وجود ندارد.</span></div></section>';

    var rows = M.teams.map(function (t) {
      var cov = X.teamCov(t.id), f = X.team_f(t.id, 'current'), ss = X.sellersOf(t.id), act = ss.filter(function (s) { return s.active; }).length, w = X.teamWorkload(t.id);
      var maxW = Math.max.apply(null, M.teams.map(function (x) { return X.teamWorkload(x.id); }));
      var ex = topExc(t.id), rd = M.ready.filter(function (r) { return r.team === t.id && r.status === 'ready'; }).length;
      if (cov === 'failed') {
        return '<tr data-row="team:' + t.id + '" tabindex="-1" class="failed"><td>' + X.who(t.sup, X.rel('sup')) + '</td><td colspan="5"><div class="row-state">' + X.cov('failed') + '<span>دریافت آمار این تیم ناموفق بود؛ مقدار صفر فرض نشده است.</span></div></td><td>' + X.cov('failed') + '</td><td class="col-actions"><button type="button" class="btn btn-soft btn-sm" data-act="retry-team">' + ic('refresh') + 'تلاش مجدد</button></td></tr>';
      }
      var covLbl = cov === 'recon' ? 'نیازمند تطبیق' : null;
      return '<tr data-row="team:' + t.id + '" tabindex="-1"' + (t.active ? '' : ' class="inactive"') + '><td>' + X.who(t.sup, X.rel('sup') + (t.active ? '' : ' ' + pill('غیرفعال', 'slate', 'ban'))) + (t.active ? '' : '<div class="cell-sub">' + esc(t.inactiveNote) + '</div>') + '</td>' +
        '<td class="n">' + fa(act) + '<span class="of"> از ' + fa(ss.length) + '</span></td>' +
        '<td class="wl"><span class="wl-n">' + fa(w) + '</span><span class="wl-bar" aria-hidden="true"><i style="width:' + Math.round(w / maxW * 100) + '%"></i></span></td>' +
        '<td class="n col-opt">' + (cov === 'stale' ? '<span class="stale-n tip" data-tip="داده این تیم قدیمی است">' + fa(f.openPre) + '</span>' : fa(f.openPre)) + '</td>' +
        '<td class="n">' + n0(rd) + '</td>' +
        '<td>' + (ex ? '<div class="exc-cell"><span class="pill t-' + ex.tone + '">' + esc(ex.t) + '</span><span class="cell-sub">مسئول: ' + esc(ex.a) + '</span></div>' : '<span class="muted">—</span>') + '</td>' +
        '<td>' + X.cov(cov, covLbl) + '</td>' +
        '<td class="col-actions"><button type="button" class="btn btn-soft btn-sm" data-act="open-team:' + t.id + '">مشاهده تیم</button></td></tr>';
    }).join('');
    var dseller = M.sellers.filter(function (s) { return s.rel === 'direct'; }).map(function (s) {
      return '<tr data-row="seller:' + s.id + '" tabindex="-1" class="direct-row"><td>' + X.who(s.name, X.rel('dseller') + ' <span class="cell-sub">بدون سرپرست میانی</span>') + '</td><td class="n">' + fa(1) + '<span class="of"> از ' + fa(1) + '</span></td><td class="wl"><span class="wl-n">' + fa(s.open) + '</span></td><td class="n col-opt">' + fa(X.seller_f(s.id).openPre) + '</td><td class="n">' + n0(M.ready.filter(function (r) { return r.seller === s.id; }).length) + '</td><td><span class="muted">—</span></td><td>' + X.cov('ok') + '</td><td class="col-actions"><button type="button" class="btn btn-soft btn-sm" data-act="open-seller:' + s.id + '">مشاهده</button></td></tr>';
    }).join('');
    var th = '<thead><tr><th>تیم و سرپرست مسئول</th><th class="n">فروشنده فعال ' + hint('فروشندگان زیر این سرپرست؛ غیرمستقیم برای شما و فقط برای مشاهده.', true) + '</th><th>پرونده نزد تیم ' + hint('مالک فعلی پرونده‌ها؛ این ستون رتبه یا ارزیابی عملکرد نیست.', true) + '</th><th class="n col-opt">پیش‌فاکتور باز</th><th class="n">آماده تبدیل ' + hint('فقط شمار؛ مدت انتظار تعریف نشده است.', true) + '</th><th>مهم‌ترین استثنا و مسئول آن</th><th>اعتماد به داده</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead>';
    var grpRow = dseller ? '<tr class="grp"><th colspan="8" scope="colgroup"><span>' + ic('user') + 'مستقیم زیر نظر شما (در صورت وجود در ساختار)</span></th></tr>' : '';
    var cap = '<section class="panel main cap-panel" aria-labelledby="cap-h"><div class="sec-h ph"><h2 id="cap-h">' + ic('users') + 'تیم‌های تحت پوشش شما</h2>' + X.basis('current') + X.grain('تیم (سرپرست مستقیم)') + '<span class="aside">به ترتیب نام · رتبه‌بندی نیست</span></div>' +
      '<div class="tbl-wrap"><table class="tbl" aria-label="تیم‌های تحت پوشش"><caption class="sr">هر ردیف یک تیم با سرپرست مسئول آن؛ ستون آخر وضعیت تازگی و پوشش داده است.</caption>' + th + '<tbody>' + rows + grpRow + dseller + '</tbody></table></div></section>';
    var pool = '<section class="panel pool-strip"><div class="sec-h ph"><h2>' + ic('inbox') + 'موجودی پنل من</h2><span class="scope-badge">' + ic('user') + 'شخصی · جدا از بار تیم‌ها</span></div><div class="pool-body"><div><b class="pool-n">' + fa(M.pool.length) + '</b> پرونده <span class="muted">· ' + fa(X.poolOk().length) + ' قابل تخصیص · ' + fa(M.pool.length - X.poolOk().length) + ' نیازمند بررسی</span></div><button type="button" class="btn btn-soft" data-act="goto:dist">' + ic('send') + 'تخصیص به سرپرستان</button></div></section>';
    var struct = '<section class="panel ind-panel"><div class="sec-h ph"><h2>' + ic('eye') + 'ساختار دیده‌شده برای شما</h2></div><p class="ind-note">' + X.rel('sup') + ' سرپرست‌ها: گیرنده تخصیص. ' + X.rel('dseller') + ' فقط اگر ساختار واقعی داشته باشد. ' + X.rel('indirect') + ' فروشندگان زیر سرپرستان: فقط مشاهده و گزارش؛ دیدن یک نفر به معنی اجازه نوشتن برای او نیست.</p></section>';
    return h.pageHead({ title: 'نمای چندتیمی', sub: 'محدوده: ' + fa(M.teams.length) + ' سرپرست مستقیم · ' + fa(M.sellers.length) + ' فروشنده در محدوده شما · ارزیابی ' + esc(M.freshness.evaluated), kpis: kpis, fresh: X.freshPart('آمار تیم‌ها'), scope: 'محدوده: زیرمجموعه مجاز شما' }) + X.banners() +
      '<div class="team-grid">' + attn + '<div class="team-main">' + cap + pool + struct + '</div></div>';
  };

  /* ================= Structure & Performance Explorer ================= */
  var MODES = [['team', 'تیم', 'users'], ['sup', 'سرپرست', 'briefcase'], ['seller', 'فروشنده', 'user'], ['conv', 'تبدیل', 'repeat'], ['inv', 'فاکتور', 'receipt'], ['case', 'بار پرونده', 'inbox']];
  function inScopeTeam(tid) { var p = st.pscope; if (!p) return true; return (String(p).charAt(0) === 'T' ? p : seller(Number(p)).team) === tid; }
  function inScopeSeller(s) { var p = st.pscope; if (!p) return true; return String(p).charAt(0) === 'T' ? s.team === p : s.id === Number(p); }

  V.perf = function () {
    var pm = st.pm, basisBtn = pm === 'team' || pm === 'inv';
    var dim = '<div class="dim-switch" role="group" aria-label="بعد تحلیل (واحد شمارش)">' + MODES.map(function (m) { return '<button type="button" data-pm="' + m[0] + '" aria-pressed="' + (pm === m[0]) + '">' + ic(m[2]) + m[1] + '</button>'; }).join('') + '</div>';
    var basis = basisBtn ? '<div class="seg" role="group" aria-label="مبنای انتساب"><button type="button" data-pb="current" aria-pressed="' + (st.pb === 'current') + '">ساختار فعلی</button><button type="button" data-pb="hist" aria-pressed="' + (st.pb === 'hist') + '">انتساب تاریخی</button></div>' : '<span class="basis-fixed">' + (pm === 'seller' ? X.basis('event') + '<span class="muted">آمار فروشنده به خود او نسبت داده می‌شود و به والد امروز وابسته نیست.</span>' : X.basis('current')) + '</span>';
    var body = ({ team: perfTeam, sup: perfSup, seller: perfSeller, conv: perfConv, inv: perfInv, 'case': perfCase })[pm]();
    var ctl = '<section class="panel explorer-ctl" aria-label="کنترل‌های کاوشگر">' + dim + basis + '</section>';
    return h.pageHead({ title: 'ساختار و عملکرد', sub: 'هر حالت یک واحد شمارش دارد و با حالت دیگر مخلوط نمی‌شود · بدون رتبه‌بندی و بدون هدف فروش', fresh: X.freshPart('داده'), scope: 'مبنا: ' + (basisBtn ? (st.pb === 'hist' ? 'انتساب تاریخی' : 'ساختار فعلی') : 'ثابت برای این حالت') }) + X.banners() + ctl +
      '<section class="panel main">' + '<div class="ex-bar">' + X.crumb() + '<span class="grow"></span>' + X.grain(body.grain) + (basisBtn ? X.basis(st.pb === 'hist' ? 'hist' : 'current') : '') + '</div>' + (body.note ? '<div class="note info inset">' + ic('info') + '<span>' + body.note + '</span></div>' : '') + body.html + '</section>';
  };

  function covOrVal(tid, val) { var c = X.teamCov(tid); return c === 'failed' ? na('دریافت نشد') : val; }

  function perfTeam() {
    var hist = st.pb === 'hist', ids = M.teams.map(function (t) { return t.id; }).concat(['direct']).filter(inScopeTeam);
    var rows = ids.map(function (tid) {
      var t = tid === 'direct' ? null : team(tid), f = X.team_f(tid, hist ? 'hist' : 'current'), cov = X.teamCov(tid);
      var nm = tid === 'direct' ? X.who('مستقیم زیر نظر شما', X.rel('dseller')) : X.who(t.sup, X.rel('sup') + (t.active ? '' : ' ' + pill('غیرفعال', 'slate', 'ban')));
      var ss = X.sellersOf(tid), cur = hist ? na('فقط در ساختار فعلی معنا دارد') : null;
      return '<tr data-row="team:' + tid + '" tabindex="-1"><td>' + nm + '</td>' +
        '<td class="n">' + (cur || fa(ss.filter(function (s) { return s.active; }).length) + '<span class="of"> از ' + fa(ss.length) + '</span>') + '</td>' +
        '<td class="n">' + (cur || covOrVal(tid, fa(X.teamWorkload(tid)))) + '</td>' +
        '<td class="n">' + covOrVal(tid, fa(f.created)) + '</td><td class="n">' + covOrVal(tid, fa(f.openPre)) + '</td><td class="n">' + covOrVal(tid, fa(f.completed)) + '</td>' +
        '<td class="n">' + covOrVal(tid, cov === 'recon' ? (f.collected ? '<span class="recon-n tip" tabindex="0" data-tip="مغایرت بین منبع پرداخت و وضعیت فاکتور؛ مبلغ نهایی نیست">' + money(f.collected) + '</span>' : na('نیازمند تطبیق: مغایرت بین منبع پرداخت و وضعیت فاکتور؛ مبلغ صفر فرض نمی‌شود')) : money(f.collected)) + '</td>' +
        '<td>' + X.cov(cov) + '</td><td class="col-actions"><button type="button" class="btn btn-sm" data-act="drill:' + tid + '">فروشندگان</button></td></tr>';
    }).join('');
    var unk = '';
    if (hist && !st.pscope) { var uf = X.facts(function (i) { return i.teamAtIssue == null; }); unk = '<tr class="unk-row"><td>' + X.who('تیم ناشناخته', '<span class="scope-badge view">' + ic('question') + 'غیرقابل بازیابی</span>') + '</td><td class="n">' + na('') + '</td><td class="n">' + na('') + '</td><td class="n">' + fa(uf.created) + '</td><td class="n">' + fa(uf.openPre) + '</td><td class="n">' + fa(uf.completed) + '</td><td class="n">' + money(uf.collected) + '</td><td>' + X.cov('undef', 'ناشناخته') + '</td><td class="col-actions"></td></tr>'; }
    var all = X.facts(function () { return true; });
    var tot = st.pscope ? '' : '<tfoot><tr class="tot"><th scope="row">جمع یکتا ' + hint('هر فاکتور فقط یک‌بار شمرده می‌شود؛ جمع از زیرجمع‌های هم‌پوشان ساخته نشده است.', true) + '</th><td></td><td></td><td class="n">' + fa(all.created) + '</td><td class="n">' + fa(all.openPre) + '</td><td class="n">' + fa(all.completed) + '</td><td class="n">' + money(all.collected) + '</td><td colspan="2"></td></tr></tfoot>';
    return { grain: 'فاکتور و پرونده به تفکیک تیم', note: hist ? '<b>انتساب تاریخی:</b> هر فاکتور به تیمی که هنگام صدور ثبت شده نسبت داده می‌شود؛ نقل‌مکان فروشنده، سابقه گذشته را جابه‌جا نمی‌کند و سابقه‌نداشته «ناشناخته» می‌ماند. بار کاری فقط در ساختار فعلی معنا دارد.' : '<b>ساختار فعلی:</b> بار کاری امروز. فروشنده‌ای که اخیراً جابه‌جا شده اینجا زیر تیم فعلی‌اش دیده می‌شود؛ اعتبار فروش‌های گذشته را با «انتساب تاریخی» ببینید.',
      html: '<div class="tbl-wrap"><table class="tbl" aria-label="عملکرد به تفکیک تیم"><thead><tr><th>تیم</th><th class="n">فروشنده فعال</th><th class="n">پرونده نزد تیم</th><th class="n">فاکتور صادرشده</th><th class="n">پیش‌فاکتور باز</th><th class="n">فروش تکمیل‌شده</th><th class="n">وصول تأییدشده ' + hint('جمع مبلغ مراحل معتبر؛ مغایرت‌دار شمرده نمی‌شود و «نیازمند تطبیق» علامت می‌خورد.', true) + '</th><th>اعتماد به داده</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + unk + '</tbody>' + tot + '</table></div>' };
  }

  function perfSup() {
    var rows = M.teams.filter(function (t) { return inScopeTeam(t.id); }).map(function (t) {
      var ss = X.sellersOf(t.id), rdy = M.ready.filter(function (r) { return r.team === t.id && r.status === 'ready'; }).length;
      var need = M.invoices.filter(function (i) { return X.seller(i.seller).team === t.id && i.next && (i.next.who === 'seller' || i.next.who === 'mis'); }).length;
      var hrIn = M.hr.filter(function (r) { return r.requester === t.sup && r.state !== 'approved' && r.state !== 'rejected'; }).length;
      return '<tr data-row="team:' + t.id + '" tabindex="-1"' + (t.active ? '' : ' class="inactive"') + '><td>' + X.who(t.sup, X.rel('sup')) + '</td><td>' + (t.active ? pill('فعال', 'teal', 'checkCircle') : pill('غیرفعال', 'slate', 'ban') + '<div class="cell-sub">' + esc(t.inactiveNote) + '</div>') + '</td>' +
        '<td class="n">' + fa(ss.length) + '</td><td class="n">' + n0(rdy) + '</td><td class="n">' + n0(need) + '</td><td class="n">' + n0(hrIn) + '</td>' +
        '<td>' + (t.active ? '<span class="muted">سرپرست خود تیم</span>' : '<span class="pill t-amber">' + ic('question') + 'ثبت نشده</span>') + '</td><td class="col-actions"><button type="button" class="btn btn-sm" data-act="open-team:' + t.id + '">جزئیات</button></td></tr>';
    }).join('');
    return { grain: 'مسئولیت سرپرست', note: 'این حالت <b>پاسخ‌گویی</b> هر سرپرست را نشان می‌دهد، نه عملکرد فروش. بدون امتیاز و رتبه؛ مسئول جایگزین فقط اگر ثبت شده باشد نمایش داده می‌شود.',
      html: '<div class="tbl-wrap"><table class="tbl" aria-label="مسئولیت سرپرستان"><thead><tr><th>سرپرست</th><th>وضعیت</th><th class="n">فروشنده زیردست</th><th class="n">آماده تبدیل در انتظار</th><th class="n">فاکتور با اقدام فروشنده/MIS</th><th class="n">درخواست HR در مسیر</th><th>مسئول جایگزین</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div>' };
  }

  function perfSeller() {
    var list = M.sellers.filter(inScopeSeller);
    var rows = list.map(function (s) {
      var f = X.seller_f(s.id), tcov = X.teamCov(s.team);
      var failed = tcov === 'failed';
      var teamCell = s.rel === 'direct' ? X.rel('dseller') : X.rel('indirect') + '<div class="cell-sub">' + esc(X.teamName(s.team)) + '</div>' + (s.moved ? '<div class="cell-sub moved">' + ic('swap') + 'منتقل‌شده از ' + esc(X.teamName(s.moved.from)) + ' · ' + esc(s.moved.when) + '</div>' : '');
      var legacyCell = tcov === 'partial' || (sim() === 'incomplete' && s.team === 'T2') ? '<span class="partial-n tip" tabindex="0" data-tip="پوشش ناقص: عدد نهایی نیست">' + fa(s.legacy) + '</span>' : fa(s.legacy);
      return '<tr data-row="seller:' + s.id + '" tabindex="-1"' + (s.active ? '' : ' class="inactive"') + '><td>' + X.who(s.name, s.active ? '' : '<span class="warn-n">غیرفعال</span>') + '</td><td>' + teamCell + '</td>' +
        '<td class="n">' + (failed ? na('دریافت نشد') : fa(s.open)) + '</td><td class="n">' + (failed ? na('دریافت نشد') : legacyCell) + '</td><td class="n">' + (failed ? na('دریافت نشد') : fa(f.created)) + '</td><td class="n">' + (failed ? na('دریافت نشد') : n0(f.openPre)) + '</td><td class="n">' + (failed ? na('دریافت نشد') : n0(f.completed)) + '</td><td class="n">' + (failed ? na('دریافت نشد') : f.created ? money(f.collected) : na('فاکتوری ثبت نشده')) + '</td><td class="col-opt muted">' + (f.last || '—') + '</td><td class="col-actions"><button type="button" class="btn btn-sm" data-act="open-seller:' + s.id + '">جزئیات</button></td></tr>';
    }).join('');
    return { grain: 'فروشنده', note: 'ستون «فقط لیدهای قدیمی» عدد <b>کل پرونده‌ها نیست</b>؛ پوشش منابع جدید هنوز تأیید نشده است. «۰» فقط یعنی محاسبه کامل شده و مقدار صفر است.',
      html: (list.length ? '<div class="tbl-wrap"><table class="tbl" aria-label="عملکرد به تفکیک فروشنده"><thead><tr><th>فروشنده</th><th>رابطه با شما و تیم فعلی</th><th class="n">پرونده باز</th><th class="n">فقط لیدهای قدیمی ' + hint('شمارش فقط جدول قدیمی؛ برچسب «کل لیدها» نیست.', true) + '</th><th class="n">فاکتور صادرشده</th><th class="n">پیش‌فاکتور باز</th><th class="n">فروش تکمیل‌شده</th><th class="n">وصول تأییدشده</th><th class="col-opt">آخرین فاکتور</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div>' : h.stateBlock('noresult', 'فروشنده‌ای در این محدوده نیست', 'محدوده را به «همه تیم‌ها» برگردانید.', '<button type="button" class="btn btn-soft" data-act="pscope:all">همه تیم‌ها</button>')) };
  }

  function perfConv() {
    var rows = M.conv.filter(function (r) { return inScopeTeam(r.team); }).map(function (r) {
      var t = team(r.team);
      return '<tr data-row="team:' + r.team + '" tabindex="-1"><td>' + X.who(t.sup, X.rel('sup')) + '</td><td>' + esc(r.converter) + '</td><td class="n">' + n0(r.ready) + '</td><td class="n">' + n0(r.assigned) + '</td><td class="n">' + n0(r.completed) + '</td><td>' + (r.assigned === 0 ? X.cov('insuff', 'داده ناکافی · گروه خالی') : X.cov('undef')) + '</td></tr>';
    }).join('');
    return { grain: 'پرونده تبدیل (Dot)', note: 'تماس، رسید، تأیید مرحله و تکمیل <b>جدا</b> شمرده می‌شوند. نرخ تبدیل هنوز تصویب نشده (OPD-07)؛ گروه خالی به معنی عملکرد ضعیف نیست.',
      html: '<div class="tbl-wrap"><table class="tbl" aria-label="تبدیل به تفکیک تیم"><thead><tr><th>سرپرست مسئول</th><th>تبدیل‌کننده</th><th class="n">آماده تبدیل</th><th class="n">تخصیص‌یافته</th><th class="n">تکمیل‌شده</th><th>نرخ تبدیل</th></tr></thead><tbody>' + rows + '</tbody></table></div>' };
  }

  function perfInv() {
    var hist = st.pb === 'hist';
    var list = M.invoices.filter(function (i) { return inScopeTeam(X.teamIdOfInv(i, 'current')) || (hist && st.pscope && X.teamIdOfInv(i, 'hist') === (String(st.pscope).charAt(0) === 'T' ? st.pscope : null)); }).filter(function (i) { var p = st.pscope; return !p || String(p).charAt(0) === 'T' || i.seller === Number(p); });
    var rows = list.map(function (i) {
      var s = seller(i.seller), tn = X.teamIdOfInv(i, 'current'), th = i.teamAtIssue;
      var thCell = th == null ? '<span class="pill t-slate">' + ic('question') + 'ناشناخته</span><div class="cell-sub">' + esc(i.teamNote || '') + '</div>' : esc(X.teamName(th));
      var differs = th != null && th !== tn;
      return '<tr data-row="inv:' + i.code + '" tabindex="-1"><td><span class="mono case-id">' + fa(i.code) + '</span><div class="cell-sub">' + X.link(i.link) + '</div></td><td>' + X.who(s.name, '<span class="muted">عامل رویداد صدور و مالک اعتبار</span>') + '</td>' +
        '<td class="' + (hist ? 'dimcol' : 'hl') + '">' + esc(X.teamName(tn)) + '</td><td class="' + (hist ? 'hl' : 'dimcol') + '">' + thCell + (differs ? '<div class="cell-sub moved">' + ic('swap') + 'با تیم امروز فرق دارد</div>' : '') + '</td>' +
        '<td>' + pill(X.INVS[i.inv].label, X.INVS[i.inv].tone, X.INVS[i.inv].icon) + '</td><td class="col-opt muted">' + esc(i.issued) + '</td><td class="col-actions"><button type="button" class="btn btn-sm" data-act="open-inv:' + i.code + '">جزئیات</button></td></tr>';
    }).join('');
    return { grain: 'فاکتور (رویداد صدور)', note: 'سه مفهوم جدا: <b>فروشنده</b> = عامل رویداد/مالک اعتبار، <b>تیم فعلی</b> = ساختار امروز، <b>تیم هنگام صدور</b> = انتساب تاریخی. تیم امروز جای تیم تاریخی نمی‌نشیند.',
      html: '<div class="tbl-wrap"><table class="tbl" aria-label="فاکتورها به تفکیک تیم"><thead><tr><th>فاکتور و اطمینان اتصال</th><th>فروشنده (' + 'عامل رویداد)</th><th>تیم فعلی <span class="sr">(ساختار فعلی)</span></th><th>تیم هنگام صدور <span class="sr">(انتساب تاریخی)</span></th><th>وضعیت فاکتور</th><th class="col-opt">صدور</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div>' };
  }

  function perfCase() {
    var rows = M.teams.concat([{ id: 'direct', sup: null }]).filter(function (t) { return inScopeTeam(t.id); }).map(function (t) {
      var atSup = M.held.filter(function (c) { return c.holder.kind === 'sup' && c.holder.team === t.id; }).length, atSel = X.teamWorkload(t.id);
      var cov = X.teamCov(t.id);
      return '<tr data-row="team:' + t.id + '" tabindex="-1"><td>' + X.who(t.sup || 'مستقیم زیر نظر شما', t.sup ? X.rel('sup') : X.rel('dseller')) + '</td><td class="n">' + (cov === 'failed' ? na('دریافت نشد') : fa(atSel)) + '</td><td class="n">' + (t.sup ? n0(atSup) : '<span class="muted">—</span>') + '</td><td>' + X.cov(cov) + '</td></tr>';
    }).join('');
    var own = '<tr class="own-row"><td>' + X.who('پنل من (کامران صدری)', '<span class="scope-badge">' + ic('user') + 'شخصی · موجودی قابل توزیع</span>') + '</td><td class="n"><span class="muted">—</span></td><td class="n">' + fa(M.pool.length) + '</td><td>' + X.cov('ok') + '</td></tr>';
    return { grain: 'پرونده یکتا نزد نگهدارنده فعلی', note: 'هر پرونده فقط یک‌بار و نزد <b>مسئول فعلی</b> شمرده می‌شود (نه مالک اولیه). پرونده با چند انتقال یک واقعیت است؛ فاکتورِ متصل به چند پرونده هم یک فاکتور است.',
      html: '<div class="tbl-wrap"><table class="tbl" aria-label="بار پرونده"><thead><tr><th>نگهدارنده فعلی</th><th class="n">نزد فروشندگان</th><th class="n">نزد خود سرپرست / پنل</th><th>اعتماد به داده</th></tr></thead><tbody>' + rows + own + '</tbody></table></div>' };
  }

  /* ================= Ready Conversion — Team View (readonly) ================= */
  V.ready = function () {
    var list = M.ready, g = st.rg;
    var wait = list.filter(function (r) { return r.status === 'ready'; }).length;
    var kpis = [
      { label: 'در صف تیم‌ها', value: fa(wait), color: 'teal', keep: true, meaning: 'پرونده‌های آماده تبدیل که هنوز تبدیل‌کننده ندارند؛ تعریف مدت انتظار تصویب نشده', basis: 'وضعیت در لحظه' },
      { label: 'تخصیص‌یافته به تبدیل‌کننده', value: fa(list.length - wait), color: 'blue', meaning: 'تبدیل‌کننده دارند', basis: 'وضعیت در لحظه' },
      { label: 'مدت انتظار / تأخیر', value: '', state: 'undefined', color: 'neutral', meaning: 'زمان شروع انتظار معتبر و سیاست سررسید تعریف نشده است', basis: 'تعریف نشده (SD-05)' },
      { label: 'نرخ تبدیل', value: '', state: 'undefined', color: 'neutral', meaning: 'مخرج و بازه تصویب نشده (OPD-07)', basis: 'تعریف نشده' }
    ];
    var segm = '<div class="seg" role="group" aria-label="نحوه نمایش"><button type="button" data-rg="team" aria-pressed="' + (g === 'team') + '">گروه‌بندی بر اساس تیم</button><button type="button" data-rg="flat" aria-pressed="' + (g === 'flat') + '">فهرست ساده</button></div>';
    var rowOf = function (r) {
      var s = seller(r.seller), t = r.team === 'direct' ? null : team(r.team), st_ = X.CONVS[r.status];
      var resp = r.direct ? X.who('کامران صدری (شما)', '<span class="muted">فروشنده مستقیم شماست</span>') : t.active ? X.who(t.sup, X.rel('sup')) : '<span class="pill t-amber">' + ic('question') + 'مسئول نامشخص</span><div class="cell-sub">' + esc(r.unresolved) + '</div>';
      return '<tr data-row="rcase:' + r.id + '" tabindex="-1"><td><span class="mono case-id">' + r.id + '</span><div class="cell-sub">' + X.src(r.source) + '</div></td><td>' + esc(r.customer) + '</td><td>' + X.who(s.name) + '</td><td>' + resp + '</td><td>' + X.nextActor({ who: r.unresolved ? 'hr' : r.status === 'assigned' ? 'seller' : 'sup', text: r.next }, null) + '</td><td>' + pill(st_.label, st_.tone, st_.icon) + (r.choice ? '' : '<div class="cell-sub">انتخاب مشتری: ثبت نشده</div>') + '</td><td class="col-opt">' + esc(r.stage) + '</td><td class="col-opt">' + X.cov('undef', 'تعریف نشده') + '</td><td class="col-actions"><button type="button" class="btn btn-sm" data-act="open-rcase:' + r.id + '">جزئیات</button></td></tr>';
    };
    var rows = '';
    if (g === 'team') {
      M.teams.concat([{ id: 'direct' }]).forEach(function (t) {
        var items = list.filter(function (r) { return r.team === t.id; }); if (!items.length) return;
        rows += '<tr class="grp"><th colspan="9" scope="colgroup"><span>' + ic('users') + (t.id === 'direct' ? 'مستقیم زیر نظر شما' : 'تیم ' + esc(t.sup) + (t.active ? '' : ' · سرپرست غیرفعال')) + '</span><span class="muted">' + fa(items.length) + ' پرونده</span></th></tr>' + items.map(rowOf).join('');
      });
    } else rows = list.map(rowOf).join('');
    var tbl = '<div class="tbl-wrap"><table class="tbl" aria-label="آماده‌های تبدیل تیم‌ها"><thead><tr><th>پرونده</th><th>مشتری</th><th>فروشنده</th><th>سرپرست مسئول</th><th>اقدام بعدی با</th><th>وضعیت تبدیل</th><th class="col-opt">زمینه مالی مجاز</th><th class="col-opt">مدت انتظار</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div>';
    return h.pageHead({ title: 'آماده‌های تبدیل تیم‌ها', sub: 'نمای نظارتی · مسئول تخصیص تبدیل، سرپرست مستقیم هر فروشنده است', kpis: kpis, fresh: X.freshPart('صف') }) + X.banners() +
      '<section class="panel main"><div class="note info inset">' + ic('eye') + '<span><b>فقط‌خواندنی.</b> از این صفحه تبدیل‌کننده تعیین یا ارجاع داده نمی‌شود. فقط مشخص است <b>مسئول اقدام بعدی کیست</b>؛ مسیر رسمی ارجاع هنوز تعریف نشده (SG-02).</span></div><div class="toolbar">' + segm + '<span class="grow"></span><span class="muted desk">' + fa(list.length) + ' پرونده · ' + X.cov('ok', 'نمونه نمایشی') + '</span></div>' + tbl + h.tfoot(list.length, list.length) + '</section>';
  };

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
      var s = seller(i.seller), rv = X.REV[i.review], t = s.team;
      return '<tr data-row="inv:' + i.code + '" tabindex="-1"><td><span class="mono case-id">' + fa(i.code) + '</span><div class="cell-sub">' + esc(i.customer) + '</div><div class="cell-sub">' + X.link(i.link) + '</div></td>' +
        '<td>' + X.who(s.name, esc(X.teamName(t))) + '</td>' +
        '<td>' + pill(X.INVS[i.inv].label, X.INVS[i.inv].tone, X.INVS[i.inv].icon) + '</td>' +
        '<td><b>مرحله ' + fa(i.stage) + ' از ' + fa(i.stages) + '</b><div class="cell-sub">' + esc(X.STG[i.stg]) + ' · ' + esc(X.EVID[i.evidence]) + '</div></td>' +
        '<td>' + (rv ? pill(rv.label, rv.tone, rv.icon) : '<span class="muted">ارسال نشده</span>') + '</td>' +
        '<td class="amt-col">' + money(i.paid) + '<div class="cell-sub">مانده: ' + money(i.inv === 'cancelled' ? 0 : i.total - i.paid) + '</div></td>' +
        '<td>' + X.nextActor(i.next, i) + (i.reason ? '<div class="why-line">' + ic('alert') + esc(i.reason) + '</div>' : '') + (i.mismatch ? '<div class="why-line warn">' + ic('question') + esc(i.mismatch) + '</div>' : '') + '</td>' +
        '<td class="col-actions"><button type="button" class="btn btn-sm" data-act="open-inv:' + i.code + '">جزئیات</button></td></tr>';
    }).join('');
    var kpis = [
      { label: 'فاکتورهای محدوده', value: fa(M.invoices.length), color: 'neutral', keep: true, meaning: 'فاکتور یکتا در تیم‌های شما', basis: 'وضعیت در لحظه' },
      { label: 'نیازمند هماهنگی', value: fa(M.invoices.filter(IQ[0].f).length), color: 'red', tone: 'bad', filter: 'iq:action', meaning: 'رد مالی یا مغایرت؛ اقدام بعدی با فروشنده یا MIS', basis: 'وضعیت در لحظه' },
      { label: 'در بررسی مالی', value: fa(M.invoices.filter(IQ[1].f).length), color: 'orange', filter: 'iq:finance', meaning: 'منتظر واحد مالی؛ اقدام شما لازم نیست', basis: 'وضعیت در لحظه' },
      { label: 'فروش تکمیل‌شده', value: fa(X.facts(function () { return true; }).completed), color: 'green', meaning: 'پرداخت کامل و تأیید مالی؛ صدور یا رسید به‌تنهایی کافی نیست', basis: 'وضعیت در لحظه · مرز تکمیل: OPD-06' }
    ];
    return h.pageHead({ title: 'فاکتورها و استثناها', sub: 'وضعیت فاکتور، مرحله پرداخت و بررسی مالی سه چیز جدا هستند · تأیید/رد/بازپرداخت مالی از این پنل ممکن نیست', kpis: kpis, fresh: X.freshPart('فاکتورها') }) + X.banners() +
      '<section class="panel main">' + tabs + h.toolbar('کد فاکتور، مشتری یا فروشنده', [{ label: 'تیم', icon: 'users' }, { label: 'بازه صدور', icon: 'clock' }]) +
      (list.length ? '<div class="tbl-wrap"><table class="tbl" aria-label="فاکتورهای ' + esc(q.label) + '"><thead><tr><th>فاکتور و اتصال</th><th>فروشنده و تیم فعلی</th><th>وضعیت فاکتور</th><th>مرحله پرداخت</th><th>بررسی مالی</th><th>پرداخت‌شده / مانده</th><th>اقدام بعدی و دلیل</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div>' + h.tfoot(list.length, list.length)
        : h.stateBlock('empty', 'فاکتوری در این صف نیست', 'صف دیگری را انتخاب کنید.', '<button type="button" class="btn btn-soft" data-iq="all">همه فاکتورها</button>')) + '</section>';
  };

  /* ================= Case Explorer & Source History (baseline: legacy facet) ================= */
  V.cases = function () {
    var segm = '<div class="seg" role="group" aria-label="دامنه منبع"><button type="button" data-src="legacy" aria-pressed="' + (st.src === 'legacy') + '">لیدهای قدیمی (پایه)</button><button type="button" data-src="all" aria-pressed="' + (st.src === 'all') + '">همه منابع تأییدشده</button></div>';
    var list = M.cases.filter(function (c) { return st.src === 'all' ? true : c.src === 'legacy'; });
    var rows = list.map(function (c) {
      var s = c.seller ? seller(c.seller) : null;
      return '<tr data-row="case:' + c.id + '" tabindex="-1"><td><span class="mono case-id">' + c.id + '</span><div class="cell-sub">' + X.src(c.src) + '</div></td><td><span class="mono phone-num tip" tabindex="0" data-tip="شماره فقط برای جستجو است و هویت پرونده نیست">' + fa(c.phone) + '</span></td><td>' + (s ? X.who(s.name, esc(X.teamName(s.team))) : esc(X.teamName(c.team))) + '</td>' +
        '<td><b>' + esc(c.raw) + '</b><div class="cell-sub">برچسب نگاشت‌شده: ' + esc(c.mapped) + '</div></td><td>' + X.link(c.link) + '</td><td class="col-opt">' + fa(c.hist.length) + ' رویداد</td><td class="col-actions"><button type="button" class="btn btn-sm" data-act="open-case:' + c.id + '">سوابق منبع</button></td></tr>';
    }).join('');
    return h.pageHead({ title: 'پرونده و سوابق منبع', sub: 'نمای زمینه‌ای فقط‌خواندنی · شماره موبایل ابزار جستجوست، نه هویت پرونده', fresh: X.freshPart('پرونده‌ها') }) + X.banners() +
      '<section class="panel main"><div class="note warn inset">' + ic('layers') + '<span><b>پوشش ناقص.</b> نمای یکپارچه پرونده هنوز تأیید نشده است؛ این صفحه فقط ' + (st.src === 'legacy' ? '<b>لیدهای قدیمی</b>' : 'منابع تأییدشده') + ' و <b>آخرین ردیف‌های محدود</b> را نشان می‌دهد و تعداد کل ادعا نمی‌کند.</span></div><div class="toolbar">' + segm + '<label class="field"><span class="sr">جستجو</span>' + ic('filter') + '<input class="input" type="search" id="queue-filter" placeholder="شناسه پرونده، فروشنده یا شماره" aria-keyshortcuts="F"></label><span class="grow"></span><span class="muted desk">آخرین ' + fa(list.length) + ' ردیف</span></div>' +
      '<div class="tbl-wrap"><table class="tbl" aria-label="پرونده‌ها و منبع"><thead><tr><th>پرونده و منبع</th><th>شماره (فقط جستجو)</th><th>فروشنده / تیم</th><th>وضعیت خام و نگاشت</th><th>اطمینان اتصال</th><th class="col-opt">سوابق</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div></section>';
  };

  /* ================= Personal (conditional) + Wallet ================= */
  V.mine = function () {
    var rows = M.myConversions.map(function (r) { return '<tr data-row="mycase:' + r.id + '" tabindex="-1"><td><span class="mono case-id">' + r.id + '</span></td><td>' + esc(r.customer) + '</td><td>' + esc(r.option) + '</td><td>' + money(r.remaining) + '</td><td>' + pill(r.stage, 'amber', 'phone') + '</td></tr>'; }).join('');
    return h.pageHead({ title: 'تبدیل‌های من', sub: 'ماژول شخصی مشروط · فقط پرونده‌هایی که خودتان طبق مجوز دریافت کرده‌اید' }) +
      '<section class="panel main"><div class="note info inset">' + ic('user') + '<span><b>مشروط و شخصی.</b> این صفحه فقط وقتی دیده می‌شود که مجوز دریافت تبدیل شخصی فعال باشد یا پرونده تخصیص‌یافته به خودتان داشته باشید (SD-02). با داده تیم‌ها قاطی نمی‌شود و مالک اعتبار فروش را عوض نمی‌کند.</span></div>' +
      (M.myConversions.length ? '<div class="tbl-wrap"><table class="tbl" aria-label="تبدیل‌های من"><thead><tr><th>پرونده</th><th>مشتری</th><th>گزینه</th><th>مانده</th><th>مرحله</th></tr></thead><tbody>' + rows + '</tbody></table></div>' : h.stateBlock('empty', 'تبدیل شخصی ندارید', 'خالی بودن به معنی حذف قابلیت نیست.', '')) + '</section>';
  };
  V.wallet = function () {
    var W = M.wallet, rows = W.tx.map(function (t) { return '<tr><td class="muted"><span class="num">' + esc(t.date) + '</span></td><td>' + pill('بستانکار', 'green', 'plus') + '</td><td><span class="tag' + (t.channel === 'آنلاین' ? ' teal' : '') + '">' + esc(t.channel) + '</span></td><td>' + esc(t.desc) + '<div class="muted" style="font-size:var(--t-micro)">' + esc(t.rule) + '</div></td><td style="text-align:left"><span class="credit">+<span class="num">' + num(t.amount) + '</span></span> <small class="muted">تومان</small></td></tr>'; }).join('');
    return h.pageHead({ title: 'کیف پول من', sub: 'شخصی و فقط‌خواندنی — فقط حق‌الزحمه خودتان؛ درآمد یا پورسانت تیم و دیگران اینجا نیست' }) +
      '<div class="wallet-sum"><div class="wcard hero"><span>موجودی ثبت‌شده</span><b><span class="num">' + num(W.balance) + '</span><small>تومان</small></b><div class="foot"><span>بدون تسویه‌نشده‌ی محاسبه‌شده</span></div></div><div class="wcard"><span>کل بستانکاری</span><b><span class="num">' + num(W.credit) + '</span><small>تومان</small></b><div class="foot"><span>' + fa(W.tx.length) + ' تراکنش</span></div></div><div class="wcard"><span>کل برداشت / تسویه</span><b><span class="num">' + num(W.debit) + '</span><small>تومان</small></b><div class="foot"><span>تسویه‌ای ثبت نشده</span></div></div></div>' +
      '<section class="panel main">' + h.banner('locked', '<b>خواندن محض.</b> باز کردن این صفحه هیچ اعتبار، ثبت یا محاسبه‌ای انجام نمی‌دهد. «ثبت‌شده» با «قابل تسویه» و «تسویه‌شده» یکی نیست.', '') + '<div class="tbl-wrap"><table class="tbl no-cursor"><thead><tr><th>تاریخ</th><th>نوع</th><th>کانال</th><th>شرح</th><th style="text-align:left">مبلغ</th></tr></thead><tbody>' + rows + '</tbody></table></div>' + h.tfoot(W.tx.length, W.tx.length) + '</section>';
  };

  /* ================= Shared Reports (multi-team scope) ================= */
  V.rep = function () {
    var teamOpts = '<option value="all"' + (st.rteam === 'all' ? ' selected' : '') + '>همه تیم‌های محدوده من</option>' + M.teams.map(function (t) { return '<option value="' + t.id + '"' + (st.rteam === t.id ? ' selected' : '') + '>تیم ' + esc(t.sup) + '</option>'; }).join('') + '<option value="direct"' + (st.rteam === 'direct' ? ' selected' : '') + '>فقط فروشنده مستقیم من</option>';
    var ctrl = '<div class="rep-ctrl"><div class="seg" role="group" aria-label="مبنای زمانی"><button type="button" data-rbasis="event" aria-pressed="' + (st.rbasis === 'event') + '">رویداد در بازه</button><button type="button" data-rbasis="now" aria-pressed="' + (st.rbasis === 'now') + '">وضعیت فعلی</button><button type="button" disabled aria-disabled="true" class="tip" data-tip="وضعیت تاریخی فقط با سابقه معتبر ممکن است؛ در این نمونه سابقه کافی ثبت نشده">وضعیت تاریخی</button></div>' +
      (st.rbasis === 'event' ? '<label class="lbl inline">از<input class="input" value="۱۴۰۵/۰۷/۰۱"></label><label class="lbl inline">تا<input class="input" value="۱۴۰۵/۰۷/۱۰"></label>' : '<span class="muted">وضعیت فعلی هنگام اجرای گزارش</span>') +
      '<label class="lbl inline">محدوده تیم<select class="input" data-rteam>' + teamOpts + '</select></label><span class="scope-note">فقط زیرمجموعه مجاز شما فهرست می‌شود؛ نقش‌های بالاتر انتخاب‌شدنی نیستند.</span></div>';
    var body;
    if (st.report) {
      var r = M.reports.filter(function (x) { return x.id === st.report; })[0];
      var lst = M.invoices.filter(function (i) { return st.rteam === 'all' || X.seller(i.seller).team === st.rteam; });
      var rows = lst.map(function (i) { return '<tr><td><span class="mono">' + fa(i.code) + '</span></td><td>' + esc(i.customer) + '</td><td>' + esc(seller(i.seller).name) + '</td><td>' + esc(X.teamName(seller(i.seller).team)) + '</td><td>' + money(i.total) + '</td><td>' + pill(X.INVS[i.inv].label, X.INVS[i.inv].tone, X.INVS[i.inv].icon) + '</td><td class="muted">' + esc(i.issued) + '</td></tr>'; }).join('');
      var cov = sim() === 'incomplete' ? 'partial' : sim() === 'stale' ? 'stale' : 'ok';
      body = '<section class="panel main"><div class="rep-head"><button type="button" class="btn btn-ghost btn-sm" data-act="close-report">' + ic('arrowL') + 'همه گزارش‌ها</button><h2>' + esc(r.name) + '</h2><span class="grow"></span>' + h.freshness({ text: 'محاسبه‌شده: امروز ۱۰:۴۲', refresh: true }) + '<button type="button" class="btn btn-sm" disabled aria-disabled="true" data-tip="خروجی فقط با مجوز صریح export و سیاست فیلد ممکن است؛ در این نمونه فعال نیست" ><span class="tip" style="display:contents">' + ic('download') + 'خروجی (نیازمند مجوز)</span></button></div>' +
        '<div class="rep-meta"><span class="chip-m">سطح: ' + esc(r.grain) + '</span><span class="chip-m">مبنا: ' + esc(st.rbasis === 'now' ? 'وضعیت فعلی' : r.basis) + '</span><span class="chip-m">محدوده: ' + (st.rteam === 'all' ? 'همه تیم‌های مجاز' : esc(X.teamName(st.rteam))) + '</span><span class="chip-m">تعریف شاخص: نسخه ۱</span><span class="chip-m">واحد پول: تومان</span><span class="chip-m">' + X.cov(cov) + '</span><span class="chip-m">خروجی: مجوز تأیید نشده</span></div>' +
        '<div class="kpis rep-kpis">' + h.kpi({ label: 'تعداد', value: fa(lst.length), color: 'neutral', meaning: 'تعداد ' + r.grain + ' یکتا؛ صفحه‌بندی فقط ردیف‌ها را محدود می‌کند', basis: 'همان محدوده جدول' }) + h.kpi({ label: 'جمع مبلغ', value: '<span class="num">' + num(lst.reduce(function (a, i) { return a + i.total; }, 0)) + '</span><small>تومان</small>', color: 'teal', meaning: 'جمع همان ردیف‌ها با همان تعریف', basis: 'همان محدوده جدول' }) + '</div>' +
        (lst.length ? '<div class="tbl-wrap"><table class="tbl no-cursor" aria-label="' + esc(r.name) + '"><thead><tr><th>کد</th><th>مشتری</th><th>فروشنده</th><th>تیم فعلی</th><th>مبلغ</th><th>وضعیت فاکتور</th><th>تاریخ صدور</th></tr></thead><tbody>' + rows + '</tbody></table></div>' + h.tfoot(lst.length, lst.length) : h.stateBlock('noresult', 'ردیفی در این محدوده نیست', 'محدوده تیم را تغییر دهید؛ دسترسی بیرون از محدوده مجاز ایجاد نمی‌شود.', '<button type="button" class="btn btn-soft" data-act="rteam-all">همه تیم‌ها</button>')) + '</section>';
    } else {
      body = '<div class="rep-grid">' + M.reports.map(function (r) { return '<button type="button" class="rep-card" data-act="open-report:' + r.id + '"><span class="rep-ico">' + ic(r.icon) + '</span><b>' + esc(r.name) + '</b><span class="rep-tags"><span>سطح: ' + esc(r.grain) + '</span><span>مبنا: ' + esc(r.basis) + '</span></span><span class="rep-go">مشاهده آمار و ریزداده ' + ic('arrowL') + '</span></button>'; }).join('') + '</div>';
    }
    return h.pageHead({ title: 'گزارش‌ها', sub: 'گزارش‌های مشترک CRM با محدوده چندتیمی شما · هر گزارش سطح شمارش، مبنای زمانی، پوشش و تازگی خودش را دارد' }) + '<section class="panel rep-ctrl-panel">' + ctrl + '</section>' + body;
  };
})();
