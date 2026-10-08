# GATE 0 — PRODUCT DECISIONS / INVARIANT FREEZE

تاریخ: 2026-09-30 · وضعیت: **FROZEN TARGET / IMPLEMENTATION DEFERRED** · دامنه: WHAT / WHY، نه طراحی بصری یا پیاده‌سازی.

## 1. Executive Decision Summary

این سند قرارداد هدف محصول را بر پایه Audit پذیرفته‌شده Freeze می‌کند. Freeze به معنای رفتار پیاده‌سازی‌شده، موفقیت QA یا تغییر اختیار کاربران فعلی نیست. قاعده حاکم **NO BREAKING CHANGE BY DEFINITION / KEEP + ADAPT** است. هیچ تغییر کد، داده، قابلیت، وضعیت ذخیره‌شده یا رابطه مالی برای پذیرش Gate 0 لازم نیست.

مراجع و برچسب‌های شاهد:

- **A**: [CRM PRODUCT & ROLE ARCHITECTURE AUDIT](CRM-PRODUCT-ROLE-ARCHITECTURE-AUDIT.fa.md)، شامل F01–F10، ماتریس نقش‌ها و معماری هدف.
- **C**: [Code Evidence & Action Registry](CODE-EVIDENCE-AND-ACTION-REGISTRY.md)، ارجاع handler/guard و فایل‌های منبع.
- **S**: [Status Transition Catalog](STATUS-TRANSITIONS.fa.md).
- **D**: [Decisions / Visibility / Interactions](DECISIONS-VISIBILITY-INTERACTIONS.fa.md).
- **E**: [شواهد Audit](../evidence/) و [اعتبارسنجی قبلی](../evidence/audit-validation.json).

LIVE VERIFIED فقط مشاهده واقعی همان رفتار خواندنی است؛ CODE VERIFIED اثبات ساختار کد است؛ BOTH توافق آن دو درباره ادعای مشخص؛ INFERENCE قرارداد هدف است. HR و Finance همچنان **CODE VERIFIED / NOT LIVE VERIFIED** هستند. Target requirements این سند مبتنی بر شواهدند ولی اجرای موفق آنها ادعا نمی‌شود. اطمینان به تعریف هدف HIGH است؛ اطمینان به رفتار فعلی مطابق A، و برای تصمیم‌های تجاری باز تعیین‌نشده است. یکسانی شماره نسخه 2.0.123 با ZIP، اثبات یکسانی byteهای نصب‌شده نیست.

| پرسش Gate | پاسخ قطعی هدف | مرجع |
|---|---|---|
| پرونده چیست؟ | واحد مشخص کار تجاری با lineage قابل اثبات؛ شخص/شماره/فاکتور نیست | 2 |
| شناسه چیست؟ | CanonicalCaseId منطقی، پایدار، غیرقابل استفاده مجدد؛ aliases با IDs فعلی حفظ می‌شوند | 2–3 |
| مالک‌ها چیستند؟ | Custody، Original، Next Actor و Commission چهار مفهوم مستقل‌اند | 4 |
| مصرف‌شده چیست؟ | وابستگی تجاری محافظت‌شده، از جمله هر رابطه مالی فعال معتبر؛ مستقل از label تحویل | 6 |
| چه چیزی Return را می‌بندد؟ | رابطه مالی/fulfillment فعال، مصرف منبع، منع scope/policy یا ابهام ارتباط | 6 |
| ارتباط مالی چگونه ثابت است؟ | Stable IDs و lineage معتبر؛ phone فقط discovery | 6–7 |
| Completed چیست؟ | وصول معتبر کل فاکتور در tolerance مصوب؛ نه صرفاً receipt/approved stage | 7 |
| KPI چگونه تعریف می‌شود؟ | grain/source/cohort/time/scope/predicate/freshness مشترک | 9 |
| اختیار چگونه اعمال می‌شود؟ | چهار لایه؛ view هیچ write حساس را ضمنی نمی‌دهد | 10 |
| Read چگونه است؟ | بدون business mutation؛ job/cache فنی جدا | 11 |
| Handoff چیست؟ | رویداد قابل ردیابی با actor، case، responsibility و نتیجه؛ ACK عمومی تحمیل نمی‌شود | 5 |
| Attribution تاریخی چیست؟ | snapshot رویداد/حق مالی، مستقل از hierarchy جاری | 12–13 |

تصمیم‌های باز بخش 15 با مسیر امن و محدودیت روشن **Deferred** هستند. هیچ‌یک مانع تهیه Supervisor Product Spec با scope فعلی و actions مشروط نیست؛ این به معنی آزادشدن rollout یا تصمیم نهایی Finance نیست.

## 2. Canonical Case Contract

**Current Reality:** MIS row، V4 distribution item، legacy lead و Dot case شناسه و lifecycle مستقل دارند. در گزارش MIS، grain پرونده `sn_mis_data_rows.id` است؛ این شناسه به‌تنهایی ID سراسری CRM نیست. ایجاد V4 invoice می‌تواند بدون legacy lead انجام شود.

**Target Contract:** Case واحد کار تجاری ردیابی‌پذیر از منبع تا نتیجه و handoff است. یک مشتری می‌تواند چند Case داشته باشد؛ یک Case می‌تواند چند invoice/stage/event داشته باشد. Customer identity، Case identity و Invoice identity مستقل‌اند. ایجاد Dot case یا legacy lead از منبع، لزوماً ایجاد Case تجاری تازه نیست؛ اتصال یا جدایی فقط با lineage اثبات‌شده و نوع رابطه معلوم می‌شود.

`CaseRef = { source_kind, source_id, canonical_id }`؛ source_kind یکی از domainهای مشخص MIS_ROW / V4_DISTRIBUTION / LEGACY_LEAD / DOT_CASE در قرارداد adapter است، نه الزام نام DB. source_id همان ID بومی بدون بازنویسی است. canonical_id یک شناسه opaque و immutable منطقی برای گروه aliases اثبات‌شده است. encoding/محل نگهداری آن انتخاب Engineering آینده است، نه schema جدید اجباری. Root مبتنی بر source domain + ID موجود است؛ IDهای برابر در دو domain یک Case نیستند.

Phone هرگز canonical_id، join مالی قطعی یا معیار ادغام نیست. تغییر شماره نباید identity یا invoice link را تغییر دهد. همان شماره می‌تواند چند پرونده معتبر داشته باشد. چند alias فقط پس از proof مشترک به یک canonical_id resolve می‌شوند. duplicate label یا شباهت نام/شماره proof نیست. اگر proof ناقص/متناقض است: identity resolution = UNKNOWN، unresolved sources حفظ می‌شوند، aggregate به عنوان incomplete اعلام می‌شود و عملیات مالی/return وابسته fail closed است. Unknown را ID ساختگی، صفر یا یک پرونده ادغام‌شده نمایش ندهید.

**Compatibility Bridge:** read-through resolver روی منابع فعلی و explicit lineage؛ هیچ merge جدول، rewrite ID یا relink invoice. رابطه continuation در برابر related-but-distinct برای موارد فاقد proof: OPD-01. تمامی aliases اصلی قابل بازیابی می‌مانند. قرارداد logical identity باید با تغییر صفحه، hierarchy یا شماره ثابت بماند؛ format آینده نباید معنی IDهای قبلی را از بین ببرد.

## 3. SourceRef

| فیلد | الزام | تاریخی؟ | تضمین |
|---|---|---|---|
| source_kind | Required | بله | domain بدون ابهام |
| source_id | Required | بله | ID بومی پایدار و قابل resolve |
| origin_batch | Required برای منبع batch-based؛ otherwise Optional/Not Applicable | بله | batch واقعی ورود، نه batch توزیع جاری |
| origin_actor | Required برای رویداد ورود جدید؛ در سابقه فاقد داده UNKNOWN | بله | actor واقعی ورود؛ نه مالک امروز |
| origin_timestamp | Required برای ورود جدید؛ در سابقه فاقد داده UNKNOWN | بله | زمان ورود با timezone مشخص؛ نه updated_at |
| canonical_id | Required برای resolved CaseRef؛ unresolved صریح | بله | identity منطقی بخش 2 |

SourceRef تضمین provenance می‌دهد، نه مالکیت جاری، مجوز، صحت شماره یا وصول پول. Optional/UNKNOWN/NOT RECOVERABLE باید از یکدیگر و از null ناشی از error قابل تشخیص باشند. اطلاعات تاریخی مجهول را از مالک یا زمان فعلی backfill حدسی نکنید.

**Current:** provenance در batch/row/transfer/flow متفاوت است. **Target:** origin ثابت، current routing مستقل. **Bridge:** Legacy و V4 هم‌زمان خوانده می‌شوند؛ alias با proof به canonical root متصل می‌شود و هر SourceRef بومی باقی می‌ماند. resolver، provenance و دلیل resolution را گزارش می‌دهد. «V4 جایگزین Legacy» فرض نمی‌شود؛ Target physical source of truth در این Gate انتخاب نشده است.

## 4. Ownership

| مفهوم | معنی / قابلیت تغییر | چه کسی؟ | History / Permissions | Commission / Reporting |
|---|---|---|---|---|
| Current Custody / CurrentOwner | نگهدار فعلی مسئول صف/تحویل؛ با assignment/return مجاز تغییر می‌کند | actor دارای action permission و scope معتبر | هر انتقال ثبت؛ یکی از ورودی‌های row permission، نه اختیار کامل | commission از آن derive نمی‌شود؛ current workload بر اساس آن |
| OriginalOwner / OriginalSeller | اولین تخصیص معتبر در معنای مربوط؛ origin actor و اولین Seller یکسان نیستند | service ثبت اولین رویداد؛ اصلاح فقط رسمی با evidence | historical، تغییر عادی ممنوع؛ اختیار جاری نمی‌دهد | attribution منبع؛ حق commission خودکار نیست |
| CurrentActionOwner / NextActor | فرد/نقش موظف به action مشخص بعدی؛ ممکن است Finance یا reviewer باشد | workflow service یا actor مجاز همان انتقال | action، reason، due و مسئول مشخص؛ دسترسی محدود همان کار | next-action/backlog؛ نه credit owner |
| Credit / CommissionOwner | recipient حق مالی طبق snapshot مرحله/قاعده/سهم | engine/command مالی مجاز؛ correction رسمی | recipient historical ثابت؛ صلاحیت دیدن سایر پرونده‌ها نمی‌دهد | ledger و historical earnings؛ hierarchy جدید آن را بازنویسی نمی‌کند |

