(function ($) {
  'use strict';

  var cfg = window.snDot || window.snAjax || window.snData || {};
  var ajaxUrl = cfg.ajaxurl || window.ajaxurl || '';
  var lastModalOpener = null;

  function responseMessage(response, fallback) {
    if (response && response.data && response.data.message) {
      return String(response.data.message);
    }
    if (response && response.message) {
      return String(response.message);
    }
    return fallback;
  }

  function showNotice($scope, message, success) {
    var $root = $scope.closest('.sn-dot-supervisor, .sn-dot-case-card, .sn-dot-converter, .sn-dot-wrap');
    if (!$root.length) {
      $root = $scope;
    }
    $root.find('> .sn-dot-live-notice').remove();
    $('<div class="sn-notice sn-dot-live-notice"></div>')
      .addClass(success ? 'sn-success' : 'sn-error')
      .text(message)
      .prependTo($root);
  }

  function reloadSupervisorTab() {
    var $host = $('#sn-tab-needs-action.sn-lazy-tab');
    var $tab = $('#sn-supervisor-panel .sn-tab[data-tab="needs-action"]');
    if (!$host.length || !$tab.length) {
      window.location.reload();
      return;
    }
    $host.removeData('loaded').removeData('loading').empty();
    $tab.trigger('click');
  }

  function refreshConverterPanel(caseId, message) {
    var $current = $('#sn-dot-converter-panel');
    if (!$current.length) {
      return null;
    }
    return $.ajax({
      url: ajaxUrl,
      method: 'POST',
      dataType: 'json',
      timeout: 30000,
      data: {
        action: 'sn_dot_refresh_converter_panel',
        nonce: cfg.nonce || '',
        context: String($current.attr('data-sn-context') || '')
      }
    }).done(function (response) {
      var html = response && response.success && response.data ? response.data.html : '';
      if (!html) {
        return;
      }
      var $container = $('<div></div>').html(html);
      var $fresh = $container.find('#sn-dot-converter-panel').first();
      if (!$fresh.length) {
        return;
      }
      $current.replaceWith($fresh);
      applyConverterFilters($fresh);
      syncConverterStatusFields($fresh);
      if (caseId) {
        var toggle = $fresh.find('#dot-case-' + caseId + ' .sn-dot-case-toggle').get(0);
        if (toggle) {
          setCaseExpanded(toggle, true);
          try { toggle.scrollIntoView({ behavior: 'smooth', block: 'center' }); } catch (error) {}
        }
      }
      showNotice($fresh, message, true);
    });
  }

  function toEnglishDigits(value) {
    return String(value || '')
      .replace(/[۰-۹]/g, function (digit) { return String('۰۱۲۳۴۵۶۷۸۹'.indexOf(digit)); })
      .replace(/[٠-٩]/g, function (digit) { return String('٠١٢٣٤٥٦٧٨٩'.indexOf(digit)); });
  }

  function formatMoneyInput(input) {
    var number = toEnglishDigits(input.value).replace(/[^0-9]/g, '');
    input.value = number ? number.replace(/\B(?=(\d{3})+(?!\d))/g, ',') : '';
  }

  function normalizeFilterValue(value) {
    return toEnglishDigits(value)
      .toLowerCase()
      .replace(/[يى]/g, 'ی')
      .replace(/ك/g, 'ک')
      .replace(/\s+/g, ' ')
      .trim();
  }

  function formatPersianCount(value) {
    return String(value).replace(/[0-9]/g, function (digit) {
      return '۰۱۲۳۴۵۶۷۸۹'.charAt(Number(digit));
    });
  }

  function setCaseExpanded(toggle, expanded) {
    if (!toggle) {
      return;
    }
    var contentId = String(toggle.getAttribute('aria-controls') || '');
    var content = contentId ? document.getElementById(contentId) : null;
    toggle.setAttribute('aria-expanded', expanded ? 'true' : 'false');
    if (content) {
      if (expanded) syncConverterStatusFields($(content));
      content.hidden = !expanded;
    }
    var article = toggle.closest('.sn-dot-converter-case');
    if (article) {
      article.classList.toggle('is-expanded', expanded);
    }
  }

  function applyConverterFilters($panel) {
    if (!$panel || !$panel.length) {
      return;
    }
    var term = normalizeFilterValue($panel.find('[data-sn-dot-filter="search"]').val());
    var status = String($panel.find('[data-sn-dot-filter="status"]').val() || '');
    var contact = String($panel.find('[data-sn-dot-filter="contact"]').val() || '');
    var sort = String($panel.find('[data-sn-dot-filter="sort"]').val() || 'newest');
    var visible = 0;
    $panel.find('[data-sn-dot-case]').each(function () {
      var $case = $(this);
      var matchesSearch = !term || normalizeFilterValue($case.attr('data-search')).indexOf(term) !== -1;
      var matchesStatus = !status || String($case.attr('data-status') || '') === status;
      var matchesContact = !contact || String($case.attr('data-contact') || 'new') === contact;
      var matches = matchesSearch && matchesStatus && matchesContact;
      $case.prop('hidden', !matches);
      if (matches) {
        visible += 1;
      }
    });
    var $list = $panel.find('.sn-dot-converter-case-list').first();
    var cases = $list.find('[data-sn-dot-case]').get();
    cases.sort(function (a, b) {
      var aUpdated = Number(a.getAttribute('data-updated') || 0);
      var bUpdated = Number(b.getAttribute('data-updated') || 0);
      var aProgress = Number(a.getAttribute('data-progress') || 0);
      var bProgress = Number(b.getAttribute('data-progress') || 0);
      if (sort === 'oldest') return aUpdated - bUpdated;
      if (sort === 'progress_desc') return (bProgress - aProgress) || (bUpdated - aUpdated);
      if (sort === 'progress_asc') return (aProgress - bProgress) || (bUpdated - aUpdated);
      return bUpdated - aUpdated;
    });
    cases.forEach(function (item) {
      var emptyState = $list.find('.sn-dot-filter-empty').get(0);
      if (emptyState) $list.get(0).insertBefore(item, emptyState);
      else $list.append(item);
    });
    $panel.find('.sn-dot-filter-result-count').text(formatPersianCount(visible));
    $panel.find('.sn-dot-filter-empty').prop('hidden', visible !== 0);
  }

  function syncSelectedOption() {
    $('.sn-dot-option').each(function () {
      var $option = $(this);
      var selected = Boolean($option.find('input[type="radio"]').prop('checked'));
      $option.toggleClass('is-selected', selected);
      $option.find('.sn-dot-option-state').text(selected ? 'انتخاب شده' : 'انتخاب نشده');
      $option.find('.sn-dot-option-open').attr('aria-pressed', selected ? 'true' : 'false');
    });
  }

  function syncConverterStatusFields($scope) {
    var $root = $scope && $scope.length ? $scope : $(document);
    $root.find('.sn-dot-converter-update').add($root.filter('.sn-dot-converter-update')).each(function () {
      var $form = $(this);
      var status = String($form.find('select[name="contact_status"]').val() || 'new');
      var $followup = $form.find('[data-sn-dot-status-field="follow_up"]');
      var $decline = $form.find('[data-sn-dot-status-field="customer_declined"]');
      $followup.prop('hidden', status !== 'follow_up').toggle(status === 'follow_up');
      var $followupDate = $followup.find('input[name="followup_date"]');
      var $followupTime = $followup.find('input[name="followup_time"]');
      var dateValue = String($followupDate.val() || '').trim();
      var timeValue = String($followupTime.val() || '').trim();
      var followupComplete = dateValue !== '' && timeValue !== '';
      $followupDate.add($followupTime).prop('required', status === 'follow_up');
      $followup.find('input[name="followup_at"]').val(followupComplete ? dateValue + 'T' + timeValue : '');
      $decline.prop('hidden', status !== 'customer_declined').toggle(status === 'customer_declined');
      $decline.find('textarea[name="decline_reason"]').prop('required', status === 'customer_declined');
	  var $paymentStep = $form.closest('[data-sn-dot-case]').find('[data-sn-dot-payment-step]').first();
	  var showPayment = status === 'payment_link' || String($paymentStep.attr('data-has-payment') || '0') === '1';
	  $paymentStep.prop('hidden', !showPayment).toggle(showPayment);
	  var canSaveStatus = status === 'no_answer' || status === 'follow_up' || status === 'customer_declined' || status === 'payment_link';
	  $form.find('.sn-converter-simple-save')
		.toggle(true)
		.text(status === 'payment_link' ? 'ثبت محصول و ادامه به ارسال لینک' : 'ذخیره نتیجه تماس')
		// Keep the action reachable. Required-field and workflow validation below
		// produces an explicit message; a permanently disabled button gave the user
		// no feedback when a cached/third-party script missed a change event.
		.prop('disabled', false)
		.attr('aria-disabled', 'false');
	  $form.toggleClass('sn-dot-contact-needs-input', !canSaveStatus || (status === 'follow_up' && !followupComplete));
    });
  }

  function closeOptionModal(modal, restoreFocus) {
    if (!modal) {
      return;
    }
    modal.classList.remove('is-open');
    modal.hidden = true;
    if (!document.querySelector('.sn-dot-option-modal:not([hidden])')) {
      document.body.classList.remove('sn-dot-modal-open');
    }
    if (restoreFocus !== false && lastModalOpener && document.documentElement.contains(lastModalOpener)) {
      lastModalOpener.focus();
    }
  }

  function openModalById(modalId, opener) {
    var modal = modalId ? document.getElementById(modalId) : null;
    if (!modal) {
      return;
    }
    document.querySelectorAll('.sn-dot-option-modal:not([hidden])').forEach(function (openModal) {
      closeOptionModal(openModal, false);
    });
    lastModalOpener = opener || lastModalOpener;
    if (modal.parentNode !== document.body) {
      document.body.appendChild(modal);
    }
    modal.hidden = false;
    document.body.classList.add('sn-dot-modal-open');
    window.requestAnimationFrame(function () {
      modal.classList.add('is-open');
      var focusTarget = modal.querySelector('.sn-dot-option-modal-close, .sn-dot-option-modal-panel');
      if (focusTarget) {
        focusTarget.focus();
      }
    });
  }

  function openOptionModal(trigger) {
    openModalById(String(trigger.getAttribute('aria-controls') || ''), trigger);
  }

  // Option HTML can contain the standard `.feature` accordion markup.
  // Handle it here instead of executing arbitrary inline <script> blocks from
  // administrator-authored HTML. This also repairs older content whose script
  // wrapper was stripped by WordPress and previously appeared as raw text.
  $(document).on('click.snDotEmbeddedFeature', '.sn-dot-option-fullscreen-content .feature-head', function (event) {
    event.preventDefault();
    var button = this;
    var item = button.closest('.feature');
    if (!item) {
      return;
    }
    var scope = item.closest('.sn-dot-option-fullscreen-content') || document;
    var wasOpen = item.classList.contains('open');
    scope.querySelectorAll('.feature.open').forEach(function (openItem) {
      if (openItem === item) {
        return;
      }
      openItem.classList.remove('open');
      var openHead = openItem.querySelector('.feature-head');
      if (openHead) {
        openHead.setAttribute('aria-expanded', 'false');
      }
    });
    item.classList.toggle('open', !wasOpen);
    button.setAttribute('aria-expanded', String(!wasOpen));
  });

  $(document).on('change.snDotOption', '.sn-dot-option input[type="radio"]', syncSelectedOption);
  $(document).on('click.snDotOptionOpen', '.sn-dot-option-open', function () {
    openOptionModal(this);
  });
  $(document).on('click.snDotOptionClose', '[data-sn-dot-modal-close]', function () {
    closeOptionModal(this.closest('.sn-dot-option-modal'));
  });
  $(document).on('click.snDotOptionSelect', '.sn-dot-modal-select', function () {
    var key = String(this.getAttribute('data-option-key') || '');
    var title = String(this.getAttribute('data-option-title') || '');
    var $form = $('.sn-dot-options-form').first();
    var $radio = $form.find('.sn-dot-option-radio').filter(function () {
      return String(this.value) === key;
    }).first();
    if (!$form.length || !$radio.length) {
      return;
    }
    $form.data('snDotPendingOption', key);
    var confirmModal = document.querySelector('.sn-dot-confirm-modal');
    if (!confirmModal) {
      return;
    }
    var titleNode = confirmModal.querySelector('[data-sn-dot-confirm-option]');
    if (titleNode) {
      titleNode.textContent = title;
    }
    var cardOpener = $radio.closest('.sn-dot-option').find('.sn-dot-option-open').get(0) || this;
    openModalById(confirmModal.id, cardOpener);
  });

  $(document).on('click.snDotOptionConfirm', '.sn-dot-confirm-yes', function () {
    var $button = $(this);
    var $form = $('.sn-dot-options-form').first();
    var key = String($form.data('snDotPendingOption') || '');
    var $radio = $form.find('.sn-dot-option-radio').filter(function () {
      return String(this.value) === key;
    }).first();
    if (!$form.length || !$radio.length || !key) {
      closeOptionModal(this.closest('.sn-dot-option-modal'));
      showNotice($form.length ? $form : $button, 'گزینه انتخاب‌شده معتبر نیست. دوباره تلاش کنید.', false);
      return;
    }
    $button.prop('disabled', true).attr('aria-busy', 'true').text('در حال ثبت…');
    $radio.prop('checked', true).trigger('change');
    closeOptionModal(this.closest('.sn-dot-option-modal'), false);
    $form.trigger('submit');
  });
  $(document).on('keydown.snDotOptionModal', function (event) {
    var modal = document.querySelector('.sn-dot-option-modal:not([hidden])');
    if (!modal) {
      return;
    }
    if (event.key === 'Escape') {
      event.preventDefault();
      closeOptionModal(modal);
      return;
    }
    if (event.key !== 'Tab') {
      return;
    }
    var focusable = Array.prototype.slice.call(modal.querySelectorAll('button:not([disabled]), a[href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'));
    if (!focusable.length) {
      event.preventDefault();
      return;
    }
    var first = focusable[0];
    var last = focusable[focusable.length - 1];
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  });
  $(document).on('input.snDotMoney', '.sn-dot-deposit-form input[name="deposit_amount"]', function () {
    formatMoneyInput(this);
  });

  $(document).on('change.snDotContactStatus', '.sn-dot-converter-update select[name="contact_status"]', function () {
    syncConverterStatusFields($(this).closest('.sn-dot-converter-update'));
  });

  $(document).on('input.snDotFollowup change.snDotFollowup', '.sn-dot-converter-update input[name="followup_date"], .sn-dot-converter-update input[name="followup_time"]', function () {
    syncConverterStatusFields($(this).closest('.sn-dot-converter-update'));
  });

  $(document).on('click.snDotCaseToggle', '.sn-dot-case-toggle', function () {
    setCaseExpanded(this, this.getAttribute('aria-expanded') !== 'true');
  });

  $(document).on('input.snDotConverterFilter change.snDotConverterFilter', '[data-sn-dot-filter]', function () {
    applyConverterFilters($(this).closest('.sn-dot-converter'));
  });

  $(document).on('click.snDotConverterFilterReset', '.sn-dot-filter-reset', function () {
    var $panel = $(this).closest('.sn-dot-converter');
    $panel.find('[data-sn-dot-filter="search"], [data-sn-dot-filter="status"], [data-sn-dot-filter="contact"]').val('');
    $panel.find('[data-sn-dot-filter="sort"]').val('newest');
    applyConverterFilters($panel);
    $panel.find('[data-sn-dot-filter="search"]').trigger('focus');
  });

  $(document).on('submit.snDotCustomer', '.sn-dot-options-form', function (event) {
    var $form = $(this);
    if (!$form.find('input[name="option_key"]:checked').length) {
      event.preventDefault();
      showNotice($form, 'لطفاً یکی از گزینه‌ها را انتخاب کنید.', false);
      return;
    }
    $form.find('button[type="submit"]').prop('disabled', true).text('در حال ثبت درخواست…');
    $form.attr('aria-busy', 'true');
  });

  $(document).on('submit.snDotAction', '.sn-dot-action-form', function (event) {
    if (!ajaxUrl) {
      return;
    }
    event.preventDefault();

    var $form = $(this);
    if ($form.data('snDotBusy')) {
      return;
    }

    var action = String($form.find('input[name="action"]').val() || '');
    if (action === 'sn_dot_converter_update') {
      var contactStatus = String($form.find('select[name="contact_status"]').val() || 'new');
	  syncConverterStatusFields($form);
	  if (['no_answer', 'follow_up', 'customer_declined', 'payment_link'].indexOf(contactStatus) === -1) {
		showNotice($form, 'یکی از نتیجه‌های تماس را انتخاب کنید.', false);
		return;
	  }
	  if (contactStatus === 'payment_link' && !$form.find('select[name="option_key"]').val() && !$form.find('input[name="option_key"]').val()) {
		showNotice($form, 'برای ارسال لینک پرداخت، ابتدا محصول نهایی را انتخاب کنید.', false);
		return;
	  }
	  var followupDate = String($form.find('input[name="followup_date"]').val() || '').trim();
	  var followupTime = String($form.find('input[name="followup_time"]').val() || '').trim();
      if (contactStatus === 'follow_up' && (!followupDate || !followupTime)) {
        showNotice($form, 'برای تماس مجدد، تاریخ و ساعت را انتخاب کنید.', false);
        return;
      }
      if (contactStatus === 'customer_declined' && !String($form.find('textarea[name="decline_reason"]').val() || '').trim()) {
        showNotice($form, 'برای ثبت انصراف، واردکردن دلیل اجباری است.', false);
        return;
      }
    }
    if (action === 'sn_dot_create_payment' && !window.confirm('لینک پرداخت ساخته و برای مشتری پیامک شود؟')) {
      return;
    }

    var $buttons = $form.find('button, input[type="submit"]');
    var keepBusyUntilRefresh = false;
    $form.data('snDotBusy', true);
    $buttons.prop('disabled', true).attr('aria-busy', 'true');

    $.ajax({
      url: ajaxUrl,
      method: 'POST',
      data: $form.serialize(),
      dataType: 'json',
      timeout: 30000
    }).done(function (response) {
      if (!response || !response.success) {
        showNotice($form, responseMessage(response, 'انجام عملیات ناموفق بود.'), false);
        return;
      }

      var message = responseMessage(response, 'عملیات با موفقیت انجام شد.');
      keepBusyUntilRefresh = true;
      showNotice($form, message, true);
      if ($form.closest('.sn-dot-supervisor').length) {
        window.setTimeout(reloadSupervisorTab, 650);
      } else {
        var caseId = Number($form.find('input[name="case_id"]').val() || (response.data && response.data.case_id) || 0);
        var refreshRequest = refreshConverterPanel(caseId, message);
        if (!refreshRequest) {
          keepBusyUntilRefresh = false;
          $form.data('snDotBusy', false);
          $buttons.prop('disabled', false).removeAttr('aria-busy');
        } else {
          refreshRequest.done(function () {
            if ($form.get(0) && document.documentElement.contains($form.get(0))) {
              keepBusyUntilRefresh = false;
              $form.data('snDotBusy', false);
              $buttons.prop('disabled', false).removeAttr('aria-busy');
              showNotice($form, message + ' برای نمایش وضعیت تازه، صفحه را یک‌بار تازه‌سازی کنید.', true);
            }
          });
          refreshRequest.fail(function (xhr) {
            keepBusyUntilRefresh = false;
            $form.data('snDotBusy', false);
            $buttons.prop('disabled', false).removeAttr('aria-busy');
            var refreshResponse = xhr && xhr.responseJSON ? xhr.responseJSON : null;
            showNotice($form, responseMessage(refreshResponse, message + ' برای نمایش وضعیت تازه، صفحه را یک‌بار تازه‌سازی کنید.'), true);
          });
        }
      }
    }).fail(function (xhr) {
      var response = xhr && xhr.responseJSON ? xhr.responseJSON : null;
      var fallback = xhr && xhr.status ? 'خطای ارتباط با سرور (' + xhr.status + ')' : 'ارتباط با سرور برقرار نشد.';
      showNotice($form, responseMessage(response, fallback), false);
    }).always(function () {
      if (keepBusyUntilRefresh) {
        return;
      }
      $form.data('snDotBusy', false);
      $buttons.prop('disabled', false).removeAttr('aria-busy');
    });
  });

  $(function () {
    syncSelectedOption();
    syncConverterStatusFields($(document));
    var postSelection = document.getElementById('sn-dot-after-selection');
    if (postSelection && String(postSelection.getAttribute('data-auto-scroll') || '') === '1') {
      window.setTimeout(function () {
        try { postSelection.scrollIntoView({ behavior: 'smooth', block: 'center' }); } catch (error) { postSelection.scrollIntoView(); }
      }, 120);
    }
    $('.sn-dot-converter').each(function () {
      applyConverterFilters($(this));
    });
    try {
      var params = new URLSearchParams(window.location.search);
      var requestedDotTab = params.get('sn_dot_tab');
      if (requestedDotTab === 'needs-action' || requestedDotTab === 'conversion-ready') {
        window.setTimeout(function () {
          $('#sn-supervisor-panel .sn-tab[data-tab="needs-action"]').trigger('click');
        }, 60);
      }
    } catch (error) {
      // Older browsers still retain the normal non-Ajax form fallback.
    }
  });
})(jQuery);
