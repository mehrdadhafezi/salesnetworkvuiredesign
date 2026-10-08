(function () {
    'use strict';

    var digitMap = {
        '۰': '0', '۱': '1', '۲': '2', '۳': '3', '۴': '4',
        '۵': '5', '۶': '6', '۷': '7', '۸': '8', '۹': '9',
        '٠': '0', '١': '1', '٢': '2', '٣': '3', '٤': '4',
        '٥': '5', '٦': '6', '٧': '7', '٨': '8', '٩': '9'
    };

    function normalizeDigits(value) {
        return String(value || '').replace(/[۰-۹٠-٩]/g, function (char) {
            return digitMap[char] || char;
        });
    }

    function apiPost(url, data, keepalive) {
        var body = new URLSearchParams();
        Object.keys(data).forEach(function (key) {
            body.append(key, data[key] == null ? '' : String(data[key]));
        });
        return fetch(url, {
            method: 'POST',
            credentials: 'same-origin',
            keepalive: !!keepalive,
            headers: {'Content-Type': 'application/x-www-form-urlencoded; charset=UTF-8'},
            body: body.toString()
        }).then(function (response) {
            return response.json().catch(function () {
                throw new Error('پاسخ نامعتبر از سرور دریافت شد.');
            });
        });
    }

    var campaignKeys = ['utm_source', 'utm_medium', 'utm_campaign', 'utm_content', 'utm_term'];

    function campaignHashParams() {
        var hash = String(window.location.hash || '').replace(/^#/, '');
        var question = hash.indexOf('?');
        return new URLSearchParams(question >= 0 ? hash.substring(question + 1) : '');
    }

    function campaignValues() {
        var query = new URLSearchParams(window.location.search || '');
        var fragment = campaignHashParams();
        var stored = {};
        try {
            stored = JSON.parse(window.sessionStorage.getItem('sn_campaign_utm_v1') || '{}');
            if (!stored || typeof stored !== 'object') { stored = {}; }
        } catch (e) { stored = {}; }
        var values = {};
        campaignKeys.forEach(function (key) {
            values[key] = String(query.get(key) || fragment.get(key) || stored[key] || '').trim().substring(0, 191);
        });
        values.sn_campaign_landing_url = window.location.href.substring(0, 2000);
        return values;
    }

    function campaignClientKey() {
        var key = '';
        try { key = window.localStorage.getItem('sn_campaign_client_v1') || ''; } catch (e) {}
        if (/^[a-zA-Z0-9_-]{16,80}$/.test(key)) { return key; }
        try {
            if (window.crypto && window.crypto.getRandomValues) {
                var bytes = new Uint8Array(16);
                window.crypto.getRandomValues(bytes);
                key = Array.prototype.map.call(bytes, function (value) {
                    return value.toString(16).padStart(2, '0');
                }).join('');
            }
        } catch (e) {}
        if (!key) { key = String(Date.now()) + '_' + Math.random().toString(36).slice(2) + Math.random().toString(36).slice(2); }
        try { window.localStorage.setItem('sn_campaign_client_v1', key); } catch (e) {}
        return key;
    }

    function setupForm(form) {
        var phone = form.querySelector('input[name="customer_phone"]');
        var nationalId = form.querySelector('input[name="national_id"]');
        var button = form.querySelector('.sn-dot-marketing-submit');
        var proofInput = form.querySelector('input[name="sn_dot_marketing_otp_proof"]');
        var nonceInput = form.querySelector('input[name="sn_dot_marketing_nonce"]');
        var formIdInput = form.querySelector('input[name="sn_dot_marketing_form_id"]');
        var submissionInput = form.querySelector('input[name="sn_dot_marketing_submission_id"]');
        var eventTokenInput = form.querySelector('input[name="sn_dot_marketing_event_token"]');
        var clientKeyInput = form.querySelector('input[name="sn_campaign_client_key"]');
        var wrap = form.closest('.sn-dot-marketing-wrap');
        var modal = wrap ? wrap.querySelector('.sn-dot-marketing-modal') : null;
        var formMessage = form.querySelector('[data-mkt-form-message]');
        var otpEnabled = form.getAttribute('data-otp-enabled') === '1';
        var ajaxUrl = form.getAttribute('data-ajax-url') || '';
        var challenge = '';
        var resendTimer = null;
        var requestBusy = false;
        var exitRequest = Promise.resolve();
        var campaignData = campaignValues();
		var clientKey = campaignClientKey();
        campaignKeys.forEach(function (key) {
            var input = form.querySelector('input[name="' + key + '"]');
            if (input && campaignData[key]) { input.value = campaignData[key]; }
        });
        var landingInput = form.querySelector('input[name="sn_campaign_landing_url"]');
        if (landingInput) { landingInput.value = campaignData.sn_campaign_landing_url; }
        if (clientKeyInput) { clientKeyInput.value = clientKey; }
        if (phone) {
            phone.addEventListener('input', function () {
                phone.value = normalizeDigits(phone.value).replace(/[^0-9+]/g, '').slice(0, 13);
                if (proofInput) { proofInput.value = ''; }
                if (submissionInput) { submissionInput.value = ''; }
                if (eventTokenInput) { eventTokenInput.value = ''; }
                challenge = '';
            });
        }
        if (nationalId) {
            nationalId.addEventListener('input', function () {
                nationalId.value = normalizeDigits(nationalId.value).replace(/[^0-9]/g, '').slice(0, 10);
            });
        }

        function setSubmitBusy() {
            if (!button) { return; }
            button.disabled = true;
            button.setAttribute('aria-busy', 'true');
            if (!button.dataset.originalHtml) {
                button.dataset.originalHtml = button.innerHTML;
            }
            button.textContent = 'در حال انتقال به درگاه...';
        }

        function submitToGateway() {
            closeModal();
            setSubmitBusy();
            window.HTMLFormElement.prototype.submit.call(form);
        }

        if (!modal) {
            form.addEventListener('submit', function () {
                setSubmitBusy();
            });
            return;
        }

        var otpStep = modal.querySelector('.sn-dot-marketing-otp-step');
        var confirmStep = modal.querySelector('.sn-dot-marketing-confirm-step');
        var continueButton = modal.querySelector('[data-mkt-continue]');
        var cancelButton = modal.querySelector('[data-mkt-cancel]');
        var verifyButton = modal.querySelector('[data-mkt-verify]');
        var resendButton = modal.querySelector('[data-mkt-resend]');
        var otpInput = modal.querySelector('.sn-dot-marketing-otp-input');
        var otpMessage = modal.querySelector('.sn-dot-marketing-otp-message');
        var otpPhone = modal.querySelector('[data-mkt-otp-phone]');
        var countdown = modal.querySelector('[data-mkt-countdown]');

        function setMessage(message, isError) {
            if (!otpMessage) { return; }
            otpMessage.textContent = message || '';
            otpMessage.classList.toggle('is-error', !!isError);
            otpMessage.classList.toggle('is-success', !!message && !isError);
        }

        function setFormMessage(message, isError) {
            if (!formMessage) { return; }
            formMessage.textContent = message || '';
            formMessage.hidden = !message;
            formMessage.classList.toggle('is-success', !!message && !isError);
        }

        function showOtpStep(maskedPhone) {
            if (confirmStep) { confirmStep.hidden = true; }
            if (otpStep) { otpStep.hidden = false; }
            if (otpPhone) { otpPhone.textContent = maskedPhone || (phone ? phone.value : ''); }
            if (otpInput) {
                otpInput.value = '';
                window.setTimeout(function () { otpInput.focus(); }, 80);
            }
        }

        function openOtpModal() {
            if (confirmStep) { confirmStep.hidden = false; }
            if (otpStep) { otpStep.hidden = true; }
            modal.hidden = false;
            modal.setAttribute('aria-hidden', 'false');
            document.documentElement.classList.add('sn-dot-marketing-modal-open');
        }

        function saveLeadSnapshot(done) {
            if (requestBusy) { return; }
            if (!ajaxUrl || !nonceInput) {
                setFormMessage('ارتباط امن فرم آماده نیست؛ صفحه را تازه‌سازی و دوباره تلاش کنید.', true);
                return;
            }
            requestBusy = true;
            setFormMessage('', false);
            setControlBusy(button, true, 'در حال ثبت درخواست...');
            exitRequest.then(function () {
                var requestData = {
                    action: 'sn_dot_marketing_save_lead',
                    nonce: nonceInput.value,
                    customer_name: (form.querySelector('[name="customer_name"]') || {}).value || '',
                    customer_phone: phone ? phone.value : '',
                    national_id: nationalId ? nationalId.value : '',
                    province: (form.querySelector('[name="province"]') || {}).value || '',
                    city: (form.querySelector('[name="city"]') || {}).value || '',
                    sn_dot_marketing_form_id: formIdInput ? formIdInput.value : 'default',
                    sn_dot_marketing_submission_id: submissionInput ? submissionInput.value : '',
                    event_token: eventTokenInput ? eventTokenInput.value : '',
                    sn_campaign_client_key: clientKey,
                    sn_campaign_landing_url: campaignData.sn_campaign_landing_url
                };
                campaignKeys.forEach(function (key) { requestData[key] = campaignData[key] || ''; });
                return apiPost(ajaxUrl, requestData);
            }).then(function (json) {
                if (!json || !json.success || !json.data || !json.data.submission_id || !json.data.event_token) {
                    throw new Error(json && json.data && json.data.message ? json.data.message : 'ثبت اطلاعات فرم انجام نشد.');
                }
                if (submissionInput) { submissionInput.value = String(json.data.submission_id); }
                if (eventTokenInput) { eventTokenInput.value = String(json.data.event_token); }
                requestBusy = false;
                setControlBusy(button, false);
                if (done) { done(); }
                recordFunnelEvent('confirmation_opened');
            }).catch(function (error) {
                requestBusy = false;
                setControlBusy(button, false);
                setFormMessage(error && error.message ? error.message : 'ثبت اطلاعات فرم انجام نشد؛ دوباره تلاش کنید.', true);
            });
        }

        function recordPreGatewayExit(reason, keepalive) {
            if (!ajaxUrl || !nonceInput || !submissionInput || !submissionInput.value || !eventTokenInput || !eventTokenInput.value) { return; }
            var eventName = otpStep && !otpStep.hidden ? 'otp_cancelled' : 'popup_cancelled';
            exitRequest = apiPost(ajaxUrl, {
                action: 'sn_dot_marketing_record_exit',
                nonce: nonceInput.value,
                customer_phone: phone ? phone.value : '',
                sn_dot_marketing_form_id: formIdInput ? formIdInput.value : 'default',
                sn_dot_marketing_submission_id: submissionInput.value,
                event_token: eventTokenInput.value,
                event: eventName,
                reason: reason || 'close_button'
            }, keepalive).catch(function () {
                return null;
            });
        }

        function recordFunnelEvent(eventName, reason, keepalive) {
            if (!ajaxUrl || !nonceInput || !submissionInput || !submissionInput.value || !eventTokenInput || !eventTokenInput.value) { return Promise.resolve(null); }
            return apiPost(ajaxUrl, {
                action: 'sn_dot_marketing_record_event',
                nonce: nonceInput.value,
                customer_phone: phone ? phone.value : '',
                sn_dot_marketing_form_id: formIdInput ? formIdInput.value : 'default',
                sn_dot_marketing_submission_id: submissionInput.value,
                event_token: eventTokenInput.value,
                event: eventName,
                reason: reason || ''
            }, keepalive).catch(function () { return null; });
        }

        function closeModal() {
            modal.hidden = true;
            modal.setAttribute('aria-hidden', 'true');
            document.documentElement.classList.remove('sn-dot-marketing-modal-open');
        }

        function setControlBusy(control, busy, text) {
            if (!control) { return; }
            control.disabled = !!busy;
            if (busy) {
                if (!control.dataset.originalText) { control.dataset.originalText = control.textContent; }
                if (text) { control.textContent = text; }
            } else if (control.dataset.originalText) {
                control.textContent = control.dataset.originalText;
            }
        }

        function startResendCountdown(seconds) {
            if (!resendButton) { return; }
            if (resendTimer) { window.clearInterval(resendTimer); }
            var remain = Math.max(0, Number(seconds) || 60);
            resendButton.disabled = true;
            function render() {
                if (countdown) { countdown.textContent = remain > 0 ? '(' + remain + ' ثانیه)' : ''; }
                if (remain <= 0) {
                    window.clearInterval(resendTimer);
                    resendTimer = null;
                    resendButton.disabled = false;
                    return;
                }
                remain -= 1;
            }
            render();
            resendTimer = window.setInterval(render, 1000);
        }

        function sendOtp() {
            if (requestBusy || !ajaxUrl || !phone || !nonceInput) { return; }
            requestBusy = true;
            setControlBusy(verifyButton, true, 'در حال ارسال کد...');
            if (resendButton) { resendButton.disabled = true; }
            setMessage('در حال ارسال کد تأیید...', false);
            apiPost(ajaxUrl, {
                action: 'sn_dot_marketing_send_otp',
                nonce: nonceInput.value,
                phone: phone.value,
                sn_dot_marketing_form_id: formIdInput ? formIdInput.value : 'default',
                sn_dot_marketing_submission_id: submissionInput ? submissionInput.value : '',
                event_token: eventTokenInput ? eventTokenInput.value : '',
                sn_campaign_client_key: clientKey
            }).then(function (json) {
                if (!json || !json.success || !json.data || !json.data.challenge) {
                    throw new Error(json && json.data && json.data.message ? json.data.message : 'ارسال کد تأیید انجام نشد.');
                }
                challenge = String(json.data.challenge);
                showOtpStep(json.data.phone || phone.value);
                setMessage('کد تأیید ارسال شد.', false);
                startResendCountdown(60);
            }).catch(function (error) {
                setMessage(error && error.message ? error.message : 'ارسال کد تأیید انجام نشد.', true);
            }).finally(function () {
                requestBusy = false;
                setControlBusy(verifyButton, false);
                if (resendButton && !resendTimer) { resendButton.disabled = false; }
            });
        }

        function verifyOtp() {
            if (requestBusy || !ajaxUrl || !phone || !nonceInput || !challenge || !otpInput) { return; }
            var code = normalizeDigits(otpInput.value).replace(/\D+/g, '').slice(0, 6);
            otpInput.value = code;
            if (!/^\d{6}$/.test(code)) {
                setMessage('کد ۶ رقمی را کامل وارد کنید.', true);
                otpInput.focus();
                return;
            }
            requestBusy = true;
            setControlBusy(verifyButton, true, 'در حال بررسی...');
            setMessage('', false);
            apiPost(ajaxUrl, {
                action: 'sn_dot_marketing_verify_otp',
                nonce: nonceInput.value,
                phone: phone.value,
                challenge: challenge,
                code: code,
                sn_dot_marketing_form_id: formIdInput ? formIdInput.value : 'default',
                sn_dot_marketing_submission_id: submissionInput ? submissionInput.value : '',
                event_token: eventTokenInput ? eventTokenInput.value : '',
                sn_campaign_client_key: clientKey
            }).then(function (json) {
                if (!json || !json.success || !json.data || !json.data.proof) {
                    throw new Error(json && json.data && json.data.message ? json.data.message : 'کد تأیید صحیح نیست.');
                }
                if (proofInput) { proofInput.value = String(json.data.proof); }
                setMessage('شماره موبایل تأیید شد. در حال انتقال به درگاه...', false);
                submitToGateway();
            }).catch(function (error) {
                setMessage(error && error.message ? error.message : 'کد تأیید صحیح نیست.', true);
                otpInput.focus();
            }).finally(function () {
                requestBusy = false;
                setControlBusy(verifyButton, false);
            });
        }

        form.addEventListener('submit', function (event) {
            if (button && button.disabled) { return; }
            if (!form.checkValidity()) {
                event.preventDefault();
                form.reportValidity();
                return;
            }
            event.preventDefault();
            if (proofInput) { proofInput.value = ''; }
            saveLeadSnapshot(openOtpModal);
        });

        if (continueButton) {
            continueButton.addEventListener('click', function () {
                if (requestBusy) { return; }
                if (!otpEnabled) { submitToGateway(); return; }
                showOtpStep(phone ? phone.value.trim() : '');
                if (!challenge) { sendOtp(); }
            });
        }
        if (cancelButton) {
            cancelButton.addEventListener('click', function () {
                if (!requestBusy) {
                    recordPreGatewayExit('cancel_button');
                    closeModal();
                }
            });
        }

        modal.querySelectorAll('[data-mkt-close]').forEach(function (control) {
            control.addEventListener('click', function () {
                if (!requestBusy) {
                    recordPreGatewayExit(control.classList.contains('sn-dot-marketing-modal-backdrop') ? 'backdrop' : 'close_button');
                    closeModal();
                }
            });
        });

        if (otpInput) {
            otpInput.addEventListener('input', function () {
                otpInput.value = normalizeDigits(otpInput.value).replace(/\D+/g, '').slice(0, 6);
            });
            otpInput.addEventListener('keydown', function (event) {
                if (event.key === 'Enter') {
                    event.preventDefault();
                    verifyOtp();
                }
            });
        }

        if (verifyButton) { verifyButton.addEventListener('click', verifyOtp); }
        if (resendButton) {
            resendButton.addEventListener('click', function () {
                if (resendButton.disabled) { return; }
                challenge = '';
                sendOtp();
            });
        }

        document.addEventListener('keydown', function (event) {
            if (event.key === 'Escape' && !modal.hidden && !requestBusy) {
                recordPreGatewayExit('escape');
                closeModal();
            }
        });

        window.addEventListener('pagehide', function () {
            if (!modal.hidden && !requestBusy) { recordPreGatewayExit('page_exit', true); }
        });
    }

    function enableButtons() {
        document.querySelectorAll('.sn-dot-marketing-submit[aria-busy="true"]').forEach(function (button) {
            button.disabled = false;
            button.removeAttribute('aria-busy');
            if (button.dataset.originalHtml) {
                button.innerHTML = button.dataset.originalHtml;
            }
        });
    }

    document.addEventListener('DOMContentLoaded', function () {
        document.querySelectorAll('.sn-dot-marketing-form').forEach(setupForm);
    });

    window.addEventListener('pageshow', enableButtons);
}());
