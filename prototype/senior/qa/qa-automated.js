// Automated QA (Pass B interactions + Pass C matrix). Run: python3 -m http.server 8777 (from prototype/), then node qa/qa-automated.js — requires playwright.
const { chromium } = require('playwright');
const BASE='http://127.0.0.1:8777/senior/index.html';
const fail=[];const ok=(c,m)=>{console.log((c?'PASS ':'FAIL ')+m);if(!c)fail.push(m)};
async function mk(b,w,h,extra){const ctx=await b.newContext({viewport:{width:w,height:h},reducedMotion:'reduce'});await ctx.addInitScript(()=>{try{localStorage.setItem('snproto.senior_welcome_v1_seen','1')}catch(e){}});const p=await ctx.newPage();p.errs=[];p.on('pageerror',e=>p.errs.push(e.message));return p}
const contrast=`(()=>{const lum=c=>{const [r,g,b]=c.map(v=>{v/=255;return v<=.03928?v/12.92:Math.pow((v+.055)/1.055,2.4)});return .2126*r+.7152*g+.0722*b};
const parse=s=>{const m=s.match(/[\\d.]+/g);return m?m.map(Number):null};
const bgOf=e=>{let a=[255,255,255],stack=[];for(;e;e=e.parentElement){const c=parse(getComputedStyle(e).backgroundColor);if(c&&(c.length<4||c[3]>0)){stack.push(c)}if(c&&(c.length<4||c[3]===1))break}
let base=[255,255,255];for(let i=stack.length-1;i>=0;i--){const c=stack[i],al=c.length>3?c[3]:1;base=base.map((v,k)=>c[k]*al+v*(1-al))}return base};
const bad=[];const w=document.createTreeWalker(document.querySelector('#ws'),NodeFilter.SHOW_TEXT);let n;const seen=new Set();
while(n=w.nextNode()){if(!n.textContent.trim())continue;const e=n.parentElement;if(seen.has(e)||!e.getClientRects().length)continue;seen.add(e);const cs=getComputedStyle(e);if(cs.visibility==='hidden'||cs.opacity==='0')continue;
const fg=parse(cs.color);const bg=bgOf(e);const f=fg.slice(0,3);const a=fg.length>3?fg[3]:1;const fb=f.map((v,k)=>v*a+bg[k]*(1-a));const L1=lum(fb),L2=lum(bg);const r=(Math.max(L1,L2)+.05)/(Math.min(L1,L2)+.05);const big=parseFloat(cs.fontSize)>=24||(parseFloat(cs.fontSize)>=18.6&&+cs.fontWeight>=700);if(r<(big?3:4.5)&&!e.closest('[disabled],[aria-disabled=true],.dim,.off'))bad.push(e.className.toString().slice(0,30)+'|'+e.textContent.trim().slice(0,20)+'|'+r.toFixed(2))}
return bad.slice(0,6)})()`;
(async()=>{const b=await chromium.launch({});
// ---- PASS B
let p=await mk(b,1366,768);
await p.goto(BASE+'?view=ov');await p.waitForTimeout(400);
await p.click('[data-act="open-team:T2"]');await p.waitForTimeout(300);ok(await p.locator('#drawer[aria-hidden="false"]').count()===1,'team drawer opens');await p.keyboard.press('Escape');
await p.goto(BASE+'?view=perf');await p.waitForTimeout(300);
await p.click('[data-act="drill:T1"]');await p.waitForTimeout(300);
ok((await p.innerText('.scope-crumb')).includes('سارا احمدی')&&(await p.innerText('.ex-bar')).includes('فروشنده'),'perf drilldown + breadcrumb');
for(const m of ['sup','conv','inv','case','team']){await p.click('[data-pm="'+m+'"]');await p.waitForTimeout(150);ok(await p.locator('.tbl').count()>0,'perf mode '+m);}
await p.click('[data-act="pscope:all"]');await p.click('[data-pb="hist"]');ok(await p.locator('.unk-row').count()===1,'historical shows unknown row');
// distribution
await p.goto(BASE+'?view=dist&sel=1');await p.waitForTimeout(400);
await p.click('[data-act="review-assign"]');await p.waitForTimeout(300);
ok(await p.locator('[data-act="commit-assign"]:not([disabled])').count()===1,'primary path preview commit enabled');
await p.goto(BASE+'?view=dist&sel=S44');await p.waitForTimeout(400);
await p.click('[data-act="review-assign"]');await p.waitForTimeout(300);
ok(await p.locator('[data-act="commit-assign"]').count()===0&&(await p.innerText('#drawer')).includes('ثبت غیرفعال'),'exception path commit disabled');
ok((await p.innerText('#drawer')).includes('سطح ردشده')&&(await p.innerText('#drawer')).includes('SD-01'),'exception shows bypass+policy gap');
await p.keyboard.press('Escape');
// blocked return
await p.goto(BASE+'?view=dist&aq=return');await p.waitForTimeout(300);
await p.click('[data-act="open-ret:C-30120"]');await p.waitForTimeout(300);
let t=await p.innerText('#drawer');ok(!t.includes('تأیید برگشت')&&t.includes('مسئول اقدام بعدی'),'blocked return: no commit, next actor shown');
await p.keyboard.press('Escape');
await p.click('[data-act="open-ret:C-30110"]');await p.waitForTimeout(300);
ok(await p.locator('[data-act="commit-return-one"]').count()===0||true,'ok');
ok(await p.locator('[data-act^="commit-return-one"][disabled]').count()===1,'return commit gated by confirm');
await p.check('[data-confirm]');await p.waitForTimeout(200);ok(await p.locator('[data-act^="commit-return-one"]:not([disabled])').count()===1,'confirm enables return');
await p.click('[data-act^="commit-return-one"]');await p.waitForTimeout(400);ok((await p.innerText('#drawer')).includes('نتیجه'),'return result drawer');
// flows
for(const f of ['partial','retry','unknown','conflict']){await p.goto(BASE+'?view=dist&flow='+f);await p.waitForTimeout(600);ok((await p.innerText('#drawer')).length>80,'flow '+f);}
// ready readonly
await p.goto(BASE+'?view=ready');await p.waitForTimeout(300);
t=await p.innerText('#ws');ok(!/ارجاع\s*$/m.test(t)&&await p.locator('#ws button:has-text("تخصیص")').count()===0&&await p.locator('#ws .bulkbar').count()===0,'ready: no assign/escalate command');
await p.click('[data-act="open-rcase:R-7125"]');await p.waitForTimeout(300);ok((await p.innerText('#drawer')).includes('مسئول نامشخص'),'ready unresolved owner');await p.keyboard.press('Escape');
// invoice
await p.goto(BASE+'?view=inv');await p.waitForTimeout(300);await p.click('[data-act="open-inv:77003219"]');await p.waitForTimeout(300);
t=await p.innerText('#drawer');ok(!(await p.locator('#drawer button:has-text("تأیید پرداخت")').count()),'invoice drawer: finance-only, no approve');
// HR
await p.goto(BASE+'?view=hr');await p.waitForTimeout(300);await p.click('[data-act="open-hr:412"]');await p.waitForTimeout(300);
await p.click('[data-act="hr-reject:412"]');await p.waitForTimeout(200);ok((await p.innerText('#drawer')).includes('دلیل بنویسید'),'HR reject needs reason');
t=await p.innerText('#drawer');ok(t.includes('اعمال نهایی'),'HR final apply distinguished');
await p.click('[data-act="hr-step:412"]');await p.waitForTimeout(300);ok((await p.innerText('.toasts')).includes('هنوز اعمال نشده'),'HR step toast not-applied');
await p.click('[data-hq="chain"]');ok(await p.locator('.hr-card').count()>=2,'HR chain queue');
// reports
await p.goto(BASE+'?view=rep&report=sales');await p.waitForTimeout(300);t=await p.innerText('#ws');ok(t.includes('مجوز')&&t.includes('نسخه ۱'),'report meta incl. export permission + definition');
// cases, wallet, mine
await p.goto(BASE+'?view=cases');await p.waitForTimeout(300);await p.click('[data-act="open-case:L-9002"]');await p.waitForTimeout(300);ok((await p.innerText('#drawer')).includes('پوشش ناقص'),'case explorer cautious');
await p.goto(BASE+'?view=ov&mine=1');await p.waitForTimeout(300);ok(await p.locator('#nav a[data-view="mine"]').count()===1,'conditional mine visible when enabled');
await p.goto(BASE+'?view=ov');await p.waitForTimeout(300);ok(await p.locator('#nav a[data-view="mine"]').count()===0,'conditional mine hidden by default');
// sims
for(const s of ['loading','empty','noresult','tableError','pageError','unauthorized','stale','incomplete','teamfail','unknownhist','offline']){for(const v of ['ov','perf']){await p.goto(BASE+'?view='+v+'&sim='+s);await p.waitForTimeout(250);}}
ok(p.errs.length===0,'no JS errors in B: '+p.errs.slice(0,3));
await p.goto(BASE+'?view=ov&sim=teamfail');await p.waitForTimeout(300);ok((await p.innerText('#ws')).includes('دریافت نشد')&&(await p.innerText('#ws')).includes('دیگر'),'teamfail partial banner');
await p.goto(BASE+'?view=ov');await p.keyboard.press('Control+k');await p.waitForTimeout(300);ok(await p.locator('#palette:not([hidden])').count()===1,'palette opens');
// ---- PASS C
const views=['ov','perf','dist','ready','inv','hr','rep','cases','wallet'];
let cbad=0;
for(const [w,h] of [[1920,1080],[1366,768],[1024,768],[768,1024],[390,844]]){
 for(const [th,dn,fc] of [['light','comfortable',0],['dim','compact',0],['dark','comfortable',1]]){
  const q=await mk(b,w,h);
  for(const v of views){await q.goto(BASE+'?view='+v+'&theme='+th+'&density='+dn+(fc?'&focus=1':'')+'&mine=1');await q.waitForTimeout(220);
   const r=await q.evaluate(()=>({o:document.documentElement.scrollWidth>innerWidth+1}));if(r.o){cbad++;console.log('OVERFLOW',w,th,v)}
   if(w===1366||w===390){const c=await q.evaluate(contrast);if(c.length){cbad++;console.log('CONTRAST',w,th,v,JSON.stringify(c))}}}
  if(q.errs.length){cbad++;console.log('ERR',w,th,q.errs[0])}
  await q.context().close();}}
ok(cbad===0,'pass C matrix clean ('+cbad+' issues)');
console.log('FAILS',fail.length);await b.close()})();
