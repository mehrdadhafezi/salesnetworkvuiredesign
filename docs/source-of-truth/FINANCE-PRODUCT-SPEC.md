# FINANCE PRODUCT SPEC

تاریخ: 2026-10-06 · **FROZEN PRODUCT TARGET / IMPLEMENTATION DEFERRED** · **PRODUCT / WHAT–WHY ONLY**

## 1. Executive Summary

Finance مسئول **FINANCIAL REVIEW + RECONCILIATION + LEDGER CONTROL + RULED POSTING** است. صحت مدرک پرداخت، تأیید مرحله، وصول کامل فاکتور، اعتبار پورسانت و تسویه پنج حقیقت مستقل‌اند. ساختار هدف بر اساس **KEEP + ADAPT** Freeze می‌شود؛ این سند تغییر مجوز، اجرای عملیات یا تأیید readiness مالی نیست.

**FINANCE ROLE RUNTIME NOT LIVE VERIFIED**

حساب واقعی مالی در discovery موجود پیدا نشد. بررسی خواندنی workforce در محیط تست شش نقش فروش/MIS را نشان داد؛ وجود لینک «پنل تایید مالی» و «ورود تایید مالی» اثبات ورود با نقش مالی نیست. Admin جایگزین Finance نشده است. Inventory و guards پایین **CODE VERIFIED**، رفتار UI نقش مالی و enforcement موفق **NOT LIVE VERIFIED** هستند. نبود حساب مانع این Product Spec نیست.

مراجع authoritative:

- **A:** [Audit پذیرفته‌شده](CRM-PRODUCT-ROLE-ARCHITECTURE-AUDIT.fa.md)، Finance، F06/F07 و cross-role boundaries.
- **G:** [Gate 0](GATE-0-PRODUCT-INVARIANTS.md)، تمام INV-001…INV-036؛ به‌ویژه financial conservation، evidence/history، view/write separation، snapshots/idempotency و compatibility.
- **C / S / D:** [Code Registry](CODE-EVIDENCE-AND-ACTION-REGISTRY.md)، [Status Catalog](STATUS-TRANSITIONS.fa.md)، [Decisions / Visibility](DECISIONS-VISIBILITY-INTERACTIONS.fa.md).
- Role contracts: [Senior](SENIOR-SUPERVISOR-PRODUCT-SPEC.md)، [Manager](SALES-MANAGER-PRODUCT-SPEC.md)، [Deputy](SALES-DEPUTY-PRODUCT-SPEC.md)، [MIS](MIS-PRODUCT-SPEC.md)، [HR](HR-PRODUCT-SPEC.md). Seller/Supervisor از A/G/D؛ standalone Spec فرض نشده است. SELLER DESIGN V1 FROZEN صرفاً بنیاد visual آینده است.
- **E:** [Role Access Map](ROLE-ACCESS-MAP.fa.md)، [بررسی محدود مالی این مرحله](../evidence/finance-access-and-code-review-2026-10-06.md).

Evidence convention: LIVE VERIFIED فقط مشاهده مشخص همان actor؛ CODE VERIFIED وجود branch/guard؛ BOTH فقط توافق واقعی همان رفتار؛ INFERENCE قرارداد محصول هدف؛ NOT VERIFIED نبود proof. هیچ workflow مالی این سند BOTH یا LIVE VERIFIED نیست. Confidence static inventory HIGH، سیاست هدف HIGH در حدود Gate 0، completeness سیاست بانکی/engine MEDIUM، runtime UNKNOWN. برابری version label با ZIP اثبات برابری کد نصب‌شده نیست.

## 2. Role Mission & Boundaries

| قرارداد | WHAT / WHY |
|---|---|
| Mission | تصمیم مالی مستند، حل اختلاف با owner مشخص، کنترل ledger و posting طبق entitlement معتبر |
| Daily Jobs | بررسی pending stages/evidence؛ رسیدگی به رد/اصلاح؛ mismatch و unknown outcomes؛ بررسی pending/partial runs و گزارش وصول |
| Occasional Jobs | refund مجاز، reconciliation دوره‌ای، review rules/runs، settlement readiness و recovery استثنایی مجاز |
| Decision Responsibilities | پذیرفتن/رد حقیقت مالی مرحله؛ تشخیص پرداخت تکراری؛ اجازه posting در scope؛ escalation ابهام unit/link/proof |
| Data Needed | invoice/stage/payment IDs، Case/source lineage، amount/unit، receipt/gateway refs، seller/team snapshot، prior review، ledger/run/key/history |
| Data Not Needed | notes فروش نامرتبط، credential، اطلاعات HR شخصی و حقوق غیرمرتبط، ابزار raw repair یا مدیریت کلی WP |
| Allowed Responsibilities | review/reconcile/ledger/posting/report فقط با action authority و current-state/row/field guards |
| Explicit Non-Responsibilities | customer lifecycle، تماس فروش، تخصیص lead، source repair، hierarchy/compensation edit، unrestricted Admin یا انتقال بانکی فرضی |

Job frequency و boundaries هدف **INFERENCE / HIGH** مبتنی بر A/G/C است؛ Finance سازمانی بودن به معنی تمام‌اختیار بودن نیست. اصل minimum necessary در field visibility نیز اعمال می‌شود.

## 3. Current Verified Inventory

تمام current facts جدول **CODE VERIFIED / HIGH** از A/C؛ runtime تمام ردیف‌ها **NOT LIVE VERIFIED**. مسیر فعلی شناسایی‌شده `/financial-approval/` و auth `/financial-login/`؛ shortcode `sn_financial_panel`. default slug کد `crm-finance` جای مسیر resolved نصب‌شده فرض نمی‌شود. Compatibility roles: `sn_financial`، `sn_financial_approval`، `sn_finance`؛ HR position=`finance`.

