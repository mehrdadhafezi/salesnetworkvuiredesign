# CRM CROSS-ROLE PRODUCT ARCHITECTURE V1

تاریخ: 2026-10-07 · **PRODUCT ARCHITECTURE / INTEGRATION ONLY** · **FROZEN TARGET — IMPLEMENTATION DEFERRED**

## 1. Executive Summary

**هشت قرارداد نقش می‌توانند در یک CRM منسجم هم‌زیست باشند.** شرط مشترک: identity مبتنی بر proof، مالکیت چندبُعدی، اختیار مستقل هر action، stateهای domain-specific و نتیجه عملیات قابل اثبات. جایگاه بالاتر سازمانی، اختیار Finance/HR/MIS یا تمام writeهای سطوح پایین را به ارث نمی‌برد. نقاط باز در بخش 19 به activation قابلیت‌های مشروط یا validation محدود شده‌اند؛ هیچ سیاست تجاری مجهول به‌جای تصمیم صاحب محصول فرض نشده است.

این Freeze قرارداد WHAT/WHY مشترک است؛ به معنی رفع F01–F10، runtime QA، مجوز deployment یا آماده‌بودن تمام sensitive writes نیست. هیچ Spec نقش بازنویسی نشده؛ هیچ live audit یا خواندن مجدد کد پلاگین انجام نشده است. **KEEP + ADAPT** و تمام **Gate 0 INV-001…INV-036** الزام‌آورند.

Authoritative inputs:

- **A / G:** [Audit پذیرفته‌شده](CRM-PRODUCT-ROLE-ARCHITECTURE-AUDIT.fa.md)، [Gate 0](GATE-0-PRODUCT-INVARIANTS.md).
- **SS / M / DP:** [Senior Spec](SENIOR-SUPERVISOR-PRODUCT-SPEC.md)، [Manager Spec](SALES-MANAGER-PRODUCT-SPEC.md)، [Deputy Spec](SALES-DEPUTY-PRODUCT-SPEC.md).
- **MIS / HR / FIN:** [MIS Spec](MIS-PRODUCT-SPEC.md)، [HR Spec](HR-PRODUCT-SPEC.md)، [Finance Spec](FINANCE-PRODUCT-SPEC.md).
- **C / S / D:** [Code Registry](CODE-EVIDENCE-AND-ACTION-REGISTRY.md)، [Status Catalog](STATUS-TRANSITIONS.fa.md)، [Decisions / Visibility](DECISIONS-VISIBILITY-INTERACTIONS.fa.md). [Cross-role appendix قبلی](CROSS-ROLE-ARCHITECTURE.fa.md) current evidence است؛ این V1 قرارداد integration هدف است، نه replacement تاریخچه Audit.
- Seller/Supervisor: Product Spec مستقل در منابع موجود یافت نشد؛ **با تأیید مستقیم کاربر در این مرحله، قراردادهای پذیرفته‌شده A/G/D مبنای integration این دو نقش هستند**. ادعای بررسی هشت فایل مستقل نمی‌شود. Prototype/Design Spec بصری business authority نیست.
- Customer lifecycle سه‌شاخه بخش 7، دستور صریح همین task و accepted business path است؛ current end-to-end runtime آن از role inventory نتیجه‌گیری نشده است.

Evidence: قرارداد integration **INFERENCE / HIGH** مبتنی بر منابع بالا؛ current facts برچسب پذیرفته‌شده خود را حفظ می‌کنند. LIVE VERIFIED فقط رفتار مشخصِ قبلاً مشاهده‌شده؛ CODE VERIFIED فقط branch/guard؛ BOTH توافق همان claim؛ NOT VERIFIED نبود proof. **HR ROLE RUNTIME NOT LIVE VERIFIED؛ FINANCE ROLE RUNTIME NOT LIVE VERIFIED.** Write execution/negative enforcement/concurrency تمام نقش‌ها صرفاً از UI یا source guard موفق فرض نمی‌شود. Version 2.0.123 اثبات یکسانی bytes نصب و ZIP نیست.

## 2. Global Role Map

این جدول حداقل ownership مشترک است، نه تکرار inventory یا grant جدید. WRITE DOMAIN یعنی domain مجاز تحت action/row/field guards؛ unconditional permission نیست.

| Role | MISSION / PRIMARY DOMAIN | WRITE DOMAIN | READ DOMAIN | MUST NOT OWN |
|---|---|---|---|---|
| Seller | اجرای کار فروش روی پرونده مجاز / Sales Operations | contact/outcome/follow-up، issuance/evidence و own conversion طبق guard؛ extra-number request | own Case/source/invoice/stage/result/history/earnings | Finance review/post/refund؛ HR final؛ MIS source repair؛ سازمان کامل |
| Supervisor | جریان کار تیم مستقیم و ready allocation / Sales Operations | owned allocation/eligible return؛ ready assignment؛ sales assist و HR request/review موجود و scoped | direct team، scoped invoices/ready/history و own account | Finance؛ HR final؛ source repair؛ blanket indirect actions |
| Senior Supervisor | هماهنگی چند تیم Supervisor / Sales Management | owned distribution؛ HR assigned review/request؛ invoice assist مشروط existing action | scoped hierarchy/performance؛ ready oversight؛ assigned personal work شرطی | direct ready assignment baseline؛ Finance/HR/MIS؛ personal receiving grant از rank |
| Sales Manager | عملیات و استثناهای چند تیم / Sales Management | owned distribution؛ Manager-owned extra-number review؛ HR step؛ existing scoped assist | territory/invoices، reason-specific archives و customer event context | Finance decision؛ HR final؛ repair MIS؛ generic archive revival/delete |
| Sales Deputy | رهبری قلمرو و پاسخ‌گویی Managerها / Sales Management | allocation موجود مشروط؛ HR request/current-reviewer step | governed territory aggregates/drilldowns و context مجاز | Manager extra-number review؛ all descendant writes؛ Finance/HR/MIS |
| MIS | کیفیت/lineage منبع و inventory / Data Operations | governed ingest/source assignment/delivery/return؛ restricted maintenance طبق policy | batch/row/source/custody/financial linkage و governed reports | financial truth decision؛ ledger؛ HR apply؛ Sales customer operation |
| HR | workforce/structure/access lifecycle / Workforce | authorized profile/hierarchy/access/compensation؛ final personnel request application؛ restricted credential policy | workforce/request/access/temporal history و needed dependency context | Finance ledger/approval؛ source repair؛ historical credit rewrite؛ general Admin |
| Finance | review/reconcile/ledger/ruled posting / Financial Operations | authorized financial-stage decision، approved financial correction/posting؛ refund conditional | financial invoice/stage/evidence/ledger/run/history/reports scope | Sales lifecycle/contact؛ HR compensation edit؛ raw source repair؛ general Admin |

Overlap مجاز: چند reader یا reviewer در یک object با authority متفاوت. Overlap نامجاز: دو final writers بدون action owner یا rank-based inheritance. HR input به commission، Finance financial decision و engine calculation سه مسئولیت‌اند؛ engine نقش نهم انسانی نیست. Admin مرز technical موجود است، نه role تازه طراحی‌شده.

## 3. Domain Architecture

| Domain | Product ownership | Sensitive write owner / boundary | Multiple readers |
|---|---|---|---|
| CRM Core | shared identity/access/guard/result contracts؛ accountable Product Owner | domain action service با policy صاحب domain؛ shared Core مجوز عمومی نمی‌دهد | هر نقش طبق scope |
| Customer / Case | Sales work و proof-based identity؛ MIS provenance | permitted Sales action؛ linkage correction restricted existing authority؛ disputed identity مشترک Product/MIS/Sales | Sales، MIS lineage، Finance linked context |
| Sales Operations | Seller execution / Supervisor team flow | action-specific Seller/assigned actor و distributor eligible | scoped management |
| Sales Management | Senior coordination، Manager operational queues، Deputy leadership | named allocation/review owner؛ chain step بدون final HR/Finance | tiers طبق scope، نه rank alone |
| Data Operations / MIS | source quality/ingest/lineage | authorized MIS source action؛ protected dependencies block destructive operation | Sales inventory، Finance proof، reports |
| Workforce / HR | personnel/structure/access/temporal input | authorized HR action؛ technical credential/access policy مستقل | scoped Sales hierarchy، MIS recipient eligibility، Finance relevant compensation snapshot |
| Financial Operations | Finance decisions/ledger/reconciliation | Finance sensitive action؛ system engine فقط approved policy؛ global flag technical rollout boundary | Sales result، MIS discrepancy، HR dependency input |
| Reporting / Analytics | مشترک metric contract؛ MIS source/report diagnosis؛ هر domain مالک semantics خودش | report job/technical state مجاز؛ underlying business correction فقط domain owner | role-scoped datasets/fields |
| Audit / History | common trace contract؛ domain صاحبان event | event actor/service؛ correction append-linked؛ overwrite تاریخی ممنوع | authorized audit readers |
| Configuration / Maintenance | domain owner + restricted technical authority موجود | action-specific authorized actor؛ global/system flags Admin boundary هدف؛ current guard discrepancy explicit | diagnostics محدود |

