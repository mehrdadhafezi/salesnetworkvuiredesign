/* Actual HR JavaScript in a DOM, including filters, queued saves and JSON payloads. */
const fs=require('fs'), path=require('path'), assert=require('assert');
const {JSDOM}=require('jsdom');
let checks=0;function check(v,m){assert(v,m);checks++;}
const html=`<div id="sn-hr-panel" data-sn-hr-server-tabs="1" data-sn-active-tab="hr-workforce">
<nav><a data-sn-hr-tab="hr-workforce" href="?sn_hr_tab=hr-workforce#hr-workforce">People</a><a data-sn-hr-tab="hr-bulk" href="?sn_hr_tab=hr-bulk#hr-bulk">Bulk</a></nav>
<input class="sn-hr-live-search" data-sn-table=".sn-hr-workforce-table"><form class="sn-hr-ajax-post-form">
<input name="action" value="sn_hr_bulk_inline_profile_update" type="hidden"><input name="_wpnonce" value="valid" type="hidden">
<select name="save_scope"><option value="selected">Selected</option><option value="visible">Visible</option></select>
<table class="sn-hr-workforce-table sn-hr-excel-filter-table"><thead><tr><th>Choose</th><th>ID</th><th>Name</th><th>Level</th><th>Actions</th></tr></thead><tbody>
${[1,2,3].map(id=>`<tr data-sn-user-id="${id}"><td><input class="sn-hr-workforce-check" name="user_ids[]" value="${id}" type="checkbox"></td><td>${id}</td><td>${['','علی','سارا','رضا'][id]}</td><td><select class="sn-hr-inline-autosave" data-sn-field="level_id"><option value="1">اول</option><option value="2">دوم</option><option value="3">سوم</option></select></td><td><span class="sn-hr-row-save-state"></span><div class="sn-hr-row-edit-fields" hidden><input type="hidden" name="rows[${id}][display_name]" value="${id===1?"O'Reilly":"کارمند"}"><input type="text" name="rows[${id}][first_name]" value="نام"><select name="rows[${id}][level_id]"><option value="1">اول</option><option value="2">دوم</option><option value="3">سوم</option></select><input name="rows[${id}][notes]" value="C:\\files"></div></td></tr>`).join('')}
</tbody></table><button type="submit">Save</button></form></div>`;
const dom=new JSDOM(html,{url:'https://crm.example/hr/?sn_hr_tab=hr-workforce',runScripts:'outside-only'});
const w=dom.window;w.eval(fs.readFileSync(require.resolve('jquery'),'utf8'));
const pending=[];w.fetch=(url,options)=>new Promise((resolve,reject)=>{
  const item={options,resolve:res=>resolve({status:200,text:()=>Promise.resolve(JSON.stringify(res))}),reject};pending.push(item);
  if(options.signal) options.signal.addEventListener('abort',()=>{const err=new Error('Aborted');err.name='AbortError';reject(err);});
});
let source=fs.readFileSync(path.join(__dirname,'../assets/js/public.js'),'utf8');
const start=source.indexOf('(function(){\n  function ready(fn)',source.indexOf('function newPositionCard'));
const end=source.indexOf('\n\n/* SN finance wallet recalculation',start);
let moduleCode=source.slice(start,end);
moduleCode=moduleCode.replace(/\}\)\(\);\s*$/, 'window.hrQA={applyTableFilters,cellValue,rowSearchText,saveSingleWorkforceRow,packWorkforceRows,sortTableRows,invalidateRow};})();');
source=source.slice(0,start)+moduleCode+source.slice(end);
w.CSS=w.CSS||{};w.CSS.escape=w.CSS.escape||((s)=>String(s));
w.scrollTo=()=>{};w.HTMLElement.prototype.scrollIntoView=()=>{};
w.eval(source);
const tick=()=>new Promise(resolve=>setImmediate(resolve));
(async()=>{
  w.document.dispatchEvent(new w.Event('DOMContentLoaded'));await tick();
  const t=w.hrQA, table=w.document.querySelector('table'), rows=[...table.querySelectorAll('tbody tr')], form=table.closest('form'), search=w.document.querySelector('.sn-hr-live-search');
  check(table.querySelectorAll('.sn-hr-excel-filter-btn').length===3,'filters installed exactly once');
  t.applyTableFilters(table);check(rows.every(r=>r.style.display!== 'none'),'empty filters show all');
  search.value='سارا';t.applyTableFilters(table);check(rows[0].style.display==='none' && rows[1].style.display==='' && rows[2].style.display==='none','search matches names');
  search.value='';table._snExcelFilterState={1:[]};t.applyTableFilters(table);check(rows.every(r=>r.style.display==='none'),'empty Excel selection hides all');
  table._snExcelFilterState={};t.applyTableFilters(table);check(rows.every(r=>r.style.display===''),'clearing restores all');
  let scans=0;const old=rows[0].querySelectorAll.bind(rows[0]);rows[0].querySelectorAll=(...args)=>{scans++;return old(...args);};
  t.invalidateRow(rows[0]);t.rowSearchText(rows[0]);let after=scans;t.rowSearchText(rows[0]);check(scans===after,'cached row search does not scan hidden forms again');
  const firstName=rows[0].querySelector('[name="rows[1][first_name]"]');firstName.value='تازه';firstName.dispatchEvent(new w.Event('input',{bubbles:true}));
  check(t.rowSearchText(rows[0]).includes('تازه'),'editing invalidates cached search');
  t.sortTableRows(table,1,'desc');check(table.tBodies[0].rows[0].getAttribute('data-sn-user-id')==='3','sorting remains functional');
  search.value='رضا';search.dispatchEvent(new w.Event('input',{bubbles:true}));search.value='سارا';search.dispatchEvent(new w.Event('input',{bubbles:true}));
  check(rows.every(r=>r.style.display===''),'typing is debounced');await new Promise(r=>setTimeout(r,160));check(rows[1].style.display==='' && rows[0].style.display==='none','latest typed filter applied');
  search.value='';t.applyTableFilters(table);
  const select=rows[0].querySelector('.sn-hr-inline-autosave');
  select.value='1';select.dispatchEvent(new w.Event('change',{bubbles:true}));
  select.value='2';select.dispatchEvent(new w.Event('change',{bubbles:true}));
  select.value='3';select.dispatchEvent(new w.Event('change',{bubbles:true}));
  check(pending.length===1 && rows[0]._snSaveRunning,'rapid changes produce one request at a time');
  const first=JSON.parse(pending[0].options.body.get('rows_json'));check(first['1'].level_id==='1','first save keeps its own snapshot');
  check(first['1'].display_name==="O'Reilly" && first['1'].notes==='C:\\files','JSON preserves text');
  pending[0].resolve({success:true});await tick();await tick();
  check(pending.length===2,'intermediate changes coalesce into one following save');
  check(JSON.parse(pending[1].options.body.get('rows_json'))['1'].level_id==='3','following save uses latest value');
  pending[1].resolve({success:true});await tick();await tick();check(!rows[0]._snSaveRunning && !rows[0].classList.contains('sn-row-saving'),'save lock released');
  rows[0].querySelector('.sn-hr-workforce-check').checked=true;
  form.dispatchEvent(new w.Event('submit',{bubbles:true,cancelable:true}));
  const bulk=pending[2].options.body, packed=JSON.parse(bulk.get('rows_json'));
  check(select.disabled,'inline controls disabled during bulk save');
  check(Object.keys(packed).join(',')==='1','bulk save sends only selected rows');
  check(![...bulk.keys()].some(k=>k.startsWith('rows[')),'row fields packed below PHP max_input_vars');
  form.dispatchEvent(new w.Event('submit',{bubbles:true,cancelable:true}));check(pending.length===3,'duplicate submits blocked');
  pending[2].resolve({success:true});await tick();await tick();check(!form.classList.contains('sn-ajax-saving') && !select.disabled,'bulk save controls restored');
  select.value='2';select.dispatchEvent(new w.Event('change',{bubbles:true}));select.value='1';select.dispatchEvent(new w.Event('change',{bubbles:true}));
  const failure=new Error('Timeout');failure.name='AbortError';pending[3].reject(failure);await tick();await tick();
  check(pending.length===4 && !rows[0]._snSaveRunning && !rows[0]._snSaveQueued,'ambiguous timeout stops queued writes and releases UI');
  check(rows[0].querySelector('.sn-hr-row-save-state').textContent.includes('وضعیت را بررسی'),'timeout explains unresolved save');
  dom.window.close();console.log(`PASS ${checks} HR DOM performance/regression checks`);
})().catch(e=>{dom.window.close();console.error(e);process.exitCode=1;});
