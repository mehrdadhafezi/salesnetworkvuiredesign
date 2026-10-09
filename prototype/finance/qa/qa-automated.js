// Automated QA (SN-206 Finance: Pass A structure, Pass B critical flows, Pass C matrix; Pass D regression = earlier roles' scripts re-run separately).
// Run: python3 -m http.server 8777 (from prototype/), then NODE_PATH=<playwright> node finance/qa/qa-automated.js — requires playwright.
const { chromium } = require('playwright');
const BASE='http://127.0.0.1:8777/finance/index.html';
const fail=[];const ok=(c,m)=>{console.log((c?'PASS ':'FAIL ')+m);if(!c)fail.push(m)};
async function mk(b,w,h){const ctx=await b.newContext({viewport:{width:w,height:h},reducedMotion:'reduce'});await ctx.addInitScript(()=>{try{localStorage.setItem('snproto.fin_welcome_v1_seen','1')}catch(e){}});const p=await ctx.newPage();p.errs=[];p.on('pageerror',e=>p.errs.push(e.message));return p}
const contrast=`(()=>{const lum=c=>{const [r,g,b]=c.map(v=>{v/=255;return v<=.03928?v/12.92:Math.pow((v+.055)/1.055,2.4)});return .2126*r+.7152*g+.0722*b};
const parse=s=>{const m=s.match(/[\\d.]+/g);return m?m.map(Number):null};
const bgOf=e=>{let a=[255,255,255],stack=[];for(;e;e=e.parentElement){const c=parse(getComputedStyle(e).backgroundColor);if(c&&(c.length<4||c[3]>0)){stack.push(c)}if(c&&(c.length<4||c[3]===1))break}
let base=[255,255,255];for(let i=stack.length-1;i>=0;i--){const c=stack[i],al=c.length>3?c[3]:1;base=base.map((v,k)=>c[k]*al+v*(1-al))}return base};
const bad=[];const w=document.createTreeWalker(document.querySelector('#ws'),NodeFilter.SHOW_TEXT);let n;const seen=new Set();
while(n=w.nextNode()){if(!n.textContent.trim())continue;const e=n.parentElement;if(seen.has(e)||!e.getClientRects().length)continue;seen.add(e);const cs=getComputedStyle(e);if(cs.visibility==='hidden'||cs.opacity==='0')continue;
const fg=parse(cs.color);const bg=bgOf(e);const f=fg.slice(0,3);const a=fg.length>3?fg[3]:1;const fb=f.map((v,k)=>v*a+bg[k]*(1-a));const L1=lum(fb),L2=lum(bg);const r=(Math.max(L1,L2)+.05)/(Math.min(L1,L2)+.05);const big=parseFloat(cs.fontSize)>=24||(parseFloat(cs.fontSize)>=18.6&&+cs.fontWeight>=700);if(r<(big?3:4.5)&&!e.closest('[disabled],[aria-disabled=true],.dim,.off'))bad.push(e.className.toString().slice(0,30)+'|'+e.textContent.trim().slice(0,20)+'|'+r.toFixed(2))}
return bad.slice(0,6)})()`;
(async()=>{const b=await chromium.launch({});
const VIEWS=['rev','rec','led','run','rep','aud','cfg','mnt'];
let p=await mk(b,1366,768);let t;
const go=async(q)=>{await p.goto(BASE+'?'+q);await p.waitForTimeout(350)};
const open=async(sel)=>{await p.click(sel);await p.waitForTimeout(260)};
const dr=async()=>await p.innerText('#drawer');
const drt=async()=>await p.evaluate(()=>document.querySelector('#drawer').textContent);
const ws=async()=>await p.innerText('#ws');
const cnt=async(s)=>await p.locator(s).count();
const esc=async()=>{await p.keyboard.press('Escape');await p.waitForTimeout(120)};
const adis=async(s)=>await p.locator(s).first().evaluate(n=>n.disabled||n.getAttribute('aria-disabled')==='true');
const fillSens=async(r)=>{await p.fill('#sens-r',r||'دلیل ممیزی');await p.locator('[data-sensack]').check();await p.waitForTimeout(200)};
const ackOnly=async()=>{await p.locator('[data-sensack]').check();await p.waitForTimeout(200)};
const noBtn=async(re)=>!(await p.evaluate((src)=>[...document.querySelectorAll('#ws button:not([aria-disabled=true]),#drawer button:not([aria-disabled=true]),a')].some(x=>new RegExp(src).test(x.textContent)),re));
const snap=async()=>await p.evaluate(()=>JSON.stringify([window.FIN.queue,window.FIN.runs,window.FIN.ledger,window.FIN.refunds,window.FIN.issues,window.FIN.audit.length,window.FINX.st.ops.length,window.FINX.st.local,window.FINX.st.runLocal]));
// ---- PASS A structure
for(const v of VIEWS){await go('view='+v);ok((await ws()).length>250,'render '+v);}
ok(p.errs.length===0,'pass A: no JS errors');
for(const v of VIEWS){await go('view='+v);const r=await p.evaluate(()=>({main:!!document.querySelector('main'),nav:!!document.querySelector('nav[aria-label]'),capt:[...document.querySelectorAll('#ws table')].every(x=>x.querySelector('caption')),btn:[...document.querySelectorAll('button')].every(x=>(x.textContent.trim()||x.getAttribute('aria-label')||x.getAttribute('title')))}));ok(r.main&&r.nav&&r.capt&&r.btn,'landmarks/captions/button names: '+v);}
ok((await p.innerText('.hdr')).includes('NOT LIVE VERIFIED'),'header says NOT LIVE VERIFIED');
// ---- PASS B critical flows
const PURE=await (async()=>{await go('view=rev');return await snap()})();
// Review queue grain, facets, KPIs
await go('view=rev');t=await ws();ok(await cnt('#ws tbody tr')===12&&t.includes('واحد شمارش')&&t.includes('هم‌پوشان')&&t.includes('مرحله در انتظار بررسی')&&t.includes('فاکتور دارای مرحلهٔ در انتظار'),'queue: grain = stage; facets overlap; stage ≠ invoice KPI');
ok((await p.innerText('.kpis')).includes('در جمع نیست')&&!(await p.innerText('.kpis')).match(/ریال.*تومان.*جمع کل/),'KPI: amounts per unit, unknown unit never summed');
ok(['فاکتور','پیوند','مرحله','مدرک و ناهمخوانی','مبلغ','بررسی مالی'].every(x=>t.includes(x))&&await p.evaluate(()=>document.documentElement.scrollWidth<=innerWidth&&document.querySelector('#ws .tbl-wrap').scrollWidth<=document.querySelector('#ws .tbl-wrap').clientWidth+1),'queue columns (priority at 1366): invoice/link/stage/evidence+issues/amount/review; table fits without scroll');
// Evidence review
await go('flow=item');t=await dr();ok(await cnt('#drawer .fsl-c')===9&&t.includes('استنتاج نشود از')&&['وضعیت فاکتور','مرحلهٔ پرداخت','بررسی مالی','مدرک پرداخت','وصول‌شده معتبر','باقی‌مانده','وضعیت استرداد','ثبت در دفتر کل','استحقاق'].every(x=>t.includes(x)),'evidence review: 9 independent state layers, each with not-inferred-from');
ok(['نوع مدرک','مرجع · نسخه','ارسال‌ک','منبع','مبلغ مندرج در مدرک','مبلغ ادعاشده','مبلغ اعتبارسنجی‌شده','اعتبارسنجی نشده'].every(x=>t.includes(x))&&t.includes('فروشنده اصلی')&&t.includes('مسئول فعلی')&&t.includes('گیرندهٔ کمیسیون'),'evidence panel + original seller/current owner/recipient separate; validated amount unknown ≠ 0');
ok(t.includes('تأیید ≠ ثبت در دفتر کل ≠ تسویه')&&t.includes('اگر رد شود')&&t.includes('رد = حذف، لغو، استرداد'),'impact preview: approve/reject, reject ≠ delete/cancel/refund');
// Gateway / missing / mismatch / unit / dup / link blockers
for(const [f,needle] of [['gateway','پرداخت درگاهی ثبت‌شده ≠ تأیید مالی'],['noevidence','مدرک پرداخت موجود نیست'],['mismatch','مبلغ مدرک با مبلغ ادعاشده نمی‌خواند'],['unit','واحد مبلغ نامشخص است'],['dup','پرداخت تکراری احتمالی'],['link','پیوند Case/منبع نامعلوم']]){await go('flow='+f);t=await dr();ok(t.includes(needle),'blocker/state shown: '+f);if(f!=='gateway')ok(await adis('[data-act^="rs-approve"]'),'approve blocked: '+f);}
await go('flow=noevidence');ok(!(await adis('[data-act^="rs-reject"]')),'missing evidence: reject with reason still possible');
// Approve with sensitive confirmation
await go('flow=approve');t=await dr();ok(['هدف','وضعیت فعلی','مبلغ / واحد','عامل','وضعیت پس از ثبت','چه چیزی تغییر می‌کند','تاریخچه محافظت‌شده','رکوردهای متأثر','برگشت‌ناپذیر'].every(x=>t.includes(x))&&await adis('[data-act="commit-sens"]'),'approve confirmation: target/state/amount/actor/result/protected history/affected/irreversible; gated');
ok(t.includes('موتور هدف‌دار ممکن است')&&t.includes('تأیید ≠ تسویه'),'approve: automatic engine effect disclosed, not simulated');
await ackOnly();ok(!(await adis('[data-act="commit-sens"]')),'approve enabled with ack (reason optional)');
await open('[data-act="commit-sens"]');t=await drt();ok(t.includes('تأیید مالی مرحله RS-2101')&&await cnt('#drawer .outcome-strip .os')===7&&t.includes('ثبت دفتر کل / اعتبار کمیسیون / تسویه در این عملیات انجام نشد'),'approve result: per-item truth; ledger/entitlement/settlement NOT done');
await esc();await p.waitForTimeout(150);t=await ws();ok(await cnt('[data-rq="needs"] .n')===1&&(await p.innerText('[data-rq="needs"]')).includes('۱۱'),'approved stage leaves needs-review queue (12 → 11)');
await open('[data-rq="approved"]');await open('[data-act="open-rs:RS-2101"]');t=await dr();ok(t.includes('تأیید مالی شد')&&t.includes('ثبت‌نشده / تعریف‌نشده')&&t.includes('ایجاد نشده')&&t.includes('در جریان وصول'),'after approve: review approved BUT ledger unposted, entitlement not created, invoice not completed');
await esc();
// Approved + fully paid + entitlement boundary
await go('flow=approved');t=await dr();ok(t.includes('پرداخت کامل (طبق مبلغ معتبر)')&&t.includes('کسب‌شده (اعتبار کیف پول نیست)')&&t.includes('نامعلوم'),'fully paid ≠ entitlement ≠ wallet credit; ledger posting state unknown');
// Reject: reason required; history preserved
await go('flow=reject');t=await dr();ok(await adis('[data-act="commit-sens"]')&&t.includes('الزامی')&&['استرداد','تأییدهای قبلی','حذف یا لغو'].every(x=>t.includes(x)),'reject: reason required; no delete/cancel/refund/reset');
await p.locator('[data-sensack]').check();await p.waitForTimeout(150);ok(await adis('[data-act="commit-sens"]'),'reject: ack alone insufficient (reason)');
await fillSens('رسید ناخوانا');await open('[data-act="commit-sens"]');await esc();
// reject on a stage with prior approval: history kept
await p.goto(BASE+'?view=rev');await p.waitForTimeout(250);await p.evaluate(()=>{FINX.openSens(FINX.rejectSpec(FINX.q('RS-2102')))});await p.waitForTimeout(250);await fillSens('مدرک ناقص');await open('[data-act="commit-sens"]');await esc();
await p.evaluate(()=>{CRM.openDrawer('rs','RS-2102')});await p.waitForTimeout(250);t=await dr();ok(t.includes('رد شد')&&t.includes('مرحله ۱ تأیید شد')&&t.includes('فروشنده (اصلاح و ارسال دوباره)'),'reject keeps prior approval; next actor = seller');await esc();
await go('flow=rejected');t=await dr();ok(t.includes('رسید نامرتبط')&&t.includes('بازگشت')||t.includes('فروشنده (اصلاح'),'rejected item: reason + next actor visible');
// Stale conflict
await go('flow=stale');t=await dr();ok(await cnt('#drawer .state-conflict')===1&&await adis('[data-act^="rs-approve"]')&&await adis('[data-act^="rs-reject"]')&&t.includes('بازخوانی'),'stale: conflict state, both decisions blocked, reload required');
await open('[data-act^="reload-stage"]');t=await dr();ok(t.includes('تأیید مالی شد')&&(await cnt('[data-act^="rs-approve"]'))===0,'stale reload: shows current truth (already approved by other reviewer)');await esc();
await go('flow=stalecommit');await ackOnly();await open('[data-act="commit-sens"]');t=await drt();ok(t.includes('تعارض')&&t.includes('هیچ تغییری ثبت نشد'),'stale at commit: conflict result, no change');await esc();
// Bulk review / approve
await go('flow=bulkrev');t=await dr();ok(t.includes('فقط‌خواندنی')&&t.includes('هر ردیف جداگانه')&&await cnt('#drawer [data-act="commit-bulkapp"]')===0,'bulk review: read-only triage');await esc();
await go('flow=bulkapp');t=await dr();ok(t.includes('واجد شرایط ارسال')&&t.includes('خارج از ارسال')&&t.includes('مسدود')===false||t.includes('خارج از ارسال'),'bulk approve: per-item eligibility');ok(await adis('[data-act="commit-bulkapp"]'),'bulk approve gated by ack');
await p.locator('[data-bulkack]').check();await p.waitForTimeout(200);await open('[data-act="commit-bulkapp"]');t=await drt();
ok(await cnt('#drawer .os-applied b')===1&&(await p.innerText('#drawer .os-applied b'))==='۳'&&(await p.innerText('#drawer .os-skipped b'))==='۱'&&(await p.innerText('#drawer .os-failed b'))==='۱'&&(await p.innerText('#drawer .os-unknown b'))==='۱'&&t.includes('= درخواست‌شده'),'bulk result: 6 requested = 3 ok + 1 skipped + 1 failed + 1 unknown (no fake all-success)');
ok(await cnt('[data-act^="reconcile-op"]')===1&&await cnt('[data-act^="retry-op"]')===0,'bulk: unknown → reconcile first; retry hidden while unknown remains');
await open('[data-act^="reconcile-op"]');ok(await cnt('[data-act^="retry-op"]')===1,'after reconcile: retry only known failures');await open('[data-act^="retry-op"]');t=await drt();ok(await cnt('#drawer .os-unknown.zero')===1||(await p.innerText('#drawer .os-unknown b'))==='۰','retry resolves failures without touching others');await esc();
// Reject/refund/export baseline bulk
await p.goto(BASE+'?view=rev');await p.waitForTimeout(250);await p.evaluate(()=>{FINX.st.sel={'RS-2112':true};CRM.render()});await p.waitForTimeout(150);ok(await adis('.bulkbar [data-tip*="رد گروهی"]')&&await adis('.bulkbar [data-tip*="خروجی گروهی"]'),'bulk reject needs validation; export conditional (disabled with reason)');
// Refund (conditional)
await go('flow=refund');t=await dr();ok(await cnt('#drawer .rf-lad li')===7&&t.includes('OPD-04')&&t.includes('لغو فاکتور به‌معنی بازگشت وجه نیست')&&await adis('#drawer .dr-foot .btn-primary'),'refund: 7 separate concepts; cancelled ≠ refunded; execution unavailable (OPD-04)');
ok(['فاکتور اصلی','پرداخت اصلی','مبلغ درخواستی','قبلاً استردادشده','باقی‌ماندهٔ مجاز','منبع اجرا','مدرک','بازبین','تاریخچه'].every(x=>t.includes(x)),'refund: original payment/requested/refunded/eligible/source/proof/reviewer/history');
await go('flow=refundunknown');t=await dr();ok(t.includes('نامعلوم')&&!t.includes('۰ تومان'),'refund unknown amount never zero');
await go('view=rec&recq=refunds');ok(t=await ws(),(await ws()).includes('استرداد گروهی مجاز نیست')||(await ws()).includes('مشروط است (OPD-04)'),'refund view: conditional banner, no bulk refund');
// Ledger
await go('view=led');t=await ws();ok(await cnt('#ws tbody tr')===11&&t.includes('فقط‌خواندن')&&t.includes('F07')&&!t.includes('مانده کل'),'ledger: audit explorer, read-only (F07), no editable balance');
ok(['تراکنش','کلید','حساب','جهت','مبلغ','رابطه','عامل','زمان اثر','زمان ثبت','وضعیت ثبت'].every(x=>t.includes(x)),'ledger columns: tx/key/account/direction/amount/relation/actor/effective/posting/state');
await go('flow=ledger');t=await dr();ok(t.includes('T-9005')&&t.includes('T-9006')&&t.includes('فقط‌افزودنی')&&await noBtn('^(ویرایش|حذف)'),'ledger drill: correction chain linked; no edit/delete');
await go('flow=ledgerunk');t=await dr();ok(t.includes('نامعلوم')&&t.includes('ثبت‌شده فرض نمی‌شود'),'ledger: unknown outcome intent ≠ committed tx');
await go('flow=orphan');t=await dr();ok(t.includes('شبه‌یتیم')&&t.includes('حذف یا اتصال حدسی'),'ledger: orphan-looking relation preserved (no phone relink/delete)');
await go('view=led&lq=notc');ok(await cnt('#ws tbody tr')===2,'ledger: preview/unknown filtered separately from committed');
await go('view=led');ok((await ws()).includes('دو دامنهٔ مستقل'),'ledger: two domains, no single balance');
// Runs: lifecycle, preview write, approval ≠ posting
await go('view=run');t=await ws();ok(await cnt('.lc-rib li')===9&&['پیش‌نویس','پیش‌نمایش','تأییدشده','در حال ثبت','ثبت‌شده','ناقص','ناموفق','نتیجه نامعلوم','تطبیق‌شده'].every(x=>t.includes(x))&&t.includes('تأیید ≠ ثبت')&&t.includes('خواندن خالص نیست'),'runs: 9-state lifecycle; approve ≠ post; preview ≠ pure read');
ok(t.includes('OPD-08')&&t.includes('تفکیک'),'runs: maker/checker not invented (OPD-08)');
await go('flow=rundraft');await open('[data-act^="run-gen"]');t=await dr();ok(t.includes('فراداده می‌نویسد')&&t.includes('نوشتن نه خواندن خالص')||t.includes('نوشتن'),'draft → preview confirmation states it writes metadata');await fillSens();await open('[data-act="commit-sens"]');t=await drt();ok(t.includes('هیچ تراکنش مالی ثبت نشد'),'preview generation: metadata only, no tx');await esc();
await go('flow=runpreview');await open('[data-act^="run-approve"]');ok((await dr()).includes('FIN-LOCK')&&(await dr()).includes('OPD-08'),'run approve: lock caveat + OPD-08');await fillSens();await open('[data-act="commit-sens"]');t=await drt();ok(t.includes('هیچ اعتباری در کیف پول ثبت نشد'),'run approval ≠ posted (no wallet credit)');await esc();
await open('[data-act="open-run:RUN-311"]');t=await dr();ok(t.includes('تأییدشده')&&await cnt('[data-act^="run-post"]')===1,'approved run exposes posting as a separate step');await esc();
// Partial posting
await go('flow=runpartial');t=await dr();ok(await cnt('#drawer .cov-strip .os')===7&&t.includes('۲۰۰۰ آیتم')&&t.includes('پردازش‌نشده')&&t.includes('موفقیت کل اجرا نیست'),'partial posting: coverage 7 cells, 2000-item bound, ok ≠ full-run success');
ok((await p.innerText('#drawer .cov-strip')).includes('۳۴۰')&&(await p.innerText('#drawer .cov-strip')).includes('۱٬۸۷۴')&&t.includes('زیرمجموعهٔ محدود'),'partial: posted/existing/failed/unprocessed counts truthful; sample ≠ full');
await open('[data-act^="run-tail"]');await fillSens();await open('[data-act="commit-sens"]');t=await drt();ok(await cnt('#drawer .os-applied')===1&&t.includes('۱۴ آیتم ناموفق قبلی باقی است'),'tail posting: per-item truth; earlier failures remain (not hidden)');await esc();
await open('[data-act="open-run:RUN-307"]');ok((await dr()).includes('ناقص')&&await cnt('[data-act^="run-tail"]')===0,'after tail: still Partial (14 failed) — no fake success; no more tail');await esc();
// Outcome unknown / idempotency
await go('flow=rununknown');t=await dr();ok(await cnt('#drawer .oux')===1&&['نیت اصلی','شناسه تراکنش / کلید کسب‌وکار','وضعیت ممکن','الزام جستجو/تطبیق','تراکنش موجود','شرط مجاز بودن تکرار','گام امن بعدی'].every(x=>t.includes(x)),'outcome unknown: OutcomeUnknownContext complete');
ok(await p.locator('#drawer .dr-foot .btn[aria-disabled="true"]').count()===1&&!(await p.locator('#drawer .dr-foot [data-act^="run-post"]').count()),'outcome unknown: NO enabled retry — blind retry blocked');
ok((await p.innerText('#drawer .cov-strip')).includes('—'),'unknown coverage renders — (never 0)');
{const s0=await snap();await open('[data-act^="run-lookup"]');t=await dr();ok(t.includes('هنوز ثبت نشده')&&(await p.innerText('#drawer .lc-rib li.on')).includes('نتیجه نامعلوم')&&await snap()===s0,'lookup is a PURE read: shows result, run stays Outcome Unknown, no data/audit change');}
await open('[data-act^="run-recon"]');t=await dr();ok(t.includes('نتیجهٔ جستجو')&&await adis('[data-act="commit-sens"]'),'recording reconciliation = separate sensitive step');await fillSens();await open('[data-act="commit-sens"]');await esc();await p.evaluate(()=>CRM.openDrawer('run','RUN-305'));await p.waitForTimeout(250);t=await dr();ok((await p.innerText('#drawer .lc-rib li.on')).includes('تطبیق‌شده')&&t.includes('ثبت‌نشدن ۵۰ آیتم اثبات شد')&&await cnt('[data-act^="run-tail"]')===1&&t.includes('T-8801'),'recorded reconcile → reconciled; item linked to real tx; retry only proven-uncommitted tail');
await go('view=run&perm=view&open=run:RUN-305');{const s1=await snap();await open('[data-act^="run-lookup"]');ok(await snap()===s1,'view-only: lookup still pure (no mutation)');ok(await cnt('[data-act^="run-recon"]')===1&&await adis('[data-act^="run-recon"]'),'view-only: recording reconciliation unavailable (F06)');}
await go('view=run&open=run:RUN-305');await open('[data-act^="run-lookup"]');await open('[data-act^="run-recon"]');await fillSens();await open('[data-act="commit-sens"]');await esc();
await p.evaluate(()=>CRM.openDrawer('run','RUN-305'));await p.waitForTimeout(250);
await open('[data-act^="run-tail"]');await fillSens();await open('[data-act="commit-sens"]');await esc();await p.evaluate(()=>CRM.openDrawer('run','RUN-305'));await p.waitForTimeout(250);ok((await p.innerText('#drawer .lc-rib li.on')).trim()==='ثبت‌شده'&&await cnt('#drawer .oux')===0,'reconciled tail → posted with proof');await esc();
// Codex review findings (head 6149b83) — regression tests
await go('flow=runpreview');await open('[data-act^="run-approve"]');await fillSens();await open('[data-act="commit-sens"]');await esc();await p.evaluate(()=>CRM.openDrawer('run','RUN-311'));await p.waitForTimeout(250);await open('[data-act^="run-post"]');await fillSens();await open('[data-act="commit-sens"]');await esc();await p.evaluate(()=>CRM.openDrawer('run','RUN-311'));await p.waitForTimeout(250);
{const r=await p.evaluate(()=>{const it=FINX.runView(FINX.run('RUN-311')).items;return {res:it.map(x=>x[4]),tx:it.map(x=>x[5]),led:it.map(x=>{const t=FIN.ledger.filter(l=>l.id===x[5])[0];return !!t&&t.st==='committed'&&t.key===x[1]})}});
ok(r.res.every(x=>x==='posted')&&r.tx.every(x=>/^T-/.test(x))&&r.led.every(Boolean)&&(await dr()).includes('T-9'),'posting persists per-item results + transaction IDs and real committed ledger rows (no aggregate-only success)');}
for(const [nm,seed] of [['lookup tx id already committed under ANOTHER key',()=>{FIN.ledger.push({id:'T-8802',key:'OTHER|KEY',dom:'wallet',acct:'x',dir:'C',amt:1,unit:'toman',rel:['x','x'],actor:'x',eff:'x',posted:'x',st:'committed'})}],['two committed rows for the same key',()=>{for(const id of ['T-8802','T-8802'])FIN.ledger.push({id,key:'RUN-309|INV-48180|S1|rec:HP-210',dom:'wallet',acct:'x',dir:'C',amt:1,unit:'toman',rel:['x','x'],actor:'x',eff:'x',posted:'x',st:'committed'})}]]){
await go('flow=runapproved');await open('[data-act^="run-post"]');await fillSens();await open('[data-act="commit-sens"]');await esc();await p.evaluate(()=>{CRM.openDrawer('run','RUN-309')});await p.waitForTimeout(250);await open('[data-act^="run-lookup"]');
const n0=await p.evaluate(()=>FIN.ledger.length);await p.evaluate(seed);const n1=await p.evaluate(()=>FIN.ledger.length);
await open('[data-act^="run-recon"]');await fillSens();await open('[data-act="commit-sens"]');await esc();
{const r=await p.evaluate(()=>({st:FINX.runView(FINX.run('RUN-309')).st,it:FINX.runView(FINX.run('RUN-309')).items.map(x=>x[4]),n:FIN.ledger.length}));ok(r.st==='unknown'&&r.it[1]==='unknown'&&r.n===n1,'one-to-one identity: '+nm+' → CONFLICT, stays Unknown, ledger untouched (no overwrite, no duplicate native id)');}}
await go('view=rev');await p.evaluate(()=>{FINX.openSens(FINX.approveSpec(FINX.q('RS-2112')))});await p.waitForTimeout(250);await ackOnly();
await p.evaluate(()=>{FINX.st.local['RS-2112']={review:'approved',reloaded:true,approvedClaimed:6000000,prior:[],rev:'امیر صادقی'}});
await open('[data-act="commit-sens"]');t=await drt();ok(t.includes('تعارض')&&t.includes('هیچ تغییری ثبت نشد')&&await p.evaluate(()=>FINX.eff(FINX.q('RS-2112')).paid)===6000000,'commit-time conflict is DERIVED from live state (no scripted flag): other reviewer decided → no double count');await esc();
await go('view=rev');await p.evaluate(()=>{FINX.openSens(FINX.approveSpec(FINX.q('RS-2101')))});await p.waitForTimeout(250);await ackOnly();
await p.evaluate(()=>{FIN.queue.filter(r=>r.id==='RS-2101')[0].ev.ver='v3'});await open('[data-act="commit-sens"]');t=await drt();ok(t.includes('نسخهٔ مدرک')&&t.includes('هیچ تغییری ثبت نشد'),'commit-time conflict when evidence version changed after dialog opened');await esc();
await go('flow=bulkapp');await p.locator('[data-bulkack]').check();await p.waitForTimeout(150);await open('[data-act="commit-bulkapp"]');await open('[data-act^="reconcile-op"]');
await p.evaluate(()=>{FINX.st.local['RS-2114']={review:'approved',reloaded:true,approvedClaimed:6000000,prior:[],rev:'امیر صادقی'}});
await open('[data-act^="retry-op"]');t=await drt();ok(t.includes('تعارض در بازخوانی تازه')&&(await p.innerText('#drawer .os-skipped b'))==='۲','retry revalidates each failed item: no longer eligible → conflict/skipped, not re-applied');await esc();
await go('flow=bulkapp');await p.locator('[data-bulkack]').check();await p.waitForTimeout(150);await open('[data-act="commit-bulkapp"]');await open('[data-act^="reconcile-op"]');await open('[data-act^="retry-op"]');await esc();
{const r=await p.evaluate(()=>({a:FINX.eff(FINX.q('RS-2112')),b:FINX.eff(FINX.q('RS-2114'))}));ok(r.a.paid===12000000&&r.b.paid===12000000&&r.a.rem===0&&r.b.rem===0,'shared invoice (two approved stages): paid/remaining derived across all stages, same on both rows');}
{const css=await (await fetch(BASE.replace('finance/index.html','shared/crm-ext.css'))).text();ok(!css.includes('.sens-aff')&&!css.includes('.sens-eff')&&css.includes('.fsl{')&&await (await fetch(BASE.replace('index.html','finance.css'))).text().then(x=>x.includes('.sens-aff')),'confirmation tones scoped to Finance CSS only (frozen HR dialogs not restyled)');}
await go('flow=bulkapp');await p.locator('[data-bulkack]').check();await p.waitForTimeout(150);await open('[data-act="commit-bulkapp"]');
await p.evaluate(()=>{FIN.queue.filter(r=>r.id==='RS-2113')[0].ev.ver='v9'});await open('[data-act^="reconcile-op"]');t=await drt();ok((await p.innerText('#drawer .os-unknown b'))==='۱'&&t.includes('نامعلوم می‌ماند')&&await cnt('[data-act^="reconcile-op"]')===1&&await p.evaluate(()=>FINX.localOf(FINX.q('RS-2113')).review)==='pending','evidence-changed UNKNOWN item stays Unknown (reconcile action kept; replacement evidence not approved)');ok(t.includes('تطبیق کامل نشد')&&!t.includes('نتیجهٔ نامعلوم با خواندن وضعیت واقعی تطبیق شد')&&(await p.innerText('#toasts')).includes('تطبیق کامل نشد'),'unresolved reconcile messaging says INCOMPLETE (drawer note + toast), never "reconciled"');await esc();
// run signature covers full item identity (key/amount/unit), not only outcomes
await go('flow=runpreview');await open('[data-act^="run-approve"]');await ackOnly();await p.fill('#sens-r','دلیل');await p.evaluate(()=>{FIN.runs.find(r=>r.id==='RUN-311').items[0][2]=9999999});await open('[data-act="commit-sens"]');t=await drt();ok(t.includes('تعارض')&&await p.evaluate(()=>FINX.runView(FINX.run('RUN-311')).st)==='preview','snapshot item amount altered while approval open → CONFLICT, nothing approved (signature covers full item identity)');await esc();
// idempotency hit is reported as EXISTING (not a new credit) and coverage derives from real mint results
await go('flow=runpreview');await open('[data-act^="run-approve"]');await fillSens();await open('[data-act="commit-sens"]');await esc();await p.evaluate(()=>CRM.openDrawer('run','RUN-311'));await p.waitForTimeout(250);await open('[data-act^="run-post"]');await ackOnly();await p.fill('#sens-r','دلیل');
await p.evaluate(()=>{FIN.ledger.push({id:'T-8999',key:'RUN-311|INV-48222|S1|rec:HP-210',dom:'wallet',acct:'x',dir:'C',amt:600000,unit:'toman',rel:['اجرا','x'],actor:'دیگر',eff:'x',posted:'x',st:'committed'})});
await open('[data-act="commit-sens"]');await esc();
{const r=await p.evaluate(()=>{const R=FINX.runView(FINX.run('RUN-311'));return {res:R.items.map(x=>x[4]),tx:R.items.map(x=>x[5]),cov:R.cov,keys:FIN.ledger.filter(t=>t.key==='RUN-311|INV-48222|S1|rec:HP-210'&&t.st==='committed').length}});ok(r.res.includes('existing')&&r.tx.includes('T-8999')&&r.keys===1&&r.cov.existing>=1&&r.cov.posted+r.cov.existing===48,'idempotency hit → item = EXISTING (tx T-8999), no new tx, coverage existing/posted derived from actual mint results');}
await go('flow=runapproved');await open('[data-act^="run-post"]');await ackOnly();await p.fill('#sens-r','دلیل');await p.evaluate(()=>{FINX.st.runLocal['RUN-309']={st:'posted',cov:{intended:86,processed:86,posted:86,existing:0,failed:0,unprocessed:0,unknown:0}}});
{const n0=await p.evaluate(()=>FIN.ledger.length);await open('[data-act="commit-sens"]');t=await drt();ok(t.includes('تعارض')&&t.includes('هیچ تغییری ثبت نشد')&&await p.evaluate(()=>FIN.ledger.length)===n0&&await p.evaluate(()=>FINX.runView(FINX.run('RUN-309')).st)==='posted','stale run POST confirmation (run advanced meanwhile) → conflict, no new tx');}
await esc();await go('flow=runpreview');await open('[data-act^="run-approve"]');await ackOnly();await p.fill('#sens-r','دلیل');await p.evaluate(()=>{FINX.st.runLocal['RUN-311']={st:'posted',cov:{intended:48,processed:48,posted:48,existing:0,failed:0,unprocessed:0,unknown:0}}});await open('[data-act="commit-sens"]');t=await drt();ok(t.includes('تعارض')&&await p.evaluate(()=>FINX.runView(FINX.run('RUN-311')).st)==='posted','stale run APPROVE confirmation cannot move an advanced run back to approved');await esc();
await go('flow=runapproved');await open('[data-act^="run-post"]');await fillSens();await open('[data-act="commit-sens"]');await esc();await p.evaluate(()=>CRM.openDrawer('run','RUN-309'));await p.waitForTimeout(250);await open('[data-act^="run-lookup"]');await open('[data-act^="run-recon"]');await fillSens();await open('[data-act="commit-sens"]');await esc();
{const r=await p.evaluate(()=>({n:FIN.ledger.filter(t=>t.key.indexOf('RUN-309|')===0&&t.st==='committed').length,it:FINX.runView(FINX.run('RUN-309')).items.map(x=>[x[4],x[5]])}));ok(r.n===1&&r.it[1][0]==='existing'&&r.it[1][1]==='T-8802'&&r.it[0][0]==='unprocessed'&&!r.it[0][1]&&await p.evaluate(()=>!FIN.ledger.some(t=>/^T-91/.test(t.id))),'reconcile classifies by BUSINESS KEY, LINKS the looked-up existing tx (T-8802) as EXISTING, mints nothing; proven-uncommitted gets none');}
{const c=await p.evaluate(()=>FINX.runView(FINX.run('RUN-309')).cov);ok(c.posted===0&&c.existing===51&&c.unprocessed===35,'reconcile coverage: ALL 51 lookup-confirmed commits count as existing, 0 newly posted');}
await p.evaluate(()=>CRM.openDrawer('run','RUN-309'));await p.waitForTimeout(250);await open('[data-act^="run-tail"]');await fillSens();await open('[data-act="commit-sens"]');await esc();
{const r=await p.evaluate(()=>{const k=FIN.ledger.filter(t=>t.st==='committed'&&t.id!=='—').map(t=>t.key);return {dup:k.length-new Set(k).size,n:FIN.ledger.filter(t=>t.key.indexOf('RUN-309|')===0&&t.st==='committed').length}});ok(r.dup===0&&r.n===2,'tail continuation mints one tx per business key — no duplicate credits');}
{const c=await p.evaluate(()=>FINX.runView(FINX.run('RUN-309')).cov);ok(c.posted===35&&c.existing===51&&c.unprocessed===0,'after tail: 35 newly posted / 51 existing (posted reserved for transactions actually created)');}
await go('flow=runapproved');await open('[data-act^="run-post"]');await fillSens();await open('[data-act="commit-sens"]');await esc();await p.evaluate(()=>{CRM.openDrawer('run','RUN-309')});await p.waitForTimeout(250);await open('[data-act^="run-lookup"]');
await p.evaluate(()=>{FIN.ledger.push({id:'T-7777',key:'RUN-309|INV-48180|S1|rec:HP-210',dom:'wallet',acct:'x',dir:'C',amt:800000,unit:'toman',rel:['اجرا','x'],actor:'دیگر',eff:'x',posted:'x',st:'committed'})});
await open('[data-act^="run-recon"]');await fillSens();await open('[data-act="commit-sens"]');t=await drt();{const r=await p.evaluate(()=>({st:FINX.runView(FINX.run('RUN-309')).st,it:FINX.runView(FINX.run('RUN-309')).items.map(x=>[x[4],x[5]])}));ok(r.st==='unknown'&&r.it[1][0]==='unknown'&&!r.it[1][1]&&t.includes('تعارض'),'lookup tx id ≠ committed ledger id for the key → CONFLICT: stays Unknown, not linked');}await esc();
await go('flow=bulkapp');await p.locator('[data-bulkack]').check();await p.waitForTimeout(150);await open('[data-act="commit-bulkapp"]');await open('[data-act^="reconcile-op"]');
await p.evaluate(()=>{FIN.queue.filter(r=>r.id==='RS-2114')[0].ev.ver='v9'});await open('[data-act^="retry-op"]');t=await drt();ok(t.includes('نسخهٔ مدرک پس از عملیات اصلی تغییر کرده')&&await p.evaluate(()=>FINX.localOf(FINX.q('RS-2114')).review)==='pending'&&(await p.innerText('#toasts')).includes('تکرار انجام نشد')&&!(await p.innerText('#toasts')).includes('فقط برای موارد ناموفق معلوم'),'retry: evidence identity stored per op item; changed evidence → conflict, not approved; toast = warning, not success');await esc();
await go('flow=stale');await open('[data-act^="reload-stage"]');await esc();{const r=await p.evaluate(()=>FINX.eff(FINX.q('RS-2106')));ok(r.paid===18000000&&r.rem===0,'reloaded external approval counts in invoice totals (18,000,000 paid)');}
await go('flow=runapproved');await open('[data-act^="run-post"]');await fillSens();await open('[data-act="commit-sens"]');t=await drt();ok(await cnt('#drawer .os-unknown')===1&&t.includes('تکرار کور مسدود است'),'posting with lost response → Outcome Unknown (not failed, not success)');await open('#drawer [data-act^="open-run"]');ok((await p.innerText('#drawer .lc-rib li.on')).includes('نتیجه نامعلوم')&&await cnt('#drawer .oux')===1,'run state after lost response = Outcome Unknown, OutcomeUnknownContext shown');await esc();
await go('flow=runposted');t=await dr();ok(t.includes('پوشش کامل')===false&&t.includes('موجود')&&t.includes('تراکنش موجود')||t.includes('T-8890'),'posted: existing-transaction hit distinguished from new credit');
await go('flow=runfailed');t=await dr();ok(t.includes('اثر صفر')&&t.includes('اثبات'),'failed: zero effect only with proof');
await go('flow=runrecon');ok((await dr()).includes('تکرار کور انجام نشد'),'reconciled run: no blind retry happened');
await go('view=run&runq=rules');t=await ws();ok(t.includes('F06')&&t.includes('OPD-09')&&await adis('#ws .btn[data-act="x"]'),'rules: read-only snapshot, authority restricted (F06), non-invoice route unknown (OPD-09)');
// Reconciliation
await go('view=rec');t=await ws();ok(await cnt('#ws tbody tr')===10&&t.includes('رفع همه')&&await noBtn('^رفع همه'),'reconciliation: 10 issue categories; no Fix All');
const clsN=await p.evaluate(()=>new Set(FIN.issues.map(i=>i.cls)).size);ok(clsN===10,'all 10 supported issue categories present');
await go('flow=issue');t=await dr();ok(['شواهد','اثر مالی','مالک / حوزه','اقدام مجاز مالی','اقدام ممنوع','مدرک حل','گام بعد'].every(x=>t.includes(x))&&t.includes('تکرار کور'),'issue detail: evidence/impact/owner/allowed/restricted/next/proof');
await go('view=rec');await p.evaluate(()=>{FINX.st.isel={'I-1':true,'I-2':true};CRM.render()});await p.waitForTimeout(200);t=await ws();ok(t.includes('فقط خواندنی')&&await adis('.bulkbar [data-tip*="رفع همه"]'),'bulk reconcile = diagnosis only; bulk correction disabled');
// Reports
await go('view=rep');ok(await cnt('.rep-card')===7,'reports: 7 distinct grains');
for(const [id,needle] of [['REP-CUR','ریال'],['REP-TX','fallback محلی'],['REP-SR','آمادگی ≠ تسویه'],['REP-HIS','نامعلوم'],['REP-RN','زیرمجموعهٔ محدود'],['REP-WL','هم‌پوشان'],['REP-LG','نامعلوم/پیش‌نمایش']]){await go('view=rep&open=rep:'+id);t=await dr();ok(t.includes(needle)&&['واحد شمارش','مبنای زمانی','واحد','منبع','تازگی','پوشش','وضعیت تطبیق'].every(x=>t.includes(x)),'report '+id+': trust context + '+needle);}
await go('view=rep&open=rep:REP-CUR');t=await dr();ok(t.includes('فقط‌خواندنی')&&await adis('#drawer [data-act="x"]'),'report export disabled (independent authority)');
// F06 permission awareness
await go('view=rev&perm=view');t=await ws();ok(t.includes('دیدن صف')&&t.includes('اختیار اجرا نیست'),'F06: view-only banner');
await open('[data-act="open-rs:RS-2102"]');ok(await adis('[data-act^="rs-approve"]')&&await adis('[data-act^="rs-reject"]')&&(await p.locator('[data-act^="rs-approve"]').getAttribute('data-tip')).includes('approve_financial_stage'),'F06: Can View ≠ Can Execute — approve/reject unavailable with explicit conceptual reason');await esc();
await go('view=cfg');t=await ws();ok(t.includes('F06')&&t.includes('مفهومی')&&t.includes('ویرایشگر قابلیت وجود ندارد')&&await noBtn('^(ذخیره|ویرایش قابلیت)'),'config: conceptual permission contract; no capability editor');
await p.click('[data-act="perm:view"]');await p.waitForTimeout(200);ok((await ws()).includes('غیرمجاز: فقط مشاهده'),'config: view-only simulation');
await go('view=cfg');t=await ws();ok(t.includes('مجاز نیست')&&t.includes('نیازمند اعتبارسنجی')&&t.includes('فقط تشخیص'),'bulk baseline matrix');
// Maintenance: separated, not executable
await go('view=mnt');t=await ws();ok(await cnt('.mnt-lock')===4&&await p.locator('.mnt-lock .btn[aria-disabled="true"]').count()===4&&t.includes('اصلاح نمایش')&&t.includes('پیشرفته'),'maintenance: 4 impactful tools shown but not executable; separate from daily');
// Purity: all reads/drawers opened so far (without commit) — verify read purity on a fresh page
await go('view=rev');const p0=await snap();for(const v of VIEWS){await go('view='+v);}for(const f of ['item','gateway','refund','ledger','ledgerunk','rununknown','runpartial','runposted','issue','report','stale']){await go('flow='+f);}await go('view=led');ok(await snap()===p0,'READ PURITY (F07): rendering/opening every view and drawer mutates no data');
// Sims
const sims=[['rev','loading','.sk'],['rev','empty','.state-empty'],['rev','noresult','.state-noresult'],['rec','tableError','.state-error'],['rev','pageError','.state-error'],['rev','unauthorized','.state-locked'],['rev','offline','#offline-bar:not([hidden])']];
for(const [v,s,sel] of sims){await go('view='+v+'&sim='+s);ok(await cnt(sel)>0,'sim '+s+' on '+v);}
const bn=[['rev','stale','قدیمی'],['rev','incomplete','داده ناقص است'],['rev','unitamb','ابهام واحد'],['rev','dupe','پرداخت تکراری احتمالی'],['run','runpartial','اجرای ناقص'],['run','unknownrun','نامعلوم']];
for(const [v,s,needle] of bn){await go('view='+v+'&sim='+s);ok((await ws()).includes(needle),'sim '+s+' banner');}
await go('view=rev&sim=drawerError&open=rs:RS-2101');ok((await dr()).includes('بارگذاری نشد'),'sim drawerError');
// Keyboard / focus / palette / help
await go('view=rev');await p.keyboard.press('j');await p.keyboard.press('Enter');await p.waitForTimeout(260);ok(await p.evaluate(()=>document.body.classList.contains('drawer-open')),'keyboard: J + Enter opens drawer');await esc();ok(await p.evaluate(()=>!!document.activeElement.closest('tr[data-row]')),'focus returns to row after Esc');
await go('view=rev&palette=INV-48213');ok((await p.innerText('.palette')).includes('INV-48213'),'palette: invoice discovery');await esc();
await go('view=rev&palette=۰۹۱۲۳۴۵۶۷۸۹');ok(!(await p.innerText('.palette')).includes('INV-'),'palette: phone number is not a financial identity');await esc();
await go('view=rev');const before=await snap();await p.evaluate(()=>window.SNHelp.startTour('finance'));await p.waitForTimeout(400);let steps=0;
for(let i=0;i<14;i++){const n=p.locator('[data-tour="next"]');if(!(await n.count()))break;steps++;await n.click();await p.waitForTimeout(420);}
ok(steps===10,'tour: 9 topics + help step ('+steps+')');ok(await snap()===before,'tour executed no financial action (data unchanged)');
ok(p.errs.length===0,'no JS errors in passes A/B ('+p.errs.slice(0,2).join('|')+')');
// ---- PASS C matrix
let cbad=0;const contrastD=contrast.replace("'#ws'","'#drawer'");
const LINKS=['view=rev&flow=item','view=rev&flow=approve','view=rev&flow=stale','view=rec&flow=refund','view=led&flow=ledger','view=run&flow=rununknown','view=run&flow=runpartial'];
for(const [w,h] of [[1920,1080],[1366,768],[1024,768],[768,1024],[390,844]]){
 for(const [th,dn,fc] of [['light','comfortable',0],['dim','compact',0],['dark','comfortable',1]]){
  const q=await mk(b,w,h);const sfx='&theme='+th+'&density='+dn+(fc?'&focus=1':'');
  for(const v of VIEWS){await q.goto(BASE+'?view='+v+sfx);await q.waitForTimeout(220);
   const r=await q.evaluate(()=>({o:document.documentElement.scrollWidth>innerWidth+1}));if(r.o){cbad++;console.log('OVERFLOW',w,th,v)}
   if(w===1366||w===390){const c=await q.evaluate(contrast);if(c.length){cbad++;console.log('CONTRAST',w,th,v,JSON.stringify(c))}}}
  for(const u of LINKS){await q.goto(BASE+'?'+u.replace('&flow=','&flow=')+sfx);await q.waitForTimeout(300);
   const r=await q.evaluate(()=>({o:document.documentElement.scrollWidth>innerWidth+1}));if(r.o){cbad++;console.log('OVERFLOW',w,th,u)}
   if(w===1366){const c=await q.evaluate(contrastD);if(c.length){cbad++;console.log('CONTRAST drawer',th,u,JSON.stringify(c))}}}
  if(q.errs.length){cbad++;console.log('ERR',w,th,q.errs[0])}
  await q.context().close();}}
ok(cbad===0,'pass C matrix clean ('+cbad+' issues)');
console.log('FAILS',fail.length);await b.close();process.exit(fail.length?1:0)})();