| Current section / feature | حقیقت فعلی | Classification | Decision / rationale |
|---|---|---|---|
| finance-overview | pending/amount/readiness context | ESSENTIAL | KEEP؛ تعریف grain و freshness برای تصمیم |
| Flow guide | preview/review/post guidance | SUPPORTING | SIMPLIFY؛ راهنمای یک engine به همه engines تعمیم نیابد |
| Regression/readiness diagnostics | technical checks | MISPLACED | MOVE Advanced؛ تصمیم روزانه از technical maintenance جدا |
| finance-payments / needs_review | invoice/stage search/sort/page/detail/approve/reject | ESSENTIAL | KEEP؛ Primary Review Queue |
| receipt_uploaded | receipt facet؛ احتمال هم‌پوشانی approved | REDUNDANT presentation | MERGE facet در همان queue؛ رکورد/وضعیت حذف نشود |
| online_paid | gateway-recorded payment | CONFUSING | RENAME «پرداخت درگاهی ثبت‌شده»؛ مساوی Finance approval نیست |
| approved | stage approval listing | SUPPORTING | KEEP؛ invoice fully paid مستقل |
| rejected | reason و returned_to_seller | ESSENTIAL | KEEP؛ correction owner/next actor معلوم |
| Invoice / evidence detail | paid/remaining/stage/ref/history | ESSENTIAL | KEEP؛ linkage و unit در تصمیم حاضر |
| Approve + optional metadata | محدودیت fields در approve handler | ESSENTIAL | SIMPLIFY؛ correction delta و financial decision جدا قابل ردگیری |
| Bulk approve | حداکثر 100 IDs در مسیر موجود | SUPPORTING | KEEP CONDITIONAL؛ نتیجه واقعی هر item |
| Reject | required reason و invoice/stage lock | ESSENTIAL | KEEP؛ رد reset/delete نیست |
| Cancel / reopen | state guard + Dot/Woo/commission dependencies | SUPPORTING | KEEP CONDITIONAL؛ اثر معکوس و entitlement مستقل |
| refund_confirmed flag | گزارش تأیید refund بیرونی؛ انتقال بانکی نیست | CONFUSING | NEEDS VALIDATION؛ target linked refund evidence طبق OPD-04 |
| Gateway CSV | Zibal ReportAPI + local fallback / selected columns | SUPPORTING | KEEP؛ completeness/source؛ export authority مستقل |
| finance-wallet / ledger | user/type/date/source/transaction filters | ESSENTIAL | KEEP؛ earned/posted/settled تفکیک |
| Wallet readiness / logs | orphan/rule/balance checks، references/history | ESSENTIAL | KEEP؛ unresolved را balance قطعی ننامد |
| Legacy recalculate | write با finance-view یا Admin | MISPLACED | MOVE Maintenance؛ F06، اختیار read مجوز write نیست |
| Manual adjustment / dry-run | legacy write Admin-only؛ preview مستقل | SUPPORTING | KEEP CONDITIONAL؛ Finance اختیار جدید خودکار نمی‌گیرد |
| Settlement readiness | آمادگی تسویه | CONFUSING | RENAME «آمادگی تسویه»؛ bank execution ادعا نشود |
| finance-commission diagnostics | rule/calculation diagnostics | SUPPORTING | MOVE Advanced یا run context |
| Rules create / toggle | product/status/payment/employment/rate/effective inputs | ESSENTIAL | KEEP؛ granular authority هدف، current F06 |
| Generate dry-run | run/item/metadata ایجاد می‌کند | ESSENTIAL | KEEP؛ Preview یک write است، pure read نیست |
| Selected run result | bounded items | ESSENTIAL | KEEP؛ subset را full run معرفی نکند |
| Review run approve/reject | approval_status، actor/note/locked_at | ESSENTIAL | KEEP؛ lock semantics هدف با field نام‌دار اثبات نشده |
| Posting activation flag | global setting write | MISPLACED | MOVE authorized Admin rollout domain؛ no permission change now |
| APPLY posting | approved run + enabled flag + item/transaction checks | ESSENTIAL | KEEP CONDITIONAL؛ exact-once/concurrency اثبات نشده |
| Commission audit | run/item/posting logs | ESSENTIAL | KEEP؛ source و key لازم |
| WP sn-commission-matrix | 6 purposes × 11 role shares | ESSENTIAL | KEEP؛ active purpose engine و legacy engine مستقل |
| Matrix save | sn_manage_wallets یا Admin | SUPPORTING | KEEP restricted configuration؛ Finance view کافی نیست |
| Matrix backfill | historical missing credits ممکن است تولید کند | MISPLACED | MOVE Advanced؛ history/entitlement protection |
| Stage commission / reversal hooks | purpose-engine stage credits/reversal | ESSENTIAL | KEEP؛ OPD-09 routing، automatic effect آشکار |
| Refund evidence / reconciliation workspace | کامل بودن integrated flow اثبات نشده | MISSING | ADD target CONDITIONAL؛ نه ادعای وجود current |
| Finance bank statement import | مستقل در صفحات audited یافت نشد | MISSING / unconfirmed need | NEEDS VALIDATION؛ HR import جای آن نیست |

REMOVE برای financial history/engine صرفاً به دلیل تکرار UI مجاز نیست. هیچ feature مهمی از حدس حذف نشده؛ MERGE مربوط به presentation/context است.

## 4. Finance vs Sales / MIS

جدول‌ها قرارداد هدف **INFERENCE / HIGH** هستند، نه اعطای capability فعلی. Scope current از A/C/D معتبر می‌ماند.

