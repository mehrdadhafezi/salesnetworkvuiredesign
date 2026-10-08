(function () {
  'use strict';

  var activeCardModal = null;
  var cardModalOpener = null;

  document.querySelectorAll('[data-sn-customer-card-modal]').forEach(function (modal) {
    document.body.appendChild(modal);
  });

  var profileAccordions = Array.prototype.slice.call(document.querySelectorAll('.sn-customer-accordion'));
  function syncAccordionState(accordion) {
    var summary = accordion.querySelector(':scope > summary');
    if (summary) summary.setAttribute('aria-expanded', accordion.open ? 'true' : 'false');
  }
  profileAccordions.forEach(function (accordion) {
    syncAccordionState(accordion);
    accordion.addEventListener('toggle', function () {
      if (accordion.open) {
        profileAccordions.forEach(function (other) {
          if (other !== accordion && other.open) other.open = false;
        });
      }
      syncAccordionState(accordion);
    });
  });
  document.addEventListener('click', function (event) {
    var target = event.target instanceof Element ? event.target.closest('.sn-customer-portal a[href^="#sn-customer-"]') : null;
    if (!target) return;
    var selector = target.getAttribute('href');
    var accordion = selector ? document.querySelector(selector) : null;
    if (!accordion || !accordion.classList.contains('sn-customer-accordion')) return;
    event.preventDefault();
    accordion.open = true;
    window.setTimeout(function () { accordion.scrollIntoView({ behavior: 'smooth', block: 'start' }); }, 0);
  });
  if (window.location.hash) {
    var initialAccordion = document.querySelector(window.location.hash);
    if (initialAccordion && initialAccordion.classList.contains('sn-customer-accordion')) initialAccordion.open = true;
  }

  function closeCardModal() {
    if (!activeCardModal) return;
    activeCardModal.hidden = true;
    activeCardModal.setAttribute('aria-hidden', 'true');
    document.body.classList.remove('sn-customer-modal-open');
    document.documentElement.classList.remove('sn-customer-modal-open');
    var opener = cardModalOpener;
    if (opener) opener.setAttribute('aria-expanded', 'false');
    activeCardModal = null;
    cardModalOpener = null;
    if (opener && typeof opener.focus === 'function') opener.focus();
  }

  function openCardModal(opener) {
    var modalId = opener.getAttribute('data-sn-customer-card-open');
    var modal = modalId ? document.getElementById(modalId) : null;
    if (!modal) return;
    if (activeCardModal) closeCardModal();
    cardModalOpener = opener;
    activeCardModal = modal;
    /* Move the active dialog to body so theme/Elementor transforms cannot shift fixed positioning. */
    if (modal.parentNode !== document.body) { document.body.appendChild(modal); }
    opener.setAttribute('aria-expanded', 'true');
    modal.hidden = false;
    modal.setAttribute('aria-hidden', 'false');
    document.body.classList.add('sn-customer-modal-open');
    document.documentElement.classList.add('sn-customer-modal-open');
    var dialog = modal.querySelector('[role="dialog"]');
    if (dialog) window.setTimeout(function () { dialog.focus(); }, 0);
  }

  document.addEventListener('click', function (event) {
    var target = event.target instanceof Element ? event.target : null;
    if (!target) return;
    var opener = target.closest('[data-sn-customer-card-open]');
    if (opener) { event.preventDefault(); openCardModal(opener); return; }
    if (activeCardModal && target.closest('[data-sn-customer-card-close]')) {
      event.preventDefault();
      closeCardModal();
    }
  });

  document.addEventListener('keydown', function (event) {
    if (event.key === 'Escape' && activeCardModal) {
      event.preventDefault();
      closeCardModal();
      return;
    }
    if (event.key === 'Tab' && activeCardModal) {
      var focusable = Array.prototype.slice.call(activeCardModal.querySelectorAll('button:not([disabled]),a[href],input:not([disabled]),select:not([disabled]),textarea:not([disabled]),[tabindex]:not([tabindex="-1"])')).filter(function (node) {
        return !node.hidden && node.getClientRects().length > 0;
      });
      if (!focusable.length) return;
      var first = focusable[0];
      var last = focusable[focusable.length - 1];
      var dialog = activeCardModal.querySelector('[role="dialog"]');
      if (event.shiftKey && (document.activeElement === first || document.activeElement === dialog)) { event.preventDefault(); last.focus(); }
      else if (!event.shiftKey && document.activeElement === dialog) { event.preventDefault(); first.focus(); }
      else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
    }
  });

  // POST responses carry the card ID so the customer immediately sees its new state.
  var requestedCard = new URLSearchParams(window.location.search).get('card');
  if (requestedCard && /^\d+$/.test(requestedCard)) {
    var returnedOpener = Array.prototype.find.call(document.querySelectorAll('[data-sn-customer-card-open]'), function (node) {
      return node.getAttribute('data-sn-customer-card-open') === 'sn-customer-card-modal-' + requestedCard;
    });
    if (returnedOpener) {
      var cardsAccordion = returnedOpener.closest('.sn-customer-accordion');
      if (cardsAccordion) cardsAccordion.open = true;
      openCardModal(returnedOpener);
      var outcomeNotice = document.querySelector('.sn-customer-portal .sn-customer-global-notice');
      var resultBody = activeCardModal && activeCardModal.querySelector('.sn-customer-card-dialog-body');
      if (outcomeNotice && resultBody) resultBody.insertBefore(outcomeNotice.cloneNode(true), resultBody.firstChild);
      if (window.history && window.history.replaceState) {
        var cleanUrl = new URL(window.location.href);
        cleanUrl.searchParams.delete('card');
        window.history.replaceState(window.history.state, '', cleanUrl.toString());
      }
    }
  }

  var customerNotices = Array.prototype.slice.call(document.querySelectorAll('.sn-customer-global-notice'));
  if (customerNotices.length) {
    function dismissCustomerNotices() {
      customerNotices.forEach(function (notice) { if (notice.isConnected) notice.remove(); });
    }
    customerNotices.forEach(function (notice) {
      var close = notice.querySelector('[data-sn-customer-notice-close]');
      if (close) close.addEventListener('click', dismissCustomerNotices);
    });
    window.setTimeout(dismissCustomerNotices, 5000);
    if (window.history && window.history.replaceState) {
      var noticeUrl = new URL(window.location.href);
      noticeUrl.searchParams.delete('sn_ops_notice');
      noticeUrl.searchParams.delete('sn_ops_kind');
      window.history.replaceState(window.history.state, '', noticeUrl.toString());
    }
  }

  var root = document.querySelector('[data-sn-customer-login]');
  if (!root || typeof window.snCustomerPortal !== 'object') return;

  var phoneForm = root.querySelector('[data-sn-customer-phone-form]');
  var otpForm = root.querySelector('[data-sn-customer-otp-form]');
  var phoneInput = phoneForm && phoneForm.querySelector('[name="phone"]');
  var codeInput = otpForm && otpForm.querySelector('[name="code"]');
  var message = root.querySelector('[data-sn-customer-message]');
  var masked = root.querySelector('[data-sn-customer-masked]');
  var back = root.querySelector('[data-sn-customer-back]');
  var resend = root.querySelector('[data-sn-customer-resend]');
  var state = { phone: '', challenge: '', timer: 0, requestingCode: false };

  function englishDigits(value) {
    return String(value || '')
      .replace(/[۰-۹]/g, function (digit) { return String('۰۱۲۳۴۵۶۷۸۹'.indexOf(digit)); })
      .replace(/[٠-٩]/g, function (digit) { return String('٠١٢٣٤٥٦٧٨٩'.indexOf(digit)); });
  }

  function digits(value) { return englishDigits(value).replace(/\D+/g, ''); }

  function showMessage(text, kind) {
    if (!message) return;
	var compact = String(text || '');
	if (kind === 'success') {
		compact = 'موفق — ' + (compact.indexOf('ورود') !== -1 ? 'ورود انجام شد.' : 'کد ارسال شد.');
	} else if (kind === 'error') {
		if (compact.indexOf('شماره') !== -1) compact = 'شماره موبایل معتبر نیست.';
		else if (compact.indexOf('منقضی') !== -1) compact = 'کد منقضی شده است.';
		else if (compact.indexOf('کد') !== -1 && compact.indexOf('ارسال') === -1) compact = 'کد ورود معتبر نیست.';
		else if (compact.indexOf('ارسال') !== -1 || compact.indexOf('پترن') !== -1) compact = 'ارسال کد ناموفق بود.';
		else compact = 'عملیات ناموفق بود.';
		compact = 'ناموفق — ' + compact;
	}
	message.textContent = compact;
    message.className = 'sn-customer-login-message' + (kind ? ' is-' + kind : '');
  }

  function setBusy(form, busy) {
    if (!form) return;
    var button = form.querySelector('button[type="submit"]');
    if (button) {
      button.disabled = Boolean(busy);
      button.classList.toggle('is-loading', Boolean(busy));
    }
    form.setAttribute('aria-busy', busy ? 'true' : 'false');
  }

  function post(action, payload) {
    var body = new URLSearchParams();
    body.set('action', action);
    body.set('nonce', window.snCustomerPortal.nonce || '');
    Object.keys(payload || {}).forEach(function (key) { body.set(key, payload[key]); });
    return fetch(window.snCustomerPortal.ajaxUrl, {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded; charset=UTF-8' },
      body: body.toString()
    }).then(function (response) {
      return response.text().then(function (raw) {
        var data = {};
        try { data = raw ? JSON.parse(raw) : {}; } catch (parseError) { data = {}; }
        // Accept both this plugin's flat response and the standard WP
        // wp_send_json_success/error envelope used by some host integrations.
        var nested = data && data.data && typeof data.data === 'object' ? data.data : {};
        var messageText = data.message || nested.message || '';
        if (!response.ok || !data.success) {
          throw new Error(messageText || ('ارتباط با سرور برقرار نشد' + (response.status ? ' (کد ' + response.status + ')' : '') + '.'));
        }
        if (!data.challenge && nested.challenge) data.challenge = nested.challenge;
        if (!data.masked_phone && nested.masked_phone) data.masked_phone = nested.masked_phone;
        if (!data.redirect_url && nested.redirect_url) data.redirect_url = nested.redirect_url;
        return data;
      });
    });
  }

  function countdown(seconds) {
    window.clearInterval(state.timer);
    var left = Number(seconds || 60);
    if (!resend) return;
    resend.disabled = true;
    resend.textContent = 'ارسال مجدد تا ' + left + ' ثانیه';
    state.timer = window.setInterval(function () {
      left -= 1;
      if (left <= 0) {
        window.clearInterval(state.timer);
        resend.disabled = false;
        resend.textContent = 'ارسال مجدد کد';
        return;
      }
      resend.textContent = 'ارسال مجدد تا ' + left + ' ثانیه';
    }, 1000);
  }

  function requestCode() {
    if (state.requestingCode) return Promise.resolve();
    var phone = digits(phoneInput && phoneInput.value);
    if (!/^09\d{9}$/.test(phone)) {
      showMessage('شماره موبایل را به صورت ۰۹xxxxxxxxx وارد کنید.', 'error');
      if (phoneInput) phoneInput.focus();
      return Promise.resolve();
    }
    state.requestingCode = true;
    state.phone = phone;
    setBusy(phoneForm, true);
    setBusy(otpForm, true);
    showMessage('در حال ارسال کد ورود…', 'info');
    return post('sn_customer_portal_send_otp', { phone: phone }).then(function (data) {
      state.challenge = data.challenge || '';
      if (masked) masked.textContent = data.masked_phone || phone;
      if (phoneForm) phoneForm.hidden = true;
      if (otpForm) otpForm.hidden = false;
      showMessage(data.message || 'کد ورود ارسال شد.', 'success');
      countdown(Math.max(1, Math.min(60, Number(data.expires_in || 60))));
      if (codeInput) { codeInput.value = ''; codeInput.focus(); }
    }).catch(function (error) {
      showMessage(error.message, 'error');
    }).finally(function () {
      state.requestingCode = false;
      setBusy(phoneForm, false);
      setBusy(otpForm, false);
    });
  }

  if (phoneInput) {
    phoneInput.addEventListener('input', function () { phoneInput.value = digits(phoneInput.value).slice(0, 11); });
  }
  if (codeInput) {
    codeInput.addEventListener('input', function () { codeInput.value = digits(codeInput.value).slice(0, 6); });
  }
  if (phoneForm) {
    phoneForm.addEventListener('submit', function (event) { event.preventDefault(); requestCode(); });
  }
  if (otpForm) {
    otpForm.addEventListener('submit', function (event) {
      event.preventDefault();
      var code = digits(codeInput && codeInput.value);
      if (!/^\d{6}$/.test(code) || !state.challenge) {
        showMessage('کد ۶ رقمی پیامک‌شده را کامل وارد کنید.', 'error');
        if (codeInput) codeInput.focus();
        return;
      }
      setBusy(otpForm, true);
      showMessage('در حال بررسی کد…', 'info');
      post('sn_customer_portal_verify_otp', { phone: state.phone, challenge: state.challenge, code: code }).then(function (data) {
        showMessage(data.message || 'ورود موفق بود.', 'success');
        window.location.assign(data.redirect_url || window.snCustomerPortal.portalUrl || window.location.href);
      }).catch(function (error) {
        showMessage(error.message, 'error');
        if (codeInput) { codeInput.select(); codeInput.focus(); }
      }).finally(function () { setBusy(otpForm, false); });
    });
  }
  if (back) {
    back.addEventListener('click', function () {
      window.clearInterval(state.timer);
      state.challenge = '';
      if (otpForm) otpForm.hidden = true;
      if (phoneForm) phoneForm.hidden = false;
      showMessage('', '');
      if (phoneInput) phoneInput.focus();
    });
  }
  if (resend) {
    resend.addEventListener('click', function () {
      if (!resend.disabled && phoneInput) { phoneInput.value = state.phone; requestCode(); }
    });
  }
})();
