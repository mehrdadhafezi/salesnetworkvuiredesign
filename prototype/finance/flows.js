/* Finance — role layer, part 3: drawers and guarded flows (review item + evidence, Sensitive Financial Action Confirmation, bulk review/approve, refund review (conditional),
   ledger trace, run console + Outcome Unknown/Idempotency context, reconciliation issue, report context, audit event, operation result).
   Prototype only: nothing is saved, posted or sent. Decision ≠ approval of the invoice ≠ entitlement ≠ wallet credit ≠ settlement. Unknown outcomes are reconciled before any retry. */
(function () {
  'use strict';
  var X = window.FINX, C = X.C, M = X.M, h = X.h, st = X.st, D = X.D;
  var esc = h.esc, fa = h.fa, ic = h.ic, pill = h.pill, hint = h.hint, num = h.num;
  var top = X.top, foot = X.foot, sec = X.sec, checks = X.checks;
  var head = function (title, badges, meta) { return '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">' + title + '</h2>' + (badges || '') + '</div>' + (meta ? '<div class="dr-meta">' + meta + '</div>' : '') + '</div>'; };
  var btn = function (cls, act, label, iconName, dis) { return '<button type="button" class="btn ' + cls + '" data-act="' + act + '"' + (dis ? ' disabled aria-disabled="true"' : '') + '>' + (iconName ? ic(iconName) : '') + label + '</button>'; };
  var keepOf = function () { return C.state.drawer ? C.state.drawer.keep : {}; };
  var ok3 = function (s) { return (s || '').trim().length > 2; };
  var NOW = 'همین الان';
  var open = function (kind, id, label, cls) { return '<button type="button" class="btn ' + (cls || 'btn-soft') + ' btn-sm" data-act="open-' + kind + ':' + id + '">' + label + '</button>'; };
  var pushAudit = function (a) { a.id = 'AU-' + (10 + M.audit.length); a.at = NOW; a.actor = M.user.name; M.audit.unshift(a); return a; };
  var ctxLine = function (a, b) { return '<p class="ind-note">' + ic('info') + ' ' + a + (b ? ' ' + b : '') + '</p>'; };

  /* ---------- Review item: evidence panel + independent state layers + decision area ---------- */
  D.rs = function (id) {
    var r0 = X.q(id), r = X.eff(r0), el = X.elig(r0), cur = X.localOf(r0), stale = r0.stale && !cur.reloaded, ev = r.ev, im = X.impact(r);
    var conflict = stale ? h.stateBlock('conflict', 'تعارض: مرحله پس از بارگذاری تغییر کرده است', 'بازبین دیگر (' + esc(r0.stale.by) + ') ' + esc(r0.stale.at) + ' این مرحله را «' + X.REV[r0.stale.res].label + '» کرد. تصمیم جدید روی داده قدیمی ثبت نمی‌شود؛ ابتدا وضعیت فعلی را بازخوانی کنید.', btn('btn-primary', 'reload-stage:' + r.id, 'بازخوانی وضعیت فعلی', 'refresh')) : '';
    var evid = !ev ? h.stateBlock('incomplete', 'مدرک پرداخت موجود نیست', 'مدرک گم‌شده یا بارگذاری‌نشده است. «نبود مدرک» صفر یا تأیید نیست؛ تأیید ممکن نیست و رد با دلیل «مدرک ناموجود» ممکن است. مسئول بعدی: فروشنده (ارائهٔ مدرک).', '') :
      X.dl([['نوع مدرک', pill(X.EVK[ev.k].label, X.EVK[ev.k].tone, X.EVK[ev.k].icon)], ['مرجع · نسخه', '<span class="mono">' + esc(ev.ref) + ' ' + esc(ev.ver) + '</span>'], ['ارسال‌کننده · زمان', esc(ev.by) + ' · ' + esc(ev.at)], ['منبع', esc(ev.src)], ['مبلغ مندرج در مدرک', X.amt(ev.amt, r.unit)], ['مبلغ ادعاشده', X.amt(r.claimed, r.unit)], ['مبلغ اعتبارسنجی‌شده', r.valid != null ? X.amt(r.valid, r.unit) : '<span class="muted">هنوز اعتبارسنجی نشده</span>'], ['واحد', r.unit ? esc(X.unitL(r.unit)) : '<span class="neg">نامشخص (OPD-06)</span>']]) +
      '<div class="resp-line"><button type="button" class="btn btn-soft btn-sm tip" data-act="view-evidence" data-tip="نمایش مدرک تابع اختیار و محافظت جدا است؛ وجود لینک اجازهٔ دانلود نیست.">' + ic('lock') + 'مشاهدهٔ مدرک (محافظت‌شده)</button></div>' +
      (r.flags.indexOf('gatewayOnly') > -1 ? '<div class="note warn inset">' + ic('alert') + '<span><b>فقط ثبت درگاهی:</b> پرداخت درگاهی ثبت‌شده با تأیید مالی یکی نیست و خودکار مرحله را تأیید نمی‌کند.</span></div>' : '') + (r.flags.indexOf('dup') > -1 ? '<div class="note warn inset">' + ic('copy') + '<span><b>پرداخت تکراری احتمالی:</b> ' + esc(r.dupOf) + '</span></div>' : '');
    var who = X.dl([['فروشنده اصلی (انتساب تاریخی)', esc(r.seller)], ['مسئول فعلی پیگیری', 'سرپرست تیم ' + esc(r.team) + ' — با فروشنده اصلی یکی فرض نشده'], ['بازبین مالی فعلی', esc(r.rev)], ['مسئول گام بعد', esc(r.next)], ['گیرندهٔ کمیسیون', 'تعیین‌شدهٔ موتور/سیاست — در این صفحه مشخص نیست'], ['مشتری (حداقل زمینه)', esc(r.cust) + ' — شماره تلفن شناسه مالی نیست']]);
    var link = X.dl([['فاکتور', '<span class="mono">' + r.inv + '</span>'], ['Case / منبع', r.cs ? '<span class="mono">' + r.cs + '</span> · ' + esc(r.src) : 'نامعلوم'], ['اطمینان پیوند', X.lk(r.lk)], ['مرحله', 'مرحله ' + fa(r.stage[0]) + ' از ' + fa(r.stage[1]) + ' · ' + esc(r.purpose)]]);
    var prior = r.prior.length ? X.tl(r.prior.map(function (p) { return [X.REV[p.res].label + ' — ' + p.who, p.at, p.note]; })) + '' : '<p class="ind-note">' + ic('info') + ' تصمیم قبلی برای این مرحله ثبت نشده است.</p>';
    var appImp = checks([['info', 'پس از تأیید (محاسبهٔ نمایشی): وصول‌شده ' + X.amtT(im.np, r.unit) + ' · باقی‌مانده ' + X.amtT(im.nr, r.unit), 'وصول کامل از مجموع وصول معتبر در تلورانس مصوب نتیجه می‌شود؛ تلورانس هنوز تعریف نشده (OPD-06) و اینجا «کامل شد» اعلام نمی‌شود'],
      ['warn', 'اثر خودکار ممکن: موتور هدف‌دار ممکن است هنگام تأیید مرحله اعتبار کمیسیون بسازد', 'این صفحه آن را اجرا یا شبیه‌سازی نمی‌کند؛ مسیر استحقاق باید در زمینه روشن باشد (OPD-09)'],
      ['info', 'گام بعد پس از تأیید: ' + (im.last ? 'قرارداد تکمیل فروش (مرحلهٔ آخر) — تأیید مرحله به‌تنهایی «فروش تکمیل‌شده» نیست' : 'ادامهٔ فروش (فروشنده)'), ''],
      ['info', 'تأیید ≠ ثبت در دفتر کل ≠ تسویه', 'تسویهٔ بانکی در این رابط وجود ندارد']]);
    var rejImp = checks([['info', 'پس از رد: بازگشت به فروشنده برای اصلاح و ارسال دوباره', 'گام بعد: فروشنده (اصلاح)؛ بازبین مالی: ' + M.user.name], ['ok', 'رد = حذف، لغو، استرداد یا بازنشانی تاریخچه نیست', 'مدرک قبلی، تأییدهای قبلی و هویت فاکتور حفظ می‌شود']]);
    var body = (conflict ? '<section class="sec">' + conflict + '</section>' : '') + sec('ابعاد مستقل وضعیت مالی', X.basis('snap'), X.layers(r) + ctxLine('هر بُعد مستقل است: رسید بارگذاری‌شده ≠ پرداخت معتبر ≠ مرحلهٔ تأییدشده ≠ فاکتور کاملاً پرداخت‌شده ≠ استحقاق صادرشده ≠ اعتبار کیف پول ≠ تسویه.'), 'primary') +
      sec('مدرک پرداخت', ev ? pill('نسخهٔ ' + esc(ev.ver), 'slate', 'file') : '', evid) + sec('فاکتور، Case و پیوند', X.lk(r.lk), link) + sec('نقش‌های جدا', '', who) +
      sec('بررسی قبلی و دلیل', '', prior) + sec('صلاحیت تصمیم', '', checks(el.checks)) + (cur.review === 'pending' ? sec('اثر تصمیم', '', '<h4 class="sub-h">اگر تأیید شود</h4>' + appImp + '<h4 class="sub-h">اگر رد شود</h4>' + rejImp) : '');
    var apprBtn = X.guardBtn('approve', 'btn-primary btn-lg', 'rs-approve:' + r.id, 'ادامه: تأیید با بررسی اثر', 'check', el.blocked);
    var rejBtn = X.guardBtn('reject', 'btn-lg', 'rs-reject:' + r.id, 'ادامه: رد با دلیل', 'xCircle', !el.reject);
    var f = cur.review === 'pending' ? foot(apprBtn + rejBtn, null, el.blocked ? 'تأیید مسدود است: ' + (el.reasonTag === 'stale' ? 'بازخوانی لازم' : 'ناهمخوانی باز') : 'پیش‌نمایش ثبت نهایی نیست · هنگام ثبت دوباره بررسی می‌شود') : foot('', null, 'این مرحله در حال حاضر قابل تصمیم‌گیری نیست · ' + X.REV[cur.review].label);
    return top('بررسی مرحلهٔ پرداخت') + head('مرحله ' + esc(r.id) + ' · ' + r.inv, X.revPill(r.review) + X.lk(r.lk), '<span class="mono">' + r.id + '</span><span>' + esc(r.purpose) + '</span>') + '<div class="dr-body">' + body + '</div>' + f;
  };

  /* ---------- Sensitive Financial Action Confirmation (target · current state · amount/unit · affected · resulting state · protected history · actor · reason) ---------- */
  X.sensBlocks = function (sp, keep) {
    var list = function (t, items, tone, icon) { return '<div class="sens-c sens-' + tone + '"><h4>' + ic(icon) + t + '</h4><ul>' + items.map(function (x) { return '<li>' + esc(x) + '</li>'; }).join('') + '</ul></div>'; };
    var need = sp.needsReason, bad = keep.err && (need && !ok3(keep.reason));
    return X.dl([['هدف', esc(sp.target)], ['وضعیت فعلی', esc(sp.current)], ['مبلغ / واحد', sp.amount], ['عامل', esc(M.user.name) + ' · مالی'], ['وضعیت پس از ثبت', esc(sp.resulting)]], 'hcp fin-dl') +
      '<div class="sens-grid">' + list('چه چیزی تغییر می‌کند', sp.changes, 'chg', 'edit') + list('چه چیزی تغییر نمی‌کند / تاریخچه محافظت‌شده', sp.unchanged, 'keep', 'lock') + list('رکوردهای متأثر', sp.affected, 'aff', 'layers') + list('برگشت‌ناپذیر / اثر خودکار', sp.irrev, 'eff', 'alert') + '</div>' +
      '<div class="sens-form"><label class="lbl" for="sens-r">' + esc(sp.reasonLabel || 'دلیل / مبنای تصمیم') + (need ? ' (الزامی)' : ' (اختیاری؛ الزام سیاستی نیازمند اعتبارسنجی)') + '</label><textarea class="input" id="sens-r" rows="2" data-sensreason' + (need ? ' aria-required="true"' : '') + (bad ? ' aria-invalid="true"' : '') + '>' + esc(keep.reason || '') + '</textarea>' + (bad ? '<p class="field-err" role="alert">دلیل لازم است.</p>' : '') +
      '<label class="confirm"><input type="checkbox" class="cbx" data-sensack' + (keep.ack ? ' checked' : '') + '><span>' + esc(sp.ack) + '</span></label></div>';
  };
  D.sens = function () {
    var sp = st.spec, keep = keepOf(), ready = (!sp.needsReason || ok3(keep.reason)) && keep.ack;
    return top('تأیید اقدام مالی حساس') + head(esc(sp.title), X.zone(sp.zone || 'cond'), '<span>' + esc(sp.subject) + '</span>') + '<div class="dr-body">' + (sp.pre || '') + X.sensBlocks(sp, keep) + '</div>' + foot(btn('btn-primary btn-lg', 'commit-sens', sp.commit, 'send', !ready), null, 'پیش‌نمایش ثبت نهایی یا مجوز نیست · هنگام ثبت دوباره بررسی می‌شود · نمایشی: چیزی ثبت نمی‌شود');
  };
  X.openSens = function (sp) { st.spec = sp; C.openDrawer('sens', sp.kind + ':' + (sp.id || 'x'), {}); };
  var tgt = function (r) { return r.inv + ' · مرحله ' + r.id + ' (' + r.purpose + ' ' + r.stage[0] + '/' + r.stage[1] + ')'; };
  // Complete payment/evidence snapshot the reviewer confirmed (financial inputs + evidence identity + relationships + current decision state).
  X.stageSig = function (r0) { var r = X.eff(r0), e = r.ev || {}; return JSON.stringify([r.inv, r.cs, r.lk, r.src, r.cust, r.seller, r.team, r.stage, r.purpose, r.claimed, r.valid, r.unit, r.total, r.paid, r.review, JSON.stringify(r.prior), e.k, e.ref, e.ver, e.amt, e.by, e.at, e.src, r.flags, r.dupOf || null, !!(r.stale && !X.localOf(r0).reloaded)]); };
  X.approveSpec = function (r0) {
    var r = X.eff(r0), im = X.impact(r);
    return { kind: 'approve', id: r.id, snap: X.stageSig(r0), evKey: r0.ev ? r0.ev.ref + ' ' + r0.ev.ver : '', title: 'تأیید مرحلهٔ پرداخت', subject: r.inv, zone: 'cond', needsReason: false, reasonLabel: 'مبنای تصمیم / مرجع', target: tgt(r), current: X.REV.pending.label + ' · مدرک ' + (r.ev ? r.ev.ref + ' ' + r.ev.ver : '—'), amount: X.amt(r.claimed, r.unit), resulting: 'تأیید مالی این مرحله (وصول کامل فاکتور جدا سنجیده می‌شود)',
      changes: ['وضعیت بررسی مالی مرحله: در انتظار ← تأیید شد', 'وصول‌شدهٔ معتبر از ' + X.amtT(r.paid, r.unit) + ' به ' + X.amtT(im.np, r.unit) + ' (محاسبهٔ نمایشی)', 'مسئول بعدی: ' + (im.last ? 'قرارداد تکمیل فروش' : 'فروشنده (ادامهٔ فروش)')],
      unchanged: ['مدرک و نسخهٔ ثبت‌شده', 'تصمیم‌های قبلی (' + fa(r.prior.length) + ' مورد)', 'مالک اولیه و انتساب تاریخی', 'هیچ اعتبار کیف پول از این صفحه ثبت نمی‌شود'],
      affected: ['مرحله ' + r.id, 'فاکتور ' + r.inv, 'تاریخچهٔ ممیزی مرحله'],
      irrev: ['تصمیم در تاریخچه ثبت و حذف‌شدنی نیست', 'موتور هدف‌دار ممکن است هنگام تأیید اعتبار خودکار بسازد (OPD-09)؛ این رابط آن را اجرا یا نمایش موفقیت نمی‌دهد', 'تأیید ≠ تسویه'],
      ack: 'مبلغ، واحد، مدرک و اثر بالا را خواندم؛ می‌دانم تأیید ≠ وصول کامل ≠ استحقاق ≠ تسویه است.', commit: 'ثبت تأیید مالی (نمایشی)' };
  };
  X.rejectSpec = function (r0) {
    var r = X.eff(r0);
    return { kind: 'reject', id: r.id, snap: X.stageSig(r0), evKey: r0.ev ? r0.ev.ref + ' ' + r0.ev.ver : '', title: 'رد مرحلهٔ پرداخت با دلیل', subject: r.inv, zone: 'cond', needsReason: true, reasonLabel: 'دلیل رد (برای فروشنده و ممیزی)', target: tgt(r), current: X.REV.pending.label, amount: X.amt(r.claimed, r.unit), resulting: 'رد شد؛ بازگشت به فروشنده برای اصلاح',
      changes: ['وضعیت بررسی مالی مرحله: در انتظار ← رد شد', 'مسئول بعدی: فروشنده (اصلاح و ارسال دوباره)'],
      unchanged: ['مدرک و نسخه‌های قبلی', 'تأییدهای قبلی (بازنشانی نمی‌شود)', 'فاکتور حذف یا لغو نمی‌شود', 'هیچ استردادی ثبت نمی‌شود', 'تاریخچه پاک نمی‌شود'],
      affected: ['مرحله ' + r.id, 'فاکتور ' + r.inv, 'اعلان اصلاح به فروشنده'],
      irrev: ['رد در تاریخچه می‌ماند؛ با ارسال دوبارهٔ مدرک، زمینهٔ بررسی جدید ساخته می‌شود', 'اعتبار قبلاً تأییدشده خودکار نابود نمی‌شود'],
      ack: 'می‌دانم «رد» به‌معنی حذف، لغو، استرداد یا بازنشانی تاریخچه نیست و دلیل برای فروشنده قابل مشاهده است.', commit: 'ثبت رد با دلیل (نمایشی)' };
  };

  /* ---------- Commit: Review decisions (prototype-local overlay; per-item truth) ---------- */
  X.commitDecision = function (sp) {
    var r0 = X.q(sp.id), r = X.eff(r0);
    if (st.flow === 'stalecommit') {   // scripted demo trigger: simulates ANOTHER reviewer deciding after the confirmation opened; the conflict itself is derived below from live state
      st.local[r.id] = { review: 'approved', reloaded: true, approvedClaimed: r.claimed, prior: r.prior.concat([{ res: 'approved', who: 'امیر صادقی', at: '۱۰ دقیقه پیش', note: 'تصمیم همزمان بازبین دیگر' }]), rev: 'امیر صادقی' };
      st.flow = null;
    }
    // Commit-time recheck of the LIVE stage (state, eligibility, evidence version) — never trusts the state the dialog was opened on.
    var cur = X.localOf(r0), live = X.elig(r0), evNow = r0.ev ? r0.ev.ref + ' ' + r0.ev.ver : '';
    var why = cur.review !== 'pending' ? 'مرحله دیگر در انتظار نیست (' + X.REV[cur.review].label + ')' : (sp.kind === 'approve' ? live.blocked : !live.reject) ? 'مرحله اکنون واجد شرایط این تصمیم نیست' : (sp.evKey != null && sp.evKey !== evNow) ? 'نسخهٔ مدرک پس از باز شدن تأیید تغییر کرده است' : (sp.snap != null && sp.snap !== X.stageSig(r0)) ? 'مبلغ، واحد، مدرک یا رابطهٔ فاکتور پس از باز شدن تأیید تغییر کرده است' : null;
    if (why) return X.newOp({ title: (sp.kind === 'approve' ? 'تأیید' : 'رد') + ' مرحله ' + r.id + ' — تعارض هنگام ثبت', kind: 'decision', requested: 1, conflict: true, items: [[r.inv + ' · ' + r.id, 'skipped', 'در بررسی نزدیک ثبت: ' + why + '؛ هیچ تغییری ثبت نشد', null, r.id]], note: 'ثبت انجام نشد و چیزی تغییر نکرد. تصمیم باید روی وضعیت فعلی دوباره بررسی شود.' });
    var pr = { res: sp.kind === 'approve' ? 'approved' : 'rejected', who: M.user.name, at: NOW, note: sp.reasonText || (sp.kind === 'approve' ? 'تأیید مرحله' : '') };
    var l = { review: pr.res, prior: r.prior.concat([pr]), rev: M.user.name, reloaded: true };
    if (sp.kind === 'approve') { l.approvedClaimed = r.claimed; l.valid = r.claimed; l.next = X.impact(r).last ? 'قرارداد تکمیل فروش (جدا)' : 'فروشنده (ادامهٔ فروش)'; }
    else { l.next = 'فروشنده (اصلاح و ارسال دوباره)'; l.retSeller = true; }
    st.local[r.id] = l;
    pushAudit({ kind: sp.kind === 'approve' ? 'تأیید مرحلهٔ پرداخت' : 'رد مرحلهٔ پرداخت', inv: r.inv, cs: r.cs || '—', ev: r.ev ? r.ev.ref + ' ' + r.ev.ver : '—', amt: X.amtT(r.claimed, r.unit), ba: 'در انتظار ← ' + X.REV[pr.res].label + (sp.kind === 'reject' ? ' (تأییدهای قبلی حفظ)' : ''), reason: sp.reasonText || '—', rev: X.REV[pr.res].label, tx: 'ثبت نشده', key: '—', run: '—', res: 'موفق (نمایشی)', corr: 'C-' + (7700 + M.audit.length) });
    return X.newOp({ title: (sp.kind === 'approve' ? 'تأیید مالی' : 'رد با دلیل') + ' مرحله ' + r.id, kind: 'decision', requested: 1, items: [[r.inv + ' · ' + r.id, 'ok', sp.kind === 'approve' ? 'تصمیم ثبت شد (نمایشی)' : 'رد ثبت شد؛ بازگشت به فروشنده']],
      note: sp.kind === 'approve' ? 'ثبت دفتر کل / اعتبار کمیسیون / تسویه در این عملیات انجام نشد و وضعیت آن‌ها «نامعلوم یا موتور‌وابسته» می‌ماند. ' + 'تأیید مرحله فاکتور را کاملاً پرداخت‌شده اعلام نکرد.' : 'تاریخچه و تأییدهای قبلی دست‌نخورده ماند؛ استردادی ثبت نشد.' });
  };

  /* ---------- Bulk review (read-only triage) and bulk approve (conditional; per-item eligibility + per-item result) ---------- */
  var evk = function (r0) { return r0.ev ? r0.ev.ref + ' ' + r0.ev.ver : ''; };
  var selRows = function () { return X.keys(st.sel).map(X.q).filter(function (r) { return r && X.eff(r).review === 'pending'; }); };
  D.bulkrev = function () {
    var rows = selRows().map(function (r0) { var r = X.eff(r0), e = X.elig(r0); return '<tr><td><b class="mono">' + r.inv + '</b><span class="cell-sub mono">' + r.id + '</span></td><td>' + X.amt(r.claimed, r.unit) + '</td><td class="wrap">' + (e.blocked ? '<span class="neg">' + ic('xCircle') + ' ' + esc(e.checks.filter(function (c) { return c[0] === 'no'; })[0][1]) + '</span>' : '<span class="chk-ok">' + ic('checkCircle') + ' بدون مانع شناخته‌شده</span>') + '</td><td class="col-actions">' + open('rs', r.id, 'باز کردن') + '</td></tr>'; }).join('');
    return top('بررسی گروهی (فقط‌خواندنی)') + head('بررسی گروهی ' + fa(selRows().length) + ' مرحله', pill('فقط‌خواندنی', 'teal', 'eye'), '') + '<div class="dr-body"><section class="sec primary">' + ctxLine('این نما برای تریاژ است و هیچ تصمیمی ثبت نمی‌کند. هر ردیف جداگانه باز و بررسی می‌شود؛ تصمیم مالی بدون بررسی هر ردیف نیست.') + '<div class="tbl-wrap"><table class="tbl no-cursor" aria-label="بررسی گروهی"><caption class="sr">هر مرحله با مبلغ و واحد و مانع شناخته‌شده</caption><thead><tr><th>مرحله</th><th>مبلغ</th><th>مانع / آمادگی</th><th class="col-actions"><span class="sr">اقدام</span></th></tr></thead><tbody>' + rows + '</tbody></table></div></section></div>' + foot('', null, 'هیچ تغییری ثبت نشد');
  };
  X.bulkSig = function () { return selRows().map(function (r0) { return r0.id + ':' + X.stageSig(r0); }).join('|'); };
  D.bulkapp = function () {
    var keep = keepOf(); if (keep.sig == null) keep.sig = X.bulkSig();
    var rows = selRows(), el = rows.map(function (r) { return [r, X.elig(r)]; }), ok = el.filter(function (x) { return !x[1].blocked; }), no = el.filter(function (x) { return x[1].blocked; });
    var by = {}; ok.forEach(function (x) { var u = X.unitL(x[0].unit); by[u] = (by[u] || 0) + x[0].claimed; });
    var imp = Object.keys(by).map(function (u) { return fa(num(by[u])) + ' ' + u; }).join(' · ') || '—';
    var over = ok.length > 100;
    var tr = el.map(function (x) { var r = X.eff(x[0]); return '<tr class="' + (x[1].blocked ? 'row-off' : '') + '"><td><b class="mono">' + r.inv + '</b><span class="cell-sub mono">' + r.id + '</span></td><td>' + X.amt(r.claimed, r.unit) + '</td><td class="wrap">' + (x[1].blocked ? '<span class="neg">' + ic('xCircle') + ' ' + esc(x[1].checks.filter(function (c) { return c[0] === 'no'; })[0][1]) + '</span><span class="cell-sub">خارج از ارسال؛ جدا بررسی شود</span>' : '<span class="chk-ok">' + ic('checkCircle') + ' واجد شرایط</span>') + '</td></tr>'; }).join('');
    var ready = ok.length && !over && keep.ack;
    return top('تأیید گروهی (مشروط)') + head('تأیید گروهی ' + fa(el.length) + ' مرحله', X.zone('cond', 'نتیجهٔ هر ردیف جدا'), '') + '<div class="dr-body"><section class="sec primary">' +
      X.dl([['انتخاب‌شده', fa(el.length) + ' مرحله'], ['واجد شرایط ارسال', fa(ok.length)], ['خارج از ارسال', fa(no.length)], ['مجموع ادعاشده (به تفکیک واحد)', imp]]) + ctxLine('«۱۰۰ موفق» از ارسال درخواست نتیجه‌گیری نمی‌شود: پس از ثبت، نتیجهٔ هر مرحله (ثبت‌شده، ناموفق، نامعلوم) جدا گزارش می‌شود. سقف مسیر فعلی ۱۰۰ شناسه است.') + (over ? '<div class="note warn inset">' + ic('alert') + '<span>تعداد از سقف ۱۰۰ بیشتر است؛ ارسال مسدود است.</span></div>' : '') +
      '<div class="tbl-wrap"><table class="tbl no-cursor" aria-label="آمادگی هر ردیف"><caption class="sr">آمادگی هر مرحله برای تأیید گروهی</caption><thead><tr><th>مرحله</th><th>مبلغ</th><th>آمادگی</th></tr></thead><tbody>' + tr + '</tbody></table></div>' +
      '<div class="sens-form"><label class="confirm"><input type="checkbox" class="cbx" data-bulkack' + (keep.ack ? ' checked' : '') + '><span>اثر هر ردیف را مرور کردم؛ می‌دانم نتیجه ممکن است جزئی یا نامعلوم باشد و تأیید ≠ ثبت دفتر کل ≠ تسویه.</span></label></div></section></div>' + foot(btn('btn-primary btn-lg', 'commit-bulkapp', 'ثبت تأیید ' + fa(ok.length) + ' مرحلهٔ واجد شرایط (نمایشی)', 'send', !ready), null, 'ردیف‌های مسدود ارسال نمی‌شوند');
  };
  X.runBulkApp = function () {
    var rows = selRows(), items = [];
    rows.forEach(function (r0) {
      var r = X.eff(r0), e = X.elig(r0), lbl = r.inv + ' · ' + r.id;
      if (e.blocked) return items.push([lbl, 'skipped', e.checks.filter(function (c) { return c[0] === 'no'; })[0][1], null, r.id, evk(r0)]);
      if (r0.bulkOutcome === 'failed') return items.push([lbl, 'failed', 'خطای گذرا؛ تصمیم ثبت نشد (اثر صفر مشخص). تکرار فقط پس از بازخوانی تازه', null, r.id, evk(r0)]);
      if (r0.bulkOutcome === 'unknown') return items.push([lbl, 'unknown', 'پاسخ نرسید؛ ممکن است ثبت شده باشد. تکرار کور ممنوع', null, r.id, evk(r0)]);
      X.applyApprove(r0, 'تأیید گروهی'); items.push([lbl, 'ok', 'تأیید ثبت شد (نمایشی)', null, r.id, evk(r0)]);
    });
    st.sel = {};
    return X.newOp({ title: 'تأیید گروهی مراحل پرداخت', kind: 'bulkapp', requested: rows.length, items: items, retry: true, note: 'نتیجهٔ هر مرحله جدا از ارسال درخواست است. ثبت دفتر کل / اعتبار / تسویه انجام نشد.' });
  };
  X.applyApprove = function (r0, note) {
    if (X.localOf(r0).review === 'approved') return false;   // idempotent: a stage already approved is never counted twice
    var r = X.eff(r0);
    st.local[r.id] = { review: 'approved', prior: r.prior.concat([{ res: 'approved', who: M.user.name, at: NOW, note: note }]), rev: M.user.name, reloaded: true, approvedClaimed: r.claimed, valid: r.claimed, next: X.impact(r).last ? 'قرارداد تکمیل فروش (جدا)' : 'فروشنده (ادامهٔ فروش)' };
    return true;
  };
  D.res = function (id) {
    var op = X.opOf(id) || st.ops[0]; if (!op) return top('نتیجه عملیات') + '<div class="dr-body">' + h.stateBlock('empty', 'نتیجه‌ای نیست', '') + '</div>';
    var c = X.counts(op), s = X.OPS_S(X.opState(op)), groups = [['ok', 'ثبت‌شده'], ['existing', 'تراکنش/تصمیم موجود'], ['skipped', 'ارسال‌نشده'], ['failed', 'ناموفق'], ['unknown', 'نامعلوم']];
    var gl = op.items ? groups.filter(function (g) { return c[g[0]]; }).map(function (g) {
      var list = op.items.filter(function (x) { return x[1] === g[0]; }), o = X.OUT[g[0]];
      return '<details class="sec"' + (g[0] === 'ok' ? '' : ' open') + '><summary><h3>' + pill(o.label, o.tone, o.icon) + '</h3><span class="muted" style="margin-right:8px;font-size:var(--t-meta)">' + fa(list.length) + '</span><span class="chev">' + ic('chev') + '</span></summary><ul class="ir-list">' + list.map(function (x) { return '<li class="ir"><span>' + esc(x[0]) + '</span><span class="ir-why">' + esc(x[2] || '—') + '</span></li>'; }).join('') + '</ul></details>';
    }).join('') : '';
    var acts = '';
    if (op.kind === 'bulkapp' && c.unknown) acts += btn('btn-primary btn-lg', 'reconcile-op:' + op.id, 'خواندن وضعیت واقعی موارد نامعلوم (فقط‌خواندنی)', 'refresh');
    if (op.kind === 'bulkapp' && c.failed && !c.unknown && op.retry) acts += btn('btn-primary btn-lg', 'retry-op:' + op.id, 'تکرار فقط ناموفق‌های معلوم', 'refresh');
    if (op.run) acts += btn('btn-soft btn-lg', 'open-run:' + op.run, 'باز کردن کنسول اجرا', 'layers');
    return top('نتیجه عملیات') + head(esc(op.title), pill(s[0], s[1], s[2]), '<span class="mono">' + op.id + '</span><span>' + esc(op.ts) + '</span>') + '<div class="dr-body"><section class="sec primary"><p class="result-line">' + ic(c.ok + c.existing === c.requested ? 'checkCircle' : 'alert') + (c.ok + c.existing === c.requested ? 'همهٔ ' + fa(num(c.requested)) + ' مورد با نتیجهٔ مشخص ثبت شد.' : fa(num(c.ok + c.existing)) + ' از ' + fa(num(c.requested)) + ' مورد ثبت شد؛ بقیه به‌تفکیک زیر آمده‌اند.') + '</p>' + X.outcomeStrip(op) +
      (c.unknown ? '<div class="note warn inset">' + ic('question') + '<span><b>نتیجه نامعلوم است.</b> ممکن است ثبت شده باشد؛ ابتدا وضعیت واقعی را بخوانید و تکرار کور انجام ندهید.</span></div>' : '') + (op.conflict ? '<div class="note warn inset">' + ic('swap') + '<span><b>تعارض:</b> وضعیت واقعی عوض شده بود؛ هیچ تغییری ثبت نشد.</span></div>' : '') + (op.note ? ctxLine(esc(op.note)) : '') + '</section>' + gl + '</div>' + foot(acts, null, 'شماره ارجاع ' + op.id + ' · نمایشی');
  };
  // Each retry/reconcile item is re-checked against the LIVE stage; an item that is no longer eligible keeps a conflict/skipped result and is never re-applied.
  var live = function (x) { var r = X.q(x[4]); return r ? { r: r, cur: X.localOf(r), e: X.elig(r), evChanged: x[5] != null && x[5] !== evk(r) } : null; };
  X.retryFailed = function (op) {
    var res = { ok: 0, conflict: 0 };
    op.items.forEach(function (x) {
      if (x[1] !== 'failed') return; var L = live(x);
      if (L && L.evChanged) { res.conflict++; x[1] = 'skipped'; x[2] = 'تعارض: نسخهٔ مدرک پس از عملیات اصلی تغییر کرده است؛ نیازمند تأیید جدید با بررسی اثر، تغییری ثبت نشد'; return; }
      if (!L || L.cur.review !== 'pending' || L.e.blocked) { res.conflict++; x[1] = 'skipped'; x[2] = 'تعارض در بازخوانی تازه: مرحله دیگر قابل تصمیم نیست (' + (L ? X.REV[L.cur.review].label : 'نامعلوم') + ')؛ تغییری ثبت نشد'; return; }
      X.applyApprove(L.r, 'تکرار پس از بازخوانی تازه'); x[1] = 'ok'; x[2] = 'بازخوانی تازه انجام شد و مرحله هنوز واجد شرایط بود؛ ثبت شد (نمایشی)'; res.ok++;
    });
    return res;
  };
  X.reconcileOp = function (op) {
    var n = 0, unres = 0;
    op.items.forEach(function (x) {
      if (x[1] !== 'unknown') return; var L = live(x);
      if (L && L.evChanged) { unres++; x[2] = 'نسخهٔ مدرک پس از عملیات اصلی تغییر کرده است؛ نتیجهٔ اصلی با مدرک ذخیره‌شده باید تطبیق شود و نامعلوم می‌ماند (مدرک جدید تأیید نمی‌شود)'; return; }
      if (L && L.cur.review === 'approved') { x[1] = 'ok'; x[2] = 'تطبیق: تصمیم قبلاً ثبت شده بود (دوباره اعمال نشد)'; return; }
      if (n++ % 2 === 0 && L && L.cur.review === 'pending' && !L.e.blocked) { X.applyApprove(L.r, 'تطبیق: ثبت شده بود'); x[1] = 'ok'; x[2] = 'تطبیق: تصمیم قبلاً ثبت شده بود'; }
      else { x[1] = 'failed'; x[2] = 'تطبیق: ثبت نشده بود؛ اکنون قابل تکرار پس از بازخوانی تازه'; }
    });
    op.note = (op.note || '') + (unres ? ' تطبیق کامل نشد: نتیجهٔ ' + fa(unres) + ' مورد به‌دلیل تغییر مدرک همچنان نامعلوم است.' : ' نتیجهٔ نامعلوم با خواندن وضعیت واقعی تطبیق شد.');
    return unres;
  };

  /* ---------- Refund review (CONDITIONAL — OPD-04): concepts are separate; nothing is executed here ---------- */
  D.ref = function (id) {
    var r = X.rf(id), order = ['requested', 'reported', 'proof_pending', 'confirmed', 'partial', 'full', 'unknown'];
    var ladder = '<ol class="rf-lad" aria-label="مفاهیم جدای استرداد">' + order.map(function (k) { var d = X.RFS[k], on = k === r.st; return '<li class="' + (on ? 'on' : '') + '"' + (on ? ' aria-current="step"' : '') + '>' + pill(d.label, on ? d.tone : 'slate', d.icon) + '</li>'; }).join('') + '</ol>';
    var rem = r.done == null ? null : r.paidOrig - r.done;
    var facts = X.dl([['فاکتور اصلی', '<span class="mono">' + r.inv + '</span>'], ['پرداخت اصلی', X.amt(r.paidOrig, r.unit)], ['مبلغ درخواستی', X.amt(r.req, r.unit)], ['قبلاً استردادشده (' + (r.st === 'partial' || r.st === 'confirmed' || r.st === 'full' ? 'ثبت‌شده' : 'گزارش‌شده/نامعلوم') + ')', X.amt(r.done, r.unit, 'نتیجه اجرا نامعلوم است؛ صفر فرض نمی‌شود')], ['باقی‌ماندهٔ مجاز (زمینه)', rem == null ? '<span class="neg">نامعلوم</span>' : X.amt(rem, r.unit)], ['منبع اجرا', esc(r.exec)], ['مدرک', r.proof ? esc(r.proof) : '<span class="neg">موجود نیست</span>'], ['بازبین', esc(r.rev)]]);
    var bad = r.invCancelled ? h.banner('incomplete', '<b>فاکتور لغو شده است.</b> این به‌معنی بازگشت وجه به مشتری نیست؛ تا مدرک اجرا، استرداد «درخواست‌شده» است.', '') : '';
    var gw = X.can('refund');
    return top('بررسی استرداد (مشروط)') + head('استرداد ' + esc(r.id), X.rfPill(r.st) + X.zone('cond', 'OPD-04'), '<span class="mono">' + r.inv + '</span>') + '<div class="dr-body">' + bad + sec('مفاهیم جدا', '', ladder + ctxLine('درخواست، اجرای گزارش‌شده و اجرای تأییدشده سه چیز جدا هستند؛ جزئی/کامل نتیجهٔ مبلغ است نه مدرک.'), 'primary') + sec('پرداخت اصلی و مبالغ', X.basis('snap'), facts) +
      sec('تاریخچه', '', X.tl(r.hist.map(function (x) { return [x, '', '']; }))) + sec('محدودیت‌ها', '', checks([['q', 'سیاست مدرک بانکی و تأییدکننده تعریف نشده است (OPD-04)', r.note], ['no', 'اجرای بانکی از این رابط وجود ندارد', 'ثبت درخواست یا مدرک جای اجرای پول نیست'], ['warn', 'استرداد گروهی مجاز نیست', 'هر پرداخت مدرک و اثر جدا دارد']])) + '</div>' + foot(X.guardBtn('refund', 'btn-primary btn-lg', 'x', 'تأیید اجرای استرداد', 'repeat', true), null, esc(gw.why).slice(0, 180));
  };

  /* ---------- Ledger trace (read-only; corrections are linked events; no edit/delete) ---------- */
  D.tx = function (id) {
    var t = M.ledger.filter(function (x) { return (x.id === id) || (x.id === '—' && x.key === id); })[0];
    var root = t.of ? X.tx(t.of) : t, chain = [root].concat((root.revs || []).map(X.tx));
    var cl = X.tl(chain.map(function (x) { return [(x.id === t.id ? '◄ ' : '') + x.id + ' — ' + (x.dir === 'C' ? 'بستانکار' : 'بدهکار') + ' ' + fa(num(x.amt)) + ' ' + X.unitL(x.unit), x.posted, x.why || (x === root ? 'رویداد اصلی' : '')]; }));
    return top('ردیابی تراکنش') + head(esc(t.id === '—' ? 'نیت بدون تراکنش' : t.id), X.lgPill(t.st) + pill(t.dom === 'wallet' ? 'کیف پول کمیسیون' : 'وصول فاکتور', 'slate', 'wallet'), '<span class="mono key">' + esc(t.key) + '</span>') + '<div class="dr-body">' +
      (t.note ? '<section class="sec">' + h.banner(t.st === 'committed' ? 'incomplete' : 'locked', esc(t.note), '') + '</section>' : '') +
      sec('هویت و مبلغ', X.basis('range'), X.dl([['شناسه تراکنش', '<span class="mono">' + esc(t.id) + '</span>'], ['کلید کسب‌وکار', '<span class="mono key">' + esc(t.key) + '</span>'], ['حساب · دامنه', esc(t.acct)], ['جهت', t.dir === 'C' ? 'بستانکار' : 'بدهکار'], ['مبلغ · واحد', X.amt(t.amt, t.unit)], ['رابطه', esc(t.rel[0]) + ' · ' + esc(t.rel[1])]]), 'primary') +
      sec('عامل و زمان', '', X.dl([['عامل', esc(t.actor)], ['زمان اثر مالی', esc(t.eff)], ['زمان ثبت', esc(t.posted)], ['وضعیت ثبت', X.lgPill(t.st)]])) +
      sec('رویدادهای مرتبط (اصلاح/معکوس)', '', t.of || t.revs ? cl : '<p class="ind-note">' + ic('info') + ' رویداد جبرانی یا اصلاحی مرتبط ثبت نشده است.</p>') +
      sec('حفاظت', '', checks([['ok', 'فقط‌افزودنی: ویرایش یا حذف تراکنش وجود ندارد', 'تصحیح با رویداد جدید و ارتباط با رویداد اصلی'], ['ok', 'نمایش خالص: رندر هیچ ثبت خودکار (F07) را اجرا نمی‌کند', ''], ['q', 'ثبت دفتر کل جدید نیازمند اختیار جدا است', 'تعدیل دستی فعلی فقط مدیر سیستم']])) + '</div>' + foot('', null, 'فقط‌خواندنی · نمایشی');
  };

  /* ---------- Run console: lifecycle · snapshot · coverage · approval chain · items · Outcome Unknown context ---------- */
  D.run = function (id) {
    var r = X.runView(X.run(id)), c = r.cov, sti = X.RUNORD.indexOf(r.st);
    var life = '<ol class="lc-rib sm" aria-label="چرخهٔ عمر اجرا">' + X.RUNORD.map(function (k) { var d = X.RUN[k], on = k === r.st; return '<li class="' + (on ? 'on' : '') + '"' + (on ? ' aria-current="step"' : '') + '>' + pill(d.label, on ? d.tone : 'slate', d.icon) + '</li>'; }).join('') + '</ol>';
    var snap = X.dl([['موتور', esc(r.engL) + (r.eng === 'purpose' ? ' — ممکن است هنگام تأیید مرحله خودکار اعتبار بسازد (OPD-09)' : ' — مسیر run/APPLY')], ['نسخهٔ قانون', 'نسخه ثبت‌شده با اجرا'], ['قفل snapshot', r.appr ? 'ثبت شد: ' + esc(r.appr.locked) + ' — تغییرناپذیری در همهٔ مسیرها اثبات نشده (FIN-LOCK)' : 'هنوز تأیید نشده'], ['پرچم فعال‌سازی ثبت', 'مقدار نصب‌شده تأیید نشده؛ تغییر آن در حوزهٔ مدیر مجاز است'], ['کلید کسب‌وکار', 'فاکتور/مرحله/هدف/گیرنده — نه فقط شناسهٔ اجرا؛ اجرای جدید حق مالی جدید نمی‌سازد']]);
    var chain = X.dl([['سازندهٔ اجرا / پیش‌نمایش', esc(r.by) + ' · ' + esc(r.at)], ['تأییدکننده', r.appr ? esc(r.appr.by) + ' · ' + esc(r.appr.at) : 'تأیید نشده'], ['ثبت‌کننده', r.st === 'posted' || r.st === 'partial' || r.st === 'reconciled' ? esc(M.user.name) + ' (نمایشی)' : 'هنوز ثبت نشده'], ['تفکیک وظیفه (maker/checker)', 'تعریف نشده (OPD-08) — هم‌عامل بودن مجاز یا ممنوع فرض نشده است']]);
    var items = r.items.length ? '<div class="tbl-wrap"><table class="tbl no-cursor" aria-label="آیتم‌های اجرا"><caption class="sr">نمونهٔ آیتم‌های اجرا با کلید کسب‌وکار، مبلغ، نتیجه و تراکنش</caption><thead><tr><th>آیتم · کلید</th><th>مبلغ</th><th>نتیجه</th><th class="col-opt">تراکنش</th></tr></thead><tbody>' + r.items.map(function (x) { var d = X.ITEMR[x[4]]; return '<tr><td class="wrap">' + esc(x[0]) + '<span class="cell-sub mono key">' + esc(x[1]) + '</span></td><td>' + X.amt(x[2], x[3]) + '</td><td class="wrap">' + pill(d[0], d[1], d[2]) + (x[6] ? '<span class="cell-sub">' + esc(x[6]) + '</span>' : '') + '</td><td class="col-opt mono">' + (x[5] || '—') + '</td></tr>'; }).join('') + '</tbody></table></div><p class="ind-note">' + ic('filter') + ' نمایش ' + fa(r.items.length) + ' آیتم نمونه از ' + (r.intended ? fa(num(r.intended)) : '—') + ' — زیرمجموعهٔ محدود، نمای کامل اجرا نیست.</p>' : '<p class="ind-note">' + ic('info') + ' هنوز آیتمی تولید نشده است.</p>';
    var oux = r.st === 'unknown' ? X.oux({ intent: 'ثبت ' + fa(num(r.intended)) + ' اعتبار کمیسیون روی snapshot تأییدشده ' + r.id, key: esc(r.id + '|فاکتور|مرحله|هدف|گیرنده'), possible: 'ممکن است بخشی یا همه ثبت شده باشد؛ «خطا» به‌معنی اثر صفر نیست', lookup: 'جستجوی تراکنش و کلید کسب‌وکار هر آیتم (فقط‌خواندنی) پیش از هر اقدام', existing: 'هنوز خوانده نشده — پس از جستجو آیتم‌های موجود «تراکنش موجود» شمرده می‌شوند', retry: 'فقط برای آیتم‌هایی که ثبت‌نشدن آن‌ها اثبات شده؛ تکرار کلی ممنوع', next: 'خواندن وضعیت واقعی (فقط‌خواندنی) ← طبقه‌بندی ثبت‌شده/ثبت‌نشده ← تصمیم جدا برای ثبت‌نشده‌ها' }) : '';
    var reconciledNote = r.st === 'reconciled' && r.tail ? '<div class="note inset">' + ic('checkCircle') + '<span><b>تطبیق انجام شد:</b> ' + fa(num(c.posted)) + ' آیتم با تراکنش یافت شد و ' + fa(num(c.unprocessed)) + ' آیتم ثبت‌نشدن آن‌ها اثبات شد. تکرار فقط برای همین دنباله مجاز است.</span></div>' : '';
    var acts = '', hintT = 'پیش‌نمایش ثبت نهایی نیست';
    if (r.st === 'draft') acts = X.guardBtn('runapprove', 'btn-primary btn-lg', 'run-gen:' + r.id, 'تولید پیش‌نمایش (نوشتن فراداده)', 'eye');
    else if (r.st === 'preview') acts = X.guardBtn('runapprove', 'btn-primary btn-lg', 'run-approve:' + r.id, 'ادامه: تأیید اجرا', 'check');
    else if (r.st === 'approved') acts = X.guardBtn('post', 'btn-primary btn-lg', 'run-post:' + r.id, 'ادامه: اجرای ثبت', 'send');
    else if ((r.st === 'partial' || (r.st === 'reconciled' && r.tail)) && c && c.unprocessed) acts = X.guardBtn('post', 'btn-primary btn-lg', 'run-tail:' + r.id, 'ادامه: فقط دنبالهٔ پردازش‌نشدهٔ اثبات‌شده (' + fa(num(c.unprocessed)) + ')', 'send');
    else if (r.st === 'unknown') { acts = (st.lookups[r.id] ? X.guardBtn('reconcile', 'btn-primary btn-lg', 'run-recon:' + r.id, 'ادامه: ثبت نتیجهٔ تطبیق', 'checkCircle') : '') + btn(st.lookups[r.id] ? 'btn-soft btn-lg' : 'btn-primary btn-lg', 'run-lookup:' + r.id, 'خواندن وضعیت واقعی (جستجوی تراکنش/کلید · فقط‌خواندنی)', 'search') + '<button type="button" class="btn btn-lg tip" aria-disabled="true" data-tip="تکرار ثبت تا تطبیق نتیجهٔ قبلی مسدود است؛ خطا یا قطع پاسخ مجوز ثبت مجدد نیست.">' + ic('ban') + 'تکرار ثبت (مسدود تا تطبیق)</button>'; hintT = 'تکرار کور وجود ندارد'; }
    else hintT = 'برای این وضعیت اقدام روزانه‌ای تعریف نشده است';
    return top('کنسول اجرا') + head(esc(r.name), X.runPill(r.st) + pill(r.engL, 'slate', 'layers'), '<span class="mono">' + r.id + '</span><span>' + esc(r.at) + '</span>') + '<div class="dr-body">' + sec('چرخهٔ عمر', '', life + (r.st === 'preview' || r.st === 'draft' ? '' : '') + ctxLine(esc(r.note)), 'primary') +
      (c ? sec('پوشش', X.cov(c.processed == null ? 'partial' : c.unprocessed ? 'bounded' : 'ok'), X.covRow(c, 'پوشش اجرا') + X.covNote(c)) : '') + (oux ? sec('نتیجهٔ نامعلوم و تکرارپذیری', '', oux) : '') + (r.st === 'unknown' && st.lookups[r.id] ? sec('نتیجهٔ جستجو (فقط‌خواندنی؛ هنوز ثبت نشده)', pill('نمایش، نه ثبت', 'teal', 'eye'), X.dl([['ثبت‌شده یافت شد', fa(num(st.lookups[r.id].committed)) + ' آیتم'], ['ثبت‌نشدن اثبات شد', fa(num(st.lookups[r.id].notCommitted)) + ' آیتم']]) + (st.lookups[r.id].byKey ? '<ul class="ir-list">' + Object.keys(st.lookups[r.id].byKey).map(function (k) { return '<li class="ir"><span class="mono key">' + esc(k) + '</span><span class="ir-why">' + (st.lookups[r.id].byKey[k] === 'committed' ? 'ثبت‌شده یافت شد' : 'ثبت‌نشدن اثبات شد') + '</span></li>'; }).join('') + '</ul>' : '') + ctxLine('این نتیجه فقط خوانده شده است؛ اجرا هنوز «نتیجه نامعلوم» است. ثبت تطبیق گام جدا با اختیار مجاز است.')) : '') + reconciledNote +
      sec('Snapshot و موتور', '', snap) + sec('زنجیرهٔ ایجاد · تأیید · ثبت', '', chain) + sec('آیتم‌ها', '', items) + '</div>' + foot(acts, null, hintT);
  };
  var runSpec = function (kind, r, o) {
    var c = r.cov || {};
    return Object.assign({ kind: kind, id: r.id, srcSig: X.runSig(r), srcSt: r.st, subject: r.id + ' · ' + r.name, zone: 'cond', amount: '<span class="muted">مجموع ' + (c.intended ? fa(num(c.intended)) + ' آیتم' : 'آیتم‌ها') + ' — مبلغ هر آیتم در کنسول؛ واحدها جمع نمی‌شوند</span>' }, o);
  };
  X.runGenSpec = function (r) { return runSpec('rungen', r, { title: 'تولید پیش‌نمایش اجرا', needsReason: false, reasonLabel: 'یادداشت (اختیاری)', target: r.id + ' · ' + r.name, current: 'پیش‌نویس — بدون ردیف مالی', resulting: 'پیش‌نمایش با ردیف‌های محاسبه‌شده', changes: ['ایجاد اجرا و آیتم‌های پیش‌نمایش (فراداده)'], unchanged: ['هیچ تراکنش یا اعتبار کیف پول', 'قوانین و نرخ‌ها'], affected: ['اجرا ' + r.id, 'آیتم‌های پیش‌نمایش'], irrev: ['تولید پیش‌نمایش یک نوشتن است نه خواندن خالص؛ فراداده باقی می‌ماند'], ack: 'می‌دانم تولید پیش‌نمایش فراداده می‌نویسد و تأیید یا ثبت نیست.', commit: 'تولید پیش‌نمایش (نمایشی)' }); };
  X.runApproveSpec = function (r) { return runSpec('runapprove', r, { title: 'تأیید اجرا', needsReason: true, reasonLabel: 'مبنای تأیید اجرا', target: r.id + ' · ' + r.name + ' — ' + fa(r.intended) + ' آیتم', current: 'پیش‌نمایش', resulting: 'تأییدشده (ثبت‌نشده) و snapshot قفل‌شده', changes: ['وضعیت اجرا: پیش‌نمایش ← تأییدشده', 'ثبت تأییدکننده و زمان قفل'], unchanged: ['هیچ تراکنش یا اعتبار کیف پول ایجاد نمی‌شود', 'ثبت (APPLY) جدا و بعدی است'], affected: ['اجرا ' + r.id, 'آیتم‌های snapshot'], irrev: ['قفل ثبت می‌شود؛ تغییرناپذیری در همهٔ مسیرها اثبات نشده (FIN-LOCK)', 'تفکیک تأییدکننده/ثبت‌کننده تعریف نشده (OPD-08)'], ack: 'می‌دانم تأیید اجرا ≠ ثبت در کیف پول است.', commit: 'ثبت تأیید اجرا (نمایشی)' }); };
  X.runPostSpec = function (r, tail) {
    var c = r.cov, n = tail ? c.unprocessed : r.intended;
    return runSpec(tail ? 'runtail' : 'runpost', r, { title: tail ? 'ادامهٔ ثبت دنبالهٔ پردازش‌نشده' : 'اجرای ثبت', needsReason: true, reasonLabel: 'دلیل اجرا', target: r.id + ' · ' + fa(num(n)) + ' آیتم از snapshot تأییدشده', current: X.RUN[r.st].label + (tail ? ' · دنبالهٔ اثبات‌شدهٔ ثبت‌نشده' : ''),
      resulting: 'نتیجهٔ هر آیتم جدا: ثبت‌شده، تراکنش موجود، ناموفق یا نامعلوم', changes: ['ثبت اعتبار کیف پول برای آیتم‌های واجد شرایط (کلید کسب‌وکار یکتا)'], unchanged: ['snapshot تأییدشده', 'آیتم‌های قبلاً ثبت‌شده (تراکنش موجود دوباره نوشته نمی‌شود)', 'تاریخچهٔ تراکنش‌ها'], affected: [fa(num(n)) + ' آیتم', 'کیف پول گیرنده‌ها', 'دفتر کل'],
      irrev: ['ثبت اعتبار کیف پول با رویداد جبرانی قابل اصلاح است، نه حذف', 'اگر پاسخ نرسد، نتیجه «نامعلوم» می‌ماند و تکرار کور مسدود است', 'اعتبار کیف پول ≠ تسویه'], ack: 'کلید کسب‌وکار، نتیجهٔ هر آیتم و اینکه ok یک فراخوانی محدود کل اجرا نیست را فهمیدم.', commit: tail ? 'ثبت دنباله (نمایشی)' : 'اجرای ثبت (نمایشی)' });
  };
  X.runSig = function (r) { return [r.st, r.intended, JSON.stringify(r.cov || null), JSON.stringify(r.appr || null), (r.keys || []).join(',')].join('|') + '|' + (r.items || []).map(function (x) { return x.map(function (v) { return String(v); }).join('~'); }).join('||'); };   // state + coverage + approval + snapshot key set + FULL item identity (label, business key, amount, unit, result, tx, note)
  var setRun = function (id, patch) { st.runLocal[id] = Object.assign(st.runLocal[id] || {}, patch); };
  var txSeq = 9100;
  var mint = function (r, it) {
    var have = M.ledger.filter(function (t) { return t.key === it[1] && t.st === 'committed'; })[0];
    if (have) return { id: have.id, existing: true };   // unique business key: an already-committed key is never minted twice (reported as EXISTING, not new)
    var id = 'T-' + (++txSeq), row = M.ledger.filter(function (t) { return t.key === it[1] && t.st !== 'committed'; })[0];
    if (row) { row.id = id; row.st = 'committed'; row.posted = NOW; row.eff = NOW; row.actor = M.user.name + ' (ثبت اجرا، نمایشی)'; delete row.note; }
    else M.ledger.push({ id: id, key: it[1], dom: 'wallet', acct: 'کیف پول کمیسیون · گیرنده (نمایشی)', dir: 'C', amt: it[2], unit: it[3], rel: ['اجرا', r.id + ' · ' + it[0].split(' · ')[0]], actor: M.user.name + ' (ثبت اجرا، نمایشی)', eff: NOW, posted: NOW, st: 'committed' });
    return { id: id, existing: false };
  };
  // Keep per-item truth and the mock ledger consistent with the aggregate outcome shown on the run.
  // Reconcile LINKS the transaction the lookup found (existing); it never mints. The found tx is mirrored/converted in the mock ledger without creating a new credit.
  // PLAN (pure — mutates nothing): preflight the COMPLETE lookup against the ledger in EVERY state, for BOTH committed and not-committed claims.
  X.planReconcile = function (r, lk) {
    var by = lk.byKey || {}, tb = lk.txByKey || {}, conflicts = [];
    // The validation set is EVERY key the lookup mentions (not only the displayed sample rows).
    var keys = Object.keys(by), committed = keys.filter(function (k) { return by[k] === 'committed'; }), notC = keys.filter(function (k) { return by[k] === 'notCommitted'; });
    var seen = {}; committed.forEach(function (k) { if (tb[k]) (seen[tb[k]] = seen[tb[k]] || []).push(k); });
    // Same-key ledger multiplicity (ANY state) for EVERY key the lookup mentions, committed or not.
    keys.forEach(function (k) { var n = M.ledger.filter(function (t) { return t.key === k; }).length; if (n > 1) conflicts.push('چند ردیف دفتر کل (هر وضعیتی) برای کلید ' + k + ' وجود دارد (' + fa(n) + ')'); });
    committed.forEach(function (k) {
      var tx = tb[k];
      if (!tx) { conflicts.push('کلید ' + k + ' ثبت‌شده گزارش شد ولی شناسهٔ تراکنش بومی ندارد (نتیجهٔ جستجو ناقص است)'); return; }   // committed claim without a native tx identity = incomplete whole-lookup plan
      var rows = M.ledger.filter(function (t) { return t.key === k; });                          // same key, ANY ledger state
      var native = rows.filter(function (t) { return t.id !== '—'; });                          // rows that already carry a native tx id
      var comKey = rows.filter(function (t) { return t.st === 'committed'; });
      var anyTx = M.ledger.filter(function (t) { return t.id === tx && t.key !== k; });          // same tx id under ANOTHER key, ANY state
      if (rows.length > 1) return;   // already reported above
      if (seen[tx].length > 1) conflicts.push('جستجو شناسهٔ ' + tx + ' را به چند کلید نسبت داده');
      else if (anyTx.length) conflicts.push('شناسهٔ ' + tx + ' در دفتر کل با کلید دیگری آمده است');
      else if (native.some(function (t) { return t.id !== tx; })) conflicts.push('کلید ' + k + ' در دفتر کل (هر وضعیتی) با شناسهٔ ' + native.filter(function (t) { return t.id !== tx; })[0].id + ' آمده ولی جستجو ' + tx + ' را برگرداند');
      else if (comKey.length > 1) conflicts.push('چند تراکنش ثبت‌شده برای کلید ' + k);
    });
    // txByKey must carry EXACTLY the committed keys: an id on a not-committed or unrelated key contradicts the lookup itself.
    Object.keys(tb).forEach(function (k) { if (by[k] !== 'committed') conflicts.push('جستجو برای کلید ' + k + ' شناسهٔ تراکنش داده ولی آن را ثبت‌شده گزارش نکرده است' + (k in by ? '' : ' (کلید ناشناس)')); });
    notC.forEach(function (k) { if (M.ledger.some(function (t) { return t.key === k && t.st === 'committed'; })) conflicts.push('جستجو کلید ' + k + ' را ثبت‌نشده گزارش کرد ولی در دفتر کل ثبت‌شده است'); });
    // Completeness: every displayed Unknown item must be resolved by the lookup, and key-level results must cover and agree with the aggregate counts.
    r.items.forEach(function (it) { if (it[4] === 'unknown' && !by[it[1]]) conflicts.push('آیتم ' + it[1] + ' در نتیجهٔ جستجو نیست (نتیجهٔ جستجو ناقص است)'); });
    if (committed.length !== lk.committed || notC.length !== lk.notCommitted) conflicts.push('نتیجهٔ جستجو در سطح کلید (' + fa(committed.length) + ' ثبت‌شده / ' + fa(notC.length) + ' ثبت‌نشده) با شمارش‌های تجمیعی (' + fa(lk.committed) + ' / ' + fa(lk.notCommitted) + ') نمی‌خواند');
    // Set equality with the run SNAPSHOT: the lookup must classify exactly the run's intended keys (no omitted, no unrelated key).
    if (!r.keys) conflicts.push('snapshot اجرا مجموعهٔ کلیدهای نیت‌شده را ندارد؛ برابری قابل اثبات نیست');
    else {
      var miss = r.keys.filter(function (k) { return !(k in by); }), extra = keys.filter(function (k) { return r.keys.indexOf(k) < 0; });
      if (miss.length || extra.length) conflicts.push('کلیدهای نتیجهٔ جستجو با کلیدهای snapshot اجرا برابر نیست (کم: ' + fa(miss.length) + '، اضافه: ' + fa(extra.length) + ')');
    }
    // The complete key set AND the aggregate total must both equal the run intent — no outcome may be silently dropped.
    if (committed.length + notC.length !== r.intended || lk.committed + lk.notCommitted !== r.intended) conflicts.push('نتیجهٔ جستجو همهٔ آیتم‌های اجرا را پوشش نمی‌دهد (کلیدها ' + fa(committed.length + notC.length) + '، مجموع تجمیعی ' + fa(lk.committed + lk.notCommitted) + '، نیت اجرا ' + fa(r.intended) + ')');
    return { conflicts: conflicts };
  };
  // APPLY (only after a conflict-free plan): link the found transactions; never mint.
  var linkItems = function (id, test, tb) {
    var r = X.runView(X.run(id)), n = 0;
    setRun(id, { items: r.items.map(function (it) {
      if (!test(it)) return it; var c = it.slice(), tx = (tb || {})[it[1]]; if (!tx) return it;
      var byKey = M.ledger.filter(function (t) { return t.key === it[1]; });
      if (!byKey.some(function (t) { return t.st === 'committed'; })) {
        var intent = byKey.filter(function (t) { return t.st !== 'committed'; })[0];
        if (intent) { intent.id = tx; intent.st = 'committed'; intent.posted = 'یافت‌شده در تطبیق'; delete intent.note; }
        else M.ledger.push({ id: tx, key: it[1], dom: 'wallet', acct: 'کیف پول کمیسیون · گیرنده (یافت‌شده در تطبیق)', dir: 'C', amt: it[2], unit: it[3], rel: ['اجرا', r.id + ' · ' + it[0].split(' · ')[0]], actor: 'ثبت‌شده پیش از تطبیق', eff: '—', posted: 'یافت‌شده در تطبیق', st: 'committed' });
      }
      c[4] = 'existing'; c[5] = tx; c[6] = 'تراکنش موجود یافت شد و پیوند شد؛ اعتبار جدید ساخته نشد'; n++; return c;
    }) });
    return n;
  };
  var setItems = function (id, test, res, note) {
    var r = X.runView(X.run(id)), cnt = { posted: 0, existing: 0 };
    setRun(id, { items: r.items.map(function (it) {
      if (!test(it)) return it; var c = it.slice(); c[4] = res; c[6] = note;
      if (res === 'posted') { var m = mint(r, it); c[5] = m.id; if (m.existing) { c[4] = 'existing'; c[6] = 'تراکنش موجود یافت شد؛ اعتبار جدید ساخته نشد'; cnt.existing++; } else cnt.posted++; }
      else c[5] = null;
      return c;
    }) });
    return cnt;   // actual mint results — aggregate coverage is derived from these, not from precomputed totals
  };
  X.commitRunCore = function (sp) {
    var r = X.runView(X.run(sp.id)), op;
    // Commit-time recheck of the LIVE run: state, coverage and item results must still match what the confirmation showed.
    if (sp.srcSig !== X.runSig(r)) return X.newOp({ title: sp.title + ' ' + r.id + ' — تعارض هنگام ثبت', kind: 'run', run: r.id, conflict: true, c: { requested: 1, eligible: 0, ok: 0, existing: 0, skipped: 1, failed: 0, unknown: 0 }, note: 'اجرا پس از باز شدن تأیید تغییر کرده است (از «' + X.RUN[sp.srcSt].label + '» به «' + X.RUN[r.st].label + '» یا پوشش/آیتم‌ها عوض شده)؛ هیچ تغییری ثبت نشد و باید روی وضعیت فعلی دوباره بررسی شود.' });
    if (sp.kind === 'rungen') { setRun(r.id, { st: 'preview', intended: 31, cov: { intended: 31, processed: 0, posted: 0, existing: 0, failed: 0, unprocessed: 31, unknown: 0 }, sim: 'ok', items: [['INV-48230 · مرحله ۱ · فروشنده', 'RUN-312|INV-48230|S1|rec:HP-301', 900000, 'toman', 'notposted', null, '']], note: 'پیش‌نمایش تولید شد (فراداده نوشته شد). تأیید و ثبت هنوز انجام نشده است.' }); op = X.newOp({ title: 'تولید پیش‌نمایش ' + r.id, kind: 'run', run: r.id, c: { requested: 31, eligible: 31, ok: 31, existing: 0, skipped: 0, failed: 0, unknown: 0 }, note: 'فقط فراداده نوشته شد؛ هیچ تراکنش مالی ثبت نشد.' }); }
    else if (sp.kind === 'runapprove') { setRun(r.id, { st: 'approved', appr: { by: M.user.name, at: NOW, locked: NOW } }); op = X.newOp({ title: 'تأیید اجرا ' + r.id, kind: 'run', run: r.id, c: { requested: 1, eligible: 1, ok: 1, existing: 0, skipped: 0, failed: 0, unknown: 0 }, note: 'اجرا تأیید و قفل شد؛ هیچ اعتباری در کیف پول ثبت نشد.' }); }
    else if (sp.kind === 'runpost') {
      if (r.sim === 'unknown') { setItems(r.id, function () { return true; }, 'unknown', 'پاسخ نرسید؛ ممکن است ثبت شده باشد'); setRun(r.id, { st: 'unknown', cov: { intended: r.intended, processed: null, posted: null, existing: null, failed: null, unprocessed: null, unknown: r.intended }, lookup: M.mkLookup('RUN-309', r.keys, { 'RUN-309|INV-48180|S1|rec:HP-301': 'notCommitted', 'RUN-309|INV-48180|S1|rec:HP-210': 'committed' }, { 'RUN-309|INV-48180|S1|rec:HP-210': 'T-8802' }, 51, 35, 8900) }); op = X.newOp({ title: 'اجرای ثبت ' + r.id, kind: 'run', run: r.id, c: { requested: r.intended, eligible: r.intended, ok: 0, existing: 0, skipped: 0, failed: 0, unknown: r.intended }, note: 'اتصال پیش از دریافت پاسخ قطع شد؛ ممکن است بخشی یا همه ثبت شده باشد. تکرار کور مسدود است.' }); }
      else { var ex = Math.round(r.intended * 0.04), po = r.intended - ex, sr = setItems(r.id, function () { return true; }, 'posted', 'ثبت‌شد (نمایشی)'); po -= sr.existing; ex += sr.existing; setRun(r.id, { st: 'posted', cov: { intended: r.intended, processed: r.intended, posted: po, existing: ex, failed: 0, unprocessed: 0, unknown: 0 } }); op = X.newOp({ title: 'اجرای ثبت ' + r.id, kind: 'run', run: r.id, c: { requested: r.intended, eligible: r.intended, ok: po, existing: ex, skipped: 0, failed: 0, unknown: 0 }, note: 'تراکنش موجود دوباره نوشته نشد؛ پوشش با شناسهٔ تراکنش هر آیتم اثبات می‌شود.' }); }
    } else if (sp.kind === 'runtail') {
      var cc = r.cov, n = cc.unprocessed, st2 = setItems(r.id, function (it) { return it[4] === 'unprocessed'; }, 'posted', 'دنباله ثبت شد (نمایشی)'), ex2 = (r.id === 'RUN-307' ? 8 : 0) + st2.existing, po2 = n - ex2, nc = { intended: cc.intended, processed: cc.intended, posted: cc.posted + po2, existing: cc.existing + ex2, failed: cc.failed, unprocessed: 0, unknown: 0 }; setRun(r.id, { st: nc.failed ? 'partial' : 'posted', cov: nc, tail: false }); op = X.newOp({ title: 'ثبت دنبالهٔ ' + r.id, kind: 'run', run: r.id, c: { requested: n, eligible: n, ok: po2, existing: ex2, skipped: 0, failed: 0, unknown: 0 }, note: nc.failed ? 'دنباله ثبت شد؛ ' + fa(nc.failed) + ' آیتم ناموفق قبلی باقی است و تصمیم جدا می‌خواهد (تکرار خودکار نیست).' : 'همهٔ آیتم‌ها اکنون با نتیجهٔ مشخص ثبت‌شده‌اند.' });
    }
    else if (sp.kind === 'runrecon') {
      var lk = st.lookups[r.id] || { committed: 0, notCommitted: r.intended }, first = true;
      if (sp.lkSig != null && (!st.lookups[r.id] || sp.lkSig !== X.lookupSig(st.lookups[r.id]))) return X.newOp({ title: 'تطبیق — تعارض هنگام ثبت', kind: 'run', run: r.id, conflict: true, c: { requested: r.intended, eligible: r.intended, ok: 0, existing: 0, skipped: 0, failed: 0, unknown: r.intended }, note: 'نتیجهٔ جستجویی که تأیید شد با نتیجهٔ فعلی یکی نیست (جایگزین یا بازخوانی شده)؛ هیچ چیز تغییر نکرد و نتیجهٔ اجرا نامعلوم می‌ماند.' });
      // PLAN first (pure). A contradictory lookup is reported at operation/run level only: ledger, items and classification stay untouched.
      var by = lk.byKey || {}, plan = X.planReconcile(r, lk);
      if (plan.conflicts.length) {
        setRun(r.id, { note: 'تطبیق انجام نشد: نتیجهٔ جستجو با دفتر کل متناقض است (' + plan.conflicts.join('؛ ') + ')؛ هیچ پیوند یا طبقه‌بندی اعمال نشد و اجرا «نتیجه نامعلوم» می‌ماند.' }); delete st.lookups[r.id];
        return X.newOp({ title: 'تطبیق ناموفق ' + r.id, kind: 'run', run: r.id, conflict: true, c: { requested: r.intended, eligible: r.intended, ok: 0, existing: 0, skipped: 0, failed: 0, unknown: r.intended }, note: 'تعارض (CONFLICT): ' + plan.conflicts.join('؛ ') + '. هیچ چیز تغییر نکرد؛ نتیجهٔ اجرا نامعلوم می‌ماند و تکرار کور مسدود است.' });
      }
      linkItems(r.id, function (it) { return it[4] === 'unknown' && by[it[1]] === 'committed'; }, lk.txByKey);
      setItems(r.id, function (it) { return it[4] === 'unknown' && by[it[1]] === 'notCommitted'; }, 'unprocessed', 'ثبت‌نشدن اثبات شد');
      var still = X.runView(X.run(r.id)).items.filter(function (it) { return it[4] === 'unknown'; }).length;
      if (still) { setRun(r.id, { note: 'جستجو برای ' + fa(still) + ' آیتم نمونه نتیجهٔ قطعی نداد؛ اجرا «نتیجه نامعلوم» می‌ماند و تطبیق کامل نشده است.' }); delete st.lookups[r.id]; op = X.newOp({ title: 'تطبیق ناقص ' + r.id, kind: 'run', run: r.id, c: { requested: r.intended, eligible: r.intended, ok: 0, existing: 0, skipped: 0, failed: 0, unknown: r.intended }, note: 'برخی کلیدها حل نشد؛ نتیجهٔ اجرا نامعلوم می‌ماند و تکرار کور مسدود است.' }); return op; }
      setRun(r.id, { st: 'reconciled', tail: lk.notCommitted > 0, cov: { intended: r.intended, processed: lk.committed, posted: 0, existing: lk.committed, failed: 0, unprocessed: lk.notCommitted, unknown: 0 }, note: 'با جستجوی تراکنش/کلید تطبیق شد: ' + fa(lk.committed) + ' آیتم ثبت‌شده بود و ثبت‌نشدن ' + fa(lk.notCommitted) + ' آیتم اثبات شد.' });
      delete st.lookups[r.id];
      op = X.newOp({ title: 'ثبت تطبیق ' + r.id, kind: 'run', run: r.id, c: { requested: r.intended, eligible: r.intended, ok: 0, existing: lk.committed, skipped: lk.notCommitted, failed: 0, unknown: 0 }, note: 'فقط تطبیق ثبت شد؛ هیچ اعتبار جدیدی نوشته نشد. ' + fa(lk.notCommitted) + ' آیتم ثبت‌نشدهٔ اثبات‌شده برای ادامهٔ جدا می‌ماند.' });
    }
    return op;
  };
  var RUNACT = { rungen: 'تولید پیش‌نمایش اجرا', runapprove: 'تأیید اجرا', runpost: 'ثبت اجرا (APPLY)', runtail: 'ثبت دنبالهٔ اجرا', runrecon: 'ثبت نتیجهٔ تطبیق نامعلوم' };
  X.commitRun = function (sp) {
    var before = X.runView(X.run(sp.id)).st, op = X.commitRunCore(sp), after = X.runView(X.run(sp.id)), c = op ? X.counts(op) : null;
    if (op) {   // EVERY run action (success, partial, unknown, conflict, incomplete) leaves actor/reason/before→after/outcome/correlation
      var meta = sp.kind === 'rungen' || sp.kind === 'runapprove', unk = after.st === 'unknown' || (c && c.unknown), tx;
      if (meta) tx = 'هیچ (' + (sp.kind === 'rungen' ? 'فقط فراداده' : 'تأیید ≠ ثبت') + ')';
      else if (unk) tx = 'نامعلوم (ممکن است ثبت شده باشد)' + (op.conflict ? ' — این تلاش بدون تغییر بود' : '');
      else if (op.conflict) tx = 'هیچ (تعارض؛ بدون تغییر)';
      else tx = (c.ok ? fa(num(c.ok)) + ' جدید' : '') + (c.ok && c.existing ? ' · ' : '') + (c.existing ? fa(num(c.existing)) + ' موجود' : '') || 'هیچ';
      pushAudit({ kind: RUNACT[sp.kind] || sp.kind, inv: '—', cs: '—', ev: '—', amt: c ? fa(num(c.requested)) + ' آیتم' : '—', ba: X.RUN[before].label + ' ← ' + X.RUN[after.st].label + (op.conflict ? ' (تعارض؛ بدون تغییر)' : ''), reason: sp.reasonText || '—', rev: '—', tx: tx, key: sp.id + '|…', run: sp.id, res: (sp.kind === 'runrecon' && !op.conflict) ? (after.st === 'reconciled' ? 'تطبیق کامل شد (موفق)' : 'تطبیق ناقص — نتیجه نامعلوم') : X.OPS_S(X.opState(op))[0], corr: 'C-' + (7900 + M.audit.length) });   // reconciliation outcome is judged on reconciliation, not posting-oriented counts
    }
    return op;
  };
  // Read-only lookup: result is DERIVED view state (st.lookups) — it changes no run, ledger or audit record. Recording the reconciliation is a separate authorised step.
  X.runLookup = function (id) { var r = X.runView(X.run(id)); st.lookups[id] = r.lookup || { committed: 0, notCommitted: r.intended }; };
  X.lookupSig = function (lk) { var st2 = function (o) { return JSON.stringify(Object.keys(o || {}).sort().map(function (k) { return [k, o[k]]; })); }; return [lk.committed, lk.notCommitted, st2(lk.byKey), st2(lk.txByKey)].join('|'); };
  X.runReconSpec = function (r) {
    var lk = st.lookups[r.id];
    return runSpec('runrecon', r, { lkSig: X.lookupSig(lk), title: 'ثبت نتیجهٔ تطبیق', needsReason: true, reasonLabel: 'مبنای تطبیق (نتیجهٔ جستجو)', target: r.id + ' · ' + r.name, current: 'نتیجه نامعلوم', resulting: 'تطبیق‌شده: ' + fa(num(lk.committed)) + ' آیتم ثبت‌شده، ' + fa(num(lk.notCommitted)) + ' آیتم ثبت‌نشدهٔ اثبات‌شده',
      changes: ['وضعیت اجرا: نتیجه نامعلوم ← تطبیق‌شده', 'نتیجهٔ هر آیتم از جستجوی تراکنش/کلید ثبت می‌شود', 'آیتم‌هایی که ثبت‌شده یافت شدند به تراکنش واقعی‌شان وصل می‌شوند'], unchanged: ['هیچ اعتبار جدیدی ثبت نمی‌شود', 'تراکنش‌های موجود دوباره نوشته نمی‌شوند', 'تکرار ثبت انجام نمی‌شود'], affected: ['اجرا ' + r.id, 'آیتم‌های نامعلوم', 'ممیزی'],
      irrev: ['نتیجهٔ تطبیق در ممیزی ثبت می‌شود', 'تکرار پس از آن فقط برای دنبالهٔ اثبات‌شدهٔ ثبت‌نشده و با تأیید جدا است'], ack: 'نتیجهٔ جستجو را بررسی کردم؛ می‌دانم این گام فقط تطبیق را ثبت می‌کند و ثبت جدیدی نیست.', commit: 'ثبت نتیجهٔ تطبیق (نمایشی)' });
  };
  /* ---------- Reconciliation issue / diagnosis ---------- */
  D.iss = function (id) {
    var i = X.iss(id), c = X.ISS[i.cls], s = X.ISST[i.state];
    var ent = i.ent[0] === 'stage' ? open('rs', i.ent[1], 'باز کردن مرحله') : i.ent[0] === 'run' ? open('run', i.ent[1], 'باز کردن اجرا') : i.ent[0] === 'tx' && X.tx(i.ent[1]) ? open('tx', i.ent[1], 'ردیابی تراکنش') : i.ent[0] === 'refund' ? open('ref', i.ent[1], 'باز کردن استرداد') : '';
    return top('مسئلهٔ تطبیق') + head(esc(i.title), pill(c[0], i.sev, c[1]) + pill(s[0], s[1], s[2]) + X.lk(i.conf), '<span class="mono">' + i.id + '</span>') + '<div class="dr-body">' +
      sec('شواهد', '', checks(i.ev.map(function (e) { return ['info', e, '']; })), 'primary') + sec('اثر مالی', '', ctxLine(esc(i.impact))) + sec('مالک / حوزه', '', X.dl([['مالک', esc(i.owner)], ['گام بعد (مسئول)', esc(i.next)], ['موجودیت متأثر', '<span class="mono">' + esc(i.ent[1]) + '</span> ' + esc(i.ent[2])]]) + '<div class="resp-line">' + ent + '</div>') +
      sec('اقدام مجاز مالی', '', '<ul class="allowed-list">' + i.allowed.map(function (a) { return '<li class="allowed">' + ic('checkCircle') + esc(a) + '</li>'; }).join('') + '</ul>') + sec('اقدام ممنوع', '', '<ul class="allowed-list">' + i.restricted.map(function (a) { return '<li class="allowed no">' + ic('ban') + esc(a) + '</li>'; }).join('') + '</ul>') + sec('مدرک حل', '', ctxLine(esc(i.proof) + ' — بستن ظاهری استثنا حل‌شدن نیست.')) + '</div>' + foot('<button type="button" class="btn btn-primary btn-lg" data-act="issue-handoff:' + i.id + '">' + ic('send') + 'ثبت ارجاع به مالک (نمایشی)</button>', null, 'رفع همه وجود ندارد · هیچ دادهٔ مالی تغییر نمی‌کند');
  };
  D.diag = function () {
    var rows = X.keys(st.isel).map(X.iss).map(function (i) { var c = X.ISS[i.cls]; return '<tr><td>' + pill(c[0], i.sev, c[1]) + '<span class="cell-sub mono">' + i.id + '</span></td><td class="wrap">' + esc(i.title) + '</td><td class="wrap">' + esc(i.allowed[0]) + '</td><td class="wrap">' + esc(i.next) + '</td></tr>'; }).join('');
    return top('تشخیص گروهی (فقط‌خواندنی)') + head('تشخیص گروهی', pill('فقط‌خواندنی', 'teal', 'eye'), '') + '<div class="dr-body"><section class="sec primary">' + ctxLine('گروه‌بندی برای بررسی است. اصلاح گروهی و «رفع همه» برای مسائل ناهمگن مالی مجاز نیست؛ هر مسئله مدرک حل و مالک خودش را دارد.') + '<div class="tbl-wrap"><table class="tbl no-cursor" aria-label="تشخیص گروهی"><caption class="sr">مسائل انتخاب‌شده با اقدام مجاز و مسئول بعدی</caption><thead><tr><th>نوع</th><th>مسئله</th><th>اقدام مجاز اول</th><th>گام بعد</th></tr></thead><tbody>' + rows + '</tbody></table></div></section></div>' + foot('', null, 'هیچ تغییری ثبت نشد');
  };

  /* ---------- Report context / Audit event ---------- */
  D.rep = function (id) {
    var r = X.rep(id), rows = r.rows;
    if (r.id === 'REP-WL') { var all = M.queue.map(X.eff), n = function (f) { return all.filter(f).length; }; rows = [['در انتظار بررسی', fa(n(function (x) { return x.review === 'pending'; })) + ' مرحله'], ['رسید بارگذاری‌شده (هم‌پوشان)', fa(n(function (x) { return x.ev && x.ev.k === 'receipt'; })) + ' مرحله'], ['ثبت درگاهی (هم‌پوشان)', fa(n(function (x) { return x.ev && x.ev.k === 'gateway'; })) + ' مرحله']]; }
    var ctx = '<div class="trust-grid" role="group" aria-label="زمینهٔ اعتماد گزارش">' + [['واحد شمارش (grain)', r.grain], ['محدوده', 'دامنهٔ مالی مجاز'], ['مبنای زمانی', r.time], ['واحد', r.unit], ['منبع', r.src], ['تازگی', r.fresh === 'stale' ? 'قدیمی — بازخوانی لازم' : 'امروز ' + M.freshness.now], ['پوشش', { ok: 'کامل', partial: 'ناقص', bounded: 'محدود (زیرمجموعه)' }[r.cov]], ['وضعیت تطبیق', r.rec]].map(function (x) { return '<div class="facet"><span class="muted">' + x[0] + '</span><b>' + esc(x[1]) + '</b></div>'; }).join('') + '</div>';
    var trs = rows.map(function (x) { return '<tr><td class="wrap">' + esc(x[0]) + '</td><td class="wrap">' + (/نامعلوم/.test(x[1]) ? '<span class="unk">' + esc(x[1]) + '</span>' : esc(x[1])) + '</td></tr>'; }).join('');
    return top('زمینهٔ گزارش') + head(esc(r.name), X.basis(r.basis) + X.cov(r.fresh === 'stale' ? 'stale' : r.cov), '<span class="mono">' + r.id + '</span>') + '<div class="dr-body">' + sec('زمینهٔ اعتماد', '', ctx, 'primary') + (r.note ? '<section class="sec">' + h.banner('incomplete', esc(r.note), '') + '</section>' : '') + sec('نمونهٔ شاخص‌ها (مثال ساختگی)', '', '<div class="tbl-wrap"><table class="tbl no-cursor" aria-label="شاخص‌ها"><caption class="sr">شاخص‌های نمونه، به تفکیک واحد</caption><thead><tr><th>شاخص</th><th>مقدار</th></tr></thead><tbody>' + trs + '</tbody></table></div>' + ctxLine('واحدهای ناسازگار جمع نمی‌شوند؛ نامعلوم صفر نیست؛ ' + (r.id === 'REP-SR' ? 'آمادگی تسویه ≠ تسویه پرداخت‌شده.' : 'خروجی هم‌پوشان‌ها قابل جمع نیست.'))) + sec('خروجی', '', ctxLine(esc(r.export)) + '<div class="resp-line">' + X.guardBtn('export', 'btn-soft', 'x', 'خروجی', 'download', true) + '</div>') + '</div>' + foot('', null, 'فقط‌خواندنی · داده ساختگی');
  };
  D.audit = function (id) {
    var a = X.audit(id), f = function (v) { return v && v !== '—' ? esc(v) : '<span class="muted">ثبت نشده</span>'; };
    return top('رویداد ممیزی') + head(esc(a.kind), pill(a.res, a.res === 'موفق' ? 'green' : 'amber', a.res === 'موفق' ? 'checkCircle' : 'question'), '<span class="mono">' + a.id + '</span><span>' + esc(a.at) + '</span>') + '<div class="dr-body">' + sec('زمینهٔ ممیزی (فیلدهای لازم)', '', X.dl([['عامل', f(a.actor)], ['فاکتور', f(a.inv)], ['Case / منبع', f(a.cs)], ['مدرک (مرجع/نسخه)', f(a.ev)], ['مبلغ · واحد', f(a.amt)], ['قبل ← بعد', f(a.ba)], ['دلیل', f(a.reason)], ['وضعیت بررسی', f(a.rev)], ['شناسه تراکنش', f(a.tx)], ['کلید کسب‌وکار', f(a.key)], ['اجرا', f(a.run)], ['نتیجه', f(a.res)], ['ارجاع همبستگی', f(a.corr)], ['زمان ثبت', f(a.at)]]) + ctxLine('فیلد خالی «ثبت نشده» است و از مالک فعلی یا حدس پر نمی‌شود. ماندگاری ممیزی در محیط واقعی تأیید نشده است.'), 'primary') + '</div>' + foot('', null, 'فقط‌خواندنی');
  };
})();
