# SALES MANAGER PRODUCT SPEC

تاریخ: 2026-10-06 · وضعیت: **FROZEN PRODUCT TARGET / IMPLEMENTATION DEFERRED** · دامنه: **PRODUCT / WHAT–WHY ONLY**.

## 1. Executive Summary

مدیر فروش مسئول هماهنگی عملیات چند تیم و رسیدگی به موانعی است که از صف روزمره فروشنده و نظارت Senior فراتر می‌روند: توزیع تحت اختیار، ناسازگاری گزارش/فاکتور، آرشیو عملیاتی، درخواست شماره و مرحله تخصیص‌شده بررسی HR. این نقش صرفاً Senior با داده بیشتر نیست؛ مسئولیت صف‌ها و تصمیم‌های مشخص دارد، بدون تبدیل‌شدن به Finance، HR نهایی، MIS یا Deputy.

ساختار محصول برای طراحی Claude آماده است. مسائل باز به قابلیت‌های مشروط یا تشخیص خطا محدود می‌شوند؛ مانع طراحی ساختار خواندنی و مسیرهای فعلی نیستند. Freeze به معنی مجازشدن rollout، تغییر دسترسی، اصلاح F01 یا موفقیت اجرای write نیست. اصل حاکم **KEEP + ADAPT** و تمام **INV-001…INV-036** از Gate 0 الزام‌آورند.

مراجع معتبر:

- **A** — [Audit پذیرفته‌شده](CRM-PRODUCT-ROLE-ARCHITECTURE-AUDIT.fa.md)، بخش Manager و F01–F10.
- **G** — [Gate 0](GATE-0-PRODUCT-INVARIANTS.md)، قراردادهای Case/Source، مالکیت، مالی، KPI، permission، pure-read و compatibility.
- **SS** — [Senior Supervisor Spec](SENIOR-SUPERVISOR-PRODUCT-SPEC.md)، مرز نقش و ماژول‌های مشترک.
- **C / S / D** — [Code Registry](CODE-EVIDENCE-AND-ACTION-REGISTRY.md)، [Status Catalog](STATUS-TRANSITIONS.fa.md)، [Decisions / Visibility](DECISIONS-VISIBILITY-INTERACTIONS.fa.md).
- **L** — [بازبینی محدود زنده 2026-10-06](../evidence/manager-reverification-2026-10-06.md). شواهد قدیمی Manager در [inventory اولیه](../evidence/manager-loaded-panels-live.json) حفظ می‌شوند؛ L فقط مشاهدات مشخص امروز را جایگزین می‌کند.

**Evidence policy:** LIVE VERIFIED = مشاهده همان رفتار خواندنی؛ CODE VERIFIED = وجود predicate/guard/transition در منبع پذیرفته‌شده؛ BOTH = توافق این دو درباره همان ادعا؛ INFERENCE = قرارداد پیشنهادی هدف، نه رفتار موجود؛ NOT VERIFIED = شاهد کافی ندارد. وجود UI یا guard اثبات موفقیت write، negative enforcement یا concurrency نیست. اطمینان ساختار HIGH؛ رفتار write فعلی NOT LIVE VERIFIED؛ علت F04 و باقی‌مانده F05 MEDIUM/UNKNOWN.

Live واقعی: Admin → HR → مشاهده پنل **Manager #3**، مسیر `/sales-manager-panel/`؛ نسخه فعال **2.0.123** یک بار مشاهده شد. یکسانی version با ZIP اثبات یکسانی build نیست. پس از بررسی، Admin بازگردانده شد. هیچ write تجاری، export یا تغییر کد انجام نشد؛ نبود side effect داخلی renderer بدون snapshot DB تضمین نمی‌شود.

## 2. Role Mission & Boundaries

| قرارداد نقش | WHAT / WHY |
|---|---|
| Mission | حفظ جریان کار در قلمرو تحت اختیار، توزیع قابل ردیابی و حل/ارجاع استثناهای عملیاتی؛ به‌جای تصاحب کار فروشنده یا تصمیم مالی |
| Daily Jobs | بررسی موجودی و گیرندگان فعال؛ توزیع واجد شرایط؛ پایش پرونده/فاکتور و next actor؛ بررسی درخواست شماره؛ مشاهده درخواست HR که نوبت Manager است |
| Occasional Jobs | بررسی آرشیو و علت خروج صف؛ تحلیل تیم‌ها/منابع و مغایرت‌ها؛ ثبت انتقال/قطع همکاری؛ گزارش دوره‌ای؛ مشاهده حساب شخصی |
| Decision Responsibilities | انتخاب گیرنده مجاز برای موجودی خود؛ بررسی درخواست شماره متعلق به Manager؛ تصمیم مرحله HR تخصیص‌شده؛ تعیین اقدام عملیاتی در مرز اختیار موجود |
| Data Needed | CaseRef/SourceRef، custody و actor تاریخی، hierarchy مجاز، recipient eligibility، financial dependency، invoice/stage/review، دلیل و freshness، request reviewer/history |
| Data Not Needed | credentials کارکنان، compensation، قوانین پورسانت، posting مالی، raw MIS repair، مشتری خارج scope، bank data غیرضروری |
| Allowed Responsibilities | خواندن scoped؛ writeهای جداگانه با guard بخش 15؛ کمک فروش روی فاکتور فقط برای action اثبات‌شده و مجاز؛ هماهنگی بدون بازنویسی مالکیت |
| Explicit Non-Responsibilities | Finance approve/reject/refund/posting؛ HR final apply؛ import/repair MIS؛ تماس/فروش به‌جای Seller؛ مدیریت کل قلمرو Deputy؛ تغییر role/capability/setting Admin |

این تعریف Mission مبتنی بر A و inventory، **INFERENCE / HIGH** است. hierarchy جاری فقط مبنای scope خواندن/eligibility مناسب است؛ اختیار write یا Credit Owner تاریخی از آن استنتاج نمی‌شود. مدیریت استثنا مسئولیت هماهنگی می‌دهد، نه مجوز ابزار اصلاح داده.

## 3. Current Verified Inventory

۹ مقصد Manager و مرکز گزارش مشترک وجود دارند؛ subsectionها یا لینک گزارش را tab مستقل جدید نشمارید. شمارش‌ها فقط نمونه محیط تست‌اند، نه معیار نهایی workload.

