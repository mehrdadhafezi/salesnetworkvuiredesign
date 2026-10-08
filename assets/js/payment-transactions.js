/* Independent transfers, shared by customer and staff forms. */
(function($,w){
 'use strict';
 if(w.snPaymentTransactions)return;
 var keys={}, contextRequest=0;
 function esc(v){return String(v==null?'':v).replace(/[&<>"']/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#039;'}[c];});}
 function number(v){return Number(String(v||'').replace(/[۰-۹]/g,function(c){return '۰۱۲۳۴۵۶۷۸۹'.indexOf(c);}).replace(/[٠-٩]/g,function(c){return '٠١٢٣٤٥٦٧٨٩'.indexOf(c);}).replace(/[,٬\s]/g,''));}
 function money(v){return Number(v||0).toLocaleString('fa-IR')+' تومان';}
 function token(){return w.crypto&&w.crypto.randomUUID?w.crypto.randomUUID():Date.now().toString(36)+'_'+Math.random().toString(36).slice(2)+'_'+Math.random().toString(36).slice(2);}
 function render(rows,summary){
  rows=rows||[];summary=summary||{};
  var h='<div class="sn-transactions"><h4>تراکنش‌های این فاکتور</h4><p class="sn-transactions-summary">در انتظار بررسی این مرحله: <strong>'+esc(money(summary.pending_amount))+'</strong> · مانده قابل ثبت: <strong>'+esc(money(summary.available_amount))+'</strong></p>';
  if(!rows.length)return h+'<p>هنوز تراکنشی ثبت نشده است.</p></div>';
  h+='<div class="sn-transaction-list">';
  rows.forEach(function(row){
   var p=row.proof||{},details=[],type={card:'کارت‌به‌کارت',paya:'پایا',pol:'پل',account:'حساب‌به‌حساب'};
   if(p.manual_transfer_type)details.push(type[p.manual_transfer_type]||p.manual_transfer_type);
   if(p.manual_card_from)details.push('کارت مبدأ: '+p.manual_card_from);
   if(p.manual_card_to)details.push('کارت مقصد: '+p.manual_card_to);
   if(p.manual_account_owner)details.push('صاحب حساب: '+p.manual_account_owner);
   if(p.manual_account_last6)details.push('حساب مبدأ: '+p.manual_account_last6);
   if(p.manual_tracking_last6)details.push('پیگیری: '+p.manual_tracking_last6);
   if(p.manual_paid_at_jalali)details.push('واریز: '+p.manual_paid_at_jalali);
   if(!details.length)details.push(row.receipt_urls&&row.receipt_urls.length?'ثبت با فیش':'اطلاعات قدیمی / درگاه');
   h+='<article class="sn-transaction-row"><div><strong>تراکنش #'+esc(row.id)+' · '+esc(row.amount_fmt||money(row.amount))+'</strong><span>'+esc(row.status_label||row.status)+'</span></div><p>'+esc(details.join(' · '))+'</p>'+w.snReceiptFiles.links(row)+'<small>مرحله '+esc(row.stage_no||row.payment_stage_no||1)+' · '+esc(row.source_label||row.payment_source_label||'')+' · '+esc(row.created_at_jalali||'')+'</small></article>';
  });
  return h+'</div></div>';
 }
 function load(target,code,id,input){
  var ticket=++contextRequest,cfg=w.snAjax||w.snData||{},$target=$(target),initialAmount=input?$(input).val():null;
  $target.html('<p>در حال دریافت تراکنش‌ها...</p>');
  return $.post(cfg.ajaxurl||w.ajaxurl||'/wp-admin/admin-ajax.php',{action:'sn_invoice_transactions',nonce:cfg.nonce||'',invoice_code:code||'',invoice_id:id||0,access_token:cfg.invoice_token||''}).done(function(res){
   if(ticket!==contextRequest)return;
   var data=res&&res.data&&typeof res.data==='object'?res.data:res;
   if(!res||!res.success){$target.text(data&&data.message||'دریافت تراکنش‌ها انجام نشد');return;}
   $target.html(render(data.transactions,data.transaction_summary));
   var available=Number((data.transaction_summary||{}).available_amount||0);
   if(input){if($(input).val()===initialAmount)$(input).val(available>0?String(available):'');$(input).attr('data-available',available).prop('readonly',false);}
  }).fail(function(){if(ticket===contextRequest)$target.text('دریافت تراکنش‌ها انجام نشد؛ اتصال را بررسی کنید.');});
 }
 w.snPaymentTransactions={render:render,load:load,number:number};
 // Keep one request key across network retries, but start a new one after success.
 $.ajaxPrefilter(function(options,original,jqXHR){
  var fd=options.data;if(!(fd instanceof FormData))return;
  var action=fd.get('action');if(['sn_upload_receipt','sn_submit_manual_payment','sn_supervisor_upload_receipt'].indexOf(action)===-1)return;
  var fields=[];fd.forEach(function(v,k){if(k==='nonce'||k==='access_token'||k==='submission_key')return;fields.push([k,v instanceof File?[v.name,v.size,v.lastModified]:v]);});
  var fingerprint=JSON.stringify(fields);
  if(!keys[fingerprint])keys[fingerprint]=token();
  var requestKey=keys[fingerprint];fd.set('submission_key',requestKey);
  jqXHR.done(function(res){if(res&&res.success&&keys[fingerprint]===requestKey)delete keys[fingerprint];});
 });
 var routes={
  'sn-seller-manual-payment-open':{modal:'#sn-seller-manual-payment-modal',amount:'#sn-seller-manual-amount'},
  'sn-supervisor-payment-open':{modal:'#sn-supervisor-manual-payment-modal',amount:'#sn-supervisor-payment-amount'},
  'sn-converter-payment-open':{modal:'#sn-converter-manual-payment-modal',amount:'#sn-converter-payment-amount'},
  'sn-supervisor-receipt-open':{modal:'#sn-supervisor-receipt-modal',amount:'#sn-supervisor-receipt-amount'},
  'sn-converter-receipt-open':{modal:'#sn-converter-receipt-modal',amount:'#sn-converter-receipt-amount'}
 };
 $(document).on('click.snTransactionsContext',Object.keys(routes).map(function(c){return '.'+c;}).join(','),function(){
  var button=this,route=routes[Object.keys(routes).filter(function(c){return button.classList.contains(c);})[0]];
  // Existing role handlers create/open the modal during this same event.
  setTimeout(function(){
   var box=$(route.modal),target=box.find('.sn-transaction-history');
   if(!target.length){target=$('<div class="sn-transaction-history"></div>');box.find('.sn-modal-body').append(target);}
   load(target,$(button).attr('data-invoice-code'),$(button).attr('data-invoice-id'),route.amount);
  },0);
 });
 $(document).on('click.snTransactionsHistory','.sn-transactions-open',function(){
  if(!$('#sn-transactions-modal').length)$('body').append('<div id="sn-transactions-modal" class="sn-modal sn-lite-modal" style="display:none"><div class="sn-modal-backdrop sn-transactions-close"></div><div class="sn-modal-card"><div class="sn-modal-head"><h3>ریز تراکنش‌ها</h3><button class="sn-modal-x sn-transactions-close" type="button">×</button></div><div class="sn-modal-body"></div></div></div>');
  $('#sn-transactions-modal').show();load('#sn-transactions-modal .sn-modal-body',$(this).attr('data-invoice-code'),$(this).attr('data-invoice-id'));
 });
 $(document).on('click.snTransactionsClose','.sn-transactions-close',function(){$('#sn-transactions-modal').hide();});
})(jQuery,window);
