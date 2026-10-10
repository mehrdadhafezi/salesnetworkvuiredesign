/* Sales Manager — role layer, part 3: distribution & return, HR chain, extra-number decision, operation results, detail drawers.
   Direct-level recipients are the normal path; skip-level recipients are a CONDITIONAL group (current technical path kept, no approval chain invented).
   Return is previewed only: activation is CONDITIONAL on Gate 0 F01 (M-G01). Nothing here writes anywhere. */
(function () {
  'use strict';
  var X = window.MGRX, C = X.C, M = X.M, h = X.h, st = X.st, V = X.V, D = X.D;
  var esc = h.esc, fa = h.fa, num = h.num, money = h.money, ic = h.ic, pill = h.pill, hint = h.hint, cbx = h.cbx, $ = h.$;
  var seller = X.seller, team = X.team, unit = X.unit, keys = X.keys, top = X.top, foot = X.foot, own = X.own, checks = X.checks, tl = X.tl;

  /* ================= Distribution & Eligible Return ================= */
  V.dist = function () {
    var ok = X.poolOk().length, review = M.pool.length - ok, issue = M.ops.filter(function (o) { var c = X.counts(o); return c.failed || c.skipped || c.rejected || c.unknown; }).length;
    var direct = M.units.filter(function (u) { return u.active; }).length + X.teamsOf('M').length + M.sellers.filter(function (s) { return s.team === 'direct'; }).length;
    var kpis = [
      { label: 'موجودی پنل من', value: fa(M.pool.length), color: 'neutral', keep: true, meaning: 'پرونده‌های تحویل‌شده به شما که هنوز تخصیص نیافته‌اند', basis: 'وضعیت در لحظه' },
      { label: 'قابل تخصیص', value: fa(ok), color: 'teal', keep: true, meaning: 'بدون وابستگی مالی و تعارض؛ هنگام ثبت دوباره بررسی می‌شود', basis: 'وضعیت در لحظه' },
      { label: 'نیازمند بررسی', value: fa(review), tone: review ? 'warn' : '', color: 'amber', meaning: 'مسدود یا نیازمند تطبیق', basis: 'وضعیت در لحظه' },
      { label: 'گیرنده مستقیم فعال', value: fa(direct), color: 'blue', meaning: 'سرپرست ارشد، سرپرست و فروشنده‌ای که مستقیم زیر نظر شما هستند', basis: 'ساختار فعلی' }
    ];
    var q = h.queues([
      { id: 'assign', label: 'تخصیص', icon: 'send', tone: 'blue', n: ok, key: '1' },
      { id: 'return', label: 'برگشت (پیش‌نمایش)', icon: 'repeat', tone: 'teal', n: M.held.filter(function (c) { return c.elig === 'ok'; }).length, key: '2' },
      { id: 'ops', label: 'سوابق عملیات', icon: 'history', tone: issue ? 'orange' : 'neutral', n: M.ops.length, key: '3' }
    ], st.aq, 'data-aq');
    var body = st.aq === 'return' ? returnView() : st.aq === 'ops' ? opsView() : assignView();
    return h.pageHead({ title: 'توزیع', sub: 'پرونده‌های پنل شما ← سرپرست ارشد، سرپرست یا فروشنده مجاز · مسئول فعلی عوض می‌شود؛ مالک اولیه و مالک اعتبار نه', kpis: kpis, fresh: X.freshPart('فهرست') }) + X.banners() +
      '<section class="panel main' + (st.aq === 'ops' ? '' : ' with-bulk') + '">' + q + body + '</section>';
  };

  function plan() {
    var ids, recs;
    if (st.amode === 'select') { ids = keys(st.asel); recs = st.recip ? [X.recip(st.recip)] : []; }
    else { recs = keys(st.recips).map(X.recip); ids = X.poolOk().map(function (c) { return c.id; }).slice(0, st.per * recs.length); }
    return { ids: ids, recs: recs, skip: recs.some(function (r) { return r.bypass.length; }) };
  }
  function recipLabel(r) {
    var base = r.level === 'سرپرست ارشد' ? 'سرپرست ارشد · پرونده نزد واحد ' + fa(X.unitWorkload(r.k)) + '، فروشنده فعال ' + fa(X.sellersOfUnit(r.k).filter(function (s) { return s.active; }).length)
      : r.level === 'سرپرست' ? 'سرپرست · پرونده نزد تیم ' + fa(X.teamWorkload(r.k)) + '، فروشنده فعال ' + fa(X.sellersOf(r.k).filter(function (s) { return s.active; }).length)
      : 'فروشنده · پرونده باز ' + fa(seller(Number(r.k.slice(1))).open);
    return r.ok ? base + (r.bypass.length ? ' · عبور از ' + r.bypass.join('، ') : ' · مستقیم') : r.why;
  }
  function recipRow(k, single, extra) {
    var r = X.recip(k), on = single ? st.recip === k : !!st.recips[k], dis = !r.ok || (!single && r.bypass.length);
    return '<label class="recip' + (r.ok ? '' : ' off') + (on ? ' on' : '') + (extra ? ' ' + extra : '') + '"><input type="' + (single ? 'radio' : 'checkbox') + '" name="recip" class="cbx" data-recip="' + k + '"' + (on ? ' checked' : '') + (dis ? ' disabled' : '') + '><span class="r-main"><b>' + esc(r.name) + '</b><span>' + esc(recipLabel(r)) + '</span></span></label>';
  }
  function assignView() {
    var selIds = keys(st.asel), recN = keys(st.recips).length, rec = st.recip ? X.recip(st.recip) : null;
    var stepCur = st.amode === 'select' ? (!selIds.length ? 0 : !rec ? 1 : 2) : (!recN ? 1 : 2);
    var head = '<div class="flow-head">' + X.steps(['انتخاب پرونده‌ها', 'انتخاب گیرنده', 'بررسی اثر', 'ثبت و نتیجه'], stepCur) +
      '<div class="seg" role="group" aria-label="روش تخصیص"><button type="button" data-amode="select" aria-pressed="' + (st.amode === 'select') + '">انتخاب پرونده‌ها</button><button type="button" data-amode="count" aria-pressed="' + (st.amode === 'count') + '">تعداد برای هر گیرنده</button></div></div>';
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
      left = '<div class="count-form"><label class="lbl">تعداد پرونده برای هر گیرنده <span class="req">*</span><input class="input" type="number" min="1" max="50" value="' + st.per + '" data-per inputmode="numeric" style="max-width:160px"></label>' +
        '<label class="lbl">دسته / منبع<select class="input" style="max-width:320px"><option>همه پرونده‌های تحویل‌شده</option><option>MIS · دسته ۹۲۰۱</option><option>MIS · دسته ۹۲۰۲</option></select></label>' +
        '<div class="note info">' + ic('info') + '<span>' + (recN ? fa(st.per) + ' × ' + fa(recN) + ' گیرنده = <b>' + fa(total) + ' پرونده</b> از ' + fa(okN) + ' پرونده قابل تخصیص، به ترتیب قدیمی‌ترین دریافت.' + (total > okN ? ' <b>تعداد درخواستی بیشتر از موجودی قابل تخصیص است.</b>' : '') : 'گیرندگان مستقیم را از فهرست انتخاب کنید.') + '</span></div>' +
        '<div class="muted" style="font-size:var(--t-meta)">روش شمارشی فقط برای گیرندگان مستقیم است؛ فهرست دقیق پرونده‌ها و نتیجه هر پرونده در بررسی اثر و نتیجه می‌آید.</div></div>';
    }
    var single = st.amode === 'select';
    var directKeys = M.units.map(function (u) { return u.id; }).concat(X.teamsOf('M').map(function (t) { return t.id; })).concat(M.sellers.filter(function (s) { return s.team === 'direct'; }).map(function (s) { return 'S' + s.id; }));
    var skipKeys = M.teams.filter(function (t) { return t.parent !== 'M'; }).map(function (t) { return t.id; }).concat(M.sellers.filter(function (s) { return s.team !== 'direct'; }).map(function (s) { return 'S' + s.id; }));
    var excOpen = st.skipOpen || (st.recip && X.isSkip(st.recip));
    var aside = '<aside class="recips" aria-labelledby="rc-h"><div class="sec-h"><h3 id="rc-h">گیرنده ' + (single ? '(یک نفر)' : '(چند گیرنده مستقیم)') + '</h3><span class="scope-badge">' + ic('users') + 'مسیر مستقیم</span></div>' + directKeys.map(function (k) { return recipRow(k, single); }).join('') +
      '<details class="exc-path"' + (excOpen ? ' open' : '') + ' data-exc><summary>' + ic('split') + '<span><b>عبور از یک یا چند سطح (skip-level)</b><span>مشروط · مسیر فنی فعلی حفظ شده · سیاست مصوب نیست (M-G08)</span></span><span class="chev">' + ic('chev') + '</span></summary>' +
      '<p class="recip-note">' + ic('alert') + 'نمایش این گزینه‌ها به معنی مجاز بودن در هر شرایطی نیست. گیرنده باید فعال و در محدوده همین اقدام باشد؛ دیدن یک نفر مجوز نوشتن برای او نیست. ' + (single ? '' : 'در حالت «تعداد برای هر گیرنده» فقط گیرندگان مستقیم انتخاب‌شدنی‌اند.') + '</p>' + skipKeys.map(function (k) { return recipRow(k, single, 'exc'); }).join('') + (st.recip && X.isSkip(st.recip) ? '<label class="lbl exc-reason">دلیل (برای ردیابی در سابقه عملیات)<textarea class="input" data-skipreason placeholder="چرا این تخصیص از سطح میانی عبور می‌کند؟">' + esc(st.skipReason) + '</textarea><span class="muted" style="font-size:var(--t-micro)">الزامی نیست؛ زنجیره تأیید جدیدی تعریف نشده است.</span></label>' : '') + '</details>' +
      '<p class="recip-note">' + ic('info') + 'گیرنده غیرفعال یا بدون مجوز اثبات‌شده انتخاب‌شدنی نیست و مالک جدید خودکار تعیین نمی‌شود.</p></aside>';
    var bb = '';
    if (st.amode === 'select' && selIds.length) bb = X.bulkbar(fa(selIds.length) + ' پرونده انتخاب شد', rec ? 'گیرنده: ' + esc(rec.name) + (rec.bypass.length ? ' · عبور از ' + esc(rec.bypass.join('، ')) : ' · مستقیم') : 'یک گیرنده انتخاب کنید', '<button type="button" class="btn btn-ghost btn-sm" data-act="clear-asel">پاک کردن انتخاب</button><button type="button" class="btn btn-primary" data-act="review-assign"' + (rec ? '' : ' disabled aria-disabled="true"') + '>' + ic('eye') + 'بررسی اثر</button>');
    if (st.amode === 'count' && recN) bb = X.bulkbar(fa(st.per * recN) + ' پرونده برای ' + fa(recN) + ' گیرنده', 'ثبت فقط پس از بررسی اثر', '<button type="button" class="btn btn-primary" data-act="review-assign">' + ic('eye') + 'بررسی اثر</button>');
    return head + '<div class="split"><div class="split-main">' + left + '</div>' + aside + '</div>' + bb;
  }

  function heldList() {
    var f = function (c) { return st.rf === 'all' ? true : st.rf === 'legacy' ? !!c.legacy : st.rf === 'review' ? (c.elig === 'unknown' || c.elig === 'conflict') : c.elig === st.rf; };
    return M.held.filter(f);
  }
  function retWhy(c) {
    if (c.elig === 'ok') return 'انتقال خودِ شما · فاکتور/پرداخت متصلی در پیش‌نمایش یافت نشد';
    return ({ fin: 'وابستگی مالی: ' + (c.invoice && c.invoice.label), notown: 'انتقال توسط سرپرست ارشد انجام شده؛ رتبه اجازه بازپس‌گیری نمی‌دهد', mis: 'منبع MIS؛ برگشت با MIS است', consumed: 'مصرف‌شده در مسیر فاکتور V4: ' + (c.invoice && c.invoice.label), cancel: 'فاکتور لغوشده — قانون آزادسازی تعیین نشده (OPD-03)' })[c.why] || c.why;
  }
  function retCase(id) { return M.held.filter(function (x) { return x.id === id; })[0]; }
  function proofCell(c) {
    var p = c.proof;
    if (p.by === 'self') return '<span class="actor-who a-self">' + ic('user') + 'شما</span><div class="cell-sub">' + esc(p.op) + ' · ' + esc(p.when) + '</div>';
    if (p.by === 'senior') return '<span class="actor-who a-senior">' + ic('user') + 'سرپرست ارشد</span><div class="cell-sub">' + esc(p.op) + '</div>';
    if (p.by === 'mis') return '<span class="actor-who a-mis">' + ic('layers') + 'MIS</span><div class="cell-sub">' + esc(p.op) + '</div>';
    return '<span class="muted">ثبت نشده</span><div class="cell-sub">' + esc(p.op) + '</div>';
  }
  function returnView() {
    var list = heldList(), okIds = list.filter(function (c) { return c.elig === 'ok'; }).map(function (c) { return c.id; });
    var selIds = keys(st.rsel).filter(retCase), allOn = okIds.length && okIds.every(function (id) { return st.rsel[id]; });
    var seg = '<div class="seg" role="group" aria-label="فیلتر قابلیت برگشت">' + [['all', 'همه'], ['ok', 'واجد شرایط'], ['blocked', 'مسدود'], ['review', 'نیازمند بررسی'], ['legacy', 'سرنخ قدیمی']].map(function (f) { return '<button type="button" data-rf="' + f[0] + '" aria-pressed="' + (st.rf === f[0]) + '">' + f[1] + '</button>'; }).join('') + '</div>';
    var rows = list.map(function (c) {
      var e = X.RET[c.elig], dis = c.elig !== 'ok';
      return '<tr data-row="ret:' + c.id + '" tabindex="-1" class="' + (st.rsel[c.id] ? 'picked' : '') + '"><td class="col-sel">' + cbx('data-sel="rsel" data-id="' + c.id + '"', st.rsel[c.id], 'انتخاب پرونده ' + c.id + (dis ? ' — ' + e.label : ''), dis) + '</td>' +
        '<td><span class="mono case-id">' + esc(c.id) + '</span>' + (c.legacy ? ' <span class="tag legacy">قدیمی</span>' : '') + '<div class="cell-sub">' + esc(c.source) + '</div></td>' +
        '<td>' + X.who(X.holderName(c), c.holder.kind === 'senior' ? X.rel('senior') : c.holder.kind === 'sup' ? X.rel('sup') : X.rel('indirect')) + '</td><td class="col-opt">' + proofCell(c) + '</td>' +
        '<td>' + (c.invoice ? '<span class="mono">' + fa(c.invoice.code) + '</span><div class="cell-sub">' + esc(c.invoice.label) + '</div>' : '<span class="muted">در پیش‌نمایش یافت نشد</span>') + (c.link ? '<div class="cell-sub">' + X.link(c.link) + '</div>' : '') + '</td>' +
        '<td>' + pill(e.label, e.tone, e.icon) + '<div class="cell-sub">' + esc(retWhy(c)) + '</div></td>' +
        '<td class="col-actions"><button type="button" class="btn btn-sm" data-act="open-ret:' + c.id + '">بررسی</button></td></tr>';
    }).join('');
    var bb = selIds.length ? X.bulkbar(fa(selIds.length) + ' پرونده برای پیش‌نمایش انتخاب شد', 'ثبت برگشت فعال نیست (مشروط به F01)', '<button type="button" class="btn btn-ghost btn-sm" data-act="clear-rsel">پاک کردن انتخاب</button><button type="button" class="btn btn-primary" data-act="review-return">' + ic('eye') + 'بررسی برگشت</button>') : '';
    return h.banner('locked', '<b>برگشت فقط پیش‌نمایش است.</b> حفاظت کامل ثبت برگشت (بررسی همه ارتباط‌های مالی با اتصال اثبات‌شده، عدم‌موفقیت در ابهام و بررسی مجدد هنگام ثبت) هنوز تکمیل نشده است (Gate 0 F01 / M-G01). متن «موارد امن» تضمین امنیت مالی نیست.', '') +
      '<div class="note info inset">' + ic('shield') + '<span><b>مدیر فروش همه چیز را بازپس نمی‌گیرد.</b> اختیار برگشت از سابقه انتقال/مسئولیت تعیین می‌شود، نه رتبه. در ابهام یا اتصال ناقص، نتیجه «مسدود/نامعلوم» است و حدس با شماره موبایل انجام نمی‌شود.</span></div>' +
      '<div class="toolbar">' + seg + '<span class="grow"></span><span class="scope-badge view">' + ic('lock') + 'ثبت و برگشت گروهی: مشروط</span><span class="muted desk">' + fa(list.length) + ' پرونده</span></div>' +
      (list.length ? '<div class="tbl-wrap"><table class="tbl selectable" aria-label="پرونده‌های پایین‌دست"><thead><tr><th class="col-sel" data-l="انتخاب">' + cbx('data-selall="rsel"', allOn, 'انتخاب همه پرونده‌های واجد شرایط پیش‌نمایش') + '</th><th>پرونده و منبع</th><th>نزد (مسئول فعلی)</th><th class="col-opt">انتقال توسط ' + hint('عامل رویداد انتقال؛ مالک فعلی یا مالک اعتبار نیست.', true) + '</th><th>فاکتور / اتصال</th><th>وضعیت برگشت و دلیل</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div>'
        : h.stateBlock('noresult', 'پرونده‌ای با این فیلتر نیست', 'فیلتر دیگری انتخاب کنید.', '<button type="button" class="btn btn-soft" data-rf="all">همه پرونده‌ها</button>')) + bb;
  }
  function opState(o) { var c = X.counts(o); return c.unknown ? 'unknown' : c.ok === c.requested ? 'complete' : c.ok ? 'partial' : 'failed'; }
  X.opState = opState;
  function opsView() {
    var rows = M.ops.map(function (o) {
      var s = X.OPS[opState(o)], c = X.counts(o);
      return '<tr data-row="result:' + o.ref + '" tabindex="-1"><td><span class="mono case-id">' + o.ref + '</span></td><td>' + (o.kind === 'assign' ? 'تخصیص' : 'برگشت') + '</td><td class="muted">' + esc(o.when) + '</td><td class="col-opt">' + esc(o.to) + '</td><td>' + pill(s[0], s[1], s[2]) + '<div class="cell-sub">' + (c.unknown ? fa(c.unknown) + ' مورد نامعلوم' : fa(c.ok) + ' از ' + fa(c.requested) + ' اعمال شد') + '</div></td><td class="col-actions"><button type="button" class="btn btn-sm' + (opState(o) === 'complete' ? '' : ' btn-soft') + '" data-act="open-op:' + o.ref + '">' + (opState(o) === 'complete' ? 'جزئیات' : 'رسیدگی') + '</button></td></tr>';
    }).join('');
    return h.toolbar('شماره عملیات یا پرونده', [{ label: 'نوع عملیات', icon: 'filter' }, { label: 'بازه زمانی', icon: 'clock' }]) +
      '<div class="tbl-wrap"><table class="tbl" aria-label="سوابق عملیات"><thead><tr><th>عملیات</th><th>نوع</th><th>زمان</th><th class="col-opt">مقصد</th><th>نتیجه</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div>' + h.tfoot(M.ops.length, M.ops.length);
  }

  /* ---------- Distribution drawers ---------- */
  D.pcase = function (id) {
    var c = M.pool.filter(function (x) { return x.id === id; })[0], e = X.ELIG[c.elig];
    return top('پرونده در پنل شما') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name" class="mono">' + esc(c.id) + '</h2>' + pill(e.label, e.tone, e.icon) + '</div><div class="dr-meta"><span>' + esc(c.source) + '</span><span>دریافت: ' + esc(c.received) + '</span></div></div>' +
      '<div class="dr-body"><section class="sec">' + own({ custody: 'پنل شما (' + M.user.name + ')', original: c.source, next: 'شما — تخصیص به گیرنده مجاز', event: 'سامانه توزیع MIS · ' + c.received, credit: 'هنوز فروشی ثبت نشده' }) + '</section>' +
      X.sec('قابلیت تخصیص', '', c.elig === 'ok' ? checks([['ok', 'وابستگی مالی ندارد', 'فاکتور، پیش‌فاکتور یا پرداخت متصل یافت نشد'], ['ok', 'نزد گیرنده‌ای نیست', 'مسئول فعلی: پنل شما'], ['info', 'هنگام ثبت دوباره بررسی می‌شود', 'این نمایش رزرو یا تضمین نیست']]) : checks([[c.elig === 'blocked' ? 'no' : 'q', e.label, c.why]]), c.elig === 'ok' ? '' : 'primary') + '</div>' +
      foot(c.elig === 'ok' ? '<button type="button" class="btn btn-lg btn-primary" data-act="pick-one:' + c.id + '">' + ic('check') + 'افزودن به انتخاب</button>' : '', null);
  };
  D.assignReview = function (id, keep) {
    var p = plan(), conflict = st.flow === 'conflict', n = p.ids.length, names = p.recs.map(function (r) { return r.name; }).join('، '), confirmed = keep && keep.confirm;
    var each = st.amode === 'select' ? n : Math.min(st.per, n);
    var impact = p.recs.map(function (r) {
      var cur = r.kind === 'senior' ? X.unitWorkload(r.k) : r.kind === 'sup' ? X.teamWorkload(r.k) : seller(Number(r.k.slice(1))).open, add = each - (conflict ? 1 : 0);
      return '<div class="mini"><div class="grow"><b>' + esc(r.name) + '</b> <span class="muted">(' + esc(r.level) + ')</span><div class="muted" style="font-size:var(--t-meta)">پرونده ' + (r.kind === 'seller' ? 'باز' : 'نزد واحد') + ' ' + fa(cur) + ' ← <b>' + fa(cur + add) + '</b>' + (r.kind !== 'seller' ? ' · گیرنده آن را میان زیرمجموعه خود پخش می‌کند' : '') + '</div></div><span class="tag">+' + fa(add) + '</span></div>';
    }).join('');
    var skip = '';
    if (p.skip) {
      var r0 = p.recs.filter(function (r) { return r.bypass.length; })[0];
      skip = X.sec('مسیر عبور از سطح (مشروط)', 'M-G08', '<div class="exc-grid"><div><span>گیرنده</span><b>' + esc(r0.name) + ' · ' + esc(r0.level) + '</b></div><div><span>سطح(های) ردشده</span><b>' + esc(r0.bypass.join('، ')) + '</b></div><div><span>مسئول فعلی پرونده‌ها</span><b>پنل شما</b></div><div><span>دلیل (برای ردیابی در سابقه)</span><b>' + (st.skipReason.trim() ? esc(st.skipReason) : '<span class="muted">وارد نشده — الزامی نیست</span>') + '</b></div></div>' +
        checks([['ok', 'گیرنده فعال و در محدوده این اقدام است', 'طبق بررسی همین لحظه'], ['warn', 'سیاست عبور از سطح مصوب نیست', 'مسیر فنی فعلی بدون تغییر حفظ شده؛ زنجیره تأیید جدیدی اختراع نشده است.'], ['info', 'مالک اولیه و مالک اعتبار تغییر نمی‌کند', 'یک رویداد انتقال با نام شما ثبت می‌شود.']]) +
        '<label class="confirm"><input type="checkbox" class="cbx" data-confirm' + (confirmed ? ' checked' : '') + '><span>می‌دانم که این تخصیص از ' + esc(r0.bypass.join('، ')) + ' عبور می‌کند و همین مسئولیت را می‌پذیرم.</span></label>', 'primary');
    }
    var canCommit = !conflict && n > 0 && (!p.skip || confirmed);
    var primary = conflict ? '<button type="button" class="btn btn-lg btn-primary" data-act="refresh-review">' + ic('refresh') + 'بازخوانی بررسی</button>' : '<button type="button" class="btn btn-lg btn-primary" data-act="commit-assign"' + (canCommit ? '' : ' disabled aria-disabled="true"') + '>' + ic('send') + 'تأیید و ثبت تخصیص</button>';
    return top('بررسی پیش از تخصیص') + '<div class="dr-head">' + X.steps(['انتخاب', 'گیرنده', 'بررسی اثر', 'نتیجه'], 2) + '<div class="dr-title" style="margin-top:12px"><h2 id="dr-name">تخصیص ' + fa(n) + ' پرونده</h2></div><div class="dr-meta"><span>به: ' + esc(names) + '</span><span>' + (st.amode === 'select' ? 'پرونده‌های انتخاب‌شده' : fa(st.per) + ' پرونده برای هر گیرنده') + '</span></div></div>' +
      '<div class="dr-body">' + (conflict ? '<section class="sec"><div class="note conflict" role="alert">' + ic('swap') + '<span><b>وضعیت پرونده تغییر کرده است.</b> پرونده C-41003 پس از این بررسی توسط MIS از پنل شما برداشته شد. بازخوانی کنید؛ هیچ تغییری اعمال نشده است.</span></div></section>' : '') +
      '<section class="sec primary"><div class="sum-grid four"><div class="sum"><span>درخواست‌شده</span><b>' + fa(n) + '</b></div><div class="sum"><span>واجد شرایط (این لحظه)</span><b style="color:var(--teal-fg)">' + fa(conflict ? n - 1 : n) + '</b></div><div class="sum"><span>مسدود (حذف شد)</span><b>۰</b></div><div class="sum"><span>تعارض</span><b' + (conflict ? ' style="color:var(--orange-fg)"' : '') + '>' + fa(conflict ? 1 : 0) + '</b></div></div></section>' +
      X.sec('اثر بر گیرنده', '', '<div class="mini-list">' + impact + '</div>') + skip +
      X.sec('این بررسی چه چیزی را تضمین نمی‌کند؟', '', checks([['info', 'پیش‌نمایش ≠ ثبت', 'هنگام ثبت، دارنده، گیرنده فعال، وضعیت پرونده، وابستگی مالی و محدوده دوباره بررسی می‌شود'], ['info', 'نتیجه به‌تفکیک هر پرونده نمایش داده می‌شود', 'درخواست‌شده / واجد شرایط / اعمال‌شده / ارسال‌نشده / ردشده / ناموفق / نامعلوم؛ «ثبت شد» یعنی انتقال ثبت شده، نه تأیید دریافت.']])) + '</div>' +
      foot(primary, '<button type="button" class="btn btn-lg btn-ghost" data-close>بازگشت به انتخاب</button>', 'چیزی بدون تأیید شما ثبت نمی‌شود');
  };

  // Operation results: per-item outcome; partial ≠ success; unknown is reconciled before any retry.
  function makeResult(kind, ids, toName, flow) {
    var items = ids.map(function (id) { return [id, 'ok', toName]; }), note = null, n = items.length;
    if (flow === 'partial' && n > 4) { items[n - 1] = [ids[n - 1], 'failed', toName, 'پاسخ سرور ناقص بود؛ قابل تکرار']; items[n - 2] = [ids[n - 2], 'skipped', toName, 'پرونده پیش از ارسال تغییر کرد (مسئول فعلی عوض شد)']; items[n - 3] = [ids[n - 3], 'rejected', toName, 'سامانه ثبت را نپذیرفت: وابستگی مالی فعال']; }
    else if (flow === 'retry') items = items.map(function (x) { return [x[0], 'failed', toName, 'خطای موقت شبکه؛ همان قصد قابل تکرار است']; });
    else if (flow === 'unknown') { items = items.map(function (x) { return [x[0], 'unknown', toName]; }); note = 'پاسخ تأیید نرسید؛ ممکن است ثبت شده باشد. پیش از تکرار با شماره عملیات تطبیق کنید.'; }
    return { ref: 'OP-5310', kind: kind, when: 'همین الان', actor: M.user.name, requested: n, to: toName, items: items, note: note };
  }
  function resultModel(id) { return id === 'live' ? st.live : M.ops.filter(function (o) { return o.ref === id; })[0]; }
  X.applyOk = function (op) {
    var okIds = op.items.filter(function (x) { return x[1] === 'ok'; }).map(function (x) { return x[0]; });
    if (op.kind === 'assign') M.pool = M.pool.filter(function (c) { return okIds.indexOf(c.id) < 0; });
    else okIds.forEach(function (id) { var c = retCase(id); if (c) { M.held = M.held.filter(function (x) { return x.id !== id; }); M.pool.push({ id: id, source: 'برگشت از ' + X.holderName(c), received: 'امروز', elig: 'ok' }); } });
    st.asel = {}; st.rsel = {};
  };
  X.runAssign = function (flow) {
    var p = plan(), op = makeResult('assign', p.ids.slice(), p.recs.map(function (r) { return r.name; }).join('، '), flow);
    st.live = op; st.flow = null; if (X.counts(op).ok) X.applyOk(op); M.ops.unshift(op);
    C.closeDrawer(); st.aq = 'ops'; C.render(); C.openDrawer('result', 'OP-5310');
  };
  D.result = function (id) {
    var o = id === 'OP-5310' ? M.ops[0] : resultModel(id);
    if (!o) return top('نتیجه') + '<div class="dr-body">' + h.stateBlock('error', 'نتیجه پیدا نشد', '') + '</div>';
    var s = opState(o), S = X.OPS[s], kind = o.kind === 'assign' ? 'تخصیص' : 'برگشت', c = X.counts(o);
    var headline = { complete: kind + ' کامل انجام شد: ' + fa(c.ok) + ' از ' + fa(c.requested) + ' پرونده اعمال شد.', partial: kind + ' ناقص انجام شد: ' + fa(c.ok) + ' از ' + fa(c.requested) + ' پرونده اعمال شد و ' + fa(c.requested - c.ok) + ' پرونده اعمال نشد.', failed: kind + ' انجام نشد: هیچ پرونده‌ای اعمال نشد.', unknown: 'نتیجه ' + kind + ' نامعلوم است: ممکن است اعمال شده باشد.' }[s];
    var bad = o.items.filter(function (x) { return x[1] !== 'ok'; }), good = o.items.filter(function (x) { return x[1] === 'ok'; });
    var itemRow = function (x) { var q = X.OUT[x[1]]; return '<li class="ir"><span class="mono">' + x[0] + '</span>' + (x[2] ? '<span class="ir-to">' + esc(x[2]) + '</span>' : '') + pill(q.label, q.tone, q.icon) + (x[3] ? '<span class="ir-why">' + esc(x[3]) + '</span>' : '') + '</li>'; };
    var retryN = c.failed;
    var primary = c.unknown ? '<button type="button" class="btn btn-lg btn-primary" data-act="reconcile:' + id + '">' + ic('refresh') + 'تطبیق با شماره عملیات</button>' : retryN ? '<button type="button" class="btn btn-lg btn-primary" data-act="retry:' + id + '">' + ic('refresh') + 'تلاش مجدد فقط برای ' + fa(retryN) + ' مورد ناموفق</button>' : '';
    return top('نتیجه عملیات', '<span class="mono" style="margin-left:8px">' + esc(o.ref) + '</span>') +
      '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">نتیجه ' + kind + '</h2>' + pill(S[0], S[1], S[2]) + '</div><div class="dr-meta"><span>' + esc(o.when) + '</span><span>' + esc(o.actor) + '</span><span>مقصد: ' + esc(o.to) + '</span></div></div>' +
      '<div class="dr-body"><section class="sec primary"><p class="result-line r-' + s + '" role="status">' + ic(S[2]) + '<span>' + esc(headline) + '</span></p>' + X.outcomeStrip(o) +
      (o.note ? '<div class="note warn" style="margin-top:12px">' + ic('question') + '<span>' + esc(o.note) + '</span></div>' : '') + (s === 'complete' ? '<div class="muted" style="margin-top:12px;font-size:var(--t-meta)">«اعمال شد» یعنی انتقال در سیستم ثبت شده است؛ تأیید دریافت توسط گیرنده جداگانه ثبت نمی‌شود.</div>' : '') + '</section>' +
      (bad.length ? '<section class="sec"><div class="sec-h"><h3>موارد نیازمند رسیدگی</h3><span class="aside">' + fa(bad.length) + ' مورد</span></div><ul class="ir-list">' + bad.map(itemRow).join('') + '</ul></section>' : '') +
      (good.length ? '<details class="sec"' + (bad.length ? '' : ' open') + '><summary><h3>اعمال‌شده‌ها</h3><span class="muted" style="margin-right:8px;font-size:var(--t-meta)">' + fa(good.length) + '</span><span class="chev">' + ic('chev') + '</span></summary><ul class="ir-list">' + good.map(itemRow).join('') + '</ul></details>' : '') + '</div>' +
      foot(primary, '<button type="button" class="btn btn-lg btn-ghost" data-close>بستن</button>', c.failed ? 'فقط موارد «ناموفق» تکرار می‌شوند؛ «ارسال‌نشده» و «ردشده» با بازخوانی بررسی می‌شوند' : '');
  };
  X.reconcile = function (id) {
    var u = resultModel(id) || M.ops[0], half = Math.ceil(u.items.length / 2);
    u.items.forEach(function (x, n) { x[1] = n < half ? 'ok' : 'failed'; if (n >= half) x[3] = 'تطبیق: ثبت نشده بود؛ اکنون قابل تکرار است'; });
    u.note = null; X.applyOk(u);
  };
  X.retryFailed = function (id) { var o = resultModel(id) || M.ops[0]; o.items.forEach(function (x) { if (x[1] === 'failed') x[1] = 'ok'; }); X.applyOk(o); };

  D.ret = function (id) {
    var c = retCase(id), e = X.RET[c.elig], holder = X.holderName(c), p = c.proof, lvl = X.holderLevel(c);
    var body, resp = null;
    if (c.elig === 'ok') body = checks([['ok', 'انتقال را خودِ شما انجام داده‌اید', p.op + ' · ' + p.when], ['q', 'ارتباط‌های مالی: در پیش‌نمایش یافت نشد', 'بررسی کامل همه حوزه‌ها (قدیمی، V4، جریان فروشنده، رویداد مالی، Dot) با اتصال اثبات‌شده هنوز به‌عنوان حفاظت فعال نیست (F01)'], ['ok', 'نزد ' + lvl + ' فعال و در محدوده شماست', holder], ['info', 'مالک اولیه، منبع و سوابق تماس بدون تغییر می‌ماند', '']]);
    else if (c.why === 'fin') { body = checks([['no', 'وابستگی مالی محافظت‌شده دارد', 'فاکتور ' + fa(c.invoice.code) + ' · ' + c.invoice.label], ['info', 'چرا مسدود است؟', 'پرونده‌ای که فاکتور، پیش‌فاکتور یا پرداخت دارد با برگشت عادی آزاد نمی‌شود تا سوابق مالی و اعتبار حفظ شود.']]); resp = { who: 'sales', text: 'فروشنده / مالی طبق وضعیت فاکتور' }; }
    else if (c.why === 'consumed') { body = checks([['no', 'پرونده مصرف‌شده است', 'مسیر پیگیری فاکتور V4 فعال است: فاکتور ' + fa(c.invoice.code) + ' · ' + c.invoice.label], ['info', 'برگشت عادی انجام نمی‌شود', 'پاک‌کردن اتصال، حل تعارض حساب نمی‌شود؛ باید اثبات اتصال با مالی/MIS تطبیق شود.']]); resp = { who: 'finance', text: 'مالی/MIS برای تطبیق؛ فروش برای اقدام عملیاتی' }; }
    else if (c.why === 'notown') { body = checks([['no', 'انتقال را سرپرست ارشد انجام داده است', p.op], ['info', 'رتبه، اجازه بازپس‌گیری نمی‌دهد', 'سابقه عامل انتقال با مقام بالاتر جایگزین نمی‌شود.']]); resp = { who: 'senior', text: 'سرپرست ارشد مسئول' }; }
    else if (c.why === 'mis') { body = checks([['no', 'منبع MIS است', 'برگشت یا آزادسازی منبع در دامنه MIS است، نه مدیر فروش'], ['info', 'شما مالک منابع وارد‌شده نیستید', '']]); resp = { who: 'mis', text: 'MIS' }; }
    else if (c.why === 'cancel') { body = checks([['no', 'فاکتور مرتبط لغو شده است', 'فاکتور ' + fa(c.invoice.code)], ['info', 'قانون آزادسازی تعیین نشده', 'برای پرونده با فاکتور لغوشده هنوز قانون آزادسازی رسمی وجود ندارد (OPD-03). لغو/رد به‌تنهایی آزادسازی نیست.']]); }
    else if (c.elig === 'conflict') { body = checks([['warn', 'وضعیت پرونده تغییر کرده است', c.why], ['info', 'پیش از هر اقدام بازخوانی کنید', 'اطلاعات نمایش‌داده‌شده ممکن است قدیمی باشد']]); }
    else { body = checks([['q', 'قابلیت برگشت قابل تأیید نیست', c.why], ['info', 'تا تطبیق، برگشت انجام نمی‌شود (fail-closed)', 'حدس با شماره موبایل انجام نمی‌شود']]); resp = { who: 'mis', text: 'MIS / مالی برای تطبیق' }; }
    var primary = c.elig === 'conflict' ? '<button type="button" class="btn btn-lg btn-primary" data-act="refresh-ret:' + c.id + '">' + ic('refresh') + 'بازخوانی پرونده</button>' : c.elig === 'ok' ? '<button type="button" class="btn btn-lg" disabled aria-disabled="true">' + ic('lock') + 'ثبت برگشت غیرفعال — مشروط به F01</button>' : '';
    return top('بررسی برگشت پرونده') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name" class="mono">' + esc(c.id) + '</h2>' + pill(e.label, e.tone, e.icon) + (c.legacy ? '<span class="tag legacy">سرنخ قدیمی</span>' : '') + '</div><div class="dr-meta"><span>' + esc(c.source) + '</span><span>نزد ' + esc(lvl) + ' ' + esc(holder) + '</span></div></div>' +
      '<div class="dr-body"><section class="sec">' + own({ custody: holder + ' (' + lvl + ')', original: c.source, next: resp ? resp.text : lvl + ' ' + holder, event: p.by === 'self' ? 'شما · ' + p.op : p.by === 'none' ? 'ثبت نشده' : p.op, credit: c.invoice ? 'طبق فاکتور ' + fa(c.invoice.code) + ' (قوانین مالی فعلی)' : 'هنوز فروشی ثبت نشده' }) + '</section>' +
      '<section class="sec' + (c.elig === 'ok' ? '' : ' primary') + '"><div class="sec-h"><h3>آیا این پرونده قابل برگشت است؟</h3></div>' + body + (resp ? '<div class="resp-line"><span>مسئول اقدام بعدی:</span>' + X.nextActor({ who: resp.who, text: resp.text }) + '</div>' : '') + '</section>' +
      (c.elig === 'ok' ? '<section class="sec"><div class="note warn">' + ic('lock') + '<span><b>پیش‌نمایش است، نه مجوز اجرا.</b> این بررسی ثبت را تضمین نمی‌کند و اجرای برگشت تا تکمیل حفاظت F01 (همه ارتباط‌های مالی، شکست در ابهام، بررسی مجدد هنگام ثبت) فعال نیست.</span></div></section>' : '') +
      X.details('تاریخچه', null, tl([['انتقال به ' + holder, (p.by === 'self' ? 'شما' : p.op) + ' · ' + p.when], ['ورود پرونده', c.source]])) + '</div>' + foot(primary, null, '');
  };
  D.returnReview = function () {
    var ids = keys(st.rsel).filter(retCase);
    return top('پیش‌نمایش برگشت گروهی') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">برگشت ' + fa(ids.length) + ' پرونده</h2>' + pill('مشروط · غیرفعال', 'amber', 'lock') + '</div><div class="dr-meta"><span>مقصد: پنل من</span></div></div><div class="dr-body">' +
      X.sec('پرونده‌ها (پیش‌نمایش)', '', '<div class="mini-list">' + ids.map(function (i) { var c = retCase(i); return '<div class="mini"><div class="grow"><span class="mono">' + i + '</span><div class="muted" style="font-size:var(--t-meta)">نزد ' + esc(X.holderName(c)) + ' · ' + esc(c.proof.op) + '</div></div>' + pill('واجد شرایط پیش‌نمایش', 'teal', 'checkCircle') + '</div>'; }).join('') + '</div>', 'primary') +
      '<section class="sec"><div class="note warn">' + ic('lock') + '<span>ثبت برگشت گروهی تا تکمیل حفاظت F01 (M-G01) فعال نیست. وقتی فعال شود: نتیجه به‌تفکیک پرونده (اعمال‌شده / ارسال‌نشده / ردشده / ناموفق / نامعلوم) نمایش داده می‌شود و ابهام «موفق» تلقی نمی‌شود.</span></div></section></div>' +
      foot('<button type="button" class="btn btn-lg" disabled aria-disabled="true">' + ic('lock') + 'ثبت برگشت — مشروط به F01</button>', null);
  };

  /* ================= HR Requests ================= */
  V.hr = function () {
    var f = st.hrForm || (st.hrForm = { seller: '', type: 'transfer', dest: '', reason: '' }), E = st.hrErr || {};
    var err = function (k) { return E[k] ? '<div class="field-err" id="hr-e-' + k + '">' + ic('alert') + '<span>' + esc(E[k]) + '</span></div>' : ''; };
    var inv = function (k) { return E[k] ? ' aria-invalid="true" aria-describedby="hr-e-' + k + '"' : ''; };
    var subj = M.sellers.filter(function (s) { return s.team !== 'direct'; });
    var form = '<section class="panel"><div class="sec-h"><h3>درخواست جدید</h3><span class="aside">فقط فروشندگان زیرمجموعه</span></div><div class="form-grid">' +
      '<label class="lbl">فروشنده <span class="req">*</span><select class="input" data-hr="seller"' + inv('seller') + '><option value="">انتخاب فروشنده</option>' + subj.map(function (s) { return '<option value="' + s.id + '"' + (String(f.seller) === String(s.id) ? ' selected' : '') + '>' + esc(s.name) + ' — ' + esc(X.teamName(s.team)) + (s.active ? '' : ' (غیرفعال)') + '</option>'; }).join('') + '</select>' + err('seller') + '</label>' +
      '<div class="lbl">نوع درخواست<span class="seg" role="group" aria-label="نوع درخواست"><button type="button" data-hrtype="transfer" aria-pressed="' + (f.type === 'transfer') + '">جابجایی به سرپرست دیگر</button><button type="button" data-hrtype="termination" aria-pressed="' + (f.type === 'termination') + '">قطع همکاری</button></span></div>' +
      (f.type === 'transfer' ? '<label class="lbl">سرپرست مقصد <span class="req">*</span><select class="input" data-hr="dest"' + inv('dest') + '><option value="">انتخاب سرپرست</option><optgroup label="تیم‌های محدوده شما">' + M.teams.filter(function (t) { return t.active; }).map(function (t) { return '<option value="' + esc(t.sup) + '"' + (f.dest === t.sup ? ' selected' : '') + '>' + esc(t.sup) + '</option>'; }).join('') + '</optgroup><optgroup label="خارج از محدوده شما — سیاست تعریف نشده"><option' + (f.dest === 'مجتبی نوری' ? ' selected' : '') + ' value="مجتبی نوری">مجتبی نوری</option></optgroup></select>' + (f.dest === 'مجتبی نوری' ? '<div class="note warn" style="margin-top:8px">' + ic('alert') + '<span>مقصد خارج از محدوده شما است. دیدن گزینه در فهرست مجوز انتقال خارج از محدوده نیست؛ تصمیم نهایی با منابع انسانی است.</span></div>' : '') + err('dest') + '</label>'
        : '<div class="note warn">' + ic('alert') + '<span>تحویل کار باز، فاکتور و مسئولیت فعال هنوز تعریف نشده است (OPD-05). ثبت درخواست به معنی قطع همکاری یا تحویل کار نیست.</span></div>') +
      '<label class="lbl">دلیل درخواست <span class="req">*</span><textarea class="input" data-hr="reason" placeholder="دلیل جابجایی یا قطع همکاری"' + inv('reason') + '>' + esc(f.reason) + '</textarea>' + err('reason') + '</label>' +
      '<div class="form-acts"><button type="button" class="btn btn-primary" data-act="hr-submit">ثبت درخواست</button><span class="muted" style="font-size:var(--t-meta)">مسیر: بررسی مرحله‌ای ← … ← اعمال نهایی توسط منابع انسانی</span></div></div></section>';
    var mine = M.hr.filter(function (r) { return r.state === 'pending_review' && r.reviewer === 'self'; }).length - Object.keys(st.hrDone).length;
    var QS = [{ id: 'review', label: 'منتظر بررسی شما', icon: 'hourglass', tone: 'orange', n: Math.max(0, mine), key: '1' }, { id: 'chain', label: 'در مسیر (فقط مشاهده)', icon: 'eye', tone: 'slate', n: M.hr.filter(function (r) { return r.state === 'pending_hr'; }).length + Object.keys(st.hrDone).filter(function (k) { return st.hrDone[k] === 'step'; }).length, key: '2' }, { id: 'closed', label: 'بسته‌شده', icon: 'history', tone: 'neutral', n: M.hr.filter(function (r) { return r.state === 'approved' || r.state === 'rejected' || r.state === 'failed'; }).length, key: '3' }];
    var inQ = function (r) { var done = st.hrDone[r.id]; return st.hq === 'review' ? r.state === 'pending_review' && r.reviewer === 'self' && !done : st.hq === 'chain' ? r.state === 'pending_hr' || done === 'step' : r.state === 'approved' || r.state === 'rejected' || r.state === 'failed' || done === 'rejected'; };
    var list = M.hr.filter(inQ);
    var cards = list.map(function (r) {
      var done = st.hrDone[r.id], s = done === 'rejected' ? X.HRS.rejected : done ? X.HRS.stepdone : X.HRS[r.state], sel = seller(r.seller);
      return '<button type="button" class="hr-card" data-act="open-hr:' + r.id + '"><div class="hr-top"><b>#' + fa(r.id) + ' · ' + (r.type === 'transfer' ? 'جابجایی' : 'قطع همکاری') + ' · ' + esc(sel.name) + (r.dest ? ' ← ' + esc(r.dest) : '') + '</b>' + pill(s.label, s.tone, s.icon) + '</div>' + X.chain(hrSteps(r)) + '</button>';
    }).join('');
    var lst = '<section class="panel"><div class="sec-h"><h3>درخواست‌های محدوده من</h3><span class="aside">بررسی گروهی ندارد؛ هر درخواست جدا بررسی می‌شود</span></div>' + h.queues(QS, st.hq, 'data-hq') + (list.length ? '<div class="hr-list">' + cards + '</div>' : h.stateBlock('empty', 'درخواستی در این صف نیست', 'صف دیگری را ببینید.', '')) + '</section>';
    return h.pageHead({ title: 'درخواست‌های HR', sub: 'بررسی شما یک مرحله از مسیر است · «تأییدشده در مرحله»، «در انتظار منابع انسانی»، «اعمال‌شده» و «اعمال ناموفق» چهار وضعیت جدا هستند' }) + '<div class="form-card">' + lst + form + '</div>';
  };
  function hrSteps(r) {
    var done = st.hrDone[r.id]; if (!done) return r.steps;
    if (done === 'rejected') return [r.steps[0], ['مدیر فروش', 'rejected', M.user.name + ' · رد با دلیل']];
    return [r.steps[0], ['مدیر فروش', 'done', M.user.name + ' · تأیید این مرحله'], ['مرحله بعد طبق زنجیره', 'cur', 'در انتظار بررسی‌کننده بعدی'], r.steps[r.steps.length - 1]];
  }
  D.hr = function (id, keep) {
    var r = M.hr.filter(function (x) { return String(x.id) === String(id); })[0], done = st.hrDone[r.id], s = done === 'rejected' ? X.HRS.rejected : done ? X.HRS.stepdone : X.HRS[r.state];
    var canReview = r.state === 'pending_review' && r.reviewer === 'self' && !done, noteErr = keep && keep.err;
    var review = canReview ? X.sec('بررسی مرحله شما', '', '<div class="note info">' + ic('shield') + '<span><b>تأیید شما فقط همین مرحله را جلو می‌برد.</b> جابجایی یا قطع همکاری اعمال نمی‌شود؛ اعمال نهایی را منابع انسانی انجام می‌دهد. دسترسی، نقش و رمز کارمند اینجا قابل تغییر نیست.</span></div>' +
      '<label class="lbl" style="margin-top:12px">یادداشت یا دلیل رد' + (noteErr ? ' <span class="req">* الزامی برای رد</span>' : '') + '<textarea class="input" data-hrnote placeholder="برای رد کردن، نوشتن دلیل الزامی است"' + (noteErr ? ' aria-invalid="true"' : '') + '>' + esc((keep && keep.note) || '') + '</textarea>' + (noteErr ? '<div class="field-err">' + ic('alert') + '<span>برای رد درخواست دلیل بنویسید.</span></div>' : '') + '</label>', 'primary') : '';
    var info = !canReview ? X.sec('وضعیت', '', checks([r.state === 'failed' ? ['no', 'اعمال نهایی ناموفق بود', 'تأیید مرحله‌ای شما انجام شده، اما تغییری اعمال نشده است؛ ' + (r.failedAt || '') + '. پیگیری با منابع انسانی است.'] : r.state === 'pending_hr' || done === 'step' ? ['info', 'مرحله‌ی شما انجام شده است', 'اقدام فعلی با ' + (done ? 'بررسی‌کننده بعدی' : 'منابع انسانی') + ' است؛ از سمت شما اقدامی لازم نیست'] : ['info', 'این درخواست بسته شده است', r.applied ? 'اعمال‌شده · ' + r.applied : 'مرحله‌ای ادامه نمی‌یابد'], ['info', '«تأیید این مرحله» با «اعمال نهایی» فرق دارد', 'تغییر واقعی فقط پس از اعمال نهایی موفق ثبت می‌شود']])) : '';
    var handover = r.type === 'termination' ? X.sec('تحویل کار (تعریف نشده)', 'OPD-05', checks([['q', 'پرونده‌های باز، فاکتورها و مسئولیت‌های فعال', 'تعیین نشده است؛ غیرفعال‌شدن پروفایل به‌تنهایی تحویل کامل کار نیست'], ['info', 'انتساب تاریخی بدون تغییر می‌ماند', 'درخواستی که پیش از بررسی تغییر کرده باشد، تعارض است و نیازمند بازخوانی تازه است']])) : '';
    return top('درخواست HR #' + fa(r.id)) + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">' + (r.type === 'transfer' ? 'جابجایی' : 'قطع همکاری') + ' · ' + esc(seller(r.seller).name) + '</h2>' + pill(s.label, s.tone, s.icon) + '</div><div class="dr-meta">' + (r.dest ? '<span>مقصد: ' + esc(r.dest) + '</span>' : '') + '<span>درخواست‌دهنده: ' + esc(r.requester) + '</span><span>تیم فعلی: ' + esc(X.teamName(seller(r.seller).team)) + '</span></div></div>' +
      '<div class="dr-body"><section class="sec primary"><div class="sec-h"><h3>مسیر تأیید</h3></div>' + X.chain(hrSteps(r)) + '</section><section class="sec"><div class="sec-h"><h3>دلیل</h3></div><p style="margin:0">' + esc(r.reason) + '</p></section>' + review + info + handover + '</div>' +
      foot(canReview ? '<button type="button" class="btn btn-lg btn-primary" data-act="hr-step:' + r.id + '">' + ic('check') + 'تأیید این مرحله و ارجاع به مرحله بعد</button><button type="button" class="btn btn-lg btn-danger" data-act="hr-reject:' + r.id + '">رد با دلیل</button>' : '', null);
  };

  /* ================= Detail drawers ================= */
  D.unit = function (uid) {
    var u = unit(uid), ts = X.teamsOf(uid), f = X.unit_f(uid, 'current'), cov = X.unitCov(uid);
    var list = ts.map(function (t) { return '<button type="button" class="mini as-btn" data-act="open-team:' + t.id + '"><div class="grow"><b>' + esc(t.sup) + '</b><div class="muted" style="font-size:var(--t-meta)">پرونده نزد تیم ' + fa(X.teamWorkload(t.id)) + ' · فروشنده ' + fa(X.sellersOf(t.id).length) + '</div></div>' + (t.active ? pill('فعال', 'teal', 'checkCircle') : pill('غیرفعال', 'slate', 'ban')) + X.rel('indirect') + '</button>'; }).join('');
    return top('واحد سرپرست ارشد') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">' + esc(u.name) + '</h2>' + (u.active ? pill('فعال', 'teal', 'checkCircle') : pill('غیرفعال', 'slate', 'ban')) + X.rel('senior') + '</div><div class="dr-meta"><span>' + fa(ts.length) + ' سرپرست</span>' + (u.active ? '' : '<span>' + esc(u.inactiveNote) + '</span>') + '<span>' + X.cov(cov) + '</span></div></div>' +
      '<div class="dr-body">' + (u.active ? '' : '<section class="sec"><div class="note warn">' + ic('alert') + '<span>این سرپرست ارشد غیرفعال است. گیرنده تخصیص نیست و مالک جدید خودکار تعیین نمی‌شود؛ مسئول جایگزین باید از طریق ساختار/منابع انسانی مشخص شود.</span></div></section>') +
      '<section class="sec primary"><div class="sec-h"><h3>بار فعلی واحد</h3>' + X.basis('current') + '</div><div class="sum-grid four"><div class="sum"><span>پرونده نزد واحد</span><b>' + fa(X.unitWorkload(uid)) + '</b></div><div class="sum"><span>فاکتور صادرشده</span><b>' + fa(f.created) + '</b></div><div class="sum"><span>پیش‌فاکتور باز</span><b>' + fa(f.openPre) + '</b></div><div class="sum"><span>فروش تکمیل‌شده</span><b>' + fa(f.completed) + '</b></div></div></section>' +
      X.details('سرپرستان این واحد', ts.length, ts.length ? '<div class="mini-list">' + list + '</div><p class="ind-note">غیرمستقیم برای شما: مشاهده و گزارش. دیدن یک سرپرست به معنی اجازه نوشتن برای او نیست.</p>' : '<p class="ind-note">زیرمجموعه‌ای ثبت نشده است.</p>', true) + '</div>' +
      foot(u.active ? '<button type="button" class="btn btn-lg btn-primary" data-act="assign-to:' + uid + '">' + ic('send') + 'تخصیص به این سرپرست ارشد</button>' : '', '<button type="button" class="btn btn-lg" data-act="drill:' + uid + '">' + ic('chart') + 'عملکرد این واحد</button>');
  };
  D.team = function (tid) {
    var t = team(tid), ss = X.sellersOf(tid), f = X.team_f(tid, 'current'), cov = X.teamCov(tid), direct = t.parent === 'M';
    var list = ss.map(function (s) { return '<button type="button" class="mini as-btn" data-act="open-seller:' + s.id + '"><div class="grow"><b>' + esc(s.name) + '</b><div class="muted" style="font-size:var(--t-meta)">پرونده باز ' + fa(s.open) + (s.moved ? ' · منتقل‌شده از ' + esc(X.teamName(s.moved.from)) : '') + '</div></div>' + (s.active ? pill('فعال', 'teal', 'checkCircle') : pill('غیرفعال', 'slate', 'ban')) + X.rel('indirect') + '</button>'; }).join('');
    return top('سرپرست و تیم') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">' + esc(t.sup) + '</h2>' + (t.active ? pill('فعال', 'teal', 'checkCircle') : pill('غیرفعال', 'slate', 'ban')) + X.rel(direct ? 'dsup' : 'sup') + '</div><div class="dr-meta"><span>' + fa(ss.length) + ' فروشنده</span><span>' + (direct ? 'مستقیم زیر نظر شما' : 'سرپرست ارشد ' + esc(unit(t.parent).name)) + '</span>' + (t.active ? '' : '<span>' + esc(t.inactiveNote) + '</span>') + '<span>' + X.cov(cov) + '</span></div></div>' +
      '<div class="dr-body">' + (t.active ? '' : '<section class="sec"><div class="note warn">' + ic('alert') + '<span>سرپرست این تیم غیرفعال است. گیرنده تخصیص نیست و مالک جدید خودکار تعیین نمی‌شود.</span></div></section>') +
      '<section class="sec primary"><div class="sec-h"><h3>بار فعلی تیم</h3>' + X.basis('current') + '</div><div class="sum-grid four"><div class="sum"><span>پرونده نزد تیم</span><b>' + fa(X.teamWorkload(tid)) + '</b></div><div class="sum"><span>فاکتور صادرشده</span><b>' + fa(f.created) + '</b></div><div class="sum"><span>پیش‌فاکتور باز</span><b>' + fa(f.openPre) + '</b></div><div class="sum"><span>فروش تکمیل‌شده</span><b>' + fa(f.completed) + '</b></div></div></section>' +
      X.details('فروشندگان این تیم', ss.length, '<div class="mini-list">' + list + '</div><p class="ind-note">دیدن یک فروشنده به معنی اجازه نوشتن برای او نیست.</p>', true) + '</div>' +
      foot(t.active ? '<button type="button" class="btn btn-lg btn-primary" data-act="assign-to:' + tid + '">' + ic('send') + (direct ? 'تخصیص به این سرپرست' : 'تخصیص (عبور از سرپرست ارشد)') + '</button>' : '', '<button type="button" class="btn btn-lg" data-act="drill:' + tid + '">' + ic('chart') + 'عملکرد این تیم</button>');
  };
  D.seller = function (id) {
    var s = seller(Number(id)), f = X.seller_f(s.id), direct = s.team === 'direct';
    return top(direct ? 'فروشنده مستقیم' : 'فروشنده زیرمجموعه') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">' + esc(s.name) + '</h2>' + (s.active ? pill('فعال', 'teal', 'checkCircle') : pill('غیرفعال', 'slate', 'ban')) + (direct ? X.rel('dseller') : X.rel('indirect')) + '</div><div class="dr-meta"><span>' + esc(X.teamName(s.team)) + '</span></div></div>' +
      '<div class="dr-body">' + (!direct ? '<section class="sec"><div class="note info">' + ic('eye') + '<span>این فروشنده زیر سرپرست است. شما مشاهده و گزارش می‌بینید؛ تماس و فروش از مسئولیت خود او می‌ماند و اجازه تخصیص ' + (s.alloc === 'ok' ? 'فقط در مسیر مشروط عبور از سطح' : 'برای او اثبات نشده') + ' است.</span></div></section>' : '') +
      (s.moved ? '<section class="sec"><div class="note conflict">' + ic('swap') + '<span>در ' + esc(s.moved.when) + ' از ' + esc(X.teamName(s.moved.from)) + ' منتقل شده است. فروش‌های قبل از انتقال به تیم هنگام صدور نسبت داده می‌شوند، نه تیم امروز.</span></div></section>' : '') +
      '<section class="sec primary"><div class="sec-h"><h3>خلاصه</h3>' + X.basis('event') + '</div><div class="sum-grid four"><div class="sum"><span>پرونده باز</span><b>' + fa(s.open) + '</b></div><div class="sum"><span>فاکتور صادرشده</span><b>' + fa(f.created) + '</b></div><div class="sum"><span>پیش‌فاکتور باز</span><b>' + fa(f.openPre) + '</b></div><div class="sum"><span>فروش تکمیل‌شده</span><b>' + fa(f.completed) + '</b></div></div></section>' +
      X.details('فاکتورها', f.list.length, '<div class="mini-list">' + (f.list.length ? f.list.map(function (i) { return '<button type="button" class="mini as-btn" data-act="open-inv:' + i.code + '"><div class="grow"><span class="mono">' + fa(i.code) + '</span><div class="muted" style="font-size:var(--t-meta)">' + esc(i.customer) + '</div></div>' + pill(X.INVS[i.inv].label, X.INVS[i.inv].tone, X.INVS[i.inv].icon) + '</button>'; }).join('') : '<div class="muted">فاکتوری ثبت نشده است.</div>') + '</div>', f.list.length > 0) + '</div>' +
      foot(!direct ? '<button type="button" class="btn btn-lg btn-primary" data-act="hr-for:' + s.id + '">' + ic('briefcase') + 'درخواست HR برای این فروشنده</button>' : '', null);
  };
  D.attn = function (id) {
    var a = X.attention().filter(function (x) { return x.id === id; })[0]; if (!a) return top('استثنا') + '<div class="dr-body">' + h.stateBlock('empty', 'موردی نیست', '') + '</div>';
    var list = function (arr, ic_, cls) { return '<ul class="elig">' + arr.map(function (t) { return '<li class="' + cls + '">' + ic(ic_) + '<span><b>' + esc(t) + '</b></span></li>'; }).join('') + '</ul>'; };
    return top('استثنای نیازمند توجه') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">' + esc(a.label) + '</h2>' + pill(fa(a.n) + ' مورد', a.tone, a.icon) + '</div></div><div class="dr-body">' +
      '<section class="sec primary"><dl class="exc-dl"><div><dt>موضوع</dt><dd>' + esc(a.subject || '—') + '</dd></div><div><dt>مالک / مسئول</dt><dd>' + esc(a.owner) + '</dd></div><div><dt>وضعیت</dt><dd>' + esc(a.state) + '</dd></div><div><dt>مسئول گام بعدی (Next Actor)</dt><dd>' + esc(a.next) + '</dd></div><div><dt>علامت حل (Resolution signal)</dt><dd>' + esc(a.signal) + '</dd></div></dl></section>' +
      X.sec('مسئولیت و اقدام مجاز شما', '', list(a.may, 'checkCircle', 'e-ok')) + X.sec('ممنوع در این پنل', '', list(a.mayNot, 'xCircle', 'e-no')) +
      X.sec('ارجاع رسمی', '', checks([['info', 'دستور ارجاع، اعلان یا SLA جدیدی تعریف نشده است', '«ارجاع» یعنی مشخص‌کردن مسئول موجود؛ مسئول جدید ساخته یا اعلان ارسال نمی‌شود (M-G11). حل‌شدن به معنی آزادشدن یا قابل‌تخصیص‌شدن پرونده نیست.']])) + '</div>' +
      foot('<button type="button" class="btn btn-lg btn-primary" data-act="attn-go:' + a.id + '">' + ic('arrowL') + 'رفتن به صفحه مرتبط</button>', null);
  };
  D.metrics = function () {
    var rows = M.kpiDefs.map(function (k) { return '<tr><td><b>' + esc(k.name) + '</b></td><td>' + esc(k.grain) + '</td><td>' + esc(k.source) + '</td><td>' + esc(k.cohort) + '</td><td>' + esc(k.time) + '</td><td>' + esc(k.scope) + '</td><td>' + esc(k.formula) + '</td><td>' + esc(k.fresh) + '</td><td>' + X.cov('undef', 'باز: ' + k.open) + '</td></tr>'; }).join('');
    return top('تعریف شاخص‌ها') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">متادیتای هر شاخص</h2></div><div class="dr-meta"><span>نام · واحد شمارش · منبع · گروه شمارش · مبنای زمانی · محدوده · فرمول · تازگی</span></div></div><div class="dr-body">' +
      '<section class="sec primary"><div class="tbl-wrap"><table class="tbl no-cursor stackable" aria-label="تعریف شاخص‌ها"><thead><tr><th>نام</th><th>واحد شمارش</th><th>منبع</th><th>گروه شمارش</th><th>مبنای زمانی</th><th>محدوده</th><th>فرمول/شرط</th><th>تازگی</th><th>بخش باز</th></tr></thead><tbody>' + rows + '</tbody></table></div></section>' +
      X.sec('توجه', '', checks([['info', 'نرخ، رتبه، هدف، امتیاز بازدهی و SLA تعریف نشده‌اند', 'هیچ‌کدام در این پنل نمایش داده نمی‌شود'], ['info', 'cohort تجاری باز است', 'تعریف شاخص قرارداد هدف است؛ اجرای فنی resolver مشترک هنوز نهایی نیست (NOT FINAL).'], ['info', 'داده نامعلوم/دریافت‌نشده هرگز صفر نیست', '']])) + '</div>' + foot('', null);
  };

  /* ---------- Invoice, archive, extra-number drawers ---------- */
  function facets(i) {
    var s = X.INVS[i.inv], rv = X.REV[i.review], done = i.inv === 'completed' && i.review === 'approved';
    var f = [['وضعیت فاکتور', pill(s.label, s.tone, s.icon)], ['مرحله پرداخت', '<b>مرحله ' + fa(i.stage) + ' از ' + fa(i.stages) + '</b><span>' + esc(X.STG[i.stg]) + '</span>'], ['مدرک پرداخت', '<b>' + esc(X.EVID[i.evidence]) + '</b>' + (i.evidence === 'receipt' ? '<span>ثبت رسید ≠ تأیید پرداخت</span>' : '')], ['بررسی مالی', rv ? pill(rv.label, rv.tone, rv.icon) : '<b class="muted">ارسال نشده</b>'], ['فروش تکمیل‌شده', done ? pill('بله', 'green', 'checkCircle') : '<b class="muted">خیر</b>'], ['اقدام بعدی با', i.next ? X.nextActor(i.next, i) : '<b class="muted">—</b>']];
    return '<div class="facets">' + f.map(function (x) { return '<div class="facet"><span class="f-l">' + x[0] + '</span><div class="f-v">' + x[1] + '</div></div>'; }).join('') + '</div>';
  }
  D.inv = function (code) {
    var i = X.invOf(code), s = seller(i.seller), tNow = s.team, canLink = i.pay === 'link' && i.inv !== 'completed' && i.inv !== 'cancelled', sent = st.resent[i.code];
    var policy = [
      ['ok', 'خواندن', 'فهرست، جزئیات، مراحل و سوابق مالی', 'مجاز در محدوده شما'],
      ['ok', 'کپی لینک پرداخت', 'کمک: خودش تغییر مالی نیست؛ فقط لینک همین فاکتور مجاز', canLink ? 'مجاز · قابل مشاهده در نمونه' : 'برای این وضعیت لینکی نیست'],
      ['pending', 'ارسال مجدد لینک پرداخت', 'کمک + اثر ارسال: هر ارسال مجوز و وضعیت مستقل دارد؛ نتیجه ارسال تأیید نشده', canLink ? 'کنترل دیده شد · نتیجه تأیید نشده' : 'برای این وضعیت لینکی نیست'],
      ['pending', 'ثبت رسید / ویرایش پیش‌پرداخت یا مرحله', 'در نمونه مدیر دیده نشد و از سرپرست ارشد کپی نمی‌شود', 'نیازمند اعتبارسنجی (OPD-10)'],
      ['locked', 'تأیید / رد / بازپرداخت / ثبت مالی / پورسانت', 'مسئول: واحد مالی؛ رتبه و دیدن فاکتور مجوز نیست', 'در این پنل ممکن نیست'],
      ['locked', 'لغو یا اتصال مجدد فاکتور', 'تغییر روابط مالی بدون اثبات ممنوع است', 'محدود / نیازمند اثبات']
    ];
    var acts = '<ul class="act-policy">' + policy.map(function (a) { return '<li class="ap-' + a[0] + '"><span class="ap-ic">' + ic(a[0] === 'ok' ? 'checkCircle' : a[0] === 'pending' ? 'question' : 'lock') + '</span><span class="ap-t"><b>' + a[1] + '</b><span>' + a[2] + '</span></span><span class="ap-s">' + a[3] + '</span></li>'; }).join('') + '</ul>';
    var waitTxt = i.review === 'pending' ? 'منتظر بررسی مالی — اقدامی از سمت شما لازم نیست' : i.inv === 'mismatch' ? 'منتظر تطبیق — مسئول: MIS / مالی' : i.next && i.next.who === 'seller' ? 'اقدام بعدی با ' + s.name : '';
    var assist = canLink ? '<button type="button" class="btn btn-lg" data-act="copy-link:' + i.code + '">' + ic('copy') + 'کپی لینک پرداخت</button><button type="button" class="btn btn-lg btn-soft" data-act="resend:' + i.code + '"' + (sent ? ' disabled aria-disabled="true"' : '') + '>' + ic('send') + (sent ? 'درخواست ارسال ثبت شد' : 'ارسال مجدد لینک') + '</button>' : '';
    return top('فاکتور') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name" class="mono">' + fa(i.code) + '</h2>' + pill(X.INVS[i.inv].label, X.INVS[i.inv].tone, X.INVS[i.inv].icon) + X.link(i.link) + '</div><div class="dr-meta"><span style="color:var(--text)">' + esc(i.customer) + '</span><span>فروشنده: ' + esc(s.name) + '</span><span>تیم فعلی: ' + esc(X.teamName(tNow)) + '</span><span>صدور: ' + esc(i.issued) + '</span></div></div>' +
      '<div class="dr-body"><section class="sec primary"><div class="sum-grid"><div class="sum"><span>مبلغ کل</span><b>' + money(i.total) + '</b></div><div class="sum"><span>پرداخت‌شده</span><b style="color:var(--green-fg)">' + money(i.paid) + '</b></div><div class="sum"><span>مانده</span><b>' + money(i.inv === 'cancelled' ? 0 : i.total - i.paid) + '</b></div></div>' +
      (i.reason ? '<div class="note danger" style="margin-top:12px">' + ic('alert') + '<span>دلیل رد مالی: ' + esc(i.reason) + ' · اقدام بعدی با ' + esc(s.name) + '</span></div>' : '') + (i.mismatch ? '<div class="note warn" style="margin-top:12px">' + ic('question') + '<span>' + esc(i.mismatch) + '. تا تطبیق، وضعیت «تکمیل‌شده» قطعی اعلام نمی‌شود.</span></div>' : '') + (i.linkNote ? '<div class="note warn" style="margin-top:12px">' + ic('layers') + '<span>' + esc(i.linkNote) + '</span></div>' : '') + (sent ? '<div class="note info" style="margin-top:12px">' + ic('send') + '<span>درخواست ارسال لینک ثبت شد؛ <b>نتیجه تحویل تأیید نشده است</b> و موفقیت ارسال فرض نمی‌شود.</span></div>' : '') + '</section>' +
      X.sec('وضعیت مالی به تفکیک', '', facets(i)) +
      X.sec('مالکیت و انتساب', '', own({ custody: s.name, original: s.name, next: i.next ? i.next.text : '—', event: s.name + ' (صدور · ' + i.issued + ')', credit: 'مالک اعتبار: ' + s.name + ' (طبق قوانین مالی فعلی)' }) + '<div class="basis-pair"><span>تیم فعلی: <b>' + esc(X.teamName(tNow)) + '</b> ' + X.basis('current') + '</span><span>تیم هنگام صدور: <b>' + (i.teamAtIssue == null ? 'ناشناخته' : esc(X.teamName(i.teamAtIssue))) + '</b> ' + X.basis('hist') + '</span></div>') +
      X.details('اقدام‌های این پنل (سیاست هر اقدام جدا)', null, acts + '<div class="note info" style="margin-top:12px">' + ic('shield') + '<span>مجوز «کمک» از نقش‌های دیگر به ارث نمی‌رسد و نمایش یک کنترل به معنی اجرای موفق یا مجوز نیست.</span></div>', true) +
      X.details('سوابق مالی', null, tl((i.review === 'rejected' ? [['رد توسط مالی: ' + i.reason, 'کارشناس مالی · ۱۴۰۵/۰۷/۰۵'], ['ثبت رسید مرحله ۱', s.name + ' · ۱۴۰۵/۰۷/۰۴']] : i.review === 'pending' ? [['ثبت رسید', s.name + ' · ۱۴۰۵/۰۷/۰۶']] : []).concat([['صدور ' + (i.inv === 'pre' ? 'پیش‌فاکتور' : 'فاکتور'), s.name + ' · ' + i.issued]]))) +
      '<section class="sec"><button type="button" class="linkish" data-act="goto-cust">' + ic('activity') + 'رفتار مشتری در زمینه این فاکتور</button></section></div>' +
      foot(assist || (waitTxt ? '<span class="wait-note">' + ic('hourglass') + esc(waitTxt) + '</span>' : ''), null);
  };
  D.arch = function (id) {
    var a = M.archives.filter(function (x) { return x.id === id; })[0], s = seller(a.seller), rc = X.ARC[a.reason], R = M.archReasons[a.reason];
    var disabled = [['revive', 'احیا / بازگرداندن', 'اختیار خاص هر علت اثبات نشده؛ بازگردانی عمومی تعریف نشده'], ['reassign', 'تخصیص مجدد', 'نیازمند اعتبارسنجی'], ['bulk', 'حل گروهی', 'مشروط به سیاست و بررسی مالی']];
    return top('بررسی آرشیو') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name" class="mono">' + esc(a.caseRef) + '</h2>' + pill(rc[0], rc[1], rc[2]) + X.src(a.src) + X.link(a.link) + '</div><div class="dr-meta"><span>شناسه آرشیو ' + a.id + '</span><span>فروشنده: ' + esc(s.name) + '</span><span>بایگانی: ' + esc(a.archived) + ' · ' + fa(a.age) + ' روز</span></div></div>' +
      '<div class="dr-body"><section class="sec primary"><div class="note info">' + ic('info') + '<span><b>قاعده این علت:</b> ' + esc(R.rule) + (a.attempts ? ' (تلاش‌ها: ' + fa(a.attempts) + ' · آخرین فعالیت: ' + esc(a.lastAct) + ')' : '') + (a.due ? ' · سررسید: ' + esc(a.due) : '') + '</span></div>' +
      (a.fin ? '<div class="sum-grid" style="margin-top:12px"><div class="sum"><span>مبلغ کل</span><b>' + money(a.fin.total) + '</b></div><div class="sum"><span>پرداخت‌شده</span><b style="color:var(--green-fg)">' + money(a.fin.paid) + '</b></div><div class="sum"><span>مانده</span><b>' + money(a.fin.total - a.fin.paid) + '</b></div></div>' : '<div class="muted" style="margin-top:12px">وابستگی مالی ثبت‌شده‌ای در این علت وجود ندارد.</div>') + (a.note ? '<div class="note warn" style="margin-top:12px">' + ic('question') + '<span>' + esc(a.note) + '</span></div>' : '') + '</section>' +
      X.sec('مالکیت', '', own({ custody: 'نامشخص در این نما', original: s.name, next: 'actor فروش/پرداخت فعلی (در این نما مشخص نیست)', event: 'رویداد بایگانی · ' + a.archived, credit: a.fin ? 'طبق فاکتور ' + fa(a.fin.inv) + ' (قوانین مالی فعلی)' : 'هنوز فروشی ثبت نشده' }) + '<div class="note warn" style="margin-top:12px">' + ic('question') + '<span>مالک نمایش‌داده‌شده (' + esc(a.owner[0]) + ': ' + esc(a.owner[1]) + ') طبق اولویت مبدل ← سرپرست ← فروشنده انتخاب شده و خودکار «مسئول فعلی» یا «مالک اعتبار» نیست.</span></div>') +
      X.sec('مسیر حل', '', checks([a.resolve === 'late-payment' ? ['ok', 'پرداخت دیرهنگام طبق قاعده موجود می‌تواند آن را حل‌شده کند', 'اجرای واقعی تأیید نشده؛ تأیید رسید یا کلیک مدیر احیای عمومی نیست'] : a.resolve === 'partial-pay' ? ['q', 'پرداخت جزئی به‌تنهایی احیا یا آزادسازی نیست', 'مانده پرداخت‌نشده است'] : ['q', 'قابلیت احیا قابل اثبات نیست', 'مسیر حل مختص علت بی‌پاسخ تعریف نشده']])) +
      X.details('اقدام‌های این پنل', null, '<ul class="act-policy"><li class="ap-ok"><span class="ap-ic">' + ic('checkCircle') + '</span><span class="ap-t"><b>مشاهده و بررسی عملیاتی</b><span>علت، سن، مالک نمایش‌داده‌شده و وابستگی مالی</span></span><span class="ap-s">مجاز</span></li>' +
        disabled.map(function (d) { return '<li class="ap-pending"><span class="ap-ic">' + ic('question') + '</span><span class="ap-t"><b>' + d[1] + '</b><span>' + d[2] + '</span></span><span class="ap-s">نیازمند اعتبارسنجی / معوق</span></li>'; }).join('') + '<li class="ap-locked"><span class="ap-ic">' + ic('lock') + '</span><span class="ap-t"><b>حذف</b><span>آرشیو حذف نیست و سبب آزادشدن وابستگی مالی نمی‌شود</span></span><span class="ap-s">مجاز نیست</span></li></ul>', true) + '</div>' + foot('', null);
  };
  D.xreq = function (id, keep) {
    var r = M.xreq.filter(function (x) { return x.id === id; })[0], s = seller(r.seller), d = st.xDone[r.id] || r.state, S = X.XRS[d], open = d === 'pending' || d === 'unknown', confirmed = keep && keep.confirm, err = st.xErr[r.id];
    var review = open ? X.sec('تصمیم شما', '', (err ? '<div class="note danger" role="alert" style="margin-bottom:12px">' + ic('xCircle') + '<span><b>ثبت انجام نشد.</b> ' + esc(err) + ' درخواست همچنان «در انتظار تصمیم» است؛ هیچ سرنخی ساخته‌شده فرض نمی‌شود.</span></div>' : '') + (d === 'unknown' ? '<div class="note warn" style="margin-bottom:12px">' + ic('question') + '<span><b>نتیجه ثبت نامعلوم است.</b> پاسخ نرسید و ممکن است تصمیم ثبت شده باشد. پیش از تکرار، وضعیت را بازخوانی/تطبیق کنید.</span></div>' : '') +
      '<div class="note info">' + ic('shield') + '<span><b>اثر تأیید (کد فعلی):</b> یک <b>سرنخ قدیمی (legacy)</b> برای ' + esc(s.name) + ' ساخته می‌شود (بدون سرپرست، وضعیت «تخصیص‌یافته»، منبع «approved_extra_number_request») و سپس درخواست «تأییدشده» می‌شود. معادل پرونده V4 نیست و نگاشت به CaseRef ' + X.link('unresolved') + ' می‌ماند. اگر ساخت یا ثبت نتیجه شکست بخورد، درخواست در انتظار می‌ماند.</span></div>' +
      '<label class="lbl" style="margin-top:12px">دلیل رد (توصیه‌شده برای ردیابی)<textarea class="input" data-xnote placeholder="در رفتار فعلی نوشتن دلیل الزامی نیست؛ برای ردیابی بهتر پیشنهاد می‌شود">' + esc((keep && keep.note) || '') + '</textarea><span class="muted" style="font-size:var(--t-micro)">الزامی‌کردن این فیلد تغییر آینده است و نیازمند QA و بازگشت‌پذیری است.</span></label>' +
      '<label class="confirm"><input type="checkbox" class="cbx" data-confirm' + (confirmed ? ' checked' : '') + '><span>تأیید می‌کنم برای تأیید، سرنخ قدیمی جدید برای <b>' + esc(s.name) + '</b> ساخته شود.</span></label>', 'primary') : '';
    return top('درخواست شماره اضافه') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name" class="mono">' + r.id + '</h2>' + pill(S.label, S.tone, S.icon) + '</div><div class="dr-meta"><span>درخواست‌دهنده: ' + esc(s.name) + '</span><span>ثبت: ' + esc(r.at) + '</span><span>مدیر تعیین‌شده: ' + esc(M.user.name) + '</span></div></div>' +
      '<div class="dr-body"><section class="sec' + (open ? '' : ' primary') + '"><dl class="exc-dl"><div><dt>شماره نرمال‌شده (فقط جستجو؛ هویت پرونده نیست)</dt><dd class="mono">' + fa(r.phone) + '</dd></div><div><dt>نام / یادداشت</dt><dd>' + (r.name ? esc(r.name) + (r.note ? ' · ' + esc(r.note) : '') : '<span class="muted">ثبت نشده</span>') + '</dd></div><div><dt>بررسی تکراری</dt><dd>تکراری نیست — فقط برای همین فروشنده در درخواست‌های pending/approved؛ بین فروشندگان و منابع دیگر بررسی نشده است</dd></div>' + (d === 'approved' ? '<div><dt>سرنخ ساخته‌شده</dt><dd><span class="mono">' + esc(r.lead || 'L-نمونه') + '</span> · سرنخ قدیمی · منبع «approved_extra_number_request»</dd></div>' : '') + (d === 'rejected' ? '<div><dt>دلیل رد</dt><dd>' + (r.reason ? esc(r.reason) : '<span class="muted">دلیل ثبت نشده</span>') + '</dd></div>' : '') + (r.by ? '<div><dt>تصمیم‌گیرنده</dt><dd>' + esc(r.by) + '</dd></div>' : '') + '</dl>' + (r.flag ? '<div class="note warn" style="margin-top:12px">' + ic('question') + '<span>' + esc(r.flag) + '</span></div>' : '') + '</section>' + review +
      X.details('موارد نیازمند اعتبارسنجی', null, checks([['q', 'فروشنده پس از درخواست غیرفعال یا جابه‌جا شده باشد', 'سیاست جدید بی‌صدا اجرا نمی‌شود'], ['q', 'مدیر تعیین‌شده قدیمی یا مدیر بدون مسیر', 'ثبت می‌شود و مسیر Admin مطرح است؛ مدیر خودکار تصاحب نمی‌کند'], ['q', 'تکراری بین فروشندگان / منابع و رقابت هم‌زمان', ''], ['q', 'کامل بودن تاریخچه (فقط آخرین ۱۰۰ ردیف)', '']])) + '</div>' +
      foot(open ? '<button type="button" class="btn btn-lg btn-primary" data-act="x-approve:' + r.id + '"' + (confirmed ? '' : ' disabled aria-disabled="true"') + '>' + ic('check') + 'تأیید و ساخت سرنخ</button><button type="button" class="btn btn-lg btn-danger" data-act="x-reject:' + r.id + '">رد درخواست</button>' + (d === 'unknown' ? '<button type="button" class="btn btn-lg" data-act="x-reconcile:' + r.id + '">' + ic('refresh') + 'تطبیق وضعیت</button>' : '') : '', null, open ? 'تصمیم هر درخواست جداگانه ثبت می‌شود' : '');
  };
})();