هر sensitive write یک decision owner و execution authority مشخص دارد؛ «shared module» به معنی shared write نیست. Domain ownership schema/table ownership تجویز نمی‌کند. F06/F08 current permission broad با target separation تفاوت دارد؛ integration این تفاوت را پنهان نمی‌کند.

## 4. Identity Model

| Term | Global meaning / canonical vs native |
|---|---|
| Customer | شخص/مشتری با identity قابل اثبات؛ ممکن است چند Phone/Case/Invoice داشته باشد؛ phone grouping فعلی customer identity قطعی نیست |
| Phone | contact/discovery attribute؛ duplicate hint؛ نه canonical Case ID، financial join یا merge permission |
| Case | واحد کار تجاری؛ logical canonical identity پایدار فقط از lineage معتبر؛ native row/lead/Dot refs حفظ |
| SourceRef | source domain + native source ID، origin batch/actor/time با confidence؛ provenance ثابت، routing امروز مستقل |
| CaseRef | source_kind/source_id + canonical_id وقتی resolved؛ unresolved صریح؛ ID برابر در دو domain یکی نیست |
| Invoice | native invoice ID مستقل؛ code reference و access token جای identity نیستند؛ چند invoice یک Case ممکن |
| Payment Stage | native stage ref + invoice + stage/purpose context؛ چند attempt یک stage ممکن |
| Financial Transaction | namespace-qualified payment/wallet/ledger/refund event identity؛ domain/key معلوم؛ یک generic ID همه ledgers نیست |
| Workforce Profile | native personnel profile ID؛ employee subject و business position؛ WP account مستقل |
| WP User | authentication/account identity؛ role/capability/direct override مستقل از position و credential |
| Hierarchy Assignment | subject-parent relation در time interval با event proof؛ نه historical credit ownership |
| Request | domain-qualified request native ID/ref؛ financial legacy review بدون ID مستقل با tuple/evidence ref موجود، نه ID ساختگی |
| Event | immutable occurrence reference با actor/object/time؛ business outcome از event type/context، نه label عمومی |

**Canonical identity:** logical stable identity؛ format/storage/migration اکنون تعیین نشده. **Native identity:** IDs فعلی همراه domain. **Alias:** رابطه اثبات‌شده native refs با نوع continuation/related؛ equality phone proof نیست. **Unknown linkage:** native refs محفوظ، resolution UNKNOWN و coverage incomplete. **Conflict:** evidence متناقض؛ automatic merge/relink ممنوع و operation وابسته fail closed. OPD-01 تصمیم موارد فاقد proof را نگه می‌دارد؛ Customer یا HR subject مجبور به Case ساختگی نمی‌شود.

MIS batch/file/run/row نیز مستقل‌اند؛ عنوان محلی «پرونده» برای batch همان Case تجاری نیست. Financial references با source-proof resolve می‌شوند، نه جایگزینی invoice.lead_id یا تبدیل اجباری Legacy به V4.

## 5. Ownership Model

در ماتریس: **V**=view فقط در scope/field مجاز؛ **Δ**=تغییر current relationship توسط action موجود و eligible؛ **E**=ثبت event/decision خود؛ **P**=subject participation؛ **—**=write target این مفهوم در baseline نیست. هیچ خانه grant actual WP permission نیست.

| Concept | Seller | Sup | Senior | Manager | Deputy | MIS | HR | Finance |
|---|---|---|---|---|---|---|---|---|
| Current Custody | V/receive | V/Δ | V/Δ | V/Δ | V/Δ conditional | V/Δ governed | V impact، نه automatic Case Δ | V linked، نه custody Δ |
| Original Owner / Original Seller | V | V | V | V | V | V/provenance | V needed | V entitlement |
| Next Actor | V/E own workflow | V/Δ authorized workflow | V/E assigned step | V/E named queue | V/E assigned step | V/E source workflow | V/E HR workflow | V/E financial workflow |
| Event Actor | E own action | E own action | E own action | E own action | E own action | E own action | E own action | E own action |
| Credit Owner | V own | V own/scoped | V own/scoped | V own/scoped | V own/scoped | — | V effective input | V/authorized entitlement correction conditional |
| Source Owner | V provenance | V provenance | V provenance | V provenance | V provenance | E origin / Δ current source stewardship | — | V financial proof |
| Request Owner / Reviewer | P extra-number | E assigned HR step | E assigned HR step | E extra-number/HR step | E assigned HR step | E own source exception | E HR final/assigned | E financial review |
| Financial Owner / Reviewer | submit/correct | scoped assist | scoped assist conditional | scoped assist conditional | V oversight | V diagnosis | V dependency input | E authorized decision |
| Workforce Subject | P own identity | V permitted subject | V permitted subject | V permitted subject | V permitted subject | V eligible recipient | V/Δ authorized personnel | V minimum input |

Original Owner اولیه منبع با Original Seller نخستین فروش یک field معنایی نیست. Source Owner تاریخی origin با current source stewardship فرق دارد. Event Actor بعد ثبت توسط هیچ نقش از hierarchy امروز redefine نمی‌شود. Original/credit history نیز هیچ نقش ordinary rewrite نمی‌کند؛ Finance تنها correction/reversal مجاز و linked under OPD-09، نه انتقال خودکار entitlement به recipient جدید. Workforce Subject شخص موضوع تصمیم است، نه reviewer. Requester با request reviewer/final applier مستقل است.

Financial reviewer لازم نیست صاحب Case باشد؛ rejected stage next actor را به مسئول correction می‌دهد و custody لزوماً عوض نمی‌شود. HR hierarchy edit access/scope را ممکن است تغییر دهد؛ Case custody و credit به‌صورت خودکار منتقل نمی‌شوند. Historical missing ownership = UNKNOWN/NOT RECOVERABLE، نه current owner.

## 6. Handoffs

Shared handoff context: domain/type، object/native refs، from/to actor-role snapshots، trigger، reason، correlation، current/next actor، event/result و failure owner. accepted_at فقط با evidence پذیرش؛ due_at فقط SLA مصوب. Current committed distribution را recipient human ACK معرفی نکنید (OPD-02).

| From → To / object | Trigger / what changes | What does NOT change | Next actor / history | Failure / unknown |
|---|---|---|---|---|
| MIS → Sales inventory / SourceRef-CaseRef | eligible source assignment/pool/delivery؛ source assignee و custody بسته به operation جدا تغییر می‌کنند | origin/native IDs/financial links؛ source assignment لزوماً delivered custody نیست | eligible sales holder؛ source row/batch + transfer refs/actor/time/result | protected/unknown block؛ MIS source owner + affected Sales، no fabricated delivery |
| Deputy → Manager / owned Case inventory | authorized committed allocation | original/event/credit history | recipient Manager؛ custody before/after و source/reason | partial item truth؛ actual sender مسؤول resolution |
| Manager → Senior / owned inventory | eligible scoped allocation | financial identity و entitlement | recipient Senior؛ transfer event | conflict/stale→recheck؛ no automatic bypass |
| Senior → Supervisor / owned inventory | eligible allocation | origin/credit/ref history | recipient Sup؛ recorded from/to/relationship | failed/unknown item reconcile before retry |
| Supervisor → Seller / eligible Case | direct eligible delivery | history/financial links و source origin | Seller own work؛ actor/recipient/time/result | eligibility changed→conflict؛ no financial reset |
| Seller → Supervisor / ready conversion Case | valid ready event؛ coordination/assignment responsibility | Seller/source/credit history؛ ready≠completed sale | direct Sup assignment؛ eligible assigned actor بعد action | missing selection/assignment proof→Incomplete؛ higher rank autoassign ممنوع |
| Sales → Finance / invoice-stage evidence | submission/resubmission→financial review context | prior evidence/approvals/invoice IDs | authorized Finance reviewer؛ evidence ref/time/amount/unit | missing proof/link/unit→Incomplete/Conflict، correction owner Sales |
| Finance → Sales / decision-result | reject→correction؛ partial→next permitted action؛ full→valid completion fact | custody و prior approvals خودکار reset نمی‌شوند | responsible Seller/authorized assistant؛ stage/reason/reviewer/history | unknown reviewer/result محفوظ؛ no assumed approval |
| Sales → HR / transfer/termination request | scoped subject/reason→assigned reviewer chain | employment/hierarchy تا actual apply؛ credit history | actual current reviewer→next reviewer/HR؛ request/chain refs | route gap/current reviewer unknown→owner diagnosis؛ no rank shortcut |
| HR → final workforce application / subject | authorized final apply→actual personnel/access effects | old event/credit/invoice history؛ profile update≠business handover | HR verifies effect؛ Sales/Finance handover actors طبق policy | failed/partial/unknown effects reconcile؛ OPD-05، no guessed successor |
| MIS ↔ Finance / discrepancy | source/query/link proof ↔ financial decision/result | domain authority و IDs؛ diagnosis≠write | named MIS source investigator و Finance reviewer؛ issue/evidence/correlation | unresolved relation preserved؛ no Fix All/relink |
| HR → Sales / structure-access effect | actual effective/applied personnel change impacts visibility/eligibility | historical teams/credit؛ current Case custody مستقل | HR actual-effect owner؛ Sales workflow/handover responsible actor | inactive/open work remains accountable؛ unknown effect≠completed offboarding |

