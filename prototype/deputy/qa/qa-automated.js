// Automated QA (SN-203 Sales Deputy: Pass A structure, Pass B interactions, Pass C matrix). Run: python3 -m http.server 8777 (from prototype/), then node qa/qa-automated.js — requires playwright.
const { chromium } = require('playwright');
const BASE='http://127.0.0.1:8777/deputy/index.html';
const fail=[];const ok=(c,m)=>{console.log((c?'PASS ':'FAIL ')+m);if(!c)fail.push(m)};
async function mk(b,w,h){const ctx=await b.newContext({viewport:{width:w,height:h},reducedMotion:'reduce'});await ctx.addInitScript(()=>{try{localStorage.setItem('snproto.deputy_welcome_v1_seen','1')}catch(e){}});const p=await ctx.newPage();p.errs=[];p.on('pageerror',e=>p.errs.push(e.message));return p}
const contrast=`(()=>{const lum=c=>{const [r,g,b]=c.map(v=>{v/=255;return v<=.03928?v/12.92:Math.pow((v+.055)/1.055,2.4)});return .2126*r+.7152*g+.0722*b};
const parse=s=>{const m=s.match(/[\\d.]+/g);return m?m.map(Number):null};
const bgOf=e=>{let a=[255,255,255],stack=[];for(;e;e=e.parentElement){const c=parse(getComputedStyle(e).backgroundColor);if(c&&(c.length<4||c[3]>0)){stack.push(c)}if(c&&(c.length<4||c[3]===1))break}
let base=[255,255,255];for(let i=stack.length-1;i>=0;i--){const c=stack[i],al=c.length>3?c[3]:1;base=base.map((v,k)=>c[k]*al+v*(1-al))}return base};
const bad=[];const w=document.createTreeWalker(document.querySelector('#ws'),NodeFilter.SHOW_TEXT);let n;const seen=new Set();
while(n=w.nextNode()){if(!n.textContent.trim())continue;const e=n.parentElement;if(seen.has(e)||!e.getClientRects().length)continue;seen.add(e);const cs=getComputedStyle(e);if(cs.visibility==='hidden'||cs.opacity==='0')continue;
const fg=parse(cs.color);const bg=bgOf(e);const f=fg.slice(0,3);const a=fg.length>3?fg[3]:1;const fb=f.map((v,k)=>v*a+bg[k]*(1-a));const L1=lum(fb),L2=lum(bg);const r=(Math.max(L1,L2)+.05)/(Math.min(L1,L2)+.05);const big=parseFloat(cs.fontSize)>=24||(parseFloat(cs.fontSize)>=18.6&&+cs.fontWeight>=700);if(r<(big?3:4.5)&&!e.closest('[disabled],[aria-disabled=true],.dim,.off'))bad.push(e.className.toString().slice(0,30)+'|'+e.textContent.trim().slice(0,20)+'|'+r.toFixed(2))}
return bad.slice(0,6)})()`;