OriginalOwner اولیه منبع و OriginalSeller اولیه فروش را در payload تفکیک کنید؛ اگر یکی وجود ندارد UNKNOWN/Not Applicable، نه current owner. NextActor باید actor یا queue مسئول مشخص داشته باشد؛ نمایش queue بدون مسئول failure/escalation، ownership کافی نیست. Next action شامل action kind، subject reference، responsible actor/role، due_at در صورت policy، blocked reason و allowed command است؛ لیست actions از مجوز واقعی می‌آید.

**Current:** ستون‌ها و نقش‌های ضمنی متفاوت‌اند. **Target:** چهار مفهوم قابل مشاهده و بدون synonym. **Bridge:** adapter با provenance از فیلدها/رویدادهای موجود، عدم backfill اجباری؛ historical مالک مجهول UNKNOWN / NOT RECOVERABLE. تعویض مدیر یا Custody، entitlement قبلی را منتقل نمی‌کند. [A/C/D]

## 5. Handoff

**Current:** توزیع دارای transfer history، Finance reject دارای returned_to_seller، HR دارای approval_path و current reviewer است؛ state machine واحد عمومی وجود ندارد. **Target:** قرارداد رویداد مشترک، با lifecycle بومی هر domain. **Bridge:** adapter رویدادهای فعلی؛ هیچ تغییر enum یا workflow فعلی صرفاً برای یکسان‌شدن label.

| فیلد | قرارداد |
|---|---|
| correlation_id | شناسه عملیات/زنجیره؛ در retry حفظ می‌شود؛ historical absent = UNKNOWN |
| case_ref | Required برای handoff پرونده؛ HR با subject employee/request ref و case_ref = Not Applicable، نه Case ساختگی |
| from_actor / to_actor | actorهای واقعی یا queue با accountable owner؛ historical unknown صریح |
| from_role / to_role | snapshot نقش هنگام رویداد، نه role امروز |
| handoff_type | نوع بومی namespace-qualified: distribution / financial_correction / HR_review / conversion_assignment |
| reason_code / note | دلیل کددار در مسیر دارای taxonomy؛ note مکمل؛ دلیل اجباری reject/return/exception؛ legacy free text حفظ |
| created_at / accepted_at | created required برای جدید؛ accepted فقط evidence پذیرش واقعی، otherwise null/unknown |
| due_at | اگر SLA مصوب دارد required؛ otherwise unspecified، SLA اختراع نشود |
| state | state بومی + canonical meaning معتبر؛ نام مشابه state عمومی تولید نمی‌کند |
| failure_owner | مسئول رسیدگی شکست/عدم تحویل؛ بعد از خطا work item رها نشود |

Stateهای پشتیبانی‌شده: transfer موفق فعلی = delivered/committed، برگشت = returned، HR = pending_review / pending_hr / approved / rejected / failed، Finance correction = returned_to_seller / resent_after_return. `pending` فقط اگر انتظار واقعی ثبت است؛ `accepted` فقط با پذیرش مستند؛ `cancelled` فقط در lifecycle دارای cancel. فهرست پنج‌تایی pending/accepted/returned/failed/cancelled به‌عنوان enum عمومی **Freeze نمی‌شود**. Return به MIS مساوی Finance rejection نیست.

ACK صریح برای **تصمیم HR توسط reviewer/final HR** و **تصمیم Finance درباره stage** لازم است و با action مجاز، actor و زمان ثبت می‌شود؛ ACK به معنای تصمیم service است، نه دکمه عمومی «دریافت». توزیع عادی فعلی atomic committed transfer است؛ recipient acknowledgement جدید برای همه توزیع‌ها الزام نشده. الزام ACK جداگانه برای تحویل پرونده حساس، termination handover و SLA: OPD-02/OPD-05. تا تصمیم، commit را receipt انسانی یا accepted_at اختراعی ننامید.

## 6. Consumed / Return Eligibility

### تعریف هدف و F01

**Consumed** یک guard dependency است، نه status واحد یا مترادف sold. پرونده دارای financial link فعال معتبر، payment/stage فعال یا وصول‌شده، fulfillment باز، یا downstream materialization محافظت‌شده مصرف شده است. «بدون invoice» به تنهایی دلیل return نیست؛ source conversion، scope، contact/reset policy و handoff باز نیز باید بررسی شوند. Materialized legacy/Dot identity با reset منبع حذف نمی‌شود.

**Current Reality / BOTH HIGH:** F01: ایجاد invoice V4 مقدار `invoices.lead_id` را صفر می‌کند ولی follow_invoice_id و flow links می‌نویسد؛ MIS نمونه دارای invoice فعال را returnable نشان داد. اجرای return انجام نشد. guard فعلی virtual lead_id کافی نیست؛ ordinary sales return نیز نیاز به همان resolution دارد. batch-delete و return predicates فعلی یکسان نیستند. [A F01/C]

**Target:** `ActiveFinancialLinkResolver(case)` مفهوم محصول است، نه کد این مرحله. تمام links معتبر برای همان Case را جمع و reconcile می‌کند؛ نبود match در یک جدول proof نبود invoice نیست. لینک‌های فعلی:

| منبع لینک | قدرت شاهد / کاربرد |
|---|---|
| `sn_invoices.lead_id` به legacy lead واقعی | AUTHORITATIVE پس از تطبیق domain و وجود رکورد |
| `sn_distribution_items.follow_invoice_id` | AUTHORITATIVE V4؛ نیازمند resolve invoice واقعی |
| `sn_seller_flow_states.last_invoice_id` | لینک ID صریح؛ proof با context همان source؛ last فقط آخرین است و جای همه links نیست |
| `sn_seller_flow_events.invoice_id` | proof تاریخی ارتباط؛ invoice جاری resolve شود؛ historical event خود به تنهایی active-state نیست |
| Dot case↔invoice/payment linkage موجود | AUTHORITATIVE در context Dot طبق adapter و guard کد؛ phone join نیست |
| virtual-ID / legacy compatibility رابطه قدیمی | LEGACY FALLBACK فقط با validation قواعد encoding/domain و رابطه واقعی؛ عدم match این مسیر نفی links دیگر نیست |
| phone/name/time proximity | DISCOVERY ONLY؛ هیچ مجوز return، merge یا financial posting نمی‌دهد |

Link به invoice مفقود/متناقض = unresolved dependency؛ return ممنوع تا reconcile مجاز، نه «بدون فاکتور». Active شامل pre_invoice/draft مالی معتبر، pending/rejected قابل اصلاح، partial و completed دارای تعهد/سوابق محافظت‌شده است. پایان مالی خودکار پرونده را returnable نمی‌کند. در cancelled موارد release نیاز به policy رسمی دارد؛ OPD-03. Guard باید union links معتبر را ببیند، نه صرفاً آخرین invoice یا وضعیت delivery.

### جدول تصمیم Return / Reset

| Case State | Invoice State | Payment State | Fulfillment State | Return Allowed? | Why? |
|---|---|---|---|---|---|
| قابل تخصیص و scope معتبر | None، resolution کامل | None | None | مشروط YES | فقط اگر تمام قواعد nonfinancial/reset/handoff مجازند |
| فعال | draft / pre_invoice معتبر | بدون وصول | None | NO | تعهد مالی فعال؛ هنوز sold نیست ولی consumed مالی است |
| فعال | منتظر پرداخت | pending | None | NO | invoice/stage فعال |
| فعال | receipt_uploaded | receipt evidence | None | NO | بررسی/اصلاح لازم؛ receipt پول تأییدشده نیست |
| فعال | pending_financial_approval | pending review | None | NO | مسئولیت Finance باز |
| فعال | rejected | current stage rejected؛ قبلی ممکن است approved | None | NO | reject invoice را unlink یا release نمی‌کند |
| فعال | partial_paid | بخشی معتبر | هر حالت | NO | مبلغ تأییدشده و remaining dependency |
| کامل مالی | paid / approved + completion predicate | کامل معتبر | باز یا پایان‌یافته | NO برای ordinary reset | history و downstream obligations حفاظت می‌شوند |
| بسته | cancelled unpaid | paid≈0 | None | NO تا explicit release policy | cancellation خود release منبع نیست؛ OPD-03 Deferred |
| بسته مالی | cancelled after payment/refund | refund flag یا proof جدا | باز/بسته | NO تا reconcile/release مصوب | cancel/refund/fulfillment یکی نیستند؛ OPD-03/04 |
| legacy materialized | invoice linked by real lead_id | هر وضعیت فعال | هر حالت | NO | ارتباط legacy باید تشخیص داده شود |
| V4 delivered | follow_invoice_id معتبر؛ lead_id=0 | حتی بدون وصول | None | NO | حل مستقیم F01 |
| Dot-linked | invoice/payment case link معتبر | pending/partial/paid | هر حالت | NO | lineage مالی Dot محافظت می‌شود |
| هر حالت | link ambiguity/missing target | Unknown | Unknown | NO | fail closed با reason و مسئول reconcile |
| بدون مالی | None | None | فعال | NO | fulfillment dependency مستقل |

### Preview، Apply و race

Preview نتیجه versioned eligibility با case_ref، evaluated_at، scope، dependencies، allowed/reasons است؛ مجوز قطعی بعدی نیست. Apply باید actor/action/row guards و dependency resolution را در نقطه commit دوباره بررسی کند. Invoice creation و return برای همان Case نباید هر دو روی فرض unused متناقض commit شوند. شرط اتمی منطقی: یا invoice معتبر ایجاد می‌شود و return رد می‌شود، یا return معتبر commit شده و invoice creation با custody/state جدید re-evaluate می‌شود؛ هرگز invoice orphan/ownerless. روش lock/transaction/versioning انتخاب Engineering آینده، بدون schema design این Gate. Race یا تغییر scope با business conflict پاسخ داده شود؛ replay blind ممنوع.