Primary distribution chain مسیر معنایی مشترک است، نه اجبار همه داده‌ها به عبور فیزیکی از تمام tiers. MIS quick delivery، Manager/Senior/Deputy direct/skip-level technical paths حفظ و **CONDITIONAL تحت OPD-10 / SD-01**؛ policy استثنا تصویب نشده و silent removal یا expansion مجاز نیست. Escalation coordination وظیفه است؛ command جدید «ارسال به مدیر» صرف جدول handoff ایجاد نمی‌شود. Queue بدون accountable failure owner کافی نیست؛ gap موجود OPD-02/SG-02 در register حفظ است.

## 7. Customer / Case Lifecycle

Accepted global business path، بدون تغییر جهت در Specها:

| Branch | مسیر ثابت | Cross-role boundary |
|---|---|---|
| ناراضی | خدمات پس از فروش → Complaint → outcomes شامل refund / barter / others | Sales context/handoff؛ Finance فقط financial refund outcome مجاز؛ Complaint closure یا barter از invoice cancel استنتاج نشود |
| بدون خدمات | Archive | archive business reason؛ deletion، resource release یا void financial history نیست |
| خدمات مازاد | API / Web Service → Biawin CRM | transfer/integration context و provenance حفظ؛ Biawin در این مرحله طراحی نمی‌شود |

Role contracts مسیر متناقض مصوب ندارند: Manager Archive Explorer reason-specific operational view است، جایگزین این branch taxonomy یا مقصد همه complaints نیست. Finance refund یک outcome مالی در Complaint-related path می‌تواند باشد، مالک تمام services-after-sale نیست. Ready conversion/Dot/invoice workflow فرآیند فروش مربوط است، جایگزین مقصد Biawin برای «خدمات مازاد» فرض نشود.

**Contract acceptance از دستور کاربر؛ INFERENCE integration / HIGH. Runtime execution و owner دقیق Complaint/barter/API delivery/ACK از منابع role Specs اثبات نشده — NOT VERIFIED.** مقصد سرویس مسئول باید پیش از اجرای integration واقعی از policy موجود resolve شود؛ actor جدید یا role نهم طراحی نمی‌شود. Unknown endpoint/outcome به success ساختگی یا redirect داخلی تبدیل نشود؛ OPD-01 provenance و OPD-02 handoff proof این محدودیت را پوشش می‌دهند.

## 8. Financial Dependency

یک قرارداد جهانی برای Seller/Sup/Senior/Manager/Deputy/MIS/Finance و impact HR: **Consumed یک dependency guard است، نه sold status.** Ordinary return/reassignment/reset/cleanup وابسته به همان Case باید تمام رابطه‌های مالی معتبر و nonfinancial obligations را resolve کند. یک role predicate آسان‌تر یا last invoice lookup جای قرارداد مشترک نیست.

Proof hierarchy از G: real legacy invoice.lead_id؛ V4 follow_invoice_id؛ flow state/event invoice refs با source context و active invoice resolution؛ Dot case/payment refs؛ compatibility IDs فقط validated fallback. Phone/name/time similarity discovery only. Union تمام valid links لازم؛ عدم match یک source نفی links دیگر نیست.

| Dependency result | ordinary operation eligibility |
|---|---|
| هیچ financial/materialized/fulfillment dependency با resolution کامل | فقط با actor/row/scope/handoff/reset policy eligible |
| draft/pre_invoice/pending/receipt/review/rejected-correctable invoice معتبر | blocked؛ هنوز وصول نشده نیز financial commitment دارد |
| partial/full paid/completed یا protected downstream materialization | blocked؛ پایان فروش history را آزاد نمی‌کند |
| cancelled unpaid / paid / refund-reported | release خودکار ممنوع؛ OPD-03/04 و reconcile لازم |
| missing target/conflicting/unresolved linkage | fail closed؛ named reconciliation owner، نه unused/orphan label قطعی |

Preview eligibility با evaluated_at/context proof؛ apply fresh recheck actor/action/row/field/state/dependencies در commit. Invoice creation و ordinary return نمی‌توانند هر دو از فرض unused ناسازگار commit شوند؛ invoice orphan یا financial ownership loss ممنوع. روش atomicity انتخاب Engineering آینده است. Financial cancellation/reversal action تخصصی از ordinary inventory return مستقل است.

**F01 P0 / BOTH برای code-live inconsistency، destructive runtime execution NOT VERIFIED؛ اصلاح نشده.** SG-08/M-G01/SD-G01/MIS-G01/FIN-LINK همان obligation هستند. ID/links/history حفظ؛ policy reset contact fields و cancelled release OPD-03، نه predicate جدا برای هر نقش.

## 9. Invoice / Payment Model

Readers در جدول همیشه scoped/field-limited؛ source of truth domain facts است، نه صرفاً یک mirror label یا KPI.

| Term | Definition | Source of truth | Readers | Decision owner |
|---|---|---|---|---|
| Invoice Created | issuance fact برای invoice stable ID؛ pre_invoice شامل | invoice/issuance event refs | Sales/MIS/Finance reports | authorized issuer/service |
| Pre-invoice | invoice قبل وصول با namespace فعلی؛ object شخص جدا نیست | current invoice lifecycle + issuance history | scoped Sales/Finance/MIS | issuance/payment domain |
| Payment Stage | موضوع مرحله/purpose/due با invoice relation | stage facts/refs | Sales/Finance، MIS needed proof | stage service؛ authorized decision |
| Receipt Submitted | evidence دریافت شده، نه وصول معتبر | evidence/version/uploader/event | responsible Sales/Finance | authorized submitter؛ validity Finance |
| Finance Review | تصمیم روی همان stage/evidence | review event/reviewer/reason | Sales result/Finance؛ MIS diagnostic context | authorized Finance reviewer |
| Stage Approved | مرحله مشخص تأیید مالی شده | stage + review/payment valid facts | Sales/Finance/MIS outcomes | Finance/finalizer under policy |
| Paid Amount | وصول معتبر با unit/basis مشخص | validated stage/payment facts، no double count | reports scope | financial validation/service |
| Remaining | obligation باقی با same-unit basis | invoice/payment conservation | scoped Sales/Finance/MIS | financial semantics |
| Invoice Fully Paid | همه مبلغ با valid payments و tolerance معتبر پوشش دارد | paid/remaining + valid lifecycle/completion evidence | scoped roles | finalizer/Finance policy |
| Sales Completed | completion fact طبق G؛ issuance/receipt/one stage کافی نیست | completed invoice predicate/event | governed reports | finalizer؛ Sales consumes outcome |
| Refund | request/report/proof/financial execution جدا، اصل invoice محفوظ | linked refund/proof/reconcile evidence | Finance، relevant Sales/MIS | Finance authorized policy؛ OPD-04 |
| Ledger Posted | financial effect با tx/business key اثبات‌شده | posted transaction references | Finance؛ own/scoped earnings/context | approved engine/posting authority |

