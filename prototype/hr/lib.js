/* HR — role layer, part 1: state, namespaced dictionaries, workforce/temporal helpers and shared presentation helpers.
   Design prototype; mock data only. HR ROLE RUNTIME IS NOT LIVE VERIFIED. Nothing here hires, transfers, terminates, changes access,
   resets credentials, impersonates or edits compensation. Product boundaries: HR-PRODUCT-SPEC.md + CRM-CROSS-ROLE-ARCHITECTURE-V1.md + GATE-0-PRODUCT-INVARIANTS.md.
   HR = Workforce + Structure + Access Lifecycle + Request Application + Temporal History. Visibility never becomes unrelated authority. */
(function () {
  'use strict';
  var C = window.CRM, M = window.HR, h = C.h;
  var esc = h.esc, fa = h.fa, ic = h.ic, pill = h.pill, hint = h.hint;
  var X = window.HRX = { C: C, M: M, h: h, V: {}, D: {} };

  /* ---------- Role state ---------- */
  X.st = {
    wq: 'all', oq: 'onb', rq: 'review', aq: 'matrix', xq: 'exc', bq: 'ops', dq: 'map', cq: 'comp',
    compAuth: 'authorized', imp: null, flow: null, ops: [], reqState: {}, staffEdit: {}, sel: {}, spec: null
  };

  /* ---------- Namespaced dictionaries (label + tone + icon; never colour alone) ---------- */
  X.EMP = { active: { label: 'فعال', tone: 'teal', icon: 'checkCircle' }, inactive: { label: 'غیرفعال', tone: 'slate', icon: 'ban' }, terminated: { label: 'پایان همکاری', tone: 'red', icon: 'xCircle' } };
  X.ACC = { ok: { label: 'هم‌خوان', tone: 'teal', icon: 'checkCircle' }, mismatch: { label: 'ناهمخوان', tone: 'orange', icon: 'swap' }, partial: { label: 'اعمال ناقص', tone: 'orange', icon: 'split' }, nouser: { label: 'بدون حساب', tone: 'amber', icon: 'question' } };
  X.LK = { ok: ['پیوند تأییدشده', 'ok', 'checkCircle', 'پیوند با شناسه پایدار ثبت شده است.'], partial: ['پیوند جزئی', 'partial', 'layers', 'فقط تطبیق نام/موبایل؛ شناسه شخص سراسری تعریف نشده و هویت شخص حدس زده نمی‌شود.'], missing: ['پیوند ناموجود', 'undef', 'question', 'هیچ حساب کاربری پیوند نخورده؛ ساخت یا پیوند فقط با مسیر مجاز ورود نیرو.'], conflict: ['تعارض پیوند', 'recon', 'swap', 'بیش از یک پروفایل به همین حساب اشاره می‌کند؛ ادغام خودکار انجام نمی‌شود.'] };
  X.REQ = {
    pending_review: { label: 'در انتظار بررسی', tone: 'orange', icon: 'hourglass', id: 'pending_review' }, approved_step: { label: 'تأیید مرحله‌ای (اعمال نشده)', tone: 'blue', icon: 'check', id: 'approved_step' },
    pending_hr: { label: 'در انتظار منابع انسانی', tone: 'orange', icon: 'hourglass', id: 'pending_hr' }, approved: { label: 'تأیید و اعمال‌شده', tone: 'green', icon: 'checkCircle', id: 'approved + applied_at' },
    rejected: { label: 'رد شده', tone: 'red', icon: 'xCircle', id: 'rejected' }, failed: { label: 'اعمال ناموفق', tone: 'red', icon: 'alert', id: 'failed' }
  };
  X.CONF = { inactiveMgr: 'مدیر غیرفعال', dupProfile: 'پروفایل تکراری', noUser: 'بدون حساب', roleMismatch: 'ناهمخوانی نقش/سمت', noParent: 'بدون والد', accessPartial: 'دسترسی ناقص', histGap: 'شکاف تاریخچه', termOpen: 'پایان همکاری با کار باز' };
  X.XCLS = { hier: ['سلسله‌مراتب نامعتبر', 'link'], inactiveMgr: ['مدیر غیرفعال با نیروی فعال', 'user'], pending: ['درخواست منتظر', 'hourglass'], failed: ['اعمال ناموفق', 'alert'], dup: ['پروفایل تکراری', 'copy'], noUser: ['پیوند حساب ناموجود', 'unlink'], roleMismatch: ['ناهمخوانی نقش و سمت', 'swap'], termOpen: ['پایان همکاری با کار باز', 'lock'], oos: ['درخواست خارج از محدوده', 'ban'], accessPartial: ['ناهمخوانی دسترسی', 'key'], histGap: ['شکاف تاریخچه انتساب', 'history'] };
  X.XST = { open: ['باز', 'orange', 'inbox'], awaiting: ['منتظر مالک', 'slate', 'hourglass'] };
  X.SRC = { position: ['نگاشت سمت', 'briefcase'], role: ['نقش (قدیمی/WP)', 'key'], direct: ['استثنای مستقیم', 'edit'], inherit: ['ارث‌بری از والد', 'users'], none: ['منبعی نیست', 'dashed'] };
  X.OUT = { ok: { label: 'اعمال شد', tone: 'green', icon: 'checkCircle' }, skipped: { label: 'ارسال نشد', tone: 'orange', icon: 'swap' }, rejected: { label: 'ردشده توسط سامانه', tone: 'red', icon: 'ban' }, failed: { label: 'ناموفق · قابل تکرار', tone: 'red', icon: 'refresh' }, unknown: { label: 'نامعلوم', tone: 'amber', icon: 'question' } };
  X.OPS_S = function (k) { return { complete: ['کامل', 'green', 'checkCircle'], partial: ['ناقص (موفقیت جزئی)', 'orange', 'split'], failed: ['ناموفق', 'red', 'xCircle'], unknown: ['نتیجه نامعلوم', 'amber', 'question'] }[k]; };
  X.ZONES = { normal: ['عادی', 'z-normal', 'eye'], cond: ['مشروط', 'z-adv', 'alert'], restr: ['محدود · حساس', 'z-maint', 'lock'] };

  /* ---------- Entity helpers ---------- */
  X.keys = function (o) { return Object.keys(o).filter(function (k) { return o[k]; }); };
  X.sim = function () { return C.state.sim; };
  X.s = function (id) { return M.staff.filter(function (p) { return p.id === id; })[0]; };
  X.name = function (id) { var p = X.s(id); return p ? p.name : '—'; };
  X.pos = function (id) { return M.positions.filter(function (p) { return p.id === id; })[0]; };
  X.posL = function (id) { var p = X.pos(id); return p ? p.label : '—'; };
  X.req = function (id) { return M.requests.filter(function (r) { return r.id === id; })[0]; };
  X.reqSt = function (r) { return X.st.reqState[r.id] || r.st; };
  X.exc = function (id) { return M.exc.filter(function (e) { return e.id === id; })[0]; };
  X.audit = function (id) { return M.audit.filter(function (e) { return e.id === id; })[0]; };
  X.empOf = function (p) { return (X.st.staffEdit[p.id] && X.st.staffEdit[p.id].emp) || p.emp; };
  X.isMgrOk = function (id) { var p = X.s(id); return !!p && X.empOf(p) === 'active'; };
  X.subordinates = function (id) { return M.staff.filter(function (p) { return p.parent === id; }); };
  X.curHist = function (p) { return p.hist[0]; };

  /* ---------- Shared presentation helpers (reusable patterns; no business capability) ---------- */
  X.cov = function (k, label) {
    var c = { ok: ['کامل', 'ok', 'checkCircle', 'همه منابع این نما دریافت شده‌اند.'], partial: ['ناقص', 'partial', 'layers', 'بخشی از داده یا تاریخچه دریافت نشده؛ صفر فرض نمی‌شود.'], stale: ['قدیمی', 'stale', 'clock', 'آخرین داده قدیمی است؛ پیش از تصمیم بازخوانی کنید.'], recon: ['نیازمند تطبیق', 'recon', 'swap', 'دو منبع با هم نمی‌خوانند؛ عدد یا وضعیت دستی تغییر داده نمی‌شود.'], undef: ['نامعلوم', 'undef', 'question', 'مدرک کافی نیست؛ حدس زده نمی‌شود.'], unauth: ['غیرمجاز', 'undef', 'lock', 'دسترسی به این میدان تأیید نشده؛ صفر یا خالی نمایش داده نمی‌شود.'] }[k] || ['', 'ok', 'info', ''];
    return '<span class="cov c-' + c[1] + ' tip" tabindex="0" data-tip="' + esc(c[3]) + '">' + ic(c[2]) + esc(label || c[0]) + '</span>';
  };
  X.lk = function (k, label) { var c = X.LK[k]; return '<span class="lc c-' + c[1] + ' tip" tabindex="0" data-tip="' + esc(c[3]) + '">' + ic(c[2]) + esc(label || c[0]) + '</span>'; };
  X.zone = function (k, extra) { var z = X.ZONES[k]; return '<span class="zone ' + z[1] + '">' + ic(z[2]) + z[0] + (extra ? ' · ' + esc(extra) : '') + '</span>'; };
  X.basis = function (k) {
    var b = { snap: ['وضعیت فعلی', 'clock', 'ساختار و وضعیت امروز؛ انتساب یا اعتبار گذشته را تعیین نمی‌کند.', 'current'], hist: ['تاریخچه بازه‌ای', 'history', 'بازه‌های ثبت‌شده با تاریخ اثر و زمان اعمال؛ جای خالی «نامعلوم» است و از مدیر امروز پر نمی‌شود.', 'hist'], none: ['بدون بازنویسی تاریخ', 'lock', 'تغییر فعلی، مالک تاریخی، عامل رویداد یا مالک اعتبار گذشته را بازنویسی نمی‌کند.', 'event'] }[k];
    return '<span class="basis b-' + b[3] + ' tip" tabindex="0" data-tip="' + esc(b[2]) + '">' + ic(b[1]) + b[0] + '</span>';
  };
  X.empPill = function (e) { var d = X.EMP[e]; return pill(d.label, d.tone, d.icon); };
  X.accPill = function (a) { var d = X.ACC[a]; return pill(d.label, d.tone, d.icon); };
  X.reqPill = function (k) { var d = X.REQ[k]; return '<span class="tip" tabindex="0" data-tip="شناسه داخلی وضعیت: ' + d.id + '">' + pill(d.label, d.tone, d.icon) + '</span>'; };
  X.who = function (name, sub) { return '<div class="who"><b>' + esc(name) + '</b>' + (sub ? '<span class="phone">' + sub + '</span>' : '') + '</div>'; };
  // IdentityLinkage — Person ↔ Workforce Profile ↔ WordPress User are three things; each link has its own state.
  X.linkage = function (p) {
    var node = function (k, label, val, cls) { return '<div class="idn ' + (cls || '') + '"><small>' + label + '</small><b class="mono">' + esc(val) + '</b></div>'; };
    var link = function (st) { return '<div class="idk">' + X.lk(st) + '</div>'; };
    return '<div class="idl" role="group" aria-label="پیوند شخص، پروفایل نیرو و حساب کاربری">' + node('p', 'شخص', p.name + ' · هویت سراسری تعریف نشده', 'idn-person') + link(p.lk[0]) + node('f', 'پروفایل نیرو', p.id + (p.dup ? ' / ' + p.dup : '')) + link(p.lk[1]) + node('u', 'حساب وردپرس', p.user ? 'WP#' + p.user : 'پیوند نشده', p.user ? '' : 'idn-missing') + '</div>';
  };
  X.banners = function (view) {
    var s = X.sim(); if (!s) return '';
    var b = function (k, t, a) { return h.banner(k, t, a || ''); };
    if (s === 'stale') return b('stale', '<b>فهرست نیروها قدیمی است (' + M.freshness.stale + ').</b> وضعیت فعلی، مدیر یا دسترسی ممکن است عوض شده باشد؛ پیش از تغییر بازخوانی کنید.', '<button type="button" class="btn btn-sm" data-refresh>' + ic('refresh') + 'بازخوانی</button>');
    if (s === 'incomplete' && (view === 'work' || view === 'comp')) return b('incomplete', '<b>داده ناقص است:</b> بخشی از پروفایل‌ها/دوره‌ها دریافت نشد. جمع‌ها نهایی نیستند و نبود مورد، صفر یا «نبود نیرو» نیست.', '<button type="button" class="btn btn-sm" data-refresh>' + ic('refresh') + 'تلاش مجدد</button>');
    if (s === 'reqchanged' && view === 'req') return b('incomplete', '<b>درخواست پیش از بررسی تغییر کرده است:</b> مدیر فعلی فرد پس از ثبت درخواست عوض شده. پیش از هر تأیید، درخواست و فرد دوباره خوانده می‌شوند؛ تأیید قبلی خودکار معتبر نیست.', '');
    if (s === 'targetinactive' && (view === 'onb' || view === 'req')) return b('incomplete', '<b>مدیر مقصد غیرفعال است.</b> انتقال به مدیر غیرفعال بدون بررسی تازه سیاست و جانشین مجاز نیست؛ هیچ مدیر جایگزینی خودکار انتخاب نمی‌شود.', '');
    if (s === 'dupprofile' && (view === 'work' || view === 'onb')) return b('incomplete', '<b>پروفایل تکراری:</b> دو پروفایل به یک حساب اشاره می‌کنند. ادغام مخرب یا ساخت خودکار انجام نمی‌شود؛ مقایسه شناسه‌های بومی لازم است.', '');
    if (s === 'accesspartial' && (view === 'acc' || view === 'work')) return b('incomplete', '<b>اعمال دسترسی ناقص:</b> برخی مجوزها اعمال شد و برخی نه. وضعیت مؤثر با وضعیت مورد انتظار فرق دارد؛ «موفق» اعلام نمی‌شود.', '');
    if (s === 'termincomplete' && (view === 'req' || view === 'onb' || view === 'work')) return b('incomplete', '<b>تحویل پایان همکاری ناتمام است (OPD-05).</b> تغییر وضعیت اشتغال، تحویل کامل کار نیست؛ گیرنده و زمان لغو دسترسی هنوز تعیین نشده است.', '');
    if (s === 'effconflict' && (view === 'onb' || view === 'req')) return b('incomplete', '<b>تعارض تاریخ اثر:</b> تاریخ درخواستی با بازه فعلی هم‌پوشانی دارد یا قبل از اعمال قبلی است. تاریخ اثر و زمان اعمال جدا ثبت می‌شوند و تاریخ گذشته خودکار پر نمی‌شود.', '');
    if (s === 'histunavail' && (view === 'work' || view === 'exc')) return b('incomplete', '<b>تاریخچه انتساب در دسترس نیست:</b> بازه‌های پیشین دریافت نشد. «نامعلوم» نمایش داده می‌شود و از مدیر امروز جایگزین نمی‌شود.', '');
    if (s === 'credpartial' && view === 'cred') return b('incomplete', '<b>بازنشانی انجام شد ولی اعلان ناموفق بود.</b> نتیجه بازنشانی و نتیجه پیامک دو نتیجه جدا هستند؛ شکست پیامک به معنی شکست بازنشانی نیست.', '');
    return '';
  };
  X.steps = function (list, cur) { return '<ol class="steps" aria-label="مراحل">' + list.map(function (s, i) { var c = i < cur ? 'done' : i === cur ? 'cur' : ''; return '<li class="' + c + '"' + (i === cur ? ' aria-current="step"' : '') + '><span class="sn">' + (i < cur ? ic('check') : fa(i + 1)) + '</span><span class="st-l">' + esc(s) + '</span></li>'; }).join('') + '</ol>'; };
  X.bulkbar = function (text, sub, actions) { return '<div class="bulkbar" role="region" aria-label="انتخاب گروهی"><div class="bb-txt"><b>' + text + '</b>' + (sub ? '<span>' + sub + '</span>' : '') + '</div><span class="grow"></span>' + actions + '</div>'; };
  X.top = function (title, extra) { return '<div class="sheet-grip" aria-hidden="true"></div><div class="dr-top"><span>' + esc(title) + '</span><span class="grow"></span>' + (extra || '') + '<button type="button" class="btn btn-ghost btn-icon btn-sm tip" data-tip="بستن (Esc)" data-close aria-label="بستن">' + ic('x') + '</button></div>'; };
  X.foot = function (primary, secondary, hintTxt) { return '<div class="dr-foot">' + (primary || '') + (secondary || '<button type="button" class="btn btn-lg btn-ghost" data-close>بستن</button>') + '<span class="grow"></span>' + (hintTxt ? '<span class="hint">' + hintTxt + '</span>' : '') + '</div>'; };
  X.tl = function (items) { return '<ul class="timeline">' + items.map(function (x) { return '<li>' + esc(x[0]) + '<span>' + esc(x[1]) + (x[2] ? ' · ' + esc(x[2]) : '') + '</span></li>'; }).join('') + '</ul>'; };
  X.checks = function (list) { return '<ul class="elig">' + list.map(function (c) { var i = { ok: 'checkCircle', no: 'xCircle', warn: 'alert', q: 'question', info: 'info' }[c[0]]; return '<li class="e-' + c[0] + '">' + ic(i) + '<span><b>' + esc(c[1]) + '</b>' + (c[2] ? '<span>' + esc(c[2]) + '</span>' : '') + '</span></li>'; }).join('') + '</ul>'; };
  X.chain = function (arr) { return '<ol class="chain">' + arr.map(function (s) { var t = { done: 'check', cur: 'hourglass', rejected: 'xCircle', skip: 'dashed' }[s[1]]; return '<li class="c-' + s[1] + '"><span class="c-dot">' + ic(t) + '</span><span class="c-t"><b>' + esc(s[0]) + '</b><span>' + esc(s[2]) + '</span></span></li>'; }).join('') + '</ol>'; };
  X.sec = function (title, aside, inner, cls) { return '<section class="sec' + (cls ? ' ' + cls : '') + '"><div class="sec-h"><h3>' + title + '</h3>' + (aside ? '<span class="aside">' + aside + '</span>' : '') + '</div>' + inner + '</section>'; };
  X.details = function (title, count, inner, open) { return '<details class="sec"' + (open ? ' open' : '') + '><summary><h3>' + title + '</h3>' + (count != null ? '<span class="muted" style="margin-right:8px;font-size:var(--t-meta)">' + fa(count) + '</span>' : '') + '<span class="chev">' + ic('chev') + '</span></summary>' + inner + '</details>'; };
  X.counts = function (o) {
    if (o.c) return o.c;
    var n = function (k) { return o.items.filter(function (x) { return x[1] === k; }).length; };
    return { requested: o.requested, eligible: o.requested - n('skipped') - n('rejected'), ok: n('ok'), skipped: n('skipped'), rejected: n('rejected'), failed: n('failed'), unknown: n('unknown') };
  };
  X.outcomeStrip = function (o) {
    var c = X.counts(o), cells = [['requested', 'درخواست‌شده', c.requested], ['eligible', 'واجد شرایط', c.eligible], ['applied', 'اعمال‌شده', c.ok], ['skipped', 'ارسال‌نشده', c.skipped], ['rejected', 'ردشده', c.rejected], ['failed', 'ناموفق', c.failed], ['unknown', 'نامعلوم', c.unknown]];
    return '<div class="outcome-strip" role="group" aria-label="خلاصه نتیجه گروهی">' + cells.map(function (x) { return '<div class="os os-' + x[0] + (x[2] ? '' : ' zero') + '"><span>' + x[1] + '</span><b>' + fa(x[2]) + '</b></div>'; }).join('') + '</div>' +
      '<p class="os-note">' + (c.ok + c.skipped + c.rejected + c.failed + c.unknown === c.requested ? ic('checkCircle') + 'مجموع «اعمال‌شده + ارسال‌نشده + ردشده + ناموفق + نامعلوم» برابر «درخواست‌شده» است.' : ic('alert') + 'جمع‌ها با هم نمی‌خوانند؛ پیش از هر اقدام بازخوانی کنید.') + '</p>';
  };
  // EffectiveIntervalTimeline — effective_from/to, applied time, event actor, source; missing history = UNKNOWN / INCOMPLETE (never inferred from today's hierarchy).
  X.eit = function (p) {
    return '<ol class="eit" aria-label="بازه‌های اثر ساختار و سمت">' + p.hist.map(function (e, i) {
      if (e.state === 'unknown') return '<li class="ei ei-unknown"><span class="ei-dot" aria-hidden="true"></span><div class="ei-b"><b>' + pill('نامعلوم (UNKNOWN)', 'slate', 'question') + '</b><span class="muted">' + esc(e.src) + ' — از مدیر امروز جایگزین نمی‌شود</span></div></li>';
      var cur = i === 0 && !e.to, inc = e.state === 'incomplete';
      return '<li class="ei' + (cur ? ' ei-cur' : '') + (inc ? ' ei-inc' : '') + '"><span class="ei-dot" aria-hidden="true"></span><div class="ei-b"><div class="ei-h"><b>' + (cur ? 'فعلی' : 'تاریخی') + '</b>' + (cur ? pill('بازه جاری', 'teal', 'clock') : '') + (inc ? pill('ناقص (INCOMPLETE)', 'amber', 'layers') : '') + '</div>' +
        '<dl class="ei-dl"><div><dt>تاریخ اثر (از — تا)</dt><dd>' + esc(e.from) + ' — ' + (e.to ? esc(e.to) : 'ادامه دارد') + '</dd></div><div><dt>مدیر در این بازه</dt><dd>' + (e.parent ? esc(X.name(e.parent)) : 'ثبت نشده') + '</dd></div><div><dt>سمت</dt><dd>' + esc(e.pos) + '</dd></div><div><dt>زمان اعمال</dt><dd>' + esc(e.applied || 'ثبت نشده') + '</dd></div><div><dt>عامل رویداد</dt><dd>' + esc(e.actor || '—') + '</dd></div><div><dt>منشأ</dt><dd>' + esc(e.src) + '</dd></div></dl></div></li>';
    }).join('') + '</ol>';
  };
  // RequestedVsApplied — requested change and actual applied change are two columns; approval of a step is neither.
  X.rva = function (req, actual, note) {
    return '<div class="rva" role="group" aria-label="تغییر درخواستی در برابر تغییر اعمال‌شده"><div class="rv-c"><span class="own-l">' + ic('file') + 'تغییر درخواستی</span><b>' + req + '</b></div><div class="rv-vs" aria-hidden="true">' + ic('swap') + '</div><div class="rv-c rv-act"><span class="own-l">' + ic('checkCircle') + 'تغییر واقعاً اعمال‌شده</span><b>' + actual + '</b></div></div>' + (note ? '<p class="ind-note">' + ic('info') + ' ' + note + '</p>' : '');
  };
  // AccessStateMatrix — source · inherited vs direct · effective · override reason · sensitive delta. Never a capability editor.
  X.asm = function (p, afterPos) {
    var A = M.access[p.id] || M.access['default'], rows = M.perms, ex = null;
    if (afterPos && afterPos !== p.pos) ex = X.accessAfter(p, afterPos);
    return '<div class="tbl-wrap"><table class="tbl no-cursor asm" aria-label="ماتریس وضعیت دسترسی"><caption class="sr">برای هر مجوز محصولی: منبع، ارث‌بری یا مستقیم، وضعیت مؤثر، دلیل استثنا و حساس بودن تغییر' + (ex ? '، و وضعیت پس از تغییر پیشنهادی' : '') + '</caption><thead><tr><th>مجوز محصولی</th><th>منبع</th><th>وضعیت مؤثر</th><th class="col-opt">دلیل استثنا</th>' + (ex ? '<th>پس از تغییر</th>' : '') + '</tr></thead><tbody>' +
      rows.map(function (r) {
        var a = A.rows[r.k], s = X.SRC[a[0]], eff = a[1] === 'allow';
        var after = ex ? ex[r.k] : null;
        var delta = after && after !== a[1];
        return '<tr' + (delta ? ' class="asm-delta"' : '') + '><td><b>' + esc(r.label) + '</b>' + (r.sens ? ' ' + pill('حساس', 'amber', 'alert') : '') + '</td><td>' + ic(s[1]) + ' ' + esc(s[0]) + '</td><td>' + pill(eff ? 'مجاز' : 'غیرمجاز', eff ? 'teal' : 'slate', eff ? 'checkCircle' : 'ban') + '</td><td class="col-opt muted wrap">' + esc(a[2] || '—') + '</td>' + (ex ? '<td>' + (delta ? pill((after === 'allow' ? 'مجاز' : 'غیرمجاز') + ' ← تغییر', after === 'allow' ? 'teal' : 'orange', 'swap') : '<span class="muted">بدون تغییر</span>') + '</td>' : '') + '</tr>';
      }).join('') + '</tbody></table></div>';
  };
  // Positions only map to a small set of permissions; overrides and unrelated roles are preserved (no reset).
  X.POSPERM = { seller: { inv: 'allow', manInv: 'deny', extra: 'deny', team: 'deny', step: 'deny', assign: 'deny' }, sup: { inv: 'allow', manInv: 'deny', extra: 'deny', team: 'allow', step: 'allow', assign: 'deny' }, senior: { inv: 'allow', manInv: 'deny', extra: 'deny', team: 'allow', step: 'allow', assign: 'deny' }, manager: { inv: 'allow', manInv: 'deny', extra: 'allow', team: 'allow', step: 'allow', assign: 'allow' }, deputy: { inv: 'allow', manInv: 'deny', extra: 'allow', team: 'allow', step: 'allow', assign: 'allow' }, hr: { inv: 'deny', manInv: 'deny', extra: 'deny', team: 'deny', step: 'deny', assign: 'deny' } };
  X.accessAfter = function (p, pos) {
    var base = X.POSPERM[pos] || X.POSPERM.seller, A = M.access[p.id] || M.access['default'], out = {};
    M.perms.forEach(function (r) { var a = A.rows[r.k]; out[r.k] = a[0] === 'direct' ? a[1] : base[r.k]; });
    return out;
  };
  X.finish = function () {};
  X.work = function () {
    var n = function (k) { return M.staff.filter(function (p) { return p.conf.indexOf(k) > -1; }).length; };
    return { noUser: n('noUser'), dup: n('dupProfile'), mism: n('roleMismatch'), noParent: n('noParent'), inactive: n('inactiveMgr'), partial: n('accessPartial'), term: n('termOpen'), gap: n('histGap') };
  };
})();