Reset مجاز فقط current routing/contact fieldsی را تغییر می‌دهد که policy reset اجازه داده است. Financial IDs، origin، historical approvals، ledger، transfer events و invoice history باقی می‌مانند. جزئیات اینکه کدام contact fields پاک/حفظ شوند OPD-03 است؛ default حفظ history و عدم destructive reset. Return جدید event است، نه حذف transfer قبلی.

**Bridge:** dual-read links فعلی، compare old/new eligibility و shadow validation؛ اختلاف مثبت مالی مانع rollout بی‌محافظ است. هیچ invoice relink، field delete یا rewrite تاریخی لازم نیست. سیاست default NO برای exceptionهای مبهم، target محافظتی است و ادعای رفتار فعلی نیست.

## 7. Invoice & Payment

| مفهوم | Identity | State owner | Source of truth / Relationship |
|---|---|---|---|
| Invoice | `sn_invoices.id`؛ code business reference، access_token identity نیست | issuance/payment services و actions مالی مجاز | aggregate مالی فعلی؛ Case links مستقل از phone؛ چند invoice برای Case ممکن |
| Pre-invoice | همان Invoice ID، مرحله صدور پیش از وصول | issuance service | `pre_invoice` وضعیت invoice؛ object مشتری/پرداخت تازه نیست |
| Payment Stage | stage ID + invoice_id + stage_no | stage service/Finance با guard | مبلغ مرحله/due و state؛ یک invoice چند stage |
| Payment | payment/transaction ID بومی namespace-qualified | gateway/registration/review service | attempt/transaction با invoice/stage refs؛ چند attempt برای یک stage ممکن |
| Financial Review | review event/decision ref؛ در legacy tuple موضوع+actor+time+eventref، نه ID جعلی | Finance reviewer دارای action permission | evidence + decision + reason؛ historical review از mirror status جدا |
| Completed Sale | fact مشتق از Invoice ID و valid completion event | finalization predicate | فاکتور تمام‌وصول‌شده؛ Stage approved یا receipt به تنهایی کافی نیست |

**Current:** mirrors invoice/stage/payment/workflow وجود دارند؛ ممکن است ناسازگار باشند. **Target:** هر object state namespace مستقل؛ disagreement با diagnostic مشخص، نه انتخاب label دلخواه. **Bridge:** adapters بدون rewrite؛ AUTHORITATIVE / LEGACY FALLBACK / DISCOVERY ONLY مطابق بخش 6. Unknown review identity از evidence موجود بیان می‌شود، جدید فقط با ref یکتا.

### Payment invariants

همه amounts در واحد پول یکسان و مشخص هستند؛ `abs(paid_amount + remaining_amount - total_amount) <= tolerance`، amounts منفی مجاز نیستند، stage due نباید خارج مانده معتبر باشد. کد finalizer آستانه 0.5 را استفاده می‌کند و بعضی queryها `<` / `<=` متفاوت‌اند. **Compatibility baseline: مقدار 0.5 در واحد بومی همان مسیر، بدون تبدیل ضمنی IRT/IRR**؛ این، تصویب rounding واحد سراسری نیست. OPD-06 تعیین واحد/rounding و boundary نهایی است. Adapter باید predicate و واحد واقعی مسیر را همراه diagnostic بدهد؛ تا تصمیم، اختلاف مرزی completed را UNKNOWN/conflict اعلام کند و شمارش اختلاف در reconciliation باقی بماند. UI money unit را مخفی نکند.

- Partial payment: paid معتبر مثبت و remaining بیش از tolerance مسیر؛ invoice هنوز Completed Sale نیست.
- Stage approved: تصمیم معتبر مالی درباره همان stage؛ approval یک stage قبلی، کل invoice را approved-completed نمی‌کند.
- Invoice completed: تمام obligation مبلغی با payment معتبر پوشش داده شده، remaining در tolerance مصوب، وضعیت/رویداد completion سازگار؛ archived/cancelled به معنی completion نیست.
- Receipt uploaded: فقط سند دریافت‌شده؛ نه bank proof و نه approval.
- Online payment registered: attempt/reference ثبت شده؛ verification و finalization جدا باید ثابت شوند؛ return URL به تنهایی payment success نیست.
- Finance approved: review همان موضوع تأیید شده؛ مبلغ کل فروش فقط با completion predicate.

Reject stage باید reason، actor، time، affected_stage و next_action را ثبت کند. Approvalهای مراحل قبل، creditهای صحیح و history حذف نمی‌شوند. اصلاح stage جدید، attempt/review قابل ردیابی دارد؛ ممنوعیت duplicate credit پابرجاست. Cancel exceptional workflow می‌تواند reversal رسمی داشته باشد، نه حذف approval.

Cancelled = توقف lifecycle عادی invoice؛ Refund required = تعهد بازپرداخت؛ Refund confirmed = اعلام/تأیید مدرک در معنای موجود؛ Refund completed = اجرای مالی reconcile‌شده با proof. `refund_confirmed` فعلی proof انتقال بانکی نیست. کد cancel پرداخت قبلی را با flag guard می‌کند ولی اجرای واقعی refund از آن استنتاج نمی‌شود. Refund object/authority/proof کامل OPD-04 است؛ تا آن زمان تنها همان evidence موجود با label دقیق، نه «بازپرداخت انجام شد».

## 8. Status Namespaces

**Current:** status strings مشابه در چند object، legacy labels و overlays مستقل. **Target:** هیچ approved / ready / completed / archived بدون namespace و subject ref معنی تجاری ندارد. **Bridge:** Central Dictionary read-through؛ values فعلی rename/delete نمی‌شوند. [S/A]

| Namespace | نمونه‌های فعلی / محدودیت |
|---|---|
| case/custody | delivered_to_seller، returned_to_mis، converted_to_lead؛ delivery عدم مصرف را ثابت نمی‌کند |
| contact | no_answer، callback، not_purchased؛ «خرید کرده» legacy completion proof نیست |
| invoice | pre_invoice، partial_paid، approved، paid، rejected، cancelled |
| invoice_workflow | awaiting_payment، awaiting_financial_approval، awaiting_assignment، completed، archived |
| payment_stage | pending، receipt_uploaded، approved، paid، rejected، cancelled |
| finance_review | pending / approved / rejected معناهای تصمیم؛ canonical mapping، نه enum DB جدید اجباری |
| conversion | ready_for_conversion، assigned، deposit_paid، completed؛ terminal guards طبق S |
| hr_request | pending_review، pending_hr، approved، rejected، failed؛ approved یعنی apply موفق |
| commission_run/item | generated / approved / rejected جدا از posted / error؛ rejected run فعلی ممکن است دوباره approved شود |
| wallet_transaction | approved status؛ credit/debit جهت و purpose_commission_reversed نوع است، نه یک status عمومی |
| report_job | building / ready / cancelled؛ هیچ‌کدام case status نیست |

Dictionary contract: machine_id، namespace، fa_label، meaning، terminal، temporary، allowed_actors، reason_required، visual_semantic؛ برای compatibility افزوده: legacy_machine_id، source_domain، canonical_meaning، mapping_confidence، mapping_version، unknown_reason. machine_id منطقی namespace-qualified است و به DB rename الزام نمی‌دهد. terminal پایان مسیر عادی همان object است؛ استثنای reopen/correction در guard جدا. allowed_actors فقط role label نیست؛ action permission + scope + state guards لازم است.

Unknown/custom legacy status با raw value حفظ می‌شود؛ mapping حدسی ممنوع، destructive action مسدود و نیاز validation نمایش داده می‌شود. Visual semantic مانند attention / pending / success / danger / neutral فقط ارائه معناست؛ بعداً Claude آن را به Frozen tokens وصل می‌کند. رنگ، آیکون و label معیار انتقال/permission نیستند. انتقال‌های قطعی و موارد `?` مرجع S هستند؛ این Gate state/transition جدید فاقد evidence Freeze نمی‌کند.

## 9. Metrics

**Current:** F02 predicate پیش‌فاکتور ناقص؛ F03 coverage بعضی queries فقط sn_leads؛ F04 اختلاف stats/table با علت نامعلوم؛ F09 time semantics MIS. **Target:** Metric Fact Layer منطقی با این grainها، بدون warehouse/DB جدید اجباری:

| Grain | کلید منطقی / معنای شمارش |
|---|---|
| Case | canonical_id resolved؛ unknown coverage جدا؛ source aliases یک بار |
| Invoice | invoice ID؛ invoice مشترک چند Case در grand total یک بار |
| Payment Stage | stage ref؛ approved stage count فروش کامل نیست |
| Payment | transaction/attempt ref با verification class؛ تعداد receipt وصول نیست |
| Person | user/person ref؛ current active employee با تاریخی فرق دارد |
| Handoff/Event | event ref با actor/time؛ چند انتقال یک Case چند Case نمی‌شود |

هر KPI باید name، grain، source، cohort، time_basis، scope، numerator، denominator، status_definition، freshness داشته باشد؛ count غیرنسبتی denominator = Not Applicable و formula صریح. Metadata ناقص = **NOT FINAL**؛ zero ناشی از failure/unknown مساوی zero واقعی نیست. numerator/denominator باید grain سازگار و scope مشترک داشته باشند.

| KPI هدف | Grain / Source | Cohort / Time / Predicate | Numerator / Denominator | Scope / Freshness |
|---|---|---|---|---|
| Invoice Created | Invoice / invoice facts | issued/created in range؛ pre_invoice داخل | distinct issued invoices / N/A | مجاز actor و row scope؛ evaluated_at + source version |
| Open Pre-invoices | Invoice / normalized current invoice | current snapshot؛ pre_invoice فعلی بدون completion/cancel | distinct matching invoices / N/A | همان scope؛ current refresh time |
| Pre-invoices Issued | Invoice issuance event | issuance in range؛ later paid بودن سابقه صدور را حذف نمی‌کند | distinct issuance invoices / N/A | همان scope؛ event coverage/freshness |
| Sales Completed | Invoice / valid finalization facts | completion event in range؛ full amount predicate؛ نه صرف issuance | distinct completed invoices / N/A | chosen attribution basis + freshness؛ OPD-06 boundary |
| Verified Collected Amount | approved/paid stage یا payment fact، یک مسیر بدون double count | valid payment event in range؛ no duplicated attempts؛ reversal جدا | sum valid amounts با unit / N/A | scope و basis صریح؛ ledger/payment reconcile time |
| Total Cases | Case / dual-read resolver | legacy+V4 و سایر sourceهای نام اعلام‌شده؛ dedup proven aliases | distinct resolved cases / N/A، unresolved count جدا | current custody یا historical cohort صریح؛ coverage time |
| Legacy Leads Only | legacy lead fact | فقط sn_leads، label صریح | distinct legacy IDs / N/A | scope/query time؛ نام «کل CRM» ممنوع |
| Team Workload | Case/current owner + Person/current hierarchy | current snapshot owned active cases | count cases / N/A | direct/indirect scope مشخص؛ current time |
| Completion Rate | Case cohort با mapped completed invoice outcome | assigned cohort و outcome window مشخص | completed eligible cases / all eligible cohort cases | OPD-07؛ تا cohort/window تصویب NOT FINAL |