| مقصد فعلی | شاهد امروز / مرجع | کلاس | تصمیم محصول و دلیل |
|---|---|---|---|
| `manager-overview` | owned 0، assigned by Manager 0؛ User #3 tile؛ L+C | ESSENTIAL | SIMPLIFY / KEEP + ADAPT؛ خلاصه عملیات و استثنا، نه metric هویتی |
| User ID tile | هویت حساب در tile؛ L | LOW-VALUE | REMOVE از KPI؛ هویت و ID برای تشخیص حساب حفظ شوند |
| `manager-distribution` | ۳ گیرنده Senior/Supervisor/Seller؛ موجودی/active cases صفر؛ count/note/source/level/return؛ L+C | ESSENTIAL | KEEP؛ قابلیت مستقیم/skip-level جدا و مشروط، بدون تغییر handler |
| `manager-report` | result 1، legacy 0، V4 1، invoice 0؛ ۶۰–۱۲۰ ردیف در درخواست؛ L+C | ESSENTIAL | MERGE facts contract با Performance؛ view عملیاتی، source facets و route باقی بمانند |
| `manager-invoices` | یک pre-invoice، total 15,000 تومان، paid/approved 0؛ resend/copy/history؛ L | ESSENTIAL | KEEP؛ action-specific، بدون افزودن اختیار Finance |
| `manager-customer-actions` | فهرست و details اکنون load می‌شوند؛ residual loading؛ L+C | SUPPORTING / CONFUSING برای حالت بارگذاری | MOVE context به Shared Customer Profile؛ KEEP تاریخچه مفید، SIMPLIFY entry مستقل؛ F05 تاریخی دائمی تلقی نشود |
| `manager-archives`: no-answer | count 0؛ شرط سه تلاش + سه روز idle؛ L+C | SUPPORTING | KEEP در Archive Explorer؛ reason و semantics مستقل |
| assessment unpaid | count 1؛ 3-day rule و row مالی؛ L+C | SUPPORTING | MERGE presentation در Explorer؛ timeline و مالی محفوظ |
| subscription/product/product* unpaid | countهای صفر؛ 5-day rule از C/A | SUPPORTING | MERGE presentation، نه thresholds یا transitions |
| `manager-extra-number-requests` | جدول خالی؛ approve/reject handler scoped؛ L+C | ESSENTIAL | KEEP / ADAPT provenance و outcome؛ review runtime NOT VERIFIED |
| `manager-hr-requests` | transfer/terminate، Seller #6، target Sup #5؛ درخواست خالی؛ L+C | SUPPORTING | KEEP؛ Manager step از final HR application جدا |
| `manager-wallet` | nav و initial rendered content؛ L+A | SUPPORTING | MOVE به Own Account؛ شخصی، مطابق pure-read؛ عمداً باز نشده |
| Shared Reports | مرکز مشترک و لینک `/crm-reports/`، time/role/person؛ L+C/A | SUPPORTING | KEEP در Reports؛ scope/field security و زمانی مشترک |
| تعریف KPI، freshness، exception resolution | قرارداد کامل در UI فعلی اثبات نشده | MISSING | ADD به قرارداد هدف؛ اجرای فنی deferred، نه metric ساختگی |

موجودی و controls بالا LIVE VERIFIED؛ binding code در مقصدها CODE VERIFIED، بنابراین BOTH فقط برای وجود توافق‌شده. پنل گزارش invoice0 و invoices1 همچنان **F04** است. نام Seller در ستون Supervisor یک ردیف گزارش دیده شد؛ attribution/mapping **NEEDS VALIDATION**، خطای داده تاریخی قطعی اعلام نمی‌شود.

Code locatorهای محدود برای ردیابی: [Manager tab dispatcher](../source/sales-network-v4/includes/class-sn-plugin.php)، `13513–13543`؛ report adapter `sn_sales_manager_v4_report_rows:13316`؛ customer renderer `13426`؛ request handlers `37531/37594`؛ HR handlers `10284–10439`؛ internal invoice action guard `29392`. این‌ها references به ZIP پذیرفته‌شده‌اند، نه byteهای نصب فعلی.

## 4. Manager vs Senior / Deputy

### Senior Supervisor → Sales Manager

| موضوع | Senior | Manager |
|---|---|---|
| Scope | چند تیم زیر Supervisorهای scoped | قلمرو Manager شامل Senior/Supervisor/Sellerهای مجاز |
| Distribution | owned → Supervisor، Seller مشروط | owned → Senior، Supervisor، Seller مشروط؛ custody مستقل |
| Exceptions | هماهنگی تیم‌ها و آماده‌های تبدیل readonly | استثناهای توزیع، فاکتور/metric، آرشیو و درخواست شماره؛ نه repair عمومی |
| Archives | مسئولیت Manager به آن منتقل نمی‌شود | Explorer اختصاصی و بررسی علت/مالک/وابستگی |
| Extra-number | queue Manager از رتبه ارث نمی‌رسد | pending request با `manager_id` خود؛ approve/reject کد موجود |
| HR | submit scoped / review assigned | همان قرارداد، مرحله متفاوت chain؛ هیچ‌کدام final HR نیستند |
| Invoice assistance | receipt/edit/resend در نمونه Senior دیده شد | resend/copy/history دیده شد؛ receipt/edit از Senior کپی نشود |
| Performance | Supervisor/Seller و conversion context | Senior → Supervisor → Seller؛ reconciliation و sources |
| Reports | scoped چندتیمی | scoped قلمرو Manager؛ time/grain یکسان |
| Bulk | مستقل از Supervisor | distribution/return شواهد مستقل؛ bulk review اثبات نشده |
| Approval | assigned HR step، نه Finance | extra-number تصمیم‌گیر؛ assigned HR step، نه Finance |

### Sales Manager → Sales Deputy

| موضوع | Manager | Deputy |
|---|---|---|
| Scope | قلمرو تحت Manager | قلمرو چند Manager و پاسخ‌گویی کلان |
| Distribution | Senior/Supervisor/Seller مجاز از custody خود | مسیر Manager/Senior/Supervisor/Seller موجود؛ scope و policy مستقل |
| Exceptions | پیگیری صف‌ها و تصمیم‌های operational مشخص | هماهنگی ظرفیت/مانع بین Managerها؛ escalation جدید تعریف نشده |
| Archives | archive explorer واقعی | queue Manager خودکار به Deputy کپی نشود |
| Extra-number | row-owned review | rank Deputy مجوز review درخواست Manager نمی‌سازد |
| HR | submit/review assigned | مسیر chain و reviewer خودش؛ نه final HR |
| Invoice assistance | observed assist controls؛ write مشروط | inventory Audit بیشتر readonly؛ اختیار Manager به آن ارث نمی‌رسد |
| Performance | تیم/پرونده/فاکتور | مقایسه قلمرو Managerها با قرارداد metric مشترک |
| Reports | scope خود | scope گسترده مجاز؛ field/write restrictions محفوظ |
| Bulk | طبق action-specific guard | نه superset خودکار bulk Manager |
| Approval | extra-number + assigned HR | فقط policy/handler اثبات‌شده؛ مقام بالاتر اختیار Finance نمی‌دهد |

منبع: A و SS؛ comparison قرارداد هدف **INFERENCE / HIGH** بر پایه inventory/code. Deputy در این مرحله مجدداً بررسی نشده است.

## 5. Target IA

| ماژول هدف | Placement | دامنه / چرا | تصمیم |
|---|---|---|---|
| Operations & Exceptions | PRIMARY | owned work، request backlog و استثناهای بخش 6 با owner/next actor | KEEP overview + ADAPT؛ اطلاعات اجرایی به‌جای tile هویتی |
| Distribution & Performance | PRIMARY | دو وظیفه مرتبط اما action و analytics مستقل؛ scope، source، سطح تیم | KEEP distribution؛ MERGE facts گزارش با performance، بدون ادغام writes |
| Invoices & Exceptions | PRIMARY | scoped invoices، payment/stage/review و اقدام مجاز | KEEP؛ context مشتری و Case قابل پیگیری |
| Archive Explorer | PRIMARY | یک مجموعه قابل بررسی با پنج reason و قواعد مستقل | MERGE presentation؛ read/review ابتدا، revival مشروط |
| Extra-number Requests | PRIMARY | queue تصمیم Manager و provenance نتیجه | KEEP؛ submit Seller در پنل خودش |
| HR Requests | PRIMARY | درخواست‌های ایجادشده/نوبت بررسی؛ next reviewer و final apply | KEEP؛ عملیات HR عمومی منتقل نشود |
| Shared Reports | PRIMARY | گزارش governed با همان metric/source/time contracts | KEEP / MOVE گزارش‌های مشترک به entry واحد |
| Own Account / Wallet | SECONDARY | identity، رمز شخصی و ledger خود | MOVE؛ separation از revenue تیم |
| Shared Customer Profile | CONTEXTUAL | رفتار/تاریخچه scoped از Case، invoice، archive | MOVE / REUSE؛ حذف business history ممنوع |
| Manual invoice assistance / archive revival / skip-level | CONDITIONAL | فقط action با policy و guard مشخص | NEEDS VALIDATION؛ نمایش اختیار اختراعی ممنوع |
| Metric reconciliation / lineage diagnostics | ADVANCED / DIAGNOSTIC | source gap، mismatch و evidence references | ADD requirement؛ read diagnosis، نه raw MIS repair |