Conservation: same-unit nonnegative paid + remaining ≈ total با approved tolerance. OPD-06؛ 0.5 بومی code baseline سراسری currency policy نیست؛ IRT/IRR conversion ضمنی ممنوع. Refund gross/refunded/net با collection basis جدا؛ refund از total rewrite حدسی تولید نشود. Earned entitlement/calculated preview/run approval/wallet credit/settlement/reversal synonyms نیستند. Stage auto-credit purpose engine با legacy APPLY هم‌زیست current؛ OPD-09 یک authority per entitlement را هدف می‌گیرد، نه ادعای single current engine.

## 10. Status Namespaces

| Namespace | Qualifier لازم / معنی مستقل |
|---|---|
| Case State | native Case domain/ref؛ Dot completed یا legacy label همان invoice state نیست |
| Sales Workflow State | contact/follow-up/ready/next-action/assignment؛ ready≠paid |
| Invoice Lifecycle | pre_invoice/partial_paid/paid/approved/cancelled/... طبق S و amount predicate |
| Payment Stage | pending/approved/paid/rejected/... فقط همان stage |
| Finance Review | pending/approved/rejected همان review subject/evidence |
| Refund | required/requested/reported/confirmed/completed concepts؛ cancelled جدا |
| HR Request | pending_review/pending_hr/approved/rejected/failed؛ intermediate step vs applied result |
| Employment | active/inactive/terminated context؛ employee status≠account/session revocation |
| Import / Source Quality | valid/duplicate/invalid؛ assignment و consumed independent |
| Operation Result | applied/skipped/rejected/failed/unknown per intent/item؛ business state enum نیست |
| System / Transport State | Loading/Offline/Stale/...؛ network failure financial outcome را تعیین نمی‌کند |

Term resolution: **approved** باید subject stage/review/run/request داشته باشد؛ run approved credit نیست، HR step approved applied نیست. **completed** باید invoice sale/task/import operation scope داشته باشد؛ task بسته پایان مالی نیست. **assigned** source-manager، Case custody، converter یا HR reviewer assignment را qualify کند. **returned** financial correction، inventory return و customer lifecycle متفاوت. **failed** business request label current با known operation failure یک proof نیست. **pending** reviewer/payment/HR/item subject را نام ببرد. Dictionary/read adapters current IDs را حفظ می‌کنند؛ enum واحد یا DB rename وجود ندارد.

## 11. Request Model

Shared conceptual envelope: domain + Request ID/native reference، requester، subject، owner/current reviewer، state، reason، history، result، next actor؛ timestamps/correlation/proof در صورت applicable. این object physical table یا full workflow مشترک نیست.

| Domain request / item | Shared shape | Must stay domain-specific |
|---|---|---|
| HR transfer/termination | requester/employee/from-to/current reviewer/result | reviewer path؛ authorized HR final apply؛ handover/effective date policy؛ current HR می‌تواند pending_review را نیز finalize کند، universal full-chain rule اختراع نشود |
| Extra-number | Seller requester/assigned Manager/state/resulting legacy lead | manager_id row owner؛ normalized phone duplicate check scope؛ approval legacy source provenance؛ Deputy substitute نیست |
| Financial evidence/review | invoice-stage/evidence/uploader/reviewer/reason/result | stage amount/proof/approval guard؛ no fabricated separate request ID در legacy؛ completion/refund/ledger independent |
| Exception/reconciliation | issue subject/domain owner/evidence/resolution/next actor | diagnosis≠write؛ source/ledger/HR/Sales guards و domain resolution proof |

Requester≠reviewer≠final applier؛ changed hierarchy request manager snapshot را silently rewrite نمی‌کند. Missing Manager/current reviewer/failure owner یک gap است؛ request approved label به‌تنهایی completion proof نیست. Shared request context grants no action و approval chain واحد تحمیل نمی‌کند.

## 12. Permission Model

چهار لایه binding: **Route → Module → Action → Row / Field**؛ authority و actual data scope در هر لایه. **View ≠ Write؛ Hierarchy ≠ Permission؛ Position ≠ WP Role ≠ Capability ≠ Credential.** Account login دسترسی module/row را ثابت نمی‌کند؛ nonce فقط کافی برای authority نیست. Conceptual permissions به WP cap جدید تبدیل نشده‌اند.

Legend: **A**=allowed only with proven action/state/row/field authority؛ **C**=conditional existing path/policy/assignment؛ **—**=must not own در baseline؛ **R**=read/diagnostic context فقط. جدول target boundary است، نه claim current guard perfection.

| Sensitive action | Seller | Sup | Senior | Manager | Deputy | MIS | HR | Finance |
|---|---|---|---|---|---|---|---|---|
| Sales contact / outcome | A own | C assigned/assist | C assigned only | C explicit action only | R | — | — | R needed |
| Allocation / ordinary return | receive | A owned | A/C owned | A/C owned | C owned | A governed | R impact | R dependency |
| Ready conversion assignment | — receiving ≠ assign | A scoped | R baseline | C only proven existing | R baseline | — | — | — |
| Invoice issue / evidence / allowed sales correction | A | C | C | C | R baseline | R proof | — | C exact financial metadata, not Sales owner |
| Extra-number approve/reject | request | — | — | A assigned Manager | — | — | access input, not review owner | — |
| HR request create / step review | — baseline | A subject/current reviewer | A subject/current reviewer | A subject/current reviewer | C route/assigned step | — | A authorized HR workflow | — |
| HR final apply / structure / compensation | — | — | — | — | — | — | A specific authority | R financial input |
| Financial stage approve/reject | — | — | — | — | — | R diagnosis | — | A distinct actions |
| Refund execution/confirmation | request/context | R | R | R | R | R diagnosis | — | C OPD-04/proof |
| Rules/run approval/posting/ledger correction | own outcome | own outcome | own outcome | own outcome | own outcome | — | effective input | A/C separate action; F06 current gap |
| Import/source destructive maintenance | — | — | — | — | — | C impact + protected guard | HR import only separate | — |
| Credentials / impersonation | — | — | — | — | — | — | C restricted existing policy | — |
| Sensitive export | C | C | C | C | C | C source/report | C workforce | C finance |

Current sensitive broad guards: F06 finance-view writes، F08 workforce/access/compensation/credential boundaries؛ no silent permission reset. Matrix/manual-adjust/global flag boundaries از FIN preserved؛ Finance به Admin عمومی تبدیل نمی‌شود. Ready oversight Senior و Manager-only extra-number review با Deputy rank توسعه نمی‌یابند. Change position ممکن است access side effects واقعی داشته باشد؛ simple edit label آن را conceal نکند. Negative runtime enforcement همه sensitive paths **NOT VERIFIED**؛ HR/Finance **NOT LIVE VERIFIED**.

## 13. KPI / Reporting Contract

Trusted metric envelope: **name، grain، source، cohort، time_basis، scope، formula/predicate، unit در صورت amount، freshness، coverage/version**؛ numerator/denominator ratios same-grain compatible، count denominator N/A. Missing metadata = NOT FINAL، denied/incomplete/error = zero واقعی نیست. Cards/tables/drilldowns/exports با same contract reconciliation لازم؛ shared semantic facts layer فیزیک DB را تعیین نمی‌کند.

| Metric | Global definition / time basis | Duplication resolved |
|---|---|---|
| Invoice Created | distinct invoice IDs issued/created in range؛ pre_invoice داخل | legacy pending alias list این metric را محدود نمی‌کند |
| Open Pre-invoices | current invoice snapshot matching open preinvoice predicate؛ cancelled/completed جدا | same name را issuance range ننامید |
| Pre-invoices Issued | issuance event in range؛ later paid شدن issuance history را حذف نمی‌کند | با Open Pre-invoices جمع/جایگزین نشود |
| Sales Completed | distinct invoice completions with validated full-paid predicate/event window | receipt/one approved stage/issuance count کافی نیست |
| Total Cases | distinct resolved logical Cases across declared source coverage؛ aliases proven dedup؛ unresolved count جدا | legacy query lead0 معنی CRM0 ندارد؛ same invoice across Cases grand total once |
| Legacy Leads Only | distinct legacy sn_leads IDs within declared scope/time | honest label؛ V4/MIS/Dot coverage ادعا نشود |
| Verified Collected Amount | sum validated payment/stage events با یک authoritative path و explicit unit/basis؛ duplicate attempts حذف معنایی | invoice total، wallet commission و bank settlement نیست؛ reversal/refund basis جدا |