pre_invoice در Invoice Created و Pre-invoices Issued می‌آید؛ در Sales Completed تا تحقق completion نمی‌آید. «Open pre-invoices» و «صدور پیش‌فاکتور» دو metric هستند. صادرشده‌ای که بعداً paid شده نباید از metric تاریخی issuance حذف شود. labels pending/draft/unpaid صرفاً پس از mapping معنای منبع، نه blanket synonym.

Card/Table/Export دارای یک metric definition باید same query contract، scope، cohort، cutoff/freshness و predicate داشته باشند. pagination فقط rows صفحه را محدود می‌کند نه total. تفاوت dataset/cutoff باید label/version شود؛ grand totals child-groupهای overlapping قابل جمع خودکار نیستند.

سه time basis: **Current Snapshot** وضعیت اکنون؛ **Event Occurred During Range** انتخاب بر اساس event زمان‌بندی‌شده؛ **Historical As-Of** وضعیت در زمان گذشته از history معتبر. MIS فعلی event range برای انتخاب Case و وضعیت جاری هنگام گردآوری دارد؛ as-of نیست و collection window معادل point-in-time snapshot دیتابیس نیست.

**Bridge / metrics safety:** dual-read و reconciliation پیش از تغییر dashboard: metric_id/version، old_total، new_total، difference، reason (source coverage/status predicate/identity/scope/cutoff/cache/unknown)، currency، resolved/unresolved counts. F04 علت هنوز NEEDS VALIDATION؛ عدد جدید را صرف «هدف» درست اعلام نکنید. انتقال dashboard فقط پس از shadow QA و rollout قابل rollback؛ هیچ query فعلی در این مرحله تغییر نمی‌کند.

## 10. Permission Architecture

| لایه | قرارداد هدف |
|---|---|
| Route | ورود به panel صحیح؛ admin access اثبات role-runtime نیست |
| Module | مشاهده workspace داده متناسب با mission |
| Action | write/review/export/import/impersonation مستقل از view، با intent معتبر |
| Row / Field | subject در scope، state guard، حساسیت فیلد و masking؛ filter ورودی scope را گسترش نمی‌دهد |

هر endpoint باید همه لایه‌های applicable را server-side بررسی کند. nonce ضد درخواست نامعتبر است، جای authorization نیست؛ hidden button هم guard نیست. hierarchy بالاتر اختیار Finance یا HR-sensitive ایجاد نمی‌کند. Role names، HR position و capabilities فعلی سه ورودی متفاوت‌اند؛ blanket replacement ممنوع.

**Finance conceptual capabilities:** view_finance؛ review_payment؛ reject_payment؛ manage_commission_rules؛ review_commission_run؛ post_wallet؛ manage_rollout؛ view_sensitive_financial_export. همگی مستقل قابل grant/revoke هستند؛ review شامل تأیید همان stage است و cancel/reopen نیاز action policy جدا دارد، نه inferred از view. Rule Author، Run Reviewer و Posting Actor قابلیت جداسازی دارند؛ الزام افراد متفاوت و استثنای emergency OPD-08 است.

**HR conceptual capabilities:** view_workforce؛ edit_workforce؛ change_access؛ view_compensation؛ edit_compensation؛ import_workforce؛ export_sensitive_workforce؛ impersonate_user؛ manage_credentials. همه مستقل؛ profile edit اجازه role/capability/credential/compensation change نمی‌دهد. Import هر field حساس علاوه بر import به field/action permission نیاز دارد. Impersonation اختیار target را به audit read-only تبدیل نمی‌کند؛ reason/log/actor chain حفظ شود.

**Current Reality:** F06 بعضی sensitive writes با sn_view_finance؛ approve/reject caps جدا؛ manual wallet adjustment Admin-only؛ Purpose Matrix save/backfill با manage_options یا sn_manage_wallets و view متفاوت. HR نقش/سمت گسترده دارد، sn_manage_hr cap ادعاشده‌ای وجود ندارد. **Bridge:** mapping صریح current guards به conceptual actions؛ current capabilities حذف ناگهانی نشوند. role-by-role rollout، deny tests، audit logging و rollback لازم است. Mapping compatibility مجاز نیست viewer هدف را پنهانی writer بداند؛ entitlementهای legacy نیاز review و explicit mapping دارند. بدون آن rollout write split آماده نیست، ولی target Freeze است. [A F06/F08/C]

## 11. Pure Read

**Target:** rendering/viewing شامل بازکردن Seller/Supervisor/Finance Wallet یا report نباید business state را تغییر دهد: commission credit، invoice/payment state، custody، customer، role یا compensation. تعداد دفعات refresh نباید درآمد یا ownership را تغییر دهد.

| Side effect | GET / read renderer هدف | مسیر مجاز آینده |
|---|---|---|
| Business Mutation | ممنوع | explicit authorized command یا job متکی به business event معتبر؛ idempotent و auditable |
| Technical Cache | مجاز مشروط | scoped، bounded/expiry، بدون تغییر facts یا اجازه گسترده‌تر؛ cache miss ≠ KPI zero |
| Temporary Report Job | ساخت/step/cancel با explicit report operation؛ مشاهده آماده فقط read | job metadata/payload جدا، owner/scope/TTL؛ business state untouched |
| Migration | ممنوع در renderer | controlled admin maintenance command با QA/snapshot/rollback |
| Maintenance/repair/backfill | ممنوع در renderer | explicit restricted command و impact/result؛ صرف page visit trigger نیست |

**Current:** F07 legacy wallet renderer با force autopost را فراخوانی می‌کند؛ matrix enabled می‌تواند credit را مانع شود؛ وقوع write جلسه اثبات نشد. HR renderer migration/ensure-column دارد؛ report job metadata write نیز وجود دارد. **Bridge:** identify callers/dependencies → compare/shadow behavior → انتقال side effect به command/event مجاز با preservation of timing/eligibility/idempotency → feature flag/rollback. حذف فوری autopost یا migration بدون تحلیل dependency، توصیه این Gate نیست. cache/report artifact مجاز را business purity ننامید؛ contract read تجاری pure، نه ادعای صفر write فنی در کل storage.

## 12. HR Temporal Integrity

Position، Manager، Employment و Compensation effective-dated هستند. تغییر حساس جدید باید effective_from، effective_to (nullable برای بازه باز)، actor، reason و subject ref داشته باشد؛ بازه‌های ناسازگار/overlap برای همان مفهوم active بدون policy صریح پذیرفته نشوند. تغییر گذشته فقط correction رسمی با before/after و provenance است، نه silent overwrite. CurrentHierarchy نمای امروز؛ HistoricalAttribution بر پایه رویداد/snapshot همان زمان است، نه recompute با والد امروز.

**Current:** compensation timeline موجود؛ close قبلی پیش از اطمینان تمام موفقیت insert/overlap و عدم transaction جامع، failure risk دارد. locked_after_payroll ستون است ولی enforcement payroll از وجود آن ثابت نیست. HR request review chain و final apply موجود؛ termination is_active را تغییر می‌دهد ولی حذف WP role/session revoke یا handover کامل اثبات نشده. **Target:** یک HR Apply حساس باید وضعیت سازگار و تاریخچه حفظ‌شده تولید کند؛ failed insert نباید compensation قبلی را بی‌دلیل قطع کند. **Bridge:** temporal read adapter، history واقعی موجود؛ unknown historical manager/pay = UNKNOWN، نه current parent backfill. enforcement تدریجی با shadow QA؛ login access ناخواسته قطع نشود. [A F08/C/S]

قبل termination apply impact شامل open cases، invoices، tasks، pending handoffs و commission ownership مشخص باشد. هیچ work item فعال ownerless نماند؛ unresolved allocation یعنی apply مالی/کاری کامل اعلام نشود. طراحی recipient/queue/SLA handover و timing access revocation OPD-05؛ baseline target تا تصمیم: prevent incomplete handover، preserve historical credit، explicit accountable owner و incomplete result. مقصد حقیقی و مجاز باید معلوم باشد؛ انتقال همه کارها به مدیر از روی حدس ممنوع.

## 13. Finance / Commission

| مفهوم | قرارداد |
|---|---|
| Generated Run | محاسبه/preview؛ نه approval و نه earned ledger |
| Approved Run | snapshot ورودی/قواعد/recipientها و totals قفل؛ تغییر بعدی run/version تازه یا correction رسمی |
| Posted Item | item دارای ledger effect معتبر و ref؛ approved run تضمین همه posted نیست |
| Wallet Credit | رخداد مالی ledger؛ credit از preview یا page view تولید نمی‌شود |
| Settlement Ready | eligibility تسویه مستقل؛ positive balance به تنهایی کافی نیست |
| Settled | پرداخت/تسویه reconcile‌شده با ref/proof مستقل؛ posted credit مساوی settlement نیست |

**Current:** legacy rule/run مسیر generated→approved→posting؛ matrix مسیر hook stage-approved و قواعد purpose/share با snapshot دارد؛ engine flag در کد default-enabled است ولی مقدار live اثبات نشده. run rejected در branch فعلی می‌تواند دوباره approve شود. **Target:** انتخاب engine/source برای هر entitlement صریح و بدون double post؛ مسیرهای فعلی به یک workflow فرضی تبدیل نمی‌شوند. active routing/coexistence policy OPD-09. Ledgerهای موجود rewrite نمی‌شوند.

