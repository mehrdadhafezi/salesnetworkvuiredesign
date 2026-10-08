# SALES DEPUTY PRODUCT SPEC

تاریخ: 2026-10-06 · **FROZEN PRODUCT TARGET / IMPLEMENTATION DEFERRED** · **PRODUCT / WHAT–WHY ONLY**.

## 1. Executive Summary

معاون فروش نقش رهبری فروش در قلمرو مجاز، هماهنگی بین Managerها، پایش hierarchy/عملکرد و تعیین مسئول پیگیری استثناها دارد. صف روزمره Manager را تصاحب نمی‌کند. Direct Allocation موجود حفظ می‌شود، اما فعال‌سازی در ساختار هدف **CONDITIONAL / NEEDS POLICY** است؛ visibility گسترده اختیار عملیات همه descendants نیست.

ساختار WHAT/WHY برای Claude Freeze است؛ write rollout، KPI نهایی و policyهای باز Freeze نشده‌اند. مبنا **KEEP + ADAPT** و تمام **Gate 0 INV-001…INV-036** است. هیچ طراحی UI، کد، permission، داده یا workflow ذخیره‌شده تغییر نکرده است.

مراجع:

- **A:** [CRM Product & Role Architecture Audit](CRM-PRODUCT-ROLE-ARCHITECTURE-AUDIT.fa.md)، بخش Deputy و F01–F10.
- **G:** [Gate 0](GATE-0-PRODUCT-INVARIANTS.md)، قرارداد Case/Source، مالکیت، مالی، metric، permission، pure-read و compatibility.
- **SS / M:** [Senior Spec](SENIOR-SUPERVISOR-PRODUCT-SPEC.md) و [Sales Manager Spec](SALES-MANAGER-PRODUCT-SPEC.md)، مرزهای نقش و shared contracts.
- **C / S / D:** [Code Registry](CODE-EVIDENCE-AND-ACTION-REGISTRY.md)، [Status Catalog](STATUS-TRANSITIONS.fa.md)، [Decisions / Visibility](DECISIONS-VISIBILITY-INTERACTIONS.fa.md).
- **L:** [Deputy live re-verification](../evidence/deputy-reverification-2026-10-06.md)؛ [Manager F04/F05 تازه](../evidence/manager-reverification-2026-10-06.md). [Deputy inventory قبلی](../evidence/deputy-loaded-panels-live.json) تاریخی باقی است.

Evidence: **LIVE VERIFIED** مشاهده همان رفتار؛ **CODE VERIFIED** وجود guard/query/transition در منبع پذیرفته‌شده؛ **BOTH** توافق درباره همان claim؛ **INFERENCE** قرارداد target؛ **NOT VERIFIED** نبود شاهد کافی. UI presence و source guard موفقیت write یا negative enforcement را ثابت نمی‌کنند. Confidence inventory/scope code HIGH؛ cause/scale/write runtime NOT VERIFIED.

Live: Admin → HR → Deputy #2، مسیر `/crm-sales-deputy/`؛ نسخه فعال **2.0.123** یک‌بار مشاهده و پس از بررسی Admin بازگردانده شد. version مشترک اثبات byte equality نیست. wallet عمداً باز نشد؛ side effects داخلی renderer بدون DB snapshot تضمین نمی‌شوند.

## 2. Role Mission & Boundaries

| قرارداد | WHAT / WHY |
|---|---|
| Role Mission | رهبری قلمرو scoped و هماهنگی cross-Manager؛ تشخیص مانع و پاسخ‌گویی بر پایه داده معتبر |
| Daily Jobs | مشاهده شاخه‌های نیازمند توجه؛ workload/custody و source coverage؛ unresolved exceptions و next actor؛ پایش عملکرد بدون ranking |
| Occasional Jobs | drilldown Manager→Case/Invoice؛ گزارش دوره‌ای؛ direct allocation موجود و مشروط؛ HR assigned step؛ حساب شخصی |
| Decision Responsibilities | تعیین نیاز هماهنگی بین Managerها؛ انتخاب مسیر تخصیص تحت اختیار اثبات‌شده؛ HR review وقتی current reviewer؛ تشخیص داده نامعتبر قبل تصمیم مدیریتی |
| Data Needed | hierarchy مجاز و فعالیت افراد؛ source-aware workload؛ invoice/payment/review context؛ owner/next actor؛ metric grain/cohort/time/coverage؛ transfer/history |
| Data Not Needed | raw import/repair، report engineering، credentials/compensation، commission rules، banking غیرضروری و out-of-scope customer data |
| Allowed Responsibilities | read scoped؛ coordination در مسیرهای موجود؛ write فقط با action-specific authority بخش 15 |
| Explicit Non-Responsibilities | Manager queue operator عمومی؛ Seller operator؛ Finance review/refund/posting؛ HR final apply/admin؛ MIS import/repair؛ system Admin |

Mission **INFERENCE / HIGH** بر پایه A/L است؛ job frequencies قرارداد هدف‌اند، نه telemetry. Deputy broad visibility به معنی entire organization access نیست. حتی اگر یک شاخه نمونه تمام داده تست را پوشش دهد، scope از hierarchy/service مجاز تعیین می‌شود.

## 3. Current Verified Inventory

۹ destination و مرکز گزارش مشترک؛ current panel تب مستقل HR/archives/extra-number ندارد. Readهای جدول پنهانِ rendered، وجود داده را نشان می‌دهند نه کارکرد drilldown بصری آینده.

