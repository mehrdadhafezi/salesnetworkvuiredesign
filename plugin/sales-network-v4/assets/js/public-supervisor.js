/* Sales Network - Public JS */
(function ($) {
  'use strict';

  // Full-width portal helpers: add body class and mark panels without sidebar/tabs
  $(function(){
    if ($('.sn-panel, .sn-invoice-page, .sn-auth-wrap').length) {
      $('body').addClass('sn-portal-page');
    }
    $('.sn-panel').each(function(){
      var $p = $(this);
      if (!$p.children('.sn-tabs, .sn-panel-toolbar').length) {
        $p.addClass('sn-no-sidebar');
      } else {
        $p.removeClass('sn-no-sidebar').addClass('sn-has-sidebar-tabs');
      }
      // Make direct content blocks stretch even when the active tab changes dynamically
      $p.children('.sn-tab-content').css({width:'100%', maxWidth:'100%'});
    });
  });


  window.snAjax = window.snAjax || window.snData || {};
  window.snData = window.snData || window.snAjax;
  const ajax = snAjax.ajaxurl;
  const nonce = snAjax.nonce;
  // admins browsing public supervisor panel may have an admin nonce
  const adminNonce = (snAjax.admin_nonce && snAjax.admin_nonce.length) ? snAjax.admin_nonce : nonce;

  // ============================================================
  // TAB SWITCHER (universal)
  // ============================================================
  // تب‌های پنل (seller panel, supervisor, invoice)
  $(document).on('click', '.sn-tab', function () {
    var $panel = $(this).closest('.sn-panel, .sn-invoice-page');
    var target = $(this).data('tab');
    $panel.find('.sn-tab').removeClass('active');
    $panel.find('.sn-tab-content').removeClass('active').hide();
    $(this).addClass('active');
    $panel.find('#sn-tab-' + target).addClass('active').show();
  });

  // تب‌های فرم ورود/ثبت‌نام — handler مجزا
  $(document).on('click', '.sn-auth-tab', function () {
    const $card = $(this).closest('.sn-auth-card');
    const target = $(this).data('tab');
    $card.find('.sn-auth-tab').removeClass('active');
    $card.find('.sn-tab-content').removeClass('active');
    $(this).addClass('active');
    $card.find('#sn-tab-' + target).addClass('active');
  });



  // ============================================================
  // UI ENHANCEMENTS: KPI, Dark mode, Skeleton, live filters
  // ============================================================
  function snFormatNumber(value) {
    var n = Number(value || 0);
    try { return n.toLocaleString('fa-IR'); } catch(e) { return String(n); }
  }

  function snFormatMoney(value) {
    var n = Number(value || 0);
    try { return n.toLocaleString('fa-IR') + ' تومان'; } catch(e) { return String(n) + ' تومان'; }
  }

  function snToFaDigits(value) { return String(value || '').replace(/\d/g, function(d){ return '۰۱۲۳۴۵۶۷۸۹'[d]; }); }
  function snToEnDigits(value) { return String(value || '').replace(/[۰-۹]/g,function(d){return '۰۱۲۳۴۵۶۷۸۹'.indexOf(d);}).replace(/[٠-٩]/g,function(d){return '٠١٢٣٤٥٦٧٨٩'.indexOf(d);}); }
  function snPad(value) { return String(value).padStart(2, '0'); }
  function snTehranNowParts() {
    try {
      var parts = new Intl.DateTimeFormat('en-CA', {timeZone:'Asia/Tehran',year:'numeric',month:'numeric',day:'numeric',hour:'numeric',minute:'numeric',hourCycle:'h23'}).formatToParts(new Date());
      var out = {}; parts.forEach(function(p){ if (p.type !== 'literal') out[p.type] = Number(p.value); });
      return {year:out.year,month:out.month,day:out.day,hour:out.hour,minute:out.minute};
    } catch(e) { var d = new Date(Date.now() + 210 * 60000); return {year:d.getUTCFullYear(),month:d.getUTCMonth()+1,day:d.getUTCDate(),hour:d.getUTCHours(),minute:d.getUTCMinutes()}; }
  }
  function snGregorianToJalali(gy, gm, gd) {
    var gdm=[0,31,59,90,120,151,181,212,243,273,304,334], gy2=(gm>2)?gy+1:gy;
    var days=355666+(365*gy)+Math.floor((gy2+3)/4)-Math.floor((gy2+99)/100)+Math.floor((gy2+399)/400)+gd+gdm[gm-1];
    var jy=-1595+33*Math.floor(days/12053); days%=12053; jy+=4*Math.floor(days/1461); days%=1461;
    if(days>365){jy+=Math.floor((days-1)/365); days=(days-1)%365;}
    var jm=(days<186)?1+Math.floor(days/31):7+Math.floor((days-186)/30);
    var jd=1+((days<186)?days%31:(days-186)%30);
    return [jy,jm,jd];
  }
  function snJalaliMonthLength(jy, jm) {
    if (jm <= 6) return 31;
    if (jm <= 11) return 30;
    return (((jy - 474) % 2820 + 474 + 38) * 682 % 2816) < 682 ? 30 : 29;
  }
  function snParseJalali(value) {
    var raw = snToEnDigits(value || '').replace(/[-. ]/g, '/');
    var m = raw.match(/^(\d{4})\/(\d{1,2})\/(\d{1,2})$/);
    if (!m) return null;
    return { jy: Number(m[1]), jm: Number(m[2]), jd: Number(m[3]) };
  }
  function snTodayJalali() {
    var now = snTehranNowParts();
    var j = snGregorianToJalali(now.year, now.month, now.day);
    return { jy: j[0], jm: j[1], jd: j[2] };
  }
  function snSetJalaliInput($input, jy, jm, jd) {
    var val = jy + '/' + snPad(jm) + '/' + snPad(jd);
    $input.val(snToFaDigits(val)).trigger('change');
  }
  function snRenderJalaliPicker($picker, selected) {
    var state = $picker.data('state') || selected || snTodayJalali();
    var monthNames = ['فروردین','اردیبهشت','خرداد','تیر','مرداد','شهریور','مهر','آبان','آذر','دی','بهمن','اسفند'];
    var week = ['ش','ی','د','س','چ','پ','ج'];
    var html = '<div class="sn-jalali-head"><button type="button" class="sn-jalali-prev" aria-label="ماه قبل">‹</button><strong>' + monthNames[state.jm - 1] + ' ' + snToFaDigits(state.jy) + '</strong><button type="button" class="sn-jalali-next" aria-label="ماه بعد">›</button></div><div class="sn-jalali-week">';
    week.forEach(function(w){ html += '<span>' + w + '</span>'; });
    html += '</div><div class="sn-jalali-days">';
    for (var i = 1, len = snJalaliMonthLength(state.jy, state.jm); i <= len; i++) {
      html += '<button type="button" class="sn-jalali-day' + (selected && selected.jy === state.jy && selected.jm === state.jm && selected.jd === i ? ' is-selected' : '') + '" data-day="' + i + '">' + snToFaDigits(i) + '</button>';
    }
    html += '</div><div class="sn-jalali-actions"><button type="button" class="sn-jalali-today">امروز</button><button type="button" class="sn-jalali-clear">پاک کردن</button></div>';
    $picker.data('state', state).html(html);
  }
  function snAttachJalaliPicker($input) {
    if (!$input.length || $input.data('snPickerReady')) return;
    $input.attr({ autocomplete: 'off', inputmode: 'none', readonly: true }).wrap('<span class="sn-jalali-wrap"></span>');
    $input.after('<button type="button" class="sn-jalali-trigger" aria-label="باز کردن تقویم">تقویم</button><div class="sn-jalali-picker" hidden></div>');
    $input.data('snPickerReady', 1);
  }
  function snOpenJalaliPicker($input) {
    snAttachJalaliPicker($input);
    var selected = snParseJalali($input.val()) || snTodayJalali();
    var $picker = $input.siblings('.sn-jalali-picker');
    snRenderJalaliPicker($picker, selected);
    $('.sn-jalali-picker').not($picker).attr('hidden', true);
    $picker.removeAttr('hidden');
  }
  $(function(){ $('.sn-jalali-date').each(function(){ snAttachJalaliPicker($(this)); }); });
  $(document).on('focus click', '.sn-jalali-date', function(){ snOpenJalaliPicker($(this)); });
  $(document).on('click', '.sn-jalali-trigger', function(){ snOpenJalaliPicker($(this).siblings('.sn-jalali-date')); });
  $(document).on('click', '.sn-jalali-prev,.sn-jalali-next', function(){
    var $picker = $(this).closest('.sn-jalali-picker');
    var state = $picker.data('state') || snTodayJalali();
    state.jm += $(this).hasClass('sn-jalali-next') ? 1 : -1;
    if (state.jm < 1) { state.jm = 12; state.jy--; }
    if (state.jm > 12) { state.jm = 1; state.jy++; }
    state.jd = Math.min(state.jd || 1, snJalaliMonthLength(state.jy, state.jm));
    snRenderJalaliPicker($picker, snParseJalali($picker.siblings('.sn-jalali-date').val()));
  });
  $(document).on('click', '.sn-jalali-day', function(){
    var $picker = $(this).closest('.sn-jalali-picker');
    var state = $picker.data('state') || snTodayJalali();
    snSetJalaliInput($picker.siblings('.sn-jalali-date'), state.jy, state.jm, Number($(this).data('day')));
    $picker.attr('hidden', true);
  });
  $(document).on('click', '.sn-jalali-today', function(){
    var $picker = $(this).closest('.sn-jalali-picker');
    var today = snTodayJalali();
    snSetJalaliInput($picker.siblings('.sn-jalali-date'), today.jy, today.jm, today.jd);
    $picker.attr('hidden', true);
  });
  $(document).on('click', '.sn-jalali-clear', function(){
    var $picker = $(this).closest('.sn-jalali-picker');
    $picker.siblings('.sn-jalali-date').val('').trigger('change');
    $picker.attr('hidden', true);
  });
  $(document).on('click', function(e){
    if (!$(e.target).closest('.sn-jalali-wrap').length) $('.sn-jalali-picker').attr('hidden', true);
  });

  function snPercent(part, total) {
    part = Number(part || 0); total = Number(total || 0);
    if (!total) return '۰٪';
    try { return Math.round((part / total) * 100).toLocaleString('fa-IR') + '٪'; } catch(e) { return Math.round((part / total) * 100) + '%'; }
  }

  function snSkeletonRows(count, cols) {
    count = count || 5; cols = cols || 4;
    var html = '<div class="sn-skeleton-card"><table class="sn-table sn-skeleton-table"><tbody>';
    for (var r = 0; r < count; r++) {
      html += '<tr>';
      for (var c = 0; c < cols; c++) html += '<td><span class="sn-skeleton-line"></span></td>';
      html += '</tr>';
    }
    html += '</tbody></table></div>';
    return html;
  }

  function snEnsureDarkToggle($scope) {
    var $panel = ($scope && $scope.length) ? $scope : $('.sn-panel, .sn-invoice-page, .sn-auth-wrap').first();
    if (!$panel.length) return;
    var $header = $panel.find('.sn-panel-header, .sn-auth-head, .sn-invoice-head').first();
    if (!$header.length) { $header = $panel; }
    if (!$header.find('.sn-dark-toggle').length) {
      $header.append('<button type="button" class="sn-dark-toggle" aria-pressed="false">🌙 حالت شب</button>');
    }
    snApplyTheme();
  }

  function snApplyTheme() {
    var enabled = false;
    try { enabled = localStorage.getItem('sn_dark_mode') === '1'; } catch(e) {}
    $('body').toggleClass('sn-dark-mode', enabled);
    $('.sn-dark-toggle').attr('aria-pressed', enabled ? 'true' : 'false').text(enabled ? '☀️ حالت روشن' : '🌙 حالت شب');
  }

  $(document).off('click.snDarkToggle', '.sn-dark-toggle').on('click.snDarkToggle', '.sn-dark-toggle', function(e){
    e.preventDefault();
    var enabled = !$('body').hasClass('sn-dark-mode');
    try { localStorage.setItem('sn_dark_mode', enabled ? '1' : '0'); } catch(err) {}
    snApplyTheme();
  });
  $(function(){
    snEnsureDarkToggle($('.sn-panel, .sn-invoice-page, .sn-auth-wrap').first());
    snApplyTheme();
  });

  // ============================================================
  // SELLER PANEL
  // ============================================================
  var SN_CITIES = {"آذربایجان شرقی": ["تبریز", "مراغه", "مرند", "اهر", "بناب", "میانه", "سراب", "شبستر", "هشترود", "عجب‌شیر", "ملکان", "اسکو", "بستان‌آباد", "هریس", "کلیبر", "ورزقان", "خداآفرین", "چاراویماق"], "آذربایجان غربی": ["ارومیه", "خوی", "مهاباد", "بوکان", "میاندوآب", "اشنویه", "نقده", "سلماس", "پیرانشهر", "سردشت", "تکاب", "چالدران", "شاهین‌دژ", "ماکو", "پلدشت", "چایپاره"], "اردبیل": ["اردبیل", "پارس‌آباد", "خلخال", "مشگین‌شهر", "گرمی", "بیله‌سوار", "نمین", "نیر", "کوثر", "سرعین"], "اصفهان": ["اصفهان", "کاشان", "خمینی‌شهر", "نجف‌آباد", "شاهین‌شهر", "فلاورجان", "لنجان", "آران و بیدگل", "شهرضا", "مبارکه", "گلپایگان", "برخوار", "تیران و کرون", "سمیرم", "اردستان", "نائین", "خوانسار", "فریدن", "فریدونشهر", "دهاقان", "چادگان"], "البرز": ["کرج", "فردیس", "نظرآباد", "ساوجبلاغ", "طالقان", "محمدشهر", "هشتگرد"], "ایلام": ["ایلام", "دهلران", "ایوان", "مهران", "آبدانان", "دره‌شهر", "چرداول", "بدره", "ملکشاهی"], "بوشهر": ["بوشهر", "بندر گناوه", "برازجان", "بندر دیر", "خورموج", "کنگان", "جم", "دیلم"], "تهران": ["تهران", "شهریار", "پاکدشت", "ورامین", "دماوند", "فیروزکوه", "اسلامشهر", "رباط‌کریم", "قرچک", "ری", "ملارد", "بهارستان", "پردیس", "قدس"], "چهارمحال و بختیاری": ["شهرکرد", "بروجن", "فارسان", "لردگان", "اردل", "کوهرنگ", "سامان", "بن"], "خراسان جنوبی": ["بیرجند", "قاین", "نهبندان", "طبس", "سرایان", "فردوس", "درمیان", "سربیشه", "خوسف", "زیرکوه", "بشرویه"], "خراسان رضوی": ["مشهد", "سبزوار", "نیشابور", "تربت حیدریه", "کاشمر", "قوچان", "تربت جام", "چناران", "فریمان", "درگز", "تایباد", "خواف", "گناباد", "بردسکن", "جوین", "جغتای", "خلیل‌آباد", "مه‌ولات"], "خراسان شمالی": ["بجنورد", "شیروان", "اسفراین", "مانه و سملقان", "جاجرم", "گرمه", "فاروج"], "خوزستان": ["اهواز", "آبادان", "خرمشهر", "دزفول", "مسجدسلیمان", "بهبهان", "اندیمشک", "شوشتر", "شوش", "ماهشهر", "رامهرمز", "امیدیه", "ایذه", "باوی", "لالی", "هندیجان", "دشت آزادگان"], "زنجان": ["زنجان", "ابهر", "خدابنده", "قیدار", "ماهنشان", "سلطانیه", "طارم", "ایجرود"], "سمنان": ["سمنان", "شاهرود", "گرمسار", "دامغان", "مهدیشهر", "آرادان", "سرخه", "میامی"], "سیستان و بلوچستان": ["زاهدان", "چابهار", "زابل", "ایرانشهر", "خاش", "سراوان", "نیکشهر", "کنارک", "دلگان", "میرجاوه", "هیرمند", "قصرقند"], "فارس": ["شیراز", "مرودشت", "کازرون", "جهرم", "فسا", "لارستان", "داراب", "آباده", "نی‌ریز", "فیروزآباد", "استهبان", "اقلید", "ممسنی", "خرم‌بید", "پاسارگاد", "بوانات", "لامرد", "سپیدان", "گراش", "خنج"], "قزوین": ["قزوین", "البرز", "بویین‌زهرا", "تاکستان", "آوج"], "قم": ["قم"], "کردستان": ["سنندج", "سقز", "مریوان", "بانه", "قروه", "کامیاران", "بیجار", "دیواندره", "سروآباد", "دهگلان"], "کرمان": ["کرمان", "رفسنجان", "سیرجان", "جیرفت", "زرند", "شهربابک", "بافت", "بردسیر", "عنبرآباد", "کهنوج", "قلعه‌گنج", "منوجان", "نرماشیر", "فهرج"], "کرمانشاه": ["کرمانشاه", "اسلام‌آباد غرب", "کنگاور", "هرسین", "صحنه", "سنقر", "پاوه", "جوانرود", "روانسر", "دالاهو"], "کهگیلویه و بویراحمد": ["یاسوج", "گچساران", "دهدشت", "کهگیلویه", "بهمئی", "لنده", "باشت", "چرام"], "گلستان": ["گرگان", "گنبدکاووس", "آزادشهر", "علی‌آباد", "کردکوی", "بندرترکمن", "مینودشت", "رامیان", "گالیکش", "مراوه‌تپه", "کلاله", "آق‌قلا", "گمیشان"], "گیلان": ["رشت", "بندر انزلی", "لاهیجان", "لنگرود", "آستارا", "صومعه‌سرا", "رودبار", "رودسر", "تالش", "فومن", "شفت", "سیاهکل", "ماسال", "رضوانشهر"], "لرستان": ["خرم‌آباد", "بروجرد", "کوهدشت", "الیگودرز", "نورآباد", "ازنا", "دلفان", "سلسله", "رومشکان", "پلدختر"], "مازندران": ["ساری", "بابل", "آمل", "قائمشهر", "نوشهر", "بابلسر", "نکا", "چالوس", "تنکابن", "رامسر", "جویبار", "محمودآباد", "فریدونکنار", "بهشهر", "نور", "میاندورود", "سوادکوه", "کلاردشت"], "مرکزی": ["اراک", "ساوه", "خمین", "محلات", "دلیجان", "آشتیان", "شازند", "تفرش", "کمیجان", "زرندیه"], "هرمزگان": ["بندرعباس", "بندر لنگه", "قشم", "میناب", "حاجی‌آباد", "خمیر", "ابوموسی", "بستک", "پارسیان", "جاسک", "رودان"], "همدان": ["همدان", "ملایر", "نهاوند", "تویسرکان", "بهار", "اسدآباد", "کبودراهنگ", "رزن", "فامنین"], "یزد": ["یزد", "میبد", "اردکان", "بافق", "ابرکوه", "طبس", "مهریز", "خاتم", "تفت", "صدوق"]};

  // تبدیل تاریخ میلادی به شمسی
  function toJalali(dateStr) {
    if (!dateStr) return '—';
    var raw = String(dateStr || '').trim();
    if (/^(13|14)\d{2}[\/\-.]/.test(raw)) return raw.replace(/-/g, '/').replace(/\./g, '/');
    try {
      var wall = raw.match(/^(\d{4})-(\d{2})-(\d{2})(?:[ T](\d{2}):(\d{2}))?/);
      var d = new Date(dateStr.replace(' ', 'T'));
      if (isNaN(d.getTime())) return '—';
      var gy = wall ? Number(wall[1]) : d.getFullYear(), gm = wall ? Number(wall[2]) : d.getMonth()+1, gd = wall ? Number(wall[3]) : d.getDate();
      // الگوریتم صحیح تبدیل گریگوری به جلالی
      var g_y = gy - 1600, g_m = gm - 1, g_d = gd - 1;
      var g_d_no = 365*g_y + Math.floor((g_y+3)/4) - Math.floor((g_y+99)/100) + Math.floor((g_y+399)/400);
      var gDays = [31,28,31,30,31,30,31,31,30,31,30,31];
      if ((gy%4==0 && gy%100!=0) || gy%400==0) gDays[1] = 29;
      for (var i=0; i<g_m; i++) g_d_no += gDays[i];
      g_d_no += g_d;
      var j_d_no = g_d_no - 79;
      var j_np = Math.floor(j_d_no/12053); j_d_no %= 12053;
      var j_y = 979 + 33*j_np + 4*Math.floor(j_d_no/1461);
      j_d_no %= 1461;
      if (j_d_no >= 366) { j_y += Math.floor((j_d_no-1)/365); j_d_no = (j_d_no-1)%365; }
      var jDays = [31,31,31,31,31,31,30,30,30,30,30,29];
      var j_m = 0;
      for (var i=0; i<12; i++) { if (j_d_no >= jDays[i]) { j_d_no -= jDays[i]; j_m++; } else break; }
      var j_d = j_d_no + 1;
      var hh = String(wall && wall[4] ? wall[4] : d.getHours()).padStart(2,'0');
      var mm = String(wall && wall[5] ? wall[5] : d.getMinutes()).padStart(2,'0');
      return j_y + '/' + String(j_m+1).padStart(2,'0') + '/' + String(j_d).padStart(2,'0') + ' ' + hh + ':' + mm;
    } catch(e) { return '—'; }
  }

  function snBuildCityOptions(province, selectedCity) {
    var opts = '<option value="">انتخاب شهر</option>';
    if (province && SN_CITIES[province]) {
      SN_CITIES[province].forEach(function(c) {
        var normalizedSelected = String(selectedCity || '').trim();
        opts += '<option value="' + c + '"' + (normalizedSelected===String(c).trim()?' selected':'') + '>' + c + '</option>';
      });
    }
    return opts;
  }


  function snEsc(value) {
    return String(value === null || value === undefined ? '' : value)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
  }

  // ============================================================
  // SUPERVISOR PANEL
  // ============================================================
  const $supPanel = $('#sn-supervisor-panel');
  if ($supPanel.length) {
    snEnsureDarkToggle($supPanel);

    function snSupervisorOpsLayout(){
      var $role = $('#sn-tab-assign .sn-distribution-role-embedded');
      if (!$role.length) return;
      var $kpis = $role.find('.sn-distribution-kpis').first();
      var $form = $role.find('.sn-distribution-transfer-form').first();
      if ($kpis.length && $form.length) { $form.insertAfter($kpis); }
      var $case = $role.find('.sn-distribution-case-summary').first();
      if ($case.length && $form.length) { $case.insertAfter($form); }
      var key = 'sn_supervisor_last_per_seller_count_' + (snAjax.user_id || 'current');
      var $count = $form.find('input[name="per_seller_count"]');
      try { var saved = localStorage.getItem(key); if (saved && !$count.val()) $count.val(saved); } catch(e) {}
      $count.off('input.snPersistCount change.snPersistCount').on('input.snPersistCount change.snPersistCount', function(){
        var val = String($(this).val() || '').trim();
        try { if (val) localStorage.setItem(key, val); } catch(e) {}
      });
      $form.off('submit.snPersistCount').on('submit.snPersistCount', function(){
        var val = String($count.val() || '').trim();
        try { if (val) localStorage.setItem(key, val); } catch(e) {}
      });
    }
    snSupervisorOpsLayout();
    $(document).on('click', '#sn-supervisor-panel .sn-tab[data-tab="assign"]', function(){ setTimeout(snSupervisorOpsLayout, 30); });

    var snSupervisorDataCache = {};
    var snSupervisorDataPending = null;
    function loadSupervisorData(force) {
      var profileEnabled=/(?:^|[?&])sn_profile=1(?:&|$)/.test(window.location.search);
      var profileStarted=profileEnabled&&window.performance?performance.now():0;
      var params = {
        action:     'sn_supervisor_data',
        nonce:      nonce,
        search:     $('#sn-seller-search').val() || '',
        filter_act: $('#sn-seller-filter-act').val() || 'all',
        date_from:  $('#sn-date-from').val() || '',
        date_to:    $('#sn-date-to').val() || '',
        time_from:  $('#sn-time-from').val() || '',
        time_to:    $('#sn-time-to').val() || '',
        seller_id:  $('#sn-summary-seller').val() || '',
        lead_status: $('#sn-summary-lead-status').val() || '',
        import_code: $('#sn-summary-import-code').val() || '',
        assignment: $('#sn-summary-assignment').val() || '',
        sn_profile: profileEnabled ? 1 : 0,
        force_refresh: profileEnabled ? 1 : 0
      };
      var cacheKey = JSON.stringify(params);
      if (!force && snSupervisorDataCache[cacheKey] && (Date.now() - snSupervisorDataCache[cacheKey].time) < 15000) {
        renderSupervisorDataResponse(snSupervisorDataCache[cacheKey].res);
        return;
      }
      if (snSupervisorDataPending && snSupervisorDataPending.readyState !== 4) { try { snSupervisorDataPending.abort(); } catch(e) {} }
      $('#sn-sellers-loading').show();
      $('#sn-sellers-table').html(snSkeletonRows(5, 7));
      snSupervisorDataPending = $.post(ajax, params, function (res) {
        snSupervisorDataCache[cacheKey] = {time: Date.now(), res: res};
        renderSupervisorDataResponse(res);
        if(profileEnabled&&window.console&&console.info) console.info('[CRM-44] supervisor data',{client_ms:Math.round(performance.now()-profileStarted),server:res&&res.profile,cache_hit:res&&res.cache_hit});
      }).fail(function(xhr){
        if (xhr && xhr.statusText === 'abort') return;
        $('#sn-sellers-loading').hide();
        $('#sn-sellers-table').html('<div class="sn-notice sn-error">خطای سرور: ' + snEsc(xhr.status || '') + '</div>');
      });
    }

    function renderSupervisorDataResponse(res) {
      $('#sn-sellers-loading').hide();
        if (!res || !res.success) {
          var msg = (res && res.message) ? res.message : 'خطا در دریافت اطلاعات فروشنده‌ها';
          $('#sn-sellers-table').html('<div class="sn-notice sn-error">' + snEsc(msg) + '</div>');
          $('#sn-sellers-checkboxes').html('<div class="sn-notice sn-error">' + snEsc(msg) + '</div>');
          return;
        }
        renderSupervisorKpis(res);
        res.sellers = Array.isArray(res.sellers) ? res.sellers : [];

        $('#sn-unassigned-count').text(res.unassigned);
        if (res.summary) {
          $('#sn-sum-total').text(res.summary.total || 0);
          $('#sn-sum-assigned').text(res.summary.assigned || 0);
          $('#sn-sum-unassigned').text(res.summary.unassigned || 0);
          $('#sn-sum-range').text(res.summary.range_assigned || 0);
          $('#sn-sum-invoiced').text(res.summary.invoiced || 0);
          $('#sn-sum-paid').text(res.summary.paid || 0);
        }

        // جدول فروشندگان
        var html = '<div class="sn-table-wrap"><table class="sn-table sn-supervisor-sellers-table" style="width:100%">' +
          '<thead><tr><th>نام</th><th>شماره</th><th>تخصیص‌یافته</th><th>فاکتور</th><th>پرداخت‌شده</th><th>وضعیت</th><th>عملیات</th></tr></thead><tbody>';
        res.sellers.forEach(function(s, i) {
          var bg = i%2===0 ? '#fff' : '#f8fafc';
          var activeLabel = s.is_active
            ? '<span style="background:#dcfce7;color:#16a34a;padding:2px 8px;border-radius:10px;font-size:.78rem">فعال</span>'
            : '<span style="background:#fee2e2;color:#dc2626;padding:2px 8px;border-radius:10px;font-size:.78rem">غیرفعال</span>';
          html += '<tr class="sn-seller-row" id="sn-seller-row-' + s.id + '" style="background:' + bg + '">' +
            '<td><strong>' + snEsc(s.name) + '</strong></td>' +
            '<td>' + snEsc(s.phone) + '</td>' +
            '<td>' + (s.lead_count || 0) + '</td>' +
            '<td>' + (s.invoice_count || 0) + '</td>' +
            '<td>' + (s.paid_count||0) + '</td>' +
            '<td>' + activeLabel + '</td>' +
            '<td><button type="button" class="sn-btn sn-btn-sm sn-seller-profile" data-id="' + s.id + '" aria-expanded="false">پروفایل</button></td>' +
          '</tr>' +
          '<tr class="sn-seller-profile-row" id="sn-seller-profile-row-' + s.id + '" style="display:none;background:#f0f9ff">' +
            '<td colspan="7"><div class="sn-seller-profile-panel" data-loaded="0"></div></td>' +
          '</tr>';
        });
        if (!res.sellers.length) {
          html += '<tr><td colspan="7"><div class="sn-notice">هیچ فروشنده‌ای برای این سرپرست پیدا نشد. فروشنده باید در HR به عنوان زیرمجموعه مستقیم این سرپرست ثبت و فعال باشد.</div></td></tr>';
        }
        html += '</tbody></table></div>';
        $("#sn-sellers-table").html(html);
        // checkbox های تخصیص
        var cbHtml = '<label style="display:block;margin-bottom:6px">' +
          '<input type="checkbox" id="sn-select-all-sellers"> <strong>انتخاب همه</strong></label><hr style="margin:6px 0">';
        res.sellers.filter(function(s){ return s.is_active; }).forEach(function(s) {
          cbHtml += '<label style="display:block;padding:3px 0"><input type="checkbox" class="sn-seller-cb" value="' + s.id + '"> ' + s.name + ' (' + s.phone + ')</label>';
        });
        $('#sn-sellers-checkboxes').html(cbHtml);

        // select فروشنده برای حالت دستی
        var optHtml = '<option value="">انتخاب فروشنده</option>';
        res.sellers.filter(function(s){ return s.is_active; }).forEach(function(s) {
          optHtml += '<option value="' + s.id + '">' + s.name + ' — ' + s.phone + '</option>';
        });
        $('#sn-manual-seller').html(optHtml);
        var currentSummarySeller = $('#sn-summary-seller').val() || '';
        $('#sn-summary-seller').html('<option value="">همه</option>' + optHtml.replace('<option value="">انتخاب فروشنده</option>', '')).val(currentSummarySeller);
        $('#sn-unassign-seller').html('<option value="">همه فروشنده‌ها</option>' + optHtml.replace('<option value="">انتخاب فروشنده</option>', ''));
        var statusOpts = '<option value="">همه</option>';
        (res.lead_statuses || []).forEach(function(st){ statusOpts += '<option value="' + snEsc(st) + '">' + snEsc(st) + '</option>'; });
        var currentStatus = $('#sn-summary-lead-status').val() || '';
        var currentUnStatus = $('#sn-unassign-lead-status').val() || '';
        $('#sn-summary-lead-status').html(statusOpts).val(currentStatus);
        $('#sn-unassign-lead-status').html(statusOpts).val(currentUnStatus);
    }


    function renderSupervisorKpis(res) {
      var sellers = res.sellers || [];
      var summary = res.summary || {};
      var totalLeads = Number(summary.total_leads || 0);
      var assigned = Number(summary.assigned || 0);
      var unassigned = Number(summary.unassigned || summary.pool_count || 0);
      var invoices = Number(summary.invoices || 0);
      var sales = Number(summary.sales || summary.revenue || 0);
      if (!totalLeads && sellers.length) {
        totalLeads = sellers.reduce(function(sum, s){ return sum + Number(s.lead_count || 0); }, 0);
        invoices = sellers.reduce(function(sum, s){ return sum + Number(s.invoice_count || s.invoices || 0); }, 0);
        sales = sellers.reduce(function(sum, s){ return sum + Number(s.revenue || s.sales || 0); }, 0);
        assigned = totalLeads;
      }
      var html = '<div class="sn-kpi-grid sn-supervisor-kpis">' +
        '<div class="sn-kpi-card"><span class="sn-kpi-icon">👥</span><small>فروشندگان فعال</small><strong>' + snFormatNumber(sellers.filter(function(s){return !!s.is_active;}).length) + '</strong><em>از ' + snFormatNumber(sellers.length) + ' فروشنده زیرمجموعه</em></div>' +
        '<div class="sn-kpi-card"><span class="sn-kpi-icon">📞</span><small>شماره‌های زیرمجموعه</small><strong>' + snFormatNumber(totalLeads) + '</strong><em>طبق فیلترهای همین تب</em></div>' +
        '<div class="sn-kpi-card"><span class="sn-kpi-icon">📌</span><small>تحویل/تخصیص به فروشنده</small><strong>' + snFormatNumber(assigned) + '</strong><em>مانده آزاد: ' + snFormatNumber(unassigned) + '</em></div>' +
        '<div class="sn-kpi-card"><span class="sn-kpi-icon">🧾</span><small>فاکتورهای تایید/پرداخت</small><strong>' + snFormatNumber(summary.paid || 0) + '</strong><em>از ' + snFormatNumber(invoices) + ' فاکتور</em></div>' +
        '<div class="sn-kpi-card sn-kpi-money"><span class="sn-kpi-icon">💳</span><small>فروش تاییدشده</small><strong>' + snFormatMoney(sales) + '</strong><em>مبنای کیف پول و گزارش مالی</em></div>' +
      '</div><div class="sn-notice sn-info sn-supervisor-filter-note">فیلترهای این تب آمار و جدول فروشندگان را بازخوانی می‌کند؛ خروجی، وضعیت فروشندگان زیرمجموعه در بازه/وضعیت انتخاب‌شده است.</div>';
      var $target = $('#sn-supervisor-kpi-cards');
      if (!$target.length) {
        $target = $('<div id="sn-supervisor-kpi-cards" class="sn-kpi-host"></div>');
        var $tab = $('#sn-tab-sellers');
        if ($tab.length) $tab.prepend($target); else $supPanel.prepend($target);
      }
      $target.html(html);
    }

    $('#sn-sellers-loading').hide();
    $('#sn-sellers-table').html('<div class="sn-card sn-notice sn-info"><strong>پنل آماده است.</strong><br>برای جلوگیری از 504، آمار فروشندگان بعد از ورود خودکار لود نمی‌شود. روی «اعمال فیلتر» یا دکمه زیر بزنید.<br><button type="button" id="sn-load-supervisor-data-manual" class="sn-btn sn-btn-primary" style="margin-top:8px">بارگذاری آمار فروشندگان</button></div>');

    // جستجو
    $('#sn-seller-search-btn').on('click', loadSupervisorData);
    var snSupFilterTimer = null;
    $('#sn-seller-search').on('input', function(){ clearTimeout(snSupFilterTimer); snSupFilterTimer = setTimeout(loadSupervisorData, 450); });
    $('#sn-seller-search').on('keydown', function(e){ if(e.key==='Enter') $('#sn-sellers-loading').hide();
    $('#sn-sellers-table').html('<div class="sn-card sn-notice sn-info"><strong>پنل آماده است.</strong><br>برای جلوگیری از 504، آمار فروشندگان بعد از ورود خودکار لود نمی‌شود. روی «اعمال فیلتر» یا دکمه زیر بزنید.<br><button type="button" id="sn-load-supervisor-data-manual" class="sn-btn sn-btn-primary" style="margin-top:8px">بارگذاری آمار فروشندگان</button></div>'); });
    $('#sn-seller-filter-act,#sn-date-from,#sn-date-to,#sn-time-from,#sn-time-to,#sn-summary-seller,#sn-summary-lead-status,#sn-summary-assignment').on('change', loadSupervisorData);
    $('#sn-summary-import-code').on('input', function(){ clearTimeout(snSupFilterTimer); snSupFilterTimer = setTimeout(loadSupervisorData, 450); });
    $(document).on('click', '#sn-summary-filter,#sn-load-supervisor-data-manual', function(){ loadSupervisorData(true); });
    $(document).on('click', '#sn-summary-export', function(){
      var rows = [['فروشنده','موبایل','لیدها','فاکتورها','پرداخت شده','وضعیت']];
      $('#sn-sellers-table tbody tr.sn-seller-row').each(function(){
        var cols = $(this).children('td').map(function(){ return $(this).text().replace(/\s+/g, ' ').trim(); }).get();
        rows.push(cols.slice(0, 6));
      });
      var csv = rows.map(function(r){ return r.map(function(c){ return '"' + String(c).replace(/"/g,'""') + '"'; }).join(','); }).join('\n');
      var blob = new Blob(["\ufeff" + csv], {type:'text/csv;charset=utf-8;'});
      var a = document.createElement('a'); a.href = URL.createObjectURL(blob); a.download = 'supervisor-report.csv'; a.click(); URL.revokeObjectURL(a.href);
    });

    // select all
    $(document).on('change', '#sn-select-all-sellers', function() {
      $('.sn-seller-cb').prop('checked', $(this).is(':checked'));
    });

    // toggle فعال/غیرفعال
    $(document).on('click', '.sn-toggle-seller', function() {
      var $btn = $(this);
      var id = $btn.data('id');
      var isActive = $btn.data('active');
      var label = isActive ? 'غیرفعال' : 'فعال';
      if (!confirm('فروشنده ' + label + ' شود؟')) return;
      $.post(ajax, { action: 'sn_toggle_seller_active', nonce: nonce, seller_id: id }, function(res) {
        if (res.success) { $('#sn-sellers-loading').hide();
    $('#sn-sellers-table').html('<div class="sn-card sn-notice sn-info"><strong>پنل آماده است.</strong><br>برای جلوگیری از 504، آمار فروشندگان بعد از ورود خودکار لود نمی‌شود. روی «اعمال فیلتر» یا دکمه زیر بزنید.<br><button type="button" id="sn-load-supervisor-data-manual" class="sn-btn sn-btn-primary" style="margin-top:8px">بارگذاری آمار فروشندگان</button></div>'); }
        else alert('❌ ' + res.message);
      });
    });

    // پروفایل فروشنده برای سرپرست - کشویی باز/بسته می‌شود
    $(document).on("click", ".sn-seller-profile", function() {
      var $btn = $(this);
      var id = $btn.data("id");
      var $row = $('#sn-seller-profile-row-' + id);
      var $panel = $row.find('.sn-seller-profile-panel');

      // اگر باز است، با کلیک مجدد بسته شود
      if ($row.is(':visible')) {
        $row.slideUp(160);
        $btn.attr('aria-expanded', 'false').removeClass('active').text('پروفایل');
        return;
      }

      // فقط یک پروفایل همزمان باز باشد
      $('.sn-seller-profile-row:visible').slideUp(160);
      $('.sn-seller-profile').attr('aria-expanded', 'false').removeClass('active').text('پروفایل');
      $btn.attr('aria-expanded', 'true').addClass('active').text('بستن پروفایل');
      $row.slideDown(160);

      if ($panel.data('loaded') === 1) {
        return;
      }

      $panel.html('<div class="sn-loading">در حال بارگذاری پروفایل...</div>');
      $.post(ajax, { action: "sn_seller_profile", nonce: nonce, seller_id: id }, function(res) {
        if (!res.success) {
          $panel.html('<div class="sn-notice sn-error">❌ ' + snEsc(res.message || 'خطا در دریافت پروفایل') + '</div>');
          return;
        }
        var s = res.seller || {}, st = res.stats || {};
        var html = '<div class="sn-card sn-seller-profile-card">' +
          '<div class="sn-profile-head"><h3>پروفایل فروشنده: ' + snEsc(s.name || '—') + '</h3>' +
          '<span class="sn-badge">' + snEsc(s.phone || '—') + '</span></div>' +
          '<p><b>تاریخ عضویت:</b> ' + snEsc(s.registered || '—') + '</p>' +
          '<div class="sn-grid sn-seller-profile-stats" style="display:grid;grid-template-columns:repeat(auto-fit,minmax(130px,1fr));gap:8px">' +
          '<div class="sn-mini-stat"><b>لیدها</b><span>' + (st.leads || 0) + '</span></div>' +
          '<div class="sn-mini-stat"><b>فاکتورها</b><span>' + (st.invoices || 0) + '</span></div>' +
          '<div class="sn-mini-stat"><b>پرداخت‌شده</b><span>' + (st.paid || 0) + '</span></div>' +
          '<div class="sn-mini-stat"><b>فروش</b><span>' + (st.revenue || 0) + '</span></div></div>';
        html += '<h4>آخرین لیدها</h4><table class="sn-table"><thead><tr><th>شماره</th><th>شهر</th><th>وضعیت</th><th>تخصیص</th></tr></thead><tbody>';
        if ((res.recent_leads || []).length) {
          (res.recent_leads || []).forEach(function(l){
            html += '<tr><td>'+snEsc(l.phone||'')+'</td><td>'+snEsc((l.province||'')+' / '+(l.city||''))+'</td><td>'+snEsc(l.lead_status||l.status||'')+'</td><td>'+snEsc(toJalali(l.assigned_at))+'</td></tr>';
          });
        } else {
          html += '<tr><td colspan="4">لیدی ثبت نشده است.</td></tr>';
        }
        html += '</tbody></table><h4>آخرین فاکتورها</h4><table class="sn-table"><thead><tr><th>کد</th><th>مشتری</th><th>مبلغ</th><th>وضعیت</th></tr></thead><tbody>';
        if ((res.recent_invoices || []).length) {
          (res.recent_invoices || []).forEach(function(i){
            html += '<tr><td>'+snEsc(i.invoice_code||'')+'</td><td>'+snEsc(i.customer_name||'')+'</td><td>'+snEsc(i.product_price||0)+'</td><td>'+snEsc(i.status||'')+'</td></tr>';
          });
        } else {
          html += '<tr><td colspan="4">فاکتوری ثبت نشده است.</td></tr>';
        }
        html += '</tbody></table></div>';
        $panel.html(html).data('loaded', 1);
      }).fail(function() {
        $panel.html('<div class="sn-notice sn-error">❌ خطای ارتباط با سرور</div>');
      });
    });

    // Load unassigned phones for manual mode
    function loadUnassigned() {
      $.post(ajax, { action: 'sn_get_unassigned', nonce: nonce }, function (res) {
        if (!res.success || !res.leads) return;
        var html = '<label style="display:block;margin-bottom:6px"><input type="checkbox" id="sn-select-all-leads"> <strong>انتخاب همه</strong></label><hr style="margin:6px 0">';
        res.leads.forEach(function (l) {
          html += '<div class="sn-phone-item"><label><input type="checkbox" class="sn-lead-cb" value="' + l.id + '"> ' + l.phone + '</label></div>';
        });
        $('#sn-unassigned-list').html(html || '<p>شماره‌ای یافت نشد</p>');
      });
    }

    // select all leads
    $(document).on('change', '#sn-select-all-leads', function() {
      $('.sn-lead-cb').prop('checked', $(this).is(':checked'));
    });

    // Mode toggle
    $('input[name="assign_mode"]').on('change', function () {
      if ($(this).val() === 'manual') {
        $('#sn-assign-count-mode').hide();
        $('#sn-assign-manual-mode').show();
        loadUnassigned();
      } else {
        $('#sn-assign-count-mode').show();
        $('#sn-assign-manual-mode').hide();
      }
    });

    // Do assign
    $('#sn-do-assign').on('click', function () {
      var $btn = $(this);
      var mode = $('input[name="assign_mode"]:checked').val();
      var data = { action: 'sn_assign_leads', nonce: nonce, mode: mode };

      if (mode === 'count') {
        var sellerIds = [];
        $('.sn-seller-cb:checked').each(function () { sellerIds.push($(this).val()); });
        if (!sellerIds.length) { showNotice('#sn-assign-notice', 'فروشنده‌ای انتخاب نشده', 'error'); return; }
        data['seller_ids[]'] = sellerIds;
        data.count_per_seller = $("#sn-count-per-seller").val();
        if (!data.count_per_seller || Number(data.count_per_seller) < 1) { showNotice('#sn-assign-notice', 'تعداد را وارد کنید', 'error'); return; }
      } else {
        var leadIds = [];
        $('.sn-lead-cb:checked').each(function () { leadIds.push($(this).val()); });
        var sellerId = $('#sn-manual-seller').val();
        if (!sellerId || !leadIds.length) { showNotice('#sn-assign-notice', 'فروشنده و شماره را انتخاب کنید', 'error'); return; }
        data['seller_ids[]'] = [sellerId];
        data['lead_ids[]']   = leadIds;
      }

      $btn.prop('disabled', true).text('در حال پردازش...');
      $.post(ajax, data, function (res) {
        $btn.prop('disabled', false).text('اعمال تخصیص');
        if (res.success) {
          showNotice('#sn-assign-notice', '✅ ' + res.message, 'success');
          $('#sn-sellers-loading').hide();
    $('#sn-sellers-table').html('<div class="sn-card sn-notice sn-info"><strong>پنل آماده است.</strong><br>برای جلوگیری از 504، آمار فروشندگان بعد از ورود خودکار لود نمی‌شود. روی «اعمال فیلتر» یا دکمه زیر بزنید.<br><button type="button" id="sn-load-supervisor-data-manual" class="sn-btn sn-btn-primary" style="margin-top:8px">بارگذاری آمار فروشندگان</button></div>');
          if (mode === 'manual') loadUnassigned();
        } else {
          showNotice('#sn-assign-notice', '❌ ' + res.message, 'error');
        }
      });
    });
  }


  // ============================================================
  // HELPERS
  // ============================================================
  function showNotice(selector, msg, type) {
    $(selector).html(`<div class="sn-notice sn-${snEsc(type || 'info')}">${snEsc(msg)}</div>`);
    setTimeout(function () { $(selector).empty(); }, 6000);
  }

}(jQuery));

/* SN vNext: supervisor assignment guard, unassign, manual card payment */
(function($){
  'use strict';
  if (typeof snAjax === 'undefined') return;
  var ajax = snAjax.ajaxurl, nonce = snAjax.nonce;
  var adminNonce = (snAjax.admin_nonce && snAjax.admin_nonce.length) ? snAjax.admin_nonce : nonce;

  function snNotice(sel, msg, type) {
    $(sel).html('<div class="sn-notice sn-' + snEsc(type || 'info') + '">' + snEsc(msg) + '</div>');
  }

  function snEsc(value) {
    return String(value === null || value === undefined ? '' : value)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/\"/g, '&quot;')
      .replace(/'/g, '&#039;');
  }

  function updateAssignButtonState(){
    var mode = $('input[name="assign_mode"]:checked').val();
    var ok = true;
    if (mode === 'count') {
      var c = $('#sn-count-per-seller').val();
      ok = !!c && Number(c) > 0;
    }
    $('#sn-do-assign').prop('disabled', !ok);
  }
  $(document).on('input change', '#sn-count-per-seller, input[name="assign_mode"]', updateAssignButtonState);
  $(updateAssignButtonState);

  $(document).on('click', '#sn-do-unassign', function(e){
    e.preventDefault();
    if (!confirm('لیدهای مطابق فیلتر از فروشنده جدا شوند؟')) return;
    var $btn = $(this);
    $btn.prop('disabled', true).text('در حال انجام...');
    $.post(ajax, {
      action: 'sn_supervisor_unassign_leads', nonce: nonce,
      seller_id: $('#sn-unassign-seller').val() || '',
      count: $('#sn-unassign-count').val() || '',
      date_from: $('#sn-unassign-date-from').val() || '',
      date_to: $('#sn-unassign-date-to').val() || '',
      time_from: $('#sn-unassign-time-from').val() || '',
      time_to: $('#sn-unassign-time-to').val() || '',
      lead_status: $('#sn-unassign-lead-status').val() || '',
      import_code: $('#sn-unassign-import-code').val() || ''
    }, function(res){
      $btn.prop('disabled', false).text('جدا کردن و برگشت به لیست قابل تخصیص');
      if (res && res.success) snNotice('#sn-unassign-notice', '✅ ' + (res.message || 'انجام شد'), 'success');
      else snNotice('#sn-unassign-notice', '❌ ' + ((res && res.message) || 'خطا در عملیات'), 'error');
    }).fail(function(xhr){
      $btn.prop('disabled', false).text('جدا کردن و برگشت به لیست قابل تخصیص');
      snNotice('#sn-unassign-notice', '❌ خطای سرور: ' + xhr.status, 'error');
    });
  });

  $(document).on('click', '#sn-card-manual-toggle', function(){
    $('#sn-card-manual-fields').slideToggle();
  });

  $(document).on('click', '#sn-submit-manual-payment', function(e){
    e.preventDefault();
    var code = $('#sn-invoice-page').data('active-code') || $('#sn-inv-code').val();
    var from = $('#sn-card-from4').val();
    var to = $('#sn-card-to4').val();
    var amount = $('#sn-card-amount').val();
    var paidAt = $('#sn-card-paid-at').val();
    if (!/^\d{4}$/.test(from)) { alert('۴ رقم کارت مبدا باید عددی باشد'); return; }
    if (to!=='' && !/^\d{4}$/.test(to)) { alert('۴ رقم کارت مقصد اختیاری است؛ در صورت ورود باید ۴ رقم باشد'); return; }
    if (!amount || isNaN(String(amount).replace(/,/g,''))) { alert('مبلغ باید عدد باشد'); return; }
    var $btn = $(this).prop('disabled', true).text('در حال ثبت...');
    $.post(ajax, {action:'sn_submit_manual_payment', nonce: nonce, invoice_code: code, card_from: from, card_to: to, amount: amount, paid_at: paidAt}, function(res){
      $btn.prop('disabled', false).text('ثبت اطلاعات واریز');
      if (res && res.success) { $('#sn-payment-section').hide(); $('<div class="sn-notice sn-success">✅ '+res.message+'</div>').insertBefore('#sn-payment-section'); if (typeof loadInvoice === 'function') loadInvoice(code); }
      else alert((res && res.message) || 'خطا در ثبت اطلاعات واریز');
    }).fail(function(xhr){ $btn.prop('disabled', false).text('ثبت اطلاعات واریز'); alert('خطای سرور: '+xhr.status); });
  });

  function snFaStatus(st){ var m={pre_invoice:'پیش‌فاکتور',pending:'در انتظار پرداخت',pending_payment:'در انتظار پرداخت',receipt_uploaded:'نیاز به بررسی فیش',pending_financial_approval:'در انتظار تایید مالی',approved:'تایید شده',paid:'پرداخت‌شده',rejected:'رد شده',cancelled:'لغوشده',assigned:'تخصیص داده شده'}; return m[st]||st||'—'; }
  // 1.0.120: legacy supervisor invoice loader disabled.
  // The final lazy invoice loader below is the single source of truth; keeping both caused duplicate Ajax calls.

  // 1.0.75: manager report rendering guard. This IIFE cannot see helpers defined in earlier closures,
  // so using toJalali/snEsc from another block can throw ReferenceError after the total count updates.
  function snManagerDateLabel(value) {
    if (value === null || value === undefined || value === '') return '—';
    try {
      if (typeof toJalali === 'function') {
        return toJalali(value);
      }
    } catch (e) {}
    return String(value).replace(/-/g, '/');
  }
  function snManagerFilters(){
    return {
      search: $('#sn-manager-search').val() || '',
      import_code: $('#sn-manager-import-code-filter').val() || $('#sn-manager-import-code').val() || '',
      date_from: $('#sn-manager-date-from').val() || '',
      date_to: $('#sn-manager-date-to').val() || '',
      time_from: $('#sn-manager-time-from').val() || '',
      time_to: $('#sn-manager-time-to').val() || '',
      status: $('#sn-manager-status').val() || 'all',
      assignment: $('#sn-manager-assignment').val() || 'all',
      supervisor_id: $('#sn-manager-supervisor-filter').val() || '',
      seller_id: $('#sn-manager-seller-filter').val() || '',
      lead_status: $('#sn-manager-lead-status').val() || ''
    };
  }
  function snLoadManagerLeads(){
    if(!$('#sn-manager-leads-list').length) return;
    $('#sn-manager-leads-list').html('در حال بارگذاری...');
    var data = snManagerFilters();
    data.action = 'sn_sales_manager_leads';
    data.nonce = nonce;
    data.limit = 500;
    $.ajax({url: ajax, type: 'POST', dataType: 'json', timeout: 25000, data: data}).done(function(res){
      var payload = (res && res.data) ? res.data : (res || {});
      if(!res || !res.success){
        $('#sn-manager-leads-list').html('❌ ' + snEsc((payload && payload.message) || (res && res.message) || 'خطا در دریافت گزارش'));
        return;
      }
      $('#sn-manager-total').text(payload.total || res.total || 0);
      try {
      var rows = payload.items || res.items || [];
      var html = '<table class="sn-table sn-manager-table"><thead><tr><th>شماره</th><th>کد</th><th>موقعیت</th><th>وضعیت</th><th>سرپرست</th><th>فروشنده</th><th>ورود</th><th>تخصیص</th></tr></thead><tbody>';
      if(!rows.length) html += '<tr><td colspan="8">موردی با این فیلتر پیدا نشد.</td></tr>';
      rows.forEach(function(l){
        var loc = l.province && l.city ? (l.province + ' / ' + l.city) : (l.province || l.city || '—');
        html += '<tr><td><code>' + snEsc(l.phone || '') + '</code></td><td>' + snEsc(l.import_code || '—') + '</td><td>' + snEsc(loc) + '</td><td>' + snEsc(l.lead_status || l.status_label || l.status || '—') + '</td><td>' + snEsc(l.supervisor_name || '—') + '</td><td>' + snEsc(l.seller_name || '—') + '</td><td>' + snEsc(snManagerDateLabel(l.imported_at)) + '</td><td>' + snEsc(snManagerDateLabel(l.assigned_at)) + '</td></tr>';
      });
      html += '</tbody></table>';
      $('#sn-manager-leads-list').html(html);
      } catch(renderErr) {
        if (window.console && console.error) console.error('SN manager report render failed', renderErr);
        $('#sn-manager-leads-list').html('❌ خطا در نمایش گزارش: ' + snEsc(renderErr && renderErr.message ? renderErr.message : 'خطای نامشخص'));
      }
    }).fail(function(xhr){
      var msg = 'خطای سرور: ' + (xhr && xhr.status ? xhr.status : 'نامشخص');
      if (xhr && xhr.responseJSON && xhr.responseJSON.message) msg = xhr.responseJSON.message;
      $('#sn-manager-leads-list').html('❌ ' + snEsc(msg));
    });
  }
  $(document).on('click', '#sn-manager-filter', function(e){ e.preventDefault(); snLoadManagerLeads(); });
  $(document).on('change', '#sn-manager-status,#sn-manager-assignment,#sn-manager-supervisor-filter,#sn-manager-seller-filter,#sn-manager-lead-status,#sn-manager-date-from,#sn-manager-date-to,#sn-manager-time-from,#sn-manager-time-to', function(){ snLoadManagerLeads(); });
  $(document).on('input', '#sn-manager-search,#sn-manager-import-code-filter', function(){ clearTimeout(window.snManagerFilterTimer); window.snManagerFilterTimer = setTimeout(snLoadManagerLeads, 450); });
  $(document).on('click', '#sn-manager-export', function(e){
    e.preventDefault();
    var base = $(this).data('export-base');
    if(!base) return;
    var data = snManagerFilters();
    data.nonce = nonce;
    window.location.href = base + '&' + $.param(data);
  });
  if($('#sn-sales-manager-panel').length) snLoadManagerLeads();
  $(document).on('click', '#sn-manager-assign', function(e){
    e.preventDefault();
    var $btn = $(this);
    var sup = $('#sn-manager-supervisor').val();
    var count = $('#sn-manager-count').val();
    if(!sup || !count || Number(count) < 1){ snNotice('#sn-manager-assign-notice', 'سرپرست و تعداد را انتخاب کنید', 'error'); return; }
    $btn.prop('disabled', true).text('در حال انتقال...');
    var data = snManagerFilters();
    data.action = 'sn_assign_supervisor_leads';
    data.nonce = adminNonce || nonce;
    data.supervisor_id = sup;
    data.count = count;
    data.import_code = $('#sn-manager-import-code').val() || data.import_code || '';
    $.post(ajax, data, function(res){
      $btn.prop('disabled', false).text('انتقال به سرپرست');
      if(res && res.success){ snNotice('#sn-manager-assign-notice', '✅ ' + (res.message || 'انجام شد'), 'success'); snLoadManagerLeads(); }
      else snNotice('#sn-manager-assign-notice', '❌ ' + ((res && res.message) || 'خطا در انتقال'), 'error');
    }).fail(function(xhr){ $btn.prop('disabled', false).text('انتقال به سرپرست'); snNotice('#sn-manager-assign-notice', '❌ خطای سرور: '+xhr.status, 'error'); });
  });
  $(document).on('click','.sn-fin-approve',function(e){e.preventDefault();var id=$(this).data('id'),$btn=$(this),$row=$btn.closest('tr');$btn.prop('disabled',true).text('در حال تایید...');$.post(ajax,{action:'sn_financial_approve_payment',nonce:nonce,invoice_id:id},function(res){if(res&&res.success){$row.find('td').eq(5).text((res.status_label||'تایید شده'));$row.find('td').last().html('<span class="sn-notice sn-success">تایید شد</span>');}else{$btn.prop('disabled',false).text('تایید');alert((res&&res.message)||'خطا');}}).fail(function(xhr){$btn.prop('disabled',false).text('تایید');alert('خطای سرور: '+xhr.status);});});
  $(document).on('click','.sn-fin-reject',function(e){e.preventDefault();var id=$(this).data('id'),reason=prompt('دلیل رد پرداخت را وارد کنید:');if(!reason)return;var $btn=$(this),$row=$btn.closest('tr');$btn.prop('disabled',true).text('در حال رد...');$.post(ajax,{action:'sn_financial_reject_payment',nonce:nonce,invoice_id:id,reason:reason},function(res){if(res&&res.success){$row.find('td').eq(5).text((res.status_label||'رد شده'));$row.find('td').last().html('<span class="sn-notice sn-error">رد شد</span>');}else{$btn.prop('disabled',false).text('رد');alert((res&&res.message)||'خطا');}}).fail(function(xhr){$btn.prop('disabled',false).text('رد');alert('خطای سرور: '+xhr.status);});});
})(jQuery);

/* Controlled repeat-action invoice forms are injected by the lazy supervisor tab. */
(function($){
  'use strict';
  if (window.snPaidReferralInvoiceFormsReady) { return; }
  window.snPaidReferralInvoiceFormsReady = true;

  function enDigits(value){
    return String(value == null ? '' : value)
      .replace(/[۰-۹]/g,function(d){ return '۰۱۲۳۴۵۶۷۸۹'.indexOf(d); })
      .replace(/[٠-٩]/g,function(d){ return '٠١٢٣٤٥٦٧٨٩'.indexOf(d); });
  }
  function amount(value){
    var normalized = enDigits(value).replace(/[٬,،\s]/g,'').replace(/[^0-9.]/g,'');
    var parsed = Number(normalized || 0);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  function showError($form, message){
    var $notice = $form.children('.sn-paid-referral-form-notice');
    if (!$notice.length) { $notice = $('<div class="sn-notice sn-error sn-paid-referral-form-notice sn-full"></div>').prependTo($form); }
    $notice.text(message);
  }
  function markBusy($form, label){
    $form.data('busy',1).find('button[type="submit"]').prop('disabled',true).attr('aria-busy','true').text(label);
  }

  $(document).off('submit.snPaidReferralCreate','.sn-paid-referral-create-form').on('submit.snPaidReferralCreate','.sn-paid-referral-create-form',function(e){
    var $form=$(this), productId=parseInt($form.find('select[name="product_ids[]"]').val(),10)||0;
    if ($form.data('busy')) { e.preventDefault(); return; }
    if (!productId) { e.preventDefault(); showError($form,'ابتدا محصول جدید را انتخاب کنید.'); return; }
    markBusy($form,'در حال صدور فاکتور...');
  });

  $(document).off('submit.snPaymentStage','.sn-payment-stage-form').on('submit.snPaymentStage','.sn-payment-stage-form',function(e){
    var $form=$(this), mode=String($form.find('.sn-stage-mode').val()||'remaining');
    if ($form.data('busy')) { e.preventDefault(); return; }
    if (mode === 'partial') {
      var choice=String($form.find('.sn-stage-preset').val()||''), value=choice === 'custom' ? amount($form.find('[name="custom_amount"]').val()) : amount(choice);
      var remaining=amount($form.attr('data-sn-remaining'));
      if (!choice || value <= 0) { e.preventDefault(); showError($form,'مبلغ این مرحله را انتخاب یا وارد کنید.'); return; }
      if (remaining > 0 && value > remaining) { e.preventDefault(); showError($form,'مبلغ این مرحله نمی‌تواند بیشتر از مانده فاکتور باشد.'); return; }
    }
    markBusy($form,'در حال صدور مرحله...');
  });
})(jQuery);

/* Direct subscription selected by a seller: supervisor may change it pre-payment. */
(function($){
  'use strict';
  var cfg=window.snAjax||window.snData||{}, ajax=cfg.ajaxurl||window.ajaxurl||'/wp-admin/admin-ajax.php';
  function esc(v){return String(v==null?'':v).replace(/[&<>"']/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#039;'}[c];});}
  $(document).off('submit.snSupervisorSubscription','.sn-supervisor-subscription-change').on('submit.snSupervisorSubscription','.sn-supervisor-subscription-change',function(e){
    e.preventDefault();
    var $form=$(this),$btn=$form.find('button[type="submit"]'),$notice=$form.closest('.sn-card').find('.sn-supervisor-subscription-notice').first();
    if($form.data('busy')) return;
    $form.data('busy',1);$btn.prop('disabled',true).text('در حال ذخیره...');
    $.ajax({url:ajax,type:'POST',dataType:'json',data:$form.serialize()}).done(function(res){
      var payload=res&&res.data?res.data:{};
      $notice.html('<div class="sn-notice '+(res&&res.success?'sn-success':'sn-error')+'">'+esc(payload.message||(res&&res.success?'اشتراک تغییر کرد.':'تغییر اشتراک انجام نشد.'))+'</div>');
      if(res&&res.success){window.setTimeout(function(){var $tab=$('#sn-tab-needs-action');$tab.removeData('loaded').attr('data-loaded','0');$('#sn-seller-supervisor-panel .sn-tab[data-tab="needs-action"],#sn-supervisor-panel .sn-tab[data-tab="needs-action"]').first().trigger('click');},700);}
    }).fail(function(xhr){var payload=xhr&&xhr.responseJSON&&xhr.responseJSON.data?xhr.responseJSON.data:{};$notice.html('<div class="sn-notice sn-error">'+esc(payload.message||'ارتباط با سرور انجام نشد.')+'</div>');}).always(function(){$form.data('busy',0);$btn.prop('disabled',false).text('ثبت تغییر');});
  });
})(jQuery);


/* SN Final AJAX/Persian/Jalali fixes - no refresh operations */
(function($){
  'use strict';
  window.snAjax = window.snAjax || window.snData || {};
  window.snData = window.snData || window.snAjax;
  var ajax = window.snAjax.ajaxurl || (window.ajaxurl || '/wp-admin/admin-ajax.php');
  var nonce = window.snAjax.nonce || window.snData.nonce || '';
  $(function(){ $('#sn-submit-manual-payment,#sn-card-manual-toggle').off('click'); });

  function esc(v){ return $('<div>').text(v == null || v === '' ? '—' : v).html(); }
  function toFaDigits(v){ return String(v).replace(/\d/g,function(d){return '۰۱۲۳۴۵۶۷۸۹'[d];}); }
  function toEnDigits(v){ return String(v).replace(/[۰-۹]/g,function(d){return '۰۱۲۳۴۵۶۷۸۹'.indexOf(d);}).replace(/[٠-٩]/g,function(d){return '٠١٢٣٤٥٦٧٨٩'.indexOf(d);}); }
  function gregorianToJalali(gy, gm, gd) { var gdm=[0,31,59,90,120,151,181,212,243,273,304,334], gy2=(gm>2)?gy+1:gy; var days=355666+(365*gy)+Math.floor((gy2+3)/4)-Math.floor((gy2+99)/100)+Math.floor((gy2+399)/400)+gd+gdm[gm-1]; var jy=-1595+33*Math.floor(days/12053); days%=12053; jy+=4*Math.floor(days/1461); days%=1461; if(days>365){jy+=Math.floor((days-1)/365); days=(days-1)%365;} var jm=(days<186)?1+Math.floor(days/31):7+Math.floor((days-186)/30); var jd=1+((days<186)?days%31:(days-186)%30); return [jy,jm,jd]; }
  function currentJalali(){ var p=snTehranNowParts(), j=gregorianToJalali(p.year,p.month,p.day); return {jy:j[0], jm:j[1], jd:j[2], hh:p.hour, mi:p.minute}; }
  function pad(n){ return String(n).padStart(2,'0'); }

  function buildJalaliPicker(){
    var $host=$('#sn-card-paid-at-picker'); if(!$host.length || $host.data('ready')) return;
    var now=currentJalali(), years='', months='', days='', hours='', mins='', rounded=now.mi;
    for(var y=now.jy-1;y<=now.jy+1;y++) years+='<option value="'+y+'" '+(y===now.jy?'selected':'')+'>'+toFaDigits(y)+'</option>';
    for(var m=1;m<=12;m++) months+='<option value="'+m+'" '+(m===now.jm?'selected':'')+'>'+toFaDigits(m)+'</option>';
    for(var d=1;d<=31;d++) days+='<option value="'+d+'" '+(d===now.jd?'selected':'')+'>'+toFaDigits(d)+'</option>';
    for(var h=0;h<24;h++) hours+='<option value="'+h+'" '+(h===now.hh?'selected':'')+'>'+toFaDigits(pad(h))+'</option>';
    for(var i=0;i<60;i++) mins+='<option value="'+i+'" '+(rounded===i?'selected':'')+'>'+toFaDigits(pad(i))+'</option>';
    $host.html('<div class="sn-jalali-row"><select id="sn-paid-jy">'+years+'</select><span>/</span><select id="sn-paid-jm">'+months+'</select><span>/</span><select id="sn-paid-jd">'+days+'</select><span class="sn-time-sep">ساعت</span><select id="sn-paid-hh">'+hours+'</select><span>:</span><select id="sn-paid-mi">'+mins+'</select></div><button type="button" class="sn-btn sn-btn-sm sn-payment-picker-today">امروز تهران</button><small>تاریخ و ساعت واریز را به شمسی انتخاب کنید.</small>').data('ready',1);
    syncJalaliPicker();
  }
  function syncJalaliPicker(){
    if(!$('#sn-card-paid-at-picker').length) return;
    var jy=$('#sn-paid-jy').val(), jm=pad($('#sn-paid-jm').val()), jd=pad($('#sn-paid-jd').val()), hh=pad($('#sn-paid-hh').val()), mi=pad($('#sn-paid-mi').val());
    $('#sn-card-paid-at').val(toFaDigits(jy+'/'+jm+'/'+jd+' '+hh+':'+mi));
  }
  $(document).on('change','#sn-paid-jy,#sn-paid-jm,#sn-paid-jd,#sn-paid-hh,#sn-paid-mi',syncJalaliPicker);
  $(document).on('click','.sn-payment-picker-today',function(){var n=currentJalali();$('#sn-paid-jy').val(n.jy);$('#sn-paid-jm').val(n.jm);$('#sn-paid-jd').val(n.jd);$('#sn-paid-hh').val(n.hh);$('#sn-paid-mi').val(n.mi);syncJalaliPicker();});
  $(document).on('click','#sn-pay-card,#sn-card-manual-toggle',function(){ $('#sn-card-manual-fields').slideDown(); setTimeout(buildJalaliPicker,30); });
  $(function(){ buildJalaliPicker(); });

  $(document).off('click', '#sn-submit-manual-payment').on('click', '#sn-submit-manual-payment', function(e){
    e.preventDefault(); buildJalaliPicker(); syncJalaliPicker();
    var code = $('#sn-invoice-page').data('active-code') || $('#sn-inv-code').val();
    var from = toEnDigits($('#sn-card-from4').val());
    var to = toEnDigits($('#sn-card-to4').val());
    var amount = toEnDigits($('#sn-card-amount').val()).replace(/,/g,'');
    if(!/^\d{4}$/.test(from)){ alert('۴ رقم آخر کارت مبدا باید عددی باشد'); return; }
    if(to!=='' && !/^\d{4}$/.test(to)){ alert('۴ رقم کارت مقصد اختیاری است؛ در صورت ورود باید ۴ رقم باشد'); return; }
    if(!amount || isNaN(amount)){ alert('مبلغ باید عدد باشد'); return; }
    var data={action:'sn_submit_manual_payment',nonce:nonce,invoice_code:code,card_from:from,card_to:to,amount:amount,paid_at:$('#sn-card-paid-at').val(),paid_jy:$('#sn-paid-jy').val(),paid_jm:$('#sn-paid-jm').val(),paid_jd:$('#sn-paid-jd').val(),paid_hh:$('#sn-paid-hh').val(),paid_mi:$('#sn-paid-mi').val()};
    var $btn=$(this).prop('disabled',true).text('در حال ثبت...');
    $.post(ajax,data,function(res){
      $btn.prop('disabled',false).text('ثبت اطلاعات واریز');
      if(res&&res.success){ $('#sn-payment-section').hide(); $('.sn-manual-result').remove(); $('<div class="sn-notice sn-success sn-manual-result">✅ '+esc(res.message||'اطلاعات واریز ثبت شد')+'</div>').insertBefore('#sn-payment-section'); if(typeof window.loadInvoice==='function') window.loadInvoice(code); }
      else alert((res&&res.message)||'خطا در ثبت اطلاعات واریز');
    }).fail(function(xhr){ $btn.prop('disabled',false).text('ثبت اطلاعات واریز'); alert('خطای سرور: '+xhr.status); });
  });

  $(document).off('click','.sn-fin-approve').on('click','.sn-fin-approve',function(e){
    e.preventDefault(); var id=$(this).data('id'), $btn=$(this), $row=$btn.closest('tr');
    $btn.prop('disabled',true).text('در حال تایید...');
    $.post(ajax,{action:'sn_financial_approve_payment',nonce:nonce,invoice_id:id},function(res){
      if(res&&res.success){ $row.addClass('sn-row-done'); $row.find('td').eq(5).text('تایید شده'); $row.find('td').last().html('<span class="sn-notice sn-success">✅ تایید شد</span>'); }
      else { $btn.prop('disabled',false).text('تایید'); alert((res&&res.message)||'خطا'); }
    }).fail(function(xhr){ $btn.prop('disabled',false).text('تایید'); alert('خطای سرور: '+xhr.status); });
  });
  $(document).off('click','.sn-fin-reject').on('click','.sn-fin-reject',function(e){
    e.preventDefault(); var reason=prompt('دلیل رد پرداخت را وارد کنید:'); if(!reason) return; var id=$(this).data('id'), $btn=$(this), $row=$btn.closest('tr');
    $btn.prop('disabled',true).text('در حال رد...');
    $.post(ajax,{action:'sn_financial_reject_payment',nonce:nonce,invoice_id:id,reason:reason},function(res){
      if(res&&res.success){ $row.addClass('sn-row-rejected'); $row.find('td').eq(5).text('رد شده'); $row.find('td').last().html('<span class="sn-notice sn-error">❌ رد شد</span>'); }
      else { $btn.prop('disabled',false).text('رد'); alert((res&&res.message)||'خطا'); }
    }).fail(function(xhr){ $btn.prop('disabled',false).text('رد'); alert('خطای سرور: '+xhr.status); });
  });

  function translatePanelText(){
    var map={'invoice':'پیش‌فاکتور','invoices':'پیش‌فاکتورها','assigned':'تخصیص داده‌شده','unassigned':'بدون تخصیص','pending':'در انتظار','approved':'تایید شده','rejected':'رد شده','paid':'پرداخت‌شده','online':'پرداخت آنلاین','card':'کارت به کارت'};
    $('.sn-panel, .sn-invoice-page').find('td,th,span,strong,option,button,label,small').contents().filter(function(){return this.nodeType===3;}).each(function(){ var t=this.nodeValue; Object.keys(map).forEach(function(k){ t=t.replace(new RegExp('\\b'+k+'\\b','g'),map[k]); }); this.nodeValue=t; });
  }
  $(document).ajaxComplete(function(){ setTimeout(translatePanelText,30); });
  $(translatePanelText);
})(jQuery);

/* SN Verified Final Fixes: AJAX receipt/manual payment + real Jalali picker + Persian labels */
(function($){
  'use strict';
  window.snAjax = window.snAjax || window.snData || {};
  window.snData = window.snData || window.snAjax;
  var ajax = window.snAjax.ajaxurl || window.snData.ajaxurl || window.ajaxurl || '/wp-admin/admin-ajax.php';
  var nonce = window.snAjax.nonce || window.snData.nonce || '';

  function esc(v){ return $('<div>').text(v == null || v === '' ? '—' : v).html(); }
  function faDigits(v){ return String(v).replace(/\d/g,function(d){return '۰۱۲۳۴۵۶۷۸۹'[d];}); }
  function enDigits(v){ return String(v).replace(/[۰-۹]/g,function(d){return '۰۱۲۳۴۵۶۷۸۹'.indexOf(d);}).replace(/[٠-٩]/g,function(d){return '٠١٢٣٤٥٦٧٨٩'.indexOf(d);}); }
  function pad(n){ return String(n).padStart(2,'0'); }
  function faStatus(v){
    var m={
      pre_invoice:'پیش‌فاکتور', pending:'در انتظار پرداخت', pending_payment:'در انتظار پرداخت',
      receipt_uploaded:'نیاز به بررسی فیش', pending_financial_approval:'نیاز به بررسی فیش',
      paid:'پرداخت‌شده درگاهی', approved:'تایید شده مالی', rejected:'رد شده', cancelled:'لغوشده',
      assigned:'تخصیص داده‌شده', unassigned:'بدون تخصیص', supervisor_pool:'در پنل سرپرست', invoiced:'پیش‌فاکتور صادر شده',
      online:'پرداخت آنلاین', gateway:'درگاه پرداخت', card:'کارت به کارت', customer_upload:'ثبت توسط مشتری', supervisor_upload:'ثبت توسط سرپرست'
    };
    return m[v] || v || '—';
  }
  function gregorianToJalali(gy, gm, gd) {
    var gdm=[0,31,59,90,120,151,181,212,243,273,304,334], gy2=(gm>2)?gy+1:gy;
    var days=355666+(365*gy)+Math.floor((gy2+3)/4)-Math.floor((gy2+99)/100)+Math.floor((gy2+399)/400)+gd+gdm[gm-1];
    var jy=-1595+33*Math.floor(days/12053); days%=12053; jy+=4*Math.floor(days/1461); days%=1461;
    if(days>365){jy+=Math.floor((days-1)/365); days=(days-1)%365;}
    var jm=(days<186)?1+Math.floor(days/31):7+Math.floor((days-186)/30);
    var jd=1+((days<186)?days%31:(days-186)%30);
    return [jy,jm,jd];
  }
  function todayJalali(){ var p=snTehranNowParts(), j=gregorianToJalali(p.year,p.month,p.day); return {jy:j[0],jm:j[1],jd:j[2],hh:p.hour,mi:Math.floor(p.minute/5)*5}; }

  function buildVerifiedJalaliPicker(){
    var $host=$('#sn-card-paid-at-picker');
    if(!$host.length) return;
    var now=todayJalali(), years='', months='', days='', hours='', mins='';
    for(var y=now.jy-2;y<=now.jy+1;y++) years+='<option value="'+y+'" '+(y===now.jy?'selected':'')+'>'+faDigits(y)+'</option>';
    for(var m=1;m<=12;m++) months+='<option value="'+m+'" '+(m===now.jm?'selected':'')+'>'+faDigits(m)+'</option>';
    for(var d=1;d<=31;d++) days+='<option value="'+d+'" '+(d===now.jd?'selected':'')+'>'+faDigits(d)+'</option>';
    for(var h=0;h<24;h++) hours+='<option value="'+h+'" '+(h===now.hh?'selected':'')+'>'+faDigits(pad(h))+'</option>';
    for(var i=0;i<60;i++) mins+='<option value="'+i+'" '+(i===now.mi?'selected':'')+'>'+faDigits(pad(i))+'</option>';
    $host.html('<div class="sn-jalali-picker-box"><div class="sn-jalali-title">انتخاب تاریخ شمسی و ساعت تهران واریز</div><div class="sn-jalali-row"><label>سال<select id="sn-paid-jy">'+years+'</select></label><label>ماه<select id="sn-paid-jm">'+months+'</select></label><label>روز<select id="sn-paid-jd">'+days+'</select></label><label>ساعت تهران<select id="sn-paid-hh">'+hours+'</select></label><label>دقیقه<select id="sn-paid-mi">'+mins+'</select></label></div><div class="sn-jalali-selected">تاریخ انتخاب‌شده: <strong id="sn-paid-at-view"></strong></div></div>');
    syncVerifiedJalaliPicker();
  }
  function syncVerifiedJalaliPicker(){
    if(!$('#sn-paid-jy').length) return;
    var val=$('#sn-paid-jy').val()+'/'+pad($('#sn-paid-jm').val())+'/'+pad($('#sn-paid-jd').val())+' '+pad($('#sn-paid-hh').val())+':'+pad($('#sn-paid-mi').val());
    $('#sn-card-paid-at').val(faDigits(val));
    $('#sn-paid-at-view').text(faDigits(val));
  }
  $(document).on('change','#sn-paid-jy,#sn-paid-jm,#sn-paid-jd,#sn-paid-hh,#sn-paid-mi',syncVerifiedJalaliPicker);

  function showCardPaymentBox(){
    $('#sn-card-info').slideDown();
    $('#sn-card-manual-toggle').hide();
    $('#sn-card-manual-fields').slideDown();
    buildVerifiedJalaliPicker();
    syncVerifiedJalaliPicker();
  }

  function refreshInvoiceBox(code){
    if(!code) code = $('#sn-invoice-page').data('active-code') || $('#sn-inv-code').val();
    if(!code) return;
    $.post(ajax,{action:'sn_invoice_info',nonce:nonce,invoice_code:code},function(res){
      if(!res || !res.success || !res.invoice) return;
      var inv=res.invoice, card=res.card||{};
      $('#sn-inv-display-code').text(inv.code||'');
      $('#sn-inv-name').text(inv.customer_name||'');
      $('#sn-inv-phone').text(inv.customer_phone||'');
      $('#sn-inv-location').text([inv.province,inv.city].filter(Boolean).join(' — ') || '—');
      $('#sn-inv-product').text(inv.product_name||'');
      $('#sn-inv-price').text(inv.price_fmt||'');
      $('#sn-inv-status').text(inv.status_label || faStatus(inv.status));
      $('#sn-card-number').text(card.number||'—');
      $('#sn-card-owner').text(card.owner||'—');
      $('#sn-invoice-lookup').hide(); $('#sn-invoice-detail').show();
      $('#sn-invoice-page').data('active-code', code).data('active-price', inv.product_price||0);
      if(inv.status==='pending_financial_approval' || inv.status==='receipt_uploaded'){
        $('#sn-payment-section').hide();
        $('#sn-inv-paid-msg').show().removeClass('sn-success sn-error').addClass('sn-info').text('پرداخت/فیش ثبت شده و در وضعیت «نیاز به بررسی فیش» قرار دارد.');
      } else if(inv.status==='paid' || inv.status==='approved'){
        $('#sn-payment-section').hide();
        $('#sn-inv-paid-msg').show().removeClass('sn-error sn-info').addClass('sn-success').text('این فاکتور پرداخت و تایید شده است.');
      } else if(inv.status==='rejected'){
        $('#sn-payment-section').show();
        $('#sn-inv-paid-msg').show().removeClass('sn-success sn-info').addClass('sn-error').text('پرداخت قبلی رد شده است. دوباره فیش یا اطلاعات واریزی را ثبت کنید.');
      } else {
        $('#sn-payment-section').show(); $('#sn-inv-paid-msg').hide();
      }
    });
  }
  window.snRefreshInvoiceBox = refreshInvoiceBox;

  $(function(){
    $('#sn-upload-receipt,#sn-pay-card,#sn-card-manual-toggle,#sn-submit-manual-payment').off('click');
    if($('#sn-card-paid-at-picker').length) buildVerifiedJalaliPicker();
  });

  $(document).off('click.snVerifiedCard','#sn-pay-card,#sn-card-manual-toggle').on('click.snVerifiedCard','#sn-pay-card,#sn-card-manual-toggle',function(e){
    e.preventDefault(); showCardPaymentBox();
  });

  $(document).off('click.snVerifiedReceipt','#sn-upload-receipt').on('click.snVerifiedReceipt','#sn-upload-receipt',function(e){
    e.preventDefault();
    var code=$('#sn-invoice-page').data('active-code') || $('#sn-inv-code').val();
    var input=$('#sn-receipt-file').get(0);
    var file=input && input.files ? input.files[0] : null;
    if(!code){ alert('کد فاکتور مشخص نیست'); return; }
    if(!file){ alert('لطفاً فایل فیش را انتخاب کنید'); return; }
    var fd=new FormData(); fd.append('action','sn_upload_receipt'); fd.append('nonce',nonce); fd.append('invoice_code',code); fd.append('receipt',file);
    var $btn=$(this).prop('disabled',true).text('در حال ارسال...');
    $.ajax({url:ajax,type:'POST',data:fd,processData:false,contentType:false,success:function(res){
      $btn.prop('disabled',false).text('ارسال فیش');
      if(res&&res.success){
        $('.sn-upload-result').remove();
        $('<div class="sn-notice sn-success sn-upload-result">✅ '+esc(res.message||'فیش ثبت شد')+'</div>').insertBefore('#sn-payment-section');
        refreshInvoiceBox(code);
      } else alert((res&&res.message)||'خطا در آپلود فیش');
    },error:function(xhr){ $btn.prop('disabled',false).text('ارسال فیش'); alert('خطای سرور: '+xhr.status); }});
  });

  $(document).off('click.snVerifiedManual','#sn-submit-manual-payment').on('click.snVerifiedManual','#sn-submit-manual-payment',function(e){
    e.preventDefault(); buildVerifiedJalaliPicker(); syncVerifiedJalaliPicker();
    var code=$('#sn-invoice-page').data('active-code') || $('#sn-inv-code').val();
    var from=enDigits($('#sn-card-from4').val()), to=enDigits($('#sn-card-to4').val()), amount=enDigits($('#sn-card-amount').val()).replace(/,/g,'');
    if(!code){ alert('کد فاکتور مشخص نیست'); return; }
    if(!/^\d{4}$/.test(from)){ alert('۴ رقم آخر کارت مبدا باید عددی باشد'); return; }
    if(to!=='' && !/^\d{4}$/.test(to)){ alert('۴ رقم کارت مقصد اختیاری است؛ در صورت ورود باید ۴ رقم باشد'); return; }
    if(!amount || isNaN(amount)){ alert('مبلغ باید عدد باشد'); return; }
    var data={action:'sn_submit_manual_payment',nonce:nonce,invoice_code:code,card_from:from,card_to:to,amount:amount,paid_at:$('#sn-card-paid-at').val(),paid_jy:$('#sn-paid-jy').val(),paid_jm:$('#sn-paid-jm').val(),paid_jd:$('#sn-paid-jd').val(),paid_hh:$('#sn-paid-hh').val(),paid_mi:$('#sn-paid-mi').val()};
    var $btn=$(this).prop('disabled',true).text('در حال ثبت...');
    $.post(ajax,data,function(res){
      $btn.prop('disabled',false).text('ثبت اطلاعات واریز');
      if(res&&res.success){
        $('.sn-manual-result').remove();
        $('<div class="sn-notice sn-success sn-manual-result">✅ '+esc(res.message||'اطلاعات واریز ثبت شد')+'</div>').insertBefore('#sn-payment-section');
        refreshInvoiceBox(code);
      } else alert((res&&res.message)||'خطا در ثبت اطلاعات واریز');
    }).fail(function(xhr){ $btn.prop('disabled',false).text('ثبت اطلاعات واریز'); alert('خطای سرور: '+xhr.status); });
  });

  function translateAllLabels(){
    var map={'invoice':'پیش‌فاکتور','invoices':'پیش‌فاکتورها','assigned':'تخصیص داده‌شده','unassigned':'بدون تخصیص','pending':'در انتظار','approved':'تایید شده','rejected':'رد شده','paid':'پرداخت‌شده','online':'پرداخت آنلاین','gateway':'درگاه پرداخت','card':'کارت به کارت','receipt_uploaded':'نیاز به بررسی فیش','pending_financial_approval':'نیاز به بررسی فیش'};
    $('.sn-panel, .sn-invoice-page, .sn-admin').find('td,th,span,strong,option,button,label,small,h1,h2,h3,h4,p').contents().filter(function(){return this.nodeType===3;}).each(function(){
      var t=this.nodeValue; Object.keys(map).forEach(function(k){ t=t.replace(new RegExp('\\b'+k+'\\b','g'),map[k]); }); this.nodeValue=t;
    });
  }
  // 1.0.120: disabled expensive full-DOM text translation on every Ajax.
  // Supervisor UI is already Persian; this scan was a visible source of tab lag on large tables.
})(jQuery);

/* SN 1.0.128 supervisor invoice tab: restored manager-like UI with bounded Ajax */
(function($){
  'use strict';
  if(!$('#sn-supervisor-panel,#sn-senior-supervisor-panel').length) return;
  var cfg = window.snAjax || window.snData || {};
  var ajax = cfg.ajaxurl || window.ajaxurl || '/wp-admin/admin-ajax.php';
  var nonce = cfg.nonce || cfg.admin_nonce || '';
  var invoiceCache = {};
  var invoicePending = null;
	var supervisorPaymentInvoiceCode = '';
	var supervisorReceiptInvoiceId = '';
  function esc(v){return String(v==null?'':v).replace(/[&<>"']/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#039;'}[c]||c;});}
	function digits(v){return String(v||'').replace(/[۰-۹]/g,function(d){return String('۰۱۲۳۴۵۶۷۸۹'.indexOf(d));}).replace(/[٠-٩]/g,function(d){return String('٠١٢٣٤٥٦٧٨٩'.indexOf(d));});}
  function num(v){ return Number(String(v||0).replace(/[^0-9.-]/g,'')) || 0; }
  function money(v){try{return num(v).toLocaleString('fa-IR')+' تومان';}catch(e){return String(v||0)+' تومان';}}
  function rawStatus(i){ return String((i&&i.payment_status)|| (i&&i.invoice_status) || (i&&i.status) || '').toLowerCase(); }
  function isPaid(i){ var st=rawStatus(i); return ['paid','approved','financial_approved','gateway_paid','online_paid','completed','success'].indexOf(st)!==-1; }
  function amountOf(i){ return num((i&&i.final_total) || (i&&i.product_price) || (i&&i.paid_amount) || 0); }
  function updateKpis(rows, summary){
    rows = rows || [];
    var total = summary && summary.total != null ? num(summary.total) : rows.length;
    var paidCount = summary && summary.paid_count != null ? num(summary.paid_count) : rows.filter(isPaid).length;
    var amount = summary && summary.amount != null ? num(summary.amount) : rows.reduce(function(a,i){return a+amountOf(i);},0);
    var paidAmount = summary && summary.paid_amount != null ? num(summary.paid_amount) : rows.reduce(function(a,i){return a+(isPaid(i)?amountOf(i):0);},0);
    $('#sn-supervisor-invoice-kpi-total').text(total.toLocaleString('fa-IR'));
    $('#sn-supervisor-invoice-kpi-paid-count').text(paidCount.toLocaleString('fa-IR'));
    $('#sn-supervisor-invoice-kpi-amount').text(money(amount));
    $('#sn-supervisor-invoice-kpi-paid-amount').text(money(paidAmount));
  }
  function receiptUrlOf(i){ return String((i&&i.receipt_url) || (i&&i.receipt_file) || ''); }
  function canSupervisorUploadReceipt(i){
    if(i && typeof i.can_upload_receipt !== 'undefined') return !!i.can_upload_receipt;
    var st = rawStatus(i);
    return !isPaid(i) && [
      'pre_invoice','pending','pending_payment','awaiting_payment','partial_paid',
      'rejected','financial_rejected','payment_rejected','recontact_requested',
      'receipt_uploaded','pending_financial_approval','needs_finance_review'
    ].indexOf(st) !== -1;
  }
	function ensureSupervisorPaymentModal(){
	  if($('#sn-supervisor-manual-payment-modal').length) return;
	  $('body').append('<div id="sn-supervisor-manual-payment-modal" class="sn-modal sn-lite-modal sn-payment-entry-modal" aria-hidden="true" style="display:none"><div class="sn-modal-backdrop sn-supervisor-payment-close"></div><div class="sn-modal-card sn-payment-entry-card" role="dialog" aria-modal="true">'+
		'<div class="sn-modal-head sn-payment-entry-head"><div class="sn-payment-entry-title"><span class="sn-payment-entry-icon">↙</span><div><h3>ثبت اطلاعات واریز</h3><p>اطلاعات پرداخت مشتری را دقیق وارد کنید</p></div></div><button type="button" class="sn-modal-x sn-supervisor-payment-close" aria-label="بستن">×</button></div>'+
		'<div class="sn-modal-body"><div class="sn-payment-entry-grid"><label><span>مبلغ این مرحله (تومان)</span><input id="sn-supervisor-payment-amount" inputmode="numeric" readonly></label><label><span>۴ رقم آخر کارت مبدا</span><input id="sn-supervisor-payment-card-from" inputmode="numeric" maxlength="4" autocomplete="off" placeholder="مثلاً ۱۲۳۴"></label><label><span>۴ رقم آخر کارت مقصد (اختیاری)</span><input id="sn-supervisor-payment-card-to" inputmode="numeric" maxlength="4" autocomplete="off" placeholder="اختیاری؛ مثلاً ۵۶۷۸"></label><label><span>تاریخ شمسی واریز</span><input id="sn-supervisor-payment-date" class="sn-jalali-date" placeholder="۱۴۰۵/۰۷/۰۱"><button type="button" class="sn-btn sn-btn-sm sn-payment-today" data-date="#sn-supervisor-payment-date" data-hour="#sn-supervisor-payment-hour" data-minute="#sn-supervisor-payment-minute">امروز تهران</button></label><label><span>ساعت تهران (۲۴ ساعته)</span><span class="sn-payment-time-select"><select id="sn-supervisor-payment-hour"></select> : <select id="sn-supervisor-payment-minute"></select></span></label><label><span>تصویر فیش (اختیاری)</span><input id="sn-supervisor-payment-receipt" type="file" accept="image/*,.pdf,application/pdf"></label></div><div id="sn-supervisor-payment-msg" class="sn-payment-entry-message"></div><div class="sn-modal-actions sn-payment-entry-actions"><button type="button" class="sn-btn sn-btn-secondary sn-supervisor-payment-close">انصراف</button><button type="button" class="sn-btn sn-btn-primary" id="sn-supervisor-payment-submit">ثبت واریز و فیش</button></div></div></div></div>');
	}
	function ensureSupervisorReceiptModal(){
	  if($('#sn-supervisor-receipt-modal').length) return;
	  $('body').append('<div id="sn-supervisor-receipt-modal" class="sn-modal sn-lite-modal sn-payment-entry-modal" aria-hidden="true" style="display:none"><div class="sn-modal-backdrop sn-supervisor-receipt-close"></div><div class="sn-modal-card sn-payment-entry-card sn-payment-proof-card" role="dialog" aria-modal="true">'+
		'<div class="sn-modal-head sn-payment-entry-head"><div class="sn-payment-entry-title"><span class="sn-payment-entry-icon">⌁</span><div><h3>بارگذاری فیش پرداخت</h3><p>فیش را انتخاب کنید؛ ۴ رقم کارت مقصد اختیاری است</p></div></div><button type="button" class="sn-modal-x sn-supervisor-receipt-close" aria-label="بستن">×</button></div>'+
		'<div class="sn-modal-body"><div class="sn-payment-entry-grid sn-payment-proof-grid"><label><span>۴ رقم آخر کارت مقصد (اختیاری)</span><input id="sn-supervisor-receipt-card-to" inputmode="numeric" maxlength="4" autocomplete="off" placeholder="اختیاری؛ مثلاً ۵۶۷۸"></label><label class="sn-payment-file-field"><span>فایل فیش</span><input id="sn-supervisor-receipt-file" type="file" accept="image/*,.pdf"><small>تصویر یا PDF</small></label></div><div id="sn-supervisor-receipt-msg" class="sn-payment-entry-message" aria-live="polite"></div><div class="sn-modal-actions sn-payment-entry-actions"><button type="button" class="sn-btn sn-btn-secondary sn-supervisor-receipt-close">انصراف</button><button type="button" class="sn-btn sn-btn-primary sn-supervisor-upload-receipt">ارسال فیش</button></div></div></div></div>');
	}
  function render(rows, hasMore, summary){
    rows = rows || [];
    updateKpis(rows, summary || {});
    var html='<div class="sn-table-wrap"><table class="sn-table"><thead><tr><th>کد</th><th>فروشنده</th><th>مشتری</th><th>مبلغ کل / پرداخت‌شده / مانده</th><th>روش پرداخت</th><th>وضعیت</th><th>فیش پرداخت</th><th>اقدام مجدد</th><th>چت بیاوین</th><th>تاریخ</th></tr></thead><tbody>';
    if(!rows.length){ html+='<tr><td colspan="10"><p class="sn-notice">فاکتوری در این تب وجود ندارد.</p></td></tr>'; }
	rows.forEach(function(i){
		var date = i.created_at_jalali || i.approved_at_jalali || i.paid_at_jalali || i.updated_at_jalali || i.date_jalali || '—';
      var rurl = receiptUrlOf(i);
      var receiptHtml = '';
	  var giftTag = i.validation_gifted ? '<span class="sn-dot-validation-gift-tag">اعتبارسنجی هدیه</span>' : '';
	  var actionHtml = esc(i.repeat_action_label||i.next_action_label||'—');
	  if(i.can_gift_validation || i.can_resend_gift_validation){
	    var giftButtonLabel=i.can_resend_gift_validation?'تلاش مجدد ارسال لینک':'ارسال لینک انتخاب';
	    actionHtml = '<button type="button" class="sn-btn sn-btn-primary sn-btn-mini sn-dot-gift-validation" data-invoice-id="'+esc(i.id||'')+'">'+esc(giftButtonLabel)+'</button><small class="sn-dot-gift-validation-msg" aria-live="polite"></small>';
	  } else if(i.validation_gifted){
	    actionHtml = '<span class="sn-dot-validation-gift-tag">اعتبارسنجی هدیه</span><small>لینک انتخاب فعال شده است</small>';
	  }
	  actionHtml += '<div class="sn-invoice-sms-action"><button type="button" class="sn-btn sn-btn-secondary sn-btn-mini sn-resend-invoice-sms" data-invoice-id="'+esc(i.id||'')+'">ارسال مجدد لینک پرداخت</button><small class="sn-resend-invoice-sms-msg" aria-live="polite"></small><button type="button" class="sn-btn sn-btn-secondary sn-btn-mini sn-copy-invoice-link" data-invoice-id="'+esc(i.id||'')+'">کپی لینک پرداخت</button><small class="sn-copy-invoice-link-msg" aria-live="polite"></small></div>';
	  if(Number((i.payment_summary||{}).paid||0)===0 && Number((i.payment_summary||{}).stage_no||1)===1) actionHtml += '<button type="button" class="sn-btn sn-btn-secondary sn-btn-mini sn-edit-invoice" data-invoice-id="'+esc(i.id||'')+'">ویرایش پیش از پرداخت</button>';
	  actionHtml += '<button type="button" class="sn-btn sn-btn-secondary sn-btn-mini sn-invoice-finance-history" data-invoice-id="'+esc(i.id||'')+'">وضعیت و تاریخچه مالی</button>';
      if(rurl){ receiptHtml += '<a class="sn-btn sn-btn-secondary sn-btn-mini" href="'+esc(rurl)+'" target="_blank" rel="noopener">مشاهده فیش</a>'; }
	  if(i.can_submit_manual_payment){ receiptHtml += '<button type="button" class="sn-btn sn-btn-secondary sn-btn-mini sn-supervisor-payment-open" data-invoice-code="'+esc(i.invoice_code||'')+'" data-amount="'+esc((i.payment_summary||{}).due||i.current_due_amount||'')+'">ثبت واریز و فیش</button>'; }
      if(canSupervisorUploadReceipt(i)){
		receiptHtml += '<button type="button" class="sn-btn sn-btn-primary sn-btn-mini sn-payment-proof-open sn-supervisor-receipt-open" data-invoice-id="'+esc(i.id||'')+'" data-has-receipt="'+(rurl?'1':'0')+'">'+(rurl?'جایگزینی فیش':'بارگذاری فیش')+'</button>';
      }
	  if(receiptHtml){ receiptHtml = '<div class="sn-payment-cell-actions">'+receiptHtml+'</div>'; }
	  else { receiptHtml = '—'; }
      var projectHtml = i.project && i.project.membership_id ? '<button type="button" class="sn-btn sn-btn-primary sn-btn-mini sn-project-chat-btn" data-membership="'+esc(i.project.membership_id)+'" data-item="'+esc((i.project.items&&i.project.items[0]&&i.project.items[0].id)||'')+'">چت با بیاوین</button>' : '—';
      var amounts=i.payment_summary||{};
      var amountCell='<div class="sn-stage-amounts"><span>کل: '+esc(money(amounts.total||0))+'</span><span>پرداخت‌شده: '+esc(money(amounts.paid||0))+'</span><strong>مانده: '+esc(money(amounts.remaining||0))+'</strong><small>مرحله '+esc(amounts.stage_no||1)+' · قابل پرداخت فعلی: '+esc(money(amounts.due||0))+'</small></div>';
      html+='<tr data-invoice-id="'+esc(i.id||'')+'">'+
        '<td><code>'+esc(i.invoice_code || ('#'+(i.id||'')))+'</code></td>'+
        '<td>'+esc(i.seller_name||i.seller_id||'—')+'</td>'+
        '<td>'+esc(i.customer_name||'')+'<br><small>'+esc(i.customer_phone||'')+'</small></td>'+
        '<td>'+amountCell+'</td>'+
        '<td>'+esc(i.pay_method_label||'—')+'<br><small>'+esc(i.payment_source_label||'—')+'</small></td>'+
		'<td><span class="sn-status">'+esc(i.status_label||i.status||'—')+'</span>'+giftTag+'</td>'+
		'<td>'+receiptHtml+'</td>'+
		'<td>'+actionHtml+'</td>'+
		'<td>'+projectHtml+'</td>'+
        '<td>'+esc(date)+'</td>'+
      '</tr>';
    });
    html+='</tbody></table></div>';
    $('#sn-supervisor-invoices-list').html(html);
    if(hasMore){ $('#sn-supervisor-invoices-list').append('<div class="sn-muted" style="margin-top:8px">برای حفظ سرعت، ۵۰ فاکتور آخر نمایش داده شد. برای موارد قدیمی‌تر از جستجو و تاریخ استفاده کن.</div>'); }
  }
  function activeStatus(){ return String($('#sn-supervisor-invoice-status-filter').val() || $('.sn-supervisor-invoice-tabs .sn-subtab.active').data('status') || 'all'); }
  function load(tab, force){
    var $target=$('#sn-supervisor-invoices-list');
    if(!$target.length) return;
    tab = tab || $('.sn-supervisor-invoice-tabs .sn-subtab.active').data('status') || 'all';
    var q = $('#sn-supervisor-invoice-search').val() || '';
    var df = $('#sn-supervisor-invoice-date-from').val() || '';
    var dt = $('#sn-supervisor-invoice-date-to').val() || '';
    var sf = activeStatus();
    if(sf && sf !== 'all') { tab = sf; }
    var key=[tab,q,df,dt,sf].join('|');
    if(!force && invoiceCache[key]){ render(invoiceCache[key].rows, invoiceCache[key].has_more, invoiceCache[key].summary); return; }
    if(invoicePending && invoicePending.readyState!==4){ try{invoicePending.abort();}catch(e){} }
    $target.html('<div class="sn-loading">در حال بارگذاری فاکتورها...</div>');
    invoicePending=$.ajax({url:ajax,type:'POST',dataType:'json',timeout:30000,data:{action:'sn_supervisor_invoices',nonce:nonce,tab:tab,q:q,date_from:df,date_to:dt,status_filter:sf,limit:50}})
      .done(function(res){
        var payload=(res&&res.data&&typeof res.data==='object')?res.data:res;
        if(!res || !res.success){ $target.html('<p class="sn-notice sn-error">'+esc((payload&&payload.message)||'خطا در دریافت فاکتورها')+'</p>'); return; }
        var rows=(payload&&payload.items)||[];
        var summary=(payload&&payload.summary)||{};
        invoiceCache[key]={rows:rows,has_more:!!(payload&&payload.has_more),summary:summary};
        render(rows, !!(payload&&payload.has_more), summary);
      })
      .fail(function(xhr,status){ if(xhr&&xhr.statusText==='abort') return; var msg=status==='timeout'?'زمان پاسخ سرور تمام شد.':'خطای سرور در دریافت فاکتورها: '+esc((xhr&&xhr.status)||'نامشخص'); $target.html('<div class="sn-notice sn-error">'+msg+' <button type="button" class="sn-btn sn-btn-sm sn-supervisor-invoice-retry">تلاش مجدد</button></div>'); });
  }
	$(document).off('click.snSupReceiptOpen','.sn-supervisor-receipt-open').on('click.snSupReceiptOpen','.sn-supervisor-receipt-open',function(){
	  supervisorReceiptInvoiceId=String($(this).data('invoice-id')||''); ensureSupervisorReceiptModal();
	  $('#sn-supervisor-receipt-card-to,#sn-supervisor-receipt-file').val(''); $('#sn-supervisor-receipt-msg').empty();
	  $('#sn-supervisor-receipt-modal .sn-supervisor-upload-receipt').text(String($(this).data('has-receipt'))==='1'?'جایگزینی فیش':'ارسال فیش');
	  $('#sn-supervisor-receipt-modal').fadeIn(120).attr('aria-hidden','false');
	});
	$(document).off('click.snSupReceiptClose','.sn-supervisor-receipt-close').on('click.snSupReceiptClose','.sn-supervisor-receipt-close',function(){ $('#sn-supervisor-receipt-modal').fadeOut(120).attr('aria-hidden','true'); });
  $(document).off('click.snSupReceiptUpload','#sn-supervisor-receipt-modal .sn-supervisor-upload-receipt').on('click.snSupReceiptUpload','#sn-supervisor-receipt-modal .sn-supervisor-upload-receipt',function(e){
    e.preventDefault();
	var $btn=$(this), idleLabel=$btn.text(), $msg=$('#sn-supervisor-receipt-msg');
	var fileInput=$('#sn-supervisor-receipt-file')[0];
	var cardTo=digits($('#sn-supervisor-receipt-card-to').val()).replace(/\D+/g,'');
	var invoiceId=supervisorReceiptInvoiceId;
	if(!invoiceId){ $msg.css('color','#dc2626').text('شناسه فاکتور نامعتبر است.'); return; }
	if(cardTo!=='' && !/^\d{4}$/.test(cardTo)){ $msg.css('color','#dc2626').text('۴ رقم کارت مقصد اختیاری است؛ در صورت ورود باید ۴ رقم باشد.'); return; }
	if(!fileInput || !fileInput.files || !fileInput.files.length){ $msg.css('color','#dc2626').text('اول فایل فیش را انتخاب کنید.'); return; }
    var fd=new FormData();
    fd.append('action','sn_supervisor_upload_receipt');
	fd.append('nonce',nonce);
	fd.append('invoice_id',invoiceId);
	fd.append('card_to',cardTo);
	fd.append('receipt',fileInput.files[0]);
    $btn.prop('disabled',true).text('در حال ارسال...');
    $msg.css('color','#64748b').text('در حال آپلود فیش...');
    $.ajax({url:ajax,type:'POST',data:fd,dataType:'json',processData:false,contentType:false,timeout:30000})
      .done(function(res){
        var payload=(res&&res.data&&typeof res.data==='object')?res.data:res;
        if(!res || !res.success){ $msg.css('color','#dc2626').text((payload&&payload.message)||'آپلود فیش انجام نشد.'); return; }
        $msg.css('color','#16a34a').text((payload&&payload.message)||'فیش ثبت شد و برای بررسی مالی ارسال شد.');
        invoiceCache={};
		setTimeout(function(){ $('#sn-supervisor-receipt-modal').fadeOut(120); load(null,true); }, 650);
      })
      .fail(function(xhr){ $msg.css('color','#dc2626').text('خطای سرور در آپلود فیش: '+esc((xhr&&xhr.status)||'نامشخص')); })
      .always(function(){ $btn.prop('disabled',false).text(idleLabel); });
  });
	$(document).off('click.snSupervisorPaymentOpen','.sn-supervisor-payment-open').on('click.snSupervisorPaymentOpen','.sn-supervisor-payment-open',function(){
	  supervisorPaymentInvoiceCode=String($(this).data('invoice-code')||''); ensureSupervisorPaymentModal();
	  $('#sn-supervisor-payment-card-from,#sn-supervisor-payment-card-to,#sn-supervisor-payment-date,#sn-supervisor-payment-receipt').val(''); $('#sn-supervisor-payment-amount').val(String($(this).data('amount')||'')); $('#sn-supervisor-payment-msg').empty();
	  if(window.snPaymentTimeInit)window.snPaymentTimeInit('#sn-supervisor-payment-hour','#sn-supervisor-payment-minute');
	  $('#sn-supervisor-manual-payment-modal').fadeIn(120).attr('aria-hidden','false');
	});
	$(document).off('click.snSupervisorPaymentClose','.sn-supervisor-payment-close').on('click.snSupervisorPaymentClose','.sn-supervisor-payment-close',function(){ $('#sn-supervisor-manual-payment-modal').fadeOut(120).attr('aria-hidden','true'); });
	$(document).off('click.snSupervisorPaymentSubmit','#sn-supervisor-payment-submit').on('click.snSupervisorPaymentSubmit','#sn-supervisor-payment-submit',function(){
	  var from=digits($('#sn-supervisor-payment-card-from').val()).replace(/\D+/g,''), to=digits($('#sn-supervisor-payment-card-to').val()).replace(/\D+/g,''), date=digits($('#sn-supervisor-payment-date').val()).trim(), time=String($('#sn-supervisor-payment-hour').val()||'')+':'+String($('#sn-supervisor-payment-minute').val()||'');
	  var $btn=$(this), $msg=$('#sn-supervisor-payment-msg');
	  if(!/^\d{4}$/.test(from)||!/^\d{4}\/\d{2}\/\d{2}$/.test(date)||!/^\d{2}:\d{2}$/.test(time)){ $msg.html('<div class="sn-notice sn-error">۴ رقم مبدا، تاریخ و ساعت را کامل وارد کنید.</div>'); return; }
	  if(to!=='' && !/^\d{4}$/.test(to)){ $msg.html('<div class="sn-notice sn-error">۴ رقم کارت مقصد اختیاری است؛ در صورت ورود باید ۴ رقم باشد.</div>'); return; }
	  $btn.prop('disabled',true).text('در حال ثبت...');
	  var form=new FormData();form.append('action','sn_submit_manual_payment');form.append('nonce',nonce);form.append('invoice_code',supervisorPaymentInvoiceCode);form.append('card_from',from);form.append('card_to',to);form.append('paid_at',date+' '+time);form.append('amount',digits($('#sn-supervisor-payment-amount').val()));
	  var receipt=$('#sn-supervisor-payment-receipt')[0].files[0];if(receipt)form.append('receipt',receipt);
	  $.ajax({url:ajax,type:'POST',data:form,processData:false,contentType:false,dataType:'json'}).done(function(res){
		var payload=(res&&res.data&&typeof res.data==='object')?res.data:res;
		if(res&&res.success){ $msg.html('<div class="sn-notice sn-success">'+esc((payload&&payload.message)||res.message||'ثبت شد')+'</div>'); invoiceCache={}; setTimeout(function(){ $('#sn-supervisor-manual-payment-modal').fadeOut(120); load(null,true); },650); }
		else $msg.html('<div class="sn-notice sn-error">'+esc((payload&&payload.message)||(res&&res.message)||'ثبت انجام نشد')+'</div>');
	  }).fail(function(){ $msg.html('<div class="sn-notice sn-error">خطا در ارتباط با سرور</div>'); }).always(function(){ $btn.prop('disabled',false).text('ثبت اطلاعات واریز'); });
	});
	$(document).off('click.snDotGiftValidation','#sn-supervisor-invoices-list .sn-dot-gift-validation').on('click.snDotGiftValidation','#sn-supervisor-invoices-list .sn-dot-gift-validation',function(e){
	  e.preventDefault();
	  var $btn=$(this), invoiceId=String($btn.data('invoice-id')||''), $row=$btn.closest('tr'), $msg=$row.find('.sn-dot-gift-validation-msg'), idleLabel=$btn.text()||'ارسال لینک انتخاب';
	  if(!invoiceId || $btn.data('busy')) return;
	  if(!window.confirm('هزینه اعتبارسنجی از مشتری دریافت نمی‌شود و لینک انتخاب برای او پیامک خواهد شد. ادامه می‌دهید؟')) return;
	  $btn.data('busy',1).prop('disabled',true).text('در حال ارسال...');
	  $msg.removeClass('is-error is-success').text('در حال ساخت دسترسی و ارسال پیامک...');
	  $.ajax({url:ajax,type:'POST',dataType:'json',timeout:30000,data:{action:'sn_dot_gift_validation_access',nonce:nonce,invoice_id:invoiceId}})
	    .done(function(res){
	      var payload=(res&&res.data&&typeof res.data==='object')?res.data:res;
	      if(!res || !res.success){
	        $btn.data('busy',0).prop('disabled',false).text(idleLabel);
	        $msg.addClass('is-error').text((payload&&payload.message)||'ارسال لینک انتخاب انجام نشد.');
	        return;
	      }
	      $msg.addClass('is-success').text((payload&&payload.message)||'اعتبارسنجی هدیه فعال و لینک انتخاب ارسال شد.');
	      if(!$row.find('td').eq(5).find('.sn-dot-validation-gift-tag').length){ $row.find('td').eq(5).append('<span class="sn-dot-validation-gift-tag">اعتبارسنجی هدیه</span>'); }
	      if(payload&&((payload.sms_sent===true)||(payload.sms_previewed===true))){
	        $btn.replaceWith('<span class="sn-dot-validation-gift-tag">اعتبارسنجی هدیه</span>');
	      }else{
	        $btn.data('busy',0).prop('disabled',false).text('تلاش مجدد ارسال لینک');
	      }
	      invoiceCache={};
	      window.setTimeout(function(){ load(null,true); },1000);
	    })
	    .fail(function(xhr){
	      var res=xhr&&xhr.responseJSON?xhr.responseJSON:null;
	      var payload=res&&res.data?res.data:res;
	      $btn.data('busy',0).prop('disabled',false).text(idleLabel);
	      $msg.addClass('is-error').text((payload&&payload.message)||'ارتباط با سرور برقرار نشد.');
	    });
	});
  $(document).off('click.snSupInvRetry','.sn-supervisor-invoice-retry').on('click.snSupInvRetry','.sn-supervisor-invoice-retry',function(e){ e.preventDefault(); load(null,true); });
  window.snLoadSupervisorInvoicesFinal = load;
  $(document).off('click.snSupInv128','.sn-supervisor-invoice-tabs .sn-subtab').on('click.snSupInv128','.sn-supervisor-invoice-tabs .sn-subtab',function(){ $('.sn-supervisor-invoice-tabs .sn-subtab').removeClass('active'); $(this).addClass('active'); var st=String($(this).data('status')||'all'); $('#sn-supervisor-invoice-status-filter').val(st); load(st, false); });
  $(document).off('click.snSupInv128Filter','#sn-supervisor-invoice-apply-filter').on('click.snSupInv128Filter','#sn-supervisor-invoice-apply-filter',function(e){ e.preventDefault(); var sf=String($('#sn-supervisor-invoice-status-filter').val()||'all'); $('.sn-supervisor-invoice-tabs .sn-subtab').removeClass('active').filter('[data-status="'+sf+'"]').addClass('active'); if(!$('.sn-supervisor-invoice-tabs .sn-subtab.active').length){ $('.sn-supervisor-invoice-tabs .sn-subtab[data-status="all"]').addClass('active'); } load(sf, true); });
  $(document).off('keydown.snSupInv128Search input.snSupInv128Search','#sn-supervisor-invoice-search').on('keydown.snSupInv128Search input.snSupInv128Search','#sn-supervisor-invoice-search',function(e){ if(e.type==='keydown' && e.key!=='Enter') return; if(e.type==='keydown') e.preventDefault(); clearTimeout(window.snSupervisorInvoiceSearchTimer); window.snSupervisorInvoiceSearchTimer=setTimeout(function(){ load(null, true); }, e.type==='keydown'?10:450); });
  $(document).off('change.snSupInv128Dates','#sn-supervisor-invoice-date-from,#sn-supervisor-invoice-date-to,#sn-supervisor-invoice-status-filter').on('change.snSupInv128Dates','#sn-supervisor-invoice-date-from,#sn-supervisor-invoice-date-to,#sn-supervisor-invoice-status-filter',function(){ clearTimeout(window.snSupervisorInvoiceFilterTimer); window.snSupervisorInvoiceFilterTimer=setTimeout(function(){ load(null, true); }, 180); });
  $(document).off('click.snSupInv128Tab','#sn-supervisor-panel .sn-tab[data-tab="invoices"]').on('click.snSupInv128Tab','#sn-supervisor-panel .sn-tab[data-tab="invoices"]',function(){ setTimeout(function(){ load('all', false); }, 40); });
})(jQuery);

/* SN 1.0.67 finance review hard fix: single handler, no repeated alerts, inline status update */
(function($){
  'use strict';
  if (window.snFinanceReviewHardFixReady) { return; }
  window.snFinanceReviewHardFixReady = true;
  var cfg = window.snAjax || window.snData || {};
  var ajax = cfg.ajaxurl || window.ajaxurl || '/wp-admin/admin-ajax.php';
  var nonce = cfg.nonce || cfg.admin_nonce || '';
  var currentRejectId = 0;
  var busy = {};
  function esc(v){ return String(v==null?'':v).replace(/[&<>"']/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#039;'}[c];}); }
  function faStatusLabel(res, fallback){
    var label = (res && (res.status_label || res.message)) || fallback || 'انجام شد';
    var tag = (res && res.finance_review_tag) ? ('<span class="sn-finance-review-tag">'+esc(res.finance_review_tag)+'</span>') : '';
    var reviewer = (res && res.reviewed_by_name) ? ('<small class="sn-finance-review-by">مالی: '+esc(res.reviewed_by_name)+(res.reviewed_at?' - '+esc(res.reviewed_at):'')+'</small>') : '';
    return '<span class="sn-status sn-status-approved">'+esc(label)+'</span>'+tag+reviewer;
  }
  function rowMsg($row, type, msg){
    var cls = type === 'success' ? 'sn-success' : 'sn-error';
    $row.find('td').last().html('<div class="sn-inline-result '+cls+'">'+esc(msg || '')+'</div>');
  }
  function reloadFinancialPanel(){
    if (typeof window.loadFinancial === 'function') { try { window.loadFinancial(); return; } catch(e){} }
    var $active = $('.sn-financial-tabs .sn-subtab.active');
    if ($active.length) { $active.trigger('click'); return; }
    var $list = $('#sn-financial-list');
    if ($list.length) { $list.prepend('<div class="sn-notice sn-success">تغییر ثبت شد. برای بروزرسانی کامل تب را دوباره باز کنید.</div>'); }
  }
  function ensureRejectModal(){
    if ($('#sn-fin-reject-modal').length) { return; }
    $('body').append(
      '<div id="sn-fin-reject-modal" class="sn-modal sn-lite-modal sn-finance-review-modal" aria-hidden="true" style="display:none">'+
        '<div class="sn-modal-backdrop sn-fin-reject-close"></div>'+
        '<div class="sn-modal-card sn-fin-reject-card">'+
          '<div class="sn-modal-head"><h3>رد پرداخت توسط مالی</h3><button type="button" class="sn-modal-x sn-fin-reject-close">×</button></div>'+
          '<div class="sn-modal-body">'+
            '<p class="sn-muted">دلیل رد برای فروشنده نمایش داده می‌شود و فاکتور به فروشنده برمی‌گردد.</p>'+
            '<label class="sn-finance-field">دلیل رد<textarea id="sn-fin-reject-reason" rows="5" placeholder="مثلاً مغایرت مبلغ، شماره کارت، زمان واریز یا فیش نامعتبر"></textarea></label>'+
            '<div id="sn-fin-reject-msg"></div>'+
            '<div class="sn-modal-actions"><button type="button" class="sn-btn sn-btn-secondary sn-fin-reject-close">انصراف</button><button type="button" class="sn-btn sn-btn-danger" id="sn-fin-reject-submit">ثبت رد پرداخت</button></div>'+
          '</div>'+
        '</div>'+
      '</div>'
    );
  }
  function intercept(selector, handler){
    document.addEventListener('click', function(ev){
      var target = ev.target && ev.target.closest ? ev.target.closest(selector) : null;
      if (!target) { return; }
      ev.preventDefault();
      ev.stopPropagation();
      if (ev.stopImmediatePropagation) { ev.stopImmediatePropagation(); }
      handler.call(target, ev);
    }, true);
  }
  intercept('.sn-fin-approve', function(){
    var $btn = $(this), id = String($btn.data('id') || '');
    if (!id || busy['approve_'+id]) { return; }
    busy['approve_'+id] = true;
    var $row = $btn.closest('tr');
    $btn.prop('disabled', true).text('در حال تایید...');
    $.post(ajax, {action:'sn_financial_approve_payment', nonce:nonce, invoice_id:id}, function(res){
      if (res && res.success) {
        $row.find('td').eq(6).html(faStatusLabel(res, 'تایید شده'));
        rowMsg($row, 'success', res.message || 'پرداخت توسط مالی تایید شد');
        $row.addClass('sn-finance-row-reviewed');
        setTimeout(reloadFinancialPanel, 650);
      } else {
        $btn.prop('disabled', false).text('تایید');
        rowMsg($row, 'error', (res && res.message) || 'خطا در تایید پرداخت');
      }
    }).fail(function(xhr){
      $btn.prop('disabled', false).text('تایید');
      rowMsg($row, 'error', 'خطای سرور: ' + (xhr && xhr.status ? xhr.status : ''));
    }).always(function(){ delete busy['approve_'+id]; });
  });
  intercept('.sn-fin-reject', function(){
    var id = String($(this).data('id') || '');
    if (!id) { return; }
    ensureRejectModal();
    currentRejectId = id;
    $('#sn-fin-reject-reason').val('');
    $('#sn-fin-reject-msg').empty();
    $('#sn-fin-reject-modal').fadeIn(120).attr('aria-hidden','false');
    setTimeout(function(){ $('#sn-fin-reject-reason').trigger('focus'); }, 150);
  });
  $(document).off('click.snFinanceHardFixClose').on('click.snFinanceHardFixClose','.sn-fin-reject-close',function(){
    $('#sn-fin-reject-modal').fadeOut(120).attr('aria-hidden','true');
  });
  $(document).off('click.snFinanceHardFixReject').on('click.snFinanceHardFixReject','#sn-fin-reject-submit',function(e){
    e.preventDefault();
    var reason = String($('#sn-fin-reject-reason').val() || '').trim();
    if (!reason) { $('#sn-fin-reject-msg').html('<div class="sn-notice sn-error">دلیل رد را وارد کنید.</div>'); return; }
    if (!currentRejectId || busy['reject_'+currentRejectId]) { return; }
    busy['reject_'+currentRejectId] = true;
    var id = currentRejectId, $btn = $(this);
    $btn.prop('disabled', true).text('در حال ثبت...');
    $.post(ajax, {action:'sn_financial_reject_payment', nonce:nonce, invoice_id:id, reason:reason}, function(res){
      if (res && res.success) {
        $('#sn-fin-reject-msg').html('<div class="sn-notice sn-success">پرداخت رد شد و به فروشنده برگشت.</div>');
        setTimeout(function(){ $('#sn-fin-reject-modal').fadeOut(120).attr('aria-hidden','true'); reloadFinancialPanel(); }, 650);
      } else {
        $('#sn-fin-reject-msg').html('<div class="sn-notice sn-error">'+esc((res && res.message) || 'خطا در ثبت رد پرداخت')+'</div>');
      }
    }).fail(function(xhr){
      $('#sn-fin-reject-msg').html('<div class="sn-notice sn-error">خطای سرور: '+esc(xhr && xhr.status ? xhr.status : '')+'</div>');
    }).always(function(){
      delete busy['reject_'+id];
      $btn.prop('disabled', false).text('ثبت رد پرداخت');
    });
  });
  // حذف handlerهای تکراری bubble-phase بعد از نصب capture handler
  $(function(){
    $(document).off('click', '.sn-fin-approve');
    $(document).off('click', '.sn-fin-reject');
  });
})(jQuery);


/* SN 1.0.67: simple no-refresh distribution assignment */
(function($){
  'use strict';
  var cfg = window.snAjax || window.snData || {};
  var ajax = cfg.ajaxurl || '';
  if (!ajax) return;

  function esc(v){ return String(v==null?'':v).replace(/[&<>"']/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#039;'}[c] || c;}); }
  function notice($form, type, msg){
    var cls = type === 'success' ? 'sn-success' : (type === 'warning' ? 'sn-warning' : 'sn-error');
    var $box = $form.find('.sn-distribution-assign-notice').first();
    if (!$box.length) { $box = $('<div class="sn-distribution-assign-notice"></div>').prependTo($form); }
    $box.html('<div class="sn-notice '+cls+'">'+esc(msg || '')+'</div>');
  }
  function currentMode($form){
    return String($form.find('input[name="sn_assign_ui_mode"]:checked').val() || 'count');
  }

  function applyRecipientPositionFilter($form){
    var selected = String($form.find('.sn-simple-target-position').val() || '');
    var visibleCount = 0;
    $form.find('.sn-simple-recipient-chip').each(function(){
      var $chip = $(this);
      var pos = String($chip.data('position') || '');
      var match = !selected || pos === selected;
      $chip.toggle(match);
      if (!match) { $chip.find('input[name="recipient_user_ids[]"]').prop('checked', false); }
      else { visibleCount++; }
    });
    $form.find('.sn-simple-recipient-select-all').prop('checked', false);
    var $hint = $form.find('.sn-simple-recipient-position-hint');
    if (!$hint.length) { $hint = $('<div class="sn-simple-recipient-position-hint sn-muted"></div>').insertAfter($form.find('.sn-simple-target-position-field')); }
    if (selected) { $hint.text(visibleCount ? (visibleCount + ' گیرنده در این سطح پیدا شد.') : 'در این سطح گیرنده‌ای در زیرمجموعه شما پیدا نشد.'); }
    else { $hint.text('همه سطح‌های مجاز نمایش داده می‌شود.'); }
    var $empty = $form.find('.sn-simple-recipient-empty');
    if ($empty.length) { $empty.prop('hidden', visibleCount > 0); }
    $form.find('.sn-simple-recipient-list').prop('hidden', visibleCount <= 0).css('display','');
  }

  function applyCaseFilter($form){
    var selected = String($form.find('.sn-distribution-case-filter').val() || '');
    var $rows = $form.find('.sn-distribution-owned-table tbody tr[data-case-filter]');
    if (!selected) {
      $rows.show();
      $form.closest('.sn-distribution-role').find('.sn-distribution-case-card').removeClass('is-active');
      return;
    }
    $rows.each(function(){
      var $row = $(this);
      var match = String($row.data('case-filter') || '') === selected;
      $row.toggle(match);
      if (!match) { $row.find('input[name="distribution_item_ids[]"]').prop('checked', false); }
    });
    $form.closest('.sn-distribution-role').find('.sn-distribution-case-card').removeClass('is-active').filter(function(){ return String($(this).data('case-filter') || '') === selected; }).addClass('is-active');
  }

  function syncAssignMode($form){
    var mode = currentMode($form);
    var $bulk = $form.find('.sn-simple-bulk-mode');
    var $count = $form.find('.sn-simple-count-field');
    var $manual = $form.find('.sn-simple-manual-note');
    var $itemChecks = $form.find('input[name="distribution_item_ids[]"]');
    if (mode === 'manual') {
      $bulk.val('selected');
      $count.addClass('is-disabled').find('input').prop('disabled', true);
      $manual.show();
      $itemChecks.prop('disabled', false);
    } else {
      $bulk.val('round_robin_active_sellers');
      $count.removeClass('is-disabled').find('input').prop('disabled', false);
      $manual.hide();
      $itemChecks.prop('disabled', true).prop('checked', false);
    }
    applyRecipientPositionFilter($form);
    applyCaseFilter($form);
  }

  $(document).off('change.snSimpleDistributionMode', '.sn-simple-assign-form input[name="sn_assign_ui_mode"]');
  $(document).on('change.snSimpleDistributionMode', '.sn-simple-assign-form input[name="sn_assign_ui_mode"]', function(){
    syncAssignMode($(this).closest('.sn-simple-assign-form'));
  });

  $(document).off('change.snSimpleDistributionAll', '.sn-simple-recipient-select-all');
  $(document).on('change.snSimpleDistributionAll', '.sn-simple-recipient-select-all', function(){
    var checked = $(this).is(':checked');
    $(this).closest('.sn-simple-recipients').find('.sn-simple-recipient-chip:visible input[name="recipient_user_ids[]"]').prop('checked', checked);
  });

  $(function(){
    $('.sn-simple-assign-form').each(function(){ syncAssignMode($(this)); });
  });

  $(document).off('change.snSimpleDistributionTargetPosition', '.sn-simple-assign-form .sn-simple-target-position');
  $(document).on('change.snSimpleDistributionTargetPosition', '.sn-simple-assign-form .sn-simple-target-position', function(){
    applyRecipientPositionFilter($(this).closest('.sn-simple-assign-form'));
  });

  $(document).off('change.snSimpleDistributionCase', '.sn-simple-assign-form .sn-distribution-case-filter');
  $(document).on('change.snSimpleDistributionCase', '.sn-simple-assign-form .sn-distribution-case-filter', function(){
    applyCaseFilter($(this).closest('.sn-simple-assign-form'));
  });

  $(document).off('click.snSimpleDistributionCaseCard', '.sn-distribution-role .sn-distribution-case-card');
  $(document).on('click.snSimpleDistributionCaseCard', '.sn-distribution-role .sn-distribution-case-card', function(){
    var key = String($(this).data('case-filter') || '');
    var $role = $(this).closest('.sn-distribution-role');
    var $form = $role.find('.sn-simple-assign-form').first();
    $form.find('.sn-distribution-case-filter').val(key).trigger('change');
    try { $form[0].scrollIntoView({behavior:'smooth', block:'start'}); } catch(e) {}
  });

  $(document).off('submit.snSimpleDistribution', '.sn-simple-assign-form');
  $(document).on('submit.snSimpleDistribution', '.sn-simple-assign-form', function(e){
    e.preventDefault();
    var $form = $(this);
    syncAssignMode($form);
    var mode = currentMode($form);
    var recipients = $form.find('input[name="recipient_user_ids[]"]:checked').length;
    if (!recipients) {
      notice($form, 'error', 'حداقل یک گیرنده را انتخاب کنید.');
      return;
    }
    if (mode === 'count') {
      var count = Number($form.find('input[name="per_seller_count"]').val() || 0);
      if (!count || count < 1) {
        notice($form, 'error', 'تعداد شماره به هر گیرنده را وارد کنید.');
        return;
      }
    } else {
      var selectedItems = $form.find('input[name="distribution_item_ids[]"]:checked').length;
      if (!selectedItems) {
        notice($form, 'error', 'حداقل یک شماره را از جدول پایین انتخاب کنید.');
        return;
      }
    }

    var fd = new FormData(this);
    fd.set('action', 'sn_distribution_transfer_items');
    fd.delete('sn_assign_ui_mode');
    if (mode === 'count') {
      fd.delete('distribution_item_ids[]');
      fd.set('distribution_bulk_mode', 'round_robin_active_sellers');
      fd.set('distribution_recipient_scope', 'selected');
    } else {
      fd.set('distribution_bulk_mode', 'selected');
      fd.set('distribution_recipient_scope', 'selected');
      fd.delete('per_seller_count');
      fd.delete('allocation_count');
    }

    var $btn = $form.find('.sn-simple-assign-submit').first();
    var oldText = $btn.text();
    $btn.prop('disabled', true).text('در حال تخصیص...');
    notice($form, 'warning', 'در حال انجام تخصیص...');

    $.ajax({
      url: ajax,
      type: 'POST',
      data: fd,
      processData: false,
      contentType: false,
      dataType: 'json',
      timeout: 20000
    }).done(function(res){
      if (res && res.success) {
        notice($form, 'success', res.message || 'تخصیص انجام شد.');
        var assignedIds = [];
        var report = (res.report || (res.data && res.data.report) || {});
        if (report.recipient_map) {
          Object.keys(report.recipient_map).forEach(function(itemId){ assignedIds.push(String(itemId)); });
        }
        if (assignedIds.length) {
          assignedIds.forEach(function(id){
            $form.find('input[name="distribution_item_ids[]"]').filter(function(){ return String($(this).val()) === String(id); }).closest('tr').fadeOut(250, function(){ $(this).remove(); });
          });
        }
        $form.find('input[name="distribution_item_ids[]"]').prop('checked', false);
      } else {
        notice($form, 'error', (res && res.message) || 'تخصیص انجام نشد.');
      }
    }).fail(function(xhr){
      var msg = 'خطای سرور هنگام تخصیص.';
      if (xhr && xhr.status) msg += ' کد: ' + xhr.status;
      if (xhr && xhr.responseJSON && xhr.responseJSON.message) msg = xhr.responseJSON.message;
      notice($form, 'error', msg);
    }).always(function(){
      $btn.prop('disabled', false).text(oldText || 'اعمال تخصیص');
    });
  });
})(jQuery);


/* SN 1.0.67 quick fix: allocation result inline + AJAX section refresh */
(function($){
  'use strict';
  var cfg = window.snAjax || window.snData || {};
  var ajax = cfg.ajaxurl || '';
  if(!ajax) return;
  function esc(v){ return String(v==null?'':v).replace(/[&<>"']/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#039;'}[c]||c;}); }
  function notice($form,type,msg){
    var cls = type==='success'?'sn-success':(type==='warning'?'sn-warning':'sn-error');
    var $box=$form.find('.sn-distribution-assign-notice').first();
    if(!$box.length){ $box=$('<div class="sn-distribution-assign-notice sn-distribution-inline-result"></div>').appendTo($form.find('.sn-simple-assign-actions').first()); }
    $box.html('<div class="sn-notice '+cls+'">'+esc(msg||'')+'</div>');
  }
  function currentMode($form){ return String($form.find('input[name="sn_assign_ui_mode"]:checked').val()||'count'); }
  function supervisorCountKey(){ return 'sn_supervisor_last_per_seller_count_' + ((window.snAjax && window.snAjax.user_id) || (window.snData && window.snData.user_id) || 'current'); }
  function persistSupervisorCount($form){ try{ var v=String($form.find('input[name="per_seller_count"]').val()||'').trim(); if(v) localStorage.setItem(supervisorCountKey(), v); }catch(e){} }
  function restoreSupervisorCount($form){ try{ var v=localStorage.getItem(supervisorCountKey()); var $i=$form.find('input[name="per_seller_count"]'); if(v && !$i.val()) $i.val(v); }catch(e){} }
  function sync($form){
    var mode=currentMode($form), $bulk=$form.find('.sn-simple-bulk-mode'), $count=$form.find('.sn-simple-count-field'), $manual=$form.find('.sn-simple-manual-note'), $items=$form.find('input[name="distribution_item_ids[]"]');
    if(mode==='manual'){ $bulk.val('selected'); $count.addClass('is-disabled').find('input').prop('disabled',true); $manual.show(); $items.prop('disabled',false); }
    else { $bulk.val('round_robin_active_sellers'); $count.removeClass('is-disabled').find('input').prop('disabled',false); $manual.hide(); $items.prop('disabled',true).prop('checked',false); }
    var selected=String($form.find('.sn-simple-target-position').val()||''), visible=0;
    $form.find('.sn-simple-recipient-chip').each(function(){ var $chip=$(this), pos=String($chip.data('position')||''), ok=!selected||pos===selected; $chip.toggle(ok); if(!ok){$chip.find('input[name="recipient_user_ids[]"]').prop('checked',false);} else {visible++;} });
    $form.find('.sn-simple-recipient-select-all').prop('checked',false);
    var $hint=$form.find('.sn-simple-recipient-position-hint'); if(!$hint.length){$hint=$('<div class="sn-simple-recipient-position-hint sn-muted"></div>').insertAfter($form.find('.sn-simple-target-position-field'));}
    $hint.text(selected?(visible?visible+' گیرنده در این سطح پیدا شد.':'در این سطح گیرنده‌ای در زیرمجموعه شما پیدا نشد.'):'همه سطح‌های مجاز نمایش داده می‌شود.');
    var $empty=$form.find('.sn-simple-recipient-empty'); if($empty.length){$empty.prop('hidden', visible>0);} $form.find('.sn-simple-recipient-list').prop('hidden', visible<=0).css('display','');
    var caseKey=String($form.find('.sn-distribution-case-filter').val()||'');
    var $rows=$form.find('.sn-distribution-owned-table tbody tr[data-case-filter]');
    if(caseKey){ $rows.each(function(){ var ok=String($(this).data('case-filter')||'')===caseKey; $(this).toggle(ok); if(!ok){$(this).find('input[name="distribution_item_ids[]"]').prop('checked',false);} }); }
    else { $rows.show(); }
  }
  function refreshSection($oldForm,msg,type){
    var id=$oldForm.attr('id')||'', $role=$oldForm.closest('.sn-distribution-role');
    if(!id || !$role.length){ notice($oldForm,type,msg); return; }
    $.ajax({url:window.location.href,type:'GET',cache:false,timeout:15000}).done(function(html){
      var $html=$($.parseHTML(html,document,true));
      var $newForm=$html.find('#'+id);
      if($newForm.length){
        $role.replaceWith($newForm.closest('.sn-distribution-role'));
        var $fresh=$('#'+id); restoreSupervisorCount($fresh); sync($fresh); notice($fresh,type,msg);
      } else { notice($oldForm,type,msg); }
    }).fail(function(){ notice($oldForm,type,msg); });
  }
  $(document).off('submit.snSimpleDistribution', '.sn-simple-assign-form');
  $(document).off('submit.snQuickAllocationFix', '.sn-simple-assign-form');
  $(document).on('submit.snQuickAllocationFix', '.sn-simple-assign-form', function(e){
    e.preventDefault();
    var $form=$(this); sync($form);
    var mode=currentMode($form), recipients=$form.find('input[name="recipient_user_ids[]"]:checked').length;
    if(!recipients){ notice($form,'error','حداقل یک گیرنده را انتخاب کنید.'); return; }
    if(mode==='count'){
      var count=Number($form.find('input[name="per_seller_count"]').val()||0);
      if(!count||count<1){ notice($form,'error','تعداد شماره به هر گیرنده را وارد کنید.'); return; }
    } else if(!$form.find('input[name="distribution_item_ids[]"]:checked').length){ notice($form,'error','حداقل یک شماره را از جدول پایین انتخاب کنید.'); return; }
    persistSupervisorCount($form);
    var fd=new FormData(this); fd.set('action','sn_distribution_transfer_items'); fd.delete('sn_assign_ui_mode'); fd.set('distribution_recipient_scope','selected');
    if(mode==='count'){ fd.delete('distribution_item_ids[]'); fd.set('distribution_bulk_mode','round_robin_active_sellers'); }
    else { fd.set('distribution_bulk_mode','selected'); fd.delete('per_seller_count'); fd.delete('allocation_count'); }
    var $btn=$form.find('.sn-simple-assign-submit').first(), old=$btn.text();
    $btn.prop('disabled',true).text('در حال تخصیص...'); notice($form,'warning','در حال انجام تخصیص...');
    $.ajax({url:ajax,type:'POST',data:fd,processData:false,contentType:false,dataType:'json',timeout:25000}).done(function(res){
      if(res&&res.success){
        var report=(res.report||(res.data&&res.data.report)||{}), msg=res.message||'تخصیص انجام شد.';
        $form.find('input[name="recipient_user_ids[]"], .sn-simple-recipient-select-all, input[name="distribution_item_ids[]"]').prop('checked',false);
        restoreSupervisorCount($form);
        notice($form,'success',msg); if(window.snLoadSupervisorDistributionItems){ window.snLoadSupervisorDistributionItems(1,true); }
      } else { notice($form,'error',(res&&res.message)||'تخصیص انجام نشد.'); }
    }).fail(function(xhr){ var msg='خطای سرور هنگام تخصیص.'; if(xhr&&xhr.status) msg+=' کد: '+xhr.status; if(xhr&&xhr.responseJSON&&xhr.responseJSON.message) msg=xhr.responseJSON.message; notice($form,'error',msg); }).always(function(){ $btn.prop('disabled',false).text(old||'اعمال تخصیص'); });
  });
  $(document).off('change.snQuickAllocationFixMode', '.sn-simple-assign-form input[name="sn_assign_ui_mode"], .sn-simple-assign-form .sn-simple-target-position, .sn-simple-assign-form .sn-distribution-case-filter');
  $(document).on('change.snQuickAllocationFixMode', '.sn-simple-assign-form input[name="sn_assign_ui_mode"], .sn-simple-assign-form .sn-simple-target-position, .sn-simple-assign-form .sn-distribution-case-filter', function(){ sync($(this).closest('.sn-simple-assign-form')); });
  $(document).off('change.snQuickAllocationFixAll', '.sn-simple-recipient-select-all');
  $(document).on('change.snQuickAllocationFixAll', '.sn-simple-recipient-select-all', function(){ $(this).closest('.sn-simple-recipients').find('.sn-simple-recipient-chip:visible input[name="recipient_user_ids[]"]').prop('checked',$(this).is(':checked')); });
  $(document).off('click.snModernAssignCancel', '.sn-modern-assign-cancel');
  $(document).on('click.snModernAssignCancel', '.sn-modern-assign-cancel', function(){
    var $form=$(this).closest('.sn-simple-assign-form');
    $form.find('input[name="recipient_user_ids[]"], input[name="distribution_item_ids[]"], .sn-simple-recipient-select-all').prop('checked',false);
    $form.find('.sn-distribution-assign-notice').empty();
    sync($form);
  });
  $(function(){ $('.sn-simple-assign-form').each(function(){ restoreSupervisorCount($(this)); sync($(this)); }); });
})(jQuery);



/* SN 1.0.122: progressive supervisor distribution item loader */
(function($){
  'use strict';
  if (window.snSupervisorDistributionItemsReady) { return; }
  window.snSupervisorDistributionItemsReady = true;
  var cfg = window.snAjax || window.snData || {};
  var ajax = cfg.ajaxurl || window.ajaxurl || '/wp-admin/admin-ajax.php';
  var nonce = cfg.nonce || '';
  function esc(v){ return String(v==null?'':v).replace(/[&<>"']/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#039;'}[c]||c;}); }
  function currentAssignMode(){ return String($('#sn-tab-assign .sn-simple-assign-form input[name="sn_assign_ui_mode"]:checked').val() || 'count'); }
  function toggleManualChecks(){
    var manual = currentAssignMode() === 'manual';
    $('#sn-supervisor-distribution-items input[name="distribution_item_ids[]"]').prop('disabled', !manual);
  }
  function injectRows($box, html, page, hasMore){
    if (page > 1 && $box.find('table tbody').length) {
      var $tmp = $('<div/>').html(html);
      $box.find('table tbody').append($tmp.find('tr'));
    } else {
      $box.html(html);
    }
    $box.find('.sn-progressive-items-actions').remove();
    var next = page + 1;
    var $actions = $('<div class="sn-actions sn-progressive-items-actions" style="margin-top:10px"></div>');
    if (hasMore) $actions.append('<button type="button" class="sn-btn sn-btn-secondary sn-load-supervisor-items" data-page="'+next+'">بارگذاری ۵۰ شماره بعدی</button>');
    $actions.append('<button type="button" class="sn-btn sn-btn-ghost sn-reload-supervisor-items">بروزرسانی جدول</button>');
    $box.append($actions).data('page', page).data('loaded', 1).data('loading', 0);
    toggleManualChecks();
  }
  function loadItems(page, force){
    var $box = $('#sn-supervisor-distribution-items');
    if (!$box.length) return;
    page = Number(page || 1);
    if (!force && page === 1 && $box.data('loaded')) { toggleManualChecks(); return; }
    if ($box.data('loading')) return;
    var caseFilter = $('#sn-tab-assign .sn-simple-assign-form .sn-distribution-case-filter').val() || '';
    $box.data('loading', 1);
    if (page === 1) $box.html('<div class="sn-card sn-loading">در حال بارگذاری ۵۰ شماره اول...</div>');
    else $box.find('.sn-progressive-items-actions').html('<span class="sn-loading">در حال بارگذاری...</span>');
    $.ajax({url:ajax,type:'POST',dataType:'json',timeout:20000,data:{action:'sn_supervisor_distribution_items',nonce:nonce,page:page,limit:50,case_filter:caseFilter}})
      .done(function(res){
        if(!res || !res.success){ var msg=(res&&res.data&&res.data.message)||(res&&res.message)||'خطا در بارگذاری شماره‌ها'; $box.html('<div class="sn-notice sn-error">'+esc(msg)+'</div>').data('loading',0); return; }
        var data=res.data||{};
        injectRows($box, data.html||'', Number(data.page||page), !!data.has_more);
      })
      .fail(function(xhr){ if(xhr&&xhr.statusText==='abort') return; $box.html('<div class="sn-notice sn-error">خطای سرور در بارگذاری شماره‌ها: '+esc(xhr&&xhr.status?xhr.status:'')+'</div>').data('loading',0); });
  }
  window.snLoadSupervisorDistributionItems = loadItems;
  $(document).off('click.snSupItems','.sn-load-supervisor-items').on('click.snSupItems','.sn-load-supervisor-items',function(){ loadItems($(this).data('page') || 1, false); });
  $(document).off('click.snSupItemsReload','.sn-reload-supervisor-items').on('click.snSupItemsReload','.sn-reload-supervisor-items',function(){ $('#sn-supervisor-distribution-items').data('loaded',0); loadItems(1, true); });
  $(document).off('change.snSupItemsMode','#sn-tab-assign .sn-simple-assign-form input[name="sn_assign_ui_mode"]').on('change.snSupItemsMode','#sn-tab-assign .sn-simple-assign-form input[name="sn_assign_ui_mode"]',function(){ if(currentAssignMode()==='manual') loadItems(1,false); toggleManualChecks(); });
  $(document).off('change.snSupItemsCase','#sn-tab-assign .sn-simple-assign-form .sn-distribution-case-filter').on('change.snSupItemsCase','#sn-tab-assign .sn-simple-assign-form .sn-distribution-case-filter',function(){ var $box=$('#sn-supervisor-distribution-items'); $box.data('loaded',0).data('page',0); if(currentAssignMode()==='manual') loadItems(1,true); });
})(jQuery);

/* SN 1.0.118 supervisor lazy tab loader: avoids server-side rendering of heavy hidden tabs */
(function($){
  'use strict';
  if (window.snSupervisorLazyTabLoaderReady) { return; }
  window.snSupervisorLazyTabLoaderReady = true;
  var cfg = window.snAjax || window.snData || {};
  var ajax = cfg.ajaxurl || window.ajaxurl || '/wp-admin/admin-ajax.php';
  var nonce = cfg.nonce || '';
  function esc(v){ return String(v == null ? '' : v).replace(/[&<>"']/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#039;'}[c];}); }
  function relayoutAssign(){
    var $role = $('#sn-tab-assign .sn-distribution-role-embedded');
    if (!$role.length) { return; }
    var $kpis = $role.find('.sn-distribution-kpis').first();
    var $form = $role.find('.sn-distribution-transfer-form').first();
    if ($kpis.length && $form.length) { $form.insertAfter($kpis); }
    var $case = $role.find('.sn-distribution-case-summary').first();
    if ($case.length && $form.length) { $case.insertAfter($form); }
  }
  function loadLazyTab(target, force){
    var $host = $('#sn-tab-' + target + '.sn-lazy-tab');
    if (!$host.length || ($host.data('loaded') && !force) || $host.data('loading')) { return; }
    $host.data('loading', 1).html('<div class="sn-card sn-loading">در حال بارگذاری...</div>');
    $.post(ajax, {action:'sn_supervisor_lazy_tab', nonce:nonce, tab:target}, function(res){
      if (!res || !res.success) {
        var msg = (res && res.data && res.data.message) || (res && res.message) || 'خطا در بارگذاری تب';
        $host.html('<div class="sn-notice sn-error">' + esc(msg) + '</div>').data('loading', 0);
        return;
      }
      var html = (res.data && res.data.html) || res.html || '';
      $host.html(html).data('loaded', 1).data('loading', 0);
      if (target === 'assign') { setTimeout(function(){ relayoutAssign(); if(window.snLoadSupervisorDistributionItems){ window.snLoadSupervisorDistributionItems(1, false); } }, 80); }
      if (target === 'invoices' && window.snLoadSupervisorInvoicesFinal) { setTimeout(function(){ window.snLoadSupervisorInvoicesFinal(); }, 60); }
    }).fail(function(xhr){
      $host.html('<div class="sn-notice sn-error">خطای سرور در بارگذاری تب: ' + esc(xhr.status) + '</div>').data('loading', 0);
    });
  }
  $(document).on('click.snLazySupervisorTabs', '#sn-supervisor-panel .sn-tab', function(){
    var target = String($(this).data('tab') || '');
    if (target) { setTimeout(function(){ loadLazyTab(target, target === 'needs-action' || target === 'my-conversions'); }, 10); }
  });
})(jQuery);


/* SN 1.0.128 senior supervisor tab engine: public.js is not loaded on supervisor assets */
(function($){
  'use strict';
  if (window.snSeniorSupervisorTabsReady) { return; }
  window.snSeniorSupervisorTabsReady = true;
  function clean(v){ return String(v||'').replace(/^#/,'').replace(/[^A-Za-z0-9_-]/g,''); }
  function targetOf($btn){ return clean($btn.attr('data-sn-tab-target') || $btn.attr('aria-controls') || ($btn.attr('href')||'').split('#').pop()); }
  function panelOf($scope,target){ return $scope.find('.sn-tab-panel').filter(function(){ return clean($(this).attr('data-sn-tab-panel') || this.id) === target; }).first(); }
  function activate($scope,target){
    target=clean(target); if(!$scope.length || !target) return false;
    var $p=panelOf($scope,target); if(!$p.length) return false;
    $scope.find('.sn-tab-button,[data-sn-tab-target]').removeClass('sn-tab-button-active active').attr('aria-selected','false').attr('tabindex','-1');
    $scope.find('.sn-tab-button,[data-sn-tab-target]').filter(function(){ return targetOf($(this))===target; }).addClass('sn-tab-button-active active').attr('aria-selected','true').attr('tabindex','0');
    $scope.find('.sn-tab-panel').removeClass('sn-tab-panel-active active').attr('hidden','hidden');
    $p.addClass('sn-tab-panel-active active').removeAttr('hidden');
    return true;
  }
  $(document).off('click.snSeniorTabs128','#sn-senior-supervisor-panel .sn-tab-button,#sn-senior-supervisor-panel [data-sn-tab-target],.sn-senior-supervisor-panel .sn-tab-button,.sn-senior-supervisor-panel [data-sn-tab-target]')
    .on('click.snSeniorTabs128','#sn-senior-supervisor-panel .sn-tab-button,#sn-senior-supervisor-panel [data-sn-tab-target],.sn-senior-supervisor-panel .sn-tab-button,.sn-senior-supervisor-panel [data-sn-tab-target]',function(e){
      var $scope=$(this).closest('#sn-senior-supervisor-panel,.sn-senior-supervisor-panel,.sn-panel,.sn-portal');
      var target=targetOf($(this));
      if(activate($scope,target)){ e.preventDefault(); if(history && history.replaceState){ history.replaceState(null,'','#'+target); } }
    });
  $(function(){
    $('#sn-senior-supervisor-panel,.sn-senior-supervisor-panel').each(function(){
      var $scope=$(this), hash=clean(location.hash), initial='';
      if(hash && panelOf($scope,hash).length) initial=hash;
      if(!initial) initial=targetOf($scope.find('.sn-tab-button-active,[data-sn-tab-target].active').first()) || targetOf($scope.find('.sn-tab-button,[data-sn-tab-target]').first());
      activate($scope,initial);
    });
  });
})(jQuery);

/* Senior supervisor progressive loader: keep initial HTML request lightweight. */
(function($){
  'use strict';
  if (window.snSeniorSupervisorLazyReady) { return; }
  window.snSeniorSupervisorLazyReady = true;

  function clean(value){ return String(value || '').replace(/^#/, '').replace(/[^A-Za-z0-9_-]/g, ''); }
  function esc(value){ return $('<div>').text(String(value || '')).html(); }
  function targetOf($button){ return clean($button.attr('data-sn-tab-target') || $button.attr('aria-controls') || ''); }
  function requestFilters(data){
    try {
      var params = new URLSearchParams(window.location.search || '');
      ['sn_ss_date_from','sn_ss_date_to','sn_ss_seller','sn_ss_invoice_status','sn_ss_lead_status','sn_ss_invoice_code'].forEach(function(key){
        var value = params.get(key);
        if (value !== null && value !== '') { data[key] = value; }
      });
    } catch(e) {}
    return data;
  }
  function afterLoad(target){
    if (target === 'ss-distribution' && window.snLoadSupervisorDistributionItems) {
      setTimeout(function(){ window.snLoadSupervisorDistributionItems(1, false); }, 80);
    }
    if (target === 'ss-invoices' && window.snLoadSupervisorInvoicesFinal) {
      setTimeout(function(){ window.snLoadSupervisorInvoicesFinal(); }, 80);
    }
  }
  function load(target, force){
    target = clean(target);
    var $panel = $('#sn-senior-supervisor-panel[data-sn-senior-lazy="1"]');
    if (!$panel.length || !target) { return; }
    var $host = $panel.find('.sn-senior-lazy-tab[data-sn-tab-panel="' + target + '"]').first();
    if (!$host.length || ($host.data('loaded') && !force) || $host.data('loading')) { return; }
    $host.data('loading', 1).html('<div class="sn-card sn-loading">در حال بارگذاری این بخش...</div>');
    var data = requestFilters({action:'sn_senior_supervisor_lazy_tab', nonce:(window.snAjax && window.snAjax.nonce) || '', tab:target});
    $.ajax({
      url: (window.snAjax && snAjax.ajaxurl) || window.ajaxurl || '/wp-admin/admin-ajax.php',
      type: 'POST',
      dataType: 'json',
      timeout: 30000,
      data: data
    }).done(function(res){
      if (!res || res.success !== true) {
        var message = (res && res.data && res.data.message) || 'بارگذاری این بخش انجام نشد.';
        $host.html('<div class="sn-notice sn-error">' + esc(message) + ' <button type="button" class="sn-btn sn-btn-sm sn-senior-retry" data-tab="' + esc(target) + '">تلاش مجدد</button></div>').data('loading', 0);
        return;
      }
      $host.html((res.data && res.data.html) || '').data('loaded', 1).data('loading', 0);
      afterLoad(target);
    }).fail(function(xhr, status){
      var message = status === 'timeout' ? 'زمان پاسخ سرور برای این بخش تمام شد.' : 'خطای سرور در بارگذاری این بخش (' + String(xhr.status || 0) + ').';
      $host.html('<div class="sn-notice sn-error">' + esc(message) + ' <button type="button" class="sn-btn sn-btn-sm sn-senior-retry" data-tab="' + esc(target) + '">تلاش مجدد</button></div>').data('loading', 0);
    });
  }

  $(document).on('click.snSeniorLazyLoad', '#sn-senior-supervisor-panel [data-sn-tab-target]', function(){
    var target = targetOf($(this));
    if (target) { setTimeout(function(){ load(target, target === 'ss-repeat-actions' || target === 'ss-my-conversions'); }, 10); }
  });
  $(document).on('click.snSeniorLazyRetry', '#sn-senior-supervisor-panel .sn-senior-retry', function(){
    load(String($(this).data('tab') || ''), true);
  });
  $(function(){
    var $panel = $('#sn-senior-supervisor-panel[data-sn-senior-lazy="1"]');
    if (!$panel.length) { return; }
    var hash = clean(window.location.hash || '');
    var initial = hash && $panel.find('[data-sn-tab-panel="' + hash + '"]').length ? hash : targetOf($panel.find('[data-sn-tab-target].sn-tab-button-active,[data-sn-tab-target].active').first());
    if (!initial) { initial = 'ss-overview'; }
    load(initial, false);
  });
})(jQuery);

/* SN 1.0.129: lightweight portal tab fallback for role scripts without legacy public.js */
(function($){
  'use strict';
  if (window.snClassicTabsReady) { return; }
  window.snClassicTabsReady = true;
  function clean(v){ return String(v || '').replace(/^#/, '').replace(/[^A-Za-z0-9_-]/g, ''); }
  function panelKey($panel){
    var id = clean($panel.attr('id') || '');
    return 'sn_active_tab_' + (id || 'sn-panel');
  }
  function targetOf($btn){
    var target = $btn.attr('data-sn-tab-target') || $btn.attr('aria-controls') || '';
    if (!target && $btn.attr('href')) {
      var href = String($btn.attr('href') || '');
      if (href.indexOf('#') !== -1) target = href.substring(href.indexOf('#') + 1);
    }
    return clean(target);
  }
  function panelOf($scope, target){
    target = clean(target);
    return $scope.find('.sn-tab-panel').filter(function(){
      return clean($(this).attr('data-sn-tab-panel') || this.id) === target;
    }).first();
  }
  function activate($scope, target, save){
    target = clean(target);
    if (!$scope.length || !target) return false;
    var $panel = panelOf($scope, target);
    if (!$panel.length) return false;
    $scope.find('.sn-tab-button,[data-sn-tab-target]').each(function(){
      var on = targetOf($(this)) === target;
      $(this).toggleClass('sn-tab-button-active active', on).attr('aria-selected', on ? 'true' : 'false').attr('tabindex', on ? '0' : '-1');
    });
    $scope.find('.sn-tab-panel').each(function(){
      var on = clean($(this).attr('data-sn-tab-panel') || this.id) === target;
      $(this).toggleClass('sn-tab-panel-active active', on);
      if (on) $(this).removeAttr('hidden').show(); else $(this).attr('hidden', 'hidden').hide();
    });
    if (save) {
      try { window.localStorage.setItem(panelKey($scope), target); } catch(e) {}
      $scope.find('form').each(function(){
        var $h = $(this).find('input[name="sn_ui_active_tab"]');
        if (!$h.length) $h = $('<input>', {type:'hidden', name:'sn_ui_active_tab'}).appendTo(this);
        $h.val(target);
      });
    }
    return true;
  }
  $(document).off('click.snUniversalTabs129', '.sn-panel .sn-tab-button, .sn-panel [data-sn-tab-target], .sn-portal .sn-tab-button, .sn-portal [data-sn-tab-target]')
    .on('click.snUniversalTabs129', '.sn-panel .sn-tab-button, .sn-panel [data-sn-tab-target], .sn-portal .sn-tab-button, .sn-portal [data-sn-tab-target]', function(e){
      var $btn = $(this);
      if ($btn.closest('#sn-sales-manager-panel').length) return; // manager has its progressive loader
      if ($btn.closest('.sn-supervisor-invoice-tabs,.sn-invoice-status-tabs,.sn-subtabs').length) return;
      var $scope = $btn.closest('.sn-panel,.sn-portal');
      var target = targetOf($btn);
      if (activate($scope, target, true)) { e.preventDefault(); e.stopPropagation(); }
    });
  $(function(){
    $('.sn-panel,.sn-portal').has('.sn-tab-nav[data-sn-tabs], .sn-tabs[data-sn-tabs]').each(function(){
      var $scope = $(this);
      if ($scope.is('#sn-sales-manager-panel')) return;
      var saved = '';
      try { saved = window.localStorage.getItem(panelKey($scope)) || ''; } catch(e) {}
      var hash = clean(window.location.hash || '');
      var initial = (hash && panelOf($scope, hash).length) ? hash : '';
      if (!initial && saved && panelOf($scope, saved).length) initial = saved;
      if (!initial) initial = targetOf($scope.find('.sn-tab-button-active,[data-sn-tab-target].active').first()) || targetOf($scope.find('.sn-tab-button,[data-sn-tab-target]').first());
      activate($scope, initial, false);
    });
  });
})(jQuery);


// SN 1.0.142 lightweight sidebar/mobile menu toggles for role-specific panels
(function(){
  'use strict';
  if (window.snRoleSidebarToggle142) { return; }
  window.snRoleSidebarToggle142 = true;
  function ready(fn){ if(document.readyState === 'loading') document.addEventListener('DOMContentLoaded', fn); else fn(); }
  function clean(value){ return String(value || '').replace(/[^A-Za-z0-9_-]/g, ''); }
  function panelKey(panel){
    var id = clean(panel.getAttribute('id') || '');
    if(id) return 'sn_sidebar_collapsed_' + id;
    var cls = clean(String(panel.className || '').split(/\s+/).filter(Boolean).join('-'));
    return 'sn_sidebar_collapsed_' + (cls || 'panel');
  }
  function directSidebar(panel){
    var nodes = panel.children || [];
    for(var i=0;i<nodes.length;i++){
      if(nodes[i].classList && (nodes[i].classList.contains('sn-panel-toolbar') || nodes[i].classList.contains('sn-tabs'))){ return nodes[i]; }
    }
    return null;
  }
  function setDesktopState(panel, collapsed){
    var sidebar = directSidebar(panel); if(!sidebar) return;
    panel.classList.toggle('sn-sidebar-collapsed', !!collapsed);
    panel.classList.add('sn-sidebar-collapsible');
    var btn = sidebar.querySelector(':scope > .sn-sidebar-toggle') || sidebar.querySelector('.sn-sidebar-toggle');
    if(btn){
      btn.setAttribute('aria-expanded', collapsed ? 'false' : 'true');
      btn.setAttribute('title', collapsed ? 'باز کردن منو' : 'بستن منو');
      btn.setAttribute('aria-label', collapsed ? 'باز کردن منوی پنل' : 'بستن منوی پنل');
      btn.innerHTML = collapsed ? '☰' : '×';
    }
    try{ window.localStorage.setItem(panelKey(panel), collapsed ? '1' : '0'); }catch(e){}
  }
  function setupDesktop(panel){
    var sidebar = directSidebar(panel); if(!sidebar) return;
    panel.classList.add('sn-sidebar-collapsible', 'sn-has-sidebar-tabs');
    if(!sidebar.querySelector(':scope > .sn-sidebar-toggle') && !sidebar.querySelector('.sn-sidebar-toggle')){
      var btn = document.createElement('button'); btn.type = 'button'; btn.className = 'sn-sidebar-toggle'; btn.innerHTML = '×';
      sidebar.insertBefore(btn, sidebar.firstChild);
    }
    var stored = ''; try{ stored = window.localStorage.getItem(panelKey(panel)) || ''; }catch(e){}
    setDesktopState(panel, stored === '1');
  }
  function hasSidebar(panel){ return !!(panel && (panel.querySelector(':scope > .sn-tabs') || panel.querySelector(':scope > .sn-panel-toolbar'))); }
  function isMobile(){ return window.matchMedia && window.matchMedia('(max-width: 900px)').matches; }
  function setMobileOpen(panel, open){
    if(!panel) return;
    panel.classList.toggle('sn-mobile-sidebar-open', !!open);
    document.body.classList.toggle('sn-mobile-sidebar-lock', !!open);
    var btn = panel.querySelector(':scope > .sn-mobile-sidebar-toggle');
    if(btn){
      btn.setAttribute('aria-expanded', open ? 'true' : 'false');
      btn.setAttribute('aria-label', open ? 'بستن منوی پنل' : 'باز کردن منوی پنل');
      btn.setAttribute('title', open ? 'بستن منو' : 'باز کردن منو');
      btn.innerHTML = open ? '×' : '☰';
    }
  }
  function setupMobile(panel){
    if(!hasSidebar(panel)) return;
    if(!panel.querySelector(':scope > .sn-mobile-sidebar-toggle')){
      var btn = document.createElement('button'); btn.type = 'button'; btn.className = 'sn-mobile-sidebar-toggle';
      btn.setAttribute('aria-expanded','false'); btn.setAttribute('aria-label','باز کردن منوی پنل'); btn.setAttribute('title','باز کردن منو'); btn.innerHTML = '☰';
      panel.insertBefore(btn, panel.firstChild);
    }
    if(!isMobile()) setMobileOpen(panel, false);
  }
  function setupAll(){ document.querySelectorAll('.sn-panel').forEach(function(panel){ setupDesktop(panel); setupMobile(panel); }); }
  document.addEventListener('click', function(e){
    var mobileBtn = e.target && e.target.closest ? e.target.closest('.sn-mobile-sidebar-toggle') : null;
    if(mobileBtn){ var panel = mobileBtn.closest('.sn-panel'); if(panel){ e.preventDefault(); e.stopPropagation(); setMobileOpen(panel, !panel.classList.contains('sn-mobile-sidebar-open')); } return; }
    var innerBtn = e.target && e.target.closest ? e.target.closest('.sn-sidebar-toggle') : null;
    if(innerBtn){ var p = innerBtn.closest('.sn-panel'); if(p){ e.preventDefault(); e.stopPropagation(); if(isMobile()) setMobileOpen(p, !p.classList.contains('sn-mobile-sidebar-open')); else setDesktopState(p, !p.classList.contains('sn-sidebar-collapsed')); } return; }
    var tab = e.target && e.target.closest ? e.target.closest('.sn-panel .sn-tab[data-tab], .sn-panel .sn-tab-button, .sn-panel [data-sn-tab-target]') : null;
    if(tab && isMobile()){ var p2 = tab.closest('.sn-panel'); if(p2 && p2.classList.contains('sn-mobile-sidebar-open')) setTimeout(function(){ setMobileOpen(p2, false); }, 70); }
  }, true);
  document.addEventListener('keydown', function(e){ if(e.key === 'Escape'){ document.querySelectorAll('.sn-panel.sn-mobile-sidebar-open').forEach(function(p){ setMobileOpen(p, false); }); } });
  window.addEventListener('resize', function(){ if(!isMobile()) document.querySelectorAll('.sn-panel.sn-mobile-sidebar-open').forEach(function(p){ setMobileOpen(p, false); }); });
  ready(function(){ setupAll(); setTimeout(setupAll, 100); setTimeout(setupAll, 400); });
  window.snSetupSidebarToggles = window.snSetupSidebarToggles || setupAll;
  window.snSetupMobileDrawerNav = window.snSetupMobileDrawerNav || setupAll;
})();


/* SN 1.0.187 — Dot supervisor converter assignment: single + bulk Ajax, no page reload. */
(function($){
  'use strict';
  if (window.snDotSupervisorAssignAjaxReady) { return; }
  window.snDotSupervisorAssignAjaxReady = true;
  var cfg = window.snAjax || window.snData || {};
  var ajax = cfg.ajaxurl || window.ajaxurl || '/wp-admin/admin-ajax.php';
  var nonce = cfg.nonce || '';
  function esc(v){ return String(v == null ? '' : v).replace(/[&<>"']/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#039;'}[c];}); }
  function faNum(v){
    try { return Number(v || 0).toLocaleString('fa-IR'); } catch(e) { return String(v || 0); }
  }
  function notice(type, message){
    var $host = $('#sn-tab-needs-action .sn-dot-assign-notice').first();
    if (!$host.length) { $host = $('.sn-dot-supervisor .sn-dot-assign-notice').first(); }
    if (!$host.length) { return; }
    $host.html('<div class="sn-notice ' + (type === 'success' ? 'sn-success' : 'sn-error') + '">' + esc(message) + '</div>');
    window.setTimeout(function(){ $host.find('.sn-notice').fadeOut(180, function(){ $(this).remove(); }); }, 5500);
  }
  function syncBulk($scope){
    var $checks = $scope.find('.sn-dot-case-check:enabled');
    var $selected = $checks.filter(':checked');
    $scope.find('[data-sn-dot-selected-count]').text(faNum($selected.length));
    var converter = String($scope.find('.sn-dot-bulk-converter').val() || '');
    $scope.find('.sn-dot-bulk-assign-btn').prop('disabled', !$selected.length || !converter);
    var $all = $scope.find('.sn-dot-select-all');
    if ($all.length) {
      $all.prop('checked', !!$checks.length && $selected.length === $checks.length);
      $all.prop('indeterminate', $selected.length > 0 && $selected.length < $checks.length);
    }
  }
  function applyAssigned(caseId, converterId, converterName){
    var $row = $('.sn-dot-supervisor-ready-table tr[data-sn-dot-case-row="' + caseId + '"]');
    if (!$row.length) { return; }
    $row.find('.sn-dot-assigned-converter').text(converterName || ('#' + converterId));
    var $form = $row.find('.sn-dot-assign-form');
    $form.find('select[name="converter_id"]').val(String(converterId));
    $form.find('button[type="submit"]').text('تغییر تخصیص');
    var $status = $row.find('.sn-badge').first();
    if ($status.length && $.trim($status.text()) === 'آماده تبدیل') { $status.text('تخصیص به تبدیل‌کننده'); }
    $row.addClass('sn-dot-assigned-flash');
    window.setTimeout(function(){ $row.removeClass('sn-dot-assigned-flash'); }, 1200);
  }

  $(document).off('submit.snDotAssignAjax', '.sn-dot-supervisor .sn-dot-assign-form').on('submit.snDotAssignAjax', '.sn-dot-supervisor .sn-dot-assign-form', function(e){
    e.preventDefault();
    var $form = $(this);
    var $btn = $form.find('button[type="submit"]');
    var converterId = String($form.find('select[name="converter_id"]').val() || '');
    if (!converterId) { notice('error', 'ابتدا یک تبدیل‌کننده انتخاب کنید.'); return; }
    if ($form.data('busy')) { return; }
    $form.data('busy', 1); $btn.prop('disabled', true).attr('aria-busy','true');
    $.ajax({url: ajax, type:'POST', dataType:'json', data:$form.serialize()})
      .done(function(res){
        if (!res || !res.success) { notice('error', (res && res.data && res.data.message) || 'تخصیص انجام نشد.'); return; }
        var caseId = parseInt($form.find('input[name="case_id"]').val(),10) || 0;
        var converterName = $form.find('select[name="converter_id"] option:selected').text().replace(/\s*\(#\d+\)\s*$/,'');
        applyAssigned(caseId, converterId, converterName);
        notice('success', (res.data && res.data.message) || 'تخصیص انجام شد.');
      })
      .fail(function(xhr){ var msg = xhr && xhr.responseJSON && xhr.responseJSON.data && xhr.responseJSON.data.message; notice('error', msg || 'ارتباط با سرور برای تخصیص انجام نشد.'); })
      .always(function(){ $form.data('busy',0); $btn.prop('disabled',false).removeAttr('aria-busy'); });
  });

  $(document).off('change.snDotBulkCheck', '.sn-dot-supervisor .sn-dot-case-check, .sn-dot-supervisor .sn-dot-bulk-converter').on('change.snDotBulkCheck', '.sn-dot-supervisor .sn-dot-case-check, .sn-dot-supervisor .sn-dot-bulk-converter', function(){
    syncBulk($(this).closest('.sn-dot-supervisor-group.is-ready'));
  });
  $(document).off('change.snDotBulkAll', '.sn-dot-supervisor .sn-dot-select-all').on('change.snDotBulkAll', '.sn-dot-supervisor .sn-dot-select-all', function(){
    var $scope = $(this).closest('.sn-dot-supervisor-group.is-ready');
    $scope.find('.sn-dot-case-check:enabled').prop('checked', this.checked);
    syncBulk($scope);
  });
  $(document).off('click.snDotBulkAssign', '.sn-dot-supervisor .sn-dot-bulk-assign-btn').on('click.snDotBulkAssign', '.sn-dot-supervisor .sn-dot-bulk-assign-btn', function(){
    var $btn = $(this), $scope = $btn.closest('.sn-dot-supervisor-group.is-ready');
    if ($btn.data('busy')) { return; }
    var ids = $scope.find('.sn-dot-case-check:checked').map(function(){ return parseInt(this.value,10) || 0; }).get().filter(Boolean);
    var converterId = parseInt($scope.find('.sn-dot-bulk-converter').val(),10) || 0;
    if (!ids.length || !converterId) { notice('error','پرونده‌ها و تبدیل‌کننده را انتخاب کنید.'); return; }
    var converterName = $scope.find('.sn-dot-bulk-converter option:selected').text().replace(/\s*\(#\d+\)\s*$/,'');
    $btn.data('busy',1).prop('disabled',true).attr('aria-busy','true').text('در حال تخصیص...');
    $.ajax({url:ajax,type:'POST',dataType:'json',data:{action:'sn_dot_bulk_assign_converter',nonce:nonce,case_ids:ids,converter_id:converterId}})
      .done(function(res){
        if (!res || !res.success) { notice('error',(res && res.data && res.data.message) || 'تخصیص گروهی انجام نشد.'); return; }
        var doneIds = (res.data && res.data.case_ids) || ids;
        var name = (res.data && res.data.converter_name) || converterName;
        doneIds.forEach(function(id){ applyAssigned(id, converterId, name); });
        $scope.find('.sn-dot-case-check:checked').prop('checked',false);
        syncBulk($scope);
        notice('success',(res.data && res.data.message) || 'تخصیص گروهی انجام شد.');
      })
      .fail(function(xhr){ var msg=xhr&&xhr.responseJSON&&xhr.responseJSON.data&&xhr.responseJSON.data.message; notice('error',msg || 'ارتباط با سرور برای تخصیص گروهی انجام نشد.'); })
      .always(function(){ $btn.data('busy',0).removeAttr('aria-busy').text('تخصیص پرونده‌های انتخاب‌شده'); syncBulk($scope); });
  });
})(jQuery);