Posting idempotent، retry-safe و auditable: business entitlement identity برای invoice-based سهم = invoice + stage + purpose + role/share + entitlement generation/valid correction context. **Recipient snapshot ویژگی آن entitlement است، نه تنها uniqueness key.** تغییر hierarchy/recipient نباید همان سهم را دوباره credit کند. دلیل: matrix علاوه بر key شامل recipient، role-prefix بدون recipient را بررسی می‌کند. همان run/item در retry credit جدید نیست؛ run ID یا batch ID نباید uniqueness مالی را دور بزند. Correction/reversal با ref رخداد اصلی و مجوز رسمی، نه overwrite یا credit به recipient جدید روی همان key. generation تازه فقط با correction policy، نه counter دلخواه retry.

Non-invoice entitlements مثل daily attendance bonus در بعضی rules وجود دارند؛ invoice-stage key برای آنها کاربرد ندارد. Source event/period + purpose/share + recipient snapshot و منع duplicate باید قراردادی مستقل داشته باشد؛ eligibility دقیق و source key OPD-09، تا آن زمان entitlement تازه اختراع نمی‌شود. Approved historical input immutable است؛ engine قدیمی داده ناقص را از hierarchy امروز پُر نکند.

Reverse/cancel باید original credit و approval قابل ردیابی را حفظ کند. کد matrix ممکن است type original row را reversed و idempotency_key را NULL کند؛ target conservation of financial history از طریق correction mapping/event، نه ادعای byte-immutable بودن هر ستون امروز است. چگونگی replay بعد reversal باید در QA بررسی شود؛ «exactly once» اجرای فعلی اعلام نمی‌شود.

**Bridge:** resolve engine/rules/callers، dual-read eligibility و compare amounts/recipient/key، shadow ledger effects بدون posting، explicit command/event rollout با feature flag و idempotency reconciliation. Recalculate write است، نه view. مسیر migration/backfill تنها FUTURE OPTION پس از snapshot/QA؛ default هیچ تاریخی دوباره post نمی‌شود.

## 14. Failure Semantics

| نتیجه | تعریف / رفتار قرارداد |
|---|---|
| Success | تمام effectهای وعده‌داده‌شده همان unit commit شده؛ ref/version/history و next actor معتبر |
| Partial Success | فقط برای مجموعه چند item یا workflow چند مرحله؛ successful/failed/pending IDs و counts صریح؛ overall success کامل نیست |
| Failure | effect اصلی unit commit نشده؛ علت، conflict/permission/validation و failure owner مشخص |
| Retryable Failure | transient با intent/correlation حفظ‌شده و proof safe retry؛ validation/permission به خودی خود retryable نیست |
| Outcome Unknown | timeout پس از احتمال commit؛ ابتدا reconcile با operation ref، سپس retry مجاز؛ failure قطعی یا success فرض نشود |

Single-case Return/Assignment، HR Apply حساس و Finance Approve stage نباید نیمه‌اثر بی‌نام داشته باشند؛ critical state/history مالی سازگار در مرز commit منطقی. asynchronous notification/report/Woo integration outcome جدا از financial commit با job ref ثبت شود؛ عدم SMS به تنهایی پول تأییدشده را دوباره approve نمی‌کند. atomic logical contract الزام schema/transaction تکنیک خاص در این Gate نیست.

Bulk Return/Assignment/Finance approve/Commission posting ممکن است partial شود؛ هر item success/failure/conflict/already_applied/unknown و اثر committed مشخص؛ total = sum states و amount totals با currency. Retry فقط failed/retryable IDs و همان intent، نه تمام batch بدون بررسی. HR failed status recovery فعلی نامعلوم است؛ قرارداد recovery authorized/reconciled، نه reopen فرضی state. Failure owner مسئول کار unresolved می‌ماند.

**Current:** bulk results و commission posted/error counters در کد؛ atomicity برخی paths ناقص/آزمون‌نشده. **Bridge:** response/result adapter و shadow validation؛ هیچ موفقیت runtime یا rollback خودکار موجود ادعا نمی‌شود. UI future باید partial، stale، loading، empty، unauthorized و business conflict را از هم جدا کند.

## 15. Open Product Decisions

همه موارد زیر **OPEN PRODUCT DECISION — Deferred** هستند؛ safe baseline جلوی رفتار اختراعی را می‌گیرد. هیچ‌یک شرط آغاز تدوین WHAT/WHY Supervisor Spec نیست. بخش وابسته نباید به اختیار یا formula قطعی تبدیل شود. اگر spec بعدی action تازه خارج baseline بخواهد، تصمیم مربوط prerequisite همان action است.

| ID | Decision Needed / Options | Evidence | Risk / Domains | Recommended Direction (تصمیم نشده) | Who Must Decide / Dependency |
|---|---|---|---|---|---|
| OPD-01 | identity lineage بدون proof: continuation یک Case / related distinct Case / unresolved | A F03/F09، C روابط منبع | HIGH؛ Historical Data/Reports/Integrations | proof-first؛ unresolved حفظ؛ encoding canonical ref بعداً Engineering | Product Owner + MIS + Sales + Engineering؛ merge موارد مبهم deferred |
| OPD-02 | handoff حساس recipient ACK و SLA: commit کافی / ACK جدا / وابسته به نوع | A handoffs، HR review path؛ ACK عمومی اثبات نشده | MEDIUM؛ Workflow | تصمیم بر نوع؛ approved decision فعلی را حفظ، ACK انسانی اختراع نکن | Sales + MIS + HR + Product Owner؛ SLA/ACK جدید deferred |
| OPD-03 | release cancelled cases و reset contact fields: never ordinary return / explicit release / شرایط استثنا | F01، active lock exclusions متفاوت | HIGH؛ Workflow/Historical Data | ordinary return blocked؛ release با reconcile و evidence جدا، history حفظ | Sales + MIS + Finance + Product Owner؛ cancellation-release و destructive reset deferred |
| OPD-04 | refund object/proof/authority: manual documented / bank reconcile / integrated execution | cancel refund_confirmed flag؛ proof execution ندارد | HIGH؛ Workflow/Integrations/Historical Data | proof-backed workflow مستقل؛ flag completed تلقی نشود | Finance + Product Owner + Engineering؛ refund operation deferred |
| OPD-05 | termination handover recipient، acceptance، timing access: queue / manager / explicit assignee | F08 terminate profile/meta؛ complete handover نامثبت | HIGH؛ Permissions/Workflow/Historical Data | impact و accountable recipient پیش از apply؛ snapshot historical | HR + Sales + Finance + Product Owner؛ enhanced termination apply deferred |
| OPD-06 | currency/unit/rounding و boundary 0.5: source-compatible / واحد استاندارد و rounding مصوب | finalizer 0.5 و predicate boundary مختلف؛ IRT/IRR | HIGH؛ Reports/Workflow/Integrations | preserve units؛ reconcile boundary؛ هیچ تبدیل ضمنی | Finance + Product Owner + Engineering؛ normalized borderline completion deferred |
| OPD-07 | cohort/outcome window و performance attribution: current hierarchy / event actor / as-of team | F02–04/F09 و D KPI | MEDIUM؛ Reports/Historical Data | metrics جدا label؛ workload=current، credit=historical؛ target conversion rate مصوب | Sales + MIS + Finance + Product Owner؛ target/rate KPI deferred |
| OPD-08 | mandatory maker/checker: capability-only / persons separated / threshold exceptions | F06؛ Audit الزام سازمانی را اثبات نمی‌کند | HIGH؛ Permissions/Workflow | قابلیت‌ها جدا؛ threshold و override مستند | Finance + Product Owner؛ binding separation policy deferred |
| OPD-09 | commission engine routing، non-invoice entitlement key و correction generation: legacy / matrix / explicit split | C dual engines + bonus + role-prefix guard | HIGH؛ Historical Data/Workflow/Integrations | یک مسئول محاسبه هر entitlement؛ shadow reconcile قبل rollout | Finance + HR + Product Owner + Engineering؛ financial activation deferred |
| OPD-10 | Supervisor delegation beyond current scoped actions: receipt/price correction/cancel/reopen/indirect assignment | A/C UI و internal guards یکسان نیستند؛ hierarchy اختیار کامل نیست | HIGH؛ Permissions/Workflow | current rights inventory + action-specific approval؛ هیچ blanket grant | Sales + Finance + HR + Product Owner؛ expansion deferred؛ current spec conditional |

Scope و actor safety قرارداد ثابت‌اند؛ انتخاب business policy داخل این جدول Freeze نشده است. عدم تصمیم درباره rollout روش اجرا، به معنی blocker منطقی Gate نیست. Future migration اگر لازم شد فقط FUTURE OPTION با QA/snapshot/rollback؛ هیچ انتخاب migration امروز انجام نمی‌شود.

## 16. Invariant Registry

تمام rules این جدول **FROZEN TARGET / IMPLEMENTATION DEFERRED** هستند؛ Evidence به rationale/current gap ارجاع دارد، نه اثبات target implemented. QA Acceptance به scenario بخش 17 ارجاع می‌دهد. Safe Adoption Strategy شرط اجرای آینده است. Breakage Risk هزینه rollout بدون bridge، نه تغییر رخ‌داده در این مرحله.