| Section | Current evidence | Classification | Decision / rationale |
|---|---|---|---|
| `sd-overview` | Sellers1 active1 leads0 invoice1 preinvoice0 amount15,000 paid0؛ L/C | ESSENTIAL | KEEP + ADAPT به Leadership Overview؛ metric trust قبل تصمیم |
| Free-text invoice-status filter | L/C | CONFUSING | SIMPLIFY vocabulary با status namespace؛ values/URLs قدیمی محفوظ |
| `sd-hierarchy` | Manager3→Senior4، direct Sellers0، descendant1؛ L/C | ESSENTIAL | KEEP current structure جدا از historical attribution |
| `sd-managers` | یک Manager row؛ L/C | ESSENTIAL | MERGE view در Performance Explorer با level=Manager |
| `sd-seniors` | یک Senior و parent Manager؛ L/C | REDUNDANT در presentation مستقل | MERGE drill view؛ entity و parent data حذف نشوند |
| `sd-supervisors` | یک Supervisor و parent Senior؛ L/C | REDUNDANT در presentation مستقل | MERGE همان facts با level واضح |
| `sd-sellers` | Seller و parent Supervisor؛ L/C | REDUNDANT در presentation مستقل | MERGE drill؛ no sum across ancestors |
| `sd-distribution` | balance0، چهار گیرنده Manager/Senior/Sup/Seller؛ count/note/source/return؛ L/C | SUPPORTING | KEEP + CONDITIONAL؛ authority policy جدا از technical path |
| `sd-invoices` | یک pre-invoice15,000؛ ستون‌ها بدون action؛ L/C | SUPPORTING | MOVE به contextual drill/Reports؛ KEEP readonly دسترسی موجود |
| `sd-wallet` | nav موجود؛ A/C/L presence | SUPPORTING | MOVE Own Account؛ شخصی و pure-read |
| Shared report center | period/current mode، role/person؛ L/C/A | ESSENTIAL | KEEP governed Reports |
| Exception coordination / coverage / metric trust | contract کامل فعلی اثبات نشده | MISSING | ADD target requirement؛ نه escalation command یا engine جدید |
| HR actionable entry | create/review shared code مجاز؛ tab فعلی غایب | MISSING / NEEDS VALIDATION | CONDITIONAL entry برای assigned step، نه HR administration |
| User ID | header identity، KPI مستقل امروز مشاهده نشد | SUPPORTING identity | KEEP identity؛ KPI نسازید |

Code locator: [class-sn-plugin.php](../source/sales-network-v4/includes/class-sn-plugin.php)، `render_sales_deputy_panel:8333`، scope construction `8351`، invoice table `8471`، lead count `8646`، preinvoice predicate `8663`؛ code accepted ZIP است، نه استخراج installed build. اصل UI section existence BOTH؛ runtime writes NOT LIVE VERIFIED.

## 4. Deputy vs Manager / MIS

### Deputy vs Sales Manager

| Capability | Sales Manager | Sales Deputy | Why Different? |
|---|---|---|---|
| Organizational scope | قلمرو چند تیم خود | شاخه‌های Manager در scope مجاز | هماهنگی بین مدیران، نه کل سازمان خودکار |
| Hierarchy level | Senior/Sup/Seller | Manager→Senior→Sup→Seller | مسئولیت بالاتر در oversight، نه مجوز بالاتر تمام writes |
| Distribution | عملیاتی owned pool | owned pool و چهار سطح فنی موجود | direct allocation معاون مشروط به policy |
| Direct allocation | سه سطح eligible | Manager و پایین‌تر eligible | bypass مسیر Manager نیاز policy/trace، حذف خاموش ممنوع |
| Archive responsibility | Explorer و review operational واقعی | baseline queue اختصاصی ندارد | read/monitor extension مشروط، نه inherited revival |
| Extra-number | Manager-owned pending approve/reject | handler Manager را به ارث نمی‌برد | تصمیم row-owned، نه rank-based |
| HR requests | scoped submit/assigned review | shared code submit/assigned review؛ nav غایب | current reviewer مهم‌تر از رتبه |
| Invoice visibility | list + resend/copy/history controls | recent table readonly؛ no actions | assist rights خودکار از Manager منتقل نمی‌شوند |
| Performance | تیم‌ها و operational cases | cross-Manager / سلسله‌مراتب | facts مشترک با dimension متفاوت |
| Exceptions | حل operational issue در authority خود | coordination و تعیین next actor | Deputy operator تمام queueها نیست |
| Reports | Manager scope | permitted cross-Manager scope | scope service، نه unrestricted organization |
| Wallet | شخصی | شخصی | revenue تیم/سازمان نیست |
| Bulk actions | guards موجود برای distribution/return | مسیرهای خودش shared guard، سایر bulk اثبات نشده | upward-copy permission ممنوع |
| Operational authority | request/queue-specific | read leadership + conditional allocation/assigned HR | leadership ≠ Admin/Finance/HRfinal |

### Deputy vs MIS / Executive Management

Deputy نیازمند منشأ، coverage و discrepancy برای تصمیم فروش است؛ **MIS** مالک ورود/import، کیفیت منبع، repair مجاز و traceability داده است. Report engineering، ساخت query یا اصلاح lineage کار Deputy نیست. مشاهده گزارش MIS یا unresolved source مجوز raw repair نمی‌دهد.

«Executive Management» در این Spec **لایه تصمیم رهبری** است، نه WordPress role یا مجوز CRM جدید. نیاز تصمیم درباره capacity/مسئول پیگیری با کنترل‌های Finance/HR/Admin متفاوت است. Deputy برای leadership oversight باید operational context را ببیند؛ انجام تماس Seller، review صف Manager، refund یا ذخیره تنظیمات سیستم وارد نقش نمی‌شود. A/C/G مبنا؛ comparison **INFERENCE / HIGH**.

## 5. Target IA

| Module | Placement | WHAT / WHY | Decision |
|---|---|---|---|
| Leadership Overview | PRIMARY | شاخه‌های نیازمند توجه، workload، owner و metric trust | KEEP overview + ADAPT؛ no vanity KPI |
| Hierarchy & Performance | PRIMARY | current hierarchy و scoped drill در facts مشترک | MERGE level-specific views، حفظ distinct entities |
| Exceptions | PRIMARY | cross-Manager coordination و owner resolution | ADD target view؛ underlying states موجود، command جدید deferred |
| Reports | PRIMARY | governed reports، time/source/field contracts | KEEP shared Reports |
| Direct Allocation | CONDITIONAL | custody-based چهار سطح موجود، exceptional bypass | KEEP + CONDITIONAL / NEEDS POLICY؛ no silent removal |
| Scoped Invoice Context | SECONDARY / CONTEXTUAL | read invoice/stage/Finance outcome از drill | MOVE presentation؛ route و readonly behavior حفظ |
| HR request context | CONDITIONAL | submit/review assigned طبق کد مشترک | ADAPT entry پس از reachability/guard validation؛ no HR admin |
| Own Account / Wallet | SECONDARY | identity و ledger شخصی | MOVE personal، pure-read dependency |
| Metric / Case lineage diagnostics | ADVANCED / DIAGNOSTIC | incomplete source، mismatch و proof references | ADD read diagnosis؛ no repair UI |

