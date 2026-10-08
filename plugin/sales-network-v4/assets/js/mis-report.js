/* MIS report: all business values come from the scoped server snapshot. */
(() => {
'use strict';
const root=document.getElementById('sn-mis-report'); if(!root) return;
const cfg=window.snMisReport, form=root.querySelector('form'), status=document.getElementById('sn-mr-status');
let job='', run=0, page=1, pages=1, options={people:[],batches:[]}, busy=false;
const roles=['sales_deputy','sales_manager','senior_supervisor','supervisor','seller'];
const labels={valid:'معتبر',duplicate:'تکراری',invalid:'نامعتبر',no_answer:'جواب نداده',callback:'تماس مجدد',not_purchased:'عدم خرید',pre_invoice:'پیش‌فاکتور',unrecorded:'ثبت نشده',not_delivered:'بدون تحویل؛ پیگیری ندارد',empty:'دلیل خالی',short:'دو نویسه یا کمتر',meaningless:'نشانه‌ای / تکراری / مبهم',recorded:'ثبت‌شده؛ کنترل شکلی',not_applicable:'نامرتبط',approved:'تأییدشده',paid:'پرداخت‌شده',completed:'تکمیل‌شده',awaiting_payment:'منتظر پرداخت',pending_financial_approval:'منتظر تأیید مالی',cancelled:'لغوشده',rejected:'ردشده',unassigned:'بدون تخصیص',delivered_to_seller:'تحویل به فروشنده',distributed_forward:'انتقال به سطح بعد',returned_to_mis:'برگشت به MIS'};
const label=v=>labels[v]||String(v??'');
function el(tag,text,parent){const n=document.createElement(tag);if(text!==undefined)n.textContent=String(text??'');if(parent)parent.append(n);return n;}
function message(text,error=false){status.textContent=text;status.dataset.error=error?'1':'0';}
async function api(op,args={}){
 const data=new URLSearchParams({action:'sn_mis_report',nonce:cfg.nonce,op,...args});
 const r=await fetch(cfg.ajax,{method:'POST',credentials:'same-origin',body:data}); let j;
 try{j=await r.json();}catch(e){throw new Error('پاسخ معتبر از سرور دریافت نشد؛ وضعیت درخواست را بررسی و گزارش را دوباره بسازید.');}
 if(!r.ok||!j.success)throw new Error(j.data?.message||'درخواست گزارش ناموفق بود.');return j.data;
}
function table(parent,headers,rows){const w=el('div',undefined,parent);w.className='sn-mr-wrap';const t=el('table',undefined,w),h=el('tr',undefined,el('thead',undefined,t));headers.forEach(v=>el('th',v,h));const b=el('tbody',undefined,t);rows.forEach(row=>{const tr=el('tr',undefined,b);row.forEach(v=>{const td=el('td',undefined,tr);if(v instanceof Node)td.append(v);else td.textContent=String(v??'');});});return t;}
function section(parent,title){const d=el('details',undefined,parent);el('summary',title,d);return d;}
function setOptions(select,items,empty){const old=select.value;select.replaceChildren();const z=el('option',empty,select);z.value=select.name==='campaign'||select.name==='category'?'':'0';items.forEach(([v,t])=>{const o=el('option',t,select);o.value=String(v);});if([...select.options].some(o=>o.value===old))select.value=old;}
function under(id,parent){let seen=new Set();while(id&&!seen.has(id)){if(id===parent)return true;seen.add(id);id=Number(options.people.find(p=>p.id===id)?.parent||0);}return false;}
function hierarchy(){let parent=0;roles.forEach(role=>{const s=form.elements[role];setOptions(s,options.people.filter(p=>p.role===role&&(!parent||under(p.id,parent))).map(p=>[p.id,p.name+' #'+p.id]),'همه افراد مجاز این سطح');if(Number(s.value))parent=Number(s.value);});}
function batches(){const campaign=form.elements.campaign.value;setOptions(form.elements.batch,options.batches.filter(b=>!campaign||b.campaign_code===campaign).map(b=>[b.id,b.title+' #'+b.id]),'همه دسته‌ها');}
function drawSummary(s){const box=document.getElementById('sn-mr-summary');box.replaceChildren();
 const basis=form.elements.basis.selectedOptions[0].textContent;el('h3','پرونده‌های منتخب بر مبنای '+basis,box);
 el('p','گردآوری UTC: '+s.progress.created_at+' تا '+s.progress.finished_at+' — '+(s.filters.from||s.filters.to?'بازه انتخاب‌شده':'همه زمان‌ها'),box);
 const cards=el('div',undefined,box);cards.className='sn-mr-cards';const totals=s.totals;
 const metrics=[['ردیف / پرونده یکتا',totals.total],['تخصیص به مدیریت (تاریخچه)',totals.manager_assigned],['تحویل به فروشنده (تاریخچه)',totals.delivered],['بدون سابقه تحویل',Number(totals.total)-Number(totals.delivered)],['بازگشت‌خورده',totals.returned],['منتقل‌شده',totals.transferred],['دارای سابقه پیش‌فاکتور',totals.pre_case],['دارای فاکتور با پیوند قطعی',totals.linked],['پرونده با فاکتور تأییدشده',totals.approved],['پرونده با فروش تکمیل‌شده',totals.completed],['دلیل نیازمند بازبینی',totals.reason_review],['فروشنده نهایی متمایز',totals.seller_count],['اختلاف داده ثبت‌شده',totals.issue_count]];
 metrics.forEach(([title,value])=>{const c=el('div',undefined,cards);c.className='sn-mr-card';el('span',title,c);el('strong',Number(value).toLocaleString('fa-IR'),c);});
 for(const [key,title] of Object.entries({validity:'اعتبار ردیف',flow:'پیگیری جاری فروشنده',distribution:'توزیع جاری MIS',reason_quality:'کنترل کیفیت دلیل عدم خرید'})){table(section(box,title),[title,'تعداد پرونده'],s.breakdown[key].map(r=>[label(r.label),r.total]));}
 const inv=s.invoice_totals;el('h3','فاکتورهای یکتا و پرداخت؛ مستقل از پیگیری فروشنده',box);el('p',`فاکتور یکتا: ${inv.total_count} | مبلغ: ${inv.amount} | پرداخت تأییدشده: ${inv.paid} | مانده: ${inv.remaining} | فروش تکمیل‌شده: ${inv.completed} | مبلغ فروش تکمیل‌شده: ${inv.completed_amount}`,box);
 table(box,['وضعیت فاکتور','گردش پرداخت','تعداد فاکتور','مبلغ','پرداخت','مانده','تأیید','تکمیل','ناقص'],s.invoice_groups.map(g=>[label(g.status),label(g.workflow),g.total_count,g.amount,g.paid,g.remaining,g.approved,g.completed,g.partial]));
 const heads=['پرونده یکتا','تحویل','جواب نداده','تماس مجدد','عدم خرید','پیگیری پیش‌فاکتور','ثبت نشده','دارای پیش‌فاکتور','پیوند قطعی','تأیید','تکمیل'];const keys=['total','delivered','no_answer','callback','not_purchased','pre_invoice','unrecorded','pre_case','linked','approved','completed'];
 table(section(box,'تفکیک کمپین و دسته'),['کمپین','دسته واردات','دسته داده',...heads],s.groups.map(g=>[g.campaign,(options.batches.find(b=>Number(b.id)===Number(g.batch_id))?.title||'')+' #'+g.batch_id,g.category,...keys.map(k=>g[k])]));
 table(section(box,'عملکرد افراد در سلسله‌مراتب جاری HR'),['سطح','فرد',...heads],s.people.map(g=>[g.role,g.name,...keys.map(k=>g[k])]));
 table(section(box,'دریافت واقعی هر فرد در تاریخچه؛ مستقل از مالک فعلی'),['سمت هنگام رویداد','گیرنده','پرونده یکتا','تعداد رویداد تحویل','اولین دریافت','آخرین دریافت'],s.historical_people.map(h=>[h.role,h.name,h.unique_cases,h.events,h.first_at,h.last_at]));
 if(s.shared_invoices.length)table(section(box,'فاکتور مشترک بین چند پرونده؛ در مبلغ کل یک بار شمرده شده'),['شناسه فاکتور','تعداد پرونده'],s.shared_invoices.map(i=>[i.invoice_id,i.case_count]));
 const notes=section(box,'تعریف شاخص‌ها و منشأ داده');s.notes.forEach(n=>el('p',n,notes));
}
async function loadPage(n){const data=await api('page',{job,page:n});page=data.page;pages=data.pages;const box=document.getElementById('sn-mr-table');box.replaceChildren();
 table(box,['جزئیات','ردیف','کمپین / دسته','شماره و مشتری','اعتبار','تحویل‌گیرنده جاری','توزیع','پیگیری','آخرین فعالیت','دلیل اصلی','کیفیت دلیل','فاکتورها / وضعیت','پرداخت','مانده','تکمیل فروش'],data.rows.map(c=>{const b=el('button','باز کردن');b.type='button';b.addEventListener('click',()=>showDetail(c.row_id).catch(e=>message(e.message,true)));return [b,c.row_id,c.campaign+' / '+c.batch_title+' #'+c.batch_id,c.phone+'\n'+c.customer_name,label(c.validity),c.owner_name,label(c.distribution),label(c.flow)+'\n'+c.flow_source,c.last_activity,c.reason+(c.cancel_reasons?'\nانصراف فاکتور: '+c.cancel_reasons:''),label(c.reason_quality),c.invoices.map(i=>i.invoice_code+' #'+i.id+' '+label(i.status)+' / '+label(i.payment_workflow_status)).join('\n'),c.paid,c.remaining,c.completed?'بله':'خیر'];}));
 document.getElementById('sn-mr-page').textContent=`صفحه ${page} از ${pages} — ${data.total} پرونده`;
 document.getElementById('sn-mr-prev').disabled=page<=1;document.getElementById('sn-mr-next').disabled=page>=pages;
}
async function showDetail(id){const c=await api('detail',{job,row:id}),box=document.getElementById('sn-mr-detail');box.replaceChildren();el('h3','جزئیات پرونده #'+c.row_id,box);
 el('p','منشأ: sn_mis_data_rows#'+c.row_id+' / sn_mis_import_batches#'+c.batch_id+' / sn_mis_lead_pool: '+c.pool_ids.join(',')+' / sn_distribution_items: '+c.item_ids.join(','),box);
 el('p','پیگیری: '+label(c.flow)+' — '+c.flow_source,box);el('p','دلیل اصلی: '+c.reason,box);el('p','منشأ دلیل: '+c.reason_source+' — کنترل کیفیت: '+label(c.reason_quality),box);
 c.issues.forEach(v=>el('p','نیازمند بررسی: '+v,box));
 table(box,['فاکتور','کد','پیوند قطعی','وضعیت','گردش','مبلغ','پرداخت','مانده','تکمیل','دلیل انصراف','اختلاف‌ها'],c.invoices.map(i=>[i.id,i.invoice_code,i.link_source,label(i.status),label(i.payment_workflow_status),i.total,i.paid,i.remaining,i.completed?'بله':'خیر',i.payment_archive_reason||'',i.issues.join('\n')]));
 for(const i of c.invoices){table(section(box,'مراحل پرداخت فاکتور '+i.invoice_code),['مرحله','مبلغ','وضعیت','روش','زمان تأیید','زمان پرداخت'],i.stages.map(s=>[s.stage_no,s.requested_amount,s.status,s.pay_method,s.approved_at,s.paid_at]));}
 el('h4','تمام رویدادهای ثبت‌شده توزیع؛ ترتیب شناسه لاگ',box);
 table(box,['لاگ / آیتم','زمان','رویداد','اقدام‌کننده','فرستنده','گیرنده','سمت فرستنده','سمت گیرنده','متن ثبت‌شده'],[...c.history].sort((a,b)=>Number(a.id)-Number(b.id)).map(l=>[l.id+' / '+l.item_id,l.created_at,l.action,l.actor_user_id_name,l.from_user_id_name,l.to_user_id_name,l.from_position,l.to_position,l.note]));
 if(!c.history.length)el('p','تاریخچه‌ای در sn_distribution_item_logs ثبت نشده؛ مالک جاری جایگزین تاریخچه نشده است.',box);
 if(c.assigned_at)el('p','زمان تخصیص ذخیره‌شده در ردیف MIS: '+c.assigned_at+'؛ این مقدار لاگ مستقل نیست.',box);
 box.scrollIntoView({behavior:'smooth',block:'start'});
}
async function build(){const token=++run;busy=true;form.querySelector('[type=submit]').disabled=true;document.getElementById('sn-mr-results').hidden=true;document.getElementById('sn-mr-summary').replaceChildren();document.getElementById('sn-mr-detail').replaceChildren();document.getElementById('sn-mr-download').replaceChildren();
 try{if(job){await api('cancel',{job});job='';}const f=Object.fromEntries(new FormData(form));let p=await api('start',{filters:JSON.stringify(f)});job=p.job;
 while(p.status==='building'&&token===run){message('در حال گردآوری بسته‌ای؛ '+p.scanned+' ردیف بررسی شد…');p=await api('step',{job});}
 if(token!==run)return;const s=await api('summary',{job});drawSummary(s);await loadPage(1);document.getElementById('sn-mr-results').hidden=false;message('گزارش آماده است. اعداد و خروجی اکسل از یک مجموعه داده ساخته می‌شوند.');
 }catch(e){message(e.message,true);}finally{busy=false;form.querySelector('[type=submit]').disabled=false;}}
form.addEventListener('submit',e=>{e.preventDefault();if(!busy)build();});
form.addEventListener('reset',()=>setTimeout(()=>{hierarchy();batches();},0));roles.forEach(role=>form.elements[role].addEventListener('change',hierarchy));form.elements.campaign.addEventListener('change',batches);
document.getElementById('sn-mr-prev').addEventListener('click',()=>loadPage(page-1).catch(e=>message(e.message,true)));document.getElementById('sn-mr-next').addEventListener('click',()=>loadPage(page+1).catch(e=>message(e.message,true)));
document.getElementById('sn-mr-cancel').addEventListener('click',async()=>{run++;if(job&&!busy){try{await api('cancel',{job});job='';document.getElementById('sn-mr-results').hidden=true;document.getElementById('sn-mr-summary').replaceChildren();message('گزارش لغو شد.');}catch(e){message(e.message,true);}}else message('گردآوری متوقف می‌شود؛ پس از پایان بسته جاری می‌توانید گزارش را دوباره بسازید.');});
document.getElementById('sn-mr-export').addEventListener('click',async function(){this.disabled=true;const exportJob=job;const out=document.getElementById('sn-mr-export-status');try{let e;do{e=await api('export',{job:exportJob});if(job!==exportJob)throw new Error('گزارش تغییر کرده است؛ خروجی را از گزارش تازه بسازید.');out.textContent='ساخت اکسل: '+e.status;}while(e.status!=='ready');out.textContent='اکسل آماده است؛ شمارش پرونده، فاکتور و بازبینی تطبیق داده شد.';const host=document.getElementById('sn-mr-download');host.replaceChildren();const f=el('form',undefined,host);f.method='post';f.action=cfg.download;for(const [k,v]of Object.entries({action:'sn_mis_report_download',_wpnonce:cfg.downloadNonce,job:exportJob})){const n=el('input',undefined,f);n.type='hidden';n.name=k;n.value=v;}el('button','دانلود فایل XLSX',f).type='submit';}catch(e){out.textContent=e.message;}finally{this.disabled=false;}});
document.getElementById('sn-mr-install')?.addEventListener('click',async function(){this.disabled=true;try{await api('install');location.reload();}catch(e){message(e.message,true);this.disabled=false;}});
(async()=>{try{message('دریافت فیلترهای مجاز…');options=await api('options');setOptions(form.elements.campaign,[...new Set(options.batches.map(b=>b.campaign_code).filter(Boolean))].map(c=>[c,c]),'همه کمپین‌ها');setOptions(form.elements.category,[...new Set(options.batches.map(b=>b.data_category).filter(Boolean))].map(c=>[c,c]),'همه دسته‌های داده');hierarchy();batches();if(cfg.ready)await build();else message('مدیر سایت باید ابتدا ساختار گزارش را نصب کند.');}catch(e){message(e.message,true);}})();
})();