**Current Snapshot**=current state evaluated now؛ **Event Range**=events/cohort selected by event time، outcome observation basis جدا؛ **Historical As-Of**=true state at cutoff با history proof. MIS event cohort + current gathered status یک valid composite mode است، as-of نیست. Stepped gather window atomic snapshot فرض نمی‌شود. Current hierarchy workload و event/earned-credit attribution تاریخى جدا؛ ancestor aggregates additive نیستند. Conversion rate/cohort/window/ranking/targets OPD-07؛ amount boundary OPD-06.

F02 pre_invoice predicate omission، F03 legacy-only coverage **BOTH/CODE طبق A/Spec**؛ F04 0-vs-1 observed discrepancy، root cause NOT VERIFIED؛ F09 event/cohort + gather-current semantics **BOTH**. F05 تازه Manager list/detail loaded و residual loader؛ mismatch event populations قطعی نشده. هیچ مشکل KPI «با redesign حل شده» نیست. Financial ledger/report parity نیاز reconcile و tx proof؛ UI totals ledger authority نیست.

## 14. System Result Model

| Shared non-business state | Allowed meaning |
|---|---|
| Loading | read/operation هنوز final نشده؛ truth/count موقت |
| Empty | authorized complete population genuinely empty |
| No Result | valid filtered scope no match؛ کل data empty نیست |
| Unauthorized | route/module/action/row/field denied؛ sensitive leakage ممنوع |
| Stale | زمان/version truth با current source/state متفاوت؛ write intent fresh recheck |
| Incomplete | missing source/evidence/link/coverage؛ bounded rows یا unavailable domain صریح |
| Conflict | proof/state/version/scope یا concurrent business intent ناسازگار؛ no silent reset/merge |
| Partial Success | some items/effects applied و بعضی unresolved/rejected/failed؛ per-item breakdown |
| Retryable Failure | failure معلوم + safe replay expectation + fresh guard؛ classification به‌تنهایی اجازه write نمی‌دهد |
| Outcome Unknown | commit/effects ممکن ولی proof ناقص؛ error/timeout بعد possible commit نمونه |
| Offline | source/transport unavailable؛ cached truth stale؛ queued write applied نیست |

**Failed** فقط وقتی outcome مورد نظر با evidence شکست معلوم است؛ برای multi-effect operation باید applied/failed/unknown هر effect جدا باشد. Failure پیام/HTTP یا status=failed در HR/run proof zero effect نیست. **Outcome Unknown** وقتی commit status/effects قابل تعیین نیست، حتی transport failure قطعی باشد. Blind retry پس از Unknown برای تمام writes ممنوع تا audit/current-object/transaction reconciliation. Read retry مالی نوشتن پنهان نداشته باشد؛ F07 این purity را در current code تضمین نمی‌کند. Unknown را success یا failed-zero نمایش ندهید.

## 15. Bulk Contract

Shared accounting: **Requested** selected intent population؛ **Eligible** preview candidates؛ **Applied** verified committed effects؛ **Skipped** intentionally not acted with reason؛ **Rejected** policy/state denied؛ **Failed** known execution failure؛ **Unknown** unresolved effect. Eligible subset و preview fact است، terminal outcome bucket نیست.

در عملیات final و fully accounted: Requested = Applied + Skipped + Rejected + Failed + Unknown با mutually exclusive per-item final buckets. در in-progress/candidate-bounded operation، pending/unprocessed coverage صریح؛ آن‌ها را failed یا success جا نزنید. Multi-effect item breakdown نیز محفوظ؛ tx already existing idempotent hit با new credit متمایز و reason قابل ردیابی.

| Contract aspect | Global rule |
|---|---|
| Preview | exact IDs/filters/fields/scope/source/dependencies/version/impact؛ preview metadata write possible، financial effect نیست |
| Apply recheck | authority + eligible row/state/field/dependency near commit؛ preview grant نیست |
| Partial result | per-item refs/before-after/result/reason؛ processed/unprocessed counts، no fake all-success |
| Retry | known safe intent replay با stable correlation/business identity؛ Unknown first reconcile |
| Audit | actor/action/requested/eligible/actual result/time/source/run/refs؛ summary overwrite جای history نیست |
| Idempotency | no duplicate delivery/HR apply/entitlement effect for same business intent؛ mechanism Engineering future |

Support domain-specific است: MIS ingest/distribution supported but protected return/cleanup conditional؛ Sales distribution subject guards؛ Manager extra-number/HR bulk review policy unverified؛ HR sensitive bulk conditional؛ Finance bulk approve current ≤100 و posting run bounded، bulk refund NOT ALLOWED frozen scope. Shared contract هیچ domain را bulk-enabled یا authority جدید نمی‌کند.

## 16. Temporal Model

| Time concept | Meaning / cross-domain rule |
|---|---|
| Current hierarchy | امروز چه کسی parent/eligible/scope است؛ historical sales attribution نیست |
| Historical attribution | event/entitlement snapshot یا proven interval؛ current parent جای missing history نیست |
| Effective time | business relation/rule/compensation از چه زمان اثر دارد؛ scheduled engine promise از field وجود استنتاج نشود |
| Applied time | actual change اجرا/ثبت شد؛ requested/effective date مستقل |
| Event time | occurrence issuance/contact/transfer/payment/request؛ report cohort basis |
| Posting time | ledger commit occurrence؛ financial effective time/source event جدا |
| Report gather time | observation/materialization time/window؛ historical cutoff نیست |

Sales: workload current scope؛ credit/event attribution تاریخی؛ transferred Case یک event جدید، قبلی حذف نمی‌شود. HR: interval close/open، intended effective date و apply-now code paths؛ atomic overlap/race/partial behavior NOT VERIFIED، scheduled workforce apply وعده داده نشود. Finance: entitlement snapshot eligibility/rate/time، payment/refund event و posting time مستقل؛ HR امروز نرخ credit گذشته را rewrite نمی‌کند. MIS/report: origin time، import run، delivery time، cohort event و gather window جدا؛ as-of با proven history و timezone. Jalali/Gregorian presentation یک timestamp interpretation مشترک دارد؛ تبدیل تاریخ view basis business را عوض نمی‌کند. Unknown historical intervals UNKNOWN/NOT RECOVERABLE باقی می‌مانند.

## 17. Audit / History

Common logical audit envelope where applicable: **actor، subject/object native+domain refs، action، before/after، reason، event/effective/applied time، result، correlation/reference**. Role at event با role فعلی فرق دارد. Human/system actor و initiator/reviewer/applier/poster جدا؛ secrets یا unrelated PII وارد audit payload نشود. History append/correction-linked، physical audit table مشترک الزام نیست.

| Extension | Domain-specific proof |
|---|---|
| Sales transfer | Case/SourceRef، from/to custody/relationship، recipient eligibility، original/event actor، dependency/reason/outcome |
| Finance | invoice/stage/evidence version، amount/unit، review، tx ID، business key، run/item، reversal/refund/correction references، retry/unknown result |
| HR | profile/user/subject/request، reviewer chain، intended vs actual delta/access/compensation period، before-after/effective/applied/result |
| MIS | file/batch/source row/run/parser/quality، materialization/links، delivered/return/repair impact، per-row outcomes/coverage |
| Reporting | query/metric contract version، source/cohort/time/scope/fields/gather window/coverage؛ technical job metadata≠business history |

Historical missing proof به actor/time امروز نسبت داده نشود. Required audit coverage موفق از log helper presence اثبات نمی‌شود؛ actual persistence/partial failures validation آینده است. Review decisions، bank proof و ledger effect هرکدام evidence خود را دارند؛ customer behavior event مالی نیست.

## 18. Maintenance Boundary

| Level | Meaning / examples | Authority boundary |
|---|---|---|
| Normal Operation | own Sales work، scoped allocation، assigned request review، financial stage review، source ingestion | named domain action + guards؛ ingestion may have current maintenance side effects، pure read claim نکنید |
| Advanced Operation | diagnostics، reconciliation investigation، exceptional run/failed apply context | additional scope/context؛ diagnosis correction permission نیست |
| Maintenance | MIS cleanup، Finance recalculate/backfill، HR linkage recovery، legacy migration | independent impact/authority/dependency/audit؛ protected or unknown financial relation fail closed |
| Restricted Technical/Admin | schema rebuild، global activation/configuration، privileged credentials/access integration | existing authorized technical policy؛ ordinary business-role rank جای آن نیست |