Archive/extra-number leadership aggregates در صورت وجود authorized data **CONDITIONAL**؛ queue تصمیم Manager به Deputy اضافه نمی‌شود. Target IA **INFERENCE / HIGH**؛ current implementation/evidence بخش 3. هیچ navigation/layout بصری طراحی نشده است.

## 6. Leadership Overview

Overview باید به پرسش قابل تصمیم پاسخ دهد، نه رتبه‌بندی ساختگی:

| پرسش | حداقل داده / تفسیر مجاز |
|---|---|
| کدام Manager/قلمرو نیاز توجه دارد؟ | exceptionهای اثبات‌شده، pending next actor و incomplete data؛ «ضعیف‌ترین Manager» metric ندارد |
| workload/capacity کجا نامتوازن است؟ | scoped active workload/custody و eligible actors؛ تعداد Seller فعال proxy ظرفیت است، capacity کامل نیست |
| چه exception حل نشده؟ | reason/current state، responsible actor، resolution evidence؛ age فقط از timestamps معتبر، بدون SLA |
| داده کجا ناقص است؟ | failed branch/source و coverage/freshness؛ zero جای unavailable نیست |
| distribution bottleneck کجاست؟ | موجودی/recipient eligibility/custody و transfer state؛ owned0 به‌تنهایی bottleneck یا موفقیت نیست |
| Manager action یا Deputy attention؟ | operational authority Manager در برابر cross-Manager coordination و conditional Deputy action |
| next action با کیست؟ | actor از workflow معتبر، نه ancestor فعلی به‌طور پیش‌فرض |

لایه synthesis هدف **INFERENCE** است؛ موارد missing به section18 منتقل‌اند. هیچ alert rule، target، leader board، efficiency score یا capacity formula تازه تصویب نشده. scope/time/source باید در overview summary و drill یکسان باشد. User ID فقط برای identity/troubleshooting.

## 7. Hierarchy & Performance

Expected drill: **Manager → Senior → Supervisor → Seller → Case / Invoice**. چهار جدول موجود views از descendants مشترک‌اند؛ جمع عدد Manager+Senior+Supervisor+Seller double-count می‌سازد. MERGE presentation/facts contract، نه ادغام داده اشخاص یا حذف routes.

| مفهوم | قرارداد |
|---|---|
| Current Hierarchy | parent/descendants فعلی در authorized scope؛ برای navigation/current visibility |
| Historical Attribution | actor/team در زمان رویداد معتبر؛ hierarchy امروز گذشته را بازنویسی نکند |
| Credit Owner | entitlement snapshot و policy مالی؛ از report grouping نتیجه نشود |
| Event Actor | انجام‌دهنده واقعی transfer/issuance/payment/review؛ ancestor فعلی جای آن نیست |
| Grain | team/person dimension، Case/invoice/event measure جدا؛ stage count invoice count نیست |
| Scope | code Deputy برای non-Admin descendant profiles و scoped Seller IDs می‌گیرد؛ broad scope نه entire organization |

هر drill filters/time/cohort را حفظ کند یا تغییر را اعلام کند؛ inferred joins به phone مجاز نیست. missing branch/parent در Partial Hierarchy حالت incomplete دارد؛ absence به معنی inactive/no sales نیست. Source evidence: C `8333/8351/8716` و L single branch؛ multi-Manager runtime **NOT VERIFIED**. queryها هنگام renderer برای tables محاسبه می‌شوند؛ ریسک load CODE VERIFIED، latency عددی اندازه‌گیری نشده.

## 8. KPI Contract

FINAL KPI نیازمند **name، grain، source، cohort، time_basis، scope، formula/predicate، freshness** است. جدول تعریف زیر target contract است؛ پیاده‌سازی و commercial cohortهای باز **NOT FINAL**. Metadata واقعی هر query الزامی است، نه صرفاً title زیبا.

| KPI / grain | Source / formula هدف | Cohort / time / scope / freshness |
|---|---|---|
| Invoice Created / invoice | distinct valid invoice ID، نه stage/event rows | created invoice cohort و creation time؛ scoped Seller/invoice؛ query/as-of و coverage |
| Open Pre-invoices / invoice | وضعیت جاری پیش‌فاکتور از namespace معتبر؛ `pre_invoice` در mapping مربوط | Current Snapshot از eligible invoices؛ current scope و completeness؛ Issued تاریخی نیست |
| Pre-invoices Issued / issuance | رویداد صدور معتبر، unique issuance key؛ بعداً paid هم صدور باقی است | issuance Event Range؛ actor/team زمان event؛ history coverage، absent data≠0 |
| Sales Completed / invoice | وصول معتبر کل invoice با G amount conservation/tolerance؛ receipt/مرحله approved کافی نیست | completion cohort/time و attribution policy؛ OPD-06/07 باز، same scope/freshness |
| Total Cases / logical Case | union source-aware با aliases اثبات‌شده و distinct Case؛ unresolved incomplete | current workload snapshot یا event cohort جدا؛ Case scope، source coverage/time |
| Legacy Leads Only / legacy lead | فقط distinct `sn_leads.id` | legacy creation/assignment cohort اعلام‌شده؛ legacy-only scope و freshness |

**F02 — BOTH / HIGH:** L preinvoice0 در overview مقابل row پیش‌فاکتور. C `sn_sales_deputy_count_invoices:8654` bucket فقط pending/draft/unpaid/empty را می‌شمارد و `pre_invoice` ندارد. labelهای invoice/payment/review نباید بی‌قاعده مخلوط شوند؛ status paid/approved در هر ستون کد فعلی proof وصول کل نیست.

