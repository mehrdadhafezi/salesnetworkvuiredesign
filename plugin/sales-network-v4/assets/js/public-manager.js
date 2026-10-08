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
  function loadSupervisorInvoices(q){
    if(!$('#sn-supervisor-invoice-list').length) return;
    $('#sn-supervisor-invoice-list').html('در حال بارگذاری پیش‌فاکتورها...');
    $.post(ajax,{action:'sn_supervisor_invoices',nonce:nonce,q:q||''},function(res){
      if(!res||!res.success){ $('#sn-supervisor-invoice-list').html('❌ '+snEsc((res&&res.message)||'خطا در دریافت پیش‌فاکتورها')); return; }
      var rows=res.items||[];
      var html='<table class="sn-table"><thead><tr><th>کد</th><th>مشتری</th><th>فروشنده</th><th>مبلغ</th><th>پرداخت</th><th>وضعیت</th><th>فیش سرپرست</th></tr></thead><tbody>';
      if(!rows.length) html+='<tr><td colspan="7">موردی یافت نشد.</td></tr>';
      rows.forEach(function(i){ html+='<tr><td><code>'+snEsc(i.invoice_code||'')+'</code></td><td>'+snEsc(i.customer_name||'')+'<br><small>'+snEsc(i.customer_phone||'')+'</small></td><td>'+snEsc(i.seller_name||'—')+'</td><td>'+snEsc(i.amount_fmt||i.product_price||0)+'</td><td>'+snEsc(i.pay_method_label||'—')+'</td><td>'+snEsc(i.status_label||snFaStatus(i.status))+'</td><td><input type="file" class="sn-supervisor-receipt-file" data-id="'+i.id+'" accept="image/*,application/pdf"><button type="button" class="sn-btn sn-btn-sm sn-supervisor-upload-receipt" data-id="'+i.id+'">آپلود فیش</button><div class="sn-supervisor-upload-msg" data-id="'+i.id+'"></div></td></tr>'; });
      html+='</tbody></table>'; $('#sn-supervisor-invoice-list').html(html);
    }).fail(function(xhr){ $('#sn-supervisor-invoice-list').html('❌ خطای سرور: '+xhr.status); });
  }
  $(document).on('click','.sn-tab[data-tab="invoices"]',function(){loadSupervisorInvoices($('#sn-supervisor-invoice-search').val());});
  $(document).on('input','#sn-supervisor-invoice-search',function(){clearTimeout(window.snSupInvTimer);var q=$(this).val();window.snSupInvTimer=setTimeout(function(){loadSupervisorInvoices(q);},450);});
  if($('#sn-supervisor-invoice-list').length) loadSupervisorInvoices('');

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
    data.limit = 60; // 1.0.126: bounded for speed
    $.ajax({url: ajax, type: 'POST', dataType: 'json', timeout: 25000, data: data}).done(function(res){
      var payload = (res && res.data) ? res.data : (res || {});
      if(!res || !res.success){
        $('#sn-manager-leads-list').html('❌ ' + snEsc((payload && payload.message) || (res && res.message) || 'خطا در دریافت گزارش'));
        return;
      }
      $('#sn-manager-total').text(payload.total || res.total || 0);
      try {
      var rows = payload.items || res.items || [];
      var summary = '<div class="sn-kpi-grid sn-manager-report-kpis">'+
        '<div class="sn-kpi-card"><small>کل گزارش</small><strong>'+snEsc(payload.total || res.total || 0)+'</strong><em>سرنخ + دیتای V4</em></div>'+
        '<div class="sn-kpi-card"><small>سرنخ‌های قدیمی</small><strong>'+snEsc(payload.legacy_total || 0)+'</strong><em>sn_leads</em></div>'+
        '<div class="sn-kpi-card"><small>دیتای زنجیره فروش</small><strong>'+snEsc(payload.v4_total || 0)+'</strong><em>distribution</em></div>'+
        '<div class="sn-kpi-card"><small>فاکتورهای محدوده</small><strong>'+snEsc(payload.invoice_total || 0)+'</strong><em>گزارش فاکتورها</em></div>'+
      '</div>';
      var html = summary + '<table class="sn-table sn-manager-table"><thead><tr><th>شماره</th><th>کد</th><th>موقعیت</th><th>وضعیت</th><th>سرپرست</th><th>فروشنده</th><th>ورود</th><th>تخصیص</th></tr></thead><tbody>';
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
  $(document).on('click','.sn-supervisor-upload-receipt',function(e){ e.preventDefault(); var id=$(this).data('id'), file=$('.sn-supervisor-receipt-file[data-id="'+id+'"]').get(0).files[0]; if(!file){alert('لطفاً فایل فیش را انتخاب کنید');return;} var fd=new FormData(); fd.append('action','sn_supervisor_upload_receipt'); fd.append('nonce',nonce); fd.append('invoice_id',id); fd.append('receipt',file); var $btn=$(this).prop('disabled',true).text('در حال آپلود...'); $.ajax({url:ajax,type:'POST',data:fd,processData:false,contentType:false,success:function(res){$btn.prop('disabled',false).text('آپلود فیش');$('.sn-supervisor-upload-msg[data-id="'+id+'"]').html(res&&res.success?'✅ '+snEsc(res.message||'انجام شد'):'❌ '+snEsc((res&&res.message)||'خطا')); if(res&&res.success) loadSupervisorInvoices($('#sn-supervisor-invoice-search').val());},error:function(xhr){$btn.prop('disabled',false).text('آپلود فیش');alert('خطای سرور: '+xhr.status);}}); });
  $(document).on('click','.sn-fin-approve',function(e){e.preventDefault();var id=$(this).data('id'),$btn=$(this),$row=$btn.closest('tr');$btn.prop('disabled',true).text('در حال تایید...');$.post(ajax,{action:'sn_financial_approve_payment',nonce:nonce,invoice_id:id},function(res){if(res&&res.success){$row.find('td').eq(5).text((res.status_label||'تایید شده'));$row.find('td').last().html('<span class="sn-notice sn-success">تایید شد</span>');}else{$btn.prop('disabled',false).text('تایید');alert((res&&res.message)||'خطا');}}).fail(function(xhr){$btn.prop('disabled',false).text('تایید');alert('خطای سرور: '+xhr.status);});});
  $(document).on('click','.sn-fin-reject',function(e){e.preventDefault();var id=$(this).data('id'),reason=prompt('دلیل رد پرداخت را وارد کنید:');if(!reason)return;var $btn=$(this),$row=$btn.closest('tr');$btn.prop('disabled',true).text('در حال رد...');$.post(ajax,{action:'sn_financial_reject_payment',nonce:nonce,invoice_id:id,reason:reason},function(res){if(res&&res.success){$row.find('td').eq(5).text((res.status_label||'رد شده'));$row.find('td').last().html('<span class="sn-notice sn-error">رد شد</span>');}else{$btn.prop('disabled',false).text('رد');alert((res&&res.message)||'خطا');}}).fail(function(xhr){$btn.prop('disabled',false).text('رد');alert('خطای سرور: '+xhr.status);});});

  // ---- 1.0.55: تایید مالی همراه ویرایش + بررسی درخواست اضافه کردن شماره ----
  function snFinancePromptEditAndApprove(id){
    var note=prompt('اگر قبل از تایید، فاکتور را اصلاح می‌کنید دلیل/توضیح ویرایش را بنویسید. خالی هم قابل قبول است:','');
    if(note===null) return;
    var name=prompt('نام مشتری جدید را وارد کنید یا خالی بگذارید بدون تغییر بماند:',''); if(name===null) return;
    var phone=prompt('موبایل مشتری جدید را وارد کنید یا خالی بگذارید بدون تغییر بماند:',''); if(phone===null) return;
    var amount=prompt('مبلغ واریزی/فاکتور را وارد کنید یا خالی بگذارید بدون تغییر بماند:',''); if(amount===null) return;
    var from=prompt('۴ رقم کارت مبدا را وارد کنید یا خالی بگذارید بدون تغییر بماند:',''); if(from===null) return;
    var data={action:'sn_financial_approve_payment',nonce:nonce,invoice_id:id,finance_edit_note:note};
    if(name) data.customer_name=name; if(phone) data.customer_phone=phone; if(amount) data.manual_amount=amount; if(from) data.manual_card_from=from;
    $.post(ajax,data,function(res){ alert((res&&res.message)||'انجام شد'); if(res&&res.success){ reloadFinancialPanel(); } }).fail(function(xhr){ alert('خطای سرور: '+xhr.status); });
  }
  $(document).on('click','.sn-fin-edit-approve',function(e){ e.preventDefault(); snFinancePromptEditAndApprove($(this).data('id')); });
  $(document).on('click','.sn-extra-number-approve,.sn-extra-number-reject',function(){
    var id=$(this).data('id'), approve=$(this).hasClass('sn-extra-number-approve'), reason='';
    if(!approve){ reason=prompt('دلیل رد را بنویسید:',''); if(reason===null) return; }
    $.post(ajax,{action:'sn_manager_review_extra_number',nonce:nonce,request_id:id,decision:approve?'approve':'reject',reason:reason},function(res){ alert((res&&res.message)||'انجام شد'); if(res&&res.success){ location.reload(); } }).fail(function(xhr){ alert('خطای سرور: '+xhr.status); });
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

  // ---- 1.0.55: تایید مالی همراه ویرایش + بررسی درخواست اضافه کردن شماره ----
  function snFinancePromptEditAndApprove(id){
    var note=prompt('اگر قبل از تایید، فاکتور را اصلاح می‌کنید دلیل/توضیح ویرایش را بنویسید. خالی هم قابل قبول است:','');
    if(note===null) return;
    var name=prompt('نام مشتری جدید را وارد کنید یا خالی بگذارید بدون تغییر بماند:',''); if(name===null) return;
    var phone=prompt('موبایل مشتری جدید را وارد کنید یا خالی بگذارید بدون تغییر بماند:',''); if(phone===null) return;
    var amount=prompt('مبلغ واریزی/فاکتور را وارد کنید یا خالی بگذارید بدون تغییر بماند:',''); if(amount===null) return;
    var from=prompt('۴ رقم کارت مبدا را وارد کنید یا خالی بگذارید بدون تغییر بماند:',''); if(from===null) return;
    var data={action:'sn_financial_approve_payment',nonce:nonce,invoice_id:id,finance_edit_note:note};
    if(name) data.customer_name=name; if(phone) data.customer_phone=phone; if(amount) data.manual_amount=amount; if(from) data.manual_card_from=from;
    $.post(ajax,data,function(res){ alert((res&&res.message)||'انجام شد'); if(res&&res.success){ reloadFinancialPanel(); } }).fail(function(xhr){ alert('خطای سرور: '+xhr.status); });
  }
  $(document).on('click','.sn-fin-edit-approve',function(e){ e.preventDefault(); snFinancePromptEditAndApprove($(this).data('id')); });
  $(document).on('click','.sn-extra-number-approve,.sn-extra-number-reject',function(){
    var id=$(this).data('id'), approve=$(this).hasClass('sn-extra-number-approve'), reason='';
    if(!approve){ reason=prompt('دلیل رد را بنویسید:',''); if(reason===null) return; }
    $.post(ajax,{action:'sn_manager_review_extra_number',nonce:nonce,request_id:id,decision:approve?'approve':'reject',reason:reason},function(res){ alert((res&&res.message)||'انجام شد'); if(res&&res.success){ location.reload(); } }).fail(function(xhr){ alert('خطای سرور: '+xhr.status); });
  });

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
  $(document).ajaxComplete(function(){ setTimeout(translateAllLabels,20); });
  $(translateAllLabels);

  // ---- 1.0.55: تایید مالی همراه ویرایش + بررسی درخواست اضافه کردن شماره ----
  function snFinancePromptEditAndApprove(id){
    var note=prompt('اگر قبل از تایید، فاکتور را اصلاح می‌کنید دلیل/توضیح ویرایش را بنویسید. خالی هم قابل قبول است:','');
    if(note===null) return;
    var name=prompt('نام مشتری جدید را وارد کنید یا خالی بگذارید بدون تغییر بماند:',''); if(name===null) return;
    var phone=prompt('موبایل مشتری جدید را وارد کنید یا خالی بگذارید بدون تغییر بماند:',''); if(phone===null) return;
    var amount=prompt('مبلغ واریزی/فاکتور را وارد کنید یا خالی بگذارید بدون تغییر بماند:',''); if(amount===null) return;
    var from=prompt('۴ رقم کارت مبدا را وارد کنید یا خالی بگذارید بدون تغییر بماند:',''); if(from===null) return;
    var data={action:'sn_financial_approve_payment',nonce:nonce,invoice_id:id,finance_edit_note:note};
    if(name) data.customer_name=name; if(phone) data.customer_phone=phone; if(amount) data.manual_amount=amount; if(from) data.manual_card_from=from;
    $.post(ajax,data,function(res){ alert((res&&res.message)||'انجام شد'); if(res&&res.success){ reloadFinancialPanel(); } }).fail(function(xhr){ alert('خطای سرور: '+xhr.status); });
  }
  $(document).on('click','.sn-fin-edit-approve',function(e){ e.preventDefault(); snFinancePromptEditAndApprove($(this).data('id')); });
  $(document).on('click','.sn-extra-number-approve,.sn-extra-number-reject',function(){
    var id=$(this).data('id'), approve=$(this).hasClass('sn-extra-number-approve'), reason='';
    if(!approve){ reason=prompt('دلیل رد را بنویسید:',''); if(reason===null) return; }
    $.post(ajax,{action:'sn_manager_review_extra_number',nonce:nonce,request_id:id,decision:approve?'approve':'reject',reason:reason},function(res){ alert((res&&res.message)||'انجام شد'); if(res&&res.success){ location.reload(); } }).fail(function(xhr){ alert('خطای سرور: '+xhr.status); });
  });

})(jQuery);

/* SN 1.0.8 financial tabbed panel */
(function($){
  'use strict';
  if(!$('#sn-financial-panel').length || $('#sn-financial-panel').attr('data-finance-v2')==='1') return;
  var ajax=(window.snAjax||window.snData||{}).ajaxurl, nonce=(window.snAjax||window.snData||{}).nonce;
  function esc(v){return String(v==null?'':v).replace(/[&<>"']/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#039;'}[c];});}
  function loadFinancial(tab){
    tab=tab||$('.sn-financial-tabs .sn-subtab.active').data('tab')||'needs_review';
    $('#sn-financial-list').html('<div class="sn-loading">در حال بارگذاری...</div>');
    $.post(ajax,{action:'sn_financial_invoices',nonce:nonce,tab:tab,limit:30},function(res){
      if(!res || !res.success){ $('#sn-financial-list').html('<p class="sn-notice sn-error">'+esc((res&&res.message)||'خطا در دریافت اطلاعات مالی')+'</p>'); return; }
      var rows=(res&&res.items)||[]; var k=(res&&res.kpi)||{};
      $('#sn-financial-kpis').html(
        '<div class="sn-kpi-card sn-stat-card"><small>در انتظار بررسی</small><strong>'+esc(k.needs_review||0)+'</strong><em>'+esc(k.needs_amount_fmt||'۰ تومان')+'</em></div>'+
        '<div class="sn-kpi-card sn-stat-card"><small>پرداخت‌شده درگاهی</small><strong>'+esc(k.online_paid||0)+'</strong><em>پرداخت آنلاین/درگاه</em></div>'+
        '<div class="sn-kpi-card sn-stat-card"><small>فیش‌های آپلودشده</small><strong>'+esc(k.receipt_uploaded||0)+'</strong><em>کارت‌به‌کارت/فیش</em></div>'+
        '<div class="sn-kpi-card sn-stat-card"><small>تایید شده مالی</small><strong>'+esc(k.approved||0)+'</strong><em>'+esc(k.approved_amount_fmt||'۰ تومان')+'</em></div>'+
        '<div class="sn-kpi-card sn-stat-card"><small>رد شده مالی</small><strong>'+esc(k.rejected||0)+'</strong><em>نیازمند پیگیری فروشنده</em></div>'+
        '<div class="sn-kpi-card sn-stat-card"><small>برگشتی به فروشنده</small><strong>'+esc(k.returned_to_seller||0)+'</strong><em>وضعیت برگشت مالی موجود</em></div>'+
        '<div class="sn-kpi-card sn-stat-card"><small>تعداد این تب</small><strong>'+esc(k.count||0)+'</strong><em>نمایش صفحه جاری</em></div>'+
        '<div class="sn-kpi-card sn-stat-card"><small>مبلغ این تب</small><strong>'+esc(k.amount_fmt||0)+'</strong><em>بر اساس فیلتر فعال</em></div>'
      );
      if(!res || !res.success || !rows.length){ $('#sn-financial-list').html('<p class="sn-notice">موردی در این تب وجود ندارد.</p>'); return; }
      var html='<div class="sn-table-wrap"><table class="sn-table"><thead><tr><th>کد فاکتور</th><th>مشتری</th><th>فروشنده</th><th>مبلغ کل فاکتور</th><th>مبلغ پرداخت فعلی</th><th>تاییدشده / مانده</th><th>روش پرداخت</th><th>فیش / اطلاعات واریز</th><th>وضعیت</th><th>عملیات</th></tr></thead><tbody>';
      rows.forEach(function(i){ var ops='<button type="button" class="sn-btn sn-btn-sm sn-btn-secondary sn-fin-details" data-id="'+esc(i.id)+'">جزئیات فاکتور</button> '; if(i.can_approve_payment){ ops+='<button type="button" class="sn-btn sn-btn-sm sn-fin-approve" data-id="'+esc(i.id)+'">تایید</button> <button type="button" class="sn-btn sn-btn-sm sn-fin-edit-approve" data-id="'+esc(i.id)+'">ویرایش و تایید</button> '; } if(i.can_reject_payment){ ops+='<button type="button" class="sn-btn sn-btn-sm sn-fin-reject" data-id="'+esc(i.id)+'">رد</button>'; } var rejectReason=i.financial_reject_reason||i.rejected_reason||''; if(rejectReason){ ops+='<div class="sn-reject-meta"><strong>دلیل رد:</strong> '+esc(rejectReason)+'<small>'+esc(i.financial_rejected_by_name||'')+(i.financial_rejected_at?' - '+esc(i.financial_rejected_at):'')+'</small></div>'; } if(i.financial_return_label||i.seller_resend_note||i.seller_resend_receipt_reuploaded){ ops+='<div class="sn-resubmit-meta"><strong>'+esc(i.financial_return_label||'ارسال مجدد توسط فروشنده')+'</strong>'+(i.seller_resend_note?'<p>'+esc(i.seller_resend_note)+'</p>':'')+(i.seller_resend_receipt_reuploaded?'<small>فیش جدید بارگذاری شده است.</small>':'')+(i.seller_resend_at?'<small>'+esc(i.seller_resend_at)+'</small>':'')+'</div>'; } var pinfo=i.payment_info_html||i.payment_info_text||''; if(i.receipt_url && !i.payment_info_html){pinfo='<a target="_blank" href="'+esc(i.receipt_url)+'">مشاهده فیش</a>'+(i.manual_card_to_display?'<br><small>۴ رقم آخر کارت مقصد: <span dir="ltr">'+esc(i.manual_card_to_display)+'</span></small>':''); } html+='<tr><td><code>'+esc(i.invoice_code)+'</code></td><td>'+esc(i.customer_name)+'<br><small>'+esc(i.customer_phone)+'</small></td><td>'+esc(i.seller_name||i.seller_id||'—')+'</td><td><strong>'+esc(i.invoice_total_amount_fmt||i.amount_fmt||i.product_price)+'</strong></td><td><strong>'+esc(i.current_payment_amount_fmt||'۰ تومان')+'</strong><br><small>مرحله '+esc(i.latest_payment_stage_no||i.current_payment_stage||1)+'</small></td><td><small>تاییدشده: '+esc(i.approved_paid_amount_fmt||'۰ تومان')+'</small><br><small>مانده: '+esc(i.remaining_amount_fmt||'۰ تومان')+'</small></td><td>'+esc(i.pay_method_label)+' / '+esc(i.payment_source_label||i.uploaded_by_type||'')+'</td><td>'+(i.payment_info_html?pinfo:(i.receipt_url?pinfo:esc(pinfo||'—')))+'</td><td class="sn-finance-status-cell"><span class="sn-status sn-status-'+esc(i.status||'')+'">'+esc(i.status_label||i.status)+'</span></td><td>'+ops+'</td></tr>'; });
      html+='</tbody></table></div>'; $('#sn-financial-list').html(html);
    }).fail(function(xhr){
      $('#sn-financial-list').html('<p class="sn-notice sn-error">خطای سرور مالی: '+esc(xhr&&xhr.status?xhr.status:'نامشخص')+'</p>');
    });
  }
  $(document).on('click','.sn-financial-tabs .sn-subtab',function(){ $('.sn-financial-tabs .sn-subtab').removeClass('active'); $(this).addClass('active'); loadFinancial($(this).data('tab')); });
  $(document).on('click','.sn-fin-approve',function(){ $.post(ajax,{action:'sn_financial_approve_payment',nonce:nonce,invoice_id:$(this).data('id')},function(res){alert((res&&res.message)||'انجام شد'); loadFinancial();}); });
  $(document).on('click','.sn-fin-reject',function(){ var reason=prompt('دلیل رد پرداخت را وارد کنید:'); if(!reason) return; $.post(ajax,{action:'sn_financial_reject_payment',nonce:nonce,invoice_id:$(this).data('id'),reason:reason},function(res){alert((res&&res.message)||'انجام شد'); loadFinancial();}); });
  $(loadFinancial);

  // ---- 1.0.55: تایید مالی همراه ویرایش + بررسی درخواست اضافه کردن شماره ----
  function snFinancePromptEditAndApprove(id){
    var note=prompt('اگر قبل از تایید، فاکتور را اصلاح می‌کنید دلیل/توضیح ویرایش را بنویسید. خالی هم قابل قبول است:','');
    if(note===null) return;
    var name=prompt('نام مشتری جدید را وارد کنید یا خالی بگذارید بدون تغییر بماند:',''); if(name===null) return;
    var phone=prompt('موبایل مشتری جدید را وارد کنید یا خالی بگذارید بدون تغییر بماند:',''); if(phone===null) return;
    var amount=prompt('مبلغ واریزی/فاکتور را وارد کنید یا خالی بگذارید بدون تغییر بماند:',''); if(amount===null) return;
    var from=prompt('۴ رقم کارت مبدا را وارد کنید یا خالی بگذارید بدون تغییر بماند:',''); if(from===null) return;
    var data={action:'sn_financial_approve_payment',nonce:nonce,invoice_id:id,finance_edit_note:note};
    if(name) data.customer_name=name; if(phone) data.customer_phone=phone; if(amount) data.manual_amount=amount; if(from) data.manual_card_from=from;
    $.post(ajax,data,function(res){ alert((res&&res.message)||'انجام شد'); if(res&&res.success){ reloadFinancialPanel(); } }).fail(function(xhr){ alert('خطای سرور: '+xhr.status); });
  }
  $(document).on('click','.sn-fin-edit-approve',function(e){ e.preventDefault(); snFinancePromptEditAndApprove($(this).data('id')); });
  $(document).on('click','.sn-extra-number-approve,.sn-extra-number-reject',function(){
    var id=$(this).data('id'), approve=$(this).hasClass('sn-extra-number-approve'), reason='';
    if(!approve){ reason=prompt('دلیل رد را بنویسید:',''); if(reason===null) return; }
    $.post(ajax,{action:'sn_manager_review_extra_number',nonce:nonce,request_id:id,decision:approve?'approve':'reject',reason:reason},function(res){ alert((res&&res.message)||'انجام شد'); if(res&&res.success){ location.reload(); } }).fail(function(xhr){ alert('خطای سرور: '+xhr.status); });
  });

})(jQuery);


/* SN 1.0.23 financial reject modal + stable operations */
(function($){
  'use strict';
  var cfg=window.snAjax||window.snData||{}; var ajax=cfg.ajaxurl||window.ajaxurl||'/wp-admin/admin-ajax.php'; var nonce=cfg.nonce||''; var currentRejectId=0;
  function ensureRejectModal(){
    if($('#sn-fin-reject-modal').length) return;
    $('body').append('<div id="sn-fin-reject-modal" class="sn-modal sn-lite-modal" aria-hidden="true"><div class="sn-modal-backdrop sn-fin-reject-close"></div><div class="sn-modal-card sn-fin-reject-card"><div class="sn-modal-head"><h3>دلیل رد پرداخت</h3><button type="button" class="sn-modal-x sn-fin-reject-close">×</button></div><div class="sn-modal-body"><p class="sn-muted">دلیل رد شدن برای فروشنده نمایش داده می‌شود.</p><textarea id="sn-fin-reject-reason" rows="5" placeholder="دلیل رد پرداخت را بنویسید..." style="width:100%;box-sizing:border-box"></textarea><div id="sn-fin-reject-msg"></div><div class="sn-modal-actions"><button type="button" class="sn-btn sn-btn-secondary sn-fin-reject-close">انصراف</button><button type="button" class="sn-btn sn-btn-danger" id="sn-fin-reject-submit">ثبت رد پرداخت</button></div></div></div></div>');
  }
  function reloadFinancialPanel(){
    var $active=$('.sn-financial-tabs .sn-subtab.active');
    if($active.length){ $active.trigger('click'); }
    else if($('#sn-financial-panel').length){ location.reload(); }
  }
  $(document).off('click','.sn-fin-approve').on('click','.sn-fin-approve',function(e){
    e.preventDefault(); e.stopImmediatePropagation();
    var id=$(this).data('id'), $btn=$(this), $row=$btn.closest('tr');
    $btn.prop('disabled',true).text('در حال تایید...');
    $.post(ajax,{action:'sn_financial_approve_payment',nonce:nonce,invoice_id:id},function(res){
      if(res&&res.success){ $row.find('td').eq(5).text((res.status_label||'تایید شده')); $row.find('td').last().html('<span class="sn-notice sn-success">✅ تایید شد</span>'); setTimeout(reloadFinancialPanel,350); }
      else { $btn.prop('disabled',false).text('تایید'); alert((res&&res.message)||'خطا'); }
    }).fail(function(xhr){ $btn.prop('disabled',false).text('تایید'); alert('خطای سرور: '+xhr.status); });
  });
  $(document).off('click','.sn-fin-reject').on('click','.sn-fin-reject',function(e){
    e.preventDefault(); e.stopImmediatePropagation();
    ensureRejectModal(); currentRejectId=$(this).data('id'); $('#sn-fin-reject-reason').val(''); $('#sn-fin-reject-msg').empty(); $('#sn-fin-reject-modal').fadeIn(120).attr('aria-hidden','false'); setTimeout(function(){ $('#sn-fin-reject-reason').focus(); },150);
  });
  $(document).on('click','.sn-fin-reject-close',function(){ $('#sn-fin-reject-modal').fadeOut(120).attr('aria-hidden','true'); });
  $(document).on('click','#sn-fin-reject-submit',function(){
    var reason=String($('#sn-fin-reject-reason').val()||'').trim();
    if(!reason){ $('#sn-fin-reject-msg').html('<p class="sn-notice sn-error">لطفاً دلیل رد پرداخت را وارد کنید.</p>'); return; }
    var $btn=$(this).prop('disabled',true).text('در حال ثبت...');
    $.post(ajax,{action:'sn_financial_reject_payment',nonce:nonce,invoice_id:currentRejectId,reason:reason},function(res){
      $btn.prop('disabled',false).text('ثبت رد پرداخت');
      if(res&&res.success){ $('#sn-fin-reject-msg').html('<p class="sn-notice sn-success">✅ پرداخت رد شد.</p>'); setTimeout(function(){ $('#sn-fin-reject-modal').fadeOut(120); reloadFinancialPanel(); },500); }
      else { $('#sn-fin-reject-msg').html('<p class="sn-notice sn-error">'+esc((res&&res.message)||'خطا در ثبت')+'</p>'); }
    }).fail(function(xhr){ $btn.prop('disabled',false).text('ثبت رد پرداخت'); $('#sn-fin-reject-msg').html('<p class="sn-notice sn-error">خطای سرور: '+xhr.status+'</p>'); });
  });

  // ---- 1.0.55: تایید مالی همراه ویرایش + بررسی درخواست اضافه کردن شماره ----
  function snFinancePromptEditAndApprove(id){
    var note=prompt('اگر قبل از تایید، فاکتور را اصلاح می‌کنید دلیل/توضیح ویرایش را بنویسید. خالی هم قابل قبول است:','');
    if(note===null) return;
    var name=prompt('نام مشتری جدید را وارد کنید یا خالی بگذارید بدون تغییر بماند:',''); if(name===null) return;
    var phone=prompt('موبایل مشتری جدید را وارد کنید یا خالی بگذارید بدون تغییر بماند:',''); if(phone===null) return;
    var amount=prompt('مبلغ واریزی/فاکتور را وارد کنید یا خالی بگذارید بدون تغییر بماند:',''); if(amount===null) return;
    var from=prompt('۴ رقم کارت مبدا را وارد کنید یا خالی بگذارید بدون تغییر بماند:',''); if(from===null) return;
    var data={action:'sn_financial_approve_payment',nonce:nonce,invoice_id:id,finance_edit_note:note};
    if(name) data.customer_name=name; if(phone) data.customer_phone=phone; if(amount) data.manual_amount=amount; if(from) data.manual_card_from=from;
    $.post(ajax,data,function(res){ alert((res&&res.message)||'انجام شد'); if(res&&res.success){ reloadFinancialPanel(); } }).fail(function(xhr){ alert('خطای سرور: '+xhr.status); });
  }
  $(document).on('click','.sn-fin-edit-approve',function(e){ e.preventDefault(); snFinancePromptEditAndApprove($(this).data('id')); });
  $(document).on('click','.sn-extra-number-approve,.sn-extra-number-reject',function(){
    var id=$(this).data('id'), approve=$(this).hasClass('sn-extra-number-approve'), reason='';
    if(!approve){ reason=prompt('دلیل رد را بنویسید:',''); if(reason===null) return; }
    $.post(ajax,{action:'sn_manager_review_extra_number',nonce:nonce,request_id:id,decision:approve?'approve':'reject',reason:reason},function(res){ alert((res&&res.message)||'انجام شد'); if(res&&res.success){ location.reload(); } }).fail(function(xhr){ alert('خطای سرور: '+xhr.status); });
  });

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
        $row.find('.sn-finance-status-cell').html(faStatusLabel(res, 'تایید شده'));
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
    var retainedPerSellerCount=String($oldForm.find('input[name="per_seller_count"]').val()||'');
    if(!id || !$role.length){ notice($oldForm,type,msg); return; }
    $.ajax({url:window.location.href,type:'GET',cache:false,timeout:15000}).done(function(html){
      var $html=$($.parseHTML(html,document,true));
      var $newForm=$html.find('#'+id);
      if($newForm.length){
        $role.replaceWith($newForm.closest('.sn-distribution-role'));
        var $fresh=$('#'+id);
        if(retainedPerSellerCount){ $fresh.find('input[name="per_seller_count"]').val(retainedPerSellerCount); }
        sync($fresh); notice($fresh,type,msg);
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
    var fd=new FormData(this); fd.set('action','sn_distribution_transfer_items'); fd.delete('sn_assign_ui_mode'); fd.set('distribution_recipient_scope','selected');
    if(mode==='count'){ fd.delete('distribution_item_ids[]'); fd.set('distribution_bulk_mode','round_robin_active_sellers'); }
    else { fd.set('distribution_bulk_mode','selected'); fd.delete('per_seller_count'); fd.delete('allocation_count'); }
    var $btn=$form.find('.sn-simple-assign-submit').first(), old=$btn.text();
    $btn.prop('disabled',true).text('در حال تخصیص...'); notice($form,'warning','در حال انجام تخصیص...');
    $.ajax({url:ajax,type:'POST',data:fd,processData:false,contentType:false,dataType:'json',timeout:25000}).done(function(res){
      if(res&&res.success){
        var report=(res.report||(res.data&&res.data.report)||{}), msg=res.message||'تخصیص انجام شد.';
        $form.find('input[name="recipient_user_ids[]"], .sn-simple-recipient-select-all, input[name="distribution_item_ids[]"]').prop('checked',false);
        refreshSection($form,msg,'success');
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
  $(function(){ $('.sn-simple-assign-form').each(function(){ sync($(this)); }); });
})(jQuery);


/* SN 1.0.126 legacy sales-manager manual lazy loader removed in 1.0.127. */


/* SN 1.0.127 root progressive panel loader: no manual buttons, no double public.js dependency */
(function($){
  'use strict';
  if (window.snManagerRootProgressiveReady) { return; }
  window.snManagerRootProgressiveReady = true;
  var cfg = window.snAjax || window.snData || {};
  var ajaxUrl = cfg.ajaxurl || window.ajaxurl || '/wp-admin/admin-ajax.php';
  var n = cfg.nonce || '';
  function esc(v){ return String(v == null ? '' : v).replace(/[&<>"']/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#039;'}[c];}); }
  function skeleton(label){ return '<div class="sn-card sn-loading"><strong>'+esc(label||'در حال بارگذاری')+'</strong><div class="sn-skeleton-card"><span class="sn-skeleton-line"></span><span class="sn-skeleton-line"></span><span class="sn-skeleton-line"></span></div></div>'; }

  function switchManagerTab(target){
    var $panel = $('#sn-sales-manager-panel');
    if (!$panel.length || !target) return;
    $panel.find('.sn-tab-button, [data-sn-tab-target]').each(function(){
      var on = String($(this).attr('data-sn-tab-target') || $(this).attr('aria-controls') || '') === String(target);
      $(this).toggleClass('sn-tab-button-active active', on).attr('aria-selected', on ? 'true' : 'false');
    });
    $panel.find('.sn-tab-panel').each(function(){
      var id = String($(this).attr('data-sn-tab-panel') || this.id || '');
      var on = id === String(target);
      $(this).toggleClass('sn-tab-panel-active active', on);
      if (on) $(this).removeAttr('hidden').show(); else $(this).attr('hidden','hidden').hide();
    });
  }

  function loadProgressiveDistribution(page, force){
    var $box = $('#sn-supervisor-distribution-items');
    if (!$box.length) return;
    page = Number(page || 1);
    if (!force && page === 1 && $box.data('loaded')) { syncManualChecks(); return; }
    if ($box.data('loading')) return;
    var caseFilter = $('.sn-tab-panel-active .sn-simple-assign-form .sn-distribution-case-filter, .sn-tab-panel.active .sn-simple-assign-form .sn-distribution-case-filter, .sn-simple-assign-form .sn-distribution-case-filter').first().val() || '';
    $box.data('loading', 1);
    if (page === 1) { $box.html(skeleton('در حال بارگذاری ۵۰ شماره اول')); }
    else { $box.find('.sn-progressive-items-actions').html('<span class="sn-loading">در حال بارگذاری...</span>'); }
    $.ajax({url:ajaxUrl,type:'POST',dataType:'json',timeout:15000,data:{action:'sn_supervisor_distribution_items',nonce:n,page:page,limit:50,case_filter:caseFilter}})
      .done(function(res){
        if(!res || !res.success){ var msg=(res&&res.data&&res.data.message)||(res&&res.message)||'خطا در بارگذاری شماره‌ها'; $box.html('<div class="sn-notice sn-error">'+esc(msg)+'</div>').data('loading',0); return; }
        var data=res.data||{};
        var html=data.html||'';
        if(page>1 && $box.find('table tbody').length){
          var $tmp=$('<div>'+html+'</div>');
          $box.find('table tbody').append($tmp.find('tr'));
        } else {
          $box.html(html);
        }
        $box.find('.sn-progressive-items-actions').remove();
        var $actions=$('<div class="sn-actions sn-progressive-items-actions" style="margin-top:10px"></div>');
        if(data.has_more){ $actions.append('<button type="button" class="sn-btn sn-btn-secondary sn-load-supervisor-items" data-page="'+(page+1)+'">نمایش ۵۰ شماره بعدی</button>'); }
        $actions.append('<button type="button" class="sn-btn sn-btn-ghost sn-reload-supervisor-items">بروزرسانی</button>');
        $box.append($actions).data('page',page).data('loaded',1).data('loading',0);
        syncManualChecks();
      })
      .fail(function(xhr){ if(xhr&&xhr.statusText==='abort') return; $box.html('<div class="sn-notice sn-error">خطای سرور در بارگذاری شماره‌ها: '+esc(xhr&&xhr.status?xhr.status:'نامشخص')+'</div>').data('loading',0); });
  }
  function syncManualChecks(){
    var $form = $('.sn-simple-assign-form').first();
    var manual = $form.find('input[name="sn_assign_ui_mode"]:checked').val() === 'manual';
    $form.find('input[name="distribution_item_ids[]"]').prop('disabled', !manual);
  }
  window.snLoadProgressiveDistributionItems = loadProgressiveDistribution;
  $(document).on('click.snRootProgItems','.sn-load-supervisor-items',function(){ loadProgressiveDistribution($(this).data('page') || 1, false); });
  $(document).on('click.snRootProgItemsReload','.sn-reload-supervisor-items',function(){ $('#sn-supervisor-distribution-items').data('loaded',0); loadProgressiveDistribution(1, true); });
  $(document).on('change.snRootProgItemsMode','.sn-simple-assign-form input[name="sn_assign_ui_mode"]',function(){ if(String($(this).val())==='manual'){ loadProgressiveDistribution(1,false); } setTimeout(syncManualChecks,10); });
  $(document).on('change.snRootProgItemsCase','.sn-simple-assign-form .sn-distribution-case-filter',function(){ var $box=$('#sn-supervisor-distribution-items'); $box.data('loaded',0); loadProgressiveDistribution(1,true); });

  var invoiceCache = {}, invoicePending = null;
  function num(v){ return Number(String(v||0).replace(/[^0-9.-]/g,'')) || 0; }
  function money(v){ try{return num(v).toLocaleString('fa-IR')+' تومان';}catch(e){return String(v||0)+' تومان';} }
  function rawStatus(i){ return String((i&&i.payment_status)||(i&&i.invoice_status)||(i&&i.status)||'').toLowerCase(); }
  function isPaid(i){ return ['paid','approved','financial_approved','gateway_paid','online_paid','completed','success'].indexOf(rawStatus(i))!==-1; }
  function amountOf(i){ return num((i&&i.final_total)||(i&&i.product_price)||(i&&i.paid_amount)||0); }
  function updateInvoiceKpis(rows, summary){
    rows=rows||[];
    var total = summary && summary.total != null ? num(summary.total) : rows.length;
    var paidCount = summary && summary.paid_count != null ? num(summary.paid_count) : rows.filter(isPaid).length;
    var amount = summary && summary.amount != null ? num(summary.amount) : rows.reduce(function(a,i){return a+amountOf(i);},0);
    var paidAmount = summary && summary.paid_amount != null ? num(summary.paid_amount) : rows.reduce(function(a,i){return a+(isPaid(i)?amountOf(i):0);},0);
    $('#sn-supervisor-invoice-kpi-total').text(total.toLocaleString('fa-IR'));
    $('#sn-supervisor-invoice-kpi-paid-count').text(paidCount.toLocaleString('fa-IR'));
    $('#sn-supervisor-invoice-kpi-amount').text(money(amount));
    $('#sn-supervisor-invoice-kpi-paid-amount').text(money(paidAmount));
  }
	function renderInvoices(rows, hasMore, summary){
    rows=rows||[];
    updateInvoiceKpis(rows, summary||{});
    var html='<div class="sn-table-wrap"><table class="sn-table"><thead><tr><th>کد</th><th>فروشنده</th><th>مشتری</th><th>مبلغ</th><th>روش پرداخت</th><th>وضعیت</th><th>اقدام مجدد</th><th>چت بیاوین</th><th>تاریخ</th></tr></thead><tbody>';
    if(!rows.length){ html+='<tr><td colspan="9"><p class="sn-notice">فاکتوری در این تب وجود ندارد.</p></td></tr>'; }
		rows.forEach(function(i){
			var date=i.approved_at_jalali||i.paid_at_jalali||i.created_at_jalali||i.updated_at_jalali||i.date_jalali||'—';
      var projectHtml = i.project && i.project.membership_id ? '<button type="button" class="sn-btn sn-btn-primary sn-btn-mini sn-project-chat-btn" data-membership="'+esc(i.project.membership_id)+'" data-item="'+esc((i.project.items&&i.project.items[0]&&i.project.items[0].id)||'')+'">چت با بیاوین</button>' : '—';
      var invoiceActions=esc(i.repeat_action_label||i.next_action_label||'—')+'<div class="sn-invoice-sms-action"><button type="button" class="sn-btn sn-btn-secondary sn-btn-mini sn-resend-invoice-sms" data-invoice-id="'+esc(i.id||'')+'">ارسال مجدد لینک پرداخت</button><small class="sn-resend-invoice-sms-msg" aria-live="polite"></small><button type="button" class="sn-btn sn-btn-secondary sn-btn-mini sn-copy-invoice-link" data-invoice-id="'+esc(i.id||'')+'">کپی لینک پرداخت</button><small class="sn-copy-invoice-link-msg" aria-live="polite"></small><button type="button" class="sn-btn sn-btn-secondary sn-btn-mini sn-invoice-finance-history" data-invoice-id="'+esc(i.id||'')+'">وضعیت و تاریخچه مالی</button></div>';
      html+='<tr><td><code>'+esc(i.invoice_code||('#'+(i.id||'')))+'</code></td><td>'+esc(i.seller_name||'')+'</td><td>'+esc(i.customer_name||'')+'<br><small>'+esc(i.customer_phone||'')+'</small></td><td>'+esc(i.amount_fmt||money(amountOf(i)))+'</td><td>'+esc(i.pay_method_label||'—')+'<br><small>'+esc(i.payment_source_label||'—')+'</small></td><td><span class="sn-status">'+esc(i.status_label||i.status||'—')+'</span></td><td>'+invoiceActions+'</td><td>'+projectHtml+'</td><td>'+esc(date)+'</td></tr>';
    });
    html+='</tbody></table></div>';
    $('#sn-supervisor-invoices-list').html(html);
    if(hasMore){ $('#sn-supervisor-invoices-list').append('<div class="sn-muted" style="margin-top:8px">۵۰ فاکتور آخر نمایش داده شد؛ برای موارد قدیمی‌تر جستجو/تاریخ را محدود کن.</div>'); }
  }
  function activeInvoiceStatus(){ return String($('#sn-supervisor-invoice-status-filter').val() || $('.sn-supervisor-invoice-tabs .sn-subtab.active').data('status') || 'all'); }
  function loadInvoices(tab, force){
    var $target=$('#sn-supervisor-invoices-list');
    if(!$target.length) return;
    tab=tab||$('.sn-supervisor-invoice-tabs .sn-subtab.active').data('status')||'all';
    var q=$('#sn-supervisor-invoice-search').val()||'';
    var df=$('#sn-supervisor-invoice-date-from').val()||'';
    var dt=$('#sn-supervisor-invoice-date-to').val()||'';
    var sf=activeInvoiceStatus();
    if(sf && sf!=='all') tab=sf;
    var key=[tab,q,df,dt,sf].join('|');
    if(!force && invoiceCache[key]){ renderInvoices(invoiceCache[key].rows, invoiceCache[key].has_more, invoiceCache[key].summary); return; }
    if(invoicePending && invoicePending.readyState!==4){ try{ invoicePending.abort(); }catch(e){} }
    $target.html(skeleton('در حال بارگذاری فاکتورها'));
    invoicePending=$.ajax({url:ajaxUrl,type:'POST',dataType:'json',timeout:15000,data:{action:'sn_supervisor_invoices',nonce:n,tab:tab,q:q,date_from:df,date_to:dt,status_filter:sf,limit:50}})
      .done(function(res){
        var payload=(res&&res.data&&typeof res.data==='object')?res.data:res;
        if(!res || !res.success){ $target.html('<p class="sn-notice sn-error">'+esc((payload&&payload.message)||'خطا در دریافت فاکتورها')+'</p>'); return; }
        var rows=(payload&&payload.items)||[];
        var summary=(payload&&payload.summary)||{};
        invoiceCache[key]={rows:rows,has_more:!!(payload&&payload.has_more),summary:summary};
        renderInvoices(rows, !!(payload&&payload.has_more), summary);
      })
      .fail(function(xhr){ if(xhr&&xhr.statusText==='abort') return; $target.html('<p class="sn-notice sn-error">خطای سرور در دریافت فاکتورها: '+esc((xhr&&xhr.status)||'نامشخص')+'</p>'); });
  }
  window.snLoadRootProgressiveInvoices = loadInvoices;
  $(document).off('click.snRootInv','.sn-supervisor-invoice-tabs .sn-subtab').on('click.snRootInv','.sn-supervisor-invoice-tabs .sn-subtab',function(){ $('.sn-supervisor-invoice-tabs .sn-subtab').removeClass('active'); $(this).addClass('active'); var st=String($(this).data('status')||'all'); $('#sn-supervisor-invoice-status-filter').val(st); loadInvoices(st, false); });
  $(document).off('click.snRootInvFilter','#sn-supervisor-invoice-apply-filter').on('click.snRootInvFilter','#sn-supervisor-invoice-apply-filter',function(e){ e.preventDefault(); var sf=String($('#sn-supervisor-invoice-status-filter').val()||'all'); $('.sn-supervisor-invoice-tabs .sn-subtab').removeClass('active').filter('[data-status="'+sf+'"]').addClass('active'); if(!$('.sn-supervisor-invoice-tabs .sn-subtab.active').length){ $('.sn-supervisor-invoice-tabs .sn-subtab[data-status="all"]').addClass('active'); } loadInvoices(sf, true); });
  $(document).off('keydown.snRootInv input.snRootInv','#sn-supervisor-invoice-search').on('keydown.snRootInv input.snRootInv','#sn-supervisor-invoice-search',function(e){ if(e.type==='keydown' && e.key!=='Enter') return; if(e.type==='keydown') e.preventDefault(); clearTimeout(window.snRootInvoiceSearchTimer); window.snRootInvoiceSearchTimer=setTimeout(function(){ loadInvoices(null,true); }, e.type==='keydown'?10:450); });
  $(document).off('change.snRootInvDates','#sn-supervisor-invoice-date-from,#sn-supervisor-invoice-date-to,#sn-supervisor-invoice-status-filter').on('change.snRootInvDates','#sn-supervisor-invoice-date-from,#sn-supervisor-invoice-date-to,#sn-supervisor-invoice-status-filter',function(){ clearTimeout(window.snRootInvoiceFilterTimer); window.snRootInvoiceFilterTimer=setTimeout(function(){ loadInvoices(null,true); },180); });
  $(document).on('click.snRootUpload','.sn-sup-upload-receipt',function(){ var id=$(this).data('id'), file=$('.sn-sup-receipt-file[data-id="'+id+'"]').prop('files')[0]; if(!file){alert('فایل فیش را انتخاب کنید'); return;} var fd=new FormData(); fd.append('action','sn_supervisor_upload_receipt'); fd.append('nonce',n); fd.append('invoice_id',id); fd.append('receipt',file); $.ajax({url:ajaxUrl,method:'POST',data:fd,processData:false,contentType:false,success:function(res){alert((res&&res.message)||'انجام شد'); loadInvoices(null,true);}}); });

  function loadManagerTab(target){
    var $host = $('#sn-sales-manager-panel .sn-manager-lazy-tab[data-sn-tab-panel="' + target + '"]');
    if (!$host.length || $host.data('loaded') || $host.data('loading')) { return; }
    $host.data('loading', 1).html(skeleton('در حال بارگذاری'));
    $.ajax({url: ajaxUrl, type:'POST', dataType:'json', timeout: 15000, data:{action:'sn_sales_manager_lazy_tab', nonce:n, tab:target}})
      .done(function(res){
        if (!res || !res.success) { var msg=(res&&res.data&&res.data.message)||(res&&res.message)||'خطا در بارگذاری تب'; $host.html('<div class="sn-notice sn-error">'+esc(msg)+'</div>').data('loading',0); return; }
        $host.html((res.data && res.data.html) || res.html || '').data('loaded',1).data('loading',0);
        if (target === 'manager-report') { setTimeout(function(){ $('#sn-manager-filter').trigger('click'); }, 80); }
        if (target === 'manager-distribution') { setTimeout(function(){ loadProgressiveDistribution(1, false); }, 120); }
		if (target === 'manager-invoices') { setTimeout(function(){ loadInvoices('all', false); }, 80); }
		if (target === 'manager-customer-actions') { setTimeout(function(){ var button=$('#manager-customer-actions .sn-staff-customer-actions-filter').get(0); if(button){button.click();} }, 80); }
      })
      .fail(function(xhr){ $host.html('<div class="sn-notice sn-error">خطای سرور در بارگذاری تب: '+esc(xhr&&xhr.status?xhr.status:'نامشخص')+'</div>').data('loading',0); });
  }

  $(document).off('click.snManagerLazyBtn', '#sn-sales-manager-panel .sn-manager-load-tab');
  $(document).off('click.snManagerArchiveTab', '#sn-sales-manager-panel [data-sn-manager-archive-tab]').on('click.snManagerArchiveTab', '#sn-sales-manager-panel [data-sn-manager-archive-tab]', function(e){
    e.preventDefault();
    var tab=String($(this).attr('data-sn-manager-archive-tab')||'no_answer');
    if(['no_answer','assessment_unpaid','subscription_unpaid','product_unpaid','product_star_unpaid'].indexOf(tab)===-1) return;
    var $wrap=$(this).closest('.sn-manager-archives');
    $wrap.find('[data-sn-manager-archive-tab]').removeClass('sn-btn-primary').addClass('sn-btn-secondary').attr('aria-selected','false');
    $(this).removeClass('sn-btn-secondary').addClass('sn-btn-primary').attr('aria-selected','true');
    $wrap.find('[data-sn-manager-archive-panel]').prop('hidden',true).filter('[data-sn-manager-archive-panel="'+tab+'"]').prop('hidden',false);
  });
  $(document).on('click.snManagerRootTabs', '#sn-sales-manager-panel .sn-tab-button, #sn-sales-manager-panel [data-sn-tab-target]', function(e){
    e.preventDefault();
    var target = String($(this).attr('data-sn-tab-target') || $(this).attr('aria-controls') || '');
    if (!target) return;
    switchManagerTab(target);
    setTimeout(function(){ loadManagerTab(target); }, 10);
  });
  $(function(){
    var $panel=$('#sn-sales-manager-panel');
    if(!$panel.length) return;
    var initial = $panel.find('.sn-tab-button-active').first().attr('data-sn-tab-target') || 'manager-overview';
    switchManagerTab(initial);
    setTimeout(function(){ loadManagerTab(initial); }, 20);
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


/* SN 1.0.149 finance invoice details: invoice total + current payment + full payment history */
(function($){
  'use strict';
  if(!$('#sn-financial-panel').length || $('#sn-financial-panel').attr('data-finance-v2')==='1' || window.snFinanceInvoiceDetails149Ready) return;
  window.snFinanceInvoiceDetails149Ready=true;
  var cfg=window.snAjax||window.snData||{};
  var ajax=cfg.ajaxurl||window.ajaxurl||'/wp-admin/admin-ajax.php';
  var nonce=cfg.nonce||cfg.admin_nonce||'';
  function esc(v){return String(v==null?'':v).replace(/[&<>"']/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#039;'}[c];});}
  function ensureModal(){
    if($('#sn-finance-invoice-detail-modal').length) return;
    $('body').append('<div id="sn-finance-invoice-detail-modal" class="sn-modal sn-lite-modal sn-finance-invoice-detail-modal" aria-hidden="true" style="display:none"><div class="sn-modal-backdrop sn-fin-detail-close"></div><div class="sn-modal-card sn-fin-detail-card" style="max-width:980px;width:min(96vw,980px)"><div class="sn-modal-head"><h3>جزئیات فاکتور و ریز واریزی‌ها</h3><button type="button" class="sn-modal-x sn-fin-detail-close">×</button></div><div class="sn-modal-body" id="sn-finance-invoice-detail-body"><div class="sn-loading">در حال بارگذاری...</div></div></div></div>');
  }
  function rowTable(rows, kind){
    if(!rows || !rows.length) return '<div class="sn-notice">موردی ثبت نشده است.</div>';
    var h='<div class="sn-table-wrap"><table class="sn-table"><thead><tr>';
    if(kind==='stages') h+='<th>مرحله</th><th>نوع</th><th>مبلغ مرحله</th><th>روش</th><th>وضعیت</th><th>مرجع</th><th>صادرکننده / تاییدکننده</th><th>تاریخ</th>';
    else h+='<th>مرحله</th><th>مبلغ واریز</th><th>روش</th><th>منبع</th><th>وضعیت</th><th>کد پیگیری</th><th>ثبت‌کننده</th><th>تاریخ</th>';
    h+='</tr></thead><tbody>';
    rows.forEach(function(r){
      if(kind==='stages'){
        h+='<tr><td>'+esc(r.stage_no)+'</td><td>'+esc(r.stage_type_label||'—')+'</td><td><strong>'+esc(r.amount_fmt||'۰ تومان')+'</strong></td><td>'+esc(r.pay_method_label||'—')+'</td><td>'+esc(r.status_label||r.status||'—')+'</td><td>'+esc(r.payment_ref_id||'—')+'</td><td>'+esc(r.issued_by_name||'—')+(r.approved_by_name?'<br><small>تایید: '+esc(r.approved_by_name)+'</small>':'')+'</td><td>'+esc(r.paid_at_jalali||r.created_at_jalali||'—')+'</td></tr>';
      } else {
        h+='<tr><td>'+esc(r.stage_no)+'</td><td><strong>'+esc(r.amount_fmt||'۰ تومان')+'</strong></td><td>'+esc(r.pay_method_label||'—')+'</td><td>'+esc(r.payment_source_label||'—')+'</td><td>'+esc(r.status_label||r.status||'—')+'</td><td>'+esc(r.ref_id||'—')+'</td><td>'+esc(r.uploaded_by_name||'—')+'</td><td>'+esc(r.created_at_jalali||'—')+'</td></tr>';
      }
    });
    return h+'</tbody></table></div>';
  }
  $(document).on('click.snFinanceDetails149','.sn-fin-details',function(e){
    e.preventDefault();
    var id=String($(this).data('id')||''); if(!id) return;
    ensureModal();
    var $m=$('#sn-finance-invoice-detail-modal'), $b=$('#sn-finance-invoice-detail-body');
    $b.html('<div class="sn-loading">در حال بارگذاری جزئیات...</div>');
    $m.fadeIn(120).attr('aria-hidden','false');
    $.post(ajax,{action:'sn_financial_invoice_details',nonce:nonce,invoice_id:id},function(res){
      if(!res || !res.success){$b.html('<div class="sn-notice sn-error">'+esc((res&&res.message)||'خطا در دریافت جزئیات فاکتور')+'</div>');return;}
      var i=res.invoice||{}, stages=res.stages||[], payments=res.payments||[];
      var settled=i.is_fully_settled?'<span class="sn-status sn-status-approved">تسویه کامل</span>':'<span class="sn-status">تسویه نشده</span>';
      var h='<div class="sn-card"><div style="display:flex;justify-content:space-between;gap:12px;flex-wrap:wrap"><div><strong>فاکتور '+esc(i.invoice_code||'')+'</strong><br><small>'+esc(i.customer_name||'')+' — '+esc(i.customer_phone||'')+'</small></div><div>'+settled+'</div></div>'+
        '<div class="sn-kpi-grid sn-card-grid" style="margin-top:12px"><div class="sn-kpi-card"><small>مبلغ کل فاکتور</small><strong>'+esc(i.total_amount_fmt||'۰ تومان')+'</strong></div><div class="sn-kpi-card"><small>جمع واریزی تاییدشده</small><strong>'+esc(i.paid_amount_fmt||'۰ تومان')+'</strong></div><div class="sn-kpi-card"><small>مانده</small><strong>'+esc(i.remaining_amount_fmt||'۰ تومان')+'</strong></div></div>'+
        '<p><strong>فروشنده:</strong> '+esc(i.seller_name||'—')+' &nbsp; <strong>محصول:</strong> '+esc(i.product_name||'—')+' &nbsp; <strong>تاریخ صدور:</strong> '+esc(i.created_at_jalali||'—')+'</p></div>'+
        '<div class="sn-card"><h4>مراحل پرداخت فاکتور</h4>'+rowTable(stages,'stages')+'</div>'+
        '<div class="sn-card"><h4>ریز واریزی‌ها</h4>'+(i.is_fully_settled?'<div class="sn-notice sn-success">این فاکتور به‌طور کامل تسویه شده است. تمام واریزی‌های ثبت‌شده در جدول زیر قابل مشاهده‌اند.</div>':'<div class="sn-notice">فاکتور هنوز کامل تسویه نشده است؛ واریزی‌های ثبت‌شده تا این لحظه نمایش داده می‌شوند.</div>')+rowTable(payments,'payments')+'</div>';
      $b.html(h);
    }).fail(function(xhr){$b.html('<div class="sn-notice sn-error">خطای سرور: '+esc(xhr&&xhr.status?xhr.status:'نامشخص')+'</div>');});
  });
  $(document).on('click.snFinanceDetails149','.sn-fin-detail-close',function(){ $('#sn-finance-invoice-detail-modal').fadeOut(100).attr('aria-hidden','true'); });
})(jQuery);


/* SN 1.0.189 finance workspace v2: search/sort/pagination/receipt review/bulk approval/cancel */
(function($){
  'use strict';
  var $panel=$('#sn-financial-panel[data-finance-v2="1"]');
  if(!$panel.length || window.snFinanceWorkspace189Ready) return;
  window.snFinanceWorkspace189Ready=true;
  var cfg=window.snAjax||window.snData||{};
  var ajax=cfg.ajaxurl||window.ajaxurl||'/wp-admin/admin-ajax.php';
  var nonce=cfg.nonce||cfg.admin_nonce||'';
  var state={tab:'needs_review',page:1,limit:30,q:'',sort:'deposit_newest',rows:{},total:0,totalPages:1};
  var searchTimer=null, financeXhr=null, financeSeq=0;
  function esc(v){return String(v==null?'':v).replace(/[&<>"']/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#039;'}[c];});}
  function fa(v){try{return Number(v||0).toLocaleString('fa-IR');}catch(e){return String(v||0);}}
  function notify(type,msg){
    var $n=$('#sn-finance-v2-notice');
    if(!$n.length){$n=$('<div id="sn-finance-v2-notice"></div>').prependTo('#finance-payments .sn-card.sn-section');}
    $n.attr('class','sn-fin-v2-notice '+(type==='error'?'is-error':'is-success')).text(msg||'').show();
    clearTimeout($n.data('t')); $n.data('t',setTimeout(function(){$n.fadeOut(180);},4500));
  }
  function ensureToolbar(){
    if($('#sn-finance-v2-toolbar').length) return;
    var h='<div id="sn-finance-v2-kpis" class="sn-kpi-grid sn-card-grid sn-fin-v2-kpis"></div><div id="sn-finance-v2-toolbar" class="sn-fin-v2-toolbar">'+
      '<div class="sn-fin-v2-search-wrap"><span>⌕</span><input id="sn-finance-v2-search" type="search" autocomplete="off" placeholder="جستجو: نام، موبایل، کد فاکتور، ۴ رقم کارت، مبلغ، کد پیگیری و..."></div>'+
      '<select id="sn-finance-v2-sort">'+
        '<option value="deposit_newest">جدیدترین واریز</option><option value="deposit_oldest">قدیمی‌ترین واریز</option>'+
        '<option value="newest">جدیدترین تغییر</option><option value="oldest">قدیمی‌ترین تغییر</option>'+
        '<option value="invoice_asc">کد فاکتور ↑</option><option value="invoice_desc">کد فاکتور ↓</option>'+
        '<option value="customer_asc">نام مشتری ↑</option><option value="customer_desc">نام مشتری ↓</option>'+
        '<option value="seller_asc">فروشنده ↑</option><option value="seller_desc">فروشنده ↓</option>'+
        '<option value="status_asc">وضعیت ↑</option><option value="status_desc">وضعیت ↓</option>'+
        '<option value="phone_asc">موبایل ↑</option><option value="phone_desc">موبایل ↓</option>'+
        '<option value="amount_desc">مبلغ کل بیشتر</option><option value="amount_asc">مبلغ کل کمتر</option>'+
        '<option value="payment_desc">مبلغ واریز بیشتر</option><option value="payment_asc">مبلغ واریز کمتر</option>'+
        '<option value="receipt_first">فیش‌دارها اول</option><option value="manual_first">اطلاعات واریزی اول</option>'+
      '</select>'+
      '<select id="sn-finance-v2-limit"><option value="20">۲۰ در صفحه</option><option value="30" selected>۳۰ در صفحه</option><option value="50">۵۰ در صفحه</option><option value="100">۱۰۰ در صفحه</option></select>'+
      '<button type="button" class="sn-btn sn-btn-secondary" id="sn-finance-v2-refresh">بروزرسانی</button>'+
    '</div>'+
    '<div id="sn-finance-v2-bulk" class="sn-fin-v2-bulk" hidden><div><strong id="sn-finance-v2-selected-count">۰</strong> فاکتور انتخاب شده</div><button type="button" class="sn-btn sn-btn-primary" id="sn-finance-v2-bulk-approve">تایید گروهی</button><button type="button" class="sn-btn sn-btn-secondary" id="sn-finance-v2-clear">لغو انتخاب</button></div>';
    $(h).insertBefore('#sn-financial-list');
  }
  function renderKpi(k){
    var html=
      '<div class="sn-kpi-card sn-stat-card"><small>کل در انتظار بررسی</small><strong>'+esc(fa(k.queue_total||k.needs_review||0))+'</strong><em>'+esc(k.needs_amount_fmt||'۰ تومان')+'</em></div>'+
      '<div class="sn-kpi-card sn-stat-card"><small>نتیجه فیلتر فعلی</small><strong>'+esc(fa(k.filtered_count||k.count||0))+'</strong><em>تعداد کل، نه فقط این صفحه</em></div>'+
      '<div class="sn-kpi-card sn-stat-card"><small>فیش‌های آپلودشده</small><strong>'+esc(fa(k.receipt_uploaded||0))+'</strong><em>فیش یا رسید</em></div>'+
      '<div class="sn-kpi-card sn-stat-card"><small>تایید شده مالی</small><strong>'+esc(fa(k.approved||0))+'</strong><em>'+esc(k.approved_amount_fmt||'۰ تومان')+'</em></div>'+
      '<div class="sn-kpi-card sn-stat-card"><small>رد شده مالی</small><strong>'+esc(fa(k.rejected||0))+'</strong><em>قابل پیگیری فروشنده</em></div>'+
      '<div class="sn-kpi-card sn-stat-card"><small>برگشتی / لغو مالی</small><strong>'+esc(fa(k.returned_to_seller||0))+'</strong><em>ثبت‌شده در لاگ مالی</em></div>';
    $('#sn-finance-v2-kpis').html(html);
    $('#sn-financial-kpis').html(html);
  }
  function statusClass(i){
    var s=String(i.status||i.payment_status||i.invoice_status||'').toLowerCase();
    if(/approved|paid/.test(s)) return 'is-approved'; if(/reject|cancel/.test(s)) return 'is-rejected'; return 'is-pending';
  }
  function financeTag(i){
    return i&&i.is_dot_marketing?'<span class="sn-fin-v2-source-tag is-dot-marketing">'+esc(i.finance_tag||'مارکتینگ دات فلو')+'</span>':'';
  }
  function rowActions(i){
    var h='<div class="sn-fin-v2-actions">';
    if(i.has_receipt) h+='<button type="button" class="sn-btn sn-btn-sm sn-btn-secondary sn-fin-v2-receipt" data-sn-fin-act="receipt" data-id="'+esc(i.id)+'">مشاهده فیش</button>';
    else h+='<button type="button" class="sn-btn sn-btn-sm sn-btn-secondary sn-fin-v2-info" data-sn-fin-act="receipt" data-id="'+esc(i.id)+'">اطلاعات واریز</button>';
    if(i.can_approve_payment) h+='<button type="button" class="sn-btn sn-btn-sm sn-btn-primary sn-fin-v2-approve" data-sn-fin-act="approve" data-id="'+esc(i.id)+'">تایید</button>';
    if(i.can_reject_payment) h+='<button type="button" class="sn-btn sn-btn-sm sn-btn-danger sn-fin-v2-reject" data-sn-fin-act="reject" data-id="'+esc(i.id)+'">رد</button>';
    h+='<button type="button" class="sn-btn sn-btn-sm sn-btn-secondary sn-fin-v2-history" data-sn-fin-act="history" data-id="'+esc(i.id)+'">دلیل / پاسخ</button>';
    if(i.can_reopen_financial) h+='<button type="button" class="sn-btn sn-btn-sm sn-btn-secondary sn-fin-v2-reopen" data-sn-fin-act="reopen" data-id="'+esc(i.id)+'">بازگشت به بررسی</button>';
    if(i.can_cancel_invoice) h+='<button type="button" class="sn-btn sn-btn-sm sn-fin-v2-cancel" data-sn-fin-act="cancel" data-id="'+esc(i.id)+'">لغو / عودت</button>';
    return h+'</div>';
  }
  function renderRows(rows){
    state.rows={}; rows.forEach(function(i){state.rows[String(i.id)]=i;});
    if(!rows.length){$('#sn-financial-list').html('<div class="sn-fin-v2-empty">موردی با این فیلتر پیدا نشد.</div>'); renderPager(); updateBulk(); return;}
    var h='<div class="sn-fin-v2-summary">نمایش <strong>'+esc(fa(rows.length))+'</strong> مورد از <strong>'+esc(fa(state.total))+'</strong> نتیجه</div>'+
      '<div class="sn-table-wrap sn-fin-v2-table-wrap"><table class="sn-table sn-fin-v2-table"><thead><tr><th class="sn-fin-v2-check"><input type="checkbox" id="sn-fin-v2-check-all"></th><th>فاکتور</th><th>مشتری</th><th>فروشنده</th><th>واریز فعلی</th><th>مبلغ کل / مانده</th><th>تاریخ واریز</th><th>روش / منبع</th><th>فیش / اطلاعات</th><th>وضعیت</th><th>عملیات</th></tr></thead><tbody>';
    rows.forEach(function(i){
      var check=i.can_approve_payment?'<input type="checkbox" class="sn-fin-v2-select" value="'+esc(i.id)+'">':'<span class="sn-fin-v2-no-check">—</span>';
      var paymentInfo=i.has_receipt?'<button type="button" class="sn-fin-v2-receipt-link sn-fin-v2-receipt" data-sn-fin-act="receipt" data-id="'+esc(i.id)+'">📎 فیش بارگذاری‌شده</button>'+(i.manual_card_to_display?'<small class="sn-fin-v2-card-destination">۴ رقم مقصد: <span dir="ltr">'+esc(i.manual_card_to_display)+'</span></small>':''):('<span class="sn-fin-v2-manual">'+esc(i.payment_info_text||'—')+'</span>');
      h+='<tr data-id="'+esc(i.id)+'"><td class="sn-fin-v2-check">'+check+'</td>'+
        '<td><code>'+esc(i.invoice_code||'')+'</code>'+financeTag(i)+'<small>#'+esc(i.id)+'</small></td>'+
        '<td><strong>'+esc(i.customer_name||'—')+'</strong><small dir="ltr">'+esc(i.customer_phone||'—')+'</small></td>'+
        '<td>'+esc(i.seller_name||i.seller_id||'—')+'</td>'+
        '<td><strong>'+esc(i.current_payment_amount_fmt||'۰ تومان')+'</strong><small>مرحله '+esc(i.latest_payment_stage_no||1)+'</small></td>'+
        '<td><strong>'+esc(i.invoice_total_amount_fmt||i.amount_fmt||'۰ تومان')+'</strong><small>مانده: '+esc(i.remaining_amount_fmt||'۰ تومان')+'</small></td>'+
        '<td>'+esc(i.deposit_date_label||'—')+'</td>'+
        '<td>'+esc(i.pay_method_label||'—')+'<small>'+esc(i.payment_source_label||'')+'</small></td>'+
        '<td>'+paymentInfo+'</td>'+
        '<td><span class="sn-fin-v2-status '+statusClass(i)+'">'+esc(i.status_label||i.status||'—')+'</span>'+(i.financial_return_label?'<small>'+esc(i.financial_return_label)+'</small>':'')+'</td>'+
        '<td>'+rowActions(i)+'</td></tr>';
    });
    h+='</tbody></table></div><div id="sn-fin-v2-pager"></div>';
    $('#sn-financial-list').html(h); renderPager(); updateBulk();
  }
  function renderPager(){
    var $p=$('#sn-fin-v2-pager'); if(!$p.length) return;
    var total=state.totalPages||1, cur=state.page||1, h='<div class="sn-fin-v2-pager"><div>صفحه <strong>'+esc(fa(cur))+'</strong> از <strong>'+esc(fa(total))+'</strong> — کل: <strong>'+esc(fa(state.total))+'</strong></div><div class="sn-fin-v2-pages">';
    h+='<button type="button" data-page="'+Math.max(1,cur-1)+'" '+(cur<=1?'disabled':'')+'>قبلی</button>';
    var from=Math.max(1,cur-2), to=Math.min(total,cur+2); if(from>1) h+='<button type="button" data-page="1">۱</button><span>…</span>';
    for(var n=from;n<=to;n++) h+='<button type="button" data-page="'+n+'" class="'+(n===cur?'active':'')+'">'+fa(n)+'</button>';
    if(to<total) h+='<span>…</span><button type="button" data-page="'+total+'">'+fa(total)+'</button>';
    h+='<button type="button" data-page="'+Math.min(total,cur+1)+'" '+(cur>=total?'disabled':'')+'>بعدی</button></div></div>';
    $p.html(h);
  }
  function loadFinance(page){
    state.page=page||state.page||1;
    var seq=++financeSeq;
    if(financeXhr && financeXhr.readyState!==4){try{financeXhr.abort();}catch(e){}}
    $('#sn-financial-list').html('<div class="sn-loading">در حال بارگذاری...</div>');
    financeXhr=$.post(ajax,{action:'sn_financial_invoices',nonce:nonce,tab:state.tab,page:state.page,limit:state.limit,q:state.q,sort:state.sort},function(res){
      if(seq!==financeSeq)return;
      if(!res||!res.success){$('#sn-financial-list').html('<div class="sn-notice sn-error">'+esc((res&&res.message)||'خطا در دریافت اطلاعات مالی')+'</div>');return;}
      state.page=parseInt(res.page||1,10)||1; state.total=parseInt(res.total||0,10)||0; state.totalPages=parseInt(res.total_pages||1,10)||1;
      renderKpi(res.kpi||{}); renderRows(res.items||[]);
    }).fail(function(xhr,status){if(seq!==financeSeq||status==='abort')return;$('#sn-financial-list').html('<div class="sn-notice sn-error">خطای سرور مالی: '+esc(xhr&&xhr.status?xhr.status:'نامشخص')+'</div>');}).always(function(){if(seq===financeSeq)financeXhr=null;});
  }
  window.snFinanceV2Load=loadFinance;
  function selectedIds(){return $('.sn-fin-v2-select:checked').map(function(){return parseInt(this.value,10)||0;}).get().filter(Boolean);}
  function updateBulk(){
    var ids=selectedIds(), $b=$('#sn-finance-v2-bulk'); $('#sn-finance-v2-selected-count').text(fa(ids.length));
    if(ids.length)$b.removeAttr('hidden');else $b.attr('hidden','hidden');
    var all=$('.sn-fin-v2-select').length, checked=$('.sn-fin-v2-select:checked').length; $('#sn-fin-v2-check-all').prop('checked',all>0&&checked===all).prop('indeterminate',checked>0&&checked<all);
  }
  function approveOne(id, cb){
    $.post(ajax,{action:'sn_financial_approve_payment',nonce:nonce,invoice_id:id},function(res){if(res&&res.success){notify('success',res.message||'تایید شد'); if(cb)cb(true,res); else loadFinance(state.page);}else{notify('error',(res&&res.message)||'تایید انجام نشد'); if(cb)cb(false,res);}}).fail(function(xhr){notify('error','خطای سرور: '+(xhr.status||''));if(cb)cb(false,null);});
  }
  function rejectOne(id, inlineReason){
    var reason=inlineReason===undefined?window.prompt('دلیل رد پرداخت را وارد کنید:',''):inlineReason; if(reason===null)return; reason=$.trim(reason); if(!reason){notify('error','دلیل رد الزامی است');return;}
    $.post(ajax,{action:'sn_financial_reject_payment',nonce:nonce,invoice_id:id,reason:reason},function(res){if(res&&res.success){notify('success',res.message||'پرداخت رد شد');closeReceipt();loadFinance(state.page);}else notify('error',(res&&res.message)||'ثبت رد انجام نشد');}).fail(function(xhr){notify('error','خطای سرور: '+(xhr.status||''));});
  }
  function reopenOne(id){
    var reason=window.prompt('دلیل بازگشت این فاکتور به «در انتظار بررسی» را وارد کنید:','');
    if(reason===null)return; reason=$.trim(reason); if(!reason){notify('error','دلیل بازگشت الزامی است');return;}
    if(!window.confirm('تایید مالی این فاکتور برداشته شود و دوباره به «در انتظار بررسی» برگردد؟'))return;
    $.post(ajax,{action:'sn_financial_reopen_invoice',nonce:nonce,invoice_id:id,reason:reason},function(res){
      if(res&&res.success){notify('success',res.message||'فاکتور به بررسی برگشت');closeReceipt();loadFinance(state.page);}
      else notify('error',(res&&res.message)||'بازگشت به بررسی انجام نشد');
    }).fail(function(xhr){notify('error','خطای سرور: '+(xhr.status||''));});
  }
  function ensureReceiptModal(){
    if($('#sn-fin-v2-receipt-modal').length)return;
    $('body').append('<div id="sn-fin-v2-receipt-modal" class="sn-fin-v2-modal" aria-hidden="true" style="display:none"><div class="sn-fin-v2-backdrop sn-fin-v2-receipt-close"></div><div class="sn-fin-v2-modal-card sn-fin-v2-receipt-card"><div class="sn-fin-v2-modal-head"><div><h3>بررسی فیش و واریزی</h3><p>اطلاعات مشتری و عملیات مالی در کنار فیش</p></div><button type="button" class="sn-fin-v2-modal-x sn-fin-v2-receipt-close">×</button></div><div id="sn-fin-v2-receipt-body"></div></div></div>');
  }
  function closeReceipt(){$('#sn-fin-v2-receipt-modal').fadeOut(100).attr('aria-hidden','true');}
  function receiptMedia(url){
    if(!url)return '<div class="sn-fin-v2-no-receipt">برای این واریز تصویر فیش ثبت نشده است.</div>';
    var clean=String(url).split('?')[0].toLowerCase();
    if(/\.pdf$/.test(clean))return '<iframe class="sn-fin-v2-receipt-frame" src="'+esc(url)+'" title="فیش پرداخت"></iframe>';
    return '<a href="'+esc(url)+'" target="_blank" rel="noopener"><img class="sn-fin-v2-receipt-img" src="'+esc(url)+'" alt="فیش پرداخت"></a>';
  }
  function openReceipt(id){
    var i=state.rows[String(id)]; if(!i)return; ensureReceiptModal();
    var meta='<div class="sn-fin-v2-receipt-meta">'+
      '<div><small>کد فاکتور</small><strong>'+esc(i.invoice_code||'—')+'</strong>'+financeTag(i)+'</div><div><small>نام مشتری</small><strong>'+esc(i.customer_name||'—')+'</strong></div><div><small>شماره مشتری</small><strong dir="ltr">'+esc(i.customer_phone||'—')+'</strong></div>'+
			'<div><small>مبلغ واریز فعلی</small><strong>'+esc(i.current_payment_amount_fmt||'۰ تومان')+'</strong></div><div><small>تاریخ واریز</small><strong>'+esc(i.deposit_date_label||'—')+'</strong></div><div><small>۴ رقم کارت مبدا</small><strong dir="ltr">'+esc(i.manual_card_from_display||'—')+'</strong></div><div><small>۴ رقم آخر کارت مقصد</small><strong dir="ltr">'+esc(i.manual_card_to_display||'—')+'</strong></div></div>';
		meta='<div class="sn-fin-v2-current-status"><strong>وضعیت فعلی: '+esc(i.status_label||i.status||'—')+'</strong><span>مبلغ کل فاکتور: '+esc(i.invoice_total_amount_fmt||'۰ تومان')+'</span></div>'+meta;
    var buttons='<div class="sn-fin-v2-receipt-actions">';
    if(i.can_approve_payment)buttons+='<button type="button" class="sn-btn sn-btn-primary sn-fin-v2-modal-approve" data-id="'+esc(i.id)+'">تایید این واریز</button>';
    if(i.can_reject_payment)buttons+='<label class="sn-fin-v2-reject-inline">دلیل رد مالی<textarea class="sn-fin-v2-reject-reason" rows="2" placeholder="علت رد را دقیق ثبت کنید"></textarea></label><button type="button" class="sn-btn sn-btn-danger sn-fin-v2-modal-reject" data-id="'+esc(i.id)+'">رد این واریز</button>';
    if(i.can_reopen_financial)buttons+='<button type="button" class="sn-btn sn-btn-secondary sn-fin-v2-modal-reopen" data-id="'+esc(i.id)+'">بازگشت به بررسی</button>';
    buttons+='<button type="button" class="sn-btn sn-btn-secondary sn-fin-v2-receipt-close">بستن</button></div>';
    $('#sn-fin-v2-receipt-body').html('<div class="sn-fin-v2-receipt-layout"><div class="sn-fin-v2-receipt-side">'+meta+'<div class="sn-fin-v2-receipt-extra"><strong>فروشنده:</strong> '+esc(i.seller_name||'—')+'<br><strong>روش:</strong> '+esc(i.pay_method_label||'—')+' / '+esc(i.payment_source_label||'—')+'<br><strong>مانده:</strong> '+esc(i.remaining_amount_fmt||'۰ تومان')+'</div><div class="sn-fin-v2-inline-history">در حال دریافت سوابق بررسی...</div>'+buttons+'</div><div class="sn-fin-v2-media">'+receiptMedia(i.receipt_url||'')+'</div></div>');
    $('#sn-fin-v2-receipt-modal').fadeIn(100).attr('aria-hidden','false');
    $.post(ajax,{action:'sn_financial_invoice_details',nonce:nonce,invoice_id:id},function(res){
      if(!res||!res.success){$('#sn-fin-v2-receipt-modal .sn-fin-v2-inline-history').text('سوابق بررسی در دسترس نیست.');return;}
      var acts=(res.activities||[]).slice(0,8),html='<strong>سوابق بررسی</strong>';
      html+=acts.length?acts.map(function(a){return '<article><strong>'+esc(a.label||a.action)+'</strong><small>'+esc(a.actor_name||'')+' · '+esc(a.created_at_jalali||'')+'</small><p>'+esc(a.note||a.description||'')+'</p></article>';}).join(''):'<p>هنوز سابقه‌ای ثبت نشده است.</p>';
      $('#sn-fin-v2-receipt-modal .sn-fin-v2-inline-history').html(html);
    }).fail(function(){$('#sn-fin-v2-receipt-modal .sn-fin-v2-inline-history').text('سوابق بررسی در دسترس نیست.');});
  }
  function ensureHistoryModal(){
    if($('#sn-fin-v2-history-modal').length)return;
    $('body').append('<div id="sn-fin-v2-history-modal" class="sn-fin-v2-modal" aria-hidden="true" style="display:none"><div class="sn-fin-v2-backdrop sn-fin-v2-history-close"></div><div class="sn-fin-v2-modal-card sn-fin-v2-history-card"><div class="sn-fin-v2-modal-head"><div><h3>دلیل رد و پاسخ فروشنده</h3><p>سوابق مالی و ارتباط مجدد این فاکتور</p></div><button type="button" class="sn-fin-v2-modal-x sn-fin-v2-history-close">×</button></div><div id="sn-fin-v2-history-body" class="sn-fin-v2-history-body"><div class="sn-loading">در حال دریافت...</div></div></div></div>');
  }
  function closeHistory(){$('#sn-fin-v2-history-modal').fadeOut(100).attr('aria-hidden','true');}
  function openHistory(id){
    var i=state.rows[String(id)]||{}; ensureHistoryModal(); var quick='';
    if(i.financial_reject_reason)quick+='<div class="sn-fin-v2-history-quick is-reject"><small>آخرین دلیل رد مالی</small><strong>'+esc(i.financial_reject_reason)+'</strong></div>';
    if(i.seller_response_note)quick+='<div class="sn-fin-v2-history-quick is-response"><small>آخرین پاسخ فروشنده</small><strong>'+esc(i.seller_response_note)+'</strong></div>';
    if(i.recontact_note_display)quick+='<div class="sn-fin-v2-history-quick"><small>یادداشت ارتباط مجدد</small><strong>'+esc(i.recontact_note_display)+'</strong></div>';
    $('#sn-fin-v2-history-body').html(quick+'<div class="sn-loading">در حال دریافت تاریخچه...</div>'); $('#sn-fin-v2-history-modal').fadeIn(100).attr('aria-hidden','false');
    $.post(ajax,{action:'sn_financial_invoice_details',nonce:nonce,invoice_id:id},function(res){ if(!(res&&res.success)){ $('#sn-fin-v2-history-body').append('<div class="sn-notice sn-error">'+esc((res&&res.message)||'تاریخچه دریافت نشد')+'</div>'); return; } var acts=res.activities||[],html=quick; if(!acts.length)html+='<div class="sn-fin-v2-history-empty">برای این فاکتور سابقه رد یا پاسخ ثبت نشده است.</div>'; else{html+='<div class="sn-fin-v2-history-list">';acts.forEach(function(a){html+='<article class="sn-fin-v2-history-item"><div><strong>'+esc(a.label||a.action||'رویداد')+'</strong><small>'+esc(a.actor_name||'')+' · '+esc(a.created_at_jalali||'')+'</small></div><p>'+esc(a.note||a.description||'—')+'</p></article>';});html+='</div>';} $('#sn-fin-v2-history-body').html(html); }).fail(function(xhr){$('#sn-fin-v2-history-body').append('<div class="sn-notice sn-error">خطای سرور: '+esc(xhr&&xhr.status?xhr.status:'')+'</div>');});
  }
  function ensureCancelModal(){
    if($('#sn-fin-v2-cancel-modal').length)return;
    $('body').append('<div id="sn-fin-v2-cancel-modal" class="sn-fin-v2-modal" aria-hidden="true" style="display:none"><div class="sn-fin-v2-backdrop sn-fin-v2-cancel-close"></div><div class="sn-fin-v2-modal-card sn-fin-v2-cancel-card"><div class="sn-fin-v2-modal-head"><div><h3>لغو / ثبت عودت وجه</h3><p>این عملیات دسترسی فروشنده را شبیه‌سازی نمی‌کند و مستقیم با لاگ مالی ثبت می‌شود.</p></div><button type="button" class="sn-fin-v2-modal-x sn-fin-v2-cancel-close">×</button></div><div class="sn-fin-v2-cancel-body"><div id="sn-fin-v2-cancel-summary"></div><label>دلیل لغو / عودت<textarea id="sn-fin-v2-cancel-reason" rows="4" placeholder="علت را دقیق ثبت کنید"></textarea></label><label class="sn-fin-v2-refund-check" id="sn-fin-v2-refund-wrap"><input type="checkbox" id="sn-fin-v2-refund-confirm"> <span>تایید می‌کنم عودت وجه دریافت‌شده خارج از این افزونه انجام شده یا مسئولیت پیگیری آن مشخص است.</span></label><div class="sn-fin-v2-warning">این اکشن پول بانکی، کیف پول، پورسانت یا سفارش ووکامرس را خودکار Reverse نمی‌کند؛ فقط فاکتور را برای ادامه عملیات می‌بندد و لاگ مالی ثبت می‌کند.</div><div class="sn-fin-v2-receipt-actions"><button type="button" class="sn-btn sn-btn-danger" id="sn-fin-v2-cancel-submit">ثبت لغو مالی</button><button type="button" class="sn-btn sn-btn-secondary sn-fin-v2-cancel-close">انصراف</button></div></div></div></div>');
  }
  function openCancel(id){
    var i=state.rows[String(id)]; if(!i)return; ensureCancelModal(); $('#sn-fin-v2-cancel-modal').data('id',id);
    $('#sn-fin-v2-cancel-reason').val(''); $('#sn-fin-v2-refund-confirm').prop('checked',false); $('#sn-fin-v2-refund-wrap').toggle(!!i.finance_cancel_requires_refund_confirm);
    $('#sn-fin-v2-cancel-summary').html('<div class="sn-fin-v2-cancel-summary"><strong>'+esc(i.invoice_code||'')+' — '+esc(i.customer_name||'')+'</strong><span>واریزی تاییدشده: '+esc(i.approved_paid_amount_fmt||'۰ تومان')+' | مانده: '+esc(i.remaining_amount_fmt||'۰ تومان')+'</span></div>');
    $('#sn-fin-v2-cancel-modal').fadeIn(100).attr('aria-hidden','false');
  }
  function closeCancel(){$('#sn-fin-v2-cancel-modal').fadeOut(100).attr('aria-hidden','true');}

  function syncGatewayExport(){
    var show=state.tab==='online_paid';
    $('#sn-financial-gateway-export').prop('hidden',!show).attr('aria-hidden',show?'false':'true');
  }
  ensureToolbar();
  syncGatewayExport();
  $(document).off('click','.sn-financial-tabs .sn-subtab').on('click.snFinV2','.sn-financial-tabs .sn-subtab',function(){ $('.sn-financial-tabs .sn-subtab').removeClass('active'); $(this).addClass('active'); state.tab=String($(this).data('tab')||'needs_review'); state.page=1; syncGatewayExport(); loadFinance(1); });
  $(document).on('input.snFinV2','#sn-finance-v2-search',function(){var v=this.value||'';clearTimeout(searchTimer);searchTimer=setTimeout(function(){state.q=$.trim(v);state.page=1;loadFinance(1);},280);});
  $(document).on('change.snFinV2','#sn-finance-v2-sort',function(){state.sort=this.value||'deposit_newest';state.page=1;loadFinance(1);});
  $(document).on('change.snFinV2','#sn-finance-v2-limit',function(){state.limit=parseInt(this.value,10)||30;state.page=1;loadFinance(1);});
  $(document).on('click.snFinV2','#sn-finance-v2-refresh',function(){loadFinance(state.page);});
  $(document).on('click.snFinV2','#sn-fin-v2-pager button[data-page]',function(){var p=parseInt($(this).data('page'),10)||1;if(!this.disabled)loadFinance(p);});
  $(document).on('change.snFinV2','#sn-fin-v2-check-all',function(){$('.sn-fin-v2-select').prop('checked',this.checked);updateBulk();});
  $(document).on('change.snFinV2','.sn-fin-v2-select',updateBulk);
  $(document).on('click.snFinV2','#sn-finance-v2-clear',function(){$('.sn-fin-v2-select,#sn-fin-v2-check-all').prop('checked',false);updateBulk();});
  $(document).on('click.snFinV2','#sn-finance-v2-bulk-approve',function(){var ids=selectedIds();if(!ids.length)return;if(!window.confirm('تایید گروهی '+fa(ids.length)+' فاکتور انجام شود؟'))return;var $b=$(this).prop('disabled',true).text('در حال تایید...');$.post(ajax,{action:'sn_financial_bulk_approve',nonce:nonce,invoice_ids:ids},function(res){$b.prop('disabled',false).text('تایید گروهی');if(res&&res.success){notify('success',res.message||'تایید گروهی انجام شد');loadFinance(state.page);}else notify('error',(res&&res.message)||'عملیات گروهی انجام نشد');}).fail(function(xhr){$b.prop('disabled',false).text('تایید گروهی');notify('error','خطای سرور: '+(xhr.status||''));});});
  $('#sn-financial-panel').off('click.snFinV2Actions','[data-sn-fin-act]').on('click.snFinV2Actions','[data-sn-fin-act]',function(e){e.preventDefault();e.stopPropagation();var $b=$(this),id=parseInt($b.attr('data-id'),10)||0,act=String($b.attr('data-sn-fin-act')||'');if(!id||$b.prop('disabled'))return;if(act==='approve'){approveOne(id);return;}if(act==='reject'){rejectOne(id);return;}if(act==='reopen'){reopenOne(id);return;}if(act==='receipt'){openReceipt(id);return;}if(act==='history'){openHistory(id);return;}if(act==='cancel'){openCancel(id);return;}});
  $(document).on('click.snFinV2','.sn-fin-v2-receipt-close',closeReceipt);
  $(document).on('click.snFinV2','.sn-fin-v2-modal-approve',function(){var id=$(this).data('id'),$b=$(this).prop('disabled',true).text('در حال تایید...');approveOne(id,function(ok){$b.prop('disabled',false).text('تایید این واریز');if(ok){closeReceipt();loadFinance(state.page);}});});
  $(document).on('click.snFinV2','.sn-fin-v2-modal-reject',function(){rejectOne($(this).data('id'),$('#sn-fin-v2-receipt-modal .sn-fin-v2-reject-reason').val()||'');});
  $(document).on('click.snFinV2','.sn-fin-v2-modal-reopen',function(){reopenOne($(this).data('id'));});
  $(document).on('click.snFinV2','.sn-fin-v2-cancel-close',closeCancel);
  $(document).on('click.snFinV2','.sn-fin-v2-history-close',closeHistory);
  $(document).on('click.snFinV2','#sn-fin-v2-cancel-submit',function(){var id=$('#sn-fin-v2-cancel-modal').data('id'),i=state.rows[String(id)]||{},reason=$.trim($('#sn-fin-v2-cancel-reason').val()||'');if(!reason){notify('error','دلیل لغو الزامی است');return;}if(i.finance_cancel_requires_refund_confirm&&!$('#sn-fin-v2-refund-confirm').is(':checked')){notify('error','برای فاکتور دارای واریزی تاییدشده، تایید عودت وجه الزامی است');return;}var $b=$(this).prop('disabled',true).text('در حال ثبت...');$.post(ajax,{action:'sn_financial_cancel_invoice',nonce:nonce,invoice_id:id,reason:reason,refund_confirmed:$('#sn-fin-v2-refund-confirm').is(':checked')?1:0},function(res){$b.prop('disabled',false).text('ثبت لغو مالی');if(res&&res.success){notify('success',res.message||'لغو ثبت شد');closeCancel();loadFinance(state.page);}else notify('error',(res&&res.message)||'لغو انجام نشد');}).fail(function(xhr){$b.prop('disabled',false).text('ثبت لغو مالی');notify('error','خطای سرور: '+(xhr.status||''));});});
  $(document).on('keydown.snFinV2',function(e){if(e.key==='Escape'){closeReceipt();closeCancel();closeHistory();}});
  loadFinance(1);
})(jQuery);

// CRM-45: one authoritative review handler; update the reviewed request in place.
(function($){
  'use strict';
  var cfg=window.snAjax||window.snData||{},ajax=cfg.ajaxurl||window.ajaxurl||'/wp-admin/admin-ajax.php',nonce=cfg.nonce||'';
  var selector='.sn-extra-number-approve,.sn-extra-number-reject';
  $(document).off('click',selector).on('click.snCrm45',selector,function(e){
    e.preventDefault();
    var $button=$(this),$row=$button.closest('tr[data-request-id]');
    var approve=$button.hasClass('sn-extra-number-approve');
    var reason='';
    if(!approve){reason=prompt('دلیل رد را بنویسید:','');if(reason===null)return;}
    $row.find(selector).prop('disabled',true);
    $.post(ajax,{action:'sn_manager_review_extra_number',nonce:nonce,request_id:$button.data('id'),decision:approve?'approve':'reject',reason:reason},function(res){
      if(!res||!res.success){alert((res&&res.message)||'ثبت تغییر انجام نشد');return;}
      var status=String(res.status|| (approve?'approved':'rejected'));
      $row.find('td').eq(4).find('.sn-badge').text(status==='approved'?'تایید شده':'رد شده');
      $row.find('td').last().empty().append($('<span class="sn-notice"></span>').addClass(status==='approved'?'sn-success':'sn-warning').text(status==='approved'?'تایید شد':'رد شد'));
      if(status==='rejected'&&res.reason){$row.find('td').last().append($('<div class="sn-muted"></div>').text('دلیل: '+res.reason));}
    }).fail(function(xhr){alert('خطای سرور: '+xhr.status);}).always(function(){$row.find(selector).prop('disabled',false);});
  });
})(jQuery);
