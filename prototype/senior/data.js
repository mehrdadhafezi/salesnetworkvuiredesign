/* Senior Supervisor prototype — MOCK data only (never a source of truth).
   Shapes follow SENIOR-SUPERVISOR-PRODUCT-SPEC.md and Gate 0: stable CaseRef independent of phone, four ownership concepts
   (+ credit owner), namespaced statuses, eligibility with reasons, financial facets kept separate, metric metadata.
   The sample structure (Senior → 4 Supervisors → Sellers, 1 direct Seller, 1 moved Seller) is NOT hardcoded product logic. */
window.SEN = {
  user: { name: 'کامران صدری', id: '۱۸۲۰۱', role: 'سرپرست ارشد', initials: 'ک‌ص', parent: 'مدیر فروش' },
  freshness: { now: '۶ دقیقه پیش', stale: '۵۲ دقیقه پیش', evaluated: '۱۴۰۵/۰۷/۱۰ ۱۰:۴۲' },

  // Direct Supervisors (current hierarchy). T4 is inactive on purpose (inactive recipient / unresolved responsible actor).
  teams: [
    { id: 'T1', sup: 'سارا احمدی', supId: 5, active: true, cov: 'ok' },
    { id: 'T2', sup: 'رضا مختاری', supId: 21, active: true, cov: 'ok' },
    { id: 'T3', sup: 'لیلا فرهادی', supId: 33, active: true, cov: 'ok' },
    { id: 'T4', sup: 'ندا کیانی', supId: 45, active: false, inactiveNote: 'غیرفعال از ۱۴۰۵/۰۷/۰۴', cov: 'ok' }
  ],

  // rel: direct = reports straight to the Senior; indirect = under a Supervisor (view-only for this role).
  sellers: [
    { id: 6,  name: 'مهدی زاده',   team: 'T1', rel: 'indirect', active: true,  open: 9,  legacy: 3 },
    { id: 11, name: 'نگار رحیمی',  team: 'T1', rel: 'indirect', active: true,  open: 14, legacy: 1 },
    { id: 14, name: 'علی کاظمی',   team: 'T1', rel: 'indirect', active: true,  open: 6,  legacy: 0 },
    { id: 19, name: 'پریسا نوری',  team: 'T1', rel: 'indirect', active: false, open: 4,  legacy: 2, note: 'غیرفعال از ۱۴۰۵/۰۷/۰۵' },
    { id: 23, name: 'حمید قاسمی',  team: 'T2', rel: 'indirect', active: true,  open: 17, legacy: 4 },
    { id: 27, name: 'الهام صفوی',  team: 'T2', rel: 'indirect', active: true,  open: 12, legacy: 0 },
    { id: 31, name: 'بهزاد کریمی', team: 'T2', rel: 'indirect', active: true,  open: 8,  legacy: 5 },
    { id: 35, name: 'مینا شریفی',  team: 'T3', rel: 'indirect', active: true,  open: 11, legacy: 2 },
    { id: 38, name: 'داود امینی',  team: 'T3', rel: 'indirect', active: true,  open: 13, legacy: 1 },
    { id: 40, name: 'کیان رضایی',  team: 'T3', rel: 'indirect', active: true,  open: 7,  legacy: 0, moved: { from: 'T2', when: '۱۴۰۵/۰۷/۰۲' } },
    { id: 46, name: 'سمیرا طاهری', team: 'T4', rel: 'indirect', active: true,  open: 5,  legacy: 1 },
    { id: 44, name: 'حسین بابایی', team: 'direct', rel: 'direct', active: true, open: 6, legacy: 0 }
  ],

  // Own pool = delivered to this Senior, not yet allocated. Reported separately from team workload.
  pool: [
    { id: 'C-31001', source: 'MIS · دسته ۹۱۰۲', received: '۱۴۰۵/۰۷/۰۹', elig: 'ok' },
    { id: 'C-31002', source: 'MIS · دسته ۹۱۰۲', received: '۱۴۰۵/۰۷/۰۹', elig: 'ok' },
    { id: 'C-31003', source: 'MIS · دسته ۹۱۰۲', received: '۱۴۰۵/۰۷/۰۹', elig: 'ok' },
    { id: 'C-31004', source: 'MIS · دسته ۹۱۰۲', received: '۱۴۰۵/۰۷/۰۹', elig: 'ok' },
    { id: 'C-31005', source: 'MIS · دسته ۹۱۰۲', received: '۱۴۰۵/۰۷/۰۹', elig: 'ok' },
    { id: 'C-31006', source: 'MIS · دسته ۹۱۰۲', received: '۱۴۰۵/۰۷/۰۹', elig: 'ok' },
    { id: 'C-31007', source: 'MIS · دسته ۹۱۰۳', received: '۱۴۰۵/۰۷/۱۰', elig: 'ok' },
    { id: 'C-31008', source: 'MIS · دسته ۹۱۰۳', received: '۱۴۰۵/۰۷/۱۰', elig: 'ok' },
    { id: 'C-31009', source: 'MIS · دسته ۹۱۰۳', received: '۱۴۰۵/۰۷/۱۰', elig: 'ok' },
    { id: 'C-31010', source: 'MIS · دسته ۹۱۰۳', received: '۱۴۰۵/۰۷/۱۰', elig: 'ok' },
    { id: 'C-30890', source: 'برگشت از سرپرست', received: '۱۴۰۵/۰۷/۰۶', elig: 'blocked', why: 'به پیش‌فاکتور ۴۹۱۶۴۷۵۴ متصل است (پرداخت‌نشده)' },
    { id: 'C-30771', source: 'MIS · دسته ۹۰۸۸', received: '۱۴۰۵/۰۶/۳۰', elig: 'unknown', why: 'اتصال فاکتور قابل تأیید نیست؛ نیاز به تطبیق MIS' }
  ],

  // Cases currently below the Senior. proof = who executed the transfer that put it there (only own transfers are returnable).
  held: [
    { id: 'C-30110', source: 'MIS · دسته ۹۰۹۰', holder: { kind: 'sup', team: 'T2' }, proof: { by: 'self', op: 'OP-3302', when: '۱۴۰۵/۰۷/۰۸' }, elig: 'ok' },
    { id: 'C-30111', source: 'MIS · دسته ۹۰۹۰', holder: { kind: 'sup', team: 'T2' }, proof: { by: 'self', op: 'OP-3302', when: '۱۴۰۵/۰۷/۰۸' }, elig: 'ok' },
    { id: 'C-30112', source: 'MIS · دسته ۹۰۹۰', holder: { kind: 'sup', team: 'T1' }, proof: { by: 'self', op: 'OP-3290', when: '۱۴۰۵/۰۷/۰۵' }, elig: 'blocked', why: 'fin', invoice: { code: '89663612', label: 'پیش‌فاکتور · پرداخت‌نشده' } },
    { id: 'C-30120', source: 'MIS · دسته ۹۰۸۵', holder: { kind: 'seller', id: 11 }, proof: { by: 'sup', op: 'تخصیص توسط سارا احمدی', when: '۱۴۰۵/۰۷/۰۶' }, elig: 'blocked', why: 'notown' },
    { id: 'C-30125', source: 'MIS · دسته ۹۰۷۰', holder: { kind: 'sup', team: 'T3' }, proof: { by: 'mis', op: 'توزیع MIS', when: '۱۴۰۵/۰۷/۰۱' }, elig: 'blocked', why: 'mis' },
    { id: 'C-30130', source: 'MIS · دسته ۹۰۹۱', holder: { kind: 'sup', team: 'T3' }, proof: { by: 'self', op: 'OP-3302', when: '۱۴۰۵/۰۷/۰۸' }, elig: 'conflict', why: 'مسئول فعلی پس از بارگذاری فهرست تغییر کرده است (انتقال به فروشنده توسط سرپرست)' },
    { id: 'C-30133', source: 'MIS · دسته ۹۰۸۸', holder: { kind: 'sup', team: 'T2' }, proof: { by: 'self', op: 'OP-3290', when: '۱۴۰۵/۰۷/۰۵' }, elig: 'unknown', why: 'اتصال یک فاکتور فاقد کد معتبر است؛ نیاز به تطبیق (MIS / مالی)', link: 'unresolved' },
    { id: 'C-30140', source: 'پیگیری فاکتور V4', holder: { kind: 'sup', team: 'T1' }, proof: { by: 'self', op: 'OP-3290', when: '۱۴۰۵/۰۷/۰۵' }, elig: 'blocked', why: 'consumed', invoice: { code: '55410376', label: 'پرداخت مرحله‌ای · مرحله ۲ از ۳' } },
    { id: 'C-30150', source: 'MIS · دسته ۹۰۷۵', holder: { kind: 'sup', team: 'T2' }, proof: { by: 'self', op: 'OP-3290', when: '۱۴۰۵/۰۷/۰۵' }, elig: 'blocked', why: 'cancel', invoice: { code: '39001276', label: 'لغو شده' } },
    { id: 'C-30160', source: 'MIS · دسته ۹۰۹۱', holder: { kind: 'seller', id: 38 }, proof: { by: 'self', op: 'OP-3302', when: '۱۴۰۵/۰۷/۰۸' }, elig: 'ok' },
    { id: 'L-9001',  source: 'سرنخ قدیمی', legacy: true, holder: { kind: 'seller', id: 40 }, proof: { by: 'none', op: 'سابقه انتقال توسط شما ثبت نشده', when: '—' }, elig: 'unknown', why: 'برگشت سرنخ قدیمی برای نقش سرپرست ارشد هنوز اعتبارسنجی نشده است', link: 'partial' }
  ],

  ops: [
    { ref: 'OP-3302', kind: 'assign', when: 'دیروز ۱۱:۲۰', actor: 'کامران صدری', total: 12, ok: 10, failed: 1, conflict: 1, unknown: 0, to: 'رضا مختاری، لیلا فرهادی',
      items: [['C-30110', 'ok', 'رضا مختاری'], ['C-30111', 'ok', 'رضا مختاری'], ['C-30130', 'ok', 'لیلا فرهادی'], ['C-30160', 'ok', 'لیلا فرهادی'], ['C-30161', 'ok', 'لیلا فرهادی'], ['C-30162', 'ok', 'لیلا فرهادی'], ['C-30163', 'ok', 'رضا مختاری'], ['C-30164', 'ok', 'رضا مختاری'], ['C-30165', 'ok', 'رضا مختاری'], ['C-30166', 'ok', 'رضا مختاری'], ['C-30167', 'retry', 'لیلا فرهادی', 'پاسخ سرور ناقص بود؛ قابل تکرار'], ['C-30168', 'conflict', 'لیلا فرهادی', 'پرونده پس از بررسی توسط MIS برداشته شد']] },
    { ref: 'OP-3298', kind: 'return', when: '۱۴۰۵/۰۷/۰۸ ۱۶:۴۰', actor: 'کامران صدری', total: 3, ok: 3, failed: 0, conflict: 0, unknown: 0, to: 'پنل من', items: [['C-30081', 'ok'], ['C-30082', 'ok'], ['C-30083', 'ok']] },
    { ref: 'OP-3290', kind: 'assign', when: '۱۴۰۵/۰۷/۰۵ ۰۹:۰۵', actor: 'کامران صدری', total: 5, ok: 0, failed: 0, conflict: 0, unknown: 5, to: 'سارا احمدی', note: 'پاسخ تأیید نرسید؛ ممکن است ثبت شده باشد. پیش از تکرار تطبیق کنید.',
      items: [['C-30112', 'unknown'], ['C-30140', 'unknown'], ['C-30133', 'unknown'], ['C-30150', 'unknown'], ['C-30170', 'unknown']] }
  ],

  // Ready conversion (readonly for Senior). sup: responsible Supervisor team id; null = unresolved responsible actor.
  ready: [
    { id: 'R-7101', customer: 'محمد زمانی', team: 'T1', seller: 6, status: 'ready', choice: 'اشتراک طلایی', source: 'v4', stage: 'پرداخت کامل · تأیید مالی انجام شد', next: 'سرپرست مسئول' },
    { id: 'R-7104', customer: 'مریم احمدی', team: 'T1', seller: 11, status: 'ready', choice: null, source: 'v4', stage: 'پرداخت کامل · تأیید مالی انجام شد', next: 'سرپرست مسئول' },
    { id: 'R-7110', customer: 'امیر رحیمی', team: 'T2', seller: 23, status: 'ready', choice: 'اشتراک نقره‌ای', source: 'v4', stage: 'پرداخت کامل · تأیید مالی انجام شد', next: 'سرپرست مسئول' },
    { id: 'R-7112', customer: 'زهرا کاشانی', team: 'T2', seller: 27, status: 'assigned', choice: 'اشتراک ۱۰ ستاره', source: 'v4', stage: 'پرداخت کامل · تأیید مالی انجام شد', next: 'تبدیل‌کننده: رضا مختاری' },
    { id: 'R-7120', customer: 'فاطمه جعفری', team: 'T3', seller: 38, status: 'ready', choice: 'اشتراک طلایی', source: 'v4', stage: 'پرداخت کامل · تأیید مالی انجام شد', next: 'سرپرست مسئول' },
    { id: 'R-7125', customer: 'سینا رزاقی', team: 'T4', seller: 46, status: 'ready', choice: null, source: 'v4', stage: 'پرداخت کامل · تأیید مالی انجام شد', next: 'مسئول نامشخص', unresolved: 'سرپرست این تیم غیرفعال است و جایگزینی ثبت نشده' },
    { id: 'R-7130', customer: 'کیوان نادری', team: 'direct', seller: 44, status: 'ready', choice: null, source: 'v4', stage: 'پرداخت کامل · تأیید مالی انجام شد', next: 'مسئول مستقیم: کامران صدری', direct: true }
  ],

  myConversions: [
    { id: 'R-7140', customer: 'نرگس اکبری', option: 'اشتراک نقره‌ای', stage: 'تماس انجام نشده', remaining: 7500000 }
  ],

  // Invoices (financial facets stay separate). teamAtIssue: team recorded at issue time; null = unknown / not recoverable.
  invoices: [
    { code: '89663612', customer: 'بسی نظری',     seller: 6,  teamAtIssue: 'T1', inv: 'pre',       stage: 1, stages: 1, stg: 'pending',  evidence: 'none',    review: 'none',     total: 15000000, paid: 0,        issued: '۱۴۰۵/۰۷/۰۴', next: { who: 'customer', text: 'پرداخت توسط مشتری' }, link: 'confirmed', bucket: 'pre' },
    { code: '49164754', customer: 'محمد زمانی',   seller: 11, teamAtIssue: 'T1', inv: 'pre',       stage: 1, stages: 1, stg: 'review',   evidence: 'receipt', review: 'pending',  total: 6000000,  paid: 0,        issued: '۱۴۰۵/۰۷/۰۶', next: { who: 'finance', text: 'بررسی مالی رسید' }, link: 'confirmed', bucket: 'finance' },
    { code: '71942055', customer: 'مریم احمدی',   seller: 11, teamAtIssue: 'T1', inv: 'staged',    stage: 2, stages: 3, stg: 'rejected', evidence: 'receipt', review: 'rejected', total: 9000000,  paid: 3000000,  issued: '۱۴۰۵/۰۶/۲۸', reason: 'مبلغ رسید با مبلغ مرحله مطابقت ندارد', next: { who: 'seller', text: 'ثبت رسید اصلاح‌شده' }, link: 'confirmed', bucket: 'rejected' },
    { code: '55410376', customer: 'علی نوری',     seller: 14, teamAtIssue: 'T1', inv: 'staged',    stage: 2, stages: 3, stg: 'pending',  evidence: 'none',    review: 'none',     total: 9000000,  paid: 3000000,  issued: '۱۴۰۵/۰۶/۲۳', next: { who: 'customer', text: 'پرداخت مرحله ۲' }, link: 'confirmed', bucket: 'staged' },
    { code: '62010458', customer: 'پیمان سلطانی', seller: 23, teamAtIssue: 'T2', inv: 'completed', stage: 3, stages: 3, stg: 'approved', evidence: 'online',  review: 'approved', total: 12000000, paid: 12000000, issued: '۱۴۰۵/۰۶/۲۰', next: null, link: 'confirmed', bucket: 'done' },
    { code: '44120987', customer: 'امیر رحیمی',   seller: 23, teamAtIssue: 'T2', inv: 'completed', stage: 1, stages: 1, stg: 'approved', evidence: 'online',  review: 'approved', total: 7500000,  paid: 7500000,  issued: '۱۴۰۵/۰۷/۰۱', next: null, link: 'confirmed', bucket: 'done' },
    { code: '58812390', customer: 'فاطمه جعفری',  seller: 38, teamAtIssue: 'T3', inv: 'completed', stage: 1, stages: 1, stg: 'approved', evidence: 'receipt', review: 'approved', total: 7500000,  paid: 7500000,  issued: '۱۴۰۵/۰۷/۰۳', next: null, link: 'partial', bucket: 'done', linkNote: 'اتصال به پرونده فقط از طریق شماره موبایل دیده می‌شود؛ اثبات اتصال ندارد' },
    { code: '33021774', customer: 'رضا حیدری',    seller: 40, teamAtIssue: 'T2', inv: 'pre',       stage: 1, stages: 1, stg: 'pending',  evidence: 'none',    review: 'none',     total: 5000000,  paid: 0,        issued: '۱۴۰۵/۰۶/۲۹', next: { who: 'customer', text: 'پرداخت توسط مشتری' }, link: 'confirmed', bucket: 'pre' },
    { code: '33021801', customer: 'نسرین فتحی',   seller: 40, teamAtIssue: 'T3', inv: 'staged',    stage: 1, stages: 2, stg: 'review',   evidence: 'receipt', review: 'pending',  total: 8000000,  paid: 0,        issued: '۱۴۰۵/۰۷/۰۷', next: { who: 'finance', text: 'بررسی مالی رسید' }, link: 'confirmed', bucket: 'finance' },
    { code: '28887140', customer: 'کامیار صادقی', seller: 40, teamAtIssue: null, inv: 'completed', stage: 1, stages: 1, stg: 'approved', evidence: 'online',  review: 'approved', total: 6500000,  paid: 6500000,  issued: '۱۴۰۵/۰۶/۱۵', next: null, link: 'confirmed', bucket: 'done', teamNote: 'هنگام صدور، سابقه تیم ثبت نشده است' },
    { code: '77003219', customer: 'شیرین عطایی',  seller: 46, teamAtIssue: 'T4', inv: 'mismatch',  stage: 1, stages: 1, stg: 'approved', evidence: 'receipt', review: 'approved', total: 4000000,  paid: 4000000,  issued: '۱۴۰۵/۰۷/۰۲', next: { who: 'mis', text: 'تطبیق وضعیت (MIS / مالی)' }, mismatch: 'پرداخت تأیید شده ولی وضعیت فاکتور هنوز تکمیل ثبت نشده', link: 'conflict', bucket: 'action' },
    { code: '90011827', customer: 'ماندانا ایزدی', seller: 44, teamAtIssue: 'direct', inv: 'pre',     stage: 1, stages: 1, stg: 'pending',  evidence: 'none',    review: 'none',     total: 3500000,  paid: 0,        issued: '۱۴۰۵/۰۷/۰۸', next: { who: 'customer', text: 'پرداخت توسط مشتری' }, link: 'confirmed', bucket: 'pre' }
  ],

  // Conversion cohort (assigned-case predicate source = Dot cases; rate NOT FINAL — OPD-07).
  conv: [
    { team: 'T1', converter: 'سارا احمدی (سرپرست)', assigned: 2, ready: 2, completed: 0 },
    { team: 'T2', converter: 'رضا مختاری (سرپرست)', assigned: 1, ready: 1, completed: 0 },
    { team: 'T3', converter: '—', assigned: 0, ready: 1, completed: 0 },
    { team: 'T4', converter: '—', assigned: 0, ready: 1, completed: 0 }
  ],

  // Case Explorer baseline: legacy facet + limited latest rows. Two cases share one phone on purpose (phone ≠ identity).
  cases: [
    { id: 'L-9001', src: 'legacy', phone: '09131112200', seller: 40, raw: 'جواب نداده', mapped: 'تماس انجام نشده', link: 'partial', hist: [['تخصیص به کیان رضایی', 'سارا احمدی · ۱۴۰۴/۱۱/۰۲'], ['ورود سرنخ', 'واردسازی قدیمی · ۱۴۰۴/۱۱/۰۱']] },
    { id: 'L-9002', src: 'legacy', phone: '09131112200', seller: 11, raw: 'خرید کرده', mapped: 'نیازمند تطبیق', link: 'unresolved', hist: [['وضعیت خام «خرید کرده» ثبت شد', 'نگار رحیمی · ۱۴۰۴/۱۱/۲۰']], note: '«خرید کرده» قدیمی اثبات تکمیل فروش نیست' },
    { id: 'C-30133', src: 'v4', phone: '09127774400', seller: null, team: 'T2', raw: 'در بررسی', mapped: 'در بررسی', link: 'unresolved', hist: [['اتصال فاکتور بدون کد معتبر', 'سیستم · ۱۴۰۵/۰۷/۰۵'], ['تحویل به رضا مختاری', 'کامران صدری · ۱۴۰۵/۰۷/۰۵']] },
    { id: 'C-30160', src: 'mis', phone: '09196660022', seller: 38, raw: 'تماس مجدد', mapped: 'تماس مجدد', link: 'confirmed', hist: [['تخصیص به داود امینی', 'لیلا فرهادی · ۱۴۰۵/۰۷/۰۸'], ['تحویل به لیلا فرهادی', 'کامران صدری · ۱۴۰۵/۰۷/۰۸'], ['ورود شماره', 'MIS · ۱۴۰۵/۰۶/۲۸']] }
  ],

  hr: [
    { id: 412, type: 'transfer', seller: 14, requester: 'سارا احمدی', dest: 'رضا مختاری', state: 'pending_review', reviewer: 'self', reason: 'تعادل ظرفیت دو تیم پس از مرخصی طولانی', steps: [['درخواست', 'done', 'سارا احمدی · ۱۴۰۵/۰۷/۰۹'], ['سرپرست ارشد', 'cur', 'کامران صدری — بررسی مرحله‌ای'], ['مرحله بعد طبق زنجیره', 'skip', 'پس از تأیید شما تعیین می‌شود'], ['منابع انسانی', 'skip', 'اعمال نهایی']] },
    { id: 409, type: 'termination', seller: 31, requester: 'رضا مختاری', dest: null, state: 'pending_review', reviewer: 'self', reason: 'عدم حضور بیش از حد مجاز', steps: [['درخواست', 'done', 'رضا مختاری · ۱۴۰۵/۰۷/۰۸'], ['سرپرست ارشد', 'cur', 'کامران صدری — بررسی مرحله‌ای'], ['مرحله بعد طبق زنجیره', 'skip', 'پس از تأیید شما تعیین می‌شود'], ['منابع انسانی', 'skip', 'اعمال نهایی']] },
    { id: 405, type: 'transfer', seller: 40, requester: 'رضا مختاری', dest: 'لیلا فرهادی', state: 'pending_hr', reviewer: 'hr', reason: 'نیاز تیم مقصد', steps: [['درخواست', 'done', 'رضا مختاری · ۱۴۰۵/۰۶/۲۸'], ['سرپرست ارشد', 'done', 'کامران صدری · ۱۴۰۵/۰۶/۲۹ — تأیید این مرحله'], ['منابع انسانی', 'cur', 'در انتظار اعمال نهایی']] },
    { id: 398, type: 'transfer', seller: 23, requester: 'سارا احمدی', dest: 'رضا مختاری', state: 'approved', reviewer: null, reason: 'تغییر ساختار تیم', applied: '۱۴۰۵/۰۶/۲۰', steps: [['درخواست', 'done', 'سارا احمدی · ۱۴۰۵/۰۶/۱۵'], ['سرپرست ارشد', 'done', 'کامران صدری · ۱۴۰۵/۰۶/۱۶ — تأیید این مرحله'], ['منابع انسانی', 'done', 'اعمال شد · ۱۴۰۵/۰۶/۲۰']] },
    { id: 391, type: 'termination', seller: 27, requester: 'رضا مختاری', dest: null, state: 'rejected', reviewer: null, reason: 'درخواست بدون مستندات', steps: [['درخواست', 'done', 'رضا مختاری · ۱۴۰۵/۰۶/۰۵'], ['سرپرست ارشد', 'rejected', 'کامران صدری · رد با دلیل: مستندات ناقص']] }
  ],

  wallet: { balance: 2960000, credit: 2960000, debit: 0, tx: [
    { date: '۱۴۰۵/۰۶/۱۸', desc: 'حق‌الزحمه فروش — فاکتور ۴۴۱۲۰۹۸۷', rule: 'طبق قوانین مالی فعلی', channel: 'آنلاین', amount: 1460000 },
    { date: '۱۴۰۵/۰۶/۰۷', desc: 'حق‌الزحمه فروش — فاکتور ۶۲۰۱۰۴۵۸', rule: 'طبق قوانین مالی فعلی', channel: 'کارت‌به‌کارت', amount: 1500000 }
  ] },

  // Metric reconciliation notes (F02/F03) for the advanced diagnostics drawer.
  recon: [
    { metric: 'پیش‌فاکتور باز', old: '۰', now: '۴', diff: '+۴', reason: 'تعریف قدیمی پیش‌فاکتورهای ردیف‌شده در فاکتور را نمی‌شمرد (F02)', state: 'recon' },
    { metric: 'لیدهای قدیمی', old: '۱۱ (فقط جدول قدیمی)', now: 'پوشش کامل تأیید نشده', diff: '—', reason: 'برچسب «کل لیدها» نیست؛ منابع V4 و MIS هنوز در resolver تأییدشده نیستند (F03)', state: 'partial' },
    { metric: 'نرخ تبدیل', old: '۰٪', now: 'تعریف نشده', diff: '—', reason: 'مخرج و بازه تصویب نشده (OPD-07)؛ خالی بودن به معنی شکست نیست', state: 'undef' },
    { metric: 'مدت انتظار آماده‌ها', old: '—', now: 'تعریف نشده', diff: '—', reason: 'زمان شروع انتظار معتبر تعریف نشده (SD-05)', state: 'undef' }
  ],

  reports: [
    { id: 'invoices_register', name: 'فاکتورها', grain: 'فاکتور', basis: 'رویداد در بازه (زمان صدور)', icon: 'receipt' },
    { id: 'pre_invoices', name: 'پیش‌فاکتورهای صادرشده', grain: 'فاکتور (رویداد صدور)', basis: 'رویداد در بازه', icon: 'file' },
    { id: 'sales', name: 'فروش‌های تکمیل‌شده', grain: 'فاکتور تکمیل‌شده', basis: 'رویداد تکمیل در بازه', icon: 'checkCircle' },
    { id: 'online_sales', name: 'فروش آنلاین', grain: 'پرداخت', basis: 'رویداد در بازه', icon: 'link' },
    { id: 'card_sales', name: 'فروش کارت‌به‌کارت تأییدشده', grain: 'مرحله پرداخت', basis: 'رویداد تأیید در بازه', icon: 'receipt' },
    { id: 'customer_profiles', name: 'پروفایل مشتری‌ها', grain: 'گروه شماره (نه هویت قطعی)', basis: 'وضعیت در لحظه', icon: 'users' },
    { id: 'customer_invoice_details', name: 'جزئیات فاکتور مشتری', grain: 'فاکتور', basis: 'رویداد در بازه', icon: 'file' },
    { id: 'mis_assignments', name: 'تخصیص دادهٔ MIS', grain: 'رویداد انتقال', basis: 'رویداد در بازه', icon: 'swap' },
    { id: 'finance_pending', name: 'پیش‌فاکتورهای در انتظار مالی', grain: 'مرحله پرداخت', basis: 'وضعیت در لحظه', icon: 'hourglass' },
    { id: 'finance_rejected', name: 'پیش‌فاکتورهای ردشده', grain: 'مرحله پرداخت', basis: 'رویداد رد در بازه', icon: 'alert' }
  ]
};