Target IA **INFERENCE / HIGH** است. سازماندهی محتوا تعیین شده؛ sidebar، layout، tab styling یا interaction بصری طراحی نشده‌اند. Routeها و internal IDs فعلی حفظ می‌شوند. empty queue دلیل حذف module نیست.

## 6. Operations & Exceptions

وجود underlying state/action از A/C/L؛ تجمیع در console یک **INFERENCE target** است. «ارجاع» در جدول زیر مشخص‌کردن actor مسئول در workflow موجود است؛ command، notification یا SLA جدید نیست.

| Exception / مالک | مسئولیت Manager / اقدام مجاز | اقدام ممنوع | Next Actor / Resolution signal |
|---|---|---|---|
| Distribution conflict؛ custody actor | بررسی current owner و eligibility؛ preview خواندنی/refresh قبل اقدام مجاز | overwrite ownership یا retry کور | actor صاحب custody/مسئول موجود؛ نتیجه scoped معتبر یا conflict صریح |
| Inactive recipient؛ hierarchy/HR owner | انتخاب recipient فعال مجاز؛ گزارش محدودیت | فعال‌کردن user/تغییر hierarchy | actor مجاز HR؛ eligibility معتبر یا انتخاب جایگزین |
| Consumed-return conflict؛ مالی/Case dependency | مشاهده دلیل block و حفظ پرونده | recall صرفاً با rank، clear invoice link | sales/finance actor مربوط؛ protected state واضح، نه الزاماً برگشت |
| Unresolved Case linkage؛ source owner | دیدن aliases و incomplete؛ توقف action وابسته | merge بر اساس phone / invoice relink | actor فنی/MIS مجاز از مسیر موجود؛ proof قابل ردیابی؛ مالک escalation جدید deferred |
| Stale/incomplete metric؛ report owner | دیدن coverage/time/filter؛ drill به records معتبر | تبدیل missing به صفر یا قضاوت عملکرد | report مسئول؛ coverage و freshness اعلام‌شده |
| F04 invoice mismatch؛ report/invoice owners | مقایسه grain/cohort و نمایش مغایرت | اصلاح مبلغ/status برای هماهنگی ظاهری | owner داده/گزارش موجود؛ reconciliation روی cohort همسان |
| Customer history failure؛ history service | retry read و استفاده از context موجود با incomplete label | تولید event یا ادعای «هیچ رفتار ندارد» | service owner؛ load معتبر یا failure صریح؛ F05 امروز در بخش 10 |
| Archive review؛ Manager | بررسی reason/age/owner/payment و مسیر actor موجود | delete یا revive بدون policy | sales/payment actor فعلی؛ resolved event واجد شرط یا archived باقی‌مانده |
| Extra-number pending؛ assigned Manager | بررسی row و approve/reject تحت guard | bypass scope، ساخت Case خارج provenance | Seller پس از approved؛ rejected با reason یا pending با failure روشن |
| HR pending؛ current reviewer | بررسی request assigned، approve/reject مرحله خود | final apply، credential edit یا عبور از chain | next reviewer سپس HR؛ transition/history یا failed apply مشخص |
| Finance outcome نیازمند اقدام فروش؛ Sales responsible actor | دیدن reason، stage و assist مجاز | Finance approval/refund/posting | Seller/actor فروش مجاز؛ resubmission معتبر یا حالت مالی نهایی |

مصرف‌شدن یا archived باقی‌ماندن می‌تواند نتیجه صحیح باشد؛ resolved console نباید به معنی «همه چیز آزاد/قابل تخصیص شد» باشد. زمان‌بندی escalation، ACK عمومی و تغییر assignment خودکار **DEFERRED** است.

## 7. Distribution / Return

### Distribution contract

**Current:** موجودی متعلق به Manager و recipient list سه سطح در L؛ helpers scoped recipients و active descendants در C (`23262/23374`). اجرای تخصیص امروز انجام نشده. سه نفر نمونه اثبات نمی‌کند هر descendant همیشه writable است.

**Target:** هر عملیات، actor، current custody، CaseRef/SourceRef و intended recipient را دارد. recipient باید در scope مجاز همان action، فعال و برای destination مجاز باشد. balance قابل تخصیص از predicate eligibility جدا از «تمام پرونده‌های قابل مشاهده» است. مصرف مالی، lifecycle یا unresolved identity می‌تواند action را ببندد.

| جزء | قرارداد هدف |
|---|---|
| Direct | تحویل به سطح مستقیم مجاز در hierarchy؛ historic owner/credit تغییر ضمنی نمی‌کند |
| Skip-level | مسیر فنی Senior/Supervisor/Seller موجود: **KEEP + CONDITIONAL**؛ تا policy مصوب scope واقعی و guards فعلی حفظ؛ دلیل استثنا و traceability هدف‌اند، نه اختراع approval chain |
| Preview | تعداد/شناسه‌های واجد شرایط، محدودیت source، گیرنده و block reasons؛ نتیجه یک snapshot است |
| Apply | آینده: recheck scope، active status، row custody، state و financial dependency در لحظه apply؛ preview مجوز قطعی نیست |
| History | actor/from/to، time، case/source، نتیجه و دلیل؛ سابقه تخصیص و attribution حفظ |
| Partial result | requested / eligible / applied / skipped / failed / unknown به تفکیک item؛ success کلی برای اجرای جزئی ممنوع |

### Return — Gate 0 F01 binding

کد ordinary return (`handle_distribution_return_items:20731/20758`) از انتقال تاریخی actor و custody جاری کمک می‌گیرد؛ کنترل invoice در آن مسیر کامل اثبات نشده. lock قدیمی نیز `invoice.lead_id` را بررسی می‌کند، درحالی‌که V4 می‌تواند `lead_id=0` و `follow_invoice_id` داشته باشد. بنابراین متن UI «موارد امن» تضمین امنیت مالی نیست. **CODE VERIFIED gap؛ unsafe execution NOT VERIFIED**.

Return هدف طبق G§6 و INV-006…010:

1. اختیار برگشت از transfer/custody/action policy اثبات می‌شود؛ Manager حق recall کل subtree را صرفاً با rank ندارد.
2. همه روابط مالی معتبر legacy/V4/Seller flow/financial event/Dot با lineage بررسی شوند؛ phone فقط discovery است.
3. financial consumption/fulfillment فعال ordinary return را می‌بندد؛ missing یا contradictory linkage **fail closed**.
4. cancelled/rejected به‌تنهایی release نمی‌دهد؛ OPD-03 تصمیم reset/release را باز نگه می‌دارد.
5. eligibility در apply دوباره و با حفاظت race بررسی شود؛ skip/error/unknown شفاف و history دست‌نخورده.