HR credentials/impersonation current code HR gate دارد؛ این integration آن را falsely current Admin-only نمی‌نامد. Target restricted granular policy HR-G04/05 deferred؛ impersonation inherently read-only نیست. Finance manual adjustment/matrix/backfill/global flag current guards FIN§3/15 محفوظ، target separation F06؛ bank settlement execution ساخته نمی‌شود. MIS current import cleanup/schema calls technical آثار دارد؛ pure diagnosis و business/destructive writes target separated، DB metadata jobs مجاز با disclosure طبق G INV-026/027. Normal read بدون business mutation الزام است؛ technical cache/job write معادل permission-free business change نیست.

## 19. Open Decision Register

Canonical register پایین existing IDs را حفظ و duplicate references را alias می‌کند؛ ID جدید برای مسئله‌ای که existing contract پوشش می‌دهد ساخته نشده. **Design Blocking? = No در محدوده frozen baseline/conditional states**؛ وابسته به feature activation ممکن است policy/UI outcome final نشود. **Implementation Blocking** scoped است، نه کل CRM. Ownerها accountable decision/validation owners هدف‌اند، نه انتساب actor runtime ناشناخته. Local aliases نیز در ردیف خود track می‌شوند؛ role Specs خاموش ویرایش نشده‌اند.

| Canonical ID / aliases | Decision / gap | Roles / domains | Priority | Design Blocking? | Implementation Blocking? | Evidence Needed | Owner |
|---|---|---|---|---|---|---|---|
| OPD-01؛ SG-06،MIS-G02؛ identity part M-G04 | ambiguous lineage/aliases/Case coverage؛ continuation vs distinct | All/Core/MIS/Reports | P1 | No | dependent joins/merges and trusted total؛ no destructive path | stable source proof + coverage/conflict catalog | Product + MIS + Sales + Engineering |
| OPD-02؛ SG-02،M-G11،SD-G07،MIS-G12 | sensitive ACK/SLA/escalation owner and supported routing | cross-domain handoffs | P2 | No | new escalation/ACK activation | existing owner/path/acceptance evidence + policy | Product + Sales + MIS + HR |
| OPD-03؛ M-G09 revival/release policy part | cancelled release/reset/archive revival authority | Sales/MIS/Finance | P1 | No | release/reset/revival extension | reason-specific dependencies/history policy | Product + Sales + MIS + Finance |
| OPD-04 | refund object/proof/authority | FIN/Sales/Complaint | P1 | No | refund execution/confirmation new contract | bank/source proof & permitted outcome policy | Product + Finance |
| OPD-05؛ M-G07،HR-G02 | termination/transfer accountable handover + access timing | HR/Sales/Finance | P0 impact | No | enhanced final handover | open work/financial impact + named eligible successor policy | Product + HR + Sales + Finance |
| OPD-06؛ currency part M-G12/SD-G04 | unit/rounding/0.5 boundary | Financial/reports/all Sales | P1 | No | normalization/borderline completion | approved money contract + source-unit proof | Product + Finance + Engineering |
| OPD-07؛ SD-05،SG-05؛ cohort parts M-G12/SD-G04 | cohort/window/rate/imbalance/attribution | Sales/MIS/Reports | P1 | No | trusted rates/ranking/targets/as-of attribution | same-cohort metadata + historical coverage | Product + Sales + MIS + Finance |
| OPD-08؛ review-separation part MIS-G08 | maker/checker thresholds/emergency exceptions | Finance； MIS approval only when applicable | P1 | No | mandatory actor-separation activation | organizational policy؛ MIS extension not auto Finance rule | Product + Finance؛ MIS policy owner for its plans |
| OPD-09 | entitlement engine routing/noninvoice keys/correction | Finance/HR/Sales | P1 | No | financial engine activation/cross-engine safe posting | active routes/flags + entitlement/key proof | Product + Finance + HR + Engineering |
| OPD-10؛ SD-01،SD-02،SD-03،SG-03،SG-04،SG-07،M-G08،SD-G05 | skip-level、personal receiving、invoice assist/delegation | Sales tiers/MIS delivery + Finance boundary | P1/P2 | No | expanded/default bypass or assist grants | per-action existing guard/row/field + business approval | Product + Sales + Finance + HR؛ MIS source paths |
| SD-04 | HR cross-scope transfer destinations/noop policy | HR/Sales | P1 | No | newly normalized destination policy | valid subject/target/current parent/chain scope evidence | Product + HR + Sales |
| F01؛ SG-08、M-G01、SD-G01、MIS-G01、FIN-LINK | all-link protected return/reset/race contract | all inventory writers | P0 | No | safe ordinary return/reset rollout | union-link resolution + commit conflict/history proof | Sales + MIS + Finance + Engineering |
| F02؛ preinvoice parts SG-01/SD-G02/MIS-G04 | current preinvoice predicate omission | Sales/Reports | P1 | No | trusted Open Pre-invoices parity | same predicate/IDs across card/detail/report | Sales + MIS/Reporting |
| F03؛ legacy parts SG-01/SD-G02/MIS-G04 | legacy-only count vs Total Cases | Sales/MIS/Core | P1 | No | trusted cross-source totals | honest source coverage + resolver proof | Sales + MIS/Reporting |
| F04؛ M-G02、SD-G03؛ parity part MIS-G04 | observed invoice count discrepancy | Sup/Manager/shared Reports | P1 | No | performance trust affected metrics | identical grain/scope/time/source/version parity | Sales + MIS + Engineering |
| F05؛ M-G03 | Manager residual loader/event populations | Sales/customer history | P1 | No | claimed fully coherent history state | latest evidence list/detail loaded؛ state/population parity | Sales + Engineering |
| F06 | finance-view sensitive writes | Finance/access | P1 | No | sensitive authority separation acceptance | distinct action guards + negative runtime proof | Finance + Product + Engineering |
| F07؛ M-G06、SD-G11 | conditional renderer legacy autopost | Finance/shared own wallet | P1 | No | certified pure business read | active-engine policy + no unintended tx proof | Finance + Engineering |
| F08؛ HR-G03 | workforce/position/access sensitive side effects | HR/access | P1 | No | sensitive write authority split | per-route/action/field actual effect + negative proof | HR + Product + Engineering |
| F09؛ MIS-G05 | event selection + gathered current state vs as-of | MIS/Reporting | P2 | No | true historical-as-of claims | event/cutoff/history/gather semantics evidence | MIS + domain report owners |
| M-G04 nonidentity request edges؛ SD-G08 extra-number context | inactive/reparented requester/no Manager/duplicate review/result scope | Manager/Seller/Deputy optional read | P1/P2 | No | expanded queue/read/edge handling | assigned Manager snapshot/current eligibility/outcome proof | Sales + HR + Product |
| M-G10؛ SD-G10 | domain-specific bulk review/exception policy | Sales/request queues | P2 | No | new bulk sensitive actions | per-item authority/reason/result + support evidence | Product + Sales |
| M-G13؛ SD-G12 | source/stage/next-actor context completeness & hierarchy loading | Sales/Reports | P2 | No | trusted contextual display/coverage | actual IDs/context coverage & honest partial states | Sales + MIS + Engineering |
| SD-G06؛ M-G05 | shell/action/scoped authority alignment | Sales/access | P1 | No | sensitive action acceptance | route/module/action/row/field negative tests | Product + Sales + Engineering |
| SD-G09؛ HR-G11 | HR entry/current reviewer/finalize pending_review policy | Sales Deputy/HR/request | P2 | No | new entry/chain constraints | actual route & current reviewer； HR bypass authority policy | HR + Sales + Product |
| MIS-G03 | readiness/health contradict readable report | MIS/technical | P1 | No | blind repair prohibited؛ readiness trust | current health/schema proof； cause unknown | MIS + Engineering |
| MIS-G06؛ FIN-RUN-COVERAGE | bounded/stepped candidate coverage؛ posting tail | MIS reports / Finance run | P1/P2 | No | full-population or all-posted claim | processed/unprocessed/gather window vs full intent | MIS + Finance + Engineering |
| MIS-G07 | maintenance/repair authority | MIS/technical | P1 | No | destructive repair/cleanup activation | exact row/field/impact/financial guards | MIS + Product + Engineering |
| MIS-G08 non-review portion | plan candidate/hierarchy/apply responsibility | MIS/Sales | P2 | No | plan/new forecast activation | actual plan/eligible recipients/approval effects | MIS + Sales + Product |
| MIS-G09؛ HR-G08、FIN-RECOVERY | domain partial/unknown apply/retry proof | MIS/HR/Finance + global operations | P1 | No | safe retry/recovery | actual effects/audit reconciliation per domain | domain owner + Engineering |
| MIS-G10؛ HR-G09 export portion؛ FIN-PARITY download portion | export parity/field/download/privacy | MIS/HR/Finance/Reports | P1/P2 | No | sensitive export/download acceptance | exact fields/scope/source/expiry & role negative tests | domain data owners + Engineering |
| MIS-G11 | latest import vs latest operation/history | MIS/audit | P2 | No | trusted operation history | run type/history/result provenance | MIS + Engineering |
| HR-G01； FIN-RUNTIME | real role runtime accounts absent | HR/Finance | P1 validation | No | role-specific runtime acceptance | dedicated nonAdmin role accounts + read/write-negative proof later | HR + Finance + environment owner |
| HR-G04 | credential sensitive authority/send/storage | HR/access | P1 | No | credential policy activation | allowed targets/credential lifecycle/audit policy | HR + Product + security owner |
| HR-G05 | impersonation purpose/action/expiry/session audit | HR/access | P1 | No | controlled impersonation acceptance | durable start/end/action scope + recovery proof | HR + Product + security owner |
| HR-G06 | current/effective/applied intervals/history integrity | HR/Sales/Reports | P1 | No | historical hierarchy/as-of trust | interval/overlap/partial/race proof | HR + Engineering |
| HR-G07 | compensation authority/paid period/snapshot integrity | HR/Finance | P1 | No | compensation change + historical credit protection | effective periods/allowed edits/lock/rollback evidence | HR + Finance + Product |
| HR-G09 bulk portion | workforce bulk fields/effects | HR | P2 | No | new sensitive bulk defaults | server IDs/field/scope/per-effect proof | HR + Product |
| HR-G10 | onboarding profile/account identity and complete effect | HR/Core access | P2 | No | asserted complete activation | native link/conflict/access/hierarchy effect proof | HR + Engineering |
| HR-G12 | documents/training complete workflow unknown | HR | P3 | No | new scope only | user need & supported workflow evidence | HR + Product |
| FIN-IDEMPOTENCY | transaction/key exactly-once across retries/engines | Finance | P1 | No | duplicate-safe posting acceptance | same intent/new run/race keys & committed tx proof | Finance + Engineering |
| FIN-LOCK | approved run snapshot all-path immutable proof | Finance | P1 | No | approved-input safety | snapshot/rule/input lock coverage | Finance + Engineering |
| FIN-PARITY ledger portion | ledger/current/historical report consistency | Finance/Reporting | P2 | No | trusted financial aggregates | tx-domain basis/reversal/as-of parity | Finance + Reporting |
| FIN-TAXONOMY | overlapping receipt/review facets & distinct states | Finance/Reports | P2 | No | trusted counts, not basic layout | declared grain/partition & predicate proof | Finance + Reporting |
| FIN-IMPORT | bank statement import need/support | Finance | P3 | No | new import scope | reconciliation need/system/data contract | Finance + Product |

