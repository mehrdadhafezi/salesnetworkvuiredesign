(function ($) {
  'use strict';
  var products = {}, loaded = false, loading = false;
  var rowSelector = '#sn-products-multi .sn-product-row,#sn-converter-products-multi .sn-product-row';
  function selectIn($row) { return $row.find('.sn-product-select,.sn-converter-product-select').first(); }
  window.snCatalogLineDiscount = function ($row) {
    var $select = selectIn($row), $discount = $row.find('.sn-catalog-discount-select');
    return String($discount.attr('data-product-id')) === String($select.val()) ? Number($discount.find('option:selected').attr('data-amount') || 0) : 0;
  };
  window.snCatalogSelections = function ($container) {
    var values = [];
    $container.find('.sn-product-row').each(function () { var $row = $(this); if (selectIn($row).val()) values.push(String($row.find('.sn-catalog-discount-select').val() || '')); });
    return values;
  };
  function updateRow($row) {
    var $product = selectIn($row), id = String($product.val() || ''), product = products[id];
    var $label = $row.find('.sn-catalog-discount-field');
    if (!$label.length) $label = $('<label class="sn-catalog-discount-field"><span>تخفیف هر واحد</span><select class="sn-catalog-discount-select" aria-label="تخفیف ثابت هر واحد محصول"></select></label>').appendTo($row);
    var $discount = $label.find('select'), key = id + ':' + loaded, previous = String($discount.val() || '');
    if ($discount.attr('data-catalog-key') === key) return;
    if ($discount.attr('data-product-id') !== id) previous = '';
    $discount.attr({'data-product-id': id, 'data-catalog-key': key}).empty().append($('<option value="">').text(loaded ? 'بدون تخفیف' : 'در حال دریافت تخفیف‌ها…'));
    ((product && product.discounts) || []).forEach(function (d) { $('<option>').val(d.id).attr('data-amount', d.amount).text(d.label).appendTo($discount); });
    $discount.val(previous); if ($discount.val() === null) $discount.val('');
    $label.prop('hidden', !id); $discount.prop('disabled', !loaded || !product || !product.discounts.length);
    $discount.trigger('change');
  }
  function updateCase($label) {
    var $form = $label.closest('form'), key = String($form.find('[name="option_key"]').val() || ''), $select = $label.find('select');
    if ($label.attr('data-option-key') === key) return;
    var previous = $label.attr('data-option-key') == null ? String($label.attr('data-selected') || '') : '';
    var map; try { map = JSON.parse($label.attr('data-discounts') || '{}'); } catch (e) { map = {}; }
    $label.attr('data-option-key', key); $select.empty().append('<option value="">بدون تخفیف</option>');
    (map[key] || []).forEach(function (d) { $('<option>').val(d.id).text(d.label).appendTo($select); });
    $select.val(previous); if ($select.val() === null) $select.val('');
    $label.prop('hidden', !key);
  }
  function refresh() { $(rowSelector).each(function () { updateRow($(this)); }); $('.sn-catalog-case-discount[data-discounts]').each(function () { updateCase($(this)); }); }
  window.snCatalogRefresh = refresh;
  function load() {
    if (loading || loaded || !$(rowSelector).length) return;
    loading = true;
    $.post(snSalesCatalog.ajaxUrl, {action: 'sn_sales_catalog_staff', nonce: snSalesCatalog.nonce}, null, 'json').done(function (res) {
      if (!res || !res.success) return;
      (res.data.products || []).forEach(function (p) { products[String(p.id)] = p; }); loaded = true; refresh();
    }).always(function () {
      loading = false;
      if (!loaded) $(rowSelector).find('.sn-catalog-discount-select option:first').text('دریافت تخفیف ناموفق؛ برای تلاش دوباره محصول را انتخاب کنید');
    });
  }
  $(document).on('change.snCatalog', '.sn-product-select,.sn-converter-product-select', function () { updateRow($(this).closest('.sn-product-row')); load(); });
  $(document).on('change.snCatalog', '.sn-dot-converter-update [name="option_key"]', function () { updateCase($(this).closest('form').find('.sn-catalog-case-discount[data-discounts]')); });
  $(document).ajaxComplete(function (_, __, settings) { if (String(settings.data || '').indexOf('action=sn_create_invoice') !== -1) refresh(); });
  $(function () {
    refresh(); load();
    var scheduled = false;
    new MutationObserver(function (records) {
      var relevant = records.some(function (r) { return Array.prototype.some.call(r.addedNodes, function (n) { return n.nodeType === 1 && (n.matches('.sn-product-row,.sn-catalog-case-discount') || n.querySelector('.sn-product-row,.sn-catalog-case-discount')); }); });
      if (relevant && !scheduled) { scheduled = true; setTimeout(function () { scheduled = false; refresh(); load(); }, 0); }
    }).observe(document.body, {childList: true, subtree: true});
  });
})(jQuery);