این الزام target است، **اصلاح اجرا نشده**. F01 یک dependency برای rollout write، نه مانع طراحی ماژول خواندنی است.

## 8. Performance & KPI Contract

Drill: **Senior → Supervisor → Seller → Case / Invoice**، در scope مجاز؛ levelها grain metric را عوض نمی‌کنند. hierarchy جاری برای visibility با credit/attribution تاریخی جداست. User ID یک identity است، KPI نیست. Ranking، target، conversion rate، efficiency score و SLA تعریف نمی‌شوند.

تمام KPIهای FINAL نیاز دارند: `name / grain / source / cohort / time_basis / scope / formula-predicate / freshness`. جدول زیر قرارداد تعریف را Freeze می‌کند؛ cohort تجاری باز و اجرای shared resolver هنوز **NOT FINAL** است. هر KPI instantiated باید metadata واقعی خود را همراه داشته باشد.

| Name / grain | Source و formula/predicate هدف | Cohort / time_basis / scope / freshness |
|---|---|---|
| Invoice Created / invoice | منبع معتبر invoice؛ تعداد distinct invoice IDهای ایجادشده، نه stage یا Case | invoice creation cohort؛ `created_at`؛ permitted invoice scope؛ query time + completeness؛ window مشخص الزامی |
| Open Pre-invoices / invoice | وضعیت جاری پیش‌فاکتور از dictionary G/S، شامل `pre_invoice` در دامنه مناسب؛ approved receipt به‌تنهایی completed نیست | current eligible invoice cohort؛ Current Snapshot؛ same scope؛ as-of timestamp؛ mapping namespaces صریح |
| Pre-invoices Issued / issuance event | صدور تاریخی معتبر پیش‌فاکتور، حتی اگر بعداً paid شود؛ unique issuance key نه count مراحل | Event Range با issuance time؛ attribution همان event؛ coverage history؛ absent issuance record ≠ 0 |
| Sales Completed / invoice | وصول معتبر کل invoice با amount conservation و tolerance مصوب G؛ receipt_uploaded/مرحله approved به‌تنهایی کافی نیست | completion event cohort؛ completion time؛ scope/credit policy اعلام‌شده؛ OPD-06 و OPD-07 برای جزئیات باز |
| Total Cases / logical Case | distinct Case با aliases اثبات‌شده و union منابع معتبر؛ unresolvedها incomplete، نه merge phone | source coverage صریح؛ current workload snapshot یا event cohort جدا؛ Case scope؛ freshness هر source |
| Legacy Leads Only / legacy lead | فقط distinct `sn_leads.id`؛ آن را «کل پرونده» ننامید | legacy-only cohort؛ creation/assignment time مشخص؛ legacy scope؛ last query و coverage label |

**F02:** missing `pre_invoice` در predicate نقش‌های قبلاً Auditشده CODE VERIFIED است؛ آن predicate را بدون شاهد به Manager نسبت ندهید. قرارداد پیش‌فاکتور مشترک لازم است، اما Open و Issued یکی نیستند.

**F03:** legacy-only metrics پوشش کل CRM ندارند. گزارش Manager امروز V4+legacy می‌آورد؛ این به معنی completeness کل sources یا alias dedup قطعی نیست. نتیجه bounded ۶۰–۱۲۰ ردیفی را total کل قلمرو ننامید. Current workload، transfer event و Invoice Created cohortهای متفاوت‌اند.

**F04:** L مجدداً invoice metric0 و invoice tab1 را نشان داد. علت **NOT VERIFIED**؛ cache/build/predicate/cohort فرضیه‌اند. ابتدا scope، filters، grain، time و source را همسان کنید؛ تا reconciliation عدد را trusted final KPI اعلام نکنید. Label Supervisor/Seller نیز جداگانه نیازمند validation است.

Missing/unavailable/partially loaded هیچ‌گاه zero مشروع نیست. Currency، row count، amount و rate با grain مشخص جدا هستند؛ commercial rates و historical attribution policy تحت OPD-07 deferred‌اند.

## 9. Invoices

**MUST SEE وقتی scoped:** invoice ID/state، Seller و hierarchy context، Case/source link، payment stage، Finance review outcome، total/paid/remaining با unit، next actor/reason و freshness/completeness. چند invoice یک customer یا Case را یک invoice نکنید؛ amount/stage/history مستقل حفظ شود.

| Action | کلاس | شاهد و تصمیم |
|---|---|---|
| List / filters / scoped detail | READ | L+A؛ KEEP، scope/field guard الزامی |
| Financial status/history | READ | control امروز دیده شد؛ history data امروز اجرا/تأیید نشد؛ KEEP قرارداد stage/payment/review مستقل |
| Copy payment link | ASSIST | L control؛ copy خودش تغییر مالی نیست؛ لینک حساس فقط همان invoice مجاز |
| Resend payment link | ASSIST + اثر WRITE/ارسال | L control؛ authorization/send outcome NOT VERIFIED؛ هر send permission/state/recheck مستقل، در این مرحله اجرا نشد |
| Register/upload receipt | WRITE / NEEDS VALIDATION | در Manager sample مشاهده نشد؛ C guard داخلی `29392` Manager scoped را می‌پذیرد؛ permission target خودکار از view یا Senior نتیجه نمی‌شود؛ OPD-10 |
| Edit prepayment/stage payment | WRITE / NEEDS VALIDATION | غیاب control در sample، guard داخلی به‌تنهایی تمام field/state authority را ثابت نمی‌کند؛ activation مشروط |
| Finance approve/reject | RESTRICTED | مسئول Finance؛ rank Manager و invoice visibility مجوز نیست |
| Refund / financial posting / commission rules | RESTRICTED | role mission و G حساس؛ بخشی از Manager target نیست |
| Cancel یا relink invoice | NEEDS VALIDATION / RESTRICTED بدون proof | action از نمونه و scope read استنتاج نشود؛ تغییر روابط مالی ممنوع |

Status باید invoice lifecycle، stage progress و Finance review را جدا نمایش دهد؛ یک badge «تأیید» فروش تکمیل‌شده نمی‌سازد. underlying status IDs و relationships حفظ می‌شوند. صف rejected/needs-action متعلق به sales actor مربوط است، نه مجوز Manager برای overturn Finance.

## 10. Customer Behavior

**نیاز محصول:** Manager در بررسی invoice/exception بداند لینک باز شده، محصول مشاهده شده، روش پرداخت انتخاب شده یا مرحله‌ای نیازمند اقدام actor است؛ فقط history معتبر در scope invoice/customer مربوط. رفتار مشاهده‌شده به معنی قصد خرید، contact attempt یا payment success نیست.

**F05 جدید:** placeholder دائمی Audit قدیمی امروز بازتولید نشد؛ list، search/pager و details read کار کردند. loading element بعد از data همچنان visible بود. list0 events و دو issuance-related detail entries دیده شد؛ این‌ها ممکن است populations متفاوت باشند، پس mismatch شمارنده **NEEDS VALIDATION** است، نه defect قطعی. اصل bug binder قدیمی را در target architecture تثبیت نکنید؛ root cause موفق/رفع‌شده هم ادعا نمی‌شود.