**F03 — CODE VERIFIED / HIGH:** lead query `8646` فقط sn_leads است. live leads0 به معنی zero کل Cases نیست؛ منابع V4/MIS/Dot امروز re-audit نشده‌اند. Case resolver target G، phone discovery-only و unresolved fail-closed برای writes وابسته.

**F04:** accepted discrepancy Manager report0/invoice1 در M/L تازه هنوز برقرار است. Deputy امروز overview invoice1 و table1 داشت؛ **F04 در Deputy امروز بازتولیدشده اعلام نمی‌شود**. leadership باید same-cohort reconciliation و incomplete/trust state داشته باشد؛ cause cache/build/query/cohort NOT VERIFIED، fix فرض نشود.

Missing≠zero؛ limited recent invoice table≠full scope total؛ amount/unit با count جدا. ranking، target، efficiency score، conversion rate و SLA تعریف نشده‌اند. OPD-07 cohort/window/attribution و OPD-06 rounding/currency مانع اعتماد metric نهایی‌اند، نه طراحی خواندنی explorer.

## 9. Direct Allocation / Return

### Direct Allocation

Current L: own balance0 و چهار recipient Manager3/Senior4/Sup5/Seller6؛ controls وجود دارند و writes اجرا نشدند. Shared recipient helpers در C eligibility مجاز/active را می‌سازند؛ وجود descendant نه grant همه writes. **KEEP + CONDITIONAL / NEEDS POLICY** برای business authority/skip-level.

| موضوع | قرارداد target |
|---|---|
| Ownership/custody | actor باید موجودی/action eligibility خود را ثابت کند؛ دیدن کل hierarchy حق تصرف pool Manager نیست |
| Recipient | scope همان action، active و destination eligible در apply؛ نه صرف انتخاب در UI |
| Direct vs skip-level | Manager direct مسیر معمول ساختار؛ Senior/Sup/Seller technical paths موجود محفوظ؛ bypass policy/reason مشروط و traceable |
| Source | SourceRef/CaseRef با IDs بومی و lineage؛ phone canonical نیست |
| Financial dependency | consumed/protected cases طبق G ordinary allocation/return-dependent policy؛ link ambiguity مجوز آزادسازی نمی‌دهد |
| Preview | items/count/source/recipient/block reason؛ snapshot، نه authority نهایی |
| Apply recheck | actor/recipient scope و active status، row custody/state/financial dependency با race protection |
| History / partial results | actor/from/to/time/reason/results، per-item applied/skipped/failed/unknown؛ no all-success روی partial |

Reason requirement هدف traceability است؛ policy جدید approval bypass یا mandatory field بدون rollout صریح اجرا نشود. scope کل سازمان، بین دو Manager آزاد یا administrative repair از این flow نتیجه نمی‌شود.

### Return / Recall — G F01

Current shared return handler C `20731/20758` Deputy position را می‌پذیرد و به historical forward/delivery actor و current custody تکیه دارد؛ financial protection کامل در آن مسیر اثبات نشده. رتبه Deputy حق recall همه descendants نمی‌دهد.

Allowed return basis: action permission + transfer proof معتبر actor + current row/state eligibility. Current custody، original owner و credit owner مستقل؛ transfer actor امروز ancestor بودن را جایگزین نمی‌کند. طبق G§6، INV-006…010:

- همه valid financial links legacy/V4/follow_invoice/Seller state/event/Dot بررسی شوند؛ یک `lead_id` کافی نیست.
- consumption/fulfillment فعال ordinary return را block کند؛ unknown/contradictory linkage **fail closed**.
- cancelled/rejected خودکار release نیست؛ OPD-03 باز می‌ماند.
- preview≠apply؛ concurrent invoice/return نیاز recheck/atomic protection؛ history مالی/تخصیص حفظ.

**Target protection، اصلاح اجرا نشده؛ unsafe runtime loss NOT VERIFIED.** F01 activation dependency P0 است؛ نوشته «موارد امن» در UI ضمانت نیست. نیازمند QA/rollback آینده؛ rank، empty financial view یا phone match مجوز برگشت نیست.

## 10. Exceptions

Underlying states از A/C/G؛ coordination view target **INFERENCE** است. «next actor» نام مسئول مسیر موجود است، نه command یا notification تازه. Evidence ف04 Manager در M، سایر failures ممکن/guard-supported و نه رخداد live اثبات‌شده‌اند.

| Exception / owner | Deputy responsibility / allowed action | Prohibited action | Next actor / resolution signal |
|---|---|---|---|
| Manager unresolved operational issue؛ operational owner | read context و هماهنگی cross-Manager در policy موجود | تصاحب یا approve همه queueها | همان Manager/actor؛ state/outcome evidence؛ escalation formal deferred |
| Cross-team capacity/workload؛ Managers | مقایسه workload scoped و eligible actors | target/score ساختگی یا transfer خارج authority | Managers؛ workload facts و برنامه مجاز معلوم، SLA ندارد |
| Distribution conflict؛ custody actor | reread scope/custody/recipient، conditional own action | overwrite pool Manager | authorized custody actor؛ valid result یا explicit conflict |
| Stale/incomplete metrics؛ report owner | coverage/time/source و drill، جلوگیری از تصمیم غلط | missing→0، ranking روی ناقص | report owner موجود؛ declared coverage/freshness |
| F04 unresolved؛ report/invoice owners | same-cohort reconcile مشاهده‌ای | relink/status/amount edit برای تطبیق | owners داده/گزارش؛ parity یا توضیح population معتبر |
| Inactive hierarchy actor؛ HR owner | دیدن eligibility restriction و هماهنگی | activate/edit user/hierarchy | authorized HR؛ state معتبر یا recipient جایگزین |
| Consumed-return conflict؛ مالی/Case dependency | read block reason و حفظ work | recall با rank یا clear links | actor فروش/مالی مربوط؛ protected outcome آشکار |
| Unresolved lineage؛ source owner | diagnose refs، incomplete، fail closed | merge phone / raw MIS repair | authorized data owner؛ proof resolve؛ routing جدید deferred |
| Invoice needs action؛ sales/Finance actors | see review reason/stage/next actor | Finance overturn/refund/assist بدون grant | Sales responsible actor؛ valid next transition |
| HR chain waiting؛ current reviewer | فقط review اگر خود Deputy assigned؛ observe مجاز | bypass chain/final apply | next reviewer/HR؛ step versus applied/failed روشن |
| Organizational scope conflict؛ action/service owner | scope diagnosis و stopping invalid action | unrestricted org assignment/export | actor مجاز موجود؛ scope ثابت یا denial روشن |

