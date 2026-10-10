/* Sales Deputy — role layer, part 3: conditional direct allocation & eligible-return preview, HR assigned-step context, operation results,
   and detail drawers (branch/hierarchy, owner-aware exception, metrics, scoped invoice).
   Direct allocation is CONDITIONAL / NEEDS POLICY (SD-G05): the Manager level is the ordinary path; every skip-level path is exceptional, traceable and
   keeps the current technical route (no approval chain invented). Return is previewed only: activation depends on Gate 0 F01 (SD-G01).
   HR: the Deputy's step is not the final apply. Nothing here writes anywhere. */
(function () {
  'use strict';
  var X = window.DEPX, C = X.C, M = X.M, h = X.h, st = X.st, V = X.V, D = X.D;
  var esc = h.esc, fa = h.fa, num = h.num, money = h.money, ic = h.ic, pill = h.pill, hint = h.hint, cbx = h.cbx;
  var seller = X.seller, team = X.team, senior = X.senior, keys = X.keys, top = X.top, foot = X.foot, own = X.own, checks = X.checks, tl = X.tl;

  /* ================= Conditional Direct Allocation & Eligible Return ================= */
  V.alloc = function () {
    var pool = X.poolList(), ok = X.poolOk().length, review = pool.length - ok, issue = M.ops.filter(function (o) { var c = X.counts(o); return c.failed || c.skipped || c.rejected || c.unknown; }).length;
    var kpis = [
      { label: 'موجودی شخصی من', value: fa(pool.length), color: 'neutral', keep: true, meaning: 'پرونده‌هایی که اثبات شده نزد شما هستند؛ صفر بودن موجودی ثبت دریافت را اثبات یا رد نمی‌کند', basis: 'وضعیت در لحظه' },
      { label: 'قابل تخصیص', value: fa(ok), color: 'teal', keep: true, meaning: 'بدون وابستگی مالی و تعارض؛ هنگام ثبت دوباره بررسی می‌شود', basis: 'وضعیت در لحظه' },
      { label: 'نیازمند بررسی', value: fa(review), tone: review ? 'warn' : '', color: 'amber', meaning: 'مسدود یا نیازمند تطبیق', basis: 'وضعیت در لحظه' },
      { label: 'گیرنده مسیر اصلی (مدیر)', value: fa(M.managers.filter(function (m) { return m.active; }).length), color: 'blue', meaning: 'مدیرانِ فعال و در محدوده شما؛ سایر سطوح استثنایی‌اند', basis: 'ساختار فعلی' }
    ];
    var q = h.queues([
      { id: 'assign', label: 'تخصیص مشروط', icon: 'send', tone: 'blue', n: ok, key: '1' },
      { id: 'return', label: 'برگشت (پیش‌نمایش)', icon: 'repeat', tone: 'teal', n: M.held.filter(function (c) { return c.elig === 'ok'; }).length, key: '2' },
      { id: 'ops', label: 'سوابق عملیات', icon: 'history', tone: issue ? 'orange' : 'neutral', n: M.ops.length, key: '3' }
    ], st.aq, 'data-aq');
    var body = st.aq === 'return' ? returnView() : st.aq === 'ops' ? opsView() : assignView();
    return h.pageHead({ title: 'تخصیص مستقیم (مشروط)', sub: 'فقط از موجودی خودتان و فقط برای گیرنده مجاز · مسئول فعلی عوض می‌شود؛ مالک اولیه و مالک اعتبار نه · دیدن زیرمجموعه حق تصرف پرونده‌های مدیران نیست', kpis: kpis, fresh: X.freshPart('فهرست') }) + X.banners() +
      '<section class="panel main' + (st.aq === 'ops' ? '' : ' with-bulk') + '">' + h.banner('locked', '<b>مشروط / نیازمند سیاست (SD-G05).</b> مسیر فنی فعلی چهار سطح گیرنده را نگه می‌دارد، اما سیاست کسب‌وکاری تخصیص مستقیم معاون و عبور از مدیر مصوب نیست. این صفحه مجوز جدید یا زنجیره تأیید جدید ایجاد نمی‌کند.', '') + q + body + '</section>';
  };

  function plan() {
    var ids, recs;
    if (st.flow === 'outscope') return { ids: X.poolOk().slice(0, 2).map(function (c) { return c.id; }), recs: [X.recip('XM')], bypass: false, out: true };
    if (st.amode === 'select') { ids = keys(st.asel); recs = st.recip ? [X.recip(st.recip)] : []; }
    else { recs = keys(st.recips).map(X.recip); ids = X.poolOk().map(function (c) { return c.id; }).slice(0, st.per * recs.length); }
    return { ids: ids, recs: recs, bypass: recs.some(function (r) { return r.bypass.length; }), out: recs.some(function (r) { return r.scopeOut; }) };
  }
  function loadOf(r) { return r.kind === 'manager' ? X.mgrLoad(r.k) : r.kind === 'senior' ? X.seniorLoad(r.k) : r.kind === 'sup' ? X.teamLoad(r.k) : seller(Number(r.k.slice(1))).open; }
  function recipLabel(r) {
    if (r.scopeOut) return r.why;
    var base = r.kind === 'manager' ? 'مدیر مستقیم · پرونده نزد فروشندگان شاخه ' + fa(X.mgrLoad(r.k)) + '، فروشنده فعال ' + fa(X.sellersOfMgr(r.k).filter(function (s) { return s.active; }).length)
      : r.kind === 'senior' ? 'سرپرست ارشد · پرونده نزد واحد ' + fa(X.seniorLoad(r.k)) : r.kind === 'sup' ? 'سرپرست · پرونده نزد تیم ' + fa(X.teamLoad(r.k)) : 'فروشنده · پرونده باز ' + fa(loadOf(r));
    return r.ok ? base + (r.bypass.length ? ' · عبور از ' + r.bypass.join('، ') : '') : r.why;
  }
  function recipRow(k, single, extra) {
    var r = X.recip(k), on = single ? st.recip === k : !!st.recips[k], dis = !r.ok || (!single && r.bypass.length);
    return '<label class="recip' + (r.ok ? '' : ' off') + (on ? ' on' : '') + (extra ? ' ' + extra : '') + '"><input type="' + (single ? 'radio' : 'checkbox') + '" name="recip" class="cbx" data-recip="' + k + '"' + (on ? ' checked' : '') + (dis ? ' disabled' : '') + '><span class="r-main"><b>' + esc(r.name) + (r.scopeOut ? ' ' + X.rel('out') : '') + '</b><span>' + esc(recipLabel(r)) + '</span></span></label>';
  }
  function assignView() {
    var selIds = keys(st.asel), recN = keys(st.recips).length, rec = st.recip ? X.recip(st.recip) : null, pool = X.poolList();
    var stepCur = st.amode === 'select' ? (!selIds.length ? 0 : !rec ? 1 : 2) : (!recN ? 1 : 2);
    var head = '<div class="flow-head">' + X.steps(['انتخاب پرونده‌ها', 'انتخاب گیرنده', 'بررسی اثر', 'ثبت و نتیجه'], stepCur) +
      '<div class="seg" role="group" aria-label="روش تخصیص"><button type="button" data-amode="select" aria-pressed="' + (st.amode === 'select') + '">انتخاب پرونده‌ها</button><button type="button" data-amode="count" aria-pressed="' + (st.amode === 'count') + '">تعداد برای هر مدیر</button></div></div>';
    var left;
    if (!pool.length) left = h.stateBlock('empty', 'موجودی شخصی شما خالی است', 'پرونده‌ای نزد شما ثبت نشده است. صفر بودن موجودی ثبت دریافت را اثبات یا رد نمی‌کند؛ پرونده‌های مدیران را نمی‌توانید از این مسیر جابه‌جا کنید.', '');
    else if (st.amode === 'select') {
      var okIds = X.poolOk().map(function (c) { return c.id; }), allOn = okIds.length && okIds.every(function (id) { return st.asel[id]; });
      var rows = pool.map(function (c) {
        var e = X.ELIG[c.elig], dis = c.elig !== 'ok';
        return '<tr data-row="pcase:' + c.id + '" tabindex="-1" class="' + (st.asel[c.id] ? 'picked' : '') + (dis ? ' dim' : '') + '"><td class="col-sel">' + cbx('data-sel="asel" data-id="' + c.id + '"', st.asel[c.id], 'انتخاب پرونده ' + c.id + (dis ? ' — ' + e.label : ''), dis) + '</td>' +
          '<td><span class="mono case-id">' + esc(c.id) + '</span></td><td><span class="tag">' + esc(c.source) + '</span></td><td class="col-opt muted">' + esc(c.received) + '</td>' +
          '<td>' + pill(e.label, e.tone, e.icon) + (c.why ? '<div class="cell-sub">' + esc(c.why) + '</div>' : '') + '</td></tr>';
      }).join('');
      left = h.toolbar('شناسه پرونده یا دسته', [{ label: 'منبع / دسته', icon: 'filter' }, { label: 'تاریخ دریافت', icon: 'clock' }], '<span class="muted desk">' + fa(pool.length) + ' پرونده نزد شما</span>') +
        '<div class="tbl-wrap"><table class="tbl selectable" aria-label="پرونده‌های موجودی شخصی"><thead><tr><th class="col-sel" data-l="انتخاب">' + cbx('data-selall="asel"', allOn, 'انتخاب همه پرونده‌های قابل تخصیص') + '</th><th>پرونده</th><th>منبع</th><th class="col-opt">دریافت</th><th>قابلیت تخصیص ' + hint('هنگام ثبت دوباره بررسی می‌شود؛ پیش‌نمایش پرونده را رزرو نمی‌کند.', true) + '</th></tr></thead><tbody>' + rows + '</tbody></table></div>';
    } else {
      var total = st.per * recN, okN = X.poolOk().length;
      left = '<div class="count-form"><label class="lbl">تعداد پرونده برای هر مدیر <span class="req">*</span><input class="input" type="number" min="1" max="50" value="' + st.per + '" data-per inputmode="numeric" style="max-width:160px"></label>' +
        '<div class="note info">' + ic('info') + '<span>' + (recN ? fa(st.per) + ' × ' + fa(recN) + ' مدیر = <b>' + fa(total) + ' پرونده</b> از ' + fa(okN) + ' پرونده قابل تخصیص، به ترتیب قدیمی‌ترین دریافت.' + (total > okN ? ' <b>تعداد درخواستی بیشتر از موجودی قابل تخصیص است.</b>' : '') : 'مدیرهای مسیر اصلی را از فهرست انتخاب کنید.') + '</span></div>' +
        '<div class="muted" style="font-size:var(--t-meta)">روش شمارشی فقط برای مسیر اصلی (مدیر) است؛ عبور از مدیر با انتخاب صریح پرونده‌ها انجام می‌شود. فهرست دقیق و نتیجه هر پرونده در بررسی اثر و نتیجه می‌آید.</div></div>';
    }
    var single = st.amode === 'select';
    var primaryKeys = M.managers.map(function (m) { return m.id; }).concat(['XM']);
    var grp = function (title, sub, ks) { var op = st.recip && ks.indexOf(st.recip) > -1; return '<details class="exc-group"' + (op ? ' open' : '') + '><summary class="exc-gh"><b>' + title + ' <span class="muted">(' + fa(ks.length) + ')</span></b><span>' + sub + '</span></summary>' + ks.map(function (k) { return recipRow(k, single, 'exc'); }).join('') + '</details>'; };
    var seniorKs = M.seniors.map(function (u) { return u.id; }), supKs = M.teams.map(function (t) { return t.id; }), sellKs = M.sellers.map(function (s) { return 'S' + s.id; });
    var excOpen = st.bypassOpen || (st.recip && X.isBypass(st.recip));
    var aside = '<aside class="recips" aria-labelledby="rc-h"><div class="sec-h"><h3 id="rc-h">گیرنده ' + (single ? '(یک نفر)' : '(چند مدیر)') + '</h3><span class="scope-badge">' + ic('users') + 'مسیر اصلی · مدیر</span></div>' +
      '<p class="recip-note">' + ic('info') + 'مسیر معمول ساختار: <b>معاون ← مدیر</b>. هر گیرنده‌ی پایین‌تر از مدیر، مسیر عبور (bypass) است.</p>' + primaryKeys.map(function (k) { return recipRow(k, single); }).join('') +
      '<details class="exc-path"' + (excOpen ? ' open' : '') + ' data-exc><summary>' + ic('split') + '<span><b>عبور از مدیر (استثنایی)</b><span>مشروط · سیاست مصوب نیست (SD-G05) · مسیر فنی حفظ شده</span></span><span class="chev">' + ic('chev') + '</span></summary>' +
      '<p class="recip-note">' + ic('alert') + 'نمایش این گزینه‌ها به معنی مجاز بودن نیست. گیرنده باید فعال و در محدوده همین اقدام باشد و مسئول مدیر عبورشده حذف نمی‌شود؛ دیدن یک نفر مجوز نوشتن برای او نیست. ' + (single ? '' : 'در حالت «تعداد برای هر مدیر» فقط مسیر اصلی انتخاب‌شدنی است.') + '</p>' +
      grp('سرپرست ارشد', 'عبور از مدیر', seniorKs) + grp('سرپرست', 'عبور از مدیر و سرپرست ارشد (در صورت وجود)', supKs) + grp('فروشنده', 'عبور از هر سطح میانی', sellKs) +
      (st.recip && X.isBypass(st.recip) ? '<label class="lbl exc-reason">دلیل (برای ردیابی در سابقه عملیات)<textarea class="input" data-bypassreason placeholder="چرا این تخصیص از مدیر عبور می‌کند؟">' + esc(st.bypassReason) + '</textarea><span class="muted" style="font-size:var(--t-micro)">الزامی نیست؛ زنجیره تأیید جدیدی تعریف نشده است.</span></label>' : '') + '</details>' +
      '<p class="recip-note">' + ic('info') + 'گیرنده غیرفعال، خارج از محدوده یا بدون مجوز نوشتن اثبات‌شده انتخاب‌شدنی نیست و مالک جدید خودکار تعیین نمی‌شود.</p></aside>';
    var bb = '';
    if (st.amode === 'select' && selIds.length) bb = X.bulkbar(fa(selIds.length) + ' پرونده انتخاب شد', rec ? 'گیرنده: ' + esc(rec.name) + (rec.bypass.length ? ' · عبور از ' + esc(rec.bypass.join('، ')) : ' · مسیر اصلی') : 'یک گیرنده انتخاب کنید', '<button type="button" class="btn btn-ghost btn-sm" data-act="clear-asel">پاک کردن انتخاب</button><button type="button" class="btn btn-primary" data-act="review-assign"' + (rec ? '' : ' disabled aria-disabled="true"') + '>' + ic('eye') + 'بررسی اثر</button>');
    if (st.amode === 'count' && recN) bb = X.bulkbar(fa(st.per * recN) + ' پرونده برای ' + fa(recN) + ' مدیر', 'ثبت فقط پس از بررسی اثر', '<button type="button" class="btn btn-primary" data-act="review-assign">' + ic('eye') + 'بررسی اثر</button>');
    return head + '<div class="split"><div class="split-main">' + left + '</div>' + aside + '</div>' + bb;
  }

  function heldList() {
    var f = function (c) { return st.rf === 'all' ? true : st.rf === 'legacy' ? !!c.legacy : st.rf === 'review' ? (c.elig === 'unknown' || c.elig === 'conflict') : c.elig === st.rf; };
    return M.held.filter(f);
  }
  function retWhy(c) {
    if (c.elig === 'ok') return 'انتقال خودِ شما · فاکتور/پرداخت متصلی در پیش‌نمایش یافت نشد';
    if (c.elig === 'conflict') return c.why;
    return ({ fin: 'وابستگی مالی: ' + (c.invoice && c.invoice.label), notown: 'انتقال توسط مدیر انجام شده؛ رتبه اجازه بازپس‌گیری نمی‌دهد', mis: 'منبع MIS؛ برگشت با MIS است', consumed: 'مصرف‌شده در مسیر فاکتور V4: ' + (c.invoice && c.invoice.label), cancel: 'فاکتور لغوشده — قانون آزادسازی تعیین نشده (OPD-03)' })[c.why] || c.why;
  }
  function retCase(id) { return M.held.filter(function (x) { return x.id === id; })[0]; }
  function proofCell(c) {
    var p = c.proof;
    if (p.by === 'self') return '<span class="actor-who a-self">' + ic('user') + 'شما</span><div class="cell-sub">' + esc(p.op) + ' · ' + esc(p.when) + '</div>';
    if (p.by === 'manager') return '<span class="actor-who a-manager">' + ic('user') + 'مدیر</span><div class="cell-sub">' + esc(p.op) + '</div>';
    if (p.by === 'mis') return '<span class="actor-who a-mis">' + ic('layers') + 'MIS</span><div class="cell-sub">' + esc(p.op) + '</div>';
    return '<span class="muted">ثبت نشده</span><div class="cell-sub">' + esc(p.op) + '</div>';
  }
  function returnView() {
    var list = heldList(), okIds = list.filter(function (c) { return c.elig === 'ok'; }).map(function (c) { return c.id; });
    var selIds = keys(st.rsel).filter(retCase), allOn = okIds.length && okIds.every(function (id) { return st.rsel[id]; });
    var seg = '<div class="seg" role="group" aria-label="فیلتر قابلیت برگشت">' + [['all', 'همه'], ['ok', 'واجد شرایط'], ['blocked', 'مسدود'], ['review', 'تعارض / نیازمند تطبیق'], ['legacy', 'سرنخ قدیمی']].map(function (f) { return '<button type="button" data-rf="' + f[0] + '" aria-pressed="' + (st.rf === f[0]) + '">' + f[1] + '</button>'; }).join('') + '</div>';
    var rows = list.map(function (c) {
      var e = X.RET[c.elig], dis = c.elig !== 'ok';
      return '<tr data-row="ret:' + c.id + '" tabindex="-1" class="' + (st.rsel[c.id] ? 'picked' : '') + '"><td class="col-sel">' + cbx('data-sel="rsel" data-id="' + c.id + '"', st.rsel[c.id], 'انتخاب پرونده ' + c.id + (dis ? ' — ' + e.label : ''), dis) + '</td>' +
        '<td><span class="mono case-id">' + esc(c.id) + '</span>' + (c.legacy ? ' <span class="tag legacy">قدیمی</span>' : '') + '<div class="cell-sub">' + esc(c.source) + '</div></td>' +
        '<td>' + X.who(X.holderName(c), c.holder.kind === 'manager' ? X.rel('manager') : c.holder.kind === 'senior' ? X.rel('senior') : c.holder.kind === 'sup' ? X.rel('sup') : X.rel('seller')) + '</td><td class="col-opt">' + proofCell(c) + '</td>' +
        '<td>' + (c.invoice ? '<span class="mono">' + fa(c.invoice.code) + '</span><div class="cell-sub">' + esc(c.invoice.label) + '</div>' : '<span class="muted">در پیش‌نمایش یافت نشد</span>') + (c.link ? '<div class="cell-sub">' + X.link(c.link) + '</div>' : '') + '</td>' +
        '<td>' + pill(e.label, e.tone, e.icon) + '<div class="cell-sub">' + esc(retWhy(c)) + '</div></td>' +
        '<td class="col-actions"><button type="button" class="btn btn-sm" data-act="open-ret:' + c.id + '">بررسی</button></td></tr>';
    }).join('');
    var bb = selIds.length ? X.bulkbar(fa(selIds.length) + ' پرونده برای پیش‌نمایش انتخاب شد', 'ثبت برگشت فعال نیست (مشروط به F01)', '<button type="button" class="btn btn-ghost btn-sm" data-act="clear-rsel">پاک کردن انتخاب</button><button type="button" class="btn btn-primary" data-act="review-return">' + ic('eye') + 'بررسی برگشت</button>') : '';
    return h.banner('locked', '<b>برگشت فقط پیش‌نمایش است.</b> حفاظت کامل ثبت برگشت (بررسی همه ارتباط‌های مالی با اتصال اثبات‌شده، شکست در ابهام و بررسی مجدد هنگام ثبت) هنوز تکمیل نشده است (Gate 0 F01 / SD-G01). متن «واجد شرایط» تضمین امنیت مالی نیست.', '') +
      '<div class="note info inset">' + ic('shield') + '<span><b>معاون همه چیز را بازپس نمی‌گیرد.</b> اختیار برگشت از سابقه انتقال معتبر خودِ شما و وضعیت فعلی ردیف می‌آید، نه رتبه. در ابهام یا اتصال ناقص، نتیجه «مسدود/نامعلوم» است و با شماره موبایل حدس زده نمی‌شود. لغو/رد فاکتور خودکار آزادسازی نیست.</span></div>' +
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

  /* ---------- Allocation drawers ---------- */
  D.pcase = function (id) {
    var c = M.pool.filter(function (x) { return x.id === id; })[0], e = X.ELIG[c.elig];
    return top('پرونده در موجودی شخصی شما') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name" class="mono">' + esc(c.id) + '</h2>' + pill(e.label, e.tone, e.icon) + '</div><div class="dr-meta"><span>' + esc(c.source) + '</span><span>دریافت: ' + esc(c.received) + '</span></div></div>' +
      '<div class="dr-body"><section class="sec">' + own({ custody: 'موجودی شخصی شما (' + M.user.name + ')', original: c.source, next: 'شما — تخصیص به گیرنده مجاز (مشروط)', event: 'سامانه توزیع MIS · ' + c.received, credit: 'هنوز فروشی ثبت نشده' }) + '</section>' +
      X.sec('قابلیت تخصیص', '', c.elig === 'ok' ? checks([['ok', 'وابستگی مالی ندارد', 'فاکتور، پیش‌فاکتور یا پرداخت متصل یافت نشد'], ['ok', 'نزد گیرنده‌ای نیست', 'مسئول فعلی: موجودی شخصی شما'], ['info', 'هنگام ثبت دوباره بررسی می‌شود', 'این نمایش رزرو یا تضمین نیست']]) : checks([[c.elig === 'blocked' ? 'no' : 'q', e.label, c.why]]), c.elig === 'ok' ? '' : 'primary') + '</div>' +
      foot(c.elig === 'ok' ? '<button type="button" class="btn btn-lg btn-primary" data-act="pick-one:' + c.id + '">' + ic('check') + 'افزودن به انتخاب</button>' : '', null);
  };
  D.assignReview = function (id, keep) {
    var p = plan(), flow = st.flow, n = p.ids.length, names = p.recs.map(function (r) { return r.name; }).join('، '), confirmed = keep && keep.confirm;
    var each = st.amode === 'select' || p.out ? n : Math.min(st.per, n), conflict = flow === 'conflict', recipchg = flow === 'recipchg', out = p.out;
    var r0 = p.recs[0], byp = p.bypass ? p.recs.filter(function (r) { return r.bypass.length; })[0] : null;
    var impact = p.recs.map(function (r) {
      var cur = loadOf(r), add = out ? 0 : each - (conflict ? 1 : 0);
      return '<div class="mini"><div class="grow"><b>' + esc(r.name) + '</b> <span class="muted">(' + esc(r.level) + ')</span><div class="muted" style="font-size:var(--t-meta)">' + (out ? 'هیچ تغییری انجام نمی‌شود — گیرنده در محدوده نیست' : 'پرونده ' + (r.kind === 'seller' ? 'باز' : 'نزد زیرمجموعه') + ' ' + fa(cur) + ' ← <b>' + fa(cur + add) + '</b>' + (r.kind !== 'seller' ? ' · گیرنده آن را میان زیرمجموعه خود پخش می‌کند' : '')) + '</div></div><span class="tag">' + (out ? '۰' : '+' + fa(add)) + '</span></div>';
    }).join('');
    var sources = {}; p.ids.forEach(function (i) { var c = M.pool.filter(function (x) { return x.id === i; })[0]; if (c) sources[c.source] = (sources[c.source] || 0) + 1; });
    var srcTxt = Object.keys(sources).map(function (k) { return k + ' (' + fa(sources[k]) + ')'; }).join('، ') || '—';
    var relTxt = out ? 'هیچ رابطه‌ای در محدوده شما ثبت نیست' : !r0 ? '—' : r0.bypass.length ? 'گیرنده ' + r0.level + ' · عبور از ' + r0.bypass.join('، ') : 'گیرنده مسیر اصلی (مدیر مستقیم)';
    var pv = '<div class="exc-grid"><div><span>مسئول فعلی (Custody)</span><b>موجودی شخصی شما — برای ' + fa(n) + ' از ' + fa(n) + ' پرونده</b></div><div><span>رابطه گیرنده</span><b>' + esc(relTxt) + '</b></div><div><span>محدوده</span><b>' + (out ? '✗ خارج از محدوده مجاز' : '✓ در محدوده مجاز شما') + '</b></div><div><span>منبع (SourceRef)</span><b>' + esc(srcTxt) + '</b></div><div><span>واجد شرایط بودن گیرنده</span><b>' + (out ? 'اثبات نشد' : recipchg ? '✗ پس از بررسی غیرفعال شد' : r0 && r0.ok ? '✓ فعال و دارای مجوز اثبات‌شده برای این اقدام' : '—') + '</b></div><div><span>وابستگی مالی</span><b>' + (n ? 'در ' + fa(n) + ' پرونده انتخاب‌شده یافت نشد (' + fa(0) + ' مسدود)' : '—') + '</b></div></div>';
    var recheck = X.sec('بررسی دوباره هنگام ثبت (Apply recheck)', 'پیش‌نمایش ≠ ثبت', checks([['info', 'محدوده و فعال بودن گیرنده', 'دوباره از سرویس خوانده می‌شود؛ انتخاب در این فهرست مجوز نهایی نیست'], ['info', 'مسئول فعلی، وضعیت و وابستگی مالی هر پرونده', 'با محافظ رقابت هم‌زمان؛ پرونده تغییرکرده «ارسال‌نشده» می‌شود'], ['info', 'نتیجه به‌تفکیک هر پرونده نمایش داده می‌شود', 'درخواست‌شده / واجد شرایط / اعمال‌شده / ارسال‌نشده / ردشده / ناموفق / نامعلوم؛ «ثبت شد» یعنی انتقال ثبت شده، نه تأیید دریافت.']]));
    var skip = '';
    if (byp) {
      skip = X.sec('مسیر عبور از مدیر (استثنایی · مشروط)', 'SD-G05', '<div class="exc-grid"><div><span>گیرنده</span><b>' + esc(byp.name) + ' · ' + esc(byp.level) + '</b></div><div><span>سطح(های) ردشده</span><b>' + esc(byp.bypass.join('، ')) + '</b></div><div><span>مسئول فعلی پرونده‌ها</span><b>موجودی شخصی شما</b></div><div><span>دلیل (برای ردیابی در سابقه)</span><b>' + (st.bypassReason.trim() ? esc(st.bypassReason) : '<span class="muted">وارد نشده — الزامی نیست</span>') + '</b></div></div>' +
        checks([['ok', 'گیرنده فعال و در محدوده این اقدام است', 'طبق بررسی همین لحظه'], ['warn', 'سیاست تخصیص مستقیم/عبور از مدیر مصوب نیست', 'مسیر فنی فعلی بدون تغییر حفظ شده؛ زنجیره تأیید جدیدی اختراع نشده است. مدیر عبورشده خودکار مطلع نمی‌شود.'], ['info', 'مالک اولیه و مالک اعتبار تغییر نمی‌کند', 'یک رویداد انتقال با نام شما ثبت می‌شود.']]) +
        '<label class="confirm"><input type="checkbox" class="cbx" data-confirm' + (confirmed ? ' checked' : '') + '><span>می‌دانم که این تخصیص از ' + esc(byp.bypass.join('، ')) + ' عبور می‌کند و همین مسئولیت را می‌پذیرم.</span></label>', 'primary');
    }
    var alert = '';
    if (conflict) alert = '<section class="sec"><div class="note conflict" role="alert">' + ic('swap') + '<span><b>وضعیت پرونده تغییر کرده است.</b> پرونده C-51003 پس از این بررسی از موجودی شما برداشته شد. بازخوانی کنید؛ هیچ تغییری اعمال نشده است.</span></div></section>';
    if (recipchg) alert = '<section class="sec"><div class="note conflict" role="alert">' + ic('swap') + '<span><b>گیرنده پیش از ثبت تغییر کرد (Recipient changed).</b> ' + esc(names || 'گیرنده') + ' در بازبینی همین لحظه دیگر واجد شرایط نیست. بازنویسی یا انتخاب خودکار گیرنده دیگر انجام نمی‌شود؛ گیرنده را دوباره انتخاب کنید.</span></div></section>';
    if (out) alert = '<section class="sec"><div class="note conflict" role="alert">' + ic('shield') + '<span><b>تخصیص خارج از محدوده مجاز متوقف شد.</b> ' + esc(names) + ' در محدوده این اقدام نیست. دیدن نام یا گزارش مشترک مجوز نوشتن نیست و تخصیص بین دو مدیر یا خارج از محدوده از این مسیر مجاز نمی‌شود.</span></div></section>';
    var canCommit = !conflict && !recipchg && !out && n > 0 && (!byp || confirmed);
    var primary = conflict ? '<button type="button" class="btn btn-lg btn-primary" data-act="refresh-review">' + ic('refresh') + 'بازخوانی بررسی</button>' : recipchg || out ? '<button type="button" class="btn btn-lg btn-primary" data-act="refresh-recip">' + ic('refresh') + 'انتخاب مجدد گیرنده</button>' : '<button type="button" class="btn btn-lg btn-primary" data-act="commit-assign"' + (canCommit ? '' : ' disabled aria-disabled="true"') + '>' + ic('send') + 'تأیید و ثبت تخصیص</button>';
    return top('بررسی پیش از تخصیص') + '<div class="dr-head">' + X.steps(['انتخاب', 'گیرنده', 'بررسی اثر', 'نتیجه'], 2) + '<div class="dr-title" style="margin-top:12px"><h2 id="dr-name">تخصیص ' + fa(n) + ' پرونده</h2>' + (byp ? pill('عبور از مدیر · مشروط', 'amber', 'split') : out ? pill('خارج از محدوده', 'red', 'lock') : pill('مسیر اصلی', 'teal', 'checkCircle')) + '</div><div class="dr-meta"><span>به: ' + esc(names) + '</span><span>' + (st.amode === 'select' || out ? 'پرونده‌های انتخاب‌شده' : fa(st.per) + ' پرونده برای هر مدیر') + '</span></div></div>' +
      '<div class="dr-body">' + alert + '<section class="sec primary"><div class="sum-grid four"><div class="sum"><span>درخواست‌شده</span><b>' + fa(n) + '</b></div><div class="sum"><span>واجد شرایط (این لحظه)</span><b style="color:var(--teal-fg)">' + fa(out || recipchg ? 0 : conflict ? n - 1 : n) + '</b></div><div class="sum"><span>مسدود (حذف شد)</span><b>۰</b></div><div class="sum"><span>تعارض</span><b' + (conflict || recipchg || out ? ' style="color:var(--orange-fg)"' : '') + '>' + fa(conflict ? 1 : recipchg || out ? n : 0) + '</b></div></div></section>' +
      X.sec('پیش‌نمایش: حضانت، گیرنده، محدوده و منبع', 'snapshot، نه مجوز نهایی', pv) + X.sec('اثر بر گیرنده', '', '<div class="mini-list">' + impact + '</div>') + skip + recheck + '</div>' +
      foot(primary, '<button type="button" class="btn btn-lg btn-ghost" data-close>بازگشت به انتخاب</button>', 'چیزی بدون تأیید شما ثبت نمی‌شود');
  };

  // Operation results: per-item outcome; partial ≠ success; unknown is reconciled before any retry.
  function makeResult(kind, ids, toName, flow) {
    var items = ids.map(function (id) { return [id, 'ok', toName]; }), note = null, n = items.length;
    if (flow === 'partial' && n > 4) { items[n - 1] = [ids[n - 1], 'failed', toName, 'پاسخ سرور ناقص بود؛ قابل تکرار']; items[n - 2] = [ids[n - 2], 'skipped', toName, 'پرونده پیش از ارسال تغییر کرد (مسئول فعلی عوض شد)']; items[n - 3] = [ids[n - 3], 'rejected', toName, 'سامانه ثبت را نپذیرفت: وابستگی مالی فعال']; }
    else if (flow === 'retry') items = items.map(function (x) { return [x[0], 'failed', toName, 'خطای موقت شبکه؛ همان قصد قابل تکرار است']; });
    else if (flow === 'unknown') { items = items.map(function (x) { return [x[0], 'unknown', toName]; }); note = 'پاسخ تأیید نرسید؛ ممکن است ثبت شده باشد. پیش از تکرار با شماره عملیات تطبیق کنید.'; }
    return { ref: 'OP-6110', kind: kind, when: 'همین الان', actor: M.user.name, requested: n, to: toName, items: items, note: note };
  }
  function resultModel(id) { return id === 'live' ? st.live : M.ops.filter(function (o) { return o.ref === id; })[0]; }
  X.applyOk = function (op) {
    var okIds = op.items.filter(function (x) { return x[1] === 'ok'; }).map(function (x) { return x[0]; });
    if (op.kind === 'assign') M.pool = M.pool.filter(function (c) { return okIds.indexOf(c.id) < 0; });
    else okIds.forEach(function (id) { var c = retCase(id); if (c) { M.held = M.held.filter(function (x) { return x.id !== id; }); M.pool.push({ id: id, source: 'برگشت از ' + X.holderName(c), received: 'امروز', elig: 'ok' }); } });
    st.asel = {}; st.rsel = {};
  };
  X.runAssign = function (flow) {
    var p = plan(), op = makeResult('assign', p.ids.slice(), p.recs.map(function (r) { return r.name; }).join('، ') + (p.bypass ? ' (عبور از مدیر)' : ''), flow);
    st.live = op; st.flow = null; if (X.counts(op).ok) X.applyOk(op); M.ops.unshift(op);
    C.closeDrawer(); st.aq = 'ops'; C.render(); C.openDrawer('result', 'OP-6110');
  };
  D.result = function (id) {
    var o = id === 'OP-6110' ? M.ops[0] : resultModel(id);
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
    else if (c.why === 'notown') { body = checks([['no', 'انتقال را مدیر انجام داده است', p.op], ['info', 'رتبه، اجازه بازپس‌گیری نمی‌دهد', 'سابقه عامل انتقال با مقام بالاتر جایگزین نمی‌شود؛ معاون همه زیرمجموعه را بازپس نمی‌گیرد.']]); resp = { who: 'manager', text: 'مدیر شاخه' }; }
    else if (c.why === 'mis') { body = checks([['no', 'منبع MIS است', 'برگشت یا آزادسازی منبع در دامنه MIS است، نه معاون فروش'], ['info', 'شما مالک منابع وارد‌شده نیستید', '']]); resp = { who: 'mis', text: 'MIS' }; }
    else if (c.why === 'cancel') { body = checks([['no', 'فاکتور مرتبط لغو شده است', 'فاکتور ' + fa(c.invoice.code)], ['info', 'قانون آزادسازی تعیین نشده', 'برای پرونده با فاکتور لغوشده هنوز قانون آزادسازی رسمی وجود ندارد (OPD-03). لغو/رد به‌تنهایی آزادسازی نیست.']]); }
    else if (c.elig === 'conflict') { body = checks([['warn', 'وضعیت پرونده تغییر کرده است', c.why], ['info', 'پیش از هر اقدام بازخوانی کنید', 'اطلاعات نمایش‌داده‌شده ممکن است قدیمی باشد؛ بازنویسی انجام نمی‌شود']]); }
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
    return top('پیش‌نمایش برگشت گروهی') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">برگشت ' + fa(ids.length) + ' پرونده</h2>' + pill('مشروط · غیرفعال', 'amber', 'lock') + '</div><div class="dr-meta"><span>مقصد: موجودی شخصی من</span></div></div><div class="dr-body">' +
      X.sec('پرونده‌ها (پیش‌نمایش)', '', '<div class="mini-list">' + ids.map(function (i) { var c = retCase(i); return '<div class="mini"><div class="grow"><span class="mono">' + i + '</span><div class="muted" style="font-size:var(--t-meta)">نزد ' + esc(X.holderName(c)) + ' · ' + esc(c.proof.op) + '</div></div>' + pill('واجد شرایط پیش‌نمایش', 'teal', 'checkCircle') + '</div>'; }).join('') + '</div>', 'primary') +
      '<section class="sec"><div class="note warn">' + ic('lock') + '<span>ثبت برگشت گروهی تا تکمیل حفاظت F01 (SD-G01) فعال نیست. وقتی فعال شود: نتیجه به‌تفکیک پرونده (اعمال‌شده / ارسال‌نشده / ردشده / ناموفق / نامعلوم) نمایش داده می‌شود و ابهام «موفق» تلقی نمی‌شود.</span></div></section></div>' +
      foot('<button type="button" class="btn btn-lg" disabled aria-disabled="true">' + ic('lock') + 'ثبت برگشت — مشروط به F01</button>', null);
  };

  /* ================= HR request context (assigned current-reviewer step only; no HR administration) ================= */
  function hrSteps(r) {
    var done = st.hrDone[r.id]; if (!done) return r.steps;
    var s = r.steps.map(function (x) { return x.slice(); });
    var cur = s.map(function (x) { return x[1]; }).indexOf('cur');
    if (done === 'rejected') return s.slice(0, cur).concat([['معاون فروش (شما)', 'rejected', M.user.name + ' · رد با دلیل']]);
    s[cur] = ['معاون فروش (شما)', 'done', M.user.name + ' · تأیید این مرحله']; if (s[cur + 1]) s[cur + 1] = [s[cur + 1][0], 'cur', 'در انتظار اعمال نهایی'];
    return s;
  }
  function hrState(r) { var d = st.hrDone[r.id]; return d === 'rejected' ? X.HRS.rejected : d ? X.HRS.stepdone : r.state === 'pending_review' && r.reviewer !== 'self' ? X.HRS.observe : X.HRS[r.state]; }
  V.hr = function () {
    var inQ = function (r) { var done = st.hrDone[r.id]; return st.hq === 'review' ? r.state === 'pending_review' && r.reviewer === 'self' && !done : st.hq === 'chain' ? (r.state === 'pending_review' && r.reviewer !== 'self') || r.state === 'pending_hr' || done === 'step' : r.state === 'approved' || r.state === 'rejected' || r.state === 'failed' || done === 'rejected'; };
    var mine = M.hr.filter(function (r) { return r.state === 'pending_review' && r.reviewer === 'self' && !st.hrDone[r.id]; }).length;
    var QS = [{ id: 'review', label: 'منتظر بررسی شما', icon: 'hourglass', tone: 'orange', n: mine, key: '1' }, { id: 'chain', label: 'در مسیر (فقط مشاهده)', icon: 'eye', tone: 'slate', n: M.hr.filter(function (r) { return (r.state === 'pending_review' && r.reviewer !== 'self') || r.state === 'pending_hr'; }).length + Object.keys(st.hrDone).filter(function (k) { return st.hrDone[k] === 'step'; }).length, key: '2' }, { id: 'closed', label: 'بسته‌شده', icon: 'history', tone: 'neutral', n: M.hr.filter(function (r) { return r.state === 'approved' || r.state === 'rejected' || r.state === 'failed'; }).length, key: '3' }];
    var list = M.hr.filter(inQ);
    var cards = list.map(function (r) {
      var s = hrState(r), sel = seller(r.seller);
      return '<button type="button" class="hr-card" data-act="open-hr:' + r.id + '"><div class="hr-top"><b>#' + fa(r.id) + ' · ' + (r.type === 'transfer' ? 'جابجایی' : 'قطع همکاری') + ' · ' + esc(sel.name) + (r.dest ? ' ← ' + esc(r.dest) : '') + '</b>' + pill(s.label, s.tone, s.icon) + '</div>' + X.chain(hrSteps(r)) + '</button>';
    }).join('');
    return h.pageHead({ title: 'زمینه درخواست HR', sub: 'فقط وقتی شما بررسی‌کننده فعلی هستید · «تأیید این مرحله» با «اعمال نهایی توسط منابع انسانی» فرق دارد · ابزار مدیریت HR نیست', fresh: X.freshPart('درخواست‌ها') }) + X.banners() +
      '<section class="panel main">' + h.banner('locked', '<b>مشروط (SD-G09).</b> در نسخه فعلی پنل معاون صفحه مستقلی برای HR نیست و ورود/مرحله‌ی بررسی باید با مسیر، ماژول و ردیف اعتبارسنجی شود. بالا بودن رتبه بررسی‌کننده نمی‌سازد؛ فقط درخواستی که واقعاً به شما ارجاع شده قابل بررسی است.', '') + h.queues(QS, st.hq, 'data-hq') +
      (list.length ? '<div class="hr-list">' + cards + '</div>' : h.stateBlock('empty', 'درخواستی در این صف نیست', 'صف دیگری را ببینید. خالی بودن صف به معنی نبود مسیر نیست.', '')) +
      '<div class="bulk-note">' + ic('lock') + '<span>بررسی گروهی ندارد؛ هر درخواست جدا بررسی می‌شود. اعمال نهایی، دسترسی/نقش/رمز کارمند و تحویل کار (OPD-05) از این پنل ممکن نیست.</span></div></section>';
  };
  D.hr = function (id, keep) {
    var r = M.hr.filter(function (x) { return String(x.id) === String(id); })[0], done = st.hrDone[r.id], s = hrState(r), sel = seller(r.seller), steps = hrSteps(r);
    var canReview = r.state === 'pending_review' && r.reviewer === 'self' && !done, noteErr = keep && keep.err;
    var prior = steps.filter(function (x) { return x[1] === 'done' && !/^منابع/.test(x[0]) && !/شما/.test(x[0]); }).map(function (x) { return x[0]; }).join('، ') || '—';
    var cur = steps.filter(function (x) { return x[1] === 'cur'; })[0];
    var nxt = r.state === 'pending_review' && r.reviewer === 'manager' && !done ? 'معاون فروش (اگر در زنجیره باشد)، سپس منابع انسانی' : 'منابع انسانی (اعمال نهایی)';
    var vs = '<div class="step-vs"><div class="sv sv-step"><b>' + ic('check') + 'تأیید این مرحله</b><span>' + (canReview ? 'فقط بررسی شما را جلو می‌برد و درخواست را به مرحله بعد می‌فرستد.' : done === 'step' ? 'انجام شد — توسط شما ثبت شد.' : 'مرحله‌ی شما نیست یا قبلاً انجام شده است.') + '</span></div><div class="sv sv-final"><b>' + ic('lock') + 'اعمال نهایی توسط منابع انسانی</b><span>' + (r.state === 'approved' ? 'اعمال شد · ' + esc(r.applied) : r.state === 'failed' ? 'ناموفق بود · ' + esc(r.failedAt) : 'هنوز اعمال نشده است. فقط منابع انسانی انجام می‌دهد؛ از اینجا فعال نیست.') + '</span></div></div>';
    var review = canReview ? X.sec('بررسی مرحله شما', '', '<div class="note info">' + ic('shield') + '<span><b>تأیید شما فقط همین مرحله را جلو می‌برد.</b> جابجایی یا قطع همکاری اعمال نمی‌شود؛ اعمال نهایی را منابع انسانی انجام می‌دهد. دسترسی، نقش و رمز کارمند اینجا قابل تغییر نیست.</span></div>' +
      '<label class="lbl" style="margin-top:12px">یادداشت یا دلیل رد' + (noteErr ? ' <span class="req">* الزامی برای رد</span>' : '') + '<textarea class="input" data-hrnote placeholder="برای رد کردن، نوشتن دلیل الزامی است"' + (noteErr ? ' aria-invalid="true"' : '') + '>' + esc((keep && keep.note) || '') + '</textarea>' + (noteErr ? '<div class="field-err">' + ic('alert') + '<span>برای رد درخواست دلیل بنویسید.</span></div>' : '') + '</label>', 'primary') : '';
    var info = !canReview ? X.sec('وضعیت', '', checks([r.state === 'failed' ? ['no', 'اعمال نهایی ناموفق بود', 'تأیید مرحله‌ای انجام شده، اما تغییری اعمال نشده است؛ ' + (r.failedAt || '') + '. پیگیری با منابع انسانی است.'] : r.state === 'pending_hr' || done === 'step' ? ['info', 'مرحله‌ی شما انجام شده است', 'اقدام فعلی با ' + (done ? 'مرحله بعد' : 'منابع انسانی') + ' است؛ از سمت شما اقدامی لازم نیست'] : r.state === 'pending_review' ? ['info', 'درخواست در مرحله‌ی بررسی‌کننده دیگری است', 'شما فقط مشاهده می‌کنید؛ گذر از زنجیره یا بررسی به‌جای او مجاز نیست'] : ['info', 'این درخواست بسته شده است', r.applied ? 'اعمال‌شده · ' + r.applied : 'مرحله‌ای ادامه نمی‌یابد'], ['info', '«تأیید این مرحله» با «اعمال نهایی» فرق دارد', 'تغییر واقعی فقط پس از اعمال نهایی موفق ثبت می‌شود']])) : '';
    var handover = r.type === 'termination' ? X.sec('تحویل کار (تعریف نشده)', 'OPD-05', checks([['q', 'پرونده‌های باز، فاکتورها و مسئولیت‌های فعال', 'تعیین نشده است؛ غیرفعال‌شدن پروفایل به‌تنهایی تحویل کامل کار نیست'], ['info', 'انتساب تاریخی بدون تغییر می‌ماند', 'درخواستی که پیش از بررسی تغییر کرده باشد، تعارض است و نیازمند بازخوانی تازه است']])) : '';
    return X.top('درخواست HR #' + fa(r.id)) + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">' + (r.type === 'transfer' ? 'جابجایی' : 'قطع همکاری') + ' · ' + esc(sel.name) + '</h2>' + pill(s.label, s.tone, s.icon) + '</div><div class="dr-meta">' + (r.dest ? '<span>مقصد: ' + esc(r.dest) + '</span>' : '') + '<span>درخواست‌دهنده: ' + esc(r.requester) + '</span><span>تیم فعلی: ' + esc(X.teamName(sel.team)) + '</span></div></div>' +
      '<div class="dr-body"><section class="sec primary"><div class="sec-h"><h3>مسیر تأیید</h3></div>' + X.chain(steps) + '<dl class="exc-dl hr-dl"><div><dt>بررسی‌کننده فعلی</dt><dd>' + (cur ? esc(cur[0]) + ' — ' + esc(cur[2]) : r.state === 'pending_hr' || done === 'step' ? 'منابع انسانی' : '—') + '</dd></div><div><dt>بررسی‌کنندگان قبلی</dt><dd>' + esc(prior) + '</dd></div><div><dt>بررسی‌کننده بعدی</dt><dd>' + esc(nxt) + '</dd></div></dl></section>' + vs +
      '<section class="sec"><div class="sec-h"><h3>دلیل</h3></div><p style="margin:0">' + esc(r.reason) + '</p></section>' + review + info + handover + '</div>' +
      foot(canReview ? '<button type="button" class="btn btn-lg btn-primary" data-act="hr-step:' + r.id + '">' + ic('check') + 'تأیید این مرحله و ارجاع به مرحله بعد</button><button type="button" class="btn btn-lg btn-danger" data-act="hr-reject:' + r.id + '">رد با دلیل</button>' : '', null);
  };

  /* ================= Hierarchy detail drawers (visibility ≠ authority) ================= */
  function sumGrid(f, load) { return '<div class="sum-grid four"><div class="sum"><span>پرونده نزد فروشندگان</span><b>' + fa(load) + '</b></div><div class="sum"><span>فاکتور صادرشده</span><b>' + fa(f.created) + '</b></div><div class="sum"><span>پیش‌فاکتور باز</span><b>' + fa(f.openPre) + '</b></div><div class="sum"><span>فروش تکمیل‌شده</span><b>' + fa(f.completed) + '</b></div></div>'; }
  function miniBtn(act, title, sub, right) { return '<button type="button" class="mini as-btn" data-act="' + act + '"><div class="grow"><b>' + esc(title) + '</b><div class="muted" style="font-size:var(--t-meta)">' + sub + '</div></div>' + right + '</button>'; }
  var onOff = function (a) { return a ? pill('فعال', 'teal', 'checkCircle') : pill('غیرفعال', 'slate', 'ban'); };
  D.mgr = function (mid) {
    var m = X.mgr(mid), f = X.mgr_f(mid, 'current'), cov = X.mgrCov(mid), us = X.seniorsOfMgr(mid), dts = X.directTeams(mid), ex = X.branchExc(mid);
    var srcRows = M.sources.map(function (s) { return { id: s.id, label: s.label, state: cov === 'failed' ? 'failed' : s.state, note: s.note }; });
    var list = us.map(function (u) { return miniBtn('open-senior:' + u.id, 'سرپرست ارشد ' + u.name, 'پرونده نزد واحد ' + fa(X.seniorLoad(u.id)) + ' · فروشنده ' + fa(X.sellersOfSenior(u.id).length), onOff(u.active) + X.rel('senior')); }).concat(dts.map(function (t) { return miniBtn('open-team:' + t.id, 'سرپرست ' + t.sup, 'مستقیم زیر مدیر · پرونده نزد تیم ' + fa(X.teamLoad(t.id)), onOff(t.active) + X.rel('dsup')); })).join('');
    return top('شاخه مدیر') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">مدیر ' + esc(m.name) + '</h2>' + onOff(m.active) + X.rel('manager') + '</div><div class="dr-meta"><span>' + fa(us.length) + ' سرپرست ارشد</span><span>' + fa(X.sellersOfMgr(mid).length) + ' فروشنده</span><span>' + X.cov(cov) + '</span></div></div>' +
      '<div class="dr-body">' + (cov === 'failed' ? '<section class="sec"><div class="note warn">' + ic('xCircle') + '<span>دریافت داده این شاخه ناموفق بود. هیچ عددی صفر فرض نشده و موفقیت بقیه شاخه‌ها این شاخه را کامل نمی‌کند.</span></div></section>' : '') +
      (cov === 'failed' ? '' : '<section class="sec primary"><div class="sec-h"><h3>بار فعلی شاخه</h3>' + X.basis('current') + '</div>' + sumGrid(f, X.mgrLoad(mid)) + '</section>') +
      X.sec('پوشش منابع شاخه', 'BranchCoverage', X.covMatrix(srcRows, 'پوشش منابع شاخه')) +
      X.sec('استثناهای این شاخه', ex.length ? fa(ex.length) + ' مورد' : '', ex.length ? '<div class="mini-list">' + ex.map(function (e) { return miniBtn('open-exc:' + e.id, M.excClasses[e.cls].label, esc(e.subject) + '<br>مسئول: ' + esc(e.owner || 'نامشخص (UNKNOWN)'), pill(e.state, e.owner ? 'orange' : 'slate', e.owner ? 'hourglass' : 'question')); }).join('') + '</div>' : '<p class="ind-note">استثنای ثبت‌شده‌ای نیست. این فقط برای داده دریافت‌شده درست است، نه اثبات نبود مشکل.</p>') +
      X.details('زنجیره زیر این مدیر', us.length + dts.length, '<div class="mini-list">' + (list || '<div class="muted">زیرمجموعه‌ای ثبت نشده است.</div>') + '</div><p class="ind-note">غیرمستقیم برای شما: مشاهده و گزارش. دیدن یک نفر به معنی اجازه نوشتن برای او نیست و صف عملیاتی مدیر به شما منتقل نمی‌شود.</p>', true) + '</div>' +
      foot('<button type="button" class="btn btn-lg btn-primary" data-act="drill:' + mid + '">' + ic('chart') + 'عملکرد این شاخه</button>', '<button type="button" class="btn btn-lg" data-act="assign-to:' + mid + '"' + (m.active ? '' : ' disabled aria-disabled="true"') + '>' + ic('send') + 'تخصیص مشروط به این مدیر</button>', 'تخصیص مشروط است و فقط از موجودی خودتان');
  };
  D.senior = function (uid) {
    var u = X.senior(uid), ts = X.teamsOfSenior(uid), f = X.senior_f(uid, 'current'), cov = X.seniorCov(uid);
    var list = ts.map(function (t) { return miniBtn('open-team:' + t.id, 'سرپرست ' + t.sup, 'پرونده نزد تیم ' + fa(X.teamLoad(t.id)) + ' · فروشنده ' + fa(X.sellersOfTeam(t.id).length), onOff(t.active) + X.rel('sup')); }).join('');
    return top('سرپرست ارشد') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">' + esc(u.name) + '</h2>' + onOff(u.active) + X.rel('senior') + '</div><div class="dr-meta"><span>' + esc(X.mgrName(u.mgr)) + '</span><span>' + fa(ts.length) + ' سرپرست</span>' + (u.active ? '' : '<span>' + esc(u.inactiveNote) + '</span>') + '<span>' + X.cov(cov) + '</span></div></div>' +
      '<div class="dr-body">' + (u.active ? '' : '<section class="sec"><div class="note warn">' + ic('alert') + '<span>این سرپرست ارشد غیرفعال است. گیرنده تخصیص نیست و مسئول جایگزین خودکار تعیین نمی‌شود؛ با منابع انسانی/ساختار هماهنگ می‌شود.</span></div></section>') +
      '<section class="sec primary"><div class="sec-h"><h3>بار فعلی واحد</h3>' + X.basis('current') + '</div>' + sumGrid(f, X.seniorLoad(uid)) + '</section>' +
      X.details('سرپرستان این واحد', ts.length, '<div class="mini-list">' + (list || '<div class="muted">زیرمجموعه‌ای ثبت نشده است.</div>') + '</div>', true) + '</div>' +
      foot('<button type="button" class="btn btn-lg btn-primary" data-act="drill:' + uid + '">' + ic('chart') + 'عملکرد این واحد</button>', u.active ? '<button type="button" class="btn btn-lg" data-act="assign-to:' + uid + '">' + ic('split') + 'تخصیص استثنایی (عبور از مدیر)</button>' : '', u.active ? 'استثنایی و مشروط' : '');
  };
  D.team = function (tid) {
    var t = X.team(tid), ss = X.sellersOfTeam(tid), f = X.team_f(tid, 'current'), cov = X.teamCov(tid), un = X.seniorOfTeam(tid);
    var list = ss.map(function (s) { return miniBtn('open-seller:' + s.id, s.name, 'پرونده باز ' + fa(s.open) + (s.moved ? ' · منتقل‌شده از ' + esc(X.teamName(s.moved.from)) : ''), onOff(s.active) + X.rel('seller')); }).join('');
    return top('سرپرست و تیم') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">' + esc(t.sup) + '</h2>' + onOff(t.active) + X.rel(un ? 'sup' : 'dsup') + '</div><div class="dr-meta"><span>' + fa(ss.length) + ' فروشنده</span><span>' + esc(X.mgrName(X.mgrOfTeam(tid))) + (un ? ' › ' + esc(X.seniorName(un)) : ' › مستقیم') + '</span>' + (t.active ? '' : '<span>' + esc(t.inactiveNote) + '</span>') + '<span>' + X.cov(cov) + '</span></div></div>' +
      '<div class="dr-body">' + (t.active ? '' : '<section class="sec"><div class="note warn">' + ic('alert') + '<span>سرپرست این تیم غیرفعال است. گیرنده تخصیص نیست و مسئول جایگزین خودکار تعیین نمی‌شود.</span></div></section>') +
      '<section class="sec primary"><div class="sec-h"><h3>بار فعلی تیم</h3>' + X.basis('current') + '</div>' + sumGrid(f, X.teamLoad(tid)) + '</section>' +
      X.details('فروشندگان این تیم', ss.length, '<div class="mini-list">' + list + '</div><p class="ind-note">دیدن یک فروشنده به معنی اجازه نوشتن برای او نیست.</p>', true) + '</div>' +
      foot('<button type="button" class="btn btn-lg btn-primary" data-act="drill:' + tid + '">' + ic('chart') + 'عملکرد این تیم</button>', t.active ? '<button type="button" class="btn btn-lg" data-act="assign-to:' + tid + '">' + ic('split') + 'تخصیص استثنایی (عبور از مدیر)</button>' : '', t.active ? 'استثنایی و مشروط' : '');
  };
  D.seller = function (id) {
    var s = X.seller(Number(id)), f = X.seller_f(s.id), ch = X.chainOfSeller(s);
    return top('فروشنده زیرمجموعه') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">' + esc(s.name) + '</h2>' + onOff(s.active) + X.rel('seller') + '</div><div class="dr-meta"><span>' + esc(X.mgrName(ch.mgr)) + (ch.senior ? ' › ' + esc(X.seniorName(ch.senior)) : '') + (ch.team ? ' › ' + esc(X.teamName(ch.team)) : ' › مستقیم') + '</span></div></div>' +
      '<div class="dr-body"><section class="sec"><div class="note info">' + ic('eye') + '<span>شما برای این فروشنده فقط مشاهده و گزارش می‌بینید؛ تماس، فروش و پیگیری با مسئولیت خود او و زنجیره‌اش می‌ماند. اجازه تخصیص برای او ' + (s.alloc === 'ok' ? 'فقط در مسیر استثنایی عبور از مدیر' : 'اثبات نشده') + ' است.</span></div></section>' +
      (s.moved ? '<section class="sec"><div class="note conflict">' + ic('swap') + '<span>در ' + esc(s.moved.when) + ' از ' + esc(X.teamName(s.moved.from)) + ' منتقل شده است. فروش‌های قبل از انتقال به تیم هنگام صدور نسبت داده می‌شوند، نه ساختار امروز.</span></div></section>' : '') +
      '<section class="sec primary"><div class="sec-h"><h3>خلاصه</h3>' + X.basis('event') + '</div><div class="sum-grid four"><div class="sum"><span>پرونده باز</span><b>' + fa(s.open) + '</b></div><div class="sum"><span>فاکتور صادرشده</span><b>' + fa(f.created) + '</b></div><div class="sum"><span>پیش‌فاکتور باز</span><b>' + fa(f.openPre) + '</b></div><div class="sum"><span>فروش تکمیل‌شده</span><b>' + fa(f.completed) + '</b></div></div></section>' +
      X.details('فاکتورها', f.list.length, '<div class="mini-list">' + (f.list.length ? f.list.map(function (i) { return '<button type="button" class="mini as-btn" data-act="open-inv:' + i.code + '"><div class="grow"><span class="mono">' + fa(i.code) + '</span><div class="muted" style="font-size:var(--t-meta)">' + esc(i.customer) + '</div></div>' + pill(X.INVS[i.inv].label, X.INVS[i.inv].tone, X.INVS[i.inv].icon) + '</button>'; }).join('') : '<div class="muted">فاکتوری ثبت نشده است.</div>') + '</div>', f.list.length > 0) + '</div>' +
      foot('', null);
  };

  /* ================= Owner-aware exceptions ================= */
  D.attn = function (cid) {
    var a = X.attention().filter(function (x) { return x.id === cid; })[0]; if (!a) return top('استثنا') + '<div class="dr-body">' + h.stateBlock('empty', 'موردی نیست', '') + '</div>';
    return top('دسته استثنا') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">' + esc(a.label) + '</h2>' + pill(fa(a.n) + ' مورد', a.tone, a.icon) + '</div></div><div class="dr-body">' +
      X.sec('موارد این دسته', '', '<div class="mini-list">' + a.list.map(function (e) { return miniBtn('open-exc:' + e.id, e.subject, 'مالک: ' + esc(e.owner || 'نامشخص (UNKNOWN)') + ' · مسئول بعدی: ' + esc(e.next), pill(e.state, e.owner ? 'orange' : 'slate', e.owner ? 'hourglass' : 'question')); }).join('') + '</div>', 'primary') +
      X.sec('ارجاع رسمی', '', checks([['info', 'دستور ارجاع، اعلان یا SLA جدیدی تعریف نشده است', '«مسئول بعدی» همان مالک مسیر موجود است؛ مسئول جدید ساخته یا اعلان ارسال نمی‌شود (SD-G07). حل‌شدن به معنی آزادشدن یا قابل‌تخصیص‌شدن پرونده نیست.']])) + '</div>' +
      foot('<button type="button" class="btn btn-lg btn-primary" data-act="goto-exc:' + cid + '">' + ic('arrowL') + 'دیدن همه در صفحه استثناها</button>', null);
  };
  D.exc = function (id) {
    var e = X.excOf(id); if (!e) return top('استثنا') + '<div class="dr-body">' + h.stateBlock('empty', 'موردی نیست', '') + '</div>';
    var list = function (arr, ic_, cls) { return '<ul class="elig">' + arr.map(function (t) { return '<li class="' + cls + '">' + ic(ic_) + '<span><b>' + esc(t) + '</b></span></li>'; }).join('') + '</ul>'; };
    var c = M.excClasses[e.cls];
    return top('استثنای نیازمند هماهنگی') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">' + esc(c.label) + '</h2>' + pill(e.state, e.owner ? 'orange' : 'slate', e.owner ? 'hourglass' : 'question') + '</div><div class="dr-meta"><span>' + (e.mgr ? esc(X.mgrName(e.mgr)) : 'بین چند شاخه / بدون شاخه') + '</span>' + (e.since ? '<span>از ' + esc(e.since) + '</span>' : '<span>زمان معتبر ثبت نشده</span>') + '</div></div><div class="dr-body">' +
      '<section class="sec primary"><dl class="exc-dl"><div><dt>موضوع</dt><dd>' + esc(e.subject) + '</dd></div><div><dt>مالک / مسئول (Responsible owner)</dt><dd>' + (e.owner ? esc(e.owner) : '<span class="pill t-slate">' + ic('question') + 'نامشخص (UNKNOWN)</span> — رتبه معاون مالک پیش‌فرض نیست') + '</dd></div><div><dt>وضعیت فعلی</dt><dd>' + esc(e.state) + '</dd></div><div><dt>چرا توجه لازم است؟</dt><dd>' + esc(e.why) + '</dd></div><div><dt>مسئول گام بعدی (Next Actor)</dt><dd>' + esc(e.next) + '</dd></div><div><dt>علامت حل (Resolution signal)</dt><dd>' + esc(e.signal) + '</dd></div></dl></section>' +
      X.sec('اقدام مجاز شما', '', list(e.may, 'checkCircle', 'e-ok')) + X.sec('ممنوع در این پنل', '', list(e.mayNot, 'xCircle', 'e-no')) +
      X.sec('ارجاع رسمی', '', checks([['info', 'دستور ارجاع، اعلان یا SLA جدیدی تعریف نشده است', '«ارجاع» یعنی مشخص‌کردن مسئول موجود؛ مسئول جدید ساخته یا اعلان ارسال نمی‌شود (SD-G07). اقدام گروهی روی استثناها نیازمند اعتبارسنجی است.']])) + '</div>' +
      foot('<button type="button" class="btn btn-lg btn-primary" data-act="exc-go:' + e.id + '">' + ic('arrowL') + 'رفتن به صفحه مرتبط</button>', null);
  };
  D.metrics = function () {
    var rows = M.kpiDefs.map(function (k) { return '<tr><td><b>' + esc(k.name) + '</b></td><td>' + esc(k.grain) + '</td><td>' + esc(k.source) + '</td><td>' + esc(k.cohort) + '</td><td>' + esc(k.time) + '</td><td>' + esc(k.scope) + '</td><td>' + esc(k.formula) + '</td><td>' + esc(k.fresh) + '</td><td>' + X.cov('undef', 'باز: ' + k.open) + '</td></tr>'; }).join('');
    return top('تعریف شاخص‌ها') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">متادیتای هر شاخص</h2></div><div class="dr-meta"><span>نام · واحد شمارش · منبع · گروه شمارش · مبنای زمانی · محدوده · فرمول · تازگی</span></div></div><div class="dr-body">' +
      '<section class="sec primary"><div class="tbl-wrap"><table class="tbl no-cursor stackable" aria-label="تعریف شاخص‌ها"><thead><tr><th>نام</th><th>واحد شمارش</th><th>منبع</th><th>گروه شمارش</th><th>مبنای زمانی</th><th>محدوده</th><th>فرمول/شرط</th><th>تازگی</th><th>بخش باز</th></tr></thead><tbody>' + rows + '</tbody></table></div></section>' +
      X.sec('توجه', '', checks([['info', 'نرخ، رتبه، هدف، امتیاز بازدهی و SLA تعریف نشده‌اند', 'هیچ‌کدام در این پنل نمایش داده نمی‌شود'], ['info', 'cohort تجاری باز است', 'تعریف شاخص قرارداد هدف است؛ اجرای فنی resolver مشترک هنوز نهایی نیست (NOT FINAL).'], ['info', 'F02 و F03 رفع‌شده فرض نمی‌شوند', 'شرط شمارش پیش‌فاکتور و پوشش منابع کل پرونده‌ها شکاف باز پیاده‌سازی‌اند.'], ['info', 'داده نامعلوم/دریافت‌نشده هرگز صفر نیست', '']])) + '</div>' + foot('', null);
  };

  /* ================= Scoped invoice drawer (READ only) ================= */
  function facets(i) {
    var s = X.INVS[i.inv], rv = X.REV[i.review], done = i.inv === 'completed' && i.review === 'approved';
    var f = [['وضعیت فاکتور', pill(s.label, s.tone, s.icon)], ['مرحله پرداخت', '<b>مرحله ' + fa(i.stage) + ' از ' + fa(i.stages) + '</b><span>' + esc(X.STG[i.stg]) + '</span>'], ['مدرک پرداخت', '<b>' + esc(X.EVID[i.evidence]) + '</b>' + (i.evidence === 'receipt' ? '<span>ثبت رسید ≠ تأیید پرداخت</span>' : '')], ['بررسی مالی', rv ? pill(rv.label, rv.tone, rv.icon) : '<b class="muted">ارسال نشده</b>'], ['فروش تکمیل‌شده', done ? pill('بله', 'green', 'checkCircle') : '<b class="muted">خیر</b>'], ['اقدام بعدی با', i.next ? X.nextActor(i.next, i) : '<b class="muted">—</b>']];
    return '<div class="facets">' + f.map(function (x) { return '<div class="facet"><span class="f-l">' + x[0] + '</span><div class="f-v">' + x[1] + '</div></div>'; }).join('') + '</div>';
  }
  D.inv = function (code) {
    var i = M.invoices.filter(function (x) { return x.code === code; })[0], s = seller(i.seller), ch = X.chainOfSeller(s), St = X.INVS[i.inv];
    var policy = [['ok', 'checkCircle', 'مشاهده فاکتور، مرحله و بررسی مالی', 'خواندنی و در محدوده شما', 'مجاز'], ['pending', 'question', 'کپی / ارسال مجدد لینک پرداخت', 'کمک‌ابزار مدیر است و به معاون منتقل نمی‌شود؛ در پنل معاون دیده نشده — بدون مجوز مفروض', 'نیازمند اعتبارسنجی'], ['pending', 'question', 'ویرایش رسید / پیش‌پرداخت', 'مشاهده محدوده مجوز نوشتن نیست؛ نگهبان اقدام پرداخت معاون را شامل نمی‌شود', 'محدود'], ['locked', 'lock', 'تأیید/رد مالی، بازپرداخت، ثبت مالی', 'خارج از مأموریت؛ دیدن گزارش مالی اختیار نمی‌دهد', 'در این نقش نیست'], ['locked', 'lock', 'پورسانت، لغو، اتصال مجدد', 'مقام معاون مجوز تغییر مالی یا هویت پرونده نیست', 'محدود']];
    return top('زمینه فاکتور') + '<div class="dr-head"><div class="dr-title"><h2 id="dr-name" class="mono">' + fa(i.code) + '</h2>' + pill(St.label, St.tone, St.icon) + X.link(i.link) + '</div><div class="dr-meta"><span>' + esc(i.customer) + '</span><span>فروشنده: ' + esc(s.name) + '</span><span>صدور: ' + esc(i.issued) + '</span></div></div>' +
      '<div class="dr-body"><section class="sec primary">' + facets(i) + '<div class="sum-grid" style="margin-top:12px"><div class="sum"><span>مبلغ کل</span><b>' + money(i.total) + '</b></div><div class="sum"><span>پرداخت‌شده</span><b style="color:var(--green-fg)">' + money(i.paid) + '</b></div><div class="sum"><span>مانده</span><b>' + money(i.inv === 'cancelled' ? 0 : i.total - i.paid) + '</b></div></div>' +
      (i.reason ? '<div class="note warn" style="margin-top:12px">' + ic('alert') + '<span>' + esc(i.reason) + '</span></div>' : '') + (i.mismatch ? '<div class="note warn" style="margin-top:12px">' + ic('question') + '<span>' + esc(i.mismatch) + '</span></div>' : '') + (i.inv === 'pre' ? '<div class="note warn" style="margin-top:12px">' + ic('swap') + '<span>این پیش‌فاکتور در شاخص «پیش‌فاکتور باز» نمای کلی شمرده نمی‌شود (F02)؛ رفع‌شده فرض نشده است.</span></div>' : '') + '</section>' +
      X.sec('زنجیره و منبع', '', '<div class="ctx-chain big"><span>' + esc(X.mgrName(ch.mgr)) + '</span>' + (ch.senior ? '<span>' + esc(X.seniorName(ch.senior)) + '</span>' : '') + (ch.team ? '<span>سرپرست ' + esc(team(ch.team).sup) + '</span>' : '') + '<b>' + esc(s.name) + '</b></div><p class="ind-note" style="padding:8px 0 0">' + esc(i.case) + ' — زنجیره فعلی است؛ انتساب تاریخی و مالک اعتبار از آن نتیجه نمی‌شود.</p>') +
      X.sec('مالکیت', '', own({ custody: 'نامربوط به فاکتور (حضانت پرونده در جای دیگر)', original: s.name, next: i.next ? (X.nextActor(i.next, i).replace(/<[^>]+>/g, ' ').replace(/\s+/g, ' ').trim()) : 'ندارد', event: 'فروشنده ' + s.name + ' · صدور ' + i.issued, credit: 'طبق قوانین مالی فعلی؛ از گروه‌بندی گزارش یا والد امروز نتیجه نمی‌شود' })) +
      X.details('اقدام‌های این پنل', null, '<ul class="act-policy">' + policy.map(function (p) { return '<li class="ap-' + p[0] + '"><span class="ap-ic">' + ic(p[1]) + '</span><span class="ap-t"><b>' + p[2] + '</b><span>' + p[3] + '</span></span><span class="ap-s">' + p[4] + '</span></li>'; }).join('') + '</ul>', true) + '</div>' +
      foot(i.link !== 'confirmed' || i.inv === 'pre' ? '<button type="button" class="btn btn-lg btn-primary" data-act="goto-diag:' + (i.inv === 'pre' ? 'f02' : 'lineage') + '">' + ic('info') + 'عیب‌یابی شاخص و منشأ</button>' : '', null, 'فقط‌خواندنی');
  };
})();