| Capability | Sales | Finance | Boundary |
|---|---|---|---|
| Invoice visibility | own/team scope | authorized financial scope | shared identity؛ role-specific fields |
| Receipt submission | evidence + required sales context | review evidence | submit≠validate |
| Stage review | correction/respond | approve/reject مالی | Sales approval سازمانی جای financial stage approval نیست |
| Payment confirmation | گزارش ادعای وصول | تأیید معتبر طبق proof/state | gateway success≠manual review خودکار |
| Rejection | پاسخ/اصلاح | reason + current-stage decision | prior approvals محفوظ |
| Next actor | مسئول اصلاح/ادامه فروش | review یا escalation owner | next actor≠commission owner |
| Refund | request/context/customer coordination | authorized financial verification | bank execution policy OPD-04 |
| Financial completion | consumption نتیجه معتبر | وصول کامل را معتبر می‌کند | completed فقط receipt/stage approval نیست |
| Customer contact | Sales مسئول | صرفاً financial context لازم | Finance owner lifecycle نمی‌شود |
| Sales correction | commercial fields مجاز | discrepancy referral | invoice identity/financial history silent rewrite ممنوع |

| Capability | MIS | Finance | Boundary |
|---|---|---|---|
| Source reconciliation | provenance/query diagnosis | اثر مالی و proof | source repair اجازه ledger write نیست |
| Invoice linkage | lineage evidence/ambiguity | financial link safety | phone join یا automatic relink ممنوع |
| Amount discrepancy | source/report input diagnosis | authoritative financial resolution | اختلاف به دلخواه با total جدید حذف نشود |
| Payment evidence | technical availability context | authenticity/review decision | MIS receipt approval ندارد |
| Refund | source/report discrepancy | request/proof/financial effect | cancellation label proof نیست |
| Ledger | diagnosis داده/گزارش | ledger authority | MIS silent balance repair نمی‌کند |
| Posting | technical failure diagnosis | approved authorized financial execution | retry outcome معلوم لازم |
| Report correction | query/provenance issue | financial semantics/parity confirmation | report fix از financial record rewrite جدا |
| Financial identity | preserve source IDs/proof | preserve invoice/stage/tx/entitlement key | unresolved relation fail closed؛ customer phone فقط discovery |

## 5. Target IA

| سطح / Module | WHAT | WHY / current adaptation |
|---|---|---|
| Primary — Review Queue | تصمیم روی pending stages/evidence؛ rejected follow-up و approved facets | نیاز روزانه؛ finance-payments KEEP |
| Primary — Reconciliation / Refund | discrepancy، unresolved outcome، refund request/proof/result | یک owner/result قابل ردیابی؛ capabilities target conditional |
| Primary — Ledger | posted transactions، balance provenance، reversal/correction | snapshot KPI جای ledger نیست |
| Primary — Rules / Runs / Posting | rules/read-only snapshot، preview/review/post outcomes | engine-specific ownership و approval≠credit |
| Primary — Reports | current financial state، event، ledger و as-of گزارش مستقل | جلوگیری از double count و جمع ناهم‌grain |
| Secondary — Audit / History | correlated review/evidence/run/tx/source history | traceability میان modules |
| Secondary — Configuration | فقط configuration دارای role-authority | global activation/matrix از view مستقل |
| Advanced — Maintenance / recovery | recalc/backfill/diagnostics/exception handling | high-impact corrections مجوز روزانه نیست |

این IA **INFERENCE / HIGH** است؛ هیچ tab ID، menu route، layout یا schema جدید تجویز نمی‌کند. Read modules نباید business mutation پنهان داشته باشند. F07 مانع ادعای pure-read بودن current wallet renderer است.

## 6. Review Queue

Queue grain: **یک reviewable payment stage/evidence context مرتبط با invoice**؛ invoice ممکن است چند stage داشته باشد. Current invoice-centric list با stage adapter حفظ شود؛ count نوع grain را صریح اعلام کند. Facets receipt/approved لزوماً partition نیستند.

| Queue item data | قرارداد |
|---|---|
| Invoice / Case / source | stable invoice ID، CaseRef/SourceRef؛ linkage RESOLVED/UNKNOWN/CONFLICT با proof |
| Customer context | حداقل identity/context مورد نیاز تصمیم؛ شماره شناسه مالی نیست |
| Seller / team | مسئول جاری، original seller و historical attribution جدا |
| Submitted evidence | evidence reference/version، uploader/time/source؛ missing یا unavailable صریح |
| Stage / amount / unit | active stage ID، purpose، claimed vs validated amount، unit/currency و due/paid/remaining |
| Prior review / reason | actor/time/result، prior approvals/rejections، correction delta و دلیل |
| Next actor / timestamps | مسئول review/correction/escalation؛ submitted/review/effective/update times با timezone |
| Linkage confidence | evidence type/proof؛ unknown اجازه action وابسته نمی‌دهد |

**receipt submitted → receipt validated → stage approved** توالی منطقی است؛ validated و approved ممکن است در محصول یک action ثبت شوند اما معنای آن‌ها مستقل می‌ماند. **invoice fully paid** از مجموع وصول معتبر در tolerance مصوب و **sale completed** از قرارداد G حاصل می‌شود، نه از وجود فایل یا approval یک مرحله. Finance review age از زمان ورود به review، invoice age از زمان invoice؛ جایگزین هم نیستند.

## 7. Financial State Model

| Dimension | معنی | از چه چیزی استنتاج نشود؟ |
|---|---|---|
| Invoice Status | lifecycle فعلی invoice و compatibility code | stage/evidence/ledger status |
| Payment Stage | purpose، ترتیب، مبلغ و state همان stage | کل invoice completed |
| Finance Review | pending/approved/rejected + reviewer/reason | صرف gateway record |
| Payment Evidence | submitted/available/validated/disputed/missing به‌عنوان concepts | پول وصول‌شده قطعی |
| Paid Amount | validated received amount طبق قرارداد مالی | جمع receipt claims یا wallet credits |
| Remaining | بدهی با مبنای روشن | cancellation label یا display zero |
| Refund State | requested/reported/proof pending/confirmed/partial/full concepts | cancelled invoice |
| Ledger Posting State | unposted/posted/partial/failed/unknown/reconciled concepts | approved run یا generated preview |