**Target decision:** MOVE customer behavior/history به **Shared Customer Profile** contextual؛ invoice/case/source/time را حفظ کنید. Manager-specific entry می‌تواند filtered view همین context باشد، نه customer database موازی. Shared profile business logic را از prototype نگیرد. نمایش event time/source و distinction system issuance versus customer action لازم است؛ اضافه‌کردن tracking یا event جدید deferred است.

| State | معنی صحیح |
|---|---|
| Loading | درخواست خواندن هنوز در جریان؛ اگر data آمده residual loading را موفقیت کامل ننامید |
| Partial | بخشی از event domains/pages حاضر؛ coverage باقی‌مانده مشخص |
| Error | history load failed؛ «بدون فعالیت» نیست |
| No Result | فیلتر معتبر match ندارد |
| Authorized Empty | scope/query کامل و مجاز، هیچ رویداد از population تعریف‌شده ندارد |
| Stale / Incomplete | cached/old events یا missing source؛ time/coverage آشکار |

هیچ event fabrication، customer scoring، merge با phone یا read-all-customer privilege پیشنهاد نمی‌شود.

## 11. Archive Explorer

Manager archive responsibility واقعی است؛ **MERGE presentation, KEEP distinct business semantics**. آرشیو حذف نیست و سبب آزادشدن financial dependency نمی‌شود. Code: [Seller Flow](../source/sales-network-v4/includes/class-sn-seller-flow.php) `render_manager_archives:980`؛ [Dot Flow](../source/sales-network-v4/includes/class-sn-dot-flow.php) constants `23/24` و payment archive renderer `3395`.

| Reason/type | قاعده موجود حفظ‌شدنی | Review / revival contract |
|---|---|---|
| No-answer | حداقل سه تلاش و سه روز عدم فعالیت پس از تلاش سوم؛ A/C و متن L | attempts/last activity/source/Seller/archive time؛ manual revive اختیار ثابت‌شده ندارد |
| Assessment unpaid | active payment link که طی سه روز تکمیل نشده؛ C/L؛ امروز یک row | total/paid/remaining/unit، due/archive، Case/invoice و actor؛ qualifying payment transition می‌تواند resolve کند، runtime اجرا نشده |
| Subscription unpaid | پنج روز طبق anchor/predicate همان flow | قواعد payment و نوع حفظ؛ با assessment یک deadline نشود |
| Product unpaid | پنج روز با predicate همان نوع | product identity/payment dependency مستقل |
| Product* unpaid | پنج روز در flow مخصوص؛ نوع ذخیره‌شده حفظ | با product معمولی یکی نشود؛ type-specific resolution محفوظ |

Explorer dimensions هدف: reason/type، age از archive time، source، Seller/team، financial dependency و revival eligibility **اگر قابل اثبات**. age filter threshold جدید workflow ایجاد نمی‌کند. کارت‌های 0/1 امروز snapshot نمونه‌اند.

**Visibility:** payment archives در C با `sales_manager_user_id` scoped و `resolved_at IS NULL`؛ renderer حداکثر 200 row دارد، absence beyond bound یا source unavailable empty قطعی نیست. مالک نمایش‌داده‌شده فعلی در renderer priority converter→supervisor→seller است؛ آن را automatically Current Custody یا Credit Owner ننامید. no-answer source و scope مسیر خودش را حفظ می‌کند.

**Actions:** READ / REVIEW = supported context؛ DELETE = NOT ALLOWED در target Manager؛ manual REVIVE/REASSIGN/BULK RESOLVE = **NEEDS VALIDATION / DEFERRED**. پرداخت دیرهنگام طبق handler واجد شرط resolution دارد؛ approve receipt، پرداخت جزئی یا یک click Manager را revival عمومی تلقی نکنید. هیچ archive mutation اجرا نشده.

## 12. Extra-number Requests

این مسیر «درخواست Seller برای افزودن شماره» است، نه ویرایش یک شماره canonical موجود. وجود queue امروز BOTH؛ review transitions **CODE VERIFIED / NOT LIVE VERIFIED**.

Target record: request ID، source/requester، assigned Manager، normalized phone، name/note، duplicate check نتیجه با scope آن، status، reviewer/time، decision/reason، resulting lead/CaseRef و audit history. phone مستقل از Case identity و source tag است.

| مرحله فعلی | رفتار کد / قرارداد |
|---|---|
| Submit | login+nonce، Seller role/position یا Admin، per-user extra-number access، mobile validation؛ name/note optional |
| Duplicate check | همان Seller + normalized phone در requestهای pending/approved؛ cross-Seller، سایر sources و canonical duplicate proof را پوشش نمی‌دهد |
| Routing | Manager بالادستی در request ذخیره می‌شود؛ اگر پیدا نشود request ثبت و مسیر Admin در پیام مطرح است؛ Manager آن را خودکار claim نکند |
| Pending review | Manager position/یا Admin، row `manager_id=actor`، status pending، transaction و row lock؛ visibility subtree به‌تنهایی کافی نیست |
| Reject | pending→rejected؛ rejected_by/time/reject_reason، activity log؛ reason از POST گرفته می‌شود ولی nonempty requirement در این handler نیست |
| Approve | ساخت **legacy lead** با seller_id، supervisor_id=0، status assigned، source_tag `approved_extra_number_request` و extra_number_request_id؛ سپس request→approved با lead_id، actor/time و log |
| Failure | اگر ساخت lead/result ثبت نشود، rollback و request pending طبق مسیر کد؛ runtime atomicity/side-effect کامل بدون QA تضمین نمی‌شود |

Code locators: `ajax_seller_request_extra_number:37531`، `ajax_manager_review_extra_number:37594`، `sn_render_manager_extra_number_requests:37648` در class-sn-plugin.php. Queue فقط آخرین 100 row را render می‌کند؛ full history completeness اثبات نشده و UI فعلی reason/result/history را کامل ارائه نمی‌کند.

**KEEP** approve/reject scoped موجود؛ **ADAPT** record visibility برای تصمیم traceable؛ **ADD target requirement** resulting Case/source و history قابل resolve، بدون relink یا rewrite lead قدیمی. Case mapping برابر با phone یا تبدیل اجباری lead legacy به V4 نیست. Approval bypass قرارداد G نمی‌کند؛ mapping ناقص با UNKNOWN/INCOMPLETE آشکار شود.

**NEEDS VALIDATION:** Seller inactive/جابجا‌شده پس از request، requester بدون Manager، stale manager assignment، cross-source duplication، uniqueness/race submit و population تاریخچه. manager_id snapshot در guard از hierarchy recheck جدید متمایز است؛ policy جدید را silently enforce نکنید. الزام reason تصمیم در target برای traceability مفید است، اما اجباری‌کردن field موجود **FUTURE IMPLEMENTATION — REQUIRES QA / ROLLBACK**.

## 13. HR Requests

Current forms L؛ chain/guards در C (`10284/10338/10397/10439`). **Submit/review/apply CODE VERIFIED؛ runtime NOT LIVE VERIFIED**. HR Role-specific runtime در Audit همچنان NOT LIVE VERIFIED است؛ Admin جای آن نیست.