| ID / Title | Rule / Applies To | Reason / Evidence | QA Acceptance | Safe Adoption Strategy | Breakage Risk / Domains |
|---|---|---|---|---|---|
| INV-001 Stable Case Identity | CaseRef مستقل از phone و domain IDs حفظ؛ Core/MIS/Sales/Finance | F01/F03؛ A/C | QA-001 | ADAPTER FIRST | HIGH؛ Database/Historical Data/Integrations |
| INV-002 Proven Alias Resolution | فقط lineage اثبات‌شده merge منطقی؛ unknown حفظ؛ Core/Reports | A F03/F09 | QA-002 | DUAL READ | HIGH؛ Historical Data/Reports |
| INV-003 Source Provenance | origin ثابت؛ unknown تاریخی حدسی پر نشود؛ Core/MIS | C batch/transfer؛ §3 | QA-003 | ADAPTER FIRST | MEDIUM؛ Historical Data |
| INV-004 Four Owners | custody/original/next/credit مستقل؛ همه نقش‌ها | A/D | QA-004 | ADAPTER FIRST | HIGH؛ Permissions/Workflow/Historical Data |
| INV-005 Traceable Handoff | subject/actors/time/reason/result/failure owner؛ ACK فقط با proof؛ Core/HR/Finance | A/C/S؛ §5 | QA-005 | ADAPTER FIRST | MEDIUM؛ Workflow/Integrations |
| INV-006 All Financial Links | return/reset همه لینک‌های معتبر را resolve کند، نه virtual lead فقط؛ MIS/Sales | F01 BOTH HIGH | QA-006 | DUAL READ | HIGH؛ Workflow/Historical Data |
| INV-007 Consumption Guard | فعال/partial/completed/rejected قابل اصلاح محافظت؛ cancellation release خودکار نیست؛ Core | F01/C/S؛ OPD-03 | QA-007 | SHADOW VALIDATION | HIGH؛ Workflow |
| INV-008 Apply Recheck | preview کافی نیست؛ commit scope/dependency دوباره؛ Return/Assignment | F01 guard gap؛ target §6 | QA-008 | FEATURE FLAG | HIGH؛ Workflow/Database |
| INV-009 Return/Invoice Mutual Safety | concurrent creation/return contradictory commit ممنوع؛ Core | F01؛ target §6 | QA-009 | SHADOW VALIDATION | HIGH؛ Database/Workflow/Integrations |
| INV-010 History Conservation | return/reset مالی و handoff history را پاک نکند؛ Core | F01/C؛ §6 | QA-010 | ADAPTER FIRST | HIGH؛ Historical Data |
| INV-011 Separate Financial Objects | invoice/stage/payment/review/complete distinct refs/states؛ Sales/Finance | C/S | QA-011 | ADAPTER FIRST | MEDIUM؛ Workflow/Reports |
| INV-012 Monetary Conservation | paid+remaining≈total یک unit؛ nonnegative؛ boundary conflict explicit؛ Finance | C finalizer؛ OPD-06 | QA-012 | SHADOW VALIDATION | HIGH؛ Workflow/Reports/Integrations |
| INV-013 Completion Proof | stage/receipt/registered attempt completed sale نیست؛ Finance/Reports | C/S؛ F02 | QA-013 | DUAL READ | HIGH؛ Reports/Workflow |
| INV-014 Rejection History | reason/actor/time/stage/nextaction و previous approvals حفظ؛ Finance | C reject transaction/S | QA-014 | ADAPTER FIRST | HIGH؛ Historical Data/Workflow |
| INV-015 Refund Distinction | cancel/required/confirmed/completed جدا؛ flag proof پول نیست؛ Finance | C cancel؛ OPD-04 | QA-015 | ADAPTER FIRST | HIGH؛ Integrations/Workflow |
| INV-016 Namespaced Status | label/state فقط با domain/subject و guarded transition؛ همه | S | QA-016 | ADAPTER FIRST | MEDIUM؛ Workflow/Integrations |
| INV-017 Preserve Legacy Status | mapping/dictionary بدون rename/delete؛ unknown raw preserved؛ Core/UI | S؛ compatibility request | QA-017 | ADAPTER FIRST | HIGH؛ Database/Integrations/Historical Data |
| INV-018 Complete KPI Metadata | بدون grain/source/cohort/time/scope/formula/freshness NOT FINAL؛ Reports | F02–04/F09 | QA-018 | ADAPTER FIRST | MEDIUM؛ Reports |
| INV-019 Preinvoice Semantics | created/issued شامل pre_invoice؛ completed صرفاً fullpaid؛ Reports | F02 BOTH | QA-019 | DUAL READ | MEDIUM؛ Reports |
| INV-020 Case Coverage | total cases تمام sources اعلام‌شده + proven dedup؛ legacy label صریح؛ Reports | F03 | QA-020 | DUAL READ | HIGH؛ Reports/Historical Data |
| INV-021 Report Parity | card/table/export same contract/scope/cohort/cutoff؛ pagination total ثابت؛ Reports | F04؛ C exports | QA-021 | SHADOW VALIDATION | MEDIUM؛ Reports |
| INV-022 Temporal Semantics | current/event-range/as-of جدا؛ MIS/Reports | F09 BOTH | QA-022 | ADAPTER FIRST | MEDIUM؛ Reports |
| INV-023 Four Permission Layers | route/module/action/row-field server guards؛ همه | A/C F06/F08 | QA-023 | FEATURE FLAG | HIGH؛ Permissions/Workflow |
| INV-024 View Is Not Write | conceptual view هیچ sensitive command را grant نمی‌کند؛ Finance/HR | F06/F08 CODE | QA-024 | SHADOW VALIDATION | HIGH؛ Permissions |
| INV-025 Split Sensitive Authorities | HR/Finance concepts مستقل؛ maker/checker configurable نه imposed؛ HR/Finance | F06/F08؛ OPD-08 | QA-025 | FEATURE FLAG | HIGH؛ Permissions/Workflow |
| INV-026 Pure Business Read | rendering financial/customer/custody/access state را تغییر ندهد؛ همه | F07 CODE؛ caller analysis required | QA-026 | SHADOW VALIDATION | HIGH؛ Workflow/Integrations |
| INV-027 Technical Effects Bounded | cache/job metadata scoped؛ migration/maintenance explicit، نه renderer؛ Core/Reports/HR | C reportstore/HR renderer | QA-027 | FEATURE FLAG | HIGH؛ Database/Permissions/Workflow |
| INV-028 Effective-Dated HR | sensitive change effective interval/actor/reason؛ failure oldinterval را خراب نکند؛ HR | F08/C timeline | QA-028 | SHADOW VALIDATION | HIGH؛ Historical Data/Workflow |
| INV-029 Historical Attribution | current hierarchy snapshot credit/history را rewrite نکند؛ HR/Finance/Reports | F08/F09؛ C matrix snapshot | QA-029 | ADAPTER FIRST | HIGH؛ Historical Data/Reports |
| INV-030 Termination Accountability | impact قبل apply؛ active work ownerless نشود؛ HR/Sales | F08؛ OPD-05 | QA-030 | FEATURE FLAG | HIGH؛ Permissions/Workflow |
| INV-031 Locked Approved Input | approved run snapshot حفظ؛ posted/settled مستقل؛ Finance | C runs/S | QA-031 | ADAPTER FIRST | HIGH؛ Historical Data/Workflow |
| INV-032 Idempotent Entitlement | invoice/stage/purpose/share identity مستقل از current recipient؛ retry duplicate نکند؛ Finance | C matrix role-prefix/run guards؛ OPD-09 | QA-032 | SHADOW VALIDATION | HIGH؛ Historical Data/Workflow |
| INV-033 Explicit Engine / Correction | entitlement یک engine مسئول؛ reversal refs حفظ؛ bonus key مستقل؛ Finance | C dualengines/S؛ OPD-09 | QA-033 | FEATURE FLAG | HIGH؛ Integrations/Historical Data |
| INV-034 Truthful Failure Results | perunit commit و bulk partial/unknown/retry distinctions؛ همه writes | C bulk/run/HR؛ §14 | QA-034 | ADAPTER FIRST | HIGH؛ Workflow/Integrations |
| INV-035 Compatibility Preservation | data/IDs/relationships/history preserved؛ no immediate migration؛ همه | درخواست Compatibility + A | QA-035 | ADAPTER FIRST | HIGH؛ Database/Permissions/Historical Data/Integrations |
| INV-036 Reconcile Before Cutover | dualread→compare→shadowQA→gradualwrite؛ metrics old/new/reason و rollback؛ Core/Reports | F02–04؛ درخواست Compatibility | QA-036 | DUAL READ | HIGH؛ Reports/Workflow/Integrations |

### Compatibility & Migration Safety

این بخش برای **هر INV-001…INV-036** الزام‌آور است: **Existing Data Preserved? YES · Historical IDs Preserved? YES · Existing Relationships Preserved? YES · Rollback Possible? REQUIRED · Migration Required Now? NO.** Historical Preservation = YES؛ historical unknown بدون جعل باقی می‌ماند. هیچ contractی به حذف legacy tables، relink invoice، reset payment یا rewrite ledger نیاز فوری ندارد. اگر طرح اجرای آینده خلاف این شروط باشد، همان implementation **BLOCKER** است؛ target این Gate مجبور به آن نیست.

| Invariants مهم | Current Reality | Target Contract | Compatibility Bridge | Safe Adoption Strategy | Breakage Risk | Migration Needed? | Historical Preservation | Rollback Requirement |
|---|---|---|---|---|---|---|---|---|
| 001–003، 020 | IDs مستقل/coverage legacy | logical identity + provenance/coverage | source adapters؛ explicit alias proof؛ unresolved diagnostics | ADAPTER FIRST → DUAL READ | HIGH | NO؛ future option فقط بعد QA | YES؛ original IDs/links intact | نسخه resolver قبلی + mapping version قابل بازگشت؛ aliases حذف نشوند |
| 004–005، 029 | ownership/actor ضمنی و hierarchy جاری | چهار owner و traceable handoff/historical snapshot | read-through columns/events؛ unknown preserved | ADAPTER FIRST | HIGH | NO؛ backfill فقط FUTURE OPTION | YES؛ credit/origin rewrite نشود | نمایش/resolution قبلی؛ committed history پاک نشود |
| 006–010 | guard virtual ID ناقص و destructive current resets | all-links lock + apply recheck + safe concurrency | resolver فعلی links + compare eligibility، guarded rollout | DUAL READ → SHADOW VALIDATION → FEATURE FLAG | HIGH | NO | YES؛ هیچ invoice relink یا historydelete | flag/routing rollback با حفظ protection؛ بازفعال‌سازی silent unsafe return مجاز نیست |
| 011–017 | mirrors/aliases/0.5/flagrefund | object/state/proof جدا | dictionary/status/payment adapters و discrepancy report | ADAPTER FIRST → SHADOW VALIDATION | HIGH | NO | YES؛ DB statuses/approvals unchanged | mapping version و parser قبلی؛ new facts حذف نشوند |
| 018–022، 036 | inconsistent cards/source/time | governed metric facts/parity | old_total/new_total/difference/reason؛ dual-read shadow totals | DUAL READ → SHADOW VALIDATION | HIGH | NO | YES؛ reports قدیمی ناگهانی replace نشوند | query-contract version/flag قبلی؛ semantics version label |
| 023–025 | broad HR/view-finance writes و چند guard | conceptual leastprivilege split | explicit currentcap mapping؛ role-by-role denyQA/logs | SHADOW VALIDATION → FEATURE FLAG | HIGH | NO | YES؛ currentcaps بی‌برنامه حذف نشوند | سابقه entitlement/routing؛ fallback نباید امنیت هدف را پنهانی bypass کند |
| 026–027 | conditional read autopost/migration/reportjob | pure business read + bounded technical ops | callers/dependencies/shadow؛ move effects to explicit command/event | SHADOW VALIDATION → FEATURE FLAG | HIGH | NO | YES؛ existing credits/job facts retained | command/event routing/flag rollback، duplicate prevention حفظ |
| 028–030 | HR current profile/timeline و ناقص handover | temporal consistency/accountability | historical read adapter؛ unknown و explicit handover guards | ADAPTER FIRST → FEATURE FLAG | HIGH | NO | YES؛ access/credit unintended rewrite ممنوع | before/after و effective correction؛ login continuity reviewed |
| 031–033 | legacy run و purpose matrix/ledger | snapshot/idempotency/explicit engines | shadow calculations/key reconciliation؛ formal corrections | SHADOW VALIDATION → FEATURE FLAG | HIGH | NO | YES؛ ledger/posted items not rewritten | routingrollback + credit reconciliation؛ rollback مساوی حذف ledger نیست |
| 034–035 | partial responses و چند قرارداد قدیمی | honest results/zero immediate breakage | response adapter و staging evidence | ADAPTER FIRST | HIGH | NO | YES؛ source responses/events auditable | versioned response/parser و command routing؛ effects حفظ |

