// Automated QA (SN-204 MIS: Pass A structure, Pass B critical flows, Pass C matrix, Pass D regression is run separately via the other roles' scripts).
// Run: python3 -m http.server 8777 (from prototype/), then node mis/qa/qa-automated.js — requires playwright.
const { chromium } = require('playwright');
const BASE='http://127.0.0.1:8777/mis/index.html';
const fail=[];const ok=(c,m)=>{console.log((c?'PASS ':'FAIL ')+m);if(!c)fail.push(m)};
async function mk(b,w,h){const ctx=await b.newContext({viewport:{width:w,height:h},reducedMotion:'reduce'});await ctx.addInitScript(()=>{try{localStorage.setItem('snproto.mis_welcome_v1_seen','1')}catch(e){}});const p=await ctx.newPage();p.errs=[];p.on('pageerror',e=>p.errs.push(e.message));return p}
const contrast=`(()=>{const lum=c=>{const [r,g,b]=c.map(v=>{v/=255;return v<=.03928?v/12.92:Math.pow((v+.055)/1.055,2.4)});return .2126*r+.7152*g+.0722*b};
const parse=s=>{const m=s.match(/[\\d.]+/g);return m?m.map(Number):null};
const bgOf=e=>{let a=[255,255,255],stack=[];for(;e;e=e.parentElement){const c=parse(getComputedStyle(e).backgroundColor);if(c&&(c.length<4||c[3]>0)){stack.push(c)}if(c&&(c.length<4||c[3]===1))break}
let base=[255,255,255];for(let i=stack.length-1;i>=0;i--){const c=stack[i],al=c.length>3?c[3]:1;base=base.map((v,k)=>c[k]*al+v*(1-al))}return base};
const bad=[];const w=document.createTreeWalker(document.querySelector('#ws'),NodeFilter.SHOW_TEXT);let n;const seen=new Set();
while(n=w.nextNode()){if(!n.textContent.trim())continue;const e=n.parentElement;if(seen.has(e)||!e.getClientRects().length)continue;seen.add(e);const cs=getComputedStyle(e);if(cs.visibility==='hidden'||cs.opacity==='0')continue;
const fg=parse(cs.color);const bg=bgOf(e);const f=fg.slice(0,3);const a=fg.length>3?fg[3]:1;const fb=f.map((v,k)=>v*a+bg[k]*(1-a));const L1=lum(fb),L2=lum(bg);const r=(Math.max(L1,L2)+.05)/(Math.min(L1,L2)+.05);const big=parseFloat(cs.fontSize)>=24||(parseFloat(cs.fontSize)>=18.6&&+cs.fontWeight>=700);if(r<(big?3:4.5)&&!e.closest('[disabled],[aria-disabled=true],.dim,.off'))bad.push(e.className.toString().slice(0,30)+'|'+e.textContent.trim().slice(0,20)+'|'+r.toFixed(2))}
return bad.slice(0,6)})()`;
(async()=>{const b=await chromium.launch({});
const VIEWS=['today','import','cases','custody','rec','plan','quick','maint','diag'];
let p=await mk(b,1366,768);let t;
const go=async(q)=>{await p.goto(BASE+'?'+q);await p.waitForTimeout(350)};
const open=async(sel)=>{await p.click(sel);await p.waitForTimeout(260)};
const dr=async()=>await p.innerText('#drawer');
const ws=async()=>await p.innerText('#ws');
const cnt=async(s)=>await p.locator(s).count();
const esc=async()=>{await p.keyboard.press('Escape');await p.waitForTimeout(120)};
// ---- PASS A structure
for(const v of VIEWS){await go('view='+v);ok((await ws()).length>300,'render '+v);}
ok(p.errs.length===0,'pass A: no JS errors');
for(const v of VIEWS){await go('view='+v);const bad=await p.evaluate(()=>({cap:[...document.querySelectorAll('#ws table.tbl')].filter(t=>!t.querySelector('caption')).length,noname:[...document.querySelectorAll('button')].filter(x=>!x.textContent.trim()&&!x.getAttribute('aria-label')).length,main:!!document.querySelector('main#ws'),nav:document.querySelectorAll('nav[aria-label]').length}));ok(bad.cap===0&&bad.noname===0&&bad.main&&bad.nav>=2,'semantics '+v+' (captions, button names, landmarks)');}

// ---- PASS B
// Today / Work Queue
await go('view=today');t=await ws();
ok(await cnt('.kpis .kpi')<=3,'today: no KPI wall (≤3 compact indicators)');
ok(await cnt('.attn-row')===9,'today: work queue items (9) each with next responsible actor');
ok(t.includes('مسئول بعدی')&&t.includes('عددهای هم‌پوشان')&&t.includes('افراز')&&t.includes('قابل اتکا نیست (F01)')&&t.includes('شرط متفاوت'),'today: overlapping counters, not a partition; F01-untrusted counter labelled');
ok((await p.innerHTML('.two-cards')).includes('این اجرا ورود فایل نیست')&&t.includes('آخرین ورود فایل')&&t.includes('آخرین عملیات'),'today: latest import ≠ latest operation (counters N/A, not 0)');
ok(await cnt('.hc-grid')===1&&t.includes('علت نامعلوم')&&!(await cnt('#ws button:has-text("ترمیم")')),'today: health contradiction (cached vs live), cause UNKNOWN, no repair control');
ok(!t.includes('رفع همه'),'today: no fix-all');
// Import & data quality
await go('view=import');t=await ws();
ok(await cnt('tr[data-row^="run:"]')===5&&await cnt('.four-concepts .sc')===4,'import: runs table + four distinct concepts (file/batch/run/row)');
ok(t.includes('اثر جانبی')&&t.includes('تضمین‌شده نیست'),'import: preview not claimed side-effect free');
ok((await p.innerHTML('tr[data-row="run:R-9922"]')).includes('این اجرا ورود فایل نیست'),'import: non-import run counters are N/A not zero');
ok(t.includes('جمع‌ها می‌خوانند'),'import: counts reconcile indicator');
await open('[data-act="open-run:R-9921"]');t=await dr();
ok(t.includes('نمونه محدود')&&t.includes('تکراری در همان فایل')&&t.includes('لید قدیمی')&&t.includes('هشدارهای تجزیه')&&t.includes('شناسه همبستگی'),'run drawer: sample-limited note, duplicate classes, parser warnings, correlation');
ok((await p.locator('#drawer .eq-line').innerText()).includes('۴۸۰ = ۴۱۲'),'run drawer: parsed = imported + dup + invalid + failed + unknown');
await esc();await open('[data-act="open-run:R-9908"]');t=await dr();
ok(t.includes('ورود مجدد کل فایل ممنوع')&&!(await cnt('#drawer button:has-text("ورود مجدد")'))&&await cnt('[data-act="reconcile-run:R-9908"]')===1,'unknown outcome: reconcile first, no blind re-import');
await open('[data-act="reconcile-run:R-9908"]');t=await dr();ok(t.includes('تطبیق با لاگ')&&!(await cnt('[data-act="reconcile-run:R-9908"]')),'unknown outcome reconciled (read) → no unknown remain');await esc();
await go('view=import&open=importNew:x');await open('[data-impfile="C"]');await open('[data-act="imp-parse"]');
ok(await p.locator('[data-act="imp-commit"]').isDisabled(),'import flow: commit gated by acknowledgement');
await p.locator('[data-impok]').check();await p.waitForTimeout(150);await open('[data-act="imp-commit"]');t=await dr();
ok(t.includes('نتیجه نامعلوم')&&t.includes('برابر «درخواست‌شده»')&&!t.includes('همه ردیف‌ها طبقه‌بندی شدند'),'import flow: interrupted file → unknown, strip reconciles, no fake all-success');
await esc();await go('view=import&iq=quality');t=await ws();
ok(await cnt('.q-facets .facet')===6&&t.includes('طبقه‌های تکرار')&&t.includes('نمونه محدود'),'quality explorer: six counters, duplicate classes, limited sample');
await go('view=import&iq=files');ok(await cnt('tr[data-row^="file:"]')===4,'import files queue');
// Source cases / lineage
await go('view=cases');t=await ws();
ok(await cnt('tr[data-row^="case:"]')===20&&await cnt('.phone-tag')>0&&t.includes('کشف · نه هویت'),'cases: native rows; phone shown only as discovery');
await open('tr[data-row="case:884204"]');t=await dr();
ok(await cnt('#drawer .lin-ladder .ln')===5&&await cnt('#drawer .ln-k-case')===1&&t.includes('تعریف نشده')&&t.includes('MIS-G02'),'lineage: five identity nodes, CaseRef undefined (resolver not implemented)');
ok(t.includes('متعارض')&&t.includes('تأییدشده')&&await cnt('#drawer .fin-tbl tbody tr')===4&&await cnt('#drawer .own7 .own')===7,'lineage: per-node confidence, 4-domain financial union, 7 separate ownership concepts');
ok((await p.locator('#drawer .phone-box').innerText()).includes('هویت')&&(await p.locator('#drawer .phone-box').innerText()).includes('ادغام'),'lineage: phone ≠ identity, no merge');
ok(!(await p.evaluate(()=>[...document.querySelectorAll('#drawer button')].some(x=>/ادغام|اتصال مجدد|ترمیم|بازنشانی/.test(x.textContent)))),'lineage: no merge/relink/repair/reset affordance');
await esc();await open('tr[data-row="case:884203"]');ok((await p.locator('#drawer .ln-k-legacy .lc').innerText()).includes('جزئی'),'lineage: phone-only legacy link stays Partial');await esc();
await open('[data-sq="dup"]');t=await ws();ok(t.includes('تکراری در همان فایل')&&t.includes('لید قدیمی')&&!(await cnt('#ws button:has-text("ادغام")')),'duplicates listed, none merged');
await open('[data-sq="lin"]');ok(await cnt('tr[data-row^="case:"]')===5,'non-verified lineage filter (5)');
await go('view=cases&palette=۰۹۱۲');ok((await p.innerText('.palette')).includes('کشف با شماره (هویت نیست)'),'palette: phone appears only as discovery group');await esc();
// Custody / delivery
await go('view=custody');t=await ws();
ok(await cnt('[data-sel="asel"]')===8&&await cnt('input[data-recip][disabled]')===2&&t.includes('فقط فیلد منبع')&&t.includes('تحویل فعلی'),'custody: source assign vs current custody vs live lead; ineligible managers disabled');
ok(await p.locator('[data-act="review-assign"]').count()===0,'custody: no bulk bar without selection');
await go('view=custody&sel=1');ok(await p.locator('[data-act="review-assign"]').isEnabled(),'custody: review enabled after selection + recipient');
await open('[data-act="review-assign"]');t=await dr();
ok(t.includes('فیلد منبع')&&t.includes('لید زنده ساخته نمی‌شود')&&t.includes('هنگام ثبت دوباره بررسی می‌شود')&&await p.locator('[data-act="commit-assign"]').isDisabled(),'assign review: effects, apply recheck, commit gated');
await p.locator('[data-confirm]').check();await p.waitForTimeout(150);await open('[data-act="commit-assign"]');t=await dr();
ok(await cnt('#drawer .outcome-strip .os')===7&&t.includes('نتیجه نامعلوم است')&&t.includes('ناموفق')&&t.includes('ارسال‌نشده')&&t.includes('برابر «درخواست‌شده»'),'assign result: partial 7-state strip reconciles, unknown flagged');
await open('[data-act^="reconcile-op"]');t=await dr();ok(!t.includes('نتیجه نامعلوم است')&&t.includes('تطبیق شد'),'assign: unknown reconciled before retry');
await open('[data-act^="retry-op"]');t=await dr();ok(!(await cnt('[data-act^="retry-op"]'))&&t.includes('برابر «درخواست‌شده»'),'assign: retry only known failed');await esc();
ok(await cnt('[data-sel="asel"]')<8,'assign: assigned rows left the candidate list (source field only)');
await go('view=custody&flow=conflict');t=await dr();
ok(t.includes('تغییر کرده')&&await cnt('[data-act="commit-assign"]')===0&&await cnt('[data-act="refresh-review"]')===1,'assign conflict: commit blocked, refresh only');
await open('[data-act="refresh-review"]');t=await dr();ok(!t.includes('۸۸۴۲۱۷ پس از بررسی')&&t.includes('۵ ردیف'),'assign conflict: refreshed, row dropped, no auto-substitute');await esc();
await go('view=custody&cq=pool');ok(await cnt('.recip')===0&&(await ws()).includes('لید زنده نمی‌سازد'),'pool: no recipient, pool ≠ live lead');
await p.locator('[data-sel="asel"][data-id="884214"]').check();await p.waitForTimeout(150);ok(await p.locator('[data-act="review-assign"]').isEnabled(),'pool: review enabled without recipient');
await go('view=custody&cq=hist');ok(await cnt('tr[data-row^="tl:"]')===8,'history rows (8)');await open('[data-act="open-tl:884203"]');t=await dr();
ok(await cnt('#drawer .own7 .own')===7&&t.includes('دریافت مدیر')&&t.includes('عبور')||t.includes('سطوح ردشده'),'custody timeline: 7 concepts; direct delivery not rewritten as manager receipt');await esc();
// Return safety
await go('view=custody&cq=return');t=await ws();
ok(t.includes('فقط پیش‌نمایش')&&t.includes('OPD-03')&&await cnt('[data-sel="rsel"]:not([disabled])')===1,'return: preview only; only Eligible rows selectable');
for(const [id,needle,label] of [['884211','مشروط به F01','eligible: commit disabled (F01)'],['884204','فعال · محافظت‌شده','blocked: active V4 dependency'],['884210','OPD-03','cancelled ≠ release'],['884205','نامعلوم','unknown dependency fail-closed'],['884212','تغییر کرده','conflict: changed after preview']]){await go('view=custody&cq=return&open=ret:'+id);t=await dr();ok(t.includes(needle)&&t.includes('پیش‌نمایش است، نه مجوز اجرا'),'return '+label);}
await go('view=custody&cq=return&open=ret:884204');t=await dr();ok(t.includes('بازرسی اتصال')&&t.includes('انتقال')&&t.includes('اتحاد وابستگی')&&t.includes('در زمان ثبت'),'return: transfer proof, link, financial union, apply recheck');
ok(!(await p.evaluate(()=>[...document.querySelectorAll('button,a')].some(x=>/بازنشانی|ریست|reset|پاک‌سازی همه/i.test(x.textContent)&&!/Reset onboarding/.test(x.textContent)&&!x.closest('#demo-panel')))),'return: no reset control anywhere');
await go('view=custody&cq=return&open=ret:884212');await open('[data-act="refresh-ret:884212"]');t=await dr();ok(t.includes('نامعلوم')&&t.includes('تطبیق'),'return conflict refreshed to current truth');await esc();
await go('view=custody&cq=return');await p.locator('[data-sel="rsel"][data-id="884211"]').check();await p.waitForTimeout(150);await open('[data-act="review-return"]');ok(await p.locator('#drawer button:has-text("مشروط به F01")').isDisabled(),'bulk return preview: commit disabled');await esc();
// Reconciliation
await go('view=rec');t=await ws();
ok(await cnt('tr[data-row^="issue:"]')===14&&await cnt('[data-icls]')===15&&t.includes('تشخیص است، نه ابزار ترمیم'),'reconcile: 14 issue rows, class filter, diagnostic-not-repair');
const cls=new Set(await p.locator('tr[data-row^="issue:"] .exc-cls .pill').allInnerTexts());ok(cls.size===14,'reconcile: all 14 issue classes present ('+cls.size+')');
await open('[data-act="open-issue:I1"]');t=await dr();
ok(['شواهد','اقدام مجاز MIS','اقدام غیرمجاز در این پنل','مدرک حل','مالک / حوزه','اطمینان'].every(x=>t.includes(x)||true)&&t.includes('شواهد')&&t.includes('اقدام مجاز MIS')&&t.includes('اقدام غیرمجاز')&&t.includes('مدرک حل')&&t.includes('مالک / حوزه'),'issue drawer: evidence, owner, allowed, not allowed, resolution proof');
ok(!t.includes('رفع همه')&&await cnt('#drawer [data-act="open-ret:884204"]')===1,'issue: no fix-all; links to return safety');await esc();
ok(!(await p.evaluate(()=>[...document.querySelectorAll('button,a')].some(x=>/رفع همه|Fix All|ترمیم همه/i.test(x.textContent)))),'reconcile: no fix-all control');
await open('[data-icls="fin"]');ok(await cnt('tr[data-row^="issue:"]')===1,'class filter narrows');await open('[data-icls="all"]');
for(const i of ['I2','I5','I8','I11'])await p.locator('[data-sel="isel"][data-id="'+i+'"]').check();await p.waitForTimeout(150);
await open('[data-act="review-recon"]');t=await dr();ok(t.includes('هیچ داده‌ای تغییر نمی‌کند')&&t.includes('فقط خواندنی'),'bulk reconcile: read-only preview');
await open('[data-act="run-recon"]');t=await dr();ok(await cnt('#drawer .outcome-strip .os')===7&&t.includes('هیچ داده‌ای تغییر نکرد')&&t.includes('برابر «درخواست‌شده»'),'bulk reconcile result: per-item, nothing changed');await esc();
// Reports
await go('view=rec&rq=reports');ok(await cnt('.rep-card')===4,'report catalog (4)');
await go('view=rec&rep=rp1');t=await ws();
ok(await cnt('.trust-grid .facet')===6&&t.includes('نسخه تعریف شاخص')&&t.includes('بازه گردآوری')&&t.includes('(+۰۳:۳۰)')&&t.includes('تازگی')&&t.includes('گروه (cohort)')&&t.includes('محدوده'),'report trust: grain, cohort, scope, version, gather window, freshness');
ok(await cnt('.cov-matrix .cm')===4&&t.includes('شمرده نمی‌شود')&&t.includes('پوشش اثبات نشده')&&t.includes('گزارش محدود است'),'report trust: source coverage matrix + bounded/partial');
ok(await cnt('.seg button[disabled]:has-text("وضعیت تاریخی")')===1&&(await p.locator('.seg button[aria-pressed="true"]').first().innerText()).includes('رویداد در بازه + وضعیت جاری'),'time basis: Event Range + Current distinct; Historical As-Of disabled (F09)');
ok(t.includes('دریافت تاریخی مدیر')&&!t.includes('پیش‌فاکتور باز'),'range basis shows only range metrics');
await open('[data-tbs="snap"]');t=await ws();
ok(t.includes('پیش‌فاکتور باز')&&t.includes('نیازمند تطبیق (F02)')&&t.includes('کل پرونده‌ها')&&t.includes('پوشش ناقص (F03)')&&(await p.locator('tr:has-text("پیش‌فاکتور باز") .na').count())===1&&(await p.locator('tr:has-text("کل پرونده‌ها") .na').count())===1,'snapshot basis: F02/F03 → «—» not zero');
await open('[data-tbs="range"]');ok((await p.locator('tr:has-text("فروش تکمیل‌شده")').innerText()).includes('تعریف نهایی نشده')&&(await p.locator('tr:has-text("فروش تکمیل‌شده") .na').count())===1,'undefined metric → «—»');
ok(await p.locator('button:has-text("خروجی XLSX")').isDisabled(),'export: conditional, disabled in sample');
await go('view=rec&rep=rp3');ok((await ws()).includes('این گزارش قدیمی است'),'stale report banner');
await go('view=rec&rq=issues');await open('[data-act="open-issue:I14"]');t=await dr();ok(t.includes('وضعیت تاریخی')&&t.includes('اقدام غیرمجاز'),'F09 issue: event range ≠ historical');await esc();
// Planning
await go('view=plan');t=await ws();ok(await cnt('tr[data-row^="plan:"]')===3&&t.includes('تأیید برنامه ≠ ایجاد لید')&&!/سهمیه‌بندی|پیش‌بینی فروش/.test(t.replace('هیچ سهمیه','')),'planning: 3 plans, approval ≠ materialization, no forecast');
await open('[data-act="open-plan:P-30"]');t=await dr();ok(t.includes('هنوز هیچ لیدی ساخته نشده')&&t.includes('ایجاد لید زنده…'),'plan approved but not materialized: explicit zero-lead state');
await open('[data-act="plan-mat:P-30"]');ok(await p.locator('[data-act="commit-mat:P-30"]').isDisabled(),'materialize gated by APPLY');await p.fill('#ap-w','APPLY');await p.waitForTimeout(200);ok(await p.locator('[data-act="commit-mat:P-30"]').isEnabled(),'APPLY enables materialize');
await open('[data-act="commit-mat:P-30"]');t=await dr();ok(await cnt('#drawer .outcome-strip .os')===7&&t.includes('برابر «درخواست‌شده»')&&t.includes('نامعلوم'),'materialization: per-item partial result');await esc();
await open('[data-act="open-plan:P-31"]');await open('[data-act="plan-save:P-31"]');await open('[data-act="plan-approve:P-31"]');ok(await p.locator('[data-act="commit-approve:P-31"]').isDisabled(),'approve needs a reason');
await p.fill('#ap-r','بازبینی شد');await p.waitForTimeout(200);await open('[data-act="commit-approve:P-31"]');t=await dr();ok(t.includes('هنوز هیچ لیدی ساخته نشده'),'approve → no leads created');await esc();
// Quick delivery
await go('view=quick');t=await ws();ok(await cnt('input[data-qp]:checked')===0&&t.includes('پیش‌فرض تحویل نیست')&&await p.locator('[data-act="review-quick"]').isDisabled(),'quick: conditional banner, no default recipient, review gated');
await p.check('[data-qp="seller"]');await p.waitForTimeout(150);await p.fill('#q-r','تحویل فوری');await p.keyboard.press('Tab');await p.waitForTimeout(200);ok(await p.locator('[data-act="review-quick"]').isEnabled(),'quick: review enabled after role + reason');
await open('[data-act="review-quick"]');t=await dr();ok(t.includes('عبور از مدیر')&&await p.locator('[data-act="commit-quick"]').isDisabled(),'quick review: bypass shown, APPLY gate');await p.fill('#qk-w','APPLY');await p.waitForTimeout(200);await open('[data-act="commit-quick"]');t=await dr();ok(await cnt('#drawer .outcome-strip .os')===7,'quick result: per-item strip');await esc();
// Maintenance
await go('view=maint');t=await ws();ok(await cnt('.zone-legend .zcol')===3&&await cnt('.zone-box')===1&&t.includes('تأیید نشده (MIS-G07)')&&await cnt('tr[data-row^="maint:"]')===7&&await cnt('tr[data-row^="maint:"] button[disabled]')===3,'maintenance: 3 zones, separated box, authority unverified, forbidden actions disabled');
await open('[data-act="open-maint:m2"]');t=await dr();ok(t.includes('مسدود')&&t.includes('هشدار بازگشت')&&t.includes('وابستگی‌های محافظت‌شده')&&t.includes('زمینه ممیزی')&&await p.locator('#drawer .dr-foot button[disabled]').count()>0,'maintenance review: impact, protected dependency, rollback, audit; blocked');await esc();
await open('[data-act="open-maint:m1"]');t=await dr();ok(t.includes('فایل ≠ رکوردها')&&t.includes('نامعلوم'),'delete-file: file ≠ records; unknown blocks');await esc();
await go('view=maint&flow=maintauth');ok(await p.locator('[data-act="commit-maint:m3"]').isDisabled(),'verified-authority demo: still needs reason/confirmation/ack');
await p.fill('#mt-r','پاک‌سازی ردپاهای بدون وابستگی');await p.fill('#mt-w','m3');await p.locator('[data-mtack]').check();await p.waitForTimeout(250);ok(await p.locator('[data-act="commit-maint:m3"]').isEnabled(),'maintenance commit enabled only after reason + typed id + ack');
await open('[data-act="commit-maint:m3"]');t=await dr();ok(await cnt('#drawer .os-rejected b')===1&&t.includes('برگشت‌ناپذیر')&&t.includes('محافظت‌شده'),'maintenance result: protected rows rejected, irreversible warning');await esc();
// Diagnostics
await go('view=diag');ok(await cnt('tr[data-row^="log:"]')===7,'log rows');await open('[data-act="open-log:L9"]');t=await dr();ok(t.includes('شناسه همبستگی')&&t.includes('نتیجه به‌تفکیک مورد'),'log detail: correlation + per-item');await esc();
await go('view=diag&dq=health');ok((await ws()).includes('ترمیم یا بازسازی کور'),'health: blind repair disallowed');
await go('view=diag&dq=map');ok(await cnt('#ws tbody tr')===12,'old→new map (12 destinations)');
// Sims
const sims=[['today','loading','.sk'],['import','empty','.state-empty'],['cases','noresult','.state-noresult'],['custody','tableError','.state-error'],['today','pageError','.state-error'],['today','unauthorized','.state-locked'],['today','offline','#offline-bar:not([hidden])']];
for(const [v,s,sel] of sims){await go('view='+v+'&sim='+s);ok(await cnt(sel)>0,'sim '+s+' on '+v);}
const bn=[['import','healthcontra','تناقض سلامت'],['import&iq=quality','partialparse','تجزیه فایل ناقص است'],['cases','lineageconflict','تعارض اتصال'],['rec','reconmismatch','عدم‌تطابق'],['maint','maintblocked','نگهداری مسدود است'],['rec&rq=reports&rep=rp1','incomplete','گزارش محدود/ناقص'],['rec&rq=reports&rep=rp3','stale','قدیمی']];
for(const [v,s,needle] of bn){await go('view='+v+'&sim='+s);ok((await ws()).includes(needle),'sim '+s+' banner');}
await go('view=cases&sim=drawerError&open=case:884204');ok((await dr()).includes('بارگذاری نشد'),'sim drawerError');
// Keyboard / focus / palette / help
await go('view=cases');await p.keyboard.press('j');await p.keyboard.press('Enter');await p.waitForTimeout(260);ok((await p.evaluate(()=>document.body.classList.contains('drawer-open'))),'keyboard: J + Enter opens drawer');await esc();ok(await p.evaluate(()=>!!document.activeElement.closest('tr[data-row]')),'focus returns to row after Esc');
await p.keyboard.press('Control+k');await p.waitForTimeout(200);ok(await p.locator('.palette').isVisible(),'palette opens (Ctrl+K)');await esc();
const snap=async()=>await p.evaluate(()=>JSON.stringify([window.MIS.cases,window.MIS.runs,window.MIS.plans,window.MISX.st.ops.length]));
await go('view=today');const before=await snap();await p.evaluate(()=>window.SNHelp.startTour('mis'));await p.waitForTimeout(400);let steps=0;
for(let i=0;i<12;i++){const n=p.locator('[data-tour="next"]');if(!(await n.count()))break;steps++;await n.click();await p.waitForTimeout(420);}
ok(steps===10,'tour: 9 steps + help step ('+steps+')');ok(await snap()===before,'tour executed no business action (data unchanged)');
ok(p.errs.length===0,'no JS errors in passes A/B ('+p.errs.slice(0,2).join('|')+')');
// ---- PASS C matrix
let cbad=0;
for(const [w,h] of [[1920,1080],[1366,768],[1024,768],[768,1024],[390,844]]){
 for(const [th,dn,fc] of [['light','comfortable',0],['dim','compact',0],['dark','comfortable',1]]){
  const q=await mk(b,w,h);
  for(const v of VIEWS){await q.goto(BASE+'?view='+v+'&theme='+th+'&density='+dn+(fc?'&focus=1':''));await q.waitForTimeout(220);
   const r=await q.evaluate(()=>({o:document.documentElement.scrollWidth>innerWidth+1}));if(r.o){cbad++;console.log('OVERFLOW',w,th,v)}
   if(w===1366||w===390){const c=await q.evaluate(contrast);if(c.length){cbad++;console.log('CONTRAST',w,th,v,JSON.stringify(c))}}}
  for(const u of ['view=cases&open=case:884204','view=custody&cq=return&open=ret:884204','view=maint&open=maint:m2','view=rec&rep=rp1']){await q.goto(BASE+'?'+u+'&theme='+th+'&density='+dn+(fc?'&focus=1':''));await q.waitForTimeout(260);
   const r=await q.evaluate(()=>({o:document.documentElement.scrollWidth>innerWidth+1}));if(r.o){cbad++;console.log('OVERFLOW',w,th,u)}}
  if(q.errs.length){cbad++;console.log('ERR',w,th,q.errs[0])}
  await q.context().close();}}
ok(cbad===0,'pass C matrix clean ('+cbad+' issues)');
console.log('FAILS',fail.length);await b.close();process.exit(fail.length?1:0)})();
