(function ($) {
  'use strict';

  var cfg = window.snAjax || window.snData || {};
  var ajaxUrl = cfg.ajaxurl || '';
  var nonce = cfg.nonce || '';
  if (!$('#sn-dot-converter-panel').length) return;
  function panelRoot() { return $('#sn-dot-converter-panel'); }

  function esc(v) {
    return String(v == null ? '' : v).replace(/[&<>"']/g, function (c) {
      return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#039;'}[c];
    });
  }
  function fa(v) {
    try { return Number(v || 0).toLocaleString('fa-IR'); } catch (e) { return String(v || 0); }
  }
  function money(v) { return fa(v) + ' تومان'; }
  function digits(v) {
    return String(v || '')
      .replace(/[۰-۹]/g, function (d) { return String('۰۱۲۳۴۵۶۷۸۹'.indexOf(d)); })
      .replace(/[٠-٩]/g, function (d) { return String('٠١٢٣٤٥٦٧٨٩'.indexOf(d)); });
  }
  function numeric(v) { return Number(digits(v).replace(/[^0-9.]/g, '')) || 0; }
  function notice(message, success) {
    $('#sn-converter-invoice-notice').html('<div class="sn-notice ' + (success ? 'sn-success' : 'sn-error') + '">' + esc(message) + '</div>');
  }
  function invoiceNotice(message, success) {
    $('#sn-converter-invoices-notice').html('<div class="sn-notice ' + (success ? 'sn-success' : 'sn-error') + '">' + esc(message) + '</div>');
  }
  function activateTab(tab) {
    var $panel = panelRoot();
    var $button = $panel.find('.sn-tab[data-tab="' + tab + '"]');
    if ($button.length) { $button.trigger('click'); return; }
    $panel.find('.sn-tab').removeClass('active').attr('aria-selected', 'false');
    $panel.find('.sn-tab-content').removeClass('active').hide();
    $button.addClass('active').attr('aria-selected', 'true');
    $('#sn-tab-' + tab).addClass('active').show();
  }

  // Manual invoice form ---------------------------------------------------
  function maxProducts() {
    var n = Number(panelRoot().attr('data-sn-max-products') || $('#sn-converter-add-product').attr('data-max-products') || 0);
    return isFinite(n) && n > 0 ? Math.floor(n) : 0;
  }
  function productOptions() {
    return $('#sn-converter-products-multi .sn-converter-product-select').first().html() || '<option value="">انتخاب محصول</option>';
  }
  function invoiceTotal() {
    var total = 0;
    $('#sn-converter-products-multi .sn-product-row').each(function () {
      var $row = $(this);
      var price = Number($row.find('.sn-converter-product-select option:selected').data('price') || 0);
      var qty = Math.max(1, Number($row.find('.sn-converter-product-qty').val() || 1));
      total += price * qty;
    });
    return total;
  }
  function syncProducts() {
    var count = $('#sn-converter-products-multi .sn-product-row').length;
    var limit = maxProducts();
    $('#sn-converter-products-total').text('جمع: ' + money(invoiceTotal()));
    $('.sn-converter-remove-product').toggle(count > 1);
    $('#sn-converter-add-product').prop('disabled', limit > 0 && count >= limit);
    syncShippingFields();
    syncPrepayment();
  }
  function hasPhysicalProduct() {
    var found = false;
    $('#sn-converter-products-multi .sn-product-row').each(function () {
      var type = String($(this).find('.sn-converter-product-select option:selected').attr('data-sn-product-type') || '');
      if (type === 'product') found = true;
    });
    return found;
  }
  function syncShippingFields() {
    var required = hasPhysicalProduct();
    $('.sn-converter-shipping-field').prop('hidden', !required);
    $('#sn-converter-cust-address').prop('required', required);
  }
  function syncPrepayment() {
    var assessment=$('#sn-converter-products-multi .sn-converter-product-select option:selected[data-sn-product-type="assessment"]').length>0;
    $('#sn-converter-payment-plan option[value="partial"]').prop('disabled',assessment);
    if(assessment && $('#sn-converter-payment-plan').val()==='partial') $('#sn-converter-payment-plan').val('full');
    var partial = $('#sn-converter-payment-plan').val() === 'partial';
    $('.sn-converter-prepayment-fields').prop('hidden', !partial);
    if (!partial) {
      $('#sn-converter-prepayment-choice').val('');
      $('#sn-converter-prepayment-custom').val('').prop('hidden', true);
      return;
    }
    $('#sn-converter-prepayment-custom').prop('hidden', $('#sn-converter-prepayment-choice').val() !== 'custom');
  }

  $(document).on('click.snConverterInvoice', '#sn-converter-add-product', function () {
    var count = $('#sn-converter-products-multi .sn-product-row').length;
    var limit = maxProducts();
    if (limit > 0 && count >= limit) {
      notice('حداکثر تعداد محصول مجاز برای هر پیش‌فاکتور انتخاب شده است.', false);
      return;
    }
    $('#sn-converter-products-multi').append(
      '<div class="sn-product-row">' +
      '<select class="sn-converter-product-select">' + productOptions() + '</select>' +
      '<input type="number" class="sn-converter-product-qty" min="1" value="1" aria-label="تعداد">' +
      '<button type="button" class="sn-btn sn-btn-ghost sn-converter-remove-product">حذف</button>' +
      '</div>'
    );
    syncProducts();
  });
  $(document).on('click.snConverterInvoice', '.sn-converter-remove-product', function () {
    $(this).closest('.sn-product-row').remove();
    syncProducts();
  });
  $(document).on('change.snConverterInvoice input.snConverterInvoice', '.sn-converter-product-select,.sn-converter-product-qty', syncProducts);
  $(document).on('change.snConverterInvoice', '#sn-converter-payment-plan,#sn-converter-prepayment-choice', syncPrepayment);
  $(document).on('input.snConverterInvoice', '#sn-converter-prepayment-custom', function () {
    var raw = digits(this.value).replace(/[^0-9]/g, '');
    this.value = raw ? raw.replace(/\B(?=(\d{3})+(?!\d))/g, ',') : '';
  });

  $(document).on('click.snConverterInvoice', '#sn-converter-create-invoice', function () {
    if (!ajaxUrl) return;
    var $button = $(this);
    var name = String($('#sn-converter-cust-name').val() || '').trim();
    var phone = digits($('#sn-converter-cust-phone').val()).replace(/\D+/g, '');
    var province = String($('#sn-converter-cust-prov').val() || '').trim();
    var city = String($('#sn-converter-cust-city').val() || '').trim();
    var address = String($('#sn-converter-cust-address').val() || '').trim();
    var postalCode = digits($('#sn-converter-cust-postal').val()).replace(/\D+/g, '');
    var productIds = [];
    var qtys = [];
    $('#sn-converter-products-multi .sn-product-row').each(function () {
      var id = Number($(this).find('.sn-converter-product-select').val() || 0);
      if (id > 0) {
        productIds.push(id);
        qtys.push(Math.max(1, Number($(this).find('.sn-converter-product-qty').val() || 1)));
      }
    });
    if (!name) { notice('نام مشتری را وارد کنید.', false); return; }
    if (!/^09\d{9}$/.test(phone)) { notice('شماره موبایل مشتری معتبر نیست.', false); return; }
    if (!productIds.length) { notice('حداقل یک محصول انتخاب کنید.', false); return; }
    if (hasPhysicalProduct() && (!province || !city || !address)) {
      notice('برای محصول عادی، استان، شهر و آدرس کامل مشتری الزامی است.', false);
      return;
    }
    if (postalCode && postalCode.length !== 10) {
      notice('کد پستی باید ۱۰ رقم باشد.', false);
      return;
    }

    var plan = String($('#sn-converter-payment-plan').val() || 'full');
    var prepayment = 0;
    if (plan === 'partial') {
      var choice = String($('#sn-converter-prepayment-choice').val() || '');
      prepayment = choice === 'custom' ? numeric($('#sn-converter-prepayment-custom').val()) : numeric(choice);
      var total = invoiceTotal();
      if (prepayment <= 0 || prepayment > total) {
        notice('مبلغ این مرحله باید بیشتر از صفر و حداکثر برابر مبلغ کل فاکتور باشد.', false);
        return;
      }
    }

    $button.prop('disabled', true).text('در حال صدور پیش‌فاکتور...');
    notice('در حال ثبت و ارسال پیامک...', true);
    $.ajax({
      url: ajaxUrl,
      method: 'POST',
      dataType: 'json',
      timeout: 45000,
      data: {
        action: 'sn_create_invoice',
        nonce: nonce,
        customer_name: name,
        customer_phone: phone,
        customer_phone_secondary: digits($('#sn-converter-cust-phone-secondary').val()).replace(/\D+/g, ''),
        province: province,
        city: city,
        customer_address: address,
        customer_postal_code: postalCode,
        product_id: productIds[0],
        product_ids: productIds,
        product_qtys: qtys,
        payment_plan: plan,
        prepayment_amount: prepayment
      }
    }).done(function (res) {
      var data = res && res.data ? res.data : {};
      if (!res || !res.success) {
        notice((data && data.message) || (res && res.message) || 'صدور پیش‌فاکتور انجام نشد.', false);
        return;
      }
      var code = data.invoice_code || data.code || '';
      var smsMessage = data.sms_sent === false ? ' پیش‌فاکتور صادر شد اما ارسال پیامک ناموفق بود.' : ' پیش‌فاکتور صادر شد و پیامک مشتری ارسال شد.';
      notice('پیش‌فاکتور ' + (code ? code : '') + smsMessage, true);
      $('#sn-converter-cust-name,#sn-converter-cust-phone,#sn-converter-cust-phone-secondary,#sn-converter-cust-city,#sn-converter-cust-address,#sn-converter-cust-postal').val('');
      $('#sn-converter-cust-prov').val('');
      $('#sn-converter-products-multi .sn-product-row').slice(1).remove();
      $('#sn-converter-products-multi .sn-converter-product-select').val('');
      $('#sn-converter-products-multi .sn-converter-product-qty').val('1');
      $('#sn-converter-payment-plan').val('full');
      syncProducts();
      window.setTimeout(function () { activateTab('invoices'); }, 900);
    }).fail(function (xhr) {
      var res = xhr && xhr.responseJSON ? xhr.responseJSON : null;
      var msg = res && res.data && res.data.message ? res.data.message : (res && res.message ? res.message : 'ارتباط با سرور برقرار نشد.');
      notice(msg, false);
    }).always(function () {
      $button.prop('disabled', false).text('صدور پیش‌فاکتور و ارسال پیامک');
    });
  });

  // Converter invoice workspace: manual invoices + Dot conversion invoices.
  var activeInvoiceStatus = 'all';
  var invoiceRequest = null;
	var converterPaymentInvoiceCode = '';
	var converterReceiptInvoiceId = '';
	function ensureConverterPaymentModal() {
	  if ($('#sn-converter-manual-payment-modal').length) return;
	  $('body').append('<div id="sn-converter-manual-payment-modal" class="sn-modal sn-lite-modal sn-payment-entry-modal" aria-hidden="true" style="display:none"><div class="sn-modal-backdrop sn-converter-payment-close"></div><div class="sn-modal-card sn-payment-entry-card" role="dialog" aria-modal="true">'+
		'<div class="sn-modal-head sn-payment-entry-head"><div class="sn-payment-entry-title"><span class="sn-payment-entry-icon">↙</span><div><h3>ثبت اطلاعات واریز</h3><p>اطلاعات پرداخت مشتری را دقیق وارد کنید</p></div></div><button type="button" class="sn-modal-x sn-converter-payment-close" aria-label="بستن">×</button></div>'+
		'<div class="sn-modal-body"><div class="sn-payment-entry-grid"><label><span>مبلغ این مرحله (تومان)</span><input id="sn-converter-payment-amount" inputmode="numeric" readonly></label><label><span>۴ رقم آخر کارت مبدا</span><input id="sn-converter-payment-card-from" inputmode="numeric" maxlength="4" autocomplete="off" placeholder="مثلاً ۱۲۳۴"></label><label><span>۴ رقم آخر کارت مقصد (اختیاری)</span><input id="sn-converter-payment-card-to" inputmode="numeric" maxlength="4" autocomplete="off" placeholder="اختیاری؛ مثلاً ۵۶۷۸"></label><label><span>تاریخ شمسی واریز</span><input id="sn-converter-payment-date" class="sn-jalali-date" placeholder="۱۴۰۵/۰۷/۰۱"><button type="button" class="sn-btn sn-btn-sm sn-payment-today" data-date="#sn-converter-payment-date" data-hour="#sn-converter-payment-hour" data-minute="#sn-converter-payment-minute">امروز تهران</button></label><label><span>ساعت تهران (۲۴ ساعته)</span><span class="sn-payment-time-select"><select id="sn-converter-payment-hour"></select> : <select id="sn-converter-payment-minute"></select></span></label><label><span>تصویر فیش (اختیاری)</span><input id="sn-converter-payment-receipt" type="file" accept="image/*,.pdf,application/pdf"></label></div><div id="sn-converter-payment-msg" class="sn-payment-entry-message"></div><div class="sn-modal-actions sn-payment-entry-actions"><button type="button" class="sn-btn sn-btn-secondary sn-converter-payment-close">انصراف</button><button type="button" class="sn-btn sn-btn-primary" id="sn-converter-payment-submit">ثبت واریز و فیش</button></div></div></div></div>');
	}
	function ensureConverterReceiptModal() {
	  if ($('#sn-converter-receipt-modal').length) return;
	  $('body').append('<div id="sn-converter-receipt-modal" class="sn-modal sn-lite-modal sn-payment-entry-modal" aria-hidden="true" style="display:none"><div class="sn-modal-backdrop sn-converter-receipt-close"></div><div class="sn-modal-card sn-payment-entry-card sn-payment-proof-card" role="dialog" aria-modal="true">'+
		'<div class="sn-modal-head sn-payment-entry-head"><div class="sn-payment-entry-title"><span class="sn-payment-entry-icon">⌁</span><div><h3>بارگذاری فیش پرداخت</h3><p>فیش را انتخاب کنید؛ ۴ رقم کارت مقصد اختیاری است</p></div></div><button type="button" class="sn-modal-x sn-converter-receipt-close" aria-label="بستن">×</button></div>'+
		'<div class="sn-modal-body"><div class="sn-payment-entry-grid sn-payment-proof-grid"><label><span>۴ رقم آخر کارت مقصد (اختیاری)</span><input id="sn-converter-receipt-card-to" inputmode="numeric" maxlength="4" autocomplete="off" placeholder="اختیاری؛ مثلاً ۵۶۷۸"></label><label class="sn-payment-file-field"><span>فایل فیش</span><input id="sn-converter-receipt-file" type="file" accept="image/*,.pdf"><small>تصویر یا PDF</small></label></div><div id="sn-converter-receipt-msg" class="sn-payment-entry-message" aria-live="polite"></div><div class="sn-modal-actions sn-payment-entry-actions"><button type="button" class="sn-btn sn-btn-secondary sn-converter-receipt-close">انصراف</button><button type="button" class="sn-btn sn-btn-primary sn-converter-receipt-upload">ارسال فیش</button></div></div></div></div>');
	}
  function statusLabel(row) { return row.status_label || row.review_status_label || row.status || '—'; }
  function syncInvoiceSummary(summary) {
    summary = summary || {};
    $('.sn-converter-invoice-status-tabs .sn-subtab').each(function () {
      var key = String($(this).data('status') || 'all');
      $(this).find('.sn-invoice-tab-count').text(fa(summary[key] || 0));
    });
  }
  function loadInvoices() {
    if (!ajaxUrl) {
      $('#sn-converter-invoices-loading').prop('hidden', true);
      $('#sn-converter-invoices-list').html('<div class="sn-notice sn-error">آدرس ارتباط با سرور در این صفحه در دسترس نیست.</div>');
      return;
    }
    if (invoiceRequest && invoiceRequest.readyState !== 4) invoiceRequest.abort();
    $('#sn-converter-invoices-loading').prop('hidden', false).text('در حال بارگذاری فاکتورها...');
    $('#sn-converter-invoices-list').html('');
    invoiceRequest = $.ajax({
      url: ajaxUrl,
      method: 'POST',
      dataType: 'json',
      timeout: 20000,
      data: {
        action: 'sn_dot_converter_invoices',
        nonce: nonce,
        tab: activeInvoiceStatus,
        q: $('#sn-converter-invoice-search').val() || '',
        page: 1,
        limit: 60
      }
    }).done(function (res) {
      var rows = res && (res.invoices || res.items) ? (res.invoices || res.items) : [];
      if (!res || !res.success) {
        $('#sn-converter-invoices-list').html('<div class="sn-notice sn-error">' + esc((res && res.message) || 'بارگذاری فاکتورها انجام نشد.') + '</div>');
        return;
      }
      syncInvoiceSummary(res.summary || {});
      if (!rows.length) {
        $('#sn-converter-invoices-list').html('<div class="sn-card sn-empty-state">فاکتوری در این بخش وجود ندارد.</div>');
        return;
      }
      var html = '<div class="sn-table-wrap sn-converter-invoice-table-wrap"><table class="sn-table sn-converter-invoices-table"><thead><tr><th>فاکتور</th><th>نوع</th><th>مشتری</th><th>موضوع</th><th>مبلغ کل / پرداخت‌شده / مانده</th><th>وضعیت</th><th>تاریخ</th><th>اقدام پرداخت</th></tr></thead><tbody>';
      rows.forEach(function (row) {
        var sourceClass = row.source_type === 'conversion' ? 'is-conversion' : 'is-manual';
        var caseMeta = row.case_id ? '<small class="sn-converter-case-ref">پرونده #' + esc(row.case_id) + '</small>' : '';
        var reject = (row.status === 'rejected' && row.rejected_reason) ? '<small class="sn-reject-reason">' + esc(row.rejected_reason) + '</small>' : '';
        var stageAction = '<span class="sn-muted">—</span>';
        if (row.source_type === 'manual' && row.can_issue_next_stage) {
          stageAction = '<div class="sn-converter-next-stage" data-invoice-id="' + esc(row.id) + '" data-remaining="' + esc(row.remaining_amount || 0) + '">' +
            '<small>مانده: <strong>' + esc(money(row.remaining_amount || 0)) + '</strong> · مرحله بعد: ' + esc(Number(row.current_payment_stage || 1) + 1) + '</small>' +
            '<div class="sn-converter-next-stage-controls"><input type="text" class="sn-converter-next-stage-amount" inputmode="numeric" placeholder="مبلغ مرحله جدید">' +
            '<button type="button" class="sn-btn sn-btn-sm sn-btn-secondary sn-converter-stage-partial">صدور مرحله</button>' +
            '<button type="button" class="sn-btn sn-btn-sm sn-btn-primary sn-converter-stage-remaining">تسویه کل مانده</button></div></div>';
        } else if (row.source_type === 'manual' && Number(row.current_due_amount || 0) > 0) {
          stageAction = '<span class="sn-status-pill sn-status-warning">مرحله ' + esc(row.current_payment_stage || 1) + ' در انتظار پرداخت</span>';
        } else if (row.source_type === 'conversion' && row.case_id) {
          stageAction = '<small class="sn-muted">از داخل پرونده #' + esc(row.case_id) + '</small>';
        }
		if (row.can_submit_manual_payment) {
		  stageAction += '<div class="sn-converter-payment-actions"><button type="button" class="sn-btn sn-btn-sm sn-btn-secondary sn-converter-payment-open" data-invoice-code="'+esc(row.invoice_code||'')+'" data-amount="'+esc(row.current_due_amount||row.amount||'')+'">ثبت واریز و فیش</button></div>';
		}
		if (row.receipt_url) {
		  stageAction += '<a class="sn-btn sn-btn-sm sn-btn-secondary" href="'+esc(row.receipt_url)+'" target="_blank" rel="noopener">مشاهده فیش</a>';
		}
		if (row.can_upload_receipt) {
		  stageAction += '<button type="button" class="sn-btn sn-btn-sm sn-btn-primary sn-payment-proof-open sn-converter-receipt-open" data-invoice-id="'+esc(row.id||'')+'" data-has-receipt="'+(row.receipt_url?'1':'0')+'">'+(row.receipt_url?'جایگزینی فیش':'بارگذاری فیش')+'</button>';
		}
		stageAction += '<div class="sn-invoice-sms-action"><button type="button" class="sn-btn sn-btn-sm sn-btn-secondary sn-resend-invoice-sms" data-invoice-id="'+esc(row.id||'')+'">ارسال مجدد لینک پرداخت</button><small class="sn-resend-invoice-sms-msg" aria-live="polite"></small><button type="button" class="sn-btn sn-btn-sm sn-btn-secondary sn-copy-invoice-link" data-invoice-id="'+esc(row.id||'')+'">کپی لینک پرداخت</button><small class="sn-copy-invoice-link-msg" aria-live="polite"></small></div>';
		if(row.source_type==='manual' && Number(row.paid_total_amount||0)===0 && Number(row.current_payment_stage||1)===1) stageAction += '<button type="button" class="sn-btn sn-btn-sm sn-btn-secondary sn-edit-invoice" data-invoice-id="'+esc(row.id||'')+'">ویرایش پیش از پرداخت</button>';
		stageAction += '<button type="button" class="sn-btn sn-btn-sm sn-btn-secondary sn-invoice-finance-history" data-invoice-id="'+esc(row.id||'')+'">وضعیت و تاریخچه مالی</button>';
		if (row.project && row.project.membership_id) {
		  stageAction += '<div class="sn-project-invoice-action"><span class="sn-project-tag">' + esc(row.project.tag || 'اشتراک · بیاوین') + '</span><button type="button" class="sn-btn sn-btn-sm sn-btn-primary sn-project-chat-btn" data-membership="' + esc(row.project.membership_id) + '" data-item="' + esc((row.project.items&&row.project.items[0]&&row.project.items[0].id)||'') + '">چت با بیاوین</button></div>';
		}
        var amountCell='<div class="sn-stage-amounts"><span>کل: '+esc(money(row.payment_total_amount||0))+'</span><span>پرداخت‌شده: '+esc(money(row.paid_total_amount||0))+'</span><strong>مانده: '+esc(money(row.remaining_amount||0))+'</strong><small>مبلغ این مرحله: '+esc(money(row.amount||0))+'</small></div>';
        html += '<tr>' +
          '<td><code>' + esc(row.invoice_code || '') + '</code>' + caseMeta + '</td>' +
          '<td><span class="sn-converter-invoice-source ' + sourceClass + '">' + esc(row.source_label || 'صدور دستی') + '</span></td>' +
          '<td><strong>' + esc(row.customer_name || '—') + '</strong><small dir="ltr">' + esc(row.customer_phone || '—') + '</small></td>' +
          '<td>' + esc(row.product_name || '—') + '</td>' +
          '<td>' + amountCell + '</td>' +
          '<td><span class="sn-status sn-status-' + esc(row.status || '') + '">' + esc(statusLabel(row)) + '</span>' + reject + '</td>' +
          '<td><small>' + esc(row.created_at_label || '—') + '</small></td>' +
          '<td>' + stageAction + '</td>' +
          '</tr>';
      });
      html += '</tbody></table></div>';
      $('#sn-converter-invoices-list').html(html);
    }).fail(function (xhr, status) {
      if (status === 'abort') return;
      var res = xhr && xhr.responseJSON ? xhr.responseJSON : null;
      var msg = res && res.message ? res.message : (status === 'timeout' ? 'بارگذاری فاکتورها بیش از حد طول کشید. دوباره تلاش کنید.' : 'خطای ارتباط با سرور در بارگذاری فاکتورها.');
      $('#sn-converter-invoices-list').html('<div class="sn-notice sn-error">' + esc(msg) + '</div>');
    }).always(function () {
      $('#sn-converter-invoices-loading').prop('hidden', true);
      invoiceRequest = null;
    });
  }

  $(document).off('input.snConverterNextStage', '.sn-converter-next-stage-amount').on('input.snConverterNextStage', '.sn-converter-next-stage-amount', function () {
    var raw = digits(this.value).replace(/[^0-9]/g, '');
    this.value = raw ? raw.replace(/\B(?=(\d{3})+(?!\d))/g, ',') : '';
  });

	$(document).off('click.snConverterPaymentOpen','.sn-converter-payment-open').on('click.snConverterPaymentOpen','.sn-converter-payment-open',function(){
	  converterPaymentInvoiceCode=String($(this).data('invoice-code')||''); ensureConverterPaymentModal();
	  $('#sn-converter-payment-card-from,#sn-converter-payment-card-to,#sn-converter-payment-date,#sn-converter-payment-receipt').val(''); $('#sn-converter-payment-amount').val(String($(this).data('amount')||'')); $('#sn-converter-payment-msg').empty();
	  if(window.snPaymentTimeInit)window.snPaymentTimeInit('#sn-converter-payment-hour','#sn-converter-payment-minute');
	  $('#sn-converter-manual-payment-modal').fadeIn(120).attr('aria-hidden','false');
	});
	$(document).off('click.snConverterPaymentClose','.sn-converter-payment-close').on('click.snConverterPaymentClose','.sn-converter-payment-close',function(){ $('#sn-converter-manual-payment-modal').fadeOut(120).attr('aria-hidden','true'); });
	$(document).off('click.snConverterPaymentSubmit','#sn-converter-payment-submit').on('click.snConverterPaymentSubmit','#sn-converter-payment-submit',function(){
	  var from=digits($('#sn-converter-payment-card-from').val()).replace(/\D+/g,''), to=digits($('#sn-converter-payment-card-to').val()).replace(/\D+/g,''), date=digits($('#sn-converter-payment-date').val()).trim(), time=String($('#sn-converter-payment-hour').val()||'')+':'+String($('#sn-converter-payment-minute').val()||'');
	  var $btn=$(this), $msg=$('#sn-converter-payment-msg');
	  if(!/^\d{4}$/.test(from)||!/^\d{4}\/\d{2}\/\d{2}$/.test(date)||!/^\d{2}:\d{2}$/.test(time)){ $msg.html('<div class="sn-notice sn-error">۴ رقم مبدا، تاریخ و ساعت را کامل وارد کنید.</div>'); return; }
	  if(to!=='' && !/^\d{4}$/.test(to)){ $msg.html('<div class="sn-notice sn-error">۴ رقم کارت مقصد اختیاری است؛ در صورت ورود باید ۴ رقم باشد.</div>'); return; }
	  $btn.prop('disabled',true).text('در حال ثبت...');
	  var form=new FormData();form.append('action','sn_submit_manual_payment');form.append('nonce',nonce);form.append('invoice_code',converterPaymentInvoiceCode);form.append('card_from',from);form.append('card_to',to);form.append('paid_at',date+' '+time);form.append('amount',digits($('#sn-converter-payment-amount').val()));
	  var receipt=$('#sn-converter-payment-receipt')[0].files[0];if(receipt)form.append('receipt',receipt);
	  $.ajax({url:ajaxUrl,type:'POST',data:form,processData:false,contentType:false,dataType:'json'}).done(function(res){
		var payload=(res&&res.data&&typeof res.data==='object')?res.data:res;
		if(res&&res.success){ $msg.html('<div class="sn-notice sn-success">'+esc((payload&&payload.message)||res.message||'ثبت شد')+'</div>'); setTimeout(function(){ $('#sn-converter-manual-payment-modal').fadeOut(120); loadInvoices(); },650); }
		else $msg.html('<div class="sn-notice sn-error">'+esc((payload&&payload.message)||(res&&res.message)||'ثبت انجام نشد')+'</div>');
	  }).fail(function(){ $msg.html('<div class="sn-notice sn-error">خطا در ارتباط با سرور</div>'); }).always(function(){ $btn.prop('disabled',false).text('ثبت اطلاعات واریز'); });
	});

	$(document).off('click.snConverterReceiptOpen','.sn-converter-receipt-open').on('click.snConverterReceiptOpen','.sn-converter-receipt-open',function(){
	  converterReceiptInvoiceId=String($(this).data('invoice-id')||''); ensureConverterReceiptModal();
	  $('#sn-converter-receipt-card-to,#sn-converter-receipt-file').val(''); $('#sn-converter-receipt-msg').empty();
	  $('#sn-converter-receipt-modal .sn-converter-receipt-upload').text(String($(this).data('has-receipt'))==='1'?'جایگزینی فیش':'ارسال فیش');
	  $('#sn-converter-receipt-modal').fadeIn(120).attr('aria-hidden','false');
	});
	$(document).off('click.snConverterReceiptClose','.sn-converter-receipt-close').on('click.snConverterReceiptClose','.sn-converter-receipt-close',function(){ $('#sn-converter-receipt-modal').fadeOut(120).attr('aria-hidden','true'); });
	$(document).off('click.snConverterReceipt','.sn-converter-receipt-upload').on('click.snConverterReceipt','.sn-converter-receipt-upload',function(){
	  var $btn=$(this), idle=$btn.text(), input=$('#sn-converter-receipt-file')[0], to=digits($('#sn-converter-receipt-card-to').val()).replace(/\D+/g,''), $msg=$('#sn-converter-receipt-msg');
	  if(to!=='' && !/^\d{4}$/.test(to)){ $msg.html('<div class="sn-notice sn-error">۴ رقم کارت مقصد اختیاری است؛ در صورت ورود باید ۴ رقم باشد.</div>'); return; }
	  if(!input||!input.files||!input.files.length){ $msg.text('فایل فیش را انتخاب کنید.'); return; }
	  var fd=new FormData(); fd.append('action','sn_supervisor_upload_receipt'); fd.append('nonce',nonce); fd.append('invoice_id',converterReceiptInvoiceId); fd.append('card_to',to); fd.append('receipt',input.files[0]);
	  $btn.prop('disabled',true).text('در حال ارسال...');
	  $.ajax({url:ajaxUrl,type:'POST',data:fd,dataType:'json',processData:false,contentType:false,timeout:30000}).done(function(res){
		var payload=(res&&res.data&&typeof res.data==='object')?res.data:res;
		if(res&&res.success){ $msg.html('<div class="sn-notice sn-success">'+esc((payload&&payload.message)||res.message||'فیش ثبت شد.')+'</div>'); setTimeout(function(){ $('#sn-converter-receipt-modal').fadeOut(120); loadInvoices(); },650); }
		else $msg.text((payload&&payload.message)||(res&&res.message)||'آپلود انجام نشد.');
	  }).fail(function(){ $msg.text('خطا در ارتباط با سرور'); }).always(function(){ $btn.prop('disabled',false).text(idle); });
	});

  function issueNextStage($box, mode) {
    if (!ajaxUrl || !$box.length || $box.data('busy')) return;
    var invoiceId = Number($box.data('invoice-id') || 0);
    var remaining = Number($box.data('remaining') || 0);
    var amount = mode === 'partial' ? numeric($box.find('.sn-converter-next-stage-amount').val()) : 0;
    if (mode === 'partial' && (amount <= 0 || amount > remaining)) {
      invoiceNotice('مبلغ مرحله جدید باید بیشتر از صفر و حداکثر برابر مانده فاکتور باشد.', false);
      return;
    }
    if (mode === 'remaining' && !window.confirm('کل مانده این فاکتور به عنوان مرحله بعد صادر شود؟')) return;
    $box.data('busy', true).find('button,input').prop('disabled', true);
    invoiceNotice('در حال صدور مرحله جدید پرداخت...', true);
    $.ajax({
      url: ajaxUrl, method: 'POST', dataType: 'json', timeout: 30000,
      data: { action: 'sn_converter_issue_payment_stage', nonce: nonce, invoice_id: invoiceId, stage_mode: mode, amount: amount }
    }).done(function (res) {
      if (!res || !res.success) { invoiceNotice((res && res.message) || 'صدور مرحله جدید انجام نشد.', false); return; }
      invoiceNotice(res.message || 'مرحله جدید پرداخت صادر شد.', true);
      loadInvoices();
    }).fail(function (xhr) {
      var res = xhr && xhr.responseJSON ? xhr.responseJSON : null;
      invoiceNotice((res && res.message) || 'ارتباط با سرور برای صدور مرحله جدید برقرار نشد.', false);
    }).always(function () {
      $box.data('busy', false).find('button,input').prop('disabled', false);
    });
  }

  $(document).off('click.snConverterStagePartial', '.sn-converter-stage-partial').on('click.snConverterStagePartial', '.sn-converter-stage-partial', function () {
    issueNextStage($(this).closest('.sn-converter-next-stage'), 'partial');
  });
  $(document).off('click.snConverterStageRemaining', '.sn-converter-stage-remaining').on('click.snConverterStageRemaining', '.sn-converter-stage-remaining', function () {
    issueNextStage($(this).closest('.sn-converter-next-stage'), 'remaining');
  });

  $(document).off('click.snConverterInvoiceTab', '#sn-dot-converter-panel .sn-tab[data-tab="invoices"]').on('click.snConverterInvoiceTab', '#sn-dot-converter-panel .sn-tab[data-tab="invoices"]', function () {
    window.setTimeout(loadInvoices, 50);
  });
  $(document).off('click.snConverterInvoiceStatus', '.sn-converter-invoice-status-tabs .sn-subtab').on('click.snConverterInvoiceStatus', '.sn-converter-invoice-status-tabs .sn-subtab', function () {
    $('.sn-converter-invoice-status-tabs .sn-subtab').removeClass('active');
    $(this).addClass('active');
    activeInvoiceStatus = String($(this).data('status') || 'all');
    loadInvoices();
  });
  var searchTimer = null;
  $(document).off('input.snConverterInvoiceSearch', '#sn-converter-invoice-search').on('input.snConverterInvoiceSearch', '#sn-converter-invoice-search', function () {
    clearTimeout(searchTimer);
    searchTimer = window.setTimeout(loadInvoices, 320);
  });

  $(function () {
    syncProducts();
    syncPrepayment();
  });
})(jQuery);