Target concepts نام دقیق DB/status IDs نیستند؛ current IDs و S حفظ می‌شوند. **paid + remaining = total within approved tolerance** الزام G است؛ amounts باید یک unit و basis داشته باشند. OPD-06 تعیین conversion/rounding/tolerance binding است؛ tolerance پراکنده کد یا عبارت 0.5 سیاست قطعی محصول نمی‌شود. Refund dimension gross collection/refunded/net را جدا نگه می‌دارد؛ از این سند فرمول جدید برای rewrite paid/total تاریخی استخراج نشود. Overpayment، disputed evidence یا unknown unit exception است، نه remaining منفیِ بی‌توضیح یا zero خودکار.

## 8. Approve / Reject

| قرارداد | Approve | Reject |
|---|---|---|
| Authority | approve_financial_stage + financial scope | reject_financial_stage + financial scope |
| Current-state guard | همان invoice/stage اکنون reviewable؛ guard recheck نزدیک commit | همان stage قابل رد؛ paid/completed ordinary reject ممنوع مطابق C |
| Evidence | reference و amount/unit/source/link معتبر؛ ابهام fail closed | evidence/reason context محفوظ؛ missing evidence دلیل قابل ثبت |
| Allowed fields | فقط current allowed payment metadata؛ تغییر مالی با delta/authority | review reason و permitted rejection state؛ sales correction ارجاع |
| Reason | decision basis/ref و correction reason؛ policy mandatory detail نیاز validation | reason required مطابق current code |
| Next actor | partial: Seller/authorized continuation؛ full: completed contract | responsible Seller/correction actor؛ unresolved escalation مستقل |
| History | before/after، stage، actor، evidence version، amount/unit | previous evidence/approvals/history/identity بدون حذف |
| Idempotency | retry همان intent duplicate financial effect نسازد | retry همان decision history مخرب یا stage reset نسازد |

Current approve `sn_approve_payment`، reject `sn_reject_payment`، invoice/stage locking و reason در C **CODE VERIFIED**؛ transaction-wide race safety **NOT VERIFIED**. Current repeated complete success به معنی تأیید universally safe retries نیست. Stale/concurrent decision باید conflict با current truth باشد؛ client disabled button ضمانت endpoint نیست. Reject→correction→resubmit→new review context؛ reject history باقی می‌ماند. Cancel/reopen action مستقل با اثر Woo/Dot/commission؛ «رد» مترادف cancellation/refund نیست.

## 9. Refund

Current `refund_confirmed` در cancellation proof انتقال پول بانکی نیست؛ handler انتقال بانکی اجرا نمی‌کند (**CODE VERIFIED**). Target refund یک financial event مرتبط با original invoice/payment است؛ customer request، reported execution و confirmed execution جدا هستند.

| موضوع | قرارداد هدف / محدودیت |
|---|---|
| Request | requester/reason، original invoice/stage/payment ref، requested amount/unit و partial/full intent |
| Reported vs confirmed | reported نتیجه ادعایی؛ confirmed نیاز authority و proof policy مصوب؛ flag تنها کافی فرض نشود |
| Source / authority | gateway/bank/manual source معلوم؛ process_refund با scope؛ request permission جای execution نیست |
| Proof | reference، زمان، مبلغ/unit، result؛ **OPD-04** نوع bank proof و verifier را تعیین می‌کند؛ این سند policy جدید نمی‌سازد |
| Partial / full | amount cap و سابقه refunds لازم؛ duplicate/refund بیش از eligible amount exception؛ limit دقیق وابسته policy |
| Resulting balance | collected/refunded/net/remaining basis شفاف؛ cancellation خودکار net zero نیست |
| History / relation | original invoice ID و payment history ثابت؛ refund/reversal refs و actor/reason/result افزوده |
| Unknown execution | proof/outcome unknown؛ retry پولی خودکار ممنوع تا reconcile |

Refund proof missing→responsible proof provider/Finance reviewer؛ bank execution به capability یا UI خیالی تبدیل نشود. Lifecycle و داده لازم Freeze؛ execution policy **DEFERRED / CONDITIONAL**، runtime **NOT LIVE VERIFIED**.

## 10. Ledger

Finance ledger domain حقیقت رویدادهای posted را بررسی می‌کند؛ invoice collection ledger و commission wallet ledger اگر domainهای مستقل‌اند، یک balance واحد بدون basis ساخته نشود.

| Context لازم | WHAT / WHY |
|---|---|
| Transaction identity / business key | tx ID پایدار + entitlement/event identity؛ retry همان event را دوباره ننویسد |
| Debit / credit | direction و account/domain معلوم؛ debit همیشه refund مشتری نیست |
| Source / relation | invoice/stage/refund/commission/run/source refs؛ unrelated یا unresolved صریح |
| Amount / unit | signed/direction convention، currency/unit، gross/net basis |
| Actor / effective time | initiating actor/system، posting time و financial effective time جدا |
| Posting state | committed tx proof از requested/preview/unknown جدا |
| Reversal / correction | linked compensating event با reason و original tx؛ history overwrite ممنوع |

Ledger **append/audit oriented** است؛ تصحیح مجاز با سابقه و relation، نه حذف خطا و بازنویسی balance گذشته. Current wallet transaction filters/metadata/logs **CODE VERIFIED**؛ همه requirements audit coverage یا immutable storage موفق فرض نشده‌اند. Legacy manual adjustment Admin-only بودن، اختیار post_ledger_entry برای Finance فعلی ایجاد نمی‌کند. Recalculate/backfill را «اصلاح نمایش» ننامید؛ احتمال اثر مالی باید در محصول آشکار باشد.

## 11. Rules / Runs / Posting

