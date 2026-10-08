// Automated QA (SN-202 Sales Manager: Pass B interactions + Pass C matrix). Run: python3 -m http.server 8777 (from prototype/), then node qa/qa-automated.js — requires playwright.
const { chromium } = require('playwright');
const BASE='http://127.0.0.1:8777/manager/index.html';
const fail=[];const ok=(c,m)=>{console.log((c?'PASS ':'FAIL ')+m);if(!c)fail.push(m)};
async function mk(b,w,h,extra){const ctx=await b.newContext({viewport:{width:w,height:h},reducedMotion:'reduce'});await ctx.addInitScript(()=>{try{localStorage.setItem('snproto.manager_welcome_v1_seen','1')}catch(e){}});const p=await ctx.newPage();p.errs=[];p.on('pageerror',e=>p.errs.push(e.message));return p}
const contrast=`(()=>{const lum=c=>{const [r,g,b]=c.map(v=>{v/=255;return v<=.03928?v/12.92:Math.pow((v+.055)/1.055,2.4)});return .2126*r+.7152*g+.0722*b};
const parse=s=>{const m=s.match(/[\\d.]+/g);return m?m.map(Number):null};
const bgOf=e=>{let a=[255,255,255],stack=[];for(;e;e=e.parentElement){const c=parse(getComputedStyle(e).backgroundColor);if(c&&(c.length<4||c[3]>0)){stack.push(c)}if(c&&(c.length<4||c[3]===1))break}
let base=[255,255,255];for(let i=stack.length-1;i>=0;i--){const c=stack[i],al=c.length>3?c[3]:1;base=base.map((v,k)=>c[k]*al+v*(1-al))}return base};
const bad=[];const w=document.createTreeWalker(document.querySelector('#ws'),NodeFilter.SHOW_TEXT);let n;const seen=new Set();
while(n=w.nextNode()){if(!n.textContent.trim())continue;const e=n.parentElement;if(seen.has(e)||!e.getClientRects().length)continue;seen.add(e);const cs=getComputedStyle(e);if(cs.visibility==='hidden'||cs.opacity==='0')continue;
const fg=parse(cs.color);const bg=bgOf(e);const f=fg.slice(0,3);const a=fg.length>3?fg[3]:1;const fb=f.map((v,k)=>v*a+bg[k]*(1-a));const L1=lum(fb),L2=lum(bg);const r=(Math.max(L1,L2)+.05)/(Math.min(L1,L2)+.05);const big=parseFloat(cs.fontSize)>=24||(parseFloat(cs.fontSize)>=18.6&&+cs.fontWeight>=700);if(r<(big?3:4.5)&&!e.closest('[disabled],[aria-disabled=true],.dim,.off'))bad.push(e.className.toString().slice(0,30)+'|'+e.textContent.trim().slice(0,20)+'|'+r.toFixed(2))}
return bad.slice(0,6)})()`;
(async()=>{const b=await chromium.launch({});
// ---- PASS B (interactions)
let p=await mk(b,1366,768);let t;
const open=async(sel)=>{await p.click(sel);await p.waitForTimeout(300)};
const dr=async()=>await p.innerText('#drawer');
// overview / exception console
await p.goto(BASE+'?view=ov');await p.waitForTimeout(400);
t=await p.innerText('#ws');
ok(!/شناسه حساب/.test(t.split('عملیات و استثناها')[0]||'')&&await p.locator('.kpis .kpi:has-text("۱۸۰۰۳")').count()===0,'User ID is identity context, not a KPI tile');
ok(await p.locator('.attn-row').count()>=9,'exception console lists spec exception classes');
await open('[data-attn="cons"]');t=await dr();ok(t.includes('مسئول گام بعدی')&&t.includes('علامت حل')&&t.includes('ممنوع در این پنل')&&t.includes('اقدام مجاز شما'),'exception drawer: owner/may/may-not/next actor/resolution signal');
ok(!(await p.locator('#drawer button:has-text("ارجاع")').count()),'no escalation command');
await p.keyboard.press('Escape');
await open('[data-act="open-unit:U3"]');t=await dr();ok(t.includes('غیرفعال')&&await p.locator('#drawer [data-act^="assign-to"]').count()===0,'inactive سرپرست ارشد: not a recipient');await p.keyboard.press('Escape');
await p.goto(BASE+'?view=ov');await p.waitForTimeout(300);ok((await p.innerText('.attn-foot')).includes('آزادشدن'),'resolved ≠ released note');
// distribution
await p.goto(BASE+'?view=dist&sel=U1');await p.waitForTimeout(400);
await open('[data-act="review-assign"]');ok(await p.locator('[data-act="commit-assign"]:not([disabled])').count()===1,'direct recipient: preview then commit enabled');ok(!(await dr()).includes('عبور از سطح (مشروط)'),'direct path has no skip-level panel');await p.keyboard.press('Escape');
await p.goto(BASE+'?view=dist&sel=T1');await p.waitForTimeout(400);
await open('[data-act="review-assign"]');t=await dr();
ok(t.includes('سطح(های) ردشده')&&t.includes('M-G08')&&t.includes('سیاست عبور از سطح مصوب نیست'),'skip-level: bypass + policy gap shown');
ok(await p.locator('[data-act="commit-assign"][disabled]').count()===1,'skip-level commit gated by explicit confirmation');
await p.check('[data-confirm]');await p.waitForTimeout(200);ok(await p.locator('[data-act="commit-assign"]:not([disabled])').count()===1,'confirm enables skip-level commit (no approval chain invented)');await p.keyboard.press('Escape');
await p.goto(BASE+'?view=dist&sel=U3');await p.waitForTimeout(300);ok(await p.locator('[data-recip="U3"][disabled]').count()===1,'inactive Senior recipient disabled');
await p.goto(BASE+'?view=dist&aq=assign');await p.waitForTimeout(300);await p.click('.exc-path summary');await p.waitForTimeout(150);ok(await p.locator('[data-recip="S40"][disabled]').count()===1,'seller with unverified write eligibility disabled (visible ≠ writable)');
await p.click('[data-amode="count"]');await p.waitForTimeout(200);ok(await p.locator('[data-recip="T1"][disabled]').count()===1,'count mode: only direct recipients');
// bulk outcome language
for(const f of ['partial','retry','unknown','conflict']){await p.goto(BASE+'?view=dist&flow='+f);await p.waitForTimeout(600);t=await dr();ok(t.length>80,'flow '+f);
 if(f==='partial'){for(const w of ['درخواست‌شده','واجد شرایط','اعمال‌شده','ارسال‌نشده','ردشده','ناموفق','نامعلوم'])ok(t.includes(w),'outcome strip has '+w);ok(t.includes('برابر «درخواست‌شده» است'),'outcome counts reconcile');ok(await p.locator('.outcome-strip .os:not(.zero)').count()>=5,'partial shows applied+skipped+rejected+failed non-zero, never "complete"');ok(!t.includes('کامل انجام شد'),'partial is not reported as complete');}
 if(f==='unknown'){ok(await p.locator('[data-act^="reconcile"]').count()===1&&await p.locator('[data-act^="retry"]').count()===0,'unknown: reconcile, no blind retry');}
 if(f==='retry'){ok(await p.locator('[data-act^="retry"]').count()===1,'retry only failed');}}
// return = preview only
await p.goto(BASE+'?view=dist&aq=return');await p.waitForTimeout(300);t=await p.innerText('#ws');ok(t.includes('برگشت فقط پیش‌نمایش است')&&t.includes('F01'),'return: F01 conditional banner');
await open('[data-act="open-ret:C-40110"]');t=await dr();ok(await p.locator('#drawer [data-act^="commit-return"]').count()===0&&t.includes('مشروط به F01'),'return eligible: commit disabled / conditional');await p.keyboard.press('Escape');
await open('[data-act="open-ret:C-40120"]');t=await dr();ok(t.includes('رتبه، اجازه بازپس‌گیری نمی‌دهد')&&t.includes('مسئول اقدام بعدی'),'return blocked: rank ≠ authority + next actor');await p.keyboard.press('Escape');
await open('[data-act="open-ret:C-40133"]');t=await dr();ok(t.includes('fail-closed'),'unknown link fails closed');await p.keyboard.press('Escape');
await p.goto(BASE+'?view=dist&aq=return&rf=all');await p.waitForTimeout(250);await p.check('[data-sel="rsel"][data-id="C-40110"]');await p.waitForTimeout(250);await open('[data-act="review-return"]');ok(await p.locator('#drawer [data-act="commit-return"]').count()===0,'bulk return preview has no commit');await p.keyboard.press('Escape');
// performance
await p.goto(BASE+'?view=perf');await p.waitForTimeout(300);
ok(await p.locator('.tb-tag.tb-event').count()>0,'event columns tagged');
await p.click('[data-tbs="snap"]');await p.waitForTimeout(200);ok(await p.locator('.tb-tag.tb-snap').count()>0&&await p.locator('.tb-tag.tb-event').count()===0,'snapshot vs event never mixed');
ok(await p.locator('.explorer-ctl button[disabled]:has-text("وضعیت تاریخی")').count()===1,'historical as-of disabled (no proof)');
await p.click('[data-act="drill:U1"]');await p.waitForTimeout(250);ok((await p.innerText('.scope-crumb')).includes('کامران صدری'),'drill Senior → breadcrumb');
await p.click('[data-act="drill:T1"]');await p.waitForTimeout(250);ok((await p.innerText('.scope-crumb')).split('‹').length>=3,'drill Supervisor → seller breadcrumb');
for(const m of ['unit','sup','seller','inv','case','recon']){await p.click('[data-pm="'+m+'"]');await p.waitForTimeout(150);ok(await p.locator('.tbl,.rc-grid').count()>0,'perf mode '+m);}
await p.click('[data-pm="recon"]');t=await p.innerText('#ws');ok(t.includes('علت اختلاف')&&t.includes('تأیید نشده')&&t.includes('صفر صحیح نیست'),'F04: cause unverified, not zero');
ok(!(await p.locator('#ws button:has-text("اصلاح")').count()),'F04: no fix/overwrite control');
await p.click('[data-act="pscope:all"]');await p.click('[data-pm="unit"]');await p.click('[data-pb="hist"]');ok(await p.locator('.unk-row').count()===1,'historical unknown row');
await open('[data-act="open-metrics"]');t=await dr();ok(t.includes('گروه شمارش')&&t.includes('مبنای زمانی')&&t.includes('فرمول')&&t.includes('تازگی')&&t.includes('باز:'),'metric metadata incl. open cohort');
// invoices
await p.goto(BASE+'?view=inv&iq=all');await p.waitForTimeout(300);await open('[data-act="open-inv:89663612"]');t=await dr();
ok(t.includes('OPD-10')&&t.includes('در این پنل ممکن نیست')&&!(await p.locator('#drawer button:has-text("تأیید پرداخت")').count()),'invoice: finance-only locked, receipt/edit needs validation');
ok(await p.locator('#drawer [data-act^="copy-link"]').count()===1&&await p.locator('#drawer [data-act^="resend"]').count()===1,'invoice assist: copy/resend only');
await p.click('#drawer [data-act^="resend"]');await p.waitForTimeout(300);ok((await dr()).includes('نتیجه تحویل تأیید نشده'),'resend outcome not assumed');await p.keyboard.press('Escape');
await open('[data-act="open-inv:62010458"]');ok(await p.locator('#drawer [data-act^="resend"]').count()===0,'completed invoice: no resend');await p.keyboard.press('Escape');
// archives
await p.goto(BASE+'?view=arch');await p.waitForTimeout(300);t=await p.innerText('#ws');ok(t.includes('۲۰۰')&&t.includes('حذف نیست'),'archive: bound + not deletion');
for(const r of ['noanswer','assess','sub','product','productx']){await p.click('[data-ar="'+r+'"]');await p.waitForTimeout(150);}
await p.click('[data-ar="sub"]');t=await p.innerText('#ws');ok(t.includes('برش')&&t.includes('پنج روز'),'archive: authorized-empty is bounded, rule shown');
await p.click('[data-ar="all"]');await open('[data-act="open-arch:AR-812"]');t=await dr();ok(t.includes('احیا')&&t.includes('نیازمند اعتبارسنجی')&&t.includes('مجاز نیست'),'archive drawer: revive deferred, delete not allowed');
ok(!(await p.locator('#drawer button:has-text("احیا")').count()),'no revive button');ok(t.includes('مالک نمایش‌داده‌شده'),'archive owner ≠ custody');await p.keyboard.press('Escape');
await open('[data-act="open-arch:AR-824"]');ok((await dr()).includes('پرداخت جزئی به‌تنهایی احیا'),'partial payment ≠ revival');await p.keyboard.press('Escape');
// extra-number
await p.goto(BASE+'?view=xnum');await p.waitForTimeout(300);t=await p.innerText('#ws');ok(t.includes('۱۰۰')&&t.includes('بررسی گروهی ندارد'),'xnum: 100-row bound + no bulk');
await open('[data-act="open-xreq:XR-1201"]');t=await dr();ok(t.includes('سرنخ قدیمی')&&t.includes('اتصال نامشخص')&&t.includes('فقط برای همین فروشنده'),'xnum: approval side effect + CaseRef mapping + dup scope explicit');
ok(await p.locator('[data-act^="x-approve"][disabled]').count()===1,'xnum approve gated by confirm');
ok(!(await dr()).includes('الزامی برای رد'),'xnum reject reason not forced');
await p.check('[data-confirm]');await p.waitForTimeout(200);await p.click('[data-act^="x-approve"]');await p.waitForTimeout(400);ok((await p.innerText('.toasts')).includes('نگاشت'),'xnum approved toast');
await p.goto(BASE+'?flow=xfail');await p.waitForTimeout(700);await p.check('[data-confirm]');await p.waitForTimeout(200);await p.click('[data-act^="x-approve"]');await p.waitForTimeout(300);t=await dr();ok(t.includes('ثبت انجام نشد')&&t.includes('در انتظار تصمیم'),'xnum failure keeps pending');
await p.goto(BASE+'?flow=xunknown');await p.waitForTimeout(700);await p.check('[data-confirm]');await p.waitForTimeout(200);await p.click('[data-act^="x-approve"]');await p.waitForTimeout(400);t=await dr();ok(t.includes('نامعلوم')&&await p.locator('[data-act^="x-reconcile"]').count()===1,'xnum unknown → reconcile');
await p.goto(BASE+'?view=xnum&xq=decided');await p.waitForTimeout(300);ok(await p.locator('.tbl tbody tr').count()>=3,'xnum decided queue');
// HR
await p.goto(BASE+'?view=hr');await p.waitForTimeout(300);await open('[data-act="open-hr:512"]');
await p.click('[data-act="hr-reject:512"]');await p.waitForTimeout(200);ok((await dr()).includes('دلیل بنویسید'),'HR reject needs reason');
t=await dr();ok(t.includes('اعمال نهایی'),'HR final apply distinguished');
await p.click('[data-act="hr-step:512"]');await p.waitForTimeout(300);ok((await p.innerText('.toasts')).includes('هنوز اعمال نشده'),'HR step toast not-applied');
await p.click('[data-hq="chain"]');ok(await p.locator('.hr-card').count()>=2,'HR chain queue');
await p.click('[data-hq="closed"]');ok((await p.innerText('#ws')).includes('اعمال ناموفق'),'HR failed apply is a distinct state');
await p.goto(BASE+'?view=hr&hq=review');await p.waitForTimeout(300);await open('[data-act="open-hr:509"]');ok((await dr()).includes('OPD-05'),'termination: handover undefined');await p.keyboard.press('Escape');
ok(await p.locator('#ws .bulkbar').count()===0&&(await p.innerText('#ws')).includes('بررسی گروهی ندارد'),'HR: no bulk review');
// reports
await p.goto(BASE+'?view=rep&report=invoices_register');await p.waitForTimeout(300);t=await p.innerText('#ws');ok(t.includes('مجوز')&&t.includes('نسخه ۱')&&t.includes('۶۰ تا ۱۲۰')&&t.includes('اعتبارسنجی'),'report: meta, bounded rows, label validation, export conditional');
ok(await p.locator('.rep-head button[disabled]:has-text("خروجی")').count()===1,'export disabled (permission unverified)');
// customer behaviour states
for(const s of ['partial','residual','loaded','error','noresult','empty','stale','loading']){await p.goto(BASE+'?view=cust&cs='+s);await p.waitForTimeout(250);t=await p.innerText('#ws');ok(t.length>200,'cust state '+s);
 if(s==='error')ok(t.includes('بدون فعالیت')&&t.includes('نیست'),'cust error ≠ no activity');
 if(s==='residual')ok(t.includes('نشانگر بارگذاری هنوز فعال')&&t.includes('موفقیت کامل نیست'),'cust residual loading not success');
 if(s==='empty')ok(t.includes('مجاز')&&t.includes('تعریف‌شده'),'cust authorized-empty is scoped');}
ok((await p.innerText('#ws')).length>0&&true,'ok');
await p.goto(BASE+'?view=cust&cs=partial');await p.waitForTimeout(250);ok((await p.innerText('#ws')).includes('دو جمعیت متفاوت'),'cust populations differ — not defect claimed');
// wallet pure read
await p.goto(BASE+'?view=wallet');await p.waitForTimeout(300);ok((await p.innerText('#ws')).includes('خواندن محض'),'wallet pure-read');
// no forbidden authority anywhere
for(const v of ['ov','dist','perf','inv','arch','xnum','hr','rep','cust','wallet']){await p.goto(BASE+'?view='+v);await p.waitForTimeout(220);t=await p.innerText('#ws');ok(!/تأیید پرداخت|ثبت بازپرداخت|اعمال نهایی را انجام/.test(t.replace(/اعمال نهایی را (منابع|با)/g,'')),'no finance/HR-final control on '+v);}
// sims
for(const s of ['loading','empty','noresult','tableError','pageError','unauthorized','stale','incomplete','unitfail','unknownhist','offline']){for(const v of ['ov','perf','arch']){await p.goto(BASE+'?view='+v+'&sim='+s);await p.waitForTimeout(250);}}
ok(p.errs.length===0,'no JS errors in B: '+p.errs.slice(0,3));
await p.goto(BASE+'?view=ov&sim=unitfail');await p.waitForTimeout(300);t=await p.innerText('#ws');ok(t.includes('دریافت نشد')&&t.includes('بقیه واحدها'),'unit failure: partial banner, others loaded');
await p.goto(BASE+'?view=perf&sim=stale');await p.waitForTimeout(300);ok((await p.innerText('#ws')).includes('قدیمی'),'stale unit labelled');
await p.goto(BASE+'?view=ov');await p.keyboard.press('Control+k');await p.waitForTimeout(300);ok(await p.locator('#palette:not([hidden])').count()===1,'palette opens');
// ---- PASS C (matrix)
const views=['ov','dist','perf','inv','arch','xnum','hr','rep','cust','wallet'];
let cbad=0;
for(const [w,h] of [[1920,1080],[1366,768],[1024,768],[768,1024],[390,844]]){
 for(const [th,dn,fc] of [['light','comfortable',0],['dim','compact',0],['dark','comfortable',1]]){
  const q=await mk(b,w,h);
  for(const v of views){await q.goto(BASE+'?view='+v+'&theme='+th+'&density='+dn+(fc?'&focus=1':''));await q.waitForTimeout(220);
   const r=await q.evaluate(()=>({o:document.documentElement.scrollWidth>innerWidth+1}));if(r.o){cbad++;console.log('OVERFLOW',w,th,v)}
   if(w===1366||w===390){const c=await q.evaluate(contrast);if(c.length){cbad++;console.log('CONTRAST',w,th,v,JSON.stringify(c))}}}
  if(q.errs.length){cbad++;console.log('ERR',w,th,q.errs[0])}
  await q.context().close();}}
ok(cbad===0,'pass C matrix clean ('+cbad+' issues)');
console.log('FAILS',fail.length);await b.close()})();