Trace closure: all SG-01…08、SD-01…05、M-G01…13、SD-G01…12、MIS-G01…12、HR-G01…12 و FIN-* ردیف‌های FIN§18 در register بالا به existing canonical issue mapped هستند. SD-G04 و M-G12 میان OPD-06/07 split refs؛ SD-G08 archive visibility portion زیر OPD-03/OPD-10 و extra-number زیر M-G04؛ MIS-G04 میان F02/03/04؛ FIN-PARITY میان reports و sensitive downloads. **F10** duplication presentation با reuse domain/request/report contracts sections 3/11/13/23 **RESOLVED BY EXISTING CONTRACT در target**؛ current duplication حذف/رفع نشده، issue تازه سیاستی لازم ندارد. ثبت aliases حذف سابقه IDهای قبلی نیست.

## 20. Cross-role Conflict Review

| Candidate conflict | Classification | Resolution / remaining limit |
|---|---|---|
| Sales evidence/assist vs Finance confirmation | RESOLVED BY EXISTING CONTRACT | submission/correction جدا از stage approval؛ OPD-10 allowed fields؛ F06 actual guards gap محفوظ |
| Deputy rank vs Manager extra-number final review | NO CONFLICT | assigned Manager row/action owner؛ Deputy read optional، review NOT IN ROLE |
| Senior ready oversight vs Sup assignment | NO CONFLICT | ready readonly baseline؛ personal assigned work conditional، no inherited assign grant |
| MIS source-manager assignment vs custody delivery | RESOLVED BY EXISTING CONTRACT | separate Source Owner/current custody/event؛ staged≠delivered |
| Multiple distributors vs shared source ownership | RESOLVED BY EXISTING CONTRACT | owned eligible action at each level؛ origin/credit fixed؛ skip-level NEEDS POLICY OPD-10 |
| HR final apply vs Sales chain reviewer | RESOLVED BY EXISTING CONTRACT | intermediate decision≠actual apply؛ HR pending_review branch preserved، mandatory full-chain policy not invented |
| HR termination flag vs ownerless Sales/Finance work | PRODUCT GAP | OPD-05/HR-G02؛ named handover required before claiming complete offboarding |
| Two commission engines vs Finance ruled posting | NEEDS POLICY | OPD-09؛ legacy APPLY not sole engine؛ one entitlement authority، no double post; existing engines kept |
| View role vs write authority | PRODUCT GAP | F06/F08؛ four layers target؛ current runtime enforcement not asserted fixed |
| Same approved/completed/assigned label across objects | RESOLVED BY EXISTING CONTRACT | namespace + subject + proof؛ stored enums unchanged |
| Same preinvoice/lead/invoice metrics differ | PRODUCT GAP | definitions globally unified؛ F02/F03/F04 still need parity proof |
| MIS report range vs historical as-of | RESOLVED BY EXISTING CONTRACT | event cohort/current gather labeled؛ F09 historical runtime remains unverified |
| Hierarchy change vs earned credit/owner history | RESOLVED BY EXISTING CONTRACT | current scope independent immutable historical attribution؛ HR-G06/07 validation remains |
| Unknown operation vs Failed label/retry | RESOLVED BY EXISTING CONTRACT | known effects vs uncertain commit؛ reconcile first؛ recovery runtime IMPLEMENTATION VALIDATION |
| Read report job vs business mutation | RESOLVED BY EXISTING CONTRACT | disclosed technical job/cache permitted؛ F07 business credit on read PRODUCT GAP |
| Complaint/refund/archive/Biawin vs operational queues | NO CONFLICT in target | user-approved branches preserved؛ endpoint/actor runtime NOT VERIFIED، OPD-01/02 before execution |
| Queue with no next actor / invisible HR entry | PRODUCT GAP | OPD-02/SD-G09/HR-G11؛ no guessed owner, command or grant |
| Full posting summary vs bounded items/partial errors | IMPLEMENTATION VALIDATION | FIN-RUN-COVERAGE؛ honest tail/per-item tx required؛ no all-posted claim |
| Shared Request contract vs one universal approval chain | NO CONFLICT | shared envelope only؛ domain states/authority retained |

Architecture conflict به policy/validation unresolved محدود شده و safe conditional baseline دارد. هیچ role دو final authorities متناقض را در یک decision claiming نمی‌کند؛ sensitive unknown policy route فعال مصوب اعلام نشده. Integration target با current defects یکسان فرض نمی‌شود.

## 21. Implementation Readiness

Classification dimensions مستقل‌اند: DESIGN READY یعنی WHAT/WHY کافی؛ IMPLEMENTATION CONTRACT READY یعنی invariant/boundary تعریف‌شده، نه code/test ready؛ IMPLEMENTATION BLOCKED برای operation وابسته تصمیم/validation؛ RUNTIME VALIDATION REQUIRED برای actual role/enforcement/outcome. هیچ implementation plan یا execution order این بخش ارائه نمی‌دهد.

