/* Finance — role layer, part 2: page views (Review Queue, Reconciliation/Refund, Ledger, Rules/Runs/Posting, Reports, Audit, Configuration & permissions, Maintenance).
   Presentation of Product-Spec-approved facts only. Read pages never mutate (F07); unknown / unauthorized / missing evidence render as truthful states, never as 0 or as a guess. */
(function () {
  'use strict';
  var X = window.FINX, C = X.C, M = X.M, h = X.h, st = X.st, V = X.V;
  var esc = h.esc, fa = h.fa, ic = h.ic, pill = h.pill, num = h.num;
  var ph = function (icon, title, id, tags, aside) { return '<div class="sec-h ph"><h2 id="' + id + '">' + ic(icon) + title + '</h2>' + (tags || '') + (aside ? '<span class="aside">' + aside + '</span>' : '') + '</div>'; };
  var open = function (kind, id, label, cls) { return '<button type="button" class="btn ' + (cls || 'btn-soft') + ' btn-sm" data-act="open-' + kind + ':' + id + '">' + label + '</button>'; };
  var table = function (label, caption, head, body, cls) { return '<div class="tbl-wrap"><table class="tbl' + (cls ? ' ' + cls : '') + '" aria-label="' + esc(label) + '"><caption class="sr">' + esc(caption) + '</caption><thead><tr>' + head + '</tr></thead><tbody>' + body + '</tbody></table></div>'; };
  var FLAGS = { noEvidence: ['مدرک ناموجود', 'amber', 'dashed'], mismatch: ['اختلاف مبلغ', 'orange', 'alert'], unit: ['واحد نامشخص', 'amber', 'question'], dup: ['تکراری احتمالی', 'red', 'copy'], link: ['پیوند نامعلوم', 'amber', 'question'], gatewayOnly: ['فقط ثبت درگاهی', 'violet', 'link'] };
  var flagChips = function (r) { var f = r.flags.map(function (k) { return pill(FLAGS[k][0], FLAGS[k][1], FLAGS[k][2]); }); if (r.stale && !X.localOf(r).reloaded) f.push(pill('تعارض: تغییر کرده', 'red', 'swap')); return f.length ? f.join(' ') : '<span class="muted">—</span>'; };
  var cap = function (v, f) { return f ? '' : ''; };

  /* ================= Review Queue ================= */
  V.rev = function () {
    var all = M.queue.map(X.eff), q = st.rq;
    var F = { needs: function (r) { return r.review === 'pending'; }, receipt: function (r) { return r.ev && r.ev.k === 'receipt'; }, online: function (r) { return r.ev && r.ev.k === 'gateway'; }, approved: function (r) { return r.review === 'approved'; }, rejected: function (r) { return r.review === 'rejected'; }, issues: function (r) { return r.flags.length || (r.stale && !X.localOf(r).reloaded); } };
    var n = function (k) { return all.filter(F[k]).length; };
    var list = all.filter(F[q]), pend = all.filter(F.needs), invN = {}; pend.forEach(function (r) { invN[r.inv] = 1; });
    var by = { toman: 0, rial: 0 }, unkU = 0; pend.forEach(function (r) { if (r.unit) by[r.unit] += r.claimed; else unkU++; });
    var kpis = [
      { label: 'مرحله در انتظار بررسی', value: fa(pend.length), color: 'orange', meaning: 'واحد شمارش: مرحلهٔ قابل بررسی. با تعداد فاکتور جمع یا مقایسه نمی‌شود.', basis: 'وضعیت فعلی' },
      { label: 'فاکتور دارای مرحلهٔ در انتظار', value: fa(Object.keys(invN).length), color: 'slate', meaning: 'واحد شمارش: فاکتور؛ یک فاکتور ممکن است چند مرحله داشته باشد. با شمارش مرحله هم‌واحد نیست.', basis: 'وضعیت فعلی' },
      { label: 'نیازمند توجه (ناهمخوانی/تعارض)', value: fa(pend.filter(F.issues).length), color: 'red', keep: true, meaning: 'مدرک ناموجود، اختلاف مبلغ، واحد نامشخص، پیوند نامعلوم، تکراری احتمالی یا تعارض؛ این موارد تا حل شدن قابل تأیید نیستند.', basis: 'وضعیت فعلی' },
      { label: 'مبلغ ادعاشده در انتظار (به تفکیک واحد)', value: '<span class="mt-v">' + fa(num(by.toman)) + ' تومان</span><small class="of">' + fa(num(by.rial)) + ' ریال · ' + fa(unkU) + ' مورد واحد نامشخص (در جمع نیست)</small>', color: 'amber', meaning: 'ادعاشده است، نه وصول معتبر. واحدها جمع نمی‌شوند و واحد نامشخص صفر فرض نمی‌شود.', basis: 'وضعیت فعلی' }
    ];
    var qs = h.queues([{ id: 'needs', label: 'نیازمند بررسی', icon: 'hourglass', n: n('needs'), key: '1', tone: 'orange' }, { id: 'receipt', label: 'رسید بارگذاری‌شده', icon: 'receipt', n: n('receipt'), key: '2', tone: 'blue' }, { id: 'online', label: 'پرداخت درگاهی ثبت‌شده', icon: 'link', n: n('online'), key: '3', tone: 'slate' }, { id: 'approved', label: 'تأییدشده', icon: 'checkCircle', n: n('approved'), key: '4', tone: 'green' }, { id: 'rejected', label: 'ردشده (پیگیری اصلاح)', icon: 'xCircle', n: n('rejected'), key: '5', tone: 'red' }, { id: 'issues', label: 'ناهمخوانی/تعارض', icon: 'alert', n: n('issues'), key: '6', tone: 'amber' }], q, 'data-rq');
    var sel = X.keys(st.sel).filter(function (id) { return X.q(id) && X.eff(X.q(id)).review === 'pending'; });
    var rows = list.map(function (r) {
      var ev = r.ev ? pill(X.EVK[r.ev.k].label, X.EVK[r.ev.k].tone, X.EVK[r.ev.k].icon) + '<span class="cell-sub mono">' + esc(r.ev.ref) + ' ' + esc(r.ev.ver) + '</span><span class="cell-sub">' + esc(r.ev.at) + '</span>' : pill('مدرک ناموجود', 'amber', 'dashed') + '<span class="cell-sub">گم‌شده یا بارگذاری نشده</span>';
      var chk = r.review === 'pending' ? h.cbx('data-sel="sel" data-id="' + r.id + '"', st.sel[r.id], 'انتخاب مرحله ' + r.id) : '';
      return '<tr data-row="rs:' + r.id + '" tabindex="-1"><td class="col-sel">' + chk + '</td><td><b class="mono">' + r.inv + '</b><span class="cell-sub mono">' + r.id + ' · ' + (r.cs || 'Case نامعلوم') + '</span><span class="idc">' + X.lk(r.lk) + '</span></td>' +
        '<td class="col-opt-wide wrap"><b>' + esc(r.cust) + '</b><span class="cell-sub">' + esc(r.seller) + ' · تیم ' + esc(r.team) + '</span></td><td class="wrap"><b>مرحله ' + fa(r.stage[0]) + ' از ' + fa(r.stage[1]) + '</b><span class="cell-sub">' + esc(r.purpose) + '</span></td><td class="wrap">' + ev + '<div class="flagrow">' + flagChips(r) + '</div></td>' +
        '<td class="wrap"><span class="cell-sub">ادعا</span>' + X.amt(r.claimed, r.unit) + '<span class="cell-sub">اعتبارسنجی‌شده</span>' + (r.valid != null ? X.amt(r.valid, r.unit) : '<span class="muted">اعتبارسنجی نشده</span>') + '</td>' +
        '<td class="col-opt wrap"><span class="cell-sub">کل: <b>' + X.amtT(r.total, r.unit) + '</b></span><span class="cell-sub">وصول‌شده: ' + X.amtT(r.paid, r.unit) + '</span><span class="cell-sub">باقی‌مانده: ' + X.amtT(r.rem, r.unit) + '</span></td>' +
        '<td class="wrap">' + X.revPill(r.review) + (r.prior.length ? '<span class="cell-sub">' + fa(r.prior.length) + ' تصمیم قبلی · ' + esc(r.prior[r.prior.length - 1].who) + '</span>' : '<span class="cell-sub">بدون تصمیم قبلی</span>') + '</td>' +
        '<td class="col-opt-wide wrap"><b>' + esc(r.rev) + '</b><span class="cell-sub">گام بعد: ' + esc(r.next) + '</span></td><td class="col-actions">' + open('rs', r.id, 'بررسی') + '</td></tr>';
    }).join('');
    var bar = sel.length ? X.bulkbar(fa(sel.length) + ' مرحله انتخاب شد', 'واحد شمارش: مرحله · هر ردیف جدا بررسی می‌شود', '<button type="button" class="btn" data-act="clear-sel">پاک کردن انتخاب</button><button type="button" class="btn btn-soft" data-act="open-bulkrev">' + ic('eye') + 'بررسی گروهی (فقط‌خواندنی)</button>' + X.guardBtn('approve', 'btn-primary', 'open-bulkapp', 'تأیید گروهی (مشروط)', 'check') + '<button type="button" class="btn tip" aria-disabled="true" data-tip="رد گروهی نیازمند اعتبارسنجی است: دلیل و مسئول اصلاح هر مرحله جدا لازم است؛ وجود مسیر فعلی فرض نشده است.">' + ic('xCircle') + 'رد گروهی (نیازمند اعتبارسنجی)</button><button type="button" class="btn tip" aria-disabled="true" data-tip="خروجی گروهی: مشروط؛ اختیار جدا، محدوده، ستون‌ها و ممیزی لازم است.">' + ic('download') + 'خروجی (مشروط)</button>') : '';
    return h.pageHead({ title: 'صف بررسی مالی', sub: 'واحد: مرحلهٔ پرداخت قابل بررسی (مرتبط با فاکتور) · مدرک ≠ تأیید ≠ وصول کامل ≠ استحقاق ≠ تسویه · هیچ تصمیمی خودکار نیست', kpis: kpis, fresh: X.freshPart('صف'), scope: 'دامنهٔ مالی سازمان به‌معنی دسترسی به همهٔ اطلاعات بانکی یا شخصی نیست' }) + X.banners('rev') +
      '<section class="panel main has-bulk" aria-labelledby="rv-h">' + ph('receipt', 'مراحل پرداخت', 'rv-h', X.basis('snap') + X.grain('مرحلهٔ قابل بررسی'), 'فیلترها هم‌پوشان‌اند، افراز نیستند') + qs + '<p class="ind-note">' + ic('info') + ' «رسید بارگذاری‌شده» و «تأییدشده» ممکن است یک مرحله را هم‌زمان شامل شوند؛ شمارش‌ها قابل جمع نیستند. «پرداخت درگاهی ثبت‌شده» تأیید مالی نیست.</p>' + h.toolbar('جستجو: شناسهٔ فاکتور، مرحله یا Case (فقط صف بارگذاری‌شده)…', [], '') + bar +
      table('صف بررسی مالی', 'هر ردیف یک مرحلهٔ پرداخت قابل بررسی مرتبط با فاکتور؛ ستون‌ها: فاکتور و پیوند، مشتری، مرحله، مدرک و ناهمخوانی‌ها، مبلغ، وضعیت بررسی و مسئول.', '<th class="col-sel"><span class="sr">انتخاب</span></th><th>فاکتور · پیوند</th><th class="col-opt-wide">مشتری · فروشنده</th><th>مرحله</th><th>مدرک و ناهمخوانی</th><th>مبلغ</th><th class="col-opt">کل · وصول · باقی‌مانده</th><th>بررسی مالی</th><th class="col-opt-wide">بازبین · گام بعد</th><th class="col-actions"><span class="sr">اقدام</span></th>', rows) + h.tfoot(list.length, all.length) + '</section>';
  };

  /* ================= Reconciliation / Refund ================= */
  V.rec = function () {
    var q = st.recq, qs = h.queues([{ id: 'issues', label: 'مسائل تطبیق', icon: 'swap', n: M.issues.length, key: '1', tone: 'orange' }, { id: 'refunds', label: 'استرداد (مشروط)', icon: 'repeat', n: M.refunds.length, key: '2', tone: 'slate' }], q, 'data-recq'), body;
    if (q === 'issues') {
      var sel = X.keys(st.isel);
      var rows = M.issues.map(function (i) {
        var c = X.ISS[i.cls], s = X.ISST[i.state];
        return '<tr data-row="iss:' + i.id + '" tabindex="-1"><td class="col-sel">' + h.cbx('data-sel="isel" data-id="' + i.id + '"', st.isel[i.id], 'انتخاب مسئله ' + i.id) + '</td><td class="wrap"><div class="exc-cls">' + pill(c[0], i.sev, c[1]) + '</div><b class="exc-subj">' + esc(i.title) + '</b></td><td class="col-opt wrap"><span class="mono">' + esc(i.ent[1]) + '</span><span class="cell-sub">' + esc(i.ent[2]) + '</span></td><td class="col-opt wrap">' + esc(i.owner) + '</td><td>' + X.lk(i.conf) + '</td><td>' + pill(s[0], s[1], s[2]) + '</td><td class="col-opt-wide wrap"><span class="allowed">' + ic('eye') + esc(i.allowed[0]) + '</span></td><td class="col-actions">' + open('iss', i.id, 'شواهد') + '</td></tr>';
      }).join('');
      body = h.banner('locked', '<b>فضای تشخیص و ارجاع است، نه «رفع همه».</b> هر مسئله شواهد، اثر مالی، مالک، اقدام مجاز مالی، اقدام ممنوع، گام بعدی و مدرک حل دارد. بستن ظاهری استثنا حل‌شدن نیست و تطبیق گروهی فقط تشخیص خواندنی است.', '') +
        (sel.length ? X.bulkbar(fa(sel.length) + ' مسئله برای تشخیص گروهی', 'فقط خواندنی · هیچ داده یا مانده‌ای تغییر نمی‌کند', '<button type="button" class="btn" data-act="clear-isel">پاک کردن انتخاب</button><button type="button" class="btn btn-primary" data-act="open-diag">' + ic('eye') + 'تشخیص گروهی (' + fa(sel.length) + ')</button><button type="button" class="btn tip" aria-disabled="true" data-tip="اصلاح گروهی / «رفع همه» برای مسائل ناهمگن مالی مجاز نیست.">' + ic('ban') + 'اصلاح گروهی (غیرمجاز)</button>') : '') +
        table('مسائل تطبیق مالی', 'هر ردیف یک مسئله با نوع، موجودیت متأثر، مالک حوزه، اطمینان پیوند، وضعیت و اقدام مجاز مالی.', '<th class="col-sel"><span class="sr">انتخاب</span></th><th>مسئله</th><th class="col-opt">موجودیت</th><th class="col-opt">مالک / حوزه</th><th>اطمینان</th><th>وضعیت</th><th class="col-opt-wide">اقدام مجاز مالی</th><th class="col-actions"><span class="sr">اقدام</span></th>', rows) + h.tfoot(M.issues.length, M.issues.length);
    } else {
      var rr = M.refunds.map(function (r) {
        return '<tr data-row="ref:' + r.id + '" tabindex="-1"><td><b class="mono">' + r.id + '</b><span class="cell-sub mono">' + r.inv + '</span></td><td>' + X.rfPill(r.st) + (r.invCancelled ? '<span class="cell-sub">' + ic('alert') + ' فاکتور لغو شده · لغو ≠ استرداد</span>' : '') + '</td><td class="wrap"><span class="cell-sub">درخواستی</span>' + X.amt(r.req, r.unit) + '<span class="cell-sub">انجام‌شده (گزارش‌شده/ثبت‌شده)</span>' + X.amt(r.done, r.unit, 'نتیجه اجرا نامعلوم است') + '</td><td class="col-opt wrap">' + esc(r.exec) + '</td><td class="col-opt wrap">' + (r.proof ? esc(r.proof) : '<span class="neg">مدرک موجود نیست</span>') + '</td><td class="col-opt-wide">' + esc(r.rev) + '</td><td class="col-actions">' + open('ref', r.id, 'بررسی') + '</td></tr>';
      }).join('');
      body = h.banner('locked', '<b>استرداد مشروط است (OPD-04).</b> نوع مدرک بانکی، تأییدکنندهٔ مجاز و اجرای استرداد تعریف نشده است. درخواست، گزارش اجرا و تأیید اجرا سه چیز جدا هستند؛ فاکتور لغو‌شده یعنی پول بازگشته نیست. استرداد گروهی مجاز نیست.', '') +
        table('استردادها', 'هر ردیف یک درخواست استرداد مرتبط با فاکتور و پرداخت اصلی؛ وضعیت، مبالغ درخواستی و انجام‌شده، منبع اجرا و مدرک.', '<th>استرداد</th><th>وضعیت</th><th>مبالغ</th><th class="col-opt">منبع اجرا</th><th class="col-opt">مدرک</th><th class="col-opt-wide">بازبین</th><th class="col-actions"><span class="sr">اقدام</span></th>', rr) + h.tfoot(M.refunds.length, M.refunds.length);
    }
    return h.pageHead({ title: 'تطبیق و استرداد', sub: 'مسئله‌ها با مالک و مدرک حل · استرداد مشروط و بدون ادعای اجرای بانکی · «رفع همه» وجود ندارد', fresh: X.freshPart('مسائل'), scope: X.zone('cond', 'تشخیص خواندنی') }) + X.banners('rec') + '<section class="panel main has-bulk" aria-labelledby="rc-h">' + ph('swap', 'تطبیق و استرداد', 'rc-h', '', '') + qs + body + '</section>';
  };

  /* ================= Ledger (audit oriented, read-only) ================= */
  V.led = function () {
    var f = st.lq, dm = st.ldom;
    var list = M.ledger.filter(function (t) { return (dm === 'all' || t.dom === dm) && (f === 'all' ? true : f === 'commit' ? t.st === 'committed' : f === 'notc' ? t.st !== 'committed' : (t.of || (t.revs && t.revs.length))); });
    var kpis = [
      { label: 'تراکنش ثبت‌شده', value: fa(M.ledger.filter(function (t) { return t.st === 'committed'; }).length), color: 'green', meaning: 'واحد: تراکنش با شناسه. جمع مبلغ عمداً نشان داده نمی‌شود؛ دفتر کل از جمع‌های رابط نتیجه‌گیری نمی‌شود.', basis: 'رویداد در بازه' },
      { label: 'رویداد جبرانی (معکوس/اصلاح)', value: fa(M.ledger.filter(function (t) { return t.of; }).length), color: 'teal', meaning: 'رویدادهای جدیدی که به رویداد اصلی وصل‌اند؛ تاریخ بازنویسی نمی‌شود.', basis: 'رویداد در بازه' },
      { label: 'نیت بدون تراکنش (پیش‌نمایش/نامعلوم)', value: fa(M.ledger.filter(function (t) { return t.st !== 'committed'; }).length), color: 'amber', keep: true, meaning: 'تراکنش ثبت‌شده نیستند و در هیچ مانده‌ای شمرده نمی‌شوند.', basis: 'رویداد در بازه' },
      { label: 'شبه‌یتیم', value: fa(M.ledger.filter(function (t) { return t.orphan; }).length), color: 'red', meaning: 'رابطه با فاکتور اثبات نشده؛ حذف یا اتصال حدسی ممنوع است.', basis: 'وضعیت فعلی' }
    ];
    var qs = h.queues([{ id: 'all', label: 'همه', icon: 'rows', n: M.ledger.length, key: '1' }, { id: 'commit', label: 'ثبت‌شده', icon: 'checkCircle', n: M.ledger.filter(function (t) { return t.st === 'committed'; }).length, key: '2', tone: 'green' }, { id: 'notc', label: 'پیش‌نمایش / نامعلوم', icon: 'question', n: M.ledger.filter(function (t) { return t.st !== 'committed'; }).length, key: '3', tone: 'amber' }, { id: 'corr', label: 'اصلاح و معکوس‌سازی', icon: 'repeat', n: M.ledger.filter(function (t) { return t.of || (t.revs && t.revs.length); }).length, key: '4', tone: 'slate' }], f, 'data-lq');
    var dchips = '<div class="ex-bar" role="group" aria-label="دامنهٔ دفتر">' + [['all', 'همهٔ دامنه‌ها'], ['collection', 'وصول فاکتور'], ['wallet', 'کیف پول کمیسیون']].map(function (c) { return '<button type="button" class="chip-btn' + (dm === c[0] ? ' on' : '') + '" data-ldom="' + c[0] + '" aria-pressed="' + (dm === c[0]) + '">' + c[1] + '</button>'; }).join('') + '<span class="of">دو دامنهٔ مستقل؛ بدون مبنا یک مانده واحد ساخته نمی‌شود.</span></div>';
    var rows = list.map(function (t) {
      var link = t.of ? '<span class="cell-sub">' + ic('link') + ' جبران/اصلاح ' + esc(t.of) + '</span>' : t.revs ? '<span class="cell-sub">' + ic('link') + ' اصلاح‌شده با ' + t.revs.map(esc).join('، ') + '</span>' : '';
      return '<tr data-row="tx:' + (t.id === '—' ? t.key : t.id) + '" tabindex="-1"' + (t.of ? ' class="row-corr"' : '') + '><td><b class="mono">' + esc(t.id) + '</b><span class="cell-sub mono key">' + esc(t.key) + '</span></td><td class="wrap">' + esc(t.acct) + '<span class="cell-sub">' + (t.dom === 'wallet' ? 'کیف پول کمیسیون' : 'وصول فاکتور') + '</span></td><td>' + pill(t.dir === 'C' ? 'بستانکار' : 'بدهکار', t.dir === 'C' ? 'teal' : 'orange', t.dir === 'C' ? 'plus' : 'arrow') + '</td><td>' + X.amt(t.amt, t.unit) + '</td><td class="col-opt wrap">' + esc(t.rel[0]) + '<span class="cell-sub">' + esc(t.rel[1]) + '</span>' + link + '</td><td class="col-opt-wide wrap">' + esc(t.actor) + '</td><td class="col-opt wrap"><span class="cell-sub">اثر مالی</span>' + esc(t.eff) + '<span class="cell-sub">ثبت</span>' + esc(t.posted) + '</td><td>' + X.lgPill(t.st) + (t.orphan ? '<span class="cell-sub">' + ic('unlink') + ' شبه‌یتیم</span>' : '') + '</td><td class="col-actions">' + open('tx', t.id === '—' ? t.key : t.id, 'ردیابی') + '</td></tr>';
    }).join('');
    return h.pageHead({ title: 'دفتر کل', sub: 'کاوشگر ممیزی، نه ویرایشگر مانده · رویداد جدید جبران می‌کند و رویداد قدیمی حذف یا بازنویسی نمی‌شود · نمایش فقط‌خواندنی است', kpis: kpis, fresh: X.freshPart('تراکنش‌ها'), scope: 'فیلدهای دفتر کل تابع اختیار مشاهدهٔ جدا هستند' }) + X.banners('led') +
      h.banner('locked', '<b>خواندنِ خالص (F07):</b> رندر این صفحه هیچ ثبت خودکار قدیمی کیف پول را اجرا یا شبیه‌سازی نمی‌کند. اگر اثر موتور مربوط باشد فقط وضعیت/زمینه نشان داده می‌شود؛ ثبت دفتر کل جدا و صریح است.', '') +
      '<section class="panel main" aria-labelledby="lg-h">' + ph('wallet', 'تراکنش‌ها و نیت‌ها', 'lg-h', X.basis('range') + X.grain('تراکنش'), 'زمان اثر مالی و زمان ثبت جدا هستند') + qs + dchips +
      table('کاوشگر دفتر کل', 'هر ردیف یک تراکنش یا نیت: شناسه و کلید کسب‌وکار، حساب و دامنه، جهت، مبلغ و واحد، رابطه، عامل، زمان اثر و زمان ثبت، وضعیت ثبت.', '<th>تراکنش · کلید</th><th>حساب · دامنه</th><th>جهت</th><th>مبلغ</th><th class="col-opt">رابطه</th><th class="col-opt-wide">عامل</th><th class="col-opt">زمان اثر · ثبت</th><th>وضعیت ثبت</th><th class="col-actions"><span class="sr">اقدام</span></th>', rows) + h.tfoot(list.length, M.ledger.length) + '</section>';
  };

  /* ================= Rules / Runs / Posting ================= */
  V.run = function () {
    var q = st.runq, runs = M.runs.map(X.runView);
    var cnt = {}; runs.forEach(function (r) { cnt[r.st] = (cnt[r.st] || 0) + 1; });
    var qs = h.queues([{ id: 'runs', label: 'اجراها', icon: 'layers', n: runs.length, key: '1', tone: 'orange' }, { id: 'rules', label: 'قوانین (فقط‌خواندنی)', icon: 'file', n: M.rules.length, key: '2' }], q, 'data-runq'), body;
    if (q === 'runs') {
      var rib = '<ol class="lc-rib" aria-label="چرخهٔ عمر اجرا">' + X.RUNORD.map(function (k) { var d = X.RUN[k]; return '<li class="' + (cnt[k] ? '' : 'zero') + '">' + pill(d.label, d.tone, d.icon) + '<b>' + fa(cnt[k] || 0) + '</b></li>'; }).join('') + '</ol>';
      var rows = runs.map(function (r) {
        var c = r.cov;
        var cv = c ? (c.processed == null ? '<span class="neg">' + ic('question') + ' پوشش نامعلوم</span>' : '<b>' + fa(num(c.posted + c.existing)) + '</b> از ' + fa(num(c.intended)) + (c.unprocessed ? '<span class="cell-sub neg">' + ic('alert') + ' ' + fa(num(c.unprocessed)) + ' پردازش‌نشده</span>' : '') + (c.failed ? '<span class="cell-sub neg">' + fa(num(c.failed)) + ' ناموفق</span>' : '')) : '<span class="muted">محاسبه نشده</span>';
        return '<tr data-row="run:' + r.id + '" tabindex="-1"><td><b class="mono">' + r.id + '</b><span class="cell-sub">' + esc(r.name) + '</span></td><td class="col-opt wrap">' + esc(r.engL) + '</td><td>' + X.runPill(r.st) + '</td><td class="wrap">' + cv + '</td><td class="col-opt wrap">' + (r.appr ? esc(r.appr.by) + '<span class="cell-sub">' + esc(r.appr.at) + '</span>' : '<span class="muted">تأیید نشده</span>') + '</td><td class="col-opt-wide wrap">' + esc(X.runNext(r)) + '</td><td class="col-actions">' + open('run', r.id, 'کنسول') + '</td></tr>';
      }).join('');
      body = h.banner('locked', '<b>تأیید ≠ ثبت.</b> تولید پیش‌نمایش خودش فراداده می‌نویسد (خواندن خالص نیست). تأیید اجرا اعتبار کیف پول نمی‌سازد؛ ثبت فقط با اجرای جدا و روی همان snapshot، با کلید کسب‌وکار یکتا. «تفکیک تأییدکننده و ثبت‌کننده» تعریف نشده (OPD-08) و فرض هم نشده است.', '') + rib +
        table('اجراهای کمیسیون و ثبت', 'هر ردیف یک اجرا: وضعیت چرخهٔ عمر، موتور، پوشش (ثبت‌شده از کل نیت‌شده)، تأییدکننده و گام بعد.', '<th>اجرا</th><th class="col-opt">موتور</th><th>وضعیت</th><th>پوشش</th><th class="col-opt">تأییدکننده</th><th class="col-opt-wide">گام بعد</th><th class="col-actions"><span class="sr">اقدام</span></th>', rows) + h.tfoot(runs.length, runs.length);
    } else {
      var rr = M.rules.map(function (r) { return '<tr><td><b class="mono">' + r.id + '</b></td><td class="wrap">' + esc(r.purpose) + '</td><td class="col-opt">' + esc(r.eng) + '</td><td>' + (r.on ? pill('فعال', 'teal', 'checkCircle') : pill('غیرفعال/نامشخص', 'slate', 'ban')) + '</td><td class="col-opt">' + esc(r.from) + '</td><td class="col-opt wrap">' + esc(r.src) + '</td></tr>'; }).join('');
      body = h.banner('locked', '<b>نمای فقط‌خواندنی.</b> ایجاد یا فعال/غیرفعال کردن قانون امروز با همان دروازهٔ مشاهده انجام می‌شود (F06)؛ هدف: اختیار جدا. نرخ‌ها از جبران خدمات مؤثر منابع انسانی می‌آیند و اینجا ویرایش نمی‌شوند. مسیر غیرفاکتور (OPD-09) تعریف نشده است.', '') +
        table('قوانین', 'قوانین کمیسیون: هدف، موتور، وضعیت، تاریخ اثر و منبع نرخ.', '<th>قانون</th><th>هدف</th><th class="col-opt">موتور</th><th>وضعیت</th><th class="col-opt">تاریخ اثر</th><th class="col-opt">منبع نرخ</th>', rr, 'no-cursor') + '<div class="resp-line cat-pad">' + X.guardBtn('runapprove', 'btn-soft', 'x', 'ایجاد / تغییر قانون', 'edit', true) + '</div>';
    }
    return h.pageHead({ title: 'قوانین، اجراها و ثبت', sub: 'پیش‌نویس ← پیش‌نمایش ← تأییدشده ← ثبت ← ثبت‌شده / ناقص / ناموفق / نامعلوم ← تطبیق‌شده · هر آیتم نتیجهٔ جدا دارد', fresh: X.freshPart('اجراها'), scope: 'موتور فعال و مسیر استحقاق باید در زمینه آشکار باشد؛ مقدار پرچم نصب‌شده تأیید نشده است' }) + X.banners('run') + '<section class="panel main" aria-labelledby="rn-h">' + ph('layers', 'اجراها و ثبت', 'rn-h', X.basis('snap') + X.grain('اجرا / آیتم'), '') + qs + body + '</section>';
  };
  X.runView = function (r) { var l = st.runLocal[r.id]; if (!l) return r; var o = {}; for (var k in r) o[k] = r[k]; for (k in l) o[k] = l[k]; return o; };
  X.runNext = function (r) { return { draft: 'تولید پیش‌نمایش (نوشتن فراداده)', preview: 'بازبین مجاز اجرا', approved: 'مجری مجاز ثبت', posting: 'منتظر نتیجه', posted: 'پایان — شواهد تراکنش موجود', partial: 'ادامهٔ دنباله با همان snapshot', failed: 'بررسی علت؛ اثر صفر اثبات شده', unknown: 'مالی: جستجوی تراکنش/کلید', reconciled: 'پایان؛ دنبالهٔ اثبات‌شدهٔ ثبت‌نشده در صورت وجود' }[r.st]; };

  /* ================= Reports ================= */
  V.rep = function () {
    return h.pageHead({ title: 'گزارش‌ها', sub: 'هفت گزارش با واحد شمارش مستقل · هر گزارش محدوده، مبنای زمانی، واحد، منبع، تازگی، پوشش و وضعیت تطبیق را نشان می‌دهد · واحدهای ناسازگار جمع نمی‌شوند', fresh: X.freshPart('گزارش‌ها'), scope: X.zone('normal', 'خواندنی') }) + X.banners('rep') +
      '<section class="panel main" aria-labelledby="rp-h">' + ph('chart', 'فهرست گزارش‌ها', 'rp-h', '', '') + '<div class="rep-grid cat-pad">' + M.reports.map(function (r) {
        return '<button type="button" class="rep-card" data-act="open-rep:' + r.id + '"><span class="rep-ico">' + ic('chart') + '</span><b>' + esc(r.name) + '</b><span class="rep-tags">' + X.basis(r.basis) + X.grain(r.grain) + X.cov(r.fresh === 'stale' ? 'stale' : r.cov) + '</span><span class="rep-go">زمینهٔ اعتماد و شاخص‌ها ' + ic('arrowL') + '</span></button>';
      }).join('') + '</div></section>';
  };

  /* ================= Audit / History ================= */
  V.aud = function () {
    var rows = M.audit.map(function (a) { return '<tr data-row="audit:' + a.id + '" tabindex="-1"><td class="col-opt">' + esc(a.at) + '</td><td class="wrap"><b>' + esc(a.kind) + '</b><span class="cell-sub">' + esc(a.actor) + '</span></td><td class="col-opt mono">' + esc(a.inv) + '</td><td class="col-opt-wide mono">' + esc(a.run) + '</td><td class="wrap">' + esc(a.ba) + '</td><td>' + pill(a.res, a.res === 'موفق' ? 'green' : 'amber', a.res === 'موفق' ? 'checkCircle' : 'question') + '</td><td class="col-actions">' + open('audit', a.id, 'جزئیات') + '</td></tr>'; }).join('');
    return h.pageHead({ title: 'ممیزی و تاریخچه', sub: 'تاریخچهٔ مرتبط مرور، مدرک، اجرا، تراکنش و منبع · فقط حداقل فیلد لازم · عامل انسانی/سیستمی و زمان اثر/زمان ثبت جدا', fresh: X.freshPart('ممیزی'), scope: X.zone('normal', 'خواندنی') }) + X.banners('aud') +
      '<section class="panel main" aria-labelledby="au-h">' + ph('history', 'رویدادها', 'au-h', X.basis('range'), 'ماندگاری ممیزی تأیید نشده') + table('رویدادهای ممیزی', 'هر ردیف یک رویداد با زمان، عامل، موجودیت، تغییر، نتیجه و ارجاع.', '<th class="col-opt">زمان</th><th>رویداد · عامل</th><th class="col-opt">فاکتور</th><th class="col-opt-wide">اجرا</th><th>قبل ← بعد</th><th>نتیجه</th><th class="col-actions"><span class="sr">اقدام</span></th>', rows) + h.tfoot(M.audit.length, M.audit.length) + '</section>';
  };

  /* ================= Configuration & action-aware permission contract ================= */
  V.cfg = function () {
    var full = st.perm === 'full';
    var seg = '<div class="seg" role="group" aria-label="شبیه‌سازی نقش (فقط دمو)"><button type="button" data-act="perm:full" aria-pressed="' + full + '">طرح هدف با اختیار فرضی مالی</button><button type="button" data-act="perm:view" aria-pressed="' + !full + '">فقط مشاهده</button></div>';
    var rows = M.perms.map(function (p) {
      var g = X.can(p.k), s = !p.write ? (full ? ['مجاز (خواندن)', 'teal', 'eye'] : ['مجاز (خواندن)', 'teal', 'eye']) : g.ok ? ['مجاز (طرح هدف)', 'teal', 'checkCircle'] : st.perm === 'view' ? ['غیرمجاز: فقط مشاهده', 'slate', 'ban'] : p.onlyAdmin ? ['فقط مدیر سیستم', 'slate', 'lock'] : ['مشروط: ' + p.cond, 'amber', 'alert'];
      return '<tr><td class="wrap"><b class="mono">' + esc(p.concept) + '</b><span class="cell-sub">' + esc(p.note) + '</span></td><td>' + pill(p.write ? 'نوشتن / اجرا' : 'خواندن', p.write ? 'orange' : 'blue', p.write ? 'edit' : 'eye') + '</td><td class="col-opt wrap">' + esc(p.cur) + (p.f06 ? '<span class="cell-sub neg">' + ic('alert') + ' F06: مشاهده برای نوشتن حساس هم کافی بوده است</span>' : '') + '</td><td>' + pill(s[0], s[1], s[2]) + (!g.ok && p.write ? '<span class="cell-sub">' + esc(g.why) + '</span>' : '') + '</td></tr>';
    }).join('');
    var bulk = [['بررسی گروهی', 'الزامی', 'teal', 'خواندن و تریاژ؛ تصمیم بدون بررسی هر ردیف نیست'], ['تأیید گروهی', 'مشروط', 'amber', 'حداکثر ۱۰۰ شناسه در مسیر فعلی؛ نتیجهٔ واقعی هر مورد، بدون «همه موفق»'], ['رد گروهی', 'نیازمند اعتبارسنجی', 'slate', 'دلیل و مسئول اصلاح هر مورد؛ وجود مسیر فعلی فرض نشده'], ['استرداد گروهی', 'مجاز نیست', 'red', 'OPD-04 حل نشده؛ اثر بانکی و مدرک هر پرداخت لازم است'], ['تطبیق گروهی', 'فقط تشخیص', 'amber', 'اصلاح گروهی / «رفع همه» غیرمجاز'], ['ثبت گروهی', 'مشروط', 'amber', 'snapshot تأییدشده، کلید یکتا، نتیجهٔ هر آیتم؛ تکرار نامعلوم ممنوع'], ['خروجی گروهی', 'مشروط', 'amber', 'اختیار جدا؛ محدوده/ستون/تعداد دقیق و کامل بودن منبع']];
    var brow = bulk.map(function (b) { return '<tr><td><b>' + b[0] + '</b></td><td>' + pill(b[1], b[2], b[2] === 'red' ? 'ban' : b[2] === 'teal' ? 'checkCircle' : 'alert') + '</td><td class="wrap">' + b[3] + '</td></tr>'; }).join('');
    return h.pageHead({ title: 'پیکربندی و اختیار اقدام‌ها', sub: 'قرارداد هدف action-aware · نام‌ها مفهومی‌اند و مجوز واقعی (WordPress) ساخته نشده · دیدن ≠ اجرا · سلسله‌مراتب فروش اختیار مالی نمی‌دهد · ویرایشگر قابلیت وجود ندارد', fresh: X.freshPart('قرارداد'), scope: X.zone('cond', 'نمایش، نه ویرایش') }) + X.banners('cfg') +
      '<section class="panel main" aria-labelledby="pm-h">' + ph('shield', 'ماتریس اقدام و اختیار', 'pm-h', '', 'F06: دروازهٔ مشاهده برای برخی نوشتن‌ها استفاده می‌شود') + '<div class="ex-bar"><b>شبیه‌سازی (فقط دمو):</b>' + seg + '<span class="of">این کلید یک نمایش است؛ نقش یا مجوزی را تغییر نمی‌دهد.</span></div>' +
      table('ماتریس اختیار', 'برای هر اقدام مفهومی: نوع، شواهد فعلی و وضعیت اجرا تحت نقش شبیه‌سازی‌شده.', '<th>اقدام مفهومی</th><th>نوع</th><th class="col-opt">وضعیت فعلی (شواهد)</th><th>وضعیت اجرا</th>', rows, 'no-cursor') + '</section>' +
      '<section class="panel" aria-labelledby="bk-h">' + ph('layers', 'اقدام‌های گروهی: خط پایه', 'bk-h', '', '') + table('اقدام‌های گروهی', 'هر اقدام گروهی با طبقه‌بندی و قرارداد.', '<th>اقدام</th><th>طبقه‌بندی</th><th>قرارداد</th>', brow, 'no-cursor') + '</section>';
  };

  /* ================= Advanced · Maintenance / recovery ================= */
  V.mnt = function () {
    var cards = M.maint.map(function (m) {
      return '<article class="mnt-card' + (m.readOnly ? '' : ' mnt-lock') + '" aria-labelledby="' + m.id + '-t"><div class="mnt-h">' + ic(m.readOnly ? 'eye' : 'lock') + '<b id="' + m.id + '-t">' + esc(m.name) + '</b>' + (m.readOnly ? pill('خواندنی', 'teal', 'eye') : pill('حساس · محدود', 'red', 'lock')) + '</div>' +
        X.dl([['اثر مالی ممکن', esc(m.fx)], ['اختیار لازم', esc(m.auth)], ['چرا اینجا اجرا نمی‌شود', esc(m.why)]]) + '<div class="resp-line">' + (m.readOnly ? '<button type="button" class="btn btn-soft btn-sm" data-act="diag-ro">' + ic('eye') + 'اجرای تشخیص خواندنی</button>' : '<button type="button" class="btn btn-sm tip" aria-disabled="true" data-tip="این اقدام در رابط روزانهٔ مالی اجرا نمی‌شود؛ بدون اختیار جدا و پیش‌نمایش اثر مجاز نیست.">' + ic('lock') + 'اجرا از این رابط ممکن نیست</button>') + '</div></article>';
    }).join('');
    return h.pageHead({ title: 'نگهداری و بازیابی (پیشرفته)', sub: 'جدا از بررسی روزانه · اقدام‌های پراثر اینجا فقط توضیح داده می‌شوند · «ترمیم» ظاهر نمایش یا «رفع همه» وجود ندارد', fresh: X.freshPart('تشخیص‌ها'), scope: X.zone('restr', 'دسترسی جدا') }) + X.banners('mnt') +
      h.banner('locked', '<b>بازمحاسبه و پرکردن تاریخی «اصلاح نمایش» نیستند؛</b> ممکن است اثر مالی واقعی داشته باشند. این صفحه هیچ‌یک را اجرا نمی‌کند و ویرایشگر مانده یا ترمیم‌همه ندارد.', '') + '<section class="panel main" aria-labelledby="mn-h">' + ph('lock', 'ابزارهای پیشرفته', 'mn-h', X.zone('restr'), '') + '<div class="mnt-grid cat-pad">' + cards + '</div></section>';
  };
})();