Resolved ممکن است «write مجاز نیست، مسئول معلوم شد» باشد؛ لازم نیست Case منتقل یا مالی آزاد شود. owner unresolved به‌عنوان UNKNOWN بماند؛ rank Deputy default owner نیست. Formal escalation/ACK/SLA **DEFERRED**.

## 11. Invoices

Deputy inventory فعلی readonly recent invoices است: code/Seller/state/amount/payment status/date، بدون button/link در جدول. C `8471` همین renderer و `sn_sales_deputy_recent_invoices:8775` مسیر scoped را دارد. **BOTH / HIGH** presence؛ negative authority tests و full detail completeness NOT VERIFIED.

Target MUST SEE وقتی scoped: invoice، Manager/Senior/Supervisor/Seller context، Case/SourceRef، invoice lifecycle، payment stage، Finance review، total/paid/remaining/unit، reason/next actor و freshness/completeness. افزودن fieldها read contract target است، نه وعده endpoint فعلی.

| Control / action | Classification | Decision / evidence |
|---|---|---|
| Recent invoice table | READ | KEEP / MOVE contextual؛ L/C؛ bounded sample not full coverage |
| Shared report invoice drill | READ | governed Reports context؛ shell دیده شد، full data behavior امروز اجرا نشد |
| Copy/resend payment link | ASSIST / NEEDS VALIDATION | در Deputy controls دیده نشد؛ از Manager ارث نمی‌رسد، no assumed grant |
| Receipt / prepayment edits | WRITE / RESTRICTED در baseline | scoped view مجوز نیست؛ C internal payment action guard `29392` management list Deputy را شامل نمی‌کند؛ سایر guards/action rights globally disproven نیستند |
| Finance approve/reject/refund/posting | RESTRICTED | خارج mission؛ Financial report visibility authority نمی‌دهد |
| Commission rules، cancel/relink | RESTRICTED / NEEDS VALIDATION بدون action proof | مقام Deputy مجوز تغییر مالی/identity نیست |

Invoice state، payment completion و Finance review مستقل‌اند. staged approval یا status_label فعلی proof completed sale نیست؛ G financial contracts حاکم. F04 و field completeness قبل trusted summary. هیچ invoice action اجرا نشده و action جدید پیشنهادی قطعی نیست.

## 12. Archive / Extra-number Boundary

هیچ Deputy destination/control برای archive/extra-number در L دیده نشد. این absence فقط current panel را توصیف می‌کند؛ کل shared endpoints یا آینده را نفی نمی‌کند.

| نیاز / action | تصمیم Deputy | Evidence دقیق / شرط |
|---|---|---|
| Manager archive aggregates | CONDITIONAL READ / MONITOR | target leadership ممکن است به reason/backlog نیاز داشته باشد؛ Deputy-specific aggregate موجود در panel اثبات نشده؛ authorized scoped data لازم |
| Drill archive records | CONDITIONAL READ / NEEDS VALIDATION | Seller Flow `render_manager_archives:980` و Dot `render_manager_payment_archive_tab:3395` manager-owned scope دارند؛ Deputy visibility خودکار از hierarchy نتیجه نشود |
| Act/revive/delete archive | NOT IN ROLE در baseline | Deputy-specific policy/handler اثبات نشده؛ Manager هم generic revival اثبات‌شده ندارد؛ 3/5-day/payment rules M حفظ |
| Extra-number backlog aggregate | CONDITIONAL READ / MONITOR | useful leadership signal فقط با authorized scope/population؛ query/data فعلی Deputy NOT VERIFIED |
| Review Manager-owned extra-number | NOT IN ROLE | C `ajax_manager_review_extra_number:37594`: Admin یا sales_manager position؛ non-Admin manager_id actor و pending؛ Deputy rank جای row owner نیست |
| Request outcome context | CONDITIONAL READ | request/resulting legacy lead provenance و field guard باید مستقل ثابت شود؛ phone canonical نیست |

هیچ grant برای these reads ساخته نشده؛ طراحی extension فقط با unavailable/unauthorized state و conditional requirement ممکن است، نه data ساختگی یا guaranteed access. Manager queue به Deputy copy نشود. business archive reasons و historical request links unchanged؛ deletion/approve-all/forced reassignment ممنوع در baseline هدف.

## 13. HR Requests

Shared code Deputy را در `sn_hr_change_request_can_create:10284` می‌پذیرد؛ subjects descendants Seller در `10296/10311`. C current reviewer guard `10397` و step transitions `10439` طبق chain مشترک. Deputy renderer `8333–8496` entry جدا ندارد؛ L نیز absence تأیید کرد. **CODE VERIFIED eligibility؛ UI entry/review execution NOT LIVE VERIFIED**.

| جزء | قرارداد Deputy |
|---|---|
| Submit scoped | transfer_seller / terminate_seller برای permitted Seller؛ target Supervisor/reason و pending conflict guards؛ no user/credential edit |
| Subject scope | current permitted descendant Seller؛ visible entire subtree مجوز تمام نوع HR action نیست |
| Current reviewer | request assigned to Deputy در pending_review؛ not merely ancestor |
| Next reviewer | next parent reviewer طبق hierarchy chain یا pending_hr؛ Deputy آخرین reviewer غیرHR بودن universal نیست |
| Approval | intermediate approve→next step یا pending_hr؛ reject→rejected/history؛ final apply فقط HR authority |
| Final outcome | approved/applied_at در HR success؛ failed apply مستقل؛ approved Deputy step ≠ applied personnel change |
| History | requester/subject/decision/reason، from/to، reviewer path، effective/applied time؛ attribution تاریخی محفوظ |

