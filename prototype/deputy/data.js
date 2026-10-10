/* Sales Deputy prototype — MOCK data only (never a source of truth).
   Shapes follow SALES-DEPUTY-PRODUCT-SPEC.md + Gate 0: stable CaseRef independent of phone, five ownership concepts, namespaced statuses,
   eligibility with reasons, financial facets kept separate, metric metadata (name/grain/source/cohort/time_basis/scope/formula/freshness).
   The sample structure (Deputy → 3 Managers → Seniors / Supervisors / Sellers) is NOT hardcoded product logic, and a multi-Manager
   runtime is NOT VERIFIED (spec §7). Counts are an illustrative snapshot, never a workload benchmark. */
window.DEP = {
  user: { name: 'علیرضا شکوهی', id: '۱۸۰۰۱', role: 'معاون فروش', initials: 'ع‌ش' },
  freshness: { now: '۶ دقیقه پیش', stale: '۱ ساعت و ۲۰ دقیقه پیش', evaluated: '۱۴۰۵/۰۷/۱۰ ۱۰:۴۸' },

  // Branches the Deputy is scoped to (NOT the entire organisation). Visibility never implies write authority.
  managers: [
    { id: 'MG1', name: 'فرهاد نیک‌زاد', active: true },
    { id: 'MG2', name: 'سعید پارسا', active: true },
    { id: 'MG3', name: 'مهناز فتحی', active: true }
  ],
  seniors: [
    { id: 'U1', name: 'کامران صدری', mgr: 'MG1', active: true },
    { id: 'U2', name: 'فرشید منصوری', mgr: 'MG1', active: true },
    { id: 'U3', name: 'آرش مهدوی', mgr: 'MG1', active: false, inactiveNote: 'غیرفعال از ۱۴۰۵/۰۷/۰۳' },
    { id: 'U4', name: 'بهنام رستمی', mgr: 'MG2', active: true },
    { id: 'U5', name: 'نوید عزیزی', mgr: 'MG3', active: true }
  ],
  // parent = Senior id ('U..') or Manager id ('MG..' = directly under the Manager, no middle level).
  teams: [
    { id: 'T1', sup: 'سارا احمدی', parent: 'U1', active: true },
    { id: 'T2', sup: 'رضا مختاری', parent: 'U1', active: true },
    { id: 'T3', sup: 'لیلا فرهادی', parent: 'U2', active: true },
    { id: 'T4', sup: 'ندا کیانی', parent: 'U2', active: false, inactiveNote: 'غیرفعال از ۱۴۰۵/۰۷/۰۴' },
    { id: 'T6', sup: 'مریم جلالی', parent: 'U4', active: true },
    { id: 'T7', sup: 'امید صالحی', parent: 'U4', active: true },
    { id: 'T8', sup: 'حسین کاظمی', parent: 'MG2', active: true },
    { id: 'T9', sup: 'شیدا مرادی', parent: 'U5', active: true }
  ],
  // team = Supervisor id, or 'dMGx' = seller directly under that Manager. alloc: 'ok' = write eligibility provable; 'unverified' = visible, not provably writable.
  sellers: [
    { id: 6,  name: 'مهدی زاده',    team: 'T1', active: true,  open: 9,  legacy: 3, alloc: 'ok' },
    { id: 11, name: 'نگار رحیمی',   team: 'T1', active: true,  open: 14, legacy: 1, alloc: 'ok' },
    { id: 19, name: 'پریسا نوری',   team: 'T1', active: false, open: 4,  legacy: 2, alloc: 'ok' },
    { id: 23, name: 'حمید قاسمی',   team: 'T2', active: true,  open: 17, legacy: 4, alloc: 'ok' },
    { id: 31, name: 'بهزاد کریمی',  team: 'T2', active: true,  open: 8,  legacy: 5, alloc: 'ok' },
    { id: 35, name: 'مینا شریفی',   team: 'T3', active: true,  open: 11, legacy: 2, alloc: 'ok' },
    { id: 38, name: 'داود امینی',   team: 'T3', active: true,  open: 13, legacy: 1, alloc: 'ok' },
    { id: 40, name: 'کیان رضایی',   team: 'T3', active: true,  open: 7,  legacy: 0, alloc: 'unverified', moved: { from: 'T2', when: '۱۴۰۵/۰۷/۰۲' } },
    { id: 46, name: 'سمیرا طاهری',  team: 'T4', active: true,  open: 5,  legacy: 1, alloc: 'ok' },
    { id: 71, name: 'محسن ایزدی',   team: 'T6', active: true,  open: 10, legacy: 2, alloc: 'ok' },
    { id: 72, name: 'الناز بهرامی', team: 'T6', active: true,  open: 12, legacy: 1, alloc: 'ok' },
    { id: 73, name: 'رامین سلیمانی', team: 'T7', active: true, open: 9,  legacy: 0, alloc: 'ok' },
    { id: 74, name: 'کاوه رحمانی',  team: 'T8', active: true,  open: 8,  legacy: 3, alloc: 'ok' },
    { id: 75, name: 'ترانه یزدانی', team: 'T8', active: true,  open: 6,  legacy: 1, alloc: 'ok' },
    { id: 76, name: 'پیمان حیدری',  team: 'dMG2', active: true, open: 5, legacy: 0, alloc: 'ok' },
    { id: 81, name: 'شهرزاد کمالی', team: 'T9', active: true,  open: 7,  legacy: 2, alloc: 'ok' },
    { id: 82, name: 'بابک نصیری',   team: 'T9', active: true,  open: 6,  legacy: 1, alloc: 'ok' }
  ],

  // Deputy's OWN custody pool (a technical owned-pool path exists; receipt is not proven by a zero/non-zero balance).
  pool: [
    { id: 'C-51001', source: 'MIS · دسته ۹۳۰۱', received: '۱۴۰۵/۰۷/۰۹', elig: 'ok' },
    { id: 'C-51002', source: 'MIS · دسته ۹۳۰۱', received: '۱۴۰۵/۰۷/۰۹', elig: 'ok' },
    { id: 'C-51003', source: 'MIS · دسته ۹۳۰۱', received: '۱۴۰۵/۰۷/۰۹', elig: 'ok' },
    { id: 'C-51004', source: 'MIS · دسته ۹۳۰۱', received: '۱۴۰۵/۰۷/۰۹', elig: 'ok' },
    { id: 'C-51005', source: 'MIS · دسته ۹۳۰۲', received: '۱۴۰۵/۰۷/۱۰', elig: 'ok' },
    { id: 'C-51006', source: 'MIS · دسته ۹۳۰۲', received: '۱۴۰۵/۰۷/۱۰', elig: 'ok' },
    { id: 'C-50890', source: 'برگشت از مدیر', received: '۱۴۰۵/۰۷/۰۶', elig: 'blocked', why: 'به پیش‌فاکتور ۴۹۱۶۴۷۵۴ متصل است (پرداخت‌نشده)' },
    { id: 'C-50771', source: 'MIS · دسته ۹۲۸۸', received: '۱۴۰۵/۰۶/۳۰', elig: 'unknown', why: 'اتصال فاکتور قابل تأیید نیست؛ نیاز به تطبیق MIS' }
  ],

  // Cases below the Deputy. proof = who executed the transfer that put it there. A Deputy rank is NOT a recall right for every descendant.
  held: [
    { id: 'C-50110', source: 'MIS · دسته ۹۲۹۰', holder: { kind: 'manager', mgr: 'MG1' }, proof: { by: 'self', op: 'OP-6102', when: '۱۴۰۵/۰۷/۰۸' }, elig: 'ok' },
    { id: 'C-50111', source: 'MIS · دسته ۹۲۹۰', holder: { kind: 'manager', mgr: 'MG2' }, proof: { by: 'self', op: 'OP-6102', when: '۱۴۰۵/۰۷/۰۸' }, elig: 'ok' },
    { id: 'C-50112', source: 'MIS · دسته ۹۲۹۰', holder: { kind: 'senior', unit: 'U4' }, proof: { by: 'self', op: 'OP-6090', when: '۱۴۰۵/۰۷/۰۵' }, elig: 'blocked', why: 'fin', invoice: { code: '27719086', label: 'پیش‌فاکتور · در بررسی مالی' } },
    { id: 'C-50120', source: 'MIS · دسته ۹۲۸۵', holder: { kind: 'sup', team: 'T6' }, proof: { by: 'manager', op: 'تخصیص توسط سعید پارسا', when: '۱۴۰۵/۰۷/۰۶' }, elig: 'blocked', why: 'notown' },
    { id: 'C-50125', source: 'MIS · دسته ۹۲۷۰', holder: { kind: 'manager', mgr: 'MG3' }, proof: { by: 'mis', op: 'توزیع MIS', when: '۱۴۰۵/۰۷/۰۱' }, elig: 'blocked', why: 'mis' },
    { id: 'C-50130', source: 'MIS · دسته ۹۲۹۱', holder: { kind: 'manager', mgr: 'MG1' }, proof: { by: 'self', op: 'OP-6102', when: '۱۴۰۵/۰۷/۰۸' }, elig: 'conflict', why: 'مسئول فعلی پس از بارگذاری فهرست تغییر کرد (از مدیر به سرپرست ارشد)' },
    { id: 'C-50133', source: 'MIS · دسته ۹۲۸۸', holder: { kind: 'sup', team: 'T2' }, proof: { by: 'self', op: 'OP-6090', when: '۱۴۰۵/۰۷/۰۵' }, elig: 'unknown', link: 'partial', why: 'اتصال یک فاکتور فاقد کد معتبر است؛ نیاز به تطبیق' },
    { id: 'C-50140', source: 'پیگیری فاکتور V4', holder: { kind: 'sup', team: 'T1' }, proof: { by: 'self', op: 'OP-6090', when: '۱۴۰۵/۰۷/۰۵' }, elig: 'blocked', why: 'consumed', invoice: { code: '55410376', label: 'پرداخت مرحله‌ای · در جریان' } },
    { id: 'C-50150', source: 'MIS · دسته ۹۲۷۵', holder: { kind: 'sup', team: 'T7' }, proof: { by: 'self', op: 'OP-6090', when: '۱۴۰۵/۰۷/۰۵' }, elig: 'blocked', why: 'cancel', invoice: { code: '39001276', label: 'لغو شده' } },
    { id: 'L-9001', source: 'سرنخ قدیمی', legacy: true, holder: { kind: 'seller', id: 82 }, proof: { by: 'none', op: 'سابقه انتقال توسط شما ثبت نشده', when: '—' }, elig: 'unknown', link: 'unresolved', why: 'برگشت سرنخ قدیمی بر پایه سابقه انتقال اثبات‌پذیر نیست' }
  ],

  // Operations. Item states: ok · skipped · rejected · failed · unknown.
  ops: [
    { ref: 'OP-6102', kind: 'assign', when: 'دیروز ۱۱:۲۰', actor: 'علیرضا شکوهی', to: 'فرهاد نیک‌زاد، سعید پارسا', requested: 6,
      items: [['C-50110', 'ok', 'فرهاد نیک‌زاد'], ['C-50111', 'ok', 'سعید پارسا'], ['C-50130', 'ok', 'فرهاد نیک‌زاد'], ['C-50125', 'ok', 'مهناز فتحی'], ['C-50126', 'ok', 'مهناز فتحی'], ['C-50127', 'ok', 'مهناز فتحی']] },
    { ref: 'OP-6098', kind: 'return', when: '۱۴۰۵/۰۷/۰۸ ۱۶:۴۰', actor: 'علیرضا شکوهی', to: 'پنل من', requested: 3,
      items: [['C-50081', 'ok'], ['C-50082', 'ok'], ['C-50083', 'ok']] },
    { ref: 'OP-6090', kind: 'assign', when: '۱۴۰۵/۰۷/۰۵ ۰۹:۰۵', actor: 'علیرضا شکوهی', to: 'کامران صدری (عبور از مدیر)', requested: 5, note: 'پاسخ تأیید نرسید؛ ممکن است ثبت شده باشد. پیش از تکرار تطبیق کنید.',
      items: [['C-50112', 'unknown'], ['C-50140', 'unknown'], ['C-50133', 'unknown'], ['C-50150', 'unknown'], ['C-50171', 'unknown']] }
  ],

  // Invoices (financial facets stay separate). teamAtIssue: Supervisor team recorded at issue time; null = unknown / not recoverable.
  invoices: [
    { code: '89663612', customer: 'بسی نظری',      seller: 6,  teamAtIssue: 'T1', inv: 'pre',       stage: 1, stages: 1, stg: 'pending',  evidence: 'none',    review: 'none',    total: 15000000, paid: 0,        issued: '۱۴۰۵/۰۷/۰۴', next: { who: 'seller', text: 'پیگیری پرداخت مشتری' }, link: 'confirmed', case: 'CaseRef C-50201 · پیگیری فاکتور V4' },
    { code: '49164754', customer: 'محمد زمانی',    seller: 11, teamAtIssue: 'T1', inv: 'pre',       stage: 1, stages: 1, stg: 'review',   evidence: 'receipt', review: 'pending', total: 6000000,  paid: 0,        issued: '۱۴۰۵/۰۷/۰۶', next: { who: 'finance', text: 'بررسی رسید ثبت‌شده' }, link: 'confirmed', case: 'CaseRef C-50202 · MIS' },
    { code: '71942055', customer: 'مریم احمدی',    seller: 11, teamAtIssue: 'T1', inv: 'staged',    stage: 2, stages: 3, stg: 'rejected', evidence: 'receipt', review: 'rejected', total: 9000000, paid: 3000000,  issued: '۱۴۰۵/۰۶/۲۸', next: { who: 'seller', text: 'اصلاح و ارسال مجدد رسید مرحله ۲' }, reason: 'رسید خوانا نیست (رد مالی)', link: 'confirmed', case: 'CaseRef C-50203 · MIS' },
    { code: '55410376', customer: 'علی نوری',      seller: 6,  teamAtIssue: 'T1', inv: 'staged',    stage: 2, stages: 3, stg: 'pending',  evidence: 'none',    review: 'none',    total: 9000000,  paid: 3000000,  issued: '۱۴۰۵/۰۶/۲۳', next: { who: 'customer', text: 'پرداخت مرحله ۲' }, link: 'confirmed', case: 'CaseRef C-50140 · پیگیری فاکتور V4' },
    { code: '62010458', customer: 'پیمان سلطانی',  seller: 23, teamAtIssue: 'T2', inv: 'completed', stage: 3, stages: 3, stg: 'approved', evidence: 'online',  review: 'approved', total: 12000000, paid: 12000000, issued: '۱۴۰۵/۰۶/۲۰', next: null, link: 'confirmed', case: 'CaseRef C-50204 · MIS' },
    { code: '44120987', customer: 'امیر رحیمی',    seller: 23, teamAtIssue: 'T2', inv: 'completed', stage: 1, stages: 1, stg: 'approved', evidence: 'online',  review: 'approved', total: 7500000,  paid: 7500000,  issued: '۱۴۰۵/۰۷/۰۱', next: null, link: 'confirmed', case: 'CaseRef C-50205 · MIS' },
    { code: '58812390', customer: 'فاطمه جعفری',   seller: 38, teamAtIssue: 'T3', inv: 'completed', stage: 1, stages: 1, stg: 'approved', evidence: 'receipt', review: 'approved', total: 7500000,  paid: 7500000,  issued: '۱۴۰۵/۰۷/۰۳', next: null, link: 'partial', case: 'CaseRef C-50206 · سرنخ قدیمی' },
    { code: '33021774', customer: 'رضا حیدری',     seller: 40, teamAtIssue: 'T2', inv: 'pre',       stage: 1, stages: 1, stg: 'pending',  evidence: 'none',    review: 'none',    total: 5000000,  paid: 0,        issued: '۱۴۰۵/۰۶/۲۹', next: { who: 'seller', text: 'پیگیری پرداخت مشتری' }, link: 'confirmed', case: 'CaseRef C-50207 · MIS' },
    { code: '27719086', customer: 'زهرا کاظمی',    seller: 71, teamAtIssue: 'T6', inv: 'pre',       stage: 1, stages: 1, stg: 'review',   evidence: 'receipt', review: 'pending', total: 8000000,  paid: 0,        issued: '۱۴۰۵/۰۷/۰۷', next: { who: 'finance', text: 'بررسی رسید ثبت‌شده' }, link: 'confirmed', case: 'CaseRef C-50112 · MIS' },
    { code: '18805532', customer: 'سامان برزگر',   seller: 73, teamAtIssue: null, teamNote: 'سابقه تیم هنگام صدور ثبت نشده است', inv: 'staged', stage: 1, stages: 2, stg: 'approved', evidence: 'online', review: 'approved', total: 10000000, paid: 5000000, issued: '۱۴۰۵/۰۶/۱۸', next: { who: 'customer', text: 'پرداخت مرحله ۲' }, link: 'confirmed', case: 'CaseRef C-50208 · MIS' },
    { code: '90431275', customer: 'نیلوفر رضوی',   seller: 74, teamAtIssue: 'T8', inv: 'completed', stage: 1, stages: 1, stg: 'approved', evidence: 'online',  review: 'approved', total: 6500000,  paid: 6500000,  issued: '۱۴۰۵/۰۷/۰۲', next: null, link: 'confirmed', case: 'CaseRef C-50209 · MIS' },
    { code: '66120843', customer: 'کامران یوسفی',  seller: 76, teamAtIssue: 'dMG2', inv: 'mismatch', stage: 1, stages: 1, stg: 'review', evidence: 'online', review: 'none', total: 9500000, paid: 9500000, issued: '۱۴۰۵/۰۷/۰۵', next: { who: 'mis', text: 'تطبیق منبع پرداخت و وضعیت فاکتور' }, mismatch: 'پرداخت آنلاین ثبت شده اما وضعیت فاکتور پرداخت‌نشده است', link: 'conflict', case: 'CaseRef نامشخص' },
    { code: '31250974', customer: 'ستاره فرجی',    seller: 81, teamAtIssue: 'T9', inv: 'pre',       stage: 1, stages: 1, stg: 'pending',  evidence: 'none',    review: 'none',    total: 4500000,  paid: 0,        issued: '۱۴۰۵/۰۷/۰۸', next: { who: 'seller', text: 'پیگیری پرداخت مشتری' }, link: 'confirmed', case: 'CaseRef C-50210 · MIS' },
    { code: '82204519', customer: 'هادی بهبودی',   seller: 82, teamAtIssue: 'T9', inv: 'completed', stage: 1, stages: 1, stg: 'approved', evidence: 'receipt', review: 'approved', total: 5500000,  paid: 5500000,  issued: '۱۴۰۵/۰۷/۰۱', next: null, link: 'unresolved', case: 'CaseRef نامشخص (منبع MIS)' }
  ],

  // Owner-aware exceptions (spec §10). owner null = UNKNOWN (rank of the Deputy is never a default owner).
  // cond: only shown under the matching demo state. No escalation / ACK / SLA command exists (SD-G07).
  excClasses: {
    mgrop:    { label: 'مسئله عملیاتی حل‌نشده مدیر', icon: 'inbox' },
    cap:      { label: 'ظرفیت / بار بین مدیران', icon: 'users' },
    distconf: { label: 'تعارض توزیع', icon: 'split' },
    stale:    { label: 'شاخص قدیمی یا ناقص', icon: 'clock' },
    f04:      { label: 'نیاز به تطبیق F04', icon: 'swap' },
    inactive: { label: 'مسئول سلسله‌مراتب غیرفعال', icon: 'user' },
    consumed: { label: 'تعارض برگشت / مصرف‌شده', icon: 'lock' },
    lineage:  { label: 'منشأ پرونده نامشخص', icon: 'unlink' },
    invoice:  { label: 'فاکتور نیازمند اقدام فروش', icon: 'receipt' },
    hr:       { label: 'مسیر HR در انتظار', icon: 'briefcase' },
    scope:    { label: 'تعارض محدوده سازمانی', icon: 'shield' }
  },
  exc: [
    { id: 'E1', cls: 'mgrop', mgr: 'MG1', subject: 'نتیجه عملیات OP-6090 نامعلوم است (تخصیص ۵ پرونده)', owner: 'فرهاد نیک‌زاد (مدیر) · عامل عملیات: شما', state: 'نتیجه نامعلوم', since: '۱۴۰۵/۰۷/۰۵',
      why: 'پاسخ تأیید نرسید و ممکن است ثبت شده باشد. تا تطبیق، تکرار کورکورانه مجاز نیست.', may: ['دیدن نتیجه هر مورد و تطبیق با شماره عملیات', 'هماهنگی با مدیر در مسیر موجود'], mayNot: ['تکرار بی‌تطبیق', 'تصاحب صف مدیر'], next: 'شما (تطبیق) سپس مدیر شاخه', signal: 'نتیجه به‌تفکیک هر پرونده (اعمال‌شده / ناموفق)', go: 'alloc', aq: 'ops', open: 'result:OP-6090' },
    { id: 'E2', cls: 'cap', mgr: null, subject: 'بار پرونده نزد شاخه‌ها یکسان نیست: فرهاد نیک‌زاد ۱۰۷ · سعید پارسا ۵۰ · مهناز فتحی ۱۳ (فقط مقایسه واقعیت)', owner: 'مدیران شاخه‌ها', state: 'نیازمند بررسی هماهنگی', since: null,
      why: 'تعداد فروشنده فعال فقط نماینده‌ای از ظرفیت است، نه ظرفیت کامل؛ هدف یا امتیازی تعریف نشده است.', may: ['مقایسه بار و فروشندگان واجد شرایط هر شاخه', 'طرح موضوع با مدیران در مسیر موجود'], mayNot: ['تعیین هدف یا امتیاز', 'انتقال پرونده خارج از اختیار خود'], next: 'مدیران شاخه‌ها', signal: 'بار و برنامه مجاز معلوم است؛ SLA وجود ندارد', go: 'perf', pm: 'case' },
    { id: 'E3', cls: 'distconf', mgr: 'MG1', subject: 'C-50130 پس از بارگذاری فهرست تغییر کرد (مسئول فعلی عوض شد)', owner: 'مسئول فعلی پرونده (Custody)', state: 'تعارض · بازخوانی لازم', since: '۱۴۰۵/۰۷/۱۰',
      why: 'گیرنده یا مسئول فعلی بین پیش‌نمایش و ثبت تغییر کرده است؛ بازنویسی انجام نمی‌شود.', may: ['بازخوانی محدوده، مسئول فعلی و گیرنده', 'اقدام مشروط فقط روی موجودی خودتان'], mayNot: ['بازنویسی موجودی مدیر', 'اعمال بدون بررسی مجدد'], next: 'مسئول فعلی مجاز پرونده', signal: 'نتیجه معتبر یا تعارض صریح', go: 'alloc', aq: 'return', rf: 'review' },
    { id: 'E4', cls: 'stale', mgr: 'MG2', subject: 'داده شاخه سعید پارسا قدیمی است', owner: 'مسئول گزارش', state: 'نیازمند بازخوانی', since: null, cond: 'stale',
      why: 'آخرین داده معتبر ' + '۱ ساعت و ۲۰ دقیقه پیش است؛ جمع کل شامل داده قدیمی آن شاخه است.', may: ['دیدن پوشش، بازه و تازگی', 'رفتن به رکوردهای معتبر'], mayNot: ['تبدیل ناقص به صفر', 'رتبه‌بندی روی داده ناقص'], next: 'مسئول گزارش موجود', signal: 'پوشش و تازگی اعلام‌شده', go: 'perf', pm: 'mgr' },
    { id: 'E5', cls: 'stale', mgr: 'MG3', subject: 'دریافت داده شاخه مهناز فتحی ناموفق بود', owner: 'مسئول گزارش / سرویس سلسله‌مراتب', state: 'دریافت نشد · قابل تکرار', since: null, cond: 'branchfail',
      why: 'یک شاخه دریافت نشد؛ موفقیت بقیه شاخه‌ها پاک نمی‌شود و درخت کامل اعلام نمی‌شود.', may: ['تلاش مجدد برای همان شاخه', 'استفاده از شاخه‌های دریافت‌شده با برچسب ناقص'], mayNot: ['صفر فرض‌کردن آمار شاخه', 'اعلام جمع نهایی'], next: 'مسئول گزارش موجود', signal: 'دریافت موفق یا خطای صریح', go: 'ov' },
    { id: 'E6', cls: 'stale', mgr: null, subject: 'پیش‌فاکتور باز: نمای کلی ۰ در برابر ردیف‌های پیش‌فاکتور (F02)؛ کل پرونده‌ها: فقط منبع قدیمی شمرده شده (F03)', owner: 'مالک داده / گزارش', state: 'نیازمند تطبیق · تعریف نهایی نشده', since: null,
      why: 'شرط شمارش نمای کلی «پیش‌فاکتور» را نمی‌شمارد و صفر صادرشدن به معنی نبود ردیف نیست؛ ۰ لید قدیمی به معنی ۰ پرونده نیست.', may: ['مقایسه محدوده و شرط شمارش در عیب‌یابی', 'دیدن پوشش هر منبع'], mayNot: ['اصلاح عدد برای هماهنگی ظاهری', 'اعلام F02/F03 رفع‌شده'], next: 'مالک داده / گزارش موجود', signal: 'تطبیق هم‌واحد یا تعریف تصویب‌شده', go: 'diag', dq: 'f02' },
    { id: 'E7', cls: 'f04', mgr: 'MG2', subject: 'مغایرت گزارش مدیر (۰) و برگه فاکتورها (۱) دیده شده است؛ در معاون امروز بازتولید نشده', owner: 'مالک داده / مالک گزارش', state: 'نیازمند تطبیق · علت اثبات‌نشده', since: null,
      why: 'خطر ناهمخوانی سیستمی باقی است؛ علت حافظه موقت، ساخت، شرط یا گروه شمارش اثبات نشده.', may: ['تطبیق مشاهده‌ای هم‌گروه (محدوده، واحد، زمان، منبع)'], mayNot: ['اصلاح مبلغ/وضعیت/اتصال برای تطبیق'], next: 'مالکان داده و گزارش', signal: 'برابری یا توضیح معتبر جمعیت‌ها', go: 'diag', dq: 'f04' },
    { id: 'E8', cls: 'inactive', mgr: 'MG1', subject: 'سرپرست ارشد آرش مهدوی و سرپرست ندا کیانی غیرفعال‌اند', owner: 'منابع انسانی (مالک ساختار و کاربر)', state: 'مسئول جایگزین ثبت نشده', since: '۱۴۰۵/۰۷/۰۳',
      why: 'گیرنده غیرفعال انتخاب‌شدنی نیست و زیرمجموعه ممکن است بدون مسئول بماند.', may: ['دیدن محدودیت واجد شرایط بودن', 'انتخاب گیرنده فعال دیگر', 'هماهنگی با منابع انسانی'], mayNot: ['فعال‌کردن کاربر', 'ویرایش کاربر یا ساختار'], next: 'منابع انسانی مجاز', signal: 'وضعیت معتبر یا گیرنده جایگزین', go: 'perf', pm: 'senior', scope: 'MG1' },
    { id: 'E9', cls: 'consumed', mgr: 'MG1', subject: 'C-50140 مصرف‌شده در مسیر فاکتور V4 (فاکتور ۵۵۴۱۰۳۷۶)', owner: 'فروش / مالی مرتبط', state: 'مسدود', since: '۱۴۰۵/۰۷/۰۵',
      why: 'برگشت عادی مجاز نیست؛ پاک‌کردن اتصال تعارض را حل نمی‌کند. لغو/رد به‌تنهایی آزادسازی نیست (OPD-03).', may: ['دیدن دلیل مسدودی', 'حفظ پرونده در محل فعلی'], mayNot: ['بازپس‌گیری فقط به اتکای رتبه', 'پاک‌کردن اتصال فاکتور'], next: 'مسئول فروش یا مالی مرتبط', signal: 'وضعیت محافظت‌شده روشن؛ لزوماً برگشت نیست', go: 'alloc', aq: 'return', rf: 'blocked', open: 'ret:C-50140' },
    { id: 'E10', cls: 'lineage', mgr: 'MG3', subject: 'فاکتور ۸۲۲۰۴۵۱۹ و ۶۶۱۲۰۸۴۳: اتصال پرونده نامشخص/متعارض', owner: null, state: 'مسئول نامشخص (UNKNOWN)', since: null,
      why: 'مسئول منبع هنوز مشخص نیست؛ رتبه معاون مالک پیش‌فرض نیست و اتصال با شماره موبایل حدس زده نمی‌شود.', may: ['بررسی مرجع‌ها و ناقص‌بودن در عیب‌یابی', 'توقف تصمیم وابسته'], mayNot: ['ادغام با شماره موبایل', 'اصلاح خام MIS', 'اتصال مجدد فاکتور'], next: 'مالک داده مجاز (نامشخص)', signal: 'اثبات قابل ردیابی؛ مسیر ارجاع جدید معوق است (SD-G07)', go: 'diag', dq: 'lineage' },
    { id: 'E11', cls: 'invoice', mgr: 'MG1', subject: 'فاکتور ۷۱۹۴۲۰۵۵ رد مالی شد (نگار رحیمی)', owner: 'فروشنده: نگار رحیمی', state: 'در انتظار اقدام فروش', since: '۱۴۰۵/۰۷/۰۸',
      why: 'رسید رد شده است؛ مرحله بعد با فروشنده مسئول است. نتیجه مالی به فروش برمی‌گردد.', may: ['دیدن دلیل، مرحله و اقدام‌کننده بعدی'], mayNot: ['تأیید/رد/بازپرداخت مالی', 'کمک‌ابزار ارسال لینک بدون مجوز مشخص'], next: 'فروشنده مسئول', signal: 'ارسال مجدد معتبر یا وضعیت نهایی مالی', go: 'inv', iq: 'action', open: 'inv:71942055' },
    { id: 'E12', cls: 'hr', mgr: 'MG1', subject: 'درخواست HR #612 در انتظار بررسی شماست؛ #598 در انتظار منابع انسانی', owner: 'شما (بررسی‌کننده فعلی) ← مرحله بعد ← منابع انسانی', state: 'در انتظار بررسی مرحله‌ای',
      since: '۱۴۰۵/۰۷/۰۹', why: 'فقط درخواست‌هایی که واقعاً به شما ارجاع شده‌اند قابل بررسی‌اند؛ تأیید شما اعمال نهایی نیست.', may: ['تأیید/رد مرحله خودتان', 'مشاهده مسیر سایر درخواست‌ها'], mayNot: ['عبور از زنجیره', 'اعمال نهایی', 'تغییر دسترسی/نقش'], next: 'بررسی‌کننده بعدی، سپس منابع انسانی', signal: 'گذار ثبت‌شده یا اعمال ناموفق مشخص', go: 'hr' },
    { id: 'E13', cls: 'scope', mgr: null, subject: 'پیش‌نمایش تخصیص به مدیر «خارج از محدوده مجاز» (مدیر شاخه مجاور) متوقف شد', owner: 'مالک سرویس محدوده / اقدام', state: 'محدوده ثابت نشده', since: null,
      why: 'گیرنده بیرون از محدوده همین اقدام است؛ دیدن نام او مجوز نوشتن نیست و محدوده کل سازمان فرض نمی‌شود.', may: ['تشخیص محدوده و توقف اقدام نامعتبر'], mayNot: ['تخصیص آزاد بین دو مدیر', 'خروجی/تخصیص بی‌محدوده'], next: 'مالک سرویس / actor مجاز', signal: 'محدوده ثابت یا رد روشن', go: 'alloc', aq: 'assign' }
  ],

  // Archive / extra-number leadership aggregates (CONDITIONAL READ). null = unavailable / not authorized — never zero.
  archAgg: [
    { mgr: 'MG1', noanswer: 6, assess: 2, sub: 1, product: 3, productx: 1 },
    { mgr: 'MG2', noanswer: 4, assess: 1, sub: 0, product: 2, productx: 0 },
    { mgr: 'MG3', noanswer: null, assess: null, sub: null, product: null, productx: null }
  ],
  xnumAgg: [
    { mgr: 'MG1', pending: 3, decided: 7 },
    { mgr: 'MG2', pending: null, decided: null },
    { mgr: 'MG3', pending: 1, decided: 2 }
  ],
  archReasons: {
    noanswer: { label: 'بی‌پاسخ', rule: 'سه تلاش بی‌پاسخ و سه روز' },
    assess: { label: 'ارزیابی پرداخت‌نشده', rule: 'سه روز' },
    sub: { label: 'اشتراک پرداخت‌نشده', rule: 'پنج روز' },
    product: { label: 'محصول پرداخت‌نشده', rule: 'پنج روز' },
    productx: { label: 'محصول* پرداخت‌نشده', rule: 'پنج روز' }
  },

  // HR requests. reviewer: 'self' (Deputy is the CURRENT reviewer) | 'manager' | 'hr'. Rank alone never assigns a review.
  hr: [
    { id: 612, type: 'transfer', seller: 38, dest: 'رضا مختاری', state: 'pending_review', reviewer: 'self', requester: 'لیلا فرهادی', reason: 'تغییر ساختار تیم و تعادل بار',
      steps: [['درخواست‌دهنده', 'done', 'لیلا فرهادی · ثبت'], ['سرپرست ارشد', 'done', 'فرشید منصوری · تأیید مرحله'], ['مدیر فروش', 'done', 'فرهاد نیک‌زاد · تأیید مرحله'], ['معاون فروش (شما)', 'cur', 'بررسی‌کننده فعلی'], ['منابع انسانی', 'skip', 'اعمال نهایی']] },
    { id: 609, type: 'termination', seller: 19, state: 'pending_review', reviewer: 'self', requester: 'سارا احمدی', reason: 'عدم فعالیت طولانی',
      steps: [['درخواست‌دهنده', 'done', 'سارا احمدی · ثبت'], ['سرپرست ارشد', 'done', 'کامران صدری · تأیید مرحله'], ['مدیر فروش', 'done', 'فرهاد نیک‌زاد · تأیید مرحله'], ['معاون فروش (شما)', 'cur', 'بررسی‌کننده فعلی'], ['منابع انسانی', 'skip', 'اعمال نهایی']] },
    { id: 605, type: 'transfer', seller: 72, dest: 'امید صالحی', state: 'pending_review', reviewer: 'manager', requester: 'مریم جلالی', reason: 'نزدیکی به مشتریان منطقه',
      steps: [['درخواست‌دهنده', 'done', 'مریم جلالی · ثبت'], ['سرپرست ارشد', 'done', 'بهنام رستمی · تأیید مرحله'], ['مدیر فروش', 'cur', 'سعید پارسا · بررسی‌کننده فعلی'], ['معاون فروش', 'skip', 'مرحله بعد (اگر در زنجیره باشد)'], ['منابع انسانی', 'skip', 'اعمال نهایی']] },
    { id: 598, type: 'transfer', seller: 31, dest: 'سارا احمدی', state: 'pending_hr', reviewer: 'hr', requester: 'رضا مختاری', reason: 'درخواست خود فروشنده',
      steps: [['درخواست‌دهنده', 'done', 'رضا مختاری · ثبت'], ['سرپرست ارشد', 'done', 'کامران صدری · تأیید مرحله'], ['مدیر فروش', 'done', 'فرهاد نیک‌زاد · تأیید مرحله'], ['معاون فروش (شما)', 'done', 'تأیید این مرحله'], ['منابع انسانی', 'cur', 'در انتظار اعمال نهایی']] },
    { id: 590, type: 'transfer', seller: 46, dest: 'لیلا فرهادی', state: 'failed', reviewer: 'hr', requester: 'ندا کیانی', reason: 'غیرفعال شدن سرپرست قبلی', failedAt: 'سرپرست مقصد در زمان اعمال فعال نبود',
      steps: [['درخواست‌دهنده', 'done', 'ندا کیانی · ثبت'], ['سرپرست ارشد', 'done', 'فرشید منصوری · تأیید مرحله'], ['مدیر فروش', 'done', 'فرهاد نیک‌زاد · تأیید مرحله'], ['معاون فروش (شما)', 'done', 'تأیید این مرحله'], ['منابع انسانی', 'rejected', 'اعمال ناموفق']] },
    { id: 584, type: 'transfer', seller: 35, dest: 'سارا احمدی', state: 'approved', reviewer: 'hr', requester: 'لیلا فرهادی', reason: 'تعادل بار', applied: '۱۴۰۵/۰۶/۲۸',
      steps: [['درخواست‌دهنده', 'done', 'لیلا فرهادی · ثبت'], ['سرپرست ارشد', 'done', 'فرشید منصوری · تأیید مرحله'], ['مدیر فروش', 'done', 'فرهاد نیک‌زاد · تأیید مرحله'], ['معاون فروش (شما)', 'done', 'تأیید این مرحله'], ['منابع انسانی', 'done', 'اعمال‌شده ۱۴۰۵/۰۶/۲۸']] },
    { id: 579, type: 'termination', seller: 75, state: 'rejected', reviewer: 'hr', requester: 'حسین کاظمی', reason: 'گزارش عملکرد',
      steps: [['درخواست‌دهنده', 'done', 'حسین کاظمی · ثبت'], ['مدیر فروش', 'rejected', 'سعید پارسا · رد با دلیل']] }
  ],

  kpiDefs: [
    { id: 'created', name: 'فاکتور صادرشده', grain: 'فاکتور', source: 'منبع معتبر فاکتور', cohort: 'فاکتورهای ایجادشده در بازه', time: 'رویداد · زمان ایجاد', scope: 'فروشندگان و فاکتورهای مجاز شاخه‌ها', formula: 'تعداد شناسه یکتای فاکتور ایجادشده؛ مرحله یا پرونده نیست', fresh: 'زمان پرس‌وجو + تکمیل‌بودن', open: 'بازه الزامی است' },
    { id: 'openpre', name: 'پیش‌فاکتور باز', grain: 'فاکتور', source: 'وضعیت جاری فاکتور (فرهنگ لغت وضعیت‌ها)', cohort: 'فاکتورهای واجد شرایط فعلی', time: 'وضعیت در لحظه', scope: 'همان محدوده', formula: 'پیش‌فاکتور فعلی تکمیل‌نشده؛ رسید تأییدشده به‌تنهایی «تکمیل» نیست', fresh: 'ساعت ارزیابی', open: 'شرط شمارش F02 / نام‌گذاری فضاهای وضعیت' },
    { id: 'issued', name: 'پیش‌فاکتور صادرشده', grain: 'رویداد صدور', source: 'سابقه صدور معتبر', cohort: 'رویداد صدور در بازه', time: 'رویداد در بازه', scope: 'انتساب همان رویداد', formula: 'کلید یکتای صدور؛ حتی اگر بعداً پرداخت شده باشد. نبود سابقه صدور ≠ ۰', fresh: 'پوشش سابقه', open: 'پوشش تاریخچه صدور' },
    { id: 'done', name: 'فروش تکمیل‌شده', grain: 'فاکتور', source: 'وصول معتبر کل فاکتور', cohort: 'رویداد تکمیل', time: 'رویداد · زمان تکمیل', scope: 'سیاست اعتبار اعلام‌شده', formula: 'وصول کل فاکتور با حفظ مبلغ و تلورانس مصوب؛ رسید/مرحله تأییدشده کافی نیست', fresh: 'زمان تکمیل', open: 'مرز تکمیل (OPD-06) و انتساب (OPD-07) باز است' },
    { id: 'cases', name: 'کل پرونده‌ها', grain: 'پرونده منطقی', source: 'اجتماع منابع معتبر با نام‌های جایگزین اثبات‌شده', cohort: 'بار فعلی یا رویداد — جدا', time: 'وضعیت در لحظه', scope: 'محدوده پرونده', formula: 'پرونده یکتا؛ ناتمام‌های اتصال «ناقص» می‌مانند و با شماره موبایل ادغام نمی‌شوند', fresh: 'تازگی هر منبع', open: 'پوشش منبع F03 صریح است' },
    { id: 'legacy', name: 'فقط لیدهای قدیمی', grain: 'لید قدیمی', source: 'جدول سرنخ‌های قدیمی', cohort: 'گروه فقط‌قدیمی', time: 'ایجاد / تخصیص (مشخص شود)', scope: 'محدوده لید قدیمی', formula: 'شناسه یکتای لید؛ «کل پرونده» نامیده نمی‌شود', fresh: 'آخرین پرس‌وجو + پوشش', open: 'زمان مبنا را مشخص کنید' }
  ],

  // Source coverage for Total Cases (F03): which sources are actually counted today. Unknown ≠ zero.
  sources: [
    { id: 'legacy', label: 'سرنخ قدیمی', state: 'counted', note: 'تنها منبعی که در شمارش پایه استفاده می‌شود' },
    { id: 'v4', label: 'پیگیری فاکتور V4', state: 'notcounted', note: 'در شمارش پایه نیست؛ بازبینی امروز انجام نشده' },
    { id: 'mis', label: 'MIS', state: 'unknown', note: 'پوشش اثبات نشده' },
    { id: 'dot', label: 'Dot', state: 'unknown', note: 'پوشش اثبات نشده' }
  ],
  f02: { tile: 0, note: 'شرط شمارش نمای کلی فقط وضعیت‌های pending / draft / unpaid / خالی را می‌شمارد و «pre_invoice» ندارد (شواهد کد + زنده).' },
  f04: {
    panel: { name: 'گزارش مدیر · شاخص فاکتور', value: 0, grain: 'فاکتور', rows: '۶۰ تا ۱۲۰ ردیف در هر درخواست' },
    tab: { name: 'برگه فاکتورها', value: 1, grain: 'پیش‌فاکتور', rows: 'فهرست جاری' },
    checks: [
      ['scope', 'محدوده', 'q', 'یکسان بودن محدوده دو نما تأیید نشده است'],
      ['grain', 'واحد شمارش', 'q', 'فاکتور در برابر پیش‌فاکتور؛ هم‌واحد بودن اثبات نشده'],
      ['time', 'مبنای زمانی', 'q', 'رویداد در بازه در برابر وضعیت فعلی'],
      ['source', 'منبع', 'q', 'V4 + قدیمی در برابر برگه فاکتورها'],
      ['filters', 'فیلترها', 'q', 'فیلتر پیش‌فرض دو نما یکسان نیست یا ثبت نشده']
    ],
    hypotheses: ['حافظه موقت (cache)', 'نسخه ساخت (build)', 'شرط شمارش (predicate)', 'گروه شمارش (cohort)']
  },

  wallet: { balance: 5640000, credit: 5640000, debit: 0, tx: [
    { date: '۱۴۰۵/۰۷/۰۹', amount: 2400000, channel: 'آنلاین', desc: 'حق‌الزحمه فاکتور ۶۲۰۱۰۴۵۸', rule: 'طبق قاعده فعلی حساب شخصی' },
    { date: '۱۴۰۵/۰۷/۰۵', amount: 1640000, channel: 'رسید', desc: 'حق‌الزحمه فاکتور ۴۴۱۲۰۹۸۷', rule: 'طبق قاعده فعلی حساب شخصی' },
    { date: '۱۴۰۵/۰۶/۲۸', amount: 1600000, channel: 'آنلاین', desc: 'حق‌الزحمه فاکتور ۵۸۸۱۲۳۹۰', rule: 'طبق قاعده فعلی حساب شخصی' }
  ] },

  reports: [
    { id: 'perf_by_mgr', name: 'عملکرد شاخه‌ها (مدیر ← فروشنده)', grain: 'فاکتور / فروشنده', basis: 'رویداد در بازه', icon: 'chart' },
    { id: 'invoices_register', name: 'ثبت فاکتورها در محدوده', grain: 'فاکتور', basis: 'رویداد صدور', icon: 'receipt' },
    { id: 'case_load', name: 'بار پرونده و پوشش منابع', grain: 'پرونده منطقی', basis: 'وضعیت در لحظه', icon: 'layers' },
    { id: 'hr_chain', name: 'مسیر درخواست‌های HR در محدوده', grain: 'درخواست', basis: 'وضعیت در لحظه', icon: 'briefcase' }
  ]
};