برای **هر HIGH** در registry یا decisions، قبل implementation: backup، snapshot، staging، rollback plan و negative tests لازم است. این‌ها شرط اجرای آینده‌اند، در این مرحله backup/migration/rollout اجرا نمی‌شود. Rollback business effects با correction رسمی، نه rewrite/erase؛ feature flag مسیر را برمی‌گرداند ولی پول posted را ناپدید نمی‌کند. هدف zero-breakage به معنای تضمین آزموده‌شده اجرای آینده نیست؛ preservation و مسیر پذیرش امن الزام قراردادی هستند.

اگر Source جدید پیشنهاد شد: **Dual Read → Compare → Shadow QA → Gradual Write → Possible Migration**؛ migration فقط پس از اثبات و QA، FUTURE OPTION. Backfill historical unknown بدون evidence ممنوع؛ backfill مجاز آینده نیز بعد snapshot/QA و حفظ اصل رکورد. KEEP + ADAPT پیش‌فرض همه domains است.

## 17. QA Acceptance Matrix

اینها **قرارداد آزمون آینده، NOT EXECUTED IN GATE 0** هستند؛ هیچ scenario تغییر داده اکنون اجرا نشده است. Test fixtures روی staging و roles واقعی، همراه negative assertions؛ HR/Finance runtime هنوز NOT LIVE VERIFIED. هر QA به INV هم‌شماره نگاشت دارد.

| QA / INV | Given | When | Then |
|---|---|---|---|
| QA-001 / 001 | Case با invoice ID-link و phone فعلی | شماره عوض یا Case دیگری با همان شماره ساخته شود | CaseRef و invoice قبلی ثابت؛ case دوم مستقل؛ هیچ phone relink |
| QA-002 / 002 | MIS/V4/legacy aliases با proof، و pair مشابه بدون proof | resolver/aggregate خوانده شود | proofpair یک canonical Case؛ ambiguous pair unresolved، نه merge حدسی |
| QA-003 / 003 | provenance historical ناقص و currentowner جدید | SourceRef resolve شود | origin UNKNOWN، currentowner به origin جعل نشود؛ native ID/batch حفظ |
| QA-004 / 004 | originalSeller A، custody B، NextActor Finance C، credit A | custody به D منتقل شود | nextaction فقط طبق workflow؛ credit/original unchanged؛ permissions از custody تنها blanket نشوند |
| QA-005 / 005 | distribution committed بدون recipient ACK و HR reviewer decision | handoff adapter خوانده شود | accepted_at جعلی ندارد؛ HR decision actor/time؛ failure مسئول معلوم؛ subject HR case جعلی نیست |
| QA-006 / 006 | V4 follow_invoice_id فعال و lead_id=0؛ legacy real lead invoice؛ Dot financial link | MIS و ordinary sales return preview/apply انجام شود | هر سه blocked با proof source؛ missing one path invoice را پنهان نکند |
| QA-007 / 007 | Case بدون هیچ dependency و policy مجاز؛ rejected invoice؛ cancelled unpaid بی‌release | return eligibility بررسی شود | اول مشروط allowed؛ دو مورد بعد blocked؛ reject/cancel auto-release نیست؛ fulfillment-only نیز blocked |
| QA-008 / 008 | preview allowed | پیش از apply invoice ایجاد یا actor scope تغییر کند | apply recheck رد با conflict/reason؛ هیچ reset current/financial fields |
| QA-009 / 009 | same Case بدون invoice؛ دو intent return/create هم‌زمان | commit رقابتی اجرا شود | فقط ترتیب سازگار موفق؛ loser re-evaluate/fail safely؛ invoice orphan یا دو فرض unused متناقض نداریم |
| QA-010 / 010 | Case مالی‌ندارد ولی transfer/contact history دارد | return مجاز commit شود | current custody مطابق intent؛ financial/source refs و تمام historical events محفوظ؛ return event افزوده |
| QA-011 / 011 | invoice با دو stage و چند payment attempt | stage/review detail خوانده شود | refs مستقل و invoice relationship ثابت؛ approval یک stage به همه objects کپی نشده |
| QA-012 / 012 | amounts در unit مشخص؛ remaining اطراف boundary 0.5 | reconcile/finalization preview انجام شود | conservation/nonnegative؛ unit explicit؛ incompatible boundary conflict، نه currency conversion پنهانی |
| QA-013 / 013 | receipt/registeredonline بدون verification؛ یک stage approved و remaining>tolerance | sales completed KPI/detail خوانده شود | completed=false؛ فقط completion معتبر تمام invoice وارد فروش کامل |
| QA-014 / 014 | stage1 approved؛ stage2 pendingfinance | stage2 رد شود | reason/actor/time/stage/nextaction ثبت؛ stage1 approval/history و credit صحیح پاک نشوند |
| QA-015 / 015 | cancelled invoice با refund_confirmed=1 ولی bank execution proof ندارد | refund outcome نمایش داده شود | «refund completed» ادعا نشود؛ proof gap و unresolved obligation صریح |
| QA-016 / 016 | invoice approved، run approved، stage approved با subjects جدا | dictionary/allowed actions خوانده شود | namespaces/معنا/guards متفاوت؛ label مشابه authority ایجاد نکند |
| QA-017 / 017 | legacy custom status و known aliases | adapter فعال/rollback شود | original DB values untouched؛ unknown raw retained؛ mapping version قابل rollback |
| QA-018 / 018 | KPI بدون cohort یا freshness | metric publish آماده شود | NOT FINAL؛ zero معنای error نگیرد؛ complete metadata prerequisite |
| QA-019 / 019 | یک pre_invoice سپس later fully paid در بازه دیگری | issued/open/completed metrics در دو بازه خوانده شوند | issuance تاریخی حفظ؛ currentopen پس completion کم؛ completed در event window خودش؛ همه roles predicate مشترک |
| QA-020 / 020 | legacy و V4 و alias اثبات‌شده + unresolved source | Total Cases و legacy-only خوانده شوند | total declared coverage dedup؛ unknown/incomplete صریح؛ legacy-only label محدود |
| QA-021 / 021 | identical query/scope/cohort/cutoff و چند صفحه | card/table/export با pages متفاوت خوانده شود | totals/amounts یکسان؛ pagination total تغییر ندهد؛ exception diagnostics مشخص |
| QA-022 / 022 | event در بازه گذشته، state اکنون تغییرکرده | MIS eventrange/current/asof requested شود | current gathered state label؛ asof فقط history معتبر، نه جعل از now؛ collection window نمایش |
| QA-023 / 023 | hiddencontrol و subject خارجscope؛ nonce معتبر | endpoint مستقیم فراخوانی شود | server deny؛ route/module/action/rowfield همگی لازم؛ no mutation |
| QA-024 / 024 | conceptual Finance viewer-only و HR viewer-only | finance rule/post/recalculate و HR access-change درخواست شوند | denied no mutation؛ conceptual explicit writes لازم؛ mapping legacy grant عینی audit شود |
| QA-025 / 025 | HR editprofile بدون changeaccess/compensation؛ Finance ruleauthor بدون review/post | درخواست حساس مستقیم/import و review/post ارسال شود | denied؛ capability separation کار کند؛ makerchecker policy فقط مصوب enforced |
| QA-026 / 026 | eligible legacy credit fixture و موجودی اولیه ledger | Finance/Seller/Supervisor Wallet چندبار باز و report مشاهده شود | هیچ ledger credit یا commercial/customer/custody state change؛ credit صرف explicit command/event |
| QA-027 / 027 | report/cache و HR renderer قبل migration | view و explicit report job اجرا شود | فنی scoped/TTL؛ commercialstate untouched؛ migration در renderer اجرا نشود؛ targetuser scope leakage ندارد |
| QA-028 / 028 | compensation active interval، newapply با insertion failure | HR apply اجرا شود | interval قبلی بی‌دلیل بسته نشود؛ failure truthful؛ successful update effective dates/reason/actor ثبت |
| QA-029 / 029 | approved credit/sale event با manager قدیمی | hierarchy عوض و historical report باز شود | recipient/event attribution unchanged؛ currentworkload hierarchy جدید، historical با basis قبلی |
| QA-030 / 030 | departingSeller با opencase/invoice/task/handoff | termination preview/apply بدون accountable recipient | impact visible؛ incomplete apply complete اعلام نشود؛ ownerless work ممنوع؛ credit history preserved |
| QA-031 / 031 | approved run snapshot و posted subset | rules/HR تغییر و run باز/تکرار شود | snapshotunchanged؛ unposted،posted،settlementready،settled جدا؛ replay amount override ندارد |
| QA-032 / 032 | entitlement قبلاً creditشده؛ recipient جاری عوض | retry same stage/share با run تازه شود | duplicate credit ندارد؛ original recipient snapshot حفظ؛ correction با اصل ref |
| QA-033 / 033 | legacy و matrix candidate هم‌زمان و reversal history | routing/shadow/post/retry بررسی شود | یک effect هر entitlement؛ reversal traceable؛ bonus invoicekey جعلی ندارد؛ engine policy بدون تصمیم فعال نشود |
| QA-034 / 034 | bulk شامل success،conflict،transient و timeout-aftercommit | UI result و retry پردازش شود | partial/unknown صریح؛ peritem effects/IDs؛ reconcile before retry؛ موفق‌ها duplicate نشوند |
| QA-035 / 035 | baseline snapshot IDs/links/statuses/ledger/caps/workflows | adapter/shadow rollout و rollback روی staging انجام شود | data/IDs/relationships/history حفظ؛ هیچ destructive migration لازم؛ native workflow continuity/denytests پاس |
| QA-036 / 036 | old/new metrics متفاوت یا caller-dependent read effect | cutover پیشنهاد شود | old/new/diff/reason evidence پیش از switch؛ unresolved highrisk rollout blocked؛ version/flag rollback بدون data rewrite |