Code candidate target Supervisors ممکن است all active باشد؛ option list final authority خارج scope را ثابت نمی‌کند. HR branch/current reviewer واقعی باید در request تازه دیده شود؛ unknown reviewer label با Deputy حدس زده نشود. **OPD-05 handover DEFERRED**: open work/finance/custody/entitlement بعد transfer/termination بدون business handover proof تغییر‌یافته فرض نشود.

Target: CONDITIONAL assigned HR request context با role/module/row guard؛ source eligibility به معنی route reachability نیست. طراحی HR admin عمومی یا activate final apply در Deputy ممنوع. HR Role-specific runtime همچنان NOT LIVE VERIFIED؛ Admin substitute نیست.

## 14. Reports / Wallet

**Reports:** Manager grouping و Senior/Supervisor/Seller drill در permitted scope. Source coverage، grain/cohort/time، freshness و field security همان G/M/SS contracts؛ مشترک بودن report موجب organization-wide access نمی‌شود. Shared report center shell در L دیده شد؛ report execution/export امروز انجام نشد.

| Time mode | قرارداد |
|---|---|
| Current Snapshot | current state در as-of/query time؛ current hierarchy visibility با historical credit جدا |
| Event Range | event داخل بازه و attribution زمان event؛ creation/assignment/payment times interchangeable نیستند |
| Historical As-Of | فقط reconstructable state/history coverage کافی؛ current fields collected today proof گذشته نیست، F09 |

Broader scope = چند Manager مجاز، نه unrestricted org. Sensitive fields شامل compensation، bank/receipt غیرضروری، financial/internal repair columns باید مستقل restricted باشند. Export explicit permission، same query cohort/source/time/fields، bounds/partial outcome و security parity لازم دارد؛ export هنوز NOT VERIFIED. Recent table / bounded report total به معنی full population نیست.

**Own Account / Wallet:** personal ledger/entitlement خود Deputy؛ نه team commission، organization revenue یا Finance ledger. nav حفظ، entry ثانویه. risky renderer عمداً باز نشد؛ current Deputy renderer dual-wallet call دارد، F07 در codebase accepted conditional effects ریسک است نه وقوع اثبات‌شده در Deputy امروز. Pure-read G§11 binding؛ جابه‌جایی فنی بدون dependency isolation/QA ممنوع، presentation proposal permission/engine را تغییر نمی‌دهد.

## 15. Permissions / Bulk Actions

Conceptual permissions، **نه WP capability names جدید**. common read guards: login/role entry، module permission، row scope و field policy مستقل. C Deputy panel برای non-Admin fast HR position sales_deputy و service availability را بررسی می‌کند؛ descendant/scoped Seller data به‌تنهایی write privilege نمی‌دهد.

| Concept | ارزیابی target |
|---|---|
| `view_deputy_overview` | KEEP scoped leadership facts؛ synthesis target |
| `view_manager_hierarchy` | KEEP current hierarchy؛ history/credit separate |
| `view_scoped_cases` | CONDITIONAL source-complete read/drill؛ current lead-only counts NOT complete Cases |
| `view_cross_manager_performance` | KEEP authorized branches با governed metrics؛ no entire org assumption |
| `assign_scoped_cases` | KEEP technical path، CONDITIONAL policy؛ own custody + eligible recipients |
| `return_eligible_cases` | current handler exists؛ F01 guarded rollout CONDITIONAL |
| `view_scoped_invoices` | KEEP readonly oversight؛ no inherited assist |
| `view_exceptions` | ADD target coordination view؛ data/action authority separate |
| `submit_hr_request` | CODE-supported scoped subject؛ current entry NEEDS VALIDATION |
| `review_assigned_hr_request` | CODE-supported current-reviewer step؛ no final apply |
| `view_reports` | KEEP report/row/field scope/time contracts |
| `export_reports` | CONDITIONAL explicit export authority؛ NOT runtime-tested |
| `view_own_wallet` | KEEP personal؛ pure-read/F07 dependency |

Important WRITE/APPROVAL guards target؛ implementation success ادعا نمی‌شود:

| Action | Scope | State guard | Row guard | Field restriction |
|---|---|---|---|---|
| Direct allocation | own custody، action-eligible recipient | allocatable، active recipient، permitted path | Case/source و current owner در apply دوباره | destination/count/reason/note مجاز؛ no original/credit rewrite |
| Return | own valid transfer/action basis | nonconsumed، all financial links verified؛ ambiguous blocked | transfer proof + current state/custody/race recheck | no clearing invoice links/history؛ OPD-03 reset deferred |
| HR submit | permitted Seller، supported type | valid subject/target، pending duplicate rule | subject/profile scope fresh | type/target/reason؛ no credentials/role/compensation |
| HR step review | request assigned current reviewer | pending_review for Deputy، fresh state | current reviewer unchanged | step decision/reason؛ final application HR only |
| Export | report/cohort/branches permitted | query contract valid و freshness declared | rows همان authorized dataset | approved field whitelist؛ no banking/HR spill |
| Invoice/Archive/Extra-number mutation | no baseline Deputy authority proven | **RESTRICTED / NEEDS VALIDATION** | rank insufficient | controls/fields خودکار اضافه نشوند |

| Bulk | Classification | Evidence / rationale |
|---|---|---|
| Bulk Direct Allocation | SUPPORTED controls / CONDITIONAL policy | shared count-per-recipient flow L/C؛ mission occasional، REQUIRED روزانه نیست |
| Bulk Return | SUPPORTED code / CONDITIONAL activation | shared Deputy guard C؛ F01 حفاظت کامل target لازم |
| Bulk Export | CONDITIONAL | explicit permission/field parity/bounds؛ no execution test |
| Bulk Exception Action | NEEDS VALIDATION | categories/actions heterogeneous؛ generic resolve-all handler/policy اثبات نشده |
| Bulk HR Review | NOT ALLOWED در frozen baseline | assigned requestهای مستقل؛ source single-review guard، batch policy proof ندارد |
| Bulk Archive Review | NOT ALLOWED برای mutation / NEEDS VALIDATION برای read-selection | Deputy action authority اثبات نشده؛ monitor extension conditional |
| Bulk Extra-number Review | NOT ALLOWED | Manager-owned handler با position/row checks؛ ancestor حق approve-all ندارد |