| جزء | قرارداد Manager |
|---|---|
| Subject scope | Seller مجاز در subtree actor هنگام submit؛ درخواست out-of-scope فقط با رتبه Manager مجاز نمی‌شود |
| Type | `transfer_seller` یا `terminate_seller`؛ target Supervisor فقط transfer، reason و profile data لازم |
| Destination | فرم فعلی Sup#5؛ code candidate list ممکن است all-active Supervisors باشد؛ دیدن option مجوز انتقال خارج scope را ثابت نمی‌کند |
| Current reviewer | request نشان دهد دقیقاً نوبت کدام actor است؛ Manager فقط assigned step |
| Approval path | pending_review → next parent reviewer یا pending_hr؛ intermediate reject→rejected با history |
| Final apply | HR authority پس از chain؛ approved/applied_at در success؛ failed در apply failure؛ Manager step این نتیجه نیست |
| History | requester/subject، from/to، reasons، reviewer transitions، final outcome و effect time؛ history hierarchy را overwrite نکند |

Allowed: submit scoped؛ review pending request وقتی current reviewer خود Manager است. Prohibited: تغییر credential/role/capability، direct HR final apply، فرض انتقال موفق از approval intermediate. Target state باید approved step، pending HR و applied/failed را جدا کند؛ internal statuses S حفظ شوند.

**OPD-05 DEFERRED:** پرونده باز، invoice، active responsibility، future/current custody، entitlement و termination handover. Code inactive کردن profile/meta یا تغییر hierarchy proof کامل business handover نیست. Historical attribution طبق G§12 دست‌نخورده؛ هر request changed before review conflict است و نیازمند fresh read، نه overwrite.

## 14. Reports / Wallet

**Reports:** Shared Reports با scope مجاز Manager، hierarchy drilldown، source coverage، grain، filters/cohort، time basis، freshness و field security. Inventory مشترک C/A شامل invoices register، customer profiles، customer invoice details، MIS assignments، pre-invoices، sales، online sales، card sales، finance pending/rejected است؛ داشتن report financial به معنی Finance action نیست. فقط shell امروز دوباره مشاهده شد؛ هیچ report/export اجرا نشد.

| حالت زمانی | معنی | محدودیت |
|---|---|---|
| Current Snapshot | وضعیت جاری در query/as-of مشخص | historical credit را hierarchy جاری rewrite نکند |
| Event Range | رویداد در بازه تعریف‌شده | event time مستقل از creation/assignment؛ snapshot today را event historical ننامید |
| Historical As-Of | وضعیت قابل بازسازی در یک لحظه گذشته | فقط با evidence/event coverage کافی؛ current fields gathered today proof As-Of نیست، F09 |

**Export:** permission و allowed fields مستقل از screen read؛ همان cohort/time/source/metric contract، output bounds و incompleteness صریح. CSV نباید bypass row/field guard باشد. Raw MIS/finance-sensitive columns role-sensitive باقی می‌مانند. حفظ سازگاری reportهای legacy با source label، نه silently تغییر total. saved preset امکانات اضافی ضروری این Freeze نیست.

**Wallet:** Own Account، شخصی و فقط ledger/entitlement متعلق به خود actor؛ نه team revenue، team commission یا organizational Finance. ورود اولیه DOM محتوای wallet داشت، اما tab عمداً باز نشد. F07/G§11 pure-read الزام target است؛ بعضی rendererهای قدیمی مسیر conditional auto-post دارند، وقوع آن امروز اثبات نشده. انتقال entry صرفاً presentation است؛ dependency extraction و QA لازم است تا business posting به مسیر read وابسته نماند. هیچ renderer یا engine تغییر نکرد.

## 15. Permissions / Bulk Actions

نام‌های زیر **conceptual product permissions** هستند؛ WP capability جدید نیستند و هیچ grant/change انجام نشده. Shared read guard: login/role entry، module authority، row scope و field visibility جدا بررسی شوند. Manager shell در C role/position/Admin و report capability را قبول می‌کند؛ ورود shell اثبات write همه moduleها نیست.

| Permission مفهومی | ارزیابی هدف / شاهد |
|---|---|
| `view_manager_operations` | KEEP scoped overview؛ A/L؛ exception aggregation target |
| `view_scoped_teams` | KEEP hierarchy context/drill؛ A/C، نه writable subtree |
| `view_scoped_cases` | KEEP source-aware؛ A/L؛ union completeness مشروط |
| `assign_scoped_cases` | KEEP current eligible flow؛ C/L controls؛ scope/state/recipient مستقل |
| `return_eligible_cases` | CONDITIONAL rollout تحت F01؛ current handler موجود، target protection کامل نیست |
| `view_manager_performance` | KEEP؛ metric NOT FINAL تا بخش 8 |
| `view_scoped_invoices` | KEEP؛ L/C؛ field restrictions |
| `assist_invoice_action` | action-by-action؛ observed resend/copy/history؛ write extras OPD-10 |
| `view_customer_history` | KEEP contextual scoped invoice/customer events؛ L/C |
| `review_extra_number_request` | KEEP row-owned pending review؛ C؛ runtime NOT VERIFIED |
| `submit_hr_request` | KEEP scoped Seller و supported type؛ C/L |
| `review_assigned_hr_request` | KEEP فقط current reviewer/state؛ C؛ HR final جدا |
| `view_archives` | KEEP manager-assigned reasons؛ C/L |
| `review_archive_case` | READ/operational review؛ revival write NEEDS VALIDATION |
| `view_reports` | KEEP scoped governed report؛ field security مستقل |
| `export_reports` | CONDITIONAL؛ explicit export permission+scope+fields؛ execution NOT VERIFIED |
| `view_own_wallet` | KEEP personal؛ G pure-read، F07 activation dependency |

Write/approval target guards؛ این جدول الزام محصول است، ادعای enforcement موجود نیست:

| Action | Scope | State guard | Row guard | Field restriction |
|---|---|---|---|---|
| Distribution | own custody + action-eligible recipient | allocatable، active recipient، nonconsumed | source/Case و current owner در apply ثابت | فقط destination/count/note مجاز؛ no lineage/credit rewrite |
| Return | own valid transfer/custody policy | G financial gate، unresolved blocked | all links + state/owner recheck، history proof | no invoice clearing، no protected history/reset خارج policy |
| Resend link | permitted invoice assist | current link/stage/action eligibility | همان invoice و current scope | no amount/payee/status edit؛ send effect مستقل |
| Receipt/prepayment assist | OPD-10 تا proof کامل | permitted stage/lifecycle/payment rules | scoped invoice + fresh state | فقط fields صریحاً allowed؛ no Finance review/refund/post |
| Extra-number decision | stored manager_id actor طبق C | pending، valid decision | locked current request؛ requester/hierarchy change policy deferred | decision/reason؛ source/result links سیستم، نه user relink |
| HR submit | permitted Seller | supported request type، duplicate pending policy | subject/profile/target validated | type/target/reason؛ no credential/role/compensation |
| HR step review | assigned current reviewer | pending_review برای non-HR | current reviewer و version/state تازه | step decision/reason؛ final apply reserved HR |
| Archive revival | **NEEDS VALIDATION** | proof reason-specific لازم | scoped archive+financial dependency | no generic unarchive/delete؛ activation deferred |
| Export | allowed report/cohort | report valid و scope current | selected rows همان query contract | whitelist role fields؛ no sensitive spill |

