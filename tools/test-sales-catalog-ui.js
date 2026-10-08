/* Run with jsdom and jquery available on NODE_PATH. No network or WordPress needed. */
const {JSDOM} = require('jsdom');
const fs = require('fs');
const path = require('path');
const assert = require('assert');
const base = path.resolve(__dirname, '..');
const wait = () => new Promise(resolve => setTimeout(resolve, 25));
function environment(html) {
  const dom = new JSDOM(html, {url: 'https://crm.test/', runScripts: 'outside-only'});
  const $ = require('jquery')(dom.window); dom.window.$ = dom.window.jQuery = $;
  return {dom, $};
}
(async () => {
  const {dom, $} = environment('<div id="sn-products-multi"><div class="sn-product-row"><select class="sn-product-select"><option value="">انتخاب</option><option value="100" data-price="1000">A</option><option value="101" data-price="2000">B</option></select><input class="sn-product-qty" value="2"></div></div><form class="sn-dot-converter-update"><select name="option_key"><option value="a">A</option><option value="b">B</option></select><label class="sn-catalog-case-discount" data-selected="10000"><select name="catalog_discount_id"></select></label></form>');
  dom.window.snSalesCatalog = {ajaxUrl:'/ajax',nonce:'test'};
  $('.sn-catalog-case-discount').attr('data-discounts', JSON.stringify({a:[{id:'10000',amount:100,label:'100 تومان'}],b:[{id:'20000',amount:200,label:'200 تومان'}]}));
  $.post = () => { const deferred=$.Deferred(); setTimeout(()=>deferred.resolve({success:true,data:{products:[{id:100,discounts:[{id:'10000',amount:100,label:'100 تومان'},{id:'20000',amount:200,label:'200 تومان'}]},{id:101,discounts:[]}]}}),0); return deferred.promise(); };
  dom.window.eval(fs.readFileSync(path.join(base,'assets/js/sales-catalog-staff.js'),'utf8'));
  await wait(); await wait();
  $('.sn-product-select').val('100').trigger('change');
  assert.equal($('.sn-catalog-discount-select option').length,3);
  $('.sn-catalog-discount-select').val('20000').trigger('change');
  assert.equal(dom.window.snCatalogLineDiscount($('.sn-product-row')),200);
  assert.deepEqual(Array.from(dom.window.snCatalogSelections($('#sn-products-multi'))),['20000']);
  const empty = $('.sn-product-row').first().clone(); empty.find('.sn-product-select').val(''); $('#sn-products-multi').prepend(empty);
  await wait();
  assert.deepEqual(Array.from(dom.window.snCatalogSelections($('#sn-products-multi'))),['20000']);
  $('.sn-product-row').last().find('.sn-product-select').val('101').trigger('change');
  assert.equal(dom.window.snCatalogLineDiscount($('.sn-product-row').last()),0);
  assert.equal($('.sn-product-row').last().find('.sn-catalog-discount-select').val(),'');
  assert.equal($('[name="catalog_discount_id"]').val(),'10000');
  $('[name="option_key"]').val('b').trigger('change');
  assert.equal($('[name="catalog_discount_id"]').val(),'');
  assert.equal($('[name="catalog_discount_id"] option:last').val(),'20000');
  dom.window.close();
  const admin = environment('<section class="sn-sales-catalog"><select class="sn-catalog-manager"><option value="10">manager</option></select><button class="sn-catalog-load">load</button><div class="sn-catalog-notice"></div><div class="sn-catalog-editor" hidden><input type="radio" class="sn-catalog-mode-input" name="mode" value="all"><input type="radio" class="sn-catalog-mode-input" name="mode" value="selected"><input class="sn-catalog-search"><table><tbody></tbody></table><button class="sn-catalog-save">save</button></div></section>');
  const a=admin.$; let saved;
  admin.dom.window.snSalesCatalogAdmin={ajaxUrl:'/ajax',nonce:'test'};
  a.ajax=options=>{const d=a.Deferred(); setTimeout(()=>{
    if(options.data.action==='sn_sales_catalog_load')d.resolve({success:true,data:{policy:{mode:'selected',products:{100:{discounts:['10000']}},revision:'v1'},products:[{id:100,name:'<img src=x onerror=alert(1)>',price:1000},{id:101,name:'B',price:2000}]}});
    else {saved=JSON.parse(options.data.policy);d.resolve({success:true,data:{policy:{revision:'v2'},message:'saved'}});}
  },0);return d.promise();};
  admin.dom.window.eval(fs.readFileSync(path.join(base,'assets/js/sales-catalog-admin.js'),'utf8'));
  await wait(); a('.sn-catalog-load').trigger('click'); await wait();
  assert.equal(a('tbody tr').length,2); assert.equal(a('tbody img').length,0); assert.equal(a('.sn-catalog-amount').val(),'100');
  assert.equal(a('.sn-catalog-allowed:checked').length,1);
  a('.sn-catalog-save').trigger('click'); await wait();
  assert.deepEqual(saved,{mode:'selected',revision:'v1',products:{100:{discounts:['100']}}});
  a('.sn-catalog-save').trigger('click'); await wait();assert.equal(saved.revision,'v2');
  admin.dom.window.close();
  console.log('PASS 15 catalog UI checks: options, selection alignment, reset, converter draft, admin save, revision and HTML escaping');
})().catch(error=>{console.error(error);process.exit(1);});