| Domain / capability | Classification | Limit / accepted prerequisite |
|---|---|---|
| Sales scoped read context/IA | DESIGN READY؛ IMPLEMENTATION CONTRACT READY | real data scope/partial states preserved؛ KPI affected values not trusted-final |
| Shared identity / ownership / status dictionary | DESIGN READY؛ IMPLEMENTATION CONTRACT READY | OPD-01 unresolved joins conditional؛ no physical migration |
| MIS read / reconciliation diagnosis | DESIGN READY؛ IMPLEMENTATION CONTRACT READY؛ RUNTIME VALIDATION REQUIRED for scale | source quality/coverage/health honest؛ mutation not part of diagnosis |
| HR runtime / sensitive apply | DESIGN READY baseline؛ RUNTIME VALIDATION REQUIRED؛ IMPLEMENTATION BLOCKED affected writes | real HR account、HR-G03…08/OPD-05 authority/effects/handover |
| Finance runtime | DESIGN READY baseline؛ RUNTIME VALIDATION REQUIRED | real Finance/nonAdmin accounts absent؛ not Admin-equivalent |
| F01 ordinary return/reset safety | IMPLEMENTATION CONTRACT READY؛ IMPLEMENTATION BLOCKED rollout؛ RUNTIME VALIDATION REQUIRED | union dependency/race/history proof، OPD-03 release conditional |
| Finance F06/F07 | DESIGN READY boundaries؛ IMPLEMENTATION BLOCKED sensitive/pure-read acceptance | separated authority و no unintended business credit proof |
| Historical hierarchy / attribution | DESIGN READY honest timeline؛ IMPLEMENTATION BLOCKED full as-of claims | HR-G06/07/OPD-07 interval/snapshot coverage |
| Posting / idempotency / unknown recovery | DESIGN READY states؛ IMPLEMENTATION CONTRACT READY invariants؛ IMPLEMENTATION BLOCKED execution readiness | OPD-08/09، FIN-LOCK/IDEMPOTENCY/RECOVERY/COVERAGE |
| KPI current/event contracts | DESIGN READY؛ IMPLEMENTATION CONTRACT READY | F02/03/04 reconciliation؛ actual values not fixed |
| Historical-as-of / rates/targets | IMPLEMENTATION BLOCKED claim/activation | F09/OPD-07 history/cohort proof |
| Refund / cancellation release | DESIGN READY conditional structure؛ IMPLEMENTATION BLOCKED policy-dependent execution | OPD-03/04/06؛ flag not bank proof |
| New skip-level/default assist/escalation | IMPLEMENTATION BLOCKED extension | OPD-02/10/SD-04؛ current paths preserved |
| Customer lifecycle cross-system execution | DESIGN READY branch contract؛ RUNTIME VALIDATION REQUIRED | actual existing Complaint/API/Biawin recipient/result proof؛ no redesign |

Foundation planning می‌تواند بعداً این classifications را ورودی بگیرد؛ این سند آن مرحله را شروع نمی‌کند. Runtime omission HR/Finance design blocker نیست و activation readiness را نیز آزاد نمی‌کند.

## 22. Compatibility

**KEEP + ADAPT** برای تمام integration. Shared conceptual identity/request/audit/report نیاز به DB merge، ID rewrite، status rewrite، history rewrite، financial relink، forced migration یا permission reset ندارد. Existing native IDs/routes/aliases/source origin/transfer histories/invoice-stage-payment/refund/ledger/run/key refs و reviewer/temporal history حفظ می‌شوند. Shared read-through meaning به معنی table unification نیست.

Migration یا structural rewrite اگر در آینده لازم تشخیص داده شود: **MIGRATION CANDIDATE — NOT APPROVED**؛ در V1 هیچ‌کدام default requirement نیست. نام‌گذاری presentation namespace-qualified، stored values را rename نمی‌کند. Role Spec تغییر تنها explicit future revision؛ این سند silent override نیست. Unknown history نه پاک می‌شود نه با current hierarchy جعل می‌شود.

Risky future changes به مالی، access، handover، protected return و engine routing **FUTURE IMPLEMENTATION — REQUIRES QA / ROLLBACK**؛ بحث فنی deployment/migration strategy اکنون ممنوع. Local hash preservation فقط unchanged workspace code را اثبات می‌کند، installed bytes parity نیست.

## 23. Design System Handoff

### GLOBAL CONCEPTS CLAUDE MAY ASSUME

این global WHAT/WHY contracts، هشت role boundary، proof-first identity، distinct custody/origin/next/event/credit، independent financial stages و domain-qualified states؛ shared reports with grain/scope/time/coverage؛ role-specific action visibility. Seller frozen design foundation visual authority آینده است؛ prototypes هیچ financial/workflow policy تعیین نمی‌کنند.

### GLOBAL CONCEPTS CLAUDE MUST NOT INVENT

Canonical ID/schema encoding، DB merge، WP cap، stored status IDs، blanket hierarchy writes، customer lifecycle redirects، bank refund proof/settlement execution، maker/checker threshold، entitlement routing، universal ACK/SLA، current-history backfill، rank-based approval، new archive restore/Fix All یا outcome success بدون evidence.

### CROSS-ROLE SHARED COMPONENT NEEDS

Identity/source context؛ current-vs-historical ownership context؛ scoped filters/explorer/report metadata؛ request/reviewer/result history envelope؛ next actor/blocked reason؛ truthful bulk per-item outcome؛ sensitive-action impact context؛ reconciliation/unknown outcome context؛ temporal/audit context. این needs مفهومی‌اند؛ هیچ component، interaction، layout یا visual state طراحی نمی‌شود و reusable component آماده فرض نشده است.

### ROLE-SPECIFIC COMPONENTS THAT MUST REMAIN ROLE-SPECIFIC

Seller contact/action work؛ Supervisor ready-assignment authority؛ Senior multiteam oversight؛ Manager extra-number review/reason-specific Archive Explorer؛ Deputy leadership accountability؛ MIS parser/source quality/maintenance؛ HR workforce/access/compensation/final apply؛ Finance evidence review/refund/ledger/run posting. Shared frame data/actions آن‌ها را universal نمی‌کند.

### TERMINOLOGY TO STANDARDIZE

Case vs source batch/row/legacy lead؛ Customer vs Phone؛ current custody/original source owner/original Seller/next actor/event actor/credit owner؛ submitted/validated/stage approved/fully paid/completed sale؛ requested/approved step/applied workforce change؛ earned/preview/approved posting/wallet credit/settled/reversed؛ current snapshot/event range/as-of. Accepted ناراضی/بدون خدمات/خدمات مازاد branch names و existing domain namespaces حفظ.

### GLOBAL STATES TO STANDARDIZE

Loading، Empty، No Result، Unauthorized، Stale، Incomplete، Conflict، Partial Success، Retryable Failure، Outcome Unknown، Offline؛ explicit known Failed context و per-effect proof. Financial uncertain commit retry blocked؛ transport status نتیجه business intent نیست.

### DEFERRED / CONDITIONAL UX

OPD-01…10 و register§19؛ runtime HR/Finance؛ new escalation/ACK/skip-level/receiving rights؛ policy-bound refund/release؛ sensitive HR access/compensation/credentials؛ engine routing/posting/unknown recovery؛ rate/target/as-of و export coverage. Claude می‌تواند honest unavailable/conditional context طراحی کند؛ activation، policy و data بسازد یا tests موفق فرض کند نه. **No Claude execution in this task.**

## 24. Freeze Review

| Freeze check | Verdict |
|---|---|
| Eight role missions/authorities coexist | YES؛ shared readers، distinct named sensitive action owner؛ no rank-derived final authority |
| Identity/ownership/financial dependency consistent | YES target؛ current F01/coverage/engine gaps explicit |
| Customer lifecycle branches preserved | YES contract؛ end-to-end runtime NOT VERIFIED؛ no Biawin redesign |
| States/KPIs/time/handoffs share meaning | YES؛ namespaces و request envelope preserve domain authority؛ values/coverage not claimed fixed |
| Open policies deduplicated with owners | YES register§19؛ aliases traceable، scoped activation prerequisites |
| True unresolved architecture blocker | NONE within conditional frozen baseline؛ design acceptance≠implementation readiness |
| Seller/Supervisor input limitation | disclosed؛ user approved A/G/D basis؛ eight standalone Specs falsely claimed NO |
| Existing role Specs silently modified | NO |
| Live/code/DB/permission/data/implementation changes | NONE؛ documentation artifact only |

محلی: 24 بخش ordered، source links و unchanged code/input hashes در [validation evidence](../evidence/cross-role-integration-validation-2026-10-07.md) بررسی می‌شوند. این verification مالی یا business runtime QA نیست.

CRM CROSS-ROLE PRODUCT ARCHITECTURE V1 FROZEN
READY FOR DESIGN SYSTEM INTEGRATION + IMPLEMENTATION FOUNDATION PLANNING
NO CODE CHANGED.