| Bulk operation | Classification | دلیل و محدودیت |
|---|---|---|
| Bulk Distribution | REQUIRED / SUPPORTED controls | count-per-selected-recipient flow موجود؛ runtime apply NOT VERIFIED؛ truthful per-item outcome و recheck هدف |
| Bulk Return | SUPPORTED code / CONDITIONAL rollout | count/selection در C؛ F01 حفاظت target الزام پیش از activation آینده |
| Bulk Archive Review | NEEDS VALIDATION | read filtering supported؛ multi-row mutation/revival اثبات نشده؛ approve-all اختراع نشود |
| Bulk Extra-number Review | NEEDS VALIDATION | handler تک request است؛ queue empty دلیل bulk authority نیست |
| Bulk HR Review | NOT ALLOWED در این frozen target | request/reviewer/type/outcome مستقل؛ batch approval شاهد ندارد؛ تنها review جداگانه supported |
| Bulk Export | CONDITIONAL | report export permission، field/time/scope parity و bounded output؛ execution NOT VERIFIED |

NOT ALLOWED در target به معنی تغییر capability فعلی نیست. همه bulkهای آینده: per-item applied/skipped/failed/unknown، count سازگار، retry فقط نتایج معلوم eligible؛ Outcome Unknown با blind retry یا «همه موفق» جایگزین نشود، طبق G INV-034.

## 16. Handoffs

| مسیر | قرارداد handoff / نتیجه |
|---|---|
| Upstream / MIS → Manager | source/case و transfer trace؛ current custody انتقال معتبر، origin ثابت؛ import/repair در Manager نیست |
| Manager → Senior/Supervisor/Seller | actor/from/to/time/source/result؛ direct/skip-level policy بخش 7؛ original و credit owner ضمنی تغییر نکنند |
| Seller → Manager extra-number | pending request با assigned Manager؛ approve→resulting legacy lead برای Seller با provenance؛ reject→reason/history |
| Archive/exception → Manager | operational review reason/owner/dependency؛ مشاهده Manager مالک مالی یا Seller operator نمی‌سازد |
| Finance outcome → Sales responsible actor | review reason و stage context؛ sales assist/resubmission مجاز؛ Manager فقط در scope/action مجاز کمک می‌کند |
| HR workflow → Manager reviewer → next reviewer/HR | request/current step/history؛ intermediate approval از final apply/failed جدا |
| Manager → Deputy | فقط مسیر/اختیار policy موجود و اثبات‌شده؛ console جدید escalation/ACK/SLA **DEFERRED**؛ نقش مقصد احتمالی مجوز command نمی‌سازد |

چهار مفهوم در تمام این مسیرها مستقل‌اند: **Current Custody** = مسئول نگهداری/کار جاری؛ **Original Owner** = مالک در origin؛ **Next Actor** = مسئول گام بعدی؛ **Credit Owner** = entitlement/attribution تاریخی تحت policy. current hierarchy/row label proof هیچ‌یک از سه دیگری نیست. References G§4–5 و INV-004/005/029؛ ownership rewrite، انتقال خودکار credit یا universal ACK تعریف نشده.

## 17. Data Visibility / System States

| سطح | داده |
|---|---|
| MUST SEE | scope، case/source identity و unresolved flag؛ current custody و recipient eligibility؛ Seller/team، invoice/payment/review states، unit/remaining، next actor/reason؛ request reviewer/result/history؛ archive reason/age/dependency؛ time/coverage/freshness |
| NICE TO HAVE | فیلتر reason/source/age/team، aggregate با قرارداد کامل، event context برای investigation، actor تاریخی و linked evidence مجاز؛ هیچ commercial score جدید |
| SHOULD HIDE / RESTRICT | Finance approval/posting/refund controls؛ commission rules/دیگران؛ HR credentials/compensation؛ raw MIS repair؛ out-of-scope customers؛ bank data غیرضروری؛ technical statuses برای کاربران عادی؛ User ID به‌عنوان KPI |

technical statuses در diagnostic view مجاز می‌توانند برای traceability همراه label تجاری دیده شوند؛ stored values حذف/rename نمی‌شوند. receipt/bank field فقط با نیاز همان action و field permission؛ hierarchy visibility مجوز افشای اطلاعات کارکنان نیست.

| State | معنی / مثال Manager |
|---|---|
| Loading | read هنوز pending؛ loader پس از data موفق نباید بدون explanation باقی بماند، residual F05 |
| Empty | authorized complete scope بی‌پرونده؛ owned0 تنها همان pool را توصیف می‌کند |
| No Result | فیلتر درست اما match ندارد؛ متفاوت از empty کل قلمرو |
| Unauthorized | module/action/row/field denied؛ hidden count یا empty mask جای denial نیست |
| Stale | query/cache قدیمی؛ freshness و بازه معلوم؛ refresh خواندنی |
| Incomplete | یک تیم/source/page fail کرده یا archive coverage bounded؛ no valid zero/total |
| Conflict | owner/state تغییر کرده؛ consumed return، duplicate request یا HR reviewer changed؛ reread قبل اقدام |
| Partial Success | برخی itemها نتیجه دارند؛ team batch distribution outcomes جدا و totals قابل reconcile |
| Retryable Failure | read/server failure یا known failed item با safe retry؛ موفقیت حدس زده نشود |
| Outcome Unknown | درخواست write outcome قطعی ندارد؛ ابتدا reconcile، نه blind resend/reapply |
| Offline | شبکه قطع؛ cached context با stale label؛ تصمیم/تأیید/ارسال جدید به‌عنوان completed نمایش داده نشود |

Metric discrepancy F04 حالت Conflict/Incomplete تحلیلی است، نه صفر صحیح؛ F05 history details می‌تواند موجود باشد ولی loading یا event population نامشخص incomplete باقی بماند. هر state به دامنه خودش تعلق دارد؛ یک تیم failed تمام تیم‌های موفق را failed یا complete نمی‌کند. Operational workflow statuses، request status و transport result namespaces مستقل‌اند.

## 18. Product Gaps & Deferred Decisions

| ID / Priority | وضعیت / Evidence / Confidence | محدودیت و شرط رفع |
|---|---|---|
| M-G01 / P0 — F01 return protection | PRODUCT GAP — DO NOT IMPLEMENT؛ C/A/G HIGH | financial links union، fail-closed و race QA قبل rollout write؛ unsafe runtime loss ادعا نشده |
| M-G02 / P1 — F04 parity | PRODUCT GAP — DO NOT IMPLEMENT؛ L HIGH discrepancy، cause UNKNOWN | same cohort/grain/scope/time reconciliation؛ count را silently اصلاح نکنید |
| M-G03 / P1 — F05 state/history | PRODUCT GAP — DO NOT IMPLEMENT؛ L HIGH changed behavior | original placeholder no longer reproduces؛ residual loading verified؛ count populations و root cause NEEDS VALIDATION |
| M-G04 / P1 — Case/source and extra-number edge cases | DEFERRED / CONDITIONAL؛ C HIGH structure، policy UNKNOWN | legacy result provenance حفظ؛ inactive/reparented Seller، duplicate/race، missing Manager، history completeness؛ OPD-01 وابسته |
| M-G05 / P1 — shell/action authority alignment | PRODUCT GAP — DO NOT IMPLEMENT؛ C HIGH guard presence | view entry با write متفاوت؛ negative/action-field tests آینده؛ privilege escalation فعلی قطعی نیست |
| M-G06 / P1 — pure-read wallet | DEFERRED / CONDITIONAL؛ A/C/G | F07 dependency isolation قبل جابه‌جایی فنی؛ posting امروز NOT VERIFIED |
| M-G07 / P1 — HR handover | DEFERRED؛ OPD-05 | profile update ≠ handover؛ open work/finance/credit و failed apply QA |
| M-G08 / P2 — skip-level policy | DEFERRED / CONDITIONAL؛ L+C HIGH path | KEEP مسیر موجود؛ scope/reason/approval policy بدون assumption؛ no expanded descendant writes |
| M-G09 / P2 — manual archive revival | NEEDS VALIDATION / DEFERRED؛ C/L inventory HIGH | reason-specific authority/financial guard/revival semantics؛ no generic restore یا delete |
| M-G10 / P2 — bulk review | DEFERRED؛ C single request / L empty | archive/extra-number bulk semantics و policy بررسی نشده؛ HR batch در target فعلی مجاز نیست |
| M-G11 / P2 — escalation ownership | DEFERRED؛ OPD-02 | actorهای workflow شناخته‌شده حفظ؛ routing/ACK/SLA جدید اختراع نشود |
| M-G12 / P1 — KPI cohorts / attribution / units | DEFERRED؛ OPD-06/07؛ F02/03 | Final metric metadata، history coverage، unit/tolerance؛ rate/ranking/target NOT FINAL |
| M-G13 / P2 — field/context completeness | PRODUCT GAP — DO NOT IMPLEMENT؛ A/L | invoice/source/next actor و request outcome/history قرارداد هدف؛ join label و archive owner semantics validated قبل trusted display |

