/* Supervisor panel — role layer on the Shared CRM runtime (../shared/crm-core.js). Design prototype; mock data only.
   Nothing here assigns, returns, uploads, sends or approves anything. Product boundaries follow Gate 0 §19. */
(function () {
  'use strict';
  var C = window.CRM, M = window.SUP, h = C.h;
  var $ = h.$, $$ = h.$$, esc = h.esc, fa = h.fa, num = h.num, money = h.money, ic = h.ic, pill = h.pill, hint = h.hint, cbx = h.cbx;

  /* ---------- Role state ---------- */
  var st = {
    aq: 'assign', amode: 'select', asel: {}, recip: null, recips: {}, per: 3,
    rf: 'all', rsel: {}, cq: 'team', csel: {}, iq: 'action', report: null, rbasis: 'event',
    flow: null, live: null, showMine: true, hrErr: null
  };

  /* ---------- Namespaced status dictionaries (label + tone + icon; never colour alone) ---------- */
  var ELIG = { ok: { label: 'قابل تخصیص', tone: 'teal', icon: 'checkCircle' }, blocked: { label: 'مسدود', tone: 'red', icon: 'lock' }, unknown: { label: 'نیاز به تطبیق', tone: 'amber', icon: 'question' }, conflict: { label: 'تغییر کرده', tone: 'orange', icon: 'swap' } };
  var RET = { ok: { label: 'قابل برگشت', tone: 'teal', icon: 'checkCircle' }, blocked: { label: 'مسدود', tone: 'red', icon: 'lock' }, unknown: { label: 'نیاز به تطبیق', tone: 'amber', icon: 'question' }, conflict: { label: 'تغییر کرده', tone: 'orange', icon: 'swap' } };
  var INVS = { pre: { label: 'پیش‌فاکتور', tone: 'violet', icon: 'file' }, staged: { label: 'مرحله‌ای', tone: 'teal', icon: 'layers' }, completed: { label: 'تکمیل‌شده', tone: 'green', icon: 'checkCircle' }, cancelled: { label: 'لغو شده', tone: 'slate', icon: 'ban' }, mismatch: { label: 'مغایرت', tone: 'amber', icon: 'question' } };
  var STG = { pending: 'در انتظار پرداخت', review: 'رسید در انتظار بررسی', rejected: 'رد شده', approved: 'تأییدشده', cancelled: 'لغو' };
  var EVID = { none: '—', receipt: 'رسید ثبت‌شده', online: 'پرداخت آنلاین (درگاه)' };
  var REV = { none: null, pending: { label: 'در انتظار مالی', tone: 'orange', icon: 'hourglass' }, approved: { label: 'تأیید مالی', tone: 'green', icon: 'checkCircle' }, rejected: { label: 'رد مالی', tone: 'red', icon: 'alert' } };
  var OUT = { ok: { label: 'ثبت شد', tone: 'green', icon: 'checkCircle' }, retry: { label: 'ناموفق · قابل تکرار', tone: 'red', icon: 'refresh' }, failed: { label: 'ناموفق', tone: 'red', icon: 'xCircle' }, conflict: { label: 'تعارض', tone: 'orange', icon: 'swap' }, unknown: { label: 'نامعلوم', tone: 'amber', icon: 'question' } };
  var HRS = { pending_review: { label: 'در انتظار بررسی مرحله‌ای', tone: 'orange', icon: 'hourglass' }, pending_hr: { label: 'در انتظار منابع انسانی', tone: 'orange', icon: 'hourglass' }, approved: { label: 'اعمال‌شده', tone: 'green', icon: 'checkCircle' }, rejected: { label: 'رد شده', tone: 'red', icon: 'xCircle' }, failed: { label: 'اعمال ناموفق', tone: 'red', icon: 'alert' } };

  var seller = function (id) { return M.sellers.filter(function (s) { return s.id === id; })[0]; };
  var active = function () { return M.sellers.filter(function (s) { return s.active; }); };
  var sum = function (k) { return M.sellers.reduce(function (a, s) { return a + s[k]; }, 0); };
  var keys = function (o) { return Object.keys(o).filter(function (k) { return o[k]; }); };
  var poolOk = function () { return M.pool.filter(function (c) { return c.elig === 'ok'; }); };
  var sim = function () { return C.state.sim; };
  var fresh = function (label) { return { text: label + ': به‌روزشده ' + (sim() === 'stale' ? M.freshness.statsStale : C.state.fresh === 'now' ? 'همین الان' : M.freshness.stats), stale: sim() === 'stale' }; };
  var phoneCell = function (p) { return '<span class="mono phone-num">' + fa(p) + '</span>'; };
  var who = function (name, sub) { return '<div class="who"><b>' + esc(name) + '</b>' + (sub ? '<span class="phone">' + sub + '</span>' : '') + '</div>'; };
  var nextActor = function (n, inv) {
    if (!n) return '<span class="muted">—</span>';
    var lbl = { customer: 'مشتری', seller: 'فروشنده' + (inv ? ': ' + seller(inv.seller).name : ''), finance: 'واحد مالی', mis: 'MIS', self: 'شما' }[n.who];
    return '<div class="actor"><span class="actor-who a-' + n.who + '">' + ic(n.who === 'finance' ? 'hourglass' : n.who === 'mis' ? 'layers' : 'user') + esc(lbl) + '</span><span class="actor-what">' + esc(n.text) + '</span></div>';
  };
  function staleBanners() {
    if (sim() === 'stale') return h.banner('stale', '<b>اطلاعات این صفحه ' + M.freshness.statsStale + ' محاسبه شده است.</b> اعداد ممکن است با وضعیت فعلی پرونده‌ها فرق داشته باشند؛ پیش از تصمیم بازخوانی کنید.', '<button type="button" class="btn btn-sm" data-refresh>' + ic('refresh') + 'بازخوانی</button>');
    if (sim() === 'incomplete') return h.banner('incomplete', '<b>بخشی از داده‌ها دریافت نشد: آمار فاکتورهای تیم.</b> شاخص‌های وابسته به‌جای صفر «نامعتبر» نمایش داده می‌شوند؛ بقیه صفحه معتبر است.', '<button type="button" class="btn btn-sm" data-refresh>' + ic('refresh') + 'تلاش مجدد</button>');
    return '';
  }
  var finKpi = function (k) { if (sim() === 'incomplete') k.state = 'unavailable'; else if (sim() === 'stale') k.state = 'stale'; return k; };
  var steps = function (list, cur) { return '<ol class="steps" aria-label="مراحل">' + list.map(function (s, i) { var c = i < cur ? 'done' : i === cur ? 'cur' : ''; return '<li class="' + c + '"' + (i === cur ? ' aria-current="step"' : '') + '><span class="sn">' + (i < cur ? ic('check') : fa(i + 1)) + '</span><span class="st-l">' + esc(s) + '</span></li>'; }).join('') + '</ol>'; };
  var bulkbar = function (text, sub, actions) { return '<div class="bulkbar" role="region" aria-label="انتخاب گروهی"><div class="bb-txt"><b>' + text + '</b>' + (sub ? '<span>' + sub + '</span>' : '') + '</div><span class="grow"></span>' + actions + '</div>'; };

  /* ---------- Attention (exception-first; only categories backed by existing data) ---------- */
  function attention() {
    var dueBy = M.sellers.filter(function (s) { return s.due; }).map(function (s) { return fa(s.due) + ' نزد ' + s.name; }).join('، ');
    var inact = M.sellers.filter(function (s) { return !s.active && s.open; });
    var opIssue = M.ops.filter(function (o) { return o.failed || o.conflict || o.unknown; });
    var finBlocked = M.held.filter(function (c) { return c.elig === 'blocked'; }).length;
    var invAction = M.invoices.filter(function (i) { return i.bucket === 'action'; });
    return [
      { id: 'due', n: sum('due'), tone: 'red', icon: 'clock', label: 'تماس سررسیده', sub: dueBy, owner: 'فروشنده', act: 'attn:due' },
      { id: 'inv', n: invAction.length, tone: 'red', icon: 'receipt', label: 'فاکتور نیازمند اقدام', sub: 'فیش ردشده، لینک ارسال‌نشده، مغایرت ثبت', owner: 'فروشنده / MIS', go: 'inv', iq: 'action' },
      { id: 'inactive', n: inact.reduce(function (a, s) { return a + s.open; }, 0), tone: 'amber', icon: 'user', label: 'پرونده نزد فروشنده غیرفعال', sub: inact.map(function (s) { return s.name + ' · ' + s.inactiveNote; }).join('، '), owner: 'شما', go: 'assign', aq: 'return' },
      { id: 'ops', n: opIssue.length, tone: 'orange', icon: 'split', label: 'عملیات ناقص یا نامعلوم', sub: opIssue.map(function (o) { return o.ref; }).join('، '), owner: 'شما', go: 'assign', aq: 'ops' },
      { id: 'fin', n: finBlocked, tone: 'slate', icon: 'lock', label: 'پرونده دارای وابستگی مالی', sub: 'با برگشت عادی آزاد نمی‌شوند', owner: 'اطلاع', go: 'assign', aq: 'return', rf: 'blocked' }
    ].filter(function (a) { return a.n > 0; });
  }

  /* ================= Views ================= */
  var V = {};

  V.team = function () {
    var A = attention(), act = active().length;
    var kpis = [
      { label: 'فروشنده فعال مستقیم', value: fa(act) + '<small>از ' + fa(M.sellers.length) + '</small>', color: 'teal', keep: true, meaning: 'فروشندگانی که مستقیم زیر نظر شما هستند و الان فعال‌اند', basis: 'وضعیت در لحظه' },
      { label: 'پرونده باز تیم', value: fa(sum('open')), color: 'neutral', keep: true, meaning: 'پرونده‌های فعال نزد فروشندگان مستقیم', basis: 'وضعیت در لحظه' },
      { label: 'تماس سررسیده', value: fa(sum('due')), tone: 'bad', color: 'red', meaning: 'تماس مجددی که زمانش گذشته و نتیجه‌ای ثبت نشده', basis: 'وضعیت در لحظه' },
      finKpi({ label: 'پیش‌فاکتور باز', value: fa(sum('openPre')), color: 'violet', filter: 'inv:pre', meaning: 'پیش‌فاکتورهای فعلی بدون تکمیل یا لغو', basis: 'وضعیت در لحظه' }),
      finKpi({ label: 'فروش تکمیل‌شده (مهر)', value: fa(1), color: 'green', meaning: 'فاکتورهایی که پرداخت کامل و تأییدشده دارند — نه صرف صدور', basis: 'رویداد تکمیل از ۱ مهر تا امروز' })
    ];
    var attn = '<section class="panel attn-panel" aria-labelledby="attn-h"><div class="sec-h ph"><h2 id="attn-h">' + ic('inbox') + 'نیازمند توجه</h2><span class="aside">' + fa(A.length) + ' دسته</span></div>' +
      (A.length ? '<ul class="attn">' + A.map(function (a) {
        return '<li><button type="button" class="attn-row" data-attn="' + a.id + '" style="--c:var(--' + a.tone + '-dot)"><span class="attn-ico t-' + a.tone + '">' + ic(a.icon) + '</span><span class="attn-txt"><b>' + esc(a.label) + '</b><span><span class="attn-own">مسئول: ' + esc(a.owner) + '</span>' + esc(a.sub) + '</span></span><span class="attn-n" aria-label="' + fa(a.n) + ' مورد">' + fa(a.n) + '</span>' + ic('arrowL') + '</button></li>';
      }).join('') + '</ul>' : h.stateBlock('empty', 'موردی نیازمند توجه نیست', 'استثنای ثبت‌شده‌ای در تیم شما وجود ندارد.')) +
      '<div class="attn-foot">' + ic('info') + '<span>درخواست HR منتظر اقدام شما: ندارد · فقط دسته‌هایی نمایش داده می‌شوند که داده معتبر دارند.</span></div></section>';
    var rows = M.sellers.map(function (s) {
      var actN = M.invoices.filter(function (i) { return i.seller === s.id && i.next && i.next.who === 'seller'; }).length;
      var pre = sim() === 'incomplete' ? '<span class="na tip" data-tip="آمار فاکتور دریافت نشد">—</span>' : fa(s.openPre);
      return '<tr data-row="seller:' + s.id + '" tabindex="-1"' + (s.active ? '' : ' class="inactive"') + '><td>' + who(s.name, phoneCell(s.phone)) + '</td>' +
        '<td>' + (s.active ? pill('فعال', 'teal', 'checkCircle') : pill('غیرفعال', 'slate', 'ban') + '<div class="cell-sub">' + esc(s.inactiveNote) + '</div>') + '</td>' +
        '<td class="n">' + fa(s.open) + '</td><td class="n">' + fa(s.untouched) + '</td>' +
        '<td class="n">' + (s.due ? '<span class="due-n">' + ic('clock') + fa(s.due) + '</span>' : '<span class="muted">۰</span>') + '</td>' +
        '<td class="n">' + (actN ? '<span class="warn-n">' + fa(actN) + '</span>' : '<span class="muted">۰</span>') + '</td>' +
        '<td class="n">' + pre + '</td><td class="col-opt muted">' + esc(s.lastAssign) + '</td>' +
        '<td class="col-actions"><div class="row-actions">' + (s.active ? '<button type="button" class="btn btn-soft btn-sm" data-act="assign-to:' + s.id + '">تخصیص</button>' : '<button type="button" class="btn btn-sm" data-act="return-from:' + s.id + '">بررسی برگشت</button>') +
        '<button type="button" class="btn btn-ghost btn-icon btn-sm tip" data-tip="بیشتر" data-act="seller-more:' + s.id + '" aria-label="اقدام‌های بیشتر برای ' + esc(s.name) + '">' + ic('more') + '</button></div></td></tr>';
    }).join('');
    var cap = '<section class="panel main cap-panel" aria-labelledby="cap-h"><div class="sec-h ph"><h2 id="cap-h">' + ic('users') + 'ظرفیت تیم مستقیم</h2><span class="scope-badge">' + ic('users') + 'مستقیم · قابل تخصیص</span><span class="aside">مرتب‌شده بر اساس نام · رتبه‌بندی نیست</span></div>' +
      '<div class="tbl-wrap"><table class="tbl" aria-label="ظرفیت فروشندگان مستقیم"><thead><tr><th>فروشنده</th><th>وضعیت</th><th class="n">پرونده باز</th><th class="n">بدون وضعیت ' + hint('پرونده‌هایی که هنوز نتیجه تماسی برایشان ثبت نشده است.', true) + '</th><th class="n">تماس سررسیده</th><th class="n">نیازمند اقدام ' + hint('فاکتورهای این فروشنده که اقدام بعدی‌شان با خود اوست.', true) + '</th><th class="n">پیش‌فاکتور باز</th><th class="col-opt">آخرین تخصیص</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div></section>';
    var ind = '<section class="panel ind-panel"><div class="sec-h ph"><h2>' + ic('eye') + 'زیرمجموعه غیرمستقیم</h2><span class="scope-badge view">' + ic('eye') + 'فقط مشاهده</span></div><p class="ind-note">در ساختار فعلی، زیرمجموعه غیرمستقیمی برای شما ثبت نشده است. اگر وجود داشته باشد، فقط برای مشاهده و گزارش نمایش داده می‌شود و گیرنده تخصیص نیست.</p></section>';
    return C.h.pageHead({ title: 'نمای تیم', sub: 'تیم مستقیم: ' + fa(M.sellers.length) + ' فروشنده (' + fa(act) + ' فعال) · سرپرست بالادست: ' + esc(M.user.parent), kpis: kpis, fresh: fresh('آمار تیم'), scope: 'محدوده: تیم مستقیم شما' }) + staleBanners() +
      '<div class="team-grid">' + attn + '<div class="team-main">' + cap + ind + '</div></div>';
  };

  /* ----- Assignment & Return ----- */
  V.assign = function () {
    var ok = poolOk().length, review = M.pool.length - ok, issue = M.ops.filter(function (o) { return o.failed || o.conflict || o.unknown; }).length;
    var kpis = [
      { label: 'موجودی پنل من', value: fa(M.pool.length), color: 'neutral', keep: true, meaning: 'پرونده‌هایی که به شما تحویل شده و هنوز نزد فروشنده نیستند', basis: 'وضعیت در لحظه' },
      { label: 'قابل تخصیص', value: fa(ok), color: 'teal', keep: true, meaning: 'بدون وابستگی مالی و بدون تعارض — در زمان ثبت دوباره بررسی می‌شود', basis: 'وضعیت در لحظه' },
      { label: 'نیازمند بررسی', value: fa(review), tone: review ? 'warn' : '', color: 'amber', meaning: 'مسدود یا نیازمند تطبیق', basis: 'وضعیت در لحظه' },
      { label: 'گیرنده مجاز', value: fa(active().length), color: 'blue', meaning: 'فقط فروشندگان مستقیم فعال', basis: 'وضعیت در لحظه' }
    ];
    var q = h.queues([
      { id: 'assign', label: 'تخصیص به فروشنده', icon: 'send', tone: 'blue', n: ok, key: '1' },
      { id: 'return', label: 'برگشت به پنل من', icon: 'repeat', tone: 'teal', n: M.held.filter(function (c) { return c.elig === 'ok'; }).length, key: '2' },
      { id: 'ops', label: 'سوابق عملیات', icon: 'history', tone: issue ? 'orange' : 'neutral', n: M.ops.length, key: '3' }
    ], st.aq, 'data-aq');
    var body = st.aq === 'return' ? returnView() : st.aq === 'ops' ? opsView() : assignView();
    return C.h.pageHead({ title: 'تخصیص و برگشت', sub: 'پرونده‌های پنل شما ← فروشندگان مستقیم · برگشت فقط برای پرونده‌های بدون وابستگی مالی', kpis: kpis, fresh: fresh('فهرست') }) + staleBanners() +
      '<section class="panel main' + (st.aq === 'ops' ? '' : ' with-bulk') + '">' + q + body + '</section>';
  };

  function assignView() {
    var selIds = keys(st.asel), rec = st.recip ? seller(st.recip) : null, recN = keys(st.recips).length;
    var stepCur = st.amode === 'select' ? (!selIds.length ? 0 : !rec ? 1 : 2) : (!recN ? 1 : 2);
    var head = '<div class="flow-head">' + steps(['انتخاب پرونده‌ها', 'انتخاب گیرنده', 'بررسی اثر', 'ثبت و نتیجه'], stepCur) +
      '<div class="seg" role="group" aria-label="روش تخصیص"><button type="button" data-amode="select" aria-pressed="' + (st.amode === 'select') + '">انتخاب پرونده‌ها</button><button type="button" data-amode="count" aria-pressed="' + (st.amode === 'count') + '">تعداد برای هر گیرنده</button></div></div>';
    var left;
    if (st.amode === 'select') {
      var okIds = poolOk().map(function (c) { return c.id; }), allOn = okIds.length && okIds.every(function (id) { return st.asel[id]; });
      var rows = M.pool.map(function (c) {
        var e = ELIG[c.elig], dis = c.elig !== 'ok';
        return '<tr data-row="case:' + c.id + '" tabindex="-1" class="' + (st.asel[c.id] ? 'picked' : '') + (dis ? ' dim' : '') + '"><td class="col-sel">' + cbx('data-sel="asel" data-id="' + c.id + '"', st.asel[c.id], 'انتخاب پرونده ' + c.id + (dis ? ' — ' + e.label : ''), dis) + '</td>' +
          '<td><span class="mono case-id">' + esc(c.id) + '</span></td><td>' + phoneCell(c.phone) + '</td><td><span class="tag">' + esc(c.source) + '</span></td><td class="col-opt muted">' + esc(c.received) + '</td>' +
          '<td>' + pill(e.label, e.tone, e.icon) + (c.why ? '<div class="cell-sub">' + esc(c.why) + '</div>' : '') + '</td></tr>';
      }).join('');
      left = h.toolbar('شماره پرونده، موبایل یا دسته', [{ label: 'منبع / دسته', icon: 'filter' }, { label: 'تاریخ دریافت', icon: 'clock' }], '<span class="muted desk">' + fa(M.pool.length) + ' پرونده در پنل شما</span>') +
        '<div class="tbl-wrap"><table class="tbl selectable" aria-label="پرونده‌های قابل تخصیص"><thead><tr><th class="col-sel" data-l="انتخاب">' + cbx('data-selall="asel"', allOn, 'انتخاب همه پرونده‌های قابل تخصیص') + '</th><th>پرونده</th><th>شماره</th><th>منبع</th><th class="col-opt">دریافت</th><th>قابلیت تخصیص ' + hint('در زمان ثبت دوباره بررسی می‌شود؛ پیش‌نمایش پرونده را رزرو نمی‌کند.', true) + '</th></tr></thead><tbody>' + rows + '</tbody></table></div>';
    } else {
      var total = st.per * recN, okN = poolOk().length;
      left = '<div class="count-form"><label class="lbl">تعداد پرونده برای هر گیرنده <span class="req">*</span><input class="input" type="number" min="1" max="50" value="' + st.per + '" data-per inputmode="numeric" style="max-width:160px"></label>' +
        '<label class="lbl">انتخاب پرونده / کد دیتا<select class="input" style="max-width:320px"><option>همه پرونده‌های تحویل‌شده</option><option>MIS · دسته ۸۸۱۲</option><option>MIS · دسته ۸۸۱۳</option></select></label>' +
        '<div class="note info">' + ic('info') + '<span>' + (recN ? fa(st.per) + ' × ' + fa(recN) + ' گیرنده = <b>' + fa(total) + ' پرونده</b> از ' + fa(okN) + ' پرونده قابل تخصیص، به ترتیب قدیمی‌ترین دریافت.' + (total > okN ? ' <b>تعداد درخواستی بیشتر از موجودی قابل تخصیص است.</b>' : '') : 'گیرنده‌ها را از فهرست انتخاب کنید.') + '</span></div>' +
        '<div class="muted" style="font-size:var(--t-meta)">این همان روش فعلی تخصیص است؛ فهرست دقیق پرونده‌ها در مرحله بررسی نمایش داده می‌شود.</div></div>';
    }
    var recList = active().concat(M.sellers.filter(function (s) { return !s.active; })).map(function (s) {
      var single = st.amode === 'select', on = single ? st.recip === s.id : !!st.recips[s.id];
      return '<label class="recip' + (s.active ? '' : ' off') + (on ? ' on' : '') + '"><input type="' + (single ? 'radio' : 'checkbox') + '" name="recip" class="cbx" data-recip="' + s.id + '"' + (on ? ' checked' : '') + (s.active ? '' : ' disabled') + '>' +
        '<span class="r-main"><b>' + esc(s.name) + '</b><span>' + (s.active ? 'باز ' + fa(s.open) + '، بدون وضعیت ' + fa(s.untouched) + '، سررسیده ' + fa(s.due) : 'غیرفعال — گیرنده تخصیص نیست') + '</span></span></label>';
    }).join('');
    var aside = '<aside class="recips" aria-labelledby="rc-h"><div class="sec-h"><h3 id="rc-h">گیرنده ' + (st.amode === 'select' ? '(یک نفر)' : '(چند نفر)') + '</h3><span class="scope-badge">' + ic('users') + 'فقط مستقیم</span></div>' + recList +
      '<p class="recip-note">' + ic('info') + 'کاربران غیرمستقیم و فروشندگان غیرفعال گیرنده نیستند. دیدن یک کاربر به معنی اجازه تخصیص به او نیست.</p></aside>';
    var bb = '';
    if (st.amode === 'select' && selIds.length) bb = bulkbar(fa(selIds.length) + ' پرونده انتخاب شد', rec ? 'گیرنده: ' + esc(rec.name) : 'یک گیرنده انتخاب کنید', '<button type="button" class="btn btn-ghost btn-sm" data-act="clear-asel">پاک کردن انتخاب</button><button type="button" class="btn btn-primary" data-act="review-assign"' + (rec ? '' : ' disabled aria-disabled="true"') + '>' + ic('eye') + 'بررسی اثر و تخصیص</button>');
    if (st.amode === 'count' && recN) bb = bulkbar(fa(st.per * recN) + ' پرونده برای ' + fa(recN) + ' گیرنده', keys(st.recips).map(function (id) { return seller(Number(id)).name; }).join('، '), '<button type="button" class="btn btn-primary" data-act="review-assign">' + ic('eye') + 'بررسی اثر و تخصیص</button>');
    return head + '<div class="split"><div class="split-main">' + left + '</div>' + aside + '</div>' + bb;
  }

  function heldList() {
    var byFilter = function (c) { return st.rf === 'all' ? true : st.rf === 'legacy' ? !!c.legacy : st.rf === 'review' ? (c.elig === 'unknown' || c.elig === 'conflict') : c.elig === st.rf; };
    return M.held.filter(function (c) { return byFilter(c) && (!st.rseller || c.seller === st.rseller); });
  }
  function retWhy(c) {
    if (c.elig === 'ok') return c.invoice ? '' : 'بدون فاکتور یا پرداخت متصل';
    if (c.why === 'fin') return 'وابستگی مالی: ' + c.invoice.label;
    if (c.why === 'cancel') return 'فاکتور لغوشده — قانون آزادسازی تعیین نشده';
    return c.why;
  }
  function returnView() {
    var list = heldList(), okIds = list.filter(function (c) { return c.elig === 'ok'; }).map(function (c) { return c.id; });
    var selIds = keys(st.rsel).filter(retCase), allOn = okIds.length && okIds.every(function (id) { return st.rsel[id]; });
    var seg = '<div class="seg" role="group" aria-label="فیلتر قابلیت برگشت">' + [['all', 'همه'], ['ok', 'قابل برگشت'], ['blocked', 'مسدود'], ['review', 'نیازمند بررسی'], ['legacy', 'سرنخ قدیمی']].map(function (f) { return '<button type="button" data-rf="' + f[0] + '" aria-pressed="' + (st.rf === f[0]) + '">' + f[1] + '</button>'; }).join('') + '</div>';
    var rows = list.map(function (c) {
      var e = RET[c.elig], s = seller(c.seller), dis = c.elig !== 'ok';
      return '<tr data-row="ret:' + c.id + '" tabindex="-1" class="' + (st.rsel[c.id] ? 'picked' : '') + '"><td class="col-sel">' + cbx('data-sel="rsel" data-id="' + c.id + '"', st.rsel[c.id], 'انتخاب پرونده ' + c.id + (dis ? ' — ' + e.label : ''), dis) + '</td>' +
        '<td><span class="mono case-id">' + esc(c.id) + '</span>' + (c.legacy ? ' <span class="tag legacy">قدیمی</span>' : '') + '<div class="cell-sub">' + phoneCell(c.phone) + '</div></td>' +
        '<td>' + who(s.name, s.active ? '' : '<span class="warn-n">غیرفعال</span>') + '</td><td class="col-opt"><span class="tag">' + esc(c.source) + '</span></td>' +
        '<td class="col-opt"><span class="ns">تماس:</span> ' + esc(c.contact) + '</td>' +
        '<td>' + (c.invoice ? '<span class="mono">' + fa(c.invoice.code) + '</span><div class="cell-sub">' + esc(c.invoice.label) + '</div>' : '<span class="muted">ندارد</span>') + '</td>' +
        '<td>' + pill(e.label, e.tone, e.icon) + (retWhy(c) ? '<div class="cell-sub">' + esc(retWhy(c)) + '</div>' : '') + '</td>' +
        '<td class="col-actions"><button type="button" class="btn btn-sm" data-act="open-ret:' + c.id + '">بررسی</button></td></tr>';
    }).join('');
    var bb = selIds.length ? bulkbar(fa(selIds.length) + ' پرونده قابل برگشت انتخاب شد', 'فقط پرونده‌های بدون وابستگی مالی قابل انتخاب‌اند', '<button type="button" class="btn btn-ghost btn-sm" data-act="clear-rsel">پاک کردن انتخاب</button><button type="button" class="btn btn-primary" data-act="review-return">' + ic('eye') + 'بررسی برگشت</button>') : '';
    return '<div class="note info inset">' + ic('shield') + '<span>برگشت، پرونده را از فروشنده به پنل شما برمی‌گرداند. پرونده‌هایی که فاکتور، پیش‌فاکتور یا پرداخت دارند با برگشت عادی آزاد نمی‌شوند؛ دلیل هر مورد کنار آن نوشته شده است.' + (st.rseller ? ' <b>فیلتر: ' + esc(seller(st.rseller).name) + '</b> <button type="button" class="linkish" data-act="clear-rseller">حذف فیلتر</button>' : '') + '</span></div>' +
      '<div class="toolbar">' + seg + '<span class="grow"></span><span class="muted desk">' + fa(list.length) + ' پرونده</span></div>' +
      (list.length ? '<div class="tbl-wrap"><table class="tbl selectable" aria-label="پرونده‌های نزد فروشندگان"><thead><tr><th class="col-sel" data-l="انتخاب">' + cbx('data-selall="rsel"', allOn, 'انتخاب همه پرونده‌های قابل برگشت') + '</th><th>پرونده</th><th>نزد (مسئول فعلی)</th><th class="col-opt">منبع</th><th class="col-opt">وضعیت تماس</th><th>فاکتور مرتبط</th><th>قابلیت برگشت</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div>'
        : h.stateBlock('noresult', 'پرونده‌ای با این فیلتر نیست', 'فیلتر دیگری انتخاب کنید.', '<button type="button" class="btn btn-soft" data-rf="all">همه پرونده‌ها</button>')) + bb;
  }

  function opState(o) { return o.unknown ? 'unknown' : (o.failed || o.conflict) && o.ok ? 'partial' : (o.failed || o.conflict) ? 'failed' : 'complete'; }
  var OPS = { complete: ['کامل', 'green', 'checkCircle'], partial: ['ناقص', 'orange', 'split'], failed: ['ناموفق', 'red', 'xCircle'], unknown: ['نامعلوم', 'amber', 'question'] };
  function opsView() {
    var rows = M.ops.map(function (o) {
      var s = OPS[opState(o)];
      return '<tr data-row="result:' + o.ref + '" tabindex="-1"><td><span class="mono case-id">' + o.ref + '</span></td><td>' + (o.kind === 'assign' ? 'تخصیص' : 'برگشت') + '</td><td class="muted">' + esc(o.when) + '</td><td class="col-opt">' + esc(o.to) + '</td>' +
        '<td>' + pill(s[0], s[1], s[2]) + '<div class="cell-sub">' + (o.unknown ? fa(o.unknown) + ' مورد نامعلوم' : fa(o.ok) + ' از ' + fa(o.total) + ' ثبت شد') + '</div></td>' +
        '<td class="col-actions"><button type="button" class="btn btn-sm' + (opState(o) === 'complete' ? '' : ' btn-soft') + '" data-act="open-op:' + o.ref + '">' + (opState(o) === 'complete' ? 'جزئیات' : 'رسیدگی') + '</button></td></tr>';
    }).join('');
    return h.toolbar('شماره عملیات یا پرونده', [{ label: 'نوع عملیات', icon: 'filter' }, { label: 'بازه زمانی', icon: 'clock' }]) +
      '<div class="tbl-wrap"><table class="tbl" aria-label="سوابق عملیات"><thead><tr><th>عملیات</th><th>نوع</th><th>زمان</th><th class="col-opt">مقصد</th><th>نتیجه</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div>' + h.tfoot(M.ops.length, M.ops.length);
  }

  /* ----- Ready Conversion ----- */
  V.conv = function () {
    var R = M.ready, waiting = R.filter(function (r) { return r.status === 'ready'; });
    var kpis = [
      { label: 'آماده تخصیص', value: fa(waiting.length), color: 'blue', keep: true, meaning: 'پرونده‌های آماده تبدیل که هنوز به کسی تخصیص نشده‌اند', basis: 'وضعیت در لحظه' },
      { label: 'انتخاب ثبت‌شده', value: fa(R.filter(function (r) { return r.choice; }).length), color: 'teal', meaning: 'مشتری محصول نهایی را انتخاب کرده است', basis: 'وضعیت در لحظه' },
      { label: 'انتخاب نکرده', value: fa(R.filter(function (r) { return !r.choice; }).length), color: 'amber', meaning: 'محصول نهایی هنگام تماس انتخاب می‌شود', basis: 'وضعیت در لحظه' },
      { label: 'تبدیل‌کننده فعال', value: fa(M.converters.length), color: 'neutral', meaning: 'تبدیل‌کنندگان فعال زیرمجموعه شما', basis: 'وضعیت در لحظه' },
      { label: 'نرخ تبدیل', state: 'undefined', color: 'slate', meaning: 'تا تعیین تعریف رسمی (دوره و جامعه آماری) نمایش داده نمی‌شود؛ صفر نیست' }
    ];
    var mine = st.showMine && M.myConversions.length;
    var qi = [{ id: 'team', label: 'صف تیم', icon: 'inbox', tone: 'blue', n: waiting.length, key: '1' }, { id: 'converters', label: 'تبدیل‌کنندگان', icon: 'users', tone: 'neutral', n: M.converters.length, key: '2' }];
    if (mine) qi.push({ id: 'mine', label: 'تبدیل‌های من', icon: 'user', tone: 'violet', n: M.myConversions.length, key: '3' });
    if (st.cq === 'mine' && !mine) st.cq = 'team';
    var body;
    if (st.cq === 'converters') body = h.stateBlock('empty', 'تبدیل‌کننده فعالی زیر این سرپرست تعریف نشده است', 'تا زمانی که تبدیل‌کننده‌ای تعریف نشده، پرونده‌های آماده را می‌توانید به خودتان تخصیص دهید.', '<button type="button" class="btn btn-soft" data-cq="team">رفتن به صف تیم</button>');
    else if (st.cq === 'mine') body = mineView();
    else {
      var okIds = waiting.map(function (r) { return r.id; }), allOn = okIds.length && okIds.every(function (id) { return st.csel[id]; });
      var rows = R.map(function (r) {
        var assigned = r.status !== 'ready';
        return '<tr data-row="rcase:' + r.id + '" tabindex="-1" class="' + (st.csel[r.id] ? 'picked' : '') + '"><td class="col-sel">' + cbx('data-sel="csel" data-id="' + r.id + '"', st.csel[r.id], 'انتخاب پرونده ' + r.id + (assigned ? ' — قبلاً تخصیص شده' : ''), assigned) + '</td>' +
          '<td><span class="mono case-id">' + r.id + '</span><div class="cell-sub">فاکتور ' + fa(r.invoice) + '</div></td><td>' + who(r.customer, phoneCell(r.phone)) + '</td><td>' + money(r.credit) + '</td>' +
          '<td>' + (r.choice ? '<span class="tag">' + esc(r.choice) + '</span>' : pill('انتخاب نکرده', 'amber', 'question')) + '</td>' +
          '<td class="col-opt">' + pill('پرداخت تکمیل · تأیید مالی', 'green', 'checkCircle') + '</td>' +
          '<td>' + (assigned ? pill('تخصیص به شما', 'violet', 'user') : '<span class="muted">—</span>') + '</td>' +
          '<td class="col-actions">' + (assigned ? '<button type="button" class="btn btn-sm" data-cq="mine">باز کردن</button>' : '<button type="button" class="btn btn-soft btn-sm" data-act="conv-one:' + r.id + '">تخصیص</button>') + '</td></tr>';
      }).join('');
      var sel = keys(st.csel);
      body = '<div class="note info inset">' + ic('info') + '<span>«آماده تبدیل» یعنی پرداخت فاکتور اعتبارسنجی تکمیل و تأیید مالی شده است؛ به معنی فروش نهایی نیست.</span></div>' + h.toolbar('پرونده، مشتری یا موبایل', [{ label: 'انتخاب مشتری', icon: 'filter' }]) +
        '<div class="tbl-wrap"><table class="tbl selectable" aria-label="صف آماده تبدیل تیم"><thead><tr><th class="col-sel" data-l="انتخاب">' + cbx('data-selall="csel"', allOn, 'انتخاب همه پرونده‌های آماده') + '</th><th>پرونده</th><th>مشتری</th><th>اعتبار</th><th>انتخاب مشتری</th><th class="col-opt">پرداخت</th><th>تبدیل‌کننده</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div>' +
        (sel.length ? bulkbar(fa(sel.length) + ' پرونده انتخاب شد', '', '<button type="button" class="btn btn-ghost btn-sm" data-act="clear-csel">پاک کردن انتخاب</button><button type="button" class="btn btn-primary" data-act="conv-many">تخصیص…</button>') : '');
    }
    return C.h.pageHead({ title: 'تبدیل', sub: 'صف آماده تخصیص تبدیل تیم' + (mine ? ' · تبدیل‌هایی که به خودتان تخصیص داده‌اید جدا نمایش داده می‌شوند' : ''), kpis: kpis, fresh: fresh('صف') }) + staleBanners() +
      '<section class="panel main with-bulk">' + h.queues(qi, st.cq, 'data-cq') + body + '</section>';
  };
  function mineView() {
    var rows = M.myConversions.map(function (c) {
      return '<tr data-row="mycase:' + c.id + '" tabindex="-1"><td><span class="mono case-id">' + c.id + '</span></td><td>' + who(c.customer, phoneCell(c.phone)) + '</td><td>' + esc(c.option) + '</td><td>' + money(c.remaining) + '</td><td class="muted">' + esc(c.stage) + '</td><td><span class="next due">' + ic('clock') + esc(c.due) + '</span></td>' +
        '<td class="col-actions"><button type="button" class="btn btn-soft btn-sm" data-act="open-mycase:' + c.id + '">ثبت نتیجه تماس</button></td></tr>';
    }).join('');
    return '<div class="note info inset">' + ic('user') + '<span>کار شخصی شما؛ جدا از مدیریت صف تیم. فقط پرونده‌هایی که با «تخصیص به خودم» برداشته‌اید.</span></div>' +
      '<div class="tbl-wrap"><table class="tbl" aria-label="تبدیل‌های من"><thead><tr><th>پرونده</th><th>مشتری</th><th>گزینه</th><th>مانده</th><th>مرحله</th><th>سررسید</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div>';
  }

  /* ----- Invoices & Exceptions ----- */
  var IQ = [
    { id: 'action', label: 'نیازمند اقدام', icon: 'alert', tone: 'red', key: '1', f: function (i) { return i.bucket === 'action'; } },
    { id: 'review', label: 'در بررسی مالی', icon: 'hourglass', tone: 'orange', key: '2', f: function (i) { return i.review === 'pending'; } },
    { id: 'pre', label: 'پیش‌فاکتور', icon: 'file', tone: 'violet', key: '3', f: function (i) { return i.inv === 'pre'; } },
    { id: 'staged', label: 'مرحله‌ای ناتمام', icon: 'layers', tone: 'teal', key: '4', f: function (i) { return i.inv === 'staged' && i.paid < i.total; } },
    { id: 'done', label: 'تکمیل‌شده', icon: 'checkCircle', tone: 'green', key: '5', f: function (i) { return i.inv === 'completed'; } },
    { id: 'all', label: 'همه', tone: 'neutral', key: '0', f: function () { return true; } }
  ];
  function invPrimary(i) {
    if (i.next && i.next.who === 'seller' && i.review === 'rejected') return { label: 'بارگذاری فیش اصلاح‌شده', icon: 'upload' };
    if (i.next && i.next.who === 'seller' && i.inv === 'pre') return { label: 'ارسال مجدد لینک پرداخت', icon: 'send' };
    return null;
  }
  V.inv = function () {
    var I = M.invoices, q = IQ.filter(function (x) { return x.id === st.iq; })[0] || IQ[0], list = I.filter(q.f);
    var remaining = I.reduce(function (s, i) { return s + (i.inv === 'cancelled' ? 0 : i.total - i.paid); }, 0);
    var kpis = [
      ({ label: 'نیازمند اقدام', value: fa(I.filter(IQ[0].f).length), tone: 'bad', color: 'red', keep: true, filter: 'iq:action', meaning: 'اقدام بعدی با فروشنده یا تطبیق لازم است', basis: 'وضعیت در لحظه' }),
      ({ label: 'در انتظار بررسی مالی', value: fa(I.filter(IQ[1].f).length), color: 'orange', filter: 'iq:review', meaning: 'رسید ثبت شده و منتظر واحد مالی است — ثبت رسید به معنی تأیید پرداخت نیست', basis: 'وضعیت در لحظه' }),
      ({ label: 'پیش‌فاکتور باز', value: fa(I.filter(IQ[2].f).length), color: 'violet', filter: 'iq:pre', meaning: 'پیش‌فاکتورهای فعلی بدون تکمیل یا لغو', basis: 'وضعیت در لحظه' }),
      finKpi({ label: 'فروش تکمیل‌شده (مهر)', value: fa(I.filter(IQ[4].f).length), color: 'green', meaning: 'پرداخت کامل و تأییدشده — نه صرف صدور', basis: 'رویداد تکمیل از ۱ مهر تا امروز' }),
      finKpi({ label: 'مانده قابل وصول', value: '<span class="num">' + num(remaining) + '</span><small>تومان</small>', color: 'teal', meaning: 'جمع مانده فاکتورهای لغونشده', basis: 'وضعیت در لحظه' })
    ];
    var qi = IQ.map(function (x) { return { id: x.id, label: x.label, icon: x.icon, tone: x.tone, key: x.key, n: I.filter(x.f).length }; });
    var rows = list.map(function (i) {
      var s = INVS[i.inv], rv = REV[i.review], p = invPrimary(i), pct = i.total ? Math.round(i.paid / i.total * 100) : 0;
      return '<tr data-row="inv:' + i.code + '" tabindex="-1"><td><span class="mono case-id">' + fa(i.code) + '</span><div class="cell-sub">' + esc(seller(i.seller).name) + '</div></td>' +
        '<td>' + who(i.customer, phoneCell(i.phone)) + '</td>' +
        '<td><div class="prog"><div class="line"><span>' + money(i.total) + '</span><span>مانده ' + (i.total - i.paid > 0 && i.inv !== 'cancelled' ? '<span class="num">' + num(i.total - i.paid) + '</span>' : '—') + '</span></div><div class="bar' + (i.inv === 'mismatch' ? ' unsure' : '') + '"><i style="width:' + pct + '%"></i></div></div></td>' +
        '<td>' + pill(s.label, s.tone, s.icon) + '</td>' +
        '<td class="col-opt"><div class="facet-line"><span>مرحله ' + fa(i.stage) + ' از ' + fa(i.stages) + '، ' + esc(STG[i.stg]) + '</span><span class="cell-sub">مدرک: ' + esc(EVID[i.evidence]) + '</span></div></td>' +
        '<td>' + (rv ? pill(rv.label, rv.tone, rv.icon) : '<span class="muted">—</span>') + '</td>' +
        '<td>' + nextActor(i.next, i) + '</td>' +
        '<td class="col-actions"><div class="row-actions">' + (p ? '<button type="button" class="btn btn-soft btn-sm" data-act="inv-primary:' + i.code + '">' + ic(p.icon) + esc(p.label) + '</button>' : '<button type="button" class="btn btn-sm" data-act="open-inv:' + i.code + '">جزئیات</button>') +
        '<button type="button" class="btn btn-ghost btn-icon btn-sm tip" data-tip="بیشتر" data-act="inv-more:' + i.code + '" aria-label="اقدام‌های بیشتر فاکتور ' + i.code + '">' + ic('more') + '</button></div></td></tr>';
    }).join('');
    var cards = list.map(function (i) {
      var s = INVS[i.inv], rv = REV[i.review], p = invPrimary(i);
      return '<div class="lcard" data-row="inv:' + i.code + '"><div class="top">' + who(i.customer, 'فاکتور <span class="mono">' + fa(i.code) + '</span> · ' + esc(seller(i.seller).name)) + pill(s.label, s.tone, s.icon) + '</div>' +
        '<div class="meta"><span>کل ' + money(i.total) + '</span><span>مانده ' + money(Math.max(0, i.total - i.paid)) + '</span><span>مرحله: ' + esc(STG[i.stg]) + '</span>' + (rv ? pill(rv.label, rv.tone, rv.icon) : '') + '</div>' +
        '<div class="meta">' + nextActor(i.next, i) + '</div>' +
        '<div class="acts"><button type="button" class="btn" data-act="open-inv:' + i.code + '">جزئیات</button>' + (p ? '<button type="button" class="btn btn-soft" data-act="inv-primary:' + i.code + '">' + esc(p.label) + '</button>' : '<button type="button" class="btn" data-act="inv-more:' + i.code + '">' + ic('more') + 'بیشتر</button>') + '</div></div>';
    }).join('');
    return C.h.pageHead({ title: 'فاکتورها و استثناها', sub: 'فاکتورهای فروشندگان مستقیم شما · تأیید یا رد پرداخت فقط در اختیار واحد مالی است', kpis: kpis, fresh: fresh('فاکتورها') }) + staleBanners() +
      '<section class="panel main has-cards">' + h.queues(qi, st.iq, 'data-iq') +
      h.toolbar('کد فاکتور، مشتری یا موبایل', [{ label: 'فروشنده', icon: 'filter' }, { label: 'بازه تاریخ', icon: 'clock' }, { label: 'روش پرداخت', icon: 'filter' }], '<span class="muted desk">' + fa(list.length) + ' فاکتور</span>') +
      (list.length ? '<div class="tbl-wrap"><table class="tbl" aria-label="فاکتورهای تیم"><thead><tr><th>کد / فروشنده</th><th>مشتری</th><th>مبلغ و پرداخت</th><th>فاکتور</th><th class="col-opt">مرحله و مدرک</th><th>بررسی مالی ' + hint('ثبت رسید به معنی تأیید پرداخت نیست؛ تأیید فقط توسط واحد مالی است.', true) + '</th><th>اقدام بعدی · مسئول</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div><div class="card-list">' + cards + '</div>' + h.tfoot(list.length, list.length)
        : '<div class="empty">' + h.stateBlock('empty', 'فاکتوری در این صف نیست', 'صف دیگری را انتخاب کنید.', '<button type="button" class="btn btn-soft" data-iq="all">همه فاکتورها</button>') + '</div>') + '</section>';
  };

  /* ----- Customer History (shared customer context) ----- */
  function customers() {
    var map = {};
    M.invoices.forEach(function (i) { var c = map[i.phone] = map[i.phone] || { phone: i.phone, name: i.customer, cases: [], invs: [] }; c.invs.push(i); });
    M.held.forEach(function (x) { var c = map[x.phone] = map[x.phone] || { phone: x.phone, name: 'نام ثبت نشده', cases: [], invs: [] }; c.cases.push(x); });
    return Object.keys(map).map(function (k) { return map[k]; });
  }
  V.cust = function () {
    var list = customers();
    var rows = list.map(function (c) {
      var lastInv = c.invs[0], custody = c.cases[0] ? seller(c.cases[0].seller).name : lastInv ? seller(lastInv.seller).name : '—';
      return '<tr data-row="cust:' + c.phone + '" tabindex="-1"><td>' + who(c.name, phoneCell(c.phone)) + '</td><td class="n">' + fa(c.cases.length) + '</td><td>' + esc(custody) + '</td>' +
        '<td>' + (lastInv ? '<span class="mono">' + fa(lastInv.code) + '</span> ' + pill(INVS[lastInv.inv].label, INVS[lastInv.inv].tone, INVS[lastInv.inv].icon) : '<span class="muted">ندارد</span>') + '</td>' +
        '<td class="col-opt muted">' + esc((M.customerEvents[c.phone] || [['—', '—']])[0][1]) + '</td><td class="col-actions"><button type="button" class="btn btn-sm" data-act="open-cust:' + c.phone + '">مشاهده</button></td></tr>';
    }).join('');
    return C.h.pageHead({ title: 'سوابق مشتری', sub: 'فقط مشتریانی که پرونده یا فاکتورشان در محدوده تیم شماست' }) +
      '<section class="panel main">' + h.toolbar('نام، موبایل یا کد فاکتور مشتری', [{ label: 'فروشنده', icon: 'filter' }, { label: 'نوع رویداد', icon: 'activity' }], '<span class="scope-badge">' + ic('shield') + 'محدوده تیم شما</span>') +
      '<div class="tbl-wrap"><table class="tbl" aria-label="مشتریان تیم"><thead><tr><th>مشتری</th><th class="n">پرونده</th><th>نزد</th><th>آخرین فاکتور</th><th class="col-opt">آخرین فعالیت</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div>' + h.tfoot(list.length, list.length) + '</section>';
  };

  /* ----- HR Requests ----- */
  function chain(stepsArr) {
    return '<ol class="chain">' + stepsArr.map(function (s) { var t = { done: 'check', cur: 'hourglass', rejected: 'xCircle', skip: 'dashed' }[s[1]]; return '<li class="c-' + s[1] + '"><span class="c-dot">' + ic(t) + '</span><span class="c-t"><b>' + esc(s[0]) + '</b><span>' + esc(s[2]) + '</span></span></li>'; }).join('') + '</ol>';
  }
  V.hr = function () {
    var f = st.hrForm || (st.hrForm = { seller: '', type: 'transfer', dest: '', reason: '' }), E = st.hrErr || {};
    var err = function (k) { return E[k] ? '<div class="field-err" id="hr-e-' + k + '">' + ic('alert') + '<span>' + esc(E[k]) + '</span></div>' : ''; };
    var inv = function (k) { return E[k] ? ' aria-invalid="true" aria-describedby="hr-e-' + k + '"' : ''; };
    var form = '<section class="panel"><div class="sec-h"><h3>درخواست جدید</h3><span class="aside">فقط فروشندگان مستقیم</span></div><div class="form-grid">' +
      '<label class="lbl">فروشنده <span class="req">*</span><select class="input" data-hr="seller"' + inv('seller') + '><option value="">انتخاب فروشنده</option>' + M.sellers.map(function (s) { return '<option value="' + s.id + '"' + (String(f.seller) === String(s.id) ? ' selected' : '') + '>' + esc(s.name) + (s.active ? '' : ' (غیرفعال)') + '</option>'; }).join('') + '</select>' + err('seller') + '</label>' +
      '<div class="lbl">نوع درخواست<span class="seg" role="group" aria-label="نوع درخواست"><button type="button" data-hrtype="transfer" aria-pressed="' + (f.type === 'transfer') + '">جابجایی به سرپرست دیگر</button><button type="button" data-hrtype="termination" aria-pressed="' + (f.type === 'termination') + '">قطع همکاری</button></span></div>' +
      (f.type === 'transfer' ? '<label class="lbl">سرپرست مقصد <span class="req">*</span><select class="input" data-hr="dest"' + inv('dest') + '><option value="">انتخاب سرپرست</option><option' + (f.dest === 'رضا مختاری' ? ' selected' : '') + '>رضا مختاری</option><option' + (f.dest === 'لیلا فرهادی' ? ' selected' : '') + '>لیلا فرهادی</option></select><span class="muted" style="font-size:var(--t-micro)">خودتان به‌عنوان مقصد قابل انتخاب نیستید.</span>' + err('dest') + '</label>'
        : '<div class="note warn">' + ic('alert') + '<span>پیش از اعمال، منابع انسانی اثر این درخواست را بر پرونده‌ها، فاکتورها و کارهای باز فروشنده بررسی می‌کند. ثبت درخواست به معنی قطع همکاری نیست.</span></div>') +
      '<label class="lbl">دلیل درخواست <span class="req">*</span><textarea class="input" data-hr="reason" placeholder="دلیل جابجایی یا قطع همکاری"' + inv('reason') + '>' + esc(f.reason) + '</textarea>' + err('reason') + '</label>' +
      '<div class="form-acts"><button type="button" class="btn btn-primary" data-act="hr-submit">ثبت درخواست</button><span class="muted" style="font-size:var(--t-meta)">مسیر: بررسی مرحله‌ای ← اعمال نهایی توسط منابع انسانی</span></div></div></section>';
    var list = '<section class="panel"><div class="sec-h"><h3>درخواست‌های من</h3><span class="aside">' + fa(M.hr.length) + ' درخواست</span></div><div class="hr-list">' + M.hr.map(function (r) {
      var s = HRS[r.state], sel = seller(r.seller);
      return '<button type="button" class="hr-card" data-act="open-hr:' + r.id + '"><div class="hr-top"><b>#' + fa(r.id) + ' · ' + (r.type === 'transfer' ? 'جابجایی' : 'قطع همکاری') + ' · ' + esc(sel.name) + (r.dest ? ' ← ' + esc(r.dest) : '') + '</b>' + pill(s.label, s.tone, s.icon) + '</div>' + chain(r.steps) + '</button>';
    }).join('') + '</div></section>';
    return C.h.pageHead({ title: 'درخواست‌های HR', sub: 'درخواست ثبت می‌شود؛ اعمال واقعی فقط پس از تأیید نهایی منابع انسانی انجام می‌شود' }) + '<div class="form-card">' + form + list + '</div>';
  };

  /* ----- Reports (shared report pattern, role scope) ----- */
  V.rep = function () {
    var ctrl = '<div class="rep-ctrl"><div class="seg" role="group" aria-label="مبنای زمانی"><button type="button" data-rbasis="event" aria-pressed="' + (st.rbasis === 'event') + '">رویداد در بازه</button><button type="button" data-rbasis="now" aria-pressed="' + (st.rbasis === 'now') + '">وضعیت در لحظه</button></div>' +
      (st.rbasis === 'event' ? '<label class="lbl inline">از<input class="input" value="۱۴۰۵/۰۷/۰۱"></label><label class="lbl inline">تا<input class="input" value="۱۴۰۵/۰۷/۱۰"></label>' : '<span class="muted">وضعیت فعلی هنگام اجرای گزارش</span>') +
      '<label class="lbl inline">محدوده<select class="input"><option>تیم من (مستقیم)</option><option>فقط خودم</option>' + M.sellers.map(function (s) { return '<option>' + esc(s.name) + '</option>'; }).join('') + '</select></label></div>';
    var body;
    if (st.report) {
      var r = M.reports.filter(function (x) { return x.id === st.report; })[0];
      var rows = M.invoices.slice(0, 6).map(function (i) { return '<tr><td><span class="mono">' + fa(i.code) + '</span></td><td>' + esc(i.customer) + '</td><td>' + esc(seller(i.seller).name) + '</td><td>' + money(i.total) + '</td><td>' + pill(INVS[i.inv].label, INVS[i.inv].tone, INVS[i.inv].icon) + '</td><td class="muted">' + esc(i.issued) + '</td></tr>'; }).join('');
      body = '<section class="panel main"><div class="rep-head"><button type="button" class="btn btn-ghost btn-sm" data-act="close-report">' + ic('arrowL') + 'همه گزارش‌ها</button><h2>' + esc(r.name) + '</h2><span class="grow"></span>' + h.freshness({ text: 'محاسبه‌شده: امروز ۱۰:۴۲', refresh: true }) + '<button type="button" class="btn btn-sm" data-demo="نمایشی: فایل CSV ساخته نشد">' + ic('download') + 'خروجی CSV</button></div>' +
        '<div class="rep-meta"><span class="chip-m">سطح: ' + esc(r.grain) + '</span><span class="chip-m">مبنا: ' + esc(st.rbasis === 'now' ? 'وضعیت در لحظه' : r.basis) + '</span><span class="chip-m">محدوده: تیم من (مستقیم)</span><span class="chip-m">واحد پول: تومان</span></div>' +
        '<div class="kpis rep-kpis">' + h.kpi({ label: 'تعداد', value: fa(6), color: 'neutral', meaning: 'تعداد ' + r.grain + ' منحصربه‌فرد؛ صفحه‌بندی فقط ردیف‌ها را محدود می‌کند' }) + h.kpi({ label: 'جمع مبلغ', value: '<span class="num">' + num(M.invoices.slice(0, 6).reduce(function (a, i) { return a + i.total; }, 0)) + '</span><small>تومان</small>', color: 'teal', meaning: 'جمع همان ردیف‌های گزارش با همان تعریف' }) + '</div>' +
        '<div class="tbl-wrap"><table class="tbl no-cursor" aria-label="' + esc(r.name) + '"><thead><tr><th>کد</th><th>مشتری</th><th>فروشنده</th><th>مبلغ</th><th>وضعیت فاکتور</th><th>تاریخ صدور</th></tr></thead><tbody>' + rows + '</tbody></table></div>' + h.tfoot(6, 6) + '</section>';
    } else {
      body = '<div class="rep-grid">' + M.reports.map(function (r) { return '<button type="button" class="rep-card" data-act="open-report:' + r.id + '"><span class="rep-ico">' + ic(r.icon) + '</span><b>' + esc(r.name) + '</b><span class="rep-tags"><span>سطح: ' + esc(r.grain) + '</span><span>مبنا: ' + esc(r.basis) + '</span></span><span class="rep-go">مشاهده آمار و ریزداده ' + ic('arrowL') + '</span></button>'; }).join('') + '</div>';
    }
    return C.h.pageHead({ title: 'گزارش‌ها', sub: 'گزارش‌های مشترک CRM با محدوده تیم شما · هر گزارش سطح شمارش و مبنای زمانی خودش را دارد' }) + '<section class="panel rep-ctrl-panel">' + ctrl + '</section>' + body;
  };

  /* ----- Own wallet (shared personal wallet pattern) ----- */
  V.wallet = function () {
    var W = M.wallet;
    var rows = W.tx.map(function (t) { return '<tr><td class="muted"><span class="num">' + esc(t.date) + '</span></td><td>' + pill('بستانکار', 'green', 'plus') + '</td><td><span class="tag' + (t.channel === 'آنلاین' ? ' teal' : '') + '">' + esc(t.channel) + '</span></td><td>' + esc(t.desc) + '<div class="muted" style="font-size:var(--t-micro)">' + esc(t.rule) + '</div></td><td style="text-align:left"><span class="credit">+<span class="num">' + num(t.amount) + '</span></span> <small class="muted">تومان</small></td></tr>'; }).join('');
    return C.h.pageHead({ title: 'کیف پول من', sub: 'شخصی — فقط حق‌الزحمه شما؛ درآمد یا فروش تیم اینجا نیست' }) +
      '<div class="wallet-sum"><div class="wcard hero"><span>موجودی قابل تسویه</span><b><span class="num">' + num(W.balance) + '</span><small>تومان</small></b><div class="foot"><span>آخرین واریز: ۱۴۰۵/۰۶/۱۸</span></div></div>' +
      '<div class="wcard"><span>کل بستانکاری</span><b><span class="num">' + num(W.credit) + '</span><small>تومان</small></b><div class="foot"><span>' + fa(W.tx.length) + ' تراکنش</span></div></div>' +
      '<div class="wcard"><span>کل برداشت / تسویه</span><b><span class="num">' + num(W.debit) + '</span><small>تومان</small></b><div class="foot"><span>تسویه‌ای ثبت نشده</span></div></div></div>' +
      '<section class="panel main"><div class="toolbar"><div class="seg" role="group" aria-label="نوع پرداخت"><button type="button" aria-pressed="true">همه</button><button type="button" aria-pressed="false" data-demo="نمایشی: فیلتر آنلاین">پرداخت آنلاین</button><button type="button" aria-pressed="false" data-demo="نمایشی: فیلتر کارت‌به‌کارت">کارت‌به‌کارت</button></div><span class="grow"></span><button type="button" class="chip-btn" aria-label="بازه تاریخ">' + ic('clock') + '<span>بازه تاریخ</span></button></div>' +
      '<div class="tbl-wrap"><table class="tbl no-cursor"><thead><tr><th>تاریخ</th><th>نوع</th><th>کانال</th><th>شرح</th><th style="text-align:left">مبلغ</th></tr></thead><tbody>' + rows + '</tbody></table></div>' + h.tfoot(W.tx.length, W.tx.length) + '</section>';
  };

  /* ================= Drawers ================= */
  var top = function (title, extra) { return '<div class="sheet-grip" aria-hidden="true"></div><div class="dr-top"><span>' + esc(title) + '</span><span class="grow"></span>' + (extra || '') + '<button type="button" class="btn btn-ghost btn-icon btn-sm tip" data-tip="بستن (Esc)" data-close aria-label="بستن">' + ic('x') + '</button></div>'; };
  var foot = function (primary, secondary, hintTxt) { return '<div class="dr-foot">' + (primary || '') + (secondary || '<button type="button" class="btn btn-lg btn-ghost" data-close>بستن</button>') + '<span class="grow"></span>' + (hintTxt ? '<span class="hint">' + hintTxt + '</span>' : '') + '</div>'; };
  var tl = function (items) { return '<ul class="timeline">' + items.map(function (x) { return '<li>' + esc(x[0]) + '<span>' + esc(x[1]) + '</span></li>'; }).join('') + '</ul>'; };
  function own(o) {
    return '<div class="own-grid">' + [['مسئول فعلی', 'پرونده الان نزد کیست', o.custody, 'user'], ['مالک اولیه', 'نخستین دریافت‌کننده پرونده', o.original, 'history'], ['اقدام بعدی با', 'چه کسی باید کار بعدی را انجام دهد', o.next, 'arrowL'], ['مالک پورسانت', 'طبق قوانین مالی فعلی', o.commission, 'wallet']].map(function (x) {
      return '<div class="own"><span class="own-l">' + ic(x[3]) + esc(x[0]) + hint(x[1]) + '</span><b>' + esc(x[2]) + '</b></div>';
    }).join('') + '</div>';
  }
  function checks(list) { return '<ul class="elig">' + list.map(function (c) { var i = { ok: 'checkCircle', no: 'xCircle', warn: 'alert', q: 'question', info: 'info' }[c[0]]; return '<li class="e-' + c[0] + '">' + ic(i) + '<span><b>' + esc(c[1]) + '</b>' + (c[2] ? '<span>' + esc(c[2]) + '</span>' : '') + '</span></li>'; }).join('') + '</ul>'; }

  var D = {};
  D.seller = function (id) {
    var s = seller(Number(id)), cases = M.held.filter(function (c) { return c.seller === s.id; }), invs = M.invoices.filter(function (i) { return i.seller === s.id; });
    return top('فروشنده مستقیم') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">' + esc(s.name) + '</h2>' + (s.active ? pill('فعال', 'teal', 'checkCircle') : pill('غیرفعال', 'slate', 'ban')) + '</div><div class="dr-meta"><span class="phone mono phone-num">' + fa(s.phone) + '</span><span>آخرین تخصیص: ' + esc(s.lastAssign) + '</span></div>' +
      (s.active ? '' : '<div class="note warn" style="margin-top:12px">' + ic('alert') + '<span>این فروشنده ' + esc(s.inactiveNote) + ' و گیرنده تخصیص نیست. ' + fa(s.open) + ' پرونده هنوز نزد اوست؛ می‌توانید قابلیت برگشت آن‌ها را بررسی کنید.</span></div>') + '</div>' +
      '<div class="dr-body"><section class="sec primary"><div class="sec-h"><h3>بار کاری فعلی</h3><span class="aside">وضعیت در لحظه</span></div><div class="sum-grid four"><div class="sum"><span>پرونده باز</span><b>' + fa(s.open) + '</b></div><div class="sum"><span>بدون وضعیت</span><b>' + fa(s.untouched) + '</b></div><div class="sum"><span>تماس سررسیده</span><b' + (s.due ? ' style="color:var(--red-fg)"' : '') + '>' + fa(s.due) + '</b></div><div class="sum"><span>پیش‌فاکتور باز</span><b>' + fa(s.openPre) + '</b></div></div></section>' +
      '<details class="sec" open><summary><h3>پرونده‌ها در این نما</h3><span class="muted" style="margin-right:8px;font-size:var(--t-meta)">' + fa(cases.length) + '</span><span class="chev">' + ic('chev') + '</span></summary><div class="mini-list">' + (cases.length ? cases.map(function (c) { var e = RET[c.elig]; return '<div class="mini"><div class="grow"><span class="mono">' + c.id + '</span><div class="muted" style="font-size:var(--t-meta)">تماس: ' + esc(c.contact) + '</div></div>' + pill(e.label === 'قابل برگشت' ? 'قابل برگشت' : e.label, e.tone, e.icon) + '</div>'; }).join('') : '<div class="muted">پرونده‌ای در این نما نیست.</div>') + '</div></details>' +
      '<details class="sec"' + (invs.length ? ' open' : '') + '><summary><h3>فاکتورها</h3><span class="muted" style="margin-right:8px;font-size:var(--t-meta)">' + fa(invs.length) + '</span><span class="chev">' + ic('chev') + '</span></summary><div class="mini-list">' + invs.map(function (i) { return '<button type="button" class="mini as-btn" data-act="open-inv:' + i.code + '"><div class="grow"><span class="mono">' + fa(i.code) + '</span><div class="muted" style="font-size:var(--t-meta)">' + esc(i.customer) + '</div></div>' + pill(INVS[i.inv].label, INVS[i.inv].tone, INVS[i.inv].icon) + '</button>'; }).join('') + '</div></details></div>' +
      foot(s.active ? '<button type="button" class="btn btn-lg btn-primary" data-act="assign-to:' + s.id + '">' + ic('send') + 'تخصیص به این فروشنده</button>' : '<button type="button" class="btn btn-lg btn-primary" data-act="return-from:' + s.id + '">' + ic('repeat') + 'بررسی برگشت پرونده‌ها</button>', '<button type="button" class="btn btn-lg" data-act="hr-for:' + s.id + '">درخواست HR</button>');
  };
  D.case = function (id) {
    var c = M.pool.filter(function (x) { return x.id === id; })[0], e = ELIG[c.elig];
    return top('پرونده در پنل شما') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name" class="mono">' + esc(c.id) + '</h2>' + pill(e.label, e.tone, e.icon) + '</div><div class="dr-meta"><span class="phone mono phone-num">' + fa(c.phone) + '</span><span>' + esc(c.source) + '</span><span>دریافت: ' + esc(c.received) + '</span></div></div>' +
      '<div class="dr-body"><section class="sec">' + own({ custody: 'پنل شما (سارا احمدی)', original: c.source, next: 'شما — تخصیص به فروشنده', commission: 'هنوز فروشی ثبت نشده' }) + '</section>' +
      '<section class="sec"><div class="sec-h"><h3>قابلیت تخصیص</h3></div>' + (c.elig === 'ok' ? checks([['ok', 'وابستگی مالی ندارد', 'فاکتور، پیش‌فاکتور یا پرداخت متصل یافت نشد'], ['ok', 'نزد فروشنده‌ای نیست', 'مسئول فعلی: پنل شما'], ['info', 'هنگام ثبت دوباره بررسی می‌شود', 'این بررسی پرونده را رزرو نمی‌کند']]) : c.elig === 'blocked' ? checks([['no', 'وابستگی مالی دارد', c.why], ['info', 'پرونده‌های دارای فاکتور فعال با تخصیص عادی جابه‌جا نمی‌شوند', 'تا سوابق مالی و پورسانت حفظ شود']]) : checks([['q', 'وضعیت قابل تأیید نیست', c.why], ['info', 'تا تطبیق، تخصیص انجام نمی‌شود', 'مسئول تطبیق: MIS']])) + '</section>' +
      '<details class="sec" open><summary><h3>تاریخچه</h3><span class="chev">' + ic('chev') + '</span></summary>' + tl([['تحویل به پنل شما', 'سیستم توزیع MIS · ' + c.received], ['ورود شماره', c.source]]) + '</details></div>' +
      foot(c.elig === 'ok' ? '<button type="button" class="btn btn-lg btn-primary" data-act="pick-one:' + c.id + '">' + ic('check') + 'افزودن به انتخاب</button>' : '<button type="button" class="btn btn-lg" aria-disabled="true" data-tip="مسیر ارجاع رسمی هنوز تعریف نشده است (شکاف محصول)">' + ic('flag') + 'ارجاع (تعریف نشده)</button>');
  };

  function assignPlan() {
    var ids, recs;
    if (st.amode === 'select') { ids = keys(st.asel); recs = st.recip ? [seller(st.recip)] : []; }
    else { recs = keys(st.recips).map(function (k) { return seller(Number(k)); }); ids = poolOk().slice(0, Math.min(poolOk().length, st.per * recs.length)).map(function (c) { return c.id; }); }
    return { ids: ids, recs: recs };
  }
  D.assignReview = function () {
    var p = assignPlan(), conflict = st.flow === 'conflict', n = p.ids.length, per = p.recs.length ? Math.ceil(n / p.recs.length) : 0;
    var impact = p.recs.map(function (s) { var add = (st.amode === 'select' ? n : Math.min(st.per, n)) - (conflict ? 1 : 0); return '<div class="mini"><div class="grow"><b>' + esc(s.name) + '</b><div class="muted" style="font-size:var(--t-meta)">پرونده باز ' + fa(s.open) + ' ← <b>' + fa(s.open + add) + '</b> · بدون وضعیت ' + fa(s.untouched) + ' ← <b>' + fa(s.untouched + add) + '</b></div></div><span class="tag">+' + fa(add) + '</span></div>'; }).join('');
    return top('بررسی پیش از تخصیص') + '<div class="dr-head">' + steps(['انتخاب', 'گیرنده', 'بررسی اثر', 'نتیجه'], 2) + '<div class="dr-title" style="margin-top:12px"><h2 id="dr-name">تخصیص ' + fa(n) + ' پرونده</h2></div><div class="dr-meta"><span>به: ' + esc(p.recs.map(function (s) { return s.name; }).join('، ')) + '</span><span>' + (st.amode === 'select' ? 'پرونده‌های انتخاب‌شده' : fa(st.per) + ' پرونده برای هر گیرنده') + '</span></div></div>' +
      '<div class="dr-body">' + (conflict ? '<section class="sec"><div class="note conflict" role="alert">' + ic('swap') + '<span><b>وضعیت پرونده تغییر کرده است.</b> پرونده C-20843 پس از این بررسی توسط MIS از پنل شما برداشته شد. پیش از ثبت، بررسی را بازخوانی کنید؛ هیچ تغییری اعمال نشده است.</span></div></section>' : '') +
      '<section class="sec primary"><div class="sum-grid four"><div class="sum"><span>انتخاب‌شده</span><b>' + fa(n) + '</b></div><div class="sum"><span>قابل تخصیص</span><b style="color:var(--teal-fg)">' + fa(conflict ? n - 1 : n) + '</b></div><div class="sum"><span>مسدود (حذف شد)</span><b>۰</b></div><div class="sum"><span>تعارض</span><b' + (conflict ? ' style="color:var(--orange-fg)"' : '') + '>' + fa(conflict ? 1 : 0) + '</b></div></div></section>' +
      '<section class="sec"><div class="sec-h"><h3>اثر بر گیرنده</h3><span class="aside">وضعیت فعلی ← پس از ثبت</span></div><div class="mini-list">' + impact + '</div></section>' +
      '<section class="sec"><div class="sec-h"><h3>چه چیزی تغییر می‌کند</h3></div>' + checks([['info', 'مسئول فعلی: پنل شما ← ' + p.recs.map(function (s) { return s.name; }).join('، '), 'برای هر پرونده یک رویداد انتقال با نام شما و زمان ثبت می‌شود'], ['info', 'مالک اولیه و منبع: بدون تغییر', 'MIS · دسته ورودی حفظ می‌شود'], ['info', 'مالک پورسانت با این عملیات تعیین نمی‌شود', 'هنگام فروش، طبق قوانین مالی فعلی'], ['warn', 'این بررسی پیش‌نمایش است، نه رزرو', 'هنگام ثبت، وضعیت هر پرونده دوباره بررسی می‌شود و ممکن است بخشی ثبت نشود']]) + '</section>' +
      '<section class="sec"><label class="lbl">یادداشت <span class="muted">(اختیاری)</span><input class="input" placeholder="مثلاً تخصیص کمپین امروز"></label></section></div>' +
      foot(conflict ? '<button type="button" class="btn btn-lg btn-primary" data-act="refresh-review">' + ic('refresh') + 'بازخوانی بررسی</button>' : '<button type="button" class="btn btn-lg btn-primary" data-act="commit-assign">' + ic('send') + 'تخصیص ' + fa(n) + ' پرونده</button>', '<button type="button" class="btn btn-lg btn-ghost" data-close>بازگشت به انتخاب</button>', 'چیزی بدون تأیید شما ثبت نمی‌شود');
  };

  function resultModel(id) { return id === 'live' ? st.live : M.ops.filter(function (o) { return o.ref === id; })[0]; }
  D.result = function (id) {
    var o = resultModel(id); if (!o) return top('نتیجه') + '<div class="dr-body">' + h.stateBlock('error', 'نتیجه پیدا نشد', '') + '</div>';
    var s = opState(o), S = OPS[s], kind = o.kind === 'assign' ? 'تخصیص' : 'برگشت';
    var headline = { complete: kind + ' کامل انجام شد: ' + fa(o.ok) + ' از ' + fa(o.total) + ' پرونده ثبت شد.', partial: kind + ' ناقص انجام شد: ' + fa(o.ok) + ' از ' + fa(o.total) + ' پرونده ثبت شد و ' + fa(o.total - o.ok) + ' پرونده ثبت نشد.', failed: kind + ' انجام نشد: هیچ پرونده‌ای ثبت نشد.', unknown: 'نتیجه ' + kind + ' نامعلوم است: ممکن است ثبت شده باشد.' }[s];
    var bad = o.items.filter(function (x) { return x[1] !== 'ok'; }), good = o.items.filter(function (x) { return x[1] === 'ok'; });
    var itemRow = function (x) { var q = OUT[x[1]]; return '<li class="ir"><span class="mono">' + x[0] + '</span>' + (x[2] ? '<span class="ir-to">' + esc(x[2]) + '</span>' : '') + pill(q.label, q.tone, q.icon) + (x[3] ? '<span class="ir-why">' + esc(x[3]) + '</span>' : '') + '</li>'; };
    var retryN = o.items.filter(function (x) { return x[1] === 'retry'; }).length, confN = o.items.filter(function (x) { return x[1] === 'conflict'; }).length;
    var primary = o.unknown ? '<button type="button" class="btn btn-lg btn-primary" data-act="reconcile:' + id + '">' + ic('refresh') + 'تطبیق با شماره عملیات</button>'
      : retryN ? '<button type="button" class="btn btn-lg btn-primary" data-act="retry:' + id + '">' + ic('refresh') + 'تلاش مجدد فقط برای ' + fa(retryN) + ' مورد ناموفق</button>' : '';
    return top('نتیجه عملیات', '<span class="mono" style="margin-left:8px">' + esc(o.ref) + '</span>') +
      '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">نتیجه ' + kind + '</h2>' + pill(S[0], S[1], S[2]) + '</div><div class="dr-meta"><span>' + esc(o.when) + '</span><span>' + esc(o.actor) + '</span><span>مقصد: ' + esc(o.to) + '</span></div></div>' +
      '<div class="dr-body"><section class="sec primary"><p class="result-line r-' + s + '" role="status">' + ic(S[2]) + '<span>' + esc(headline) + '</span></p>' +
      '<div class="sum-grid four rs"><div class="sum s-ok"><span>ثبت شد</span><b>' + fa(o.ok) + '</b></div><div class="sum s-fail"><span>ناموفق</span><b>' + fa(o.failed) + '</b></div><div class="sum s-conf"><span>تعارض</span><b>' + fa(o.conflict) + '</b></div><div class="sum s-unk"><span>نامعلوم</span><b>' + fa(o.unknown) + '</b></div></div>' +
      (o.note ? '<div class="note warn" style="margin-top:12px">' + ic('question') + '<span>' + esc(o.note) + '</span></div>' : '') +
      (s === 'complete' ? '<div class="muted" style="margin-top:12px;font-size:var(--t-meta)">«ثبت شد» یعنی تغییر مسئول در سیستم ثبت شده است؛ تأیید دریافت توسط گیرنده جداگانه ثبت نمی‌شود.</div>' : '') + '</section>' +
      (bad.length ? '<section class="sec"><div class="sec-h"><h3>موارد نیازمند رسیدگی</h3><span class="aside">' + fa(bad.length) + ' مورد</span></div><ul class="ir-list">' + bad.map(itemRow).join('') + '</ul>' + (confN ? '<div class="muted" style="margin-top:8px;font-size:var(--t-meta)">موارد «تعارض» تکرار نمی‌شوند؛ وضعیتشان پس از بررسی تغییر کرده و باید دوباره انتخاب شوند.</div>' : '') + '</section>' : '') +
      (good.length ? '<details class="sec"' + (bad.length ? '' : ' open') + '><summary><h3>ثبت‌شده‌ها</h3><span class="muted" style="margin-right:8px;font-size:var(--t-meta)">' + fa(good.length) + '</span><span class="chev">' + ic('chev') + '</span></summary><ul class="ir-list">' + good.map(itemRow).join('') + '</ul></details>' : '') + '</div>' +
      foot(primary, '<button type="button" class="btn btn-lg btn-ghost" data-close>بستن</button>');
  };

  function retCase(id) { return M.held.filter(function (x) { return x.id === id; })[0]; }
  D.ret = function (id, keep) {
    var c = retCase(id), e = RET[c.elig], s = seller(c.seller), orig = seller(c.original);
    var commission = c.invoice && c.invoice.code !== '—' ? 'طبق فاکتور ' + fa(c.invoice.code) + ' (قوانین مالی فعلی)' : 'هنوز فروشی ثبت نشده';
    var body;
    if (c.elig === 'ok') body = checks([['ok', 'فاکتور یا پیش‌فاکتوری متصل نیست', 'بررسی‌شده: پیش‌فاکتور، فاکتور مرحله‌ای، فاکتور لغوشده' + (c.legacy ? '، اتصال‌های سرنخ قدیمی' : '')], ['ok', 'تبدیل یا فروش تکمیل‌شده‌ای ثبت نشده', ''], ['ok', 'نزد فروشنده مستقیم شماست', 'مسئول فعلی: ' + s.name], ['info', 'وضعیت تماس فعلی: ' + c.contact, 'با برگشت، سوابق تماس و تخصیص حذف نمی‌شود']]);
    else if (c.why === 'fin') body = checks([['no', 'وابستگی مالی محافظت‌شده دارد', 'متصل به فاکتور ' + fa(c.invoice.code) + ' · ' + c.invoice.label], ['info', 'چرا مسدود است؟', 'پرونده‌ای که فاکتور، پیش‌فاکتور یا پرداخت دارد با برگشت عادی آزاد نمی‌شود تا سوابق مالی و پورسانت حفظ شود.']]) + '<button type="button" class="btn btn-sm" style="margin-top:8px" data-act="open-inv:' + c.invoice.code + '">' + ic('receipt') + 'مشاهده فاکتور ' + fa(c.invoice.code) + '</button>';
    else if (c.why === 'cancel') body = checks([['no', 'فاکتور مرتبط لغو شده است', 'فاکتور ' + fa(c.invoice.code)], ['info', 'قانون آزادسازی تعیین نشده', 'برای پرونده‌های دارای فاکتور لغوشده هنوز قانون آزادسازی رسمی وجود ندارد؛ برگشت عادی ممکن نیست.']]);
    else if (c.elig === 'conflict') body = checks([['warn', 'وضعیت پرونده تغییر کرده است', c.why], ['info', 'پیش از هر اقدام بازخوانی کنید', 'اطلاعات نمایش‌داده‌شده ممکن است قدیمی باشد']]);
    else body = checks([['q', 'قابلیت برگشت قابل تأیید نیست', c.why], ['info', 'تا تطبیق، برگشت انجام نمی‌شود', 'مسئول تطبیق: MIS']]);
    var ok = c.elig === 'ok', confirmed = keep && keep.confirm;
    var primary = ok ? '<button type="button" class="btn btn-lg btn-primary" data-act="commit-return-one:' + c.id + '"' + (confirmed ? '' : ' disabled aria-disabled="true"') + '>' + ic('repeat') + 'تأیید برگشت این پرونده</button>'
      : c.elig === 'conflict' ? '<button type="button" class="btn btn-lg btn-primary" data-act="refresh-ret:' + c.id + '">' + ic('refresh') + 'بازخوانی پرونده</button>'
      : '<button type="button" class="btn btn-lg" aria-disabled="true" data-tip="مسیر ارجاع رسمی هنوز تعریف نشده است (شکاف محصول)">' + ic('flag') + 'ارجاع (تعریف نشده)</button>';
    return top('بررسی برگشت پرونده') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name" class="mono">' + esc(c.id) + '</h2>' + pill(e.label, e.tone, e.icon) + (c.legacy ? '<span class="tag legacy">سرنخ قدیمی</span>' : '') + '</div><div class="dr-meta"><span class="phone mono phone-num">' + fa(c.phone) + '</span><span>' + esc(c.source) + '</span><span>نزد فروشنده از ' + esc(c.since) + '</span></div></div>' +
      '<div class="dr-body"><section class="sec">' + own({ custody: s.name + (s.active ? '' : ' (غیرفعال)'), original: orig.name, next: s.name + ' — ' + c.contact, commission: commission }) + '</section>' +
      '<section class="sec' + (ok ? '' : ' primary') + '"><div class="sec-h"><h3>آیا این پرونده قابل برگشت است؟</h3></div>' + body + '</section>' +
      (ok ? '<section class="sec"><div class="sec-h"><h3>اثر برگشت</h3></div>' + checks([['info', 'مسئول فعلی: ' + s.name + ' ← پنل شما', 'یک رویداد برگشت با نام شما ثبت می‌شود؛ انتقال قبلی حذف نمی‌شود'], ['info', 'مالک اولیه، منبع و سوابق تماس: بدون تغییر', '']]) +
        '<label class="confirm"><input type="checkbox" class="cbx" data-confirm' + (confirmed ? ' checked' : '') + '><span>تأیید می‌کنم این پرونده از <b>' + esc(s.name) + '</b> گرفته و به پنل من برمی‌گردد.</span></label>' +
        '<div class="muted" style="margin-top:8px;font-size:var(--t-meta)">این بررسی پیش‌نمایش است؛ هنگام ثبت، قابلیت برگشت دوباره بررسی می‌شود.</div></section>' : '') +
      '<details class="sec"><summary><h3>تاریخچه</h3><span class="chev">' + ic('chev') + '</span></summary>' + tl([['تماس: ' + c.contact, s.name], ['تخصیص به ' + s.name, M.user.name + ' · ' + c.since], ['ورود پرونده', c.source]]) + '</details></div>' +
      foot(primary, null, ok ? 'چیزی بدون تأیید شما ثبت نمی‌شود' : '');
  };
  D.returnReview = function (id, keep) {
    var ids = keys(st.rsel).filter(retCase), list = ids.map(retCase), by = {};
    list.forEach(function (c) { by[c.seller] = (by[c.seller] || 0) + 1; });
    var confirmed = keep && keep.confirm;
    return top('بررسی برگشت گروهی') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">برگشت ' + fa(ids.length) + ' پرونده به پنل من</h2></div><div class="dr-meta"><span>فقط پرونده‌های قابل برگشت انتخاب شده‌اند</span></div></div>' +
      '<div class="dr-body"><section class="sec primary"><div class="sum-grid"><div class="sum"><span>انتخاب‌شده</span><b>' + fa(ids.length) + '</b></div><div class="sum"><span>قابل برگشت</span><b style="color:var(--teal-fg)">' + fa(ids.length) + '</b></div><div class="sum"><span>مسدود</span><b>۰</b></div></div></section>' +
      '<section class="sec"><div class="sec-h"><h3>از چه کسانی</h3></div><div class="mini-list">' + Object.keys(by).map(function (k) { var s = seller(Number(k)); return '<div class="mini"><div class="grow"><b>' + esc(s.name) + '</b><div class="muted" style="font-size:var(--t-meta)">پرونده باز ' + fa(s.open) + ' ← ' + fa(s.open - by[k]) + '</div></div><span class="tag">−' + fa(by[k]) + '</span></div>'; }).join('') + '</div></section>' +
      '<section class="sec">' + checks([['info', 'سوابق تماس و تخصیص حذف نمی‌شود', 'برای هر پرونده یک رویداد برگشت ثبت می‌شود'], ['warn', 'پیش‌نمایش، رزرو نیست', 'هنگام ثبت، هر پرونده دوباره بررسی می‌شود و ممکن است بخشی برنگردد']]) +
      '<label class="confirm"><input type="checkbox" class="cbx" data-confirm' + (confirmed ? ' checked' : '') + '><span>تأیید می‌کنم این ' + fa(ids.length) + ' پرونده از فروشندگان گرفته و به پنل من برمی‌گردد.</span></label></section></div>' +
      foot('<button type="button" class="btn btn-lg btn-primary" data-act="commit-return"' + (confirmed ? '' : ' disabled aria-disabled="true"') + '>' + ic('repeat') + 'برگشت ' + fa(ids.length) + ' پرونده</button>', '<button type="button" class="btn btn-lg btn-ghost" data-close>بازگشت</button>', 'چیزی بدون تأیید شما ثبت نمی‌شود');
  };

  D.conv = function (id) {
    var ids = id === 'many' ? keys(st.csel) : [id];
    return top('تخصیص پرونده آماده تبدیل') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">' + (ids.length > 1 ? 'تخصیص ' + fa(ids.length) + ' پرونده' : 'پرونده ' + esc(ids[0])) + '</h2></div><div class="dr-meta"><span>' + esc(ids.join('، ')) + '</span></div></div>' +
      '<div class="dr-body"><section class="sec"><div class="sec-h"><h3>به چه کسی؟</h3></div><div class="mini-list">' +
      '<label class="recip on"><input type="radio" class="cbx" name="conv-to" checked><span class="r-main"><b>خودم (' + esc(M.user.name) + ')</b><span>پرونده در «تبدیل‌های من» قرار می‌گیرد و اقدام بعدی با شما خواهد بود.</span></span></label>' +
      '<label class="recip off"><input type="radio" class="cbx" name="conv-to" disabled><span class="r-main"><b>تبدیل‌کننده</b><span>تبدیل‌کننده فعالی زیر این سرپرست تعریف نشده است.</span></span></label></div></section>' +
      '<section class="sec">' + checks([['info', 'این تخصیص فقط مسئول اقدام بعدی (تماس تبدیل) را تعیین می‌کند', ''], ['info', '«آماده تبدیل» به معنی فروش نهایی نیست', 'فروش پس از پرداخت کامل و تأیید مالی تکمیل می‌شود']]) + '</section></div>' +
      foot('<button type="button" class="btn btn-lg btn-primary" data-act="commit-conv:' + (id === 'many' ? 'many' : id) + '">' + ic('user') + 'تخصیص به خودم</button>');
  };
  D.mycase = function (id) {
    var c = M.myConversions.filter(function (x) { return x.id === id; })[0], o = (C.state.drawer && C.state.drawer.keep && C.state.drawer.keep.outcome) || null;
    var opts = [['no_answer', 'جواب نداده', 'amber', 'phone'], ['follow_up', 'تماس مجدد', 'blue', 'clock'], ['declined', 'انصراف', 'red', 'xCircle'], ['link', 'ادامه — ارسال لینک پرداخت', 'green', 'send']];
    return top('تبدیل من · ' + c.id) + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">' + esc(c.customer) + '</h2>' + pill('تخصیص به شما', 'violet', 'user') + '</div><div class="dr-meta"><span class="phone mono phone-num">' + fa(c.phone) + '</span><span>' + esc(c.option) + '</span></div><div class="dr-facts"><span class="fact-chip due">' + ic('clock') + 'سررسید: ' + esc(c.due) + '</span></div></div>' +
      '<div class="dr-body"><section class="sec primary"><div class="sec-h"><h3>نتیجه تماس</h3></div><div class="outcomes" style="grid-template-columns:repeat(3,1fr)">' + opts.map(function (x, i) { return '<button type="button" class="outcome' + (i === 3 ? ' wide' : '') + '" style="--c:var(--' + x[2] + '-dot)" data-act="outcome:' + x[0] + '" aria-pressed="' + (o === x[0]) + '"><b>' + ic(x[3]) + esc(x[1]) + '</b><small></small></button>'; }).join('') + '</div></section></div>' +
      foot('<button type="button" class="btn btn-lg btn-primary" data-demo="نمایشی: نتیجه تماس ذخیره نشد"' + (o ? '' : ' disabled') + '>' + (o ? 'ذخیره نتیجه تماس' : 'یک نتیجه انتخاب کنید') + '</button>');
  };
  D.rcase = function (id) { var r = M.ready.filter(function (x) { return x.id === id; })[0]; return r.status === 'ready' ? D.conv(id) : D.mycase(id); };

  function facets(i) {
    var s = INVS[i.inv], rv = REV[i.review], done = i.inv === 'completed';
    var f = [['فاکتور', pill(s.label, s.tone, s.icon)], ['مرحله پرداخت', '<b>مرحله ' + fa(i.stage) + ' از ' + fa(i.stages) + '</b><span>' + esc(STG[i.stg]) + '</span>'], ['مدرک پرداخت', '<b>' + esc(EVID[i.evidence]) + '</b>' + (i.evidence === 'receipt' ? '<span>ثبت رسید ≠ تأیید پرداخت</span>' : '')], ['بررسی مالی', rv ? pill(rv.label, rv.tone, rv.icon) : '<b class="muted">ارسال نشده</b>'], ['فروش تکمیل‌شده', done ? pill('بله', 'green', 'checkCircle') : '<b class="muted">خیر</b>'], ['اقدام بعدی · مسئول', nextActor(i.next, i)]];
    return '<div class="facets">' + f.map(function (x) { return '<div class="facet"><span class="f-l">' + x[0] + '</span><div class="f-v">' + x[1] + '</div></div>'; }).join('') + '</div>';
  }
  D.inv = function (code) {
    var i = M.invoices.filter(function (x) { return x.code === code; })[0], s = seller(i.seller), p = invPrimary(i), stagesHtml = '';
    for (var n = 1; n <= i.stages; n++) { var cls = n < i.stage || (i.paid >= i.total && i.inv !== 'mismatch') ? 'done' : n === i.stage ? 'cur' : ''; stagesHtml += '<div class="stage ' + cls + '"><span class="n">' + fa(n) + '</span><div class="grow">مرحله ' + fa(n) + '<div class="muted" style="font-size:var(--t-meta)">' + (cls === 'done' ? 'پرداخت و تأیید مالی' : cls === 'cur' ? STG[i.stg] : 'بعدی') + '</div></div>' + money(Math.round(i.total / i.stages)) + '</div>'; }
    var edit = i.paid === 0 && !i.sub.locked && i.inv === 'pre';
    var allowed = ['<button type="button" class="btn btn-sm" data-demo="نمایشی: فرم ثبت واریز و فیش باز نشد">' + ic('upload') + 'ثبت واریز و فیش</button>', '<button type="button" class="btn btn-sm" data-demo="نمایشی: لینک انتخاب ارسال نشد">' + ic('send') + 'ارسال لینک انتخاب</button>', '<button type="button" class="btn btn-sm" data-demo="نمایشی: لینک کپی نشد">' + ic('link') + 'کپی لینک پرداخت</button>'];
    if (edit) allowed.push('<button type="button" class="btn btn-sm" data-demo="نمایشی: ویرایش پیش از پرداخت">' + ic('edit') + 'ویرایش پیش از پرداخت</button>');
    var primaryBtn = p ? '<button type="button" class="btn btn-lg btn-primary" data-demo="نمایشی: «' + p.label + '» انجام نشد">' + ic(p.icon) + esc(p.label) + '</button>' : '';
    var waitTxt = i.review === 'pending' ? 'منتظر بررسی مالی — اقدامی از سمت شما لازم نیست' : i.inv === 'mismatch' ? 'منتظر تطبیق — مسئول: MIS' : '';
    return top('فاکتور') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name" class="mono">' + fa(i.code) + '</h2>' + pill(INVS[i.inv].label, INVS[i.inv].tone, INVS[i.inv].icon) + '</div><div class="dr-meta"><span style="color:var(--text)">' + esc(i.customer) + '</span><span class="mono phone-num">' + fa(i.phone) + '</span><span>فروشنده: ' + esc(s.name) + '</span><span>صدور: ' + esc(i.issued) + '</span></div></div>' +
      '<div class="dr-body"><section class="sec primary"><div class="sum-grid"><div class="sum"><span>مبلغ کل</span><b>' + money(i.total) + '</b></div><div class="sum"><span>پرداخت‌شده</span><b style="color:var(--green-fg)">' + money(i.paid) + '</b></div><div class="sum"><span>مانده</span><b>' + money(i.inv === 'cancelled' ? 0 : i.total - i.paid) + '</b></div></div>' +
      (i.reason ? '<div class="note danger" style="margin-top:12px">' + ic('alert') + '<span>دلیل رد مالی: ' + esc(i.reason) + ' · اقدام بعدی با ' + esc(s.name) + '</span></div>' : '') + (i.mismatch ? '<div class="note warn" style="margin-top:12px">' + ic('question') + '<span>' + esc(i.mismatch) + '. تا تطبیق، وضعیت «تکمیل‌شده» قطعی اعلام نمی‌شود.</span></div>' : '') + '</section>' +
      '<section class="sec"><div class="sec-h"><h3>وضعیت مالی به تفکیک</h3>' + hint('هر بخش معنای جدا دارد؛ مثلاً «رسید ثبت‌شده» یعنی مدرک بارگذاری شده، نه پرداخت تأییدشده.') + '</div>' + facets(i) + '</section>' +
      '<details class="sec" open><summary><h3>مراحل پرداخت</h3><span class="muted" style="margin-right:8px;font-size:var(--t-meta)">' + fa(i.stages) + ' مرحله</span><span class="chev">' + ic('chev') + '</span></summary><div class="stages">' + stagesHtml + '</div></details>' +
      '<details class="sec"><summary><h3>اشتراک انتخاب‌شده</h3><span class="chev">' + ic('chev') + '</span></summary><div class="mini"><div class="grow"><b>' + esc(i.sub.name) + '</b><div class="muted" style="font-size:var(--t-meta)">' + (i.sub.locked ? 'پس از شروع پرداخت، نوع و قیمت برای حفاظت مالی قفل است' : 'تا پیش از ثبت هر پرداخت قابل تغییر است') + '</div></div>' + (i.sub.locked ? pill('قفل', 'slate', 'lock') : '<button type="button" class="btn btn-sm" data-demo="نمایشی: تغییر اشتراک انجام نشد">تغییر اشتراک</button>') + '</div></details>' +
      '<details class="sec"><summary><h3>اقدام‌های مجاز شما</h3><span class="chev">' + ic('chev') + '</span></summary><div class="allowed">' + allowed.join('') + '</div><div class="note info" style="margin-top:12px">' + ic('shield') + '<span>تأیید، رد یا لغو پرداخت و ثبت پورسانت در اختیار واحد مالی است و از این پنل انجام نمی‌شود.</span></div></details>' +
      '<details class="sec"><summary><h3>سوابق مالی</h3><span class="chev">' + ic('chev') + '</span></summary>' + tl((i.review === 'rejected' ? [['رد توسط مالی: ' + i.reason, 'کارشناس مالی · ۱۴۰۵/۰۷/۰۵'], ['ثبت رسید مرحله ۱', s.name + ' · ۱۴۰۵/۰۷/۰۴']] : i.review === 'pending' ? [['ثبت رسید', s.name + ' · ۱۴۰۵/۰۷/۰۶']] : []).concat([['صدور ' + (i.inv === 'pre' ? 'پیش‌فاکتور' : 'فاکتور'), s.name + ' · ' + i.issued]])) + '</details></div>' +
      foot(primaryBtn || (waitTxt ? '<span class="wait-note">' + ic('hourglass') + esc(waitTxt) + '</span>' : ''), null);
  };
  D.cust = function (phone) {
    var c = customers().filter(function (x) { return x.phone === phone; })[0], ev = M.customerEvents[phone] || [], cs = c.cases[0];
    var invOwner = c.invs[0] ? seller(c.invs[0].seller).name : null;
    return top('سوابق مشتری') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">' + esc(c.name) + '</h2></div><div class="dr-meta"><span class="phone mono phone-num">' + fa(c.phone) + '</span><span>' + fa(c.cases.length) + ' پرونده، ' + fa(c.invs.length) + ' فاکتور</span></div></div>' +
      '<div class="dr-body"><section class="sec">' + own({ custody: cs ? seller(cs.seller).name : invOwner || '—', original: cs ? seller(cs.original).name : invOwner || '—', next: cs ? 'تماس: ' + cs.contact : c.invs[0] && c.invs[0].next ? c.invs[0].next.text : '—', commission: c.invs[0] ? 'طبق فاکتور ' + fa(c.invs[0].code) : 'هنوز فروشی ثبت نشده' }) + '</section>' +
      (c.cases.length ? '<section class="sec"><div class="sec-h"><h3>پرونده‌ها</h3></div><div class="mini-list">' + c.cases.map(function (x) { return '<div class="mini"><div class="grow"><span class="mono">' + x.id + '</span><div class="muted" style="font-size:var(--t-meta)">نزد ' + esc(seller(x.seller).name) + ' · تماس: ' + esc(x.contact) + '</div></div><span class="tag">' + esc(x.source) + '</span></div>'; }).join('') + '</div></section>' : '') +
      (c.invs.length ? '<section class="sec"><div class="sec-h"><h3>فاکتورها</h3></div><div class="mini-list">' + c.invs.map(function (i) { var rv = REV[i.review]; return '<button type="button" class="mini as-btn" data-act="open-inv:' + i.code + '"><div class="grow"><span class="mono">' + fa(i.code) + '</span><div class="muted" style="font-size:var(--t-meta)">' + esc(i.product) + '</div></div>' + money(i.total) + pill(INVS[i.inv].label, INVS[i.inv].tone, INVS[i.inv].icon) + (rv ? pill(rv.label, rv.tone, rv.icon) : '') + '</button>'; }).join('') + '</div></section>' : '') +
      '<section class="sec"><div class="sec-h"><h3>تاریخچه اخیر</h3><span class="aside">رفتار مشتری و رویدادهای فاکتور</span></div>' + (ev.length ? tl(ev) : '<div class="muted">رویدادی ثبت نشده است.</div>') + '</section>' +
      '<section class="sec"><div class="note info">' + ic('shield') + '<span>اطلاعات حساس مالی و منابع انسانی در این نما نمایش داده نمی‌شود.</span></div></section></div>' + foot('', null);
  };
  D.hr = function (id) {
    var r = M.hr.filter(function (x) { return String(x.id) === String(id); })[0], s = HRS[r.state];
    return top('درخواست HR #' + fa(r.id)) + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">' + (r.type === 'transfer' ? 'جابجایی' : 'قطع همکاری') + ' · ' + esc(seller(r.seller).name) + '</h2>' + pill(s.label, s.tone, s.icon) + '</div><div class="dr-meta">' + (r.dest ? '<span>مقصد: ' + esc(r.dest) + '</span>' : '') + '<span>درخواست‌دهنده: ' + esc(M.user.name) + '</span></div></div>' +
      '<div class="dr-body"><section class="sec primary"><div class="sec-h"><h3>مسیر تأیید</h3></div>' + chain(r.steps) + '</section><section class="sec"><div class="sec-h"><h3>دلیل</h3></div><p style="margin:0">' + esc(r.reason) + '</p></section>' +
      '<section class="sec">' + checks([['info', '«تأیید مرحله بررسی» با «اعمال نهایی» فرق دارد', 'تغییر واقعی فقط پس از اعمال نهایی توسط منابع انسانی انجام می‌شود'], r.state === 'pending_hr' ? ['info', 'اقدام فعلی با: منابع انسانی', 'از سمت شما اقدامی لازم نیست'] : ['info', 'این درخواست بسته شده است', '']]) + '</section></div>' + foot('', null);
  };
  D.attn = function (id) {
    if (id !== 'due') return '';
    var list = [['C-20188', 'نگار رحیمی', 'تماس مجدد · ۱۴۰۵/۰۷/۰۹ ۱۶:۰۰'], ['C-20195', 'نگار رحیمی', 'تماس مجدد · ۱۴۰۵/۰۷/۰۹ ۱۸:۳۰'], ['C-20311', 'مهدی زاده', 'تماس مجدد · ۱۴۰۵/۰۷/۰۹ ۱۱:۰۰']];
    return top('نیازمند توجه') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">تماس سررسیده</h2>' + pill(fa(list.length) + ' مورد', 'red', 'clock') + '</div><div class="dr-meta"><span>تماس‌های مجددی که زمانشان گذشته و نتیجه‌ای ثبت نشده</span></div></div>' +
      '<div class="dr-body"><section class="sec"><div class="mini-list">' + list.map(function (x) { return '<div class="mini"><div class="grow"><span class="mono">' + x[0] + '</span><div class="muted" style="font-size:var(--t-meta)">' + esc(x[2]) + '</div></div><span class="actor-who a-seller">' + ic('user') + esc(x[1]) + '</span></div>'; }).join('') + '</div></section>' +
      '<section class="sec">' + checks([['info', 'اقدام بعدی با فروشنده است', 'سررسید فقط بر اساس زمان تماس مجددی است که خود فروشنده ثبت کرده؛ قانون SLA جداگانه‌ای تعریف نشده است.']]) + '</section></div>' + foot('', null);
  };

  /* ================= Commit simulations (no data leaves the page) ================= */
  function runAssign(flow) {
    var p = assignPlan(), ref = 'OP-' + (2300 + M.ops.length), items = p.ids.map(function (id, n) { var to = p.recs.length ? p.recs[st.amode === 'select' ? 0 : n % p.recs.length].name : ''; return [id, 'ok', to]; });
    if (flow === 'partial' && items.length > 1) { items[items.length - 1][1] = 'conflict'; items[items.length - 1][3] = 'پیش از ثبت توسط MIS از پنل شما برداشته شد'; items[items.length - 2][1] = 'retry'; items[items.length - 2][3] = 'پاسخ سرور دریافت نشد؛ ثبت نشد و قابل تکرار است'; }
    if (flow === 'retry') items.forEach(function (x) { x[1] = 'retry'; x[3] = 'سرویس تخصیص پاسخ نداد؛ ثبت نشد'; });
    if (flow === 'unknown') items.forEach(function (x) { x[1] = 'unknown'; });
    var count = function (k) { return items.filter(function (x) { return x[1] === k; }).length; };
    var op = { ref: ref, kind: 'assign', when: 'همین الان', actor: M.user.name, to: p.recs.map(function (s) { return s.name; }).join('، '), total: items.length, ok: count('ok'), failed: count('retry'), conflict: count('conflict'), unknown: count('unknown'), items: items, note: flow === 'unknown' ? 'زمان پاسخ تمام شد؛ ممکن است ثبت شده باشد. پیش از هر تکرار، با شماره عملیات تطبیق داده می‌شود.' : null };
    applyOk(op); M.ops.unshift(op); st.asel = {}; st.recip = null; st.recips = {}; st.flow = null;
    if (C.state.drawer) C.closeDrawer(); C.render(); C.openDrawer('result', ref);
  }
  function applyOk(op) {
    op.items.forEach(function (x) {
      if (x[1] !== 'ok' || x[4]) return; x[4] = true;
      if (op.kind === 'assign') { M.pool = M.pool.filter(function (c) { return c.id !== x[0]; }); var s = M.sellers.filter(function (z) { return z.name === x[2]; })[0]; if (s) { s.open++; s.untouched++; } }
      else { var c = retCase(x[0]); if (c) { var sl = seller(c.seller); sl.open--; M.held = M.held.filter(function (z) { return z.id !== x[0]; }); M.pool.unshift({ id: c.id, phone: c.phone, source: 'برگشت از فروشنده', received: 'امروز', elig: 'ok' }); } }
    });
  }
  function runReturn(ids) {
    var ref = 'OP-' + (2300 + M.ops.length), items = ids.map(function (id) { return [id, 'ok', 'پنل من']; });
    var op = { ref: ref, kind: 'return', when: 'همین الان', actor: M.user.name, to: 'پنل من', total: items.length, ok: items.length, failed: 0, conflict: 0, unknown: 0, items: items };
    applyOk(op); M.ops.unshift(op); st.rsel = {}; if (C.state.drawer) C.closeDrawer(); C.render(); C.openDrawer('result', ref);
  }

  /* ================= Events ================= */
  function onClick(t, e) {
    var a = t.getAttribute('data-act');
    if (t.matches('[data-aq]')) { st.aq = t.getAttribute('data-aq'); C.state.cursor = 0; C.render(); return true; }
    if (t.matches('[data-cq]')) { st.cq = t.getAttribute('data-cq'); if (C.state.view !== 'conv') C.go('conv'); else { if (C.state.drawer) C.closeDrawer(); C.render(); } return true; }
    if (t.matches('[data-iq]')) { st.iq = t.getAttribute('data-iq'); if (C.state.drawer) C.closeDrawer(); C.render(); return true; }
    if (t.matches('[data-rf]')) { st.rf = t.getAttribute('data-rf'); C.render(); return true; }
    if (t.matches('[data-amode]')) { st.amode = t.getAttribute('data-amode'); C.render(); return true; }
    if (t.matches('[data-rbasis]')) { st.rbasis = t.getAttribute('data-rbasis'); C.render(); return true; }
    if (t.matches('[data-hrtype]')) { st.hrForm.type = t.getAttribute('data-hrtype'); st.hrErr = null; C.render(); return true; }
    if (t.matches('[data-kpi-filter]')) { var kf = t.getAttribute('data-kpi-filter'); if (kf.indexOf('iq:') === 0) { st.iq = kf.slice(3); C.render(); } else if (kf === 'inv:pre') { st.iq = 'pre'; C.go('inv'); } return true; }
    if (t.matches('[data-attn]')) {
      var A = attention().filter(function (x) { return x.id === t.getAttribute('data-attn'); })[0];
      if (A.act) { C.openDrawer('attn', 'due'); return true; }
      if (A.iq) st.iq = A.iq; if (A.aq) st.aq = A.aq; st.rf = A.rf || 'all'; st.rseller = A.id === 'inactive' ? 19 : null; C.go(A.go); return true;
    }
    if (!a) return false;
    var arg = a.indexOf(':') > -1 ? a.slice(a.indexOf(':') + 1) : null, verb = arg != null ? a.slice(0, a.indexOf(':')) : a;
    e.preventDefault(); e.stopPropagation();
    switch (verb) {
      case 'assign-to': C.closeMenu(); st.aq = 'assign'; st.amode = 'select'; st.recip = Number(arg); C.go('assign'); C.toast('گیرنده انتخاب شد: ' + seller(Number(arg)).name + '. پرونده‌ها را انتخاب کنید.', 'info'); return true;
      case 'return-from': C.closeMenu(); st.aq = 'return'; st.rf = 'all'; st.rseller = Number(arg); C.go('assign'); return true;
      case 'clear-rseller': st.rseller = null; C.render(); return true;
      case 'seller-more': C.openMenu(t, [{ label: 'جزئیات فروشنده', icon: 'user', act: 'open-seller:' + arg }, { label: 'بررسی برگشت پرونده‌ها', icon: 'repeat', act: 'return-from:' + arg }, { label: 'درخواست HR برای این فروشنده', icon: 'briefcase', act: 'hr-for:' + arg }]); return true;
      case 'open-seller': C.closeMenu(); C.openDrawer('seller', arg); return true;
      case 'hr-for': C.closeMenu(); st.hrForm = { seller: arg, type: 'transfer', dest: '', reason: '' }; st.hrErr = null; C.go('hr'); return true;
      case 'clear-asel': st.asel = {}; C.render(); return true;
      case 'clear-rsel': st.rsel = {}; C.render(); return true;
      case 'clear-csel': st.csel = {}; C.render(); return true;
      case 'pick-one': st.asel[arg] = true; C.closeDrawer(); C.render(); return true;
      case 'review-assign': if (t.disabled) return true; C.openDrawer('assignReview', 'x'); return true;
      case 'refresh-review': delete st.asel['C-20843']; M.pool = M.pool.filter(function (c) { return c.id !== 'C-20843'; }); st.flow = null; C.render(); C.openDrawer('assignReview', 'x'); C.toast('بررسی بازخوانی شد؛ ۱ پرونده از انتخاب حذف شد.', 'warning'); return true;
      case 'commit-assign': runAssign(st.flow); return true;
      case 'open-op': C.openDrawer('result', arg); return true;
      case 'retry': var o = resultModel(arg); o.items.forEach(function (x) { if (x[1] === 'retry') x[1] = 'ok'; }); o.ok += o.failed; o.failed = 0; applyOk(o); C.render(); C.openDrawer('result', arg); C.toast('تلاش مجدد فقط برای موارد ناموفق انجام شد (نمایشی).', 'success'); return true;
      case 'reconcile': var u = resultModel(arg); var half = Math.ceil(u.total / 2); u.items.forEach(function (x, n) { x[1] = n < half ? 'ok' : 'retry'; if (n >= half) x[3] = 'تطبیق: ثبت نشده بود؛ اکنون قابل تکرار است'; }); u.ok = half; u.failed = u.total - u.ok; u.unknown = 0; u.note = null; applyOk(u); C.render(); C.openDrawer('result', arg); C.toast('تطبیق انجام شد (نمایشی): ' + fa(u.ok) + ' مورد ثبت شده بود و ' + fa(u.failed) + ' مورد ثبت نشده بود.', 'info'); return true;
      case 'open-ret': C.openDrawer('ret', arg); return true;
      case 'refresh-ret': var rc = retCase(arg); rc.elig = 'ok'; rc.contact = 'جواب نداده (۱ از ۳)'; rc.why = null; C.render(); C.openDrawer('ret', arg); C.toast('پرونده بازخوانی شد؛ وضعیت فعلی نمایش داده می‌شود.', 'info'); return true;
      case 'review-return': C.openDrawer('returnReview', 'x'); return true;
      case 'commit-return': if (t.disabled) return true; runReturn(keys(st.rsel).filter(retCase)); return true;
      case 'commit-return-one': if (t.disabled) return true; runReturn([arg]); return true;
      case 'conv-one': C.openDrawer('conv', arg); return true;
      case 'conv-many': C.openDrawer('conv', 'many'); return true;
      case 'commit-conv': var ids = arg === 'many' ? keys(st.csel) : [arg]; ids.forEach(function (id) { var r = M.ready.filter(function (x) { return x.id === id; })[0]; r.status = 'assigned'; r.converter = 'self'; M.myConversions.push({ id: r.id, customer: r.customer, phone: r.phone, option: r.choice || 'انتخاب نکرده', remaining: r.credit * 5, stage: 'تماس انجام نشده', due: '—' }); }); st.csel = {}; st.showMine = true; C.closeDrawer(); C.render(); C.toast(fa(ids.length) + ' پرونده به شما تخصیص یافت (نمایشی) و در «تبدیل‌های من» قرار گرفت.', 'success'); return true;
      case 'open-mycase': C.openDrawer('mycase', arg); return true;
      case 'outcome': C.state.drawer.keep.outcome = arg; C.rerenderDrawer(); return true;
      case 'open-inv': C.closeMenu(); if (C.state.view !== 'inv' && !C.state.drawer) { /* keep page */ } C.openDrawer('inv', arg); return true;
      case 'inv-primary': C.openDrawer('inv', arg); return true;
      case 'inv-more':
        var inv = M.invoices.filter(function (x) { return x.code === arg; })[0], items = [{ label: 'جزئیات و سوابق مالی', icon: 'history', act: 'open-inv:' + arg }, '-', { head: 'اقدام‌های مجاز شما' }, { label: 'ثبت واریز و فیش', icon: 'upload' }, { label: 'ارسال لینک انتخاب', icon: 'send' }, { label: 'کپی لینک پرداخت', icon: 'link' }];
        if (inv.paid === 0 && !inv.sub.locked && inv.inv === 'pre') items.push({ label: 'ویرایش پیش از پرداخت', icon: 'edit' });
        items.push('-', { label: 'تأیید / رد پرداخت', icon: 'lock', disabled: true, why: 'فقط واحد مالی', badge: 'فقط مالی' });
        C.openMenu(t, items); return true;
      case 'open-cust': C.openDrawer('cust', arg); return true;
      case 'open-hr': C.openDrawer('hr', arg); return true;
      case 'hr-submit':
        var f = st.hrForm, E = {};
        if (!f.seller) E.seller = 'یک فروشنده انتخاب کنید.';
        if (f.type === 'transfer' && !f.dest) E.dest = 'سرپرست مقصد را انتخاب کنید.';
        if (!String(f.reason).trim()) E.reason = 'نوشتن دلیل برای این درخواست الزامی است.';
        st.hrErr = Object.keys(E).length ? E : null; C.render();
        if (st.hrErr) { var first = $('[aria-invalid="true"]'); if (first) first.focus(); return true; }
        C.toast('درخواست ثبت شد (نمایشی). وضعیت: در انتظار بررسی مرحله‌ای — هنوز اعمال نشده است.', 'success'); st.hrForm = null; C.render(); return true;
      case 'open-report': st.report = arg; C.render(); return true;
      case 'close-report': st.report = null; C.render(); return true;
    }
    return false;
  }
  // Selection, recipients and form fields use change/input events (labels fire clicks twice).
  document.addEventListener('change', function (e) {
    var t = e.target;
    if (t.matches('[data-sel]')) { var bag = st[t.getAttribute('data-sel')]; bag[t.getAttribute('data-id')] = t.checked; rerenderKeepFocus(t); return; }
    if (t.matches('[data-selall]')) {
      var k = t.getAttribute('data-selall'), ids = k === 'asel' ? poolOk().map(function (c) { return c.id; }) : k === 'rsel' ? heldList().filter(function (c) { return c.elig === 'ok'; }).map(function (c) { return c.id; }) : M.ready.filter(function (r) { return r.status === 'ready'; }).map(function (r) { return r.id; });
      ids.forEach(function (id) { st[k][id] = t.checked; }); rerenderKeepFocus(t); return;
    }
    if (t.matches('[data-recip]')) { var id = Number(t.getAttribute('data-recip')); if (st.amode === 'select') st.recip = id; else st.recips[id] = t.checked; rerenderKeepFocus(t); return; }
    if (t.matches('[data-confirm]')) { C.state.drawer.keep.confirm = t.checked; C.rerenderDrawer(); var c = $('#drawer [data-confirm]'); if (c) c.focus(); return; }
    if (t.matches('[data-hr]')) { st.hrForm[t.getAttribute('data-hr')] = t.value; if (st.hrErr) { delete st.hrErr[t.getAttribute('data-hr')]; } return; }
    if (t.matches('[data-per]')) { st.per = Math.max(1, Math.min(50, Number(t.value) || 1)); rerenderKeepFocus(t); }
  });
  document.addEventListener('input', function (e) { if (e.target.matches('[data-hr="reason"]')) st.hrForm.reason = e.target.value; });
  function rerenderKeepFocus(t) {
    var sel = t.hasAttribute('data-id') ? '[data-sel="' + t.getAttribute('data-sel') + '"][data-id="' + t.getAttribute('data-id') + '"]' : t.hasAttribute('data-selall') ? '[data-selall="' + t.getAttribute('data-selall') + '"]' : t.hasAttribute('data-recip') ? '[data-recip="' + t.getAttribute('data-recip') + '"]' : '[data-per]';
    var y = window.scrollY; C.render(); window.scrollTo(0, y); var n = $(sel); if (n) n.focus({ preventScroll: true });
  }

  /* ================= Palette records ================= */
  function palette(nq, match) {
    var out = [];
    if (!nq) return [{ g: 'اقدام سریع', icon: 'send', label: 'تخصیص پرونده به فروشنده', run: function () { st.aq = 'assign'; C.go('assign'); } }, { g: 'اقدام سریع', icon: 'repeat', label: 'بررسی برگشت پرونده', run: function () { st.aq = 'return'; C.go('assign'); } }, { g: 'اقدام سریع', icon: 'alert', label: 'فاکتورهای نیازمند اقدام', run: function () { st.iq = 'action'; C.go('inv'); } }];
    M.sellers.forEach(function (s) { if (match(s.name) || match(s.phone)) out.push({ g: 'فروشندگان', icon: 'user', label: s.name, meta: s.active ? 'فعال' : 'غیرفعال', run: function () { C.go('team'); C.openDrawer('seller', s.id); } }); });
    M.held.forEach(function (c) { if (match(c.id) || match(c.phone)) out.push({ g: 'پرونده‌ها', icon: 'file', label: c.id, meta: 'نزد ' + seller(c.seller).name, run: function () { st.aq = 'return'; st.rf = 'all'; C.go('assign'); C.openDrawer('ret', c.id); } }); });
    M.pool.forEach(function (c) { if (match(c.id) || match(c.phone)) out.push({ g: 'پرونده‌ها', icon: 'inbox', label: c.id, meta: 'در پنل شما', run: function () { st.aq = 'assign'; C.go('assign'); C.openDrawer('case', c.id); } }); });
    M.invoices.forEach(function (i) { if (match(i.code) || match(i.customer) || match(i.phone)) out.push({ g: 'فاکتورها', icon: 'receipt', label: 'فاکتور ' + fa(i.code), meta: i.customer + ' · ' + INVS[i.inv].label, run: function () { st.iq = 'all'; C.go('inv'); C.openDrawer('inv', i.code); } }); });
    return out;
  }

  /* ================= Boot ================= */
  C.boot({
    home: 'team', titleSuffix: 'پنل سرپرست (نمونه)', palettePlaceholder: 'نام فروشنده، شماره پرونده، کد فاکتور یا یک فرمان…',
    nav: function () {
      var A = attention().reduce(function (s, a) { return s + (a.id === 'fin' ? 0 : 1); }, 0);
      return [
        { id: 'team', label: 'نمای تیم', icon: 'users', group: 'work', mobile: 'تیم', count: function () { return A; }, alert: true },
        { id: 'assign', label: 'تخصیص و برگشت', icon: 'swap', group: 'work', mobile: 'تخصیص', count: function () { return poolOk().length; } },
        { id: 'conv', label: 'تبدیل', icon: 'repeat', group: 'work', mobile: 'تبدیل', count: function () { return M.ready.filter(function (r) { return r.status === 'ready'; }).length; } },
        { id: 'inv', label: 'فاکتورها و استثناها', icon: 'receipt', group: 'work', mobile: 'فاکتورها', count: function () { return M.invoices.filter(IQ[0].f).length; }, alert: true },
        { id: 'cust', label: 'سوابق مشتری', icon: 'activity', group: 'ctx' },
        { id: 'hr', label: 'درخواست‌های HR', icon: 'briefcase', group: 'ctx' },
        { id: 'rep', label: 'گزارش‌ها', icon: 'chart', group: 'ctx' },
        { id: 'wallet', label: 'کیف پول من', icon: 'wallet', group: 'me' }
      ];
    },
    views: V, drawers: D, wideDrawers: ['inv', 'result', 'assignReview'],
    pageTours: { team: 'sup', assign: 'assign', inv: 'inv' },
    workload: function () { var A = attention(); return '<span>' + ic('users') + '</span><span><b>' + fa(active().length) + '</b> فروشنده فعال</span><span class="sep extra"></span><span class="due extra">' + ic('inbox') + ' <b style="color:inherit">' + fa(A.filter(function (a) { return a.id !== 'fin'; }).length) + '</b> نیازمند توجه</span>'; },
    userMenuTop: function () { return [{ label: 'کیف پول من', icon: 'wallet', go: 'wallet' }]; },
    empty: {
      team: ['فروشنده مستقیمی ثبت نشده است', 'هنوز فروشنده‌ای زیر نظر شما تعریف نشده است.', ''],
      assign: ['پرونده‌ای در پنل شما نیست', 'هنوز پرونده‌ای برای تخصیص به شما تحویل نشده است.', ''],
      conv: ['پرونده آماده تبدیلی وجود ندارد', 'پرونده‌ها پس از تکمیل پرداخت و تأیید مالی فاکتور اعتبارسنجی اینجا می‌آیند.', ''],
      inv: ['فاکتوری در این صف نیست', 'صف دیگری را انتخاب کنید.', '<button type="button" class="btn btn-soft" data-iq="all">همه فاکتورها</button>'],
      cust: ['مشتری‌ای در محدوده تیم نیست', '', ''], wallet: ['تراکنشی ثبت نشده است', '', ''], rep: ['گزارشی در دسترس نیست', '', '']
    },
    onClick: onClick, palette: palette,
    init: function (state, qs) { if (qs.get('aq')) st.aq = qs.get('aq'); if (qs.get('cq')) st.cq = qs.get('cq'); if (qs.get('iq')) st.iq = qs.get('iq'); if (qs.get('report')) st.report = qs.get('report'); if (qs.get('mine') === '0') st.showMine = false; },
    afterBoot: function (qs) { setTimeout(function () { bootLinks(qs); }, 0); }
  });
  function bootLinks(qs) { if (qs.get('flow')) SUPAPI.flow(qs.get('flow')); if (qs.get('sel')) SUPAPI.preselect(); if (qs.get('hrerr')) { st.hrForm = { seller: '', type: 'transfer', dest: '', reason: '' }; var b = $('[data-act="hr-submit"]'); if (b) b.click(); } }

  // Small API for tours / demo panel / deep links (prototype only).
  var SUPAPI = window.SUPAPI = {
    st: st,
    preselect: function () { st.aq = 'assign'; st.amode = 'select'; st.asel = { 'C-20841': true, 'C-20842': true, 'C-20843': true }; st.recip = 11; if (C.state.view !== 'assign') C.go('assign'); else C.render(); },
    flow: function (f) {
      if (C.state.drawer) C.closeDrawer();
      if (f === 'nomine') { st.showMine = !st.showMine; st.cq = 'team'; C.go('conv'); return C.toast(st.showMine ? 'تبدیل‌های من نمایش داده می‌شود (۱ پرونده).' : 'سرپرست تبدیل شخصی ندارد؛ «تبدیل‌های من» پنهان شد.', 'info'); }
      SUPAPI.preselect(); st.flow = f;
      if (f === 'conflict') return C.openDrawer('assignReview', 'x');
      runAssign(f);
    },
    returnPick: function () { st.aq = 'return'; st.rf = 'all'; st.rseller = null; st.rsel = {}; M.held.filter(function (c) { return c.elig === 'ok'; }).slice(0, 2).forEach(function (c) { st.rsel[c.id] = true; }); if (C.state.view !== 'assign') C.go('assign'); else C.render(); }
  };
})();
