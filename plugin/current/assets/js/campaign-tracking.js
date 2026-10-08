(function () {
	'use strict';
	var cfg = window.SN_CAMPAIGN_TRACKING || {};
	if (!cfg.ajaxUrl || !window.URLSearchParams || !window.fetch) return;

	var keys = ['utm_source', 'utm_medium', 'utm_campaign', 'utm_content', 'utm_term'];
	var storageKey = 'sn_campaign_utm_v1';
	var activeKey = 'sn_campaign_active';

	function hashParams() {
		var hash = String(window.location.hash || '').replace(/^#/, '');
		var question = hash.indexOf('?');
		return new URLSearchParams(question >= 0 ? hash.substring(question + 1) : '');
	}

	function readStored() {
		try {
			var parsed = JSON.parse(window.sessionStorage.getItem(storageKey) || '{}');
			return parsed && typeof parsed === 'object' ? parsed : {};
		} catch (e) {
			return {};
		}
	}

	function collectUtm() {
		var query = new URLSearchParams(window.location.search || '');
		var fragment = hashParams();
		var stored = readStored();
		var values = {};
		keys.forEach(function (key) {
			values[key] = String(query.get(key) || fragment.get(key) || stored[key] || '').trim().substring(0, 191);
		});
		return values;
	}

	function clientKey() {
		var key = '';
		try { key = window.localStorage.getItem('sn_campaign_client_v1') || ''; } catch (e) {}
		if (/^[a-zA-Z0-9_-]{16,80}$/.test(key)) return key;
		try {
			if (window.crypto && window.crypto.getRandomValues) {
				var bytes = new Uint8Array(16);
				window.crypto.getRandomValues(bytes);
				key = Array.prototype.map.call(bytes, function (value) { return value.toString(16).padStart(2, '0'); }).join('');
			}
		} catch (e) {}
		if (!key) key = String(Date.now()) + '_' + Math.random().toString(36).slice(2) + Math.random().toString(36).slice(2);
		try { window.localStorage.setItem('sn_campaign_client_v1', key); } catch (e) {}
		return key;
	}

	var values = collectUtm();
	var hasUtm = keys.some(function (key) { return values[key] !== ''; });
	if (hasUtm) {
		try {
			window.sessionStorage.setItem(activeKey, '1');
			window.sessionStorage.setItem(storageKey, JSON.stringify(values));
		} catch (e) {}
	}
	var active = hasUtm;
	if (!active) {
		try { active = window.sessionStorage.getItem(activeKey) === '1'; } catch (e) {}
	}
	if (!active) return;

	var body = new URLSearchParams();
	body.set('action', cfg.action || 'sn_campaign_track');
	body.set('event', 'landing_view');
	body.set('url', window.location.href.substring(0, 2000));
	body.set('client_key', clientKey());
	keys.forEach(function (key) { body.set(key, values[key]); });
	var exitBody = new URLSearchParams(body.toString());
	exitBody.set('event', 'landing_exit');
	var attempts = 0;
	var exitSent = false;
	var retryDelays = [800, 2500, 6000];
	function send() {
		attempts += 1;
		window.fetch(cfg.ajaxUrl, {
			method: 'POST',
			credentials: 'same-origin',
			headers: { 'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8' },
			body: body.toString(),
			keepalive: true
		}).then(function (response) {
			if (!response.ok) throw new Error('campaign_http_' + response.status);
			return response.json();
		}).then(function (json) {
			if (!json || !json.success || !json.data || json.data.captured !== true) throw new Error('campaign_not_captured');
		}).catch(function () {
			if (attempts <= retryDelays.length) {
				window.setTimeout(send, retryDelays[attempts - 1]);
			}
		});
	}
	send();
	function sendExit() {
		if (exitSent) return;
		exitSent = true;
		// The exit event is deliberately independent of the initial capture result:
		// a fast close can happen before the first request finishes. The server can
		// then create the session and store the exit in the same request.
		if (window.navigator && typeof window.navigator.sendBeacon === 'function') {
			try { window.navigator.sendBeacon(cfg.ajaxUrl, exitBody); return; } catch (e) {}
		}
		try {
			window.fetch(cfg.ajaxUrl, {
				method: 'POST',
				credentials: 'same-origin',
				headers: { 'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8' },
				body: exitBody.toString(),
				keepalive: true
			});
		} catch (e) {}
	}
	window.addEventListener('pagehide', sendExit);
	document.addEventListener('visibilitychange', function () {
		if (document.visibilityState === 'hidden') sendExit();
	});
})();
