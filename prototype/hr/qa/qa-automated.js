// Automated QA (SN-205 HR: Pass A structure, Pass B critical flows, Pass C matrix, Pass D regression is run separately via the other roles' scripts).
// Run: python3 -m http.server 8777 (from prototype/), then node hr/qa/qa-automated.js — requires playwright.
const { chromium } = require('playwright');
const BASE='http://127.0.0.1:8777/hr/index.html';
const fail=[];const ok=(c,m)=>{console.log((c?'PASS ':'FAIL ')+m);if(!c)fail.push(m)};
async function mk(b,w,h){const ctx=await b.newContext({viewport:{width:w,height:h},reducedMotion:'reduce'});await ctx.addInitScript(()=>{try{localStorage.setItem('snproto.hr_welcome_v1_seen','1')}catch(e){}});const p=await ctx.newPage();p.errs=[];p.on('pageerror',e=>p.errs.push(e.message));return p}
const contrast=`(()=>{const lum=c=>{const [r,g,b]=c.map(v=>{v/=255;return v<=.03928?v/12.92:Math.pow((v+.055)/1.055,2.4)});return .2126*r+.7152*g+.0722*b};
const parse=s=>{const m=s.match(/[\\d.]+/g);return m?m.map(Number):null};
const bgOf=e=>{let a=[255,255,255],stack=[];for(;e;e=e.parentElement){const c=parse(getComputedStyle(e).backgroundColor);if(c&&(c.length<4||c[3]>0)){stack.push(c)}if(c&&(c.length<4||c[3]===1))break}
let base=[255,255,255];for(let i=stack.length-1;i>=0;i--){const c=stack[i],al=c.length>3?c[3]:1;base=base.map((v,k)=>c[k]*al+v*(1-al))}return base};
const bad=[];const w=document.createTreeWalker(document.querySelector('#ws'),NodeFilter.SHOW_TEXT);let n;const seen=new Set();
while(n=w.nextNode()){if(!n.textContent.trim())continue;const e=n.parentElement;if(seen.has(e)||!e.getClientRects().length)continue;seen.add(e);const cs=getComputedStyle(e);if(cs.visibility==='hidden'||cs.opacity==='0')continue;
const fg=parse(cs.color);const bg=bgOf(e);const f=fg.slice(0,3);const a=fg.length>3?fg[3]:1;const fb=f.map((v,k)=>v*a+bg[k]*(1-a));const L1=lum(fb),L2=lum(bg);const r=(Math.max(L1,L2)+.05)/(Math.min(L1,L2)+.05);const big=parseFloat(cs.fontSize)>=24||(parseFloat(cs.fontSize)>=18.6&&+cs.fontWeight>=700);if(r<(big?3:4.5)&&!e.closest('[disabled],[aria-disabled=true],.dim,.off'))bad.push(e.className.toString().slice(0,30)+'|'+e.textContent.trim().slice(0,20)+'|'+r.toFixed(2))}
return bad.slice(0,6)})()`;
(async()=>{const b=await chromium.launch({});
const VIEWS=['work','onb','req','acc','comp','exc','cred','bulk','diag'];
let p=await mk(b,1366,768);let t;
const go=async(q)=>{await p.goto(BASE+'?'+q);await p.waitForTimeout(350)};
const open=async(sel)=>{await p.click(sel);await p.waitForTimeout(260)};
const dr=async()=>await p.innerText('#drawer');
const drt=async()=>await p.evaluate(()=>document.querySelector('#drawer').textContent);
const ws=async()=>await p.innerText('#ws');
const cnt=async(s)=>await p.locator(s).count();
const esc=async()=>{await p.keyboard.press('Escape');await p.waitForTimeout(120)};
const dis=async(s)=>await p.locator(s).isDisabled();
const fillSens=async(r)=>{await p.fill('#sens-r',r||'دلیل ممیزی');await p.locator('[data-sensack]').check();await p.waitForTimeout(200)};
const noBtn=async(re)=>!(await p.evaluate((src)=>[...document.querySelectorAll('button,a')].some(x=>new RegExp(src).test(x.textContent)),re));
// ---- PASS A structure
for(const v of VIEWS){await go('view='+v);ok((await ws()).length>300,'render '+v);}
ok(p.errs.length===0,'pass A: no JS errors');
for(const v of VIEWS){await go('view='+v);const bad=await p.evaluate(()=>({cap:[...document.querySelectorAll('#ws table.tbl')].filter(t=>!t.querySelector('caption')).length,noname:[...document.querySelectorAll('button')].filter(x=>!x.textContent.trim()&&!x.getAttribute('aria-label')).length,main:!!document.querySelector('main#ws'),nav:document.querySelectorAll('nav[aria-label]').length}));ok(bad.cap===0&&bad.noname===0&&bad.main&&bad.nav>=2,'semantics '+v);}
ok((await p.innerText('.demo-tag')).includes('NOT LIVE VERIFIED'),'header: HR runtime NOT LIVE VERIFIED visible');
// ---- PASS B
// Workforce
await go('view=work');t=await ws();
ok(await cnt('tr[data-row^="staff:"]')===16&&await cnt('.kpis .kpi')<=3&&!/نمره|امتیاز|درصد آمادگی/.test(t),'workforce: 16 profiles, no score/percentage KPI');
ok(await cnt('tr[data-row^="staff:"]:first-child .idcell .lc')===2&&t.includes('مدیر فعلی — اعتبار تاریخی نیست'),'workforce: person↔profile and profile↔account linkage shown separately; current manager ≠ historical');
await open('[data-wq="conf"]');ok(await cnt('tr[data-row^="staff:"]')===9,'workforce: conflict queue (9)');
await go('view=work&open=staff:HP-303');t=await dr();
ok(await cnt('#drawer .idl .idn')===3&&await cnt('#drawer .own7 .own')===7&&t.includes('هویت شخص سراسری هنوز تعریف نشده'),'profile: Person/Profile/User + 7 separate entities; no global person ID');
ok(await cnt('#drawer .eit .ei')===2&&await cnt('#drawer .ei-unknown')===1&&t.includes('نامعلوم (UNKNOWN)')&&t.includes('ناقص (INCOMPLETE)')&&t.includes('از مدیر امروز جایگزین نمی‌شود'),'timeline: UNKNOWN/INCOMPLETE never inferred from today');
ok(['تاریخ اثر','زمان اعمال','عامل رویداد','مدیر در این بازه'].every(x=>t.includes(x))&&t.includes('بازنویسی نمی‌کند'),'timeline: effective, applied, actor, manager per interval; no rewrite');
ok(await noBtn('غیرفعال.?کردن کاربر|Disable user|حذف نیرو|حذف کاربر')&&await cnt('#drawer [data-act^="open-term"]')===1&&await cnt('#drawer [data-act^="open-xfer"]')===1,'profile: no «disable user»/delete shortcut; termination goes through impact review');
await go('view=work&open=staff:HP-306');t=await dr();ok(t.includes('پیوند ناموجود')&&t.includes('پیوند نشده')&&t.includes('ورود نیرو ناتمام'),'missing user linkage visible');
await go('view=work&open=staff:HP-305');t=await dr();ok(t.includes('HP-305b')&&t.includes('ادغام مخرب')&&await noBtn('ادغام'),'duplicate profile: no destructive merge');
// Onboarding
await go('view=onb');t=await ws();ok(await cnt('#ws .onb-st')===2&&t.includes('گام عادی ورود نیرو نیست')&&t.includes('گام ساختگی ندارد'),'onboarding: staged, credentials not an ordinary step, no invented workflow');
await open('[data-act="open-onbwiz"]');ok(await cnt('input[data-ochoice]:checked')===0&&await dis('[data-act="onb-next"]')&&(await dr()).includes('تعارض'),'wizard: identity conflict, no default, next gated');
await p.check('[data-ochoice="user"]');await p.waitForTimeout(200);await open('[data-act="onb-next"]');await open('[data-act="onb-next"]');ok(await dis('[data-act="onb-next"]'),'wizard: position required');
await p.check('[data-opos="seller"]');await p.waitForTimeout(200);await open('[data-act="onb-next"]');ok(await cnt('input[data-opar="HP-130"][disabled]')===1,'wizard: inactive manager not selectable');
await p.check('[data-opar="HP-110"]');await p.waitForTimeout(200);await open('[data-act="onb-next"]');t=await dr();
ok(await cnt('#drawer .sens-grid .sens-c')===4&&await cnt('#drawer input[type=password]')===0&&t.includes('ایجاد یا ارسال نمی‌شود')&&await dis('[data-act="onb-commit"]'),'wizard: access step = sensitive confirmation; no credential field; commit gated');
await p.fill('#sens-r','ورود نیروی جدید');await p.locator('[data-sensack]').check();await p.waitForTimeout(200);await open('[data-act="onb-commit"]');t=await dr();
ok(await cnt('#drawer .outcome-strip .os')===7&&t.includes('ورود قابل‌استفاده تأیید نشده')&&t.includes('جزو ورود نیرو نیست')&&t.includes('ناقص تکمیل شد'),'onboarding result: per-effect, access unresolved, credentials skipped, not all-success');await esc();
// Transfer preview
await go('view=onb&flow=xfer');t=await dr();
ok(['تاریخ اثر درخواستی','زمان اعمال واقعی','مسیر بررسی','مدیر فعلی','مدیر مقصد','حفظ تاریخچه','اثر روی دسترسی','اثر روی مسئولیت‌های عملیاتی'].every(x=>t.includes(x))&&t.includes('OPD-05')&&t.includes('انتقال پرونده'),'transfer preview: subject/current/target/reason path/effective vs applied/access/responsibilities/history/OPD-05');
ok(await p.locator('[data-act^="xfer-review"]').isEnabled(),'transfer preview: continue enabled for valid target');
await open('[data-act^="xfer-review"]');t=await dr();ok(await cnt('#drawer .sens-grid .sens-c')===4&&await dis('[data-act="commit-sens"]')&&t.includes('تغییر نمی‌کند'),'sensitive confirmation: changes/unchanged/affected/effective + reason gate');
await fillSens('نیاز تیم');await open('[data-act="commit-sens"]');t=await dr();ok(await cnt('#drawer .outcome-strip .os')===7&&t.includes('مسئول فعلی کارها و مالک اعتبار تغییر نکرد'),'transfer result: per-effect, credit/ownership untouched');await esc();
await p.evaluate(()=>CRM.openDrawer('staff','HP-301'));await p.waitForTimeout(260);ok(await cnt('#drawer .eit .ei')===1+2&&(await dr()).includes('ویرایش مستقیم'),'transfer applied: new interval added, previous kept (history preserved)');
await go('view=onb&flow=targetinactive');t=await dr();ok(t.includes('مدیر مقصد غیرفعال است')&&await dis('[data-act^="xfer-review"]')&&await cnt('input[data-xt="HP-130"][disabled]')===1,'target manager inactive: blocked');
await go('view=onb&flow=effconflict');t=await dr();ok(t.includes('تعارض تاریخ اثر')&&await dis('[data-act^="xfer-review"]'),'effective-date conflict: blocked, no backfill');
await go('view=onb&flow=xpartial');await fillSens();await open('[data-act="commit-sens"]');t=await dr();ok(await cnt('#drawer .os-failed b')===1&&await cnt('[data-act^="retry-op"]')===1&&t.includes('ناموفق'),'transfer partial: access effect failed, retry only that');
await open('[data-act^="retry-op"]');t=await dr();ok(!(await cnt('[data-act^="retry-op"]')),'transfer partial: retry only known failed');
await go('view=onb&flow=xunknown');await fillSens();await open('[data-act="commit-sens"]');t=await dr();ok(await cnt('#drawer .os-unknown b')===1&&await cnt('[data-act^="reconcile-op"]')===1&&t.includes('نتیجه نامعلوم است'),'transfer unknown: reconcile first');
await open('[data-act^="reconcile-op"]');ok(!(await dr()).includes('نتیجه نامعلوم است')&&await p.evaluate(()=>HR.staff.filter(x=>x.id==='HP-301')[0].parent==='HP-120'&&HR.staff.filter(x=>x.id==='HP-301')[0].hist.length===3),'transfer unknown reconciled: parent + history synced');await esc();
// Termination
await go('view=req&flow=term');t=await dr();
ok(['وضعیت اشتغال','وضعیت دسترسی','مدیر فعلی','کار باز','مسئول فعلی','مسئولیت آینده','مسئولیت فاکتور و مشتری','مرز جبران خدمات','موارد تحویل حل‌نشده'].every(x=>t.includes(x))&&await cnt('#drawer .rva')===1,'termination: full impact review sections');
ok(t.includes('نامشخص (UNKNOWN)')&&t.includes('OPD-05')&&t.includes('لغو نمی‌شود')&&t.includes('تحویل ناتمام')&&await p.locator('#drawer .dr-foot button[disabled]').count()>0&&!/بازتخصیص خودکار پیشنهاد می‌شود/.test(t),'termination: handover incomplete → blocked; next actor UNKNOWN; access revocation not implied; no auto reassignment');
await go('view=req&flow=termclear');t=await dr();ok(t.includes('کار باز ندارد')&&await p.locator('[data-act^="term-review"]').isEnabled()&&t.includes('آف‌بوردینگ کامل')&&t.includes('فقط وضعیت اشتغال را عوض می‌کند'),'termination: clear subject still not full offboarding');
await open('[data-act^="term-review"]');await fillSens('پایان قرارداد');await open('[data-act="commit-sens"]');t=await dr();ok(await cnt('#drawer .os-applied b')===1&&(await p.locator('#drawer .os-applied b').innerText())==='۲'&&await cnt('#drawer .os-skipped b')>0&&t.includes('آف‌بوردینگ کامل ادعا نمی‌شود')&&t.includes('لغو نقش و نشست'),'termination result: only employment state changed; access/handover skipped');await esc();
await p.evaluate(()=>CRM.openDrawer('staff','HP-311'));await p.waitForTimeout(260);ok((await dr()).includes('پایان همکاری'),'terminated state reflected');
// Requests
await go('view=req');t=await ws();ok(await cnt('tr[data-row^="req:"]')===3&&t.includes('شناسه‌های داخلی وضعیت حفظ شده‌اند'),'requests: HR queue (3), internal IDs preserved');
await open('[data-act="open-req:R-502"]');t=await dr();ok(await cnt('#drawer .steps')===1&&await cnt('#drawer .rva')===1&&t.includes('هنوز اعمال نشده')&&await p.locator('[data-act="req-apply:R-502"]').isEnabled(),'request pending_hr: lifecycle, requested vs not-yet-applied');
await esc();await open('[data-rq="chain"]');ok(await cnt('tr[data-row^="req:"]')===4,'chain queue (4)');
await open('[data-act="open-req:R-501"]');t=await dr();ok(t.includes('HR-G11')&&await p.locator('[data-act="req-step:R-501"]').isEnabled()&&await noBtn('اعمال نهایی'),'pending_review: step approve only, no chain-bypass apply');
await open('[data-act="req-reject:R-501"]');ok(await p.locator('#rj-n').getAttribute('aria-invalid')==='true'&&(await dr()).includes('دلیل لازم'),'reject requires reason (aria-invalid)');
await p.fill('#rj-n','ظرفیت تیم');await open('[data-act="req-step:R-501"]');t=await dr();ok(t.includes('در انتظار منابع انسانی')&&t.includes('هنوز اعمال نشده'),'step approval ≠ applied (becomes pending HR, not applied)');await esc();
await go('view=req&flow=reqchanged');t=await dr();ok(t.includes('پیش از بررسی تغییر کرده')&&await dis('[data-act="req-step:R-508"]'),'request changed before review: blocked until reread');
await go('view=req&flow=reqoos');ok((await dr()).includes('خارج از محدوده')&&await dis('[data-act="req-step:R-509"]'),'out-of-scope: no rank bypass');
await go('view=req&flow=reqtermblock');t=await dr();ok(t.includes('OPD-05')&&await dis('[data-act="req-apply:R-504"]'),'termination request: final apply blocked while handover unresolved');
await go('view=req&rq=review');await open('[data-act="open-req:R-503"]');ok(await dis('[data-act="req-apply:R-503"]')&&(await dr()).includes('مدیر مقصد غیرفعال است'),'request with inactive target: apply blocked');
await go('view=req&flow=reqapplied');t=await dr();ok(t.includes('اعمال')&&t.includes('مدیر فرهاد نجفی')&&await cnt('#drawer [data-act^="req-"]')===0,'applied request: actual before/after proof, no actions');
await go('view=req&flow=reqfailed');t=await dr();ok(t.includes('نامعلوم')&&await dis('.dr-foot button:has-text("اعمال دوباره")'),'failed request: effects unknown, no blind re-apply');
await open('[data-act="req-reconcile:R-506"]');t=await dr();ok(t.includes('بازه فعلی بسته شده و بازه جدید ثبت نشده')&&t.includes('سیاست بازیابی تعریف نشده'),'failed: reconcile reads actual effect; recovery policy undefined');await esc();
await go('view=req&rq=review');await open('[data-act="open-req:R-502"]');await open('[data-act="req-apply:R-502"]');ok(await dis('[data-act="commit-sens"]')&&(await dr()).includes('اعمال نهایی'),'final apply: separate confirmation');
await fillSens('اعمال');await open('[data-act="commit-sens"]');t=await dr();ok(await cnt('#drawer .os-applied b')===1&&(await p.locator('#drawer .os-applied b').innerText())==='۴','final apply result: per-effect');await esc();
ok(await p.evaluate(()=>HR.staff.filter(x=>x.id==='HP-302')[0].hist[0].src==='درخواست R-502'&&HR.audit[0].req==='R-502'),'request-driven transfer keeps request provenance in history and audit');
await open('[data-rq="closed"]');ok((await ws()).includes('تأیید و اعمال‌شده'),'applied request moved to closed');
await go('view=req&flow=applyfail');await open('[data-act="req-apply:R-502"]');await fillSens('اعمال');await open('[data-act="commit-sens"]');t=await dr();ok((await p.locator('#drawer .os-unknown b').innerText())==='۱'&&(await p.locator('#drawer .os-failed b').innerText())==='۰'&&await cnt('[data-act^="retry-op"]')===0&&t.includes('اعمال دوباره کور ممنوع'),'apply failure: effect is Unknown, retry NOT offered before reconcile, blind re-apply forbidden');
ok(await p.evaluate(()=>HRX.st.reqState['R-502']==='failed'&&(x=>x.parent===null&&x.parentUnknown===true&&x.hist[1].to)(HR.staff.filter(x=>x.id==='HP-302')[0])),'apply failure: request failed; closure kept, current manager Unknown (not previous manager) until reconciled');
await open('[data-act^="reconcile-op"]');ok(await p.evaluate(()=>HRX.st.reqState['R-502']==='approved'&&HR.staff.filter(x=>x.id==='HP-302')[0].parent==='HP-120'&&HR.staff.filter(x=>x.id==='HP-302')[0].hist[0].src==='درخواست R-502'),'reconcile syncs domain state (parent, history with request provenance, request status)');
ok(await p.evaluate(()=>HR.audit[0].req==='R-502'&&HR.audit[0].chain.indexOf('تأیید مرحله‌ای')>-1&&HR.audit[0].res==='complete'),'audit keeps request ID and review chain; result updated after reconcile');await esc();
// Structure & access
await go('view=acc');t=await ws();ok(await cnt('tr[data-row^="access:"]')===16&&t.includes('ویرایشگر قابلیت‌های وردپرس')&&!(await cnt('#ws input[type=checkbox]')),'access: inspector (16), not a capability editor');
await go('view=acc&flow=access');t=await dr();
ok(await cnt('#drawer table.asm')===1&&await cnt('#drawer .asm-delta')>0&&t.includes('پس از تغییر')&&t.includes('استثنای مستقیم')&&t.includes('ارث‌بری از والد')&&t.includes('نگاشت سمت')&&t.includes('حساس'),'access matrix: source, inherited/direct, effective, override reason, sensitive delta, comparison');
ok(t.includes('چهار چیز جدا')&&t.includes('اعتبارنامه')&&t.includes('بازنشانی نمی‌شوند'),'access: position/role/permission/credential separated; overrides preserved');
await open('[data-act^="access-review"]');t=await dr();ok(t.includes('استثناهای مستقیم')&&t.includes('نقش‌ها و قابلیت‌های نامرتبط')&&await dis('[data-act="commit-sens"]'),'access change confirmation: unchanged list + gate');
await fillSens('اصلاح نگاشت');await open('[data-act="commit-sens"]');t=await dr();ok(await cnt('#drawer .outcome-strip .os')===7,'access result strip');await esc();
await go('view=acc&flow=accpartial');await open('[data-act^="access-review"]');await fillSens();await open('[data-act="commit-sens"]');t=await dr();ok(await cnt('#drawer .os-failed b')===1&&t.includes('وضعیت مؤثر با مورد انتظار فرق دارد'),'access partial apply: distinct');await esc();
await go('view=acc&open=access:HP-307');t=await dr();ok(t.includes('نقش (قدیمی/WP)')&&t.includes('نقش سرپرست باقی مانده')&&t.includes('ناهمخوان'),'access mismatch: leftover legacy role is the source of extra permissions');await esc();
await go('view=acc&aq=struct');t=await ws();ok(await cnt('tr[data-row^="pos:"]')===6&&t.includes('واحد / تیم ≠ سلسله‌مراتب گزارش‌دهی'),'structure: positions + unit ≠ reporting parent');
await open('[data-act="open-pos:seller"]');ok((await dr()).includes('مسدود توسط ارجاع')&&await p.locator('#drawer .dr-foot button[disabled]').count()>0,'position deactivate blocked by dependencies');await esc();
// Compensation
await go('view=comp');t=await ws();ok(await cnt('tr[data-row^="comp:"]')===5&&t.includes('جدا از کیف پول')&&t.includes('قفل پس از حقوق')&&(await p.locator('tr[data-row="comp:HP-306"] .na').count())>=1,'compensation: separate from wallet/Finance; unset → «—» not 0');
await open('[data-act="open-comp:HP-303"]');t=await dr();ok(await cnt('#drawer .eit .ei')===2&&t.includes('همپوشانی')&&t.includes('ناقص')&&t.includes('قبل ← بعد')&&t.includes('دوباره محاسبه نمی‌کند')&&t.includes('کمیسیون (ارجاع'),'comp history: interval, before/after, overlap, reference-only commission, no recalc');
ok(await dis('[data-act^="comp-review"]'),'comp edit gated');
await p.fill('#cp-b','۳۴٬۰۰۰٬۰۰۰');await p.fill('#cp-f','۱۴۰۵/۰۸/۰۱');await p.fill('#cp-r','بازبینی');await p.waitForTimeout(200);await open('[data-act^="comp-review"]');t=await dr();ok(t.includes('بدون بازمحاسبه')&&t.includes('کیف پول')&&await dis('[data-act="commit-sens"]'),'comp confirmation: past credit not recalculated; wallet/Finance untouched');
await fillSens('بازبینی');await open('[data-act="commit-sens"]');t=await dr();ok(await cnt('#drawer .os-applied b')>0&&t.includes('بدون بازمحاسبه')||t.includes('دوباره محاسبه نشد'),'comp result');await esc();
await go('view=comp&flow=compunknown');await p.fill('#cp-b','۳۴٬۰۰۰٬۰۰۰');await p.fill('#cp-f','۱۴۰۵/۰۸/۰۱');await p.fill('#cp-r','بازبینی');await p.waitForTimeout(200);await open('[data-act^="comp-review"]');await fillSens();await open('[data-act="commit-sens"]');t=await dr();ok(await cnt('#drawer .os-unknown b')===1&&t.includes('دوره قبلی بسته شد')&&await p.evaluate(()=>HR.comp['HP-301'].periods.length===2),'comp: failure after closing previous period → unknown, new period not claimed');await open('[data-act^="reconcile-op"]');ok(await p.evaluate(()=>HR.comp['HP-301'].periods.length===3&&HR.comp['HP-301'].base==='۳۴٬۰۰۰٬۰۰۰'),'comp reconcile syncs recovered period');await esc();
await go('view=comp&compAuth=denied');t=await ws();ok(await cnt('.state-locked')===1&&!/[۰-۹]{2},[۰-۹]{3}|[۰-۹]٬[۰-۹]{3}/.test(t)&&t.includes('صفر یا نبود جبران خدمات نیست'),'compensation unauthorized: locked, no values, not zero');
// Exceptions / audit
await go('view=exc');t=await ws();ok(await cnt('tr[data-row^="exc:"]')===11&&t.includes('تشخیص است، نه فرمان')&&await noBtn('تشدید|ارجاع'),'exceptions: 11 classes, diagnostic, no escalation command/SLA');
const xc=new Set(await p.locator('tr[data-row^="exc:"] .exc-cls .pill').allInnerTexts());ok(xc.size===11,'exceptions: 11 distinct classes ('+xc.size+')');
await open('[data-act="open-exc:X8"]');t=await dr();ok(['موضوع و مالک','اثر','اقدام مجاز منابع انسانی','اقدام ممنوع','مدرک حل'].every(x=>t.includes(x))&&t.includes('پرونده بدون مسئول')&&t.includes('OPD')||t.includes('تحویل'),'exception drawer: subject/owner/impact/allowed/restricted/proof');await esc();
await open('[data-xq="audit"]');ok(await cnt('tr[data-row^="audit:"]')>=6,'audit timeline');await open('[data-act="open-audit:A8"]');t=await dr();ok(['عامل','قبل','بعد','دلیل','زنجیره بررسی','تاریخ اثر','زمان اعمال','شناسه همبستگی'].every(x=>t.includes(x))&&t.includes('ماسک'),'audit item: actor/before/after/reason/chain/effective/applied/correlation; masked');await esc();
// Credentials / impersonation
await go('view=cred');t=await ws();ok(await cnt('tr[data-row^="staff:"]')===15&&t.includes('مسیر محدود')&&t.includes('فقط‌خواندنی نیست')&&(await cnt('.pill:has-text("سیاست هدف مجاز تأیید نشده")'))>0,'credentials: restricted, leadership targets blocked');
await open('[data-act="open-credreset:HP-110"]');ok(await dis('[data-act^="cred-review"]')&&await dis('#pw-1')&&(await dr()).includes('ادامه مسدود'),'credential reset: leadership target blocked');await esc();
await open('[data-act="open-credreset:HP-301"]');ok(await dis('[data-act^="cred-review"]'),'credential reset gated');
await p.fill('#pw-1','Secret#1234');await p.fill('#pw-2','Other#1234');await p.locator('#pw-2').blur();await p.waitForTimeout(150);ok(await p.locator('#pw-2').getAttribute('aria-invalid')==='true'&&await p.locator('#pw-2e').isVisible(),'mismatch → aria-invalid + message');
await p.fill('#pw-2','Secret#1234');await p.fill('#pw-r','درخواست کاربر');await p.locator('[data-sms]').check();await p.waitForTimeout(150);ok(await p.locator('[data-act^="cred-review"]').isEnabled(),'credential reset enabled when valid');
await open('[data-act^="cred-review"]');t=await dr();ok(t.includes('گذرواژه در لاگ یا ممیزی ثبت نمی‌شود')&&!t.includes('Secret#1234')&&await dis('[data-act="commit-sens"]'),'confirmation: secret never shown');
await fillSens();await open('[data-act="commit-sens"]');t=await drt();ok(await cnt('#drawer .outcome-strip .os')===7&&t.includes('بازنشانی گذرواژه')&&t.includes('اعلان پیامکی')&&!(await p.evaluate(()=>document.documentElement.innerHTML.includes('Secret#1234'))),'reset result: reset and SMS separate; secret absent from DOM');await esc();
await go('view=cred&flow=credpartial');await p.fill('#pw-1','Secret#1234');await p.fill('#pw-2','Secret#1234');await p.fill('#pw-r','درخواست');await p.locator('[data-sms]').check();await p.waitForTimeout(150);await open('[data-act^="cred-review"]');await fillSens();await open('[data-act="commit-sens"]');t=await drt();ok(await cnt('#drawer .os-failed b')===1&&t.includes('انجام شد؛ مقدار ثبت نشد')&&t.includes('ارسال ناموفق؛ بازنشانی معتبر است'),'reset success + SMS failure = separate outcomes');await esc();
await go('view=cred&flow=imp');t=await dr();ok(['عامل','هدف','وضعیت شروع','بازگشت','فقط‌خواندنی نیست','ممیزی'].every(x=>t.includes(x))&&await dis('[data-act^="imp-review"]')&&!(await cnt('#drawer input[type=password]')),'impersonation: actor/target/start/return/not read-only/audit; gated');
await p.check('[data-purpose="0"]');await p.locator('[data-ack0]').check();await p.waitForTimeout(200);await open('[data-act^="imp-review"]');t=await dr();ok(t.includes('فقط‌خواندنی نیست')&&await dis('[data-act="commit-sens"]'),'impersonation confirmation');
await fillSens();await open('[data-act="commit-sens"]');await p.waitForTimeout(300);ok(await cnt('.imp-bar')===1&&(await p.innerText('.imp-bar')).includes('هر اقدام با هویت او ثبت می‌شود')&&await cnt('[data-act="imp-end"]')===1,'active session warning + return action');
await open('[data-act="open-impstart:HP-301"]');ok((await dr()).includes('جلسه تو در تو ممنوع')&&await dis('[data-act^="imp-review"]'),'no nested impersonation');await esc();
await open('[data-act="imp-end"]');ok(await cnt('.imp-bar')===0,'return ends session');
await go('view=cred&flow=implead');ok((await dr()).includes('ادامه مسدود'),'impersonation: leadership target blocked');
// Bulk
await go('view=bulk');t=await ws();ok(await cnt('#ws tbody tr')===7&&t.includes('ممنوع در خط پایه')&&t.includes('نیازمند اعتبارسنجی')&&await cnt('#ws tbody tr button[disabled]')===3,'bulk baseline: termination not allowed, access needs validation, no destructive default');
await open('[data-act="open-bulkwf"]');ok(await dis('[data-act="commit-bulkwf"]'),'bulk workforce gated');await p.locator('[data-bulkack]').check();await p.waitForTimeout(150);await open('[data-act="commit-bulkwf"]');t=await dr();ok(await cnt('#drawer .outcome-strip .os')===7&&t.includes('برابر «درخواست‌شده»')&&await cnt('#drawer .os-unknown b')===1,'bulk workforce: per-item applied/skipped/failed/unknown');await esc();
await open('[data-act="open-bulkcred"]');await p.locator('[data-bulkack]').check();await p.waitForTimeout(150);await open('[data-act="commit-bulkcred"]');t=await dr();ok(t.includes('شکست پیامک شکست بازنشانی نیست')&&await cnt('#drawer .outcome-strip .os')===7,'bulk credential reset: reset vs SMS separate');await esc();
await open('[data-act="open-export"]');ok((await dr()).includes('ماسک‌بودن لاگ ≠ ماسک‌بودن خروجی')&&await p.locator('#drawer .dr-foot button[disabled]').count()>0,'export: conditional, disabled');await esc();
// Diagnostics
await go('view=diag');ok(await cnt('#ws tbody tr')===11,'old→new map (11)');await open('[data-dq="legacy"]');ok(await cnt('#ws tbody tr')===4,'legacy mapping');await open('[data-dq="log"]');ok((await ws()).includes('***')||await cnt('.rawlog')===1,'raw log masked');
// Sims
const sims=[['work','loading','.sk'],['work','empty','.state-empty'],['work','noresult','.state-noresult'],['onb','tableError','.state-error'],['req','pageError','.state-error'],['work','unauthorized','.state-locked'],['work','offline','#offline-bar:not([hidden])']];
for(const [v,s,sel] of sims){await go('view='+v+'&sim='+s);ok(await cnt(sel)>0,'sim '+s+' on '+v);}
const bn=[['work','stale','قدیمی'],['work','incomplete','داده ناقص است'],['req&rq=chain','reqchanged','پیش از بررسی تغییر کرده'],['onb','targetinactive','مدیر مقصد غیرفعال'],['work','dupprofile','پروفایل تکراری'],['acc','accesspartial','اعمال دسترسی ناقص'],['req','termincomplete','تحویل پایان همکاری ناتمام'],['onb','effconflict','تعارض تاریخ اثر'],['work','histunavail','تاریخچه انتساب در دسترس نیست'],['cred','credpartial','اعلان ناموفق بود']];
for(const [v,s,needle] of bn){await go('view='+v+'&sim='+s);ok((await ws()).includes(needle),'sim '+s+' banner');}
await go('view=work&sim=drawerError&open=staff:HP-303');ok((await dr()).includes('بارگذاری نشد'),'sim drawerError');
// Keyboard / focus / palette / help
await go('view=work');await p.keyboard.press('j');await p.keyboard.press('Enter');await p.waitForTimeout(260);ok(await p.evaluate(()=>document.body.classList.contains('drawer-open')),'keyboard: J + Enter opens drawer');await esc();ok(await p.evaluate(()=>!!document.activeElement.closest('tr[data-row]')),'focus returns to row after Esc');
await go('view=work&palette=۰۹۱۹');ok((await p.innerText('.palette')).includes('کشف با شماره (هویت نیست)'),'palette: phone only as discovery');await esc();
const snap=async()=>await p.evaluate(()=>JSON.stringify([window.HR.staff,window.HR.requests,window.HR.comp,window.HR.audit.length,window.HRX.st.ops.length,window.HRX.st.imp]));
await go('view=work');const before=await snap();await p.evaluate(()=>window.SNHelp.startTour('hr'));await p.waitForTimeout(400);let steps=0;
for(let i=0;i<14;i++){const n=p.locator('[data-tour="next"]');if(!(await n.count()))break;steps++;await n.click();await p.waitForTimeout(420);}
ok(steps===9,'tour: 8 topics + help step ('+steps+')');ok(await snap()===before,'tour executed no business action (data unchanged)');
ok(p.errs.length===0,'no JS errors in passes A/B ('+p.errs.slice(0,2).join('|')+')');
// ---- PASS C matrix
let cbad=0;const contrastD=contrast.replace("'#ws'","'#drawer'");
const LINKS=['view=work&open=staff:HP-303','view=req&open=term:HP-304','view=req&open=req:R-502','view=acc&open=access:HP-307','view=comp&open=comp:HP-303','view=cred&imp=1','view=onb&flow=xfer'];
for(const [w,h] of [[1920,1080],[1366,768],[1024,768],[768,1024],[390,844]]){
 for(const [th,dn,fc] of [['light','comfortable',0],['dim','compact',0],['dark','comfortable',1]]){
  const q=await mk(b,w,h);const sfx='&theme='+th+'&density='+dn+(fc?'&focus=1':'');
  for(const v of VIEWS){await q.goto(BASE+'?view='+v+sfx);await q.waitForTimeout(220);
   const r=await q.evaluate(()=>({o:document.documentElement.scrollWidth>innerWidth+1}));if(r.o){cbad++;console.log('OVERFLOW',w,th,v)}
   if(w===1366||w===390){const c=await q.evaluate(contrast);if(c.length){cbad++;console.log('CONTRAST',w,th,v,JSON.stringify(c))}}}
  for(const u of LINKS){await q.goto(BASE+'?'+u+sfx);await q.waitForTimeout(280);
   const r=await q.evaluate(()=>({o:document.documentElement.scrollWidth>innerWidth+1}));if(r.o){cbad++;console.log('OVERFLOW',w,th,u)}
   if(w===1366&&u.indexOf('open=')>-1){const c=await q.evaluate(contrastD);if(c.length){cbad++;console.log('CONTRAST drawer',th,u,JSON.stringify(c))}}}
  if(q.errs.length){cbad++;console.log('ERR',w,th,q.errs[0])}
  await q.context().close();}}
ok(cbad===0,'pass C matrix clean ('+cbad+' issues)');
console.log('FAILS',fail.length);await b.close();process.exit(fail.length?1:0)})();
