/* Supervisor prototype — MOCK data only (never a source of truth). Shapes follow the live Supervisor panel (2026-10-04)
   and Gate 0 contracts: stable case IDs independent of phone, four ownership concepts, namespaced statuses,
   eligibility with reasons, financial facets kept separate, metric metadata (meaning / time basis / freshness). */
window.SUP = {
  user: { name: 'سارا احمدی', id: '۱۸۲۰۵', role: 'سرپرست', initials: 'س‌ا', parent: 'کامران صدری (سرپرست ارشد)' },
  freshness: { stats: '۸ دقیقه پیش', statsStale: '۴۷ دقیقه پیش' },

  // Direct team (operational assignment scope). Indirect subtree: none for this supervisor in the current hierarchy.
  sellers: [
    { id: 6,  name: 'مهدی زاده',    phone: '09125002233', active: true,  open: 9,  untouched: 3, due: 1, blocked: 1, openPre: 2, lastAssign: '۱۴۰۵/۰۷/۰۸' },
    { id: 11, name: 'نگار رحیمی',   phone: '09121230045', active: true,  open: 14, untouched: 6, due: 2, blocked: 0, openPre: 1, lastAssign: '۱۴۰۵/۰۷/۰۹' },
    { id: 14, name: 'علی کاظمی',    phone: '09351112233', active: true,  open: 6,  untouched: 0, due: 0, blocked: 2, openPre: 3, lastAssign: '۱۴۰۵/۰۷/۰۲' },
    { id: 19, name: 'پریسا نوری',   phone: '09129870011', active: false, inactiveNote: 'غیرفعال از ۱۴۰۵/۰۷/۰۵', open: 4, untouched: 4, due: 0, blocked: 0, openPre: 0, lastAssign: '۱۴۰۵/۰۶/۲۸' }
  ],
  indirect: [],

  // Owned balance = cases delivered to this supervisor, not yet with a seller.
  pool: [
    { id: 'C-20841', phone: '09121110001', source: 'MIS · دسته ۸۸۱۲', received: '۱۴۰۵/۰۷/۰۹', elig: 'ok' },
    { id: 'C-20842', phone: '09121110002', source: 'MIS · دسته ۸۸۱۲', received: '۱۴۰۵/۰۷/۰۹', elig: 'ok' },
    { id: 'C-20843', phone: '09121110003', source: 'MIS · دسته ۸۸۱۲', received: '۱۴۰۵/۰۷/۰۹', elig: 'ok' },
    { id: 'C-20844', phone: '09121110004', source: 'MIS · دسته ۸۸۱۲', received: '۱۴۰۵/۰۷/۰۹', elig: 'ok' },
    { id: 'C-20845', phone: '09121110005', source: 'MIS · دسته ۸۸۱۲', received: '۱۴۰۵/۰۷/۰۹', elig: 'ok' },
    { id: 'C-20846', phone: '09121110006', source: 'MIS · دسته ۸۸۱۲', received: '۱۴۰۵/۰۷/۰۹', elig: 'ok' },
    { id: 'C-20847', phone: '09121110007', source: 'MIS · دسته ۸۸۱۳', received: '۱۴۰۵/۰۷/۱۰', elig: 'ok' },
    { id: 'C-20848', phone: '09121110008', source: 'MIS · دسته ۸۸۱۳', received: '۱۴۰۵/۰۷/۱۰', elig: 'ok' },
    { id: 'C-20849', phone: '09121110009', source: 'MIS · دسته ۸۸۱۳', received: '۱۴۰۵/۰۷/۱۰', elig: 'ok' },
    { id: 'C-20850', phone: '09121110010', source: 'MIS · دسته ۸۸۱۳', received: '۱۴۰۵/۰۷/۱۰', elig: 'ok' },
    { id: 'C-20512', phone: '09128885511', source: 'برگشت از فروشنده', received: '۱۴۰۵/۰۷/۰۶', elig: 'blocked', why: 'به پیش‌فاکتور ۴۹۱۶۴۷۵۴ متصل است (پرداخت‌نشده)' },
    { id: 'C-19970', phone: '09127774400', source: 'MIS · دسته ۸۷۹۰', received: '۱۴۰۵/۰۶/۳۰', elig: 'unknown', why: 'اتصال فاکتور قابل تأیید نیست؛ نیاز به تطبیق MIS' }
  ],

  // Cases currently with direct sellers (custody) — candidates for return review.
  held: [
    { id: 'C-20310', phone: '09121234500', seller: 6,  original: 6,  source: 'MIS · دسته ۸۷۹۵', since: '۱۴۰۵/۰۷/۰۱', contact: 'بدون وضعیت', invoice: null, elig: 'ok' },
    { id: 'C-20311', phone: '09121234501', seller: 6,  original: 6,  source: 'MIS · دسته ۸۷۹۵', since: '۱۴۰۵/۰۷/۰۱', contact: 'جواب نداده (۱ از ۳)', invoice: null, elig: 'ok' },
    { id: 'C-20290', phone: '09100766169', seller: 6,  original: 6,  source: 'MIS · دسته ۸۷۹۰', since: '۱۴۰۵/۰۶/۲۹', contact: 'پیش‌فاکتور', invoice: { code: '89663612', label: 'پیش‌فاکتور · پرداخت‌نشده' }, elig: 'blocked', why: 'fin' },
    { id: 'C-20188', phone: '09359990011', seller: 11, original: 11, source: 'MIS · دسته ۸۷۸۱', since: '۱۴۰۵/۰۶/۲۵', contact: 'تماس مجدد · ۱۴۰۵/۰۷/۱۱', invoice: null, elig: 'ok' },
    { id: 'C-20140', phone: '09123336655', seller: 14, original: 14, source: 'MIS · دسته ۸۷۸۱', since: '۱۴۰۵/۰۶/۲۳', contact: 'پیش‌فاکتور', invoice: { code: '55410376', label: 'پرداخت مرحله‌ای · مرحله ۲ از ۳' }, elig: 'blocked', why: 'fin' },
    { id: 'C-20077', phone: '09196660022', seller: 14, original: 6,  source: 'انتقال از فروشنده دیگر', since: '۱۴۰۵/۰۶/۲۰', contact: 'عدم خرید', invoice: { code: '39001276', label: 'لغو شده' }, elig: 'blocked', why: 'cancel' },
    { id: 'C-20051', phone: '09124445566', seller: 19, original: 19, source: 'MIS · دسته ۸۷۷۰', since: '۱۴۰۵/۰۶/۱۸', contact: 'بدون وضعیت', invoice: null, elig: 'ok' },
    { id: 'C-20052', phone: '09124445567', seller: 19, original: 19, source: 'MIS · دسته ۸۷۷۰', since: '۱۴۰۵/۰۶/۱۸', contact: 'بدون وضعیت', invoice: null, elig: 'conflict', why: 'وضعیت پس از بارگذاری فهرست تغییر کرده است (تماس جدید ثبت شد)' },
    { id: 'C-19830', phone: '09127778899', seller: 11, original: 11, source: 'MIS · دسته ۸۷۵۵', since: '۱۴۰۵/۰۶/۱۰', contact: 'در بررسی', invoice: { code: '—', label: 'اتصال نامشخص' }, elig: 'unknown', why: 'یک اتصال فاکتور بدون کد معتبر ثبت شده است' },
    { id: 'L-7731',  phone: '09131112200', seller: 11, original: 11, source: 'سرنخ قدیمی', legacy: true, since: '۱۴۰۴/۱۱/۰۲', contact: 'جواب نداده', invoice: null, elig: 'ok' },
    { id: 'L-7732',  phone: '09131112201', seller: 6,  original: 6,  source: 'سرنخ قدیمی', legacy: true, since: '۱۴۰۴/۱۱/۰۲', contact: 'خرید کرده (قدیمی)', invoice: null, elig: 'unknown', why: '«خرید کرده» قدیمی، اثبات تکمیل فروش نیست؛ نیاز به بررسی' }
  ],

  ops: [
    { ref: 'OP-2291', kind: 'assign', when: 'امروز ۱۰:۱۲', actor: 'سارا احمدی', total: 10, ok: 8, failed: 1, conflict: 1, unknown: 0, to: 'نگار رحیمی، مهدی زاده',
      items: [['C-20801', 'ok', 'نگار رحیمی'], ['C-20802', 'ok', 'نگار رحیمی'], ['C-20803', 'ok', 'نگار رحیمی'], ['C-20804', 'ok', 'نگار رحیمی'], ['C-20805', 'ok', 'مهدی زاده'], ['C-20806', 'ok', 'مهدی زاده'], ['C-20807', 'ok', 'مهدی زاده'], ['C-20808', 'ok', 'مهدی زاده'], ['C-20809', 'retry', 'مهدی زاده', 'پاسخ سرور دریافت نشد؛ تخصیص انجام نشد و قابل تکرار است'], ['C-20810', 'conflict', 'مهدی زاده', 'این پرونده پیش از ثبت توسط MIS بازگردانده شده بود']] },
    { ref: 'OP-2275', kind: 'return', when: 'دیروز ۱۶:۴۰', actor: 'سارا احمدی', total: 3, ok: 3, failed: 0, conflict: 0, unknown: 0, to: 'پنل من', items: [['C-20290', 'ok'], ['C-20291', 'ok'], ['C-20292', 'ok']] },
    { ref: 'OP-2268', kind: 'assign', when: '۱۴۰۵/۰۷/۰۸ ۰۹:۰۵', actor: 'سارا احمدی', total: 5, ok: 0, failed: 0, conflict: 0, unknown: 5, to: 'علی کاظمی', items: [['C-20701', 'unknown'], ['C-20702', 'unknown'], ['C-20703', 'unknown'], ['C-20704', 'unknown'], ['C-20705', 'unknown']], note: 'زمان پاسخ تمام شد؛ ممکن است ثبت شده باشد. پیش از تکرار، با شماره عملیات تطبیق داده می‌شود.' }
  ],

  // Ready conversion: validation invoices whose payment completed AND passed financial review.
  ready: [
    { id: 'R-5521', customer: 'محمد زمانی', phone: '09128903454', credit: 2000000, choice: 'اشتراک طلایی', status: 'ready', converter: null, invoice: '49164754' },
    { id: 'R-5524', customer: 'مریم احمدی', phone: '09351112222', credit: 2000000, choice: null, status: 'ready', converter: null, invoice: '71942055' },
    { id: 'R-5530', customer: 'امیر رحیمی', phone: '09195587237', credit: 1500000, choice: 'اشتراک نقره‌ای', status: 'ready', converter: null, invoice: '44120987' },
    { id: 'R-5509', customer: 'فاطمه جعفری', phone: '09360001122', credit: 2000000, choice: 'اشتراک ۱۰ ستاره', status: 'assigned', converter: 'self', invoice: '58812390' }
  ],
  converters: [],
  myConversions: [
    { id: 'R-5509', customer: 'فاطمه جعفری', phone: '09360001122', option: 'اشتراک ۱۰ ستاره', remaining: 10000000, stage: 'تماس انجام نشده', due: 'امروز ۱۶:۰۰' }
  ],

  // Invoices in scope — each facet is separate (Gate 0: invoice ≠ payment stage ≠ evidence ≠ finance review ≠ completed sale).
  invoices: [
    { code: '89663612', seller: 6,  customer: 'بسی',        phone: '09100766169', product: 'اعتبارسنجی ۱۵ هزار', total: 15000, paid: 0, stages: 1, stage: 1, inv: 'pre',      stg: 'pending',  evidence: 'none', review: 'none',     next: { who: 'customer', text: 'پرداخت مشتری' }, sub: { name: 'اعتبارسنجی', locked: false }, issued: '۱۴۰۵/۰۷/۰۸', bucket: 'pre' },
    { code: '71942055', seller: 11, customer: 'مریم احمدی',  phone: '09351112222', product: 'اشتراک برنزی',        total: 18500000, paid: 0, stages: 2, stage: 1, inv: 'staged', stg: 'rejected', evidence: 'receipt', review: 'rejected', reason: 'فیش خوانا نیست', next: { who: 'seller', text: 'اصلاح فیش مرحله ۱' }, sub: { name: 'اشتراک برنزی', locked: true }, issued: '۱۴۰۵/۰۷/۰۳', bucket: 'action' },
    { code: '62830417', seller: 14, customer: 'حمید صادقی',  phone: '09127654320', product: 'اشتراک طلایی',        total: 125000000, paid: 0, stages: 1, stage: 1, inv: 'pre',  stg: 'review',   evidence: 'receipt', review: 'pending', next: { who: 'finance', text: 'بررسی رسید' }, sub: { name: 'اشتراک طلایی', locked: true }, issued: '۱۴۰۵/۰۷/۰۵', bucket: 'review' },
    { code: '55410376', seller: 14, customer: 'نگار حسینی',  phone: '09123456781', product: 'اشتراک نقره‌ای',      total: 48000000, paid: 32000000, stages: 3, stage: 3, inv: 'staged', stg: 'pending', evidence: 'none', review: 'none', next: { who: 'customer', text: 'پرداخت مرحله ۳' }, sub: { name: 'اشتراک نقره‌ای', locked: true }, issued: '۱۴۰۵/۰۶/۲۲', bucket: 'staged' },
    { code: '44120987', seller: 6,  customer: 'امیر رحیمی',  phone: '09195587237', product: 'اشتراک نقره‌ای + پکیج', total: 64000000, paid: 64000000, stages: 2, stage: 2, inv: 'completed', stg: 'approved', evidence: 'receipt', review: 'approved', next: null, sub: { name: 'اشتراک نقره‌ای', locked: true }, issued: '۱۴۰۵/۰۶/۱۵', bucket: 'done' },
    { code: '49164754', seller: 11, customer: 'محمد زمانی',  phone: '09128903454', product: 'اشتراک ۱۰ ستاره',     total: 10000000, paid: 0, stages: 1, stage: 1, inv: 'pre', stg: 'pending', evidence: 'none', review: 'none', next: { who: 'seller', text: 'ارسال لینک پرداخت' }, sub: { name: 'اشتراک ۱۰ ستاره', locked: false }, issued: '۱۴۰۵/۰۷/۰۹', bucket: 'action' },
    { code: '39001276', seller: 14, customer: 'علی نوری',    phone: '09129999847', product: 'اعتبارسنجی ۲۵۰ هزار', total: 250000, paid: 0, stages: 1, stage: 1, inv: 'cancelled', stg: 'cancelled', evidence: 'none', review: 'none', next: null, sub: { name: 'اعتبارسنجی', locked: false }, issued: '۱۴۰۵/۰۶/۲۰', bucket: 'other' },
    { code: '58812390', seller: 11, customer: 'فاطمه جعفری', phone: '09360001122', product: 'اشتراک ۱۰ ستاره',     total: 10000000, paid: 10000000, stages: 1, stage: 1, inv: 'mismatch', stg: 'approved', evidence: 'online', review: 'approved', next: { who: 'mis', text: 'تطبیق مغایرت ثبت' }, sub: { name: 'اشتراک ۱۰ ستاره', locked: true }, issued: '۱۴۰۵/۰۶/۲۸', bucket: 'action', mismatch: 'مبلغ ثبت‌شده در درگاه با فاکتور هم‌خوان نیست' }
  ],

  customerEvents: {
    '09100766169': [['باز کردن لینک فاکتور', 'امروز ۰۹:۲۰'], ['مشاهده اطلاعات محصول', 'امروز ۰۹:۲۱'], ['ارسال پیامک پیش‌فاکتور', '۱۴۰۵/۰۷/۰۸']],
    '09351112222': [['ارسال فیش مرحله ۱', '۱۴۰۵/۰۷/۰۴'], ['رد فیش توسط مالی', '۱۴۰۵/۰۷/۰۵'], ['باز کردن لینک فاکتور', '۱۴۰۵/۰۷/۰۳']],
    '09128903454': [['انتخاب روش پرداخت آنلاین', 'دیروز'], ['باز کردن لینک فاکتور', 'دیروز']]
  },

  hr: [
    { id: 41, type: 'transfer', seller: 19, dest: 'رضا مختاری (سرپرست)', reason: 'درخواست خود فروشنده برای جابجایی تیم', state: 'pending_hr', steps: [['ثبت درخواست', 'done', 'سارا احمدی · ۱۴۰۵/۰۷/۰۵'], ['بررسی مرحله‌ای', 'done', 'کامران صدری · ۱۴۰۵/۰۷/۰۶'], ['اعمال نهایی توسط منابع انسانی', 'cur', 'در انتظار']] },
    { id: 38, type: 'termination', seller: 14, dest: null, reason: 'عدم حضور مکرر', state: 'rejected', steps: [['ثبت درخواست', 'done', 'سارا احمدی · ۱۴۰۵/۰۶/۲۰'], ['بررسی مرحله‌ای', 'rejected', 'کامران صدری · «ابتدا جلسه بازخورد برگزار شود»'], ['اعمال نهایی توسط منابع انسانی', 'skip', '—']] }
  ],

  wallet: { balance: 1840000, credit: 1840000, debit: 0, tx: [
    { date: '۱۴۰۵/۰۶/۱۸ ۱۱:۰۲', channel: 'کارت‌به‌کارت', desc: 'حق سرپرستی فاکتور ۴۴۱۲۰۹۸۷', rule: 'قانون مالی #۹ · سهم سرپرست', amount: 640000 },
    { date: '۱۴۰۵/۰۶/۱۰ ۰۹:۴۰', channel: 'آنلاین', desc: 'حق سرپرستی فاکتور ۵۸۸۱۲۳۹۰', rule: 'قانون مالی #۹ · سهم سرپرست', amount: 1200000 }
  ] },

  reports: [
    { id: 'invoices_register', name: 'فاکتورها', grain: 'فاکتور', basis: 'رویداد در بازه (زمان صدور)', icon: 'receipt' },
    { id: 'customer_profiles', name: 'پروفایل مشتری‌ها', grain: 'گروه شماره (نه هویت قطعی)', basis: 'وضعیت در لحظه', icon: 'users' },
    { id: 'customer_invoice_details', name: 'جزئیات فاکتور مشتری', grain: 'فاکتور', basis: 'رویداد در بازه', icon: 'file' },
    { id: 'mis_assignments', name: 'تخصیص دادهٔ MIS', grain: 'رویداد انتقال', basis: 'رویداد در بازه', icon: 'swap' },
    { id: 'pre_invoices', name: 'پیش‌فاکتورهای صادرشده', grain: 'فاکتور (رویداد صدور)', basis: 'رویداد در بازه', icon: 'file' },
    { id: 'sales', name: 'فروش‌های تکمیل‌شده', grain: 'فاکتور تکمیل‌شده', basis: 'رویداد تکمیل در بازه', icon: 'checkCircle' },
    { id: 'online_sales', name: 'فروش آنلاین', grain: 'پرداخت', basis: 'رویداد در بازه', icon: 'link' },
    { id: 'card_sales', name: 'فروش کارت‌به‌کارت تأییدشده', grain: 'مرحله پرداخت', basis: 'رویداد تأیید در بازه', icon: 'receipt' },
    { id: 'finance_pending', name: 'پیش‌فاکتورهای در انتظار مالی', grain: 'مرحله پرداخت', basis: 'وضعیت در لحظه', icon: 'hourglass' },
    { id: 'finance_rejected', name: 'پیش‌فاکتورهای ردشده', grain: 'مرحله پرداخت', basis: 'رویداد رد در بازه', icon: 'alert' }
  ]
};
