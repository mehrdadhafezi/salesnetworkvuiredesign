(function () {
  'use strict';

  var config = window.snHrTransfer || {};
  var messages = config.messages || {};

  function text(value) {
    return value === null || value === undefined ? '' : String(value);
  }

  function init(root) {
    var exportForm = root.querySelector('[data-sn-export-form]');
    var importForm = root.querySelector('[data-sn-import-form]');
    if (!exportForm) return;

    var picker = exportForm.querySelector('[data-sn-picker]');
    var searchInput = exportForm.querySelector('[data-sn-user-search]');
    var searchButton = exportForm.querySelector('[data-sn-search-button]');
    var searchStatus = exportForm.querySelector('[data-sn-search-status]');
    var resultsBox = exportForm.querySelector('[data-sn-search-results]');
    var addVisibleButton = exportForm.querySelector('[data-sn-add-visible]');
    var selectedBox = exportForm.querySelector('[data-sn-selected-users]');
    var selectedInputs = exportForm.querySelector('[data-sn-selected-inputs]');
    var selectedCount = exportForm.querySelector('[data-sn-selected-count]');
    var selected = new Map();
    var visible = [];
    var timer = 0;
    var requestNumber = 0;

    function scope() {
      var checked = exportForm.querySelector('input[name="export_scope"]:checked');
      return checked ? checked.value : 'all';
    }

    function syncScope() {
      if (picker) picker.hidden = scope() !== 'selected';
      if (scope() === 'selected' && searchInput && !searchInput.dataset.initialized) {
        searchInput.dataset.initialized = '1';
        runSearch();
      }
    }

    function userLabel(user) {
      var bits = [text(user.name) || ('#' + text(user.id))];
      if (user.position) bits.push(text(user.position));
      if (user.login) bits.push('@' + text(user.login));
      if (user.employeeCode) bits.push('کد ' + text(user.employeeCode));
      return bits.join(' — ');
    }

    function userMeta(user) {
      var bits = [];
      if (user.position) bits.push(text(user.position));
      if (user.login) bits.push('@' + text(user.login));
      if (user.employeeCode) bits.push('کد ' + text(user.employeeCode));
      if (!bits.length && user.email) bits.push(text(user.email));
      return bits.join(' — ');
    }

    function renderSelected() {
      selectedBox.textContent = '';
      selectedInputs.textContent = '';
      if (!selected.size) {
        var empty = document.createElement('span');
        empty.className = 'sn-hr-transfer-empty';
        empty.textContent = 'هنوز کاربری انتخاب نشده است.';
        selectedBox.appendChild(empty);
      }
      selected.forEach(function (user, id) {
        var chip = document.createElement('span');
        chip.className = 'sn-hr-transfer-chip';
        chip.appendChild(document.createTextNode(userLabel(user)));
        var remove = document.createElement('button');
        remove.type = 'button';
        remove.setAttribute('aria-label', 'حذف ' + userLabel(user));
        remove.textContent = '×';
        remove.addEventListener('click', function () {
          selected.delete(id);
          renderSelected();
          renderResults();
        });
        chip.appendChild(remove);
        selectedBox.appendChild(chip);

        var hidden = document.createElement('input');
        hidden.type = 'hidden';
        hidden.name = 'selected_user_ids[]';
        hidden.value = id;
        selectedInputs.appendChild(hidden);
      });
      selectedCount.textContent = selected.size.toLocaleString('fa-IR') + ' ' + (messages.person || 'نفر');
    }

    function renderResults() {
      resultsBox.textContent = '';
      visible.forEach(function (user) {
        var row = document.createElement('button');
        row.type = 'button';
        row.className = 'sn-hr-transfer-result';
        row.disabled = selected.has(String(user.id));
        var title = document.createElement('strong');
        title.textContent = text(user.name) || ('#' + text(user.id));
        var meta = document.createElement('span');
        meta.textContent = userMeta(user);
        row.appendChild(title);
        row.appendChild(meta);
        row.addEventListener('click', function () {
          selected.set(String(user.id), user);
          renderSelected();
          renderResults();
        });
        resultsBox.appendChild(row);
      });
      addVisibleButton.hidden = !visible.length;
    }

    async function runSearch() {
      if (!config.ajaxurl || !config.nonce) return;
      var query = searchInput.value.trim();
      var currentRequest = ++requestNumber;
      if (query && !/^\d+$/.test(query) && query.length < 2) {
        visible = [];
        renderResults();
        searchStatus.textContent = messages.minimumSearch || 'برای جست‌وجوی متنی حداقل ۲ حرف وارد کنید.';
        searchButton.disabled = false;
        return;
      }
      searchStatus.textContent = messages.searching || 'در حال جست‌وجو…';
      searchButton.disabled = true;
      var body = new FormData();
      body.append('action', 'sn_hr_transfer_search_users');
      body.append('nonce', config.nonce);
      body.append('q', query);
      try {
        var response = await fetch(config.ajaxurl, { method: 'POST', credentials: 'same-origin', body: body });
        var payload = await response.json();
        if (currentRequest !== requestNumber) return;
        if (!response.ok || !payload.success) throw new Error('search_failed');
        visible = Array.isArray(payload.data && payload.data.users) ? payload.data.users : [];
        searchStatus.textContent = visible.length ? visible.length.toLocaleString('fa-IR') + ' نتیجه' : (messages.noResults || 'کاربری پیدا نشد.');
        renderResults();
      } catch (error) {
        if (currentRequest !== requestNumber) return;
        visible = [];
        renderResults();
        searchStatus.textContent = messages.error || 'جست‌وجو انجام نشد.';
      } finally {
        if (currentRequest === requestNumber) searchButton.disabled = false;
      }
    }

    exportForm.querySelectorAll('input[name="export_scope"]').forEach(function (radio) {
      radio.addEventListener('change', syncScope);
    });
    searchButton.addEventListener('click', runSearch);
    searchInput.addEventListener('input', function () {
      window.clearTimeout(timer);
      timer = window.setTimeout(runSearch, 350);
    });
    addVisibleButton.addEventListener('click', function () {
      visible.forEach(function (user) { selected.set(String(user.id), user); });
      renderSelected();
      renderResults();
    });
    exportForm.addEventListener('submit', function (event) {
      if (scope() === 'selected' && !selected.size) {
        event.preventDefault();
        window.alert(messages.needSelection || 'حداقل یک کاربر را انتخاب کنید.');
      }
    });
    if (importForm) {
      importForm.addEventListener('submit', function (event) {
        var submitter = event.submitter;
        var isApply = submitter && submitter.name === 'run_mode' && submitter.value === 'apply';
        var confirmation = importForm.querySelector('[data-sn-apply-confirm]');
        if (isApply && confirmation && !confirmation.checked) {
          event.preventDefault();
          window.alert(messages.needApplyConfirm || 'برای Apply تأیید لازم است.');
        }
      });
    }
    syncScope();
    renderSelected();
  }

  function boot() {
    document.querySelectorAll('[data-sn-hr-transfer]').forEach(init);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', boot);
  } else {
    boot();
  }
})();