Current: rules create/toggle؛ generate dry-run با run/items metadata؛ review approval_status و lock metadata؛ approved run + posting flag + APPLY؛ per-item existing tx check/update/log. این مسیر **CODE VERIFIED** است، نه guarantee همه engineها.

| Conceptual lifecycle | شرط / نتیجه هدف |
|---|---|
| Draft | rule/calculation intent؛ financial write نشده |
| Preview | snapshot inputs/rules/eligibility/amount/unit/scope و engine مشخص؛ generated run یک metadata write است |
| Approved | authorized decision؛ approved snapshot locked؛ approval≠wallet credit |
| Posting | همان approved snapshot؛ recheck authority/state/business key؛ execution identity |
| Posted | transaction IDs و coverage کامل intent اثبات‌شده |
| Partial | فقط بخشی committed؛ posted/skipped/error/unprocessed counts و next actor |
| Failed | failure معلوم با evidence؛ zero effect تنها با proof |
| Outcome Unknown | response/connection ambiguity؛ failure label مجوز repost نیست |
| Reconciled | outcome با tx/business refs resolved؛ correction/retry مجاز روشن |

این‌ها concepts هستند؛ exact stored IDs جدید تجویز نمی‌شود. G INV-031/032/033: immutable approved snapshot، unique business posting key، tx ID، retry audit و engine-specific correction binding. Key صرف run ID نیست؛ ایجاد run جدید حق مالی جدید برای همان entitlement نمی‌سازد. Business identity شامل invoice/stage/purpose/recipient/entitlement domain مطابق engine contract؛ non-invoice routing **OPD-09**.

Targeted code: review handler `class-sn-plugin.php:33353` approved run را reject نمی‌کند و locked_at ثبت می‌کند؛ این به‌تنهایی همه paths snapshot immutability را ثابت نمی‌کند. Posting `:33400` query را به **2000 items** محدود می‌کند؛ status ok/partial آن handler به معنی پوشش هر item کل run نیست. Existing transaction hit از new credit جدا و unprocessed tail آشکار باید باشد. Per-item checks اثبات atomic exactly-once میان concurrent requests یا engines نیستند. Reuse و retry باید result هر business item را حفظ کنند؛ all-success ساختگی ممنوع.

Purpose engine ممکن است هنگام approved stage خودکار credit کند؛ legacy run APPLY تنها مسیر مالی نیست. Active engine/entitlement route و flags باید در context آشکار باشند؛ default flag کد مقدار نصب‌شده را اثبات نمی‌کند. Maker/checker و امکان same actor approval/posting تابع **OPD-08 DEFERRED** است؛ جداسازی وظیفه پیش‌فرض اجرایی جعل نشود.

## 12. Commission / Entitlement Boundary

| مفهوم | owner / boundary | نتیجه |
|---|---|---|
| Earned entitlement | Sales event + approved business policy/engine | حق بالقوه؛ wallet credit نیست |
| Calculated preview | rules/engine با HR effective compensation inputs | snapshot محاسبه؛ approval نیست |
| Approved posting | authorized Finance review بر اساس policy | اجرای ledger هنوز مستقل |
| Wallet credit | engine/posting tx و business key | settlement بانکی نیست |
| Settlement | readiness و execution/proof domain | bank transfer فقط با policy/system proof |
| Reversal | engine financial correction مرتبط original entitlement | history حذف نمی‌شود؛ current HR نرخ قبلی را عوض نمی‌کند |

HR مالک compensation/structure temporal settings، Sales مالک work/event و context، engine مالک calculation/routing، Finance مالک authorized financial decision/posting/reconciliation است. Original seller، current owner، next actor و commission recipient یکسان فرض نشوند. تاریخی شدن hierarchy حق قبلی را rewrite نمی‌کند.

Current purpose matrix stage_amount مبنا دارد؛ snapshot purpose/role/rate/stage/recipient و key invoice/stage/purpose/role/recipient ثبت می‌شود؛ stage rejection اعتبارات قبلاً تأییدشده را خودکار نابود نمی‌کند؛ cancellation/reopen reversal branches وابسته reason/state وجود دارند (**CODE VERIFIED**). Effective concurrency، engine coexistence و non-invoice entitlement routing **NOT VERIFIED / OPD-09 binding**. نه حذف engine پیشنهاد می‌شود، نه یک entitlement در دو engine مجاز فرض می‌شود.

## 13. Reconciliation

تمام مسیرهای هدف **INFERENCE**؛ existence integrated workspace و successful correction **NOT VERIFIED**. Resolve یعنی evidence/outcome، نه بستن ظاهری exception.

| Issue | Owner / Finance responsibility | Allowed correction | Restricted correction | Resolution proof |
|---|---|---|---|---|
| Amount mismatch | Finance؛ MIS source diagnosis | review/clarify authorized financial delta | overwrite total/history برای حذف اختلاف | matching amounts/unit/basis + reason |
| Duplicate payment | Finance | inspect refs، disputed duplicate، authorized reversal policy | delete payment یا automatic credit twice | unique payment refs و financial outcome |
| Missing mirror | MIS technical owner؛ Finance effect check | refer evidence، reconcile linked records | guessed invoice creation/link | stable source/invoice refs + parity |
| Stage/review mismatch | Finance | recheck current stage/history | reset previous approvals | active stage و decision trace |
| Invoice/report mismatch | MIS query + Finance semantics | report diagnosis/correct query contract | financial record rewrite به خاطر table total | same grain/scope/time/source parity |
| Refund mismatch | Finance؛ execution provider proof | reconcile report/proof/result | cancel=refund فرض | original payment/refund refs و net effect |
| Ledger mismatch | Finance؛ engineering diagnosis conditional | authorized linked correction | silent balance overwrite | tx sequence/balance provenance |
| Unknown posting outcome | Finance + technical diagnosis | tx/key lookup، classify committed/unprocessed | blind retry یا error=zero effect | tx IDs، per-item coverage و audit |
| Currency/unit ambiguity | Finance policy owner | obtain approved unit/conversion contract | guessed ×10 / rounding | OPD-06 مصوب و source units |
| Orphan-looking financial relation | MIS lineage + Finance protection | preserve، resolve proof/escalate | phone-based relink یا delete orphan | Case/source/invoice/tx proof |