QA-006…010 حداقل مجموعه F01 است و باید در هر دو مسیر MIS و sales return، تک‌مورد/bulk/preview/apply بررسی شود. عدم وجود live HR/Finance accounts فقط اجرای role QA را pending می‌کند، نه تعریف target را. Acceptance این سند، pass بودن این QAها را اعلام نمی‌کند.

## 18. Claude Handoff Contract

**WHAT CLAUDE MAY ASSUME:** CaseRef پایدار در قرارداد هدف با unknown explicit؛ ownership چهارگانه؛ status namespace/dictionary؛ action permissions و scope مستقل؛ metric metadata و freshness؛ next_action مسئول‌دار؛ return eligibility با reason/proof و apply recheck؛ partial/unknown failure واضح؛ compatibility adapters و preservation الزامی. این فرض‌ها contract هستند، نه API/data آماده یا fixes انجام‌شده.

**WHAT CLAUDE MUST NOT INVENT:** statuses/transitionهای تجاری، authority جدید، role/capability grants، metric formulas/cohort، currency/rounding، refund proof، ACK/SLA، release cancelled cases، assignment به indirect users، commission routing/recipient policy یا historical values. OPDها به‌صورت Deferred/blocked-action نشان داده شوند؛ demo fixture هرگز source of truth نیست.

SELLER DESIGN V1 FROZEN فقط visual foundation است؛ نباید current Seller UI به visual target تبدیل شود. Supervisor Spec باید ابتدا WHAT/WHY را تصویب کند؛ در این مرحله Claude اجرا نشده و UI طراحی نمی‌شود. dictionary visual_semantic بعداً به tokens همان foundation وصل شود؛ هیچ color businessrule را تعیین نمی‌کند. Existing IDs/statuses/workflows هنگام طراحی جدید به حذف اجباری تعبیر نشوند.

## 19. Supervisor Product Spec Input

وضعیت: **READY FOR PRODUCT SPEC؛ requirements زیر هدف محصول‌اند، نه UI design یا permission rollout.** Live role مشاهده شده ولی mutation موفق تأیید نشده. Mission: مسئول تخصیص و رفع توقف کار تیم مستقیم، کنترل ready conversion و invoice exceptions scoped، پیگیری HR request در chain؛ نه جانشین MIS، HR یا Finance.

| موضوع | ورودی WHAT / WHY | Dependency / Evidence |
|---|---|---|
| Primary Jobs | بررسی ظرفیت direct Sellers؛ تحویل eligible cases؛ پیگیری due/blocked؛ تخصیص ready به eligible converter/self طبق policy؛ اصلاح scoped invoice evidence مجاز؛ ثبت/بررسی HRrequests مجاز | A/D Supervisor؛ INV-004/006/023 |
| Occasional Jobs | return eligible unused، reviewhistory، subscription correction پیش از پرداخت مجاز، ownwallet، گزارش/reconcile اختلاف، escalation | C؛ هیچ destructive reset blanket ندارد |
| Navigation destinations | Team/Workload؛ Assignment & Return؛ Ready Conversion؛ My Assigned Conversions شرطی؛ Invoices & Exceptions؛ Customer History؛ HR Requests؛ Reports؛ Own Wallet. subscription جز context invoice، distribution logs جز history، legacy unassign همان domain entry با semantics جدا | inventory فعلی 12tabs + sharedreport؛ grouping requirement نه layout یا حذف routes |
| Must See | directteam capacity، currentowner/nextactor/source، eligibility و dependencies، due/reason، invoice/stage/due/paid/remaining/unit، active financial links، transferhistory، request reviewer/state، query freshness | INV-001…014/018/023 |
| Nice to Have | performance trend با cohort مصوب؛ converter detail scoped؛ profile summary؛ own earnings context | OPD-07؛ data completeness شرط |
| Should Hide / restricted | compensation/credentials دیگران، org-wide financial docs، Finance approve/post/rules، raw MISrepair، userID به‌عنوان KPI، customer خارج scope | hide امنیت نیست؛ INV-023–025 |
| Primary Actions | scoped assignment/eligible return؛ ready assignment؛ ownassigned conversion actions طبق guard؛ invoice evidence/receipt و selection-link action فقط دارای current right؛ HRrequest create/review در chain؛ readreport/history | observed controls≠executed actions؛ OPD-10 برای expansion |
| Bulk Actions required? | assignment count/batch و legacy unassign/return current موجود؛ preserve capability با item-level eligibility/results. bulk Finance approve، credential یا HRapply برای Supervisor لازم/مجاز اعلام نشده | INV-006…010/034؛ bulkUI تصمیم visual آینده |
| Team scope | baseline operational assignment = eligible direct recipients؛ نمونه direct Seller6. team reporting می‌تواند subtree مجاز را بخواند ولی scope/timebasis باید اعلام شود | A/C؛ current hierarchy currentworkload نه historicalcredit |
| Direct vs indirect users | directrecipient rules explicit؛ indirect کاربران aggregate/drilldown فقط در scope مجاز؛ hierarchy به خودی خود assignment/write right نمی‌دهد | INV-023؛ indirect write expansion OPD-10 Deferred |
| Assignment rules | active eligible recipient، holder/action permission، source/currentstate و consumed resolver؛ preview/apply recheck؛ no orphan؛ each movement auditable | INV-004…010؛ existing recipients/statuses preserved |
| Return rules | all financiallinks block؛ ordinary return روی cancelled/rejected خودکار مجاز نیست؛ nofinancial فقط اگر سایر policyها اجازه بدهند؛ immutablehistory باقی | F01؛ OPD-03 exception/reset deferred |
| Ready conversion rules | ready_for_conversion از Dot valid guard؛ eligible assignee/self path موجود؛ ready معنی invoicecompleted ندارد؛ myconversions فقط assigned-to-self، team queue مستقل | C/S؛ converter eligibility/expansion خارج guard OPD-10 |
| Invoice permissions | scoped read و actions واقعی guard جدا؛ receipt/preinvoice edit و selection-link controls فعلی مشاهده شدند؛ subscription edit قبل هر payment و پس از آن locked. approve/reject/cancel/post authority blanket ندارد | A/C؛ قیمت/metadata corrections و internal payment mismatch OPD-10؛ actions مشروط |
| HR request responsibilities | request transfer/termination Seller scoped، reason/target معتبر؛ reviewer فقط currentreviewer/chain، finalapply HR؛ approve intermediary≠terminationdone | C/S؛ selfdestination noop validation؛ OPD-05 handover enhanced deferred |
| Reports | shared center حفظ با grain/time/scope: invoices_register، customer_profiles، customer_invoice_details، mis_assignments، pre_invoices، sales، online_sales، card_sales، finance_pending، finance_rejected؛ availability actualguard و sensitivefieldpolicy | C ExecutiveReport؛ profiles phonegroup identity نیست؛ export action مستقل |
| KPIs | active/allSellers و assigned/free با current definitions؛ InvoiceCreated/OpenPreinvoice/SalesCompleted جدا؛ converter stats فقط cohortمصوب؛ ownwallet به جای teamrevenue؛ blocked/overdue/age افزوده فقط با داده معتبر | F02–04؛ INV-018…022؛ rate/targets OPD-07 NOT FINAL |
| Exceptions | consumed return، orphan/link ambiguity، inactive recipient، changedcustody/scope، stale metrics، invoice mirror mismatch، rejected stage، failedHRapply، duplicate/corrected financialattempt | accountable nextactor؛ no fabricated empty/zero |
| Error states | loading، empty صحیح، unauthorized، validationerror، dependencyconflict، stale/incomplete، partial، retryable، outcomeunknown مستقل؛ preserve intent و context | INV-034؛ manualstats504 root cause validation باقی |
| Permissions | viewteam/readscopedcase/readinvoice جدا از assign/return/conversionassign/uploadreceipt/editallowedpreinvoice/submitHR/reviewcurrentHR/export؛ conceptual فقط؛ fieldguards لازم | INV-023–025؛ WPcap naming/mapping rollout آینده |
| Handoffs | upstream distribution→Supervisor؛ Supervisor→directSeller؛ ready→converter/self؛ rejectionFinance→responsibleSeller؛ HRrequest→parentreviewer→HR؛ escalation→failureowner | INV-005؛ ACK عمومی/SLAs OPD-02؛ historical actors حفظ |

معیار ورود Spec: INV-001…036 در تعریف requirements رعایت شوند؛ این شرط به معنای fixes از پیش اجراشده نیست. هیچ صفحه/feature جدید نباید unknown lineage، privilege expansion یا rate formula تصمیم‌نشده را به happy path قطعی تبدیل کند. Features وابسته OPD-02/03/05/07/10 به‌شکل مشروط/Deferred در Spec باقی می‌مانند؛ تصمیم درباره آنها prerequisite activation همان قابلیت است، نه مانع تدوین ساختار Supervisor مبتنی بر baseline فعلی. Current routes/IDs/workflows تا اجرای جداگانه و QA حفظ می‌شوند.

**GATE 0 PRODUCT INVARIANTS FROZEN — READY FOR SUPERVISOR PRODUCT SPEC**

**NO CODE CHANGED.**
