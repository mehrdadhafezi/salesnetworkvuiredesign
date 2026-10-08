/* Sales Manager prototype — MOCK data only (never a source of truth).
   Shapes follow SALES-MANAGER-PRODUCT-SPEC.md + Gate 0: stable CaseRef independent of phone, five ownership concepts, namespaced statuses,
   eligibility with reasons, financial facets kept separate, metric metadata (name/grain/source/cohort/time_basis/scope/formula/freshness).
   The sample structure (Manager → 3 سرپرست ارشدs / 1 direct Supervisor / 1 direct Seller) is NOT hardcoded product logic.
   Counts are an illustrative snapshot, never a workload benchmark. */
window.MGR = {
  user: { name: 'فرهاد نیک‌زاد', id: '۱۸۰۰۳', role: 'مدیر فروش', initials: 'ف‌ن' },
  freshness: { now: '۸ دقیقه پیش', stale: '۱ ساعت و ۱۰ دقیقه پیش', evaluated: '۱۴۰۵/۰۷/۱۰ ۱۰:۴۲' },

  // Direct reports of the Manager. U3 is inactive on purpose (inactive recipient). Visibility never implies write authority.
  units: [
    { id: 'U1', name: 'کامران صدری', level: 'senior', active: true },
    { id: 'U2', name: 'فرشید منصوری', level: 'senior', active: true },
    { id: 'U3', name: 'آرش مهدوی', level: 'senior', active: false, inactiveNote: 'غیرفعال از ۱۴۰۵/۰۷/۰۳' }
  ],
  // Supervisors (teams). parent = سرپرست ارشد unit id, or 'M' = directly under the Manager (no middle level).
  teams: [
    { id: 'T1', sup: 'سارا احمدی', parent: 'U1', active: true },
    { id: 'T2', sup: 'رضا مختاری', parent: 'U1', active: true },
    { id: 'T3', sup: 'لیلا فرهادی', parent: 'U2', active: true },
    { id: 'T4', sup: 'ندا کیانی', parent: 'U2', active: false, inactiveNote: 'غیرفعال از ۱۴۰۵/۰۷/۰۴' },
    { id: 'T5', sup: 'مجتبی عباسی', parent: 'M', active: true }
  ],
  // alloc: 'ok' = write eligibility is provable for this action; 'unverified' = visible but not provably writable (M-G05).
  sellers: [
    { id: 6,  name: 'مهدی زاده',   team: 'T1', active: true,  open: 9,  legacy: 3, alloc: 'ok' },
    { id: 11, name: 'نگار رحیمی',  team: 'T1', active: true,  open: 14, legacy: 1, alloc: 'ok' },
    { id: 19, name: 'پریسا نوری',  team: 'T1', active: false, open: 4,  legacy: 2, alloc: 'ok', note: 'غیرفعال از ۱۴۰۵/۰۷/۰۵' },
    { id: 23, name: 'حمید قاسمی',  team: 'T2', active: true,  open: 17, legacy: 4, alloc: 'ok' },
    { id: 31, name: 'بهزاد کریمی', team: 'T2', active: true,  open: 8,  legacy: 5, alloc: 'ok' },
    { id: 35, name: 'مینا شریفی',  team: 'T3', active: true,  open: 11, legacy: 2, alloc: 'ok' },
    { id: 38, name: 'داود امینی',  team: 'T3', active: true,  open: 13, legacy: 1, alloc: 'ok' },
    { id: 40, name: 'کیان رضایی',  team: 'T3', active: true,  open: 7,  legacy: 0, alloc: 'unverified', moved: { from: 'T2', when: '۱۴۰۵/۰۷/۰۲' } },
    { id: 46, name: 'سمیرا طاهری', team: 'T4', active: true,  open: 5,  legacy: 1, alloc: 'ok' },
    { id: 50, name: 'الهام صفوی',  team: 'T5', active: true,  open: 12, legacy: 0, alloc: 'ok' },
    { id: 51, name: 'مرتضی دادگر', team: 'T5', active: true,  open: 6,  legacy: 2, alloc: 'ok' },
    { id: 60, name: 'پرویز نادری', team: 'direct', active: true, open: 6, legacy: 0, alloc: 'ok' }
  ],

  // Manager's own pool = delivered to this Manager, not yet allocated. Reported separately from team workload.
  pool: [
    { id: 'C-41001', source: 'MIS · دسته ۹۲۰۱', received: '۱۴۰۵/۰۷/۰۹', elig: 'ok' },
    { id: 'C-41002', source: 'MIS · دسته ۹۲۰۱', received: '۱۴۰۵/۰۷/۰۹', elig: 'ok' },
    { id: 'C-41003', source: 'MIS · دسته ۹۲۰۱', received: '۱۴۰۵/۰۷/۰۹', elig: 'ok' },
    { id: 'C-41004', source: 'MIS · دسته ۹۲۰۱', received: '۱۴۰۵/۰۷/۰۹', elig: 'ok' },
    { id: 'C-41005', source: 'MIS · دسته ۹۲۰۱', received: '۱۴۰۵/۰۷/۰۹', elig: 'ok' },
    { id: 'C-41006', source: 'MIS · دسته ۹۲۰۱', received: '۱۴۰۵/۰۷/۰۹', elig: 'ok' },
    { id: 'C-41007', source: 'MIS · دسته ۹۲۰۲', received: '۱۴۰۵/۰۷/۱۰', elig: 'ok' },
    { id: 'C-41008', source: 'MIS · دسته ۹۲۰۲', received: '۱۴۰۵/۰۷/۱۰', elig: 'ok' },
    { id: 'C-41009', source: 'MIS · دسته ۹۲۰۲', received: '۱۴۰۵/۰۷/۱۰', elig: 'ok' },
    { id: 'C-41010', source: 'MIS · دسته ۹۲۰۲', received: '۱۴۰۵/۰۷/۱۰', elig: 'ok' },
    { id: 'C-40890', source: 'برگشت از سرپرست ارشد', received: '۱۴۰۵/۰۷/۰۶', elig: 'blocked', why: 'به پیش‌فاکتور ۴۹۱۶۴۷۵۴ متصل است (پرداخت‌نشده)' },
    { id: 'C-40771', source: 'MIS · دسته ۹۱۸۸', received: '۱۴۰۵/۰۶/۳۰', elig: 'unknown', why: 'اتصال فاکتور قابل تأیید نیست؛ نیاز به تطبیق MIS' }
  ],

  // Cases currently below the Manager. proof = who executed the transfer that put it there (return needs own valid transfer/custody policy).
  held: [
    { id: 'C-40110', source: 'MIS · دسته ۹۱۹۰', holder: { kind: 'senior', unit: 'U1' }, proof: { by: 'self', op: 'OP-5302', when: '۱۴۰۵/۰۷/۰۸' }, elig: 'ok' },
    { id: 'C-40111', source: 'MIS · دسته ۹۱۹۰', holder: { kind: 'senior', unit: 'U1' }, proof: { by: 'self', op: 'OP-5302', when: '۱۴۰۵/۰۷/۰۸' }, elig: 'ok' },
    { id: 'C-40112', source: 'MIS · دسته ۹۱۹۰', holder: { kind: 'sup', team: 'T1' }, proof: { by: 'self', op: 'OP-5290', when: '۱۴۰۵/۰۷/۰۵' }, elig: 'blocked', why: 'fin', invoice: { code: '89663612', label: 'پیش‌فاکتور · پرداخت‌نشده' } },
    { id: 'C-40120', source: 'MIS · دسته ۹۱۸۵', holder: { kind: 'seller', id: 11 }, proof: { by: 'senior', op: 'تخصیص توسط کامران صدری', when: '۱۴۰۵/۰۷/۰۶' }, elig: 'blocked', why: 'notown' },
    { id: 'C-40125', source: 'MIS · دسته ۹۱۷۰', holder: { kind: 'sup', team: 'T3' }, proof: { by: 'mis', op: 'توزیع MIS', when: '۱۴۰۵/۰۷/۰۱' }, elig: 'blocked', why: 'mis' },
    { id: 'C-40130', source: 'MIS · دسته ۹۱۹۱', holder: { kind: 'senior', unit: 'U2' }, proof: { by: 'self', op: 'OP-5302', when: '۱۴۰۵/۰۷/۰۸' }, elig: 'conflict', why: 'مسئول فعلی پس از بارگذاری فهرست تغییر کرده است (انتقال توسط سرپرست ارشد به سرپرست)' },
    { id: 'C-40133', source: 'MIS · دسته ۹۱۸۸', holder: { kind: 'sup', team: 'T2' }, proof: { by: 'self', op: 'OP-5290', when: '۱۴۰۵/۰۷/۰۵' }, elig: 'unknown', why: 'اتصال یک فاکتور فاقد کد معتبر است؛ نیاز به تطبیق (MIS / مالی)', link: 'unresolved' },
    { id: 'C-40140', source: 'پیگیری فاکتور V4', holder: { kind: 'sup', team: 'T1' }, proof: { by: 'self', op: 'OP-5290', when: '۱۴۰۵/۰۷/۰۵' }, elig: 'blocked', why: 'consumed', invoice: { code: '55410376', label: 'پرداخت مرحله‌ای · مرحله ۲ از ۳' } },
    { id: 'C-40150', source: 'MIS · دسته ۹۱۷۵', holder: { kind: 'sup', team: 'T2' }, proof: { by: 'self', op: 'OP-5290', when: '۱۴۰۵/۰۷/۰۵' }, elig: 'blocked', why: 'cancel', invoice: { code: '39001276', label: 'لغو شده' } },
    { id: 'C-40160', source: 'MIS · دسته ۹۱۹۱', holder: { kind: 'seller', id: 38 }, proof: { by: 'self', op: 'OP-5302', when: '۱۴۰۵/۰۷/۰۸' }, elig: 'ok' },
    { id: 'L-9001',  source: 'سرنخ قدیمی', legacy: true, holder: { kind: 'seller', id: 40 }, proof: { by: 'none', op: 'سابقه انتقال توسط شما ثبت نشده', when: '—' }, elig: 'unknown', why: 'برگشت سرنخ قدیمی برای نقش مدیر فروش هنوز اعتبارسنجی نشده است', link: 'partial' }
  ],

  // Operations. Item states: ok (applied) · skipped (not sent: eligibility changed) · rejected (refused by system) · failed (retryable) · unknown (no confirmation).
  ops: [
    { ref: 'OP-5302', kind: 'assign', when: 'دیروز ۱۱:۲۰', actor: 'فرهاد نیک‌زاد', to: 'کامران صدری، فرشید منصوری', requested: 14,
      items: [['C-40110', 'ok', 'کامران صدری'], ['C-40111', 'ok', 'کامران صدری'], ['C-40130', 'ok', 'فرشید منصوری'], ['C-40160', 'ok', 'فرشید منصوری'], ['C-40161', 'ok', 'فرشید منصوری'], ['C-40162', 'ok', 'فرشید منصوری'], ['C-40163', 'ok', 'کامران صدری'], ['C-40164', 'ok', 'کامران صدری'], ['C-40165', 'ok', 'کامران صدری'], ['C-40166', 'ok', 'کامران صدری'], ['C-40167', 'failed', 'فرشید منصوری', 'پاسخ سرور ناقص بود؛ قابل تکرار'], ['C-40168', 'skipped', 'فرشید منصوری', 'پرونده پیش از ارسال از پنل شما برداشته شد (مسئول فعلی تغییر کرد)'], ['C-40169', 'rejected', 'فرشید منصوری', 'سامانه ثبت را نپذیرفت: وابستگی مالی فعال'], ['C-40170', 'ok', 'کامران صدری']] },
    { ref: 'OP-5298', kind: 'return', when: '۱۴۰۵/۰۷/۰۸ ۱۶:۴۰', actor: 'فرهاد نیک‌زاد', to: 'پنل من', requested: 3,
      items: [['C-40081', 'ok'], ['C-40082', 'ok'], ['C-40083', 'ok']] },
    { ref: 'OP-5290', kind: 'assign', when: '۱۴۰۵/۰۷/۰۵ ۰۹:۰۵', actor: 'فرهاد نیک‌زاد', to: 'سارا احمدی', requested: 5, note: 'پاسخ تأیید نرسید؛ ممکن است ثبت شده باشد. پیش از تکرار تطبیق کنید.',
      items: [['C-40112', 'unknown'], ['C-40140', 'unknown'], ['C-40133', 'unknown'], ['C-40150', 'unknown'], ['C-40171', 'unknown']] }
  ],

  // Invoices (financial facets stay separate). teamAtIssue: team recorded at issue time; null = unknown / not recoverable.
  invoices: [
    { code: '89663612', customer: 'بسی نظری',     seller: 6,  teamAtIssue: 'T1', inv: 'pre',       stage: 1, stages: 1, stg: 'pending',  evidence: 'none',    review: 'none',     total: 15000000, paid: 0,        issued: '۱۴۰۵/۰۷/۰۴', next: { who: 'customer', text: 'پرداخت توسط مشتری' }, link: 'confirmed', bucket: 'pre', pay: 'link' },
    { code: '49164754', customer: 'محمد زمانی',   seller: 11, teamAtIssue: 'T1', inv: 'pre',       stage: 1, stages: 1, stg: 'review',   evidence: 'receipt', review: 'pending',  total: 6000000,  paid: 0,        issued: '۱۴۰۵/۰۷/۰۶', next: { who: 'finance', text: 'بررسی مالی رسید' }, link: 'confirmed', bucket: 'finance' },
    { code: '71942055', customer: 'مریم احمدی',   seller: 11, teamAtIssue: 'T1', inv: 'staged',    stage: 2, stages: 3, stg: 'rejected', evidence: 'receipt', review: 'rejected', total: 9000000,  paid: 3000000,  issued: '۱۴۰۵/۰۶/۲۸', reason: 'مبلغ رسید با مبلغ مرحله مطابقت ندارد', next: { who: 'seller', text: 'ثبت رسید اصلاح‌شده' }, link: 'confirmed', bucket: 'rejected', pay: 'link' },
    { code: '55410376', customer: 'علی نوری',     seller: 6,  teamAtIssue: 'T1', inv: 'staged',    stage: 2, stages: 3, stg: 'pending',  evidence: 'none',    review: 'none',     total: 9000000,  paid: 3000000,  issued: '۱۴۰۵/۰۶/۲۳', next: { who: 'customer', text: 'پرداخت مرحله ۲' }, link: 'confirmed', bucket: 'staged', pay: 'link' },
    { code: '62010458', customer: 'پیمان سلطانی', seller: 23, teamAtIssue: 'T2', inv: 'completed', stage: 3, stages: 3, stg: 'approved', evidence: 'online',  review: 'approved', total: 12000000, paid: 12000000, issued: '۱۴۰۵/۰۶/۲۰', next: null, link: 'confirmed', bucket: 'done' },
    { code: '44120987', customer: 'امیر رحیمی',   seller: 23, teamAtIssue: 'T2', inv: 'completed', stage: 1, stages: 1, stg: 'approved', evidence: 'online',  review: 'approved', total: 7500000,  paid: 7500000,  issued: '۱۴۰۵/۰۷/۰۱', next: null, link: 'confirmed', bucket: 'done' },
    { code: '58812390', customer: 'فاطمه جعفری',  seller: 38, teamAtIssue: 'T3', inv: 'completed', stage: 1, stages: 1, stg: 'approved', evidence: 'receipt', review: 'approved', total: 7500000,  paid: 7500000,  issued: '۱۴۰۵/۰۷/۰۳', next: null, link: 'partial', bucket: 'done', linkNote: 'اتصال به پرونده فقط از طریق شماره موبایل دیده می‌شود؛ اثبات اتصال ندارد' },
    { code: '33021774', customer: 'رضا حیدری',    seller: 40, teamAtIssue: 'T2', inv: 'pre',       stage: 1, stages: 1, stg: 'pending',  evidence: 'none',    review: 'none',     total: 5000000,  paid: 0,        issued: '۱۴۰۵/۰۶/۲۹', next: { who: 'customer', text: 'پرداخت توسط مشتری' }, link: 'confirmed', bucket: 'pre', pay: 'link' },
    { code: '33021801', customer: 'نسرین فتحی',   seller: 40, teamAtIssue: 'T3', inv: 'staged',    stage: 1, stages: 2, stg: 'review',   evidence: 'receipt', review: 'pending',  total: 8000000,  paid: 0,        issued: '۱۴۰۵/۰۷/۰۷', next: { who: 'finance', text: 'بررسی مالی رسید' }, link: 'confirmed', bucket: 'finance' },
    { code: '28887140', customer: 'کامیار صادقی', seller: 40, teamAtIssue: null, inv: 'completed', stage: 1, stages: 1, stg: 'approved', evidence: 'online',  review: 'approved', total: 6500000,  paid: 6500000,  issued: '۱۴۰۵/۰۶/۱۵', next: null, link: 'confirmed', bucket: 'done', teamNote: 'هنگام صدور، سابقه تیم ثبت نشده است' },
    { code: '77003219', customer: 'شیرین عطایی',  seller: 46, teamAtIssue: 'T4', inv: 'mismatch',  stage: 1, stages: 1, stg: 'approved', evidence: 'receipt', review: 'approved', total: 4000000,  paid: 4000000,  issued: '۱۴۰۵/۰۷/۰۲', next: { who: 'mis', text: 'تطبیق وضعیت (MIS / مالی)' }, mismatch: 'پرداخت تأیید شده ولی وضعیت فاکتور هنوز تکمیل ثبت نشده', link: 'conflict', bucket: 'action' },
    { code: '90011827', customer: 'ماندانا ایزدی', seller: 60, teamAtIssue: 'direct', inv: 'pre',    stage: 1, stages: 1, stg: 'pending',  evidence: 'none',    review: 'none',     total: 3500000,  paid: 0,        issued: '۱۴۰۵/۰۷/۰۸', next: { who: 'customer', text: 'پرداخت توسط مشتری' }, link: 'confirmed', bucket: 'pre', pay: 'link' },
    { code: '66120044', customer: 'داریوش کمالی', seller: 50, teamAtIssue: 'T5', inv: 'staged',    stage: 1, stages: 2, stg: 'pending',  evidence: 'none',    review: 'none',     total: 10000000, paid: 0,        issued: '۱۴۰۵/۰۷/۰۹', next: { who: 'customer', text: 'پرداخت مرحله ۱' }, link: 'confirmed', bucket: 'staged', pay: 'link' }
  ],

  // Archive Explorer: five independent reasons (rules preserved, not merged). owner shown = renderer priority converter→supervisor→seller (NOT custody).
  archReasons: {
    noanswer: { label: 'بی‌پاسخ', rule: 'حداقل سه تلاش تماس و سه روز عدم فعالیت پس از تلاش سوم.', days: 3 },
    assess: { label: 'ارزیابی پرداخت‌نشده', rule: 'لینک پرداخت فعال که طی سه روز تکمیل نشده است.', days: 3 },
    sub: { label: 'اشتراک پرداخت‌نشده', rule: 'پنج روز طبق همان نقطه شروع و شرط جریان اشتراک.', days: 5 },
    product: { label: 'محصول پرداخت‌نشده', rule: 'پنج روز طبق شرط همان نوع محصول.', days: 5 },
    productx: { label: 'محصول* پرداخت‌نشده', rule: 'پنج روز در جریان مخصوص؛ نوع ذخیره‌شده حفظ می‌شود و با «محصول» یکی نیست.', days: 5 }
  },
  archives: [
    { id: 'AR-801', reason: 'noanswer', caseRef: 'L-8820', src: 'legacy', seller: 23, archived: '۱۴۰۵/۰۷/۰۶', age: 4, attempts: 3, lastAct: '۱۴۰۵/۰۷/۰۳', fin: null, owner: ['فروشنده', 'حمید قاسمی'], link: 'partial', resolve: 'none', note: 'منبع و محدوده این نوع مسیر خودش را دارد؛ مالک نمایش‌داده‌شده مالک فعلی پرونده نیست.' },
    { id: 'AR-802', reason: 'noanswer', caseRef: 'L-8831', src: 'legacy', seller: 35, archived: '۱۴۰۵/۰۷/۰۷', age: 3, attempts: 4, lastAct: '۱۴۰۵/۰۷/۰۴', fin: null, owner: ['فروشنده', 'مینا شریفی'], link: 'partial', resolve: 'none' },
    { id: 'AR-812', reason: 'assess', caseRef: 'C-30881', src: 'v4', seller: 11, archived: '۱۴۰۵/۰۷/۰۶', age: 4, due: '۱۴۰۵/۰۷/۰۳', fin: { total: 2000000, paid: 0, inv: '70011230' }, owner: ['سرپرست', 'سارا احمدی'], link: 'confirmed', resolve: 'late-payment', note: 'پرداخت دیرهنگام طبق قاعده موجود می‌تواند آن را «حل‌شده» کند؛ اجرای واقعی تأیید نشده است.' },
    { id: 'AR-824', reason: 'product', caseRef: 'C-30902', src: 'v4', seller: 38, archived: '۱۴۰۵/۰۷/۰۵', age: 5, due: '۱۴۰۵/۰۶/۳۰', fin: { total: 12000000, paid: 4000000, inv: '66098121' }, owner: ['مبدل', 'لیلا فرهادی'], link: 'partial', resolve: 'partial-pay', note: 'پرداخت جزئی به‌تنهایی احیا یا آزادسازی عمومی نیست.' }
  ],

  // Extra-number requests. dup = the check that actually runs: same Seller + normalized phone in pending/approved only.
  xreq: [
    { id: 'XR-1201', seller: 23, phone: '09121234567', name: 'خانم رحمانی', note: 'معرفی‌شده توسط مشتری قبلی', state: 'pending', at: '۱۴۰۵/۰۷/۰۹ ۱۴:۱۰', dup: 'none' },
    { id: 'XR-1202', seller: 19, phone: '09357770011', name: '', note: '', state: 'pending', at: '۱۴۰۵/۰۷/۰۸ ۰۹:۳۰', dup: 'none', flag: 'فروشنده پس از ثبت درخواست غیرفعال شده است؛ سیاست رسیدگی تعریف نشده (NEEDS VALIDATION)' },
    { id: 'XR-1195', seller: 31, phone: '09194445500', name: 'آقای مرادی', note: '', state: 'approved', at: '۱۴۰۵/۰۷/۰۴', by: 'فرهاد نیک‌زاد · ۱۴۰۵/۰۷/۰۵', dup: 'none', lead: 'L-9120' },
    { id: 'XR-1190', seller: 35, phone: '09123332211', name: 'خانم افشار', note: 'مشتری ثبت‌نام‌نشده', state: 'rejected', at: '۱۴۰۵/۰۷/۰۳', by: 'فرهاد نیک‌زاد · ۱۴۰۵/۰۷/۰۳', dup: 'none', reason: 'شماره قبلاً در لیست سرپرست دیگری بوده است' },
    { id: 'XR-1188', seller: 23, phone: '09101110099', name: '', note: '', state: 'rejected', at: '۱۴۰۵/۰۷/۰۲', by: 'فرهاد نیک‌زاد · ۱۴۰۵/۰۷/۰۲', dup: 'none', reason: '' }
  ],

  // HR requests at the Manager's workflow position. Final apply belongs to HR; failed apply stays a distinct state.
  hr: [
    { id: 512, type: 'transfer', seller: 23, requester: 'کامران صدری', dest: 'مجتبی عباسی', state: 'pending_review', reviewer: 'self', reason: 'تعادل ظرفیت پس از مرخصی طولانی', steps: [['درخواست', 'done', 'کامران صدری · ۱۴۰۵/۰۷/۰۹'], ['مدیر فروش', 'cur', 'فرهاد نیک‌زاد — بررسی مرحله‌ای'], ['مرحله بعد طبق زنجیره', 'skip', 'پس از تأیید شما تعیین می‌شود'], ['منابع انسانی', 'skip', 'اعمال نهایی']] },
    { id: 509, type: 'termination', seller: 19, requester: 'سارا احمدی', dest: null, state: 'pending_review', reviewer: 'self', reason: 'عدم حضور بیش از حد مجاز', steps: [['درخواست', 'done', 'سارا احمدی · ۱۴۰۵/۰۷/۰۸'], ['سرپرست ارشد', 'done', 'کامران صدری · ۱۴۰۵/۰۷/۰۹ — تأیید این مرحله'], ['مدیر فروش', 'cur', 'فرهاد نیک‌زاد — بررسی مرحله‌ای'], ['منابع انسانی', 'skip', 'اعمال نهایی']] },
    { id: 505, type: 'transfer', seller: 40, requester: 'رضا مختاری', dest: 'لیلا فرهادی', state: 'pending_hr', reviewer: 'hr', reason: 'نیاز تیم مقصد', steps: [['درخواست', 'done', 'رضا مختاری · ۱۴۰۵/۰۶/۲۸'], ['مدیر فروش', 'done', 'فرهاد نیک‌زاد · ۱۴۰۵/۰۶/۲۹ — تأیید این مرحله'], ['منابع انسانی', 'cur', 'در انتظار اعمال نهایی']] },
    { id: 498, type: 'transfer', seller: 31, requester: 'کامران صدری', dest: 'سارا احمدی', state: 'approved', reviewer: null, reason: 'تغییر ساختار تیم', applied: '۱۴۰۵/۰۶/۲۰', steps: [['درخواست', 'done', 'کامران صدری · ۱۴۰۵/۰۶/۱۵'], ['مدیر فروش', 'done', 'فرهاد نیک‌زاد · ۱۴۰۵/۰۶/۱۶ — تأیید این مرحله'], ['منابع انسانی', 'done', 'اعمال شد · ۱۴۰۵/۰۶/۲۰']] },
    { id: 495, type: 'transfer', seller: 35, requester: 'فرشید منصوری', dest: 'مجتبی عباسی', state: 'failed', reviewer: null, reason: 'نیاز تیم مقصد', failedAt: '۱۴۰۵/۰۶/۱۸', steps: [['درخواست', 'done', 'فرشید منصوری · ۱۴۰۵/۰۶/۱۴'], ['مدیر فروش', 'done', 'فرهاد نیک‌زاد · ۱۴۰۵/۰۶/۱۵ — تأیید این مرحله'], ['منابع انسانی', 'rejected', 'اعمال ناموفق · ۱۴۰۵/۰۶/۱۸ — تغییری اعمال نشده']] },
    { id: 491, type: 'termination', seller: 46, requester: 'فرشید منصوری', dest: null, state: 'rejected', reviewer: null, reason: 'درخواست بدون مستندات', steps: [['درخواست', 'done', 'فرشید منصوری · ۱۴۰۵/۰۶/۰۵'], ['مدیر فروش', 'rejected', 'فرهاد نیک‌زاد · رد با دلیل: مستندات ناقص']] }
  ],

  // Customer behaviour / issuance context for one scoped invoice. kind: issuance (system) vs customer (customer action) are never mixed.
  cust: {
    inv: '71942055', customer: 'مریم احمدی',
    events: [
      { t: '۱۴۰۵/۰۷/۰۵ ۱۰:۱۲', kind: 'issuance', src: 'سامانه', text: 'لینک پرداخت مرحله ۲ صادر شد' },
      { t: '۱۴۰۵/۰۷/۰۵ ۱۰:۱۲', kind: 'issuance', src: 'سامانه', text: 'پیش‌فاکتور مرحله ۲ برای مشتری ثبت شد' }
    ],
    customerEvents: []
  },

  // KPI definitions (Product Spec §8). cohort is an OPEN commercial decision → shown as «تصویب نشده», never filled in.
  kpiDefs: [
    { id: 'created', name: 'فاکتور صادرشده', grain: 'فاکتور', source: 'منبع معتبر فاکتور', cohort: 'فاکتورهای ایجادشده در بازه', time: 'رویداد · زمان ایجاد', scope: 'محدوده فاکتورهای مجاز', formula: 'تعداد شناسه یکتای فاکتور ایجادشده؛ مرحله یا پرونده نیست', fresh: 'زمان پرس‌وجو + تکمیل‌بودن', open: 'بازه الزامی است' },
    { id: 'openpre', name: 'پیش‌فاکتور باز', grain: 'فاکتور', source: 'وضعیت جاری فاکتور (فرهنگ لغت وضعیت‌ها)', cohort: 'فاکتورهای واجد شرایط فعلی', time: 'وضعیت در لحظه', scope: 'همان محدوده', formula: 'پیش‌فاکتور فعلی تکمیل‌نشده؛ رسید تأییدشده به‌تنهایی «تکمیل» نیست', fresh: 'ساعت ارزیابی', open: 'نام‌گذاری فضاهای وضعیت صریح می‌شود' },
    { id: 'issued', name: 'پیش‌فاکتور صادرشده', grain: 'رویداد صدور', source: 'سابقه صدور معتبر', cohort: 'رویداد صدور در بازه', time: 'رویداد در بازه', scope: 'انتساب همان رویداد', formula: 'کلید یکتای صدور؛ حتی اگر بعداً پرداخت شده باشد. نبود سابقه صدور ≠ ۰', fresh: 'پوشش سابقه', open: 'پوشش تاریخچه صدور' },
    { id: 'done', name: 'فروش تکمیل‌شده', grain: 'فاکتور', source: 'وصول معتبر کل فاکتور', cohort: 'رویداد تکمیل', time: 'رویداد · زمان تکمیل', scope: 'سیاست اعتبار اعلام‌شده', formula: 'وصول کل فاکتور با حفظ مبلغ و تلورانس مصوب؛ رسید/مرحله تأییدشده کافی نیست', fresh: 'زمان تکمیل', open: 'مرز تکمیل (OPD-06) و انتساب (OPD-07) باز است' },
    { id: 'cases', name: 'کل پرونده‌ها', grain: 'پرونده منطقی', source: 'اجتماع منابع معتبر با نام‌های جایگزین اثبات‌شده', cohort: 'بار فعلی یا رویداد — جدا', time: 'وضعیت در لحظه', scope: 'محدوده پرونده', formula: 'پرونده یکتا؛ ناتمام‌های اتصال «ناقص» می‌مانند و با شماره موبایل ادغام نمی‌شوند', fresh: 'تازگی هر منبع', open: 'پوشش منبع صریح است' },
    { id: 'legacy', name: 'فقط لیدهای قدیمی', grain: 'لید قدیمی', source: 'جدول سرنخ‌های قدیمی', cohort: 'گروه فقط‌قدیمی', time: 'ایجاد / تخصیص (مشخص شود)', scope: 'محدوده لید قدیمی', formula: 'شناسه یکتای لید؛ «کل پرونده» نامیده نمی‌شود', fresh: 'آخرین پرس‌وجو + پوشش', open: 'زمان مبنا را مشخص کنید' }
  ],

  // F04: same-looking metric disagreement. Cause NOT VERIFIED — hypotheses only.
  recon: {
    panel: { name: 'گزارش مدیر · شاخص فاکتور', value: 0, grain: 'فاکتور', rows: '۶۰ تا ۱۲۰ ردیف در هر درخواست' },
    tab: { name: 'برگه فاکتورها', value: 1, grain: 'پیش‌فاکتور', rows: 'فهرست جاری' },
    checks: [
      ['scope', 'محدوده', 'q', 'یکسان بودن محدوده دو نما تأیید نشده است'],
      ['grain', 'واحد شمارش', 'q', 'فاکتور در برابر پیش‌فاکتور؛ هم‌واحد بودن اثبات نشده'],
      ['time', 'مبنای زمانی', 'q', 'رویداد در بازه در برابر وضعیت فعلی'],
      ['source', 'منبع', 'q', 'V4 + قدیمی در برابر برگه فاکتورها'],
      ['filters', 'فیلترها', 'q', 'فیلتر پیش‌فرض دو نما یکسان نیست یا ثبت نشده']
    ],
    hypotheses: ['حافظه موقت (cache)', 'نسخه ساخت (build)', 'شرط شمارش (predicate)', 'گروه شمارش (cohort)'],
    label: 'نام فروشنده در ستون سرپرست یک ردیف گزارش دیده شده است؛ نگاشت انتساب نیازمند اعتبارسنجی است و خطای تاریخی قطعی اعلام نمی‌شود.'
  },

  wallet: { balance: 4210000, credit: 4210000, debit: 0, tx: [
    { date: '۱۴۰۵/۰۶/۱۸', desc: 'حق‌الزحمه فروش شخصی — فاکتور ۴۴۱۲۰۹۸۷', rule: 'طبق قوانین مالی فعلی', channel: 'آنلاین', amount: 2210000 },
    { date: '۱۴۰۵/۰۶/۰۷', desc: 'حق‌الزحمه فروش شخصی — فاکتور ۶۲۰۱۰۴۵۸', rule: 'طبق قوانین مالی فعلی', channel: 'کارت‌به‌کارت', amount: 2000000 }
  ] },

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
