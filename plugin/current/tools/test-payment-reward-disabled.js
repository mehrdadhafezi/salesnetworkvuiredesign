/* Run with jsdom and jquery on NODE_PATH. Uses real invoice JS and cached legacy data. */
const fs=require('fs');
const path=require('path');
const assert=require('assert');
const {JSDOM}=require('jsdom');
const dom=new JSDOM('<div id="sn-invoice-page"><div id="sn-invoice-option-host"></div><div id="sn-payment-section"></div></div>',{url:'https://crm.example/invoice/',runScripts:'outside-only'});
dom.window.eval(fs.readFileSync(require.resolve('jquery'),'utf8'));
const $=dom.window.jQuery;
const calls=[]; $.post=(...args)=>{calls.push(args);throw new Error('No network action expected');};
let source=fs.readFileSync(path.join(__dirname,'../assets/js/public-invoice.js'),'utf8');
const marker='})(jQuery);'; assert(source.endsWith(marker+'\n') || source.endsWith(marker));
source=source.replace(marker,'window.rewardTest={state,paymentRewardEnabled,rewardContactForm,renderPaymentRewardNotice,renderOptions,productInfo};'+marker);
dom.window.eval(source);
const t=dom.window.rewardTest;
let checks=0;
function check(v,msg){assert(v,msg);checks++;}
for(const status of ['pending','paid','approved','partial_paid']){
  for(const claimed of [false,true]){
    const inv={status,is_paid:status==='paid',customer_phone:'09123456789',payment_reward_claim_submitted:claimed,payment_reward_claim_phone:'09999999999',items:[{product_name:'محصول قدیمی',qty:1,payment_reward_enabled:true,has_discount_coupon:true}]};
    t.state.invoice=inv;t.state.settings={payment_reward_text:'Old banner',payment_reward_form_text:'Old form'};
    check(!t.paymentRewardEnabled(inv),'legacy product enabled flag ignored');
    check(t.rewardContactForm(inv,true)==='','no inline reward form or old claim notice');
    check(t.rewardContactForm(inv,false)==='','no modal reward form');
    $('body').append('<div class="sn-payment-reward-notice">old</div><div class="sn-payment-reward-direct-box">old</div>');
    t.renderPaymentRewardNotice(inv);
    check(!$('.sn-payment-reward-notice,.sn-payment-reward-direct-box').length,'old blocks removed on refresh');
    t.renderOptions(inv);
    check(!$('[data-option="reward-contact"]').length,'no reward action');
    check($('[data-option="discount"]').length===1,'ordinary coupon action retained');
    check(!t.productInfo(inv).includes('جایزه بعد از پرداخت آنلاین'),'product reward badge hidden');
  }
}
$('body').append('<button class="sn-invoice-option" data-option="reward-contact">old option</button><input id="sn-reward-phone" value="09999999999"><button id="sn-submit-reward-phone">old submit</button>');
$('[data-option="reward-contact"]').trigger('click');
check(!$('.sn-invoice-option-modal').length,'stale button cannot reopen reward modal');
$('#sn-submit-reward-phone').trigger('click');
check(calls.length===0,'stale submit cannot post registration or SMS');
dom.window.close();
console.log(`PASS ${checks} payment-rewards disabled DOM checks`);