(async()=>{const b=await chromium.launch({});
const VIEWS=['ov','perf','exc','rep','alloc','hr','mon','inv','diag','wallet'];
let p=await mk(b,1366,768);let t;
const open=async(sel)=>{await p.click(sel);await p.waitForTimeout(300)};
const dr=async()=>await p.innerText('#drawer');
const go=async(q)=>{await p.goto(BASE+'?'+q);await p.waitForTimeout(400)};
// ---- PASS A structure
for(const v of VIEWS){await go('view='+v);ok((await p.innerText('#ws')).length>300,'render '+v);}
// ---- PASS B
await go('view=ov');t=await p.innerText('#ws');
ok(await p.locator('.kpis .kpi:has-text("۱۸۰۰۱")').count()===0,'User ID is identity context, not a KPI');
ok(!/رتبه‌بندی نیست/.test('')&&!(await p.locator('#ws :text-matches("ضعیف‌ترین|بهترین")').count()),'no best/worst ranking text');
ok(await p.locator('tr[data-row^="mgr:"]').count()===3,'overview: three Manager branches');
ok(t.includes('F02')&&t.includes('F03')&&t.includes('نیازمند تطبیق')&&t.includes('پوشش ناقص'),'metric trust: F02 recon + F03 partial shown');
ok(t.includes('این کل سازمان نیست'),'scope is not entire org');
await open('[data-attn="inactive"]');t=await dr();ok(t.includes('موارد این دسته')&&t.includes('منابع انسانی'),'class drawer lists instances');await p.keyboard.press('Escape');
await open('[data-act="open-mgr:MG3"]');t=await dr();ok(t.includes('پوشش منابع شاخه')&&t.includes('شمرده نمی‌شود')&&t.includes('پوشش اثبات نشده'),'branch drawer: source coverage matrix');
ok(await p.locator('#drawer [data-act^="assign-to"]').count()===1&&(await dr()).includes('مشروط'),'branch drawer: allocation only conditional');await p.keyboard.press('Escape');
// exceptions
await go('view=exc');ok(await p.locator('tr[data-row^="exc:"]').count()>=11,'exceptions: instances listed');
t=await p.innerText('#ws');ok(t.includes('نامشخص (UNKNOWN)'),'unknown owner stays UNKNOWN');ok(t.includes('نیازمند اعتبارسنجی'),'bulk exception action needs validation');
await open('[data-act="open-exc:E1"]');t=await dr();ok(t.includes('مسئول گام بعدی')&&t.includes('علامت حل')&&t.includes('ممنوع در این پنل')&&t.includes('اقدام مجاز شما')&&t.includes('مالک / مسئول'),'exception drawer: subject/owner/state/why/may/next/signal');
ok(!(await p.locator('#drawer button:has-text("ارجاع")').count()),'no escalation command');await p.keyboard.press('Escape');
await open('[data-act="open-exc:E10"]');ok((await dr()).includes('رتبه معاون مالک پیش‌فرض نیست'),'unknown owner: rank is not default owner');await p.keyboard.press('Escape');
const classes=new Set();for(const c of await p.locator('.exc-cls .pill').allInnerTexts())classes.add(c.trim());ok(classes.size>=11,'all 11 spec exception classes present ('+classes.size+')');
// performance
await go('view=perf');ok(await p.locator('.tb-tag.tb-event').count()>0,'event columns tagged');
await p.click('[data-tbs="snap"]');await p.waitForTimeout(200);ok(await p.locator('.tb-tag.tb-snap').count()>0&&await p.locator('.tb-tag.tb-event').count()===0,'snapshot vs event never mixed');
ok(await p.locator('.explorer-ctl button[disabled]:has-text("وضعیت تاریخی")').count()===1,'historical as-of disabled (no proof)');
ok((await p.innerText('.four-concepts')).includes('مالک اعتبار'),'four concepts legend');
await p.click('[data-act="drill:MG1"]');await p.waitForTimeout(250);ok((await p.innerText('.scope-crumb')).includes('فرهاد'),'drill Manager → crumb');
ok(await p.locator('tr[data-row="senior:U1"]').count()===1&&await p.locator('tr[data-row="senior:U4"]').count()===0,'drill narrows to the Manager branch');
await p.click('[data-act="drill:U1"]');await p.waitForTimeout(250);ok(await p.locator('tr[data-row="team:T1"]').count()===1,'drill Senior → Supervisors');
await p.click('[data-act="drill:T1"]');await p.waitForTimeout(250);ok((await p.innerText('.scope-crumb')).split('‹').length>=4&&await p.locator('tr[data-row="seller:6"]').count()===1,'drill Supervisor → Sellers, full chain');
for(const m of ['mgr','senior','sup','seller','inv','case']){await p.click('[data-pm="'+m+'"]');await p.waitForTimeout(150);ok((await p.innerText('#ws')).length>200,'perf mode '+m);}
await p.click('[data-act="pscope:all"]');await p.click('[data-pm="mgr"]');await p.click('[data-pb="hist"]');ok(await p.locator('.unk-row').count()===1,'historical unknown row (not parent today)');
await p.click('[data-pm="inv"]');t=await p.innerText('#ws');ok(t.includes('زنجیره هنگام صدور')&&t.includes('مالک اعتبار')&&t.includes('ناشناخته'),'invoice mode: current vs historical vs credit vs event');
await open('[data-act="open-metrics"]');t=await dr();ok(t.includes('گروه شمارش')&&t.includes('فرمول')&&t.includes('تازگی')&&t.includes('F02'),'metric metadata');await p.keyboard.press('Escape');
// diagnostics
for(const q of ['f02','f03','f04','lineage']){await go('view=diag&dq='+q);t=await p.innerText('#ws');ok(t.length>400,'diag '+q);ok(!(await p.locator('#ws button:has-text("اصلاح"), #ws button:has-text("ادغام")').count()),'diag '+q+': no repair control');}
await go('view=diag&dq=f02');t=await p.innerText('#ws');ok(t.includes('pre_invoice')&&t.includes('رفع‌شده فرض نشده'),'F02: predicate evidence, not fixed');
await go('view=diag&dq=f03');t=await p.innerText('#ws');ok(t.includes('شمرده می‌شود')&&t.includes('شمرده نمی‌شود')&&t.includes('پوشش ناقص')&&t.includes('صفر لید قدیمی به معنی صفر پرونده نیست'),'F03: coverage matrix, zero legacy ≠ zero cases');
await go('view=diag&dq=f04');t=await p.innerText('#ws');ok(t.includes('بازتولید اعلام نمی‌شود')&&t.includes('تأیید نشده'),'F04: not reproduced claim, cause unverified');
// allocation
await go('view=alloc&sel=MG1');await open('[data-act="review-assign"]');t=await dr();
ok(t.includes('مسیر اصلی')&&!t.includes('مسیر عبور از مدیر'),'primary Manager path: no bypass panel');
ok(t.includes('حضانت')&&t.includes('رابطه گیرنده')&&t.includes('محدوده')&&t.includes('منبع')&&t.includes('وابستگی مالی')&&t.includes('Apply recheck'),'preview shows custody/relationship/scope/source/financial/recheck');
ok(await p.locator('[data-act="commit-assign"]:not([disabled])').count()===1,'direct: commit enabled after preview');await p.keyboard.press('Escape');
await go('view=alloc&sel=U1');await open('[data-act="review-assign"]');t=await dr();
ok(t.includes('عبور از مدیر')&&t.includes('SD-G05')&&t.includes('سیاست تخصیص مستقیم/عبور از مدیر مصوب نیست'),'bypass: exceptional + policy gap');
ok(await p.locator('[data-act="commit-assign"][disabled]').count()===1,'bypass commit gated by acknowledgement');
await p.check('[data-confirm]');await p.waitForTimeout(200);ok(await p.locator('[data-act="commit-assign"]:not([disabled])').count()===1,'ack enables commit (no approval chain invented)');await p.keyboard.press('Escape');
await go('view=alloc&aq=assign');ok(await p.locator('[data-recip="U3"][disabled]').count()===1&&await p.locator('[data-recip="XM"][disabled]').count()===1,'inactive Senior and out-of-scope Manager disabled');
ok(await p.locator('[data-recip="S40"][disabled]').count()===1,'seller with unverified write eligibility disabled (visible ≠ writable)');
await p.click('[data-amode="count"]');await p.waitForTimeout(200);ok(await p.locator('[data-recip="U1"][disabled]').count()===1&&await p.locator('[data-recip="MG1"]:not([disabled])').count()===1,'count mode: only primary path');
for(const f of ['partial','retry','unknown','conflict','recipchg','outscope']){await go('view=alloc&flow='+f);await p.waitForTimeout(400);t=await dr();ok(t.length>80,'flow '+f);
 if(f==='partial'){for(const w of ['درخواست‌شده','واجد شرایط','اعمال‌شده','ارسال‌نشده','ردشده','ناموفق','نامعلوم'])ok(t.includes(w),'outcome strip '+w);ok(t.includes('برابر «درخواست‌شده» است'),'outcome counts reconcile');}
 if(f==='unknown')ok(await p.locator('[data-act^="reconcile"]').count()===1&&await p.locator('[data-act^="retry"]').count()===0,'unknown: reconcile, no blind retry');
 if(f==='retry')ok(await p.locator('[data-act^="retry"]').count()===1,'retry only failed');
 if(f==='recipchg')ok(t.includes('گیرنده پیش از ثبت تغییر کرد')&&await p.locator('[data-act="commit-assign"]').count()===0,'recipient changed: conflict, no commit');
 if(f==='outscope')ok(t.includes('خارج از محدوده')&&await p.locator('[data-act="commit-assign"]').count()===0,'out-of-scope allocation blocked');}
await go('view=alloc&aq=return');t=await p.innerText('#ws');ok(t.includes('برگشت فقط پیش‌نمایش است')&&t.includes('F01')&&t.includes('معاون همه چیز را بازپس نمی‌گیرد'),'return: preview only, rank ≠ recall');
for(const [c,txt] of [['C-50110','مشروط به F01'],['C-50120','رتبه، اجازه بازپس‌گیری نمی‌دهد'],['C-50133','fail-closed'],['C-50130','بازخوانی'],['C-50140','مصرف‌شده'],['C-50150','OPD-03']]){await open('[data-act="open-ret:'+c+'"]');ok((await dr()).includes(txt),'return '+c+' → '+txt);await p.keyboard.press('Escape');}
await go('view=alloc&aq=return');await p.check('[data-sel="rsel"][data-id="C-50110"]');await p.waitForTimeout(250);await open('[data-act="review-return"]');ok(await p.locator('#drawer [data-act="commit-return"]').count()===0,'bulk return preview has no commit');await p.keyboard.press('Escape');
await go('view=alloc&sim=emptypool');ok((await p.innerText('#ws')).includes('موجودی شخصی شما خالی است'),'empty pool state (zero ≠ proof)');
// invoices
await go('view=inv&iq=all');t=await p.innerText('#ws');ok(t.includes('فقط‌خواندنی')&&t.includes('نمونه اخیر')&&t.includes('تومان'),'invoice: read-only, bounded sample, unit');
ok(await p.locator('#ws [data-act^="resend"],#ws [data-act^="copy-link"]').count()===0,'invoice list: no assist controls');
await open('[data-act="open-inv:71942055"]');t=await dr();ok(t.includes('نیازمند اعتبارسنجی')&&t.includes('در این نقش نیست')&&t.includes('مالک اعتبار'),'invoice drawer: policy + ownership');
ok(await p.locator('#drawer [data-act^="resend"],#drawer [data-act^="copy-link"],#drawer button:has-text("تأیید مالی")').count()===0,'invoice drawer: no write/assist/finance buttons');await p.keyboard.press('Escape');
await open('[data-act="open-inv:89663612"]');ok((await dr()).includes('F02'),'pre-invoice drawer notes F02');await p.keyboard.press('Escape');
// HR
await go('view=hr');t=await p.innerText('#ws');ok(t.includes('SD-G09')&&t.includes('بررسی گروهی ندارد'),'HR conditional + no bulk');
await open('[data-act="open-hr:612"]');t=await dr();ok(t.includes('تأیید این مرحله')&&t.includes('اعمال نهایی توسط منابع انسانی')&&t.includes('بررسی‌کننده فعلی')&&t.includes('بررسی‌کنندگان قبلی')&&t.includes('بررسی‌کننده بعدی'),'HR drawer: step vs final, current/prior/next reviewer');
await p.click('[data-act="hr-reject:612"]');await p.waitForTimeout(200);ok((await dr()).includes('دلیل بنویسید'),'HR reject needs reason');
await p.click('[data-act="hr-step:612"]');await p.waitForTimeout(300);ok((await p.innerText('.toasts')).includes('هنوز اعمال نشده'),'HR step toast not-applied');
await go('view=hr&hq=chain');await open('[data-act="open-hr:605"]');t=await dr();ok(t.includes('فقط مشاهده')&&await p.locator('#drawer [data-act^="hr-step"]').count()===0,'HR not-assigned: observe only (rank ≠ reviewer)');await p.keyboard.press('Escape');
await go('view=hr&hq=closed');ok((await p.innerText('#ws')).includes('اعمال ناموفق'),'HR failed apply distinct');
await go('view=hr&hq=review');await open('[data-act="open-hr:609"]');ok((await dr()).includes('OPD-05'),'termination: handover undefined');await p.keyboard.press('Escape');
// monitor
await go('view=mon');t=await p.innerText('#ws');ok(t.includes('برش نمونه')&&t.includes('حذف نیست')&&t.includes('مجاز نیست'),'monitor: bounded, null ≠ 0');
ok(await p.locator('#ws button:has-text("تأیید"), #ws button:has-text("احیا")').count()===0,'monitor: no review/revive controls');
for(const q of ['archrec','xnum','xout']){await p.click('[data-mq="'+q+'"]');await p.waitForTimeout(150);}
await p.click('[data-mq="arch"]');await p.click('[data-monauth="unauth"]');ok((await p.innerText('#ws')).includes('مجاز نشده است'),'monitor: unauthorized state, no fake data');
await p.click('[data-monauth="unavail"]');ok((await p.innerText('#ws')).includes('در دسترس نیست'),'monitor: unavailable state');
// reports
await go('view=rep&report=perf_by_mgr');t=await p.innerText('#ws');ok(t.includes('نه کل سازمان')&&t.includes('۶۰ تا ۱۲۰')&&t.includes('میدان‌های حساس محدود')&&t.includes('نسخه ۱'),'report: scope, bound, sensitive fields, metadata');
ok(await p.locator('.rep-head button[disabled]:has-text("خروجی")').count()===1,'export disabled (permission unverified)');
ok(await p.locator('.rep-ctrl button[disabled]:has-text("وضعیت تاریخی")').count()===1,'report historical as-of disabled');
await p.selectOption('[data-rscope]','MG1');await p.waitForTimeout(200);ok((await p.innerText('#ws')).includes('سرپرست ارشد'),'report drill Manager → Senior/Supervisor');
ok(await p.locator('[data-rscope] option:text("خارج از محدوده")[disabled]').count()===1,'out-of-scope branch not selectable');
await go('view=wallet');ok((await p.innerText('#ws')).includes('خواندن محض'),'wallet pure-read');
// forbidden authority text
for(const v of VIEWS){await go('view='+v);t=await p.innerText('#ws');ok(!/ثبت بازپرداخت|تأیید پرداخت/.test(t),'no finance authority copy in '+v);}
// sims
for(const s of ['loading','empty','noresult','tableError','pageError','unauthorized','stale','incomplete','branchfail','parthier','unknownhist','offline','emptypool']){for(const v of ['ov','perf','exc']){await go('view='+v+'&sim='+s);}}
ok(p.errs.length===0,'no JS errors in B: '+p.errs.slice(0,3));
await go('view=ov&sim=branchfail');t=await p.innerText('#ws');ok(t.includes('دریافت نشد')&&t.includes('شاخه‌های دیگر بارگذاری شدند')&&await p.locator('tr[data-row="mgr:MG1"]:not(.failed)').count()===1,'one branch failed: others loaded, not zero');
await go('view=ov&sim=parthier');ok((await p.innerText('#ws')).includes('سلسله‌مراتب')&&(await p.innerText('#ws')).includes('ناقص'),'partial hierarchy state');
await go('view=perf&sim=stale');ok((await p.innerText('#ws')).includes('قدیمی'),'stale branch labelled');
await go('view=ov');await p.keyboard.press('Control+k');await p.waitForTimeout(300);ok(await p.locator('#palette:not([hidden])').count()===1,'palette opens');
// keyboard / focus: drawer focus return
await go('view=exc');await p.focus('[data-act="open-exc:E2"]');await p.keyboard.press('Enter');await p.waitForTimeout(300);ok(await p.evaluate(()=>document.querySelector('#drawer').contains(document.activeElement)),'drawer takes focus');await p.keyboard.press('Escape');await p.waitForTimeout(200);
ok(await p.evaluate(()=>document.activeElement&&document.activeElement.tagName==='TR'&&document.activeElement.closest('#ws')),'focus returns to the list row (shared drawer behaviour)');
ok(await p.locator('main#ws').count()===1&&await p.locator('nav[aria-label]').count()>=2&&await p.locator('table[aria-label]').count()>=1,'landmarks + labelled tables');
// ---- PASS C matrix
let cbad=0;
for(const [w,h] of [[1920,1080],[1366,768],[1024,768],[768,1024],[390,844]]){
 for(const [th,dn,fc] of [['light','comfortable',0],['dim','compact',0],['dark','comfortable',1]]){
  const q=await mk(b,w,h);
  for(const v of VIEWS){await q.goto(BASE+'?view='+v+'&theme='+th+'&density='+dn+(fc?'&focus=1':''));await q.waitForTimeout(220);
   const r=await q.evaluate(()=>({o:document.documentElement.scrollWidth>innerWidth+1}));if(r.o){cbad++;console.log('OVERFLOW',w,th,v)}
   if(w===1366||w===390){const c=await q.evaluate(contrast);if(c.length){cbad++;console.log('CONTRAST',w,th,v,JSON.stringify(c))}}}
  if(q.errs.length){cbad++;console.log('ERR',w,th,q.errs[0])}
  await q.context().close();}}
ok(cbad===0,'pass C matrix clean ('+cbad+' issues)');
console.log('FAILS',fail.length);await b.close()})();
