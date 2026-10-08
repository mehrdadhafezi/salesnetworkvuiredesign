(function () {
  'use strict';

  window.snDotTestSmsAssetVersion = '1.0.160-centered-corporate-ui';

  var cfg = window.snDotTestSms || {};
  var ajaxUrl = String(cfg.ajaxUrl || '');
  var nonce = String(cfg.nonce || '');
  var expectedRunId = String(cfg.runId || '');
  var interval = Math.max(2500, Number(cfg.pollInterval || 4000));
  var pollTimer = null;
  var polling = false;
  var stopped = false;

  if (!ajaxUrl || !nonce || !expectedRunId || !window.fetch) {
    return;
  }

  function element(tag, className, text) {
    var node = document.createElement(tag);
    if (className) {
      node.className = className;
    }
    if (typeof text !== 'undefined') {
      node.textContent = String(text);
    }
    return node;
  }

  function host() {
    var current = document.getElementById('sn-dot-test-sms-host');
    if (current) {
      return current;
    }
    current = element('div', 'sn-dot-test-sms-host');
    current.id = 'sn-dot-test-sms-host';
    current.setAttribute('dir', 'rtl');
    current.setAttribute('aria-live', 'polite');
    current.setAttribute('aria-label', String(cfg.label || 'پیامک آزمایشی'));
    document.body.appendChild(current);
    return current;
  }

  function firstSafeUrl(message) {
    var match = String(message || '').match(/https?:\/\/[^\s<>"']+/i);
    if (!match) {
      return '';
    }
    var value = match[0].replace(/[،؛,.!?\])}]+$/g, '');
    try {
      var parsed = new URL(value, window.location.href);
      return parsed.protocol === 'https:' || parsed.protocol === 'http:' ? parsed.href : '';
    } catch (error) {
      return '';
    }
  }

  function copyText(value, button) {
    function done() {
      var original = button.textContent;
      button.textContent = 'کپی شد';
      window.setTimeout(function () { button.textContent = original; }, 1400);
    }
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(value).then(done).catch(function () {});
      return;
    }
    var textarea = document.createElement('textarea');
    textarea.value = value;
    textarea.setAttribute('readonly', 'readonly');
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    document.body.appendChild(textarea);
    textarea.select();
    try { document.execCommand('copy'); done(); } catch (error) {}
    textarea.remove();
  }

  function dismiss(card) {
    if (!card || card.classList.contains('is-leaving')) {
      return;
    }
    window.clearTimeout(Number(card.dataset.dismissTimer || 0));
    card.classList.add('is-leaving');
    window.setTimeout(function () { card.remove(); }, 220);
  }

  function armDismiss(card) {
    window.clearTimeout(Number(card.dataset.dismissTimer || 0));
    var timer = window.setTimeout(function () { dismiss(card); }, 26000);
    card.dataset.dismissTimer = String(timer);
  }

  function showMessage(item) {
	if (stopped) {
	  return;
	}
    var container = host();
    while (container.children.length >= 4) {
      container.removeChild(container.firstElementChild);
    }

    var card = element('section', 'sn-dot-test-sms');
    card.setAttribute('role', 'status');

    var header = element('header', 'sn-dot-test-sms-head');
    var identity = element('div', 'sn-dot-test-sms-identity');
    identity.appendChild(element('span', 'sn-dot-test-sms-icon', 'SMS'));
    var heading = element('div');
    heading.appendChild(element('strong', '', String(item.type_label || 'پیامک فلو')));
    heading.appendChild(element('small', '', 'حالت آزمایشی • ارسال واقعی انجام نشد'));
    identity.appendChild(heading);
    var close = element('button', 'sn-dot-test-sms-close', '×');
    close.type = 'button';
    close.setAttribute('aria-label', 'بستن پیامک آزمایشی');
    close.addEventListener('click', function () { dismiss(card); });
    header.appendChild(identity);
    header.appendChild(close);
    card.appendChild(header);

    var meta = element('div', 'sn-dot-test-sms-meta');
    meta.appendChild(element('span', '', 'مقصد: ' + String(item.phone || '—')));
    meta.appendChild(element('span', '', String(item.created_label || 'اکنون')));
    card.appendChild(meta);

    var body = element('div', 'sn-dot-test-sms-body', String(item.message || ''));
    card.appendChild(body);

    var actions = element('footer', 'sn-dot-test-sms-actions');
    var copy = element('button', 'sn-dot-test-sms-copy', 'کپی متن');
    copy.type = 'button';
    copy.addEventListener('click', function () { copyText(String(item.message || ''), copy); });
    actions.appendChild(copy);
    var url = firstSafeUrl(item.message);
    if (url) {
      var link = element('a', 'sn-dot-test-sms-link', 'ورود به صفحه');
      link.href = url;
      link.target = '_blank';
      link.rel = 'noopener noreferrer';
      actions.appendChild(link);
    }
    if (Number(item.case_id || 0) > 0) {
      actions.appendChild(element('span', 'sn-dot-test-sms-case', 'پرونده #' + String(item.case_id)));
    }
    card.appendChild(actions);

    card.addEventListener('mouseenter', function () {
      window.clearTimeout(Number(card.dataset.dismissTimer || 0));
      card.classList.add('is-paused');
    });
    card.addEventListener('mouseleave', function () {
      card.classList.remove('is-paused');
      armDismiss(card);
    });

    container.appendChild(card);
    window.requestAnimationFrame(function () { card.classList.add('is-visible'); });
    armDismiss(card);
  }

  function schedule(delay) {
    window.clearTimeout(pollTimer);
    if (!stopped) {
      pollTimer = window.setTimeout(poll, delay);
    }
  }

  function stopAndClear() {
	stopped = true;
	window.clearTimeout(pollTimer);
	var current = document.getElementById('sn-dot-test-sms-host');
	if (current) {
	  current.remove();
	}
  }

  function poll() {
    if (polling || stopped || document.visibilityState === 'hidden') {
      schedule(interval);
      return;
    }
    polling = true;
    var body = new URLSearchParams();
    body.set('action', 'sn_dot_poll_test_sms');
    body.set('nonce', nonce);
	body.set('run_id', expectedRunId);
	body.set('request_id', String(Date.now()));
    fetch(ajaxUrl, {
      method: 'POST',
      credentials: 'same-origin',
	  cache: 'no-store',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded; charset=UTF-8' },
      body: body.toString()
    }).then(function (response) {
      if (!response.ok) {
        throw new Error('poll_failed');
      }
      return response.json();
    }).then(function (payload) {
      var data = payload && payload.data ? payload.data : {};
	  if (!(payload && payload.success && data.enabled === true)) {
		stopAndClear();
		return;
	  }
	  if (String(data.run_id || '') !== expectedRunId) {
		stopAndClear();
		return;
	  }
      var messages = payload && payload.success && Array.isArray(data.messages) ? data.messages : [];
      messages.forEach(showMessage);
    }).catch(function () {
      schedule(Math.max(interval * 2, 8000));
    }).finally(function () {
      polling = false;
      schedule(interval);
    });
  }

  document.addEventListener('visibilitychange', function () {
    if (document.visibilityState === 'visible' && !stopped) {
      schedule(120);
    }
  });

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function () { schedule(250); });
  } else {
    schedule(250);
  }
})();
