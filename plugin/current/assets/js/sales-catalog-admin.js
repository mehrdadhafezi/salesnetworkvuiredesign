/* Manager product access and fixed discount policy editor. */
(function ($) {
  'use strict';
  $(function () {
    $('.sn-sales-catalog').each(function () {
      var $root = $(this), loadedId = '', revision = '', busy = false, serial = 0;
      function message(text, error) { $root.find('.sn-catalog-notice').text(text).toggleClass('is-error', !!error); }
      function mode() { return $root.find('.sn-catalog-mode-input:checked').val() || 'all'; }
      function syncMode() { $root.find('.sn-catalog-allowed').prop('disabled', mode() === 'all'); }
      function request(action, data) { return $.ajax({url: snSalesCatalogAdmin.ajaxUrl, method: 'POST', dataType: 'json', data: $.extend({action: action, nonce: snSalesCatalogAdmin.nonce}, data)}); }
      function errorMessage(xhr) { return xhr.responseJSON && xhr.responseJSON.data && xhr.responseJSON.data.message || 'ارتباط با سرور انجام نشد؛ دوباره تلاش کنید.'; }
      function addAmount($cell, amount) {
        var $line = $('<div class="sn-catalog-amount-row">');
        $('<input type="text" inputmode="decimal" class="sn-catalog-amount" aria-label="مبلغ تخفیف ثابت به تومان" placeholder="مبلغ به تومان">').val(amount || '').appendTo($line);
        $('<button type="button" class="button sn-catalog-remove-amount" aria-label="حذف این مبلغ تخفیف">حذف</button>').appendTo($line);
        $cell.find('.sn-catalog-amounts').append($line);
      }
      $root.on('change', '.sn-catalog-manager', function () { serial++; loadedId = ''; $root.find('.sn-catalog-editor').prop('hidden', true); message('برای دریافت تنظیمات این مدیر، بارگذاری را بزنید.'); });
      $root.on('click', '.sn-catalog-load', function () {
        if (busy) return;
        var id = String($root.find('.sn-catalog-manager').val() || ''), ticket = ++serial;
        if (!id) { message('مدیر فروش را انتخاب کنید.', true); return; }
        busy = true; message('در حال بارگذاری…');
        request('sn_sales_catalog_load', {manager_id: id}).done(function (res) {
          if (ticket !== serial) return;
          if (!res.success) { message(res.data && res.data.message || 'بارگذاری انجام نشد.', true); return; }
          var policy = res.data.policy, $body = $root.find('tbody').empty(); loadedId = id; revision = policy.revision || '';
          $root.find('.sn-catalog-mode-input').prop('checked', false).filter('[value="' + policy.mode + '"]').prop('checked', true);
          res.data.products.forEach(function (product) {
            var row = policy.products[product.id], $tr = $('<tr>').attr('data-product-id', product.id).data('name', product.name);
            $('<td>').append($('<input type="checkbox" class="sn-catalog-allowed" aria-label="اجازه فروش این محصول">').prop('checked', !!row)).appendTo($tr);
            $('<td>').text(product.name + ' (#' + product.id + ')').appendTo($tr);
            $('<td>').text(Number(product.price).toLocaleString('fa-IR') + ' تومان').appendTo($tr);
            var $cell = $('<td><div class="sn-catalog-amounts"></div><button type="button" class="button sn-catalog-add-amount">افزودن مبلغ تخفیف</button></td>').appendTo($tr);
            ((row && row.discounts) || []).forEach(function (cents) { addAmount($cell, Number(cents) / 100); });
            $body.append($tr);
          });
          $root.find('.sn-catalog-editor').prop('hidden', false); $root.find('.sn-catalog-search').val(''); syncMode(); message('تنظیمات بارگذاری شد.');
        }).fail(function (xhr) { if (ticket === serial) message(errorMessage(xhr), true); }).always(function () { busy = false; });
      });
      $root.on('change', '.sn-catalog-mode-input', syncMode);
      $root.on('click', '.sn-catalog-add-amount', function () { var $cell = $(this).closest('td'); if ($cell.find('.sn-catalog-amount').length < 30) addAmount($cell, ''); });
      $root.on('click', '.sn-catalog-remove-amount', function () { $(this).closest('.sn-catalog-amount-row').remove(); });
      $root.on('click', '.sn-catalog-select-all,.sn-catalog-select-none', function () { $root.find('.sn-catalog-allowed').prop('checked', $(this).hasClass('sn-catalog-select-all')); });
      $root.on('input', '.sn-catalog-search', function () { var q = String($(this).val()).trim().toLowerCase(); $root.find('tbody tr').each(function () { $(this).toggle(String($(this).data('name')).toLowerCase().indexOf(q) !== -1); }); });
      $root.on('click', '.sn-catalog-save', function () {
        if (busy || !loadedId || loadedId !== String($root.find('.sn-catalog-manager').val())) return;
        var policy = {mode: mode(), revision: revision, products: {}}, ticket = serial;
        $root.find('tbody tr').each(function () {
          var $tr = $(this), discounts = $tr.find('.sn-catalog-amount').map(function () { return String($(this).val()).trim(); }).get().filter(Boolean);
          if (policy.mode === 'all' || $tr.find('.sn-catalog-allowed').prop('checked')) policy.products[$tr.attr('data-product-id')] = {discounts: discounts};
        });
        busy = true; $root.find('.sn-catalog-save').prop('disabled', true); message('در حال ذخیره…');
        request('sn_sales_catalog_save', {manager_id: loadedId, policy: JSON.stringify(policy)}).done(function (res) {
          if (ticket !== serial) return;
          if (res.success) { revision = res.data.policy.revision; message(res.data.message); }
          else message(res.data && res.data.message || 'ذخیره انجام نشد.', true);
        }).fail(function (xhr) { if (ticket === serial) message(errorMessage(xhr), true); }).always(function () { busy = false; $root.find('.sn-catalog-save').prop('disabled', false); });
      });
    });
  });
})(jQuery);