No capability changed by target restrictions. All supported future bulk: requested/eligible/applied/skipped/failed/unknown per item، fresh apply checks و truthful Partial Success طبق INV-034. unknown outcome ابتدا reconcile؛ retry کور ممنوع. Lower-role bulk rights upward-copy نشوند.

## 16. Handoffs

| Path | قرارداد / evidence limit |
|---|---|
| Upstream → Deputy custody | only if actual transfer proof؛ L owned0 receipt را ثابت نمی‌کند؛ technical owned-pool path C موجود |
| Deputy → Manager | own authorized allocation + actor/from/to/time/Case/source؛ L controls، runtime NOT VERIFIED |
| Deputy → Senior/Supervisor/Seller | موجود technical path KEEP، conditional bypass policy/reason؛ no silent privilege expansion |
| Manager → Deputy unresolved exception | فقط policy/path موجود؛ broad coordination target، formal escalation command/ACK/SLA DEFERRED |
| Finance → Sales-visible outcome | scoped invoice stage/review/reason و responsible Sales actor؛ Deputy oversight، no Finance action |
| HR → Deputy review → next/HR | actual current-reviewer assignment، intermediate outcome/history؛ final applied/failed HR |
| Reports → Deputy | read facts/coverage/time/source؛ report result مجوز state change نمی‌دهد |

**Current Custody، Original Owner، Next Actor، Credit Owner** چهار مفهوم مستقل G§4–5 هستند. انتقال فعلی یا parent امروز history/commission attribution را rewrite نمی‌کند؛ Event Actor نیز جداست. هیچ generic ACK، automatic escalation، recipient reassignment یا universal hierarchy recall تعریف نشده. Operational owner unresolved با UNKNOWN اعلام شود.

## 17. Data Visibility / States

| Visibility | قرارداد |
|---|---|
| MUST SEE | authorized Manager branches، current hierarchy/active eligibility، source-aware workload/CaseRef، invoice/stage/review و paid/remaining/unit، owner/next actor/reason، metric coverage/time/freshness، assigned HR path/outcome |
| NICE TO HAVE | filtered branch/source/age context، historical event actor/attribution اثبات‌شده، scoped archive/request monitor پس از authority validation، diagnostic reconciliation |
| SHOULD HIDE / RESTRICT | Finance approval/posting/refund، commission rules/others ledger، HR credentials/compensation، raw MIS repair، out-of-scope customers، unnecessary sensitive bank data، technical statuses پیش‌فرض، User ID as KPI |

Technical raw status در authorized diagnostic با business label قابل استفاده است؛ stored IDs rename/delete نشوند. Customer identity/phone فقط در scoped detail با field need؛ leadership summary به همه PII نیاز ندارد.

| State | Deputy example / meaning |
|---|---|
| Loading | hierarchy/report read pending؛ aggregate نهایی نمایش داده نشود |
| Empty | complete authorized scope بدون data؛ one empty branch≠whole organization empty |
| No Result | فیلتر Manager/time/status match ندارد |
| Unauthorized | branch/action/field denied؛ mask با zero/empty نکنید |
| Stale | cached facts/time قدیمی؛ freshness visible |
| Incomplete | یک Manager branch fail یا partial hierarchy/report source coverage؛ totals NOT FINAL |
| Conflict | recipient changed before apply؛ out-of-scope allocation یا consumed return؛ no overwrite |
| Partial Success | برخی items/branches outcome معتبر، بقیه failed/skipped/unknown؛ counts per item |
| Retryable Failure | known read failure یا failed item قابل retry با recheck |
| Outcome Unknown | allocation/return response نامعلوم؛ reconcile پیش از repeat |
| Offline | network unavailable؛ cached read stale و new write completed فرض نشود |

Exception owner unresolved UNKNOWN است؛ metric discrepancy F04 trust issue است، نه zero مشروع. one Manager branch failed موفقیت دیگر branches را پاک نمی‌کند یا whole tree complete نمی‌سازد. Source incompleteness و lack of authority جدا از transport error‌اند؛ workflow statusها با these response states ادغام نشوند.

## 18. Product Gaps / Deferred

| ID / Priority | وضعیت / Evidence / Confidence | شرط / محدودیت |
|---|---|---|
| SD-G01 / P0 — F01 return | PRODUCT GAP — DO NOT IMPLEMENT؛ C/G HIGH | all-link financial guards/race/history QA قبل write rollout؛ unsafe runtime loss NOT VERIFIED |
| SD-G02 / P1 — F02/F03 | PRODUCT GAP — DO NOT IMPLEMENT؛ L/C HIGH | pre_invoice predicate و honest legacy-only scope؛ full Case resolver/coverage؛ no fixed-by-design claim |
| SD-G03 / P1 — F04 metric trust | PRODUCT GAP — DO NOT IMPLEMENT؛ M/L discrepancy HIGH؛ Deputy reproduction absent | same cohort/time/source reconciliation؛ causes NOT VERIFIED |
| SD-G04 / P1 — cohort/attribution/currency | DEFERRED؛ OPD-06/07 | metadata و complete histories برای final KPI؛ no score/rate/target |
| SD-G05 / P2 — direct-allocation/skip-level policy | DEFERRED / CONDITIONAL؛ L/C HIGH technical path | scope/custody/reason و business authority؛ existing path silent remove نشود |
| SD-G06 / P1 — write authority across hierarchy | NEEDS VALIDATION؛ guard/code proof فقط | action/module/row/field negative tests؛ visibility ≠ write؛ no privilege escalation claim بدون execution evidence |
| SD-G07 / P2 — escalation/exception ownership | DEFERRED؛ OPD-02 | next actor از workflow، command/ACK/SLA تازه نیست |
| SD-G08 / P2 — archive/extra-number visibility | DEFERRED / CONDITIONAL؛ L absence، C Manager-specific scope | aggregate/drill authority/data coverage قبل activation؛ review NOT IN ROLE |
| SD-G09 / P2 — HR entry/reviewer position | PRODUCT GAP — DO NOT IMPLEMENT؛ C eligibility/L no nav | authenticated route/module entry، actual current reviewer؛ no final apply؛ OPD-05 handover باقی |
| SD-G10 / P2 — bulk exception/review | DEFERRED؛ no policy proof | heterogeneous actions و per-item truthfulness؛ no batch permission inheritance |
| SD-G11 / P1 — pure-read wallet | DEFERRED / CONDITIONAL؛ G/F07 | renderer/engine dependency checks؛ side effect occurrence today NOT VERIFIED |
| SD-G12 / P2 — invoice context/hierarchy load | PRODUCT GAP — DO NOT IMPLEMENT؛ L/C | missing stage/source/next actor read contract و partial branch states؛ measured scale/latency NOT VERIFIED |

