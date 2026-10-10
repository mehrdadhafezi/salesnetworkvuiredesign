// SN-205 post-merge regression (6 Codex findings on head d20f3c6). Run: python3 -m http.server 8777 (from prototype/), then node hr/qa/qa-postmerge.js — requires playwright.
const { chromium } = require('playwright');
const BASE='http://127.0.0.1:8777/hr/index.html';
const fail=[];const ok=(c,m)=>{console.log((c?'PASS ':'FAIL ')+m);if(!c)fail.push(m)};
async function mk(b,w,h){const ctx=await b.newContext({viewport:{width:w,height:h},reducedMotion:'reduce'});await ctx.addInitScript(()=>{try{localStorage.setItem('snproto.hr_welcome_v1_seen','1')}catch(e){}});const p=await ctx.newPage();p.errs=[];p.on('pageerror',e=>p.errs.push(e.message));return p}
(async()=>{const b=await chromium.launch({});
let p;const fresh=async(q,w=1366,h=768)=>{if(p)await p.context().close();p=await mk(b,w,h);await p.goto(BASE+'?'+(q||'view=work'));await p.waitForTimeout(350)};
const ev=(f,a)=>p.evaluate(f,a);

// ===== F1 hierarchy: closure confirmed + insert Unknown => assignment Unknown, never the previous manager
await fresh('view=work');
let r=await ev(()=>{const X=HRX,p=X.s('HP-301'),n0=p.hist.length,prev=p.parent;X.st.flow='xunknown';const op=X.commitXfer(X.xferSpec(p,X.s('HP-120')));window.__op=op;
 return {n0,prev,n1:p.hist.length,closed:!!p.hist[1].to,top:p.hist[0].state,topParent:p.hist[0].parent,parent:p.parent,unk:p.parentUnknown,txt:X.mgrText(p),known120:p.hist.filter(h=>h.parent==='HP-120'&&h.state!=='unknown'&&!h.to).length,st:X.opState(op),aud:op.audit.after}});
ok(r.closed&&r.n1===r.n0+1&&r.top==='unknown','F1: old interval closure preserved; unknown placeholder on top ('+r.n0+'→'+r.n1+')');
ok(r.parent===null&&r.unk===true&&r.txt.includes('نامعلوم')&&r.prev==='HP-210'&&r.known120===0,'F1: current manager not the previous manager and not the target; Unknown (no known new interval)');
ok(r.st==='unknown'&&r.aud.includes('نامعلوم'),'F1: operation stays Unknown (not Failed); audit says effective manager unknown');
await ev(()=>HRAPI.flow('staff')); await ev(()=>{HRX.C.closeDrawer();HRX.C.openDrawer('staff','HP-301')});await p.waitForTimeout(300);
let dt=await p.evaluate(()=>document.querySelector('#drawer').innerText);
ok(/مدیر فعلی[\s\S]{0,40}نامعلوم/.test(dt)&&!/مدیر فعلی\s*\n?\s*(الهام|مهسا)/.test(dt)&&await p.locator('#drawer .ei-unknown').count()>=1,'F1: profile drawer shows manager Unknown + unknown interval; previous manager not shown as current');
await ev(()=>{HRX.C.closeDrawer();HRX.C.go('work')});await p.waitForTimeout(250);
ok((await p.innerText('tr[data-row="staff:HP-301"]')).includes('نامعلوم (UNKNOWN)'),'F1: workforce table cell shows Unknown manager');
r=await ev(()=>{const X=HRX,p=X.s('HP-301'),op=window.__op,n=p.hist.length;X.reconcileOp(op);const a={parent:p.parent,unk:p.parentUnknown,n:p.hist.length,live:p.hist.filter(h=>h.parent==='HP-120'&&!h.to).length,placeholder:p.hist.filter(h=>h.state==='unknown'&&h.src.indexOf('بازه قبلی بسته')>-1).length,n0:n};
 X.reconcileOp(op);X.retryFailed(op);X.settleOp(op);return Object.assign(a,{n2:p.hist.length,live2:p.hist.filter(h=>h.parent==='HP-120'&&!h.to).length})});
ok(r.parent==='HP-120'&&r.unk===false&&r.live===1&&r.placeholder===0,'F1: reconcile → exactly one live interval, placeholder removed');
ok(r.n2===r.n&&r.live2===1,'F1: repeated reconcile/retry/settle adds no duplicate interval ('+r.n+'→'+r.n2+')');
r=await ev(()=>{const X=HRX,p=X.s('HP-302');X.st.flow='xunknown';const op=X.commitXfer(X.xferSpec(p,X.s('HP-120')));op.items[1][1]='failed';op.items[1][2]='x';X.settleOp(op);const a={st:X.opState(op),unk:p.parentUnknown,txt:p.hist[0].src,n:p.hist.length};X.retryFailed(op);X.retryFailed(op);return Object.assign(a,{parent:p.parent,live:p.hist.filter(h=>h.parent==='HP-120'&&!h.to).length,n2:p.hist.length,unk2:p.parentUnknown})});
ok(r.unk&&r.txt.includes('ثبت نشد')&&r.st==='partial','F1: reconcile → «not recorded» keeps closure + manager still undetermined (Failed ≠ Unknown wording)');
ok(r.parent==='HP-120'&&r.live===1&&r.n2===r.n&&!r.unk2,'F1: retry of known-failed insert yields one interval, no duplicate');
r=await ev(()=>{const X=HRX;X.st.flow='applyfail';X.st.reqState={};const rq=X.req('R-502'),sp=X.reqApplySpec(rq);sp.reasonText='x';const op=X.commitReqApply(sp);const a={st:X.reqSt(rq),hr:rq.chain[2][1],parent:X.s('HP-302').parentUnknown};X.reconcileOp(op);return Object.assign(a,{st2:X.reqSt(rq),hr2:rq.chain[2][1]})});
ok(r.st==='failed'&&r.hr==='cur'&&r.parent===true,'F1/F6: request apply with Unknown insert → request failed(Unknown), HR step not closed');
ok(r.st2==='approved'&&r.hr2==='done','F1/F6: reconcile settles request and closes HR step');

// ===== F2 access retry settlement
await fresh('view=acc');
r=await ev(()=>{const X=HRX,M=HR,p=X.s('HP-301');const sp=X.accessSpec(p,'sup');X.st.flow='accpartial';const op=X.commitAccess(sp);const A=M.access['HP-301'];
 return {keys:sp.keys.join(','),team:A.rows.team[1],step:A.rows.step[1],partial:A.partial||'',role:p.role,defTeam:M.access['default'].rows.team[1],extra:A.rows.extra.join('|'),st:X.opState(op),id:op.id}});
ok(r.keys==='team,step'&&r.team==='allow'&&r.step==='deny'&&r.role==='partial'&&r.partial.includes('تأیید مرحله'),'F2: partial apply → confirmed permission synced, failed one keeps previous effective state, partial marker names it');
ok(r.defTeam==='deny'&&r.extra.startsWith('direct|allow'),'F2: shared default rows not mutated; direct exception preserved');
r=await ev(()=>{const X=HRX,M=HR,p=X.s('HP-301'),op=X.st.ops[0];X.retryFailed(op);const A=M.access['HP-301'];const a={step:A.rows.step.join('|'),team:A.rows.team.join('|'),partial:A.partial,role:p.role,st:X.opState(op),extra:A.rows.extra[1]};X.retryFailed(op);X.settleOp(op);return Object.assign(a,{step2:A.rows.step[1],role2:p.role,partial2:A.partial})});
ok(r.step==='position|allow'&&r.team==='position|allow'&&r.partial===undefined&&r.role==='ok'&&r.st==='complete'&&r.extra==='allow','F2: retry → effective rows synchronized to successful outcome; partial marker cleared; role ok');
ok(r.step2==='allow'&&r.role2==='ok'&&r.partial2===undefined,'F2: retry/settle repeated is idempotent');
await ev(()=>HRX.C.go('acc'));await p.waitForTimeout(200);await ev(()=>HRX.C.openDrawer('access','HP-301',{}));await p.waitForTimeout(300);
let t=await p.innerText('#drawer');ok(!t.includes('اعمال دسترسی ناقص:'),'F2: inspector no longer shows partial note after settled retry');
r=await ev(()=>{const X=HRX,M=HR,p=X.s('HP-304');const op={kind:'access',p:p.id,items:[['a','ok',''],['b','ok',''],['c','ok','']]};X.settleOp(op);return {role:p.role,partial:M.access['HP-304'].partial}});
ok(r.role==='partial'&&r.partial.includes('قابل بازسازی نیست'),'F2: effective state not reconstructable → stays partial (never silently ok)');

// ===== F3 termination handover guard
await fresh('view=req');
r=await ev(()=>{const X=HRX,M=HR,out={};const run=(id)=>{const p=X.s(id);const a0=M.audit.length;const op=X.commitTerm(X.termSpec(p));return {blocked:!!op.blocked,emp:X.empOf(p),edit:!!X.st.staffEdit[id],da:M.audit.length-a0,why:op.items[0][2],ok:op.items.filter(i=>i[1]==='ok').length}};
 out.open=run('HP-304');const p=X.s('HP-311');const keep=p.open;p.open.verified=false;out.unver=run('HP-311');p.open.verified=true;delete p.open;out.none=run('HP-311');p.open=keep;p.open.leads=1;out.pos=run('HP-311');p.open.leads=0;
 out.clear=run('HP-311');out.again=run('HP-311');return out});
ok(r.open.blocked&&!r.open.edit&&r.open.emp!=='terminated'&&r.open.ok===0&&r.open.why.includes('کار باز'),'F3: open work > 0 → apply refused, nothing written');
ok(r.unver.blocked&&r.unver.why.includes('تأیید نشده')&&!r.unver.edit,'F3: unverified snapshot → fail closed');
ok(r.none.blocked&&r.none.why.includes('شواهد')&&!r.none.edit,'F3: missing evidence → fail closed');
ok(r.pos.blocked&&!r.pos.edit,'F3: protected/open total non-zero with verified evidence → fail closed');
ok(!r.clear.blocked&&r.clear.emp==='terminated'&&r.clear.edit,'F3: evidence exists + verified + total 0 → applies');
ok(r.again.blocked&&r.again.da===1&&r.again.why.includes('قبلاً'),'F3: second apply is refused (no duplicate settlement); only a blocked-attempt audit row');
await fresh('view=req&rq=review&flow=reqtermblock');await p.waitForTimeout(300);
ok(await p.locator('#drawer [data-act^="req-apply:"]').isDisabled(),'F3: request R-504 final apply disabled (handover open)');
r=await ev(()=>{const X=HRX;X.st.reqState={};const rq=X.req('R-504'),sp=X.reqApplySpec(rq);sp.reasonText='x';const op=X.commitReqApply(sp);return {blocked:!!op.blocked,st:X.reqSt(rq),cur:rq.chain[2][1],emp:X.empOf(X.s('HP-304'))}});
ok(r.blocked&&r.st==='pending_hr'&&r.cur==='cur'&&r.emp!=='terminated','F3: forced request apply is refused; request stays pending_hr, chain untouched');

// ===== F4 compensation temporal validation
await fresh('view=comp');
r=await ev(()=>{const X=HRX,c=HR.comp['HP-301'];return ['۱۴۰۵/۰۴/۰۱','۱۴۰۴/۰۱/۰۱','۱۴۰۵/۰۳/۱۵','۱۴۰۵/۰۸/۰۱','1405/08/01','۱۴۰۵/۱۳/۰۱','abc',''].map(d=>X.compTemporal(c,d).ok)});
ok(JSON.stringify(r)==='[false,false,false,true,true,false,false,false]','F4: same-day, backdated, overlapping-history, malformed and empty dates invalid; later date valid');
await ev(()=>HRAPI.flow('comp'));await p.waitForTimeout(350);
const fillC=async(d)=>{await p.fill('#cp-b','۳۴٬۰۰۰٬۰۰۰');await p.fill('#cp-r','بازبینی');await p.fill('#cp-f',d);await p.waitForTimeout(150)};
await fillC('۱۴۰۵/۰۱/۰۱');
ok(await p.locator('#drawer [data-act^="comp-review:"]').isDisabled()&&await p.locator('#cp-f[aria-invalid="true"]').count()===1&&await p.locator('#cp-fe').count()===1,'F4: backdated date → commit disabled, field aria-invalid + message');
await fillC('۱۴۰۵/۰۸/۰۱');
ok(!(await p.locator('#drawer [data-act^="comp-review:"]').isDisabled())&&await p.locator('#cp-fe').count()===0,'F4: valid later date enables commit');
r=await ev(()=>{const X=HRX,c=HR.comp['HP-303'],n=c.periods.length,cur0=JSON.stringify(c.periods[0]);const sp=X.compSpec(X.s('HP-303'),{base:'۱',from:'۱۴۰۵/۰۲/۰۱'});sp.reasonText='x';const op=X.commitComp(sp);return {blocked:!!op.blocked,n:c.periods.length===n,same:JSON.stringify(c.periods[0])===cur0,open:!c.periods[0].to}});
ok(r.blocked&&r.n&&r.same&&r.open,'F4: forced commit with invalid range refused; current interval not rewritten');
r=await ev(()=>{const X=HRX,c=HR.comp['HP-301'];X.st.flow='compunknown';const sp=X.compSpec(X.s('HP-301'),{base:'۳۴٬۰۰۰٬۰۰۰',from:'۱۴۰۵/۰۸/۰۱'});sp.reasonText='x';const n=c.periods.length;X.commitComp(sp);const a={n1:c.periods.length-n,closed:!!c.periods[0].to};const op2=X.commitComp(sp);return Object.assign(a,{blocked2:!!op2.blocked,n2:c.periods.length-n})});
ok(r.n1===0&&r.closed&&r.blocked2&&r.n2===0,'F4: Unknown insert keeps closure; blind re-commit blocked (no overlap/duplicate period)');
r=await ev(()=>{const X=HRX,c=HR.comp['HP-302'];X.st.flow=null;const sp=X.compSpec(X.s('HP-302'),{base:'۳۰٬۰۰۰٬۰۰۰',from:'۱۴۰۵/۰۹/۰۱'});sp.reasonText='x';const op=X.commitComp(sp);return {n:c.periods.length,cur:!c.periods[0].to,old:c.periods[1].to,st:X.opState(op)}});
ok(r.n===2&&r.cur&&r.old&&r.st==='complete','F4: valid commit closes old and adds exactly one period');

// ===== F5 persistent impersonation boundary
const VIEWS=['work','onb','req','acc','comp','exc','cred','bulk','diag'];
await fresh('view=work&imp=1');
let allv=true;for(const v of VIEWS){await p.evaluate(x=>HRX.C.go(x),v);await p.waitForTimeout(120);const s=await p.evaluate(()=>{const n=document.querySelector('#imp-shell .imp-bar');if(!n)return null;const r=n.getBoundingClientRect();return {y:r.top,vis:r.height>20&&getComputedStyle(n).visibility!=='hidden',btn:!!n.querySelector('[data-act="imp-end"]'),n:document.querySelectorAll('.imp-bar').length}});if(!s||s.y<0||s.y>60||!s.vis||!s.btn||s.n!==1){allv=false;console.log('imp missing on',v,JSON.stringify(s))}}
ok(allv,'F5: warning boundary + Return control visible (once) on all 9 HR views');
await p.evaluate(()=>HRX.C.go('work'));await p.evaluate(()=>window.scrollTo(0,900));await p.waitForTimeout(150);
ok(await p.evaluate(()=>{const r=document.querySelector('#imp-shell .imp-bar').getBoundingClientRect();return r.top>=0&&r.top<2}),'F5: boundary stays pinned while scrolling');
await p.evaluate(()=>HRX.C.openDrawer('staff','HP-303'));await p.waitForTimeout(300);
ok(await p.evaluate(()=>{const n=document.querySelector('#imp-shell .imp-bar').getBoundingClientRect(),d=document.querySelector('#drawer').getBoundingClientRect();const el=document.elementFromPoint(n.left+20,n.top+n.height/2);return n.height>20&&d.top>=n.bottom-1&&!!el.closest('#imp-shell')}),'F5: boundary stays visible above an open drawer (drawer shifted below it)');
await p.keyboard.press('Escape');
await p.evaluate(()=>HRX.C.go('exc'));await p.goto(BASE+'?view=req&sim=pageError&imp=1');await p.waitForTimeout(300);
ok(await p.locator('#imp-shell .imp-bar [data-act="imp-end"]').count()===1,'F5: boundary also visible in page-error simulation');
await p.goto(BASE+'?view=req&imp=1');await p.waitForTimeout(250);await p.click('#imp-shell [data-act="imp-end"]');await p.waitForTimeout(200);
ok(await p.locator('.imp-bar').count()===0&&await p.evaluate(()=>!document.body.classList.contains('imp-on')),'F5: Return ends session from any view; boundary removed');
// start via real flow then navigate away
await fresh('view=cred');await p.evaluate(()=>HRAPI.flow('imp'));await p.waitForTimeout(300);await p.locator('[data-purpose="0"]').check();await p.locator('[data-ack0]').check();await p.click('[data-act^="imp-review:"]');await p.waitForTimeout(250);await p.fill('#sens-r','پشتیبانی');await p.locator('[data-sensack]').check();await p.click('[data-act="commit-sens"]');await p.waitForTimeout(350);
await p.evaluate(()=>HRX.C.go('comp'));await p.waitForTimeout(200);
ok(await p.locator('#imp-shell .imp-bar').count()===1&&(await p.innerText('#imp-shell')).includes('زهرا کریمی'),'F5: started via real flow, navigating away keeps the boundary');
for(const [w,h] of [[390,844],[768,1024],[1366,768]]){await fresh('view=work&imp=1',w,h);const o=await p.evaluate(()=>({ov:document.documentElement.scrollWidth>innerWidth+1,sh:document.querySelector('#imp-shell').scrollWidth>innerWidth+1}));ok(!o.ov&&!o.sh,'F5: no horizontal overflow with boundary at '+w);}

// ===== F6 request reviewer-chain closure
await fresh('view=req');
r=await ev(()=>{const X=HRX;X.st.reqState={};X.st.flow=null;const rq=X.req('R-502'),pre=rq.chain.map(c=>c[2]);const sp=X.reqApplySpec(rq);sp.reasonText='x';const op=X.commitReqApply(sp);const a={st:X.reqSt(rq),states:rq.chain.map(c=>c[1]).join(','),cur:rq.cur,next:rq.next,keeps:rq.chain.every((c,i)=>c[2].indexOf(pre[i])===0),applied:!!rq.appliedAt,len:rq.chain.length,txt:rq.chain[2][2]};
 X.settleOp(op);X.retryFailed(op);X.reconcileOp(op);return Object.assign(a,{txt2:rq.chain[2][2],len2:rq.chain.length,hist:X.s('HP-302').hist.filter(h=>h.parent==='HP-120'&&!h.to).length})});
ok(r.st==='approved'&&r.states==='done,done,done'&&r.cur==='—'&&r.next==='—'&&r.applied,'F6: transfer settled → active HR step completed, current/next cleared');
ok(r.keeps&&r.len===3&&r.txt.includes('اعمال نهایی'),'F6: provenance preserved (original entries kept, closure appended)');
ok(r.txt2===r.txt&&r.len2===3&&r.hist===1,'F6: idempotent — no double closure text, no duplicate interval on repeat settle/retry/reconcile');
r=await ev(()=>{const X=HRX,M=HR;M.requests.push({id:'R-T1',type:'terminate',subj:'HP-311',by:'t',reason:'t',to:null,st:'pending_hr',cur:'منابع انسانی',next:'—',chain:[['درخواست‌دهنده','done','t'],['تأیید مرحله‌ای','done','t'],['منابع انسانی','cur','اعمال نهایی']],eff:null});const rq=X.req('R-T1'),sp=X.reqApplySpec(rq);sp.reasonText='x';const op=X.commitReqApply(sp);return {op:!op.blocked,st:X.reqSt(rq),states:rq.chain.map(c=>c[1]).join(','),cur:rq.cur,next:rq.next}});
ok(r.op&&r.st==='approved'&&r.states==='done,done,done'&&r.cur==='—'&&r.next==='—','F6: terminate request settled → chain closed, reviewers cleared');
await p.evaluate(()=>{HRX.st.rq='closed';HRX.C.go('req');HRX.C.openDrawer('req','R-502')});await p.waitForTimeout(300);
t=await p.innerText('#drawer');ok(/بررسی‌کننده فعلی\s*\n?\s*—/.test(t)&&!/hourglass/.test(await p.innerHTML('#drawer'))&&await p.locator('#drawer .chain .c-cur').count()===0,'F6: request drawer shows no active reviewer step after settlement');


// ===== Codex round 2 (head 06011df)
await fresh('view=req');
r=await ev(()=>{const X=HRX,p=X.s('HP-311');const k=p.open;const run=()=>{const op=X.commitTerm(X.termSpec(p));return {b:!!op.blocked,e:X.empOf(p)}};const out={};
 p.open={verified:true,leads:0,inv:0};out.partial=run();p.open={verified:true,leads:0,inv:0,cust:0,tasks:NaN};out.nan=run();p.open={verified:true,leads:0,inv:0,cust:0,tasks:-1};out.neg=run();p.open=k;return out});
ok(r.partial.b&&r.nan.b&&r.neg.b&&r.partial.e!=='terminated','R2: verified snapshot with missing/NaN/negative counts → fail closed (missing ≠ zero)');
r=await ev(()=>{const X=HRX,p=X.s('HP-304');p.role='ok';const op={kind:'access',p:p.id,items:[['a','ok',''],['b','ok','']]};op.audit={};X.settleOp(op);return {s:X.opState(op),a:op.audit.res,role:p.role}});
ok(r.s==='partial'&&r.a==='partial'&&r.role==='partial','R2: unreconstructable access op reports partial in state and audit (not complete)');
r=await ev(()=>{const X=HRX,p=X.s('HP-301');const a0=HR.audit.length;const hl=p.hist.length;X.st.flow='xunknown';const op=X.commitXfer(X.xferSpec(p,X.s('HP-120')));const n1=p.hist.length;X.st.flow=null;const op2=X.commitXfer(X.xferSpec(p,X.s('HP-110')));const a={blocked:!!op2.blocked,same:p.hist.length===n1,unk:p.parentUnknown,parent:p.parent,st:X.opState(op2),aud:HR.audit[0].res};X.reconcileOp(op);return Object.assign(a,{after:p.parent,live:p.hist.filter(h=>!h.to).length})});
ok(r.blocked&&r.same&&r.unk&&r.parent===null&&r.st==='blocked'&&r.aud==='blocked','R2: second transfer while manager Unknown is blocked, no mutation; state/audit = blocked');
ok(r.after==='HP-120'&&r.live===1,'R2: original op reconcile still settles to the first target (one live interval)');
await ev(()=>{HRX.C.closeDrawer();HRX.C.go('onb');HRX.C.openDrawer('xfer','HP-301',{target:'HP-110'})});await p.waitForTimeout(250);
r=await ev(()=>{const X=HRX,p=X.s('HP-302');X.st.flow='xunknown';X.commitXfer(X.xferSpec(p,X.s('HP-120')));X.st.flow=null;HRX.C.closeDrawer();HRX.C.openDrawer('xfer','HP-302',{target:'HP-110'});return 1});await p.waitForTimeout(250);
ok(await p.locator('#drawer [data-act^="xfer-review:"]').isDisabled(),'R2: transfer preview continue disabled while manager Unknown');
r=await ev(()=>{const X=HRX;return ['abc۱۴۰۵/۰۸/۰۱','۱۴۰۵/۰۸/۰۱-extra','x1405/08/01','۱۴۰۵/۰۸/۰۱','ماقبل ۱۴۰۵/۰۸/۰۱'].map(d=>X.dateKey(d)!=null)});
ok(JSON.stringify(r)==='[false,false,false,true,false]','R2: user date must match the whole string (no prefix/suffix, no internal marker)');
await fresh('view=comp');
r=await ev(()=>{const X=HRX,c=HR.comp['HP-301'];X.st.flow='compunknown';const sp=X.compSpec(X.s('HP-301'),{base:'۳۴٬۰۰۰٬۰۰۰',from:'۱۴۰۵/۰۸/۰۱'});sp.reasonText='x';const op=X.commitComp(sp);const a={mark:c.unresolved===op.id,st:X.opState(op)};X.st.flow=null;const sp2=X.compSpec(X.s('HP-301'),{base:'۱',from:'۱۴۰۶/۰۱/۰۱'});sp2.reasonText='x';const n=c.periods.length;const op2=X.commitComp(sp2);Object.assign(a,{blocked:!!op2.blocked,aud:HR.audit[0].res,same:c.periods.length===n,valid:X.compTemporal(c,'۱۴۰۶/۰۱/۰۱').ok});X.reconcileOp(op);return Object.assign(a,{clear:c.unresolved===undefined,n2:c.periods.length-n,okNow:X.compTemporal(c,'۱۴۰۶/۰۱/۰۱').ok})});
ok(r.mark&&r.st==='unknown'&&r.blocked&&r.same&&!r.valid&&r.aud==='blocked','R2: compensation Unknown marks domain; any further write (even later date) blocked, outcome=blocked');
ok(r.clear&&r.n2===1&&r.okNow,'R2: reconcile clears the marker, one period inserted, writes allowed again');
await fresh('view=req&rq=review&flow=reqtermblock');await p.waitForTimeout(300);
t=await p.evaluate(()=>{HRX.X;const op=HRX.commitTerm(HRX.termSpec(HRX.s('HP-304')));return HRX.C.openDrawer('result',op.id),op.id});await p.waitForTimeout(300);
ok((await p.innerText('#drawer')).includes('مسدود'),'R2: result drawer labels guarded no-op as «مسدود», not failed');
for(const w of [390,360]){await fresh('view=work&imp=1',w,700);await p.evaluate(()=>HRX.C.openDrawer('staff','HP-303'));await p.waitForTimeout(350);
 const g=await p.evaluate(()=>{const d=document.querySelector('#drawer').getBoundingClientRect();const b=document.querySelector('#imp-shell').getBoundingClientRect();return {bottom:d.bottom,vh:innerHeight,top:d.top,barBottom:b.bottom}});
 ok(g.bottom<=g.vh+1&&g.top>=g.barBottom-1,'R2: mobile drawer ('+w+') fits below impersonation bar and above viewport bottom');}

// ===== Codex round 3 (head 32d45a6)
await fresh('view=comp');
r=await ev(()=>{const X=HRX,c=HR.comp['HP-303'];return {user:X.compTemporal(c,'ماقبل ۱۴۰۵/۰۸/۰۱').ok,plain:X.compTemporal(c,'۱۴۰۵/۰۸/۰۱').ok,internal:X.dateKey('ماقبل ۱۴۰۵/۰۸/۰۱',true)!=null,strict:X.dateKey('ماقبل ۱۴۰۵/۰۸/۰۱')==null}});
ok(!r.user&&r.plain&&r.internal&&r.strict,'R3: user input «ماقبل …» rejected; internal end-date form parsed only on the internal path');
await ev(()=>{HRX.st.flow='compunknown';const sp=HRX.compSpec(HRX.s('HP-301'),{base:'۳۴٬۰۰۰٬۰۰۰',from:'۱۴۰۵/۰۸/۰۱'});sp.reasonText='x';window.__cop=HRX.commitComp(sp);HRX.st.flow=null;HRX.C.openDrawer('comp','HP-301',{})});await p.waitForTimeout(300);
ok(await p.locator('#drawer [data-act^="open-op:"]').count()===1,'R3: compensation Unknown banner has a control to reopen the operation');
await p.click('#drawer [data-act^="open-op:"]');await p.waitForTimeout(300);
ok(await p.locator('#drawer [data-act^="reconcile-op:"]').count()===1,'R3: control opens the result drawer with reconcile action');
await p.click('#drawer [data-act^="reconcile-op:"]');await p.waitForTimeout(300);
ok(await ev(()=>HR.comp['HP-301'].unresolved===undefined),'R3: reconciling from the reopened result clears the compensation marker');
await fresh('view=work');
await ev(()=>{HRX.st.flow='xunknown';HRX.commitXfer(HRX.xferSpec(HRX.s('HP-301'),HRX.s('HP-120')));HRX.st.flow=null;HRX.C.openDrawer('staff','HP-301')});await p.waitForTimeout(300);
ok(await p.locator('#drawer [data-act^="open-op:"]').count()===1,'R3: profile with Unknown manager has a control to reopen the operation');
await p.click('#drawer [data-act^="open-op:"]');await p.waitForTimeout(300);await p.click('#drawer [data-act^="reconcile-op:"]');await p.waitForTimeout(300);
ok(await ev(()=>HRX.s('HP-301').parentUnknown===false&&HRX.s('HP-301').parent==='HP-120'&&HRX.s('HP-301').unkOp===undefined),'R3: reconcile from profile path settles the hierarchy');
await ev(()=>{HRX.st.flow='xunknown';HRX.commitXfer(HRX.xferSpec(HRX.s('HP-302'),HRX.s('HP-120')));HRX.st.flow=null;HRX.C.closeDrawer();HRX.C.openDrawer('xfer','HP-302',{target:'HP-110'})});await p.waitForTimeout(300);
ok(await p.locator('#drawer [data-act^="open-op:"]').count()===1,'R3: transfer preview offers the same path back to reconciliation');
// ===== cross-cutting
ok(p.errs.length===0,'no JS errors in post-merge pass ('+p.errs.slice(0,2).join('|')+')');
console.log('FAILS',fail.length);await b.close();process.exit(fail.length?1:0)})();
