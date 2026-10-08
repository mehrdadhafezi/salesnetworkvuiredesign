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
  // Classic tabs are handled globally in public.js when present.
  // Keep a tiny fallback for pages that enqueue only public-seller.js.
  $(document).on('click', '.sn-tab', function (e) {
    if (window.snClassicTabsReady) { return; }
    var $panel = $(this).closest('.sn-panel, .sn-invoice-page');
    var target = String($(this).data('tab') || '').replace(/[^A-Za-z0-9_-]/g, '');
    if (!$panel.length || !target || !$panel.find('#sn-tab-' + target).length) { return; }
    $panel.find('.sn-tab').removeClass('active').attr('aria-selected', 'false');
    $panel.find('.sn-tab-content').removeClass('active').hide();
    $(this).addClass('active').attr('aria-selected', 'true');
    $panel.find('#sn-tab-' + target).addClass('active').show();
    try { window.localStorage.setItem('sn_active_tab_' + String($panel.attr('id') || 'seller'), target); } catch(err) {}
    e.preventDefault();
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
  function esc(v){ return String(v==null?'':v).replace(/[&<>\"']/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','\"':'&quot;',"'":'&#039;'}[c] || c;}); }
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

  const $sellerPanel = $('#sn-seller-panel');
  if ($sellerPanel.length) {
    var sellerCanIssueInvoice = String($sellerPanel.attr('data-sn-can-issue-invoice') || '0') === '1';
    var sellerManualInvoiceEnabled = String($sellerPanel.attr('data-sn-manual-invoice-enabled') || '1') === '1';
    snEnsureDarkToggle($sellerPanel);
    let allLeadStatuses = [
      {slug:'no_answer', label:'جواب نداده', color:'#f59e0b'},
      {slug:'callback', label:'تماس مجدد', color:'#3b82f6'},
      {slug:'duplicate', label:'تکراری', color:'#64748b'},
      {slug:'not_purchased', label:'عدم خرید', color:'#ef4444'},
      {slug:'pre_invoice', label:'پیش‌فاکتور', color:'#0ea5e9'}
    ];
    let allLeads        = [];
    let sellerLeadRequestSeq = 0;
    let activeFilter    = 'no-status';
    let expandedLeadId  = null;
    let saveTimers      = {};
    let pendingSaves    = {};
    let activeSavePayload = {};
    let dirtySaveQueued = {};
    let lastSavedPayloadHash = {};
    let sellerInvoices  = [];
    let sellerInvoiceSummary = {};
    let sellerPaidAmount = 0;
    let sellerLeadPage = 1;
    let sellerLeadPerPage = 50;
    let sellerLeadSearch = '';
    let sellerLeadMeta = { page: 1, per_page: 50, total: 0, total_all: 0, total_pages: 1 };
    let sellerLeadSearchTimer = null;

	// ---- بارگذاری چهار وضعیت استاندارد مرحله اول از سرور ----
    $.post(ajax, { action: 'sn_get_lead_statuses', nonce: nonce }).done(function (res) {
      if (res.success && res.statuses && res.statuses.length) {
        allLeadStatuses = res.statuses;
      }
	}).always(function () {
	  // حتی اگر endpoint وضعیت‌ها موقتاً خطا داد، فهرست شماره‌ها با قرارداد
	  // ثابت چهارحالته بالا قابل استفاده می‌ماند و روی spinner متوقف نمی‌شود.
      renderLeadFilterBar();
      loadLeads();
    });

    // بارگذاری خلاصه فاکتورها کمی بعد از نمایش اولیه انجام شود تا پنل سریع‌تر باز شود.
    setTimeout(loadInvoices, 1200);
    setTimeout(updateRepeatActionTabBadge, 250);
    $(document).on('click', '#sn-seller-panel .sn-tab[data-tab="repeat-actions"]', function(){ setTimeout(updateRepeatActionTabBadge, 120); });

	// ---- رندر نوار فیلتر مرحله اول ----

    function setSellerTabBadge(selector, count) {
      var $badge = $(selector);
      if (!$badge.length) return;
      count = Number(count || 0);
      if (count > 0) {
        try { $badge.text(count.toLocaleString('fa-IR')); } catch(e) { $badge.text(String(count)); }
        $badge.removeAttr('hidden').addClass('is-visible');
      } else {
        $badge.attr('hidden', 'hidden').removeClass('is-visible').text('۰');
      }
    }

    function updateRepeatActionTabBadge() {
      var count = 0;
      var $rows = $('#sn-tab-repeat-actions .sn-paid-referral-actions tbody tr');
      if ($rows.length) {
        count = $rows.filter(function(){
          return $(this).find('.sn-empty-state').length === 0 && $(this).children('td').length > 1 && !$(this).hasClass('sn-repeat-action-cancelled');
        }).length;
      }
      setSellerTabBadge('#sn-seller-tab-badge-repeat-actions', count);
      return count;
    }

    function sellerLeadHasNoStatus(lead) {
      return !String((lead && lead.seller_flow_status) || '').trim()
        && !String((lead && lead.lead_status) || '').trim();
    }

    function updateSellerTabBadges() {
      var noStatus = Number(sellerLeadMeta.no_status_total || 0);
      if (!noStatus && allLeads.length) {
        noStatus = allLeads.filter(sellerLeadHasNoStatus).length;
      }
      setSellerTabBadge('#sn-seller-tab-badge-no-status', noStatus);
      updateRepeatActionTabBadge();
    }

    function renderLeadFilterBar() {
      var bar = '<div class="sn-lead-tools">';
      bar += '<div class="sn-lead-filters" style="display:flex;gap:6px;flex-wrap:wrap;align-items:center">';
      // بدون وضعیت اول
      var nsActive = activeFilter === 'no-status';
      bar += '<button class="sn-btn sn-btn-sm ' + (nsActive ? 'sn-btn-primary' : 'sn-btn-ghost') + '" data-filter="no-status">📋 بدون وضعیت</button>';
	  // چهار وضعیت استاندارد حتی اگر فعلاً ردیفی نداشته باشند نمایش داده می‌شوند.
      allLeadStatuses.forEach(function(s) {
        var filterKey = String(s.slug || snSellerFlowSlug(s.label) || s.label || '');
        var active = activeFilter === filterKey;
        var style  = active
          ? 'background:' + s.color + ';color:#fff;border:none;box-shadow:0 2px 6px ' + s.color + '55'
          : 'background:transparent;border:1.5px solid ' + s.color + ';color:' + s.color;
        bar += '<button class="sn-btn sn-btn-sm" style="' + style + '" data-filter="' + esc(filterKey) + '">' + esc(s.label) + '</button>';
      });
      if (Number(sellerLeadMeta.recontact_total || 0) > 0 || allLeads.some(function(l){ return !!l.has_recontact; })) {
        bar += '<button class="sn-btn sn-btn-sm ' + (activeFilter === 'recontact' ? 'sn-btn-primary' : 'sn-btn-ghost') + '" data-filter="recontact">ارتباط مجدد با کارشناس</button>';
      }
      if (Number(sellerLeadMeta.extra_total || 0) > 0 || allLeads.some(function(l){ return !!l.is_extra_number_request; })) {
        bar += '<button class="sn-btn sn-btn-sm ' + (activeFilter === 'extra-number' ? 'sn-btn-primary' : 'sn-btn-ghost') + '" data-filter="extra-number">شماره‌های درخواستی</button>';
      }
      // همه آخر
      bar += '<button class="sn-btn sn-btn-sm ' + (activeFilter === 'all' ? 'sn-btn-primary' : 'sn-btn-ghost') + '" data-filter="all">همه</button>';
      bar += '</div>';
      bar += '<div class="sn-lead-search-box"><input type="search" id="sn-seller-lead-search" value="' + esc(sellerLeadSearch) + '" placeholder="جستجوی شماره، نام، شهر یا کد پرونده"><select id="sn-seller-lead-per-page"><option value="20"' + (sellerLeadPerPage===20?' selected':'') + '>۲۰ تایی</option><option value="50"' + (sellerLeadPerPage===50?' selected':'') + '>۵۰ تایی</option><option value="100"' + (sellerLeadPerPage===100?' selected':'') + '>۱۰۰ تایی</option></select></div>';
      bar += '</div>';
      $('#sn-leads-filter-bar').html(bar);
      $('#sn-leads-filter-bar [data-filter]').each(function(){
        $(this).attr('aria-pressed', String($(this).attr('data-filter') === activeFilter));
      });
    }

    $(document).on('click', '.sn-lead-filters button', function () {
      activeFilter = String($(this).data('filter') || 'all');
      sellerLeadPage = 1;
      expandedLeadId = null;
      renderLeadFilterBar();
      loadLeads();
    });

    $(document).on('input', '#sn-seller-lead-search', function () {
      sellerLeadSearch = String($(this).val() || '').trim();
      sellerLeadPage = 1;
      clearTimeout(sellerLeadSearchTimer);
      sellerLeadSearchTimer = setTimeout(function(){ loadLeads(); }, 350);
    });

    $(document).on('change', '#sn-seller-lead-per-page', function () {
      var value = Number($(this).val() || 50);
      sellerLeadPerPage = [20, 50, 100].indexOf(value) !== -1 ? value : 50;
      sellerLeadPage = 1;
      loadLeads();
    });

    $(document).on('click', '.sn-seller-page-btn', function () {
      var page = Number($(this).data('page') || 1);
      if (!page || page < 1 || page === sellerLeadPage) return;
      sellerLeadPage = page;
      expandedLeadId = null;
      loadLeads();
    });

    // ---- بارگذاری leads از سرور ----
    function loadLeads() {
      var requestSeq=++sellerLeadRequestSeq;
      var profileEnabled = /(?:^|[?&])sn_profile=1(?:&|$)/.test(window.location.search);
      var profileStarted = profileEnabled && window.performance ? performance.now() : 0;
      var profileData = null;
      $('#sn-leads-loading').show();
      $('#sn-leads-list').html(snSkeletonRows(5, 3));
      $.ajax({
        url: ajax,
        type: 'POST',
        dataType: 'json',
        timeout: 30000,
        data: {
          action: 'sn_seller_leads',
          nonce: nonce,
          page: sellerLeadPage,
          per_page: sellerLeadPerPage,
          filter: activeFilter,
          search: sellerLeadSearch,
          sn_profile: profileEnabled ? 1 : 0
        }
      })
        .done(function (res) {
          if(requestSeq!==sellerLeadRequestSeq)return;
          profileData = res && res.profile;
          $('#sn-leads-loading').hide();
          sellerLeadMeta = (res && res.pagination) ? res.pagination : { page: sellerLeadPage, per_page: sellerLeadPerPage, total: 0, total_all: 0, total_pages: 1 };
          sellerLeadPage = Number(sellerLeadMeta.page || sellerLeadPage || 1);
          sellerLeadPerPage = Number(sellerLeadMeta.per_page || sellerLeadPerPage || 50);
          if (!res || !res.success) {
            allLeads = [];
            renderLeadFilterBar();
            renderSellerKpis();
            fillLeadDropdown();
            updateSellerTabBadges();
            $('#sn-leads-list').html('<div class="sn-notice sn-error">' + esc((res && res.message) || 'لیست شماره‌ها بارگذاری نشد.') + '</div>');
            return;
          }
          allLeads = res.leads || [];
          renderLeadFilterBar();
          renderSellerKpis();
          fillLeadDropdown();
          updateSellerTabBadges();
          if (!allLeads.length) {
            var emptyText = sellerLeadSearch ? 'شماره‌ای مطابق جستجو و فیلتر فعلی یافت نشد.' : 'هنوز شماره‌ای در این وضعیت وجود ندارد.';
            $('#sn-leads-list').html('<div class="sn-notice sn-info" style="text-align:center;padding:24px">' + esc(emptyText) + '</div>' + renderLeadPagination());
            return;
          }
          renderLeadsTable();
        })
        .fail(function(xhr){
          if(requestSeq!==sellerLeadRequestSeq)return;
          $('#sn-leads-loading').hide();
          allLeads = [];
          renderLeadFilterBar();
          renderSellerKpis();
          fillLeadDropdown();
          updateSellerTabBadges();
          var msg = 'لیست شماره‌ها بارگذاری نشد.';
          if (xhr && xhr.status) msg += ' کد خطا: ' + xhr.status;
          $('#sn-leads-list').html('<div class="sn-notice sn-error">' + esc(msg) + '<br><button type="button" class="sn-btn sn-btn-sm sn-btn-secondary" id="sn-reload-seller-leads">تلاش مجدد</button></div>');
        }).always(function(){
          if(profileEnabled && window.console && console.info){
            console.info('[CRM-44] seller leads', {client_ms:Math.round(performance.now()-profileStarted),server:profileData,rows:Number(sellerLeadMeta.current_count||0)});
          }
        });
    }

    $(document).on('click', '#sn-reload-seller-leads', function(){ loadLeads(); });
    window.snSellerReloadLeads = loadLeads;

    function fillLeadDropdown() {
      var $sel = $('#sn-lead-select');
      $sel.empty().append('<option value="">— بدون تخصیص —</option>');
      allLeads.forEach(function(l) {
        if (l.status !== 'invoiced') {
          var tag = l.is_extra_number_request ? ' — درخواستی' : (l.is_v4_distribution ? ' — MIS' : '');
          $sel.append('<option value="' + l.id + '" data-phone="' + l.phone + '">' + l.phone + tag + '</option>');
        }
      });
    }


    function renderSellerKpis() {
      var totalLeads = Number(sellerLeadMeta.total_all || allLeads.length || 0);
      var noStatus = Number(sellerLeadMeta.no_status_total || 0);
      if (!noStatus && allLeads.length) noStatus = allLeads.filter(sellerLeadHasNoStatus).length;
      var statusDone = Number(sellerLeadMeta.status_done_total || Math.max(0, totalLeads - noStatus));
      var invoiceCount = Number(sellerInvoiceSummary.pre_invoice || sellerInvoices.length || 0);
      var paidCount = Number(sellerInvoiceSummary.paid || sellerInvoices.filter(function(i){ return ['paid','approved'].indexOf(String(i.status || i.payment_status || i.invoice_status || '')) !== -1; }).length);
      var revenue = sellerPaidAmount || sellerInvoices.reduce(function(sum, i){
        var st = String(i.status || i.payment_status || i.invoice_status || '');
        if (['paid','approved'].indexOf(st) !== -1) return sum + Number(i.product_price || i.amount || 0);
        return sum;
      }, 0);
      var html = '<div class="sn-kpi-grid sn-seller-kpis">' +
        '<div class="sn-kpi-card"><span class="sn-kpi-icon">📞</span><small>کل شماره‌ها</small><strong>' + snFormatNumber(totalLeads) + '</strong><em>شماره‌های تخصیص‌یافته</em></div>' +
        '<div class="sn-kpi-card"><span class="sn-kpi-icon">✅</span><small>پیگیری‌شده</small><strong>' + snFormatNumber(statusDone) + '</strong><em>' + snPercent(statusDone, totalLeads) + ' از کل شماره‌ها</em></div>' +
        '<div class="sn-kpi-card"><span class="sn-kpi-icon">🧾</span><small>پیش‌فاکتور</small><strong>' + snFormatNumber(invoiceCount) + '</strong><em>تبدیل: ' + snPercent(invoiceCount, totalLeads) + '</em></div>' +
        '<div class="sn-kpi-card sn-kpi-money"><span class="sn-kpi-icon">💰</span><small>فروش تاییدشده</small><strong>' + '<span class="sn-money-number" dir="ltr">' + snFormatNumber(revenue) + '</span> <span class="sn-money-currency">تومان</span>' + '</strong><em>' + snFormatNumber(paidCount) + ' پرداخت موفق</em></div>' +
      '</div>';
      var $target = $('#sn-seller-kpi-cards');
      if (!$target.length) {
        $target = $('<div id="sn-seller-kpi-cards" class="sn-kpi-host"></div>');
        var $tab = $('#sn-tab-leads');
        if ($tab.length) $tab.prepend($target); else $sellerPanel.prepend($target);
      }
      $target.html(html);
    }

    function renderLeadPagination() {
      var meta = sellerLeadMeta || {};
      var total = Number(meta.total || 0);
      var totalAll = Number(meta.total_all || total || 0);
      var page = Math.max(1, Number(meta.page || sellerLeadPage || 1));
      var totalPages = Math.max(1, Number(meta.total_pages || 1));
      var perPage = Number(meta.per_page || sellerLeadPerPage || 50);
      var from = total ? ((page - 1) * perPage + 1) : 0;
      var to = Math.min(total, page * perPage);
      var html = '<div class="sn-seller-pagination">';
      html += '<div class="sn-seller-pagination-info">نمایش ' + snFormatNumber(from) + ' تا ' + snFormatNumber(to) + ' از ' + snFormatNumber(total) + ' مورد فیلترشده';
      if (totalAll && totalAll !== total) html += ' / کل: ' + snFormatNumber(totalAll);
      html += '</div>';
      html += '<div class="sn-seller-pagination-actions">';
      html += '<button type="button" class="sn-btn sn-btn-sm sn-seller-page-btn" data-page="1"' + (page <= 1 ? ' disabled' : '') + '>اول</button>';
      html += '<button type="button" class="sn-btn sn-btn-sm sn-seller-page-btn" data-page="' + (page - 1) + '"' + (page <= 1 ? ' disabled' : '') + '>قبلی</button>';
      var start = Math.max(1, page - 2), end = Math.min(totalPages, page + 2);
      for (var i = start; i <= end; i++) {
        html += '<button type="button" class="sn-btn sn-btn-sm sn-seller-page-btn ' + (i === page ? 'sn-btn-primary' : 'sn-btn-ghost') + '" data-page="' + i + '">' + snFormatNumber(i) + '</button>';
      }
      html += '<button type="button" class="sn-btn sn-btn-sm sn-seller-page-btn" data-page="' + (page + 1) + '"' + (page >= totalPages ? ' disabled' : '') + '>بعدی</button>';
      html += '<button type="button" class="sn-btn sn-btn-sm sn-seller-page-btn" data-page="' + totalPages + '"' + (page >= totalPages ? ' disabled' : '') + '>آخر</button>';
      html += '</div></div>';
      return html;
    }

    // ---- رندر لیست leads — کارت‌های کلیک‌پذیر ----
    function renderLeadsTable() {
      var filtered = allLeads.slice();

      if (!filtered.length) {
        $('#sn-leads-list').html('<div class="sn-notice sn-info" style="text-align:center;padding:24px">شماره‌ای با این وضعیت یافت نشد.</div>' + renderLeadPagination());
        return;
      }

      var statusOptions = allLeadStatuses.map(function(s) {
        return '<option value="' + s.label + '">' + s.label + '</option>';
      }).join('');

      var html = '<table class="sn-table" style="border-collapse:collapse;width:100%">' +
        '<thead><tr>' +
        '<th style="width:180px">شماره</th>' +
        '<th style="width:160px">تاریخ تخصیص</th>' +
        '<th>اطلاعات مشتری</th>' +
        '</tr></thead><tbody>';

      filtered.forEach(function(l, idx) {
        var rowBg = idx % 2 === 0 ? '#fff' : '#f8fafc';
        var distAttr = l.distribution_item_id ? ' data-distribution-id="' + esc(l.distribution_item_id) + '"' : '';
		var flowSlug = String(l.seller_flow_status || '');
		var noAnswerAttempts = Math.max(0, Math.min(3, Number(l.no_answer_attempts || 0)));
		var noAnswerDue = String(l.archive_due_at || '');
		var archiveError = String(l.archive_error || '');
		var sourceKind = String(l.source_kind || (l.distribution_item_id ? 'v4_distribution_item' : 'legacy_lead'));

        // badge وضعیت تماس
        var statusBadge = '';
        if (l.lead_status) {
          var found = allLeadStatuses.find(function(s){ return s.label === l.lead_status; });
          statusBadge = found
			? '<span style="background:' + esc(found.color) + ';color:#fff;padding:1px 7px;border-radius:8px;font-size:.75rem;margin-right:4px">' + esc(found.label) + '</span>'
			: '<span style="background:#e2e8f0;color:#475569;padding:1px 7px;border-radius:8px;font-size:.75rem">' + esc(l.lead_status) + '</span>';
        }

        // خلاصه اطلاعات مشتری
        var custInfo = [];
		if (l.customer_name) custInfo.push('<strong>' + esc(l.customer_name) + '</strong>');
		if (l.province && l.city) custInfo.push(esc(l.province) + ' — ' + esc(l.city));
		else if (l.province) custInfo.push(esc(l.province));
        if (l.sales_prediction) custInfo.push('احتمال: ' + esc(l.sales_prediction));
		if (l.duplicate_marked_by && flowSlug === 'duplicate') custInfo.push('ثبت تکراری توسط: ' + esc(l.duplicate_marked_by));
        if (l.lead_badge_label) custInfo.push('<span class="sn-lead-source-badge ' + esc(l.lead_badge_class || '') + '">' + esc(l.lead_badge_label) + '</span>');
		if (l.note) custInfo.push('📝 ' + esc(l.note));
        if (l.has_recontact) {
		  custInfo.push('<span class="sn-recontact-lead-chip">درخواست ارتباط مجدد' + (l.recontact_invoice_code ? ' — ' + esc(l.recontact_invoice_code) : '') + '</span>');
        }
        var custHtml = custInfo.length
          ? custInfo.join(' | ')
          : '<span style="color:#94a3b8;font-size:.78rem">اطلاعاتی ثبت نشده</span>';

        html += '<tr style="background:' + rowBg + '" id="sn-lead-row-' + l.id + '">' +
          // ستون شماره — کلیک‌پذیر
          '<td>' +
			'<button type="button" class="sn-phone-copy" data-phone="' + esc(l.phone) + '" title="کپی شماره">' + esc(l.phone) + '</button>' +
            '<button type="button" class="sn-phone-toggle sn-btn sn-btn-sm sn-btn-ghost" data-id="' + l.id + '"' + distAttr + ' aria-expanded="false" onclick="return window.snToggleLeadEditor ? window.snToggleLeadEditor(this, event) : false;">مشاهده/ویرایش</button>' +
			'<div class="sn-copy-msg" data-phone="' + esc(l.phone) + '"></div>' +
            '<div class="sn-lead-status-badge-wrap" data-id="' + l.id + '" style="margin-top:3px">' + (statusBadge || '') + '</div>' +
            (l.lead_badge_label ? '<div class="sn-lead-source-badge-wrap"><span class="sn-lead-source-badge ' + esc(l.lead_badge_class || '') + '">' + esc(l.lead_badge_label) + '</span></div>' : '') +
          '</td>' +
          // تاریخ تخصیص
          '<td style="font-size:.78rem;color:#64748b;white-space:nowrap;vertical-align:top;padding-top:6px">' + toJalali(l.assigned_at) + '</td>' +
          // اطلاعات مشتری خلاصه
          '<td style="font-size:.82rem;color:#475569;vertical-align:top;padding-top:6px">' + custHtml + '</td>' +
        '</tr>' +

        // ---- dropdown row ----
        '<tr id="sn-expand-' + l.id + '" class="sn-lead-expand-row" data-id="' + l.id + '" style="display:none">' +
          '<td colspan="3" style="padding:0;border-top:2px solid #3b82f6">' +
            '<div class="sn-lead-editor">' +

              // ردیف اول: اطلاعات مشتری
              '<div class="sn-lead-editor-grid">' +
                '<div class="sn-lead-field sn-lead-field-name"><label>نام مشتری</label>' +
				  '<input type="text" class="sn-cust-name sn-auto-save" data-id="' + l.id + '"' + distAttr + ' value="' + esc(l.customer_name || '') + '" placeholder="نام و نام خانوادگی"></div>' +
                '<div class="sn-lead-field"><label>استان</label>' +
                  '<select class="sn-cust-prov sn-auto-save" data-id="' + l.id + '"' + distAttr + '>' +
                    '<option value="">انتخاب استان</option>' +
                    Object.keys(SN_CITIES).map(function(p){ return '<option value="' + p + '"' + (l.province===p?' selected':'') + '>' + p + '</option>'; }).join('') +
                  '</select></div>' +
                '<div class="sn-lead-field"><label>شهر</label>' +
                  '<select class="sn-cust-city sn-auto-save" data-id="' + l.id + '"' + distAttr + '>' +
                    snBuildCityOptions(l.province, l.city) +
                  '</select></div>' +
                '<div class="sn-lead-field"><label>احتمال فروش</label>' +
                  '<select class="sn-cust-pred sn-auto-save" data-id="' + l.id + '"' + distAttr + '>' +
                    '<option value="">انتخاب کنید</option>' +
                    ['ضعیف','متوسط','بالا','١٠٠٪'].map(function(v){ return '<option value="' + v + '"' + (l.sales_prediction===v?' selected':'') + '>' + v + '</option>'; }).join('') +
                  '</select></div>' +
                '<div class="sn-lead-field"><label>وضعیت تماس</label>' +
                  '<select class="sn-lead-status-select sn-auto-save" data-id="' + l.id + '"' + distAttr + '>' +
                    '<option value="">— تعیین وضعیت —</option>' + statusOptions +
                  '</select></div>' +
				'<div class="sn-lead-field sn-full sn-seller-flow-details" data-id="' + l.id + '">' +
				  '<div class="sn-flow-no-answer" data-id="' + l.id + '" style="display:' + (flowSlug === 'no_answer' ? 'block' : 'none') + '">' +
				    '<div class="sn-flow-counter"><strong>تماس بی‌پاسخ: <span class="sn-no-answer-count">' + noAnswerAttempts + '</span> از ۳</strong>' +
				      (noAnswerAttempts > 0 && noAnswerAttempts < 3 ? '<button type="button" class="sn-btn sn-btn-sm sn-btn-warning sn-add-no-answer-attempt" data-id="' + l.id + '" data-source-kind="' + esc(sourceKind) + '"' + distAttr + '>ثبت تماس بی‌پاسخ بعدی</button>' : '') +
				    '</div>' +
				    (noAnswerAttempts >= 3 ? '<p class="sn-flow-warning">سه تلاش ثبت شده است. اگر تا سه روز فعالیت دیگری ثبت نشود، این شماره به بایگانی مدیر فروش منتقل می‌شود.' + (noAnswerDue ? '<br><small>موعد بررسی: ' + esc(toJalali(noAnswerDue)) + '</small>' : '') + (archiveError === 'sales_manager_not_found' ? '<br><strong>انتقال انجام نشده: مدیر فروش این فروشنده در HR مشخص نیست.</strong>' : '') + '</p>' : '<p class="sn-note">هر تماس واقعیِ بی‌پاسخ را جدا ثبت کن؛ سقف این مرحله سه تلاش است.</p>') +
				  '</div>' +
				  '<div class="sn-flow-not-purchased" data-id="' + l.id + '" style="display:' + (flowSlug === 'not_purchased' ? 'block' : 'none') + '">' +
				    '<label>دلیل عدم خرید *</label><textarea class="sn-not-purchase-reason" data-id="' + l.id + '"' + distAttr + ' placeholder="دلیل اعلام‌شده یا نتیجه نهایی مکالمه را بنویسید">' + esc(l.not_purchase_reason || '') + '</textarea>' +
				    '<p class="sn-field-error sn-not-purchase-error" data-id="' + l.id + '" hidden>ثبت دلیل برای این وضعیت اجباری است.</p>' +
				  '</div>' +
				  '<div class="sn-flow-callback" data-id="' + l.id + '" style="display:' + (flowSlug === 'callback' ? 'block' : 'none') + '">' +
				    '<label>زمان تماس مجدد *</label><div style="display:flex;gap:6px;flex-wrap:wrap;align-items:center">' +
				    '<input class="sn-jalali-date sn-callback-date" data-id="' + l.id + '" id="sn-callback-date-' + l.id + '" value="' + esc(String(l.callback_at || '').split(' ')[0] || '') + '" placeholder="تاریخ شمسی" style="width:150px">' +
				    '<select class="sn-callback-hour" data-id="' + l.id + '" id="sn-callback-hour-' + l.id + '" aria-label="ساعت تماس"></select> : ' +
				    '<select class="sn-callback-minute" data-id="' + l.id + '" id="sn-callback-minute-' + l.id + '" aria-label="دقیقه تماس"></select>' +
				    '</div>' +
				    '<p class="sn-field-error sn-callback-error" data-id="' + l.id + '" hidden>تاریخ و ساعت تماس مجدد را انتخاب کنید.</p>' +
				  '</div>' +
				'</div>' +
                '<div class="sn-lead-field sn-lead-field-note"><label>یادداشت</label>' +
				  '<textarea class="sn-lead-note sn-auto-save" data-id="' + l.id + '"' + distAttr + ' placeholder="خلاصه مکالمه، نیاز مشتری، زمان پیگیری...">' + esc(l.note || '') + '</textarea></div>' +
                '<div class="sn-inline-save-state" data-id="' + l.id + '" aria-live="polite"></div>' +
              '</div>' +

              // ردیف دوم: ذخیره قطعی + صدور پیش‌فاکتور
              '<div class="sn-lead-editor-actions">' +
                '<div style="display:flex;gap:8px;flex-wrap:wrap;align-items:center">' +
                  '<button type="button" class="sn-btn sn-btn-sm sn-btn-secondary sn-save-lead-editor" data-id="' + l.id + '"' + distAttr + '>💾 ذخیره تغییرات</button>' +
				  (sellerCanIssueInvoice ? '<button type="button" class="sn-btn sn-btn-sm sn-btn-primary sn-use-lead" data-phone="' + esc(l.phone) + '" data-id="' + l.id + '">📄 صدور پیش‌فاکتور</button>' : '') +
                '</div>' +
              '</div>' +
            '</div>' +
          '</td>' +
        '</tr>';
      });

      html += '</tbody></table>';
      html += renderLeadPagination();
      $('#sn-leads-list').html(html);

      // ست کردن مقدار فعلی وضعیت تماس
      filtered.forEach(function(l) {
		if (window.snPaymentTimeInit) window.snPaymentTimeInit('#sn-callback-hour-' + l.id, '#sn-callback-minute-' + l.id);
		else {
		  var $hour = $('#sn-callback-hour-' + l.id), $minute = $('#sn-callback-minute-' + l.id);
		  for (var h=0;h<24;h++) $hour.append($('<option>').val(snPad(h)).text(snPad(h)));
		  for (var m=0;m<60;m++) $minute.append($('<option>').val(snPad(m)).text(snPad(m)));
		}
		var time = String(l.callback_at || '').split(' ')[1] || '';
		if (time) {
		  $('#sn-callback-hour-' + l.id).val(time.split(':')[0]);
		  $('#sn-callback-minute-' + l.id).val(time.split(':')[1]);
		}
        if (l.lead_status) {
          $('select.sn-lead-status-select[data-id="' + l.id + '"]').val(l.lead_status);
        }
      });

      if (expandedLeadId) {
        $('#sn-expand-' + expandedLeadId).show();
      }
    }

    function getLeadById(id) {
      return allLeads.find(function(l){ return String(l.id) === String(id); }) || null;
    }

    function collectLeadFormData(id, suppliedPayload) {
      id = String(id || '');
      if (suppliedPayload && typeof suppliedPayload === 'object') {
        return $.extend({}, suppliedPayload, { lead_id: id });
      }
      var lead = getLeadById(id) || {};
      function val(selector, fallback) {
        var $el = $(selector);
        if ($el.length) {
          var v = $el.val();
          return v == null ? '' : String(v);
        }
        return fallback == null ? '' : String(fallback);
      }
      function distId() {
        var $el = $('[data-id="' + id + '"][data-distribution-id]').first();
        var v = $el.length ? String($el.attr('data-distribution-id') || '') : String(lead.distribution_item_id || '');
        return v.replace(/\D+/g, '');
      }
      return {
        lead_id: id,
        distribution_item_id: distId(),
		source_kind: String(lead.source_kind || (distId() ? 'v4_distribution_item' : 'legacy_lead')),
        customer_name: val('.sn-cust-name[data-id="' + id + '"]', lead.customer_name).trim(),
        province: val('.sn-cust-prov[data-id="' + id + '"]', lead.province),
        city: val('.sn-cust-city[data-id="' + id + '"]', lead.city),
        sales_prediction: val('.sn-cust-pred[data-id="' + id + '"]', lead.sales_prediction),
        note: val('.sn-lead-note[data-id="' + id + '"]', lead.note),
		lead_status: val('select.sn-lead-status-select[data-id="' + id + '"]', lead.lead_status),
		not_purchase_reason: val('.sn-not-purchase-reason[data-id="' + id + '"]', lead.not_purchase_reason).trim(),
		callback_at: val('.sn-callback-date[data-id="' + id + '"]', String(lead.callback_at || '').split(' ')[0]).trim() + ' ' + val('.sn-callback-hour[data-id="' + id + '"]', '09') + ':' + val('.sn-callback-minute[data-id="' + id + '"]', '00')
      };
    }

    function setLeadSaveState(id, state, message) {
      var $box = $('.sn-inline-save-state[data-id="' + id + '"]');
      if (!$box.length) return;
      if (state === 'saving') {
        $box.text(message || 'در حال ذخیره...').css('color', '#2563eb');
      } else if (state === 'saved') {
        $box.text(message || 'ذخیره شد').css('color', '#16a34a');
      } else if (state === 'error') {
        $box.text(message || 'خطا در ذخیره').css('color', '#dc2626');
      } else {
        $box.text(message || '').css('color', '#64748b');
      }
    }

    function syncLeadToMemory(id, data) {
      var lead = getLeadById(id);
      if (!lead || !data) return;
      lead.customer_name = data.customer_name || '';
      lead.province = data.province || '';
      lead.city = data.city || '';
      lead.sales_prediction = data.sales_prediction || '';
      lead.note = data.note || '';
      if (Object.prototype.hasOwnProperty.call(data, 'lead_status')) lead.lead_status = data.lead_status || '';
      if (data.distribution_item_id) lead.distribution_item_id = data.distribution_item_id;
	  if (Object.prototype.hasOwnProperty.call(data, 'seller_flow_status')) lead.seller_flow_status = data.seller_flow_status || '';
	  if (Object.prototype.hasOwnProperty.call(data, 'no_answer_attempts')) lead.no_answer_attempts = Number(data.no_answer_attempts || 0);
	  if (Object.prototype.hasOwnProperty.call(data, 'not_purchase_reason')) {
		lead.not_purchase_reason = data.not_purchase_reason || '';
		$('.sn-not-purchase-reason[data-id="' + id + '"]').val(lead.not_purchase_reason);
	  }
	  if (Object.prototype.hasOwnProperty.call(data, 'archive_due_at')) lead.archive_due_at = data.archive_due_at || '';
	  if (Object.prototype.hasOwnProperty.call(data, 'callback_at')) lead.callback_at = data.callback_at || '';
	  if (Object.prototype.hasOwnProperty.call(data, 'archive_error')) lead.archive_error = data.archive_error || '';
    }

	function snSellerFlowSlug(statusValue) {
	  var value = String(statusValue || '').trim();
	  if (value === 'جواب نداده') return 'no_answer';
	  if (value === 'تماس مجدد') return 'callback';
	  if (value === 'تکراری') return 'duplicate';
	  if (value === 'عدم خرید') return 'not_purchased';
	  if (value === 'پیش‌فاکتور' || value === 'پیش فاکتور' || value === 'پیش‌فاکتور شده') return 'pre_invoice';
	  return '';
	}

	function syncSellerFlowDetails(id, statusValue) {
	  id = String(id || '');
	  var slug = snSellerFlowSlug(statusValue);
	  $('.sn-flow-no-answer[data-id="' + id + '"]').toggle(slug === 'no_answer');
	  $('.sn-flow-not-purchased[data-id="' + id + '"]').toggle(slug === 'not_purchased');
	  $('.sn-flow-callback[data-id="' + id + '"]').toggle(slug === 'callback');
	  if (slug !== 'not_purchased') {
		$('.sn-not-purchase-error[data-id="' + id + '"]').attr('hidden', 'hidden');
	  }
	}

	function validateSellerFlowPayload(id, payload) {
	  if (snSellerFlowSlug(payload.lead_status) === 'callback' && !/^\d{4}\/\d{1,2}\/\d{1,2}\s+\d{2}:\d{2}$/.test(snToEnDigits(String(payload.callback_at || '')).trim())) {
		$('.sn-callback-error[data-id="' + id + '"]').removeAttr('hidden');
		setLeadSaveState(id, 'error', 'تاریخ و ساعت تماس مجدد را انتخاب کنید.');
		return false;
	  }
	  $('.sn-callback-error[data-id="' + id + '"]').attr('hidden', 'hidden');
	  if (snSellerFlowSlug(payload.lead_status) !== 'not_purchased') return true;
	  if (String(payload.not_purchase_reason || '').trim()) {
		$('.sn-not-purchase-error[data-id="' + id + '"]').attr('hidden', 'hidden');
		return true;
	  }
	  syncSellerFlowDetails(id, payload.lead_status);
	  $('.sn-not-purchase-error[data-id="' + id + '"]').removeAttr('hidden');
	  setLeadSaveState(id, 'error', 'برای ثبت «عدم خرید»، دلیل را وارد کنید.');
	  var field = $('.sn-not-purchase-reason[data-id="' + id + '"]').get(0);
	  if (field && typeof field.focus === 'function') field.focus();
	  return false;
	}

    function updateLeadStatusBadge(id, statusValue) {
      var $wrap = $('.sn-lead-status-badge-wrap[data-id="' + id + '"]');
      if (!$wrap.length) return;
      var value = String(statusValue || '');
      if (!value) { $wrap.html(''); return; }
      var found = allLeadStatuses.find(function(s){ return String(s.label) === value; });
      if (found) {
        $wrap.html('<span style="background:' + found.color + ';color:#fff;padding:1px 7px;border-radius:8px;font-size:.75rem;margin-right:4px">' + esc(found.label) + '</span>');
      } else {
        $wrap.html('<span style="background:#e2e8f0;color:#475569;padding:1px 7px;border-radius:8px;font-size:.75rem">' + esc(value) + '</span>');
      }
    }

    function updateLeadRowSummary(id, data) {
      // حفظ حالت باز فرم: بعد از ذخیره، کل جدول re-render نمی‌شود.
      // فقط خلاصه ردیف فعلی تا حد ممکن به‌روز می‌شود.
      var row = document.getElementById('sn-lead-row-' + String(id));
      if (!row) return;
      var cells = row.querySelectorAll('td');
      if (!cells || cells.length < 3) return;
      var summary = [];
      if (data.customer_name) summary.push('<strong>' + esc(data.customer_name) + '</strong>');
      if (data.province && data.city) summary.push(esc(data.province) + ' — ' + esc(data.city));
      else if (data.province) summary.push(esc(data.province));
      if (data.sales_prediction) summary.push('احتمال: ' + esc(data.sales_prediction));
      if (data.note) summary.push('📝 ' + esc(data.note));
      cells[2].innerHTML = summary.length ? summary.join(' | ') : '<span style="color:#94a3b8;font-size:.78rem">اطلاعاتی ثبت نشده</span>';
      updateLeadStatusBadge(id, data.lead_status);
    }

    function snPayloadHash(payload) {
      try { return JSON.stringify(payload || {}); } catch(e) { return String(Date.now()); }
    }

    function saveLeadData(id, options) {
      id = String(id || '');
      if (!id) return null;
      options = options || {};
      clearTimeout(saveTimers[id]);

      var payload = collectLeadFormData(id, options.payload || null);
	  if (!validateSellerFlowPayload(id, payload)) return null;
      var hash = snPayloadHash(payload);
      if (!options.force && lastSavedPayloadHash[id] === hash) {
        setLeadSaveState(id, 'idle', '');
        return null;
      }

      // اگر ذخیره قبلی هنوز در جریان است، درخواست جدید را abort نکن؛ آخرین snapshot را در صف نگه دار.
      // abort در بعضی مرورگرها باعث می‌شد وضعیت «در حال ذخیره...» بماند یا ذخیره واقعی دیرتر از UI انجام شود.
      if (pendingSaves[id] && pendingSaves[id].readyState !== 4) {
        dirtySaveQueued[id] = $.extend({}, options, { force: true, payload: payload });
        setLeadSaveState(id, 'saving', 'در حال ذخیره آخرین تغییرات...');
        return pendingSaves[id];
      }
      activeSavePayload[id] = payload;

      setLeadSaveState(id, 'saving', options.savingText || 'در حال ذخیره...');
      pendingSaves[id] = $.ajax({
        url: ajax,
        type: 'POST',
        dataType: 'json',
        data: {
          action: 'sn_save_customer_info',
          nonce: nonce,
          lead_id: id,
          distribution_item_id: payload.distribution_item_id || '',
		  source_kind: payload.source_kind || '',
          changed_field: options.changedField || '',
          customer_name: payload.customer_name,
          province: payload.province,
          city: payload.city,
          sales_prediction: payload.sales_prediction,
          note: payload.note,
		  lead_status: payload.lead_status,
		  not_purchase_reason: payload.not_purchase_reason || '',
		  callback_at: payload.callback_at || ''
        }
      }).done(function(r) {
        if (r && r.success) {
          var savedPayload = $.extend({}, payload, {
            customer_name: Object.prototype.hasOwnProperty.call(r, 'customer_name') ? r.customer_name : payload.customer_name,
            province: Object.prototype.hasOwnProperty.call(r, 'province') ? r.province : payload.province,
            city: Object.prototype.hasOwnProperty.call(r, 'city') ? r.city : payload.city,
            sales_prediction: Object.prototype.hasOwnProperty.call(r, 'sales_prediction') ? r.sales_prediction : payload.sales_prediction,
            note: Object.prototype.hasOwnProperty.call(r, 'note') ? r.note : payload.note,
            lead_status: Object.prototype.hasOwnProperty.call(r, 'lead_status') ? r.lead_status : payload.lead_status,
			distribution_item_id: Object.prototype.hasOwnProperty.call(r, 'distribution_item_id') ? r.distribution_item_id : payload.distribution_item_id,
			seller_flow_status: Object.prototype.hasOwnProperty.call(r, 'seller_flow_status') ? r.seller_flow_status : snSellerFlowSlug(payload.lead_status),
			no_answer_attempts: Object.prototype.hasOwnProperty.call(r, 'no_answer_attempts') ? r.no_answer_attempts : (getLeadById(id) || {}).no_answer_attempts,
			callback_at: Object.prototype.hasOwnProperty.call(r, 'callback_at') ? r.callback_at : payload.callback_at,
			not_purchase_reason: Object.prototype.hasOwnProperty.call(r, 'not_purchase_reason') ? r.not_purchase_reason : payload.not_purchase_reason,
			archive_due_at: Object.prototype.hasOwnProperty.call(r, 'archive_due_at') ? r.archive_due_at : (getLeadById(id) || {}).archive_due_at,
			archive_error: Object.prototype.hasOwnProperty.call(r, 'archive_error') ? r.archive_error : (getLeadById(id) || {}).archive_error
          });
          lastSavedPayloadHash[id] = snPayloadHash(savedPayload);
          syncLeadToMemory(id, savedPayload);
          updateLeadRowSummary(id, savedPayload);
		  syncSellerFlowDetails(id, savedPayload.lead_status);
          setLeadSaveState(id, 'saved', options.savedText || 'ذخیره شد');

          if (options.changedField === 'lead_status') {
            // Move the row using the canonical workflow status. Display labels are
            // presentation only and may differ between legacy/MIS data.
            activeFilter = String(savedPayload.seller_flow_status || snSellerFlowSlug(savedPayload.lead_status) || 'no-status');
            sellerLeadPage = 1;
            renderLeadFilterBar();
            setTimeout(function(){ loadLeads(); }, 450);
          }

          setTimeout(function(){
            if (!dirtySaveQueued[id]) setLeadSaveState(id, 'idle', '');
          }, 1200);
          if (typeof options.onSuccess === 'function') {
            options.onSuccess(r, savedPayload);
          }
        } else {
          var message = (r && r.message) ? r.message : 'خطا در ذخیره';
          if (r && r.db_error) message += ' — ' + r.db_error;
          setLeadSaveState(id, 'error', message);
          if (typeof options.onError === 'function') options.onError(r);
        }
      }).fail(function(xhr, status) {
        if (status === 'abort') return;
        var msg = 'خطا در ارتباط با سرور';
        if (xhr && xhr.responseJSON && xhr.responseJSON.message) msg = xhr.responseJSON.message;
        else if (xhr && xhr.responseText) msg = String(xhr.responseText).replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim().slice(0, 180);
        else if (xhr && xhr.status) msg += ' (' + xhr.status + ')';
        setLeadSaveState(id, 'error', msg || 'خطا در ارتباط با سرور');
        if (typeof options.onError === 'function') options.onError(xhr);
      }).always(function() {
        delete pendingSaves[id];
        delete activeSavePayload[id];
        if (dirtySaveQueued[id]) {
          var next = dirtySaveQueued[id];
          delete dirtySaveQueued[id];
          setTimeout(function(){ saveLeadData(id, next); }, 10);
        }
      });
      return pendingSaves[id];
    }

    function queueLeadAutoSave(id, delay, options) {
      id = String(id || '');
      if (!id) return;
      delay = typeof delay === 'number' ? delay : 0;
      clearTimeout(saveTimers[id]);
      saveTimers[id] = setTimeout(function() {
        saveLeadData(id, $.extend({}, options || {}, { force: true }));
      }, delay);
    }

    function snEscapeSelector(value) {
      value = String(value || '');
      if ($.escapeSelector) return $.escapeSelector(value);
      return value.replace(/([ #;?%&,.+*~\':"!^$[\]()=>|\/])/g, '\\$1');
    }

    function snFindLeadExpandRow(button, id) {
      var key = String(id || '');
      var row = null;
      // getElementById احتیاج به escape ندارد و برای IDهای عددی/مجازی پایدارتر از jQuery selector است.
      if (key) {
        row = document.getElementById('sn-expand-' + key);
      }
      if (!row && button) {
        var tr = button.closest ? button.closest('tr') : null;
        if (tr && tr.nextElementSibling && tr.nextElementSibling.classList && tr.nextElementSibling.classList.contains('sn-lead-expand-row')) {
          row = tr.nextElementSibling;
        }
      }
      return row ? $(row) : $();
    }

    function snShowLeadToggleError(button, msg) {
      var $cell = $(button).closest('td');
      if ($cell.length && !$cell.find('.sn-inline-error').length) {
        $cell.append('<div class="sn-inline-error" style="color:#dc2626;font-size:.78rem;margin-top:4px">' + msg + '</div>');
      }
    }

    // تابع global: هم inline onclick و هم listener عادی از همین استفاده می‌کنند تا مشکل event delegation/cache/selector حذف شود.
    window.snToggleLeadEditor = function(button, ev) {
      if (ev) {
        if (typeof ev.preventDefault === 'function') ev.preventDefault();
        if (typeof ev.stopPropagation === 'function') ev.stopPropagation();
        ev.snSellerToggleHandled = true;
      }
      var id = button ? (button.getAttribute('data-id') || '') : '';
      var $expand = snFindLeadExpandRow(button, id);
      if (!$expand.length) {
        snShowLeadToggleError(button, 'ردیف ویرایش برای این شماره پیدا نشد');
        return false;
      }

      $('.sn-lead-expand-row').not($expand).each(function(){ this.style.display = 'none'; });
      $('.sn-phone-toggle').not(button).attr('aria-expanded', 'false');

      var row = $expand.get(0);
      var visible = row && row.style.display !== 'none' && window.getComputedStyle(row).display !== 'none';
      if (visible) {
        try { saveLeadData(id, { force: true, changedField: 'close_editor', savingText: 'در حال ذخیره تغییرات...', savedText: 'تغییرات ذخیره شد' }); } catch(e) {}
        row.style.display = 'none';
        $(button).attr('aria-expanded', 'false');
        expandedLeadId = null;
      } else {
        row.style.display = 'table-row';
        $(button).attr('aria-expanded', 'true');
        expandedLeadId = String(id);
        setTimeout(function(){
          var first = row.querySelector('input,select,textarea,button');
          if (first && typeof first.focus === 'function') first.focus({preventScroll:true});
        }, 0);
      }
      return false;
    };

    if (!window.snSellerNativeToggleBound) {
      window.snSellerNativeToggleBound = true;
      document.addEventListener('click', function(ev) {
        var target = ev.target;
        var btn = target && target.closest ? target.closest('.sn-phone-toggle') : null;
        if (!btn) return;
        window.snToggleLeadEditor(btn, ev);
      }, true);
    }

    // fallback jQuery برای مرورگرهای قدیمی یا اگر listener capture حذف شد.
    $(document).on('click', '.sn-phone-toggle', function(e) {
      if (e.originalEvent && e.originalEvent.snSellerToggleHandled) return false;
      return window.snToggleLeadEditor(this, e);
    });

    $(document).on('click', '.sn-phone-copy', function(e) {
      e.preventDefault();
      var phone = String($(this).data('phone') || '');
      var $msg = $('.sn-copy-msg[data-phone="' + phone + '"]');
      function done() {
        $msg.text('شماره کپی شد').show();
        setTimeout(function(){ $msg.fadeOut(150); }, 1400);
      }
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(phone).then(done).catch(done);
      } else {
        var input = $('<input>').val(phone).appendTo('body').select();
        try { document.execCommand('copy'); } catch(e) {}
        input.remove();
        done();
      }
    });

    function snFieldLeadId(el) {
      return String($(el).data('id') || el.getAttribute('data-id') || '');
    }

    function snSaveFromField(el, changedField, immediate) {
      var id = snFieldLeadId(el);
      if (!id) return;
      var opts = {
        force: true,
        changedField: changedField || '',
        savingText: changedField === 'lead_status' ? 'در حال ذخیره وضعیت...' : 'در حال ذخیره...',
        savedText: changedField === 'lead_status' ? 'وضعیت ذخیره شد' : 'ذخیره شد'
      };
      if (immediate) saveLeadData(id, opts);
      else queueLeadAutoSave(id, 0, opts);
    }

    // تغییر استان → آپدیت شهرها + ذخیره قطعی همان لحظه تغییر
    $(document).on('change', '.sn-cust-prov', function() {
      var id = snFieldLeadId(this);
      var prov = $(this).val();
      $('.sn-cust-city[data-id="' + id + '"]').html(snBuildCityOptions(prov, ''));
      snSaveFromField(this, 'province', true);
    });

    // تغییر شهر / پیش‌بینی فروش = ذخیره مستقیم، بدون صف
    $(document).on('change', '.sn-cust-city', function() { snSaveFromField(this, 'city', true); });
    $(document).on('change', '.sn-cust-pred', function() { snSaveFromField(this, 'sales_prediction', true); });

	function snHandleSellerStatusChange(el) {
	  var id = snFieldLeadId(el);
	  var statusValue = String($(el).val() || '');
	  syncSellerFlowDetails(id, statusValue);
	  if (snSellerFlowSlug(statusValue) === 'pre_invoice') {
		var previousStatus = String((getLeadById(id) || {}).lead_status || '');
		$(el).val(previousStatus);
		syncSellerFlowDetails(id, previousStatus);
		var invoiceButton = $('.sn-use-lead[data-id="' + id + '"]').get(0);
		if (invoiceButton && sellerCanIssueInvoice) {
		  $(invoiceButton).trigger('click');
		} else {
		  setLeadSaveState(id, 'error', 'دسترسی صدور پیش‌فاکتور برای این حساب فعال نیست.');
		}
		return;
	  }
	  if (snSellerFlowSlug(statusValue) === 'not_purchased' && !String($('.sn-not-purchase-reason[data-id="' + id + '"]').val() || '').trim()) {
		$('.sn-not-purchase-error[data-id="' + id + '"]').removeAttr('hidden');
		setLeadSaveState(id, 'idle', 'برای ثبت این وضعیت، ابتدا دلیل عدم خرید را بنویسید.');
		var reasonField = $('.sn-not-purchase-reason[data-id="' + id + '"]').get(0);
		if (reasonField && typeof reasonField.focus === 'function') reasonField.focus();
		return;
	  }
	  if (snSellerFlowSlug(statusValue) === 'callback' && !String($('.sn-callback-date[data-id="' + id + '"]').val() || '').trim()) {
		setLeadSaveState(id, 'idle', 'برای تماس مجدد، تاریخ و ساعت را انتخاب کنید.');
		return;
	  }
	  saveLeadData(id, {
		force: true,
		changedField: 'lead_status',
		savingText: 'در حال ذخیره وضعیت...',
		savedText: 'وضعیت ذخیره شد'
	  });
	}

	// وضعیت تماس: فقط یک مسیر قطعی. بعد از پاسخ موفق سرور، ردیف زیر تب همان وضعیت منتقل می‌شود.
	$(document).on('change', '.sn-lead-status-select', function(e) {
	  if (e && e.originalEvent && e.originalEvent.snSellerStatusHandled) return;
	  snHandleSellerStatusChange(this);
	});
	$(document).on('change', '.sn-callback-date, .sn-callback-hour, .sn-callback-minute', function() {
	  var id = snFieldLeadId(this);
	  if (snSellerFlowSlug($('select.sn-lead-status-select[data-id="' + id + '"]').val()) === 'callback' && $('.sn-callback-date[data-id="' + id + '"]').val()) {
		saveLeadData(id, { force: true, changedField: 'lead_status', savedText: 'زمان تماس مجدد ذخیره شد' });
	  }
	});

    if (!window.snSellerStatusNativeSaveBound) {
      window.snSellerStatusNativeSaveBound = true;
      document.addEventListener('change', function(ev) {
		var el = ev.target && ev.target.closest ? ev.target.closest('.sn-lead-status-select') : null;
		if (!el) return;
		ev.snSellerStatusHandled = true;
		snHandleSellerStatusChange(el);
      }, true);
    }

    // نام و یادداشت: هنگام تایپ ذخیره نکن؛ فقط وقتی کاربر با Tab/ماوس از فیلد خارج شد، مستقیم ذخیره کن.
	$(document).on('input', '.sn-cust-name, .sn-lead-note, .sn-not-purchase-reason', function() {
	  var id = snFieldLeadId(this);
	  clearTimeout(saveTimers[id]);
	  if ($(this).hasClass('sn-not-purchase-reason') && String($(this).val() || '').trim()) {
		$('.sn-not-purchase-error[data-id="' + id + '"]').attr('hidden', 'hidden');
	  }
	  setLeadSaveState(id, 'idle', 'در حال ویرایش؛ با خروج از فیلد ذخیره می‌شود.');
    });

	$(document).on('focusout', '.sn-cust-name', function() { snSaveFromField(this, 'customer_name', true); });
	$(document).on('focusout', '.sn-lead-note', function() { snSaveFromField(this, 'note', true); });
	$(document).on('focusout', '.sn-not-purchase-reason', function() {
	  var id = snFieldLeadId(this);
	  var lead = getLeadById(id) || {};
	  var changedField = String(lead.lead_status || '') === 'عدم خرید' ? 'not_purchase_reason' : 'lead_status';
	  saveLeadData(id, { force: true, changedField: changedField, savingText: 'در حال ثبت دلیل...', savedText: 'دلیل عدم خرید ذخیره شد' });
	});

	$(document).on('click', '.sn-add-no-answer-attempt', function(e) {
	  e.preventDefault();
	  var button = this;
	  var $button = $(button);
	  var id = snFieldLeadId(button);
	  var lead = getLeadById(id) || {};
	  var distributionId = String($button.attr('data-distribution-id') || lead.distribution_item_id || '').replace(/\D+/g, '');
	  var sourceKind = String($button.attr('data-source-kind') || lead.source_kind || (distributionId ? 'v4_distribution_item' : 'legacy_lead'));
	  var sourceId = distributionId || String(id).replace(/\D+/g, '');
	  var attemptToken = String($button.attr('data-attempt-token') || '');
	  if (!attemptToken) {
		attemptToken = 'na-' + String(Date.now()) + '-' + Math.random().toString(36).slice(2, 12);
		$button.attr('data-attempt-token', attemptToken);
	  }
	  if (!sourceId) {
		setLeadSaveState(id, 'error', 'شناسه این شماره برای ثبت تماس معتبر نیست.');
		return;
	  }
	  $button.prop('disabled', true).text('در حال ثبت...');
	  setLeadSaveState(id, 'saving', 'در حال ثبت تماس بی‌پاسخ...');
	  $.ajax({
		url: ajax,
		type: 'POST',
		dataType: 'json',
		data: {
		  action: 'sn_seller_flow_no_answer_attempt',
		  nonce: nonce,
		  source_kind: sourceKind,
		  source_id: sourceId,
		  distribution_item_id: distributionId,
		  attempt_token: attemptToken
		}
	  }).done(function(r) {
		if (!r || !r.success) {
		  setLeadSaveState(id, 'error', (r && r.message) ? r.message : 'ثبت تماس انجام نشد.');
		  $button.prop('disabled', false).text('ثبت تماس بی‌پاسخ بعدی');
		  return;
		}
		var state = $.extend({}, r, { lead_status: 'جواب نداده', seller_flow_status: 'no_answer' });
		$button.removeAttr('data-attempt-token');
		syncLeadToMemory(id, state);
		$('.sn-no-answer-count', '.sn-flow-no-answer[data-id="' + id + '"]').text(String(r.no_answer_attempts || 0));
		setLeadSaveState(id, 'saved', r.message || 'تماس بی‌پاسخ ثبت شد.');
		setTimeout(function(){ loadLeads(); }, 350);
	  }).fail(function(xhr) {
		var message = xhr && xhr.responseJSON && xhr.responseJSON.message ? xhr.responseJSON.message : 'خطا در ارتباط با سرور';
		setLeadSaveState(id, 'error', message);
		$button.prop('disabled', false).text('ثبت تماس بی‌پاسخ بعدی');
	  });
	});

    $(document).on('keydown', '.sn-lead-note, .sn-cust-name', function (e) {
      if (e.key === 'Enter' && (!$(this).hasClass('sn-lead-note') || e.ctrlKey || e.metaKey)) {
        e.preventDefault();
        snSaveFromField(this, $(this).hasClass('sn-lead-note') ? 'note' : 'customer_name', true);
      }
    });

    // ذخیره دستی: مسیر قطعی برای وقتی کاربر نمی‌خواهد منتظر blur/change بماند.
    $(document).on('click', '.sn-save-lead-editor', function(e) {
      e.preventDefault();
      var id = snFieldLeadId(this);
      if (!id) return;
      saveLeadData(id, { force: true, changedField: 'manual_save', savingText: 'در حال ذخیره همه تغییرات...', savedText: 'همه تغییرات ذخیره شد' });
    });

    // وقتی فروشنده از فرم باز خارج می‌شود یا تب را عوض می‌کند، آخرین snapshot فرم ذخیره شود؛ بدون auto-save حین تایپ.
    $(document).on('mousedown.snSellerSaveBeforeTab', '#sn-seller-panel .sn-tab, #sn-seller-panel .sn-use-lead', function() {
      $('.sn-lead-expand-row:visible').each(function(){
        var id = String($(this).data('id') || '');
        if (id) {
          try { saveLeadData(id, { force: true, changedField: 'leave_editor', savingText: 'در حال ذخیره تغییرات...', savedText: 'تغییرات ذخیره شد' }); } catch(e) {}
        }
      });
    });

    // دکمه صدور پیش‌فاکتور در لیست شماره‌ها
    $(document).on('click', '.sn-use-lead', function () {
	  if (!sellerCanIssueInvoice || !$('#sn-tab-new-invoice').length) return;
      var phone = String($(this).data('phone') || '');
      var id    = String($(this).data('id')    || '');
      var lead  = getLeadById(id);

      if (saveTimers[id]) {
        clearTimeout(saveTimers[id]);
      }

      var currentFormSnapshot = null;
      if ($('.sn-cust-name[data-id="' + id + '"]').length) {
        currentFormSnapshot = {
          customer_name: $('.sn-cust-name[data-id="' + id + '"]').val() || '',
          phone: phone,
          province: $('.sn-cust-prov[data-id="' + id + '"]').val() || '',
          city: $('.sn-cust-city[data-id="' + id + '"]').val() || '',
          sales_prediction: $('.sn-cust-pred[data-id="' + id + '"]').val() || '',
          note: $('.sn-lead-note[data-id="' + id + '"]').val() || '',
          lead_status: $('select.sn-lead-status-select[data-id="' + id + '"]').val() || ''
        };
      }

      var openInvoiceForm = function() {
        var latestLead = $.extend({}, lead || {}, getLeadById(id) || {}, currentFormSnapshot || {});
        $('#sn-cust-name').val(latestLead.customer_name || '');
        $('#sn-cust-phone').val(latestLead.phone || phone);
        $('#sn-cust-prov').val(latestLead.province || '');
        var $cityField = $('#sn-cust-city');
        if ($cityField.is('select')) {
          $cityField.html(snBuildCityOptions(latestLead.province || '', latestLead.city || ''));
          $cityField.val((latestLead.city || '').trim());
          if (!latestLead.city) {
            $cityField.val('');
          }
        } else {
          $cityField.val(latestLead.city || '');
        }

        if ($('#sn-lead-select option[value="' + id + '"]').length === 0) {
          $('#sn-lead-select').append('<option value="' + id + '" data-phone="' + (latestLead.phone || phone) + '">' + (latestLead.phone || phone) + '</option>');
        }
        $('#sn-lead-select').val(id);
        $('#sn-product').val('');

        var $tabBtn = $('.sn-panel#sn-seller-panel .sn-tab[data-tab="new-invoice"]');
        if ($tabBtn.length) {
          $tabBtn.trigger('click');
        } else {
          $('.sn-panel#sn-seller-panel .sn-tab').removeClass('active');
          $('.sn-panel#sn-seller-panel .sn-tab-content').removeClass('active').hide();
          $('.sn-panel#sn-seller-panel .sn-tab[data-tab="new-invoice"]').addClass('active');
          $('#sn-tab-new-invoice').addClass('active').show();
        }

        setTimeout(function() {
          var $form = $('#sn-tab-new-invoice');
          if ($form.length) {
            $('html,body').animate({ scrollTop: $form.offset().top - 80 }, 400);
          }
        }, 150);
      };

      // صدور پیش‌فاکتور نباید درگیر صف/ذخیره قبلی شود؛ با snapshot فعلی فرم مستقیم به تب صدور فاکتور می‌رود.
      openInvoiceForm();
    });

    // autofill فرم فاکتور از lead انتخاب‌شده
    $(document).on('change', '#sn-lead-select', function () {
      var selectedId = $(this).val();
      var lead = getLeadById(selectedId);
      var phone = $(this).find(':selected').data('phone');
      if (lead) {
        $('#sn-cust-name').val(lead.customer_name || '');
        $('#sn-cust-phone').val(lead.phone || phone || '');
        $('#sn-cust-prov').val(lead.province || '');
        var $cityField = $('#sn-cust-city');
        if ($cityField.is('select')) {
          $cityField.html(snBuildCityOptions(lead.province || '', lead.city || ''));
          $cityField.val((lead.city || '').trim());
          if (!lead.city) {
            $cityField.val('');
          }
        } else {
          $cityField.val(lead.city || '');
        }
      } else if (phone) {
        $('#sn-cust-phone').val(phone);
      }
    });

    $(document).on('change', '#sn-cust-prov', function () {
      var prov = $(this).val() || '';
      var $cityField = $('#sn-cust-city');
      if ($cityField.is('select')) {
        var currentCity = $cityField.val() || '';
        $cityField.html(snBuildCityOptions(prov, currentCity));
        if (currentCity) {
          $cityField.val(currentCity);
        }
      }
    });

    function loadInvoices() {
      $.post(ajax, { action: 'sn_seller_invoices', nonce: nonce, summary_only: 1 }, function (res) {
        sellerInvoiceSummary = (res && res.summary) || {};
        sellerPaidAmount = Number((res && res.paid_amount) || 0);
        sellerInvoices = [];
        renderSellerKpis();
        if ($('#sn-tab-invoices').hasClass('active') && typeof window.snLoadSellerInvoices === 'function') {
          window.snLoadSellerInvoices();
        }
      });
    }

    // Create invoice
    $(document).on('click', '#sn-create-invoice', function () {
      if (window.snSellerInvoiceAuthoritativeHandler) { return; }
      const $btn = $(this);
      const lead_id = $('#sn-lead-select').val();
      const selectedLead = lead_id ? getLeadById(lead_id) : null;
      const name    = ($('#sn-cust-name').val() || '').trim() || (selectedLead && selectedLead.customer_name ? selectedLead.customer_name : '');
      const phone   = ($('#sn-cust-phone').val() || '').trim() || (selectedLead && selectedLead.phone ? selectedLead.phone : '');
      let prov      = $('#sn-cust-prov').val() || (selectedLead && selectedLead.province ? selectedLead.province : '');
      let city      = ($('#sn-cust-city').val() || '').trim() || (selectedLead && selectedLead.city ? String(selectedLead.city).trim() : '');
      const prod    = $('#sn-product').val();

      if (!city && lead_id && $('.sn-cust-city[data-id="' + lead_id + '"]').length) {
        city = ($('.sn-cust-city[data-id="' + lead_id + '"]').val() || '').trim();
      }
      if (!prov && lead_id && $('.sn-cust-prov[data-id="' + lead_id + '"]').length) {
        prov = $('.sn-cust-prov[data-id="' + lead_id + '"]').val() || '';
      }

      if (!name || !phone || !prod) {
        showNotice('#sn-invoice-notice', 'لطفاً نام، موبایل و محصول را وارد کنید.', 'error');
        return;
      }

      $btn.prop('disabled', true).text('در حال صدور...');
      
      // XHR مستقیم — jQuery ممکنه response خراب رو parse نکنه
      var xhrObj = new XMLHttpRequest();
      xhrObj.open('POST', ajax, true);
      xhrObj.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded; charset=UTF-8');
      xhrObj.onload = function() {
        $btn.prop('disabled', false).text('صدور پیش‌فاکتور و ارسال پیامک');
        var raw = xhrObj.responseText || '';
        // پیدا کردن JSON حتی اگه قبلش garbage باشه
        var res = null;
        var m = raw.match(/\{[\s\S]*"success"[\s\S]*\}/);
        if (m) { try { res = JSON.parse(m[0]); } catch(e) {} }
        if (!res) { try { res = JSON.parse(raw); } catch(e) {} }
        
        if (res && res.success) {
          var payload = res.data || res;
          var smsMsg = payload.sms_message ? ' ' + payload.sms_message : '';
          $('#sn-cust-name,#sn-cust-phone,#sn-cust-city').val('');
          $('#sn-product,#sn-lead-select,#sn-cust-prov').val('');
          showNotice('#sn-invoice-notice',
            'پیش‌فاکتور با موفقیت صادر شد. کد: ' + (payload.invoice_code || payload.code || '') + smsMsg,
            'success');
          loadLeads();
          loadInvoices();
          switchToInvoicesTabAfterCreate(payload.invoice_code || payload.code || '');
        } else {
          var errPayload = (res && res.data) ? res.data : (res || {});
          var msg = errPayload.message || (res && res.message) || ('کد HTTP: ' + xhrObj.status);
          showNotice('#sn-invoice-notice', '❌ ' + msg, 'error');
          console.error('Invoice fail. Raw:', raw.substring(0,400));
        }
      };
      xhrObj.onerror = function() {
        $btn.prop('disabled', false).text('صدور پیش‌فاکتور و ارسال پیامک');
        showNotice('#sn-invoice-notice', '❌ خطای شبکه', 'error');
      };
      xhrObj.send('action=sn_create_invoice&nonce=' + encodeURIComponent(nonce)
        + '&customer_name=' + encodeURIComponent(name)
        + '&customer_phone=' + encodeURIComponent(phone)
        + '&province=' + encodeURIComponent(prov)
        + '&city=' + encodeURIComponent(city)
        + '&product_id=' + encodeURIComponent(prod)
        + '&lead_id=' + encodeURIComponent(lead_id||'')
      );
    });
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

  function switchToInvoicesTabAfterCreate(code) {
    setTimeout(function () {
      var $tab = $('#sn-seller-panel .sn-tab[data-tab="invoices"]');
      if ($tab.length) { $tab.trigger('click'); }
      window.snLastCreatedInvoiceCode = code || '';
      $('#sn-seller-invoice-search').val('');
      var $pre = $('.sn-invoice-status-tabs .sn-subtab[data-status="pre_invoice"]');
      if ($pre.length) { $pre.trigger('click'); }
    }, 1800);
  }

}(jQuery));

/* SN 1.0.8 complete workflow-lite additions */
(function($){
  'use strict';
  if (!$('#sn-seller-panel').length) return;
  var ajax = (window.snAjax||window.snData||{}).ajaxurl;
  var nonce = (window.snAjax||window.snData||{}).nonce;
  function money(v){ try { return Number(v||0).toLocaleString('fa-IR') + ' تومان'; } catch(e){ return (v||0)+' تومان'; } }
  function esc(v){ return String(v==null?'':v).replace(/[&<>"']/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#039;'}[c];}); }
  function showNotice(selector,msg,type){
    $(selector).html('<div class="sn-notice sn-'+esc(type||'info')+'">'+esc(msg||'')+'</div>');
    setTimeout(function(){ $(selector).empty(); },6000);
  }
  function snToEnDigits147(value){return String(value||'').replace(/[۰-۹]/g,function(d){return '۰۱۲۳۴۵۶۷۸۹'.indexOf(d);}).replace(/[٠-٩]/g,function(d){return '٠١٢٣٤٥٦٧٨٩'.indexOf(d);});}
  function snFormatNumber147(value){var n=Number(value||0);try{return n.toLocaleString('fa-IR');}catch(e){return String(n);}}
  function snFormatMoney147(value){return snFormatNumber147(value)+' تومان';}
  function switchToInvoicesTabAfterCreate(code){
    setTimeout(function(){
      var $tab=$('#sn-seller-panel .sn-tab[data-tab="invoices"]');
      if($tab.length){ $tab.trigger('click'); }
      window.snLastCreatedInvoiceCode = code || '';
      $('#sn-seller-invoice-search').val('');
      var $pre=$('.sn-invoice-status-tabs .sn-subtab[data-status="pre_invoice"]');
      if($pre.length){ $pre.trigger('click'); }
    },1800);
  }
  function productOptions(){ return $('#sn-product').html() || ''; }
  function maxInvoiceProducts(){ var v=Number($('#sn-seller-panel').attr('data-sn-max-products')||$('#sn-add-product-row').attr('data-max-products')||0); return isFinite(v)&&v>0?Math.floor(v):0; }
  function recalcProducts(){ var total=0; $('.sn-product-row').each(function(){ var $r=$(this), price=Number($r.find('option:selected').data('price')||0), q=Math.max(1,Number($r.find('.sn-product-qty').val()||1)); total += price*q; }); $('#sn-products-total').text('جمع: '+money(total)); $('.sn-remove-product').toggle($('.sn-product-row').length>1); var max=maxInvoiceProducts(),count=$('#sn-products-multi .sn-product-row').length; $('#sn-add-product-row').prop('disabled',max>0&&count>=max).attr('title',max>0&&count>=max?'حداکثر تعداد محصول انتخاب شده است':''); }
  $(document).off('click','#sn-add-product-row').on('click.snProductLimit149','#sn-add-product-row',function(){ var max=maxInvoiceProducts(),count=$('#sn-products-multi .sn-product-row').length;if(max>0&&count>=max){$('#sn-invoice-notice').html('<div class="sn-notice sn-error">حداکثر '+max+' محصول در هر پیش‌فاکتور قابل انتخاب است.</div>');recalcProducts();return;} $('#sn-products-multi').append('<div class="sn-product-row"><select class="sn-product-select">'+productOptions()+'</select><input type="number" class="sn-product-qty" min="1" value="1"><button type="button" class="sn-btn sn-btn-ghost sn-remove-product">حذف</button></div>'); recalcProducts(); });
  $(document).off('click','.sn-remove-product').on('click.snProductLimit149','.sn-remove-product',function(){ $(this).closest('.sn-product-row').remove(); recalcProducts(); });
  $(document).off('change input','.sn-product-select,.sn-product-qty').on('change.snProductLimit149 input.snProductLimit149','.sn-product-select,.sn-product-qty',recalcProducts);
  $(recalcProducts);

  var activeInvoiceTab = 'all';
  var sellerInvoiceXhr = null;
  var sellerInvoiceRequestKey = '';
	var manualPaymentInvoiceCode = '';
	function ensureSellerManualPaymentModal(){
	  if($('#sn-seller-manual-payment-modal').length) return;
	  $('body').append('<div id="sn-seller-manual-payment-modal" class="sn-modal sn-lite-modal sn-payment-entry-modal" aria-hidden="true" style="display:none">'+
		'<div class="sn-modal-backdrop sn-seller-manual-payment-close"></div><div class="sn-modal-card sn-payment-entry-card" role="dialog" aria-modal="true" aria-label="ثبت اطلاعات واریز">'+
		'<div class="sn-modal-head sn-payment-entry-head"><div class="sn-payment-entry-title"><span class="sn-payment-entry-icon">↙</span><div><h3>ثبت اطلاعات واریز</h3><p>اطلاعات پرداخت مشتری را دقیق وارد کنید</p></div></div><button type="button" class="sn-modal-x sn-seller-manual-payment-close" aria-label="بستن">×</button></div>'+
		'<div class="sn-modal-body"><div class="sn-payment-entry-note">مبلغ مرحلهٔ جاری، اطلاعات کارت، زمان تهران و تصویر فیش را همین‌جا ثبت کنید.</div><div class="sn-payment-entry-grid">'+
		'<label><span>مبلغ واریز این مرحله (تومان)</span><input id="sn-seller-manual-amount" inputmode="numeric" readonly></label>'+
		'<label><span>۴ رقم آخر کارت مبدا</span><input id="sn-seller-manual-card-from" inputmode="numeric" maxlength="4" autocomplete="off" placeholder="مثلاً ۱۲۳۴"></label>'+
		'<label><span>۴ رقم آخر کارت مقصد (اختیاری)</span><input id="sn-seller-manual-card-to" inputmode="numeric" maxlength="4" autocomplete="off" placeholder="اختیاری؛ مثلاً ۵۶۷۸"></label>'+
		'<label><span>تاریخ شمسی واریز</span><input id="sn-seller-manual-paid-date" class="sn-jalali-date" placeholder="۱۴۰۵/۰۷/۰۱"><button type="button" class="sn-btn sn-btn-sm sn-payment-today" data-date="#sn-seller-manual-paid-date" data-hour="#sn-seller-manual-hour" data-minute="#sn-seller-manual-minute">امروز تهران</button></label>'+
		'<label><span>ساعت تهران (۲۴ ساعته)</span><span class="sn-payment-time-select"><select id="sn-seller-manual-hour" aria-label="ساعت"></select><span aria-hidden="true">:</span><select id="sn-seller-manual-minute" aria-label="دقیقه"></select></span></label><label class="sn-payment-receipt-field"><span>تصویر فیش (اختیاری)</span><input id="sn-seller-manual-receipt" type="file" accept="image/*,.pdf,application/pdf"></label></div><div id="sn-seller-manual-payment-msg" class="sn-payment-entry-message"></div>'+
		'<div class="sn-modal-actions sn-payment-entry-actions"><button type="button" class="sn-btn sn-btn-secondary sn-seller-manual-payment-close">انصراف</button><button type="button" class="sn-btn sn-btn-primary" id="sn-seller-manual-payment-submit">ثبت اطلاعات واریز</button></div></div></div></div>');
	}
	$(document).on('click','.sn-seller-manual-payment-open',function(){
	  manualPaymentInvoiceCode=String($(this).data('invoice-code')||''); ensureSellerManualPaymentModal();
	  $('#sn-seller-manual-card-from,#sn-seller-manual-card-to,#sn-seller-manual-paid-date,#sn-seller-manual-receipt').val(''); $('#sn-seller-manual-amount').val(String($(this).data('amount')||'')); $('#sn-seller-manual-payment-msg').empty();
	  if(window.snPaymentTimeInit)window.snPaymentTimeInit('#sn-seller-manual-hour','#sn-seller-manual-minute');
	  $('#sn-seller-manual-payment-modal').fadeIn(120).attr('aria-hidden','false');
      var modal=document.getElementById('sn-seller-manual-payment-modal');
      window.snFinancialContext(modal, [['مشتری',this.getAttribute('data-customer-name')],['کد فاکتور',manualPaymentInvoiceCode],['وضعیت',this.getAttribute('data-status-label')],['مبلغ مرحله (تومان)',this.getAttribute('data-amount')]]);
      window.snFinancialDialog.open(modal, this, function(){ $('#sn-seller-manual-payment-modal').hide().attr('aria-hidden','true'); }, '#sn-seller-manual-payment-submit');
	});
	$(document).on('click','.sn-seller-manual-payment-close',function(){ $('#sn-seller-manual-payment-modal').fadeOut(120).attr('aria-hidden','true'); });
	$(document).on('click','#sn-seller-manual-payment-submit',function(){
	  var from=snToEnDigits147($('#sn-seller-manual-card-from').val()).replace(/\D+/g,''), to=snToEnDigits147($('#sn-seller-manual-card-to').val()).replace(/\D+/g,''), date=snToEnDigits147($('#sn-seller-manual-paid-date').val()).trim(), time=String($('#sn-seller-manual-hour').val()||'')+':'+String($('#sn-seller-manual-minute').val()||'');
	  var $msg=$('#sn-seller-manual-payment-msg'), $btn=$(this);
	  if(!/^\d{4}$/.test(from) || !/^\d{4}\/\d{2}\/\d{2}$/.test(date) || !/^\d{2}:\d{2}$/.test(time)){ $msg.html('<div class="sn-notice sn-error">۴ رقم مبدا، تاریخ و ساعت را کامل وارد کنید.</div>'); return; }
	  if(to!=='' && !/^\d{4}$/.test(to)){ $msg.html('<div class="sn-notice sn-error">۴ رقم کارت مقصد اختیاری است؛ در صورت ورود باید ۴ رقم باشد.</div>'); return; }
	  $btn.prop('disabled',true).text('در حال ثبت...');
	  var form=new FormData();form.append('action','sn_submit_manual_payment');form.append('nonce',nonce);form.append('invoice_code',manualPaymentInvoiceCode);form.append('card_from',from);form.append('card_to',to);form.append('paid_at',date+' '+time);form.append('amount',snToEnDigits147($('#sn-seller-manual-amount').val()));
	  var receipt=$('#sn-seller-manual-receipt')[0].files[0];if(receipt)form.append('receipt',receipt);
	  $.ajax({url:ajax,type:'POST',data:form,processData:false,contentType:false,dataType:'json'}).done(function(res){
		var payload=(res&&res.data&&typeof res.data==='object')?res.data:res;
		if(res&&res.success){ $msg.html('<div class="sn-notice sn-success">'+esc((payload&&payload.message)||res.message||'ثبت شد')+'</div>'); setTimeout(function(){ $('#sn-seller-manual-payment-modal').fadeOut(120); loadSellerInvoices(); },650); }
		else $msg.html('<div class="sn-notice sn-error">'+esc((payload&&payload.message)||(res&&res.message)||'ثبت انجام نشد')+'</div>');
	  }).fail(function(){ $msg.html('<div class="sn-notice sn-error">خطا در ارتباط با سرور</div>'); }).always(function(){ $btn.prop('disabled',false).text('ثبت اطلاعات واریز'); });
	});
  function loadSellerInvoices(){
    $('#sn-invoices-loading').show(); $('#sn-invoices-list').html('<div class="sn-loading">در حال بارگذاری...</div>');
    var payload = {action:'sn_seller_invoices',nonce:nonce,tab:activeInvoiceTab,limit:30,q:($('#sn-seller-invoice-search').val()||'')};
    var requestKey = JSON.stringify(payload);
    if (sellerInvoiceXhr && sellerInvoiceXhr.readyState !== 4 && sellerInvoiceRequestKey === requestKey) {
      return;
    }
    if (sellerInvoiceXhr && sellerInvoiceXhr.readyState !== 4) {
      sellerInvoiceXhr.abort();
    }
    sellerInvoiceRequestKey = requestKey;
    sellerInvoiceXhr = $.post(ajax,payload,function(res){
      $('#sn-invoices-loading').hide();
      var rows = (res && (res.invoices || res.items)) || [];
      var total = Number((res && res.total) || rows.length || 0);
      var query = ($('#sn-seller-invoice-search').val()||'').trim();
      if(!res || !res.success || !rows.length){
        var emptyMsg = query ? 'موردی برای این جستجو پیدا نشد. جستجو را پاک کنید تا همه ردیف‌های این تب دیده شوند.' : 'موردی در این تب وجود ندارد.';
        $('#sn-invoices-list').html('<p class="sn-notice">'+esc(emptyMsg)+'</p>');
        return;
      }
      var meta = '<div class="sn-helper-card sn-invoice-list-meta"><strong>نمایش '+esc(rows.length)+' از '+esc(total)+'</strong>'+(query?'<p class="sn-note">جستجو فعال است: '+esc(query)+' <button type="button" class="sn-btn sn-btn-sm sn-clear-invoice-search">پاک کردن جستجو</button></p>':'<p class="sn-note">برای دیدن همه ردیف‌های این وضعیت، جستجو را خالی بگذارید.</p>')+'</div>';
      var html='<div class="sn-table-wrap"><table class="sn-table sn-seller-invoices-table"><thead><tr><th>کد</th><th>مشتری</th><th>موبایل</th><th>محصول‌ها</th><th>مبلغ کل / پرداخت‌شده / مانده</th><th>وضعیت</th><th>نتیجه تبدیل اعتبارسنجی</th><th>دلیل رد/عملیات</th></tr></thead><tbody>';
      rows.forEach(function(inv){
        var contextAttrs=' data-customer-name="'+esc(inv.customer_name||'')+'" data-invoice-code="'+esc(inv.invoice_code||'')+'" data-status-label="'+esc(inv.status_label||'')+'"';
        var ops='';
        if(String(inv.status)==='rejected') ops='<div class="sn-reject-reason"><strong>دلیل رد:</strong> '+esc(inv.financial_reject_reason||inv.rejected_reason||'بدون دلیل ثبت‌شده')+'</div><button type="button" class="sn-btn sn-btn-sm sn-resend-financial"'+contextAttrs+' data-amount="'+esc((inv.payment_summary||{}).due||inv.current_due_amount||'')+'" data-id="'+esc(inv.id)+'">ارسال مجدد برای بررسی مالی</button>';
        if(String(inv.status)==='cancelled' && (String(inv.repeat_action_status||'')==='cancelled_by_finance' || String(inv.financial_return_state||'').indexOf('cancelled_by_finance')===0)) ops='<div class="sn-reject-reason"><strong>لغو شده توسط مالی</strong><br>'+esc(inv.repeat_action_cancel_reason||inv.payment_archive_reason||'بدون توضیح ثبت‌شده')+'</div>';
        if(String(inv.status)==='recontact_requested' || String(inv.invoice_status)==='recontact_requested') ops += '<div class="sn-recontact-actions"><select class="sn-recontact-next-status" data-id="'+esc(inv.id)+'"><option value="pending">بازگشت به پیش‌فاکتور/پرداخت</option><option value="callback_scheduled">تماس مجدد زمان‌بندی شد</option><option value="customer_followed">پیگیری انجام شد</option><option value="pending_payment">آماده پرداخت</option></select><button type="button" class="sn-btn sn-btn-sm sn-mark-recontact-status" data-id="'+esc(inv.id)+'">ثبت وضعیت</button></div>';
		if(inv.is_conversion_payment) ops+='<span class="sn-badge sn-status">فاکتور تبدیل</span>';
		if(inv.can_submit_manual_payment && String(inv.status)!=='rejected') ops+='<button type="button" class="sn-btn sn-btn-sm sn-btn-secondary sn-seller-manual-payment-open"'+contextAttrs+' data-amount="'+esc((inv.payment_summary||{}).due||inv.current_due_amount||'')+'">ثبت واریز و فیش</button>';
		if(Number((inv.payment_summary||{}).paid||0)===0 && Number((inv.payment_summary||{}).stage_no||1)===1) ops+='<button type="button" class="sn-btn sn-btn-sm sn-btn-secondary sn-edit-invoice"'+contextAttrs+' data-invoice-id="'+esc(inv.id||'')+'">ویرایش پیش از پرداخت</button>';
		ops+='<div class="sn-invoice-sms-action"><button type="button" class="sn-btn sn-btn-sm sn-btn-secondary sn-resend-invoice-sms" data-invoice-id="'+esc(inv.id||'')+'">ارسال مجدد لینک پرداخت</button><small class="sn-resend-invoice-sms-msg" aria-live="polite"></small><button type="button" class="sn-btn sn-btn-sm sn-btn-secondary sn-copy-invoice-link" data-invoice-id="'+esc(inv.id||'')+'">کپی لینک پرداخت</button><small class="sn-copy-invoice-link-msg" aria-live="polite"></small></div>';
		ops+='<button type="button" class="sn-btn sn-btn-sm sn-btn-secondary sn-invoice-finance-history" data-invoice-id="'+esc(inv.id||'')+'">وضعیت و تاریخچه مالی</button>';
		if(inv.project && inv.project.membership_id) ops += '<div class="sn-project-invoice-action"><span class="sn-project-tag">'+esc(inv.project.tag||'اشتراک · بیاوین')+'</span><button type="button" class="sn-btn sn-btn-sm sn-btn-primary sn-project-chat-btn" data-membership="'+esc(inv.project.membership_id)+'" data-item="'+esc((inv.project.items&&inv.project.items[0]&&inv.project.items[0].id)||'')+'">چت با بیاوین</button></div>';
        var rowClass = (window.snLastCreatedInvoiceCode && String(inv.invoice_code) === String(window.snLastCreatedInvoiceCode)) ? ' class="sn-highlight-row"' : '';
        var issuedTag = (!inv.issued_by_is_self && inv.issued_by_label) ? '<br><span class="sn-badge sn-issued-by-badge">صادرشده توسط '+esc(inv.issued_by_label)+'</span>' : '';
        var conversion = inv.dot_conversion_result || null;
        var conversionCell = '<span class="sn-dot-seller-result-empty">—</span>';
        if(conversion){
          var sold = !!conversion.subscription_sold;
          var stateClass = sold ? ' is-sold' : (String(conversion.sale_state||'')==='not_sold' ? ' is-not-sold' : ' is-progress');
          var stateIcon = sold ? '✓' : (String(conversion.sale_state||'')==='not_sold' ? '×' : '…');
          conversionCell = '<div class="sn-dot-seller-result'+stateClass+'"><span class="sn-dot-seller-result-status"><i aria-hidden="true">'+stateIcon+'</i> '+esc(conversion.sale_state_label||'در جریان تبدیل')+'</span><small class="sn-dot-seller-result-label">وضعیت تبدیل: '+esc(conversion.conversion_status_label||'نامشخص')+'</small><strong>'+esc(conversion.option_title||'هنوز انتخاب نشده')+'</strong>'+(Number(conversion.paid_amount||0)>0?'<small>پرداخت‌شده: '+esc(conversion.paid_amount_fmt||money(conversion.paid_amount))+(Number(conversion.total_amount||0)>0?' از '+esc(conversion.total_amount_fmt||money(conversion.total_amount)):'')+'</small>':'')+'</div>';
        }
        var amounts=inv.payment_summary||{};
        var amountCell='<div class="sn-stage-amounts"><span>کل: '+esc(money(amounts.total||0))+'</span><span>پرداخت‌شده: '+esc(money(amounts.paid||0))+'</span><strong>مانده: '+esc(money(amounts.remaining||0))+'</strong><small>مرحله '+esc(amounts.stage_no||1)+' · قابل پرداخت فعلی: '+esc(money(amounts.due||0))+'</small></div>';
        html+='<tr'+rowClass+'><td><code>'+esc(inv.invoice_code)+'</code></td><td>'+esc(inv.customer_name)+'</td><td>'+esc(inv.customer_phone)+'</td><td>'+esc(inv.product_name||inv.product_id)+'</td><td>'+amountCell+'</td><td><span class="sn-status sn-status-'+esc(inv.status)+'">'+esc(inv.status_label||inv.status)+'</span>'+issuedTag+'</td><td>'+conversionCell+'</td><td>'+ops+'</td></tr>';
      });
      html+='</tbody></table></div>'; $('#sn-invoices-list').html(meta+html);
    }).always(function(){ sellerInvoiceXhr = null; sellerInvoiceRequestKey = ''; });
  }
  window.snLoadSellerInvoices = loadSellerInvoices;
  $(document).on('click','.sn-invoice-status-tabs .sn-subtab',function(){ $('.sn-invoice-status-tabs .sn-subtab').removeClass('active'); $(this).addClass('active'); activeInvoiceTab=$(this).data('status')||'all'; loadSellerInvoices(); });
  $(document).on('click','#sn-seller-panel .sn-tab[data-tab="invoices"]',function(){ setTimeout(loadSellerInvoices,20); });
  $(document).on('input','#sn-seller-invoice-search',function(){ clearTimeout(window.snSellerInvoiceSearchTimer); window.snSellerInvoiceSearchTimer=setTimeout(loadSellerInvoices,350); });
  $(document).on('click','.sn-clear-invoice-search',function(){ $('#sn-seller-invoice-search').val(''); loadSellerInvoices(); });
  $(document).on('click','.sn-mark-recontact-status',function(){
    var $btn=$(this), id=$btn.data('id'), st=$btn.closest('.sn-recontact-actions').find('.sn-recontact-next-status').val() || 'pending';
    $btn.prop('disabled',true).text('در حال ثبت...');
    $.post(ajax,{action:'sn_seller_mark_recontact_status',nonce:nonce,invoice_id:id,new_status:st},function(res){
      if(res&&res.success){ showNotice('#sn-invoice-notice', (res.message||'وضعیت ثبت شد'), 'success'); loadSellerInvoices(); }
      else { showNotice('#sn-invoice-notice', (res&&res.message)||'خطا در ثبت وضعیت', 'error'); }
    }).fail(function(){ showNotice('#sn-invoice-notice','خطای شبکه در ثبت وضعیت ارتباط مجدد','error'); }).always(function(){ $btn.prop('disabled',false).text('ثبت وضعیت'); });
  });
  var currentResendInvoiceId = 0;
  function ensureResendFinancialModal(){
    if($('#sn-seller-resend-modal').length) return;
    var html = ''+
      '<div id="sn-seller-resend-modal" class="sn-modal sn-lite-modal sn-seller-resend-modal" aria-hidden="true">'+
        '<div class="sn-modal-backdrop sn-resend-close"></div>'+
        '<div class="sn-modal-card sn-seller-resend-card" role="dialog" aria-modal="true" aria-labelledby="sn-resend-title">'+
          '<div class="sn-modal-head sn-resend-head">'+
            '<div><span class="sn-modal-kicker">فاکتور برگشتی</span><h3 id="sn-resend-title">ارسال مجدد برای بررسی مالی</h3></div>'+
            '<button type="button" class="sn-modal-x sn-resend-close" aria-label="بستن">×</button>'+
          '</div>'+
          '<div class="sn-modal-body sn-resend-body">'+
            '<div class="sn-resend-help">اطلاعات اصلاح‌شده واریز را وارد کن تا دوباره برای مالی ارسال شود.</div>'+
            '<div class="sn-resend-form-grid">'+
              '<label class="sn-field sn-full"><span>یادداشت برای مالی</span><textarea id="sn-resend-note" rows="3" placeholder="مثلاً: بررسی مجدد / فیش اصلاح شد / مبلغ واریزی اصلاح شد"></textarea></label>'+
              '<label class="sn-field"><span>۴ رقم کارت مبدا</span><input id="sn-resend-card-from" inputmode="numeric" maxlength="4" placeholder="اختیاری"></label>'+
			  '<label class="sn-field"><span>۴ رقم آخر کارت مقصد</span><input id="sn-resend-card-to" inputmode="numeric" maxlength="4" autocomplete="off" placeholder="اختیاری؛ مثلاً ۵۶۷۸"></label>'+
              '<label class="sn-field"><span>مبلغ اصلاحی</span><input id="sn-resend-amount" inputmode="numeric" placeholder="اختیاری"></label>'+
              '<label class="sn-field"><span>تاریخ شمسی واریز</span><input id="sn-resend-paid-date" class="sn-jalali-date" placeholder="۱۴۰۵/۰۷/۰۱"><button type="button" class="sn-btn sn-btn-sm sn-payment-today" data-date="#sn-resend-paid-date" data-hour="#sn-resend-hour" data-minute="#sn-resend-minute">امروز تهران</button></label>'+
			  '<label class="sn-field"><span>ساعت تهران (۲۴ ساعته)</span><span class="sn-payment-time-select"><select id="sn-resend-hour" aria-label="ساعت"></select><span aria-hidden="true">:</span><select id="sn-resend-minute" aria-label="دقیقه"></select></span></label>'+
			  '<label class="sn-field"><span>فیش اصلاح‌شده (اختیاری)</span><input id="sn-resend-receipt" type="file" accept="image/*,.pdf,application/pdf"></label>'+
            '</div>'+
            '<div id="sn-resend-msg" class="sn-resend-msg"></div>'+
            '<div class="sn-modal-actions sn-resend-actions"><button type="button" class="sn-btn sn-btn-secondary sn-resend-close">انصراف</button><button type="button" class="sn-btn sn-btn-primary" id="sn-resend-submit">ارسال به مالی</button></div>'+
          '</div>'+
        '</div>'+
      '</div>';
    $('body').append(html);
  }
  $(document).on('click','.sn-resend-financial',function(){
    currentResendInvoiceId=$(this).data('id')||0;
    ensureResendFinancialModal();
    $('#sn-resend-note,#sn-resend-card-from,#sn-resend-card-to,#sn-resend-amount,#sn-resend-paid-date,#sn-resend-receipt').val('');
	if(window.snPaymentTimeInit)window.snPaymentTimeInit('#sn-resend-hour','#sn-resend-minute');
    $('#sn-resend-msg').empty();
    $('#sn-seller-resend-modal').fadeIn(120).attr('aria-hidden','false');
    window.snFinancialContext(document.getElementById('sn-seller-resend-modal'), [['مشتری',this.getAttribute('data-customer-name')],['کد فاکتور',this.getAttribute('data-invoice-code')],['شناسه',currentResendInvoiceId],['وضعیت',this.getAttribute('data-status-label')],['مبلغ مرحله (تومان)',this.getAttribute('data-amount')]]);
    window.snFinancialDialog.open(document.getElementById('sn-seller-resend-modal'), this, function(){ $('#sn-seller-resend-modal').hide().attr('aria-hidden','true'); }, '#sn-resend-submit');
    setTimeout(function(){ $('#sn-resend-note').focus(); },120);
  });
  $(document).on('click','.sn-resend-close',function(){ $('#sn-seller-resend-modal').fadeOut(120).attr('aria-hidden','true'); });
  $(document).on('click','#sn-resend-submit',function(){
    var data=new FormData();
    data.append('action','sn_seller_resend_financial');
    data.append('nonce',nonce);
    data.append('invoice_id',currentResendInvoiceId);
    data.append('note',$('#sn-resend-note').val()||'');
    data.append('card_from',$('#sn-resend-card-from').val()||'');
    data.append('card_to',$('#sn-resend-card-to').val()||'');
    data.append('amount',$('#sn-resend-amount').val()||'');
    var resendDate=snToEnDigits147($('#sn-resend-paid-date').val()||'');
	data.append('paid_at',resendDate?resendDate+' '+$('#sn-resend-hour').val()+':'+$('#sn-resend-minute').val():'');
	var resendReceipt=$('#sn-resend-receipt')[0].files[0];if(resendReceipt)data.append('receipt',resendReceipt);
    var $btn=$(this).prop('disabled',true).text('در حال ارسال...');
    $.ajax({url:ajax,type:'POST',data:data,processData:false,contentType:false,success:function(res){
      if(res&&res.success){ $('#sn-resend-msg').html('<div class="sn-notice sn-success">'+esc(res.message||'ارسال شد')+'</div>'); setTimeout(function(){ $('#sn-seller-resend-modal').fadeOut(120); loadSellerInvoices(); },600); }
      else { $('#sn-resend-msg').html('<div class="sn-notice sn-error">'+esc((res&&res.message)||'خطا')+'</div>'); }
    }}).fail(function(xhr){ $('#sn-resend-msg').html('<div class="sn-notice sn-error">خطای سرور: '+esc(xhr.status)+'</div>'); }).always(function(){ $btn.prop('disabled',false).text('ارسال به مالی'); });
  });

  var customerActionsPage = 1, customerActionsLoading = false, customerActionsLoaded = false;
  function actionLabel(key, desc){
    var m={invoice_created:'پیش‌فاکتور صادر شد',payment_pending:'پرداخت یا فیش در انتظار بررسی ثبت شد',payment_approved:'پرداخت / تایید مالی ثبت شد',invoice_recontact_requested:'درخواست ارتباط مجدد',customer_invoice_view:'مشاهده لینک فاکتور',customer_product_info:'مطالعه اطلاعات محصول',customer_lottery_info:'مشاهده شانس قرعه‌کشی',customer_wheel_open:'باز کردن گردونه',customer_wheel_spin:'چرخاندن گردونه',customer_reward_apply:'اعمال جایزه',customer_reward_decline:'عدم استفاده از جایزه',customer_coupon_open:'باز کردن کد تخفیف',customer_coupon_apply:'اعمال کد تخفیف',customer_coupon_remove:'لغو کد تخفیف',customer_recontact:'درخواست ارتباط مجدد',customer_pay_online:'انتخاب پرداخت آنلاین',customer_pay_card:'انتخاب کارت‌به‌کارت',customer_receipt_upload:'آپلود فیش',customer_manual_payment:'ثبت اطلاعات واریزی',payment_approved_by_finance:'تایید مالی',finance_approved:'تایید مالی'};
    return desc || m[key] || key || '—';
  }
  function ensureActionsModal(){
    if($('#sn-customer-actions-modal').length) return;
    $('body').append('<div id="sn-customer-actions-modal" class="sn-modal sn-lite-modal" aria-hidden="true"><div class="sn-modal-backdrop sn-actions-close"></div><div class="sn-modal-card"><div class="sn-modal-head"><h3>جزئیات رفتار مشتری</h3><button type="button" class="sn-modal-x sn-actions-close">×</button></div><div id="sn-customer-actions-detail" class="sn-modal-body"><div class="sn-loading">در حال بارگذاری...</div></div></div></div>');
  }
  function loadCustomerActions(reset){
    var $box=$('#sn-customer-actions-list'), $loading=$('#sn-customer-actions-loading');
    if(!$box.length || customerActionsLoading) return;
    if(reset){ customerActionsPage=1; customerActionsLoaded=false; $box.empty(); }
    customerActionsLoading=true;
    if(customerActionsPage===1){ $loading.show(); $box.html('<div class="sn-loading">در حال بارگذاری...</div>'); }
    if(!ajax || !nonce){ customerActionsLoading=false; $loading.hide(); $box.html('<p class="sn-notice sn-error">تنظیمات AJAX پیدا نشد. صفحه را رفرش کنید.</p>'); return; }
    $.ajax({url:ajax,type:'POST',timeout:15000,data:{action:'sn_seller_customer_actions',nonce:nonce,page:customerActionsPage,limit:20}}).done(function(res){
      $loading.hide(); customerActionsLoading=false;
      if(!res||!res.success){ $box.html('<p class="sn-notice sn-error">خطا در دریافت رفتار مشتریان.</p>'); return; }
      var rows=(res.items||[]);
      if(!rows.length && customerActionsPage===1){ $box.html('<p class="sn-notice">هنوز پیش‌فاکتوری برای نمایش رفتار مشتری وجود ندارد.</p>'); return; }
      var html = customerActionsPage===1 ? '<div class="sn-table-wrap"><table class="sn-table sn-customer-actions-table"><thead><tr><th>شماره مشتری</th><th>اسم</th><th>شماره فاکتور</th><th>آخرین فعالیت</th><th>مشاهده کامل</th></tr></thead><tbody>' : '';
      rows.forEach(function(r){
        var latest=actionLabel(r.latest_action_key,r.latest_action);
        if(!r.action_count || Number(r.action_count)===0) latest='هیچ فعالیتی ثبت نشده';
        var time=r.latest_at_jalali ? '<br><small>'+esc(r.latest_at_jalali)+'</small>' : '';
        html+='<tr><td>'+esc(r.customer_phone||'—')+'</td><td>'+esc(r.customer_name||'—')+'</td><td><code>'+esc(r.invoice_code||'—')+'</code></td><td><span class="sn-action-badge">'+esc(latest)+'</span>'+time+'</td><td><button type="button" class="sn-btn sn-btn-sm sn-view-customer-actions" data-id="'+esc(r.invoice_id)+'" data-code="'+esc(r.invoice_code||'')+'">مشاهده کامل</button></td></tr>';
      });
      if(customerActionsPage===1) html+='</tbody></table></div><div class="sn-actions-more-wrap"></div>';
      if(customerActionsPage===1){ $box.html(html); } else { $box.find('tbody').append($(html)); }
      customerActionsLoaded=true;
      var $more=$box.find('.sn-actions-more-wrap');
      if(res.has_more){ $more.html('<button type="button" class="sn-btn sn-btn-secondary sn-load-more-actions">نمایش بیشتر</button>'); customerActionsPage++; }
      else { $more.html(res.total ? '<small class="sn-muted">همه ردیف‌ها نمایش داده شد.</small>' : ''); }
    }).fail(function(xhr){ customerActionsLoading=false; $loading.hide(); $box.html('<p class="sn-notice sn-error">خطای ارتباط با سرور یا زمان‌بر شدن درخواست. دوباره تلاش کنید.</p>'); });
  }
  function loadCustomerActionDetail(invoiceId, code){
    ensureActionsModal();
    $('#sn-customer-actions-modal').fadeIn(120).attr('aria-hidden','false');
    $('#sn-customer-actions-detail').html('<div class="sn-loading">در حال بارگذاری...</div>');
    $.ajax({url:ajax,type:'POST',timeout:15000,data:{action:'sn_seller_customer_actions',nonce:nonce,invoice_id:invoiceId,limit:200}}).done(function(res){
      if(!res||!res.success){ $('#sn-customer-actions-detail').html('<p class="sn-notice sn-error">خطا در دریافت جزئیات.</p>'); return; }
      var rows=(res.items||((res.data&&res.data.items)||[]));
      var html='<div class="sn-profile-card"><h4>فاکتور '+esc(code||'')+'</h4>';
      if(!rows.length){ html+='<p class="sn-notice">برای این فاکتور هنوز فعالیتی ثبت نشده است.</p>'; }
      else{
        html+='<div class="sn-timeline">';
        rows.forEach(function(r){
          var ctx=''; try{var parsed=JSON.parse(r.context||'{}'); ctx=parsed.label||parsed.product||parsed.coupon||parsed.reward||'';}catch(e){ctx='';}
          html+='<div class="sn-timeline-item"><div class="sn-timeline-date">'+esc(r.created_at_jalali||r.created_at||'')+'</div><strong>'+esc(actionLabel(r.action,r.description))+'</strong>'+(ctx?'<p>'+esc(ctx)+'</p>':'')+'</div>';
        });
        html+='</div>';
      }
      html+='</div>';
      $('#sn-customer-actions-detail').html(html);
    }).fail(function(xhr){ $('#sn-customer-actions-detail').html('<p class="sn-notice sn-error">خطای سرور: '+xhr.status+'</p>'); });
  }
  $(document).on('click','#sn-seller-panel .sn-tab[data-tab="customer-actions"]',function(){ if(!customerActionsLoaded) setTimeout(function(){loadCustomerActions(true);},20); });
  $(document).on('click','.sn-load-more-actions',function(){ loadCustomerActions(false); });
  $(document).on('click','.sn-view-customer-actions',function(){ loadCustomerActionDetail($(this).data('id'),$(this).data('code')); });
  $(document).on('click','.sn-actions-close',function(){ $('#sn-customer-actions-modal').fadeOut(120).attr('aria-hidden','true'); });

  // override create invoice button to submit multi products (keeps old endpoint and SMS flow)
  $(document).off('click.snMulti','#sn-create-invoice').on('click.snMulti','#sn-create-invoice',function(e){
    if(window.snSellerInvoiceAuthoritativeHandler){return;}
    e.preventDefault(); e.stopImmediatePropagation();
    var $btn=$(this), lead_id=$('#sn-lead-select').val()||'', name=($('#sn-cust-name').val()||'').trim(), phone=($('#sn-cust-phone').val()||'').trim(), prov=$('#sn-cust-prov').val()||'', city=($('#sn-cust-city').val()||'').trim();
    var ids=[], qtys=[]; $('.sn-product-row').each(function(){ var pid=$(this).find('.sn-product-select').val(); if(pid){ ids.push(pid); qtys.push(Math.max(1,Number($(this).find('.sn-product-qty').val()||1))); } });
    if(!name || !phone || !ids.length){ $('#sn-invoice-notice').html('<div class="sn-notice sn-error">نام، موبایل و حداقل یک محصول الزامی است.</div>'); return; }
    $btn.prop('disabled',true).text('در حال صدور...');
    var data={action:'sn_create_invoice',nonce:nonce,customer_name:name,customer_phone:phone,province:prov,city:city,lead_id:lead_id,product_id:ids[0]};
    ids.forEach(function(v,i){ data['product_ids['+i+']']=v; data['product_qtys['+i+']']=qtys[i]; });
    $.post(ajax,data,function(res){ $btn.prop('disabled',false).text('صدور پیش‌فاکتور و ارسال پیامک'); var payload=(res&&res.data)?res.data:(res||{}); if(res&&res.success){ var code=payload.invoice_code||payload.code||''; var smsLine=''; if(payload.sms_status==='sent'||payload.sms_sent===true){ smsLine='<br>پیامک ارسال شد.'; } else if(payload.sms_status==='failed'||payload.sms_sent===false){ smsLine='<br>پیش‌فاکتور صادر شد اما پیامک ارسال نشد.'; } else if(payload.sms_message){ smsLine='<br>'+esc(payload.sms_message); } $('#sn-invoice-notice').html('<div class="sn-notice sn-success">'+esc(payload.message||'پیش‌فاکتور با موفقیت صادر شد.')+(code?'<br>کد پیش‌فاکتور: <strong>'+esc(code)+'</strong>':'')+smsLine+'</div>'); $('#sn-cust-name,#sn-cust-phone,#sn-cust-city').val(''); $('.sn-product-row:not(:first)').remove(); $('.sn-product-select').val(''); $('.sn-product-qty').val(1); recalcProducts(); loadSellerInvoices(); switchToInvoicesTabAfterCreate(code); } else { $('#sn-invoice-notice').html('<div class="sn-notice sn-error">'+esc(payload.message||(res&&res.message)||'خطا')+'</div>'); } });
  });

  $(document).on('click','.sn-wallet-tabs .sn-subtab',function(){ $('.sn-wallet-tabs .sn-subtab').removeClass('active'); $(this).addClass('active'); var f=$(this).data('wallet-filter'); var $rows=$('.sn-wallet-box tbody tr,.sn-wallet-box .sn-wallet-transaction'); $rows.show(); if(f!=='all'){ $rows.each(function(){ var pm=String($(this).data('payment-method')||''), t=$(this).text(); if(f==='online' && pm!=='online' && t.indexOf('آنلاین')===-1 && t.indexOf('online')===-1) $(this).hide(); if(f==='card_to_card' && pm!=='card_to_card' && pm!=='card' && t.indexOf('کارت')===-1 && t.indexOf('card')===-1) $(this).hide(); }); } });

  // ---- 1.0.55: درخواست اضافه کردن شماره + جستجوی رفتار مشتری + کنسل اقدام مجدد ----
  $(document).on('click', '#sn-submit-extra-number-request', function(){
    var $btn=$(this), phone=String($('#sn-extra-number-phone').val()||'').trim(), name=String($('#sn-extra-number-name').val()||'').trim(), note=String($('#sn-extra-number-note').val()||'').trim();
    if(!phone){ $('#sn-extra-number-notice').html('<div class="sn-notice sn-error">شماره موبایل را وارد کنید.</div>'); return; }
    $btn.prop('disabled',true).text('در حال ثبت...');
    $.post(ajax,{action:'sn_seller_request_extra_number',nonce:nonce,phone:phone,customer_name:name,note:note},function(res){
      $btn.prop('disabled',false).text('ثبت درخواست');
      $('#sn-extra-number-notice').html('<div class="sn-notice '+(res&&res.success?'sn-success':'sn-error')+'">'+esc((res&&res.message)||'خطا')+'</div>');
      if(res&&res.success){ $('#sn-extra-number-phone,#sn-extra-number-name,#sn-extra-number-note').val(''); }
    }).fail(function(xhr){ $btn.prop('disabled',false).text('ثبت درخواست'); $('#sn-extra-number-notice').html('<div class="sn-notice sn-error">خطای سرور: '+xhr.status+'</div>'); });
  });

  function ensureCustomerActionSearch(){
    if($('#sn-customer-actions-search').length) return;
    var html='<div class="sn-card sn-lite-filter"><label>جستجوی شماره مشتری یا کد فاکتور<input type="search" id="sn-customer-actions-search" placeholder="0912... یا INV-..."></label><button type="button" class="sn-btn sn-btn-sm" id="sn-customer-actions-search-btn">جستجو</button><button type="button" class="sn-btn sn-btn-sm sn-btn-secondary" id="sn-customer-actions-clear-btn">پاک کردن</button></div>';
    $('#sn-customer-actions-loading').before(html);
  }
  $(document).on('click','#sn-seller-panel .sn-tab[data-tab="customer-actions"]',function(){ setTimeout(ensureCustomerActionSearch,30); });
  $(document).on('click','#sn-customer-actions-search-btn',function(){ customerActionsPage=1; loadCustomerActions(true); });
  $(document).on('click','#sn-customer-actions-clear-btn',function(){ $('#sn-customer-actions-search').val(''); customerActionsPage=1; loadCustomerActions(true); });
  $(document).on('click','#sn-customer-actions-load-more',function(){ customerActionsPage++; loadCustomerActions(false); });
  var _oldLoadCustomerActions = (typeof loadCustomerActions === 'function') ? loadCustomerActions : null;
  loadCustomerActions = function(reset){
    ensureCustomerActionSearch();
    if(reset){ customerActionsPage=1; $('#sn-customer-actions-list').empty(); }
    var $box=$('#sn-customer-actions-list'), $loading=$('#sn-customer-actions-loading');
    if(!$box.length) return;
    $loading.show();
    $.ajax({url:ajax,type:'POST',timeout:15000,data:{action:'sn_seller_customer_actions',nonce:nonce,page:customerActionsPage,limit:20,q:String($('#sn-customer-actions-search').val()||'')}}).done(function(res){
      $loading.hide();
      if(!res||!res.success){ $box.html('<p class="sn-notice sn-error">خطا در دریافت رفتار مشتریان.</p>'); return; }
      var rows=(res.items||((res.data&&res.data.items)||[]));
      var html = customerActionsPage===1 ? '<div class="sn-table-wrap"><table class="sn-table sn-customer-actions-table"><thead><tr><th>شماره مشتری</th><th>اسم</th><th>شماره فاکتور</th><th>آخرین فعالیت</th><th>مشاهده کامل</th></tr></thead><tbody>' : '';
      if(!rows.length && customerActionsPage===1){ html+='<tr><td colspan="5"><div class="sn-empty-state">موردی پیدا نشد.</div></td></tr>'; }
      rows.forEach(function(r){ var latest=r.latest_action||'—', time=r.latest_at_jalali?'<br><small>'+esc(r.latest_at_jalali)+'</small>':''; html+='<tr><td>'+esc(r.customer_phone||'—')+'</td><td>'+esc(r.customer_name||'—')+'</td><td><code>'+esc(r.invoice_code||'—')+'</code></td><td><span class="sn-action-badge">'+esc(latest)+'</span>'+time+'</td><td><button type="button" class="sn-btn sn-btn-sm sn-view-customer-actions" data-id="'+esc(r.invoice_id)+'" data-code="'+esc(r.invoice_code||'')+'">مشاهده کامل</button></td></tr>'; });
      html += '</tbody></table></div>';
      if(customerActionsPage===1) $box.html(html); else $box.append(html);
      $('#sn-customer-actions-load-more').remove();
      if((res.has_more)||((res.data&&res.data.has_more))){ $box.after('<button type="button" id="sn-customer-actions-load-more" class="sn-btn sn-btn-sm">نمایش بیشتر</button>'); }
    }).fail(function(xhr){ $loading.hide(); $box.html('<p class="sn-notice sn-error">خطای سرور: '+xhr.status+'</p>'); });
  };

  $(document).on('click','.sn-paid-referral-cancel',function(){
    var button=this, id=$(this).data('item-id'), reason=prompt('دلیل کنسل/انصراف را بنویسید:','');
    if(reason===null) return;
    if(!String(reason).trim()){alert('دلیل انصراف الزامی است.');return;}
    $(button).prop('disabled',true);
    $.post(ajax,{action:'sn_paid_referral_cancel',nonce:nonce,item_id:id,reason:reason},function(res){
      if(res&&res.success){
        var $row=$(button).closest('tr').addClass('sn-repeat-action-cancelled');
        $row.find('td').last().html('<span class="sn-status-pill sn-status-danger">انصرافی</span><div class="sn-referral-cancel-reason">دلیل انصراف: '+esc(res.reason||reason)+'</div><button type="button" class="sn-btn sn-btn-sm sn-paid-referral-restore" data-item-id="'+esc(id)+'">بازگرداندن پرونده</button>');
      }else{alert((res&&res.message)||'ثبت تغییر انجام نشد');}
    }).fail(function(xhr){alert('خطای سرور: '+xhr.status);}).always(function(){$(button).prop('disabled',false);});
  });

  $(document).on('click','.sn-paid-referral-restore',function(){
    var button=this, id=$(button).data('item-id');
    $(button).prop('disabled',true);
    $.post(ajax,{action:'sn_paid_referral_restore',nonce:nonce,item_id:id},function(res){
      if(res&&res.success){
        $(button).closest('tr').removeClass('sn-repeat-action-cancelled').find('td').last().html('<span class="sn-status-pill sn-status-success">بازگردانده شد</span><div class="sn-notice sn-success">پرونده فعال شد. برای نمایش اقدام‌های بعدی، تب «نیاز به اقدام مجدد» را دوباره باز کنید.</div>');
      }else{alert((res&&res.message)||'بازگردانی انجام نشد');}
    }).fail(function(xhr){alert('خطای سرور: '+xhr.status);}).always(function(){$(button).prop('disabled',false);});
  });

})(jQuery);

/* 2.0.57 — show exact subscriptions already purchased by the selected customer. */
(function($){
  'use strict';
  if(!$('#sn-seller-panel').length) return;
  var cfg=window.snAjax||window.snData||{}, ajax=cfg.ajaxurl||'', nonce=cfg.nonce||'';
  var timer=null, request=null, requestNo=0, lastPhone='';
  function normalizePhone(value){
    return String(value||'').replace(/[۰-۹]/g,function(d){return String('۰۱۲۳۴۵۶۷۸۹'.indexOf(d));}).replace(/[٠-٩]/g,function(d){return String('٠١٢٣٤٥٦٧٨٩'.indexOf(d));}).replace(/\D+/g,'');
  }
  function ensureTags(){
    $('#sn-products-multi .sn-product-row').each(function(){
      if(!$(this).find('.sn-purchased-product-tag').length){$(this).append('<small class="sn-purchased-product-tag">خریداری‌شده</small>');}
    });
  }
  function syncTags(){
    ensureTags();
    $('#sn-products-multi .sn-product-row').each(function(){
      var purchased=String($(this).find('.sn-product-select option:selected').attr('data-sn-purchased')||'')==='1';
      $(this).toggleClass('has-purchased-subscription',purchased);
    });
  }
  function applyPurchased(ids){
    var purchased={}; (ids||[]).forEach(function(id){purchased[String(id)]=true;});
    $('#sn-products-multi .sn-product-select option').each(function(){
      var $option=$(this), id=String($option.val()||''), type=String($option.attr('data-sn-product-type')||'');
      if(!$option.attr('data-sn-base-label')) $option.attr('data-sn-base-label',$option.text());
      var marked=type==='subscription'&&!!purchased[id];
      $option.attr('data-sn-purchased',marked?'1':'0').text($option.attr('data-sn-base-label')+(marked?' · خریداری‌شده':''));
    });
    syncTags();
  }
  function loadPurchased(){
    var phone=normalizePhone($('#sn-cust-phone').val());
    if(!/^09\d{9}$/.test(phone)){lastPhone='';applyPurchased([]);return;}
    if(phone===lastPhone) return;
    lastPhone=phone; requestNo+=1; var current=requestNo;
    if(request&&request.readyState!==4) request.abort();
    request=$.ajax({url:ajax,type:'POST',dataType:'json',timeout:15000,data:{action:'sn_customer_purchased_subscriptions',nonce:nonce,phone:phone}})
      .done(function(res){if(current!==requestNo)return;applyPurchased(res&&res.success&&res.data?res.data.product_ids:[]);})
      .fail(function(xhr,status){if(status!=='abort'&&current===requestNo)applyPurchased([]);});
  }
  function schedule(){clearTimeout(timer);timer=setTimeout(loadPurchased,320);}
  $(document).on('input.snPurchasedSubscriptions change.snPurchasedSubscriptions blur.snPurchasedSubscriptions','#sn-cust-phone',schedule);
  $(document).on('change.snPurchasedSubscriptions','.sn-product-select',syncTags);
  $(document).on('click.snPurchasedSubscriptions','#sn-add-product-row',function(){setTimeout(function(){applyPurchased($('#sn-product option[data-sn-purchased="1"]').map(function(){return this.value;}).get());},0);});
  $(document).ajaxComplete(function(){setTimeout(schedule,30);});
  $(function(){ensureTags();schedule();});
  window.snClearPurchasedSubscriptionTags=function(){lastPhone='';requestNo+=1;applyPurchased([]);};
})(jQuery);

/* Seller-owned Dot conversion workspace. */
(function($){
  'use strict';
  if(!$('#sn-seller-panel').length) return;
  var cfg=window.snAjax||window.snData||{};
  var ajax=cfg.ajaxurl||window.ajaxurl||'';
  var nonce=cfg.nonce||'';
  function esc(v){return String(v==null?'':v).replace(/[&<>"']/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#039;'}[c]||c;});}
  function loadSellerConversions(){
    var $host=$('#sn-seller-my-conversions');
    if(!$host.length||$host.data('loading')) return;
    $host.data('loading',1).html('<div class="sn-card sn-loading">در حال بارگذاری آماده‌های تبدیل...</div>');
    $.ajax({url:ajax,type:'POST',dataType:'json',timeout:30000,data:{action:'sn_dot_refresh_converter_panel',nonce:nonce,context:'seller'}})
      .done(function(res){
        var html=res&&res.success&&res.data?res.data.html:'';
        if(!html){var msg=(res&&res.data&&res.data.message)||(res&&res.message)||'فهرست آماده‌های تبدیل دریافت نشد.';$host.html('<div class="sn-notice sn-error">'+esc(msg)+'</div>');return;}
        $host.html(html).data('loaded',1);
      })
      .fail(function(xhr){var res=xhr&&xhr.responseJSON?xhr.responseJSON:null;var msg=(res&&res.data&&res.data.message)||'خطای ارتباط با سرور در دریافت آماده‌های تبدیل';$host.html('<div class="sn-notice sn-error">'+esc(msg)+'</div>');})
      .always(function(){$host.data('loading',0);});
  }
  $(document).on('click.snSellerConversions','#sn-seller-panel .sn-tab[data-tab="my-conversions"]',function(){setTimeout(loadSellerConversions,20);});
  window.snLoadSellerConversions=loadSellerConversions;
})(jQuery);

/* Special products are issued alone; full and staged payment are supported. */
(function($){
  'use strict';
  function applyProductCatalogFilter(){
    var query=String($('#sn-product-search').val()||'').toLowerCase().trim();
    var type=String($('#sn-product-type-filter').val()||'all');
    $('.sn-product-select').each(function(){
      var $select=$(this);
      $select.find('option').each(function(){
        var $option=$(this), value=String($option.val()||''), text=String($option.text()||'').toLowerCase();
        if(!value){$option.prop('disabled',false).prop('hidden',false);return;}
        var optionType=String($option.attr('data-sn-product-type')||'product');
        var visible=(!query||text.indexOf(query)!==-1)&&(type==='all'||optionType===type);
        // Keep an already-selected value usable while the catalogue is filtered.
        if($option.is(':selected')) visible=true;
        $option.prop('disabled',!visible).prop('hidden',!visible);
      });
      $select.find('optgroup').each(function(){
        var any=$(this).find('option').filter(function(){return !this.disabled||this.selected;}).length>0;
        $(this).prop('disabled',!any).prop('hidden',!any);
      });
    });
  }
	function syncDotAssessment(){
	  var $rows=$('#sn-products-multi .sn-product-row'), hasSpecial=false, selectedType='';
    $rows.each(function(){ var type=String($(this).find('option:selected').attr('data-sn-product-type')||''); if(['assessment','subscription','product_star'].indexOf(type)!==-1) hasSpecial=true; });
    if(hasSpecial){
      var $dotRow=$rows.filter(function(){var type=String($(this).find('option:selected').attr('data-sn-product-type')||'');return ['assessment','subscription','product_star'].indexOf(type)!==-1;}).first();
	  selectedType=String($dotRow.find('option:selected').attr('data-sn-product-type')||'');
	  var $firstRow=$rows.first();
	  if(!$dotRow.is($firstRow)){
		// Move the selected row itself so its price/label state stays in sync.
		$dotRow.prependTo('#sn-products-multi');
	  }
      $rows.not($dotRow).remove();
	  $dotRow.find('.sn-product-select').attr('id','sn-product');
	  $dotRow.find('.sn-product-qty').val(1).prop('disabled',true).attr('title','تعداد این نوع محصول ثابت است');
      $('#sn-payment-plan option[value="partial"]').prop('disabled',false);
	  var note=selectedType==='assessment'?'پیامک نتیجه اعتبارسنجی پس از تکمیل کل مبلغ ارسال می‌شود.':'این محصول مستقیم وارد چرخه مربوط به خودش می‌شود و نیاز به انتخاب محصول دوم ندارد.';
	  if(!$('#sn-dot-assessment-payment-note').length){ $('#sn-payment-plan').after('<small id="sn-dot-assessment-payment-note" class="sn-muted"></small>'); }
	  $('#sn-dot-assessment-payment-note').text('پرداخت کامل یا مرحله‌ای مجاز است؛ '+note);
	  $('#sn-add-product-row').prop('disabled',true).attr('title','این نوع محصول باید به‌تنهایی ثبت شود');
    } else {
	  $rows.find('.sn-product-qty').prop('disabled',false).removeAttr('title');
      $('#sn-payment-plan option[value="partial"]').prop('disabled',false);
      $('#sn-dot-assessment-payment-note').remove();
	  $('#sn-add-product-row').prop('disabled',false).removeAttr('title');
	}
	var hasAssessment=selectedType==='assessment';
	$('#sn-dot-conversion-route').prop('hidden',!hasAssessment).toggle(hasAssessment);
	if(!hasAssessment){$('input[name="sn_dot_conversion_route"],input[name="dot_send_assessment_sms"]').prop('checked',false);}
	var smsChoiceMade=hasAssessment&&$('input[name="dot_send_assessment_sms"]:checked').length>0;
	var $ownerOptions=$('#sn-dot-conversion-owner-options');
	$('input[name="sn_dot_conversion_route"]').prop('disabled',!smsChoiceMade);
	$ownerOptions.toggleClass('is-disabled',!smsChoiceMade).attr('aria-disabled',smsChoiceMade?'false':'true');
	$('#sn-dot-conversion-owner-help').prop('hidden',smsChoiceMade).toggle(!smsChoiceMade);
	var hasPhysical=$('#sn-products-multi .sn-product-select option:selected').filter(function(){return String($(this).attr('data-sn-product-type')||'')==='product';}).length>0;
	$('.sn-shipping-customer-fields').prop('hidden',!hasPhysical).toggle(hasPhysical);
	$('#sn-cust-address').prop('required',hasPhysical);
  }
  $(document).on('change.snDotAssessment','.sn-product-select',syncDotAssessment);
	$(document).on('change.snDotAssessmentSms','input[name="dot_send_assessment_sms"]',syncDotAssessment);
	$(document).on('input.snProductCatalog change.snProductCatalog','#sn-product-search,#sn-product-type-filter',applyProductCatalogFilter);
	$(document).on('ajaxComplete.snDotAssessment',syncDotAssessment);
	$(function(){syncDotAssessment();applyProductCatalogFilter();});
})(jQuery);

/* SN 1.0.8 final override: remove legacy single-product invoice click */
(function($){
  if(!$('#sn-seller-panel').length) return;
  var $sellerPanel=$('#sn-seller-panel');
  var ajax=(window.snAjax||window.snData||{}).ajaxurl, nonce=(window.snAjax||window.snData||{}).nonce;
  window.snSellerInvoiceAuthoritativeHandler='2.0.36';
  // This handler lives in its own closure; read the permission from the panel
  // instead of referencing the similarly named variable in the legacy closure.
  var sellerManualInvoiceEnabled=String($sellerPanel.attr('data-sn-manual-invoice-enabled')||'1')!=='0';
  function esc(v){ return String(v==null?'':v).replace(/[&<>"']/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#039;'}[c];}); }
  function maxInvoiceProducts(){
    var value=Number($('#sn-seller-panel').attr('data-sn-max-products')||$('#sn-add-product-row').attr('data-max-products')||0);
    return isFinite(value)&&value>0?Math.floor(value):0;
  }
  function snToEnDigits147(value){return String(value||'').replace(/[۰-۹]/g,function(d){return '۰۱۲۳۴۵۶۷۸۹'.indexOf(d);}).replace(/[٠-٩]/g,function(d){return '٠١٢٣٤٥٦٧٨٩'.indexOf(d);});}
  function snFormatNumber147(value){var n=Number(value||0);try{return n.toLocaleString('fa-IR');}catch(e){return String(n);}}
  function snFormatMoney147(value){return snFormatNumber147(value)+' تومان';}
  function switchToInvoicesTabAfterCreate(code){
    setTimeout(function(){
      var $tab=$('#sn-seller-panel .sn-tab[data-tab="invoices"]');
      if($tab.length){ $tab.trigger('click'); }
      window.snLastCreatedInvoiceCode = code || '';
      $('#sn-seller-invoice-search').val('');
      var $pre=$('.sn-invoice-status-tabs .sn-subtab[data-status="pre_invoice"]');
      if($pre.length){ $pre.trigger('click'); }
    },1800);
  }
  function recalc(){ var total=0; $('.sn-product-row').each(function(){ total += Number($(this).find('option:selected').data('price')||0) * Math.max(1,Number($(this).find('.sn-product-qty').val()||1)); }); try{$('#sn-products-total').text('جمع: '+total.toLocaleString('fa-IR')+' تومان');}catch(e){$('#sn-products-total').text('جمع: '+total+' تومان');} }
  function snPaymentNumber(value){
    return Number(snToEnDigits147(String(value||'')).replace(/[^0-9.]/g,'')) || 0;
  }
  function snInvoiceFormTotal147(){
    var total=0;
    $('#sn-products-multi .sn-product-row').each(function(){
      var price=Number($(this).find('.sn-product-select option:selected').data('price')||0);
      var qty=Math.max(1,Number($(this).find('.sn-product-qty').val()||1));
      total += price*qty;
    });
    $('#sn-products-total').text('جمع: '+snFormatNumber147(total)+' تومان').data('total',total);
    return total;
  }
  function snSelectedPrepayment147(){
    var choice=String($('#sn-prepayment-choice').val()||'');
    return choice==='custom' ? snPaymentNumber($('#sn-prepayment-custom').val()) : snPaymentNumber(choice);
  }
  function snSyncPaymentPlan147(){
    var assessment=$('#sn-products-multi .sn-product-select option:selected[data-sn-dot-assessment="1"]').length>0;
    $('#sn-payment-plan option[value="partial"]').prop('disabled',assessment);
    if(assessment && $('#sn-payment-plan').val()==='partial') $('#sn-payment-plan').val('full');
    var partial=String($('#sn-payment-plan').val()||'full')==='partial';
    var total=snInvoiceFormTotal147();
    var $fields=$('.sn-prepayment-fields');
    var $choice=$('#sn-prepayment-choice');
    var choice=String($choice.val()||'');

    // Preset values are global settings, but an individual invoice may be smaller.
    // Never leave an impossible preset selectable because it makes the 3-box preview
    // look valid while the server correctly rejects the request.
    $choice.find('option').each(function(){
      var raw=String($(this).attr('value')||'');
      if(!raw || raw==='custom'){ $(this).prop('disabled',false); return; }
      var preset=snPaymentNumber(raw);
      $(this).prop('disabled', total<=0 || preset<=0 || preset>=total);
    });
    if(choice && choice!=='custom' && $choice.find('option:selected').prop('disabled')){
      $choice.val('');
      choice='';
    }

    var due=partial ? snSelectedPrepayment147() : total;
    var validPartial=partial && total>0 && due>0 && due<=total;
    var remaining=validPartial ? Math.max(0,total-due) : 0;
    $fields.toggleClass('is-visible',partial).css('display',partial?'block':'none').attr('aria-hidden',partial?'false':'true');
    if(!partial){
      $choice.val('');
      $('#sn-prepayment-custom').val('').hide();
      due=total;
    } else {
      $('#sn-prepayment-custom').toggle(String($choice.val()||'')==='custom');
    }

    $('#sn-prepayment-total-preview').text(snFormatMoney147(total));
    if(!partial){
      $('#sn-prepayment-due-preview').text(snFormatMoney147(total)).removeClass('is-invalid');
      $('#sn-prepayment-remaining-preview').text(snFormatMoney147(0)).removeClass('is-invalid');
    } else if(validPartial){
      $('#sn-prepayment-due-preview').text(snFormatMoney147(due)).removeClass('is-invalid');
      $('#sn-prepayment-remaining-preview').text(snFormatMoney147(remaining)).removeClass('is-invalid');
    } else {
      var hasInput=String($choice.val()||'')!=='' || String($('#sn-prepayment-custom').val()||'').trim()!=='';
      $('#sn-prepayment-due-preview').text(hasInput && due>0 ? snFormatMoney147(due) : 'انتخاب نشده').toggleClass('is-invalid',hasInput);
      $('#sn-prepayment-remaining-preview').text('—').removeClass('is-invalid');
    }
  }
  $(document).off('change.snPayment147','#sn-payment-plan').on('change.snPayment147','#sn-payment-plan',snSyncPaymentPlan147);
  $(document).off('change.snPayment147','#sn-prepayment-choice').on('change.snPayment147','#sn-prepayment-choice',function(){
    $('#sn-prepayment-custom').toggle(String($(this).val()||'')==='custom');
    snSyncPaymentPlan147();
  });
  $(document).off('input.snPayment147 change.snPayment147','#sn-prepayment-custom,.sn-product-select,.sn-product-qty')
    .on('input.snPayment147 change.snPayment147','#sn-prepayment-custom,.sn-product-select,.sn-product-qty',snSyncPaymentPlan147);
  $(document).off('click.snPayment147','#sn-seller-panel .sn-tab[data-tab="new-invoice"]')
    .on('click.snPayment147','#sn-seller-panel .sn-tab[data-tab="new-invoice"]',function(){setTimeout(snSyncPaymentPlan147,0);});
  $(function(){snSyncPaymentPlan147();});

  // Remove every legacy invoice-create handler and bind one authoritative staged-payment handler.
  $(document).off('click','#sn-create-invoice').on('click.snInvoiceCreate165','#sn-create-invoice',function(e){
    e.preventDefault(); e.stopImmediatePropagation();
    var $btn=$(this), lead_id=$('#sn-lead-select').val()||'', name=($('#sn-cust-name').val()||'').trim(), phone=($('#sn-cust-phone').val()||'').trim(), phoneSecondary=($('#sn-cust-phone-secondary').val()||'').trim(), prov=$('#sn-cust-prov').val()||'', city=($('#sn-cust-city').val()||'').trim(), address=($('#sn-cust-address').val()||'').trim(), postal=($('#sn-cust-postal-code').val()||'').trim();
	if($btn.data('snInvoiceBusy') || window.snSellerInvoiceRequestPending){return;}
	if(!sellerManualInvoiceEnabled && !lead_id){$('#sn-invoice-notice').html('<div class="sn-notice sn-error">صدور آزاد غیرفعال است؛ از تب «شماره‌های من» یک شماره تخصیص‌یافته را انتخاب کنید.</div>');return;}
    var ids=[], qtys=[];
    $('#sn-products-multi .sn-product-row').each(function(){var pid=$(this).find('.sn-product-select').val();if(pid){ids.push(pid);qtys.push(Math.max(1,Number($(this).find('.sn-product-qty').val()||1)));}});
    if(!name || !phone || !ids.length){$('#sn-invoice-notice').html('<div class="sn-notice sn-error">نام، موبایل و حداقل یک محصول الزامی است.</div>');return;}
	var hasPhysical=$('#sn-products-multi .sn-product-select option:selected').filter(function(){return String($(this).attr('data-sn-product-type')||'')==='product';}).length>0;
	if(hasPhysical && (!prov || !city || !address)){$('#sn-invoice-notice').html('<div class="sn-notice sn-error">برای محصول عادی، استان، شهر و آدرس کامل مشتری الزامی است.</div>');return;}
	postal=snToEnDigits147(postal).replace(/\D+/g,'');
	if(postal && postal.length!==10){$('#sn-invoice-notice').html('<div class="sn-notice sn-error">کد پستی باید ۱۰ رقم باشد.</div>');return;}
    var maxProducts149=maxInvoiceProducts(); if(maxProducts149>0 && ids.length>maxProducts149){$('#sn-invoice-notice').html('<div class="sn-notice sn-error">حداکثر '+maxProducts149+' محصول در هر پیش‌فاکتور قابل انتخاب است.</div>');return;}
    var paymentPlan=String($('#sn-payment-plan').val()||'full');
    if(paymentPlan!=='partial') paymentPlan='full';
    var invoiceTotal=snInvoiceFormTotal147();
    var prepaymentAmount=paymentPlan==='partial' ? snSelectedPrepayment147() : 0;
    if(invoiceTotal<=0){$('#sn-invoice-notice').html('<div class="sn-notice sn-error">مبلغ محصولات معتبر نیست.</div>');return;}
    if(paymentPlan==='partial' && (prepaymentAmount<=0 || prepaymentAmount>invoiceTotal)){
      $('#sn-invoice-notice').html('<div class="sn-notice sn-error">مبلغ این مرحله باید بیشتر از صفر و حداکثر برابر مبلغ کل فاکتور باشد.</div>');
      $('.sn-prepayment-fields').addClass('is-visible').show();
      return;
    }
	var hasAssessment=$('#sn-products-multi .sn-product-select option:selected').filter(function(){return String($(this).attr('data-sn-product-type')||'')==='assessment';}).length>0;
	var dotSmsChoice=hasAssessment ? $('input[name="dot_send_assessment_sms"]:checked').val() : '1';
	if(hasAssessment && typeof dotSmsChoice==='undefined'){$('#sn-invoice-notice').html('<div class="sn-notice sn-error">ابتدا ارسال یا عدم ارسال پیامک نتیجه اعتبارسنجی را انتخاب کنید.</div>');return;}
	var dotConversionRoute=hasAssessment ? $('input[name="sn_dot_conversion_route"]:checked').val() : 'supervisor_queue';
	if(hasAssessment && !dotConversionRoute){$('#sn-invoice-notice').html('<div class="sn-notice sn-error">گزینه «ارسال برای سرپرست / تبدیل توسط خودم» را انتخاب کنید.</div>');return;}
	var dotSendAssessmentSms=String(dotSmsChoice)==='1' ? 1 : 0;
    var sendBale=$('#sn-send-bale').is(':checked')?1:0;
    var data={action:'sn_create_invoice',nonce:nonce,customer_name:name,customer_phone:phone,customer_phone_secondary:phoneSecondary,province:prov,city:city,customer_address:address,customer_postal_code:postal,lead_id:lead_id,product_id:ids[0],product_ids:ids,product_qtys:qtys,payment_plan:paymentPlan,prepayment_amount:prepaymentAmount,dot_conversion_route:dotConversionRoute,dot_send_assessment_sms:dotSendAssessmentSms,dot_assessment_decision_ui:hasAssessment?1:0,send_bale:sendBale};
    var endpoint=String(ajax||'');
    if(endpoint && endpoint.indexOf('action=sn_create_invoice')===-1){endpoint+=(endpoint.indexOf('?')===-1?'?':'&')+'action=sn_create_invoice';}
    if(!endpoint){$('#sn-invoice-notice').html('<div class="sn-notice sn-error">آدرس ارتباط با سرور در صفحه موجود نیست؛ صفحه را تازه‌سازی کنید.</div>');return;}
    $btn.data('snInvoiceBusy',1).prop('disabled',true).text('در حال صدور...');
    window.snSellerInvoiceRequestPending=true;
    $('#sn-invoice-notice').html('<div class="sn-notice sn-info">در حال ثبت پیش‌فاکتور؛ لطفاً صبر کنید...</div>');
    function handleInvoiceResult165(res){
      var payload=(res&&res.data)?res.data:(res||{});
      if(res&&res.success){
        var code=payload.invoice_code||payload.code||'', due=Number(payload.current_due_amount||0), plan=String(payload.payment_plan||paymentPlan);
        var html='<div class="sn-notice sn-success">'+esc(payload.message||res.message||'پیش‌فاکتور با موفقیت صادر شد.')+(code?'<br>کد پیش‌فاکتور: <strong>'+esc(code)+'</strong>':'')+'<br>نوع پرداخت: <strong>'+(plan==='partial'?'پیش‌پرداخت':'پرداخت کامل')+'</strong><br>مبلغ قابل پرداخت مشتری در این مرحله: <strong>'+esc(snFormatMoney147(due))+'</strong>';
	    if(payload.dot_conversion_route_label){html+='<br>مسیر تبدیل: <strong>'+esc(payload.dot_conversion_route_label)+'</strong>';}
	    if(payload.dot_send_assessment_sms_label){html+='<br>ادامه اعتبارسنجی: <strong>'+esc(payload.dot_send_assessment_sms_label)+'</strong>';}
	    var smsResultMessage=payload.sms_message || ((payload.sms_status==='sent'||payload.sms_sent===true)?'درخواست پیامک توسط سرویس پذیرفته شد.':'ارسال پیامک تأیید نشد.');
	    html+='<br>'+esc(smsResultMessage);
        var baleResultMessage='';
        if(Number(payload.bale_requested)===0 || payload.bale_status==='not_requested'){baleResultMessage='ارسال در بله انتخاب نشده بود.';}
        else if(payload.bale_status==='sent'||payload.bale_sent===true){baleResultMessage='پیام بله ارسال شد.';}
        else{baleResultMessage='پیش‌فاکتور صادر شد اما ارسال در بله انجام نشد.';}
        html+='<br>'+esc(baleResultMessage);
	    $('#sn-invoice-sms-result').remove();
        var deliveryClass=(payload.sms_sent===true && (Number(payload.bale_requested)===0 || payload.bale_sent===true))?'sn-success':'sn-info';
	    $('#sn-tab-invoices').prepend('<div id="sn-invoice-sms-result" class="sn-notice '+deliveryClass+'" role="status">فاکتور '+esc(code)+' — '+esc(smsResultMessage)+'<br>'+esc(baleResultMessage)+'</div>');
        html+='</div>'; $('#sn-invoice-notice').html(html);
        $('#sn-cust-name,#sn-cust-phone,#sn-cust-phone-secondary,#sn-cust-city,#sn-cust-address,#sn-cust-postal-code,#sn-prepayment-custom').val('');
	    $('#sn-payment-plan').val('full'); $('#sn-prepayment-choice').val('');
	    $('input[name="sn_dot_conversion_route"],input[name="dot_send_assessment_sms"]').prop('checked',false);
        $('#sn-send-bale').prop('checked',true);
        $('#sn-products-multi .sn-product-row:not(:first)').remove(); $('#sn-products-multi .sn-product-select').val(''); $('#sn-products-multi .sn-product-qty').val(1);
        snSyncPaymentPlan147();
        if(typeof window.snLoadSellerInvoices==='function'){window.snLoadSellerInvoices();} if(typeof window.snSellerReloadLeads==='function'){window.snSellerReloadLeads();} switchToInvoicesTabAfterCreate(code);
      }else{$('#sn-invoice-notice').html('<div class="sn-notice sn-error">'+esc(payload.message||(res&&res.message)||'صدور پیش‌فاکتور انجام نشد.')+'</div>');}
    }
    $.ajax({url:endpoint,type:'POST',data:data,dataType:'json',timeout:30000}).done(handleInvoiceResult165).fail(function(xhr,status){
      var response=xhr&&xhr.responseJSON?xhr.responseJSON:null;
      if(!response && xhr && xhr.responseText){
        var raw=String(xhr.responseText||'');
        try{response=JSON.parse(raw);}catch(ignore){
          var jsonMatch=raw.match(/\{[\s\S]*"success"[\s\S]*\}/);
          if(jsonMatch){try{response=JSON.parse(jsonMatch[0]);}catch(ignoreMatchedJson){}}
        }
      }
      if(response && typeof response.success!=='undefined'){handleInvoiceResult165(response);return;}
      var payload=response&&response.data?response.data:(response||{});
      var msg=payload.message||(response&&response.message)||'';
      if(!msg){
        if(status==='timeout'){msg='زمان پاسخ سرور تمام شد. دوباره تلاش کنید.';}
        else if(Number(xhr&&xhr.status)===400 && String(xhr&&xhr.responseText||'').trim()==='0'){msg='درخواست صدور توسط وردپرس شناسایی نشد؛ صفحه را با Ctrl+F5 تازه‌سازی و دوباره امتحان کنید.';}
        else{msg='ارتباط با سرور کامل نشد'+(xhr&&xhr.status?' (کد '+xhr.status+')':'')+'.';}
      }
      $('#sn-invoice-notice').html('<div class="sn-notice sn-error">'+esc(msg)+'</div>');
    }).always(function(){
      window.snSellerInvoiceRequestPending=false;
      $btn.removeData('snInvoiceBusy').prop('disabled',false).text('صدور پیش‌فاکتور و ارسال پیامک / بله');
    });
  });

})(jQuery);


/* SN 1.0.56: seller panel light mode + extra number request reliable binding */
(function($){
  'use strict';
  if(!$('#sn-seller-panel').length) return;
  var cfg = window.snAjax || window.snData || {};
  var ajax = cfg.ajaxurl;
  var nonce = cfg.nonce;
  function esc(v){ return String(v==null?'':v).replace(/[&<>"']/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#039;'}[c];}); }
  function show($box, type, msg){ $box.html('<div class="sn-notice '+(type==='success'?'sn-success':'sn-error')+'">'+esc(msg||'')+'</div>'); }

  // قبلی‌ها چند بار bind شده بودند؛ همه را پاک می‌کنیم و یک handler قطعی می‌گذاریم.
  $(document).off('click', '#sn-submit-extra-number-request');
  $(document).on('click.snExtraNumber', '#sn-submit-extra-number-request', function(e){
    e.preventDefault();
    var $btn=$(this), $notice=$('#sn-extra-number-notice');
    if($btn.data('busy')) return;
    var phone=String($('#sn-extra-number-phone').val()||'').trim();
    var name=String($('#sn-extra-number-name').val()||'').trim();
    var note=String($('#sn-extra-number-note').val()||'').trim();
    if(!phone){ show($notice,'error','شماره موبایل را وارد کنید.'); return; }
    $btn.data('busy',1).prop('disabled',true).text('در حال ثبت...');
    $.ajax({url:ajax,type:'POST',timeout:12000,data:{action:'sn_seller_request_extra_number',nonce:nonce,phone:phone,customer_name:name,note:note}})
      .done(function(res){
        if(res && res.success){
          show($notice,'success',res.message||'درخواست ثبت شد.');
          $('#sn-extra-number-phone,#sn-extra-number-name,#sn-extra-number-note').val('');
        } else {
          show($notice,'error',(res&&res.message)||'ثبت درخواست انجام نشد.');
        }
      })
      .fail(function(xhr){ var msg='خطای سرور: '+xhr.status; if(xhr&&xhr.responseJSON&&xhr.responseJSON.message){ msg=xhr.responseJSON.message; } else if(xhr&&xhr.responseText){ var t=String(xhr.responseText).replace(/<[^>]+>/g,' ').replace(/\s+/g,' ').trim(); if(t){ msg += ' — ' + t.slice(0,180); } } show($notice,'error',msg); })
      .always(function(){ $btn.data('busy',0).prop('disabled',false).text('ثبت درخواست'); });
  });

  $(document).on('click.snExtraTab', '#sn-seller-panel .sn-tab[data-tab="extra-number-request"]', function(){
    setTimeout(function(){ $('#sn-extra-number-phone').trigger('focus'); }, 80);
  });

  function loadWallet(e){
    if(e && e.preventDefault){ e.preventDefault(); e.stopPropagation(); }
    var $box=$('#sn-seller-wallet-box');
    if(!$box.length){ return false; }
    if(String($box.attr('data-loaded')||$box.data('loaded')||'')==='loading'){ return false; }
    var ajaxUrl = (window.snAjax && window.snAjax.ajaxurl) || (window.snData && window.snData.ajaxurl) || window.ajaxurl || ajax;
    var ajaxNonce = (window.snAjax && window.snAjax.nonce) || (window.snData && window.snData.nonce) || nonce;
    if(!ajaxUrl){ $box.html('<div class="sn-notice sn-error">آدرس AJAX پیدا نشد. صفحه را کامل رفرش کنید.</div>'); return false; }
    $box.attr('data-loaded','loading').data('loaded','loading').html('<div class="sn-loading">در حال بارگذاری کیف پول...</div>');
    $.ajax({url:ajaxUrl,type:'POST',timeout:30000,data:{action:'sn_seller_wallet_box',nonce:ajaxNonce,light:1,sn_wallet_light:1}})
      .done(function(res){
        var html = (res && res.html) || (res && res.data && res.data.html) || '';
        if(res && res.success && html){
          $box.attr('data-loaded','1').data('loaded','1').replaceWith(html);
        } else {
          $box.attr('data-loaded','0').data('loaded','0').html('<div class="sn-notice sn-error">'+esc((res&&res.message)||'کیف پول بارگذاری نشد.')+'<br><button type="button" class="sn-btn sn-btn-secondary" id="sn-load-seller-wallet">تلاش مجدد</button></div>');
        }
      })
      .fail(function(xhr){
        var msg = 'خطای سرور در بارگذاری کیف پول';
        if(xhr && xhr.status){ msg += ': '+xhr.status; }
        if(xhr && xhr.responseJSON && xhr.responseJSON.message){ msg += ' — '+xhr.responseJSON.message; }
        $box.attr('data-loaded','0').data('loaded','0').html('<div class="sn-notice sn-error">'+esc(msg)+'<br><button type="button" class="sn-btn sn-btn-secondary" id="sn-load-seller-wallet">تلاش مجدد</button></div>');
      });
    return false;
  }
  $(document).off('click.snWalletLoad').on('click.snWalletLoad', '#sn-load-seller-wallet, [data-sn-load-wallet]', loadWallet);
  $(document).off('click.snWalletTab').on('click.snWalletTab', '#sn-seller-panel .sn-tab[data-tab="wallet"]', function(){ setTimeout(function(){ loadWallet(); }, 80); });

  // اگر نسخه‌ای از قالب کیف پول بدون wrapper قدیمی رندر شده باشد، همینجا آن را قابل بارگذاری AJAX می‌کنیم.
  $(function(){
    var $walletTab = $('#sn-tab-wallet');
    if($walletTab.length && !$('#sn-seller-wallet-box').length && $walletTab.find('.sn-wallet-box').length){
      $walletTab.find('.sn-wallet-box').first().wrap('<div id="sn-seller-wallet-box" data-loaded="0"></div>');
    }
  });

})(jQuery);


/* SN 1.0.67 quick fix: reliable seller customer actions */
(function($){
  'use strict';
  if(!$('#sn-seller-panel').length) return;
  var cfg = window.snAjax || window.snData || {};
  var ajax = cfg.ajaxurl || '';
  var nonce = cfg.nonce || '';
  var page = 1;
  function esc(v){ return String(v==null?'':v).replace(/[&<>"']/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#039;'}[c] || c;}); }
  function label(key, desc){
    var m={invoice_created:'پیش‌فاکتور صادر شد',payment_pending:'پرداخت یا فیش در انتظار بررسی ثبت شد',payment_approved:'پرداخت / تایید مالی ثبت شد',invoice_recontact_requested:'درخواست ارتباط مجدد',customer_invoice_view:'مشاهده لینک فاکتور',customer_product_info:'مطالعه اطلاعات محصول',customer_lottery_info:'مشاهده شانس قرعه‌کشی',customer_wheel_open:'باز کردن گردونه',customer_wheel_spin:'چرخاندن گردونه',customer_reward_apply:'اعمال جایزه',customer_reward_decline:'عدم استفاده از جایزه',customer_coupon_open:'باز کردن کد تخفیف',customer_coupon_apply:'اعمال کد تخفیف',customer_coupon_remove:'لغو کد تخفیف',customer_recontact:'درخواست ارتباط مجدد',customer_pay_online:'انتخاب پرداخت آنلاین',customer_pay_card:'انتخاب کارت‌به‌کارت',customer_receipt_upload:'آپلود فیش',customer_manual_payment:'ثبت اطلاعات واریزی',seller_resend_to_financial:'ارسال مجدد به مالی',payment_approved_by_finance:'تایید مالی',finance_approved:'تایید مالی'};
    return desc || m[key] || key || '—';
  }
  function ensureSearch(){
    if($('#sn-customer-actions-search').length) return;
    $('#sn-customer-actions-loading').before('<div class="sn-card sn-lite-filter sn-customer-actions-filter"><label>جستجوی شماره مشتری یا کد فاکتور<input type="search" id="sn-customer-actions-search" placeholder="0912... یا کد فاکتور"></label><button type="button" class="sn-btn sn-btn-sm" id="sn-customer-actions-search-btn">جستجو</button><button type="button" class="sn-btn sn-btn-sm sn-btn-secondary" id="sn-customer-actions-clear-btn">پاک کردن</button></div>');
  }
  function renderRows(rows, total, hasMore, reset){
    var $box=$('#sn-customer-actions-list');
    if(reset){ $box.empty(); }
    if(!rows.length && reset){ $box.html('<div class="sn-empty-state">هنوز فاکتوری برای نمایش رفتار مشتری وجود ندارد.</div>'); return; }
    var html = reset ? '<div class="sn-table-wrap"><table class="sn-table sn-customer-actions-table"><thead><tr><th>شماره مشتری</th><th>اسم</th><th>شماره فاکتور</th><th>آخرین فعالیت</th><th>مشاهده کامل</th></tr></thead><tbody>' : '';
    rows.forEach(function(r){
      var latest=label(r.latest_action_key,r.latest_action);
      if(!r.action_count || Number(r.action_count)===0) latest = latest || 'پیش‌فاکتور صادر شده';
      var time=r.latest_at_jalali?'<br><small>'+esc(r.latest_at_jalali)+'</small>':'';
      html+='<tr><td>'+esc(r.customer_phone||'—')+'</td><td>'+esc(r.customer_name||'—')+'</td><td><code>'+esc(r.invoice_code||'—')+'</code></td><td><span class="sn-action-badge">'+esc(latest)+'</span>'+time+'</td><td><button type="button" class="sn-btn sn-btn-sm sn-view-customer-actions" data-id="'+esc(r.invoice_id)+'" data-code="'+esc(r.invoice_code||'')+'">مشاهده کامل</button></td></tr>';
    });
    if(reset){ html+='</tbody></table></div><div class="sn-actions-more-wrap"></div>'; $box.html(html); } else { $box.find('tbody').append(html); }
    var $more=$box.find('.sn-actions-more-wrap');
    if(!$more.length){ $more=$('<div class="sn-actions-more-wrap"></div>').appendTo($box); }
    $more.html(hasMore?'<button type="button" id="sn-customer-actions-load-more" class="sn-btn sn-btn-sm">نمایش بیشتر</button>':'<small class="sn-muted">نمایش '+esc($box.find('tbody tr').length)+' از '+esc(total||rows.length)+'</small>');
  }
  function load(reset){
    ensureSearch();
    if(reset) page=1;
    var $loading=$('#sn-customer-actions-loading'), $box=$('#sn-customer-actions-list');
    $loading.show(); if(reset) $box.html('<div class="sn-loading">در حال بارگذاری...</div>');
    $.ajax({url:ajax,type:'POST',dataType:'json',timeout:20000,data:{action:'sn_seller_customer_actions',nonce:nonce,page:page,limit:20,q:String($('#sn-customer-actions-search').val()||'')}})
      .done(function(res){
        $loading.hide();
        if(!res || !res.success){ $box.html('<div class="sn-notice sn-error">رفتار مشتریان دریافت نشد. صفحه را رفرش نکن؛ دوباره روی جستجو بزن.</div>'); return; }
        var rows=res.items || (res.data&&res.data.items) || [];
        renderRows(rows, Number(res.total || (res.data&&res.data.total) || rows.length || 0), !!(res.has_more || (res.data&&res.data.has_more)), reset);
        if(res.has_more || (res.data&&res.data.has_more)) page++;
      })
      .fail(function(xhr){ $loading.hide(); $box.html('<div class="sn-notice sn-error">خطای سرور رفتار مشتریان: '+esc(xhr.status||'')+'</div>'); });
  }
  function ensureModal(){
    if($('#sn-customer-actions-modal').length) return;
    $('body').append('<div id="sn-customer-actions-modal" class="sn-modal sn-lite-modal" aria-hidden="true"><div class="sn-modal-backdrop sn-actions-close"></div><div class="sn-modal-card"><div class="sn-modal-head"><h3>جزئیات کامل رفتار مشتری</h3><button type="button" class="sn-modal-x sn-actions-close">×</button></div><div id="sn-customer-actions-detail" class="sn-modal-body"></div></div></div>');
  }
  function detail(id, code){
    ensureModal(); $('#sn-customer-actions-modal').fadeIn(120).attr('aria-hidden','false'); $('#sn-customer-actions-detail').html('<div class="sn-loading">در حال بارگذاری...</div>');
    $.ajax({url:ajax,type:'POST',dataType:'json',timeout:20000,data:{action:'sn_seller_customer_actions',nonce:nonce,invoice_id:id,limit:300}})
      .done(function(res){
        var rows=(res&&res.items)||(res&&res.data&&res.data.items)||[];
        var html='<div class="sn-profile-card"><h4>فاکتور '+esc(code||'')+'</h4>';
        if(!res || !res.success){ html+='<p class="sn-notice sn-error">جزئیات دریافت نشد.</p>'; }
        else if(!rows.length){ html+='<p class="sn-notice">برای این فاکتور هنوز فعالیتی ثبت نشده است.</p>'; }
        else { html+='<div class="sn-timeline">'; rows.forEach(function(r){ var ctx=''; try{var parsed=JSON.parse(r.context||'{}'); ctx=parsed.label||parsed.product||parsed.coupon||parsed.reward||parsed.note||'';}catch(e){} html+='<div class="sn-timeline-item"><div class="sn-timeline-date">'+esc(r.created_at_jalali||r.created_at||'')+'</div><strong>'+esc(label(r.action,r.description))+'</strong>'+(ctx?'<p>'+esc(ctx)+'</p>':'')+'</div>'; }); html+='</div>'; }
        html+='</div>'; $('#sn-customer-actions-detail').html(html);
      })
      .fail(function(xhr){ $('#sn-customer-actions-detail').html('<div class="sn-notice sn-error">خطای سرور: '+esc(xhr.status||'')+'</div>'); });
  }
  $(document).off('click','#sn-customer-actions-search-btn');
  $(document).off('click','#sn-customer-actions-clear-btn');
  $(document).off('click','#sn-customer-actions-load-more');
  $(document).off('click','.sn-view-customer-actions');
  $(document).on('click.snCustomerActionsFix','#sn-seller-panel .sn-tab[data-tab="customer-actions"]',function(){ setTimeout(function(){ load(true); },80); });
  $(document).on('click.snCustomerActionsFix','#sn-customer-actions-search-btn',function(){ load(true); });
  $(document).on('click.snCustomerActionsFix','#sn-customer-actions-clear-btn',function(){ $('#sn-customer-actions-search').val(''); load(true); });
  $(document).on('click.snCustomerActionsFix','#sn-customer-actions-load-more',function(){ load(false); });
  $(document).on('click.snCustomerActionsFix','.sn-view-customer-actions',function(e){ e.preventDefault(); detail($(this).data('id'),$(this).data('code')); });
  $(document).on('click.snCustomerActionsFix','.sn-actions-close',function(){ $('#sn-customer-actions-modal').fadeOut(120).attr('aria-hidden','true'); });
  window.snCustomerActionsReload = function(){ load(true); };
})(jQuery);
