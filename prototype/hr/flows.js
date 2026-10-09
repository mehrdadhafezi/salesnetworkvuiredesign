/* HR — role layer, part 3: drawers and guarded flows (workforce profile, hierarchy change preview, termination impact, request review,
   access-state inspector, compensation history, restricted credentials/impersonation, onboarding, bulk) + Sensitive Action Confirmation.
   Prototype only: nothing is saved or sent. Preview ≠ apply; approval of a step ≠ applied change; apply is re-checked; Outcome Unknown is reconciled first. */
(function () {
  'use strict';
  var X = window.HRX, C = X.C, M = X.M, h = X.h, st = X.st, D = X.D;
  var esc = h.esc, fa = h.fa, ic = h.ic, pill = h.pill, hint = h.hint, $ = h.$;
  var top = X.top, foot = X.foot, sec = X.sec, tl = X.tl, checks = X.checks;
  var head = function (title, badges, meta) { return '<div class="dr-head"><div class="dr-title"><h2 id="dr-name">' + title + '</h2>' + (badges || '') + '</div>' + (meta ? '<div class="dr-meta">' + meta + '</div>' : '') + '</div>'; };
  var btn = function (cls, act, label, iconName, dis) { return '<button type="button" class="btn ' + cls + '" data-act="' + act + '"' + (dis ? ' disabled aria-disabled="true"' : '') + '>' + (iconName ? ic(iconName) : '') + label + '</button>'; };
  var keepOf = function () { return C.state.drawer ? C.state.drawer.keep : {}; };
  var now = 'همین الان';
  var ok3 = function (s) { return (s || '').trim().length > 2; };

  /* ---------- Operation engine: per-effect/item truth. requested = applied + skipped + rejected + failed + unknown. ---------- */
  var seq = 200;
  X.newOp = function (o) { o.id = 'OP-' + (++seq); o.ts = now; st.ops.unshift(o); return o; };
  X.opOf = function (id) { return st.ops.filter(function (o) { return o.id === id; })[0]; };
  X.opState = function (op) { if (op.blocked) return 'blocked'; if (op.unrecon) return 'partial'; var c = X.counts(op); return c.unknown ? 'unknown' : c.ok === c.requested ? 'complete' : c.ok === 0 && c.requested ? 'failed' : 'partial'; };
  var pushAudit = function (a) { a.id = 'A' + (20 + M.audit.length); a.at = now; a.actor = M.user.name; M.audit.unshift(a); return a; };


  // The operation items are the only claim of what happened; domain state follows them (applied once), never the other way round.
  // Close the active HR review step once the request is finally settled (idempotent: only a step still «cur» is closed; history text is appended, never replaced).
  X.closeReqChain = function (r) {
    r.chain.forEach(function (c) { if (c[1] === 'cur') { c[1] = 'done'; c[2] = c[2] + ' · اعمال نهایی ' + now + ' · ' + M.user.name; } });
    r.cur = '—'; r.next = '—'; r.settled = true;
  };
  // Access rows follow the operation items: only permissions whose item is confirmed OK take the expected value; the rest keep their previous effective state.
  X.syncAccess = function (op, p) {
    var A = M.access[p.id]; if (!A) A = M.access[p.id] = { rows: JSON.parse(JSON.stringify(M.access['default'].rows)) };
    if (!A.own) { A.rows = JSON.parse(JSON.stringify(A.rows)); A.own = true; }
    var it = op.items, keys = op.keys, exp = op.expected, mappable = !!(keys && exp && it.length - 1 === Math.max(keys.length, 1)), pending = [];
    if (mappable) keys.forEach(function (k, i) { var x = it[i + 1]; if (x && x[1] === 'ok') A.rows[k] = ['position', exp[k]]; else pending.push(X.permLabel(k) + (x && x[1] === 'unknown' ? ' (نامعلوم)' : '')); });
    var allOk = it.every(function (x) { return x[1] === 'ok'; });
    op.unrecon = !mappable;
    if (allOk && mappable) { p.role = 'ok'; delete A.partial; }
    else { p.role = 'partial'; A.partial = !mappable ? 'وضعیت مؤثر قابل بازسازی نیست؛ نتیجه ناقص/نامعلوم می‌ماند تا بازخوانی' : 'مجوزهای اعمال‌نشده یا نامعلوم: ' + (pending.join('، ') || 'سمت'); }
  };
  var c0 = function (p) { return M.comp[p.id]; };
  X.permLabel = function (k) { var r = M.perms.filter(function (x) { return x.k === k; })[0]; return r ? r.label : k; };

  // The operation items are the only claim of what happened; domain state follows them (applied once), never the other way round.
  X.settleOp = function (op) {
    var it = op.items, p = op.p ? X.s(op.p) : null, okk = function (i) { return it[i] && it[i][1] === 'ok'; };
    if (op.kind === 'xfer' && !op.done) {
      // Closing the old interval is its own confirmed effect: it is kept even when the insert outcome is Unknown/Failed (no re-close, no duplicate interval).
      if (okk(0) && !op.closed) { if (!p.hist[0].to) p.hist[0].to = now; op.closed = true; }
      if (op.closed) {
        if (okk(1)) {
          var ph = op.unk ? p.hist.indexOf(op.unk) : -1; if (ph > -1) p.hist.splice(ph, 1); op.unk = null;
          p.hist.unshift({ from: now, to: null, parent: op.t, pos: X.posL(p.pos), applied: now, actor: M.user.name, src: op.reqId ? 'درخواست ' + op.reqId : 'ویرایش مستقیم', state: 'known' }); p.parent = op.t; p.parentUnknown = false; delete p.unkOp; op.done = true;
          if (op.audit) { op.audit.after = 'مدیر: ' + X.name(op.t); op.audit.applied = now; }
          if (op.reqId) { var r = X.req(op.reqId); st.reqState[r.id] = 'approved'; r.appliedAt = now; r.actual = { before: op.fromName, after: X.name(op.t), ok: true }; X.closeReqChain(r); }
        } else {
          var failed = it[1] && it[1][1] === 'failed', src = 'بازه قبلی بسته شد؛ ' + (failed ? 'بازه جدید ثبت نشد — مدیر مؤثر تعیین نشده' : 'ثبت بازه جدید نامعلوم است — مدیر مؤثر تا تطبیق نامعلوم');
          if (!op.unk) { p.unkOp = op.id; op.unk = { from: now, to: null, parent: null, pos: '—', state: 'unknown', src: src }; p.hist.unshift(op.unk); } else op.unk.src = src;
          p.parent = null; p.parentUnknown = true;
          if (op.audit) op.audit.after = 'مدیر مؤثر نامعلوم (بازه پیشین بسته شد؛ بازه جدید ' + (failed ? 'ثبت نشد' : 'نامعلوم') + ')';
        }
      }
    }
    if (op.kind === 'comp' && !op.done && okk(1)) {
      var c = M.comp[p.id], old = c.periods[0]; c.periods.unshift({ from: op.from, to: null, base: op.base, reason: op.reasonText || '', actor: M.user.name, applied: now, before: old ? old.base : '—' }); c.base = op.base; op.done = true;
      if (op.audit) { op.audit.after = 'پایه: ' + op.base; op.audit.applied = now; }
    }
    if (op.kind === 'comp' && op.done && c0(p)) delete c0(p).unresolved;
    if (op.kind === 'access') X.syncAccess(op, p);
    if (op.audit) op.audit.res = X.opState(op);
  };

  D.result = function (id) {
    var op = X.opOf(id) || st.ops[0]; if (!op) return top('نتیجه عملیات') + '<div class="dr-body">' + h.stateBlock('empty', 'نتیجه‌ای نیست', '') + '</div>';
    var c = X.counts(op), s = X.OPS_S(X.opState(op)), groups = [['ok', 'اعمال‌شده'], ['skipped', 'ارسال‌نشده'], ['rejected', 'ردشده'], ['failed', 'ناموفق'], ['unknown', 'نامعلوم']];
    var gl = groups.filter(function (g) { return c[g[0]]; }).map(function (g) {
      var list = op.items.filter(function (x) { return x[1] === g[0]; }), o = X.OUT[g[0]];
      return '<details class="sec"' + (g[0] === 'ok' ? '' : ' open') + '><summary><h3>' + pill(o.label, o.tone, o.icon) + '</h3><span class="muted" style="margin-right:8px;font-size:var(--t-meta)">' + fa(list.length) + '</span><span class="chev">' + ic('chev') + '</span></summary><ul class="ir-list">' + list.map(function (x) { return '<li class="ir"><span>' + esc(x[0]) + '</span><span class="ir-why">' + esc(x[2] || '—') + (x[3] ? ' · پیامک: ' + esc(x[3] === 'ok' ? 'ارسال شد' : x[3] === 'failed' ? 'ناموفق (جدا از بازنشانی)' : 'ارسال نشد') : '') + '</span></li>'; }).join('') + '</ul></details>';
    }).join('');
    var acts = '';
    if (c.unknown) acts += btn('btn-primary btn-lg', 'reconcile-op:' + op.id, 'تطبیق نتیجه نامعلوم (خواندن وضعیت واقعی)', 'refresh');
    if (c.failed && !c.unknown && op.retry !== false) acts += btn('btn-primary btn-lg', 'retry-op:' + op.id, 'تکرار فقط ناموفق‌های معلوم', 'refresh');
    var smsLine = op.sms ? '<p class="ind-note">' + ic('info') + ' اعلان پیامکی جدا از بازنشانی: ' + fa(op.sms.ok) + ' ارسال، ' + fa(op.sms.failed) + ' ناموفق. شکست پیامک شکست بازنشانی نیست.</p>' : '';
    return top('نتیجه عملیات') + head(esc(op.title), pill(s[0], s[1], s[2]), '<span class="mono">' + op.id + '</span><span>' + esc(op.ts) + '</span>') + '<div class="dr-body"><section class="sec primary"><p class="result-line">' + ic(c.ok === c.requested ? 'checkCircle' : 'alert') + (c.ok === c.requested ? 'همه ' + fa(c.requested) + ' اثر اعمال شد.' : fa(c.ok) + ' از ' + fa(c.requested) + ' اثر اعمال شد؛ بقیه به‌تفکیک زیر آمده‌اند.') + '</p>' + X.outcomeStrip(op) +
      (c.unknown ? '<div class="note warn inset">' + ic('question') + '<span><b>نتیجه نامعلوم است.</b> ممکن است ثبت شده باشد؛ ابتدا وضعیت واقعی را بخوانید و تکرار کور انجام ندهید.</span></div>' : '') + (op.note ? '<p class="ind-note">' + ic('info') + ' ' + esc(op.note) + '</p>' : '') + smsLine + '</section>' + gl + '</div>' + foot(acts, null, 'شماره ارجاع ' + op.id);
  };
  X.retryFailed = function (op) { op.items.forEach(function (x) { if (x[1] === 'failed') { x[1] = 'ok'; x[2] = 'تکرار پس از بررسی تازه'; } }); X.settleOp(op); };
  X.reconcileOp = function (op) { var n = 0; op.items.forEach(function (x) { if (x[1] === 'unknown') { x[1] = n++ % 2 === 0 ? 'ok' : 'failed'; x[2] = x[1] === 'ok' ? 'تطبیق: ثبت شده بود' : 'تطبیق: ثبت نشده بود؛ اکنون قابل تکرار'; } }); op.note = (op.note || '') + ' نتیجه نامعلوم با خواندن وضعیت واقعی تطبیق شد.'; X.settleOp(op); };

  /* ---------- Workforce profile ---------- */
  D.staff = function (id) {
    var p = X.s(id), pos = X.pos(p.pos), emp = X.empOf(p), parent = p.parent ? X.s(p.parent) : null;
    var cell = function (l, t, v) { return '<div class="own"><span class="own-l">' + ic('info') + esc(l) + hint(t) + '</span><b>' + v + '</b></div>'; };
    var grid = '<div class="own-grid own7">' + cell('سمت', 'سمت کاری؛ با نقش و مجوز یکی نیست', esc(pos.label)) + cell('سطح', 'طبقه کاری', esc(p.lvl)) + cell('نقش عملیاتی نگاشت‌شده', 'نقش وردپرس/قدیمی؛ سمت نیست', esc(pos.role)) + cell('واحد / تیم', 'لزوماً همان والد گزارش‌دهی نیست', esc(p.unit)) + cell('مدیر فعلی', 'والد فعلی؛ اعتبار تاریخی نیست', p.parentUnknown ? '<b>نامعلوم (UNKNOWN)</b>' : parent ? esc(parent.name) + (X.isMgrOk(parent.id) ? '' : ' <span class="neg">(غیرفعال)</span>') : 'ثبت نشده') + cell('وضعیت اشتغال', 'غیرفعال بودن به معنی لغو کامل ورود نیست', X.empPill(emp)) + cell('وضعیت دسترسی', 'دسترسی واقعی با پرچم فعال بودن یکی نیست', X.accPill(p.role === 'nouser' ? 'nouser' : p.role)) + '</div>';
    var actions = '<div class="resp-line">' + btn('btn-soft', 'open-xfer:' + p.id, 'پیش‌نمایش تغییر مدیر', 'swap') + (p.pos === 'seller' ? btn('btn-soft', 'open-term:' + p.id, 'بررسی اثر پایان همکاری', 'lock') : '') + btn('btn-soft', 'open-access:' + p.id, 'بازرس دسترسی', 'key') + (M.comp[p.id] ? btn('btn-soft', 'open-comp:' + p.id, 'جبران خدمات', 'wallet') : '') + '</div>';
    var unkNote = p.parentUnknown && p.unkOp ? '<section class="sec"><div class="note warn">' + ic('question') + '<span><b>مدیر مؤثر نامعلوم است.</b> نتیجه ثبت بازه جدید باید تطبیق شود.</span>' + btn('btn-sm', 'open-op:' + p.unkOp, 'باز کردن نتیجه عملیات و تطبیق', 'refresh') + '</div></section>' : '';
    var confl = p.conf.length ? sec('ناهمخوانی‌های باز', '', checks(p.conf.map(function (k) { return ['warn', X.CONF[k], k === 'dupProfile' ? 'دو پروفایل ' + p.id + ' و ' + p.dup + ' به یک حساب اشاره می‌کنند؛ ادغام مخرب انجام نمی‌شود' : k === 'noUser' ? 'ورود نیرو ناتمام؛ حساب پیوند نشده' : k === 'inactiveMgr' ? 'مدیر فعلی غیرفعال است؛ جانشین خودکار تعیین نمی‌شود' : k === 'termOpen' ? 'کار باز دارد؛ تحویل (OPD-05) تعریف نشده' : k === 'histGap' ? 'مدیر و سمت پیش از ۱۴۰۵/۰۳/۰۱ نامشخص است' : k === 'roleMismatch' ? 'نقش قدیمی باقی مانده؛ بازنگاشت اجباری نمی‌شود' : k === 'accessPartial' ? 'مجوزهای مورد انتظار کامل اعمال نشد' : 'والد ثبت نشده؛ لزوماً نقص نیست (بسته به سمت)']; }))) : '';
    return top('پروفایل نیرو') + head(esc(p.name), X.empPill(emp) + X.accPill(p.role === 'nouser' ? 'nouser' : p.role), '<span class="mono">' + p.id + '</span><span>موبایل ' + p.mob + ' (فقط تطبیق)</span>') + '<div class="dr-body">' + sec('پیوند هویت', X.basis('snap'), X.linkage(p) + '<p class="ind-note">' + ic('info') + ' شخص، پروفایل نیرو و حساب وردپرس سه شناسه مستقل‌اند؛ برابری شناسه عددی در دو جدول یکی بودن آن‌ها نیست. هویت شخص سراسری هنوز تعریف نشده.</p>', 'primary') + unkNote + sec('موجودیت‌های جدا', '', grid + actions) + confl +
      sec('تاریخچه اشتغال و ساختار', X.basis('hist'), X.eit(p) + '<p class="ind-note">' + ic('lock') + ' ساختار فعلی انتساب، مالک اعتبار و عامل رویداد گذشته را بازنویسی نمی‌کند. بازه نامعلوم از مدیر امروز پر نمی‌شود.</p>') + '</div>' + foot('', null, 'اجرای واقعی در نمونه انجام نمی‌شود · HR runtime NOT LIVE VERIFIED');
  };

  /* ---------- Sensitive Action Confirmation (generic: what changes / what does not / affected / effective time / audit reason) ---------- */
  X.sensBlocks = function (sp, keep) {
    var list = function (t, items, tone, icon) { return '<div class="sens-c sens-' + tone + '"><h4>' + ic(icon) + t + '</h4><ul>' + items.map(function (x) { return '<li>' + esc(x) + '</li>'; }).join('') + '</ul></div>'; };
    return '<div class="sens-grid">' + list('چه چیزی تغییر می‌کند', sp.changes, 'chg', 'edit') + list('چه چیزی تغییر نمی‌کند', sp.unchanged, 'keep', 'lock') + list('موجودیت‌های متأثر', sp.affected, 'aff', 'users') + '<div class="sens-c sens-eff"><h4>' + ic('clock') + 'زمان اثر</h4><p>' + esc(sp.effective) + '</p></div></div>' +
      '<div class="sens-form"><label class="lbl" for="sens-r">دلیل (برای ممیزی · الزامی)</label><textarea class="input" id="sens-r" rows="2" data-sensreason aria-required="true"' + (keep.err && !ok3(keep.reason) ? ' aria-invalid="true"' : '') + '>' + esc(keep.reason || '') + '</textarea>' + (keep.err && !ok3(keep.reason) ? '<p class="field-err" role="alert">دلیل لازم است.</p>' : '') +
      '<label class="confirm"><input type="checkbox" class="cbx" data-sensack' + (keep.ack ? ' checked' : '') + '><span>' + esc(sp.ack || 'اثر بالا را خواندم و می‌دانم نتیجه ممکن است ناقص باشد.') + '</span></label></div>';
  };
  D.sens = function () {
    var sp = st.spec, keep = keepOf(), ready = ok3(keep.reason) && keep.ack;
    return top('تأیید با بررسی اثر') + head(esc(sp.title), X.zone(sp.zone || 'cond'), '<span>' + esc(sp.subject) + '</span>') + '<div class="dr-body">' + sp.pre + X.sensBlocks(sp, keep) + '</div>' + foot(btn('btn-primary btn-lg', 'commit-sens', sp.commit, 'send', !ready), null, 'پیش‌نمایش رزرو یا مجوز نهایی نیست · هنگام ثبت دوباره بررسی می‌شود');
  };
  X.openSens = function (sp) { st.spec = sp; C.openDrawer('sens', sp.kind + ':' + (sp.id || 'x'), {}); };

  /* ---------- Hierarchy Change Preview (transfer) ---------- */
  D.xfer = function (id) {
    var p = X.s(id), keep = keepOf(), cands = M.staff.filter(function (x) { return x.id !== p.id && ['manager', 'sup', 'senior', 'deputy'].indexOf(x.pos) > -1; });
    var t = keep.target ? X.s(keep.target) : null, o = p.open, cur = p.parent ? X.s(p.parent) : null;
    var cyc = t && X.isDesc(t.id, p.id), inact = t && !X.isMgrOk(t.id), same = t && t.id === p.parent, effc = st.flow === 'effconflict' && t;
    var list = '<div class="recips-inline" role="radiogroup" aria-label="مدیر مقصد">' + cands.map(function (c) {
      var okc = X.isMgrOk(c.id) && !X.isDesc(c.id, p.id), on = keep.target === c.id;
      return '<label class="recip' + (okc ? '' : ' off') + (on ? ' on' : '') + '"><input type="radio" name="xt" class="cbx" data-xt="' + c.id + '"' + (on ? ' checked' : '') + (okc ? '' : ' disabled') + '><span class="r-main"><b>' + esc(c.name) + '</b><span>' + esc(X.posL(c.pos)) + (X.isMgrOk(c.id) ? '' : ' — غیرفعال: قابل انتخاب نیست') + (X.isDesc(c.id, p.id) ? ' — زیرمجموعه خود فرد (چرخه)' : '') + '</span></span></label>';
    }).join('') + '</div>';
    var eff = [['subject', 'فرد', esc(p.name) + ' · ' + esc(X.posL(p.pos))], ['cur', 'مدیر فعلی', p.parentUnknown ? '<b>نامعلوم (UNKNOWN)</b>' : cur ? esc(cur.name) : 'ثبت نشده'], ['tgt', 'مدیر مقصد', t ? esc(t.name) : 'انتخاب نشده'], ['pos', 'بافت سمت', 'سمت و سطح ' + esc(p.lvl) + ' تغییر نمی‌کند'], ['rq', 'تاریخ اثر درخواستی', 'پشتیبانی‌شده تأیید نشده؛ فقط «زمان اعمال» ثبت می‌شود'], ['ap', 'زمان اعمال واقعی', 'هنگام اعمال ثبت می‌شود (کنونی: ' + now + ')'], ['ch', 'مسیر بررسی', 'ویرایش مستقیم: منابع انسانی؛ نهایی‌کردن درخواست جدا است (تأیید مرحله‌ای ≠ انتقال)']];
    var accessAfter = '<p class="ind-note">' + ic('key') + ' دسترسی: سمت و نقش بدون تغییر؛ فقط ارث‌بری استثناهای فاکتور/شماره اضافه از والد ممکن است عوض شود (جزئیات در بازرس دسترسی).</p>';
    var resp = o ? checks([['info', fa(o.leads) + ' لید · ' + fa(o.inv) + ' فاکتور · ' + fa(o.cust) + ' مشتری · ' + fa(o.tasks) + ' وظیفه باز', 'مسئولیت عملیاتی فعلی نزد خود فرد و فروش می‌ماند'], ['warn', 'دیده‌شدن زیر مدیر جدید ≠ انتقال پرونده یا استحقاق مالی', 'تحویل کار باز (OPD-05) مشروط و تعریف‌نشده است؛ انتقال خودکار انجام نمی‌شود']]) : checks([['q', 'اثر کار باز برای این نقش ارزیابی نشده', 'نامعلوم است و صفر فرض نمی‌شود']]);
    var unkm = !!p.parentUnknown, unkBtn = unkm && p.unkOp ? '<div class="resp-line">' + btn('btn-soft', 'open-op:' + p.unkOp, 'باز کردن نتیجه عملیات و تطبیق', 'refresh') + '</div>' : '';
    var bl = unkm ? ['no', 'مدیر مؤثر نامعلوم است', 'تغییر پیشین باید ابتدا تطبیق شود؛ نوشتن جدید مسدود است'] : same ? ['no', 'مدیر مقصد با مدیر فعلی یکی است', 'تغییری وجود ندارد'] : cyc ? ['no', 'چرخه: مقصد زیرمجموعه خود فرد است', 'سلسله‌مراتب نامعتبر می‌شود'] : inact ? ['no', 'مدیر مقصد غیرفعال است', 'جانشین خودکار انتخاب نمی‌شود'] : effc ? ['warn', 'تعارض تاریخ اثر', 'بازه جدید با بازه فعلی هم‌پوشانی دارد؛ تاریخ اثر و زمان اعمال جدا ثبت می‌شوند'] : t ? ['ok', 'مقصد فعال و بدون چرخه است', 'بررسی دوباره هنگام ثبت'] : ['info', 'مدیر مقصد را انتخاب کنید', ''];
    var blocked = !t || same || cyc || inact || effc || unkm;
    return top('پیش‌نمایش تغییر مدیر') + head('تغییر مدیر ' + esc(p.name), pill('پیش‌نمایش', 'blue', 'eye') + pill('ویرایش مستقیم', 'slate', 'edit'), '<span class="mono">' + p.id + '</span>') + '<div class="dr-body">' + sec('مدیر مقصد', '', list, 'primary') +
      sec('خلاصه تغییر', '', '<dl class="exc-dl hcp">' + eff.map(function (e) { return '<div><dt>' + e[1] + '</dt><dd>' + e[2] + '</dd></div>'; }).join('') + '</dl>') + sec('بررسی', '', checks([bl]) + unkBtn) + sec('اثر روی دسترسی', '', accessAfter) + sec('اثر روی مسئولیت‌های عملیاتی', '', resp) +
      sec('حفظ تاریخچه', '', checks([['ok', 'بازه مدیر پیشین بسته می‌شود نه حذف', 'تاریخچه حفظ می‌شود'], ['ok', 'مالک اولیه، عامل رویداد و مالک اعتبار گذشته بدون بازنویسی', 'ساختار فعلی تاریخ گذشته را تعیین نمی‌کند'], ['q', 'بسته‌شدن بازه فعلی و ثبت بازه جدید در یک تراکنش اثبات نشده', 'خطای میانه‌راه ممکن است «نامعلوم» بماند (HR-G06)']])) + '</div>' + foot(btn('btn-primary btn-lg', 'xfer-review:' + p.id, 'ادامه: تأیید با بررسی اثر', 'send', blocked), null, 'پیش‌نمایش ثبت نهایی نیست');
  };
  X.isDesc = function (id, anc) { var p = X.s(id), n = 0; while (p && p.parent && n++ < 20) { if (p.parent === anc) return true; p = X.s(p.parent); } return false; };
  X.xferSpec = function (p, t) {
    var cur = X.mgrText(p), o = p.open;
    return { kind: 'xfer', id: p.id, zone: 'cond', title: 'تأیید تغییر مدیر', subject: p.name + ' · ' + p.id, pre: '', changes: ['مدیر: ' + cur + ' ← ' + t.name, 'بازه جدید با تاریخ اثر برابر زمان اعمال'], unchanged: ['سمت، سطح و نقش عملیاتی', 'مالک اولیه، عامل رویداد و مالک اعتبار گذشته', 'مسئول فعلی پرونده‌ها/فاکتورها (تحویل کار جدا و تعریف‌نشده است)'], affected: [p.name, 'مدیر پیشین: ' + cur, 'مدیر جدید: ' + t.name, o ? fa(o.leads + o.inv + o.cust + o.tasks) + ' مورد کار باز (بدون انتقال)' : 'اثر کار باز ارزیابی نشده'], effective: 'زمان اعمال (' + now + '). تاریخ اثر آینده‌نگر پشتیبانی‌شده تأیید نشده.', commit: 'ثبت تغییر مدیر', p: p.id, t: t.id };
  };
  X.commitXfer = function (sp, forceV) {
    if (X.s(sp.p).parentUnknown) {   // a prior hierarchy write is unresolved: only its reconcile/retry path may settle it
      var q = X.s(sp.p), xi = [['بستن بازه فعلی مدیر', 'rejected', 'نتیجه تغییر پیشین نامعلوم است؛ ابتدا تطبیق'], ['ثبت بازه جدید با مدیر مقصد', 'rejected', 'نتیجه تغییر پیشین نامعلوم است؛ ابتدا تطبیق']];
      var xop = X.newOp({ kind: 'xfer', blocked: true, retry: false, title: 'تغییر مدیر ' + q.name + ' (مسدود)', requested: 2, items: xi, note: 'مسدود و بدون تغییر: مدیر مؤثر نامعلوم است. تغییر پیشین را از نتیجه عملیات تطبیق دهید.' });
      xop.audit = pushAudit({ act: 'تغییر مدیر (مسدود)', subj: q.id, req: sp.r || '—', before: 'مدیر: نامعلوم', after: 'بدون تغییر — تطبیق لازم', reason: sp.reasonText || '', chain: sp.r ? 'درخواست‌دهنده ← تأیید مرحله‌ای ← منابع انسانی' : 'مسیر مستقیم', eff: '—', applied: '—', res: 'blocked', corr: 'hr-' + xop.id.slice(3) });
      return xop;
    }
    var p = X.s(sp.p), t = X.s(sp.t), v = forceV || st.flow, rq = sp.r || null, fromName = p.parentUnknown ? 'نامعلوم' : p.parent ? X.name(p.parent) : '—';
    var items = [['بستن بازه فعلی مدیر', 'ok', ''], ['ثبت بازه جدید با مدیر مقصد', v === 'xunknown' ? 'unknown' : 'ok', v === 'xunknown' ? 'پاسخ ثبت نرسید؛ ممکن است ثبت شده باشد' : ''], ['به‌روزرسانی ارث‌بری دسترسی', v === 'xpartial' ? 'failed' : 'ok', v === 'xpartial' ? 'اعمال دسترسی ناموفق؛ قابل تکرار' : ''], ['حفظ تاریخچه و اعتبار گذشته (بدون بازنویسی)', 'ok', '']];
    if (v === 'xunknown') items[1][0] = 'ثبت بازه جدید با مدیر مقصد (پس از بستن بازه فعلی)';
    var op = X.newOp({ kind: 'xfer', title: (rq ? 'اعمال ' + rq + ': ' : '') + 'تغییر مدیر ' + p.name + ' به ' + t.name, requested: items.length, items: items, p: p.id, t: t.id, reqId: rq, fromName: fromName, note: 'تغییر ساختار فقط دیده‌شدن/مسیر را عوض می‌کند؛ مسئول فعلی کارها و مالک اعتبار تغییر نکرد.' });
    op.audit = pushAudit({ act: rq ? 'اعمال درخواست انتقال' : 'تغییر مدیر', subj: p.id, req: rq || '—', before: 'مدیر: ' + fromName, after: 'نامعلوم', reason: sp.reasonText || '', chain: rq ? 'درخواست‌دهنده ← تأیید مرحله‌ای ← منابع انسانی' : 'مسیر مستقیم', eff: now, applied: '—', res: 'unknown', corr: 'hr-' + op.id.slice(3) });
    X.settleOp(op);
    return op;
  };

  /* ---------- Termination impact review ---------- */
  D.term = function (id) {
    var p = X.s(id), o = p.open || { leads: 0, inv: 0, cust: 0, tasks: 0, verified: false }, total = o.leads + o.inv + o.cust + o.tasks, done = X.empOf(p) === 'terminated';
    var open = !o.verified ? 'unknown' : total > 0 ? 'blocked' : 'clear';
    var rows = '<table class="tbl no-cursor" aria-label="کار باز"><caption class="sr">کار باز و محافظت‌شده نزد فرد</caption><thead><tr><th>نوع</th><th class="n">تعداد</th><th>وضعیت مسئولیت پس از پایان همکاری</th></tr></thead><tbody>' + [['لید و پرونده فروش', o.leads], ['فاکتور و پیش‌فاکتور', o.inv], ['مشتری مسئول', o.cust], ['وظیفه آینده', o.tasks]].map(function (r) { return '<tr><td>' + r[0] + '</td><td class="n">' + (o.verified ? fa(r[1]) : na('تأیید نشده')) + '</td><td>' + (!o.verified ? pill('نامعلوم', 'amber', 'question') : r[1] ? pill('مسئول بعدی نامشخص (OPD-05)', 'red', 'lock') : pill('کاری باقی نمانده', 'teal', 'checkCircle')) + '</td></tr>'; }).join('') + '</tbody></table>';
    var unres = open === 'clear' ? [] : ['گیرنده/جانشین تحویل کار باز تعیین نشده (OPD-05)', 'مسئول فعلی هر لید/فاکتور پس از پایان همکاری نامشخص (UNKNOWN)', 'مسئول مشتری و پیگیری فاکتور تعیین نشده', 'زمان لغو نقش و نشست تعریف نشده'];
    var statePill = done ? pill('پایان همکاری ثبت شده؛ تحویل ناتمام', 'amber', 'split') : open === 'clear' ? pill('کار باز ندارد (تأییدشده)', 'teal', 'checkCircle') : open === 'unknown' ? pill('اثر کار باز نامعلوم', 'amber', 'question') : pill('تحویل ناتمام · مسدود/مشروط', 'red', 'lock');
    var cmp = ' <div class="own-grid"><div class="own"><span class="own-l">' + ic('wallet') + 'اعتبار کسب‌شده گذشته</span><b>حفظ می‌شود؛ بدون بازمحاسبه یا بازپس‌گیری</b></div><div class="own"><span class="own-l">' + ic('history') + 'استحقاق آینده</span><b>تصمیم جدا؛ این اقدام آن را تعیین نمی‌کند</b></div></div>';
    return top('بررسی اثر پایان همکاری') + head('پایان همکاری ' + esc(p.name), statePill, '<span class="mono">' + p.id + '</span><span>پایان همکاری ≠ غیرفعال‌کردن کاربر</span>') + '<div class="dr-body">' +
      sec('وضعیت اشتغال و دسترسی (دو چیز جدا)', '', '<div class="rva"><div class="rv-c"><span class="own-l">' + ic('user') + 'وضعیت اشتغال</span><b>' + esc(X.EMP[X.empOf(p)].label) + ' ← پایان همکاری</b></div><div class="rv-vs" aria-hidden="true">' + ic('swap') + '</div><div class="rv-c rv-act"><span class="own-l">' + ic('key') + 'وضعیت دسترسی</span><b>نقش/نشست: لغو نمی‌شود · زمان لغو تعریف نشده</b></div></div><p class="ind-note">' + ic('alert') + ' شاخه فعلی کد فقط پرچم فعال‌بودن و تاریخ پایان را ثبت می‌کند؛ لغو نقش، نشست و انتقال کامل کار از آن نتیجه نمی‌شود. زمان لغو دسترسی باید صریح تعیین شود.</p>', 'primary') +
      sec('مدیر و مسئولیت فعلی', '', '<dl class="exc-dl hcp"><div><dt>مدیر فعلی</dt><dd>' + (p.parentUnknown ? '<b>نامعلوم (UNKNOWN)</b>' : p.parent ? esc(X.name(p.parent)) + (X.isMgrOk(p.parent) ? '' : ' (غیرفعال)') : 'ثبت نشده') + '</dd></div><div><dt>مسئول فعلی کارها</dt><dd>خود فرد (تا تحویل)</dd></div><div><dt>مسئول بعدی / مسئولیت آینده</dt><dd><b>نامشخص (UNKNOWN)</b> — تا تعیین سیاست OPD-05</dd></div></dl>') + sec('کار باز و محافظت‌شده', open === 'blocked' ? pill('مسدودکننده', 'red', 'lock') : '', rows) +
      sec('مسئولیت فاکتور و مشتری', '', checks([['info', 'فاکتورها و ارتباط‌های مالی حفظ می‌شود', 'پایان همکاری حذف مالی نیست'], ['info', 'مسئول فروش/مالی فعلی دست‌نخورده', 'تغییر خودکار نداریم']])) + sec('مرز جبران خدمات', '', cmp) +
      sec('موارد تحویل حل‌نشده', unres.length ? pill(fa(unres.length) + ' مورد', 'amber', 'alert') : '', unres.length ? checks(unres.map(function (u) { return ['no', u, '']; })) + '<p class="ind-note">' + ic('lock') + ' بازنشانی یا انتقال خودکار مسئولیت پیشنهاد نمی‌شود؛ مسئول بعدی باید نام‌برده و مجاز باشد.</p>' : checks([['ok', 'کار باز نیست و مسئولیت بی‌صاحب نمی‌ماند', 'تأییدشده در این پیش‌نمایش']]) + '<p class="ind-note">' + ic('info') + ' حتی بدون کار باز، این اقدام فقط وضعیت اشتغال را عوض می‌کند و «آف‌بوردینگ کامل» نیست.</p>') + '</div>' +
      foot(done ? btn('btn-lg', 'x', 'پایان همکاری قبلاً ثبت شده', 'check', true) : open === 'clear' ? btn('btn-primary btn-lg', 'term-review:' + p.id, 'ادامه: تأیید با بررسی اثر', 'send') : btn('btn-lg', 'x', 'اعمال غیرفعال — تحویل ناتمام (OPD-05)', 'lock', true), null, 'هیچ بازتخصیص خودکار وجود ندارد');
  };
  X.termSpec = function (p) {
    return { kind: 'term', id: p.id, zone: 'restr', title: 'تأیید پایان همکاری', subject: p.name + ' · ' + p.id, pre: '', changes: ['وضعیت اشتغال: فعال ← پایان همکاری', 'پروفایل غیرفعال و تاریخ پایان ثبت می‌شود'], unchanged: ['نقش و نشست حساب (لغو نمی‌شود؛ زمان‌بندی تعریف نشده)', 'اعتبار کسب‌شده گذشته و فاکتورها', 'مالک اولیه و تاریخچه'], affected: [p.name, 'مدیر فعلی: ' + (p.parentUnknown ? 'نامعلوم' : p.parent ? X.name(p.parent) : '—'), 'کار باز: ندارد (تأییدشده)'], effective: 'زمان اعمال (' + now + '). تاریخ اثر درخواستی جدا نگه داشته می‌شود.', ack: 'می‌دانم این فقط وضعیت اشتغال را عوض می‌کند و لغو دسترسی یا تحویل کامل نیست.', commit: 'ثبت پایان همکاری', p: p.id };
  };
  // Fail-closed handover guard: evidence must exist, be verified, and protected/open work must be zero. Anything else (including a repeat apply) changes nothing.
  X.termGuard = function (p) {
    var o = p.open;
    if (X.empOf(p) === 'terminated') return { ok: false, why: 'پایان همکاری قبلاً ثبت شده؛ ثبت دوباره انجام نمی‌شود' };
    if (!o) return { ok: false, why: 'شواهد کار باز/تحویل وجود ندارد؛ مسدود (OPD-05)' };
    if (!o.verified) return { ok: false, why: 'اسنپ‌شات کار باز تأیید نشده؛ مسدود (OPD-05)' };
    var keys = ['leads', 'inv', 'cust', 'tasks'];
    if (!keys.every(function (k) { return typeof o[k] === 'number' && isFinite(o[k]) && o[k] >= 0; })) return { ok: false, why: 'شمارش کار باز ناقص یا نامعتبر است (همه ابعاد لازم است)؛ صفر فرض نمی‌شود (OPD-05)' };
    var total = o.leads + o.inv + o.cust + o.tasks;
    if (total > 0) return { ok: false, why: fa(total) + ' مورد کار باز/محافظت‌شده؛ گیرنده تحویل تعریف نشده (OPD-05)' };
    return { ok: true, why: '' };
  };
  X.commitTerm = function (sp) {
    var g = X.termGuard(X.s(sp.p));
    if (!g.ok) {
      var q = X.s(sp.p), bi = [['وضعیت اشتغال ← پایان همکاری', 'rejected', g.why], ['غیرفعال‌سازی پروفایل و تاریخ پایان', 'rejected', g.why], ['لغو نقش و نشست حساب', 'skipped', 'این اقدام لغو نمی‌کند'], ['تحویل کار باز', 'skipped', 'گیرنده‌ای تعریف نشده']];
      var bop = X.newOp({ kind: 'term', blocked: true, retry: false, title: 'پایان همکاری ' + q.name + ' (مسدود)', requested: bi.length, items: bi, note: 'مسدود و بدون تغییر: ' + g.why + '. هیچ وضعیتی نوشته نشد.' });
      pushAudit({ act: 'پایان همکاری (مسدود)', subj: q.id, req: sp.r || '—', before: 'وضعیت: ' + X.EMP[X.empOf(q)].label, after: 'بدون تغییر — ' + g.why, reason: sp.reasonText || '', chain: sp.r ? 'درخواست‌دهنده ← تأیید مرحله‌ای ← منابع انسانی' : 'مسیر مستقیم', eff: '—', applied: '—', res: 'blocked', corr: 'hr-' + bop.id.slice(3) });
      return bop;
    }
    var p = X.s(sp.p), items = [['وضعیت اشتغال ← پایان همکاری', 'ok', ''], ['غیرفعال‌سازی پروفایل و تاریخ پایان', 'ok', ''], ['لغو نقش و نشست حساب', 'skipped', 'این اقدام لغو نمی‌کند؛ زمان لغو دسترسی تعریف نشده'], ['تحویل کار باز', 'skipped', 'کار بازی نبود (تأییدشده)']];
    var op = X.newOp({ kind: 'term', title: 'پایان همکاری ' + p.name, requested: items.length, items: items, note: 'فقط وضعیت اشتغال تغییر کرد؛ آف‌بوردینگ کامل ادعا نمی‌شود و دسترسی به‌جای دیگر باید صریحاً لغو شود.' });
    st.staffEdit[p.id] = { emp: 'terminated' };
    pushAudit({ act: sp.r ? 'اعمال درخواست پایان همکاری' : 'پایان همکاری', subj: p.id, req: sp.r || '—', before: 'وضعیت: فعال', after: 'وضعیت: پایان همکاری (دسترسی لغو نشد)', reason: sp.reasonText || '', chain: sp.r ? 'درخواست‌دهنده ← تأیید مرحله‌ای ← منابع انسانی' : 'مسیر مستقیم', eff: now, applied: now, res: 'partial', corr: 'hr-' + op.id.slice(3) });
    return op;
  };

  /* ---------- Request review ---------- */
  D.req = function (id) {
    var r = X.req(id), s = X.reqSt(r), p = X.s(r.subj), keep = keepOf(), rec = st.recon && st.recon[id];
    var stepIdx = { pending_review: 1, approved_step: 2, pending_hr: 3, approved: 5, rejected: 2, failed: 4 }[s];
    var lifecycle = X.steps(['درخواست', 'بررسی', 'تأیید/رد مرحله‌ای', 'منابع انسانی', 'اعمال', 'نتیجه'], stepIdx);
    var reqTxt = r.type === 'transfer' ? 'مدیر: ' + esc(X.name(r.from)) + ' ← ' + esc(X.name(r.to)) : 'پایان همکاری · اثر درخواستی ' + esc(r.eff || 'ثبت نشده');
    var act;
    if (s === 'approved') act = r.type === 'transfer' ? 'مدیر: ' + esc(r.actual.before) + ' ← ' + esc(r.actual.after) + ' · اعمال ' + esc(r.appliedAt) : 'اعمال شد · ' + esc(r.appliedAt);
    else if (s === 'failed') act = rec ? 'پس از تطبیق: بازه فعلی بسته شده و بازه جدید ثبت نشده — مدیر مؤثر <b>نامعلوم</b>' : '<b>نامعلوم</b> — اثر واقعی خوانده نشده';
    else if (s === 'rejected') act = 'هیچ تغییری اعمال نشد';
    else act = '<b>هنوز اعمال نشده</b> — ' + (s === 'approved_step' ? 'تأیید مرحله‌ای تغییر نیست' : 'در انتظار مرحله بعد');
    var notes = [];
    if (r.changed) notes.push(['warn', 'درخواست پیش از بررسی تغییر کرده', r.changed + '؛ پیش از تأیید دوباره بخوانید']);
    if (r.targetInactive) notes.push(['no', 'مدیر مقصد غیرفعال است', 'اعمال بدون بررسی تازه سیاست مجاز نیست']);
    if (r.oos) notes.push(['no', 'درخواست خارج از محدوده است', 'بررسی با رتبه بالاتر انجام نمی‌شود']);
    var o = p.open, tg = r.type === 'terminate' ? X.termGuard(p) : { ok: true }, termBlock = !tg.ok;
    if (termBlock) notes.push(['no', 'تحویل کار باز تأیید نشده است (OPD-05)', tg.why + ' — اعمال نهایی مسدود (fail-closed)']);
    if (s === 'pending_review') notes.push(['info', 'منابع انسانی در کد می‌تواند مرحله منتظر را نهایی کند', 'عبور از زنجیره بدون سیاست مصوب است (HR-G11) و اینجا عرضه نمی‌شود']);
    var blockedStep = r.changed || r.oos || r.targetInactive;
    var acts = '';
    if (s === 'pending_review' || s === 'approved_step') acts = btn('btn-primary btn-lg', 'req-step:' + r.id, 'ثبت تأیید مرحله‌ای', 'check', blockedStep) + btn('btn-lg btn-danger', 'req-reject:' + r.id, 'رد با دلیل', 'xCircle');
    else if (s === 'pending_hr') acts = btn('btn-primary btn-lg', 'req-apply:' + r.id, 'اعمال نهایی…', 'send', r.targetInactive || r.oos || termBlock || p.parentUnknown) + btn('btn-lg btn-danger', 'req-reject:' + r.id, 'رد با دلیل', 'xCircle');
    else if (s === 'failed') acts = btn('btn-primary btn-lg', 'req-reconcile:' + r.id, 'خواندن اثر واقعی (تطبیق)', 'refresh', rec) + btn('btn-lg', 'x', 'اعمال دوباره — سیاست بازیابی تعریف نشده', 'lock', true);
    var rejForm = (s === 'pending_review' || s === 'approved_step' || s === 'pending_hr') ? '<section class="sec"><label class="lbl" for="rj-n">دلیل رد (فقط برای «رد با دلیل»)</label><textarea class="input" id="rj-n" rows="2" data-rejnote' + (keep.err ? ' aria-invalid="true"' : '') + '>' + esc(keep.note || '') + '</textarea>' + (keep.err ? '<p class="field-err" role="alert">برای رد، دلیل لازم است.</p>' : '') + '</section>' : '';
    return top('بررسی درخواست') + head(esc(p.name), X.reqPill(s), '<span class="mono">' + r.id + '</span><span>' + (r.type === 'transfer' ? 'انتقال' : 'پایان همکاری') + '</span><span>درخواست‌دهنده: ' + esc(r.by) + '</span>') + '<div class="dr-body"><section class="sec">' + lifecycle + '</section>' +
      sec('درخواستی در برابر واقعاً اعمال‌شده', '', X.rva(reqTxt, act, s === 'failed' ? 'علت: ' + esc(r.fail) + '. موفقیت شاخه کد = تغییر تأییدشده نیست.' : s === 'approved_step' || s === 'pending_hr' ? 'تأیید مرحله‌ای، تغییر اعمال‌شده نیست.' : '' ), 'primary') +
      (notes.length ? sec('بررسی تازه', '', checks(notes)) : '') + sec('زنجیره بررسی', '', X.chain(r.chain) + '<dl class="exc-dl hcp"><div><dt>بررسی‌کننده فعلی</dt><dd>' + esc(r.cur) + '</dd></div><div><dt>بررسی‌کننده بعدی</dt><dd>' + esc(r.next) + '</dd></div><div><dt>تاریخ اثر درخواستی</dt><dd>' + esc(r.eff || 'ثبت نشده — فقط زمان اعمال') + '</dd></div><div><dt>زمان اعمال</dt><dd>' + esc(r.appliedAt || 'اعمال نشده') + '</dd></div><div><dt>دلیل</dt><dd>' + esc(r.reason) + (r.rejNote ? ' · رد: ' + esc(r.rejNote) : '') + '</dd></div></dl>') +
      X.details('تاریخچه', 3, tl([['درخواست ثبت شد', r.chain[0][2]], [X.REQ[s].label, 'وضعیت فعلی · شناسه ' + X.REQ[s].id], ['اثر واقعی', s === 'approved' ? r.appliedAt : 'هنوز ثبت نشده']])) + rejForm + '</div>' + foot(acts, null, 'تأیید مرحله‌ای ≠ اعمال نهایی');
  };
  X.reqStep = function (r) { st.reqState[r.id] = r.next === 'منابع انسانی' ? 'pending_hr' : 'approved_step'; r.chain.forEach(function (c) { if (c[1] === 'cur') { c[1] = 'done'; c[2] = c[2] + ' · تأیید مرحله‌ای ' + now; } }); r.cur = r.next === 'منابع انسانی' ? 'منابع انسانی' : r.next; r.next = r.next === 'منابع انسانی' ? '—' : 'منابع انسانی'; if (st.reqState[r.id] === 'pending_hr') { r.chain.forEach(function (c) { if (c[0] === 'منابع انسانی') c[1] = 'cur'; }); } };
  X.reqReject = function (r, note) { st.reqState[r.id] = 'rejected'; r.rejNote = note; r.cur = '—'; r.next = '—'; r.chain.forEach(function (c) { if (c[1] === 'cur') { c[1] = 'rejected'; c[2] = c[2] + ' · رد با دلیل'; } }); };
  X.reqApplySpec = function (r) {
    var p = X.s(r.subj);
    if (r.type === 'transfer') { var sp = X.xferSpec(p, X.s(r.to)); sp.kind = 'reqapply'; sp.id = r.id; sp.title = 'اعمال نهایی درخواست ' + r.id; sp.r = r.id; sp.pre = '<section class="sec"><div class="note warn">' + ic('alert') + '<span>تأیید مرحله‌ای ثبت شده بود؛ این گام جدا «اعمال نهایی» است و پیش از ثبت، درخواست، فرد و مدیر مقصد دوباره خوانده می‌شوند.</span></div></section>'; return sp; }
    var tsp = X.termSpec(p); tsp.kind = 'reqapply'; tsp.r = r.id; tsp.id = r.id; tsp.title = 'اعمال نهایی درخواست ' + r.id; return tsp;
  };
  X.commitReqApply = function (sp) {
    var r = X.req(sp.r), p = X.s(r.subj), v = st.flow, op;
    if (r.type === 'transfer') {
      sp.p = p.id; sp.t = r.to; op = X.commitXfer(sp, v === 'applyfail' ? 'xunknown' : null);
      if (op.blocked) { r.blockNote = op.note; return op; }
      if (!op.done) { st.reqState[r.id] = 'failed'; r.fail = 'ثبت بازه جدید پس از بستن بازه فعلی ناموفق/نامعلوم بود'; r.actual = { before: op.fromName, after: 'نامعلوم', ok: false }; op.note = 'اثر واقعی نامعلوم است: ابتدا بخوانید؛ اعمال دوباره کور ممنوع است.'; }
      return op;
    }
    sp.p = p.id; op = X.commitTerm(sp);
    if (op.blocked) { r.blockNote = op.note; return op; }
    st.reqState[r.id] = 'approved'; r.appliedAt = now; r.actual = { before: 'فعال', after: 'پایان همکاری', ok: true }; X.closeReqChain(r); return op;
  };

  /* ---------- Access-State Inspector ---------- */
  D.access = function (id) {
    var p = X.s(id), keep = keepOf(), A = M.access[p.id] || M.access['default'], after = keep.after || null;
    var pl = '<div class="recips-inline" role="radiogroup" aria-label="سمت پیشنهادی برای مقایسه">' + M.positions.filter(function (x) { return x.id !== 'hr'; }).map(function (x) { var on = (after || p.pos) === x.id; return '<label class="recip' + (on ? ' on' : '') + '"><input type="radio" name="ap" class="cbx" data-after="' + x.id + '"' + (on ? ' checked' : '') + '><span class="r-main"><b>' + esc(x.label) + '</b><span>' + (x.id === p.pos ? 'سمت فعلی' : 'مقایسه') + '</span></span></label>'; }).join('') + '</div>';
    var delta = after && after !== p.pos;
    return top('بازرس دسترسی') + head(esc(p.name), X.accPill(p.role === 'nouser' ? 'nouser' : p.role), '<span class="mono">' + p.id + '</span><span>' + esc(X.posL(p.pos)) + '</span>') + '<div class="dr-body">' + sec('چهار چیز جدا', '', '<div class="own-grid"><div class="own"><span class="own-l">' + ic('briefcase') + 'سمت کاری' + hint('سمت کاری فرد در ساختار فروش') + '</span><b>' + esc(X.posL(p.pos)) + '</b></div><div class="own"><span class="own-l">' + ic('key') + 'نقش / قابلیت وردپرس' + hint('نقش فنی؛ سمت نیست') + '</span><b>' + esc(A.actualRole || X.pos(p.pos).role) + '</b></div><div class="own"><span class="own-l">' + ic('check') + 'مجوز محصولی' + hint('نتیجه نگاشت سمت، نقش و استثناها') + '</span><b>در ماتریس زیر</b></div><div class="own"><span class="own-l">' + ic('lock') + 'اعتبارنامه' + hint('گذرواژه و ورود؛ در بخش محدود') + '</span><b>' + (p.user ? 'حساب پیوند‌خورده · بدون نمایش رمز' : 'حساب پیوند نشده') + '</b></div></div>', 'primary') +
      (A.partial ? '<section class="sec"><div class="note warn">' + ic('split') + '<span><b>اعمال دسترسی ناقص:</b> ' + esc(A.partial) + '</span></div></section>' : '') +
      sec('ماتریس وضعیت دسترسی', X.basis('snap'), X.asm(p, delta ? after : null)) + sec('مقایسه با سمت پیشنهادی', '', pl + '<p class="ind-note">' + ic('info') + ' مقایسه فقط می‌خواند. استثناهای مستقیم و نقش‌های نامرتبط با تغییر سمت بازنشانی نمی‌شوند؛ ویرایش قابلیت وردپرس عرضه نمی‌شود.</p>') + '</div>' +
      foot(btn('btn-primary btn-lg', 'access-review:' + p.id, 'ادامه: تأیید تغییر سمت با بررسی اثر', 'send', !delta), null, 'تغییر حساس · نیازمند دلیل');
  };
  X.accessSpec = function (p, pos) {
    var A = M.access[p.id] || M.access['default'], aft = X.accessAfter(p, pos), ch = [];
    M.perms.forEach(function (r) { var a = A.rows[r.k][1]; if (aft[r.k] !== a) ch.push(r.label + ': ' + (a === 'allow' ? 'مجاز' : 'غیرمجاز') + ' ← ' + (aft[r.k] === 'allow' ? 'مجاز' : 'غیرمجاز') + (r.sens ? ' (حساس)' : '')); });
    return { kind: 'access', id: p.id, zone: 'restr', title: 'تأیید تغییر سمت و دسترسی', subject: p.name + ' · ' + p.id, pre: '', changes: ['سمت: ' + X.posL(p.pos) + ' ← ' + X.posL(pos)].concat(ch), unchanged: ['استثناهای مستقیم فاکتور/شماره اضافه', 'نقش‌ها و قابلیت‌های نامرتبط', 'اعتبارنامه (گذرواژه) و سابقه', 'مدیر فعلی و تاریخچه'], affected: [p.name, 'نگاشت سمت ← نقش عملیاتی', fa(ch.length) + ' مجوز محصولی'], effective: 'زمان اعمال (' + now + '). فعال‌بودن پرچم، اثبات ورود قابل‌استفاده نیست.', commit: 'ثبت تغییر سمت', p: p.id, pos: pos, n: ch.length, keys: M.perms.filter(function (r) { return A.rows[r.k][1] !== aft[r.k]; }).map(function (r) { return r.k; }) };
  };
  X.commitAccess = function (sp) {
    var p = X.s(sp.p), v = st.flow, n = Math.max(sp.n, 1), items = [['به‌روزرسانی سمت', 'ok', '']];
    for (var i = 0; i < n; i++) items.push(['مجوز ' + fa(i + 1) + ' از ' + fa(n), v === 'accpartial' && i === n - 1 ? 'failed' : 'ok', v === 'accpartial' && i === n - 1 ? 'اعمال ناموفق؛ وضعیت مؤثر با مورد انتظار فرق دارد' : '']);
    var op = X.newOp({ kind: 'access', title: 'تغییر سمت ' + p.name, requested: items.length, items: items, p: p.id, keys: sp.keys, expected: X.accessAfter(p, sp.pos), note: 'نقش‌های نامرتبط و استثناهای مستقیم بازنشانی نشدند.' });
    p.pos = sp.pos; X.settleOp(op);
    op.audit = pushAudit({ act: 'تغییر سمت', subj: p.id, req: '—', before: 'سمت پیشین', after: X.posL(sp.pos) + (p.role === 'partial' ? ' (دسترسی ناقص)' : ''), reason: sp.reasonText || '', chain: 'مسیر مستقیم', eff: now, applied: now, res: X.opState(op), corr: 'hr-' + op.id.slice(3) });
    return op;
  };
  D.pos = function (id) {
    var p = X.pos(id), n = M.staff.filter(function (s) { return s.pos === id; }).length;
    return top('اثر غیرفعال‌سازی سمت') + head(esc(p.label), pill('فقط بررسی اثر', 'blue', 'eye'), '<span>غیرفعال‌سازی ≠ ویرایش برچسب</span>') + '<div class="dr-body">' + sec('ارجاع‌ها', '', checks([[n ? 'no' : 'ok', fa(n) + ' نیروی فعلی با این سمت', n ? 'غیرفعال‌سازی تا جابه‌جایی آن‌ها مجاز نیست' : 'ارجاعی نیست'], ['info', 'نگاشت سمت ← نقش عملیاتی: ' + p.role, 'تغییر نگاشت دسترسی واقعی را عوض می‌کند'], ['info', 'تاریخچه و ممیزی حفظ می‌شود', 'حذف مخرب پیشنهاد نمی‌شود']])) + '</div>' + foot(btn('btn-lg', 'x', 'غیرفعال‌سازی — ' + (n ? 'مسدود توسط ارجاع' : 'نیازمند اعتبارسنجی'), 'lock', true), null, '');
  };

  /* ---------- Compensation ---------- */
  D.comp = function (id) {
    var p = X.s(id), c = M.comp[id], keep = keepOf(), denied = st.compAuth === 'denied';
    if (denied) return top('جبران خدمات') + head(esc(p.name), pill('غیرمجاز', 'slate', 'lock'), '') + '<div class="dr-body">' + h.stateBlock('locked', 'دسترسی به جبران خدمات فعال نیست', 'مقدار نمایش داده نمی‌شود؛ این به معنی صفر نیست.', '') + '</div>' + foot('', null, '');
    var per = c.periods.map(function (e, i) {
      var cur = i === 0 && !e.to;
      return '<li class="ei' + (cur ? ' ei-cur' : '') + (e.gap ? ' ei-inc' : '') + '"><span class="ei-dot" aria-hidden="true"></span><div class="ei-b"><div class="ei-h"><b>' + (cur ? 'فعلی' : 'تاریخی') + '</b>' + (cur ? pill('دوره جاری', 'teal', 'clock') : '') + (e.gap ? pill('ناقص', 'amber', 'layers') : '') + '</div><dl class="ei-dl"><div><dt>تاریخ اثر</dt><dd>' + esc(e.from) + ' — ' + (e.to ? esc(e.to) : 'ادامه دارد') + '</dd></div><div><dt>پایه</dt><dd>' + esc(e.base) + ' ' + c.cur + '</dd></div><div><dt>قبل ← بعد</dt><dd>' + esc(e.before) + ' ← ' + esc(e.base) + '</dd></div><div><dt>دلیل</dt><dd>' + esc(e.reason) + '</dd></div><div><dt>زمان اعمال</dt><dd>' + esc(e.applied || 'ثبت نشده') + '</dd></div><div><dt>عامل</dt><dd>' + esc(e.actor) + '</dd></div></dl></div></li>';
    }).join('');
    var tv = X.compTemporal(c, keep.from), valid = keep.base && ok3(keep.reason) && tv.ok, p0 = c.periods[0];
    var form = '<div class="form-grid"><div><label class="lbl" for="cp-b">پایه جدید (' + (c.cur || 'IRT') + ')</label><input class="input" id="cp-b" inputmode="numeric" data-cpbase value="' + esc(keep.base || '') + '" placeholder="مثلاً ۳۴٬۰۰۰٬۰۰۰"></div><div><label class="lbl" for="cp-f">تاریخ اثر</label><input class="input" id="cp-f" data-cpfrom value="' + esc(keep.from || '') + '" placeholder="۱۴۰۵/۰۸/۰۱"' + (!tv.ok && !tv.empty ? ' aria-invalid="true" aria-describedby="cp-fe"' : '') + '>' + (!tv.ok && !tv.empty ? '<p class="field-err" id="cp-fe" role="alert">' + esc(tv.msg) + '</p>' : '') + '</div><div><label class="lbl" for="cp-r">دلیل</label><input class="input" id="cp-r" data-cpreason value="' + esc(keep.reason || '') + '"></div></div><p class="ind-note">' + ic('alert') + ' بازه قبلی پیش از ثبت بازه جدید بسته می‌شود و تراکنش یکپارچه تأیید نشده؛ قفل پس از حقوق نیز تأییدنشده است. اعتبار کسب‌شده گذشته دوباره محاسبه نمی‌شود.</p>';
    return top('جبران خدمات') + head(esc(p.name), pill('حساس', 'amber', 'alert') + (c.com ? pill('واجد کمیسیون: ' + c.rule, 'teal', 'checkCircle') : pill('بدون کمیسیون', 'slate', 'ban')), '<span class="mono">' + p.id + '</span><span>جدا از کیف پول و مالی</span>') + '<div class="dr-body">' +
      sec('وضعیت فعلی', X.basis('hist'), c.base ? '<div class="facets"><div class="facet"><span class="muted">پایه</span><b>' + esc(c.base) + ' ' + c.cur + '</b></div><div class="facet"><span class="muted">کمیسیون (ارجاع، نه موتور)</span><b>' + (c.com ? esc(c.rule) + ' · ' + esc(c.mode) : 'ندارد') + '</b></div><div class="facet"><span class="muted">واحد</span><b>' + c.cur + '</b></div></div>' : h.stateBlock('empty', 'دوره‌ای ثبت نشده', 'جبران خدمات تعریف نشده؛ صفر نیست.', ''), 'primary') +
      (c.unresolved ? '<section class="sec"><div class="note warn">' + ic('question') + '<span><b>نتیجه ثبت دوره پیشین نامعلوم است (' + esc(c.unresolved) + ').</b> دوره قبلی بسته شد؛ تا تطبیق، هر ثبت جدید مسدود است.</span>' + btn('btn-sm', 'open-op:' + c.unresolved, 'باز کردن نتیجه عملیات و تطبیق', 'refresh') + '</div></section>' : '') +
      (c.overlap ? '<section class="sec"><div class="note warn">' + ic('alert') + '<span><b>هشدار یکپارچگی:</b> ' + esc(c.overlap) + '</span></div></section>' : '') +
      sec('تاریخچه دوره‌ها', '', '<ol class="eit" aria-label="دوره‌های جبران خدمات">' + per + '</ol><p class="ind-note">' + ic('lock') + ' ویرایش فعلی HR اعتبار کسب‌شده، استحقاق گذشته یا مبلغ پرداخت‌شده را دوباره محاسبه نمی‌کند.</p>') + sec('ثبت دوره جدید (حساس)', X.zone('restr'), form) + '</div>' +
      foot(btn('btn-primary btn-lg', 'comp-review:' + p.id, 'ادامه: تأیید با بررسی اثر', 'send', !valid), null, 'قفل پس از حقوق: تأییدنشده');
  };
  X.compSpec = function (p, keep) {
    var c = M.comp[p.id], p0 = c.periods[0];
    return { kind: 'comp', id: p.id, zone: 'restr', title: 'تأیید دوره جبران خدمات', subject: p.name + ' · ' + p.id, pre: '', changes: ['پایه: ' + (p0 ? p0.base : '—') + ' ← ' + keep.base + ' ' + (c.cur || 'IRT'), 'بازه قبلی بسته می‌شود و بازه جدید از ' + keep.from + ' ثبت می‌شود'], unchanged: ['اعتبار کسب‌شده و استحقاق گذشته (بدون بازمحاسبه)', 'کیف پول، پرداخت و موتور کمیسیون', 'ارجاع قاعده کمیسیون'], affected: [p.name, 'دوره فعلی و تاریخچه', 'ممیزی قبل/بعد'], effective: 'تاریخ اثر ' + keep.from + ' · زمان اعمال ' + now + ' (دو چیز جدا).', ack: 'می‌دانم قفل پس از حقوق تأیید نشده و شکست میانه‌راه ممکن است نامعلوم بماند.', commit: 'ثبت دوره جبران خدمات', p: p.id, base: keep.base, from: keep.from };
  };
  X.commitComp = function (sp) {
    var p = X.s(sp.p), c = M.comp[p.id], v = st.flow, unk = v === 'compunknown', tv = X.compTemporal(c, sp.from);
    if (!tv.ok) {   // re-checked at commit: an invalid temporal range never closes or rewrites the current interval
      var bi = [['بستن دوره قبلی', 'rejected', tv.msg || 'تاریخ اثر نامعتبر'], ['ثبت دوره جدید', 'rejected', tv.msg || 'تاریخ اثر نامعتبر'], ['حفظ اعتبار کسب‌شده گذشته (بدون بازمحاسبه)', 'skipped', 'تغییری انجام نشد']];
      var bop = X.newOp({ kind: 'comp', blocked: true, retry: false, title: 'دوره جبران خدمات ' + p.name + ' (مسدود)', requested: 3, items: bi, p: p.id, note: 'مسدود و بدون تغییر: ' + (tv.msg || 'تاریخ اثر نامعتبر') + ' دوره فعلی دست‌نخورده ماند.' });
      pushAudit({ act: 'ثبت دوره جبران (مسدود)', subj: p.id, req: '—', before: 'پایه: ' + (c.periods[0] ? c.periods[0].base : '—'), after: 'بدون تغییر — ' + (tv.msg || 'تاریخ نامعتبر'), reason: sp.reasonText || '', chain: '—', eff: sp.from || '—', applied: '—', res: 'blocked', corr: 'hr-' + bop.id.slice(3) });
      return bop;
    }
    var items = [['بستن دوره قبلی', 'ok', ''], ['ثبت دوره جدید', unk ? 'unknown' : 'ok', unk ? 'پاسخ ثبت نرسید پس از بستن دوره قبلی' : ''], ['حفظ اعتبار کسب‌شده گذشته (بدون بازمحاسبه)', 'ok', '']];
    var op = X.newOp({ kind: 'comp', title: 'دوره جبران خدمات ' + p.name, requested: 3, items: items, p: p.id, base: sp.base, from: sp.from, reasonText: sp.reasonText, note: unk ? 'دوره قبلی بسته شد؛ وضعیت دوره جدید نامعلوم است — پیش از هر تکرار بخوانید.' : 'اعتبار کسب‌شده گذشته دوباره محاسبه نشد.' });
    var old = c.periods[0]; if (old && !old.to) old.to = 'ماقبل ' + sp.from;
    if (unk) c.unresolved = op.id;
    op.audit = pushAudit({ act: 'ثبت دوره جبران', subj: p.id, req: '—', before: 'پایه: ' + (old ? old.base : '—'), after: 'نامعلوم', reason: sp.reasonText || '', chain: '—', eff: sp.from, applied: '—', res: 'unknown', corr: 'hr-' + op.id.slice(3) });
    X.settleOp(op);
    return op;
  };

  /* ---------- Restricted credentials / impersonation ---------- */
  D.credreset = function (id) {
    var p = X.s(id), keep = keepOf(), lead = ['deputy', 'manager', 'senior', 'sup'].indexOf(p.pos) > -1;
    var valid = false;
    return top('بازنشانی گذرواژه (محدود)') + head(esc(p.name), X.zone('restr'), '<span class="mono">WP#' + p.user + '</span>') + '<div class="dr-body">' + sec('صلاحیت هدف', '', checks([lead ? ['no', 'هدف دارای سطح مدیریتی است', 'سیاست هدف مجاز تأیید نشده (HR-G04)؛ ادامه مسدود'] : ['ok', 'حساب داخلی غیرممتاز (نمونه)', 'بررسی دوباره هنگام ثبت'], ['info', 'گذرواژه هرگز نمایش یا ثبت نمی‌شود', 'ممیزی فقط «بازنشانی انجام شد» و مخاطب ماسک‌شده را نگه می‌دارد']]), 'primary') +
      sec('گذرواژه جدید', '', '<div class="form-grid"><div><label class="lbl" for="pw-1">گذرواژه جدید</label><input class="input" id="pw-1" type="password" autocomplete="new-password" dir="ltr" data-pw' + (lead ? ' disabled' : '') + '><p class="field-err" id="pw-1e" role="alert" hidden>حداقل ۸ نویسه لازم است.</p></div><div><label class="lbl" for="pw-2">تکرار گذرواژه</label><input class="input" id="pw-2" type="password" autocomplete="new-password" dir="ltr" data-pw2' + (lead ? ' disabled' : '') + '><p class="field-err" id="pw-2e" role="alert" hidden>تکرار با گذرواژه یکی نیست.</p></div><div><label class="lbl" for="pw-r">دلیل</label><input class="input" id="pw-r" data-pwreason' + (lead ? ' disabled' : '') + '></div></div><label class="confirm"><input type="checkbox" class="cbx" data-sms' + (lead ? ' disabled' : '') + '><span>ارسال اعلان پیامکی (اقدام جدا؛ نتیجه‌اش جدا از بازنشانی گزارش می‌شود)</span></label>') + '</div>' +
      foot(btn('btn-primary btn-lg', 'cred-review:' + p.id, 'ادامه: تأیید با بررسی اثر', 'send', !valid), null, 'مقدار گذرواژه در هیچ‌جا نمایش داده نمی‌شود');
  };
  X.credSpec = function (p, keep) { return { kind: 'cred', id: p.id, zone: 'restr', title: 'تأیید بازنشانی گذرواژه', subject: p.name + ' · WP#' + p.user, pre: '', changes: ['گذرواژه حساب بازنشانی می‌شود', keep.sms ? 'اعلان پیامکی جداگانه ارسال می‌شود' : 'اعلان پیامکی ارسال نمی‌شود'], unchanged: ['نقش، سمت و مجوزهای محصولی', 'سابقه کار و مالکیت', 'گذرواژه در لاگ یا ممیزی ثبت نمی‌شود'], affected: [p.name, 'نشست‌های فعال (لغو خودکار تأیید نشده)', keep.sms ? 'مخاطب ماسک‌شده ' + p.mob : 'بدون پیامک'], effective: 'همان لحظه (' + now + ')', ack: 'می‌دانم نتیجه بازنشانی و پیامک دو نتیجه جدا هستند.', commit: 'ثبت بازنشانی', p: p.id, sms: !!keep.sms }; };
  X.commitCred = function (sp) {
    var p = X.s(sp.p), v = st.flow, smsFail = v === 'credpartial' || !!sp.smsFail;
    var items = [['بازنشانی گذرواژه', 'ok', 'انجام شد؛ مقدار ثبت نشد'], ['اعلان پیامکی', sp.sms ? (smsFail ? 'failed' : 'ok') : 'skipped', sp.sms ? (smsFail ? 'ارسال ناموفق؛ بازنشانی معتبر است' : 'ارسال شد') : 'درخواست نشد']];
    var op = X.newOp({ kind: 'cred', title: 'بازنشانی گذرواژه ' + p.name, requested: 2, items: items, retry: false, note: 'نتیجه بازنشانی و نتیجه پیامک جدا هستند.' });
    pushAudit({ act: 'بازنشانی گذرواژه', subj: p.id, req: '—', before: '—', after: 'بازنشانی انجام شد' + (sp.sms ? (smsFail ? '؛ پیامک ناموفق' : '؛ پیامک ارسال شد') : ''), reason: sp.reasonText || '', chain: '—', eff: now, applied: now, res: smsFail ? 'partial' : 'complete', corr: 'hr-' + op.id.slice(3) });
    return op;
  };
  D.impstart = function (id) {
    var p = X.s(id), keep = keepOf(), lead = ['deputy', 'manager', 'senior', 'sup'].indexOf(p.pos) > -1, valid = !lead && keep.purpose && keep.ack0 && !st.imp;
    var purposes = ['بررسی مشکل پشتیبانی گزارش‌شده', 'تأیید پیکربندی دسترسی پس از تغییر سمت'];
    return top('نمایش به‌جای کاربر (محدود)') + head('نمایش به‌جای ' + esc(p.name), X.zone('restr') + pill('فقط‌خواندنی نیست', 'red', 'alert'), '<span class="mono">WP#' + p.user + '</span>') + '<div class="dr-body">' + sec('مشخصات جلسه', '', '<dl class="exc-dl hcp"><div><dt>عامل</dt><dd>' + esc(M.user.name) + ' (منابع انسانی)</dd></div><div><dt>هدف</dt><dd>' + esc(p.name) + ' · ' + esc(X.posL(p.pos)) + '</dd></div><div><dt>وضعیت شروع</dt><dd>' + (st.imp ? '<b>جلسه دیگری فعال است — جلسه تو در تو ممنوع</b>' : 'جلسه فعالی نیست') + '</dd></div><div><dt>بازگشت</dt><dd>دکمه «بازگشت به حساب منابع انسانی» یا انقضا</dd></div></dl>', 'primary') +
      sec('هشدارها', '', checks([lead ? ['no', 'هدف دارای سطح مدیریتی است', 'سیاست هدف مجاز تأیید نشده؛ ادامه مسدود'] : ['ok', 'هدف داخلی غیرممتاز (نمونه)', ''], ['warn', 'این جلسه فقط‌خواندنی نیست', 'احراز هویت به هدف تغییر می‌کند؛ مسدودیت همگانی نوشتن تأیید نشده'], ['warn', 'داده مشتری، جبران خدمات و مالی هدف ممکن است دیده شود', 'محدودیت دقیق میدان/اقدام نیازمند اعتبارسنجی است'], ['warn', 'جلسه فعال است تا بازگشت یا انقضا', 'خروج خودکار ۳۰ دقیقه تضمین نیست'], ['q', 'ثبت دائمی شروع/پایان/دلیل تأیید نشده', 'ردپای موقت ممیزی کامل نیست']])) +
      sec('هدف جلسه (الزامی)', '', '<div class="recips-inline" role="radiogroup" aria-label="هدف جلسه">' + purposes.map(function (t, i) { var on = keep.purpose === t; return '<label class="recip' + (on ? ' on' : '') + '"><input type="radio" name="ip" class="cbx" data-purpose="' + i + '"' + (on ? ' checked' : '') + (lead ? ' disabled' : '') + '><span class="r-main"><b>' + esc(t) + '</b><span>پشتیبانی/تأیید؛ نه انجام کار فروش یا مالی به نام کارمند</span></span></label>'; }).join('') + '</div><label class="confirm"><input type="checkbox" class="cbx" data-ack0' + (keep.ack0 ? ' checked' : '') + (lead ? ' disabled' : '') + '><span>می‌دانم هر اقدام در جلسه با هویت هدف انجام می‌شود و باید برگردم.</span></label>') + '</div>' +
      foot(btn('btn-primary btn-lg', 'imp-review:' + p.id, 'ادامه: تأیید با بررسی اثر', 'send', !valid), null, 'پشتیبانی حساس · نه فقط‌خواندنی');
  };
  X.impSpec = function (p, keep) { return { kind: 'imp', id: p.id, zone: 'restr', title: 'تأیید شروع جلسه نمایش', subject: p.name + ' · WP#' + p.user, pre: '', changes: ['احراز هویت جلسه به ' + p.name + ' تغییر می‌کند', 'نوار هشدار تا بازگشت نمایش داده می‌شود'], unchanged: ['سمت، نقش و دسترسی هدف', 'گذرواژه (نمایش داده نمی‌شود)'], affected: [p.name, 'داده و اقدام‌های قابل‌دسترس هدف', 'ممیزی: عامل/هدف/هدف جلسه/شروع'], effective: 'همان لحظه (' + now + ') تا بازگشت یا انقضا', ack: 'می‌دانم جلسه فقط‌خواندنی نیست و ممیزی ماندگار آن تأیید نشده است.', commit: 'شروع جلسه', p: p.id, purpose: keep.purpose }; };

  /* ---------- Exceptions / audit ---------- */
  D.exc = function (id) {
    var e = X.exc(id), c = X.XCLS[e.cls], s = X.XST[e.st], sub = e.subj;
    var link = sub[0] === 'staff' ? btn('btn-soft', 'open-staff:' + sub[2], 'پروفایل نیرو', 'user') : sub[0] === 'request' ? btn('btn-soft', 'open-req:' + sub[1], 'بررسی درخواست', 'inbox') : '';
    return top('استثنا') + head(esc(e.title), pill(c[0], e.tone, c[1]) + pill(s[0], s[1], s[2]), '<span class="mono">' + e.id + '</span>') + '<div class="dr-body">' + sec('موضوع و مالک', '', '<div class="facets"><div class="facet"><span class="muted">موضوع</span><b class="mono">' + esc(sub[1]) + '</b></div><div class="facet"><span class="muted">مالک</span><b>' + esc(e.owner) + '</b></div><div class="facet"><span class="muted">وضعیت</span><b>' + esc(s[0]) + '</b></div></div><div class="resp-line">' + link + '</div>', 'primary') + sec('اثر', '', '<p class="ind-note">' + esc(e.impact) + '</p>') + sec('اقدام مجاز منابع انسانی', '', checks(e.allowed.map(function (x) { return ['ok', x, '']; }))) + sec('اقدام ممنوع', '', checks(e.notAllowed.map(function (x) { return ['no', x, '']; }))) + sec('مدرک حل', '', checks(e.proof.map(function (x) { return ['q', x, '']; }))) +
      '<section class="sec"><div class="note info">' + ic('info') + '<span>تشخیص مسئله اختیار اصلاح همه حوزه‌ها را نمی‌دهد. SLA یا دستور تشدید تعریف نشده است.</span></div></section></div>' + foot('', null, 'تشخیص ≠ مجوز نوشتن');
  };
  D.audit = function (id) {
    var a = X.audit(id), r = X.OPS_S(a.res);
    return top('رویداد ممیزی') + head(esc(a.act), pill(r[0], r[1], r[2]), '<span class="mono">' + a.id + '</span><span>' + esc(a.at) + '</span>') + '<div class="dr-body">' + sec('ردیابی', '', '<dl class="exc-dl hcp"><div><dt>عامل</dt><dd>' + esc(a.actor) + '</dd></div><div><dt>موضوع / درخواست</dt><dd class="mono">' + esc(a.subj) + ' / ' + esc(a.req) + '</dd></div><div><dt>قبل</dt><dd>' + esc(a.before) + '</dd></div><div><dt>بعد</dt><dd>' + esc(a.after) + '</dd></div><div><dt>دلیل</dt><dd>' + esc(a.reason || '—') + '</dd></div><div><dt>زنجیره بررسی</dt><dd>' + esc(a.chain) + '</dd></div><div><dt>تاریخ اثر</dt><dd>' + esc(a.eff) + '</dd></div><div><dt>زمان اعمال</dt><dd>' + esc(a.applied) + '</dd></div><div><dt>شناسه همبستگی</dt><dd class="mono">' + esc(a.corr) + '</dd></div></dl>', 'primary') + '<section class="sec"><p class="ind-note">' + ic('info') + ' ممیزی کامل و تغییرناپذیر تأیید نشده؛ حقوق، گذرواژه و داده بانکی در این نما ماسک می‌شوند.</p></section></div>' + foot('', null, '');
  };

  /* ---------- Onboarding wizard (staged; credentials are NOT a step) ---------- */
  D.onbwiz = function (id, keep) {
    keep = keep || keepOf(); var step = keep.step || 0, STEPS = ['شناسایی هویت', 'پروفایل/پیوند', 'سمت و سطح', 'جایگاه سلسله‌مراتب', 'دسترسی', 'تکمیل'], body, primary = '', back = step > 0 && step < 5 ? btn('btn-lg btn-ghost', 'onb-back', 'قبلی') : null;
    var stp = '<section class="sec">' + X.steps(STEPS, step) + '</section>';
    if (step === 0) {
      var ch = [['user', 'پیوند به کاربر موجود WP#۴۱۰۶ (ali.fathi)', 'تطبیق نام و ورود — جزئی'], ['profile', 'پیوند به پروفایل موجود HP-312 (بدون حساب)', 'تطبیق موبایل — جزئی'], ['new', 'ساخت کاربر و پروفایل جدید (مسیر مجاز کد)', 'خطر تکرار: دو تطبیق جزئی وجود دارد']];
      body = stp + sec('شخص ورودی', '', '<div class="facets"><div class="facet"><span class="muted">نام</span><b>علی فتحی</b></div><div class="facet"><span class="muted">موبایل (فقط تطبیق)</span><b class="mono">۰۹۱۲···۷۷۳۳</b></div><div class="facet"><span class="muted">هویت شخص سراسری</span><b>تعریف نشده</b></div></div>') +
        sec('نتیجه تطبیق — انتخاب صریح لازم است', pill('تعارض', 'orange', 'swap'), '<div class="recips-inline" role="radiogroup" aria-label="انتخاب تطبیق هویت">' + ch.map(function (c) { var on = keep.choice === c[0]; return '<label class="recip' + (on ? ' on' : '') + '"><input type="radio" name="oc" class="cbx" data-ochoice="' + c[0] + '"' + (on ? ' checked' : '') + '><span class="r-main"><b>' + esc(c[1]) + '</b><span>' + esc(c[2]) + '</span></span></label>'; }).join('') + '</div><p class="ind-note">' + ic('alert') + ' موبایل/نام/ورود مبهم، پیش از ادامه تعارض است؛ هیچ پیوند، ادغام یا ساخت خودکار انجام نمی‌شود و گزینه پیش‌فرض ندارد.</p>');
      primary = btn('btn-primary btn-lg', 'onb-next', 'ادامه', 'arrowL', !keep.choice);
    } else if (step === 1) {
      var lab = { user: 'پیوند به کاربر موجود WP#۴۱۰۶؛ پروفایل جدید ساخته می‌شود', profile: 'پیوند پروفایل HP-312 به یک حساب (ساخت یا پیوند مجاز)', new: 'ساخت کاربر داخلی و پروفایل جدید' }[keep.choice];
      body = stp + sec('اثر این گام', '', checks([['ok', lab, 'شناسه پروفایل و حساب مستقل می‌مانند'], ['info', 'سابقه موجود دست‌نخورده', 'هیچ ادغام مخربی نیست'], ['warn', 'اعتبارنامه اینجا ساخته یا ارسال نمی‌شود', 'مسیر محدود جداست']]));
      primary = btn('btn-primary btn-lg', 'onb-next', 'ادامه', 'arrowL');
    } else if (step === 2) {
      body = stp + sec('سمت و سطح', '', '<div class="recips-inline" role="radiogroup" aria-label="سمت">' + M.positions.filter(function (p) { return ['seller', 'sup', 'senior'].indexOf(p.id) > -1; }).map(function (p) { var on = keep.pos === p.id; return '<label class="recip' + (on ? ' on' : '') + '"><input type="radio" name="op" class="cbx" data-opos="' + p.id + '"' + (on ? ' checked' : '') + '><span class="r-main"><b>' + esc(p.label) + '</b><span>سطح ' + fa(p.level) + ' · ' + esc(p.role) + '</span></span></label>'; }).join('') + '</div><p class="ind-note">' + ic('key') + ' سمت، نگاشت به نقش عملیاتی و دسترسی را تحت تأثیر می‌گذارد؛ اثر در گام دسترسی نشان داده می‌شود.</p>');
      primary = btn('btn-primary btn-lg', 'onb-next', 'ادامه', 'arrowL', !keep.pos);
    } else if (step === 3) {
      var mg = M.staff.filter(function (x) { return ['manager', 'sup'].indexOf(x.pos) > -1; });
      body = stp + sec('مدیر مستقیم', '', '<div class="recips-inline" role="radiogroup" aria-label="مدیر مستقیم">' + mg.map(function (m) { var okm = X.isMgrOk(m.id), on = keep.par === m.id; return '<label class="recip' + (okm ? '' : ' off') + (on ? ' on' : '') + '"><input type="radio" name="opar" class="cbx" data-opar="' + m.id + '"' + (on ? ' checked' : '') + (okm ? '' : ' disabled') + '><span class="r-main"><b>' + esc(m.name) + '</b><span>' + esc(X.posL(m.pos)) + (okm ? '' : ' — غیرفعال: قابل انتخاب نیست') + '</span></span></label>'; }).join('') + '</div><p class="ind-note">' + ic('info') + ' چرخه ایجاد نمی‌شود؛ مدیر فعلی ثبت می‌شود و تاریخ اثر برابر زمان اعمال است.</p>');
      primary = btn('btn-primary btn-lg', 'onb-next', 'ادامه', 'arrowL', !keep.par);
    } else if (step === 4) {
      var pos = keep.pos, base = X.POSPERM[pos], sp = { changes: ['پروفایل و پیوند حساب طبق انتخاب هویت', 'سمت: ' + X.posL(pos), 'مدیر: ' + X.name(keep.par), 'نگاشت سمت ← نقش عملیاتی (دسترسی)'], unchanged: ['اعتبارنامه (گذرواژه/پیامک) — ایجاد یا ارسال نمی‌شود', 'کاربران و پروفایل‌های دیگر', 'آموزش/سند — گام تعریف‌نشده'], affected: ['علی فتحی', 'مدیر ' + X.name(keep.par), 'مجوزهای محصولی ' + X.posL(pos)], effective: 'زمان اعمال (' + now + ')', ack: 'می‌دانم فعال‌شدن پرچم، اثبات ورود قابل‌استفاده نیست و اثرهای باز ممکن است بماند.' };
      body = stp + sec('پیش‌نمایش دسترسی', '', '<ul class="elig">' + M.perms.map(function (r) { return '<li class="e-' + (base[r.k] === 'allow' ? 'ok' : 'info') + '">' + ic(base[r.k] === 'allow' ? 'checkCircle' : 'ban') + '<span><b>' + esc(r.label) + '</b><span>' + (base[r.k] === 'allow' ? 'مجاز' : 'غیرمجاز') + (r.sens ? ' · حساس' : '') + '</span></span></li>'; }).join('') + '</ul>') + X.sensBlocks(sp, keep);
      primary = btn('btn-primary btn-lg', 'onb-commit', 'ثبت ورود نیرو', 'send', !(ok3(keep.reason) && keep.ack));
    } else {
      var eff = keep.effects || [];
      body = stp + '<section class="sec primary"><p class="result-line">' + ic('alert') + 'ورود نیرو ناقص تکمیل شد؛ هر اثر جدا گزارش می‌شود.</p>' + X.outcomeStrip({ requested: eff.length, items: eff }) + '</section>' + sec('اثرها', '', '<ul class="ir-list">' + eff.map(function (e) { var o = X.OUT[e[1]]; return '<li class="ir">' + pill(o.label, o.tone, o.icon) + '<span>' + esc(e[0]) + '</span><span class="ir-why">' + esc(e[2] || '—') + '</span></li>'; }).join('') + '</ul><p class="ind-note">' + ic('info') + ' یک پیام موفقیت دلیل کامل‌بودن نیست؛ فعال‌سازی ورود قابل‌استفاده تأیید نشده است.</p>');
      primary = btn('btn-primary btn-lg', 'goto:work', 'بازگشت به نیروها', 'arrowL');
    }
    return top('ورود نیروی جدید') + head('ورود نیروی جدید', pill('مرحله‌ای', 'blue', 'layers'), '<span>اعتبارنامه جزو این مراحل نیست</span>') + '<div class="dr-body">' + body + '</div>' + foot(primary, back, 'ثبت واقعی در نمونه انجام نمی‌شود');
  };
  X.commitOnb = function (keep) {
    var eff = [['پروفایل نیرو', 'ok', keep.choice === 'profile' ? 'پروفایل HP-312 به‌روز شد' : 'پروفایل جدید ساخته شد'], ['پیوند حساب وردپرس', keep.choice === 'new' ? 'ok' : 'ok', keep.choice === 'new' ? 'حساب داخلی ساخته شد (مسیر مجاز)' : 'حساب پیوند خورد'], ['سمت و سطح', 'ok', X.posL(keep.pos)], ['جایگاه سلسله‌مراتب', 'ok', 'مدیر: ' + X.name(keep.par) + ' · تاریخ اثر = زمان اعمال'], ['فعال‌سازی دسترسی', 'unknown', 'ورود قابل‌استفاده تأیید نشده؛ پرچم فعال بودن اثبات نیست'], ['اعتبارنامه', 'skipped', 'جزو ورود نیرو نیست؛ مسیر محدود جدا']];
    keep.effects = eff; keep.step = 5; pushAudit({ act: 'ورود نیرو', subj: 'جدید', req: '—', before: '—', after: 'پروفایل/پیوند/سلسله‌مراتب اعمال شد؛ دسترسی نامعلوم', reason: keep.reasonText || keep.reason || '', chain: '—', eff: now, applied: now, res: 'partial', corr: 'hr-onb' });
  };

  /* ---------- Bulk ---------- */
  D.bulkwf = function () {
    var keep = keepOf(), ids = ['HP-301', 'HP-302', 'HP-303', 'HP-305', 'HP-306', 'HP-308', 'HP-311'];
    return top('به‌روزرسانی گروهی نیروها (مشروط)') + head('۷ نیرو · میدان: واحد / تیم', pill('مشروط', 'amber', 'alert'), '<span>فقط میدان‌های ارائه‌شده</span>') + '<div class="dr-body">' + sec('شناسه‌های دقیق و میدان', '', '<ul class="mini-list">' + ids.map(function (i) { return '<li class="mini"><div class="grow"><b>' + esc(X.name(i)) + '</b><div class="muted" style="font-size:var(--t-meta)"><span class="mono">' + i + '</span> · ' + esc(X.s(i).unit) + '</div></div>' + pill('واجد شرایط (پیش‌نمایش)', 'teal', 'checkCircle') + '</li>'; }).join('') + '</ul>', 'primary') + sec('مرز', '', checks([['ok', 'فقط شناسه‌های انتخاب‌شده و میدان «واحد/تیم»', 'فیلتر صفحه ≠ همه'], ['no', 'سمت، مدیر، دسترسی و جبران خدمات در این گروهی نیست', 'تغییر حساس جدا و مشروط است'], ['warn', 'نتیجه هر نیرو جدا می‌آید', 'خلاصه موفق، همه‌چیز کامل نیست']])) + '<section class="sec"><label class="confirm"><input type="checkbox" class="cbx" data-bulkack' + (keep.ack ? ' checked' : '') + '><span>می‌دانم نتیجه ممکن است ناقص باشد و تغییر حساس در این عملیات نیست.</span></label></section></div>' + foot(btn('btn-primary btn-lg', 'commit-bulkwf', 'ثبت به‌روزرسانی گروهی', 'send', !keep.ack), null, 'APPLY در نسخه واقعی');
  };
  X.runBulkWf = function () {
    var ids = ['HP-301', 'HP-302', 'HP-303', 'HP-305', 'HP-306', 'HP-308', 'HP-311'], PAT = ['ok', 'ok', 'skipped', 'failed', 'ok', 'unknown', 'ok'], R = { skipped: 'تغییر پس از بررسی: پروفایل در حال ویرایش دیگری است', failed: 'خطای ثبت؛ قابل تکرار', unknown: 'پاسخ ثبت نرسید' };
    var items = ids.map(function (i, n) { return [X.name(i) + ' (' + i + ')', PAT[n], R[PAT[n]] || '']; });
    var op = X.newOp({ kind: 'bulkwf', title: 'به‌روزرسانی گروهی واحد/تیم', requested: items.length, items: items, note: 'فقط میدان واحد/تیم؛ سمت، مدیر و دسترسی بدون تغییر ماند.' });
    C.closeDrawer(); C.render(); C.openDrawer('result', op.id);
  };
  D.bulkcred = function () {
    var keep = keepOf();
    return top('بازنشانی گروهی اعتبارنامه (محدود)') + head('۶ حساب غیرممتاز', X.zone('restr'), '<span>مسیر حساس جدا از به‌روزرسانی عادی</span>') + '<div class="dr-body">' + sec('مرز', '', checks([['warn', 'مسیر محدود/مشروط', 'خط پایه: هیچ بازنشانی گروهی پیش‌فرض نیست'], ['no', 'گذرواژه یکسان برای همه توصیه نمی‌شود', 'هر نتیجه جدا است'], ['info', 'بازنشانی و پیامک دو نتیجه جدا', 'مقدار گذرواژه نمایش یا ثبت نمی‌شود'], ['no', 'اهداف مدیریتی در این گروهی نیستند', 'سیاست هدف مجاز تأیید نشده (HR-G04)']]), 'primary') + '<section class="sec"><label class="confirm"><input type="checkbox" class="cbx" data-bulkack' + (keep.ack ? ' checked' : '') + '><span>می‌دانم شکست پیامک به معنی شکست بازنشانی نیست و نتیجه ممکن است ناقص باشد.</span></label></section></div>' + foot(btn('btn-primary btn-lg', 'commit-bulkcred', 'ثبت بازنشانی گروهی', 'send', !keep.ack), null, 'APPLY در نسخه واقعی');
  };
  X.runBulkCred = function () {
    var users = ['HP-301', 'HP-302', 'HP-303', 'HP-305', 'HP-307', 'HP-309'], P = [['ok', 'ok'], ['ok', 'failed'], ['ok', 'ok'], ['failed', 'na'], ['ok', 'failed'], ['unknown', 'na']];
    var items = users.map(function (u, i) { return [X.name(u) + ' (' + u + ')', P[i][0], P[i][0] === 'ok' ? 'بازنشانی انجام شد؛ مقدار ثبت نشد' : P[i][0] === 'failed' ? 'بازنشانی ناموفق؛ قابل تکرار پس از بررسی' : 'پاسخ بازنشانی نرسید', P[i][1]]; });
    var op = X.newOp({ kind: 'bulkcred', title: 'بازنشانی گروهی اعتبارنامه', requested: 6, items: items, sms: { ok: 1, failed: 2 }, retry: true, note: 'بازنشانی و اعلان پیامکی دو نتیجه جدا هستند.' });
    C.closeDrawer(); C.render(); C.openDrawer('result', op.id);
  };
  D.export = function () {
    return top('خروجی گروهی (مشروط)') + head('خروجی نیروها', pill('مشروط · غیرفعال', 'amber', 'lock'), '<span>قرارداد sn_hr_users_v1</span>') + '<div class="dr-body">' + sec('دامنه', '', checks([['ok', 'انتخاب‌شده (حداکثر ۵٬۰۰۰) در برابر همه', 'فیلتر صفحه ≠ همه؛ شناسه‌ها و دامنه از سرور'], ['no', 'حقوق، مخاطب کامل، داده بانکی و گذرواژه در خروجی نیست', 'فهرست میدان مجاز لازم است'], ['q', 'ماسک‌بودن لاگ ≠ ماسک‌بودن خروجی', 'رفتار خروجی در زمان اجرا تأیید نشده']]), 'primary') + '</div>' + foot(btn('btn-lg', 'x', 'اجرای خروجی غیرفعال در نمونه', 'lock', true), null, '');
  };
})();
