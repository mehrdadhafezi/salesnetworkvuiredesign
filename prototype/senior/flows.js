/* Senior Supervisor — role layer, part 3: distribution & return, HR chain, operation results, detail drawers.
   Senior → Supervisor is the primary distribution path. Direct-to-Seller is a CONDITIONAL exception path (SD-01): it can be previewed,
   but the commit stays disabled — no approval policy is invented. Nothing here writes anywhere. */
(function () {
  'use strict';
  var X = window.SENX, C = X.C, M = X.M, h = X.h, st = X.st, V = X.V, D = X.D;
  var esc = h.esc, fa = h.fa, num = h.num, money = h.money, ic = h.ic, pill = h.pill, hint = h.hint, cbx = h.cbx, $ = h.$;
  var seller = X.seller, team = X.team, keys = X.keys, top = X.top, foot = X.foot, own = X.own, checks = X.checks, tl = X.tl;

  /* ================= Distribution & Eligible Return ================= */
  V.dist = function () {
    var ok = X.poolOk().length, review = M.pool.length - ok, issue = M.ops.filter(function (o) { return o.failed || o.conflict || o.unknown; }).length;
    var kpis = [
      { label: 'موجودی پنل من', value: fa(M.pool.length), color: 'neutral', keep: true, meaning: 'پرونده‌های تحویل‌شده به شما که هنوز تخصیص نیافته‌اند', basis: 'وضعیت در لحظه' },
      { label: 'قابل تخصیص', value: fa(ok), color: 'teal', keep: true, meaning: 'بدون وابستگی مالی و تعارض؛ هنگام ثبت دوباره بررسی می‌شود', basis: 'وضعیت در لحظه' },
      { label: 'نیازمند بررسی', value: fa(review), tone: review ? 'warn' : '', color: 'amber', meaning: 'مسدود یا نیازمند تطبیق', basis: 'وضعیت در لحظه' },
      { label: 'گیرنده اصلی (سرپرست فعال)', value: fa(M.teams.filter(function (t) { return t.active; }).length), color: 'blue', meaning: 'مسیر اصلی: شما ← سرپرست', basis: 'ساختار فعلی' }
    ];
    var q = h.queues([
      { id: 'assign', label: 'تخصیص به سرپرست', icon: 'send', tone: 'blue', n: ok, key: '1' },
      { id: 'return', label: 'برگشت به پنل من', icon: 'repeat', tone: 'teal', n: M.held.filter(function (c) { return c.elig === 'ok'; }).length, key: '2' },
      { id: 'ops', label: 'سوابق عملیات', icon: 'history', tone: issue ? 'orange' : 'neutral', n: M.ops.length, key: '3' }
    ], st.aq, 'data-aq');
    var body = st.aq === 'return' ? returnView() : st.aq === 'ops' ? opsView() : assignView();
    return h.pageHead({ title: 'توزیع و برگشت', sub: 'پرونده‌های پنل شما ← سرپرستان (مسیر اصلی) · برگشت فقط برای انتقال‌های خودِ شما و بدون وابستگی مالی', kpis: kpis, fresh: X.freshPart('فهرست') }) + X.banners() +
      '<section class="panel main' + (st.aq === 'ops' ? '' : ' with-bulk') + '">' + q + body + '</section>';
  };

  function recipRef(k) { return k.charAt(0) === 'S' ? { kind: 'seller', s: seller(Number(k.slice(1))) } : { kind: 'sup', t: team(k) }; }
  function recipName(r) { return r.kind === 'sup' ? r.t.sup : r.s.name; }
  function plan() {
    var ids, recs;
    if (st.amode === 'select') { ids = keys(st.asel); recs = st.recip ? [recipRef(st.recip)] : []; }
    else { recs = keys(st.recips).map(recipRef); ids = X.poolOk().map(function (c) { return c.id; }).slice(0, st.per * recs.length); }
    return { ids: ids, recs: recs, exc: recs.some(function (r) { return r.kind === 'seller'; }) };
  }
  function assignView() {
    var selIds = keys(st.asel), recN = keys(st.recips).length, rec = st.recip ? recipRef(st.recip) : null;
    var stepCur = st.amode === 'select' ? (!selIds.length ? 0 : !rec ? 1 : 2) : (!recN ? 1 : 2);
    var head = '<div class="flow-head">' + X.steps(['انتخاب پرونده‌ها', 'انتخاب گیرنده', 'بررسی اثر', 'ثبت و نتیجه'], stepCur) +
      '<div class="seg" role="group" aria-label="روش تخصیص"><button type="button" data-amode="select" aria-pressed="' + (st.amode === 'select') + '">انتخاب پرونده‌ها</button><button type="button" data-amode="count" aria-pressed="' + (st.amode === 'count') + '">تعداد برای هر سرپرست</button></div></div>';
    var left;
    if (st.amode === 'select') {
      var okIds = X.poolOk().map(function (c) { return c.id; }), allOn = okIds.length && okIds.every(function (id) { return st.asel[id]; });
      var rows = M.pool.map(function (c) {
        var e = X.ELIG[c.elig], dis = c.elig !== 'ok';
        return '<tr data-row="pcase:' + c.id + '" tabindex="-1" class="' + (st.asel[c.id] ? 'picked' : '') + (dis ? ' dim' : '') + '"><td class="col-sel">' + cbx('data-sel="asel" data-id="' + c.id + '"', st.asel[c.id], 'انتخاب پرونده ' + c.id + (dis ? ' — ' + e.label : ''), dis) + '</td>' +
          '<td><span class="mono case-id">' + esc(c.id) + '</span></td><td><span class="tag">' + esc(c.source) + '</span></td><td class="col-opt muted">' + esc(c.received) + '</td>' +
          '<td>' + pill(e.label, e.tone, e.icon) + (c.why ? '<div class="cell-sub">' + esc(c.why) + '</div>' : '') + '</td></tr>';
      }).join('');
      left = h.toolbar('شناسه پرونده یا دسته', [{ label: 'منبع / دسته', icon: 'filter' }, { label: 'تاریخ دریافت', icon: 'clock' }], '<span class="muted desk">' + fa(M.pool.length) + ' پرونده در پنل شما</span>') +
        '<div class="tbl-wrap"><table class="tbl selectable" aria-label="پرونده‌های قابل تخصیص"><thead><tr><th class="col-sel" data-l="انتخاب">' + cbx('data-selall="asel"', allOn, 'انتخاب همه پرونده‌های قابل تخصیص') + '</th><th>پرونده</th><th>منبع</th><th class="col-opt">دریافت</th><th>قابلیت تخصیص ' + hint('هنگام ثبت دوباره بررسی می‌شود؛ پیش‌نمایش پرونده را رزرو نمی‌کند.', true) + '</th></tr></thead><tbody>' + rows + '</tbody></table></div>';
    } else {
      var total = st.per * recN, okN = X.poolOk().length;
      left = '<div class="count-form"><label class="lbl">تعداد پرونده برای هر سرپرست <span class="req">*</span><input class="input" type="number" min="1" max="50" value="' + st.per + '" data-per inputmode="numeric" style="max-width:160px"></label>' +
        '<label class="lbl">دسته / منبع<select class="input" style="max-width:320px"><option>همه پرونده‌های تحویل‌شده</option><option>MIS · دسته ۹۱۰۲</option><option>MIS · دسته ۹۱۰۳</option></select></label>' +
        '<div class="note info">' + ic('info') + '<span>' + (recN ? fa(st.per) + ' × ' + fa(recN) + ' سرپرست = <b>' + fa(total) + ' پرونده</b> از ' + fa(okN) + ' پرونده قابل تخصیص، به ترتیب قدیمی‌ترین دریافت.' + (total > okN ? ' <b>تعداد درخواستی بیشتر از موجودی قابل تخصیص است.</b>' : '') : 'سرپرستان گیرنده را از فهرست انتخاب کنید.') + '</span></div>' +
        '<div class="muted" style="font-size:var(--t-meta)">روش شمارشی فقط برای مسیر اصلی (سرپرست) است؛ فهرست دقیق پرونده‌ها در مرحله بررسی می‌آید.</div></div>';
    }
    var single = st.amode === 'select';
    var supList = M.teams.map(function (t) {
      var on = single ? st.recip === t.id : !!st.recips[t.id];
      return '<label class="recip' + (t.active ? '' : ' off') + (on ? ' on' : '') + '"><input type="' + (single ? 'radio' : 'checkbox') + '" name="recip" class="cbx" data-recip="' + t.id + '"' + (on ? ' checked' : '') + (t.active ? '' : ' disabled') + '>' +
        '<span class="r-main"><b>' + esc(t.sup) + '</b><span>' + (t.active ? 'سرپرست مستقیم · پرونده نزد تیم ' + fa(X.teamWorkload(t.id)) + '، فروشنده فعال ' + fa(X.sellersOf(t.id).filter(function (s) { return s.active; }).length) : 'غیرفعال — گیرنده نیست (' + esc(t.inactiveNote) + ')') + '</span></span></label>';
    }).join('');
    var excOpen = st.exc || (st.recip && st.recip.charAt(0) === 'S');
    var excSellers = M.sellers.filter(function (s) { return s.active; }).map(function (s) {
      var on = st.recip === 'S' + s.id, mid = s.rel === 'direct' ? 'بدون سرپرست میانی' : 'عبور از ' + team(s.team).sup;
      return '<label class="recip exc' + (on ? ' on' : '') + (single ? '' : ' off') + '"><input type="radio" name="recip" class="cbx" data-recip="S' + s.id + '"' + (on ? ' checked' : '') + (single ? '' : ' disabled') + '><span class="r-main"><b>' + esc(s.name) + '</b><span>' + (s.rel === 'direct' ? 'فروشنده مستقیم شما' : 'غیرمستقیم · ' + esc(X.teamName(s.team))) + ' · ' + mid + '</span></span></label>';
    }).join('');
    var excReason = st.recip && st.recip.charAt(0) === 'S' ? '<label class="lbl exc-reason">دلیل مسیر استثنایی <span class="req">*</span><textarea class="input" data-excreason placeholder="چرا این پرونده بدون عبور از سرپرست به فروشنده می‌رود؟">' + esc(st.excReason) + '</textarea><span class="muted" style="font-size:var(--t-micro)">این الزام برای ثبت استثناست؛ سیاست تأیید تعریف نشده است.</span></label>' : '';
    var aside = '<aside class="recips" aria-labelledby="rc-h"><div class="sec-h"><h3 id="rc-h">گیرنده ' + (single ? '(یک نفر)' : '(چند سرپرست)') + '</h3><span class="scope-badge">' + ic('users') + 'مسیر اصلی: سرپرست</span></div>' + supList +
      '<details class="exc-path"' + (excOpen ? ' open' : '') + ' data-exc><summary>' + ic('split') + '<span><b>مسیر استثنایی: مستقیم به فروشنده</b><span>مشروط · نیازمند سیاست (SD-01)</span></span><span class="chev">' + ic('chev') + '</span></summary>' +
      '<p class="recip-note">' + ic('alert') + 'عبور از سرپرست رفتار عادی نیست. نمایش این گزینه به معنی مجاز بودن آن نیست؛ ثبت واقعی تا تعیین سیاست غیرفعال است.</p>' + excSellers + excReason + '</details>' +
      '<p class="recip-note">' + ic('info') + 'فروشندگان غیرفعال و کاربران غیرمستقیم گیرنده پیش‌فرض نیستند. دیدن یک کاربر به معنی اجازه تخصیص به او نیست.</p></aside>';
    var bb = '';
    if (st.amode === 'select' && selIds.length) bb = X.bulkbar(fa(selIds.length) + ' پرونده انتخاب شد', rec ? 'گیرنده: ' + esc(recipName(rec)) + (rec.kind === 'seller' ? ' · مسیر استثنایی' : '') : 'یک گیرنده انتخاب کنید', '<button type="button" class="btn btn-ghost btn-sm" data-act="clear-asel">پاک کردن انتخاب</button><button type="button" class="btn btn-primary" data-act="review-assign"' + (rec ? '' : ' disabled aria-disabled="true"') + '>' + ic('eye') + 'بررسی اثر</button>');
    if (st.amode === 'count' && recN) bb = X.bulkbar(fa(st.per * recN) + ' پرونده برای ' + fa(recN) + ' سرپرست', 'ثبت فقط پس از بررسی اثر', '<button type="button" class="btn btn-primary" data-act="review-assign">' + ic('eye') + 'بررسی اثر</button>');
    return head + '<div class="split"><div class="split-main">' + left + '</div>' + aside + '</div>' + bb;
  }

  function heldList() {
    var f = function (c) { return st.rf === 'all' ? true : st.rf === 'legacy' ? !!c.legacy : st.rf === 'review' ? (c.elig === 'unknown' || c.elig === 'conflict') : c.elig === st.rf; };
    return M.held.filter(f);
  }
  function retWhy(c) {
    if (c.elig === 'ok') return 'انتقال خودِ شما · بدون فاکتور یا پرداخت متصل';
    return ({ fin: 'وابستگی مالی: ' + (c.invoice && c.invoice.label), notown: 'انتقال توسط سرپرست انجام شده؛ بازپس‌گیری با همان سرپرست است', mis: 'منبع MIS؛ برگشت با MIS است', consumed: 'مصرف‌شده در مسیر فاکتور V4: ' + (c.invoice && c.invoice.label), cancel: 'فاکتور لغوشده — قانون آزادسازی تعیین نشده (OPD-03)' })[c.why] || c.why;
  }
  function retCase(id) { return M.held.filter(function (x) { return x.id === id; })[0]; }
  function proofCell(c) {
    var p = c.proof;
    if (p.by === 'self') return '<span class="actor-who a-self">' + ic('user') + 'شما</span><div class="cell-sub">' + esc(p.op) + ' · ' + esc(p.when) + '</div>';
    if (p.by === 'sup') return '<span class="actor-who a-sup">' + ic('user') + 'سرپرست</span><div class="cell-sub">' + esc(p.op) + '</div>';
    if (p.by === 'mis') return '<span class="actor-who a-mis">' + ic('layers') + 'MIS</span><div class="cell-sub">' + esc(p.op) + '</div>';
    return '<span class="muted">ثبت نشده</span><div class="cell-sub">' + esc(p.op) + '</div>';
  }
  function returnView() {
    var list = heldList(), okIds = list.filter(function (c) { return c.elig === 'ok'; }).map(function (c) { return c.id; });
    var selIds = keys(st.rsel).filter(retCase), allOn = okIds.length && okIds.every(function (id) { return st.rsel[id]; });
    var seg = '<div class="seg" role="group" aria-label="فیلتر قابلیت برگشت">' + [['all', 'همه'], ['ok', 'قابل برگشت'], ['blocked', 'مسدود'], ['review', 'نیازمند بررسی'], ['legacy', 'سرنخ قدیمی']].map(function (f) { return '<button type="button" data-rf="' + f[0] + '" aria-pressed="' + (st.rf === f[0]) + '">' + f[1] + '</button>'; }).join('') + '</div>';
    var rows = list.map(function (c) {
      var e = X.RET[c.elig], dis = c.elig !== 'ok';
      return '<tr data-row="ret:' + c.id + '" tabindex="-1" class="' + (st.rsel[c.id] ? 'picked' : '') + '"><td class="col-sel">' + cbx('data-sel="rsel" data-id="' + c.id + '"', st.rsel[c.id], 'انتخاب پرونده ' + c.id + (dis ? ' — ' + e.label : ''), dis) + '</td>' +
        '<td><span class="mono case-id">' + esc(c.id) + '</span>' + (c.legacy ? ' <span class="tag legacy">قدیمی</span>' : '') + '<div class="cell-sub">' + esc(c.source) + '</div></td>' +
        '<td>' + X.who(X.holderName(c), c.holder.kind === 'sup' ? X.rel('sup') : X.rel('indirect')) + '</td><td class="col-opt">' + proofCell(c) + '</td>' +
        '<td>' + (c.invoice ? '<span class="mono">' + fa(c.invoice.code) + '</span><div class="cell-sub">' + esc(c.invoice.label) + '</div>' : '<span class="muted">ندارد</span>') + (c.link ? '<div class="cell-sub">' + X.link(c.link) + '</div>' : '') + '</td>' +
        '<td>' + pill(e.label, e.tone, e.icon) + '<div class="cell-sub">' + esc(retWhy(c)) + '</div></td>' +
        '<td class="col-actions"><button type="button" class="btn btn-sm" data-act="open-ret:' + c.id + '">بررسی</button></td></tr>';
    }).join('');
    var bb = selIds.length ? X.bulkbar(fa(selIds.length) + ' پرونده قابل برگشت انتخاب شد', 'برگشت گروهی مشروط است؛ هر پرونده هنگام ثبت دوباره بررسی می‌شود', '<button type="button" class="btn btn-ghost btn-sm" data-act="clear-rsel">پاک کردن انتخاب</button><button type="button" class="btn btn-primary" data-act="review-return">' + ic('eye') + 'بررسی برگشت</button>') : '';
    return '<div class="note info inset">' + ic('shield') + '<span><b>سرپرست ارشد همه چیز را بازپس نمی‌گیرد.</b> فقط انتقال‌هایی که خودتان انجام داده‌اید و هیچ فاکتور، پرداخت یا اتصال مالی ندارند قابل برگشت‌اند. انتقال سرپرستان و منبع MIS با خودشان است؛ برگشت مشروط به مجوز صریح و بررسی دوباره هنگام ثبت است.</span></div>' +
      '<div class="toolbar">' + seg + '<span class="grow"></span><span class="scope-badge view">' + ic('lock') + 'برگشت گروهی: مشروط</span><span class="muted desk">' + fa(list.length) + ' پرونده</span></div>' +
      (list.length ? '<div class="tbl-wrap"><table class="tbl selectable" aria-label="پرونده‌های پایین‌دست"><thead><tr><th class="col-sel" data-l="انتخاب">' + cbx('data-selall="rsel"', allOn, 'انتخاب همه پرونده‌های قابل برگشت') + '</th><th>پرونده و منبع</th><th>نزد (مسئول فعلی)</th><th class="col-opt">انتقال توسط ' + hint('عامل رویداد انتقال؛ مالک فعلی یا مالک اعتبار نیست.', true) + '</th><th>فاکتور / اتصال</th><th>قابلیت برگشت و دلیل</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div>'
        : h.stateBlock('noresult', 'پرونده‌ای با این فیلتر نیست', 'فیلتر دیگری انتخاب کنید.', '<button type="button" class="btn btn-soft" data-rf="all">همه پرونده‌ها</button>')) + bb;
  }
  function opState(o) { return o.unknown ? 'unknown' : (o.failed || o.conflict) && o.ok ? 'partial' : (o.failed || o.conflict) ? 'failed' : 'complete'; }
  function opsView() {
    var rows = M.ops.map(function (o) {
      var s = X.OPS[opState(o)];
      return '<tr data-row="result:' + o.ref + '" tabindex="-1"><td><span class="mono case-id">' + o.ref + '</span></td><td>' + (o.kind === 'assign' ? 'تخصیص' : 'برگشت') + '</td><td class="muted">' + esc(o.when) + '</td><td class="col-opt">' + esc(o.to) + '</td><td>' + pill(s[0], s[1], s[2]) + '<div class="cell-sub">' + (o.unknown ? fa(o.unknown) + ' مورد نامعلوم' : fa(o.ok) + ' از ' + fa(o.total) + ' ثبت شد') + '</div></td><td class="col-actions"><button type="button" class="btn btn-sm' + (opState(o) === 'complete' ? '' : ' btn-soft') + '" data-act="open-op:' + o.ref + '">' + (opState(o) === 'complete' ? 'جزئیات' : 'رسیدگی') + '</button></td></tr>';
    }).join('');
    return h.toolbar('شماره عملیات یا پرونده', [{ label: 'نوع عملیات', icon: 'filter' }, { label: 'بازه زمانی', icon: 'clock' }]) +
      '<div class="tbl-wrap"><table class="tbl" aria-label="سوابق عملیات"><thead><tr><th>عملیات</th><th>نوع</th><th>زمان</th><th class="col-opt">مقصد</th><th>نتیجه</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div>' + h.tfoot(M.ops.length, M.ops.length);
  }

  /* ---------- Distribution drawers ---------- */
  D.pcase = function (id) {
    var c = M.pool.filter(function (x) { return x.id === id; })[0], e = X.ELIG[c.elig];
    return top('پرونده در پنل شما') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name" class="mono">' + esc(c.id) + '</h2>' + pill(e.label, e.tone, e.icon) + '</div><div class="dr-meta"><span>' + esc(c.source) + '</span><span>دریافت: ' + esc(c.received) + '</span></div></div>' +
      '<div class="dr-body"><section class="sec">' + own({ custody: 'پنل شما (کامران صدری)', original: c.source, next: 'شما — تخصیص به سرپرست', event: 'سامانه توزیع MIS · ' + c.received, credit: 'هنوز فروشی ثبت نشده' }) + '</section>' +
      X.sec('قابلیت تخصیص', '', c.elig === 'ok' ? checks([['ok', 'وابستگی مالی ندارد', 'فاکتور، پیش‌فاکتور یا پرداخت متصل یافت نشد'], ['ok', 'نزد گیرنده‌ای نیست', 'مسئول فعلی: پنل شما'], ['info', 'هنگام ثبت دوباره بررسی می‌شود', 'این نمایش رزرو یا تضمین نیست']]) : checks([[c.elig === 'blocked' ? 'no' : 'q', e.label, c.why]]), c.elig === 'ok' ? '' : 'primary') + '</div>' +
      foot(c.elig === 'ok' ? '<button type="button" class="btn btn-lg btn-primary" data-act="pick-one:' + c.id + '">' + ic('check') + 'افزودن به انتخاب</button>' : '', null);
  };
  D.assignReview = function () {
    var p = plan(), conflict = st.flow === 'conflict', n = p.ids.length, names = p.recs.map(recipName).join('، ');
    var impact = p.recs.map(function (r) {
      if (r.kind === 'sup') { var add = (st.amode === 'select' ? n : Math.min(st.per, n)) - (conflict ? 1 : 0), w = X.teamWorkload(r.t.id); return '<div class="mini"><div class="grow"><b>' + esc(r.t.sup) + '</b><div class="muted" style="font-size:var(--t-meta)">پرونده نزد تیم ' + fa(w) + ' ← <b>' + fa(w + add) + '</b> · سرپرست آن را میان فروشندگان خود پخش می‌کند</div></div><span class="tag">+' + fa(add) + '</span></div>'; }
      var s = r.s; return '<div class="mini"><div class="grow"><b>' + esc(s.name) + '</b><div class="muted" style="font-size:var(--t-meta)">پرونده باز ' + fa(s.open) + ' ← <b>' + fa(s.open + n) + '</b></div></div><span class="tag">+' + fa(n) + '</span></div>';
    }).join('');
    var exc = '';
    if (p.exc) {
      var s0 = p.recs[0].s, bypass = s0.rel === 'direct' ? 'ندارد — فروشنده مستقیم شماست' : team(s0.team).sup + (team(s0.team).active ? '' : ' (غیرفعال)');
      exc = X.sec('مسیر استثنایی: مستقیم به فروشنده', 'مشروط · SD-01', '<div class="exc-grid"><div><span>رابطه</span><b>' + (s0.rel === 'direct' ? 'فروشنده مستقیم' : 'فروشنده غیرمستقیم') + '</b></div><div><span>سطح ردشده</span><b>' + esc(bypass) + '</b></div><div><span>مسئول فعلی پرونده‌ها</span><b>پنل شما</b></div><div><span>دلیل ثبت‌شده</span><b>' + (st.excReason.trim() ? esc(st.excReason) : '<span class="warn-n">وارد نشده</span>') + '</b></div></div>' +
        checks([['ok', 'قابلیت تخصیص: همه پرونده‌ها بدون وابستگی مالی', ''], ['info', 'تعارض وابستگی: ندارد', 'در صورت وجود، پیش از ثبت نمایش داده می‌شود'], ['warn', 'سیاست مجاز بودن عبور از سرپرست تعریف نشده است', 'ثبت واقعی تا تعیین سیاست غیرفعال است؛ تاریخچه و مالک اعتبار بدون تغییر می‌ماند.']]), 'primary');
    }
    var canCommit = !p.exc && !conflict && n > 0;
    var primary = conflict ? '<button type="button" class="btn btn-lg btn-primary" data-act="refresh-review">' + ic('refresh') + 'بازخوانی بررسی</button>' : p.exc ? '<button type="button" class="btn btn-lg" disabled aria-disabled="true" data-tip="ثبت مسیر استثنایی تا تعیین سیاست غیرفعال است (SD-01)">' + ic('lock') + 'ثبت غیرفعال — نیازمند سیاست</button>' : '<button type="button" class="btn btn-lg btn-primary" data-act="commit-assign"' + (canCommit ? '' : ' disabled aria-disabled="true"') + '>' + ic('send') + 'تأیید و ثبت تخصیص</button>';
    return top('بررسی پیش از تخصیص') + '<div class="dr-head">' + X.steps(['انتخاب', 'گیرنده', 'بررسی اثر', 'نتیجه'], 2) + '<div class="dr-title" style="margin-top:12px"><h2 id="dr-name">تخصیص ' + fa(n) + ' پرونده</h2></div><div class="dr-meta"><span>به: ' + esc(names) + '</span><span>' + (st.amode === 'select' ? 'پرونده‌های انتخاب‌شده' : fa(st.per) + ' پرونده برای هر سرپرست') + '</span></div></div>' +
      '<div class="dr-body">' + (conflict ? '<section class="sec"><div class="note conflict" role="alert">' + ic('swap') + '<span><b>وضعیت پرونده تغییر کرده است.</b> پرونده C-31003 پس از این بررسی توسط MIS از پنل شما برداشته شد. بازخوانی کنید؛ هیچ تغییری اعمال نشده است.</span></div></section>' : '') +
      '<section class="sec primary"><div class="sum-grid four"><div class="sum"><span>انتخاب‌شده</span><b>' + fa(n) + '</b></div><div class="sum"><span>قابل تخصیص</span><b style="color:var(--teal-fg)">' + fa(conflict ? n - 1 : n) + '</b></div><div class="sum"><span>مسدود (حذف شد)</span><b>۰</b></div><div class="sum"><span>تعارض</span><b' + (conflict ? ' style="color:var(--orange-fg)"' : '') + '>' + fa(conflict ? 1 : 0) + '</b></div></div></section>' +
      X.sec('اثر بر گیرنده', '', '<div class="mini-list">' + impact + '</div>') + exc +
      X.sec('این بررسی چه چیزی را تضمین نمی‌کند؟', '', checks([['info', 'پیش‌نمایش ≠ ثبت', 'هنگام ثبت، دارنده، گیرنده فعال، وابستگی و محدوده دوباره بررسی می‌شود'], ['info', 'مالک اولیه و مالک اعتبار تغییر نمی‌کند', 'یک رویداد انتقال با نام شما ثبت می‌شود؛ «ثبت شد» یعنی انتقال ثبت شده، نه تأیید دریافت.']])) + '</div>' +
      foot(primary, '<button type="button" class="btn btn-lg btn-ghost" data-close>بازگشت به انتخاب</button>', 'چیزی بدون تأیید شما ثبت نمی‌شود');
  };

  // Operation results: per-item outcome, partial ≠ success, unknown reconciled before any retry.
  function makeResult(kind, ids, toName, flow) {
    var items = ids.map(function (id, i) { return [id, 'ok', toName]; }), note = null;
    if (flow === 'partial' && items.length > 2) { items[items.length - 1] = [ids[ids.length - 1], 'retry', toName, 'پاسخ سرور ناقص بود؛ قابل تکرار']; items[items.length - 2] = [ids[ids.length - 2], 'conflict', toName, 'پرونده پس از بررسی تغییر کرد']; }
    else if (flow === 'retry') items = items.map(function (x) { return [x[0], 'retry', toName, 'خطای موقت شبکه؛ همان قصد قابل تکرار است']; });
    else if (flow === 'unknown') { items = items.map(function (x) { return [x[0], 'unknown', toName]; }); note = 'پاسخ تأیید نرسید؛ ممکن است ثبت شده باشد. پیش از تکرار با شماره عملیات تطبیق کنید.'; }
    var c = function (k) { return items.filter(function (x) { return x[1] === k; }).length; };
    return { ref: 'OP-3310', kind: kind, when: 'همین الان', actor: M.user.name, total: items.length, ok: c('ok'), failed: c('retry') + c('failed'), conflict: c('conflict'), unknown: c('unknown'), to: toName, items: items, note: note };
  }
  function resultModel(id) { return id === 'live' ? st.live : M.ops.filter(function (o) { return o.ref === id; })[0]; }
  X.applyOk = function (op) {
    var okIds = op.items.filter(function (x) { return x[1] === 'ok'; }).map(function (x) { return x[0]; });
    if (op.kind === 'assign') M.pool = M.pool.filter(function (c) { return okIds.indexOf(c.id) < 0; });
    else okIds.forEach(function (id) { var c = retCase(id); if (c) { M.held = M.held.filter(function (x) { return x.id !== id; }); M.pool.push({ id: id, source: 'برگشت از ' + X.holderName(c), received: 'امروز', elig: 'ok' }); } });
    st.asel = {}; st.rsel = {};
  };
  X.runAssign = function (flow) {
    var p = plan(), ids = p.ids.slice(), op = makeResult('assign', ids, p.recs.map(recipName).join('، '), flow);
    st.live = op; st.flow = null; if (op.ok) X.applyOk(op); M.ops.unshift(op);
    C.closeDrawer(); st.aq = 'ops'; C.render(); C.openDrawer('result', 'OP-3310');
  };
  X.runReturn = function (ids) {
    var op = makeResult('return', ids, 'پنل من', st.flow); st.live = op; st.flow = null; if (op.ok) X.applyOk(op); M.ops.unshift(op);
    C.closeDrawer(); st.aq = 'ops'; C.render(); C.openDrawer('result', 'OP-3310');
  };
  D.result = function (id) {
    var o = id === 'OP-3310' ? M.ops[0] : resultModel(id);
    if (!o) return top('نتیجه') + '<div class="dr-body">' + h.stateBlock('error', 'نتیجه پیدا نشد', '') + '</div>';
    var s = opState(o), S = X.OPS[s], kind = o.kind === 'assign' ? 'تخصیص' : 'برگشت';
    var headline = { complete: kind + ' کامل انجام شد: ' + fa(o.ok) + ' از ' + fa(o.total) + ' پرونده ثبت شد.', partial: kind + ' ناقص انجام شد: ' + fa(o.ok) + ' از ' + fa(o.total) + ' پرونده ثبت شد و ' + fa(o.total - o.ok) + ' پرونده ثبت نشد.', failed: kind + ' انجام نشد: هیچ پرونده‌ای ثبت نشد.', unknown: 'نتیجه ' + kind + ' نامعلوم است: ممکن است ثبت شده باشد.' }[s];
    var bad = o.items.filter(function (x) { return x[1] !== 'ok'; }), good = o.items.filter(function (x) { return x[1] === 'ok'; });
    var itemRow = function (x) { var q = X.OUT[x[1]]; return '<li class="ir"><span class="mono">' + x[0] + '</span>' + (x[2] ? '<span class="ir-to">' + esc(x[2]) + '</span>' : '') + pill(q.label, q.tone, q.icon) + (x[3] ? '<span class="ir-why">' + esc(x[3]) + '</span>' : '') + '</li>'; };
    var retryN = o.items.filter(function (x) { return x[1] === 'retry'; }).length;
    var primary = o.unknown ? '<button type="button" class="btn btn-lg btn-primary" data-act="reconcile:' + id + '">' + ic('refresh') + 'تطبیق با شماره عملیات</button>' : retryN ? '<button type="button" class="btn btn-lg btn-primary" data-act="retry:' + id + '">' + ic('refresh') + 'تلاش مجدد فقط برای ' + fa(retryN) + ' مورد ناموفق</button>' : '';
    return top('نتیجه عملیات', '<span class="mono" style="margin-left:8px">' + esc(o.ref) + '</span>') +
      '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">نتیجه ' + kind + '</h2>' + pill(S[0], S[1], S[2]) + '</div><div class="dr-meta"><span>' + esc(o.when) + '</span><span>' + esc(o.actor) + '</span><span>مقصد: ' + esc(o.to) + '</span></div></div>' +
      '<div class="dr-body"><section class="sec primary"><p class="result-line r-' + s + '" role="status">' + ic(S[2]) + '<span>' + esc(headline) + '</span></p>' +
      '<div class="sum-grid four rs"><div class="sum s-ok"><span>ثبت شد</span><b>' + fa(o.ok) + '</b></div><div class="sum s-fail"><span>ناموفق</span><b>' + fa(o.failed) + '</b></div><div class="sum s-conf"><span>تعارض</span><b>' + fa(o.conflict) + '</b></div><div class="sum s-unk"><span>نامعلوم</span><b>' + fa(o.unknown) + '</b></div></div>' +
      (o.note ? '<div class="note warn" style="margin-top:12px">' + ic('question') + '<span>' + esc(o.note) + '</span></div>' : '') + (s === 'complete' ? '<div class="muted" style="margin-top:12px;font-size:var(--t-meta)">«ثبت شد» یعنی انتقال در سیستم ثبت شده است؛ تأیید دریافت توسط گیرنده جداگانه ثبت نمی‌شود.</div>' : '') + '</section>' +
      (bad.length ? '<section class="sec"><div class="sec-h"><h3>موارد نیازمند رسیدگی</h3><span class="aside">' + fa(bad.length) + ' مورد</span></div><ul class="ir-list">' + bad.map(itemRow).join('') + '</ul></section>' : '') +
      (good.length ? '<details class="sec"' + (bad.length ? '' : ' open') + '><summary><h3>ثبت‌شده‌ها</h3><span class="muted" style="margin-right:8px;font-size:var(--t-meta)">' + fa(good.length) + '</span><span class="chev">' + ic('chev') + '</span></summary><ul class="ir-list">' + good.map(itemRow).join('') + '</ul></details>' : '') + '</div>' +
      foot(primary, '<button type="button" class="btn btn-lg btn-ghost" data-close>بستن</button>');
  };
  X.reconcile = function (id) {
    var u = resultModel(id) || M.ops[0], half = Math.ceil(u.total / 2);
    u.items.forEach(function (x, n) { x[1] = n < half ? 'ok' : 'retry'; if (n >= half) x[3] = 'تطبیق: ثبت نشده بود؛ اکنون قابل تکرار است'; });
    u.ok = half; u.failed = u.total - u.ok; u.unknown = 0; u.note = null; X.applyOk(u);
  };
  X.retryFailed = function (id) { var o = resultModel(id) || M.ops[0]; o.items.forEach(function (x) { if (x[1] === 'retry') x[1] = 'ok'; }); o.ok += o.failed; o.failed = 0; X.applyOk(o); };

  D.ret = function (id, keep) {
    var c = retCase(id), e = X.RET[c.elig], holder = X.holderName(c), p = c.proof;
    var lvl = c.holder.kind === 'sup' ? 'سرپرست' : 'فروشنده', t = c.holder.kind === 'sup' ? c.holder.team : seller(c.holder.id).team;
    var body, resp = null;
    if (c.elig === 'ok') body = checks([['ok', 'انتقال را خودِ شما انجام داده‌اید', p.op + ' · ' + p.when], ['ok', 'فاکتور، پیش‌فاکتور یا پرداخت متصل نیست', 'بررسی‌شده: فاکتور مرحله‌ای، پیگیری V4، فاکتور لغوشده و اتصال‌های قدیمی'], ['ok', 'نزد ' + lvl + ' فعال و در محدوده شماست', holder], ['info', 'مالک اولیه، منبع و سوابق تماس بدون تغییر می‌ماند', '']]);
    else if (c.why === 'fin') { body = checks([['no', 'وابستگی مالی محافظت‌شده دارد', 'فاکتور ' + fa(c.invoice.code) + ' · ' + c.invoice.label], ['info', 'چرا مسدود است؟', 'پرونده‌ای که فاکتور، پیش‌فاکتور یا پرداخت دارد با برگشت عادی آزاد نمی‌شود تا سوابق مالی و اعتبار حفظ شود.']]); resp = { who: 'finance', text: 'واحد مالی / فروشنده طبق وضعیت فاکتور' }; }
    else if (c.why === 'consumed') { body = checks([['no', 'پرونده مصرف‌شده است', 'مسیر پیگیری فاکتور V4 فعال است: فاکتور ' + fa(c.invoice.code) + ' · ' + c.invoice.label], ['info', 'برگشت عادی انجام نمی‌شود', 'حذف اتصال، حل تعارض حساب نمی‌شود؛ باید اثبات اتصال با مالی/MIS تطبیق شود.']]); resp = { who: 'finance', text: 'مالی/MIS برای تطبیق؛ سرپرست برای اقدام عملیاتی' }; }
    else if (c.why === 'notown') { body = checks([['no', 'انتقال را سرپرست انجام داده است', p.op], ['info', 'رتبه، اجازه بازپس‌گیری نمی‌دهد', 'سابقه عامل انتقال با مقام بالاتر جایگزین نمی‌شود.']]); resp = { who: 'sup', text: 'سرپرست ' + team(t).sup }; }
    else if (c.why === 'mis') { body = checks([['no', 'منبع MIS است', 'برگشت یا آزادسازی منبع در دامنه MIS است، نه سرپرست ارشد'], ['info', 'شما مالک منابع وارد‌شده نیستید', '']]); resp = { who: 'mis', text: 'MIS' }; }
    else if (c.why === 'cancel') { body = checks([['no', 'فاکتور مرتبط لغو شده است', 'فاکتور ' + fa(c.invoice.code)], ['info', 'قانون آزادسازی تعیین نشده', 'برای پرونده با فاکتور لغوشده هنوز قانون آزادسازی رسمی وجود ندارد (OPD-03).']]); resp = null; }
    else if (c.elig === 'conflict') { body = checks([['warn', 'وضعیت پرونده تغییر کرده است', c.why], ['info', 'پیش از هر اقدام بازخوانی کنید', 'اطلاعات نمایش‌داده‌شده ممکن است قدیمی باشد']]); }
    else { body = checks([['q', 'قابلیت برگشت قابل تأیید نیست', c.why], ['info', 'تا تطبیق، برگشت انجام نمی‌شود', 'حدس با شماره موبایل انجام نمی‌شود']]); resp = { who: 'mis', text: 'MIS / مالی برای تطبیق' }; }
    var ok = c.elig === 'ok', confirmed = keep && keep.confirm;
    var primary = ok ? '<button type="button" class="btn btn-lg btn-primary" data-act="commit-return-one:' + c.id + '"' + (confirmed ? '' : ' disabled aria-disabled="true"') + '>' + ic('repeat') + 'تأیید برگشت این پرونده</button>' : c.elig === 'conflict' ? '<button type="button" class="btn btn-lg btn-primary" data-act="refresh-ret:' + c.id + '">' + ic('refresh') + 'بازخوانی پرونده</button>' : '';
    return top('بررسی برگشت پرونده') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name" class="mono">' + esc(c.id) + '</h2>' + pill(e.label, e.tone, e.icon) + (c.legacy ? '<span class="tag legacy">سرنخ قدیمی</span>' : '') + '</div><div class="dr-meta"><span>' + esc(c.source) + '</span><span>نزد ' + lvl + ' ' + esc(holder) + '</span></div></div>' +
      '<div class="dr-body"><section class="sec">' + own({ custody: holder + ' (' + lvl + ')', original: c.source, next: resp ? resp.text : lvl + ' ' + holder, event: p.by === 'self' ? 'شما · ' + p.op : p.by === 'none' ? 'ثبت نشده' : p.op, credit: c.invoice ? 'طبق فاکتور ' + fa(c.invoice.code) + ' (قوانین مالی فعلی)' : 'هنوز فروشی ثبت نشده' }) + '</section>' +
      '<section class="sec' + (ok ? '' : ' primary') + '"><div class="sec-h"><h3>آیا این پرونده قابل برگشت است؟</h3></div>' + body + (resp ? '<div class="resp-line"><span>مسئول اقدام بعدی:</span>' + X.nextActor({ who: resp.who, text: resp.text }) + '</div>' : '') + '</section>' +
      (ok ? '<section class="sec"><div class="sec-h"><h3>اثر برگشت</h3></div>' + checks([['info', 'مسئول فعلی: ' + holder + ' ← پنل شما', 'یک رویداد برگشت با نام شما ثبت می‌شود؛ انتقال قبلی حذف نمی‌شود']]) +
        '<label class="confirm"><input type="checkbox" class="cbx" data-confirm' + (confirmed ? ' checked' : '') + '><span>تأیید می‌کنم این پرونده از <b>' + esc(holder) + '</b> گرفته و به پنل من برمی‌گردد.</span></label><div class="muted" style="margin-top:8px;font-size:var(--t-meta)">این بررسی پیش‌نمایش است و ثبت را تضمین نمی‌کند؛ هنگام ثبت، وابستگی مالی و دارنده دوباره بررسی می‌شود.</div></section>' : '') +
      X.details('تاریخچه', null, tl([['انتقال به ' + holder, (p.by === 'self' ? 'شما' : p.op) + ' · ' + p.when], ['ورود پرونده', c.source]])) + '</div>' + foot(primary, null, ok ? 'چیزی بدون تأیید شما ثبت نمی‌شود' : '');
  };
  D.returnReview = function (id, keep) {
    var ids = keys(st.rsel).filter(retCase), confirmed = keep && keep.confirm;
    return top('بررسی برگشت گروهی') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">برگشت ' + fa(ids.length) + ' پرونده</h2>' + pill('مشروط', 'amber', 'lock') + '</div><div class="dr-meta"><span>مقصد: پنل من</span></div></div><div class="dr-body">' +
      X.sec('پرونده‌ها', '', '<div class="mini-list">' + ids.map(function (i) { var c = retCase(i); return '<div class="mini"><div class="grow"><span class="mono">' + i + '</span><div class="muted" style="font-size:var(--t-meta)">نزد ' + esc(X.holderName(c)) + ' · ' + esc(c.proof.op) + '</div></div>' + pill('قابل برگشت', 'teal', 'checkCircle') + '</div>'; }).join('') + '</div>', 'primary') +
      '<section class="sec"><label class="confirm"><input type="checkbox" class="cbx" data-confirm' + (confirmed ? ' checked' : '') + '><span>تأیید می‌کنم این ' + fa(ids.length) + ' پرونده به پنل من برگردد.</span></label><div class="muted" style="margin-top:8px;font-size:var(--t-meta)">نتیجه به‌تفکیک پرونده ثبت می‌شود؛ ناقص یا نامعلوم با موفقیت کامل یکی نیست.</div></section></div>' +
      foot('<button type="button" class="btn btn-lg btn-primary" data-act="commit-return"' + (confirmed && ids.length ? '' : ' disabled aria-disabled="true"') + '>' + ic('repeat') + 'تأیید و ثبت برگشت</button>', null);
  };

  /* ================= HR Requests ================= */
  V.hr = function () {
    var f = st.hrForm || (st.hrForm = { seller: '', type: 'transfer', dest: '', reason: '' }), E = st.hrErr || {};
    var err = function (k) { return E[k] ? '<div class="field-err" id="hr-e-' + k + '">' + ic('alert') + '<span>' + esc(E[k]) + '</span></div>' : ''; };
    var inv = function (k) { return E[k] ? ' aria-invalid="true" aria-describedby="hr-e-' + k + '"' : ''; };
    var subj = M.sellers.filter(function (s) { return s.rel === 'indirect'; });
    var form = '<section class="panel"><div class="sec-h"><h3>درخواست جدید</h3><span class="aside">فقط فروشندگان زیرمجموعه</span></div><div class="form-grid">' +
      '<label class="lbl">فروشنده <span class="req">*</span><select class="input" data-hr="seller"' + inv('seller') + '><option value="">انتخاب فروشنده</option>' + subj.map(function (s) { return '<option value="' + s.id + '"' + (String(f.seller) === String(s.id) ? ' selected' : '') + '>' + esc(s.name) + ' — ' + esc(X.teamName(s.team)) + (s.active ? '' : ' (غیرفعال)') + '</option>'; }).join('') + '</select>' + err('seller') + '</label>' +
      '<div class="lbl">نوع درخواست<span class="seg" role="group" aria-label="نوع درخواست"><button type="button" data-hrtype="transfer" aria-pressed="' + (f.type === 'transfer') + '">جابجایی به سرپرست دیگر</button><button type="button" data-hrtype="termination" aria-pressed="' + (f.type === 'termination') + '">قطع همکاری</button></span></div>' +
      (f.type === 'transfer' ? '<label class="lbl">سرپرست مقصد <span class="req">*</span><select class="input" data-hr="dest"' + inv('dest') + '><option value="">انتخاب سرپرست</option><optgroup label="تیم‌های محدوده شما">' + M.teams.filter(function (t) { return t.active; }).map(function (t) { return '<option value="' + esc(t.sup) + '"' + (f.dest === t.sup ? ' selected' : '') + '>' + esc(t.sup) + '</option>'; }).join('') + '</optgroup><optgroup label="خارج از محدوده شما — سیاست تعریف نشده (SD-04)"><option' + (f.dest === 'مجتبی عباسی' ? ' selected' : '') + ' value="مجتبی عباسی">مجتبی عباسی</option></optgroup></select>' + (f.dest === 'مجتبی عباسی' ? '<div class="note warn" style="margin-top:8px">' + ic('alert') + '<span>مقصد خارج از محدوده شما است. قانون چنین جابجایی‌ای تعریف نشده؛ فرم این گزینه را قطعی نمی‌داند و تصمیم نهایی با منابع انسانی است.</span></div>' : '') + err('dest') + '</label>'
        : '<div class="note warn">' + ic('alert') + '<span>پیش از اعمال، منابع انسانی اثر این درخواست را بر پرونده‌ها، فاکتورها و کارهای باز فروشنده بررسی می‌کند. ثبت درخواست به معنی قطع همکاری نیست.</span></div>') +
      '<label class="lbl">دلیل درخواست <span class="req">*</span><textarea class="input" data-hr="reason" placeholder="دلیل جابجایی یا قطع همکاری"' + inv('reason') + '>' + esc(f.reason) + '</textarea>' + err('reason') + '</label>' +
      '<div class="form-acts"><button type="button" class="btn btn-primary" data-act="hr-submit">ثبت درخواست</button><span class="muted" style="font-size:var(--t-meta)">مسیر: بررسی مرحله‌ای ← … ← اعمال نهایی توسط منابع انسانی</span></div></div></section>';
    var mine = M.hr.filter(function (r) { return r.state === 'pending_review' && r.reviewer === 'self'; }).length - Object.keys(st.hrDone).length;
    var QS = [{ id: 'review', label: 'منتظر بررسی شما', icon: 'hourglass', tone: 'orange', n: Math.max(0, mine), key: '1' }, { id: 'chain', label: 'در مسیر (فقط مشاهده)', icon: 'eye', tone: 'slate', n: M.hr.filter(function (r) { return r.state === 'pending_hr'; }).length + Object.keys(st.hrDone).length, key: '2' }, { id: 'closed', label: 'بسته‌شده', icon: 'history', tone: 'neutral', n: M.hr.filter(function (r) { return r.state === 'approved' || r.state === 'rejected'; }).length, key: '3' }];
    var inQ = function (r) { var done = st.hrDone[r.id]; return st.hq === 'review' ? r.state === 'pending_review' && r.reviewer === 'self' && !done : st.hq === 'chain' ? r.state === 'pending_hr' || !!done && done !== 'rejected' : r.state === 'approved' || r.state === 'rejected' || done === 'rejected'; };
    var list = M.hr.filter(inQ);
    var cards = list.map(function (r) {
      var done = st.hrDone[r.id], s = done === 'rejected' ? X.HRS.rejected : done ? X.HRS.stepdone : X.HRS[r.state], sel = seller(r.seller);
      return '<button type="button" class="hr-card" data-act="open-hr:' + r.id + '"><div class="hr-top"><b>#' + fa(r.id) + ' · ' + (r.type === 'transfer' ? 'جابجایی' : 'قطع همکاری') + ' · ' + esc(sel.name) + (r.dest ? ' ← ' + esc(r.dest) : '') + '</b>' + pill(s.label, s.tone, s.icon) + '</div>' + X.chain(hrSteps(r)) + '</button>';
    }).join('');
    var lst = '<section class="panel"><div class="sec-h"><h3>درخواست‌های محدوده من</h3><span class="aside">بررسی گروهی ندارد؛ هر درخواست جدا بررسی می‌شود</span></div>' + h.queues(QS, st.hq, 'data-hq') + (list.length ? '<div class="hr-list">' + cards + '</div>' : h.stateBlock('empty', 'درخواستی در این صف نیست', 'صف دیگری را ببینید.', '')) + '</section>';
    return h.pageHead({ title: 'درخواست‌های HR', sub: 'بررسی شما یک مرحله از مسیر است · اعمال نهایی همیشه با منابع انسانی است' }) + '<div class="form-card">' + lst + form + '</div>';
  };
  function hrSteps(r) {
    var done = st.hrDone[r.id]; if (!done) return r.steps;
    if (done === 'rejected') return [r.steps[0], ['سرپرست ارشد', 'rejected', 'کامران صدری · رد با دلیل']];
    return [r.steps[0], ['سرپرست ارشد', 'done', 'کامران صدری · تأیید این مرحله'], ['مرحله بعد طبق زنجیره', 'cur', 'در انتظار بررسی‌کننده بعدی'], r.steps[r.steps.length - 1]];
  }
  D.hr = function (id, keep) {
    var r = M.hr.filter(function (x) { return String(x.id) === String(id); })[0], done = st.hrDone[r.id], s = done === 'rejected' ? X.HRS.rejected : done ? X.HRS.stepdone : X.HRS[r.state];
    var canReview = r.state === 'pending_review' && r.reviewer === 'self' && !done, noteErr = keep && keep.err;
    var review = canReview ? X.sec('بررسی مرحله شما', '', '<div class="note info">' + ic('shield') + '<span><b>تأیید شما فقط همین مرحله را جلو می‌برد.</b> جابجایی یا قطع همکاری اعمال نمی‌شود؛ اعمال نهایی را منابع انسانی انجام می‌دهد.</span></div>' +
      '<label class="lbl" style="margin-top:12px">یادداشت یا دلیل رد' + (noteErr ? ' <span class="req">* الزامی برای رد</span>' : '') + '<textarea class="input" data-hrnote placeholder="برای رد کردن، نوشتن دلیل الزامی است"' + (noteErr ? ' aria-invalid="true"' : '') + '>' + esc((keep && keep.note) || '') + '</textarea>' + (noteErr ? '<div class="field-err">' + ic('alert') + '<span>برای رد درخواست دلیل بنویسید.</span></div>' : '') + '</label>', 'primary') : '';
    var info = !canReview ? X.sec('وضعیت', '', checks([r.state === 'pending_hr' || done === 'step' ? ['info', 'مرحله‌ی شما انجام شده است', 'اقدام فعلی با ' + (done ? 'بررسی‌کننده بعدی' : 'منابع انسانی') + ' است؛ از سمت شما اقدامی لازم نیست'] : ['info', 'این درخواست بسته شده است', r.applied ? 'اعمال‌شده · ' + r.applied : 'مرحله‌ای ادامه نمی‌یابد'], ['info', '«تأیید این مرحله» با «اعمال نهایی» فرق دارد', 'تغییر واقعی فقط پس از اعمال نهایی موفق ثبت می‌شود']])) : '';
    return top('درخواست HR #' + fa(r.id)) + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">' + (r.type === 'transfer' ? 'جابجایی' : 'قطع همکاری') + ' · ' + esc(seller(r.seller).name) + '</h2>' + pill(s.label, s.tone, s.icon) + '</div><div class="dr-meta">' + (r.dest ? '<span>مقصد: ' + esc(r.dest) + '</span>' : '') + '<span>درخواست‌دهنده: ' + esc(r.requester) + '</span><span>تیم فعلی: ' + esc(X.teamName(seller(r.seller).team)) + '</span></div></div>' +
      '<div class="dr-body"><section class="sec primary"><div class="sec-h"><h3>مسیر تأیید</h3></div>' + X.chain(hrSteps(r)) + '</section><section class="sec"><div class="sec-h"><h3>دلیل</h3></div><p style="margin:0">' + esc(r.reason) + '</p></section>' + review + info + '</div>' +
      foot(canReview ? '<button type="button" class="btn btn-lg btn-primary" data-act="hr-step:' + r.id + '">' + ic('check') + 'تأیید این مرحله و ارجاع به مرحله بعد</button><button type="button" class="btn btn-lg btn-danger" data-act="hr-reject:' + r.id + '">رد با دلیل</button>' : '', null);
  };

  /* ================= Detail drawers ================= */
  D.team = function (tid) {
    var t = team(tid), ss = X.sellersOf(tid), f = X.team_f(tid, 'current'), cov = X.teamCov(tid);
    var list = ss.map(function (s) { return '<button type="button" class="mini as-btn" data-act="open-seller:' + s.id + '"><div class="grow"><b>' + esc(s.name) + '</b><div class="muted" style="font-size:var(--t-meta)">پرونده باز ' + fa(s.open) + (s.moved ? ' · منتقل‌شده از ' + esc(X.teamName(s.moved.from)) : '') + '</div></div>' + (s.active ? pill('فعال', 'teal', 'checkCircle') : pill('غیرفعال', 'slate', 'ban')) + X.rel('indirect') + '</button>'; }).join('');
    return top('تیم و سرپرست مسئول') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">' + esc(t.sup) + '</h2>' + (t.active ? pill('فعال', 'teal', 'checkCircle') : pill('غیرفعال', 'slate', 'ban')) + X.rel('sup') + '</div><div class="dr-meta"><span>' + fa(ss.length) + ' فروشنده</span>' + (t.active ? '' : '<span>' + esc(t.inactiveNote) + '</span>') + '<span>' + X.cov(cov) + '</span></div></div>' +
      '<div class="dr-body">' + (t.active ? '' : '<section class="sec"><div class="note warn">' + ic('alert') + '<span>سرپرست این تیم غیرفعال است. گیرنده تخصیص نیست و مالک جدید خودکار تعیین نمی‌شود؛ مسئول جایگزین باید از طریق ساختار/منابع انسانی مشخص شود.</span></div></section>') +
      '<section class="sec primary"><div class="sec-h"><h3>بار فعلی تیم</h3>' + X.basis('current') + '</div><div class="sum-grid four"><div class="sum"><span>پرونده نزد تیم</span><b>' + fa(X.teamWorkload(tid)) + '</b></div><div class="sum"><span>فاکتور صادرشده</span><b>' + fa(f.created) + '</b></div><div class="sum"><span>پیش‌فاکتور باز</span><b>' + fa(f.openPre) + '</b></div><div class="sum"><span>فروش تکمیل‌شده</span><b>' + fa(f.completed) + '</b></div></div></section>' +
      X.details('فروشندگان این تیم', ss.length, '<div class="mini-list">' + list + '</div><p class="ind-note">غیرمستقیم برای شما: مشاهده و گزارش. دیدن یک فروشنده به معنی اجازه نوشتن برای او نیست.</p>', true) + '</div>' +
      foot(t.active ? '<button type="button" class="btn btn-lg btn-primary" data-act="assign-to:' + tid + '">' + ic('send') + 'تخصیص به این سرپرست</button>' : '', '<button type="button" class="btn btn-lg" data-act="drill:' + tid + '">' + ic('chart') + 'عملکرد این تیم</button>');
  };
  D.seller = function (id) {
    var s = seller(Number(id)), f = X.seller_f(s.id), cases = M.held.filter(function (c) { return c.holder.kind === 'seller' && c.holder.id === s.id; });
    return top(s.rel === 'direct' ? 'فروشنده مستقیم' : 'فروشنده زیرمجموعه') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">' + esc(s.name) + '</h2>' + (s.active ? pill('فعال', 'teal', 'checkCircle') : pill('غیرفعال', 'slate', 'ban')) + (s.rel === 'direct' ? X.rel('dseller') : X.rel('indirect')) + '</div><div class="dr-meta"><span>' + esc(X.teamName(s.team)) + '</span></div></div>' +
      '<div class="dr-body">' + (s.rel === 'indirect' ? '<section class="sec"><div class="note info">' + ic('eye') + '<span>این فروشنده زیر سرپرست است. شما فقط مشاهده و گزارش می‌بینید؛ تخصیص، برگشت و تماس از مسئولیت سرپرست او می‌ماند.</span></div></section>' : '') +
      (s.moved ? '<section class="sec"><div class="note conflict">' + ic('swap') + '<span>در ' + esc(s.moved.when) + ' از ' + esc(X.teamName(s.moved.from)) + ' منتقل شده است. فروش‌های قبل از انتقال به تیم هنگام صدور نسبت داده می‌شوند، نه تیم امروز.</span></div></section>' : '') +
      '<section class="sec primary"><div class="sec-h"><h3>خلاصه</h3>' + X.basis('event') + '</div><div class="sum-grid four"><div class="sum"><span>پرونده باز</span><b>' + fa(s.open) + '</b></div><div class="sum"><span>فاکتور صادرشده</span><b>' + fa(f.created) + '</b></div><div class="sum"><span>پیش‌فاکتور باز</span><b>' + fa(f.openPre) + '</b></div><div class="sum"><span>فروش تکمیل‌شده</span><b>' + fa(f.completed) + '</b></div></div></section>' +
      X.details('فاکتورها', f.list.length, '<div class="mini-list">' + (f.list.length ? f.list.map(function (i) { return '<button type="button" class="mini as-btn" data-act="open-inv:' + i.code + '"><div class="grow"><span class="mono">' + fa(i.code) + '</span><div class="muted" style="font-size:var(--t-meta)">' + esc(i.customer) + '</div></div>' + pill(X.INVS[i.inv].label, X.INVS[i.inv].tone, X.INVS[i.inv].icon) + '</button>'; }).join('') : '<div class="muted">فاکتوری ثبت نشده است.</div>') + '</div>', f.list.length > 0) + '</div>' +
      foot(s.rel === 'indirect' ? '<button type="button" class="btn btn-lg btn-primary" data-act="hr-for:' + s.id + '">' + ic('briefcase') + 'درخواست HR برای این فروشنده</button>' : '', null);
  };
  D.attn = function (id) {
    var a = X.attention().filter(function (x) { return x.id === id; })[0]; if (!a) return top('نیازمند هماهنگی') + '<div class="dr-body">' + h.stateBlock('empty', 'موردی نیست', '') + '</div>';
    return top('نیازمند هماهنگی') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">' + esc(a.label) + '</h2>' + pill(fa(a.n) + ' مورد', a.tone, a.icon) + '</div></div><div class="dr-body">' +
      '<section class="sec primary"><dl class="exc-dl"><div><dt>موضوع</dt><dd>' + esc(a.subject) + '</dd></div><div><dt>مسئول اقدام</dt><dd>' + esc(a.owner) + '</dd></div><div><dt>وضعیت</dt><dd>' + esc(a.state) + '</dd></div><div><dt>دلیل</dt><dd>' + esc(a.reason) + '</dd></div><div><dt>اقدام مجاز بعدی برای شما</dt><dd>' + esc(a.next) + '</dd></div></dl></section>' +
      X.sec('ارجاع رسمی', '', checks([['info', 'دستور ارجاع/تشدید در این مرحله تعریف نشده است', 'فقط مسئول و وضعیت نشان داده می‌شود؛ مسئول جدید ساخته یا اعلان ارسال نمی‌شود (SG-02).']])) + '</div>' +
      foot(a.drawer ? '<button type="button" class="btn btn-lg btn-primary" data-act="open-diag">' + ic('chart') + 'جزئیات تطبیق</button>' : '<button type="button" class="btn btn-lg btn-primary" data-act="attn-go:' + a.id + '">' + ic('arrowL') + 'رفتن به صفحه مرتبط</button>', null);
  };
})();
