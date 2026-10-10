/* HR — role layer, part 2: page views (Workforce, Onboarding/Transfer, Requests, Structure & Access, Compensation, Exceptions/Audit,
   Credentials/Impersonation, Import/Bulk/Export, Legacy mapping/Diagnostics). Presentation of Product-Spec-approved facts only.
   Read pages never mutate; unknown / unauthorized / missing history render as a truthful state, never as 0 or as today's value. */
(function () {
  'use strict';
  var X = window.HRX, C = X.C, M = X.M, h = X.h, st = X.st, V = X.V;
  var esc = h.esc, fa = h.fa, ic = h.ic, pill = h.pill, hint = h.hint;
  var na = function (t) { return '<span class="na tip" tabindex="0" data-tip="' + esc(t || 'داده در دسترس نیست') + '">—</span>'; };
  var ph = function (icon, title, id, tags, aside) { return '<div class="sec-h ph"><h2 id="' + id + '">' + ic(icon) + title + '</h2>' + (tags || '') + (aside ? '<span class="aside">' + aside + '</span>' : '') + '</div>'; };
  var open = function (kind, id, label, cls) { return '<button type="button" class="btn ' + (cls || 'btn-soft') + ' btn-sm" data-act="open-' + kind + ':' + id + '">' + label + '</button>'; };
  var table = function (label, caption, head, body, cls) { return '<div class="tbl-wrap"><table class="tbl' + (cls ? ' ' + cls : '') + '" aria-label="' + esc(label) + '"><caption class="sr">' + esc(caption) + '</caption><thead><tr>' + head + '</tr></thead><tbody>' + body + '</tbody></table></div>'; };
  var confChips = function (p) { return p.conf.length ? p.conf.map(function (k) { return pill(X.CONF[k], k === 'termOpen' || k === 'dupProfile' ? 'red' : 'amber', 'alert'); }).join(' ') : '<span class="muted">—</span>'; };
  var parentCell = function (p) { return p.parentUnknown ? '<b>نامعلوم (UNKNOWN)</b><span class="cell-sub neg">' + ic('question') + ' ثبت بازه جدید تأیید نشده</span>' : p.parent ? '<b>' + esc(X.name(p.parent)) + '</b>' + (X.isMgrOk(p.parent) ? '' : '<span class="cell-sub neg">' + ic('alert') + ' غیرفعال</span>') : '<span class="muted">ثبت نشده</span>'; };

  /* ================= Workforce Explorer ================= */
  V.work = function () {
    var q = st.wq, all = M.staff;
    var list = all.filter(function (p) { return q === 'all' ? true : q === 'conf' ? p.conf.length : q === 'mgr' ? p.conf.indexOf('inactiveMgr') > -1 : p.conf.indexOf('histGap') > -1 || p.hist.some(function (e) { return e.state !== 'known'; }); });
    var W = X.work(), conflicts = all.filter(function (p) { return p.conf.length; }).length, histInc = all.filter(function (p) { return p.hist.some(function (e) { return e.state !== 'known'; }); }).length;
    var kpis = [
      { label: 'پروفایل در فهرست', value: fa(all.length), color: 'slate', meaning: 'فقط فهرست بارگذاری‌شده؛ جستجوی سراسری کاربران نیست و شاخص عملکرد یا درصد آمادگی نیست', basis: 'وضعیت فعلی' },
      { label: 'پروفایل با ناهمخوانی باز', value: fa(conflicts), color: 'orange', keep: true, meaning: 'پیوند ناقص، تکرار، ناهمخوانی نقش/سمت، مدیر غیرفعال… ؛ هر مورد مالک و اقدام مجاز دارد', basis: 'وضعیت فعلی' },
      { label: 'تاریخچه ناقص یا نامعلوم', value: fa(histInc), color: 'amber', meaning: 'بازه‌هایی که نامعلوم/ناقص‌اند؛ از مدیر امروز پر نمی‌شوند', basis: 'تاریخچه بازه‌ای' }
    ];
    var qs = h.queues([{ id: 'all', label: 'همه', icon: 'users', n: all.length, key: '1' }, { id: 'conf', label: 'ناهمخوانی باز', icon: 'alert', n: conflicts, key: '2', tone: 'orange' }, { id: 'mgr', label: 'مدیر غیرفعال', icon: 'user', n: W.inactive, key: '3', tone: 'amber' }, { id: 'hist', label: 'تاریخچه ناقص', icon: 'history', n: histInc, key: '4', tone: 'slate' }], q, 'data-wq');
    var rows = list.map(function (p) {
      var pos = X.pos(p.pos);
      return '<tr data-row="staff:' + p.id + '" tabindex="-1"><td>' + X.who(p.name, '<span class="mono muted">' + p.id + '</span>') + '</td><td><div class="idcell"><span class="idc">' + X.lk(p.lk[0], 'شخص↔پروفایل: ' + X.LK[p.lk[0]][0]) + '</span><span class="idc">' + X.lk(p.lk[1], 'پروفایل↔حساب: ' + X.LK[p.lk[1]][0]) + '</span></div></td>' +
        '<td>' + esc(pos.label) + '<span class="cell-sub">سطح ' + esc(p.lvl) + ' · ' + esc(pos.role) + '</span></td><td class="col-opt">' + parentCell(p) + '<span class="cell-sub">مدیر فعلی — اعتبار تاریخی نیست</span></td><td>' + X.empPill(X.empOf(p)) + '</td><td>' + X.accPill(p.role === 'nouser' ? 'nouser' : p.role) + '</td><td class="col-opt wrap">' + confChips(p) + '</td><td class="col-actions">' + open('staff', p.id, 'پروفایل') + '</td></tr>';
    }).join('');
    return h.pageHead({ title: 'نیروها', sub: 'شخص ↔ پروفایل نیرو ↔ حساب وردپرس سه چیز جدا هستند · سمت، سطح، نقش و واحد هم جدا · هیچ ادغام یا اصلاح خودکار انجام نمی‌شود', kpis: kpis, cta: '<button type="button" class="btn btn-primary" data-act="open-onbwiz">' + ic('plus') + 'ورود نیروی جدید</button>', fresh: X.freshPart('فهرست'), scope: 'کل سازمان به معنی دسترسی به همه اطلاعات حساس نیست' }) + X.banners('work') +
      '<section class="panel main" aria-labelledby="wf-h">' + ph('users', 'فهرست نیروها', 'wf-h', X.basis('snap'), 'زمان اعمال و تاریخ اثر در پروفایل جداست') + qs + h.toolbar('جستجو: نام، شناسه پروفایل یا حساب (فقط فهرست بارگذاری‌شده)…', [], '') +
      table('فهرست نیروها', 'هر ردیف یک پروفایل نیرو؛ ستون‌های پیوند، سمت، مدیر فعلی، وضعیت اشتغال و دسترسی جدا هستند.', '<th>پروفایل نیرو</th><th>پیوند هویت</th><th>سمت · سطح</th><th class="col-opt">مدیر فعلی</th><th>اشتغال</th><th>دسترسی</th><th class="col-opt">ناهمخوانی‌ها</th><th class="col-actions"><span class="sr">اقدام</span></th>', rows) + h.tfoot(list.length, all.length) + '</section>';
  };

  /* ================= Onboarding / Transfer ================= */
  V.onb = function () {
    var q = st.oq, qs = h.queues([{ id: 'onb', label: 'ورود نیرو', icon: 'user', n: 2, key: '1' }, { id: 'transfer', label: 'تغییر مدیر (انتقال)', icon: 'swap', n: M.staff.length, key: '2' }], q, 'data-oq'), body;
    if (q === 'onb') {
      var items = [
        { ref: 'HP-306', name: 'ندا سلیمی', st: [['شناسایی هویت', 'ok'], ['پروفایل/پیوند', 'bad'], ['سمت و سطح', 'ok'], ['جایگاه سلسله‌مراتب', 'ok'], ['فعال‌سازی دسترسی', 'pending'], ['تکمیل', 'unres']], note: 'پروفایل ساخته شد ولی حساب پیوند نشد؛ دسترسی اعمال نشده' },
        { ref: 'ON-2', name: 'علی فتحی (ورودی جدید)', st: [['شناسایی هویت', 'conflict'], ['پروفایل/پیوند', 'pending'], ['سمت و سطح', 'pending'], ['جایگاه سلسله‌مراتب', 'pending'], ['فعال‌سازی دسترسی', 'pending'], ['تکمیل', 'pending']], note: 'موبایل با یک کاربر و یک پروفایل موجود مطابقت دارد؛ پیش از ادامه باید انتخاب صریح شود' }
      ];
      var icn = { ok: ['checkCircle', 'تأیید', 'teal'], bad: ['xCircle', 'ناموفق', 'red'], pending: ['dashed', 'در انتظار', 'slate'], unres: ['question', 'اثر باز', 'amber'], conflict: ['swap', 'تعارض', 'orange'] };
      var rows = items.map(function (it) {
        return '<tr' + (it.ref.indexOf('HP-') === 0 ? ' data-row="staff:' + it.ref + '" tabindex="-1"' : '') + '><td>' + X.who(it.name, '<span class="mono muted">' + it.ref + '</span>') + '</td><td><ol class="onb-st" aria-label="مراحل ورود نیرو">' + it.st.map(function (s) { var d = icn[s[1]]; return '<li class="os-' + s[1] + '">' + pill(s[0] + ': ' + d[1], d[2], d[0]) + '</li>'; }).join('') + '</ol></td><td class="wrap">' + esc(it.note) + '</td><td class="col-actions">' + (it.ref.indexOf('HP-') === 0 ? open('staff', it.ref, 'پروفایل') : '<button type="button" class="btn btn-soft btn-sm" data-act="open-onbwiz">ادامه شناسایی</button>') + '</td></tr>';
      }).join('');
      body = h.banner('info', '<b>ورود نیرو مرحله‌ای است ولی گام ساختگی ندارد.</b> مراحل از کارکردهای کد-تأییدشده می‌آیند؛ آموزش، سند و گام‌های حقوقی تعریف نشده‌اند. اعتبارنامه (گذرواژه/پیامک) گام عادی ورود نیرو نیست و فقط از «اعتبارنامه و جایگزینی» (محدود) انجام می‌شود.', '') +
        table('ورود نیرو در جریان', 'هر ردیف یک ورود نیرو با وضعیت هر مرحله جدا؛ یک پیام موفقیت دلیل کامل‌بودن نیست.', '<th>نیرو</th><th>مراحل</th><th>اثر باز / وضعیت</th><th class="col-actions"><span class="sr">اقدام</span></th>', rows, 'no-cursor');
    } else {
      var trows = M.staff.filter(function (p) { return p.parent || p.pos !== 'deputy'; }).map(function (p) {
        var o = p.open;
        return '<tr data-row="staff:' + p.id + '" tabindex="-1"><td>' + X.who(p.name, '<span class="mono muted">' + p.id + '</span>') + '</td><td>' + esc(X.posL(p.pos)) + '</td><td class="col-opt">' + parentCell(p) + '</td><td class="col-opt">' + (o ? (o.verified ? fa(o.leads + o.inv + o.cust + o.tasks) + ' مورد' : na('اثر کار باز تأیید نشده')) : na('اثر کار باز برای این نقش ارزیابی نشده')) + '</td><td class="col-actions"><button type="button" class="btn btn-soft btn-sm" data-act="open-xfer:' + p.id + '">پیش‌نمایش تغییر مدیر</button></td></tr>';
      }).join('');
      body = h.banner('stale', '<b>پیش‌نمایش ثبت نهایی نیست.</b> تغییر مستقیم سلسله‌مراتب مسیر کد-موجودِ جداست و با نهایی‌کردن درخواست انتقال یکی نیست. تأیید مرحله‌ای ≠ انتقال اعمال‌شده. تحویل کار باز (OPD-05) مشروط و تعریف‌نشده است و تاریخ اثر آینده‌نگر پشتیبانی‌شده تأیید نشده.', '<button type="button" class="btn btn-sm" data-act="goto:req">درخواست‌ها</button>') + table('تغییر مدیر', 'هر ردیف یک نیرو؛ پیش‌نمایش تغییر مدیر اثر دسترسی، کار باز و حفظ تاریخچه را نشان می‌دهد.', '<th>نیرو</th><th>سمت</th><th class="col-opt">مدیر فعلی</th><th class="col-opt">کار باز (فروش)</th><th class="col-actions"><span class="sr">اقدام</span></th>', trows) + h.tfoot(M.staff.length, M.staff.length);
    }
    return h.pageHead({ title: 'ورود نیرو و انتقال', sub: 'شناسایی ← پروفایل/پیوند ← سمت ← سلسله‌مراتب ← دسترسی ← تکمیل · تغییر مدیر با پیش‌نمایش اثر', cta: q === 'onb' ? '<button type="button" class="btn btn-primary" data-act="open-onbwiz">' + ic('plus') + 'ورود نیروی جدید</button>' : '', fresh: X.freshPart('ورود/انتقال'), scope: X.zone('normal', 'ورود و انتقال') }) + X.banners('onb') + '<section class="panel main" aria-labelledby="ob-h">' + ph('user', 'ورود نیرو و انتقال', 'ob-h', '', '') + qs + body + '</section>';
  };

  /* ================= Requests ================= */
  V.req = function () {
    var q = st.rq, all = M.requests, S = function (r) { return X.reqSt(r); };
    var G = { review: function (r) { return S(r) === 'pending_hr'; }, chain: function (r) { return S(r) === 'pending_review' || S(r) === 'approved_step'; }, closed: function (r) { return ['approved', 'rejected', 'failed'].indexOf(S(r)) > -1; } };
    var n = function (k) { return all.filter(G[k]).length; };
    var qs = h.queues([{ id: 'review', label: 'منتظر اعمال منابع انسانی', icon: 'hourglass', n: n('review'), key: '1', tone: 'orange' }, { id: 'chain', label: 'در مسیر بررسی', icon: 'eye', n: n('chain'), key: '2' }, { id: 'closed', label: 'بسته‌شده', icon: 'history', n: n('closed'), key: '3' }], q, 'data-rq');
    var list = all.filter(G[q]);
    var rows = list.map(function (r) {
      var p = X.s(r.subj), flags = [];
      if (r.changed) flags.push(pill('تغییر پیش از بررسی', 'orange', 'swap'));
      if (r.targetInactive) flags.push(pill('مدیر مقصد غیرفعال', 'red', 'ban'));
      if (r.oos) flags.push(pill('خارج از محدوده', 'slate', 'lock'));
      if (r.type === 'terminate' && p.open && p.open.leads + p.open.inv > 0 && (S(r) === 'pending_hr' || S(r) === 'approved_step')) flags.push(pill('تحویل ناتمام', 'red', 'lock'));
      var reqText = r.type === 'transfer' ? 'تغییر مدیر: ' + X.name(r.from) + ' ← ' + X.name(r.to) : 'پایان همکاری' + (r.eff ? ' · اثر درخواستی ' + r.eff : '');
      return '<tr data-row="req:' + r.id + '" tabindex="-1"><td>' + X.who(p.name, '<span class="mono muted">' + r.id + '</span> · ' + (r.type === 'transfer' ? 'انتقال' : 'پایان همکاری')) + '</td><td class="wrap">' + esc(reqText) + '</td><td>' + X.reqPill(S(r)) + '</td><td class="col-opt">' + esc(r.cur) + '<span class="cell-sub">بعدی: ' + esc(r.next) + '</span></td><td class="col-opt wrap">' + (flags.join(' ') || '<span class="muted">—</span>') + '</td><td class="col-opt">' + esc(r.appliedAt || 'اعمال نشده') + '</td><td class="col-actions">' + open('req', r.id, 'بررسی') + '</td></tr>';
    }).join('');
    return h.pageHead({ title: 'درخواست‌های منابع انسانی', sub: 'درخواست ← بررسی ← تأیید مرحله‌ای/رد ← منابع انسانی ← اعمال ← اعمال‌شده یا ناموفق · تأیید مرحله‌ای ≠ تغییر اعمال‌شده', fresh: X.freshPart('درخواست‌ها'), scope: 'شناسه‌های داخلی وضعیت حفظ شده‌اند' }) + X.banners('req') +
      '<section class="panel main" aria-labelledby="rq-h">' + ph('inbox', 'درخواست‌ها', 'rq-h', X.basis('snap'), 'فهرست محدودشده؛ همه فیلدها برای همه مجاز نیست') + qs + h.toolbar('جستجو: شناسه درخواست یا نیرو…', [], '') +
      table('درخواست‌ها', 'هر ردیف یک درخواست با وضعیت، بررسی‌کننده فعلی و بعدی و پرچم‌های تعارض؛ تأیید مرحله‌ای با اعمال نهایی یکی نیست.', '<th>درخواست · نیرو</th><th>تغییر درخواستی</th><th>وضعیت</th><th class="col-opt">بررسی‌کننده فعلی / بعدی</th><th class="col-opt">پرچم‌ها</th><th class="col-opt">زمان اعمال</th><th class="col-actions"><span class="sr">اقدام</span></th>', rows) + h.tfoot(list.length, all.length) + '</section>';
  };

  /* ================= Structure & Access ================= */
  V.acc = function () {
    var q = st.aq, qs = h.queues([{ id: 'matrix', label: 'ماتریس وضعیت دسترسی', icon: 'key', key: '1', n: M.staff.filter(function (p) { return p.role !== 'ok'; }).length, tone: 'orange' }, { id: 'struct', label: 'سمت، سطح و واحد', icon: 'layers', key: '2' }], q, 'data-aq'), body;
    if (q === 'matrix') {
      var rows = M.staff.map(function (p) {
        var a = M.access[p.id];
        return '<tr data-row="access:' + p.id + '" tabindex="-1"><td>' + X.who(p.name, '<span class="mono muted">' + p.id + '</span>') + '</td><td>' + esc(X.posL(p.pos)) + '<span class="cell-sub">نگاشت سمت ← نقش عملیاتی</span></td><td>' + X.accPill(p.role === 'nouser' ? 'nouser' : p.role) + '</td><td class="col-opt wrap">' + (p.role === 'ok' ? '<span class="muted">مورد انتظار = واقعی</span>' : esc((a && (a.partial || a.actualRole)) || (p.role === 'nouser' ? 'حساب پیوند نشده؛ دسترسی اعمال نمی‌شود' : '—'))) + '</td><td class="col-actions">' + open('access', p.id, 'بازرس دسترسی') + '</td></tr>';
      }).join('');
      body = h.banner('locked', '<b>بازرس دسترسی است، نه ویرایشگر قابلیت‌های وردپرس.</b> سمت کاری، مجوز محصولی، نقش/قابلیت و اعتبارنامه چهار چیز جدا هستند؛ نقش‌های نامرتبط و استثناها با تغییر سمت بازنشانی نمی‌شوند.', '') + table('وضعیت دسترسی نیروها', 'هر ردیف یک نیرو با سمت و وضعیت دسترسی مورد انتظار در برابر واقعی.', '<th>نیرو</th><th>سمت و نگاشت</th><th>دسترسی</th><th class="col-opt">تفاوت مورد انتظار و واقعی</th><th class="col-actions"><span class="sr">اقدام</span></th>', rows) + h.tfoot(M.staff.length, M.staff.length);
    } else {
      var prow = M.positions.map(function (p) {
        return '<tr data-row="pos:' + p.id + '" tabindex="-1"><td><b>' + esc(p.label) + '</b></td><td class="n">' + (p.level == null ? na('سطح تعریف نشده') : fa(p.level)) + '</td><td>' + esc(p.role) + '<span class="cell-sub">نگاشت فعلی حفظ می‌شود</span></td><td class="n">' + fa(M.staff.filter(function (s) { return s.pos === p.id; }).length) + '</td><td class="col-actions">' + open('pos', p.id, 'بررسی اثر غیرفعال‌سازی') + '</td></tr>';
      }).join('');
      var units = [['تیم شمال', 'HP-110', 'والد گزارش‌دهی: معاون (HP-100)'], ['تیم غرب', 'HP-130', 'مدیر غیرفعال؛ والد گزارش‌دهی: معاون'], ['تیم شمال ۱', 'HP-210', 'والد گزارش‌دهی: مدیر فرهاد نجفی']];
      body = '<div class="split2">' + table('سمت‌ها و سطح‌ها', 'هر ردیف یک سمت؛ غیرفعال‌سازی یا حذف مستقل از ویرایش برچسب است و ارجاع‌ها را نشان می‌دهد.', '<th>سمت</th><th class="n">سطح</th><th>نقش عملیاتی نگاشت‌شده</th><th class="n">نیروی فعلی</th><th class="col-actions"><span class="sr">اقدام</span></th>', prow, 'no-cursor') +
        '<div class="u-box">' + X.sec('واحد / تیم ≠ سلسله‌مراتب گزارش‌دهی', '', '<ul class="elig">' + units.map(function (u) { return '<li class="e-info">' + ic('layers') + '<span><b>' + esc(u[0]) + '</b><span>مدیر: ' + esc(X.name(u[1])) + ' · ' + esc(u[2]) + '</span></span></li>'; }).join('') + '</ul><p class="ind-note">' + ic('info') + ' واحد/دپارتمان/تیم لزوماً همان والد reports_to نیست؛ این دو جدا ثبت می‌شوند.</p>') + '</div></div>';
    }
    return h.pageHead({ title: 'ساختار و دسترسی', sub: 'سمت کاری ≠ مجوز محصولی ≠ نقش/قابلیت ≠ اعتبارنامه ≠ استثنای مستقیم/ارث · هیچ ویرایش قابلیت وردپرس عرضه نمی‌شود', fresh: X.freshPart('دسترسی'), scope: X.zone('normal', 'مشاهده') + ' ' + X.zone('cond', 'تغییر با تأیید اثر') }) + X.banners('acc') + '<section class="panel main" aria-labelledby="ac-h">' + ph('key', 'ساختار و دسترسی', 'ac-h', X.basis('snap'), '') + qs + body + '</section>';
  };

  /* ================= Compensation ================= */
  V.comp = function () {
    var denied = st.compAuth === 'denied', list = M.staff.filter(function (p) { return M.comp[p.id]; });
    var body;
    if (denied) body = h.stateBlock('locked', 'دسترسی به جبران خدمات برای حساب شما فعال نیست', 'نمایش‌ندادن مقدار به معنی صفر یا نبود جبران خدمات نیست. میدان حقوق و ارجاع کمیسیون فقط برای دارندگان مجوز جداگانه دیده می‌شود.', '<button type="button" class="btn" data-act="comp-auth:authorized">نمایش حالت مجاز (دمو)</button>');
    else {
      var rows = list.map(function (p) {
        var c = M.comp[p.id], per = c.periods[0];
        return '<tr data-row="comp:' + p.id + '" tabindex="-1"><td>' + X.who(p.name, '<span class="mono muted">' + p.id + '</span>') + '</td><td class="n">' + (c.base ? esc(c.base) + '<span class="of"> ' + c.cur + '</span>' : na('جبران خدمات ثبت نشده؛ صفر نیست')) + '</td><td class="col-opt">' + (c.com ? pill('واجد کمیسیون', 'teal', 'checkCircle') + '<span class="cell-sub">ارجاع قاعده: ' + esc(c.rule) + ' · ' + esc(c.mode) + '</span>' : pill('بدون کمیسیون', 'slate', 'ban')) + '</td><td class="col-opt">' + (per ? esc(per.from) + ' — ' + (per.to || 'ادامه') : na('بدون دوره')) + '</td><td class="col-opt wrap">' + (c.overlap ? pill('همپوشانی بازه', 'orange', 'alert') : '') + (c.periods.some(function (x) { return x.gap; }) ? ' ' + pill('دوره ناقص', 'amber', 'layers') : '') + (!c.overlap && !c.periods.some(function (x) { return x.gap; }) ? '<span class="muted">—</span>' : '') + '</td><td class="col-actions">' + open('comp', p.id, 'تاریخچه') + '</td></tr>';
      }).join('');
      body = h.banner('locked', '<b>جبران خدمات از کیف پول، عملکرد فروش، پرداخت مالی و موتور کمیسیون جداست.</b> تغییر اینجا اعتبار کسب‌شده گذشته را دوباره محاسبه نمی‌کند و مدرک پرداخت نیست. قفل پس از حقوق (payroll lock) تأییدنشده است.', '') + table('جبران خدمات (مجازها)', 'هر ردیف یک نیروی دارای جبران خدمات؛ مقدار تعریف‌نشده با خط تیره نمایش داده می‌شود نه صفر.', '<th>نیرو</th><th class="n">پایه فعلی</th><th class="col-opt">کمیسیون (ارجاع)</th><th class="col-opt">دوره فعلی</th><th class="col-opt">هشدار یکپارچگی</th><th class="col-actions"><span class="sr">اقدام</span></th>', rows) + h.tfoot(list.length, list.length);
    }
    return h.pageHead({ title: 'جبران خدمات', sub: 'مبلغ پایه، واحد، دوره اثر، ارجاع کمیسیون و تاریخچه · فقط برای مجازها · جدا از کیف پول و مالی', fresh: X.freshPart('جبران خدمات'), scope: X.zone('restr', 'حساس') }) + X.banners('comp') + '<section class="panel main" aria-labelledby="cm-h">' + ph('wallet', 'جبران خدمات', 'cm-h', X.basis('hist'), '') + body + '</section>';
  };

  /* ================= Exceptions / Audit ================= */
  V.exc = function () {
    var q = st.xq, qs = h.queues([{ id: 'exc', label: 'استثناها', icon: 'inbox', n: M.exc.length, key: '1', tone: 'orange' }, { id: 'audit', label: 'خط زمانی ممیزی', icon: 'history', n: M.audit.length, key: '2' }], q, 'data-xq'), body;
    if (q === 'exc') {
      var rows = M.exc.map(function (e) {
        var c = X.XCLS[e.cls], s = X.XST[e.st];
        return '<tr data-row="exc:' + e.id + '" tabindex="-1"><td class="wrap"><div class="exc-cls">' + pill(c[0], e.tone, c[1]) + '</div><b class="exc-subj">' + esc(e.title) + '</b></td><td class="col-opt wrap">' + esc(e.owner) + '</td><td>' + pill(s[0], s[1], s[2]) + '</td><td class="col-opt wrap muted">' + esc(e.impact) + '</td><td class="col-opt wrap"><span class="allowed">' + ic('eye') + esc(e.allowed[0]) + '</span></td><td class="col-actions">' + open('exc', e.id, 'شواهد') + '</td></tr>';
      }).join('');
      body = h.banner('locked', '<b>این صف تشخیص است، نه فرمان.</b> هیچ SLA یا آستانه «بیش‌ازحد منتظر» ساخته نشده؛ سن فقط از زمان معتبر خوانده می‌شود. هر مورد مالک، اقدام مجاز منابع انسانی، اقدام ممنوع و مدرک حل دارد.', '') + table('استثناهای منابع انسانی', 'هر ردیف یک استثنا با موضوع، مالک، وضعیت، اثر و اقدام مجاز.', '<th>استثنا</th><th class="col-opt">مالک</th><th>وضعیت</th><th class="col-opt">اثر</th><th class="col-opt">اقدام مجاز</th><th class="col-actions"><span class="sr">اقدام</span></th>', rows) + h.tfoot(M.exc.length, M.exc.length);
    } else {
      var arows = M.audit.map(function (a) {
        var r = X.OPS_S(a.res);
        return '<tr data-row="audit:' + a.id + '" tabindex="-1"><td>' + X.who(a.act, esc(a.at)) + '</td><td class="col-opt">' + esc(X.name(a.subj)) + '<span class="cell-sub mono">' + esc(a.subj) + (a.req !== '—' ? ' · ' + a.req : '') + '</span></td><td class="col-opt wrap">' + esc(a.before) + ' ← ' + esc(a.after) + '</td><td class="col-opt">' + esc(a.eff) + '<span class="cell-sub">اعمال: ' + esc(a.applied) + '</span></td><td>' + pill(r[0], r[1], r[2]) + '</td><td class="col-actions">' + open('audit', a.id, 'جزئیات') + '</td></tr>';
      }).join('');
      body = table('خط زمانی ممیزی', 'هر ردیف یک رویداد با عامل، موضوع، قبل/بعد، تاریخ اثر، زمان اعمال و نتیجه؛ ممیزی کامل و تغییرناپذیر تأیید نشده است.', '<th>رویداد · زمان</th><th class="col-opt">موضوع</th><th class="col-opt">قبل ← بعد</th><th class="col-opt">تاریخ اثر / اعمال</th><th>نتیجه</th><th class="col-actions"><span class="sr">اقدام</span></th>', arows) + '<p class="ind-note">' + ic('info') + ' گزارش آخرین عملیات بازنویسی می‌شود و تاریخچه کامل نیست؛ نبود ردیف اثبات نمی‌کند که رویدادی رخ نداده. گذرواژه، حقوق و داده بانکی در ممیزی نمایش داده نمی‌شوند.</p>';
    }
    return h.pageHead({ title: 'استثناها و ممیزی', sub: 'ناهمخوانی‌ها، اعمال ناموفق و شکاف‌های تاریخچه با مالک و مدرک حل · بدون SLA ساختگی', fresh: X.freshPart('استثناها'), scope: X.zone('normal', 'تشخیص خواندنی') }) + X.banners('exc') + '<section class="panel main" aria-labelledby="ex-h">' + ph('inbox', 'استثناها و ممیزی', 'ex-h', '', '') + qs + body + '</section>';
  };

  /* ================= Credentials / Impersonation (RESTRICTED) ================= */
  V.cred = function () {
    var rows = M.staff.filter(function (p) { return p.user; }).map(function (p) {
      var lead = ['deputy', 'manager', 'senior', 'sup'].indexOf(p.pos) > -1, el = lead ? ['red', 'هدف دارای سطح مدیریتی؛ سیاست هدف مجاز تأیید نشده (HR-G04)', 'lock'] : ['teal', 'هدف داخلی غیرممتاز (نمونه)', 'checkCircle'];
      return '<tr data-row="staff:' + p.id + '" tabindex="-1"><td>' + X.who(p.name, '<span class="mono muted">WP#' + p.user + '</span> · ' + p.id) + '</td><td>' + esc(X.posL(p.pos)) + '</td><td class="col-opt wrap">' + pill(el[1], el[0], el[2]) + '</td><td class="col-actions"><div class="row-actions">' + '<button type="button" class="btn btn-soft btn-sm" data-act="open-credreset:' + p.id + '">' + ic('key') + 'بازنشانی</button><button type="button" class="btn btn-soft btn-sm" data-act="open-impstart:' + p.id + '">' + ic('eye') + 'نمایش به‌جای کاربر</button></div></td></tr>';
    }).join('');
    var sessions = X.checks([['q', 'سابقه نمایش به‌جای کاربر', 'ثبت دائمی شروع/پایان/دلیل در کد بررسی‌شده تأیید نشده؛ فقط ردپای موقت عامل-هدف (۳۰ دقیقه) دیده شده است'], ['warn', 'این حالت فقط‌خواندنی نیست', 'در جلسه، احراز هویت به هدف تغییر می‌کند و مسدودیت همگانی نوشتن تأیید نشده'], ['info', 'بازگشت', 'با دکمه بازگشت یا انقضا؛ خروج خودکار تضمین نشده']]);
    return h.pageHead({ title: 'اعتبارنامه و نمایش به‌جای کاربر', sub: 'مسیر محدود و حساس · جدا از ویرایش نیرو · هیچ گذرواژه یا رازی نمایش داده نمی‌شود', scope: X.zone('restr', 'محدود'), fresh: X.freshPart('اهداف') }) + X.banners('cred') +
      h.banner('locked', '<b>مسیر محدود/مشروط است.</b> در کد فعلی دروازه عمومی منابع انسانی این کارها را می‌پذیرد (ادعای «فقط مدیرکل» خلاف شواهد است)؛ سیاست دقیق هدف مجاز، ارسال اعتبارنامه و ممیزی ماندگار هنوز تعریف نشده است (HR-G04/G05). هیچ مورد از این صفحه در نمونه اجرا نمی‌شود.', '') +
      '<section class="panel main" aria-labelledby="cr-h">' + ph('key', 'اهداف و اقدام‌های حساس', 'cr-h', X.zone('restr'), '') + table('اهداف اعتبارنامه', 'هر ردیف یک حساب داخلی با شرط هدف مجاز؛ اهداف مدیریتی مسدود نمایش داده می‌شوند.', '<th>حساب</th><th>سمت</th><th class="col-opt">صلاحیت هدف</th><th class="col-actions"><span class="sr">اقدام</span></th>', rows, 'no-cursor') + '</section>' +
      '<section class="panel" aria-labelledby="im-h">' + ph('eye', 'نمایش به‌جای کاربر: وضعیت و سابقه', 'im-h', pill('پشتیبانی · نه فقط‌خواندنی', 'amber', 'alert'), '') + '<div class="cat-pad">' + sessions + '</div></section>';
  };
  X.impBar = function () {
    var i = st.imp; if (!i) return '';
    return '<div class="imp-bar" role="alert">' + ic('eye') + '<div><b>در حال نمایش به‌جای ' + esc(X.name(i.target)) + ' — هر اقدام با هویت او ثبت می‌شود</b><span>هدف: ' + esc(X.name(i.target)) + ' · هدف جلسه: ' + esc(i.purpose) + ' · شروع ' + esc(i.at) + ' · این جلسه فقط‌خواندنی نیست؛ بازگشت لازم است</span></div><button type="button" class="btn btn-sm btn-danger" data-act="imp-end">' + ic('logout') + 'بازگشت به حساب منابع انسانی</button></div>';
  };

  /* ================= Import / Bulk / Export (secondary, conditional) ================= */
  V.bulk = function () {
    var A = [
      { k: 'wf', name: 'به‌روزرسانی گروهی نیروها', cls: 'COND', note: 'فقط شناسه‌ها و میدان‌های ارائه‌شده؛ تغییر حساس جدا', act: ['open-bulkwf', 'بررسی اثر'], tone: 'amber' },
      { k: 'tr', name: 'انتقال گروهی', cls: 'COND', note: 'ویرایش مستقیم گروهی ≠ نهایی‌کردن درخواست‌ها؛ OPD-05', act: ['goto-onb', 'پیش‌نمایش انتقال تکی'], tone: 'amber' },
      { k: 'te', name: 'پایان همکاری گروهی', cls: 'NO', note: 'در خط پایه ممنوع: اثبات تحویل کار گروهی وجود ندارد', act: null, tone: 'red' },
      { k: 'ac', name: 'تغییر دسترسی گروهی', cls: 'VAL', note: 'تغییر سمت می‌تواند دسترسی را عوض کند؛ نیازمند اعتبارسنجی', act: null, tone: 'slate' },
      { k: 'cr', name: 'بازنشانی اعتبارنامه گروهی', cls: 'RESTR', note: 'مسیر حساس جداگانه؛ نه به‌روزرسانی عادی', act: ['open-bulkcred', 'بررسی اثر'], tone: 'red' },
      { k: 'co', name: 'به‌روزرسانی جبران خدمات گروهی', cls: 'COND', note: 'میدان و دوره صریح؛ یکپارچگی دوره‌ها نیازمند QA', act: null, tone: 'amber' },
      { k: 'ex', name: 'خروجی گروهی', cls: 'COND', note: 'انتخاب‌شده ≠ همه؛ فهرست میدان مجاز', act: ['open-export', 'قرارداد خروجی'], tone: 'amber' }
    ];
    var CL = { COND: ['مشروط', 'amber', 'alert'], NO: ['ممنوع در خط پایه', 'red', 'ban'], VAL: ['نیازمند اعتبارسنجی', 'slate', 'question'], RESTR: ['محدود · مشروط', 'red', 'lock'] };
    var rows = A.map(function (a) { var c = CL[a.cls]; return '<tr><td><b>' + esc(a.name) + '</b></td><td>' + pill(c[0], c[1], c[2]) + '</td><td class="wrap muted">' + esc(a.note) + '</td><td class="col-actions">' + (a.act ? (a.act[0] === 'goto-onb' ? '<button type="button" class="btn btn-soft btn-sm" data-act="goto:onb">' + a.act[1] + '</button>' : '<button type="button" class="btn btn-soft btn-sm" data-act="' + a.act[0] + '">' + a.act[1] + '</button>') : '<button type="button" class="btn btn-sm" disabled aria-disabled="true">' + ic('lock') + 'غیرفعال</button>') + '</td></tr>'; }).join('');
    var imp = '<section class="panel" aria-labelledby="ip-h">' + ph('upload', 'ورود/خروج فایل', 'ip-h', '', 'قرارداد sn_hr_users_v1') + '<div class="cat-pad"><div class="facets"><div class="facet"><span class="muted">قالب</span><b>CSV / XLSX · رفت‌وبرگشت</b></div><div class="facet"><span class="muted">سقف رابط</span><b>۵ مگابایت · ۱۰٬۰۰۰ ردیف</b></div><div class="facet"><span class="muted">خروجی انتخاب‌شده</span><b>حداکثر ۵٬۰۰۰ · قطعه‌های ۵۰۰</b></div></div><p class="ind-note">' + ic('alert') + ' «بررسی خشک» (dry-run) ممکن است گزینه‌ها را به‌روز کند و صرفاً خواندنی تضمین نمی‌شود؛ اعمال با کلمه APPLY و بررسی تعارض است. رفتار بازنویسی و امنیت در زمان اجرا تأیید نشده؛ فیلتر صفحه ≠ همه.</p></div></section>';
    return h.pageHead({ title: 'ورود، گروهی و خروجی', sub: 'ثانویه و مشروط · هیچ عمل مخرب گروهی پیش‌فرض نیست', scope: X.zone('cond', 'گروهی'), fresh: X.freshPart('عملیات') }) + X.banners('bulk') +
      '<section class="panel main" aria-labelledby="bk-h">' + ph('layers', 'عملیات گروهی', 'bk-h', '', 'خط پایه') + table('عملیات گروهی', 'هر ردیف یک عملیات گروهی با رده‌بندی خط پایه؛ همه نتیجه‌ها اعمال‌شده/ارسال‌نشده/ناموفق/نامعلوم را جدا می‌دهند.', '<th>عملیات</th><th>رده</th><th>توضیح</th><th class="col-actions"><span class="sr">اقدام</span></th>', rows, 'no-cursor') + '</section>' + imp;
  };

  /* ================= Legacy mapping / Diagnostics ================= */
  V.diag = function () {
    var q = st.dq, qs = h.queues([{ id: 'map', label: 'نقشه ابزارهای قدیمی', icon: 'compass', key: '1' }, { id: 'legacy', label: 'نگاشت قدیمی', icon: 'layers', key: '2' }, { id: 'log', label: 'لاگ فنی (پیشرفته)', icon: 'shield', key: '3' }], q, 'data-dq'), body;
    if (q === 'map') {
      var map = [['hr-workforce', 'نیروها', 'work'], ['hr-manual-add', 'ورود نیرو', 'onb'], ['hr-hierarchy', 'ورود نیرو و انتقال › تغییر مدیر', 'onb'], ['hr-change-requests', 'درخواست‌ها', 'req'], ['hr-positions + hr-structure', 'ساختار و دسترسی', 'acc'], ['overrides فاکتور/شماره اضافه', 'ساختار و دسترسی › ماتریس', 'acc'], ['hr-compensation', 'جبران خدمات', 'comp'], ['hr-logs + hr-overview', 'استثناها و ممیزی (+ نمای نیروها)', 'exc'], ['حساب/گذرواژه/پیامک + مشاهده پنل', 'اعتبارنامه و نمایش به‌جای کاربر (محدود)', 'cred'], ['hr-csv + hr-bulk', 'ورود، گروهی و خروجی', 'bulk'], ['hr-extra (ویرایشگر موازی)', 'ادغام در پروفایل نیرو', 'work']];
      body = table('نقشه ابزارهای قدیمی', 'هر ردیف یک ابزار فعلی و محل آن در ساختار جدید؛ مسیرها و شناسه‌ها تغییر نکرده‌اند.', '<th>ابزار فعلی</th><th>در ساختار جدید</th><th class="col-actions"><span class="sr">رفتن</span></th>', map.map(function (m) { return '<tr><td class="mono">' + esc(m[0]) + '</td><td>' + esc(m[1]) + '</td><td class="col-actions"><button type="button" class="btn btn-soft btn-sm" data-act="goto:' + m[2] + '">رفتن</button></td></tr>'; }).join(''), 'no-cursor');
    } else if (q === 'legacy') {
      body = table('نگاشت قدیمی', 'سازگاری قدیمی؛ هیچ بازنگاشت اجباری یا اصلاح همگانی انجام نمی‌شود.', '<th>منبع قدیمی</th><th>در ساختار جدید</th><th>توضیح</th>', M.legacy.map(function (l) { return '<tr><td>' + esc(l.src) + '</td><td>' + esc(l.map) + '</td><td class="wrap muted">' + esc(l.note) + '</td></tr>'; }).join(''), 'no-cursor');
    } else {
      body = '<div class="cat-pad">' + X.checks([['info', 'لاگ فنی با میدان‌های مجاز نمایش داده می‌شود', 'حقوق و داده حساس ماسک می‌شود؛ ماسک شدن لاگ، ماسک شدن خروجی‌ها را تضمین نمی‌کند'], ['warn', 'گزارش آخرین عملیات بازنویسی می‌شود', 'تاریخچه کامل نیست'], ['info', 'JSON خام فقط برای عیب‌یابی', 'بدون اصلاح از این صفحه']]) + '<pre class="rawlog" dir="ltr" tabindex="0" aria-label="نمونه لاگ فنی ماسک‌شده">{ "op":"hr_change_apply","request":"R-506","subject":"HP-309","before":{"parent":"HP-110"},"after":null,"salary":"***","result":"failed","corr":"hr-7a02" }</pre></div>';
    }
    return h.pageHead({ title: 'نگاشت قدیمی و عیب‌یابی', sub: 'پیشرفته · فقط خواندنی · تاریخچه و مقدارها دست‌نخورده', scope: X.zone('normal', 'تشخیص'), fresh: X.freshPart('تشخیص') }) + X.banners('diag') + '<section class="panel main" aria-labelledby="dg-h">' + ph('shield', 'نگاشت قدیمی و عیب‌یابی', 'dg-h', '', '') + qs + body + '</section>';
  };
  X.freshPart = function (label) { var s = X.sim() === 'stale'; return { text: label + ': ' + (s ? M.freshness.stale + ' (قدیمی)' : (C.state.fresh === 'now' ? 'همین الان' : 'امروز ' + M.freshness.now)) + ' · ارزیابی ' + M.freshness.evaluated, stale: s }; };
})();