Handoff: Finance→Seller برای correction با invoice/stage/reason/evidence/current next actor؛ Finance↔MIS برای source/query proof بدون transfer financial authority؛ Finance↔HR فقط effective compensation identity context؛ Finance↔Admin/technical برای flag/unknown engine recovery بدون ledger authority ضمنی. هر handoff actor/time/responsibility/result دارد؛ ACK/SLA عمومی بدون OPD-02 ساخته نشود. **No Fix All** برای مجموعه heterogeneous financial issues.

## 14. Reports

| Report / grain | Scope / source / time basis | Unit / freshness / reconciliation | Export policy |
|---|---|---|---|
| Review workload؛ reviewable stage | authorized financial scope؛ submitted/review time | count، age؛ overlapping facets واضح | scoped evidence-minimal |
| Current invoice financial state؛ invoice | invoice/payment/stage current truth؛ snapshot time | total/paid/remaining با unit؛ incomplete status | same filters/grain؛ all rows vs page مشخص |
| Transaction events؛ payment/refund event | payment/gateway sources؛ event time | gross/refund/net به basis؛ source completeness | external API/local fallback declared |
| Ledger activity/balance؛ tx/account | posted tx domain؛ effective vs posting time | unit، posting/reconcile state؛ balance provenance | restricted ledger fields |
| Runs / posting؛ run/item | immutable snapshot + tx refs؛ generated/review/post times | preview vs posted، processed/unprocessed coverage | bounded subset ≠ full export |
| Historical as-of | event/snapshot evidence؛ cutoff/timezone | recoverable history only؛ UNKNOWN صریح | historical reconstruction proof required |
| Settlement readiness | eligible posted credits/obligations | readiness≠paid settlement | banking data minimum necessary |

KPI contract برای هر metric: grain/source/scope/predicate/cohort/time/unit/freshness/completeness. Pending stages با invoice count جمع نشوند؛ approved stage با completed invoice یکی نیست؛ positive wallet balance با وصول customer یا settlement یکی نیست. Amount aggregates با different units یا source completeness نامعلوم ممنوع. Ledger truth از UI totals نتیجه‌گیری نشود.

Current gateway CSV با نبود token ممکن است local fallback و blank Zibal fields داشته باشد؛ این «کل تراکنش‌های درگاه» اثبات‌شده نیست. Financial export action مستقل authority/field restriction/audit دارد؛ این مرحله هیچ export اجرا نکرد. Bank statement import مستقل **NEEDS VALIDATION / P3 conditional**، نه نیاز قطعی یا قابلیت ساخته‌شده.

## 15. Permissions / Bulk Actions

Permissions پایین **conceptual target** هستند، نه WP names یا migration proposal. همه writes نیاز scope + current-state guard + row guard + field restriction + audit دارند. View menu و nonce به‌تنهایی sensitive-write authority نیستند.

| Conceptual permission | دامنه / محدودیت هدف |
|---|---|
| view_finance_queue / view_scoped_financial_data | authorized domain/rows؛ data scope independent of view label |
| review_payment_evidence | permitted evidence و financial fields؛ download protection مستقل |
| approve_financial_stage / reject_financial_stage | current reviewable stage، evidence/version، allowed delta/reason، reviewer history |
| view_refunds / process_refund | original payment linkage، amount cap/policy/proof؛ OPD-04 |
| view_ledger / post_ledger_entry | separate read/write؛ domain/account/unit؛ original/key/reversal refs؛ manual adjustment current Admin boundary |
| view_finance_runs / approve_finance_run | snapshot coverage، allowed state، approval lock/audit؛ OPD-08 |
| execute_posting | approved snapshot، active engine، row entitlement/key، known outcome/retry policy |
| reconcile_financial_data | investigation distinct from correction authority؛ no relink/rewrite-all |
| view_finance_reports / export_finance_reports | shared report contract؛ export scope/field/size/source declared |
| view_finance_audit | correlated history با minimum necessary fields |

Current code: view `sn_view_finance` یا HR finance؛ approve `sn_approve_payment`؛ reject/cancel `sn_reject_payment`؛ reopen approve؛ wallet tab `sn_view_wallet_commission`. Rules create/toggle، generate/review run، posting activation/APPLY و recalculate از finance-view gate استفاده می‌کنند: **F06 CODE VERIFIED / P1 PRODUCT GAP — DO NOT IMPLEMENT**. Matrix save/backfill `sn_manage_wallets` یا Admin؛ legacy manual adjustment `manage_options`؛ هیچ‌یک خودکار به Finance واگذار نمی‌شود. Usercap mapping HR لزوماً walletcap نمی‌دهد. Runtime negative tests هنوز انجام نشده.

| Bulk action | Classification | قرارداد / current evidence |
|---|---|---|
| Bulk Review | REQUIRED target | خواندن مجموعه و triage؛ تصمیم مالی بی‌بررسی هر row نیست |
| Bulk Approve | SUPPORTED current / CONDITIONAL target | current ≤100 IDs؛ per-item current guard/evidence/outcome؛ no blanket success |
| Bulk Reject | NEEDS VALIDATION | reason و correction owner هر item؛ وجود current handler فرض نشود |
| Bulk Refund | NOT ALLOWED in frozen scope | OPD-04 unresolved؛ bank effects و per-payment proof لازم |
| Bulk Reconcile | CONDITIONAL diagnosis only | grouped investigation؛ bulk correction/Fix All NOT ALLOWED |
| Bulk Posting | SUPPORTED run path / CONDITIONAL | approved snapshot، entitlement keys، per-item tx/skipped/error/unprocessed؛ retry unknown ممنوع |
| Bulk Export | CONDITIONAL | independent authority، exact scope/count/fields/source completeness؛ اجرا نشده |

