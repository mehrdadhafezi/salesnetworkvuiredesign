/* Run with jsdom and jquery on NODE_PATH; fixture produced by test-kavenegar-switch.php DIR. */
const fs=require('fs');
const path=require('path');
const assert=require('assert');
const {JSDOM}=require('jsdom');
(async()=>{
  const html=fs.readFileSync(path.resolve(process.argv[2],'sms-settings.html'),'utf8');
  const dom=new JSDOM(html,{runScripts:'outside-only',url:'https://crm.example'});
  dom.window.eval(fs.readFileSync(require.resolve('jquery'),'utf8')); const $=dom.window.jQuery;
  for(const script of dom.window.document.querySelectorAll('script')) dom.window.eval(script.textContent);
  await new Promise(r=>setTimeout(r,20));
  let checks=0;
  for(const provider of ['faraz','kavenegar','melipayamak','faraz']){
    $('#sn_sms_provider').val(provider).trigger('change');
    for(const row of dom.window.document.querySelectorAll('.sn-provider-row')){
      assert.equal(row.style.display==='none',!row.classList.contains('sn-provider-'+provider)); checks++;
    }
    assert.equal($('#sn_sms_provider').val(),provider); checks++;
  }
  for(const event of ['customer_otp','invoice','online_payment','card_payment','payment_reward']){
    assert.equal($('[name="sn_kavenegar_template_'+event+'"]').length,1); checks++;
  }
  assert.equal($('#sn-test-sms-type option[value="otp"]').length,1); checks++;
  for(const name of ['sn_kavenegar_template_payment_reward','sn_faraz_pattern_payment_reward','sn_payment_reward_sms_enabled','sn_payment_reward_sms_template']){
    assert.equal($('[name="'+name+'"]').prop('disabled'),true); checks++;
  }
  assert.equal($('[name="sn_kavenegar_template_customer_otp"]').prop('disabled'),false); checks++;
  $('[name="sn_sms_api_key_faraz"]').val('faraz-preserved');
  $('[name="sn_sms_api_key_kavenegar"]').val('kave-preserved');
  $('#sn_sms_provider').val('kavenegar').trigger('change');
  const data=$('#sn-settings-sms :input').serializeArray();
  assert(data.some(x=>x.name==='sn_sms_api_key_faraz' && x.value==='faraz-preserved')); checks++;
  assert(data.some(x=>x.name==='sn_sms_api_key_kavenegar' && x.value==='kave-preserved')); checks++;
  dom.window.close();
  console.log('PASS '+checks+' SMS settings DOM checks');
})().catch(error=>{console.error(error);process.exit(1)});