OPD-01 lineage، OPD-03 cancelled release، OPD-08 maker/checker، OPD-09 entitlement routing و OPD-10 expanded assist همچنان binding/deferred‌اند. P3 extras نظیر saved views تنها پس از نیاز اثبات‌شده و حل trust/source issues؛ در این Freeze compulsory نیستند.

**Design blocker:** ندارد؛ read leadership/structure/context و conditional existing flows قابل طراحی‌اند. **Implementation/activation blockers** بالا باقی‌اند؛ Freeze مجوز اجرا یا policy shortcut نیست. Report engineering/MIS Spec خارج این مرحله می‌ماند.

## 19. Compatibility / Safe Adoption

**KEEP + ADAPT** برای همه target recommendations. Preserve IDs، routes/internal tab IDs، history، legacy sources، source/ownership/invoice relationships، workflows و permissions تا rollout صریح. Merge فقط views/facts contract با adapters سازگار؛ destructive entity merge نیست.

| Target adaptation | Bridge / future acceptance |
|---|---|
| Leadership Overview | existing facts با honest labels/trust؛ no authority/metric formula silently changed |
| Hierarchy/Performance Explorer | existing level routes/context mapped؛ no double-count ancestors؛ history/credit مستقل |
| Conditional allocation/return | current technical paths retained؛ policy/financial guards QA؛ no subtree grant |
| Invoice/Report context | readonly scope/field parity و metadata؛ recent list≠full total |
| Archive/extra-number monitor | activated only with proved read scope؛ Manager decision handlers preserved |
| HR context | current reviewer/chain/final HR distinction؛ no inferred successful handover |
| Wallet placement | presentation entry personal؛ pure-read dependency isolation قبل technical rollout |

No destructive migration، status rename، ID rewrite، invoice relink، historical attribution rewrite یا silent capability expansion. هر تغییر پرریسک **FUTURE IMPLEMENTATION — REQUIRES QA / ROLLBACK**. rollback نباید payments/approvals/history را حذف کند.

Future QA ضروری: multiple Manager branches و out-of-scope denial؛ current/historical/credit/event attribution؛ all-source Case coverage و no phone joins؛ KPI same-cohort parity؛ recipient inactive/reparented before apply؛ own-transfer return با همه financial-link domains و concurrent invoice؛ partial/unknown/offline outcomes؛ HR current reviewer/final failed apply؛ archive/request monitor read-versus-action separation؛ report/export sensitive-field parity؛ wallet pure-read. هیچ چنین write test یا implementation اکنون اجرا نشده.

## 20. Claude Design Handoff

### WHAT CLAUDE MAY ASSUME

Deputy leadership/cross-Manager mission، چهار Primary ماژول IA، Own Account ثانویه و allocation conditional Freeze‌اند. Manager operator نیست. Shared G/SS/M contracts معتبر؛ existing routes/IDs/history محفوظ. Current nine destinations evidence هستند، نه الزام حفظ presentation تکراری. Seller/Supervisor/Senior prototypes فقط visual design foundations؛ business logic از A/G/C/L.

### WHAT CLAUDE MUST NOT INVENT

Finance/HR final/Admin/MIS rights؛ all descendants writable؛ organization-wide visibility؛ universal recall؛ archive/extra-number review Deputy؛ generic escalation/ACK/SLA؛ ranking/target/rate/score؛ phone canonical identity؛ unknown→0؛ snapshot-as-history؛ successful apply از button/code presence؛ F02/F03/F04 حل‌شده با ظاهر جدید.

### SHARED COMPONENTS TO REUSE

Role identity/shell contracts، hierarchy scope/drill، Case/Source context، governed Performance/Reports، invoice stage/review/history read، request reviewer/history، personal account و common system states. Reuse permission inheritance نیست؛ shared visuals، role-specific authority محفوظ.

### DEPUTY-SPECIFIC DESIGN NEEDS

Leadership Overview؛ Manager Hierarchy Explorer؛ Cross-Manager Performance Explorer؛ Exception Coordination؛ Conditional Direct Allocation؛ Metric Trust/Reconciliation. WHAT: owner/next actor، scope/time/coverage و current versus historical context. HOW/layout/styling/interaction visuals بعداً توسط Claude؛ اکنون هیچ design ساخته نشده.

### LIKELY NEW SHARED COMPONENTS

Branch coverage/trust indication؛ Metric Reconciliation context؛ owner-aware exception context؛ Performance Explorer hierarchy extension؛ conditional allocation context؛ assigned Request Review context. این‌ها product needs قابل reuse هستند، نه component implementation یا new command authority.

### DEFERRED / CONDITIONAL ITEMS

Direct-allocation/skip-level policy؛ F01 activation؛ final KPI/cohort/units و F04 reconciliation؛ formal escalation owner؛ archive/extra-number monitor scope و no review authority؛ HR entry/actual reviewer و OPD-05 handover؛ bulk semantics؛ F07 wallet dependencies. Conditional actions با limits واقعی طراحی شوند؛ unresolved business rule با تصمیم بصری نهایی نشود.

**SALES DEPUTY PRODUCT SPEC FROZEN — READY FOR CLAUDE REDESIGN**

**NO CODE CHANGED.**