/* 2.0.57 — show exact subscriptions already purchased by the selected customer. */
(function($){
  'use strict';
  if(!$('#sn-dot-converter-panel').length) return;
  var cfg=window.snAjax||window.snData||{}, ajax=cfg.ajaxurl||'', nonce=cfg.nonce||'';
  var timer=null, request=null, requestNo=0, lastPhone='';
  function phoneDigits(value){return String(value||'').replace(/[۰-۹]/g,function(d){return String('۰۱۲۳۴۵۶۷۸۹'.indexOf(d));}).replace(/[٠-٩]/g,function(d){return String('٠١٢٣٤٥٦٧٨٩'.indexOf(d));}).replace(/\D+/g,'');}
  function ensureTags(){
    $('#sn-converter-products-multi .sn-product-row').each(function(){if(!$(this).find('.sn-purchased-product-tag').length)$(this).append('<small class="sn-purchased-product-tag">خریداری‌شده</small>');});
  }
  function syncTags(){
    ensureTags();
    $('#sn-converter-products-multi .sn-product-row').each(function(){var marked=String($(this).find('.sn-converter-product-select option:selected').attr('data-sn-purchased')||'')==='1';$(this).toggleClass('has-purchased-subscription',marked);});
  }
  function applyPurchased(ids){
    var purchased={};(ids||[]).forEach(function(id){purchased[String(id)]=true;});
    $('#sn-converter-products-multi .sn-converter-product-select option').each(function(){var $option=$(this),id=String($option.val()||''),type=String($option.attr('data-sn-product-type')||'');if(!$option.attr('data-sn-base-label'))$option.attr('data-sn-base-label',$option.text());var marked=type==='subscription'&&!!purchased[id];$option.attr('data-sn-purchased',marked?'1':'0').text($option.attr('data-sn-base-label')+(marked?' · خریداری‌شده':''));});
    syncTags();
  }
  function loadPurchased(){
    var phone=phoneDigits($('#sn-converter-cust-phone').val());
    if(!/^09\d{9}$/.test(phone)){lastPhone='';applyPurchased([]);return;}
    if(phone===lastPhone)return;
    lastPhone=phone;requestNo+=1;var current=requestNo;
    if(request&&request.readyState!==4)request.abort();
    request=$.ajax({url:ajax,type:'POST',dataType:'json',timeout:15000,data:{action:'sn_customer_purchased_subscriptions',nonce:nonce,phone:phone}})
      .done(function(res){if(current!==requestNo)return;applyPurchased(res&&res.success&&res.data?res.data.product_ids:[]);})
      .fail(function(xhr,status){if(status!=='abort'&&current===requestNo)applyPurchased([]);});
  }
  function schedule(){clearTimeout(timer);timer=setTimeout(loadPurchased,320);}
  $(document).on('input.snConverterPurchased change.snConverterPurchased blur.snConverterPurchased','#sn-converter-cust-phone',schedule);
  $(document).on('change.snConverterPurchased','.sn-converter-product-select',syncTags);
  $(document).on('click.snConverterPurchased','#sn-converter-add-product',function(){setTimeout(function(){applyPurchased($('#sn-converter-products-multi .sn-converter-product-select').first().find('option[data-sn-purchased="1"]').map(function(){return this.value;}).get());},0);});
  $(document).ajaxComplete(function(){setTimeout(schedule,30);});
  $(function(){ensureTags();schedule();});
})(jQuery);
