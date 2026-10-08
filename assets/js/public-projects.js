(function ($) {
  'use strict';

  var cfg = window.snProjects || {};
  if (!cfg.ajaxUrl || !cfg.nonce) return;

  var dashboardState = {
    items: [], experts: [], role: cfg.role || '', selectedMembership: 0,
    selectedItem: 0, selectedItems: {}, page: 0, total: 0,
    hasMore: false, loading: false
  };
  var notificationState = {total: null, threads: [], loading: false, refreshQueued: false, timer: 0};
  var dashboardRequestSerial = 0, dashboardSearchTimer = 0;
  function isOperationsChatRole() { return ['operations_sales_expert','operations_sales_supervisor'].indexOf(String(cfg.role || '')) >= 0; }

  function esc(value) {
    return String(value == null ? '' : value).replace(/[&<>"']/g, function (c) {
      return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#039;'}[c];
    });
  }
  function fa(value) { try { return Number(value || 0).toLocaleString('fa-IR'); } catch (e) { return String(value || 0); } }
  function jalaliDateTime(value) {
    var raw = String(value || '').trim();
    if (!raw) return '—';
    if (/^(13|14)\d{2}[\/\-.]/.test(raw)) return raw.replace(/[\-.]/g, '/');
    try {
      var iso = raw.replace(' ', 'T');
      if (!/[zZ]|[+\-]\d{2}:?\d{2}$/.test(iso)) iso += '+03:30';
      var date = new Date(iso);
      if (isNaN(date.getTime())) return raw;
      return new Intl.DateTimeFormat('fa-IR-u-ca-persian', {timeZone:'Asia/Tehran',year:'numeric',month:'2-digit',day:'2-digit',hour:'2-digit',minute:'2-digit',hourCycle:'h23'}).format(date);
    } catch (e) { return raw; }
  }
  function request(action, data) {
    return $.ajax({url: cfg.ajaxUrl, method: 'POST', dataType: 'json', timeout: 25000, data: $.extend({action: action, nonce: cfg.nonce}, data || {})});
  }
  function messageOf(res, fallback) { return (res && res.message) || fallback || 'عملیات انجام نشد.'; }
  function toast(message, ok) {
    var $box = $('#sn-project-toast');
    if (!$box.length) $box = $('<div id="sn-project-toast" class="sn-project-toast" role="status"></div>').appendTo('body');
    $box.attr('class', 'sn-project-toast ' + (ok ? 'is-success' : 'is-error')).text(message || '').addClass('is-visible');
    window.clearTimeout(window.snProjectToastTimer);
    window.snProjectToastTimer = window.setTimeout(function () {$box.removeClass('is-visible');}, 4500);
  }

  function ensureModal() {
    if ($('#sn-project-modal').length) return;
    $('body').append(
      '<div id="sn-project-modal" class="sn-project-modal" aria-hidden="true" dir="rtl">' +
        '<div class="sn-project-modal-backdrop sn-project-modal-close"></div>' +
        '<div class="sn-project-modal-card" role="dialog" aria-modal="true" aria-labelledby="sn-project-modal-title">' +
          '<header><div><span class="sn-project-kicker">BIAVIN PROJECT</span><h2 id="sn-project-modal-title">پرونده اشتراک</h2></div><button type="button" class="sn-project-modal-x sn-project-modal-close" aria-label="بستن">×</button></header>' +
          '<div class="sn-project-modal-body"><div class="sn-project-modal-loading">در حال دریافت پرونده...</div></div>' +
        '</div>' +
      '</div>'
    );
  }
  function closeModal() { $('#sn-project-modal').removeClass('is-open').attr('aria-hidden', 'true'); $('body').removeClass('sn-project-modal-open'); }
  function openModal() { ensureModal(); $('#sn-project-modal').addClass('is-open').attr('aria-hidden', 'false'); $('body').addClass('sn-project-modal-open'); }

  function loadThread(membershipId, itemId) {
    dashboardState.selectedMembership = Number(membershipId || 0);
    dashboardState.selectedItem = Number(itemId || 0);
    openModal();
    $('#sn-project-modal .sn-project-modal-body').html('<div class="sn-project-modal-loading">در حال دریافت پرونده...</div>');
    request('sn_project_thread', {membership_id: membershipId, item_id: itemId || 0}).done(function (res) {
      if (!res || !res.success) {
        $('#sn-project-modal .sn-project-modal-body').html('<div class="sn-notice sn-error">' + esc(messageOf(res, 'پرونده دریافت نشد.')) + '</div>');
        return;
      }
      dashboardState.selectedItem = Number(res.selected_item_id || 0);
      renderThread(res);
      window.setTimeout(function () { loadNotifications(true); }, 120);
    }).fail(function (xhr) {
      var res = xhr && xhr.responseJSON;
      $('#sn-project-modal .sn-project-modal-body').html('<div class="sn-notice sn-error">' + esc(messageOf(res, 'خطای ارتباط با سرور.')) + '</div>');
    });
  }

  function unreadFor(membershipId, itemId) {
    var total = 0;
    notificationState.threads.forEach(function (row) {
      var sameItem = Number(itemId || 0) > 0 && Number(row.item_id) === Number(itemId);
      var sameMembership = Number(itemId || 0) < 1 && Number(row.membership_id) === Number(membershipId);
      if (sameItem || sameMembership) total += Number(row.unread_count || 0);
    });
    return total;
  }
  function unreadBadge(membershipId, itemId) {
    var count = unreadFor(membershipId, itemId);
    return count > 0 ? '<span class="sn-project-inline-unread" aria-label="' + esc(fa(count) + ' پیام خوانده‌نشده') + '">' + fa(count) + '</span>' : '';
  }
  function itemTabs(items, selectedId) {
    return '<div class="sn-project-content-tabs">' + (items || []).map(function (item) {
      var active = Number(item.id) === Number(selectedId) ? ' is-active' : '';
      var verified = item.purchase_verified ? '<i title="خرید تاییدشده">✓</i>' : '';
      return '<button type="button" class="sn-project-content-tab' + active + '" data-item="' + esc(item.id) + '"><strong>' + esc(item.content_name) + '</strong><span>' + esc(item.workflow_status_label) + '</span>' + verified + unreadBadge(item.membership_id, item.id) + '</button>';
    }).join('') + '</div>';
  }
  function selectedItem(res) {
    var found = null;
    (res.items || []).forEach(function (item) { if (Number(item.id) === Number(res.selected_item_id)) found = item; });
    return found || (res.items || [])[0] || {};
  }
  function invoiceList(item) {
    var rows = item.invoices || [];
    if (!rows.length) return '<div class="sn-project-empty-mini">هنوز فاکتور تاییدشده‌ای به این محتوا نسبت داده نشده است.</div>';
    return '<div class="sn-project-invoices-mini">' + rows.map(function (row) {
      return '<div><code>' + esc(row.invoice_code || ('#' + row.invoice_id)) + '</code><strong>' + esc(row.amount_fmt) + '</strong><small>' + esc(jalaliDateTime(row.verified_at || row.paid_at || '')) + '</small></div>';
    }).join('') + '</div>';
  }
  function messageList(messages) {
    if (!messages || !messages.length) return '<div class="sn-project-messages"><div class="sn-project-chat-empty"><span>💬</span><strong>شروع گفت‌وگو</strong><small>هنوز پیامی در این محتوا ارسال نشده است.</small></div></div>';
    return '<div class="sn-project-messages">' + messages.map(function (msg) {
      var file = msg.attachment ? '<a class="sn-project-attachment" href="' + esc(msg.attachment.url) + '"><span>📎</span><span><strong>' + esc(msg.attachment.name) + '</strong><small>' + fa(Math.ceil(Number(msg.attachment.size || 0) / 1024)) + ' KB</small></span></a>' : '';
      var initial = String(msg.sender_name || '؟').trim().charAt(0) || '؟';
      return '<article class="sn-project-message-row ' + (msg.is_self ? 'is-self' : '') + '"><span class="sn-project-message-avatar">' + esc(initial) + '</span><div class="sn-project-message-bubble"><header><strong>' + esc(msg.sender_name) + '</strong><time>' + esc(jalaliDateTime(msg.created_at)) + '</time></header>' + (msg.text ? '<p>' + esc(msg.text).replace(/\n/g, '<br>') + '</p>' : '') + file + '</div></article>';
    }).join('') + '</div>';
  }
  function noteList(notes) {
    if (!notes || !notes.length) return '<div class="sn-project-empty-mini">یادداشت داخلی ثبت نشده است.</div>';
    return '<div class="sn-project-notes">' + notes.map(function (note) { return '<div><strong>' + esc(note.author_name) + '</strong><p>' + esc(note.note).replace(/\n/g, '<br>') + '</p><small>' + esc(jalaliDateTime(note.created_at)) + '</small></div>'; }).join('') + '</div>';
  }
  function managementControls(item, canManage) {
    if (!canManage) return '';
    var statuses = {waiting:'در انتظار', no_answer:'جواب نداده', follow_up:'تماس مجدد', cancelled:'انصراف', normal:'عادی', upsell:'افزایشی', purchased:'خرید کرده (قدیمی)', regretted:'پشیمان شده (قدیمی)'};
    var options = Object.keys(statuses).map(function (key) { return '<option value="' + key + '" ' + (String(item.workflow_status) === key ? 'selected' : '') + '>' + statuses[key] + '</option>'; }).join('');
    var expert = '';
    if (dashboardState.role === 'project_manager' || dashboardState.role === 'admin') {
      var expertOptions = '<option value="0">— بدون کارشناس —</option>' + dashboardState.experts.map(function (row) { return '<option value="' + esc(row.id) + '" ' + (Number(item.expert_user_id) === Number(row.id) ? 'selected' : '') + '>' + esc(row.name) + '</option>'; }).join('');
      expert = '<label>کارشناس پروژه<select class="sn-project-expert-select" data-item="' + esc(item.id) + '">' + expertOptions + '</select></label>';
    }
	var action = '';
	if (item.workflow_status === 'normal' || item.workflow_status === 'upsell') {
	  var latest = item.latest_action || null, isUpsell = item.workflow_status === 'upsell';
	  var catalog = isUpsell && item.upgrade_rules_locked ? (item.upgrade_options || []) : (item.action_catalog || []);
	  var emptyLabel = isUpsell && item.upgrade_rules_locked && !catalog.length ? '— حالت افزایشی برای این کارت تعریف نشده —' : '— انتخاب محصول مقصد —';
	  var targetOptions = '<option value="">' + emptyLabel + '</option>' + catalog.map(function (row) { var price = Number(row.price || row.upgrade_amount || 0); return '<option value="' + esc(row.id || row.product_id) + '" data-credit="' + esc(row.credit || row.target_credit) + '" data-price="' + esc(price) + '">' + esc(row.name) + ' · اعتبار ' + fa(row.credit || row.target_credit) + (isUpsell && item.upgrade_rules_locked ? (price > 0 ? ' · پرداخت ' + fa(price) + ' تومان' : ' · رایگان') : '') + '</option>'; }).join('');
	  action = '<div class="sn-project-action-box"><strong>اقدام ' + (item.workflow_status === 'upsell' ? 'افزایشی' : 'عادی') + '</strong>' + (latest ? '<span class="sn-project-action-state">آخرین اقدام: ' + esc(latest.status) + (latest.target_product_name ? ' · ' + esc(latest.target_product_name) : '') + '</span>' : '') +
		'<div class="sn-project-action-form"><select class="sn-project-action-mode"><option value="product">انتخاب محصول</option><option value="wallet">شارژ کیف پول</option></select><select class="sn-project-action-target">' + targetOptions + '</select><input class="sn-project-action-credit" type="number" min="0" step="1" placeholder="اعتبار مقصد"' + (isUpsell && item.upgrade_rules_locked ? ' readonly' : '') + '><input class="sn-project-action-amount" type="number" min="0" step="1" placeholder="مبلغ فاکتور (تومان)"' + (isUpsell && item.upgrade_rules_locked ? ' readonly' : '') + '><button type="button" class="sn-btn sn-btn-sm sn-btn-primary sn-project-create-action" data-item="' + esc(item.id) + '" data-action-type="' + esc(item.workflow_status) + '"' + (isUpsell && item.upgrade_rules_locked && !catalog.length ? ' disabled' : '') + '>ثبت اقدام</button></div>' +
        (latest && ['awaiting_payment','retry_requested','unpaid'].indexOf(String(latest.status)) !== -1 ? '<div class="sn-project-action-buttons"><button type="button" class="sn-btn sn-btn-sm sn-project-retry-action" data-action="' + esc(latest.id) + '">فرصت مجدد پرداخت</button>' + (item.workflow_status === 'upsell' ? '<button type="button" class="sn-btn sn-btn-sm sn-btn-danger sn-project-cancel-upsell" data-action="' + esc(latest.id) + '">لغو افزایشی و ادامه عادی</button>' : '') + '</div>' : '') +
        (latest && ['ready','paid'].indexOf(String(latest.status)) !== -1 ? '<button type="button" class="sn-btn sn-btn-sm sn-project-fulfill-action" data-action="' + esc(latest.id) + '">ثبت تکمیل اقدام</button>' : '') + '</div>';
    }
    var apiAction = '';
    if (item.api_enabled && (item.workflow_status === 'normal' || item.workflow_status === 'upsell')) {
      apiAction = '<div class="sn-project-api-box" data-item="' + esc(item.id) + '"><header><div><strong>تحویل در سایت پروژه</strong><small>اتصال: ' + esc(item.api_project_code || 'پروژه مقصد') + '</small></div><span>API امن</span></header>' +
        '<section><h4>خرید مستقیم برای مشتری</h4><div class="sn-project-api-product-search"><input type="search" class="sn-project-api-search" placeholder="جستجوی محصول سایت مقصد"><button type="button" class="sn-btn sn-btn-sm sn-project-api-load-products">دریافت محصولات</button></div><select class="sn-project-api-product"><option value="">ابتدا محصولات را دریافت کنید</option></select><input type="number" class="sn-project-api-quantity" min="1" max="20" value="1" aria-label="تعداد"><button type="button" class="sn-btn sn-btn-primary sn-project-api-purchase">ثبت سفارش مشتری</button></section>' +
        '<section><h4>شارژ کیف پول مشتری</h4><select class="sn-project-api-wallet-type"><option value="cash">کیف پول نقدی</option><option value="installment">کیف پول اقساطی</option></select><input type="number" class="sn-project-api-wallet-amount" min="1" step="1" value="' + esc(item.content_credit || '') + '" placeholder="مبلغ شارژ"><input type="text" class="sn-project-api-note" maxlength="250" placeholder="یادداشت اختیاری"><button type="button" class="sn-btn sn-btn-primary sn-project-api-charge">شارژ کیف پول</button></section><div class="sn-project-api-result" aria-live="polite"></div></div>';
    }
    return '<div class="sn-project-manage-controls"><label>وضعیت پیگیری<select class="sn-project-status-select" data-item="' + esc(item.id) + '" data-current-status="' + esc(item.workflow_status || 'waiting') + '">' + options + '</select></label>' + expert + '<button type="button" class="sn-btn sn-btn-sm sn-project-add-note" data-item="' + esc(item.id) + '">یادداشت داخلی</button><button type="button" class="sn-btn sn-btn-sm sn-project-link-invoice" data-item="' + esc(item.id) + '">انتساب فاکتور تاییدشده</button></div>' + action + apiAction;
  }
  function renderThread(res) {
    var item = selectedItem(res);
    var verifiedTag = item.purchase_verified ? '<span class="sn-project-verified-tag">خرید تاییدشده</span>' : '<span class="sn-project-pending-tag">بدون خرید تاییدشده</span>';
    var membershipTag = String(item.membership_status || 'active') === 'active' ? '' : '<span class="sn-project-pending-tag">اشتراک معلق مالی</span>';
    var opsSide = isOperationsChatRole();
    var participantLabel = opsSide ? 'طرف فروش' : 'کارشناس فروش عملیات';
    var participantName = opsSide ? (item.sales_chat_name || item.origin_name || '—') : (item.operations_sales_expert_name || item.expert_name || '—');
    var chatTitle = opsSide ? 'گفت‌وگو با چرخه فروش' : 'گفت‌وگوی بیاوین';
    var chatSubtitle = opsSide ? 'ارتباط مستقیم با آخرین صادرکننده فاکتور اشتراک' : 'ارتباط مستقیم با کارشناس فروش عملیات پرونده';
    var html = itemTabs(res.items, res.selected_item_id) +
      '<section class="sn-project-item-summary"><div><span>اشتراک</span><strong>' + esc(item.subscription_name || '—') + '</strong></div><div><span>محتوا</span><strong>' + esc(item.content_name || '—') + '</strong></div><div><span>مشتری</span><strong>' + esc(item.customer_name || '—') + '</strong><a href="tel:' + esc(item.customer_phone || '') + '">' + esc(item.customer_phone || '—') + '</a></div><div><span>' + esc(participantLabel) + '</span><strong>' + esc(participantName) + '</strong></div></section>' +
      '<div class="sn-project-state-line"><span class="sn-project-workflow-tag">' + esc(item.workflow_status_label || 'در انتظار') + '</span>' + membershipTag + verifiedTag + '<strong>مجموع خرید تاییدشده: ' + esc(item.purchase_total_fmt || '۰ تومان') + '</strong>' + (item.cancel_reason ? '<small class="sn-project-cancel-reason">علت انصراف: ' + esc(item.cancel_reason) + '</small>' : '') + '</div>' +
      managementControls(item, !!res.can_manage) +
      '<details class="sn-project-invoice-details"><summary>فاکتورهای منتسب به این محتوا (' + fa(item.purchase_invoice_count || 0) + ')</summary>' + invoiceList(item) + '</details>' +
      (res.can_manage ? '<details class="sn-project-note-details"><summary>یادداشت‌های داخلی</summary>' + noteList(res.notes) + '</details>' : '') +
      '<section class="sn-project-chat"><header class="sn-project-chat-head"><div class="sn-project-chat-symbol">✦</div><div><h3>' + esc(chatTitle) + '</h3><p>' + esc(chatSubtitle) + '</p></div><span class="sn-project-chat-live">فعال</span></header>' + messageList(res.messages) +
        '<form class="sn-project-message-form" enctype="multipart/form-data"><input type="hidden" name="item_id" value="' + esc(item.id) + '"><textarea name="message" rows="3" placeholder="پیام خود را بنویسید..."></textarea><div class="sn-project-composer-tools"><label class="sn-project-file-label">📎 افزودن فایل<input type="file" name="attachment" accept=".jpg,.jpeg,.png,.webp,.pdf,.txt,.doc,.docx,.xls,.xlsx"></label><span class="sn-project-selected-file">حداکثر ۸ مگابایت</span><button type="submit" class="sn-btn sn-btn-primary">ارسال پیام <span aria-hidden="true">←</span></button></div></form>' +
      '</section>';
    $('#sn-project-modal-title').text((item.customer_name || 'پرونده اشتراک') + ' · ' + (item.subscription_name || ''));
    $('#sn-project-modal .sn-project-modal-body').html(html);
    var $messages = $('#sn-project-modal .sn-project-messages');
    if ($messages.length) $messages.scrollTop($messages[0].scrollHeight);
  }

  $(document).on('click', '.sn-project-chat-btn', function () {
    var membershipId = Number($(this).data('membership') || 0), itemId = Number($(this).data('item') || 0);
    if (!itemId) {
      notificationState.threads.some(function (row) {
        if (Number(row.membership_id) !== membershipId) return false;
        itemId = Number(row.item_id || 0); return true;
      });
    }
    loadThread(membershipId, itemId);
  });
  $(document).on('click', '.sn-project-modal-close', closeModal);
  $(document).on('keydown', function (event) { if (event.key === 'Escape') closeModal(); });
  $(document).on('click', '.sn-project-content-tab', function () { loadThread(dashboardState.selectedMembership, $(this).data('item')); });
  $(document).on('change', '.sn-project-message-form input[type="file"]', function () {
    var file = this.files && this.files[0];
    $(this).closest('form').find('.sn-project-selected-file').text(file ? file.name : 'حداکثر ۸ مگابایت');
  });

  $(document).on('submit', '.sn-project-message-form', function (event) {
    event.preventDefault();
    var form = this, file = form.elements.attachment && form.elements.attachment.files[0];
    if (file && Number(file.size) > Number(cfg.maxFileSize || 8388608)) { toast('حجم فایل بیشتر از ۸ مگابایت است.', false); return; }
    var data = new FormData(form); data.append('action', 'sn_project_send_message'); data.append('nonce', cfg.nonce);
    var $button = $(form).find('button[type="submit"]').prop('disabled', true).text('در حال ارسال...');
    $.ajax({url: cfg.ajaxUrl, method: 'POST', data: data, processData: false, contentType: false, dataType: 'json', timeout: 45000}).done(function (res) {
      if (res && res.success) { loadThread(dashboardState.selectedMembership, dashboardState.selectedItem); }
      else toast(messageOf(res, 'ارسال پیام انجام نشد.'), false);
    }).fail(function (xhr) { toast(messageOf(xhr && xhr.responseJSON, 'خطای شبکه در ارسال پیام.'), false); }).always(function () {$button.prop('disabled', false).html('ارسال پیام <span aria-hidden="true">←</span>');});
  });

  $(document).on('click', '.sn-project-api-load-products', function () {
    var $button = $(this), $box = $button.closest('.sn-project-api-box'), itemId = $box.data('item'), search = $box.find('.sn-project-api-search').val() || '';
    $button.prop('disabled', true).text('در حال دریافت...');
    request('sn_project_api_products', {item_id: itemId, search: search}).done(function (res) {
      var $select = $box.find('.sn-project-api-product').empty();
      if (!res || !res.success || !res.products || !res.products.length) { $select.append('<option value="">محصولی پیدا نشد</option>'); toast(messageOf(res, 'محصولی در سایت مقصد پیدا نشد.'), false); return; }
      $select.append('<option value="">— انتخاب محصول —</option>');
      res.products.forEach(function (product) { $select.append($('<option/>', {value: product.id, text: (product.name || ('#' + product.id)) + ' · ' + money(product.price || 0)}).attr('data-name', product.name || '')); });
      toast('فهرست محصولات سایت مقصد دریافت شد.', true);
    }).fail(function (xhr) { toast(messageOf(xhr && xhr.responseJSON, 'دریافت محصولات سایت مقصد ناموفق بود.'), false); }).always(function () { $button.prop('disabled', false).text('دریافت محصولات'); });
  });

  $(document).on('click', '.sn-project-api-purchase', function () {
    var $button = $(this), $box = $button.closest('.sn-project-api-box'), $product = $box.find('.sn-project-api-product option:selected'), productId = Number($product.val() || 0);
    if (!productId) { toast('ابتدا محصول سایت مقصد را انتخاب کنید.', false); return; }
    $button.prop('disabled', true).text('در حال ثبت سفارش...');
    request('sn_project_api_purchase', {item_id: $box.data('item'), product_id: productId, product_name: $product.attr('data-name') || $product.text().split(' · ')[0], quantity: Number($box.find('.sn-project-api-quantity').val() || 1)}).done(function (res) {
      if (!res || !res.success) { toast(messageOf(res, 'ثبت سفارش انجام نشد.'), false); return; }
      toast(messageOf(res, 'سفارش مشتری ثبت شد.'), true);
      if (res.payment_url) { window.open(res.payment_url, '_blank', 'noopener'); }
      $('.sn-project-api-result').text(res.payment_url ? 'لینک پرداخت سفارش در تب جدید باز شد.' : 'سفارش در سایت مقصد ثبت شد.');
      loadThread(dashboardState.selectedMembership, dashboardState.selectedItem);
    }).fail(function (xhr) { toast(messageOf(xhr && xhr.responseJSON, 'ثبت سفارش سایت مقصد ناموفق بود.'), false); }).always(function () { $button.prop('disabled', false).text('ثبت سفارش مشتری'); });
  });

  $(document).on('click', '.sn-project-api-charge', function () {
    var $button = $(this), $box = $button.closest('.sn-project-api-box'), amount = Number(String($box.find('.sn-project-api-wallet-amount').val() || '').replace(/[^0-9.-]/g, '')) || 0;
    if (amount <= 0) { toast('مبلغ شارژ را وارد کنید.', false); return; }
    $button.prop('disabled', true).text('در حال شارژ...');
    request('sn_project_api_charge_wallet', {item_id: $box.data('item'), wallet_type: $box.find('.sn-project-api-wallet-type').val(), amount: amount, note: $box.find('.sn-project-api-note').val() || ''}).done(function (res) {
      if (!res || !res.success) { toast(messageOf(res, 'شارژ کیف پول انجام نشد.'), false); return; }
      toast(messageOf(res, 'کیف پول مشتری شارژ شد.'), true); $box.find('.sn-project-api-result').text('شارژ با موفقیت در سایت مقصد ثبت شد.'); loadThread(dashboardState.selectedMembership, dashboardState.selectedItem);
    }).fail(function (xhr) { toast(messageOf(xhr && xhr.responseJSON, 'شارژ کیف پول سایت مقصد ناموفق بود.'), false); }).always(function () { $button.prop('disabled', false).text('شارژ کیف پول'); });
  });

  $(document).on('change', '.sn-project-status-select', function () {
    var $select = $(this), status = String($select.val() || ''), previousStatus = String($select.attr('data-current-status') || 'waiting'), cancelReason = '';
    if (status === 'cancelled') {
      cancelReason = window.prompt('علت انصراف را وارد کنید (اجباری):', '') || '';
      if (!cancelReason.trim()) { $select.val(previousStatus); toast('ثبت علت انصراف اجباری است.', false); return; }
    }
    $select.attr('data-current-status', status).prop('disabled', true);
    request('sn_project_update_status', {item_id: $select.data('item'), status: status, cancel_reason: cancelReason}).done(function (res) {
      toast(messageOf(res), !!(res && res.success));
      if (res && res.success) { loadThread(dashboardState.selectedMembership, dashboardState.selectedItem); loadDashboard(false); }
      else { loadThread(dashboardState.selectedMembership, dashboardState.selectedItem); }
    }).always(function () {$select.prop('disabled', false);});
  });
  $(document).on('change', '.sn-project-action-target', function () {
	var selected = $(this).find('option:selected'), credit = selected.data('credit'), price = selected.data('price'), form = $(this).closest('.sn-project-action-form');
	form.find('.sn-project-action-credit').val(credit || '');
	if (price !== undefined && Number(price) > 0) form.find('.sn-project-action-amount').val(price);
  });
  $(document).on('click', '.sn-project-create-action', function () {
    var $button = $(this), $box = $button.closest('.sn-project-action-box'), target = $box.find('.sn-project-action-target option:selected');
    request('sn_project_create_action', {item_id:$button.data('item'), action_type:$button.data('action-type'), fulfillment_mode:$box.find('.sn-project-action-mode').val(), target_product_id:target.val() || 0, target_product_name:target.text().split(' · ')[0], target_credit:$box.find('.sn-project-action-credit').val(), amount:$box.find('.sn-project-action-amount').val()}).done(function(res){ toast(messageOf(res), !!(res && res.success)); if(res && res.success) loadThread(dashboardState.selectedMembership, dashboardState.selectedItem); }).always(function(){ $button.prop('disabled',false); });
  });
  $(document).on('click', '.sn-project-retry-action', function () {
    var $button=$(this).prop('disabled',true); request('sn_project_retry_action_payment',{action_id:$button.data('action')}).done(function(res){toast(messageOf(res),!!(res&&res.success)); if(res&&res.success) loadThread(dashboardState.selectedMembership,dashboardState.selectedItem);}).always(function(){$button.prop('disabled',false);});
  });
  $(document).on('click', '.sn-project-cancel-upsell', function () {
    var reason=window.prompt('علت لغو افزایشی را وارد کنید (اختیاری):')||'', $button=$(this).prop('disabled',true); request('sn_project_cancel_upsell',{action_id:$button.data('action'),reason:reason}).done(function(res){toast(messageOf(res),!!(res&&res.success)); if(res&&res.success) loadThread(dashboardState.selectedMembership,dashboardState.selectedItem);}).always(function(){$button.prop('disabled',false);});
  });
  $(document).on('click', '.sn-project-fulfill-action', function () {
    var $button=$(this).prop('disabled',true); request('sn_project_fulfill_action',{action_id:$button.data('action')}).done(function(res){toast(messageOf(res),!!(res&&res.success)); if(res&&res.success) loadThread(dashboardState.selectedMembership,dashboardState.selectedItem);}).always(function(){$button.prop('disabled',false);});
  });
  $(document).on('change', '.sn-project-expert-select', function () {
    var $select = $(this).prop('disabled', true);
    request('sn_project_assign_expert', {item_id: $select.data('item'), expert_id: $select.val()}).done(function (res) {
      toast(messageOf(res), !!(res && res.success));
      if (res && res.success) { loadThread(dashboardState.selectedMembership, dashboardState.selectedItem); loadDashboard(false); }
    }).always(function () {$select.prop('disabled', false);});
  });
  $(document).on('click', '.sn-project-add-note', function () {
    var note = window.prompt('یادداشت داخلی را وارد کنید:'); if (!note) return;
    request('sn_project_add_note', {item_id: $(this).data('item'), note: note}).done(function (res) { toast(messageOf(res), !!(res && res.success)); if (res && res.success) loadThread(dashboardState.selectedMembership, dashboardState.selectedItem); });
  });
  $(document).on('click', '.sn-project-link-invoice', function () {
    var invoice = window.prompt('کد یا شناسه فاکتور تاییدشده همین مشتری را وارد کنید. فاکتور باید بعد از فعال‌شدن اشتراک باشد:'); if (!invoice) return;
    request('sn_project_link_invoice', {item_id: $(this).data('item'), invoice: invoice}).done(function (res) { toast(messageOf(res), !!(res && res.success)); if (res && res.success) { loadThread(dashboardState.selectedMembership, dashboardState.selectedItem); loadDashboard(false); } });
  });

  function renderStats(stats) {
    stats = stats || {};
    return '<div><span>کل محتواها</span><strong>' + fa(stats.total) + '</strong></div><div><span>در انتظار</span><strong>' + fa(stats.waiting) + '</strong></div><div><span>جواب نداده</span><strong>' + fa(stats.no_answer) + '</strong></div><div><span>تماس مجدد</span><strong>' + fa(stats.follow_up) + '</strong></div><div><span>انصراف</span><strong>' + fa(stats.cancelled) + '</strong></div><div><span>عادی</span><strong>' + fa(stats.normal) + '</strong></div><div><span>افزایشی</span><strong>' + fa(stats.upsell) + '</strong></div><div><span>مجموع خرید تاییدشده</span><strong>' + fa(stats.verified_total) + ' تومان</strong></div>';
  }
  function canBulkAssign() { return dashboardState.role === 'project_manager' || dashboardState.role === 'admin'; }
  function selectedIds() { return Object.keys(dashboardState.selectedItems).filter(function (id) { return !!dashboardState.selectedItems[id]; }).map(Number); }
  function expertOptions() {
    return '<option value="0">— برداشتن تخصیص —</option>' + dashboardState.experts.map(function (row) { return '<option value="' + esc(row.id) + '">' + esc(row.name) + '</option>'; }).join('');
  }
  function bulkToolbar() {
    if (!canBulkAssign()) return '';
    return '<div class="sn-project-bulk-toolbar"><div><strong>تخصیص گروهی</strong><span><b class="sn-project-selected-count">۰</b> مورد انتخاب شده</span></div><label><span>کارشناس پروژه</span><select class="sn-project-bulk-expert">' + expertOptions() + '</select></label><button type="button" class="sn-btn sn-btn-primary sn-project-bulk-assign" disabled>تخصیص موارد انتخابی</button></div>';
  }
  function syncBulkControls() {
    var count = selectedIds().length;
    $('.sn-project-selected-count').text(fa(count));
    $('.sn-project-bulk-assign').prop('disabled', count < 1);
    var selectable = dashboardState.items.filter(function (item) { return !!item.can_manage; });
    var checked = selectable.filter(function (item) { return !!dashboardState.selectedItems[String(item.id)]; }).length;
    $('.sn-project-select-all').prop('checked', selectable.length > 0 && checked === selectable.length).prop('indeterminate', checked > 0 && checked < selectable.length);
  }
  function renderDashboardItems() {
    var rows = dashboardState.items;
    if (!rows.length) { $('.sn-project-list').html('<div class="sn-project-empty">پرونده‌ای با این فیلتر پیدا نشد.</div>'); return; }
    var selectHead = canBulkAssign() ? '<th class="sn-project-check-cell"><input type="checkbox" class="sn-project-select-all" aria-label="انتخاب همه موارد بارگذاری‌شده"></th>' : '';
    var html = bulkToolbar() + '<div class="sn-project-table-wrap"><table class="sn-project-table"><thead><tr>' + selectHead + '<th>مشتری</th><th>اشتراک / محتوا</th><th>مسئول</th><th>وضعیت</th><th>خرید تاییدشده</th><th>عملیات</th></tr></thead><tbody>';
    rows.forEach(function (item) {
      var verified = item.purchase_verified ? '<span class="sn-project-verified-tag">' + esc(item.purchase_total_fmt) + '</span><small>' + fa(item.purchase_invoice_count) + ' فاکتور</small>' : '<span class="sn-project-pending-tag">ثبت نشده</span>';
      var selected = dashboardState.selectedItems[String(item.id)] ? ' checked' : '';
      var selectCell = canBulkAssign() ? '<td class="sn-project-check-cell">' + (item.can_manage ? '<input type="checkbox" class="sn-project-row-select" value="' + esc(item.id) + '" aria-label="انتخاب ' + esc(item.customer_name) + '"' + selected + '>' : '') + '</td>' : '';
      html += '<tr data-project-item="' + esc(item.id) + '">' + selectCell + '<td><strong>' + esc(item.customer_name) + '</strong><a href="tel:' + esc(item.customer_phone) + '">' + esc(item.customer_phone) + '</a></td><td><strong>' + esc(item.subscription_name) + '</strong><small>' + esc(item.content_name) + '</small></td><td><span>فروش: ' + esc(item.origin_name) + '</span><small>کارشناس: ' + esc(item.expert_name) + '</small></td><td><span class="sn-project-workflow-tag is-' + esc(item.workflow_status) + '">' + esc(item.workflow_status_label) + '</span></td><td>' + verified + '</td><td><button type="button" class="sn-btn sn-btn-sm sn-btn-primary sn-project-chat-btn" data-membership="' + esc(item.membership_id) + '" data-item="' + esc(item.id) + '">بررسی و چت' + unreadBadge(item.membership_id, item.id) + '</button></td></tr>';
    });
    html += '</tbody></table></div>';
    if (dashboardState.hasMore) html += '<div class="sn-project-load-more-wrap"><button type="button" class="sn-btn sn-btn-secondary sn-project-load-more">بارگذاری موارد بیشتر (' + fa(rows.length) + ' از ' + fa(dashboardState.total) + ')</button></div>';
    $('.sn-project-list').html(html);
    syncBulkControls();
  }
  function loadDashboard(append) {
    var $panel = $('[data-sn-project-panel]'); if (!$panel.length) return;
    append = !!append;
    if (dashboardState.loading && append) return;
    var page = append ? dashboardState.page + 1 : 1;
    var serial = ++dashboardRequestSerial;
    dashboardState.loading = true;
    if (!append) { dashboardState.items = []; dashboardState.selectedItems = {}; dashboardState.page = 0; $('.sn-project-list').html('<div class="sn-project-loading">در حال دریافت پرونده‌ها...</div>'); }
    else $('.sn-project-load-more').prop('disabled', true).text('در حال بارگذاری...');
    request('sn_project_dashboard', {page: page, query: String($('.sn-project-search').val() || '').trim(), status: String($('.sn-project-status-filter').val() || '')}).done(function (res) {
      if (serial !== dashboardRequestSerial) return;
      if (!res || !res.success) { $('.sn-project-list').html('<div class="sn-notice sn-error">' + esc(messageOf(res)) + '</div>'); return; }
      var pagination = res.pagination || {};
      dashboardState.items = append ? dashboardState.items.concat(res.items || []) : (res.items || []);
      dashboardState.experts = res.experts || []; dashboardState.role = res.role || dashboardState.role;
      dashboardState.page = Number(pagination.page || page); dashboardState.total = Number(pagination.total || dashboardState.items.length); dashboardState.hasMore = !!pagination.has_more;
      $('.sn-project-stats').html(renderStats(res.stats)); renderDashboardItems();
    }).fail(function (xhr) { if (serial === dashboardRequestSerial) $('.sn-project-list').html('<div class="sn-notice sn-error">' + esc(messageOf(xhr && xhr.responseJSON, 'خطای ارتباط با سرور.')) + '</div>'); }).always(function () { if (serial === dashboardRequestSerial) dashboardState.loading = false; });
  }
  $(document).on('click', '.sn-project-refresh', function () { loadDashboard(false); loadNotifications(true); });
  $(document).on('click', '.sn-project-load-more', function () { loadDashboard(true); });
  $(document).on('input', '.sn-project-search', function () { window.clearTimeout(dashboardSearchTimer); dashboardSearchTimer = window.setTimeout(function () { loadDashboard(false); }, 350); });
  $(document).on('change', '.sn-project-status-filter', function () { loadDashboard(false); });
  $(document).on('change', '.sn-project-row-select', function () {
    dashboardState.selectedItems[String($(this).val())] = this.checked;
    $(this).closest('tr').toggleClass('is-selected', this.checked);
    syncBulkControls();
  });
  $(document).on('change', '.sn-project-select-all', function () {
    var checked = this.checked;
    dashboardState.items.forEach(function (item) { if (item.can_manage) dashboardState.selectedItems[String(item.id)] = checked; });
    $('.sn-project-row-select').prop('checked', checked).closest('tr').toggleClass('is-selected', checked);
    syncBulkControls();
  });
  $(document).on('click', '.sn-project-bulk-assign', function () {
    var ids = selectedIds(); if (!ids.length) return;
    var $button = $(this), $select = $('.sn-project-bulk-expert'), expertId = Number($select.val() || 0);
    var expertName = expertId > 0 ? String($select.find('option:selected').text() || '') : 'بدون کارشناس';
    if (!window.confirm('تخصیص ' + fa(ids.length) + ' مورد به «' + expertName + '» انجام شود؟')) return;
    $button.prop('disabled', true).text('در حال تخصیص...');
    request('sn_project_bulk_assign_expert', {item_ids: ids, expert_id: expertId}).done(function (res) {
      toast(messageOf(res), !!(res && res.success));
      if (res && res.success) { dashboardState.selectedItems = {}; loadDashboard(false); }
    }).fail(function (xhr) { toast(messageOf(xhr && xhr.responseJSON, 'تخصیص گروهی انجام نشد.'), false); }).always(function () { $button.prop('disabled', false).text('تخصیص موارد انتخابی'); });
  });

  function notificationTarget() {
    if ($('[data-sn-project-panel]').length) return $('[data-sn-project-panel] .sn-project-panel-head').first();
    if ($('#sn-seller-panel').length) return $('#sn-seller-panel .sn-panel-header').first();
    if ($('#sn-dot-converter-panel').length) return $('#sn-dot-converter-panel .sn-panel-header').first();
    if ($('#sn-supervisor-panel, #sn-senior-supervisor-panel, #sn-sales-manager-panel').length) return $('#sn-supervisor-panel .sn-panel-header, #sn-senior-supervisor-panel .sn-panel-header, #sn-sales-manager-panel .sn-panel-header').first();
    if ($('.sn-ops-panel .sn-ops-hero-actions').length) return $('.sn-ops-panel .sn-ops-hero-actions').first();
    return $();
  }
  function ensureNotificationCenter() {
    if (['admin','project_manager','project_expert','seller','converter','sales_supervisor','operations_sales_expert','operations_sales_supervisor'].indexOf(String(cfg.role || '')) < 0) return;
    if ($('.sn-project-notification-center').length) return;
    var $target = notificationTarget(); if (!$target.length) return;
    var opsSide = isOperationsChatRole();
    var notificationLabel = opsSide ? 'فروش' : 'بیاوین';
    var html = '<div class="sn-project-notification-center"><button type="button" class="sn-project-notification-toggle" aria-expanded="false" aria-label="اعلان‌های گفت‌وگو با ' + esc(notificationLabel) + '"><span aria-hidden="true">🔔</span><b class="sn-project-notification-badge" hidden>۰</b></button><div class="sn-project-notification-popover" hidden><header><div><strong>پیام‌های جدید ' + esc(notificationLabel) + '</strong><small>گفت‌وگوهای خوانده‌نشده</small></div><button type="button" class="sn-project-notification-refresh" aria-label="بروزرسانی">↻</button></header><div class="sn-project-notification-list"><div class="sn-project-notification-empty">در حال دریافت اعلان‌ها...</div></div></div></div>';
    $target.append(html);
  }
  function renderInlineUnreadBadges() {
    $('.sn-project-chat-btn').each(function () {
      var $button = $(this), count = unreadFor($button.data('membership'), $button.data('item') || 0);
      $button.find('.sn-project-inline-unread').remove();
      if (count > 0) $button.append('<span class="sn-project-inline-unread" aria-label="' + esc(fa(count) + ' پیام خوانده‌نشده') + '">' + fa(count) + '</span>');
    });
  }
  function renderNotifications() {
    ensureNotificationCenter();
    var total = Number(notificationState.total || 0), $badge = $('.sn-project-notification-badge');
    $badge.text(fa(total)).prop('hidden', total < 1);
    var rows = notificationState.threads || [], html = '';
    if (!rows.length) html = '<div class="sn-project-notification-empty"><span>✓</span><strong>پیام خوانده‌نشده‌ای ندارید</strong></div>';
    else rows.forEach(function (row) {
      html += '<button type="button" class="sn-project-notification-item" data-membership="' + esc(row.membership_id) + '" data-item="' + esc(row.item_id) + '"><span class="sn-project-notification-avatar">' + esc(String(row.last_sender_name || '؟').trim().charAt(0) || '؟') + '</span><span class="sn-project-notification-copy"><strong>' + esc(row.customer_name || 'مشتری') + ' · ' + esc(row.content_name || '') + '</strong><small>' + esc(row.last_sender_name || '') + ': ' + esc(row.preview || 'فایل جدید') + '</small><time>' + esc(jalaliDateTime(row.last_message_at || '')) + '</time></span><b>' + fa(row.unread_count || 0) + '</b></button>';
    });
    $('.sn-project-notification-list').html(html);
    renderInlineUnreadBadges();
  }
  function loadNotifications(silent) {
    if (notificationState.loading) { if (silent) notificationState.refreshQueued = true; return; }
    if (document.hidden && !silent) return;
    notificationState.loading = true;
    request('sn_project_notifications').done(function (res) {
      if (!res || !res.success) return;
      var nextTotal = Number(res.total || 0);
      if (!silent && notificationState.total !== null && nextTotal > notificationState.total) {
        toast(fa(nextTotal - notificationState.total) + ' پیام جدید در گفت‌وگو با ' + (isOperationsChatRole() ? 'فروش' : 'بیاوین') + ' دارید.', true);
      }
      notificationState.total = nextTotal; notificationState.threads = res.threads || [];
      renderNotifications();
    }).always(function () {
      notificationState.loading = false;
      if (notificationState.refreshQueued) { notificationState.refreshQueued = false; loadNotifications(true); }
    });
  }
  function positionNotificationPopover($toggle, $popover) {
    if (!$toggle.length || !$popover.length || !$toggle[0]) return;
    var rect = $toggle[0].getBoundingClientRect();
    var viewportWidth = Math.max(document.documentElement.clientWidth || 0, window.innerWidth || 0);
    var viewportHeight = Math.max(document.documentElement.clientHeight || 0, window.innerHeight || 0);
    var width = Math.min(390, Math.max(240, viewportWidth - 28));
    var left = Math.max(14, Math.min(viewportWidth - width - 14, rect.left));
    $popover.css({width: width + 'px', left: Math.round(left) + 'px', right: 'auto', top: Math.round(rect.bottom + 10) + 'px'});
    var popHeight = $popover.outerHeight() || 0;
    if (popHeight && rect.bottom + 10 + popHeight > viewportHeight - 14) {
      var above = rect.top - popHeight - 10;
      if (above >= 14) $popover.css('top', Math.round(above) + 'px');
    }
  }
  function closeNotificationPopover() {
    $('.sn-project-notification-popover').prop('hidden', true);
    $('.sn-project-notification-toggle').attr('aria-expanded', 'false');
  }
  $(document).on('click', '.sn-project-notification-toggle', function (event) {
    event.stopPropagation();
    var $toggle = $(this), $popover = $('.sn-project-notification-popover').first(), open = $popover.prop('hidden');
    if (open) {
      if (!$popover.parent().is('body')) $popover.appendTo(document.body);
      $popover.prop('hidden', false);
      positionNotificationPopover($toggle, $popover);
      $toggle.attr('aria-expanded', 'true');
      loadNotifications(true);
    } else {
      closeNotificationPopover();
    }
  });
  $(document).on('click', '.sn-project-notification-popover', function (event) { event.stopPropagation(); });
  $(document).on('click', function () { closeNotificationPopover(); });
  $(document).on('click', '.sn-project-notification-refresh', function () { loadNotifications(true); });
  $(document).on('click', '.sn-project-notification-item', function () {
    $('.sn-project-notification-popover').prop('hidden', true);
    loadThread($(this).data('membership'), $(this).data('item'));
  });
  $(window).on('resize scroll', function () { var $popover = $('.sn-project-notification-popover').first(); if ($popover.length && !$popover.prop('hidden')) positionNotificationPopover($('.sn-project-notification-toggle').first(), $popover); });
  $(document).on('visibilitychange', function () { if (!document.hidden) loadNotifications(false); });
  $(document).ajaxComplete(function () { window.setTimeout(renderInlineUnreadBadges, 20); });

  function walletMarkup(wallet) {
    wallet = wallet || {}; var tx = wallet.transactions || [];
    var recent = tx.length ? '<details open><summary>آخرین تراکنش‌های بیاوین</summary>' + tx.slice(0, 10).map(function (row) { return '<div class="sn-project-wallet-tx"><span>' + esc(row.description || row.type) + '</span><strong class="' + (row.direction === 'debit' ? 'is-debit' : 'is-credit') + '">' + (row.direction === 'debit' ? '− ' : '+ ') + esc(row.amount_fmt) + '</strong><small>' + esc(jalaliDateTime(row.created_at)) + '</small></div>'; }).join('') + '</details>' : '<div class="sn-project-wallet-empty">هنوز تراکنش بیاوین ثبت نشده است.</div>';
    return '<div class="sn-project-wallet-balance"><span>موجودی بیاوین</span><strong>' + esc(wallet.balance_fmt || '۰ تومان') + '</strong></div>' + recent;
  }
  function loadWallet(force) {
    var $bodies = $('.sn-project-wallet-body'); if (!$bodies.length) return;
    if (!force && $bodies.filter('[data-loaded="1"]').length === $bodies.length) return;
    $bodies.attr('data-loaded', 'loading').html('<div class="sn-project-wallet-loading">در حال دریافت موجودی...</div>');
    request('sn_project_wallet').done(function (res) {
      if (!res || !res.success) { $bodies.attr('data-loaded', '0').html('<span class="sn-error">' + esc(messageOf(res)) + '</span>'); return; }
      $bodies.attr('data-loaded', '1').html(walletMarkup(res.wallet));
    }).fail(function (xhr) { $bodies.attr('data-loaded', '0').html('<span class="sn-error">' + esc(messageOf(xhr && xhr.responseJSON, 'کیف پول دریافت نشد.')) + '</span>'); });
  }
  $(document).on('click', '[data-sn-wallet-view]', function () {
    var view = String($(this).data('sn-wallet-view') || 'sales'), $root = $(this).closest('#sn-tab-wallet');
    $root.find('[data-sn-wallet-view]').removeClass('is-active').addClass('sn-btn-secondary').attr('aria-selected', 'false');
    $(this).addClass('is-active').removeClass('sn-btn-secondary').attr('aria-selected', 'true');
    $root.find('[data-sn-wallet-panel]').prop('hidden', true).filter('[data-sn-wallet-panel="' + view + '"]').prop('hidden', false);
    if (view === 'biavin') loadWallet(false);
  });
  $(document).on('click', '.sn-project-wallet-refresh', function () { loadWallet(true); });
  $(document).on('click', '#sn-dot-converter-panel .sn-tab[data-tab="wallet"]', function () { window.setTimeout(function () { loadWallet(false); }, 50); });

  $(function () {
    ensureModal(); ensureNotificationCenter(); loadDashboard(false); loadNotifications(true);
    notificationState.timer = window.setInterval(function () { loadNotifications(false); }, 30000);
  });
}(jQuery));
