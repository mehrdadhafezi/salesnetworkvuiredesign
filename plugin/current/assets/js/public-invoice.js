/* Sales Network - Invoice Page JS (1.0.169 catalog summary removed) */
(function($){
  'use strict';
  var $page = $('#sn-invoice-page');
  if (!$page.length) return;
  var cfg = window.snAjax || window.snData || {};
  var ajax = cfg.ajaxurl, nonce = cfg.nonce;
  var accessToken = cfg.invoice_token || $page.data('token') || '';
  var state = { invoice:null, settings:{}, card:{} };

  function esc(v){return String(v==null?'':v).replace(/[&<>"']/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#039;'}[c];});}
  function strip(v){return $('<div>').html(String(v||'')).text();}
  function fa(v){return String(v==null?'':v).replace(/\d/g,function(d){return '۰۱۲۳۴۵۶۷۸۹'[d];});}
  function en(v){return String(v==null?'':v).replace(/[۰-۹]/g,function(d){return '۰۱۲۳۴۵۶۷۸۹'.indexOf(d);}).replace(/[٠-٩]/g,function(d){return '٠١٢٣٤٥٦٧٨٩'.indexOf(d);});}
  function pad(v){return String(v).padStart(2,'0');}
  function money(v){try{return Number(v||0).toLocaleString('fa-IR')+' تومان';}catch(e){return String(v||0)+' تومان';}}
  function g2j(gy,gm,gd){var gdm=[0,31,59,90,120,151,181,212,243,273,304,334],gy2=(gm>2)?gy+1:gy,days=355666+365*gy+Math.floor((gy2+3)/4)-Math.floor((gy2+99)/100)+Math.floor((gy2+399)/400)+gd+gdm[gm-1],jy=-1595+33*Math.floor(days/12053);days%=12053;jy+=4*Math.floor(days/1461);days%=1461;if(days>365){jy+=Math.floor((days-1)/365);days=(days-1)%365;}var jm=(days<186)?1+Math.floor(days/31):7+Math.floor((days-186)/30),jd=1+((days<186)?days%31:(days-186)%30);return [jy,jm,jd];}

  function statusLabel(st){var m={pre_invoice:'پیش‌فاکتور',pending:'در انتظار پرداخت',pending_payment:'در انتظار پرداخت',receipt_uploaded:'نیاز به بررسی فیش',pending_financial_approval:'در انتظار تایید مالی',approved:'تایید شده',paid:'پرداخت‌شده',rejected:'رد شده',recontact_requested:'ارتباط مجدد با کارشناس',card_submitted:'فیش ثبت شده',partial_paid:'پیش‌پرداخت تایید شده',payment_archived:'بایگانی انصرافی'};return m[st]||st||'—';}
  function productPriceHtml(it){
    if (Number(it.is_free)) return '<strong class="sn-free-label">رایگان</strong>';
    if (Number(it.has_sale) && Number(it.regular_price)>Number(it.sale_price) && Number(it.sale_price)>0) return '<del>'+money(it.regular_price)+'</del> <ins>'+money(it.sale_price)+'</ins>';
    return '<strong>'+money(it.unit_price || it.sale_price || it.total_price)+'</strong>';
  }
  function digits(v){return en(v).replace(/\D+/g,'');}
  function destinationLast4(){var d=digits((state.card&&state.card.number)||$('#sn-card-number').text()||''); return d.length>=4 ? d.slice(-4) : '';}
  function invoiceAmount(){var inv=state.invoice||{}; if(inv.current_due_amount!==undefined&&inv.current_due_amount!==null)return Number(inv.current_due_amount)||0; return Number($page.data('active-price')||0)||0;}
  function syncManualPaymentAutoFields(){
    var to=destinationLast4(), summary=(state.invoice||{}).transaction_summary||{}, amount=summary.available_amount!=null?Number(summary.available_amount):invoiceAmount();
    $('#sn-card-to4').val(to);
    $('#sn-card-to4-display').text(to ? fa(to) : 'شماره کارت مقصد در تنظیمات ثبت نشده است');
    if(!$('#sn-card-amount').val())$('#sn-card-amount').val(amount > 0 ? String(amount) : '');
    if(!$('#sn-receipt-amount').val())$('#sn-receipt-amount').val(amount > 0 ? String(amount) : '');
    $('#sn-card-amount-display').text('مانده قابل ثبت: '+money(amount));
  }

  function invoiceProductsInline(inv){
    var items = inv.items || [];
    if(!items.length) return esc(inv.product_name || '—');
    var h = '<div class="sn-products-inline-list">';
    items.forEach(function(it, index){
      var qty = Number(it.qty||1);
      h += '<div class="sn-product-inline-item">';
      h += '<div class="sn-product-inline-name">'+esc(it.product_name||'محصول')+(qty>1?' <small>× '+fa(qty)+'</small>':'')+'</div>';
      var fixed = !inv.is_dot_conversion_payment && (inv.catalog_discounts || []).find(function(d){ return Number(d.line_index) === index; });
      var priceHtml = fixed ? '<del>'+money(fixed.original_unit_price)+'</del> <ins>'+money(Number(fixed.original_unit_price)-Number(fixed.unit_discount))+'</ins>' : productPriceHtml(it);
      h += '<div class="sn-product-inline-price">'+priceHtml+'</div>';
      h += '</div>';
    });
    return h + '</div>';
  }

  function renderPaymentStages(inv, isStagedPayment){
    var stages=Array.isArray(inv.payment_stages)?inv.payment_stages:[], $box=$('#sn-payment-stage-history');
    if(!$box.length) return;
    if(!isStagedPayment || !stages.length){$box.empty().hide();return;}
    var labels={full:'پرداخت کامل',partial:'پیش‌پرداخت',remaining:'تکمیل مانده'}, statuses={pending:'در انتظار پرداخت',pending_financial_approval:'در انتظار تایید مالی',receipt_uploaded:'در انتظار تایید مالی',paid:'پرداخت‌شده',approved:'تایید مالی',rejected:'رد شده',expired:'منقضی شده'};
    var h='<div class="sn-payment-stage-history-title">'+(inv.is_dot_conversion_payment?'مراحل پرداخت این پرونده':'سوابق پرداخت این فاکتور')+'</div><div class="sn-payment-stage-list">';
    stages.forEach(function(st){
      var stageNo=Number(st.stage_no||1), currentNo=Number(inv.current_payment_stage||1), ok=st.status==='paid'||st.status==='approved', rejected=st.status==='rejected'||st.status==='expired', dt=st.display_date_jalali||'';
      var stateClass=(ok?'is-paid':(rejected?'is-rejected':'is-pending'))+(stageNo===currentNo?' is-current':'');
      var stateIcon=ok?'✓':(stageNo===currentNo?'●':'○');
      h+='<div class="sn-payment-stage-item '+stateClass+'"><span class="sn-payment-stage-state" aria-hidden="true">'+stateIcon+'</span><span class="sn-payment-stage-index">مرحله '+fa(stageNo)+(stageNo===currentNo?' · مرحله فعلی':'')+'</span><span class="sn-payment-stage-kind">'+esc(st.stage_type_label||labels[st.stage_type]||'پرداخت مرحله‌ای')+'</span><strong>'+money(st.requested_amount||0)+'</strong><small>'+esc(st.status_label||statuses[st.status]||st.status||'—')+(dt&&dt!=='—'?' — '+esc(fa(dt)):'')+'</small></div>';
    });
    $box.html(h+'</div>').show();
  }

  function renderMain(inv, card){
	$page.toggleClass('sn-is-dot-payment', !!inv.is_dot_conversion_payment);
	var duplicateWarning=String(inv.duplicate_subscription_warning||'');
	$('#sn-duplicate-subscription-warning').prop('hidden',!duplicateWarning).toggle(!!duplicateWarning).text(duplicateWarning);
    $('#sn-inv-display-code').text(inv.code||'');
    $('#sn-inv-name').text(inv.customer_name||'');
    var rawPhone=String(inv.customer_phone||''); var maskedPhone=rawPhone.length>=7?rawPhone.slice(0,4)+'***'+rawPhone.slice(-4):rawPhone;
    var second=String(inv.customer_phone_secondary||''), maskedSecond=second.length>=7?second.slice(0,4)+'***'+second.slice(-4):second;
    $('#sn-inv-phone').text(maskedPhone+(maskedSecond?' · شماره دوم: '+maskedSecond:''));
    $('#sn-inv-location').text([inv.province,inv.city].filter(Boolean).join(' — ') || '—');
    $('#sn-inv-product').html(invoiceProductsInline(inv));
    $('#sn-inv-status').text(inv.status_label || statusLabel(inv.status)); $('#sn-inv-issued-at').text(fa(inv.issued_at_jalali||'—'));
    var totalAmount=Number(inv.payment_total_amount||inv.final_total||inv.product_price||0), paidAmount=Number(inv.paid_total_amount||0), remaining=Number(inv.remaining_amount!=null?inv.remaining_amount:Math.max(0,totalAmount-paidAmount)), due=Number(inv.current_due_amount||0);
	var isStagedPayment=inv.is_staged_payment===true || String(inv.payment_plan||'')==='partial';
	var cardToCardEnabled=String((state.settings&&state.settings.card_to_card_enabled)!=null?state.settings.card_to_card_enabled:'1')==='1';
	var completed=(remaining<=0)||inv.is_paid||inv.is_fully_paid, paymentAllowed=inv.payment_allowed!==false&&remaining>0&&due>0, percent=totalAmount>0?Math.max(0,Math.min(100,Math.round((paidAmount/totalAmount)*100))):0;
    $('#sn-invoice-title').text(completed?'فاکتور فروش تکمیل‌شده':(isStagedPayment?'پیش‌فاکتور و برنامه پرداخت':'پیش‌فاکتور'));
	$('#sn-inv-total-label').text(isStagedPayment?'مبلغ کل این پرونده':'مبلغ قابل پرداخت');
	$('#sn-inv-total-help').text(isStagedPayment?'مبنای محاسبه تمام پرداخت‌های این پرونده':'').toggle(isStagedPayment);
    $('#sn-inv-total-amount').text(money(totalAmount)); $('#sn-inv-paid-amount').text(money(paidAmount)); $('#sn-inv-remaining-amount').text(money(remaining));
	$('#sn-payment-progress').toggle(isStagedPayment);
    $('#sn-payment-meter-bar').css('width',percent+'%'); $('#sn-payment-percent').text(fa(percent)+'٪'); $('#sn-card-due-label').text(isStagedPayment?'مبلغ دقیق قابل واریز در این مرحله:':'مبلغ دقیق قابل واریز:'); $('#sn-card-due-amount').text(due>0?money(due):'—');
    $('#sn-pay-online').text(due>0?'💳 پرداخت آنلاین '+money(due):'💳 پرداخت آنلاین (درگاه)');
    $('#sn-pay-card').text(due>0?'📱 کارت به کارت '+money(due):'📱 کارت به کارت');
    renderPaymentStages(inv, isStagedPayment);
	var cardNumber=paymentAllowed&&cardToCardEnabled&&card?(card.number||''):'', cardValid=paymentAllowed&&cardToCardEnabled&&digits(cardNumber).length===16;
	$('#sn-card-number').text(cardValid?cardNumber:'');
	$('#sn-card-owner').text(paymentAllowed&&card&&card.owner?card.owner:'');
	$('#sn-pay-online').prop('disabled',!paymentAllowed||Number((inv.transaction_summary||{}).pending_amount||0)>0);
	$('#sn-pay-card').toggle(cardToCardEnabled).prop('disabled',!paymentAllowed||!cardValid).attr('title',!cardToCardEnabled?'پرداخت کارت‌به‌کارت غیرفعال است':(!paymentAllowed?(inv.payment_block_reason||'پرداخت برای این فاکتور فعال نیست'):(cardValid?'':'شماره کارت مقصد در تنظیمات ثبت نشده است')));
	$('#sn-card-info').hide();
	$page.data('payment-allowed',paymentAllowed?1:0).data('payment-block-reason',inv.payment_block_reason||'پرداخت برای این فاکتور فعال نیست.');
	if(!paymentAllowed){ $('#sn-card-from4,#sn-card-to4,#sn-card-amount,#sn-card-paid-at').val(''); $('#sn-receipt-file').val(''); }
    $('#sn-card-amount,#sn-receipt-amount,#sn-receipt-file,#sn-card-manual-receipt,#sn-card-from4').val('');
    $('.sn-receipt-selection').empty();
    syncManualPaymentAutoFields();
    if(!$('#sn-public-transactions').length)$('<div id="sn-public-transactions"></div>').insertBefore('#sn-payment-section');
    $('#sn-public-transactions').html(window.snPaymentTransactions.render(inv.transactions||[],inv.transaction_summary||{}));
	if (inv.is_paid || inv.is_fully_paid || remaining<=0) { $('#sn-payment-section').hide(); $('#sn-inv-paid-msg').show().removeClass('sn-error sn-info').addClass('sn-success').text(inv.is_dot_conversion_payment?'پرداخت این پرونده ۱۰۰٪ تکمیل و تسویه شده است.':'واریزی این فاکتور به طور کامل تکمیل شده است.'); }
	else if (inv.is_dot_conversion_payment && due<=0 && remaining>0) { $('#sn-payment-section').hide(); $('#sn-inv-paid-msg').show().removeClass('sn-error sn-success').addClass('sn-info').text('این مرحله تأیید شده است. مرحله بعد پس از صدور لینک جدید فعال می‌شود.'); }
    else if (inv.status === 'payment_archived' || inv.payment_workflow_status === 'archived') { $('#sn-payment-section').hide(); $('#sn-inv-paid-msg').show().removeClass('sn-success sn-info').addClass('sn-error').text('این فاکتور با انصراف نهایی بایگانی شده است.'); }
    else if (inv.status === 'partial_paid' || (inv.payment_workflow_status === 'awaiting_assignment' && due<=0)) { $('#sn-payment-section').hide(); $('#sn-inv-paid-msg').show().removeClass('sn-error sn-success').addClass('sn-info').text('مرحله قبلی تأیید شده است. مرحله بعد توسط کارشناس صادر می‌شود.'); }
    else if (inv.status === 'receipt_uploaded' || inv.status === 'pending_financial_approval' || inv.status === 'card_submitted') { $('#sn-payment-section').toggle(Number((inv.transaction_summary||{}).available_amount||0)>0); $('#sn-inv-paid-msg').show().removeClass('sn-success sn-error').addClass('sn-info').text('تراکنش‌های ثبت‌شده در انتظار بررسی مالی هستند. مانده قابل ثبت: '+money((inv.transaction_summary||{}).available_amount||0)); }
	else if (!paymentAllowed) { $('#sn-payment-section').hide(); $('#sn-inv-paid-msg').show().removeClass('sn-success sn-error').addClass('sn-info').text(inv.payment_block_reason||(isStagedPayment?'مرحله پرداخت جدیدی برای این فاکتور صادر نشده است.':'پرداخت این فاکتور در حال حاضر فعال نیست.')); }
	else if (inv.status === 'rejected') { $('#sn-payment-section').show(); $('#sn-inv-paid-msg').show().removeClass('sn-success sn-info').addClass('sn-error').text('پرداخت قبلی رد شده است. می‌توانید دوباره فیش یا اطلاعات واریز را ثبت کنید.'); }
	else if (inv.status === 'recontact_requested') { $('#sn-payment-section').show(); $('#sn-inv-paid-msg').show().removeClass('sn-success sn-error').addClass('sn-info').text('درخواست ارتباط مجدد ثبت شده است. در صورت تمایل، همین پیش‌فاکتور همچنان برای پرداخت آنلاین یا آپلود فیش باز است.'); }
	else { $('#sn-payment-section').show(); $('#sn-inv-paid-msg').hide(); }
    $('#sn-invoice-lookup').hide(); $('#sn-invoice-detail').show();
    $page.data('active-code', inv.code).data('active-price', inv.current_due_amount || 0);
    renderOptions(inv);
    renderPaymentRewardNotice(inv);
  }
  function loadInvoice(code, cb){
    $.post(ajax,{action:'sn_invoice_info',nonce:nonce,invoice_code:code,access_token:accessToken},function(res){
      if (!(res&&res.success)) { showInline('sn-error', (res&&res.message)||'فاکتور یافت نشد'); return; }
      state.invoice = res.invoice; state.settings = res.settings || {}; state.card = res.card || {};
      renderMain(state.invoice, state.card);
      logOnce('invoice_viewed','مشاهده لینک فاکتور');
      if (typeof cb === 'function') cb(state.invoice);
    }).fail(function(xhr){var msg=(xhr.responseJSON&&xhr.responseJSON.message)||'خطا در ارتباط با سرور؛ صفحه را تازه‌سازی کنید و دوباره تلاش کنید.';showInline('sn-error',msg);});
  }
  window.snRefreshInvoiceLive = function(cb){ var code=$page.data('active-code') || $('#sn-inv-code').val(); if(code) loadInvoice(code,cb); };
  function showInline(type,msg){$('.sn-runtime-notice').remove(); $('<div class="sn-notice '+type+' sn-runtime-notice">'+esc(msg)+'</div>').insertBefore('#sn-invoice-detail').delay(4500).fadeOut(250,function(){$(this).remove();});}
  function showPaymentInline(type,msg){
    var $target = $('#sn-card-info:visible, #sn-payment-section:visible').first();
    if (!$target.length) { showInline(type,msg); return; }
    $target.find('.sn-payment-local-msg').remove();
    $('<div class="sn-notice '+type+' sn-payment-local-msg">'+esc(msg)+'</div>').prependTo($target);
  }
	function paymentActionAllowed(){
	  if(Number($page.data('payment-allowed')||0)===1) return true;
	  showInline('sn-info',String($page.data('payment-block-reason')||'پرداخت برای این فاکتور فعال نیست.'));
	  return false;
	}
  var snActionQueue=[], snActionTimer=null;
  function flushActionQueue(){
    if(!snActionQueue.length || !ajax) return;
    var code=$page.data('active-code') || $('#sn-inv-code').val(); if(!code) return;
    var batch=snActionQueue.splice(0,10);
    $.post(ajax,{action:'sn_invoice_customer_actions_batch',nonce:nonce,invoice_code:code,access_token:accessToken,events:JSON.stringify(batch)}).always(function(){ if(snActionQueue.length) snActionTimer=setTimeout(flushActionQueue,1200); });
  }
  function logAction(event,label,extra){
    var code=$page.data('active-code') || $('#sn-inv-code').val(); if(!code || !ajax) return;
    snActionQueue.push({event:event,label:label||'',extra:extra||{}});
    if(snActionQueue.length>30) snActionQueue=snActionQueue.slice(-30);
    if(snActionTimer) clearTimeout(snActionTimer);
    snActionTimer=setTimeout(flushActionQueue,900);
  }
  function logOnce(event,label){
    var code=$page.data('active-code') || $('#sn-inv-code').val(); if(!code) return;
    var key='sn_'+event+'_'+code;
    try{ if(sessionStorage.getItem(key)) return; sessionStorage.setItem(key,'1'); }catch(e){}
    logAction(event,label||'');
  }
  $(window).on('beforeunload', function(){ try{ flushActionQueue(); }catch(e){} });
  function modal(title, body){$('.sn-invoice-option-modal').remove();$('body').append('<div class="sn-invoice-option-modal sn-clean-modal" dir="rtl"><div class="sn-modal-backdrop"></div><div class="sn-modal-box"><button type="button" class="sn-modal-close">×</button><h3>'+esc(title)+'</h3><div class="sn-modal-content">'+body+'</div></div></div>');}
  function productInfo(inv){
    if (state.settings.info_show_product_info === '0') return '';
    var showDesc = state.settings.info_show_short_desc !== '0', showPrice = state.settings.info_show_price !== '0', showLottery = state.settings.info_show_lottery !== '0', showCoupon = state.settings.info_show_coupon !== '0', showImage = state.settings.info_show_image !== '0', showGallery = state.settings.info_show_gallery !== '0';
    var h='<div class="sn-option-products">';
    (inv.items||[]).forEach(function(it){
      h+='<div class="sn-option-product">';
      if (showImage && it.image_url) h+='<img class="sn-option-product-image" src="'+esc(it.image_url)+'" alt="'+esc(it.product_name||'محصول')+'">';
      h+='<h4>'+esc(it.product_name)+'</h4>';
      if (showGallery && Array.isArray(it.gallery_urls) && it.gallery_urls.length) { h+='<div class="sn-option-gallery">'; it.gallery_urls.slice(0,6).forEach(function(src){h+='<img src="'+esc(src)+'" alt="گالری محصول">';}); h+='</div>'; }
      if (showDesc) h+='<p>'+esc(strip(it.short_description)||'توضیح کوتاه محصول ثبت نشده است.')+'</p>';
      if (showPrice) h+='<div class="sn-option-price">'+productPriceHtml(it)+'</div>';
      if (showLottery && Number(it.lottery_chance_count||0)>0) h+='<small class="sn-info-chip">'+fa(Number(it.lottery_chance_count||0)*Number(it.qty||1))+' شانس قرعه‌کشی</small>';
      if (paymentRewardEnabled(inv) && it.payment_reward_enabled) h+='<small class="sn-info-chip">جایزه بعد از پرداخت آنلاین</small>';
      if (showCoupon && it.has_discount_coupon) h+='<small class="sn-info-chip">امکان استفاده از کد تخفیف</small>';
      h+='</div>';
    });
    return h+'</div>';
  }
  function totalLottery(inv){if(state.settings.info_show_lottery === '0') return 0; var n=0;(inv.items||[]).forEach(function(i){n+=Number(i.lottery_chance_count||0)*Number(i.qty||1);});return n;}
  function lotteryInfo(inv){var c=totalLottery(inv), company=(state.settings.wheel_company_name||'کمپین'), tmpl=state.settings.lottery_text_template||'با پرداخت این فاکتور {count} شانس برای شرکت در قرعه‌کشی {company} دریافت می‌کنید.';return '<p class="sn-option-lead">'+esc(tmpl.replace('{count}',fa(c)).replace('{company}',company))+'</p><div class="sn-lottery-count">'+fa(c)+' شانس</div>';}
  function allSegments(inv){var out=[];(inv.items||[]).forEach(function(i){ if(i.has_lucky_wheel && Array.isArray(i.wheel_segments)) i.wheel_segments.forEach(function(s){ if((s.label||'').trim()) out.push(s); });}); return out;}
  function hasWheel(inv){return state.settings.info_show_wheel !== '0' && (inv.items||[]).some(function(i){return !!i.has_lucky_wheel;});}
  function hasCoupon(inv){return state.settings.info_show_coupon !== '0' && (inv.items||[]).some(function(i){return !!i.has_discount_coupon;});}
  function wheelSegmentPath(startDeg, endDeg){
    var r=98, toRad=Math.PI/180, a1=(startDeg-90)*toRad, a2=(endDeg-90)*toRad;
    var x1=Math.cos(a1)*r, y1=Math.sin(a1)*r, x2=Math.cos(a2)*r, y2=Math.sin(a2)*r;
    var large=(endDeg-startDeg)>180?1:0;
    return 'M 0 0 L '+x1.toFixed(3)+' '+y1.toFixed(3)+' A '+r+' '+r+' 0 '+large+' 1 '+x2.toFixed(3)+' '+y2.toFixed(3)+' Z';
  }
  function shortenLabel(label){
    label = strip(label || 'جایزه');
    return label.length > 18 ? label.slice(0,17)+'…' : label;
  }
  function wheelDisc(segs){
    if(!segs.length) segs=[{label:'بدون جایزه',type:'empty_reward'}];
    var n=segs.length, step=360/n, colors=['#8b5cf6','#06b6d4','#22c55e','#f59e0b','#ef4444','#3b82f6','#ec4899','#14b8a6'];
    var svg='<svg class="sn-wheel-svg" viewBox="-105 -105 210 210" aria-label="گردونه شانس" role="img">';
    for(var i=0;i<n;i++){
      var a0=i*step, a1=(i+1)*step, mid=a0+step/2, label=shortenLabel(segs[i].label||('گزینه '+(i+1)));
      svg+='<path class="sn-wheel-slice" d="'+wheelSegmentPath(a0,a1)+'" fill="'+colors[i%colors.length]+'"></path>';
      svg+='<g class="sn-wheel-text" transform="rotate('+mid+') translate(0 -62) rotate('+(-mid)+')"><text text-anchor="middle" dominant-baseline="middle">'+esc(label)+'</text></g>';
    }
    svg+='<circle class="sn-wheel-hub" cx="0" cy="0" r="17"></circle></svg>';
    return '<div class="sn-wheel-shell"><div class="sn-wheel-real-wrap"><div class="sn-wheel-pointer">▼</div><div class="sn-wheel-disc sn-wheel-real" data-parts="'+n+'">'+svg+'</div></div></div>';
  }
  function wheelResultCard(title, desc, icon, tone){
    return '<div class="sn-wheel-result-card sn-wheel-result-'+esc(tone||'success')+'"><div class="sn-wheel-result-icon">'+esc(icon||'🎁')+'</div><div><strong>'+esc(title||'نتیجه گردونه')+'</strong><p>'+esc(desc||'')+'</p></div></div>';
  }
  function wheelInfo(inv){
    var item=(inv.items||[]).filter(function(i){return i.has_lucky_wheel;})[0]||{}, segs=allSegments(inv);
    var used=!!(inv.is_paid||inv.wheel_used), result=inv.wheel_reward_summary?wheelResultCard('جایزه شما', inv.wheel_reward_summary, '🎁', 'success'):'';
    return '<p class="sn-option-lead sn-wheel-lead">'+esc(strip(item.wheel_description)||'گردونه را بچرخانید و جایزه خود را ببینید.')+'</p>'+wheelDisc(segs)+'<div class="sn-wheel-action-row"><button type="button" id="sn-modal-spin-wheel" class="sn-btn sn-btn-primary sn-wheel-start-btn" '+(used?'disabled':'')+'>'+(used?'گردونه قبلاً چرخانده شده':'شروع چرخش')+'</button></div><div id="sn-modal-wheel-result" class="sn-modal-result">'+result+'</div>';
  }
  function couponInfo(){var inv=state.invoice||{}, applied=inv.coupon_code?'<div class="sn-applied-coupon"><span>کد فعال: <strong>'+esc(inv.coupon_code)+'</strong></span><button type="button" class="sn-btn sn-btn-secondary" id="sn-remove-manual-coupon">لغو کد تخفیف</button></div>':'';return '<p class="sn-option-lead">کد تخفیف</p>'+applied+'<div class="sn-coupon-box"><input type="text" id="sn-manual-coupon-code" placeholder="کد تخفیف"><button type="button" class="sn-btn sn-btn-primary" id="sn-apply-manual-coupon">اعمال کد تخفیف</button></div><div id="sn-coupon-result" class="sn-modal-result"></div>';}
  function recontactInfo(){return '<p class="sn-option-lead">'+esc(state.settings.recontact_popup_text||'اگر پیش از پرداخت فاکتور از کارشناس خود سوالی دارید، دکمه ارتباط مجدد با کارشناس را بزنید.')+'</p><button type="button" id="sn-modal-recontact-send" class="sn-btn sn-btn-secondary">ارتباط مجدد با کارشناس</button><div id="sn-modal-recontact-result" class="sn-modal-result"></div>';}
  // Retired payment reward referrals stay disabled even with cached invoice data.
  function paymentRewardEnabled(inv){return false;}
  function paymentRewardText(inv){var txt=(state.settings.payment_reward_text||'').trim(); return txt || 'اگر این محصول را از درگاه پرداخت آنلاین تهیه کنید، بعد از پرداخت جایزه یا گردونه شانس برای شما فعال می‌شود.';}
  function rewardContactSubmitted(inv){return !!(inv && inv.payment_reward_claim_submitted);}
  function rewardContactForm(inv, inline){
    if(!paymentRewardEnabled(inv)) return '';
    inv = inv || state.invoice || {};
    var text = (state.settings.payment_reward_form_text || 'شماره موبایل فردی را وارد کنید که می‌خواهید جایزه برای او ثبت و پیامک شود. شماره نباید همان شماره خریدار فاکتور باشد.');
    if(rewardContactSubmitted(inv)) return '<div class="sn-notice sn-success">شماره جایزه قبلاً ثبت شده است: <strong>'+esc(inv.payment_reward_claim_phone||'')+'</strong></div>';
    var cls = inline ? ' sn-reward-contact-form-inline' : '';
    return '<div class="sn-reward-contact-form'+cls+'"><p class="sn-option-lead">'+esc(text)+'</p><div class="sn-reward-customer-note">شماره مشتری فاکتور: <strong>'+esc(inv.customer_phone||'')+'</strong></div><label>شماره موبایل دریافت‌کننده جایزه</label><input type="tel" id="sn-reward-phone" class="sn-reward-phone-input" placeholder="09xxxxxxxxx" inputmode="tel"><button type="button" id="sn-submit-reward-phone" class="sn-btn sn-btn-primary">ثبت و ارسال پیامک</button><div id="sn-reward-phone-result" class="sn-modal-result sn-reward-phone-result"></div></div>';
  }
  function renderPaymentRewardNotice(inv){
    $('.sn-payment-reward-notice,.sn-payment-reward-direct-box').remove();
    if(!paymentRewardEnabled(inv)) return;
    var paid = (inv.status==='paid'||inv.status==='approved'||inv.is_paid);
    var cls = paid ? 'sn-success' : 'sn-info';
    var text = paid ? (rewardContactSubmitted(inv) ? 'شماره دریافت‌کننده جایزه ثبت شده است.' : 'پرداخت شما موفق بود. شماره تماس دریافت‌کننده جایزه را همین‌جا ثبت کنید.') : paymentRewardText(inv);
    var $notice = $('<div class="sn-notice '+cls+' sn-payment-reward-notice">🎁 '+esc(text)+'</div>').insertBefore('#sn-payment-section');
    if(paid){
      $('<div class="sn-payment-reward-direct-box"><h3>ثبت شماره دریافت‌کننده جایزه</h3>'+rewardContactForm(inv, true)+'</div>').insertAfter($notice);
    }
  }
  function renderOptions(inv){
    $('.sn-invoice-option-row').remove();
    var h='<div class="sn-invoice-option-row">';
    if(state.settings.info_show_product_info !== '0') h+='<button type="button" class="sn-invoice-option" data-option="product">اطلاعات محصول</button>';
    if(totalLottery(inv)>0) h+='<button type="button" class="sn-invoice-option" data-option="lottery">شانس قرعه‌کشی</button>';
    if(hasWheel(inv)) h+='<button type="button" class="sn-invoice-option" data-option="wheel">گردونه شانس</button>';
    if(paymentRewardEnabled(inv) && (inv.status==='paid'||inv.status==='approved'||inv.is_paid)) h+='<button type="button" class="sn-invoice-option" data-option="reward-contact">ثبت شماره جایزه</button>';
    if(hasCoupon(inv)) h+='<button type="button" class="sn-invoice-option" data-option="discount">کد تخفیف</button>';
    h+='</div>';
    var $host=$('#sn-invoice-option-host');
    if($host.length){$host.html($(h).children().length?h:'');}
  }
  function resultIndexFromResponse(res, segs){
    var seg = (res.payload && res.payload.segment) ? res.payload.segment : null;
    var label = String((seg && seg.label) || res.reward_label || res.reward_value || res.summary || '').trim();
    for (var i=0;i<segs.length;i++){ var candidate=String(segs[i].label||segs[i].reward_value||'').trim(); if(candidate && (candidate===label || label.indexOf(candidate)!==-1)) return i; }
    return Math.max(0, Math.min((segs.length||1)-1, Number(res.reward_index||0)||0));
  }
  function spinToIndex($disc, index, total){
    total = Math.max(1,total); var step=360/total; var center=(index*step)+(step/2); state.spinBase = (state.spinBase||0) + 1440; var finalDeg = state.spinBase + (360-center); $disc.css({transition:'none'}); $disc[0] && $disc[0].offsetHeight; $disc.css({transition:'transform 4.15s cubic-bezier(.08,.72,.12,1)',transform:'rotate('+finalDeg+'deg)'});
  }
  function ensurePaymentDateTime(){
    var $host=$('#sn-card-paid-at-picker'); if(!$host.length){ $('#sn-card-paid-at').after('<div id="sn-card-paid-at-picker" class="sn-inline-paid-datetime"></div>'); $host=$('#sn-card-paid-at-picker'); }
    var d=new Date(), parts=new Intl.DateTimeFormat('en-US',{timeZone:'Asia/Tehran',year:'numeric',month:'numeric',day:'numeric',hour:'2-digit',minute:'2-digit',hourCycle:'h23'}).formatToParts(d),now={};parts.forEach(function(p){now[p.type]=Number(p.value);});
    var j=g2j(now.year,now.month,now.day), ys='',ms='',ds='',hs='',mis='';
    for(var y=j[0]-1;y<=j[0]+1;y++) ys+='<option value="'+y+'" '+(y===j[0]?'selected':'')+'>'+fa(y)+'</option>';
    for(var m=1;m<=12;m++) ms+='<option value="'+m+'" '+(m===j[1]?'selected':'')+'>'+fa(m)+'</option>';
    for(var dd=1;dd<=31;dd++) ds+='<option value="'+dd+'" '+(dd===j[2]?'selected':'')+'>'+fa(dd)+'</option>';
    for(var h=0;h<24;h++) hs+='<option value="'+h+'" '+(h===now.hour?'selected':'')+'>'+fa(pad(h))+'</option>';
    for(var mi=0;mi<60;mi++) mis+='<option value="'+mi+'" '+(mi===now.minute?'selected':'')+'>'+fa(pad(mi))+'</option>';
    $host.html('<div class="sn-inline-date-box"><div class="sn-inline-date-title">تاریخ شمسی و ساعت تهران واریز (۲۴ ساعته)</div><div class="sn-inline-date-row"><label>سال<select id="sn-paid-jy">'+ys+'</select></label><label>ماه<select id="sn-paid-jm">'+ms+'</select></label><label>روز<select id="sn-paid-jd">'+ds+'</select></label><label>ساعت<select id="sn-paid-hh">'+hs+'</select></label><label>دقیقه<select id="sn-paid-mi">'+mis+'</select></label></div><button type="button" class="sn-btn sn-btn-sm" id="sn-public-payment-today">امروز تهران</button><div class="sn-inline-date-selected">انتخاب شده: <strong id="sn-paid-at-view"></strong></div></div>');
    function sync(){var val=$('#sn-paid-jy').val()+'/'+pad($('#sn-paid-jm').val())+'/'+pad($('#sn-paid-jd').val())+' '+pad($('#sn-paid-hh').val())+':'+pad($('#sn-paid-mi').val()); $('#sn-card-paid-at').val(val); $('#sn-paid-at-view').text(fa(val));}
    $(document).off('change.snCleanDt','#sn-paid-jy,#sn-paid-jm,#sn-paid-jd,#sn-paid-hh,#sn-paid-mi').on('change.snCleanDt','#sn-paid-jy,#sn-paid-jm,#sn-paid-jd,#sn-paid-hh,#sn-paid-mi',sync); sync();
	$(document).off('click.snToday','#sn-public-payment-today').on('click.snToday','#sn-public-payment-today',function(){var p=new Intl.DateTimeFormat('en-US',{timeZone:'Asia/Tehran',year:'numeric',month:'numeric',day:'numeric',hour:'2-digit',minute:'2-digit',hourCycle:'h23'}).formatToParts(new Date()),v={};p.forEach(function(x){v[x.type]=Number(x.value);});var today=g2j(v.year,v.month,v.day);$('#sn-paid-jy').val(today[0]);$('#sn-paid-jm').val(today[1]);$('#sn-paid-jd').val(today[2]);$('#sn-paid-hh').val(v.hour);$('#sn-paid-mi').val(v.minute);sync();});
  }

  $(document).off('click','#sn-load-invoice').on('click','#sn-load-invoice',function(e){e.preventDefault();var code=$.trim($('#sn-inv-code').val()); if(code) loadInvoice(code);});
	$(document).off('click','#sn-pay-online').on('click','#sn-pay-online',function(e){e.preventDefault();if(!paymentActionAllowed())return;logAction('pay_online_clicked','کلیک پرداخت آنلاین');var code=$page.data('active-code'), amount=invoiceAmount(); var $btn=$(this).prop('disabled',true).text('در حال اتصال به درگاه...'); $.post(ajax,{action:'sn_pay_online',nonce:nonce,invoice_code:code,access_token:accessToken},function(res){if(res&&res.success&&res.redirect) window.location.href=res.redirect; else showInline('sn-error',(res&&res.message)||'خطا در اتصال به درگاه');}).always(function(){$btn.prop('disabled',Number($page.data('payment-allowed')||0)!==1).text(amount>0?'💳 پرداخت آنلاین '+money(amount):'💳 پرداخت آنلاین (درگاه)');});});
	$(document).off('click','#sn-pay-card').on('click','#sn-pay-card',function(e){e.preventDefault();if(!paymentActionAllowed())return;if($(this).prop('disabled')){showPaymentInline('sn-error','شماره کارت مقصد معتبر در تنظیمات ثبت نشده است');return;}logAction('card_payment_selected','انتخاب پرداخت کارت‌به‌کارت');$('#sn-card-info').slideDown(); if(!$('.sn-card-choice').length){$('<div class="sn-card-choice"><button type="button" class="sn-btn sn-btn-secondary" id="sn-choice-upload">بارگذاری فیش</button><button type="button" class="sn-btn sn-btn-secondary" id="sn-choice-manual">وارد کردن اطلاعات پرداختی</button></div>').insertBefore('#sn-receipt-file');} $('#sn-card-manual-toggle').hide(); $('#sn-receipt-amount-field,#sn-receipt-file,#sn-upload-receipt,#sn-card-manual-fields').hide();});
  $(document).off('click','#sn-choice-upload').on('click','#sn-choice-upload',function(e){e.preventDefault();logAction('receipt_upload_selected','انتخاب بارگذاری فیش');$('.sn-card-choice .sn-btn').removeClass('active');$(this).addClass('active');$('#sn-card-manual-fields').hide();$('#sn-receipt-amount-field,#sn-receipt-file,#sn-upload-receipt').show();});
  $(document).off('click','#sn-choice-manual,#sn-card-manual-toggle').on('click','#sn-choice-manual,#sn-card-manual-toggle',function(e){e.preventDefault();logAction('manual_payment_selected','انتخاب ورود اطلاعات واریزی');$('.sn-card-choice .sn-btn').removeClass('active');$('#sn-choice-manual').addClass('active');$('#sn-receipt-amount-field,#sn-receipt-file,#sn-upload-receipt').hide();$('#sn-card-manual-fields').show();syncManualPaymentAutoFields();ensurePaymentDateTime();});
	$(document).off('click','#sn-upload-receipt').on('click','#sn-upload-receipt',function(e){e.preventDefault();if(!paymentActionAllowed())return;var code=$page.data('active-code'), file=$('#sn-receipt-file')[0] && $('#sn-receipt-file')[0].files[0]; if(!file){showPaymentInline('sn-error','لطفاً فایل فیش را انتخاب کنید');return;} var fd=new FormData(); fd.append('action','sn_upload_receipt'); fd.append('nonce',nonce); fd.append('invoice_code',code); fd.append('access_token',accessToken); fd.append('amount',window.snPaymentTransactions.number($('#sn-receipt-amount').val())); if(!window.snReceiptFiles.append(fd,$('#sn-receipt-file')[0]))return; var $btn=$(this).prop('disabled',true).text('در حال ارسال...'); $.ajax({url:ajax,type:'POST',data:fd,processData:false,contentType:false,success:function(res){if(res&&res.success){showPaymentInline('sn-success',res.message||'فیش ثبت شد'); logAction('receipt_uploaded','ارسال فیش پرداخت'); loadInvoice(code);} else showPaymentInline('sn-error',(res&&res.message)||'خطا در آپلود');},complete:function(){$btn.prop('disabled',Number($page.data('payment-allowed')||0)!==1).text('ارسال فیش');}});});
	$(document).on('click','#sn-choice-manual,#sn-card-manual-toggle',function(){if(!$('#sn-card-manual-receipt').length)$('#sn-card-manual-fields').append('<label class="sn-payment-receipt-field">تصویر فیش (اختیاری)<input id="sn-card-manual-receipt" type="file" multiple accept=".jpg,.jpeg,.png,.webp,.pdf"></label>');});
	$(document).off('click','#sn-submit-manual-payment').on('click','#sn-submit-manual-payment',function(e){e.preventDefault();if(!paymentActionAllowed())return;syncManualPaymentAutoFields();var code=$page.data('active-code'), from=en($('#sn-card-from4').val()), to=en($('#sn-card-to4').val()), amount=en($('#sn-card-amount').val()), paidAt=en($('#sn-card-paid-at').val()); if(!/^\d{4}$/.test(from)){showPaymentInline('sn-error','۴ رقم آخر کارت مبدا باید عددی باشد');return;} if(!/^\d{4}$/.test(to)){showPaymentInline('sn-error','شماره کارت مقصد در تنظیمات پلاگین کامل نیست');return;} amount=String(amount).replace(/,/g,''); if(!amount || isNaN(amount) || Number(amount)<=0){showPaymentInline('sn-error','مبلغ فاکتور معتبر نیست');return;} if(!paidAt){showPaymentInline('sn-error','تاریخ و ساعت واریز را انتخاب کنید');return;} var m=paidAt.match(/^(\d{4})\/(\d{2})\/(\d{2}) (\d{2}):(\d{2})$/); if(!m){showPaymentInline('sn-error','فرمت تاریخ و ساعت واریز معتبر نیست');return;} var jm=Number(m[2]), jd=Number(m[3]), hh=Number(m[4]), mi=Number(m[5]), maxd=(jm<=6?31:(jm<=11?30:29)); if(jm<1||jm>12||jd<1||jd>maxd||hh<0||hh>23||mi<0||mi>59){showPaymentInline('sn-error','تاریخ یا ساعت واریز معتبر نیست');return;} var $btn=$(this).prop('disabled',true).text('در حال ثبت...'); var fd=new FormData();[['action','sn_submit_manual_payment'],['nonce',nonce],['invoice_code',code],['access_token',accessToken],['card_from',from],['card_to',to],['amount',amount],['paid_at',paidAt]].forEach(function(v){fd.append(v[0],v[1]);});if(!window.snReceiptFiles.append(fd,$('#sn-card-manual-receipt')[0])){$btn.prop('disabled',false).text('ثبت اطلاعات واریز');return;} $.ajax({url:ajax,type:'POST',data:fd,processData:false,contentType:false,dataType:'json'}).done(function(res){if(res&&res.success){showPaymentInline('sn-success',res.message||'اطلاعات واریز ثبت شد'); logAction('manual_payment_submitted','ثبت اطلاعات واریزی'); loadInvoice(code);} else showPaymentInline('sn-error',(res&&res.message)||'خطا در ثبت');}).always(function(){$btn.prop('disabled',Number($page.data('payment-allowed')||0)!==1).text('ثبت اطلاعات واریز');});});

  $(document).off('click','#sn-submit-reward-phone').on('click','#sn-submit-reward-phone',function(e){
    if(!paymentRewardEnabled(state.invoice)){e.preventDefault();return;}
    e.preventDefault();
    var $btn=$(this), $form=$btn.closest('.sn-reward-contact-form'), $result=$form.find('.sn-reward-phone-result');
    if(!$result.length) $result=$('#sn-reward-phone-result');
    var inv=state.invoice||{}, code=$page.data('active-code')||$('#sn-inv-code').val(), phone=en(($form.find('.sn-reward-phone-input').val() || $('#sn-reward-phone').val() || '')).replace(/\D+/g,'');
    var customer=en(inv.customer_phone||'').replace(/\D+/g,''); if(customer.indexOf('98')===0) customer='0'+customer.slice(2); if(phone.indexOf('98')===0) phone='0'+phone.slice(2);
    if(!/^09\d{9}$/.test(phone)){ $result.html('<div class="sn-notice sn-error">شماره موبایل معتبر وارد کنید</div>'); return; }
    if(customer && phone===customer){ $result.html('<div class="sn-notice sn-error">شماره جایزه نباید همان شماره مشتری فاکتور باشد</div>'); return; }
    $btn.prop('disabled',true).text('در حال ثبت...');
    $.post(ajax,{action:'sn_submit_payment_reward_contact',nonce:nonce,invoice_code:code,access_token:accessToken,reward_phone:phone},function(res){
      $result.html('<div class="sn-notice '+(res&&res.success?'sn-success':'sn-error')+'">'+esc((res&&res.message)||'خطا')+'</div>');
      if(res&&res.success){ logAction('payment_reward_contact_submitted','ثبت شماره جایزه',{reward_phone:phone}); window.snRefreshInvoiceLive(function(fresh){ if(hasWheel(fresh)){ setTimeout(function(){ modal('گردونه شانس',wheelInfo(fresh)); },900); } }); }
    }).always(function(){ $btn.prop('disabled',false).text('ثبت و ارسال پیامک'); });
  });
  $(document).off('click','.sn-modal-backdrop,.sn-modal-close').on('click','.sn-modal-backdrop,.sn-modal-close',function(){$('.sn-invoice-option-modal').remove();});
  $(document).off('click','.sn-invoice-option').on('click','.sn-invoice-option',function(e){e.preventDefault();var inv=state.invoice,opt=$(this).data('option'); if(!inv)return; if(opt==='product'){ if(state.settings.info_show_product_info === '0') return; logAction('product_info_viewed','مطالعه اطلاعات محصول');modal('اطلاعات محصول',productInfo(inv));} if(opt==='lottery'){ if(state.settings.info_show_lottery === '0' || totalLottery(inv)<=0) return; logAction('lottery_info_viewed','مشاهده شانس قرعه‌کشی');modal('شانس قرعه‌کشی',lotteryInfo(inv));} if(opt==='wheel'){ if(!hasWheel(inv)) return; logAction('wheel_opened','باز کردن گردونه شانس');modal('گردونه شانس',wheelInfo(inv));} if(opt==='reward-contact'){if(!paymentRewardEnabled(inv)) return;logAction('payment_reward_contact_opened','باز کردن فرم شماره جایزه');modal('ثبت شماره جایزه',rewardContactForm(inv));} if(opt==='discount'){ if(!hasCoupon(inv)) return; logAction('coupon_opened','باز کردن کد تخفیف');modal('کد تخفیف',couponInfo(inv));} if(opt==='recontact'){logAction('recontact_popup_viewed','مشاهده توضیح ارتباط مجدد');modal('ارتباط مجدد با کارشناس',recontactInfo());}});
  $(document).off('click','#sn-modal-spin-wheel').on('click','#sn-modal-spin-wheel',function(e){
    e.preventDefault(); logAction('wheel_spin_clicked','کلیک روی شروع گردونه'); var code=$page.data('active-code')||$('#sn-inv-code').val(), $btn=$(this).prop('disabled',true).text('در حال چرخش...'), segs=allSegments(state.invoice), $disc=$('.sn-wheel-disc');
    $.post(ajax,{action:'sn_spin_invoice_wheel',nonce:nonce,invoice_code:code,access_token:accessToken,apply:'spin_only'},function(res){
      if(!(res&&res.success)){ $('#sn-modal-wheel-result').html('<div class="sn-notice sn-error">'+esc((res&&res.message)||'خطا')+'</div>'); $btn.prop('disabled',false).text('چرخاندن گردونه'); return; }
      var idx=resultIndexFromResponse(res,segs); spinToIndex($disc,idx,Math.max(1,segs.length));
      setTimeout(function(){
        var type=res.reward_type||'', rawMsg=res.summary||res.message||'', msg=esc(rawMsg); var action='', card='';
        if(type==='discount_coupon'){
          card=wheelResultCard('تبریک! کد تخفیف برنده شدید', rawMsg || 'یک کد تخفیف برای این فاکتور دریافت کردید.', '🏷️', 'success');
          action='<div class="sn-wheel-apply-box"><p>می‌خواهید این جایزه روی همین فاکتور اعمال شود؟</p><div class="sn-wheel-apply-actions"><button type="button" class="sn-btn sn-btn-primary" id="sn-apply-wheel-reward">بله، اعمال شود</button><button type="button" class="sn-btn sn-btn-secondary" id="sn-decline-wheel-reward">خیر، استفاده نمی‌کنم</button></div></div>';
        } else if(type==='free_product'){
          card=wheelResultCard('تبریک! محصول رایگان برنده شدید', rawMsg || 'یک محصول رایگان به فاکتور شما اضافه می‌شود.', '🎁', 'success');
          action='<div class="sn-wheel-apply-box"><p>می‌خواهید این جایزه به همین فاکتور اضافه شود؟</p><div class="sn-wheel-apply-actions"><button type="button" class="sn-btn sn-btn-primary" id="sn-apply-wheel-reward">بله، اضافه شود</button></div></div>';
        } else {
          card=wheelResultCard('نتیجه گردونه', rawMsg || 'این بار جایزه‌ای دریافت نشد.', '🙂', 'empty');
          action='<div class="sn-wheel-apply-box sn-wheel-empty-note">این نتیجه ثبت شد.</div>';
        }
        $('#sn-modal-wheel-result').html(card+action);
        $btn.text('گردونه چرخانده شد');
      },3900);
    });
  });
  $(document).off('click','#sn-apply-wheel-reward').on('click','#sn-apply-wheel-reward',function(e){e.preventDefault();logAction('wheel_reward_apply_clicked','درخواست اعمال جایزه گردونه');var code=$page.data('active-code')||$('#sn-inv-code').val(), $btn=$(this).prop('disabled',true).text('در حال اعمال...'); $.post(ajax,{action:'sn_apply_invoice_wheel_reward',nonce:nonce,invoice_code:code,access_token:accessToken},function(res){$('#sn-modal-wheel-result').append('<div class="sn-notice '+(res&&res.success?'sn-success':'sn-error')+'">'+esc((res&&res.message)||'خطا')+'</div>'); if(res&&res.success) window.snRefreshInvoiceLive();}).always(function(){$btn.text('اعمال شد');});});
  $(document).off('click','#sn-decline-wheel-reward').on('click','#sn-decline-wheel-reward',function(e){e.preventDefault();logAction('wheel_reward_declined','عدم استفاده از جایزه گردونه');$('#sn-modal-wheel-result').append(wheelResultCard('جایزه استفاده نشد', 'این جایزه مصرف شد و دیگر قابل استفاده نیست.', '⚠️', 'empty')); window.snRefreshInvoiceLive();});
  $(document).off('click','#sn-apply-manual-coupon').on('click','#sn-apply-manual-coupon',function(e){e.preventDefault();logAction('coupon_apply_clicked','درخواست اعمال کد تخفیف');var code=$page.data('active-code')||$('#sn-inv-code').val(), coupon=$.trim($('#sn-manual-coupon-code').val()); var $btn=$(this).prop('disabled',true).text('در حال بررسی...'); $.post(ajax,{action:'sn_apply_invoice_coupon',nonce:nonce,invoice_code:code,access_token:accessToken,coupon_code:coupon},function(res){$('#sn-coupon-result').html('<div class="sn-notice '+(res&&res.success?'sn-success':'sn-error')+'">'+esc((res&&res.message)||'خطا')+'</div>'); if(res&&res.success) window.snRefreshInvoiceLive();}).always(function(){$btn.prop('disabled',false).text('اعمال کد تخفیف');});});
  $(document).off('click','#sn-remove-manual-coupon').on('click','#sn-remove-manual-coupon',function(e){e.preventDefault();logAction('coupon_remove_clicked','لغو کد تخفیف');var code=$page.data('active-code')||$('#sn-inv-code').val(), $btn=$(this).prop('disabled',true).text('در حال لغو...'); $.post(ajax,{action:'sn_remove_invoice_coupon',nonce:nonce,invoice_code:code,access_token:accessToken},function(res){$('#sn-coupon-result').html('<div class="sn-notice '+(res&&res.success?'sn-success':'sn-error')+'">'+esc((res&&res.message)||'خطا')+'</div>'); if(res&&res.success) window.snRefreshInvoiceLive(function(){modal('کد تخفیف',couponInfo());});}).always(function(){$btn.prop('disabled',false).text('لغو کد تخفیف');});});
  $(document).off('click','#sn-modal-recontact-send').on('click','#sn-modal-recontact-send',function(e){e.preventDefault();logAction('recontact_requested','درخواست ارتباط مجدد با کارشناس');var code=$page.data('active-code')||$('#sn-inv-code').val(); $.post(ajax,{action:'sn_invoice_recontact',nonce:nonce,invoice_code:code,access_token:accessToken,note:''},function(res){$('#sn-modal-recontact-result').html('<div class="sn-notice '+(res&&res.success?'sn-success':'sn-error')+'">'+esc((res&&res.message)||'خطا')+'</div>'); if(res&&res.success) window.snRefreshInvoiceLive();});});

  var initCode=$page.data('code'), initResult=$page.data('result');
  if(initCode) { $('#sn-inv-code').val(initCode); loadInvoice(initCode, function(inv){ if(initResult==='partial_success'){try{document.querySelector('#sn-payment-progress').scrollIntoView({behavior:'smooth',block:'center'});}catch(e){}} if(initResult==='success' && paymentRewardEnabled(inv) && !rewardContactSubmitted(inv)){ try{ document.querySelector('.sn-payment-reward-direct-box') && document.querySelector('.sn-payment-reward-direct-box').scrollIntoView({behavior:'smooth', block:'center'}); }catch(e){} } else if(initResult==='success' && paymentRewardEnabled(inv) && hasWheel(inv)){ setTimeout(function(){ modal('گردونه شانس', wheelInfo(inv)); }, 450); } }); }
})(jQuery);