Unsupported یا denied result به معنای zero business effect نیست مگر proof؛ concurrent/stale/partial outcomes جدا گزارش شوند.

## 16. Auditability

Required logical context: **actor، action، invoice، Case/source، evidence reference/version، amount/unit، before/after، reason، review state، transaction ID، business key، run ID، timestamp، result، correlation/reference**. Actor human/system و initiating/reviewing/posting actor جدا باشند؛ effective time و recorded time یکی فرض نشوند. Unknown تاریخی صریح، نه backfill حدسی از owner فعلی.

Current activity/payment stage history، review actor/reason، run/item/tx logs و wallet metadata **CODE VERIFIED**؛ جامع بودن همه required fields/retention/access و successful audit persistence **NOT LIVE VERIFIED**. این list schema پیشنهادی نیست. Retry و existing-tx hit با new posting متفاوت‌اند؛ attempted/committed/partial/unknown ثبت منطقی مستقل دارند. Correction و reversal به event اصلی متصل می‌مانند؛ delete/overwrite راه پاک‌کردن ردپای مالی نیست. Audit visibility نیز authority و privacy دارد؛ دانلود evidence از صرف hyperlink presence مجاز/امن فرض نشود.

## 17. Data Visibility / States

| Visibility | داده |
|---|---|
| MUST SEE | IDs/linkage proof، current stage/evidence، amount/unit/paid/remaining basis، prior review/reason/next actor، posting/refund state، tx/key/run refs، completeness/time |
| NICE TO HAVE | relevant customer/sales context، scoped team/historical attribution، concise engine/rule/compensation eligibility snapshot |
| SHOULD HIDE / RESTRICT | full bank details غیرلازم، unrelated receipts/customer finances، HR salary/personnel details، credentials، unrelated Sales notes، raw MIS repair tools، unrestricted admin settings |

Financial scope organization-domain بودن current code، دسترسی همه bank/evidence/HR fields را اثبات نمی‌کند. Real Finance account و read-only negative account needed for runtime coverage؛ cap change در این مرحله ممنوع.

| System state | معنای محصول / next actor |
|---|---|
| Loading | truth هنوز آماده نیست؛ action بر stale cached row انجام نشود |
| Empty | scope معتبر، queue خالی؛ با denied/error متفاوت |
| No Result | filter matching ندارد؛ total سازمانی صفر ادعا نشود |
| Unauthorized | action/row/field denied؛ جزئیات sensitive leak نشود |
| Stale | review یا run تغییر کرده؛ current-state reload/recheck پیش از intent جدید |
| Incomplete | missing evidence/link/coverage/proof؛ subset یا unknown unit روشن |
| Conflict | duplicate payment، concurrent review، ledger key conflict؛ review/reconcile لازم |
| Partial Success | بعضی items committed؛ per-item tx/ref/error/unprocessed و owner |
| Retryable Failure | outcome معلوم و retry امن طبق idempotency contract؛ network error کافی نیست |
| Outcome Unknown | possible commit بدون پاسخ؛ reconcile first، blind retry blocked |
| Offline | current financial truth قابل تأیید نیست؛ queued مالی موفق فرض نشود |

Examples: stale review→current reviewer context؛ amount mismatch/currency ambiguity→Finance policy/reconcile؛ missing evidence→Seller correction؛ refund proof missing→proof provider/Finance؛ ledger posting conflict→existing tx/key بررسی؛ partial run→unprocessed items شمارش؛ unknown retry→tx lookup و reconcile. این‌ها **target requirements / INFERENCE**؛ current successful rendering یا enforcement **NOT LIVE VERIFIED**.

## 18. Product Gaps / Deferred

| ID / priority | Gap / evidence / confidence | Disposition |
|---|---|---|
| FIN-RUNTIME / P1 verification | real Finance account absent؛ discovery LIVE فقط Admin context، runtime UNKNOWN | PRODUCT GAP — DO NOT IMPLEMENT؛ design blocker نیست |
| F06 / P1 | view gate sensitive writes؛ CODE VERIFIED / HIGH | PRODUCT GAP — DO NOT IMPLEMENT؛ granular authority هدف |
| F07 / P1 | wallet renderer legacy autopost conditional on engine/eligibility guards؛ CODE VERIFIED / HIGH؛ occurrence NOT LIVE VERIFIED | PRODUCT GAP — DO NOT IMPLEMENT؛ read vs business effect separation |
| FIN-LINK / P0 conditional risk | ambiguous financial identity/lineage آسیب بالقوه؛ G binding، occurrence specific NOT VERIFIED | fail closed target؛ no destructive relink |
| OPD-04 / P1 | refund bank proof/execution authority unresolved | DEFERRED / CONDITIONAL؛ no invented proof policy |
| OPD-06 / P1 | unit/conversion/rounding/tolerance contract | DEFERRED / CONDITIONAL؛ ambiguous amounts block dependent intent |
| OPD-08 / P1 | maker/checker actor separation | DEFERRED / CONDITIONAL؛ no invented same-actor policy |
| OPD-09 / P1 | active engine/entitlement routing/noninvoice key | DEFERRED / CONDITIONAL؛ engine coexistence محفوظ |
| FIN-IDEMPOTENCY / P1 | item/tx checks موجود؛ exactly-once across race/engines NOT VERIFIED | PRODUCT GAP — DO NOT IMPLEMENT؛ unique business truth لازم |
| FIN-RECOVERY / P1 | partial/unknown outcome/retry proof completeness NOT VERIFIED | PRODUCT GAP — DO NOT IMPLEMENT؛ reconcile before retry |
| FIN-RUN-COVERAGE / P1 | post handler LIMIT 2000، full coverage از summary اثبات نمی‌شود؛ CODE VERIFIED / HIGH | PRODUCT GAP — DO NOT IMPLEMENT؛ processed/tail truth |
| FIN-LOCK / P1 | locked_at ثبت می‌شود؛ all-path immutable snapshot NOT VERIFIED | PRODUCT GAP — DO NOT IMPLEMENT؛ G target remains binding |
| FIN-PARITY / P2 | ledger/report parity و evidence URL protection runtime unknown | PRODUCT GAP — DO NOT IMPLEMENT؛ report contract/field coverage |
| FIN-TAXONOMY / P2 | overlap facets و stage/invoice ambiguity؛ CODE VERIFIED | KEEP + ADAPT؛ no duplicate aggregation |
| FIN-IMPORT / P3 | bank statement import need/availability unconfirmed | NEEDS VALIDATION / DEFERRED؛ no new scope |

