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
  // TAB SWITCHER (universal, persistent, no reset after actions)
  // ============================================================
  function snPanelStableId($panel) {
    var id = String($panel.attr('id') || '').replace(/[^A-Za-z0-9_-]/g, '');
    if (id) { return id; }
    var cls = String($panel.attr('class') || '').split(/\s+/).filter(Boolean).join('-').replace(/[^A-Za-z0-9_-]/g, '');
    return cls || 'sn-panel';
  }
  function snPanelTabKey($panel) {
    return 'sn_active_tab_' + snPanelStableId($panel);
  }
  function snSaveActiveClassicTab($panel, target) {
    target = String(target || '').replace(/[^A-Za-z0-9_-]/g, '');
    if (!$panel.length || !target) { return; }
    try { window.localStorage.setItem(snPanelTabKey($panel), target); } catch(e) {}
    $panel.attr('data-sn-active-tab', target);
    $panel.find('form').each(function(){
      var $form = $(this);
      var $hidden = $form.find('input[name="sn_ui_active_tab"]');
      if (!$hidden.length) {
        $hidden = $('<input>', {type:'hidden', name:'sn_ui_active_tab'}).appendTo($form);
      }
      $hidden.val(target);
    });
  }
  function snActivateClassicTab($panel, target) {
    target = String(target || '').replace(/[^A-Za-z0-9_-]/g, '');
	if ($panel.attr('id') === 'sn-seller-panel' && target === 'new-invoice' && String($panel.attr('data-sn-manual-invoice-enabled') || '1') !== '1') { return false; }
    if (!$panel.length || !target || !$panel.find('#sn-tab-' + target).length) { return false; }
    $panel.find('.sn-tab').removeClass('active').attr('aria-selected', 'false');
    $panel.find('.sn-tab-content').removeClass('active').hide();
    $panel.find('.sn-tab[data-tab="' + target + '"]').addClass('active').attr('aria-selected', 'true');
    $panel.find('#sn-tab-' + target).addClass('active').show();
    snSaveActiveClassicTab($panel, target);
    snMovePanelNoticesToActiveTab($panel);
    return true;
  }
  function snMovePanelNoticesToActiveTab($panel) {
    var $active = $panel.children('.sn-tab-content.active, .sn-tab-panel-active, .sn-tab-panel.active').first();
    if (!$active.length) { return; }
    var $notices = $panel.children('.sn-notice, .sn-alert, .notice, .updated, .error').not('.sn-ui-notice-placed');
    if ($notices.length) {
      var $slot = $active.children('.sn-local-notice-slot').first();
      if (!$slot.length) { $slot = $('<div class="sn-local-notice-slot"></div>').prependTo($active); }
      $notices.addClass('sn-ui-notice-placed').prependTo($slot);
    }
  }
  // تب‌های پنل (seller panel, supervisor, invoice)
  window.snClassicTabsReady = true;
  $(document).on('click', '.sn-tab', function (e) {
    var $panel = $(this).closest('.sn-panel, .sn-invoice-page');
    var target = $(this).data('tab');
    if (snActivateClassicTab($panel, target)) { e.preventDefault(); }
  });
  $(function(){
    $('.sn-panel, .sn-invoice-page').has('> .sn-tabs .sn-tab').each(function(){
      var $panel = $(this);
      var saved = '';
      try { saved = window.localStorage.getItem(snPanelTabKey($panel)) || ''; } catch(e) {}
      var hash = String(window.location.hash || '').replace('#', '').replace(/[^A-Za-z0-9_-]/g, '');
      var initial = (hash && $panel.find('#sn-tab-' + hash).length) ? hash : saved;
      if (initial && $panel.find('#sn-tab-' + initial).length) { snActivateClassicTab($panel, initial); }
      else { snSaveActiveClassicTab($panel, $panel.find('.sn-tab.active').first().data('tab') || $panel.find('.sn-tab').first().data('tab')); }
      snMovePanelNoticesToActiveTab($panel);
    });
  });
  $(document).on('submit', '.sn-panel form, .sn-invoice-page form', function(){
    var $panel = $(this).closest('.sn-panel, .sn-invoice-page');
    var target = $panel.attr('data-sn-active-tab') || $panel.find('.sn-tab.active').first().data('tab') || '';
    if (target) { snSaveActiveClassicTab($panel, target); }
  });

  function snPortalTabTarget($btn) {
    var target = $btn.attr('data-sn-tab-target') || $btn.attr('aria-controls') || '';
    if (!target && $btn.attr('href')) {
      var href = String($btn.attr('href') || '');
      if (href.indexOf('#') !== -1) {
        target = href.substring(href.indexOf('#') + 1);
      }
    }
    return String(target || '').replace(/^#/, '').replace(/[^A-Za-z0-9_-]/g, '');
  }

  function snBuildTabUrl(target) {
    target = String(target || '').replace(/^#/, '').replace(/[^A-Za-z0-9_-]/g, '');
    var query = window.location.search || '';
    try {
      var params = new URLSearchParams(query);
      if (target) { params.set('sn_ui_active_tab', target); }
      query = params.toString();
    } catch(e) {}
    return window.location.pathname + (query ? '?' + query : '') + (target ? '#' + target : '');
  }

  function snPanelByTabTarget($scope, target) {
    var $byAttr = $scope.find('.sn-tab-panel').filter(function () {
      return String($(this).attr('data-sn-tab-panel') || '') === target;
    }).first();
    if ($byAttr.length) {
      return $byAttr;
    }
    return $scope.find('.sn-tab-panel').filter(function () {
      return String(this.id || '') === target;
    }).first();
  }

  function snActivatePortalTab($scope, target) {
    target = String(target || '').replace(/^#/, '').replace(/[^A-Za-z0-9_-]/g, '');
    if (!target || !$scope.length) {
      return false;
    }
    var $targetPanel = snPanelByTabTarget($scope, target);
    if (!$targetPanel.length) {
      return false;
    }
    var $buttons = $scope.find('.sn-tab-button, [data-sn-tab-target]');
    var $panels = $scope.find('.sn-tab-panel');
    $buttons.removeClass('sn-tab-button-active active')
      .attr('aria-selected', 'false')
      .attr('tabindex', '-1');
    $buttons.filter(function () {
      return snPortalTabTarget($(this)) === target;
    }).addClass('sn-tab-button-active active')
      .attr('aria-selected', 'true')
      .attr('tabindex', '0');
    $panels.removeClass('sn-tab-panel-active active').attr('hidden', 'hidden');
    $targetPanel.addClass('sn-tab-panel-active active').removeAttr('hidden');
    return true;
  }

  $(function () {
    $('.sn-panel, .sn-portal').has('.sn-tab-nav[data-sn-tabs], .sn-tabs[data-sn-tabs]').each(function () {
      var $scope = $(this);
      $scope.addClass('sn-tabs-initialized');
      $scope.find('.sn-tab-button, [data-sn-tab-target]').attr('role', 'tab');
      $scope.find('.sn-tab-panel').attr('role', 'tabpanel');
      var hash = String(window.location.hash || '').replace('#', '').replace(/[^A-Za-z0-9_-]/g, '');
      var initial = hash && snPanelByTabTarget($scope, hash).length ? hash : '';
      if (!initial) {
        initial = snPortalTabTarget($scope.find('.sn-tab-button-active').first()) || snPortalTabTarget($scope.find('.sn-tab-button, [data-sn-tab-target]').first());
      }
      snActivatePortalTab($scope, initial);
    });
  });

  $(document).on('click', '.sn-tab-button, [data-sn-tab-target]', function (e) {
    var $btn = $(this);
    var target = snPortalTabTarget($btn);
    var $scope = $btn.closest('.sn-panel, .sn-portal, .sn-invoice-page');
    if (!$scope.length) {
      $scope = $('body');
    }
    if (!target || !snActivatePortalTab($scope, target)) {
      return;
    }
    e.preventDefault();
    if (window.history && window.history.replaceState) {
      window.history.replaceState(null, '', snBuildTabUrl(target));
    }
  });

  function snUpdateSelectedCount(selector) {
    if (!selector) { return; }
    $('[data-sn-selected-count="' + selector + '"]').text($(selector + ':checked').length);
  }

  $(document).on('change', '[data-sn-select-all]', function () {
    var selector = $(this).attr('data-sn-select-all');
    if (!selector) { return; }
	var $scope = $(this).closest('form, .sn-card, .sn-panel');
	if (!$scope.length) { $scope = $(document.body); }
	var $targets = $scope.find(selector).filter(':enabled');
	var $visible = $targets.filter(':visible');
	if (this.checked) {
	  // A filtered Select All must never keep hidden rows selected.
	  $targets.prop('checked', false);
	}
	$visible.prop('checked', this.checked).trigger('change');
    snUpdateSelectedCount(selector);
  });

  $(document).on('change', '.sn-hr-workforce-check, .sn-hr-bulk-check', function () {
    snUpdateSelectedCount('.sn-hr-workforce-check');
    snUpdateSelectedCount('.sn-hr-bulk-check');
  });

  $(function () {
    snUpdateSelectedCount('.sn-hr-workforce-check');
    snUpdateSelectedCount('.sn-hr-bulk-check');
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

  const $sellerPanel = $('#sn-seller-panel');
  // Seller pages load assets/js/public-seller.js as the dedicated controller.
  // Do not run the legacy seller block from public.js there; it binds duplicate
  // autosave handlers that re-render the table after every save and collapse the editor.
  if (false && $sellerPanel.length && !(window.snAjax && window.snAjax.asset_key === 'seller')) {
    snEnsureDarkToggle($sellerPanel);
    let allLeadStatuses = [];
    let allLeads        = [];
    let activeFilter    = 'no-status';
    let expandedLeadId  = null;
    let saveTimers      = {};
    let pendingSaves    = {};
    let sellerInvoices  = [];

    // ---- بارگذاری وضعیت‌ها از DB ادمین ----
    $.post(ajax, { action: 'sn_get_lead_statuses', nonce: nonce }, function (res) {
      if (res.success && res.statuses && res.statuses.length) {
        allLeadStatuses = res.statuses;
      }
      renderLeadFilterBar();
      loadLeads();
    });

    loadInvoices();

    // ---- رندر نوار فیلتر (همیشه از DB میخونه) ----
    function renderLeadFilterBar() {
      var bar = '<div class="sn-lead-filters" style="display:flex;gap:6px;flex-wrap:wrap;margin-bottom:14px;align-items:center">';
      // بدون وضعیت اول
      var nsActive = activeFilter === 'no-status';
      bar += '<button class="sn-btn sn-btn-sm ' + (nsActive ? 'sn-btn-primary' : 'sn-btn-ghost') + '" data-filter="no-status">📋 بدون وضعیت</button>';
      // وضعیت‌های DB — حتی اگه کسی اون وضعیت رو نداشته باشه، تب نمایش داده میشه
      allLeadStatuses.forEach(function(s) {
        var active = activeFilter === s.label;
        var style  = active
          ? 'background:' + s.color + ';color:#fff;border:none;box-shadow:0 2px 6px ' + s.color + '55'
          : 'background:transparent;border:1.5px solid ' + s.color + ';color:' + s.color;
        bar += '<button class="sn-btn sn-btn-sm" style="' + style + '" data-filter="' + s.label + '">' + s.label + '</button>';
      });
      // همه آخر
      bar += '<button class="sn-btn sn-btn-sm ' + (activeFilter === 'all' ? 'sn-btn-primary' : 'sn-btn-ghost') + '" data-filter="all">همه</button>';
      bar += '</div>';
      $('#sn-leads-filter-bar').html(bar);
    }

    $(document).on('click', '.sn-lead-filters button', function () {
      activeFilter = $(this).data('filter');
      renderLeadFilterBar();
      renderLeadsTable();
    });

    // ---- بارگذاری leads از سرور ----
    function loadLeads() {
      $('#sn-leads-loading').show();
      $('#sn-leads-list').html(snSkeletonRows(5, 3));
      $.post(ajax, { action: 'sn_seller_leads', nonce: nonce }, function (res) {
        $('#sn-leads-loading').hide();
        if (!res.success || !res.leads || !res.leads.length) {
          allLeads = [];
          renderSellerKpis();
          $('#sn-leads-list').html('<p class="sn-notice">هنوز شماره‌ای به شما تخصیص نیافته.</p>');
          return;
        }
        allLeads = res.leads;
        renderSellerKpis();
        renderLeadsTable();
        fillLeadDropdown();
      });
    }

    function fillLeadDropdown() {
      var $sel = $('#sn-lead-select');
      $sel.empty().append('<option value="">— بدون تخصیص —</option>');
      allLeads.forEach(function(l) {
        if (l.status !== 'invoiced') {
          $sel.append('<option value="' + l.id + '" data-phone="' + l.phone + '">' + l.phone + '</option>');
        }
      });
    }


    function renderSellerKpis() {
      var totalLeads = allLeads.length;
      var noStatus = allLeads.filter(function(l){ return !l.lead_status; }).length;
      var statusDone = totalLeads - noStatus;
      var invoiceCount = sellerInvoices.length;
      var paidCount = sellerInvoices.filter(function(i){ return ['paid','approved'].indexOf(String(i.status || i.payment_status || i.invoice_status || '')) !== -1; }).length;
      var revenue = sellerInvoices.reduce(function(sum, i){
        var st = String(i.status || i.payment_status || i.invoice_status || '');
        if (['paid','approved'].indexOf(st) !== -1) return sum + Number(i.product_price || i.amount || 0);
        return sum;
      }, 0);
      var html = '<div class="sn-kpi-grid sn-seller-kpis">' +
        '<div class="sn-kpi-card"><span class="sn-kpi-icon">📞</span><small>کل لیدها</small><strong>' + snFormatNumber(totalLeads) + '</strong><em>شماره‌های تخصیص‌یافته</em></div>' +
        '<div class="sn-kpi-card"><span class="sn-kpi-icon">✅</span><small>پیگیری‌شده</small><strong>' + snFormatNumber(statusDone) + '</strong><em>' + snPercent(statusDone, totalLeads) + ' از کل لیدها</em></div>' +
        '<div class="sn-kpi-card"><span class="sn-kpi-icon">🧾</span><small>پیش‌فاکتور</small><strong>' + snFormatNumber(invoiceCount) + '</strong><em>تبدیل: ' + snPercent(invoiceCount, totalLeads) + '</em></div>' +
        '<div class="sn-kpi-card sn-kpi-money"><span class="sn-kpi-icon">💰</span><small>فروش تاییدشده</small><strong>' + snFormatMoney(revenue) + '</strong><em>' + snFormatNumber(paidCount) + ' پرداخت موفق</em></div>' +
      '</div>';
      var $target = $('#sn-seller-kpi-cards');
      if (!$target.length) {
        $target = $('<div id="sn-seller-kpi-cards" class="sn-kpi-host"></div>');
        var $tab = $('#sn-tab-leads');
        if ($tab.length) $tab.prepend($target); else $sellerPanel.prepend($target);
      }
      $target.html(html);
    }

    // ---- رندر لیست leads — کارت‌های کلیک‌پذیر ----
    function renderLeadsTable() {
      var filtered = allLeads.filter(function(l) {
        if (activeFilter === 'all')       return true;
        if (activeFilter === 'no-status') return !l.lead_status || l.lead_status === '';
        return l.lead_status === activeFilter;
      });

      if (!filtered.length) {
        $('#sn-leads-list').html('<div class="sn-notice sn-info" style="text-align:center;padding:24px">شماره‌ای با این وضعیت یافت نشد.</div>');
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

        // badge وضعیت تماس
        var statusBadge = '';
        if (l.lead_status) {
          var found = allLeadStatuses.find(function(s){ return s.label === l.lead_status; });
          statusBadge = found
            ? '<span style="background:' + found.color + ';color:#fff;padding:1px 7px;border-radius:8px;font-size:.75rem;margin-right:4px">' + found.label + '</span>'
            : '<span style="background:#e2e8f0;color:#475569;padding:1px 7px;border-radius:8px;font-size:.75rem">' + l.lead_status + '</span>';
        }

        // خلاصه اطلاعات مشتری
        var custInfo = [];
        if (l.customer_name) custInfo.push('<strong>' + l.customer_name + '</strong>');
        if (l.province && l.city) custInfo.push(l.province + ' — ' + l.city);
        else if (l.province) custInfo.push(l.province);
        if (l.sales_prediction) custInfo.push('احتمال: ' + l.sales_prediction);
        if (l.note) custInfo.push('📝 ' + l.note);
        var custHtml = custInfo.length
          ? custInfo.join(' | ')
          : '<span style="color:#94a3b8;font-size:.78rem">اطلاعاتی ثبت نشده</span>';

        html += '<tr style="background:' + rowBg + '" id="sn-lead-row-' + l.id + '">' +
          // ستون شماره — کلیک‌پذیر
          '<td>' +
            '<button type="button" class="sn-phone-copy" data-phone="' + l.phone + '" title="کپی شماره">' + l.phone + '</button>' +
            '<button type="button" class="sn-phone-toggle sn-btn sn-btn-sm sn-btn-ghost" data-id="' + l.id + '">مشاهده/ویرایش</button>' +
            '<div class="sn-copy-msg" data-phone="' + l.phone + '"></div>' +
            (statusBadge ? '<div style="margin-top:3px">' + statusBadge + '</div>' : '') +
          '</td>' +
          // تاریخ تخصیص
          '<td style="font-size:.78rem;color:#64748b;white-space:nowrap;vertical-align:top;padding-top:6px">' + toJalali(l.assigned_at) + '</td>' +
          // اطلاعات مشتری خلاصه
          '<td style="font-size:.82rem;color:#475569;vertical-align:top;padding-top:6px">' + custHtml + '</td>' +
        '</tr>' +

        // ---- dropdown row ----
        '<tr id="sn-expand-' + l.id + '" style="display:none">' +
          '<td colspan="3" style="padding:0;border-top:2px solid #3b82f6">' +
            '<div class="sn-lead-editor">' +

              // ردیف اول: اطلاعات مشتری
              '<div class="sn-lead-editor-grid">' +
                '<div class="sn-lead-field sn-lead-field-name"><label>نام مشتری</label>' +
                  '<input type="text" class="sn-cust-name" data-id="' + l.id + '" value="' + (l.customer_name||'').replace(/"/g,'&quot;') + '" placeholder="نام و نام خانوادگی"></div>' +
                '<div class="sn-lead-field"><label>استان</label>' +
                  '<select class="sn-cust-prov" data-id="' + l.id + '">' +
                    '<option value="">انتخاب استان</option>' +
                    Object.keys(SN_CITIES).map(function(p){ return '<option value="' + p + '"' + (l.province===p?' selected':'') + '>' + p + '</option>'; }).join('') +
                  '</select></div>' +
                '<div class="sn-lead-field"><label>شهر</label>' +
                  '<select class="sn-cust-city" data-id="' + l.id + '">' +
                    snBuildCityOptions(l.province, l.city) +
                  '</select></div>' +
                '<div class="sn-lead-field"><label>احتمال فروش</label>' +
                  '<select class="sn-cust-pred sn-auto-save" data-id="' + l.id + '">' +
                    '<option value="">انتخاب کنید</option>' +
                    ['ضعیف','متوسط','بالا','١٠٠٪'].map(function(v){ return '<option value="' + v + '"' + (l.sales_prediction===v?' selected':'') + '>' + v + '</option>'; }).join('') +
                  '</select></div>' +
                '<div class="sn-lead-field"><label>وضعیت تماس</label>' +
                  '<select class="sn-lead-status-select sn-auto-save" data-id="' + l.id + '">' +
                    '<option value="">— تعیین وضعیت —</option>' + statusOptions +
                  '</select></div>' +
                '<div class="sn-lead-field sn-lead-field-note"><label>یادداشت</label>' +
                  '<textarea class="sn-lead-note sn-auto-save" data-id="' + l.id + '" placeholder="خلاصه مکالمه، نیاز مشتری، زمان پیگیری...">' + (l.note||'').replace(/</g,'&lt;') + '</textarea></div>' +
                '<div class="sn-inline-save-state" data-id="' + l.id + '"></div>' +
              '</div>' +

              // ردیف دوم: فقط صدور پیش‌فاکتور
              '<div class="sn-lead-editor-actions">' +
                '<div><button class="sn-btn sn-btn-sm sn-btn-primary sn-use-lead" data-phone="' + l.phone + '" data-id="' + l.id + '" ' +
                  'style="background:#7c3aed;border-color:#7c3aed">📄 صدور پیش‌فاکتور</button></div>' +
              '</div>' +
            '</div>' +
          '</td>' +
        '</tr>';
      });

      html += '</tbody></table>';
      $('#sn-leads-list').html(html);

      // ست کردن مقدار فعلی وضعیت تماس
      filtered.forEach(function(l) {
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

    function collectLeadFormData(id) {
      return {
        customer_name: $('.sn-cust-name[data-id="' + id + '"]').val().trim(),
        province: $('.sn-cust-prov[data-id="' + id + '"]').val(),
        city: $('.sn-cust-city[data-id="' + id + '"]').val(),
        sales_prediction: $('.sn-cust-pred[data-id="' + id + '"]').val(),
        note: $('.sn-lead-note[data-id="' + id + '"]').val(),
        lead_status: $('select.sn-lead-status-select[data-id="' + id + '"]').val()
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
      if (!lead) return;
      lead.customer_name = data.customer_name;
      lead.province = data.province;
      lead.city = data.city;
      lead.sales_prediction = data.sales_prediction;
      lead.note = data.note;
      lead.lead_status = data.lead_status;
    }

    function saveLeadData(id, options) {
      options = options || {};
      if (pendingSaves[id]) {
        pendingSaves[id].abort();
      }
      var payload = collectLeadFormData(id);
      setLeadSaveState(id, 'saving', options.savingText || 'در حال ذخیره...');
      pendingSaves[id] = $.ajax({
        url: ajax,
        type: 'POST',
        data: {
          action: 'sn_save_customer_info',
          nonce: nonce,
          lead_id: id,
          customer_name: payload.customer_name,
          province: payload.province,
          city: payload.city,
          sales_prediction: payload.sales_prediction,
          note: payload.note,
          lead_status: payload.lead_status
        }
      }).done(function(r) {
        if (r && r.success) {
          syncLeadToMemory(id, payload);
          renderLeadFilterBar();
          renderLeadsTable();
          setLeadSaveState(id, 'saved', options.savedText || 'ذخیره شد');
          setTimeout(function(){ setLeadSaveState(id, 'idle', ''); }, 1500);
        } else {
          setLeadSaveState(id, 'error', (r && r.message) ? r.message : 'خطا در ذخیره');
        }
      }).fail(function(xhr, status) {
        if (status !== 'abort') {
          setLeadSaveState(id, 'error', 'خطا در ارتباط با سرور');
        }
      }).always(function() {
        delete pendingSaves[id];
      });
    }

    function queueLeadAutoSave(id, delay) {
      delay = typeof delay === 'number' ? delay : 700;
      clearTimeout(saveTimers[id]);
      saveTimers[id] = setTimeout(function() {
        saveLeadData(id);
      }, delay);
    }

    // toggle dropdown شماره
    $(document).on('click', '.sn-phone-toggle', function() {
      var id = $(this).data('id');
      var $expand = $('#sn-expand-' + id);
      var $allExpands = $('[id^="sn-expand-"]').not($expand);
      $allExpands.hide();
      if ($expand.is(':visible')) {
        $expand.hide();
        expandedLeadId = null;
      } else {
        $expand.show();
        expandedLeadId = String(id);
      }
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

    // تغییر استان → آپدیت شهرها + ذخیره خودکار
    $(document).on('change', '.sn-cust-prov', function() {
      var id = $(this).data('id');
      var prov = $(this).val();
      $('.sn-cust-city[data-id="' + id + '"]').html(snBuildCityOptions(prov, ''));
      queueLeadAutoSave(id, 200);
    });

    // تغییر شهر / پیش‌بینی فروش / وضعیت تماس = ذخیره فوری
    $(document).on('change', '.sn-cust-city, .sn-cust-pred, .sn-lead-status-select', function() {
      var id = $(this).data('id');
      queueLeadAutoSave(id, 200);
    });

    // نام و یادداشت = ذخیره debounce
    $(document).on('input', '.sn-cust-name, .sn-lead-note', function() {
      var id = $(this).data('id');
      setLeadSaveState(id, 'saving', 'در انتظار ذخیره...');
      queueLeadAutoSave(id, 700);
    });

    $(document).on('keydown', '.sn-lead-note, .sn-cust-name', function (e) {
      if (e.key === 'Enter' && (!$(this).hasClass('sn-lead-note') || e.ctrlKey || e.metaKey)) {
        e.preventDefault();
        var id = $(this).data('id');
        clearTimeout(saveTimers[id]);
        saveLeadData(id, { savingText: 'در حال ذخیره...', savedText: 'ذخیره شد' });
      }
    });

    // دکمه صدور پیش‌فاکتور در لیست شماره‌ها
    $(document).on('click', '.sn-use-lead', function () {
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

      var hasOpenEditor = $('.sn-cust-name[data-id="' + id + '"]').length > 0;
      if (hasOpenEditor) {
        saveLeadData(id, { savingText: 'در حال ذخیره قبل از صدور...', savedText: 'اطلاعات ذخیره شد' });
        setTimeout(openInvoiceForm, 350);
      } else {
        openInvoiceForm();
      }
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
      $('#sn-invoices-loading').show();
      $('#sn-invoices-list').html(snSkeletonRows(5, 7));
      $.post(ajax, { action: 'sn_seller_invoices', nonce: nonce }, function (res) {
        $('#sn-invoices-loading').hide();
        if (!res.success || !res.invoices || !res.invoices.length) {
          sellerInvoices = [];
          renderSellerKpis();
          $('#sn-invoices-list').html('<p class="sn-notice">هنوز فاکتوری صادر نشده.</p>');
          return;
        }
        sellerInvoices = res.invoices || [];
        renderSellerKpis();
        const statusMap = { pending: 'در انتظار پرداخت', pre_invoice: 'پیش‌فاکتور', receipt_uploaded: 'نیاز به بررسی فیش', pending_financial_approval: 'نیاز به بررسی فیش', paid: 'پرداخت‌شده', approved: 'تایید شده', partial_paid: 'پیش‌پرداخت تایید شده', payment_archived: 'بایگانی انصرافی', rejected: 'رد شده', cancelled: 'لغو' };
        let html = '<div class="sn-table-wrap"><table class="sn-table"><thead><tr><th>کد</th><th>مشتری</th><th>موبایل</th><th>محصول</th><th>مبلغ</th><th>وضعیت</th><th>تاریخ</th></tr></thead><tbody>';
        res.invoices.forEach(function (inv) {
          html += `<tr>
            <td><code>${inv.invoice_code}</code></td>
            <td>${inv.customer_name}</td>
            <td>${inv.customer_phone}</td>
            <td>${inv.product_name || inv.product_id}</td>
            <td>${Number(inv.product_price).toLocaleString('fa-IR')} ت</td>
            <td><span class="sn-status sn-status-${inv.status}">${statusMap[inv.status] || inv.status}</span></td>
            <td>${toJalali(inv.created_at)}</td>
          </tr>`;
        });
        html += '</tbody></table></div>';
        $('#sn-sellers-table').html(html);
        $('#sn-invoices-list').html(html);
      });
    }

    // Invoice products and staged payment controls
    function snMoneyNumber(v) {
      return Number(String(v || '').replace(/[۰-۹]/g, function(d){ return '۰۱۲۳۴۵۶۷۸۹'.indexOf(d); }).replace(/[٠-٩]/g, function(d){ return '٠١٢٣٤٥٦٧٨٩'.indexOf(d); }).replace(/[^0-9.]/g, '')) || 0;
    }
    function snInvoiceFormTotal() {
      var total = 0;
      $('#sn-products-multi .sn-product-row').each(function(){
        var price = Number($(this).find('.sn-product-select option:selected').data('price') || 0);
        var qty = Math.max(1, Number($(this).find('.sn-product-qty').val() || 1));
        total += price * qty;
      });
      $('#sn-products-total').text('جمع: ' + Number(total).toLocaleString('fa-IR') + ' تومان').data('total', total);
      return total;
    }
    function snMaxInvoiceProducts(){ var v=Number($('#sn-seller-panel').attr('data-sn-max-products')||$('#sn-add-product-row').attr('data-max-products')||0); return isFinite(v)&&v>0?Math.floor(v):0; }
    function snSyncProductLimit(){ var max=snMaxInvoiceProducts(), count=$('#sn-products-multi .sn-product-row').length, $b=$('#sn-add-product-row'); if(max>0){$b.prop('disabled',count>=max).attr('title',count>=max?'حداکثر تعداد محصول انتخاب شده است':'');}else{$b.prop('disabled',false).removeAttr('title');} }
    $(document).on('change input', '.sn-product-select,.sn-product-qty', function(){ snInvoiceFormTotal(); snSyncProductLimit(); });
    $(document).on('click', '#sn-add-product-row', function(){
      var max=snMaxInvoiceProducts(), count=$('#sn-products-multi .sn-product-row').length;
      if(max>0 && count>=max){ $('#sn-invoice-notice').html('<div class="sn-notice sn-error">حداکثر '+max+' محصول در هر پیش‌فاکتور قابل انتخاب است.</div>'); snSyncProductLimit(); return; }
      var $first = $('#sn-products-multi .sn-product-row').first();
      var $row = $first.clone(false);
      $row.find('.sn-product-select').removeAttr('id').val('');
      $row.find('.sn-product-qty').val('1');
      $row.find('.sn-remove-product').show();
      $('#sn-products-multi').append($row); snInvoiceFormTotal(); snSyncProductLimit();
    });
    $(document).on('click', '.sn-remove-product', function(){ $(this).closest('.sn-product-row').remove(); snInvoiceFormTotal(); snSyncProductLimit(); });
    $(snSyncProductLimit);
    $(document).on('change', '#sn-payment-plan', function(){
      var partial = $(this).val() === 'partial';
      $('.sn-prepayment-fields').toggle(partial);
      if (!partial) { $('#sn-prepayment-choice').val(''); $('#sn-prepayment-custom').val('').hide(); }
    });
    $(document).on('change', '#sn-prepayment-choice', function(){ $('#sn-prepayment-custom').toggle($(this).val() === 'custom'); });

    // Create invoice
    $(document).on('click', '#sn-create-invoice', function () {
      // The seller-specific bundle owns the authoritative submit flow. This
      // keeps the legacy fallback inert if an optimizer changes script order.
      if (window.snSellerInvoiceAuthoritativeHandler) { return; }
      const $btn = $(this);
      const lead_id = $('#sn-lead-select').val();
      const selectedLead = lead_id ? getLeadById(lead_id) : null;
      const name = ($('#sn-cust-name').val() || '').trim() || (selectedLead && selectedLead.customer_name ? selectedLead.customer_name : '');
      const phone = ($('#sn-cust-phone').val() || '').trim() || (selectedLead && selectedLead.phone ? selectedLead.phone : '');
      let prov = $('#sn-cust-prov').val() || (selectedLead && selectedLead.province ? selectedLead.province : '');
      let city = ($('#sn-cust-city').val() || '').trim() || (selectedLead && selectedLead.city ? String(selectedLead.city).trim() : '');
      var productIds = [], productQtys = [];
      $('#sn-products-multi .sn-product-row').each(function(){ var pid=$(this).find('.sn-product-select').val(); if(pid){productIds.push(pid);productQtys.push(Math.max(1,Number($(this).find('.sn-product-qty').val()||1)));} });
      const prod = productIds.length ? productIds[0] : '';
      const paymentPlan = $('#sn-payment-plan').val() || 'full';
      const invoiceTotal = snInvoiceFormTotal();
      var prepaymentAmount = 0;
      if (paymentPlan === 'partial') {
        var choice = $('#sn-prepayment-choice').val();
        prepaymentAmount = choice === 'custom' ? snMoneyNumber($('#sn-prepayment-custom').val()) : snMoneyNumber(choice);
        if (!choice || prepaymentAmount <= 0 || prepaymentAmount >= invoiceTotal) {
          showNotice('#sn-invoice-notice', 'برای پیش‌پرداخت، مبلغی بیشتر از صفر و کمتر از جمع فاکتور وارد کنید.', 'error'); return;
        }
      }
      if (!city && lead_id && $('.sn-cust-city[data-id="' + lead_id + '"]').length) city = ($('.sn-cust-city[data-id="' + lead_id + '"]').val() || '').trim();
      if (!prov && lead_id && $('.sn-cust-prov[data-id="' + lead_id + '"]').length) prov = $('.sn-cust-prov[data-id="' + lead_id + '"]').val() || '';
      if (!name || !phone || !prod) { showNotice('#sn-invoice-notice', 'لطفاً نام، موبایل و حداقل یک محصول را وارد کنید.', 'error'); return; }
      $btn.prop('disabled', true).text('در حال صدور...');
      var params = new URLSearchParams();
      params.append('action','sn_create_invoice'); params.append('nonce',nonce); params.append('customer_name',name); params.append('customer_phone',phone); params.append('customer_phone_secondary',($('#sn-cust-phone-secondary').val()||'').trim()); params.append('province',prov); params.append('city',city); params.append('product_id',prod); params.append('lead_id',lead_id||''); params.append('payment_plan',paymentPlan); params.append('prepayment_amount',String(prepaymentAmount||0));
      productIds.forEach(function(pid){params.append('product_ids[]',pid);}); productQtys.forEach(function(q){params.append('product_qtys[]',String(q));});
      var xhrObj = new XMLHttpRequest(); xhrObj.open('POST', ajax, true); xhrObj.setRequestHeader('Content-Type','application/x-www-form-urlencoded; charset=UTF-8');
      xhrObj.onload = function(){
        $btn.prop('disabled',false).text('صدور پیش‌فاکتور و ارسال پیامک'); var raw=xhrObj.responseText||'',res=null,m=raw.match(/\{[\s\S]*"success"[\s\S]*\}/); if(m){try{res=JSON.parse(m[0]);}catch(e){}} if(!res){try{res=JSON.parse(raw);}catch(e){}}
        if(res&&res.success){
          $('#sn-cust-name,#sn-cust-phone,#sn-cust-city,#sn-prepayment-custom').val(''); $('#sn-product,#sn-lead-select,#sn-cust-prov,#sn-prepayment-choice').val(''); $('#sn-payment-plan').val('full').trigger('change'); $('#sn-products-multi .sn-product-row:gt(0)').remove(); $('#sn-products-multi .sn-product-qty').val('1'); snInvoiceFormTotal();
          var due = res.current_due_amount || (res.data&&res.data.current_due_amount) || 0;
          showNotice('#sn-invoice-notice','✅ پیش‌فاکتور <strong>'+(res.invoice_code||res.code||(res.data&&res.data.invoice_code)||'')+'</strong> صادر شد. مبلغ این مرحله: '+Number(due).toLocaleString('fa-IR')+' تومان.','success'); loadLeads();loadInvoices();setTimeout(function(){$sellerPanel.find('.sn-tab[data-tab="invoices"]').trigger('click');},2000);
        } else {var msg=(res&&(res.message||(res.data&&res.data.message)))?(res.message||(res.data&&res.data.message)):('کد HTTP: '+xhrObj.status);showNotice('#sn-invoice-notice','❌ '+msg,'error');console.error('Invoice fail. Raw:',raw.substring(0,400));}
      };
      xhrObj.onerror=function(){$btn.prop('disabled',false).text('صدور پیش‌فاکتور و ارسال پیامک');showNotice('#sn-invoice-notice','❌ خطای شبکه','error');}; xhrObj.send(params.toString());
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
  // SUPERVISOR PANEL
  // ============================================================
  const $supPanel = $('#sn-supervisor-panel');
  // Supervisor has a dedicated script (public-supervisor.js).
  // Keep universal helpers from this base file, but do not bind the legacy
  // supervisor data/profile/toggle handlers again; double binding caused
  // profile rows to open and immediately close, and duplicate AJAX actions.
  if ($supPanel.length && !(window.snAjax && window.snAjax.asset_key === 'supervisor')) {
    snEnsureDarkToggle($supPanel);

    function loadSupervisorData() {
      $('#sn-sellers-loading').show();
      $('#sn-sellers-table').html(snSkeletonRows(5, 7));
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
        assignment: $('#sn-summary-assignment').val() || ''
      };
      $.post(ajax, params, function (res) {
        $('#sn-sellers-loading').hide();
        if (!res || !res.success) {
          var msg = (res && res.message) ? res.message : 'خطا در دریافت اطلاعات فروشنده‌ها';
          $('#sn-sellers-table').html('<div class="sn-notice sn-error">' + snEsc(msg) + '</div>');
          $('#sn-sellers-checkboxes').html('<div class="sn-notice sn-error">' + snEsc(msg) + '</div>');
          return;
        }
        renderSupervisorKpis(res);
        res.sellers = Array.isArray(res.sellers) ? res.sellers : [];

        // unassigned count is shown inside KPI cards; no separate header badge in the cleaned supervisor panel.
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
            '<td><button type="button" class="sn-btn sn-btn-sm sn-seller-profile" data-id="' + s.id + '" aria-expanded="false">پروفایل</button> <button type="button" class="sn-btn sn-btn-sm sn-toggle-seller" data-id="' + s.id + '" data-active="' + (s.is_active?1:0) + '">' +
              (s.is_active ? '🔴 غیرفعال کن' : '🟢 فعال کن') + '</button></td>' +
          '</tr>' +
          '<tr class="sn-seller-profile-row" id="sn-seller-profile-row-' + s.id + '" style="display:none;background:#f0f9ff">' +
            '<td colspan="7"><div class="sn-seller-profile-panel" data-loaded="0"></div></td>' +
          '</tr>';
        });
        if (!res.sellers.length) {
          html += '<tr><td colspan="7"><div class="sn-notice">هیچ فروشنده HR مستقیمی برای این سرپرست تعریف نشده است. فروشنده باید در منابع انسانی با سمت فروشنده و مدیر مستقیم همین سرپرست ثبت شود.</div></td></tr>';
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
      });
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
      var v4Owned = Number(summary.v4_owned_items || 0);
      var v4Transferable = Number(summary.v4_transferable_items || 0);
      var v4TransferredOut = Number(summary.v4_transferred_out || 0);
      var paid = Number(summary.paid || 0);
      var activeSellers = Number(summary.hr_sellers_active || sellers.filter(function(s){return !!s.is_active;}).length);
      var html = '<div class="sn-kpi-grid sn-supervisor-kpis">' +
        '<div class="sn-kpi-card"><span class="sn-kpi-icon">👥</span><small>فروشنده‌های HR مستقیم</small><strong>' + snFormatNumber(activeSellers) + '</strong><em>از ' + snFormatNumber(sellers.length) + ' نیرو</em></div>' +
        '<div class="sn-kpi-card"><span class="sn-kpi-icon">📦</span><small>شماره‌های تحویل‌شده V4</small><strong>' + snFormatNumber(v4Owned) + '</strong><em>قابل تخصیص: ' + snFormatNumber(v4Transferable) + '</em></div>' +
        '<div class="sn-kpi-card"><span class="sn-kpi-icon">🔁</span><small>تخصیص‌داده‌شده توسط سرپرست</small><strong>' + snFormatNumber(v4TransferredOut) + '</strong><em>مسیر V4</em></div>' +
        '<div class="sn-kpi-card"><span class="sn-kpi-icon">📞</span><small>سرنخ‌های زنده/قدیمی</small><strong>' + snFormatNumber(totalLeads) + '</strong><em>در بازه انتخابی</em></div>' +
        '<div class="sn-kpi-card sn-kpi-money"><span class="sn-kpi-icon">💳</span><small>فاکتور / پرداخت</small><strong>' + snFormatNumber(invoices) + ' / ' + snFormatNumber(paid) + '</strong><em>فروش: ' + snFormatMoney(sales) + '</em></div>' +
      '</div>';
      var $target = $('#sn-supervisor-kpi-cards');
      if (!$target.length) {
        $target = $('<div id="sn-supervisor-kpi-cards" class="sn-kpi-host"></div>');
        var $tab = $('#sn-tab-sellers');
        if ($tab.length) $tab.prepend($target); else $supPanel.prepend($target);
      }
      $target.html(html);
    }

    loadSupervisorData();

    // جستجو
    $('#sn-seller-search-btn').on('click', loadSupervisorData);
    var snSupFilterTimer = null;
    $('#sn-seller-search').on('input', function(){ clearTimeout(snSupFilterTimer); snSupFilterTimer = setTimeout(loadSupervisorData, 450); });
    $('#sn-seller-search').on('keydown', function(e){ if(e.key==='Enter') loadSupervisorData(); });
    $('#sn-seller-filter-act,#sn-date-from,#sn-date-to,#sn-time-from,#sn-time-to,#sn-summary-seller,#sn-summary-lead-status,#sn-summary-assignment').on('change', loadSupervisorData);
    $('#sn-summary-import-code').on('input', function(){ clearTimeout(snSupFilterTimer); snSupFilterTimer = setTimeout(loadSupervisorData, 450); });
    $(document).on('click', '#sn-summary-filter', loadSupervisorData);
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
        if (res.success) { loadSupervisorData(); }
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
          loadSupervisorData();
          if (mode === 'manual') loadUnassigned();
        } else {
          showNotice('#sn-assign-notice', '❌ ' + res.message, 'error');
        }
      });
    });
  }

  // ============================================================
  const $invPage = $('#sn-invoice-page');
  if ($invPage.length) {
    const initCode   = $invPage.data('code');
    const initResult = $invPage.data('result');

    if (initCode) {
      $('#sn-inv-code').val(initCode);
      if (initResult !== 'success' && initResult !== 'failed') {
        loadInvoice(initCode);
      } else if (initResult !== 'success') {
        loadInvoice(initCode);
      }
    }

    $('#sn-load-invoice').on('click', function () {
      const code = $('#sn-inv-code').val().trim();
      if (!code) return;
      loadInvoice(code);
    });

    function loadInvoice(code) {
      $.post(ajax, { action: 'sn_invoice_info', nonce: nonce, invoice_code: code }, function (res) {
        if (!res.success) {
          alert(res.message || 'فاکتور یافت نشد');
          return;
        }
        const inv = res.invoice;
        const card = res.card;

        $('#sn-inv-display-code').text(inv.code);
        $('#sn-inv-name').text(inv.customer_name);
        $('#sn-inv-phone').text(inv.customer_phone);
        const loc = [inv.province, inv.city].filter(Boolean).join(' — ');
        $('#sn-inv-location').text(loc || '—');
        $('#sn-inv-product').text(inv.product_name);
        $('#sn-inv-price').text(inv.price_fmt);
        $('#sn-inv-status').text(inv.status_label || snFaStatus(inv.status));
        $('#sn-card-number').text(card.number || '—');
        $('#sn-card-owner').text(card.owner || '—');

        if (inv.status === 'paid' || inv.status === 'approved') {
          $('#sn-payment-section').hide();
          $('#sn-inv-paid-msg').show();
        } else if (inv.status === 'receipt_uploaded' || inv.status === 'pending_financial_approval') {
          $('#sn-payment-section').hide();
          $('#sn-inv-paid-msg').show().removeClass('sn-success').addClass('sn-info').text('پرداخت/فیش شما ثبت شده و در انتظار بررسی مالی است.');
        } else if (inv.status === 'rejected') {
          $('#sn-payment-section').show();
          $('#sn-inv-paid-msg').show().removeClass('sn-success').addClass('sn-error').text('پرداخت قبلی رد شده است. می‌توانید دوباره فیش یا اطلاعات واریز را ثبت کنید.');
        } else {
          $('#sn-payment-section').show();
          $('#sn-inv-paid-msg').hide();
        }

        $('#sn-invoice-lookup').hide();
        $('#sn-invoice-detail').show();

        // store code for payment actions
        $invPage.data('active-code', code);
        $invPage.data('active-price', inv.product_price);
      });
    }

    // Online payment
    $('#sn-pay-online').on('click', function () {
      const code = $invPage.data('active-code');
      const $btn = $(this);
      $btn.prop('disabled', true).text('در حال اتصال به درگاه...');
      $.post(ajax, { action: 'sn_pay_online', nonce: nonce, invoice_code: code }, function (res) {
        $btn.prop('disabled', false).text('💳 پرداخت آنلاین (درگاه)');
        if (res.success && res.redirect) {
          window.location.href = res.redirect;
        } else {
          alert(res.message || 'خطا در اتصال به درگاه');
        }
      });
    });

    // Card payment
    $('#sn-pay-card').on('click', function () {
      $('#sn-card-info').slideDown();
      $('#sn-card-manual-fields').slideDown(); if(!$('#sn-card-paid-at').val()) $('#sn-card-paid-at').val(snCurrentJalaliDateTime());
    });


    function snToPersianDigits(v) {
      return String(v).replace(/\d/g, function(d){ return '۰۱۲۳۴۵۶۷۸۹'[d]; });
    }
    function snGregorianToJalali(gy, gm, gd) {
      var g_d_m=[0,31,59,90,120,151,181,212,243,273,304,334];
      var gy2=(gm>2)?(gy+1):gy;
      var days=355666+(365*gy)+Math.floor((gy2+3)/4)-Math.floor((gy2+99)/100)+Math.floor((gy2+399)/400)+gd+g_d_m[gm-1];
      var jy=-1595+(33*Math.floor(days/12053)); days%=12053;
      jy+=4*Math.floor(days/1461); days%=1461;
      if(days>365){jy+=Math.floor((days-1)/365); days=(days-1)%365;}
      var jm=(days<186)?1+Math.floor(days/31):7+Math.floor((days-186)/30);
      var jd=1+((days<186)?(days%31):((days-186)%30));
      return [jy,jm,jd];
    }
    function snCurrentJalaliDateTime() {
      var p=snTehranNowParts(), j=snGregorianToJalali(p.year, p.month, p.day);
      var pad=function(n){return String(n).padStart(2,'0');};
      return snToPersianDigits(j[0]+'/'+pad(j[1])+'/'+pad(j[2])+' '+pad(p.hour)+':'+pad(p.minute));
    }

    // Upload receipt
    $('#sn-upload-receipt').on('click', function () {
      const code = $invPage.data('active-code');
      const file = $('#sn-receipt-file')[0].files[0];
      if (!file) { alert('لطفاً فایل فیش را انتخاب کنید'); return; }

      const fd = new FormData();
      fd.append('action', 'sn_upload_receipt');
      fd.append('nonce', nonce);
      fd.append('invoice_code', code);
      fd.append('amount',window.snPaymentTransactions.number($('#sn-receipt-amount').val())); if(!window.snReceiptFiles.append(fd,$('#sn-receipt-file')[0]))return;

      const $btn = $(this);
      $btn.prop('disabled', true).text('در حال ارسال...');
      $.ajax({
        url: ajax, type: 'POST', data: fd,
        processData: false, contentType: false,
        success: function (res) {
          $btn.prop('disabled', false).text('ارسال فیش');
          if (res.success) {
            $('#sn-card-info').hide();
            $('#sn-payment-section').hide();
            $('<div class="sn-notice sn-success">✅ ' + res.message + '</div>').insertBefore('#sn-payment-section');
            loadInvoice(code);
          } else {
            alert(res.message || 'خطا در آپلود');
          }
        },
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
    $(sel).html('<div class="sn-notice sn-' + (type || 'info') + '">' + msg + '</div>');
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
    if (!/^\d{4}$/.test(from) || !/^\d{4}$/.test(to)) { alert('۴ رقم کارت باید عددی باشد'); return; }
    if (!amount || isNaN(String(amount).replace(/,/g,''))) { alert('مبلغ باید عدد باشد'); return; }
    var $btn = $(this).prop('disabled', true).text('در حال ثبت...');
    $.post(ajax, {action:'sn_submit_manual_payment', nonce: nonce, invoice_code: code, card_from: from, card_to: to, amount: amount, paid_at: paidAt}, function(res){
      $btn.prop('disabled', false).text('ثبت اطلاعات واریز');
      if (res && res.success) { $('#sn-payment-section').hide(); $('<div class="sn-notice sn-success">✅ '+res.message+'</div>').insertBefore('#sn-payment-section'); if (typeof loadInvoice === 'function') loadInvoice(code); }
      else alert((res && res.message) || 'خطا در ثبت اطلاعات واریز');
    }).fail(function(xhr){ $btn.prop('disabled', false).text('ثبت اطلاعات واریز'); alert('خطای سرور: '+xhr.status); });
  });

  function snFaStatus(st){ var m={pre_invoice:'پیش‌فاکتور',pending:'در انتظار پرداخت',pending_payment:'در انتظار پرداخت',receipt_uploaded:'نیاز به بررسی فیش',pending_financial_approval:'در انتظار تایید مالی',approved:'تایید شده',paid:'پرداخت‌شده',rejected:'رد شده',cancelled:'لغوشده',assigned:'تخصیص داده شده'}; return m[st]||st||'—'; }
  function snSupervisorInvoiceTarget(){
    var $t = $('#sn-supervisor-invoices-list');
    return $t.length ? $t : $('#sn-supervisor-invoice-list');
  }
  function loadSupervisorInvoices(q){
    var $target = snSupervisorInvoiceTarget();
    if(!$target.length) return;
    $target.html('در حال بارگذاری پیش‌فاکتورها...');
    $.post(ajax,{action:'sn_supervisor_invoices',nonce:nonce,q:q||''},function(res){
      if(!res||!res.success){ $target.html('❌ '+snEsc((res&&res.message)||'خطا در دریافت پیش‌فاکتورها')); return; }
      var rows=res.items||[];
      var html='<table class="sn-table"><thead><tr><th>کد</th><th>مشتری</th><th>فروشنده</th><th>مبلغ</th><th>پرداخت</th><th>وضعیت</th><th>چت بیاوین</th><th>فیش سرپرست</th></tr></thead><tbody>';
      if(!rows.length) html+='<tr><td colspan="8">موردی یافت نشد.</td></tr>';
      rows.forEach(function(i){ var p=i.project&&i.project.membership_id?'<button type="button" class="sn-btn sn-btn-sm sn-btn-primary sn-project-chat-btn" data-membership="'+snEsc(i.project.membership_id)+'" data-item="'+snEsc((i.project.items&&i.project.items[0]&&i.project.items[0].id)||'')+'">چت با بیاوین</button>':'—'; html+='<tr><td><code>'+snEsc(i.invoice_code||'')+'</code></td><td>'+snEsc(i.customer_name||'')+'<br><small>'+snEsc(i.customer_phone||'')+'</small></td><td>'+snEsc(i.seller_name||'—')+'</td><td>'+snEsc(i.product_price||0)+'</td><td>'+snEsc(i.pay_method_label||'—')+'</td><td>'+snEsc(i.status_label||snFaStatus(i.status))+'</td><td>'+p+'</td><td><input type="text" inputmode="decimal" class="sn-legacy-transfer-amount" data-id="'+i.id+'" placeholder="مبلغ این تراکنش (تومان)"><input type="file" multiple class="sn-supervisor-receipt-file" data-id="'+i.id+'" accept="image/*,application/pdf"><button type="button" class="sn-btn sn-btn-sm sn-supervisor-upload-receipt" data-id="'+i.id+'">آپلود فیش</button><div class="sn-supervisor-upload-msg" data-id="'+i.id+'"></div></td></tr>'; });
      html+='</tbody></table>'; $target.html(html);
    }).fail(function(xhr){ $target.html('❌ خطای سرور: '+xhr.status); });
  }
  $(document).on('click','.sn-tab[data-tab="invoices"]',function(){loadSupervisorInvoices($('#sn-supervisor-invoice-search').val());});
  $(document).on('input','#sn-supervisor-invoice-search',function(){clearTimeout(window.snSupInvTimer);var q=$(this).val();window.snSupInvTimer=setTimeout(function(){loadSupervisorInvoices(q);},450);});
  if(snSupervisorInvoiceTarget().length) loadSupervisorInvoices('');

  // 1.0.75: manager report rendering guard. This IIFE cannot see helpers defined in earlier closures,
  // so using toJalali/snEsc from another block can throw ReferenceError after the total count updates.

  function snEsc(value) {
    return String(value === null || value === undefined ? '' : value)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/\"/g, '&quot;')
      .replace(/'/g, '&#039;');
  }

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
  $(document).on('click','.sn-supervisor-upload-receipt',function(e){ e.preventDefault(); var id=$(this).data('id'), file=$('.sn-supervisor-receipt-file[data-id="'+id+'"]').get(0).files[0]; if(!file){alert('لطفاً فایل فیش را انتخاب کنید');return;} var fd=new FormData(); fd.append('action','sn_supervisor_upload_receipt'); fd.append('nonce',nonce); fd.append('invoice_id',id); fd.append('amount',window.snPaymentTransactions.number($('.sn-legacy-transfer-amount[data-id="'+id+'"]').val())); if(!window.snReceiptFiles.append(fd,$('.sn-supervisor-receipt-file[data-id="'+id+'"]').get(0)))return; var $btn=$(this).prop('disabled',true).text('در حال آپلود...'); $.ajax({url:ajax,type:'POST',data:fd,processData:false,contentType:false,success:function(res){$btn.prop('disabled',false).text('آپلود فیش');$('.sn-supervisor-upload-msg[data-id="'+id+'"]').html(res&&res.success?'✅ '+snEsc(res.message||'انجام شد'):'❌ '+snEsc((res&&res.message)||'خطا')); if(res&&res.success) loadSupervisorInvoices($('#sn-supervisor-invoice-search').val());},error:function(xhr){$btn.prop('disabled',false).text('آپلود فیش');alert('خطای سرور: '+xhr.status);}}); });
  $(document).on('click','.sn-fin-approve',function(e){e.preventDefault();var id=$(this).data('id'),$btn=$(this),$row=$btn.closest('tr');$btn.prop('disabled',true).text('در حال تایید...');$.post(ajax,{action:'sn_financial_approve_payment',nonce:nonce,invoice_id:id},function(res){if(res&&res.success){$row.find('td').eq(5).text((res.status_label||'تایید شده'));$row.find('td').last().html('<span class="sn-notice sn-success">تایید شد</span>');}else{$btn.prop('disabled',false).text('تایید');alert((res&&res.message)||'خطا');}}).fail(function(xhr){$btn.prop('disabled',false).text('تایید');alert('خطای سرور: '+xhr.status);});});
  $(document).on('click','.sn-fin-reject',function(e){e.preventDefault();var id=$(this).data('id'),reason=prompt('دلیل رد پرداخت را وارد کنید:');if(!reason)return;var $btn=$(this),$row=$btn.closest('tr');$btn.prop('disabled',true).text('در حال رد...');$.post(ajax,{action:'sn_financial_reject_payment',nonce:nonce,invoice_id:id,reason:reason},function(res){if(res&&res.success){$row.find('td').eq(5).text((res.status_label||'رد شده'));$row.find('td').last().html('<span class="sn-notice sn-error">رد شد</span>');}else{$btn.prop('disabled',false).text('رد');alert((res&&res.message)||'خطا');}}).fail(function(xhr){$btn.prop('disabled',false).text('رد');alert('خطای سرور: '+xhr.status);});});
})(jQuery);

/* HR hierarchy assignment selector filter */
(function(){
  'use strict';
  document.addEventListener('input', function(event){
    var input = event.target && event.target.closest ? event.target.closest('input[data-sn-filter-select]') : null;
    if (!input) return;
    var targetId = input.getAttribute('data-sn-filter-select') || '';
    var select = targetId ? document.getElementById(targetId) : null;
    if (!select) return;
    var needle = (input.value || '').toLowerCase().trim();
    Array.prototype.forEach.call(select.options, function(option){
      var text = (option.textContent || '').toLowerCase();
      option.hidden = needle !== '' && text.indexOf(needle) === -1;
    });
  });
})();


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
    var now=currentJalali(), years='', months='', days='', hours='', mins='', rounded=Math.min(55, Math.floor(now.mi/5)*5);
    for(var y=now.jy-1;y<=now.jy+1;y++) years+='<option value="'+y+'" '+(y===now.jy?'selected':'')+'>'+toFaDigits(y)+'</option>';
    for(var m=1;m<=12;m++) months+='<option value="'+m+'" '+(m===now.jm?'selected':'')+'>'+toFaDigits(m)+'</option>';
    for(var d=1;d<=31;d++) days+='<option value="'+d+'" '+(d===now.jd?'selected':'')+'>'+toFaDigits(d)+'</option>';
    for(var h=0;h<24;h++) hours+='<option value="'+h+'" '+(h===now.hh?'selected':'')+'>'+toFaDigits(pad(h))+'</option>';
    for(var i=0;i<60;i+=5) mins+='<option value="'+i+'" '+(rounded===i?'selected':'')+'>'+toFaDigits(pad(i))+'</option>';
    $host.html('<div class="sn-jalali-row"><select id="sn-paid-jy">'+years+'</select><span>/</span><select id="sn-paid-jm">'+months+'</select><span>/</span><select id="sn-paid-jd">'+days+'</select><span class="sn-time-sep">ساعت</span><select id="sn-paid-hh">'+hours+'</select><span>:</span><select id="sn-paid-mi">'+mins+'</select></div><small>تاریخ و ساعت واریز را به شمسی انتخاب کنید.</small>').data('ready',1);
    syncJalaliPicker();
  }
  function syncJalaliPicker(){
    if(!$('#sn-card-paid-at-picker').length) return;
    var jy=$('#sn-paid-jy').val(), jm=pad($('#sn-paid-jm').val()), jd=pad($('#sn-paid-jd').val()), hh=pad($('#sn-paid-hh').val()), mi=pad($('#sn-paid-mi').val());
    $('#sn-card-paid-at').val(toFaDigits(jy+'/'+jm+'/'+jd+' '+hh+':'+mi));
  }
  $(document).on('change','#sn-paid-jy,#sn-paid-jm,#sn-paid-jd,#sn-paid-hh,#sn-paid-mi',syncJalaliPicker);
  $(document).on('click','#sn-pay-card,#sn-card-manual-toggle',function(){ $('#sn-card-manual-fields').slideDown(); setTimeout(buildJalaliPicker,30); });
  $(function(){ buildJalaliPicker(); });

  $(document).off('click', '#sn-submit-manual-payment').on('click', '#sn-submit-manual-payment', function(e){
    e.preventDefault(); buildJalaliPicker(); syncJalaliPicker();
    var code = $('#sn-invoice-page').data('active-code') || $('#sn-inv-code').val();
    var from = toEnDigits($('#sn-card-from4').val());
    var to = toEnDigits($('#sn-card-to4').val());
    var amount = toEnDigits($('#sn-card-amount').val()).replace(/,/g,'');
    if(!/^\d{4}$/.test(from) || !/^\d{4}$/.test(to)){ alert('۴ رقم آخر کارت باید عددی باشد'); return; }
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
    for(var i=0;i<60;i+=5) mins+='<option value="'+i+'" '+(i===now.mi?'selected':'')+'>'+faDigits(pad(i))+'</option>';
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
    var fd=new FormData(); fd.append('action','sn_upload_receipt'); fd.append('nonce',nonce); fd.append('invoice_code',code); fd.append('amount',window.snPaymentTransactions.number($('#sn-receipt-amount').val())); if(!window.snReceiptFiles.append(fd,$('#sn-receipt-file')[0]))return;
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
    if(!/^\d{4}$/.test(from) || !/^\d{4}$/.test(to)){ alert('۴ رقم آخر کارت باید عددی باشد'); return; }
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
})(jQuery);

/* SN Operational UI Hardening - final pass */
(function(){
  'use strict';
  function ready(fn){ if(document.readyState !== 'loading'){ fn(); } else { document.addEventListener('DOMContentLoaded', fn); } }
  function cleanId(v){ return String(v || '').replace(/^#/, '').replace(/[^A-Za-z0-9_-]/g, ''); }
  function buildTabUrl(target){
    target = cleanId(target);
    var query = window.location.search || '';
    try{
      var params = new URLSearchParams(query);
      if(target){ params.set('sn_ui_active_tab', target); }
      query = params.toString();
    }catch(e){}
    return location.pathname + (query ? '?' + query : '') + (target ? '#' + target : '');
  }
  function scrollToTabPanel(el, smooth){
    if(!el || !el.getBoundingClientRect) return;
    var adminbar = document.getElementById('wpadminbar');
    var offset = (adminbar ? adminbar.offsetHeight : 0) + 18;
    var top = el.getBoundingClientRect().top + (window.pageYOffset || document.documentElement.scrollTop || 0) - offset;
    if(top < 0) top = 0;
    try{ window.scrollTo({top: top, behavior: smooth ? 'smooth' : 'auto'}); }
    catch(e){ window.scrollTo(0, top); }
  }

  function panelKey(panel){
    if(!panel) return 'sn-panel';
    var id = cleanId(panel.getAttribute('id') || '');
    if(id) return id;
    var cls = cleanId((panel.className || '').toString().split(/\s+/).filter(Boolean).join('-'));
    return cls || 'sn-panel';
  }
  function currentTarget(panel){
    if(!panel) return '';
    var active = panel.querySelector('.sn-tab.active[data-tab], .sn-tab-button-active[data-sn-tab-target], .sn-tab-button.active[data-sn-tab-target], [data-sn-tab-target].active');
    if(active){ return cleanId(active.getAttribute('data-tab') || active.getAttribute('data-sn-tab-target') || active.getAttribute('aria-controls') || ''); }
    var activePanel = panel.querySelector('.sn-tab-content.active, .sn-tab-panel-active, .sn-tab-panel.active:not([hidden])');
    if(activePanel){ return cleanId(activePanel.getAttribute('data-sn-tab-panel') || activePanel.id || '').replace(/^sn-tab-/, ''); }
    return '';
  }
  function saveTarget(panel, target){
    target = cleanId(target);
    if(!panel || !target) return;
    panel.setAttribute('data-sn-active-tab', target);
    try{ localStorage.setItem('sn_active_tab_' + panelKey(panel), target); }catch(e){}
    panel.querySelectorAll('form').forEach(function(form){
      var h = form.querySelector('input[name="sn_ui_active_tab"]');
      if(!h){ h = document.createElement('input'); h.type = 'hidden'; h.name = 'sn_ui_active_tab'; form.appendChild(h); }
      h.value = target;
    });
  }
  function noticeSlot(panel){
    if(!panel) return null;
    var active = panel.querySelector('.sn-tab-content.active, .sn-tab-panel-active, .sn-tab-panel.active:not([hidden])');
    if(!active) return null;
    var slot = active.querySelector(':scope > .sn-local-notice-slot');
    if(!slot){ slot = document.createElement('div'); slot.className = 'sn-local-notice-slot'; active.insertBefore(slot, active.firstChild); }
    return slot;
  }
  function moveNotices(panel){
    var slot = noticeSlot(panel);
    if(!slot || !panel) return;
    Array.prototype.slice.call(panel.children).forEach(function(el){
      if(!el || el.classList.contains('sn-ui-notice-placed')) return;
      if(el.matches && el.matches('.sn-notice, .sn-alert, .notice, .updated, .error')){
        el.classList.add('sn-ui-notice-placed');
        slot.insertBefore(el, slot.firstChild);
      }
    });
  }
  function activatePortal(panel, target, options){
    target = cleanId(target);
    if(!panel || !target) return false;
    var targetPanel = null;
    panel.querySelectorAll('.sn-tab-panel').forEach(function(p){
      if(!targetPanel && cleanId(p.getAttribute('data-sn-tab-panel') || p.id) === target){ targetPanel = p; }
    });
    if(!targetPanel) return false;
    panel.querySelectorAll('.sn-tab-button, [data-sn-tab-target]').forEach(function(btn){
      var bt = cleanId(btn.getAttribute('data-sn-tab-target') || btn.getAttribute('aria-controls') || '');
      var on = bt === target;
      btn.classList.toggle('sn-tab-button-active', on);
      btn.classList.toggle('active', on);
      btn.setAttribute('aria-selected', on ? 'true' : 'false');
      if(btn.tagName === 'BUTTON' && !btn.getAttribute('type')) btn.setAttribute('type','button');
    });
    panel.querySelectorAll('.sn-tab-panel').forEach(function(p){
      var on = p === targetPanel;
      p.classList.toggle('sn-tab-panel-active', on);
      p.classList.toggle('active', on);
      if(on) p.removeAttribute('hidden'); else p.setAttribute('hidden','hidden');
    });
    saveTarget(panel, target);
    moveNotices(panel);
    if(options && options.scroll){ scrollToTabPanel(targetPanel, true); }
    return true;
  }
  function activateClassic(panel, target, options){
    target = cleanId(target);
	if(panel && panel.id === 'sn-seller-panel' && target === 'new-invoice' && String(panel.getAttribute('data-sn-manual-invoice-enabled') || '1') !== '1') return false;
    if(!panel || !target) return false;
    var targetPanel = panel.querySelector('#sn-tab-' + CSS.escape(target));
    if(!targetPanel) return false;
    panel.querySelectorAll('.sn-tab[data-tab]').forEach(function(btn){
      var on = cleanId(btn.getAttribute('data-tab')) === target;
      btn.classList.toggle('active', on);
      btn.setAttribute('aria-selected', on ? 'true' : 'false');
      if(btn.tagName === 'BUTTON' && !btn.getAttribute('type')) btn.setAttribute('type','button');
    });
    panel.querySelectorAll('.sn-tab-content').forEach(function(p){
      var on = p === targetPanel;
      p.classList.toggle('active', on);
      p.style.display = on ? '' : 'none';
    });
    saveTarget(panel, target);
    moveNotices(panel);
    if(options && options.scroll){ scrollToTabPanel(targetPanel, true); }
    return true;
  }
  function normalizePanels(){
    document.querySelectorAll('.sn-panel').forEach(function(panel){
      if(panel.querySelector(':scope > .sn-tabs, :scope > .sn-panel-toolbar')){
        panel.classList.remove('sn-no-sidebar');
        panel.classList.add('sn-has-sidebar-tabs');
      }
      panel.querySelectorAll('.sn-tab, .sn-tab-button, [data-sn-tab-target]').forEach(function(btn){
        if(btn.tagName === 'BUTTON') btn.setAttribute('type','button');
      });
      var key = 'sn_active_tab_' + panelKey(panel);
      var stored = '';
      try{ stored = localStorage.getItem(key) || ''; }catch(e){}
      var queryTab = '';
      try{ queryTab = new URLSearchParams(window.location.search).get('sn_ui_active_tab') || ''; }catch(e){}
      var hashTab = window.location.hash.replace('#','');
      var explicit = !!(cleanId(queryTab) || cleanId(hashTab));
      var requested = cleanId(queryTab || hashTab || panel.getAttribute('data-sn-active-tab') || stored || '');
      if(requested){
        var activated = activatePortal(panel, requested, {scroll:false}) || activateClassic(panel, requested, {scroll:false});
        if(activated && explicit){ setTimeout(function(){ var active = panel.querySelector('.sn-tab-panel-active, .sn-tab-panel.active:not([hidden]), .sn-tab-content.active'); scrollToTabPanel(active || panel, false); }, 80); }
      }
      else { saveTarget(panel, currentTarget(panel)); moveNotices(panel); }
    });
  }
  document.addEventListener('click', function(e){
    var btn = e.target && e.target.closest ? e.target.closest('.sn-tab[data-tab], .sn-tab-button, [data-sn-tab-target]') : null;
    if(!btn) return;
    var panel = btn.closest('.sn-panel, .sn-invoice-page, .sn-portal');
    var target = cleanId(btn.getAttribute('data-tab') || btn.getAttribute('data-sn-tab-target') || btn.getAttribute('aria-controls') || '');
    if(!panel || !target) return;
    var ok = activatePortal(panel, target, {scroll:true}) || activateClassic(panel, target, {scroll:true});
    if(ok){
      e.preventDefault();
      if(history && history.replaceState){ history.replaceState(null, '', buildTabUrl(target)); }
    }
  }, true);
  document.addEventListener('submit', function(e){
    var form = e.target;
    if(!form || !form.closest) return;
    var panel = form.closest('.sn-panel, .sn-invoice-page, .sn-portal');
    if(!panel) return;
    var target = currentTarget(panel) || panel.getAttribute('data-sn-active-tab') || '';
    if(target) saveTarget(panel, target);
  }, true);
  ready(function(){ normalizePanels(); setTimeout(normalizePanels, 60); setTimeout(normalizePanels, 300); });
  window.snNormalizeOperationalPanels = normalizePanels;
})();


// SN v2.3 sidebar hamburger/collapse final fix
(function(){
  'use strict';
  if (window.snPortalShellReady) { return; }
  function ready(fn){
    if(document.readyState === 'loading') document.addEventListener('DOMContentLoaded', fn);
    else fn();
  }
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
      if(nodes[i].classList && (nodes[i].classList.contains('sn-panel-toolbar') || nodes[i].classList.contains('sn-tabs'))){
        return nodes[i];
      }
    }
    return null;
  }
  function setState(panel, collapsed){
    var sidebar = directSidebar(panel);
    if(!sidebar) return;
    panel.classList.toggle('sn-sidebar-collapsed', !!collapsed);
    panel.classList.add('sn-sidebar-collapsible');
    var btn = sidebar.querySelector(':scope > .sn-sidebar-toggle') || sidebar.querySelector('.sn-sidebar-toggle');
    if(btn){
      btn.setAttribute('aria-expanded', collapsed ? 'false' : 'true');
      btn.setAttribute('title', collapsed ? 'باز کردن منو' : 'بستن منو');
      btn.setAttribute('aria-label', collapsed ? 'باز کردن منوی پنل' : 'بستن منوی پنل');
    }
    try{ window.localStorage.setItem(panelKey(panel), collapsed ? '1' : '0'); }catch(e){}
  }
  function setupPanel(panel){
    var sidebar = directSidebar(panel);
    if(!sidebar) return;
    panel.classList.add('sn-sidebar-collapsible');
    if(!sidebar.querySelector(':scope > .sn-sidebar-toggle') && !sidebar.querySelector('.sn-sidebar-toggle')){
      var btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'sn-sidebar-toggle';
      btn.innerHTML = '☰';
      sidebar.insertBefore(btn, sidebar.firstChild);
    }
    var stored = '';
    try{ stored = window.localStorage.getItem(panelKey(panel)) || ''; }catch(e){}
    // Default is open so labels are readable; user can collapse to icons only.
    setState(panel, stored === '1');
  }
  function setupAll(){
    var panels = document.querySelectorAll('.sn-panel.sn-has-sidebar-tabs, .sn-panel');
    panels.forEach(function(panel){
      if(directSidebar(panel)) setupPanel(panel);
    });
  }
  document.addEventListener('click', function(e){
    var btn = e.target && e.target.closest ? e.target.closest('.sn-sidebar-toggle') : null;
    if(!btn) return;
    var panel = btn.closest('.sn-panel');
    if(!panel) return;
    e.preventDefault();
    e.stopPropagation();
    setState(panel, !panel.classList.contains('sn-sidebar-collapsed'));
  }, true);
  ready(function(){ setupAll(); setTimeout(setupAll, 80); setTimeout(setupAll, 350); });
  window.snSetupSidebarToggles = setupAll;
})();

// SN mobile full-screen drawer navigation
(function(){
  'use strict';
  if (window.snPortalShellReady) { return; }
  function ready(fn){
    if(document.readyState === 'loading') document.addEventListener('DOMContentLoaded', fn);
    else fn();
  }
  function hasSidebar(panel){
    return !!(panel && panel.querySelector && (panel.querySelector(':scope > .sn-tabs') || panel.querySelector(':scope > .sn-panel-toolbar')));
  }
  function isMobile(){ return window.matchMedia && window.matchMedia('(max-width: 900px)').matches; }
  function setOpen(panel, open){
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
  function ensureButton(panel){
    if(!hasSidebar(panel)) return;
    if(!panel.querySelector(':scope > .sn-mobile-sidebar-toggle')){
      var btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'sn-mobile-sidebar-toggle';
      btn.setAttribute('aria-expanded','false');
      btn.setAttribute('aria-label','باز کردن منوی پنل');
      btn.setAttribute('title','باز کردن منو');
      btn.innerHTML = '☰';
      panel.insertBefore(btn, panel.firstChild);
    }
  }
  function setup(){
    document.querySelectorAll('.sn-panel').forEach(function(panel){
      if(hasSidebar(panel)) ensureButton(panel);
      if(!isMobile()) setOpen(panel, false);
    });
  }
  document.addEventListener('click', function(e){
    var mobileBtn = e.target && e.target.closest ? e.target.closest('.sn-mobile-sidebar-toggle') : null;
    if(mobileBtn){
      var panel = mobileBtn.closest('.sn-panel');
      if(panel){
        e.preventDefault();
        e.stopPropagation();
        setOpen(panel, !panel.classList.contains('sn-mobile-sidebar-open'));
      }
      return;
    }
    var innerBtn = e.target && e.target.closest ? e.target.closest('.sn-sidebar-toggle') : null;
    if(innerBtn && isMobile()){
      var p = innerBtn.closest('.sn-panel');
      if(p){
        e.preventDefault();
        e.stopPropagation();
        setOpen(p, !p.classList.contains('sn-mobile-sidebar-open'));
      }
      return;
    }
    var tab = e.target && e.target.closest ? e.target.closest('.sn-panel .sn-tab[data-tab], .sn-panel .sn-tab-button, .sn-panel [data-sn-tab-target]') : null;
    if(tab && isMobile()){
      var panel2 = tab.closest('.sn-panel');
      if(panel2 && panel2.classList.contains('sn-mobile-sidebar-open')){
        setTimeout(function(){ setOpen(panel2, false); }, 70);
      }
    }
  }, true);
  document.addEventListener('keydown', function(e){
    if(e.key !== 'Escape') return;
    var open = document.querySelector('.sn-panel.sn-mobile-sidebar-open');
    if(open) setOpen(open, false);
  });
  window.addEventListener('resize', function(){
    if(!isMobile()){
      document.querySelectorAll('.sn-panel.sn-mobile-sidebar-open').forEach(function(panel){ setOpen(panel, false); });
    }
  });
  ready(function(){ setup(); setTimeout(setup, 100); setTimeout(setup, 400); });
  window.snSetupMobileDrawerNav = setup;
})();

/* SN HR lookup AJAX saves: positions/levels without full page reload */
(function(){
  'use strict';
  function ready(fn){ if(document.readyState === 'loading') document.addEventListener('DOMContentLoaded', fn); else fn(); }
  function cfg(){ return window.snAjax || window.snData || {}; }
  function ajaxUrl(){ return (cfg().ajaxurl || window.ajaxurl || '/wp-admin/admin-ajax.php'); }
  function esc(s){ return String(s == null ? '' : s).replace(/[&<>'"]/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;',"'":'&#039;','"':'&quot;'}[c];}); }
  function msgText(code){
    var map={
      hr_position_saved:'سمت ذخیره شد.', hr_position_deleted:'سمت حذف شد.', hr_position_delete_failed:'حذف سمت انجام نشد.', hr_position_update_failed:'ذخیره سمت انجام نشد.',
      hr_level_saved:'سطح ذخیره شد.', hr_level_deleted:'سطح حذف شد.', hr_level_delete_failed:'حذف سطح انجام نشد.', hr_level_update_failed:'ذخیره سطح انجام نشد.', hr_level_activated:'سطح فعال شد.', hr_level_deactivated:'سطح غیرفعال شد.'
    };
    return map[code] || code || 'عملیات انجام شد.';
  }
  function notify(form, message, ok){
    var box = form.querySelector('.sn-ajax-form-msg');
    if(!box){ box=document.createElement('div'); box.className='sn-ajax-form-msg'; form.appendChild(box); }
    box.textContent = message;
    box.classList.toggle('sn-success', !!ok);
    box.classList.toggle('sn-error', !ok);
    clearTimeout(box._snTimer);
    box._snTimer=setTimeout(function(){ box.textContent=''; box.classList.remove('sn-success','sn-error'); }, 4500);
  }
  function addHidden(form, name, value){
    var input=document.createElement('input'); input.type='hidden'; input.name=name; input.value=value; form.appendChild(input); return input;
  }
  function currentNonce(form){ var n=form.querySelector('input[name="_wpnonce"]'); return n ? n.value : ''; }
  function newPositionCard(row, nonce){
    var id=Number(row.id||0), label=esc(row.label||''), slug=esc(row.slug||''), desc=esc(row.description||''), active=Number(row.is_active||0)===1;
    return '<form method="post" action="'+esc((window.snAdminPostUrl||'/wp-admin/admin-post.php'))+'" class="sn-hr-edit-card sn-hr-ajax-lookup-form">'
      +'<input type="hidden" name="position_id" value="'+id+'"><div class="sn-hr-edit-head"><strong>'+label+'</strong><span class="sn-hr-badge">#'+id+'</span><span class="sn-hr-badge">'+slug+'</span><span class="sn-hr-muted">سمت سفارشی</span></div>'
      +'<div class="sn-hr-form-grid"><label>عنوان<input type="text" name="label" value="'+label+'"></label><label>توضیح<input type="text" name="description" value="'+desc+'"></label><label class="sn-hr-check"><input type="checkbox" name="is_active" value="1" '+(active?'checked':'')+'> فعال</label><div class="sn-hr-actions"><input type="hidden" name="action" value="sn_hr_save_position"><input type="hidden" name="_wpnonce" value="'+esc(nonce)+'"><button class="sn-btn" type="submit" name="position_status_action" value="save">ذخیره</button><button class="sn-btn sn-btn-danger" type="submit" name="position_status_action" value="delete" data-sn-confirm="این سمت حذف شود؟ نیروهای وابسته بدون سمت می‌شوند.">حذف</button></div></div></form>';
  }
  function newLevelCard(row, nonce){
    var id=Number(row.id||0), label=esc(row.label||''), slug=esc(row.slug||''), desc=esc(row.description||''), active=Number((row.is_active != null ? row.is_active : row.active)||0)===1;
    return '<div class="sn-hr-edit-card"><div class="sn-hr-edit-head"><strong>'+label+'</strong><span class="sn-hr-badge">#'+id+'</span><span class="sn-hr-badge">'+slug+'</span></div>'
      +'<form method="post" class="sn-hr-inline-form sn-hr-ajax-lookup-form"><input type="hidden" name="level_id" value="'+id+'"><div class="sn-hr-form-grid"><label>عنوان<input type="text" name="label" value="'+label+'"></label><label>توضیح<input type="text" name="description" value="'+desc+'"></label><label class="sn-hr-check"><input type="hidden" name="is_active" value="0"><input type="checkbox" name="is_active" value="1" '+(active?'checked':'')+'> فعال</label><div class="sn-hr-actions"><input type="hidden" name="action" value="sn_hr_save_level"><input type="hidden" name="_wpnonce" value="'+esc(nonce)+'"><button class="sn-btn" type="submit" name="level_status_action" value="save">ذخیره</button></div></div></form>'
      +'<form method="post" class="sn-hr-inline-form sn-hr-toggle-form sn-hr-ajax-lookup-form"><input type="hidden" name="action" value="sn_hr_save_level"><input type="hidden" name="level_id" value="'+id+'"><input type="hidden" name="_wpnonce" value="'+esc(nonce)+'"><button class="sn-btn sn-btn-secondary" type="submit" name="level_status_action" value="'+(active?'deactivate':'activate')+'">'+(active?'غیرفعال‌سازی':'فعال‌سازی')+'</button><button class="sn-btn sn-btn-danger" type="submit" name="level_status_action" value="delete" data-sn-confirm="این سطح حذف شود؟ نیروهای وابسته بدون سطح می‌شوند.">حذف</button></form></div>';
  }
  function resetNewForm(form){
    if(!form.classList.contains('sn-hr-new-card')) return;
    form.querySelectorAll('input[type="text"], textarea').forEach(function(i){ i.value=''; });
    form.querySelectorAll('input[type="checkbox"]').forEach(function(i){ i.checked=true; });
  }
  ready(function(){
    document.addEventListener('click', function(e){
      var b=e.target && e.target.closest ? e.target.closest('.sn-hr-ajax-lookup-form [data-sn-confirm], .sn-hr-ajax-lookup-form button[onclick]') : null;
      if(!b) return;
      var text=b.getAttribute('data-sn-confirm') || (b.getAttribute('name') && String(b.value).indexOf('delete')!==-1 ? 'این مورد حذف شود؟' : '');
      if(text && !window.confirm(text)){ e.preventDefault(); e.stopPropagation(); return; }
      if(text){ b.setAttribute('data-sn-confirmed','1'); setTimeout(function(){ b.removeAttribute('data-sn-confirmed'); }, 1200); }
      if(b.hasAttribute('onclick')) b.removeAttribute('onclick');
    }, true);
    document.addEventListener('submit', function(e){
      var form=e.target;
      if(!form || !form.matches || !form.matches('form.sn-hr-ajax-lookup-form')) return;
      var actionInput=form.querySelector('input[name="action"]');
      var action=actionInput ? actionInput.value : '';
      if(action !== 'sn_hr_save_position' && action !== 'sn_hr_save_level') return;
      e.preventDefault();
      var submitter=e.submitter || document.activeElement;
      if(submitter && submitter.name && String(submitter.value).indexOf('delete')!==-1 && !submitter.getAttribute('data-sn-confirmed')){
        var confirmText=submitter.getAttribute('data-sn-confirm') || 'این مورد حذف شود؟';
        if(!window.confirm(confirmText)) return;
      }
      var fd=new FormData(form);
      fd.set('action', action);
      fd.set('sn_ajax_lookup','1');
      if(submitter && submitter.name){ fd.set(submitter.name, submitter.value || ''); }
      var buttons=form.querySelectorAll('button, input[type="submit"]');
      buttons.forEach(function(b){ b.disabled=true; });
      form.classList.add('sn-ajax-saving');
      fetch(ajaxUrl(), {method:'POST', credentials:'same-origin', body:fd})
        .then(function(r){ return r.json(); })
        .then(function(res){
          var data=(res && (res.data || {})) || {};
          var code=data.message || '';
          if(!res || !res.success){ throw new Error(msgText(code || 'خطا در ذخیره‌سازی')); }
          notify(form, msgText(code), true);
          var row=data.row || {};
          if(data.deleted){
            var card=form.closest('.sn-hr-edit-card'); if(card) card.remove();
            return;
          }
          var id=Number(data.id || row.id || 0);
          if(id){
            var idInput=form.querySelector('input[name="position_id"], input[name="level_id"]'); if(idInput) idInput.value=String(id);
          }
          var head=form.closest('.sn-hr-edit-card') ? form.closest('.sn-hr-edit-card').querySelector('.sn-hr-edit-head strong') : form.querySelector('.sn-hr-edit-head strong');
          if(head && row.label) head.textContent=row.label;
          if(action === 'sn_hr_save_level' && form.classList.contains('sn-hr-toggle-form')){
            var toggleBtn=form.querySelector('button[name="level_status_action"]');
            var activeNow=Number((row.is_active != null ? row.is_active : row.active)||0)===1;
            if(toggleBtn){ toggleBtn.value=activeNow?'deactivate':'activate'; toggleBtn.textContent=activeNow?'غیرفعال‌سازی':'فعال‌سازی'; }
            var editCard=form.closest('.sn-hr-edit-card');
            if(editCard){ var cb=editCard.querySelector('input[type="checkbox"][name="is_active"]'); if(cb) cb.checked=activeNow; }
          }
          if(form.classList.contains('sn-hr-new-card') && row.id){
            var list=form.closest('.sn-hr-edit-list');
            if(list){
              var html = action==='sn_hr_save_position' ? newPositionCard(row, currentNonce(form)) : newLevelCard(row, currentNonce(form));
              form.insertAdjacentHTML('beforebegin', html);
            }
            resetNewForm(form);
          }
        })
        .catch(function(err){ notify(form, err && err.message ? err.message : 'خطا در ذخیره‌سازی', false); })
        .finally(function(){ buttons.forEach(function(b){ b.disabled=false; }); form.classList.remove('sn-ajax-saving'); });
    }, true);
  });
})();

(function(){
  function ready(fn){ if(document.readyState==='loading') document.addEventListener('DOMContentLoaded', fn); else fn(); }
  function ajaxUrl(){ return (window.snAjax && window.snAjax.ajaxurl) || window.ajaxurl || '/wp-admin/admin-ajax.php'; }
  function parseJsonResponse(response){
    return response.text().then(function(raw){
      var text=String(raw || '').replace(/^\uFEFF/, '').trim();
      var data=null;
      if(text){
        try{ data=JSON.parse(text); }
        catch(e){
          var first=text.indexOf('{'), last=text.lastIndexOf('}');
          if(first!==-1 && last>first){ try{ data=JSON.parse(text.slice(first,last+1)); }catch(ignore){} }
        }
      }
      if(!data || typeof data!=='object'){
        var detail=text.replace(/<[^>]*>/g,' ').replace(/\s+/g,' ').trim().slice(0,240);
        var err=new Error(detail || ('پاسخ نامعتبر سرور ('+String(response.status||0)+')'));
        err.httpStatus=response.status||0;
        throw err;
      }
      return data;
    });
  }
  function text(el){ return (el && (el.innerText || el.textContent) || '').toLowerCase().trim(); }
  function notifyNear(form, message, ok){
    var box=form.querySelector('.sn-ajax-form-msg');
    if(!box){ box=document.createElement('div'); box.className='sn-ajax-form-msg'; form.insertBefore(box, form.firstChild); }
    box.className='sn-ajax-form-msg sn-alert ' + (ok ? 'sn-success sn-alert-success' : 'sn-error sn-alert-error');
    box.textContent=message || (ok ? 'انجام شد.' : 'خطا در انجام عملیات.');
    box.removeAttribute('hidden');
  }
  function notifySaving(form, message){
    var box=form.querySelector('.sn-ajax-form-msg');
    if(!box){ box=document.createElement('div'); box.className='sn-ajax-form-msg'; form.insertBefore(box, form.firstChild); }
    box.className='sn-ajax-form-msg sn-alert sn-info';
    box.style.display='block';
    box.textContent=message || 'در حال ثبت...';
    box.removeAttribute('hidden');
  }
  function visibleRows(table){ return Array.prototype.slice.call(table.querySelectorAll('tbody tr')).filter(function(r){ return r.style.display !== 'none'; }); }
  var rowCache = new WeakMap();
  function invalidateRow(row){ if(row) rowCache.delete(row); }
  function cachedRow(row){ var cache=rowCache.get(row); if(!cache){ cache={cells:{}}; rowCache.set(row,cache); } return cache; }
  function cellValue(row, idx){
    if(!row) return '';
    var cache=cachedRow(row);
    if(Object.prototype.hasOwnProperty.call(cache.cells,idx)) return cache.cells[idx];
    var cell=row && row.children ? row.children[idx] : null;
    if(!cell) return '';
    var select=cell.querySelector('select');
    if(select){ return cache.cells[idx]=(select.options[select.selectedIndex] ? select.options[select.selectedIndex].text : '').trim(); }
    return cache.cells[idx]=(cell.innerText || cell.textContent || '').replace(/\s+/g,' ').trim();
  }
  function escHtml(value){
    return String(value == null ? '' : value).replace(/[&<>"]/g, function(ch){ return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;'}[ch] || ch; });
  }
  function rowSearchText(row){
    if(!row || !row.children) return '';
    var cache=cachedRow(row); if(cache.search!==undefined) return cache.search;
    var parts=[];
    Array.prototype.slice.call(row.children).forEach(function(cell, idx){
      parts.push(cellValue(row, idx));
      var select=cell.querySelector('select');
      if(select){ Array.prototype.slice.call(select.options || []).forEach(function(opt){ if(opt && opt.selected) parts.push(opt.value || ''); }); }
      cell.querySelectorAll('input[type="hidden"], input[type="text"], input[type="number"], textarea').forEach(function(input){ parts.push(input.value || ''); });
    });
    parts.push(row.getAttribute('data-sn-user-id') || '');
    return cache.search=parts.join(' ').replace(/\s+/g,' ').toLowerCase().trim();
  }
  function updateExcelFilterButtons(table){
    if(!table) return;
    var state=table._snExcelFilterState || {};
    table.querySelectorAll('.sn-hr-excel-filter-btn').forEach(function(btn){
      var idx=btn.getAttribute('data-sn-col');
      var selected=state[idx];
      var active=Array.isArray(selected) && selected.length < uniqueColumnValues(table, parseInt(idx,10)).length;
      btn.classList.toggle('is-filtered', !!active);
    });
  }
  function applyTableFilters(table){
    if(!table) return;
    var global='';
    document.querySelectorAll('.sn-hr-live-search[data-sn-table]').forEach(function(input){
      if(input.getAttribute('data-sn-table') && table.matches(input.getAttribute('data-sn-table'))) global=String(input.value||'').toLowerCase().trim();
    });
    var selects=[];
    document.querySelectorAll('.sn-hr-live-select[data-sn-table]').forEach(function(sel){
      if(sel.getAttribute('data-sn-table') && table.matches(sel.getAttribute('data-sn-table'))) selects.push(sel);
    });
    var excelState=table._snExcelFilterState || {};
    table.querySelectorAll('tbody tr').forEach(function(row){
      var hay=global ? rowSearchText(row) : '';
      var ok=!global || hay.indexOf(global)!==-1;
      selects.forEach(function(sel){
        var val=String(sel.value||'').toLowerCase().trim();
        var idx=parseInt(sel.getAttribute('data-sn-col')||'-1',10);
        if(val && idx>=0 && cellValue(row, idx).toLowerCase().indexOf(val)===-1) ok=false;
      });
      Object.keys(excelState).forEach(function(key){
        var idx=parseInt(key,10), selected=excelState[key];
        if(Array.isArray(selected)){
          var val=cellValue(row, idx) || '—';
          if(selected.indexOf(val)===-1) ok=false;
        }
      });
      row.style.display=ok?'':'none';
	  if(!ok){ row.querySelectorAll('.sn-hr-workforce-check:checked').forEach(function(cb){ cb.checked=false; }); }
    });
    updateExcelFilterButtons(table);
	document.querySelectorAll('[data-sn-selected-count=".sn-hr-workforce-check"]').forEach(function(el){ el.textContent=String(document.querySelectorAll('.sn-hr-workforce-check:checked').length); });
	document.querySelectorAll('[data-sn-select-all=".sn-hr-workforce-check"]').forEach(function(cb){ cb.checked=false; });
  }
  function closeExcelMenu(){
    document.querySelectorAll('.sn-hr-excel-filter-menu').forEach(function(m){ m.remove(); });
    document.querySelectorAll('.sn-hr-excel-filter-btn.is-open').forEach(function(b){ b.classList.remove('is-open'); });
  }
  function uniqueColumnValues(table, idx){
    var vals=[], seen=new Set();
    table.querySelectorAll('tbody tr').forEach(function(row){
      var v=cellValue(row, idx) || '—';
      if(!seen.has(v)){ seen.add(v); vals.push(v); }
    });
    vals.sort(function(a,b){ return String(a).localeCompare(String(b), 'fa'); });
    return vals;
  }
	function sortTableRows(table, idx, direction){
	  if(!table) return;
	  var body=table.querySelector('tbody'); if(!body) return;
	  var rows=Array.prototype.slice.call(body.querySelectorAll('tr'));
	  rows.forEach(function(row, order){ if(!row.hasAttribute('data-sn-original-order')) row.setAttribute('data-sn-original-order', String(order)); });
	  var collator;
	  try{ collator=new Intl.Collator('fa',{numeric:true,sensitivity:'base'}); }catch(e){ collator=null; }
	  rows.sort(function(a,b){
		if(direction==='original') return Number(a.getAttribute('data-sn-original-order')||0)-Number(b.getAttribute('data-sn-original-order')||0);
		var av=cellValue(a,idx), bv=cellValue(b,idx);
		var cmp=collator?collator.compare(av,bv):String(av).localeCompare(String(bv));
		return direction==='desc'?-cmp:cmp;
	  });
	  var fragment=document.createDocumentFragment();
	  rows.forEach(function(row){ fragment.appendChild(row); }); body.appendChild(fragment);
	  table._snSortState=direction==='original'?null:{col:idx,direction:direction};
	  applyTableFilters(table);
	}
  function openExcelMenu(table, idx, btn){
    closeExcelMenu();
    var values=uniqueColumnValues(table, idx);
    var state=table._snExcelFilterState || (table._snExcelFilterState={});
    var selected=Array.isArray(state[idx]) ? state[idx].slice() : values.slice();
    var menu=document.createElement('div'); menu.className='sn-hr-excel-filter-menu'; menu.setAttribute('dir','rtl');
    var label=(btn.closest('th') && (btn.closest('th').getAttribute('data-sn-label') || text(btn.closest('th')))) || 'ستون';
	var html='<div class="sn-hr-excel-filter-title">فیلتر «'+escHtml(label).replace(/▾/g,'').trim()+'»</div>'
	  +'<div class="sn-hr-excel-sort-actions"><button type="button" class="sn-btn sn-btn-secondary sn-hr-excel-sort" data-direction="asc">مرتب‌سازی صعودی</button><button type="button" class="sn-btn sn-btn-secondary sn-hr-excel-sort" data-direction="desc">مرتب‌سازی نزولی</button><button type="button" class="sn-btn sn-btn-ghost sn-hr-excel-sort" data-direction="original">ترتیب اولیه</button></div>'
	  +'<input type="search" class="sn-hr-excel-filter-search" placeholder="جستجو در همین ستون">'
      +'<label class="sn-check sn-hr-excel-select-all"><input type="checkbox" class="sn-hr-excel-filter-all" '+(selected.length===values.length?'checked':'')+'> انتخاب همه</label>'
      +'<div class="sn-hr-excel-filter-values">';
    values.forEach(function(v){ html+='<label class="sn-check sn-hr-excel-value"><input type="checkbox" value="'+escHtml(v)+'" '+(selected.indexOf(v)!==-1?'checked':'')+'> <span>'+escHtml(v)+'</span></label>'; });
    html+='</div><div class="sn-hr-excel-filter-actions"><button type="button" class="sn-btn sn-btn-secondary sn-hr-excel-clear">پاک کردن فیلتر</button></div>';
    menu.innerHTML=html; document.body.appendChild(menu); btn.classList.add('is-open');
    var rect=btn.getBoundingClientRect(); var width=Math.min(320, Math.max(240, rect.width+220));
    menu.style.width=width+'px';
    var left=Math.max(10, Math.min(window.innerWidth-width-10, rect.left - width + rect.width));
    menu.style.left=left+'px'; menu.style.top=(rect.bottom+8)+'px';
    menu._snTable=table; menu._snCol=idx;
  }
  function applyExcelMenuSelection(menu){
    var table=menu && menu._snTable, idx=menu && menu._snCol;
    if(!table || idx == null) return;
    var checked=Array.prototype.slice.call(menu.querySelectorAll('.sn-hr-excel-value input:checked')).map(function(cb){ return cb.value; });
    var allVals=uniqueColumnValues(table, idx);
    table._snExcelFilterState=table._snExcelFilterState||{};
    if(checked.length===allVals.length){ delete table._snExcelFilterState[idx]; }
    else { table._snExcelFilterState[idx]=checked; }
    applyTableFilters(table);
  }
  function installExcelFilters(table){
    if(!table || table.dataset.snExcelFiltersReady==='modern') return;
    table.querySelectorAll('thead .sn-hr-table-filter-row').forEach(function(r){ r.remove(); });
    var headRow=table.querySelector('thead tr'); if(!headRow) return;
	table.querySelectorAll('tbody tr').forEach(function(row, order){ if(!row.hasAttribute('data-sn-original-order')) row.setAttribute('data-sn-original-order', String(order)); });
	Array.prototype.slice.call(headRow.children).forEach(function(th, idx){
      if(idx===0 || idx===headRow.children.length-1) return;
      var labelEl=th.querySelector('.sn-hr-th-label');
      var label=(labelEl ? (labelEl.textContent||'') : (th.getAttribute('data-sn-label') || th.textContent || '')).replace(/[⌄▾]/g,'').trim();
      th.setAttribute('data-sn-label', label);
      var btn=th.querySelector('.sn-hr-excel-filter-btn');
      if(btn){ btn.setAttribute('data-sn-col', String(idx)); return; }
      th.innerHTML='<span class="sn-hr-th-inner"><span class="sn-hr-th-label">'+escHtml(label)+'</span><button type="button" class="sn-hr-excel-filter-btn" data-sn-col="'+idx+'" aria-label="فیلتر '+escHtml(label)+'" title="فیلتر '+escHtml(label)+'"><span aria-hidden="true">▾</span></button></span>';
    });
    table.dataset.snExcelFiltersReady='modern';
  }
  function filterList(input){
    var sel=input.getAttribute('data-sn-list'); if(!sel) return;
    var root=document.querySelector(sel); if(!root) return;
    var q=String(input.value||'').toLowerCase().trim();
    root.querySelectorAll('.sn-hr-bulk-user-row').forEach(function(row){ row.style.display=!q || text(row).indexOf(q)!==-1 ? '' : 'none'; });
  }
  function updateBulkGroups(form){
    var select=form.querySelector('.sn-hr-bulk-action-select'); if(!select) return;
    var active=select.value || 'manager';
    form.querySelectorAll('.sn-hr-bulk-field-group').forEach(function(g){ g.style.display=(g.getAttribute('data-sn-bulk-group')===active)?'':'none'; });
  }
  function syncInlineToHidden(control){
    var row=control.closest('tr'), uid=control.getAttribute('data-sn-user-id') || (row && row.getAttribute('data-sn-user-id')), field=control.getAttribute('data-sn-field');
    if(!row || !uid || !field) return null;
    var hidden=row.querySelector('.sn-hr-row-edit-fields [name="rows['+uid+']['+field+']"]');
    if(hidden){ hidden.value=control.value; if(hidden.tagName==='SELECT') hidden.dispatchEvent(new Event('change', {bubbles:true})); }
    return {row:row, uid:uid, field:field, hidden:hidden};
  }
  function syncRowFromFields(row, fieldRoot){
    if(!row) return;
    var uid=row.getAttribute('data-sn-user-id'); if(!uid) return;
    var root=fieldRoot || row.querySelector('.sn-hr-row-edit-fields');
    row.querySelectorAll('.sn-hr-inline-autosave[data-sn-field]').forEach(function(control){
      var field=control.getAttribute('data-sn-field');
      var hidden=root && root.querySelector('[name="rows['+uid+']['+field+']"]');
      if(hidden && control.value !== hidden.value){ control.value=hidden.value; control.dispatchEvent(new Event('change', {bubbles:false})); }
    });
    var nameField=root && root.querySelector('[data-sn-display-name]');
    if(nameField){ var nameCell=row.children && row.children[2]; if(nameCell){ nameCell.childNodes[0].nodeValue=nameField.value || nameCell.textContent; } }
    invalidateRow(row);
    var table=row.closest('table'); if(table) applyTableFilters(table);
  }
  function rowSaveState(row, message, ok){
    var box=row && row.querySelector('.sn-hr-row-save-state'); if(!box) return;
    box.textContent=message||''; box.classList.toggle('is-ok', !!ok); box.classList.toggle('is-error', ok===false);
    clearTimeout(box._snTimer); if(message){ box._snTimer=setTimeout(function(){ box.textContent=''; box.classList.remove('is-ok','is-error'); }, 3000); }
  }
  // JSON avoids PHP max_input_vars truncation when saving a whole page.
  function packWorkforceRows(fd){
    var rows={}, keys=[];
    fd.forEach(function(value,key){
      var match=/^rows\[(\d+)\]\[([a-z_]+)\]$/.exec(key);
      if(match){ rows[match[1]]=rows[match[1]]||{}; rows[match[1]][match[2]]=value; keys.push(key); }
    });
    keys.forEach(function(key){ fd.delete(key); });
    fd.set('rows_json',JSON.stringify(rows));
  }
  function postHrRequest(fd){
    var controller=typeof AbortController==='function' ? new AbortController() : null;
    var timer=controller ? setTimeout(function(){controller.abort();},45000) : null;
    return fetch(ajaxUrl(),{method:'POST',credentials:'same-origin',body:fd,signal:controller ? controller.signal : undefined})
      .then(parseJsonResponse).finally(function(){if(timer) clearTimeout(timer);});
  }
  function saveSingleWorkforceRow(form, row, done, fieldRoot){
    if(!form || !row) return;
    var uid=row.getAttribute('data-sn-user-id'); if(!uid) return;
    if(row._snSaveRunning){
      var queued=row._snSaveQueued || {callbacks:[]};
      if(done) queued.callbacks.push(done);
      queued.root=fieldRoot; row._snSaveQueued=queued; return;
    }
    row._snSaveRunning=true;
    var requestSettled=false;
    var action=(form.querySelector('input[name="action"]')||{}).value || 'sn_hr_bulk_inline_profile_update';
    var root=fieldRoot || row.querySelector('.sn-hr-row-edit-fields');
    if(!root){ root=form.querySelector('.sn-hr-profile-modal:not([hidden]) .sn-hr-row-edit-fields'); }
    var fd=new FormData();
    fd.set('action', action);
    var nonce=form.querySelector('input[name="_wpnonce"]'); if(nonce) fd.set('_wpnonce', nonce.value);
    fd.set('run_mode','apply'); fd.set('confirm_apply','APPLY'); fd.set('save_scope','selected'); fd.set('reason','hr_single_profile_save');
    var modalForRoot = root && root.closest ? root.closest('.sn-hr-profile-modal') : null;
    var includeComp = !!(modalForRoot && modalForRoot.querySelector('.sn-hr-save-compensation-fields:checked'));
    if(includeComp){ fd.set('save_compensation_fields','1'); }
    fd.append('user_ids[]', uid); fd.append('visible_user_ids[]', uid);
    if(root){ root.querySelectorAll('[name^="rows['+uid+']"]').forEach(function(input){
      if((input.type==='checkbox' || input.type==='radio') && !input.checked) return;
      fd.append(input.name, input.value);
    }); }
    // Access selectors live in the table, outside the modal's editable fields.
    appendWorkforceAccessFields(fd, row);
    packWorkforceRows(fd);
    row.classList.add('sn-row-saving'); rowSaveState(row, 'در حال ذخیره...', true);
    postHrRequest(fd).then(function(res){
      requestSettled=true;
      var ok=!!(res && res.success), msg=(res && (res.message || (res.data && res.data.message))) || (ok?'ذخیره شد.':'خطا در ذخیره‌سازی');
      if(ok && !row._snSaveQueued){ syncRowFromFields(row, root); rowSaveState(row, 'ذخیره شد', true); }
      else { rowSaveState(row, msg, false); }
      if(done){ if(row._snSaveQueued) row._snSaveQueued.callbacks.push(done); else done(ok,msg); }
    }).catch(function(err){
      var msg=err && err.name==='AbortError' ? 'پاسخ ذخیره دریافت نشد؛ پیش از ذخیره دوباره وضعیت را بررسی کنید.' : 'خطا در ارتباط با سرور.';
      rowSaveState(row,msg,false); if(done) done(false,msg);
      if(row._snSaveQueued) row._snSaveQueued.callbacks.forEach(function(cb){cb(false,msg);});
      row._snSaveQueued=null;
    }).finally(function(){
      row._snSaveRunning=false; row.classList.remove('sn-row-saving');
      var queued=row._snSaveQueued; row._snSaveQueued=null;
      if(requestSettled && queued){ saveSingleWorkforceRow(form,row,function(ok,msg){queued.callbacks.forEach(function(cb){cb(ok,msg);});},queued.root); }
    });
  }
  function appendWorkforceAccessFields(fd, row){
    var uid=row && row.getAttribute('data-sn-user-id');
    if(!uid) return;
    row.querySelectorAll('.sn-hr-inline-autosave[data-sn-field]').forEach(function(control){
      var field=control.getAttribute('data-sn-field');
      if(['manual_invoice_access','extra_number_request_access'].indexOf(field)===-1) return;
      fd.set('rows['+uid+']['+field+']', control.value);
    });
  }
  function getStoredBulkIds(){ try{ var raw=window.localStorage.getItem('sn_hr_bulk_selected_ids'); var ids=raw?JSON.parse(raw):[]; return Array.isArray(ids)?ids.map(String):[]; }catch(e){ return []; } }
  function applyBulkIds(ids){
    ids=(ids||[]).map(String);
    document.querySelectorAll('.sn-hr-bulk-check').forEach(function(cb){ cb.checked=ids.indexOf(String(cb.value))!==-1; });
    if(window.snUpdateSelectedCount){ window.snUpdateSelectedCount('.sn-hr-bulk-check'); }
    document.querySelectorAll('[data-sn-selected-count=".sn-hr-bulk-check"]').forEach(function(el){ el.textContent=String(ids.length); });
    var badge=document.querySelector('.sn-hr-bulk-selected-summary'); if(badge){ badge.textContent=ids.length ? (ids.length+' نیرو از مدیریت نیروها منتقل شد') : 'نیرویی منتقل نشده است'; }
  }
  function selectedWorkforceIds(){
    return Array.prototype.slice.call(document.querySelectorAll('.sn-hr-workforce-check:checked')).map(function(i){ return String(i.value); });
  }
  function transferSelectedToBulk(){
    var ids=selectedWorkforceIds();
    try{ window.localStorage.setItem('sn_hr_bulk_selected_ids', JSON.stringify(ids)); }catch(e){}
    applyBulkIds(ids); setTimeout(function(){applyBulkIds(ids);},80); setTimeout(function(){applyBulkIds(ids);},300);
  }
  function openBulkPasswordModal(){
    var modal=document.querySelector('.sn-hr-bulk-password-modal'); if(!modal) return;
    var ids=selectedWorkforceIds();
    var count=modal.querySelector('.sn-hr-bulk-password-count'); if(count) count.textContent=String(ids.length);
    var state=modal.querySelector('.sn-hr-bulk-password-state'); if(state){ state.textContent=ids.length ? '' : 'هیچ نیرویی انتخاب نشده است.'; state.classList.toggle('is-error', !ids.length); state.classList.remove('is-ok'); }
    modal.hidden=false; modal._snSelectedIds=ids; document.body.classList.add('sn-hr-modal-open');
    var first=modal.querySelector('.sn-hr-bulk-password-new'); if(first) setTimeout(function(){ first.focus(); }, 50);
  }
  function closeBulkPasswordModal(){
    document.querySelectorAll('.sn-hr-bulk-password-modal:not([hidden])').forEach(function(modal){ modal.hidden=true; modal._snSelectedIds=[]; });
    if(!document.querySelector('.sn-hr-profile-modal:not([hidden])')) document.body.classList.remove('sn-hr-modal-open');
  }
  function initHrServerTabs(){
    var panel=document.querySelector('[data-sn-hr-server-tabs]'); if(!panel) return;
    function tabLink(id){return Array.prototype.find.call(panel.querySelectorAll('[data-sn-hr-tab]'),function(link){return link.getAttribute('data-sn-hr-tab')===id;});}
    function navigate(id){
      var link=tabLink(id); if(!link) return;
      var url=new URL(link.href,location.href);
      if(id==='hr-bulk'){
        transferSelectedToBulk();
        var ids=selectedWorkforceIds();
        if(ids.length) url.searchParams.set('sn_hr_selected',ids.join(',')); else url.searchParams.delete('sn_hr_selected');
      }
      location.assign(url.href);
    }
    document.addEventListener('click',function(e){
      var link=e.target.closest('[data-sn-open-tab],a[href^="#hr-"], [data-sn-hr-tab]');
      if(!link || !panel.contains(link) || e.ctrlKey || e.metaKey || e.shiftKey || e.altKey || (e.button!==undefined && e.button!==0)) return;
      var id=link.getAttribute('data-sn-open-tab') || link.getAttribute('data-sn-hr-tab') || (link.getAttribute('href')||'').slice(1);
      if(!tabLink(id)) return;
      e.preventDefault(); e.stopImmediatePropagation(); navigate(id);
    },true);
    // Older bookmarks and POST redirects still use #hr-... .
    var hash=location.hash.slice(1);
    if(tabLink(hash) && hash!==panel.getAttribute('data-sn-active-tab')) location.replace(tabLink(hash).href);
  }
  ready(function(){
    document.querySelectorAll('.sn-hr-excel-filter-table').forEach(installExcelFilters);
    initHrServerTabs();
    if(location.hash==='#hr-bulk' || document.querySelector('.sn-tab-button-active[data-sn-tab-target="hr-bulk"], [data-sn-hr-server-tabs][data-sn-active-tab="hr-bulk"]')){ setTimeout(function(){ applyBulkIds(getStoredBulkIds()); }, 120); }
    document.addEventListener('input', function(e){
      var input=e.target;
      invalidateRow(input.closest('tr[data-sn-user-id]') || (input.closest('.sn-hr-profile-modal')||{})._snRow);
      if(input.matches('.sn-hr-live-search[data-sn-table]')){ var table=document.querySelector(input.getAttribute('data-sn-table')); clearTimeout(input._snFilterTimer); input._snFilterTimer=setTimeout(function(){applyTableFilters(table);},120); }
      if(input.matches('.sn-hr-live-search[data-sn-list]')) filterList(input);
      if(input.matches('.sn-hr-excel-filter-search')){ var q=String(input.value||'').toLowerCase().trim(); var menu=input.closest('.sn-hr-excel-filter-menu'); if(menu){ menu.querySelectorAll('.sn-hr-excel-value').forEach(function(label){ label.style.display=!q || text(label).indexOf(q)!==-1 ? '' : 'none'; }); } }
    });
    document.addEventListener('change', function(e){
      invalidateRow(e.target.closest('tr[data-sn-user-id]') || (e.target.closest('.sn-hr-profile-modal')||{})._snRow);
      if(e.target.matches('.sn-hr-live-select[data-sn-table]')){ var table=document.querySelector(e.target.getAttribute('data-sn-table')); applyTableFilters(table); }
      if(e.target.matches('.sn-hr-bulk-action-select')) updateBulkGroups(e.target.closest('form'));
      if(e.target.matches('.sn-hr-inline-autosave[data-sn-field]')){ var info=syncInlineToHidden(e.target); if(info && info.row){ saveSingleWorkforceRow(e.target.closest('form'), info.row); } }
    });
    document.querySelectorAll('.sn-hr-bulk-form').forEach(updateBulkGroups);
    document.addEventListener('click', function(e){
      var filterBtn=e.target.closest('.sn-hr-excel-filter-btn');
      if(filterBtn){
        e.preventDefault(); e.stopPropagation();
        var table=filterBtn.closest('table'); var idx=parseInt(filterBtn.getAttribute('data-sn-col')||'-1',10);
        if(table && idx>=0) openExcelMenu(table, idx, filterBtn);
        return;
      }
      var menu=e.target.closest('.sn-hr-excel-filter-menu');
      if(menu){
        var search=e.target.closest('.sn-hr-excel-filter-search');
        if(search){ return; }
        var all=e.target.closest('.sn-hr-excel-filter-all');
        if(all){ menu.querySelectorAll('.sn-hr-excel-value input').forEach(function(cb){ cb.checked=all.checked; }); applyExcelMenuSelection(menu); return; }
        var valueCb=e.target.closest('.sn-hr-excel-value input');
        if(valueCb){ applyExcelMenuSelection(menu); var allBox=menu.querySelector('.sn-hr-excel-filter-all'); if(allBox){ var allVals=menu.querySelectorAll('.sn-hr-excel-value input'); var checkedVals=menu.querySelectorAll('.sn-hr-excel-value input:checked'); allBox.checked=allVals.length===checkedVals.length; } return; }
        var apply=e.target.closest('.sn-hr-excel-apply');
        if(apply){ applyExcelMenuSelection(menu); closeExcelMenu(); return; }
        var clearFilter=e.target.closest('.sn-hr-excel-clear');
        if(clearFilter){ var table2=menu._snTable, idx2=menu._snCol; if(table2 && table2._snExcelFilterState){ delete table2._snExcelFilterState[idx2]; applyTableFilters(table2); } closeExcelMenu(); return; }
		var sortButton=e.target.closest('.sn-hr-excel-sort');
		if(sortButton){ sortTableRows(menu._snTable, menu._snCol, sortButton.getAttribute('data-direction')||'asc'); closeExcelMenu(); return; }
        return;
      }
      if(!e.target.closest('.sn-hr-excel-filter-menu')) closeExcelMenu();
      var openBulkPass=e.target.closest('.sn-hr-open-bulk-password-modal');
      if(openBulkPass){ e.preventDefault(); openBulkPasswordModal(); return; }
      if(e.target.closest('[data-sn-hr-bulk-password-close]')){ e.preventDefault(); closeBulkPasswordModal(); return; }
      var submitBulkPass=e.target.closest('.sn-hr-submit-bulk-password');
      if(submitBulkPass){
        e.preventDefault();
        var modal=submitBulkPass.closest('.sn-hr-bulk-password-modal'); if(!modal) return;
        var ids=selectedWorkforceIds();
        if(!ids.length && Array.isArray(modal._snSelectedIds)) ids=modal._snSelectedIds;
        var state=modal.querySelector('.sn-hr-bulk-password-state');
        var newPass=modal.querySelector('.sn-hr-bulk-password-new');
        var repPass=modal.querySelector('.sn-hr-bulk-password-repeat');
        if(state){ state.textContent='در حال ثبت رمز...'; state.classList.remove('is-error','is-ok'); }
        if(!ids.length){ if(state){ state.textContent='هیچ نیرویی انتخاب نشده است.'; state.classList.add('is-error'); } return; }
        var fd=new FormData();
        fd.set('action','sn_hr_bulk_set_password'); fd.set('_wpnonce', modal.getAttribute('data-sn-nonce') || ''); fd.set('password_scope','selected'); fd.set('confirm_apply','APPLY');
        fd.set('new_password', newPass ? newPass.value : ''); fd.set('new_password_repeat', repPass ? repPass.value : '');
        ids.forEach(function(id){ fd.append('user_ids[]', id); });
        submitBulkPass.disabled=true;
        postHrRequest(fd).then(function(res){
          var ok=!!(res && res.success); var msg=(res && (res.message || (res.data && res.data.message))) || (ok?'رمز نیروهای انتخاب‌شده ثبت شد.':'ثبت رمز انجام نشد.');
          if(state){ state.textContent=msg; state.classList.toggle('is-ok', ok); state.classList.toggle('is-error', !ok); }
          if(ok){ if(newPass) newPass.value=''; if(repPass) repPass.value=''; setTimeout(closeBulkPasswordModal, 800); }
        }).catch(function(err){ if(state){ state.textContent=(err && err.message) ? err.message : 'خطا در ارتباط با سرور.'; state.classList.add('is-error'); state.classList.remove('is-ok'); } }).finally(function(){ submitBulkPass.disabled=false; });
        return;
      }
      var passBtn=e.target.closest('.sn-hr-set-password-inline');
      if(passBtn){
        e.preventDefault();
        var card=passBtn.closest('.sn-hr-password-card'), state=card && card.querySelector('.sn-hr-password-state');
        var fields=card && card.closest('.sn-hr-row-edit-fields');
        var uid=card && (card.getAttribute('data-sn-hr-password-user') || (fields && fields.getAttribute('data-sn-profile-title') || '').match(/#(\d+)/));
        if(uid && typeof uid!=='string') uid=uid[1];
        var newPass=card && card.querySelector('.sn-hr-password-new');
        var repPass=card && card.querySelector('.sn-hr-password-repeat');
        var gen=card && card.querySelector('.sn-hr-password-generate');
        var sms=card && card.querySelector('.sn-hr-password-sms');
        if(state){ state.textContent='در حال ثبت رمز...'; state.classList.remove('is-error','is-ok'); }
        var fd=new FormData();
        fd.set('action','sn_hr_set_user_password'); fd.set('_wpnonce', passBtn.getAttribute('data-sn-nonce') || ''); fd.set('user_id', String(uid||''));
        if(gen && gen.checked) fd.set('generate_random','1');
        if(sms && sms.checked) fd.set('send_sms','1');
        fd.set('new_password', newPass ? newPass.value : ''); fd.set('new_password_repeat', repPass ? repPass.value : '');
        passBtn.disabled=true;
        postHrRequest(fd).then(function(res){
          var ok=!!(res && res.success); var msg=(res && (res.message || (res.data && res.data.message))) || (ok?'رمز ثبت شد.':'ثبت رمز انجام نشد.');
          if(state){ state.textContent=msg; state.classList.toggle('is-ok', ok); state.classList.toggle('is-error', !ok); }
          if(ok && newPass && repPass){ newPass.value=''; repPass.value=''; if(gen) gen.checked=false; }
        }).catch(function(err){ if(state){ state.textContent=(err && err.message) ? err.message : 'خطا در ارتباط با سرور.'; state.classList.add('is-error'); state.classList.remove('is-ok'); } }).finally(function(){ passBtn.disabled=false; });
        return;
      }
      var clear=e.target.closest('.sn-hr-clear-live-filters');
      if(clear){ e.preventDefault(); var table=document.querySelector(clear.getAttribute('data-sn-table')); document.querySelectorAll('.sn-hr-live-search[data-sn-table], .sn-hr-live-select[data-sn-table]').forEach(function(i){ if(table && i.getAttribute('data-sn-table') && table.matches(i.getAttribute('data-sn-table'))){ i.value=''; } }); if(table){ table._snExcelFilterState={}; table.querySelectorAll('.sn-hr-col-filter').forEach(function(i){ i.value=''; }); applyTableFilters(table); } }
      var open=e.target.closest('.sn-hr-open-profile-modal');
      if(open){
        e.preventDefault();
        var form=open.closest('form'), row=open.closest('tr'), fields=row && row.querySelector('.sn-hr-row-edit-fields'), modal=form && form.querySelector('.sn-hr-profile-modal'), body=modal && modal.querySelector('.sn-hr-modal-body');
        if(fields && modal && body){
          modal._snReturnCell=fields.parentNode; modal._snFields=fields; modal._snRow=row;
          body.appendChild(fields); fields.hidden=false; modal.hidden=false; document.body.classList.add('sn-hr-modal-open');
          var title=modal.querySelector('.sn-hr-modal-title'); if(title) title.textContent=fields.getAttribute('data-sn-profile-title') || 'ویرایش نیرو';
          var cb=row.querySelector('.sn-hr-workforce-check'); if(cb) cb.checked=true;
        }
      }
      if(e.target.closest('[data-sn-hr-modal-close]')){
        e.preventDefault();
        var modal=e.target.closest('.sn-hr-profile-modal');
        if(modal && modal._snFields && modal._snReturnCell){ modal._snFields.hidden=true; modal._snReturnCell.appendChild(modal._snFields); modal.hidden=true; document.body.classList.remove('sn-hr-modal-open'); }
      }
      var tabLink=e.target.closest('[data-sn-open-tab], a[href="#hr-bulk"]');
      if(tabLink){
        var target=tabLink.getAttribute('data-sn-open-tab') || (tabLink.getAttribute('href')==='#hr-bulk' ? 'hr-bulk' : '');
        if(target==='hr-bulk') transferSelectedToBulk();
        var btn=document.querySelector('[data-sn-tab-target="'+target+'"]');
        if(btn){ setTimeout(function(){ btn.click(); if(target==='hr-bulk') transferSelectedToBulk(); }, 10); }
      }
      var directTab=e.target.closest('[data-sn-tab-target="hr-bulk"]');
      if(directTab){ setTimeout(function(){ applyBulkIds(getStoredBulkIds()); }, 80); }
    });
    document.addEventListener('keydown', function(e){ if(e.key==='Escape'){ document.querySelectorAll('.sn-hr-profile-modal:not([hidden]) [data-sn-hr-modal-close]').forEach(function(b){ b.click(); }); closeBulkPasswordModal(); } });
    document.addEventListener('submit', function(e){
      var form=e.target;
      if(!form.matches || !form.matches('.sn-hr-ajax-post-form')) return;
      var action=(form.querySelector('input[name="action"]')||{}).value || '';
      if(['sn_hr_profile_action','sn_hr_bulk_inline_profile_update','sn_hr_bulk_workforce_action','sn_hr_simple_add_person','sn_hr_bulk_set_password'].indexOf(action)===-1) return;
      e.preventDefault();
      if(form.classList.contains('sn-ajax-saving')) return;
      if(action==='sn_hr_bulk_set_password'){
        form.querySelectorAll('input[name="user_ids[]"][data-sn-generated="1"]').forEach(function(i){ i.remove(); });
        var ids=Array.prototype.slice.call(document.querySelectorAll('.sn-hr-workforce-check:checked')).map(function(i){ return String(i.value); });
        if(!ids.length){ notifyNear(form, 'هیچ نیرویی انتخاب نشده است.', false); return; }
        ids.forEach(function(id){ var h=document.createElement('input'); h.type='hidden'; h.name='user_ids[]'; h.value=id; h.setAttribute('data-sn-generated','1'); form.appendChild(h); });
      }
      var submitter=e.submitter || document.activeElement;
      var openModal=form.querySelector('.sn-hr-profile-modal:not([hidden])');
      if(action==='sn_hr_bulk_inline_profile_update' && openModal && openModal._snRow && submitter && submitter.closest && submitter.closest('.sn-hr-profile-modal')){
        saveSingleWorkforceRow(form, openModal._snRow, function(ok, msg){ notifyNear(form, msg, ok); if(ok){ var closeBtn=openModal.querySelector('[data-sn-hr-modal-close]'); if(closeBtn) closeBtn.click(); } }, openModal._snFields || openModal.querySelector('.sn-hr-row-edit-fields'));
        return;
      }
	  var fd=new FormData(form); fd.set('action', action);
	  if(action==='sn_hr_bulk_inline_profile_update'){
        if(form.querySelector('.sn-row-saving')){notifyNear(form,'ابتدا منتظر پایان ذخیره ردیف‌ها بمانید.',false);return;}
		form.querySelectorAll('.sn-hr-workforce-table tbody tr[data-sn-user-id]').forEach(function(row){ appendWorkforceAccessFields(fd, row); });
		fd.delete('visible_user_ids[]');
		var workforceTable=form.querySelector('.sn-hr-workforce-table');
		if(workforceTable){ visibleRows(workforceTable).forEach(function(row){ var uid=row.getAttribute('data-sn-user-id'); if(uid) fd.append('visible_user_ids[]',uid); }); }
        var ids=(fd.get('save_scope')==='visible' ? fd.getAll('visible_user_ids[]') : fd.getAll('user_ids[]')).map(String);
        if(!ids.length){notifyNear(form,'هیچ نیرویی انتخاب نشده است.',false);return;}
        var remove=[]; fd.forEach(function(value,key){var m=/^rows\[(\d+)\]/.exec(key);if(m && ids.indexOf(m[1])===-1) remove.push(key);});
        remove.forEach(function(key){fd.delete(key);}); packWorkforceRows(fd);
	  }
      if(submitter && submitter.name) fd.set(submitter.name, submitter.value || '');
      var buttons=form.querySelectorAll('button,input[type="submit"],.sn-hr-inline-autosave'); buttons.forEach(function(b){ b.disabled=true; }); form.classList.add('sn-ajax-saving');
      if(action==='sn_hr_simple_add_person'){ notifySaving(form, 'در حال ثبت نیروی جدید...'); }
      postHrRequest(fd).then(function(res){
        var ok=!!(res && res.success); var msg=(res && (res.message || (res.data && res.data.message))) || form.getAttribute('data-sn-ajax-message') || (ok?'انجام شد.':'خطا در ذخیره‌سازی');
        notifyNear(form, msg, ok);
        if(ok){
          document.querySelectorAll('.sn-hr-profile-modal:not([hidden])').forEach(function(m){ if(m._snRow) syncRowFromFields(m._snRow); });
          document.querySelectorAll('.sn-hr-profile-modal:not([hidden]) [data-sn-hr-modal-close]').forEach(function(b){ b.click(); });
          if(form.getAttribute('data-sn-ajax-reload')==='1'){
            var targetUrl=(res && res.redirect) || window.location.href;
            var hash=form.getAttribute('data-sn-ajax-reload-hash') || '';
            var nextUrl = hash && targetUrl.indexOf('#')===-1 ? targetUrl + '#' + hash : targetUrl;
            setTimeout(function(){ if(nextUrl===window.location.href){ window.location.reload(); } else { window.location.href = nextUrl; } }, 450);
          }
        }
      }).catch(function(err){ notifyNear(form, (err && err.message) ? err.message : 'خطا در ارتباط با سرور.', false); }).finally(function(){ buttons.forEach(function(b){ b.disabled=false; }); form.classList.remove('sn-ajax-saving'); });
    }, true);
  });
})();


/* SN finance wallet recalculation: retroactive commission posting */
(function($){
  'use strict';
  $(document).off('click.snWalletRecalcPublic').on('click.snWalletRecalcPublic', '#sn-wallet-recalculate-public', function(e){
    e.preventDefault();
    var cfg = window.snAjax || window.snData || {};
    var ajaxUrl = cfg.ajaxurl || window.ajaxurl || '';
    var nonce = cfg.nonce || cfg.admin_nonce || '';
    var $btn = $(this), $box = $('#sn-wallet-recalculate-public-result');
    function esc(v){ return String(v==null?'':v).replace(/[&<>"']/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#039;'}[c]||c;}); }
    if(!ajaxUrl){ $box.html('<div class="sn-notice sn-error">آدرس AJAX پیدا نشد. صفحه را رفرش کنید.</div>'); return; }
    if(!confirm('پورسانت پرداخت‌های قبلی بر اساس قوانین فعلی محاسبه شود؟ فقط موارد بدون تراکنش قبلی ساخته می‌شوند.')) return;
    $btn.prop('disabled', true).text('در حال محاسبه...');
    $box.html('<div class="sn-loading">در حال بررسی فاکتورهای پرداخت‌شده...</div>');
    $.ajax({url:ajaxUrl,type:'POST',timeout:60000,data:{action:'sn_wallet_recalculate',nonce:nonce}})
      .done(function(res){
        if(res && res.success){ $box.html('<div class="sn-notice sn-success">'+esc(res.message||'محاسبه مجدد انجام شد.')+'</div>'); }
        else { $box.html('<div class="sn-notice sn-error">'+esc((res&&res.message)||'محاسبه مجدد انجام نشد.')+'</div>'); }
      })
      .fail(function(xhr){
        var msg='خطای سرور: '+(xhr&&xhr.status?xhr.status:'نامشخص');
        if(xhr&&xhr.responseJSON&&xhr.responseJSON.message){ msg=xhr.responseJSON.message; }
        $box.html('<div class="sn-notice sn-error">'+esc(msg)+'</div>');
      })
      .always(function(){ $btn.prop('disabled', false).text('محاسبه مجدد پورسانت‌های قبلی'); });
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

/* Payment queue actions use JSON and update in place, including lazy-loaded forms. */
(function(){
  if(window.snPaymentQueueActionsReady)return;window.snPaymentQueueActionsReady=true;
  document.addEventListener('submit',function(e){
    var form=e.target,box=form.closest&&form.closest('.sn-paid-referral-actions');
    if(!box)return;
    var field=form.querySelector('[name="action"]'),action=field&&field.value;
    if(['sn_paid_referral_route','sn_paid_referral_create_follow_invoice'].indexOf(action)<0)return;
    e.preventDefault();e.stopImmediatePropagation();
    if(form.dataset.saving==='1'||!form.reportValidity())return;
    form.dataset.saving='1';
    var buttons=Array.from(form.querySelectorAll('button[type="submit"]')),labels=buttons.map(function(b){return b.textContent;});
    buttons.forEach(function(b){b.disabled=true;b.setAttribute('aria-busy','true');b.textContent='در حال ثبت…';});
    var note=box.querySelector('.sn-queue-feedback');
    if(!note){note=document.createElement('div');note.className='sn-notice sn-queue-feedback';note.setAttribute('role','status');box.prepend(note);}
    note.textContent='در حال ثبت درخواست…';
    var caseRow=form.closest('[data-referral-item-id]'),caseId=caseRow&&caseRow.getAttribute('data-referral-item-id');
    var cfg=window.snAjax||window.snData||{};
    fetch(cfg.ajaxurl||window.ajaxurl||'/wp-admin/admin-ajax.php',{method:'POST',credentials:'same-origin',body:new FormData(form)})
      .then(function(r){if(!r.ok)throw Error('پاسخ سرور دریافت نشد ('+r.status+'). پیش از ارسال مجدد، وضعیت پرونده را تازه کنید.');return r.json();})
      .then(function(r){
        if(!r||!r.success)throw Error((r&&(r.message||(r.data&&r.data.message)))||'عملیات انجام نشد.');
        if(r.html){var holder=document.createElement('div');holder.innerHTML=r.html;var fresh=holder.querySelector('.sn-paid-referral-actions');if(fresh){box.replaceWith(fresh);box=fresh;box.prepend(note);if(caseId){var row=Array.from(box.querySelectorAll('[data-referral-item-id]')).find(function(r){return r.getAttribute('data-referral-item-id')===caseId;});var details=row&&row.querySelector('.sn-queue-case');if(details)details.open=true;}}}
        note.className='sn-notice sn-success sn-queue-feedback';note.textContent=r.message||'انجام شد.';
      }).catch(function(err){note.className='sn-notice sn-error sn-queue-feedback';note.textContent=err.message||'خطا در ارتباط؛ وضعیت پرونده را بررسی کنید.';})
      .finally(function(){delete form.dataset.saving;buttons.forEach(function(b,i){b.disabled=false;b.removeAttribute('aria-busy');b.textContent=labels[i];});});
  },true);
})();

(function(){
 if(window.snQueueAccordionReady)return;window.snQueueAccordionReady=true;
 document.addEventListener('toggle',function(e){
  var item=e.target;if(!item.matches||!item.matches('.sn-queue-case')||!item.open)return;
  var box=item.closest('.sn-paid-referral-actions');if(!box)return;
  box.querySelectorAll('.sn-queue-case[open]').forEach(function(other){if(other!==item)other.open=false;});
 },true);
})();
