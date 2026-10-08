/* Mock data only — fictional names and numbers. Nothing here comes from or goes to the live CRM. */
window.MOCK = {
  user: { name: 'مهدی زاده', id: '۲۳۱۰۴', role: 'فروشنده', initials: 'م‌ز' },

  // Seller stage-one flow. Keys mirror the real flow (no_answer max 3, callback needs date+time, not_purchased needs a reason).
  leadStatuses: {
    none:          { label: 'بدون وضعیت', tone: 'neutral' },
    no_answer:     { label: 'جواب نداده', tone: 'amber' },
    callback:      { label: 'تماس مجدد',  tone: 'blue' },
    not_purchased: { label: 'عدم خرید',   tone: 'red' },
    duplicate:     { label: 'تکراری',     tone: 'slate' },
    invoiced:      { label: 'پیش‌فاکتور', tone: 'violet' }
  },

  leads: [
    { id: 9043349, name: 'امیر رحیمی',     phone: '09195587237', city: 'تهران',  province: 'تهران',   source: 'MIS',        assigned: 'امروز ۲۲:۵۴',  assignedAgo: 'امروز',       status: 'none',          attempts: 0, prob: '۱۰۰٪',  note: '', next: null },
    { id: 9043348, name: 'سارا محمدی',     phone: '09194141619', city: 'کرج',    province: 'البرز',   source: 'MIS',        assigned: '۱۴۰۵/۰۵/۱۹',   assignedAgo: '۲ ماه پیش',   status: 'none',          attempts: 0, prob: '',      note: '', next: null },
    { id: 9043347, name: 'رضا کریمی',      phone: '09194932289', city: 'قم',     province: 'قم',      source: 'MIS',        assigned: '۱۴۰۵/۰۵/۱۹',   assignedAgo: '۲ ماه پیش',   status: 'none',          attempts: 0, prob: '',      note: '', next: null },
    { id: 9043346, name: 'نگار حسینی',     phone: '09123456781', city: 'اصفهان', province: 'اصفهان',  source: 'درخواستی',   assigned: '۱۴۰۵/۰۷/۰۵',   assignedAgo: '۳ روز پیش',   status: 'no_answer',     attempts: 2, prob: 'متوسط', note: 'دو بار زنگ خورد، جواب نداد', next: { kind: 'attempts', text: 'بی‌پاسخ ۲ از ۳', warn: true } },
    { id: 9043345, name: 'حمید صادقی',     phone: '09127654320', city: 'شیراز',  province: 'فارس',    source: 'MIS',        assigned: '۱۴۰۵/۰۷/۰۲',   assignedAgo: '۶ روز پیش',   status: 'callback',      attempts: 0, prob: 'بالا',  note: 'بعد از ساعت ۱۰ صبح تماس بگیرید', next: { kind: 'callback', text: 'فردا ۱۰:۳۰', warn: false } },
    { id: 9043344, name: 'مریم احمدی',     phone: '09351112233', city: 'مشهد',   province: 'خراسان رضوی', source: 'ارتباط مجدد', assigned: '۱۴۰۵/۰۶/۲۸', assignedAgo: '۱۰ روز پیش', status: 'callback',      attempts: 0, prob: 'متوسط', note: '', next: { kind: 'callback', text: 'امروز ۱۶:۰۰', warn: true } },
    { id: 9043343, name: 'علی نوری',       phone: '09129998877', city: 'تبریز',  province: 'آذربایجان شرقی', source: 'MIS', assigned: '۱۴۰۵/۰۶/۲۰', assignedAgo: '۱۸ روز پیش', status: 'no_answer',     attempts: 3, prob: 'ضعیف',  note: '', next: { kind: 'archive', text: 'بایگانی تا ۲ روز دیگر', warn: true } },
    { id: 9043342, name: 'فاطمه جعفری',    phone: '09360001122', city: 'رشت',    province: 'گیلان',   source: 'MIS',        assigned: '۱۴۰۵/۰۶/۱۵',   assignedAgo: '۲۳ روز پیش',  status: 'invoiced',      attempts: 0, prob: '۱۰۰٪',  note: 'اشتراک ۱۰ ستاره', next: { kind: 'invoice', text: 'منتظر پرداخت', warn: false } },
    { id: 9043341, name: 'محسن قاسمی',     phone: '09105554433', city: 'اهواز',  province: 'خوزستان', source: 'MIS',        assigned: '۱۴۰۵/۰۶/۱۰',   assignedAgo: '۲۸ روز پیش',  status: 'not_purchased', attempts: 0, prob: 'ضعیف',  note: 'قیمت را بالا دانست', next: null },
    { id: 9043340, name: 'زهرا موسوی',     phone: '09191230000', city: 'تهران',  province: 'تهران',   source: 'MIS',        assigned: '۱۴۰۵/۰۶/۰۸',   assignedAgo: '۱ ماه پیش',   status: 'duplicate',     attempts: 0, prob: '',      note: 'قبلاً توسط همکار ثبت شده', next: null }
  ],

  invoices: [
    { code: '49164754', customer: 'محمد زمانی',  phone: '09128903454', product: 'اشتراک ۱۰ ستاره',       total: 10000000,  paid: 0,         stage: 1, stages: 1, status: 'pre_invoice',  statusLabel: 'پیش‌فاکتور',        tone: 'violet', next: 'send_link' },
    { code: '66920523', customer: 'تست کتری',    phone: '09926696954', product: 'اعتبارسنجی ۲۵۰ هزار',   total: 250000,    paid: 250000,    stage: 1, stages: 1, status: 'approved',     statusLabel: 'تایید شده مالی',    tone: 'green',  next: null, conversion: 'هنوز فروخته نشده' },
    { code: '55410376', customer: 'نگار حسینی',  phone: '09123456781', product: 'اشتراک نقره‌ای',         total: 48000000,  paid: 10000000,  stage: 2, stages: 3, status: 'staged',       statusLabel: 'مرحله‌ای',          tone: 'blue',   next: 'register_payment' },
    { code: '62830417', customer: 'حمید صادقی',  phone: '09127654320', product: 'اشتراک طلایی',           total: 125000000, paid: 0,         stage: 1, stages: 1, status: 'finance_review', statusLabel: 'در بررسی مالی',   tone: 'amber',  next: null },
    { code: '71942055', customer: 'مریم احمدی',  phone: '09351112233', product: 'اشتراک برنزی',           total: 18500000,  paid: 5000000,   stage: 1, stages: 2, status: 'rejected',     statusLabel: 'رد شده',            tone: 'red',    next: 'resend_finance', rejectReason: 'فیش خوانا نیست' },
    { code: '58812390', customer: 'فاطمه جعفری', phone: '09360001122', product: 'اشتراک ۱۰ ستاره',        total: 10000000,  paid: 10000000,  stage: 1, stages: 1, status: 'approved',     statusLabel: 'تایید شده مالی',    tone: 'green',  next: null },
    { code: '44120987', customer: 'امیر رحیمی',  phone: '09195587237', product: 'اشتراک نقره‌ای + پکیج', total: 64000000,  paid: 32000000,  stage: 2, stages: 2, status: 'staged',       statusLabel: 'مرحله‌ای',          tone: 'blue',   next: 'register_payment' },
    { code: '39001276', customer: 'علی نوری',    phone: '09129998877', product: 'اعتبارسنجی ۲۵۰ هزار',   total: 250000,    paid: 0,         stage: 1, stages: 1, status: 'cancelled',    statusLabel: 'لغو شده',           tone: 'slate',  next: null }
  ],

  conversions: [
    { id: 104, invoice: '66920523', customer: 'تست کتری',   phone: '09926696954', option: 'اشتراک نقره‌ای', remaining: 10000000, stage: 'تماس انجام شد', result: 'مشتری تأیید کرد', tone: 'green' },
    { id: 97,  invoice: '58812390', customer: 'فاطمه جعفری', phone: '09360001122', option: 'اشتراک طلایی',  remaining: 42000000, stage: 'لینک پرداخت ارسال شد', result: 'آماده ارسال لینک', tone: 'blue' },
    { id: 91,  invoice: '44120987', customer: 'امیر رحیمی',  phone: '09195587237', option: '—',            remaining: 0,        stage: 'تخصیص به تبدیل‌کننده', result: 'جدید', tone: 'neutral' }
  ],

  customerActions: [
    { phone: '09128903454', name: 'محمد زمانی',  invoice: '49164754', last: 'باز کردن لینک فاکتور', when: '۱۰ دقیقه پیش', count: 7,  hot: true },
    { phone: '09123456781', name: 'نگار حسینی',  invoice: '55410376', last: 'کلیک روی گردونه',       when: '۱ ساعت پیش',   count: 4,  hot: true },
    { phone: '09195587237', name: 'امیر رحیمی',  invoice: '44120987', last: 'پرداخت مرحله ۲',         when: 'دیروز',        count: 12, hot: false },
    { phone: '09351112233', name: 'مریم احمدی',  invoice: '71942055', last: 'مشاهده اطلاعات محصول',   when: '۳ روز پیش',    count: 3,  hot: false },
    { phone: '09360001122', name: 'فاطمه جعفری', invoice: '58812390', last: 'پرداخت موفق',            when: '۵ روز پیش',    count: 9,  hot: false }
  ],

  wallet: {
    balance: 3128910, credit: 3128910, debit: 0,
    tx: [
      { date: '۱۴۰۵/۰۶/۱۷ ۱۴:۰۱', type: 'credit', channel: 'کارت به کارت', amount: 7500,    desc: 'پورسانت فاکتور ۶۶۹۲۰۵۲۳', rule: 'قانون مالی #۶' },
      { date: '۱۴۰۵/۰۶/۱۰ ۱۶:۰۹', type: 'credit', channel: 'کارت به کارت', amount: 7500,    desc: 'پورسانت فاکتور ۶۷۹۶۶۴۰۲', rule: 'قانون مالی #۶' },
      { date: '۱۴۰۵/۰۶/۰۵ ۲۲:۲۵', type: 'credit', channel: 'آنلاین',      amount: 1130000, desc: 'پورسانت فاکتور ۵۵۶۱۰۳۷۶', rule: 'قانون مالی #۴' },
      { date: '۱۴۰۵/۰۵/۲۳ ۱۸:۵۶', type: 'credit', channel: 'کارت به کارت', amount: 7500,    desc: 'پورسانت فاکتور ۲۶۶۷۱۳۱۱', rule: 'قانون مالی #۶' },
      { date: '۱۴۰۵/۰۵/۲۳ ۰۲:۰۴', type: 'credit', channel: 'آنلاین',      amount: 300000,  desc: 'پورسانت فاکتور ۵۲۶۲۱۰۱۸', rule: 'قانون مالی #۴' },
      { date: '۱۴۰۵/۰۵/۲۲ ۲۳:۱۵', type: 'credit', channel: 'کارت به کارت', amount: 480,     desc: 'پورسانت فاکتور ۴۳۹۹۷۵۰۱', rule: 'قانون مالی #۶' },
      { date: '۱۴۰۵/۰۵/۲۲ ۲۲:۴۴', type: 'credit', channel: 'آنلاین',      amount: 30000,   desc: 'پورسانت فاکتور ۹۶۸۸۰۲۹۷', rule: 'قانون مالی #۴' },
      { date: '۱۴۰۵/۰۵/۱۹ ۲۲:۲۷', type: 'credit', channel: 'آنلاین',      amount: 300000,  desc: 'پورسانت فاکتور ۸۳۱۱۴۵۱۱', rule: 'قانون مالی #۴' }
    ]
  },

  extraRequests: [
    { phone: '09120003344', name: 'کامران یزدی', when: '۱۴۰۵/۰۷/۰۶', status: 'در انتظار تایید مدیر', tone: 'amber' },
    { phone: '09331114455', name: '—',           when: '۱۴۰۵/۰۶/۲۹', status: 'تایید شد',             tone: 'green' },
    { phone: '09015556677', name: 'پرستو عباسی', when: '۱۴۰۵/۰۶/۱۲', status: 'رد شد',                tone: 'red' }
  ]
};