F07 «هر render حتماً credit می‌کند» نیست؛ matrix engine یا eligibility guard ممکن است جلوگیری کند. هیچ risk به‌عنوان fixed گزارش نمی‌شود. تصمیم‌های باز حد execution را مشروط می‌کنند؛ WHAT/WHY structure قابل Freeze است و **هیچ FINANCE DESIGN BLOCKER** شناسایی نشده. این backlog implementation plan نیست.

## 19. Compatibility / Safe Adoption

**KEEP + ADAPT**: invoice IDs، payment/stage/refund history، ledger transactions، run IDs، posting refs، Case/customer/source links و audit history حفظ می‌شوند. Current status namespaces، roles/caps/routes و engine references قرارداد compatibility دارند؛ title/IA change مجوز status rename یا migration نیست.

ممنوع در این Spec و بدون قرارداد آینده: destructive financial rewrite، invoice relink، ledger overwrite، historical balance rewrite، reset prior approvals، rate/HR-based historical attribution rewrite، implicit capability expansion. Legacy/purpose engines و adapters صرفاً به علت duplication ظاهری حذف نمی‌شوند. Partial/unknown historical records با UNKNOWN باقی می‌مانند؛ backfill حدسی «پاکسازی» نیست.

Risky future changes در permissions، pure-read wallet، engine routing، refund، unit policy، approval lock و recovery: **FUTURE IMPLEMENTATION — REQUIRES QA / ROLLBACK**. Product acceptance evidence آینده باید real Finance + read-only roles، stale/concurrent review، duplicate/stage approval، partial/full refund، per-item run tail، repeat/new-run entitlement keys، unknown outcome، history/report parity و unchanged IDs را پوشش دهد. این معیار WHAT است؛ coding sequence یا rollout plan ارائه نشده.

این مرحله هیچ financial action، export، configuration save، code/plugin/DB/role/capability/data mutation اجرا نکرد. اعتبارسنجی محلی سند و hash کد در E ثبت می‌شود؛ برابری hash محلی اثبات code parity سایت نیست.

## 20. Claude Design Handoff

### WHAT CLAUDE MAY ASSUME

Finance mission/IA و boundaries این سند Freeze شده‌اند؛ Review Queue مرکز تصمیم، evidence/stage/invoice/refund/ledger/posting مستقل، source/IDs/history ثابت، next actor و result truthful. Shared Seller frozen foundation منبع visual آینده است؛ current live UI هدف visual نیست. Requirements این سند current runtime verified تلقی نشوند.

### WHAT CLAUDE MUST NOT INVENT

WP capabilities، stored status IDs، engine routing، bank proof/maker-checker policy، payment/settlement execution، completed از receipt، refund از cancelled، full run success از bounded summary، exact-once از disabled button، یا Finance authority از Admin. هیچ auto-post/auto-fix/relink، hidden balance rewrite یا capability expansion طراحی فرض نشود.

### SHARED COMPONENTS TO REUSE

از shared foundation approved: scoped inventory/filter/report context، Case/source identity context، actor/next actor/history، truthful empty/error/partial/unauthorized states و permission awareness. Reuse در WHAT/WHY است؛ visual composition با Claude. Sales/MIS/HR actions به Finance منتقل نمی‌شوند.

### FINANCE-SPECIFIC DESIGN NEEDS

Finance Review Queue برای تصمیم stage؛ Payment Evidence Panel برای proof/version؛ Financial State Timeline برای independent dimensions/history؛ Reconciliation Workspace برای issue owner/result؛ Refund Review مشروط OPD-04؛ Ledger Explorer برای tx/key/domain/reversal؛ Posting Run Console برای snapshot/approval/coverage/tx outcomes. وجود این needs به معنی current screens یا آماده بودن writes نیست؛ layout/interaction/UI در این سند طراحی نشده.

### LIKELY NEW SHARED COMPONENTS

Idempotency / Outcome Unknown context برای possible committed effects و retry eligibility؛ Sensitive Financial Action Confirmation برای scope/current row/amount/unit/impact/authority/result context؛ provenance/completeness context مشترک Reports؛ correlated audit/result context برای partial items. Shared بودن reuse hypothesis است، نه component implementation requirement یا confirm flow بصری طراحی‌شده.

### DEFERRED / CONDITIONAL ITEMS

Real Finance runtime validation، F06/F07 remediation، OPD-04/06/08/09، snapshot/idempotency across engines، partial/unknown recovery، bank import، evidence protection و ledger/report parity. Claude می‌تواند limitations و states را طبق قرارداد نمایش دهد؛ policy یا success behavior جدید نمی‌سازد. **STOP AFTER THIS SPEC؛ NO IMPLEMENTATION؛ NO CLAUDE EXECUTION.**

FINANCE PRODUCT SPEC FROZEN — READY FOR CLAUDE REDESIGN
NO CODE CHANGED.
