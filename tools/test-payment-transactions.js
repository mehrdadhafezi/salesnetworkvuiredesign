#!/usr/bin/env node
'use strict';
const assert=require('assert'),fs=require('fs'),path=require('path'),vm=require('vm');
const root=path.resolve(__dirname,'..');
let prefilter, registered=[], tests=0;
const chain={on(){return chain;}};
const $=()=>chain;$.ajaxPrefilter=fn=>{prefilter=fn;};
const window={crypto:{randomUUID:()=>require('crypto').randomUUID()},snReceiptFiles:{links:r=>(r.receipt_urls||[]).map(u=>'<a href="'+u+'">receipt</a>').join('')}};
vm.runInNewContext(fs.readFileSync(path.join(root,'assets/js/payment-transactions.js'),'utf8'),{window,jQuery:$,document:{},FormData,File,setTimeout});
const api=window.snPaymentTransactions;
function test(name,fn){fn();tests++;console.log('PASS '+name);}
function form(amount='400000',mode='info'){
 const f=new FormData();f.append('action','sn_submit_manual_payment');f.append('invoice_code','SN-test');f.append('amount',amount);f.append('entry_mode',mode);f.append('card_from','1234');return f;
}
function prepare(fd){let done;prefilter({data:fd},{},{done(fn){done=fn;}});return {key:fd.get('submission_key'),done};}
test('request retry has same key; confirmed success starts a new transaction',()=>{
 const a=prepare(form()),b=prepare(form());assert(a.key&&a.key===b.key);b.done({success:false});assert.strictEqual(prepare(form()).key,a.key);
 a.done({success:true});assert.notStrictEqual(prepare(form()).key,a.key);
});
test('different amount or transfer details do not reuse the request key',()=>{
 const a=prepare(form()),b=prepare(form('600000'));assert.notStrictEqual(a.key,b.key);
 const f=form();f.set('card_from','5678');assert.notStrictEqual(a.key,prepare(f).key);
});
test('gateway and unrelated AJAX requests are not modified',()=>{
 const f=form();f.set('action','sn_pay_online');prepare(f);assert.strictEqual(f.get('submission_key'),null);
});
test('mixed receipt and bank-only transfers render as separate rows',()=>{
 const html=api.render([{id:11,amount:400000,status_label:'در انتظار',stage_no:1,proof:{manual_card_from:'1234',manual_paid_at_jalali:'1405/07/14 12:30'},receipt_urls:[]},{id:12,amount:600000,status_label:'در انتظار',stage_no:1,proof:{entry_mode:'receipt'},receipt_urls:['https://example.test/slip.pdf']}],{pending_amount:1000000,available_amount:0});
 assert.strictEqual((html.match(/class="sn-transaction-row"/g)||[]).length,2);assert(html.includes('1234'));assert(html.includes('1405/07/14 12:30'));assert(html.includes('slip.pdf'));
});
test('bank descriptions are escaped and Persian/Arabic amounts normalize',()=>{
 assert.strictEqual(api.number('۴۰۰٬۰۰۰'),400000);assert.strictEqual(api.number('٦٠٠,٠٠٠'),600000);
 const html=api.render([{id:1,proof:{manual_account_owner:'<script>alert(1)</script>'}}],{});assert(!html.includes('<script>'));assert(html.includes('&lt;script&gt;'));
});
test('common save inserts new rows, validates totals under lock and snapshots proof',()=>{
 const php=fs.readFileSync(path.join(root,'includes/class-sn-plugin.php'),'utf8');
 const start=php.indexOf('private function sn_save_invoice_manual_payment('),end=php.indexOf('private function sn_supervisor_can_upload_receipt_for_status',start),save=php.slice(start,end);
 assert(save.indexOf('FOR UPDATE')<save.indexOf('SN_Payment_Transactions::pending_rows'));
 assert(save.indexOf('SN_Payment_Transactions::amount_error')<save.indexOf('$wpdb->insert($payments_table'));
 assert(save.includes("'proof_data'=>wp_json_encode($proof)"));assert(save.includes('submission_key=%s'));
 assert(!/foreach\s*\([^\n]*as\s+\$key\)/.test(save),'bank-field validation must preserve the submission key');
 assert(!save.includes('$wpdb->update($payments_table'));
 const finalize=php.slice(php.indexOf('private function sn_finalize_current_payment_stage('),php.indexOf('private function sn_send_invoice_stage_sms('));
 assert(finalize.indexOf("!$transaction_summary['ready']")<finalize.indexOf('$new_paid ='));
 assert(php.includes("['view','card','manual','upload']"));
});
test('customer and sales amount fields are editable and bank/receipt totals are independent',()=>{
 const php=fs.readFileSync(path.join(root,'includes/class-sn-plugin.php'),'utf8');assert(php.includes('inputmode="decimal" id="sn-card-amount"'));assert(php.includes('id="sn-receipt-amount"'));
 for(const name of ['public-seller.js','public-supervisor.js','public-converter.js']){
  const s=fs.readFileSync(path.join(root,'assets/js',name),'utf8');
  const tags=s.match(/<input[^>]*id="sn-(?:seller-manual|supervisor-payment|converter-payment)-amount"[^>]*>/g)||[];
  assert(tags.length);assert(tags.every(t=>!t.includes('readonly')));
 }
});
console.log(tests+' independent-payment checks passed');