OPD-03 cancelled/rejected release؛ OPD-08 maker/checker؛ OPD-09 entitlement engine routing و OPD-10 expanded invoice delegation نیز binding/deferred می‌مانند. هیچ‌کدام با mock data یا ظاهر تازه حل‌شده اعلام نشود. P3 features مانند saved views فقط پس از قراردادهای P0/P1 و نیاز اثبات‌شده؛ feature جدید الزامی این مرحله نیست.

**Design blocker:** هیچ blocker ساختاری باقی نیست؛ Claude می‌تواند hierarchy/context و guarded current flows را طراحی کند، با conditional treatment مسائل باز. **Implementation/activation dependencies** جدول بالا باقی‌اند و Freeze آنها را waive نمی‌کند.

## 19. Compatibility / Safe Adoption

تمام تصمیم‌ها **KEEP + ADAPT**. Preserve existing IDs، routes/internal tab IDs، aliases/relationships، origin و legacy sources، history، current workflows و permissions تا rollout صریح. MERGE یعنی presentation/facts contract مشترک با adapter سازگار، نه ادغام destructive records یا تغییر قواعد.

| تصمیم هدف | bridge سازگار | شرط آینده |
|---|---|---|
| Overview و IA جدید | route/tab mapping محفوظ؛ User ID به identity context منتقل | totals/source parity و no authority expansion |
| Performance/shared reports | adapters با source/grain/time metadata؛ legacy-only labels honest | bounded versus full counts، dedup proof، F04 reconcile |
| Shared Customer Profile | context به scoped existing invoice/events؛ history باقی | field/row security، F05 state/event population |
| Archive Explorer | reason-specific queries/transitions؛ 3/5-day rules intact | snapshot freshness، resolution/payment/owner validation؛ no delete |
| Request queues | IDs، pending/review chain، source_tag/result links محفوظ | race/idempotency، inactive/reparented actors، truthful failures |
| Personal wallet placement | فقط entry presentation | pure-read/engine dependency checks، F07 |
| Write guards و return safety | approved future rollout؛ changes کوچک قابل rollback | cross-source financial links، preview/apply race، partial/unknown outcomes |

No destructive migration، status rename، ID rewrite، invoice relink، historical attribution rewrite، silent new capability یا backfill حدسی. هر تغییر پرریسک: **FUTURE IMPLEMENTATION — REQUIRES QA / ROLLBACK**. Rollback نباید approval/payment/history را پاک کند؛ QA باید اثر دقیق مسیرهای existing را پوشش دهد، نه فقط ظاهر.

Minimum future acceptance: read scope/field و negative permissions؛ multi-level recipient scope؛ own-transfer return با هر financial-link domain؛ concurrent invoice/return؛ request stale/duplicate/routing؛ HR intermediate versus final/failed apply؛ archive late-payment dependency؛ KPI same-cohort parity؛ wallet pure-read؛ report/export field parity؛ partial/offline/unknown outcomes. این‌ها شرایط rollout آینده‌اند؛ هیچ test write یا implementation در این مرحله اجرا نشده.

## 20. Claude Design Handoff

### WHAT CLAUDE MAY ASSUME

Manager mission، هفت ماژول اصلی بخش 5، Own Account ثانویه و Shared Customer Profile contextual Freeze هستند. مسیرهای موجود و data entities حفظ می‌شوند. Shared facts/state/action contracts Gate 0 معتبرند. Frozen Seller/Supervisor/Senior طراحی foundation مشترک‌اند، نه منبع business logic. Existing empty datasets قابلیت‌ها را حذف نمی‌کنند.

### WHAT CLAUDE MUST NOT INVENT

مجوز Finance/HR نهایی/MIS/Admin؛ writable همه descendants؛ generic revive/approve-all؛ SLA/escalation/ACK جدید؛ score/rate/target/ranking؛ phone canonical identity؛ cancelled/rejected release؛ approval مساوی paid/completed؛ source-free KPI؛ historical-as-of از current data؛ guard success یا repaired F04/F05 بدون evidence. هیچ unresolved action را با CTA قطعی یا data ساختگی مجاز جلوه ندهد.

### SHARED COMPONENTS TO REUSE

قرارداد role shell/identity، hierarchy scope/drill، scoped Case/source context، invoice stage/review/history، Shared Customer Profile، governed Reports، HR request step/history، personal account و common system states. reuse به معنی ظاهر مشترک با business boundaries مستقل است؛ lower-role action permissions کپی نشوند.

### MANAGER-SPECIFIC DESIGN NEEDS

Operations & Exceptions برای owner/next actor/resolution؛ توزیع به سه سطح با direct/conditional skip-level؛ Archive Explorer با reasons مستقل؛ Extra-number Review Queue با provenance/result؛ multi-team performance و reconciliation؛ HR step در context Manager. این‌ها WHAT هستند؛ layout، styling و interaction visual به Claude واگذار می‌شود.

### LIKELY NEW SHARED COMPONENTS

Operations Exception Console؛ reason-aware Archive Explorer؛ Request Review Queue؛ Extra-number Review Queue؛ Performance Explorer extension؛ Metric Reconciliation indicator. اشتراک بین roleها فقط در قرارداد context/state/history، نه لزوماً authority یا همه destinations. ضرورت دقیق reuse در design مرحله بعد بررسی شود؛ component یا prototype در این مرحله ساخته نشده.

### DEFERRED / CONDITIONAL ITEMS

F01 write activation، F04 metric trust، residual F05/event count semantics، skip-level policy، manual revival، bulk review، extra-number stale/duplicate edges، OPD-05 handover، OPD-06/07 units/cohorts، OPD-10 invoice assistance expansion و F07 wallet read dependencies. Conditional flows با حدود روشن طراحی شوند؛ هیچ permission، formula یا policy باز با تصمیم بصری نهایی نشود.

**SALES MANAGER PRODUCT SPEC FROZEN — READY FOR CLAUDE REDESIGN**

**NO CODE CHANGED.**
